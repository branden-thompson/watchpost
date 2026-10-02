package app

// mapairuv.go — the UV index and air quality on the map (0.18.0 D-137 to
// D-140): each its own row, off by default, one tint with temperature and
// feels-like, asked only while on. UV rides Open-Meteo's temperature request
// - free where Open-Meteo is temperature's source; the air is the model's US
// AQI as the tint and AirNow's measured AQI as markers over it.

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/agememo"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// withUV adds the UV index for the mode: EPA's forecast for the largest
// cities in view first, as markers (D-186) - and Open-Meteo's grid beside them
// only over the boxes whose Open-Meteo forecast was asked anyway, which the
// client's cache answers again (D-191): no call is made for UV alone.
//
// VALID UV IS RECORDED AND REPLAYED (W18.4, D-167): each hour Open-Meteo
// answered, up to the current one, goes into the history; where it was asked
// and did not answer, Radar mode draws the hours recorded, and the chip says
// RECORDED.
func withUV(ctx context.Context, t tty.MapTemperature, om *temperature.OpenMeteo, asked func(box string) bool, ask tty.MapAsk, now time.Time, store *history.Store, cities *uvCities) tty.MapTemperature {
	live, replayed := 0, 0
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		if asked == nil || !asked(b.Name) {
			continue // not asked for anything else: UV alone asks nothing of Open-Meteo (D-191)
		}
		lat := temperature.LatticeFor(b.Name, b.Box)
		s, err := om.Fetch(ctx, lat, now)
		if err != nil {
			t.Problems = append(t.Problems, "UV: Open-Meteo did not answer for "+b.Name) // D-124
			if !ask.Forecast {
				past := replayUV(store, b.Name, lat, askAnchor(ask, now))
				t.UV, replayed = append(t.UV, past...), replayed+len(past)
			}
			continue
		}
		live++
		recordUV(store, b.Name, s, askAnchor(ask, now))
		m := temperature.Measure{Lattice: lat, Hours: s.Hours, Hourly: s.UV, Max: s.UVMax}
		hours, days := measureGrids(m, tty.UVLayer, b.Name, ask, now, func(id string, vals []float64, valid, anchor time.Time) (tuimaps.Overlay, bool) {
			return fieldGrid(id, lat, vals, valid, anchor, tuimaps.UVGrid)
		})
		t.UV, t.UVDays = append(t.UV, hours...), append(t.UVDays, days...)
	}
	var chips []string // what drew, the history last (D-173, D-183)
	if ask.UV && cities != nil {
		if marks := cities.markers(ctx, ask.View, tty.UVCitiesByCount(ask.UVCities), askAnchor(ask, now), ask.Forecast); len(marks) > 0 {
			t.UV, chips = append(t.UV, marks...), append(chips, "EPA")
			t = withNote(t, tty.UVLayer, "UV: EPA's forecast for cities across the view.")
		}
	}
	if live > 0 {
		chips = append(chips, "O-METEO")
	}
	if replayed > 0 {
		chips = append(chips, recordedChip)
	}
	if len(chips) > 0 {
		t.Chips = withChips(t.Chips, tty.UVLayer, chips...)
	}
	return t
}

// withAir adds air quality's tint, asked only while its row is on: AirNow's
// current-AQI contours (D-193), keyless - each field box a grid in the AQI
// scale, each cell its contour's category, none where no contour reaches;
// through Radar mode's loop, and on Forecast mode's Now alone. AirNow's
// monitors are drawn over it (D-139); Open-Meteo's air-quality API is not
// asked (D-185).
func withAir(ctx context.Context, t tty.MapTemperature, airnow *airquality.Provider, grids *airGrids, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	if !ask.Air || airnow == nil {
		return t
	}
	contours, err := airnow.Contours(ctx, now)
	if err != nil {
		t.Problems = append(t.Problems, "Air quality: AirNow's contours did not answer - "+err.Error()) // D-124
		return t
	}
	nowStep, _ := forecastDays(askAnchor(ask, now))
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		g, ok := grids.grid(ctx, b.Name, b.Box, contours)
		if !ok {
			continue
		}
		o := tuimaps.AirQualityGrid(tty.AirLayer+"/"+b.Name+"/airnow", g, contours.Hour)
		o.Keeps, o.Credit = 3*time.Hour, airquality.Attribution // AirNow redraws them about hourly
		if ask.Forecast {
			o.During = nowStep.Span // the current hour's: Now's alone
		}
		t.Air = append(t.Air, o)
	}
	// WHY THE FORECAST DAYS ARE NOT FILLED IS SAID (U2-52, D-203): AirNow's
	// contours are the hour's; its forecasts are by area, two days of them.
	if len(t.Air) > 0 && ask.Forecast {
		t = withNote(t, tty.AirLayer, "Air quality: the fill is AirNow's current hour; Today and Tomorrow are its forecasts by area, as markers; none is published past Tomorrow.")
	} else if len(t.Air) > 0 {
		t = withNote(t, tty.AirLayer, "Air quality: the fill is AirNow's current hour, drawn through the loop.")
	}
	return t
}

