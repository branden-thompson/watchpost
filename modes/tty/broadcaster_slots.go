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
		c, decided := b.slotCard(cards, i)
		if decided {
			rows = append(rows, lane.box(c, strconv.Itoa(i), "STANDARD")...)
			continue
		}
		// THE HANDLE IS DRAWN ON AN EMPTY SLOT TOO: it is addressable — the
		// operator can put something in it — so it carries its address.
		waiting := lineup.Card{Headline: b.waiting(o)}
		rows = append(rows, lane.box(waiting, strconv.Itoa(i), "STANDARD")...)
	}
	return rows
}
