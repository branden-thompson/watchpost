package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: AI-1 (endpoint map, units, batched zone alerts, UA mandate) +
// architecture §10.1 (Fragment shapes). Fixtures recorded from the live-probe
// shapes in 02-analysis/research/AI-1-nws-api.md.

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func testServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var alertCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/"):
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(p, "/stations"):
			_, _ = w.Write(fixture(t, "stations.json"))
		case strings.HasSuffix(p, "/observations/latest"):
			_, _ = w.Write(fixture(t, "obs.json"))
		case strings.HasSuffix(p, "/forecast/hourly"):
			_, _ = w.Write(fixture(t, "hourly.json"))
		case strings.HasPrefix(p, "/gridpoints/") && !strings.Contains(p, "/forecast"):
			_, _ = w.Write(fixture(t, "gridpoint.json"))
		case strings.HasSuffix(p, "/forecast"):
			_, _ = w.Write(fixture(t, "forecast.json"))
		case p == "/alerts/active":
			alertCalls.Add(1)
			if got := r.URL.Query().Get("zone"); !strings.Contains(got, "CAZ554") || !strings.Contains(got, "CAC073") {
				t.Errorf("alerts must batch forecastZone AND county UGCs, got zone=%q", got)
			}
			if got := r.URL.Query().Get("status"); got != "actual" {
				t.Errorf("alerts must filter status=actual, got %q", got)
			}
			_, _ = w.Write(fixture(t, "alerts.json"))
		default:
			t.Errorf("unexpected path %s", p)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &alertCalls
}

func newProvider(t *testing.T, base string) *Provider {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return New(c, base)
}

var oceanside = snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2405, Lon: -117.2912, TZ: "America/Los_Angeles"}

func TestFetchObsNormalizesUnits(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil {
		t.Fatal(err)
	}
	pd := frag.PerLocation[snapshot.Key(oceanside)]
	c := pd.Current
	if c == nil {
		t.Fatal("no current conditions")
	}
	if c.Temp == nil || *c.Temp != 22.8 {
		t.Fatalf("temp = %v, want 22.8 (already degC)", c.Temp)
	}
	if c.Wind == nil || *c.Wind < 3.9 || *c.Wind > 4.1 {
		t.Fatalf("wind = %v m/s, want ~4.0 (14.4 km/h converted)", c.Wind)
	}
	if c.Pressure == nil || *c.Pressure < 1014 || *c.Pressure > 1015 {
		t.Fatalf("pressure = %v hPa, want ~1014.2 (Pa converted)", c.Pressure)
	}
	if c.WindGust != nil {
		t.Fatal("null gust must stay nil (null-parity rule)")
	}
	if c.Source.Provider != "nws" || c.Source.ModelOrStation != "KOKB" {
		t.Fatalf("source = %+v, want nws/KOKB (nearest station)", c.Source)
	}
	if c.Condition != "partly_cloudy" {
		t.Fatalf("condition = %q", c.Condition)
	}
	// UAT 60: the station's distance rides in Source (Oceanside -> KOKB ~6 km).
	if c.Source.DistanceKm == nil || *c.Source.DistanceKm < 5 || *c.Source.DistanceKm > 7 {
		t.Fatalf("station distance = %v, want ~6 km", c.Source.DistanceKm)
	}
}

func TestFetchForecastMapsPeriods(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindForecast, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil {
		t.Fatal(err)
	}
	pd := frag.PerLocation[snapshot.Key(oceanside)]
	if len(pd.Hourly) != 0 {
		t.Fatal("UAT 72: the daily kind must not carry the hourly series")
	}
	hf, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindForecastHourly, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil {
		t.Fatal(err)
	}
	if hp := hf.PerLocation[snapshot.Key(oceanside)]; len(hp.Hourly) != 2 {
		t.Fatalf("hourly = %d, want 2", len(hp.Hourly))
	} else if h := hp.Hourly[0]; h.Temp == nil || *h.Temp < 22.7 || *h.Temp > 22.9 { // 73F -> 22.8C
		t.Fatalf("hourly temp = %v, want ~22.8 (F converted)", h.Temp)
	}
	if len(pd.Daily) != 1 {
		t.Fatalf("daily = %d, want 1 (day+night pair folded)", len(pd.Daily))
	}
	d := pd.Daily[0]
	if d.TempMax == nil || d.TempMin == nil || *d.TempMax < 23.8 || *d.TempMax > 24.0 || *d.TempMin < 17.1 || *d.TempMin > 17.3 {
		t.Fatalf("daily = max %v min %v, want ~23.9/~17.2", d.TempMax, d.TempMin)
	}
	if d.PrecipProb == nil || *d.PrecipProb != 20 {
		t.Fatalf("daily precip prob = %v, want 20 (nil night value must not zero it)", d.PrecipProb)
	}
}

