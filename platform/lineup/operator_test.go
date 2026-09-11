package lineup

// OPERATOR INTENT (FR-3, P4).
//
// FR-3.3 IS THE RELEASE'S UI-INTEGRITY REQUIREMENT and it names its own trap:
// "`Lineup.Set` refuses to reorder by design, so a promote routed through it
// would silently no-op." The exit it demands is that a mutant making promote
// update display state without reordering is CAUGHT — so every assertion below
// is against what the SCHEDULE WALKS, never against a field a console renders.
//
// "An action must never be shown as taken unless the schedule took it."

import (
	"testing"
	"time"
)

func opBase() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }

// threeQueued is a station holding three reports, in the order they arrived.
func threeQueued(t *testing.T) Director {
	t.Helper()
	d := New(Settings{Max: 5}, opBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	for _, ref := range []string{"one", "two", "three"} {
		d, _ = d.Step(NeedsRead{Ref: ref, Headline: "REPORT FOR " + ref})
	}
	if got := len(d.lineup.Cards(MainTrack)); got != 3 {
		t.Fatalf("the fixture needs three cards; got %d", got)
	}
	return d
}

// order is what the schedule WALKS, which is the only thing FR-3.2 accepts as
// evidence. It asks Next repeatedly rather than reading the slice, because the
// slice is what a naive promote would update while Next kept its old answer.
func order(d Director) []string {
	var out []string
	seen := map[string]bool{}
	for range 10 { // bounded by the main track's ten slots (P10-02)
		c, _, ok := d.lineup.Next()
		if !ok || seen[c.ID] {
			break
		}
		seen[c.ID] = true
		out = append(out, c.ID)
		// Walk past it the way preparation does, without changing its state.
		next, err := d.lineup.Remove(c.ID)
		if err != nil {
			break
		}
		d.lineup = next
	}
	return out
}

func TestAPromoteChangesWhatTheScheduleWalks(t *testing.T) {
	d := threeQueued(t)
	if got := order(d); got[0] != ReadID("one") {
		t.Fatalf("the fixture starts in arrival order; got %v", got)
	}

	d, _ = d.Step(Moved{ID: ReadID("three"), To: 0})

	got := order(d)
	if got[0] != ReadID("three") {
		t.Errorf("a promote must change the order the schedule WALKS, not a display field (FR-3.3); "+
			"Next still offers %q first", got[0])
	}
	if len(got) != 3 {
		t.Errorf("and it moves a card rather than adding or losing one; got %d", len(got))
	}
}

func TestADemoteChangesWhatTheScheduleWalks(t *testing.T) {
	d := threeQueued(t)

	d, _ = d.Step(Moved{ID: ReadID("one"), To: 2})

	got := order(d)
	if got[len(got)-1] != ReadID("one") {
		t.Errorf("a demote sends it back in the running order; got %v", got)
	}
}

// FR-3.5: the operator keeps working while the bed is playing. A paused main
// track holds the SCHEDULE, not the operator.
func TestManagementActionsSurviveAPausedMainTrack(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(CutOver{ToBed: true})
	if d.advances(MainTrack) {
		t.Fatal("the fixture needs the main track paused")
	}

	d, _ = d.Step(Moved{ID: ReadID("three"), To: 0})

	if got := order(d); got[0] != ReadID("three") {
		t.Errorf("promote, demote and drop all stay live against a paused track (FR-3.5); got %v", got)
	}
}

// A CARD ON THE AIR IS NOT IN THE RUNNING ORDER. Moving it would be moving
// something that is already being read, and the schedule would be describing a
// queue position for a card that has left the queue.
func TestTheCardOnTheAirCannotBeMoved(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(Built{ID: ReadID("one"), Script: Say("Currently sixty-one degrees.")})
	if c, ok := d.lineup.OnAir(); !ok || c.ID != ReadID("one") {
		t.Fatal("the fixture needs the first card on the air")
	}
	before := order(d)

	d, _ = d.Step(Moved{ID: ReadID("one"), To: 2})

	if got := order(d); len(got) != len(before) || got[0] != before[0] {
		t.Errorf("a card being READ is not in the running order and must not move; %v became %v", before, got)
	}
	// AND ITS POSITION IS THE ASSERTION HERE, deliberately against the rule
	// FR-3.2 sets for everything else. `Next()` skips an on-air card either
	// way, so a plant that removes the lock is invisible to the walk — and the
	// harm IS positional: the schedule would be describing a queue place for a
	// card that has left the queue.
	if got := d.lineup.Cards(MainTrack); got[0].ID != ReadID("one") {
		t.Errorf("the card on the air stays where it is; the track now leads with %q", got[0].ID)
	}
}

// FR-3.7's destructive half, and its exit names the case that matters.
func TestADroppedCardLeavesTheScheduleAndLandsOnThePile(t *testing.T) {
	d := threeQueued(t)

	d, _ = d.Step(Dropped{ID: ReadID("two")})

	for _, id := range order(d) {
		if id == ReadID("two") {
			t.Error("a dropped card is out of the running order")
		}
	}
	pile := d.lineup.Discarded()
	if len(pile) != 1 || pile[0].ID != ReadID("two") {
		t.Fatalf("and it is on the pile, where the undo can reach it; got %v", pile)
	}
}

// FR-3.7's recovery half: "the undo carries NO TIMER — an expiring undo is a
// hidden clock, and a hidden clock under pressure is a trap."
func TestADroppedCardCanBeRestoredLongAfterwards(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(Dropped{ID: ReadID("two")})

	// An hour later. Nothing about the undo may depend on how long it took the
	// operator to decide.
	d, _ = d.Step(Tick{Now: opBase().Add(time.Hour)})
	d, _ = d.Step(Restored{ID: ReadID("two")})

	found := false
	for _, id := range order(d) {
		if id == ReadID("two") {
			found = true
		}
	}
	if !found {
		t.Error("a considered-but-wrong drop is recoverable, with no clock on it (FR-3.7)")
	}
}

// FR-3.4: "an operator-placed card reports FromOperator." This is the writer
// `Origin.FromOperator` has never had.
func TestARestoredCardIsAttributableToTheOperator(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(Dropped{ID: ReadID("two")})
	d, _ = d.Step(Restored{ID: ReadID("two")})

	for _, c := range d.lineup.Cards(MainTrack) {
		if c.ID != ReadID("two") {
			continue
		}
		if c.Origin != FromOperator {
			t.Errorf("a human put this card back, and the card says so; got origin %v", c.Origin)
		}
		// A NEW CARD, NOT A REVIVED ONE. "REFUSED, DONE and DISCARDED lead
		// nowhere: a card that could be revived is a card that can be read
		// twice." So it re-enters the machine at the beginning, and its words
		// materialise at standby again — the old ones may since have become
		// untrue, which is the whole reason the staleness window exists.
		if c.State != Admitted {
			t.Errorf("it re-enters the schedule as a fresh admission; got state %v", c.State)
		}
		if !c.Script.Empty() {
			t.Error("and with no words: they materialise at standby, against data fetched then")
		}
		return
	}
	t.Fatal("the restored card is not on the track")
}

// THE UNDO IS SPENT WHEN IT IS USED. An entry left on the pile is an undo the
// console keeps offering for a card that is already back — and the second press
// is refused by the schedule's identity check, so the operator sees a control
// that does nothing.
func TestRestoringSpendsThePileEntry(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(Dropped{ID: ReadID("two")})
	d, _ = d.Step(Restored{ID: ReadID("two")})

	if got := d.lineup.Discarded(); len(got) != 0 {
		t.Errorf("the undo is spent; the pile still offers %v", got)
	}
}

// FR-3.7's exit, in its own words: "asserted for an ACTIVE warning
// specifically, which is the case that matters."
//
// AND IT IS THE CASE THAT DECIDES THE LANE. A restored takeover must go back on
// the PRIORITY RAIL, not into the programme — a hazard rescheduled as ordinary
// programming would read behind every weather report ahead of it.
func TestARestoredTakeoverGoesBackOnTheRail(t *testing.T) {
	d := New(Settings{Max: 5}, opBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})
	if len(d.lineup.Cards(AlertRail)) != 1 {
		t.Fatal("the fixture needs a live warning on the rail")
	}

	d, _ = d.Step(Dropped{ID: BurstID("a1")})
	if len(d.lineup.Cards(AlertRail)) != 0 {
		t.Fatal("the operator dropped it")
	}

	d, _ = d.Step(Restored{ID: BurstID("a1")})

	if got := len(d.lineup.Cards(AlertRail)); got != 1 {
		t.Errorf("a restored warning goes back to the PRIORITY rail, or it reads behind every report "+
			"ahead of it; the rail holds %d", got)
	}
	if got := len(d.lineup.Cards(MainTrack)); got != 0 {
		t.Errorf("and not into the programme; the main track holds %d", got)
	}
}

