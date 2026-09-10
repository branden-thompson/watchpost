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
	// MEASURED 2026-09-09: 13 at P1(e); RE-PINNED to 14 at P2(a).
	//
	// THE +1 IS THE STATION BANNER (stationLine), and it was caught by this
	// gate rather than noticed later — which is the whole point of pinning at
	// the measurement instead of at a comfortable margin.
	//
	// IT WILL MOVE. P2..P5 add lanes, state and controls; each re-pins this
	// DELIBERATELY, with the reason in the commit, the way the goldens are
	// re-recorded. A budget nobody re-pins is a budget nobody reads.
	//
	// RE-PINNED TO 65 AT P4 (D-52), AND THIS ONE IS NOT A ROUNDING. The card row
	// became a go-studs `data_table_row`, which costs ~25 ALLOCATIONS PER CARD
	// against the hand-rolled row's effectively zero. Measured, not estimated:
	//
	//	cards   0    1    2    5   10
	//	allocs 14   40   65  141  266
	//
	// So the FIXED cost is unchanged at 14 and the whole regression is per-card
	// and linear. A full ten-card line-up is ~266.
	//
	// IT IS ACCEPTED RATHER THAN OPTIMISED, and the standing rules decide that
	// rather than my judgement: any table is a go-studs table, and a dependency
	// is never re-implemented for speed — patch narrowly, go upstream, or accept
	// and record the cost. This is the third. `docs/accepted-costs.md` carries
	// the entry, and what would re-open it.
	//
	// WHAT THE COMPONENT BUYS IS THE DEFECT IT MAKES IMPOSSIBLE. It sizes the
	// fill column with the badge's width ALREADY RESERVED, so a centred title
	// cannot run into the badge — which the hand-rolled draft did, silently,
	// producing "…(COASTAL)D•".
	//
	// Building the row once per FRAME rather than once per card was tried and
	// saved 5 of the 51: the cost is inside RenderRow, not construction.
	// RE-PINNED TO 72 AT D-55: the lane now builds a SECOND row, with a leading
	// non-truncatable column for the fabricated-event mark. Two rows rather than
	// one because a fixed column present on every card would reserve its width
	// on every card — every real hazard would sit 14 cells off-centre to make
	// room for a label it never carries.
	//
	// +7 FOR THE WHOLE FRAME, not per card: the second row is built once in
	// `newCardLane`, and the per-card cost is unchanged.
	//
	// NOT OPTIMISED, ON THE HUM LEAD'S STANDING RULING (D-53): "we never over
	// optimize on theoretical software … Attempting to do that now would not
	// only be too soon, but may corner us and make intended functionality
	// hard-to-impossible." Measure it, record it, surface the number, keep
	// building.
	// RE-PINNED TO 151 AT D-58: the frame is now PADDED to the terminal in both
	// dimensions, which is one allocation per line and the whole difference.
	//
	// IT IS A CORRECTNESS FIX, NOT A FEATURE. A ragged frame is invisible in the
	// alt-screen — the rest is simply blank — but `render.Overlay` composites
	// against the BASE's measured size, so a short, narrow frame pinned every
	// window to the top rail and let the composite grow sideways past its own
	// right edge. UAT found it on screen.
	//
	// NOT OPTIMISED, per D-53. Padding a frame line by line is the obvious thing
	// to do better later, and "later" is after the layout is judged good.
	// RE-PINNED TO 164 AT D-59: the masthead. It is a framed box with a title
	// ladder and two inner rows, drawn through the Observer's own BoxTitled —
	// so the cost is the frame the reference has always had, arriving.
	//
	// The layout phase will move this number several more times. Each re-pin
	// says WHAT arrived; none of them optimise, per D-53.
	// 170 AT D-59's CORRECTION: the masthead now draws what the reference draws
	// — the version in the title, the `Updated:` stamp with its freshness tone,
	// and the API summary — instead of the three of those I had dropped.
	bcFrameAllocs = 170
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
