package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcatva/groundwork"
)

func index(t *testing.T) Index {
	t.Helper()
	ix, err := ParseIndex(groundwork.Index)
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

func entry(step string, skipped ...Skip) LogEntry {
	return LogEntry{Step: step, Skipped: skipped}
}

func TestShipNeedsAReviewOrTheUsersWordToSkipIt(t *testing.T) {
	ix := index(t)
	if errs := ix.CheckLog([]LogEntry{entry("Build"), entry("Review"), entry("Ship")}); len(errs) > 0 {
		t.Errorf("build, review, ship was refused: %v", errs)
	}
	if errs := ix.CheckLog([]LogEntry{entry("Build"), entry("Ship")}); len(errs) == 0 {
		t.Error("ship straight after build was accepted")
	}
	skipped := entry("Ship", Skip{Step: "Review", Because: `"no review, ship it"`})
	if errs := ix.CheckLog([]LogEntry{entry("Build"), skipped}); len(errs) > 0 {
		t.Errorf("a skipped review with the user's word was refused: %v", errs)
	}
	if errs := ix.CheckLog([]LogEntry{entry("Plan")}); len(errs) == 0 {
		t.Error("a log that starts at Plan was accepted")
	}
}

func TestAQuestionRecommendsOneOfItsOptions(t *testing.T) {
	q := Question{Number: 1, Decision: "d", Now: "n", Sources: "s", Why: "w", Recommend: "c",
		Options: []Option{{"a", "x", "e"}, {"b", "y", "e"}}}
	if err := Validate(q); err == nil || !strings.Contains(err.Error(), "not one of its options") {
		t.Errorf("a recommendation outside the options passed: %v", err)
	}
	q.Recommend = "a"
	if err := Validate(q); err != nil {
		t.Errorf("a good question failed: %v", err)
	}
	q.Options = q.Options[:1]
	if err := Validate(q); err == nil {
		t.Error("a question with one option passed")
	}
}

func TestOnlyDesignHoldsANameInventory(t *testing.T) {
	if err := Validate(ReviewerReport{Reviewer: "Design"}); err == nil {
		t.Error("a Design report with no name inventory passed")
	}
	claims := ReviewerReport{Reviewer: "Claims", NameInventory: []Name{{Name: "x", Holds: "h", Elsewhere: "e"}}}
	if err := Validate(claims); err == nil {
		t.Error("a Claims report with a name inventory passed")
	}
	if err := Validate(ReviewerReport{Reviewer: "Tester"}); err == nil {
		t.Error("an unknown reviewer passed")
	}
}

func TestCheckReportRefusesUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.json")
	if err := os.WriteFile(path, []byte(`{"reviewer": "Behaviour", "findings": [], "severity": "high"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckReport(path); err == nil {
		t.Error("a field the contract does not have passed")
	}
}
