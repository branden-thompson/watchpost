package snapshot

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkAssembler25Locations is the RS-20 evidence (architecture §10.8):
// whole-snapshot rebuild cost at the M4 target scale. Run with -benchmem.
func BenchmarkAssembler25Locations(b *testing.B) {
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
				PerLocation: map[LocationKey]PartialData{Key(ref): {Current: &Conditions{Temp: &v, Source: SourceInfo{Provider: prov}}}}})
			a.Apply(Fragment{Provider: prov, Kind: KindForecast, FetchedAt: t0,
				PerLocation: map[LocationKey]PartialData{Key(ref): {Hourly: hours}}})
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		s := a.Snapshot()
		if len(s.Locations) != 25 {
			b.Fatal("bad snapshot")
		}
	}
}
