package tty

// map_station_test.go — 0.19.0 W7 (FR-2.1, FR-2.3, FR-2.4; D-195, D-196):
// the Broadcaster's map, by the UI mode's map role - on the station, its
// service area always drawn, its tower marked, no place keys, its words.

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
	"github.com/branden-thompson/watchpost/platform/units"
)

// bonsall is the station's transmitter in these tests: inland of the
// selected place, Oceanside, so the two are told apart.
var bonsall = snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23}

// stationMap is the map opened on the Broadcaster, the transmitter and a
// 100-mile service radius set unless the config sets its own.
func stationMap(t *testing.T, cfg Config) Dashboard {
	t.Helper()
	if cfg.Transmitter == nil {
		tx := bonsall
		cfg.Transmitter = &tx
	}
	if cfg.ServiceRadiusMi == 0 {
		cfg.ServiceRadiusMi = 100
	}
	d := mapDash(t, cfg)
	d.surface = SurfaceBroadcaster
	d, _ = pressKey(d, "g")
	return feedAndSettle(t, d)
}

// near reports whether the map's centre is within a hundredth of a degree of
// a place.
func near(d Dashboard, lat, lon float64) bool {
	c, _ := d.mapPane.m.Centre()
	return math.Abs(c.Lat-lat) < 0.01 && math.Abs(c.Lon-lon) < 0.01
}

// TestTheConsoleOpensTheMapOnTheStation is FR-2.1 (D-195): on the console, g
// opens the map window - with no window open and over one - on the station:
// the transmitter set in Settings.
func TestTheConsoleOpensTheMapOnTheStation(t *testing.T) {
	tx := bonsall
	base := NewRouter(mapDash(t, Config{Transmitter: &tx, ServiceRadiusMi: 100}))
	base.active, base.observer.surface = SurfaceBroadcaster, SurfaceBroadcaster
	for _, keys := range [][]string{{"g"}, {"a", "g"}, {"a", "S", "g"}} {
		r := base
		for _, k := range keys {
			r = press(t, r, k)
		}
		t.Cleanup(r.observer.closeMap)
		if r.observer.modal != modalMap || r.observer.mapPane.m == nil {
			t.Errorf("%v on the console: the window shown is %v, not the map", keys, r.observer.modal)
			continue
		}
		if !near(r.observer, tx.Lat, tx.Lon) {
			c, _ := r.observer.mapPane.m.Centre()
			t.Errorf("%v: the map is on %v; the station is at %.2f, %.2f", keys, c, tx.Lat, tx.Lon)
		}
	}
}

// TestTheConsoleKeysDoNotClash is FR-2.1: no console key means two things,
// and g is the map's on the console as on Observer (D-56).
func TestTheConsoleKeysDoNotClash(t *testing.T) {
	owner := map[string]term.Action{}
	for act, b := range broadcasterKeyMap() {
		for _, k := range b.Keys {
			if other, ok := owner[k]; ok {
				t.Errorf("%q is bound to both %s and %s on the console", k, other, act)
			}
			owner[k] = act
		}
	}
	if owner["g"] != actMap {
		t.Errorf("g on the console is %q, not the map", owner["g"])
	}
	d := stationMap(t, Config{})
	help := stripANSITest(strings.Join(d.helpLines(d.opts()), "\n"))
	if !regexp.MustCompile(`\bg\s+- Map\b`).MatchString(help) {
		t.Errorf("the console's Help does not list the map's key:\n%s", help)
	}
}

