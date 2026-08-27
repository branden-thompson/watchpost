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
	if p.CachedGrids() != 0 || len(p.grids) != 0 {
		t.Fatal("an empty set empties the cache and the grid memo")
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
		if _, err := p.gridExtremes(context.Background(), g.gridURL); err != nil {
			t.Fatal(err)
		}
	}
	if p.GridDecodes() != 1 {
		t.Fatalf("three reads of one body: one decode, got %d", p.GridDecodes())
	}
	p.Retain(nil)
	if len(p.grids) != 0 {
		t.Fatal("Retain prunes the memo with the cache")
	}
}
