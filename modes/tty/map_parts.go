package tty

// map_parts.go — the map window's parts, each drawn one way whatever the
// mode (0.18.0 D-103, the HUM LEAD's "consolidated into re-usable
// components"): the chip box over the map, the badge a tab on its frame's
// upper right (D-120), and the scrubber under it. The modes hand them their words; nothing here knows
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

// chipGrounds are the sources' chips' grounds, by the name a chip shows
// (D-83, D-134): every source its own.
var chipGrounds = map[string]render.Token{
	"MRMS": render.MapRadarMRMSBG, "IEM": render.MapRadarIEMBG, "HRRR": render.MapRadarModelBG,
	"NWS": render.MapChipNWSBG, "NDFD": render.MapChipNDFDBG, "O-METEO": render.MapChipOMeteoBG,
	"USGS": render.MapChipUSGSBG, "NIFC": render.MapChipNIFCBG, "HMS": render.MapChipHMSBG,
	"NDBC": render.MapChipNDBCBG, "CO-OPS": render.MapChipCOOPSBG, "AIRNOW": render.MapChipAirNowBG,
	"RECORDED": render.MapChipRecordedBG, // not a source: the history, appended after one (D-173, D-178)
}

// ChipKnown reports whether a source has a chip of its own, so the app can
// hold every source it names to one (D-134).
func ChipKnown(name string) bool {
	_, ok := chipGrounds[name]
	return ok
}

// chipFace is a source's chip, one way everywhere (D-134, the HUM LEAD's
// "<space><space><label><space><space>"): its name, bold, black or white as
// reads the more, on the source's own ground. MRMS≈ is MRMS's (D-132).
func chipFace(face string) string {
	ground, ok := chipGrounds[strings.TrimRight(face, "≈~")]
	if !ok {
		return "  " + face + "  "
	}
	return render.TintRaw("  "+face+"  ", render.ChipTones(ground))
}

// mapBadge is the badge's words, in either mode (D-92, U2-19), on one line
// (D-120): a title, a source chip and a moment, each where there is one.
func mapBadge(title, chip, moment string) string {
	var parts []string
	for _, p := range []string{title, chip, moment} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return " " + strings.Join(parts, "  ") + " "
}

// tabMinLeft is the least of the frame's top edge a tab leaves to the
// window's title.
const tabMinLeft = 12

// titleChrome is the top edge around a window's title up to the tab: the
// corner and its rule before the title, a space and a rule after it.
const titleChrome = 6

// tabCol is the column a badge's tab opens at on a window's frame.
func tabCol(rows []string, words string) int {
	if len(rows) == 0 {
		return 0
	}
	return render.Width(rows[0]) - render.Width(words) - 2
}

// withTab joins a badge to a window's frame as a tab on its top right
// (D-120): the frame's top edge opens into it, its words on the first row
// against the frame's right side, and its bottom edge closes into that side.
// As wide as its words; where the window is too narrow to leave its title
// room, the frame is left as it is.
func withTab(rows []string, words string, g render.BoxGlyphs) []string {
	w := render.Width(words)
	if len(rows) < 3 || w == 0 {
		return rows
	}
	col := tabCol(rows, words)
	if col < tabMinLeft {
		return rows
	}
	return spliceBox(rows, []string{g.T, g.Rail + words, g.BL + strings.Repeat(g.Rule, w) + g.R}, 0, col)
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
