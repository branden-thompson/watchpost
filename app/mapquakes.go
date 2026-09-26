package app

// mapquakes.go — the map's earthquakes (0.18.0 D-80, UAT-1 U1-43): the
// significant earthquakes the ticker already fetches from USGS - the ones the
// [w] window lists under Disasters - drawn where they are, when they are in
// view. A layer of its own, switched in the Overlays menu; it fetches nothing.

import (
	"strconv"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// quakeLayerKey is the earthquakes' key in the registry, and their overlays'
// ids' first part.
const quakeLayerKey = "quake"

// The earthquakes, registered: on by default, and free - the ticker holds them.
func init() {
	registerMapLayer(mapLayer{key: quakeLayerKey, label: "Earthquakes", on: true, cost: func(mapInputs) (int64, int) { return 0, 0 }})
}

// quakeKeeps is how long a quake is drawn as current: the feed is the past
// week's, and the feed dropping it is what takes it off the map.
const quakeKeeps = 8 * 24 * time.Hour

// quakesIn is the feed's earthquakes with a point inside the view.
func quakesIn(feed []globalfeed.Event, v tty.MapView) []globalfeed.Event {
	var out []globalfeed.Event
	for _, e := range feed {
		if e.Class == globalfeed.ClassQuake && e.HasPoint && v.Contains(e.Lat, e.Lon) {
			out = append(out, e)
		}
	}
	return out
}

// quakeOverlays is each quake as a circle at its epicentre, sized by its
// magnitude and labelled with it. Drawn in the track's colour and with no
// severity, so the library never reports a quake as an alert over a place.
func quakeOverlays(quakes []globalfeed.Event) []tuimaps.Overlay {
	var out []tuimaps.Overlay
	for _, e := range quakes {
		label, mag := "Earthquake", 0.0
		if e.Quake != nil && e.Quake.Mag != nil {
			mag = *e.Quake.Mag
			label = "M " + strconv.FormatFloat(mag, 'f', 1, 64)
		}
		out = append(out, tuimaps.Overlay{ID: quakeLayerKey + "/" + e.ID, Valid: e.At, Keeps: quakeKeeps,
			Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: e.Lon, Lat: e.Lat},
				RadiusKm: quakeRadiusKm(mag), Role: tuimaps.Track, Label: label, ID: e.ID}}})
	}
	return out
}

// quakeRadiusKm is a quake's circle: 10 km at magnitude 4 or less, doubling
// with each whole magnitude - 20 km at 5, 80 km at 7 - so the larger one is
// the larger mark, up to 400 km.
func quakeRadiusKm(mag float64) float64 {
	r := 10.0
	for m := 4.0; m < mag && r < 400; m++ {
		r *= 2
	}
	return min(r, 400)
}
