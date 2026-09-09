//go:build race

package usgs

// raceEnabled: the allocation pins count mallocs, which the race detector
// distorts (Go's own tree skips them under -race), so they run in the non-race
// CI step (`make alloc-budget`) and skip here. The same two files exist in
// modes/tty, which is where this pattern started — a build-tagged constant
// cannot be shared between packages without shipping a package to hold it.
const raceEnabled = true
