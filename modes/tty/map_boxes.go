package tty

// map_boxes.go — what sits over the map (0.18.0, UAT-1): the [A] Description
// box over its upper left (U1-7, D-63), the controls over its lower right
// with the pressed chip blinking (U1-11), the [O] Overlays menu of the
// weather layers and the map's own detail (U1-9, U1-10, D-65), and the title
// that names what is in view (U1-12, D-64). The legend is the colour row
// under the map, drawn by map_scrub.go's scrubRows (0.18.0 D-103).
//
// EVERY BOX IS SPLICED OVER THE DRAWN LINES BY DISPLAY COLUMN, so the map's
// own colours either side of it are kept (render.SpliceCells, D-66), and every
// box is drawn in Update's lines, never by a library call in View (D-41).

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

const (
	actMapAlerts   term.Action = "map.alerts"   // U1-7: the Description box (D-365)
	actMapOverlays term.Action = "map.overlays" // U1-9: the Overlays menu
)

// spliceBox lays a box over lines with its top-left cell at (row, col). Rows
// past the lines are not drawn: the box is cut, the map never grows.
func spliceBox(lines, box []string, row, col int) []string {
	out := append([]string(nil), lines...)
	for i, b := range box {
		if r := row + i; r >= 0 && r < len(out) {
			out[r] = render.SpliceCells(out[r], b, col)
		}
	}
	return out
}

// boxed frames content in a box of an inner width, titled on its top edge.
func boxed(title string, content []string, inner int) []string {
	row := func(s string) string { return "│" + render.PadTo(render.TruncateCells(s, inner), inner) + "│" }
	top := "┌─ " + title + " "
	out := []string{render.TruncateCells(top+strings.Repeat("─", max(inner+1-render.Width(top), 0)), inner+1) + "┐"}
	for _, c := range content {
		out = append(out, row(c))
	}
	return append(out, "└"+strings.Repeat("─", inner)+"┘")
}

// descriptionBoxInner is the Description box's inner width for a map this wide:
// about two fifths of it, never under 26 cells or over 46.
func descriptionBoxInner(cols int) int { return min(max(cols*2/5, 26), 46, max(cols-4, 1)) }

// boxAlerts is how many alerts the box says at a map's height: about two
// lines each in two thirds of the map, the place's line and the frame aside;
// the last says how many more there are (D-78).
func boxAlerts(size tuimaps.Size) int { return max((size.Rows*2/3-4)/2, 2) }

// mapDescOverflow ends a description too long for its box: where the whole
// text is (D-365).
const mapDescOverflow = "… the rest: Settings, Maps, Description, Instead of the map"

// descriptionBox is the description in a box (D-63). Longer than two thirds of the map,
// it ends with a line saying where the whole text is.
func (mm mapModel) descriptionBox(size tuimaps.Size, e mapEnv) []string {
	inner := descriptionBoxInner(size.Cols)
	var content []string
	for _, l := range mm.describeUpTo(boxAlerts(size), e) {
		content = append(content, render.WrapText(l, inner-2)...)
	}
	for i := range content {
		content[i] = " " + content[i]
	}
	if most := max(size.Rows*2/3-2, 2); len(content) > most {
		var tail []string // D-365: the remedy named, whole, in the rows it needs
		for _, l := range render.WrapText(mapDescOverflow, inner-2) {
			tail = append(tail, " "+l)
		}
		content = append(content[:max(most-len(tail), 1)], tail...)
	}
	return boxed("Description", content, inner)
}

// withDescriptionBox lays the box over the map's upper left while it is open.
func (mm mapModel) withDescriptionBox(lines []string, size tuimaps.Size, e mapEnv) []string {
	if !mm.mapPane.alertsOn || mm.mapPane.menuOn || len(lines) == 0 {
		return lines // the menu takes the corner while it is open
	}
	return spliceBox(lines, mm.descriptionBox(size, e), 0, insetCols)
}

// insetCols is where the map's first cell is in a drawn line: the first
// column, since the map runs border to border (UAT-1 U1-27).
const insetCols = 0

// controlsInner is the controls box's inner width.
const controlsInner = 17

// controlsBox is the map's keys as chips (U1-11), the one pressed inverted
// until its blink ends - the same acknowledgement the Settings pickers give.
func (mm mapModel) controlsBox(e mapEnv) []string {
	o := e.Opts
	return boxed("Controls", []string{
		"    " + mm.controlCap(actMapPanUp, "↑", e) + "     " + mm.controlCap(actMapZoomIn, "+", e),
		" " + mm.controlCap(actMapPanLeft, "←", e) + mm.controlCap(actMapPanDown, "↓", e) + mm.controlCap(actMapPanRight, "→", e) + "  " + mm.controlCap(actMapZoomOut, "-", e),
		mm.placeRow(e),
		" " + regionCaps(o, slices.Contains(mapRegionActs, mm.mapPane.flash) && time.Now().Before(mm.mapPane.flashEnd)) + " region", // D-77: the region keys, prominently
	}, controlsInner)
}

// placeRow is the controls box's place line: [ and ] where the UI mode
// steps through the places; where it does not (D-195), what the red ring is.
func (mm mapModel) placeRow(e mapEnv) string {
	if !mm.mapRole(e).placeKeys {
		return " red: service"
	}
	return " " + mm.controlCap(actMapPrev, "[", e) + mm.controlCap(actMapNext, "]", e) + " place"
}

