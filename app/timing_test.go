package app

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
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

// TestTheFeedTimesItsStages is W14's next measure: the alerts' answer still
// takes ~5.3 s cold after the inputs were asked together, and which stage
// holds it was being guessed. With the instrument on, a feed ask says how
// long each stage took - each input, the zones, the overlays - so the next
// change is aimed at a measured stage.
func TestTheFeedTimesItsStages(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_TIMING", "1")
	loc, srv := m1Fixture(t, "01-covers-oak-ridge")
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL), timings: newTimingLog()}
	lp.mapFeed(context.Background(), tty.MapAsk{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{loc}}, Place: &loc})
	got := map[string]bool{}
	for _, r := range lp.timings.last() {
		if r.Trigger == "feed" {
			got[r.Event] = true
		}
	}
	for _, want := range []string{"inputs", "input:alerts", "zones", "overlays", "whole"} {
		if !got[want] {
			t.Errorf("the feed did not time %q: %v", want, got)
		}
	}
	quiet := &livePipelines{zoneShapes: zoneStore(t, srv.URL)} // the instrument off: nothing kept, nothing asked of it
	quiet.mapFeed(context.Background(), tty.MapAsk{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{loc}}, Place: &loc})
	if quiet.timings.last() != nil {
		t.Error("with the instrument off the feed kept timings")
	}
}
