package usgs

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/seismic"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestAQuakeKeepsWhereItHappened. The position arrives from the service, is
// used to work out how far away and which way, and was then thrown away - so a
// map could not draw the epicentre, and nothing could recover it from "134 km
// north-north-west of somewhere" (MG-5).
func TestAQuakeKeepsWhereItHappened(t *testing.T) {
	here := snapshot.LocationRef{Label: "Fort Wayne", Lat: 41.08, Lon: -85.14}
	at := time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)
	// A quake off to the west, well inside any sensible rule.
	f := feature{mag: 5.2, magType: "mww", place: "somewhere west", depthKm: 10, at: at,
		lat: 41.50, lon: -88.00, typ: "earthquake", sig: 400}

	// A rule wide enough that nothing is filtered: this test is about the
	// position surviving, not about the graduated radius.
	p := &Provider{rules: seismic.Rules{
		Bands: []seismic.Band{{UpperMag: 10, RadiusMi: 5000}},
		Types: []string{"earthquake"},
	}}
	st := p.stateFor(here, []feature{f}, at)
	if st == nil || len(st.Quakes) != 1 {
		t.Fatalf("the quake was not kept: %+v", st)
	}
	q := st.Quakes[0]
	if q.Lat != f.lat || q.Lon != f.lon {
		t.Errorf("the epicentre is %v,%v; the service said %v,%v", q.Lat, q.Lon, f.lat, f.lon)
	}
	// The figures already relied on must not move: they are computed from the
	// same two numbers.
	if want := geo.HaversineKM(here.Lat, here.Lon, f.lat, f.lon); q.DistanceKm != want {
		t.Errorf("distance is %v; from the kept position it is %v", q.DistanceKm, want)
	}
	if q.Bearing == "" {
		t.Error("the compass word is gone")
	}
}
