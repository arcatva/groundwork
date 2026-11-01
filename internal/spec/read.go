package spec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func strict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// CheckReport reads a reviewer's report and holds it to its contract.
func CheckReport(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var r ReviewerReport
	if err := strict(b, &r); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return Validate(r)
}

// CheckLogFile reads a run log, one entry per line, and holds each entry to its contract and the
// whole log to the flow in ix.
func CheckLogFile(path string, ix Index) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var entries []LogEntry
	var errs []error
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e LogEntry
		if err := strict([]byte(line), &e); err != nil {
			errs = append(errs, fmt.Errorf("line %d: %w", n, err))
			continue
		}
		if err := Validate(e); err != nil {
			errs = append(errs, fmt.Errorf("line %d: %w", n, err))
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return errors.Join(append(errs, ix.CheckLog(entries)...)...)
}
