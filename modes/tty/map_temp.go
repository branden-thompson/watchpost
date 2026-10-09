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
	"context"
	"fmt"
	"slices"
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

// BuoyLayer and TideLayer are the sea's stations (D-127, D-128): each its
// own row, off by default, asked only while on.
const (
	BuoyLayer = "buoys"
	TideLayer = "tides"
)

// WaveLayer is wave height's (D-126): its own row, off by default, drawn
// over the sea alone.
const WaveLayer = "waves"

// UVLayer is the UV index's (D-137) and AirLayer air quality's (D-139): each
// its own row, off by default, one tint with temperature and feels-like;
// asked only while on.
const (
	UVLayer  = "uv"
	AirLayer = "air"
)

// FireLayer is fire's (D-121): the perimeters, the named incidents and the
// satellite hotspots, one row, on by default.
const FireLayer = "fire"

// FeelsLayer is feels-like temperature's (D-119): its own row, off by
// default, never on with temperature - the two share one tint.
const FeelsLayer = "feels"

// RainLayer is Forecast mode's rain and snow (W12.3, D-117): on by default,
// drawn in Forecast mode alone - Radar mode's rain is the radar.
const RainLayer = "rain"

// tempSourceOpenMeteo is the file's word for Open-Meteo; anything else is
// NDFD, the default (D-185, D-190).
const tempSourceOpenMeteo = "open-meteo"

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
	// Rain is Forecast mode's rain and snow (W12.3, D-116): Now's hour and
	// each day's heaviest in radar's scale, each day's totals marked on it;
	Rain []tuimaps.Overlay
	// Feels, FeelsHigh and FeelsLow are feels-like's (D-119), as Overlays,
	// High and Low are temperature's.
	Feels, FeelsHigh, FeelsLow []tuimaps.Overlay
	// Waves are wave height's (D-126): Radar mode's every hour, or Forecast
	// mode's Now; WaveDays Forecast mode's each day's highest.
	Waves, WaveDays []tuimaps.Overlay
	// UV and UVDays are the UV index's (D-137), Air and AirDays the model's
	// US AQI (D-139): Radar mode's every hour, or Forecast mode's Now; each
	// day's highest, or worst.
	UV, UVDays   []tuimaps.Overlay
	Air, AirDays []tuimaps.Overlay
	// Chips are the sources each of these layers is drawn from, by the
	// layer's key, as its badge names them (D-133): [O-METEO], [NDFD].
	Chips map[string][]string
	// Problems are what went wrong that the listener cannot act on: the
	// diagnostics', never said to them (D-124).
	Problems []string
	Source   string
	Notes    []string // temperature's, feels like's and wind's
	// LayerNotes are another layer's notes, by its key: shown while that
	// layer is on (U2-52, D-203). Each of these turns temperature off, so
	// their notes cannot ride on Notes.
	LayerNotes map[string][]string
	// Quota is a source's spent quota, while one is (W18.1, D-165): the map
	// says it in its notice. Nil when nothing is refused.
	Quota *MapQuota
	// Filled are the days Open-Meteo filled where the source had nothing
	// (D-100), as "‹day›/high" or "‹day›/low", the day counted from today.
	Filled map[string]bool
}

