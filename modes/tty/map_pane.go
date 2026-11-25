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
	ionomaps "github.com/branden-thompson/go-ionomaps"
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

// mapBrailleNote is Help's line on the picture outside --ascii (F-191,
// FR-7.1): the terminal cannot be asked whether its font has braille, so the
// words say what a font without it shows and name the remedy.
const mapBrailleNote = "The map is drawn in braille. If it shows boxes or question marks, the font has none: choose one that has braille, or run Watchpost with --ascii for the words alone."

// The window's last line says what the picture is while it is not whole
// (FR-3.4, W1.15's loading indicator). A whole picture leaves it blank.
const (
	mapLoadingText = "Loading map detail…"
	// mapAlertsLoadingText is said until the alerts for the view are in (D-266).
	mapAlertsLoadingText = "Alerts for the map are loading."
	mapOfflineText       = "Offline: the basemap could not be fetched, so this is the coarser picture the map already holds."
	mapCoarseText        = "This is a coarser picture: the map has no finer tiles for this view."
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
	m       *tuimaps.Map
	lines   []string
	changed uint64
	ticks   uint64
	region  geo.Region // the region the map is held inside (FR-2.1)
	outside string     // the place that is in no region, when it is not (FR-2.5)
	prop    bool       // in the Propagation mode (D-153); never saved (D-154)

	// The Propagation mode's own session (W4.1, C-3): made as the mode opens and
	// dropped whole when it is left; a late result, answer or tick of a left
	// session is dropped by its generation.
	propSess    propSession // the open mode's own state, dropped whole when it is left (C-3)
	propGen     uint64
	propTickSeq uint64 // the latest refresh tick set: an earlier one asks nothing

	// The snapshot held and the answers worked out from it, kept across sessions
	// until the map is released, so the mode shows them at once when it opens
	// again; each answer's key carries the snapshot as kept (propMade, RC-5).
	propSnap        *ionomaps.Snapshot
	propErr         error
	propSnapCorrect bool                      // the no-readings correction the snapshot shown was asked with (D-105)
	propUpdatedAt   time.Time                 // when the last was applied, by the window's clock (FR-3.6)
	propBuilt       uint64                    // how many snapshots have been kept: the answers' keys carry it (L-F3)
	propChart       propChart                 // the reference chart's answers (W5.1)
	propChartAsked  propChartKey              // what the chart was last asked for: asked once
	propReach       propReach                 // the frequency's reach (W5.2)
	propReachAsked  propReachKey              // what the reach was last asked for: asked once
	propCentre      propCentre                // the map centre's answer (FR-1.11, D-180)
	propCentreAsked propCentreKey             // what the centre was last asked for: asked once
	propAheadSnaps  map[int]ionomaps.Snapshot // the hours ahead worked out from the snapshot held, each a snapshot of its own
	propAheadAsked  int                       // the hour ahead last asked for: asked once

	// What the listener set in the mode, kept across sessions.
	propLayer     int     // MUF(3000) or foF2 (D-155), kept across sessions until the map is released
	propChartRow  int     // the chart's row its typical day is drawn for: 0 the near-vertical area
	propFreqMHz   float64 // the frequency held (D-178), 0 for none
	propFreqLabel string  // and as it is said: "14.200 MHz" or "40m"
	propAhead     int     // the hour shown ahead of the snapshot's, 0 for now (FR-1.13)

	// The mode's entries as they are typed; a find in flight ends with its
	// session (dropFind, D-27).
	propFreqQuery  string             // what the entry holds as it is typed
	propFreqErr    string             // why the entry did not take it
	propFindQuery  string             // the place finder's query as it is typed (FR-1.10)
	propFindErr    string             // why it found none
	propFindBusy   bool               // a find in flight
	propFindSeq    uint64             // the latest find: an earlier one's answer is dropped
	propFindCancel context.CancelFunc // cancels the find in flight, its request with it

	// The place found, never kept past the mode (D-27).
	propFound   *snapshot.LocationRef // the place found and marked; never kept (D-27)
	foundMarked bool                  // its mark is on the map

	// What is handed to the map, and where the mode answers from.
	propGiven  map[string]tuimaps.Overlay // the Propagation overlays handed in
	zoneGiven  map[string]tuimaps.Overlay // the service area handed in (D-195)
	tower      tuimaps.Place              // the tower's mark placed, or none
	roleDrawn  Surface                    // the UI mode the place and marks were set for
	nightGiven map[string]tuimaps.Overlay // the night handed in (D-199)
	nightAt    time.Time                  // the moment the night handed in is for
	propOrigin propPlace                  // where the Propagation mode answers from, worked out in Update (D-193)

	// The reader's place in the words (D-211).
	wordsAt       wordsPlace  // the reader's place in the words, kept until they scroll (D-211)
	wordsAnchored wordsAnchor // what the words were laid out from when the place was last found

	status    tuimaps.Status // the last frame's: whole, or still sharpening
	pending   bool           // work was waiting when it was drawn
	offline   bool           // a tile failed since the picture was last whole
	gen       uint64         // raised by every draw: the window's memo keys on it, so a frame drawn after a landing is never replayed over (F-30)
	failed    string         // why the map could not be built or drawn, said in the window
	diskOff   bool           // a clear let the disk cache go; named again when the app answers (PF-4)
	clearErr  error          // what the live map refused in a clear, for the app's answer (IS-M5)
	calls     *[]string      // tests only: the library calls made, by name, in order
	where     *[]callSite    // tests only: each call and the goroutine it ran on (W2.2)
	workers   *mapWorkers    // the commands running for this map, joined on close (W2.6)
	tickAt    time.Time      // the tick outstanding, at the library's NextCall (W2.2)
	alertsOn  bool           // the Description box is open; it waits for A (D-87)
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
	given        map[string]tuimaps.Overlay // what was last handed to the map, by id: an unchanged overlay is not handed in again (U1-28)
	notes        []string                   // the feed's notes, printed under the map
	undrawn      map[string]bool            // the undrawn areas' words already handed to the diagnostics, each said once while it lasts
	inMissing    map[string]bool            // the feed's alerts whose missing zones hold the place
	inView       []snapshot.Alert           // the feed's alerts the station does not hold, which the description names

	// The radar (W8): its request generation, the loops handed in by id, the
	// source for the chip and its note, and the loop's line as last drawn.
	radarBusy, radarAgain bool // one request at a time, a later want kept (D-85)
	// lanes are the feed's two parts (D-268): the alerts, delivered first and
	// on their own, and the other layers after them.
	lanes [feedLanes]feedLane
	// viewAsked is the newest move whose settle tick has asked for its view:
	// below viewGen, a move is still waiting to ask (the instrument's "still").
	viewAsked uint64
	clock     mapClock // the timing instrument's clock (map_timing.go)
	// radarRegion is the region of the loop being fetched, and radarStop
	// cancels it: leaving the region leaves its answer nowhere to draw (D-130).
	radarRegion                       string
	radarStop                         context.CancelFunc
	radarAt                           time.Time
	radarGiven                        map[string]tuimaps.Overlay
	radarSource, radarNote, radarLine string
	radarAhead                        string                 // the source of the loop's hours ahead, "HRRR" (D-113), or none
	radarTimeline                     []string               // the loop's timeline (D-86), drawn in Update
	radarBadgeTime                    string                 // the badge's time row (D-92), drawn in Update
	fcTimeline                        []string               // Forecast mode's steps (D-94), drawn in Update
	report                            tuimaps.PlaceReport    // the library's answers for the selected place, as last drawn
	motion                            []tuimaps.MotionReport // the radar's observed motion near it (D-363)
	legend                            []tuimaps.LegendEntry  // what the map draws now, for the legend (W1.17)
	drawnSev                          map[string]bool        // the severities the feed drew, by word: the legend keys these (D-54, "as drawn")
	feedGen                           uint64                 // the feed last asked for; an older answer is dropped
	feed                              *MapFeed               // the feed as last answered, layers applied: set again with each mode's spans (D-98)

	// The temperature (W10): one request at a time as the radar's, the answer
	// held, the grids handed in by id, and the anchor it was built at.
	tempBusy, tempAgain bool
	tempAt, tempAnchor  time.Time
	temp                MapTemperature
	tempGiven           map[string]tuimaps.Overlay
	tempHanded          bool // the last setTemp handed the library a changed grid: its work then draws the map (D-85)
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
	region geo.Region // the region the frame was bound to (D-28, D-77); none in the Propagation mode
	mode   mapMode
}

