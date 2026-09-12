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
	i -= b.liveOffset()
	if i >= 0 && i < len(cards) {
		return cards[i], true
	}
	return lineup.Card{}, false
}

// liveOffset is how far the line-up sits below the LIVE slot (D-84).
//
// LIVE IS WHAT IS ON THE AIR, AND ON STANDBY NOTHING IS. The HUM LEAD:
//
//	"LIVE should remain EMPTY … The UP NEXT card should be getting the attention
//	 of the Composer … it will be the first thing that goes ON AIR when the human
//	 operator hits SHIFT+ENTER."
//
// So a station on standby draws its line-up from UP NEXT down, with the LIVE
// slot empty and waiting. The instant the operator goes on air the head of the
// queue takes the air and everything moves up one — which is the same movement
// D-40 already rules for a dropped card, so the operator has seen it before.
//
// IT IS THE POWER THAT DECIDES, NOT WHETHER A CARD HAPPENS TO BE ON THE AIR. A
// running station between two reads — the next card still building — has nothing
// on the air for a moment, and keying this off that would shunt every card down
// a row and back for the length of one build. The power is stable; "is something
// playing right now" is not.
func (b Broadcaster) liveOffset() int {
	if b.power == lineup.Running {
		return 0
	}
	return 1
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
			// AND A READ SLOT ON A STATION AT REST GETS THE STANDBY BOX (D-89).
			// The HUM LEAD: "when it's on standby — like it is on first open/play
			// — that should be blank, we should have an empty state for that live
			// slot", and then the design for it: "a grey box with a centered text
			// of: NO REPORTS READ OR ACTIVE IN STANDBY MODE". A shimmer there
			// would promise a read that is not coming, because nothing is going
			// to air at all.
			//
			// IT IS NOT A CARD, SO IT IS NOT DRAWN AS ONE. Handing this path a
			// `lineup.Card{}` gave it a zero Slot — which IS a location report —
			// and the box came out titled `LOCATION REPORT •STANDARD• [0]`: a
			// report that does not exist, graded, with a chip that opens nothing.
			//
			// THE LIVE SLOT ALONE, AND NOT EVERY READ SLOT. The first draft gave
			// the notice to both, and a test caught it: an empty UP NEXT on a
			// station that has just opened is the Director still choosing, which
			// is what the SHIMMER says. Telling the operator "no reports read or
			// active" about the slot the Composer is working on right now would be
			// the console reporting an absence where there is work in progress.
			//
			// LIVE IS POSITION 0 BY DEFINITION, which is what `liveOffset` exists
			// to guarantee: on standby the line-up is drawn from UP NEXT down
			// precisely so that nothing can be in slot 0 until the operator goes
			// on air.
			if i == 0 && b.power != lineup.Running {
				rows = append(rows, lane.standbyBox(bcReadCardRows)...)
				continue
			}
		}
		if r.reads {
			rows = append(rows, lane.boxOf(c, handle, "STANDARD", b.readBody(o, lane, c, handle, decided))...)
			continue
		}
		rows = append(rows, lane.boxOf(c, handle, "STANDARD", b.flatBody(o, c, handle, decided))...)
	}
	return rows
}

// bcReadLines is how many rows a read card gives its manifest.
//
// FOUR, FROM THE v2 REFERENCE, which lists a location report's four sources —
// the forecast, the marine report, the fire report and the seismic report. It is
// a MANIFEST and never the read itself (D-87, HUM LEAD 2026-09-11): "The full
// script on the top level card doesn't make sense when I can drill down to read
// the whole thing. What DOES MAKE SENSE for the operator is to get a SUMMARY of
// what the report contains *before* it goes on air."
const bcReadLines = 4

const (
	// bcFlatCardRows is a card the operator only ORDERS (D-87): border, title,
	// its status, a blank, its details control, border.
	bcFlatCardRows = 6

	// bcReadCardRows is a card the operator READS FROM: the flat card's header
	// plus the data stamp, the manifest and its heading, and the contents.
	bcReadCardRows = bcFlatCardRows + 3 + bcReadLines
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
	if !decided {
		rows := []string{""}
		// AN EMPTY SLOT HAS NOTHING TO MANIFEST, and drawing the headings over
		// nothing would promise contents that are not coming.
		for range bcReadCardRows - bcFlatCardRows + 1 {
			rows = append(rows, "")
		}
		return append(rows, "")
	}
	rows := []string{bcCardInset + "STATUS:  " + cardStatus(c, decided)}
	rows = append(rows, bcCardInset+cardPulled(o, c, b.clock, lane.inner()-2*len(bcCardInset)), "",
		bcCardInset+"READ CONTENTS", bcCardInset+manifestHeading(lane.inner()-2*len(bcCardInset)))
	rows = append(rows, b.manifestRows(c, lane.inner()-2*len(bcCardInset))...)
	return append(rows, "", bcCardInset+" "+o.KeyCap(handle)+"  Report Details")
}

// flatBody is the interior of a card the operator only ORDERS (D-87): what it is
// waiting for, and the way in to the rest of it.
func (b Broadcaster) flatBody(o render.Opts, c lineup.Card, handle string, decided bool) []string {
	if !decided {
		return []string{"", "", ""}
	}
	return []string{
		bcCardInset + "STATUS:  " + cardStatus(c, decided),
		"",
		bcCardInset + " " + o.KeyCap(handle) + "  Report Details",
	}
}

// bcCardInset is where a read card's own text begins, counted off the reference.
const bcCardInset = "   "
