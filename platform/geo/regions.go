package geo

// regions.go — the regions a map is held inside (0.18.0 W4.1, FR-2.1, D-28).
//
// THE BOUND IS PER REGION, NOT ONE RECTANGLE (D-28). A single US-centred box
// excluded Anchorage and San Juan; a global frame is never drawn (D-8). Each
// region is every place and every water the station's APIs cover there -
// land, coast, the marine zones off it - boxed with a margin for its waters,
// and the map is never wider than the region its selected place is in.

import "math"

// The regions, by name.
const (
	RegionContiguous = "the contiguous United States"
	RegionAlaska     = "Alaska"
	RegionHawaii     = "Hawaii"
	RegionCaribbean  = "Puerto Rico and the U.S. Virgin Islands"
	RegionMarianas   = "Guam and the Northern Mariana Islands"
	RegionSamoa      = "American Samoa"
)

// Region is a box in degrees. West may be east of East: the box then crosses
// the antimeridian, as Alaska's does to hold the western Aleutians.
type Region struct {
	Name       string
	W, S, E, N float64
}

// regions is the set, each with its coastal and marine waters.
var regions = []Region{
	{Name: RegionContiguous, W: -130, S: 23, E: -64, N: 50},    // the Gulf, the Great Lakes and both coasts' waters
	{Name: RegionAlaska, W: 170, S: 50, E: -129, N: 73},        // across the antimeridian to Attu; the Arctic coast
	{Name: RegionHawaii, W: -162.5, S: 17.5, E: -153.5, N: 23}, // every inhabited island and its channels
	{Name: RegionCaribbean, W: -68.5, S: 17, E: -64, N: 19},
	{Name: RegionMarianas, W: 144, S: 12.5, E: 146.5, N: 21},
	{Name: RegionSamoa, W: -171.5, S: -15, E: -168, N: -10.5},
}

// Regions is every region, in the order they are looked for.
func Regions() []Region { return append([]Region(nil), regions...) }

// RegionOf is the region a place is in, and false for a place in none: a
// stated state, never a wider view (FR-2.5).
func RegionOf(lat, lon float64) (Region, bool) {
	for _, r := range regions {
		if r.Contains(lat, lon) {
			return r, true
		}
	}
	return Region{}, false
}

// Contains reports whether a place is inside the region.
func (r Region) Contains(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < r.S || lat > r.N {
		return false
	}
	lon = math.Mod(lon+540, 360) - 180 // into [-180, 180)
	if r.W <= r.E {
		return lon >= r.W && lon <= r.E
	}
	return lon >= r.W || lon <= r.E // across the antimeridian
}

// Centre is the middle of the region's box.
func (r Region) Centre() (lat, lon float64) {
	width := r.E - r.W
	if width < 0 {
		width += 360
	}
	lon = math.Mod(r.W+width/2+540, 360) - 180
	return (r.S + r.N) / 2, lon
}
