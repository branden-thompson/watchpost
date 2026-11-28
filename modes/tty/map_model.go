package tty

// map_model.go — the map's model (0.19.6 design MS-1): the state the map
// window owns, a field of the Dashboard, and the values it is called with
// and answers in (0.19.6 D-77, D-79, D-81).

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// mapModel is the map's own state, the Dashboard's mapModel field (0.19.6
// D-81): the Dashboard reads it as d.mapModel.mapPane and the rest, and
// calls its methods with an Env it builds (observer_map.go).
type mapModel struct {
	mapPane mapPane
	// mapCloses counts the map's closings: a release carries the count it was
	// scheduled at, so only the newest close's lets the map go (0.18.0 D-221).
	mapCloses uint64
	// mapReleaseAfter is how long the map stays closed before it is let go.
	mapReleaseAfter time.Duration
	// mapChoices are the map's Settings, built from the Config by their own
	// keyed literal (0.19.5 D-14, A-16). Embedded, so mm.mapDesc and the rest
	// read as one of the model's own fields.
	mapChoices
	// mapCost is the last estimate of a refresh's cost, asked in Update and read
	// by the frame (0.18.0 batch 9).
	mapCost   MapCost
	mapKeys   term.KeyMap
	chartKeys term.KeyMap // the Bands chart window's own scope (0.19.0 D-256)
	// ackSeen is the acknowledgement's wording last seen (0.19.0 A-9, W2.5),
	// from Config.PropagationAck at launch and the map's own once it is
	// closed (0.19.6 design MS-1).
	ackSeen int
	// mapHooks is the map's hooks, held once (0.19.6 A-31, D-77): copies of
	// the Dashboard share it, so a hook is changed only by replacing the
	// pointer, never written through it.
	mapHooks *Hooks
}

// mapModal is which of the map's windows is open on top (0.19.6 design MS-2,
// D-79): the Dashboard's modal as the map sees it, mapped in one switch each
// way (observer_map.go).
type mapModal uint8

// The map's windows; mapModalNone is any window that is not the map's.
const (
	mapModalNone mapModal = iota
	mapModalMap
	mapModalPropAck
	mapModalPropChart
	mapModalPropFind
	mapModalPropFreq
)

// mapPick is what the selection is resolved from (0.19.6 design MS-2): the
// ingredients, not the answer, so the place is built only where it is asked
// for, as the Dashboard builds it.
type mapPick struct {
	Snap, Recent *snapshot.Snapshot
	Lookup       *snapshot.LocationRef
	Selected     int
}

// mapStation is the station the map draws its service area for.
type mapStation struct {
	Transmitter *snapshot.LocationRef
	RadiusMi    int
}

// mapEnv is what the map reads of the Dashboard, built for each call by
// Dashboard.mapEnv (0.19.6 design MS-2, A-31, D-21): plain values, no size
// derived from them, no hook, and no func field but the Dashboard's own
// clock and tick hook.
type mapEnv struct {
	Window       mapModal
	Shown        bool // the map window is open, on top or under another
	Width        int
	Height       int
	Opts         render.Opts
	ReduceMotion bool
	NoColor      bool
	Surface      Surface
	Pick         mapPick
	Station      mapStation
	Layers       []MapLayer
	Keys         term.KeyMap
	Scroll       int // the open window's scroll, lent: mapWant.Scroll returns it
	Now          func() time.Time
	Ticks        func(time.Duration, tea.Msg) tea.Cmd
}

// rows is the open window's body budget, derived as the Dashboard derives
// it (0.19.6 A-31): no size is lent.
func (e mapEnv) rows() int { return modalRows(e.Height, e.Window == mapModalMap) }

// ceiling is the Map window's body budget.
func (e mapEnv) ceiling() int { return mapCeilingAt(e.Height) }

// cols is the Map window's width.
func (e mapEnv) cols() int { return mapWindowColsAt(e.Width) }

// modalWidth is the open map window's width.
func (e mapEnv) modalWidth() int { return mapWindowWidth(e.Window, e.Width) }

// mapWindowWidth is a map window's width at a terminal's width: the map's
// share of it, or a Propagation window's own.
func mapWindowWidth(w mapModal, width int) int {
	switch w {
	case mapModalMap:
		return mapWindowColsAt(width)
	case mapModalPropAck:
		return propAckWidth
	case mapModalPropChart:
		return propChartWidth
	case mapModalPropFreq:
		return propFreqWidth
	case mapModalPropFind:
		return propFindWidth
	}
	return 0
}

