package tty

// map_temp.go — the map's temperature and its two modes (0.18.0 W10, F-182,
// D-93 to D-98).
//
// THE MAP HAS TWO MODES (D-94). RADAR (R on): everything drawn matches the
// radar frame's time - each frame its own hour's temperature (D-96), an alert
// only once it has begun (D-98). FORECAST (R off): Now, Today, Tomorrow and
// Day 3 to Day 7, stepped with the radar's keys; each day its high or, with
// < or >, its low (D-97), and the alerts in effect during it (D-98).
//
// NOTHING IS SWAPPED IN AT A STEP. Every hour's and every day's grid is handed
// to the library up front, each with its span (go-tuiMaps L-15.1), and the
// library draws the one the moment meets - the loop's frame, or the step the
// host says (L-15.2). A grid handed in at the step would blank until Work
// prepared it: U1-28's blink, once a step.

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

// TemperatureLayer is temperature's key: the app registers the layer under
// it, and its overlays' ids begin with it.
const TemperatureLayer = "temperature"

// WindLayer is the wind's (W11, D-110).
const WindLayer = "wind"

// tempSourceNDFD is the file's word for NDFD; anything else is Open-Meteo,
// the default (D-101).
const tempSourceNDFD = "ndfd"

// MapTemperature is the temperature the map draws (W10): Radar mode's every
// hour, or Forecast mode's Now, each grid with its span; Forecast mode's
// days, high and low, each with its day's; the source drawn, and the notes -
// Open-Meteo's credit (CC BY 4.0), or a day a source has no value for.
type MapTemperature struct {
	Overlays  []tuimaps.Overlay
	High, Low []tuimaps.Overlay
	// Wind and WindDays are the wind's (W11, D-108): Radar mode's every hour,
	// or Forecast mode's Now; Forecast mode's each day's peak.
	Wind, WindDays []tuimaps.Overlay
	Source         string
	Notes          []string
	// Filled are the days Open-Meteo filled where the source had nothing
	// (D-100), as "‹day›/high" or "‹day›/low", the day counted from today.
	Filled map[string]bool
}

// mapTempMsg is a temperature answer, to the ask it was made in.
type mapTempMsg struct {
	temp   MapTemperature
	anchor time.Time
}

// ForecastStep is one of Forecast mode's steps: its words, and the span the
// map's moment is set to while it shows.
type ForecastStep struct {
	Label string
	Span  tuimaps.Span
}

// ForecastSteps are Forecast mode's steps from an anchor, the start of the
// listener's hour (D-94, D-97): Now, the anchor itself; Today, the rest of the
// anchor's day; then Tomorrow and Day 3 to Day 7, each a whole day in the
// anchor's zone. The app builds each day's grids with the same spans.
func ForecastSteps(anchor time.Time) []ForecastStep {
	loc := anchor.Location()
	day0 := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
	out := []ForecastStep{{Label: "Now", Span: tuimaps.Span{From: anchor, Until: anchor}}}
	for k := range forecastDays {
		from, until := day0.AddDate(0, 0, k), day0.AddDate(0, 0, k+1).Add(-time.Nanosecond)
		label := from.Format("Mon")
		switch k {
		case 0:
			from, label = anchor.Add(time.Nanosecond), "Today"
		case 1:
			label = "Tomorrow"
		}
		out = append(out, ForecastStep{Label: label, Span: tuimaps.Span{From: from, Until: until}})
	}
	return out
}

// forecastDays is Today and the six days after it (D-94: Today, Tomorrow,
// Day 3 to Day 7).
const forecastDays = 7

// radarMode reports whether the map is in Radar mode: the radar layer is on
// (D-94). Off, it is in Forecast mode.
func (d Dashboard) radarMode() bool { return d.layerOn(RadarLayer) }

// tempAnchor is the start of the listener's hour: Forecast mode's Now, and
// what its days are counted from.
func (d Dashboard) tempAnchor() time.Time {
	now := d.now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
}

