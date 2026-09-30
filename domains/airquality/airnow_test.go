package airquality

// airnow_test.go — 0.18.0 D-138: AirNow's reporting areas, their measured
// AQI and their forecasts, from the national file.

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// captured is when the fixture was recorded: 17:51 in California. The file
// is trimmed to California's, Alaska's and Puerto Rico's rows.
var captured = time.Date(2026, 9, 29, 0, 51, 0, 0, time.UTC)

type fileGet struct {
	t     *testing.T
	asked *[]string
}

func (g fileGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	*g.asked = append(*g.asked, rawURL)
	return os.ReadFile("testdata/reportingarea.dat")
}

func areasOf(t *testing.T) (map[string]Area, []string) {
	t.Helper()
	var asked []string
	list, err := New(fileGet{t, &asked}, "").Areas(context.Background(), captured)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Area{}
	for _, a := range list {
		out[a.Name+", "+a.State] = a
	}
	return out, asked
}

// TestTheAreasAreRead is D-138: one national file from AirNow's host; each
// area once, where it is, its measured AQI its primary pollutant's - never
// another pollutant's row, never yesterday's.
func TestTheAreasAreRead(t *testing.T) {
	areas, asked := areasOf(t)
	if len(asked) != 1 || asked[0] != "https://files.airnowtech.org/airnow/today/reportingarea.dat" {
		t.Errorf("asked %v; want AirNow's one file", asked)
	}
	if len(areas) != 167 {
		t.Errorf("%d areas; want the fixture's 167", len(areas))
	}
	a, ok := areas["Anchorage, AK"]
	if !ok || a.Now == nil || a.Now.AQI != 11 || a.Now.Category != "Good" || math.Abs(a.Lat-61.2167) > 1e-9 {
		t.Fatalf("Anchorage is %+v; want its primary pollutant's 11, Good - not PM2.5's 0, not yesterday's 24", a)
	}
}

// TestAForecastIsPlacedByItsDate is D-138: a forecast's offset counts from
// its issue, so yesterday's "tomorrow" is today; it is placed by its valid
// date against the area's own today, the latest issue kept; a forecast of a
// category alone is that category's floor.
func TestAForecastIsPlacedByItsDate(t *testing.T) {
	areas, _ := areasOf(t)
	av := areas["Antelope Vly, CA"]
	if av.Forecast[0].AQI != 45 || av.Forecast[1].AQI != 50 {
		t.Errorf("Antelope Valley's forecasts are %+v; want today 45 (issued yesterday as tomorrow), tomorrow 50", av.Forecast)
	}
	at := areas["Atascadero, CA"]
	if r := at.Forecast[1]; !math.IsNaN(r.AQI) || r.Category != "Good" || r.Value() != 0 {
		t.Errorf("Atascadero's tomorrow is %+v (value %v); want Good alone, its floor 0", r, r.Value())
	}
	if (Reading{AQI: math.NaN(), Category: "Unhealthy for Sensitive Groups"}).Value() != 101 {
		t.Error("a category alone is not its floor")
	}
	if !strings.Contains(Host(), "airnowtech.org") {
		t.Errorf("the host is %q", Host())
	}
}

// TestTheLatestIssueIsKept is D-138: a day forecast by two issues takes the
// later one's, whatever the file's order.
func TestTheLatestIssueIsKept(t *testing.T) {
	file := "09/28/26|09/29/26||PDT|1|F|Y|Somewhere|CA|34.0|-118.0|OZONE|70|Moderate|No||x\n" +
		"09/27/26|09/29/26||PDT|2|F|Y|Somewhere|CA|34.0|-118.0|OZONE|30|Good|No||x\n"
	areas := parse(file, captured)
	if len(areas) != 1 || areas[0].Forecast[1].AQI != 70 {
		t.Errorf("tomorrow is %+v; want the later issue's 70", areas)
	}
}

// TestTheFileIsParsedOncePerHour is W14's P-7: the national file (~1.9 MB)
// is served from the cache on every map ask, and was parsed again each time.
// Its parse reads the moment only to the UTC hour (each area's today, a
// whole-hour offset), so within an hour the same body is parsed once; a new
// hour parses again, and so does a new body.
func TestTheFileIsParsedOncePerHour(t *testing.T) {
	var asked []string
	p := New(fileGet{t, &asked}, "")
	at := captured.UTC().Truncate(time.Hour).Add(10 * time.Minute)
	first, _ := p.Areas(context.Background(), at)
	again, _ := p.Areas(context.Background(), at.Add(20*time.Minute))
	if p.Parses() != 1 || len(first) == 0 || len(again) != len(first) {
		t.Fatalf("two asks in one hour: %d parses, %d then %d areas; want one parse, the same areas", p.Parses(), len(first), len(again))
	}
	if _, _ = p.Areas(context.Background(), at.Add(time.Hour)); p.Parses() != 2 {
		t.Errorf("a new hour must parse again (its today may have turned): %d parses", p.Parses())
	}
}
