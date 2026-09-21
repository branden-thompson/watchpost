package zones

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// serveZones answers like the service does, counting what was asked for.
func serveZones(t *testing.T, hits *atomic.Int64) *httptest.Server {
	t.Helper()
	dallas, err := os.ReadFile("../../../../platform/geo/testdata/zone-dallas.geojson")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if id != "TXZ119" && id != "OHZ001" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/geo+json")
		w.Header().Set("Cache-Control", "public, max-age=426444")
		_, _ = w.Write([]byte(`{"properties":{"id":"` + id + `","name":"Dallas"},"geometry":` + string(dallas) + `}`))
	}))
}

func newStore(t *testing.T, base string) *Store {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost-test"})
	if err != nil {
		t.Fatal(err)
	}
	return New(c, base)
}

// TestAZoneIsFetchedOnceAndKeptWholeWithItsName. A shape is stored as numbers,
// not as the text it arrived in (MG-12), and the name rides with it because a
// description will want it and it is in the same answer (MG-9).
func TestAZoneIsFetchedOnceAndKeptWholeWithItsName(t *testing.T) {
	var hits atomic.Int64
	srv := serveZones(t, &hits)
	defer srv.Close()
	s := newStore(t, srv.URL)

	z, err := s.Zone(context.Background(), "TXZ119")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name != "Dallas" {
		t.Errorf("the zone's name is %q", z.Name)
	}
	if z.Area.Vertices() != 80 {
		t.Errorf("the zone kept %d positions; the service sent 80", z.Area.Vertices())
	}
	// Asked for again, it is not fetched again.
	if _, err := s.Zone(context.Background(), "TXZ119"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Errorf("the service was asked %d times for one zone", hits.Load())
	}
}

// TestAZoneThatDoesNotExistIsAnErrorNotAPanic.
func TestAZoneThatDoesNotExistIsAnErrorNotAPanic(t *testing.T) {
	var hits atomic.Int64
	srv := serveZones(t, &hits)
	defer srv.Close()
	s := newStore(t, srv.URL)
	if _, err := s.Zone(context.Background(), "ZZZ999"); err == nil {
		t.Error("a zone the service does not have was accepted")
	}
}

// TestManyZonesAreAskedForOnceEach is the tail the measurements found: an
// alert can name forty-two zones, and the same zone can be named by several
// alerts. Each distinct id costs one request, and the answer says which were
// got and which were not - the caller decides what to do about a gap (MG-10).
func TestManyZonesAreAskedForOnceEach(t *testing.T) {
	var hits atomic.Int64
	srv := serveZones(t, &hits)
	defer srv.Close()
	s := newStore(t, srv.URL)

	got, missing := s.Zones(context.Background(), []string{"TXZ119", "OHZ001", "TXZ119", "NOPE1"})
	if len(got) != 2 {
		t.Errorf("%d zones came back; two of the four ids exist", len(got))
	}
	if len(missing) != 1 || missing[0] != "NOPE1" {
		t.Errorf("the missing ids are %v; want [NOPE1]", missing)
	}
	// Two, not three: `NOPE1` names neither a forecast zone nor a county, so
	// it is refused here rather than asked hopefully of a public service.
	if hits.Load() != 2 {
		t.Errorf("the service was asked %d times for the two ids that are zone ids", hits.Load())
	}
}

// TestSeedingTakesTheWatchedZonesBeforeAnyAlertNeedsThem is why seeding exists
// (MG-7). On-demand alone is coldest during severe weather - when many zones
// activate at once and the map matters most - and a watched location has about
// two zones, so this costs a second, once.
func TestSeedingTakesTheWatchedZonesBeforeAnyAlertNeedsThem(t *testing.T) {
	var hits atomic.Int64
	srv := serveZones(t, &hits)
	defer srv.Close()
	s := newStore(t, srv.URL)

	s.Seed(context.Background(), []string{"TXZ119", "OHZ001", "TXZ119"})
	if s.Held() != 2 {
		t.Errorf("seeding holds %d zones; two distinct ids were given", s.Held())
	}
	before := hits.Load()
	if _, err := s.Zone(context.Background(), "TXZ119"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != before {
		t.Error("a seeded zone was fetched again when it was asked for")
	}
}

// TestSeedingDoesNotFailWhenTheServiceIsUnreachable: a cold start with no
// network is an ordinary morning, not an error. What could not be got is simply
// not held, and the ordinary path will ask again.
func TestSeedingDoesNotFailWhenTheServiceIsUnreachable(t *testing.T) {
	s := newStore(t, "http://127.0.0.1:1") // nothing listens here
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.Seed(ctx, []string{"TXZ119"})
	if s.Held() != 0 {
		t.Errorf("%d zones were held from a service that never answered", s.Held())
	}
}

// TestManyZonesAreFetchedTogether is RT-3: the tail is forty-two zones, and
// fetched one after another at a sixth of a second each that is six seconds of
// nothing. They are asked for together.
func TestManyZonesAreFetchedTogether(t *testing.T) {
	var hits atomic.Int64
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(120 * time.Millisecond) // about what the service takes
		w.Header().Set("Cache-Control", "public, max-age=426444")
		_, _ = w.Write([]byte(`{"properties":{"id":"z","name":"Z"},"geometry":{"type":"Polygon","coordinates":[[[-85,41],[-84,41],[-84,42],[-85,41]]]}}`))
	}))
	defer slow.Close()
	// A client whose token bucket is not the thing under test. **In the real
	// one the bucket is the bound, not the ordering**: five a second means a
	// forty-two-zone alert takes eight seconds however it is issued, which is
	// the whole reason the watched places' zones are seeded (MG-7, RT-2).
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost-test", RatePerSec: 200})
	if err != nil {
		t.Fatal(err)
	}
	s := New(c, slow.URL)

	ids := make([]string, 12)
	for i := range ids {
		ids[i] = string(rune('A'+i)) + "XZ001"
	}
	start := time.Now()
	got, _ := s.Zones(context.Background(), ids)
	took := time.Since(start)
	if len(got) != len(ids) {
		t.Fatalf("%d of %d zones came back", len(got), len(ids))
	}
	// Serially this is twelve times 120 ms. Together it is near one.
	if took > 700*time.Millisecond {
		t.Errorf("twelve zones took %v; fetched together they should take about one request's time", took.Round(time.Millisecond))
	}
}

