// Package propanswers is the Propagation mode's answers, worked out from a
// go-ionomaps snapshot and an ask (0.19.0 D-212): the near-vertical area's
// bands and each target's, the map centre's, a frequency's reach, and their
// limits and words. It holds no state and draws nothing: the TTY renders
// what it answers, and a later mode - Analyst, Scout (D-27, D-28) - asks it
// with targets of its own.
package propanswers

import (
	"errors"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// Target is a place the chart answers for: its name, its short name and
// where it is.
type Target struct {
	Name, Short string
	At          ionomaps.LatLon
}

// Continents are the chart's default targets (FR-1.9, A-45): on each
// inhabited continent, the city where its amateur radio is busiest.
var Continents = []Target{
	{"North America", "NA", ionomaps.LatLon{Lat: 41.88, Lon: -87.63}},  // Chicago
	{"South America", "SA", ionomaps.LatLon{Lat: -23.55, Lon: -46.63}}, // São Paulo
	{"Europe", "EU", ionomaps.LatLon{Lat: 50.11, Lon: 8.68}},           // Frankfurt
	{"Africa", "AF", ionomaps.LatLon{Lat: -26.20, Lon: 28.05}},         // Johannesburg
	{"Asia", "AS", ionomaps.LatLon{Lat: 35.68, Lon: 139.69}},           // Tokyo
	{"Oceania", "OC", ionomaps.LatLon{Lat: -33.87, Lon: 151.21}},       // Sydney
}

// Near reports whether a target is within the near-vertical area around an
// origin: no path to it, its answer the area's.
func (t Target) Near(origin ionomaps.LatLon, radiusKm float64) bool {
	if !(radiusKm >= 0) {
		return false // a radius negative or not a number holds nothing
	}
	if !(t.At.Lat >= -90 && t.At.Lat <= 90) {
		return false // a target off the globe is near nothing
	}
	return geo.HaversineKM(origin.Lat, origin.Lon, t.At.Lat, t.At.Lon) <= radiusKm
}

// Ask is what a chart is worked out from: the origin, the near-vertical
// radius and the targets, in their order (D-212: a found place, D-204, or a
// shortlist, D-27, beside or in place of the continents).
type Ask struct {
	Origin   ionomaps.LatLon
	RadiusKm float64
	Targets  []Target
}

// Chart is a chart's answers: the near-vertical area's, and each target's
// in the ask's order.
type Chart struct {
	Local   ionomaps.Answer
	Targets []ionomaps.Answer
}

// The asks refused. ErrAnswers is an answer of another length than the
// places asked: the library's to fix, said in the diagnostics (D-124).
// ErrAsk is an ask no answer can be worked out from: a radius negative or not
// a number, or a place off the globe (a NaN among them, L-F12).
var (
	ErrAnswers = errors.New("go-ionomaps answered a different number of places than were asked")
	ErrAsk     = errors.New("propanswers: the ask's radius is not a distance, or a place in it is off the globe")
)

// WorkChart works a chart out (W5.1): the near-vertical area's bands, and
// each target's path from the origin - within the area, the area's answer.
// It takes about 5 ms; a caller keeps it off the UI goroutine (FR-1.17).
func WorkChart(snap ionomaps.Snapshot, ask Ask) (Chart, error) {
	if !(ask.RadiusKm >= 0) {
		return Chart{}, ErrAsk
	}
	if !(ask.Origin.Lat >= -90 && ask.Origin.Lat <= 90) {
		return Chart{}, ErrAsk
	}
	if !(ask.Origin.Lon >= -180 && ask.Origin.Lon <= 180) {
		return Chart{}, ErrAsk
	}
	local, err := snap.Bands([]ionomaps.LatLon{ask.Origin}, ask.RadiusKm)
	if err != nil {
		return Chart{}, err
	}
	to := make([]ionomaps.LatLon, len(ask.Targets))
	for i, t := range ask.Targets {
		if !(t.At.Lat >= -90 && t.At.Lat <= 90) {
			return Chart{}, ErrAsk
		}
		if !(t.At.Lon >= -180 && t.At.Lon <= 180) {
			return Chart{}, ErrAsk
		}
		to[i] = t.At
	}
	paths, err := snap.Path(ask.Origin, to)
	if err != nil {
		return Chart{}, err
	}
	if len(local) != 1 || len(paths) != len(ask.Targets) {
		return Chart{}, ErrAnswers
	}
	for i, t := range ask.Targets {
		if t.Near(ask.Origin, ask.RadiusKm) {
			paths[i] = local[0] // no path: the near-vertical area's answer
		}
	}
	return Chart{Local: local[0], Targets: paths}, nil
}

// Centre is the map centre's answer (FR-1.11, D-180): the bands from the
// origin to it, its nearest station, and how far it is.
type Centre struct {
	Bands   []ionomaps.BandStatus
	Nearest ionomaps.Nearest
	Km      float64
}

// WorkCentre is the centre's answer: within the near-vertical radius of the
// origin, the area's; farther, the path's.
func WorkCentre(snap ionomaps.Snapshot, origin, centre ionomaps.LatLon, radiusKm float64) (Centre, error) {
	if !(radiusKm >= 0) {
		return Centre{}, ErrAsk
	}
	if !(origin.Lat >= -90 && origin.Lat <= 90) {
		return Centre{}, ErrAsk
	}
	if !(origin.Lon >= -180 && origin.Lon <= 180) {
		return Centre{}, ErrAsk
	}
	if !(centre.Lat >= -90 && centre.Lat <= 90) {
		return Centre{}, ErrAsk
	}
	if !(centre.Lon >= -180 && centre.Lon <= 180) {
		return Centre{}, ErrAsk
	}
	c := Centre{Km: geo.HaversineKM(origin.Lat, origin.Lon, centre.Lat, centre.Lon)}
	var got []ionomaps.Answer
	var err error
	if c.Km <= radiusKm {
		got, err = snap.Bands([]ionomaps.LatLon{origin}, radiusKm)
	} else {
		got, err = snap.Path(origin, []ionomaps.LatLon{centre})
	}
	if err == nil && len(got) == 1 {
		c.Bands, c.Nearest = got[0].Bands, got[0].Nearest
	}
	return c, err
}
