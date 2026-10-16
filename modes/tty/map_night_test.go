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
