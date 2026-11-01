// Package lint checks the repo's own rules: the skill's structure, its word budget, its neutrality,
// the agents' manifests, and the eval scenarios' contract. `groundwork lint` runs it; so does CI.
package lint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/gen"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/scenario"
)

// WordBudget is the skill's word limit. A change that needs more words removes others, or raises the
// budget in the same commit with the reason in its message.
const WordBudget = 6300

// Problem is one broken rule, where it is broken.
type Problem struct {
	Check string
	File  string
	Line  int
	Text  string
}

func (p Problem) String() string {
	where := p.File
	if p.Line > 0 {
		where = fmt.Sprintf("%s:%d", p.File, p.Line)
	}
	return fmt.Sprintf("%s: %s: %s", p.Check, where, p.Text)
}

// Run runs every check on the repo and returns what it found, in a stable order.
func Run(layout paths.Layout) []Problem {
	var all []Problem
	for _, check := range []func(paths.Layout) []Problem{Structure, Generated, Lean, Neutral, Manifests, Scenarios} {
		all = append(all, check(layout)...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].String() < all[j].String() })
	return all
}

type skill struct {
	root      string
	standards map[string]string // stem -> path relative to standard/
	text      map[string]string // path relative to the skill -> text
}

func load(layout paths.Layout) (skill, error) {
	s := skill{root: layout.Skill, standards: map[string]string{}, text: map[string]string{}}
	err := filepath.WalkDir(layout.Skill, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, _ := filepath.Rel(layout.Skill, path)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s.text[rel] = string(b)
		if strings.HasPrefix(rel, "standard/") {
			s.standards[strings.TrimSuffix(filepath.Base(rel), ".md")] = strings.TrimPrefix(rel, "standard/")
		}
		return nil
	})
	return s, err
}

