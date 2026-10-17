package tty

// map_night_test.go — 0.19.0 D-199: Day / night, a map detail option in the
// weather modes, one night overlay the map works out for every mode.

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// TestDayNightIsAMapDetailOption is D-199: off by default in the weather
// modes; switched on from the Overlays menu, the night is handed in for the
// moment shown, held by the library and said; in Radar mode it follows the
// clock; it stays on as the detail level changes, leaves the level its own,
// and goes when switched off.
func TestDayNightIsAMapDetailOption(t *testing.T) {
	d := menuMap(t)
	if d.detailOn(nightDetailKey) || len(d.mapPane.nightGiven) != 0 || slices.Contains(d.mapPane.m.Overlays(), mapNightID) {
		t.Fatal("Day / night is on, or the night drawn, before it was switched on")
	}
	level := d.detailLevelShown()
	d = menuKey(t, at(t, d, nightDetailKey), "space")
	night, ok := d.mapPane.nightGiven[mapNightID]
	if want, _ := d.nightWanted(); !d.detailOn(nightDetailKey) || !ok || !night.Valid.Equal(want) {
		t.Fatalf("switched on, the night handed in is %v (%v); the moment shown's, %v", night.Valid, ok, want)
	}
	if !slices.Contains(d.mapPane.m.Overlays(), mapNightID) {
		t.Error("the library does not hold the night")
	}
	if !strings.Contains(strings.Join(d.describeLinesAll(), " "), "The night side is shaded and its edge dotted") {
		t.Error("the words do not say the night")
	}
	if d.detailLevelShown() != level {
		t.Errorf("Day / night on makes the level read %q; it is no level's", d.detailLevelShown())
	}
	d.mapPane.menuOn = false
	if d = pressMap(t, d, "R"); d.mapMode() != modeRadar {
		t.Fatal("control: R did not enter Radar mode")
	}
	if !d.mapPane.nightGiven[mapNightID].Valid.Equal(d.now().UTC().Truncate(nightStep)) {
		t.Error("in Radar mode the night is not the clock's")
	}
	later := d.now().Add(2 * nightStep)
	d.now = func() time.Time { return later }
	m, _ := d.Update(struct{}{})
	if d = m.(Dashboard); !d.mapPane.nightGiven[mapNightID].Valid.Equal(later.UTC().Truncate(nightStep)) {
		t.Error("the night does not follow the clock in Radar mode")
	}
	if d = d.cycleDetailLevel(true); !d.detailOn(nightDetailKey) || len(d.mapPane.nightGiven) != 1 {
		t.Error("a new detail level turned Day / night off")
	}
	d = menuKey(t, at(t, d, nightDetailKey), "space")
	if len(d.mapPane.nightGiven) != 0 || slices.Contains(d.mapPane.m.Overlays(), mapNightID) {
		t.Error("switched off, the night is still drawn")
	}
}

// TestTheNightIsForTheStepShown is D-199 in Forecast mode: the night is for
// the middle of the step shown, not the clock.
func TestTheNightIsForTheStepShown(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	if d.mapMode() != modeForecast {
		t.Fatal("control: the map is not in Forecast mode")
	}
	d.mapDetailChoice = layerChoiceKey(map[string]bool{nightDetailKey: true})
	steps := d.forecastSteps()
	if len(steps) < 2 {
		t.Fatal("control: one step only")
	}
	d.mapPane.fcStep = 1
	sp := steps[1].Span
	at, on := d.nightWanted()
	if want := sp.From.Add(sp.Until.Sub(sp.From) / 2).UTC().Truncate(nightStep); !on || !at.Equal(want) {
		t.Errorf("the night is for %v (%v); the step shown's middle is %v", at, on, want)
	}
}

