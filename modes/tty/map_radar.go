package tty

// map_radar.go — the map's radar (0.18.0 W8, FR-5): a loop per radar box from
// the app, set and drawn in Update (D-41), the source named by a chip in the
// map's upper right (D-83), and the listener's playback keys (D-61).
//
// RADAR NEVER HOLDS UP THE ALERTS (W8.12): it is its own command, apart from
// the feed. IT IS SHOWN WHOLE (D-85): the loop is loaded before it is drawn,
// one request at a time, so it never blinks or stands on a single frame.

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// RadarLayer is radar's key: the app registers the layer under it, and its
// overlays' ids begin with it.
const RadarLayer = "radar"

// MapRadar is the radar the map draws: a loop per radar box, the source in
// use, and a note to say beside it - that MRMS's colours are approximate, or
// that no radar covers the region (D-83).
type MapRadar struct {
	Overlays []tuimaps.Overlay
	Source   string // "MRMS", "IEM", or "" where none covers
	Note     string
	// Ahead is the source of the loop's hours ahead (D-113): "HRRR", or ""
	// where the loop ends at now.
	Ahead string
	// Problems are what went wrong that the listener cannot act on: the
	// diagnostics', never said to them (D-124).
	Problems []string
	// AheadIn is when to ask again for the hours ahead still owed: soon while
	// HRRR's are on their way, later after it failed; zero when nothing is
	// owed (D-204).
	AheadIn time.Duration
}

// mapRadarMsg is a radar answer.
type mapRadarMsg struct {
	radar  MapRadar
	region string // the region it was asked for (D-130)
}

// radarRefresh is how long a loaded loop stands before new data asks again:
// the newest frame is five minutes apart at most (D-85).
const radarRefresh = 2 * time.Minute

// askRadar asks the app for the whole loop, off the UI goroutine - ONE
// REQUEST AT A TIME (D-85). A later ask made while one runs is kept and asked
// when the answer lands, never in its place: were every later ask to supersede
// the last, a six-second loop would never be drawn at all. BUT A REGION LEFT
// IS CANCELLED (D-130): its answer could not be drawn, and waiting for it
// would make `1` two cold loops, about fifteen seconds (UAT-2 U2-35).
func (mm mapModel) askRadar(e mapEnv) (mapModel, tea.Cmd) {
	radar := mm.mapHooks.MapRadar
	if radar == nil || mm.mapPane.m == nil || e.Window != mapModalMap {
		return mm, nil
	}
	region := mm.mapPane.region.Name
	if mm.mapPane.radarBusy {
		mm.mapPane.radarAgain = true
		if mm.mapPane.radarRegion != region && mm.mapPane.radarStop != nil {
			mm.mapPane.radarStop()
		}
		return mm, nil
	}
	mm.mapPane.radarBusy, mm.mapPane.radarAt, mm.mapPane.radarRegion = true, e.Now(), region
	if !mm.layerOn(RadarLayer, e) {
		return mm, func() tea.Msg { return mapRadarMsg{region: region} } // off: an empty answer takes the loops away, and the app is not asked
	}
	stopped, stop := context.WithCancel(context.Background())
	mm.mapPane.radarStop = stop
	ask, workers := mm.mapAsk(e), mm.mapPane.workers
	return mm, workers.cmd(stopped, nil, func(ctx context.Context) tea.Msg {
		return mapRadarMsg{radar: radar(ctx, ask), region: region}
	})
}

// refreshRadar asks again on new data only once the loop has stood for
// radarRefresh.
func (mm mapModel) refreshRadar(e mapEnv) (mapModel, tea.Cmd) {
	if e.Now().Sub(mm.mapPane.radarAt) < radarRefresh {
		return mm, nil
	}
	return mm.askRadar(e)
}

