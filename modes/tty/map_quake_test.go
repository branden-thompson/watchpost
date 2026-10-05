package tty

// map_quake_test.go — 0.18.0 D-122 at the window: the quakes the map draws
// are a Setting - M2.5+ or M1.0+, the past week or the past day - and the
// ask carries it, with the listener's clock for the quakes' times.

import (
	"context"
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

// AN UNDRAWN ALERT AREA IS THE DIAGNOSTICS' ALONE (D-124, D-240; the HUM
// LEAD's screenshot, 2026-10-03: "Gale Warning: Clarence Strait could not be
// drawn" five times under the map while panning). Nothing the listener can do
// about it, so nothing under the map and nothing in the Status window's notes;
// the diagnostics hear each words once while the feed carries them, however
// many alerts share them, and again only after they went and came back.
func TestAnUndrawnAreaIsTheDiagnosticsAlone(t *testing.T) {
	var problems []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapProblem: func(p string) { problems = append(problems, p) },
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}}})
	d, _ = pressKey(d, "g")
	gale := "Gale Warning: Clarence Strait could not be drawn (0 of 1 zones shown). Guajome Park, CA does not lie in it."
	feed := MapFeed{Undrawn: []string{gale, gale, gale, gale, gale}}
	d = d.setFeed(feed)
	for _, n := range d.mapNotesNow() {
		if strings.Contains(n, "could not be drawn") {
			t.Errorf("an undrawn area was said to the listener: %q", n)
		}
	}
	d = d.setFeed(feed)
	if len(problems) != 1 || problems[0] != gale {
		t.Errorf("the diagnostics were told %q; want the words once", problems)
	}
	d = d.setFeed(MapFeed{})
	d.setFeed(feed)
	if len(problems) != 2 {
		t.Errorf("words that went and came back were told %d times in all; want twice", len(problems))
	}
}

// THE UNDRAWN WORDS FOLLOW THE ALERT LAYER (D-240): the feed the window is
// handed keeps them while the alert areas are on, and drops them with the
// areas' other words when they are off.
func TestTheUndrawnWordsFollowTheAlertLayer(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}}})
	feed := MapFeed{Undrawn: []string{"Gale Warning: Clarence Strait could not be drawn"}}
	if got := d.feedForLayers(feed).Undrawn; len(got) != 1 {
		t.Errorf("with the alert areas on, the undrawn words are %q", got)
	}
	off := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: false}}})
	if got := off.feedForLayers(feed).Undrawn; len(got) != 0 {
		t.Errorf("with the alert areas off, the undrawn words are %q", got)
	}
}

// A LIBRARY WARNING IS THE DIAGNOSTICS' (D-124, go-tuiMaps D-104): every
// warning but a failed tile - which drives the window's own offline note -
// is handed to the diagnostics with its kind and count, never said under the
// map.
func TestALibraryWarningReachesTheDiagnostics(t *testing.T) {
	if got := warningText(tuimaps.Warning{Kind: tuimaps.NearImageColours, Count: 3}); got != "Map warning: near-image-colours (3 times)" {
		t.Errorf("a near-colour warning reads %q", got)
	}
	if got := warningText(tuimaps.Warning{Kind: tuimaps.TableFallback, Count: 1}); got != "Map warning: table-fallback" {
		t.Errorf("a single warning reads %q", got)
	}
}

// The wiring: a frame drawn with two overlays whose ids differ only by case
// makes the library warn near-duplicate-id, and the warning reaches the
// diagnostics.
func TestAFramesWarningsReachTheDiagnostics(t *testing.T) {
	var problems []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapProblem: func(p string) { problems = append(problems, p) },
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: "quake", Label: "Earthquakes", On: true}}})
	d, _ = pressKey(d, "g")
	point := func(id string) tuimaps.Overlay {
		return tuimaps.Overlay{ID: id, Valid: time.Now(), Keeps: time.Hour,
			Features: []tuimaps.Feature{{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: -117.3, Lat: 33.3}}}, Role: tuimaps.QuakeDay}}}
	}
	d.setFeed(MapFeed{Overlays: []tuimaps.Overlay{point("quake/near"), point("quake/Near")}})
	for _, p := range problems {
		if strings.HasPrefix(p, "Map warning: near-duplicate-id") {
			return
		}
	}
	t.Errorf("the library's near-duplicate-id warning did not reach the diagnostics: %q", problems)
}

// A REFUSED ALERT UPDATE KEEPS THE AREA DRAWN (D-257): an alert already on
// the map whose next overlay the library refuses stays drawn, as a refused
// radar loop or temperature grid does (U2-14), and the refusal goes to the
// diagnostics.
func TestARefusedAlertUpdateKeepsTheAreaDrawn(t *testing.T) {
	var problems []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapProblem: func(p string) { problems = append(problems, p) },
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}}})
	d, _ = pressKey(d, "g")
	d = d.setFeed(boxFeed(-117.6, -117.1, false)(context.Background(), MapAsk{}))
	if _, ok := d.mapPane.given["alert/w1"]; !ok || len(d.mapPane.m.Overlays()) == 0 {
		t.Fatal("the alert was not drawn: this measures nothing")
	}
	d = d.setFeed(MapFeed{Overlays: []tuimaps.Overlay{{ID: "alert/w1"}}}) // no valid time, no feature: refused
	if _, ok := d.mapPane.given["alert/w1"]; !ok || len(d.mapPane.m.Overlays()) == 0 {
		t.Error("a refused update took the drawn alert off the map")
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "Alert areas: 1 not drawn") {
		t.Errorf("the diagnostics were told %q; want the refusal once", problems)
	}
}

// TestABoundTheLibraryRefusesGoesToTheDiagnostics is D-124 for the region's
// bound: the library refusing it is ours to fix, so the diagnostics are told,
// never the listener.
func TestABoundTheLibraryRefusesGoesToTheDiagnostics(t *testing.T) {
	var problems []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapProblem: func(p string) { problems = append(problems, p) }})
	d, _ = pressKey(d, "g")
	d.mapPane.region.S, d.mapPane.region.N = d.mapPane.region.N, d.mapPane.region.S // south above north: no box
	problems = nil
	d.boundMap()
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "Map bound:") {
		t.Errorf("the diagnostics were told %q; want the refused bound once", problems)
	}
}
