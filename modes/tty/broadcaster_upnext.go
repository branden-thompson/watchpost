package tty

// broadcaster_upnext.go — the beat that is next, and the hazard that would
// interrupt it, side by side (D-97).
//
// THE v3 MOCK PUTS THEM LEVEL. UP NEXT keeps the manifest it gained at D-87 and
// gains a label cell of its own; the takeover sits beside it at the same height.
//
// BOTH ARE ALWAYS DRAWN (HUM LEAD, 2026-09-12: "Up Next and Alert are always
// present — if there are no active alerts taking over, then the box is simply
// empty").  A box that appeared when a hazard arrived would move the card the
// operator is reading at the worst possible moment, which is the same argument
// `priorityWidth` already made for reserving the column.
//
// THE VERTICAL RAIL RETIRES WITH THIS.  What `railColumn` spelled in single
// letters down the side, the label cell says once — and the cells it frees are
// cells the manifest can use.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

const (
	// bcUpNextLabelW is the label cell's width, from the reference.
	bcUpNextLabelW = 14
	// bcUpNextLabel and bcAlertLabel are what the two boxes call themselves.
	bcUpNextLabel = "UP  NEXT"
)

// readPair is UP NEXT and the takeover, level with each other.
func (b Broadcaster) readPair() []string {
	left := b.upNextBox()
	right := b.alertBox(len(left))
	gap := strings.Repeat(" ", bcColumnGap)
	lw, rw := b.upNextWidth(), b.priorityWidth()

	return joinColumns(left, right, lw, rw, gap)
}

// joinColumns lays two boxes side by side, each held to its OWN width.
//
// EVERY CELL IS PADDED AND TRUNCATED TO ITS COLUMN, and that is the rule: a row
// that comes in short pulls its neighbour leftward on that row alone, which is
// occlusion arriving by accident rather than by design. HUM LEAD, 2026-09-11:
// "no more card occlusion."
//
// IT IS A FUNCTION SO THE RULE CAN BE TESTED, and that is the whole reason it
// moved out of `readPair`. Both boxes are built by `shell`, which already pads
// every row to the box's own width — so with today's callers the pad here is a
// no-op and mutant mS0 removed it without changing a single frame. The rule it
// guards is a width DISAGREEMENT between the box and the join, which is the
// defect `withControl` actually shipped (a second copy of the air box's width,
// and the `b` chip fell off the row). A guard whose fault no caller can express
// is still a guard; it just cannot be tested THROUGH those callers.
func joinColumns(left, right []string, lw, rw int, gap string) []string {
	n := max(len(left), len(right))
	out := make([]string, 0, n)
	for i := range n { // bounded by the taller box (P10-02)
		at := func(col []string, w int) string {
			if i < len(col) {
				return render.PadTo(render.TruncateCells(col[i], w), w)
			}
			return strings.Repeat(" ", w)
		}
		out = append(out, at(left, lw)+gap+at(right, rw))
	}
	return out
}

// upNextWidth is what the left box gets: everything the alert box and the gap
// between them do not take.
func (b Broadcaster) upNextWidth() int {
	return max(0, b.frameWidth()-b.priorityWidth()-bcColumnGap)
}

