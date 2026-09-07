package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// I-7 — AN ALERT IS NEVER READ TWICE.
//
// A burst's card ID is its LEAD's, so two bursts with different leads can carry
// the same alert; and the seen store is written line by line AS EACH IS SAID,
// so an alert queued but not yet spoken is invisible both to the producer's
// unread filter and to the Director's duplicate-ID check. The overlapping
// alerts were then read back to back, and an alert that repeats itself is an
// alert a listener stops trusting.
//
// PINNED AT THE BUILD, WHICH IS WHERE THE RACE LANDS. The producer's filter
// runs when the burst is OFFERED; the gap is between that and the compose, so a
// test that offers a second burst cannot reach it — unread has already dropped
// the overlap by then. What is reachable, and what Broadcaster makes routine
// (OffAir holds the rail while nothing is marked, so every cycle queues another
// overlapping burst), is a card reaching the composer carrying a ref the store
// has meanwhile learnt about. That is exactly this.
func TestACardCarryingAnAlreadyReadAlertIsDeclined(t *testing.T) {
	now := time.Now()
	shared := globalfeed.Event{
		ID: "shared", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning",
		Location: "Cherry, NE", Severity: globalfeed.SevOrange, At: now, Until: now.Add(time.Hour),
	}
	lead := globalfeed.Event{
		ID: "lead-b", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "Raleigh, NC", Severity: globalfeed.SevRed, At: now, Until: now.Add(time.Hour),
	}

	build := func(t *testing.T, alreadyRead bool) []lineup.Event {
		t.Helper()
		nar := testDirector(&scriptVoice{}, nil)
		nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
		deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
		deck.alerts = newAlertStore()
		deck.alerts.note([]globalfeed.Event{lead, shared}, now)
		if alreadyRead {
			// The card on air marked it as its line was said.
			deck.seen.mark([]globalfeed.Event{shared}, now)
		}
		st := newStation(t, deck)
		return st.x.run(context.Background(), lineup.BuildCard{
			ID: "burst:lead-b", Slot: lineup.BreakingAlert, Subject: "Tornado Warning",
			Refs: []string{lead.ID, shared.ID},
		})
	}

	// THE CONTROL FIRST, or "declined" below proves only that the build is
	// broken: the same card, with nothing yet read, must compose.
	var built bool
	for _, ev := range build(t, false) {
		if _, ok := ev.(lineup.Built); ok {
			built = true
		}
	}
	if !built {
		t.Fatal("the card does not compose even with nothing read; this test would pass for the wrong reason")
	}

	for _, ev := range build(t, true) {
		if _, ok := ev.(lineup.Built); ok {
			t.Error("a card carrying an alert another card has already read was composed; " +
				"the listener hears that alert a second time, back to back")
		}
	}
}
