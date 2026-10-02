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
	"math"
	"reflect"
	"slices"
	"strconv"
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
	region    geo.Region     // the region the map is held inside (FR-2.1)
	outside   string         // the place that is in no region, when it is not (FR-2.5)
	status    tuimaps.Status // the last frame's: whole, or still sharpening
	pending   bool           // work was waiting when it was drawn
	offline   bool           // a tile failed since the picture was last whole
	gen       uint64         // raised by every draw: the window's memo keys on it, so a frame drawn after a landing is never replayed over (F-30)
	failed    string         // why the map could not be built or drawn, said in the window
	calls     *[]string      // tests only: the library calls made, by name, in order
	where     *[]callSite    // tests only: each call and the goroutine it ran on (W2.2)
	workers   *mapWorkers    // the commands running for this map, joined on close (W2.6)
	tickAt    time.Time      // the tick outstanding, at the library's NextCall (W2.2)
	alertsOn  bool           // the Area Alerts box is open; it waits for A (D-87)
	menuOn    bool           // the Overlays menu is open (D-65)
	menuAt    int            // the menu's cursor
	flash     term.Action    // the map key last pressed, blinking in the controls (U1-11)
	edge      geo.Direction  // the edge whose chip is showing (D-81)
	edgeShown bool           // a press held still at the edge: the next the same way crosses
	flashEnd  time.Time      // when its blink ends
	// menuFlash is the Overlays menu's picker chip blinking at row
	// menuFlashAt until menuFlashEnd: Settings' pickers' feedback (D-147).
	menuFlash    pickerFlash
	menuFlashAt  int
	menuFlashEnd time.Time
	title        string                     // what is in view, named at the last draw (D-64)
	viewGen      uint64                     // raised by every move; the settle tick of the newest asks the feed (D-66)
	views        *[]mapView                 // tests only: every view drawn, for M2's instrument
	shown        map[string]bool            // the overlays the feed set, so a gone alert is taken off
	given        map[string]tuimaps.Overlay // what was last handed to the map, by id: an unchanged overlay is not handed in again (U1-28)
	notes        []string                   // the feed's notes, printed under the map
	inMissing    map[string]bool            // the feed's alerts whose missing zones hold the place
	inView       []snapshot.Alert           // the feed's alerts the station does not hold, which the description names

	// The radar (W8): its request generation, the loops handed in by id, the
	// source for the chip and its note, and the loop's line as last drawn.
	radarBusy, radarAgain bool // one request at a time, a later want kept (D-85)
	// feedApplied is the newest feed answer drawn: equal to feedGen, the
	// alerts asked for are in (the timing instrument's "settled").
	feedApplied uint64
	// The feed's one ask in flight (D-157): what it is for, its number, and
	// its cancel; wanted again when new data lands while it runs.
	feedBusy, feedAgain bool
	feedSeq             uint64
	feedFor             feedKey
	feedCtx             context.Context
	feedStop            context.CancelFunc
	// viewAsked is the newest move whose settle tick has asked for its view:
	// below viewGen, a move is still waiting to ask (the instrument's "still").
	viewAsked uint64
	clock     mapClock // the timing instrument's clock (timing.go)
	// radarRegion is the region of the loop being fetched, and radarStop
	// cancels it: leaving the region leaves its answer nowhere to draw (D-130).
	radarRegion                       string
	radarStop                         context.CancelFunc
	radarAt                           time.Time
	radarGiven                        map[string]tuimaps.Overlay
	radarSource, radarNote, radarLine string
	radarAhead                        string                // the source of the loop's hours ahead, "HRRR" (D-113), or none
	radarTimeline                     []string              // the loop's timeline (D-86), drawn in Update
	radarBadgeTime                    string                // the badge's time row (D-92), drawn in Update
	fcTimeline                        []string              // Forecast mode's steps (D-94), drawn in Update
	report                            tuimaps.PlaceReport   // the library's answers for the selected place, as last drawn
	legend                            []tuimaps.LegendEntry // what the map draws now, for the legend (W1.17)
	drawnSev                          map[string]bool       // the severities the feed drew, by word: the legend keys these (D-54, "as drawn")
	feedGen                           uint64                // the feed last asked for; an older answer is dropped
	feed                              *MapFeed              // the feed as last answered, layers applied: set again with each mode's spans (D-98)

	// The temperature (W10): one request at a time as the radar's, the answer
	// held, the grids handed in by id, and the anchor it was built at.
	tempBusy, tempAgain bool
	tempAt, tempAnchor  time.Time
	temp                MapTemperature
	tempGiven           map[string]tuimaps.Overlay
	// Forecast mode (D-94): the step shown, high or low, and its playback.
	fcStep, fcHeld   int
	fcLow, fcPlaying bool
	fcGen            uint64
	// tempAuto is the temperature Forecast mode turned on (D-103), its alone
	// and never saved (D-104); modeChip is the chip that says so, until a key.
	tempAuto, modeChip bool
}

