package tty

// map_pane.go — the map window's pane (0.18.0 W1.1, W1.3, W2.1, W2.2).
//
// THE MAP IS DRAWN IN UPDATE, AND VIEW ONLY PRINTS (D-41). Every library call
// but Work is made on the Bubble Tea goroutine, from a message handler, and
// what it drew is stored on the Dashboard with the counters it was drawn at
// (D-45): View reads the stored lines and calls nothing. Work is the one call
// the library lets any goroutine make, so it runs as a command, and the
// message it returns is where what it landed gets drawn.

import (
	"math"
	"reflect"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// actMap opens and closes the map window (FR-1.1).
const actMap term.Action = "map.toggle"

// asciiMapText is what --ascii shows in place of the picture (FR-1.7, FR-1.8):
// the picture is braille, which --ascii never prints, and the remedy.
const asciiMapText = "The map is drawn in braille, which --ascii mode does not print. Run Watchpost without --ascii to see it."

// The window's last line says what the picture is while it is not whole
// (FR-3.4, W1.15's loading indicator). A whole picture leaves it blank.
const (
	mapLoadingText = "Loading map detail…"
	mapOfflineText = "Offline: the basemap could not be fetched, so this is the coarser picture the map already holds."
	mapCoarseText  = "This is a coarser picture: the map has no finer tiles for this view."
)

// noSelectionText is what the window says when nothing is selected (FR-1.3):
// before the first snapshot, or with the selection out of range.
const noSelectionText = "No location is selected. Choose one from the Watchlist or Recent, and the map opens on it."

// mapWorkLimit bounds one Work command, so a stuck fetch cannot hold the
// command's goroutine for good.
const mapWorkLimit = 30 * time.Second

// mapPane is what the Dashboard holds of the map: the library's map, the
// lines last drawn, and the counters they were drawn at (FR-8.4, D-45).
type mapPane struct {
	m         *tuimaps.Map
	lines     []string
	changed   uint64
	ticks     uint64
	region    geo.Region                 // the region the map is held inside (FR-2.1)
	outside   string                     // the place that is in no region, when it is not (FR-2.5)
	status    tuimaps.Status             // the last frame's: whole, or still sharpening
	pending   bool                       // work was waiting when it was drawn
	offline   bool                       // a tile failed since the picture was last whole
	gen       uint64                     // raised by every draw: the window's memo keys on it, so a frame drawn after a landing is never replayed over (F-30)
	failed    string                     // why the map could not be built or drawn, said in the window
	calls     *[]string                  // tests only: the library calls made, by name, in order
	where     *[]callSite                // tests only: each call and the goroutine it ran on (W2.2)
	workers   *mapWorkers                // the commands running for this map, joined on close (W2.6)
	tickAt    time.Time                  // the tick outstanding, at the library's NextCall (W2.2)
	alertsOn  bool                       // the Area Alerts box is open; every open opens it (D-63)
	menuOn    bool                       // the Overlays menu is open (D-65)
	menuAt    int                        // the menu's cursor
	flash     term.Action                // the map key last pressed, blinking in the controls (U1-11)
	flashEnd  time.Time                  // when its blink ends
	title     string                     // what is in view, named at the last draw (D-64)
	viewGen   uint64                     // raised by every move; the settle tick of the newest asks the feed (D-66)
	views     *[]mapView                 // tests only: every view drawn, for M2's instrument
	shown     map[string]bool            // the overlays the feed set, so a gone alert is taken off
	given     map[string]tuimaps.Overlay // what was last handed to the map, by id: an unchanged overlay is not handed in again (U1-28)
	notes     []string                   // the feed's notes, printed under the map
	inMissing map[string]bool            // the feed's alerts whose missing zones hold the place
	national  []snapshot.Alert           // the feed's alerts the station does not hold, which the description names
	report    tuimaps.PlaceReport        // the library's answers for the selected place, as last drawn
	legend    []tuimaps.LegendEntry      // what the map draws now, for the legend (W1.17)
	legendOn  bool                       // the legend is open over the map (D-44)
	drawnSev  map[string]bool            // the severities the feed drew, by word: the legend keys these (D-54, "as drawn")
	feedGen   uint64                     // the feed last asked for; an older answer is dropped
}

// mapView is one drawn frame's view: where, how close, and how big.
type mapView struct {
	centre tuimaps.LonLat
	zoom   float64
	size   tuimaps.Size
}

// MapFeed is what the map draws from the station's data (0.18.0 W5): an
// overlay per alert, and the notes the window prints under the map.
type MapFeed struct {
	Overlays  []tuimaps.Overlay
	Notes     []string
	InMissing map[string]bool // alerts whose missing zones hold the selected place, by alert id
	// National are the alerts drawn that the station does not hold - the
	// national scope's (W5.3) - so the description can name them in full.
	National []snapshot.Alert
}

// mapFeedMsg is the feed's answer, to the request it was asked in.
type mapFeedMsg struct {
	gen  uint64
	feed MapFeed
}

// mapWorkedMsg is one Work command's outcome.
type mapWorkedMsg struct {
	did bool
}

// call makes one library call, recording its name for the tests that check
// where calls are made (W2.2).
func (p mapPane) call(name string, f func()) {
	if p.calls != nil {
		*p.calls = append(*p.calls, name)
	}
	if p.where != nil {
		*p.where = append(*p.where, callSite{name: name, goroutine: goroutineID()})
	}
	f()
}

// toggleMap opens the map window, or closes it when it is open.
func (d Dashboard) toggleMap() Dashboard {
	if d.modal == modalMap {
		return d.close()
	}
	d = d.open(modalMap)
	loc := d.selectedLocation()
	if loc == nil || d.mapsOff {
		return d // FR-1.3, FR-1.6: the window says so; nothing is built for it
	}
	if d.mapPane.m == nil {
		if d.cfg.NewMap == nil {
			d.mapPane.failed = "The map is not available in this build."
			return d
		}
		m, err := d.cfg.NewMap(d.mapBodySize())
		if err != nil {
			d.mapPane.failed = "The map could not be started: " + err.Error()
			return d
		}
		d.mapPane.m, d.mapPane.failed, d.mapPane.workers = m, "", newMapWorkers()
	}
	d.mapPane.alertsOn, d.mapPane.menuOn = d.mapDesc == mapDescWith, false // D-63: the Area Alerts box opens with the map; with the description off it waits for A
	m, scale, km := d.mapPane.m, d.mapScale.zoom(), float64(d.mapNearbyKm)
	d.mapPane.call("Zoom", func() { _ = m.Zoom(scale) })        // every open is at the chosen scale (W4.3); the bound holds it
	d.mapPane.call("SetNearby", func() { _ = m.SetNearby(km) }) // the description's "near" as chosen (W9.2)
	d = d.applyDetail().refreshMapCost().followSelection().requestFeed().renderMap()
	return d.withCmd(tea.Batch(d.mapWorkCmd(), d.mapFeedCmd()))
}

// boundMap holds the map inside its region (W4, W9.1: the library's bound
// replaces a host clamp). The least zoom is the one at which the window's
// view fits inside the region on both axes, so no frame is wider than the
// region either way; it depends on the window's size, so every resize sets
// it again. It is worked out in the library's published scale - 256-dot
// tiles, a braille cell two dots wide and four high.
func (d Dashboard) boundMap() Dashboard {
	m, r := d.mapPane.m, d.mapPane.region
	if m == nil || r.Name == "" {
		return d
	}
	size := d.mapBodySize()
	width := r.E - r.W
	if width < 0 {
		width += 360
	}
	fitX := math.Log2(float64(size.Cols*2) * 360 / (width * 256))
	fitY := math.Log2(float64(size.Rows*4) / ((mercatorY(r.S) - mercatorY(r.N)) * 256))
	least := min(max(fitX, fitY, 0), tuimaps.MaxZoom)
	d.mapPane.call("SetBound", func() {
		_ = m.SetBound(tuimaps.Bound{MinZoom: least, W: r.W, S: r.S, E: r.E, N: r.N})
	})
	return d
}

// mercatorY is a latitude's place down the world, from 0 at the top to 1.
func mercatorY(lat float64) float64 {
	rad := lat * math.Pi / 180
	return (1 - math.Log(math.Tan(rad)+1/math.Cos(rad))/math.Pi) / 2
}

// mapBodySize is the map's size in cells: the window's body, which the
// window's frame and wrapping leave as they are.
func (d Dashboard) mapBodySize() tuimaps.Size {
	cols := d.mapCols()                                                    // border to border (U1-27)
	avail := d.modalMax() - mapStatusRows - len(d.noteLines(d.mapTextW())) // under the map its notes, then the status and the chips; the description is a box over it (D-63)
	return tuimaps.Size{Cols: cols, Rows: max(min(avail, d.modalMax()), 1)}
}

// mapFits reports whether the map can be drawn at the floor (FR-1.4).
func (d Dashboard) mapFits() bool {
	s := d.mapBodySize()
	return s.Cols >= mapMinBody.Cols && s.Rows >= mapMinBody.Rows
}

// descBlock is the description as the window lays it above the map: wrapped
// to the body's width, and a blank line under it. Empty when its mode is off.
func (d Dashboard) descBlock(width int) []string {
	if d.mapDesc == mapDescOff && !d.cfg.ASCII {
		return nil
	}
	var out []string
	for _, l := range d.describeLines() {
		out = append(out, render.WrapText(l, width)...)
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	return out
}

// renderMap draws the map into the pane. It is called from Update only.
func (d Dashboard) renderMap() Dashboard {
	m := d.mapPane.m
	if m == nil || d.mapPane.outside != "" {
		return d // FR-2.5: nothing wider is drawn in its place
	}
	var frame tuimaps.Frame
	var err error
	miles := d.units == render.UnitF
	d.mapPane.call("Units", func() { m.Units(miles, miles) }) // the description's distances and temperatures in the station's units
	d = d.reportPlace()                                       // first: the description's length decides the map's size
	if !d.mapFits() {
		d.mapPane.lines = nil
		d.mapPane.gen++
		return d // FR-1.4: nothing under the floor is drawn; the notice and the description say it
	}
	d.mapPane.call("Render", func() { frame, err = m.Render(d.mapBodySize(), d.now()) })
	if err != nil {
		d.mapPane.failed = "The map could not be drawn: " + err.Error()
		return d
	}
	d.mapPane.lines, d.mapPane.failed = insetLines(frame.Lines), ""
	if d.mapPane.views != nil {
		c, z := m.Centre()
		*d.mapPane.views = append(*d.mapPane.views, mapView{centre: c, zoom: z, size: d.mapBodySize()})
	}
	d.mapPane.status = frame.Status
	d.mapPane.call("Legend", func() { d.mapPane.legend = m.Legend() })
	d.mapPane.call("Pending", func() { d.mapPane.pending = m.Pending() > 0 })
	var warnings []tuimaps.Warning
	d.mapPane.call("Warnings", func() { warnings = m.Warnings() })
	for _, w := range warnings {
		if w.Kind == tuimaps.TileFailed {
			d.mapPane.offline = true // held until the picture is whole again
		}
	}
	if frame.Status == tuimaps.Complete {
		d.mapPane.offline = false
	}
	d.mapPane.title = d.mapTitleAt(d.mapBodySize())
	d.mapPane.gen++
	d.mapPane.changed, d.mapPane.ticks = frame.Changed, frame.FrameTicks
	return d
}

// mapWorkCmd is the next Work command, or nil when the window is closed or
// nothing is pending.
func (d Dashboard) mapWorkCmd() tea.Cmd {
	m := d.mapPane.m
	if m == nil || d.modal != modalMap {
		return nil
	}
	pending := 0
	d.mapPane.call("Pending", func() { pending = m.Pending() })
	if pending == 0 {
		return nil
	}
	pane := d.mapPane
	return func() tea.Msg {
		ctx, done, ok := pane.workers.begin()
		if !ok {
			return mapWorkedMsg{} // the map closed: nothing is touched
		}
		defer done()
		did := false
		pane.call("Work", func() { did, _ = m.Work(ctx) }) // a failed tile is the library's to retry, and its warning says so
		return mapWorkedMsg{did: did}
	}
}

// applyMapWorked draws what a Work command landed, and asks for the next.
func (d Dashboard) applyMapWorked(v mapWorkedMsg) (tea.Model, tea.Cmd) {
	if v.did && d.modal == modalMap {
		d = d.renderMap()
	}
	return d, d.mapWorkCmd()
}

// insetLines indents each drawn line by one cell, as every window's body is.
func insetLines(lines []string) []string {
	return append([]string(nil), lines...) // flush: the map runs border to border (UAT-1 U1-27)
}

// mapCols is the map window's width inside its two borders: the map runs
// border to border (UAT-1 U1-27), where every other window insets its body.
func (d Dashboard) mapCols() int { return max(d.modalWidth()-2, 1) }

// mapTextW is how wide the window's words wrap: one cell inside the map's
// width, for the space every line of words starts with, so none is cut at
// the border.
func (d Dashboard) mapTextW() int { return max(d.mapCols()-1, 1) }

// mapTitle names the place the map is on.
func (d Dashboard) mapTitle() string {
	if d.mapPane.title != "" && d.modal == modalMap && d.mapPane.m != nil {
		return "Map " + d.opts().Glyphs().Dot + " " + d.mapPane.title // D-64: what is in view
	}
	if loc := d.selectedLocation(); loc != nil {
		return "Map " + d.opts().Glyphs().Dot + " " + loc.Label
	}
	return "Map"
}

// mapBodyLines is what the window shows: the map, or a stated reason why not.
func (d Dashboard) mapBodyLines() []string {
	switch {
	case d.selectedLocation() == nil:
		return []string{noSelectionText}
	case d.mapsOff:
		return []string{mapsOffText}
	case d.mapPane.outside != "":
		return []string{d.mapPane.outside + " is outside every region the map covers - the contiguous United States, Alaska, Hawaii, Puerto Rico and the Virgin Islands, Guam and the Northern Marianas, and American Samoa - so no map is drawn for it."}
	case d.mapPane.failed != "":
		return []string{d.mapPane.failed}
	}
	width := d.mapTextW()
	picture := !d.cfg.ASCII && d.mapDesc != mapDescInstead && d.mapFits()
	var out []string // D-69: the map says nothing of what it sends; Settings does, beside the maps row
	switch {
	case d.cfg.ASCII:
		out = []string{asciiMapText, ""} // FR-1.7, FR-1.8: the description in place of the picture
	case d.mapDesc != mapDescInstead && !d.mapFits():
		out = append(render.WrapText(belowFloorText(d.mapBodySize()), width), "") // FR-1.4: the notice carries the description
	}
	if !picture {
		for _, l := range d.descBlock(width) {
			out = append(out, " "+l)
		}
	}
	if !picture {
		for _, l := range d.noteLines(width) {
			out = append(out, " "+l)
		}
		return out
	}
	size := d.mapBodySize()
	out = append(out, d.withLegend(d.withControls(d.withOverlays(d.withAreaAlerts(d.mapPane.lines, size)), size))...)
	for _, l := range d.noteLines(width) {
		out = append(out, " "+l)
	}
	for _, l := range strings.Split(d.mapStatusLine(), "\n") {
		out = append(out, " "+l)
	}
	return out
}

// reportPlace keeps the library's answers for the selected place, which the
// description reads. It needs no Work and no frame.
func (d Dashboard) reportPlace() Dashboard {
	m, loc := d.mapPane.m, d.selectedLocation()
	d.mapPane.report = tuimaps.PlaceReport{}
	if m == nil || loc == nil {
		return d
	}
	var rep tuimaps.Report
	d.mapPane.call("Report", func() {
		rep, _ = m.Report([]tuimaps.Place{{ID: "selected", Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}}})
	})
	if len(rep.Places) > 0 {
		d.mapPane.report = rep.Places[0]
	}
	return d
}

// noteLines are the feed's notes, wrapped to the map's width.
func (d Dashboard) noteLines(width int) []string {
	var out []string
	for _, n := range d.mapPane.notes {
		out = append(out, render.WrapText(n, width)...)
	}
	if w := costWarning(d.mapCost); w != "" {
		out = append(out, render.WrapText(w, width)...) // FR-9.2: said where the cost is seen
	}
	return out
}

// mapStatusLine says what the picture is while it is not whole, and names
// the window's boxes on a line of their own.
func (d Dashboard) mapStatusLine() string {
	status := d.mapStatusText()
	var chips []string
	for _, c := range []struct {
		act  term.Action
		name string
	}{{actMapAlerts, "Area Alerts"}, {actMapOverlays, "Overlays"}, {actMapLegend, "Legend"}} { // D-44, D-63, D-65: each names its key as bound
		if keys := d.mapKeys[c.act].Keys; len(keys) > 0 {
			chips = append(chips, d.opts().KeyCap(keys[0])+" "+c.name)
		}
	}
	width := d.mapTextW()
	// TWO LINES, ALWAYS: the status, then the chips (UAT-1 U1-2 - three chips
	// beside the status cut it even at 133 columns). Both are reserved whether
	// or not the status says anything, so the map's size never depends on the
	// last frame's status.
	return render.TruncateCells(status, width) + "\n" + render.TruncateCells(strings.Join(chips, "  "), width)
}

// mapStatusRows is how many rows the status and the chips take.
const mapStatusRows = 2

// closeMap lets the library's map go. The app calls it when the station
// stops; a closed pane builds a new map on the next open.
func (d Dashboard) closeMap() {
	d.mapPane.workers.close() // cancelled and joined first: nothing is inside the map when it closes (W2.6)
	if m := d.mapPane.m; m != nil {
		d.mapPane.call("Close", func() { m.Close() })
	}
}

// The map window's own actions (D-61). While the window is open it owns the
// keys these are bound to; every other key still reaches the Observer. The
// legend, playback and description-scroll actions join with their tasks
// (W1.17, W8.9a, W1.6), because a key bound to nothing yet is a dead key.
const (
	actMapPanUp      term.Action = "map.pan.up"
	actMapPanDown    term.Action = "map.pan.down"
	actMapPanLeft    term.Action = "map.pan.left"
	actMapPanRight   term.Action = "map.pan.right"
	actMapPrev       term.Action = "map.location.prev"
	actMapNext       term.Action = "map.location.next"
	actMapZoomIn     term.Action = "map.zoom.in"
	actMapZoomOut    term.Action = "map.zoom.out"
	actMapScrollUp   term.Action = "map.scroll.up"
	actMapScrollDown term.Action = "map.scroll.down"
	actMapLegend     term.Action = "map.legend"
)

// mapActions is the map window's actions in the order Help lists them.
var mapActions = []term.Action{actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight, actMapPrev, actMapNext, actMapZoomIn, actMapZoomOut, actMapScrollUp, actMapScrollDown, actMapAlerts, actMapOverlays, actMapLegend}

// defaultMapKeyMap is D-61's bindings for the open map window.
func defaultMapKeyMap() term.KeyMap {
	return term.KeyMap{
		actMapPanUp:      {Keys: []string{"up"}, Help: "Pan North"},
		actMapPanDown:    {Keys: []string{"down"}, Help: "Pan South"},
		actMapPanLeft:    {Keys: []string{"left"}, Help: "Pan West"},
		actMapPanRight:   {Keys: []string{"right"}, Help: "Pan East"},
		actMapPrev:       {Keys: []string{"["}, Help: "Previous Location"},
		actMapNext:       {Keys: []string{"]"}, Help: "Next Location"},
		actMapZoomIn:     {Keys: []string{"+", "="}, Help: "Zoom In"},
		actMapZoomOut:    {Keys: []string{"-"}, Help: "Zoom Out"},
		actMapScrollUp:   {Keys: []string{"pgup"}, Help: "Scroll Up"},
		actMapScrollDown: {Keys: []string{"pgdown"}, Help: "Scroll Down"},
		actMapLegend:     {Keys: []string{"L"}, Help: "Legend"},      // D-44: shift+L, from the [ L ] Legend chip
		actMapAlerts:     {Keys: []string{"A"}, Help: "Area Alerts"}, // D-63: over the upper left, as the legend is the upper right
		actMapOverlays:   {Keys: []string{"O"}, Help: "Overlays"},    // D-65: the weather layers and the map's detail
	}
}

// mapKeysFrom merges the [keys] entries that name the map's actions into its
// own scope, which is separate from the Observer's: its keys are the map's
// only while the window is open.
func mapKeysFrom(overrides term.KeyMap) (term.KeyMap, error) {
	own := term.KeyMap{}
	for act, b := range overrides {
		if _, ok := defaultMapKeyMap()[act]; ok {
			own[act] = b
		}
	}
	keys, _, err := term.Merge(defaultMapKeyMap(), own)
	return keys, err
}

// handleMapKey is the open map window's keyboard (D-61): a key the map binds
// is the map's; any other key goes on to the Observer's handling.
func (d Dashboard) handleMapKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	act, bound := d.mapKeys.Lookup(key.String())
	if d.mapPane.menuOn && act != actMapOverlays { // D-65: the open menu owns its keys
		if nd, ok := d.handleOverlaysKey(key.String()); ok {
			nd = nd.renderMap()
			save := nd.uiApplyCmd()
			nd.setup.uiDirty = false
			return nd, tea.Batch(save, nd.mapWorkCmd(), nd.mapFeedCmd()), true
		}
	}
	if !bound {
		return d, nil, false
	}
	switch act {
	case actMapAlerts: // D-63
		d.mapPane.alertsOn = !d.mapPane.alertsOn
		d.mapPane.gen++
		return d, nil, true
	case actMapOverlays: // D-65
		d.mapPane.menuOn, d.mapPane.menuAt = !d.mapPane.menuOn, 0
		d.mapPane.gen++
		return d, nil, true
	case actMapScrollUp: // D-61: PgUp and PgDn scroll the window's body, the description first
		d.modalScroll = max(d.modalScroll-max(d.modalMax()-1, 1), 0)
		return d, nil, true
	case actMapScrollDown:
		d.modalScroll = min(d.modalScroll+max(d.modalMax()-1, 1), max(len(d.modalLines())-d.modalMax(), 0))
		return d, nil, true
	}
	if d.mapPane.m == nil {
		return d, nil, false
	}
	d = d.flashMapKey(act) // U1-11: the controls' chip blinks
	size, m := d.mapBodySize(), d.mapPane.m
	stepX, stepY := max(size.Cols/4, 1), max(size.Rows/4, 1) // a quarter of the view a press
	switch act {
	case actMapPanUp:
		d.mapPane.call("PanCells", func() { _ = m.PanCells(0, -stepY) })
	case actMapPanDown:
		d.mapPane.call("PanCells", func() { _ = m.PanCells(0, stepY) })
	case actMapPanLeft:
		d.mapPane.call("PanCells", func() { _ = m.PanCells(-stepX, 0) })
	case actMapPanRight:
		d.mapPane.call("PanCells", func() { _ = m.PanCells(stepX, 0) })
	case actMapZoomIn:
		d.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(1) })
	case actMapZoomOut:
		d.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(-1) })
	case actMapLegend:
		d.mapPane.legendOn = !d.mapPane.legendOn
		return d, nil, true
	case actMapPrev:
		d = d.handleNav("nav-up").followSelection().requestFeed() // the notes speak of the place
	case actMapNext:
		d = d.handleNav("nav-down").followSelection().requestFeed()
	}
	d = d.renderMap()
	d, settle := d.viewMoved()
	return d, tea.Batch(d.mapWorkCmd(), d.mapFeedCmd(), settle), true
}

