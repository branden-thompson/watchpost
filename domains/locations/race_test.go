//go:build race

package locations

// raceEnabled: the race detector slows pure CPU work 5–10×, so timing
// budgets measured for the product allow for it (CI's `make verify` runs
// `go test -race` on shared runners).
const raceEnabled = true
