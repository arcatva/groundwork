package sandbox

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSettingsHideFromShellAndReadTool(t *testing.T) {
	s := Settings([]string{"/a/prs", "/a/prs", "/b/results"})
	deny := s["sandbox"].(map[string]any)["filesystem"].(map[string]any)["denyRead"].([]string)
	if strings.Join(deny, ",") != "/a/prs,/b/results" {
		t.Errorf("denyRead = %v", deny)
	}
	rules := s["permissions"].(map[string]any)["deny"].([]string)
	if rules[0] != "Read(//a/prs/**)" {
		t.Errorf("Read rule = %q", rules[0])
	}
	if s["disableAllHooks"] != true {
		t.Error("hooks are not disabled")
	}
	if _, ok := Settings(nil)["permissions"]; ok {
		t.Error("nothing hidden, yet a deny rule")
	}
}

func TestCollided(t *testing.T) {
	if !Collided("fatal: '/tmp/claude-502/basewt' already exists") {
		t.Error("a leftover in the way was not flagged")
	}
	if Collided("Preparing worktree (new branch 'x')") || Collided("git worktree add /tmp/claude-502/review-base pr") {
		t.Error("using the temp folder was flagged")
	}
}

func TestTidyMovesLeftoversAndKeepsClaudesOwn(t *testing.T) {
	dir := t.TempDir()
	shared := SharedTemp{Root: filepath.Join(dir, "shared"), Archive: filepath.Join(dir, "archive")}
	names := []string{"basewt", "-session-dir", "cache-123", "bash-edit-diff"}
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(shared.Root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	created := map[string]bool{}
	for _, n := range names {
		created[n] = true
	}
	moved, err := shared.Tidy(created, "run1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(moved, ",") != "basewt" {
		t.Errorf("moved %v, want [basewt]", moved)
	}
	if _, err := os.Stat(filepath.Join(shared.Archive, "run1", "basewt")); err != nil {
		t.Error("the leftover is not in the archive")
	}
	var left []string
	for n := range shared.Entries() {
		left = append(left, n)
	}
	sort.Strings(left)
	if strings.Join(left, ",") != "-session-dir,bash-edit-diff,cache-123" {
		t.Errorf("left %v", left)
	}
}

func TestDiff(t *testing.T) {
	got := Diff(map[string]bool{"a": true, "b": true}, map[string]bool{"a": true})
	if len(got) != 1 || !got["b"] {
		t.Errorf("got %v", got)
	}
}
