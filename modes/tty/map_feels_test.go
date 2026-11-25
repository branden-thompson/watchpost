package tty

// map_feels_test.go — 0.18.0 D-119 at the window: feels-like is its own
// row of the Overlays menu, never on with temperature; drawn as temperature
// is, and said.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// givenOf counts the grids of a layer handed in.
func givenOf(d Dashboard, layer string) int {
	n := 0
	for id := range d.mapModel.mapPane.tempGiven {
		if strings.HasPrefix(id, layer+"/") {
			n++
		}
	}
	return n
}

// chooseFeels is → on the Temperature row, Feels like, and space on it
// while it is not the tint chosen (U2-39, D-142).
func chooseFeels(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d = pressCode(d, 'O', "O")
	for i, r := range d.mapModel.overlayRows(d.mapEnv()) {
		if r.key == TemperatureLayer {
			d.mapModel.mapPane.menuAt = i
		}
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = settleRadar(t, m, cmd)
	if !d.mapModel.tintChosen(TemperatureLayer, d.mapEnv()) {
		m, cmd, _ = d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		d = settleRadar(t, m, cmd)
	}
	return pressCode(d, 'O', "O")
}

// TestFeelsLikeAndTemperatureAreNeverBothOn is D-119 as U2-39 lays it out:
// the Temperature row's → picks Feels like, turning temperature off, and ←
// Actual again; only the one on is drawn.
func TestFeelsLikeAndTemperatureAreNeverBothOn(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	if menuHas(d, FeelsLayer) || d.mapModel.layerOn(FeelsLayer, d.mapEnv()) || d.mapModel.pickFeels(d.mapEnv()) {
		t.Fatal("feels-like has a row of its own, or is on; want it the Temperature row's choice, Actual")
	}
	d = chooseFeels(t, d)
	if !d.mapModel.layerOn(FeelsLayer, d.mapEnv()) || d.mapModel.layerOn(TemperatureLayer, d.mapEnv()) {
		t.Fatalf("feels-like on: feels %v, temperature %v; want feels-like alone", d.mapModel.layerOn(FeelsLayer, d.mapEnv()), d.mapModel.layerOn(TemperatureLayer, d.mapEnv()))
	}
	if givenOf(d, FeelsLayer) == 0 || givenOf(d, TemperatureLayer) != 0 {
		t.Errorf("feels-like on drew %d feels-like and %d temperature grids", givenOf(d, FeelsLayer), givenOf(d, TemperatureLayer))
	}
	d = chooseFeels(t, d) // → again: Actual
	if d.mapModel.layerOn(FeelsLayer, d.mapEnv()) || !d.mapModel.layerOn(TemperatureLayer, d.mapEnv()) || givenOf(d, FeelsLayer) != 0 {
		t.Errorf("temperature on again: feels %v, temperature %v, %d feels-like grids", d.mapModel.layerOn(FeelsLayer, d.mapEnv()), d.mapModel.layerOn(TemperatureLayer, d.mapEnv()), givenOf(d, FeelsLayer))
	}
}

// TestFeelsLikeInForecastModeIsSaid is D-119 in Forecast mode: with
// feels-like on, temperature is not turned on for it; its days flip between
// high and low; the badge and the colour row say feels like.
func TestFeelsLikeInForecastModeIsSaid(t *testing.T) {
	var asks []MapAsk
	d := chooseFeels(t, openTempMapWith(t, true, false, &asks))
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m, cmd)
	if d.mapModel.mapPane.tempAuto || d.mapModel.layerOn(TemperatureLayer, d.mapEnv()) {
		t.Error("feels-like was on, and Forecast mode turned temperature on as if nothing were")
	}
	if n := givenOf(d, FeelsLayer); n != 1+forecastDays {
		t.Errorf("Forecast mode drew %d feels-like grids; want Now and every day's high", n)
	}
	d = pressCode(d, '<', "<")
	low := 0
	for id := range d.mapModel.mapPane.tempGiven {
		if strings.HasPrefix(id, FeelsLayer+"/") && strings.HasSuffix(id, "/low") {
			low++
		}
	}
	if low != forecastDays {
		t.Errorf("< drew %d feels-like lows; want every day's", low)
	}
	d.mapModel.mapPane.fcStep = 1
	if badge := ttytest.StripANSI(d.mapModel.forecastBadge(d.mapEnv())); !strings.Contains(badge, "FORECAST") || !strings.Contains(badge, "TODAY FEELS LOWS") {
		t.Errorf("the badge is %q; want FORECAST and TODAY FEELS LOWS - the room D-120's one line gives", badge)
	}
	d = switchLayer(t, d, WindLayer)
	d.mapModel.mapPane.fcStep = 1
	if got := d.mapModel.badgeStep(d.mapEnv()); got != "TODAY FEELS LOWS" {
		t.Errorf("with feels-like and wind on, the badge's step is %q; want TODAY LOWS - wind is not alone", got)
	}
	d.mapModel.mapPane.legend = []tuimaps.LegendEntry{{Preset: "temperature", Classes: []tuimaps.Class{{Label: "50 to 60", Colour: tuimaps.RGB{R: 240, G: 232, B: 144}, Drawn: true}}}}
	if row := ttytest.StripANSI(d.mapModel.tempLegendRow(80, d.mapEnv())); !strings.HasPrefix(row, "FEELS LIKE │") {
		t.Errorf("the colour row is %q; want FEELS LIKE's", row)
	}
}

