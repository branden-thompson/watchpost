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
	return railColumnToned(label, rows, g, railTone(label))
}

// railColumnToned is railColumn with the region's own ground painted behind its
// letters (D-86, HUM LEAD 2026-09-11): "Color BKGs for the left RAIL: LIVE =
// RED, UP NEXT = ORANGE, SCHEDULED LINEUP = BLUE."
//
// THE WALLS ARE LEFT UNPAINTED. They belong to the frame, which runs the whole
// height of the console; painting them would make the region's colour bleed
// into the structure around it and the band would read as a hole in the frame
// rather than as a label on it.
func railColumnToned(label string, rows int, g render.Glyphs, bg string) []string {
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
		band := " " + cell + " "
		if render.BGVisible(bg) {
			band = render.TintRaw(band, render.Tok(render.GroupText)+";"+bg)
		}
		out = append(out, g.Rail+band+g.Rail)
	}
	return out
}

// railTone is a region's ground, by name.
//
// KEYED ON THE LABEL, which is what the rail already is: `section` is handed a
// name and draws it, and the region table is the one place those names are
// declared. A parallel table keyed on the region's INDEX would be a second list
// to keep in step with `bcRegions`.
//
// SCHEDULED AND LINE UP SHARE ONE GROUND, because they are one stack of cards
// the rail happens to name in two halves — the same reason no break is drawn
// between them (HUM LEAD, UAT 2026-09-10: "this is one area they should be
// continuous").
//
// ANYTHING ELSE IS UNPAINTED. PRIORITY is the overlay's own rail and carries the
// takeover's colour, not a region's; the blank spacer has no region at all.
func railTone(label string) string {
	switch label {
	case "LIVE ON AIR":
		return render.Tok(render.RailLiveBG)
	case "UP NEXT":
		return render.Tok(render.RailNextBG)
	case "SCHEDULED LINE UP":
		return render.Tok(render.RailQueueBG)
	}
	return ""
}

const (
	// bcRailGap is the air between the rail's divider and a card's border, from
	// the reference: the divider sits at column 4 and the card begins at 9.
	bcRailGap = 4

	// bcRightChrome is the frame's own columns at the right: three of air and the
	// scroll rail, and nothing after it (D-87).
	//
	//	… ┛   │
	//	  144 148        (at the reference's 150)
	//
	// THE OUTER WALL IS GONE (HUM LEAD, 2026-09-11): "Notice the line on the far
	// right is gone — so ONLY the vertical scroll on positions 2-9 is present on
	// the right hand side." It carried an inner wall AND an outer one, which the
	// running order never needed: the cards are boxes with their own borders, so
	// a frame around them was a second edge saying the same thing.
	bcRightChrome = 5

	// bcPriorityWidth is the alert column's box, and it is DERIVED FROM WHAT IT
	// MUST HOLD rather than from a share of the frame.
	//
	// Its widest line is an alert's when and where —
	// `   <LOCATION> • <MM/DD> HH:MM - <MM/DD> HH:MM   ` — which is 54 cells at
	// the reference with the card's own inset on both sides. A column narrower
	// than its one fixed-shape line would wrap every alert in the burst.
	bcPriorityWidth = 54

	// bcColumnGap is the air between the two tracks' columns.
	bcColumnGap = 3
)

// cardBoxWidth is how wide a card's BOX is once the frame has taken its own
// columns — the ONE place that arithmetic lives (D-51's seam).
//
// At the reference's 150 columns this is 132, which puts the card's borders at
// 9 and 140 exactly where the mock draws them.
func (b Broadcaster) cardBoxWidth() int {
	w := b.trackArea() - b.priorityWidth() - bcColumnGap
	if w < 4 {
		return 0
	}
	return w
}

// trackArea is everything the two tracks have to share: the frame, less the
// rail on the left and the scroll on the right.
func (b Broadcaster) trackArea() int {
	w := b.width - bcRailWidth - bcRailGap - bcRightChrome
	if w < 0 {
		return 0
	}
	return w
}

