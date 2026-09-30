package tty

// map_temp_test.go — 0.18.0 W10 at the window (D-93 to D-98): the two
// modes, R between them; Forecast mode's steps, played and stepped by the
// host; the days' high or low; every grid handed in up front with its span;
// an alert in effect always on now's frame; temperature asked only while on.

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// tempAnswer is a MapTemperature with one grid a step: Radar mode's hours,
// or Forecast mode's Now and each day's high and low, spans as the app gives.
func tempAnswer(asks *[]MapAsk) func(context.Context, MapAsk) MapTemperature {
	layerGrid := func(layer, id string, sp tuimaps.Span, v float64) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 2, Values: []float64{v, v, v, v}}
		o := tuimaps.TemperatureGrid(layer+"/us/"+id, g, tuimaps.Fahrenheit, sp.From)
		o.Keeps, o.During = 72*time.Hour, sp
		return o
	}
	grid := func(id string, sp tuimaps.Span, v float64) tuimaps.Overlay {
		return layerGrid(TemperatureLayer, id, sp, v)
	}
	feels := func(id string, sp tuimaps.Span, v float64) tuimaps.Overlay { return layerGrid(FeelsLayer, id, sp, v) }
	preset := func(make func(string, tuimaps.Grid, time.Time) tuimaps.Overlay, layer string, v float64) func(string, tuimaps.Span) tuimaps.Overlay {
		return func(id string, sp tuimaps.Span) tuimaps.Overlay { // D-137, D-139
			g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 2, Values: []float64{v, v, v, v}}
			o := make(layer+"/us/"+id, g, sp.From)
			o.Keeps, o.During = 72*time.Hour, sp
			return o
		}
	}
	uv, air := preset(tuimaps.UVGrid, UVLayer, 7), preset(tuimaps.AirQualityGrid, AirLayer, 120)
	waves := func(id string, sp tuimaps.Span) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 2, Values: []float64{5, 5, 5, 5}, Lines: true}
		o := tuimaps.WaveGrid(WaveLayer+"/us/"+id, g, tuimaps.Feet, sp.From)
		o.Keeps, o.During = 72*time.Hour, sp
		return o
	}
	wind := func(id string, sp tuimaps.Span) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 2, Values: []float64{15, 15, 15, 15}}
		o := tuimaps.WindGrid(WindLayer+"/us/"+id, g, []float64{270, 270, 270, 270}, tuimaps.MilesPerHour, sp.From)
		o.Keeps, o.During = 72*time.Hour, sp
		return o
	}
	rain := func(id string, sp tuimaps.Span) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 2, Values: []float64{40, 40, 40, 40},
			Type: tuimaps.Type{Preset: "radar", Unit: "dBZ"}, Marks: []string{"1.0in", "", "", ""}}
		return tuimaps.Overlay{ID: RainLayer + "/us/" + id, Valid: sp.From, Keeps: 72 * time.Hour, During: sp, Grid: &g}
	}
	return func(_ context.Context, ask MapAsk) MapTemperature {
		*asks = append(*asks, ask)
		if !ask.Forecast {
			h := ask.Anchor
			om := []string{"O-METEO"}
			return MapTemperature{Source: "Open-Meteo", Chips: map[string][]string{TemperatureLayer: om, FeelsLayer: om, WindLayer: om, WaveLayer: {"NDFD", "O-METEO"}},
				Overlays: []tuimaps.Overlay{grid("h0", tuimaps.Span{From: h.Add(-time.Hour), Until: h.Add(-time.Nanosecond)}, 60),
					grid("h1", tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}, 62)},
				Wind:  []tuimaps.Overlay{wind("h1", tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)})},
				Feels: []tuimaps.Overlay{feels("h1", tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}, 65)},
				UV:    []tuimaps.Overlay{uv("h1", tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)})},
				Air:   []tuimaps.Overlay{air("h1", tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)})}}
		}
		steps := ForecastSteps(ask.Anchor)
		out := MapTemperature{Source: "NDFD", Overlays: []tuimaps.Overlay{grid("now", steps[0].Span, 61)},
			Wind: []tuimaps.Overlay{wind("now", steps[0].Span)}, Rain: []tuimaps.Overlay{rain("now", steps[0].Span)},
			Feels: []tuimaps.Overlay{feels("now", steps[0].Span, 64)}, Waves: []tuimaps.Overlay{waves("now", steps[0].Span)},
			UV: []tuimaps.Overlay{uv("now", steps[0].Span)}, Air: []tuimaps.Overlay{air("now", steps[0].Span)},
			Chips: map[string][]string{TemperatureLayer: {"NDFD"}, FeelsLayer: {"NDFD"}, WindLayer: {"NDFD"},
				WaveLayer: {"NDFD", "O-METEO"}, RainLayer: {"O-METEO"}}}
		for k, s := range steps[1:] {
			out.High = append(out.High, grid("d"+string(rune('0'+k))+"/high", s.Span, 80))
			out.Low = append(out.Low, grid("d"+string(rune('0'+k))+"/low", s.Span, 50))
			out.WindDays = append(out.WindDays, wind("d"+string(rune('0'+k)), s.Span))
			out.Rain = append(out.Rain, rain("d"+string(rune('0'+k)), s.Span))
			out.FeelsHigh = append(out.FeelsHigh, feels("d"+string(rune('0'+k))+"/high", s.Span, 85))
			out.FeelsLow = append(out.FeelsLow, feels("d"+string(rune('0'+k))+"/low", s.Span, 45))
			out.WaveDays = append(out.WaveDays, waves("d"+string(rune('0'+k)), s.Span))
			out.UVDays = append(out.UVDays, uv("d"+string(rune('0'+k)), s.Span))
			out.AirDays = append(out.AirDays, air("d"+string(rune('0'+k)), s.Span))
		}
		return out
	}
}