// MapFeed is what the map draws from the station's data (0.18.0 W5): an
// overlay per alert, and the notes the window prints under the map.
type MapFeed struct {
	Overlays []tuimaps.Overlay
	Notes    []string
	// Undrawn says which alert areas are not fully drawn - nothing the
	// listener can act on, so the diagnostics' alone (D-124, D-240). Where a
	// missing zone holds the place, the description says so (InMissing).
	Undrawn   []string
	InMissing map[string]bool // alerts whose missing zones hold the selected place, by alert id
	// InView are the alerts drawn that the station does not hold - the
	// view's (D-66) - so the description can name them in full.
	InView []snapshot.Alert
	// Times are when each overlay is, by id: the window draws each only while
	// the mode's moment meets it (D-98).
	Times map[string]TimedOverlay
	// AlertsFailed says the view's alerts could not be asked: the description
	// says so rather than that none is there (D-266).
	AlertsFailed bool
}

// mapFeedMsg is the feed's answer, to the request it was asked in.
type mapFeedMsg struct {
	gen  uint64 // the data asked for (requestFeed's generation)
	seq  uint64 // the ask's number: a cancelled ask's answer carries an older one (D-157)
	lane int    // the part it answers (D-268): laneAlerts, the zero value, or laneLayers
	feed MapFeed
}

// The feed's lanes (D-268), and how many there are.
const (
	laneAlerts = iota
	laneLayers
	feedLanes
)

// laneParts is the part each lane asks the app for.
var laneParts = [feedLanes]FeedPart{laneAlerts: FeedAlerts, laneLayers: FeedLayers}

