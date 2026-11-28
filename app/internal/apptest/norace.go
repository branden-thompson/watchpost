//go:build !race

package apptest

// RaceSlowdown is 1 without the race detector: a test's timing is held to
// the product bound itself.
const RaceSlowdown = 1

const RaceEnabled = false
