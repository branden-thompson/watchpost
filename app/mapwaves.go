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
	anchor := ask.Anchor
	if anchor.IsZero() {
		anchor = now.Truncate(time.Hour)
	}
	unit := tuimaps.Metres
	if ask.Fahrenheit {
		unit = tuimaps.Feet
	}
	steps := tty.ForecastSteps(anchor)
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
			for i, h := range w.Hours {
				if h.After(radarHorizon(ask, anchor)) {
					continue // past the loop's hours ahead (U2-32)
				}
				if o, ok := waveGrid(tty.WaveLayer+"/"+b.Name+"/"+h.UTC().Format("2006-01-02T15"), lat, w.Hourly[i], unit, stampOf(h, anchor), anchor); ok {
					o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
					t.Waves = append(t.Waves, o)
				}
			}
			continue
		}
		if vals, ok := w.At(anchor); ok {
			if o, ok := waveGrid(tty.WaveLayer+"/"+b.Name+"/now", lat, vals, unit, anchor, anchor); ok {
				o.During = steps[0].Span
				t.Waves = append(t.Waves, o)
			}
		}
		for k := range temperature.Days {
			if k+1 >= len(steps) {
				break
			}
			if o, ok := waveGrid(tty.WaveLayer+"/"+b.Name+"/d"+strconv.Itoa(k), lat, w.Max[k], unit, anchor, anchor); ok {
				o.During = steps[k+1].Span
				t.WaveDays = append(t.WaveDays, o)
			}
		}
	}
	if len(t.Waves)+len(t.WaveDays) > 0 {
		t.WaveNotes = []string{temperature.OpenMeteoWavesCredit + "."}
	}
	return t
}

// waveGrid is a lattice's wave height, metres, as the library's wave grid in
// the listener's unit, lined - labelled contours over faint bands, as the
// temperature's are (D-126); false when no point has any.
func waveGrid(id string, l temperature.Lattice, metres []float64, unit tuimaps.WaveUnit, valid, anchor time.Time) (tuimaps.Overlay, bool) {
	f := l.InterpolateOut(metres) // to the coast: the library draws it over the sea alone (U2-31)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	if unit == tuimaps.Feet {
		for i, v := range f.Values {
			f.Values[i] = v / 0.3048 // a missing value stays missing
		}
	}
	o := tuimaps.WaveGrid(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Lines: true}, unit, valid)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour // as temperature's: an hour past is that hour's, not stale
	return o, true
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