// openTempMap opens the map with radar and temperature, at 01:00 on a
// Monday, the loop's newest frame five minutes before.
func openTempMap(t *testing.T, radarOn bool, asks *[]MapAsk) Dashboard {
	t.Helper()
	return openTempMapWith(t, radarOn, true, asks)
}

// openTempMapWith is openTempMap with temperature on or off.
func openTempMapWith(t *testing.T, radarOn, tempOn bool, asks *[]MapAsk) Dashboard {
	t.Helper()
	return openFieldsMap(t, radarOn, tempOn, false, asks)
}

// openFieldsMap is openTempMap with temperature and wind each on or off.
func openFieldsMap(t *testing.T, radarOn, tempOn, windOn bool, asks *[]MapAsk) Dashboard {
	t.Helper()
	var asked []string
	cfg := Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapTemperature: tempAnswer(asks),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: radarOn},
			{Key: TemperatureLayer, Label: "Temperature", On: tempOn}, {Key: WindLayer, Label: "Wind", On: windOn},
			{Key: RainLayer, Label: "Rain & snow"}, {Key: FeelsLayer, Label: "Feels like"}, {Key: WaveLayer, Label: "Waves"},
			{Key: UVLayer, Label: "UV"}, {Key: AirLayer, Label: "Air quality"}}} // off: these are temperature's and wind's tests
	d := mapDash(t, cfg)
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

func TestForecastStepsAreNowAndTheDays(t *testing.T) {
	anchor := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC) // a Monday afternoon
	steps := ForecastSteps(anchor)
	var labels []string
	for _, s := range steps {
		labels = append(labels, s.Label)
	}
	if got := strings.Join(labels, ","); got != "Now,Today,Tomorrow,Wed,Thu,Fri,Sat,Sun" {
		t.Fatalf("the steps are %s; want Now, Today, Tomorrow and Day 3 to Day 7 by name (D-94)", got)
	}
	if !steps[0].Span.From.Equal(anchor) || !steps[0].Span.Until.Equal(anchor) {
		t.Errorf("Now is %v; want the anchor alone", steps[0].Span)
	}
	if !steps[1].Span.From.After(anchor) || steps[1].Span.Until.Day() != 24 {
		t.Errorf("Today is %v; want the rest of the anchor's day, Now not in it", steps[1].Span)
	}
	for i := 2; i < len(steps); i++ {
		if gap := steps[i].Span.From.Sub(steps[i-1].Span.Until); gap != time.Nanosecond {
			t.Errorf("%s does not follow %s: a gap of %v", steps[i].Label, steps[i-1].Label, gap)
		}
	}
}

