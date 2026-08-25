package wfigs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// geojson is the shape live-probed on 2026-08-25: a named, sized incident
// 30 km from Oceanside; a dispatch-only record in the ring; one far away.
const geojson = `{"type":"FeatureCollection","features":[
{"type":"Feature","geometry":{"type":"Point","coordinates":[-117.05,33.45]},"properties":{"IncidentName":"Timber","FireDiscoveryDateTime":1786246500000,"PercentContained":26,"IncidentSize":12915,"POOState":"US-CA","IncidentTypeCategory":"WF"}},
{"type":"Feature","geometry":{"type":"Point","coordinates":[-117.30,33.30]},"properties":{"IncidentName":"LAC-301056","FireDiscoveryDateTime":1787552851000,"PercentContained":null,"IncidentSize":null,"DiscoveryAcres":3,"POOState":"US-CA","IncidentTypeCategory":"WF"}},
{"type":"Feature","geometry":{"type":"Point","coordinates":[-120.03,39.72]},"properties":{"IncidentName":"Bug","FireDiscoveryDateTime":1786228249000,"PercentContained":97,"IncidentSize":93733,"POOState":"US-CA","IncidentTypeCategory":"WF"}}
]}`

func TestFetchListsNearbyIncidentsLargestFirst(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/geo+json")
		_, _ = w.Write([]byte(geojson))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: -1})
	p := New(c, srv.URL+"/query", fire.DefaultRules())
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	if !strings.Contains(query, "IncidentTypeCategory") || !strings.Contains(query, "f=geojson") {
		t.Fatalf("one national query: %s", query)
	}
	fs := frag.PerLocation[snapshot.Key(oceanside)].Fire
	if fs == nil || len(fs.Incidents) != 2 {
		t.Fatalf("two incidents inside 50 km: %+v", fs)
	}
	timber := fs.Incidents[0]
	if timber.Name != "Timber" || *timber.Acres != 12915 || *timber.PercentContained != 26 || timber.State != "US-CA" || timber.Discovered.Year() != 2026 {
		t.Fatalf("the sized incident first, with its fields: %+v", timber)
	}
	if timber.Source.DistanceKm == nil || *timber.Source.DistanceKm < 20 || *timber.Source.DistanceKm > 40 {
		t.Fatalf("distance rides along: %+v", timber.Source)
	}
	if fs.Incidents[1].Name != "LAC-301056" || fs.Incidents[1].Acres == nil || *fs.Incidents[1].Acres != 3 { // no IncidentSize yet: DiscoveryAcres stands in
		t.Fatalf("dispatch-only records list last, unsized: %+v", fs.Incidents[1])
	}
}
