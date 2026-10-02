package app

// maptemp.go — the map's temperature (0.18.0 W10, F-182, D-93 to D-98): a
// lattice a radar box - the same fixed boxes as the radar's, never the view
// (D-47) - from Open-Meteo in Radar mode (D-96) or the listener's choice in
// Forecast mode (D-93: NDFD by default), interpolated into grids the library
// draws, each grid carrying its span: Radar mode's every hour, Forecast
// mode's Now and each day's high and low (go-tuiMaps L-15.1).

import (
	"context"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/units"
)

// The temperature, registered: OFF BY DEFAULT (D-99), and loaded in the
// background while the map is open so switching it on in the Overlays menu
// draws at once. The wind rides the same requests, off by default too (D-110);
// its cost is temperature's, so it registers none of its own.
func init() {
	registerMapLayer(mapLayer{key: tty.TemperatureLayer, label: "Temperature", on: false, cost: tempLayerCost})
	registerMapLayer(mapLayer{key: tty.FeelsLayer, label: "Feels like", on: false, cost: func(mapInputs) (int64, int) { return 0, 0 }}) // D-119: in temperature's requests
	registerMapLayer(mapLayer{key: tty.WindLayer, label: "Wind", on: false, cost: func(mapInputs) (int64, int) { return 0, 0 }})
	// UV and Air quality ride temperature's source (D-137, D-139): off by
	// default, after wind in the Overlays menu.
	registerMapLayer(mapLayer{key: tty.UVLayer, label: "UV", on: false, cost: func(mapInputs) (int64, int) { return 0, 0 }})      // its chip is the answer's, while it draws (D-183)
	registerMapLayer(mapLayer{key: tty.AirLayer, label: "Air quality", on: false, cost: airLayerCost, chips: []string{"AIRNOW"}}) // AirNow's contours and monitors (D-193)
}

// tempSources is the temperature the app holds: both sources over one
// hardened client.
type tempSources struct {
	ndfd, om temperature.Source
	gate     *temperature.QuotaGate // Open-Meteo's asks, held while a quota is spent (W18.1, D-165)
	uvCities *uvCities              // UV's first source: EPA's index for the cities in view (D-167, D-186)
	land     *landPoints            // the points Open-Meteo Marine answered nothing for (D-194)
	rain     *temperature.OpenMeteo // the rain and snow, Open-Meteo's always (D-118); the waves beyond NDFD (D-125)
	waves    *temperature.NDFD      // the waves where NDFD reaches (D-125)
}

// tempSourcesOver is both sources over one temperature client (overClient).
func tempSourcesOver(c *httpx.Client) *tempSources {
	return tempSourcesAt(c, "", "", temperature.DefaultQuotaState())
}

// tempSourcesAt is both sources over one getter, at the hosts given ("" the
// production ones): Open-Meteo's asks go through the quota gate - its holds
// shared with every instance through state ("" for none, design 4b) - NDFD's
// not.
func tempSourcesAt(c temperature.Getter, omBase, ndfdBase, state string) *tempSources {
	gate := temperature.NewSharedQuotaGate(c, time.Now, state)
	om, ndfd := temperature.NewOpenMeteo(gate, omBase), temperature.NewNDFD(c, ndfdBase)
	return &tempSources{ndfd: ndfd, om: om, gate: gate, rain: om, waves: ndfd, uvCities: &uvCities{epa: uv.NewEPA(c, "")}, land: &landPoints{}}
}

// quotaSpent is Open-Meteo's spent quota as the map says it, or nil.
func (ts *tempSources) quotaSpent() *tty.MapQuota {
	if ts == nil || ts.gate == nil {
		return nil
	}
	q, ok := ts.gate.Refused()
	if !ok {
		return nil
	}
	return &tty.MapQuota{Source: "Open-Meteo", Period: q.Period, Resets: q.Resets}
}

// sourceFor is the mode's source: Open-Meteo in Radar mode, the one with the
// past hours (D-96); in Forecast mode the listener's choice where it covers,
// Open-Meteo otherwise.
func (ts *tempSources) sourceFor(ask tty.MapAsk) temperature.Source {
	if ask.TempNDFD && ts.ndfd.Covers(ask.Region) { // both modes, the listener's (D-190); keyless first (D-185)
		return ts.ndfd
	}
	return ts.om
}

// answeredFor is a source that keeps the boxes it answered for: Open-Meteo
// as NDFD's filler, whose forecast UV's grid may ride (D-191).
type answeredFor struct {
	temperature.Source
	mu    sync.Mutex
	boxes map[string]bool
}

// Fetch asks the source, and keeps the box when it answers.
func (a *answeredFor) Fetch(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Series, error) {
	ask := a.Source.Fetch // the wrapped source's, not this one
	s, err := ask(ctx, l, now)
	if err == nil {
		a.mu.Lock()
		a.boxes[l.Name] = true
		a.mu.Unlock()
	}
	return s, err
}

