package tty

// card_window.go — the Observer's card window: the card the console hands it to
// draw, and the two things an operator does to a scheduled card from it (D-118).
// The console builds a card's rows; the Dashboard holds them and draws the
// window, so this half stays with the Dashboard (0.19.6 A-21, design CK-2).
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
	"slices"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/modes/tty/console"
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
	lo, hi := console.ScheduledFrom, console.MainTrackSlots-1
	rows := []string{"", centre("MOVE THIS CARD TO POSITION")}
	// THE FIELD IS THE ANSWER, and it is centred under the question so the eye
	// does not have to travel to find what it is typing into.
	rows = append(rows, "", centre("[ "+render.PadTo(d.cardMoveTo, 2)+" ]"+o.Glyphs().Cursor), "")
	// THE RANGE IS DERIVED, NOT WRITTEN. `ScheduledFrom` and `MainTrackSlots`
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
	if err != nil || n < console.ScheduledFrom || n > console.MainTrackSlots-1 {
		return 0
	}
	return n
}

// showCard hands the Dashboard a Broadcaster card to draw, and opens the window.
//
// THE GENERATION MOVES ONLY ON A REAL CHANGE, which is what lets this be called
// on every update without defeating the memo. A card the operator is reading can
// re-hydrate under them — `RefreshAfter` is half of `StaleAfter` — and a window
// that kept its first frame while the data moved is exactly the freeze F-30 was
// filed for. Re-handing on every update is how the window cannot lag; comparing
// before bumping is how it does not redraw for nothing.
func (d Dashboard) showCard(id string, rows func(render.Opts) (string, []string), ground string, at render.Opts) Dashboard {
	// THE GROUND IS PART OF "THE SAME WINDOW". Left out of this comparison the
	// generation would not move when only the tone changed — a burst whose worst
	// alert is superseded by a more severe one would keep the old colour while
	// the body updated, which is the memo defect F-30 one field along.
	if d.cardID == id && d.cardGround == ground && d.cardRows != nil && sameCard(d.cardRows, rows, at) {
		return d
	}
	d.cardID, d.cardRows, d.cardGround, d.cardGen = id, rows, ground, d.cardGen+1
	return d
}

// sameCard reports whether two renderers would draw the same window at `at`.
func sameCard(a, b func(render.Opts) (string, []string), at render.Opts) bool {
	at1, a1 := a(at)
	at2, b1 := b(at)
	return at1 == at2 && slices.Equal(a1, b1)
}

// cardLines is the open card's body, asked of the console at THIS window's opts.
//
// NIL-SAFE, because `modalCard` can be opened by a fixture that has not handed a
// renderer over — and a window that draws nothing is a failure the gates report
// as such (fixtureFor) rather than a panic in the middle of a frame.
func (d Dashboard) cardLines(o render.Opts) []string {
	if d.cardRows == nil {
		return nil
	}
	_, body := d.cardRows(o)
	return body
}

// cardTitleOf is the open card's title, in the window's own glyphs.
func (d Dashboard) cardTitleOf(o render.Opts) string {
	if d.cardRows == nil {
		return ""
	}
	title, _ := d.cardRows(o)
	return title
}
