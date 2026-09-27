package tty

// map_parts.go — the map window's parts, each drawn one way whatever the
// mode (0.18.0 D-103, the HUM LEAD's "consolidated into re-usable
// components"): the chip box over the map, the badge in its upper right, and
// the scrubber under it. The modes hand them their words; nothing here knows
// radar from forecast.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// chipBox is a one-line chip in a box over the map: the edge chip that names
// the region beyond (D-81) and the mode's (D-103).
func chipBox(text string) []string {
	w := render.Width(text) + 2
	return []string{"┌" + strings.Repeat("─", w) + "┐", "│ " + text + " │", "└" + strings.Repeat("─", w) + "┘"}
}

// badgeW is the badge's width in cells: "RADAR DATA" and a space either side.
const badgeW = 12

// mapBadge is the badge in the map's upper right, in either mode (D-92,
// U2-19): a title, a source chip and a moment, three rows flush right.
func mapBadge(title, chip, moment string) []string {
	right := func(s string) string {
		if s == "" {
			return strings.Repeat(" ", badgeW)
		}
		return strings.Repeat(" ", max(badgeW-render.Width(s)-1, 0)) + s + " "
	}
	return []string{right(title), right(chip), right(moment)}
}

// scrubAlign is where a label under the scrubber sits against its point.
type scrubAlign uint8

const (
	alignCentre scrubAlign = iota
	alignLeft
	alignRight
)

// scrubLabel is a word under the scrubber at a point along it, from 0 at its
// start to 1 at its end.
type scrubLabel struct {
	text  string
	at    float64
	align scrubAlign
}

// scrubber is a timeline the listener steps along (D-86, D-94): where the
// cursor is, the ticks on the bar, the words over the cursor and under the
// bar. Radar's loop and Forecast mode's steps are both one.
type scrubber struct {
	cursor float64
	ticks  []float64
	above  string
	below  []scrubLabel
}

// draw is the scrubber's three rows at a width: the words over the cursor;
// the bar between the step keys' chips, its ticks and its cursor; the words
// under it. Too narrow for a bar of ten cells, three blank rows.
func (d Dashboard) draw(s scrubber, width int) []string {
	o := d.opts()
	arrow := strings.NewReplacer("shift+left", "shift+←", "shift+right", "shift+→") // the sketch's faces (D-86)
	if o.ASCII {
		arrow = strings.NewReplacer()
	}
	back, on := o.KeyCap(arrow.Replace(d.firstKey(actMapBack))), o.KeyCap(arrow.Replace(d.firstKey(actMapOn)))
	lead := render.Width(back) + 1
	bar := width - lead - render.Width(on) - 1 - 2 // the two end marks
	if bar < 10 {
		return []string{"", "", ""}
	}
	at := func(frac float64) int { return min(max(int(frac*float64(bar-1)+0.5), 0), bar-1) }
	cells := []rune(strings.Repeat("─", bar))
	for _, t := range s.ticks {
		cells[at(t)] = '┼'
	}
	cur := at(s.cursor)
	cells[cur] = '█'
	above := newPlacer(width)
	above.centre(s.above, lead+1+cur)
	below := newPlacer(width)
	for _, l := range s.below {
		col := lead + 1 + at(l.at)
		switch l.align {
		case alignLeft:
			below.left(l.text, col-1)
		case alignRight:
			below.right(l.text, col+1)
		default:
			below.centre(l.text, col)
		}
	}
	return []string{above.String(), back + " ├" + string(cells) + "┤ " + on, below.String()}
}