// uvAsked is where Open-Meteo's forecast was asked anyway, which UV's grid
// may ride (D-191): every box where it is the source, else those it filled.
func uvAsked(isSource bool, filled *answeredFor) func(box string) bool {
	if isSource {
		return func(string) bool { return true }
	}
	return filled.answered
}

// answered reports whether the source answered for a box.
func (a *answeredFor) answered(box string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.boxes[box]
}

// tempChips are the chips temperature, feels-like and wind - one request -
// are credited by on their badges (D-133): Open-Meteo's alone, NDFD's with
// Open-Meteo's where it filled what NDFD lacked, or NDFD's alone.
func tempChips(source string, filled bool) map[string][]string {
	chips := []string{"NDFD"}
	switch {
	case source == "Open-Meteo":
		chips = []string{"O-METEO"}
	case filled:
		chips = []string{"NDFD", "O-METEO"}
	}
	return map[string][]string{tty.TemperatureLayer: chips, tty.FeelsLayer: chips, tty.WindLayer: chips}
}

// tempFrameNote is what Radar mode says of its temperature (D-96).
const tempFrameNote = "Each radar frame draws its own hour's temperature."

// mapTemperature is the window's temperature for the mode, every grid built
// with its span. Where the chosen source has nothing for a day, Open-Meteo
// fills it (D-100).
func (lp *livePipelines) mapTemperature(ctx context.Context, ask tty.MapAsk) tty.MapTemperature {
	if lp.temp == nil || ask.Region == "" {
		return tty.MapTemperature{}
	}
	lp.lastMapRegion.Store(ask.Region) // the history records the map's region too (D-172)
	src := lp.temp.sourceFor(ask)
	var fill temperature.Source
	filled := &answeredFor{Source: lp.temp.om, boxes: map[string]bool{}}
	if src != lp.temp.om {
		fill = filled // the boxes it answered for, UV's grid rides along there (D-191)
	}
	rescue := &fallback{past: recordedHour(lp.historyStore())} // NDFD has no hour before the current one: the history's are the loop's past (W19.1)
	if src == lp.temp.om && lp.temp.ndfd != nil && lp.temp.ndfd.Covers(ask.Region) {
		rescue.src = lp.temp.ndfd // Open-Meteo refused: NDFD draws what it can, the history the hours before (W18.2, W18.3b)
	}
	now := time.Now()
	t := buildTemperature(ctx, src, fill, ask, now, rescue)
	if ask.Forecast && lp.temp.rain != nil { // Forecast mode's rain and snow (W12.3): held, whether or not its row is on (D-99)
		rescue := &rainRescue{store: lp.historyStore()} // recorded days, then NDFD's totals where it reaches (D-168)
		if lp.temp.waves != nil && lp.temp.waves.Covers(ask.Region) {
			rescue.ndfd = lp.temp.waves // NDFD's client
		}
		t = withRainDays(ctx, t, lp.temp.rain, ask, now, rescue)
	}
	if lp.temp.waves != nil && lp.temp.rain != nil && ask.Region != geo.RegionSamoa { // the waves (D-125): held as the rest is (D-99); NDFD has no Samoa
		t = withWaves(ctx, t, lp.temp.waves, lp.temp.rain, ask, now, waveKeep{store: lp.historyStore(), land: lp.temp.land}) // NDFD first, the history its hours, Open-Meteo past its reach (D-194)
	}
	if lp.temp.rain != nil { // Open-Meteo: the UV and the model's US AQI (D-137, D-139)
		t = withUV(ctx, t, lp.temp.rain, uvAsked(src == lp.temp.om, filled), ask, now, lp.historyStore(), lp.temp.uvCities) // EPA's cities first (D-186); valid UV kept, and replayed when refused (D-167)
		t = withAir(ctx, t, lp.airnow, ask, now)                                                                            // AirNow's contours (D-193)
	}
	t.Quota = lp.temp.quotaSpent() // the map says it (D-165)
	return t
}

