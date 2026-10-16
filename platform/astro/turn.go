package astro

import (
	"math"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// horizon is the elevation the sun rises and sets at, refraction and its
// disc allowed for, as SunTimes uses.
const horizon = -0.833

// Elevation is the sun's elevation in degrees at a place and moment (NOAA's
// general solar position equations, to about a tenth of a degree).
func Elevation(lat, lon float64, at time.Time) float64 {
	if err := invariant.Check(lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180, "astro: coordinates out of range"); err != nil {
		return math.NaN()
	}
	u := at.UTC()
	hour := float64(u.Hour()) + float64(u.Minute())/60 + float64(u.Second())/3600
	g := 2 * math.Pi / 365 * (float64(u.YearDay()-1) + (hour-12)/24)
	eqTime := 229.18 * (0.000075 + 0.001868*math.Cos(g) - 0.032077*math.Sin(g) - 0.014615*math.Cos(2*g) - 0.040849*math.Sin(2*g))
	decl := 0.006918 - 0.399912*math.Cos(g) + 0.070257*math.Sin(g) - 0.006758*math.Cos(2*g) +
		0.000907*math.Sin(2*g) - 0.002697*math.Cos(3*g) + 0.00148*math.Sin(3*g)
	trueSolar := hour*60 + eqTime + 4*lon // minutes
	ha := rad(trueSolar/4 - 180)
	phi := rad(lat)
	cosZ := math.Sin(phi)*math.Sin(decl) + math.Cos(phi)*math.Cos(decl)*math.Cos(ha)
	return 90 - math.Acos(max(min(cosZ, 1), -1))*180/math.Pi
}

// Up reports whether the sun is above the horizon at a place and moment.
func Up(lat, lon float64, at time.Time) bool { return Elevation(lat, lon, at) > horizon }

// The search for the next turn: a step to find it within a day and a
// half, then halving to the second.
const (
	turnStep   = 2 * time.Minute
	turnSteps  = 36 * 60 / 2
	turnHalves = 7
)

// NextTurn is the next sunrise or sunset after a moment at a place, to the
// second; false where the sun neither rises nor sets within a day and a
// half - the polar day or night.
func NextTurn(lat, lon float64, at time.Time) (time.Time, bool) {
	up := Up(lat, lon, at)
	for i := range turnSteps {
		from, to := at.Add(time.Duration(i)*turnStep), at.Add(time.Duration(i+1)*turnStep)
		if Up(lat, lon, to) == up {
			continue
		}
		for range turnHalves {
			mid := from.Add(to.Sub(from) / 2)
			if Up(lat, lon, mid) == up {
				from = mid
			} else {
				to = mid
			}
		}
		return to.Truncate(time.Second), true
	}
	return time.Time{}, false
}
