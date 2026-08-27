package geo

import "testing"

// Quality pass Q6 (L3-F9): one compass arithmetic for the three word tables.
func TestCompassIndexMatchesTheThreeOldFormulas(t *testing.T) {
	for deg := 0.0; deg < 360; deg += 0.5 {
		if got, want := CompassIndex(deg, 16), int((deg+11.25)/22.5)%16; got != want {
			t.Fatalf("16-point at %v: %d, want %d", deg, got, want)
		}
		if got, want := CompassIndex(deg, 8), int((deg+22.5)/45)%8; got != want {
			t.Fatalf("8-point at %v: %d, want %d", deg, got, want)
		}
	}
	if CompassIndex(359.9, 16) != 0 || CompassIndex(0, 8) != 0 || CompassIndex(180, 8) != 4 || CompassIndex(90, 0) != 0 {
		t.Fatal("north wraps; zero points is safe")
	}
}
