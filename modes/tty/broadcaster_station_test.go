package tty

// broadcaster_station_test.go — the station line, Variant C (D-21).
//
// VARIANT C WAS RATIFIED AND IS ALREADY BUILT: "the station state as a labelled
// field, the transition in parentheses." It SUPERSEDES the reference mock's
// centred `{ *** ON AIR | BROADCASTING *** }` banner and the separate
// `[ SHIFT + ENTER ] GO TO STANDBY` control row — one line carries both.
//
// WHAT WAS WRONG WAS THE DRAWING, NOT THE WORDING. The transition sat at a
// HARD-CODED COLUMN, padded with literal spaces, so it only landed correctly at
// one width — the geometry the HUM LEAD ruled out.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

func stationAt(t *testing.T, w int, p lineup.Power) []string {
	t.Helper()
	b := NewBroadcaster()
	b.width, b.height, b.ascii = w, 74, true
	b.power = p
	return b.stationLine()
}

// EVERY POWER STATE HAS A LINE, swept from the registry rather than listed —
// a state added without one would draw nothing where the operator looks first.
func TestEveryStationStateSaysWhatItIs(t *testing.T) {
	for _, p := range []lineup.Power{lineup.Stopped, lineup.Running, lineup.OffAir} {
		got := stationAt(t, 150, p)
		if len(got) == 0 || strings.TrimSpace(got[0]) == "" {
			t.Fatalf("power %v draws no station line", p)
		}
		if !strings.Contains(got[0], "STATION:") {
			t.Errorf("power %v: Variant C is a LABELLED FIELD; got %q", p, got[0])
		}
		if !strings.Contains(got[0], "SHIFT + ENTER") {
			t.Errorf("power %v: and the transition rides on the same line; got %q", p, got[0])
		}
	}
}

// THE TRANSITION IS RIGHT-ANCHORED AT EVERY WIDTH. It was padded to a fixed
// column, which lands correctly at exactly one terminal size and nowhere else.
func TestTheTransitionHintIsAnchoredToTheRightEdge(t *testing.T) {
	for _, w := range []int{100, 110, 120, 130, 150} {
		b := NewBroadcaster()
		b.width, b.height, b.ascii = w, 74, true
		b.power = lineup.Running
		got := b.stationLine()[0]
		// IT FILLS THE LANE, which is the frame less its gutter — the same
		// number every lane row is drawn against, from `laneWidth` (D-51).
		if c, want := utf8.RuneCountInString(got), b.laneWidth(); c != want {
			t.Errorf("width %d: the station line is %d cells, want the lane's %d\n%q", w, c, want, got)
			continue
		}
		if strings.HasSuffix(got, "  ") {
			t.Errorf("width %d: the transition is not at the right edge:\n%q", w, got)
		}
		if !strings.HasSuffix(strings.TrimRight(got, " "), ")") {
			t.Errorf("width %d: the transition ends the line; got %q", w, got)
		}
	}
}

// AND IT SAYS WHERE IT WOULD GO, which is the whole of Variant C's parenthetical:
// ON AIR offers STANDBY, and the two stopped states offer ON AIR.
func TestTheTransitionNamesTheStateItWouldReach(t *testing.T) {
	live := stationAt(t, 150, lineup.Running)[0]
	if !strings.Contains(live, "STANDBY") {
		t.Errorf("a live station's control offers STANDBY; got %q", live)
	}
	for _, p := range []lineup.Power{lineup.Stopped, lineup.OffAir} {
		got := stationAt(t, 150, p)[0]
		if !strings.Contains(got, "ON AIR") {
			t.Errorf("power %v: the control offers ON AIR; got %q", p, got)
		}
	}
}
