package app

// maprain_test.go — 0.18.0 W12.2 and W12.3 (D-115 to D-118): the hours ahead
// outside the lower 48 as Open-Meteo's model rain in radar's colours, and
// Forecast mode's days - each day's heaviest hour and its totals.

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// omRain answers any lattice's rain from Open-Meteo in UTC: every hour from
// start the same rate, and each day its heaviest hour, its rain and its snow.
type omRain struct {
	start      time.Time
	hours      int
	rate       float64
	rain, snow [temperature.Days]float64
	asks       []string
}

func (o *omRain) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	o.asks = append(o.asks, rawURL)
	q, _ := url.ParseQuery(rawURL[strings.Index(rawURL, "?")+1:])
	n := len(strings.Split(q.Get("latitude"), ","))
	type point struct {
		Offset int `json:"utc_offset_seconds"`
		Hourly struct {
			Time   []string  `json:"time"`
			Precip []float64 `json:"precipitation"`
		} `json:"hourly"`
		Daily struct {
			Time    []string  `json:"time"`
			Rain    []float64 `json:"rain_sum"`
			Showers []float64 `json:"showers_sum"`
			Snow    []float64 `json:"snowfall_sum"`
		} `json:"daily"`
	}
	var p point
	for h := range o.hours {
		p.Hourly.Time = append(p.Hourly.Time, o.start.Add(time.Duration(h)*time.Hour).Format("2006-01-02T15:04"))
		p.Hourly.Precip = append(p.Hourly.Precip, o.rate)
	}
	if q.Get("forecast_days") != "" {
		for k := range temperature.Days {
			p.Daily.Time = append(p.Daily.Time, o.start.AddDate(0, 0, k).Format("2006-01-02"))
			p.Daily.Rain, p.Daily.Showers, p.Daily.Snow = append(p.Daily.Rain, o.rain[k]), append(p.Daily.Showers, 0), append(p.Daily.Snow, o.snow[k])
		}
	}
	pts := make([]point, n)
	for i := range pts {
		pts[i] = p
	}
	return json.Marshal(pts)
}

// TestTheHoursAheadOutsideTheLower48AreModelRain is W12.2 (D-115): where
// HRRR is not, the loop's hours ahead are Open-Meteo's hourly rain, a frame
// an hour after the newest observed and up to the horizon, each marked
// forecast, painted in radar's scale with a table of the app's own; the
// source named, and said to be a model's rain, not radar.
func TestTheHoursAheadOutsideTheLower48AreModelRain(t *testing.T) {
	newest := time.Date(2026, 9, 27, 20, 40, 0, 0, time.UTC)
	get := &omRain{start: time.Date(2026, 9, 27, 20, 0, 0, 0, time.UTC), hours: 5, rate: 5}
	om := temperature.NewOpenMeteo(get, "")
	observed := tuimaps.RadarImage(tty.RadarLayer+"/hi", tuimaps.Image{Frames: []tuimaps.LoopFrame{{Valid: newest, Gap: true}},
		Provider: tuimaps.ProviderMRMS, West: -164, South: 15, East: -151, North: 26, Projection: tuimaps.PlateCarree}, newest)
	out := withModelRain(context.Background(), tty.MapRadar{Overlays: []tuimaps.Overlay{observed}, Source: "MRMS"}, om,
		geo.RegionHawaii, geo.Box{}, newest, newest.Add(3*time.Hour))
	if out.Ahead != "Open-Meteo" || len(out.Overlays) != 2 {
		t.Fatalf("ahead %q, %d loops; want Open-Meteo and a forecast loop beside the observed", out.Ahead, len(out.Overlays))
	}
	if out.Overlays[0].During.Until != newest {
		t.Errorf("the observed loop is drawn during %v; want until its newest frame", out.Overlays[0].During)
	}
	if !strings.Contains(out.Note, "model") || !strings.Contains(out.Note, "not radar") {
		t.Errorf("the note is %q; want it said that the hours ahead are a model's rain, not radar", out.Note)
	}
	fc := out.Overlays[1]
	img := fc.Image
	if !strings.HasPrefix(fc.ID, tty.RadarLayer+"/fc-") || len(img.Table) == 0 || !img.Exact || fc.During.From != img.Frames[0].Valid {
		t.Fatalf("the forecast loop %s: table of %d, exact %v, during %v", fc.ID, len(img.Table), img.Exact, fc.During)
	}
	var times []string
	for _, f := range img.Frames {
		if !f.Forecast {
			t.Error("a frame of the hours ahead is not marked forecast")
		}
		times = append(times, f.Valid.Format("15:04"))
	}
	if strings.Join(times, " ") != "21:00 22:00 23:00" {
		t.Errorf("the frames are at %v; want each hour after the newest observed, to the horizon", times)
	}
	onTheHour := time.Date(2026, 9, 27, 21, 0, 0, 0, time.UTC) // a newest frame on the hour: that hour is observed, never forecast too
	again := withModelRain(context.Background(), tty.MapRadar{Overlays: []tuimaps.Overlay{observed}}, om, geo.RegionHawaii, geo.Box{}, onTheHour, onTheHour.Add(2*time.Hour))
	if first := again.Overlays[len(again.Overlays)-1].Image.Frames[0].Valid; !first.After(onTheHour) {
		t.Errorf("with the newest observed frame at 21:00 the hours ahead begin at %v; want after it", first.Format("15:04"))
	}
	pic, err := png.Decode(bytes.NewReader(img.Frames[0].PNG))
	if err != nil {
		t.Fatal(err)
	}
	b := pic.Bounds()
	r, g, bl, _ := pic.At(b.Dx()/2, b.Dy()/2).RGBA()
	want := radar.DBZOfRate(5)
	for _, e := range img.Table {
		if uint32(e.Colour.R)*0x101 == r && uint32(e.Colour.G)*0x101 == g && uint32(e.Colour.B)*0x101 == bl {
			if math.Abs(e.Value-want) > 0.5 {
				t.Errorf("5 mm an hour is painted as %v dBZ; want %.1f", e.Value, want)
			}
			return
		}
	}
	t.Error("the frame's colour is not in its table")
}