// followSelection puts the map on the selected location (FR-1.2), held
// inside the region that location is in (FR-2.1). A location in no region
// draws nothing wider in its place: the window says so (FR-2.5).
func (d Dashboard) followSelection() Dashboard {
	loc, m := d.selectedLocation(), d.mapPane.m
	if loc == nil || m == nil {
		return d
	}
	region, ok := geo.RegionOf(loc.Lat, loc.Lon)
	if !ok {
		d.mapPane.outside = loc.Label
		return d
	}
	d.mapPane.outside, d.mapPane.region = "", region
	d = d.boundMap()
	d.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}) })
	return d
}

// requestFeed asks for the map's alerts again: on opening, on new data, and
// when the place changes. Only the newest request's answer is drawn.
func (d Dashboard) requestFeed() Dashboard {
	d.mapPane.feedGen++
	return d
}

// mapFeedCmd asks the app for the alerts the map draws, off the UI goroutine
// (the zones they name may be fetched).
func (d Dashboard) mapFeedCmd() tea.Cmd {
	feed, loc := d.cfg.MapFeed, d.selectedLocation()
	if feed == nil || d.mapPane.m == nil || d.modal != modalMap || loc == nil {
		return nil
	}
	gen, ask, place, workers := d.mapPane.feedGen, d.mapAsk(), *loc, d.mapPane.workers
	ask.Place = &place // the command's own copy: the model may move on while it runs
	return func() tea.Msg {
		ctx, done, ok := workers.begin()
		if !ok {
			return nil // the map closed: the app is not asked
		}
		defer done()
		return mapFeedMsg{gen: gen, feed: feed(ctx, ask)}
	}
}

