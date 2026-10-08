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
	tuimaps "github.com/branden-thompson/go-tuimaps"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
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
	Hours:    72 * time.Hour,
	Days:     30 * 24 * time.Hour,
	MaxBytes: largeHistoryBound,
}

// omUVHourly is Open-Meteo's UV index over each field box, each hour it
// answered, up to the current one: replayed when it does not (D-167).
var omUVHourly = history.Dataset{
	Name: "openmeteo-uv", Version: 1, Step: time.Hour,
	Title:       "Open-Meteo, the UV index",
	Description: "Open-Meteo's UV index over each field box of a region, each hour it answered.",
	Fields:      []history.Field{{Name: "uv", Label: "UV index", Unit: "index", Decimals: 1}},
	Hours:       72 * time.Hour,
	Days:        30 * 24 * time.Hour,
	MaxBytes:    historyBound,
}

// omRainDays is Open-Meteo's rain and snow over each field box, each day it
// answered - today and the days ahead - keyed at the day's start: replayed in
// Forecast mode when it does not answer (D-168).
var omRainDays = history.Dataset{
	Name: "openmeteo-rain-days", Version: 1, Step: time.Hour,
	Title:       "Open-Meteo, rain and snow by day",
	Description: "Open-Meteo's forecast rain and snow over each field box of a region, each day it answered: the heaviest hour, the rain and the snowfall.",
	Fields: []history.Field{
		{Name: "peak", Label: "Heaviest hour", Unit: "mm/h", Decimals: 1},
		{Name: "rain", Label: "Rain", Unit: "mm", Decimals: 1},
		{Name: "snow", Label: "Snow", Unit: "cm", Decimals: 1},
	},
	Hours:    72 * time.Hour,
	Days:     30 * 24 * time.Hour,
	MaxBytes: historyBound,
}

// ndfdWaves is NDFD's significant wave height over each field box, an hour a
// record: its next hour kept as that hour's - NDFD's waves start there - so
// the current hour and the loop's past ones are drawn from it (D-194).
var ndfdWaves = history.Dataset{
	Name: "ndfd-waves", Version: 1, Step: time.Hour,
	Title:       "NDFD, wave height",
	Description: "The National Weather Service's significant wave height over each field box's sea, an hour a record.",
	Fields:      []history.Field{{Name: "waves", Label: "Wave height", Unit: "m", Decimals: 1}},
	Hours:       72 * time.Hour,
	Days:        30 * 24 * time.Hour,
	MaxBytes:    historyBound,
}

// epaUVCities is EPA's UV forecast for each city the map has read, an hour a
// record: each city a series, a point at the city, its name, state and zone in
// the record's document - kept for trends, and drawn when the city is in view
// again the same day, in any session (D-224, D-225).
var epaUVCities = history.Dataset{
	Name: "epa-uv-cities", Version: 1, Step: time.Hour,
	Title:       "EPA, the UV index by city",
	Description: "The EPA's hourly UV index forecast for each city the map has read, an hour a record.",
	Fields:      []history.Field{{Name: "uv", Label: "UV index", Unit: "index", Decimals: 0}},
	Hours:       72 * time.Hour,
	Days:        30 * 24 * time.Hour,
	MaxBytes:    historyBound,
}

// ndfdRainDays is NDFD's rain and snow over each field box, each day it gave:
// NDFD serves no history of its forecasts, so they are kept (W22.2, D-226).
var ndfdRainDays = history.Dataset{
	Name: "ndfd-rain-days", Version: 1, Step: time.Hour,
	Title:       "NDFD, rain and snow by day",
	Description: "The National Weather Service's forecast rain (liquid-equivalent) and snowfall over each field box of a region, each day it gave.",
	Fields: []history.Field{
		{Name: "rain", Label: "Rain", Unit: "mm", Decimals: 1},
		{Name: "snow", Label: "Snow", Unit: "cm", Decimals: 1},
	},
	Hours:    72 * time.Hour,
	Days:     30 * 24 * time.Hour,
	MaxBytes: historyBound,
}

// The datasets' byte bounds (D-143): each past what the longest retention - a
// year of hourly detail, five years of trends, which a document dataset keeps
// whole - holds at the rates measured (0.19.0 build log, W1.2), so none is cut
// short at those rates; the bound is the backstop for a rate nobody measured.
// The two largest, NDFD's current hour and AirNow's national file, have the
// larger.
const (
	largeHistoryBound = 4 << 30
	historyBound      = 512 << 20
)

