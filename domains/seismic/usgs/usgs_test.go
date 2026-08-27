package usgs

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/seismic"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// quake is a test event the fake USGS server filters and serves.
type quake struct {
	id            string
	mag           float64
	lat, lon      float64
	depth         float64
	ago           time.Duration
	typ           string
	alert         string
	tsunami, felt int
}

func (q quake) json() string {
	feltJSON := "null"
	if q.felt > 0 {
		feltJSON = strconv.Itoa(q.felt)
	}
	return fmt.Sprintf(`{"type":"Feature","id":%q,"properties":{"mag":%.2f,"magType":"ml","place":%q,"time":%d,"tsunami":%d,"alert":%q,"felt":%s,"sig":%d,"type":%q},"geometry":{"type":"Point","coordinates":[%f,%f,%f]}}`,
		q.id, q.mag, q.id+" place", time.Now().Add(-q.ago).UnixMilli(), q.tsunami, q.alert, feltJSON, int(q.mag*100), q.typ, q.lon, q.lat, q.depth)
}

// fakeUSGS is a realistic FDSN endpoint: it filters the catalog by the
// latitude/longitude/maxradiuskm circle and the [minmagnitude,maxmagnitude]
// window, exactly as the real feed does, and records every request URL.
type fakeUSGS struct {
	mu      sync.Mutex
	catalog []quake
	reqs    []url.Values
}

func (f *fakeUSGS) handler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.mu.Lock()
	f.reqs = append(f.reqs, q)
	f.mu.Unlock()
	lat, _ := strconv.ParseFloat(q.Get("latitude"), 64)
	lon, _ := strconv.ParseFloat(q.Get("longitude"), 64)
	radKm, _ := strconv.ParseFloat(q.Get("maxradiuskm"), 64)
	minMag, _ := strconv.ParseFloat(q.Get("minmagnitude"), 64)
	maxMag := math.Inf(1)
	if s := q.Get("maxmagnitude"); s != "" {
		maxMag, _ = strconv.ParseFloat(s, 64)
	}
	var feats []string
	for _, e := range f.catalog {
		if e.mag < minMag || e.mag > maxMag {
			continue
		}
		if geo.HaversineKM(lat, lon, e.lat, e.lon) > radKm {
			continue
		}
		feats = append(feats, e.json())
	}
	w.Header().Set("Content-Type", "application/json")
	body := `{"type":"FeatureCollection","features":[`
	for i, s := range feats {
		if i > 0 {
			body += ","
		}
		body += s
	}
	body += `]}`
	_, _ = w.Write([]byte(body))
}

func (f *fakeUSGS) urls() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.reqs...)
}

func serveFake(t *testing.T, catalog []quake) (*fakeUSGS, *httptest.Server) {
	t.Helper()
	f := &fakeUSGS{catalog: catalog}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	return f, srv
}

func client(t *testing.T) *httpx.Client {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var oceanside = snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.20, Lon: -117.38}

// degNorth is a north offset in degrees for a km distance.
func degNorth(km float64) float64 { return km / 111.0 }

func TestFetchAppliesTheGraduatedRuleAndSorts(t *testing.T) {
	cat := []quake{
		{id: "big", mag: 4.5, lat: oceanside.Lat + degNorth(120), lon: oceanside.Lon, depth: 8, ago: 2 * time.Hour, typ: "earthquake"}, // M4.5 at 120 km: 150 mi band → shown
		{id: "far", mag: 2.6, lat: oceanside.Lat + degNorth(50), lon: oceanside.Lon, depth: 3, ago: 24 * time.Hour, typ: "earthquake"}, // M2.6 at 50 km: 20 mi band → hidden
		{id: "near", mag: 2.6, lat: oceanside.Lat + degNorth(15), lon: oceanside.Lon, depth: 3, ago: 3 * time.Hour, typ: "earthquake"}, // M2.6 at 15 km: 20 mi band → shown
		{id: "blast", mag: 3.1, lat: oceanside.Lat + degNorth(2), lon: oceanside.Lon, depth: 1, ago: time.Hour, typ: "quarry blast"},   // not an earthquake → hidden
	}
	f, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	st := frag.PerLocation[snapshot.Key(oceanside)].Seismic
	if st == nil || st.AsOf.IsZero() {
		t.Fatal("a feed that answered sets AsOf")
	}
	if len(st.Quakes) != 2 {
		t.Fatalf("only the two in-band earthquakes show: %+v", st.Quakes)
	}
	if st.Quakes[0].Mag != 4.5 || st.Quakes[1].Mag != 2.6 { // largest first
		t.Fatalf("largest magnitude first: %v", st.Quakes)
	}
	q := st.Quakes[0]
	if q.Bearing != "N" || q.DistanceKm < 110 || q.DistanceKm > 130 || q.DepthKm != 8 || q.MagType != "ml" {
		t.Fatalf("distance/bearing/depth computed from the location: %+v", q)
	}
	// Two concentric queries went out: a near-field window and a regional one.
	urls := f.urls()
	if len(urls) != 2 {
		t.Fatalf("concentric fetch = near + regional = 2 queries, got %d", len(urls))
	}
	var sawNear, sawRegional bool
	for _, u := range urls {
		if u.Get("format") != "geojson" {
			t.Fatalf("query is geojson: %v", u)
		}
		if _, err := time.Parse("2006-01-02", u.Get("starttime")); err != nil {
			t.Fatalf("starttime is the lookback window: %q", u.Get("starttime"))
		}
		if u.Get("maxmagnitude") == "3.50" {
			sawNear = true
		}
		if u.Get("minmagnitude") == "3.50" && u.Get("maxmagnitude") == "" {
			sawRegional = true
		}
	}
	if !sawNear || !sawRegional {
		t.Fatalf("expected a near [0,3.5) and a regional [3.5,∞) query: %v", urls)
	}
}

