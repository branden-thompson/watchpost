package seismic

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The ratified step function, boundary by boundary (objectives §4).
func TestRadiusMiForEveryBand(t *testing.T) {
	r := DefaultRules()
	rows := []struct {
		mag float64
		mi  float64
	}{
		{0.4, 3}, {0.99, 3},
		{1.0, 10}, {2.49, 10},
		{2.5, 20}, {3.49, 20},
		{3.5, 40}, {3.99, 40},
		{4.0, 100}, {4.49, 100},
		{4.5, 150}, {4.99, 150},
		{5.0, 400}, {5.99, 400},
		{6.0, 500}, {6.99, 500},
		{7.0, 1000}, {9.1, 1000},
	}
	for _, row := range rows {
		if got := r.RadiusMiFor(row.mag); got != row.mi {
			t.Errorf("RadiusMiFor(%.2f) = %.0f mi, want %.0f", row.mag, got, row.mi)
		}
	}
}

// Keep enforces the graduated rule at the objectives' own examples.
func TestKeepGraduatedRule(t *testing.T) {
	r := DefaultRules()
	mi := func(m float64) float64 { return m * mileKm }
	// M2.6 → 20 mi band: hidden at 200 km, shown at 40 km... but 40 km is 25 mi > 20, so hidden.
	if r.Keep(2.6, mi(21)) {
		t.Fatal("M2.6 at 21 mi is beyond its 20 mi band")
	}
	if !r.Keep(2.6, mi(19)) {
		t.Fatal("M2.6 at 19 mi is inside its 20 mi band")
	}
	// M4.2 → 100 mi: shown at 88 mi, hidden at 120 mi.
	if !r.Keep(4.2, mi(88)) || r.Keep(4.2, mi(120)) {
		t.Fatal("M4.2 band is 100 mi")
	}
	// M6 → 500 mi: shown at 480 mi, hidden at 520.
	if !r.Keep(6.0, mi(480)) || r.Keep(6.0, mi(520)) {
		t.Fatal("M6.0 band is 500 mi")
	}
	// Exactly on the band edge is inside (≤).
	if !r.Keep(3.4, mi(20)) {
		t.Fatal("distance exactly at the band radius is inside")
	}
}

func TestMaxRadiusAndValidity(t *testing.T) {
	r := DefaultRules()
	if err := r.Valid(); err != nil {
		t.Fatal(err)
	}
	if km := r.MaxRadiusKm(); km < 1000*mileKm-1 || km > 1000*mileKm+1 {
		t.Fatalf("widest reach is 1000 mi in km: %.1f", km)
	}
	// A non-ascending or empty ruleset is refused.
	if (Rules{Bands: []Band{{2, 10}, {1, 20}}, LookbackDays: 7}).Valid() == nil {
		t.Fatal("non-ascending bands must be refused")
	}
	if (Rules{LookbackDays: 7}).Valid() == nil {
		t.Fatal("no bands must be refused")
	}
}

func TestSortLargestThenNearest(t *testing.T) {
	qs := []snapshot.Quake{
		{Mag: 2.1, DistanceKm: 5},
		{Mag: 4.5, DistanceKm: 80},
		{Mag: 2.1, DistanceKm: 2},
		{Mag: 4.5, DistanceKm: 10},
	}
	Sort(qs)
	got := [][2]float64{}
	for _, q := range qs {
		got = append(got, [2]float64{q.Mag, q.DistanceKm})
	}
	want := [][2]float64{{4.5, 10}, {4.5, 80}, {2.1, 2}, {2.1, 5}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sort: largest magnitude first, then nearest — got %v, want %v", got, want)
		}
	}
}

// The plan is concentric: a near-field query and one regional query, and it
// covers every band (nothing the rule would show can fall outside both).
func TestQueryPlanIsConcentric(t *testing.T) {
	r := DefaultRules()
	plan := r.QueryPlan()
	if len(plan) != 2 {
		t.Fatalf("default bands ⇒ near + regional = 2 queries, got %d: %+v", len(plan), plan)
	}
	near, reg := plan[0], plan[1]
	// The near query has no effective magnitude floor (FloorMag is below any
	// real quake) so a Keep-visible negative-magnitude event underfoot is not
	// dropped (REVIEW P5 F2); its window is [FloorMag, 3.5) within 20 mi.
	if near.MinMag != r.FloorMag() || near.MinMag >= 0 || near.MaxMag != 3.5 || near.RadiusMi != 20 {
		t.Fatalf("near query is [FloorMag,3.5) within 20 mi, got %+v", near)
	}
	if reg.MinMag != 3.5 || reg.MaxMag != 0 || reg.RadiusMi != 1000 {
		t.Fatalf("regional query is [3.5,∞) within 1000 mi, got %+v", reg)
	}
	// The pivot magnitude is shared: near's ceiling == regional's floor, so no
	// band falls in a gap between the two queries.
	if near.MaxMag != reg.MinMag {
		t.Fatalf("near ceiling %.1f must equal regional floor %.1f (no gap)", near.MaxMag, reg.MinMag)
	}
}

// The equivalence property: the concentric plan, unioned and Keep-filtered,
// returns exactly the quakes a single wide magnitude-0 query would — for
// every band's magnitude, at distances just inside and just outside its
// radius. This is the correctness guarantee that lets P2 fetch cheaply.
func TestQueryPlanEquivalenceWithBruteForce(t *testing.T) {
	r := DefaultRules()
	plan := r.QueryPlan()
	// A quake at magnitude m, distance d (mi) is *planned-visible* if some
	// plan query would fetch it (mag in window, within radius) AND Keep passes.
	plannedVisible := func(m, dMi float64) bool {
		for _, q := range plan {
			if m < q.MinMag || (q.MaxMag != 0 && m >= q.MaxMag) {
				continue
			}
			if dMi <= q.RadiusMi && r.Keep(m, dMi*mileKm) {
				return true
			}
		}
		return false
	}
	for _, m := range []float64{0.5, 1.0, 2.4, 2.5, 3.4, 3.5, 3.9, 4.0, 4.4, 4.9, 5.5, 6.5, 7.5} {
		for _, dMi := range []float64{1, 3, 9, 10, 19, 20, 39, 40, 99, 100, 149, 150, 399, 400, 499, 500, 999, 1000, 1200} {
			brute := r.Keep(m, dMi*mileKm) // the single wide-query truth
			if plannedVisible(m, dMi) != brute {
				t.Fatalf("equivalence broken at M%.1f %.0f mi: plan=%v brute=%v", m, dMi, plannedVisible(m, dMi), brute)
			}
		}
	}
}

func TestTypesFilterIsConfigurable(t *testing.T) {
	r := DefaultRules()
	if !r.Wants("earthquake") || r.Wants("quarry blast") {
		t.Fatal("default shows earthquakes only")
	}
	r.Types = []string{"earthquake", "explosion"}
	if !r.Wants("explosion") {
		t.Fatal("types is a config-driven allowlist (D4)")
	}
}
