package zones

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
		w.Write([]byte(`{"properties":{"id":"` + id + `","name":"Dallas"},"geometry":` + string(dallas) + `}`))
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
	if hits.Load() != 3 {
		t.Errorf("the service was asked %d times for three distinct ids", hits.Load())
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
