package tty

// broadcaster_standbybox_test.go — D-89, the LIVE slot's empty state.
//
// HUM LEAD, 2026-09-11: "empty state needs to be a grey box with a centered text
// of: NO REPORTS READ OR ACTIVE IN STANDBY MODE".
//
// IT CLOSES D-84's POINT 4, which was the one part of that ruling left
// undesigned: "LIVE should remain EMPTY — we'll need to design an empty-state for
// that slot before when the Station is in STANDBY (DEAD AIR) mode."

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// liveRows is the LIVE slot's box, which is the first box of the running order.
func liveRows(t *testing.T, frame string) []string {
	t.Helper()
	rows := strings.Split(frame, "\n")
	for i, r := range rows {
		if strings.Contains(r, "LIVE") || strings.Contains(r, "│ L │") {
			return rows[max(0, i-2):min(len(rows), i+13)]
		}
	}
	t.Fatalf("the frame has no LIVE region, so this test measures nothing")
	return nil
}

func TestTheLiveSlotDrawsItsStandbyNoticeAtRest(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"), card(t, "b", "Bonsall, CA"))
	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, bcStandbyNotice) {
		t.Fatalf("the LIVE slot is empty at rest and says nothing about it:\n%s",
			strings.Join(liveRows(t, got), "\n"))
	}
	// CENTRED, which is the ruling's own word. The notice sits in the middle of
	// the box's width, so the gap before it and the gap after it match to within
	// the one cell an odd remainder leaves.
	for _, r := range liveRows(t, got) {
		if !strings.Contains(r, bcStandbyNotice) {
			continue
		}
		body := r[strings.Index(r, "┃")+len("┃") : strings.LastIndex(r, "┃")]
		lead := len(body) - len(strings.TrimLeft(body, " "))
		tail := len(body) - len(strings.TrimRight(body, " "))
		if d := lead - tail; d < -1 || d > 1 {
			t.Errorf("the notice is not centred: %d cells before, %d after\n%q", lead, tail, r)
		}
		return
	}
}

// AND CENTRED VERTICALLY TOO, which is the half a mutant found unpinned (mX2).
// "A centered text" in a box eleven rows tall means the middle row: pinned to the
// top it reads as a HEADING for contents that are not there, which is the one
// thing an empty state must not look like.
func TestTheStandbyNoticeIsCentredVertically(t *testing.T) {
	lane := newCardLane(80, render.Opts{}.Glyphs())
	rows := lane.standbyBox(bcReadCardRows)
	if len(rows) != bcReadCardRows {
		t.Fatalf("the standby box is %d rows, not %d", len(rows), bcReadCardRows)
	}
	// The interior, borders excluded — the rows the notice can sit in.
	body := rows[1 : len(rows)-1]
	at := -1
	for i, r := range body {
		if strings.Contains(stripANSITest(r), bcStandbyNotice) {
			if at >= 0 {
				t.Fatalf("the notice is drawn twice, at rows %d and %d", at, i)
			}
			at = i
		}
	}
	if at < 0 {
		t.Fatalf("the notice is not in the box's interior at all")
	}
	above, below := at, len(body)-1-at
	if d := above - below; d < -1 || d > 1 {
		t.Errorf("the notice sits at interior row %d of %d: %d rows above, %d below — "+
			"a line this far from centre reads as a heading, not as an empty state",
			at, len(body), above, below)
	}
}

// THE BOX CARRIES NO CARD'S FURNITURE, and that half is a FIX. The slot used to
// be handed a `lineup.Card{}`, whose zero Slot IS a location report — so the
// console drew `LOCATION REPORT •STANDARD• [0]` over an empty box: a report that
// does not exist, graded, with a chip that opens nothing.
func TestTheStandbyBoxNamesNoReportAndOffersNoHandle(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	live := strings.Join(liveRows(t, stripANSITest(b.View().Content)), "\n")
	for _, forbidden := range []string{"LOCATION REPORT", "STANDARD", chipFor("0")} {
		if strings.Contains(live, forbidden) {
			t.Errorf("the standby box carries %q; there is no report in that slot to name, grade or open:\n%s",
				forbidden, live)
		}
	}
}