// forecastSteps are the steps as the window stands.
func (d Dashboard) forecastSteps() []ForecastStep { return ForecastSteps(d.tempAnchor()) }

// toggleTempSource switches Forecast mode's temperature between Open-Meteo
// and NDFD (D-93, D-101).
func (d Dashboard) toggleTempSource() Dashboard {
	d.mapTempNDFD = !d.mapTempNDFD
	return d.uiTouched() // the next ask is for the source chosen
}

// tempSourceKey is the file's word: "ndfd", or empty for Open-Meteo.
func tempSourceKey(ndfd bool) string {
	if ndfd {
		return tempSourceNDFD
	}
	return ""
}

// tempSourceLabel is the row's words, which say Radar mode's is Open-Meteo
// whatever is chosen here (D-96).
func (d Dashboard) tempSourceLabel() string {
	if d.mapTempNDFD {
		return "NDFD (NWS)"
	}
	return "Open-Meteo"
}

// tempRefresh is how long a temperature answer stands before new data asks
// again; the hour turning asks at once (both sources move on the hour).
const tempRefresh = 20 * time.Minute

// askTemp asks the app for the temperature, off the UI goroutine, one
// request at a time as the radar is (D-85). ASKED EVEN WITH THE LAYER OFF
// while the map is open (D-99): held, so switching it on draws at once.
func (d Dashboard) askTemp() (Dashboard, tea.Cmd) {
	temp := d.cfg.MapTemperature
	if temp == nil || d.mapPane.m == nil || d.modal != modalMap {
		return d, nil
	}
	if d.mapPane.tempBusy {
		d.mapPane.tempAgain = true
		return d, nil
	}
	anchor := d.tempAnchor()
	d.mapPane.tempBusy, d.mapPane.tempAt = true, d.now()
	ask, workers := d.mapAsk(), d.mapPane.workers
	return d, func() tea.Msg {
		ctx, done, ok := workers.begin()
		if !ok {
			return nil // the map closed: the app is not asked
		}
		defer done()
		return mapTempMsg{temp: temp(ctx, ask), anchor: anchor}
	}
}

// refreshTemp asks again once the answer has stood, or at once when the
// listener's hour has turned: the steps and the hours move with it.
func (d Dashboard) refreshTemp() (Dashboard, tea.Cmd) {
	if d.now().Sub(d.mapPane.tempAt) < tempRefresh && d.mapPane.tempAnchor.Equal(d.tempAnchor()) {
		return d, nil
	}
	return d.askTemp()
}

// applyMapTemp keeps the answer and hands its grids in.
func (d Dashboard) applyMapTemp(v mapTempMsg) (tea.Model, tea.Cmd) {
	d.mapPane.tempBusy = false
	if d.mapPane.m == nil || d.modal != modalMap {
		return d, nil
	}
	d.mapPane.temp, d.mapPane.tempAnchor = v.temp, v.anchor // held whether or not it is drawn (D-99)
	d, set := d.setTemp()
	if !set || d.mapPane.m.Pending() == 0 {
		d = d.renderMap() // else the work's answer draws it, whole (D-85)
	}
	cmds := []tea.Cmd{d.mapWorkCmd()}
	if d.mapPane.tempAgain {
		d.mapPane.tempAgain = false
		var again tea.Cmd
		d, again = d.askTemp()
		cmds = append(cmds, again)
	}
	return d, tea.Batch(cmds...)
}

// tempOverlays are the grids the mode draws: Radar mode's hours; Forecast
// mode's Now and each day's high, or low (D-97).
func (d Dashboard) tempOverlays() []tuimaps.Overlay {
	t := d.mapPane.temp
	var out []tuimaps.Overlay
	if d.layerOn(TemperatureLayer) { // held, not drawn, while off (D-99)
		out = append(out, t.Overlays...)
		switch {
		case d.radarMode():
		case d.mapPane.fcLow:
			out = append(out, t.Low...)
		default:
			out = append(out, t.High...)
		}
	}
	if d.layerOn(WindLayer) { // the wind's, beside it or alone (D-110)
		out = append(out, t.Wind...)
		if !d.radarMode() {
			out = append(out, t.WindDays...)
		}
	}
	return out
}

