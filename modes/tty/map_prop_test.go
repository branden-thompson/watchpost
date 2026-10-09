package tty

import (
	"math"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/term"
)

// pressMap presses a key the map binds and settles what it asked for.
func pressMap(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	m, cmd := d.Update(mapKeyMsg(t, key))
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

// TestThePropagationModeIsAKeymapAction is W2.1 (FR-1.1, D-153, D-154): P
// is a map action Help lists and [keys] can rebind; it enters the
// Propagation mode from Radar or Forecast and returns to the mode it came
// from; R leaves it to Radar; it is never saved, so the next open is the
// weather mode last used.
func TestThePropagationModeIsAKeymapAction(t *testing.T) {
	b := defaultMapKeyMap()[actMapProp]
	if !slices.Equal(b.Keys, []string{"P"}) || b.Help != "Propagation On / Off" || !slices.Contains(mapActions, actMapProp) {
		t.Fatalf("P is %+v, listed %v; want P, Propagation On / Off, among the map's actions", b, slices.Contains(mapActions, actMapProp))
	}
	var help []string
	for _, r := range mapHelpRows(defaultMapKeyMap(), false) {
		help = append(help, r.keys+"  "+r.help)
	}
	if text := strings.Join(help, "\n"); !strings.Contains(text, "P") || !strings.Contains(text, "Propagation") {
		t.Errorf("Help does not list P and the Propagation mode:\n%s", text)
	}
	if keys, err := mapKeysFrom(term.KeyMap{actMapProp: {Keys: []string{"ctrl+p"}}}); err != nil || !slices.Equal(keys[actMapProp].Keys, []string{"ctrl+p"}) {
		t.Errorf("[keys] did not rebind P: %v, %v", keys[actMapProp].Keys, err)
	}
	for _, from := range []struct {
		name  string
		radar bool
		want  mapMode
	}{{"Radar", true, modeRadar}, {"Forecast", false, modeForecast}} {
		var asks []MapAsk
		d := openTempMap(t, from.radar, &asks)
		choice := d.mapLayerChoice
		if d = pressMap(t, d, "P"); d.mapMode() != modePropagation {
			t.Fatalf("from %s, P gave mode %d", from.name, d.mapMode())
		}
		if d.mapLayerChoice != choice || d.setup.uiDirty {
			t.Errorf("from %s, the Propagation mode was saved (D-154)", from.name)
		}
		if d = pressMap(t, d, "P"); d.mapMode() != from.want {
			t.Errorf("P from the Propagation mode gave %d; want %s again", d.mapMode(), from.name)
		}
		if d = pressMap(t, pressMap(t, d, "P"), "R"); d.mapMode() != modeRadar {
			t.Errorf("from %s, R in the Propagation mode gave %d; want Radar (D-153)", from.name, d.mapMode())
		}
		d = pressMap(t, pressMap(t, d, "P"), "g") // closed in the Propagation mode
		if d = pressMap(t, d, "g"); d.mapMode() != modeRadar {
			t.Errorf("from %s, the map reopened in mode %d; want the weather mode last used (D-154)", from.name, d.mapMode())
		}
	}
}

// TestTheModeIsNamedInEveryPath is W2.1 (FR-1.1): with the picture, with
// the description instead and under --ascii, the window names the
// Propagation mode in words.
func TestTheModeIsNamedInEveryPath(t *testing.T) {
	for _, mode := range screenModes {
		d := pressMap(t, screenDash(t, mode.desc, mode.ascii), "P")
		screen := stripANSITest(d.View().Content)
		if !strings.Contains(screen, "PROPAGATION") {
			t.Errorf("%s: the window does not name the Propagation mode:\n%s", mode.name, screen)
		}
		if mode.desc != "off" && !strings.Contains(strings.Join(d.describeLinesAll(), " "), "Propagation mode") {
			t.Errorf("%s: the description does not say the Propagation mode: %q", mode.name, d.describeLinesAll())
		}
	}
}

// TestEachModeDrawsOnlyItsLayers is W2.2 (FR-1.4, D-21): in the Propagation
// mode no weather layer is drawn - no radar loop, no temperature or wind,
// no alert area, no badge for one - and none is asked for; back in a
// weather mode they return.
func TestEachModeDrawsOnlyItsLayers(t *testing.T) {
	var asks []MapAsk
	d := openFieldsMap(t, true, true, true, &asks)
	if len(d.mapPane.given) == 0 || len(d.mapPane.radarGiven) == 0 || len(d.badges()) == 0 {
		t.Fatalf("the weather mode drew %d areas, %d radar frames, %d badges; the test needs some", len(d.mapPane.given), len(d.mapPane.radarGiven), len(d.badges()))
	}
	asked := len(asks)
	m, _ := d.Update(mapKeyMsg(t, "P"))
	if now := m.(Dashboard); len(now.mapPane.given) != 0 {
		t.Errorf("the Propagation mode drew %d weather areas before the feed's next answer", len(now.mapPane.given))
	}
	d = pressMap(t, d, "P")
	for _, key := range []string{AlertLayer, RadarLayer, TemperatureLayer, WindLayer, RainLayer, FeelsLayer, WaveLayer, UVLayer, AirLayer} {
		if d.layerOn(key) {
			t.Errorf("%s is on in the Propagation mode", key)
		}
	}
	if len(d.mapPane.given) != 0 || len(d.mapPane.radarGiven) != 0 || len(d.badges()) != 0 || len(d.tempOverlays()) != 0 {
		t.Errorf("the Propagation mode draws %d areas, %d radar frames, %d badges, %d temperature grids", len(d.mapPane.given), len(d.mapPane.radarGiven), len(d.badges()), len(d.tempOverlays()))
	}
	if len(asks) != asked {
		t.Errorf("entering the Propagation mode asked for the temperature %d times", len(asks)-asked)
	}
	if d = d.retime(); len(d.mapPane.given) != 0 {
		t.Errorf("a step's re-timing in the Propagation mode drew %d weather areas again", len(d.mapPane.given))
	}
	d = pressMap(t, d, "P")
	if len(d.mapPane.given) == 0 || len(d.mapPane.radarGiven) == 0 || len(d.badges()) == 0 {
		t.Errorf("back in Radar mode: %d areas, %d radar frames, %d badges", len(d.mapPane.given), len(d.mapPane.radarGiven), len(d.badges()))
	}
}

// TestEachModeKeepsItsOwnBound is W2.2 (FR-1.2, D-21): the Propagation mode
// is held to no region - its frames go past Oceanside's - and on leaving it
// every frame is inside a region again, the place's or a neighbour's.
func TestEachModeKeepsItsOwnBound(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	region := d.mapPane.region
	views := &[]mapView{}
	d.mapPane.views = views
	d = pressMap(t, d, "P")
	for range 12 {
		d = pressCode(d, '-', "-")
	}
	if _, z := d.mapPane.m.Centre(); z < worldZoom(d.mapBodySize())-1e-9 {
		t.Fatalf("the Propagation mode zoomed out to %v, past the whole world (%v)", z, worldZoom(d.mapBodySize()))
	}
	d = driveEverywhere(d)
	wider := 0
	for _, v := range *views {
		if v.mode != modePropagation {
			t.Fatalf("a frame in mode %d while the Propagation mode was on", v.mode)
		}
		if !inside(v, region) {
			wider++
		}
	}
	if wider == 0 {
		t.Errorf("none of %d Propagation frames went past %s", len(*views), region.Name)
	}
	*views = nil
	d = driveEverywhere(pressMap(t, d, "P"))
	for _, v := range *views { // a pan across an edge moves to the neighbour (D-77): each frame inside its own region
		if v.mode == modePropagation || v.region.Name == "" || !inside(v, v.region) {
			t.Fatalf("back in a weather mode a frame is outside its region: %+v", v)
		}
	}
	if len(*views) < 100 {
		t.Errorf("%d weather frames drawn after leaving; the drive draws more", len(*views))
	}
}

// TestThePropagationModePansAndZooms is W2.2 (FR-1.7): it zooms out past
// any region's least zoom, to the whole world, and pans across an ocean.
func TestThePropagationModePansAndZooms(t *testing.T) {
	d := pressMap(t, openMap(t, Config{}, 133, 44), "P")
	least := regionFitZoom(d.mapPane.region, d.mapBodySize())
	for range 12 {
		d = pressCode(d, '-', "-")
	}
	if _, z := d.mapPane.m.Centre(); z >= least || z < worldZoom(d.mapBodySize())-1e-9 {
		t.Errorf("zoomed out to %v; want below the region's least, %v, and no further than the world's, %v", z, least, worldZoom(d.mapBodySize()))
	}
	for range 4 {
		d = pressCode(d, '+', "+")
	}
	start, _ := d.mapPane.m.Centre()
	for range 30 {
		d = pressCode(d, tea.KeyLeft, "")
	}
	if c, _ := d.mapPane.m.Centre(); math.Mod(start.Lon-c.Lon+360, 360) < 30 {
		t.Errorf("30 pans west moved from %v to %v, not out over the Pacific", start.Lon, c.Lon)
	}
}

// londonDash is the map's dashboard with London, a place in no region,
// selected.
func londonDash(t *testing.T) Dashboard {
	t.Helper()
	s := placedSnap()
	s.Locations[0].Label, s.Locations[0].Lat, s.Locations[0].Lon = "London", 51.5, -0.12
	m, _ := mapDash(t, Config{}).Update(SnapshotMsg{Snap: s})
	return m.(Dashboard)
}

// TestAPlaceOutsideEveryRegionOpensThePropagationMode is W2.3 (FR-1.3): a
// place in no region draws no weather map, and P draws the Propagation
// mode there, on the place; P again gives the stated state back.
func TestAPlaceOutsideEveryRegionOpensThePropagationMode(t *testing.T) {
	d := londonDash(t)
	views := &[]mapView{}
	d.mapPane.views = views
	d, _ = pressKey(d, "g")
	if !strings.Contains(bodyText(d), "London is outside") || len(*views) != 0 {
		t.Fatalf("a weather mode for London drew %d frames: %s", len(*views), bodyText(d))
	}
	d = pressMap(t, d, "P")
	if strings.Contains(bodyText(d), "is outside") || len(*views) == 0 {
		t.Fatalf("the Propagation mode for London drew %d frames: %s", len(*views), bodyText(d))
	}
	if c := (*views)[len(*views)-1].centre; math.Abs(c.Lat-51.5) > 1 || math.Abs(c.Lon+0.12) > 1 {
		t.Errorf("the Propagation mode is centred on %v, not London", c)
	}
	if d = pressMap(t, d, "P"); !strings.Contains(bodyText(d), "London is outside") {
		t.Errorf("leaving the Propagation mode for London: %s", bodyText(d))
	}
}

// TestTheOutsideMessageNamesThePropagationKey is W2.3 (FR-1.3): the stated
// state names the key that opens the Propagation mode, as it is bound.
func TestTheOutsideMessageNamesThePropagationKey(t *testing.T) {
	d, _ := pressKey(londonDash(t), "g")
	if text := bodyText(d); !strings.Contains(text, "Press P for the Propagation mode") {
		t.Errorf("the outside message does not name P: %s", text)
	}
	keys, err := mapKeysFrom(term.KeyMap{actMapProp: {Keys: []string{"ctrl+p"}}})
	if err != nil {
		t.Fatal(err)
	}
	d.mapKeys = keys
	if text := bodyText(d); !strings.Contains(text, "Press ctrl+p for the Propagation mode") {
		t.Errorf("the outside message does not follow [keys]: %s", text)
	}
}

// TestNoRegionIsNamedInThePropagationMode: the region keys still show a
// region in the Propagation mode, unbound, and a pan at the world's edge
// shows no neighbour's chip.
func TestNoRegionIsNamedInThePropagationMode(t *testing.T) {
	d := pressMap(t, openMap(t, Config{}, 133, 44), "P")
	d = pressCode(d, '2', "2") // Alaska
	if c, _ := d.mapPane.m.Centre(); c.Lat < 50 {
		t.Errorf("2 in the Propagation mode left the view at %v, not Alaska", c)
	}
	for range 80 {
		d = pressCode(d, tea.KeyUp, "")
	}
	if d.mapPane.edgeShown {
		t.Error("the Propagation mode, bound to no region, showed a neighbour's chip")
	}
	if _, ok := geo.RegionNumbered(2); !ok {
		t.Fatal("no region 2")
	}
}
