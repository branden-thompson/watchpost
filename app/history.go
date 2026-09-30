package app

// history.go — what the sources said, recorded (W18.3b; D-166, D-172,
// D-176): the app's side of platform/history.
//
// RECORDED WHENEVER WATCHPOST RUNS (D-172). Every five minutes - from a
// random start in the first two, so several instances seldom meet - the
// recorder asks NDFD for the current hour of every field box of the
// station's region and of the region the map last drew, and records it:
// whether or not Open-Meteo is answering, so that when it is not, a loop's
// past hours can be drawn from what was recorded (D-166). An hour is fetched
// by one instance of however many run (the store's claim); a box it cannot
// fetch is tried again next time. The store prunes once an hour.
//
// NOTHING HERE IS SAID TO THE LISTENER (D-124): a disk that cannot be
// written, an NDFD that does not answer - each counted in the store's
// Stats, for the diagnostics.

import (
	"context"
	"math"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// ndfdHourly is NDFD's current hour over each field box: the fallback's
// record, and past its 72 hours a month of days for trends (D-176).
var ndfdHourly = history.Dataset{
	Name: "ndfd-hourly", Version: 1, Step: time.Hour,
	Title:       "NDFD, the current hour",
	Description: "The National Weather Service's gridded forecast for the current hour, over each field box of a region: temperature, feels-like and wind.",
	Fields: []history.Field{
		{Name: "temp", Label: "Temperature", Unit: "°C", Decimals: 1},
		{Name: "feels", Label: "Feels like", Unit: "°C", Decimals: 1},
		{Name: "wind", Label: "Wind", Unit: "km/h", Decimals: 1},
		{Name: "gust", Label: "Gusts", Unit: "km/h", Decimals: 1},
		{Name: "wind_from", Label: "Wind from", Unit: "°", Decimals: 0},
	},
	Hours: 72 * time.Hour,
	Days:  30 * 24 * time.Hour,
}

// historyEvery is how often the recorder looks for an hour to record.
const historyEvery = 5 * time.Minute

// historian records the history: the store, NDFD's current hour, and the
// regions to record.
type historian struct {
	store   *history.Store
	hour    func(context.Context, temperature.Lattice, time.Time) (temperature.Series, error)
	regions func() []string
	pruned  atomic.Int64 // the hour last pruned, Unix
}

// startHistory opens the store and records while ctx lives (D-172).
func (lp *livePipelines) startHistory(ctx context.Context) {
	if lp.temp == nil {
		return
	}
	ndfd, ok := lp.temp.ndfd.(*temperature.NDFD)
	if !ok || ndfd == nil {
		return
	}
	h := &historian{store: history.Open(history.DefaultRoot(), time.Now, ndfdHourly), hour: ndfd.Hour, regions: lp.historyRegions}
	lp.mu.Lock()
	lp.history = h.store
	lp.mu.Unlock()
	go func() {
		start := time.Duration(rand.Int64N(int64(2 * time.Minute)))
		select {
		case <-ctx.Done():
			return
		case <-time.After(start):
		}
		h.pass(ctx, time.Now())
		everyTick(ctx, historyEvery, func(now time.Time) { h.pass(ctx, now) })
	}()
}

// historyRegions are the regions recorded: the station's, and the map's
// last, once each.
func (lp *livePipelines) historyRegions() []string {
	var out []string
	st := lp.currentStation().transmitter
	if r, ok := geo.RegionOf(st.Lat, st.Lon); ok {
		out = append(out, r.Name)
	}
	if m, _ := lp.lastMapRegion.Load().(string); m != "" && (len(out) == 0 || out[0] != m) {
		out = append(out, m)
	}
	return out
}

// pass records the current hour of every box of the regions not yet
// recorded, and prunes once an hour.
func (h *historian) pass(ctx context.Context, now time.Time) {
	if h == nil || h.store == nil {
		return
	}
	if h.hour == nil || h.regions == nil {
		return
	}
	for _, region := range h.regions() { // one or two (P10-02)
		for _, b := range recordedBoxes(region) { // a region's boxes (P10-02)
			h.record(ctx, temperature.LatticeFor(b.Name, b.Box), now)
		}
	}
	hour := now.Truncate(time.Hour).Unix()
	if h.pruned.Swap(hour) != hour {
		h.store.RollUpAndPrune()
	}
}

// recordedBoxes are every field box a view of region can draw - the whole
// region's, and in the lower 48 each of the grid's a narrower view draws -
// as fieldBoxes shapes them, so a replayed box is the drawn one.
func recordedBoxes(region string) []fieldBox {
	out := fieldBoxes(region, geo.Box{W: -180, S: -90, E: 180, N: 90})
	if len(out) == 0 {
		return nil
	}
	for _, g := range radar.GridBoxes(region) { // the lower 48's grid (P10-02)
		lon, lat := (g.W+g.E)/2, (g.S+g.N)/2
		out = append(out, fieldBoxes(region, geo.Box{W: lon - 0.1, S: lat - 0.1, E: lon + 0.1, N: lat + 0.1})...)
	}
	return out
}

// record fetches and records one box's current hour, when this instance
// claims it.
func (h *historian) record(ctx context.Context, lat temperature.Lattice, now time.Time) {
	key := history.Key{Source: "ndfd", Place: lat.Name}
	if !h.store.Claim(ndfdHourly.Name, key, now) {
		return // recorded, or another instance's to do
	}
	s, err := h.hour(ctx, lat, now)
	if err != nil {
		return // its claim goes stale, and it is tried again (D-124: counted, never said)
	}
	rec, ok := ndfdRecord(s, key, now)
	if !ok {
		return
	}
	h.store.Put(ndfdHourly.Name, rec)
}

// ndfdRecord is a series' current hour as a record: false when NDFD had no
// temperature for it at all.
func ndfdRecord(s temperature.Series, key history.Key, now time.Time) (history.Record, bool) {
	temp, at, ok := s.HourAt(now)
	if !ok || allMissing(temp) {
		return history.Record{}, false
	}
	n := s.Lattice.Cols * s.Lattice.Rows
	if err := invariant.Check(len(temp) == n, "an hour covers its lattice"); err != nil {
		return history.Record{}, false
	}
	values := map[string][]float64{"temp": temp}
	if v, ok := s.FeelsAt(at); ok && len(v) == n {
		values["feels"] = v
	}
	if speed, from, _, ok := s.WindAt(at); ok && len(speed) == n && len(from) == n {
		values["wind"], values["wind_from"] = speed, from
	}
	if g, ok := s.GustAt(at); ok && len(g) == n {
		values["gust"] = g
	}
	return history.Record{Key: key, At: at, IssuedAt: now, Values: values,
		Shape: history.Shape{Box: s.Lattice.Box, Cols: s.Lattice.Cols, Rows: s.Lattice.Rows}}, true
}

// recordedChip is the one chip a layer drawing anything from the history
// appends after its source's (D-173): its ground the muted violet D-178 gave.
const recordedChip = "RECORDED"

// pastHours is how many hours before the current one a replay reaches: as
// many as Open-Meteo gives (past_hours=3), so a loop looks as it does live.
const pastHours = 3

// fallback is what draws a box Open-Meteo did not answer: NDFD's hours, and
// the hours before the current one from the history (W18.2, W18.3b).
type fallback struct {
	src  temperature.Source
	past func(box string, hour time.Time) (history.Record, bool)
}

// recordedHour reads a box's recorded NDFD hour from store; nil without one.
func recordedHour(store *history.Store) func(string, time.Time) (history.Record, bool) {
	if store == nil {
		return nil
	}
	return func(box string, hour time.Time) (history.Record, bool) {
		return store.Get(ndfdHourly.Name, history.Key{Source: "ndfd", Place: box}, hour)
	}
}

// withRecorded is a series with the recorded hours before anchor - as many
// as were recorded of pastHours - set before its own, each field in its
// place, a field not recorded missing; and how many were. A record of
// another shape than the series' lattice is not drawn.
func withRecorded(s temperature.Series, past func(string, time.Time) (history.Record, bool), box string, anchor time.Time) (temperature.Series, int) {
	if past == nil {
		return s, 0
	}
	shape := history.Shape{Box: s.Lattice.Box, Cols: s.Lattice.Cols, Rows: s.Lattice.Rows}
	n := shape.Cols * shape.Rows
	if n <= 0 {
		return s, 0
	}
	var out temperature.Series
	out.Lattice = s.Lattice
	count := 0
	for back := pastHours; back >= 1; back-- { // three (P10-02)
		hour := anchor.Add(-time.Duration(back) * time.Hour)
		rec, ok := past(box, hour)
		if !ok || rec.Shape != shape || len(rec.Values["temp"]) != n {
			continue
		}
		field := func(name string) []float64 {
			if v := rec.Values[name]; len(v) == n {
				return v
			}
			return missingRow(n)
		}
		out.Hours = append(out.Hours, hour)
		out.Hourly, out.Feels = append(out.Hourly, field("temp")), append(out.Feels, field("feels"))
		out.WindSpeed, out.WindFrom, out.WindGust = append(out.WindSpeed, field("wind")), append(out.WindFrom, field("wind_from")), append(out.WindGust, field("gust"))
		count++
	}
	if count == 0 {
		return s, 0
	}
	s.Hours = append(out.Hours, s.Hours...)
	s.Hourly, s.Feels = append(out.Hourly, s.Hourly...), append(out.Feels, s.Feels...)
	s.WindSpeed, s.WindFrom, s.WindGust = append(out.WindSpeed, s.WindSpeed...), append(out.WindFrom, s.WindFrom...), append(out.WindGust, s.WindGust...)
	return s, count
}

// missingRow is n values, each missing.
func missingRow(n int) []float64 {
	out := make([]float64, n)
	for i := range out { // P10-02
		out[i] = math.NaN()
	}
	return out
}
