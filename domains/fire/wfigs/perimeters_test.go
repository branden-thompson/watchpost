package wfigs

// perimeters_test.go — 0.18.0 D-121: the map's fire - every active incident,
// and the perimeters of the box it is in, generalised to its resolution.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// serve answers every request with a body, recording the queries.
func serve(t *testing.T, body []byte, queries *[]url.Values) *Provider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*queries = append(*queries, r.URL.Query())
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	return New(c, srv.URL+"/query", fire.DefaultRules())
}

// TestPerimetersAreAskedForTheirBoxGeneralised is D-121: one query for the
// box, generalised to about a thousandth of its width - never the whole
// country's 14.6 MB - each perimeter with its name, acres and containment,
// its areas and their holes.
func TestPerimetersAreAskedForTheirBoxGeneralised(t *testing.T) {
	body, err := os.ReadFile("testdata/perimeters.json")
	if err != nil {
		t.Fatal(err)
	}
	var qs []url.Values
	ps, err := serve(t, body, &qs).Perimeters(context.Background(), -125, 42, -116, 49)
	if err != nil {
		t.Fatal(err)
	}
	q := qs[0]
	if q.Get("geometry") != "-125,42,-116,49" || q.Get("geometryType") != "esriGeometryEnvelope" || q.Get("maxAllowableOffset") == "" || q.Get("f") != "geojson" {
		t.Errorf("the perimeters asked %v; want the box, generalised, as GeoJSON", q)
	}
	if len(ps) != 4 {
		t.Fatalf("%d perimeters; want the fixture's 4", len(ps))
	}
	byName := map[string]Perimeter{}
	for _, p := range ps {
		byName[p.Name] = p
	}
	deep, miners := byName["Deep Canyon"], byName["Miners"]
	if len(deep.Areas) != 1 || len(deep.Areas[0]) != 2 || deep.Acres == nil || deep.Contained == nil || *deep.Contained != 100 {
		t.Errorf("Deep Canyon is %d areas (%v rings), %v acres, %v contained; want one area and its hole, 100%% contained", len(deep.Areas), len(deep.Areas[0]), deep.Acres, deep.Contained)
	}
	if len(miners.Areas) != 2 {
		t.Errorf("Miners is %d areas; want its two", len(miners.Areas))
	}
}

// TestTheProductionPerimetersAreTheInteragencyLayer: the perimeters' layer
// is the incidents' host's, its interagency perimeters - a new path on the
// closed list's host, no new host.
func TestTheProductionPerimetersAreTheInteragencyLayer(t *testing.T) {
	p := New(nil, "", fire.DefaultRules())
	if !strings.Contains(p.perimeters, "services3.arcgis.com") || !strings.Contains(p.perimeters, "WFIGS_Interagency_Perimeters_Current") {
		t.Errorf("the perimeters' layer is %s", p.perimeters)
	}
}

// TestTheMapReadsEveryIncident is D-121: the map is handed every active
// incident, however far from the station's places - the same query and the
// same memo the places' fire reads.
func TestTheMapReadsEveryIncident(t *testing.T) {
	var qs []url.Values
	ins, err := serve(t, []byte(geojson), &qs).Incidents(context.Background())
	if err != nil || len(ins) != 3 {
		t.Fatalf("%d incidents (%v); want all 3", len(ins), err)
	}
	if !strings.Contains(qs[0].Get("where"), "IncidentTypeCategory") {
		t.Errorf("the map asked %v; want the places' own query", qs[0])
	}
}

// TestABoxsPerimetersAreDecodedOnce is W14's P-7: a box's perimeters (up to
// ~378 KB) are served from the cache on every map ask, and were decoded
// again each time. The same box's same body is decoded once.
func TestABoxsPerimetersAreDecodedOnce(t *testing.T) {
	body, err := os.ReadFile("testdata/perimeters.json")
	if err != nil {
		t.Fatal(err)
	}
	var qs []url.Values
	p := serve(t, body, &qs)
	for range 3 {
		if ps, err := p.Perimeters(context.Background(), -125, 42, -116, 49); err != nil || len(ps) != 4 {
			t.Fatalf("%d perimeters, %v", len(ps), err)
		}
	}
	if n := p.PerimeterParses(); n != 1 {
		t.Errorf("three asks of one box decoded it %d times; want once", n)
	}
}

// TestABadPerimetersBodyIsNotServedAgain: a body that does not decode is
// forgotten by the cache, so the next ask fetches afresh rather than being
// handed the same bad bytes for the rest of their cache life.
func TestABadPerimetersBodyIsNotServedAgain(t *testing.T) {
	var qs []url.Values
	p := serve(t, []byte("<html>not GeoJSON</html>"), &qs)
	for range 2 {
		if _, err := p.Perimeters(context.Background(), -125, 42, -116, 49); err == nil {
			t.Fatal("a body that does not decode was read as perimeters")
		}
	}
	if len(qs) != 2 {
		t.Errorf("two asks after a bad body reached the host %d times; the bad body was served from the cache", len(qs))
	}
}
