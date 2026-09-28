package tty

// map_feels_test.go — 0.18.0 D-119 at the window: feels-like is its own
// row of the Overlays menu, never on with temperature; drawn as temperature
// is, and said.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// givenOf counts the grids of a layer handed in.
func givenOf(d Dashboard, layer string) int {
	n := 0
	for id := range d.mapPane.tempGiven {
		if strings.HasPrefix(id, layer+"/") {
			n++
		}
	}
	return n
}

// TestFeelsLikeAndTemperatureAreNeverBothOn is D-119: switching feels-like on
// turns temperature off, and the reverse; only the one on is drawn.
func TestFeelsLikeAndTemperatureAreNeverBothOn(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	if !menuHas(d, FeelsLayer) || d.layerOn(FeelsLayer) {
		t.Fatal("the Overlays menu has no feels-like row, off")
	}
	d = switchLayer(t, d, FeelsLayer)
	if !d.layerOn(FeelsLayer) || d.layerOn(TemperatureLayer) {
		t.Fatalf("feels-like on: feels %v, temperature %v; want feels-like alone", d.layerOn(FeelsLayer), d.layerOn(TemperatureLayer))
	}
	if givenOf(d, FeelsLayer) == 0 || givenOf(d, TemperatureLayer) != 0 {
		t.Errorf("feels-like on drew %d feels-like and %d temperature grids", givenOf(d, FeelsLayer), givenOf(d, TemperatureLayer))
	}
	d = switchLayer(t, d, TemperatureLayer)
	if d.layerOn(FeelsLayer) || !d.layerOn(TemperatureLayer) || givenOf(d, FeelsLayer) != 0 {
		t.Errorf("temperature on again: feels %v, temperature %v, %d feels-like grids", d.layerOn(FeelsLayer), d.layerOn(TemperatureLayer), givenOf(d, FeelsLayer))
	}
}

// TestFeelsLikeInForecastModeIsSaid is D-119 in Forecast mode: with
// feels-like on, temperature is not turned on for it; its days flip between
// high and low; the badge and the colour row say feels like.
func TestFeelsLikeInForecastModeIsSaid(t *testing.T) {
	var asks []MapAsk
	d := switchLayer(t, openTempMapWith(t, true, false, &asks), FeelsLayer)
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if d.mapPane.tempAuto || d.layerOn(TemperatureLayer) {
		t.Error("feels-like was on, and Forecast mode turned temperature on as if nothing were")
	}
	if n := givenOf(d, FeelsLayer); n != 1+forecastDays {
		t.Errorf("Forecast mode drew %d feels-like grids; want Now and every day's high", n)
	}
	d = pressCode(d, '<', "<")
	low := 0
	for id := range d.mapPane.tempGiven {
		if strings.HasPrefix(id, FeelsLayer+"/") && strings.HasSuffix(id, "/low") {
			low++
		}
	}
	if low != forecastDays {
		t.Errorf("< drew %d feels-like lows; want every day's", low)
	}
	d.mapPane.fcStep = 1
	if badge := stripANSITest(d.forecastBadge()); !strings.Contains(badge, "FORECAST") || !strings.Contains(badge, "TODAY FEELS LOWS") {
		t.Errorf("the badge is %q; want FORECAST and TODAY FEELS LOWS - the room D-120's one line gives", badge)
	}
	d = switchLayer(t, d, WindLayer)
	d.mapPane.fcStep = 1
	if got := d.badgeStep(); got != "TODAY FEELS LOWS" {
		t.Errorf("with feels-like and wind on, the badge's step is %q; want TODAY LOWS - wind is not alone", got)
	}
	d.mapPane.legend = []tuimaps.LegendEntry{{Preset: "temperature", Classes: []tuimaps.Class{{Label: "50 to 60", Colour: tuimaps.RGB{R: 240, G: 232, B: 144}, Drawn: true}}}}
	if row := stripANSITest(d.tempLegendRow(80)); !strings.HasPrefix(row, "FEELS LIKE │") {
		t.Errorf("the colour row is %q; want FEELS LIKE's", row)
	}
}
