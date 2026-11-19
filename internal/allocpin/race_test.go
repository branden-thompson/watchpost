//go:build race

package allocpin

// raceEnabled: the race detector adds allocations of its own, so the counts
// are held without it (make alloc-budget) and skip here.
const raceEnabled = true
