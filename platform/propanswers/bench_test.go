package propanswers

import (
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// BenchmarkTheChartIsWorkedOut is FR-1.17's figure (BUILD red team P-4): the
// chart for the origin's area and the six continents, the typical day of
// every band included, as the window asks it off the UI goroutine.
func BenchmarkTheChartIsWorkedOut(b *testing.B) {
	snap := snapAt(time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC), 18)
	ask := Ask{Origin: ionomaps.LatLon{Lat: 33.2, Lon: -117.4}, RadiusKm: 400, Targets: Continents}
	for b.Loop() {
		if _, err := WorkChart(snap, ask); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTheCentreIsWorkedOut is FR-1.17's figure for the centre: a path
// from the origin to a point 3,000 km away.
func BenchmarkTheCentreIsWorkedOut(b *testing.B) {
	snap := snapAt(time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC), 18)
	origin, centre := ionomaps.LatLon{Lat: 33.2, Lon: -117.4}, ionomaps.LatLon{Lat: 40.7, Lon: -74}
	for b.Loop() {
		if _, err := WorkCentre(snap, origin, centre, 400); err != nil {
			b.Fatal(err)
		}
	}
}
