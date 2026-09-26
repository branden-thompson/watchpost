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
