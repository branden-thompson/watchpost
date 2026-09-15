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

// `slotRows` RETIRED WITH THE CARD COLUMN (D-110). It drew a REGION of the
// running order as a column of boxes — the shape D-94 replaced with a table and
// D-97 replaced above it with the UP NEXT / takeover pair. Nothing but its own
// tests had called it since; two drawers of one running order is exactly what the
// `dupes` gate exists for, and this one had already stopped being called.

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
		// AN EMPTY SLOT HAS NOTHING TO MANIFEST, and drawing the headings over
		// nothing would promise contents that are not coming.
		//
		// IT STILL CARRIES ITS HANDLE, WHICH IS F-97 (D-110). The chip used to ride
		// the title row and an undecided slot got one anyway; the way in moved to
		// the footer, and a first draft gave the footer only to a decided card —
		// which would have made the slot unaddressable again, one row along.
		rows := make([]string, bcReadCardRows-1)
		for i := range rows { // bounded by the card's height (P10-02)
			rows[i] = ""
		}
		return append(rows, b.cardControls(o, handle))
	}
	room := lane.inner() - 2*len(bcCardInset)
	// THE REFERENCE'S OWN ORDER (D-110): air, what it is doing and how fresh it
	// is, air, the manifest under a centred caption, and the way in at the bottom.
	//
	// THE AIR IS PART OF IT. The card is the one thing on this console an operator
	// READS rather than scans, and the reference gives its three questions room to
	// be separate answers instead of a block of labels.
	rows := []string{
		"",
		bcCardInset + "STATUS: " + cardStatus(c, decided),
		bcCardInset + "DATA PULL: " + cardPulledShort(o, c, b.clock),
		"",
		// "READ MANIFEST", CENTRED, and it is a CAPTION over the table rather than
		// a row of it — which is what the centring says and what "READ CONTENTS"
		// hard against the left margin did not.
		// CENTRED OVER THE CELL THE ROW IS DRAWN IN, not over the lane's interior.
		// The rows this returns are padded to the CELL's width by the box around
		// them, so centring two cells short put the caption three cells left of
		// the middle — visible, and the kind of thing only a measurement catches.
		centerText(bcManifestCaption, lane.lane),
		bcCardInset + manifestHeading(room),
	}
	rows = append(rows, b.manifestRows(c, room)...)
	for len(rows) < bcReadCardRows-1 { // bounded by the card's height (P10-02)
		rows = append(rows, "")
	}
	return append(rows, b.cardControls(o, handle))
}

// bcManifestCaption is what the reference calls the contents table.
//
// "READ MANIFEST", NOT "READ CONTENTS" (D-110). The word is the HUM LEAD's own
// from D-87 — "a SUMMARY of what the report contains *before* it goes on air" —
// and a manifest is exactly that: a list of what is aboard, not the cargo.
const bcManifestCaption = "READ MANIFEST"

// cardControls is the row along the bottom of a read card: the way in.
//
// THE PRESENTER IS GONE WITH THE CONTROL IT BELONGED TO (D-131). The v3 plan's
// item 6 was "the per-card PRESENTER control"; that was dropped, and the row
// went on drawing `PRESENTER: N/A` — the label of a control that no longer
// exists, reporting nothing, on every card — HUM LEAD, UAT 2026-09-14: "since
// we removed per card presenters, we need to remove the PRESENTER - N/A from
// the Up Next Card."
//
// THE VOICE ITSELF IS NOT GONE, and this is the distinction worth keeping.
// `Card.ReadBy` is still resolved from the role cast and is still SAID where it
// is a fact rather than a control: the LIVE NOW row names who is presenting the
// card on air, and the detail window carries READ BY. What was removed is the
// per-card OVERRIDE and its label, not the answer to "who reads this".
func (b Broadcaster) cardControls(o render.Opts, handle string) string {
	return bcCardInset + " " + o.KeyCap(handle) + "  Read / Manage"
}

// `flatBody` RETIRED WITH THE CARD COLUMN (D-110). It was the interior of a slot
// the operator only ORDERS, and those are rows of `LineupTable` now (D-94) — a
// table cell has no interior to draw.

// bcCardInset is where a read card's own text begins, counted off the reference.
const bcCardInset = "   "
