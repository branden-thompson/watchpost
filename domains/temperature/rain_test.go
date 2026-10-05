package temperature

// rain_test.go — 0.18.0 W12.2 and W12.3 over the recorded answers (D-115,
// D-116, D-118): Open-Meteo's rain and snow, its hours ahead and its days.

import (
	"context"
	"math"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// rainGet answers with the rain fixtures: the days' answer or the hours'.
type rainGet struct {
	t    *testing.T
	asks []string
}

func (r *rainGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	r.asks = append(r.asks, rawURL)
	if strings.Contains(rawURL, "forecast_days") {
		return fixture(r.t, "openmeteo-rain-days.json"), nil
	}
	return fixture(r.t, "openmeteo-rain-hours.json"), nil
}

// rainCaptured is when the rain fixtures were recorded: 13:35 in Juneau.
var rainCaptured = time.Date(2026, 9, 27, 21, 35, 0, 0, time.UTC)

// rainLattice is the rain fixtures' 3x2 lattice over south-east Alaska and
// the edge of British Columbia: two zones, an hour apart.
var rainLattice = Lattice{Name: "rain", Box: geo.Box{W: -135, S: 57, E: -131, N: 59}, Cols: 3, Rows: 2}

// The fixtures' points by name: Juneau's side in Alaska's zone, the
// mountains east of it in British Columbia's.
const (
	glacierBay = 0 // 59N 135W, Alaska
	stikine    = 5 // 57N 131W, British Columbia: snow
)

// TestOpenMeteoReadsTheHoursAhead is W12.2 (D-115): each hour's
// precipitation from the current one, in mm, read in UTC though the points
// answer in two zones.
func TestOpenMeteoReadsTheHoursAhead(t *testing.T) {
	g := &rainGet{t: t}
	r, err := NewOpenMeteo(g, "").Rain(context.Background(), rainLattice, rainCaptured, 3)
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(g.asks[0][strings.Index(g.asks[0], "?")+1:])
	if q.Get("hourly") != "precipitation" || q.Get("forecast_hours") != "4" || q.Get("daily") != "" {
		t.Errorf("the hours ahead asked %v; want precipitation for the current hour and three more, no days", q)
	}
	at := func(h time.Time, p int) float64 {
		for i, x := range r.Hours {
			if x.Equal(h) {
				return r.Hourly[i][p]
			}
		}
		return math.NaN()
	}
	if v := at(time.Date(2026, 9, 27, 21, 0, 0, 0, time.UTC), glacierBay); !near(v, 3.6) {
		t.Errorf("Glacier Bay at 21Z has %v mm; want 3.6 (13:00 in Alaska)", v)
	}
	if v := at(time.Date(2026, 9, 27, 21, 0, 0, 0, time.UTC), stikine); !near(v, 2.5) {
		t.Errorf("the Stikine at 21Z has %v mm; want 2.5 (14:00 in British Columbia)", v)
	}
}

// TestOpenMeteoReadsEachDaysRainAndSnow is W12.3 (D-116): each day's
// heaviest hour, its rain - showers counted with it - in mm, and its snow in
// cm, each on the point's own local date.
func TestOpenMeteoReadsEachDaysRainAndSnow(t *testing.T) {
	g := &rainGet{t: t}
	r, err := NewOpenMeteo(g, "").Rain(context.Background(), rainLattice, rainCaptured, 0)
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(g.asks[0][strings.Index(g.asks[0], "?")+1:])
	if q.Get("forecast_days") != "7" || !strings.Contains(q.Get("daily"), "snowfall_sum") || !strings.Contains(q.Get("daily"), "showers_sum") {
		t.Errorf("the days asked %v; want seven, their rain, showers and snowfall", q)
	}
	for _, c := range []struct {
		name               string
		day, point         int
		peak, rain, snowCm float64
	}{
		{"Glacier Bay today", 0, glacierBay, 4.7, 51.9, 0.07},
		{"Glacier Bay on day 7", 6, glacierBay, 1.0, 14.8, 0},
		{"the Stikine today", 0, stikine, 6.4, 6.7, 38.92},
		{"the Stikine tomorrow", 1, stikine, 4.8, 11.7, 10.57},
	} {
		if !near(r.Peak[c.day][c.point], c.peak) || !near(r.RainSum[c.day][c.point], c.rain) || !near(r.SnowSum[c.day][c.point], c.snowCm) {
			t.Errorf("%s: heaviest %v mm/h, rain %v mm, snow %v cm; want %v, %v, %v", c.name,
				r.Peak[c.day][c.point], r.RainSum[c.day][c.point], r.SnowSum[c.day][c.point], c.peak, c.rain, c.snowCm)
		}
	}
}

// TestRainNotAnsweredIsMissing: a point Open-Meteo gave no value is missing,
// never dry.
func TestRainNotAnsweredIsMissing(t *testing.T) {
	r := newRain(rainLattice)
	if !math.IsNaN(r.Peak[0][0]) || !math.IsNaN(r.RainSum[3][2]) || !math.IsNaN(r.SnowSum[6][5]) {
		t.Error("a new answer holds numbers before anything is read")
	}
}
