package app

// mapnational_test.go — 0.18.0 W5.3 (FR-4.3, D-23): the alert scope. With
// "national", the map draws the national severe events in the selected
// place's region as well as the station's own alerts - never wider than the
// region (D-28) - and the estimate counts what they fetch.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// square is a small polygon around a point.
func square(lon, lat float64) geo.Shape {
	return geo.Shape{{{{Lon: lon - 0.1, Lat: lat - 0.1}, {Lon: lon + 0.1, Lat: lat - 0.1}, {Lon: lon + 0.1, Lat: lat + 0.1}, {Lon: lon - 0.1, Lat: lat - 0.1}}}}
}

var nationalUntil = time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC)

// nationalFeed is the national feed as the ticker holds it: a warning in
// Texas with its polygon, a watch in New Mexico by its zone, a warning in
// Alaska, a superseded warning in Oklahoma, and a quake.
func nationalFeed() []globalfeed.Event {
	sev := func(id, typ string, lat, lon float64, point bool, area geo.Shape, zones ...string) globalfeed.Event {
		return globalfeed.Event{ID: "https://api.weather.gov/alerts/" + id, Class: globalfeed.ClassSevereWx, Type: typ, Place: typ + " area",
			Lat: lat, Lon: lon, HasPoint: point, Until: nationalUntil, Source: "NWS",
			Severe: &globalfeed.SevereDetail{Severity: "Extreme", Area: area, AffectedZones: zones}}
	}
	sup := sev("urn:oid:ok1", "Tornado Warning", 35.4, -97.5, true, square(-97.5, 35.4))
	sup.Superseded = true
	return []globalfeed.Event{
		sev("urn:oid:tx1", "Tornado Warning", 32.8, -96.8, true, square(-96.8, 32.8)),
		sev("urn:oid:nm1", "Tornado Watch", 0, 0, false, nil, "https://api.weather.gov/zones/forecast/NMZ238"),
		sev("urn:oid:ak1", "Tornado Warning", 61.2, -149.9, true, square(-149.9, 61.2)),
		sup,
		{ID: "us7000quake", Class: globalfeed.ClassQuake, Type: "Earthquake", Lat: 33, Lon: -117, HasPoint: true},
	}
}

// TestTheNationalScopeAddsTheRegionsSevereEvents is W5.3's count of drawn
// alerts per scope: the station's scope draws the station's alerts alone;
// national adds the region's severe events - the Texas warning and the New
// Mexico watch, drawn by its zone - and never Alaska's, a superseded one or
// a quake.
func TestTheNationalScopeAddsTheRegionsSevereEvents(t *testing.T) {
	loc, srv := m1Fixture(t, "04-watch-and-warning-roswell")
	stationAlerts := len(loc.Alerts)
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	none := func(snapshot.Location) []string { return nil }
	station := lp.mapFeedWith(context.Background(), lp.inputsWith(tty.MapAsk{Snap: snap, Place: &loc, Scope: tty.ScopeStation}, nationalFeed()), none)
	if len(station.Overlays) != stationAlerts {
		t.Errorf("the station's scope drew %d overlays for %d alerts", len(station.Overlays), stationAlerts)
	}
	national := lp.mapFeedWith(context.Background(), lp.inputsWith(tty.MapAsk{Snap: snap, Place: &loc, Scope: tty.ScopeNational}, nationalFeed()), none)
	if len(national.Overlays) != stationAlerts+2 {
		t.Fatalf("the national scope drew %d overlays, want the station's %d and two", len(national.Overlays), stationAlerts)
	}
	ids := map[string]bool{}
	for _, o := range national.Overlays {
		ids[o.ID] = true
	}
	if !ids[alertLayerKey+"/urn:oid:tx1"] || !ids[alertLayerKey+"/urn:oid:nm1"] {
		t.Errorf("the national overlays are %v; want the Texas warning and the New Mexico watch", ids)
	}
	if len(snap.Locations) != 1 {
		t.Error("the national scope changed the station's snapshot")
	}
}

// TestANationalEventTheStationHoldsIsDrawnOnce: an alert on both the
// station's places and the national feed is one overlay.
func TestANationalEventTheStationHoldsIsDrawnOnce(t *testing.T) {
	place := snapshot.Location{Label: "Dallas, TX", Lat: 32.8, Lon: -96.8,
		Alerts: []snapshot.Alert{{ID: "urn:oid:tx1", Event: "Tornado Warning", Severity: "Extreme", Area: square(-96.8, 32.8), Expires: nationalUntil}}}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{place}}
	lp := &livePipelines{}
	feed := lp.mapFeedWith(context.Background(), lp.inputsWith(tty.MapAsk{Snap: snap, Place: &place, Scope: tty.ScopeNational}, nationalFeed()[:1]),
		func(snapshot.Location) []string { return nil })
	if len(feed.Overlays) != 1 {
		t.Errorf("an alert held twice drew %d overlays", len(feed.Overlays))
	}
}

