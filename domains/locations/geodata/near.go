package geodata

// near.go — what is inside a fence, nearest first (D-72).
//
// THE BROADCASTER'S CANDIDATE LOCATIONS COME FROM HERE. The Producer had only
// the listener's WATCHLIST to offer, so the schedule could never be deeper than
// the number of distinct places the listener happened to watch — three, in the
// HUM LEAD's UAT, against a console that draws ten slots (F-81, F-82).
//
// NO PROVIDER CALL, AND NO NEW DATA. Both answers were already embedded in this
// package: 34,106 US cities with population and 41,490 zip centroids. That is
// also why "major locations first" needs no ranking rule — the city table is
// already population-filtered, so ordering by DISTANCE is ordering by
// major-first, and the HUM LEAD's own example (Fallbrook, Vista, Oceanside,
// Temecula) falls straight out of it.
//
// TWO SCANS RATHER THAN ONE, because they answer different questions. The city
// table says what the REGION is; the zip table says what is HYPER-LOCAL — "the
// big value here is the hyper local station reports" — and it is the only one
// of the two that knows Bonsall's own 92003 exists.
//
// A FULL SCAN, DELIBERATELY. It is 34k haversines at startup and on a settings
// change, which measures in tens of milliseconds; a bounding-box prefilter would
// be an optimisation against a product that is not finished (D-53).

import (
	"sort"
	"strconv"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// MilesBetween is the distance between two coordinates in STATUTE MILES, which
// is the unit the station's fence is set in.
//
// ONE OWNER FOR THE CONVERSION. `geo.HaversineKM` is the one owner of the
// distance; this is the one owner of "…in the miles the operator typed", so a
// fence and a card cannot disagree about how far away a place is.
func MilesBetween(lat1, lon1, lat2, lon2 float64) float64 {
	return geo.HaversineKM(lat1, lon1, lat2, lon2) * milesPerKM
}

// milesPerKM converts kilometres to statute miles.
const milesPerKM = 0.621371

// Near is the US cities inside a fence of radiusMi around (lat, lon), nearest
// first, at most limit of them.
//
// A RADIUS OF ZERO ADMITS NOTHING, which matches `lineup.Fence`: a station with
// no fence set is not asking for a region, and the safe reading of an unset
// number is "nothing" rather than "everywhere".
func (i *Index) Near(lat, lon, radiusMi float64, limit int) []City {
	offs := i.nearOffsets(lat, lon, radiusMi, limit, len(i.cityOffs), func(n int) (int32, float64, float64, bool) {
		off := i.cityOffs[n]
		if field(i.cities, off, 3) != "US" {
			return 0, 0, 0, false
		}
		la, err1 := strconv.ParseFloat(field(i.cities, off, 4), 64)
		lo, err2 := strconv.ParseFloat(field(i.cities, off, 5), 64)
		return off, la, lo, err1 == nil && err2 == nil
	})
	out := make([]City, 0, len(offs))
	for _, o := range offs { // bounded by the limit (P10-02)
		out = append(out, i.parseCity(o))
	}
	return out
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