// buildTemperature fetches each box's lattice and makes its grids. EVERY
// ANSWER WITHIN AN HOUR IS THE SAME (UAT-2 U2-13): what a grid says of itself
// is worked out from the hour's start, never the clock, so the window hands
// nothing in again and nothing blinks.
//
// A BOX OPEN-METEO DOES NOT ANSWER IS NDFD'S, where rescue is given (W18.2,
// D-165): a spent quota refuses every box. NDFD has no hour before the
// current one, so in Radar mode its current hour is drawn under the loop's
// earlier frames too (D-166's cold start) - the chips say NDFD.
func buildTemperature(ctx context.Context, src, fill temperature.Source, ask tty.MapAsk, now time.Time, rescue *fallback) tty.MapTemperature {
	t := tempBuild{ctx: ctx, src: src, fill: fill, rescue: rescue, ask: ask, now: now,
		out: tty.MapTemperature{Source: src.Name()}, missing: map[string]bool{}, credit: src.Name() == "Open-Meteo"}
	boxes := fieldBoxes(ask.Region, ask.View)
	if len(boxes) == 0 {
		t.out.Notes = []string{"No temperature is drawn for " + ask.Region + "."}
		return t.out
	}
	t.anchor = askAnchor(ask, now)
	t.unit = tuimaps.Celsius
	if ask.Fahrenheit {
		t.unit = tuimaps.Fahrenheit
	}
	for _, b := range boxes {
		s, from, err := t.fetch(b)
		if err != nil {
			t.refused(b.Name, err)
			continue
		}
		if !ask.Forecast {
			keyless := from == fromRescue || (src.Name() == "NDFD" && from != fromFill) // NDFD drew this box: it has no hour before the current one
			t.radar(s, b.Name, keyless)
			continue
		}
		t.forecast(s, b)
	}
	return t.finish(len(boxes))
}

// tempBuild is buildTemperature's work: its sources and ask, what it has
// drawn, and the tallies its chips and notes are made from.
type tempBuild struct {
	ctx         context.Context
	src, fill   temperature.Source
	rescue      *fallback
	ask         tty.MapAsk
	now, anchor time.Time
	unit        tuimaps.Unit
	out         tty.MapTemperature
	missing     map[string]bool // the notes, each said once
	credit      bool            // Open-Meteo drew some of it
	fellBack    int             // boxes the fill drew
	rescued     int             // boxes the rescue drew
	replayed    int             // hours from the history
}

// boxFrom is which source drew a box.
type boxFrom int

const (
	fromSource boxFrom = iota
	fromFill
	fromRescue
)

// fetch is a box's series: the source's on its own lattice (D-201: NDFD's
// twice as dense, keyless; Open-Meteo's 80 points, each billed, D-185); else
// the fill's, Open-Meteo's (D-101: NDFD refuses Hawaii's whole lattice,
// which straddles its grid's edge); else the rescue's, NDFD's.
func (t *tempBuild) fetch(b fieldBox) (temperature.Series, boxFrom, error) {
	omLat, ndfdLat := temperature.LatticeFor(b.Name, b.Box), temperature.NDFDLatticeFor(b.Name, b.Box)
	lat := omLat
	if t.src.Name() == "NDFD" {
		lat = ndfdLat
	}
	s, err := t.src.Fetch(t.ctx, lat, t.now)
	if err == nil {
		return s, fromSource, nil
	}
	if t.fill != nil {
		if s, err = t.fill.Fetch(t.ctx, omLat, t.now); err == nil {
			t.fellBack++
			t.credit = true
			t.missing[t.src.Name()+" did not answer for part of the map; Open-Meteo is drawn there."] = true
			return s, fromFill, nil
		}
	}
	if t.rescue != nil && t.rescue.src != nil {
		if s, err = t.rescue.src.Fetch(t.ctx, ndfdLat, t.now); err == nil {
			t.rescued++
			return s, fromRescue, nil
		}
	}
	return s, fromSource, err
}

// refused says a box no source drew: NDFD's in a note naming the Setting that
// draws another (D-124), any other source's to the diagnostics.
func (t *tempBuild) refused(box string, err error) {
	if t.src.Name() == "NDFD" {
		t.missing["Temperature is unavailable: NDFD did not answer. Settings → Maps → Temperature: Open-Meteo draws it instead."] = true
		return
	}
	t.out.Problems = append(t.out.Problems, "Temperature: "+t.src.Name()+" did not answer for "+box+" - "+err.Error())
}

// radar is a box's hours for Radar mode: temperature, feels-like and wind,
// with the hours before the current one from the history where NDFD drew it
// (D-166, W19.1), and every observed frame drawn (U2-55 to U2-57).
func (t *tempBuild) radar(s temperature.Series, box string, keyless bool) {
	horizon := radarHorizon(t.ask, t.anchor)
	temp := func(values [][]float64) func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
		return func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
			return unitGrid(id, s.Lattice.InterpolateWide(values[i]), convertIf(t.unit == tuimaps.Fahrenheit, units.FahrenheitOf), t.unit, valid, t.anchor, tuimaps.TemperatureGrid)
		}
	}
	if keyless && t.rescue != nil {
		var past int
		s, past = withRecorded(s, t.rescue.past, box, t.anchor)
		t.replayed += past
	}
	hours, feels, wind := hourGrids(tty.TemperatureLayer, box, s.Hours, len(s.Hourly), t.anchor, horizon, temp(s.Hourly)),
		hourGrids(tty.FeelsLayer, box, s.Hours, len(s.Feels), t.anchor, horizon, temp(s.Feels)), windHourGrids(s, box, t.anchor, horizon, t.ask.Fahrenheit)
	fillPast(hours, t.anchor)
	fillPast(feels, t.anchor)
	fillPast(wind, t.anchor)
	t.out.Overlays, t.out.Feels, t.out.Wind = append(t.out.Overlays, hours...), append(t.out.Feels, feels...), append(t.out.Wind, wind...)
}

