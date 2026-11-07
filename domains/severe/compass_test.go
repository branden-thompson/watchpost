package severe

import "testing"

// compassShipped is compass as 0.19.3 shipped it, the integer form.
func compassShipped(deg int) string {
	names := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return names[(((deg%360)+360)%360+22)/45%8]
}

// TestTheTropicalHeadingMatchesTheShippedForm is 0.19.5 Q5.4 (DR-7, DV-7): a
// storm's heading goes through geo.CompassIndex, and names what the integer
// form named for every whole degree from -720 to 720.
func TestTheTropicalHeadingMatchesTheShippedForm(t *testing.T) {
	for deg := -720; deg <= 720; deg++ { // bounded (P10-02)
		if got, want := compass(deg), compassShipped(deg); got != want {
			t.Fatalf("compass(%d) = %s; the integer form gave %s", deg, got, want)
		}
	}
}