// applyMapRadar sets the radar's loops and takes off the boxes it no longer
// has. A loop handed in is drawn once the library has prepared it - the
// work's answer draws it - so the map is not redrawn in between, where the
// radar would blink out (D-85). A want kept while this ran is asked now.
func (mm mapModel) applyMapRadar(v mapRadarMsg, e mapEnv) (mapModel, tea.Cmd) {
	mm.mapPane.radarBusy = false
	if mm.mapPane.radarStop != nil {
		mm.mapPane.radarStop() // its context let go
		mm.mapPane.radarStop = nil
	}
	m := mm.mapPane.m
	if m == nil || e.Window != mapModalMap {
		return mm, nil
	}
	if v.region != mm.mapPane.region.Name { // a region left (D-130): nothing of it is drawn; the kept ask goes now
		mm.mapPane.radarAgain = false
		d, again := mm.askRadar(e)
		return d, again
	}
	mm = mm.timed("answered:radar", e)
	if !mm.layerOn(RadarLayer, e) {
		v.radar = MapRadar{}
	}
	given, set, refused := mm.replaceLoops(mm.mapPane.radarGiven, v.radar.Overlays) // a refusal SAID, never swallowed: unsaid, it reads as "loading" for ever (UAT-2 U2-5)
	switch {                                                                        // a refusal is ours, never the listener's to act on: the diagnostics' (D-124)
	case refused != nil && len(given) == 0:
		v.radar.Source, v.radar.Note = "", ""
		mm.problem("Radar: not drawn - "+refused.Error(), e)
	case refused != nil:
		mm.problem("Radar: not updated, the last loop kept - "+refused.Error(), e)
	}
	for _, p := range v.radar.Problems {
		mm.problem(p, e)
	}
	mm.mapPane.radarGiven, mm.mapPane.radarSource, mm.mapPane.radarNote, mm.mapPane.radarAhead = given, v.radar.Source, v.radar.Note, v.radar.Ahead
	mm = mm.showStep(e).retime(e) // the newest frame is Radar mode's now: the alerts' spans move with it (D-98)
	if !set || m.Pending() == 0 {
		mm = mm.renderMap(e) // nothing left to prepare: draw now; else the work's answer draws it, whole, and the picture on screen stands until then
	}
	cmds := []tea.Cmd{mm.mapWorkCmd(e)}
	if mm.mapPane.radarAgain {
		mm.mapPane.radarAgain = false
		var again tea.Cmd
		mm, again = mm.askRadar(e)
		cmds = append(cmds, again)
	}
	if wait, region := v.radar.AheadIn, v.region; wait > 0 { // the hours ahead still owed (D-204)
		cmds = append(cmds, tickAfter(e.Ticks, wait, mapRadarAgainMsg{region: region}))
	}
	return mm, tea.Batch(cmds...)
}

// replaceLoops hands in an answer's loops in place of the ones held, all or
// none (U2-59). The loops leaving go first, so the image budget holds the new
// ones rather than both - a zoom across the lower 48's wide line swaps one
// loop for up to four. Where a new box's loop is still refused, the new
// loops go and the held ones come back: a zoom never leaves the map without
// the radar it had. refused is the first refusal.
func (mm mapModel) replaceLoops(had map[string]tuimaps.Overlay, want []tuimaps.Overlay) (given map[string]tuimaps.Overlay, set bool, refused error) {
	m := mm.mapPane.m
	wanted := map[string]bool{}
	for _, o := range want {
		wanted[o.ID] = true
	}
	kept, gone := map[string]tuimaps.Overlay{}, []string{}
	for id, o := range had {
		if wanted[id] {
			kept[id] = o
		} else {
			gone = append(gone, id)
		}
	}
	sort.Strings(gone)
	for _, id := range gone {
		mm.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
	}
	newRefused := false
	given, set = mm.reconcile(kept, want, func(o tuimaps.Overlay, err error) {
		if refused == nil {
			refused = err
		}
		_, held := had[o.ID]
		newRefused = newRefused || !held
	})
	if !newRefused {
		return given, set, refused
	}
	for id := range given { // the swap is undone: the new boxes go, the old come back
		if _, held := had[id]; !held {
			mm.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
			delete(given, id)
		}
	}
	for _, id := range gone {
		var err error
		mm.mapPane.call("Set", func() { _, err = m.Set(had[id]) })
		if err == nil {
			given[id] = had[id]
		}
	}
	return given, true, refused
}

// mapRadarAgainMsg is the time to ask the radar again for the hours ahead an
// answer owed (D-204).
type mapRadarAgainMsg struct{ region string }

// applyRadarAgain asks the radar again for the hours ahead, while the map is
// open on the region they were owed for and the radar is on.
func (mm mapModel) applyRadarAgain(v mapRadarAgainMsg, e mapEnv) (mapModel, tea.Cmd) {
	if e.Window != mapModalMap || mm.mapPane.m == nil || v.region != mm.mapPane.region.Name || !mm.layerOn(RadarLayer, e) {
		return mm, nil
	}
	return mm.askRadar(e)
}

