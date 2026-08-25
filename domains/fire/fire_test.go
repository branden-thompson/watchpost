package fire

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func f64(v float64) *float64 { return &v }

func TestRulesKeepAndBold(t *testing.T) {
	r := DefaultRules()
	if err := r.Valid(); err != nil {
		t.Fatal(err)
	}
	if r.Keep("low", f64(20)) {
		t.Fatal("low confidence is below nominal")
	}
	if !r.Keep("nominal", f64(5)) || !r.Keep("high", f64(100)) || !r.Keep("analyst", nil) {
		t.Fatal("nominal+, FRP ≥ 5 or unknown FRP pass")
	}
	if r.Keep("high", f64(2)) {
		t.Fatal("a 2 MW detection is below the floor")
	}
	if !r.Bold(snapshot.Hotspot{FRPMW: f64(60)}) || r.Bold(snapshot.Hotspot{FRPMW: f64(10)}) || r.Bold(snapshot.Hotspot{}) {
		t.Fatal("bold at ≥ 50 MW only")
	}
	if (Rules{RadiusKm: 0}).Valid() == nil || (Rules{RadiusKm: 1, IncidentRadiusKm: 1, MinConfidence: "maybe"}).Valid() == nil {
		t.Fatal("bad rules are refused")
	}
}

func TestNearAndBounds(t *testing.T) {
	oceanside := snapshot.LocationRef{Lat: 33.24, Lon: -117.29}
	if km, ok := Near(oceanside, 33.16, -117.35, 25); !ok || km < 8 || km > 12 {
		t.Fatalf("Carlsbad is ~10 km away: %.1f %v", km, ok)
	}
	if _, ok := Near(oceanside, 34.05, -118.24, 25); ok {
		t.Fatal("Los Angeles is outside 25 km")
	}
	w, s, e, n := Bounds(33.24, -117.29, 25)
	if !(w < -117.29 && e > -117.29 && s < 33.24 && n > 33.24) || (n-s) < 0.4 || (n-s) > 0.5 {
		t.Fatalf("25 km ≈ 0.45° of latitude: %v %v %v %v", w, s, e, n)
	}
}

func TestClusterMergesPassesAndSortsNearest(t *testing.T) {
	day := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	hs := []snapshot.Hotspot{
		{Lat: 33.500, Lon: -117.200, DetectedAt: day, FRPMW: f64(12), DistanceKm: f64(30)},
		{Lat: 33.501, Lon: -117.201, DetectedAt: day.Add(time.Hour), FRPMW: f64(40), DistanceKm: f64(30)},       // same fire, later pass, stronger
		{Lat: 33.300, Lon: -117.300, DetectedAt: day, FRPMW: f64(8), DistanceKm: f64(5)},                        // a different, nearer fire
		{Lat: 33.500, Lon: -117.200, DetectedAt: day.Add(-30 * time.Hour), FRPMW: f64(90), DistanceKm: f64(30)}, // yesterday: its own cluster
	}
	got := Cluster(hs)
	if len(got) != 3 {
		t.Fatalf("two passes of one fire merge, yesterday stays apart: %d", len(got))
	}
	if *got[0].DistanceKm != 5 {
		t.Fatalf("nearest first: %+v", got[0])
	}
	found := false
	for _, h := range got {
		if h.Lat == 33.501 && *h.FRPMW == 40 {
			found = true
		}
	}
	if !found {
		t.Fatalf("the strongest pass represents the cluster: %+v", got)
	}
}

func TestAnalystPointsOutrankEveryThreshold(t *testing.T) {
	r := DefaultRules()
	r.MinConfidence = "high"
	if r.Keep("nominal", f64(50)) || !r.Keep("high", f64(50)) || !r.Keep("analyst", f64(50)) {
		t.Fatal("at high: nominal drops, high and analyst-curated (HMS) pass")
	}
}
