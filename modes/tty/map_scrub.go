package tty

// map_scrub.go — the rows under the map, in either mode (0.18.0 D-103,
// D-105): the colour row; the notes; the picture's own status; the loop's
// row over its timeline, the controls beside them; the region keys; the
// chips. The controls sit here rather than on the map, so the map is all map.

import (
	"strconv"
	"strings"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
)

// scrubBoxW is the controls box's width beside the timeline: its inner width
// and its two borders.
const scrubBoxW = controlsInner + 2

// scrubBoxFits reports whether the controls sit beside the timeline: under
// 60 columns the timeline takes the width, and the keys are in Help.
func (mm mapModel) scrubBoxFits(e mapEnv) bool { return mm.mapTextW(e) >= 60 }

// scrubW is the timeline's width: the text width, less the controls box and
// a gap when it fits beside it.
func (mm mapModel) scrubW(e mapEnv) int {
	if mm.scrubBoxFits(e) {
		return mm.mapTextW(e) - scrubBoxW - 1
	}
	return mm.mapTextW(e)
}

// scrubBoxRows are the rows the loop's row and its timeline take, with the
// controls box beside them: the row, the three of the timeline, and the
// box's bottom edge.
const scrubBoxRows = 1 + radarRows + 1

// timelineBlock is a weather mode's rows beside the controls: the loop's
// row, the timeline's rows, and the cost estimate under its right end.
func (mm mapModel) timelineBlock(tl []string, e mapEnv) []string {
	block := []string{mm.loopRow(mm.scrubW(e), e)}
	for i := range radarRows {
		l := ""
		if i < len(tl) {
			l = tl[i]
		}
		block = append(block, l)
	}
	est := costEstimate(mm.mapCost) // under the timeline's right end, as the HUM LEAD drew it (D-133)
	return append(block, strings.Repeat(" ", max(mm.scrubW(e)-render.Width(est), 0))+est)
}

// scrubRows are the rows under the map while the timeline is held (D-103).
func (mm mapModel) scrubRows(width int, e mapEnv) []string {
	var out, notes []string
	switch {
	case mm.mapMode(e) == modeRadar:
		out, notes = append(out, " "+mm.radarLegendRow(width)), mm.noteLines(width, e)
	case mm.mapMode(e) == modeForecast && mm.rainOn(e):
		head, preset := mm.rainKey()
		out, notes = append(out, " "+mm.rainRow(head, preset, width)), mm.noteLines(width, e) // D-117: radar's colours, said to be a model's; D-184: NDFD's totals in their own
	case mm.mapMode(e) == modeForecast:
		out, notes = append(out, " "+mm.tempLegendRow(width, e)), mm.noteLines(width, e) // W10.10: the bands' colours, as radar's are
	case mm.mapMode(e) == modePropagation:
		legend := mm.propLegendRows(width) // drawn once: its bands on their colours, then their MHz, its note line (D-160)
		out = append(out, " "+legend[0])
		if mm.propNoteShown(e) {
			notes = legend[1:]
		}
	}
	for _, l := range notes {
		out = append(out, " "+l)
	}
	out = append(out, " "+render.TruncateCells(mm.pictureStatus(e), width)) // blank when whole: the map's size never waits on it
	var block []string
	switch mm.mapMode(e) {
	case modeRadar:
		block = mm.timelineBlock(mm.mapPane.radarTimeline, e)
	case modeForecast:
		block = mm.timelineBlock(mm.mapPane.fcTimeline, e) // D-94: Forecast mode's steps
	case modePropagation:
		block = mm.propBlock(mm.scrubW(e), e) // D-157: the same rows, its own answer
	}
	if mm.scrubBoxFits(e) {
		box := mm.scrubControls(e)
		for i := range block {
			block[i] = render.PadTo(block[i], mm.scrubW(e)+1)
			if i < len(box) {
				block[i] += box[i]
			}
		}
	}
	for _, l := range block {
		out = append(out, " "+l)
	}
	return append(out, "", " "+mm.regionsRow(width, e), "", " "+mm.chipsRow(width, e))
}

// badgeGap is the space between two badges, as the HUM LEAD drew it.
const badgeGap = "    "

// badgeRows are the layers drawn, a badge each, "FIRE [NIFC]/[HMS]", in the
// Overlays menu's order (D-133): the one place under the map their sources
// are named, in full in the Status window (D-131). A layer off, one with
// nothing drawn yet, and the radar - its chip is where it was - have none. A
// second row only when one cannot hold them; a badge is never cut.
func (mm mapModel) badgeRows(width int, e mapEnv) []string {
	var rows []string
	row := ""
	for _, b := range mm.badges(e) {
		switch {
		case row == "":
			row = b
		case render.Width(row)+len(badgeGap)+render.Width(b) <= width:
			row += badgeGap + b
		default:
			rows, row = append(rows, row), b
		}
	}
	if row != "" {
		rows = append(rows, row)
	}
	return rows
}

