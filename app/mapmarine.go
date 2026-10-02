package app

// mapmarine.go — the sea's stations on the map (0.18.0 D-127, D-128): the
// buoys with their latest readings, from NDBC's one national file; the tide
// stations with their next high or low while few enough are in view, a
// request a station. Each its own row, off by default, and asked only while
// on - the tides are a request a station.

import (
	"context"
	"math"
	"strconv"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/units"
)

// The sea's stations, registered: off by default (D-127, D-128).
func init() {
	registerMapLayer(mapLayer{key: tty.BuoyLayer, label: "Buoys", on: false, cost: buoyLayerCost, chips: []string{"NDBC"}})
	registerMapLayer(mapLayer{key: tty.TideLayer, label: "Tides", on: false, cost: tideLayerCost, chips: []string{"CO-OPS"}})
}

// buoyAge is the oldest reading drawn: two hours (D-127).
const buoyAge = 2 * time.Hour

// tideLabelMost is the most tide stations in view that are each labelled
// with their next high or low, a request each (D-128); past it, markers.
const tideLabelMost = 20

// marineKeeps is how long a station is drawn as current: its feed refreshes
// in minutes, and a refresh that drops it takes it off the map.
const marineKeeps = time.Hour

// tideMark is a tide station in view and, while few are, its next tide.
type tideMark struct {
	station coops.Station
	next    snapshot.TideEvent
	known   bool
}

// marineProviders are the places' own NDBC and CO-OPS providers, found among
// the marine set.
func (lp *livePipelines) marineProviders() (*ndbc.Provider, *coops.Provider) {
	var b *ndbc.Provider
	var c *coops.Provider
	for _, p := range lp.marine {
		switch p := p.(type) {
		case *ndbc.Provider:
			b = p
		case *coops.Provider:
			c = p
		}
	}
	return b, c
}

// buoysIn are the buoys' latest readings, asked only while their row is on.
func (lp *livePipelines) buoysIn(ctx context.Context, ask tty.MapAsk) []ndbc.Obs {
	b, _ := lp.marineProviders()
	if !ask.Buoys || b == nil {
		return nil
	}
	obs, _ := b.LatestObs(ctx)
	return obs
}

// tidesIn are the tide stations in view, asked only while their row is on:
// each with its next tide while few enough are in view.
func (lp *livePipelines) tidesIn(ctx context.Context, ask tty.MapAsk, now time.Time) []tideMark {
	_, c := lp.marineProviders()
	if !ask.Tides || c == nil {
		return nil
	}
	stations, err := c.TideStations(ctx)
	if err != nil {
		return nil
	}
	in := stationsIn(stations, ask.View)
	out := make([]tideMark, len(in))
	for i, s := range in {
		out[i].station = s
		if len(in) <= tideLabelMost {
			out[i].next, err = c.NextTide(ctx, s.ID, now)
			out[i].known = err == nil
		}
	}
	return out
}

// tidesHeld is the tide stations in view that the provider already holds,
// unlabelled and asking nothing: what the estimate counts (W14, C-2).
func (lp *livePipelines) tidesHeld(ask tty.MapAsk) []tideMark {
	_, c := lp.marineProviders()
	if !ask.Tides || c == nil {
		return nil
	}
	in := stationsIn(c.HeldTideStations(), ask.View)
	out := make([]tideMark, len(in))
	for i, s := range in {
		out[i].station = s
	}
	return out
}

// stationsIn is the stations inside the view.
func stationsIn(stations []coops.Station, view tty.MapView) []coops.Station {
	var in []coops.Station
	for _, s := range stations {
		if view.Contains(s.Lat, s.Lon) {
			in = append(in, s)
		}
	}
	return in
}

// buoyOverlay is the buoys in view that read in the last two hours, each a
// marker in the buoy's role labelled with its waves and water, or its wind
// where it has no waves (D-127) - one overlay for them all (D-129: an
// overlay a station starved the basemap, UAT-2 U2-34).
func buoyOverlay(obs []ndbc.Obs, view tty.MapView, now time.Time, imperial bool) (tuimaps.Overlay, bool) {
	var feats []tuimaps.Feature
	for _, o := range obs {
		if !view.Contains(o.Lat, o.Lon) || now.Sub(o.At) > buoyAge {
			continue
		}
		label := buoyLabel(o, imperial)
		if label == "" {
			continue // nothing read: nothing to say
		}
		feats = append(feats, tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: o.Lon, Lat: o.Lat}}}, Role: tuimaps.Buoy, Label: label, ID: o.ID})
	}
	if len(feats) == 0 {
		return tuimaps.Overlay{}, false
	}
	return tuimaps.Overlay{ID: tty.BuoyLayer + "/buoys", Valid: overlayStamp(now), Keeps: marineKeeps, Credit: ndbc.Attribution, Features: feats}, true
}