// feedLane is one part of the feed: its one ask in flight (D-157) - what it
// is for, its number and its cancel, wanted again when new data lands while
// it runs - and its newest answer drawn, with the place and view it was for.
type feedLane struct {
	busy, again bool
	seq         uint64
	asked       feedKey
	ctx         context.Context
	stop        context.CancelFunc
	applied     uint64   // the generation of the newest answer drawn
	landed      bool     // an answer has been drawn
	landedFor   feedKey  // the place and view the newest answer drawn was for
	feed        *MapFeed // the newest answer drawn, layers applied
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

// mapWorkedMsg is one Work command's outcome: whether it landed anything,
// and whether it was not started, mapWorkLoops being in Work (PF-3).
type mapWorkedMsg struct {
	did, skipped bool
}

// call makes one library call, recording its name for the tests that check
// where calls are made (W2.2). A panic inside it is stopped and kept, and
// the Update it was made in, or the next, marks the map failed (QA-11).
func (p mapPane) call(name string, f func()) {
	if p.calls != nil {
		*p.calls = append(*p.calls, name)
	}
	if p.where != nil {
		*p.where = append(*p.where, callSite{name: name, goroutine: goroutineID()})
	}
	defer func() {
		if r := recover(); r != nil {
			p.workers.keepPanic(name, r)
		}
	}()
	f()
}

// opened is the map in the window the Dashboard has just opened for it
// (0.19.6 design MS-1's S1): built, zoomed and drawn, and its data asked;
// built is false where nothing was, and the window says why.
func (mm mapModel) opened(e mapEnv) (mapModel, tea.Cmd, bool) {
	mm = mm.timeFrom("open", e)
	loc := mm.mapPlace(e)
	if loc == nil || mm.mapsOff {
		return mm, nil, false // FR-1.3, FR-1.6: the window says so; nothing is built for it
	}
	if mm.mapPane.m == nil {
		if mm.mapHooks.NewMap == nil {
			mm.mapPane.failed = "The map is not available in this build."
			return mm, nil, false
		}
		m, err := mm.mapHooks.NewMap(mm.mapBodySize(e))
		if err != nil {
			mm.mapPane.failed = mapFailedText // D-124: a path, and the detail to the diagnostics
			mm.problem("Map: not started - "+err.Error(), e)
			return mm, nil, false
		}
		mm.mapPane.m, mm.mapPane.failed, mm.mapPane.workers = m, "", newMapWorkers()
	}
	mm.mapPane.alertsOn, mm.mapPane.menuOn = false, false // D-87: the Description box waits for A
	m, scale, km := mm.mapPane.m, mm.mapScale.zoom(), float64(mm.mapNearbyKm)
	mm.mapPane.call("Zoom", func() { _ = m.Zoom(scale) })        // every open is at the chosen scale (W4.3); the bound holds it
	mm.mapPane.call("SetNearby", func() { _ = m.SetNearby(km) }) // the description's "near" as chosen (W9.2)
	mm = mm.resetPlayback()                                      // Forecast mode opens on Now, stopped (D-94)
	mm.mapPane.modeChip = false
	mm.mapPane.prop = false // every open is a weather mode: the Propagation mode is never kept (D-154)
	mm = mm.stopPropagation(e).setProp(e)
	mm = mm.ensureMainOverlay(e) // D-103: a map opened in Forecast mode is never blank
	mm = mm.applyDetail(e).applyPlayback().showStep(e).refreshMapCost(e).followSelection(e).setZone(e).requestFeed()
	mm.mapPane.viewAsked = mm.mapPane.viewGen // the open asks for its view: a move's tick dropped while closed is answered here (W14)
	mm = mm.renderMap(e)
	mm, feed := mm.askFeed(e)
	mm, radar := mm.askRadar(e)
	mm, temp := mm.askTemp(e)
	return mm, tea.Batch(mm.mapWorkCmd(e), feed, radar, temp), true
}

// boundMap holds the map inside its region (W4, W9.1: the library's bound,
// not a host clamp). The least zoom is the one at which the window's
// view fits inside the region on both axes, so no frame is wider than the
// region either way; it depends on the window's size, so every resize sets
// it again. It is worked out in the library's published scale - 256-dot
// tiles, a braille cell two dots wide and four high.
func (mm mapModel) boundMap(e mapEnv) mapModel {
	m, r := mm.mapPane.m, mm.mapPane.region
	if m == nil {
		return mm
	}
	var b tuimaps.Bound // the Propagation mode's: none, the whole world (D-21)
	name := "the whole world"
	switch mm.mapMode(e) {
	case modeRadar, modeForecast:
		if r.Name == "" {
			return mm
		}
		b, name = tuimaps.Bound{MinZoom: regionFitZoom(r, mm.mapBodySize(e)), W: r.W, S: r.S, E: r.E, N: r.N}, r.Name
	case modePropagation:
	}
	var err error
	mm.mapPane.call("SetBound", func() {
		err = m.SetBound(b)
	})
	if err != nil {
		mm.problem("Map bound: "+name+" refused - "+err.Error(), e) // ours to fix, never the listener's (D-124)
	}
	return mm.holdWorld(e)
}

// worldLatitude is as far north and south as the map draws.
const worldLatitude = 85.0511

// worldZoom is the Propagation mode's least zoom: the whole world fills the
// map one way or the other.
func worldZoom(size tuimaps.Size) float64 {
	return regionFitZoom(geo.Region{W: -180, S: -worldLatitude, E: 180, N: worldLatitude}, size)
}

// holdWorld keeps the Propagation mode's view at worldZoom or nearer. Its map
// has no bound, so that a view may cross any meridian, and the library would
// otherwise zoom out past the world; the weather modes are held by theirs.
func (mm mapModel) holdWorld(e mapEnv) mapModel {
	m := mm.mapPane.m
	if m == nil {
		return mm
	}
	if mm.mapMode(e) == modePropagation {
		least := worldZoom(mm.mapBodySize(e))
		if _, z := m.Centre(); z < least {
			mm.mapPane.call("Zoom", func() { _ = m.Zoom(least) })
		}
	}
	return mm
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
func (mm mapModel) mapBodySize(e mapEnv) tuimaps.Size {
	cols := mm.mapCols(e)                                                       // border to border (U1-27)
	avail := e.rows() - mm.mapChromeRows(mm.noteRowCount(mm.mapTextW(e), e), e) // under the map its notes, the status and the chips, and the radar's held rows (D-86, D-89); the description is a box over it (D-63)
	return tuimaps.Size{Cols: cols, Rows: max(min(avail, e.rows()), 1)}
}

// mapFits reports whether the map can be drawn at the floor (FR-1.4).
func (mm mapModel) mapFits(e mapEnv) bool {
	s := mm.mapBodySize(e)
	return s.Cols >= mapMinBody.Cols && s.Rows >= mapMinBody.Rows
}

// descBlock is the description as the window lays it above the map: wrapped
// to the body's width, and a blank line under it. Empty when its mode is off.
func (mm mapModel) descBlock(width int, e mapEnv) []string {
	if mm.mapDesc == mapDescOff && !e.Opts.ASCII {
		return nil
	}
	var out []string
	for _, l := range mm.describeLinesAll(e) {
		out = append(out, render.WrapText(l, width)...)
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	return out
}

// renderMap draws the map into the pane. It is called from Update only.
func (mm mapModel) renderMap(e mapEnv) mapModel {
	mm = mm.followRole(e) // a swap of UI modes with the map open: its place and marks follow (D-195)
	m := mm.mapPane.m
	if m == nil || mm.outsideStated(e) {
		return mm // FR-2.5: nothing wider is drawn in its place
	}
	var frame tuimaps.Frame
	var err error
	miles := e.Opts.Units == render.UnitF
	mm.mapPane.call("Units", func() { m.Units(miles, miles) }) // the description's distances and temperatures in the station's units
	depth := mm.mapDepthHint(e)
	mm.mapPane.call("ColourDepth", func() { m.ColourDepth(depth) }) // the Setting's depth, or the terminal's (D-207); the same hint again changes nothing
	mm = mm.applyMotion(e)                                          // Reduce motion (D-360), likewise
	mm = mm.reportPlace(e)                                          // first: the description's length decides the map's size
	if !mm.mapFits(e) {
		mm.mapPane.lines = nil
		mm.mapPane.gen++
		mm.timeBelowFloor() // the workload's render check: a map not drawn is never a quiet pass (W14)
		return mm           // FR-1.4: nothing under the floor is drawn; the notice and the description say it
	}
	mm.mapPane.call("Render", func() { frame, err = m.Render(mm.mapBodySize(e), e.Now()) })
	if err != nil {
		mm.mapPane.failed = mapFailedText
		mm.problem("Map: not drawn - "+err.Error(), e)
		return mm
	}
	mm.mapPane.lines, mm.mapPane.failed = insetLines(frame.Lines), ""
	if mm.mapPane.views != nil {
		c, z := m.Centre()
		*mm.mapPane.views = append(*mm.mapPane.views, mapView{centre: c, zoom: z, size: mm.mapBodySize(e), region: mm.mapPane.region, mode: mm.mapMode(e)})
	}
	mm.mapPane.status = frame.Status
	mm.mapPane.call("Legend", func() { mm.mapPane.legend = m.Legend() })
	mm.mapPane.call("Pending", func() { mm.mapPane.pending = m.Pending() > 0 })
	var warnings []tuimaps.Warning
	mm.mapPane.call("Warnings", func() { warnings = m.Warnings() })
	for _, w := range warnings { // the library's, drained each frame (P10-02)
		if w.Kind == tuimaps.TileFailed {
			mm.mapPane.offline = true // held until the picture is whole again
			continue
		}
		if w.Kind == tuimaps.SetRefused {
			continue // setFeed said each layer's refusals once already (U2-29)
		}
		mm.problem(warningText(w), e) // nothing the listener can act on: the diagnostics' (D-124)
	}
	if frame.Status == tuimaps.Complete {
		mm.mapPane.offline = false
	}
	mm.mapPane.title = mm.mapTitleAt(mm.mapBodySize(e), e)
	if mm.inPropagation(e) {
		mm.mapPane.propOrigin = mm.propOriginNow(e) // read here, in Update, as the title is; the frame only prints it (D-41, D-193)
	}
	mm.mapPane.radarLine = mm.radarStatus(e)                     // read from the library here, in Update; the frame only prints it (D-41)
	mm.mapPane.radarTimeline = mm.radarTimeline(mm.scrubW(e), e) // beside the controls (D-103)
	mm.mapPane.fcTimeline = mm.forecastTimeline(mm.scrubW(e), e)
	mm.mapPane.radarBadgeTime = mm.radarBadgeTimeNow(e)
	mm.recordFrame(frame.Status, e) // D-198: nothing unless the recorder is on
	mm.mapPane.gen++
	mm.mapPane.changed, mm.mapPane.ticks = frame.Changed, frame.FrameTicks
	mm = mm.timeDraw(e)
	return mm
}

// mapWorkCmd is the next Work command, or nil when the window is closed,
// nothing is pending, or mapWorkLoops are in Work already - each of them asks
// again while work waits, so the work is theirs (PF-3).
func (mm mapModel) mapWorkCmd(e mapEnv) tea.Cmd {
	m, w := mm.mapPane.m, mm.mapPane.workers
	if m == nil || e.Window != mapModalMap || w.loopsFull() {
		return nil
	}
	pending := 0
	mm.mapPane.call("Pending", func() { pending = m.Pending() })
	if pending == 0 {
		return nil
	}
	pane := mm.mapPane
	return w.cmd(context.Background(), mapWorkedMsg{}, func(ctx context.Context) tea.Msg {
		if !w.startLoop() { // commands made before any ran: the bound is checked again as each starts
			return mapWorkedMsg{skipped: true}
		}
		defer w.endLoop()
		did := false
		pane.call("Work", func() { did, _ = m.Work(ctx) }) // a failed tile is the library's to retry, and its warning says so
		return mapWorkedMsg{did: did}
	})
}

// applyMapWorked draws what a Work command landed, and asks for the next. A
// loop that was not started asks for nothing: the loops in Work do (PF-3).
func (mm mapModel) applyMapWorked(v mapWorkedMsg, e mapEnv) (mapModel, tea.Cmd) {
	if v.skipped {
		return mm, nil
	}
	if v.did && e.Window == mapModalMap {
		mm = mm.renderMap(e)
	}
	return mm, mm.mapWorkCmd(e)
}

// insetLines indents each drawn line by one cell, as every window's body is.
func insetLines(lines []string) []string {
	return append([]string(nil), lines...) // flush: the map runs border to border (UAT-1 U1-27)
}

// mapCols is the map window's width inside its two borders: the map runs
// border to border (UAT-1 U1-27), where every other window insets its body.
func (mm mapModel) mapCols(e mapEnv) int { return max(e.cols()-2, 1) }

// mapTextW is how wide the window's words wrap: one cell inside the map's
// width, for the space every line of words starts with, so none is cut at
// the border.
func (mm mapModel) mapTextW(e mapEnv) int { return max(mm.mapCols(e)-1, 1) }

// mapTitle names the place the map is on.
func (mm mapModel) mapTitle(e mapEnv) string {
	if mm.mapPane.title != "" && e.Window == mapModalMap && mm.mapPane.m != nil {
		return "Map " + e.Opts.Glyphs().Dot + " " + mm.mapPane.title // D-64: what is in view
	}
	if loc := mm.mapPlace(e); loc != nil {
		return "Map " + e.Opts.Glyphs().Dot + " " + loc.Label
	}
	return "Map"
}

// mapBodyLines is what the window shows: the map, or a stated reason why not.
func (mm mapModel) mapBodyLines(e mapEnv) []string {
	switch {
	case mm.mapPlace(e) == nil:
		return []string{noSelectionText}
	case mm.mapsOff:
		return []string{mapsOffText}
	case mm.outsideStated(e):
		return []string{mm.outsideText()}
	case mm.mapPane.failed != "":
		return []string{mm.mapPane.failed}
	}
	width := mm.mapTextW(e)
	picture := mm.mapPicture(e)
	var out []string // D-69: the map says nothing of what it sends; Settings does, beside the maps row
	switch {
	case e.Opts.ASCII:
		out = []string{asciiMapText, ""} // FR-1.7, FR-1.8: the description in place of the picture
	case mm.mapDesc != mapDescInstead && !mm.mapFits(e):
		out = append(render.WrapText(belowFloorText(mm.mapBodySize(e)), width), "") // FR-1.4: the notice carries the description
	}
	if !picture {
		text := mm.textInstead(out, width, e)
		if len(text) > e.rows() {
			text = mm.textInstead(out, width-modalRailCols, e) // it will scroll: wrapped clear of the rail, so the window need not wrap it again into stubs (D-211)
		}
		return text
	}
	size := mm.mapBodySize(e)
	out = append(out, mm.withModeChip(mm.withEdgeChip(mm.withControls(mm.withQuotaNotice(mm.withOverlays(mm.withPropCentre(mm.withDescriptionBox(mm.mapPane.lines, size, e), size, e), e), size, e), size, e), size), size)...) // the notice under every control (D-181)
	if !mm.radarTimelineOn() {
		for _, l := range mm.noteLines(width, e) {
			out = append(out, " "+l)
		}
		for _, l := range strings.Split(mm.mapStatusLine(e), "\n") {
			out = append(out, " "+l)
		}
		return out
	}
	return append(out, mm.scrubRows(width, e)...) // D-103: the rows under the map, in either mode
}

// modalRailCols are the columns a scrolling window's rail takes from its
// text (wrapModalAt).
const modalRailCols = 3

// textInstead is the window's text in place of the picture, after its
// leading lines: the description, or the open menu; what the picture's rows
// say - the region beyond an edge, the status, the loop's line (AX-5); and
// the notes, wrapped to a width.
func (mm mapModel) textInstead(lead []string, width int, e mapEnv) []string {
	out := slices.Clone(lead)
	block := mm.descBlock(width, e)
	if mm.mapPane.menuOn {
		block = mm.overlaysText(e) // D-267: the open menu owns the keys, so it is drawn where the picture would be
	}
	for _, l := range block {
		out = append(out, " "+l)
	}
	for _, said := range []string{mm.edgeWords(), mm.mapStatusText(e)} {
		for _, l := range render.WrapText(said, width-1) {
			out = append(out, " "+l)
		}
	}
	switch mm.mapMode(e) {
	case modeRadar, modeForecast:
		if mm.radarTimelineOn() {
			for _, l := range render.WrapText(mm.loopWords(e), width-1) {
				out = append(out, " "+l)
			}
		}
	case modePropagation: // its own words carry its hours
	}
	for _, l := range mm.noteLines(width, e) {
		out = append(out, " "+l)
	}
	return out
}

// mapPicture reports whether the window draws the picture: not under
// --ascii, not with the description in its place, not below the floor.
func (mm mapModel) mapPicture(e mapEnv) bool {
	return !e.Opts.ASCII && mm.mapDesc != mapDescInstead && mm.mapFits(e)
}

// followMenuFocus scrolls the window so the menu's focused row is in view
// while the menu is drawn as text (D-267): the text can be longer than the
// window, and a row the cursor is on is never off screen.
func (mm mapModel) followMenuFocus(e mapEnv) (mapModel, mapWant) {
	if !mm.mapPane.menuOn || mm.mapPicture(e) {
		return mm, mapWant{}
	}
	lines, mark := mm.windowLines(e), e.Opts.Glyphs().Pointer+" "
	at := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(render.Plain(l), mark) })
	if at < 0 {
		return mm, mapWant{}
	}
	rows, scroll := e.rows(), e.Scroll
	switch {
	case at < scroll:
		scroll = at
	case at >= scroll+rows:
		scroll = at - rows + 1
	}
	return mm, mapWant{Scroll: min(scroll, max(len(lines)-rows, 0)), ScrollSet: true}
}

// reportPlace keeps the library's answers for the selected place, which the
// description reads. It needs no Work and no frame.
func (mm mapModel) reportPlace(e mapEnv) mapModel {
	m, loc := mm.mapPane.m, mm.mapPlace(e)
	mm.mapPane.report, mm.mapPane.motion = tuimaps.PlaceReport{}, nil
	if m == nil || loc == nil {
		return mm
	}
	var rep tuimaps.Report
	mm.mapPane.call("Report", func() {
		rep, _ = m.Report([]tuimaps.Place{{ID: "selected", Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}}})
	})
	if len(rep.Places) > 0 {
		mm.mapPane.report = rep.Places[0]
	}
	mm.mapPane.motion = rep.Motion
	return mm
}

