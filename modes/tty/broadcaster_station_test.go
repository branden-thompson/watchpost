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
	"github.com/branden-thompson/watchpost/platform/render"
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
		if len(got) < 2 || strings.TrimSpace(got[0]) == "" {
			t.Fatalf("power %v draws no station line", p)
		}
		if !strings.Contains(got[0], "STATION AIR:") {
			t.Errorf("power %v: Variant C is a LABELLED FIELD; got %q", p, got[0])
		}
		// AND THE TRANSITION RIDES THE SECOND ROW NOW (D-107). The reference puts
		// the GAIN bar at the right of the state's own row, so the control that
		// changes the state moved down beside the transmitter it belongs with.
		if !strings.Contains(got[1], "SHIFT + ENTER") {
			t.Errorf("power %v: the transition has nowhere to be; got %q", p, got[1])
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
		got := b.stationLine()[1]
		// IT FILLS THE BAND'S TEXT COLUMN — the terminal less the inset it keeps
		// on EACH side (D-80). It asked `sectionWidth()`, which is the width of
		// a region inside the frame's WALLS, and this band has had colour for
		// its edge since D-70: three cells too wide, and the right-hand inset
		// had nowhere to go.
		if c, want := utf8.RuneCountInString(got), b.bandWidth(); c != want {
			t.Errorf("width %d: the station line is %d cells, want the band's %d\n%q", w, c, want, got)
			continue
		}
		if !strings.HasSuffix(strings.TrimRight(got, " "), ")") {
			t.Errorf("width %d: the transition ends the line; got %q", w, got)
		}
	}
}

