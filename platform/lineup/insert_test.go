package lineup

import (
	"fmt"
	"testing"
)

// reportCard is one location report the lineup will hold. `admittedCard` in
// director_test.go takes a whole Card; this names the one shape every test here
// wants.
func reportCard(t *testing.T, id string) Card {
	t.Helper()
	return admittedCard(t, Card{ID: id, Slot: LocationReport, Subject: id, Headline: id})
}

// runningOrder is the running order's ids, in order. `order` in operator_test.go asks
// the same of a DIRECTOR; these tests drive the Lineup itself.
func runningOrder(l Lineup) []string {
	var out []string
	for _, c := range l.Projection(MainTrack) { // bounded by the track (P10-02)
		out = append(out, c.ID)
	}
	return out
}

func seeded(t *testing.T, n int) Lineup {
	t.Helper()
	var l Lineup
	for i := range n { // bounded by the ask (P10-02)
		next, err := l.Queue(MainTrack, reportCard(t, fmt.Sprintf("c%02d", i)))
		if err != nil {
			t.Fatalf("seeding %d: %v", i, err)
		}
		l = next
	}
	return l
}

// TestInsertPushesTheRestDown.
//
// HUM LEAD, 2026-09-14: "Line-Up Slot: ___ / Cards from this position will be
// pushed down by 1."
func TestInsertPushesTheRestDown(t *testing.T) {
	l := seeded(t, 4)
	out, err := l.Insert(MainTrack, reportCard(t, "new"), 2)
	if err != nil {
		t.Fatalf("inserting: %v", err)
	}
	want := []string{"c00", "c01", "new", "c02", "c03"}
	got := runningOrder(out)
	if len(got) != len(want) {
		t.Fatalf("the running order holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the running order holds %v, want %v", got, want)
		}
	}
}

// AND PRIORITIZE IS THE SAME ACT AT ZERO (HUM LEAD): "PRIORITIZE - Move to 'UP
// NEXT' / Pushes all existing line-up cards down by 1."
func TestPrioritizeGoesToTheFrontAndPushesEverythingDown(t *testing.T) {
	l := seeded(t, 3)
	out, err := l.Insert(MainTrack, reportCard(t, "urgent"), 0)
	if err != nil {
		t.Fatalf("prioritizing: %v", err)
	}
	if got := runningOrder(out); got[0] != "urgent" {
		t.Errorf("PRIORITIZE put the card at %q; it goes to the front", got[0])
	}
	if got := len(runningOrder(out)); got != 4 {
		t.Errorf("the running order holds %d after a prioritize of 3, want 4", got)
	}
}

// TestTheLastCardFallsIntoTheDiscardPile.
//
// HUM LEAD, 2026-09-14: "Last Card fall off, can go into our discard pile and the
// producer can re-request a copy of that card if needed."
//
// A FULL TRACK IS NOT A REASON TO REFUSE THE OPERATOR (ruling 6): the Director
// executes their will, and what it costs is said rather than the act being
// declined.
func TestTheLastCardFallsIntoTheDiscardPile(t *testing.T) {
	full := seeded(t, MainTrackCap)
	if got := len(runningOrder(full)); got != MainTrackCap {
		t.Fatalf("the fixture holds %d, want a full track of %d", got, MainTrackCap)
	}
	lastBefore := runningOrder(full)[MainTrackCap-1]

	out, err := full.Insert(MainTrack, reportCard(t, "new"), 0)
	if err != nil {
		t.Fatalf("inserting into a full track: %v", err)
	}
	if got := len(runningOrder(out)); got != MainTrackCap {
		t.Errorf("the running order holds %d after an insert into a full track, want %d",
			got, MainTrackCap)
	}
	if got := runningOrder(out)[0]; got != "new" {
		t.Errorf("the requested card is at %q, want the front", got)
	}
	// THE ONE THAT FELL IS RECOVERABLE, which is the whole of the ruling: the
	// Producer can re-request it, and that is only true if it is on the pile.
	if _, _, ok := out.takeDiscarded(lastBefore); !ok {
		t.Errorf("%q fell off the bottom and is not on the discard pile; it is simply gone", lastBefore)
	}
}

// AND THE CARD THAT FALLS IS THE LAST ONE, not the one that happens to sit at
// the end of the SCHEDULE. The Director's own structural cards are not in the
// running order and must not be pushed out of it.
func TestAStructuralCardIsNeverTheOneThatFallsOff(t *testing.T) {
	l := seeded(t, MainTrackCap)
	// A structural card at the very end of the schedule, which the operator
	// cannot see and did not ask about.
	// ITS WORDS ARE FIXED AT PROPOSAL — a structural card never exists without
	// them, which is the invariant a bare one trips.
	sta := admittedCard(t, Card{ID: "transition", Slot: Transition, Headline: "coming up",
		Script: Say("Coming up on Watchpost.")})
	l, err := l.Queue(MainTrack, sta)
	if err != nil {
		t.Fatalf("queueing the structural card: %v", err)
	}

	out, err := l.Insert(MainTrack, reportCard(t, "new"), 0)
	if err != nil {
		t.Fatalf("inserting: %v", err)
	}
	if _, _, held := out.find("transition"); !held {
		t.Error("the structural card was pushed out of a running order it was never in")
	}
	// AND EXACTLY ONE CARD WAS SHED, which is what the count being over the
	// RUNNING ORDER rather than the whole track actually buys.
	//
	// THE STRUCTURAL CHECK ABOVE DID NOT CATCH THIS. Counting every card sheds
	// one EXTRA — `lastVisible` never picks a structural card, so the transition
	// survives either way and a second report falls off to make room for it.
	// Mutant mAY3 survived on exactly that gap: I asserted the structural card
	// was safe and never that the reports were.
	if got := len(runningOrder(out)); got != MainTrackCap {
		t.Errorf("the running order holds %d reports after an insert, want %d — "+
			"a structural card is costing a report its slot", got, MainTrackCap)
	}
}

// AND ASKING TWICE FOR THE SAME REPORT IS REFUSED, NOT DUPLICATED. An operator
// can reasonably press the key twice; two cards under one identity is a schedule
// that cannot say which one it is reading.
func TestTheSameCardCannotBeRequestedTwice(t *testing.T) {
	l := seeded(t, 2)
	c := reportCard(t, "dup")
	l, err := l.Insert(MainTrack, c, 0)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := l.Insert(MainTrack, c, 0); err == nil {
		t.Error("the same card was inserted twice; the schedule now holds two under one identity")
	}
}