func TestFetchAlertsBatchesZonesAndParsesCAP(t *testing.T) {
	srv, calls := testServer(t)
	p := newProvider(t, srv.URL)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindAlerts, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("one batched alert call expected, got %d", calls.Load())
	}
	al := frag.PerLocation[snapshot.Key(oceanside)].Alerts
	if len(al) != 1 {
		t.Fatalf("alerts = %d, want 1", len(al))
	}
	a := al[0]
	if a.Event != "Heat Advisory" || a.Severity != "moderate" || a.ID == "" {
		t.Fatalf("alert = %+v", a)
	}
	if a.Sent.IsZero() || a.Instruction == "" {
		t.Fatal("CAP fields must carry through (sent, instruction)")
	}
}

func TestResolveCachesPoints(t *testing.T) {
	var pointCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/points/") {
			pointCalls.Add(1)
			_, _ = w.Write(fixture(t, "points.json"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stations") {
			_, _ = w.Write(fixture(t, "stations.json"))
			return
		}
		_, _ = w.Write(fixture(t, "obs.json"))
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	for range 3 {
		if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs, Locations: []snapshot.LocationRef{oceanside}}); err != nil {
			t.Fatal(err)
		}
	}
	if pointCalls.Load() != 1 {
		t.Fatalf("points must be resolved once and cached (AI-1: ~daily), got %d calls", pointCalls.Load())
	}
}

func TestIdentity(t *testing.T) {
	p := newProvider(t, "http://example.invalid")
	if p.ID() != "nws" {
		t.Fatal("ID must be nws")
	}
	ds := strings.Join(p.Domains(), ",")
	if !strings.Contains(ds, "weather") || !strings.Contains(ds, "alerts") {
		t.Fatalf("domains = %s", ds)
	}
}

// UAT 59 (Carlsbad, CA): the nearest NWS station (CBDSD, a mesonet site)
// reports no sky condition and an intermittent temperature — the row read
// "UNKNOWN n/a". Observations fall through the nearest stations until one
// reports a complete (temperature + condition) observation.
func TestFetchObsFallsBackPastSparseStation(t *testing.T) {
	var obsCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/"):
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(p, "/stations"):
			if r.URL.Query().Get("limit") == "1" {
				t.Error("stations must be requested as a fallback list, not limit=1")
			}
			_, _ = w.Write(fixture(t, "stations.json"))
		case strings.Contains(p, "/stations/KOKB/observations/latest"):
			obsCalls.Add(1)
			_, _ = w.Write(fixture(t, "obs_sparse.json"))
		case strings.Contains(p, "/stations/KCRQ/observations/latest"):
			obsCalls.Add(1)
			_, _ = w.Write(fixture(t, "obs.json"))
		default:
			t.Errorf("unexpected path %s", p)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	for range 2 {
		frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs, Locations: []snapshot.LocationRef{oceanside}})
		if err != nil || frag.Err != nil {
			t.Fatal(err, frag.Err)
		}
		c := frag.PerLocation[snapshot.Key(oceanside)].Current
		if c == nil || c.Temp == nil || c.Condition != "partly_cloudy" || c.Source.ModelOrStation != "KCRQ" {
			t.Fatalf("sparse KOKB must fall through to KCRQ, got %+v", c)
		}
	}
	// The station that last reported completely is tried first next cycle:
	// 2 calls on the first fetch (KOKB sparse, KCRQ), 1 on the second.
	if obsCalls.Load() != 3 {
		t.Fatalf("obs calls = %d, want 3 (preferred station remembered)", obsCalls.Load())
	}
}

