package frame

// tick.go — the shimmer's clock both surfaces run on: the tick message, the
// one way a clock is made, and the call that arms it (0.19.6 A-21).

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// TickMsg drives the loading shimmer (UAT 18.2b) and (Q3) every
// other wall-clock element of the frame: the marquee (when the visualizer
// tick is not already redrawing), the volume blink's clearing, the [S]
// ages and the Details labels. It runs only while one of them is showing
// (plan §2.5 tick predicate: tickNeeded).
type TickMsg struct{}

// TickEvery is the one way this model makes a clock (metric D, 2026-09-08).
// TWO CLOCKS ARE CORRECT — 300 ms for the shimmer and ages, 50 ms for the
// visualizer's bars, which the shimmer tick is far too slow to draw — so this
// collapses how one is BUILT, not how many there are.
func TickEvery(d time.Duration, m tea.Msg) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return m })
}

// ShimmerTick is the shimmer's 300 ms clock, made the one way TickEvery
// makes one (0.19.6 D-59).
func ShimmerTick() tea.Cmd { return TickEvery(300*time.Millisecond, TickMsg{}) }

// ArmShimmer is THE shimmer-tick arming rule, for whichever surface asks
// (0.16.0 D-56).
//
// WHAT DIFFERS BETWEEN THE SURFACES IS THE PREDICATE, NOT THE ARMING. Observer
// animates while something is loading; the console animates while a slot is
// still waiting on the Director. Those are two questions with one answer about
// what to do with the answer — so the question stays with each surface and this
// is called with it.
func ArmShimmer(armed, needed bool, cmd, next tea.Cmd) (bool, tea.Cmd) {
	if armed || !needed {
		return armed, cmd
	}
	if cmd == nil {
		return true, next
	}
	return true, tea.Batch(cmd, next)
}