// TestThePropagationModeAlwaysDrawsTheNight is D-174 kept by D-199: the
// Propagation mode draws the night whatever the switch, and its menu offers
// no switch for it.
func TestThePropagationModeAlwaysDrawsTheNight(t *testing.T) {
	d, _ := drawnDash(t)
	if d.detailOn(nightDetailKey) || len(d.mapPane.nightGiven) != 1 {
		t.Fatalf("the Propagation mode draws %d nights with Day / night off; one", len(d.mapPane.nightGiven))
	}
	for _, r := range d.overlayRows() {
		if r.key == nightDetailKey {
			t.Error("the Propagation mode's menu offers a Day / night switch: it is always drawn there")
		}
	}
}

// TestTheTerminatorIsDrawnForTheFieldsTime is W3.1 (D-174): in the
// Propagation mode the night is drawn for the field's own time, not the
// clock's, however long the field is held; an hour ahead, for the middle of
// that hour.
func TestTheTerminatorIsDrawnForTheFieldsTime(t *testing.T) {
	d, _ := drawnDash(t)
	later := drawnAt.Add(50 * time.Minute)
	d.now = func() time.Time { return later }
	m, _ := d.Update(struct{}{})
	d = m.(Dashboard)
	if got := d.mapPane.nightGiven[mapNightID].Valid; !got.Equal(drawnAt.Truncate(nightStep)) {
		t.Errorf("a field made at %v, the clock at %v: the night is drawn for %v", drawnAt, later, got)
	}
	next := drawnAt.Add(40 * time.Minute)
	m, _ = d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: propSnapAt(next, 18)}})
	if d = m.(Dashboard); !d.mapPane.nightGiven[mapNightID].Valid.Equal(next) {
		t.Errorf("a field made at %v: the night is drawn for %v", next, d.mapPane.nightGiven[mapNightID].Valid)
	}
	d = stepAhead(t, d, "shift+right")
	if got, want := d.mapPane.nightGiven[mapNightID].Valid, next.Add(time.Hour+30*time.Minute); !got.Equal(want) {
		t.Errorf("an hour ahead the night is drawn for %v; the middle of the hour, %v", got, want)
	}
}

// TestALayerAddsItsSentence is FR-3.3 (AX-C, N-9): a layer's sentence is
// said in each of the four cases the alerts' words return early from - the
// alerts switched off, the alerts never asked, the selected place out of
// view, and no selected place at all.
func TestALayerAddsItsSentence(t *testing.T) {
	on := menuKey(t, at(t, menuMap(t), nightDetailKey), "space")
	on.mapPane.menuOn = false
	const night = "The night side is shaded and its edge dotted"
	says := func(d Dashboard) bool { return strings.Contains(strings.Join(d.describeLinesAll(), " "), night) }
	if !says(on) {
		t.Fatal("control: the night is not said with everything in place")
	}
	off := on
	off.cfg.MapLayers = slices.Clone(on.cfg.MapLayers)
	for i := range off.cfg.MapLayers {
		if off.cfg.MapLayers[i].Key == AlertLayer {
			off.cfg.MapLayers[i].On = false
		}
	}
	off.mapLayerChoice = ""
	unasked := on
	unasked.mapPane.lanes[laneAlerts] = feedLane{} // an array: the copy is its own
	away := on
	for range 12 {
		away = pressMap(t, away, "right")
	}
	nowhere := on
	nowhere.selected = 1 << 20
	for _, c := range []struct {
		name    string
		d       Dashboard
		control func(Dashboard) bool
	}{
		{"the alerts switched off", off, func(d Dashboard) bool { return slices.Contains(d.describeLinesAll(), alertLayerOffText) }},
		{"the alerts never asked", unasked, func(d Dashboard) bool { return slices.Contains(d.describeLinesAll(), mapAlertsLoadingText) }},
		{"the place out of view", away, func(d Dashboard) bool {
			loc := d.mapPlace()
			return loc != nil && !d.viewBox(d.mapBodySize()).Contains(loc.Lat, loc.Lon)
		}},
		{"no selected place", nowhere, func(d Dashboard) bool { return d.mapPlace() == nil }},
	} {
		if !c.control(c.d) {
			t.Errorf("%s: control: the case is not in force", c.name)
			continue
		}
		if !says(c.d) {
			t.Errorf("%s: the night's sentence is not said: %q", c.name, c.d.describeLinesAll())
		}
	}
}
