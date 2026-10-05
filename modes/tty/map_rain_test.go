package tty

// map_rain_test.go — 0.18.0 W12 at the window (D-113 to D-117): Forecast
// mode's rain and snow, a row of the Overlays menu on by default and drawn in
// Forecast mode alone, said to be a model's rain, not radar; and the loop's
// hours ahead where they are Open-Meteo's.

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// openRainMap is openFieldsMap with the rain and snow row, on as the app
// registers it, and temperature off: Forecast mode turns it on (D-104).
func openRainMap(t *testing.T, radarOn bool) Dashboard {
	t.Helper()
	var asks []MapAsk
	var asked []string
	cfg := Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapTemperature: tempAnswer(&asks),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: radarOn},
			{Key: RainLayer, Label: "Rain & snow", On: true}, {Key: TemperatureLayer, Label: "Temperature"}, {Key: FeelsLayer, Label: "Feels like"}, {Key: WindLayer, Label: "Wind"}, {Key: UVLayer, Label: "UV"}, {Key: AirLayer, Label: "Air quality"}}}
	d := mapDash(t, cfg)
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

// switchLayer switches a layer in the Overlays menu, and settles.
func switchLayer(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	d = pressCode(d, 'O', "O")
	for i, r := range d.overlayRows() {
		if r.key == key {
			d.mapPane.menuAt = i
		}
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	return pressCode(settleRadar(t, m.(Dashboard), cmd), 'O', "O")
}

// rainGiven counts the rain grids handed in.
func rainGiven(d Dashboard) int {
	n := 0
	for id := range d.mapPane.tempGiven {
		if strings.HasPrefix(id, RainLayer+"/") {
			n++
		}
	}
	return n
}

// menuHas reports whether the Overlays menu has a row of the key.
func menuHas(d Dashboard, key string) bool {
	for _, r := range d.overlayRows() {
		if r.key == key {
			return true
		}
	}
	return false
}

// TestRainIsForecastModesRow is D-117: the rain and snow is a row of the
// Overlays menu in Forecast mode, on by default, its grids handed in with
// temperature's; in Radar mode it is neither a row nor drawn - the radar is
// the rain there.
func TestRainIsForecastModesRow(t *testing.T) {
	d := openRainMap(t, true)
	if menuHas(d, RainLayer) || rainGiven(d) != 0 {
		t.Fatalf("Radar mode lists the rain row (%v) or draws %d rain grids; the radar is its rain", menuHas(d, RainLayer), rainGiven(d))
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	d = settleRadar(t, m.(Dashboard), cmd)
	if !menuHas(d, RainLayer) || !d.layerOn(RainLayer) {
		t.Fatal("Forecast mode's menu has no rain and snow row, on")
	}
	if n := rainGiven(d); n != 1+forecastDays {
		t.Errorf("Forecast mode handed in %d rain grids; want Now's and every day's, up front", n)
	}
	if !d.layerOn(TemperatureLayer) {
		t.Error("with rain on, Forecast mode did not turn temperature on as the tint under it (D-104, D-117)")
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"}) // back to Radar mode, Forecast mode's answer still held
	if n := rainGiven(m.(Dashboard)); n != 0 {
		t.Errorf("back in Radar mode, %d of Forecast mode's rain grids stayed drawn over the radar", n)
	}
}

// TestForecastModesRainSaysItIsNotRadar is D-117's label: where the rain is
// drawn, the colour row under the map is radar's colours said to be a
// model's rain, not radar; the credit is under the map; with the rain off,
// the row is temperature's again.
func TestForecastModesRainSaysItIsNotRadar(t *testing.T) {
	d := openRainMap(t, false)
	d.mapPane.legend = []tuimaps.LegendEntry{{Preset: "radar", Classes: []tuimaps.Class{{Label: "10 to 20", Colour: tuimaps.RGB{R: 34, G: 119, B: 136}, Drawn: true}}},
		{Preset: "temperature", Classes: []tuimaps.Class{{Label: "50 to 60", Colour: tuimaps.RGB{R: 240, G: 232, B: 144}, Drawn: true}}}}
	row := stripANSITest(d.scrubRows(100)[0])
	if !strings.Contains(row, "MODEL RAIN") || !strings.Contains(row, "NOT RADAR") {
		t.Errorf("with rain drawn, the colour row is %q; want radar's colours said to be a model's, not radar", row)
	}
	if b := stripANSITest(strings.Join(d.badges(), " ")); !strings.Contains(b, "RAIN   O-METEO  ") {
		t.Errorf("the badges are %q; want the rain's source (D-133)", b)
	}
	d = switchLayer(t, d, RainLayer)
	if row := stripANSITest(d.scrubRows(100)[0]); !strings.Contains(row, "TEMPERATURE") || rainGiven(d) != 0 {
		t.Errorf("with rain off the row is %q and %d rain grids are drawn", row, rainGiven(d))
	}
}

// NDFD'S TOTALS ARE KEYED IN THEIR OWN SCALE (W18.5, D-184): where the rain
// drawn is NDFD's totals alone, the colour row keys the totals' classes,
// said to be NDFD's totals - not a model's rain in radar's colours.
func TestNDFDsTotalsAreKeyedInTheirOwnScale(t *testing.T) {
	d := openRainMap(t, false)
	d.mapPane.legend = []tuimaps.LegendEntry{{Preset: "qpf", Classes: []tuimaps.Class{{Label: "under 0.25"}, {Label: "0.25 to 2.5", Colour: tuimaps.RGB{R: 133, G: 248, B: 24}, Drawn: true}}},
		{Preset: "temperature", Classes: []tuimaps.Class{{Label: "50 to 60", Colour: tuimaps.RGB{R: 240, G: 232, B: 144}, Drawn: true}}}}
	row := stripANSITest(d.scrubRows(100)[0])
	if !strings.Contains(row, "NDFD TOTALS") || strings.Contains(row, "MODEL RAIN") || !strings.Contains(row, "under 0.25") {
		t.Errorf("with NDFD's totals drawn, the colour row is %q; want the totals' classes, said to be NDFD's", row)
	}
	d.mapPane.legend = append(d.mapPane.legend, tuimaps.LegendEntry{Preset: "radar", Classes: []tuimaps.Class{{Label: "10 to 20", Colour: tuimaps.RGB{R: 34, G: 119, B: 136}, Drawn: true}}})
	if row := stripANSITest(d.scrubRows(100)[0]); !strings.Contains(row, "MODEL RAIN") {
		t.Errorf("with a model's rain drawn too, the row is %q; want radar's colours, said to be a model's", row)
	}
}

// TestTheBadgeSaysRainWhenItIsAlone: with temperature and wind off, the
// badge's step names the day's rain, not its highs.
func TestTheBadgeSaysRainWhenItIsAlone(t *testing.T) {
	d := openRainMap(t, false)
	d = switchLayer(t, d, TemperatureLayer)
	d.mapPane.fcStep = 1
	if got := d.badgeStep(); got != "TODAY RAIN" {
		t.Errorf("the badge's step is %q; want TODAY RAIN", got)
	}
}

// TestTheHoursAheadNameOpenMeteo is D-115 at the window: hours ahead that
// are Open-Meteo's are named on the badge and the loop's row, as HRRR's are.
func TestTheHoursAheadNameOpenMeteo(t *testing.T) {
	feed := aheadFeed(t)
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: func(ctx context.Context, ask MapAsk) MapRadar {
		out := feed(ctx, ask)
		out.Ahead = "Open-Meteo"
		return out
	}, MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = shiftKey(settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd), tea.KeyRight)
	if badge := stripANSITest(d.radarBadge()); !strings.Contains(badge, "RADAR FCST") || !strings.Contains(badge, "O-METEO") {
		t.Errorf("ahead of now the badge is %q; want RADAR FCST and O-METEO", badge)
	}
	if row := stripANSITest(d.loopRow(d.scrubW())); !strings.Contains(row, "FORECAST   O-METEO  ") {
		t.Errorf("ahead of now the loop's row is %q; want it to lead FORECAST O-METEO", row)
	}
}
