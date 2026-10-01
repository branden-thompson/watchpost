package app

// uvspread_test.go — D-202 (UAT-2 U2-51): UV's EPA cities spread over the
// view, the count the listener's - not the most populous alone, which in the
// lower 48 are New York and its boroughs, Los Angeles, Chicago, Houston,
// Phoenix and Philadelphia: none between the coasts.

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

func rankedUS(t *testing.T) []geodata.City {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	return idx.TopUS(uvRanked)
}

// spacedApart fails when two cities sit closer than km.
func spacedApart(t *testing.T, name string, got []geodata.City, km float64) {
	t.Helper()
	for i, a := range got {
		for _, b := range got[i+1:] {
			if d := geo.HaversineKM(a.Lat, a.Lon, b.Lat, b.Lon); d < km {
				t.Errorf("%s: %s and %s are %.0f km apart; want at least %.0f", name, a.Name, b.Name, d, km)
			}
		}
	}
}

// THE LOWER 48 GETS CITIES ACROSS IT (D-202): at each count the listener can
// choose, close to that many, none within 100 km of another, and every part
// of the country holds some - the West, the Plains, the Southeast and the
// Northeast - not the largest metros alone.
func TestUVCitiesAreSpreadOverTheView(t *testing.T) {
	ranked := rankedUS(t)
	us := regionBox(geo.RegionContiguous)
	for _, n := range []int{8, 24, 48} { // the Setting's counts (D-202)
		got := spreadInView(ranked, us, n)
		if len(got) > n || len(got) < n*2/3 {
			t.Errorf("%d cities asked for: %d chosen", n, len(got))
		}
		spacedApart(t, "the lower 48", got, uvCitySpacingKm)
		parts := map[string]int{}
		for _, c := range got {
			switch {
			case c.Lon < -104:
				parts["West"]++
			case c.Lon < -90:
				parts["Plains"]++
			case c.Lat < 37:
				parts["Southeast"]++
			default:
				parts["Northeast"]++
			}
		}
		for _, p := range []string{"West", "Plains", "Southeast", "Northeast"} {
			if parts[p] == 0 {
				t.Errorf("%d cities: none in the %s (%v)", n, p, parts)
			}
		}
		names := make([]string, len(got))
		for i, c := range got {
			names[i] = c.Name
		}
		if slices.Contains(names, "Brooklyn") && slices.Contains(names, "Queens") {
			t.Errorf("%d cities: Brooklyn and Queens both, one place at this zoom", n)
		}
	}
}

// A STATE'S VIEW STILL GETS ITS COUNT (D-202): the spacing shrinks with the
// view, so California's 24 are not cut to a handful by 100 km.
func TestAStatesViewGetsItsCount(t *testing.T) {
	for name, view := range map[string]geo.Box{
		"California":          {W: -124.5, S: 32.5, E: -114, N: 42},
		"Southern California": {W: -119, S: 32.5, E: -116, N: 34.5}, // some 280 km across: 100 km apart would leave a handful
	} {
		got := spreadInView(rankedUS(t), view, 24)
		if len(got) < 16 || len(got) > 24 {
			t.Errorf("%s: %d cities of 24", name, len(got))
		}
		for _, c := range got {
			if !view.Contains(c.Lat, c.Lon) || c.TZ == "" {
				t.Errorf("%s: %s is outside the view or has no zone", name, c.Name)
			}
		}
	}
}

// THE CHOICE IS KEPT IN RANK ORDER, AND ONLY CITIES EPA CAN BE READ FOR:
// outside the view, or with no zone, a city is passed over.
func TestSpreadCitiesKeepTheRanking(t *testing.T) {
	view := geo.Box{W: -120, S: 30, E: -110, N: 40}
	ranked := []geodata.City{{Name: "Out", Lat: 45, Lon: -100, TZ: "America/Chicago"}, {Name: "NoZone", Lat: 35, Lon: -115},
		{Name: "A", Lat: 31, Lon: -119, TZ: "America/Los_Angeles"}, {Name: "B", Lat: 39, Lon: -111, TZ: "America/Denver"},
		{Name: "C", Lat: 31.01, Lon: -119.01, TZ: "America/Los_Angeles"}} // C beside A: passed over
	got := spreadInView(ranked, view, 8)
	var names []string
	for _, c := range got {
		names = append(names, c.Name)
	}
	if !slices.Equal(names, []string{"A", "B"}) {
		t.Errorf("chose %v; want A then B", names)
	}
	// A cell's second city, taken in the second round, still comes before a
	// smaller city taken in the first: the markers are drawn largest first.
	rounds := []geodata.City{{Name: "R1", Lat: 31, Lon: -119, TZ: "America/Los_Angeles"},
		{Name: "R2", Lat: 34, Lon: -117, TZ: "America/Los_Angeles"}, {Name: "R3", Lat: 39, Lon: -111, TZ: "America/Denver"}}
	names = names[:0]
	for _, c := range spreadInView(rounds, view, 3) {
		names = append(names, c.Name)
	}
	if !slices.Equal(names, []string{"R1", "R2", "R3"}) {
		t.Errorf("chose %v; want the ranking's order, R1 R2 R3", names)
	}
	if spreadInView(nil, view, 8) != nil || spreadInView(ranked, view, 0) != nil {
		t.Error("no ranking, or no count, chose cities")
	}
}

// slowEPA answers EPA's fixture after a pause, counting how many asks are in
// flight at once.
type slowEPA struct {
	mu            sync.Mutex
	asked         []string
	inFlight, max atomic.Int32
}

func (g *slowEPA) GetText(_ context.Context, u string, _ ...httpx.Option) ([]byte, error) {
	n := g.inFlight.Add(1)
	defer g.inFlight.Add(-1)
	for {
		m := g.max.Load()
		if n <= m || g.max.CompareAndSwap(m, n) {
			break
		}
	}
	g.mu.Lock()
	g.asked = append(g.asked, u)
	g.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	return (&epaFixture{}).GetText(context.Background(), u)
}

// THE CITIES ARE ASKED A FEW AT A TIME (D-202): never one by one - 24 serial
// asks would hold the map's UV for seconds - and never all at once; the
// markers come back in the ranking's order whatever order they answered in.
func TestUVCitiesAreAskedAFewAtATime(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	now := time.Date(2026, 9, 30, 12, 20, 0, 0, la)
	var ranked []geodata.City
	for i := range 12 {
		ranked = append(ranked, geodata.City{Name: "City" + string(rune('A'+i)), State: "CA", Lat: 33 + float64(i)*0.1, Lon: -117, TZ: "America/Los_Angeles"})
	}
	get := &slowEPA{}
	c := &uvCities{epa: uv.NewEPA(get, ""), cities: func(geo.Box, int) []geodata.City { return ranked }}
	marks := c.markers(context.Background(), geo.Box{W: -118, S: 32, E: -116, N: 35}, 24, now.Truncate(time.Hour), true)
	if len(get.asked) != len(ranked) {
		t.Fatalf("%d asks for %d cities", len(get.asked), len(ranked))
	}
	if m := get.max.Load(); m < 2 || m > uvAskers {
		t.Errorf("%d asks in flight at most; want between 2 and %d", m, uvAskers)
	}
	if len(marks) == 0 {
		t.Fatal("no markers")
	}
	feats := marks[0].Features
	for i := 1; i < len(feats); i++ {
		if feats[i-1].Label > feats[i].Label {
			t.Fatalf("the markers are not in the ranking's order: %q before %q", feats[i-1].Label, feats[i].Label)
		}
	}
}
