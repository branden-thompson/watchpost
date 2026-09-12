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
	// THE TITLE ROW GOES THROUGH THE LANE, not in as plain text: that is what
	// carries the badge and the handle's CHIP, and a card the operator cannot
	// address is the defect F-97 was filed for.
	rows := append([]string{lane.render(c, "1", "STANDARD")}, b.readBody(o, lane, c, "1", decided)...)

	// THE LABEL SITS AGAINST THE CARD'S MIDDLE, which is where the reference puts
	// it — a caption beside a tall cell, not a heading over it.
	out := []string{bx.TL + strings.Repeat(bx.Rule, bcUpNextLabelW) + bx.T + strings.Repeat(bx.Rule, body) + bx.TR}
	at := (len(rows) - 1) / 2
	for i, r := range rows { // bounded by the card's own height (P10-02)
		cell := strings.Repeat(" ", bcUpNextLabelW)
		if i == at {
			cell = render.PadTo(centerText(bcUpNextLabel, bcUpNextLabelW), bcUpNextLabelW)
		}
		out = append(out, bx.Rail+cell+bx.Rail+render.PadTo(render.TruncateCells(r, body), body)+bx.Rail)
	}
	return append(out, bx.BL+strings.Repeat(bx.Rule, bcUpNextLabelW)+bx.B+strings.Repeat(bx.Rule, body)+bx.BR)
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
	out := lane.boxOf(rail[0], "A", "PRIORITY", b.burstBody(rail[0], w))
	if len(out) > rows {
		out = out[:rows] // it never grows the frame (FR-7.3)
	}
	return out
}

// emptyAlertBox is the takeover's box with nothing in it.
func (b Broadcaster) emptyAlertBox(lane cardLane, rows int) []string {
	body := make([]string, max(0, rows-3))
	for i := range body { // bounded by the box's height (P10-02)
		body[i] = ""
	}
	return lane.boxOf(lineup.Card{Slot: lineup.BreakingAlert}, "A", "PRIORITY", body)
}
