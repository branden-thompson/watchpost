package tty

// poolnote.go — what a window says about a location typed against the STATION
// POOL, and how it says it.
//
// TWO WINDOWS ASK THE SAME QUESTION (D-129). The Line-Up Request window asks it
// of its Location field, and the console's `[l]` lookup asks it of its search
// box. They must answer with the same sentence in the same colour, or the
// operator learns two different meanings for one refusal — so the wording and
// the rendering live here rather than being written out twice.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// poolNote is the fact, then what the operator can do about it.
//
// AN UNREACHABLE LOCATION IS NOT AN ERROR. It is a real place the station
// cannot broadcast about, and Observer can show it — so the second half points
// there instead of the window refusing to understand.
func poolNote(query string, ref *snapshot.LocationRef, outside bool) (fact, aside string) {
	const elsewhere = "Observer supports location lookup outside Broadcast Radius"
	switch {
	case strings.TrimSpace(query) == "":
		return "", ""
	case ref == nil:
		return "Location not found in Pool.", elsewhere
	case outside:
		return "Outside the station's service radius.", elsewhere
	}
	return "", ""
}

// modalHelperWidth is how much room a helper line has inside a window OF THE
// GIVEN WIDTH, before that window wraps it: the content, less the panel's own
// inset on both sides and the helper's lead glyph.
//
// THE WINDOW'S WIDTH, AND NOT THE TERMINAL'S. Handing poolNoteLines a width
// wider than the window it draws into is the same defect as not wrapping at
// all: the lines come back tinted and too long, the FRAME wraps them a second
// time, and the second wrap happens after the colour — so the tail arrives
// plain. The console's lookup is 56 cells inside a 133-cell terminal, and it
// was being wrapped to 112: "Broadcast Radius" came out grey under a red
// sentence (HUM LEAD, UAT 2026-09-14, second screenshot of the same rule).
func modalHelperWidth(width int) int {
	return max(12, width-4-2*modalInset-4)
}

// poolNoteLines draws the note at the given content width.
//
// WRAPPED FIRST, THEN TINTED LINE BY LINE. A tint applied to the whole string
// is a pair of escape codes at its two ENDS, so a wrap leaves every line after
// the first with no colour on it — which is exactly how "Broadcast Radius"
// trailed off into plain grey mid-sentence (HUM LEAD, UAT 2026-09-14, with the
// screenshot). Styling survives a wrap only if every line carries it.
func poolNoteLines(o render.Opts, fact, aside string, width int) []string {
	if fact == "" {
		return nil
	}
	tone := render.Tok(render.NameWarning)
	var out []string
	for _, l := range render.WrapText(fact, width) { // bounded by the text (P10-02)
		out = append(out, "  "+o.Glyphs().Alert+" "+render.Tint(l, tone))
	}
	for _, l := range render.WrapText(aside, width) { // bounded by the text (P10-02)
		out = append(out, "    "+render.Italic(render.Tint(l, tone)))
	}
	return out
}