// TestANationalEventReadsAsAnAlert: the conversion keeps what the window
// shows and the zone store needs - the bare id, the event, the severity, the
// time it ends, the zone codes and the polygon.
func TestANationalEventReadsAsAnAlert(t *testing.T) {
	here := snapshot.Location{Lat: 33.4, Lon: -104.5} // Roswell, NM
	got := nationalAlerts(nationalFeed(), &here)
	if len(got) != 2 {
		t.Fatalf("%d national alerts in the contiguous region, want 2: %+v", len(got), got)
	}
	tx, nm := got[0], got[1]
	if tx.ID != "urn:oid:tx1" || tx.Event != "Tornado Warning" || tx.Severity != "Extreme" || !tx.Expires.Equal(nationalUntil) || tx.Area.Empty() {
		t.Errorf("the Texas warning reads %+v", tx)
	}
	if nm.ID != "urn:oid:nm1" || len(nm.AffectedZones) != 1 || nm.AffectedZones[0] != "NMZ238" || !nm.Area.Empty() {
		t.Errorf("the New Mexico watch reads %+v", nm)
	}
	if got := nationalAlerts(nationalFeed(), &snapshot.Location{Lat: 61.2, Lon: -149.9}); len(got) != 1 || got[0].ID != "urn:oid:ak1" {
		t.Errorf("from Anchorage the national alerts are %+v, want Alaska's alone", got)
	}
	if got := nationalAlerts(nationalFeed(), nil); got != nil {
		t.Errorf("with no place selected: %+v", got)
	}
	if got := nationalAlerts(nationalFeed(), &snapshot.Location{Lat: 0, Lon: 0}); got != nil {
		t.Errorf("a place in no region: %+v", got)
	}
}

// TestTheEstimateCountsTheNationalZones: the national scope adds the zones
// the region's zone-only events name; a polygon costs nothing.
func TestTheEstimateCountsTheNationalZones(t *testing.T) {
	here := snapshot.Location{Lat: 33.4, Lon: -104.5}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{here}}
	lp := &livePipelines{}
	all := func(string) bool { return true }
	if c := refreshCost(lp.inputsWith(tty.MapAsk{Snap: snap, Place: &here, Scope: tty.ScopeStation}, nationalFeed()), all); c.Requests != 0 {
		t.Errorf("the station's scope with no alerts costs %+v", c)
	}
	if c := refreshCost(lp.inputsWith(tty.MapAsk{Snap: snap, Place: &here, Scope: tty.ScopeNational}, nationalFeed()), all); c.Requests != 1 || c.Bytes != zoneShapeBytes {
		t.Errorf("the national scope costs %+v, want the watch's one zone", c)
	}
}

// TestTheScopeReachesTheWindowAndTheFile is W5.3's wiring: the file's scope
// is handed to the window and Settings' answer written back; production's
// inputs read the ticker's national feed through the severe deck.
func TestTheScopeReachesTheWindowAndTheFile(t *testing.T) {
	cfg := (&livePipelines{}).ttyConfig("t", Options{}, false, config.Config{MapAlertScope: "national"}, nil, nil, nil, nil, nil, nil)
	if cfg.MapAlertScope != "national" {
		t.Errorf("the window is handed scope %q", cfg.MapAlertScope)
	}
	withConfigFile(t)
	if err := setUIHook(tty.UIPrefs{Units: "metric", MapAlertScope: "national"}); err != nil {
		t.Fatal(err)
	}
	if got, err := config.Load(); err != nil || got.MapAlertScope != "national" {
		t.Errorf("the save wrote %q (%v)", got.MapAlertScope, err)
	}
	deck := newSevereDeck(func(tea.Msg) {})
	deck.SetFeed(nationalFeed(), nil)
	lp := &livePipelines{severe: deck}
	here := snapshot.Location{Lat: 33.4, Lon: -104.5}
	if in := lp.mapInputs(tty.MapAsk{Place: &here, Scope: tty.ScopeNational}); len(in.national) != 2 {
		t.Errorf("production's inputs hold %d national alerts, want the ticker's two in the region", len(in.national))
	}
	if in := lp.mapInputs(tty.MapAsk{Place: &here, Scope: tty.ScopeStation}); in.national != nil {
		t.Error("the station's scope read the national feed")
	}
}

// withConfigFile points the config at a file the station already has:
// Settings saves over one, and a first run's is refused by design.
func withConfigFile(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "watchpost"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "watchpost", "config.toml"), []byte("units = \"imperial\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTheFeedHandsTheWindowTheNationalAlerts: the window describes an alert
// the station does not hold from the alerts the feed drew.
func TestTheFeedHandsTheWindowTheNationalAlerts(t *testing.T) {
	loc, srv := m1Fixture(t, "04-watch-and-warning-roswell")
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	feed := lp.mapFeedWith(context.Background(), lp.inputsWith(tty.MapAsk{Snap: snap, Place: &loc, Scope: tty.ScopeNational}, nationalFeed()),
		func(snapshot.Location) []string { return nil })
	if len(feed.National) != 2 || feed.National[0].ID != "urn:oid:tx1" {
		t.Errorf("the feed hands the window %+v", feed.National)
	}
}

// TestTheZoneStoreAsksAsWatchpost is W3.9's other half: the store the map
// resolves zones through is built on the station's data client, whose agent
// names watchpost.
func TestTheZoneStoreAsksAsWatchpost(t *testing.T) {
	if !strings.HasPrefix(UserAgent, "watchpost ") {
		t.Errorf("the data client's agent is %q", UserAgent)
	}
}