func TestRSwitchesBetweenRadarAndForecastModes(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "R] Radar On") {
		t.Errorf("the chips are %q; want R Radar On among them (D-94)", stripANSITest(d.mapStatusLine()))
	}
	for _, r := range d.overlayRows() {
		if r.key == RadarLayer {
			t.Error("the Overlays menu lists radar; R is the mode, not an overlay (D-94)")
		}
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if d.radarMode() || len(d.mapPane.radarGiven) != 0 {
		t.Fatalf("R left Radar mode on (%v) or its loops (%d)", d.radarMode(), len(d.mapPane.radarGiven))
	}
	if !d.radarTimelineOn() || len(d.mapPane.fcTimeline) != radarRows {
		t.Errorf("Forecast mode holds no timeline rows (%d); the steps are its scrubber (D-94)", len(d.mapPane.fcTimeline))
	}
	status := stripANSITest(d.mapStatusLine())
	if !strings.Contains(status, "R] Radar Off") || !strings.Contains(status, "Forecast · Now") {
		t.Errorf("Forecast mode's lines are %q", status)
	}
	if badge := stripANSITest(d.forecastBadge()); !strings.Contains(badge, "FORECAST") || !strings.Contains(badge, "NDFD") || !strings.Contains(badge, "NOW") {
		t.Errorf("Forecast mode's badge is %q; want FORECAST, the source and the step", badge)
	}
	if last := asks[len(asks)-1]; !last.Forecast {
		t.Error("the temperature was not asked again for Forecast mode")
	}
	m, cmd, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	if d = settleRadar(t, m.(Dashboard), cmd); !d.radarMode() || len(d.mapPane.radarGiven) == 0 {
		t.Error("R again did not bring Radar mode and its loop back")
	}
}

func TestEveryGridIsHandedInUpFrontWithItsSpan(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	if n := len(d.mapPane.tempGiven); n != 2 {
		t.Fatalf("Radar mode handed in %d grids; want both hours at once (L-15.1)", n)
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if n := len(d.mapPane.tempGiven); n != 1+forecastDays {
		t.Fatalf("Forecast mode handed in %d grids; want Now and every day's high", n)
	}
	for id, o := range d.mapPane.tempGiven {
		if o.During == (tuimaps.Span{}) {
			t.Errorf("%s has no span: it would be drawn on every step", id)
		}
		if strings.HasSuffix(id, "/low") {
			t.Errorf("%s handed in with the highs", id)
		}
	}
	calls := []string{}
	d.mapPane.calls = &calls
	for _, key := range []struct {
		code rune
		mod  tea.KeyMod
	}{{tea.KeyRight, tea.ModShift}, {tea.KeyRight, tea.ModShift}, {tea.KeyLeft, tea.ModShift}} {
		m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: key.code, Mod: key.mod})
		d = m.(Dashboard)
	}
	for _, c := range calls {
		if c == "Set" || c == "Remove" {
			t.Fatalf("a step handed a grid in (%s): a grid swapped in blinks until Work prepares it (U1-28)", c)
		}
	}
	if d.mapPane.fcStep != 1 {
		t.Errorf("⇧→ ⇧→ ⇧← left the step at %d; want 1, Today", d.mapPane.fcStep)
	}
	// THE STEP IS WHAT IS DRAWN: Now's 61 F and Tomorrow's high of 80 F are
	// different frames, the library choosing by the moment the step set.
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	d = m.(Dashboard)
	now := strings.Join(d.mapPane.lines, "\n")
	for range 2 {
		m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
		d = m.(Dashboard)
	}
	if strings.Join(d.mapPane.lines, "\n") == now {
		t.Error("Tomorrow drew Now's frame: the step did not move the library's moment")
	}
}

func TestTheDaysFlipBetweenHighAndLow(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	d = m.(Dashboard)
	if got := d.stepWords(); !strings.HasSuffix(got, " high") {
		t.Errorf("Today says %q; a day draws its high by default (D-97)", got)
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: '>', Text: ">"})
	d = m.(Dashboard)
	lows := 0
	for id := range d.mapPane.tempGiven {
		if strings.HasSuffix(id, "/high") {
			t.Errorf("%s still handed in after > flipped to the lows", id)
		}
		if strings.HasSuffix(id, "/low") {
			lows++
		}
	}
	if lows != forecastDays || !strings.HasSuffix(d.stepWords(), " low") {
		t.Errorf("> gave %d lows and %q", lows, d.stepWords())
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	if got := m.(Dashboard).stepWords(); got != "Now" {
		t.Errorf("Now says %q; the switch does not touch Now", got)
	}
}

func TestForecastPlaybackStepsAndHoldsTheLastDay(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	if !d.mapPane.fcPlaying || cmd == nil {
		t.Fatal("space did not play the forecast")
	}
	tick := func() {
		mm, _ := d.Update(forecastTickMsg{gen: d.mapPane.fcGen})
		d = mm.(Dashboard)
	}
	last := len(d.forecastSteps()) - 1
	for range last {
		tick()
	}
	if d.mapPane.fcStep != last {
		t.Fatalf("after %d ticks the step is %d; want the last", last, d.mapPane.fcStep)
	}
	tick()
	if d.mapPane.fcStep != last {
		t.Error("the last day was not held")
	}
	tick()
	if d.mapPane.fcStep != 0 {
		t.Errorf("after the hold the step is %d; want Now again", d.mapPane.fcStep)
	}
	stale := d.mapPane.fcGen
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	mm, _ := d.Update(forecastTickMsg{gen: stale})
	if d = mm.(Dashboard); d.mapPane.fcPlaying || d.mapPane.fcStep != 0 {
		t.Error("a tick of a stopped play moved the step")
	}
}

