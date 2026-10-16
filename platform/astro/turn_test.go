package astro

import (
	"testing"
	"time"
)

// TestTheSunIsUpByDayAndDownByNight is 0.19.0 FR-3.1 (AX-E): the sun's
// elevation at a place is above the horizon at its local noon and below it
// at its local midnight, and near the zenith under the sun.
func TestTheSunIsUpByDayAndDownByNight(t *testing.T) {
	noon := time.Date(2026, 10, 11, 19, 50, 0, 0, time.UTC) // about solar noon at Oceanside
	if e := Elevation(33.2, -117.38, noon); e < 40 || e > 55 {
		t.Errorf("Oceanside at noon in October: elevation %.1f; about 49", e)
	}
	if e := Elevation(33.2, -117.38, noon.Add(-12*time.Hour)); e > -40 {
		t.Errorf("Oceanside at midnight: elevation %.1f; far below the horizon", e)
	}
	if e := Elevation(-7, 0, time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)); e < 85 {
		t.Errorf("under the October sun at midday: elevation %.1f; near 90", e)
	}
}

// TestTheNextTurnIsTheSunriseOrSunset is FR-3.1 (AX-E): from the night the
// next turn is the sunrise SunTimes gives, from the day the sunset, to a
// couple of minutes; and Up says which side the place is on.
func TestTheNextTurnIsTheSunriseOrSunset(t *testing.T) {
	night := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	rise, set, ok := SunTimes(33.2, -117.38, night.Add(-8*time.Hour), time.UTC) // the local calendar day of 11 October
	if !ok {
		t.Fatal("the sun rises at Oceanside in October")
	}
	near := func(a, b time.Time) bool { return a.Sub(b).Abs() < 3*time.Minute }
	turn, ok := NextTurn(33.2, -117.38, night)
	if !ok || Up(33.2, -117.38, night) || !near(turn, rise) {
		t.Errorf("from 08:00 UTC the next turn is %v (ok %v, up %v); the sunrise is %v", turn, ok, Up(33.2, -117.38, night), rise)
	}
	day := rise.Add(time.Hour)
	turn, ok = NextTurn(33.2, -117.38, day)
	if !ok || !Up(33.2, -117.38, day) || !near(turn, set) {
		t.Errorf("an hour after sunrise the next turn is %v (ok %v); the sunset is %v", turn, ok, set)
	}
}

// TestThePolarNightHasNoTurn is FR-3.1: where the sun neither rises nor sets
// within a day there is no turn to say.
func TestThePolarNightHasNoTurn(t *testing.T) {
	at := time.Date(2026, 12, 21, 12, 0, 0, 0, time.UTC)
	if _, ok := NextTurn(89, 0, at); ok || Up(89, 0, at) {
		t.Errorf("at 89 degrees north at the solstice the sun turned or is up")
	}
	if _, ok := NextTurn(-89, 0, at); ok || !Up(-89, 0, at) {
		t.Errorf("at 89 degrees south at the solstice the sun turned or is down")
	}
}
