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
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/term"
)

// actMap opens and closes the map window (FR-1.1).
const actMap term.Action = "map.toggle"

// asciiMapText is what --ascii shows in place of the picture (FR-1.7, FR-1.8):
// the picture is braille, which --ascii never prints, and the remedy.
const asciiMapText = "The map is drawn in braille, which --ascii mode does not print. Run Watchpost without --ascii to see it."

// noSelectionText is what the window says when nothing is selected (FR-1.3):
// before the first snapshot, or with the selection out of range.
const noSelectionText = "No location is selected. Choose one from the Watchlist or Recent, and the map opens on it."

// mapDefaultZoom is the scale the map opens at: about a state, the region a
// watchlist most often sits in (D-54). The Setting that chooses it is W4.3.
const mapDefaultZoom = 6

// mapWorkLimit bounds one Work command, so a stuck fetch cannot hold the
// command's goroutine for good.
const mapWorkLimit = 30 * time.Second

// mapPane is what the Dashboard holds of the map: the library's map, the
// lines last drawn, and the counters they were drawn at (FR-8.4, D-45).
type mapPane struct {
	m       *tuimaps.Map
	lines   []string
	changed uint64
	ticks   uint64
	gen     uint64    // raised by every draw: the window's memo keys on it, so a frame drawn after a landing is never replayed over (F-30)
	failed  string    // why the map could not be built or drawn, said in the window
	calls   *[]string // tests only: the library calls made, by name, in order
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
	f()
}

// toggleMap opens the map window, or closes it when it is open.
func (d Dashboard) toggleMap() Dashboard {
	if d.modal == modalMap {
		return d.close()
	}
	d = d.open(modalMap)
	loc := d.selectedLocation()
	if loc == nil {
		return d // FR-1.3: the window says so; nothing is built for it
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
		d.mapPane.m, d.mapPane.failed = m, ""
		d.mapPane.call("Zoom", func() { _ = m.Zoom(mapDefaultZoom) })
	}
	d = d.followSelection().renderMap()
	return d.withCmd(d.mapWorkCmd())
}

// mapBodySize is the map's size in cells: the window's body, which the
// window's frame and wrapping leave as they are.
func (d Dashboard) mapBodySize() tuimaps.Size {
	return tuimaps.Size{Cols: max(d.modalWidth()-8, 1), Rows: d.modalMax()} // three clear cells inside each border, as every window's body has
}

// renderMap draws the map into the pane. It is called from Update only.
func (d Dashboard) renderMap() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	var frame tuimaps.Frame
	var err error
	d.mapPane.call("Render", func() { frame, err = m.Render(d.mapBodySize(), d.now()) })
	if err != nil {
		d.mapPane.failed = "The map could not be drawn: " + err.Error()
		return d
	}
	d.mapPane.lines, d.mapPane.failed = insetLines(frame.Lines), ""
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
		ctx, cancel := context.WithTimeout(context.Background(), mapWorkLimit)
		defer cancel()
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
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = " " + l
	}
	return out
}

// mapTitle names the place the map is on.
func (d Dashboard) mapTitle() string {
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
	case d.mapPane.failed != "":
		return []string{d.mapPane.failed}
	case d.cfg.ASCII:
		return []string{asciiMapText}
	}
	return d.mapPane.lines
}

// closeMap lets the library's map go. The app calls it when the station
// stops; a closed pane builds a new map on the next open.
func (d Dashboard) closeMap() {
	if m := d.mapPane.m; m != nil {
		d.mapPane.call("Close", func() { m.Close() })
	}
}

// The map window's own actions (D-61). While the window is open it owns the
// keys these are bound to; every other key still reaches the Observer. The
// legend, playback and description-scroll actions join with their tasks
// (W1.17, W8.9a, W1.6), because a key bound to nothing yet is a dead key.
const (
	actMapPanUp    term.Action = "map.pan.up"
	actMapPanDown  term.Action = "map.pan.down"
	actMapPanLeft  term.Action = "map.pan.left"
	actMapPanRight term.Action = "map.pan.right"
	actMapPrev     term.Action = "map.location.prev"
	actMapNext     term.Action = "map.location.next"
	actMapZoomIn   term.Action = "map.zoom.in"
	actMapZoomOut  term.Action = "map.zoom.out"
)

// mapActions is the map window's actions in the order Help lists them.
var mapActions = []term.Action{actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight, actMapPrev, actMapNext, actMapZoomIn, actMapZoomOut}

// defaultMapKeyMap is D-61's bindings for the open map window.
func defaultMapKeyMap() term.KeyMap {
	return term.KeyMap{
		actMapPanUp:    {Keys: []string{"up"}, Help: "Pan North"},
		actMapPanDown:  {Keys: []string{"down"}, Help: "Pan South"},
		actMapPanLeft:  {Keys: []string{"left"}, Help: "Pan West"},
		actMapPanRight: {Keys: []string{"right"}, Help: "Pan East"},
		actMapPrev:     {Keys: []string{"["}, Help: "Previous Location"},
		actMapNext:     {Keys: []string{"]"}, Help: "Next Location"},
		actMapZoomIn:   {Keys: []string{"+", "="}, Help: "Zoom In"},
		actMapZoomOut:  {Keys: []string{"-"}, Help: "Zoom Out"},
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
	if !bound || d.mapPane.m == nil {
		return d, nil, false
	}
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
	case actMapPrev:
		d = d.handleNav("nav-up").followSelection()
	case actMapNext:
		d = d.handleNav("nav-down").followSelection()
	}
	d = d.renderMap()
	return d, d.mapWorkCmd(), true
}

// followSelection puts the map on the selected location (FR-1.2).
func (d Dashboard) followSelection() Dashboard {
	loc, m := d.selectedLocation(), d.mapPane.m
	if loc == nil || m == nil {
		return d
	}
	d.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}) })
	return d
}
