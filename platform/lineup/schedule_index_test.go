package lineup

// schedule_index_test.go — a position past the end of a SHORT running order is
// the end, not a refusal.
//
// FOUND BY RED TEAM (round 2) AT BUILD EXIT, 0.16.0.  The Line-Up Request
// window defaults to slot 15 — the HUM LEAD's own ruling, "default to the bottom
// - position 15" — and `scheduleIndex` accepted a position only when it named a
// VISIBLE card or fell exactly at the end.  With a three-card running order, the
// normal case, `to=15` was refused: `Insert`'s invariant failed, `onRequested`
// returned no effects, nothing was queued — and `requestSchedule` had ALREADY
// closed the window on `valid()`.
//
// THE OPERATOR WAS SHOWN A SCHEDULED REQUEST THAT WAS NEVER TAKEN, which is
// FR-3.3's named trap, on this release's headline new control, at its own
// default.  The console compounds it: `broadcaster_lineup.go` deliberately draws
// slots 02..15 EMPTY because "a slot is an ADDRESS the operator can put
// something in" — every one of those addresses was refused.
//
// THIS FUNCTION'S OWN COMMENT ALREADY RULED IT: "A position PAST the last
// visible card means the end of the schedule."  The code did not do that.

import "testing"

func shortOrder(t *testing.T, n int) Lineup {
	t.Helper()
	var l Lineup
	for i := range n {
		id := string(rune('a' + i))
		c, err := Propose(Card{ID: id, Slot: LocationReport, Subject: id, Headline: id, State: Proposed})
		if err != nil {
			t.Fatal(err)
		}
		adm, err := c.To(Admitted)
		if err != nil {
			t.Fatal(err)
		}
		if l, err = l.Queue(MainTrack, adm); err != nil {
			t.Fatal(err)
		}
	}
	return l
}

// A REQUEST PAST THE END LANDS AT THE END.
//
// DRIVEN THROUGH `Insert`, NOT `scheduleIndex`. The first draft of this test
// asserted against `scheduleIndex` directly and pushed the clamp INTO it — which
// broke `Reorder`, where a slot the running order never drew is meaningless and
// must stay refused. One function, two questions: the clamp belongs to the
// caller that asks the question it answers.
func TestARequestPastTheEndLandsAtTheEnd(t *testing.T) {
	for _, to := range []int{3, 4, 15} {
		l := shortOrder(t, 3)
		out, err := l.Insert(MainTrack, requestCardFor(t, "req"), to)
		if err != nil {
			t.Errorf("position %d refused: %v — the window's own default is 15, and the console "+
				"draws every one of those slots as an address", to, err)
			continue
		}
		vis := out.visible(MainTrack)
		if vis != 4 {
			t.Errorf("position %d produced %d visible cards, want 4", to, vis)
		}
		last := out.tracks[MainTrack][len(out.tracks[MainTrack])-1]
		if last.ID != "req" {
			t.Errorf("position %d put the card at %q; past the end means the END", to, last.ID)
		}
	}
}

// AND A MOVE PAST THE RUNNING ORDER IS STILL REFUSED, which is the rule the
// first draft broke. Stated here beside its twin so the distinction is visible.
func TestAMovePastTheRunningOrderIsStillRefused(t *testing.T) {
	l := shortOrder(t, 3)
	if _, err := l.Reorder("a", 9); err == nil {
		t.Error("a MOVE to a slot the running order never drew must be refused; only a REQUEST clamps")
	}
}

func requestCardFor(t *testing.T, id string) Card {
	t.Helper()
	c, err := Propose(Card{ID: id, Slot: LocationReport, Subject: id, Headline: id,
		State: Proposed, Origin: FromOperator})
	if err != nil {
		t.Fatal(err)
	}
	adm, err := c.To(Admitted)
	if err != nil {
		t.Fatal(err)
	}
	return adm
}

// AND A POSITION THAT NAMES A VISIBLE CARD IS UNCHANGED — the fix must not turn
// every request into an append.
func TestAPositionThatNamesACardStillNamesIt(t *testing.T) {
	l := shortOrder(t, 3)
	for to := range 3 {
		idx, ok := l.scheduleIndex(MainTrack, to)
		if !ok {
			t.Fatalf("position %d refused; it names a visible card", to)
		}
		if got := l.tracks[MainTrack][idx].ID; got != string(rune('a'+to)) {
			t.Errorf("position %d resolved to %q, want %q", to, got, string(rune('a'+to)))
		}
	}
}

// AND THE REQUEST ACTUALLY LANDS, driven through the Director the way the
// operator's press does — the seam the console closes its window on.
func TestARequestAtTheWindowsDefaultSlotIsQueued(t *testing.T) {
	d := Director{lineup: shortOrder(t, 3)}
	before := d.lineup.visible(MainTrack)

	card, err := Propose(Card{ID: "req", Slot: LocationReport, Subject: "req",
		Headline: "Rainbow, CA", State: Proposed, Origin: FromOperator})
	if err != nil {
		t.Fatal(err)
	}
	adm, err := card.To(Admitted)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := d.onRequested(Requested{Card: adm, To: 15}) // the window's default
	if got := out.lineup.visible(MainTrack); got != before+1 {
		t.Fatalf("the request was dropped: %d visible cards before, %d after — the window had "+
			"already closed as though it were scheduled", before, got)
	}
}
