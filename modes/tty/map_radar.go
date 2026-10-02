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
// when the answer lands, never in its place: every later ask superseding the
// last is how a six-second loop was never drawn at all. BUT A REGION LEFT IS
// CANCELLED (D-130): its answer could not be drawn, and waiting for it made
// `1` two cold loops, about fifteen seconds (UAT-2 U2-35).
func (d Dashboard) askRadar() (Dashboard, tea.Cmd) {
	radar := d.cfg.MapRadar
	if radar == nil || d.mapPane.m == nil || d.modal != modalMap {
		return d, nil
	}
	region := d.mapPane.region.Name
	if d.mapPane.radarBusy {
		d.mapPane.radarAgain = true
		if d.mapPane.radarRegion != region && d.mapPane.radarStop != nil {
			d.mapPane.radarStop()
		}
		return d, nil
	}
	d.mapPane.radarBusy, d.mapPane.radarAt, d.mapPane.radarRegion = true, d.now(), region
	if !d.layerOn(RadarLayer) {
		return d, func() tea.Msg { return mapRadarMsg{region: region} } // off: an empty answer takes the loops away, and the app is not asked
	}
	stopped, stop := context.WithCancel(context.Background())
	d.mapPane.radarStop = stop
	ask, workers := d.mapAsk(), d.mapPane.workers
	return d, workers.cmd(stopped, nil, func(ctx context.Context) tea.Msg {
		return mapRadarMsg{radar: radar(ctx, ask), region: region}
	})
}

// refreshRadar asks again on new data only once the loop has stood for
// radarRefresh.
func (d Dashboard) refreshRadar() (Dashboard, tea.Cmd) {
	if d.now().Sub(d.mapPane.radarAt) < radarRefresh {
		return d, nil
	}
	return d.askRadar()
}

// applyMapRadar sets the radar's loops and takes off the boxes it no longer
// has. A loop handed in is drawn once the library has prepared it - the
// work's answer draws it - so the map is not redrawn in between, where the
// radar would blink out (D-85). A want kept while this ran is asked now.
func (d Dashboard) applyMapRadar(v mapRadarMsg) (tea.Model, tea.Cmd) {
	d.mapPane.radarBusy = false
	if d.mapPane.radarStop != nil {
		d.mapPane.radarStop() // its context let go
		d.mapPane.radarStop = nil
	}
	m := d.mapPane.m
	if m == nil || d.modal != modalMap {
		return d, nil
	}
	if v.region != d.mapPane.region.Name { // a region left (D-130): nothing of it is drawn; the kept ask goes now
		d.mapPane.radarAgain = false
		d, again := d.askRadar()
		return d, again
	}
	d = d.timed("answered:radar")
	if !d.layerOn(RadarLayer) {
		v.radar = MapRadar{}
	}
	var refused error
	given, set, removed := d.reconcile(d.mapPane.radarGiven, v.radar.Overlays, func(_ tuimaps.Overlay, err error) {
		refused = err // SAID, never swallowed: a refused loop read as "loading" for ever (UAT-2 U2-5)
	})
	switch { // a refusal is ours, never the listener's to act on: the diagnostics' (D-124)
	case refused != nil && len(given) == 0:
		v.radar.Source, v.radar.Note = "", ""
		d.problem("Radar: not drawn - " + refused.Error())
	case refused != nil:
		d.problem("Radar: not updated, the last loop kept - " + refused.Error())
	}
	for _, p := range v.radar.Problems {
		d.problem(p)
	}
	d.mapPane.radarGiven, d.mapPane.radarSource, d.mapPane.radarNote, d.mapPane.radarAhead = given, v.radar.Source, v.radar.Note, v.radar.Ahead
	d = d.showStep().retime() // the newest frame is Radar mode's now: the alerts' spans move with it (D-98)
	if removed || !set || m.Pending() == 0 {
		d = d.renderMap() // nothing left to prepare: draw now; else the work's answer draws it, whole
	}
	cmds := []tea.Cmd{d.mapWorkCmd()}
	if d.mapPane.radarAgain {
		d.mapPane.radarAgain = false
		var again tea.Cmd
		d, again = d.askRadar()
		cmds = append(cmds, again)
	}
	if wait, region := v.radar.AheadIn, v.region; wait > 0 { // the hours ahead still owed (D-204)
		cmds = append(cmds, tea.Tick(wait, func(time.Time) tea.Msg { return mapRadarAgainMsg{region: region} }))
	}
	return d, tea.Batch(cmds...)
}

// mapRadarAgainMsg is the time to ask the radar again for the hours ahead an
// answer owed (D-204).
type mapRadarAgainMsg struct{ region string }

// applyRadarAgain asks the radar again for the hours ahead, while the map is
// open on the region they were owed for and the radar is on.
func (d Dashboard) applyRadarAgain(v mapRadarAgainMsg) (tea.Model, tea.Cmd) {
	if d.modal != modalMap || d.mapPane.m == nil || v.region != d.mapPane.region.Name || !d.layerOn(RadarLayer) {
		return d, nil
	}
	return d.askRadar()
}

