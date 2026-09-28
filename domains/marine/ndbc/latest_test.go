package ndbc

// latest_test.go — 0.18.0 D-127: the map's buoys, every station's latest
// reading from NDBC's one national file.

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestTheMapReadsEveryStationsLatest is D-127: one request, each station's
// position, time and readings - wave height, water temperature, wind - and
// "MM", NDBC's missing, as nothing.
func TestTheMapReadsEveryStationsLatest(t *testing.T) {
	srv, requests := server(t)
	client, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	obs, err := New(client, srv.URL).LatestObs(context.Background())
	if err != nil || len(obs) != 31 || requests.Load() != 1 {
		t.Fatalf("%d stations in %d requests (%v); want the fixture's 31 in one", len(obs), requests.Load(), err)
	}
	by := map[string]Obs{}
	for _, o := range obs {
		by[o.ID] = o
	}
	b := by["46025"]
	if b.Lat != 33.765 || b.Lon != -119.077 || !b.At.Equal(time.Date(2026, 9, 28, 13, 50, 0, 0, time.UTC)) ||
		b.WaveM == nil || *b.WaveM != 1.2 || b.WaterC == nil || *b.WaterC != 22.6 || b.WindMS == nil || *b.WindMS != 4.0 {
		t.Errorf("46025 is %+v; want 1.2 m of waves, 22.6 C of water, 4 m/s of wind at 13:50Z", b)
	}
	if w := by["46086"]; w.WaveM != nil || w.WaterC != nil || w.WindMS == nil || *w.WindMS != 6.0 {
		t.Errorf("46086 is %+v; want wind alone", w)
	}
	if w := by["46219"]; w.WindMS != nil || w.WaveM == nil {
		t.Errorf("46219 is %+v; want waves and no wind", w)
	}
}