// priorityWidth is the alert column's box (D-87).
//
// THE TRACKS ARE SPLIT, NOT COMPOSITED (HUM LEAD, 2026-09-11): "We're gonna
// split the PRIORITY and MAIN tracks visually — no more card occlusion, and this
// works because of the data that we're ACTUALLY showing."
//
// SO THE COLUMN IS ALWAYS RESERVED, whether or not the rail holds anything. A
// width that moved when a hazard arrived would shift every card in the running
// order sideways at the moment the operator is reading one — which is the same
// rule the LIVE slot's offset follows (D-84) and the same reason the read cards'
// height is fixed.
//
// FIXED AT ITS CONTENT'S WIDTH UNTIL THERE IS NOT ROOM, and then it gives way
// first: the main track is the thing the operator manages, so a frame too narrow
// for both cuts the column that is empty most of the time. Below half the track
// area the alert column takes half, which is the point where neither can be
// drawn honestly and the breakpoint ruling (D-50) takes over.
func (b Broadcaster) priorityWidth() int {
	area := b.trackArea() - bcColumnGap
	if area <= 0 {
		return 0
	}
	if w := bcPriorityWidth; w <= area/2 {
		return w
	}
	return area / 2
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
	"LIVE ON AIR":       {"LIVE ON AIR", "LIVE", "ON AIR", "AIR"},
	"UP NEXT":           {"UP NEXT", "NEXT", "UP"},
	"SCHEDULED LINE UP": {"SCHEDULED LINE UP", "SCHEDULED", "SCHED", "SCH"},
	// THE OLD NAMES KEEP THEIR LADDERS. D-87 merged the two queue regions into
	// one, and these are what the rail said before it — kept because the ladder
	// is a property of the WORD, not of the region that happens to use it, and
	// a caller naming one should not fall through to being cut.
	"SCHEDULED": {"SCHEDULED", "SCHED", "SCH"},
	"LINE UP":   {"LINE UP", "LINE", "UP"},
	"LIVE":      {"LIVE"},
	"BED":       {"BED"},
	// THE OVERLAY IS AS TALL AS THE TAKEOVER IT HOLDS, which for a single card
	// is four rows — too few for eight letters. It sheds to a word rather than
	// being cut: "PRIO" reads as damage, and this label appears exactly when
	// something is interrupting the broadcast.
	"PRIORITY": {"PRIORITY", "ALERT", "!"},
}

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
// chromeAt is `chrome` for the SCROLLING region, told where its window sits.
//
// THE THUMB TRACKS THE WINDOW, and until D-87 it could not: `chrome` passed a
// hard-coded `lo` of 0 to `Railify`, so the rail drew a thumb that never moved
// however far the operator scrolled. It was invisible while everything fitted on
// one screen, and became a lie the moment the cards outgrew the terminal.
func (b Broadcaster) chromeAt(body []string, off, total int) []string {
	return b.railed(body, true, off, 0, total)
}

func (b Broadcaster) chrome(body []string, rail bool, shown, total int) []string {
	return b.railed(body, rail, 0, shown, total)
}

func (b Broadcaster) railed(body []string, rail bool, lo, shown, total int) []string {
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
		//
		// THE CAPS ARE NOT PART OF THE WINDOW, and getting that wrong lost the
		// thumb at the bottom of the list. `Railify` places the thumb at
		// `lo*(window-1)/maxLo` and INDEXES ITS OWN `lines` with it, so its
		// contract is that the window IS the track it was handed — pass a window
		// two rows larger (the caps) and the last position falls off the end,
		// silently drawing no thumb at all. Caught by a test that scrolled to the
		// bottom and looked; invisible while everything fitted on one screen.
		track := len(body) - 2
		for i, m := range render.Railify(make([]string, track), 1, lo,
			max(total-2, 1), max(track, 1), glyphs) {
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
		// THREE OF AIR AND THE SCROLL, AND NOTHING AFTER IT (D-87). The frame's
		// outer wall on this side is gone: the cards are boxes with their own
		// borders, so a wall around them was a second edge saying the same thing.
		out[i] = render.PadTo(r+"   ", b.width-2) + mark
	}
	return out
}