// radarChipText is the chip's words: the source in use, or nothing.
func (mm mapModel) radarChipText(e mapEnv) string {
	if mm.mapPane.radarSource == "" || !mm.layerOn(RadarLayer, e) {
		return ""
	}
	return " " + mm.radarFace(e.Opts.ASCII) + " "
}

// radarFace is the radar's source as its chips say it: MRMS≈, its colours
// read from its legend and so approximate (D-132), or IEM.
func (mm mapModel) radarFace(ascii bool) string {
	if mm.mapPane.radarSource != "MRMS" {
		return mm.mapPane.radarSource
	}
	if ascii {
		return "MRMS~"
	}
	return "MRMS≈"
}

// radarBadge is the radar's badge (D-92), three rows: RADAR DATA; the
// source's chip in the badge's colours; the frame's time in the listener's
// clock, STALE before it when the newest frame is old. It takes the
// library's top-row stamp over (go-tuiMaps D-87).
func (mm mapModel) radarBadge(e mapEnv) string {
	if mm.radarChipText(e) == "" {
		return ""
	}
	if mm.atForecast() { // the loop's hours ahead: a model's frames, said so (D-113)
		chip := chipFace(mm.aheadName()) // its ground and label alone, no brackets (D-174)
		return mapBadge(render.Tint("RADAR FCST", render.Tok(render.ModalTitle)), chip, mm.mapPane.radarBadgeTime)
	}
	chip := chipFace(mm.radarFace(e.Opts.ASCII))
	return mapBadge(render.Tint("RADAR DATA", render.Tok(render.ModalTitle)), chip, mm.mapPane.radarBadgeTime)
}

// aheadName is the hours ahead's source as its chip says it: HRRR, or
// O-METEO where they are Open-Meteo's (D-115), as temperature's chip says it.
func (mm mapModel) aheadName() string {
	if mm.mapPane.radarAhead == "Open-Meteo" {
		return "O-METEO"
	}
	return mm.mapPane.radarAhead
}

// newestObserved is the loop's newest observed frame, "right now" - never
// the far end of its hours ahead (W12.1): the newest's age, and
// STALE, read it (FR-5.4).
func newestObserved(st tuimaps.LoopState) time.Time {
	if st.Now.IsZero() {
		return st.Newest
	}
	return st.Now
}

// atForecast reports whether the loop's moment shown is one of its hours
// ahead (D-113).
func (mm mapModel) atForecast() bool {
	return mm.mapPane.m != nil && mm.mapPane.radarAhead != "" && mm.mapPane.m.Loop().Forecast
}

// mapBadgeWords are the badge's words for the mode (D-92, D-94): radar's, or
// Forecast mode's in its place. The library draws no stamp while the badge
// says the moment (go-tuiMaps D-87).
func (mm mapModel) mapBadgeWords(e mapEnv) string {
	switch mm.mapMode(e) {
	case modePropagation:
		return mm.propagationBadge()
	case modeForecast:
		if mm.mapHooks.MapRadar != nil {
			return mm.forecastBadge(e)
		}
	case modeRadar:
	}
	return mm.radarBadge(e)
}

// The playback keys (D-61): space plays and stops, "," and "." step a frame
// back and on, "n" returns to the newest.
const (
	actMapPlay   term.Action = "map.play"
	actMapBack   term.Action = "map.back"
	actMapOn     term.Action = "map.forward"
	actMapNewest term.Action = "map.newest"
)

// handlePlayback drives the library's loop: the host keeps no playback state
// of its own (go-tuiMaps L-1.13).
func (mm mapModel) handlePlayback(act term.Action, e mapEnv) (mapModel, bool) {
	m := mm.mapPane.m
	switch act {
	case actMapPlay:
		if m.Loop().Playing {
			mm.mapPane.call("Stop", func() { _ = m.Stop() })
		} else {
			mm.mapPane.call("Play", func() { _ = m.Play() })
		}
	case actMapBack:
		mm.mapPane.call("Step", func() { _ = m.Step(-1) })
	case actMapOn:
		mm.mapPane.call("Step", func() { _ = m.Step(1) })
	case actMapNewest:
		mm.mapPane.call("Reset", func() { _ = m.Reset() })
	default:
		return mm, false
	}
	return mm.renderMap(e), true
}