// withNote adds a note of a layer's own, said while that layer is on (D-203).
func withNote(t tty.MapTemperature, layer string, notes ...string) tty.MapTemperature {
	if len(notes) == 0 {
		return t
	}
	by := make(map[string][]string, len(t.LayerNotes)+1) // a copy: t is passed by value and its map is shared
	for k, v := range t.LayerNotes {
		by[k] = v
	}
	by[layer] = append(append([]string(nil), by[layer]...), notes...)
	t.LayerNotes = by
	return t
}

// airGrids keeps each box's contour grid by the box and the file's hour (W14
// P-19): the same hour's contours over the same box are the same cells, so an
// ask again, or a pan within the boxes, rasterises nothing. A nil airGrids
// rasterises every ask.
type airGrids struct {
	m     lazyMemo[airGridKey, airGrid]
	mu    sync.Mutex
	built int // grids rasterised, for the tests
}

// airGridKey is a box and the contours' hour.
type airGridKey struct {
	box  string
	hour time.Time
}

// airGrid is a box's grid, and whether any contour holds it.
type airGrid struct {
	g  tuimaps.Grid
	ok bool
}

// airGridRules keep a box's grid for its file's hour and the next, for the
// boxes a view holds.
var airGridRules = agememo.Options{Fresh: 2 * time.Hour, Max: 32}

// grid is box's grid from c, kept by the box and c's hour.
func (a *airGrids) grid(ctx context.Context, name string, box geo.Box, c airquality.Contours) (tuimaps.Grid, bool) {
	if a == nil {
		return contourGrid(box, c)
	}
	got, _ := a.m.memo(airGridRules).Do(ctx, airGridKey{box: name, hour: c.Hour}, func() (airGrid, error) {
		a.mu.Lock()
		a.built++
		a.mu.Unlock()
		g, ok := contourGrid(box, c)
		return airGrid{g: g, ok: ok}, nil
	})
	return got.g, got.ok
}

// contourCells is the most cells a side of a box's contour grid has: fine
// enough for a state, the work an hour's file once a box.
const contourCells = 160

// contourGrid is a box's cells, each its contour's category as an AQI in it
// (airquality.CategoryAQI), missing where no contour holds the cell's
// middle; false where none holds any.
func contourGrid(box geo.Box, c airquality.Contours) (tuimaps.Grid, bool) {
	w, h := box.E-box.W, box.N-box.S
	if w <= 0 || h <= 0 {
		return tuimaps.Grid{}, false
	}
	cols := contourCells
	rows := min(max(int(float64(cols)*h/w), 2), contourCells)
	g := tuimaps.Grid{West: box.W, South: box.S, East: box.E, North: box.N, Cols: cols, Rows: rows, Values: make([]float64, cols*rows)}
	any := false
	for i, cat := range c.Raster(box.W, box.S, box.E, box.N, cols, rows) { // a scanline, not a point test a cell (P10-02)
		g.Values[i] = math.NaN()
		if cat >= 0 {
			g.Values[i], any = airquality.CategoryAQI(cat), true
		}
	}
	return g, any
}

