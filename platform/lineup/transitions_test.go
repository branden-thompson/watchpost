package lineup

import (
	"strings"
	"testing"
	"time"
)

// transitions_test.go — D-43: transitions belong to the JOIN, not to the card.
//
// THE HUM LEAD'S OWN SENTENCE IS THE JOIN MODEL: "the Director will re-evaluate
// if transitions are needed for that card AT THAT TIME, and the answer MAY BE
// DIFFERENT depending on what the line looks like at that time."  That is as
// true of a drop as of a resurrection — dropping the card between A and C
// changes what A→C needs.
//
// AND IT IS FREE, which is what makes re-deriving on every change the cheap
// option rather than the expensive one: `slots()` gives Transition
// textAtStandby:false, so a transition's words are FIXED AT PROPOSAL.  No
// composer, no network, nothing to wait for.

func seed(t *testing.T, cards ...Card) Director {
	t.Helper()
	d := New(Settings{Max: 5}, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	for _, c := range cards {
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
	d, _ = d.settle()
	return d
}

func aReport(id, subject string, o Origin) Card {
	return Card{ID: id, Slot: LocationReport, Origin: o, Subject: subject, Headline: strings.ToUpper(subject)}
}

// joins is every structural card the reconcile owns, in schedule order.
func joins(d Director) []string {
	var out []string
	for _, c := range d.lineup.Cards(MainTrack) {
		if isJoinID(c.ID) {
			out = append(out, c.ID)
		}
	}
	return out
}

func TestNothingIsBookendedWhenNoCardIsTheOperators(t *testing.T) {
	d := seed(t, aReport("a", "oceanside", FromDirector), aReport("b", "carlsbad", FromDirector))

	if got := joins(d); len(got) != 0 {
		t.Errorf("the Director adds nothing to a running order that asks for nothing; got %v", got)
	}
}

func TestAnOperatorsCardIsBookended(t *testing.T) {
	// D-43's stated rule, and it exercises all three of the HUM LEAD's cases at
	// once: `a→op` is the leading transition, `op→b` the tailing one, and
	// together they are "bothTransitions".
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator),
		aReport("b", "carlsbad", FromDirector))

	got := joins(d)
	want := []string{joinID("a", "op"), joinID("op", "b")}
	if !equal(got, want) {
		t.Fatalf("an operator's card is bookended;\n got %v\nwant %v", got, want)
	}
	// AND THEY SIT AT THE JOIN, not merely somewhere in the track. A transition
	// read in the wrong place is worse than none: it announces something that
	// is not what comes next.
	if s := ids(d.lineup.Cards(MainTrack)); !equal(s, []string{"a", joinID("a", "op"), "op", joinID("op", "b"), "b"}) {
		t.Errorf("the transitions must sit AT the joins; the schedule reads %v", s)
	}
	// THE OPERATOR SEES NONE OF IT (D-44).
	if p := ids(d.lineup.Projection(MainTrack)); !equal(p, []string{"a", "op", "b"}) {
		t.Errorf("the line-up is unchanged by the Director's own cards; got %v", p)
	}
}

func TestTheTransitionsAreReDerivedWhenTheOperatorsCardMoves(t *testing.T) {
	// "those transition cards MOVE with the operator's card in the event the
	// operator then promotes or quashes that card up or down the rotation."
	//
	// Under join-ownership they do not move — they are re-derived, and the
	// result is what the HUM LEAD asked for WITHOUT an association to keep in
	// step. Promoting `op` to the head leaves one join, not two, because a card
	// at the head has nothing before it to hand off from.
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator),
		aReport("b", "carlsbad", FromDirector))

	d, _ = d.Step(Moved{ID: "op", To: 0})

	if p := ids(d.lineup.Projection(MainTrack)); !equal(p, []string{"op", "a", "b"}) {
		t.Fatalf("the operator's move must happen; the line-up reads %v", p)
	}
	got := joins(d)
	want := []string{joinID("op", "a")}
	if !equal(got, want) {
		t.Errorf("the transitions follow the card to its new join;\n got %v\nwant %v", got, want)
	}
	if s := ids(d.lineup.Cards(MainTrack)); !equal(s, []string{"op", joinID("op", "a"), "a", "b"}) {
		t.Errorf("and the stale one is gone from the schedule; it reads %v", s)
	}
}

func TestDroppingTheOperatorsCardTakesItsTransitionsAndTheyDoNotReachThePile(t *testing.T) {
	// "the associated transition cards simply 'go away' — they do not show up in
	// the discard pile, because if the card is resurrected the Director will
	// re-evaluate if transitions are needed for that card AT THAT TIME."
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator),
		aReport("b", "carlsbad", FromDirector))
	if len(joins(d)) != 2 {
		t.Fatalf("fixture: want two joins, got %v", joins(d))
	}

	d, _ = d.Step(Dropped{ID: "op"})

	if got := joins(d); len(got) != 0 {
		t.Errorf("the transitions go away with the card they served; got %v", got)
	}
	if s := ids(d.lineup.Cards(MainTrack)); !equal(s, []string{"a", "b"}) {
		t.Errorf("the schedule keeps only what the operator scheduled; got %v", s)
	}
	pile := ids(d.lineup.Discarded())
	if !equal(pile, []string{"op"}) {
		t.Errorf("ONLY the operator's own card reaches the undo pile; it holds %v", pile)
	}
}