// TestWavesAreTheirOwnRow is D-126 at the window: off by default, drawn
// when switched on - Forecast mode's Now and each day's highest - with the
// credit under the map; taken off again when switched off.
func TestWavesAreTheirOwnRow(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	if !menuHas(d, WaveLayer) || d.mapModel.layerOn(WaveLayer, d.mapEnv()) || givenOf(d, WaveLayer) != 0 {
		t.Fatalf("the waves are on %v with %d grids; want a row, off", d.mapModel.layerOn(WaveLayer, d.mapEnv()), givenOf(d, WaveLayer))
	}
	d = switchLayer(t, d, WaveLayer)
	if n := givenOf(d, WaveLayer); n != 1+forecastDays {
		t.Errorf("Forecast mode drew %d wave grids; want Now and every day's highest", n)
	}
	if b := ttytest.StripANSI(strings.Join(d.mapModel.badges(d.mapEnv()), " ")); !strings.Contains(b, "WAVES   NDFD    O-METEO  ") {
		t.Errorf("the badges are %q; want the waves' sources (D-133)", b)
	}
	if d = switchLayer(t, d, WaveLayer); givenOf(d, WaveLayer) != 0 {
		t.Error("switched off, the waves stayed")
	}
}

// TestTheAskSaysWhetherTheSeasStationsAreOn is D-127 and D-128: their rows
// are off by default and the ask says so - the app asks for them only while
// on; switched on, the ask says that.
func TestTheAskSaysWhetherTheSeasStationsAreOn(t *testing.T) {
	d := mapDash(t, Config{Map: &Hooks{MapFeed: boxFeed(-117.6, -117.1, false)},
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: BuoyLayer, Label: "Buoys"}, {Key: TideLayer, Label: "Tides"}}})
	d, _ = pressKey(d, "g")
	if a := d.mapModel.mapAsk(d.mapEnv()); a.Buoys || a.Tides {
		t.Fatalf("with both rows off the ask says buoys %v, tides %v", a.Buoys, a.Tides)
	}
	d = switchLayer(t, switchLayer(t, d, BuoyLayer), TideLayer)
	if a := d.mapModel.mapAsk(d.mapEnv()); !a.Buoys || !a.Tides {
		t.Errorf("with both rows on the ask says buoys %v, tides %v", a.Buoys, a.Tides)
	}
}
