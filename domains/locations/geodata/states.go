package geodata

// states.go — the states' extents and names, for the map's title (0.18.0
// UAT-1 U1-12, D-64): "Southern California" is where a view's centre sits in
// the state's extent, and the extent is its own cities'.

import (
	"sync"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// Extent is a state's box in degrees, from its cities.
type Extent = geo.Box

// StateExtents is each US state's (and territory's) box over its cities,
// computed once, in one pass.
func (i *Index) StateExtents() map[string]Extent {
	i.extentsOnce.Do(func() {
		out := map[string]Extent{}
		for _, p := range i.usCities() { // bounded by the index (P10-02); the scan's own parsed rows
			st, lat, lon := field(i.cities, p.off, 2), p.lat, p.lon
			if st == "" {
				continue
			}
			e, ok := out[st]
			if !ok {
				e = Extent{W: lon, S: lat, E: lon, N: lat}
			}
			e.W, e.E, e.S, e.N = min(e.W, lon), max(e.E, lon), min(e.S, lat), max(e.N, lat)
			out[st] = e
		}
		i.extents = out
	})
	return i.extents
}

// stateNames are the states' and territories' names by postal code.
var stateNames = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas", "CA": "California", "CO": "Colorado",
	"CT": "Connecticut", "DE": "Delaware", "DC": "the District of Columbia", "FL": "Florida", "GA": "Georgia",
	"HI": "Hawaii", "ID": "Idaho", "IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas", "KY": "Kentucky",
	"LA": "Louisiana", "ME": "Maine", "MD": "Maryland", "MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota",
	"MS": "Mississippi", "MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada", "NH": "New Hampshire",
	"NJ": "New Jersey", "NM": "New Mexico", "NY": "New York", "NC": "North Carolina", "ND": "North Dakota",
	"OH": "Ohio", "OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island",
	"SC": "South Carolina", "SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah", "VT": "Vermont",
	"VA": "Virginia", "WA": "Washington", "WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
	"PR": "Puerto Rico", "VI": "the U.S. Virgin Islands", "GU": "Guam", "MP": "the Northern Mariana Islands",
	"AS": "American Samoa",
}

// StateName is a postal code's state or territory; an unknown code reads as
// itself.
func StateName(code string) string {
	if n, ok := stateNames[code]; ok {
		return n
	}
	return code
}

// extentsCache is the index's computed extents.
type extentsCache struct {
	extentsOnce sync.Once
	extents     map[string]Extent
}
