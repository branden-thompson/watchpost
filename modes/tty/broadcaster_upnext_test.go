package tty

// broadcaster_upnext_test.go — the UP NEXT card in the reference's own shape
// (D-110).
//
// HUM LEAD, UAT 2026-09-12, handing back the mock section: "notice where the
// location for Up Next is".

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func upNextAt(t *testing.T, c lineup.Card) []string {
	t.Helper()
	b := bcWith(t, c)
	b.width, b.height, b.ascii = 150, 74, true
	return strings.Split(stripANSITest(strings.Join(b.upNextBox(), "\n")), "\n")
}

// cardCellOf is the card's own cell of a pair row: everything right of the label
// column's rail, less the box's closing one.
func cardCellOf(t *testing.T, row string) string {
	t.Helper()
	at := strings.LastIndex(row[:len(row)-1], "|")
	if at < 0 {
		t.Fatalf("no card cell in %q", row)
	}
	return strings.TrimSuffix(row[at+1:], "|")
}

// THE CARD NAMES ITSELF IN ITS OWN BORDER, not in a row inside it.
//
//	┏━━ LOCATION REPORT • Oceanside, CA 92057 ━━━━━━ • STANDARD • ━━━┓
//
// IT WAS A ROW, and that row spent a line of the card saying what the frame
// around it could say for free — on the one box on this console whose job is to
// hold a manifest.
func TestTheUpNextCardNamesItselfInItsRule(t *testing.T) {
	rows := upNextAt(t, card(t, "a", "Oceanside, CA 92057"))
	if len(rows) < 3 {
		t.Fatalf("the card drew %d rows", len(rows))
	}
	for _, want := range []string{"LOCATION REPORT", "Oceanside, CA 92057", "STANDARD"} {
		if !strings.Contains(rows[0], want) {
			t.Errorf("the rule does not carry %q:\n%s", want, rows[0])
		}
	}
	// AND NOTHING INSIDE REPEATS IT. A headline in the border and in the body is
	// the same fact twice on a box that is read at a glance.
	for i, r := range rows[1:] {
		if strings.Contains(r, "LOCATION REPORT") {
			t.Errorf("interior row %d repeats the card's name:\n%s", i+1, r)
		}
	}
}

// THE BODY IS THE REFERENCE'S ORDER: air, what it is doing and how fresh, air,
// the manifest under a centred caption, and the way in at the bottom.
func TestTheUpNextCardFollowsTheReferencesOrder(t *testing.T) {
	c := card(t, "a", "Oceanside, CA 92057")
	c.BuiltAt = time.Now().Add(-10 * time.Minute)
	rows := upNextAt(t, c)

	at := func(want string) int {
		for i, r := range rows { // bounded by the card (P10-02)
			if strings.Contains(r, want) {
				return i
			}
		}
		t.Errorf("the card is missing %q:\n%s", want, strings.Join(rows, "\n"))
		return -1
	}
	status, pull, caption := at("STATUS:"), at("DATA PULL:"), at(bcManifestCaption)
	heading, control := at("RANGE / INCIDENTS"), at("Read / Manage")
	if status < 0 || pull < 0 || caption < 0 || heading < 0 || control < 0 {
		return
	}
	for _, o := range [][2]int{{status, pull}, {pull, caption}, {caption, heading}, {heading, control}} {
		if o[0] >= o[1] {
			t.Errorf("the card's rows are out of the reference's order: %d before %d", o[0], o[1])
		}
	}
	// THE CAPTION IS CENTRED OVER THE CARD'S OWN CELL, which is what makes it a
	// caption over the table rather than a row of it. Measured inside the cell:
	// the label column to its left is a different box entirely.
	cell := cardCellOf(t, rows[caption])
	lead := len(cell) - len(strings.TrimLeft(cell, " "))
	tail := len(cell) - len(strings.TrimRight(cell, " "))
	if d := lead - tail; d < -2 || d > 2 {
		t.Errorf("%q is not centred in its cell: %d cells of air left, %d right\n%q",
			bcManifestCaption, lead, tail, cell)
	}
	// AND THE STAMP IS THE SHORT FORM. "Saturday September 12, 2026 @ 16:02:45" is
	// forty-two cells of a card that has about sixty.
	if strings.Contains(rows[pull], "September") || !strings.Contains(rows[pull], "AGO)") {
		t.Errorf("the card's stamp is not the reference's short form:\n%s", rows[pull])
	}
}

// THE WAY IN IS AT THE BOTTOM, BESIDE WHO WILL SAY IT — and an undecided slot
// still has one.
//
// F-97 ONE ROW ALONG. The chip used to ride the title row and an empty slot got
// one anyway; the way in moved to the footer, and a footer given only to a
// decided card would have made the slot unaddressable again.
func TestTheUpNextCardIsAddressableEmptyOrNot(t *testing.T) {
	empty := NewBroadcaster()
	empty.width, empty.height, empty.ascii = 150, 74, true
	for _, tc := range []struct {
		name string
		rows []string
	}{
		{"a decided slot", upNextAt(t, card(t, "a", "Oceanside, CA 92057"))},
		{"an empty slot", strings.Split(stripANSITest(strings.Join(empty.upNextBox(), "\n")), "\n")},
	} {
		rows := tc.rows
		last := rows[len(rows)-2] // the row above the closing border
		if !strings.Contains(last, chipFor("1")) {
			t.Errorf("%s: the card is not addressable; %q is missing from %q", tc.name, chipFor("1"), last)
		}
		if !strings.Contains(last, "PRESENTER:") {
			t.Errorf("%s: the footer does not say who will read it: %q", tc.name, last)
		}
	}
}

