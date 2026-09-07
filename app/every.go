package app

// every.go — the one clock-driven loop in the app.

import (
	"context"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// everyTick runs fn every d until the context ends.
//
// IT EXISTS SO THERE IS ONE EXEMPTION INSTEAD OF ONE PER POLLER. A loop that
// runs until it is cancelled cannot state a bound in iterations — that is what
// a clock is — so each one needed its own ratified P10-02 exemption, and the
// count grew with every new poller. **Zero, one, many: at many, share** (HUM
// LEAD, 2026-09-03). This is the many.
//
// IT WAITS BEFORE IT RUNS. A caller that wants an immediate first pass calls fn
// itself and then this — which is what the release watch does, and it keeps the
// difference at the call site instead of hiding it behind a boolean nobody can
// read from the outside.
//
// The exits are the two a poller has: the context ending, and the ticker
// stopping. Both return; neither leaks.
func everyTick(ctx context.Context, d time.Duration, fn func(time.Time)) {
	if err := invariant.Check(d > 0, "a poller ticks on a positive interval"); err != nil {
		return // a zero or negative interval is a spin, not a schedule
	}
	if err := invariant.Check(fn != nil, "a poller has something to run"); err != nil {
		return
	}
	t := time.NewTicker(d)
	defer t.Stop()
	for { // bounded by the context (P10-02): every exit is a cancel
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			fn(now)
		}
	}
}