// noteLines are the rows under the map before its status: the badges
// (D-133), then the notes that come and go, each only while it applies
// (D-132), then the cost warning; wrapped to the map's width.
func (mm mapModel) noteLines(width int, e mapEnv) []string {
	if mm.mapMode(e) == modePropagation { // its badges and words are in its block (D-157); its one note, the legend's MHz (D-160)
		if !mm.propNoteShown(e) {
			return nil
		}
		return []string{mm.propLegendRows(width)[1]}
	}
	out := mm.badgeRows(width, e)
	if mm.notesYield(width, e) {
		out = append(out, render.WrapText(notesYieldLine(len(mm.mapNotesNow(e))), width)...) // D-159: the map keeps its floor
	} else {
		for _, n := range mm.mapNotesNow(e) {
			out = append(out, render.WrapText(n, width)...)
		}
	}
	out = append(out, mapCostLine(mm.mapCost, width)...) // FR-9.2: said where the cost is seen, in D-82's words laid out as D-89
	return out
}

// propNoteShown reports whether the Propagation mode's note line, its
// legend's MHz, is shown: not where no picture is meant (not mapPicture: its
// fit counts this line).
func (mm mapModel) propNoteShown(e mapEnv) bool {
	return !e.Opts.ASCII && mm.mapDesc != mapDescInstead
}

// noteRowCount is how many rows noteLines gives at a width, counted without
// drawing the Propagation mode's legend: the map's size asks it many times a
// frame.
func (mm mapModel) noteRowCount(width int, e mapEnv) int {
	if mm.mapMode(e) == modePropagation {
		if !mm.propNoteShown(e) {
			return 0
		}
		return 1
	}
	return len(mm.noteLines(width, e))
}

