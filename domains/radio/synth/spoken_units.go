package synth

// spoken_units.go — how a quantity is said: temperatures, heights, speeds,
// distances, depths and spans of time, in the listener's units where the
// listener has a choice. One file, so a wording rule ("the listener's units,
// in one spelling", TestTheConditionsSayTheListenersUnits) is kept in one
// place (0.19.5 DR-2).

import (
	"fmt"
	"math"
	"time"

	"github.com/branden-thompson/watchpost/platform/units"
)

func degrees(c float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f degrees", units.FahrenheitOf(c))
	}
	return fmt.Sprintf("%.0f degrees Celsius", c)
}

// heightWords is "3 feet" / "1.5 metres" — one decimal only when it matters.
func heightWords(m *float64, imperial bool) string {
	if m == nil {
		return ""
	}
	if imperial {
		return decimalWords(units.FeetOf(*m), "foot", "feet")
	}
	return decimalWords(*m, "metre", "metres")
}

// tempWords is the water temperature, worded as current conditions word the
// air ("74 degrees", "23 degrees Celsius"): one wording for every spoken
// temperature (0.19.5 D-6).
func tempWords(c *float64, imperial bool) string {
	if c == nil {
		return ""
	}
	return degrees(*c, imperial)
}

// windWords is "13 miles per hour" / "21 kilometers per hour", spelled out
// because "mph" reads as three letters.
func windWords(mps *float64, imperial bool) string {
	if mps == nil {
		return ""
	}
	if imperial {
		return fmt.Sprintf("%.0f miles per hour", units.MphOf(*mps))
	}
	return fmt.Sprintf("%.0f kilometers per hour", units.KmhOf(*mps))
}

// knotWords is "1.4 knots". Currents are ALWAYS knots (MVS-D-21) — the
// mariner's convention, whatever the listener's unit.
func knotWords(mps float64) string { return decimalWords(units.KnotsOf(mps), "knot", "knots") }

// decimalWords reads a number with one decimal only when the fraction matters,
// so "3.0 feet" is spoken as "3 feet" and "0.6 metres" keeps its decimal.
func decimalWords(v float64, singular, plural string) string {
	unit := plural
	if math.Abs(v-1) < 0.05 {
		unit = singular
	}
	if math.Abs(v-math.Round(v)) < 0.05 {
		return fmt.Sprintf("%.0f %s", v, unit)
	}
	return fmt.Sprintf("%.1f %s", v, unit)
}

// depthWords: "9 kilometers" / "1 kilometer" (whole units).
func depthWords(km float64) string { return distanceWords(km, false) } // a depth is kilometres whatever the listener's units

// distanceWords: "15 miles" / "25 kilometers" (whole units; "1 mile").
func distanceWords(km float64, imperial bool) string {
	if imperial {
		mi := int(math.Round(units.MilesOf(km)))
		if mi == 1 {
			return "1 mile"
		}
		return fmt.Sprintf("%d miles", mi)
	}
	k := int(math.Round(km))
	if k == 1 {
		return "1 kilometer"
	}
	return fmt.Sprintf("%d kilometers", k)
}

// ringWords is the ring's size as an adjective ("16 mile", "25 kilometer").
func ringWords(km float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%d mile", int(math.Round(units.MilesOf(km))))
	}
	return fmt.Sprintf("%d kilometer", int(math.Round(km)))
}

// durationWords: "5 hours and 35 minutes", "3 days and 4 hours", "12 minutes".
func durationWords(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	plural := func(n int, unit string) string {
		if n == 1 {
			return "1 " + unit
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	switch {
	case days > 0 && hours > 0:
		return plural(days, "day") + " and " + plural(hours, "hour")
	case days > 0:
		return plural(days, "day")
	case hours > 0 && mins > 0:
		return plural(hours, "hour") + " and " + plural(mins, "minute")
	case hours > 0:
		return plural(hours, "hour")
	}
	return plural(mins, "minute")
}
