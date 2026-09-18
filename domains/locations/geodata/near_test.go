package geodata

import (
	"testing"
)

// bonsall is the reference epicentre this feature was designed against — the
// HUM LEAD's own station (D-72).
const bonsallLat, bonsallLon = 33.2881, -117.2256

// THE FENCE HOLDS WHAT IS INSIDE IT, NEAREST FIRST.
//
// THE BROADCASTER'S CANDIDATES COME FROM HERE (F-81, ruled 2026-09-10). The
// Producer had only the listener's watchlist to offer, so a three-location
// watchlist capped the schedule at three cards and seven of the console's ten
// slots shimmered for ever (F-82). The data to answer it was ALREADY IN THE
// TREE — 34,106 US cities with population and 41,490 zip centroids — and no
// provider call is needed to reach it.
func TestNearReturnsTheFencesCitiesNearestFirst(t *testing.T) {
	idx := loadForTest(t)
	got := idx.Near(bonsallLat, bonsallLon, 20, 50)
	if len(got) < 8 {
		t.Fatalf("a 20-mile fence around Bonsall holds at least eight cities; got %d", len(got))
	}
	last := -1.0
	for _, c := range got {
		d := MilesBetween(bonsallLat, bonsallLon, c.Lat, c.Lon)
		if d > 20 {
			t.Errorf("%s is %.1f mi out and the fence is 20", c.Label(), d)
		}
		if d < last {
			t.Errorf("out of order: %s at %.1f mi follows %.1f", c.Label(), d, last)
		}
		last = d
		if c.Country != "US" {
			t.Errorf("%s is not in the US", c.Label())
		}
	}
	// "MAJOR LOCATIONS FIRST" NEEDS NO RANKING RULE: the city table is already
	// population-filtered, so distance order IS major-first. The HUM LEAD's own
	// example — Fallbrook, Vista, Oceanside, Temecula — falls out of it.
	head := got[0].Label() + " " + got[1].Label()
	if head != "Vista, CA Fallbrook, CA" && head != "Fallbrook, CA Vista, CA" {
		t.Errorf("the two nearest cities to Bonsall are Vista and Fallbrook; got %q", head)
	}
}

// A LIMIT IS A LIMIT, and it keeps the NEAREST rather than whichever the scan
// met first.
func TestNearHonoursItsLimit(t *testing.T) {
	idx := loadForTest(t)
	full := idx.Near(bonsallLat, bonsallLon, 50, 500)
	if len(full) < 20 {
		t.Fatalf("a 50-mile fence holds plenty; got %d", len(full))
	}
	short := idx.Near(bonsallLat, bonsallLon, 50, 5)
	if len(short) != 5 {
		t.Fatalf("the limit is five; got %d", len(short))
	}
	for i, c := range short {
		if c.Label() != full[i].Label() {
			t.Errorf("row %d is %q, want the %q the unlimited scan found", i, c.Label(), full[i].Label())
		}
	}
}

// AND AN EMPTY FENCE IS EMPTY, WHICH IS A REAL ANSWER.
//
// MEASURED, AND IT IS WHY THE RADIUS RULING MATTERS: a two-mile fence around
// Bonsall holds NO city at all, and a ten-mile one holds two. The floor is legal
// and it does not fill a ten-slot line-up (F-83).
func TestATightFenceHoldsAlmostNothing(t *testing.T) {
	idx := loadForTest(t)
	if got := idx.Near(bonsallLat, bonsallLon, 2, 50); len(got) != 0 {
		t.Errorf("a two-mile fence around Bonsall holds no city; got %d", len(got))
	}
	if got := idx.Near(bonsallLat, bonsallLon, 0, 50); len(got) != 0 {
		t.Errorf("a fence with no radius admits nothing; got %d", len(got))
	}
}

// THE ZIP TIER IS THE HYPER-LOCAL ONE (D-72). The HUM LEAD: "the big value here
// is the hyper local station reports."
//
// It is where Bonsall's own 92003 lives, and where the `92057` in his example
// comes from — the city table has neither, because both are too small for it.
func TestNearZipsReachesWhereTheCityTableDoesNot(t *testing.T) {
	idx := loadForTest(t)
	got := idx.NearZips(bonsallLat, bonsallLon, 20, 200)
	if len(got) < 20 {
		t.Fatalf("a 20-mile fence holds many zip centroids; got %d", len(got))
	}
	want := map[string]bool{"92003": false, "92057": false}
	last := -1.0
	for _, z := range got {
		d := MilesBetween(bonsallLat, bonsallLon, z.Lat, z.Lon)
		if d > 20 {
			t.Errorf("%s is %.1f mi out and the fence is 20", z.Zip, d)
		}
		if d < last {
			t.Errorf("out of order: %s at %.1f mi follows %.1f", z.Zip, d, last)
		}
		last = d
		if _, ok := want[z.Zip]; ok {
			want[z.Zip] = true
		}
	}
	for zip, found := range want {
		if !found {
			t.Errorf("%s is inside a 20-mile fence around Bonsall and the scan missed it", zip)
		}
	}
}

func loadForTest(t *testing.T) *Index {
	t.Helper()
	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return idx
}
