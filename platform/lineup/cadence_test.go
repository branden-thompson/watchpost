package lineup

import (
	"testing"
	"time"
)

// cadence_test.go — F-76: what the Director remembers, and how little (D-48).
//
//	"it's bounded by the types of cards/reports we have (so it's bounded) and
//	it's just a 'since last read of this type' — read history is too overweight
//	and violates our 'done/discarded cards can pile up' concern."
//
//	"I would also architect this in a way where the Director can still function
//	and make other evaluations in the event this particular stack fails or is
//	somehow disabled (maybe a broadcaster specific setting — where the Operator
//	does/does not want 'last read' to be a factor in Director prioritization)."

func cadenceDirector(t *testing.T, weigh bool) Director {
	t.Helper()
	d := New(Settings{Max: 5, Depth: 4, WeighLastRead: weigh},
		time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	return d
}

func TestTheDirectorRemembersWhenAKindWasLastRead(t *testing.T) {
	base := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	d := cadenceDirector(t, true)
	if _, ever := d.lastReadOf(LocationReport); ever {
		t.Fatal("nothing has been read yet")
	}

	d = seedRead(t, d, "a", "oceanside", LocationReport)

	at, ever := d.lastReadOf(LocationReport)
	if !ever {
		t.Fatal("a card read in full must be remembered as a read of its kind")
	}
	if !at.Equal(base) {
		t.Errorf("it is remembered at the clock the Director held; got %v want %v", at, base)
	}
	// AND ONLY ITS OWN KIND. The whole value of the record is telling one kind
	// of read from another; a write that touched every slot would say the
	// station had just done everything.
	if _, ever := d.lastReadOf(SevereRead); ever {
		t.Error("reading a location report says nothing about when a severe read last happened")
	}
}

func TestADiscardedCardWasNeverRead(t *testing.T) {
	// The record is "since last READ", not "since last removed". A dropped card
	// was taken away precisely so the listener would not hear it, so counting it
	// would tell the Director the station had covered something it had not.
	d := cadenceDirector(t, true)
	d = seedCard(t, d, "a", "oceanside", LocationReport)

	d, _ = d.Step(Dropped{ID: "a"})

	if _, ever := d.lastReadOf(LocationReport); ever {
		t.Error("a card the operator dropped was never read to anyone")
	}
}

func TestTheOverdueKindOutranksTheRecentlyReadOne(t *testing.T) {
	// The HUM LEAD's own example: "it's a low priority read (station credits) but
	// we're due for a location report because we haven't had one in a while."
	//
	// STOOD IN WITH A SEVERE RECAP, because station credits HAVE NO SLOT YET —
	// that is D-31's card-type work, and the registry says a slot nobody
	// proposes is dead code. What is under test is the cadence mechanism, which
	// discriminates between KINDS whatever those kinds turn out to be.
	d := cadenceDirector(t, true)
	d = seedRead(t, d, "s0", "hazard", SevereRead) // credits just went out
	d.now = d.now.Add(6 * time.Minute)

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "hazard", Headline: "SEVERE RECAP", Slot: SevereRead},
		{Ref: "oceanside", Headline: "OCEANSIDE, CA", Slot: LocationReport},
	}})

	got := subjects(d)
	if len(got) == 0 {
		t.Fatal("the top-off queued nothing at all")
	}
	if got[0] != "oceanside" {
		t.Errorf("the kind nothing has read in six minutes takes the slot first; got %v", got)
	}
}

func TestTheOperatorCanTurnLastReadOffAndTheDirectorStillChooses(t *testing.T) {
	// THE DEGRADATION REQUIREMENT, stated as a test: with the cadence term off
	// the Director must still produce an order — not refuse, not stall, not fall
	// back to the producer's whim. It falls through to the watchlist, which is
	// what it ranked by before this term existed.
	d := New(Settings{Max: 5, Depth: 4, WeighLastRead: false,
		Watchlist: []string{"hazard", "oceanside"}},
		time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d = seedRead(t, d, "s0", "hazard", SevereRead)
	d.now = d.now.Add(6 * time.Minute)

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "oceanside", Headline: "OCEANSIDE, CA", Slot: LocationReport},
		{Ref: "hazard", Headline: "SEVERE RECAP", Slot: SevereRead},
	}})

	got := subjects(d)
	if len(got) < 2 {
		t.Fatalf("the Director must still fill the slots; got %v", got)
	}
	if got[0] != "hazard" {
		t.Errorf("with the term off the watchlist decides, exactly as it did before; got %v", got)
	}
}

func TestNothingIsOverdueBeforeAnythingHasBeenRead(t *testing.T) {
	// A COLD START MUST NOT DISCRIMINATE. Every kind is equally unread, so the
	// term contributes nothing and the order falls through — which is the same
	// degradation path as the setting being off, reached by a different route.
	d := New(Settings{Max: 5, Depth: 4, WeighLastRead: true,
		Watchlist: []string{"hazard", "oceanside"}},
		time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "oceanside", Headline: "OCEANSIDE, CA", Slot: LocationReport},
		{Ref: "hazard", Headline: "SEVERE RECAP", Slot: SevereRead},
	}})

	got := subjects(d)
	if len(got) < 2 || got[0] != "hazard" {
		t.Errorf("with nothing read, the watchlist decides; got %v", got)
	}
}

