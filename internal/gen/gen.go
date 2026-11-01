// Package gen writes what is generated from the skill's index and contracts: the tables in SKILL.md
// and review.md, and the JSON Schemas under schemas/. `groundwork gen` writes them; lint fails when a
// committed file differs from what gen would write.
package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arcatva/groundwork/internal/spec"
)

// File is a generated file's path, relative to the skill, and its content.
type File struct {
	Path    string
	Content string
}

// region is a generated table inside a markdown file, between `<!-- gen:NAME -->` and `<!-- /gen -->`.
type region struct {
	file, name string
	render     func(spec.Index) string
}

var regions = []region{
	{"SKILL.md", "steps", steps},
	{"SKILL.md", "rules", rules},
	{"review.md", "reviewers", reviewers},
}

func steps(ix spec.Index) string {
	var b strings.Builder
	b.WriteString("| step | produces | file |\n|---|---|---|\n")
	for _, s := range ix.Steps {
		fmt.Fprintf(&b, "| %s | %s | `%s` |\n", s.Name, s.Produces, s.File)
	}
	return b.String()
}

func rules(ix spec.Index) string {
	var b strings.Builder
	b.WriteString("| file | read when |\n|---|---|\n")
	for _, r := range ix.Rules {
		fmt.Fprintf(&b, "| `%s` | %s |\n", r.File, r.ReadWhen)
	}
	return b.String()
}

func reviewers(ix spec.Index) string {
	var b strings.Builder
	b.WriteString("| reviewer | its question | its files |\n|---|---|---|\n")
	for _, r := range ix.Reviewers {
		files := make([]string, len(r.Files))
		for i, f := range r.Files {
			files[i] = "`" + f + "`"
		}
		list := strings.Join(files, ", ")
		if r.Before != "" {
			list = r.Before + " " + list
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Name, r.Question, list)
	}
	return b.String()
}

// Schemas are the contracts a machine checks, by the file each is written to under schemas/.
var Schemas = []struct {
	File  string
	Title string
	Value any
}{
	{"reviewer-report.schema.json", "A reviewer's report", spec.ReviewerReport{}},
	{"log-entry.schema.json", "One entry of the run log, .groundwork/log.jsonl", spec.LogEntry{}},
	{"index.schema.json", "The skill's index: its steps, reviewers and rule files", spec.Index{}},
}

// All returns every generated file of the skill in dir, with its generated parts as they should be.
func All(skill string) ([]File, error) {
	ix, err := spec.LoadIndex(skill)
	if err != nil {
		return nil, err
	}
	var out []File
	texts := map[string]string{}
	for _, r := range regions {
		text, ok := texts[r.file]
		if !ok {
			b, err := os.ReadFile(filepath.Join(skill, r.file))
			if err != nil {
				return nil, err
			}
			text = string(b)
		}
		if text, err = replace(text, r.name, r.render(ix)); err != nil {
			return nil, fmt.Errorf("%s: %w", r.file, err)
		}
		texts[r.file] = text
	}
	for _, name := range []string{"SKILL.md", "review.md"} {
		out = append(out, File{name, texts[name]})
	}
	for _, s := range Schemas {
		b, err := json.MarshalIndent(spec.Schema(s.Value, s.Title), "", "  ")
		if err != nil {
			return nil, err
		}
		out = append(out, File{filepath.Join("schemas", s.File), string(b) + "\n"})
	}
	return out, nil
}

func replace(text, name, body string) (string, error) {
	open, end := "<!-- gen:"+name+" -->\n", "<!-- /gen -->"
	i := strings.Index(text, open)
	if i < 0 {
		return "", fmt.Errorf("no %q marker", strings.TrimSpace(open))
	}
	j := strings.Index(text[i:], end)
	if j < 0 {
		return "", fmt.Errorf("no end marker after %q", strings.TrimSpace(open))
	}
	return text[:i+len(open)] + body + text[i+j:], nil
}

// Write writes every generated file, and returns those it changed.
func Write(skill string) ([]string, error) {
	files, err := All(skill)
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, f := range files {
		path := filepath.Join(skill, f.Path)
		old, _ := os.ReadFile(path)
		if string(old) == f.Content {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return changed, err
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return changed, err
		}
		changed = append(changed, f.Path)
	}
	return changed, nil
}

// Stale returns the generated files whose committed content differs from what Write would write.
func Stale(skill string) ([]string, error) {
	files, err := All(skill)
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, f := range files {
		old, _ := os.ReadFile(filepath.Join(skill, f.Path))
		if string(old) != f.Content {
			stale = append(stale, f.Path)
		}
	}
	return stale, nil
}
