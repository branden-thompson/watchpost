package lineup

import (
	"testing"
	"time"
)

// P3: the Director proposes a card when a location needs a SYNTHESISED read.
//
// THE BED IS NOT A TRACK, and the merge must not flatten that. Tuning to a
// LIVE relay stays a tune — `Track`'s own comment says the bed is "a
// selectable resource the Director may cut over to, not a queue of cards".
// What becomes a card is the SYNTHESISED read, which is the thing the arbiter
// can own and the relay is not.

func TestASynthesisedReadBecomesAMainTrackCard(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})

	d, fx := d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	cards := d.lineup.Cards(MainTrack)
	if len(cards) != 1 {
		t.Fatalf("a location needing a read becomes ONE main-track card; got %d", len(cards))
	}
	if got := cards[0].Slot; got != LocationReport {
		t.Errorf("it is a location report; got slot %v", got)
	}
	if cards[0].Subject != "oceanside" {
		t.Errorf("the card must name the location it reads; got %q", cards[0].Subject)
	}
	// IT MUST BE BUILDABLE. A card with no headline is refused at proposal by
	// the model's own invariant, so a producer that omitted one would fail
	// silently and queue nothing.
	if cards[0].Headline == "" {
		t.Error("a card carries a headline from the moment it is proposed")
	}
	// THE STEP MUST ALSO SET THE CARD MOVING. Queueing it and stopping leaves a
	// card the schedule holds and nothing ever composes — the station shows a
	// lineup and plays silence. Found by a plant that survived: nothing here
	// asked for the effects, so skipping the settle changed no assertion.
	var built *BuildCard
	published := false
	for _, e := range fx {
		switch v := e.(type) {
		case BuildCard:
			b := v
			built = &b
		case Publish:
			published = true
		}
	}
	if built == nil {
		t.Fatal("queueing the read must also ask for its words; got no BuildCard")
	}
	if built.ID != ReadID("oceanside") {
		t.Errorf("the build is for THIS card; got %q", built.ID)
	}
	if built.Slot != LocationReport {
		t.Errorf("the executor is chosen by the slot on the effect (BD-8); got %v", built.Slot)
	}
	if !published {
		t.Error("the readers are told the schedule changed; got no Publish")
	}
}

func TestASecondNeedForTheSameLocationDoesNotQueueTwice(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	if got := len(d.lineup.Cards(MainTrack)); got != 1 {
		t.Errorf("the same location asked for twice must not be read twice — this is FR-2.5's "+
			"no-double-speak property at the SCHEDULE level, before any audio exists; got %d cards", got)
	}
}

// A STOPPED STATION PLANS, AND DOES NOT PERFORM (D-84, HUM LEAD 2026-09-11).
//
// THIS TEST USED TO ASSERT THE OPPOSITE — "a stopped station is not building a
// programme; got %d cards" — under DR-3's "admission is a promise to read". The
// ruling overturned it: "being able to see, manage, and change the line up PRIOR
// to going on air is a fundamental requirement — otherwise the user might as
// well just use Observer."
//
// WHAT IT PINS NOW IS THE HALF THAT DID NOT CHANGE. The card is accepted, and it
// does not take the air: `airOnce` is the one asker of `advances` left.
func TestANeedOnAStoppedStationIsQueuedButNotRead(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base) // STOPPED is the zero value
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	if got := len(d.lineup.Cards(MainTrack)); got != 1 {
		t.Errorf("the operator must be able to see and manage the line-up before going on air; got %d cards", got)
	}
	if c, on := d.lineup.OnAir(MainTrack); on {
		t.Errorf("a stopped station put %q on the air; planning is not performing", c.ID)
	}
}

// THE MECHANISM, NOT THE OUTCOME. The two tests above would both stay green if
// the refusal came from somewhere else — a remembered set of refs, a counter, a
// flag. What actually makes a second need harmless is that the id is a function
// of the ref and the lineup refuses two cards with one identity, so that is
// what gets pinned. Twice burned by asserting an outcome that occurs either
// way (P2d, P3c).
func TestARotationCardIsNamedAfterItsLocationAndNothingElse(t *testing.T) {
	// WRITTEN THROUGH A SLICE, NOT AS TWO IDENTICAL CALLS. `ReadID("x") !=
	// ReadID("x")` reads to a linter as identical expressions either side of a
	// comparison, which is a warning worth heeding rather than silencing: what
	// the rule is actually about is that the SAME INPUT gives the same answer,
	// so the input is a value that travels.
	refs := []string{"oceanside", "bonsall", "vista"}
	first := make([]string, len(refs))
	for i, r := range refs { // bounded by the slice (P10-02)
		first[i] = ReadID(r)
	}
	seen := map[string]string{}
	for i, r := range refs { // bounded by the slice (P10-02)
		// TWICE IN A ROW, and that adjacency is the whole test. Comparing only
		// across the two passes let a counter-based mutant through: with three
		// refs and a counter taken modulo three, every ref got the same suffix
		// on both passes and the id looked pure. Two adjacent calls differ
		// under ANY per-call variation, whatever its period. Found by the plant
		// that survived the first rewrite of this test.
		again, andAgain := ReadID(r), ReadID(r)
		if again != andAgain || again != first[i] {
			t.Errorf("%q named %q, then %q, then %q — the same location must name the same card, "+
				"every time; this IS the no-double-speak rule", r, first[i], again, andAgain)
		}
		if had, ok := seen[first[i]]; ok {
			t.Errorf("%q and %q name the same card %q; two locations are two cards", had, r, first[i])
		}
		seen[first[i]] = r
	}
	// A rotation read and a burst led by an alert of the same name are
	// different cards; sharing an id would wedge the schedule.
	if ReadID("x") == BurstID("x") {
		t.Error("a rotation card and a burst must not collide in one schedule")
	}
	if ReadID("") != "" {
		t.Error("a need with no location names no card")
	}
}

func TestTheLocationCanBeReadAgainOnceItsCardHasLeft(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	// The card is read and leaves the schedule. Reached through the lineup
	// rather than the air, because what is under test is the ROTATION coming
	// round again, not how a card ends.
	next, err := d.lineup.Remove(ReadID("oceanside"))
	if err != nil {
		t.Fatalf("removing the card the test just queued: %v", err)
	}
	d.lineup = next

	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	if got := len(d.lineup.Cards(MainTrack)); got != 1 {
		t.Errorf("a rotation comes ROUND: the location is read again once its card has left, "+
			"or the station reads each location once and then plays nothing; got %d cards", got)
	}
}

func TestARotationCardIsTheDirectorsOwn(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	cards := d.lineup.Cards(MainTrack)
	if len(cards) != 1 {
		t.Fatalf("want one card; got %d", len(cards))
	}
	// The rotation is the programme the DIRECTOR runs — the same origin the
	// stale-tune transition carries (stale.go). Observer proposes hazards it
	// fetched; the operator proposes what they placed by hand. The HUM LEAD's
	// ruling is that a live card's origin never changes, so getting it right at
	// proposal is the only chance.
	if got := cards[0].Origin; got != FromDirector {
		t.Errorf("a rotation read is the Director's own card; got origin %v", got)
	}
}

func TestANeedWithNoHeadlineQueuesNothing(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside"})
	// DR-7: a card is showable, loggable and countable from the moment it
	// exists. An unheadlined card would reach the Broadcaster as a blank slot.
	if got := len(d.lineup.Cards(MainTrack)); got != 0 {
		t.Errorf("a card carries a headline from proposal, or it is not a card; got %d", got)
	}
}
