// Package agent runs one headless agent and turns what it printed into a trace a grader can read.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/arcatva/groundwork/internal/sandbox"
	"github.com/arcatva/groundwork/internal/text"
)

// Event is one line of Claude Code's stream-json output.
type Event = map[string]any

// Run is what a run printed and how it ended.
type Run struct {
	Events   []Event
	Stdout   string
	ExitCode int // -1 when the run timed out or could not start
	Stderr   string
	TimedOut bool
	Minutes  float64
	// PlainTrace is the trace of an agent that prints text instead of events: its output.
	PlainTrace *string
}

// Result is the run's final result event, or nil.
func (r Run) Result() Event {
	for i := len(r.Events) - 1; i >= 0; i-- {
		if r.Events[i]["type"] == "result" {
			return r.Events[i]
		}
	}
	return nil
}

// Finished says whether the run ended on its own, with a result that is not an error.
func (r Run) Finished() bool {
	res := r.Result()
	return !r.TimedOut && r.ExitCode == 0 && res != nil && res["is_error"] != true
}

// Reason says why the run did not finish; "" when it did.
func (r Run) Reason() string {
	switch {
	case r.Finished():
		return ""
	case r.TimedOut:
		return "timeout"
	}
	// The result's kind, the exit code and the end of stderr: enough to tell why from the record.
	sub, _ := r.Result()["subtype"].(string)
	return strings.TrimSpace(fmt.Sprintf("%s exit %d %s", sub, r.ExitCode, text.Tail(strings.TrimSpace(r.Stderr), 300)))
}

// Cost is what the run cost in US dollars, when it said.
func (r Run) Cost() *float64 {
	if v, ok := r.Result()["total_cost_usd"].(float64); ok {
		return &v
	}
	return nil
}

// Report is the final reply: what the user would see.
func (r Run) Report() string {
	s, _ := r.Result()["result"].(string)
	return s
}

// Init is what the run started with: model, tools, MCP servers, permission mode, plugins.
func (r Run) Init() map[string]any {
	out := map[string]any{}
	for _, e := range r.Events {
		if e["type"] == "system" && e["subtype"] == "init" {
			for _, key := range []string{"model", "tools", "mcp_servers", "permissionMode", "plugins"} {
				out[key] = e[key]
			}
			break
		}
	}
	return out
}

// Trace is the run as lines a grader can read.
func (r Run) Trace() string {
	if r.PlainTrace != nil {
		return *r.PlainTrace
	}
	return Trace(r.Events)
}

// ParseEvents reads every whole JSON event; a line cut off by a timeout is skipped, not fatal.
func ParseEvents(stdout string) []Event {
	var events []Event
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(line), &e) == nil {
			events = append(events, e)
		}
	}
	return events
}

func execute(name string, args []string, dir string, timeout time.Duration, withStderr bool) Run {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	started := time.Now()
	err := cmd.Run()
	run := Run{Stdout: out.String(), Stderr: errOut.String(), Minutes: float64(int(time.Since(started).Minutes()*10)) / 10}
	if withStderr {
		run.Stdout += errOut.String()
	}
	var exitErr *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		run.TimedOut, run.ExitCode = true, -1
	case err == nil:
		run.ExitCode = 0
	case errors.As(err, &exitErr):
		run.ExitCode = exitErr.ExitCode()
	default:
		run.ExitCode = -1
	}
	return run
}

// ClaudeOptions are a headless Claude Code run's settings, budget, plugin and time limit.
type ClaudeOptions struct {
	Settings map[string]any
	Budget   float64
	Plugin   string // "" for none
	Timeout  time.Duration
}

// Claude runs headless Claude Code in dir. User settings stay out (they would bring the user's own
// plugins and hooks), and so do MCP servers (the user's own connectors).
func Claude(prompt, dir string, opt ClaudeOptions) Run {
	settings, _ := json.Marshal(opt.Settings)
	args := []string{
		"-p", prompt,
		"--output-format", "stream-json", "--verbose",
		"--setting-sources", "project,local", "--settings", string(settings), "--strict-mcp-config",
		"--permission-mode", "acceptEdits", "--permission-prompts", "none",
		"--allowedTools",
	}
	args = append(args, sandbox.Tools...)
	args = append(args, "--max-budget-usd", strconv.FormatFloat(opt.Budget, 'f', -1, 64))
	if opt.Plugin != "" {
		// The skill's own files must be readable from the run, as they are in a real install.
		args = append(args, "--plugin-dir", opt.Plugin, "--add-dir", opt.Plugin)
	}
	run := execute("claude", args, dir, opt.Timeout, false)
	run.Events = ParseEvents(run.Stdout)
	return run
}

