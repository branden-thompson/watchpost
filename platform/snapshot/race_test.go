//go:build race

package snapshot

// raceEnabled: the allocation pins count mallocs, which the race detector
// distorts, so they run in the non-race step (make alloc-budget) and skip here.
const raceEnabled = true
