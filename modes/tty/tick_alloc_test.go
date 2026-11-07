package tty

import (
	"strings"
	"testing"
)

// P-1 — THE TICK PATH HAS A BUDGET.
//
// TestFrameAllocBudget measures View() only, so this measures the one per-300ms
// cost in the app: advanceTicker, which keeps the scroll inside the current
// lane's loop length. The length comes from the tape memo (tapeOf), which the
// frame shares, so a tick builds no tape while the ticker, the lane, the glyph
// set, the clock format and the day stand still (0.19.5 PF-8, F-32). A tick
// that rebuilt the tape would cost about seventy allocations.
//
// The budget is the measurement × 1.05, the same rule as every other pin here.
const tickAllocBudget = 3 * 1.05

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
// THE PIN ABOVE CANNOT SEE THE MARKING. benchDash's marquee holds no
// fabricated events, so the marking's cost — the per-lane scan and the banner
// row — is invisible to every other allocation gate in this package. A budget
// whose fixture cannot reach the code is not a budget on it.
func TestTickAdvanceAllocBudgetWithATestEventOnTheTape(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := benchDash(t, 133, 44).(Dashboard)
	if len(d.ticker) == 0 {
		t.Fatal("the bench dashboard has no marquee; this would measure advanceTicker's early return")
	}
	marked := append([]TickerItem(nil), d.ticker...)
	marked[0].Test = true
	d.setTicker(marked)  // through its one writer, so the tape memo sees the change (0.19.5 PF-8)
	_ = d.View().Content // warm the theme and the kit's probes
	if !strings.Contains(stripANSITest(d.View().Content), testEventMark) {
		t.Fatal("the fixture's test event is not reaching the band; this measures the unmarked path")
	}
	tick := testing.AllocsPerRun(50, func() { d.advanceTicker() })
	frame := testing.AllocsPerRun(20, func() { missEveryMemo(d); _ = d.View().Content })
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