// Command runs any other agent's command line, with {prompt} and {skill_dir} filled in. Its trace is
// its output, and it counts as finished when it exits 0.
func Command(template, prompt, dir, skillDir string, timeout time.Duration) (Run, error) {
	parts, err := Split(template)
	if err != nil {
		return Run{}, err
	}
	if len(parts) == 0 {
		return Run{}, errors.New("empty agent command")
	}
	for i, p := range parts {
		parts[i] = strings.NewReplacer("{prompt}", prompt, "{skill_dir}", skillDir).Replace(p)
	}
	run := execute(parts[0], parts[1:], dir, timeout, true)
	run.PlainTrace = &run.Stdout
	if run.ExitCode == 0 {
		// A command has no result event; its whole output stands for the reply.
		run.Events = []Event{{"type": "result", "result": run.Stdout}}
	}
	return run, nil
}

// Split breaks a command line into words the way a shell would for quotes and backslashes.
func Split(line string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	escaped := false
	for _, c := range line {
		switch {
		case escaped:
			cur.WriteRune(c)
			escaped, inWord = false, true
		case c == '\\' && quote != '\'':
			escaped = true
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteRune(c)
			}
		case c == '\'' || c == '"':
			quote, inWord = c, true
		case c == ' ' || c == '\t' || c == '\n':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unfinished quote or escape in %q", line)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// Trace renders events as lines a grader can read: a subagent's lines are marked [sub], and a system
// notice repeated back to back (a denied permission, say) is kept once with its count and its kind.
func Trace(events []Event) string {
	var lines []string
	lastNotice, repeats := "", 0
	flush := func() {
		if repeats > 0 {
			lines = append(lines, fmt.Sprintf("(the notice above, %d more times)", repeats))
		}
		repeats = 0
	}
	for _, e := range events {
		sub := ""
		if id, _ := e["parent_tool_use_id"].(string); id != "" {
			sub = "[sub]"
		}
		if msg, ok := e["message"].(string); ok {
			if msg == lastNotice {
				repeats++
				continue
			}
			flush()
			lastNotice = msg
			var kind []string
			for _, k := range []string{"type", "subtype", "tool_name"} {
				if v, _ := e[k].(string); v != "" {
					kind = append(kind, v)
				}
			}
			lines = append(lines, fmt.Sprintf("%s[%s] %s", sub, strings.Join(kind, ":"), text.Head(msg, 300)))
			continue
		}
		flush()
		lastNotice = ""
		msg, _ := e["message"].(map[string]any)
		content, _ := msg["content"].([]any)
		for _, raw := range content {
			part, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			switch part["type"] {
			case "text":
				s, _ := part["text"].(string)
				lines = append(lines, fmt.Sprintf("%s[text] %s", sub, s))
			case "tool_use":
				name, _ := part["name"].(string)
				input, _ := json.Marshal(part["input"])
				lines = append(lines, fmt.Sprintf("%s[tool] %s %s", sub, name, text.Head(string(input), 600)))
			case "tool_result":
				body, ok := part["content"].(string)
				if !ok {
					raw, _ := json.Marshal(part["content"])
					body = string(raw)
				}
				lines = append(lines, fmt.Sprintf("%s[result] %s", sub, text.Head(body, 400)))
			}
		}
	}
	flush()
	return strings.Join(lines, "\n")
}

// Clipped is the head and the tail of a long text: what a run set out to do, and how it ended.
func Clipped(s string, limit int) string {
	n := text.Len(s)
	if n <= limit {
		return s
	}
	head := limit / 4
	return fmt.Sprintf("%s\n[... %d characters cut ...]\n%s", text.Head(s, head), n-limit, text.Tail(s, limit-head))
}
