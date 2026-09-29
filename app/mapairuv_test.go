package app

// mapairuv_test.go — 0.18.0 D-137 to D-140: UV and air quality, from
// Open-Meteo's answers and AirNow's file, asked only while on.

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// omGet answers Open-Meteo's forecast and air-quality requests for however
// many points are asked: UV 5 every hour, 8 each day's highest; AQI 60
// every hour; and keeps each address.
type omGet struct{ asked []string }

func (g *omGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	g.asked = append(g.asked, rawURL)
	u, _ := url.Parse(rawURL)
	n := len(strings.Split(u.Query().Get("latitude"), ","))
	var hours, days []string
	for h := -3; h <= 12; h++ {
		hours = append(hours, `"`+tempNow.Truncate(time.Hour).Add(time.Duration(h)*time.Hour).Format("2006-01-02T15:04")+`"`)
	}
	for k := range temperature.Days {
		days = append(days, `"`+tempNow.AddDate(0, 0, k).Format("2006-01-02")+`"`)
	}
	fill := func(v string, count int) string { return strings.TrimSuffix(strings.Repeat(v+",", count), ",") }
	point := `{"utc_offset_seconds":0,"hourly":{"time":[` + strings.Join(hours, ",") + `],"uv_index":[` + fill("5", len(hours)) + `]},"daily":{"time":[` + strings.Join(days, ",") + `],"uv_index_max":[` + fill("8", len(days)) + `]}}`
	if strings.Contains(rawURL, "/v1/air-quality") {
		point = `{"utc_offset_seconds":0,"hourly":{"time":[` + strings.Join(hours, ",") + `],"us_aqi":[` + fill("60", len(hours)) + `]}}`
	}
	return []byte("[" + fill(point, n) + "]"), nil
}

// TestUVIsItsGridsInEachMode is D-137: Radar mode's every hour and Forecast
// mode's Now and each day's highest, as UV grids; asked only while UV is on
// where Open-Meteo is not temperature's source already.
func TestUVIsItsGridsInEachMode(t *testing.T) {
	get := &omGet{}
	om := temperature.NewOpenMeteo(get, "")
	ask := tempAsk(false)
	if got := withUV(context.Background(), tty.MapTemperature{}, om, false, ask, tempNow); len(got.UV) != 0 || len(get.asked) != 0 {
		t.Fatalf("with UV off and NDFD the source, %d UV grids and %d requests; want none", len(got.UV), len(get.asked))
	}
	ask.UV = true
	radar := withUV(context.Background(), tty.MapTemperature{}, om, false, ask, tempNow)
	if len(radar.UV) == 0 || radar.UV[0].Grid.Type.Preset != "uv" || radar.UV[0].Grid.Values[0] != 5 || len(radar.UVDays) != 0 {
		t.Fatalf("Radar mode's UV is %d hours, %d days", len(radar.UV), len(radar.UVDays))
	}
	ask = tempAsk(true)
	fc := withUV(context.Background(), tty.MapTemperature{}, om, true, ask, tempNow) // Open-Meteo the source: free, whatever the row
	if len(fc.UV) != len(fieldBoxes(ask.Region, ask.View)) || len(fc.UVDays) == 0 || fc.UVDays[0].Grid.Values[0] != 8 {
		t.Errorf("Forecast mode's UV is %d Now grids and %d days; want Now a box and each day's highest, 8", len(fc.UV), len(fc.UVDays))
	}
}

