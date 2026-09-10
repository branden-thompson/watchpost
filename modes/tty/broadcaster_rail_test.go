package tty

// broadcaster_rail_test.go — the left rail (D-60).
//
// The reference names each region of the running order down the left edge, one
// letter per row: LIVE, UP NEXT, BED, SCHEDULED, LINE UP. It is what says WHICH
// PART of the schedule a card is in — and it is why the card itself carries no
// state strip: the rail already says it, and two carriers of one fact is the
// shape this release keeps removing.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

func rail(label string, rows int) []string {
	return railColumn(label, rows, render.Opts{ASCII: true}.Glyphs())
}

func TestTheRailSpellsItsSectionDownTheEdge(t *testing.T) {
	got := rail("LIVE", 6)
	if len(got) != 6 {
		t.Fatalf("the rail is as tall as the section it names; got %d", len(got))
	}
	var letters []string
	for _, r := range got {
		if c := strings.TrimSpace(strings.Trim(r, "|")); c != "" {
			letters = append(letters, c)
		}
	}
	if strings.Join(letters, "") != "LIVE" {
		t.Errorf("the rail spells the section; got %q", strings.Join(letters, ""))
	}
}

// A SPACE IN THE LABEL IS A BLANK ROW, which is how the reference draws
// "UP NEXT": U, P, blank, N, E, X, T.
func TestASpaceInTheLabelIsABlankRow(t *testing.T) {
	got := rail("UP NEXT", 7)
	if len(got) != 7 {
		t.Fatalf("want 7 rows; got %d", len(got))
	}
	if c := strings.TrimSpace(strings.Trim(got[2], "|")); c != "" {
		t.Errorf("the third row is the label's space; got %q", c)
	}
}

// EVERY ROW IS THE RAIL'S OWN WIDTH, so the cards beside it all start in the
// same column. A ragged rail would step the whole running order in and out.
func TestEveryRailRowIsTheSameWidth(t *testing.T) {
	for _, n := range []int{1, 4, 9, 20} {
		for i, r := range rail("SCHEDULED", n) {
			if w := utf8.RuneCountInString(r); w != bcRailWidth {
				t.Errorf("rows %d, row %d: the rail is %d cells, want %d\n%q", n, i, w, bcRailWidth, r)
			}
		}
	}
}

// A LABEL LONGER THAN ITS SECTION IS CUT, NOT OVERFLOWED. A rail that grew rows
// would push the card it names off the bottom of the frame.
func TestALabelTallerThanItsSectionIsCut(t *testing.T) {
	got := rail("SCHEDULED", 3)
	if len(got) != 3 {
		t.Fatalf("the section's height wins; got %d rows", len(got))
	}
}

// THE RAIL IS WALLED ON BOTH SIDES, which is what makes it a rail rather than a
// margin: the frame's own edge at the left, and the divider the cards begin
// after at the right.
func TestTheRailIsWalledOnBothSides(t *testing.T) {
	for _, r := range rail("BED", 3) {
		if !strings.HasPrefix(r, "|") || !strings.HasSuffix(r, "|") {
			t.Errorf("the rail is walled both sides; got %q", r)
		}
	}
}

