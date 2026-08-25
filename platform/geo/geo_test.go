package geo

import "testing"

func TestBearingDeg(t *testing.T) {
	cases := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		lo, hi                 float64
	}{
		{"due north", 33, -117, 34, -117, 359.5, 360.5},
		{"due east", 0, 0, 0, 1, 89.5, 90.5},
		{"due south", 34, -117, 33, -117, 179.5, 180.5},
		{"north-west", 33, -117, 34, -118.2, 310, 320},
	}
	for _, c := range cases {
		got := BearingDeg(c.lat1, c.lon1, c.lat2, c.lon2)
		if got == 360 {
			got = 0
		}
		if c.name == "due north" && got < 1 {
			continue // 0 and 360 are the same heading
		}
		if got < c.lo || got > c.hi {
			t.Fatalf("%s: bearing %.1f, want %.0f..%.0f", c.name, got, c.lo, c.hi)
		}
	}
}

func TestHaversineKM(t *testing.T) {
	// Carlsbad, CA (92008) to McClellan-Palomar Airport (KCRQ): ~5.5 km.
	if d := HaversineKM(33.1602, -117.325, 33.1283, -117.2797); d < 5 || d > 6 {
		t.Fatalf("Carlsbad->KCRQ = %.2f km, want ~5.5", d)
	}
	if d := HaversineKM(33.2, -117.38, 33.2, -117.38); d != 0 {
		t.Fatalf("same point must be 0, got %v", d)
	}
	// Antipodal-ish sanity: LA to Sydney ~12,050 km.
	if d := HaversineKM(34.05, -118.24, -33.87, 151.21); d < 12000 || d > 12100 {
		t.Fatalf("LA->Sydney = %.0f km", d)
	}
}