// mapNotesNow is the map's notes as they stand: the feed's, the radar's
// missing data, what a temperature source lacks. The one list the window
// and the Status window's MAP STATUS both read (D-159), so they cannot
// disagree.
func (mm mapModel) mapNotesNow(e mapEnv) []string {
	out := append([]string(nil), mm.mapPane.notes...)
	if mm.mapPane.radarSource != "" && mm.mapPane.radarNote != "" && mm.layerOn(RadarLayer, e) {
		out = append(out, mm.mapPane.radarNote) // D-84's missing data; a source that did not answer, with its Setting
	}
	return append(out, mm.tempNotes(e)...) // W10: what a source lacks
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
func (mm mapModel) mapChromeRows(noteRows int, e mapEnv) int {
	rows := mapStatusRows + noteRows
	if !mm.radarTimelineOn() {
		return rows
	}
	rows += radarRows + radarExtraRows
	if mm.mapMode(e) == modePropagation {
		rows += propBlockRows - scrubBoxRows // its block's blank row before the place (D-163)
	}
	return rows
}

// notesYield reports whether the notes must collapse to one line (D-159):
// only where even the window's ceiling (D-194) cannot hold the map at its
// floor with every note line shown, and one line would. Growing the window
// comes first.
func (mm mapModel) notesYield(width int, e mapEnv) bool {
	notes := mm.mapNotesNow(e)
	if len(notes) < 2 {
		return false
	}
	full := len(mm.badgeRows(width, e)) + len(mapCostLine(mm.mapCost, width))
	for _, n := range notes {
		full += len(render.WrapText(n, width))
	}
	return mapMinBody.Rows+mm.mapChromeRows(full, e) > e.ceiling()
}

// mapStatusLine says what the picture is while it is not whole, and names
// the window's boxes on a line of their own.
func (mm mapModel) mapStatusLine(e mapEnv) string {
	// D-89's order: the radar's line, the estimate after a slash, then what
	// the picture's status says - W8.8's loop and its age always first.
	lead := mm.mapPane.radarLine
	switch mm.mapMode(e) {
	case modeForecast:
		if mm.mapHooks.MapRadar != nil {
			lead = mm.forecastStatus(e) // D-94: Forecast mode's line in the radar's place
		}
	case modePropagation:
		lead = mm.propStatusWords(e)
	case modeRadar:
	}
	if est := costEstimate(mm.mapCost); est != "" {
		lead = strings.TrimPrefix(lead+" / "+est, " / ")
	}
	status := mm.mapStatusText(e)
	if lead != "" {
		status = strings.TrimSuffix(lead+" · "+status, " · ")
	}
	chips := mm.mapChips(e)
	width := mm.mapTextW(e)
	// TWO LINES, ALWAYS: the status, then the chips (UAT-1 U1-2 - three chips
	// beside the status would cut it even at 133 columns). Both are reserved whether
	// or not the status says anything, so the map's size never depends on the
	// last frame's status.
	return render.TruncateCells(status, width) + "\n" + render.TruncateCells(strings.Join(chips, "  "), width)
}

// mapChips are the window's keys as chips, each naming its key as bound and
// saying its state: the Description box, the mode, Overlays (D-63, D-65,
// D-94; no legend chip, D-103; AX-6, D-365).
func (mm mapModel) mapChips(e mapEnv) []string {
	type chip struct {
		act  term.Action
		name string
	}
	var controls []chip
	switch mm.mapMode(e) {
	case modeRadar, modeForecast:
		description := "Description Off"
		if mm.mapPane.alertsOn {
			description = "Description On"
		}
		controls = []chip{{actMapAlerts, description}, {actMapRadar, mm.radarChipWords(e)}}
	case modePropagation: // the mode's own controls in the weather's place (D-155, D-156)
		controls = []chip{{actMapLayer, propLayerName(mm.mapPane.propLayer)}, {actMapRefresh, "Refresh Now"}}
	}
	var chips []string
	for _, c := range append(controls, chip{actMapProp, mm.propChipWords(e)}, chip{actMapOverlays, "Overlays"}) { // the menu open says itself, the cursor on its row (D-362)
		if keys := mm.mapKeys[c.act].Keys; len(keys) > 0 {
			chips = append(chips, e.Opts.KeyCap(keys[0])+" "+c.name)
		}
	}
	return chips
}

// mapStatusRows is how many rows the status and the chips take.
const mapStatusRows = 2

// mapReleaseAfter is how long the map stays closed before it is let go
// (D-221): a reopen within it is instant, after it a cold open.
const mapReleaseAfter = 5 * time.Minute

// mapReleaseMsg is a close's release coming due, with the close it was
// scheduled at.
type mapReleaseMsg struct{ gen uint64 }

// applyMapRelease lets the map go when it is still closed and no later close
// has scheduled a release of its own (D-221).
func (mm mapModel) applyMapRelease(v mapReleaseMsg, e mapEnv) mapModel {
	if v.gen != mm.mapCloses || e.Shown || mm.mapPane.m == nil {
		return mm
	}
	return mm.releaseMap()
}

// releaseMap closes the library's map and lets go of everything the window
// held for it - the pictures it handed in and the answers they came from -
// so the next open builds a new one. The generations go on from where they
// were, so an answer to an ask made before the release is never taken for a
// new one.
func (mm mapModel) releaseMap() mapModel {
	mm.closeMap()
	p := mm.mapPane
	for _, stop := range []context.CancelFunc{p.lanes[laneAlerts].stop, p.lanes[laneLayers].stop, p.radarStop} { // the asks in flight (P10-02)
		if stop != nil {
			stop()
		}
	}
	mm.mapPane = mapPane{calls: p.calls, where: p.where, views: p.views, clock: p.clock, changed: p.changed, ticks: p.ticks,
		gen: p.gen + 1, viewGen: p.viewGen, viewAsked: p.viewAsked, feedGen: p.feedGen, lanes: keptSeqs(p.lanes), fcGen: p.fcGen,
		propGen: p.propGen, propTickSeq: p.propTickSeq}
	return mm
}

// closeMap lets the library's map go. The app calls it when the station
// stops; a closed pane builds a new map on the next open.
func (mm mapModel) closeMap() {
	mm.mapPane.workers.shut() // cancelled and joined first: nothing is inside the map when it closes (W2.6)
	if m := mm.mapPane.m; m != nil {
		mm.mapPane.call("Close", func() { m.Close() })
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

// mapActions is the map window's actions in the order Help lists them, the
// Bands chart's two among them: their scope is the chart's own, and Help says
// them on the pan row (D-256).
var mapActions = append([]term.Action{actMapPanUp, actMapPanDown, actMapPanLeft, actMapPanRight, actMapPrev, actMapNext, actMapZoomIn, actMapZoomOut, actMapScrollUp, actMapScrollDown, actMapAlerts, actMapRadar, actMapProp, actMapLayer, actMapRefresh, actMapChart, actChartPrev, actChartNext, actMapFreq, actMapFind, actMapOverlays,
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
		actMapAlerts:     {Keys: []string{"A"}, Help: "Description"},          // D-63, D-365: the words over the upper left
		actMapOverlays:   {Keys: []string{"O"}, Help: "Overlays"},             // D-65: the weather layers and the map's detail
		actMapRadar:      {Keys: []string{"R"}, Help: "Radar On / Off"},       // D-94: Radar mode, else Forecast mode
		actMapProp:       {Keys: []string{"P"}, Help: "Propagation On / Off"}, // D-153: the Propagation mode, and back
		actMapLayer:      {Keys: []string{"L"}, Help: "Layer: MUF / foF2"},    // D-155: the Propagation mode's layer
		actMapRefresh:    {Keys: []string{"U"}, Help: "Refresh Now"},          // D-156: its update now, under D-39
		actMapChart:      {Keys: []string{"B"}, Help: "Best Bands Chart"},     // D-176: the reference chart over the map
		actMapFreq:       {Keys: []string{"F"}, Help: "Your Frequency"},       // D-178: its reach drawn over the field
		actMapFind:       {Keys: []string{"/"}, Help: "Find a Place"},         // FR-1.10: anywhere, never kept
		actMapPlay:       {Keys: []string{"space"}, Help: "Play / Stop"},      // D-61: the playback keys, radar's or the forecast's
		actMapBack:       {Keys: []string{"shift+left"}, Help: "Step Back"},   // D-86: at the timeline's ends
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
	return scopeKeysFrom(defaultMapKeyMap(), overrides)
}

// scopeKeysFrom merges the [keys] entries that name a scope's own actions
// into it; the rest are other scopes', never this one's error.
func scopeKeysFrom(defaults, overrides term.KeyMap) (term.KeyMap, error) {
	own := term.KeyMap{}
	for act, b := range overrides {
		if _, ok := defaults[act]; ok {
			own[act] = b
		}
	}
	keys, _, err := term.Merge(defaults, own)
	return keys, err
}

// onMapKey is the open map window's keyboard (D-61): a key the map binds
// is the map's; any other key goes on to the Observer's handling.
func (mm mapModel) onMapKey(key tea.KeyPressMsg, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	act, bound := mm.mapKeys.Lookup(key.String())
	if mm.mapPane.modeChip { // any key takes the mode's chip away (D-103), as the edge chip's
		mm.mapPane.modeChip = false
		mm.mapPane.gen++
	}
	if mm.mapPane.menuOn && act != actMapOverlays { // D-65: the open menu owns its keys
		if nd, cmd, w, ok := mm.overlaysMenuKey(key.String(), e); ok {
			return nd, cmd, w, true
		}
	}
	if !bound {
		return mm, nil, mapWant{}, false
	}
	mm = mm.timeFrom(string(act), e)
	if nd, w, ok := mm.mapWindowKey(act, e); ok {
		nd, focus := nd.followMenuFocus(e.lent(w)) // D-267: the menu opened, or its scroll moved, without a picture
		return nd, nil, w.and(focus), true
	}
	if mm.mapPane.m == nil {
		return mm, nil, mapWant{}, false
	}
	switch act {
	case actMapRadar: // D-94: Radar mode on and off
		nd, cmd, w := mm.flashMapKey(act, e).switchMode(e)
		return nd, cmd, w, true
	case actMapProp: // D-153: the Propagation mode, and back
		nd, cmd, w := mm.flashMapKey(act, e).togglePropagation(e)
		return nd, cmd, w, true
	case actMapLayer: // D-155: MUF(3000) or foF2
		return mm.flashMapKey(act, e).flipPropLayer(e), mm.mapWorkCmd(e), mapWant{}, true
	case actMapRefresh: // D-156: an update now
		nd, cmd := mm.flashMapKey(act, e).refreshProp(e)
		return nd, cmd, mapWant{}, true
	case actMapChart: // D-176: the reference chart
		mm = mm.flashMapKey(act, e)
		return mm, nil, mm.openPropChart(e), true
	case actMapFreq: // D-178: your frequency
		nd, w := mm.flashMapKey(act, e).openPropFreq(e)
		return nd, nil, w, true
	case actMapFind: // FR-1.10: find a place
		nd, w := mm.flashMapKey(act, e).openPropFind(e)
		return nd, nil, w, true
	case actMapHighLow: // D-97: the days' high or low
		return mm.flashMapKey(act, e).flipHighLow(e), mm.mapWorkCmd(e), mapWant{}, true
	}
	mm = mm.flashMapKey(act, e) // U1-11: the controls' chip blinks
	switch mm.mapMode(e) {
	case modeForecast: // the host steps (D-94)
		if nd, cmd, ok := mm.handleForecastPlayback(act, e); ok {
			return nd, tea.Batch(cmd, nd.mapWorkCmd(e)), mapWant{}, true
		}
	case modeRadar:
		if nd, ok := mm.handlePlayback(act, e); ok {
			return nd, nd.mapWorkCmd(e), mapWant{}, true
		}
	case modePropagation: // the hours ahead (FR-1.13)
		if nd, ok := mm.stepPropAhead(act, e); ok {
			nd, settle := nd.viewMoved(e) // an hour step settles as a pan does: the reach is asked once the hour stands (D-227)
			return nd, tea.Batch(nd.mapWorkCmd(e), settle), mapWant{}, true
		}
	}
	return mm.moveMapView(act, e)
}

// overlaysMenuKey is a key while the Overlays menu is open; false when the
// menu does not take it.
func (mm mapModel) overlaysMenuKey(key string, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	if nd, cmd, w, ok := mm.radioChangesMode(key, e); ok {
		return nd, cmd, w, true
	}
	nd, w, ok := mm.handleOverlaysKey(key, e)
	if !ok {
		return mm, nil, mapWant{}, false
	}
	// AN ARROW MOVES THE CURSOR AND NOTHING ELSE (UAT-2 U2-13, U2-14): only a
	// switch touches the map, so a cursor move asks for no radar or
	// temperature and hands no grid in again.
	if nd.mapLayerChoice == mm.mapLayerChoice && nd.mapDetailChoice == mm.mapDetailChoice && nd.mapDetailLevel == mm.mapDetailLevel {
		nd, focus := nd.followMenuFocus(e)
		return nd, nil, w.and(focus), true
	}
	nd, cmd, applied, _ := nd.menuApplied(mm, e)
	return nd, cmd, w.and(applied), true
}

// menuApplied puts a switch made in the menu on the map, d the window before
// it: the temperature drawn or taken off at once, UV and air quality asked
// only while on, the choice saved, the feed asked again.
func (mm mapModel) menuApplied(before mapModel, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	mm = mm.timeFrom("overlay", e)
	mm = mm.setTemp(e) // temperature switched: drawn at once from what is held (D-99), or taken off
	var temp tea.Cmd
	if (mm.layerOn(UVLayer, e) && !before.layerOn(UVLayer, e)) || (mm.layerOn(AirLayer, e) && !before.layerOn(AirLayer, e)) {
		mm, temp = mm.askTemp(e) // UV and air quality are asked only while on (D-137, D-139)
	}
	mm = mm.renderMap(e)
	mm, feed := mm.askFeed(e)
	focused, w := mm.followMenuFocus(e)
	w.Settings.Save, w.Settings.Led = true, [3]tea.Cmd{mm.mapWorkCmd(e), feed, temp} // the choice saved, first in the batch; radar is R's, not the menu's (D-94)
	return focused, nil, w, true
}

// mapWindowKey is a key that works on the window itself, with a map or
// without: the alerts, the Overlays menu and the body's scroll.
func (mm mapModel) mapWindowKey(act term.Action, e mapEnv) (mapModel, mapWant, bool) {
	var w mapWant
	switch act {
	case actMapAlerts: // D-63
		mm.mapPane.alertsOn = !mm.mapPane.alertsOn
		mm.mapPane.gen++
	case actMapOverlays: // D-65
		mm.mapPane.menuOn, mm.mapPane.menuAt = !mm.mapPane.menuOn, 0
		mm.mapPane.gen++
	case actMapScrollUp: // D-61: PgUp and PgDn scroll the window's body, the description first
		w.Scroll, w.ScrollSet = max(e.Scroll-max(e.rows()-1, 1), 0), true
	case actMapScrollDown:
		w.Scroll, w.ScrollSet = min(e.Scroll+max(e.rows()-1, 1), max(len(mm.windowLines(e))-e.rows(), 0)), true
	default:
		return mm, w, false
	}
	return mm, w, true
}

// moveMapView is a key that moves the view: a pan, a zoom, the next place or
// a region.
func (mm mapModel) moveMapView(act term.Action, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	size, m := mm.mapBodySize(e), mm.mapPane.m
	stepX, stepY := max(size.Cols/4, 1), max(size.Rows/4, 1) // a quarter of the view a press
	// ANY KEY TAKES THE EDGE'S CHIP AWAY (D-81); a press the same way again
	// is the one that crosses.
	armed := mm.mapPane.edgeShown
	mm.mapPane.edgeShown = false
	switch act {
	case actMapPanUp:
		mm = mm.panOrCross(0, -stepY, geo.North, armed, e)
	case actMapPanDown:
		mm = mm.panOrCross(0, stepY, geo.South, armed, e)
	case actMapPanLeft:
		mm = mm.panOrCross(-stepX, 0, geo.West, armed, e)
	case actMapPanRight:
		mm = mm.panOrCross(stepX, 0, geo.East, armed, e)
	case actMapZoomIn:
		mm.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(1) })
	case actMapZoomOut:
		mm.mapPane.call("ZoomBy", func() { _ = m.ZoomBy(-1) })
	case actMapPrev, actMapNext:
		nav := term.Action("nav-down")
		if act == actMapPrev {
			nav = "nav-up"
		}
		if mm.mapRole(e).placeKeys { // D-195: the Broadcaster's map stays on its station
			return mm, nil, mapWant{Nav: nav, Then: mapStepPlaceMoved}, true // the Dashboard moves the selection; the map follows it
		}
	default:
		if n := slices.Index(mapRegionActs, act); n >= 0 {
			if r, ok := geo.RegionNumbered(n + 1); ok {
				mm = mm.showRegion(r, e)
			}
		}
	}
	mm, cmd := mm.afterMove(e)
	return mm, cmd, mapWant{}, true
}

// placeMoved is the map after the Dashboard moved the selection from it
// (0.19.6 design MS-1's S5): it follows the place, the notes speak of it,
// and the view is drawn as after any move.
func (mm mapModel) placeMoved(e mapEnv) (mapModel, tea.Cmd) {
	return mm.followSelection(e).requestFeed().afterMove(e)
}

// afterMove is every move's end: the world held, the view marked moved and
// drawn, and the map's work and the settle tick asked.
func (mm mapModel) afterMove(e mapEnv) (mapModel, tea.Cmd) {
	mm = mm.holdWorld(e)
	mm, settle := mm.viewMoved(e) // marked moved before it is drawn: its view is not in until its settle tick asks (W14's instrument)
	mm = mm.renderMap(e)
	return mm, tea.Batch(mm.mapWorkCmd(e), settle) // the alerts are the settle tick's to ask: never on every key (D-66, W14's P-2)
}

// panOrCross pans by cells. Where the region's edge holds the map still, the
// first press shows a chip naming the region beyond (D-81), and a second
// press the same way while it shows moves on to it (D-77). An edge with
// nothing beyond it stays an edge.
func (mm mapModel) panOrCross(dx, dy int, dir geo.Direction, armed bool, e mapEnv) mapModel {
	m := mm.mapPane.m
	before, _ := m.Centre()
	mm.mapPane.call("PanCells", func() { _ = m.PanCells(dx, dy) })
	after, _ := m.Centre()
	if math.Abs(after.Lat-before.Lat) > 1e-9 || math.Abs(after.Lon-before.Lon) > 1e-9 || mm.mapMode(e) == modePropagation {
		return mm // moved, or at the world's edge in the Propagation mode: no region lies beyond
	}
	next, ok := geo.Neighbour(mm.mapPane.region.Name, dir)
	if !ok {
		return mm
	}
	if armed && mm.mapPane.edge == dir {
		return mm.showRegion(next, e)
	}
	mm.mapPane.edge, mm.mapPane.edgeShown = dir, true
	return mm
}

// showRegion binds the map to a region and shows the whole of it (D-77): the
// listener's way to every region the station's APIs cover, one at a time
// (D-28). The selected place is unchanged.
func (mm mapModel) showRegion(r geo.Region, e mapEnv) mapModel {
	m := mm.mapPane.m
	mm.mapPane.outside, mm.mapPane.region = "", r
	mm = mm.boundMap(e)
	lat, lon := r.Centre()
	least := regionFitZoom(r, mm.mapBodySize(e))
	mm.mapPane.call("Zoom", func() { _ = m.Zoom(least) })
	mm.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: lon, Lat: lat}) })
	return mm
}

