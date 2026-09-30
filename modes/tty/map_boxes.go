package tty

// map_boxes.go — what sits over the map (0.18.0, UAT-1): the [A] Area Alerts
// box over its upper left (U1-7, D-63), the controls over its lower right
// with the pressed chip blinking (U1-11), the [O] Overlays menu of the
// weather layers and the map's own detail (U1-9, U1-10, D-65), and the title
// that names what is in view (U1-12, D-64). The legend over the upper right is
// map_pane.go's (D-44).
//
// EVERY BOX IS SPLICED OVER THE DRAWN LINES BY DISPLAY COLUMN, so the map's
// own colours either side of it are kept (render.SpliceCells, D-66), and every
// box is drawn in Update's lines, never by a library call in View (D-41).

import (
	"math"
	"slices"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

const (
	actMapAlerts   term.Action = "map.alerts"   // U1-7: the Area Alerts box
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

// areaAlertsInner is the Area Alerts box's inner width for a map this wide:
// about two fifths of it, never under 26 cells or over 46.
func areaAlertsInner(cols int) int { return min(max(cols*2/5, 26), 46, max(cols-4, 1)) }

// boxAlerts is how many alerts the box says at a map's height: about two
// lines each in two thirds of the map, the place's line and the frame aside;
// the last says how many more there are (D-78).
func boxAlerts(size tuimaps.Size) int { return max((size.Rows*2/3-4)/2, 2) }

// areaAlertsBox is the description in a box (D-63). Longer than two thirds of the map,
// it ends with a line saying where the whole text is.
func (d Dashboard) areaAlertsBox(size tuimaps.Size) []string {
	inner := areaAlertsInner(size.Cols)
	var content []string
	for _, l := range d.describeUpTo(boxAlerts(size)) {
		content = append(content, render.WrapText(l, inner-2)...)
	}
	for i := range content {
		content[i] = " " + content[i]
	}
	if most := max(size.Rows*2/3-2, 2); len(content) > most {
		content = append(content[:most-1], " … the rest: see Settings")
	}
	return boxed("Area Alerts", content, inner)
}

// withAreaAlerts lays the box over the map's upper left while it is open.
func (d Dashboard) withAreaAlerts(lines []string, size tuimaps.Size) []string {
	if !d.mapPane.alertsOn || d.mapPane.menuOn || len(lines) == 0 {
		return lines // the menu takes the corner while it is open
	}
	return spliceBox(lines, d.areaAlertsBox(size), 0, insetCols)
}

// insetCols is where the map's first cell is in a drawn line: the first
// column, since the map runs border to border (UAT-1 U1-27).
const insetCols = 0

// controlsInner is the controls box's inner width.
const controlsInner = 17

// controlsBox is the map's keys as chips (U1-11), the one pressed inverted
// until its blink ends - the same acknowledgement the Settings pickers give.
func (d Dashboard) controlsBox() []string {
	o := d.opts()
	cap := d.controlCap
	return boxed("Controls", []string{
		"    " + cap(actMapPanUp, "↑") + "     " + cap(actMapZoomIn, "+"),
		" " + cap(actMapPanLeft, "←") + cap(actMapPanDown, "↓") + cap(actMapPanRight, "→") + "  " + cap(actMapZoomOut, "-"),
		" " + cap(actMapPrev, "[") + cap(actMapNext, "]") + " place",
		" " + regionCaps(o, slices.Contains(mapRegionActs, d.mapPane.flash) && time.Now().Before(d.mapPane.flashEnd)) + " region", // D-77: the region keys, prominently
	}, controlsInner)
}

// controlCap is a key's chip, inverted while it blinks (U1-11); a region
// key blinks all six chips' - the one pressed among them.
func (d Dashboard) controlCap(act term.Action, face string) string {
	o := d.opts()
	if d.mapPane.flash == act && time.Now().Before(d.mapPane.flashEnd) {
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
func (d Dashboard) withControls(lines []string, size tuimaps.Size) []string {
	if d.radarTimelineOn() || size.Rows < 14 || size.Cols < 60 || len(lines) < size.Rows {
		return lines
	}
	box := d.controlsBox()
	// ABOVE THE LAST ROW: the library writes the scale and the credit there,
	// and the credit is the attribution, which is never covered (FR-14,
	// UAT-1 U1-26).
	return spliceBox(lines, box, size.Rows-1-len(box), insetCols+size.Cols-(controlsInner+2))
}

// edgeChipText is the chip's words: the region beyond, with the arrow that
// crosses to it on the side it points (D-81) - "US CARIBBEAN →", "← HAWAII".
func (d Dashboard) edgeChipText() string {
	next, ok := geo.Neighbour(d.mapPane.region.Name, d.mapPane.edge)
	if !ok {
		return ""
	}
	name := strings.ToUpper(mapRegionLabels[geo.NumberOf(next.Name)-1])
	switch d.mapPane.edge {
	case geo.West:
		return "← " + name
	case geo.North:
		return "↑ " + name
	case geo.South:
		return "↓ " + name
	}
	return name + " →"
}

// withEdgeChip lays the chip at the edge it names while it shows: east and
// west at the middle of that side, north at the top's middle, south above the
// credit row (FR-14, never covered).
func (d Dashboard) withEdgeChip(lines []string, size tuimaps.Size) []string {
	text := d.edgeChipText()
	if !d.mapPane.edgeShown || text == "" || len(lines) < size.Rows || size.Rows < 5 {
		return lines
	}
	chip := chipBox(text)
	w := render.Width(text) + 2
	row, col := size.Rows/2-1, insetCols+(size.Cols-w-2)/2
	switch d.mapPane.edge {
	case geo.East:
		col = insetCols + size.Cols - (w + 2)
	case geo.West:
		col = insetCols
	case geo.North:
		row = 0
	case geo.South:
		row = size.Rows - 1 - len(chip)
	}
	return spliceBox(lines, chip, row, col)
}

// modeChipText is the chip D-103 shows when Forecast mode turned a main
// overlay on for the listener: which one.
const modeChipText = "FORECAST MODE: TEMP ENABLED"

// withModeChip lays the mode's chip at the map's top centre while it shows
// (D-103): Forecast mode turned temperature on, so the map is not blank. Any
// key takes it away, as the edge chip's (D-81).
func (d Dashboard) withModeChip(lines []string, size tuimaps.Size) []string {
	if !d.mapPane.modeChip || len(lines) < 3 {
		return lines
	}
	chip := chipBox(modeChipText)
	row := 0
	if d.mapPane.temp.Quota != nil {
		row = 3 // under the quota's notice, which holds the top (D-165)
	}
	return spliceBox(lines, chip, row, insetCols+(size.Cols-render.Width(chip[0]))/2)
}

// quotaNoticeText is the map's notice of a spent quota (W18.1, D-165), in
// the HUM LEAD's words: the period's limit, whose, and when it resets in the
// listener's clock - with the day where that is not today.
func (d Dashboard) quotaNoticeText() string {
	q := d.mapPane.temp.Quota
	if q == nil {
		return ""
	}
	now := d.now()
	at := q.Resets.In(now.Location())
	when := d.clockFmt.Time(at)
	if y, m, dd := at.Date(); y != now.Year() || m != now.Month() || dd != now.Day() {
		when = d.clockFmt.WeekdayDateTime(at)
	}
	return "! " + q.Period + " " + q.Source + " API Usage Exceeded. Resets " + when
}

// withQuotaNotice lays the spent quota's notice at the map's top centre,
// on its dark-orange ground (D-165), while the answer says a quota is spent.
func (d Dashboard) withQuotaNotice(lines []string, size tuimaps.Size) []string {
	text := d.quotaNoticeText()
	if text == "" || len(lines) < 3 {
		return lines
	}
	box := chipBox(text)
	tones := render.ChipTones(render.MapNoticeQuotaBG)
	for i, l := range box {
		box[i] = render.TintRaw(l, tones)
	}
	return spliceBox(lines, box, 0, insetCols+(size.Cols-render.Width(box[0]))/2)
}

// flashMapKey marks a map key pressed, for the controls box to blink.
func (d Dashboard) flashMapKey(act term.Action) Dashboard {
	d.mapPane.flash, d.mapPane.flashEnd = act, time.Now().Add(pickerFlashDur)
	return d
}

// mapDetailLayer is one of the library's basemap layers the listener can
// switch (D-65): its key in the file, its words, the library's layer, the
// least detail level whose preset turns it on, and when it first shows.
type mapDetailLayer struct {
	key, label string
	layer      tuimaps.Layer
	level      tuimaps.Detail // the least level whose preset switches it on (D-79)
	shows      string         // when the style first draws it, said beside it: "" when it always does
}

// mapDetailLayers is the map's own detail, in the menu's order. THE SWITCHES
// ARE THE TRUTH (D-79): the detail level is a preset that sets them, and a
// switch that is on is drawn whatever the level. "roads" is the major roads
// (go-tuiMaps D-82). MINOR ROADS ARE NOT OFFERED: the style draws them only
// at city zoom, which this map is not for, and a switch that seems to do
// nothing reads as broken (UAT-1 U1-42).
func mapDetailLayers() []mapDetailLayer {
	return []mapDetailLayer{
		{"borders", "Borders", tuimaps.BorderLayer, tuimaps.DetailEssential, ""},
		{"water", "Lakes", tuimaps.WaterLayer, tuimaps.DetailEssential, ""}, // inland water: the sea and its coast are never switched (U1-39, go-tuiMaps D-85)
		{"rivers", "Rivers", tuimaps.RiverLayer, tuimaps.DetailWeather, ""},
		{"names", "Place names", tuimaps.LabelLayer, tuimaps.DetailWeather, ""},
		{"roads", "Major roads", tuimaps.RoadLayer, tuimaps.DetailWeather, ""},
		{"rail", "Rail", tuimaps.RailLayer, tuimaps.DetailStandard, "county zoom"},  // the style's zoom 8; the county scale is 9
		{"parks", "Parks", tuimaps.ParkLayer, tuimaps.DetailStandard, "state zoom"}, // the style's zoom 6, the state scale
	}
}

// mapDetailLevels are the levels in the picker's order (D-67).
func mapDetailLevels() []tuimaps.Detail {
	return []tuimaps.Detail{tuimaps.DetailEssential, tuimaps.DetailWeather, tuimaps.DetailFull} // D-144: Standard and Full drew the same offered switches
}

// detailLevelByKey reads the file's word; a saved "standard" is All now
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
func (d Dashboard) cycleDetailLevel(forward bool) Dashboard {
	levels := mapDetailLevels()
	at := 0
	for i, l := range levels {
		if l == d.mapDetailLevel {
			at = i
		}
	}
	step := 1
	if !forward {
		step = len(levels) - 1
	}
	d.mapDetailLevel = levels[(at+step)%len(levels)]
	d.mapDetailChoice = "" // the preset: every switch as the level sets it
	return d.applyDetail()
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
func (d Dashboard) detailLevelShown() string {
	for _, l := range mapDetailLayers() {
		if d.detailOn(l.key) != (l.level <= d.mapDetailLevel) {
			return "Custom"
		}
	}
	return detailLevelLabel(d.mapDetailLevel)
}

// detailOn reports whether a detail layer is drawn: the listener's switch,
// or the level's preset (D-79).
func (d Dashboard) detailOn(key string) bool {
	if on, ok := choiceOf(d.mapDetailChoice, key); ok {
		return on
	}
	for _, l := range mapDetailLayers() {
		if l.key == key {
			return l.level <= d.mapDetailLevel
		}
	}
	return true
}

// choiceOf reads one key from a choice word ("roads=on,rail=off").
func choiceOf(word, key string) (on, ok bool) {
	for _, part := range strings.Split(word, ",") {
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
func (d Dashboard) applyDetail() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	// THE LIBRARY DRAWS ALL IT HAS; THE SWITCHES THIN IT (D-79). Its own level
	// would override a switch the listener turned on.
	all := tuimaps.DetailFull // the record names what is passed, so a test reads the level the library was given
	d.mapPane.call("SetDetail:"+all.String(), func() { _ = m.SetDetail(all) })
	d.mapPane.call("Layers:minor-roads:off", func() { m.Layers(tuimaps.MinorRoadLayer, false) }) // city zoom only: not offered (U1-42)
	for _, l := range mapDetailLayers() {
		on, layer := d.detailOn(l.key), l.layer
		word := "off"
		if on {
			word = "on"
		}
		d.mapPane.call("Layers:"+l.key+":"+word, func() { m.Layers(layer, on) })
	}
	return d
}

// setDetail switches a detail layer and keeps the choice.
func (d Dashboard) setDetail(key string, on bool) Dashboard {
	choice := choicesOf(d.mapDetailChoice)
	if choice == nil {
		choice = map[string]bool{}
	}
	choice[key] = on
	d.mapDetailChoice = layerChoiceKey(choice)
	return d.applyDetail()
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
	weather    bool   // a registry layer or its group; otherwise the map's own detail
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
func (d Dashboard) overlayRows() []overlayRow {
	registered := map[string]string{}
	var order []string
	for _, l := range d.cfg.MapLayers {
		registered[l.Key] = l.Label
		order = append(order, l.Key)
	}
	var out []overlayRow
	for _, r := range []struct{ key, label string }{{TemperatureLayer, "Temperature"}, {UVLayer, "UV Index"}, {AirLayer, "Air Quality"}} {
		if _, ok := registered[r.key]; ok {
			out = append(out, overlayRow{kind: menuRadio, key: r.key, label: r.label, weather: true})
		}
	}
	placed := map[string]bool{RadarLayer: true, TemperatureLayer: true, FeelsLayer: true, UVLayer: true, AirLayer: true, AlertLayer: true}
	group := func(key, label string, layers []string, fill bool) {
		var members []overlayRow
		for _, k := range layers {
			if _, ok := registered[k]; !ok || (k == RainLayer && d.radarMode()) {
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
			members, placed[k] = append(members, overlayRow{kind: kind, key: k, label: label, group: key, weather: true}), true
		}
		if fill {
			for _, k := range order {
				if !placed[k] && !slices.Contains(hazardLayers, k) {
					members, placed[k] = append(members, overlayRow{kind: menuCheck, key: k, label: registered[k], group: key, weather: true}), true
				}
			}
		}
		if len(members) > 0 {
			out = append(append(out, overlayRow{kind: menuGroup, key: key, label: label, weather: true}), members...)
		}
	}
	group(groupPoints, "Data Points", pointLayers, true)
	group(groupHazards, "Hazards", hazardLayers, false)
	if _, ok := registered[AlertLayer]; ok {
		out = append(out, overlayRow{kind: menuGroup, key: AlertLayer, label: "Alert Areas", weather: true})
		for _, c := range AlertCategories() { // D-80: [w]'s categories
			out = append(out, overlayRow{kind: menuCheck, key: categoryChoice(c.Key), label: strings.ToUpper(c.Key[:1]) + c.Key[1:], group: AlertLayer, weather: true}) // short, to fit two to a line
		}
	}
	out = append(out, overlayRow{kind: menuPreset, key: detailLevelKey, label: "Preset"})
	for _, l := range mapDetailLayers() {
		out = append(out, overlayRow{kind: menuDetail, key: l.key, label: l.label})
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
func (d Dashboard) overlaysBox() []string {
	o := d.opts()
	head := func(s string) string { return render.Tint(s, render.Tok(render.ModalTitle)) } // as Settings' groups (D-103)
	var content []string
	if h, _ := costWarningParts(d.mapCost); h != "" {
		for i, l := range render.WrapText(menuWarning, overlayMenuW-6) {
			lead := "   "
			if i == 0 {
				lead = " " + render.Tint("!", render.Tok(render.ListPointer)) + " "
			}
			content = append(content, " "+lead+l)
		}
		content = append(content, "")
	}
	rows := d.overlayRows()
	cellW := (overlayMenuW - 4) / 2
	chips := newArrowChips(o)
	at := -1                          // the row whose picker is being drawn
	choice := func(s string) string { // THE APP'S PICKER, [←] value [→], blinking as pressed (D-147)
		return pickerCellW(s, chips, d.menuFlashFor(at), menuChoiceW)
	}
	line := func(left, right string) string {
		return render.PadTo(left, overlayMenuW-2-render.Width(right)) + right
	}
	heading := ""
	for i := 0; i < len(rows); i++ {
		r := rows[i]
		section := "MAP DETAIL"
		if r.weather {
			section = "OVERLAYS"
		}
		if section != heading {
			if heading != "" {
				content = append(content, "")
			}
			content, heading = append(content, " "+head(section)), section
		}
		mark := o.ListMark(i == d.mapPane.menuAt)
		at = i
		switch r.kind {
		case menuRadio:
			face := r.label
			if r.key == TemperatureLayer {
				pick := "Actual"
				if d.pickFeels() {
					pick = "Feels like"
				}
				content = append(content, line(" "+mark+radioMark(d.tintChosen(r.key), o.ASCII)+" "+face, choice(pick)))
				continue
			}
			content = append(content, " "+mark+radioMark(d.tintChosen(r.key), o.ASCII)+" "+face)
		case menuGroup:
			if i > 0 && rows[i-1].kind != menuGroup {
				content = append(content, "")
			}
			state := "Enabled"
			if !d.rowGroupOn(r.key) {
				state = "Disabled"
			}
			content = append(content, line(" "+mark+r.label, choice(state)))
		case menuFire:
			mode := strings.ToUpper(d.fireMode()[:1]) + d.fireMode()[1:]
			content = append(content, d.dimmed(r, line(" "+mark+checkMark(o, d.ticked(r.key))+" "+r.label, choice(mode))))
		case menuPreset:
			content = append(content, line(" "+mark+"Preset:", choice(d.detailLevelShown())))
		default: // two to a line
			cell := func(r overlayRow, at int) string {
				on := d.detailOn(r.key)
				if r.weather {
					on = d.ticked(r.key)
				}
				return d.dimmed(r, render.PadTo(o.ListMark(at == d.mapPane.menuAt)+checkMark(o, on)+" "+r.label, cellW))
			}
			row := " " + cell(r, i)
			if i+1 < len(rows) && rows[i+1].kind == r.kind && rows[i+1].group == r.group {
				row += " " + cell(rows[i+1], i+1)
				i++
			}
			content = append(content, row)
		}
	}
	var shows []string // what a switch says of when the style first draws it, so a switch on but not yet seen does not read as broken (U1-42)
	for _, l := range mapDetailLayers() {
		if l.shows != "" {
			shows = append(shows, l.label+" from "+strings.TrimSuffix(l.shows, " zoom")+" zoom")
		}
	}
	if len(shows) > 0 {
		content = append(content, "") // apart from the switches, for scanning (D-147)
		for _, l := range render.WrapText(strings.Join(shows, ", ")+".", overlayMenuW-4) {
			content = append(content, "   "+render.Tint(l, render.Tok(render.TableMuted)))
		}
	}
	content = append(content, "", " "+o.KeyCap("↑↓")+" move "+o.KeyCap("←→")+" choose "+o.KeyCap("space")+" switch")
	return boxed(render.Tint("MAP DETAILS / OVERLAYS", render.Tok(render.ModalTitle)), content, overlayMenuW)
}

// dimmed is a box of a disabled group, dimmed: its tick kept (D-143).
func (d Dashboard) dimmed(r overlayRow, s string) string {
	if r.group == "" || d.rowGroupOn(r.group) {
		return s
	}
	return render.Tint(render.Plain(s), render.Tok(render.TableMuted))
}

// rowGroupOn is a group's switch: the alert layer's own for Alert Areas.
func (d Dashboard) rowGroupOn(group string) bool {
	if group == AlertLayer {
		return d.ticked(AlertLayer)
	}
	return d.groupOn(group)
}

// tintChosen reports whether a radio row is the tint chosen: Temperature's
// is either of its measures (D-119).
func (d Dashboard) tintChosen(key string) bool {
	if key == TemperatureLayer {
		return d.layerOn(TemperatureLayer) || d.layerOn(FeelsLayer)
	}
	return d.layerOn(key)
}

// pickFeels reports whether Temperature's row reads Feels like: the measure
// drawn, or the one last picked while none is.
func (d Dashboard) pickFeels() bool {
	switch {
	case d.layerOn(FeelsLayer):
		return true
	case d.layerOn(TemperatureLayer):
		return false
	}
	on, _ := choiceOf(d.mapLayerChoice, pickFeelsKey)
	return on
}

// menuChoiceW is the menu's pickers' value width: its longest, "Feels like".
const menuChoiceW = 10

// menuFlashFor is the blink of a menu row's picker: only the row pressed,
// only while its window is open - Settings' pickers' rule (D-147).
func (d Dashboard) menuFlashFor(row int) pickerFlash {
	if row != d.mapPane.menuFlashAt || d.mapPane.menuFlash == flashNone || !time.Now().Before(d.mapPane.menuFlashEnd) {
		return flashNone
	}
	return d.mapPane.menuFlash
}

// hasChoice reports whether a row has a picker: ←→ step it.
func hasChoice(r overlayRow) bool {
	return r.kind == menuGroup || r.kind == menuFire || r.kind == menuPreset || (r.kind == menuRadio && r.key == TemperatureLayer)
}

// pickFeelsKey keeps Temperature's measure while no tint is chosen.
const pickFeelsKey = "pick:feels"

// withOverlays lays the menu over the map's upper left while it is open.
func (d Dashboard) withOverlays(lines []string) []string {
	if !d.mapPane.menuOn || len(lines) == 0 {
		return lines
	}
	return spliceBox(lines, d.overlaysBox(), 0, insetCols)
}

// handleOverlaysKey is the menu's keys while it is open: it owns ↑↓, ←→,
// space, esc and its own key (modal control priority, D-61). ←→ change a
// row's choice; space switches a row.
func (d Dashboard) handleOverlaysKey(key string) (Dashboard, bool) {
	rows := d.overlayRows()
	if len(rows) == 0 {
		return d, key == "esc"
	}
	r := rows[d.mapPane.menuAt%len(rows)]
	switch key {
	case "up":
		d.mapPane.menuAt = (d.mapPane.menuAt + len(rows) - 1) % len(rows)
	case "down":
		d.mapPane.menuAt = (d.mapPane.menuAt + 1) % len(rows)
	case "left", "right":
		if hasChoice(r) { // the chip pressed blinks, as Settings' pickers do (D-147)
			d.mapPane.menuFlash, d.mapPane.menuFlashAt, d.mapPane.menuFlashEnd = flashLeft, d.mapPane.menuAt, time.Now().Add(pickerFlashDur)
			if key == "right" {
				d.mapPane.menuFlash = flashRight
			}
		}
		d = d.chooseOnRow(r, key == "right")
	case "space", "enter":
		d = d.switchRow(r)
	case "esc":
		d.mapPane.menuOn = false
	default:
		return d, false
	}
	d.mapPane.gen++ // the menu is drawn with the window: a switch or a move redraws it
	return d, true
}

// withChoice is the layer choices with one set, as their word.
func (d Dashboard) withChoice(set func(choice map[string]bool)) Dashboard {
	choice := choicesOf(d.mapLayerChoice)
	if choice == nil {
		choice = map[string]bool{}
	}
	set(choice)
	d.mapLayerChoice = layerChoiceKey(choice)
	d.setup.uiDirty = true // written as Settings writes it (uiApplyCmd)
	return d.refreshMapCost().requestFeed()
}

// switchRow is space on a row.
func (d Dashboard) switchRow(r overlayRow) Dashboard {
	switch r.kind {
	case menuRadio:
		key := r.key
		if key == TemperatureLayer && d.pickFeels() {
			key = FeelsLayer
		}
		chosen := d.tintChosen(r.key)
		d.mapPane.tempAuto = false // the listener's switch from here on (D-104)
		return d.withChoice(func(c map[string]bool) {
			for _, t := range oneTint {
				c[t] = false // one tint at a time (D-119, D-137, D-139); space on the chosen clears it (D-142)
			}
			if !chosen {
				c[key] = true
			}
		})
	case menuGroup:
		on := d.rowGroupOn(r.key)
		return d.withChoice(func(c map[string]bool) { c[r.key] = !on })
	case menuCheck, menuFire:
		on := d.ticked(r.key)
		return d.withChoice(func(c map[string]bool) { c[r.key] = !on })
	case menuPreset:
		d.setup.uiDirty = true
		return d.cycleDetailLevel(true)
	}
	d.setup.uiDirty = true
	return d.setDetail(r.key, !d.detailOn(r.key))
}

// chooseOnRow is ←→ on a row with a choice; on any other row, nothing - the
// open menu owns the keys.
func (d Dashboard) chooseOnRow(r overlayRow, forward bool) Dashboard {
	switch r.kind {
	case menuRadio:
		if r.key != TemperatureLayer {
			return d
		}
		feels, chosen := !d.pickFeels(), d.tintChosen(TemperatureLayer) // Actual and Feels like: two, either way is the other
		return d.withChoice(func(c map[string]bool) {
			c[pickFeelsKey] = feels
			if chosen {
				c[TemperatureLayer], c[FeelsLayer] = !feels, feels // the tint drawn follows the measure (D-119)
			}
		})
	case menuGroup:
		return d.switchRow(r)
	case menuFire:
		at := slices.Index(fireModes, d.fireMode())
		step := 1
		if !forward {
			step = len(fireModes) - 1
		}
		next := fireModes[(at+step)%len(fireModes)]
		return d.withChoice(func(c map[string]bool) {
			for _, m := range fireModes[1:] {
				c["fire:"+m] = m == next
			}
		})
	case menuPreset:
		d.setup.uiDirty = true
		return d.cycleDetailLevel(forward)
	}
	return d
}

// mapTitleAt names what is in view (D-64): the app's namer for the view's
// centre and width, with the selected place while it is in view. With no
// namer, the selected place.
func (d Dashboard) mapTitleAt(size tuimaps.Size) string {
	loc, m := d.selectedLocation(), d.mapPane.m
	if loc == nil {
		return ""
	}
	if m == nil {
		return loc.Label
	}
	v := d.viewBox(size)
	if !v.Contains(loc.Lat, loc.Lon) {
		return d.viewName() // D-78: away from the place, never its name
	}
	var name string
	if d.cfg.MapAreaName != nil {
		centre, _ := m.Centre()
		widthKm := (v.E - v.W) * 111.32 * math.Cos(centre.Lat*math.Pi/180) // across the view at its centre
		name = d.cfg.MapAreaName(centre, widthKm)
	}
	if name == "" || name == loc.Label {
		return loc.Label
	}
	return name + " " + d.opts().Glyphs().Dot + " " + loc.Label
}

// MapView is the box of the map in view, in degrees (0.18.0 D-66): what
// "Alerts in view" asks the app about, and what the title names.
type MapView = geo.Box

// viewBox is the map's view at a size, from the library's centre and zoom
// and its published scale (256-dot tiles, a braille cell two dots wide and
// four high); nothing when no map is built.
func (d Dashboard) viewBox(size tuimaps.Size) MapView {
	m := d.mapPane.m
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
