package app

// mapairuv.go — the UV index and air quality on the map (0.18.0 D-137 to
// D-140): each its own row, off by default, one tint with temperature and
// feels-like, asked only while on. UV rides Open-Meteo's temperature request
// - free where Open-Meteo is temperature's source; the air is the model's US
// AQI as the tint and AirNow's measured AQI as markers over it.

import (
	"context"
	"strconv"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// withUV adds the UV index for the mode (D-137), from Open-Meteo's answer:
// the one temperature asked for where it is the source - the client's cache
// answers it again - or its own, asked only while UV is on.
func withUV(ctx context.Context, t tty.MapTemperature, om *temperature.OpenMeteo, free bool, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	if !free && !ask.UV {
		return t
	}
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		lat := temperature.LatticeFor(b.Name, b.Box)
		s, err := om.Fetch(ctx, lat, now)
		if err != nil {
			t.Problems = append(t.Problems, "UV: Open-Meteo did not answer for "+b.Name) // D-124
			continue
		}
		m := temperature.Measure{Lattice: lat, Hours: s.Hours, Hourly: s.UV, Max: s.UVMax}
		hours, days := measureGrids(m, tty.UVLayer, b.Name, ask, now, func(id string, vals []float64, valid, anchor time.Time) (tuimaps.Overlay, bool) {
			return fieldGrid(id, lat, vals, valid, anchor, tuimaps.UVGrid)
		})
		t.UV, t.UVDays = append(t.UV, hours...), append(t.UVDays, days...)
	}
	return t
}

// withAir adds the model's US AQI for the mode (D-139), asked only while Air
// quality is on: a request a field box, to the air-quality API.
func withAir(ctx context.Context, t tty.MapTemperature, om *temperature.OpenMeteo, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	if !ask.Air {
		return t
	}
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		lat := temperature.LatticeFor(b.Name, b.Box)
		m, err := om.AirQuality(ctx, lat, now)
		if err != nil {
			t.Problems = append(t.Problems, "Air quality: Open-Meteo did not answer for "+b.Name) // D-124
			continue
		}
		hours, days := measureGrids(m, tty.AirLayer, b.Name, ask, now, func(id string, vals []float64, valid, anchor time.Time) (tuimaps.Overlay, bool) {
			return fieldGrid(id, lat, vals, valid, anchor, tuimaps.AirQualityGrid)
		})
		t.Air, t.AirDays = append(t.Air, hours...), append(t.AirDays, days...)
	}
	return t
}

// measureGrids are one measure's grids for the mode, as temperature's: Radar
// mode's every hour to the loop's horizon, each during its hour; Forecast
// mode's Now and each day's value, each during its step.
func measureGrids(m temperature.Measure, layer, box string, ask tty.MapAsk, now time.Time,
	grid func(id string, vals []float64, valid, anchor time.Time) (tuimaps.Overlay, bool)) (hours, days []tuimaps.Overlay) {
	anchor := ask.Anchor
	if anchor.IsZero() {
		anchor = now.Truncate(time.Hour)
	}
	if !ask.Forecast {
		for i, h := range m.Hours {
			if h.After(radarHorizon(ask, anchor)) || i >= len(m.Hourly) {
				continue // past the loop's hours ahead (U2-32)
			}
			if o, ok := grid(layer+"/"+box+"/"+h.UTC().Format("2006-01-02T15"), m.Hourly[i], stampOf(h, anchor), anchor); ok {
				o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
				hours = append(hours, o)
			}
		}
		return hours, nil
	}
	steps := tty.ForecastSteps(anchor)
	if vals, ok := m.At(anchor); ok {
		if o, ok := grid(layer+"/"+box+"/now", vals, anchor, anchor); ok {
			o.During = steps[0].Span
			hours = append(hours, o)
		}
	}
	for k := range temperature.Days {
		if k+1 >= len(steps) || m.Max[k] == nil {
			break
		}
		if o, ok := grid(layer+"/"+box+"/d"+strconv.Itoa(k), m.Max[k], anchor, anchor); ok {
			o.During = steps[k+1].Span
			days = append(days, o)
		}
	}
	return hours, days
}

// fieldGrid is one lattice's values as a preset's grid, lined - labelled
// contours over faint bands, as temperature's (D-102); false when no point
// has any.
func fieldGrid(id string, l temperature.Lattice, vals []float64, valid, anchor time.Time, preset func(string, tuimaps.Grid, time.Time) tuimaps.Overlay) (tuimaps.Overlay, bool) {
	f := l.Interpolate(vals)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	o := preset(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Lines: true}, valid)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour // as temperature's: an hour past is that hour's, not stale
	return o, true
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
func airnowOverlays(areas []airquality.Area, view tty.MapView, anchor time.Time) ([]tuimaps.Overlay, map[string]tty.TimedOverlay) {
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
		o := tuimaps.Overlay{ID: tty.AirLayer + "/airnow", Valid: time.Now(), Keeps: time.Hour, Credit: airquality.Attribution, Features: now}
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

// airBytes are a field box's air-quality request on the wire: about 25 KB
// for six points, five days hourly (2026-09-29).
const airBytes = 330_000

// airnowBytes is AirNow's national file: 1.9 MB (2026-09-29).
const airnowBytes = 1_950_000

// airLayerCost is a request a field box to the air-quality API, and AirNow's
// one file.
func airLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := len(fieldBoxes(in.region, in.view))
	return int64(boxes)*airBytes + airnowBytes, boxes + 1
}

// airHosts are air quality's entries for the Status window's MAP block, with
// their credits (D-131).
func airHosts() []tty.MapSource {
	return []tty.MapSource{{Name: "EPA AirNow", Host: airquality.Host(), Use: "every reporting area's measured US AQI and its forecast, one national file, while Air quality is on"}} // its credit is About's (D-148)
}
