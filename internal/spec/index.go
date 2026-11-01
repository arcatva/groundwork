package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Index is the skill's own metadata, `skills/groundwork/index.json`: the flow's steps and how one may
// follow another, the reviewers and their rule files, and when to read each rule file. The tables in
// SKILL.md and review.md are generated from it.
type Index struct {
	Steps     []Step     `json:"steps" check:"min=1"`
	Reviewers []Reviewer `json:"reviewers" check:"min=1"`
	Rules     []Rule     `json:"rules" check:"min=1"`
}

// Step is one step of the flow.
type Step struct {
	Name     string   `json:"name" check:"required"`
	Produces string   `json:"produces" check:"required"`
	File     string   `json:"file" check:"required"`
	Start    bool     `json:"start" desc:"a session may begin here"`
	After    []string `json:"after" desc:"the steps it may follow"`
}

// Reviewer is one of the review's reviewers.
type Reviewer struct {
	Name     string   `json:"name" check:"required"`
	Question string   `json:"question" check:"required"`
	Before   string   `json:"before" desc:"what the reviewer reads before its rule files"`
	Files    []string `json:"files" desc:"its rule files, under standard/"`
}

// Rule is one rule file and when to read it.
type Rule struct {
	File     string `json:"file" check:"required"`
	ReadWhen string `json:"read_when" check:"required"`
}

// Check holds the steps' order to steps that exist, and each reviewer's files to rule files.
func (ix Index) Check() []error {
	var errs []error
	names := map[string]bool{}
	for _, s := range ix.Steps {
		names[s.Name] = true
	}
	for _, s := range ix.Steps {
		for _, a := range s.After {
			if !names[a] {
				errs = append(errs, fmt.Errorf("step %s follows %q, which is no step", s.Name, a))
			}
		}
	}
	rules := map[string]bool{}
	for _, r := range ix.Rules {
		rules[strings.TrimPrefix(r.File, "standard/")] = true
	}
	for _, rv := range ix.Reviewers {
		for _, f := range rv.Files {
			if !rules[f] {
				errs = append(errs, fmt.Errorf("reviewer %s reads %q, which is no rule file", rv.Name, f))
			}
		}
	}
	return errs
}

// LoadIndex reads and validates the index of the skill in dir.
func LoadIndex(skill string) (Index, error) {
	b, err := os.ReadFile(filepath.Join(skill, "index.json"))
	if err != nil {
		return Index{}, err
	}
	return ParseIndex(b)
}

// ParseIndex decodes and validates an index.
func ParseIndex(b []byte) (Index, error) {
	var ix Index
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ix); err != nil {
		return ix, fmt.Errorf("index.json: %w", err)
	}
	if err := Validate(ix); err != nil {
		return ix, fmt.Errorf("index.json: %w", err)
	}
	return ix, nil
}

// CheckLog holds a run log to the flow: each entry's step may follow the one before it, or the entry
// names the step it went past (Skipped), with the user's word, and that step bridges the gap.
func (ix Index) CheckLog(entries []LogEntry) []error {
	var errs []error
	prev := ""
	for i, e := range entries {
		ok := ix.Follows(e.Step, prev)
		for _, s := range e.Skipped {
			ok = ok || (ix.Follows(s.Step, prev) && ix.Follows(e.Step, s.Step))
		}
		if !ok {
			from := prev
			if from == "" {
				from = "the start"
			}
			errs = append(errs, fmt.Errorf("entry %d: %s cannot follow %s unless the entry names the step it went past, with the user's word", i+1, e.Step, from))
		}
		prev = e.Step
	}
	return errs
}

// Follows says whether step may come after prev ("" for the start of a log). A step may repeat; a
// start step may begin new work after anything; any other step needs one of its After steps first.
func (ix Index) Follows(step, prev string) bool {
	for _, s := range ix.Steps {
		if s.Name == step {
			return s.Start || step == prev || (prev != "" && contains(s.After, prev))
		}
	}
	return false
}