// radarChipText is the chip's words: the source in use, or nothing.
func (d Dashboard) radarChipText() string {
	if d.mapPane.radarSource == "" || !d.layerOn(RadarLayer) {
		return ""
	}
	return " " + d.radarFace() + " "
}

// radarFace is the radar's source as its chips say it: MRMS≈, its colours
// read from its legend and so approximate (D-132), or IEM.
func (d Dashboard) radarFace() string {
	if d.mapPane.radarSource != "MRMS" {
		return d.mapPane.radarSource
	}
	if d.cfg.ASCII {
		return "MRMS~"
	}
	return "MRMS≈"
}

// radarBadge is the radar's badge (D-92), three rows: RADAR DATA; the
// source's chip in the badge's colours; the frame's time in the listener's
// clock, STALE before it when the newest frame is old. It takes the
// library's top-row stamp over (go-tuiMaps D-87).
func (d Dashboard) radarBadge() string {
	if d.radarChipText() == "" {
		return ""
	}
	if d.atForecast() { // the loop's hours ahead: a model's frames, said so (D-113)
		chip := chipFace(d.aheadName()) // its ground and label alone, no brackets (D-174)
		return mapBadge(render.Tint("RADAR FCST", render.Tok(render.ModalTitle)), chip, d.mapPane.radarBadgeTime)
	}
	chip := chipFace(d.radarFace())
	return mapBadge(render.Tint("RADAR DATA", render.Tok(render.ModalTitle)), chip, d.mapPane.radarBadgeTime)
}

// aheadName is the hours ahead's source as its chip says it: HRRR, or
// O-METEO where they are Open-Meteo's (D-115), as temperature's chip says it.
func (d Dashboard) aheadName() string {
	if d.mapPane.radarAhead == "Open-Meteo" {
		return "O-METEO"
	}
	return d.mapPane.radarAhead
}

// newestObserved is the loop's newest observed frame, "right now" - never
// the far end of its hours ahead (W12.1's defect): the newest's age, and
// STALE, read it (FR-5.4).
func newestObserved(st tuimaps.LoopState) time.Time {
	if st.Now.IsZero() {
		return st.Newest
	}
	return st.Now
}

// atForecast reports whether the loop's moment shown is one of its hours
// ahead (D-113).
func (d Dashboard) atForecast() bool {
	return d.mapPane.m != nil && d.mapPane.radarAhead != "" && d.mapPane.m.Loop().Forecast
}

// mapBadgeWords are the badge's words for the mode (D-92, D-94): radar's, or
// Forecast mode's in its place. The library draws no stamp while the badge
// says the moment (go-tuiMaps D-87).
func (d Dashboard) mapBadgeWords() string {
	if !d.radarMode() && d.cfg.MapRadar != nil {
		return d.forecastBadge()
	}
	return d.radarBadge()
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
func (d Dashboard) handlePlayback(act term.Action) (Dashboard, bool) {
	m := d.mapPane.m
	switch act {
	case actMapPlay:
		if m.Loop().Playing {
			d.mapPane.call("Stop", func() { _ = m.Stop() })
		} else {
			d.mapPane.call("Play", func() { _ = m.Play() })
		}
	case actMapBack:
		d.mapPane.call("Step", func() { _ = m.Step(-1) })
	case actMapOn:
		d.mapPane.call("Step", func() { _ = m.Step(1) })
	case actMapNewest:
		d.mapPane.call("Reset", func() { _ = m.Reset() })
	default:
		return d, false
	}
	return d.renderMap(), true
}

// radarFrameEvery is the slow rate, one frame a second, the default (FR-5.9,
// D-52); the motion Setting that offers off and normal joins with W8.9.
const radarFrameEvery = time.Second

// applyPlayback turns the library's playback on at the slow rate: the map
// opens stopped on the newest frame, and space plays (W8.9a).
func (d Dashboard) applyPlayback() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	d.mapPane.call("SetPlayback", func() { _ = m.SetPlayback(tuimaps.PlaybackOn) })
	d.mapPane.call("ShowStamp:false", func() { m.ShowStamp(false) }) // the badge and the timeline say the moment and its age (D-92)
	d.mapPane.call("SetPlaybackStep", func() { _ = m.SetPlaybackStep(radarFrameEvery) })
	return d
}

// radarStale is how old the newest frame may be before it is marked stale:
// twice the slower source's cadence (FR-5.4).
const radarStale = 10 * time.Minute

