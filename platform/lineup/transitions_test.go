package lineup

import (
	"testing"
	"time"
)

// transitions_test.go — MVS-D-80's rule, as a property of the CARD KIND (D-49).
//
// THE TRIGGER IS RULED AND IT IS NOT MINE (HUM LEAD 2026-09-05, follow-ups.md
// F-27): a transition "fires when something that INTERRUPTED THE PROGRAMME
// leaves the air and the programme resumes", and it does NOT fire
// location-to-location because "the location scripts already announce their
// location."
//
// THAT REASON IS THE GENERAL RULE. A kind that announces itself needs no
// lead-in; a kind that interrupted the programme needs a hand-back after it. Both
// are properties of the KIND, so both are rows in the slot registry — which is
// what makes adding an inter-card card "low cost" in the HUM LEAD's words rather
// than in mine.
//
// AN EARLIER ARM FIRED ON `Origin == FromOperator` AND WAS WRONG. D-43's
// bookended operator card was "an example to show the function of the DIRECTOR
// understanding how a card fits into the line-up", not a requirement — and as a
// rule it fired location-to-location, which MVS-D-80 forbids and S-5 calls
// "jarring to a listening audience."

const returnWords = "Watchpost Radio now returns to its regularly scheduled programming."

func seedT(t *testing.T, s Settings, cards ...Card) Director {
	t.Helper()
	if s.Max == 0 {
		s.Max = 5
	}
	d := New(s, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	for _, c := range cards {
		proposed, err := Propose(c)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
		admitted, err := proposed.To(Admitted)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
		d.lineup, err = d.lineup.Queue(trackFor(c.Slot), admitted)
		if err != nil {
			t.Fatalf("fixture %s: %v", c.ID, err)
		}
	}
	d, _ = d.settle()
	return d
}

func aReport(id, subject string, o Origin) Card {
	return Card{ID: id, Slot: LocationReport, Origin: o, Subject: subject, Headline: subject}
}

func aTakeover(id, subject string) Card {
	return Card{ID: id, Slot: BreakingAlert, Origin: FromObserver, Subject: subject, Headline: subject}
}

// structural is every card the Director added itself, per track, in order.
func structural(d Director, t Track) []string {
	var out []string
	for _, c := range d.lineup.Cards(t) {
		if isJoinID(c.ID) {
			out = append(out, c.ID)
		}
	}
	return out
}

func TestALocationReportGetsNoTransitionEitherSide(t *testing.T) {
	// MVS-D-80, in as many words: it "does NOT fire location-to-location", and
	// S-5 says Broadcaster must not offer the inter-report transitions Observer
	// allows, "which would be jarring to a listening audience."
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aReport("a", "oceanside", FromDirector),
		aReport("b", "carlsbad", FromDirector))

	if got := structural(d, MainTrack); len(got) != 0 {
		t.Errorf("two location reports need nothing between them; got %v", got)
	}
}

func TestAnOperatorsLocationReportIsStillJustALocationReport(t *testing.T) {
	// D-43 REREAD: the bookending was an EXAMPLE of the Director understanding
	// how a card fits, not a rule keyed on origin. A card's kind decides what it
	// needs; who asked for it does not change what it sounds like.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aReport("a", "oceanside", FromDirector),
		aReport("op", "fallbrook", FromOperator),
		aReport("b", "carlsbad", FromDirector))

	if got := structural(d, MainTrack); len(got) != 0 {
		t.Errorf("origin is not what makes a handoff necessary; got %v", got)
	}
}

// airing drives a card to ON AIR.
func airing(t *testing.T, d Director, id string) Director {
	t.Helper()
	c, ok := d.find(id)
	if !ok {
		t.Fatalf("%s is not in the schedule", id)
	}
	if c.Slot.textAtStandby() {
		d, _ = d.Step(Built{ID: id, Script: Say("the read for " + c.Subject + ".")})
	}
	d, _ = d.settle()
	if live, on := d.lineup.OnAir(); !on || live.ID != id {
		t.Fatalf("%s never reached the air (on air: %q)", id, live.ID)
	}
	return d
}

