package lineup

// THE DISCARD PILE (D-35, HUM LEAD 2026-09-09).
//
// > "refused cards can pile up if we're not careful, since technically once
// > it's refused, it's in a final state that is temporally unbounded. We can
// > create a Discard pile … the discard pile needs to act as a stack with a
// > trapdoor — we cap the number of cards the discard pile can hold, and the
// > oldest (at the bottom) fall through the trapdoor when another discarded card
// > is set on top … perhaps 5."
//
// IT IS NOT A TRACK, AND THE REASON IS MEASURED. `stopped()` is
// `lineup.held() == 0` and `held()` counts every card on every track, so a
// discarded card parked in one would mean the schedule NEVER READS AS STOPPED
// and the relay-fault window could never fire. The cap is what stops the pile
// becoming a track by accident.

import (
	"testing"
	"time"
)

func discardBase() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }

// aCard is a well-formed admitted card, by name.
func aCard(t *testing.T, id string) Card {
	t.Helper()
	c, err := Propose(Card{ID: id, Slot: LocationReport, Origin: FromDirector,
		Subject: id, Headline: "REPORT FOR " + id})
	if err != nil {
		t.Fatalf("proposing %s: %v", id, err)
	}
	c, err = c.To(Admitted)
	if err != nil {
		t.Fatalf("admitting %s: %v", id, err)
	}
	return c
}

func TestADiscardedCardGoesOnThePileNewestFirst(t *testing.T) {
	var l Lineup
	l = l.discard(aCard(t, "one"))
	l = l.discard(aCard(t, "two"))

	got := l.Discarded()
	if len(got) != 2 {
		t.Fatalf("both are on the pile; got %d", len(got))
	}
	if got[0].ID != "two" {
		t.Errorf("the newest is on TOP — an operator undoing reaches for what they just dropped; got %q", got[0].ID)
	}
}

// THE TRAPDOOR. The pile is bounded because a final state is temporally
// unbounded: without a cap, a 24/7 station accumulates every card it ever
// dropped.
func TestTheOldestFallsThroughTheTrapdoor(t *testing.T) {
	var l Lineup
	for i := range discardDepth + 3 {
		l = l.discard(aCard(t, string(rune('a'+i))))
	}

	got := l.Discarded()
	if len(got) != discardDepth {
		t.Fatalf("the pile holds at most %d; got %d", discardDepth, len(got))
	}
	if got[0].ID != string(rune('a'+discardDepth+2)) {
		t.Errorf("the newest is still on top after the trapdoor opened; got %q", got[0].ID)
	}
	for _, c := range got {
		if c.ID == "a" {
			t.Error("the oldest fell through — it must not still be here")
		}
	}
}

// THE PILE IS NOT A TRACK, and this is the assertion that keeps it that way.
func TestThePileDoesNotCountAsSchedule(t *testing.T) {
	d := New(Settings{Max: 5}, discardBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d.lineup = d.lineup.discard(aCard(t, "dropped"))

	if got := d.lineup.held(); got != 0 {
		t.Errorf("a discarded card is not held: held() counts what the schedule owes a read, and a "+
			"pile that counted would mean the station never reads as stopped; got %d", got)
	}
	if !d.stopped() {
		t.Error("so a schedule holding nothing but a discard pile IS stopped, and the fault window can " +
			"still fire — which is the whole reason the pile is not a track")
	}
}

// A CLONE KEEPS THE PILE. Every mutator starts at clone, so a short copy would
// lose the operator's undo silently — the same defect the track invariant
// exists for, one field along.
func TestACloneKeepsThePile(t *testing.T) {
	var l Lineup
	l = l.discard(aCard(t, "kept"))

	next, err := l.Queue(MainTrack, aCard(t, "queued"))
	if err != nil {
		t.Fatalf("queueing: %v", err)
	}

	if got := len(next.Discarded()); got != 1 {
		t.Errorf("queueing a card must not lose the pile; got %d discarded", got)
	}
}

// THE CONSOLE IS TOLD, because Publish carries the lineup by value and the undo
// control has no other source (FR-3.7's restore).
func TestThePublishedLineupCarriesThePile(t *testing.T) {
	d := New(Settings{Max: 5}, discardBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d.lineup = d.lineup.discard(aCard(t, "dropped"))

	_, fx := d.Step(Tick{Now: discardBase().Add(time.Second)})

	for _, f := range fx {
		if p, ok := f.(Publish); ok {
			if got := len(p.Lineup.Discarded()); got != 1 {
				t.Errorf("the console cannot offer an undo for a pile it was never told about; got %d", got)
			}
			return
		}
	}
	t.Fatal("no Publish came out of the step")
}

// THE PILE HAS A PRODUCTION WRITER TODAY, and it is the staleness drop.
//
// PD-3 drops a card whose data has aged past the window, and the reason is
// sharper than "the observation is old": the window exists to stop the station
// asserting something UNTRUE. That is a deliberate removal, so it belongs on the
// operator's pile — before this, a dropped card simply vanished and nothing
// could say what had been taken away or why.
func TestAStaleDropLandsOnThePile(t *testing.T) {
	// THE REAL PATH, driven rather than shortcut. Staleness is "a question about
	// a card that is about to be READ" (PD-3), so it fires only when the air
	// frees and the next card has been standing by too long — which means the
	// fixture needs a hazard holding the air while the report ages behind it.
	d := New(Settings{Max: 5}, discardBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})
	d, _ = d.Step(Built{ID: BurstID("a1"), Script: Say("A tornado warning has been declared.")})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	d, _ = d.Step(Built{ID: ReadID("oceanside"), Script: Say("Currently sixty-one degrees.")})
	if len(d.lineup.Cards(MainTrack)) != 1 {
		t.Fatalf("the fixture needs the report standing by behind the hazard; got %d", len(d.lineup.Cards(MainTrack)))
	}

	// The report ages past the window while the hazard reads, so speaking it
	// would assert something untrue.
	d, _ = d.Step(Tick{Now: discardBase().Add(StaleAfter + time.Minute)})
	d, _ = d.Step(Finished{ID: BurstID("a1")})

	got := d.lineup.Discarded()
	if len(got) != 1 {
		t.Fatalf("the dropped card goes where the operator can see what was taken away; got %d", len(got))
	}
	if got[0].ID != ReadID("oceanside") {
		t.Errorf("and it is the card that was dropped; got %q", got[0].ID)
	}
	if got[0].State != Discarded {
		t.Errorf("carrying the state it left in, so the pile says what happened to it; got %v", got[0].State)
	}
}

// A FAILED CARD IS NOT A DROPPED CARD, and this is the line between them.
//
// A routed decline is the schedule routing AROUND a fault (DR-21) — the
// producer offers the alert again, nobody chose anything. Putting those on the
// undo pile would fill it with things the operator never did, one per rotation
// turn on some paths, and an undo buffer full of noise is one nobody reaches
// for under pressure.
func TestAFailedCardDoesNotGoOnThePile(t *testing.T) {
	d := New(Settings{Max: 5}, discardBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	d, _ = d.Step(Failed{ID: ReadID("oceanside"), Reason: "the report composed nothing to say", Routed: true})

	if got := len(d.lineup.Discarded()); got != 0 {
		t.Errorf("a card the schedule routed around is not one the operator dropped; got %d on the pile", got)
	}
}