// TestTheHoursAheadNeverDrawADryHour: a dry hour paints nothing - no echo,
// transparent, never radar's lightest class.
func TestTheHoursAheadNeverDrawADryHour(t *testing.T) {
	f := temperature.Field{Cols: 2, Rows: 1, Values: []float64{0, math.NaN()}}
	pic, err := png.Decode(bytes.NewReader(ratePNG(f)))
	if err != nil {
		t.Fatal(err)
	}
	for x := range 2 {
		if _, _, _, a := pic.At(x, 0).RGBA(); a != 0 {
			t.Errorf("pixel %d is painted; a dry or unknown hour is transparent", x)
		}
	}
}

// TestAnUnboundedRateIsTheHeaviestClass: a rate whose reflectivity runs past
// the scale - huge, or infinite - paints the scale's heaviest colour, the
// conversion to a palette index held to the palette.
func TestAnUnboundedRateIsTheHeaviestClass(t *testing.T) {
	f := temperature.Field{Cols: 3, Rows: 1, Values: []float64{1e308, math.Inf(1), math.MaxFloat64}}
	pic, err := png.Decode(bytes.NewReader(ratePNG(f)))
	if err != nil {
		t.Fatal(err)
	}
	want := rateColour(255)
	for x := range 3 {
		r, g, b, a := pic.At(x, 0).RGBA()
		if a == 0 || r != uint32(want.R)*0x101 || g != uint32(want.G)*0x101 || b != uint32(want.B)*0x101 {
			t.Errorf("a rate of %v paints %v %v %v %v; want the heaviest class", f.Values[x], r>>8, g>>8, b>>8, a>>8)
		}
	}
}

// rainDaysGet is a week of rain: day 1 an inch of rain, day 2 two inches of
// snow, at the heaviest 5 mm an hour.
func rainDaysGet() *omRain {
	g := &omRain{start: time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), hours: 7 * 24, rate: 5}
	g.rain[0], g.snow[1], g.rain[1] = 25.4, 5.08, 1
	return g
}