// MapQuota is a source's spent quota (W18.1, D-165): whose, which period's
// limit - Daily, Hourly - and when it resets.
type MapQuota struct {
	Source, Period string
	Resets         time.Time
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

// mapMode is the map's mode (D-94). A place that decides by it names the
// modes it means - == a mode, or a switch listing every mode - so a mode
// added later falls into no other's behaviour (W2.0, C-M2).
type mapMode uint8

// The map's modes.
const (
	modeRadar    mapMode = iota + 1 // the radar loop: the radar layer on
	modeForecast                    // the days ahead: the radar layer off
)

// mapMode is the mode the map is in: Radar while the radar layer is on
// (D-94), else Forecast.
func (d Dashboard) mapMode() mapMode {
	if d.chosen(RadarLayer) {
		return modeRadar
	}
	return modeForecast
}

// tempAnchor is the start of the listener's hour: Forecast mode's Now, and
// what its days are counted from.
func (d Dashboard) tempAnchor() time.Time {
	now := d.now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
}

// forecastSteps are the steps as the window stands.
func (d Dashboard) forecastSteps() []ForecastStep { return ForecastSteps(d.tempAnchor()) }

// toggleTempSource switches the map's temperature, both modes, between NDFD
// and Open-Meteo (D-93, D-190).
func (d Dashboard) toggleTempSource() Dashboard {
	d.mapTempNDFD = !d.mapTempNDFD
	return d.uiTouched() // the next ask is for the source chosen
}

// tempSourceKey is the file's word: empty for NDFD, the default, or
// "open-meteo".
func tempSourceKey(ndfd bool) string {
	if ndfd {
		return ""
	}
	return tempSourceOpenMeteo
}

// meteredNote is said under the Temperature row while Open-Meteo is chosen
// (D-190): its quota is billed a point at a time (D-185).
const meteredNote = "Open-Meteo is metered: every point of the map is a call, and 10,000 a day are shared by this machine's address. NDFD has no quota."

// rainDetailFull is the file's word for the rain's full density (D-192).
const rainDetailFull = "full"

// rainDetailKey is the file's word: "full", or empty for coarse.
func rainDetailKey(full bool) string {
	if full {
		return rainDetailFull
	}
	return ""
}

// rainDetailLabel is the row's words.
func rainDetailLabel(full bool) string {
	if full {
		return "Full"
	}
	return "Coarse"
}

// rainFullNote is said under the row while the full density is chosen
// (D-192, D-23): what it costs.
const rainFullNote = "Full density asks Open-Meteo about four times the calls for the rain from Day 4 on: it bills every point of the map."

// toggleRainDetail switches the rain's density past NDFD's reach (D-192).
func (d Dashboard) toggleRainDetail() Dashboard {
	d.mapRainFull = !d.mapRainFull
	return d.uiTouched() // the next ask is at the density chosen
}

// tempSourceLabel is the row's words: the source of both modes (D-190).
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
	return d, workers.cmd(context.Background(), nil, func(ctx context.Context) tea.Msg {
		return mapTempMsg{temp: temp(ctx, ask), anchor: anchor}
	})
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
	d = d.timed("answered:temp")
	d.mapPane.temp, d.mapPane.tempAnchor = v.temp, v.anchor // held whether or not it is drawn (D-99)
	for _, p := range v.temp.Problems {
		d.problem(p) // D-124: the diagnostics', never the listener's
	}
	d = d.setTemp()
	if !d.mapPane.tempHanded || d.mapPane.m.Pending() == 0 {
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
		case d.mapMode() == modeRadar:
		case d.mapPane.fcLow:
			out = append(out, t.Low...)
		default:
			out = append(out, t.High...)
		}
	}
	if d.layerOn(FeelsLayer) { // feels-like, as temperature (D-119)
		out = append(out, t.Feels...)
		switch {
		case d.mapMode() == modeRadar:
		case d.mapPane.fcLow:
			out = append(out, t.FeelsLow...)
		default:
			out = append(out, t.FeelsHigh...)
		}
	}
	if d.layerOn(WindLayer) { // the wind's, beside it or alone (D-110)
		out = append(out, t.Wind...)
		if d.mapMode() == modeForecast {
			out = append(out, t.WindDays...)
		}
	}
	if d.layerOn(WaveLayer) { // the waves, over the sea alone (D-126)
		out = append(out, t.Waves...)
		if d.mapMode() == modeForecast {
			out = append(out, t.WaveDays...)
		}
	}
	if d.layerOn(RainLayer) && d.mapMode() == modeForecast { // Forecast mode's rain and snow (D-117)
		out = append(out, t.Rain...)
	}
	for _, m := range []struct {
		layer      string
		hours, day []tuimaps.Overlay
	}{{UVLayer, t.UV, t.UVDays}, {AirLayer, t.Air, t.AirDays}} { // D-137, D-139
		if d.layerOn(m.layer) {
			out = append(out, m.hours...)
			if d.mapMode() == modeForecast {
				out = append(out, m.day...)
			}
		}
	}
	return out
}

// rainRowHead is the colour row's head while Forecast mode draws its rain:
// it looks like radar, and it is not (D-117).
const rainRowHead = "MODEL RAIN · NOT RADAR │ "

