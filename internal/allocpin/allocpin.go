package allocpin

import (
	"math"
	"runtime"
	"runtime/debug"
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

// windows is how many times a pin's window is measured; the least is kept.
//
// ANOTHER GOROUTINE ALLOCATES INSIDE THE WINDOW. GOMAXPROCS 1 slows the rest of
// the test binary down; it does not stop it, and the counters are the whole
// process's. A timer or a program a test before this one left running adds a
// few allocations: under the commit gate, "layout alone" read 4,543 bytes where
// it reads 4,536 on its own, 7 bytes over 20 runs, past the comparator's 0.1 %.
// Such an allocation only ever adds, so the least of three windows is the
// path's own cost, and a path's real change is in every one of them.
//
// AND THE COLLECTOR IS OFF INSIDE THE WINDOWS, after one collection to settle.
// A collection empties every sync.Pool, and a pooled buffer the path then
// takes is a new allocation: the map, severe and Propagation windows read one
// allocation more in some runs and not others, and time to tone 183 or 185,
// by when a collection fell. With the collector off, runs repeat to the byte.
const windows = 3

// Pin is testing.AllocsPerRun, reported: f runs once to warm, then runs times
// between two reads of the runtime's memory statistics, at GOMAXPROCS 1 and
// with the collector off, in each of three windows, the least count and the
// least bytes kept. It reports
// them under label and returns the count as AllocsPerRun's figure, so a pin's
// own budget check reads it unchanged.
func Pin(tb testing.TB, label string, runs int, f func()) float64 {
	tb.Helper()
	if err := invariant.Check(runs > 0, "a pin measures at least one run"); err != nil {
		tb.Fatal(err)
	}
	if err := invariant.Check(f != nil, "a pin measures a path"); err != nil {
		tb.Fatal(err)
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	runtime.GC()
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	f()
	c := Cost{Allocs: math.MaxUint64, Bytes: math.MaxUint64}
	for range windows { // bounded by windows (P10-02)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range runs { // bounded by runs (P10-02)
			f()
		}
		runtime.ReadMemStats(&after)
		c.Allocs = min(c.Allocs, (after.Mallocs-before.Mallocs)/uint64(runs))
		c.Bytes = min(c.Bytes, (after.TotalAlloc-before.TotalAlloc)/uint64(runs))
	}
	Report(tb, label, c)
	return float64(c.Allocs)
}

// MeasureEach is the cost of a path that cannot run twice on one fixture, or
// whose fixture must be fresh for each run: each run calls setup, unmeasured,
// for the path, and only that path runs inside the window. Nothing runs to
// warm, so the first window carries any one-time cost. As Pin does, it
// measures three windows of runs and keeps the least. It reports nothing: the
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
	runtime.GC()
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	c := Cost{Allocs: math.MaxUint64, Bytes: math.MaxUint64}
	for range windows { // bounded by windows (P10-02)
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
		c.Allocs = min(c.Allocs, total.Allocs/uint64(runs))
		c.Bytes = min(c.Bytes, total.Bytes/uint64(runs))
	}
	return c
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
