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
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

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

// areaAlertsBox is the description in a box (D-63): the disclosure on the
// session's first open, then the words. Longer than two thirds of the map,
// it ends with a line saying where the whole text is.
func (d Dashboard) areaAlertsBox(size tuimaps.Size) []string {
	inner := areaAlertsInner(size.Cols)
	var content []string
	if d.mapPane.disclose && d.cfg.MapDisclosure != "" {
		content = append(render.WrapText(d.cfg.MapDisclosure, inner-2), "")
	}
	for _, l := range d.describeLines() {
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

// insetCols is the blank cell each drawn line starts with (insetLines); the
// window's own padding is outside the lines.
const insetCols = 1

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
	}, controlsInner)
}

// withControls lays the controls over the map's lower right, where there is
// room for them beside the legend: a map under 14 rows or 60 columns draws
// none (its keys are in Help and on the status line).
func (d Dashboard) withControls(lines []string, size tuimaps.Size) []string {
	if size.Rows < 14 || size.Cols < 60 || len(lines) < size.Rows {
		return lines
	}
	box := d.controlsBox()
	return spliceBox(lines, box, size.Rows-len(box), insetCols+size.Cols-(controlsInner+2))
}

// flashMapKey marks a map key pressed, for the controls box to blink.
func (d Dashboard) flashMapKey(act term.Action) Dashboard {
	d.mapPane.flash, d.mapPane.flashEnd = act, time.Now().Add(pickerFlashDur)
	return d
}

// mapDetailLayer is one of the library's basemap layers the listener can
// switch (D-65): its key in the file, its words, the library's layer, and
// Watchpost's default - weather first.
type mapDetailLayer struct {
	key, label string
	layer      tuimaps.Layer
	on         bool
}

// mapDetailLayers is the map's own detail, in the menu's order (D-65: roads,
// rail and parks and reserves off by default).
func mapDetailLayers() []mapDetailLayer {
	return []mapDetailLayer{
		{"borders", "Borders", tuimaps.BorderLayer, true},
		{"water", "Water", tuimaps.WaterLayer, true},
		{"rivers", "Rivers", tuimaps.RiverLayer, true},
		{"names", "Place names", tuimaps.LabelLayer, true},
		{"roads", "Roads", tuimaps.RoadLayer, false},
		{"rail", "Rail", tuimaps.RailLayer, false},
		{"parks", "Parks and reserves", tuimaps.ParkLayer, false},
	}
}

// detailOn reports whether a detail layer is drawn: the listener's choice,
// or Watchpost's default.
func (d Dashboard) detailOn(key string) bool {
	if on, ok := choiceOf(d.mapDetailChoice, key); ok {
		return on
	}
	for _, l := range mapDetailLayers() {
		if l.key == key {
			return l.on
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
	}
	for _, l := range mapDetailLayers() {
		out = append(out, overlayRow{key: l.key, label: l.label})
	}
	return out
}

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
		content = append(content, " "+o.ListMark(i == d.mapPane.menuAt)+checkMark(o, on)+" "+r.label)
	}
	content = append(content, " "+o.KeyCap("↑↓")+" move "+o.KeyCap("space")+" switch")
	return boxed("Overlays", content, 30)
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
	if d.cfg.MapAreaName == nil || m == nil {
		return loc.Label
	}
	centre, zoom := m.Centre()
	world := 256 * math.Exp2(zoom)                                 // dots round the world at this zoom
	lonSpan := float64(size.Cols*2) / world * 360                  // degrees across the view
	ySpan := float64(size.Rows*4) / world                          // the view's height, in the world's Mercator units
	widthKm := lonSpan * 111.32 * math.Cos(centre.Lat*math.Pi/180) // across the view at its centre
	name := d.cfg.MapAreaName(centre, widthKm)
	inView := math.Abs(loc.Lon-centre.Lon) <= lonSpan/2 && math.Abs(mercatorY(loc.Lat)-mercatorY(centre.Lat)) <= ySpan/2
	switch {
	case name == "":
		return loc.Label
	case !inView, name == loc.Label:
		return name
	}
	return name + " " + d.opts().Glyphs().Dot + " " + loc.Label
}