// controlCap is a key's chip, inverted while it blinks (U1-11); a region
// key blinks all six chips' - the one pressed among them.
func (mm mapModel) controlCap(act term.Action, face string, e mapEnv) string {
	o := e.Opts
	if mm.mapPane.flash == act && clockOf(e.Now).Before(mm.mapPane.flashEnd) {
		return o.KeyCapInverted(face)
	}
	return o.KeyCap(face)
}

// regionCaps is the region keys' chips, "1-6", both inverted while any
// region key blinks.
func regionCaps(o render.Opts, blink bool) string {
	face := o.KeyCap
	if blink {
		face = o.KeyCapInverted
	}
	return face("1") + "-" + face("6")
}

// withControls lays the controls over the map's lower right, where there is
// room for them beside the legend: a map under 14 rows or 60 columns draws
// none (its keys are in Help and on the status line).
func (mm mapModel) withControls(lines []string, size tuimaps.Size, e mapEnv) []string {
	return spliceAt(lines, size, func(s tuimaps.Size) ([]string, int, int, bool) { return mm.controlsPlace(s, e) })
}

// spliceAt lays the box a placement names over the map, while it shows.
func spliceAt(lines []string, size tuimaps.Size, place func(tuimaps.Size) ([]string, int, int, bool)) []string {
	box, row, col, ok := place(size)
	if !ok || len(lines) < size.Rows {
		return lines
	}
	return spliceBox(lines, box, row, col)
}

// controlsPlace is the controls box and where it lies, while it shows. ABOVE
// THE LAST ROW: the library writes the scale and the credit there, and the
// credit is the attribution, which is never covered (FR-14, UAT-1 U1-26).
func (mm mapModel) controlsPlace(size tuimaps.Size, e mapEnv) (box []string, row, col int, ok bool) {
	if mm.radarTimelineOn() || size.Rows < 14 || size.Cols < 60 {
		return nil, 0, 0, false
	}
	box = mm.controlsBox(e)
	return box, size.Rows - 1 - len(box), insetCols + size.Cols - (controlsInner + 2), true
}

// edgeChipText is the chip's words: the region beyond, with the arrow that
// crosses to it on the side it points (D-81) - "US CARIBBEAN →", "← HAWAII".
func (mm mapModel) edgeChipText() string {
	next, ok := geo.Neighbour(mm.mapPane.region.Name, mm.mapPane.edge)
	if !ok {
		return ""
	}
	name := strings.ToUpper(mapRegionLabels[geo.NumberOf(next.Name)-1])
	switch mm.mapPane.edge {
	case geo.West:
		return "← " + name
	case geo.North:
		return "↑ " + name
	case geo.South:
		return "↓ " + name
	}
	return name + " →"
}

// edgeWords are the edge chip as a sentence, for a window without the
// picture (AX-5): which edge, and the region the next press crosses to.
func (mm mapModel) edgeWords() string {
	if !mm.mapPane.edgeShown {
		return ""
	}
	next, ok := geo.Neighbour(mm.mapPane.region.Name, mm.mapPane.edge)
	if !ok {
		return ""
	}
	side := "east"
	switch mm.mapPane.edge {
	case geo.West:
		side = "west"
	case geo.North:
		side = "north"
	case geo.South:
		side = "south"
	}
	return "At the " + side + " edge: press again to cross to " + mapRegionLabels[geo.NumberOf(next.Name)-1] + "."
}

// withEdgeChip lays the chip at the edge it names while it shows: east and
// west at the middle of that side, north at the top's middle, south above the
// credit row (FR-14, never covered).
func (mm mapModel) withEdgeChip(lines []string, size tuimaps.Size) []string {
	return spliceAt(lines, size, mm.edgeChipPlace)
}

// edgeChipPlace is the edge chip and where it lies, while it shows.
func (mm mapModel) edgeChipPlace(size tuimaps.Size) (chip []string, row, col int, ok bool) {
	text := mm.edgeChipText()
	if !mm.mapPane.edgeShown || text == "" || size.Rows < 5 {
		return nil, 0, 0, false
	}
	chip = chipBox(text)
	w := render.Width(text) + 2
	row, col = size.Rows/2-1, insetCols+(size.Cols-w-2)/2
	switch mm.mapPane.edge {
	case geo.East:
		col = insetCols + size.Cols - (w + 2)
	case geo.West:
		col = insetCols
	case geo.North:
		row = 0
	case geo.South:
		row = size.Rows - 1 - len(chip)
	}
	return chip, row, col, true
}

// modeChipText is the chip D-103 shows when Forecast mode turned a main
// overlay on for the listener: which one.
const modeChipText = "FORECAST MODE: TEMP ENABLED"

// withModeChip lays the mode's chip at the map's top centre while it shows
// (D-103): Forecast mode turned temperature on, so the map is not blank. Any
// key takes it away, as the edge chip's (D-81).
func (mm mapModel) withModeChip(lines []string, size tuimaps.Size) []string {
	if !mm.mapPane.modeChip || len(lines) < 3 {
		return lines
	}
	chip := chipBox(modeChipText)
	return spliceBox(lines, chip, 0, insetCols+(size.Cols-render.Width(chip[0]))/2)
}

// openMeteoLayers are the overlays drawn from Open-Meteo's forecast and
// marine services, whose quota the notice speaks of (D-181); Air quality's
// service has a quota of its own.
var openMeteoLayers = []string{TemperatureLayer, FeelsLayer, WindLayer, UVLayer, RainLayer, WaveLayer}

