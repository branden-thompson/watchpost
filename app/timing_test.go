package app

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// TestTheTimingLogIsOnlyThereWhenAskedFor is W14's instrument (D-154): under
// WATCHPOST_DEBUG_TIMING=1 the dashboard is handed a keeper; otherwise it is
// handed nothing, and nothing is measured.
func TestTheTimingLogIsOnlyThereWhenAskedFor(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_TIMING", "")
	if l := newTimingLog(); l != nil || l.hook() != nil || l.last() != nil {
		t.Error("with the switch off there must be no keeper and no hook")
	}
	t.Setenv("WATCHPOST_DEBUG_TIMING", "1")
	if l := newTimingLog(); l == nil || l.hook() == nil {
		t.Error("with the switch on the dashboard must be handed a keeper")
	}
}

// TestTheTimingsReachTheCounters: each interval kept in order with a rising
// sequence number - so a reader polling /debug/counters loses and repeats
// nothing - the newest timingsKept of them, in the counters record.
func TestTheTimingsReachTheCounters(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_TIMING", "1")
	l := newTimingLog()
	for i := range timingsKept + 3 {
		l.hook()(tty.Timing{Trigger: "open", Event: "settled", After: time.Duration(i) * time.Millisecond})
	}
	got := l.last()
	if len(got) != timingsKept || got[0].Seq != 4 || got[len(got)-1].Seq != timingsKept+3 {
		t.Fatalf("kept %d, seq %d..%d; want the newest %d, 4..%d", len(got), got[0].Seq, got[len(got)-1].Seq, timingsKept, timingsKept+3)
	}
	if got[len(got)-1].At.IsZero() {
		t.Error("each interval says when it was measured, so a reader can place it in a phase")
	}
	if got[len(got)-1].MS != float64(timingsKept+2) || got[0].Trigger != "open" || got[0].Event != "settled" {
		t.Errorf("a record says %+v", got[len(got)-1])
	}
	root := t.TempDir()
	d := testDumper(t, root, time.Now())
	src := d.sources
	d.sources = func() diagSources { s := src(); s.timings = l; return s }
	if rec := d.record(time.Now()); len(rec.Timings) != timingsKept {
		t.Errorf("the counters record carries %d timings, want %d", len(rec.Timings), timingsKept)
	}
}
