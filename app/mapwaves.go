package app

// mapwaves.go — the map's wave height (0.18.0 D-125, D-126): NDFD's where
// it reaches and Open-Meteo Marine's beyond it, a lattice a field box, drawn
// over the sea alone as bands of a wave scale with labelled contours
// (go-tuiMaps L-20). Its own row, off by default; loaded in the background
// while the map is open, as the temperature is (D-99).

import (
	"context"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/history"
)

// Waves, registered: off by default (D-126).
func init() {
	registerMapLayer(mapLayer{key: tty.WaveLayer, label: "Waves", on: false, cost: waveLayerCost})
}

// waveSource is NDFD's wave height.
type waveSource interface {
	Waves(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Waves, error)
}

// marineSource is Open-Meteo Marine's, asked a point at a time (D-194).
type marineSource interface {
	WavesAt(ctx context.Context, l temperature.Lattice, only []int, now time.Time) (temperature.Waves, error)
}

// waveKeep is what the waves draw on beside their sources (D-194): the
// history's NDFD hours, and the points Open-Meteo has answered nothing for.
type waveKeep struct {
	store *history.Store
	land  *landPoints
}

// landPoints are the lattice points Open-Meteo Marine has answered nothing
// for, by box: land, or past its reach - never asked again (D-194).
type landPoints struct {
	mu   sync.Mutex
	land map[string]map[int]bool
}

// is reports whether a box's point is known land.
func (lp *landPoints) is(box string, i int) bool {
	if lp == nil {
		return false
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.land[box][i]
}

// learn keeps a box's points Open-Meteo answered nothing for.
func (lp *landPoints) learn(box string, points []int) {
	if lp == nil || len(points) == 0 {
		return
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.land == nil {
		lp.land = map[string]map[int]bool{}
	}
	if lp.land[box] == nil {
		lp.land[box] = map[int]bool{}
	}
	for _, i := range points { // the points asked (P10-02)
		lp.land[box][i] = true
	}
}

// withWaves adds the waves for the mode (D-126), keyless first (D-194): each
// field box NDFD's; the current hour and the hours before it the history's -
// the recorder keeps NDFD's next hour as that hour's - and on a cold start
// NDFD's next hour stretched under the loop; Open-Meteo Marine asked only for
// the points NDFD does not reach and that it has answered for before. Radar
// mode's every hour up to now, Forecast mode's Now and each day's highest,
// each during its step. What no source answered is the diagnostics' (D-124).
func withWaves(ctx context.Context, t tty.MapTemperature, ndfd waveSource, om marineSource, ask tty.MapAsk, now time.Time, keep waveKeep) tty.MapTemperature {
	anchor := askAnchor(ask, now)
	unit := tuimaps.Metres
	if ask.Fahrenheit {
		unit = tuimaps.Feet
	}
	nowStep, days := forecastDays(anchor)
	fromNDFD, fromOM, replayed := false, false, 0 // what drew: the chips name those (D-183)
	missing := map[string]bool{}
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		lat := temperature.LatticeFor(b.Name, b.Box)
		w, nerr := ndfd.Waves(ctx, lat, now)
		if nerr == nil {
			fromNDFD = true
			past := recordedWaves(&w, keep.store, b.Name, anchor)
			replayed += past
			if _, ok := w.At(anchor); !ok {
				if next, ok := w.At(anchor.Add(time.Hour)); ok {
					w.SetHour(anchor, next) // nothing recorded: NDFD's next hour, drawn back under the loop (D-194)
				}
			}
		} else {
			w = temperature.Waves{Lattice: lat}
		}
		emptyDay := ask.Forecast && nerr == nil && anEmptyDay(w, len(days)) // a day past NDFD's reach: Open-Meteo's (D-195)
		if need := beyondNDFD(w, nerr != nil || emptyDay, keep.land, b.Name); len(need) > 0 {
			if far, err := om.WavesAt(ctx, lat, need, now); err == nil {
				keep.land.learn(b.Name, answeredNothing(far, need))
				if nerr != nil {
					w = far
				} else {
					w = w.FilledFrom(far)
				}
				fromOM = true
			}
		}
		if len(w.Hourly) == 0 && nerr != nil {
			t.Problems = append(t.Problems, "Waves: neither NDFD nor Open-Meteo answered for "+b.Name)
			continue
		}
		if !ask.Forecast {
			hours := hourGrids(tty.WaveLayer, b.Name, w.Hours, len(w.Hourly), anchor, radarHorizon(ask, anchor), func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
				return waveGrid(id, lat, w.Hourly[i], unit, valid, anchor)
			})
			fillPast(hours, anchor) // every observed frame drawn (U2-55 to U2-57)
			t.Waves = append(t.Waves, hours...)
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
			} else if emptyDay && fromNDFD {
				missing["Waves: none for "+step.Label+" - past NDFD's reach, and Open-Meteo did not answer."] = true // D-195
			}
		}
	}
	if len(t.Waves)+len(t.WaveDays) > 0 {
		var chips []string // D-133, D-173; the credit in full is the Status window's
		if fromNDFD {
			chips = append(chips, "NDFD")
		}
		if fromOM {
			chips = append(chips, "O-METEO")
		}
		if replayed > 0 {
			chips = append(chips, recordedChip)
		}
		t.Chips = withChips(t.Chips, tty.WaveLayer, chips...) // what drew, never what might have (D-183)
	}
	var said []string
	for note := range missing {
		said = append(said, note)
	}
	sort.Strings(said)
	return withNote(t, tty.WaveLayer, said...)
}