// forecast is a box's days for Forecast mode: an empty Today from the hours
// recorded (D-189), else Open-Meteo for that day; the feels-like hour now
// from the history, NDFD's starting at the next hour (D-188).
func (t *tempBuild) forecast(s temperature.Series, b fieldBox) {
	if t.rescue != nil && recordedToday(&s, t.rescue.past, b.Name, t.anchor) {
		t.replayed++
	}
	if t.fill != nil && fillDays(t.ctx, &s, t.fill, temperature.LatticeFor(b.Name, b.Box), t.now, &t.out) {
		t.credit = true
	}
	if t.rescue != nil && recordedFeelsNow(&s, t.rescue.past, b.Name, t.anchor) {
		t.replayed++
	}
	forecastGrids(&t.out, s, b.Name, t.anchor, t.unit, t.missing)
	feelsForecastGrids(&t.out, s, b.Name, t.anchor, t.unit)
	windForecastGrids(&t.out, s, b.Name, t.anchor, t.ask.Fahrenheit)
}

// finish names the sources in the chips and says the notes: the fill's chip
// where it drew every box; NDFD's where it drew where Open-Meteo did not
// (W18.2), alone or beside Open-Meteo's.
func (t *tempBuild) finish(boxes int) tty.MapTemperature {
	out, credit := t.out, t.credit
	if t.fellBack == boxes {
		out.Source = t.fill.Name()
	}
	switch {
	case t.rescued > 0 && t.rescued == boxes:
		out.Source, credit = t.rescue.src.Name(), false
	case t.rescued > 0:
		out.Source, credit = t.rescue.src.Name(), true
	}
	out.Chips = tempChips(out.Source, credit)                                                                                                                                                                                                  // the credit is the badge's, in full the Status window's (D-131)
	for key, drew := range map[string]int{tty.TemperatureLayer: len(out.Overlays) + len(out.High) + len(out.Low), tty.FeelsLayer: len(out.Feels) + len(out.FeelsHigh) + len(out.FeelsLow), tty.WindLayer: len(out.Wind) + len(out.WindDays)} { // three (P10-02)
		if drew == 0 {
			delete(out.Chips, key) // a layer drawing nothing names no source (D-183)
		} else if t.replayed > 0 {
			out.Chips[key] = append(append([]string(nil), out.Chips[key]...), recordedChip) // some of it from the history (D-173)
		}
	}
	var said []string
	for n := range t.missing {
		said = append(said, n)
	}
	sort.Strings(said)
	out.Notes = append(out.Notes, said...)
	return out
}

// fillDays puts Open-Meteo's values in every day the series has nothing for
// (D-100), asking it only when one is empty, and marks each day filled;
// true when anything was. Open-Meteo is asked on its own lattice and its
// values put on the series' points (D-201): NDFD's lattice is denser.
func fillDays(ctx context.Context, s *temperature.Series, fill temperature.Source, lat temperature.Lattice, now time.Time, out *tty.MapTemperature) bool {
	var got *temperature.Series
	filled := false
	for k := range temperature.Days {
		for _, side := range []struct {
			name string
			vals *[]float64
			from func(*temperature.Series) []float64
		}{{"high", &s.High[k], func(f *temperature.Series) []float64 { return f.High[k] }},
			{"low", &s.Low[k], func(f *temperature.Series) []float64 { return f.Low[k] }},
			{"wind", &s.PeakSpeed[k], func(f *temperature.Series) []float64 { return f.PeakSpeed[k] }},
			{"feelshigh", &s.FeelsHigh[k], func(f *temperature.Series) []float64 { return f.FeelsHigh[k] }},
			{"feelslow", &s.FeelsLow[k], func(f *temperature.Series) []float64 { return f.FeelsLow[k] }}} {
			if !allMissing(*side.vals) {
				continue
			}
			if got == nil {
				f, err := fill.Fetch(ctx, lat, now)
				if err != nil {
					return filled
				}
				got = &f
			}
			if allMissing(side.from(got)) {
				continue
			}
			onto := func(v []float64) []float64 { return got.Lattice.Resample(v, s.Lattice) }
			*side.vals = onto(side.from(got))
			if side.name == "wind" {
				s.PeakFrom[k], s.PeakGust[k] = got.Lattice.ResampleNearest(got.PeakFrom[k], s.Lattice), onto(got.PeakGust[k]) // the direction - never blended - and the gust with the speed they came with (D-136)
			}
			if out.Filled == nil {
				out.Filled = map[string]bool{}
			}
			out.Filled[strconv.Itoa(k)+"/"+side.name], filled = true, true
		}
	}
	return filled
}