// measureGrids are one measure's grids for the mode, as temperature's: Radar
// mode's every hour to the loop's horizon, each during its hour; Forecast
// mode's Now and each day's value, each during its step.
func measureGrids(m temperature.Measure, layer, box string, ask tty.MapAsk, now time.Time,
	grid func(id string, vals []float64, valid, anchor time.Time) (tuimaps.Overlay, bool)) (hours, days []tuimaps.Overlay) {
	anchor := askAnchor(ask, now)
	if !ask.Forecast {
		return hourGrids(layer, box, m.Hours, len(m.Hourly), anchor, radarHorizon(ask, anchor), func(i int, id string, valid time.Time) (tuimaps.Overlay, bool) {
			return grid(id, m.Hourly[i], valid, anchor)
		}), nil
	}
	nowStep, steps := forecastDays(anchor)
	if vals, ok := m.At(anchor); ok {
		if o, ok := grid(layer+"/"+box+"/now", vals, anchor, anchor); ok {
			o.During = nowStep.Span
			hours = append(hours, o)
		}
	}
	for k, step := range steps {
		if m.Max[k] == nil {
			break
		}
		if o, ok := grid(layer+"/"+box+"/d"+strconv.Itoa(k), m.Max[k], anchor, anchor); ok {
			o.During = step.Span
			days = append(days, o)
		}
	}
	return hours, days
}

// fieldGrid is one lattice's values as a preset's grid, lined - labelled
// contours over faint bands, as temperature's (D-102); false when no point
// has any.
func fieldGrid(id string, l temperature.Lattice, vals []float64, valid, anchor time.Time, preset func(string, tuimaps.Grid, time.Time) tuimaps.Overlay) (tuimaps.Overlay, bool) {
	return linedGrid(id, l.Interpolate(vals), nil, valid, anchor, preset)
}

// airnowIn is AirNow's reporting areas, asked only while Air quality is on.
func (lp *livePipelines) airnowIn(ctx context.Context, ask tty.MapAsk) []airquality.Area {
	if !ask.Air || lp.airnow == nil {
		return nil
	}
	areas, err := lp.airnow.Areas(ctx, time.Now())
	if err != nil {
		return nil // the map's own problem to say: the tint still draws (D-124)
	}
	return areas
}

// airLabelMost is the most monitors in view each labelled with its name and
// AQI (D-139); past it, markers.
const airLabelMost = 30

// airnowOverlays are the reporting areas in view as markers in their AQI's
// category (D-139): the measured one, drawn through the loop and on Now; and
// in Forecast mode AirNow's own forecast for today and tomorrow, each during
// its step. Labelled "Name 42", or "Name Good" where AirNow forecasts a
// category alone, while few enough are in view.
func airnowOverlays(areas []airquality.Area, view tty.MapView, anchor, at time.Time) ([]tuimaps.Overlay, map[string]tty.TimedOverlay) {
	var in []airquality.Area
	for _, a := range areas {
		if view.Contains(a.Lat, a.Lon) {
			in = append(in, a)
		}
	}
	if len(in) == 0 {
		return nil, nil
	}
	labelled := len(in) <= airLabelMost
	mark := func(a airquality.Area, r airquality.Reading) tuimaps.Feature {
		f := tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: a.Lon, Lat: a.Lat}}}, Role: tuimaps.AirQualityRole(r.Value()), ID: a.Name + ", " + a.State}
		if labelled {
			f.Label = a.Name + " " + readingWord(r)
		}
		return f
	}
	var out []tuimaps.Overlay
	times := map[string]tty.TimedOverlay{}
	var now []tuimaps.Feature
	for _, a := range in {
		if a.Now != nil {
			now = append(now, mark(a, *a.Now))
		}
	}
	if len(now) > 0 {
		o := tuimaps.Overlay{ID: tty.AirLayer + "/airnow", Valid: overlayStamp(at), Keeps: time.Hour, Credit: airquality.Attribution, Features: now}
		out, times[o.ID] = append(out, o), tty.TimedOverlay{Happened: true} // so now: through the loop, and on Now alone in Forecast mode
	}
	if anchor.IsZero() {
		return out, times
	}
	steps := tty.ForecastSteps(anchor)
	for day := 0; day <= 1 && day+1 < len(steps); day++ {
		var feats []tuimaps.Feature
		for _, a := range in {
			if r, ok := a.Forecast[day]; ok {
				feats = append(feats, mark(a, r))
			}
		}
		if len(feats) == 0 {
			continue
		}
		span := steps[day+1].Span
		o := tuimaps.Overlay{ID: tty.AirLayer + "/airnow/d" + strconv.Itoa(day), Valid: anchor, Keeps: 48 * time.Hour, Credit: airquality.Attribution, Features: feats}
		out, times[o.ID] = append(out, o), tty.TimedOverlay{From: span.From, Until: span.Until} // AirNow's forecast, during its day's step
	}
	return out, times
}

