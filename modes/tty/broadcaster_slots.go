package tty

// broadcaster_slots.go — the console draws its ten slots always (D-64).
//
// HUM LEAD, UAT 2026-09-10: "render the full Broadcaster Dashboard, including
// all 10 cards … Cards should SHIMMER while the producers are proposing and the
// director is deciding … Right now this looks broken, so if I was a user who
// came upon this, I would not expect this to be working as is."
//
// "(nothing scheduled)" WAS A DEAD END. It said the station had nothing and gave
// the operator nowhere to look. Ten shimmering slots say the machine is working
// and the data is coming — which is what Observer has said since UAT 18.2, so
// this is the console ADOPTING that rather than inventing a second answer.

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// slotCard is what the Director has put in slot i, and whether it has decided
// yet.
//
// AN UNDECIDED SLOT IS NOT AN ERROR. It is the ordinary state of a station that
// has just started: the Producer is proposing and the Director is choosing, and
// the console's job is to say so rather than to report an absence.
func (b Broadcaster) slotCard(cards []lineup.Card, i int) (lineup.Card, bool) {
	if i < len(cards) {
		return cards[i], true
	}
	return lineup.Card{}, false
}

// waiting is the placeholder for a slot the Director has not filled — the same
// shimmer Observer draws, so the two surfaces cannot come to mean different
// things by "still loading".
func (b Broadcaster) waiting(o render.Opts) string {
	return "waiting for the line-up " + o.LoadingDots()
}

// tickNeeded reports whether anything on the console is still animating.
//
// ONLY WHILE A SLOT IS EMPTY. A full line-up is a still frame, and a tick that
// kept firing would redraw a console nobody is watching change — the same
// discipline Observer's own `tickNeeded` applies.
func (b Broadcaster) tickNeeded() bool {
	return len(b.lineup.Projection(lineup.MainTrack)) < MainTrackSlots
}

// armTick starts the shimmer when the frame needs one and none is in flight —
// the console's own twin of the Dashboard's.
func (b Broadcaster) armTick(cmd tea.Cmd) (Broadcaster, tea.Cmd) {
	b.tickArmed, cmd = armShimmer(b.tickArmed, b.tickNeeded(), cmd)
	return b, cmd
}

// slotRows is every slot in a region, decided or waiting.
func (b Broadcaster) slotRows(r bcRegion, cards []lineup.Card, lane cardLane) []string {
	o := b.opts()
	rows := []string{}
	for i := r.from; i < r.upto; i++ { // bounded by the region (P10-02)
		handle := strconv.Itoa(i)
		c, decided := b.slotCard(cards, i)
		if !decided {
			// THE HANDLE IS DRAWN ON AN EMPTY SLOT TOO: it is addressable — the
			// operator can put something in it — so it carries its address.
			c = lineup.Card{Headline: b.waiting(o)}
			// AND A READ SLOT ON A STATION AT REST IS SIMPLY EMPTY (D-68). The
			// HUM LEAD: "when it's on standby — like it is on first open/play —
			// that should be blank, we should have an empty state for that live
			// slot." A shimmer there would promise a read that is not coming,
			// because nothing is going to air at all.
			if r.reads && b.power != lineup.Running {
				c = lineup.Card{}
			}
		}
		if r.reads {
			rows = append(rows, lane.boxOf(c, handle, "STANDARD", b.readBody(o, lane, c, handle, decided))...)
			continue
		}
		rows = append(rows, lane.box(c, handle, "STANDARD")...)
	}
	return rows
}

// bcReadLines is how many lines of the script a read card shows.
//
// FIVE, FROM THE REFERENCE — its LIVE card runs eleven rows: the border, the
// title, a blank, five of script, a blank, the controls, the border. It is a
// WINDOW onto the read and never the whole of it, which is what `Details (Full
// Read)` is for.
const bcReadLines = 5

const (
	// bcFlatCardRows is a card the operator only ORDERS: border, title, a row,
	// border.
	bcFlatCardRows = 4

	// bcReadCardRows is a card the operator READS FROM: the flat card plus the
	// script window, a blank, and the card's own controls.
	bcReadCardRows = bcFlatCardRows + bcReadLines + 2
)

