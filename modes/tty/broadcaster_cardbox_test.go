package tty

// broadcaster_cardbox_test.go — the card as a BOX (D-52's rendering, D-57's marks).
//
// The reference draws every card as four rows: a top border, the title row, a
// body row, and a bottom border. The console drew flat rows until now, which is
// why D-57's TEST CORNERS had nowhere to sit — corners need borders.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

func boxOf(t *testing.T, box int, c lineup.Card, handle, badge string) []string {
	t.Helper()
	g := render.Opts{ASCII: true}.Glyphs()
	return newCardLane(box, g).box(c, handle, badge)
}

func TestACardIsFourRowsAndFillsItsBox(t *testing.T) {
	c := aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057")
	for _, box := range []int{82, 112, 132} {
		rows := boxOf(t, box, c, "6", "STANDARD")
		if len(rows) != bcCardRows {
			t.Fatalf("box %d: a card is %d rows; got %d", box, bcCardRows, len(rows))
		}
		for i, r := range rows {
			if w := utf8.RuneCountInString(r); w != box {
				t.Errorf("box %d row %d: %d cells\n%q", box, i, w, r)
			}
		}
	}
}

func TestACardHasBordersOnEveryEdge(t *testing.T) {
	rows := boxOf(t, 132, aCard(t, "OCEANSIDE, CA"), "6", "STANDARD")
	top, bottom := rows[0], rows[len(rows)-1]
	if !strings.HasPrefix(top, "+") || !strings.HasSuffix(top, "+") {
		t.Errorf("the top row is a border; got %q", top[:12])
	}
	if !strings.HasPrefix(bottom, "+") || !strings.HasSuffix(bottom, "+") {
		t.Errorf("the bottom row is a border; got %q", bottom[:12])
	}
	for _, i := range []int{1, 2} {
		if !strings.HasPrefix(rows[i], "|") || !strings.HasSuffix(rows[i], "|") {
			t.Errorf("row %d must be walled on both sides; got %q", i, rows[i])
		}
	}
}

// THE TITLE STILL OBEYS THE ANCHORING RULE, one width in from the borders. The
// box does not get to move the handle.
func TestTheBoxedCardKeepsItsHandleRightMost(t *testing.T) {
	rows := boxOf(t, 132, aCard(t, "OCEANSIDE, CA"), "6", "STANDARD")
	title := strings.TrimSuffix(strings.TrimPrefix(rows[1], "|"), "|")
	if !strings.HasSuffix(strings.TrimRight(title, " "), "[ 6 ]") {
		t.Errorf("the handle is right-most inside the box; got %q", title)
	}
}

// D-57: THE TEST MARKS GO IN THE CORNERS.
//
//	"we have a UI where overlays happen — we can mark 'TEST' on the card corners
//	of a fabricated event"
//
// A SINGLE MARK HAS A SINGLE POINT OF FAILURE, and the priority track sits ON
// TOP of the main track — so a takeover card is the one most likely to be partly
// occluded, which makes one mark on that card type the weakest possible
// placement. Corners cannot all be lost to truncation, occlusion, or a cropped
// screenshot.
func TestAFabricatedCardIsMarkedInItsCorners(t *testing.T) {
	c := aCard(t, "TORNADO WARNING + 2 more")
	c.Test = true
	rows := boxOf(t, 132, c, "T", "PRIORITY")

	if !strings.Contains(rows[1], testEventMark) {
		t.Errorf("the mark leads the title row:\n%q", rows[1])
	}
	corners := rows[2]
	if strings.Count(corners, "TEST") < 2 {
		t.Errorf("both bottom corners carry the mark:\n%q", corners)
	}
	// LEFT AND RIGHT, not two marks huddled together.
	l, r := strings.Index(corners, "TEST"), strings.LastIndex(corners, "TEST")
	if r-l < 40 {
		t.Errorf("the marks are corners, not neighbours: %d and %d\n%q", l, r, corners)
	}
	// AND THE BADGE KEEPS ITS LANE (HUM LEAD: "Keeping PRIORITY is fine").
	if !strings.Contains(rows[1], "PRIORITY") {
		t.Errorf("the lane badge is not replaced by the mark:\n%q", rows[1])
	}
}

func TestARealCardHasNoCorners(t *testing.T) {
	rows := boxOf(t, 132, aCard(t, "OCEANSIDE, CA"), "6", "STANDARD")
	for i, r := range rows {
		if strings.Contains(r, "TEST") {
			t.Errorf("row %d of a real hazard's card says TEST:\n%q", i, r)
		}
	}
}

// THE REFERENCE'S OWN GEOMETRY, at the reference's own card width.
//
// `mock-broadcaster-v1.txt` draws the LINE UP card between columns 9 and 140 —
// 132 cells — with the handle at 134..138 and the border at 140, so ONE cell
// sits between them. The component right-aligns a badge FLUSH, which put the
// handle hard against the border until the tail carried that cell itself.
//
// MEASURED IN RUNES, because the badge's bullets are three bytes each and a
// byte offset here reads as a plausible wrong number.
func TestTheBoxedCardMatchesTheReferenceGeometry(t *testing.T) {
	rows := boxOf(t, 132, aCard(t, "LOCATION REPORT • OCEANSIDE, CA 92057"), "6", "STANDARD")
	title := []rune(rows[1])
	if got := len(title); got != 132 {
		t.Fatalf("the reference's card is 132 cells; got %d", got)
	}
	if got, want := string(title[len(title)-3:]), "] |"; got != want {
		t.Errorf("the reference leaves one cell between the handle and the border; got %q want %q", got, want)
	}
}
