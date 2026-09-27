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
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The temperature, registered: OFF BY DEFAULT (D-99), and loaded in the
// background while the map is open so switching it on in the Overlays menu
// draws at once.
func init() {
	registerMapLayer(mapLayer{key: tty.TemperatureLayer, label: "Temperature", on: false, cost: tempLayerCost})
}

// tempSources is the temperature the app holds: both sources over one
// hardened client.
type tempSources struct {
	ndfd, om temperature.Source
}

// tempSourcesOver is both sources over one temperature client (overClient).
func tempSourcesOver(c *httpx.Client) *tempSources {
	return &tempSources{ndfd: temperature.NewNDFD(c, ""), om: temperature.NewOpenMeteo(c, "")}
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

// tempFrameNote is what Radar mode says of its temperature (D-96).
const tempFrameNote = "Each radar frame draws its own hour's temperature."

// mapTemperature is the window's temperature for the mode, every grid built
// with its span. Where the chosen source has nothing for a day, Open-Meteo
// fills it (D-100).
func (lp *livePipelines) mapTemperature(ctx context.Context, ask tty.MapAsk) tty.MapTemperature {
	if lp.temp == nil || ask.Region == "" {
		return tty.MapTemperature{}
	}
	src := lp.temp.sourceFor(ask)
	var fill temperature.Source
	if src != lp.temp.om {
		fill = lp.temp.om
	}
	return buildTemperature(ctx, src, fill, ask, time.Now())
}

// buildTemperature fetches each box's lattice and makes its grids. EVERY
// ANSWER WITHIN AN HOUR IS THE SAME (UAT-2 U2-13): what a grid says of itself
// is worked out from the hour's start, never the clock, so the window hands
// nothing in again and nothing blinks.
func buildTemperature(ctx context.Context, src, fill temperature.Source, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	out := tty.MapTemperature{Source: src.Name()}
	boxes := radar.BoxesFor(ask.Region, ask.View)
	if len(boxes) == 0 {
		out.Notes = []string{"No temperature is drawn for " + ask.Region + "."}
		return out
	}
	anchor := ask.Anchor
	if anchor.IsZero() {
		anchor = now.Truncate(time.Hour)
	}
	unit := tuimaps.Celsius
	if ask.Fahrenheit {
		unit = tuimaps.Fahrenheit
	}
	missing := map[string]bool{}
	credit := src.Name() == "Open-Meteo"
	fellBack := 0
	for _, b := range boxes {
		lat := temperature.LatticeFor(b.Name, geo.Box{W: b.W, S: b.S, E: b.E, N: b.N})
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
		if err != nil {
			missing["Temperature is unavailable: "+src.Name()+" did not answer."] = true
			continue
		}
		if !ask.Forecast {
			out.Overlays = append(out.Overlays, hourGrids(s, b.Name, anchor, unit)...)
			continue
		}
		if fill != nil && fillDays(ctx, &s, fill, lat, now, &out) {
			credit = true
		}
		forecastGrids(&out, s, b.Name, anchor, unit, missing)
	}
	if fellBack == len(boxes) {
		out.Source = fill.Name() // every box is Open-Meteo's: the chip names it
	}
	if credit {
		out.Notes = append(out.Notes, temperature.OpenMeteoCredit+".")
	}
	if !ask.Forecast {
		out.Notes = append(out.Notes, tempFrameNote)
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
			{"low", &s.Low[k], func(f *temperature.Series) []float64 { return f.Low[k] }}} {
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
			if out.Filled == nil {
				out.Filled = map[string]bool{}
			}
			out.Filled[strconv.Itoa(k)+"/"+side.name], filled = true, true
		}
	}
	return filled
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

// hourGrids are Radar mode's grids: every hour up to the current one, each
// drawn during its hour (D-96).
func hourGrids(s temperature.Series, box string, anchor time.Time, unit tuimaps.Unit) []tuimaps.Overlay {
	var out []tuimaps.Overlay
	for i, h := range s.Hours {
		if h.After(anchor) {
			continue // a forecast hour: no radar frame is in it
		}
		o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/"+h.UTC().Format("2006-01-02T15"), s.Lattice, s.Hourly[i], unit, h, anchor)
		if !ok {
			continue
		}
		o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
		out = append(out, o)
	}
	return out
}

// forecastGrids are Forecast mode's grids: Now, and each day's high and low,
// each drawn during its step (D-94, D-97). A day no source has anything for
// is said, not drawn empty.
func forecastGrids(out *tty.MapTemperature, s temperature.Series, box string, anchor time.Time, unit tuimaps.Unit, missing map[string]bool) {
	steps := tty.ForecastSteps(anchor)
	if vals, _, ok := s.HourAt(anchor); ok {
		if o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/now", s.Lattice, vals, unit, anchor, anchor); ok {
			o.During = steps[0].Span
			out.Overlays = append(out.Overlays, o)
		}
	}
	for k := range temperature.Days {
		if k+1 >= len(steps) {
			break
		}
		for _, side := range []struct {
			name string
			vals []float64
			into *[]tuimaps.Overlay
		}{{"high", s.High[k], &out.High}, {"low", s.Low[k], &out.Low}} {
			o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/d"+strconv.Itoa(k)+"/"+side.name, s.Lattice, side.vals, unit, anchor, anchor)
			if !ok {
				missing["No "+side.name+" for "+steps[k+1].Label+" from this source: it has passed, or is past the source's reach."] = true
				continue
			}
			o.During = steps[k+1].Span
			*side.into = append(*side.into, o)
		}
	}
}

// tempGrid is one lattice's values as a grid the library draws, in the
// listener's unit; false when every value is missing. Its currency counts
// from the hour's start (U2-13).
func tempGrid(id string, l temperature.Lattice, values []float64, unit tuimaps.Unit, valid, anchor time.Time) (tuimaps.Overlay, bool) {
	f := l.Interpolate(values)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	if unit == tuimaps.Fahrenheit {
		for i, v := range f.Values {
			f.Values[i] = v*9/5 + 32 // a missing value stays missing
		}
	}
	o := tuimaps.TemperatureGrid(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values, Lines: true}, unit, valid) // one look in both modes (D-102, go-tuiMaps L-15.4)
	o.Keeps = anchor.Sub(valid) + 3*time.Hour // current past the next refresh: an hour past is that hour's, not stale
	return o, true
}

// tempRequestBytes is one lattice's answer on the wire, measured: 63 KB for
// 96 points from Open-Meteo, 110 KB for 100 from NDFD (2026-09-26).
const tempRequestBytes = 80_000

// tempLayerCost is what the temperature would fetch in a refresh as if
// nothing were held: a request a box.
func tempLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := len(radar.BoxesFor(in.region, in.view))
	return int64(boxes) * tempRequestBytes, boxes
}

// tempHosts are the temperature's entries for the Status window's MAP block.
func tempHosts() []tty.MapSource {
	return hostsFor(temperature.Hosts(), "temperatures at points across fixed boxes around the region shown (never the view itself)")
}
