package tty

// columns.go — the two-column layout the wide floating windows share (Help,
// API Status — HUM LEAD UAT 2026-08-28): blocks of lines laid side by side
// when the terminal is wide enough, stacked when it is not. One owner, so
// the next window that widens takes the same rules.

import "github.com/branden-thompson/watchpost/platform/render"

// columnGap is the air between two columns; panelFrame is what the panel
// spends around a body that fits (its frame, 4), panelRail what the scroll
// rail and its gap add when the body does not (3) — the same rule the
// panel itself draws by (wrapModal: w-4 / w-7).
const (
	columnGap  = 4
	panelFrame = 4
	panelRail  = 3
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

// columnMargin is the air between the right column and the panel's edge —
// 3, matching the 3-cell inset on the left (HUM LEAD UAT 2026-08-28).
const columnMargin = 3

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
