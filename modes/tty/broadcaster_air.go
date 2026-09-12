package tty

// broadcaster_air.go — what is on the air, and what is under it (D-95).
//
// THE v3 MOCK PUTS THEM IN ONE BOX, two rows with a label column:
//
//	┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
//	┃  ⏺ LIVE NOW    ┃   LOCATION REPORT • Oceanside, CA … 0  Full Read  ┃
//	┣━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
//	┃  ○ RELAY BED   ┃   [←] KIG78 Coachella CA … [→]        ACTIVE   b  ┃
//	┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
//
// THEY BELONG TOGETHER BECAUSE THEY ARE ONE QUESTION: what is going out. The
// line-up and the bed are mutually exclusive by product rule (D-11, FR-4.2) and
// by the engine having one source, so a box that holds both and marks which is
// live says the exclusivity in its shape.
//
// IT REPLACES THE LIVE CARD (D-89 retires with it). A slot the station reads from
// needed a manifest when it was one of ten cards; as one row of two it needs to
// say what is on the air and how to reach it.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// bcAirLabelW is the label column's width, from the reference.
const bcAirLabelW = 16

// airBox is the LIVE NOW / RELAY BED pair.
func (b Broadcaster) airBox() []string {
	o := b.opts()
	g := o.Glyphs()
	bx := render.HeavyBox(b.ascii)
	inner := b.bandWidth() - 2
	body := inner - bcAirLabelW - 1
	if body < 12 {
		return nil // no room to say anything useful; the notice covers this (FR-7.3)
	}

	rule := func(l, mid, r string) string {
		return l + strings.Repeat(bx.Rule, bcAirLabelW) + mid + strings.Repeat(bx.Rule, body) + r
	}
	row := func(mark, label, content string) string {
		cell := render.PadTo(render.TruncateCells("  "+mark+" "+label, bcAirLabelW), bcAirLabelW)
		return bx.Rail + cell + bx.Rail + render.PadTo(render.TruncateCells(content, body), body) + bx.Rail
	}

	// THE MARK SAYS WHICH IS CARRYING, which is the exclusivity drawn rather
	// than described: one of these two is the programme and the other is not.
	// `Idle` and `Live` ARE THE STATE MARKS (D-62), and they already mean exactly
	// this: "a thing's own state where it is NAMED". Borrowing another ramp here
	// would give one glyph two meanings, which is the mistake that comment
	// records an early draft making.
	liveMark, bedMark := g.Idle, g.Idle
	if b.bed.Carrying {
		bedMark = g.Live
	} else if b.power == lineup.Running {
		liveMark = g.Live
	}

	return []string{
		rule(bx.TL, bx.T, bx.TR),
		row(liveMark, "LIVE NOW", b.liveLine(o)),
		rule(bx.L, bx.X, bx.R),
		row(bedMark, "RELAY BED", b.bedLine(o)),
		rule(bx.BL, bx.B, bx.BR),
	}
}

// liveLine is what is on the air, and the key that opens it.
//
// THE EMPTY STATE IS A SENTENCE, NOT A BOX (D-95, replacing D-89's). The HUM
// LEAD's standby wording was written for a card thirteen rows tall; as one row it
// says the same thing in the space it has.
func (b Broadcaster) liveLine(o render.Opts) string {
	c, decided := b.slotCard(b.mainTrack(), 0)
	if !decided {
		return bcCardInset + bcStandbyNotice
	}
	line := bcCardInset + cardTitle(c, o.Glyphs())
	if who := detailReadBy(c); who != "N/A" {
		line += "  " + o.Glyphs().Bullet + "  Presented by  " + who
	}
	return b.withControl(line, o.KeyCap("0")+"  Full Read")
}

// bedLine is the relay selector and whether the bed is carrying.
func (b Broadcaster) bedLine(o render.Opts) string {
	state := "INACTIVE"
	if b.bed.Carrying {
		state = "ACTIVE"
	}
	return b.withControl(bcCardInset+b.bedSelector(o), state+"   "+o.KeyCap("b"))
}

// withControl right-anchors a row's control against the box's own width, so the
// two rows' keys line up under each other however long their content is.
func (b Broadcaster) withControl(line, control string) string {
	body := b.bandWidth() - 2 - bcAirLabelW - 1
	gap := body - render.Width(line) - render.Width(control) - len(bcCardInset)
	if gap < 1 {
		gap = 1
	}
	return line + strings.Repeat(" ", gap) + control
}

// mainTrack is the rolling window of the running order, as the frame draws it.
//
// ONE OWNER, because three things ask for it now — the air box, the UP NEXT card
// and the table — and a second `Projection(...)[:MainTrackSlots]` would be a
// second answer to how far the window reaches.
func (b Broadcaster) mainTrack() []lineup.Card {
	main := b.lineup.Projection(lineup.MainTrack)
	if len(main) > MainTrackSlots {
		main = main[:MainTrackSlots]
	}
	return main
}