// serviceArea checks the service area and the tower are handed in: a zone of
// the service radius around the station, held by the library, the tower
// marked.
func serviceArea(t *testing.T, mode string, d Dashboard, lat, lon, mi float64) {
	t.Helper()
	o, ok := d.mapPane.zoneGiven[serviceAreaID]
	if !ok || len(o.Features) != 1 {
		t.Fatalf("%s: no service area handed in", mode)
	}
	if f := o.Features[0]; f.Kind != tuimaps.Circle || f.Role != tuimaps.Zone || f.RadiusKm != mi*units.KmPerMile || f.Centre != (tuimaps.LonLat{Lon: lon, Lat: lat}) {
		t.Errorf("%s: the service area is %+v; a zone of %.0f mi around the tower", mode, f, mi)
	}
	if !slices.Contains(d.mapPane.m.Overlays(), serviceAreaID) {
		t.Errorf("%s: the library does not hold the service area", mode)
	}
	if !slices.ContainsFunc(d.mapPane.m.Places(), func(p tuimaps.Place) bool { return p.ID == towerPlaceID }) {
		t.Errorf("%s: the tower is not marked", mode)
	}
}

// TestTheServiceAreaIsAlwaysDrawn is FR-2.1 (D-195): on the Broadcaster the
// service area and the tower are drawn in Radar mode, in Forecast mode and in
// the Propagation mode, and follow a new radius saved in Settings; Observer's
// map has neither.
func TestTheServiceAreaIsAlwaysDrawn(t *testing.T) {
	d := stationMap(t, Config{})
	serviceArea(t, "radar", d, bonsall.Lat, bonsall.Lon, 100)
	serviceArea(t, "forecast", pressMap(t, d, "R"), bonsall.Lat, bonsall.Lon, 100)
	prop := chartDash(t) // the Propagation mode, opened on Observer, then swapped to the console
	tx := bonsall
	prop.cfg.Transmitter, prop.cfg.ServiceRadiusMi, prop.surface = &tx, 100, SurfaceBroadcaster
	prop = prop.renderMap()
	serviceArea(t, "propagation", prop, bonsall.Lat, bonsall.Lon, 100)
	if !near(prop, bonsall.Lat, bonsall.Lon) {
		t.Error("a swap to the console with the map open leaves the map off the station")
	}
	d.cfg.SetServiceRadius, d.cfg.ServiceRadiusMinMi, d.cfg.ServiceRadiusMaxMi = func(int) {}, 10, 500
	d.setup.serviceMi = "60"
	serviceArea(t, "saved", d.commitToModel(), bonsall.Lat, bonsall.Lon, 60)
	obs := openMap(t, Config{ServiceRadiusMi: 100}, 133, 44)
	if len(obs.mapPane.zoneGiven) != 0 || slices.ContainsFunc(obs.mapPane.m.Places(), func(p tuimaps.Place) bool { return p.ID == towerPlaceID }) {
		t.Error("Observer's map draws the station's service area or tower")
	}
}

// TestTheBroadcastersMapOffersNoPlaceKeys is FR-2.4 (D-195): on the
// Broadcaster [ and ] do not move the map off the station, the controls box
// says what the red ring is in their place and Help does not list them;
// Observer's map steps through its places.
func TestTheBroadcastersMapOffersNoPlaceKeys(t *testing.T) {
	two := placedSnap()
	two.Locations = append(two.Locations, snapshot.Location{Label: "Escondido, CA", Zip: "92025", Lat: 33.12, Lon: -117.09})
	open := func(surface Surface) Dashboard {
		tx := bonsall
		d := mapDash(t, Config{Transmitter: &tx, ServiceRadiusMi: 100})
		m, _ := d.Update(SnapshotMsg{Snap: two})
		d = m.(Dashboard)
		d.surface = surface
		d, _ = pressKey(d, "g")
		return feedAndSettle(t, d)
	}
	bc := open(SurfaceBroadcaster)
	moved := pressMap(t, bc, "]")
	if moved.selected != bc.selected || !near(moved, bonsall.Lat, bonsall.Lon) {
		t.Error("] on the Broadcaster's map moved it off the station")
	}
	if box := stripANSITest(strings.Join(bc.controlsBox(), "\n")); strings.Contains(box, "place") || !strings.Contains(box, "red: service") {
		t.Errorf("the Broadcaster's controls box offers the place keys, or does not say the red ring:\n%s", box)
	}
	if help := stripANSITest(strings.Join(bc.helpLines(bc.opts()), "\n")); strings.Contains(help, "Previous / Next Location") || !strings.Contains(help, " MAP") || !strings.Contains(help, "Pan") {
		t.Errorf("the Broadcaster's Help lists the place keys, or not the map's group:\n%s", help)
	}
	obs := open(SurfaceObserver)
	if stepped := pressMap(t, obs, "]"); stepped.selected == obs.selected {
		t.Error("control: ] on Observer's map does not step to the next place, so this proves nothing")
	}
	if box := stripANSITest(strings.Join(obs.controlsBox(), "\n")); !strings.Contains(box, "place") {
		t.Errorf("Observer's controls box does not offer the place keys:\n%s", box)
	}
}

