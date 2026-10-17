package synth

import (
	"testing"
	"time"
)

// TestPSIsReadWhole is FR-9.2's reading of the process table: ps's uid,
// parent, start time and arguments, and a line it cannot read is no process.
func TestPSIsReadWhole(t *testing.T) {
	p, ok := parsePS("  501     1 Thu Oct 15 09:00:01 2026     say -o /tmp/watchpost-say-1/out.wav -f /tmp/watchpost-say-1/in.txt")
	want := time.Date(2026, 10, 15, 9, 0, 1, 0, time.Local)
	if !ok || p.UID != 501 || p.PPID != 1 || !p.Started.Equal(want) || p.Args != "say -o /tmp/watchpost-say-1/out.wav -f /tmp/watchpost-say-1/in.txt" {
		t.Errorf("read %+v (%v)", p, ok)
	}
	for _, bad := range []string{"", "501 1", "x 1 Thu Oct 15 09:00:01 2026 say", "501 1 not a date at all here say"} {
		if _, ok := parsePS(bad); ok {
			t.Errorf("%q read as a process", bad)
		}
	}
}