// TestTheStoreDoesNotGrowForEver is RT-4. A zone asked for once is worth
// keeping; every zone ever asked for, on a program that runs for days, is not.
func TestTheStoreDoesNotGrowForEver(t *testing.T) {
	var hits atomic.Int64
	srv := serveZones(t, &hits)
	defer srv.Close()
	s := newStore(t, srv.URL)
	s.mu.Lock()
	for i := range maxHeld + 50 {
		s.held[string(rune('a'+i%26))+string(rune('a'+i/26))] = Zone{ID: "x"}
	}
	s.mu.Unlock()
	s.forget()
	if n := s.Held(); n > maxHeld {
		t.Errorf("the store holds %d shapes; the cap is %d", n, maxHeld)
	}
}

// serveByKind answers only at the path the real service answers at: a county
// zone under /zones/county and a forecast zone under /zones/forecast, each a
// 404 at the other. Probed live against api.weather.gov, which is where these
// two lines come from:
//
//	/zones/forecast/INC003 -> 404
//	/zones/county/INC003   -> 200
func serveByKind(t *testing.T, asked *[]string) *httptest.Server {
	t.Helper()
	dallas, err := os.ReadFile("../../../../platform/geo/testdata/zone-dallas.geojson")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*asked = append(*asked, r.URL.Path)
		mu.Unlock()
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		kind := "/zones/forecast/"
		if len(id) > 2 && id[2] == 'C' {
			kind = "/zones/county/"
		}
		if r.URL.Path != kind+id {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/geo+json")
		_, _ = w.Write([]byte(`{"properties":{"id":"` + id + `","name":"somewhere"},"geometry":` + string(dallas) + `}`))
	}))
}

// TestACountyZoneIsFetchedWhereCountyZonesLive.
//
// **A zone id says which kind it is in its third character** - Z for a
// forecast zone, C for a county - and the two live at different paths.
// Everything was asked for under /zones/forecast, so every county id answered
// 404 and resolved to nothing at all.
//
// Measured against the live service on the day this was written: of 332 active
// alerts, 47 named county zones and nothing else, and 45 of those 47 were
// Flood Warnings. There is no mixed case to fall back on - not one alert named
// both kinds - so a county-only alert had no area, permanently, and flood is
// the hazard a map is most wanted for.
func TestACountyZoneIsFetchedWhereCountyZonesLive(t *testing.T) {
	var asked []string
	srv := serveByKind(t, &asked)
	defer srv.Close()
	s := newStore(t, srv.URL)

	got, missing := s.Zones(context.Background(), []string{"INC003", "INZ027"})
	if len(missing) != 0 {
		t.Errorf("could not get %v", missing)
	}
	if len(got) != 2 {
		t.Fatalf("%d of two zones came back: %v", len(got), got)
	}
	for _, id := range []string{"INC003", "INZ027"} {
		if got[id].Area.Empty() {
			t.Errorf("%s came back with no shape", id)
		}
	}
	sort.Strings(asked)
	want := []string{"/zones/county/INC003", "/zones/forecast/INZ027"}
	if len(asked) != 2 || asked[0] != want[0] || asked[1] != want[1] {
		t.Errorf("asked for %v; the two kinds live at %v", asked, want)
	}
}