// TestForecastModeDrawsEachDaysRainWithItsTotals is W12.3 (D-116, D-118):
// Now's hour and each day's heaviest hour in radar's scale, each during its
// step; each day's total marked on it - rain, or snow marked apart - in the
// listener's units; Open-Meteo credited.
func TestForecastModeDrawsEachDaysRainWithItsTotals(t *testing.T) {
	ask := tempAsk(true)
	ask.Fahrenheit = true
	out := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(rainDaysGet(), ""), ask, tempNow, nil)
	boxes := len(fieldBoxes(ask.Region, ask.View))
	if len(out.Rain) != boxes*(1+temperature.Days) {
		t.Fatalf("%d rain grids for %d boxes; want Now and seven days each", len(out.Rain), boxes)
	}
	steps := tty.ForecastSteps(ask.Anchor)
	marks := map[string]bool{}
	for _, o := range out.Rain {
		g := o.Grid
		if g.Type.Preset != "radar" || g.Type.Unit != "dBZ" {
			t.Fatalf("%s is typed %+v; want radar's scale", o.ID, g.Type)
		}
		step := 0
		if i := strings.Index(o.ID, "/d"); i >= 0 {
			step = int(o.ID[i+2]-'0') + 1
		}
		if o.During != steps[step].Span {
			t.Errorf("%s is drawn during %v; want its step's %v", o.ID, o.During, steps[step].Span)
		}
		if step == 0 && g.Marks != nil {
			t.Errorf("Now's rain carries totals: it is an hour, not a day")
		}
		for _, m := range g.Marks {
			if m != "" {
				marks[o.ID[strings.LastIndex(o.ID, "/")+1:]+" "+m] = true
			}
		}
	}
	if !marks["d0 1.0in"] || !marks["d1 *2.0in"] {
		t.Errorf("the totals marked are %v; want an inch of rain on day 1 and two of snow, marked apart, on day 2", marks)
	}
	if got := strings.Join(out.Chips[tty.RainLayer], "/"); got != "O-METEO" {
		t.Errorf("the rain's badge names %q; want Open-Meteo (D-133)", got)
	}
}

// TestATotalIsMarkedInTheListenersUnits is D-116's marks: rain in inches or
// mm, snow in inches or cm with a * before it; a day with snow says its snow;
// a trace says nothing.
func TestATotalIsMarkedInTheListenersUnits(t *testing.T) {
	for _, c := range []struct {
		rainMM, snowCM float64
		imperial       bool
		want           string
	}{
		{25.4, 0, true, "1.0in"}, {6.35, 0, true, ".25in"}, {0.1, 0, true, ""},
		{12.4, 0, false, "12mm"}, {2.5, 0, false, "2.5mm"}, {0.1, 0, false, ""},
		{1, 5.08, true, "*2.0in"}, {1, 30.48, true, "*12in"}, {1, 4, false, "*4.0cm"}, {1, 0.1, false, "1.0mm"},
		{math.NaN(), math.NaN(), true, ""},
	} {
		if got := totalMark(c.rainMM, c.snowCM, c.imperial); got != c.want {
			t.Errorf("%v mm of rain, %v cm of snow (imperial %v): %q; want %q", c.rainMM, c.snowCM, c.imperial, got, c.want)
		}
	}
}

// TestTheRainIsCostedInItsOwnMode is D-117 with D-23: the rain and snow row
// costs a request a box in Forecast mode and nothing in Radar mode, where
// the hours ahead outside the lower 48 are the radar's own cost.
func TestTheRainIsCostedInItsOwnMode(t *testing.T) {
	in := mapInputs{region: geo.RegionHawaii, forecast: true}
	if b, r := rainLayerCost(in); r != 1 || b != rainDaysBytes {
		t.Errorf("Forecast mode's rain costs %d bytes in %d requests; want one box's week", b, r)
	}
	in.forecast = false
	if b, r := rainLayerCost(in); b != 0 || r != 0 {
		t.Errorf("Radar mode's rain row costs %d bytes in %d requests; want none", b, r)
	}
}

// ndfdTotalsGet answers NDFD's six-hour rain and snow for any lattice asked,
// every point the same: thirteen periods from 18:00 UTC on tempNow's day - one
// on day 0, four on each of days 1 to 3 - each a tenth of an inch of rain,
// and on day 2 half an inch of snow each.
type ndfdTotalsGet struct{ asks int }

