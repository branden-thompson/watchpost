package tty

// map_airuv_test.go — 0.18.0 D-137 to D-140: UV and air quality at the map,
// each its own row, one tint with temperature and feels-like, asked while on.

import (
	"strings"
	"testing"
)

// TestUVAndAirAreOneTintWithTemperature is D-137 and D-139: switching UV on
// switches temperature, feels-like and air quality off, and air quality on
// switches UV off - one tint at a time.
func TestUVAndAirAreOneTintWithTemperature(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	if !d.layerOn(TemperatureLayer) {
		t.Fatal("the test needs temperature on")
	}
	d = switchLayer(t, d, UVLayer)
	if !d.layerOn(UVLayer) || d.layerOn(TemperatureLayer) || d.layerOn(FeelsLayer) || d.layerOn(AirLayer) {
		t.Fatalf("UV on: temperature %v, feels %v, air %v; want UV alone", d.layerOn(TemperatureLayer), d.layerOn(FeelsLayer), d.layerOn(AirLayer))
	}
	d = switchLayer(t, d, AirLayer)
	if !d.layerOn(AirLayer) || d.layerOn(UVLayer) {
		t.Errorf("air on: UV %v; want air quality alone", d.layerOn(UVLayer))
	}
}

// TestSwitchingUVOnAsksForIt is D-137: UV is asked only while on, so
// switching it on asks the temperature again with UV said to be on.
func TestSwitchingUVOnAsksForIt(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, true, &asks)
	before := len(asks)
	d = switchLayer(t, d, UVLayer)
	if len(asks) <= before || !asks[len(asks)-1].UV || asks[len(asks)-1].Air {
		t.Errorf("switching UV on asked %d times more, the last %+v; want an ask with UV on", len(asks)-before, asks[len(asks)-1])
	}
	if givenOf(d, UVLayer) == 0 {
		t.Error("no UV grid was handed in")
	}
	if row := stripANSITest(d.tempLegendRow(120)); !strings.HasPrefix(row, "UV │") || !strings.Contains(row, "EXTREME") {
		t.Errorf("the key row is %q; want UV's, low to extreme", row)
	}
}

// TestForecastModeDrawsTheAirsDays is D-139: in Forecast mode air quality is
// Now's and each day's worst, the badge naming it.
func TestForecastModeDrawsTheAirsDays(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	d = switchLayer(t, d, AirLayer)
	if n := givenOf(d, AirLayer); n != 1+forecastDays {
		t.Errorf("Forecast mode drew %d air grids; want Now and every day's worst", n)
	}
	if step := d.badgeStep(); step != "NOW AIR" {
		t.Errorf("the badge's step is %q; want NOW AIR", step)
	}
	if row := stripANSITest(d.tempLegendRow(120)); !strings.HasPrefix(row, "AIR QUALITY │") {
		t.Errorf("the key row is %q; want air quality's", row)
	}
}