// followSelection puts the map on the selected location (FR-1.2), held
// inside the region that location is in (FR-2.1). A location in no region
// draws nothing wider in its place: the window says so (FR-2.5).
func (mm mapModel) followSelection(e mapEnv) mapModel {
	loc, m := mm.mapPlace(e), mm.mapPane.m
	if loc == nil || m == nil {
		return mm
	}
	region, ok := geo.RegionOf(loc.Lat, loc.Lon)
	mm.mapPane.outside = ""
	if !ok {
		mm.mapPane.outside = loc.Label
	}
	if ok {
		mm.mapPane.region = region
	}
	if mm.outsideStated(e) {
		return mm
	}
	mm = mm.boundMap(e) // the region's, or in the Propagation mode none
	mm.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}) })
	return mm
}

// requestFeed marks newer data wanted: on opening, on new data, when the
// view or the place changes. askFeed asks for it.
func (mm mapModel) requestFeed() mapModel {
	mm.mapPane.feedGen++
	return mm
}

// feedKey is the place and view an ask made now would be for.
func (mm mapModel) feedKey(e mapEnv) feedKey {
	k := feedKey{view: mm.mapPane.viewGen}
	if loc := mm.mapPlace(e); loc != nil {
		k.place = snapshot.Key(refOf(*loc))
	}
	return k
}

// keptSeqs is the lanes' ask numbers, carried over a release so an answer
// to an ask made before it is never taken for a new one.
func keptSeqs(lanes [feedLanes]feedLane) [feedLanes]feedLane {
	var out [feedLanes]feedLane
	for i := range lanes {
		out[i].seq = lanes[i].seq
	}
	return out
}

// askFeed asks the app for the map's feed, each lane apart (D-268): the
// alerts and the other layers, so a slow layer never holds the alerts back.
func (mm mapModel) askFeed(e mapEnv) (mapModel, tea.Cmd) {
	if mm.mapHooks.MapFeed == nil || mm.mapPane.m == nil || e.Window != mapModalMap || mm.mapPlace(e) == nil {
		return mm, nil
	}
	cmds := make([]tea.Cmd, 0, feedLanes)
	for lane := range feedLanes { // the two lanes (P10-02)
		var cmd tea.Cmd
		mm, cmd = mm.askLane(lane, e)
		cmds = append(cmds, cmd)
	}
	return mm, tea.Batch(cmds...)
}

// askLane asks for one lane - ONE ASK IN FLIGHT (D-157). While one runs for
// this place and view, new data marks the lane wanted again, and one fresh ask
// follows its answer. One made stale - the place or the view has changed - is
// cancelled, and the new one asked at once. Asking anew on every snapshot
// would drop the answer being waited for, so a feed slower than the
// snapshots would never be drawn (W14's C-1).
func (mm mapModel) askLane(lane int, e mapEnv) (mapModel, tea.Cmd) {
	key, l := mm.feedKey(e), &mm.mapPane.lanes[lane]
	if l.busy && key == l.asked {
		l.again = true
		return mm, nil
	}
	if l.stop != nil {
		l.stop() // stale: its place or view has gone
	}
	l.seq++
	l.ctx, l.stop = context.WithCancel(context.Background())
	l.busy, l.again, l.asked = true, false, key
	return mm, mm.mapFeedCmd(lane, e)
}

