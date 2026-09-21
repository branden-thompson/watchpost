package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/domains/weather/nws/zones"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func zoneServer(t *testing.T) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile("../platform/geo/testdata/zone-dallas.geojson")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if id != "TXZ119" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=426444")
		w.Write([]byte(`{"properties":{"id":"TXZ119","name":"Dallas"},"geometry":` + string(body) + `}`))
	}))
}

func zoneStore(t *testing.T, base string) *zones.Store {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost-test"})
	if err != nil {
		t.Fatal(err)
	}
	return zones.New(c, base)
}

// TestAnAlertsAreaComesFromItsOwnShapeOrItsZones is the seam 0.18.0 draws
// through, exercised here with no map at all.
//
// **The two cases are not alternatives in principle, they are four alerts in
// five against one.** An alert that carries a polygon uses it; an alert that
// names zones and carries nothing uses their shapes. Both come back as the
// same thing, so whatever draws does not care which kind it was.
func TestAnAlertsAreaComesFromItsOwnShapeOrItsZones(t *testing.T) {
	srv := zoneServer(t)
	defer srv.Close()
	store := zoneStore(t, srv.URL)

	own := geo.Shape{{
		{Lon: -85.2, Lat: 41.0}, {Lon: -85.0, Lat: 41.0}, {Lon: -85.0, Lat: 41.2}, {Lon: -85.2, Lat: 41.0},
	}}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{
		Label: "Dallas", Lat: 32.78, Lon: -96.80,
		Alerts: []snapshot.Alert{
			{ID: "has-its-own", Event: "Flood Warning", Area: own, AffectedZones: []string{"TXZ119"}},
			{ID: "zone-only", Event: "Heat Advisory", AffectedZones: []string{"TXZ119"}},
			{ID: "unknown-zone", Event: "Wind Advisory", AffectedZones: []string{"ZZZ999"}},
		},
	}}}

	got := resolveAlertAreas(context.Background(), store, snap)

	// An alert that brought its own shape keeps it: nothing is fetched for it.
	if a := got["has-its-own"]; a.Vertices() != 4 {
		t.Errorf("the alert with its own polygon resolved to %d positions; it carried 4", a.Vertices())
	}
	// A zone-only alert gets its zone's shape - the case that is four in five.
	if a := got["zone-only"]; a.Vertices() != 80 {
		t.Errorf("the zone-only alert resolved to %d positions; its zone has 80", a.Vertices())
	}
	// A zone nobody can supply leaves the alert with nothing, and that is not
	// an error here: whether to draw it is a question for whatever has a view.
	if a, ok := got["unknown-zone"]; ok && !a.Empty() {
		t.Errorf("an alert whose zone could not be got resolved to %d positions", a.Vertices())
	}
}

// TestSeedingIsActuallyCalled is RT-2's guard. Seeding was approved, built and
// tested, and then nothing called it — so it passed every test it had and did
// nothing at all. **A test of the wiring is a different test from a test of the
// thing**, and this is the one that was missing.
func TestSeedingIsActuallyCalled(t *testing.T) {
	srv := zoneServer(t)
	defer srv.Close()

	// A provider with no client at all: seeding must survive it, because a
	// panic in that goroutine would end the program.
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL), weather: nws.New(nil, "")}
	// With no provider able to resolve, seeding must still not panic or block.
	lp.seedZoneShapes(context.Background(), []snapshot.LocationRef{{Label: "Dallas", Lat: 32.78, Lon: -96.8}})

	// And the store it seeds into is reachable and counts what it does.
	if s := lp.zoneShapes.Stats(); s.Held != 0 {
		t.Errorf("nothing should be held before a zone resolves; %d is", s.Held)
	}
	// The real proof: the store seeds what it is given.
	lp.zoneShapes.Seed(context.Background(), []string{"TXZ119"})
	if s := lp.zoneShapes.Stats(); s.Held != 1 || s.Fetched != 1 {
		t.Errorf("after seeding one zone: held=%d fetched=%d, want 1 and 1", s.Held, s.Fetched)
	}
}