// setTemp hands in the grids the mode draws and takes off the rest; an
// unchanged grid is not handed in again (U1-28).
func (d Dashboard) setTemp() (Dashboard, bool) {
	m := d.mapPane.m
	given, set := map[string]tuimaps.Overlay{}, false
	for _, o := range d.tempOverlays() {
		if prev, ok := d.mapPane.tempGiven[o.ID]; ok && reflect.DeepEqual(prev, o) {
			given[o.ID] = o
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err != nil {
			d.mapPane.tempRefused = "Temperature could not be drawn: " + err.Error() // said, never swallowed (U2-5)
			if prev, ok := d.mapPane.tempGiven[o.ID]; ok {
				given[o.ID] = prev // the grid drawn stays (U2-14)
			}
			continue
		}
		given[o.ID], set = o, true
	}
	for id := range d.mapPane.tempGiven {
		if _, ok := given[id]; !ok {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
		}
	}
	if len(given) > 0 {
		d.mapPane.tempRefused = ""
	}
	d.mapPane.tempGiven = given
	return d, set
}

// tempOn reports whether temperature is drawn now.
func (d Dashboard) tempOn() bool { return d.layerOn(TemperatureLayer) && len(d.mapPane.tempGiven) > 0 }

// The mode's keys: R switches Radar mode on and off (D-94); < and > flip
// Forecast mode's days between high and low (D-97).
const (
	actMapRadar    term.Action = "map.radar"
	actMapHighLow  term.Action = "map.highlow"
	forecastLabel              = "FORECAST"
	forecastBadgeW             = radarBadgeW
)

// switchMode turns Radar mode on or off (D-94): the radar layer's switch,
// saved as the Overlays menu saves it. The loop, the temperature and every
// timed overlay are asked or set again for the mode, and Forecast mode opens
// on Now, stopped.
func (d Dashboard) switchMode() (Dashboard, tea.Cmd) {
	choice := choicesOf(d.mapLayerChoice)
	if choice == nil {
		choice = map[string]bool{}
	}
	choice[RadarLayer] = !d.radarMode()
	d.mapLayerChoice = layerChoiceKey(choice)
	d.setup.uiDirty = true
	save := d.uiApplyCmd()
	d.setup.uiDirty = false
	d.mapPane.fcStep, d.mapPane.fcPlaying = 0, false
	d.mapPane.fcGen++
	d.mapPane.tempAuto = false // Forecast mode's alone (D-104)
	d = d.ensureMainOverlay()
	d, _ = d.setTemp() // what is held, drawn or taken off at once (D-99)
	d = d.refreshMapCost().showStep().retime()
	d = d.renderMap()
	d, radar := d.askRadar()
	d, temp := d.askTemp()
	return d, tea.Batch(save, radar, temp, d.mapWorkCmd())
}

// ensureMainOverlay turns temperature on when Forecast mode would otherwise
// draw no main overlay (D-103) - a blank map reads as broken - and shows the
// chip that says so. It is Forecast mode's alone, never saved (D-104).
func (d Dashboard) ensureMainOverlay() Dashboard {
	if d.radarMode() || d.cfg.MapTemperature == nil || d.layerOn(TemperatureLayer) || d.layerOn(WindLayer) {
		return d // a main overlay is on already: temperature or wind (D-110)
	}
	d.mapPane.tempAuto, d.mapPane.modeChip = true, true
	d.mapPane.gen++
	return d
}

// showStep sets the library's moment for the mode: Forecast mode's step, or
// in Radar mode none - the loop's frame is the moment (L-15.2).
func (d Dashboard) showStep() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	var sp tuimaps.Span
	if !d.radarMode() {
		steps := d.forecastSteps()
		sp = steps[min(d.mapPane.fcStep, len(steps)-1)].Span
	}
	d.mapPane.call("ShowMoment", func() { _ = m.ShowMoment(sp.From, sp.Until) })
	return d
}