// quotaNoticeText is the map's notice of Open-Meteo's spent quota, in the HUM
// LEAD's words (D-182, D-183), while an overlay it would draw is on: every
// such overlay drawn from a fallback, "Falling back to ‹source›"; none,
// "Resets ‹time›" (the listener's clock, with the day where not today); both,
// "Resets ‹time› // ‹n› Fall-back(s) active". "" when there is nothing to say.
func (mm mapModel) quotaNoticeText(e mapEnv) string {
	q := mm.mapPane.temp.Quota
	if q == nil {
		return ""
	}
	fallbacks, missing, sources := mm.quotaLayers(e)
	if fallbacks == 0 && missing == 0 {
		return ""
	}
	head := "! OPEN-METEO: Quota Exceeded; "
	if missing == 0 && len(sources) > 0 {
		return head + "Falling back to " + strings.Join(sources, " & ")
	}
	if missing == 0 {
		return head + "Falling back to recorded data" // the history alone (D-166)
	}
	now := e.Now()
	at := q.Resets.In(now.Location())
	when := e.Opts.Clock.Time(at)
	if y, m, dd := at.Date(); y != now.Year() || m != now.Month() || dd != now.Day() {
		when = e.Opts.Clock.WeekdayDateTime(at)
	}
	if fallbacks == 0 {
		return head + "Resets " + when
	}
	return head + "Resets " + when + " // " + strconv.Itoa(fallbacks) + " Fall-back" + frame.Plural(fallbacks) + " active"
}

// quotaLayers counts the Open-Meteo overlays on by their state, from the
// sources they are drawn from (D-183): drawn from another source, a
// fallback; drawing nothing, missing; drawn from Open-Meteo still - its
// last answer, held - neither. sources are the fallbacks' own, once each.
func (mm mapModel) quotaLayers(e mapEnv) (fallbacks, missing int, sources []string) {
	mode := mm.mapMode(e)
	for _, key := range openMeteoLayers { // six (P10-02)
		if !mm.layerOn(key, e) || rainIsRadar(key, mode) {
			continue
		}
		chips := mm.mapPane.temp.Chips[key]
		switch {
		case len(chips) == 0:
			missing++
		case slices.Contains(chips, "O-METEO"):
		default:
			fallbacks++
			for _, c := range chips { // a layer's few (P10-02)
				if c != "RECORDED" && !slices.Contains(sources, c) { // the history is not a source (D-173)
					sources = append(sources, c)
				}
			}
		}
	}
	return fallbacks, missing, sources
}

// withQuotaNotice lays the quota's notice at the map's lower right, on the
// row above the credit row, on its dark-orange ground (D-182) - giving way to
// a control (D-181): above the controls box while it shows, and above an
// edge chip it would meet. A map too short to hold it above them draws none.
func (mm mapModel) withQuotaNotice(lines []string, size tuimaps.Size, e mapEnv) []string {
	text := mm.quotaNoticeText(e)
	if text == "" || len(lines) < size.Rows {
		return lines
	}
	box := chipBox(text)
	w := render.Width(box[0])
	row, col := size.Rows-1-len(box), insetCols+size.Cols-w
	if _, top, _, ok := mm.controlsPlace(size, e); ok {
		row = top - len(box)
	}
	if chip, top, left, ok := mm.edgeChipPlace(size); ok && top < row+len(box) && top+len(chip) > row && left < col+w && left+render.Width(chip[0]) > col {
		row = top - len(box)
	}
	if row < 0 {
		return lines
	}
	tones := render.ChipTones(render.MapNoticeQuotaBG)
	for i, l := range box {
		box[i] = render.TintRaw(l, tones)
	}
	return spliceBox(lines, box, row, col)
}

// flashMapKey marks a map key pressed, for the controls box to blink.
func (mm mapModel) flashMapKey(act term.Action, e mapEnv) mapModel {
	mm.mapPane.flash, mm.mapPane.flashEnd = act, markEndsAt(e.Now, e.ReduceMotion)
	return mm
}

// mapDetailLayer is one of the library's basemap layers the listener can
// switch (D-65): its key in the file, its words, the library's layer, the
// least detail level whose preset turns it on, and when it first shows.
type mapDetailLayer struct {
	key, label string
	layer      tuimaps.Layer
	level      tuimaps.Detail // the least level whose preset switches it on (D-79)
	shows      string         // when the style first draws it, said beside it: "" when it always does
	overlay    bool           // drawn by the map as an overlay, not a layer of the library's basemap (D-199)
}

// detailNever is a preset no level reaches: a switch that is off until
// turned on, whatever the level (D-199).
const detailNever = tuimaps.DetailFull + 1

// mapDetailLayers is the map's own detail, in the menu's order. THE SWITCHES
// ARE THE TRUTH (D-79): the detail level is a preset that sets them, and a
// switch that is on is drawn whatever the level. "roads" is the major roads
// (go-tuiMaps D-82). MINOR ROADS ARE NOT OFFERED: the style draws them only
// at city zoom, which this map is not for, and a switch that seems to do
// nothing reads as broken (UAT-1 U1-42).
func mapDetailLayers() []mapDetailLayer {
	return []mapDetailLayer{
		{"borders", "Borders", tuimaps.BorderLayer, tuimaps.DetailEssential, "", false},
		{"water", "Lakes", tuimaps.WaterLayer, tuimaps.DetailEssential, "", false}, // inland water: the sea and its coast are never switched (U1-39, go-tuiMaps D-85)
		{"rivers", "Rivers", tuimaps.RiverLayer, tuimaps.DetailWeather, "", false},
		{"names", "Place names", tuimaps.LabelLayer, tuimaps.DetailWeather, "", false},
		{"roads", "Major roads", tuimaps.RoadLayer, tuimaps.DetailWeather, "", false},
		{"rail", "Rail", tuimaps.RailLayer, tuimaps.DetailStandard, "county zoom", false},  // the style's zoom 8; the county scale is 9
		{"parks", "Parks", tuimaps.ParkLayer, tuimaps.DetailStandard, "state zoom", false}, // the style's zoom 6, the state scale
		{nightDetailKey, "Day / night", 0, detailNever, "", true},                          // D-199: the night side shaded, its edge dotted; always so in the Propagation mode
	}
}

