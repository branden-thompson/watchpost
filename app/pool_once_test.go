package app

// pool_once_test.go — W14 P-13 (D-208): a place the station's pool and the
// watchlist share is fetched once, by the priority pipeline; and the pool's
// resolved points survive a commit.

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// A WATCHED PLACE IS NOT FETCHED AGAIN FOR THE POOL (D-208): the priority
// pipeline fetches every watched place; the recent pipeline takes the pool's
// others alone.
func TestAWatchedPlaceInThePoolIsFetchedOnce(t *testing.T) {
	home := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29}
	other := snapshot.LocationRef{Label: "Vista, CA", Zip: "92081", Lat: 33.20, Lon: -117.24}
	lp := poolPipes(t, []snapshot.LocationRef{home, other})
	if err := lp.commit([]snapshot.LocationRef{home}, nil); err != nil {
		t.Fatal(err)
	}
	held := map[snapshot.LocationKey]bool{}
	for _, l := range lp.recent.asm.Snapshot().Locations { // bounded by the list (P10-02)
		held[snapshot.Key(snapshot.LocationRef{Lat: l.Lat, Lon: l.Lon})] = true
	}
	if held[snapshot.Key(home)] {
		t.Error("the watched place is in the recent pipeline too: fetched twice a cycle")
	}
	if !held[snapshot.Key(other)] {
		t.Error("the pool's other place fell out of the recent pipeline")
	}
}

// THE POOL'S RESOLVED POINTS SURVIVE A COMMIT (W14 P-13): Retain keeps every
// place the app fetches - the watchlist, the recent list and the station's
// pool - so a pool place is not looked up again (/points) after every
// lookup, favourite or watchlist edit.
func TestThePoolsPointsSurviveACommit(t *testing.T) {
	var points atomic.Int32
	fixtures := nwsFixtures(t)
	client, base := offlineClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/points/") {
			points.Add(1)
		}
		fixtures.ServeHTTP(w, r)
	}))
	place := refOceanside
	lp := poolPipes(t, []snapshot.LocationRef{place})
	lp.weather = nws.New(client, base)
	lp.weather.ZonesFor(context.Background(), place)
	if points.Load() != 1 {
		t.Fatalf("the pool place was resolved %d times; want once, the premise", points.Load())
	}
	if err := lp.commit(nil, nil); err != nil {
		t.Fatal(err)
	}
	lp.weather.ZonesFor(context.Background(), place)
	if n := points.Load(); n != 1 {
		t.Errorf("the pool place was resolved again after a commit (%d asks): Retain dropped it", n)
	}
}
