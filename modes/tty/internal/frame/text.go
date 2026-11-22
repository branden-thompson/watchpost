package frame

// text.go — the text helpers both surfaces share: centring and clipping a
// line, the test event's mark, and a plural's suffix (0.19.6 A-21).

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestEventMark is what the band says when what it shows was fabricated.
//
// PREPENDED AND POSTPENDED, ON THE ITEM (HUM LEAD 2026-09-07). The tape
// scrolls: a marker at one end only is off-window half the time, and a marker
// at both ends means the item cannot be on screen without one of them. NOT
// LANE CHROME in the band's top row, though an 18-cell prefix per item at the
// 80-column floor makes the marker the majority of the tape — the ruling
// overrides that argument, and it is the ITEM that is fabricated rather than
// the lane it happens to be filed under.
//
// TWO ASTERISKS, NOT THREE, and the ruling's own wording is the reason twice
// over: it is what was written, and at 14 cells it fits the severe window's
// EVENT column ahead of a product where "*** TEST EVENT ***" does not — that
// column truncates, and a mark that pushes the product out of its own column
// is a mark that hides what it is marking.
const TestEventMark = "**TEST EVENT**"

// CenterText centres text within width (clipping by DISPLAY WIDTH when it is
// wider) — the breaking-news event sits in the middle of the band. A
// non-positive width yields empty (a negative slice bound would panic — P4 F3).
func CenterText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	w := render.Width(text)
	if w >= width {
		return clipToWidth(text, width)
	}
	left := (width - w) / 2
	return render.PadTo(strings.Repeat(" ", left)+text, width)
}

// clipToWidth truncates text to at most width display cells (wide runes count
// as their cell width, so the result never overflows the band).
//
// ONE OWNER (0.16.0 D-141): `render.TruncateCells`, which skips escape
// sequences when counting and remembers whether the last one was a reset, so it
// can close a span it cuts through. A rune walk asking `render.Width` for each
// would measure a lone \x1b as 0 but '[', '1' and 'm' as 1 apiece, so EVERY
// BOLD SPAN WOULD COST EIGHT PHANTOM CELLS: text that fits is cut anyway, the
// cut lands mid-escape leaving a bare \x1b in the frame, and the SGR is never
// closed so the weight bleeds into whatever follows.
//
// Width measurement is not a thing to reimplement per caller — the same shape
// as 0.16.0 D-129a (a caveat wrapped to the wrong width) and 0.16.0 D-138 (a
// band centred instead of wrapped).
func clipToWidth(text string, width int) string {
	return render.TruncateCells(text, width)
}

// Plural is the suffix a count's noun takes: "s" for any count but one.
func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// NoTransmitter is what a place says for the station before it has an
// epicentre (0.16.0 D-72): the console's station row and the Propagation mode's
// stand-in origin both say it, an honest absence, never a mock's values
// (0.19.6 A-31).
const NoTransmitter = "(no transmitter set)"
