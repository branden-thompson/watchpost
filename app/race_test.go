//go:build race

package app

// raceSlowdown is how much longer work takes under the race detector, for
// the timings a test holds to a product bound: a burst cut at the app's end
// built its field in 60 ms, 405 ms under race on the developer's machine,
// and past 2 s under race on a hosted runner. The product bound itself is
// held in the legs without -race, where the mutants run.
const raceSlowdown = 10