// mapDetailLevels are the levels in the picker's order (D-67).
func mapDetailLevels() []tuimaps.Detail {
	return []tuimaps.Detail{tuimaps.DetailEssential, tuimaps.DetailWeather, tuimaps.DetailFull} // D-144: Standard and Full would draw the same offered switches
}

// detailLevelByKey reads the file's word; a saved "standard" reads as All
// (D-144), and anything else Weather's, the default (D-67).
func detailLevelByKey(key string) tuimaps.Detail {
	if key == tuimaps.DetailStandard.String() {
		return tuimaps.DetailFull
	}
	for _, l := range mapDetailLevels() {
		if l.String() == key {
			return l
		}
	}
	return tuimaps.DetailWeather
}

// detailLevelLabel is a preset's words (D-144): Minimal, Standard or All.
func detailLevelLabel(l tuimaps.Detail) string {
	switch l {
	case tuimaps.DetailEssential:
		return "Minimal"
	case tuimaps.DetailWeather:
		return "Standard"
	}
	return "All"
}

// cycleDetailLevel moves the level round the four and sets every switch to
// its preset (D-79).
func (mm mapModel) cycleDetailLevel(forward bool, e mapEnv) mapModel {
	night := mm.detailOn(nightDetailKey)
	mm.mapDetailLevel = cycleIn(mapDetailLevels(), itself, mm.mapDetailLevel, forward)
	mm.mapDetailChoice = "" // the preset: every switch as the level sets it ...
	if night {
		mm.mapDetailChoice = layerChoiceKey(map[string]bool{nightDetailKey: true}) // ... but Day / night, which no level sets (D-199)
	}
	return mm.applyDetail(e)
}

// detailShows is what a layer says beside its switch: when the style first
// draws it, so a switch that is on but not yet seen does not read as broken.
func detailShows(l mapDetailLayer) string {
	if l.shows == "" {
		return ""
	}
	return " (" + l.shows + ")"
}

// detailLevelShown is the level's words, or "Custom" once a switch differs
// from the level's preset (D-79).
func (mm mapModel) detailLevelShown() string {
	for _, l := range mapDetailLayers() {
		if !l.overlay && mm.detailOn(l.key) != (l.level <= mm.mapDetailLevel) { // Day / night is no level's (D-199)
			return "Custom"
		}
	}
	return detailLevelLabel(mm.mapDetailLevel)
}

// detailOn reports whether a detail layer is drawn: the listener's switch,
// or the level's preset (D-79).
func (mm mapModel) detailOn(key string) bool {
	if on, ok := choiceOf(mm.mapDetailChoice, key); ok {
		return on
	}
	for _, l := range mapDetailLayers() {
		if l.key == key {
			return l.level <= mm.mapDetailLevel
		}
	}
	return true
}

// chosenOn reports whether a choice word holds key, switched on: not chosen
// reads as off.
func chosenOn(word, key string) bool {
	on, ok := choiceOf(word, key)
	return ok && on
}

// choiceOf reads one key from a choice word ("roads=on,rail=off").
func choiceOf(word, key string) (on, ok bool) {
	for part := range strings.SplitSeq(word, ",") { // no slice: a frame asks this for every layer
		if k, v, found := strings.Cut(part, "="); found && k == key {
			return v == "on", true
		}
	}
	return false, false
}