func TestAnAlertInEffectIsAlwaysOnNowsFrame(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	now := d.now()
	newest := d.mapPane.m.Loop().Now
	if !newest.Before(now) {
		t.Fatalf("the loop's newest frame %v is not before now %v: nothing to test", newest, now)
	}
	issued := TimedOverlay{From: now.Add(-time.Minute), Until: now.Add(time.Hour)} // after the newest frame
	if sp := d.spanFor(issued); !sp.From.Equal(newest) {
		t.Errorf("an alert issued after the newest frame spans from %v; want the newest frame, now's (D-98)", sp.From)
	}
	older := TimedOverlay{From: newest.Add(-30 * time.Minute)}
	if sp := d.spanFor(older); !sp.From.Equal(older.From) {
		t.Errorf("an alert begun mid-loop spans from %v; want its onset", sp.From)
	}
	later := TimedOverlay{From: now.Add(20 * time.Hour)}
	if sp := d.spanFor(later); !sp.From.Equal(later.From) {
		t.Errorf("a watch for tomorrow spans from %v; want its onset kept", sp.From)
	}
	quake := TimedOverlay{From: now.Add(-3 * time.Hour), Happened: true}
	if sp := d.spanFor(quake); !sp.Until.IsZero() {
		t.Errorf("in Radar mode a quake ends at %v; want open", sp.Until)
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if sp := d.spanFor(quake); !sp.Until.Equal(d.tempAnchor()) {
		t.Errorf("in Forecast mode a quake ends at %v; want Now alone (D-98)", sp.Until)
	}
}

// TestTemperatureOffIsHeldNotDrawn is D-99: off by default, the map's
// temperature is still asked while the map is open, and held; switched on,
// it is drawn at once from what is held; off again, its grids go.
func TestTemperatureOffIsHeldNotDrawn(t *testing.T) {
	var asks []MapAsk
	d := openTempMapWith(t, true, false, &asks)
	if len(asks) == 0 {
		t.Fatal("with the map open and temperature off, it was never asked (D-99: loaded in the background)")
	}
	if len(d.mapPane.tempGiven) != 0 {
		t.Fatalf("temperature off drew %d grids", len(d.mapPane.tempGiven))
	}
	d = switchTemp(t, d)
	if len(d.mapPane.tempGiven) == 0 {
		t.Fatal("switched on, the held temperature was not drawn")
	}
	if d = switchTemp(t, d); len(d.mapPane.tempGiven) != 0 {
		t.Errorf("switched off, %d grids stayed", len(d.mapPane.tempGiven))
	}
}

// switchTemp switches temperature in the Overlays menu, and settles.
func switchTemp(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d = pressCode(d, 'O', "O")
	for i, r := range d.overlayRows() {
		if r.key == TemperatureLayer {
			d.mapPane.menuAt = i
		}
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = settleRadar(t, m.(Dashboard), cmd)
	return pressCode(d, 'O', "O")
}

// TestMovingThroughTheMenuAsksNothing is UAT-2 U2-13 and U2-14: an arrow in
// the Overlays menu moves its cursor and nothing else - every press asked the
// radar and the temperature again, and the grids handed in again blinked.
func TestMovingThroughTheMenuAsksNothing(t *testing.T) {
	var asks []MapAsk
	var radarAsks []string
	cfg := Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &radarAsks), MapTemperature: tempAnswer(&asks),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true},
			{Key: TemperatureLayer, Label: "Temperature", On: true}, {Key: UVLayer, Label: "UV"}, {Key: AirLayer, Label: "Air quality"}}}
	d := mapDash(t, cfg)
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	d = pressCode(d, 'O', "O")
	calls := []string{}
	d.mapPane.calls = &calls
	temps, radars := len(asks), len(radarAsks)
	for range 40 {
		m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyDown})
		d = settleRadar(t, m.(Dashboard), cmd)
	}
	if len(asks) != temps || len(radarAsks) != radars {
		t.Errorf("40 arrows asked the temperature %d times and the radar %d", len(asks)-temps, len(radarAsks)-radars)
	}
	for _, c := range calls {
		if c == "Set" || c == "Remove" || c == "Render" {
			t.Fatalf("an arrow in the menu made a %s call: only a switch touches the map", c)
		}
	}
}

