package geo

import (
	"math"
	"testing"
)

// TestNaNCoordinatesAreRefused is FR-4.9 (W0.3, N-3): a coordinate pair is
// two finite decimal numbers on the globe; "NaN", "Inf", exponents, hex
// floats, a third number or one off the globe are refused - strconv alone
// takes the first four.
func TestNaNCoordinatesAreRefused(t *testing.T) {
	for _, s := range []string{"NaN,NaN", "nan, 10", "33, NaN", "Inf,0", "+Inf,-Inf", "1e1,2", "0x1p-2,3", "33,-117,5", "91,0", "0,181", "-90.5,0", "33", "", "33.2,", ",-117", "33,-117 extra", "33..2,1", "Oceanside, CA"} {
		if p, ok := ParseCoordinates(s); ok {
			t.Errorf("%q was taken as %+v", s, p)
		}
	}
	for s, want := range map[string]LatLon{
		"33.2,-117.38":     {33.2, -117.38},
		" 33.2 , -117.38 ": {33.2, -117.38},
		"33.2 -117.38":     {33.2, -117.38},
		"-90,180":          {-90, 180},
		"+45.,.5":          {45, 0.5},
	} {
		if p, ok := ParseCoordinates(s); !ok || p != want {
			t.Errorf("%q gave %+v (%v); want %+v", s, p, ok, want)
		}
	}
}

// TestALocatorIsTheCentreOfItsSquare is FR-1.10 (D-79): a Maidenhead
// locator of four, six or eight characters, either case, is the centre of
// its square; anything else is refused.
func TestALocatorIsTheCentreOfItsSquare(t *testing.T) {
	for s, want := range map[string]LatLon{
		"DM13":     {33.5, -117},                   // the square 2° by 1° from 118°W, 33°N
		"dm13":     {33.5, -117},                   // either case
		"DM13af":   {33.0 + 5.5/24, -118 + 1.0/24}, // subsquare a (lon 0), f (lat 5)
		"JJ00aa00": {1.0 / 480, 1.0 / 240},         // field J is 0°, 0°
		"RR99xx99": {90 - 1.0/480, 180 - 1.0/240},
	} {
		p, ok := ParseLocator(s)
		if !ok || math.Abs(p.Lat-want.Lat) > 1e-9 || math.Abs(p.Lon-want.Lon) > 1e-9 {
			t.Errorf("%q gave %+v (%v); want %+v", s, p, ok, want)
		}
	}
	for _, s := range []string{"DM1", "DM13a", "SM13", "DS13", "DMA3", "DM13ay", "DM13az", "DM13af4", "DM13af4x", "92057", "Oceanside", ""} {
		if p, ok := ParseLocator(s); ok {
			t.Errorf("%q was taken as %+v", s, p)
		}
	}
	if LocatorLabel("dm13AF45") != "DM13af45" || LocatorLabel("dm13") != "DM13" {
		t.Errorf("a locator is written %q, %q", LocatorLabel("dm13AF45"), LocatorLabel("dm13"))
	}
}
