package ndbc

// latest.go — the map's buoys (0.18.0 D-127): every station's latest
// reading, from NDBC's one national file - 103 KB every ten minutes, where
// the places' marine reads a station's own product.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Obs is one station's latest reading: where it is, when it read, and what
// it read - wave height in metres, water temperature in Celsius, wind in
// metres a second - each nil where it reported none.
type Obs struct {
	ID, Name string
	Lat, Lon float64
	At       time.Time
	WaveM    *float64
	WaterC   *float64
	WindMS   *float64
}

// LatestObs is every station's latest reading (D-127).
func (p *Provider) LatestObs(ctx context.Context) ([]Obs, error) {
	raw, err := p.client.GetText(ctx, p.base+"/data/latest_obs/latest_obs.txt", httpx.TTL(obsTTL))
	if err != nil {
		return nil, fmt.Errorf("ndbc latest: %w", err)
	}
	var out []Obs
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 19 || strings.HasPrefix(f[0], "#") {
			continue
		}
		lat, err1 := strconv.ParseFloat(f[1], 64)
		lon, err2 := strconv.ParseFloat(f[2], 64)
		at, err3 := time.Parse("2006 01 02 15 04", strings.Join(f[3:8], " "))
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		out = append(out, Obs{ID: f[0], Lat: lat, Lon: lon, At: at, WindMS: reading(f[9]), WaveM: reading(f[11]), WaterC: reading(f[18])})
	}
	return out, nil
}

// reading is one column's value, nil for NDBC's "MM".
func reading(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