// TestTheServiceAreaIsSaid is FR-2.4 (D-195, FR-3.1): without the picture the
// words say the red circle is the station's service area, its radius and its
// tower; Observer's say nothing of one.
func TestTheServiceAreaIsSaid(t *testing.T) {
	words := strings.Join(stationMap(t, Config{}).describeLinesAll(), " ")
	if !strings.Contains(words, "The red circle is the station's service area: ") || !strings.Contains(words, "around Bonsall, CA") {
		t.Errorf("the words do not say the service area: %q", words)
	}
	if obs := strings.Join(openMap(t, Config{ServiceRadiusMi: 100}, 133, 44).describeLinesAll(), " "); strings.Contains(obs, "service area") {
		t.Errorf("Observer's words speak of a service area: %q", obs)
	}
}

// TestTheNearVerticalAnswersUseTheServiceRadius is FR-2.3 (D-195): on the
// Broadcaster the near-vertical answers are to the service radius, said so;
// on Observer to the Setting's, NVIS's.
func TestTheNearVerticalAnswersUseTheServiceRadius(t *testing.T) {
	d := stationDash(t)
	tx := bonsall
	d.cfg.Transmitter, d.cfg.ServiceRadiusMi = &tx, 100
	d = workChart(t, d.renderMap())
	if k := d.mapPane.propChart.key.radiusKm; k != 161 {
		t.Errorf("the Broadcaster's chart answers to %.0f km; the service radius is 161", k)
	}
	if block := strings.Join(d.propBlock(d.scrubW()), "\n"); !strings.Contains(stripANSITest(block), "Local, to ~161 km (service area): ") {
		t.Errorf("the block does not say the service radius:\n%s", stripANSITest(block))
	}
	if block := stripANSITest(strings.Join(chartDash(t).propBlock(80), "\n")); !strings.Contains(block, "km (NVIS): ") {
		t.Errorf("Observer's block does not answer to the NVIS radius:\n%s", block)
	}
}

// TestTheStationIsTheWatchlistsOwnRecord is D-195: a transmitter at a
// watched place is that place's own record, so its alerts and conditions
// come with it; one elsewhere is the place Settings gave.
func TestTheStationIsTheWatchlistsOwnRecord(t *testing.T) {
	d := mapDash(t, Config{})
	d.surface = SurfaceBroadcaster
	own := d.snap.Locations[0]
	d.cfg.Transmitter = &snapshot.LocationRef{Label: "the station", Lat: own.Lat, Lon: own.Lon}
	if got := d.stationPlace(); got != &d.snap.Locations[0] || d.noTransmitter() {
		t.Errorf("a transmitter at %s is %+v, not the watchlist's record", own.Label, got)
	}
	tx := bonsall
	d.cfg.Transmitter = &tx
	if got := d.stationPlace(); got == nil || got.Label != bonsall.Label || d.noTransmitter() {
		t.Errorf("a transmitter elsewhere is %+v; the place Settings gave", got)
	}
	d.cfg.Transmitter = nil
	if got := d.stationPlace(); got != d.selectedLocation() || !d.noTransmitter() {
		t.Errorf("with no transmitter the station is %+v; the selected place, standing in", got)
	}
}
