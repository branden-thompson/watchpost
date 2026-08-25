package coops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: B3 UAT 61 — NOAA CO-OPS tides & currents. Fixtures recorded live on
// 2026-08-24 (metric, GMT): La Jolla hilo predictions, San Diego water
// level, San Diego Bay Entrance currents, and the HTTP-200 error envelope.

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func server(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(r.URL.Path, "/stations.json"):
			_, _ = w.Write(fixture(t, "stations_"+q.Get("type")+".json"))
		case q.Get("product") == "predictions":
			if q.Get("units") != "metric" || q.Get("time_zone") != "gmt" || q.Get("interval") != "hilo" || q.Get("datum") != "MLLW" {
				t.Errorf("predictions query: %s", r.URL.RawQuery)
			}
			if q.Get("station") == "9999901" {
				_, _ = w.Write([]byte(`{"error":{"message":"No Predictions data was found. Please make sure the Datum input is valid."}}`))
				return
			}
			_, _ = w.Write(fixture(t, "predictions.json"))
		case q.Get("product") == "water_level":
			if q.Get("station") == "9410230" {
				_, _ = w.Write(fixture(t, "error.json")) // gauge without MLLW: HTTP 200 error envelope
				return
			}
			_, _ = w.Write(fixture(t, "water_level.json"))
		case q.Get("product") == "currents_predictions":
			if q.Get("station") == "PCT0076" {
				t.Error("type-W current stations publish no predictions and must never be requested")
			}
			_, _ = w.Write(fixture(t, "currents.json"))
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func newProvider(t *testing.T, base string) *Provider {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000, MaxRetries: -1})
	if err != nil {
		t.Fatal(err)
	}
	p := New(c, base)
	p.Now = func() time.Time { return time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC) }
	return p
}

var (
	sanDiego  = snapshot.LocationRef{Label: "San Diego, CA", Zip: "92101", Lat: 32.7157, Lon: -117.1611}
	oceanside = snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	phoenix   = snapshot.LocationRef{Label: "Phoenix, AZ", Zip: "85001", Lat: 33.45, Lon: -112.07}
)

func TestTidesLevelAndCurrentsByProximity(t *testing.T) {
	srv, requests := server(t)
	p := newProvider(t, srv.URL)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{sanDiego, oceanside, phoenix}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v / %v", err, frag.Err)
	}
	sd := frag.PerLocation[snapshot.Key(sanDiego)].Marine
	if sd == nil || len(sd.Tides) != 8 || sd.Tides[0].Type != "H" || sd.Tides[0].Height != 1.64 {
		t.Fatalf("San Diego tides: %+v", sd)
	}
	if sd.Tides[0].Time != time.Date(2026, 8, 24, 2, 2, 0, 0, time.UTC) {
		t.Fatalf("GMT stamps parse as UTC: %v", sd.Tides[0].Time)
	}
	if sd.TideLevel != nil || sd.TideStation == "" || sd.TideStationKM == nil || *sd.TideStationKM > 5 {
		t.Fatalf("San Diego station (the level belongs to the obs provider, UAT 72): %+v", sd)
	}
	obs, err := NewObs(p).Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarineObs, Locations: []snapshot.LocationRef{sanDiego, oceanside, phoenix}})
	if err != nil || obs.Err != nil {
		t.Fatalf("obs fetch: %v / %v", err, obs.Err)
	}
	if lvl := obs.PerLocation[snapshot.Key(sanDiego)].Marine; lvl == nil || lvl.TideLevel == nil || *lvl.TideLevel != 1.131 || lvl.Source.Provider != "coops-obs" {
		t.Fatalf("San Diego observed level: %+v", lvl)
	}
	if _, ok := obs.PerLocation[snapshot.Key(oceanside)]; ok {
		t.Fatal("Oceanside's nearest gauge answered an error envelope: no level block")
	}
	if len(sd.Currents) == 0 || !strings.HasPrefix(sd.CurrentStation, "G St. Pier") { // nearest predictable (non-W) current station to 92101; B St. Pier (W) is skipped
		t.Fatalf("San Diego currents: %+v", sd)
	}
	if c := sd.Currents[0]; c.Type != "flood" || c.Speed < 0.52 || c.Speed > 0.54 {
		t.Fatalf("currents convert cm/s to m/s magnitude: %+v", c)
	}
	if ebb := sd.Currents[2]; ebb.Type != "ebb" || ebb.Speed < 0 {
		t.Fatalf("ebb speed is a magnitude: %+v", ebb)
	}
	oc := frag.PerLocation[snapshot.Key(oceanside)].Marine
	if oc == nil || len(oc.Tides) != 8 || oc.TideStation != "San Clemente" {
		t.Fatalf("Oceanside skips the datum-less pier (error envelope) for the next station, San Clemente: %+v", oc)
	}
	if oc.TideLevel != nil {
		t.Fatal("predictions never carry a level (UAT 72: the obs provider owns it)")
	}
	if len(oc.Currents) != 0 {
		t.Fatal("no current station within 40 km of Oceanside")
	}
	if _, ok := frag.PerLocation[snapshot.Key(phoenix)]; ok {
		t.Fatal("inland location must get no tide block")
	}
	// Station products live in the client cache: a second cycle re-hits
	// nothing but is still served (predictions for an hour, level for ten).
	before := requests.Load()
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{sanDiego, oceanside}}); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != before {
		t.Fatalf("second cycle must be served from the client cache (%d -> %d requests)", before, got)
	}
	if p.ID() != "coops" || p.Domains()[0] != "marine" {
		t.Fatal("provider identity")
	}
}

func TestErrorEnvelopeIsAnError(t *testing.T) {
	var doc struct{ apiError }
	if err := decode(t, fixture(t, "error.json"), &doc); err != nil {
		t.Fatal(err)
	}
	if err := doc.err(); err == nil || !strings.Contains(err.Error(), "no MLLW") {
		t.Fatalf("HTTP-200 error envelope must surface as an error, got %v", err)
	}
}