func TestASlotOutsideTheRegistryIsNotOverdue(t *testing.T) {
	// FAIL SOFT, NOT CLOSED. A corrupt slot must cost the Director its cadence
	// opinion, not its ability to choose — which is the HUM LEAD's requirement
	// that "the Director can still function ... in the event this particular
	// stack fails".
	d := cadenceDirector(t, true)
	if _, ever := d.lastReadOf(Slot(-1)); ever {
		t.Error("a slot outside the registry has no reading")
	}
	if _, ever := d.lastReadOf(numSlots); ever {
		t.Error("a slot outside the registry has no reading")
	}
	if got := d.overdue(Slot(99)); got != 0 {
		t.Errorf("and it is not overdue either; got %v", got)
	}
}

// seedCard queues one card and settles.
func seedCard(t *testing.T, d Director, id, subject string, slot Slot) Director {
	t.Helper()
	c := Card{ID: id, Slot: slot, Origin: FromDirector, Subject: subject, Headline: subject}
	if slot.structural() {
		c.Script = Say("the words for " + subject + ".")
	}
	proposed, err := Propose(c)
	if err != nil {
		t.Fatalf("fixture %s: %v", id, err)
	}
	admitted, err := proposed.To(Admitted)
	if err != nil {
		t.Fatalf("fixture %s: %v", id, err)
	}
	d.lineup, err = d.lineup.Queue(trackFor(slot), admitted)
	if err != nil {
		t.Fatalf("fixture %s: %v", id, err)
	}
	d, _ = d.settle()
	return d
}

// seedRead drives one card all the way through a completed read.
func seedRead(t *testing.T, d Director, id, subject string, slot Slot) Director {
	t.Helper()
	d = seedCard(t, d, id, subject, slot)
	if slot.textAtStandby() {
		d, _ = d.Step(Built{ID: id, Script: Say("the report for " + subject + ".")})
	}
	d, _ = d.settle()
	if c, live := d.lineup.OnAir(); !live || c.ID != id {
		t.Fatalf("fixture %s never reached the air (on air: %q)", id, c.ID)
	}
	d, _ = d.Step(Finished{ID: id})
	return d
}

// A READ THAT WAS CUT OFF IS NOT A READ, and a plant found this untested.
//
// `onDropped` refuses the on-air card outright, so the drop test above never
// reaches the recording line at all. The case that DOES is the one D-45 names as
// the only thing that can take a live card off the air: the operator goes to
// standby mid-sentence. The listener heard part of a report; the station has not
// covered that location, and a Director told otherwise would wait a full cycle
// before coming back to it.
func TestAReadCutOffByStandbyWasNotARead(t *testing.T) {
	d := cadenceDirector(t, true)
	d = seedCard(t, d, "a", "oceanside", LocationReport)
	d, _ = d.Step(Built{ID: "a", Script: Say("the report for oceanside.")})
	d, _ = d.settle()
	if c, live := d.lineup.OnAir(); !live || c.ID != "a" {
		t.Fatalf("fixture: the card must be reading; on air %q", c.ID)
	}

	d, _ = d.Step(Powered{To: Stopped}) // GO TO STANDBY, mid-read

	if at, ever := d.lastReadOf(LocationReport); ever {
		t.Errorf("a read cut off mid-sentence is not a read; it was recorded at %v", at)
	}
}

func TestTheCardTakesTheKindTheProducerProposed(t *testing.T) {
	// Without this the Director could quietly turn every proposal into a
	// location report — the cadence term would still ORDER correctly and the
	// card queued would be the wrong kind, read by the wrong voice.
	d := cadenceDirector(t, true)

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "hazard", Headline: "SEVERE RECAP", Slot: SevereRead},
	}})

	cards := d.lineup.Cards(MainTrack)
	if len(cards) != 1 {
		t.Fatalf("want one card; got %d", len(cards))
	}
	if cards[0].Slot != SevereRead {
		t.Errorf("the card is the KIND the producer proposed; got %v", cards[0].Slot)
	}
}

func TestTheTopOffRefusesWhatDoesNotBelongOnTheMainTrack(t *testing.T) {
	// A takeover drains on the RAIL (DR-3). Admitted here it would ride the
	// programme instead — and it would still count against the depth, so the
	// slot it was meant to fill stays empty while the walk counts it spent.
	d := cadenceDirector(t, true)

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "tornado", Headline: "TORNADO WARNING", Slot: BreakingAlert},
		{Ref: "oceanside", Headline: "OCEANSIDE, CA", Slot: LocationReport},
	}})

	if got := subjects(d); !equal(got, []string{"oceanside"}) {
		t.Errorf("only main-track cards top the main track off; got %v", got)
	}
	if n := len(d.lineup.Cards(AlertRail)); n != 0 {
		t.Errorf("and the rail is not something the top-off fills either; it holds %d", n)
	}
}

func TestTheProducerCannotProposeTheDirectorsOwnCards(t *testing.T) {
	// D-43: transitions are DERIVED from the running order. One arriving as a
	// proposal would be a second author of the same thing.
	//
	// IT HOLDS BY ACCIDENT TODAY and this test says so: a structural card's
	// words are fixed at proposal, `Proposal` carries none, and `check` refuses
	// a wordless transition — so the refusal happens for an unrelated reason.
	// That is the D-42 shape, and it is stated in `admit` for the same reason.
	d := cadenceDirector(t, true)

	d, _ = d.Step(Offered{Proposals: []Proposal{
		{Ref: "handoff", Headline: "COMING UP", Slot: Transition},
		{Ref: "oceanside", Headline: "OCEANSIDE, CA", Slot: LocationReport},
	}})

	if got := subjects(d); !equal(got, []string{"oceanside"}) {
		t.Errorf("the Director's own cards are not the producer's to propose; got %v", got)
	}
}
