package temperature

// air.go — the US AQI (0.18.0 D-138): Open-Meteo's air-quality API, a
// model's (CAMS) on the same lattices, hourly for five days; each day's worst
// hour worked out on the point's own date, since the API gives no daily value.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// airDays is how far the air-quality API forecasts.
const airDays = 5

// OpenMeteoAirCredit is its credit line: CC BY 4.0, as the rest of
// Open-Meteo's (D-131).
const OpenMeteoAirCredit = "Air quality: Open-Meteo.com (CC BY 4.0), CAMS's US AQI model, interpolated"

// AirQuality asks once for a lattice's US AQI: three hours back and five days
// on, each point's hours its own local ones.
func (s *OpenMeteo) AirQuality(ctx context.Context, l Lattice, now time.Time) (Air, error) {
	var lats, lons []string
	for _, p := range l.Points() {
		lats, lons = append(lats, ftoa(p.Lat)), append(lons, ftoa(p.Lon))
	}
	q := url.Values{"latitude": {strings.Join(lats, ",")}, "longitude": {strings.Join(lons, ",")},
		"hourly": {"us_aqi"}, "past_hours": {"3"}, "forecast_days": {fmt.Sprint(airDays)}, "timezone": {"auto"}}
	body, err := s.get.GetText(ctx, s.air+"/v1/air-quality?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Air{}, fmt.Errorf("Open-Meteo air quality: %w", err)
	}
	type point struct {
		Offset int `json:"utc_offset_seconds"`
		Hourly struct {
			Time []string   `json:"time"`
			AQI  []*float64 `json:"us_aqi"`
		} `json:"hourly"`
	}
	var pts []point
	if err := json.Unmarshal(body, &pts); err != nil {
		var one point
		if err1 := json.Unmarshal(body, &one); err1 != nil {
			return Air{}, fmt.Errorf("Open-Meteo air quality: %w", err)
		}
		pts = []point{one}
	}
	if n := l.Cols * l.Rows; len(pts) != n {
		return Air{}, fmt.Errorf("Open-Meteo air quality: %d points answered for %d asked", len(pts), n)
	}
	out := newWaves(l)
	for at, p := range pts {
		zone := time.FixedZone("", p.Offset)
		for i, ts := range p.Hourly.Time {
			t, err := time.ParseInLocation("2006-01-02T15:04", ts, zone)
			if err != nil || i >= len(p.Hourly.AQI) || p.Hourly.AQI[i] == nil {
				continue
			}
			v := *p.Hourly.AQI[i]
			out.Hourly[out.hourIndex(t)][at] = v
			if k := dayOffset(t, now); k >= 0 && k < Days {
				if cur := out.Max[k][at]; math.IsNaN(cur) || v > cur {
					out.Max[k][at] = v // the day's worst hour, on the point's own date
				}
			}
		}
	}
	return out, nil
}