// recordedFeelsNow puts the history's recorded hour in the current hour's
// feels-like where the source has none - NDFD answers it from the next hour,
// and the recorder keeps that next hour as the hour's own (D-188) - and on a
// cold start NDFD's next hour; true when the history's was drawn.
func recordedFeelsNow(s *temperature.Series, past func(string, time.Time) (history.Record, bool), box string, anchor time.Time) bool {
	if vals, ok := s.FeelsAt(anchor); ok && !allMissing(vals) {
		return false
	}
	i := s.HourIndex(anchor)
	if i < 0 || i >= len(s.Feels) {
		return false
	}
	n := s.Lattice.Cols * s.Lattice.Rows
	if past != nil {
		if rec, ok := past(box, anchor); ok {
			if v, ok := onLattice(rec, s.Lattice); ok && len(v["feels"]) == n && !allMissing(v["feels"]) {
				s.Feels[i] = v["feels"]
				return true
			}
		}
	}
	if next, ok := s.FeelsAt(anchor.Add(time.Hour)); ok && len(next) == n {
		s.Feels[i] = next // the cold start: NDFD's next hour
	}
	return false
}

// recordedToday fills Today's empty high, low, feels-like and peak wind from
// the hours the history recorded since the day began - their highest and
// lowest (D-189): NDFD drops a day's maximum once it has passed. True when
// anything was.
func recordedToday(s *temperature.Series, past func(string, time.Time) (history.Record, bool), box string, anchor time.Time) bool {
	if past == nil {
		return false
	}
	n := s.Lattice.Cols * s.Lattice.Rows
	high, low, feelsHigh, feelsLow, wind := missingValues(n), missingValues(n), missingValues(n), missingValues(n), missingValues(n)
	day := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, anchor.Location())
	for h := day; !h.After(anchor); h = h.Add(time.Hour) { // a day's hours (P10-02)
		rec, ok := past(box, h)
		if !ok {
			continue
		}
		v, ok := onLattice(rec, s.Lattice)
		if !ok {
			continue
		}
		extremes(high, low, v["temp"])
		extremes(feelsHigh, feelsLow, v["feels"])
		extremes(wind, nil, v["wind"])
	}
	filled := false
	for _, side := range []struct {
		into *[]float64
		from []float64
	}{{&s.High[0], high}, {&s.Low[0], low}, {&s.FeelsHigh[0], feelsHigh}, {&s.FeelsLow[0], feelsLow}, {&s.PeakSpeed[0], wind}} { // five (P10-02)
		if allMissing(*side.into) && !allMissing(side.from) {
			*side.into, filled = side.from, true
		}
	}
	return filled
}

// extremes widens hi and lo, point by point, to an hour's values.
func extremes(hi, lo, vals []float64) {
	for p, v := range vals { // a lattice's points (P10-02)
		if math.IsNaN(v) || p >= len(hi) {
			continue
		}
		if math.IsNaN(hi[p]) || v > hi[p] {
			hi[p] = v
		}
		if lo != nil && (math.IsNaN(lo[p]) || v < lo[p]) {
			lo[p] = v
		}
	}
}

// missingValues is n values, every one missing.
func missingValues(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	return out
}

// shapeOf is a lattice's shape as the history keeps it.
func shapeOf(l temperature.Lattice) history.Shape {
	return history.Shape{Box: l.Box, Cols: l.Cols, Rows: l.Rows}
}

// allMissing reports whether a source had nothing at any point.
func allMissing(vals []float64) bool {
	for _, v := range vals {
		if !math.IsNaN(v) {
			return false
		}
	}
	return true
}

// fieldBox is a fixed box temperature and wind are asked for.
type fieldBox struct {
	Name string
	geo.Box
}

