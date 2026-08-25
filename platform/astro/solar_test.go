package astro

import (
	"testing"
	"time"
)

// Spec: B3 UAT 32 — sunrise/sunset from geometry. Reference values from the
// NOAA solar calculator: Oceanside, CA (33.2, -117.38) on 2026-08-24 rises
// ~06:17 and sets ~19:25 PDT. Tolerance ±5 min covers refraction/altitude
// modelling differences and the reference's own minute rounding.

func TestSunTimesOceansideLateAugust(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2026, 8, 24, 12, 0, 0, 0, loc)
	rise, set, ok := SunTimes(33.2, -117.38, date, loc)
	if !ok {
		t.Fatal("sun rises in Oceanside in August")
	}
	within := func(got time.Time, h, m int) bool {
		want := time.Date(2026, 8, 24, h, m, 0, 0, loc)
		d := got.Sub(want)
		return d > -5*time.Minute && d < 5*time.Minute
	}
	if !within(rise, 6, 17) {
		t.Fatalf("sunrise %v, want ~06:17 PDT", rise)
	}
	if !within(set, 19, 25) {
		t.Fatalf("sunset %v, want ~19:25 PDT", set)
	}
	if rise.Location() != loc || set.Location() != loc {
		t.Fatal("times must be returned in the requested location")
	}
}

func TestSunTimesPolarNight(t *testing.T) {
	// Longyearbyen in late December: the sun never rises.
	loc := time.UTC
	if _, _, ok := SunTimes(78.2, 15.6, time.Date(2026, 12, 21, 12, 0, 0, 0, loc), loc); ok {
		t.Fatal("polar night must report ok=false")
	}
}

func TestSunTimesRefusesBadInput(t *testing.T) {
	if _, _, ok := SunTimes(95, 0, time.Now(), time.UTC); ok {
		t.Fatal("latitude out of range must fail closed")
	}
	if _, _, ok := SunTimes(0, 0, time.Time{}, time.UTC); ok {
		t.Fatal("zero date must fail closed")
	}
}
