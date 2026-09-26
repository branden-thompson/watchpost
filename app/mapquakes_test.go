package app

// mapquakes_test.go — 0.18.0 D-80 (UAT-1 U1-43): the map draws the alerts by
// [w]'s categories, never a forecast, and the ticker's earthquakes in view.

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestTheAlertsAreFiledAsTheSevereWindowFilesThem(t *testing.T) {
	square := geo.Shape{{{{Lon: -117.5, Lat: 33.0}, {Lon: -117.3, Lat: 33.0}, {Lon: -117.3, Lat: 33.2}, {Lon: -117.5, Lat: 33.0}}}}
	var alerts []snapshot.Alert
	for i, ev := range []string{"Tornado Warning", "Flood Watch", "Small Craft Advisory", "Special Weather Statement", "Marine Weather Statement", "Hydrologic Outlook", "Air Quality Alert"} {
		alerts = append(alerts, snapshot.Alert{ID: string(rune('a' + i)), Event: ev, Severity: "Moderate", Area: square})
	}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	in := mapInputs{snap: &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "x", Alerts: alerts}}}, place: &here}
	out := (&livePipelines{}).mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	var ids []string
	for _, o := range out.Overlays {
		ids = append(ids, o.ID)
	}
	want := []string{"alert/warnings/a", "alert/watches/b", "alert/advisories/c", "alert/statements/d", "alert/marine/e", "alert/advisories/g"}
	for _, w := range want {
		if !slices.Contains(ids, w) {
			t.Errorf("no %s among %v", w, ids)
		}
	}
	if slices.ContainsFunc(ids, func(id string) bool { return id[len(id)-2:] == "/f" }) {
		t.Errorf("the forecast was drawn: %v", ids)
	}
}

func TestTheQuakesInViewAreDrawnAndNoOthers(t *testing.T) {
	mag := func(m float64) *globalfeed.QuakeDetail { return &globalfeed.QuakeDetail{Mag: &m} }
	feed := []globalfeed.Event{
		{ID: "q1", Class: globalfeed.ClassQuake, Lat: 33.5, Lon: -117.0, HasPoint: true, At: time.Now(), Quake: mag(6.2)},
		{ID: "q2", Class: globalfeed.ClassQuake, Lat: 61.0, Lon: -150.0, HasPoint: true, Quake: mag(7.0)}, // Alaska: out of view
		{ID: "q3", Class: globalfeed.ClassQuake, HasPoint: false, Quake: mag(5.0)},                        // no point
		{ID: "t1", Class: globalfeed.ClassTropical, Lat: 33.4, Lon: -117.1, HasPoint: true},               // not a quake
	}
	v := tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}
	in := quakesIn(feed, v)
	if len(in) != 1 || in[0].ID != "q1" {
		t.Fatalf("the quakes in view are %v, want q1 alone", in)
	}
	o := quakeOverlays(in)
	if len(o) != 1 || o[0].ID != quakeLayerKey+"/q1" || o[0].Features[0].Label != "M 6.2" || o[0].Features[0].Centre.Lat != 33.5 {
		t.Fatalf("the quake is drawn as %+v", o)
	}
	if r := o[0].Features[0].RadiusKm; r != 80 {
		t.Errorf("an M 6.2 is %v km, want 80", r)
	}
	if o[0].Features[0].Severity != 0 {
		t.Error("a quake carries a severity, so the library would report it as an alert over a place")
	}
	if quakeRadiusKm(3) != 10 || quakeRadiusKm(5) != 20 || quakeRadiusKm(9.5) != 400 {
		t.Errorf("the radii are %v %v %v", quakeRadiusKm(3), quakeRadiusKm(5), quakeRadiusKm(9.5))
	}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	out := (&livePipelines{}).mapFeedWith(context.Background(), mapInputs{snap: &snapshot.Snapshot{}, place: &here, quakes: in}, func(snapshot.Location) []string { return nil })
	if len(out.Overlays) != 1 || out.Overlays[0].ID != quakeLayerKey+"/q1" {
		t.Errorf("the feed drew %v", out.Overlays)
	}
}

// TestTheQuakesComeFromTheTickersFeed is D-80's wiring: the inputs read the
// earthquakes from the severe deck's copy of the ticker's feed, kept to the
// view - the map asks nothing of its own.
func TestTheQuakesComeFromTheTickersFeed(t *testing.T) {
	m := 5.4
	lp := &livePipelines{severe: newSevereDeck(nil)}
	lp.severe.feed = []globalfeed.Event{{ID: "q1", Class: globalfeed.ClassQuake, Lat: 33.5, Lon: -117.0, HasPoint: true, Quake: &globalfeed.QuakeDetail{Mag: &m}},
		{ID: "q2", Class: globalfeed.ClassQuake, Lat: 61.0, Lon: -150.0, HasPoint: true, Quake: &globalfeed.QuakeDetail{Mag: &m}}}
	in := lp.mapInputs(tty.MapAsk{View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}})
	if len(in.quakes) != 1 || in.quakes[0].ID != "q1" {
		t.Errorf("the inputs hold quakes %v, want q1 alone", in.quakes)
	}
}
