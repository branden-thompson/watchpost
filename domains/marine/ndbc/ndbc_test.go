package ndbc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: B3 UAT 29 — free NDBC buoy observations: nearest reporting station
// within MaxDistanceKM, realtime2 parse (MM = missing), inland locations
// with no buoy in range get nothing.

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
		switch {
		case r.URL.Path == "/activestations.xml":
			_, _ = w.Write(fixture(t, "activestations.xml"))
		case strings.HasSuffix(r.URL.Path, "/46224_5day.txt"):
			_, _ = w.Write(fixture(t, "46224.txt"))
		case strings.HasSuffix(r.URL.Path, "/LJPC1_5day.txt"):
			w.WriteHeader(404) // listed as active, publishes no standard-met product
		case strings.HasSuffix(r.URL.Path, "/OCPC1_5day.txt"):
			_, _ = w.Write([]byte(tideGauge)) // listed as "ocpc1": the product file is upper-case
		case strings.HasSuffix(r.URL.Path, "/46999_5day.txt"):
			_, _ = w.Write([]byte(wavesOnly))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func TestParseRealtimeLatestRow(t *testing.T) {
	m, err := ParseRealtime(fixture(t, "46224.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if *m.WaveHeight != 0.9 || *m.WavePeriod != 14 || *m.SwellDirDeg != 187 || *m.WaterTemp != 23.3 {
		t.Fatalf("latest row: %+v", m)
	}
	if m.ObservedAt.Year() != 2026 || m.ObservedAt.Hour() != 17 || m.ObservedAt.Minute() != 26 {
		t.Fatalf("observed-at: %v", m.ObservedAt)
	}
	if m.SwellHeight != nil {
		t.Fatal("realtime product carries no swell height column")
	}
}

func TestNearestBuoyWithinRadiusFeedsCoastalOnly(t *testing.T) {
	srv, _ := server(t)
	client, _ := httpx.New(httpx.Config{UserAgent: "test", MaxRetries: -1, RatePerSec: 1000})
	p := New(client, srv.URL)
	coastal := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	inland := snapshot.LocationRef{Label: "Phoenix, AZ", Zip: "85001", Lat: 33.45, Lon: -112.07}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarineObs, Locations: []snapshot.LocationRef{coastal, inland}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v / %v", err, frag.Err)
	}
	mar := frag.PerLocation[snapshot.Key(coastal)].Marine
	if mar == nil || mar.Buoy != "46999" || *mar.WaveHeight != 1.2 {
		t.Fatalf("coastal location must read the nearest wave-reporting buoy: %+v", mar)
	}
	if *mar.BuoyDistanceKM > 30 {
		t.Fatalf("the nearshore buoy is ~5 km out, got %.1f km", *mar.BuoyDistanceKM)
	}
	if _, ok := frag.PerLocation[snapshot.Key(inland)]; ok {
		t.Fatal("inland location (no buoy within range) must get no marine data")
	}
	if p.ID() != "ndbc" || p.Domains()[0] != "marine-obs" {
		t.Fatal("provider identity")
	}
}

// tideGauge is a realtime2 product from a NOS tide station: wind, pressure
// and water temperature — no waves.
const tideGauge = `#YY  MM DD hh mm WDIR WSPD GST  WVHT   DPD   APD MWD   PRES  ATMP  WTMP  DEWP  VIS PTDY  TIDE
#yr  mo dy hr mn degT m/s  m/s     m   sec   sec degT   hPa  degC  degC  degC  nmi  hPa    ft
2026 08 24 17 30 270  4.1  5.2    MM    MM    MM  MM 1013.2  22.1  21.0    MM   MM   MM    MM
`

// wavesOnly is a nearshore wave buoy without a temperature sensor.
const wavesOnly = `#YY  MM DD hh mm WDIR WSPD GST  WVHT   DPD   APD MWD   PRES  ATMP  WTMP  DEWP  VIS PTDY  TIDE
#yr  mo dy hr mn degT m/s  m/s     m   sec   sec degT   hPa  degC  degC  degC  nmi  hPa    ft
2026 08 24 17 26  MM   MM   MM   1.2    12   8.0 200     MM    MM    MM    MM   MM   MM    MM
`

func TestFallsThroughToNearestReportingBuoy(t *testing.T) {
	// UAT 59 (San Francisco / downtown San Diego / Seattle / Miami): the
	// nearest "active" station is often a pier or land site with no wave
	// product (404) or wind only. The chain walks outward to the nearest
	// buoy that actually reports sea state; a 404 never fails the batch.
	srv, requests := server(t)
	client, _ := httpx.New(httpx.Config{UserAgent: "test", MaxRetries: -1, RatePerSec: 1000})
	p := New(client, srv.URL)
	coastal := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarineObs, Locations: []snapshot.LocationRef{coastal}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v / %v", err, frag.Err)
	}
	mar := frag.PerLocation[snapshot.Key(coastal)].Marine
	if mar == nil || mar.Buoy != "46999" || mar.WaveHeight == nil || *mar.WaveHeight != 1.2 {
		t.Fatalf("waves: skip LJPC1 (404) for the nearest reporting buoy 46999: %+v", mar)
	}
	if mar.WaterTemp == nil || *mar.WaterTemp != 21.0 {
		t.Fatalf("water temperature must fill from the nearest station reporting one (OCPC1 tide gauge): %+v", mar)
	}
	// A second location sharing the coast reuses the cached products.
	nearby := snapshot.LocationRef{Label: "Carlsbad, CA", Zip: "92008", Lat: 33.16, Lon: -117.33}
	before := requests.Load()
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarineObs, Locations: []snapshot.LocationRef{nearby}}); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != before {
		t.Fatalf("nearby locations must share cached station products within obsTTL via the client cache (%d -> %d requests)", before, got)
	}
}
