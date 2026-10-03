package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// countingEPA answers EPA's fixture and keeps every ask; DownTown is refused.
type countingEPA struct {
	mu    sync.Mutex
	asked []string
}

func (g *countingEPA) GetText(ctx context.Context, u string, opts ...httpx.Option) ([]byte, error) {
	g.mu.Lock()
	g.asked = append(g.asked, u)
	g.mu.Unlock()
	if strings.Contains(u, "DOWNTOWN") { // the address names the city upper-cased
		return nil, errors.New("refused")
	}
	return (&epaFixture{}).GetText(ctx, u, opts...)
}

// asks is how many asks were made, and empties the list.
func (g *countingEPA) asks() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := len(g.asked)
	g.asked = nil
	return n
}

// A CITY'S UV IS ASKED ONCE A DAY (W14 P-14, D-211): EPA forecasts each city
// hour by hour for today, so a pan, zoom or refresh asks only for the cities
// not yet read today - the rest are drawn from what was read. The next day
// every city is asked again, and a city EPA did not answer is asked again at
// the next view.
func TestACitysUVIsAskedOnceADay(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	city := func(name string) geodata.City {
		return geodata.City{Name: name, State: "CA", Lat: 33, Lon: -117, TZ: "America/Los_Angeles"}
	}
	view := []geodata.City{city("Vista"), city("Oceanside"), city("DownTown")}
	get := &countingEPA{}
	c := &uvCities{epa: uv.NewEPA(get, ""), cities: func(geo.Box, int) []geodata.City { return view }}
	box := geo.Box{W: -118, S: 32, E: -116, N: 35}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)

	first := c.markers(context.Background(), box, 24, noon, true)
	if n := get.asks(); n != 3 || len(first) == 0 {
		t.Fatalf("the first view asked %d times and drew %d overlays; want the three cities asked and markers drawn", n, len(first))
	}
	view = append(view, city("Carlsbad")) // a pan: one city more
	again := c.markers(context.Background(), box, 24, noon.Add(10*time.Minute), true)
	if n := get.asks(); n != 2 {
		t.Errorf("the next view asked %d times; want the new city and the one EPA refused, nothing it had read today", n)
	}
	if len(again) == 0 || len(again[0].Features) != len(first[0].Features)+1 {
		t.Errorf("the next view draws %v; want every city read today and the new one", featuresOf(again))
	}
	c.markers(context.Background(), box, 24, noon.Add(24*time.Hour), true)
	if n := get.asks(); n != 4 {
		t.Errorf("the next day asked %d times; want every city again", n)
	}
}

// featuresOf is the overlays' feature counts, for a failure's message.
func featuresOf(overlays []tuimaps.Overlay) []int {
	var out []int
	for _, o := range overlays {
		out = append(out, len(o.Features))
	}
	return out
}
