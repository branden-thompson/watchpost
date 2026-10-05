package tty

// credit_row.go — one source's credit, one way wherever it is drawn (D-235):
// its chip, what it gives, its endpoint, and a note where it has one.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// CreditLine is one data source's credit: its chip (the map's own where it
// has one), a phrase for what it gives, its endpoint at the right margin,
// and a note under the phrase - a condition of the source's own, such as
// AirNow's - only where there is one.
type CreditLine struct {
	Badge, What, Host, Note string
}

// creditIndent is the air before a row's chip.
const creditIndent = 2

// badgeWidth is the chip column a set of rows shares: its widest chip, so
// every row's dash stands in one column, a source added later included.
func badgeWidth(lines []CreditLine) int {
	w := 0
	for _, c := range lines { // the set's rows (P10-02)
		w = max(w, render.Width(chipFace(c.Badge)))
	}
	return w
}

// creditRows is a source's credit in width cells: the chip in a column of
// badgeW, " - ", the phrase, the endpoint light blue at the right margin (on
// a line of its own where the phrase leaves it no room), and the note under
// the phrase. Measured plain, so the colours never move the margin.
func creditRows(c CreditLine, width, badgeW int) []string {
	chip := chipFace(c.Badge)
	lead := strings.Repeat(" ", creditIndent) + chip + strings.Repeat(" ", max(0, badgeW-render.Width(chip))) + " - "
	left := lead + c.What
	host := render.Tint(c.Host, render.Tok(render.AboutHost))
	gap := width - render.Width(left) - render.Width(c.Host)
	var out []string
	switch {
	case c.Host == "":
		out = append(out, left)
	case gap >= 2:
		out = append(out, left+strings.Repeat(" ", gap)+host)
	default:
		out = append(out, left, strings.Repeat(" ", max(0, width-render.Width(c.Host)))+host)
	}
	if c.Note != "" {
		out = append(out, strings.Repeat(" ", render.Width(lead))+render.Tint(c.Note, "3;"+render.Tok(render.AboutNote))) // italic, a shade back (D-236)
	}
	return out
}
