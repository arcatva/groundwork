package judge

import (
	"errors"
	"testing"
)

func TestCheckVerdicts(t *testing.T) {
	two := []Verdict{{1, true, "a"}, {2, false, "b"}}
	if err := CheckVerdicts(two, 2); err != nil {
		t.Errorf("a complete set was refused: %v", err)
	}
	for name, v := range map[string][]Verdict{
		"missing":      {{1, true, "a"}},
		"repeated":     {{1, true, "a"}, {1, false, "b"}},
		"out of range": {{1, true, "a"}, {3, true, "c"}},
	} {
		if err := CheckVerdicts(v, 2); !errors.Is(err, ErrShape) {
			t.Errorf("%s: got %v, want a shape error", name, err)
		}
	}
}

func TestDecodeFirst(t *testing.T) {
	var v []Verdict
	if err := decodeFirst("Sure:\n[{\"criterion\": 1, \"pass\": true, \"evidence\": \"x\"}]\nDone.", &v); err != nil || len(v) != 1 {
		t.Errorf("got %v, %v", v, err)
	}
	if err := decodeFirst("no json here", &v); !errors.Is(err, ErrShape) {
		t.Errorf("got %v, want a shape error", err)
	}
	var b struct {
		Pass bool `json:"pass"`
	}
	if err := decodeFirst(`{"pass": "yes"}`, &b); !errors.Is(err, ErrShape) {
		t.Errorf("a string where a boolean belongs was accepted: %v", err)
	}
}
