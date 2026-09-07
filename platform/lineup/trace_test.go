package lineup

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// DR-23 — EVERY EVENT THE DIRECTOR ACCEPTS HAS A LINE.
//
// The requirement is a timeline that reads as ONE timeline, and the way that
// fails is not a missing feature but a missing case: an event added later with
// no line here is silent in the log, and the gap looks like nothing happening
// rather than like a gap. That is the same failure Describe's own closed-set
// test exists to prevent, in the other direction.
func TestDR23EveryEventDescribesItself(t *testing.T) {
	for _, ev := range []Event{
		Arrived{Arrivals: many("a", category.Warnings, 2)},
		Tick{Now: planNow},
		Built{ID: "a00", Script: Say("words")},
		Finished{ID: "a00"},
		Failed{ID: "a00", Reason: "no voice"},
		Powered{To: Running},
		Powered{To: Stopped},
		Tuned{Ref: "oceanside-ca", Live: true},
		Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute},
		Ended{},
	} {
		got := DescribeEvent(ev)
		if strings.TrimSpace(got) == "" {
			t.Errorf("%T has no line in the timeline: an event nobody can see is a gap that reads as quiet", ev)
		}
	}
	if got := DescribeEvent(nil); got != "" {
		t.Errorf("a nil event describes nothing, got %q", got)
	}
}

// The line carries what the reader needs to follow it: which card, which
// station, how many. A timeline of bare type names says only that something
// happened.
func TestDR23TheLineNamesItsSubject(t *testing.T) {
	for _, tc := range []struct {
		ev   Event
		want string
	}{
		{Built{ID: "a00", Script: Say("w")}, "a00"},
		{Finished{ID: "a01"}, "a01"},
		{Failed{ID: "a02", Reason: "no voice could read it"}, "no voice"},
		{Tuned{Ref: "oceanside-ca"}, "oceanside-ca"},
		{Arrived{Arrivals: many("a", category.Warnings, 3)}, "3"},
		{Programme{Watchlist: []string{"a", "b"}, Dwell: 30 * time.Second}, "30s"},
		{Powered{To: Stopped}, "STOPPED"},
	} {
		if got := DescribeEvent(tc.ev); !strings.Contains(got, tc.want) {
			t.Errorf("%T reads %q, which does not name %q", tc.ev, got, tc.want)
		}
	}
}

// THE SCHEDULE'S OWN LINE CANNOT DISAGREE WITH THE SCHEDULE.
//
// DR-23 asks for a line per card state transition. The transitions happen in a
// pure function that may not write, so the line is taken from the schedule the
// step produced instead — which is strictly better: a hand-written transition
// line can claim a state the lineup does not hold, and this cannot.
func TestDR23TheTraceShowsEveryCardAndItsState(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	if got := d.Lineup().Trace(); got != "empty" {
		t.Errorf("an empty schedule says so, got %q", got)
	}

	// TWO BURSTS, so the trace has two cards to show and can be caught showing
	// only one: a burst is one card (MVS-D-77).
	d, _ = run(d, Powered{To: Running},
		Arrived{Arrivals: many("a", category.Warnings, 2)},
		Arrived{Arrivals: many("b", category.Warnings, 2)})
	d, _ = run(d, Built{ID: BurstID("a00"), Script: Say("words")})

	got := d.Lineup().Trace()
	for _, want := range []string{"ALERT RAIL", BurstID("a00"), "ON AIR", BurstID("b00")} {
		if !strings.Contains(got, want) {
			t.Errorf("the trace must show %q; it reads %q", want, got)
		}
	}
	// It agrees with the lineup because it is READ from it: the card the
	// schedule says is on the air is the one the line says is on the air.
	onAir, _ := d.Lineup().OnAir()
	if !strings.Contains(got, onAir.ID+":ON AIR") {
		t.Errorf("the trace and the schedule disagree about who is reading: %q vs %q", got, onAir.ID)
	}
}