// TestAZoneIdOfNoKnownKindIsRefused: the kind is read, never guessed. An id
// that names neither kind is not quietly sent to one of them.
func TestAZoneIdOfNoKnownKindIsRefused(t *testing.T) {
	var asked []string
	srv := serveByKind(t, &asked)
	defer srv.Close()
	s := newStore(t, srv.URL)

	for _, id := range []string{"INX003", "IN", ""} {
		if _, err := s.Zone(context.Background(), id); err == nil {
			t.Errorf("%q was accepted as a zone id", id)
		}
	}
	if len(asked) != 0 {
		t.Errorf("the service was asked for %v; none of those is a zone id", asked)
	}
}

// TestTheStoreForgetsThroughItsOwnFrontDoor is RT-4's call site, which the
// first test of it never touched: it called forget by hand, so deleting the
// line that calls forget left the suite green. This drives Zone, which is the
// only way the cap ever fires in the running program.
func TestTheStoreForgetsThroughItsOwnFrontDoor(t *testing.T) {
	var asked []string
	srv := serveByKind(t, &asked)
	defer srv.Close()
	s := newStore(t, srv.URL)
	s.limit = 4 // the real cap needs two thousand fetches to reach

	for i := range 9 {
		id := "INZ" + string(rune('A'+i)) + "00"
		if _, err := s.Zone(context.Background(), id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if n := s.Held(); n > s.limit {
		t.Errorf("the store holds %d shapes through its own path; the cap is %d", n, s.limit)
	}
	if n := s.Held(); n == 0 {
		t.Error("the store holds nothing at all; the cap is not meant to empty it")
	}
}

// TestAskingForMoreZonesThanExistIsBoundedAndSaysSo is the fan-out the
// infosec review found: nothing capped how many zones one resolve could
// demand, and the client paces everything the program does at five a second,
// so a large enough alerts response is hours with no weather fetched.
func TestAskingForMoreZonesThanExistIsBoundedAndSaysSo(t *testing.T) {
	var asked []string
	srv := serveByKind(t, &asked)
	defer srv.Close()
	// The token bucket is not what is under test here; the cap is.
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost-test", RatePerSec: 500})
	if err != nil {
		t.Fatal(err)
	}
	s := New(c, srv.URL)

	ids := make([]string, maxAtOnce+40)
	for i := range ids {
		ids[i] = "INZ" + string(rune('a'+i/676%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i%26))
	}
	got, missing := s.Zones(context.Background(), ids)
	if len(got) > maxAtOnce {
		t.Errorf("%d zones were fetched; the cap is %d", len(got), maxAtOnce)
	}
	if len(missing) < 40 {
		t.Errorf("%d ids were reported missing; the %d past the cap must be reported, not dropped",
			len(missing), len(ids)-maxAtOnce)
	}
	if len(asked) > maxAtOnce {
		t.Errorf("the service was asked %d times; the cap is %d", len(asked), maxAtOnce)
	}
}

// TestAZonesNameIsBoundedLikeEveryOtherStringFromOutside. The name arrives in
// the same answer as the shape and was the one field in this release that was
// never clamped: the transport admits 32 MiB, and the store keeps two thousand
// shapes, so an unbounded name is tens of gigabytes of a name.
func TestAZonesNameIsBoundedLikeEveryOtherStringFromOutside(t *testing.T) {
	huge := strings.Repeat("a", 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		_, _ = w.Write([]byte(`{"properties":{"id":"INZ027","name":"` + huge + `"},` +
			`"geometry":{"type":"Polygon","coordinates":[[[-85,41],[-84,41],[-84,42],[-85,41]]]}}`))
	}))
	defer srv.Close()
	s := newStore(t, srv.URL)

	z, err := s.Zone(context.Background(), "INZ027")
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(z.Name)) > plaintext.MaxFieldRunes {
		t.Errorf("a name of %d runes was kept whole; the bound is %d",
			len([]rune(z.Name)), plaintext.MaxFieldRunes)
	}
}

// TestAPanicFetchingOneZoneDoesNotEndTheProgram.
//
// **A recover only catches panics in its own goroutine.** The guard for this
// was written in the caller, while the fetches happen in children of it, so it
// had never once fired - proved by handing the store a client that is not
// there, which killed the test binary outright. The guard is inside the
// goroutine that can panic now, and the zone it was fetching is reported
// missing like any other it could not get.
func TestAPanicFetchingOneZoneDoesNotEndTheProgram(t *testing.T) {
	s := New(nil, "http://127.0.0.1:1") // no client at all
	got, missing := s.Zones(context.Background(), []string{"INZ027", "INC003"})
	if len(got) != 0 {
		t.Errorf("%d zones came back from a store with no client", len(got))
	}
	if len(missing) != 2 {
		t.Errorf("%d ids reported missing; both were asked for and neither could be got", len(missing))
	}
	if s.Stats().Failed == 0 {
		t.Error("nothing was counted as failed; a fetch that panics is a fetch that failed")
	}
}