// TestTheAirIsTheModelsGridsAskedWhileOn is D-139: the model's US AQI from
// the air-quality API, asked only while Air quality is on.
func TestTheAirIsTheModelsGridsAskedWhileOn(t *testing.T) {
	get := &omGet{}
	om := temperature.NewOpenMeteo(get, "")
	ask := tempAsk(true)
	if got := withAir(context.Background(), tty.MapTemperature{}, om, ask, tempNow); len(got.Air) != 0 || len(get.asked) != 0 {
		t.Fatalf("with the row off: %d grids, %d requests; want none", len(got.Air), len(get.asked))
	}
	ask.Air = true
	got := withAir(context.Background(), tty.MapTemperature{}, om, ask, tempNow)
	if len(got.Air) == 0 || got.Air[0].Grid.Type.Preset != "aqi" || got.Air[0].Grid.Values[0] != 60 || len(got.AirDays) == 0 {
		t.Errorf("the air is %d Now grids, %d days", len(got.Air), len(got.AirDays))
	}
	for _, u := range get.asked {
		if !strings.HasPrefix(u, "https://air-quality-api.open-meteo.com/v1/air-quality?") {
			t.Errorf("asked %s; want the air-quality API", u)
		}
	}
}

// airnowFile answers AirNow's file from its fixture, counting the asks.
type airnowFile struct{ asked int }

func (g *airnowFile) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	g.asked++
	return os.ReadFile("../domains/airquality/testdata/reportingarea.dat")
}

// TestAirNowsMonitorsAreMarkersInTheirCategory is D-139: while Air quality
// is on, AirNow's reporting areas in view are markers in their AQI's
// category - labelled with name and AQI while few are in view - the measured
// drawn now, AirNow's forecast during today's and tomorrow's steps; with the
// row off, the file is not asked.
func TestAirNowsMonitorsAreMarkersInTheirCategory(t *testing.T) {
	get := &airnowFile{}
	lp := &livePipelines{airnow: airquality.New(get, "")}
	ask := tty.MapAsk{Region: "", View: tty.MapView{W: -150.5, S: 60.9, E: -149.3, N: 61.5}} // Anchorage
	if areas := lp.airnowIn(context.Background(), ask); areas != nil || get.asked != 0 {
		t.Fatalf("with the row off AirNow was asked %d times", get.asked)
	}
	ask.Air = true
	areas := lp.airnowIn(context.Background(), ask)
	anchor := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	overlays, times := airnowOverlays(areas, ask.View, anchor)
	if len(overlays) == 0 || overlays[0].ID != tty.AirLayer+"/airnow" || !times[overlays[0].ID].Happened {
		t.Fatalf("the monitors are %+v; want the measured, drawn now", overlays)
	}
	f := overlays[0].Features[0]
	if f.Label != "Anchorage 11" || f.Role != tuimaps.AirQualityRole(11) || overlays[0].Credit == "" {
		t.Errorf("Anchorage is %q in %v, credit %q; want \"Anchorage 11\" in Good's colour, credited", f.Label, f.Role, overlays[0].Credit)
	}
	california, caTimes := airnowOverlays(areas, tty.MapView{W: -125, S: 32, E: -114, N: 42}, anchor)
	for _, f := range california[0].Features {
		if f.Label != "" {
			t.Fatalf("over all of California's areas, more than %d, %q is labelled; want markers alone", airLabelMost, f.Label)
		}
	}
	steps := tty.ForecastSteps(anchor)
	if len(california) != 3 {
		t.Fatalf("California's monitors are %d overlays; want the measured, today's forecast and tomorrow's", len(california))
	}
	for day, o := range california[1:] {
		if tm := caTimes[o.ID]; tm.From != steps[day+1].Span.From || tm.Until != steps[day+1].Span.Until {
			t.Errorf("%s is timed %+v; want its day's step, %v", o.ID, tm, steps[day+1].Span)
		}
	}
}

// TestAirQualityCostsItsRequests is D-139 with FR-9.2: a request a field box
// and AirNow's file.
func TestAirQualityCostsItsRequests(t *testing.T) {
	in := mapInputs{region: tempAsk(false).Region, view: tty.MapView{W: -125, S: 24, E: -66, N: 50}}
	b, r := airLayerCost(in)
	if boxes := len(fieldBoxes(in.region, in.view)); r != boxes+1 || b < airnowBytes {
		t.Errorf("air quality costs %d bytes in %d requests; want a request a box (%d) and AirNow's file", b, r, boxes)
	}
}
