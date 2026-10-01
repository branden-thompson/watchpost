package app

// mapcost_test.go — W18.6 (D-185): what the map costs Open-Meteo's quota,
// measured without spending it. The map's own temperature pipeline asks a
// stand-in that answers nothing and keeps every address; each distinct
// address is weighed as Open-Meteo bills it (temperature.CallWeight) - once
// a refresh, as the hour's cache would ask it.

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// costGet keeps every address asked and answers none.
type costGet struct {
	mu    sync.Mutex
	asked map[string]bool
}

func (c *costGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked[rawURL] = true
	return nil, errors.New("the measure answers nothing")
}

// openMeteoRefresh is one refresh of the map's temperature pipeline for an ask:
// its weight against Open-Meteo's quota, by service.
func openMeteoRefresh(t *testing.T, ask tty.MapAsk) (total float64, by map[string]float64) {
	t.Helper()
	get := &costGet{asked: map[string]bool{}}
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	lp.mapTemperature(context.Background(), ask)
	if !ask.Forecast && ask.Region != geo.RegionContiguous { // where HRRR is not, Radar mode's hours ahead are a model's rain (D-115)
		view := geo.Box(ask.View)
		withModelRain(context.Background(), tty.MapRadar{}, lp.temp.rain, ask.Region, view, ask.Anchor, ask.Anchor.Add(12*time.Hour))
	}
	by = map[string]float64{}
	for u := range get.asked {
		if w := temperature.CallWeight(u); w > 0 {
			host := u[strings.Index(u, "//")+2:]
			by[host[:strings.IndexAny(host, "?")]] += w
			total += w
		}
	}
	return total, by
}

// costAsk is a view of a region with every Open-Meteo row on.
func costAsk(region string, view geo.Box, forecast bool) tty.MapAsk {
	return tty.MapAsk{Region: region, View: tty.MapView(view), Forecast: forecast,
		UV: true, Air: true, Fahrenheit: true, Anchor: tempNow.Truncate(time.Hour)}
}

// THE MAP'S OPEN-METEO WEIGHT A REFRESH, WITHIN ITS BUDGET (W18.6, D-185):
// the weight of a refresh - each hour the map is open, and each view change
// into new boxes - for the lower 48, a state and Alaska, both modes, every
// Open-Meteo row on. The budgets are today's, measured: a change that spends
// more fails here, and W19 lowers them as each layer moves to its keyless
// source. 10,000 a day is the free tier.
func TestTheMapsOpenMeteoWeightIsWithinItsBudget(t *testing.T) {
	california := geo.Box{W: -124.5, S: 32.5, E: -114, N: 42}
	for _, c := range []struct {
		name   string
		ask    tty.MapAsk
		budget float64
	}{
		{"lower 48, Radar mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), false), 265.2},
		{"lower 48, Forecast mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true), 343.2},
		{"California, Radar mode", costAsk(geo.RegionContiguous, california, false), 544},
		{"California, Forecast mode", costAsk(geo.RegionContiguous, california, true), 704},
		{"Alaska, Radar mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), false), 695.2},
		{"Alaska, Forecast mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), true), 695.2},
	} {
		total, by := openMeteoRefresh(t, c.ask)
		var parts []string
		for host, w := range by {
			parts = append(parts, host+" "+strconvF(w))
		}
		sort.Strings(parts)
		t.Logf("%-26s %8s a refresh  (%s)", c.name, strconvF(total), strings.Join(parts, "; "))
		if total > c.budget {
			t.Errorf("%s weighs %v a refresh; its budget is %v (D-185)", c.name, total, c.budget)
		}
	}
}

func strconvF(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