// applyMapFeed sets the feed's overlays, takes off the ones it no longer
// has, keeps its notes, and draws.
func (d Dashboard) applyMapFeed(v mapFeedMsg) (tea.Model, tea.Cmd) {
	m := d.mapPane.m
	if m == nil || v.gen != d.mapPane.feedGen {
		return d, nil // an older request's answer: a newer one is on its way
	}
	v.feed = d.feedForLayers(v.feed) // a layer switched off draws nothing (W1.11)
	d = d.refreshMapCost()
	shown, given := map[string]bool{}, map[string]tuimaps.Overlay{}
	notes := append([]string(nil), v.feed.Notes...)
	for _, o := range v.feed.Overlays {
		// AN UNCHANGED OVERLAY IS NOT HANDED IN AGAIN (UAT-1 U1-28). A Set
		// replaces what the library prepared, and until a Work prepares it
		// again the area is not drawn: every new snapshot re-sent the same
		// alerts, and the area blinked out between frames.
		if prev, ok := d.mapPane.given[o.ID]; ok && reflect.DeepEqual(prev, o) {
			shown[o.ID], given[o.ID] = true, o
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err != nil {
			notes = append(notes, "An alert could not be drawn: "+err.Error())
			continue
		}
		shown[o.ID], given[o.ID] = true, o
	}
	for id := range d.mapPane.shown {
		if !shown[id] {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
		}
	}
	d.mapPane.shown, d.mapPane.given, d.mapPane.notes, d.mapPane.inMissing, d.mapPane.national = shown, given, notes, v.feed.InMissing, v.feed.National
	d.mapPane.drawnSev = map[string]bool{}
	for _, o := range v.feed.Overlays {
		for _, f := range o.Features {
			if w := strings.ToLower(f.Severity.Word()); w != "" {
				d.mapPane.drawnSev[w] = true
			}
		}
	}
	d = d.renderMap()
	return d, d.mapWorkCmd()
}

// mapHelpRow is one row of Help's MAP group: several actions that are one
// idea, listed on one line so the group fits (D-61's ten keys in four rows).
type mapHelpRow struct{ keys, help string }

// mapHelpRows are the MAP group's rows, read from the merged keys so a
// [keys] rebind shows.
func mapHelpRows(keys term.KeyMap) []mapHelpRow {
	join := func(acts ...term.Action) string {
		var all []string
		for _, a := range acts {
			all = append(all, keys[a].Keys...)
		}
		return strings.Join(all, ", ")
	}
	return []mapHelpRow{
		{join(actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight), "Pan"},
		{join(actMapPrev, actMapNext), "Previous / Next Location"},
		{join(actMapZoomIn, actMapZoomOut), "Zoom In / Out"},
		{join(actMapScrollUp, actMapScrollDown), "Scroll"},
		{join(actMapAlerts, actMapOverlays, actMapLegend), "Area Alerts / Overlays / Legend"},
	}
}

// legendWidth is the legend box's width in cells.
const legendWidth = 26

// withLegend lays the legend over the map's top right corner when it is open
// (D-44: a picture-in-picture window over the map). The map's lines keep their
// colours to the left of the box.
func (d Dashboard) withLegend(lines []string) []string {
	if !d.mapPane.legendOn || len(lines) == 0 {
		return lines
	}
	// SPLICED, ONE ROW DOWN: the library writes the stale word and the frame
	// time along the top row's right end, and the splice keeps the map's
	// colours on either side (UAT-1 U1-19).
	return spliceBox(lines, d.legendBox(), 1, max(render.Width(lines[0])-legendWidth, 0))
}

// legendBox is the legend's lines: a key for everything on the map that needs
// one (D-54) - in P1-a the alert severities drawn, each with the digit its
// outline repeats.
func (d Dashboard) legendBox() []string {
	inner := legendWidth - 2
	row := func(s string) string { return "│" + render.PadTo(render.TruncateCells(s, inner), inner) + "│" }
	out := []string{"┌─ Legend " + strings.Repeat("─", max(inner-9, 0)) + "┐"}
	rows := 0
	seen := map[string]bool{}
	for _, e := range d.mapPane.legend {
		for _, c := range e.Classes {
			if e.Preset == "alert" && !d.mapPane.drawnSev[c.Label] {
				continue // only the severities on the map (D-54: contextual)
			}
			key := e.Preset + "/" + c.Label
			if seen[key] {
				continue // two alerts of one severity are one key
			}
			seen[key] = true
			if e.Preset == "alert" {
				out = append(out, row(" "+c.Mark+" "+strings.ToUpper(c.Label)))
			} else {
				out = append(out, row(" "+c.Label))
			}
			rows++
		}
	}
	if rows == 0 {
		out = append(out, row(" nothing keyed"))
	}
	return append(out, "└"+strings.Repeat("─", inner)+"┘")
}

// mapStatusText is what the status line says of the picture while it is not
// whole: loading, offline or coarser; nothing when it is whole (FR-3.4).
func (d Dashboard) mapStatusText() string {
	switch {
	case d.mapPane.offline:
		return mapOfflineText
	case d.mapPane.status == tuimaps.Complete:
		return ""
	case d.mapPane.pending:
		return mapLoadingText
	}
	return mapCoarseText
}

// mapSettleDelay is how long the view must stay still before "Alerts in view"
// asks for the areas it now shows (D-66): never on every key.
const mapSettleDelay = 600 * time.Millisecond

// mapViewSettledMsg is a move's settle tick, to the move it followed.
type mapViewSettledMsg struct{ gen uint64 }

// viewMoved marks the view moved and schedules its settle tick; only the
// newest move's tick asks anything.
func (d Dashboard) viewMoved() (Dashboard, tea.Cmd) {
	d.mapPane.viewGen++
	gen := d.mapPane.viewGen
	return d, tea.Tick(mapSettleDelay, func(time.Time) tea.Msg { return mapViewSettledMsg{gen: gen} })
}

// applyViewSettled asks the feed again once the view has stood still, when
// the alerts drawn are the view's own (D-66); a superseded tick is dropped.
func (d Dashboard) applyViewSettled(v mapViewSettledMsg) (tea.Model, tea.Cmd) {
	if d.modal != modalMap || v.gen != d.mapPane.viewGen || d.mapScope != ScopeInView {
		return d, nil
	}
	d = d.requestFeed()
	return d, d.mapFeedCmd()
}
