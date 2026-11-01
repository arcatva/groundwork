package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arcatva/groundwork/internal/agent"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/sandbox"
)

// Review runs one suite's review of one PR and writes its record to out.
func Review(layout paths.Layout, s Suite, plugin string, pr PullRequest, out string, budget float64, shared sandbox.SharedTemp) (ReviewRecord, error) {
	task, err := pr.Task()
	if err != nil {
		return ReviewRecord{}, err
	}
	human, err := pr.Human()
	if err != nil {
		return ReviewRecord{}, err
	}
	before := shared.Entries()
	tmp, err := os.MkdirTemp("", "review-")
	if err != nil {
		return ReviewRecord{}, err
	}
	defer os.RemoveAll(tmp)
	work := filepath.Join(tmp, "repo")
	if err := pr.Workspace(work); err != nil {
		return ReviewRecord{}, err
	}
	prompt := task
	if s.Invoke != "" {
		prompt = s.Invoke + " " + task
	}
	run := agent.Claude(prompt, work, agent.ClaudeOptions{
		Settings: RunSettings(layout, shared, before), Budget: budget, Plugin: plugin, Timeout: time.Hour,
	})
	runID := strings.TrimSuffix(filepath.Base(out), ".json")
	left, err := shared.Tidy(sandbox.Diff(shared.Entries(), before), runID)
	if err != nil {
		return ReviewRecord{}, err
	}
	var reason *string
	if r := run.Reason(); r != "" {
		reason = &r
	}
	rec := ReviewRecord{
		Skill: s.Name, PR: pr.Number(), Init: run.Init(), Minutes: run.Minutes, CostUSD: run.Cost(),
		Finished: run.Finished(), Reason: reason,
		Contaminated:     Leaked(run.Events, human),
		ReachedForHidden: ReachedForHidden(run.Events),
		Collided:         sandbox.Collided(run.Stdout),
		LeftInSharedTmp:  left,
		Report:           run.Report(),
		Trace:            run.Trace(),
	}
	if err := writeJSON(out, rec); err != nil {
		return rec, err
	}
	cost := "unknown"
	if rec.CostUSD != nil {
		cost = fmt.Sprintf("%.2f", *rec.CostUSD)
	}
	fmt.Printf("#%s %-28s finished=%t contaminated=%t collided=%t $%s %.1fmin\n",
		pr.Number(), runID, rec.Finished, rec.Contaminated, rec.Collided, cost, rec.Minutes)
	return rec, nil
}

// RunAll runs every suite (or the ones named in only) on every PR, `runs` times, one review at a time
// since they would share one temp folder, and returns the results folder.
func RunAll(layout paths.Layout, prs, only []string, budget float64, runs int) (string, error) {
	dir := filepath.Join(layout.CompareResults, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// Every suite is copied once, before any run starts, so edits made meanwhile cannot leak in.
	var suites []Suite
	plugins := map[string]string{}
	origins := map[string]Origin{}
	for _, s := range All {
		if len(only) > 0 && !slices.Contains(only, s.Name) {
			continue
		}
		suites = append(suites, s)
		if s.Invoke == "" {
			continue
		}
		dest, err := os.MkdirTemp("", "suite-"+s.Name+"-")
		if err != nil {
			return dir, err
		}
		if origins[s.Name], err = s.Plugin(layout, dest); err != nil {
			return dir, err
		}
		plugins[s.Name] = dest
	}
	names := make([]string, len(suites))
	for i, s := range suites {
		names[i] = s.Name
	}
	if err := writeJSON(filepath.Join(dir, "suites.json"), Suites{names, origins, budget, runs}); err != nil {
		return dir, err
	}
	lock, err := sandbox.Acquire(layout.Results)
	if err != nil {
		return dir, err
	}
	defer lock.Release()
	if err := ReadOnly(layout.Venv); err != nil {
		return dir, err
	}
	shared := sandbox.NewSharedTemp(layout.Archive)
	for k := 1; k <= runs; k++ {
		for _, number := range prs {
			for _, s := range suites {
				out := filepath.Join(dir, fmt.Sprintf("%s-%s-r%d.json", number, s.Name, k))
				if _, err := Review(layout, s, plugins[s.Name], NamedPR(layout, number), out, budget, shared); err != nil {
					return dir, err
				}
			}
		}
	}
	return dir, nil
}
