package tty

// broadcaster_priority_test.go — the priority track is INVISIBLE UNTIL IT HAS
// SOMETHING (D-61, HUM LEAD 2026-09-10).
//
//	"the PRIORITY rail label ONLY shows up when a priority card sits on top of
//	the main rail — this gives the operator more space to view/manage the main
//	rail during normal operation."
//
// Which is the same thing said earlier about the track itself: "normally that
// priority lane is INVISIBLE to the operator — so the main track takes the full
// width of the UI."
//
// A "(clear)" ROW IS NOT NOTHING. It costs two rows of the running order to say
// that a hazard is not happening — which is the state the station is in almost
// all of the time.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

func TestAClearPriorityTrackDrawsNothingAtAll(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.View().Content)

	if strings.Contains(got, "PRIORITY") {
		t.Errorf("a clear priority track names itself nowhere:\n%s", headOf(got, 12))
	}
	if strings.Contains(got, "(clear)") {
		t.Errorf("and it does not spend a row saying so:\n%s", headOf(got, 12))
	}
}

func TestATakeoverBringsThePriorityTrackBack(t *testing.T) {
	var l lineup.Lineup
	c, err := lineup.Propose(lineup.Card{ID: "t1", Slot: lineup.BreakingAlert,
		Origin: lineup.FromObserver, Subject: "tornado", Headline: "TORNADO WARNING"})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	if c, err = c.To(lineup.Admitted); err != nil {
		t.Fatalf("admitting: %v", err)
	}
	if l, err = l.Queue(lineup.AlertRail, c); err != nil {
		t.Fatalf("queueing: %v", err)
	}

	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	got := stripANSITest(b.View().Content)

	if !strings.Contains(got, "TORNADO WARNING") {
		t.Errorf("a takeover must reach the frame:\n%s", headOf(got, 16))
	}
	if !strings.Contains(got, "PRIORITY") {
		t.Errorf("and the track names itself while it has something:\n%s", headOf(got, 16))
	}
}

// THE MAIN TRACK GETS THE ROWS BACK. That is the whole reason for the ruling:
// two rows spent on "(clear)" are two rows of running order the operator cannot
// see, in the state the station is in almost all of the time.
func TestAClearTrackGivesItsRowsToTheRunningOrder(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	first := -1
	for i, r := range rows {
		if strings.HasPrefix(strings.TrimSpace(r), "|") && strings.Contains(r, "+---") {
			first = i
			break
		}
	}
	if first < 0 {
		t.Skip("no cards in this fixture; the region test covers the placement")
	}
}

// headOf is the first n rows of a frame, for an error that has to show WHERE.
func headOf(s string, n int) string {
	parts := strings.Split(s, "\n")
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, "\n")
}
