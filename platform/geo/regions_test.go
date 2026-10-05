package geo

import "testing"

// TestEveryCoveredPlaceHasARegion is 0.18.0 W4.1 (FR-2.1, FR-2.5, D-28): every
// region the station's APIs cover holds its places and its waters - the
// contiguous US with the Gulf and the Great Lakes, Alaska across the
// antimeridian, Hawaii, the Caribbean and Pacific territories - and a place in
// none of them is in none.
func TestEveryCoveredPlaceHasARegion(t *testing.T) {
	for _, c := range []struct {
		name     string
		lat, lon float64
		region   string
	}{
		{"Oceanside, CA", 33.2, -117.38, RegionContiguous},
		{"Key West, FL", 24.55, -81.78, RegionContiguous},
		{"a Gulf of Mexico buoy", 26.0, -90.0, RegionContiguous},
		{"a Lake Superior buoy", 47.6, -86.6, RegionContiguous},
		{"Bar Harbor, ME", 44.39, -68.2, RegionContiguous},
		{"Anchorage, AK", 61.22, -149.9, RegionAlaska},
		{"Utqiagvik, AK", 71.29, -156.79, RegionAlaska},
		{"Adak, AK", 51.88, -176.66, RegionAlaska},
		{"Attu, AK, west of the antimeridian", 52.93, 173.2, RegionAlaska},
		{"Attu written west of -180", 52.93, -186.8, RegionAlaska},
		{"Honolulu written east of 180", 21.3, 202.14, RegionHawaii},
		{"Hilo, HI", 19.72, -155.08, RegionHawaii},
		{"Lihue, HI", 21.98, -159.37, RegionHawaii},
		{"San Juan, PR", 18.47, -66.11, RegionCaribbean},
		{"Charlotte Amalie, VI", 18.34, -64.93, RegionCaribbean},
		{"Hagatna, GU", 13.48, 144.75, RegionMarianas},
		{"Saipan, MP", 15.18, 145.75, RegionMarianas},
		{"Pago Pago, AS", -14.28, -170.7, RegionSamoa},
	} {
		r, ok := RegionOf(c.lat, c.lon)
		if !ok || r.Name != c.region {
			t.Errorf("%s is in %q (%v), want %s", c.name, r.Name, ok, c.region)
		}
	}
	for _, c := range []struct {
		name     string
		lat, lon float64
	}{{"London", 51.5, -0.12}, {"Tokyo", 35.68, 139.69}, {"Sydney", -33.87, 151.2}, {"Mexico City", 19.43, -99.13}} {
		if r, ok := RegionOf(c.lat, c.lon); ok {
			t.Errorf("%s is in %s; it is in no region the station covers", c.name, r.Name)
		}
	}
	for _, r := range Regions() {
		if !r.Contains(r.Centre()) {
			t.Errorf("%s does not hold its own centre", r.Name)
		}
	}
}
