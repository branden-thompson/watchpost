package tty

import (
	"strings"
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

// AND WITH A TEST EVENT ON THE TAPE (FR-4.4), against the same budget.
//
// THE EXISTING PIN CANNOT SEE THIS CHANGE. benchDash's marquee holds no
// fabricated events, so the marking's cost — the per-lane scan and the banner
// row — is invisible to every allocation gate in this package. A budget whose
// fixture cannot reach the new code is not a budget on it.
func TestTickAdvanceAllocBudgetWithATestEventOnTheTape(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := benchDash(t, 133, 44).(Dashboard)
	if len(d.ticker) == 0 {
		t.Fatal("the bench dashboard has no marquee; this would measure advanceTicker's early return")
	}
	d.ticker[0].Test = true
	_ = d.View().Content // warm the theme and the kit's probes
	if !strings.Contains(stripANSITest(d.View().Content), testEventMark) {
		t.Fatal("the fixture's test event is not reaching the band; this measures the unmarked path")
	}
	tick := testing.AllocsPerRun(50, func() { d.advanceTicker() })
	frame := testing.AllocsPerRun(20, func() { d.memo.ok = false; _ = d.View().Content })
	t.Logf("with a test event: advanceTicker %.0f (budget %.0f) · frame miss %.0f (budget %.0f)",
		tick, tickAllocBudget, frame, frameAllocBudget["133x44"])
	if tick > tickAllocBudget {
		t.Errorf("advanceTicker allocates %.0f per tick with a marked event, budget %.0f", tick, tickAllocBudget)
	}
	if frame > frameAllocBudget["133x44"] {
		t.Errorf("the marked frame allocates %.0f per View(), budget %.0f — the marking is not free "+
			"and this is the pin that says how much it costs", frame, frameAllocBudget["133x44"])
	}
}
