//go:build !race

package app

// raceSlowdown is 1 without the race detector: a test's timing is held to
// the product bound itself.
const raceSlowdown = 1
