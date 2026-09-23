package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestRunRefusesWhatWouldPrintNothingUseful is the positive control for every
// guard in run: each case must FAIL, and the happy case must still print a
// rule. A guard that silently stopped matching would leave this test asserting
// nothing, so each case names the refusal it expects.
func TestRunRefusesWhatWouldPrintNothingUseful(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"a third argument is not ignored", []string{".", home, "extra"}, "at most two arguments"},
		{"an empty root is refused", []string{""}, "empty repository root"},
		{"an explicitly empty HOME is refused", []string{".", ""}, "empty HOME argument"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(c.args, home, &out)
			if err == nil {
				t.Fatalf("no error; wrote %q", out.String())
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not name %q", err, c.want)
			}
			if out.Len() != 0 {
				t.Fatalf("a refused run wrote to standard output: %q", out.String())
			}
		})
	}

	t.Run("a good run prints one rule", func(t *testing.T) {
		var out bytes.Buffer
		if err := run([]string{"."}, home, &out); err != nil {
			t.Fatalf("run: %v", err)
		}
		got := strings.TrimSuffix(out.String(), "\n")
		if got == "" {
			t.Fatal("printed an empty rule")
		}
		if strings.Contains(got, "\n") {
			t.Fatalf("printed %d lines, want one: %q", strings.Count(out.String(), "\n"), out.String())
		}
	})
}

// TestAFailedWriteIsReported is the positive control for the checked write: a
// destination that always fails must produce an error, not a silent exit zero
// that hands a gate half a rule.
func TestAFailedWriteIsReported(t *testing.T) {
	err := run([]string{"."}, t.TempDir(), brokenWriter{})
	if err == nil {
		t.Fatal("a failing destination produced no error")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error %q does not carry the write failure", err)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken destination") }
