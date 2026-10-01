package app

// mapcost_test.go — W18.6 (D-185): what the map costs Open-Meteo's quota,
// measured without spending it. The map's own temperature pipeline asks a
// stand-in that answers nothing and keeps every address; each distinct
// address is weighed as Open-Meteo bills it (temperature.CallWeight) - once
// a refresh, as the hour's cache would ask it.

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// costGet keeps every address asked; it answers NDFD as NDFD answers - see
// ndfdAsItIs - and nothing else, so Open-Meteo is asked exactly where the
// map would ask it.
type costGet struct {
	mu    sync.Mutex
	asked map[string]bool
	now   time.Time
}

func (c *costGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked[rawURL] = true
	if strings.Contains(rawURL, "graphical.weather.gov") {
		return ndfdAsItIs(rawURL, c.now), nil
	}
	return nil, errors.New("the measure answers nothing")
}

// ndfdAsItIs is an NDFD answer for every point asked, with NDFD's own gaps:
// the hours from the current one for a week, but feels-like from the next
// hour (D-119); the evening's answer, whatever the hour the measure runs -
// each day's maximum from tomorrow's 08:00 and minimum from tonight's 20:00:
// no Today high or low (D-189's case, the worst); wind, gusts and waves
// hourly; six-hour rain and snow.
func ndfdAsItIs(rawURL string, now time.Time) []byte {
	q, _ := url.ParseQuery(rawURL[strings.Index(rawURL, "?")+1:])
	pts := strings.Fields(q.Get("listLatLon"))
	hour := now.UTC().Truncate(time.Hour)
	day := time.Date(hour.Year(), hour.Month(), hour.Day(), 0, 0, 0, 0, time.UTC)
	var b strings.Builder
	layout := func(key string, starts []time.Time, span time.Duration) {
		b.WriteString(`<time-layout><layout-key>` + key + `</layout-key>`)
		for _, s := range starts {
			b.WriteString(`<start-valid-time>` + s.Format(time.RFC3339) + `</start-valid-time><end-valid-time>` + s.Add(span).Format(time.RFC3339) + `</end-valid-time>`)
		}
		b.WriteString(`</time-layout>`)
	}
	every := func(from time.Time, step time.Duration, n int) []time.Time {
		var out []time.Time
		for k := range n {
			out = append(out, from.Add(time.Duration(k)*step))
		}
		return out
	}
	daily := func(from int, at time.Duration) []time.Time {
		var out []time.Time
		for k := from; k < from+7; k++ {
			out = append(out, day.AddDate(0, 0, k).Add(at))
		}
		return out
	}
	hours, feels, maxs, mins, sixes := every(hour, time.Hour, 168), every(hour.Add(time.Hour), time.Hour, 167), daily(1, 8*time.Hour), daily(0, 20*time.Hour), every(hour, 6*time.Hour, 13)
	values := func(n int, v string) string { return strings.Repeat(`<value>`+v+`</value>`, n) }
	b.WriteString(`<dwml><data>`)
	for i, p := range pts {
		ll := strings.Split(p, ",")
		b.WriteString(`<location><location-key>point` + strconv.Itoa(i+1) + `</location-key><point latitude="` + ll[0] + `" longitude="` + ll[1] + `"/></location>`)
	}
	layout("k-p1h", hours, time.Hour)
	layout("k-p1h-appt", feels, time.Hour)
	layout("k-p24h-max", maxs, 12*time.Hour)
	layout("k-p24h-min", mins, 12*time.Hour)
	layout("k-p6h", sixes, 6*time.Hour)
	for i := range pts {
		b.WriteString(`<parameters applicable-location="point` + strconv.Itoa(i+1) + `">` +
			`<temperature type="hourly" units="Fahrenheit" time-layout="k-p1h">` + values(len(hours), "60") + `</temperature>` +
			`<temperature type="apparent" units="Fahrenheit" time-layout="k-p1h-appt">` + values(len(feels), "61") + `</temperature>` +
			`<temperature type="maximum" units="Fahrenheit" time-layout="k-p24h-max">` + values(len(maxs), "70") + `</temperature>` +
			`<temperature type="minimum" units="Fahrenheit" time-layout="k-p24h-min">` + values(len(mins), "50") + `</temperature>` +
			`<wind-speed type="sustained" units="knots" time-layout="k-p1h">` + values(len(hours), "10") + `</wind-speed>` +
			`<wind-speed type="gust" units="knots" time-layout="k-p1h">` + values(len(hours), "15") + `</wind-speed>` +
			`<direction type="wind" units="degrees true" time-layout="k-p1h">` + values(len(hours), "180") + `</direction>` +
			`<water-state time-layout="k-p1h"><waves type="significant" units="feet">` + values(len(hours), "3") + `</waves></water-state>` +
			`<precipitation type="liquid" units="inches" time-layout="k-p6h">` + values(len(sixes), "0.10") + `</precipitation>` +
			`<precipitation type="snow" units="inches" time-layout="k-p6h">` + values(len(sixes), "0.00") + `</precipitation>` +
			`</parameters>`)
	}
	b.WriteString(`</data></dwml>`)
	return []byte(b.String())
}

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
			lat := temperature.LatticeFor(b.Name, b.Box)
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
// as the recorder leaves it (D-188, D-189). 10,000 a day is the free tier.
func TestTheMapsOpenMeteoWeightIsWithinItsBudget(t *testing.T) {
	california := geo.Box{W: -124.5, S: 32.5, E: -114, N: 42}
	for _, c := range []struct {
		name   string
		ask    tty.MapAsk
		warm   bool
		budget float64
	}{
		{"lower 48, Radar mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), false), false, 156},
		{"lower 48, Forecast mode", costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true), false, 343.2},
		{"California, Radar mode", costAsk(geo.RegionContiguous, california, false), false, 320},
		{"California, Forecast mode", costAsk(geo.RegionContiguous, california, true), false, 704},
		{"Alaska, Radar mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), false), false, 474},
		{"Alaska, Forecast mode", costAsk(geo.RegionAlaska, regionBox(geo.RegionAlaska), true), false, 695.2},
		{"lower 48, Radar, no UV/air", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), false)), false, 78},
		{"lower 48, Forecast, no UV/air", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true)), false, 265.2},
		{"California, Forecast, no UV/air", quiet(costAsk(geo.RegionContiguous, california, true)), false, 544},
		{"lower 48, Forecast, no UV/air, warm", quiet(costAsk(geo.RegionContiguous, regionBox(geo.RegionContiguous), true)), true, 156},
		{"California, Forecast, no UV/air, warm", quiet(costAsk(geo.RegionContiguous, california, true)), true, 320},
		{"California, Forecast, warm", costAsk(geo.RegionContiguous, california, true), true, 480},
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