// upNextBox is the next beat: a label cell, and the card's own manifest beside it.
func (b Broadcaster) upNextBox() []string {
	o := b.opts()
	bx := render.HeavyBox(b.ascii)
	inner := b.upNextWidth() - 2
	body := inner - bcUpNextLabelW - 1
	if body < 20 {
		return nil
	}

	c, decided := b.slotCard(b.mainTrack(), 1)
	// THE LANE IS THE CELL'S OWN WIDTH. A first draft built it two cells wider —
	// "the lane counts its own rails" — and the row was then truncated back into
	// the cell, which cut exactly the two cells the handle's CHIP sits in. The
	// rails belong to the box this draws INSIDE, not to the lane.
	lane := newCardLane(body, o.Glyphs())
	if !decided {
		c = lineup.Card{Headline: b.waiting(o)}
	}
	// THE CARD NAMES ITSELF IN ITS OWN BORDER (D-110), which is where the
	// reference puts it: `┏━━ LOCATION REPORT • Oceanside, CA 92057 ━━━ • STANDARD
	// • ━━━┓`. It used to be a ROW inside the box, and that row cost the manifest
	// a line to say what the frame around it could say for free.
	//
	// AND THE HANDLE LEAVES THE TITLE WITH IT. The chip rode the title row; the
	// reference puts the way in at the BOTTOM, beside the presenter — one row for
	// what the operator DOES with this card, rather than a key in the caption.
	rows := b.readBody(o, lane, c, "1", decided)

	// THE LABEL SITS AGAINST THE CARD'S MIDDLE, which is where the reference puts
	// it — a caption beside a tall cell, not a heading over it.
	// THE BOX HAS A GROUND OF ITS OWN (D-114, HUM LEAD 2026-09-13: "the UP Next
	// Box probably needs a bkg color other than none - I suggest the same Blue as
	// the modal for now").
	//
	// "FOR NOW" ENDED ON 2026-09-15, exactly as that note anticipated: "let's
	// make the cell background color of the UP NEXT box the same 'blue' token
	// color used as the bkg for 'DIRECTION' and 'TODAY' column - this will help
	// add a bit of visual distinction that will also be themeable."
	//
	// TWO GROUNDS, NOT ONE, AND THAT IS THE CORRECTION (D-136). The first pass
	// read "the cell background color of the UP NEXT box" as the BOX and painted
	// all of it — label and report together — in the bands' blue and the bands'
	// bold white. HUM LEAD, UAT 2026-09-15, with a diagram: the LABEL CELL is
	// TODAY BLUE; the report beside it is "STANDARD MODAL BLUE", and its
	// "content ... should not be all bold and white, but the standard text
	// color".
	//
	// THE WORD WAS "CELL" AND IT MEANT CELL. A label cell beside a report is
	// exactly the shape `D I R E C T I O N` has beside its rows, which is why
	// that band was the colour named — the point was the CELL, not the box.
	//
	// `GroupTodayBG` IS THAT BLUE, AND IT IS THE SAME OBJECT — not a colour
	// matched by eye. It is the ground under `D I R E C T I O N` in the console's
	// own line-up table (render/lineup_table.go) and under `T O D A Y` in
	// Observer's (render/table.go), so the three cannot drift and a theme moves
	// all of them together. `GroupText` comes with it: the tone every band in the
	// app carries its text in, and `aaPairs` already registers {GroupText, bands},
	// so the cell inherits a contrast answer measured in every theme.
	labelGround := render.Tok(render.GroupText) + ";" + render.Tok(render.GroupTodayBG)
	fg, bg := render.ModalTone(b.darkBG)
	ground := fg + ";" + bg

	out := []string{bx.TL + strings.Repeat(bx.Rule, bcUpNextLabelW) + bx.T +
		boxRule(bx.Rule, cardRuleTitle(c, o.Glyphs()), lane.badgeOf("STANDARD"), body) + bx.TR}
	at := (len(rows) - 1) / 2
	for i, r := range rows { // bounded by the card's own height (P10-02)
		cell := strings.Repeat(" ", bcUpNextLabelW)
		if i == at {
			// BOLD WHITE, RULED (HUM LEAD, 2026-09-14): "Let's make 'UP NEXT' in
			// the up next box BOLD WHITE so it contrasts a bit more." The box
			// wears the modal's blue ground (D-114) and the caption was plain
			// text on it, which at this size reads as part of the fill.
			//
			// THE TOKEN, NOT "white". `TextBright` is what this app already
			// calls emphasized plain text, so the caption moves with the theme
			// — the Light theme's bright is not the dark one's, and a literal
			// here would be a second answer to a question the theme owns.
			//
			// THE CELL'S WIDTH IS UNCHANGED, and the reason is `PadTo` and
			// `centerText` both measuring with `displayWidth` — escape codes
			// are skipped, so styling cannot inflate the pad and shift the
			// card's column. Checked rather than assumed: the mutation that
			// pads AFTER tinting changes no frame at all, so it was written and
			// then DISCARDED rather than committed as a rule nothing measures.
			cell = render.Bold(render.Tint(
				render.PadTo(centerText(bcUpNextLabel, bcUpNextLabelW), bcUpNextLabelW),
				render.Tok(render.TextBright)))
		}
		// THE CELL IS PAINTED BEFORE THE ROW IS, and the row's own paint keeps
		// it: `TintKeeping` rewrites an inner RESET to the outer ground, and the
		// cell's opening escape is not a reset — so the label block survives and
		// the report beside it returns to the modal tone at the rail.
		out = append(out, bx.Rail+render.TintKeeping(cell, labelGround)+bx.Rail+
			render.PadTo(render.TruncateCells(r, body), body)+bx.Rail)
	}
	out = append(out, bx.BL+strings.Repeat(bx.Rule, bcUpNextLabelW)+bx.B+strings.Repeat(bx.Rule, body)+bx.BR)
	// THE WHOLE BOX IS PAINTED, BORDERS INCLUDED — the rule `shell` states for a
	// card (D-86): "a ground that stopped at the border would draw a coloured
	// window inside a colourless frame, which reads as a fill rather than as a
	// card". `TintKeeping` so the chips and the badge keep their own colours.
	for i, r := range out { // bounded by the box's height (P10-02)
		out[i] = render.TintKeeping(r, ground)
	}
	return out
}

// alertBox is the takeover, and it is drawn whether or not one is happening.
//
// AN EMPTY BOX IS THE POINT. The operator learns where a hazard WILL appear
// before one does, and the frame does not move when it arrives.
func (b Broadcaster) alertBox(rows int) []string {
	w := b.priorityWidth()
	if w < 8 || rows < 3 {
		return nil
	}
	lane := newCardLane(w, b.opts().Glyphs())
	// THE HEAD OF THE RAIL, AND ONLY IT. A burst is ONE card (MVS-D-77), so the
	// box draws the one that would interrupt the programme next — written as an
	// index rather than a loop that always returns, which staticcheck rightly
	// reads as a loop that is not one.
	rail := b.lineup.Projection(lineup.AlertRail)
	if len(rail) == 0 {
		return b.emptyAlertBox(lane, rows)
	}
	// THE LIST FILLS THE HEIGHT THE CARD BESIDE IT SETS, so the two boxes close on
	// the same row. Truncating afterwards cut the control row off the bottom —
	// which is the one thing in the box the operator presses.
	out := lane.boxOf(rail[0], "PRIORITY", b.burstBody(rail[0], w, rows-bcAlertChrome))
	if len(out) > rows {
		out = out[:rows] // it never grows the frame (FR-7.3)
	}
	return out
}

// emptyAlertBox is the takeover's box with nothing in it.
func (b Broadcaster) emptyAlertBox(lane cardLane, rows int) []string {
	body := make([]string, max(0, rows-2))
	for i := range body { // bounded by the box's height (P10-02)
		body[i] = ""
	}
	return lane.boxOf(lineup.Card{Slot: lineup.BreakingAlert}, "PRIORITY", body)
}
