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

// boxOf draws one card as a box, with nothing in it.
//
// `cardLane.box` RETIRED AT D-110 — it drew the flat, ordered slots of the
// pre-table running order, and those became rows of `LineupTable` at D-94. What
// this file is about is the BOX, which is still drawn, so the fixture reaches the
// drawer that still exists rather than keeping a wrapper alive for one caller.
func boxOf(t *testing.T, box int, c lineup.Card, handle, badge string) []string {
	t.Helper()
	_ = handle // the handle left the title row at D-110; the way in is the footer
	g := render.Opts{ASCII: true}.Glyphs()
	return newCardLane(box, g).boxOf(c, badge, nil)
}

// A CARD FILLS ITS BOX AT EVERY WIDTH.
//
// ITS HEIGHT IS THE BODY'S (D-87). A fixed four rows whatever it held — border,
// title, the corners' row, border — belongs to an overlay this console does not
// draw, so what a card is tall is what the
// region puts inside it.
func TestACardFillsItsBoxAtEveryWidth(t *testing.T) {
	c := aCard(t, "OCEANSIDE, CA 92057")
	for _, box := range []int{82, 112, 132} {
		rows := boxOf(t, box, c, "6", "STANDARD")
		// TWO ROWS WITH AN EMPTY BODY (D-110): a card is its BORDERS and whatever
		// is put between them. The title is in the top border, where the reference
		// draws it.
		if len(rows) < 2 {
			t.Fatalf("box %d: a card is at least two borders; got %d rows", box, len(rows))
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
	// EVERY INTERIOR ROW, however many the body gives it (D-87 made the height
	// the body's). A card that lost a wall on one row would read as damage.
	for i, r := range rows[1 : len(rows)-1] {
		if !strings.HasPrefix(r, "|") || !strings.HasSuffix(r, "|") {
			t.Errorf("interior row %d must be walled on both sides; got %q", i+1, r)
		}
	}
}

// THE BADGE RIDES THE RULE, at its right (D-110).
//
// THE HANDLE IS NOT ON THIS ROW. The reference puts the badge in the border —
// `━━━ • STANDARD • ━━━` — and the handle in the footer beside the presenter, so
// no row carries them both. What
// survives is that the box says what GRADE the card is without spending a line.
func TestTheBoxsRuleCarriesTheGrade(t *testing.T) {
	rule := boxOf(t, 132, aCard(t, "OCEANSIDE, CA"), "6", "STANDARD")[0]
	if !strings.Contains(rule, "STANDARD") {
		t.Errorf("the box's rule does not grade the card: %q", rule)
	}
	// AT THE RIGHT, and with the rule's own marks still closing the corner.
	if !strings.HasSuffix(rule, "---+") {
		t.Errorf("the badge pushed the corner off the rule: %q", rule)
	}
	if at := strings.Index(rule, "STANDARD"); at < len([]rune(rule))/2 {
		t.Errorf("the badge sits at %d of %d; the reference puts it at the right", at, len([]rune(rule)))
	}
}

// THE CORNERS' MARK RETIRED AT D-87, ON ITS OWN REASONING.
//
// D-57 put the fabricated-event mark at BOTH ends of a card because "a card can
// be occluded from either side by the priority overlay, and two corners cannot
// both be covered by a box that starts at the left." The tracks are columns now
// and nothing is occluded, so the title row's mark is always visible and the
// second copy cost every card a row.
//
// WHAT SURVIVES IS THE FACT ITSELF: a fabricated event still says so, once,
// where the operator reads the card's name.
// AND IT SAYS SO IN THE RULE NOW (D-110), because that is where the card's name
// went. Without this the mark would simply have stopped being drawn when the
// title row retired — a fabricated event that looks exactly like a real one,
// which is the screenshot hazard FR-4.4 exists to prevent.
func TestAFabricatedCardStillSaysSoInItsRule(t *testing.T) {
	c := aCard(t, "OCEANSIDE, CA")
	c.Test = true
	rows := boxOf(t, 132, c, "6", "STANDARD")
	if !strings.Contains(rows[0], testEventMark) {
		t.Errorf("a fabricated card does not say so in its rule: %q", rows[0])
	}
	// FIRST, BEFORE THE HEADLINE. `boxRule` drops a title the box cannot hold, so
	// a mark behind the headline would be the first thing to go — and the whole
	// point of the mark is that it cannot be the thing that is missing.
	if at, head := strings.Index(rows[0], testEventMark), strings.Index(rows[0], "OCEANSIDE"); head >= 0 && at > head {
		t.Errorf("the mark follows the headline it is warning about: %q", rows[0])
	}
	// AND A TAKEOVER SAYS IT TOO, which is the card the rule was written for.
	c.Slot = lineup.BreakingAlert
	if burst := boxOf(t, 132, c, "A", "PRIORITY")[0]; !strings.Contains(burst, testEventMark) {
		t.Errorf("a fabricated takeover does not say so in its rule: %q", burst)
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
// THE REFERENCE'S OWN RULE, at the reference's own card width (D-110).
//
// THE HANDLE'S GEOMETRY RETIRED WITH THE TITLE ROW. What the v3 reference fixes
// is the RULE: the card's name three marks in from the corner, the grade three
// marks in from the other, and the corners landing whatever the title's length.
func TestTheBoxsRuleMatchesTheReferenceGeometry(t *testing.T) {
	rule := []rune(boxOf(t, 132, aCard(t, "OCEANSIDE, CA 92057"), "6", "STANDARD")[0])
	if got := len(rule); got != 132 {
		t.Fatalf("the reference's card is 132 cells; got %d", got)
	}
	if got := string(rule[:5]); got != "+--- " {
		t.Errorf("the reference opens the rule with three marks and a space; got %q", got)
	}
	if got := string(rule[len(rule)-5:]); got != " ---+" {
		t.Errorf("and closes it with three; got %q", got)
	}
}

// THE CARD DRAWS THE MASTHEAD'S BOX (D-85, HUM LEAD 2026-09-11): "Remove the
// rounded corners -> straight corners … All Main track cards should have BOLD
// lines (like the masthead)."
//
// SHARED, NOT COPIED. `render.HeavyBox` is the set the masthead has drawn since
// 0.13.0, and the test asks the SAME function rather than repeating the marks —
// a literal here would pass while the two drifted apart, which is the thing it
// is meant to prevent.
func TestACardIsDrawnInTheMastheadsBox(t *testing.T) {
	bx := render.HeavyBox(false)
	b := Broadcaster{width: 150}
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	rows := lane.boxOf(aCard(t, "OCEANSIDE, CA"), "STANDARD", nil)

	if !strings.HasPrefix(rows[0], bx.TL) || !strings.HasSuffix(rows[0], bx.TR) {
		t.Errorf("the top border is %q; want the masthead's corners %q … %q", rows[0], bx.TL, bx.TR)
	}
	last := rows[len(rows)-1]
	if !strings.HasPrefix(last, bx.BL) || !strings.HasSuffix(last, bx.BR) {
		t.Errorf("the bottom border is %q; want %q … %q", last, bx.BL, bx.BR)
	}
	for i, r := range rows[1 : len(rows)-1] {
		if !strings.HasPrefix(r, bx.Rail) || !strings.HasSuffix(r, bx.Rail) {
			t.Errorf("interior row %d is %q; want the masthead's rail %q on both edges", i, r, bx.Rail)
		}
	}
	// AND NOT THE ROUNDED ONES, which is the half the HUM LEAD asked for by
	// name. They have no user left and are gone from the glyph set.
	for _, round := range []string{"╭", "╮", "╰", "╯"} {
		if strings.Contains(strings.Join(rows, ""), round) {
			t.Errorf("the card still draws the rounded corner %q", round)
		}
	}
}