// mapFeedCmd is a lane's ask in flight's command, off the UI goroutine (the
// zones the alerts name may be fetched), cancelled with it.
func (mm mapModel) mapFeedCmd(lane int, e mapEnv) tea.Cmd {
	feed, loc := mm.mapHooks.MapFeed, mm.mapPlace(e)
	if feed == nil || mm.mapPane.m == nil || e.Window != mapModalMap || loc == nil {
		return nil
	}
	l := mm.mapPane.lanes[lane]
	gen, seq, ask, place, workers := mm.mapPane.feedGen, l.seq, mm.mapAsk(e), *loc, mm.mapPane.workers
	asked := l.ctx
	if asked == nil {
		asked = context.Background()
	}
	ask.Place, ask.Part = &place, laneParts[lane]                      // the command's own copy: the model may move on while it runs
	return workers.cmd(asked, nil, func(ctx context.Context) tea.Msg { // a stale ask stops it too
		return mapFeedMsg{gen: gen, seq: seq, lane: lane, feed: feed(ctx, ask)}
	})
}

// alertsLanded reports whether the alerts' lane has drawn an answer for the
// place and view as they are now: until it has, the description says the
// alerts are loading, never that none is there (D-266).
func (mm mapModel) alertsLanded(e mapEnv) bool {
	if mm.mapHooks.MapFeed == nil {
		return true // no feed to wait for: the window draws no alerts at all
	}
	l := mm.mapPane.lanes[laneAlerts]
	return l.landed && l.landedFor == mm.feedKey(e)
}

// partOf is a lane's answer kept to its own part (D-268): the alert areas
// and their words from the alerts' lane, every other overlay from the
// layers', whatever an answer carries.
func partOf(f MapFeed, lane int) MapFeed {
	out := MapFeed{}
	if lane == laneAlerts {
		out = MapFeed{Notes: f.Notes, Undrawn: f.Undrawn, InMissing: f.InMissing, InView: f.InView, AlertsFailed: f.AlertsFailed}
	}
	for _, o := range f.Overlays { // the answer's overlays (P10-02)
		key, _, _ := strings.Cut(o.ID, "/")
		if (key == AlertLayer) != (lane == laneAlerts) {
			continue
		}
		out.Overlays = append(out.Overlays, o)
		if t, ok := f.Times[o.ID]; ok {
			if out.Times == nil {
				out.Times = map[string]TimedOverlay{}
			}
			out.Times[o.ID] = t
		}
	}
	return out
}

// mergedFeed is the lanes' newest answers as one feed: the alerts' areas and
// words, and the other layers' overlays (D-268).
func mergedFeed(lanes [feedLanes]feedLane) MapFeed {
	var out MapFeed
	if a := lanes[laneAlerts].feed; a != nil {
		out = *a
		out.Overlays = append([]tuimaps.Overlay(nil), a.Overlays...)
		out.Times = make(map[string]TimedOverlay, len(a.Times))
		for id, t := range a.Times {
			out.Times[id] = t
		}
	}
	if l := lanes[laneLayers].feed; l != nil {
		out.Overlays = append(out.Overlays, l.Overlays...)
		out.Notes = append(append([]string(nil), out.Notes...), l.Notes...)
		for id, t := range l.Times {
			if out.Times == nil {
				out.Times = map[string]TimedOverlay{}
			}
			out.Times[id] = t
		}
	}
	return out
}

// applyMapFeed sets the feed's overlays, takes off the ones it no longer
// has, keeps its notes, and draws.
func (mm mapModel) applyMapFeed(v mapFeedMsg, e mapEnv) (mapModel, tea.Cmd) {
	m := mm.mapPane.m
	if m == nil || v.lane < 0 || v.lane >= feedLanes || v.seq != mm.mapPane.lanes[v.lane].seq {
		return mm, nil // a cancelled ask's answer: the ask in flight is newer
	}
	l := &mm.mapPane.lanes[v.lane]
	l.busy = false
	if l.stop != nil {
		l.stop() // its context let go
		l.stop, l.ctx = nil, nil
	}
	if mm.feedKey(e) != l.asked { // the view moved since it was asked: its settle tick asks for the new one
		return mm.feedAgainCmd(v.lane, nil, e)
	}
	v.feed = mm.feedForLayers(partOf(v.feed, v.lane), e) // its own part; a layer switched off draws nothing (W1.11)
	l.applied, l.landed, l.landedFor, l.feed = v.gen, true, l.asked, &v.feed
	if v.lane == laneAlerts {
		mm = mm.timed("answered:feed", e)
	} else {
		mm = mm.timed("answered:layers", e)
	}
	mm = mm.refreshMapCost(e)
	merged := mergedFeed(mm.mapPane.lanes)
	mm.mapPane.feed = &merged
	mm = mm.setFeed(merged, e)
	return mm.feedAgainCmd(v.lane, mm.mapWorkCmd(e), e)
}

// feedAgainCmd is after a lane's answer: the one fresh ask new data wanted
// while it ran, if any (D-157), beside cmd.
func (mm mapModel) feedAgainCmd(lane int, cmd tea.Cmd, e mapEnv) (mapModel, tea.Cmd) {
	if !mm.mapPane.lanes[lane].again {
		return mm, cmd
	}
	mm, again := mm.askLane(lane, e)
	return mm, tea.Batch(cmd, again)
}

// setFeed sets the feed's overlays, each with its span in the mode (D-98),
// takes off the ones it no longer has, keeps its notes, and draws.
func (mm mapModel) setFeed(feed MapFeed, e mapEnv) mapModel {
	notes := append([]string(nil), feed.Notes...)
	refused := map[string]int{} // a layer's refusals, said once (U2-29)
	firstErr := map[string]string{}
	var order []string
	want := make([]tuimaps.Overlay, 0, len(feed.Overlays))
	for _, o := range feed.Overlays {
		if t, ok := feed.Times[o.ID]; ok {
			o.During = mm.spanFor(t, e)
		}
		want = append(want, o)
	}
	// One loop with radar's and temperature's (D-257): an unchanged overlay
	// is not handed in again (U1-28), and a refused one keeps the area drawn
	// before it (U2-14).
	given, _ := mm.reconcile(mm.mapPane.given, want, func(o tuimaps.Overlay, err error) {
		key, _, _ := strings.Cut(o.ID, "/")
		if refused[key] == 0 {
			order, firstErr[key] = append(order, key), err.Error()
		}
		refused[key]++
	})
	for _, key := range order { // ours to fix, never the listener's: the diagnostics', once a layer (U2-29, D-124)
		mm.problem(mm.layerLabel(key, e)+": "+strconv.Itoa(refused[key])+" not drawn - "+firstErr[key], e)
	}
	mm.mapPane.undrawn = mm.sayUndrawn(feed.Undrawn, e)
	mm.mapPane.given, mm.mapPane.notes, mm.mapPane.inMissing, mm.mapPane.inView = given, notes, feed.InMissing, feed.InView
	mm.mapPane.drawnSev = map[string]bool{}
	for _, o := range feed.Overlays {
		for _, f := range o.Features {
			if w := strings.ToLower(f.Severity.Word()); w != "" {
				mm.mapPane.drawnSev[w] = true
			}
		}
	}
	return mm.renderMap(e)
}

// sayUndrawn hands the diagnostics each undrawn area's words once - not again
// while the feed still carries them, however many alerts share them - and
// returns the words now carried (D-124, D-240).
func (mm mapModel) sayUndrawn(words []string, e mapEnv) map[string]bool {
	now := make(map[string]bool, len(words))
	for _, w := range words { // the feed's undrawn areas (P10-02)
		if !now[w] && !mm.mapPane.undrawn[w] {
			mm.problem(w, e)
		}
		now[w] = true
	}
	return now
}

// warningText is a library warning as the diagnostics keep it: its kind, what
// it is about, and how many times, since the library counts a warning rather
// than repeating it.
func warningText(w tuimaps.Warning) string {
	said := "Map warning: " + w.Kind.String()
	if s := w.Subject.String(); s != "" {
		said += " - " + s
	}
	if w.Count > 1 {
		said += " (" + strconv.Itoa(w.Count) + " times)"
	}
	return said
}

// mapFailedText is what the window says when the map cannot be drawn at
// all: that, and the way to try again (D-124).
const mapFailedText = "The map could not be drawn. Close it with esc and press g to open it again."

