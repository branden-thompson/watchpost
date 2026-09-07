package tty

import (
	"testing"
)

// P-1 — THE TICK PATH HAS A BUDGET NOW.
//
// TestFrameAllocBudget measures View() only, so the one per-300ms cost in the
// app was unmeasured: advanceTicker rebuilds the CURRENT LANE'S ENTIRE TAPE —
// every tapeLine, joined, converted to runes — to obtain a single integer, the
// loop length for `d.tickerScroll %= n`. It then throws the tape away, and the
// renderer builds it again.
//
// THE FIX IS NOT MADE HERE, deliberately. The obvious levers both carry risk
// this gate should not hide: a cached length needs invalidation over items,
// width, units and ascii, and a wrong one wraps the marquee in the wrong place;
// moving the clamp to render time changes what parkScroll stores, which is
// F-16's ruled parked-offset behaviour and a HUM LEAD call. See follow-ups.md
// F-32. What this pin does is stop it getting worse, and make the improvement
// measurable when it is made.
//
// The budget is the measurement × 1.05, the same rule as every other pin here.
const tickAllocBudget = 68 * 1.05

func TestTickAdvanceAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := benchDash(t, 133, 44).(Dashboard)
	_ = d.View().Content // warm the theme and the kit's probes
	// THE FIXTURE MUST ACTUALLY SCROLL, or this measures the early return.
	if len(d.ticker) == 0 {
		t.Fatal("the bench dashboard has no marquee; this would measure advanceTicker's early return")
	}
	got := testing.AllocsPerRun(50, func() { d.advanceTicker() })
	t.Logf("advanceTicker: %.0f allocs per tick (budget %.0f)", got, tickAllocBudget)
	if got > tickAllocBudget {
		t.Errorf("advanceTicker allocates %.0f per 300 ms tick, budget %.0f — this runs around the clock "+
			"whether or not anything is on screen", got, tickAllocBudget)
	}
}