// applyPlayback turns the library's playback on at the speed chosen (D-361):
// the map opens stopped on the newest frame, and space plays (W8.9a).
func (mm mapModel) applyPlayback() mapModel {
	m := mm.mapPane.m
	if m == nil {
		return mm
	}
	mm.mapPane.call("SetPlayback", func() { _ = m.SetPlayback(tuimaps.PlaybackOn) })
	mm.mapPane.call("ShowStamp:false", func() { m.ShowStamp(false) }) // the badge and the timeline say the moment and its age (D-92)
	step := mm.mapPlayback
	mm.mapPane.call("SetPlaybackStep", func() { _ = m.SetPlaybackStep(step) })
	return mm
}

// radarStale is how old the newest frame may be before it is marked stale:
// twice the slower source's cadence (FR-5.4).
const radarStale = 10 * time.Minute

// radarStatus is the loop's line: the source, the moment shown and how old
// the newest frame is - stale marked, never hidden (FR-5.4) - where it is in
// the loop, and whether it plays.
func (mm mapModel) radarStatus(e mapEnv) string {
	m := mm.mapPane.m
	if m == nil || mm.mapPane.radarSource == "" || !mm.layerOn(RadarLayer, e) {
		switch {
		case !mm.layerOn(RadarLayer, e) || mm.mapHooks.MapRadar == nil:
			return ""
		case mm.mapPane.radarNote != "":
			return mm.mapPane.radarNote
		case mm.mapPane.radarBusy:
			return "Radar loading…" // preloaded: shown when the whole loop is in (D-85)
		}
		return ""
	}
	st := m.Loop()
	if st.Count == 0 {
		return "Radar " + mm.radarChip(e.Opts.ASCII) + " loading…"
	}
	age := e.Now().Sub(newestObserved(st))
	ago := strconv.Itoa(int(age.Minutes())) + " min ago"
	if age > radarStale {
		ago += ", stale"
	}
	state := "stopped"
	if st.Playing {
		state = "playing"
	}
	shown := e.Opts.Clock.Time(st.At.In(e.Now().Location()))
	parts := []string{"Radar " + mm.radarChip(e.Opts.ASCII) + " " + shown, "frame " + strconv.Itoa(st.Index+1) + " of " + strconv.Itoa(st.Count),
		state, "newest " + ago}
	if st.Forecast {
		parts = append(parts, "forecast")
	}
	return strings.Join(parts, " · ")
}

// radarRows is the timeline's rows under the map (D-86), held whenever the
// radar layer is on so the map's size never waits on the loop.
const radarRows = 3

// radarExtraRows are the other held rows under the map (D-103): the colour
// row, the picture's status, the box's bottom edge beside the timeline, and
// the blanks round the region row.
const radarExtraRows = 6

// radarLegendRow is the radar's colours under the map (D-89), lightest to
// heaviest, from the library's legend: a swatch a class, painted in the
// class's own colour, or its words where colour is off.
func (mm mapModel) radarLegendRow(width int) string {
	return mm.rainRow("RADAR LEGEND │ ", "radar", width)
}

// rainRow is radar's colours as a row under a head of its own.
func (mm mapModel) rainRow(head, preset string, width int) string {
	classes := mm.legendClasses(preset)
	lighter, heavier := "LIGHTER ", " HEAVIER"
	room := width - render.Width(head+lighter+heavier)
	if len(classes) == 0 || room < len(classes)*3 {
		return ""
	}
	each := room / len(classes)
	var row strings.Builder
	for _, c := range classes {
		label := strings.Repeat(" ", each-1)
		if !render.ColorOn() || !c.Drawn {
			label = render.PadTo(render.TruncateCells(c.Label, each-1), each-1)
		}
		row.WriteString(render.Swatch(label, c.Colour.R, c.Colour.G, c.Colour.B) + " ")
	}
	return head + lighter + strings.TrimRight(row.String(), " ") + heavier
}

// radarChip is the source in the badge's colours (D-83), for the status line
// (D-89).
func (mm mapModel) radarChip(ascii bool) string {
	return chipFace(mm.radarFace(ascii))
}

// radarBadgeTimeNow is the badge's time row: the moment shown in the
// listener's clock, STALE before it past radarStale (FR-5.4: never hidden).
func (mm mapModel) radarBadgeTimeNow(e mapEnv) string {
	m := mm.mapPane.m
	if m == nil || mm.radarChipText(e) == "" {
		return ""
	}
	st := m.Loop()
	if st.Count == 0 {
		return ""
	}
	when := e.Opts.Clock.Time(st.At.In(e.Now().Location()))
	if e.Now().Sub(newestObserved(st)) > radarStale {
		when = "STALE " + when
	}
	return when
}

