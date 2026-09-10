package tty

// broadcaster_rail.go — the left rail (D-60).
//
// The reference names each region of the running order down the left edge, one
// letter per row: LIVE, UP NEXT, BED, SCHEDULED, LINE UP.
//
// IT IS WHY THE CARD CARRIES NO STATE OF ITS OWN. An earlier card mock had a
// "state strip" saying whether a card was live or scheduled, and the HUM LEAD
// cut it: the rail already says it. Two carriers of one fact is the shape this
// release keeps removing, and here the second one would have been on every card
// rather than once per section.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// bcRailWidth is the rail's own columns, from the reference: the frame's edge,
// a space, the letter, a space, the divider the cards begin after.
//
//	│ L │
//	0 1 2 3 4
const bcRailWidth = 5

// railColumn is the rail beside a section of `rows` rows, spelling `label` down
// it.
//
// THE LABEL IS CENTRED VERTICALLY, so a four-letter word beside a six-row
// section sits in the middle of it rather than at the top — which is what makes
// the rail read as naming the whole region instead of its first card.
//
// A SPACE IN THE LABEL IS A BLANK ROW. That is how the reference draws
// "UP NEXT": U, P, blank, N, E, X, T — the label's own shape, not a second rule
// about where to break it.
//
// A LABEL TALLER THAN ITS SECTION LADDERS DOWN rather than being cut: a rail
// reading "SCHEDULE" or "U P _ N" looks like damage, not like a short section.
// The rail can never GROW rows — it names a region, and a rail that grew would
// push the card it names off the bottom of the frame — so the word gives way
// instead, exactly as the masthead's title does.
func railColumn(label string, rows int, g render.Glyphs) []string {
	if rows <= 0 {
		return nil
	}
	letters := []rune(bcRailLabel(label, rows))
	top := (rows - len(letters)) / 2
	out := make([]string, 0, rows)
	for i := range rows { // bounded by the section (P10-02)
		cell := " "
		if i >= top && i-top < len(letters) {
			cell = string(letters[i-top])
		}
		out = append(out, g.Rail+" "+cell+" "+g.Rail)
	}
	return out
}

const (
	// bcRailGap is the air between the rail's divider and a card's border, from
	// the reference: the divider sits at column 4 and the card begins at 9.
	bcRailGap = 4

	// bcRightChrome is the frame's own columns at the right: three of air, the
	// inner wall, four more, and the outer wall — where the scroll rail lives.
	//
	//	… ╮   │    │
	//	    141 144 149      (at the reference's 150)
	bcRightChrome = 9
)

// cardBoxWidth is how wide a card's BOX is once the frame has taken its own
// columns — the ONE place that arithmetic lives (D-51's seam).
//
// At the reference's 150 columns this is 132, which puts the card's borders at
// 9 and 140 exactly where the mock draws them.
func (b Broadcaster) cardBoxWidth() int {
	w := b.width - bcRailWidth - bcRailGap - bcRightChrome
	if w < 4 {
		return 0
	}
	return w
}

// section draws one named region of the running order: its rail down the left,
// its cards' boxes beside it, and the frame's own columns at the right.
//
// THE RAIL IS AS TALL AS THE SECTION, which is why it is built from the rows
// rather than beside them: a label centred over a region it does not measure
// would drift the moment a card was added.
func (b Broadcaster) section(label string, rows []string) []string {
	if len(rows) == 0 {
		return nil
	}
	g := b.opts().Glyphs()
	rail := railColumn(label, len(rows), g)
	gap := strings.Repeat(" ", bcRailGap)
	// THE RIGHT-HAND CHROME IS NOT ADDED HERE. It carries the SCROLL RAIL, and
	// the rail is ONE continuous column over the whole running order — a thumb
	// drawn per section would put one in every region and say that each scrolls
	// on its own. `lanes` adds it once, after the regions are assembled.
	out := make([]string, 0, len(rows))
	for i, r := range rows { // bounded by the section (P10-02)
		out = append(out, rail[i]+gap+r)
	}
	return out
}

// regionGap is the blank row BETWEEN two named regions.
//
// THE REFERENCE LETS ITS SECTIONS BREATHE and this did not: every card in the
// running order sat flush against the next, so LIVE, UP NEXT, SCHEDULED and
// LINE UP read as one undifferentiated stack of boxes and the rail was the only
// thing saying otherwise (HUM LEAD, UAT 2026-09-10: "the sections of the line up
// are missing their blank row between the sections like the mocks").
//
// BETWEEN REGIONS, NEVER BETWEEN CARDS. Cards inside a region ARE consecutive in
// the reference — a gap between every card would say each one is its own
// section, which is the opposite of what the rail is for.
//
// IT KEEPS THE RAIL'S OWN WALLS, because it is a row of the frame like any
// other: an unwalled blank row is a hole in the border, which is how the frame
// came to look broken between the regions in the first place.
func (b Broadcaster) regionGap() string {
	g := b.opts().Glyphs()
	return railColumn(" ", 1, g)[0] + strings.Repeat(" ", bcRailGap+b.cardBoxWidth())
}

