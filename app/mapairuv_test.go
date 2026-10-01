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
	"math"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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
	return []byte("[" + fill(point, n) + "]"), nil
}

// everyBox and noBox say Open-Meteo's forecast was asked for every box, or
// none: where UV's grid may ride (D-191).
func everyBox(string) bool { return true }
func noBox(string) bool    { return false }

// OPEN-METEO'S UV GRID RIDES ONLY WHERE ITS FORECAST IS ASKED (D-137,
// D-191): asked for nothing else, UV asks nothing of Open-Meteo - with no EPA
// source, it draws nothing; asked, Radar mode's every hour and Forecast
// mode's Now and each day's highest, as UV grids, named O-METEO.
func TestUVIsItsGridsInEachMode(t *testing.T) {
	get := &omGet{}
	om := temperature.NewOpenMeteo(get, "")
	ask := tempAsk(false)
	ask.UV = true
	if got := withUV(context.Background(), tty.MapTemperature{}, om, noBox, ask, tempNow, nil, nil); len(got.UV) != 0 || len(get.asked) != 0 || len(got.Chips[tty.UVLayer]) != 0 {
		t.Fatalf("Open-Meteo asked for nothing else: %d UV grids, %d requests, chips %v; want none (D-191)", len(got.UV), len(get.asked), got.Chips[tty.UVLayer])
	}
	radar := withUV(context.Background(), tty.MapTemperature{}, om, everyBox, ask, tempNow, nil, nil)
	if len(radar.UV) == 0 || radar.UV[0].Grid.Type.Preset != "uv" || radar.UV[0].Grid.Values[0] != 5 || len(radar.UVDays) != 0 {
		t.Fatalf("Radar mode's UV is %d hours, %d days", len(radar.UV), len(radar.UVDays))
	}
	if got := radar.Chips[tty.UVLayer]; len(got) != 1 || got[0] != "O-METEO" {
		t.Errorf("drawn, UV's chips are %v; want Open-Meteo's (D-183)", got)
	}
	ask = tempAsk(true)
	fc := withUV(context.Background(), tty.MapTemperature{}, om, everyBox, ask, tempNow, nil, nil) // held while the map is open, whatever the row (D-99)
	if len(fc.UV) != len(fieldBoxes(ask.Region, ask.View)) || len(fc.UVDays) == 0 || fc.UVDays[0].Grid.Values[0] != 8 {
		t.Errorf("Forecast mode's UV is %d Now grids and %d days; want Now a box and each day's highest, 8", len(fc.UV), len(fc.UVDays))
	}
}

// TestTheAirIsTheModelsGridsAskedWhileOn is D-139: the model's US AQI from
// contoursFile answers AirNow's contours from their fixture, counting asks.
type contoursFile struct{ asked []string }

func (g *contoursFile) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	g.asked = append(g.asked, rawURL)
	return os.ReadFile("../domains/airquality/testdata/cur_aqi_combined.kml")
}

