package geo

import (
	"math"
	"testing"
)

// compassPointShipped is synth's compassPoint as 0.19.3 shipped it, rounding
// half away from zero: one of the two forms Compass16 replaces.
func compassPointShipped(deg float64) string {
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	i := int(math.Mod(math.Round(deg/22.5), 16))
	if i < 0 {
		i += 16
	}
	return pts[i]
}

// tableShipped is the table usgs and tty indexed with CompassIndex.
func tableShipped(deg float64) string {
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return pts[CompassIndex(deg, 16)]
}

// TestCompass16MatchesTheTablesItReplaces is 0.19.5 Q5.4 (DR-7, D-17): the 16
// abbreviations have one owner, and over every bearing a feed sends, 0 to 360
// in hundredths of a degree, it names what each of the three copies named.
func TestCompass16MatchesTheTablesItReplaces(t *testing.T) {
	for i := 0; i <= 36000; i++ { // bounded (P10-02)
		deg := float64(i) / 100
		got := Compass16(deg)
		if want := tableShipped(deg); got != want {
			t.Fatalf("Compass16(%v) = %s; the usgs and tty table gave %s", deg, got, want)
		}
		if want := compassPointShipped(deg); got != want {
			t.Fatalf("Compass16(%v) = %s; synth's compassPoint gave %s", deg, got, want)
		}
	}
}

// TestCompassIndexNormalisesANegativeBearing is D-17 (DR-7a): a bearing below
// zero, or a whole turn and more, names the point it is on the rose. A
// negative exact half-sector (-11.25°) is where synth's rounding half away
// from zero named the point before it; no feed sends one (NDBC's directions
// are 0 to 360), so the edge is recorded rather than kept (A-17).
func TestCompassIndexNormalisesANegativeBearing(t *testing.T) {
	for _, c := range []struct {
		deg  float64
		want string
	}{{-20, "NNW"}, {-90, "W"}, {-360, "N"}, {-370, "N"}, {380, "NNE"}, {720, "N"}, {-11.24, "N"}, {-11.26, "NNW"}} {
		if got := Compass16(c.deg); got != c.want {
			t.Errorf("Compass16(%v) = %s; want %s", c.deg, got, c.want)
		}
	}
}

// TestOnEarthRefusesWhatIsNoPlace is 0.19.5 Q5.6 (DR-9): the one test of
// whether a latitude and longitude name a place on Earth. The edges are on
// it, and anything past them, NaN or infinite is not.
func TestOnEarthRefusesWhatIsNoPlace(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	for _, c := range []struct {
		lat, lon float64
		want     bool
	}{
		{0, 0, true}, {90, 180, true}, {-90, -180, true}, {33.2, -117.4, true},
		{90.01, 0, false}, {0, 180.01, false}, {-90.01, 0, false}, {0, -180.01, false},
		{nan, 0, false}, {0, nan, false}, {inf, 0, false}, {0, -inf, false},
	} {
		if got := OnEarth(c.lat, c.lon); got != c.want {
			t.Errorf("OnEarth(%v, %v) = %v; want %v", c.lat, c.lon, got, c.want)
		}
	}
}