// bcRailLabel is the widest form of a section's name that fits its height.
//
// THE LADDER IS PER LABEL because the abbreviation is a word, not an algorithm:
// "SCHEDULED" shortens to "SCHED", and no general rule produces that. The last
// rung is always something — a region with a rail that said nothing would be
// indistinguishable from the gap between two regions.
func bcRailLabel(label string, rows int) string {
	forms, ok := bcRailForms[label]
	if !ok {
		forms = []string{label}
	}
	for _, f := range forms { // widest first
		if len([]rune(f)) <= rows {
			return f
		}
	}
	last := forms[len(forms)-1]
	if r := []rune(last); len(r) > rows {
		return string(r[:max(0, rows)])
	}
	return last
}

// bcRailForms is each section's name, widest first.
var bcRailForms = map[string][]string{
	"LIVE":      {"LIVE"},
	"UP NEXT":   {"UP NEXT", "NEXT", "UP"},
	"SCHEDULED": {"SCHEDULED", "SCHED", "SCH"},
	"LINE UP":   {"LINE UP", "LINE", "UP"},
	"BED":       {"BED"},
	// THE OVERLAY IS AS TALL AS THE TAKEOVER IT HOLDS, which for a single card
	// is four rows — too few for eight letters. It sheds to a word rather than
	// being cut: "PRIO" reads as damage, and this label appears exactly when
	// something is interrupting the broadcast.
	"PRIORITY": {"PRIORITY", "ALERT", "!"},
}

// spliceAt writes `patch` into `base` starting at column `col`, one row per
// entry, and returns the result.
//
// RUNE-ACCURATE, because the frame is full of box-drawing and the badge's
// bullets are three bytes each — a byte offset here reads as a plausible wrong
// number, which is how a card's handle came to sit one cell off the reference.
//
// IT NEVER GROWS THE FRAME. A patch that ran past the row is cut: the frame is
// the viewport (D-58), and something composited BESIDE it rather than onto it is
// the defect UAT found in the diagnostics window.
func spliceAt(base []string, patch []string, col int) []string {
	if col < 0 {
		return base
	}
	out := append([]string(nil), base...)
	for i, p := range patch { // bounded by the patch (P10-02)
		if i >= len(out) {
			break
		}
		row, ins := []rune(out[i]), []rune(p)
		if col >= len(row) {
			continue
		}
		for j, r := range ins {
			if col+j >= len(row) {
				break
			}
			row[col+j] = r
		}
		out[i] = string(row)
	}
	return out
}

// bcPriorityCol is where the priority overlay's box begins — just past the
// rail's divider, from the reference: the divider sits at column 4 and the
// overlay's border at 6.
const bcPriorityCol = bcRailWidth + 1

// priorityWidth is the overlay box's own width.
//
// HALF THE CARD BOX, from the reference: at its 150 columns the card runs 9..140
// (132 cells) and the overlay 6..72 (67) — so the operator keeps the right-hand
// half of every card it covers, which is the half carrying the badge and the
// HANDLE they type.
//
// PROPORTIONAL RATHER THAN FIXED, because the reference gives one width and a
// fixed 67 cells would swallow two thirds of a 100-column frame — the narrowest
// the station supports (D-50).
func (b Broadcaster) priorityWidth() int { return (b.cardBoxWidth() + 2) / 2 }

// framed adds the right-hand chrome to the assembled running order: the inner
// wall, the scroll rail, and the frame's own edge.
//
// THE RAIL IS `render.Railify`, THE ONE OWNER (D-56). It already tracks a thumb
// over a window and already pads with PadTo rather than PadBetween, "so a
// full-width line must never push the rail right" — an off-by-one that surface
// found and fixed once already.
//
// IT IS APPLIED TO THE WHOLE BODY, not per region, because the running order
// scrolls as one thing. At the reference's 150 columns this puts the rail at
// 144 and the frame's edge at 149.
//
// THE WALL AND THE RAIL ARE ONE COLUMN, WHICH IS WHAT THE REFERENCE DRAWS. This
// built them as two — a wall at 144 and a thumb at 145 — so the console carried
// a column the mock does not have and everything right of the cards sat a cell
// off (HUM LEAD, UAT 2026-09-10: "right hand lanes are off"). Counted off the
// mock: an ordinary row ends `╯   │    │` and the thumb row `█    │`, the thumb
// standing exactly WHERE the bar was. One column, two glyphs.
func (b Broadcaster) framed(body []string, shown, total int) []string {
	g := b.opts().Glyphs()
	inner := make([]string, len(body))
	for i, r := range body { // bounded by the body (P10-02)
		inner[i] = r + "   "
	}
	// THE BAR IS THE WALL. `RailGlyphsFor` already draws one on every row, and
	// that is the reference's own inner edge — an earlier version blanked it and
	// drew a separate wall, which is how the extra column got in.
	glyphs := render.RailGlyphsFor(b.ascii)
	glyphs.Bar = g.Rail
	railed := render.Railify(inner, b.width-5, 0, max(total, 1), max(shown, 1), glyphs)
	out := make([]string, len(railed))
	for i, r := range railed { // bounded by the body (P10-02)
		out[i] = r + "    " + g.Rail
	}
	return out
}
