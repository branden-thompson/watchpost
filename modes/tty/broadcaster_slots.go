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
			rows = append(rows, lane.boxOf(c, handle, "STANDARD", b.readBody(o, c, handle, decided))...)
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
// THERE IS NO SCRIPT TO SHOW YET, AND THAT IS WIRING, NOT LAYOUT (F-84). The
// script is built into `Built{ID, Script}` inside the executors and never
// reaches the card or the console; `Card` carries a Headline and no words. The
// window is built to the reference's size so the words have somewhere to land.
func (b Broadcaster) readBody(o render.Opts, c lineup.Card, handle string, decided bool) []string {
	rows := make([]string, 0, bcReadLines+2)
	for range bcReadLines { // bounded by the card's height (P10-02)
		rows = append(rows, "")
	}
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