// The near-field query is centred on the location itself (never snapped); the
// regional query snaps to the shared grid — two nearby locations issue one
// regional URL, a distant location a different one.
func TestRegionalBoxIsSharedAndNearIsPerLocation(t *testing.T) {
	_, srv := serveFake(t, nil)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	a := snapshot.LocationRef{Label: "A", Lat: 34.05, Lon: -118.25}  // Los Angeles
	b := snapshot.LocationRef{Label: "B", Lat: 34.42, Lon: -119.70}  // Santa Barbara (~140 km, same 4° cell)
	far := snapshot.LocationRef{Label: "C", Lat: 40.71, Lon: -74.01} // New York (different cell)
	now := time.Now()
	regOf := func(ref snapshot.LocationRef) string {
		for _, q := range p.queries(ref, now) {
			if q.maxMag == 0 { // the open regional window
				return q.url
			}
		}
		return ""
	}
	nearOf := func(ref snapshot.LocationRef) string {
		for _, q := range p.queries(ref, now) {
			if q.maxMag != 0 {
				return q.url
			}
		}
		return ""
	}
	if regOf(a) != regOf(b) {
		t.Fatalf("nearby locations must share one regional box:\n A=%s\n B=%s", regOf(a), regOf(b))
	}
	if regOf(a) == regOf(far) {
		t.Fatal("a distant location must use a different regional box")
	}
	if nearOf(a) == nearOf(b) {
		t.Fatal("the near-field query is per-location, not snapped — distinct centres, distinct URLs")
	}
	// The regional URL carries the snapped centre (a multiple of 4°), not A's.
	ru, _ := url.Parse(regOf(a))
	if lat := ru.Query().Get("latitude"); lat == fmt.Sprintf("%.4f", a.Lat) {
		t.Fatalf("regional latitude should be snapped to the grid, got the raw %s", lat)
	}
}

// The shared regional body is parsed once no matter how many locations read
// it, and a co-located pair issues a single regional network request.
func TestBoxMemoParsesSharedBodyOnce(t *testing.T) {
	cat := []quake{{id: "m4", mag: 4.2, lat: 34.4, lon: -118.4, depth: 10, ago: time.Hour, typ: "earthquake"}}
	f, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	// Ten locations in the same 4° cell → one shared regional request+parse,
	// plus ten distinct near-field requests.
	var locs []snapshot.LocationRef
	for i := 0; i < 10; i++ {
		locs = append(locs, snapshot.LocationRef{Label: fmt.Sprint(i), Lat: 34.0 + float64(i)*0.05, Lon: -118.0 - float64(i)*0.05})
	}
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: locs}); err != nil {
		t.Fatal(err)
	}
	boxes, parses := p.MemoStats()
	// 10 near boxes + 1 shared regional box = 11 distinct bodies parsed once.
	if boxes != 11 || parses != 11 {
		t.Fatalf("expected 11 boxes / 11 parses (regional shared), got boxes=%d parses=%d", boxes, parses)
	}
	// The regional network request went out once for all ten locations.
	regionalReqs := 0
	for _, u := range f.urls() {
		if u.Get("maxmagnitude") == "" {
			regionalReqs++
		}
	}
	if regionalReqs != 1 {
		t.Fatalf("the shared regional box is one request for the whole cell, got %d", regionalReqs)
	}
}

