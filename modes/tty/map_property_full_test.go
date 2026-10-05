//go:build property

package tty

// W2.4's full count, under the `property` tag: ten thousand sequences take
// about two minutes, which the race run and every mutant's run would repeat,
// so `make property` runs it once, in a step of its own.
func init() { propertyRuns = 10_000 }