// AIR QUALITY'S TINT IS AIRNOW'S CONTOURS (W19.4, D-193): with its row on,
// each field box is a grid in the AQI scale, each cell its contour's
// category - Fresno's Unhealthy, Los Angeles' Moderate - none where no
// contour reaches; in Forecast mode on Now alone. With the row off nothing
// is asked, and Open-Meteo never is.
func TestTheAirIsAirNowsContours(t *testing.T) {
	get := &contoursFile{}
	airnow := airquality.New(get, "")
	ask := tempAsk(false)
	ask.View = tty.MapView{W: -124.5, S: 32.5, E: -114, N: 42}
	now := time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC)
	if got := withAir(context.Background(), tty.MapTemperature{}, airnow, ask, now); len(got.Air) != 0 || len(get.asked) != 0 {
		t.Fatalf("with the row off: %d grids, %d asks; want none", len(got.Air), len(get.asked))
	}
	ask.Air = true
	got := withAir(context.Background(), tty.MapTemperature{}, airnow, ask, now)
	if len(got.Air) != len(fieldBoxes(ask.Region, geo.Box(ask.View))) || len(got.AirDays) != 0 {
		t.Fatalf("the air is %d grids, %d days; want a grid a box, no days", len(got.Air), len(got.AirDays))
	}
	at := func(lat, lon float64) float64 {
		for _, o := range got.Air {
			g := o.Grid
			if g.Type.Preset != "aqi" || lon < g.West || lon > g.East || lat < g.South || lat > g.North {
				continue
			}
			col := min(int((lon-g.West)/(g.East-g.West)*float64(g.Cols)), g.Cols-1)
			row := min(int((g.North-lat)/(g.North-g.South)*float64(g.Rows)), g.Rows-1)
			return g.Values[row*g.Cols+col]
		}
		return -1
	}
	if v := at(36.35, -119.25); v != airquality.CategoryAQI(airquality.Unhealthy) {
		t.Errorf("Fresno's south-east is %v; want Unhealthy's %v", v, airquality.CategoryAQI(airquality.Unhealthy))
	}
	if v := at(34.05, -118.25); v != airquality.CategoryAQI(airquality.Moderate) {
		t.Errorf("Los Angeles is %v; want Moderate's", v)
	}
	if v := at(39, -117); !math.IsNaN(v) {
		t.Errorf("Nevada, outside every contour, is %v; want nothing", v)
	}
	for _, u := range get.asked {
		if strings.Contains(u, "open-meteo") {
			t.Errorf("asked %s: Open-Meteo's air is not asked (D-193)", u)
		}
	}
	if notes := got.LayerNotes[tty.AirLayer]; len(notes) != 1 || !strings.Contains(notes[0], "current hour") {
		t.Errorf("Radar mode's air note is %v; want one saying the fill is the current hour's (D-203)", notes)
	}
	fc := ask
	fc.Forecast = true
	steps := tty.ForecastSteps(askAnchor(fc, now))
	fcGot := withAir(context.Background(), tty.MapTemperature{}, airnow, fc, now)
	for _, o := range fcGot.Air {
		if o.During != steps[0].Span {
			t.Errorf("Forecast mode's contours are drawn during %v; want Now's step alone", o.During)
		}
	}
	// THE FORECAST DAYS SAY WHY THEY ARE NOT FILLED (U2-52, D-203).
	note := strings.Join(fcGot.LayerNotes[tty.AirLayer], " ")
	for _, want := range []string{"current hour", "Today and Tomorrow", "past Tomorrow"} {
		if !strings.Contains(note, want) {
			t.Errorf("Forecast mode's air note %q does not say %q", note, want)
		}
	}
	if len(fcGot.Notes) != 0 {
		t.Errorf("the air's note is among temperature's, which it turns off: %v", fcGot.Notes)
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

// TestAirQualityCostsItsRequests is D-139 with FR-9.2 and D-193: AirNow's two
// files, its reporting areas and its contours, whatever the view.
func TestAirQualityCostsItsRequests(t *testing.T) {
	in := mapInputs{region: tempAsk(false).Region, view: tty.MapView{W: -125, S: 24, E: -66, N: 50}}
	b, r := airLayerCost(in)
	if r != 2 || b < airnowBytes+contoursBytes {
		t.Errorf("air quality costs %d bytes in %d requests; want AirNow's two files", b, r)
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
	withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), everyBox, ask, tempNow, store, nil)
	for back := 0; back <= pastHours; back++ {
		rec, ok := store.Get(omUVHourly.Name, history.Key{Source: "openmeteo", Place: box.Name}, anchor.Add(-time.Duration(back)*time.Hour))
		if !ok || rec.Values["uv"][0] != 5 {
			t.Errorf("the UV %d hours before was not recorded: %v %v", back, ok, rec.Values)
		}
	}
	if _, ok := store.Get(omUVHourly.Name, history.Key{Source: "openmeteo", Place: box.Name}, anchor.Add(time.Hour)); ok {
		t.Error("an hour ahead was recorded: only what was valid is kept")
	}
	got := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), everyBox, ask, tempNow, store, nil)
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
	if none := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), everyBox, ask, tempNow, nil, nil); len(none.UV) != 0 || len(none.Chips[tty.UVLayer]) != 0 {
		t.Error("with no history, a refused UV drew something")
	}
}

// epaFixture answers every EPA ask with the recorded Vista forecast.
type epaFixture struct{ asked int }

func (e *epaFixture) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	e.asked++
	return os.ReadFile(filepath.Join("..", "domains", "uv", "testdata", "epa-vista.json"))
}

