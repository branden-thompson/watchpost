package tty

// columns.go — the two-column layout the wide floating windows share (Help and
// Settings): blocks of lines laid side by side when the terminal is wide
// enough, stacked when it is not. One owner, so the next window that widens
// takes the same rules.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// columnGap is the air between two columns; panelFrame is what the panel
// spends around a body that fits (its frame, 4), panelRail what the scroll
// rail and its gap add when the body does not (3) — the same rule the
// panel itself draws by (wrapModal: w-4 / w-7).
const (
	columnGap  = 4
	panelFrame = 4
	panelRail  = 3

	// panelSide is the frame's chrome on ONE side — what the panel has already
	// inset a body line by before the window adds anything of its own.
	//
	// DERIVED, and the ONE owner of it: relayfault.go and debug.go each carried
	// their own `= 2` for this, which is the panel's number and not theirs to
	// restate (D-1, red team 2026-09-05).
	panelSide = panelFrame / 2
)

// panelChromeFor is the panel's chrome for a body of bodyLines against the
// modal's line budget.
func panelChromeFor(bodyLines, maxLines int) int {
	if bodyLines > maxLines {
		return panelFrame + panelRail
	}
	return panelFrame
}

// widest is the widest line of a block set, in cells.
func widest(blocks ...[]string) int {
	w := 0
	for _, b := range blocks {
		for _, l := range b {
			w = max(w, render.Width(l))
		}
	}
	return w
}

// modalInset is the air between a floating window's border and its content, on
// BOTH sides (HUM LEAD: UAT 2026-08-28 for the left, 2026-09-05 for the right).
//
// ONE OWNER (D-1). It was a bare 3 here, ad-hoc "   " literals in the windows
// that remembered, and nothing at all on the right — so a window whose content
// happened to be short looked correct, and the same window with a real station
// name in it ran into the border.
const modalInset = 3

// insetModalLines puts a window's body inside its margins: modalInset cells of
// air on the left, and WRAPPED — never cut — so no line reaches within
// modalInset of the right.
//
// THE SINGLE OWNER IS A HELPER, NOT A CONSTANT (red team 2026-09-05). modalInset
// alone could not be one: the windows that remembered the ruling each did their
// own arithmetic against it, in the panel's coordinates rather than the body's,
// and each got a different answer — 2, 3 and 4 cells across the app, with the
// right-hand margin unenforced. The number is not the rule; APPLYING it is.
//
// content is the room between the two margins; the panel's own chrome
// (panelSide) is already there, so this makes up only the difference. It is NOT
// a parameter: both callers passed the same 2, which is the panel's number
// rather than either window's, and a knob nothing turns is one more thing that
// can be turned wrongly (P10-07).
//
// A blank line stays blank: there is nothing to inset, and padding it would
// make the window's air visible to anything measuring trailing space.
func insetModalLines(lines []string, content int) []string {
	pad := strings.Repeat(" ", max(modalInset-panelSide, 0))
	out := make([]string, 0, len(lines))
	for _, l := range lines { // bounded by the body (P10-02)
		if strings.TrimSpace(l) == "" {
			out = append(out, l)
			continue
		}
		for _, w := range render.WrapLines([]string{strings.TrimLeft(l, " ")}, content) {
			out = append(out, pad+w)
		}
	}
	return out
}

// columnMargin is the air between the right column and the panel's edge. The
// SAME rule as the left inset, which is why it is that constant rather than its
// own 3: they were written to match and a second literal is a second thing to
// forget.
const columnMargin = modalInset

// twoColumnsWidth is the window width two columns need with this chrome.
func twoColumnsWidth(leftW, rightW, chrome int) int {
	return leftW + columnGap + rightW + columnMargin + chrome
}

// sideBySide lays two columns out row by row: the left padded to leftW +
// the gap, the right after it; a row with nothing on the right stays as
// the left line alone (no trailing pad).
func sideBySide(left, right []string, leftW int) []string {
	rows := max(len(left), len(right))
	out := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if r == "" {
			out = append(out, l)
			continue
		}
		out = append(out, render.PadTo(l, leftW+columnGap)+r)
	}
	return out
}

// stacked joins blocks top to bottom with a blank line between.
func stacked(blocks ...[]string) []string {
	var out []string
	for i, b := range blocks {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, b...)
	}
	return out
}