// readingWord is a reading as a marker says it: its AQI, or its category
// where AirNow forecasts that alone.
func readingWord(r airquality.Reading) string {
	if r.AQI == r.AQI { // not NaN
		return strconv.Itoa(int(r.AQI))
	}
	return r.Category
}

// contoursBytes is AirNow's contours file: about 2 MB (2026-10-01).
const contoursBytes = 2_000_000

// airnowBytes is AirNow's national file: 1.9 MB (2026-09-29).
const airnowBytes = 1_950_000

// airLayerCost is AirNow's two national files, the reporting areas and the
// contours (D-193), whatever the view.
func airLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	return airnowBytes + contoursBytes, 2
}

// airHosts are air quality's entries for the Status window's MAP block, with
// their credits (D-131).
func airHosts() []tty.MapSource {
	return []tty.MapSource{{Name: "EPA AirNow", Host: airquality.Host(), Layers: "air"}, // its credit is About's (D-148)
		{Name: "EPA Envirofacts", Host: uv.Host, Layers: "UV, cold start"}} // D-167
}

// uvCities is UV's first source (D-167, D-186): EPA's UV index, hour by hour
// for today, for the largest cities in view - keyless, and not a lattice:
// drawn as markers.
type uvCities struct {
	epa    *uv.EPA
	cities func(view geo.Box, n int) []geodata.City // spread over the view, at most n (D-202)
}

// uvCitySpacingKm is how near two cities may be: no closer than about 100 km
// (D-202), so New York's boroughs are one marker, not three.
const uvCitySpacingKm = 100

// uvAskers is how many cities are asked at once: never one by one, which
// holds the map's UV for seconds at 24 cities, and never all at once.
const uvAskers = 4

// markers are the cities' UV as points in their bands' colours, each
// labelled with its city and value. Radar mode: the current hour and the
// pastHours before it, each during its own hour, as the history's replay is.
// Forecast mode: Now's hour during Now, and the day's peak during Today -
// EPA forecasts today alone. None where no city answered.
func (c *uvCities) markers(ctx context.Context, view geo.Box, n int, anchor time.Time, forecast bool) []tuimaps.Overlay {
	if c == nil || c.epa == nil || c.cities == nil {
		return nil
	}
	got := c.read(ctx, c.cities(view, n))
	var out []tuimaps.Overlay
	add := func(id string, valid time.Time, during tuimaps.Span, pick func([]uv.Reading) (float64, bool)) {
		if feats := uvPoints(got, pick); len(feats) > 0 {
			out = append(out, tuimaps.Overlay{ID: tty.UVLayer + "/epa/" + id, Valid: valid, Keeps: time.Hour, Credit: uv.Attribution, Features: feats, During: during})
		}
	}
	if forecast {
		nowStep, days := forecastDays(anchor)
		add("now", anchor, nowStep.Span, func(r []uv.Reading) (float64, bool) { return uv.At(r, anchor) })
		if len(days) > 0 {
			add("today", anchor, days[0].Span, uvPeak)
		}
		return out
	}
	for back := pastHours; back >= 0; back-- { // four (P10-02)
		h := anchor.Add(-time.Duration(back) * time.Hour)
		add(h.UTC().Format("2006-01-02T15"), h, tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}, func(r []uv.Reading) (float64, bool) { return uv.At(r, h) })
	}
	return out
}

// read asks EPA for each city's hours, uvAskers at a time, and keeps the
// cities' order whatever order they answer in. A city with no zone, or that
// EPA does not answer for, is left out - counted as nothing, never said (D-124).
func (c *uvCities) read(ctx context.Context, cities []geodata.City) []cityReadings {
	answers := make([]*cityReadings, len(cities))
	slots := make(chan struct{}, uvAskers)
	var wg sync.WaitGroup
	for i, city := range cities { // at most the largest count, 48 (P10-02)
		loc, err := tz.Location(city.TZ)
		if err != nil {
			continue
		}
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; wg.Done() }()
			if readings, err := c.epa.Hourly(ctx, city.Name, city.State, loc); err == nil {
				answers[i] = &cityReadings{city, readings}
			}
		}()
	}
	wg.Wait()
	var got []cityReadings
	for _, a := range answers { // as many as asked (P10-02)
		if a != nil {
			got = append(got, *a)
		}
	}
	return got
}