// mapView is one drawn frame's view: where, how close, and how big.
type mapView struct {
	centre tuimaps.LonLat
	zoom   float64
	size   tuimaps.Size
	region geo.Region // the region the frame was bound to (D-28, D-77)
}

// MapFeed is what the map draws from the station's data (0.18.0 W5): an
// overlay per alert, and the notes the window prints under the map.
type MapFeed struct {
	Overlays  []tuimaps.Overlay
	Notes     []string
	InMissing map[string]bool // alerts whose missing zones hold the selected place, by alert id
	// InView are the alerts drawn that the station does not hold - the
	// view's (D-66) - so the description can name them in full.
	InView []snapshot.Alert
	// Times are when each overlay is, by id: the window draws each only while
	// the mode's moment meets it (D-98).
	Times map[string]TimedOverlay
}

// mapFeedMsg is the feed's answer, to the request it was asked in.
type mapFeedMsg struct {
	gen  uint64 // the data asked for (requestFeed's generation)
	seq  uint64 // the ask's number: a cancelled ask's answer carries an older one (D-157)
	feed MapFeed
}

// feedKey is what an alerts answer is for: a place and a view. An answer for
// another is not drawn, and an ask in flight for another is stale (D-157).
//
// THE VIEW IS THE LISTENER'S - a move or a resize, viewGen - NOT ITS BOX. The
// box follows the map's size, which shrinks as notes and the description fill
// in with no move at all; keyed on it, answers would be dropped with nothing
// left to ask again, and a cold open could draw no alerts (W14).
type feedKey struct {
	place snapshot.LocationKey
	view  uint64
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
	d = d.open(modalMap).timeFrom("open")
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
			d.mapPane.failed = mapFailedText // D-124: a path, and the detail to the diagnostics
			d.problem("Map: not started - " + err.Error())
			return d
		}
		d.mapPane.m, d.mapPane.failed, d.mapPane.workers = m, "", newMapWorkers()
	}
	d.mapPane.alertsOn, d.mapPane.menuOn = false, false // D-87: the Area Alerts box waits for A
	m, scale, km := d.mapPane.m, d.mapScale.zoom(), float64(d.mapNearbyKm)
	d.mapPane.call("Zoom", func() { _ = m.Zoom(scale) })        // every open is at the chosen scale (W4.3); the bound holds it
	d.mapPane.call("SetNearby", func() { _ = m.SetNearby(km) }) // the description's "near" as chosen (W9.2)
	d.mapPane.fcStep, d.mapPane.fcPlaying = 0, false            // Forecast mode opens on Now, stopped (D-94)
	d.mapPane.fcGen++
	d.mapPane.tempAuto, d.mapPane.modeChip = false, false
	d = d.ensureMainOverlay() // D-103: a map opened in Forecast mode is never blank
	d = d.applyDetail().applyPlayback().showStep().refreshMapCost().followSelection().requestFeed()
	d.mapPane.viewAsked = d.mapPane.viewGen // the open asks for its view: a move's tick dropped while closed is answered here (W14)
	d = d.renderMap()
	d, feed := d.askFeed()
	d, radar := d.askRadar()
	d, temp := d.askTemp()
	return d.withCmd(tea.Batch(d.mapWorkCmd(), feed, radar, temp))
}

// boundMap holds the map inside its region (W4, W9.1: the library's bound,
// not a host clamp). The least zoom is the one at which the window's
// view fits inside the region on both axes, so no frame is wider than the
// region either way; it depends on the window's size, so every resize sets
// it again. It is worked out in the library's published scale - 256-dot
// tiles, a braille cell two dots wide and four high.
func (d Dashboard) boundMap() Dashboard {
	m, r := d.mapPane.m, d.mapPane.region
	if m == nil || r.Name == "" {
		return d
	}
	least := regionFitZoom(r, d.mapBodySize())
	d.mapPane.call("SetBound", func() {
		_ = m.SetBound(tuimaps.Bound{MinZoom: least, W: r.W, S: r.S, E: r.E, N: r.N})
	})
	return d
}

// regionFitZoom is the least zoom a region is held at: the zoom at which it
// fills the map one way or the other.
func regionFitZoom(r geo.Region, size tuimaps.Size) float64 {
	width := r.E - r.W
	if width < 0 {
		width += 360
	}
	fitX := math.Log2(float64(size.Cols*2) * 360 / (width * 256))
	fitY := math.Log2(float64(size.Rows*4) / ((mercatorY(r.S) - mercatorY(r.N)) * 256))
	return min(max(fitX, fitY, 0), tuimaps.MaxZoom)
}

