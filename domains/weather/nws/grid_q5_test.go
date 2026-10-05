package nws

// Quality pass Q5 (Q5b-6, Q5b-7; L4-F7): the grid cache has a lifetime and
// follows the location set; the gridpoint extremes are decoded once per
// body change.

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestGridResolutionExpiresAfterADayAndKeepsThePreferredStation(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	p.now = func() time.Time { return now }
	g1, err := p.resolve(context.Background(), oceanside)
	if err != nil {
		t.Fatal(err)
	}
	p.markPreferred(g1, "KCRQ")
	now = now.Add(23 * time.Hour)
	if g, _ := p.resolve(context.Background(), oceanside); g != g1 {
		t.Fatal("inside a day the cached resolution is served")
	}
	now = now.Add(2 * time.Hour)
	g2, err := p.resolve(context.Background(), oceanside)
	if err != nil || g2 == g1 {
		t.Fatalf("past gridTTL the point is resolved again: %v same=%v", err, g2 == g1)
	}
	if g2.preferred != "KCRQ" || !g2.resolvedAt.Equal(now) {
		t.Fatalf("the preferred station carries over and the stamp moves: %q %v", g2.preferred, g2.resolvedAt)
	}
}

func TestRetainDropsLocationsNoLongerTracked(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	other := snapshot.LocationRef{Label: "Other", Lat: 33.3, Lon: -117.3}
	for _, ref := range []snapshot.LocationRef{oceanside, other} {
		if _, err := p.resolve(context.Background(), ref); err != nil {
			t.Fatal(err)
		}
	}
	if p.CachedGrids() != 2 {
		t.Fatalf("two resolutions cached: %d", p.CachedGrids())
	}
	p.Retain([]snapshot.LocationRef{oceanside})
	if p.CachedGrids() != 1 {
		t.Fatalf("the removed location's resolution is gone: %d", p.CachedGrids())
	}
	p.Retain(nil)
	grids, _ := p.grids.Stats()
	if p.CachedGrids() != 0 || grids != 0 {
		t.Fatalf("an empty set empties the cache and the grid memo: %d cached, %d memoised",
			p.CachedGrids(), grids)
	}
}

func TestGridExtremesDecodeOncePerBody(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	g, err := p.resolve(context.Background(), oceanside)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := p.gridDocument(context.Background(), g.gridURL); err != nil {
			t.Fatal(err)
		}
	}
	if p.GridDecodes() != 1 {
		t.Fatalf("three reads of one body: one decode, got %d", p.GridDecodes())
	}
	p.Retain(nil)
	if n, _ := p.grids.Stats(); n != 0 {
		t.Fatalf("Retain prunes the memo with the cache: %d entries survived", n)
	}
}

// THE MARINE READ AND THE DAILY FILL SHARE ONE DECODE (W14 P-13): both read
// the same gridpoint body - up to about 1 MB - and the marine read decoded it
// again in full every 30 minutes. One memoized decode per body serves both.
func TestTheMarineReadAndTheFillShareOneDecode(t *testing.T) {
	srv, _ := testServer(t)
	p := newProvider(t, srv.URL)
	g, err := p.resolve(context.Background(), oceanside)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.gridDocument(context.Background(), g.gridURL); err != nil {
		t.Fatal(err)
	}
	m := NewMarine(p)
	frag, err := m.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindMarine, Locations: []snapshot.LocationRef{oceanside}})
	if err != nil || frag.Err != nil {
		t.Fatalf("marine fetch: %v / %v", err, frag.Err)
	}
	if mar := frag.PerLocation[snapshot.Key(oceanside)].Marine; mar == nil || mar.SwellHeight == nil {
		t.Fatalf("the marine read lost its series: %+v", mar)
	}
	if n := p.GridDecodes(); n != 1 {
		t.Errorf("the fill and the marine read of one body decoded it %d times; want once", n)
	}
}