// badges are the layers drawn, each its label and its sources' chips.
func (mm mapModel) badges(e mapEnv) []string {
	var out []string
	mode := mm.mapMode(e)
	for _, l := range e.Layers {
		if l.Key == RadarLayer || rainIsRadar(l.Key, mode) || !mm.layerOn(l.Key, e) {
			continue
		}
		chips := l.Chips
		if len(chips) == 0 {
			chips = mm.mapPane.temp.Chips[l.Key] // a source that varies: as the answer drew it
		}
		if len(chips) == 0 {
			continue
		}
		faces := make([]string, len(chips))
		for i, c := range chips {
			faces[i] = chipFace(c)
		}
		out = append(out, badgeLabel(l)+" "+strings.Join(faces, "")) // each chip its ground alone, side by side: no brackets (D-174)
	}
	if key := mm.outlineKey(e); key != "" {
		out = append(out, key)
	}
	return out
}

// outlineKeyLabel heads the badge that keys the outline digits (AX-8).
const outlineKeyLabel = "OUTLINES"

// outlineSeverities are the severities an outline's digit stands for, most
// severe first.
var outlineSeverities = []tuimaps.Severity{tuimaps.SeverityExtreme, tuimaps.SeveritySevere, tuimaps.SeverityModerate, tuimaps.SeverityMinor, tuimaps.SeverityUnknown}

// outlineDigit is one severity's digit and word: "3 severe".
func outlineDigit(s tuimaps.Severity) string { return s.Digit() + " " + strings.ToLower(s.Word()) }

// outlineKey is the badge keying each digit the picture draws on an alert's
// outline to its severity, "" while none is drawn or no picture is meant
// (not mapPicture: the map's fit counts this badge).
func (mm mapModel) outlineKey(e mapEnv) string {
	if e.Opts.ASCII || mm.mapDesc == mapDescInstead || mm.mapMode(e) == modePropagation {
		return ""
	}
	var keyed []string
	for _, s := range outlineSeverities {
		if mm.mapPane.drawnSev[strings.ToLower(s.Word())] {
			keyed = append(keyed, outlineDigit(s))
		}
	}
	if len(keyed) == 0 {
		return ""
	}
	return outlineKeyLabel + " " + strings.Join(keyed, " · ")
}

// outlineKeyHelp is Help's line keying every outline digit (AX-8).
func outlineKeyHelp() string {
	keyed := make([]string, len(outlineSeverities))
	for i, s := range outlineSeverities {
		keyed[i] = outlineDigit(s)
	}
	return "The digit along an alert's outline is its severity: " + strings.Join(keyed, ", ") + "."
}

// badgeLabel is a layer's word on its badge, as the HUM LEAD wrote them.
func badgeLabel(l MapLayer) string {
	switch l.Key {
	case AlertLayer:
		return "ALERTS"
	case TemperatureLayer:
		return "TEMP"
	case FeelsLayer:
		return "FEELS"
	case RainLayer:
		return "RAIN"
	case "quake":
		return "QUAKES"
	}
	return strings.ToUpper(l.Label)
}

// pictureStatus is the picture's own state and the refresh's estimate, on a
// row of their own (D-105): loading, offline or coarser; nothing when whole.
func (mm mapModel) pictureStatus(e mapEnv) string {
	var parts []string
	if s := mm.mapStatusText(e); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ") // the estimate is under the timeline (D-133)
}

