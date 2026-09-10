package lineup

import (
	"testing"
	"time"
)

// projection_test.go — the LINE-UP the operator sees, versus the SCHEDULE the
// roles work from (HUM LEAD, 2026-09-05; restated 2026-09-10).
//
//	"The operator's view is a PROJECTION of the one Lineup, not a second
//	schedule."
//
// It was the identity function until now: one card per burst, nothing invisible,
// so `Cards` was already what the operator saw. The Director's structural cards
// are the first thing that makes it a real function — and one of them is ALREADY
// IN PRODUCTION. director.go queues the staleness notice onto the main track, so
// a card the operator never asked for is sitting in their numbered running order
// today.

// withNotice is a schedule holding a report, then the Director's own transition,
// then a second report — the shape the staleness drop actually produces.
func withNotice(t *testing.T) Director {
	t.Helper()
	d := New(Settings{Max: 5, Depth: 0}, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	for _, c := range []Card{
		{ID: "r0", Slot: LocationReport, Origin: FromDirector, Subject: "oceanside", Headline: "OCEANSIDE, CA"},
		{ID: "t0", Slot: Transition, Origin: FromDirector, Subject: "stale read", Headline: "Report out of date", Script: Say("That report is out of date.")},
		{ID: "r1", Slot: LocationReport, Origin: FromDirector, Subject: "carlsbad", Headline: "CARLSBAD, CA"},
	} {
		proposed, err := Propose(c)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
		admitted, err := proposed.To(Admitted)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
		d.lineup, err = d.lineup.Queue(MainTrack, admitted)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
	}
	return d
}

func TestTheProjectionHidesTheDirectorsStructuralCards(t *testing.T) {
	d := withNotice(t)

	if got := ids(d.lineup.Cards(MainTrack)); !equal(got, []string{"r0", "t0", "r1"}) {
		t.Fatalf("the SCHEDULE holds everything the roles read; got %v", got)
	}
	if got := ids(d.lineup.Projection(MainTrack)); !equal(got, []string{"r0", "r1"}) {
		t.Errorf("the LINE-UP is what the operator sees — no structural cards; got %v", got)
	}
}

func TestTheProjectionIsTheIdentityWhenNothingIsStructural(t *testing.T) {
	// The rule that made this invisible for two releases, stated so it stays
	// true: with nothing structural in the schedule the two are the same list,
	// which is why no code was needed until now.
	d := New(Settings{Max: 5}, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	if got, want := ids(d.lineup.Projection(MainTrack)), ids(d.lineup.Cards(MainTrack)); !equal(got, want) {
		t.Errorf("projection %v and schedule %v must agree when nothing is structural", got, want)
	}
}

func TestTheOperatorCannotAddressAStructuralCard(t *testing.T) {
	// The operator cannot see it, so they cannot have meant it. A move or a drop
	// that named one would be the console addressing a slot it never drew.
	d := withNotice(t)

	if _, err := d.lineup.Reorder("t0", 0); err == nil {
		t.Error("a structural card is not in the running order the operator addresses")
	}

	d, _ = d.Step(Dropped{ID: "t0"})
	if got := ids(d.lineup.Cards(MainTrack)); !equal(got, []string{"r0", "t0", "r1"}) {
		t.Errorf("and it cannot be dropped; schedule is now %v", got)
	}
	if n := len(d.lineup.Discarded()); n != 0 {
		t.Errorf("a structural card never reaches the operator's pile; it holds %d", n)
	}
}

func TestAMoveIsInProjectionSpaceNotScheduleSpace(t *testing.T) {
	// THE OFF-BY-N THIS EXISTS TO PREVENT. `Moved`'s own comment says "the
	// console already knows every slot number it drew" — those are PROJECTION
	// numbers. Indexing the schedule with one lands the card somewhere else,
	// which is FR-3.3's shape exactly: an action shown as taken that the
	// schedule took differently.
	//
	// r1 is at projection index 1 and SCHEDULE index 2. Moving it to projection
	// index 0 must put it first in the running order.
	d := withNotice(t)

	next, err := d.lineup.Reorder("r1", 0)
	if err != nil {
		t.Fatalf("moving a card the operator can see: %v", err)
	}

	if got := ids(next.Projection(MainTrack)); !equal(got, []string{"r1", "r0"}) {
		t.Errorf("the operator moved the card to slot 0 of what they SEE; got %v", got)
	}
	if n := len(next.Cards(MainTrack)); n != 3 {
		t.Errorf("and the schedule keeps every card it held; got %d", n)
	}
}

func TestAMoveToTheLastProjectionSlotIsAccepted(t *testing.T) {
	d := withNotice(t)

	next, err := d.lineup.Reorder("r0", 1)
	if err != nil {
		t.Fatalf("moving to the last slot the operator can see: %v", err)
	}
	if got := ids(next.Projection(MainTrack)); !equal(got, []string{"r1", "r0"}) {
		t.Errorf("got %v", got)
	}
}

func TestAMoveBeyondTheProjectionIsRefused(t *testing.T) {
	// The bound is what the OPERATOR can see, not what the schedule holds. With
	// the notice in it the schedule has three cards and the running order has
	// two, so slot 2 is a slot that was never drawn.
	d := withNotice(t)

	if _, err := d.lineup.Reorder("r0", 2); err == nil {
		t.Error("a card moves to a position the RUNNING ORDER has, not one the schedule has")
	}
}

func TestTheTopOffCountsTheRunningOrderNotTheSchedule(t *testing.T) {
	// A depth of ten against a schedule counting structural cards tops off to
	// about five real reports — the console draws ten slots and half of them
	// stay empty for ever.
	d := withNotice(t)
	d.settings.Depth = 3

	d, _ = d.Step(offers("encinitas", "vista"))

	if got := len(d.lineup.Projection(MainTrack)); got != 3 {
		t.Errorf("the depth is a depth of the RUNNING ORDER; it holds %d", got)
	}
}
