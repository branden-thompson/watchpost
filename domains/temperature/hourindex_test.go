package temperature

import (
	"testing"
	"time"
)

// AN HOUR GOES IN ITS PLACE: one added between hours already held lands before
// the first that is later, so the series stays in order, every row beside it.
func TestAnHourGoesInItsPlace(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 10, 3, h, 0, 0, 0, time.UTC) }
	s := Series{Lattice: Lattice{Cols: 1, Rows: 1}, Hours: []time.Time{at(10), at(12), at(13)},
		Hourly: [][]float64{{10}, {12}, {13}}, WindSpeed: [][]float64{{0}, {0}, {0}}, WindFrom: [][]float64{{0}, {0}, {0}},
		Feels: [][]float64{{0}, {0}, {0}}, WindGust: [][]float64{{0}, {0}, {0}}, UV: [][]float64{{0}, {0}, {0}}}
	if i := s.HourIndex(at(11)); i != 1 {
		t.Fatalf("11:00 went in at %d; want 1, between 10:00 and 12:00", i)
	}
	for i, h := range []int{10, 11, 12, 13} {
		if !s.Hours[i].Equal(at(h)) {
			t.Errorf("hour %d is %v; want %02d:00", i, s.Hours[i], h)
		}
	}
	if s.Hourly[2][0] != 12 || len(s.UV) != 4 {
		t.Errorf("the rows did not move with their hours: %v, %d UV rows", s.Hourly, len(s.UV))
	}
	if i := s.HourIndex(at(12)); i != 2 {
		t.Errorf("an hour held is at %d; want 2", i)
	}
}