// flipHighLow switches Forecast mode's days between high and low (D-97).
func (d Dashboard) flipHighLow() Dashboard {
	if d.radarMode() {
		return d
	}
	d.mapPane.fcLow = !d.mapPane.fcLow
	d, _ = d.setTemp()
	return d.renderMap()
}

// forecastTickMsg is Forecast mode's playback step, to the play it belongs to.
type forecastTickMsg struct{ gen uint64 }

// forecastHold is how many steps the last day is held before the loop runs
// again: two seconds at one a second, as the radar's last frame is (FR-5.9).
const forecastHold = 2

// forecastTick is the next step's tick.
func (d Dashboard) forecastTick() tea.Cmd {
	gen := d.mapPane.fcGen
	return tea.Tick(radarFrameEvery, func(time.Time) tea.Msg { return forecastTickMsg{gen: gen} })
}

// handleForecastPlayback is the playback keys in Forecast mode: the host
// steps, since there is no loop for the library to play (L-15.2).
func (d Dashboard) handleForecastPlayback(act term.Action) (Dashboard, tea.Cmd, bool) {
	last := len(d.forecastSteps()) - 1
	var cmd tea.Cmd
	switch act {
	case actMapPlay:
		d.mapPane.fcPlaying = !d.mapPane.fcPlaying
		d.mapPane.fcGen++
		if d.mapPane.fcPlaying {
			d.mapPane.fcHeld = 0
			cmd = d.forecastTick()
		}
	case actMapBack:
		d.mapPane.fcStep, d.mapPane.fcPlaying = max(d.mapPane.fcStep-1, 0), false
	case actMapOn:
		d.mapPane.fcStep, d.mapPane.fcPlaying = min(d.mapPane.fcStep+1, last), false
	case actMapNewest:
		d.mapPane.fcStep, d.mapPane.fcPlaying = 0, false
	default:
		return d, nil, false
	}
	return d.showStep().retime().renderMap(), cmd, true
}

// applyForecastTick advances a playing forecast one step, holding the last.
func (d Dashboard) applyForecastTick(v forecastTickMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.fcGen || !d.mapPane.fcPlaying || d.modal != modalMap || d.radarMode() {
		return d, nil
	}
	last := len(d.forecastSteps()) - 1
	switch {
	case d.mapPane.fcStep < last:
		d.mapPane.fcStep++
	case d.mapPane.fcHeld < forecastHold-1:
		d.mapPane.fcHeld++
		return d, d.forecastTick()
	default:
		d.mapPane.fcStep, d.mapPane.fcHeld = 0, 0
	}
	return d.showStep().retime().renderMap(), d.forecastTick()
}

// TimedOverlay is when an overlay of the feed is: its onset and its end, and
// whether it is a thing that has happened - a quake - rather than one in
// effect for a while.
type TimedOverlay struct {
	From, Until time.Time
	Happened    bool
}

// timedAnchor is the moment the mode calls now: in Radar mode the newest
// observed frame, else the start of the listener's hour.
func (d Dashboard) timedAnchor() time.Time {
	if d.radarMode() && d.mapPane.m != nil {
		if st := d.mapPane.m.Loop(); st.Count > 0 {
			return st.Now
		}
		return d.now()
	}
	return d.tempAnchor()
}

// spanFor is an overlay's span in the mode (D-98): from its onset to its end.
// AN ALERT IN EFFECT NOW IS ON NOW'S FRAME, whenever it was issued: one
// issued after the newest radar frame, or inside the listener's hour, is
// drawn from the mode's now - the map never hides a warning in effect. A
// quake has happened: from its time on, and in Forecast mode on Now alone.
func (d Dashboard) spanFor(t TimedOverlay) tuimaps.Span {
	anchor := d.timedAnchor()
	sp := tuimaps.Span{From: t.From, Until: t.Until}
	if !t.From.IsZero() && !t.From.After(d.now()) && t.From.After(anchor) {
		sp.From = anchor
	}
	if t.Happened && !d.radarMode() {
		sp.Until = anchor
	}
	if !sp.Until.IsZero() && sp.Until.Before(sp.From) {
		sp.Until = sp.From
	}
	return sp
}