func TestATakeoverHandsTheProgrammeBackAfterIt(t *testing.T) {
	// "Fires when something that interrupted the programme leaves the air and
	// the programme resumes: an alert takeover." The takeover drains on the
	// rail, so its hand-back is the last thing the rail reads before normal
	// programming continues.
	//
	// IT APPEARS WHILE THE TAKEOVER IS STILL READING, which is what F-27's
	// design note asks for and what stops the duck bouncing.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aReport("a", "oceanside", FromDirector),
		aTakeover("t1", "tornado warning"))
	if got := structural(d, AlertRail); len(got) != 0 {
		t.Fatalf("nothing is owed before the takeover has interrupted anything; got %v", got)
	}
	d = airing(t, d, "t1")

	got := structural(d, AlertRail)
	if !equal(got, []string{tailID("t1")}) {
		t.Fatalf("the takeover hands the programme back after it; the rail holds %v", ids(d.lineup.Cards(AlertRail)))
	}
	if s := ids(d.lineup.Cards(AlertRail)); !equal(s, []string{"t1", tailID("t1")}) {
		t.Errorf("and it sits AFTER the read it hands back from; got %v", s)
	}
	// IT SAYS WHAT THE SCRIPT SAYS, and the Director does not compose it.
	c, _ := d.find(tailID("t1"))
	if c.Words() != returnWords {
		t.Errorf("the words are the station's, handed in; got %q", c.Words())
	}
	// AND THE OPERATOR SEES NONE OF IT (D-44, and the HUM LEAD 2026-09-10:
	// "Operator never deals with transitions, that's the Director's job").
	if p := ids(d.lineup.Projection(AlertRail)); !equal(p, []string{"t1"}) {
		t.Errorf("the line-up shows the takeover alone; got %v", p)
	}
}

func TestNoHandBackWhenTheStationHasNoWordsForOne(t *testing.T) {
	// THE DEGRADATION PATH, and it is the same shape as D-48's: the words are
	// the station's, composed by the app from the script library. With none, the
	// Director arranges nothing rather than inventing a line — a station that
	// speaks words nobody wrote is worse than one that simply moves on.
	d := seedT(t, Settings{}, // no ProgrammeReturn
		aReport("a", "oceanside", FromDirector),
		aTakeover("t1", "tornado warning"))
	d = airing(t, d, "t1")

	if got := structural(d, AlertRail); len(got) != 0 {
		t.Errorf("with no words there is nothing to say; got %v", got)
	}
	if s := ids(d.lineup.Cards(AlertRail)); !equal(s, []string{"t1"}) {
		t.Errorf("and the takeover still reads normally; got %v", s)
	}
}

func TestATakeoverDroppedBeforeItAiredLeavesNoStrayHandBack(t *testing.T) {
	// NOTHING WAS INTERRUPTED, so there is nothing to hand back from. This is
	// what the on-air condition buys: a first attempt minted the hand-back at
	// ADMISSION, and a dropped takeover left a stray "we now return to our
	// regularly scheduled programming" with nothing before it — the station
	// announcing a return from a programme it never left.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aReport("a", "oceanside", FromDirector),
		aTakeover("t1", "tornado warning"))

	d, _ = d.Step(Dropped{ID: "t1"})

	if got := structural(d, AlertRail); len(got) != 0 {
		t.Errorf("a takeover that never aired interrupted nothing; got %v", got)
	}
	if pile := ids(d.lineup.Discarded()); !equal(pile, []string{"t1"}) {
		t.Errorf("and only the operator's own card reaches the pile; it holds %v", pile)
	}
}

func TestTheHandBackSurvivesTheTakeoverBeingRead(t *testing.T) {
	// THE DEFECT A SKIPPED TEST FOUND, kept pinned through the rewrite. When the
	// takeover finishes it leaves the schedule, and a hand-back derived only
	// from what is still there would be deleted one step before it is spoken —
	// the listener hears the programme resume with no hand-back at all, which is
	// the precise thing MVS-D-80 exists to prevent.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aTakeover("t1", "tornado warning"))
	id := tailID("t1")

	d = airing(t, d, "t1")
	d, _ = d.Step(Finished{ID: "t1"})

	if _, _, held := d.lineup.find(id); !held {
		t.Fatalf("the hand-back was deleted one step before it would be read; rail is %v",
			ids(d.lineup.Cards(AlertRail)))
	}
}