func TestTheStalenessNoticeIsNotTheReconcilesToRemove(t *testing.T) {
	// THE MODEL MUST CARRY BOTH KINDS. The staleness notice is a structural card
	// that is NOT derived from a join — it explains an absence, and no adjacency
	// implies it. A reconcile that owned every structural card would delete it,
	// silently, leaving the listener the gap it exists to explain.
	d := seed(t, aReport("a", "oceanside", FromDirector))
	notice, err := Propose(Card{ID: staleTransitionID, Slot: Transition, Origin: FromDirector,
		Subject: "stale read", Headline: "Report out of date", Script: Say(staleTransitionText)})
	if err != nil {
		t.Fatalf("the fixture notice: %v", err)
	}
	admitted, err := notice.To(Admitted)
	if err != nil {
		t.Fatalf("admitting: %v", err)
	}
	d.lineup, err = d.lineup.Queue(MainTrack, admitted)
	if err != nil {
		t.Fatalf("queueing: %v", err)
	}

	d, _ = d.settle()

	if _, _, held := d.lineup.find(staleTransitionID); !held {
		t.Error("the reconcile owns the cards it MINTS, and nothing else; it deleted the staleness notice")
	}
}

func TestReconcilingTwiceAddsNothingTheSecondTime(t *testing.T) {
	// The reconcile runs on EVERY settle, so it must be idempotent or the
	// schedule grows a transition per event. The deterministic id is what makes
	// it so: the same join always mints the same card, and the lineup refuses a
	// second under one identity.
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator))
	before := ids(d.lineup.Cards(MainTrack))

	d, _ = d.settle()
	d, _ = d.settle()

	if after := ids(d.lineup.Cards(MainTrack)); !equal(after, before) {
		t.Errorf("settling again changed the schedule;\nbefore %v\n after %v", before, after)
	}
}

// onAir drives the schedule until `id` is the card being read, deterministically
// — a report is built, takes the air and finishes; a transition needs no build.
func onAir(t *testing.T, d Director, id string) Director {
	t.Helper()
	for range 12 { // bounded (P10-02)
		if c, live := d.lineup.OnAir(); live {
			if c.ID == id {
				return d
			}
			d, _ = d.Step(Finished{ID: c.ID})
			continue
		}
		next, _, ok := d.lineup.Next()
		if !ok {
			t.Fatalf("the schedule ran dry before %s reached the air", id)
		}
		if next.Words() == "" {
			d, _ = d.Step(Built{ID: next.ID, Script: Say("the report for " + next.Subject + ".")})
			continue
		}
		d, _ = d.settle()
	}
	t.Fatalf("%s never reached the air", id)
	return d
}

// THE JOIN MUST SURVIVE ITS LEAD-IN BEING READ, and this is the defect the
// skipped on-air test uncovered.
//
// Deriving the wanted set from the RUNNING ORDER alone prunes a transition the
// moment the card before it finishes: `a` is read, leaves the schedule, and the
// hand-off introducing what follows is deleted in the same settle — one step
// before it would have been spoken. The listener hears the next report begin
// cold, and nothing anywhere reports a fault.
func TestAJoinSurvivesTheCardBeforeItBeingRead(t *testing.T) {
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator))
	id := joinID("a", "op")
	if _, _, held := d.lineup.find(id); !held {
		t.Fatalf("fixture: the join must exist; schedule is %v", ids(d.lineup.Cards(MainTrack)))
	}

	d, _ = d.Step(Built{ID: "a", Script: Say("the report for oceanside.")})
	d = onAir(t, d, "a")
	d, _ = d.Step(Finished{ID: "a"})

	if _, _, held := d.lineup.find(id); !held {
		t.Fatalf("the hand-off was deleted one step before it would be read; schedule is %v",
			ids(d.lineup.Cards(MainTrack)))
	}
}

func TestATransitionOnTheAirIsNotRemoved(t *testing.T) {
	// D-45: ON AIR is locked, and only a GO TO STANDBY or a catastrophe changes
	// it. So the reconcile has a FROZEN PREFIX — it may re-derive the running
	// order, but it can never take back something already being read.
	d := seed(t,
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator))
	id := joinID("a", "op")

	d, _ = d.Step(Built{ID: "a", Script: Say("the report for oceanside.")})
	d = onAir(t, d, id)

	// Now make the join meaningless by dropping the card it introduces.
	d, _ = d.Step(Dropped{ID: "op"})

	if _, _, held := d.lineup.find(id); !held {
		t.Error("a transition being READ cannot be taken back; the listener is mid-sentence")
	}
	if c, live := d.lineup.OnAir(); !live || c.ID != id {
		t.Errorf("and it is still the card on the air; got %v (live=%t)", c.ID, live)
	}
}