// The concentric fetch returns exactly what a single wide magnitude-0 query
// would (the equivalence property): a spread of magnitudes at a spread of
// distances, filtered by Keep, must match a brute-force reference.
func TestConcentricFetchEqualsSingleWideQuery(t *testing.T) {
	rules := seismic.DefaultRules()
	var cat []quake
	id := 0
	// For each magnitude, straddle its band edge: one event just inside (must
	// show) and one just outside (must not) — 9 visible, under the cap, and it
	// exercises inclusion and exclusion at every boundary and across the pivot.
	for _, mag := range []float64{0.6, 1.2, 2.6, 3.2, 3.6, 4.2, 4.7, 5.5, 6.5} {
		edgeMi := rules.RadiusMiFor(mag)
		for _, dMi := range []float64{edgeMi * 0.8, edgeMi * 1.2} {
			id++
			cat = append(cat, quake{id: fmt.Sprintf("e%d", id), mag: mag, lat: oceanside.Lat + degNorth(dMi*seismic.MileKm), lon: oceanside.Lon, depth: 5, ago: time.Hour, typ: "earthquake"})
		}
	}
	// Brute-force reference: every event Keep would show from the location.
	want := map[string]bool{}
	for _, e := range cat {
		if rules.Keep(e.mag, geo.HaversineKM(oceanside.Lat, oceanside.Lon, e.lat, e.lon)) {
			want[e.id] = true
		}
	}
	_, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, rules)
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	got := map[string]bool{}
	for _, q := range frag.PerLocation[snapshot.Key(oceanside)].Seismic.Quakes {
		got[q.Place[:len(q.Place)-6]] = true // "eNN place" → "eNN"
	}
	if len(got) != len(want) {
		t.Fatalf("concentric visible set size %d != brute-force %d", len(got), len(want))
	}
	for id := range want {
		if !got[id] {
			t.Fatalf("equivalence: %s should be visible but the concentric fetch missed it", id)
		}
	}
	for id := range got {
		if !want[id] {
			t.Fatalf("equivalence: %s is visible in the concentric fetch but Keep would hide it", id)
		}
	}
}

func TestAsOfDistinguishesNoneFromUnavailable(t *testing.T) {
	// Feed answered with nothing in band: AsOf set, zero quakes = "no recent activity".
	_, srv := serveFake(t, nil)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	st := frag.PerLocation[snapshot.Key(oceanside)].Seismic
	if st == nil || st.AsOf.IsZero() || len(st.Quakes) != 0 {
		t.Fatalf("answered-but-empty is AsOf set, no quakes: %+v", st)
	}
	// Feed down: no PerLocation entry, frag.Err set = "unavailable" (never a fake none).
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer down.Close()
	p2 := New(client(t), down.URL, seismic.DefaultRules())
	frag2, _ := p2.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if frag2.Err == nil || frag2.PerLocation[snapshot.Key(oceanside)].Seismic != nil {
		t.Fatal("a down feed reports an error and no state — the assembler keeps last-good, the UI reads 'unavailable'")
	}
}

func TestFetchCapsASwarm(t *testing.T) {
	var cat []quake
	for i := 0; i < 40; i++ {
		cat = append(cat, quake{id: fmt.Sprintf("s%d", i), mag: 2.6, lat: oceanside.Lat + degNorth(float64(i%10)), lon: oceanside.Lon, depth: 3, ago: time.Duration(i) * time.Hour, typ: "earthquake"})
	}
	_, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if st := frag.PerLocation[snapshot.Key(oceanside)].Seismic; len(st.Quakes) != maxQuakes {
		t.Fatalf("a swarm is capped at %d, got %d", maxQuakes, len(st.Quakes))
	}
}

// REVIEW P5 F1: a quake at the very edge of its band (inside by a fraction of
// a mile, measured in km) must show — the query radius rounds UP, so the last
// fraction is fetched rather than truncated away.
func TestBandEdgeQuakeIsNotTruncated(t *testing.T) {
	// M3.0 → 20 mi band = 32.18688 km. A quake at 32.1 km is inside; before the
	// %.0f→ceil fix the near query asked maxradiuskm=32 and dropped it.
	cat := []quake{{id: "edge", mag: 3.0, lat: oceanside.Lat + degNorth(32.1), lon: oceanside.Lon, depth: 5, ago: time.Hour, typ: "earthquake"}}
	_, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if st := frag.PerLocation[snapshot.Key(oceanside)].Seismic; st == nil || len(st.Quakes) != 1 {
		t.Fatalf("a quake at the km edge of its band must show (radius rounds up): %+v", st)
	}
}