// historyDatasets are every dataset the history holds: the Data tab's
// retention is theirs alike (D-175).
var historyDatasets = []history.Dataset{ndfdHourly, omUVHourly, omRainDays, ndfdWaves, epaUVCities, ndfdRainDays, nwsObservations, nwsAlerts, ndbcBuoys, coopsTides,
	hmsHotspots, firmsHotspots, wfigsIncidents, usgsQuakes, airnowHourly}

// historyEvery is how often the recorder looks for an hour to record.
const historyEvery = 5 * time.Minute

// historian records the history: the store, NDFD's current hour, and the
// regions to record.
type historian struct {
	store   *history.Store
	hour    func(context.Context, temperature.Lattice, time.Time) (temperature.Series, error)
	waves   func(context.Context, temperature.Lattice, time.Time) (temperature.Waves, error)  // NDFD's (D-194)
	totals  func(context.Context, temperature.Lattice, time.Time) (temperature.Totals, error) // NDFD's rain and snow (W22.2)
	air     func(context.Context, time.Time) ([]airquality.Area, error)                       // AirNow's national file (D-234)
	quakes  func(context.Context) []globalfeed.Event                                          // the USGS feed (D-234)
	regions func() []string
	pruned  atomic.Int64 // the hour last pruned, Unix
}

// startHistory opens the store at the retention chosen and records while ctx
// lives (D-172, D-175).
func (lp *livePipelines) startHistory(ctx context.Context, keep tty.HistoryRetention) {
	if lp.temp == nil {
		return
	}
	ndfd, ok := lp.temp.ndfd.(*temperature.NDFD)
	if !ok || ndfd == nil {
		return
	}
	h := &historian{store: history.Open(history.DefaultRoot(), time.Now, historyDatasets...), hour: ndfd.Hour, waves: ndfd.Waves, totals: ndfd.Totals,
		quakes: func(ctx context.Context) []globalfeed.Event { return lp.mapQuakes.fetch(ctx, historyQuakeFeed) }, regions: lp.historyRegions}
	if lp.airnow != nil {
		h.air = lp.airnow.Areas
	}
	lp.mu.Lock()
	lp.history = h.store
	lp.mu.Unlock()
	lp.applyHistory(keep)
	go func() {
		h.store.Bytes() // the store measured here, off the UI goroutine: the Data tab's reads are then kept counts (PF-2)
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
			h.record(ctx, temperature.NDFDLatticeFor(b.Name, b.Box), now)   // the lattice the map draws NDFD on (D-201)
			h.recordTotals(ctx, temperature.LatticeFor(b.Name, b.Box), now) // the lattice the map asks totals on
		}
	}
	h.recordAir(ctx, now)
	h.recordQuakes(ctx, now)
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

// record fetches and records one box's current hour on NDFD's lattice, when
// this instance claims it; its waves on the box's own, as the map merges them
// point by point with Open-Meteo Marine's (D-201, D-194).
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
	if had, ok := h.store.Get(ndfdHourly.Name, key, rec.At); ok && allMissing(rec.Values["feels"]) {
		if v, ok := onLattice(had, lat); ok && len(v["feels"]) == len(rec.Values["temp"]) && !allMissing(v["feels"]) {
			rec.Values["feels"] = v["feels"] // kept for this hour an hour ago (D-188)
		}
	}
	h.store.Put(ndfdHourly.Name, rec)
	if next, ok := nextFeels(s, rec); ok {
		h.store.Put(ndfdHourly.Name, next)
	}
	h.recordWaves(ctx, temperature.LatticeFor(lat.Name, lat.Box), now)
}

// recordWaves keeps NDFD's waves for a box, the next hour's as that hour's
// (D-194): NDFD's waves start at the next hour, and the hour it becomes is
// drawn from this. Nothing for a box NDFD gives no sea.
func (h *historian) recordWaves(ctx context.Context, lat temperature.Lattice, now time.Time) {
	if h.waves == nil {
		return
	}
	w, err := h.waves(ctx, lat, now)
	if err != nil {
		return // counted as nothing (D-124)
	}
	next := now.UTC().Truncate(time.Hour).Add(time.Hour)
	vals, ok := w.At(next)
	if !ok || allMissing(vals) {
		return
	}
	h.store.Put(ndfdWaves.Name, history.Record{Key: history.Key{Source: "ndfd", Place: lat.Name}, At: next, IssuedAt: now, Shape: shapeOf(lat), Values: map[string][]float64{"waves": vals}})
}