// retime sets the feed's overlays again with the mode's spans: after a mode
// switch, a step, or a new loop. Only an overlay whose span changed is
// handed in again.
func (d Dashboard) retime() Dashboard {
	if d.mapPane.m == nil || d.mapPane.feed == nil {
		return d
	}
	return d.setFeed(*d.mapPane.feed)
}

// forecastBadge is Forecast mode's badge in the radar badge's place (D-92),
// the HUM LEAD's layout (UAT-2 U2-19), three rows flush right: FORECAST; the
// temperature's source as a chip, [O-METEO] or [ NDFD ], when it is drawn;
// the step in capitals, NOW or FRI HIGHS.
func (d Dashboard) forecastBadge() []string {
	chip := ""
	if d.tempOn() {
		face := " " + strings.ToUpper(d.stepSource()) + " "
		if d.stepSource() == "Open-Meteo" {
			face = "O-METEO"
		}
		chip = "[" + render.TintRaw(face, d.tempChipTones()) + "]"
	}
	return mapBadge(render.Tint(forecastLabel, render.Tok(render.ModalTitle)), chip, d.badgeStep())
}

// badgeStep is the step as the badge says it: NOW, or the day and HIGHS or
// LOWS - TODAY HIGHS, FRI LOWS.
func (d Dashboard) badgeStep() string {
	if d.mapPane.fcStep == 0 {
		return "NOW"
	}
	steps := d.forecastSteps()
	s := steps[min(d.mapPane.fcStep, len(steps)-1)]
	day := strings.ToUpper(s.Span.From.Format("Mon"))
	if d.mapPane.fcStep == 1 {
		day = "TODAY"
	}
	if !d.layerOn(TemperatureLayer) && d.layerOn(WindLayer) {
		return day + " PEAK" // the day's peak wind (D-108)
	}
	if d.mapPane.fcLow {
		return day + " LOWS"
	}
	return day + " HIGHS"
}

// stepSource is the source of the step shown: Open-Meteo on a day it filled
// (D-100), else the source asked.
func (d Dashboard) stepSource() string {
	if at := d.mapPane.fcStep; at > 0 && !d.radarMode() {
		side := "high"
		switch {
		case !d.layerOn(TemperatureLayer) && d.layerOn(WindLayer):
			side = "wind"
		case d.mapPane.fcLow:
			side = "low"
		}
		if d.mapPane.temp.Filled[strconv.Itoa(at-1)+"/"+side] {
			return "Open-Meteo"
		}
	}
	return d.mapPane.temp.Source
}

// tempChipTones are the temperature source's chip colours: NDFD the NWS's
// green, Open-Meteo orange - the radar chip's two grounds.
func (d Dashboard) tempChipTones() string {
	ground := render.MapRadarMRMSBG
	if d.stepSource() == "Open-Meteo" {
		ground = render.MapRadarIEMBG
	}
	return render.Tok(ground) + ";" + render.Tok(render.MapRadarChipFG)
}

// stepWords are the step shown, short: "Now", "Sun high", "Tmrw low".
func (d Dashboard) stepWords() string {
	steps := d.forecastSteps()
	s := steps[min(d.mapPane.fcStep, len(steps)-1)]
	if d.mapPane.fcStep == 0 {
		return s.Label
	}
	word := s.Span.From.Format("Mon")
	if d.mapPane.fcLow {
		return word + " low"
	}
	return word + " high"
}