func (n *ndfdTotalsGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	n.asks++
	q, _ := url.ParseQuery(rawURL[strings.Index(rawURL, "?")+1:])
	start := time.Date(2026, 8, 24, 18, 0, 0, 0, time.UTC)
	var b strings.Builder
	b.WriteString(`<dwml><data>`)
	pts := strings.Fields(q.Get("listLatLon"))
	for i, p := range pts {
		ll := strings.Split(p, ",")
		b.WriteString(`<location><location-key>point` + strconv.Itoa(i+1) + `</location-key><point latitude="` + ll[0] + `" longitude="` + ll[1] + `"/></location>`)
	}
	b.WriteString(`<time-layout><layout-key>k-p6h-n13-1</layout-key>`)
	for k := range 13 {
		s := start.Add(time.Duration(6*k) * time.Hour)
		b.WriteString(`<start-valid-time>` + s.Format(time.RFC3339) + `</start-valid-time><end-valid-time>` + s.Add(6*time.Hour).Format(time.RFC3339) + `</end-valid-time>`)
	}
	b.WriteString(`</time-layout>`)
	for i := range pts {
		b.WriteString(`<parameters applicable-location="point` + strconv.Itoa(i+1) + `"><precipitation type="liquid" units="inches" time-layout="k-p6h-n13-1">`)
		for range 13 {
			b.WriteString(`<value>0.10</value>`)
		}
		b.WriteString(`</precipitation><precipitation type="snow" units="inches" time-layout="k-p6h-n13-1">`)
		for k := range 13 {
			v := "0.00"
			if k >= 5 && k < 9 {
				v = "0.50"
			}
			b.WriteString(`<value>` + v + `</value>`)
		}
		b.WriteString(`</precipitation></parameters>`)
	}
	b.WriteString(`</data></dwml>`)
	return []byte(b.String()), nil
}

// WHEN OPEN-METEO REFUSES, NDFD'S TOTALS DRAW FORECAST MODE'S RAIN AND SNOW
// (W18.5, D-168, D-184): each day NDFD reaches - today and three more - its
// totals in their own scale, in mm, each during its day's step, its total
// marked on it (snow apart); Now has none - NDFD has no rate; the badge says
// NDFD and a note says whose the totals are. Nothing recorded, NDFD is asked
// once a box.
func TestNDFDsTotalsDrawTheRainWhenOpenMeteoRefuses(t *testing.T) {
	ask := tempAsk(true)
	nd := &ndfdTotalsGet{}
	out := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow,
		&rainRescue{ndfd: temperature.NewNDFD(nd, "")})
	boxes := len(fieldBoxes(ask.Region, ask.View))
	if len(out.Rain) != boxes*4 || nd.asks != boxes {
		t.Fatalf("%d grids from %d asks for %d boxes; want today and three days each, NDFD asked once a box", len(out.Rain), nd.asks, boxes)
	}
	steps := tty.ForecastSteps(ask.Anchor)
	snow := false
	for _, o := range out.Rain {
		if o.Grid.Type.Preset != "qpf" || o.Grid.Type.Unit != "mm" {
			t.Fatalf("%s is typed %+v; want the totals' own scale in mm (D-184)", o.ID, o.Grid.Type)
		}
		k := int(o.ID[len(o.ID)-1] - '0')
		if o.During != steps[k+1].Span {
			t.Errorf("%s is drawn during %v; want day %d's step", o.ID, o.During, k)
		}
		if k == 1 && math.Abs(o.Grid.Values[0]-4*0.1*25.4) > 1e-6 {
			t.Errorf("day 1's total is %v mm; want four periods' %v", o.Grid.Values[0], 4*0.1*25.4)
		}
		snow = snow || (k == 2 && slices.Contains(o.Grid.Marks, "*2.0in"))
	}
	if !snow {
		t.Error("day 2's two inches of snow are not marked apart")
	}
	if got := strings.Join(out.Chips[tty.RainLayer], "/"); got != "NDFD" {
		t.Errorf("the rain's badge names %q; want NDFD", got)
	}
	if !slices.ContainsFunc(out.LayerNotes[tty.RainLayer], func(n string) bool { return strings.Contains(n, "NDFD") }) {
		t.Errorf("no rain note says the totals are NDFD's: %v", out.LayerNotes)
	}
}

