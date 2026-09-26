package tty

// map_radar.go — the map's radar (0.18.0 W8, FR-5): a loop per radar box from
// the app, set and drawn in Update (D-41), the source named by a chip in the
// map's upper right (D-83), and the listener's playback keys (D-61).
//
// RADAR NEVER HOLDS UP THE ALERTS (W8.12): it is its own command, apart from
// the feed, and asked in two steps - the newest frame first, so the picture
// is current at once, then the whole loop.

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

// mapRadarMsg is a radar answer, for the request it was asked with.
type mapRadarMsg struct {
	gen        uint64
	radar      MapRadar
	newestOnly bool
}

// requestRadar asks for the radar again: on opening, when the view settles,
// on new data, and when a layer or the source changes.
func (d Dashboard) requestRadar() Dashboard {
	d.mapPane.radarGen++
	return d
}

// mapRadarCmd asks the app for the radar off the UI goroutine: the newest
// frame alone, or the whole loop.
func (d Dashboard) mapRadarCmd(newestOnly bool) tea.Cmd {
	radar := d.cfg.MapRadar
	if radar == nil || d.mapPane.m == nil || d.modal != modalMap {
		return nil
	}
	gen, ask, workers := d.mapPane.radarGen, d.mapAsk(), d.mapPane.workers
	if !d.layerOn(RadarLayer) {
		return func() tea.Msg { return mapRadarMsg{gen: gen} } // off: an empty answer takes the loops away, and the app is not asked
	}
	return func() tea.Msg {
		ctx, done, ok := workers.begin()
		if !ok {
			return nil // the map closed: the app is not asked
		}
		defer done()
		return mapRadarMsg{gen: gen, radar: radar(ctx, ask, newestOnly), newestOnly: newestOnly}
	}
}

// applyMapRadar sets the radar's loops, takes off the boxes it no longer has,
// and, after the newest frame alone, asks for the whole loop.
func (d Dashboard) applyMapRadar(v mapRadarMsg) (tea.Model, tea.Cmd) {
	m := d.mapPane.m
	if m == nil || v.gen != d.mapPane.radarGen {
		return d, nil // an older request's answer
	}
	if !d.layerOn(RadarLayer) {
		v.radar = MapRadar{}
	}
	given := map[string]tuimaps.Overlay{}
	for _, o := range v.radar.Overlays {
		if prev, ok := d.mapPane.radarGiven[o.ID]; ok && reflect.DeepEqual(prev, o) {
			given[o.ID] = o // unchanged: handing it in again would drop what was prepared (U1-28)
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err == nil {
			given[o.ID] = o
		}
	}
	for id := range d.mapPane.radarGiven {
		if _, ok := given[id]; !ok {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
		}
	}
	d.mapPane.radarGiven, d.mapPane.radarSource, d.mapPane.radarNote = given, v.radar.Source, v.radar.Note
	d = d.renderMap()
	cmds := []tea.Cmd{d.mapWorkCmd()}
	if v.newestOnly && len(v.radar.Overlays) > 0 {
		d = d.requestRadar()
		cmds = append(cmds, d.mapRadarCmd(false)) // the newest is up: now the whole loop
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
	ground := render.MapRadarMRMSBG
	if d.mapPane.radarSource == "IEM" {
		ground = render.MapRadarIEMBG
	}
	chip := render.TintRaw(text, render.Tok(ground)+";"+render.Tok(render.MapRadarChipFG))
	return spliceBox(lines, []string{chip}, 1, insetCols+size.Cols-render.Width(text))
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
		if d.mapPane.radarNote != "" && d.layerOn(RadarLayer) {
			return d.mapPane.radarNote
		}
		return ""
	}
	st := m.Loop()
	if st.Count == 0 {
		return "Radar (" + d.mapPane.radarSource + "): loading"
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
	parts := []string{"Radar (" + d.mapPane.radarSource + ") " + shown, "frame " + strconv.Itoa(st.Index+1) + " of " + strconv.Itoa(st.Count),
		state, "newest " + ago}
	if st.Forecast {
		parts = append(parts, "forecast")
	}
	return strings.Join(parts, " · ")
}
