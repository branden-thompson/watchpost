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
// F-97 ONE ROW ALONG. The way in is in the footer, and a footer given only to a
// decided card puts an empty slot out of reach — the same defect the chip riding
// the title row was moved to close.
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
		// AND IT NO LONGER NAMES A PRESENTER (D-131). The per-card PRESENTER
		// control was dropped; the label outlived it and drew `PRESENTER: N/A`
		// on every card — HUM LEAD, UAT 2026-09-14. Pinned as an ABSENCE
		// because that is what regresses: the label is easy to put back beside
		// a field that still exists.
		if strings.Contains(last, "PRESENTER") {
			t.Errorf("%s: the footer still names the removed per-card presenter: %q", tc.name, last)
		}
	}
}

// THE BOX HAS TWO GROUNDS (D-114, D-134, corrected by D-136).
//
// HUM LEAD, 2026-09-13: "the UP Next Box probably needs a bkg color other than
// none - I suggest the same Blue as the modal FOR NOW."  Then 2026-09-15: "make
// the cell background color of the UP NEXT box the same 'blue' token color used
// as the bkg for 'DIRECTION' and 'TODAY' column."  Then, with a diagram, the
// correction: the LABEL CELL is that blue; the report beside it is "STANDARD
// MODAL BLUE" and its "content ... should not be all bold and white, but the
// standard text color".
//
// SO THE TEST MEASURES BOTH, and would have caught the over-application: the
// first version asked only whether every row carried the bands' blue, which a
// box painted entirely in it passes perfectly.
func TestTheUpNextBoxWearsTwoGrounds(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	b := bcWith(t, card(t, "a", "Oceanside, CA 92057"))
	b.width, b.height, b.darkBG = 150, 74, true
	rows := b.upNextBox()
	if len(rows) == 0 {
		t.Fatal("the card drew nothing")
	}
	today := render.Tok(render.GroupTodayBG)
	_, modal := render.ModalTone(true)

	// THE REPORT IS ON THE MODAL'S TILE, every row of it, borders included —
	// the rule `shell` states for a card: a ground that stopped at the border
	// would read as a fill rather than as a box.
	for i, r := range rows { // bounded by the box (P10-02)
		if !strings.Contains(r, modal) {
			t.Errorf("row %d does not carry the report's modal ground:\n%q", i, r)
		}
	}
	// AND THE LABEL CELL IS THE BANDS' BLUE — on the rows that HAVE a cell.
	var seen bool
	for _, r := range rows {
		if strings.Contains(r, today) {
			seen = true
		}
	}
	if !seen {
		t.Error("no row carries the bands' blue: the label cell lost its ground")
	}
	// AND THE REPORT'S OWN TEXT IS NOT PAINTED IN THE LABEL CELL'S BLUE — the
	// half the HUM LEAD reported in words: "should not be all bold and white".
	//
	// THE GROUND IN FORCE WHERE THE TEXT IS, not "does this row mention the
	// blue". EVERY body row carries a label cell, so the blue appears on all of
	// them and a Contains check answers the wrong question. What paints a run of
	// text is the last ground opened before it.
	for _, r := range rows {
		plain := stripANSITest(r)
		if !strings.Contains(plain, "STATUS:") {
			continue
		}
		head := r[:strings.Index(r, "STATUS:")]
		all := bgRe.FindAllString(head, -1)
		if len(all) == 0 {
			t.Fatalf("the report's row opens no ground at all:\n%q", r)
		}
		if got := all[len(all)-1]; got != modal {
			t.Errorf("the report's text is painted on %q, not the modal tile %q:\n%q", got, modal, r)
		}
	}
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
	// The title rides the REPORT's half of the box, which is the modal tile
	// (D-136) — the bands' blue belongs to the label cell alone.
	if _, bg := render.ModalTone(true); !strings.Contains(rule, bg) {
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
