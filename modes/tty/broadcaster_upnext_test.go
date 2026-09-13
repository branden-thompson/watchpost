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
