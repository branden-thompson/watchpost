package tty

// map_scrub.go — the rows under the map, in either mode (0.18.0 D-103,
// D-105): the colour row; the notes; the picture's own status; the loop's
// row over its timeline, the controls beside them; the region keys; the
// chips. The controls left the map for here, so the map is all map.

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// scrubBoxW is the controls box's width beside the timeline: its inner width
// and its two borders.
const scrubBoxW = controlsInner + 2

// scrubBoxFits reports whether the controls sit beside the timeline: under
// 60 columns the timeline takes the width, and the keys are in Help.
func (d Dashboard) scrubBoxFits() bool { return d.mapTextW() >= 60 }

// scrubW is the timeline's width: the text width, less the controls box and
// a gap when it fits beside it.
func (d Dashboard) scrubW() int {
	if d.scrubBoxFits() {
		return d.mapTextW() - scrubBoxW - 1
	}
	return d.mapTextW()
}

// scrubBoxRows are the rows the loop's row and its timeline take, with the
// controls box beside them: the row, the three of the timeline, and the
// box's bottom edge.
const scrubBoxRows = 1 + radarRows + 1

// scrubRows are the rows under the map while the timeline is held (D-103).
func (d Dashboard) scrubRows(width int) []string {
	var out []string
	if d.radarMode() {
		out = append(out, " "+d.radarLegendRow(width))
	} else {
		out = append(out, " "+d.tempLegendRow(width)) // W10.10: the bands' colours, as radar's are
	}
	for _, l := range d.noteLines(width) {
		out = append(out, " "+l)
	}
	out = append(out, " "+render.TruncateCells(d.pictureStatus(), width)) // blank when whole: the map's size never waits on it
	tl := d.mapPane.radarTimeline
	if !d.radarMode() {
		tl = d.mapPane.fcTimeline // D-94: Forecast mode's steps
	}
	block := []string{d.loopRow(d.scrubW())}
	for i := range radarRows {
		l := ""
		if i < len(tl) {
			l = tl[i]
		}
		block = append(block, l)
	}
	block = append(block, "")
	if d.scrubBoxFits() {
		box := d.scrubControls()
		for i := range block {
			block[i] = render.PadTo(block[i], d.scrubW()+1)
			if i < len(box) {
				block[i] += box[i]
			}
		}
	}
	for _, l := range block {
		out = append(out, " "+l)
	}
	return append(out, "", " "+d.regionsRow(width), "", " "+d.chipsRow(width))
}

// pictureStatus is the picture's own state and the refresh's estimate, on a
// row of their own (D-105): loading, offline or coarser; nothing when whole.
func (d Dashboard) pictureStatus() string {
	var parts []string
	if s := d.mapStatusText(); s != "" {
		parts = append(parts, s)
	}
	if est := costEstimate(d.mapCost); est != "" {
		parts = append(parts, est)
	}
	return strings.Join(parts, " · ")
}

// scrubLead is where the timeline's bar starts: past its back key's chip.
func (d Dashboard) scrubLead() int {
	o := d.opts()
	arrow := strings.NewReplacer("shift+left", "shift+←", "shift+right", "shift+→")
	if o.ASCII {
		arrow = strings.NewReplacer()
	}
	return render.Width(o.KeyCap(arrow.Replace(d.firstKey(actMapBack)))) + 1
}

// placed is one item of a row: its words as printed, and its width.
type placed struct {
	text string
	w    int
}

// item is a placed from printed words.
func item(s string) placed { return placed{text: s, w: render.Width(s)} }

// spread lays items along a row: the first at `from`, the last ending at
// `to`, the rest centred between them in order; never over one another.
func spread(items []placed, from, to int) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	at := 0
	for i, it := range items {
		col := from
		switch {
		case i == len(items)-1 && i > 0:
			col = to - it.w
		case i > 0:
			col = from + (to-from)*i/(len(items)-1) - it.w/2
		}
		if i > 0 {
			col = max(col, at+2)
		}
		b.WriteString(strings.Repeat(" ", max(col-at, 0)))
		b.WriteString(it.text)
		at = max(col, at) + it.w
	}
	return b.String()
}