// historyQuakeFeed is the USGS feed the historian keeps: every quake of
// magnitude 1.0 and up over the past day, so each hour's is complete.
const historyQuakeFeed = "1.0_day"

// recordTotals keeps each day NDFD gives a box its rain and snow, once an
// hour, when this instance claims it: NDFD serves no history of its
// forecasts (W22.2, D-226).
func (h *historian) recordTotals(ctx context.Context, lat temperature.Lattice, now time.Time) {
	key := history.Key{Source: "ndfd", Place: lat.Name}
	if h.totals == nil || !h.store.Claim(ndfdRainDays.Name, key, now) {
		return
	}
	t, err := h.totals(ctx, lat, now)
	if err != nil {
		return // its claim goes stale, and it is tried again (D-124)
	}
	local := now.In(time.Local)
	for k := range temperature.Days { // a week (P10-02)
		if allMissing(t.QPF[k]) && allMissing(t.Snow[k]) {
			continue
		}
		h.store.Put(ndfdRainDays.Name, history.Record{Key: key, At: dayStart(local, k), IssuedAt: now.Truncate(time.Hour), Shape: shapeOf(lat),
			Values: map[string][]float64{"rain": t.QPF[k], "snow": t.Snow[k]}})
	}
}

// recordAir keeps AirNow's national file as the hour's one record, once an
// hour, when this instance claims it (D-234).
func (h *historian) recordAir(ctx context.Context, now time.Time) {
	hour := now.Truncate(time.Hour)
	if h.air == nil || !h.store.Claim(airnowHourly.Name, airSeries, hour) {
		return
	}
	areas, err := h.air(ctx, hour)
	if err != nil || len(areas) == 0 {
		return
	}
	airNational(h.store, areas, hour)
}

// recordQuakes keeps the USGS feed's quakes by their origin hour, once an
// hour, when this instance claims it (D-234).
func (h *historian) recordQuakes(ctx context.Context, now time.Time) {
	if h.quakes == nil || !h.store.Claim(usgsQuakes.Name, quakeSeries, now) {
		return
	}
	quakeHours(h.store, h.quakes(ctx), now)
}