// anEmptyDay reports whether NDFD gave a box's waves nothing for one of the
// days shown: past its reach (D-195).
func anEmptyDay(w temperature.Waves, days int) bool {
	for k := range min(days, temperature.Days) { // a week (P10-02)
		if allMissing(w.Max[k]) {
			return true
		}
	}
	return false
}

// recordedWaves puts the history's NDFD waves in the current hour and the
// pastHours before it, where NDFD has none; how many it put.
func recordedWaves(w *temperature.Waves, store *history.Store, box string, anchor time.Time) int {
	if store == nil {
		return 0
	}
	shape := shapeOf(w.Lattice)
	n := 0
	for back := pastHours; back >= 0; back-- { // four (P10-02)
		h := anchor.Add(-time.Duration(back) * time.Hour)
		rec, ok := store.Get(ndfdWaves.Name, history.Key{Source: "ndfd", Place: box}, h)
		if ok && rec.Shape == shape && w.SetHour(h, rec.Values["waves"]) {
			n++
		}
	}
	return n
}

// beyondNDFD are a box's points NDFD gave nothing for - every point, where it
// did not answer or a day is past its reach (D-195) - that Open-Meteo has
// not answered nothing for before: the ones worth its call (D-194).
func beyondNDFD(w temperature.Waves, every bool, land *landPoints, box string) []int {
	n := w.Lattice.Cols * w.Lattice.Rows
	var out []int
	for p := range n { // a lattice's points (P10-02)
		if land.is(box, p) || (!every && reached(w, p)) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// reached reports whether NDFD gave a point anything, an hour or a day.
func reached(w temperature.Waves, p int) bool {
	for _, row := range w.Hourly { // a series' hours (P10-02)
		if p < len(row) && !math.IsNaN(row[p]) {
			return true
		}
	}
	for _, day := range w.Max {
		if p < len(day) && !math.IsNaN(day[p]) {
			return true
		}
	}
	return false
}

// answeredNothing are the points asked that Open-Meteo gave nothing for.
func answeredNothing(far temperature.Waves, asked []int) []int {
	var out []int
	for _, p := range asked { // the points asked (P10-02)
		if !reached(far, p) {
			out = append(out, p)
		}
	}
	return out
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
