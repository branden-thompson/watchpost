package coops

// map_test.go — 0.18.0 D-128: the map's tide stations, and a station's next
// high or low.

import (
	"context"
	"testing"
	"time"
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
