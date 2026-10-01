package app

// mapcost_standin_test.go - the stand-ins W18.6's measure asks (D-185): NDFD
// as NDFD answers, Open-Meteo Marine as it does, every address kept. Apart
// from the measure, which runs without -race, because the history's tests
// ask them too.

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// costGet keeps every address asked; it answers NDFD as NDFD answers - see
// ndfdAsItIs - and nothing else, so Open-Meteo is asked exactly where the
// map would ask it.
type costGet struct {
	mu    sync.Mutex
	asked map[string]bool
	now   time.Time
	ndfd  map[string][]byte // NDFD's answers built, by address: a refresh asks each again
}

func (c *costGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked[rawURL] = true
	if strings.Contains(rawURL, "graphical.weather.gov") {
		if c.ndfd == nil {
			c.ndfd = map[string][]byte{}
		}
		if _, ok := c.ndfd[rawURL]; !ok {
			c.ndfd[rawURL] = ndfdAsItIs(rawURL, c.now)
		}
		return c.ndfd[rawURL], nil
	}
	if strings.Contains(rawURL, "marine-api.open-meteo.com") {
		return marineAsItIs(rawURL, c.now), nil
	}
	return nil, errors.New("the measure answers nothing")
}

// marineAsItIs is Open-Meteo Marine's answer for the points asked: the open
// sea's - west of 127W, past NDFD's reach - and nothing ashore (D-194).
func marineAsItIs(rawURL string, now time.Time) []byte {
	q, _ := url.ParseQuery(rawURL[strings.Index(rawURL, "?")+1:])
	lons := strings.Split(q.Get("longitude"), ",")
	hour := now.UTC().Truncate(time.Hour)
	var times, days []string
	for h := -3; h <= 13; h++ {
		times = append(times, `"`+hour.Add(time.Duration(h)*time.Hour).Format("2006-01-02T15:04")+`"`)
	}
	for k := range temperature.Days {
		days = append(days, `"`+hour.AddDate(0, 0, k).Format("2006-01-02")+`"`)
	}
	var pts []string
	for _, l := range lons {
		v := "null"
		if lon, _ := strconv.ParseFloat(l, 64); lon < -127 {
			v = "2"
		}
		pts = append(pts, `{"utc_offset_seconds":0,"hourly":{"time":[`+strings.Join(times, ",")+`],"wave_height":[`+strings.TrimSuffix(strings.Repeat(v+",", len(times)), ",")+
			`]},"daily":{"time":[`+strings.Join(days, ",")+`],"wave_height_max":[`+strings.TrimSuffix(strings.Repeat(v+",", len(days)), ",")+`]}}`)
	}
	return []byte("[" + strings.Join(pts, ",") + "]")
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
	for i, p := range pts {
		waves := values(len(hours), "3")
		if lon, _ := strconv.ParseFloat(strings.Split(p, ",")[1], 64); lon < -127 || lon > -117 {
			waves = strings.Repeat(`<value xsi:nil="true"/>`, len(hours)) // ashore, or past NDFD's reach (D-194)
		}
		b.WriteString(`<parameters applicable-location="point` + strconv.Itoa(i+1) + `">` +
			`<temperature type="hourly" units="Fahrenheit" time-layout="k-p1h">` + values(len(hours), "60") + `</temperature>` +
			`<temperature type="apparent" units="Fahrenheit" time-layout="k-p1h-appt">` + values(len(feels), "61") + `</temperature>` +
			`<temperature type="maximum" units="Fahrenheit" time-layout="k-p24h-max">` + values(len(maxs), "70") + `</temperature>` +
			`<temperature type="minimum" units="Fahrenheit" time-layout="k-p24h-min">` + values(len(mins), "50") + `</temperature>` +
			`<wind-speed type="sustained" units="knots" time-layout="k-p1h">` + values(len(hours), "10") + `</wind-speed>` +
			`<wind-speed type="gust" units="knots" time-layout="k-p1h">` + values(len(hours), "15") + `</wind-speed>` +
			`<direction type="wind" units="degrees true" time-layout="k-p1h">` + values(len(hours), "180") + `</direction>` +
			`<water-state time-layout="k-p1h"><waves type="significant" units="feet">` + waves + `</waves></water-state>` +
			`<precipitation type="liquid" units="inches" time-layout="k-p6h">` + values(len(sixes), "0.10") + `</precipitation>` +
			`<precipitation type="snow" units="inches" time-layout="k-p6h">` + values(len(sixes), "0.00") + `</precipitation>` +
			`</parameters>`)
	}
	b.WriteString(`</data></dwml>`)
	return []byte(b.String())
}
