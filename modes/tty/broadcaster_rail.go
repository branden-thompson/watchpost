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
	// Three of air, the inner wall, four more, the outer wall.
	tail := "   " + g.Rail + "    " + g.Rail
	out := make([]string, 0, len(rows))
	for i, r := range rows { // bounded by the section (P10-02)
		out = append(out, rail[i]+gap+r+tail)
	}
	return out
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
}
