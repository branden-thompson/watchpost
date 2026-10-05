package temperature

// fuzz_test.go — the upstream answers are read from the network, so each
// reader is fuzzed from the recorded answers: no panic, and what it keeps is
// the package's own - every value missing or within its measure's bound, no
// hour outside the reach of now, every row a value a point.

import (
	"math"
	"testing"
	"time"
)

// maxHours is the most distinct hours a series can hold: a day back to a
// day past the days kept, both ends.
const maxHours = int((reachBack+reachAhead)/time.Hour) + 1

// checkHours fails unless the hours are in order, within reach of now and
// at most maxHours, each with a row a point in every hourly measure.
func checkHours(t *testing.T, hours []time.Time, now time.Time, n int, rows ...[][]float64) {
	t.Helper()
	if len(hours) > maxHours {
		t.Fatalf("%d hours kept; at most %d are in reach", len(hours), maxHours)
	}
	for i, h := range hours {
		if i > 0 && !hours[i-1].Before(h) {
			t.Fatalf("hours out of order: %v then %v", hours[i-1], h)
		}
		if h.Before(now.Add(-reachBack-time.Hour)) || h.After(now.Add(reachAhead)) {
			t.Fatalf("the hour %v is out of reach of %v", h, now)
		}
	}
	for _, measure := range rows {
		if len(measure) != len(hours) {
			t.Fatalf("%d rows for %d hours", len(measure), len(hours))
		}
		for _, row := range measure {
			if len(row) != n {
				t.Fatalf("a row of %d values for %d points", len(row), n)
			}
		}
	}
}

// checkBound fails unless every value of the rows is missing or within b.
func checkBound(t *testing.T, what string, b bound, rows ...[][]float64) {
	t.Helper()
	for _, measure := range rows {
		for _, row := range measure {
			for _, v := range row {
				if !math.IsNaN(v) && !b.holds(v) {
					t.Fatalf("%s: %v kept, outside %v..%v", what, v, b.lo, b.hi)
				}
			}
		}
	}
}

// checkSeries holds a series to the package's invariants.
func checkSeries(t *testing.T, s Series, now time.Time) {
	t.Helper()
	checkHours(t, s.Hours, now, s.Lattice.Cols*s.Lattice.Rows, s.Hourly, s.WindSpeed, s.WindFrom, s.Feels, s.WindGust, s.UV)
	checkBound(t, "temperature", celsiusBound, s.Hourly, s.Feels, days(s.High), days(s.Low), days(s.FeelsHigh), days(s.FeelsLow))
	checkBound(t, "wind", windBound, s.WindSpeed, s.WindGust, days(s.PeakSpeed), days(s.PeakGust))
	checkBound(t, "direction", directionBound, s.WindFrom, days(s.PeakFrom))
	checkBound(t, "UV", uvBound, s.UV, days(s.UVMax))
}

func FuzzParseOpenMeteo(f *testing.F) {
	for _, name := range []string{"openmeteo.json", "openmeteo-wind.json", "openmeteo-feels.json", "openmeteo-gust.json", "openmeteo-uv.json"} {
		f.Add(fixture(f, name))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		s := newSeries(fixtureLattice)
		if err := parseOpenMeteo(body, captured, &s); err != nil {
			return
		}
		checkSeries(t, s, captured)
	})
}

func FuzzParseDWML(f *testing.F) {
	for _, name := range []string{"ndfd-hour.xml", "ndfd-days.xml", "ndfd-days-wind.xml", "ndfd-days-feels.xml",
		"ndfd-hour-feels.xml", "ndfd-days-gust.xml", "ndfd-hour-gust.xml"} {
		f.Add(fixture(f, name))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		s := newSeries(fixtureLattice)
		if err := parseDWML(body, captured, &s); err != nil {
			return
		}
		checkSeries(t, s, captured)
	})
}

func FuzzParseRain(f *testing.F) {
	for _, name := range []string{"openmeteo-rain-days.json", "openmeteo-rain-hours.json"} {
		f.Add(fixture(f, name))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		r := newRain(rainLattice)
		if err := parseRain(body, rainCaptured, &r); err != nil {
			return
		}
		checkHours(t, r.Hours, rainCaptured, rainLattice.Cols*rainLattice.Rows, r.Hourly)
		checkBound(t, "rain", precipBound, r.Hourly, days(r.Peak))
		checkBound(t, "a day's rain and showers", bound{0, 2 * precipBound.hi}, days(r.RainSum))
		checkBound(t, "snowfall", snowBound, days(r.SnowSum))
	})
}
