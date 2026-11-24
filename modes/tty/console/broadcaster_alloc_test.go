package console

import (
	"testing"

	"github.com/branden-thompson/watchpost/internal/allocpin"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// 0.16.0 NFR-3 — the frame stays within budget, and the budget is a
// NUMBER.
//
// A PROMISE TO MEASURE IS NOT A BUDGET: with no threshold nothing can fail
// it. These are the measurements, pinned, so a regression is caught rather
// than discovered.
//
// EVERY NUMBER HERE IS MEASURED FIRST AND WRITTEN DOWN SECOND. Choosing a
// budget before measuring is how a gate ends up unable to fail.
const (
	// bcFrameAllocs and bcFrameMissAllocs are the console's frame cost at 150x74
	// on a LOADED console (0.16.0 D-120): a pool of locations whose weather the
	// running order and the pool table join. The console memoises its two tables
	// (0.16.0 D-123), so there are TWO PATHS, and a single figure would hide
	// whichever one it did not measure:
	//
	//	bcFrameAllocs      the HIT: the operator is watching, nothing has changed
	//	bcFrameMissAllocs  the MISS: every input moved, both tables rebuild
	//
	// Measured 297 and 9,195 on the loaded fixture. The miss is pinned at x1.05,
	// the same rule as every other number here.
	//
	// THE HIT'S MARGIN IS DELIBERATELY NARROW — 4%, where the miss carries 5% of
	// a number twenty times larger. Putting `locIndex` back OUTSIDE the memo
	// costs 29 allocations a frame, and a wider margin would let that through. A
	// budget whose slack is bigger than the regression it is guarding against is
	// not a budget. THE MISS COSTS WHAT AN UNMEMOISED FRAME COSTS, to the
	// allocation — which is the claim that matters for a cache: it costs nothing
	// on the path it does not help.
	//
	// WHERE THE COST IS. The running order and the location pool are go-studs
	// DataTables (0.16.0 D-94, D-98): any table is a go-studs table, and a
	// dependency is never re-implemented for speed — `docs/accepted-costs.md`
	// carries the entry, and what would re-open it. A table row is assembled cell
	// by cell, and every running-order row joins its location's weather
	// (0.16.0 D-116) — no fetch and no cadence, because `producer()` only offers
	// locations from the pool, whose weather is already on the machine. Neither
	// table builds `DayCell`s, since neither draws them (0.16.0 D-120). The frame
	// is PADDED to the terminal in both dimensions (0.16.0 D-58), because
	// `render.Overlay` composites against the BASE's measured size.
	//
	// THE `locIndex` MAP IS A SHAPE CHOICE, NOT A SAVING: it is O(1) per row where
	// a scan is O(n·m), but at the pool's twenty-five the two timings overlap.
	//
	// NOT OPTIMISED, ON THE HUM LEAD'S STANDING RULING (0.16.0 D-53): the working
	// layout comes first — measure it, record it, surface the number, keep
	// building. A re-pin is DELIBERATE, with the reason in the commit, the way the
	// goldens are re-recorded. A budget nobody re-pins is a budget nobody reads.
	bcFrameAllocs     = 309
	bcFrameMissAllocs = 9655
)

func TestConsoleFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	// A LOADED CONSOLE, WHICH IS THE PATH THAT COSTS
	// (0.16.0 D-120).
	//
	// A console with no pool and no snapshot joins no weather and draws no pool
	// table, so a number taken there guards only the cheap path.
	b := loadedConsole(t, ttytest.LoadedPoolSize)
	// THE BUDGET VALIDATES ITS OWN FIXTURE (0.16.0 D-120). A budget that cannot
	// tell whether it measured the expensive path is a budget reporting on
	// nothing — so it asks, here, before it reports.
	if joined := loadedJoins(b); joined == 0 {
		t.Fatalf("the fixture joined no weather at all; this number is about the cheap path")
	}
	_ = b.View().Content
	got := allocpin.Pin(t, "console frame 150x74 hit", 50, func() { _ = b.View().Content })
	t.Logf("console frame 150x74, %d-location pool: %.0f allocs (budget %d)",
		ttytest.LoadedPoolSize, got, bcFrameAllocs)
	if got > bcFrameAllocs {
		t.Errorf("the console frame allocates %.0f per View(), budget %d — re-pin DELIBERATELY "+
			"with the reason in the commit, or this is a regression", got, bcFrameAllocs)
	}

	// THE MISS IS MEASURED IN THE SAME TEST so the two cannot drift apart. The
	// slot holds ONE entry, so a selection that alternates never finds it — the
	// operator holding an arrow key, which is the honest worst case.
	//
	// A HIT NUMBER ALONE WOULD BE A CACHE MARKING ITS OWN HOMEWORK: it would go
	// on reading under budget while the path underneath it grew without limit,
	// because nothing would ever ask that path what it cost.
	i := 0
	missed := allocpin.Pin(t, "console frame 150x74 miss", 50, func() {
		b.selected = i & 1
		i++
		_ = b.View().Content
	})
	t.Logf("console frame MISS, %d-location pool: %.0f allocs (budget %d)",
		ttytest.LoadedPoolSize, missed, bcFrameMissAllocs)
	if missed <= got {
		t.Fatalf("the miss (%.0f) costs no more than the hit (%.0f): this is not measuring "+
			"the rebuild, and the hit number above is guarding nothing", missed, got)
	}
	if missed > bcFrameMissAllocs {
		t.Errorf("the console frame allocates %.0f per MISSED View(), budget %d — re-pin "+
			"DELIBERATELY with the reason in the commit, or this is a regression",
			missed, bcFrameMissAllocs)
	}
}
