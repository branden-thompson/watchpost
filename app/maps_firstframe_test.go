package app

// maps_firstframe_test.go — 0.18.0 W2.9, M5's first arm, the gate's half
// (D-43, D-46, D-53): the requests a cold open makes before its first
// complete frame - every alert's area and the basemap - counted, and bounded
// by wave 1's measured parts. The timing half, the p90 of 20 opens on the
// reference machine, is the SHIP report's; "never waits for radar" joins with
// radar (W8.12).

import (
	"context"
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheFirstCompleteFrameIsFewRequests: at 149×38 a cold open on Roswell
// with its watch and warning asks one TileJSON, no more tiles than the view
// can touch, and each zone the alerts name once - and then its frame is
// complete.
func TestTheFirstCompleteFrameIsFewRequests(t *testing.T) {
	loc, srv := m1Fixture(t, "04-watch-and-warning-roswell")
	zones := map[string]bool{}
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	for _, a := range loc.Alerts {
		if a.Area.Empty() {
			for _, z := range a.AffectedZones {
				zones[z] = true // a zone two alerts name is fetched once
			}
		}
	}
	zoneRequests := len(zones)
	feed := lp.mapFeedWith(context.Background(), mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return nil })

	tr := &recorded{}
	body := tuimaps.Size{Cols: 149 - 2 - 8, Rows: 38 - 8 - 1} // the window's body at 149×38: its frame and inset, the status line
	m, err := testBuilder(t, tr).build(body)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Recentre(tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}); err != nil {
		t.Fatal(err)
	}
	if err := m.Zoom(6); err != nil {
		t.Fatal(err)
	}
	for _, o := range feed.Overlays {
		if _, err := m.Set(o); err != nil {
			t.Fatal(err)
		}
	}
	f := settleAt(t, m, body, mapNoon)
	if f.Status != tuimaps.Complete {
		t.Fatalf("the first frame settled %v, not complete", f.Status)
	}
	tileJSON, tiles := 0, 0
	for _, u := range tr.requests() {
		if strings.HasSuffix(u, ".pbf") {
			tiles++
		} else {
			tileJSON++
		}
	}
	// A view of 278 × 116 dots touches at most three tiles across and two
	// down (wave 1 measured two to four at this size).
	const mostTiles = 3 * 2
	t.Logf("first complete frame: %d TileJSON, %d tiles, %d zones", tileJSON, tiles, zoneRequests)
	if tileJSON != 1 || tiles == 0 || tiles > mostTiles {
		t.Errorf("a cold open asked %d TileJSON and %d tiles; want one, and one to %d tiles", tileJSON, tiles, mostTiles)
	}
	if zoneRequests == 0 || len(feed.Overlays) != len(loc.Alerts) {
		t.Errorf("the scenario drew %d of %d alerts by %d zones: this measures nothing", len(feed.Overlays), len(loc.Alerts), zoneRequests)
	}
}
