package tty

import "testing"

// NFR-3 — the frame stays within budget, and the budget is a NUMBER.
//
// THE REVIEW'S OBJECTION WAS THAT NFR-3 WAS A PROMISE TO MEASURE, not a
// budget: no threshold meant nothing could fail it. These are the
// measurements, taken at P1 and pinned, so a regression is caught rather than
// discovered.
//
// EVERY NUMBER HERE WAS MEASURED FIRST AND WRITTEN DOWN SECOND. Choosing a
// budget before measuring is how a gate ends up unable to fail.
const (
	// routerOverheadAllocs is what the Router's delegation costs Observer per
	// frame. MEASURED 2026-09-09: Observer direct 218, through the Router 219.
	//
	// PINNED AT THE MEASUREMENT, not at a comfortable margin. A second
	// allocation appearing here is a real change in the hot path and the
	// operator's frame is drawn on every tick.
	routerOverheadAllocs = 1

	// bcFrameAllocs is the console's own frame cost at 150x74 with two cards.
	// MEASURED 2026-09-09: 13.
	//
	// IT WILL MOVE. P2..P5 add lanes, state and controls; each re-pins this
	// DELIBERATELY, with the reason in the commit, the way the goldens are
	// re-recorded. A budget nobody re-pins is a budget nobody reads.
	bcFrameAllocs = 13
)

func TestRouterCostsObserverAlmostNothingPerFrame(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := goldenDash(t, false)
	_ = d.View().Content // warm the memo and the theme
	direct := testing.AllocsPerRun(50, func() { _ = d.View().Content })

	r := NewRouter(d)
	_ = r.View().Content
	via := testing.AllocsPerRun(50, func() { _ = r.View().Content })

	t.Logf("observer direct %.0f allocs · through the router %.0f · overhead %.0f (budget %d)",
		direct, via, via-direct, routerOverheadAllocs)
	if via-direct > routerOverheadAllocs {
		t.Errorf("the Router costs Observer %.0f allocations per frame, budget %d — "+
			"NFR-1 says Observer does not regress, and the frame is drawn on every tick",
			via-direct, routerOverheadAllocs)
	}
}

func TestConsoleFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	b := bcWith(t, card(t, "a", "OCEANSIDE"), card(t, "b", "BONSALL"))
	_ = b.View().Content
	got := testing.AllocsPerRun(50, func() { _ = b.View().Content })
	t.Logf("console frame 150x74, two cards: %.0f allocs (budget %d)", got, bcFrameAllocs)
	if got > bcFrameAllocs {
		t.Errorf("the console frame allocates %.0f per View(), budget %d — re-pin DELIBERATELY "+
			"with the reason in the commit, or this is a regression", got, bcFrameAllocs)
	}
}
