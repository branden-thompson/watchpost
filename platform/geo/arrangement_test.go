package geo

import "testing"

// TestTheRegionsStandInTheirArrangement is 0.18.0 D-77 (UAT-1 U1-40): the
// regions are laid out as the HUM LEAD drew them - Alaska above; beneath, west
// to east, Guam and the Northern Marianas, American Samoa, Hawaii, the
// contiguous United States and the Caribbean - so an edge leads to its
// neighbour, and each region has its number, 1 to 6.
func TestTheRegionsStandInTheirArrangement(t *testing.T) {
	for _, c := range []struct {
		from string
		dir  Direction
		to   string
	}{
		{RegionContiguous, East, RegionCaribbean},
		{RegionCaribbean, West, RegionContiguous},
		{RegionContiguous, West, RegionHawaii},
		{RegionHawaii, East, RegionContiguous},
		{RegionHawaii, West, RegionSamoa},
		{RegionSamoa, East, RegionHawaii},
		{RegionSamoa, West, RegionMarianas},
		{RegionMarianas, East, RegionSamoa},
		{RegionContiguous, North, RegionAlaska},
		{RegionHawaii, North, RegionAlaska},
		{RegionAlaska, South, RegionContiguous},
	} {
		got, ok := Neighbour(c.from, c.dir)
		if !ok || got.Name != c.to {
			t.Errorf("%s, %v: %q (%v), want %s", c.from, c.dir, got.Name, ok, c.to)
		}
	}
	for _, c := range []struct {
		from string
		dir  Direction
	}{{RegionCaribbean, East}, {RegionMarianas, West}, {RegionAlaska, North}, {RegionContiguous, South}} {
		if got, ok := Neighbour(c.from, c.dir); ok {
			t.Errorf("%s, %v leads to %s; the arrangement ends there", c.from, c.dir, got.Name)
		}
	}
	want := []string{RegionContiguous, RegionAlaska, RegionHawaii, RegionCaribbean, RegionSamoa, RegionMarianas}
	for i, name := range want {
		r, ok := RegionNumbered(i + 1)
		if !ok || r.Name != name {
			t.Errorf("region %d is %q, want %s", i+1, r.Name, name)
		}
		if n := NumberOf(name); n != i+1 {
			t.Errorf("%s is numbered %d, want %d", name, n, i+1)
		}
	}
	if _, ok := RegionNumbered(0); ok {
		t.Error("region 0 exists")
	}
	if _, ok := RegionNumbered(7); ok {
		t.Error("region 7 exists")
	}
}

// TestHawaiiAndTheCaribbeanHoldTheirWeather is D-91 (UAT-2 U2-8, U2-9): the
// two regions reach the sea around them - south of the Big Island, where the
// hurricane was; the Caribbean Sea south of Puerto Rico - and no lower-48
// place falls into either.
func TestHawaiiAndTheCaribbeanHoldTheirWeather(t *testing.T) {
	for _, c := range []struct {
		lat, lon float64
		want     string
	}{
		{15, -155, RegionHawaii}, {21.3, -157.8, RegionHawaii}, {14, -66, RegionCaribbean}, {18.4, -66.1, RegionCaribbean},
		{25.8, -80.2, RegionContiguous}, {24.6, -81.8, RegionContiguous}, // Miami and Key West stay in the lower 48
	} {
		if r, ok := RegionOf(c.lat, c.lon); !ok || r.Name != c.want {
			t.Errorf("%v,%v is in %q, want %s", c.lat, c.lon, r.Name, c.want)
		}
	}
	us, _ := regionNamed(RegionContiguous)
	if pr, _ := regionNamed(RegionCaribbean); pr.N > us.S {
		t.Errorf("the Caribbean's box reaches %.0f°N, into the lower 48's (from %.0f°N): a view there would be bound to the wrong region", pr.N, us.S)
	}
	for _, name := range []string{RegionHawaii, RegionCaribbean} {
		r, _ := regionNamed(name)
		if r.E-r.W < 18 || r.N-r.S < 11 {
			t.Errorf("%s is %.0f° by %.0f°: too small to see the weather around it", name, r.E-r.W, r.N-r.S)
		}
	}
}
