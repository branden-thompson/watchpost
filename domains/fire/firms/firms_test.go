package firms

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

// csvBody follows the documented VIIRS area-API columns (AI-3): a high-
// confidence night detection 6 km out, a low-confidence one (dropped), a
// weak 2 MW one (dropped), and one outside the ring.
const csvBody = `latitude,longitude,bright_ti4,scan,track,acq_date,acq_time,satellite,instrument,confidence,version,bright_ti5,frp,daynight
33.29,-117.24,345.2,0.39,0.36,2026-08-25,0912,N20,VIIRS,h,2.0NRT,295.1,61.5,N
33.30,-117.25,330.0,0.39,0.36,2026-08-25,0912,N20,VIIRS,l,2.0NRT,290.0,30.0,N
33.31,-117.26,320.0,0.39,0.36,2026-08-25,0912,N20,VIIRS,n,2.0NRT,290.0,2.0,D
34.05,-118.24,345.2,0.39,0.36,2026-08-25,0912,N20,VIIRS,h,2.0NRT,295.1,80.0,D
`

func TestFetchUsesTheKeyPerLocationAndSourceAndAppliesTheRules(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(csvBody))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	key := "0123456789abcdef0123456789abcdef"
	p := New(c, srv.URL, key, fire.DefaultRules())
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	if len(paths) != 2 || !strings.Contains(paths[0], "/api/area/csv/"+key+"/VIIRS_NOAA20_NRT/") || !strings.HasSuffix(paths[0], "/1") {
		t.Fatalf("one request per source with the key in the path: %v", paths)
	}
	fs := frag.PerLocation[snapshot.Key(oceanside)].Fire
	if fs == nil || len(fs.Hotspots) != 1 {
		t.Fatalf("only the confident, strong, near detection survives (clustered across the two sources): %+v", fs)
	}
	h := fs.Hotspots[0]
	if h.Confidence != "high" || *h.FRPMW != 61.5 || h.Source.Provider != "firms" || h.Source.ModelOrStation != "NOAA-20" || h.DetectedAt.Format("2006-01-02 15:04") != "2026-08-25 09:12" {
		t.Fatalf("fields: %+v", h)
	}
	// No key: silent, nothing contributed.
	off := New(c, srv.URL, "", fire.DefaultRules())
	frag, err = off.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil || len(frag.PerLocation) != 0 || off.Enabled() {
		t.Fatalf("no key: empty fragment, no error: %+v %v", frag, err)
	}
}

func TestParseCSVRefusesAnErrorPage(t *testing.T) {
	if _, err := ParseCSV([]byte("Invalid MAP_KEY.\n")); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("an error page is not data: %v", err)
	}
	if pts, err := ParseCSV(nil); err != nil || len(pts) != 0 {
		t.Fatal("empty is empty")
	}
	if confidence("85") != "high" || confidence("50") != "nominal" || confidence("10") != "low" || confidence("?") != "nominal" {
		t.Fatal("MODIS numeric confidence maps to the scale")
	}
}

func TestFetchFailSoft(t *testing.T) {
	// Red-team B5 P2/P4: a source that fails leaves the location unserved
	// (retried, prior hotspots kept — never an empty state); a rejected key
	// is said once in words, without the key, and the cycle stops.
	key := "0123456789abcdef0123456789abcdef"
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	newClient := func() *httpx.Client {
		c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
		return c
	}
	var calls int
	flaky := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Path, "NOAA21") {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(csvBody))
	}))
	defer flaky.Close()
	p := New(newClient(), flaky.URL, key, fire.DefaultRules())
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err == nil {
		t.Fatalf("a failed source is a fragment error: %v %v", err, frag.Err)
	}
	if _, served := frag.PerLocation[snapshot.Key(oceanside)]; served {
		t.Fatal("a partly failed location must stay unserved so it retries and keeps its prior data")
	}
	calls = 0
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid MAP_KEY."))
	}))
	defer bad.Close()
	p = New(newClient(), bad.URL, key, fire.DefaultRules())
	frag, _ = p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside, {Label: "El Cajon, CA", Lat: 32.79, Lon: -116.96}}})
	if frag.Err == nil || !strings.Contains(frag.Err.Error(), "rejected the MAP_KEY") || strings.Contains(frag.Err.Error(), key) {
		t.Fatalf("a rejected key is named in words, never echoed: %v", frag.Err)
	}
	if calls != 1 {
		t.Fatalf("stop after the first rejection, got %d requests", calls)
	}
}
