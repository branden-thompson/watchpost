package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: B3 UAT 29 — coastal-waters fields from the raw gridpoint (live-
// probed shape: primarySwell*, waveHeight, wavePeriod, windWaveHeight);
// inland grids (no marine series) yield no Marine section.

func TestMarineProviderLiftsGridpointSeries(t *testing.T) {
	srv, _ := testServer(t)
	client, _ := httpx.New(httpx.Config{UserAgent: "test", MaxRetries: 0})
	base := New(client, srv.URL)
	m := NewMarine(base)
	if m.ID() != "nws-marine" || m.Domains()[0] != "marine" {
		t.Fatal("marine provider identity")
	}
	ref := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	frag, err := m.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{ref}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v / %v", err, frag.Err)
	}
	mar := frag.PerLocation[snapshot.Key(ref)].Marine
	if mar == nil {
		t.Fatal("coastal grid must yield a Marine section")
	}
	if *mar.SwellHeight != 0.6096 || *mar.SwellDirDeg != 280 || *mar.WavePeriod != 6 || *mar.WindWaveHeight != 0.06096 {
		t.Fatalf("gridpoint series not lifted: %+v", mar)
	}
	if mar.SecondarySwellHeight == nil || *mar.SecondarySwellDirDeg != 210 || *mar.SecondaryPeriod != 14 {
		t.Fatalf("secondary swell must lift (UAT 32.3): %+v", mar)
	}
	if mar.WaterTemp != nil {
		t.Fatal("NWS gridpoint carries no water temperature — that is the buoy's")
	}
	if mar.Source.Provider != "nws-marine" {
		t.Fatalf("provenance: %+v", mar.Source)
	}
}

func TestMarineProviderRefusesOtherKinds(t *testing.T) {
	m := NewMarine(New(nil, "http://x"))
	if _, err := m.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs}); err == nil {
		t.Fatal("must refuse non-marine kinds")
	}
}

func TestMarineProviderTreatsAllZeroGridAsInland(t *testing.T) {
	// Live finding: a cell one step off the coast publishes the marine series
	// as all zeros - that is land, not water.
	if positive(nil) || positive(f0()) || !positive(fpos()) {
		t.Fatal("positive() must require a present, non-zero value")
	}
}

func f0() *float64   { v := 0.0; return &v }
func fpos() *float64 { v := 0.6; return &v }

func TestInlandGridIsRememberedNotRedownloaded(t *testing.T) {
	// UAT 72: a grid whose gridpoint carries no marine series is inland for
	// a day — the 228 KB product is fetched once, not every marine cycle.
	var gridCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/"):
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(p, "/stations"):
			_, _ = w.Write(fixture(t, "stations.json"))
		case strings.HasPrefix(p, "/gridpoints/"):
			gridCalls.Add(1)
			_, _ = w.Write([]byte(`{"properties":{"waveHeight":{"uom":"wmoUnit:m","values":[{"validTime":"2026-08-24T03:00:00+00:00/PT3H","value":0}]}}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	m := NewMarine(newProvider(t, srv.URL))
	for range 3 {
		frag, err := m.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{oceanside}})
		if err != nil || frag.Err != nil {
			t.Fatal(err, frag.Err)
		}
		if _, ok := frag.PerLocation[snapshot.Key(oceanside)]; ok {
			t.Fatal("inland grid must publish no marine block")
		}
	}
	if gridCalls.Load() != 1 {
		t.Fatalf("inland grid must be downloaded once, got %d", gridCalls.Load())
	}
}
