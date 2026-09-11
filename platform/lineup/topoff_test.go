package lineup

import (
	"testing"
	"time"
)

// topoff_test.go — D-40: the Producer proposes, the Director chooses.
//
// THE GAP THIS CLOSES WAS FOUND BY WALKING A USER FLOW, not by a gate. Only two
// things queued a main-track card — the deck reporting a location needs a read,
// and the operator's undo — and NOTHING READ THE TRACK'S DEPTH. So the schedule
// held about one card while the console drew ten slots, and dropping a card left
// slot [9] empty for ever.

// offering is a Director at Running with a watchlist, ready to be topped off.
func offering(t *testing.T, depth int, watchlist ...string) Director {
	t.Helper()
	d := New(Settings{Max: 5, Depth: depth, Watchlist: watchlist},
		time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	return d
}

// subjects is the main track's cards by what they read, in schedule order.
func subjects(d Director) []string {
	var out []string
	for _, c := range d.lineup.Cards(MainTrack) {
		out = append(out, c.Subject)
	}
	return out
}

func offers(refs ...string) Offered {
	var ps []Proposal
	for _, r := range refs {
		ps = append(ps, Proposal{Ref: r, Headline: "REPORT FOR " + r})
	}
	return Offered{Proposals: ps}
}

func TestTheDirectorFillsTheMainTrackToDepth(t *testing.T) {
	d := offering(t, 3, "oceanside", "carlsbad", "encinitas", "vista")

	d, fx := d.Step(offers("oceanside", "carlsbad", "encinitas", "vista"))

	got := subjects(d)
	if len(got) != 3 {
		t.Fatalf("a depth of 3 takes exactly 3 of the 4 proposals; got %d (%v)", len(got), got)
	}
	// AND EACH ONE MUST BE SET MOVING. A card queued and left is a lineup the
	// console draws and the station never speaks — the defect a plant caught in
	// the rotation's own test by holding the effects and never asking for them.
	published := 0
	built := 0
	for _, e := range fx {
		switch e.(type) {
		case Publish:
			published++
		case BuildCard:
			built++
		}
	}
	if built == 0 {
		t.Error("topping off must also ask for the first card's words; got no BuildCard")
	}
	// ONE PUBLISH FOR THE WHOLE TOP-OFF, not one per card. "A step publishes
	// ONCE, LAST" is the settle's own rule, and three publishes would hand the
	// console three schedules for one operator action.
	if published != 1 {
		t.Errorf("a step publishes once, last; got %d Publish effects", published)
	}
}

func TestTheDirectorChoosesInWatchlistOrderRatherThanOfferOrder(t *testing.T) {
	// THE PRODUCER OFFERS IN ITS OWN ORDER and the Director is not bound by it.
	// This is the whole substance of "the Director chooses": were it taking the
	// first proposal offered, the producer would be choosing and the role split
	// would be a comment rather than a behaviour.
	d := offering(t, 2, "oceanside", "carlsbad", "encinitas")

	d, _ = d.Step(offers("encinitas", "carlsbad", "oceanside"))

	got := subjects(d)
	want := []string{"oceanside", "carlsbad"}
	if len(got) != len(want) {
		t.Fatalf("a depth of 2 takes 2; got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the Director orders by the OPERATOR'S watchlist, not the producer's offer order;\n got %v\nwant %v", got, want)
		}
	}
}

func TestAProposalOutsideTheWatchlistRanksAfterOneOnIt(t *testing.T) {
	// The watchlist is the operator's stated rotation. Something the producer
	// offers that is not on it is still worth reading — it is inside the service
	// radius — but it does not outrank what the operator asked for.
	d := offering(t, 1, "carlsbad")

	d, _ = d.Step(offers("fallbrook", "carlsbad"))

	got := subjects(d)
	if len(got) != 1 || got[0] != "carlsbad" {
		t.Fatalf("a watchlist location outranks one that is merely in range; got %v", got)
	}
}

func TestAnOfferQueuesNothingWhenTheTrackIsAlreadyDeepEnough(t *testing.T) {
	d := offering(t, 2, "oceanside", "carlsbad", "encinitas")
	d, _ = d.Step(offers("oceanside", "carlsbad"))

	before := len(subjects(d))
	d, fx := d.Step(offers("encinitas"))

	if got := len(subjects(d)); got != before {
		t.Errorf("a full track takes nothing more; held %d, now %d", before, got)
	}
	if len(fx) != 0 {
		t.Errorf("an offer that changes nothing does nothing; got %d effects", len(fx))
	}
}

