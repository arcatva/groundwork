// Package scenario loads the eval scenarios and builds the repo each one runs in.
package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/fsx"
	"github.com/arcatva/groundwork/internal/gitx"
)

// Scenario is a prompt, a rubric, and optionally a branch over the fixture. Process numbers the
// criteria only the skill's own process can meet (its log, its reviewers); the outcome score leaves
// them out, so a run without the skill is compared like for like.
type Scenario struct {
	Name    string
	Path    string
	Prompt  string
	Rubric  []string
	Branch  string
	Process map[int]bool
}

type spec struct {
	Prompt  string   `json:"prompt"`
	Rubric  []string `json:"rubric"`
	Branch  string   `json:"branch"`
	Process []int    `json:"process"`
}

var keys = map[string]bool{"prompt": true, "rubric": true, "branch": true, "process": true}

// Load reads a scenario folder and returns an error where it breaks the contract the runner reads.
func Load(path string) (Scenario, error) {
	name := filepath.Base(path)
	raw, err := os.ReadFile(filepath.Join(path, "scenario.json"))
	if err != nil {
		return Scenario{}, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return Scenario{}, fmt.Errorf("%s: %w", name, err)
	}
	for key := range all {
		if !keys[key] {
			return Scenario{}, fmt.Errorf("%s: unknown key %q", name, key)
		}
	}
	var s spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return Scenario{}, fmt.Errorf("%s: %w", name, err)
	}
	if strings.TrimSpace(s.Prompt) == "" {
		return Scenario{}, fmt.Errorf("%s: a prompt is required", name)
	}
	if len(s.Rubric) == 0 {
		return Scenario{}, fmt.Errorf("%s: the rubric is empty", name)
	}
	for i, line := range s.Rubric {
		if strings.TrimSpace(line) == "" {
			return Scenario{}, fmt.Errorf("%s: rubric line %d is empty", name, i+1)
		}
	}
	process := map[int]bool{}
	for _, n := range s.Process {
		if n < 1 || n > len(s.Rubric) {
			return Scenario{}, fmt.Errorf("%s: process number %d is outside 1..%d", name, n, len(s.Rubric))
		}
		process[n] = true
	}
	if s.Branch != "" {
		if info, err := os.Stat(filepath.Join(path, "overlay")); err != nil || !info.IsDir() {
			return Scenario{}, fmt.Errorf("%s: a branch needs an overlay folder", name)
		}
	}
	return Scenario{name, path, s.Prompt, s.Rubric, s.Branch, process}, nil
}

// All loads every scenario under dir, or only the one named.
func All(dir, only string) ([]Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Scenario
	for _, e := range entries {
		if !e.IsDir() || (only != "" && e.Name() != only) {
			continue
		}
		s, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Workspace builds the fixture as a repo on `main`, and the scenario's branch on top when it has one.
func (s Scenario) Workspace(fixture, work string) error {
	skip := func(name string) bool { return name == "__pycache__" }
	if err := fsx.CopyTree(fixture, work, skip); err != nil {
		return err
	}
	steps := [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"commit", "-qm", "base"}}
	for _, step := range steps {
		if _, err := gitx.Git(work, step...); err != nil {
			return err
		}
	}
	if s.Branch == "" {
		return nil
	}
	if _, err := gitx.Git(work, "checkout", "-qb", s.Branch); err != nil {
		return err
	}
	if err := fsx.CopyTree(filepath.Join(s.Path, "overlay"), work, skip); err != nil {
		return err
	}
	// `add -A`, not `commit -a`: an overlay's new files belong to the branch too.
	if _, err := gitx.Git(work, "add", "-A"); err != nil {
		return err
	}
	_, err := gitx.Git(work, "commit", "-qm", s.Branch)
	return err
}

// Version says which skill text a run used.
type Version struct {
	Commit      string `json:"commit"`
	Uncommitted bool   `json:"uncommitted_changes"`
	SkillSHA256 string `json:"skill_sha256"`
	SkillWords  int    `json:"skill_words"`
}

// Snapshot copies the plugin at root into dest, so edits made during a run do not leak into it, and
// says which version it is: the commit, whether the skill had uncommitted edits, and a hash of its text.
func Snapshot(root, dest string) (Version, error) {
	for _, part := range []string{".claude-plugin", "skills"} {
		if err := fsx.CopyTree(filepath.Join(root, part), filepath.Join(dest, part), nil); err != nil {
			return Version{}, err
		}
	}
	sha, words, err := Fingerprint(filepath.Join(dest, "skills"))
	if err != nil {
		return Version{}, err
	}
	status := exec.Command("git", "status", "--porcelain", "--", "skills", ".claude-plugin")
	status.Dir = root
	out, _ := status.Output()
	return Version{gitx.Head(root), strings.TrimSpace(string(out)) != "", sha, words}, nil
}

// Fingerprint hashes every markdown file under dir in path order, and counts their words.
func Fingerprint(dir string) (string, int, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(files)
	h := sha256.New()
	words := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return "", 0, err
		}
		h.Write(b)
		words += len(strings.Fields(string(b)))
	}
	return hex.EncodeToString(h.Sum(nil))[:12], words, nil
}
