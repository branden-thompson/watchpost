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
func FetchEach(ctx context.Context, refs []LocationRef, limit int,
	fn func(context.Context, LocationRef) (PartialData, error)) (map[LocationKey]PartialData, error) {
	var mu sync.Mutex
	out := make(map[LocationKey]PartialData, len(refs))
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
				return nil
			}
			out[Key(ref)] = pd
			return nil
		})
	}
	_ = g.Wait() // workers never return errors; they are collected above
	return out, errors.Join(errs...)
}
