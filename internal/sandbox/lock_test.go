package sandbox

import (
	"strings"
	"testing"
)

func TestASecondRunIsTurnedAway(t *testing.T) {
	dir := t.TempDir()
	first, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(dir); err == nil || !strings.Contains(err.Error(), "another harness run") {
		t.Errorf("a second run got the lock: %v", err)
	}
	first.Release()
	again, err := Acquire(dir)
	if err != nil {
		t.Errorf("the lock was not given back: %v", err)
	}
	again.Release()
}
