// Package paths says where the repo keeps the skill, the evals and their data.
package paths

import (
	"errors"
	"os"
	"path/filepath"
)

// Layout is the repo's folders. Everything under Results and Data is ignored by git.
type Layout struct {
	Root           string
	Skill          string
	Evals          string
	Fixture        string
	Scenarios      string
	Results        string
	CompareResults string
	Data           string
	PRs            string
	Rivals         string
	Archive        string
	Venv           string
}

// At lays out the repo rooted at root.
func At(root string) Layout {
	evals := filepath.Join(root, "evals")
	data := filepath.Join(evals, "compare-data")
	return Layout{
		Root:           root,
		Skill:          filepath.Join(root, "skills", "groundwork"),
		Evals:          evals,
		Fixture:        filepath.Join(evals, "fixture"),
		Scenarios:      filepath.Join(evals, "scenarios"),
		Results:        filepath.Join(evals, "results"),
		CompareResults: filepath.Join(evals, "results", "compare"),
		Data:           data,
		PRs:            filepath.Join(data, "prs"),
		Rivals:         filepath.Join(data, "rivals"),
		Archive:        filepath.Join(data, "tmp-archive"),
		Venv:           filepath.Join(data, "venv"),
	}
}

// Find walks up from the working directory to the repo root: the folder that holds the skill.
func Find() (Layout, error) {
	dir, err := os.Getwd()
	if err != nil {
		return Layout{}, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "skills", "groundwork", "SKILL.md")); err == nil {
			return At(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Layout{}, errors.New("not inside the groundwork repo: no skills/groundwork/SKILL.md above here")
		}
		dir = parent
	}
}
