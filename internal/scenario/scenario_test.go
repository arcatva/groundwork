package scenario

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadHoldsTheContract(t *testing.T) {
	for name, spec := range map[string]string{
		"no prompt":          `{"rubric": ["x"]}`,
		"empty rubric":       `{"prompt": "x", "rubric": []}`,
		"unknown key":        `{"prompt": "x", "rubric": ["x"], "extra": 1}`,
		"process too big":    `{"prompt": "x", "rubric": ["x"], "process": [2]}`,
		"branch, no overlay": `{"prompt": "x", "rubric": ["x"], "branch": "feature"}`,
	} {
		dir := filepath.Join(t.TempDir(), "s")
		write(t, filepath.Join(dir, "scenario.json"), spec)
		if _, err := Load(dir); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	dir := filepath.Join(t.TempDir(), "good")
	write(t, filepath.Join(dir, "scenario.json"), `{"prompt": "p", "rubric": ["a", "b"], "process": [2]}`)
	s, err := Load(dir)
	if err != nil || !s.Process[2] || s.Process[1] {
		t.Errorf("a good scenario read wrong: %+v, %v", s, err)
	}
}

// TestWorkspaceCommitsTheOverlaysNewFiles guards a past bug: `commit -a` left an overlay's new files
// untracked, so the branch under review was missing them.
func TestWorkspaceCommitsTheOverlaysNewFiles(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "fixture")
	write(t, filepath.Join(fixture, "app.py"), "x = 1\n")
	dir := filepath.Join(root, "s")
	write(t, filepath.Join(dir, "scenario.json"), `{"prompt": "p", "rubric": ["a"], "branch": "feature"}`)
	write(t, filepath.Join(dir, "overlay", "new.py"), "y = 2\n")
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "work")
	if err := s.Workspace(fixture, work); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "diff", "--name-only", "main...feature")
	cmd.Dir = work
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "new.py" {
		t.Errorf("the branch's diff is %q, want new.py", out)
	}
}

func TestFingerprintIsStable(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "b.md"), "two words")
	write(t, filepath.Join(dir, "a", "c.md"), "three more words")
	write(t, filepath.Join(dir, "skip.txt"), "not counted")
	sha, words, err := Fingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	again, _, _ := Fingerprint(dir)
	if sha != again || words != 5 || len(sha) != 12 {
		t.Errorf("sha %s (again %s), words %d", sha, again, words)
	}
}
