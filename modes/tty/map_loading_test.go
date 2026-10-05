package tty

// map_loading_test.go — D-266 and D-268 at the window: until the alerts for
// the place and view land, the description and the status line say they are
// loading, never that none covers the place; a failed ask says so; the
// alerts' lane and the other layers' land apart.

import (
	"context"
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// unlandedDash opens the map under --ascii on Oceanside with one alert in
// the feed, and delivers nothing.
func unlandedDash(t *testing.T, feed func(context.Context, MapAsk) MapFeed) Dashboard {
	t.Helper()
	d := mapDash(t, Config{ASCII: true, MapFeed: feed})
	m, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	return d
}

func TestUntilTheAlertsLandTheyAreLoading(t *testing.T) {
	d := unlandedDash(t, boxFeed(-117.6, -117.1, false))
	said := strings.Join(d.describeLinesAll(), "\n")
	if !strings.Contains(said, "Alerts for the map are loading.") || strings.Contains(said, "No alert") {
		t.Errorf("before the alerts land the description says:\n%s\nwant them loading, never none (D-266)", said)
	}
	if got := d.mapStatusText(); got != mapAlertsLoadingText && got != mapLoadingText {
		t.Errorf("the status line says %q while the alerts are not in", got)
	}
	d = feedAndSettle(t, d)
	said = strings.Join(d.describeLinesAll(), "\n")
	if strings.Contains(said, "loading") || !strings.Contains(said, "Wind Warning") {
		t.Errorf("once landed the description says:\n%s", said)
	}
}

func TestAFailedAskForTheAlertsIsSaid(t *testing.T) {
	failed := func(context.Context, MapAsk) MapFeed { return MapFeed{AlertsFailed: true} }
	d := feedAndSettle(t, unlandedDash(t, failed))
	said := strings.Join(d.describeLinesAll(), "\n")
	if !strings.Contains(said, "Alerts for the map could not be loaded.") || strings.Contains(said, "No alert") {
		t.Errorf("a failed ask is described as:\n%s\nwant it said, never none (D-266)", said)
	}
}

// TestTheAlertsAndTheLayersLandApart is D-268 at the window: the alerts'
// answer is drawn before the layers' lane answers, and the layers' answer
// adds its overlays without taking the alerts away.
func TestTheAlertsAndTheLayersLandApart(t *testing.T) {
	quake := tuimaps.Overlay{ID: "quake/a", Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: -117.3, Lat: 33.3}, RadiusDots: 4, Role: tuimaps.QuakeDay}}}
	both := func(ctx context.Context, ask MapAsk) MapFeed {
		f := boxFeed(-117.6, -117.1, false)(ctx, ask)
		f.Overlays = append(f.Overlays, quake)
		return f
	}
	d := unlandedDash(t, both)
	m, _ := d.Update(d.mapFeedCmd(laneAlerts)())
	d = m.(Dashboard)
	if _, ok := d.mapPane.given["alert/w1"]; !ok || !d.alertsLanded() {
		t.Fatal("the alerts' answer was not drawn on its own")
	}
	for _, o := range d.mapPane.feed.Overlays {
		if o.ID == "quake/a" {
			t.Error("the alerts' lane carried another layer's overlay")
		}
	}
	m, _ = d.Update(d.mapFeedCmd(laneLayers)())
	d = m.(Dashboard)
	if _, ok := d.mapPane.given["alert/w1"]; !ok {
		t.Error("the layers' answer took the alerts away")
	}
	ids := map[string]bool{}
	for _, o := range d.mapPane.feed.Overlays {
		ids[o.ID] = true
	}
	if !ids["quake/a"] || !ids["alert/w1"] {
		t.Errorf("the merged feed holds %v; want the alerts' and the layers' answers together", ids)
	}
}