// radarTimelineOn reports whether the timeline's rows are held: in both
// modes (D-94) - Radar mode's loop, or Forecast mode's steps - so the map's
// size never depends on the mode.
func (mm mapModel) radarTimelineOn() bool { return mm.mapHooks.MapRadar != nil }

// radarChipWords are the R chip's words: the mode the key switches (D-94).
func (mm mapModel) radarChipWords(e mapEnv) string {
	switch mm.mapMode(e) {
	case modeRadar:
		return "Radar On"
	case modeForecast, modePropagation:
	}
	return "Radar Off"
}

// radarTimeline is the loop as the listener sees it (D-86): the shown frame's
// time above its mark; the bar from the oldest frame to the last, the step
// keys at its ends, a tick at now; beneath, the end times, OBSERVED before
// now and FORECAST after it. Read from the library's loop in Update; the
// window keeps no playback state of its own (go-tuiMaps L-1.13).
func (mm mapModel) radarTimeline(width int, e mapEnv) []string {
	if mm.mapPane.m == nil || !mm.radarTimelineOn() {
		return nil
	}
	s, ok := mm.radarScrubber(e)
	if !ok {
		return []string{"", "", ""}
	}
	return mm.draw(s, width, e)
}

// radarScrubber is the loop's scrubber: where the frame shown is, where NOW
// is, and the words under them, all by time along the loop; false with no
// loop to draw.
func (mm mapModel) radarScrubber(e mapEnv) (scrubber, bool) {
	m := mm.mapPane.m
	if m == nil || mm.mapPane.radarSource == "" {
		return scrubber{}, false
	}
	st := m.Loop()
	if st.Count == 0 {
		return scrubber{}, false
	}
	s := scrubber{above: mm.clockAt(st.At, e)}
	// ONE AXIS, TIME (UAT-2 U2-33): the cursor and NOW are both placed by
	// time. The frames are five minutes apart observed and fifteen ahead, so a
	// cursor placed by frame number draws the newest observed frame in the
	// FORECAST half.
	if span := st.Newest.Sub(st.Oldest); span > 0 {
		s.cursor = min(max(float64(st.At.Sub(st.Oldest))/float64(span), 0), 1)
	}
	now := 1.0
	if span := st.Newest.Sub(st.Oldest); span > 0 && st.Now.Before(st.Newest) {
		now = float64(st.Now.Sub(st.Oldest)) / float64(span)
	}
	s.below = []scrubLabel{{mm.clockAt(st.Oldest, e), 0, alignLeft}}
	if now < 1 {
		s.ticks = []float64{now}
		s.below = append(s.below, scrubLabel{mm.clockAt(st.Newest, e), 1, alignRight}, scrubLabel{"NOW", now, alignCentre})
	} else {
		s.below = append(s.below, scrubLabel{"NOW · " + mm.clockAt(st.Newest, e), 1, alignRight}) // no forecast: the loop ends at now
	}
	s.below = append(s.below, scrubLabel{"OBSERVED", now / 2, alignCentre})
	if now < 1 {
		s.below = append(s.below, scrubLabel{"FORECAST", (now + 1) / 2, alignCentre})
	}
	return s, true
}

// clockAt is a moment in the listener's clock and zone.
func (mm mapModel) clockAt(t time.Time, e mapEnv) string {
	return e.Opts.Clock.Time(t.In(e.Now().Location()))
}

// firstKey is an action's first bound key, as the listener's [keys] set it.
func (mm mapModel) firstKey(act term.Action) string {
	if keys := mm.mapKeys[act].Keys; len(keys) > 0 {
		return keys[0]
	}
	return ""
}

// placer lays words on a line at columns, never over one another: a word that
// would overlap one already placed is left out.
type placer struct{ cells []rune }

func newPlacer(width int) *placer { return &placer{cells: []rune(strings.Repeat(" ", max(width, 0)))} }

func (p *placer) put(s string, from int) {
	r := []rune(s)
	if from < 0 || from+len(r) > len(p.cells) {
		return
	}
	for i := from - 1; i <= from+len(r); i++ { // a space either side
		if i >= 0 && i < len(p.cells) && p.cells[i] != ' ' {
			return
		}
	}
	copy(p.cells[from:], r)
}

func (p *placer) left(s string, col int)   { p.put(s, col) }
func (p *placer) right(s string, col int)  { p.put(s, col-len([]rune(s))+1) }
func (p *placer) centre(s string, col int) { p.put(s, col-len([]rune(s))/2) }

func (p *placer) String() string { return strings.TrimRight(string(p.cells), " ") }