func TestFetchContinuesPastFailedLocation(t *testing.T) {
	// UAT 59: a batch never fails at the first bad location — the others
	// still land in PerLocation; the failure travels in frag.Err.
	bad := snapshot.LocationRef{Label: "Nowhere", Lat: 10.5, Lon: 10.5}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/10.5"):
			w.WriteHeader(404) // NWS: point outside coverage
		case strings.HasPrefix(p, "/points/"):
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(p, "/stations"):
			_, _ = w.Write(fixture(t, "stations.json"))
		case strings.HasSuffix(p, "/observations/latest"):
			_, _ = w.Write(fixture(t, "obs.json"))
		case strings.HasSuffix(p, "/forecast/hourly"):
			_, _ = w.Write(fixture(t, "hourly.json"))
		case strings.HasSuffix(p, "/forecast"):
			_, _ = w.Write(fixture(t, "forecast.json"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	for _, kind := range []snapshot.FetchKind{snapshot.KindObs, snapshot.KindForecast} {
		frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: kind, Locations: []snapshot.LocationRef{bad, oceanside}})
		if err != nil {
			t.Fatal(err)
		}
		if frag.Err == nil || !strings.Contains(frag.Err.Error(), "Nowhere") {
			t.Fatalf("kind %d: the failed location must be named in frag.Err, got %v", kind, frag.Err)
		}
		if _, ok := frag.PerLocation[snapshot.Key(oceanside)]; !ok {
			t.Fatalf("kind %d: the good location must still land", kind)
		}
		if _, ok := frag.PerLocation[snapshot.Key(bad)]; ok {
			t.Fatalf("kind %d: a failed location must not publish", kind)
		}
	}
}

func TestResolveDedupesConcurrentPoints(t *testing.T) {
	// UAT 59: the alerts/obs/forecast tiers all fire at launch — one points
	// resolution must serve them all (singleflight), not three.
	var pointCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/points/"):
			pointCalls.Add(1)
			time.Sleep(20 * time.Millisecond)
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(r.URL.Path, "/stations"):
			_, _ = w.Write(fixture(t, "stations.json"))
		default:
			_, _ = w.Write(fixture(t, "obs.json"))
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs, Locations: []snapshot.LocationRef{oceanside}}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if pointCalls.Load() != 1 {
		t.Fatalf("concurrent resolves must share one points call, got %d", pointCalls.Load())
	}
}

func TestTodayHighFillsFromGridpointAfterSunset(t *testing.T) {
	// UAT 71: once local evening starts /forecast begins with "Tonight" —
	// no daytime period, so today's HIGH folded to nil ("n/a"). The raw
	// gridpoint still carries today's maxTemperature; it fills the hole
	// with provenance, and nws-marine's gridpoint fetch shares the download
	// through the client cache (one request per grid per cycle).
	var gridCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/"):
			_, _ = w.Write(fixture(t, "points.json"))
		case strings.HasSuffix(p, "/stations"):
			_, _ = w.Write(fixture(t, "stations.json"))
		case strings.HasSuffix(p, "/forecast/hourly"):
			_, _ = w.Write(fixture(t, "hourly.json"))
		case strings.HasSuffix(p, "/forecast"):
			_, _ = w.Write(fixture(t, "forecast_evening.json"))
		case strings.HasPrefix(p, "/gridpoints/"):
			gridCalls.Add(1)
			w.Header().Set("Cache-Control", "public, max-age=120")
			_, _ = w.Write(fixture(t, "gridpoint.json"))
		default:
			t.Errorf("unexpected path %s", p)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindForecast, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatal(err, frag.Err)
	}
	today := frag.PerLocation[snapshot.Key(oceanside)].Daily[0]
	if today.Date != "2026-08-23" || today.TempMin == nil || *today.TempMin < 17.1 || *today.TempMin > 17.3 {
		t.Fatalf("today folds from Tonight: %+v", today)
	}
	if today.TempMax == nil || *today.TempMax < 23.8 || *today.TempMax > 24.0 || today.FillFrom["temp_max"] != "nws:gridpoint" {
		t.Fatalf("today's HIGH must fill from the gridpoint with provenance: %+v", today)
	}
	if _, filled := today.FillFrom["temp_min"]; filled {
		t.Fatal("a value the forecast supplied is never overwritten")
	}
	if _, err := NewMarine(p).Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{oceanside}}); err != nil {
		t.Fatal(err)
	}
	if gridCalls.Load() != 1 {
		t.Fatalf("the daily fill and nws-marine must share one gridpoint download, got %d", gridCalls.Load())
	}
}

func TestCountyUGCFromResolvedPoint(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	if got := p.CountyUGC(context.Background(), oceanside); got != "CAC073" { // B4: radio SAME lookup
		t.Fatalf("county UGC = %q", got)
	}
	if got := p.Office(context.Background(), oceanside); got != "SGX" { // B4: synth products office
		t.Fatalf("office = %q", got)
	}
	if got := p.ForecastZone(context.Background(), oceanside); got != "CAZ554" { // UAT 81: zone-filtered narration
		t.Fatalf("forecast zone = %q", got)
	}
}