// totalsRowHead is the colour row's head while the rain drawn is NDFD's
// daily totals alone (D-168, D-184): amounts in their own scale.
const totalsRowHead = "NDFD TOTALS · NOT RADAR │ "

// rainKey is the colour row's head and the scale it keys: a model's rain in
// radar's colours wherever any is drawn, else NDFD's totals in theirs.
func (d Dashboard) rainKey() (head, preset string) {
	for _, e := range d.mapPane.legend {
		if e.Preset == "radar" {
			return rainRowHead, "radar"
		}
	}
	for _, e := range d.mapPane.legend {
		if e.Preset == "qpf" {
			return totalsRowHead, "qpf"
		}
	}
	return rainRowHead, "radar"
}

// rainOn reports whether Forecast mode draws its rain and snow now.
func (d Dashboard) rainOn() bool {
	if d.mapMode() == modeRadar || !d.layerOn(RainLayer) {
		return false
	}
	for id := range d.mapPane.tempGiven {
		if strings.HasPrefix(id, RainLayer+"/") {
			return true
		}
	}
	return false
}

// setTemp hands in the grids the mode draws and takes off the rest; an
// unchanged grid is not handed in again (U1-28). Whether it handed any in is
// kept as tempHanded, for the caller that draws only when the library will not.
func (d Dashboard) setTemp() Dashboard {
	given, set := d.reconcile(d.mapPane.tempGiven, d.tempOverlays(), func(o tuimaps.Overlay, err error) {
		d.problem("Temperature: " + o.ID + " not drawn - " + err.Error()) // never swallowed (U2-5), never the listener's to act on (D-124)
	})
	d.mapPane.tempGiven, d.mapPane.tempHanded = given, set
	return d
}

// reconcile hands the library the overlays wanted that changed, and takes
// off the ones no longer wanted: radar's loops, the temperature's grids and
// the feed's overlays (W14, S-5, D-257). An unchanged overlay is not handed in
// again - that would drop what was prepared (U1-28); one refused keeps the
// form drawn before it (U2-14), and refused is told. It returns what is given now, and whether anything was
// set.
func (d Dashboard) reconcile(had map[string]tuimaps.Overlay, want []tuimaps.Overlay, refused func(tuimaps.Overlay, error)) (given map[string]tuimaps.Overlay, set bool) {
	m := d.mapPane.m
	given = map[string]tuimaps.Overlay{}
	for _, o := range want {
		if prev, ok := had[o.ID]; ok && SameOverlay(prev, o) {
			given[o.ID] = o
			continue
		}
		var err error
		d.mapPane.call("Set", func() { _, err = m.Set(o) })
		if err != nil {
			refused(o, err)
			if prev, ok := had[o.ID]; ok {
				given[o.ID] = prev
			}
			continue
		}
		given[o.ID], set = o, true
	}
	for id := range had {
		if _, ok := given[id]; !ok {
			d.mapPane.call("Remove", func() { _, _ = m.Remove(id) })
		}
	}
	return given, set
}

// tempOn reports whether temperature - or feels-like, its other measure
// (D-119) - is drawn now.
func (d Dashboard) tempOn() bool {
	return (d.layerOn(TemperatureLayer) || d.layerOn(FeelsLayer)) && len(d.mapPane.tempGiven) > 0
}

// feelsOn reports whether feels-like is the tint drawn (D-119).
func (d Dashboard) feelsOn() bool { return d.layerOn(FeelsLayer) && !d.layerOn(TemperatureLayer) }

// The mode's keys: R switches Radar mode on and off (D-94); < and > flip
// Forecast mode's days between high and low (D-97).
const (
	actMapRadar   term.Action = "map.radar"
	actMapHighLow term.Action = "map.highlow"
	forecastLabel             = "FORECAST"
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
	choice[RadarLayer] = d.mapMode() == modeForecast
	d.mapLayerChoice = layerChoiceKey(choice)
	d.setup.uiDirty = true
	save := d.uiApplyCmd()
	d.setup.uiDirty = false
	d.mapPane.fcStep, d.mapPane.fcPlaying = 0, false
	d.mapPane.fcGen++
	d.mapPane.tempAuto = false // Forecast mode's alone (D-104)
	d = d.ensureMainOverlay()
	d = d.setTemp() // what is held, drawn or taken off at once (D-99)
	d = d.refreshMapCost().showStep().retimeDrawn()
	d, radar := d.askRadar()
	d, temp := d.askTemp()
	return d, tea.Batch(save, radar, temp, d.mapWorkCmd())
}

