// Package evals runs each scenario against a fresh copy of the fixture and grades the run on its
// rubric. Runs go one at a time, like the comparison's, since the sandbox gives them one temp folder.
package evals

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arcatva/groundwork/internal/agent"
	"github.com/arcatva/groundwork/internal/gitx"
	"github.com/arcatva/groundwork/internal/judge"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/sandbox"
	"github.com/arcatva/groundwork/internal/scenario"
	"github.com/arcatva/groundwork/internal/text"
)

// Options choose what runs and how it is graded.
type Options struct {
	Baseline  bool    // run without the skill
	Only      string  // one scenario by name
	Budget    float64 // US dollar cap per Claude Code run
	AgentCmd  string  // another agent's command line, with {prompt} and {skill_dir}
	GraderCmd string  // another grader's command line, with {prompt}
}

// Record is what a graded run leaves in the results folder.
type Record struct {
	Prompt  string           `json:"prompt"`
	Version scenario.Version `json:"version"`
	Init    map[string]any   `json:"init"`
	CostUSD *float64         `json:"cost_usd"`
	Rubric  []string         `json:"rubric"`
	Process []int            `json:"process"`
	Grades  []judge.Verdict  `json:"grades"`
	Trace   string           `json:"trace"`
	Git     string           `json:"git"`
}

// Summary is the totals across scenarios. Outcome counts leave out process-only criteria.
type Summary struct {
	Passed, Total, OutcomePassed, OutcomeTotal int
	Ungraded, Unrun                            []string
	Dir                                        string
	Version                                    scenario.Version
}