// mercatorY is a latitude's place down the world, from 0 at the top to 1.
func mercatorY(lat float64) float64 {
	rad := lat * math.Pi / 180
	return (1 - math.Log(math.Tan(rad)+1/math.Cos(rad))/math.Pi) / 2
}

// mapBodySize is the map's size in cells: the window's body, which the
// window's frame and wrapping leave as they are.
func (d Dashboard) mapBodySize() tuimaps.Size {
	cols := d.mapCols()                                                     // border to border (U1-27)
	avail := d.modalMax() - d.mapChromeRows(len(d.noteLines(d.mapTextW()))) // under the map its notes, the status and the chips, and the radar's held rows (D-86, D-89); the description is a box over it (D-63)
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
	for _, l := range d.describeLinesAll() {
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
		d.timeBelowFloor() // the workload's render check: a map not drawn is never a quiet pass (W14)
		return d           // FR-1.4: nothing under the floor is drawn; the notice and the description say it
	}
	d.mapPane.call("Render", func() { frame, err = m.Render(d.mapBodySize(), d.now()) })
	if err != nil {
		d.mapPane.failed = mapFailedText
		d.problem("Map: not drawn - " + err.Error())
		return d
	}
	d.mapPane.lines, d.mapPane.failed = insetLines(frame.Lines), ""
	if d.mapPane.views != nil {
		c, z := m.Centre()
		*d.mapPane.views = append(*d.mapPane.views, mapView{centre: c, zoom: z, size: d.mapBodySize(), region: d.mapPane.region})
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
	d.mapPane.radarLine = d.radarStatus()                 // read from the library here, in Update; the frame only prints it (D-41)
	d.mapPane.radarTimeline = d.radarTimeline(d.scrubW()) // beside the controls (D-103)
	d.mapPane.fcTimeline = d.forecastTimeline(d.scrubW())
	d.mapPane.radarBadgeTime = d.radarBadgeTimeNow()
	d.recordFrame(frame.Status) // D-198: nothing unless the recorder is on
	d.mapPane.gen++
	d.mapPane.changed, d.mapPane.ticks = frame.Changed, frame.FrameTicks
	d = d.timeDraw()
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
	return pane.workers.cmd(context.Background(), mapWorkedMsg{}, func(ctx context.Context) tea.Msg {
		did := false
		pane.call("Work", func() { did, _ = m.Work(ctx) }) // a failed tile is the library's to retry, and its warning says so
		return mapWorkedMsg{did: did}
	})
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
func (d Dashboard) mapCols() int { return max(d.mapWindowCols()-2, 1) }

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
	out = append(out, d.withModeChip(d.withEdgeChip(d.withControls(d.withQuotaNotice(d.withOverlays(d.withAreaAlerts(d.mapPane.lines, size)), size), size), size), size)...) // the notice under every control (D-181)
	if !d.radarTimelineOn() {
		for _, l := range d.noteLines(width) {
			out = append(out, " "+l)
		}
		for _, l := range strings.Split(d.mapStatusLine(), "\n") {
			out = append(out, " "+l)
		}
		return out
	}
	return append(out, d.scrubRows(width)...) // D-103: the rows under the map, in either mode
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

// noteLines are the rows under the map before its status: the badges
// (D-133), then the notes that come and go, each only while it applies
// (D-132), then the cost warning; wrapped to the map's width.
func (d Dashboard) noteLines(width int) []string {
	out := d.badgeRows(width)
	if d.notesYield(width) {
		out = append(out, render.WrapText(notesYieldLine(len(d.mapNotesNow())), width)...) // D-159: the map keeps its floor
	} else {
		for _, n := range d.mapNotesNow() {
			out = append(out, render.WrapText(n, width)...)
		}
	}
	out = append(out, mapCostLine(d.mapCost, width)...) // FR-9.2: said where the cost is seen, in D-82's words laid out as D-89
	return out
}

// mapNotesNow is the map's notes as they stand: the feed's, the radar's
// missing data, what a temperature source lacks. The one list the window
// and the Status window's MAP STATUS both read (D-159), so they cannot
// disagree.
func (d Dashboard) mapNotesNow() []string {
	out := append([]string(nil), d.mapPane.notes...)
	if d.mapPane.radarSource != "" && d.mapPane.radarNote != "" && d.layerOn(RadarLayer) {
		out = append(out, d.mapPane.radarNote) // D-84's missing data; a source that did not answer, with its Setting
	}
	return append(out, d.tempNotes()...) // W10: what a source lacks
}

// notesYieldLine is the notes collapsed to one line (D-159).
func notesYieldLine(n int) string {
	if n == 1 {
		return "1 map note is in the Status window (S)."
	}
	return strconv.Itoa(n) + " map notes are in the Status window (S)."
}

// mapChromeRows are the rows under the map for notes of n lines: the status
// and chips, the notes, and the radar timeline while it is held.
func (d Dashboard) mapChromeRows(noteRows int) int {
	rows := mapStatusRows + noteRows
	if d.radarTimelineOn() {
		rows += radarRows + radarExtraRows
	}
	return rows
}

// notesYield reports whether the notes must collapse to one line (D-159):
// only where even the window's ceiling - the terminal's height minus 8 -
// cannot hold the map at its floor with every note line shown, and one line
// would. Growing the window comes first.
func (d Dashboard) notesYield(width int) bool {
	notes := d.mapNotesNow()
	if len(notes) < 2 {
		return false
	}
	full := len(d.badgeRows(width)) + len(mapCostLine(d.mapCost, width))
	for _, n := range notes {
		full += len(render.WrapText(n, width))
	}
	ceiling := d.height - 8
	return mapMinBody.Rows+d.mapChromeRows(full) > ceiling
}

// mapStatusLine says what the picture is while it is not whole, and names
// the window's boxes on a line of their own.
func (d Dashboard) mapStatusLine() string {
	// D-89's order: the radar's line, the estimate after a slash, then what
	// the picture's status says - W8.8's loop and its age always first.
	lead := d.mapPane.radarLine
	if !d.radarMode() && d.cfg.MapRadar != nil {
		lead = d.forecastStatus() // D-94: Forecast mode's line in the radar's place
	}
	if est := costEstimate(d.mapCost); est != "" {
		lead = strings.TrimPrefix(lead+" / "+est, " / ")
	}
	status := d.mapStatusText()
	if lead != "" {
		status = strings.TrimSuffix(lead+" · "+status, " · ")
	}
	chips := d.mapChips()
	width := d.mapTextW()
	// TWO LINES, ALWAYS: the status, then the chips (UAT-1 U1-2 - three chips
	// beside the status would cut it even at 133 columns). Both are reserved whether
	// or not the status says anything, so the map's size never depends on the
	// last frame's status.
	return render.TruncateCells(status, width) + "\n" + render.TruncateCells(strings.Join(chips, "  "), width)
}

// mapChips are the window's keys as chips, each naming its key as bound:
// Area Alerts, the mode, Overlays (D-63, D-65, D-94; no legend chip,
// D-103).
func (d Dashboard) mapChips() []string {
	var chips []string
	for _, c := range []struct {
		act  term.Action
		name string
	}{{actMapAlerts, "Area Alerts"}, {actMapRadar, d.radarChipWords()}, {actMapOverlays, "Overlays"}} {
		if keys := d.mapKeys[c.act].Keys; len(keys) > 0 {
			chips = append(chips, d.opts().KeyCap(keys[0])+" "+c.name)
		}
	}
	return chips
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
)

// mapRegionActs are the region keys, 1 to 6 (D-77): each snaps the map to
// its region, in geo's numbering.
var mapRegionActs = []term.Action{"map.region.1", "map.region.2", "map.region.3", "map.region.4", "map.region.5", "map.region.6"}

// mapRegionShort are the names Help lists the region keys by.
var mapRegionShort = []string{"US", "Alaska", "Hawaii", "Caribbean", "Samoa", "Guam"}

// mapRegionLabels are the regions' names, for each key's own help, in the
// same order.
var mapRegionLabels = []string{"Continental US", "Alaska", "Hawaii", "US Caribbean", "American Samoa", "Guam & N. Marianas"}

// mapActions is the map window's actions in the order Help lists them.
var mapActions = append([]term.Action{actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight, actMapPrev, actMapNext, actMapZoomIn, actMapZoomOut, actMapScrollUp, actMapScrollDown, actMapAlerts, actMapRadar, actMapOverlays,
	actMapPlay, actMapBack, actMapOn, actMapNewest, actMapHighLow}, mapRegionActs...)

// defaultMapKeyMap is D-61's bindings for the open map window.
func defaultMapKeyMap() term.KeyMap {
	km := term.KeyMap{
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
		actMapAlerts:     {Keys: []string{"A"}, Help: "Area Alerts"},        // D-63: over the upper left, as the legend is the upper right
		actMapOverlays:   {Keys: []string{"O"}, Help: "Overlays"},           // D-65: the weather layers and the map's detail
		actMapRadar:      {Keys: []string{"R"}, Help: "Radar On / Off"},     // D-94: Radar mode, else Forecast mode
		actMapPlay:       {Keys: []string{"space"}, Help: "Play / Stop"},    // D-61: the playback keys, radar's or the forecast's
		actMapBack:       {Keys: []string{"shift+left"}, Help: "Step Back"}, // D-86: at the timeline's ends
		actMapOn:         {Keys: []string{"shift+right"}, Help: "Step On"},
		actMapNewest:     {Keys: []string{"n"}, Help: "Now"},
		actMapHighLow:    {Keys: []string{"<", ">"}, Help: "High / Low"}, // D-97: Forecast mode's days
	}
	for i, act := range mapRegionActs { // D-77: 1 to 6, a region each
		km[act] = term.Binding{Keys: []string{string(rune('1' + i))}, Help: "Region " + string(rune('1'+i)) + ": " + mapRegionLabels[i]}
	}
	return km
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
	if d.mapPane.modeChip { // any key takes the mode's chip away (D-103), as the edge chip's
		d.mapPane.modeChip = false
		d.mapPane.gen++
	}
	if d.mapPane.menuOn && act != actMapOverlays { // D-65: the open menu owns its keys
		if nd, cmd, ok := d.overlaysMenuKey(key.String()); ok {
			return nd, cmd, true
		}
	}
	if !bound {
		return d, nil, false
	}
	d = d.timeFrom(string(act))
	if nd, ok := d.mapWindowKey(act); ok {
		return nd, nil, true
	}
	if d.mapPane.m == nil {
		return d, nil, false
	}
	switch act {
	case actMapRadar: // D-94: Radar mode on and off
		nd, cmd := d.flashMapKey(act).switchMode()
		return nd, cmd, true
	case actMapHighLow: // D-97: the days' high or low
		return d.flashMapKey(act).flipHighLow(), d.mapWorkCmd(), true
	}
	d = d.flashMapKey(act) // U1-11: the controls' chip blinks
	if !d.radarMode() {    // Forecast mode: the host steps (D-94)
		if nd, cmd, ok := d.handleForecastPlayback(act); ok {
			return nd, tea.Batch(cmd, nd.mapWorkCmd()), true
		}
	} else if nd, ok := d.handlePlayback(act); ok {
		return nd, nd.mapWorkCmd(), true
	}
	return d.moveMapView(act)
}

// overlaysMenuKey is a key while the Overlays menu is open; false when the
// menu does not take it.
func (d Dashboard) overlaysMenuKey(key string) (Dashboard, tea.Cmd, bool) {
	nd, ok := d.handleOverlaysKey(key)
	if !ok {
		return d, nil, false
	}
	// AN ARROW MOVES THE CURSOR AND NOTHING ELSE (UAT-2 U2-13, U2-14): only a
	// switch touches the map, so a cursor move asks for no radar or
	// temperature and hands no grid in again.
	if nd.mapLayerChoice == d.mapLayerChoice && nd.mapDetailChoice == d.mapDetailChoice && nd.mapDetailLevel == d.mapDetailLevel {
		return nd, nil, true
	}
	nd = nd.timeFrom("overlay")
	nd, _ = nd.setTemp() // temperature switched: drawn at once from what is held (D-99), or taken off
	var temp tea.Cmd
	if (nd.layerOn(UVLayer) && !d.layerOn(UVLayer)) || (nd.layerOn(AirLayer) && !d.layerOn(AirLayer)) {
		nd, temp = nd.askTemp() // UV and air quality are asked only while on (D-137, D-139)
	}
	nd = nd.renderMap()
	save := nd.uiApplyCmd()
	nd.setup.uiDirty = false
	nd, feed := nd.askFeed()
	return nd, tea.Batch(save, nd.mapWorkCmd(), feed, temp), true // radar is R's, not the menu's (D-94)
}

// mapWindowKey is a key that works on the window itself, with a map or
// without: the alerts, the Overlays menu and the body's scroll.
func (d Dashboard) mapWindowKey(act term.Action) (Dashboard, bool) {
	switch act {
	case actMapAlerts: // D-63
		d.mapPane.alertsOn = !d.mapPane.alertsOn
		d.mapPane.gen++
	case actMapOverlays: // D-65
		d.mapPane.menuOn, d.mapPane.menuAt = !d.mapPane.menuOn, 0
		d.mapPane.gen++
	case actMapScrollUp: // D-61: PgUp and PgDn scroll the window's body, the description first
		d.modalScroll = max(d.modalScroll-max(d.modalMax()-1, 1), 0)
	case actMapScrollDown:
		d.modalScroll = min(d.modalScroll+max(d.modalMax()-1, 1), max(len(d.modalLines())-d.modalMax(), 0))
	default:
		return d, false
	}
	return d, true
}

// moveMapView is a key that moves the view: a pan, a zoom, the next place or
// a region.
func (d Dashboard) moveMapView(act term.Action) (tea.Model, tea.Cmd, bool) {
	size, m := d.mapBodySize(), d.mapPane.m
	stepX, stepY := max(size.Cols/4, 1), max(size.Rows/4, 1) // a quarter of the view a press
	// ANY KEY TAKES THE EDGE'S CHIP AWAY (D-81); a press the same way again
	// is the one that crosses.
	armed := d.mapPane.edgeShown
	d.mapPane.edgeShown = false
	switch act {
	case actMapPanUp:
		d = d.panOrCross(0, -stepY, geo.North, armed)
	case actMapPanDown:
		d = d.panOrCross(0, stepY, geo.South, armed)
	case actMapPanLeft:
		d = d.panOrCross(-stepX, 0, geo.West, armed)
	case actMapPanRight:
		d = d.panOrCross(stepX, 0, geo.East, armed)
	case actMapZoomIn:
		d.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(1) })
	case actMapZoomOut:
		d.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(-1) })
	case actMapPrev:
		d = d.handleNav("nav-up").followSelection().requestFeed() // the notes speak of the place
	case actMapNext:
		d = d.handleNav("nav-down").followSelection().requestFeed()
	default:
		if n := slices.Index(mapRegionActs, act); n >= 0 {
			if r, ok := geo.RegionNumbered(n + 1); ok {
				d = d.showRegion(r)
			}
		}
	}
	d, settle := d.viewMoved() // marked moved before it is drawn: its view is not in until its settle tick asks (W14's instrument)
	d = d.renderMap()
	return d, tea.Batch(d.mapWorkCmd(), settle), true // the alerts are the settle tick's to ask: never on every key (D-66, W14's P-2)
}

