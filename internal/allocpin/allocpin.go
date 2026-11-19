package allocpin

import (
	"runtime"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Cost is what one run of a pinned path allocates: the count of allocations
// and the bytes they take, each the total over the runs divided by the runs,
// truncated, as testing.AllocsPerRun divides.
type Cost struct {
	Allocs, Bytes uint64
}

// Pin is testing.AllocsPerRun, reported: f runs once to warm, then runs times
// between two reads of the runtime's memory statistics, at GOMAXPROCS 1 so no
// other goroutine allocates in the window. It reports the count and the bytes
// under label and returns the count as AllocsPerRun does, the same figure for
// the same f and runs, so a pin's own budget check reads it unchanged.
func Pin(tb testing.TB, label string, runs int, f func()) float64 {
	tb.Helper()
	if err := invariant.Check(runs > 0, "a pin measures at least one run"); err != nil {
		tb.Fatal(err)
	}
	if err := invariant.Check(f != nil, "a pin measures a path"); err != nil {
		tb.Fatal(err)
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	f()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs { // bounded by runs (P10-02)
		f()
	}
	runtime.ReadMemStats(&after)
	c := Cost{Allocs: (after.Mallocs - before.Mallocs) / uint64(runs), Bytes: (after.TotalAlloc - before.TotalAlloc) / uint64(runs)}
	Report(tb, label, c)
	return float64(c.Allocs)
}

// MeasureEach is the cost of a path that cannot run twice on one fixture, or
// whose fixture must be fresh for each run: each run calls setup, unmeasured,
// for the path, and only that path runs inside the window. Nothing runs to
// warm, so the first run carries any one-time cost. It reports nothing: the
// caller reports the cost, whole or by its count (ReportCount).
func MeasureEach(tb testing.TB, runs int, setup func() func()) Cost {
	tb.Helper()
	if err := invariant.Check(runs > 0, "a pin measures at least one run"); err != nil {
		tb.Fatal(err)
	}
	if err := invariant.Check(setup != nil, "a pin measures a path"); err != nil {
		tb.Fatal(err)
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	var total Cost
	for range runs { // bounded by runs (P10-02)
		f := setup()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		f()
		runtime.ReadMemStats(&after)
		total.Allocs += after.Mallocs - before.Mallocs
		total.Bytes += after.TotalAlloc - before.TotalAlloc
	}
	return Cost{Allocs: total.Allocs / uint64(runs), Bytes: total.Bytes / uint64(runs)}
}

// PinBenchmark reports a testing.Benchmark result's cost per op under label:
// the figures AllocsPerOp and AllocedBytesPerOp give.
func PinBenchmark(tb testing.TB, label string, r testing.BenchmarkResult) Cost {
	tb.Helper()
	if err := invariant.Check(r.N > 0, "the benchmark ran no iteration: it measured nothing"); err != nil {
		tb.Fatal(err)
	}
	consistent := r.MemAllocs > 0 || r.MemBytes == 0
	if err := invariant.Check(consistent, "the benchmark counted bytes with no allocation"); err != nil {
		tb.Fatal(err)
	}
	c := Cost{Allocs: r.MemAllocs / uint64(r.N), Bytes: r.MemBytes / uint64(r.N)}
	Report(tb, label, c)
	return c
}

// Report logs c under label in the one line the pin comparator reads
// (tools/pins): "pin <label>: <n> allocs, <n> bytes". The label names the pin
// across the tree, so no package or test name is part of it and a move leaves
// it the same; the comparator refuses a label reported twice.
func Report(tb testing.TB, label string, c Cost) {
	tb.Helper()
	if err := invariant.Check(label != "", "a pin is reported under a label"); err != nil {
		tb.Fatal(err)
	}
	plain := !strings.ContainsAny(label, ":\n")
	if err := invariant.Check(plain, "a pin's label holds no colon and no newline, which end it in the comparator's line"); err != nil {
		tb.Fatal(err)
	}
	tb.Logf("pin %s: %d allocs, %d bytes", label, c.Allocs, c.Bytes)
}

// ReportCount logs a pin whose count is held and whose bytes are not, with
// the reason: "pin <label>: <n> allocs (bytes not held, <why>)". The
// comparator holds the count alone, and refuses a pin that changes between
// the two forms.
func ReportCount(tb testing.TB, label string, allocs uint64, why string) {
	tb.Helper()
	plain := label != "" && !strings.ContainsAny(label, ":\n")
	if err := invariant.Check(plain, "a pin's label is given, with no colon and no newline"); err != nil {
		tb.Fatal(err)
	}
	reason := why != "" && !strings.ContainsAny(why, ":()\n")
	if err := invariant.Check(reason, "a pin whose bytes are not held says why, with no colon, bracket or newline"); err != nil {
		tb.Fatal(err)
	}
	tb.Logf("pin %s: %d allocs (bytes not held, %s)", label, allocs, why)
}
