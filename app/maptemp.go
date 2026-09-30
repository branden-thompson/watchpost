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
	"sort"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
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
	registerMapLayer(mapLayer{key: tty.UVLayer, label: "UV", on: false, cost: func(mapInputs) (int64, int) { return 0, 0 }}) // its chip is the answer's, while it draws (D-183)
	registerMapLayer(mapLayer{key: tty.AirLayer, label: "Air quality", on: false, cost: airLayerCost, chips: []string{"O-METEO", "AIRNOW"}})
}

// tempSources is the temperature the app holds: both sources over one
// hardened client.
type tempSources struct {
	ndfd, om temperature.Source
	gate     *temperature.QuotaGate // Open-Meteo's asks, held while a quota is spent (W18.1, D-165)
	uvCold   *uvCold                // UV's cold start: EPA's index for the cities in view (W18.4, D-167)
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
	return &tempSources{ndfd: ndfd, om: om, gate: gate, rain: om, waves: ndfd, uvCold: &uvCold{epa: uv.NewEPA(c, "")}}
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
	if ask.Forecast && ask.TempNDFD && ts.ndfd.Covers(ask.Region) {
		return ts.ndfd
	}
	return ts.om
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
	if src != lp.temp.om {
		fill = lp.temp.om
	}
	var rescue *fallback
	if src == lp.temp.om && lp.temp.ndfd != nil && lp.temp.ndfd.Covers(ask.Region) {
		lp.mu.Lock()
		store := lp.history
		lp.mu.Unlock()
		rescue = &fallback{src: lp.temp.ndfd, past: recordedHour(store)} // Open-Meteo refused: NDFD draws what it can, the history the hours before (W18.2, W18.3b)
	}
	now := time.Now()
	t := buildTemperature(ctx, src, fill, ask, now, rescue)
	if ask.Forecast && lp.temp.rain != nil { // Forecast mode's rain and snow (W12.3): held, whether or not its row is on (D-99)
		t = withRainDays(ctx, t, lp.temp.rain, ask, now)
	}
	if lp.temp.waves != nil && lp.temp.rain != nil && ask.Region != geo.RegionSamoa { // the waves (D-125): held as the rest is (D-99); NDFD has no Samoa
		t = withWaves(ctx, t, lp.temp.waves, lp.temp.rain, ask, now)
	}
	if lp.temp.rain != nil { // Open-Meteo: the UV and the model's US AQI (D-137, D-139)
		t = withUV(ctx, t, lp.temp.rain, src.Name() == "Open-Meteo", ask, now, lp.historyStore(), lp.temp.uvCold) // valid UV kept, and replayed when refused; EPA's on a cold start (D-167)
		t = withAir(ctx, t, lp.temp.rain, ask, now)
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
	out := tty.MapTemperature{Source: src.Name()}
	boxes := fieldBoxes(ask.Region, ask.View)
	if len(boxes) == 0 {
		out.Notes = []string{"No temperature is drawn for " + ask.Region + "."}
		return out
	}
	anchor := askAnchor(ask, now)
	unit := tuimaps.Celsius
	if ask.Fahrenheit {
		unit = tuimaps.Fahrenheit
	}
	missing := map[string]bool{}
	credit := src.Name() == "Open-Meteo"
	fellBack, rescued, replayed := 0, 0, 0
	for _, b := range boxes {
		lat := temperature.LatticeFor(b.Name, b.Box)
		s, err := src.Fetch(ctx, lat, now)
		if err != nil && fill != nil {
			// A BOX THE SOURCE REFUSES IS OPEN-METEO'S (D-101): NDFD refuses
			// Hawaii's whole lattice, which straddles its grid's edge.
			if s, err = fill.Fetch(ctx, lat, now); err == nil {
				fellBack++
				credit = true
				missing[src.Name()+" did not answer for part of the map; Open-Meteo is drawn there."] = true
			}
		}
		byRescue := false
		if err != nil && rescue != nil && rescue.src != nil {
			if s, err = rescue.src.Fetch(ctx, lat, now); err == nil {
				rescued, byRescue = rescued+1, true
			}
		}
		if err != nil {
			if src.Name() == "NDFD" { // a Setting draws another (D-124)
				missing["Temperature is unavailable: NDFD did not answer. Settings → Maps → Temperature: Open-Meteo draws it instead."] = true
			} else {
				out.Problems = append(out.Problems, "Temperature: "+src.Name()+" did not answer for "+b.Name+" - "+err.Error())
			}
			continue
		}
		if !ask.Forecast {
			horizon := radarHorizon(ask, anchor)
			temp := func(values [][]float64) func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
				return func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
					return tempGrid(id, s.Lattice, values[i], unit, valid, anchor)
				}
			}
			past := 0
			if byRescue {
				s, past = withRecorded(s, rescue.past, b.Name, anchor) // the hours before, from the history (D-166)
				replayed += past
			}
			hours, feels, wind := hourGrids(tty.TemperatureLayer, b.Name, s.Hours, len(s.Hourly), anchor, horizon, temp(s.Hourly)),
				hourGrids(tty.FeelsLayer, b.Name, s.Hours, len(s.Feels), anchor, horizon, temp(s.Feels)), windHourGrids(s, b.Name, anchor, horizon, ask.Fahrenheit)
			if byRescue && past == 0 { // nothing recorded: the cold start
				stretchNow(hours, anchor)
				stretchNow(feels, anchor)
				stretchNow(wind, anchor)
			}
			out.Overlays, out.Feels, out.Wind = append(out.Overlays, hours...), append(out.Feels, feels...), append(out.Wind, wind...)
			continue
		}
		if fill != nil && fillDays(ctx, &s, fill, lat, now, &out) {
			credit = true
		}
		if fill != nil && fillFeelsNow(ctx, &s, fill, lat, anchor, now, &out) {
			credit = true
		}
		forecastGrids(&out, s, b.Name, anchor, unit, missing)
		feelsForecastGrids(&out, s, b.Name, anchor, unit)
		windForecastGrids(&out, s, b.Name, anchor, ask.Fahrenheit)
	}
	if fellBack == len(boxes) {
		out.Source = fill.Name() // every box is Open-Meteo's: the chip names it
	}
	switch { // NDFD drew where Open-Meteo did not (W18.2): the chips say so
	case rescued > 0 && rescued == len(boxes):
		out.Source, credit = rescue.src.Name(), false // every box NDFD's: its chip alone
	case rescued > 0:
		out.Source, credit = rescue.src.Name(), true // NDFD's boxes and Open-Meteo's: both chips
	}
	out.Chips = tempChips(out.Source, credit)                                                                                                                                                                                                  // the credit is the badge's, in full the Status window's (D-131)
	for key, drew := range map[string]int{tty.TemperatureLayer: len(out.Overlays) + len(out.High) + len(out.Low), tty.FeelsLayer: len(out.Feels) + len(out.FeelsHigh) + len(out.FeelsLow), tty.WindLayer: len(out.Wind) + len(out.WindDays)} { // three (P10-02)
		if drew == 0 {
			delete(out.Chips, key) // a layer drawing nothing names no source (D-183)
		} else if replayed > 0 {
			out.Chips[key] = append(append([]string(nil), out.Chips[key]...), recordedChip) // some of it from the history (D-173)
		}
	}
	var said []string
	for n := range missing {
		said = append(said, n)
	}
	sort.Strings(said)
	out.Notes = append(out.Notes, said...)
	return out
}