// UV IS EPA'S FOR THE CITIES IN VIEW FIRST (D-167, D-186, D-191): with UV on,
// the largest cities in view are drawn as markers in their bands' colours,
// labelled with the city and the value - in Radar mode the current hour and
// the three before, each in its own hour; in Forecast mode Now's hour on Now
// and the day's peak on Today - the badge EPA and a note saying whose it is;
// Open-Meteo's grid beside them only where its forecast was asked, named after
// EPA; with UV off, no city is asked.
func TestUVIsEPAsForTheCitiesInViewFirst(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	now := time.Date(2026, 9, 30, 12, 20, 0, 0, la) // the fixture's day: UV 7 at noon
	ask := tempAsk(false)
	ask.UV, ask.Anchor, ask.UVCities = true, now.Truncate(time.Hour), 48
	get := &epaFixture{}
	var counts []int // the count each choice of cities was asked for (D-202)
	cities := &uvCities{epa: uv.NewEPA(get, ""), cities: func(_ geo.Box, n int) []geodata.City {
		counts = append(counts, n)
		return []geodata.City{{Name: "Vista", State: "CA", Country: "US", Lat: 33.2, Lon: -117.24, Population: 100000, TZ: "America/Los_Angeles"}}
	}}
	labels := func(got tty.MapTemperature) map[string]tuimaps.Span {
		out := map[string]tuimaps.Span{}
		for _, o := range got.UV {
			for _, f := range o.Features {
				out[f.Label] = o.During
				if f.Label == "Vista 7" && (f.Kind != tuimaps.Point || f.Role != tuimaps.UVRole(7)) {
					t.Errorf("Vista is drawn as %v in %v; want a point in UV band %v", f.Kind, f.Role, tuimaps.UVRole(7))
				}
			}
		}
		return out
	}
	om := &omGet{}
	got := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(om, ""), noBox, ask, now, nil, cities)
	during := labels(got)
	for label, from := range map[string]time.Time{"Vista 7": ask.Anchor, "Vista 5": ask.Anchor.Add(-time.Hour), "Vista 1": ask.Anchor.Add(-3 * time.Hour)} {
		if sp := during[label]; !sp.From.Equal(from) || !sp.Until.Before(from.Add(time.Hour)) || sp.Until.Before(from.Add(time.Hour-time.Second)) {
			t.Errorf("%s is drawn during %v; want its own hour from %v, as the replay is", label, sp, from)
		}
	}
	if len(counts) == 0 || counts[0] != 48 {
		t.Errorf("the cities were chosen for counts %v; want the listener's 48 (D-202)", counts)
	}
	if chips := got.Chips[tty.UVLayer]; !slices.Equal(chips, []string{"EPA"}) || len(om.asked) != 0 {
		t.Errorf("the UV badge says %v, Open-Meteo asked %d times; want EPA alone, Open-Meteo not asked (D-191)", chips, len(om.asked))
	}
	if !slices.ContainsFunc(got.LayerNotes[tty.UVLayer], func(n string) bool { return strings.Contains(n, "EPA") }) {
		t.Errorf("no UV note says the UV is EPA's: %v", got.LayerNotes)
	}
	both := withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), everyBox, ask, now, nil, cities)
	if chips := both.Chips[tty.UVLayer]; !slices.Equal(chips, []string{"EPA", "O-METEO"}) {
		t.Errorf("with Open-Meteo's forecast asked, the badge says %v; want EPA then O-METEO", chips)
	}
	fc := ask
	fc.Forecast, fc.Anchor = true, ask.Anchor.Add(-2*time.Hour)
	onStep := map[tuimaps.Span]string{}
	for _, o := range withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), noBox, fc, now.Add(-2*time.Hour), nil, cities).UV {
		for _, f := range o.Features {
			onStep[o.During] = f.Label
		}
	}
	fcSteps := tty.ForecastSteps(fc.Anchor)
	if onStep[fcSteps[0].Span] != "Vista 3" || onStep[fcSteps[1].Span] != "Vista 7" {
		t.Errorf("Forecast mode at 10 AM draws %v; want Now's hour, 3, on Now and the day's peak, 7, on Today", onStep)
	}
	asked := get.asked
	off := ask
	off.UV = false
	if withUV(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(&omGet{}, ""), noBox, off, now, nil, cities); get.asked != asked {
		t.Error("with UV off, EPA was asked")
	}
}

// UV'S GRID RIDES WHERE OPEN-METEO'S FORECAST WAS ASKED (D-191): where it is
// the source, every box; as NDFD's filler, the boxes it answered for, and not
// one it was asked for and refused.
func TestUVsGridRidesWhereOpenMeteoAnswered(t *testing.T) {
	if !uvAsked(true, &answeredFor{boxes: map[string]bool{}})("anywhere") {
		t.Error("Open-Meteo the source, a box was not counted as asked")
	}
	ok := &answeredFor{Source: &fakeTemp{name: "Open-Meteo", now: tempNow}, boxes: map[string]bool{}}
	refused := &answeredFor{Source: &fakeTemp{name: "Open-Meteo", now: tempNow, failed: true}, boxes: map[string]bool{}}
	lat := temperature.Lattice{Name: "hi", Box: geo.Box{W: -160, S: 18, E: -154, N: 23}, Cols: 2, Rows: 2}
	if _, err := ok.Fetch(context.Background(), lat, tempNow); err != nil {
		t.Fatal(err)
	}
	if _, err := refused.Fetch(context.Background(), lat, tempNow); err == nil {
		t.Fatal("the refusing stand-in answered")
	}
	if !uvAsked(false, ok)("hi") || uvAsked(false, ok)("elsewhere") || uvAsked(false, refused)("hi") {
		t.Error("as NDFD's filler, the boxes counted are not exactly those Open-Meteo answered for")
	}
}