// scrubLead is where the timeline's bar starts: past its back key's chip.
func (mm mapModel) scrubLead(e mapEnv) int {
	o := e.Opts
	arrow := strings.NewReplacer("shift+left", "shift+←", "shift+right", "shift+→")
	if o.ASCII {
		arrow = strings.NewReplacer()
	}
	return render.Width(o.KeyCap(arrow.Replace(mm.firstKey(actMapBack)))) + 1
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
// the frame, the newest frame's age (stale marked), and STOPPED in yellow -
// REDUCE MOTION while that holds it still - or PLAYING in green; in Forecast
// mode FORECAST, the step and the same state.
// Where there is no loop yet, what the radar says of itself.
func (mm mapModel) loopRow(width int, e mapEnv) string {
	from, to := mm.scrubLead(e), width-mm.scrubLead(e)+1
	items, ok := mm.loopItems(e)
	if !ok {
		return render.PadTo("", from) + render.TruncateCells(mm.mapPane.radarLine, max(width-from, 1))
	}
	return spread(items, from, to)
}

// loopWords are the loop's row as a line of text, its items in order, for a
// window without the picture (AX-5); where there is no loop yet, what the
// radar says of itself.
func (mm mapModel) loopWords(e mapEnv) string {
	items, ok := mm.loopItems(e)
	if !ok {
		return mm.mapPane.radarLine
	}
	words := make([]string, len(items))
	for i, it := range items {
		words[i] = it.text
	}
	return strings.Join(words, " · ")
}

// loopItems are the loop's row's items, false where there is no loop yet.
func (mm mapModel) loopItems(e mapEnv) ([]placed, bool) {
	state := func(playing bool) placed {
		switch {
		case playing:
			return item(render.Tint("PLAYING", render.Tok(render.ProviderOK)))
		case e.ReduceMotion: // why space plays nothing (D-360)
			return item(render.Tint("REDUCE MOTION", render.Tok(render.ListPointer)))
		}
		return item(render.Tint("STOPPED", render.Tok(render.ListPointer)))
	}
	if mm.mapMode(e) == modeForecast {
		steps := mm.forecastSteps(e)
		at := min(mm.mapPane.fcStep, len(steps)-1)
		lead := "FORECAST"
		if mm.tempOn(e) {
			lead += " " + chipFace(mm.tempFace(e))
		}
		return []placed{item(lead), item("STEP " + strconv.Itoa(at+1) + " / " + strconv.Itoa(len(steps))), state(mm.mapPane.fcPlaying)}, true
	}
	m := mm.mapPane.m
	if m == nil || mm.mapPane.radarSource == "" || m.Loop().Count == 0 {
		return nil, false
	}
	st := m.Loop()
	age := e.Now().Sub(newestObserved(st))
	newest := "NEWEST " + strconv.Itoa(int(age.Minutes())) + " MIN AGO"
	if age > radarStale {
		newest = render.Tint(newest+", STALE", render.Tok(render.ListPointer)) // FR-5.4: never hidden, marked
	}
	lead := item("RADAR " + mm.radarChip(e.Opts.ASCII))
	if st.Forecast && mm.mapPane.radarAhead != "" { // the hours ahead: a model's, said so (D-113)
		lead = item("FORECAST " + chipFace(mm.aheadName()))
	}
	return []placed{lead, item("FRAME " + strconv.Itoa(st.Index+1) + " / " + strconv.Itoa(st.Count)),
		item(newest), state(st.Playing)}, true
}

// scrubControls is the controls box beside the timeline (D-103): the arrows
// and zoom, and the place keys; the region keys have a row of their own.
func (mm mapModel) scrubControls(e mapEnv) []string {
	return boxed("Controls", []string{
		"    " + mm.controlCap(actMapPanUp, "↑", e) + "     " + mm.controlCap(actMapZoomIn, "+", e),
		" " + mm.controlCap(actMapPanLeft, "←", e) + mm.controlCap(actMapPanDown, "↓", e) + mm.controlCap(actMapPanRight, "→", e) + "  " + mm.controlCap(actMapZoomOut, "-", e),
		mm.placeRow(e),
	}, controlsInner)[:scrubBoxRows]
}

// mapRegionNames are the region row's names (D-103), in geo's numbering.
var mapRegionNames = []string{"CONT. US", "Alaska", "Hawaii", "US Caribbean", "American Samoa", "Guam / N. Marianas"}

// regionsRow is the region keys as a row of their own (D-103): MAPS: and
// each number's region, shorter names where the full ones do not fit, the
// region the map is bound to said shown (AX-6).
func (mm mapModel) regionsRow(width int, e mapEnv) string {
	build := func(names []string) string {
		parts := []string{"MAPS:"}
		for i, act := range mapRegionActs {
			key := mm.firstKey(act)
			if key == "" {
				continue
			}
			name := names[i]
			if r, ok := geo.RegionNumbered(i + 1); ok && mm.mapPane.region.Name != "" && r.Name == mm.mapPane.region.Name {
				name += " (shown)"
			}
			parts = append(parts, mm.controlCap(act, key, e)+" "+name)
		}
		return strings.Join(parts, "   ")
	}
	row := build(mapRegionNames)
	if render.Width(row) > width {
		row = build(mapRegionShort)
	}
	return render.TruncateCells(row, width)
}

// chipsRow is the window's keys as chips: Description, the mode, Overlays
// (D-63, D-65, D-94; no legend chip, D-103).
func (mm mapModel) chipsRow(width int, e mapEnv) string {
	return render.TruncateCells(strings.Join(mm.mapChips(e), "   "), width)
}