// panOrCross pans by cells. Where the region's edge holds the map still, the
// first press shows a chip naming the region beyond (D-81), and a second
// press the same way while it shows moves on to it (D-77). An edge with
// nothing beyond it stays an edge.
func (d Dashboard) panOrCross(dx, dy int, dir geo.Direction, armed bool) Dashboard {
	m := d.mapPane.m
	before, _ := m.Centre()
	d.mapPane.call("PanCells", func() { _ = m.PanCells(dx, dy) })
	after, _ := m.Centre()
	if math.Abs(after.Lat-before.Lat) > 1e-9 || math.Abs(after.Lon-before.Lon) > 1e-9 {
		return d
	}
	if _, ok := geo.Neighbour(d.mapPane.region.Name, dir); !ok {
		return d
	}
	if armed && d.mapPane.edge == dir {
		next, _ := geo.Neighbour(d.mapPane.region.Name, dir)
		return d.showRegion(next)
	}
	d.mapPane.edge, d.mapPane.edgeShown = dir, true
	return d
}

// showRegion binds the map to a region and shows the whole of it (D-77): the
// listener's way to every region the station's APIs cover, one at a time
// (D-28). The selected place is unchanged.
func (d Dashboard) showRegion(r geo.Region) Dashboard {
	m := d.mapPane.m
	d.mapPane.outside, d.mapPane.region = "", r
	d = d.boundMap()
	lat, lon := r.Centre()
	least := regionFitZoom(r, d.mapBodySize())
	d.mapPane.call("Zoom", func() { _ = m.Zoom(least) })
	d.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: lon, Lat: lat}) })
	return d
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

