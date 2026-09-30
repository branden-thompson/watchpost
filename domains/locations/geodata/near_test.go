package geodata

import (
	"sort"
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

// THE MAP ASKS THIS ON ITS UI GOROUTINE (W14): the estimate reads the state
// under twenty-five points of the view, one scan each, on every open, feed
// landing and switch. Parsing each row's coordinates on every scan cost 31 ms
// and a million allocations a view (measured 2026-09-30); a scan now reads
// coordinates parsed once. The allocations are gated; the time is not (D-53).
const nearAllocs = 20 // measured 2026-09-30 (W14): pinned, lowered only

func TestAScanDoesNotParseTheTable(t *testing.T) {
	idx := loadForTest(t)
	idx.Near(bonsallLat, bonsallLon, 40, 1) // the coordinates are parsed once, here
	allocs := testing.AllocsPerRun(20, func() { idx.Near(bonsallLat, bonsallLon, 40, 1) })
	if allocs > nearAllocs {
		t.Errorf("a scan cost %.0f allocations; its budget is %d", allocs, nearAllocs)
	}
}

// AND THE ANSWERS ARE THE SAME ONES: every point of a grid across the country
// and its waters, at the map's radius and the Producer's, is answered as a scan
// of the table parsed row by row answers it - same cities, same order.
func TestAScanAnswersAsTheTableDoes(t *testing.T) {
	idx := loadForTest(t)
	table := parsedTable(idx)
	for lat := 18.0; lat <= 64; lat += 2.3 {
		for lon := -165.0; lon <= -66; lon += 3.1 {
			for _, fence := range []struct {
				mi    float64
				limit int
			}{{40, 1}, {20, 50}, {75, 20}} {
				got, want := idx.Near(lat, lon, fence.mi, fence.limit), referenceNear(table, lat, lon, fence.mi, fence.limit)
				if len(got) != len(want) {
					t.Fatalf("(%.1f, %.1f) within %.0f mi: %d cities, the table's scan finds %d", lat, lon, fence.mi, len(got), len(want))
				}
				for n := range got {
					if got[n] != want[n] {
						t.Fatalf("(%.1f, %.1f) within %.0f mi, row %d: %s, the table's scan says %s", lat, lon, fence.mi, n, got[n].Label(), want[n].Label())
					}
				}
			}
		}
	}
}

// parsedTable is every row of the city table, parsed, in the table's order.
func parsedTable(idx *Index) []City {
	out := make([]City, 0, len(idx.cityOffs))
	for _, off := range idx.cityOffs {
		out = append(out, idx.parseCity(off))
	}
	return out
}

// referenceNear is the scan as it was first written, over the parsed table:
// the US rows inside the fence kept, stably sorted by distance, cut to the
// limit.
func referenceNear(table []City, lat, lon, radiusMi float64, limit int) []City {
	type hit struct {
		c  City
		mi float64
	}
	var hits []hit
	for _, c := range table {
		if c.Country != "US" {
			continue
		}
		if mi := MilesBetween(lat, lon, c.Lat, c.Lon); mi <= radiusMi {
			hits = append(hits, hit{c, mi})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].mi < hits[b].mi })
	out := []City{}
	for n := 0; n < len(hits) && n < limit; n++ {
		out = append(out, hits[n].c)
	}
	return out
}
