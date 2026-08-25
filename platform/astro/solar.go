// Package astro computes sunrise/sunset from position and date (NOAA solar
// equations) — no provider carries them, so the assembler fills every
// Daily row from geometry (B3 UAT 32). Accuracy ~±2 min; polar day/night
// yield zero times.
package astro

import (
	"math"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// SunTimes returns local sunrise and sunset for the calendar date (in loc)
// at lat/lon. ok=false when the sun never rises or never sets that day.
func SunTimes(lat, lon float64, date time.Time, loc *time.Location) (rise, set time.Time, ok bool) {
	if loc == nil {
		loc = time.UTC
	}
	// Out-of-range coordinates would still produce plausible-looking times —
	// refuse them outright (an inland-zip resolve bug would otherwise hide).
	if err := invariant.Check(lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180, "astro: coordinates out of range"); err != nil {
		return time.Time{}, time.Time{}, false
	}
	if err := invariant.Check(!date.IsZero(), "astro: zero date"); err != nil {
		return time.Time{}, time.Time{}, false
	}
	y, m, d := date.In(loc).Date()
	// The equation wants the Julian DAY NUMBER (noon-based integer), i.e. the
	// 0h-UT Julian date + 0.5 — forgetting the half day lands 12h off.
	jdn := julianDay(y, int(m), d) + 0.5
	n := jdn - 2451545.0 + 0.0008
	// Sunrise equation: J* = n - lw/360 with lw the observer longitude,
	// west NEGATIVE (the sign trap: a west-positive lw lands 7h off).
	jStar := n - lon/360
	mAnom := math.Mod(357.5291+0.98560028*jStar, 360)
	mr := rad(mAnom)
	c := 1.9148*math.Sin(mr) + 0.02*math.Sin(2*mr) + 0.0003*math.Sin(3*mr)
	lambda := math.Mod(mAnom+c+180+102.9372, 360)
	lr := rad(lambda)
	jTransit := 2451545.0 + jStar + 0.0053*math.Sin(mr) - 0.0069*math.Sin(2*lr)
	decl := math.Asin(math.Sin(lr) * math.Sin(rad(23.44)))
	phi := rad(lat)
	cosW := (math.Sin(rad(-0.833)) - math.Sin(phi)*math.Sin(decl)) / (math.Cos(phi) * math.Cos(decl))
	if cosW < -1 || cosW > 1 {
		return time.Time{}, time.Time{}, false
	}
	w := math.Acos(cosW) * 180 / math.Pi
	rise, set = fromJulian(jTransit-w/360).In(loc), fromJulian(jTransit+w/360).In(loc)
	if err := invariant.Check(set.After(rise), "astro: sunset must follow sunrise"); err != nil {
		return time.Time{}, time.Time{}, false
	}
	return rise, set, true
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// julianDay is the Julian day number at 0h UT for a Gregorian date.
func julianDay(y, m, d int) float64 {
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(d) + float64(b) - 1524.5
}

// fromJulian converts a Julian date to UTC time.
func fromJulian(jd float64) time.Time {
	if err := invariant.Check(jd > 2440587.5, "astro: Julian date precedes the Unix epoch"); err != nil {
		return time.Time{}
	}
	secs := (jd - 2440587.5) * 86400
	return time.Unix(int64(secs), 0).UTC()
}