// requestFeed marks newer data wanted: on opening, on new data, when the
// view or the place changes. askFeed asks for it.
func (d Dashboard) requestFeed() Dashboard {
	d.mapPane.feedGen++
	return d
}

// feedKey is the place and view an ask made now would be for.
func (d Dashboard) feedKey() feedKey {
	k := feedKey{view: d.mapPane.viewGen}
	if loc := d.selectedLocation(); loc != nil {
		k.place = snapshot.Key(refOf(*loc))
	}
	return k
}

// askFeed asks the app for the map's alerts - ONE ASK IN FLIGHT (D-157).
// While one runs for this place and view, new data marks the feed wanted
// again, and one fresh ask follows its answer. One made stale - the place or
// the view has changed - is cancelled, and the new one asked at once. Asking
// anew on every snapshot would drop the answer being waited for, so a feed
// slower than the snapshots would never be drawn (W14's C-1).
func (d Dashboard) askFeed() (Dashboard, tea.Cmd) {
	if d.cfg.MapFeed == nil || d.mapPane.m == nil || d.modal != modalMap || d.selectedLocation() == nil {
		return d, nil
	}
	key, p := d.feedKey(), &d.mapPane
	if p.feedBusy && key == p.feedFor {
		p.feedAgain = true
		return d, nil
	}
	if p.feedStop != nil {
		p.feedStop() // stale: its place or view has gone
	}
	p.feedSeq++
	p.feedCtx, p.feedStop = context.WithCancel(context.Background())
	p.feedBusy, p.feedAgain, p.feedFor = true, false, key
	return d, d.mapFeedCmd()
}

