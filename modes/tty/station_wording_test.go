package tty

// station_wording_test.go — the two sentences FR-5.3 and FR-5.5 are about.
//
// A SUCCESSOR HAS TO CARRY THE PROPERTY, not merely the subject.  `gates.md`'s
// roster reconciliation names one for each of two retired gates, and neither
// substitute holds what its predecessor held:
//
//   - `TestTheOnAirBoundaryIsStatedToTheOperator` (FR-5.5) asserted the frame
//     contains the boundary sentence.  Its claimed successor asserts only
//     "STATION AIR:" and "SHIFT + ENTER".  **No test referenced the sentence at
//     all** — a genuine coverage hole on a safety wording, hidden behind a
//     citation that looked like a retirement.
//   - `TestTheStateIsLegibleWithoutColour` (FR-5.3) asserted the STANDBY state
//     word renders under --ascii.  The successor asserts the label, not the word.
//
// WHY THE WORDING IS SAFETY-CRITICAL AND NOT COSMETIC. `broadcaster.go` says it:
// Watchpost has no radio path — it produces audio, and a transmitter it cannot
// observe puts that over the air.  "An operator who reads a confident ON AIR and
// infers their antenna is radiating has been misled by us", which is why the
// boundary is stated on the surface and not only in a design document (FR-5.5).

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// FR-5.5 IS OPEN, AND THIS RECORDS IT AS A FACT RATHER THAN A HOPE (F-109).
//
// The boundary sentence exists in `stationLine` and CANNOT RENDER: the row that
// would draw it is gated on `statusNote`, which has already replaced it. So the
// requirement has no implementation, and the gate `gates.md` cited as its
// successor asserts something else entirely.
//
// THIS TEST PINS THE DEAD END rather than asserting the feature, because
// restoring the sentence changes a layout the HUM LEAD ruled — two rows of text,
// a band of seven — and that is a ruling, not an edit. When F-109 is ruled, this
// test is what gets inverted.
// FR-5.5 — THE BOUNDARY IS STATED, ON A LINE OF ITS OWN (F-109).
//
// THIS TEST WAS THE TRIPWIRE AND IS NOW THE GATE. It asserted the sentence did
// NOT render — D-153's dead end — and instructed its own inversion on the day
// F-109 was ruled.
//
// THE TRIPWIRE HAD A HOLE WORTH RECORDING: it watched for the literal string
// "does not observe a transmitter", the FULL assigned wording. Every ruling
// since has shortened it, so the tripwire would have stayed green straight
// through its own fix — a gate keyed to one PHRASING of a rule rather than to
// the rule. It reads the ladder itself now, so no rewording slips past it.
//
// AND THE PLACEMENT WAS RULED TWICE. Option C put it on the STATION AIR row;
// measured, it rendered only at 160 cells and above, leaving FR-5.5 dead at
// every width the console is actually drawn at. HUM LEAD, 2026-09-16: "Give
// F109C its own line in the section area then if it creates new defects." A
// line of its own covers EVERY DRAWABLE WIDTH, which is what this pins.
func TestTheOnAirBoundaryIsStatedAtEveryDrawableWidth(t *testing.T) {
	// THE CONSOLE'S FLOOR IS 100 COLUMNS — below it the frame is a refusal, not
	// a console, so there is no station section to carry anything.
	for _, w := range []int{100, 110, 120, 132, 144, 160, 200} {
		b := broadcasterWithOneCard(t)
		b.width = w
		b, _ = b.Update(StationMsg{Power: lineup.Running})
		got := stripANSITest(b.View().Content)

		var said string
		for _, words := range bcAirBoundaries() {
			if strings.Contains(got, words) {
				said = words
				break
			}
		}
		if said == "" {
			t.Errorf("width %d: FR-5.5's boundary is not on the surface.\n"+
				"An operator reading a confident ON AIR may infer their antenna is radiating;\n"+
				"Watchpost produces audio and cannot verify the transmitter that carries it.", w)
			continue
		}
		// A LINE OF ITS OWN, which is the ruling. Sharing the STATION AIR row is
		// what could not be made to fit.
		for _, line := range strings.Split(got, "\n") {
			if strings.Contains(line, said) {
				if strings.Contains(line, "STATION AIR:") {
					t.Errorf("width %d: the boundary is back on the STATION AIR row, not its own line", w)
				}
				break
			}
		}
	}
}