// AND IT SAYS WHERE IT WOULD GO, which is the whole of Variant C's parenthetical:
// ON AIR offers STANDBY, and the two stopped states offer ON AIR.
func TestTheTransitionNamesTheStateItWouldReach(t *testing.T) {
	if live := stationAt(t, 150, lineup.Running)[1]; !strings.Contains(live, "STANDBY") {
		t.Errorf("a live station's control offers STANDBY; got %q", live)
	}
	for _, p := range []lineup.Power{lineup.Stopped, lineup.OffAir} {
		if got := stationAt(t, 150, p)[1]; !strings.Contains(got, "ON AIR") {
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
//
// IT RIDES THE STATE'S OWN ROW (D-107), where the reference draws it: how loud
// the station is and whether it is on the air are one question asked twice, and
// the operator checks them together.
func TestTheStationLineCarriesTheGainControl(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	b.gain = 100
	got := stripANSITest(b.stationLine()[0])

	if !strings.Contains(got, "GAIN") {
		t.Errorf("the station's word for it is GAIN:\n%q", got)
	}
	if !strings.Contains(got, "ON AIR") {
		t.Errorf("and it shares the row with the state:\n%q", got)
	}
	if !strings.HasSuffix(strings.TrimRight(got, " "), "100") {
		t.Errorf("the level ends the row, right-anchored:\n%q", got)
	}
}

// THE BED IS DRAWN ONCE, AND THE SECTION HOLDS IT (D-107).
//
// HUM LEAD, UAT 2026-09-12: "the LIVE NOW / BED Table is also supposed to be
// INSIDE the station playing section … the BED is now duplicated in the UI -
// which is confusing - this data is also tied DIRECTLY to the ON AIR state - so
// it should all be in 1 section."
//
// IT HAD A ROW IN THE STATION LINES *AND* A ROW IN THE AIR BOX, each with its own
// selector and its own state word — two controls for one bed, which an operator
// has to test to tell apart.
func TestTheBedIsDrawnOnceInsideTheStationSection(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	for _, r := range b.stationLine() { // bounded by the section (P10-02)
		if strings.Contains(stripANSITest(r), "BED") {
			t.Errorf("the station lines still draw the bed; the air box owns it:\n%q", r)
		}
	}
	sec := stripANSITest(b.stationSection(b.opts(), "", ""))
	for _, want := range []string{"STATION AIR:", "BROADCASTING FROM:", "LIVE NOW", "RELAY BED"} {
		if !strings.Contains(sec, want) {
			t.Errorf("the station section is missing %q:\n%s", want, sec)
		}
	}
	// AND THE WORD APPEARS ONCE IN THE WHOLE FRAME, which is what "1 section"
	// means: not one row hidden and another shown, but one row.
	if n := strings.Count(stripANSITest(b.View().Content), "RELAY BED"); n != 1 {
		t.Errorf("the frame names the relay bed %d times", n)
	}
}

func TestTheGainControlReflows(t *testing.T) {
	for _, w := range []int{100, 120, 150} {
		b := NewBroadcaster()
		b.width, b.height, b.ascii = w, 74, true
		b.power = lineup.Running
		b.gain = 55
		for i, line := range b.stationLine() {
			if c, want := utf8.RuneCountInString(stripANSITest(line)), b.bandWidth(); c != want {
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
	// TWO TEXT ROWS AND THE AIR BOX SINCE D-107: the station's state with its
	// gain, where it broadcasts from with its transition, and the LIVE NOW /
	// RELAY BED pair under them. The bed's own labelled row and the standing
	// prose went with the duplication.
	//
	// AND A THIRD WHILE RUNNING, RULED 2026-09-16 (F-109): FR-5.5's boundary has
	// a line of its own. It is not the standing prose D-107 removed — that
	// explained a state the row above already names, and this qualifies a claim
	// the row above MAKES. It appears only ON AIR, which is the only state that
	// can mislead, so the band is one row taller in the state that already
	// changes its colour entire.
	if rows := b.stationLine(); len(rows) != 3 {
		t.Fatalf("ON AIR the section is the state, the transmitter and FR-5.5's boundary; got %d rows", len(rows))
	}
	off := b
	off.power = lineup.OffAir
	if rows := off.stationLine(); len(rows) != 2 {
		t.Fatalf("off the air there is no boundary to state; got %d rows", len(rows))
	}
	bed := bedRowOf(t, strings.Split(b.stationSection(b.opts(), "", ""), "\n"))
	// THE KEY IS A CHIP, so this asks the chip renderer — "[ B ]" is only what
	// the mock draws around it, and only what it falls back to without colour.
	if !strings.Contains(bed, chipFor("b")) {
		t.Errorf("and it names the key that reaches it:\n%q", bed)
	}
}

// THE SELECTOR IS ON THE BED'S ROW, which is what F-77 was open about — it went
// missing when Variant C absorbed the control line, and the ruling put it here
// rather than restoring a separate card.
func TestTheBedRowCarriesItsSelector(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	got := stripANSITest(bedRowOf(t, strings.Split(b.stationSection(b.opts(), "", ""), "\n")))
	// ASKED OF THE CHIP RENDERER, which also names the arrows in WORDS under
	// --ascii: a terminal that cannot draw them still gets a usable control,
	// and the test does not have to know which form it got.
	// SHIFTED SINCE D-111, and the chip says so: the bare arrows step the card's
	// PRESENTER now, so a chip reading `←` here would name a key that moves a
	// different control.
	for _, want := range []string{chipFor("⇧←"), chipFor("⇧→")} {
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
	got := stripANSITest(bedRowOf(t, strings.Split(b.stationSection(b.opts(), "", ""), "\n")))
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
		if got, want := utf8.RuneCountInString(stripANSITest(row)), b.bandWidth(); got != want {
			t.Errorf("row %d is %d cells with colour on, want the section's %d", i, got, want)
		}
	}
}

// bedRowOf is the station section's bed row, found by its LABEL.
//
// BY LABEL, NOT BY INDEX. These tests indexed `[1]`, and D-71 put the
// transmitter's identity there — so three of them failed at once for a reason
// none of them was about. A row found by what it SAYS survives the section
// gaining another.
// THE BED'S KEY SURVIVES THE BOX MOVING (D-107).
//
// `withControl` HAD ITS OWN COPY of the air box's body width. When the box moved
// inside the station section one copy followed and the other did not, and what
// fell off the end of the row was the `b` chip — the key that cuts the programme
// to the bed. Two carriers of one number, found the way they always are.
func TestTheBedsCutKeyIsInsideTheBox(t *testing.T) {
	for _, w := range []int{110, 130, 150, 170} {
		b := NewBroadcaster()
		b.width, b.height, b.ascii = w, 74, true
		row := stripANSITest(bedRowOf(t, b.airBox()))
		if !strings.Contains(row, chipFor("b")) {
			t.Errorf("width %d: the bed's row has lost its cut key:\n%q", w, row)
		}
		// AND THE ROW STILL CLOSES ON THE BOX'S OWN RAIL, so the chip is inside it
		// rather than having pushed the wall out.
		if !strings.HasSuffix(row, boxRailFor(b)) {
			t.Errorf("width %d: the row does not close on the box:\n%q", w, row)
		}
	}
}

// boxRailFor is the vertical the air box closes its rows with.
func boxRailFor(b Broadcaster) string { return render.HeavyBox(b.ascii).Rail }

// bedRowOf finds the bed's row wherever the section draws it.
//
// IT LOOKS IN THE SECTION, NOT IN `stationLine` (D-107). The bed has no labelled
// row of its own; it has one row of the air box the section carries — the same
// fact, in the one place the HUM LEAD asked for it.
func bedRowOf(t *testing.T, rows []string) string {
	t.Helper()
	for _, r := range rows { // bounded by the section (P10-02)
		if strings.Contains(stripANSITest(r), "RELAY BED") {
			return r
		}
	}
	t.Fatalf("no bed row in the station section:\n%q", rows)
	return ""
}
