package temperature

// rain.go — the map's rain and snow ahead (0.18.0 W12.2, W12.3; D-115,
// D-116): Open-Meteo's, in every region and on every day (D-118), because
// NDFD gives rain only as six-hour amounts to three days and no hourly rate.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// OpenMeteoRainCredit is the rain's credit line (CC BY 4.0, as the
// temperature's): a model's forecast, interpolated between its points.
const OpenMeteoRainCredit = "Rain and snow: Open-Meteo.com (CC BY 4.0), a model's forecast, interpolated"

// Rain is what Open-Meteo answered of a lattice's rain and snow. Hourly is
// by hour then point: the precipitation, rain and snow as water, in mm in the
// hour before each. Peak, RainSum and SnowSum are by day from today, then
// point, each on the point's own local date: the day's heaviest hour in mm,
// its rain - showers counted with it - in mm, and its snowfall in cm.
// Missing is NaN, never dry.
type Rain struct {
	Lattice                Lattice
	Hours                  []time.Time // on the hour, UTC, oldest first
	Hourly                 [][]float64
	Peak, RainSum, SnowSum [Days][]float64
}

// newRain is an answer with every value missing.
func newRain(l Lattice) Rain {
	n := l.Cols * l.Rows
	r := Rain{Lattice: l}
	for k := range Days {
		r.Peak[k], r.RainSum[k], r.SnowSum[k] = missing(n), missing(n), missing(n)
	}
	return r
}

// Rain asks once. hours above zero is Radar mode's hours ahead (W12.2): the
// current hour and that many after it, and no days. Zero is Forecast mode's
// days (W12.3): every hour of seven days, for each day's heaviest, and each
// day's totals.
func (s *OpenMeteo) Rain(ctx context.Context, l Lattice, now time.Time, hours int) (Rain, error) {
	var lats, lons []string
	for _, p := range l.Points() {
		lats, lons = append(lats, ftoa(p.Lat)), append(lons, ftoa(p.Lon))
	}
	q := url.Values{"latitude": {strings.Join(lats, ",")}, "longitude": {strings.Join(lons, ",")},
		"hourly": {"precipitation"}, "timezone": {"auto"}}
	if hours > 0 {
		q.Set("forecast_hours", strconv.Itoa(hours+1))
	} else {
		q.Set("daily", "rain_sum,showers_sum,snowfall_sum")
		q.Set("forecast_days", strconv.Itoa(Days))
	}
	body, err := s.get.GetText(ctx, s.base+"/v1/forecast?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Rain{}, fmt.Errorf("Open-Meteo rain: %w", err)
	}
	out := newRain(l)
	if err := parseRain(body, &out); err != nil {
		return Rain{}, fmt.Errorf("Open-Meteo rain: %w", err)
	}
	return out, nil
}

// rainPoint is one point's answer.
type rainPoint struct {
	Offset int `json:"utc_offset_seconds"`
	Hourly struct {
		Time   []string   `json:"time"`
		Precip []*float64 `json:"precipitation"`
	} `json:"hourly"`
	Daily struct {
		Time    []string   `json:"time"`
		Rain    []*float64 `json:"rain_sum"`
		Showers []*float64 `json:"showers_sum"`
		Snow    []*float64 `json:"snowfall_sum"`
	} `json:"daily"`
}

// parseRain adds an answer to a rain: a list, one a point in the order
// asked; one point alone is answered as an object.
func parseRain(body []byte, out *Rain) error {
	var pts []rainPoint
	if err := json.Unmarshal(body, &pts); err != nil {
		var one rainPoint
		if err1 := json.Unmarshal(body, &one); err1 != nil {
			return err
		}
		pts = []rainPoint{one}
	}
	if n := out.Lattice.Cols * out.Lattice.Rows; len(pts) != n {
		return fmt.Errorf("%d points answered for %d asked", len(pts), n)
	}
	for at, p := range pts {
		zone := time.FixedZone("", p.Offset)
		days := map[string]int{} // the point's local dates, today first
		for k, d := range p.Daily.Time {
			if k < Days {
				days[d] = k
			}
		}
		for i, ts := range p.Hourly.Time {
			t, err := time.ParseInLocation("2006-01-02T15:04", ts, zone)
			if err != nil || i >= len(p.Hourly.Precip) || p.Hourly.Precip[i] == nil {
				continue
			}
			v := *p.Hourly.Precip[i]
			out.Hourly[out.hourIndex(t)][at] = v
			if k, ok := days[ts[:10]]; ok {
				if cur := out.Peak[k][at]; math.IsNaN(cur) || v > cur {
					out.Peak[k][at] = v
				}
			}
		}
		for k := range min(len(p.Daily.Time), Days) {
			set(out.SnowSum[k], at, p.Daily.Snow, k)
			if k < len(p.Daily.Rain) && p.Daily.Rain[k] != nil {
				rain := *p.Daily.Rain[k]
				if k < len(p.Daily.Showers) && p.Daily.Showers[k] != nil {
					rain += *p.Daily.Showers[k]
				}
				out.RainSum[k][at] = rain
			}
		}
	}
	return nil
}

// hourIndex is the index of an hour, adding it in order.
func (r *Rain) hourIndex(t time.Time) int {
	return hourRow(&r.Hours, &r.Hourly, r.Lattice.Cols*r.Lattice.Rows, t)
}

// hourRow is the index of an hour among hours kept in order, a row of n
// values beside each: the hour and a row of missing values are added in
// their place when it is not there.
func hourRow(hours *[]time.Time, rows *[][]float64, n int, t time.Time) int {
	t = t.UTC().Truncate(time.Hour)
	at := 0
	for at < len(*hours) && (*hours)[at].Before(t) {
		at++
	}
	if at < len(*hours) && (*hours)[at].Equal(t) {
		return at
	}
	*hours = append((*hours)[:at], append([]time.Time{t}, (*hours)[at:]...)...)
	*rows = append((*rows)[:at], append([][]float64{missing(n)}, (*rows)[at:]...)...)
	return at
}
