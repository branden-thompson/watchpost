package geodata

import "testing"

// Spec: B3 UAT session 2A — RECENT/SEARCHED seeds are the top-N US cities by
// population, offline, deterministic.

func TestTopUSReturnsMostPopulousUSCities(t *testing.T) {
	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	top := idx.TopUS(25)
	if len(top) != 25 {
		t.Fatalf("want 25 cities, got %d", len(top))
	}
	names := map[string]bool{}
	for i, c := range top {
		if c.Country != "US" {
			t.Fatalf("non-US city in seed list: %+v", c)
		}
		if i > 0 && c.Population > top[i-1].Population {
			t.Fatalf("not population-descending at %d: %d > %d", i, c.Population, top[i-1].Population)
		}
		names[c.ASCII] = true
	}
	for _, must := range []string{"New York City", "Los Angeles", "Chicago"} {
		if !names[must] {
			t.Fatalf("top-25 must include %q; got %v", must, names)
		}
	}
	if idx.TopUS(0) != nil {
		t.Fatal("TopUS(0) must be nil")
	}
}
