// Command groundwork checks the skill, runs its evals, and compares it with other review skills.
//
//	groundwork lint                                  the repo's own rules: structure, budget, neutrality, manifests
//	groundwork gen                                   write the tables and schemas generated from the skill's index
//	groundwork check report FILE                     hold a reviewer's report to its contract (anywhere)
//	groundwork check log FILE                        hold a run log to its contract and the flow (anywhere)
//	groundwork eval [--baseline] [--only NAME]       scenario evals, with or without the skill
//	groundwork compare prepare REPO NUMBER...        fetch public PRs and their human review
//	groundwork compare run [--pr N,N] [--runs 3]     every suite's review of every PR, one at a time
//	groundwork compare grade [--results DIR]         blind, repeated grading against the human review
//	groundwork compare report [--results DIR]        the comparison table, in markdown
//	groundwork version
//
// Run lint, gen, eval and compare from inside the repo; check works in any project. Results go under
// evals/results/, which git ignores.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/arcatva/groundwork"
	"github.com/arcatva/groundwork/internal/compare"
	"github.com/arcatva/groundwork/internal/evals"
	"github.com/arcatva/groundwork/internal/gen"
	"github.com/arcatva/groundwork/internal/lint"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/spec"
)

// version is set at release time by -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		var exit exitCode
		if errors.As(err, &exit) {
			os.Exit(int(exit))
		}
		fmt.Fprintln(os.Stderr, "groundwork:", err)
		os.Exit(2)
	}
}

// exitCode ends the program with a code and no message: the command has already said why.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit %d", int(e)) }

func usage() error {
	fmt.Fprint(os.Stderr, `usage: groundwork <command> [arguments]

  lint                              check the repo's own rules
  gen                               write the tables and schemas generated from the skill's index
  check report FILE                 hold a reviewer's report to its contract
  check log FILE                    hold a run log to its contract and the flow
  eval [flags]                      run the scenario evals (--help for flags)
  compare prepare REPO NUMBER...    fetch public PRs and their human review
  compare run [flags]               every suite's review of every PR, one at a time
  compare grade [--results DIR]     blind, repeated grading
  compare report [--results DIR]    the comparison table, in markdown
  version
`)
	return exitCode(2)
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "check":
		// Check works in any project, with the index built into the command.
		return runCheck(args[1:])
	}
	layout, err := paths.Find()
	if err != nil {
		return err
	}
	switch args[0] {
	case "lint":
		return runLint(layout)
	case "gen":
		changed, err := gen.Write(layout.Skill)
		if err != nil {
			return err
		}
		for _, f := range changed {
			fmt.Println("wrote", f)
		}
		return nil
	case "eval":
		return runEval(layout, args[1:])
	case "compare":
		return runCompare(layout, args[1:])
	}
	return usage()
}

func runCheck(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	var err error
	switch args[0] {
	case "report":
		err = spec.CheckReport(args[1])
	case "log":
		ix, ixErr := spec.ParseIndex(groundwork.Index)
		if ixErr != nil {
			return ixErr
		}
		err = spec.CheckLogFile(args[1], ix)
	default:
		return usage()
	}
	if err != nil {
		fmt.Println(err)
		return exitCode(1)
	}
	fmt.Println("ok")
	return nil
}

func runLint(layout paths.Layout) error {
	problems := lint.Run(layout)
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		fmt.Printf("%d problems\n", len(problems))
		return exitCode(1)
	}
	fmt.Println("ok")
	return nil
}

func runEval(layout paths.Layout, args []string) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	var opt evals.Options
	fs.BoolVar(&opt.Baseline, "baseline", false, "run without the skill")
	fs.StringVar(&opt.Only, "only", "", "run one scenario by name")
	fs.Float64Var(&opt.Budget, "budget", 6, "US dollar cap per Claude Code run")
	fs.StringVar(&opt.AgentCmd, "agent-cmd", "", "run another agent: a command with {prompt}, and {skill_dir} where it takes the skill's path")
	fs.StringVar(&opt.GraderCmd, "grader-cmd", "", "grade with another command: a command with {prompt}")
	if err := fs.Parse(args); err != nil {
		return exitCode(2)
	}
	sum, err := evals.Run(layout, opt)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(layout.Root, sum.Dir)
	label := "groundwork"
	if opt.Baseline {
		label = "baseline"
	}
	fmt.Printf("%s: %d/%d, outcomes only %d/%d  skill %s (%d words)  -> %s\n",
		label, sum.Passed, sum.Total, sum.OutcomePassed, sum.OutcomeTotal, sum.Version.SkillSHA256, sum.Version.SkillWords, rel)
	if len(sum.Ungraded) > 0 {
		fmt.Printf("ungraded, not in the total: %s\n", strings.Join(sum.Ungraded, ", "))
	}
	if len(sum.Unrun) > 0 {
		fmt.Printf("unrun, not in the total: %s\n", strings.Join(sum.Unrun, ", "))
	}
	if len(sum.Ungraded)+len(sum.Unrun) > 0 {
		return exitCode(1)
	}
	return nil
}

func runCompare(layout paths.Layout, args []string) error {
	if len(args) == 0 {
		return usage()
	}
	fs := flag.NewFlagSet("compare "+args[0], flag.ContinueOnError)
	switch args[0] {
	case "prepare":
		if len(args) < 3 {
			return usage()
		}
		var numbers []int
		for _, a := range args[2:] {
			n, err := strconv.Atoi(a)
			if err != nil {
				return fmt.Errorf("PR number %q: %w", a, err)
			}
			numbers = append(numbers, n)
		}
		return compare.Prepare(layout, args[1], numbers)
	case "run":
		prList := fs.String("pr", "", "comma-separated PR numbers (default: every prepared PR)")
		only := fs.String("only", "", "comma-separated suites (default: all)")
		budget := fs.Float64("budget", 20, "US dollar cap per review")
		runs := fs.Int("runs", 3, "runs of every suite on every PR")
		if err := fs.Parse(args[1:]); err != nil {
			return exitCode(2)
		}
		prs := split(*prList)
		if len(prs) == 0 {
			entries, err := os.ReadDir(layout.PRs)
			if err != nil {
				return err
			}
			for _, e := range entries {
				prs = append(prs, e.Name())
			}
			sort.Strings(prs)
		}
		dir, err := compare.RunAll(layout, prs, split(*only), *budget, *runs)
		fmt.Println("results in", dir)
		return err
	case "grade", "report":
		results := fs.String("results", "", "the results folder (default: the newest)")
		if err := fs.Parse(args[1:]); err != nil {
			return exitCode(2)
		}
		dir := *results
		if dir == "" {
			var err error
			if dir, err = newest(layout.CompareResults); err != nil {
				return err
			}
		}
		if args[0] == "grade" {
			return compare.GradeAll(layout, dir)
		}
		md, err := compare.Report(layout, dir)
		if err != nil {
			return err
		}
		fmt.Print(md)
		return nil
	}
	return usage()
}

func split(list string) []string {
	var out []string
	for _, s := range strings.Split(list, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func newest(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no results in %s", dir)
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1]), nil
}