// fillDays puts Open-Meteo's values in every day the series has nothing for
// (D-100), asking it only when one is empty, and marks each day filled;
// true when anything was.
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
			*side.vals = side.from(got)
			if side.name == "wind" {
				s.PeakFrom[k], s.PeakGust[k] = got.PeakFrom[k], got.PeakGust[k] // the direction and the gust with the speed they came with (D-136)
			}
			if out.Filled == nil {
				out.Filled = map[string]bool{}
			}
			out.Filled[strconv.Itoa(k)+"/"+side.name], filled = true, true
		}
	}
	return filled
}

// fillFeelsNow puts Open-Meteo's feels-like in the current hour where the
// source has none (D-119): NDFD answers it from the next hour. True when it
// did.
func fillFeelsNow(ctx context.Context, s *temperature.Series, fill temperature.Source, lat temperature.Lattice, anchor, now time.Time, out *tty.MapTemperature) bool {
	if vals, ok := s.FeelsAt(anchor); ok && !allMissing(vals) {
		return false
	}
	f, err := fill.Fetch(ctx, lat, now)
	if err != nil {
		return false
	}
	vals, ok := f.FeelsAt(anchor)
	if !ok || allMissing(vals) {
		return false
	}
	s.Feels[s.HourIndex(anchor)] = vals
	if out.Filled == nil {
		out.Filled = map[string]bool{}
	}
	out.Filled["now/feels"] = true
	return true
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
// without (Hawaii's wind stopped at MRMS's box, inside the map), split at the
// antimeridian, which a grid cannot cross (Alaska).
func fieldBoxes(region string, view geo.Box) []fieldBox {
	if region == geo.RegionContiguous {
		// THE RADAR'S BOXES, THE OUTER ONES GROWN TO THE REGION'S EDGES (UAT-2
		// U2-30): the radar stops at 126W and 65W and the region reaches 130W
		// and 64W - the waves stopped at a line in the Pacific.
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

// stretchNow draws a current hour's grids under the hour before it too:
// the loop's earlier frames, where a source with no past hour has nothing
// else (D-166's cold start, until the local history holds that hour).
func stretchNow(grids []tuimaps.Overlay, anchor time.Time) {
	for i := range grids { // bounded by the grids (P10-02)
		if grids[i].During.From.Equal(anchor) {
			grids[i].During.From = anchor.Add(-time.Hour)
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
// the fields came and went as the loop played into the forecast (UAT-2
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
		if o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/now", s.Lattice, vals, unit, anchor, anchor); ok {
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
			o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/d"+strconv.Itoa(k)+"/"+side.name, s.Lattice, side.vals, unit, anchor, anchor)
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
		if o, ok := tempGrid(tty.FeelsLayer+"/"+box+"/now", s.Lattice, vals, unit, anchor, anchor); ok {
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
			if o, ok := tempGrid(tty.FeelsLayer+"/"+box+"/d"+strconv.Itoa(k)+"/"+side.name, s.Lattice, side.vals, unit, anchor, anchor); ok {
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
	f, dirs := l.InterpolateWind(speed, from)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	var gusts []float64
	if len(gust) == len(speed) && !allMissing(gust) {
		gusts = l.Interpolate(gust).Values
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
			f.Values[i] = v / 1.609344 // a missing value stays missing
		}
		for i, g := range gusts {
			gusts[i] = g / 1.609344
		}
	}
	o := tuimaps.WindGrid(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Gusts: gusts}, dirs, unit, valid)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour // as temperature's: an hour past is that hour's, not stale
	return o, true
}

// tempGrid is one lattice's values as a grid the library draws, in the
// listener's unit; false when every value is missing. Its currency counts
// from the hour's start (U2-13).
func tempGrid(id string, l temperature.Lattice, values []float64, unit tuimaps.Unit, valid, anchor time.Time) (tuimaps.Overlay, bool) {
	var toF func(float64) float64
	if unit == tuimaps.Fahrenheit {
		toF = func(c float64) float64 { return c*9/5 + 32 }
	}
	return linedGrid(id, l.Interpolate(values), toF, valid, anchor, func(id string, g tuimaps.Grid, valid time.Time) tuimaps.Overlay {
		return tuimaps.TemperatureGrid(id, g, unit, valid) // one look in both modes (D-102, go-tuiMaps L-15.4)
	})
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

// tempRequestBytes is one lattice's answer on the wire, measured: 139 KB for
// 80 points from Open-Meteo, asked for feels-like and the loop's thirteen
// hours ahead (2026-09-28; 106 KB at two hours); 110 KB for 100 from NDFD
// (2026-09-26).
const tempRequestBytes = 140_000

// tempLayerCost is what the temperature would fetch in a refresh as if
// nothing were held: a request a box.
func tempLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := len(fieldBoxes(in.region, in.view))
	return int64(boxes) * tempRequestBytes, boxes
}

// tempHosts are the temperature's entries for the Status window's MAP block.
func tempHosts() []tty.MapSource {
	return hostsFor(temperature.Hosts(), map[string]string{"NWS NDFD": "temperature, waves", "Open-Meteo": "temperature, UV, rain",
		"Open-Meteo Marine": "waves", "Open-Meteo Air Quality": "air"},
		map[string][]string{"Open-Meteo": {tempFrameNote}}) // what never changes about its data (D-132); its credit is About's (D-148)
}
