package wfigs

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

const layerBody = `{"type":"FeatureCollection","features":[
{"type":"Feature","geometry":{"type":"Point","coordinates":[-117.30,33.25]},"properties":{"IncidentName":"Bonsall","FireDiscoveryDateTime":1756000000000,"PercentContained":40,"IncidentSize":120,"POOState":"US-CA","IncidentTypeCategory":"WF"}},
{"type":"Feature","geometry":{"type":"Point","coordinates":[-117.31,33.26]},"properties":{"IncidentName":"","IncidentTypeCategory":"WF"}},
{"type":"Feature","geometry":{"type":"Point","coordinates":[-121.5,49.9]},"properties":{"IncidentName":"Far","IncidentTypeCategory":"WF"}}]}`

// Quality pass Q3 (L4-F6): the layer is decoded once per body change,
// whoever asks; the memo's size is a gauge.
func TestLayerIsDecodedOncePerBodyChange(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/geo+json")
		_, _ = w.Write([]byte(layerBody))
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "t"})
	if err != nil {
		t.Fatal(err)
	}
	p := New(c, srv.URL, fire.Rules{RadiusKm: 50, IncidentRadiusKm: 80, MinConfidence: "nominal"})
	oceanside := snapshot.LocationRef{Label: "Oceanside", Lat: 33.1959, Lon: -117.3795}
	first, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || first.Err != nil {
		t.Fatalf("fetch: %v %v", err, first.Err)
	}
	second, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if p.memo.Parses() != 1 || hits.Load() != 1 {
		t.Fatalf("one decode and one request for two fetches of the same body: parses=%d requests=%d", p.memo.Parses(), hits.Load())
	}
	if p.MemoIncidents() != 2 {
		t.Fatalf("the memo holds the named incidents with a point (2 of 3): %d", p.MemoIncidents())
	}
	a, b := first.PerLocation[snapshot.Key(oceanside)].Fire, second.PerLocation[snapshot.Key(oceanside)].Fire
	if len(a.Incidents) != 1 || len(b.Incidents) != 1 || a.Incidents[0].Name != "Bonsall" || b.Incidents[0].Name != "Bonsall" {
		t.Fatalf("both fetches answer from the one decode: %+v / %+v", a.Incidents, b.Incidents)
	}
}

// Quality pass Q3 (CQ-12): decodeLayer reads the cache's own slice.
func TestGetTextCallersMustNotMutate(t *testing.T) {
	raw := []byte(layerBody)
	before := sha256.Sum256(raw)
	if _, err := decodeLayer(raw); err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(raw) != before {
		t.Fatal("decodeLayer wrote into the body it was handed (httpx.GetText contract)")
	}
}
