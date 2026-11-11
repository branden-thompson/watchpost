package snapshot

import (
	"fmt"
	"testing"
	"time"
)

// assembler25 is twenty-five locations, each with an observation and seven
// days of hourly forecast from two providers: a priority pipeline's load.
func assembler25(tb testing.TB) *Assembler {
	tb.Helper()
	refs := make([]LocationRef, 25)
	for i := range refs {
		refs[i] = LocationRef{Label: fmt.Sprintf("L%02d", i), Lat: 30 + float64(i), Lon: -120 + float64(i)}
	}
	a := NewAssembler(refs, []string{"nws", "open-meteo"})
	t0 := time.Now()
	v := 21.5
	hours := make([]Hourly, 168) // 7 days
	for i := range hours {
		hours[i] = Hourly{Time: t0.Add(time.Duration(i) * time.Hour), Temp: &v}
	}
	for _, ref := range refs {
		for _, prov := range []string{"nws", "open-meteo"} {
			a.Apply(Fragment{Provider: prov, Kind: KindObs, FetchedAt: t0,
				PerLocation: map[LocationKey]PartialData{Key(ref): {Current: &Conditions{Temp: &v, Source: SourceInfo{Provider: prov}}}}}, nil)
			a.Apply(Fragment{Provider: prov, Kind: KindForecast, FetchedAt: t0,
				PerLocation: map[LocationKey]PartialData{Key(ref): {Hourly: hours}}}, nil)
		}
	}
	return a
}

// BenchmarkAssembler25Locations is the RS-20 evidence (watchpost-cli architecture §10.8):
// whole-snapshot rebuild cost at the M4 target scale. Run with -benchmem.
func BenchmarkAssembler25Locations(b *testing.B) {
	a := assembler25(b)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		s := a.Snapshot()
		if len(s.Locations) != 25 {
			b.Fatal("bad snapshot")
		}
	}
}

// TestAssemblerPublishAllocBudget is 0.19.5 Q2.12 (A-15, DV-11): what a
// publish costs, in allocations and bytes, measured × 1.05, so the copies
// PF-10 removes cannot come back unseen.
func TestAssemblerPublishAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	a := assembler25(t)
	r := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = a.Snapshot()
		}
	})
	const allocs, bytes = 259 * 1.05, 75_594 * 1.05
	t.Logf("a publish of 25 locations: %d allocs (budget %.0f), %d bytes (budget %.0f)", r.AllocsPerOp(), allocs, r.AllocedBytesPerOp(), bytes)
	if float64(r.AllocsPerOp()) > allocs || float64(r.AllocedBytesPerOp()) > bytes {
		t.Errorf("a publish costs %d allocations and %d bytes; its budget is %.0f and %.0f", r.AllocsPerOp(), r.AllocedBytesPerOp(), allocs, bytes)
	}
}