// REVIEW P5 F2: a Keep-visible negative-magnitude quake underfoot (USGS ml runs
// slightly negative) must show — the near query's floor is below any real quake.
func TestNegativeMagnitudeUnderfootIsShown(t *testing.T) {
	cat := []quake{{id: "neg", mag: -0.5, lat: oceanside.Lat + degNorth(3), lon: oceanside.Lon, depth: 2, ago: time.Hour, typ: "earthquake"}}
	_, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if st := frag.PerLocation[snapshot.Key(oceanside)].Seismic; st == nil || len(st.Quakes) != 1 || st.Quakes[0].Mag != -0.5 {
		t.Fatalf("a Keep-visible negative-magnitude quake underfoot must show: %+v", st)
	}
}

// REVIEW P5 F1 (partial fetch): if one concentric leg fails, the location must
// keep its last-good state — publishing the other leg's subset would silently
// drop the failed leg's whole magnitude band.
func TestPartialFetchDoesNotPublish(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("maxmagnitude") == "" { // the open regional query
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer srv.Close()
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	if frag.Err == nil {
		t.Fatal("a failed regional leg must surface an error")
	}
	if _, ok := frag.PerLocation[snapshot.Key(oceanside)]; ok {
		t.Fatal("a partial fetch must not publish — the location keeps last-good (REVIEW P5 F1)")
	}
}

// REVIEW P5 E-A: a hostile/malformed feed value (magnitude 1e300, or an absurd
// depth) is rejected at decode — it would otherwise sort first and render a
// ~300-character token that tears the modal, and read a 300-digit number aloud.
func TestImplausibleFeedValuesAreRejected(t *testing.T) {
	cat := []quake{
		{id: "hugemag", mag: 1e300, lat: oceanside.Lat, lon: oceanside.Lon, depth: 5, ago: time.Hour, typ: "earthquake"},
		{id: "deepbad", mag: 4.0, lat: oceanside.Lat, lon: oceanside.Lon, depth: 1e300, ago: time.Hour, typ: "earthquake"},
		{id: "good", mag: 4.0, lat: oceanside.Lat, lon: oceanside.Lon, depth: 10, ago: time.Hour, typ: "earthquake"},
	}
	_, srv := serveFake(t, cat)
	p := New(client(t), srv.URL, seismic.DefaultRules())
	frag, _ := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{oceanside}})
	st := frag.PerLocation[snapshot.Key(oceanside)].Seismic
	if st == nil || len(st.Quakes) != 1 || st.Quakes[0].Mag != 4.0 || st.Quakes[0].DepthKm != 10 {
		t.Fatalf("only the plausible quake survives (implausible magnitude/depth rejected): %+v", st)
	}
}

// Live proof against the real USGS feed (WATCHPOST_LIVE=1; skipped in CI).
func TestLiveUSGSAnswers(t *testing.T) {
	if os.Getenv("WATCHPOST_LIVE") == "" {
		t.Skip("set WATCHPOST_LIVE=1 to query the real USGS FDSN feed")
	}
	c, _ := httpx.New(httpx.Config{UserAgent: "watchpost-test (t@example.com)", RatePerSec: 2, MaxRetries: 1})
	p := New(c, "", seismic.DefaultRules())
	ridgecrest := snapshot.LocationRef{Label: "Ridgecrest, CA", Lat: 35.62, Lon: -117.67}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: []snapshot.LocationRef{ridgecrest}})
	if err != nil || frag.Err != nil {
		t.Fatalf("live fetch: %v %v", err, frag.Err)
	}
	st := frag.PerLocation[snapshot.Key(ridgecrest)].Seismic
	boxes, parses := p.MemoStats()
	t.Logf("Ridgecrest: %d quakes in the last %d days (memo boxes=%d parses=%d)", len(st.Quakes), seismic.DefaultRules().LookbackDays, boxes, parses)
	for _, q := range st.Quakes[:min(5, len(st.Quakes))] {
		t.Logf("  M%.1f %s  %.0f km %s  depth %.0f km  %s", q.Mag, q.Place, q.DistanceKm, q.Bearing, q.DepthKm, time.Since(q.At).Round(time.Hour))
	}
}