// THE BOX HAS A GROUND OF ITS OWN (D-114).
//
// HUM LEAD, 2026-09-13: "the UP Next Box probably needs a bkg color other than
// none - I suggest the same Blue as the modal for now."
//
// THE MODAL'S OWN TILE, THROUGH `ModalTone`, so it is the same blue and follows
// the theme. A second colour mixed here would be a second answer to what the
// app's tile blue is — and the Light theme's is not the dark one's, which is
// exactly the trap D-108 was about.
func TestTheUpNextBoxWearsTheModalsGround(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	b := bcWith(t, card(t, "a", "Oceanside, CA 92057"))
	b.width, b.height, b.darkBG = 150, 74, true
	rows := b.upNextBox()
	if len(rows) == 0 {
		t.Fatal("the card drew nothing")
	}
	fg, bg := render.ModalTone(true)
	for i, r := range rows { // bounded by the box (P10-02)
		if !strings.Contains(r, bg) {
			t.Errorf("row %d is not painted on the modal's ground:\n%q", i, r)
		}
	}
	// THE BORDERS TOO, which is the rule `shell` states for a card: a ground that
	// stopped at the border would read as a fill rather than as a box.
	if !strings.Contains(rows[0], bg) || !strings.Contains(rows[len(rows)-1], bg) {
		t.Error("the box's own borders are unpainted")
	}
	_ = fg
}

// AND THE BOX'S TITLE READS AS A MODAL'S DOES (D-118).
//
// HUM LEAD, 2026-09-13: "Let's make the Title 'LOCATION REPORT * <location>'
// Bold and White like the Modals." Now that the box wears the modal's own tile
// (D-114), the tone of its name was the only thing still telling them apart.
func TestTheBoxTitleReadsLikeAModalTitle(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	b := bcWith(t, card(t, "a", "Oceanside, CA 92057"))
	b.width, b.height, b.darkBG = 150, 74, true
	rule := b.upNextBox()[0]
	if !strings.Contains(rule, render.Tok(render.ModalTitle)) {
		t.Errorf("the box's title is not painted like a window's:\n%q", rule)
	}
	// THE RULE'S MARKS KEEP THE GROUND'S TONE: what is picked out is the NAME.
	_, bg := render.ModalTone(true)
	if !strings.Contains(rule, bg) {
		t.Errorf("the rule lost the box's ground:\n%q", rule)
	}
}

// TestTheColumnsNeverBleedIntoEachOther.
//
// HUM LEAD, 2026-09-11: "no more card occlusion."
//
// A ROW THAT COMES IN SHORT PULLS ITS NEIGHBOUR LEFTWARD on that row alone, and
// the result reads as a rendering fault rather than as a layout: the right box's
// wall zig-zags down the frame. `joinColumns` holds every cell to its own column,
// which is what stops it.
//
// DRIVEN DIRECTLY, BECAUSE NO CALLER CAN EXPRESS THE FAULT. Both boxes are built
// by `shell`, which pads every row to the box's width — so through `readPair` the
// pad is a no-op, and mutant mS0 deleted it without changing a single frame. It
// SURVIVED the whole 2026-09-13 corpus sweep for exactly that reason. The rule
// guards a width DISAGREEMENT between the box and the join, which is the defect
// `withControl` really did ship: a second copy of the air box's width, and the
// `b` chip fell off the row.
func TestTheColumnsNeverBleedIntoEachOther(t *testing.T) {
	const lw, rw = 10, 6
	gap := "  "
	left := []string{
		"1234567890",      // exactly the column
		"short",           // UNDER it — the case the rule is for
		"1234567890EXTRA", // OVER it
	}
	right := []string{"|abcd|", "|efgh|", "|ijkl|"}

	rows := joinColumns(left, right, lw, rw, gap)
	if len(rows) != 3 {
		t.Fatalf("joined %d rows, want 3", len(rows))
	}
	for i, r := range rows {
		if got, want := render.Width(r), lw+len(gap)+rw; got != want {
			t.Errorf("row %d is %d cells, want %d: %q", i, got, want, r)
		}
		// AND THE RIGHT COLUMN STARTS IN THE SAME PLACE ON EVERY ROW, which is
		// the thing the operator actually sees go wrong.
		if got := []rune(r)[lw+len(gap)]; got != '|' {
			t.Errorf("row %d starts the right column with %q, not its wall: %q", i, string(got), r)
		}
	}
}

// AND A SHORTER COLUMN IS AIR, NOT A RAGGED EDGE. The two boxes are not the same
// height — the takeover lists ten slots whether or not there are ten — so the
// rows past the end of one column must still hold their width open.
func TestTheShorterColumnHoldsItsWidthOpen(t *testing.T) {
	rows := joinColumns([]string{"abc"}, []string{"|x|", "|y|", "|z|"}, 3, 3, " ")
	if len(rows) != 3 {
		t.Fatalf("joined %d rows, want 3", len(rows))
	}
	for i, r := range rows {
		if got, want := render.Width(r), 3+1+3; got != want {
			t.Errorf("row %d is %d cells, want %d: %q", i, got, want, r)
		}
	}
}
