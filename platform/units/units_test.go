package units

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// THE CONVERSIONS AGREE WITH THEIR DEFINITIONS (W14 S-6): the international
// mile is exactly 1.609344 km, a knot exactly 1.852 km/h, a metre a second
// exactly 3.6 km/h; water freezes at 0 °C, 32 °F and boils at 100 °C, 212 °F.
func TestTheConversionsAgreeWithTheirDefinitions(t *testing.T) {
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"0 °C", FahrenheitOf(0), 32},
		{"100 °C", FahrenheitOf(100), 212},
		{"-40 °C", FahrenheitOf(-40), -40},
		{"212 °F", CelsiusOf(212), 100},
		{"a 5 °C difference", FahrenheitDelta(5), 9},
		{"1 mile", KmOf(1), 1.609344},
		{"1.609344 km", MilesOf(1.609344), 1},
		{"10 m/s", KmhOf(10), 36},
		{"36 km/h", MpsOf(36), 10},
		{"1 knot", KmhOfKnots(1), 1.852},
		{"1 foot", MetresOfFeet(1), 0.3048},
		{"0.3048 m", FeetOf(0.3048), 1},
		{"1 inch", MmOfInches(1), 25.4},
		{"25.4 mm", InchesOf(25.4), 1},
		{"1.852 km/h in knots, as m/s", KnotsOf(MpsOf(1.852)), 1},
		{"1 mph in m/s", MpsOfMph(1), 0.44704},
		{"0.44704 m/s in mph", MphOf(0.44704), 1},
	} {
		if !near(c.got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
}

// EACH PAIR IS EACH OTHER'S INVERSE.
func TestEachPairRoundTrips(t *testing.T) {
	for _, v := range []float64{-40, -3.7, 0, 0.5, 12.25, 99.9, 1000} {
		if !near(CelsiusOf(FahrenheitOf(v)), v) || !near(MilesOf(KmOf(v)), v) || !near(MpsOf(KmhOf(v)), v) ||
			!near(FeetOf(MetresOfFeet(v)), v) || !near(InchesOf(MmOfInches(v)), v) || !near(MphOf(MpsOfMph(v)), v) {
			t.Errorf("%v does not round-trip", v)
		}
	}
}
