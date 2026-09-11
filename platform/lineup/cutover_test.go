package lineup

// The operator moves the programme between the lanes (D-11, FR-4.2, D-32).
//
// MAIN AND BED ARE MUTUALLY EXCLUSIVE, and until now that was true only by
// accident of the engine having one source. The Director never asked whether
// the bed held the programme before putting a card on the air, so the rule the
// product model states was enforced nowhere the model could see.
//
// THE PAUSE IS NOT A SECOND FLAG. FR-4.2 makes it the CONSEQUENCE of the bed
// holding the programme — "the operator can cut the main track over to the bed;
// the main track then pauses" — so one piece of state carries both and there is
// no pair that can disagree (D-1).

import (
	"testing"
	"time"
)

func cutoverBase() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }

// running is a Director with the programme on and one report queued.
func running(t *testing.T) Director {
	t.Helper()
	d := New(Settings{Max: 5}, cutoverBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	if len(d.lineup.Cards(MainTrack)) != 1 {
		t.Fatalf("the fixture needs one main-track card; got %d", len(d.lineup.Cards(MainTrack)))
	}
	return d
}

// PARITY FIRST, and it is the claim the whole batch opens with (T3.2a): with
// nothing emitting a cut-over, every existing path behaves exactly as it did.
func TestNothingCarriesTheBedUntilSomethingSaysSo(t *testing.T) {
	d := running(t)
	if d.bed.carries {
		t.Error("a station nobody cut over is not on the bed")
	}
	if !d.advances(MainTrack) {
		t.Error("and its main track advances, exactly as it did before this state existed")
	}
}

func TestACutOverToTheBedPausesTheMainTrack(t *testing.T) {
	d := running(t)

	d, _ = d.Step(CutOver{ToBed: true})

	if !d.bed.carries {
		t.Fatal("the operator moved the programme to the bed")
	}
	if d.advances(MainTrack) {
		t.Error("the main track PAUSES while the bed carries the programme (FR-4.2) — " +
			"main and bed are mutually exclusive, and a card taking the air here would play over a relay")
	}
	// NO CARD IS LOST. A pause holds the schedule; it does not empty it, and
	// the operator's management actions stay live on what is held.
	if got := len(d.lineup.Cards(MainTrack)); got != 1 {
		t.Errorf("a pause holds the lineup rather than discarding it; got %d cards", got)
	}
}

// THE RAIL IS NOT PAUSED, and that asymmetry is the safety property (FR-2.2):
// "the priority track always drains first, INCLUDING while the bed is playing."
func TestTheRailStillDrainsWhileTheBedCarries(t *testing.T) {
	d := running(t)
	d, _ = d.Step(CutOver{ToBed: true})

	if !d.advances(AlertRail) {
		t.Error("a hazard reads over the bed; pausing the programme must never hold the rail")
	}
}

func TestCuttingBackResumesTheMainTrackWithItsCardsIntact(t *testing.T) {
	d := running(t)
	d, _ = d.Step(CutOver{ToBed: true})

	d, _ = d.Step(CutOver{ToBed: false})

	if d.bed.carries {
		t.Fatal("the operator moved the programme back")
	}
	if !d.advances(MainTrack) {
		t.Error("the main track resumes")
	}
	if got := len(d.lineup.Cards(MainTrack)); got != 1 {
		t.Errorf("and the card it was holding is still there; got %d", got)
	}
}

// FR-5.6's exit, asserted: "the two are separately representable and separately
// asserted." A paused main track is NOT standby, and the difference is the rail.
func TestAPausedMainTrackIsNotStandby(t *testing.T) {
	paused := running(t)
	paused, _ = paused.Step(CutOver{ToBed: true})

	standby := running(t)
	standby, _ = standby.Step(Powered{To: OffAir})

	if paused.Power() != Running {
		t.Errorf("a paused main track leaves the station ON: the bed is broadcasting; got %v", paused.Power())
	}
	if paused.advances(MainTrack) || standby.advances(MainTrack) {
		t.Error("neither advances the main track — that is the half they share")
	}
	// AND THIS IS THE HALF THAT MAKES THEM DIFFERENT. OffAir is dead air and
	// holds every track including the rail; a paused main track is a station
	// that is very much broadcasting, and a hazard still reads.
	if !paused.advances(AlertRail) {
		t.Error("a paused main track still drains the rail — the station is ON AIR, on the bed")
	}
	if standby.advances(AlertRail) {
		t.Error("STANDBY holds the rail too; that is what makes it dead air rather than a pause")
	}
}

// A REPEATED CUT-OVER IS NOT A SECOND ONE, which is the rule onPowered already
// states for the power: "a repeated command is not a second event."
func TestARepeatedCutOverChangesNothing(t *testing.T) {
	d := running(t)
	d, _ = d.Step(CutOver{ToBed: true})
	first := d.bed

	d, fx := d.Step(CutOver{ToBed: true})

	if d.bed != first {
		t.Error("asking for what is already true must not move the bed")
	}
	// AND IT MUST DO NO WORK, which is the half that has teeth. A plant
	// deleting the guard left the bed identical and re-settled anyway, so the
	// console was told the schedule had changed when it had not — the same
	// defect setMode's own comment names: "a station that re-announced itself
	// on every relay change would be telling it something that had not
	// changed."
	//
	// FOURTH INSTANCE THIS SESSION of a gate watching the STATE and not the
	// WORK, and the tell was the same every time: the step returns something
	// and the test does not ask for it.
	if len(fx) != 0 {
		t.Errorf("a repeated command is not a second event, and must publish nothing; got %d effect(s): %v",
			len(fx), fx)
	}
}
