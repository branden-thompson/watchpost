package app

// maplayers_test.go — 0.18.0 W1.13 (FR-9.3, FR-3.8): the map's registry of
// layers and sources, and W1.14's estimate (FR-9.2) read from it.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// withLayer registers a layer for one test and takes it off after.
func withLayer(t *testing.T, l mapLayer) {
	t.Helper()
	was := len(mapLayers)
	registerMapLayer(l)
	t.Cleanup(func() { mapLayers = mapLayers[:was] })
}

// TestAFakeLayerPlugsInWithoutEditingTheOthers is W1.13's architectural test
// (FR-9.3): a layer that registers itself reaches the window's Settings and
// the estimate, and nothing else was edited to let it in.
func TestAFakeLayerPlugsInWithoutEditingTheOthers(t *testing.T) {
	withLayer(t, mapLayer{key: "fake", label: "Fake layer", on: true,
		cost: func(*snapshot.Snapshot) (int64, int) { return 1_500_000, 30 }})
	cfg := (&livePipelines{}).ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	var keys []string
	for _, l := range cfg.MapLayers {
		keys = append(keys, l.Key)
	}
	if strings.Join(keys, ",") != "alert,fake" {
		t.Fatalf("the window is handed layers %v, want alert then fake", keys)
	}
	all := func(string) bool { return true }
	with := cfg.MapCost(&snapshot.Snapshot{}, all)
	if with.Bytes != 1_500_000 || with.Requests != 30 {
		t.Errorf("the fake layer's cost is not in the estimate: %+v", with)
	}
	if off := cfg.MapCost(&snapshot.Snapshot{}, func(k string) bool { return k != "fake" }); off.Requests != 0 {
		t.Errorf("a layer switched off is still counted: %+v", off)
	}
}

// TestALayerKeyIsRegisteredOnce: two members may not share a key, which is
// what the window switches overlays by.
func TestALayerKeyIsRegisteredOnce(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a second layer called alert was registered")
		}
	}()
	withLayer(t, mapLayer{key: alertLayerKey, label: "Again"})
}

// TestEveryRegisteredSourceIsOnTheClosedList is FR-3.8 against the registry:
// the sources that registered are exactly the closed list's hosts.
func TestEveryRegisteredSourceIsOnTheClosedList(t *testing.T) {
	closed := map[string]bool{"tiles.openfreemap.org": true} // FR-3.8; IEM and MRMS join with W8
	if len(mapSources) == 0 {
		t.Fatal("no source registered")
	}
	for _, s := range mapSources {
		if !closed[hostOf(s.address)] {
			t.Errorf("%s (%s) registered and is not on the closed list", s.name, s.address)
		}
	}
}

// TestEveryOverlayBelongsToARegisteredLayer: the window switches an overlay
// by the key before its id's slash, so every overlay the feed makes must
// carry a registered layer's key.
func TestEveryOverlayBelongsToARegisteredLayer(t *testing.T) {
	_, feed := feedFor(t, "03-two-alerts-harlan", nil)
	if len(feed.Overlays) == 0 {
		t.Fatal("the scenario drew nothing")
	}
	keys := map[string]bool{}
	for _, l := range mapLayers {
		keys[l.key] = true
	}
	for _, o := range feed.Overlays {
		key, _, ok := strings.Cut(o.ID, "/")
		if !ok || !keys[key] {
			t.Errorf("overlay %q belongs to no registered layer", o.ID)
		}
	}
}

// TestTheAlertLayersCostIsItsZones is W1.14's estimate for the alert areas:
// every zone the alerts name, once, as if none were held - an alert that
// brought its own polygon costs nothing - at the measured size of a zone.
func TestTheAlertLayersCostIsItsZones(t *testing.T) {
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{
		{Alerts: []snapshot.Alert{{ID: "a", AffectedZones: []string{"CAZ043", "CAZ048"}}, {ID: "b", AffectedZones: []string{"CAZ048", "CAZ050"}}}},
		{Alerts: []snapshot.Alert{{ID: "a", AffectedZones: []string{"CAZ043", "CAZ048"}},
			{ID: "c", AffectedZones: []string{"CAZ099"}, Area: geo.Shape{{{{Lon: -117, Lat: 33}, {Lon: -116, Lat: 33}, {Lon: -116, Lat: 34}, {Lon: -117, Lat: 33}}}}}}},
	}}
	bytes, requests := alertLayerCost(snap)
	if requests != 3 || bytes != 3*zoneShapeBytes {
		t.Errorf("the alert areas cost %d bytes in %d requests; want three zones", bytes, requests)
	}
	if b, r := alertLayerCost(nil); b != 0 || r != 0 {
		t.Errorf("no snapshot costs %d, %d", b, r)
	}
}

// TestTheEstimateIsReachedFromProduction: the window's estimate is the
// registry's, handed from the composition root.
func TestTheEstimateIsReachedFromProduction(t *testing.T) {
	cfg := (&livePipelines{}).ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	if cfg.MapCost == nil || len(cfg.MapLayers) == 0 {
		t.Fatal("the window is handed no layers or no estimate")
	}
	if got := cfg.MapCost(nil, func(string) bool { return true }); got != (tty.MapCost{}) {
		t.Errorf("no snapshot is estimated at %+v", got)
	}
}
