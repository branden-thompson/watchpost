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
func TestTheOnAirBoundaryIsNotYetOnTheSurface(t *testing.T) {
	b := broadcasterWithOneCard(t)
	b, _ = b.Update(StationMsg{Power: lineup.Running})

	if strings.Contains(stripANSITest(b.View().Content), "does not observe a transmitter") {
		t.Error("FR-5.5's boundary sentence now RENDERS — F-109 has been ruled and fixed. " +
			"Invert this test to assert the sentence is present, and close FR-5.5 in the record.")
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
