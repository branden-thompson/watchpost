package tty

// station_wording_test.go — the two sentences FR-5.3 and FR-5.5 are about.
//
// FOUND BY RED TEAM (round 2) AT BUILD EXIT, 0.16.0.  `gates.md`'s roster
// reconciliation named successors for two retired gates that do NOT carry their
// property:
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
// FR-5.5 — THE BOUNDARY IS STATED WHERE THE OPERATOR READS THE STATE (F-109).
//
// THIS TEST WAS THE TRIPWIRE AND IS NOW THE GATE. It asserted the sentence did
// NOT render — D-153's dead end — and instructed its own inversion on the day
// F-109 was ruled. HUM LEAD, 2026-09-16: option C, "on the STATION AIR row
// between the state and GAIN, in a shortened wording".
//
// THE TRIPWIRE HAD A HOLE WORTH RECORDING: it watched for the literal string
// "does not observe a transmitter", which is the FULL assigned wording. Ruling C
// is a SHORTENED wording, so the tripwire would have stayed green straight
// through its own fix — a gate keyed to one PHRASING of a rule rather than to
// the rule. It reads the ladder itself now, so no wording change slips past it.
func TestTheOnAirBoundaryIsStatedOnTheStationRow(t *testing.T) {
	b := broadcasterWithOneCard(t)
	b.width = 144 // the console's reference width — the rail's own column
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	got := stripANSITest(b.View().Content)

	var said string
	for _, words := range bcAirBoundaries {
		if strings.Contains(got, words) {
			said = words
			break
		}
	}
	if said == "" {
		t.Fatalf("FR-5.5's boundary is not on the surface at the reference width.\n"+
			"An operator reading a confident ON AIR may infer their antenna is radiating;\n"+
			"Watchpost produces audio and cannot verify the transmitter that carries it.\nFrame:\n%s", got)
	}
	// IT RIDES THE STATE'S OWN ROW, not some other part of the frame. A boundary
	// three rows from the claim it qualifies is one the operator reads
	// separately, or not at all.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "STATION AIR:") {
			if !strings.Contains(line, said) {
				t.Errorf("the boundary renders somewhere, but not on the STATION AIR row (ruling C):\n  %s", line)
			}
			return
		}
	}
	t.Error("no STATION AIR row in the frame")
}

// AND EVERY RUNG NEGATES THE INFERENCE THAT MATTERS.
//
// THE LADDER SHORTENS AND MUST NOT SOFTEN. "audio only" says what Watchpost does
// produce and may be dropped for room; the half about the transmitter is the
// whole safety content, and a rung without it reads as reassurance while
// claiming to be the boundary.
func TestEveryBoundaryRungStillDeniesTheTransmitter(t *testing.T) {
	if len(bcAirBoundaries) == 0 {
		t.Fatal("the ladder is empty; FR-5.5 has no words at any width")
	}
	for _, words := range bcAirBoundaries {
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
	for _, w := range []int{80, 100, 120, 132, 144, 160, 200} {
		b := broadcasterWithOneCard(t)
		b.width = w
		b, _ = b.Update(StationMsg{Power: lineup.Running})
		// THE STATION AIR ROW ONLY. The frame says "(no transmitter set)" two rows
		// down — the operator's CONFIGURATION, a different fact — and a whole-frame
		// scan reads that as a cut rung. Scoping it here is also what makes this
		// measure ruling C's placement rather than the frame's vocabulary.
		var row string
		for _, line := range strings.Split(stripANSITest(b.View().Content), "\n") {
			if strings.Contains(line, "STATION AIR:") {
				row = line
				break
			}
		}
		if row == "" {
			continue // below the floor the console draws a notice instead
		}
		whole := false
		for _, words := range bcAirBoundaries {
			if strings.Contains(row, words) {
				whole = true
			}
		}
		if whole {
			continue
		}
		// NOTHING WHOLE IS PRESENT, so no PREFIX of any rung may be either.
		for _, words := range bcAirBoundaries {
			for n := 8; n < len(words); n++ {
				if strings.Contains(row, words[:n]) {
					t.Errorf("width %d: the boundary is cut to %q — a half-said safety sentence\n  %s",
						w, words[:n], row)
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
