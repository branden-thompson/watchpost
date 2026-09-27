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
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
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
	out := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(rainDaysGet(), ""), ask, tempNow)
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
	if len(out.RainNotes) == 0 || !strings.Contains(out.RainNotes[0], "Open-Meteo") {
		t.Errorf("the rain's notes are %v; want Open-Meteo credited", out.RainNotes)
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
	in.ahead = 3
	withAhead, _ := radarLayerCost(in)
	in.ahead = 0
	without, _ := radarLayerCost(in)
	if withAhead-without != rainHoursBytes {
		t.Errorf("Hawaii's hours ahead cost %d bytes; want Open-Meteo's request, not HRRR's frames", withAhead-without)
	}
}
