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
	// COMPLETELY BLANK — NO WALLS, NOT EVEN THE RAIL'S (HUM LEAD, UAT
	// 2026-09-10): "the blank row in between sections needs to be completely
	// blank — right now the left rail is connected top to bottom; the breaks in
	// the mock were intentional."
	//
	// THE BREAK IS THE SEPARATOR. A rail that runs unbroken from LIVE to the
	// bottom of the LINE UP draws the four regions as one column with labels in
	// it; a rail that stops and starts draws four regions. The gap is doing the
	// work, and a wall through it undoes exactly that.
	return strings.Repeat(" ", b.orderWidth())
}

// railSpacer is the row of air UNDER the lane's header, before its first card.
//
// IT KEEPS THE RAIL'S WALLS, and a region gap does not — which looks like an
// inconsistency and is the reference drawn exactly. A gap BETWEEN two regions
// has to break the rail, because an unbroken rail draws four regions as one
// column with labels in it. This row breaks nothing: it is the top of the
// running order, and the rail begins here.
func (b Broadcaster) railSpacer() string {
	g := b.opts().Glyphs()
	return railColumn(" ", 1, g)[0] + strings.Repeat(" ", b.orderWidth()-bcRailWidth)
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
// IT SPLICES BY DISPLAY COLUMN, through `render.SpliceCells` — the same measure
// `Width` and `TruncateCells` use (D-66).
//
// IT DID NOT, AND F-85 SAID SO AND WAS NOT ACTED ON. The note reasoned that a
// rune-accurate splice "holds today by accident of layout", because every escape
// on the covered rows sat to the right of the span. It stopped holding the first
// time a takeover was drawn over a real card, and the HUM LEAD saw the card
// underneath lose its right-hand side (UAT 2026-09-10). **A filed follow-up is
// not a fix, and "it holds by accident" is a prediction with a date on it.**
func spliceAt(base []string, patch []string, col int) []string {
	if col < 0 {
		return base
	}
	out := append([]string(nil), base...)
	for i, p := range patch { // bounded by the patch (P10-02)
		if i >= len(out) {
			break
		}
		out[i] = render.SpliceCells(out[i], p, col)
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
	return b.chrome(body, true, shown, total)
}

// chrome adds the right-hand columns, with or without the scroll rail.
//
// THE RAIL SPANS ONLY WHAT SCROLLS (D-68). The HUM LEAD, beside the SCHEDULED
// region of his reference: "Notice the top of the scroll is here, and the left
// rail is separated." The two cards the operator reads from are always the same
// two — there is nothing to scroll past — so the gutter beside them is air, and
// a thumb drawn there would say they move when they do not.
func (b Broadcaster) chrome(body []string, rail bool, shown, total int) []string {
	g := b.opts().Glyphs()
	glyphs := render.RailGlyphsFor(b.ascii)
	// THE MARK IN COLUMN 144 FOR EACH ROW. Without a rail that is the wall, on
	// every row; with one it is the rail's own ladder — ▲ at the top, ▼ at the
	// bottom, the thumb somewhere between.
	//
	// THE CAPS ARE THE CALLER'S, WHICH IS `Railify`'S OWN CONTRACT: "callers
	// draw ▲/▼ themselves … a caller that draws ▼ on its last visible row passes
	// the rows above it". So Railify is asked only for the TRACK between them,
	// and it stays the one owner of where the thumb lands (HUM LEAD, UAT
	// 2026-09-10: "the vertical control should start and end where the mock
	// says").
	// THE MARK COLUMN BELONGS TO THE SCROLL RAIL, AND ONLY TO IT (D-85, HUM
	// LEAD 2026-09-11): "We need to remove the extra lines on the right side of
	// the UI next to LIVE and UP NEXT."
	//
	// It was the wall on every row, scrolling or not — a second vertical beside
	// two regions that have nothing to scroll, which reads as a column that
	// stopped rather than as one that was never there. The reference draws the
	// read regions one vertical narrower than the ones below them.
	marks := make([]string, len(body))
	for i := range marks { // bounded by the body (P10-02)
		marks[i] = " "
		if rail {
			marks[i] = g.Rail
		}
	}
	if rail && len(body) >= 3 {
		marks[0], marks[len(body)-1] = glyphs.Up, glyphs.Down
		// Width 1 over empty lines asks Railify for the GLYPHS and nothing else:
		// `PadTo("", 0)` is empty, so each line it returns is the mark alone.
		for i, m := range render.Railify(make([]string, len(body)-2), 1, 0,
			max(total, 1), max(shown, 1), glyphs) {
			marks[i+1] = m
		}
	}
	out := make([]string, len(body))
	for i, r := range body { // bounded by the body (P10-02)
		mark := marks[i]
		// A BREAK IS A BREAK, ON BOTH SIDES (HUM LEAD, UAT 2026-09-10): "the
		// blank row in between sections needs to be completely blank — right now
		// the left rail is connected top to bottom; the breaks in the mock were
		// intentional." A row of the running order that is entirely blank IS the
		// separator between two regions, so the inner wall stops with the rail
		// column beside it — the frame's outer edge carries on, which is what
		// the reference draws.
		//
		// THE SCROLL RAIL'S END CAPS ARE THE EXCEPTION, and they are exactly why
		// they sit on blank rows: ▲ and ▼ say where the scrolling region begins
		// and ends, which is a thing to say IN the break rather than despite it.
		if strings.TrimSpace(r) == "" && mark != glyphs.Up && mark != glyphs.Down {
			mark = " "
		}
		out[i] = render.PadTo(r+"   ", b.width-6) + mark + "    " + g.Rail
	}
	return out
}
