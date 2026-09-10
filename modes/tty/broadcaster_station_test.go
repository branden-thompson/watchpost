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
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
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

// GAIN IS OBSERVER'S VOL CONTROL, WEARING THE STATION'S WORD FOR IT (HUM LEAD,
// 2026-09-10: "gain is the VOL control in Observer").
//
// SO IT IS THE SAME CONTROL, NOT A SECOND ONE (D-56). Observer's `volControl`
// already draws the bar, already steps at the tens, already blinks the chip on a
// press, and already knows the floor and the ceiling. A second bar here would be
// a second place for the level to be drawn — and two bars disagreeing about how
// loud the station is would be worse than either.
func TestTheStationLineCarriesTheGainControl(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	b.gain = 100
	got := stripANSITest(b.stationLine()[1])

	if !strings.Contains(got, "GAIN") {
		t.Errorf("the station's word for it is GAIN:\n%q", got)
	}
	// IT RIDES ON THE SECOND ROW, beside the explanatory text — the HUM LEAD's
	// own layout, and the row Variant C left free when it absorbed the control.
	if !strings.Contains(got, "audio out of this program") {
		t.Errorf("the second row still says what ON AIR means:\n%q", got)
	}
	if !strings.HasSuffix(strings.TrimRight(got, " "), "100") {
		t.Errorf("the level ends the row, right-anchored:\n%q", got)
	}
}

func TestTheGainControlReflows(t *testing.T) {
	for _, w := range []int{100, 120, 150} {
		b := NewBroadcaster()
		b.width, b.height, b.ascii = w, 74, true
		b.power = lineup.Running
		b.gain = 55
		for i, line := range b.stationLine() {
			if c, want := utf8.RuneCountInString(stripANSITest(line)), b.laneWidth(); c != want {
				t.Errorf("width %d row %d: %d cells, want the lane's %d\n%q", w, i, c, want, stripANSITest(line))
			}
		}
	}
}

// THE STATION BAR IS A SECTION, NOT A ROW (HUM LEAD, 2026-09-10).
//
//	"that station bar should be treated as a section/box — we're going to apply
//	a background color to it based on its state (RED for ON-AIR / GREY for
//	STANDBY) — so ensuring these are sectioned is important so we're not
//	painting color row by row / col by col manually."
//
// So it goes through `render.Opts.Block`, which pads every line to the width and
// paints the whole region as ONE: it also RE-ARMS the tone at inner SGR resets,
// which is what stops a background tearing where a chip or a tinted level sits
// mid-line — and the gain bar puts three tinted runs inside this very section.
//
// THE PALETTE IS NOT CHOSEN HERE. Colour is the HUM LEAD's own pass; what this
// pins is that ONE call paints the section, so that pass is a token rather than
// a sweep through every row.

// painted turns colour ON for the duration of a test: `Block` passes content
// through untinted when colour is off, which is the default in tests — so a
// section's painting cannot be asserted without it.
func painted(t *testing.T) {
	t.Helper()
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
}

func TestTheStationBarIsPaintedAsOneSection(t *testing.T) {
	painted(t)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b.power, b.gain = lineup.Running, 55
	const fg, bg = "97", "48;5;52"

	got := strings.Split(b.stationSection(b.opts(), fg, bg), "\n")
	if len(got) < 4 {
		t.Fatalf("the section carries its own breathing room above and below; got %d rows", len(got))
	}
	for i, row := range got {
		if !strings.Contains(row, bg) {
			t.Errorf("row %d is not painted with the section's background:\n%q", i, row)
		}
		if !strings.HasSuffix(row, "\x1b[0m") {
			t.Errorf("row %d does not close its tone, so the paint bleeds past the section:\n%q", i, row)
		}
	}
}

// AND THE TONE SURVIVES THE TINTED RUNS INSIDE IT. The gain bar tints its
// filled cells and its level; each of those closes with a reset, and a reset
// mid-line would end the section's background from that column on.
func TestTheSectionsBackgroundSurvivesTheGainBar(t *testing.T) {
	painted(t)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b.power, b.gain = lineup.Running, 55
	const bg = "48;5;52"

	rows := strings.Split(b.stationSection(b.opts(), "97", bg), "\n")
	gainRow := ""
	for _, r := range rows {
		if strings.Contains(stripANSITest(r), "GAIN") {
			gainRow = r
		}
	}
	if gainRow == "" {
		t.Fatal("the gain row must be in the section")
	}
	// A bare reset anywhere but the very end would drop the background for the
	// rest of the row.
	if i := strings.Index(gainRow, "\x1b[0m"); i >= 0 && i != len(gainRow)-len("\x1b[0m") {
		t.Errorf("a bare reset at %d tears the section's background mid-row:\n%q", i, gainRow)
	}
}
