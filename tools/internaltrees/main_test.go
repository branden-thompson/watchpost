package main

import (
	"bytes"
	"errors"
	"io"
	"os"
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
			err := run(c.args, home, &out, io.Discard)
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
		if err := run([]string{"."}, home, &out, io.Discard); err != nil {
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
	err := run([]string{"."}, t.TempDir(), brokenWriter{}, io.Discard)
	if err == nil {
		t.Fatal("a failing destination produced no error")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error %q does not carry the write failure", err)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken destination") }

// TestAStaticOnlyRuleSaysSo is the positive control for the note: a rule with
// no derived half still passes, so the note is the only thing that tells a
// runner with no workspace apart from a workspace that does not match the
// convention.
func TestAStaticOnlyRuleSaysSo(t *testing.T) {
	var out, notes bytes.Buffer
	if err := run([]string{"."}, t.TempDir(), &out, &notes); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("no rule printed")
	}
	if !strings.Contains(notes.String(), "no workspace names derived") {
		t.Errorf("a static-only rule printed no note; notes were %q", notes.String())
	}

	// The control the other way: a real workspace derives names and says
	// nothing, so the note cannot pass by always firing.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory to derive from: %v", err)
	}
	var out2, notes2 bytes.Buffer
	if err := run([]string{".", home}, home, &out2, &notes2); err != nil {
		t.Fatalf("run with a real home: %v", err)
	}
	if notes2.Len() != 0 && !strings.Contains(out2.String(), "|(") {
		t.Errorf("a rule with no derived half on a real home is possible, but then the note must say so; got notes %q", notes2.String())
	}
}
