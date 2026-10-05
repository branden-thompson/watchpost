package app

// mapinview_test.go — 0.18.0 D-66 (UAT-1 U1-15): "Alerts in view". The
// areas the view touches, asked once, remembered for two minutes; the alerts
// kept to the view; nothing fetched for the estimate.

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
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
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	lp := &livePipelines{idx: idx, areaAlerts: func(_ context.Context, areas []string) ([]snapshot.Alert, error) {
		asked++
		return viewTestAlerts(), nil
	}}
	lp.areaMemo.now = func() time.Time { return now }
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	ask := tty.MapAsk{Place: &here, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}}
	if got := lp.mapInputs(ask); len(got.inView) != 0 || asked != 0 {
		t.Fatalf("the estimate, with nothing remembered, fetched (%d) or drew %d", asked, len(got.inView))
	}
	in := lp.mapInputsFetching(context.Background(), ask)
	var ids []string
	for _, a := range in.inView {
		ids = append(ids, a.ID)
	}
	if asked != 1 || !slices.Equal(ids, []string{"zones", "near"}) {
		t.Errorf("asked %d times; in view %v, want the zone-only alert and the polygon in view", asked, ids)
	}
	_ = lp.mapInputsFetching(context.Background(), ask)
	if asked != 1 {
		t.Errorf("the same areas were asked again within two minutes (%d)", asked)
	}
	now = now.Add(3 * time.Minute)
	if got := lp.mapInputs(ask); len(got.inView) != 2 || asked != 1 {
		t.Errorf("the estimate past two minutes fetched (%d) or lost the remembered answer (%d): it reads what is held, whatever its age", asked, len(got.inView))
	}
	_ = lp.mapInputsFetching(context.Background(), ask)
	if asked != 2 {
		t.Errorf("after two minutes the areas were not asked again (%d)", asked)
	}
	if got := lp.mapInputs(ask); len(got.inView) != 2 || asked != 2 {
		t.Errorf("the estimate's inputs fetched (%d) or lost the remembered answer (%d)", asked, len(got.inView))
	}
}

// TestTheFeedDrawsTheViewsAlertsAndNamesThemOnce is D-66 and D-76 at the
// feed: an alert only the view holds is drawn and handed to the window to
// name in full; one the station also holds is drawn once and left to the
// station's record.
func TestTheFeedDrawsTheViewsAlertsAndNamesThemOnce(t *testing.T) {
	square := func(lon, lat float64) geo.Shape {
		return geo.Shape{{{{Lon: lon, Lat: lat}, {Lon: lon + 0.2, Lat: lat}, {Lon: lon + 0.2, Lat: lat + 0.2}, {Lon: lon, Lat: lat}}}}
	}
	held := snapshot.Alert{ID: "held", Event: "Wind Warning", Severity: "Severe", Area: square(-117.5, 33.0)}
	seen := snapshot.Alert{ID: "view", Event: "Gale Warning", Severity: "Moderate", Area: square(-119.0, 33.5)}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38, Alerts: []snapshot.Alert{held}}
	in := mapInputs{snap: &snapshot.Snapshot{Locations: []snapshot.Location{here}}, place: &here, inView: []snapshot.Alert{held, seen}}
	out := (&livePipelines{}).mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	var ids []string
	for _, o := range out.Overlays {
		ids = append(ids, o.ID)
	}
	if len(ids) != 2 || !slices.Contains(ids, tty.AlertLayer+"/warnings/held") || !slices.Contains(ids, tty.AlertLayer+"/warnings/view") {
		t.Errorf("the feed drew %v; want the held alert once and the view's", ids)
	}
	if len(out.InView) != 1 || out.InView[0].ID != "view" {
		t.Errorf("the window is handed %v to name; want only the view's own", out.InView)
	}
}

// TestTheMapAndTheLookupAskOnTheListenersLane is D-156's wiring: the feed's
// requests and a lookup's go out on the interactive lane, never queued behind
// the station's launch burst - checked on the context that reaches the fetch.
func TestTheMapAndTheLookupAskOnTheListenersLane(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	var interactive, asked bool
	lp := &livePipelines{idx: idx, areaAlerts: func(ctx context.Context, _ []string) ([]snapshot.Alert, error) {
		asked, interactive = true, httpx.Interactive(ctx)
		return nil, nil
	}}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	lp.mapFeed(context.Background(), tty.MapAsk{Snap: &snapshot.Snapshot{}, Place: &here, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}})
	if !asked || !interactive {
		t.Errorf("the feed asked for the view's alerts (%v) on the interactive lane (%v)", asked, interactive)
	}
	ctx, done := lookupContext()
	defer done()
	if !httpx.Interactive(ctx) {
		t.Error("a lookup's resolve is not on the interactive lane")
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Error("a lookup keeps its time limit")
	}
}

// A PAN BACK ASKS NOTHING AGAIN (W14 S-12): the alerts of the last few sets of
// areas are remembered, so returning to a view seen within two minutes reads
// its answer rather than asking the service again.
func TestAPanBackAsksNothingAgain(t *testing.T) {
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
	coast := tty.MapAsk{Place: &here, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}}
	inland := tty.MapAsk{Place: &here, View: tty.MapView{W: -112.5, S: 33.0, E: -111.5, N: 34.0}}
	for _, ask := range []tty.MapAsk{coast, inland, coast, inland} {
		_ = lp.mapInputsFetching(context.Background(), ask)
	}
	if asked != 2 {
		t.Errorf("two views, each seen twice, asked %d times; want each once", asked)
	}
}
