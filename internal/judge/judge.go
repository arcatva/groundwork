// Package judge asks a grader for JSON and holds the reply to its contract.
package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/arcatva/groundwork/internal/agent"
	"github.com/arcatva/groundwork/internal/text"
)

// ErrShape is wrapped by every error that says a grader's reply broke its contract.
var ErrShape = errors.New("grader reply breaks its contract")

// Shape says a reply broke its contract, and how.
func Shape(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrShape, fmt.Sprintf(format, args...))
}

// Options say where the grader runs, which tools it may use, and which command it is.
type Options struct {
	Dir     string        // "" runs it in an empty folder, so it loads no repo's instruction files
	Tools   string        // e.g. "Read,Grep,Glob"; "" for none
	Command string        // another grader's command line with {prompt}; "" for headless Claude Code
	Timeout time.Duration // 0 for 30 minutes
}

// AskJSON asks a grader, decodes the first JSON value in its reply into v, and passes v to check,
// which returns an error made with Shape when the reply breaks its contract. A broken reply is asked
// again once.
func AskJSON(prompt string, v any, check func() error, opt Options) error {
	if opt.Timeout == 0 {
		opt.Timeout = 30 * time.Minute
	}
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		out, err := ask(prompt, opt)
		if err == nil {
			err = decodeFirst(out, v)
		}
		if err == nil {
			if err = check(); err == nil {
				return nil
			}
		}
		last = err
	}
	return fmt.Errorf("%w: no valid reply twice: %v", ErrShape, last)
}

func ask(prompt string, opt Options) (string, error) {
	dir := opt.Dir
	if dir == "" {
		empty, err := os.MkdirTemp("", "grader-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(empty)
		dir = empty
	}
	var name string
	var args []string
	if opt.Command != "" {
		parts, err := agent.Split(opt.Command)
		if err != nil || len(parts) == 0 {
			return "", fmt.Errorf("grader command: %v", err)
		}
		for i, p := range parts {
			parts[i] = strings.ReplaceAll(p, "{prompt}", prompt)
		}
		name, args = parts[0], parts[1:]
	} else {
		name = "claude"
		args = []string{"-p", prompt, "--setting-sources", "project,local", "--strict-mcp-config",
			"--tools", opt.Tools, "--permission-prompts", "none"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), opt.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run() // a reply is judged by its content; a failed run has none
	return out.String(), nil
}

// decodeFirst decodes the JSON between the first '{' or '[' and the last '}' or ']' of out.
func decodeFirst(out string, v any) error {
	start := -1
	for i, c := range out {
		if c == '{' || c == '[' {
			start = i
			break
		}
	}
	end := max(strings.LastIndex(out, "}"), strings.LastIndex(out, "]"))
	if start < 0 || end < start {
		return Shape("no JSON in the reply, which ends: %q", text.Tail(out, 300))
	}
	if err := json.Unmarshal([]byte(out[start:end+1]), v); err != nil {
		return Shape("unreadable JSON: %v", err)
	}
	return nil
}

// Verdict is a grader's call on one rubric criterion.
type Verdict struct {
	Criterion int    `json:"criterion"`
	Pass      bool   `json:"pass"`
	Evidence  string `json:"evidence"`
}

// CheckVerdicts requires one verdict per criterion, numbered 1 to count, each with its evidence.
func CheckVerdicts(verdicts []Verdict, count int) error {
	seen := make([]bool, count+1)
	for _, v := range verdicts {
		if v.Criterion < 1 || v.Criterion > count || seen[v.Criterion] {
			return Shape("criterion %d is out of 1..%d or repeated", v.Criterion, count)
		}
		seen[v.Criterion] = true
	}
	if len(verdicts) != count {
		return Shape("%d verdicts for %d criteria", len(verdicts), count)
	}
	return nil
}