// countersSet is a representative watchlist for the counters gate: a SoCal
// cluster that shares one regional box, plus a national/Alaska spread — the
// 10-favourite + RECENT shape the plan measures (§1 targets).
func countersSet() []snapshot.LocationRef {
	return []snapshot.LocationRef{
		{Label: "Los Angeles, CA", Lat: 34.05, Lon: -118.25},
		{Label: "Santa Barbara, CA", Lat: 34.42, Lon: -119.70},
		{Label: "Oceanside, CA", Lat: 33.20, Lon: -117.38},
		{Label: "San Diego, CA", Lat: 32.72, Lon: -117.16},
		{Label: "Riverside, CA", Lat: 33.95, Lon: -117.40},
		{Label: "Ridgecrest, CA", Lat: 35.62, Lon: -117.67},
		{Label: "San Francisco, CA", Lat: 37.77, Lon: -122.42},
		{Label: "Sacramento, CA", Lat: 38.58, Lon: -121.49},
		{Label: "Seattle, WA", Lat: 47.61, Lon: -122.33},
		{Label: "Portland, OR", Lat: 45.52, Lon: -122.68},
		{Label: "Salt Lake City, UT", Lat: 40.76, Lon: -111.89},
		{Label: "Denver, CO", Lat: 39.74, Lon: -104.99},
		{Label: "Houston, TX", Lat: 29.76, Lon: -95.37},
		{Label: "New York, NY", Lat: 40.71, Lon: -74.01},
		{Label: "Anchorage, AK", Lat: 61.22, Lon: -149.90},
		{Label: "Honolulu, HI", Lat: 21.31, Lon: -157.86},
	}
}

// TestSeismicCountersFloor drives the real production path (httpx cache +
// conditional GET + shared boxes + box memo) against live USGS across tier
// ticks, and logs the per-host request/byte floor and the memo bound — the
// seismic P2 gate (plan §5). Skipped unless WATCHPOST_SEISMIC_COUNTERS=1;
// WATCHPOST_COUNTERS_MINUTES (default 60) and WATCHPOST_COUNTERS_TICK_MIN
// (default 5) set the run. It never fails on network noise — it reports.
func TestSeismicCountersFloor(t *testing.T) {
	if os.Getenv("WATCHPOST_SEISMIC_COUNTERS") == "" {
		t.Skip("set WATCHPOST_SEISMIC_COUNTERS=1 for the 1-hour live counters gate")
	}
	total := envMinutes("WATCHPOST_COUNTERS_MINUTES", 60)
	step := envMinutes("WATCHPOST_COUNTERS_TICK_MIN", 5)
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost-seismic-counters/0.11 (+github.com/branden-thompson/watchpost)", RatePerSec: 30, MaxRetries: 2, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	p := New(c, "", seismic.DefaultRules())
	locs := countersSet()
	ticks := int(total/step) + 1
	t.Logf("counters: %d locations, %d ticks, %s cadence, %s window", len(locs), ticks, step, total)
	host := func() httpx.HostStats {
		for _, h := range c.RequestStats().Hosts {
			if h.Host == "earthquake.usgs.gov" {
				return h
			}
		}
		return httpx.HostStats{}
	}
	for i := 0; i < ticks; i++ {
		frag, ferr := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindSeismic, Locations: locs})
		quakes := 0
		for _, pd := range frag.PerLocation {
			if pd.Seismic != nil {
				quakes += len(pd.Seismic.Quakes)
			}
		}
		h := host()
		boxes, parses := p.MemoStats()
		t.Logf("tick %2d/%d  err=%v fragErr=%v  quakes=%d | host: attempts=%d net=%d 304=%d cache=%d bytesNet=%d bytes304saved=%d | memo: boxes=%d parses=%d",
			i+1, ticks, ferr, frag.Err, quakes, h.Attempts, h.Net, h.NotModified, h.Cache, h.BytesNet, h.Bytes304, boxes, parses)
		if i < ticks-1 {
			time.Sleep(step)
		}
	}
	h := host()
	boxes, parses := p.MemoStats()
	perHourReq := float64(h.Attempts) / (float64(total) / float64(time.Hour))
	perHourBytes := float64(h.BytesNet) / (float64(total) / float64(time.Hour))
	t.Logf("SUMMARY over %s: attempts=%d (%.0f/h) net=%d 304=%d bytesNet=%d (%.0f/h) bytes304saved=%d | memo boxes=%d (bound %d) parses=%d",
		total, h.Attempts, perHourReq, h.Net, h.NotModified, h.BytesNet, perHourBytes, h.Bytes304, boxes, maxBoxes, parses)
	if boxes > maxBoxes {
		t.Fatalf("box memo exceeded its bound %d: %d", maxBoxes, boxes)
	}
}

// envMinutes reads a minutes value from the environment, or a default.
func envMinutes(key string, def int) time.Duration {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return time.Duration(def) * time.Minute
}