// AND OFF THE AIR THERE IS NO BOUNDARY TO STATE.
//
// STOPPED AND STANDBY ASSERT NOTHING ABOUT A TRANSMITTER, so a standing line
// there would be the prose row D-107 deliberately removed — "it explained a
// state the row above already names". This one qualifies a claim the row above
// MAKES, which is why it belongs only where the claim is.
func TestTheBoundaryIsAbsentWhenTheStationIsNotOnAir(t *testing.T) {
	for _, power := range []lineup.Power{lineup.OffAir, lineup.Stopped} {
		b := broadcasterWithOneCard(t)
		b.width = 150
		b, _ = b.Update(StationMsg{Power: power})
		got := stripANSITest(b.View().Content)
		for _, words := range bcAirBoundaries() {
			if strings.Contains(got, words) {
				t.Errorf("power %v: the console states an ON AIR boundary while it is not on the air: %q", power, words)
			}
		}
	}
}

// AND EVERY RUNG NEGATES THE INFERENCE THAT MATTERS.
//
// THE LADDER SHORTENS AND MUST NOT SOFTEN. "audio only" says what Watchpost does
// produce and may be dropped for room; the half about the transmitter is the
// whole safety content, and a rung without it reads as reassurance while
// claiming to be the boundary.
func TestEveryBoundaryRungStillDeniesTheTransmitter(t *testing.T) {
	if len(bcAirBoundaries()) == 0 {
		t.Fatal("the ladder is empty; FR-5.5 has no words at any width")
	}
	for _, words := range bcAirBoundaries() {
		if !strings.Contains(words, "transmitter") {
			t.Errorf("rung %q does not mention the transmitter — the inference it exists to deny "+
				"is that the operator's antenna is radiating", words)
		}
	}
}

// AND IT IS NEVER SHOWN HALF-SAID.
//
// A SAFETY SENTENCE CUT MID-WORD IS A DIFFERENT CLAIM, NOT A SHORTER ONE.
// Measured at 120 cells, dropping the fit check renders "· audio on" — complete,
// reassuring, and the opposite of what it was cut from.
func TestTheOnAirBoundaryFallsAwayRatherThanBeingCut(t *testing.T) {
	for _, w := range []int{100, 110, 120, 132, 144, 160, 200} {
		b := broadcasterWithOneCard(t)
		b.width = w
		b, _ = b.Update(StationMsg{Power: lineup.Running})
		got := stripANSITest(b.View().Content)
		whole := false
		for _, words := range bcAirBoundaries() {
			if strings.Contains(got, words) {
				whole = true
				break
			}
		}
		if !whole {
			t.Errorf("width %d: no rung renders whole", w)
			continue
		}
		// AND NO LONGER RUNG APPEARS HALF-SAID. A rung that fits is taken entire;
		// a longer one that does not fit must be absent, never truncated to a
		// prefix that reads as a complete thought.
		for _, words := range bcAirBoundaries() {
			if strings.Contains(got, words) {
				break // this is the rung in force; longer ones were tested above it
			}
			for n := 12; n < len(words); n++ {
				if strings.Contains(got, words[:n]) {
					t.Errorf("width %d: a longer rung is cut to %q — a half-said safety sentence", w, words[:n])
					break
				}
			}
		}
	}

}

// FR-5.3 — the WORDS carry the state, so a reader with no colour still reads it.
func TestEveryStationStateIsLegibleWithoutColour(t *testing.T) {
	for _, tc := range []struct {
		power lineup.Power
		want  string
	}{
		{lineup.Running, "ON AIR"},
		{lineup.OffAir, "STANDBY"},
		{lineup.Stopped, "STOPPED"},
	} {
		b := broadcasterWithOneCard(t)
		b.ascii = true // --ascii: no colour anywhere
		b, _ = b.Update(StationMsg{Power: tc.power})
		got := stripANSITest(b.View().Content)
		if !strings.Contains(got, tc.want) {
			t.Errorf("power %v: the frame never says %q under --ascii; a colour alone fails "+
				"any terminal without one (FR-5.3)", tc.power, tc.want)
		}
	}
	// AND THE STANDBY STATE NAMES ITS CONSEQUENCE, which is the half the
	// superseded gate actually asserted: STANDBY is DEAD AIR, not merely idle.
	b := broadcasterWithOneCard(t)
	b.ascii = true
	b, _ = b.Update(StationMsg{Power: lineup.OffAir})
	if got := stripANSITest(b.View().Content); !strings.Contains(got, "DEAD AIR") {
		t.Error("STANDBY does not say it is dead air; the operator reads a state without its consequence")
	}
}