// A CARD BEING READ IS NOT DROPPABLE FROM THE RUNNING ORDER.
//
// Stopping a read in progress is a different act with a different sound, and it
// pairs its own release (DR-24). Taking the card out here would leave the band
// holding a callout for a read that had stopped — which is DR-24's original
// defect, arriving through the operator's control instead of the schedule's.
//
// THE OPERATOR IS NOT REFUSED SOMETHING THEY NEED: the control for "stop
// talking" is the station's, not the running order's.
func TestTheCardOnTheAirCannotBeDropped(t *testing.T) {
	d := threeQueued(t)
	d, _ = d.Step(Built{ID: ReadID("one"), Script: Say("Currently sixty-one degrees.")})
	if c, ok := d.lineup.OnAir(); !ok || c.ID != ReadID("one") {
		t.Fatal("the fixture needs the first card on the air")
	}

	d, _ = d.Step(Dropped{ID: ReadID("one")})

	if c, ok := d.lineup.OnAir(); !ok || c.ID != ReadID("one") {
		t.Error("the card keeps the air: a drop must not silence a read that has already been cued")
	}
	if got := d.lineup.Discarded(); len(got) != 0 {
		t.Errorf("and nothing lands on the pile; got %v", got)
	}
}

// AN UNDO THAT CANNOT BE PERFORMED MUST NOT CONSUME THE UNDO (F-74).
//
// `onRestored` took the card off the pile BEFORE the proposal and the queue
// could fail, and every failure path returned the Director it had already
// mutated. So a restore that could not be completed destroyed the pile entry
// and put nothing back — the operator's one recovery, spent on nothing, with no
// way to tell it had happened.
//
// IT IS REACHED THE ORDINARY WAY. ReadID is a pure function of the ref, so a
// location dropped and then re-queued by the rotation is holding the identity
// the undo wants. That is not an edge case; it is the rotation doing its job.
func TestARestoreTheScheduleRefusesKeepsTheCardOnThePile(t *testing.T) {
	base := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5}, base)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	d, _ = d.Step(Dropped{ID: ReadID("oceanside")})
	if len(d.lineup.Discarded()) != 1 {
		t.Fatalf("the drop must reach the pile; got %d", len(d.lineup.Discarded()))
	}
	// The rotation asks for that location again, and gets the same identity.
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})

	d, fx := d.Step(Restored{ID: ReadID("oceanside")})

	if got := len(d.lineup.Discarded()); got != 1 {
		t.Errorf("a refused restore must leave the undo intact; the pile holds %d", got)
	}
	if len(fx) != 0 {
		t.Errorf("and it did nothing, so it reports nothing; got %d effects", len(fx))
	}
}

// D-44 SUPERSEDED THE OTHER HALF OF THIS ROW. A test stood here that dropped a
// TRANSITION and asserted the pile kept it, because a structural card can never
// be re-proposed. `onDropped` now refuses a structural card outright — the
// operator never saw it, so the drop cannot have meant it — which makes that
// scenario unreachable rather than merely handled. The rule it was protecting is
// pinned by TestARestoreTheScheduleRefusesKeepsTheCardOnThePile above, and the
// refusal itself by TestTheOperatorCannotAddressAStructuralCard.

// D-42 (HUM LEAD, 2026-09-10): "transition cards are NEVER Origin.fromOperator."
//
// TODAY IT HOLDS BY ACCIDENT — the wordless-transition check refuses one before
// the origin is ever looked at, which is a rule held by a DIFFERENT rule. It is
// stated here so it survives the day a transition carries its words through.
func TestATransitionIsNeverTheOperatorsCard(t *testing.T) {
	_, err := Propose(Card{ID: "t2", Slot: Transition, Origin: FromOperator,
		Subject: "hand-off", Headline: "Coming up", Script: Say("and now, the forecast.")})
	if err == nil {
		t.Error("a transition is the Director's own structural card; the operator never originates one")
	}
}
