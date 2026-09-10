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
		if c, want := utf8.RuneCountInString(got), b.sectionWidth(); c != want {
			t.Errorf("width %d: the station line is %d cells, want the section's %d\n%q", w, c, want, got)
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
	// THE THIRD ROW: the section is STATION, BED, then the status row that
	// carries the level (D-62). The gain moved down when the bed came in.
	got := stripANSITest(b.stationLine()[2])

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
			if c, want := utf8.RuneCountInString(stripANSITest(line)), b.sectionWidth(); c != want {
				t.Errorf("width %d row %d: %d cells, want the section's %d\n%q", w, i, c, want, stripANSITest(line))
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

// D-62: THE BED LIVES IN THE BROADCAST SECTION.
//
//	"we talked about this earlier in moving the bed card into the 'broadcast
//	section' — this was because the bed was yet another conveyer of the AIR
//	STATE and we wanted to consolidate those … this way the ON AIR / STANDBY is
//	all in one section, and the user doesn't have to look to different parts of
//	the UI to determine what is and is not ON AIR."
//
// THREE THINGS SAID WHERE THE STATION IS SAID: the programme's state, the bed's
// state, and the level. The bed was a separate card at the bottom of the frame —
// a second place to look for the same question.
func TestTheBedRidesInTheStationSection(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	rows := b.stationLine()
	if len(rows) != 3 {
		t.Fatalf("the section carries the station, the bed and the status row; got %d", len(rows))
	}
	if !strings.Contains(rows[1], "BED:") {
		t.Errorf("the bed's row is the second:\n%q", rows[1])
	}
	// THE KEY IS A CHIP, so this asks the chip renderer — "[ B ]" is only what
	// the mock draws around it, and only what it falls back to without colour.
	if !strings.Contains(rows[1], chipFor("B")) {
		t.Errorf("and it names the key that reaches it:\n%q", rows[1])
	}
}

// THE SELECTOR IS ON THE BED'S ROW, which is what F-77 was open about — it went
// missing when Variant C absorbed the control line, and the ruling put it here
// rather than restoring a separate card.
func TestTheBedRowCarriesItsSelector(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	got := stripANSITest(b.stationLine()[1])
	// ASKED OF THE CHIP RENDERER, which also names the arrows in WORDS under
	// --ascii: a terminal that cannot draw them still gets a usable control,
	// and the test does not have to know which form it got.
	for _, want := range []string{chipFor("←"), chipFor("→")} {
		if !strings.Contains(got, want) {
			t.Errorf("the bed's row carries the selector; %q is missing from:\n%q", want, got)
		}
	}
}

// AND ITS OWN STATE, which is the third carrier being consolidated: the station
// says ON AIR or STANDBY, the bed says ACTIVE or INACTIVE, and both are read in
// one glance.
func TestTheBedRowSaysWhetherItIsCarrying(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.stationLine()[1])
	if !strings.Contains(got, "INACTIVE") && !strings.Contains(got, "ACTIVE") {
		t.Errorf("the bed's row says whether it is carrying:\n%q", got)
	}
}

// THE LABELS' VALUES LINE UP. "STATION:" and "[ B ] BED:" are different
// lengths, and a section whose two values began in different columns would read
// as two unrelated rows rather than as one region saying one thing.
//
// MEASURED AT THE VALUE COLUMN ITSELF, not at the value's TEXT: the bed's value
// opens with a key cap, so its arrow sits one cell in from where its value
// starts — which is what a first version of this compared, and failed on.
func TestTheSectionsLabelsShareAValueColumn(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	for i, row := range b.stationLine()[:2] {
		r := []rune(stripANSITest(row))
		if len(r) <= bcLabelCells {
			t.Fatalf("row %d is shorter than the label column", i)
		}
		if r[bcLabelCells] == ' ' {
			t.Errorf("row %d has no value at the shared column %d:\n%q", i, bcLabelCells, string(r[:40]))
		}
		if r[bcLabelCells-1] != ' ' {
			t.Errorf("row %d has no air before its value:\n%q", i, string(r[:40]))
		}
	}
}

// THE VALUE COLUMN HOLDS WITH COLOUR ON, which is the mode the operator runs in
// and the mode no other test here uses.
//
// A CHIP CARRIES SGR. With colour off `KeyCap("B")` is "[B]" — three bytes and
// three cells, so a byte count and a cell count agree and a `len()` bug is
// INVISIBLE. With colour on it is " B " wrapped in escape codes: the bytes
// roughly triple and the cells do not, so a label padded by bytes stops padding
// at all and its value slides eight columns left.
//
// COMPARED BETWEEN THE ROWS, not against a fixed column: a chip's FIRST CELL IS
// A SPACE by design, so the bed's value legitimately begins one cell later than
// the station's. What must not happen is the two drifting apart.
func TestTheValueColumnHoldsWithColourOn(t *testing.T) {
	painted(t)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b.power = lineup.Running
	rows := b.stationLine()

	start := func(row string) int {
		r := []rune(stripANSITest(row))
		for i := bcLabelCells - 1; i < len(r); i++ {
			if r[i] != ' ' {
				return i
			}
		}
		return -1
	}
	a, c := start(rows[0]), start(rows[1])
	if a < 0 || c < 0 {
		t.Fatalf("both rows must carry a value:\n%q\n%q", rows[0], rows[1])
	}
	if d := a - c; d > 1 || d < -1 {
		t.Errorf("the values start at %d and %d with colour on — the label column was padded by BYTES, "+
			"and a chip's escape codes are not cells", a, c)
	}
	// AND EVERY ROW IS STILL THE LANE'S WIDTH.
	for i, row := range rows {
		if got, want := utf8.RuneCountInString(stripANSITest(row)), b.sectionWidth(); got != want {
			t.Errorf("row %d is %d cells with colour on, want the section's %d", i, got, want)
		}
	}
}
