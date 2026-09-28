package tty

// map_quake_test.go — 0.18.0 D-122 at the window: the quakes the map draws
// are a Setting - M2.5+ or M1.0+, the past week or the past day - and the
// ask carries it, with the listener's clock for the quakes' times.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheQuakesAreASetting is D-122: M2.5+ over the past week by default;
// → steps the past day, then M1.0+'s week and day, and round again; the
// file's word opens as chosen, anything else as the default; the ask
// carries the feed's name and the clock.
func TestTheQuakesAreASetting(t *testing.T) {
	d, got := uiDash(t, rowMapQuakes)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Quakes -") || !strings.Contains(text, "M2.5+, past week") {
		t.Fatalf("the Maps tab has no quakes row at M2.5+, past week:\n%s", text)
	}
	for _, want := range []string{"2.5_day", "1.0_week", "1.0_day", "2.5_week"} {
		m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
		d = m.(Dashboard)
		if d.mapQuakeFeed != want || d.mapAsk().QuakeFeed != want {
			t.Errorf("→ gave %q (asked %q); want %q", d.mapQuakeFeed, d.mapAsk().QuakeFeed, want)
		}
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapQuakeFeed != "2.5_day" {
		t.Errorf("esc wrote %q, want 2.5_day", got.MapQuakeFeed)
	}
	if mapDash(t, Config{MapQuakeFeed: "1.0_day"}).mapQuakeFeed != "1.0_day" || mapDash(t, Config{}).mapQuakeFeed != "2.5_week" || mapDash(t, Config{MapQuakeFeed: "all_month"}).mapQuakeFeed != "2.5_week" {
		t.Error("the file's word does not open as chosen, or a word not offered is not the default")
	}
	d.clockFmt = render.ClockByKey("24h")
	if d.mapAsk().Clock != render.ClockByKey("24h") {
		t.Error("the ask does not carry the listener's clock")
	}
}

// TestRefusalsGoToTheDiagnostics is UAT-2 U2-29 and D-124: overlays the
// library refuses are ours to fix, not the listener's - a note each ran the
// notes past the map and hid it. None is said under the map; the
// diagnostics are told once a layer, by its name.
func TestRefusalsGoToTheDiagnostics(t *testing.T) {
	var problems []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapProblem: func(p string) { problems = append(problems, p) },
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: "quake", Label: "Earthquakes", On: true}}})
	d, _ = pressKey(d, "g")
	var bad []tuimaps.Overlay
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		bad = append(bad, tuimaps.Overlay{ID: "quake/" + id, Valid: time.Now(), Keeps: 8 * 24 * time.Hour,
			Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: -117.3, Lat: 33.3}, RadiusDots: 4, Role: tuimaps.QuakeDay}}})
	}
	d = d.setFeed(MapFeed{Overlays: bad})
	for _, n := range d.mapPane.notes {
		if strings.Contains(n, "could not") || strings.Contains(n, "tuimaps") {
			t.Errorf("a refusal was said under the map: %q", n)
		}
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "Earthquakes: 5 not drawn") {
		t.Errorf("the diagnostics were told %q; want one line, Earthquakes: 5 not drawn", problems)
	}
}

// TestTheAppsProblemsReachTheDiagnostics is D-124: what the app could not
// fetch and no Setting fixes arrives as a problem, and the window hands it
// on to the diagnostics, never under the map.
func TestTheAppsProblemsReachTheDiagnostics(t *testing.T) {
	var asks []MapAsk
	var problems []string
	d := openTempMap(t, true, &asks)
	d.cfg.MapProblem = func(p string) { problems = append(problems, p) }
	m, _ := d.applyMapTemp(mapTempMsg{temp: MapTemperature{Problems: []string{"Temperature: Open-Meteo did not answer"}}, anchor: d.tempAnchor()})
	d = m.(Dashboard)
	m, _ = d.applyMapRadar(mapRadarMsg{radar: MapRadar{Problems: []string{"Radar ahead: HRRR did not answer"}}, region: d.mapPane.region.Name})
	d = m.(Dashboard)
	if strings.Join(problems, "|") != "Temperature: Open-Meteo did not answer|Radar ahead: HRRR did not answer" {
		t.Errorf("the diagnostics were told %q", problems)
	}
	for _, n := range d.tempNotes() {
		if strings.Contains(n, "did not answer") {
			t.Errorf("a problem was said under the map: %q", n)
		}
	}
}
