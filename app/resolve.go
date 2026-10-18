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

// lookupContext is a lookup's: five seconds, on the interactive lane. The
// launch burst held the geocoder's request behind it on the normal lane, so
// a lookup in the first seconds could spend its whole limit queued (D-156).
func lookupContext() (context.Context, context.CancelFunc) {
	return lookupContextUnder(context.Background())
}

// lookupContextUnder is lookupContext under a caller's context, so the
// caller's cancel stops the lookup too.
func lookupContextUnder(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(httpx.WithInteractive(parent), 5*time.Second)
}

// resolveHook turns a typed query into a ref; a resolver that failed to
// build answers every query with that reason (actionable, never a panic).
func resolveHook(r *locations.Resolver, buildErr error) func(string) (snapshot.LocationRef, error) {
	if buildErr != nil {
		return func(string) (snapshot.LocationRef, error) { return snapshot.LocationRef{}, buildErr }
	}
	return func(query string) (snapshot.LocationRef, error) {
		rctx, done := lookupContext()
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

// findHook is the Propagation mode's place finder (0.19.0 FR-1.10): the
// resolver's lookup anywhere on the globe - local first, online only on a
// miss, uncached - under the lookup's own deadline. Its answer is never
// kept: nothing here saves it.
func findHook(r *locations.Resolver, buildErr error) func(context.Context, string) (snapshot.LocationRef, error) {
	if buildErr != nil {
		return func(context.Context, string) (snapshot.LocationRef, error) { return snapshot.LocationRef{}, buildErr }
	}
	return func(ctx context.Context, query string) (snapshot.LocationRef, error) {
		rctx, done := lookupContextUnder(ctx) // the listener's cancel stops the lookup
		defer done()
		ref, _, err := r.ResolveAnywhere(rctx, query)
		return ref, err
	}
}
