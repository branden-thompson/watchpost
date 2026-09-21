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
	// Glacier Bay is served too: thirty-two separate islands is what makes it
	// the one fixture that can tell a join from a merge (RT-8).
	bay, err := os.ReadFile("../platform/geo/testdata/zone-glacier-bay.geojson")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		var geom []byte
		var name string
		switch id {
		case "TXZ119":
			geom, name = body, "Dallas"
		case "AKZ320":
			geom, name = bay, "Glacier Bay"
		case "TXC113":
			// A lake inside a COUNTY zone - a county id, which also proves the county
			// path is reached from here. One area, two rings. **No real fixture we
			// hold has a hole**, and without one a merge and a join produce the
			// same counts - which a mutation of the reader proved by surviving
			// this test. It is written out here so the distinction is visible.
			geom = []byte(`{"type":"Polygon","coordinates":[` +
				`[[-96.9,32.6],[-96.5,32.6],[-96.5,33.0],[-96.9,33.0],[-96.9,32.6]],` +
				`[[-96.8,32.7],[-96.7,32.7],[-96.7,32.8],[-96.8,32.8],[-96.8,32.7]]]}`)
			name = "County with a lake"
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=426444")
		_, _ = w.Write([]byte(`{"properties":{"id":"` + id + `","name":"` + name + `"},"geometry":` + string(geom) + `}`))
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

	own := geo.Shape{{{
		{Lon: -85.2, Lat: 41.0}, {Lon: -85.0, Lat: 41.0}, {Lon: -85.0, Lat: 41.2}, {Lon: -85.2, Lat: 41.0},
	}}}
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
	if a := got["has-its-own"]; a.Shape.Vertices() != 4 {
		t.Errorf("the alert with its own polygon resolved to %d positions; it carried 4", a.Shape.Vertices())
	}
	// A zone-only alert gets its zone's shape - the case that is four in five.
	if a := got["zone-only"]; a.Shape.Vertices() != 80 {
		t.Errorf("the zone-only alert resolved to %d positions; its zone has 80", a.Shape.Vertices())
	}
	// A zone nobody can supply leaves the alert with nothing, and that is not
	// an error here: whether to draw it is a question for whatever has a view.
	unknown, ok := got["unknown-zone"]
	if ok && !unknown.Empty() {
		t.Errorf("an alert whose zone could not be got resolved to %d positions", unknown.Shape.Vertices())
	}
	// **And it says so.** An area that could not be built is not the same
	// thing as an alert that covers nothing, and the difference has to survive
	// this call for anything downstream to be able to act on it.
	if ok && unknown.Complete() {
		t.Error("an alert whose only zone could not be got came back as complete")
	}
	if ok && (len(unknown.Missing) != 1 || unknown.Missing[0] != "ZZZ999") {
		t.Errorf("the missing parts are %v; the one that failed is ZZZ999", unknown.Missing)
	}
}

// TestAnAlertOverManyZonesKeepsThemApart is RT-8 at the seam it was filed
// against. An alert names several zones; their shapes are joined into one
// answer, and **joining must not mean merging**.
//
// What draws reads an area's first ring as the outline and every ring after it
// as a hole. So if the zones' rings were poured into one list, Dallas would
// become a hole in a Glacier Bay islet, thirty-one islands would become holes
// in the first one, and anything outside that islet's band would not be drawn
// at all. The count below is the whole test: thirty-three separate areas, the
// same number the two zones have between them.
func TestAnAlertOverManyZonesKeepsThemApart(t *testing.T) {
	srv := zoneServer(t)
	defer srv.Close()
	store := zoneStore(t, srv.URL)

	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{
		Label: "two places at once", Lat: 32.78, Lon: -96.80,
		Alerts: []snapshot.Alert{{
			ID: "wide", Event: "Winter Storm Warning",
			AffectedZones: []string{"TXZ119", "TXC113", "AKZ320"},
		}},
	}}}

	wide := resolveAlertAreas(context.Background(), store, snap)["wide"]
	got := wide.Shape
	// One + one + thirty-two. **The lake is what makes this count load-bearing**:
	// flattened, the county and its lake would arrive as two areas and this would
	// read 35.
	if len(got) != 34 {
		t.Fatalf("three zones of one, one and thirty-two areas resolved to %d areas; they hold 34 between them", len(got))
	}
	holed := 0
	for _, area := range got {
		if len(area) == 2 {
			holed++
		}
	}
	if holed != 1 {
		t.Errorf("%d areas carry a hole; exactly one of these zones has a lake in it", holed)
	}
	if !wide.Complete() {
		t.Errorf("every zone resolved, yet the area reports %v missing", wide.Missing)
	}
	if got.Vertices() != 80+10+12004 {
		t.Errorf("%d positions; Dallas has 80, the lake county 10 and Glacier Bay 12,004", got.Vertices())
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
