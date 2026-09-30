package app

// mapairuv_test.go — 0.18.0 D-137 to D-140: UV and air quality, from
// Open-Meteo's answers and AirNow's file, asked only while on.

import (
	"context"
	"errors"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	if got := withUV(context.Background(), tty.MapTemperature{}, om, false, ask, tempNow, nil, nil); len(got.UV) != 0 || len(get.asked) != 0 {
		t.Fatalf("with UV off and NDFD the source, %d UV grids and %d requests; want none", len(got.UV), len(get.asked))
	}
	ask.UV = true
	radar := withUV(context.Background(), tty.MapTemperature{}, om, false, ask, tempNow, nil, nil)
	if len(radar.UV) == 0 || radar.UV[0].Grid.Type.Preset != "uv" || radar.UV[0].Grid.Values[0] != 5 || len(radar.UVDays) != 0 {
		t.Fatalf("Radar mode's UV is %d hours, %d days", len(radar.UV), len(radar.UVDays))
	}
	if got := radar.Chips[tty.UVLayer]; len(got) != 1 || got[0] != "O-METEO" {
		t.Errorf("drawn, UV's chips are %v; want Open-Meteo's (D-183)", got)
	}
	ask.UV = false
	if got := withUV(context.Background(), tty.MapTemperature{}, om, false, ask, tempNow, nil, nil).Chips[tty.UVLayer]; len(got) != 0 {
		t.Errorf("drawing nothing, UV names %v (D-183)", got)
	}
	ask.UV = true
	ask = tempAsk(true)
	fc := withUV(context.Background(), tty.MapTemperature{}, om, true, ask, tempNow, nil, nil) // Open-Meteo the source: free, whatever the row
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
	overlays, times := airnowOverlays(areas, ask.View, anchor, time.Now())
	if len(overlays) == 0 || overlays[0].ID != tty.AirLayer+"/airnow" || !times[overlays[0].ID].Happened {
		t.Fatalf("the monitors are %+v; want the measured, drawn now", overlays)
	}
	f := overlays[0].Features[0]
	if f.Label != "Anchorage 11" || f.Role != tuimaps.AirQualityRole(11) || overlays[0].Credit == "" {
		t.Errorf("Anchorage is %q in %v, credit %q; want \"Anchorage 11\" in Good's colour, credited", f.Label, f.Role, overlays[0].Credit)
	}
	california, caTimes := airnowOverlays(areas, tty.MapView{W: -125, S: 32, E: -114, N: 42}, anchor, time.Now())
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

// refusingGet is Open-Meteo refusing: every ask a spent quota.
type refusingGet struct{}

func (refusingGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	return nil, errors.New("Daily API request limit exceeded")
}

// VALID UV GOES INTO THE HISTORY, AND IS REPLAYED WHEN OPEN-METEO REFUSES
// (W18.4, D-167): each hour Open-Meteo answered, up to the current one,
// recorded once; refused, Radar mode draws the hours recorded - the three
// before the current one and the current one - each in its own hour, and the
// UV badge says RECORDED alone (its source did not answer).
func TestUVIsRecordedAndReplayed(t *testing.T) {
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, omUVHourly)
	anchor := tempNow.Truncate(time.Hour)
	ask := tempAsk(false)
	ask.UV = true
	box := fieldBoxes(ask.Region, ask.View)[0]
	withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), false, ask, tempNow, store, nil)
	for back := 0; back <= pastHours; back++ {
		rec, ok := store.Get(omUVHourly.Name, history.Key{Source: "openmeteo", Place: box.Name}, anchor.Add(-time.Duration(back)*time.Hour))
		if !ok || rec.Values["uv"][0] != 5 {
			t.Errorf("the UV %d hours before was not recorded: %v %v", back, ok, rec.Values)
		}
	}
	if _, ok := store.Get(omUVHourly.Name, history.Key{Source: "openmeteo", Place: box.Name}, anchor.Add(time.Hour)); ok {
		t.Error("an hour ahead was recorded: only what was valid is kept")
	}
	got := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), false, ask, tempNow, store, nil)
	hours := map[time.Time]bool{}
	for _, o := range got.UV {
		if strings.Contains(o.ID, "/"+box.Name+"/") {
			hours[o.During.From] = true
		}
	}
	for back := 0; back <= pastHours; back++ {
		if !hours[anchor.Add(-time.Duration(back)*time.Hour)] {
			t.Errorf("the recorded UV %d hours before was not replayed: %v", back, hours)
		}
	}
	if chips := got.Chips[tty.UVLayer]; !slices.Equal(chips, []string{"RECORDED"}) {
		t.Errorf("UV replayed says %v; want RECORDED alone", chips)
	}
	if none := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), false, ask, tempNow, nil, nil); len(none.UV) != 0 || len(none.Chips[tty.UVLayer]) != 0 {
		t.Error("with no history, a refused UV drew something")
	}
}

