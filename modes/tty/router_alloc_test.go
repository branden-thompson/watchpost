package tty

// router_alloc_test.go — what the Router costs a frame on top of the surface it
// draws (NFR-1).

import (
	"testing"

	"github.com/branden-thompson/watchpost/internal/allocpin"
)

// routerOverheadAllocs is what the Router's delegation costs Observer per
// frame. MEASURED 2026-09-09: Observer direct 218, through the Router 219.
//
// PINNED AT THE MEASUREMENT, not at a comfortable margin. A second
// allocation appearing here is a real change in the hot path and the
// operator's frame is drawn on every tick.
const routerOverheadAllocs = 1

func TestRouterCostsObserverAlmostNothingPerFrame(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := goldenDash(t, false)
	_ = d.View().Content // warm the memo and the theme
	direct := allocpin.Pin(t, "Observer frame direct", 50, func() { _ = d.View().Content })

	r := NewRouter(d)
	_ = r.View().Content
	via := allocpin.Pin(t, "Observer frame through the Router", 50, func() { _ = r.View().Content })

	t.Logf("observer direct %.0f allocs · through the router %.0f · overhead %.0f (budget %d)",
		direct, via, via-direct, routerOverheadAllocs)
	if via-direct > routerOverheadAllocs {
		t.Errorf("the Router costs Observer %.0f allocations per frame, budget %d — "+
			"NFR-1 says Observer does not regress, and the frame is drawn on every tick",
			via-direct, routerOverheadAllocs)
	}
}