func TestATransitionOnTheAirIsNotRemoved(t *testing.T) {
	// D-45: ON AIR is locked, and only a GO TO STANDBY or a catastrophe changes
	// it. The reconcile may re-derive the running order; it can never take back
	// something being read.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aTakeover("t1", "tornado warning"))
	id := tailID("t1")

	d = airing(t, d, "t1")
	d, _ = d.Step(Finished{ID: "t1"})
	d, _ = d.settle()
	if c, live := d.lineup.OnAir(); !live || c.ID != id {
		t.Fatalf("fixture: the hand-back must be reading; on air %q", c.ID)
	}

	d = d.reconcileJoins() // whatever it now thinks, it may not take this back

	if _, _, held := d.lineup.find(id); !held {
		t.Error("a transition being READ cannot be taken back; the listener is mid-sentence")
	}
}

func TestReconcilingTwiceAddsNothingTheSecondTime(t *testing.T) {
	// The reconcile runs on EVERY settle, so it must be idempotent or the
	// schedule grows a transition per event. The deterministic id is what makes
	// it so.
	d := seedT(t, Settings{ProgrammeReturn: returnWords},
		aReport("a", "oceanside", FromDirector),
		aTakeover("t1", "tornado warning"))
	d = airing(t, d, "t1")
	before := append(ids(d.lineup.Cards(MainTrack)), ids(d.lineup.Cards(AlertRail))...)

	d, _ = d.settle()
	d, _ = d.settle()

	after := append(ids(d.lineup.Cards(MainTrack)), ids(d.lineup.Cards(AlertRail))...)
	if !equal(after, before) {
		t.Errorf("settling again changed the schedule;\nbefore %v\n after %v", before, after)
	}
}

func TestTheStalenessNoticeIsNotTheReconcilesToRemove(t *testing.T) {
	// THE MODEL MUST CARRY BOTH KINDS. The staleness notice is a structural card
	// that is NOT derived from a kind's policy — it explains an absence, and no
	// arrangement implies it. A reconcile that owned every structural card would
	// delete it silently, leaving the listener the gap it exists to explain.
	d := seedT(t, Settings{ProgrammeReturn: returnWords}, aReport("a", "oceanside", FromDirector))
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

// THE REGISTRY IS THE RULE, so the rule is asserted as a table rather than
// inferred from behaviour. A row added without a reason is what this catches.
func TestEveryKindDeclaresWhatItNeedsAroundIt(t *testing.T) {
	for s := Slot(0); s < numSlots; s++ { // bounded by the registry (P10-02)
		row, ok := s.row()
		if !ok {
			t.Fatalf("%v has no row", s)
		}
		switch s {
		case BreakingAlert:
			if !row.handsBack {
				t.Error("a takeover interrupts the programme, so the listener is handed back (MVS-D-80)")
			}
		default:
			if row.handsBack {
				t.Errorf("%v hands the programme back, and nothing rules that it should", s)
			}
		}
		// NOTHING ANNOUNCES ITSELF YET. Station credits will — "please standby
		// for station identification" — and that is D-31's card type. The
		// machinery is here; the row is not, because a rule nobody has made is
		// not a rule.
		if row.announced {
			t.Errorf("%v is announced first, and nothing rules that it should", s)
		}
	}
}

// insertBeside is what BOTH sides depend on, and only one side has a production
// row today — so it is pinned directly rather than through a rule that cannot
// yet fire. When station credits get a slot (D-31) the lead arm gets its own
// behavioural test; until then this is what stops `before` silently meaning
// "after".
func TestACardCanBeInsertedOnEitherSideOfAnother(t *testing.T) {
	d := seedT(t, Settings{}, aReport("a", "oceanside", FromDirector))
	mark := func(id string) Card {
		c, err := Propose(Card{ID: id, Slot: Transition, Origin: FromDirector,
			Subject: "mark", Headline: "mark", Script: Say("a marker.")})
		if err != nil {
			t.Fatalf("proposing %s: %v", id, err)
		}
		c, err = c.To(Admitted)
		if err != nil {
			t.Fatalf("admitting %s: %v", id, err)
		}
		return c
	}

	next, err := d.lineup.insertBeside(MainTrack, "a", mark("before"), true)
	if err != nil {
		t.Fatalf("inserting before: %v", err)
	}
	next, err = next.insertBeside(MainTrack, "a", mark("after"), false)
	if err != nil {
		t.Fatalf("inserting after: %v", err)
	}

	if got := ids(next.Cards(MainTrack)); !equal(got, []string{"before", "a", "after"}) {
		t.Errorf("before means ahead of it and after means behind it; got %v", got)
	}
}