// fieldBoxes are the fixed boxes a view's temperature and wind are asked for
// (D-111) - never the view itself (D-47). In the lower 48 the radar's boxes;
// outside it the whole map region, so no part of a region the map shows goes
// without (MRMS's box ends inside Hawaii's map), split at the
// antimeridian, which a grid cannot cross (Alaska).
func fieldBoxes(region string, view geo.Box) []fieldBox {
	if region == geo.RegionContiguous {
		// THE RADAR'S BOXES, THE OUTER ONES GROWN TO THE REGION'S EDGES (UAT-2
		// U2-30): the radar stops at 126W and 65W and the region reaches 130W
		// and 64W - on the radar's boxes alone the waves stop at a line in the Pacific.
		reach, whole := geo.Box{W: 180, S: 90, E: -180, N: -90}, regionBox(region)
		for _, b := range radar.BoxesFor(region, geo.Box{W: -180, S: -90, E: 180, N: 90}) {
			reach = geo.Box{W: min(reach.W, b.W), S: min(reach.S, b.S), E: max(reach.E, b.E), N: max(reach.N, b.N)}
		}
		var out []fieldBox
		for _, b := range radar.BoxesFor(region, view) {
			fb := fieldBox{Name: b.Name, Box: geo.Box{W: b.W, S: b.S, E: b.E, N: b.N}}
			if b.W == reach.W {
				fb.W = min(fb.W, whole.W)
			}
			if b.E == reach.E {
				fb.E = max(fb.E, whole.E)
			}
			if b.S == reach.S {
				fb.S = min(fb.S, whole.S)
			}
			if b.N == reach.N {
				fb.N = max(fb.N, whole.N)
			}
			out = append(out, fb)
		}
		return out
	}
	for _, r := range geo.Regions() {
		if r.Name != region {
			continue
		}
		name := strings.ToLower(strings.Fields(r.Name)[0])
		if r.W > r.E { // across the antimeridian: two boxes
			return []fieldBox{{name + "-w", geo.Box{W: r.W, S: r.S, E: 180, N: r.N}}, {name + "-e", geo.Box{W: -180, S: r.S, E: r.E, N: r.N}}}
		}
		return []fieldBox{{name, geo.Box{W: r.W, S: r.S, E: r.E, N: r.N}}}
	}
	return nil
}

// regionBox is a region's box, the zero box for a region not known.
func regionBox(name string) geo.Box {
	for _, r := range geo.Regions() {
		if r.Name == name {
			return geo.Box{W: r.W, S: r.S, E: r.E, N: r.N}
		}
	}
	return geo.Box{}
}

// askAnchor is the hour an ask is drawn from: the listener's, or with none
// given, the current hour.
func askAnchor(ask tty.MapAsk, now time.Time) time.Time {
	if ask.Anchor.IsZero() {
		return now.Truncate(time.Hour)
	}
	return ask.Anchor
}

// hourGrids are a layer's Radar-mode grids - temperature's, feels-like's
// (D-119), wind's, UV's and air's, waves' - every hour to the loop's
// horizon, each drawn during its hour (D-96). n is how many hours have
// values; grid builds hour i's, under its id and valid time.
func hourGrids(layer, box string, hours []time.Time, n int, anchor, horizon time.Time, grid func(i int, id string, valid time.Time) (tuimaps.Overlay, bool)) []tuimaps.Overlay {
	var out []tuimaps.Overlay
	for i, h := range hours {
		if h.After(horizon) || i >= n {
			continue // past the loop's hours ahead: no radar frame is in it (U2-32)
		}
		if o, ok := grid(i, layer+"/"+box+"/"+h.UTC().Format("2006-01-02T15"), stampOf(h, anchor)); ok {
			o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
			out = append(out, o)
		}
	}
	return out
}

// fillPast draws something on every frame of the loop (UAT-2 U2-55 to
// U2-57): an hour with no grid of its own - before the current one NDFD has
// none and the history only what it recorded (D-166); NDFD's current hour
// can come back with no wind - is drawn from the next newer grid, back to
// pastHours before the hour - so no frame of the loop, observed or ahead, is
// drawn with nothing.
func fillPast(grids []tuimaps.Overlay, anchor time.Time) {
	order := make([]int, 0, len(grids))
	for i := range grids { // bounded by the grids (P10-02)
		order = append(order, i)
	}
	slices.SortFunc(order, func(a, b int) int { return grids[a].During.From.Compare(grids[b].During.From) })
	reach := anchor.Add(-pastHours * time.Hour) // the earliest moment nothing has yet been drawn for
	for _, i := range order {                   // the past and the current hour (P10-02)
		if grids[i].During.From.After(reach) {
			grids[i].During.From = reach
		}
		if !grids[i].During.Until.IsZero() {
			reach = grids[i].During.Until.Add(time.Nanosecond)
		}
	}
}

// forecastDays are Forecast mode's steps from an anchor: Now's, and each
// day's as far as both the steps and the sources' days reach (D-94).
func forecastDays(anchor time.Time) (now tty.ForecastStep, days []tty.ForecastStep) {
	steps := tty.ForecastSteps(anchor)
	return steps[0], steps[1:min(len(steps), 1+temperature.Days)]
}

// radarHorizon is the last hour Radar mode's hourly fields draw: the loop's
// hours ahead past the current one (D-113). Stopping at the current hour,
// the fields would come and go as the loop plays into the forecast (UAT-2
// U2-32).
func radarHorizon(ask tty.MapAsk, anchor time.Time) time.Time {
	return anchor.Add(time.Duration(ask.RadarAhead) * time.Hour)
}

