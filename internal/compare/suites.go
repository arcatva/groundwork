package compare

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/fsx"
	"github.com/arcatva/groundwork/internal/gitx"
	"github.com/arcatva/groundwork/internal/paths"
)

// Suite is a skill suite under comparison. A suite with no Invoke is the baseline: no skill at all.
type Suite struct {
	Name   string
	Repo   string // "" for this repo
	Skills string // the folder of the suite's skills, inside its repo
	Invoke string // how its review skill is started
}

// All lists every suite compared, each loaded from all of its skills' markdown as one plugin.
var All = []Suite{
	{Name: "groundwork", Skills: "skills", Invoke: "/groundwork:groundwork"},
	{Name: "superpowers", Repo: "obra/superpowers", Skills: "skills", Invoke: "/superpowers:requesting-code-review"},
	{Name: "matt", Repo: "mattpocock/skills", Skills: "skills/engineering", Invoke: "/matt:code-review"},
	{Name: "pstack", Repo: "michael-denyer/pstack-claude", Skills: "plugins/pstack/skills", Invoke: "/pstack:interrogate"},
	{Name: "baseline"},
}

// Dev holds the PRs groundwork was tuned on, reported apart from the held-out ones.
var Dev = map[string]bool{"22154": true}

// Named returns the suite called name.
func Named(name string) (Suite, bool) {
	for _, s := range All {
		if s.Name == name {
			return s, true
		}
	}
	return Suite{}, false
}

func (s Suite) source(layout paths.Layout) string {
	if s.Repo == "" {
		return layout.Root
	}
	return filepath.Join(layout.Rivals, s.Name)
}

// Plugin copies the suite's markdown alone (no hooks, no scripts) into dest as a plugin, and says
// where it came from: its repo, its commit, and a hash of the text, since a working tree can differ
// from its commit.
func (s Suite) Plugin(layout paths.Layout, dest string) (Origin, error) {
	source := s.source(layout)
	skills := filepath.Join(source, s.Skills)
	var files []string
	err := filepath.WalkDir(skills, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, _ := filepath.Rel(skills, path)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return Origin{}, err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		target := filepath.Join(dest, "skills", rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Origin{}, err
		}
		if err := fsx.CopyTree(filepath.Join(skills, rel), target, nil); err != nil {
			return Origin{}, err
		}
		b, _ := os.ReadFile(target)
		h.Write(b)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".claude-plugin"), 0o755); err != nil {
		return Origin{}, err
	}
	manifest := `{"name": "` + s.Name + `", "version": "compare"}`
	if err := os.WriteFile(filepath.Join(dest, ".claude-plugin", "plugin.json"), []byte(manifest), 0o644); err != nil {
		return Origin{}, err
	}
	repo := s.Repo
	if repo == "" {
		repo = "this repo"
	}
	return Origin{Source: repo, Commit: gitx.Head(source), SkillSHA256: hex.EncodeToString(h.Sum(nil))[:12]}, nil
}
