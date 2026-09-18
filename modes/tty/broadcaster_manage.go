package tty

// broadcaster_manage.go — the two things an operator does to a scheduled card
// (D-118).
//
// HUM LEAD, 2026-09-13: "Every line up position from UP NEXT -> Pos. 14 need the
// following controls in the Modal - these are **specific and unique** to the
// Broadcaster UI: [P] Change Position [k] Drop from Line-Up."
//
// THEY ARE A THIRD WINDOW, OVER THE CARD'S. `confirmOverlay` is the seam the
// ctrl+d window already uses for "a question asked on top of the window that
// raised it", and both of these are that: the card stays visible underneath, so
// the operator can still read what they are about to move or discard.
//
// THE SCHEDULE OWNS BOTH ACTS. `lineup.Moved` and `lineup.Dropped` have existed
// since 0.14.0 — "the card model carries the fields; the controls arrive with the
// Broadcaster UI" — and FR-3.3 is why they are EVENTS: "an action must never be
// shown as taken unless the schedule took it".

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// cardAction is the question the card window has open over itself.
type cardAction int

const (
	cardActionNone cardAction = iota
	// cardActionMove asks WHERE, and takes a position.
	cardActionMove
	// cardActionDrop asks WHETHER, and takes a yes.
	cardActionDrop
)

// bcManageWidth is the overlay's box. Narrower than the card beneath it, so the
// card is still legible around it — a question about a thing should not hide the
// thing.
const bcManageWidth = 62

// movePrompt is the [P] window: a position, and the two keys that settle it.
func (d Dashboard) movePrompt(o render.Opts) []string {
	centre := func(s string) string {
		return strings.Repeat(" ", max((bcManageWidth-2-2*modalInset-render.Width(s))/2+modalInset-panelSide, 0)) + s
	}
	lo, hi := bcScheduledFrom, MainTrackSlots-1
	rows := []string{"", centre("MOVE THIS CARD TO POSITION")}
	// THE FIELD IS THE ANSWER, and it is centred under the question so the eye
	// does not have to travel to find what it is typing into.
	rows = append(rows, "", centre("[ "+render.PadTo(d.cardMoveTo, 2)+" ]"+o.Glyphs().Cursor), "")
	// THE RANGE IS DERIVED, NOT WRITTEN. `bcScheduledFrom` and `MainTrackSlots`
	// are what the table actually draws, so the prompt cannot come to promise a
	// position the running order does not have.
	rows = append(rows, insetModalLines([]string{
		"Positions " + strconv.Itoa(lo) + " to " + strconv.Itoa(hi) + ". The card takes that place " +
			"in the running order and everything below it shifts down.", ""}, bcManageWidth-2*modalInset)...)
	if d.cardErr != "" {
		rows = append(rows, strings.Repeat(" ", modalInset)+o.Glyphs().Alert+" "+d.cardErr, "")
	}
	return append(rows, strings.Repeat(" ", modalInset)+o.KeyCap("esc")+"  Cancel   "+
		o.KeyCap("enter")+"  Confirm", "")
}

// dropPrompt is the [k] window: what discarding actually does, and a yes.
//
// IT SAYS WHERE THE CARD GOES AND WHERE THE PLACE STAYS, which is the HUM LEAD's
// own wording: "explaining the card will be dropped from the pool (it goes into
// the discard pile), and the location will remain in the location pool, and will
// be added by the producer at a later time." An operator who thinks dropping a
// card removes a TOWN from their station would stop using the control.
func (d Dashboard) dropPrompt(o render.Opts) []string {
	centre := func(s string) string {
		return strings.Repeat(" ", max((bcManageWidth-2-2*modalInset-render.Width(s))/2+modalInset-panelSide, 0)) + s
	}
	rows := []string{"", centre("ARE YOU SURE?"), ""}
	rows = append(rows, insetModalLines([]string{
		"This card is dropped from the running order and goes into the discard pile. " +
			"It is not read.",
		"",
		"The LOCATION stays in your location pool, and the Producer may offer it again " +
			"later — dropping a card is not removing a place from your station.",
		""}, bcManageWidth-2*modalInset)...)
	return append(rows, strings.Repeat(" ", modalInset)+o.KeyCap("esc")+"  Cancel   "+
		o.KeyCap("enter")+"  Drop from Line-Up", "")
}

// moveTarget is the typed position, and zero when it is not a usable one.
//
// THE RANGE IS THE TABLE'S. A position the running order does not draw is not a
// place a card can be moved to, and the schedule would refuse it — so the window
// says so first, where the operator can still fix it.
func (d Dashboard) moveTarget() int {
	n, err := strconv.Atoi(strings.TrimSpace(d.cardMoveTo))
	if err != nil || n < bcScheduledFrom || n > MainTrackSlots-1 {
		return 0
	}
	return n
}