// stampOf is when an hour's grid says its data was valid: the hour, or for
// an hour ahead the current one - the forecast was made now, and a time
// ahead would leave it no currency (the library's refusal, U2-29's kind).
func stampOf(h, anchor time.Time) time.Time {
	if h.After(anchor) {
		return anchor
	}
	return h
}

// forecastGrids are Forecast mode's grids: Now, and each day's high and low,
// each drawn during its step (D-94, D-97). A day no source has anything for
// is said, not drawn empty.
func forecastGrids(out *tty.MapTemperature, s temperature.Series, box string, anchor time.Time, unit tuimaps.Unit, missing map[string]bool) {
	nowStep, days := forecastDays(anchor)
	if vals, _, ok := s.HourAt(anchor); ok {
		if o, ok := unitGrid(tty.TemperatureLayer+"/"+box+"/now", s.Lattice.InterpolateWide(vals), convertIf(unit == tuimaps.Fahrenheit, units.FahrenheitOf), unit, anchor, anchor, tuimaps.TemperatureGrid); ok {
			o.During = nowStep.Span
			out.Overlays = append(out.Overlays, o)
		}
	}
	for k, step := range days {
		for _, side := range []struct {
			name string
			vals []float64
			into *[]tuimaps.Overlay
		}{{"high", s.High[k], &out.High}, {"low", s.Low[k], &out.Low}} {
			o, ok := unitGrid(tty.TemperatureLayer+"/"+box+"/d"+strconv.Itoa(k)+"/"+side.name, s.Lattice.InterpolateWide(side.vals), convertIf(unit == tuimaps.Fahrenheit, units.FahrenheitOf), unit, anchor, anchor, tuimaps.TemperatureGrid)
			if !ok {
				missing["No "+side.name+" for "+step.Label+" from this source: it has passed, or is past the source's reach."] = true
				continue
			}
			o.During = step.Span
			*side.into = append(*side.into, o)
		}
	}
}

// feelsForecastGrids are Forecast mode's feels-like (D-119): Now, and each
// day's high and low, each during its step, as temperature's are. A day
// without is simply not drawn: temperature's grids say what is missing.
func feelsForecastGrids(out *tty.MapTemperature, s temperature.Series, box string, anchor time.Time, unit tuimaps.Unit) {
	nowStep, days := forecastDays(anchor)
	if vals, ok := s.FeelsAt(anchor); ok {
		if o, ok := unitGrid(tty.FeelsLayer+"/"+box+"/now", s.Lattice.InterpolateWide(vals), convertIf(unit == tuimaps.Fahrenheit, units.FahrenheitOf), unit, anchor, anchor, tuimaps.TemperatureGrid); ok {
			o.During = nowStep.Span
			out.Feels = append(out.Feels, o)
		}
	}
	for k, step := range days {
		for _, side := range []struct {
			name string
			vals []float64
			into *[]tuimaps.Overlay
		}{{"high", s.FeelsHigh[k], &out.FeelsHigh}, {"low", s.FeelsLow[k], &out.FeelsLow}} {
			if o, ok := unitGrid(tty.FeelsLayer+"/"+box+"/d"+strconv.Itoa(k)+"/"+side.name, s.Lattice.InterpolateWide(side.vals), convertIf(unit == tuimaps.Fahrenheit, units.FahrenheitOf), unit, anchor, anchor, tuimaps.TemperatureGrid); ok {
				o.During = step.Span
				*side.into = append(*side.into, o)
			}
		}
	}
}

// windHourGrids are Radar mode's wind grids: every hour up to the current
// one, each drawn during its hour (D-108), as temperature's are.
func windHourGrids(s temperature.Series, box string, anchor, horizon time.Time, mph bool) []tuimaps.Overlay {
	return hourGrids(tty.WindLayer, box, s.Hours, len(s.WindSpeed), anchor, horizon, func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
		var gust []float64
		if i < len(s.WindGust) {
			gust = s.WindGust[i]
		}
		return windGrid(id, s.Lattice, s.WindSpeed[i], s.WindFrom[i], gust, mph, valid, anchor)
	})
}

// windForecastGrids are Forecast mode's wind grids: Now's wind, and each
// day's peak with its dominant direction, each during its step (D-108).
func windForecastGrids(out *tty.MapTemperature, s temperature.Series, box string, anchor time.Time, mph bool) {
	nowStep, days := forecastDays(anchor)
	if speed, from, _, ok := s.WindAt(anchor); ok {
		gust, _ := s.GustAt(anchor)
		if o, ok := windGrid(tty.WindLayer+"/"+box+"/now", s.Lattice, speed, from, gust, mph, anchor, anchor); ok {
			o.During = nowStep.Span
			out.Wind = append(out.Wind, o)
		}
	}
	for k, step := range days {
		if o, ok := windGrid(tty.WindLayer+"/"+box+"/d"+strconv.Itoa(k), s.Lattice, s.PeakSpeed[k], s.PeakFrom[k], s.PeakGust[k], mph, anchor, anchor); ok {
			o.During = step.Span
			out.WindDays = append(out.WindDays, o)
		}
	}
}

