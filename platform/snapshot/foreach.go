package snapshot

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"
)

// FetchEach runs fn over every location with bounded concurrency (B3 UAT
// 59/64). Failures never abort the batch: successes land in the map and
// every failure travels, joined, in the returned error (§10.1 partial
// failure). Concurrency also lets a provider reserve all of a batch's
// pacing slots at once, so a fast-cadence pipeline is never starved behind
// a slow one sharing the same client.
// IT RETURNS WHICH LOCATION EACH FAILURE BELONGS TO, and joining them was the
// defect (REVIEW red team, 2026-09-08). A joined error says "something failed"
// and nothing about WHERE, so the assembler could not tell a location the feed
// does not cover — a definitive 404 — from one it simply could not reach. It
// guessed from whether ANY location was served, which is right for a total
// outage and wrong for a partial one.
func FetchEach(ctx context.Context, refs []LocationRef, limit int,
	fn func(context.Context, LocationRef) (PartialData, error)) (map[LocationKey]PartialData, map[LocationKey]error, error) {
	var mu sync.Mutex
	out := make(map[LocationKey]PartialData, len(refs))
	failed := map[LocationKey]error{}
	var errs []error
	var g errgroup.Group
	g.SetLimit(max(1, limit))
	for _, ref := range refs {
		g.Go(func() error {
			pd, err := fn(ctx, ref)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				failed[Key(ref)] = err
				return nil
			}
			out[Key(ref)] = pd
			return nil
		})
	}
	_ = g.Wait() // workers never return errors; they are collected above
	return out, failed, errors.Join(errs...)
}

// Unreachable reports whether an error means WE COULD NOT ASK, as opposed to
// the server answering with something we did not want.
//
// The distinction decides whether an empty result is a FACT. A 404 for a point
// outside the forecast area is the service answering "we do not cover you", and
// a row saying "n/a" is then true. A refused connection is not an answer, and a
// row saying "n/a" would be a false claim during an outage.
//
// DUCK-TYPED ON PURPOSE. platform/httpx's StatusError returns its code and its
// ReachError returns 0, and both satisfy this without snapshot importing httpx —
// which would invert the dependency and put transport details in the contract
// every renderer reads.
//
// AN ERROR THAT CANNOT BE CLASSIFIED COUNTS AS UNREACHABLE: "we do not know"
// keeps the row waiting, and asserting an absence we cannot support is the one
// outcome this product should never choose.
func Unreachable(err error) bool {
	if err == nil {
		return false
	}
	var s interface{ HTTPStatus() int }
	if errors.As(err, &s) {
		return s.HTTPStatus() == 0
	}
	return true
}