// readBody is the interior of a card the operator reads from, below the row the
// test marks live on: the script, a blank, and the card's own controls.
//
// THE HEIGHT IS THE SAME WHETHER THERE IS A SCRIPT OR NOT. A card that grew when
// its words arrived would move every card below it at the moment the operator is
// reading one — so an empty read card is the same shape with nothing in it.
//
// THE SCRIPT REACHES IT (F-84, closed at D-83), AND THE ROW THAT FILED IT WAS
// HALF WRONG. It read "the script is built into `Built{ID, Script}` inside the
// executors and never reaches the card or the console; `Card` carries a Headline
// and no words" — true when it was written, and overtaken at T3.8: `Card.Script`
// has carried the words since, `Projection` returns whole cards, and `Publish`
// hands the console the lineup. The words were already here. What was missing
// was the drawing.
func (b Broadcaster) readBody(o render.Opts, lane cardLane, c lineup.Card, handle string, decided bool) []string {
	rows := b.scriptWindow(o, lane, c)
	rows = append(rows, "")
	if !decided {
		return append(rows, "") // nothing to detail, and the shape holds
	}
	// THE CONTROL IS THE CARD'S OWN HANDLE, which is the reference: `[ 0 ]
	// Details (Full Read)`. The operator types the address they can already see
	// on the card, so there is one number to learn per card rather than two.
	return append(rows, bcCardInset+o.KeyCap(handle)+" Details (Full Read)")
}

// bcCardInset is where a read card's own text begins, counted off the reference.
const bcCardInset = "   "

// scriptWindow is the card's words as the reference draws them: `bcReadLines` of
// wrapped script, inset to the same column the card's own control sits at.
//
// A WINDOW, AND THE OPERATOR KNOWS WHERE THE REST IS (HUM LEAD, 2026-09-11):
// "The 5 line preview is fine - the operator should be able to inspect the full
// text of the report by keying the number position of either the live card [0]
// or the UP Next [1] card." So this shows the OPENING and never apologises for
// the length; `[ n ] Details (Full Read)` is the way in, and it sits two rows
// below.
//
// AND IT SAYS WHEN THERE IS MORE, on the last line it has room for. A window
// with no sign of its own edge reads as a short report — which on a station is
// the difference between "that is all it says" and "that is all it fits". The
// ellipsis is the glyph set's, so it degrades to "..." with the rest of the
// console (A11Y) rather than inventing a mark of its own.
//
// THE HEIGHT IS FIXED regardless, which is readBody's rule one level up: fewer
// lines are padded, more are cut, and no card changes shape when its words land.
//
// UP NEXT GETS IT TOO, and that falls out rather than being added. D-68 gives
// slots 0 and 1 the tall box, and a card's words are composed AT STANDBY (DR-7)
// — so the operator can read one card ahead of the one on the air.
func (b Broadcaster) scriptWindow(o render.Opts, lane cardLane, c lineup.Card) []string {
	rows := make([]string, 0, bcReadLines+2)
	room := lane.inner() - 2*len(bcCardInset)
	var wrapped []string
	if room > 0 {
		lines := make([]string, 0, len(c.Script.Parts))
		for _, p := range c.Script.Parts { // bounded by the script (P10-02)
			lines = append(lines, p.Text)
		}
		wrapped = render.WrapLines(lines, room)
	}
	for i := range bcReadLines { // bounded by the card's height (P10-02)
		if i >= len(wrapped) {
			rows = append(rows, "")
			continue
		}
		line := wrapped[i]
		if i == bcReadLines-1 && len(wrapped) > bcReadLines {
			line = more(o, line, room)
		}
		rows = append(rows, bcCardInset+line)
	}
	return rows
}

// more marks a line as the last one that fits, with words still to come.
//
// IT IS SEPARATED FROM THE WORDS WHERE THERE IS ROOM, and that is not cosmetic.
// The mark means "there is MORE BELOW", not "this line continues" — and set
// flush against a sentence that ended cleanly it reads as a typo, which is
// exactly what the first draft produced: "…a high near sixty-eight tomorrow.…".
// Caught by looking at the rendered card rather than at the test, which is the
// only way that class of defect is ever caught.
//
// IT CUTS ONLY WHEN IT MUST, and then it sets the mark FLUSH — a line that had
// to be shortened genuinely does continue, so the two cases read differently on
// purpose. Cutting at all is the fallback because the box truncates what it is
// given, and a tail appended past the edge would be the one thing the reader
// needs and the one thing removed.
func more(o render.Opts, line string, room int) string {
	tail := o.Glyphs().Ellipsis
	if render.Width(line)+1+render.Width(tail) <= room {
		return line + " " + tail
	}
	keep := room - render.Width(tail)
	if keep < 1 {
		return line // no room to say it; the control two rows down still does
	}
	return render.TruncateCells(line, keep) + tail
}