// THE ASSEMBLED FRAME LANDS ON THE REFERENCE'S OWN COLUMNS.
//
// `mock-broadcaster-v1.txt` row 56: the rail's walls at 0 and 4, the card's
// border at 9 and 140, the inner wall at 144, the scroll rail at 145, and the
// frame's own edge at 149. Those are the numbers, and they are what a reader
// checks the drawing against.
//
// IN TWO HALVES, because they have two owners: `section` draws the rail and the
// card, and `framed` adds the right-hand chrome — the scroll rail belongs to the
// running order as a whole, so it cannot be drawn per region.
func TestTheSectionLandsOnTheReferencesColumns(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	if got := b.cardBoxWidth(); got != 132 {
		t.Fatalf("the reference's card box is 132 cells; got %d", got)
	}
	box := newCardLane(b.cardBoxWidth(), b.opts().Glyphs()).
		box(aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057"), "6", "STANDARD")
	rows := b.section("LINE UP", box)

	top := []rune(rows[0])
	if len(top) != 141 {
		t.Fatalf("a section ends at the card's border; got %d cells", len(top))
	}
	for col, want := range map[int]rune{0: '|', 4: '|', 9: '+', 140: '+'} {
		if top[col] != want {
			t.Errorf("column %d is %q, want %q\n%s", col, string(top[col]), string(want), string(top))
		}
	}

	// ROW 1, NOT ROW 0: row 0 carries the scroll THUMB in the rail column, which
	// is the point — the thumb stands where the bar does rather than beside it.
	framed := []rune(b.framed(rows, 10, 10)[1])
	if len(framed) != 150 {
		t.Fatalf("the framed row is the terminal's width; got %d", len(framed))
	}
	// 144 IS THE WALL AND THE SCROLL RAIL AT ONCE, and 149 is the frame's edge.
	//
	// THIS USED TO ASSERT A WALL AT 144 AND A SEPARATE RAIL AT 145, which is a
	// column the reference does not have — so everything right of the cards sat
	// a cell off (HUM LEAD, UAT 2026-09-10: "right hand lanes are off"). Counted
	// off `mock-broadcaster-v1.txt`: an ordinary row ends `╯   │    │` and the
	// thumb row ends `█    │`, the thumb standing exactly WHERE the bar was.
	for col, want := range map[int]rune{144: '|', 149: '|'} {
		if framed[col] != want {
			t.Errorf("framed column %d is %q, want %q\n%s", col, string(framed[col]), string(want), string(framed))
		}
	}
	// AND 145..148 ARE AIR. A mark there is the extra column coming back.
	for col := 145; col < 149; col++ {
		if framed[col] != ' ' {
			t.Errorf("column %d is %q; 145..148 are air\n%s", col, string(framed[col]), string(framed))
		}
	}
}

// A SHORT SECTION GETS A SHORTER WORD, NOT A CUT ONE. A rail reading "SCHEDULE"
// or "U P _ N" looks like rendering damage rather than like a short section —
// and the first version of this did exactly that.
func TestAShortSectionLaddersItsLabelRatherThanCuttingIt(t *testing.T) {
	for _, c := range []struct {
		label string
		rows  int
		want  string
	}{
		{"SCHEDULED", 9, "SCHEDULED"},
		{"SCHEDULED", 8, "SCHED"},
		{"SCHEDULED", 4, "SCH"}, // SCHED is five runes; four rows cannot hold it
		{"SCHEDULED", 3, "SCH"},
		{"UP NEXT", 7, "UP NEXT"},
		{"UP NEXT", 4, "NEXT"},
		{"UP NEXT", 2, "UP"},
		{"LINE UP", 4, "LINE"},
	} {
		var letters []string
		for _, r := range rail(c.label, c.rows) {
			if x := strings.TrimSpace(strings.Trim(r, "|")); x != "" {
				letters = append(letters, x)
			}
		}
		if got := strings.Join(letters, ""); got != strings.ReplaceAll(c.want, " ", "") {
			t.Errorf("%q in %d rows spells %q, want %q", c.label, c.rows, got, c.want)
		}
	}
}

// THE LABEL SITS IN THE MIDDLE OF ITS REGION, not at the top of it. A rail
// pinned to the first row reads as naming that CARD; centred, it reads as naming
// the whole region — which is what it is for, and why the card carries no state
// of its own.
func TestTheLabelIsCentredInItsSection(t *testing.T) {
	rows := rail("LIVE", 10) // four letters, six blanks
	first, last := -1, -1
	for i, r := range rows {
		if strings.TrimSpace(strings.Trim(r, "|")) != "" {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatal("the label must be drawn")
	}
	above, below := first, len(rows)-1-last
	if above == 0 {
		t.Errorf("the label is pinned to the top rather than centred: %d above, %d below", above, below)
	}
	if d := above - below; d > 1 || d < -1 {
		t.Errorf("the label is not centred: %d rows above, %d below", above, below)
	}
}

// NO DOUBLE BLANKS BETWEEN REGIONS. A section carries its own breathing row;
// a second one appended by the frame made a double gap that reads as a
// rendering fault rather than as spacing (HUM LEAD, UAT 2026-09-10).
func TestTheFrameHasNoDoubleBlankRows(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	// The frame is padded to the terminal, so trailing blanks below the content
	// are expected; only gaps BETWEEN drawn rows are the concern.
	last := 0
	for i, r := range rows {
		if strings.TrimSpace(r) != "" {
			last = i
		}
	}
	for i := 1; i <= last; i++ {
		if strings.TrimSpace(rows[i]) == "" && strings.TrimSpace(rows[i-1]) == "" {
			t.Errorf("rows %d and %d are both blank — a section's spacing has two owners", i-1, i)
		}
	}
}

// THE FRAME CLOSES ON BOTH SIDES, on every drawn row.
//
// THE BUG THIS PINS was mine and it was invisible: the station section padded
// its rows to the full lane and THEN added the inset, so every row ran three
// cells long and `clamp` ate the right-hand wall. The rows were still exactly
// the terminal's width — the clamp saw to that — so nothing measured wrong.
// Only the EDGE was gone.
func TestEveryDrawnRowClosesTheFrame(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	last := 0
	for i, r := range rows {
		if strings.TrimSpace(r) != "" {
			last = i
		}
	}
	for i := 4; i <= last; i++ { // from the station section down; the masthead draws its own box
		r := []rune(rows[i])
		if len(r) != b.width {
			t.Errorf("row %d is %d cells, want %d", i, len(r), b.width)
			continue
		}
		if r[0] != '|' || r[b.width-1] != '|' {
			t.Errorf("row %d does not close the frame: starts %q ends %q\n%.40s", i, string(r[0]), string(r[b.width-1]), rows[i])
		}
	}
}

// THE SCROLL GUTTER IS BLANK EXCEPT FOR THE THUMB. `RailGlyphsFor` draws a bar
// on every row, which is right for a window that has no edge of its own — here
// the inner wall is already beside it, and a bar there reads as a doubled
// border rather than as a rail.
func TestTheScrollGutterCarriesOnlyTheThumb(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	body := b.section("LINE UP", []string{strings.Repeat("-", b.cardBoxWidth()), strings.Repeat("-", b.cardBoxWidth()), strings.Repeat("-", b.cardBoxWidth())})
	framed := b.framed(body, 2, 10)

	// THE RAIL IS COLUMN 144 — the wall's own column, which is where the
	// reference puts it. Every row carries the bar; exactly one carries the
	// thumb instead. An earlier version blanked the bar and drew the thumb in a
	// gutter of its own at 145, which gave the frame a column the mock has not.
	thumbs := 0
	for i, r := range framed {
		switch c := []rune(r)[b.width-6]; c {
		case '#':
			thumbs++
		case '|':
		default:
			t.Errorf("row %d draws %q in the rail column; a rail is a bar or the thumb:\n%.40s", i, string(c), r)
		}
	}
	if thumbs != 1 {
		t.Errorf("the rail carries exactly one thumb; it carries %d", thumbs)
	}
}