// ensureMainOverlay turns temperature on when Forecast mode would otherwise
// draw no main overlay (D-103) - a blank map reads as broken - and shows the
// chip that says so. It is Forecast mode's alone, never saved (D-104).
func (d Dashboard) ensureMainOverlay() Dashboard {
	if d.mapMode() == modeRadar || d.cfg.MapTemperature == nil || d.layerOn(TemperatureLayer) || d.layerOn(WindLayer) || d.layerOn(FeelsLayer) || d.layerOn(UVLayer) || d.layerOn(AirLayer) {
		return d // a main overlay is on already: temperature, feels-like, wind, UV or air quality (D-110, D-119, D-137, D-139)
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
	if d.mapMode() == modeForecast {
		steps := d.forecastSteps()
		sp = steps[min(d.mapPane.fcStep, len(steps)-1)].Span
	}
	d.mapPane.call("ShowMoment", func() { _ = m.ShowMoment(sp.From, sp.Until) })
	return d
}

// flipHighLow switches Forecast mode's days between high and low (D-97).
func (d Dashboard) flipHighLow() Dashboard {
	if d.mapMode() == modeRadar {
		return d
	}
	d.mapPane.fcLow = !d.mapPane.fcLow
	d = d.setTemp()
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
	return d.showStep().retimeDrawn(), cmd, true
}

// applyForecastTick advances a playing forecast one step, holding the last.
func (d Dashboard) applyForecastTick(v forecastTickMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.fcGen || !d.mapPane.fcPlaying || d.modal != modalMap || d.mapMode() == modeRadar {
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
	return d.showStep().retimeDrawn(), d.forecastTick()
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
	if d.mapMode() == modeRadar && d.mapPane.m != nil {
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
	if t.Happened && d.mapMode() == modeForecast {
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

// retimeDrawn is retime, drawn once: the feed's overlays set again with the
// mode's spans, which draws, or with no feed yet, the frame drawn alone - so a
// step, a tick and the mode's switch draw the frame once, not twice (W14, P-9).
func (d Dashboard) retimeDrawn() Dashboard {
	if d.mapPane.feed == nil {
		return d.renderMap()
	}
	return d.retime()
}

// forecastBadge is Forecast mode's badge in the radar badge's place (D-92),
// the HUM LEAD's layout (UAT-2 U2-19), three rows flush right: FORECAST; the
// temperature's source as a chip, O-METEO or NDFD on its ground, when it is drawn;
// the step in capitals, NOW or FRI HIGHS.
func (d Dashboard) forecastBadge() string {
	chip := ""
	if d.tempOn() {
		chip = chipFace(d.tempFace()) // no brackets (D-174)
	}
	return mapBadge(render.Tint(forecastLabel, render.Tok(render.ModalTitle)), chip, d.badgeStep())
}

// badgeStep is the step as the badge says it: NOW, or the day and HIGHS or
// LOWS - TODAY HIGHS, FRI LOWS.
func (d Dashboard) badgeStep() string {
	if d.mapPane.fcStep == 0 {
		switch {
		case d.layerOn(UVLayer):
			return "NOW UV" // D-137
		case d.layerOn(AirLayer):
			return "NOW AIR" // D-139
		case d.feelsOn():
			return "NOW FEELS LIKE" // D-119
		}
		return "NOW"
	}
	steps := d.forecastSteps()
	s := steps[min(d.mapPane.fcStep, len(steps)-1)]
	day := strings.ToUpper(s.Span.From.Format("Mon"))
	if d.mapPane.fcStep == 1 {
		day = "TODAY"
	}
	if d.feelsOn() {
		day += " FEELS" // D-119: room on D-120's one line
	}
	switch {
	case d.layerOn(UVLayer):
		return day + " UV" // the day's highest (D-137)
	case d.layerOn(AirLayer):
		return day + " AIR" // the day's worst (D-139)
	}
	tint := d.layerOn(TemperatureLayer) || d.layerOn(FeelsLayer)
	if !tint && d.layerOn(WindLayer) {
		return day + " PEAK" // the day's peak wind (D-108)
	}
	if !tint && d.rainOn() {
		return day + " RAIN" // the day's rain and snow alone (D-116)
	}
	if d.mapPane.fcLow {
		return day + " LOWS"
	}
	return day + " HIGHS"
}

// stepSource is the source of the step shown: Open-Meteo on a day it filled
// (D-100), else the source asked.
func (d Dashboard) stepSource() string {
	if at := d.mapPane.fcStep; at > 0 && d.mapMode() == modeForecast {
		side := "high"
		switch {
		case !d.layerOn(TemperatureLayer) && !d.layerOn(FeelsLayer) && d.layerOn(WindLayer):
			side = "wind"
		case d.mapPane.fcLow:
			side = "low"
		}
		if d.feelsOn() && side != "wind" {
			side = "feels" + side // D-119: filled as temperature's are
		}
		if d.mapPane.temp.Filled[strconv.Itoa(at-1)+"/"+side] {
			return "Open-Meteo"
		}
	}
	return d.mapPane.temp.Source
}

// tempFace is the temperature source as its chip says it: O-METEO or NDFD.
func (d Dashboard) tempFace() string {
	if d.stepSource() == "Open-Meteo" {
		return "O-METEO"
	}
	return strings.ToUpper(d.stepSource())
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
	return strings.Join([]string{words, "step " + strconv.Itoa(at+1) + " of " + strconv.Itoa(len(steps)), state}, " · ")
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
	switch { // the key of what is drawn: its scale is in the map's legend
	case d.layerOn(UVLayer) && d.legendHas("uv"):
		return d.presetRow("uv", "UV │ ", "LOW ", " EXTREME", width) // D-137, D-140: the key the bands are read by
	case d.layerOn(AirLayer) && d.legendHas("aqi"):
		return d.presetRow("aqi", "AIR QUALITY │ ", "GOOD ", " HAZARDOUS", width) // D-139, D-140
	}
	if d.feelsOn() {
		return d.presetRow("temperature", "FEELS LIKE │ ", "COLDER ", " WARMER", width) // D-119
	}
	if !d.layerOn(TemperatureLayer) && d.layerOn(WindLayer) {
		return d.presetRow("wind", "WIND │ ", "CALMER ", " STRONGER", width) // the wind's colours, where temperature's are not shown (D-109)
	}
	return d.presetRow("temperature", "TEMPERATURE │ ", "COLDER ", " WARMER", width)
}

// legendHas reports whether a preset is in the map's legend: drawn.
func (d Dashboard) legendHas(preset string) bool {
	return slices.ContainsFunc(d.mapPane.legend, func(e tuimaps.LegendEntry) bool { return e.Preset == preset })
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
	return strings.Join([]string{p.temp.Source, strings.Join(p.temp.Notes, "\n"), fmt.Sprint(p.temp.LayerNotes), fmt.Sprint(p.temp.Chips), strconv.FormatBool(d.rainOn()),
		strconv.Itoa(p.fcStep), strconv.FormatBool(p.fcLow), strconv.FormatBool(p.fcPlaying),
		strconv.Itoa(len(p.tempGiven)), strings.Join(p.fcTimeline, "\n"), d.stepSource()}, "|")
}

// tempNotes are the words temperature says under the map: what its source
// lacks, while it does. Its credit is its badge's, in full the Status
// window's (D-131, D-132).
func (d Dashboard) tempNotes() []string {
	var out []string
	if d.layerOn(TemperatureLayer) || d.layerOn(FeelsLayer) {
		out = append(out, d.mapPane.temp.Notes...)
	}
	for _, key := range notedLayers { // in one order, not the map's (D-203)
		if d.layerOn(key) {
			out = append(out, d.mapPane.temp.LayerNotes[key]...)
		}
	}
	return out
}

// notedLayers are the layers with notes of their own, in the order said.
var notedLayers = [...]string{RainLayer, WaveLayer, UVLayer, AirLayer}
