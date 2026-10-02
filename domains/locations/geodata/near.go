package geodata

// near.go — what is inside a fence, nearest first (D-72).
//
// THE BROADCASTER'S CANDIDATE LOCATIONS COME FROM HERE. With only the
// listener's WATCHLIST to offer, the schedule could never be deeper than the
// number of distinct places the listener happens to watch — often a handful,
// against a console that draws ten slots (F-81, F-82).
//
// NO PROVIDER CALL, AND NO NEW DATA. Both answers are already embedded in this
// package: 34,106 US cities with population and 41,490 zip centroids. That is
// also why "major locations first" needs no ranking rule — the city table is
// population-filtered, so ordering by DISTANCE is ordering by major-first, and
// the HUM LEAD's own example (Fallbrook, Vista, Oceanside, Temecula) falls
// straight out of it.
//
// TWO SCANS RATHER THAN ONE, because they answer different questions. The city
// table says what the REGION is; the zip table says what is HYPER-LOCAL — "the
// big value here is the hyper local station reports" — and it is the only one
// of the two that knows Bonsall's own 92003 exists.
//
// A FULL SCAN, DELIBERATELY. It is 34k haversines at startup and on a settings
// change, which measures in tens of milliseconds; a bounding-box prefilter would
// be an optimisation against a product that is not finished (D-53). The map's
// estimate scans too, on its UI goroutine, so the coordinates are parsed once
// and held (W14): 2 ms a view rather than 31.

import (
	"sort"
	"strconv"
	"sync"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/units"
)

// MilesBetween is the distance between two coordinates in STATUTE MILES, which
// is the unit the station's fence is set in.
//
// ONE OWNER FOR THE CONVERSION. `geo.HaversineKM` is the one owner of the
// distance; this is the one owner of "…in the miles the operator typed", so a
// fence and a card cannot disagree about how far away a place is.
func MilesBetween(lat1, lon1, lat2, lon2 float64) float64 {
	return units.MilesOf(geo.HaversineKM(lat1, lon1, lat2, lon2))
}

// Near is the US cities inside a fence of radiusMi around (lat, lon), nearest
// first, at most limit of them.
//
// A RADIUS OF ZERO ADMITS NOTHING, which matches `lineup.Fence`: a station with
// no fence set is not asking for a region, and the safe reading of an unset
// number is "nothing" rather than "everywhere".
func (i *Index) Near(lat, lon, radiusMi float64, limit int) []City {
	us := i.usCities()
	offs := i.nearOffsets(lat, lon, radiusMi, limit, len(us), func(n int) (int32, float64, float64, bool) {
		return us[n].off, us[n].lat, us[n].lon, true
	})
	out := make([]City, 0, len(offs))
	for _, o := range offs { // bounded by the limit (P10-02)
		out = append(out, i.parseCity(o))
	}
	return out
}

// placed is a US city row and its coordinates, parsed.
type placed struct {
	off      int32
	lat, lon float64
}

// placesCache is the US city rows with their coordinates, parsed once.
//
// PARSED ONCE, BECAUSE THE MAP SCANS ON ITS UI GOROUTINE (W14). The line-up
// scans on a settings change; the map's estimate reads the state under
// twenty-five points of the view on every open, feed landing and switch, and
// parsing the table each time costs 31 ms and a million allocations a view.
// Held, the coordinates cost about 0.8 MB; the scan is still a full one.
type placesCache struct {
	placesOnce sync.Once
	places     []placed
}

// usCities is every US city row whose coordinates parse, in the table's order.
func (i *Index) usCities() []placed {
	i.placesOnce.Do(func() {
		for _, off := range i.cityOffs { // bounded by the index (P10-02)
			if field(i.cities, off, 3) != "US" {
				continue
			}
			lat, err1 := strconv.ParseFloat(field(i.cities, off, 4), 64)
			lon, err2 := strconv.ParseFloat(field(i.cities, off, 5), 64)
			if err1 == nil && err2 == nil {
				i.places = append(i.places, placed{off, lat, lon})
			}
		}
	})
	return i.places
}

// NearZips is the zip centroids inside the same fence, nearest first.
//
// THE HYPER-LOCAL TIER. Many of them share a place — Vista has four inside
// twenty miles of Bonsall — and de-duplicating is the CALLER's, because what
// counts as one place depends on what the caller is building a list FOR.
func (i *Index) NearZips(lat, lon, radiusMi float64, limit int) []ZipRow {
	offs := i.nearOffsets(lat, lon, radiusMi, limit, len(i.zipOffs), func(n int) (int32, float64, float64, bool) {
		z := i.parseZip(i.zipOffs[n])
		return i.zipOffs[n], z.Lat, z.Lon, true
	})
	out := make([]ZipRow, 0, len(offs))
	for _, o := range offs { // bounded by the limit (P10-02)
		out = append(out, i.parseZip(o))
	}
	return out
}

// nearOffsets is the scan both tiers share: every row the `at` function can
// place, kept when it is inside the fence, sorted by distance, cut to the limit.
//
// ONE SCAN, TWO TABLES. The tables differ only in how a row yields a coordinate,
// and that difference is the argument — a second copy of "filter, sort, cut"
// would be the shape D-56 exists to stop, and this is exactly the kind of place
// it grows back.
func (i *Index) nearOffsets(lat, lon, radiusMi float64, limit, rows int, at func(n int) (off int32, rowLat, rowLon float64, ok bool)) []int32 {
	if radiusMi <= 0 || limit <= 0 {
		return nil
	}
	type hit struct {
		off int32
		mi  float64
	}
	var hits []hit
	for n := range rows { // bounded by the table (P10-02)
		off, rowLat, rowLon, ok := at(n)
		if !ok {
			continue
		}
		mi := MilesBetween(lat, lon, rowLat, rowLon)
		if mi > radiusMi {
			continue
		}
		hits = append(hits, hit{off, mi})
	}
	// A STABLE SORT, so two places at the same distance keep the table's own
	// order rather than a different one on every run — a line-up that reshuffled
	// between launches for no reason would be a station the operator cannot
	// learn.
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].mi < hits[b].mi })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]int32, 0, len(hits))
	for _, h := range hits { // bounded by the limit (P10-02)
		out = append(out, h.off)
	}
	return out
}
