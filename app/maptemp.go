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
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The temperature, registered: on by default - the map shows what is real
// (D-76) - and switched in the Overlays menu.
func init() {
	registerMapLayer(mapLayer{key: tty.TemperatureLayer, label: "Temperature", on: true, cost: tempLayerCost})
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
	if ask.Forecast && !ask.TempOpenMeteo && ts.ndfd.Covers(ask.Region) {
		return ts.ndfd
	}
	return ts.om
}

// tempFrameNote is what Radar mode says of its temperature (D-96).
const tempFrameNote = "Each radar frame draws its own hour's temperature."

// mapTemperature is the window's temperature for the mode, every grid built
// with its span.
func (lp *livePipelines) mapTemperature(ctx context.Context, ask tty.MapAsk) tty.MapTemperature {
	if lp.temp == nil || ask.Region == "" {
		return tty.MapTemperature{}
	}
	return buildTemperature(ctx, lp.temp.sourceFor(ask), ask, time.Now())
}

// buildTemperature fetches each box's lattice and makes its grids.
func buildTemperature(ctx context.Context, src temperature.Source, ask tty.MapAsk, now time.Time) tty.MapTemperature {
	out := tty.MapTemperature{Source: src.Name()}
	boxes := radar.BoxesFor(ask.Region, ask.View)
	if len(boxes) == 0 {
		out.Notes = []string{"No temperature is drawn for " + ask.Region + "."}
		return out
	}
	unit := tuimaps.Celsius
	if ask.Fahrenheit {
		unit = tuimaps.Fahrenheit
	}
	missing := map[string]bool{}
	for _, b := range boxes {
		lat := temperature.LatticeFor(b.Name, geo.Box{W: b.W, S: b.S, E: b.E, N: b.N})
		s, err := src.Fetch(ctx, lat, now)
		if err != nil {
			missing["Temperature is unavailable: "+src.Name()+" did not answer."] = true
			continue
		}
		if ask.Forecast {
			forecastGrids(&out, s, b.Name, ask.Anchor, unit, missing)
			continue
		}
		out.Overlays = append(out.Overlays, hourGrids(s, b.Name, now, unit)...)
	}
	if src.Name() == "Open-Meteo" {
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

// hourGrids are Radar mode's grids: every hour up to now, each drawn during
// its hour (D-96).
func hourGrids(s temperature.Series, box string, now time.Time, unit tuimaps.Unit) []tuimaps.Overlay {
	var out []tuimaps.Overlay
	for i, h := range s.Hours {
		if h.After(now) {
			continue // a forecast hour: no radar frame is in it
		}
		o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/"+h.UTC().Format("2006-01-02T15"), s.Lattice, s.Hourly[i], unit, h, now)
		if !ok {
			continue
		}
		o.During = tuimaps.Span{From: h, Until: h.Add(time.Hour - time.Nanosecond)}
		out = append(out, o)
	}
	return out
}

// forecastGrids are Forecast mode's grids: Now, and each day's high and low,
// each drawn during its step (D-94, D-97). A day a source has nothing for is
// said, not drawn empty.
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
			o, ok := tempGrid(tty.TemperatureLayer+"/"+box+"/d"+string(rune('0'+k))+"/"+side.name, s.Lattice, side.vals, unit, anchor, anchor)
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
// listener's unit; false when every value is missing.
func tempGrid(id string, l temperature.Lattice, values []float64, unit tuimaps.Unit, valid, now time.Time) (tuimaps.Overlay, bool) {
	f := l.Interpolate(values)
	any := false
	for i, v := range f.Values {
		if math.IsNaN(v) {
			continue
		}
		any = true
		if unit == tuimaps.Fahrenheit {
			f.Values[i] = v*9/5 + 32
		}
	}
	if !any {
		return tuimaps.Overlay{}, false
	}
	o := tuimaps.TemperatureGrid(id, tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N,
		Cols: f.Cols, Rows: f.Rows, Values: f.Values}, unit, valid)
	o.Keeps = now.Sub(valid) + 2*time.Hour // current until the next refresh has long come: an hour past is that hour's, not stale
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
