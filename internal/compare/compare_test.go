package compare

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/arcatva/groundwork/internal/agent"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/sandbox"
	"github.com/arcatva/groundwork/internal/text"
)

func TestQuotedSurvivesTablesAndArrows(t *testing.T) {
	report := text.Tokens("| `app/x.py:3` | Off by one: `> limit` lets 6 through → use `>=` |")
	if !Quoted("Off by one: > limit lets 6 through", report) {
		t.Error("a quote from a table row was not found")
	}
	short := text.Tokens("Off by one: > limit lets 6 through")
	for _, q := range []string{"", "off by", "admins are not exempt here"} {
		if Quoted(q, short) {
			t.Errorf("%q counted as quoted", q)
		}
	}
}

func toolResult(s string) []agent.Event {
	return []agent.Event{{"message": map[string]any{"content": []any{map[string]any{"type": "tool_result", "content": s}}}}}
}

func TestLeakedOnlyWhenTheReviewWasRead(t *testing.T) {
	human := []Comment{{Body: "This would be a better host for the check, since the manager already knows the database it reads from."}}
	if !Leaked(toolResult("... since the manager already knows the database it reads from ..."), human) {
		t.Error("a stretch of the human review in a tool result was not flagged")
	}
	if Leaked(toolResult("def using(self, alias): ..."), human) {
		t.Error("code was flagged")
	}
	// The deny list a sandbox notice repeats is not a leak.
	if Leaked(toolResult("read-deny list: /x/evals/compare-data/prs"), human) {
		t.Error("a sandbox notice was flagged")
	}
	quotingCode := []Comment{{Body: "```\nreturn self.get_queryset().using(db)\n```"}}
	if Leaked(toolResult("return self.get_queryset().using(db)"), quotingCode) {
		t.Error("code a comment quotes was flagged")
	}
}

func TestReachedForHidden(t *testing.T) {
	try := []agent.Event{{"message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "input": map[string]any{"command": "cat /x/compare-data/prs/1/human.json"}}}}}}
	if !ReachedForHidden(try) {
		t.Error("a try at the human review was not recorded")
	}
	ok := []agent.Event{{"message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "input": map[string]any{"command": "git worktree add /tmp/claude-502/base pr"}}}}}}
	if ReachedForHidden(ok) {
		t.Error("using the temp folder was recorded as a try")
	}
}

func writeTarball(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []interface{ Close() error }{tw, gz, f} {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUnpackDropsAgentSettingsAndKeepsInstructions(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "x.tgz")
	writeTarball(t, tarball, map[string]string{
		"repo-abc/.claude/settings.json": `{"hooks": {}}`,
		"repo-abc/pkg/.claude/x":         "x",
		"repo-abc/.mcp.json":             "{}",
		"repo-abc/AGENTS.md":             "rules",
		"repo-abc/app.py":                "x = 1\n",
	})
	dest := filepath.Join(dir, "out")
	if err := Unpack(tarball, dest); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{".claude", "pkg/.claude", ".mcp.json"} {
		if _, err := os.Stat(filepath.Join(dest, gone)); err == nil {
			t.Errorf("%s was kept", gone)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "AGENTS.md")); string(b) != "rules" {
		t.Error("the repo's instruction file was not kept")
	}
}

func TestReviewersCannotChangeTheSharedPython(t *testing.T) {
	layout := paths.At(t.TempDir())
	s := RunSettings(layout, sandbox.SharedTemp{Root: t.TempDir()}, nil)
	fs := s["sandbox"].(map[string]any)["filesystem"].(map[string]any)
	if !slices.Contains(fs["denyWrite"].([]string), layout.Venv) {
		t.Error("the shared Python environment is writable")
	}
	if !slices.Contains(s["permissions"].(map[string]any)["deny"].([]string), "Write(/"+layout.Venv+"/**)") {
		t.Error("the Write tool may change the shared Python environment")
	}
}

func TestUnpackRefusesAPathOutOfTheTree(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "x.tgz")
	writeTarball(t, tarball, map[string]string{"repo-abc/../../escape": "x"})
	if err := Unpack(tarball, filepath.Join(dir, "out")); err == nil {
		t.Error("a path out of the tree was extracted")
	}
}

func TestAnonymizedHidesEverySuiteName(t *testing.T) {
	got := Anonymized("Groundwork found it; superpowers did not; MATT agreed.")
	for _, s := range All {
		if strings.Contains(strings.ToLower(got), s.Name) {
			t.Errorf("%q still names %s", got, s.Name)
		}
	}
}

func TestShuffledIsAFixedPermutation(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e"}
	one, again := shuffled(names, "22169-1-0"), shuffled(names, "22169-1-0")
	if !slices.Equal(one, again) {
		t.Error("the same seed gave two orders")
	}
	sorted := slices.Clone(one)
	slices.Sort(sorted)
	if !slices.Equal(sorted, names) {
		t.Errorf("%v is not a permutation of %v", one, names)
	}
}

// TestReportAveragesPerPRThenOverPRs builds two graded PRs by hand and checks the table: a PR's share is
// the mean over its runs, and a suite's score is the mean over PRs, so a PR with many points does not
// outweigh one with few.
func TestReportAveragesPerPRThenOverPRs(t *testing.T) {
	layout := paths.At(t.TempDir())
	results := filepath.Join(layout.CompareResults, "run")
	mustWrite := func(path string, v any) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(path, v); err != nil {
			t.Fatal(err)
		}
	}
	// PR 1 has four points; PR 2 has one.
	mustWrite(filepath.Join(layout.PRs, "1", "points.json"), []Point{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}, {ID: "P4"}})
	mustWrite(filepath.Join(layout.PRs, "2", "points.json"), []Point{{ID: "P1"}})
	grade := func(raised map[string]bool) SuiteGrade {
		votes := map[string][]bool{}
		for id, r := range raised {
			votes[id] = []bool{r, r, r}
		}
		return SuiteGrade{Raised: raised, Votes: votes, OtherFindings: [][]OtherFinding{{}, {}, {}}}
	}
	// "a" raises every point of PR 1 and none of PR 2; "b" raises one of PR 1 and all of PR 2.
	mustWrite(filepath.Join(results, "1-r1.grade.json"), GradeFile{Suites: map[string]SuiteGrade{
		"a": grade(map[string]bool{"P1": true, "P2": true, "P3": true, "P4": true}),
		"b": grade(map[string]bool{"P1": true, "P2": false, "P3": false, "P4": false}),
	}})
	mustWrite(filepath.Join(results, "2-r1.grade.json"), GradeFile{Suites: map[string]SuiteGrade{
		"a": grade(map[string]bool{"P1": false}),
		"b": grade(map[string]bool{"P1": true}),
	}})
	md, err := Report(layout, results)
	if err != nil {
		t.Fatal(err)
	}
	// a: (100% + 0%) / 2 = 50%; b: (25% + 100%) / 2 = 62.5%, shown as 62%.
	for _, want := range []string{"| a | 50% |", "| b | 62% |"} {
		if !strings.Contains(md, want) {
			t.Errorf("the table lacks %q:\n%s", want, md)
		}
	}
	if strings.Index(md, "| b |") > strings.Index(md, "| a |") {
		t.Error("the table is not ordered by score")
	}
}
