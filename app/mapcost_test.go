//go:build !race

// THE RACE DETECTOR ADDS NOTHING HERE, AND COST THE GATE THREE MINUTES: the
// measure is one goroutine weighing addresses over stand-in answers - built
// and parsed sixty times over - so it runs in the gate's legs without -race.

package app

// mapcost_test.go — W18.6 (D-185): what the map costs Open-Meteo's quota,
// measured without spending it. The map's own temperature pipeline asks a
// stand-in that answers nothing and keeps every address; each distinct
// address is weighed as Open-Meteo bills it (temperature.CallWeight) - once
// a refresh, as the hour's cache would ask it.

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
)

// openMeteoRefresh is one refresh of the map's temperature pipeline for an ask:
// its weight against Open-Meteo's quota, by service.
func openMeteoRefresh(t *testing.T, ask tty.MapAsk, warm bool) (total float64, by map[string]float64) {
	t.Helper()
	get := &costGet{asked: map[string]bool{}, now: time.Now()} // the pipeline reads the clock
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	if warm { // the recorder has run since midnight, as it does whenever watchpost runs (D-172)
		lp.history = history.Open(t.TempDir(), time.Now, historyDatasets...)
		day := time.Date(ask.Anchor.Year(), ask.Anchor.Month(), ask.Anchor.Day(), 0, 0, 0, 0, ask.Anchor.Location())
		for _, b := range fieldBoxes(ask.Region, geo.Box(ask.View)) {
			lat := temperature.NDFDLatticeFor(b.Name, b.Box) // as the recorder keeps it (D-201)
			vals := make([]float64, lat.Cols*lat.Rows)
			for h := day; !h.After(ask.Anchor); h = h.Add(time.Hour) {
				rec := history.Record{Key: history.Key{Source: "ndfd", Place: b.Name}, At: h, IssuedAt: h, Shape: shapeOf(lat),
					Values: map[string][]float64{"temp": vals, "feels": vals, "wind": vals}}
				if !lp.history.Put(ndfdHourly.Name, rec) {
					t.Fatal("could not seed the history")
				}
			}
		}
	}
	lp.mapTemperature(context.Background(), ask) // the first refresh learns the land (D-194)
	first := 0.0
	for u := range get.asked {
		first += temperature.CallWeight(u)
	}
	t.Logf("    the first refresh, learning the land: %.1f", first)
	get.asked = map[string]bool{}
	// A REFRESH IS THE NEXT HOUR, OR NEW BOXES: never the answer kept for this
	// one (P-16), which would measure nothing and pass every budget.
	lp.tempAnswers = lazyMemo[tempKey, tempCore]{}
	lp.mapTemperature(context.Background(), ask)
	if len(get.asked) == 0 {
		t.Fatal("the refresh asked nothing; it measures nothing")
	}
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
		UV: true, Air: true, Fahrenheit: true, TempNDFD: true, Anchor: time.Now().Truncate(time.Hour)} // the listener's default since D-190
}

// quiet is an ask with UV and air quality off: what temperature, feels-like,
// wind, rain and waves cost alone.
func quiet(ask tty.MapAsk) tty.MapAsk {
	ask.UV, ask.Air = false, false
	return ask
}

// THE MAP'S OPEN-METEO WEIGHT A REFRESH, WITHIN ITS BUDGET (W18.6, D-185):
// the weight of a refresh - each hour the map is open, and each view change
// into new boxes - for the lower 48, a state and Alaska, both modes, every
// Open-Meteo row on. The budgets are today's, measured: a change that spends
// more fails here, and W19 lowers them as each layer moves to its keyless
// source. NDFD is the listener's default (D-190); each row logs beside it
// what Open-Meteo chosen costs. "warm" is the history holding today's hours,
// as the recorder leaves it (D-188, D-189). Each is a refresh's second: the
// first learns the land Open-Meteo Marine answers nothing for, once a run
// (D-194) - logged. 10,000 a day is the free tier.
func TestTheMapsOpenMeteoWeightIsWithinItsBudget(t *testing.T) {
	california := geo.Box{W: -124.5, S: 32.5, E: -114, N: 42}
	for _, c := range []struct {
		name   string
		ask    tty.MapAsk
		warm   bool
		budget float64
	}{
		{"lower 48, Radar mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), false), false, 6},
		{"lower 48, Forecast mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true), false, 133.2},
		{"California, Radar mode", costAsk(geo.RegionContiguous, california, false), false, 32},
		{"California, Forecast mode", costAsk(geo.RegionContiguous, california, true), false, 296},
		{"Alaska, Radar mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), false), false, 236},
		{"Alaska, Forecast mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), true), false, 337.2},
		{"lower 48, Radar, no UV/air", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), false)), false, 6},
		{"lower 48, Forecast, no UV/air", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true)), false, 133.2},
		{"California, Forecast, no UV/air", quiet(costAsk(geo.RegionContiguous, california, true)), false, 296},
		{"lower 48, Forecast, no UV/air, warm", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true)), true, 24},
		{"California, Forecast, no UV/air, warm", quiet(costAsk(geo.RegionContiguous, california, true)), true, 72},
		{"California, Forecast, warm", costAsk(geo.RegionContiguous, california, true), true, 72},
	} {
		total, by := openMeteoRefresh(t, c.ask, c.warm)
		was := c.ask
		was.TempNDFD = false
		before, _ := openMeteoRefresh(t, was, c.warm) // Open-Meteo chosen: the cost before D-190
		t.Logf("%-32s Open-Meteo chosen %8s", c.name, strconvF(before))
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