// nextFeels is NDFD's feels-like for the hour after a record's, as that
// hour's own record (D-188): NDFD answers feels-like from the next hour, so
// an hour's record would never hold its own. False where NDFD gave none.
func nextFeels(s temperature.Series, rec history.Record) (history.Record, bool) {
	at := rec.At.Add(time.Hour)
	vals, ok := s.FeelsAt(at)
	if !ok || len(vals) != rec.Shape.Cols*rec.Shape.Rows || allMissing(vals) {
		return history.Record{}, false
	}
	return history.Record{Key: rec.Key, At: at, IssuedAt: rec.IssuedAt, Shape: rec.Shape, Values: map[string][]float64{"feels": vals}}, true
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
// place, a field not recorded missing; and how many were. A record on
// another lattice of the box is put on the series' (onLattice).
func withRecorded(s temperature.Series, past func(string, time.Time) (history.Record, bool), box string, anchor time.Time) (temperature.Series, int) {
	if past == nil {
		return s, 0
	}
	n := s.Lattice.Cols * s.Lattice.Rows
	if n <= 0 {
		return s, 0
	}
	var out temperature.Series
	out.Lattice = s.Lattice
	count := 0
	for back := pastHours; back >= 1; back-- { // three (P10-02)
		hour := anchor.Add(-time.Duration(back) * time.Hour)
		rec, ok := past(box, hour)
		if !ok {
			continue
		}
		values, ok := onLattice(rec, s.Lattice)
		if !ok || len(values["temp"]) != n {
			continue
		}
		field := func(name string) []float64 {
			if v := values[name]; len(v) == n {
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

// onLattice is a record's values on a lattice's points: as they are on the
// lattice they were recorded on, else put on its points - the history holds
// records of a box on more than one lattice (D-201), and each is drawn. A
// direction is the nearest point's, never blended. False for another box.
func onLattice(rec history.Record, l temperature.Lattice) (map[string][]float64, bool) {
	if rec.Shape == shapeOf(l) {
		return rec.Values, true
	}
	if rec.Shape.Box != l.Box || rec.Shape.Cols < 2 || rec.Shape.Rows < 2 {
		return nil, false
	}
	from := temperature.Lattice{Name: l.Name, Box: rec.Shape.Box, Cols: rec.Shape.Cols, Rows: rec.Shape.Rows}
	out := make(map[string][]float64, len(rec.Values))
	for k, v := range rec.Values { // a record's few fields (P10-02)
		switch {
		case len(v) != from.Cols*from.Rows:
		case k == "wind_from":
			out[k] = from.ResampleNearest(v, l)
		default:
			out[k] = from.Resample(v, l)
		}
	}
	return out, true
}

// missingRow is n values, each missing.
func missingRow(n int) []float64 {
	out := make([]float64, n)
	for i := range out { // P10-02
		out[i] = math.NaN()
	}
	return out
}

// historyKeeps are the Data tab's presets as durations (D-175): the hours'
// and the trends'. A key not among them is the default.
var historyKeeps = map[string]time.Duration{
	"72h": 72 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour, "90d": 90 * 24 * time.Hour,
	"1y": 365 * 24 * time.Hour, "5y": 5 * 365 * 24 * time.Hour,
}

// historyDurations is a retention as the store keeps it: the hours, and the
// rolled-up days past them - the defaults, 72 hours and 30 days, for a key
// not a preset.
func historyDurations(r tty.HistoryRetention) (hours, days time.Duration) {
	hours, days = ndfdHourly.Hours, ndfdHourly.Days
	if d, ok := historyKeeps[r.Hours]; ok {
		hours = d
	}
	if d, ok := historyKeeps[r.Trends]; ok {
		days = d
	}
	return hours, days
}

// historyStore is the store, or nil before it opens.
func (lp *livePipelines) historyStore() *history.Store {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.history
}

// applyHistory sets the running store's retention (D-175).
func (lp *livePipelines) applyHistory(r tty.HistoryRetention) {
	store := lp.historyStore()
	if store == nil {
		return
	}
	hours, days := historyDurations(r)
	for _, d := range historyDatasets { // two (P10-02)
		store.Retain(d.Name, hours, days)
	}
}

// setHistory writes the Data tab's choice and applies it at once.
func (lp *livePipelines) setHistory(r tty.HistoryRetention) {
	if err := config.Mutate(func(cfg *config.Config) error {
		cfg.HistoryHours, cfg.HistoryTrends = r.Hours, r.Trends
		return nil
	}); err != nil {
		return
	}
	lp.applyHistory(r)
}

// clearHistory empties the store: Clear history, after the Data tab's ARE
// YOU SURE (D-177). Nothing to clear is no failure.
func (lp *livePipelines) clearHistory() error {
	store := lp.historyStore()
	if store == nil {
		return nil
	}
	defer lp.forgetUsage() // the size, read again (U2-48)
	return store.Clear()
}

// usageFresh is how long the Data tab's words for the history's size are kept
// (U2-48): Settings draws every tab each frame - two or three reads a key
// press - and the size itself is the store's kept count, read without a walk
// (PF-2).
const usageFresh = 30 * time.Second

// usageCache is the history's size as last read, and when.
type usageCache struct {
	mu   sync.Mutex
	text string
	at   time.Time
}

// historyUsage is what the store holds and where, as the Data tab says it:
// read at most once in usageFresh, and again after Clear history.
func (lp *livePipelines) historyUsage() string {
	lp.usage.mu.Lock()
	defer lp.usage.mu.Unlock()
	if lp.usage.text != "" && time.Since(lp.usage.at) < usageFresh {
		return lp.usage.text
	}
	lp.usage.text, lp.usage.at = lp.readHistoryUsage(), time.Now()
	return lp.usage.text
}

// forgetUsage has the next read say the size again: the store changed.
func (lp *livePipelines) forgetUsage() {
	lp.usage.mu.Lock()
	lp.usage.text = ""
	lp.usage.mu.Unlock()
}

// readHistoryUsage is the Data tab's words for what the store holds, from
// its kept count, and where.
func (lp *livePipelines) readHistoryUsage() string {
	store := lp.historyStore()
	root := history.DefaultRoot()
	if store == nil || root == "" {
		return "No history is kept: there is no home directory to keep it in."
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(root, home) {
		root = "~" + strings.TrimPrefix(root, home)
	}
	return "Holds " + sizeWords(store.Bytes()) + ", in " + root + boundWords(store)
}

// boundWords names each dataset its byte bound has cut short of the chosen
// retention (D-143), its oldest records removed first; "" when none.
func boundWords(store *history.Store) string {
	var cut []string
	for _, d := range historyDatasets { // the datasets kept (P10-02)
		if store.Bounded(d.Name) {
			cut = append(cut, d.Title)
		}
	}
	if len(cut) == 0 {
		return ""
	}
	return ". Kept shorter than chosen, at its size limit, oldest first: " + strings.Join(cut, "; ") + "."
}

// historyCost says what keeping a longer window would take, for the Data tab's
// question (D-231).
func (lp *livePipelines) historyCost(trends bool, from, to string) string {
	return historyCostOf(lp.historyStore(), trends, from, to, time.Now())
}

// historyCostOf is the question's cost line: the bytes a longer window adds at
// the rate the history has grown since it began - an estimate, and said so.
func historyCostOf(store *history.Store, trends bool, from, to string, now time.Time) string {
	since, ok := store.Since()
	if !ok {
		return "The history is too new to measure what this costs yet: it grows as watchpost runs."
	}
	return "About " + sizeWords(historyExtra(store, trends, from, to, now)) + " more on disk, at the rate the history has grown since " + since.Format("2 January") + " (an estimate)."
}

// historyExtra is the bytes a longer window adds: each dataset's daily growth
// times the days added - for hourly detail each whole; for trends a value
// dataset's day rolled up to a twenty-fourth of its hours, a document
// dataset's whole, as the store keeps documents through the trends' window.
func historyExtra(store *history.Store, trends bool, from, to string, now time.Time) int64 {
	since, ok := store.Since()
	if !ok {
		return 0
	}
	days := max(1, now.Sub(since).Hours()/24)
	added := (historyKeeps[to] - historyKeeps[from]).Hours() / 24
	var extra float64
	for _, d := range historyDatasets { // the datasets kept (P10-02)
		rate := float64(store.BytesOf(d.Name)) / days
		if trends && len(d.Fields) > 0 {
			rate /= 24
		}
		extra += rate * max(0, added)
	}
	return int64(extra)
}

// sizeWords is a byte count as a person reads it: KB under a megabyte, MB with
// one decimal above.
func sizeWords(b int64) string {
	if b < 1<<20 {
		return strconv.FormatInt((b+1023)/1024, 10) + " KB"
	}
	return strconv.FormatFloat(float64(b)/(1<<20), 'f', 1, 64) + " MB"
}

// recordUV keeps each hour of a series' UV that Open-Meteo gave, up to the
// current one: issued at its own hour, so asking again rewrites nothing.
func recordUV(store *history.Store, box string, s temperature.Series, anchor time.Time) {
	if store == nil || len(s.UV) == 0 {
		return
	}
	shape := history.Shape{Box: s.Lattice.Box, Cols: s.Lattice.Cols, Rows: s.Lattice.Rows}
	for i, h := range s.Hours { // a series' hours (P10-02)
		if h.After(anchor) || i >= len(s.UV) || allMissing(s.UV[i]) {
			continue
		}
		store.Put(omUVHourly.Name, history.Record{Key: history.Key{Source: "openmeteo", Place: box}, At: h, IssuedAt: h, Shape: shape, Values: map[string][]float64{"uv": s.UV[i]}})
	}
}

// replayUV is a box's UV recorded for the current hour and the pastHours
// before it, each drawn during its own hour; none where nothing matches.
func replayUV(store *history.Store, box string, lat temperature.Lattice, anchor time.Time) []tuimaps.Overlay {
	if store == nil {
		return nil
	}
	shape := history.Shape{Box: lat.Box, Cols: lat.Cols, Rows: lat.Rows}
	var out []tuimaps.Overlay
	for back := pastHours; back >= 0; back-- { // four (P10-02)
		h := anchor.Add(-time.Duration(back) * time.Hour)
		rec, ok := store.Get(omUVHourly.Name, history.Key{Source: "openmeteo", Place: box}, h)
		if !ok || rec.Shape != shape {
			continue
		}
		o, ok := fieldGrid(tty.UVLayer+"/"+box+"/"+h.UTC().Format("2006-01-02T15"), lat, rec.Values["uv"], h, anchor, tuimaps.UVGrid)
		if !ok {
			continue
		}
		o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
		out = append(out, o)
	}
	return out
}