// loopRow is the loop's row over the timeline (D-105): RADAR and its source,
// the frame, the newest frame's age (stale marked), and STOPPED in yellow or
// PLAYING in green; in Forecast mode FORECAST, the step and the same state.
// Where there is no loop yet, what the radar says of itself.
func (d Dashboard) loopRow(width int) string {
	from, to := d.scrubLead(), width-d.scrubLead()+1
	state := func(playing bool) placed {
		if playing {
			return item(render.Tint("PLAYING", render.Tok(render.ProviderOK)))
		}
		return item(render.Tint("STOPPED", render.Tok(render.ListPointer)))
	}
	if !d.radarMode() {
		steps := d.forecastSteps()
		at := min(d.mapPane.fcStep, len(steps)-1)
		lead := "FORECAST"
		if d.tempOn() {
			lead += " " + render.TintRaw(" "+strings.ToUpper(d.stepSource())+" ", d.tempChipTones())
		}
		return spread([]placed{item(lead), item("STEP " + strconv.Itoa(at+1) + " / " + strconv.Itoa(len(steps))), state(d.mapPane.fcPlaying)}, from, to)
	}
	m := d.mapPane.m
	if m == nil || d.mapPane.radarSource == "" || m.Loop().Count == 0 {
		return render.PadTo("", from) + render.TruncateCells(d.mapPane.radarLine, max(width-from, 1))
	}
	st := m.Loop()
	age := d.now().Sub(st.Newest)
	newest := "NEWEST " + strconv.Itoa(int(age.Minutes())) + " MIN AGO"
	if age > radarStale {
		newest = render.Tint(newest+", STALE", render.Tok(render.ListPointer)) // FR-5.4: never hidden, marked
	}
	return spread([]placed{item("RADAR " + d.radarChip()), item("FRAME " + strconv.Itoa(st.Index+1) + " / " + strconv.Itoa(st.Count)),
		item(newest), state(st.Playing)}, from, to)
}

// scrubControls is the controls box beside the timeline (D-103): the arrows
// and zoom, and the place keys; the region keys have a row of their own.
func (d Dashboard) scrubControls() []string {
	cap := d.controlCap
	return boxed("Controls", []string{
		"    " + cap(actMapPanUp, "↑") + "     " + cap(actMapZoomIn, "+"),
		" " + cap(actMapPanLeft, "←") + cap(actMapPanDown, "↓") + cap(actMapPanRight, "→") + "  " + cap(actMapZoomOut, "-"),
		" " + cap(actMapPrev, "[") + cap(actMapNext, "]") + " place",
	}, controlsInner)[:scrubBoxRows]
}

// mapRegionNames are the region row's names (D-103), in geo's numbering.
var mapRegionNames = []string{"CONT. US", "Alaska", "Hawaii", "US Caribbean", "American Samoa", "Guam / N. Marianas"}

// regionsRow is the region keys as a row of their own (D-103): MAPS: and
// each number's region, shorter names where the full ones do not fit.
func (d Dashboard) regionsRow(width int) string {
	build := func(names []string) string {
		parts := []string{"MAPS:"}
		for i, act := range mapRegionActs {
			key := d.firstKey(act)
			if key == "" {
				continue
			}
			parts = append(parts, d.controlCap(act, key)+" "+names[i])
		}
		return strings.Join(parts, "   ")
	}
	row := build(mapRegionNames)
	if render.Width(row) > width {
		row = build(mapRegionShort)
	}
	return render.TruncateCells(row, width)
}

// chipsRow is the window's keys as chips: Area Alerts, the mode, Overlays
// (D-63, D-65, D-94; the legend retired, D-103).
func (d Dashboard) chipsRow(width int) string {
	return render.TruncateCells(strings.Join(d.mapChips(), "   "), width)
}
