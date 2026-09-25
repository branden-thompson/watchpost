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

// zonePrefixRegions are the zone-code prefixes outside the contiguous United
// States: the states and territories, and the marine areas off them. Every
// other state's code, and the contiguous marine areas, are contiguous.
var zonePrefixRegions = map[string]string{
	"AK": RegionAlaska, "PK": RegionAlaska,
	"HI": RegionHawaii, "PH": RegionHawaii,
	"PR": RegionCaribbean, "VI": RegionCaribbean,
	"GU": RegionMarianas, "MP": RegionMarianas, "PM": RegionMarianas,
	"AS": RegionSamoa, "PS": RegionSamoa,
}

// contiguousZonePrefixes are the contiguous states' and waters' codes: the
// fifty less Alaska and Hawaii, DC, and the marine areas of both coasts, the
// Gulf and the Great Lakes.
var contiguousZonePrefixes = map[string]bool{
	"AL": true, "AZ": true, "AR": true, "CA": true, "CO": true, "CT": true, "DE": true, "DC": true, "FL": true, "GA": true,
	"ID": true, "IL": true, "IN": true, "IA": true, "KS": true, "KY": true, "LA": true, "ME": true, "MD": true, "MA": true,
	"MI": true, "MN": true, "MS": true, "MO": true, "MT": true, "NE": true, "NV": true, "NH": true, "NJ": true, "NM": true,
	"NY": true, "NC": true, "ND": true, "OH": true, "OK": true, "OR": true, "PA": true, "RI": true, "SC": true, "SD": true,
	"TN": true, "TX": true, "UT": true, "VT": true, "VA": true, "WA": true, "WV": true, "WI": true, "WY": true,
	"PZ": true, "AN": true, "AM": true, "GM": true, "LM": true, "LE": true, "LH": true, "LO": true, "LS": true, "LC": true, "SL": true,
}

// RegionOfZone is the region an NWS zone code is in - "TXZ277", "PKZ120" -
// for an alert that names zones and carries no point (0.18.0 W5.3). The
// Atlantic marine areas numbered 7xx lie off Puerto Rico and the Virgin
// Islands. A code of no known prefix is in no region.
func RegionOfZone(code string) (Region, bool) {
	if len(code) < 3 || (code[2] != 'Z' && code[2] != 'C') {
		return Region{}, false
	}
	prefix, name := code[:2], ""
	switch {
	case prefix == "AM" && len(code) > 3 && code[3] == '7':
		name = RegionCaribbean
	case zonePrefixRegions[prefix] != "":
		name = zonePrefixRegions[prefix]
	case contiguousZonePrefixes[prefix]:
		name = RegionContiguous
	default:
		return Region{}, false
	}
	for _, r := range regions {
		if r.Name == name {
			return r, true
		}
	}
	return Region{}, false
}
