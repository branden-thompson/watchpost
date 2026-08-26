//go:build race

package tty

// raceEnabled: the allocation pins count mallocs, which the race detector
// distorts (Go's own tree skips them under -race), so they run in the
// non-race CI step (`make alloc-budget`, red-team R2-8) and skip here.
const raceEnabled = true
