package tty

// map_radar.go — the map's radar (0.18.0 W8, FR-5): a loop per radar box from
// the app, set and drawn in Update (D-41), the source named by a chip in the
// map's upper right (D-83), and the listener's playback keys (D-61).
//
// RADAR NEVER HOLDS UP THE ALERTS (W8.12): it is its own command, apart from
// the feed. IT IS SHOWN WHOLE (D-85): the loop is loaded before it is drawn,
// one request at a time, so it never blinks or stands on a single frame.

import (
	"reflect"
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
}

// mapRadarMsg is a radar answer.
type mapRadarMsg struct {
	radar MapRadar
}

// radarRefresh is how long a loaded loop stands before new data asks again:
// the newest frame is five minutes apart at most (D-85).
const radarRefresh = 2 * time.Minute

// askRadar asks the app for the whole loop, off the UI goroutine - ONE
// REQUEST AT A TIME (D-85). A later ask made while one runs is kept and asked
// when the answer lands, never in its place: every later ask superseding the
// last is how a six-second loop was never drawn at all.
func (d Dashboard) askRadar() (Dashboard, tea.Cmd) {
	radar := d.cfg.MapRadar
	if radar == nil || d.mapPane.m == nil || d.modal != modalMap {
		return d, nil
	}
	if d.mapPane.radarBusy {
		d.mapPane.radarAgain = true
		return d, nil
	}
	d.mapPane.radarBusy, d.mapPane.radarAt = true, d.now()
	if !d.layerOn(RadarLayer) {
		return d, func() tea.Msg { return mapRadarMsg{} } // off: an empty answer takes the loops away, and the app is not asked
	}
	ask, workers := d.mapAsk(), d.mapPane.workers
	return d, func() tea.Msg {
		ctx, done, ok := workers.begin()
		if !ok {
			return nil // the map closed: the app is not asked
		}
		defer done()
		return mapRadarMsg{radar: radar(ctx, ask)}
	}
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
	m := d.mapPane.m
	if m == nil || d.modal != modalMap {
		return d, nil
	}
	if !d.layerOn(RadarLayer) {
		v.radar = MapRadar{}
	}
	given, set := map[string]tuimaps.Overlay{}, false
	var refused error
	for _, o := range v.radar.Overlays {
		if prev, ok := d.mapPane.radarGiven[o.ID]; ok && reflect.DeepEqual(prev, o) {
			given[o.ID] = o // unchanged: handing it in again would drop what was prepared (U1-28)
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err != nil {
			refused = err // SAID, never swallowed: a refused loop read as "loading" for ever (UAT-2 U2-5)
			continue
		}
		given[o.ID], set = o, true
	}
	if refused != nil && len(given) == 0 {
		v.radar.Source, v.radar.Note = "", "Radar could not be drawn: "+refused.Error()
	}
	removed := false
	for id := range d.mapPane.radarGiven {
		if _, ok := given[id]; !ok {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
			removed = true
		}
	}
	d.mapPane.radarGiven, d.mapPane.radarSource, d.mapPane.radarNote = given, v.radar.Source, v.radar.Note
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
	return d, tea.Batch(cmds...)
}

// radarChipText is the chip's words: the source in use, or nothing.
func (d Dashboard) radarChipText() string {
	if d.mapPane.radarSource == "" || !d.layerOn(RadarLayer) {
		return ""
	}
	return " " + d.mapPane.radarSource + " "
}

// withRadarChip lays the source's chip in the map's upper right while radar
// is drawn (D-83): MRMS on green, IEM on orange, the name in bold. On the
// second row: the first is the library's, and its time is never covered
// (UAT-1 U1-26).
func (d Dashboard) withRadarChip(lines []string, size tuimaps.Size) []string {
	text := d.radarChipText()
	if text == "" || len(lines) == 0 {
		return lines
	}
	return spliceBox(lines, []string{d.radarChip()}, 1, insetCols+size.Cols-render.Width(text))
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
	age := d.now().Sub(st.Newest)
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

// radarExtraRows are D-89's other held rows while radar is on: the colour
// row, the blank before the timeline and the blank before the chips.
const radarExtraRows = 3

// radarLegendRow is the radar's colours under the map (D-89), lightest to
// heaviest, from the library's legend: a swatch a class, painted in the
// class's own colour, or its words where colour is off.
func (d Dashboard) radarLegendRow(width int) string {
	var classes []tuimaps.Class
	for _, e := range d.mapPane.legend {
		if e.Preset == "radar" {
			classes = e.Classes
			break
		}
	}
	head, lighter, heavier := "RADAR LEGEND │ ", "LIGHTER ", " HEAVIER"
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
// (D-89) as for the map's corner.
func (d Dashboard) radarChip() string {
	ground := render.MapRadarMRMSBG
	if d.mapPane.radarSource == "IEM" {
		ground = render.MapRadarIEMBG
	}
	return render.TintRaw(" "+d.mapPane.radarSource+" ", render.Tok(ground)+";"+render.Tok(render.MapRadarChipFG))
}

// radarTimelineOn reports whether the timeline's rows are held.
func (d Dashboard) radarTimelineOn() bool { return d.cfg.MapRadar != nil && d.layerOn(RadarLayer) }

// radarTimeline is the loop as the listener sees it (D-86): the shown frame's
// time above its mark; the bar from the oldest frame to the last, the step
// keys at its ends, a tick at now; beneath, the end times, OBSERVED before
// now and FORECAST after it. Read from the library's loop in Update; the
// window keeps no playback state of its own (go-tuiMaps L-1.13).
func (d Dashboard) radarTimeline(width int) []string {
	m := d.mapPane.m
	if m == nil || !d.radarTimelineOn() {
		return nil
	}
	st := m.Loop()
	if d.mapPane.radarSource == "" || st.Count == 0 {
		return []string{"", "", ""}
	}
	o := d.opts()
	arrow := strings.NewReplacer("shift+left", "shift+←", "shift+right", "shift+→") // the sketch's faces (D-86)
	if o.ASCII {
		arrow = strings.NewReplacer()
	}
	back, on := o.KeyCap(arrow.Replace(d.firstKey(actMapBack))), o.KeyCap(arrow.Replace(d.firstKey(actMapOn)))
	lead := render.Width(back) + 1
	bar := width - lead - render.Width(on) - 1 - 2 // the two end marks
	if bar < 10 {
		return []string{"", "", ""}
	}
	at := func(frac float64) int { return min(max(int(frac*float64(bar-1)+0.5), 0), bar-1) }
	cur := 0
	if st.Count > 1 {
		cur = at(float64(st.Index) / float64(st.Count-1))
	}
	now := bar - 1
	if span := st.Newest.Sub(st.Oldest); span > 0 && st.Now.Before(st.Newest) {
		now = at(float64(st.Now.Sub(st.Oldest)) / float64(span))
	}
	cells := []rune(strings.Repeat("─", bar))
	if now < bar-1 {
		cells[now] = '┼'
	}
	cells[cur] = '█'
	clock := func(t time.Time) string { return d.clockFmt.Time(t.In(d.now().Location())) }
	above := newPlacer(width)
	above.centre(clock(st.At), lead+1+cur)
	below := newPlacer(width)
	below.left(clock(st.Oldest), lead)
	if now < bar-1 {
		below.right(clock(st.Newest), lead+bar+1)
		below.centre("NOW", lead+1+now)
	} else {
		below.right("NOW · "+clock(st.Newest), lead+bar+1) // no forecast: the loop ends at now
	}
	below.centre("OBSERVED", lead+1+now/2)
	if now < bar-1 {
		below.centre("FORECAST", lead+1+(now+bar)/2)
	}
	return []string{above.String(), back + " ├" + string(cells) + "┤ " + on, below.String()}
}

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