// problem hands the diagnostics something that went wrong that the listener
// cannot act on (D-124): never said to them.
func (mm mapModel) problem(p string, e mapEnv) {
	if mm.mapHooks.MapProblem != nil {
		mm.mapHooks.MapProblem(p)
	}
	if mm.mapHooks.MapFrame != nil { // the recorder keeps it between the frames it fell between
		mm.mapHooks.MapFrame(MapFrame{At: e.Now(), Problem: p})
	}
}

// layerLabel is a layer's name as the Overlays menu says it, or its key.
func (mm mapModel) layerLabel(key string, e mapEnv) string {
	for _, l := range e.Layers {
		if l.Key == key {
			return l.Label
		}
	}
	return key
}

// mapHelpRow is one row of Help's MAP group: several actions that are one
// idea, listed on one line so the group fits (D-61's ten keys in four rows).
// A row of several actions is written as pairs, each key beside its name,
// with no keys column: a screen reader reads a row in order, so keys listed
// apart from their names are matched by position, which it cannot do
// (AX-6).
type mapHelpRow struct{ keys, help string }

// panName is the pan row's words in Help, the Bands chart's own target keys
// beside them (D-256): "Pan (Bands: up/down target)" - by default the same
// arrows, in the chart's own scope - no row or group of its own, so Help
// still fits its window and its columns.
func panName(keys term.KeyMap) string {
	prev, next := keys[actChartPrev].Keys, keys[actChartNext].Keys
	if len(prev) == 0 || len(next) == 0 {
		return "Pan"
	}
	return "Pan (Bands: " + prev[0] + "/" + next[0] + " target)"
}

// shiftArrows writes the shifted arrows short, so the radar's row fits Help's
// column (D-86): as glyphs, or under --ascii as "S-left" and "S-right".
func shiftArrows(keys string, ascii bool) string {
	if ascii {
		return strings.NewReplacer("shift+left", "S-left", "shift+right", "S-right").Replace(keys)
	}
	return strings.NewReplacer("shift+left", "⇧←", "shift+right", "⇧→").Replace(keys)
}

// mapHelpRows are the MAP group's rows, read from the merged keys so a
// [keys] rebind shows: a row of one action's keys in the keys column, a row
// of several as key-name pairs (AX-6), and the hours' keys as the
// Propagation mode uses them (FR-1.14).
func mapHelpRows(keys term.KeyMap, ascii bool) []mapHelpRow {
	type named struct {
		act  term.Action
		name string
	}
	pairs := func(ns ...named) mapHelpRow {
		var out []string
		for _, n := range ns {
			if ks := keys[n.act].Keys; len(ks) > 0 {
				out = append(out, shiftArrows(strings.Join(ks, "/"), ascii)+" "+n.name) // one action's keys together: "+/= Zoom In"
			}
		}
		return mapHelpRow{help: strings.Join(out, ", ")}
	}
	rows := []mapHelpRow{
		{strings.Join(slices.Concat(keys[actMapPanUp].Keys, keys[actMapPanDown].Keys, keys[actMapPanLeft].Keys, keys[actMapPanRight].Keys), ", "), panName(keys)},
		pairs(named{actMapPrev, "Previous"}, named{actMapNext, "Next Location"}),
		pairs(named{actMapZoomIn, "Zoom In"}, named{actMapZoomOut, "Out"}, named{actMapScrollUp, "Scroll Up"}, named{actMapScrollDown, "Down"}),
		pairs(named{actMapAlerts, "Description"}, named{actMapOverlays, "Overlays"}, named{actMapChart, "Bands"}, named{actMapFreq, "Freq."}), // D-176, D-178
		pairs(named{actMapBack, "Back"}, named{actMapOn, "On (an hour in Propagation)"}, named{actMapNewest, "Now"}),                          // FR-1.14: a frame, a day, or in Propagation an hour
		pairs(named{actMapRadar, "Radar Mode"}, named{actMapPlay, "Play"}, named{actMapHighLow, "Forecast Hi-Lo"}),                            // D-94, D-97
		pairs(named{actMapProp, "Propagation"}, named{actMapLayer, "Layer"}, named{actMapRefresh, "Refresh"}, named{actMapFind, "Find"}),      // D-153, D-155, D-156; FR-1.10
	}
	// D-77: the region keys in two rows of three, each key beside its region.
	for _, half := range [][]int{{0, 1, 2}, {3, 4, 5}} {
		var ns []named
		for _, i := range half {
			ns = append(ns, named{mapRegionActs[i], mapRegionShort[i]})
		}
		r := pairs(ns...)
		if r.help != "" {
			r.help = "Region " + r.help
		}
		rows = append(rows, r)
	}
	return slices.DeleteFunc(rows, func(r mapHelpRow) bool { return r.keys == "" && r.help == "" }) // a key the UI mode does not bind is not listed (D-195)
}

// mapStatusText is what the status line says of the picture while it is not
// whole: loading, offline or coarser; nothing when it is whole (FR-3.4).
func (mm mapModel) mapStatusText(e mapEnv) string {
	switch {
	case mm.mapPane.offline:
		return mapOfflineText
	case mm.mapPane.status == tuimaps.Complete:
		return ""
	case mm.mapPane.pending:
		return mapLoadingText
	case !mm.alertsLanded(e):
		return mapAlertsLoadingText // the alerts are not in yet (D-266)
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
func (mm mapModel) viewMoved(e mapEnv) (mapModel, tea.Cmd) {
	mm.mapPane.viewGen++
	gen := mm.mapPane.viewGen
	return mm, tickAfter(e.Ticks, mapSettleDelay, mapViewSettledMsg{gen: gen})
}

// applyViewSettled asks the feed again once the view has stood still: the map
// draws every alert in view (D-66, D-76). A superseded tick is dropped; one
// that lands while a window is over the map waits for the map to be in
// front again, so the view's asks - the alerts and the Propagation mode's
// answers (P-1) - are never left waiting for another move.
func (mm mapModel) applyViewSettled(v mapViewSettledMsg, e mapEnv) (mapModel, tea.Cmd) {
	if v.gen != mm.mapPane.viewGen {
		return mm, nil
	}
	if e.Window != mapModalMap && e.Shown {
		return mm, tickAfter(e.Ticks, mapSettleDelay, v)
	}
	if e.Window != mapModalMap {
		return mm, nil // the map closed: its open asks for its view
	}
	mm.mapPane.viewAsked = v.gen
	mm, feed := mm.requestFeed().askFeed(e)
	mm, radar := mm.askRadar(e)
	mm, temp := mm.askTemp(e)
	return mm, tea.Batch(feed, radar, temp) // the view's alerts, its radar (W8) and its temperature (W10)
}

// SameOverlay reports whether an overlay is the one already handed in, so it
// is not handed in again (UAT-1 U1-28, UAT-2 U2-13). A MISSING VALUE IS NaN,
// AND NaN IS NEVER EQUAL TO ITSELF: compared as it is, a grid with any value
// missing - a wind grid's gusts said nowhere (D-136) - would be new at every
// answer, and blink. Two missing values here are the same.
//
// IT IS ASKED OF EVERY OVERLAY ON EVERY ANSWER, ON THE UI GOROUTINE (PF-10),
// so the features' rings are compared as points, not by reflection: their
// counts first, so a zone gained or lost is seen before any point is read.
// Its answer is reflect.DeepEqual's in every other respect.
func SameOverlay(a, b tuimaps.Overlay) bool {
	if (a.Grid == nil) != (b.Grid == nil) || !sameFeatures(a.Features, b.Features) {
		return false
	}
	a.Features, b.Features = nil, nil
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

// sameFeatures is reflect.DeepEqual over two overlays' features, the rings
// compared point by point once every count matches.
func sameFeatures(a, b []tuimaps.Feature) bool {
	if !sameShape(a, b) {
		return false
	}
	for i := range a { // the counts: features, rings, points (P10-02)
		if !sameShape(a[i].Rings, b[i].Rings) {
			return false
		}
		for j := range a[i].Rings {
			if !sameShape(a[i].Rings[j], b[i].Rings[j]) {
				return false
			}
		}
	}
	for i := range a { // the points, then the rest of each feature
		fa, fb := a[i], b[i]
		for j := range fa.Rings {
			if !sameMemory(fa.Rings[j], fb.Rings[j]) && !slices.Equal(fa.Rings[j], fb.Rings[j]) {
				return false
			}
		}
		fa.Rings, fb.Rings = nil, nil
		if !reflect.DeepEqual(fa, fb) {
			return false
		}
	}
	return true
}

// sameShape reports whether two slices are both nil or both not, and of one
// length: reflect.DeepEqual's first questions of a slice.
func sameShape[S ~[]E, E any](a, b S) bool {
	return len(a) == len(b) && (a == nil) == (b == nil)
}

// sameMemory reports whether two slices of one length are one slice, which
// reflect.DeepEqual counts equal without reading it.
func sameMemory[S ~[]E, E any](a, b S) bool {
	return len(a) > 0 && &a[0] == &b[0]
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
