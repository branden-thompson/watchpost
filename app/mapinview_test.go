package app

// mapinview_test.go — 0.18.0 D-66 (UAT-1 U1-15): "Alerts in view". The
// areas the view touches, asked once, remembered for two minutes; the alerts
// kept to the view; nothing fetched for the estimate or another scope.

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestAViewNamesTheAreasItTouches(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		v    tty.MapView
		want []string
	}{
		{"Southern California", tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}, []string{"CA", "PZ"}},
		{"the Texas coast", tty.MapView{W: -96.0, S: 28.3, E: -93.5, N: 30.4}, []string{"TX", "GM"}},
		{"Anchorage", tty.MapView{W: -150.6, S: 60.9, E: -149.2, N: 61.5}, []string{"AK"}},
		// open water within forty miles of the coast reads as the coast's state
		{"off Carlsbad", tty.MapView{W: -117.75, S: 33.0, E: -117.6, N: 33.1}, []string{"CA"}},
	} {
		got := viewAreas(idx, c.v)
		for _, w := range c.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: the areas are %v, missing %s", c.name, got, w)
			}
		}
	}
	if got := viewAreas(nil, tty.MapView{W: -118, S: 33, E: -117, N: 34}); len(got) != 1 || got[0] != "PZ" && got[0] != "CA" {
		t.Logf("with no index the areas are %v (marine boxes alone)", got)
	}
}

// viewTestAlerts are an area's answer: one zone-only alert, one polygon in
// the view, one far outside it and one due north of it - the view's
// longitudes, none of its latitudes.
func viewTestAlerts() []snapshot.Alert {
	in := geo.Shape{{{{Lon: -117.5, Lat: 33.0}, {Lon: -117.3, Lat: 33.0}, {Lon: -117.3, Lat: 33.2}, {Lon: -117.5, Lat: 33.0}}}}
	out := geo.Shape{{{{Lon: -121.5, Lat: 38.0}, {Lon: -121.3, Lat: 38.0}, {Lon: -121.3, Lat: 38.2}, {Lon: -121.5, Lat: 38.0}}}}
	north := geo.Shape{{{{Lon: -117.5, Lat: 36.0}, {Lon: -117.3, Lat: 36.0}, {Lon: -117.3, Lat: 36.2}, {Lon: -117.5, Lat: 36.0}}}}
	return []snapshot.Alert{
		{ID: "zones", Event: "Beach Hazards Statement", Severity: "Moderate", AffectedZones: []string{"CAZ043"}},
		{ID: "near", Event: "Flood Warning", Severity: "Severe", Area: in},
		{ID: "far", Event: "Flood Warning", Severity: "Severe", Area: out},
		{ID: "north", Event: "Flood Warning", Severity: "Severe", Area: north},
	}
}

func TestAlertsInViewAreAskedOnceAndKeptToTheView(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	asked := 0
	lp := &livePipelines{idx: idx, areaAlerts: func(_ context.Context, areas []string) ([]snapshot.Alert, error) {
		asked++
		return viewTestAlerts(), nil
	}}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	ask := tty.MapAsk{Place: &here, Scope: tty.ScopeInView, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}}
	if got := lp.mapInputs(ask); len(got.national) != 0 || asked != 0 {
		t.Fatalf("the estimate, with nothing remembered, fetched (%d) or drew %d", asked, len(got.national))
	}
	in := lp.mapInputsFetching(context.Background(), ask)
	var ids []string
	for _, a := range in.national {
		ids = append(ids, a.ID)
	}
	if asked != 1 || !slices.Equal(ids, []string{"zones", "near"}) {
		t.Errorf("asked %d times; in view %v, want the zone-only alert and the polygon in view", asked, ids)
	}
	_ = lp.mapInputsFetching(context.Background(), ask)
	if asked != 1 {
		t.Errorf("the same areas were asked again within two minutes (%d)", asked)
	}
	lp.areaMemo.at = lp.areaMemo.at.Add(-3 * time.Minute)
	_ = lp.mapInputsFetching(context.Background(), ask)
	if asked != 2 {
		t.Errorf("after two minutes the areas were not asked again (%d)", asked)
	}
	if got := lp.mapInputs(ask); len(got.national) != 2 || asked != 2 {
		t.Errorf("the estimate's inputs fetched (%d) or lost the remembered answer (%d)", asked, len(got.national))
	}
	ask.Scope = tty.ScopeStation
	if in := lp.mapInputsFetching(context.Background(), ask); in.national != nil || asked != 2 {
		t.Error("the station's scope asked for the view's areas")
	}
}
