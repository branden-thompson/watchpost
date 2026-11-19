package allocpin

import (
	"fmt"
	"testing"
)

// sink keeps the planted allocations on the heap.
var sink []byte

// TestPinCountsAsAllocsPerRunCounts holds Pin to the figure
// testing.AllocsPerRun gives for the same path, and its bytes to the size the
// path asks for, so the pins that move onto it keep their counts.
func TestPinCountsAsAllocsPerRunCounts(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	f := func() { sink = make([]byte, 4096) }
	r := &recorder{TB: t}
	got := Pin(r, "a slice", 50, f)
	if want := testing.AllocsPerRun(50, f); got != want || got != 1 {
		t.Errorf("Pin counts %.0f allocations; AllocsPerRun counts %.0f and the path makes 1", got, want)
	}
	if len(r.logs) != 1 || r.logs[0] != "pin a slice: 1 allocs, 4096 bytes" {
		t.Errorf("Pin reported %q; one 4096-byte slice is 1 allocation of 4096 bytes", r.logs)
	}
}

// TestMeasureEachCountsOnlyThePath holds MeasureEach to the path alone: the
// setup's allocations stay outside the window.
func TestMeasureEachCountsOnlyThePath(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	got := MeasureEach(t, 20, func() func() {
		sink = make([]byte, 1<<16) // the fixture, not measured
		return func() { sink = make([]byte, 512) }
	})
	if got != (Cost{Allocs: 1, Bytes: 512}) {
		t.Errorf("MeasureEach counts %+v; the path makes 1 allocation of 512 bytes", got)
	}
}

// TestPinBenchmarkIsTheResultPerOp holds PinBenchmark to the result's own
// AllocsPerOp and AllocedBytesPerOp.
func TestPinBenchmarkIsTheResultPerOp(t *testing.T) {
	r := testing.BenchmarkResult{N: 4, MemAllocs: 12, MemBytes: 400}
	if got := PinBenchmark(&recorder{TB: t}, "a benchmark", r); got != (Cost{Allocs: 3, Bytes: 100}) {
		t.Errorf("PinBenchmark = %+v; the result's per-op figures are 3 and 100", got)
	}
}

// recorder is a testing.TB that keeps what was logged and whether the test
// was failed, so the refusals can be seen.
type recorder struct {
	testing.TB
	logs   []string
	failed bool
}

func (r *recorder) Helper()                 {}
func (r *recorder) Logf(f string, a ...any) { r.logs = append(r.logs, fmt.Sprintf(f, a...)) }
func (r *recorder) Fatal(...any)            { r.failed = true }

// TestReportWritesTheComparatorsLine holds the line's form, which tools/pins
// parses, and the refusal of a label the line could not carry.
func TestReportWritesTheComparatorsLine(t *testing.T) {
	r := &recorder{TB: t}
	Report(r, "beat idle Observer 133x44", Cost{Allocs: 168, Bytes: 75988})
	if len(r.logs) != 1 || r.logs[0] != "pin beat idle Observer 133x44: 168 allocs, 75988 bytes" {
		t.Errorf("Report logged %q", r.logs)
	}
	for _, label := range []string{"", "frame: hit", "two\nlines"} { // the three refusals (P10-02)
		r := &recorder{TB: t}
		Report(r, label, Cost{})
		if !r.failed {
			t.Errorf("Report took the label %q, which the comparator's line cannot carry", label)
		}
	}
}

// TestReportCountSaysWhyTheBytesAreNotHeld holds the count-only line's form
// and its refusals: a pin that drops its bytes says why.
func TestReportCountSaysWhyTheBytesAreNotHeld(t *testing.T) {
	r := &recorder{TB: t}
	ReportCount(r, "time to tone", 185, "they follow the wall clock")
	if len(r.logs) != 1 || r.logs[0] != "pin time to tone: 185 allocs (bytes not held, they follow the wall clock)" {
		t.Errorf("ReportCount logged %q", r.logs)
	}
	for _, c := range [][2]string{{"time to tone", ""}, {"time to tone", "a: b"}, {"", "why"}} { // the refusals (P10-02)
		r := &recorder{TB: t}
		ReportCount(r, c[0], 1, c[1])
		if !r.failed {
			t.Errorf("ReportCount took label %q and reason %q", c[0], c[1])
		}
	}
}
