package app

// resolve.go — the location resolver and the dashboard's Resolve hook. Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"context"
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// resolveHook adapts the offline-first resolver (embedded geodata, online
// geocoder fallback) for the dashboard's search modal (UAT 26).
func newResolver(client *httpx.Client, idx *geodata.Index, loadErr error) (*locations.Resolver, error) {
	if loadErr != nil || idx == nil {
		return nil, fmt.Errorf("location data unavailable: %w", loadErr)
	}
	r, err := locations.New(idx, openmeteo.NewGeocoder(client, ""))
	if err != nil {
		return nil, fmt.Errorf("resolver unavailable: %w", err)
	}
	return r, nil
}

// resolveHook turns a typed query into a ref; a resolver that failed to
// build answers every query with that reason (actionable, never a panic).
func resolveHook(r *locations.Resolver, buildErr error) func(string) (snapshot.LocationRef, error) {
	if buildErr != nil {
		return func(string) (snapshot.LocationRef, error) { return snapshot.LocationRef{}, buildErr }
	}
	return func(query string) (snapshot.LocationRef, error) {
		rctx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		ref, _, err := r.Resolve(rctx, query)
		if err != nil {
			return snapshot.LocationRef{}, err
		}
		if ref.Tag == "" {
			ref.Tag = deriveTag(ref.Label)
		}
		return ref, nil
	}
}