// windowLines is the open map window's whole body, wrapped exactly as the
// window draws it: what its scroll is bounded by (view.go's modalLines).
func (mm mapModel) windowLines(e mapEnv) []string {
	o := e.Opts
	var raw []string
	w := e.modalWidth()
	switch e.Window {
	case mapModalMap:
		raw, o.Width = mm.mapBodyLines(e), w // measured at the width the map window is drawn at, not the dashboard's
	case mapModalPropAck:
		raw = mm.propAckLines(o)
	case mapModalPropChart:
		raw = mm.propChartLines(o, e)
	case mapModalPropFreq:
		raw = mm.propFreqLines(o)
	case mapModalPropFind:
		raw = mm.propFindLines(o)
	default:
		return nil // not the map's window
	}
	out, _ := wrapModalRows(raw, min(o.Width, w), e.Window == mapModalMap, e.rows())
	return out
}

// mapSettingsView is what the map's Settings rows read of the Settings
// window (0.19.6 design MS-4): its focus, the layers row's cursor, the
// focused chip's blink as of the call, and whether each of the map's three
// groups is shown, read from the row that opens it.
//
// A FIELD FOR EACH ROW THE MAP ASKS ABOUT, not a mask of all its rows: the
// window's width builds this for every group of every tab, and asking all 31
// rows costs the Settings window's redraw 6 % for three answers (0.19.6
// D-87). A row the map has not asked about has no field to read.
type mapSettingsView struct {
	Focus            setupRowID
	LayerAt          int
	Flash            pickerFlash // the focused row's chip blinking now, or flashNone
	MapsOnShown      bool        // rowMapsOn, which opens the MAP group
	LayersShown      bool        // rowMapLayers, the MAP - LAYERS group's one row
	DetailLevelShown bool        // rowMapDetailLevel, which opens the MAP - DETAIL group
}

// flashFor is row id's chip blinking: only the focused row's can be.
func (v mapSettingsView) flashFor(id setupRowID) pickerFlash {
	if id != v.Focus {
		return flashNone
	}
	return v.Flash
}

// mapStep is what the map does once the root has acted on its mapWant
// (0.19.6 design MS-1's split sites, D-79): each a direct call on the model.
type mapStep uint8

// The map's continuations.
const (
	mapStepNone mapStep = iota
	mapStepRender
	mapStepSetPropRender
	mapStepFoundRecentre
	mapStepAckAsk
	mapStepPlaceMoved
	mapStepMenuFocus
)

// mapWant is what the map asks of the Dashboard (0.19.6 design MS-2): every
// field a plain value, applied by Dashboard.applyMapWant after the call, in
// the order the map asked for them.
type mapWant struct {
	Close     bool        // the window on top closed
	Open      mapModal    // a window opened over the map
	Nav       term.Action // a key the Dashboard's selection walks (S5)
	ModalNav  term.Action // a key the open window's scroll walks (S6)
	Scroll    int         // the open window's scroll, when ScrollSet (S7)
	ScrollSet bool
	Then      mapStep // the map's own step once the rest is done
	Settings  mapSettingsWant
}

// mapSettingsWant is what the map asks of the Settings window (0.19.6
// design MS-4): the save's flag, a touch, the layers row's cursor, a note
// beside a row, and the save itself, first or last in the call's batch.
type mapSettingsWant struct {
	Dirty      bool // a display preference changed and is not yet written
	Touched    bool // uiTouched: dirty, and the window settled
	Settled    bool
	LayerAt    int
	LayerAtSet bool
	Note       string
	NoteRow    setupRowID
	NoteTone   noticeTone
	NoteSet    bool
	Save       bool       // the save runs, its command first in the batch
	SaveLast   bool       // and last instead
	Led        [3]tea.Cmd // the call's commands the save leads, in the one batch with it
}

// wantTouched is the map after a Setting changed: the Settings window marks
// it to save and settles (setup_ui.go's uiTouched).
func (mm mapModel) wantTouched() (mapModel, mapWant) {
	return mm, mapWant{Settings: mapSettingsWant{Touched: true}}
}

// wantDirty is the map after a choice the UI's save writes changed: the
// Settings window marks it to save (setup.uiDirty).
func (mm mapModel) wantDirty() (mapModel, mapWant) {
	return mm, mapWant{Settings: mapSettingsWant{Dirty: true}}
}

// and is w with v's asks after it: flags add, and a later ask of a kind
// replaces an earlier one, as a later write did.
func (w mapWant) and(v mapWant) mapWant {
	w.Close = w.Close || v.Close
	if v.Open != mapModalNone {
		w.Open = v.Open
	}
	if v.Nav != "" {
		w.Nav = v.Nav
	}
	if v.ModalNav != "" {
		w.ModalNav = v.ModalNav
	}
	if v.ScrollSet {
		w.Scroll, w.ScrollSet = v.Scroll, true
	}
	if v.Then != mapStepNone {
		w.Then = v.Then
	}
	s, t := &w.Settings, v.Settings
	s.Dirty, s.Touched, s.Settled = s.Dirty || t.Dirty, s.Touched || t.Touched, s.Settled || t.Settled
	if t.LayerAtSet {
		s.LayerAt, s.LayerAtSet = t.LayerAt, true
	}
	if t.NoteSet {
		s.Note, s.NoteRow, s.NoteTone, s.NoteSet = t.Note, t.NoteRow, t.NoteTone, true
	}
	if t.Save {
		s.Save, s.SaveLast, s.Led = true, t.SaveLast, t.Led
	}
	return w
}