// buoyLabel is a buoy's words: "4ft 73°", "1.2m 23°", or "12kt" where it
// read no waves.
func buoyLabel(o ndbc.Obs, imperial bool) string {
	if o.WaveM == nil {
		if o.WindMS == nil {
			return ""
		}
		return strconv.Itoa(int(math.Round(units.KnotsOf(*o.WindMS)))) + "kt" // wind at sea is in knots
	}
	label := strconv.FormatFloat(*o.WaveM, 'f', 1, 64) + "m"
	if imperial {
		label = strconv.Itoa(int(math.Round(units.FeetOf(*o.WaveM)))) + "ft"
	}
	if o.WaterC != nil {
		water := *o.WaterC
		if imperial {
			water = units.FahrenheitOf(water)
		}
		label += " " + strconv.Itoa(int(math.Round(water))) + "°"
	}
	return label
}

// tideOverlay is the tide stations in view, each a marker in the tide's
// role, labelled with its next high or low while twenty or fewer are in view
// (D-128) - asked, through next, only then; one overlay for them all (D-129).
func tideOverlay(stations []coops.Station, view tty.MapView, now time.Time, imperial bool, clock render.Clock, zone *time.Location, next func(id string) (snapshot.TideEvent, bool)) (tuimaps.Overlay, bool) {
	var in []coops.Station
	for _, s := range stations {
		if view.Contains(s.Lat, s.Lon) {
			in = append(in, s)
		}
	}
	if len(in) == 0 {
		return tuimaps.Overlay{}, false
	}
	feats := make([]tuimaps.Feature, 0, len(in))
	for _, s := range in {
		label := ""
		if len(in) <= tideLabelMost {
			if e, ok := next(s.ID); ok {
				label = tideLabel(e, imperial, clock, zone)
			}
		}
		feats = append(feats, tuimaps.Feature{Kind: tuimaps.Point, Rings: [][]tuimaps.LonLat{{{Lon: s.Lon, Lat: s.Lat}}}, Role: tuimaps.Tide, Label: label, ID: s.ID})
	}
	return tuimaps.Overlay{ID: tty.TideLayer + "/tides", Valid: overlayStamp(now), Keeps: marineKeeps, Credit: "NOAA CO-OPS tide predictions", Features: feats}, true
}

// tideLabel is a next tide's words: "H 3.9ft 4:01 PM", or in metres.
func tideLabel(e snapshot.TideEvent, imperial bool, clock render.Clock, zone *time.Location) string {
	height := strconv.FormatFloat(e.Height, 'f', 1, 64) + "m"
	if imperial {
		height = strconv.FormatFloat(units.FeetOf(e.Height), 'f', 1, 64) + "ft"
	}
	return e.Type + " " + height + " " + clock.Time(e.Time.In(zone))
}

// buoyBytes is NDBC's national file on the wire, measured: 103 KB
// (2026-09-28).
const buoyBytes = 103_000

// buoyLayerCost is NDBC's one file (D-127).
func buoyLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	return buoyBytes, 1
}

// tideBytes is one station's predictions on the wire: about 2 KB.
const tideBytes = 2_000

// tideLayerCost is a request a tide station in view while twenty or fewer
// are, counted from the tide stations the estimate holds; the station list is
// the places' own daily read.
func tideLayerCost(in mapInputs) (int64, int) {
	if in.region == "" || len(in.tides) == 0 || len(in.tides) > tideLabelMost {
		return 0, 0
	}
	return int64(len(in.tides)) * tideBytes, len(in.tides)
}

// marineHosts are the sea's stations' entries for the Status window's MAP
// block.
func marineHosts() []tty.MapSource {
	return []tty.MapSource{
		{Name: "NOAA NDBC", Host: "www.ndbc.noaa.gov", Layers: "buoys"},
		{Name: "NOAA CO-OPS", Host: "api.tidesandcurrents.noaa.gov", Layers: "tides"},
	}
}