// cityReadings is a city and EPA's hours for it.
type cityReadings struct {
	city     geodata.City
	readings []uv.Reading
}

// uvPoints are the cities as points, each in the band of the value picked
// from its hours, labelled "City 7"; a city without one is left out.
func uvPoints(got []cityReadings, pick func([]uv.Reading) (float64, bool)) []tuimaps.Feature {
	var feats []tuimaps.Feature
	for _, a := range got { // at most the largest count, 48 (P10-02)
		if v, ok := pick(a.readings); ok {
			feats = append(feats, tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: a.city.Lon, Lat: a.city.Lat}}},
				Role: tuimaps.UVRole(v), Label: a.city.Name + " " + strconv.FormatFloat(v, 'f', -1, 64)})
		}
	}
	return feats
}

// uvPeak is the day's highest reading: Today's UV, as Open-Meteo's daily
// highest is the day's (D-137).
func uvPeak(readings []uv.Reading) (float64, bool) {
	peak, ok := 0.0, false
	for _, r := range readings { // a day's hours (P10-02)
		if !ok || r.Index > peak {
			peak, ok = r.Index, true
		}
	}
	return peak, ok
}

// uvRanked is how many of the largest US cities UV's markers look among:
// every one past some 50,000 people, so a state's view holds a few.
const uvRanked = 1000

// spreadInView is a view's cities for UV's markers, at most n, spread over
// it (D-202): the view cut into about n cells by its shape on the ground,
// each cell's largest ranked city first, then each cell's next, round by
// round, to make up the count - cells over the sea hold none, so the land's
// take a second - none nearer another than uvCitySpacingKm, or than half a cell's
// side where cells are smaller (a state's view keeps its count). Only cities
// EPA can be read for: in the view, with a zone. The ranking's order kept.
func spreadInView(ranked []geodata.City, view geo.Box, n int) []geodata.City {
	if n <= 0 || len(ranked) == 0 {
		return nil
	}
	mid := (view.S + view.N) / 2
	wide := geo.HaversineKM(mid, view.W, mid, view.E)
	tall := geo.HaversineKM(view.S, view.W, view.N, view.W)
	if wide <= 0 || tall <= 0 {
		return nil
	}
	cols := max(1, int(math.Round(math.Sqrt(float64(n)*wide/tall))))
	rows := max(1, int(math.Round(float64(n)/float64(cols))))
	spacing := min(uvCitySpacingKm, min(wide/float64(cols), tall/float64(rows))/2)
	taken := make([]bool, len(ranked))
	var chosen []geodata.City
	near := func(c geodata.City) bool {
		for _, o := range chosen { // at most n (P10-02)
			if geo.HaversineKM(c.Lat, c.Lon, o.Lat, o.Lon) < spacing {
				return true
			}
		}
		return false
	}
	usable := func(c geodata.City) bool { return c.TZ != "" && view.Contains(c.Lat, c.Lon) }
	held := make([]int, cols*rows)
	for round := 1; round <= n && len(chosen) < n; round++ { // a city a cell a round, at most n rounds (P10-02)
		before := len(chosen)
		for i, c := range ranked { // at most uvRanked (P10-02)
			if len(chosen) == n || taken[i] || !usable(c) || near(c) {
				continue
			}
			col := min(cols-1, int((c.Lon-view.W)/(view.E-view.W)*float64(cols)))
			row := min(rows-1, int((view.N-c.Lat)/(view.N-view.S)*float64(rows)))
			if cell := row*cols + col; held[cell] < round {
				held[cell]++
				taken[i] = true
				chosen = append(chosen, c)
			}
		}
		if len(chosen) == before {
			break // nothing more the spacing allows
		}
	}
	var out []geodata.City
	for i, c := range ranked { // the ranking's order (P10-02)
		if taken[i] {
			out = append(out, c)
		}
	}
	return out
}

// citiesFrom is the cold start's cities over the index: its largest ranked
// once, on the first ask - off the UI goroutine, where the fetch runs.
func citiesFrom(idx func() *geodata.Index) func(geo.Box, int) []geodata.City {
	var once sync.Once
	var ranked []geodata.City
	return func(view geo.Box, n int) []geodata.City {
		once.Do(func() {
			if i := idx(); i != nil {
				ranked = i.TopUS(uvRanked)
			}
		})
		return spreadInView(ranked, view, n)
	}
}
