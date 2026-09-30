package app

// mapwaves.go — the map's wave height (0.18.0 D-125, D-126): NDFD's where
// it reaches and Open-Meteo Marine's beyond it, a lattice a field box, drawn
// over the sea alone as bands of a wave scale with labelled contours
// (go-tuiMaps L-20). Its own row, off by default; loaded in the background
// while the map is open, as the temperature is (D-99).

import (
	"context"
	"strconv"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// Waves, registered: off by default (D-126).
func init() {
	registerMapLayer(mapLayer{key: tty.WaveLayer, label: "Waves", on: false, cost: waveLayerCost})
}

// waveSource is one source of wave height: NDFD, or Open-Meteo Marine.
type waveSource interface {
	Waves(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Waves, error)
}

// withWaves adds the waves for the mode (D-126): each field box NDFD's,
// filled from Open-Meteo's where NDFD has none (D-125); Radar mode's every
// hour up to now, Forecast mode's Now and each day's highest, each during
// its step. What no source answered is the diagnostics' (D-124).
func withWaves(ctx context.Context, t tty.MapTemperature, ndfd, om waveSource, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	anchor := askAnchor(ask, now)
	unit := tuimaps.Metres
	if ask.Fahrenheit {
		unit = tuimaps.Feet
	}
	nowStep, days := forecastDays(anchor)
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		lat := temperature.LatticeFor(b.Name, b.Box)
		near, nerr := ndfd.Waves(ctx, lat, now)
		far, ferr := om.Waves(ctx, lat, now)
		var w temperature.Waves
		switch {
		case nerr == nil && ferr == nil:
			w = near.FilledFrom(far)
		case nerr == nil:
			w = near
		case ferr == nil:
			w = far
		default:
			t.Problems = append(t.Problems, "Waves: neither NDFD nor Open-Meteo answered for "+b.Name)
			continue
		}
		if !ask.Forecast {
			t.Waves = append(t.Waves, hourGrids(tty.WaveLayer, b.Name, w.Hours, len(w.Hourly), anchor, radarHorizon(ask, anchor), func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
				return waveGrid(id, lat, w.Hourly[i], unit, valid, anchor)
			})...)
			continue
		}
		if vals, ok := w.At(anchor); ok {
			if o, ok := waveGrid(tty.WaveLayer+"/"+b.Name+"/now", lat, vals, unit, anchor, anchor); ok {
				o.During = nowStep.Span
				t.Waves = append(t.Waves, o)
			}
		}
		for k, step := range days {
			if o, ok := waveGrid(tty.WaveLayer+"/"+b.Name+"/d"+strconv.Itoa(k), lat, w.Max[k], unit, anchor, anchor); ok {
				o.During = step.Span
				t.WaveDays = append(t.WaveDays, o)
			}
		}
	}
	if len(t.Waves)+len(t.WaveDays) > 0 {
		t.Chips = withChips(t.Chips, tty.WaveLayer, "NDFD", "O-METEO") // D-133; the credit in full is the Status window's
	}
	return t
}

// waveGrid is a lattice's wave height, metres, as the library's wave grid in
// the listener's unit, lined - labelled contours over faint bands, as the
// temperature's are (D-126); false when no point has any.
func waveGrid(id string, l temperature.Lattice, metres []float64, unit tuimaps.WaveUnit, valid, anchor time.Time) (tuimaps.Overlay, bool) {
	var toFeet func(float64) float64
	if unit == tuimaps.Feet {
		toFeet = func(m float64) float64 { return m / 0.3048 }
	}
	// To the coast: the library draws it over the sea alone (U2-31).
	return linedGrid(id, l.InterpolateOut(metres), toFeet, valid, anchor, func(id string, g tuimaps.Grid, valid time.Time) tuimaps.Overlay {
		return tuimaps.WaveGrid(id, g, unit, valid)
	})
}

// waveBytes are a field box's two wave requests on the wire: NDFD's 18 KB
// for 6 points, about 245 KB for 80; Open-Meteo's 72 KB for 80, asked for
// the loop's thirteen hours ahead (2026-09-28).
const waveBytes = 320_000

// waveLayerCost is two requests a field box, NDFD's and Open-Meteo's.
func waveLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := len(fieldBoxes(in.region, in.view))
	return int64(boxes) * waveBytes, 2 * boxes
}