// epaFixture answers every EPA ask with the recorded Vista forecast.
type epaFixture struct{ asked int }

func (e *epaFixture) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	e.asked++
	return os.ReadFile(filepath.Join("..", "domains", "uv", "testdata", "epa-vista.json"))
}

// A COLD START'S UV IS EPA'S, FOR THE CITIES IN VIEW (W18.4, D-167): Open-
// Meteo refusing and nothing recorded, the UV layer draws EPA's index for the
// largest cities in view as markers in their bands' colours, labelled with
// the city and the value; the badge says EPA and a note says whose it is.
// Anything recorded is drawn instead: the cold start is the last tier before
// the notice alone.
func TestAColdStartsUVIsEPAsForTheCitiesInView(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	now := time.Date(2026, 9, 30, 12, 20, 0, 0, la) // the fixture's day: UV 7 at noon
	ask := tempAsk(false)
	ask.UV, ask.Anchor = true, now.Truncate(time.Hour)
	get := &epaFixture{}
	cold := &uvCold{epa: uv.NewEPA(get, ""), cities: func(geo.Box) []geodata.City {
		return []geodata.City{{Name: "Vista", State: "CA", Country: "US", Lat: 33.2, Lon: -117.24, Population: 100000, TZ: "America/Los_Angeles"}}
	}}
	store := history.Open(t.TempDir(), func() time.Time { return now }, omUVHourly)
	got := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), false, ask, now, store, cold)
	var marker *tuimaps.Feature
	during := map[string]tuimaps.Span{}
	for _, o := range got.UV {
		for i := range o.Features {
			during[o.Features[i].Label] = o.During
			if o.Features[i].Label == "Vista 7" {
				marker = &o.Features[i]
			}
		}
	}
	if marker == nil {
		t.Fatalf("no marker for Vista's UV 7: %d UV overlays", len(got.UV))
	}
	for label, from := range map[string]time.Time{"Vista 7": ask.Anchor, "Vista 5": ask.Anchor.Add(-time.Hour), "Vista 1": ask.Anchor.Add(-3 * time.Hour)} {
		if sp := during[label]; !sp.From.Equal(from) || !sp.Until.Before(from.Add(time.Hour)) || sp.Until.Before(from.Add(time.Hour-time.Second)) {
			t.Errorf("%s is drawn during %v; want its own hour from %v, as the replay is", label, sp, from)
		}
	}
	if marker.Kind != tuimaps.Point || marker.Role != tuimaps.UVRole(7) {
		t.Errorf("Vista is drawn as %v in %v; want a point in UV band %v", marker.Kind, marker.Role, tuimaps.UVRole(7))
	}
	if chips := got.Chips[tty.UVLayer]; !slices.Equal(chips, []string{"EPA"}) {
		t.Errorf("the UV badge says %v; want EPA", chips)
	}
	if !slices.ContainsFunc(got.Notes, func(n string) bool { return strings.Contains(n, "EPA") }) {
		t.Errorf("no note says the UV is EPA's: %v", got.Notes)
	}
	recorded := history.Open(t.TempDir(), func() time.Time { return tempNow }, omUVHourly) // at Open-Meteo's fixture's hour
	was := tempAsk(false)
	was.UV = true
	withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), false, was, tempNow, recorded, nil)
	asked := get.asked
	replay := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), false, was, tempNow, recorded, cold)
	if len(replay.UV) == 0 || get.asked != asked || slices.Contains(replay.Chips[tty.UVLayer], "EPA") {
		t.Error("with UV recorded, EPA was asked: the history is drawn first (D-167)")
	}
	if none := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), false, ask, now, store, nil); len(none.UV) != 0 {
		t.Error("with no EPA source, a refused UV drew something")
	}
}

// THE COLD START ASKS FOR THE LARGEST CITIES IN VIEW, AT MOST maxUVCities
// (D-167): from the ranking, its order kept; a city outside the view, or with
// no zone to read EPA's hours in, is passed over.
func TestTheColdStartAsksTheLargestCitiesInView(t *testing.T) {
	view := geo.Box{W: -120, S: 30, E: -110, N: 40}
	ranked := []geodata.City{{Name: "Out", Lat: 45, Lon: -100, TZ: "America/Chicago"}, {Name: "NoZone", Lat: 35, Lon: -115}}
	for i := range maxUVCities + 2 {
		ranked = append(ranked, geodata.City{Name: "C" + strconv.Itoa(i), Lat: 35, Lon: -115, TZ: "America/Los_Angeles"})
	}
	got := largestInView(ranked, view)
	if len(got) != maxUVCities {
		t.Fatalf("%d cities asked for; want %d", len(got), maxUVCities)
	}
	for i, c := range got {
		if c.Name != "C"+strconv.Itoa(i) {
			t.Errorf("city %d is %s; want the ranking's C%d (none outside the view, none without a zone)", i, c.Name, i)
		}
	}
	if largestInView(nil, view) != nil {
		t.Error("no ranking asked for cities")
	}
}