// mapFeedCmd is the ask in flight's command, off the UI goroutine (the zones
// the alerts name may be fetched), cancelled with it.
func (d Dashboard) mapFeedCmd() tea.Cmd {
	feed, loc := d.cfg.MapFeed, d.selectedLocation()
	if feed == nil || d.mapPane.m == nil || d.modal != modalMap || loc == nil {
		return nil
	}
	gen, seq, ask, place, workers := d.mapPane.feedGen, d.mapPane.feedSeq, d.mapAsk(), *loc, d.mapPane.workers
	asked := d.mapPane.feedCtx
	if asked == nil {
		asked = context.Background()
	}
	ask.Place = &place                                                 // the command's own copy: the model may move on while it runs
	return workers.cmd(asked, nil, func(ctx context.Context) tea.Msg { // a stale ask stops it too
		return mapFeedMsg{gen: gen, seq: seq, feed: feed(ctx, ask)}
	})
}

// applyMapFeed sets the feed's overlays, takes off the ones it no longer
// has, keeps its notes, and draws.
func (d Dashboard) applyMapFeed(v mapFeedMsg) (tea.Model, tea.Cmd) {
	m := d.mapPane.m
	p := &d.mapPane
	if m == nil || v.seq != p.feedSeq {
		return d, nil // a cancelled ask's answer: the ask in flight is newer
	}
	p.feedBusy = false
	if p.feedStop != nil {
		p.feedStop() // its context let go
		p.feedStop, p.feedCtx = nil, nil
	}
	if d.feedKey() != p.feedFor { // the view moved since it was asked: its settle tick asks for the new one
		return d.feedAgainCmd(nil)
	}
	v.feed = d.feedForLayers(v.feed) // a layer switched off draws nothing (W1.11)
	d.mapPane.feedApplied = v.gen
	d = d.timed("answered:feed")
	d = d.refreshMapCost()
	d.mapPane.feed = &v.feed
	d = d.setFeed(v.feed)
	return d.feedAgainCmd(d.mapWorkCmd())
}

