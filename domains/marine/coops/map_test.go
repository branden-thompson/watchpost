package coops

// map_test.go — 0.18.0 D-128: the map's tide stations, and a station's next
// high or low.

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestTheMapReadsTheTideStationsAndTheirNextTide is D-128: every tide
// station, from the list the places' tides read; a station's next high or
// low after a moment, from its predictions.
func TestTheMapReadsTheTideStationsAndTheirNextTide(t *testing.T) {
	srv, _ := server(t)
	p := newProvider(t, srv.URL)
	stations, err := p.TideStations(context.Background())
	if err != nil || len(stations) != 12 || stations[0].ID != "9410135" || stations[0].Name == "" {
		t.Fatalf("%d stations (%v), the first %+v; want the list's 12", len(stations), err, stations[0])
	}
	next, err := p.NextTide(context.Background(), stations[0].ID, time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	if err != nil || next.Type != "H" || !next.Time.Equal(time.Date(2026, 8, 24, 16, 1, 0, 0, time.UTC)) || next.Height != 1.193 {
		t.Errorf("the next tide after 10:00Z is %+v (%v); want the high of 1.193 m at 16:01Z", next, err)
	}
}

// CLEAR MAP DATA FORGETS THE TIDE PREDICTIONS (D-269): the station's cached
// answers to tide-prediction questions go, in memory and on disk, counted, and
// are asked again - by this session and a later one; its other answers - the
// station lists, the water levels, the currents - stay.
func TestTheTidePredictionsAreForgotten(t *testing.T) {
	srv, requests := server(t)
	dir := t.TempDir()
	onDisk := func() *Provider {
		c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000, MaxRetries: 0, CacheDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = c.ForgetPrefix("settle:") }) // its disk writes settled before the directory goes
		p := New(c, srv.URL)
		p.Now = func() time.Time { return time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC) }
		return p
	}
	p := onDisk()
	ctx := context.Background()
	stations, err := p.TideStations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	ask := func(p *Provider) {
		t.Helper()
		if _, err := p.fetchLevel(ctx, "9410170"); err != nil {
			t.Fatal(err)
		}
		if _, err := p.fetchCurrents(ctx, "SDB0301"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.NextTide(ctx, stations[0].ID, after); err != nil {
		t.Fatal(err)
	}
	ask(p)
	asked := requests.Load()
	n, err := p.ForgetTides()
	if err != nil || n != 1 {
		t.Errorf("forgot %d answers (%v); want the one tide prediction", n, err)
	}
	ask(p)
	ask(onDisk()) // a later session reads the same disk
	if got := requests.Load(); got != asked {
		t.Errorf("the water level or currents were asked again (%d asks); they were not to be forgotten", got-asked)
	}
	if _, err := onDisk().NextTide(ctx, stations[0].ID, after); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != asked+1 {
		t.Errorf("a later session asked the tide prediction %d times; want once, the disk's copy forgotten", got-asked)
	}
}
