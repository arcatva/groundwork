package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arcatva/groundwork/internal/agent"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/sandbox"
	"github.com/arcatva/groundwork/internal/text"
)

// markers name the hidden places in a tool call's input.
var markers = []string{"compare-data/prs", "results/compare", "compare-data/rivals", "tmp-archive", ".claude/projects"}

// Hidden is where the human review, or another run's work, can be read: this harness's data, the
// leftovers it archives, and the session transcripts of anyone who read the review.
func Hidden(layout paths.Layout) []string {
	home, _ := os.UserHomeDir()
	return []string{layout.PRs, layout.CompareResults, layout.Rivals, layout.Archive, filepath.Join(home, ".claude", "projects")}
}

// RunSettings is a reviewer's sandbox: Hidden, and whatever was in the shared temp folder when the run
// started, unreadable.
func RunSettings(layout paths.Layout, shared sandbox.SharedTemp, existing map[string]bool) map[string]any {
	s := sandbox.Settings(append(Hidden(layout), shared.Stale(existing)...))
	// Every run uses the one Python environment: none may change it for the runs after.
	venv := []string{layout.Venv}
	if real, err := filepath.EvalSymlinks(layout.Venv); err == nil && real != layout.Venv {
		venv = append(venv, real)
	}
	s["sandbox"].(map[string]any)["filesystem"].(map[string]any)["denyWrite"] = venv
	perms := s["permissions"].(map[string]any)
	deny := perms["deny"].([]string)
	for _, v := range venv {
		deny = append(deny, "Edit(/"+v+"/**)", "Write(/"+v+"/**)")
	}
	perms["deny"] = deny
	return s
}

var codeSpans = regexp.MustCompile("(?s)```.*?```|`[^`]*`")

// Snippets is every eight words in a row of the human comments' prose. Code is left out, since a
// comment that quotes code quotes what is in the repo anyway.
func Snippets(human []Comment) []string {
	var found []string
	for _, c := range human {
		words := text.Tokens(codeSpans.ReplaceAllString(c.Body, " "))
		for i := 0; i+8 <= len(words); i++ {
			found = append(found, strings.Join(words[i:i+8], " "))
		}
	}
	return found
}

func parts(events []agent.Event, kind string) []map[string]any {
	var out []map[string]any
	for _, e := range events {
		msg, ok := e["message"].(map[string]any)
		if !ok {
			continue
		}
		content, _ := msg["content"].([]any)
		for _, raw := range content {
			if part, ok := raw.(map[string]any); ok && part["type"] == kind {
				out = append(out, part)
			}
		}
	}
	return out
}

// Leaked says whether a tool result the run read holds a stretch of the human review.
func Leaked(events []agent.Event, human []Comment) bool {
	wanted := Snippets(human)
	for _, part := range parts(events, "tool_result") {
		body, ok := part["content"].(string)
		if !ok {
			raw, _ := json.Marshal(part["content"])
			body = string(raw)
		}
		read := strings.Join(text.Tokens(body), " ")
		for _, s := range wanted {
			if strings.Contains(read, s) {
				return true
			}
		}
	}
	return false
}

// ReachedForHidden says whether a tool call named a hidden place. The sandbox denies the read; this
// records the try.
func ReachedForHidden(events []agent.Event) bool {
	for _, part := range parts(events, "tool_use") {
		input, _ := json.Marshal(part["input"])
		for _, m := range markers {
			if strings.Contains(string(input), m) {
				return true
			}
		}
	}
	return false
}