// choicesOf is a choice word as the file keeps it.
func choicesOf(word string) map[string]bool {
	if word == "" {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(word, ",") {
		if k, v, ok := strings.Cut(part, "="); ok {
			out[k] = v == "on"
		}
	}
	return out
}

// applyDetail puts every detail layer as chosen on the library's map: on each
// open, and after a switch.
func (mm mapModel) applyDetail(e mapEnv) mapModel {
	m := mm.mapPane.m
	if m == nil {
		return mm
	}
	// THE LIBRARY DRAWS ALL IT HAS; THE SWITCHES THIN IT (D-79). Its own level
	// would override a switch the listener turned on.
	all := tuimaps.DetailFull // the record names what is passed, so a test reads the level the library was given
	mm.mapPane.call("SetDetail:"+all.String(), func() { _ = m.SetDetail(all) })
	mm.mapPane.call("Layers:minor-roads:off", func() { m.Layers(tuimaps.MinorRoadLayer, false) }) // city zoom only: not offered (U1-42)
	for _, l := range mapDetailLayers() {
		if l.overlay {
			continue // the map's own overlay, handed in by setNight below
		}
		on, layer := mm.detailOn(l.key), l.layer
		word := "off"
		if on {
			word = "on"
		}
		mm.mapPane.call("Layers:"+l.key+":"+word, func() { m.Layers(layer, on) })
	}
	return mm.setNight(e) // Day / night is the map's own overlay (D-199)
}

// setDetail switches a detail layer and keeps the choice.
func (mm mapModel) setDetail(key string, on bool, e mapEnv) mapModel {
	choice := choicesOf(mm.mapDetailChoice)
	if choice == nil {
		choice = map[string]bool{}
	}
	choice[key] = on
	mm.mapDetailChoice = layerChoiceKey(choice)
	return mm.applyDetail(e)
}

// rowKind is what a row of the Overlays menu is (U2-39, D-141 to D-146).
type rowKind uint8

const (
	menuRadio  rowKind = iota + 1 // a tint: one at a time, space on the chosen clears it (D-142)
	menuGroup                     // a group's switch, ← Enabled → (D-143)
	menuCheck                     // a layer or an alert category, two to a line
	menuFire                      // Fire: a checkbox and its choice, ← All → (D-145)
	menuPreset                    // the detail preset, ← Standard → (D-144)
	menuDetail                    // a detail switch, two to a line
)

// overlayRow is one row of the Overlays menu, in the order ↑↓ moves through
// it; checkboxes are laid two to a line, but each is a row of its own.
type overlayRow struct {
	kind       rowKind
	key, label string
	group      string // the group a checkbox belongs to, whose switch dims it
	overlay    bool   // in the OVERLAYS section: a registry layer, its group, or the Propagation mode's row; otherwise MAP DETAIL
}

// oneTint are the layers that share one tint: switching one on switches the
// others off (D-119; UV and air quality, D-137, D-139).
var oneTint = []string{TemperatureLayer, FeelsLayer, UVLayer, AirLayer}

// The menu's groups, in its order (D-141): the tints; the data points; the
// hazards; the alert areas, the alert layer's own switch.
var (
	pointLayers  = []string{WindLayer, WaveLayer, BuoyLayer, TideLayer, RainLayer}
	hazardLayers = []string{QuakeLayer, FireLayer}
)

// overlayRows are the menu's rows: OVERLAYS - the tints, then Data Points,
// Hazards and Alert Areas, each its switch over its boxes - and MAP DETAIL -
// the preset over its switches. A layer the app registered that no group
// names is a data point, so a new layer plugs in without edits (W1.13).
func (mm mapModel) overlayRows(e mapEnv) []overlayRow {
	registered := map[string]string{}
	var order []string
	for _, l := range e.Layers {
		registered[l.Key] = l.Label
		order = append(order, l.Key)
	}
	var out []overlayRow
	for _, r := range []struct{ key, label string }{{TemperatureLayer, "Temperature"}, {UVLayer, "UV Index"}, {AirLayer, "Air Quality"}} {
		if _, ok := registered[r.key]; ok {
			out = append(out, overlayRow{kind: menuRadio, key: r.key, label: r.label, overlay: true})
		}
	}
	if mm.mapHooks.PropagationUpdate != nil {
		out = append(out, overlayRow{kind: menuRadio, key: propagationRowKey, label: propagationRowLabel, overlay: true}) // D-158: the mode as a tint, the OVERLAYS group's fourth (D-161, D-162)
	}
	if mm.mapMode(e) == modePropagation { // D-159: the tints and the map's detail; the weather's groups wait, their ticks kept
		return append(out, detailRows(nightDetailKey)...) // the night is always drawn in this mode: no switch to offer (D-199)
	}
	placed := map[string]bool{RadarLayer: true, TemperatureLayer: true, FeelsLayer: true, UVLayer: true, AirLayer: true, AlertLayer: true}
	mode := mm.mapMode(e)
	group := func(key, label string, layers []string, fill bool) {
		var members []overlayRow
		for _, k := range layers {
			if _, ok := registered[k]; !ok || rainIsRadar(k, mode) {
				placed[k] = true // Rain & snow is Forecast mode's alone: Radar mode's rain is the radar (D-117)
				continue
			}
			kind := menuCheck
			if k == FireLayer {
				kind = menuFire
			}
			label := registered[k]
			if k == QuakeLayer {
				label = "Quakes" // as the HUM LEAD wrote it (U2-39)
			}
			members, placed[k] = append(members, overlayRow{kind: kind, key: k, label: label, group: key, overlay: true}), true
		}
		if fill {
			for _, k := range order {
				if !placed[k] && !slices.Contains(hazardLayers, k) {
					members, placed[k] = append(members, overlayRow{kind: menuCheck, key: k, label: registered[k], group: key, overlay: true}), true
				}
			}
		}
		if len(members) > 0 {
			out = append(append(out, overlayRow{kind: menuGroup, key: key, label: label, overlay: true}), members...)
		}
	}
	group(groupPoints, "Data Points", pointLayers, true)
	group(groupHazards, "Hazards", hazardLayers, false)
	if _, ok := registered[AlertLayer]; ok {
		out = append(out, alertAreaRows()...)
	}
	return append(out, detailRows("")...)
}

// alertAreaRows are the Alert Areas group: its header and [w]'s categories
// (D-80), each short, to fit two to a line.
func alertAreaRows() []overlayRow {
	out := []overlayRow{{kind: menuGroup, key: AlertLayer, label: "Alert Areas", overlay: true}}
	for _, c := range AlertCategories() {
		out = append(out, overlayRow{kind: menuCheck, key: AlertCategorySwitch(c.Key), label: strings.ToUpper(c.Key[:1]) + c.Key[1:], group: AlertLayer, overlay: true})
	}
	return out
}

// detailRows are the map's detail: its preset and a switch a layer, less the
// one a mode always draws, if any.
func detailRows(always string) []overlayRow {
	out := []overlayRow{{kind: menuPreset, key: detailLevelKey, label: "Preset"}}
	for _, l := range mapDetailLayers() {
		if l.key != always {
			out = append(out, overlayRow{kind: menuDetail, key: l.key, label: l.label})
		}
	}
	return out
}

// detailLevelKey is the Overlays menu's detail-preset row.
const detailLevelKey = "level"

// overlayMenuW is the menu's inner width, as the HUM LEAD drew it.
const overlayMenuW = 40

// menuWarning is the menu's performance warning, in the HUM LEAD's words
// (D-146).
const menuWarning = "You may experience performance issues with this many overlays and details enabled."

// overlaysBox is the menu (U2-39): the warning when the layers on would cost
// past the thresholds; OVERLAYS - the tints as radio rows, each group's
// switch over its boxes two to a line, a disabled group's boxes dimmed and
// keeping their ticks; MAP DETAIL - the preset over its switches.
func (mm mapModel) overlaysBox(e mapEnv) []string {
	o := e.Opts
	content := mm.menuWarningLines()
	content = append(content, mm.menuRowLines(e)...)
	content = append(content, detailShowsLines()...)
	content = append(content, "", " "+o.KeyCap("↑↓")+" move "+o.KeyCap("←→")+" choose "+o.KeyCap("space")+" switch")
	return boxed(render.Tint("MAP DETAILS / OVERLAYS", render.Tok(render.ModalTitle)), content, overlayMenuW)
}

// menuWarningLines is the menu's warning and a blank line under it, while
// the layers on would cost past the thresholds; none otherwise.
func (mm mapModel) menuWarningLines() []string {
	h, _ := costWarningParts(mm.mapCost)
	if h == "" {
		return nil
	}
	var out []string
	for i, l := range render.WrapText(menuWarning, overlayMenuW-6) {
		lead := "   "
		if i == 0 {
			lead = " " + render.Tint("!", render.Tok(render.ListPointer)) + " "
		}
		out = append(out, " "+lead+l)
	}
	return append(out, "")
}

// menuRowLines is the menu's rows under their section headings: OVERLAYS
// over MAP DETAIL.
func (mm mapModel) menuRowLines(e mapEnv) []string {
	head := func(s string) string { return render.Tint(s, render.Tok(render.ModalTitle)) } // as Settings' groups (D-103)
	rows := mm.overlayRows(e)
	chips := newArrowChips(e.Opts)
	var content []string
	heading := ""
	for i := 0; i < len(rows); i++ {
		r := rows[i]
		section := "MAP DETAIL"
		if r.overlay {
			section = "OVERLAYS"
		}
		if section != heading {
			if heading != "" {
				content = append(content, "")
			}
			content, heading = append(content, " "+head(section)), section
		}
		if r.kind == menuGroup && i > 0 && rows[i-1].kind != menuGroup {
			content = append(content, "")
		}
		if l, ok := mm.menuRowLine(r, i, chips, e); ok {
			content = append(content, l)
			continue
		}
		l, pair := mm.menuCellPair(rows, i, e)
		content = append(content, l)
		if pair {
			i++
		}
	}
	return content
}

// menuLine is a menu line with its right part flush to the box's edge.
func menuLine(left, right string) string {
	return render.PadTo(left, overlayMenuW-2-render.Width(right)) + right
}

// menuRowLine is a row that takes a line of its own - a tint, a group's
// switch, fire, the preset; false for a box drawn two to a line.
func (mm mapModel) menuRowLine(r overlayRow, i int, chips arrowChips, e mapEnv) (string, bool) {
	o := e.Opts
	mark := o.ListMark(i == mm.mapPane.menuAt)
	choice := func(s string) string { // THE APP'S PICKER, [←] value [→], blinking as pressed (D-147)
		return pickerCellW(s, chips, mm.menuFlashFor(i, e), menuChoiceW)
	}
	switch r.kind {
	case menuRadio:
		face := " " + mark + radioMark(mm.tintChosen(r.key, e), o.ASCII) + " " + r.label
		if r.key != TemperatureLayer {
			return face, true
		}
		pick := "Actual"
		if mm.pickFeels(e) {
			pick = "Feels like"
		}
		return menuLine(face, choice(pick)), true
	case menuGroup:
		state := "Enabled"
		if !mm.rowGroupOn(r.key, e) {
			state = "Disabled"
		}
		return menuLine(" "+mark+r.label, choice(state)), true
	case menuFire:
		mode := strings.ToUpper(mm.fireMode()[:1]) + mm.fireMode()[1:]
		return mm.dimmed(r, menuLine(" "+mark+checkMark(o, mm.ticked(r.key, e))+" "+r.label, choice(mode)), e), true
	case menuPreset:
		return menuLine(" "+mark+"Preset:", choice(mm.detailLevelShown())), true
	}
	return "", false
}

// menuCellPair is a box row, two to a line: row i and, when it is the same
// kind in the same group, row i+1 beside it (pair true).
func (mm mapModel) menuCellPair(rows []overlayRow, i int, e mapEnv) (string, bool) {
	o := e.Opts
	cellW := (overlayMenuW - 4) / 2
	cell := func(r overlayRow, at int) string {
		on := mm.detailOn(r.key)
		if r.overlay {
			on = mm.ticked(r.key, e)
		}
		return mm.dimmed(r, render.PadTo(o.ListMark(at == mm.mapPane.menuAt)+checkMark(o, on)+" "+r.label, cellW), e)
	}
	row := " " + cell(rows[i], i)
	if i+1 < len(rows) && rows[i+1].kind == rows[i].kind && rows[i+1].group == rows[i].group {
		return row + " " + cell(rows[i+1], i+1), true
	}
	return row, false
}

// detailShowsLines is what each switch says of the zoom its style first
// draws it at, so a switch on but not yet seen does not read as broken
// (U1-42); none when no switch has one.
func detailShowsLines() []string {
	var shows []string
	for _, l := range mapDetailLayers() {
		if l.shows != "" {
			shows = append(shows, l.label+" from "+strings.TrimSuffix(l.shows, " zoom")+" zoom")
		}
	}
	if len(shows) == 0 {
		return nil
	}
	out := []string{""} // apart from the switches, for scanning (D-147)
	for _, l := range render.WrapText(strings.Join(shows, ", ")+".", overlayMenuW-4) {
		out = append(out, "   "+render.Tint(l, render.Tok(render.TableMuted)))
	}
	return out
}

// dimmed is a box of a disabled group, dimmed: its tick kept (D-143).
func (mm mapModel) dimmed(r overlayRow, s string, e mapEnv) string {
	if r.group == "" || mm.rowGroupOn(r.group, e) {
		return s
	}
	return render.Tint(render.Plain(s), render.Tok(render.TableMuted))
}

// rowGroupOn is a group's switch: the alert layer's own for Alert Areas.
func (mm mapModel) rowGroupOn(group string, e mapEnv) bool {
	if group == AlertLayer {
		return mm.ticked(AlertLayer, e)
	}
	return mm.groupOn(group)
}

// tintChosen reports whether a radio row is the tint chosen: Temperature's
// is either of its measures (D-119).
func (mm mapModel) tintChosen(key string, e mapEnv) bool {
	if key == propagationRowKey {
		return mm.mapMode(e) == modePropagation
	}
	if key == TemperatureLayer {
		return mm.layerOn(TemperatureLayer, e) || mm.layerOn(FeelsLayer, e)
	}
	return mm.layerOn(key, e)
}

// pickFeels reports whether Temperature's row reads Feels like: the measure
// drawn, or the one last picked while none is.
func (mm mapModel) pickFeels(e mapEnv) bool {
	switch {
	case mm.layerOn(FeelsLayer, e):
		return true
	case mm.layerOn(TemperatureLayer, e):
		return false
	}
	return chosenOn(mm.mapLayerChoice, pickFeelsKey)
}

// menuChoiceW is the menu's pickers' value width: its longest, "Feels like".
const menuChoiceW = 10

// menuFlashFor is the blink of a menu row's picker: only the row pressed,
// only while its window is open - Settings' pickers' rule (D-147).
func (mm mapModel) menuFlashFor(row int, e mapEnv) pickerFlash {
	if row != mm.mapPane.menuFlashAt || mm.mapPane.menuFlash == flashNone || !clockOf(e.Now).Before(mm.mapPane.menuFlashEnd) {
		return flashNone
	}
	return mm.mapPane.menuFlash
}

// hasChoice reports whether a row has a picker: ←→ step it.
func hasChoice(r overlayRow) bool {
	return r.kind == menuGroup || r.kind == menuFire || r.kind == menuPreset || (r.kind == menuRadio && r.key == TemperatureLayer)
}

// pickFeelsKey keeps Temperature's measure while no tint is chosen.
const pickFeelsKey = "pick:feels"

// withOverlays lays the menu over the map's upper left while it is open.
func (mm mapModel) withOverlays(lines []string, e mapEnv) []string {
	if !mm.mapPane.menuOn || len(lines) == 0 {
		return lines
	}
	return spliceBox(lines, mm.overlaysBox(e), 0, insetCols)
}

// overlaysText is the menu without a picture to lay it over (D-267): the
// same rows as the box, the focused one marked, the keys that close it under
// them, and a blank line.
func (mm mapModel) overlaysText(e mapEnv) []string {
	o := e.Opts
	out := []string{render.Tint("MAP DETAILS / OVERLAYS", render.Tok(render.ModalTitle)), ""}
	out = append(out, mm.menuWarningLines()...)
	out = append(out, mm.menuRowLines(e)...)
	out = append(out, detailShowsLines()...)
	return append(out, "", " "+o.KeyCap("↑↓")+" move "+o.KeyCap("←→")+" choose "+o.KeyCap("space")+" switch "+o.KeyCap("esc")+" close", "")
}

// handleOverlaysKey is the menu's keys while it is open: it owns ↑↓, ←→,
// space, esc and its own key (modal control priority, D-61). ←→ change a
// row's choice; space switches a row.
func (mm mapModel) handleOverlaysKey(key string, e mapEnv) (mapModel, mapWant, bool) {
	rows := mm.overlayRows(e)
	if len(rows) == 0 {
		return mm, mapWant{}, key == "esc"
	}
	var w mapWant
	r := rows[mm.mapPane.menuAt%len(rows)]
	switch key {
	case "up":
		mm.mapPane.menuAt = (mm.mapPane.menuAt + len(rows) - 1) % len(rows)
	case "down":
		mm.mapPane.menuAt = (mm.mapPane.menuAt + 1) % len(rows)
	case "left", "right":
		if hasChoice(r) { // the chip pressed blinks, as Settings' pickers do (D-147)
			mm.mapPane.menuFlash, mm.mapPane.menuFlashAt, mm.mapPane.menuFlashEnd = flashLeft, mm.mapPane.menuAt, markEndsAt(e.Now, e.ReduceMotion)
			if key == "right" {
				mm.mapPane.menuFlash = flashRight
			}
		}
		mm, w = mm.chooseOnRow(r, key == "right", e)
	case "space", "enter":
		mm, w = mm.switchRow(r, e)
	case "esc":
		mm.mapPane.menuOn = false
	default:
		return mm, mapWant{}, false
	}
	mm.mapPane.gen++ // the menu is drawn with the window: a switch or a move redraws it
	return mm, w, true
}

// withChoice is the layer choices with one set, as their word.
func (mm mapModel) withChoice(set func(choice map[string]bool), e mapEnv) (mapModel, mapWant) {
	choice := choicesOf(mm.mapLayerChoice)
	if choice == nil {
		choice = map[string]bool{}
	}
	set(choice)
	mm.mapLayerChoice = layerChoiceKey(choice)
	return mm.refreshMapCost(e).requestFeed().wantDirty() // written as Settings writes it (uiApplyCmd)
}

// switchRow is space on a row.
func (mm mapModel) switchRow(r overlayRow, e mapEnv) (mapModel, mapWant) {
	switch r.kind {
	case menuRadio:
		key := r.key
		if key == TemperatureLayer && mm.pickFeels(e) {
			key = FeelsLayer
		}
		chosen := mm.tintChosen(r.key, e)
		mm.mapPane.tempAuto = false // the listener's switch from here on (D-104)
		return mm.withChoice(func(c map[string]bool) {
			for _, t := range oneTint {
				c[t] = false // one tint at a time (D-119, D-137, D-139); space on the chosen clears it (D-142)
			}
			if !chosen {
				c[key] = true
			}
		}, e)
	case menuGroup:
		on := mm.rowGroupOn(r.key, e)
		return mm.withChoice(func(c map[string]bool) { c[r.key] = !on }, e)
	case menuCheck, menuFire:
		on := mm.ticked(r.key, e)
		return mm.withChoice(func(c map[string]bool) { c[r.key] = !on }, e)
	case menuPreset:
		return mm.cycleDetailLevel(true, e).wantDirty()
	}
	return mm.setDetail(r.key, !mm.detailOn(r.key), e).wantDirty()
}

// chooseOnRow is ←→ on a row with a choice; on any other row, nothing - the
// open menu owns the keys.
func (mm mapModel) chooseOnRow(r overlayRow, forward bool, e mapEnv) (mapModel, mapWant) {
	switch r.kind {
	case menuRadio:
		if r.key != TemperatureLayer {
			return mm, mapWant{}
		}
		feels, chosen := !mm.pickFeels(e), mm.tintChosen(TemperatureLayer, e) // Actual and Feels like: two, either way is the other
		return mm.withChoice(func(c map[string]bool) {
			c[pickFeelsKey] = feels
			if chosen {
				c[TemperatureLayer], c[FeelsLayer] = !feels, feels // the tint drawn follows the measure (D-119)
			}
		}, e)
	case menuGroup:
		return mm.switchRow(r, e)
	case menuFire:
		next := cycleIn(fireModes, itself, mm.fireMode(), forward)
		return mm.withChoice(func(c map[string]bool) {
			for _, m := range fireModes[1:] {
				c["fire:"+m] = m == next
			}
		}, e)
	case menuPreset:
		return mm.cycleDetailLevel(forward, e).wantDirty()
	}
	return mm, mapWant{}
}

// mapTitleAt names what is in view (D-64): the app's namer for the view's
// centre and width, with the selected place while it is in view. With no
// namer, the selected place.
func (mm mapModel) mapTitleAt(size tuimaps.Size, e mapEnv) string {
	loc, m := mm.mapPlace(e), mm.mapPane.m
	if loc == nil {
		return ""
	}
	if m == nil {
		return loc.Label
	}
	v := mm.viewBox(size)
	if !v.Contains(loc.Lat, loc.Lon) {
		return mm.viewName(e) // D-78: away from the place, never its name
	}
	var name string
	if mm.mapHooks.MapAreaName != nil {
		centre, _ := m.Centre()
		name = mm.mapHooks.MapAreaName(centre, viewWidthKm(v, centre.Lat))
	}
	if name == "" || name == loc.Label {
		return loc.Label
	}
	return name + " " + e.Opts.Glyphs().Dot + " " + loc.Label
}

// MapView is the box of the map in view, in degrees (0.18.0 D-66): what
// "Alerts in view" asks the app about, and what the title names.
type MapView = geo.Box

// viewWidthKm is how wide a view is, in kilometres, across its centre's
// latitude: the width the area namer names it by, for the title and the
// spoken description alike (0.19.5 DR-10), so the two name one area.
func viewWidthKm(v MapView, centreLat float64) float64 {
	return (v.E - v.W) * 111.32 * math.Cos(centreLat*math.Pi/180)
}

// viewBox is the map's view at a size, from the library's centre and zoom
// and its published scale (256-dot tiles, a braille cell two dots wide and
// four high); nothing when no map is built.
func (mm mapModel) viewBox(size tuimaps.Size) MapView {
	m := mm.mapPane.m
	if m == nil {
		return MapView{}
	}
	centre, zoom := m.Centre()
	world := 256 * math.Exp2(zoom)                // dots round the world at this zoom
	lonSpan := float64(size.Cols*2) / world * 360 // degrees across the view
	ySpan := float64(size.Rows*4) / world         // the view's height, in the world's Mercator units
	yc := mercatorY(centre.Lat)
	return MapView{W: centre.Lon - lonSpan/2, E: centre.Lon + lonSpan/2,
		N: latOfMercator(yc - ySpan/2), S: latOfMercator(yc + ySpan/2)}
}

// latOfMercator is the latitude at a place down the world, mercatorY's inverse.
func latOfMercator(y float64) float64 {
	return math.Atan(math.Sinh(math.Pi*(1-2*y))) * 180 / math.Pi
}
