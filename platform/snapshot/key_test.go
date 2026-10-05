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

// TestPlaceIDIsNotAnEmptyZip is #23: a ref's ZIP names its place only when it
// has one; two places without one are two places, and one place is one.
func TestPlaceIDIsNotAnEmptyZip(t *testing.T) {
	park := LocationRef{Label: "Guajome Park, CA", Lat: 33.2472, Lon: -117.2711}
	lake := LocationRef{Label: "Lake Henshaw, CA", Lat: 33.2350, Lon: -116.7600}
	if PlaceID(park) == PlaceID(lake) {
		t.Fatal("two places without a ZIP are two places")
	}
	if PlaceID(park) != PlaceID(LocationRef{Label: "Guajome Regional Park", Lat: 33.24721, Lon: -117.27109}) {
		t.Fatal("one place without a ZIP is one place, whatever it is called")
	}
	if PlaceID(LocationRef{Zip: "92081", Lat: 1}) != PlaceID(LocationRef{Zip: "92081", Lat: 2}) {
		t.Fatal("a ZIP names its place")
	}
}