// THE GROUND IS THE TOKEN AND NOT A LITERAL (HUM LEAD's standing rule on colour),
// AND IT IS NOT ANY CARD'S GROUND. An empty slot that shared a card's tint would
// be a card the operator cannot make out; one that shared the rail's red would say
// the station is live.
func TestTheStandbyBoxIsPaintedGreyFromItsOwnToken(t *testing.T) {
	got := standbyTone()
	if bg := render.Tok(render.CardEmptyBG); bg == "" || !strings.Contains(got, bg) {
		t.Fatalf("the standby tone %q is not CardEmptyBG (%q)", got, bg)
	}
	for name, tok := range map[string]render.Token{
		"a station card":   render.CardBG,
		"an operator card": render.CardOperatorBG,
		"the LIVE rail":    render.RailLiveBG,
	} {
		if strings.Contains(got, render.Tok(tok)) {
			t.Errorf("the empty LIVE slot is painted like %s; it is neither a card nor a state", name)
		}
	}
	// AND IT IS REACHED FROM THE BOX, not merely declared here — with colour
	// FORCED ON, because the tint is gated on a tty and an assertion that cannot
	// see an escape is an assertion that cannot fail (radio_panel_test.go does the
	// same, for the same reason).
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	lane := newCardLane(80, render.Opts{}.Glyphs())
	rows := lane.standbyBox(bcReadCardRows)
	if len(rows) != bcReadCardRows {
		t.Fatalf("the standby box is %d rows, not a read card's %d", len(rows), bcReadCardRows)
	}
	for _, r := range rows {
		if !strings.Contains(r, render.Tok(render.CardEmptyBG)) {
			t.Errorf("a row of the standby box is unpainted; the box is one object:\n%q", r)
		}
	}
}

// AN EMPTY UP NEXT STILL SHIMMERS. The first draft gave the notice to every read
// slot, and a station that has just opened then reported "no reports read or
// active" about the slot the Composer is working on — an absence where there is
// work in progress.
func TestUpNextShimmersWhileTheDirectorIsStillChoosing(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.View().Content)
	if strings.Count(got, bcStandbyNotice) != 1 {
		t.Errorf("the notice belongs to the LIVE slot ALONE; it appears %d times",
			strings.Count(got, bcStandbyNotice))
	}
	if !strings.Contains(got, "waiting for the line-up") {
		t.Errorf("an empty UP NEXT must say the Director is still choosing:\n%s", got)
	}
}

// GOING ON AIR REPLACES THE NOTICE WITH THE CARD, AND MOVES NOTHING ELSE. The box
// is the same height either way, for the reason readBody's height is fixed: the
// operator is watching the running order at exactly that moment, and a frame that
// shunted up a row would be the surface flinching.
func TestGoingOnAirReplacesTheNoticeAndKeepsTheHeight(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"), card(t, "b", "Bonsall, CA"))
	rest := strings.Split(stripANSITest(b.View().Content), "\n")

	live, _ := b.Update(StationMsg{Power: lineup.Running})
	air := strings.Split(stripANSITest(live.View().Content), "\n")

	if strings.Contains(strings.Join(air, "\n"), bcStandbyNotice) {
		t.Errorf("the station is on the air and the LIVE slot still says nothing is")
	}
	if len(rest) != len(air) {
		t.Errorf("the frame is %d rows at rest and %d on the air; the running order moved under the operator",
			len(rest), len(air))
	}
	// AND THE CARD THAT WAS UP NEXT IS NOW THE ONE ON THE AIR, which is the D-40
	// movement the operator has already seen.
	if !strings.Contains(strings.Join(air[:len(air)/2], "\n"), "Oceanside") {
		t.Errorf("the head of the queue did not take the air:\n%s", strings.Join(air, "\n"))
	}
}