// radarStatus is the loop's line: the source, the moment shown and how old
// the newest frame is - stale marked, never hidden (FR-5.4) - where it is in
// the loop, and whether it plays.
func (d Dashboard) radarStatus() string {
	m := d.mapPane.m
	if m == nil || d.mapPane.radarSource == "" || !d.layerOn(RadarLayer) {
		switch {
		case !d.layerOn(RadarLayer) || d.cfg.MapRadar == nil:
			return ""
		case d.mapPane.radarNote != "":
			return d.mapPane.radarNote
		case d.mapPane.radarBusy:
			return "Radar loading…" // preloaded: shown when the whole loop is in (D-85)
		}
		return ""
	}
	st := m.Loop()
	if st.Count == 0 {
		return "Radar " + d.radarChip() + " loading…"
	}
	age := d.now().Sub(newestObserved(st))
	ago := strconv.Itoa(int(age.Minutes())) + " min ago"
	if age > radarStale {
		ago += ", stale"
	}
	state := "stopped"
	if st.Playing {
		state = "playing"
	}
	shown := d.clockFmt.Time(st.At.In(d.now().Location()))
	parts := []string{"Radar " + d.radarChip() + " " + shown, "frame " + strconv.Itoa(st.Index+1) + " of " + strconv.Itoa(st.Count),
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
func (d Dashboard) radarLegendRow(width int) string {
	return d.rainRow("RADAR LEGEND │ ", "radar", width)
}

// rainRow is radar's colours as a row under a head of its own.
func (d Dashboard) rainRow(head, preset string, width int) string {
	var classes []tuimaps.Class
	for _, e := range d.mapPane.legend {
		if e.Preset == preset {
			classes = e.Classes
			break
		}
	}
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
func (d Dashboard) radarChip() string {
	return chipFace(d.radarFace())
}

// radarBadgeTimeNow is the badge's time row: the moment shown in the
// listener's clock, STALE before it past radarStale (FR-5.4: never hidden).
func (d Dashboard) radarBadgeTimeNow() string {
	m := d.mapPane.m
	if m == nil || d.radarChipText() == "" {
		return ""
	}
	st := m.Loop()
	if st.Count == 0 {
		return ""
	}
	when := d.clockFmt.Time(st.At.In(d.now().Location()))
	if d.now().Sub(newestObserved(st)) > radarStale {
		when = "STALE " + when
	}
	return when
}

// radarTimelineOn reports whether the timeline's rows are held: in both
// modes (D-94) - Radar mode's loop, or Forecast mode's steps - so the map's
// size never depends on the mode.
func (d Dashboard) radarTimelineOn() bool { return d.cfg.MapRadar != nil }

// radarChipWords are the R chip's words: the mode the key switches (D-94).
func (d Dashboard) radarChipWords() string {
	if d.radarMode() {
		return "Radar On"
	}
	return "Radar Off"
}

// radarTimeline is the loop as the listener sees it (D-86): the shown frame's
// time above its mark; the bar from the oldest frame to the last, the step
// keys at its ends, a tick at now; beneath, the end times, OBSERVED before
// now and FORECAST after it. Read from the library's loop in Update; the
// window keeps no playback state of its own (go-tuiMaps L-1.13).
func (d Dashboard) radarTimeline(width int) []string {
	if d.mapPane.m == nil || !d.radarTimelineOn() {
		return nil
	}
	s, ok := d.radarScrubber()
	if !ok {
		return []string{"", "", ""}
	}
	return d.draw(s, width)
}

// radarScrubber is the loop's scrubber: where the frame shown is, where NOW
// is, and the words under them, all by time along the loop; false with no
// loop to draw.
func (d Dashboard) radarScrubber() (scrubber, bool) {
	m := d.mapPane.m
	if m == nil || d.mapPane.radarSource == "" {
		return scrubber{}, false
	}
	st := m.Loop()
	if st.Count == 0 {
		return scrubber{}, false
	}
	s := scrubber{above: d.clockAt(st.At)}
	// ONE AXIS, TIME (UAT-2 U2-33): the cursor was placed by frame number
	// and NOW by time, and the frames are five minutes apart observed and
	// fifteen ahead - the newest observed frame drew in the FORECAST half.
	if span := st.Newest.Sub(st.Oldest); span > 0 {
		s.cursor = min(max(float64(st.At.Sub(st.Oldest))/float64(span), 0), 1)
	}
	now := 1.0
	if span := st.Newest.Sub(st.Oldest); span > 0 && st.Now.Before(st.Newest) {
		now = float64(st.Now.Sub(st.Oldest)) / float64(span)
	}
	s.below = []scrubLabel{{d.clockAt(st.Oldest), 0, alignLeft}}
	if now < 1 {
		s.ticks = []float64{now}
		s.below = append(s.below, scrubLabel{d.clockAt(st.Newest), 1, alignRight}, scrubLabel{"NOW", now, alignCentre})
	} else {
		s.below = append(s.below, scrubLabel{"NOW · " + d.clockAt(st.Newest), 1, alignRight}) // no forecast: the loop ends at now
	}
	s.below = append(s.below, scrubLabel{"OBSERVED", now / 2, alignCentre})
	if now < 1 {
		s.below = append(s.below, scrubLabel{"FORECAST", (now + 1) / 2, alignCentre})
	}
	return s, true
}

// clockAt is a moment in the listener's clock and zone.
func (d Dashboard) clockAt(t time.Time) string { return d.clockFmt.Time(t.In(d.now().Location())) }

// firstKey is an action's first bound key, as the listener's [keys] set it.
func (d Dashboard) firstKey(act term.Action) string {
	if keys := d.mapKeys[act].Keys; len(keys) > 0 {
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
