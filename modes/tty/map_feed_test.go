package tty

// map_feed_test.go — 0.18.0 W5 in the window: the alerts the app hands over
// are drawn, their notes printed under the map, and the feed is asked again
// whenever the data or the place changes.

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// squareFeed is one alert over Oceanside and one note, and counts how often
// it was asked.
func squareFeed(asked *int, label string) func(context.Context, MapAsk) MapFeed {
	return func(_ context.Context, ask MapAsk) MapFeed {
		place := ask.Place
		*asked++
		ring := []tuimaps.LonLat{{Lon: -117.6, Lat: 33.0}, {Lon: -117.1, Lat: 33.0}, {Lon: -117.1, Lat: 33.4}, {Lon: -117.6, Lat: 33.4}, {Lon: -117.6, Lat: 33.0}}
		return MapFeed{
			Overlays: []tuimaps.Overlay{{ID: "alert/" + label, Valid: time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC), Keeps: 6 * time.Hour,
				Features: []tuimaps.Feature{{Kind: tuimaps.Polygon, Rings: [][]tuimaps.LonLat{ring}, Role: tuimaps.AlertSevere, Label: label, Severity: tuimaps.SeveritySevere, ID: label}}}},
			Notes: []string{"A note about " + place.Label + "."},
		}
	}
}

// feedAndSettle runs the pending feed command, then the map's work.
func feedAndSettle(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	if cmd := d.mapFeedCmd(); cmd != nil {
		m, _ := d.Update(cmd())
		d = m.(Dashboard)
	}
	return settleMap(t, d)
}

// TestTheWindowDrawsTheFeed is W5.2 and W5.4 in the window: the alert is on
// the map with its severity word, the note under it.
func TestTheWindowDrawsTheFeed(t *testing.T) {
	asked := 0
	d := mapDash(t, Config{MapFeed: squareFeed(&asked, "Wind Warning")})
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	out := stripANSITest(d.View().Content)
	if !strings.Contains(out, "Wind Warning") {
		t.Errorf("the alert is not on the map:\n%s", out)
	}
	if !strings.Contains(out, "A note about Oceanside, CA.") {
		t.Errorf("the note is not under the map:\n%s", out)
	}
	if asked != 1 {
		t.Errorf("the feed was asked %d times on opening, want once", asked)
	}
}

// TestTheFeedFollowsTheData is W5 with W2.3's data row: a new snapshot while
// the window is open asks for the feed again, and an alert that has gone is
// taken off the map.
func TestTheFeedFollowsTheData(t *testing.T) {
	asked := 0
	label := "Wind Warning"
	d := mapDash(t, Config{})
	d.cfg.MapFeed = func(ctx context.Context, ask MapAsk) MapFeed {
		return squareFeed(&asked, label)(ctx, ask)
	}
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	label = "Heat Advisory"
	m, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
	d = feedAndSettle(t, m.(Dashboard))
	if asked != 2 {
		t.Fatalf("a new snapshot asked the feed %d times in all, want 2", asked)
	}
	out := stripANSITest(d.View().Content)
	if strings.Contains(out, "Wind Warning") || !strings.Contains(out, "Heat Advisory") {
		t.Errorf("the map did not follow the data:\n%s", out)
	}
}

// TestAStaleFeedIsDropped: a feed asked before a newer one is not applied
// over it.
func TestAStaleFeedIsDropped(t *testing.T) {
	asked := 0
	label := "Old Warning"
	d := mapDash(t, Config{})
	d.cfg.MapFeed = func(ctx context.Context, ask MapAsk) MapFeed {
		return squareFeed(&asked, label)(ctx, ask)
	}
	d, _ = pressKey(d, "g")
	old := d.mapFeedCmd()
	label = "New Warning"
	m, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
	d = m.(Dashboard)
	fresh := d.mapFeedCmd()
	freshMsg := fresh()
	label = "Old Warning"
	oldMsg := old()
	m, _ = d.Update(freshMsg)
	m, _ = m.Update(oldMsg)
	d = settleMap(t, m.(Dashboard))
	out := stripANSITest(d.View().Content)
	if strings.Contains(out, "Old Warning") {
		t.Errorf("a stale feed was drawn over a newer one:\n%s", out)
	}
	_ = tea.KeyPressMsg{}
}

// TestARecentPlacesAlertsReachTheMap is U1-14: a place selected from RECENT /
// SEARCHED holds its alerts in the recent snapshot, not the watchlist's; the
// map is asked with them, so their areas are drawn.
func TestARecentPlacesAlertsReachTheMap(t *testing.T) {
	var asked MapAsk
	d := mapDash(t, Config{MapFeed: func(_ context.Context, ask MapAsk) MapFeed { asked = ask; return MapFeed{} }})
	ny := snapshot.Location{Label: "New York, NY", Lat: 40.71, Lon: -74.01,
		Alerts: []snapshot.Alert{{ID: "ny1", Event: "Heat Advisory", Severity: "Moderate"}}}
	m, _ := d.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{ny}}})
	d = m.(Dashboard)
	d.selected = d.numPriority() // the first RECENT row
	if loc := d.selectedLocation(); loc == nil || loc.Label != "New York, NY" {
		t.Fatalf("the selection is %v, not the recent place", loc)
	}
	d, _ = pressKey(d, "g")
	if cmd := d.mapFeedCmd(); cmd != nil {
		cmd()
	}
	found := false
	if asked.Snap != nil {
		for _, l := range asked.Snap.Locations {
			for _, a := range l.Alerts {
				found = found || a.ID == "ny1"
			}
		}
	}
	if !found {
		t.Error("the map was asked without the recent place's alerts, so none is drawn")
	}
	if n := len(placedSnap().Locations); asked.Snap == nil || len(asked.Snap.Locations) != n+1 {
		got := -1
		if asked.Snap != nil {
			got = len(asked.Snap.Locations)
		}
		t.Errorf("the ask holds %d locations; want the watchlist's %d and the selected recent place", got, n)
	}
}