// TestARefusedLoopKeepsTheOneDrawn is U2-14: a loop handed in again that the
// library refuses leaves the loop already drawn on the map, and says so.
func TestARefusedLoopKeepsTheOneDrawn(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	held := len(d.mapPane.radarGiven)
	if held == 0 {
		t.Fatal("no loop to keep")
	}
	var bad MapRadar
	for id := range d.mapPane.radarGiven {
		bad.Overlays = append(bad.Overlays, tuimaps.Overlay{ID: id}) // no valid time, no picture: refused
	}
	bad.Source = "MRMS"
	m, _ := d.applyMapRadar(mapRadarMsg{radar: bad, region: d.mapPane.region.Name})
	d = m.(Dashboard)
	if len(d.mapPane.radarGiven) != held || len(d.mapPane.m.Overlays()) == 0 {
		t.Errorf("a refused loop took the drawn one away: %d held", len(d.mapPane.radarGiven))
	}
	if strings.Contains(d.mapPane.radarNote, "could not") {
		t.Errorf("a refusal the listener cannot act on was said to them: %q (D-124)", d.mapPane.radarNote)
	}
}

func TestTheTemperatureSourceIsASetting(t *testing.T) {
	d, got := uiDash(t, rowMapTempSource)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Temperature -") || !strings.Contains(text, "Open-Meteo") {
		t.Fatalf("the Maps tab has no temperature row at Open-Meteo, the default (D-101):\n%s", text)
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if !d.mapTempNDFD || d.tempSourceLabel() != "NDFD (NWS)" || !d.mapAsk().TempNDFD {
		t.Errorf("→ gave %q; want NDFD, in the ask", d.tempSourceLabel())
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapTempSource != "ndfd" {
		t.Errorf("esc wrote %q, want ndfd", got.MapTempSource)
	}
	if !mapDash(t, Config{MapTempSource: "ndfd"}).mapTempNDFD || mapDash(t, Config{}).mapTempNDFD || mapDash(t, Config{MapTempSource: "open-meteo"}).mapTempNDFD {
		t.Error("the file's word does not open the window as chosen")
	}
}

// TestAFilledStepNamesOpenMeteo is D-100: on a day Open-Meteo filled, the
// badge's chip names it; on the chosen source's own days, that source.
func TestAFilledStepNamesOpenMeteo(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	d.mapPane.temp.Filled = map[string]bool{"0/high": true}
	badge := func() string { return stripANSITest(d.forecastBadge()) }
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if d = m.(Dashboard); !strings.Contains(badge(), "O-METEO") {
		t.Errorf("Today, filled from Open-Meteo, has the badge %q", badge())
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if d = m.(Dashboard); !strings.Contains(badge(), "NDFD") {
		t.Errorf("Tomorrow, NDFD's own, has the badge %q", badge())
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: '<', Text: "<"})
	m, _, _ = m.(Dashboard).handleMapKey(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	if d = m.(Dashboard); !strings.Contains(badge(), "NDFD") {
		t.Errorf("Today's low, NDFD's own, has the badge %q", badge())
	}
}

// TestTheForecastBadgeIsTheHUMLEADsLayout is UAT-2 U2-19 as D-120 redraws
// it, one line: FORECAST; the source's chip, [O-METEO] or [ NDFD ]; the step
// in capitals, FRI HIGHS.
func TestTheForecastBadgeIsTheHUMLEADsLayout(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	want := func(w string) {
		t.Helper()
		if got := stripANSITest(d.forecastBadge()); got != w {
			t.Errorf("the badge is %q; want %q", got, w)
		}
	}
	want(" FORECAST    NDFD    NOW ")
	m0, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	d = m0.(Dashboard)
	want(" FORECAST    NDFD    TODAY HIGHS ")
	for range forecastDays - 1 {
		m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
		d = m.(Dashboard)
	}
	day := strings.ToUpper(d.forecastSteps()[forecastDays].Span.From.Format("Mon"))
	want(" FORECAST    NDFD    " + day + " HIGHS ")
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: '>', Text: ">"})
	d = m.(Dashboard)
	d.mapPane.temp.Source = "Open-Meteo"
	want(" FORECAST    O-METEO    " + day + " LOWS ")
}

// TestTheTemperatureKeyReads is UAT-2 U2-18: each band's value is written in
// black or white, whichever reads on the band.
func TestTheTemperatureKeyReads(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	var d Dashboard
	d.mapPane.legend = []tuimaps.LegendEntry{{Preset: "temperature", Classes: []tuimaps.Class{
		{Label: "59 to 68", Colour: tuimaps.RGB{R: 240, G: 232, B: 144}, Drawn: true},
		{Label: "under -22", Colour: tuimaps.RGB{R: 34, B: 68}, Drawn: true}}}}
	row := d.tempLegendRow(80)
	if !strings.Contains(row, "38;2;0;0;0;48;2;240;232;144") || !strings.Contains(row, "38;2;255;255;255;48;2;34;0;68") {
		t.Errorf("the key's words are not set to read on each band: %q", row)
	}
}