// OPEN-METEO'S RAIN DAYS ARE RECORDED, AND REPLAYED (W18.5, D-168, D-192):
// each day it answered goes into the history; refused, NDFD's totals draw
// today and three days on, and the days past them draw as they were recorded
// - radar's scale, their totals marked - the badge NDFD then RECORDED.
// Without NDFD, every recorded day is drawn.
func TestOpenMeteosRainDaysAreRecordedAndReplayed(t *testing.T) {
	ask := tempAsk(true)
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, omRainDays)
	withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(rainDaysGet(), ""), ask, tempNow, &rainRescue{store: store})
	boxes := len(fieldBoxes(ask.Region, ask.View))
	alone := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow, &rainRescue{store: store})
	if len(alone.Rain) != boxes*temperature.Days || strings.Join(alone.Chips[tty.RainLayer], "/") != "RECORDED" {
		t.Fatalf("without NDFD, %d grids, chips %v; want the seven recorded days each, RECORDED", len(alone.Rain), alone.Chips[tty.RainLayer])
	}
	marks := map[string]bool{}
	for _, o := range alone.Rain {
		if o.Grid.Type.Preset != "radar" {
			t.Fatalf("%s is typed %+v; a recorded day draws as it did", o.ID, o.Grid.Type)
		}
		for _, m := range o.Grid.Marks {
			if m != "" {
				marks[o.ID[strings.LastIndex(o.ID, "/")+1:]+" "+m] = true
			}
		}
	}
	if !marks["d0 1.0in"] || !marks["d1 *2.0in"] {
		t.Errorf("the recorded totals marked are %v; want day 1's inch of rain and day 2's two of snow", marks)
	}
	nd := &ndfdTotalsGet{}
	both := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow,
		&rainRescue{ndfd: temperature.NewNDFD(nd, ""), store: store})
	qpf, radar := 0, 0
	for _, o := range both.Rain {
		switch o.Grid.Type.Preset {
		case "qpf":
			qpf++
		case "radar":
			radar++
		}
	}
	if qpf != boxes*4 || radar != boxes*(temperature.Days-4) || strings.Join(both.Chips[tty.RainLayer], "/") != "NDFD/RECORDED" {
		t.Errorf("with NDFD: %d of its days, %d recorded, chips %v; want four and three a box, NDFD then RECORDED", qpf, radar, both.Chips[tty.RainLayer])
	}
	cold := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow, nil)
	if len(cold.Rain) != 0 || len(cold.Problems) == 0 {
		t.Error("with nothing to fall back on, a refused rain drew something, or said nothing")
	}
}

// NDFD'S TOTALS ARE THE RAIN'S FIRST DAYS, OPEN-METEO THE REST ON A COARSER
// GRID (D-185, D-187, D-192): Open-Meteo answering, today and three days on
// are NDFD's totals in their scale; Now and the days past are Open-Meteo's,
// asked on a quarter of a box's points - all of them where the listener chose
// the full density. The badge names both.
func TestNDFDsTotalsAreTheRainsFirstDays(t *testing.T) {
	ask := tempAsk(true)
	boxes := len(fieldBoxes(ask.Region, ask.View))
	for _, full := range []bool{false, true} {
		ask.RainFull = full
		om := rainDaysGet()
		got := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(om, ""), ask, tempNow, &rainRescue{ndfd: temperature.NewNDFD(&ndfdTotalsGet{}, "")})
		qpf, radar := 0, 0
		for _, o := range got.Rain {
			switch o.Grid.Type.Preset {
			case "qpf":
				qpf++
			case "radar":
				radar++
			}
		}
		if qpf != boxes*4 || radar != boxes*(1+temperature.Days-4) {
			t.Errorf("full %v: %d of NDFD's days and %d of Open-Meteo's grids; want four, and Now with three days, a box", full, qpf, radar)
		}
		if chips := strings.Join(got.Chips[tty.RainLayer], "/"); chips != "O-METEO/NDFD" {
			t.Errorf("full %v: the badge names %q; want both", full, chips)
		}
		most := 0
		for _, a := range om.asks {
			q, _ := url.ParseQuery(a[strings.Index(a, "?")+1:])
			most = max(most, len(strings.Split(q.Get("latitude"), ",")))
		}
		if want := temperature.MaxPoints / 4; !full && (most > want || most < 4) {
			t.Errorf("coarse, Open-Meteo was asked for %d points a box; want at most %d (D-192)", most, want)
		}
		if full && most <= temperature.MaxPoints/4 {
			t.Errorf("full, Open-Meteo was asked for %d points a box; want the box's full lattice", most)
		}
	}
	ask.RainFull = false
	down := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(rainDaysGet(), ""), ask, tempNow, &rainRescue{ndfd: temperature.NewNDFD(refusingGet{}, "")})
	if len(down.Rain) != boxes*(1+temperature.Days) || strings.Join(down.Chips[tty.RainLayer], "/") != "O-METEO" {
		t.Errorf("NDFD refusing: %d grids, chips %v; want Open-Meteo's Now and seven days a box, O-METEO alone", len(down.Rain), down.Chips[tty.RainLayer])
	}
}
