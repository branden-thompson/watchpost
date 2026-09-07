package lineup

import (
	"testing"
)

// queued walks a proposal to ADMITTED and puts it on a track. Admission is the
// only door into the lineup, so this is the only way a test gets a card in.
func queued(t *testing.T, l Lineup, track Track, c Card) Lineup {
	t.Helper()
	out, err := l.Queue(track, at(t, proposed(t, c), Admitted))
	if err != nil {
		t.Fatalf("Queue(%v, %s): %v", track, c.ID, err)
	}
	return out
}

// ids is what a track reads as, for comparing an arrangement to the one a test
// asked for.
func ids(cards []Card) []string {
	out := make([]string, 0, len(cards))
	for _, c := range cards {
		out = append(out, c.ID)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTheLineupTakesAdmittedCardsAndNothingElse is DR-3's guarantee made
// structural: bounds apply at ADMISSION, so entry to the lineup IS admission
// and a card that never passed the pre-screen has no way in. This is what
// removes `breakingCap`'s defect rather than moving it — there is no later
// moment at which something can be cut.
func TestTheLineupTakesAdmittedCardsAndNothingElse(t *testing.T) {
	for s := State(0); s < numStates; s++ {
		c := report("Bonsall")
		c.State = s
		_, err := Lineup{}.Queue(MainTrack, c)
		if got, want := err == nil, s == Admitted; got != want {
			t.Errorf("Queue of a card at %v accepted = %v, want %v", s, got, want)
		}
	}
}

// TestTheLineupRefusesACardItCannotAddress: two cards under one identity make
// Set and Remove ambiguous, and an ambiguous Remove is a card read twice.
func TestTheLineupRefusesACardItCannotAddress(t *testing.T) {
	l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	twin := at(t, proposed(t, report("Bonsall")), Admitted)
	if _, err := l.Queue(AlertRail, twin); err == nil {
		t.Error("the lineup took a second card under one identity")
	}
	if _, err := l.Queue(numTracks, twin); err == nil {
		t.Error("the lineup took a card onto a track that does not exist")
	}
}

// TestTheAlertRailDrainsBeforeTheMainTrack is DR-3's precedence. The rail is
// always present and usually empty; when it holds anything, those cards are read
// first and in order, and normal programming resumes only when it is dry.
func TestTheAlertRailDrainsBeforeTheMainTrack(t *testing.T) {
	l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	l = queued(t, l, MainTrack, report("Oceanside"))
	if c, track, ok := l.Next(); !ok || c.ID != "Bonsall" || track != MainTrack {
		t.Fatalf("Next() = %q on %v (ok=%v), want Bonsall on the main track", c.ID, track, ok)
	}
	l = queued(t, l, AlertRail, alert("a1"))
	l = queued(t, l, AlertRail, alert("a2"))
	c, track, ok := l.Next()
	if !ok || c.ID != "a1" || track != AlertRail {
		t.Fatalf("Next() = %q on %v (ok=%v), want a1 on the alert rail", c.ID, track, ok)
	}
}

// TestNoAdmittedCardIsEverSkipped drains a full lineup and asserts that every
// card queued comes back out, in rail-then-main order and in the order it was
// queued within each. This is DR-3's "no admitted card is ever dropped unread"
// at the level where it can be stated as a property.
func TestNoAdmittedCardIsEverSkipped(t *testing.T) {
	l := Lineup{}
	for _, id := range []string{"Bonsall", "Oceanside", "Ramona"} {
		l = queued(t, l, MainTrack, report(id))
	}
	for _, id := range []string{"a1", "a2"} {
		l = queued(t, l, AlertRail, alert(id))
	}
	var read []string
	for range 10 { // bounded: five cards, and a bound that cannot hide a loop
		c, _, ok := l.Next()
		if !ok {
			break
		}
		read = append(read, c.ID)
		next, err := l.Remove(c.ID)
		if err != nil {
			t.Fatalf("Remove(%s): %v", c.ID, err)
		}
		l = next
	}
	want := []string{"a1", "a2", "Bonsall", "Oceanside", "Ramona"}
	if !equal(read, want) {
		t.Errorf("drained %v, want %v", read, want)
	}
}

// TestACardOnTheAirIsNotOfferedAgain, and neither is one that has finished. Next
// offers only a card still on its way to the air — stated positively, because a
// list of states to skip grows a hole every time a state is added.
func TestACardOnTheAirIsNotOfferedAgain(t *testing.T) {
	for s := State(0); s < numStates; s++ {
		l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
		held := l.Cards(MainTrack)[0]
		held.State = s
		if s == OnAir || s == Standby {
			held.Script = Say("the script")
		}
		moved, err := l.Set(held)
		if err != nil {
			continue // Set refuses what the card model refuses; those states are covered above
		}
		_, _, ok := moved.Next()
		if want := s == Admitted || s == Standby; ok != want {
			t.Errorf("with its only card at %v, Next() offered = %v, want %v", s, ok, want)
		}
	}
}

// TestACardOnTheAirRefusesEditsButMayStillLeaveTheAir is DR-6 and DR-24 in one
// place, because they are one rule seen from two sides. The lock is against
// EDITS. It is not against the state machine: a lock that could keep a card on
// the air would wedge the station, which is exactly the failure the Lineup
// exists to make impossible.
func TestACardOnTheAirRefusesEditsButMayStillLeaveTheAir(t *testing.T) {
	l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	held := l.Cards(MainTrack)[0]
	held = at(t, held, Standby).mustText(t)
	held = at(t, held, OnAir)
	l, err := l.Set(held)
	if err != nil {
		t.Fatalf("Set to the air: %v", err)
	}

	edited := held
	edited.Script = Say("different words")
	if _, err := l.Set(edited); err == nil {
		t.Error("the card on the air accepted an edit")
	}
	edited = held
	edited.ReadBy = "Rishi"
	if _, err := l.Set(edited); err == nil {
		t.Error("the voice changed under a card on the air")
	}

	for _, exit := range []State{Done, Discarded} {
		if _, err := l.Set(at(t, held, exit)); err != nil {
			t.Errorf("a card on the air could not leave it for %v: %v", exit, err)
		}
	}
}

// TestACardKeepsItsSlotAndItsOriginForLife. Set exists to move a card's state
// and fill in its words; swapping what a card IS under a live identity would
// change what is read without changing what the lineup says is read.
func TestACardKeepsItsSlotAndItsOriginForLife(t *testing.T) {
	l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	held := l.Cards(MainTrack)[0]

	swapped := held
	swapped.Slot = BreakingAlert
	if _, err := l.Set(swapped); err == nil {
		t.Error("a card changed slot under a live identity")
	}
	swapped = held
	swapped.Origin = FromOperator
	if _, err := l.Set(swapped); err == nil {
		t.Error("a card changed origin under a live identity")
	}
	if _, err := l.Set(at(t, proposed(t, report("Ramona")), Admitted)); err == nil {
		t.Error("Set added a card the lineup was not holding")
	}
	if _, err := l.Remove("Ramona"); err == nil {
		t.Error("Remove took a card the lineup was not holding")
	}
}

// TestSetReplacesACardWithoutMovingIt. A state change must not reorder the
// schedule: a card that jumped its queue on being built would be read out of
// the order the planner decided.
func TestSetReplacesACardWithoutMovingIt(t *testing.T) {
	l := Lineup{}
	for _, id := range []string{"Bonsall", "Oceanside", "Ramona"} {
		l = queued(t, l, MainTrack, report(id))
	}
	middle := at(t, l.Cards(MainTrack)[1], Standby)
	moved, err := l.Set(middle)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := []string{"Bonsall", "Oceanside", "Ramona"}
	if got := ids(moved.Cards(MainTrack)); !equal(got, want) {
		t.Errorf("track reads %v, want %v", got, want)
	}
	if got := moved.Cards(MainTrack)[1].State; got != Standby {
		t.Errorf("the replaced card is at %v, want %v", got, Standby)
	}
}

// TestAReaderCannotReachTheLineupsOwnStorage is DR-1's single writer defended
// where the compiler cannot help: the tracks are unexported, so nothing outside
// this package can assign to them, but a slice handed out would alias the same
// backing array and make every reader a second writer.
func TestAReaderCannotReachTheLineupsOwnStorage(t *testing.T) {
	l := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	got := l.Cards(MainTrack)
	got[0].ID = "vandalised"
	got[0].State = Done
	if held := l.Cards(MainTrack)[0]; held.ID != "Bonsall" || held.State != Admitted {
		t.Errorf("the lineup now holds %q at %v; a reader wrote into it", held.ID, held.State)
	}
	if l.Cards(numTracks) != nil {
		t.Error("a track outside the registry returned cards")
	}
}

// TestTwoLineupsBuiltFromOneNeverShareStorage. A Lineup is a value, and the
// pump holds more than one at a time — Step returns a new Director while the
// old one is still on the stack. Remove leaves spare capacity behind the
// length, so a Queue that appended in place would write the second lineup's
// card into the first one's array. That is a schedule silently rewritten.
func TestTwoLineupsBuiltFromOneNeverShareStorage(t *testing.T) {
	base := Lineup{}
	for _, id := range []string{"Bonsall", "Oceanside", "Ramona"} {
		base = queued(t, base, MainTrack, report(id))
	}
	base, err := base.Remove("Ramona") // leaves capacity behind the length
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	one := queued(t, base, MainTrack, report("Fallbrook"))
	two := queued(t, base, MainTrack, report("Vista"))

	if got, want := ids(one.Cards(MainTrack)), []string{"Bonsall", "Oceanside", "Fallbrook"}; !equal(got, want) {
		t.Errorf("the first lineup reads %v, want %v", got, want)
	}
	if got, want := ids(two.Cards(MainTrack)), []string{"Bonsall", "Oceanside", "Vista"}; !equal(got, want) {
		t.Errorf("the second lineup reads %v, want %v", got, want)
	}
	if got, want := ids(base.Cards(MainTrack)), []string{"Bonsall", "Oceanside"}; !equal(got, want) {
		t.Errorf("the lineup they were both built from reads %v, want %v", got, want)
	}
}

// TestSetDoesNotDisturbTheLineupItCameFrom is the same trap on the other
// writer: Set indexes into a track and assigns, which without a copy first
// would edit the receiver in place.
func TestSetDoesNotDisturbTheLineupItCameFrom(t *testing.T) {
	base := queued(t, Lineup{}, MainTrack, report("Bonsall"))
	moved, err := base.Set(at(t, base.Cards(MainTrack)[0], Standby))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := base.Cards(MainTrack)[0].State; got != Admitted {
		t.Errorf("the lineup it came from is now at %v, want %v", got, Admitted)
	}
	if got := moved.Cards(MainTrack)[0].State; got != Standby {
		t.Errorf("the new lineup is at %v, want %v", got, Standby)
	}
}

// TestRemoveDoesNotDisturbTheLineupItCameFrom, and takes exactly one card.
func TestRemoveDoesNotDisturbTheLineupItCameFrom(t *testing.T) {
	base := Lineup{}
	for _, id := range []string{"Bonsall", "Oceanside", "Ramona"} {
		base = queued(t, base, MainTrack, report(id))
	}
	shorter, err := base.Remove("Oceanside")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got, want := ids(shorter.Cards(MainTrack)), []string{"Bonsall", "Ramona"}; !equal(got, want) {
		t.Errorf("after Remove the track reads %v, want %v", got, want)
	}
	if got, want := ids(base.Cards(MainTrack)), []string{"Bonsall", "Oceanside", "Ramona"}; !equal(got, want) {
		t.Errorf("the lineup it came from reads %v, want %v", got, want)
	}
}

// TestAnEmptyLineupOffersNothing. The rail is always present and usually empty
// (DR-3), so this is the ordinary state, not an edge case.
func TestAnEmptyLineupOffersNothing(t *testing.T) {
	var l Lineup
	if _, _, ok := l.Next(); ok {
		t.Error("an empty lineup offered a card")
	}
	for track := Track(0); track < numTracks; track++ {
		if got := l.Cards(track); len(got) != 0 {
			t.Errorf("%v holds %d cards, want none", track, len(got))
		}
	}
	if _, err := l.Remove("Bonsall"); err == nil {
		t.Error("Remove took a card from an empty lineup")
	}
}

// TestTheTwoTracksNameThemselves — the debug line DR-23 asks for is one line per
// card transition, and a track with no name makes that line unreadable.
func TestTheTwoTracksNameThemselves(t *testing.T) {
	if got, want := MainTrack.String(), "MAIN TRACK"; got != want {
		t.Errorf("MainTrack = %q, want %q", got, want)
	}
	if got, want := AlertRail.String(), "ALERT RAIL"; got != want {
		t.Errorf("AlertRail = %q, want %q", got, want)
	}
	for _, bad := range []Track{-1, numTracks, 1 << 20} {
		if got := bad.String(); got != "" {
			t.Errorf("Track(%d).String() = %q, want empty", bad, got)
		}
	}
}

// TestQueueRefusesACardWhoseWordsShouldAlreadyBeOnIt — the OTHER door.
//
// Propose is not the only way in: Queue and Set write to the schedule too, and
// a wordless burst head queued directly reached standby, described no build,
// and stood there for ever with the rail stopped behind it — DR-3's guarantee
// failing from the other side, silently, because an invariant that returns an
// error neither panics nor logs. The rule lives in check now, which every
// write goes through.
func TestQueueRefusesACardWhoseWordsShouldAlreadyBeOnIt(t *testing.T) {
	var l Lineup
	for _, slot := range []Slot{Transition} { // the one structural slot left (T3.10 red team)
		if _, err := l.Queue(AlertRail, Card{ID: "x", Slot: slot, Headline: "h", State: Admitted}); err == nil {
			t.Errorf("a wordless %v was queued; it can never be built and would wedge the rail", slot)
		}
	}
	// The control: the same card carrying its words is queued without complaint.
	next, err := l.Queue(AlertRail, Card{ID: "x", Slot: Transition, Headline: "h", State: Admitted, Script: Say("the words")})
	if err != nil {
		t.Fatalf("a burst head carrying its words was refused: %v", err)
	}
	if got := len(next.Cards(AlertRail)); got != 1 {
		t.Errorf("the rail holds %d cards, want the one that was queued", got)
	}
	// And a report is still queued with no words at all — that is DR-7's other
	// half, and this rule must not have broken it.
	if _, err := l.Queue(MainTrack, Card{ID: "r", Slot: LocationReport, Headline: "h", Subject: "Bonsall", State: Admitted}); err != nil {
		t.Errorf("a report was refused for having no words yet: %v", err)
	}
}
