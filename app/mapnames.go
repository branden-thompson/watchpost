package app

// mapnames.go — the map's title names what is in view (0.18.0 UAT-1 U1-12,
// D-64), by scale, from the station's own city index and the regions: no
// request, no new data.

import (
	"math"
	"strings"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// The scales, by how many kilometres the view is across (D-64): under the
// first, a place; under the second, the part of a state; under the third, the
// state; wider, the region.
const (
	titlePlaceKm = 80
	titlePartKm  = 700
	titleStateKm = 1500
	kmPerMile    = 1.609344
)

// mapAreaNamer is the window's namer over an index: nil names nothing, and
// the window draws the selected place.
func mapAreaNamer(idx *geodata.Index) func(tuimaps.LonLat, float64) string {
	return func(at tuimaps.LonLat, widthKm float64) string {
		if idx == nil {
			return ""
		}
		region, ok := geo.RegionOf(at.Lat, at.Lon)
		if !ok {
			return ""
		}
		if widthKm >= titleStateKm {
			return region.Name
		}
		near := idx.Near(at.Lat, at.Lon, max(widthKm/2, 50)/kmPerMile, 20)
		if len(near) == 0 {
			return region.Name // open water: the region is what is in view
		}
		if widthKm < titlePlaceKm {
			return placeInView(near).Label()
		}
		state := near[0].State
		if widthKm >= titlePartKm {
			return geodata.StateName(state)
		}
		return partOfState(idx.StateExtents()[state], at) + " " + geodata.StateName(state)
	}
}

// placeInView is the place nearest the centre that is a town of some size,
// else the nearest place: a hamlet at the centre names nothing a listener
// knows.
func placeInView(near []geodata.City) geodata.City {
	for _, c := range near {
		if c.Population >= 5000 {
			return c
		}
	}
	return near[0]
}

// partOfState is where a point sits in a state's extent, on the axis it sits
// further out on ("Southern"), both when it is far out on both
// ("Northwestern"), or "Central".
func partOfState(e geodata.Extent, at tuimaps.LonLat) string {
	if e.E <= e.W || e.N <= e.S {
		return "Central"
	}
	dx := (at.Lon-e.W)/(e.E-e.W) - 0.5
	dy := (at.Lat-e.S)/(e.N-e.S) - 0.5
	ns, ew := "North", "east"
	if dy < 0 {
		ns = "South"
	}
	if dx < 0 {
		ew = "west"
	}
	// Both only when far out on both: a place well to the north that is also
	// somewhat west is "Northern", as people say it.
	const strong, weak = 0.35, 0.15
	switch {
	case math.Abs(dx) >= strong && math.Abs(dy) >= strong:
		return ns + ew + "ern" // "Northwestern"
	case math.Max(math.Abs(dx), math.Abs(dy)) < weak:
		return "Central"
	case math.Abs(dy) >= math.Abs(dx):
		return ns + "ern"
	}
	return strings.ToUpper(ew[:1]) + ew[1:] + "ern"
}
