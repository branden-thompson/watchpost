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
	cap := func(act term.Action, face string) string {
		if d.mapPane.flash == act && time.Now().Before(d.mapPane.flashEnd) {
			return o.KeyCapInverted(face)
		}
		return o.KeyCap(face)
	}
	return boxed("Controls", []string{
		"    " + cap(actMapPanUp, "↑") + "     " + cap(actMapZoomIn, "+"),
		" " + cap(actMapPanLeft, "←") + cap(actMapPanDown, "↓") + cap(actMapPanRight, "→") + "  " + cap(actMapZoomOut, "-"),
		" " + cap(actMapPrev, "[") + cap(actMapNext, "]") + " place",
		" " + regionCaps(o, slices.Contains(mapRegionActs, d.mapPane.flash) && time.Now().Before(d.mapPane.flashEnd)) + " region", // D-77: the region keys, prominently
	}, controlsInner)
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
	if size.Rows < 14 || size.Cols < 60 || len(lines) < size.Rows {
		return lines
	}
	box := d.controlsBox()
	// ABOVE THE LAST ROW: the library writes the scale and the credit there,
	// and the credit is the attribution, which is never covered (FR-14,
	// UAT-1 U1-26).
	return spliceBox(lines, box, size.Rows-1-len(box), insetCols+size.Cols-(controlsInner+2))
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
	return []tuimaps.Detail{tuimaps.DetailEssential, tuimaps.DetailWeather, tuimaps.DetailStandard, tuimaps.DetailFull}
}

// detailLevelByKey reads the file's word; anything else is Weather, the
// default (D-67).
func detailLevelByKey(key string) tuimaps.Detail {
	for _, l := range mapDetailLevels() {
		if l.String() == key {
			return l
		}
	}
	return tuimaps.DetailWeather
}

// detailLevelLabel is a level's words.
func detailLevelLabel(l tuimaps.Detail) string {
	s := l.String()
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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

// overlayRow is one row of the Overlays menu.
type overlayRow struct {
	key, label string
	weather    bool // a registry layer; otherwise the map's own detail
}

// overlayRows are the menu's rows: the weather layers the app registered,
// then the map's detail.
func (d Dashboard) overlayRows() []overlayRow {
	var out []overlayRow
	for _, l := range d.cfg.MapLayers {
		out = append(out, overlayRow{key: l.Key, label: l.Label, weather: true})
		if l.Key == AlertLayer {
			for _, c := range AlertCategories() { // D-80: [w]'s categories, under the alert areas
				out = append(out, overlayRow{key: categoryChoice(c.Key), label: "  " + c.Label, weather: true})
			}
		}
	}
	out = append(out, overlayRow{key: detailLevelKey, label: "Detail level"})
	for _, l := range mapDetailLayers() {
		out = append(out, overlayRow{key: l.key, label: l.label})
	}
	return out
}

// detailLevelKey is the Overlays menu's detail-level row.
const detailLevelKey = "level"

// overlaysBox is the menu (D-65), the row under the cursor marked.
func (d Dashboard) overlaysBox() []string {
	o := d.opts()
	var content []string
	heading := ""
	for i, r := range d.overlayRows() {
		group := "MAP DETAIL"
		on := d.detailOn(r.key)
		if r.weather {
			group, on = "WEATHER", d.layerOn(r.key)
		}
		if group != heading {
			content, heading = append(content, " "+group), group
		}
		if r.key == detailLevelKey && !r.weather {
			content = append(content, " "+o.ListMark(i == d.mapPane.menuAt)+"Detail: "+d.detailLevelShown()+"  (space: next)")
			continue
		}
		hint := ""
		for _, l := range mapDetailLayers() {
			if !r.weather && l.key == r.key {
				hint = detailShows(l)
			}
		}
		content = append(content, " "+o.ListMark(i == d.mapPane.menuAt)+checkMark(o, on)+" "+r.label+hint)
	}
	content = append(content, " "+o.KeyCap("↑↓")+" move "+o.KeyCap("space")+" switch")
	return boxed("Overlays", content, 40)
}

// withOverlays lays the menu over the map's upper left while it is open.
func (d Dashboard) withOverlays(lines []string) []string {
	if !d.mapPane.menuOn || len(lines) == 0 {
		return lines
	}
	return spliceBox(lines, d.overlaysBox(), 0, insetCols)
}

// handleOverlaysKey is the menu's keys while it is open: it owns ↑↓, space,
// esc and its own key (modal control priority, D-61).
func (d Dashboard) handleOverlaysKey(key string) (Dashboard, bool) {
	rows := d.overlayRows()
	switch key {
	case "up":
		d.mapPane.menuAt = (d.mapPane.menuAt + len(rows) - 1) % max(len(rows), 1)
	case "down":
		d.mapPane.menuAt = (d.mapPane.menuAt + 1) % max(len(rows), 1)
	case "space", "enter":
		if len(rows) == 0 {
			return d, true
		}
		r := rows[d.mapPane.menuAt%len(rows)]
		if r.weather {
			choice := choicesOf(d.mapLayerChoice)
			if choice == nil {
				choice = map[string]bool{}
			}
			choice[r.key] = !d.layerOn(r.key)
			d.mapLayerChoice = layerChoiceKey(choice)
			d = d.refreshMapCost().requestFeed()
		} else if r.key == detailLevelKey {
			d = d.cycleDetailLevel(true)
		} else {
			d = d.setDetail(r.key, !d.detailOn(r.key))
		}
		d.setup.uiDirty = true // written as Settings writes it (uiApplyCmd)
	case "esc":
		d.mapPane.menuOn = false
	default:
		return d, false
	}
	d.mapPane.gen++ // the menu is drawn with the window: a switch or a move redraws it
	return d, true
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