// THE LINE-UP IS PLANNED BEFORE IT IS BROADCAST (D-84, HUM LEAD 2026-09-11).
//
// THE INVERSE OF WHAT THIS USED TO ASSERT. It pinned "a stopped programme admits
// nothing" under DR-3's "admission is a promise to read" — which left the
// operator with ten empty slots and nothing to inspect, reorder or drop until
// after they had gone on the air. The ruling: "When the user enters Broadcaster
// Mode, the station is in STANDBY … At this point Producers should be grabbing
// locations from the pool, and proposing reports to the Director … The director
// should be choosing and populating the line-up."
//
// THE OLD HAZARD WAS "a rotation nobody can drop", AND THE OPERATOR CAN DROP IT.
// That is the surface's whole reason to exist.
func TestAnOfferFillsTheLineUpWhileTheProgrammeIsOnStandby(t *testing.T) {
	d := New(Settings{Max: 5, Depth: 3, Watchlist: []string{"oceanside"}},
		time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	d, _ = d.Step(offers("oceanside", "carlsbad"))

	if got := subjects(d); len(got) != 2 {
		t.Errorf("a station on standby must fill its line-up so the operator can manage it; got %v", got)
	}
	// AND NOTHING TAKES THE AIR. Planning is not performing, and `airOnce` is
	// the one thing left that asks.
	if c, on := d.lineup.OnAir(MainTrack); on {
		t.Errorf("a stopped programme put %q on the air", c.ID)
	}
}

// AND A TRACK THE BED HAS PAUSED KEEPS ITS LINE-UP TOPPED OFF (D-84).
//
// The same inversion, for the same reason. A cut-over PAUSES the main track
// (FR-4.2) — it does not abandon it — so the operator can go on planning the
// reads that resume when they cut back. Refusing to admit here left them
// managing an empty console while a relay played.
func TestTheBedHoldingTheProgrammeDoesNotStopTheTopOff(t *testing.T) {
	d := offering(t, 3, "oceanside", "carlsbad")
	d, _ = d.Step(CutOver{ToBed: true})

	d, _ = d.Step(offers("oceanside", "carlsbad", "bonsall"))

	if got := subjects(d); len(got) == 0 {
		t.Errorf("a paused track still plans what it will read when it comes back; got %v", got)
	}
	if c, on := d.lineup.OnAir(MainTrack); on {
		t.Errorf("the bed carries the programme and %q took the air anyway", c.ID)
	}
}

func TestTheTopOffNeverQueuesALocationTheScheduleIsAlreadyHolding(t *testing.T) {
	// ReadID is a pure function of the ref, and that IS the no-double-speak
	// mechanism (FR-2.5). A top-off that re-offered a location already in the
	// running order would either be refused by the lineup — wasting the slot it
	// was meant to fill — or, worse, read the same place twice.
	d := offering(t, 3, "oceanside", "carlsbad", "encinitas")
	d, _ = d.Step(NeedsRead{Ref: "carlsbad", Headline: "CARLSBAD, CA"})

	d, _ = d.Step(offers("oceanside", "carlsbad", "encinitas"))

	got := subjects(d)
	if len(got) != 3 {
		t.Fatalf("the top-off fills the REMAINING slots; got %d (%v)", len(got), got)
	}
	seen := map[string]int{}
	for _, s := range got {
		seen[s]++
	}
	for s, n := range seen {
		if n > 1 {
			t.Errorf("no location is scheduled twice; %q appears %d times in %v", s, n, got)
		}
	}
	if seen["encinitas"] != 1 {
		t.Errorf("the slot the duplicate would have wasted goes to the next candidate; got %v", got)
	}
}

func TestAMalformedProposalIsSkippedAndTheRestStillFill(t *testing.T) {
	// A proposal with no headline cannot become a card — the model refuses it at
	// the door. The producer's bug must not cost the slots the other proposals
	// could have filled, which is the same call onNeedsRead makes about a
	// malformed need.
	d := offering(t, 2, "oceanside", "carlsbad")
	ev := Offered{Proposals: []Proposal{
		{Ref: "oceanside"}, // no headline
		{Ref: "carlsbad", Headline: "CARLSBAD, CA"},
	}}

	d, _ = d.Step(ev)

	got := subjects(d)
	if len(got) != 1 || got[0] != "carlsbad" {
		t.Fatalf("a malformed proposal is skipped and the well-formed ones still fill; got %v", got)
	}
}

func TestAZeroDepthTopsOffNothing(t *testing.T) {
	// The depth is the DIRECTOR'S, and a station that never set one is not
	// asking to be topped off. Zero must be off rather than "fill for ever".
	d := offering(t, 0, "oceanside", "carlsbad")

	d, fx := d.Step(offers("oceanside", "carlsbad"))

	if got := subjects(d); len(got) != 0 {
		t.Errorf("a depth of zero tops off nothing; got %v", got)
	}
	if len(fx) != 0 {
		t.Errorf("and does no work; got %d effects", len(fx))
	}
}

func TestATopOffCardIsAWordlessLocationReport(t *testing.T) {
	// DR-7 is what makes a proposal cheap: the template carries a name and the
	// words materialise at standby. A proposal that arrived with words would be
	// a Composer decision made by a Producer.
	d := offering(t, 1, "oceanside")

	d, _ = d.Step(offers("oceanside"))

	cards := d.lineup.Cards(MainTrack)
	if len(cards) != 1 {
		t.Fatalf("want one card; got %d", len(cards))
	}
	c := cards[0]
	if c.Slot != LocationReport {
		t.Errorf("a topped-off card is a location report; got %v", c.Slot)
	}
	if c.ID != ReadID("oceanside") {
		t.Errorf("it takes the rotation's identity for the location, so it cannot double-speak; got %q", c.ID)
	}
	if c.Headline == "" {
		t.Error("it carries the headline the producer named it with")
	}
	if c.State != Standby && c.State != Admitted {
		t.Errorf("it enters the lineup admitted; got %v", c.State)
	}
}

func TestAnEmptyOfferChangesNothing(t *testing.T) {
	d := offering(t, 3, "oceanside")

	d, fx := d.Step(Offered{})

	if got := subjects(d); len(got) != 0 {
		t.Errorf("an offer of nothing queues nothing; got %v", got)
	}
	if len(fx) != 0 {
		t.Errorf("and does no work; got %d effects", len(fx))
	}
}
