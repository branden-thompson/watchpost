package geodata

import "testing"

// TestStateExtentsHoldTheirCities is 0.18.0 UAT-1 U1-12 (D-64): each state's
// extent, from its own cities, holds its known cities and not its neighbours'.
func TestStateExtentsHoldTheirCities(t *testing.T) {
	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ext := idx.StateExtents()
	for _, c := range []struct {
		state    string
		lat, lon float64
	}{{"CA", 33.2, -117.38}, {"CA", 40.59, -122.39}, {"TX", 29.76, -95.37}, {"NY", 40.71, -74.01}, {"AK", 61.2, -149.9}} {
		e, ok := ext[c.state]
		if !ok || !e.Contains(c.lat, c.lon) {
			t.Errorf("%s's extent %+v does not hold %v, %v", c.state, e, c.lat, c.lon)
		}
	}
	if ext["CA"].Contains(29.76, -95.37) {
		t.Error("California's extent holds Houston")
	}
}

// TestStateNamesAreWhole: a code reads as the state's or territory's name;
// an unknown code reads as itself.
func TestStateNamesAreWhole(t *testing.T) {
	for code, want := range map[string]string{"CA": "California", "NY": "New York", "DC": "the District of Columbia", "PR": "Puerto Rico", "ZZ": "ZZ"} {
		if got := StateName(code); got != want {
			t.Errorf("%s reads %q, want %q", code, got, want)
		}
	}
}