// GitState is status, history and diff after a run. New files and commits count as changes too;
// marking new files as intended (add -N) puts their content in the diff without staging it.
func GitState(work string) (string, error) {
	status, err := gitx.Git(work, "status", "--short", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	if _, err := gitx.Git(work, "add", "-A", "-N"); err != nil {
		return "", err
	}
	log, err := gitx.Git(work, "log", "--oneline", "--all")
	if err != nil {
		return "", err
	}
	diff, err := gitx.Git(work, "diff", "HEAD")
	return status + log + diff, err
}

// Grade asks a grader for one verdict per rubric criterion.
func Grade(s scenario.Scenario, trace, state, graderCmd string) ([]judge.Verdict, error) {
	var criteria strings.Builder
	for i, c := range s.Rubric {
		fmt.Fprintf(&criteria, "%d. %s\n", i+1, c)
	}
	prompt := "You grade one run of a coding agent. Below are the criteria, the run's trace (its messages, " +
		"tool calls and results) and the git state after it. For each criterion, decide pass or fail " +
		"from the trace and the git state alone, and quote the evidence. Reply with only a JSON array: " +
		`[{"criterion": 1, "pass": true, "evidence": "..."}].` + "\n\n" +
		"## Criteria\n" + criteria.String() + "\n## Trace\n" + agent.Clipped(trace, 60000) +
		"\n\n## Git state after the run\n" + text.Head(state, 20000)
	var verdicts []judge.Verdict
	err := judge.AskJSON(prompt, &verdicts, func() error { return judge.CheckVerdicts(verdicts, len(s.Rubric)) },
		judge.Options{Command: graderCmd, Timeout: 10 * time.Minute})
	return verdicts, err
}

// Prompt is what the scenario asks, with the skill invoked unless this is a baseline run.
func Prompt(s scenario.Scenario, opt Options) string {
	switch {
	case opt.Baseline:
		return s.Prompt
	case opt.AgentCmd != "":
		return "Use the groundwork skill. " + s.Prompt
	default:
		return "/groundwork:groundwork " + s.Prompt
	}
}

// Run runs the scenarios and writes one record per scenario, printing a line each as it goes.
func Run(layout paths.Layout, opt Options) (Summary, error) {
	label := "groundwork"
	if opt.Baseline {
		label = "baseline"
	}
	if opt.AgentCmd != "" {
		label += "-other"
	}
	only := opt.Only
	if only == "" {
		only = "all"
	}
	dir := filepath.Join(layout.Results, fmt.Sprintf("%s-%s-%s-%d", time.Now().Format("20060102-150405"), label, only, os.Getpid()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Summary{}, err
	}
	pluginDir, err := os.MkdirTemp("", "groundwork-plugin-")
	if err != nil {
		return Summary{}, err
	}
	version, err := scenario.Snapshot(layout.Root, pluginDir)
	if err != nil {
		return Summary{}, err
	}
	if opt.AgentCmd != "" && !strings.Contains(opt.AgentCmd, "{skill_dir}") {
		// Without {skill_dir} the agent runs whatever skill it has installed, not this snapshot.
		version.SkillSHA256 = "the agent's own install"
	}
	if err := writeJSON(filepath.Join(dir, "run.json"), map[string]any{"label": label, "options": opt, "version": version}); err != nil {
		return Summary{}, err
	}
	scenarios, err := scenario.All(layout.Scenarios, opt.Only)
	if err != nil {
		return Summary{}, err
	}
	lock, err := sandbox.Acquire(layout.Results)
	if err != nil {
		return Summary{}, err
	}
	defer lock.Release()
	shared := sandbox.NewSharedTemp(filepath.Join(layout.Archive, "evals"))
	sum := Summary{Dir: dir, Version: version}
	for _, s := range scenarios {
		if err := runOne(layout, opt, s, pluginDir, shared, &sum); err != nil {
			return sum, err
		}
	}
	return sum, nil
}

func runOne(layout paths.Layout, opt Options, s scenario.Scenario, pluginDir string, shared sandbox.SharedTemp, sum *Summary) error {
	prompt := Prompt(s, opt)
	out := filepath.Join(sum.Dir, s.Name+".json")
	before := shared.Entries()
	tmp, err := os.MkdirTemp("", "scenario-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	work := filepath.Join(tmp, "repo")
	if err := s.Workspace(layout.Fixture, work); err != nil {
		return err
	}
	var run agent.Run
	if opt.AgentCmd != "" {
		// A baseline run gets an empty folder where the skill would be.
		skillDir := filepath.Join(pluginDir, "skills", "groundwork")
		if opt.Baseline {
			skillDir = filepath.Join(tmp, "no-skill")
			_ = os.MkdirAll(skillDir, 0o755)
		}
		if run, err = agent.Command(opt.AgentCmd, prompt, work, skillDir, 30*time.Minute); err != nil {
			return err
		}
	} else {
		plugin := pluginDir
		if opt.Baseline {
			plugin = ""
		}
		run = agent.Claude(prompt, work, agent.ClaudeOptions{
			Settings: sandbox.Settings(shared.Stale(before)), Budget: opt.Budget, Plugin: plugin, Timeout: 30 * time.Minute,
		})
	}
	state := ""
	if run.Finished() {
		if state, err = GitState(work); err != nil {
			return err
		}
	}
	if opt.AgentCmd == "" {
		if _, err := shared.Tidy(sandbox.Diff(shared.Entries(), before), filepath.Base(sum.Dir)+"-"+s.Name); err != nil {
			return err
		}
	}
	if !run.Finished() {
		sum.Unrun = append(sum.Unrun, s.Name)
		fmt.Printf("%-20s UNRUN: run did not finish: %s\n", s.Name, run.Reason())
		return writeJSON(out, map[string]any{"prompt": prompt, "error": run.Reason()})
	}
	verdicts, err := Grade(s, run.Trace(), state, opt.GraderCmd)
	if err != nil {
		if !errors.Is(err, judge.ErrShape) {
			return err
		}
		sum.Ungraded = append(sum.Ungraded, s.Name)
		fmt.Printf("%-20s UNGRADED: %v\n", s.Name, err)
		return writeJSON(out, map[string]any{"prompt": prompt, "error": err.Error(), "trace": run.Trace()})
	}
	passed := 0
	process := []int{}
	for n := range s.Process {
		process = append(process, n)
	}
	sort.Ints(process)
	for _, v := range verdicts {
		if v.Pass {
			passed++
		}
		if !s.Process[v.Criterion] {
			sum.OutcomeTotal++
			if v.Pass {
				sum.OutcomePassed++
			}
		}
	}
	sum.Passed += passed
	sum.Total += len(s.Rubric)
	cost := "unknown"
	if c := run.Cost(); c != nil {
		cost = fmt.Sprintf("%.2f", *c)
	}
	fmt.Printf("%-20s %d/%d  cost $%s\n", s.Name, passed, len(s.Rubric), cost)
	for _, v := range verdicts {
		if !v.Pass {
			fmt.Printf("    fail %d: %s\n", v.Criterion, s.Rubric[v.Criterion-1])
		}
	}
	return writeJSON(out, Record{prompt, sum.Version, run.Init(), run.Cost(), s.Rubric, process, verdicts, run.Trace(), state})
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