// forecastStatus is Forecast mode's line in the radar line's place: the step,
// where it is among the steps, and whether it plays.
func (d Dashboard) forecastStatus() string {
	steps := d.forecastSteps()
	at := min(d.mapPane.fcStep, len(steps)-1)
	state := "stopped"
	if d.mapPane.fcPlaying {
		state = "playing"
	}
	words := "Forecast · " + steps[at].Label
	if at > 0 && d.tempOn() {
		words += " (" + d.stepWords() + ")"
	}
	parts := []string{words, "step " + strconv.Itoa(at+1) + " of " + strconv.Itoa(len(steps)), state}
	if d.mapPane.tempRefused != "" {
		parts = append(parts, d.mapPane.tempRefused)
	}
	return strings.Join(parts, " · ")
}

// forecastTimeline is Forecast mode's three rows in the radar timeline's
// place (D-94): the step's words above its mark; the steps as a bar, the step
// keys at its ends; beneath, each step's short name where it fits, and the
// high/low keys.
func (d Dashboard) forecastTimeline(width int) []string {
	steps := d.forecastSteps()
	n := float64(len(steps) - 1)
	cur := min(d.mapPane.fcStep, len(steps)-1)
	s := scrubber{cursor: float64(cur) / n, above: steps[cur].Label}
	for i, st := range steps {
		name := st.Label
		if i > 1 {
			name = st.Span.From.Format("Mon")
		}
		s.ticks = append(s.ticks, float64(i)/n)
		s.below = append(s.below, scrubLabel{name, float64(i) / n, alignCentre})
	}
	if width < 2*len(steps)+20 {
		return []string{"", "", ""}
	}
	return d.draw(s, width)
}

// tempLegendRow is the temperature's colours under the map in Forecast mode,
// as the radar's row is (W10.10): a swatch a band, coldest to warmest, each
// band's lower bound written where colour is off.
func (d Dashboard) tempLegendRow(width int) string {
	if !d.layerOn(TemperatureLayer) && d.layerOn(WindLayer) {
		return d.presetRow("wind", "WIND │ ", "CALMER ", " STRONGER", width) // the wind's colours, where temperature's are not shown (D-109)
	}
	return d.presetRow("temperature", "TEMPERATURE │ ", "COLDER ", " WARMER", width)
}

// presetRow is a preset's colours as a row of swatches, low to high, each
// class's words on it in black or white, whichever reads (U2-18).
func (d Dashboard) presetRow(preset, head, colder, warmer string, width int) string {
	var classes []tuimaps.Class
	for _, e := range d.mapPane.legend {
		if e.Preset == preset {
			classes = e.Classes
			break
		}
	}
	room := width - render.Width(head+colder+warmer)
	if len(classes) == 0 || room < len(classes)*2 {
		return ""
	}
	each := room / len(classes)
	var row strings.Builder
	for _, c := range classes {
		label := render.PadTo(render.TruncateCells(c.Label, each-1), each-1)
		row.WriteString(render.SwatchText(label, c.Colour.R, c.Colour.G, c.Colour.B) + " ") // words that read on the band (U2-18)
	}
	return head + colder + strings.TrimRight(row.String(), " ") + warmer
}

// tempMemoKey is everything of the temperature and Forecast mode the frame
// shows, as one word for the window's memo (F-30).
func (d Dashboard) tempMemoKey() string {
	p := d.mapPane
	return strings.Join([]string{p.temp.Source, strings.Join(p.temp.Notes, "\n"), p.tempRefused,
		strconv.Itoa(p.fcStep), strconv.FormatBool(p.fcLow), strconv.FormatBool(p.fcPlaying),
		strconv.Itoa(len(p.tempGiven)), strings.Join(p.fcTimeline, "\n"), d.stepSource()}, "|")
}

// tempNotes are the words temperature says under the map: its notes (the
// credit), and in Radar mode that each frame draws its own hour.
func (d Dashboard) tempNotes() []string {
	if !d.layerOn(TemperatureLayer) {
		return nil
	}
	out := append([]string(nil), d.mapPane.temp.Notes...)
	if d.mapPane.tempRefused != "" {
		out = append(out, d.mapPane.tempRefused)
	}
	return out
}