// lent is e with the scroll w returns, for what the map reads next in the
// same call (0.19.6 design MS-1's S7: lent, and returned).
func (e mapEnv) lent(w mapWant) mapEnv {
	if w.ScrollSet {
		e.Scroll = w.Scroll
	}
	return e
}

// press is a key in one of the map's windows (0.19.6 design MS-6): the map's
// answer, what it asks of the Dashboard, and whether the key was the map's.
func (mm mapModel) press(w mapModal, key tea.KeyPressMsg, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	var cmd tea.Cmd
	var want mapWant
	switch w {
	case mapModalMap:
		return mm.onMapKey(key, e)
	case mapModalPropAck:
		mm, cmd, want = mm.onPropAckKey(key, e)
	case mapModalPropChart:
		mm, cmd, want = mm.onPropChartKey(key)
	case mapModalPropFind:
		mm, cmd, want = mm.onPropFindKey(key, e)
	case mapModalPropFreq:
		mm, cmd, want = mm.onPropFreqKey(key)
	default:
		return mm, nil, mapWant{}, false
	}
	return mm, cmd, want, true
}

// continueAfter is the map's own step once the Dashboard has done what the
// map asked (0.19.6 design MS-1's split sites): each a direct call.
func (mm mapModel) continueAfter(s mapStep, e mapEnv) (mapModel, tea.Cmd, mapWant) {
	var cmd tea.Cmd
	switch s {
	case mapStepRender:
		mm = mm.renderMap(e)
	case mapStepSetPropRender:
		mm = mm.setProp(e).renderMap(e)
	case mapStepFoundRecentre:
		mm, cmd = mm.foundRecentre(e)
	case mapStepAckAsk:
		mm, cmd = mm.ackAsked(e)
	case mapStepPlaceMoved:
		mm, cmd = mm.placeMoved(e)
	case mapStepMenuFocus:
		var w mapWant
		mm, w = mm.followMenuFocus(e)
		return mm, nil, w
	}
	return mm, cmd, mapWant{}
}

// answer is the map's handling of its own messages - its feeds, its clock,
// the Propagation mode's update and answers - each set and drawn in Update
// (D-41); false for a message that is none of them (routeMap).
func (mm mapModel) answer(msg tea.Msg, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	var cmd tea.Cmd
	var w mapWant
	switch v := msg.(type) {
	case mapClearedMsg:
		mm, w = mm.applyMapCleared(v, e) // 0.18.0 W3.8: Settings says what went
	case mapFeedMsg:
		mm, cmd = mm.applyMapFeed(v, e) // the alerts, set and drawn in Update (D-41)
	case mapRadarMsg:
		mm, cmd = mm.applyMapRadar(v, e) // W8: the radar, set and drawn in Update (D-41)
	case mapRadarAgainMsg:
		mm, cmd = mm.applyRadarAgain(v, e) // D-204: the hours ahead owed
	case mapTempMsg:
		mm, cmd = mm.applyMapTemp(v, e) // W10: the temperature, set and drawn in Update (D-41)
	case forecastTickMsg:
		mm, cmd = mm.applyForecastTick(v, e) // D-94: Forecast mode's playback
	case propChartMsg, propReachMsg, propCentreMsg:
		mm = mm.applyPropAnswer(v, e) // W5: the chart, the reach, the centre
	case propAheadMsg:
		mm = mm.applyPropAhead(v, e)
	case propFoundMsg:
		mm, cmd, w = mm.applyPropFound(v, e) // FR-1.10: a find's answer, kept in Update
	case propUpdatedMsg:
		mm, cmd = mm.applyPropUpdated(v, e) // W4.1: the Propagation mode's update, kept in Update
	case propTickMsg:
		mm, cmd = mm.applyPropTick(v, e) // FR-4.10: its refresh
	case mapWorkedMsg:
		mm, cmd = mm.applyMapWorked(v, e) // what a Work command landed is drawn here, in Update (D-41)
	case mapTickMsg:
		mm = mm.applyMapTick(v, e) // 0.18.0 W2.2: the library asked to be drawn now
	case mapReleaseMsg:
		mm = mm.applyMapRelease(v, e) // D-221: the map closed a while is let go
	case mapViewSettledMsg:
		mm, cmd = mm.applyViewSettled(v, e) // 0.18.0 D-66: the view stood still - its alerts are asked
	default:
		return mm, nil, mapWant{}, false
	}
	return mm, cmd, w, true
}
