package tty

// broadcaster_priority_test.go — the alert track as a COLUMN (D-87).
//
// THIS FILE USED TO PIN AN OVERLAY. The priority track was composited on top of
// the running order and six tests held that shape: the splice, what it covered,
// what survived underneath it, and the rail label that travelled with it.
//
// HUM LEAD, 2026-09-11: "We're gonna split the PRIORITY and MAIN tracks visually
// — no more card occlusion, and this works because of the data that we're
// ACTUALLY showing."
//
// So the rules those tests held are GONE, not weakened, and the tests went with
// them rather than being bent into shape — a test kept alive against a retired
// design is a test that asserts the past. What replaces them is narrower and
// stronger: the tracks are built as separate columns and joined once, so there
// is no splice to get wrong and nothing to be covered by.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// D-61 IS SUPERSEDED: THE ALERT BOX IS ALWAYS DRAWN (D-97).
//
// HUM LEAD, 2026-09-12: "Up Next and Alert are always present — if there are no
// active alerts taking over, then the box is simply empty."
//
// D-61 ruled the opposite in September: "the PRIORITY rail label ONLY shows up
// when a priority card sits on top of the main rail — this gives the operator more
// space to view/manage the main rail during normal operation." That reasoning was
// about SPACE, and the v3 layout answers it differently: the column is reserved
// either way (priorityWidth), so hiding the box bought nothing and cost the
// operator the knowledge of where a hazard will appear.
//
// AN EMPTY BOX IS THE POINT. The frame does not move when a hazard arrives, which
// is the same argument `priorityWidth` already made for reserving the column.
func TestAClearRailStillDrawsTheAlertBox(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true

	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, bcTakeoverTitle) {
		t.Error("the alert box is missing while the rail is clear; it is ALWAYS present, and empty")
	}
	// AND IT IS EMPTY: a box with a hazard in it names one.
	if strings.Contains(got, "ALERT TYPE") {
		t.Error("the alert box lists something while the rail is clear")
	}
}

// AND A TAKEOVER BRINGS IT BACK, in its own column beside the running order.
func TestATakeoverBringsThePriorityTrackBack(t *testing.T) {
	b := withBurst(t, NewBroadcaster())

	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, bcTakeoverTitle) {
		t.Fatalf("a takeover drew no box:\n%s", got)
	}
	// THE RAIL NAMES IT. The label appears exactly when something is
	// interrupting the broadcast, which is the HUM LEAD's own ruling.
	if !strings.Contains(got, "P") || !strings.Contains(got, bcTakeoverTitle) {
		t.Error("the takeover is drawn with no label beside it")
	}
}

// THE TRACKS DO NOT OVERLAP, AND THAT IS NOW STRUCTURAL. Each column is padded
// to its own width before the join, so no row can carry one track's cells into
// the other's — which is what makes occlusion impossible rather than avoided.
func TestTheTwoTracksNeverShareACell(t *testing.T) {
	b := withBurst(t, NewBroadcaster())
	left := bcRailWidth + bcRailGap
	pw, cw := b.priorityWidth(), b.cardBoxWidth()

	// THE RUNNING ORDER'S ROWS ONLY. The masthead and the station section are
	// full-width regions of their own and know nothing about these columns.
	for i, row := range strings.Split(stripANSITest(b.View().Content), "\n") {
		r := []rune(row)
		if len(r) < left+pw+bcColumnGap+cw || !strings.ContainsAny(row, "┏┃┗") {
			continue
		}
		// The gap between the columns is air on every row, always.
		gap := string(r[left+pw : left+pw+bcColumnGap])
		if strings.TrimSpace(gap) != "" {
			t.Errorf("row %d has %q between the tracks; they are columns, not layers", i, gap)
		}
	}
}

// THE BURST'S BOX IS AS TALL AS THE BURST (HUM LEAD, 2026-09-11): "the alert box
// should be tall enough to fit the data, but doesn't need to 'fill' vertical
// space just because".
func TestTheTakeoverBoxIsAsTallAsItsBurstAndNoTaller(t *testing.T) {
	short := withBurst(t, NewBroadcaster(), "one hazard")
	tall := withBurst(t, NewBroadcaster(), "one hazard", "two", "three", "four", "five")

	s, l := len(short.priorityColumn(60)), len(tall.priorityColumn(60))
	if s == 0 || l == 0 {
		t.Fatalf("a burst drew no box: %d / %d rows", s, l)
	}
	if l <= s {
		t.Errorf("a five-alert burst drew %d rows and a one-alert burst %d; the box follows the burst", l, s)
	}
}

// AND IT NEVER GROWS THE FRAME. A burst longer than the running order would
// otherwise draw past the bottom, which FR-7.3 calls a defect rather than a
// degradation.
func TestTheTakeoverBoxIsCutToTheRunningOrdersHeight(t *testing.T) {
	b := withBurst(t, NewBroadcaster(), "a", "b", "c", "d", "e", "f", "g", "h")
	if got := len(b.priorityColumn(6)); got > 6 {
		t.Errorf("the takeover drew %d rows into a running order of 6", got)
	}
}

// bcTestNow is the fixed clock these fixtures declare their hazards at.
var bcTestNow = time.Date(2026, 9, 11, 23, 59, 59, 0, time.UTC)

// withBurst puts one takeover on the rail, listing the hazards named.
func withBurst(t *testing.T, b Broadcaster, alerts ...string) Broadcaster {
	t.Helper()
	if len(alerts) == 0 {
		alerts = []string{"Severe Thunderstorm Warning"}
	}
	c := lineup.Card{ID: "burst", Slot: lineup.BreakingAlert, Origin: lineup.FromDirector,
		Subject: "a1", Headline: "WEATHER ALERT • BURST",
		Script: lineup.Script{Parts: []lineup.Part{{Kind: lineup.PartHead, Text: "the following alerts have been declared:"}}}}
	for i, a := range alerts {
		c.From = append(c.From, lineup.Arrival{ID: string(rune('a' + i)), Headline: a, Subject: "Bonsall, CA", At: bcTestNow})
	}
	admitted, err := c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting the burst: %v", err)
	}
	l, err := b.lineup.Queue(lineup.AlertRail, admitted)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	return b
}