// TestForecastModeTurnsTemperatureOnForItself is D-103 and D-104: R into
// Forecast mode with temperature off turns it on - a blank map reads as
// broken - and a chip at the top centre says so until a key. It is Forecast
// mode's alone: nothing saved, off again in Radar mode.
func TestForecastModeTurnsTemperatureOnForItself(t *testing.T) {
	var asks []MapAsk
	d := openTempMapWith(t, true, false, &asks)
	saved := d.mapLayerChoice
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if !d.layerOn(TemperatureLayer) || len(d.mapPane.tempGiven) == 0 {
		t.Fatal("Forecast mode with temperature off drew no temperature: a blank map")
	}
	if _, ok := choiceOf(d.mapLayerChoice, TemperatureLayer); ok {
		t.Errorf("the temperature Forecast mode turned on was saved: %q (D-104)", d.mapLayerChoice)
	}
	top := strings.Join(d.withModeChip(d.mapPane.lines, d.mapBodySize()), "\n")
	if !strings.Contains(stripANSITest(top), "FORECAST MODE: TEMP ENABLED") {
		t.Error("no chip says temperature was turned on")
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if d = m.(Dashboard); d.mapPane.modeChip {
		t.Error("a key left the chip up")
	}
	m, cmd, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if d.layerOn(TemperatureLayer) || len(d.mapPane.tempGiven) != 0 {
		t.Error("back in Radar mode, the temperature Forecast mode turned on stayed on")
	}
	if d.mapLayerChoice != saved && strings.Contains(d.mapLayerChoice, TemperatureLayer) {
		t.Errorf("the choice changed: %q", d.mapLayerChoice)
	}
	m, cmd, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	if d = settleRadar(t, m.(Dashboard), cmd); !d.mapPane.modeChip {
		t.Error("into Forecast mode again, the chip did not say it turned temperature on")
	}
}

// TestSwitchingTheAutoTemperatureOffIsTheListeners is D-104: the menu's
// switch on a temperature Forecast mode turned on turns it off, as it reads.
func TestSwitchingTheAutoTemperatureOffIsTheListeners(t *testing.T) {
	var asks []MapAsk
	d := openTempMapWith(t, false, false, &asks)
	if !d.mapPane.tempAuto || !d.layerOn(TemperatureLayer) {
		t.Fatal("a map opened in Forecast mode with temperature off was blank (D-103)")
	}
	if d = switchTemp(t, d); d.layerOn(TemperatureLayer) || d.mapPane.tempAuto {
		t.Error("the switch on an auto temperature, which read on, did not turn it off")
	}
}

// TestTheOverlaysBoxIsStyledAsSettings is D-103: the title MAP DETAILS /
// OVERLAYS and the group headers in Settings' heading style, a blank row
// between groups and before the keys.
func TestTheOverlaysBoxIsStyledAsSettings(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	box := d.overlaysBox()
	head := strings.Split(render.Tint("§", render.Tok(render.ModalTitle)), "§")[0]
	if !strings.Contains(box[0], head+"MAP DETAILS / OVERLAYS") {
		t.Errorf("the title is %q", box[0])
	}
	var blanks, heads int
	for i, l := range box {
		p := strings.TrimSpace(strings.Trim(stripANSITest(l), "│"))
		switch p {
		case "OVERLAYS", "MAP DETAIL": // U2-39
			heads++
			if !strings.Contains(l, head+p) {
				t.Errorf("the header %q is not in the heading style", p)
			}
		case "":
			blanks++
		}
		if strings.Contains(p, "move") && strings.TrimSpace(strings.Trim(stripANSITest(box[i-1]), "│")) != "" {
			t.Error("no blank row before the keys")
		}
	}
	if heads != 2 || blanks < 2 {
		t.Errorf("%d headers, %d blank rows", heads, blanks)
	}
}

// TestTheLoopRowInForecastMode is D-105 in Forecast mode: FORECAST and the
// source, the step, and the state - PLAYING in green.
func TestTheLoopRowInForecastMode(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	row := d.loopRow(d.scrubW())
	plain := stripANSITest(row)
	if !strings.Contains(plain, "FORECAST") || !strings.Contains(plain, "STEP 1 / 8") || !strings.Contains(plain, "PLAYING") {
		t.Errorf("the row is %q", plain)
	}
	if green := strings.Split(render.Tint("§", render.Tok(render.ProviderOK)), "§")[0]; !strings.Contains(row, green+"PLAYING") {
		t.Errorf("PLAYING is not green: %q", row)
	}
}

// TestWindSharesTheMapWithTemperature is D-110: wind and temperature switch
// apart and show together; wind alone draws only its own.
func TestWindSharesTheMapWithTemperature(t *testing.T) {
	var asks []MapAsk
	count := func(d Dashboard) (temp, wind int) {
		for id := range d.mapPane.tempGiven {
			switch {
			case strings.HasPrefix(id, WindLayer+"/"):
				wind++
			case strings.HasPrefix(id, TemperatureLayer+"/"):
				temp++
			}
		}
		return temp, wind
	}
	if temp, wind := count(openFieldsMap(t, true, true, true, &asks)); temp == 0 || wind == 0 {
		t.Errorf("both on drew %d temperature and %d wind grids; want both", temp, wind)
	}
	if temp, wind := count(openFieldsMap(t, true, false, true, &asks)); temp != 0 || wind == 0 {
		t.Errorf("wind alone drew %d temperature and %d wind grids", temp, wind)
	}
	if _, wind := count(openFieldsMap(t, true, true, false, &asks)); wind != 0 {
		t.Errorf("wind off drew %d wind grids (D-110: off by default, held)", wind)
	}
}

// TestWindIsForecastModesMainOverlayToo is D-110 with D-103: in Forecast mode
// with wind on, temperature is not turned on for it; the badge says the day's
// peak, and the row under the map keys the wind.
func TestWindIsForecastModesMainOverlayToo(t *testing.T) {
	var asks []MapAsk
	d := openFieldsMap(t, true, false, true, &asks)
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if d.mapPane.tempAuto || d.mapPane.modeChip || d.layerOn(TemperatureLayer) {
		t.Error("wind was on, and Forecast mode turned temperature on as if nothing were")
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	d = m.(Dashboard)
	if got := d.badgeStep(); got != "TODAY PEAK" {
		t.Errorf("the badge's step is %q; want TODAY PEAK, the day's peak wind (D-108)", got)
	}
	d.mapPane.legend = []tuimaps.LegendEntry{{Preset: "wind", Classes: []tuimaps.Class{{Label: "under 5", Colour: tuimaps.RGB{R: 188, G: 57, B: 130}, Drawn: true}}}}
	if row := stripANSITest(d.tempLegendRow(80)); !strings.Contains(row, "WIND") || !strings.Contains(row, "STRONGER") {
		t.Errorf("the row under the map is %q; want the wind's colours", row)
	}
}

// A STEP IS DRAWN ONCE (W14, P-9). Forecast mode's step, its playback tick
// and the mode's switch each timed the feed again - which draws - and then
// drew again, the same frame twice. Each now draws once, with the feed in and
// before any has landed.
func TestAForecastStepIsDrawnOnce(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	renders := func(calls []string) int {
		n := 0
		for _, c := range calls {
			if c == "Render" {
				n++
			}
		}
		return n
	}
	calls := &[]string{}
	d.mapPane.calls = calls
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	if n := renders(*calls); n != 1 {
		t.Errorf("the switch to Forecast mode drew %d times; want once", n)
	}
	d = settleRadar(t, m.(Dashboard), cmd)
	for _, fed := range []bool{true, false} {
		if !fed {
			d.mapPane.feed = nil // no feed landed yet: the step must still be drawn
		}
		*calls = nil
		d, _, _ = d.handleForecastPlayback(actMapOn)
		if n := renders(*calls); n != 1 {
			t.Errorf("a step (feed in: %v) drew %d times; want once", fed, n)
		}
		d.mapPane.fcPlaying = true
		*calls = nil
		next, _ := d.applyForecastTick(forecastTickMsg{gen: d.mapPane.fcGen})
		d = next.(Dashboard)
		if n := renders(*calls); n != 1 {
			t.Errorf("a playback tick (feed in: %v) drew %d times; want once", fed, n)
		}
		d.mapPane.fcPlaying = false
	}
}

// THE RADAR'S LOOPS AND THE TEMPERATURE'S GRIDS ARE RECONCILED AS THE FEED'S
// ARE (W14, S-5 - one reconcile now): an unchanged loop or grid is not handed
// in again, which would drop what was prepared and blink (U1-28); and a
// landing that takes a loop off draws at once, though the loop handed in
// beside it is still preparing - the old one is not left on screen.
func TestTheRadarAndTemperatureAreReconciled(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	if len(d.mapPane.radarGiven) == 0 {
		t.Fatal("no loop to reconcile")
	}
	calls := &[]string{}
	d.mapPane.calls = calls
	var same, moved MapRadar
	for _, o := range d.mapPane.radarGiven {
		same.Overlays = append(same.Overlays, o)
		o.ID += "/moved"
		img := *o.Image
		img.West-- // another box: a picture the library has not prepared
		o.Image = &img
		moved.Overlays = append(moved.Overlays, o)
	}
	same.Source, moved.Source = d.mapPane.radarSource, d.mapPane.radarSource
	m, _ := d.applyMapRadar(mapRadarMsg{radar: same, region: d.mapPane.region.Name})
	d = m.(Dashboard)
	if slices.Contains(*calls, "Set") {
		t.Errorf("an unchanged loop was handed in again: %v", *calls)
	}
	*calls = nil
	d.mapPane.feed = nil // no feed yet: retime draws nothing, so the landing's own rule is what draws
	m, _ = d.applyMapRadar(mapRadarMsg{radar: moved, region: d.mapPane.region.Name})
	d = m.(Dashboard)
	if !slices.Contains(*calls, "Set") || !slices.Contains(*calls, "Remove") {
		t.Fatalf("a moved loop was not handed in and the old taken off: %v", *calls)
	}
	if !slices.Contains(*calls, "Render") {
		t.Errorf("a landing that took a loop off did not draw at once: %v", *calls)
	}

	var asks []MapAsk
	d = openTempMap(t, true, &asks)
	if len(d.mapPane.tempGiven) == 0 {
		t.Fatal("no grid to reconcile")
	}
	calls = &[]string{}
	d.mapPane.calls = calls
	d, _ = d.setTemp()
	if slices.Contains(*calls, "Set") || slices.Contains(*calls, "Remove") {
		t.Errorf("unchanged grids were handed in again or taken off: %v", *calls)
	}
}

// A SPENT QUOTA IS SAID ON THE MAP (W18.1, D-165): the HUM LEAD's PIP, top
// centre, on a dark-orange ground - "! Daily Open-Meteo API Usage Exceeded.
// Resets <time>", the time in the listener's clock - while the answer says
// the quota is spent, and gone once an answer does not.
func TestASpentQuotaIsSaidOnTheMap(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	now := d.now()
	resets := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location()) // later today: the time alone
	temp := d.mapPane.temp
	temp.Quota = &MapQuota{Source: "Open-Meteo", Period: "Daily", Resets: resets}
	m, _ := d.applyMapTemp(mapTempMsg{temp: temp, anchor: d.tempAnchor()})
	d = m.(Dashboard)
	lines := d.mapBodyLines()
	want := "! Daily Open-Meteo API Usage Exceeded. Resets " + d.clockFmt.Time(resets.In(d.now().Location()))
	row := -1
	for i, l := range lines {
		if strings.Contains(stripANSITest(l), want) {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatalf("the notice %q is not on the map:\n%s", want, stripANSITest(strings.Join(lines, "\n")))
	}
	if row > 3 {
		t.Errorf("the notice is on row %d; want it at the map's top", row)
	}
	plain := stripANSITest(lines[row])
	at := len([]rune(plain[:strings.Index(plain, want)]))
	if left, right := at, len([]rune(plain))-at-len([]rune(want)); left < 4 || right < 4 || abs(left-right) > 6 {
		t.Errorf("the notice is not centred: %d cells left, %d right", left, right)
	}
	rendering.SetColorEnabledForTest(true)
	coloured := d.mapBodyLines()[row]
	rendering.SetColorEnabledForTest(false)
	if !strings.Contains(coloured, render.Tok(render.MapNoticeQuotaBG)) {
		t.Errorf("the notice is not on its ground %q", render.Tok(render.MapNoticeQuotaBG))
	}
	tomorrow := resets.Add(24 * time.Hour) // another day: the day is said too
	temp.Quota = &MapQuota{Source: "Open-Meteo", Period: "Daily", Resets: tomorrow}
	m, _ = d.applyMapTemp(mapTempMsg{temp: temp, anchor: d.tempAnchor()})
	d = m.(Dashboard)
	if day := "Resets " + d.clockFmt.WeekdayDateTime(tomorrow); !strings.Contains(stripANSITest(strings.Join(d.mapBodyLines(), "\n")), day) {
		t.Errorf("a reset on another day does not say the day: want %q", day)
	}
	temp.Quota = nil
	m, _ = d.applyMapTemp(mapTempMsg{temp: temp, anchor: d.tempAnchor()})
	if strings.Contains(stripANSITest(strings.Join(m.(Dashboard).mapBodyLines(), "\n")), "API Usage Exceeded") {
		t.Error("the notice stayed after an answer that was not refused")
	}
}
