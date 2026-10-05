package app

// mapfeels_test.go — 0.18.0 D-119: feels-like, drawn as temperature is, in
// temperature's requests.

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// TestFeelsLikeFollowsTheModes is D-119: Radar mode's every hour, Forecast
// mode's Now and each day's high and low, each during its step, in the
// listener's unit, drawn with temperature's look.
func TestFeelsLikeFollowsTheModes(t *testing.T) {
	src := &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}
	radarMode := buildTemperature(context.Background(), src, nil, tempAsk(false), tempNow, nil)
	if len(radarMode.Feels) != 4 {
		t.Fatalf("Radar mode's feels-like is %d hours; want the four up to now", len(radarMode.Feels))
	}
	for _, o := range radarMode.Feels {
		if !strings.HasPrefix(o.ID, tty.FeelsLayer+"/") || o.Grid.Type.Preset != "temperature" || !o.Grid.Lines || math.Abs(o.Grid.Values[0]-53.6) > 1e-6 {
			t.Errorf("%s: %v %+v lines %v; want temperature's look at 53.6 F", o.ID, o.Grid.Values[0], o.Grid.Type, o.Grid.Lines)
		}
	}
	ask := tempAsk(true)
	fc := buildTemperature(context.Background(), src, nil, ask, tempNow, nil)
	steps := tty.ForecastSteps(ask.Anchor)
	if len(fc.Feels) != 1 || fc.Feels[0].During != steps[0].Span {
		t.Fatalf("Forecast mode's Now feels-like is %d grids", len(fc.Feels))
	}
	if len(fc.FeelsHigh) != temperature.Days || len(fc.FeelsLow) != temperature.Days || fc.FeelsHigh[2].During != steps[3].Span ||
		math.Abs(fc.FeelsHigh[2].Grid.Values[0]-71.6) > 1e-6 || math.Abs(fc.FeelsLow[2].Grid.Values[0]-37.4) > 1e-6 {
		t.Errorf("the days' feels-like: %d highs, %d lows; want every day's, 71.6 F and 37.4 F, during its step", len(fc.FeelsHigh), len(fc.FeelsLow))
	}
}

// TestFeelsLikeIsOffByDefaultAndCostsNothing is D-119: its row is off, and it
// rides temperature's requests.
func TestFeelsLikeIsOffByDefaultAndCostsNothing(t *testing.T) {
	for _, l := range mapLayers {
		if l.key == tty.FeelsLayer {
			if l.on || l.cost != nil {
				t.Errorf("feels-like is on %v, with a cost of its own %v; want off, and nothing of its own", l.on, l.cost != nil)
			}
			return
		}
	}
	t.Fatal("feels-like is not registered")
}
