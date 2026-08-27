package snapshot

import (
	"fmt"
	"math"
	"testing"
)

// Quality pass Q3: Key moved from Sprintf to strconv on the row path; the
// digits must not move with it.
func TestKeyMatchesTheSprintfForm(t *testing.T) {
	vals := []float64{0, math.Copysign(0, -1), 33.19, -117.36, 33.00005, -0.00004, 0.00005, 89.99995, -179.99999, 1e-7, 45.123456789, -45.9999500001}
	for _, lat := range vals {
		for _, lon := range vals {
			want := LocationKey(fmt.Sprintf("%.4f,%.4f", lat, lon))
			if got := Key(LocationRef{Lat: lat, Lon: lon}); got != want {
				t.Fatalf("Key(%v,%v) = %q, want %q", lat, lon, got, want)
			}
		}
	}
}

func BenchmarkKey(b *testing.B) {
	ref := LocationRef{Lat: 33.1959, Lon: -117.3795}
	b.ReportAllocs()
	for b.Loop() {
		_ = Key(ref)
	}
}
