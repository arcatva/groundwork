package agent

import (
	"strings"
	"testing"
)

func TestTraceFoldsOnlyBackToBackNotices(t *testing.T) {
	notice := Event{"type": "system", "subtype": "permission_denied", "message": "Permission denied"}
	text := Event{"message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "between"}}}}

	folded := Trace([]Event{notice, notice, notice})
	if strings.Count(folded, "Permission denied") != 1 || !strings.Contains(folded, "2 more times") {
		t.Errorf("back-to-back notices not folded once with their count:\n%s", folded)
	}
	apart := Trace([]Event{notice, text, notice})
	if strings.Count(apart, "Permission denied") != 2 || strings.Contains(apart, "more times") {
		t.Errorf("notices apart were folded:\n%s", apart)
	}
	if !strings.Contains(apart, "[system:permission_denied]") {
		t.Errorf("a notice lost its kind:\n%s", apart)
	}
}

func TestTraceMarksSubagents(t *testing.T) {
	e := Event{"parent_tool_use_id": "t1", "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "hi"}}}}
	if got := Trace([]Event{e}); got != "[sub][text] hi" {
		t.Errorf("got %q", got)
	}
}

func TestClippedKeepsHeadAndTail(t *testing.T) {
	s := strings.Repeat("H", 100) + strings.Repeat("M", 1000) + strings.Repeat("T", 100)
	out := Clipped(s, 400)
	if !strings.HasPrefix(out, strings.Repeat("H", 100)) || !strings.HasSuffix(out, strings.Repeat("T", 100)) {
		t.Errorf("head or tail lost: %q...", out[:120])
	}
	if Clipped("short", 400) != "short" {
		t.Error("a short text was changed")
	}
}

func TestSplitLikeAShell(t *testing.T) {
	got, err := Split(`my-agent --skill {skill_dir} 'two words' "say \"hi\"" a\ b`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"my-agent", "--skill", "{skill_dir}", "two words", `say "hi"`, "a b"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, err := Split(`unfinished 'quote`); err == nil {
		t.Error("an unfinished quote was accepted")
	}
}

func TestRunEnds(t *testing.T) {
	ok := Run{ExitCode: 0, Events: []Event{{"type": "result", "result": "done", "total_cost_usd": 1.5}}}
	if !ok.Finished() || ok.Reason() != "" || ok.Report() != "done" || *ok.Cost() != 1.5 {
		t.Errorf("a finished run reads wrong: %+v", ok)
	}
	capped := Run{ExitCode: 1, Events: []Event{{"type": "result", "is_error": true, "subtype": "error_max_budget_usd"}}}
	if capped.Finished() || !strings.HasPrefix(capped.Reason(), "error_max_budget_usd exit 1") {
		t.Errorf("a run cut short by its budget reads as %q", capped.Reason())
	}
	if (Run{TimedOut: true, ExitCode: -1}).Reason() != "timeout" {
		t.Error("a timed-out run does not say so")
	}
}

func TestParseEventsSkipsACutOffLine(t *testing.T) {
	events := ParseEvents("{\"type\":\"system\"}\nnoise\n{\"type\":\"result\",\"res")
	if len(events) != 1 {
		t.Errorf("got %d events, want 1", len(events))
	}
}