func (s skill) flow() []string {
	var out []string
	for rel := range s.text {
		if rel == "review.md" || strings.HasPrefix(rel, "flow/") {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

var (
	reviewerRow   = regexp.MustCompile(`(?m)^\| (\w+) \| .*? \| (.*) \|$`)
	standardPtr   = regexp.MustCompile("`(?:standard/)?((?:evolvability|functional)/[a-z-]+\\.md)`")
	stepPtr       = regexp.MustCompile("`(flow/[a-z-]+\\.md|review\\.md|contracts\\.md)`")
	mdName        = regexp.MustCompile(`\b([a-z-]+)\.md\b`)
	normative     = regexp.MustCompile(`\bMUST\b|\bSHOULD\b|\b[Hh]ouse:|\b[Hh]ouse rule\b|\b[Pp]refer\b`)
	noReviewer    = map[string]bool{"visual-representation": true}
	reviewerNames = []string{"Behaviour", "Claims", "Design"}
)

// Structure: each standard file belongs to exactly one reviewer and is indexed in SKILL.md; every
// pointer names a file that exists; standard files stand alone; the flow and the review state no
// rule of their own, only point at standard/.
func Structure(layout paths.Layout) []Problem {
	s, err := load(layout)
	if err != nil {
		return []Problem{{"structure", layout.Skill, 0, err.Error()}}
	}
	var out []Problem
	add := func(file string, line int, format string, args ...any) {
		out = append(out, Problem{"structure", file, line, fmt.Sprintf(format, args...)})
	}
	review := s.text["review.md"]
	_, table, _ := strings.Cut(review, "| reviewer | its question | its files |")
	table, _, _ = strings.Cut(table, "\n\n")
	rows := map[string]string{}
	for _, m := range reviewerRow.FindAllStringSubmatch(table, -1) {
		rows[m[1]] = m[2]
	}
	var got []string
	for name := range rows {
		got = append(got, name)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(reviewerNames, ",") {
		add("review.md", 0, "reviewers are %v, want %v", got, reviewerNames)
	}
	for stem, rel := range s.standards {
		if noReviewer[stem] {
			continue
		}
		owners := 0
		for _, files := range rows {
			if strings.Contains(files, "`"+rel+"`") {
				owners++
			}
		}
		if owners != 1 {
			add("review.md", 0, "standard/%s belongs to %d reviewers, want 1", rel, owners)
		}
		if !strings.Contains(s.text["SKILL.md"], "`standard/"+rel+"`") {
			add("SKILL.md", 0, "standard/%s is not in the index", rel)
		}
	}
	for rel, text := range s.text {
		if rel != "SKILL.md" && rel != "review.md" && rel != "contracts.md" && !strings.HasPrefix(rel, "flow/") && !strings.HasPrefix(rel, "standard/") {
			continue
		}
		for _, m := range standardPtr.FindAllStringSubmatch(text, -1) {
			if _, err := os.Stat(filepath.Join(s.root, "standard", m[1])); err != nil {
				add(rel, 0, "points at standard/%s, which does not exist", m[1])
			}
		}
		for _, m := range stepPtr.FindAllStringSubmatch(text, -1) {
			if _, err := os.Stat(filepath.Join(s.root, m[1])); err != nil {
				add(rel, 0, "points at %s, which does not exist", m[1])
			}
		}
	}
	for stem, rel := range s.standards {
		for _, m := range mdName.FindAllStringSubmatch(s.text["standard/"+rel], -1) {
			if m[1] != stem && (m[1] == "review" || s.standards[m[1]] != "") {
				add("standard/"+rel, 0, "points at %s.md: give the concern one file", m[1])
			}
		}
	}
	for _, rel := range s.flow() {
		for i, line := range strings.Split(s.text[rel], "\n") {
			if normative.MatchString(line) {
				add(rel, i+1, "states a rule of its own; point at standard/ instead: %s", strings.TrimSpace(line))
			}
		}
	}
	return out
}

var (
	heading      = regexp.MustCompile(`(?m)^## (.+)$`)
	quotedOne    = regexp.MustCompile("\"([A-Z][a-z]+)\"\\s+(?:in|of)\\s+(?:>\\s+)?`SKILL\\.md`")
	quotedList   = regexp.MustCompile("sections\\s+((?:\"[A-Z][a-z]+\",?\\s+(?:and\\s+)?)+)of\\s+(?:>\\s+)?`SKILL\\.md`")
	quotedInList = regexp.MustCompile(`"([A-Z][a-z]+)"`)
)

var comment = regexp.MustCompile(`(?s)<!--.*?-->`)

// Generated: the tables and schemas made from the skill's index are as `groundwork gen` would write
// them, so a hand edit cannot drift from the index.
func Generated(layout paths.Layout) []Problem {
	stale, err := gen.Stale(layout.Skill)
	if err != nil {
		return []Problem{{"generated", "skills/groundwork/index.json", 0, err.Error()}}
	}
	var out []Problem
	for _, f := range stale {
		out = append(out, Problem{"generated", "skills/groundwork/" + f, 0, "differs from what `groundwork gen` writes; edit index.json and run it"})
	}
	return out
}

// Lean: the skill stays within its word budget, and every section a file quotes by name exists.
func Lean(layout paths.Layout) []Problem {
	s, err := load(layout)
	if err != nil {
		return []Problem{{"lean", layout.Skill, 0, err.Error()}}
	}
	var out []Problem
	words := 0
	for _, text := range s.text {
		// Generation markers are HTML comments, not instructions: they do not count.
		words += len(strings.Fields(comment.ReplaceAllString(text, " ")))
	}
	if words > WordBudget {
		out = append(out, Problem{"lean", "skills/groundwork", 0, fmt.Sprintf("the skill is %d words; the budget is %d", words, WordBudget)})
	}
	headings := map[string]bool{}
	for _, m := range heading.FindAllStringSubmatch(s.text["SKILL.md"], -1) {
		headings[strings.TrimSpace(m[1])] = true
	}
	for rel, text := range s.text {
		var names []string
		for _, m := range quotedOne.FindAllStringSubmatch(text, -1) {
			names = append(names, m[1])
		}
		for _, m := range quotedList.FindAllStringSubmatch(text, -1) {
			for _, n := range quotedInList.FindAllStringSubmatch(m[1], -1) {
				names = append(names, n[1])
			}
		}
		for _, name := range names {
			if !headings[name] {
				out = append(out, Problem{"lean", rel, 0, fmt.Sprintf("quotes section %q of SKILL.md, which has no such heading", name)})
			}
		}
	}
	return out
}

var (
	vendors   = regexp.MustCompile(`(?i)figma|confluence|jira|atlassian|\bslack\b|datadog|sentry\.io|notion\.so|linear\.app|\bsonnet\b|\bopus\b|\bhaiku\b|\bgpt-\d`)
	agentName = regexp.MustCompile(`(?i)\b(gemini|claude|codex|cursor)\b(\.md)?`)
	mrWord    = regexp.MustCompile(`\bMRs?\b`)
)

// Neutral: inside the skill, no vendor a reader must use and no model only one provider sells (an
// agent's name may appear only as an instruction file's name, such as CLAUDE.md); a merge request is
// called a PR; and no word listed in the untracked .neutral-words file is in any tracked file.
func Neutral(layout paths.Layout) []Problem {
	s, err := load(layout)
	if err != nil {
		return []Problem{{"neutral", layout.Skill, 0, err.Error()}}
	}
	var out []Problem
	for rel, text := range s.text {
		for i, line := range strings.Split(text, "\n") {
			bad := vendors.FindString(line)
			for _, m := range agentName.FindAllStringSubmatch(line, -1) {
				if m[2] == "" {
					bad = m[1]
				}
			}
			if bad != "" {
				out = append(out, Problem{"neutral", "skills/groundwork/" + rel, i + 1, fmt.Sprintf("names %q: %s", bad, strings.TrimSpace(line))})
			}
			if mrWord.MatchString(line) {
				out = append(out, Problem{"neutral", "skills/groundwork/" + rel, i + 1, "calls a pull request an MR: " + strings.TrimSpace(line)})
			}
		}
	}
	if readme, err := os.ReadFile(filepath.Join(layout.Root, "README.md")); err == nil {
		for i, line := range strings.Split(string(readme), "\n") {
			if mrWord.MatchString(line) {
				out = append(out, Problem{"neutral", "README.md", i + 1, "calls a pull request an MR: " + strings.TrimSpace(line)})
			}
		}
	}
	private, err := privateWords(layout.Root)
	if err != nil || private == nil {
		return out
	}
	files, err := trackedText(layout.Root)
	if err != nil {
		return append(out, Problem{"neutral", ".", 0, err.Error()})
	}
	for _, rel := range files {
		f, err := os.Open(filepath.Join(layout.Root, rel))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for n := 1; sc.Scan(); n++ {
			if w := private.FindString(sc.Text()); w != "" {
				out = append(out, Problem{"neutral", rel, n, "holds a private word listed in .neutral-words"})
			}
		}
		f.Close()
	}
	return out
}

func privateWords(root string) (*regexp.Regexp, error) {
	b, err := os.ReadFile(filepath.Join(root, ".neutral-words"))
	if err != nil {
		return nil, nil // no list, nothing to check
	}
	var words []string
	for _, w := range strings.Split(string(b), "\n") {
		if w = strings.TrimSpace(w); w != "" {
			words = append(words, regexp.QuoteMeta(w))
		}
	}
	if len(words) == 0 {
		return nil, nil
	}
	// Whole words only: a short private word is often a piece of an ordinary one.
	return regexp.Compile(`(?i)\b(?:` + strings.Join(words, "|") + `)\b`)
}

var textSuffix = map[string]bool{".md": true, ".json": true, ".go": true, ".sh": true, ".txt": true, ".yml": true, ".yaml": true, ".mod": true, ".py": true, "": true}

func trackedText(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rel := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		info, err := os.Lstat(filepath.Join(root, rel))
		if err != nil || !info.Mode().IsRegular() || !textSuffix[filepath.Ext(rel)] {
			continue
		}
		files = append(files, rel)
	}
	return files, nil
}

type manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Plugins     []struct {
		Name string `json:"name"`
	} `json:"plugins"`
}

// Manifests: every agent's manifest describes the same plugin (one name, version and description),
// every marketplace lists it, and SKILL.md's heading states its version, so a run log can be traced
// to the skill text.
func Manifests(layout paths.Layout) []Problem {
	var out []Problem
	read := func(rel string) (manifest, bool) {
		var m manifest
		b, err := os.ReadFile(filepath.Join(layout.Root, rel))
		if err == nil {
			err = json.Unmarshal(b, &m)
		}
		if err != nil {
			out = append(out, Problem{"manifests", rel, 0, err.Error()})
			return m, false
		}
		return m, true
	}
	ref, ok := read(".claude-plugin/plugin.json")
	if !ok {
		return out
	}
	plugins, _ := filepath.Glob(filepath.Join(layout.Root, ".*-plugin", "plugin.json"))
	for _, path := range plugins {
		rel, _ := filepath.Rel(layout.Root, path)
		m, ok := read(rel)
		if ok && (m.Name != ref.Name || m.Version != ref.Version || m.Description != ref.Description) {
			out = append(out, Problem{"manifests", rel, 0, "name, version or description differs from .claude-plugin/plugin.json"})
		}
	}
	markets, _ := filepath.Glob(filepath.Join(layout.Root, ".*-plugin", "marketplace.json"))
	markets = append(markets, filepath.Join(layout.Root, ".agents", "plugins", "marketplace.json"))
	for _, path := range markets {
		rel, _ := filepath.Rel(layout.Root, path)
		m, ok := read(rel)
		if !ok {
			continue
		}
		listed := false
		for _, p := range m.Plugins {
			listed = listed || p.Name == ref.Name
		}
		if !listed {
			out = append(out, Problem{"manifests", rel, 0, "does not list " + ref.Name})
		}
	}
	skillMD, err := os.ReadFile(filepath.Join(layout.Skill, "SKILL.md"))
	if err != nil {
		return append(out, Problem{"manifests", "skills/groundwork/SKILL.md", 0, err.Error()})
	}
	_, after, _ := strings.Cut(string(skillMD), "\n# ")
	title, _, _ := strings.Cut(after, "\n")
	if want := "Groundwork " + ref.Version; title != want {
		out = append(out, Problem{"manifests", "skills/groundwork/SKILL.md", 0, fmt.Sprintf("heading is %q, want %q", title, want)})
	}
	return out
}

// Scenarios: every eval scenario keeps the contract the runner reads.
func Scenarios(layout paths.Layout) []Problem {
	if _, err := scenario.All(layout.Scenarios, ""); err != nil {
		return []Problem{{"scenarios", "evals/scenarios", 0, err.Error()}}
	}
	return nil
}