// feedAgainCmd is after an answer: the one fresh ask new data wanted while
// it ran, if any (D-157), beside cmd.
func (d Dashboard) feedAgainCmd(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if !d.mapPane.feedAgain {
		return d, cmd
	}
	d, again := d.askFeed()
	return d, tea.Batch(cmd, again)
}

// setFeed sets the feed's overlays, each with its span in the mode (D-98),
// takes off the ones it no longer has, keeps its notes, and draws.
func (d Dashboard) setFeed(feed MapFeed) Dashboard {
	m := d.mapPane.m
	shown, given := map[string]bool{}, map[string]tuimaps.Overlay{}
	notes := append([]string(nil), feed.Notes...)
	refused := map[string]int{} // a layer's refusals, said once (U2-29)
	firstErr := map[string]string{}
	var order []string
	for _, o := range feed.Overlays {
		if t, ok := feed.Times[o.ID]; ok {
			o.During = d.spanFor(t)
		}
		// AN UNCHANGED OVERLAY IS NOT HANDED IN AGAIN (UAT-1 U1-28). A Set
		// replaces what the library prepared, and until a Work prepares it
		// again the area is not drawn: re-sending the same alerts with every
		// new snapshot would blink the area out between frames.
		if prev, ok := d.mapPane.given[o.ID]; ok && SameOverlay(prev, o) {
			shown[o.ID], given[o.ID] = true, o
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err != nil {
			key, _, _ := strings.Cut(o.ID, "/")
			if refused[key] == 0 {
				order, firstErr[key] = append(order, key), err.Error()
			}
			refused[key]++
			continue
		}
		shown[o.ID], given[o.ID] = true, o
	}
	for _, key := range order { // ours to fix, never the listener's: the diagnostics', once a layer (U2-29, D-124)
		d.problem(d.layerLabel(key) + ": " + strconv.Itoa(refused[key]) + " not drawn - " + firstErr[key])
	}
	for id := range d.mapPane.shown {
		if !shown[id] {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
		}
	}
	d.mapPane.shown, d.mapPane.given, d.mapPane.notes, d.mapPane.inMissing, d.mapPane.inView = shown, given, notes, feed.InMissing, feed.InView
	d.mapPane.drawnSev = map[string]bool{}
	for _, o := range feed.Overlays {
		for _, f := range o.Features {
			if w := strings.ToLower(f.Severity.Word()); w != "" {
				d.mapPane.drawnSev[w] = true
			}
		}
	}
	return d.renderMap()
}

// mapFailedText is what the window says when the map cannot be drawn at
// all: that, and the way to try again (D-124).
const mapFailedText = "The map could not be drawn. Close it with esc and press g to open it again."

// problem hands the diagnostics something that went wrong that the listener
// cannot act on (D-124): never said to them.
func (d Dashboard) problem(p string) {
	if d.cfg.MapProblem != nil {
		d.cfg.MapProblem(p)
	}
}

// layerLabel is a layer's name as the Overlays menu says it, or its key.
func (d Dashboard) layerLabel(key string) string {
	for _, l := range d.cfg.MapLayers {
		if l.Key == key {
			return l.Label
		}
	}
	return key
}

// mapHelpRow is one row of Help's MAP group: several actions that are one
// idea, listed on one line so the group fits (D-61's ten keys in four rows).
type mapHelpRow struct{ keys, help string }

// shiftArrows writes the shifted arrows short, so the radar's row fits Help's
// column (D-86): as glyphs, or under --ascii as "S-left" and "S-right".
func shiftArrows(keys string, ascii bool) string {
	if ascii {
		return strings.NewReplacer("shift+left", "S-left", "shift+right", "S-right").Replace(keys)
	}
	return strings.NewReplacer("shift+left", "⇧←", "shift+right", "⇧→").Replace(keys)
}

// mapHelpRows are the MAP group's rows, read from the merged keys so a
// [keys] rebind shows.
func mapHelpRows(keys term.KeyMap, ascii bool) []mapHelpRow {
	keys0 := func(act term.Action) []string { return keys[act].Keys }
	join := func(acts ...term.Action) string {
		var all []string
		for _, a := range acts {
			all = append(all, keys[a].Keys...)
		}
		return strings.Join(all, ", ")
	}
	rows := []mapHelpRow{
		{join(actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight), "Pan"},
		{join(actMapPrev, actMapNext), "Previous / Next Location"},
		{join(actMapZoomIn, actMapZoomOut), "Zoom In / Out"},
		{join(actMapScrollUp, actMapScrollDown), "Scroll"},
		{join(actMapAlerts, actMapOverlays), "Area Alerts / Overlays"},
		{shiftArrows(join(actMapPlay, actMapBack, actMapOn, actMapNewest), ascii), "Play / Back / On / Now"},
		{join(actMapRadar, actMapHighLow), "Radar Mode / Forecast Hi-Lo"}, // D-94, D-97
	}
	// D-77: the region keys in two rows of three, each number's region named
	// in order - six rows push Help past its window.
	for _, half := range [][]int{{0, 1, 2}, {3, 4, 5}} {
		var keys, names []string
		for _, i := range half {
			keys, names = append(keys, keys0(mapRegionActs[i])...), append(names, mapRegionShort[i])
		}
		rows = append(rows, mapHelpRow{strings.Join(keys, ", "), "Region: " + strings.Join(names, ", ")})
	}
	return rows
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

// applyViewSettled asks the feed again once the view has stood still: the map
// draws every alert in view (D-66, D-76). A superseded tick is dropped.
func (d Dashboard) applyViewSettled(v mapViewSettledMsg) (tea.Model, tea.Cmd) {
	if d.modal != modalMap || v.gen != d.mapPane.viewGen {
		return d, nil
	}
	d.mapPane.viewAsked = v.gen
	d, feed := d.requestFeed().askFeed()
	d, radar := d.askRadar()
	d, temp := d.askTemp()
	return d, tea.Batch(feed, radar, temp) // the view's alerts, its radar (W8) and its temperature (W10)
}

// SameOverlay reports whether an overlay is the one already handed in, so it
// is not handed in again (UAT-1 U1-28, UAT-2 U2-13). A MISSING VALUE IS NaN,
// AND NaN IS NEVER EQUAL TO ITSELF: compared as it is, a grid with any value
// missing - a wind grid's gusts said nowhere (D-136) - would be new at every
// answer, and blink. Two missing values here are the same.
func SameOverlay(a, b tuimaps.Overlay) bool {
	if (a.Grid == nil) != (b.Grid == nil) {
		return false
	}
	if a.Grid != nil {
		ga, gb := *a.Grid, *b.Grid
		if !sameFloats(ga.Values, gb.Values) || !sameFloats(ga.From, gb.From) || !sameFloats(ga.Gusts, gb.Gusts) {
			return false
		}
		ga.Values, gb.Values, ga.From, gb.From, ga.Gusts, gb.Gusts = nil, nil, nil, nil, nil, nil
		a.Grid, b.Grid = &ga, &gb
	}
	return reflect.DeepEqual(a, b)
}

// sameFloats reports whether two lists hold the same values, a missing one
// matching a missing one.
func sameFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] && !(math.IsNaN(a[i]) && math.IsNaN(b[i])) {
			return false
		}
	}
	return true
}