// gustMargin is how far a gust must beat the sustained wind to be said:
// 10 mph, the METAR rule (D-136), in km/h.
const gustMargin = 16.09344

// windGrid is one lattice's wind as the library's vector grid, in mph or
// km/h as the listener's units are, its gusts said where they beat the
// sustained wind by gustMargin (D-136); false when no point has any.
func windGrid(id string, l temperature.Lattice, speed, from, gust []float64, mph bool, valid, anchor time.Time) (tuimaps.Overlay, bool) {
	f, dirs := l.InterpolateWindWide(speed, from) // blank only where all four points are (D-201)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	var gusts []float64
	if len(gust) == len(speed) && !allMissing(gust) {
		gusts = l.InterpolateWide(gust).Values
		if len(gusts) != len(f.Values) {
			gusts = nil // never a grid the library would refuse
		}
		for i, g := range gusts {
			if !(g-f.Values[i] >= gustMargin) {
				gusts[i] = math.NaN() // no gust worth saying: the speed alone
			}
		}
	}
	unit := tuimaps.KilometresPerHour
	if mph {
		unit = tuimaps.MilesPerHour
		for i, v := range f.Values {
			f.Values[i] = units.MilesOf(v) // a missing value stays missing
		}
		for i, g := range gusts {
			gusts[i] = units.MilesOf(g)
		}
	}
	o := tuimaps.WindGrid(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Gusts: gusts}, dirs, unit, valid)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour // as temperature's: an hour past is that hour's, not stale
	return o, true
}

// unitGrid is linedGrid for a preset drawn in a unit: the library's grid for
// it made by preset, with the unit the values were converted to. The
// temperature and feels-like pass InterpolateWide (D-201) and TemperatureGrid,
// one look in both modes (D-102, go-tuiMaps L-15.4); the waves InterpolateOut,
// to the coast, the library drawing them over the sea alone (U2-31), and
// WaveGrid.
func unitGrid[U any](id string, f temperature.Field, convert func(float64) float64, unit U, valid, anchor time.Time, preset func(string, tuimaps.Grid, U, time.Time) tuimaps.Overlay) (tuimaps.Overlay, bool) {
	return linedGrid(id, f, convert, valid, anchor, func(id string, g tuimaps.Grid, valid time.Time) tuimaps.Overlay {
		return preset(id, g, unit, valid)
	})
}

// convertIf is convert where on, else nil: no conversion, the values as they are.
func convertIf(on bool, convert func(float64) float64) func(float64) float64 {
	if on {
		return convert
	}
	return nil
}

// linedGrid is an interpolated field as a preset's grid, lined - labelled
// contours over faint bands (D-102) - each value converted first where
// convert is given (a missing value stays missing); false when no point has
// any. Its currency counts from the hour's start: an hour past is that
// hour's, not stale (U2-13).
func linedGrid(id string, f temperature.Field, convert func(float64) float64, valid, anchor time.Time, preset func(string, tuimaps.Grid, time.Time) tuimaps.Overlay) (tuimaps.Overlay, bool) {
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	if convert != nil {
		for i, v := range f.Values {
			f.Values[i] = convert(v)
		}
	}
	o := preset(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Lines: true}, valid)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour
	return o, true
}

// ndfdAskBytes is an NDFD answer for a hundred points: 110 KB (2026-09-26),
// the days'; the hour's is smaller, and counted as the days' - an estimate
// over, never under.
const ndfdAskBytes = 110_000

// tempLayerCost is what the temperature would fetch in a refresh as if
// nothing were held, from NDFD, the default source (D-190): each box's
// denser lattice a hundred points an ask, the days and the hour each (D-201).
func tempLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	asks := 0
	for _, b := range fieldBoxes(in.region, in.view) { // a region's boxes (P10-02)
		l := temperature.NDFDLatticeFor(b.Name, b.Box)
		asks += 2 * ((l.Cols*l.Rows + 99) / 100)
	}
	return int64(asks) * ndfdAskBytes, asks
}

// tempHosts are the temperature's entries for the Status window's MAP block.
func tempHosts() []tty.MapSource {
	return hostsFor(temperature.Hosts(), map[string]string{"NWS NDFD": "temperature, waves", "Open-Meteo": "temperature, UV, rain",
		"Open-Meteo Marine": "waves"},
		map[string][]string{"Open-Meteo": {tempFrameNote}}) // what never changes about its data (D-132); its credit is About's (D-148)
}
