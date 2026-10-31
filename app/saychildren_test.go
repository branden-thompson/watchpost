package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// sweepProcs is a process table with one orphaned say of the user's own.
type sweepProcs struct{ ended []int }

func (s *sweepProcs) Inspect(pid int) (synth.Proc, bool) {
	return synth.Proc{UID: os.Getuid(), PPID: 1, Started: time.Unix(1_760_000_000, 0), Args: "say -o /tmp/watchpost-say-7/out.wav"}, pid == 4242
}
func (s *sweepProcs) End(pid int) error { s.ended = append(s.ended, pid); return nil }

// TestTheRunSweepsOrphansAndEndsItsChildren is FR-9.1 and FR-9.2 at the
// app: a run's start ends the orphan a killed watchpost recorded, records
// its own children in the same place, and its stop ends them.
func TestTheRunSweepsOrphansAndEndsItsChildren(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "say-4242"), []byte("4000 "+strconv.Itoa(1_760_000_000)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	procs := &sweepProcs{}
	stop, swept, _ := sayChildren(dir, procs)
	defer synth.SetRecordDir("")
	if swept != 1 || len(procs.ended) != 1 || procs.ended[0] != 4242 {
		t.Errorf("the start swept %d, ended %v; the one orphan", swept, procs.ended)
	}
	stop()
	if live := synth.LiveChildren(); len(live) != 0 {
		t.Errorf("after the stop, children still run: %v", live)
	}
}

// TestTheSweepIsToldToTheDiagnostics is the BUILD red team's I-3: what a
// start's sweep ended, and what it could not, is in the diagnostics; a sweep
// that found nothing says nothing.
func TestTheSweepIsToldToTheDiagnostics(t *testing.T) {
	if w := sweptWords(0, 0); w != "" {
		t.Errorf("an empty sweep says %q", w)
	}
	if w := sweptWords(2, 1); !strings.Contains(w, "2 speech process") || !strings.Contains(w, "1 could not be") {
		t.Errorf("a sweep of two ended and one not says %q", w)
	}
}
