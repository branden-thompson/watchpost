package tty

// map_prefs_test.go — 0.18.0 batch 9: the Maps tab's default scale (W1.11,
// W4.3, FR-2.3), the nearby distance (W9.2 folded, FR-7.4), the layers the
// registry names (W1.11, W1.13, FR-9.3) and the cost warning (W1.14, FR-9.2).

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// alertLayers is the registry as the app hands it today: one layer.
var alertLayers = []MapLayer{{Key: "alert", Label: "Alert areas", On: true}}

// TestTheDefaultScaleOpensTheMapAsChosen is W4.3 (FR-2.3): the map opens at
// the chosen scale - the region, a state, a county - on every open, while the
// region's bound holds whatever is chosen.
func TestTheDefaultScaleOpensTheMapAsChosen(t *testing.T) {
	zoomAt := func(scale string) float64 {
		d := mapDash(t, Config{MapScale: scale})
		d, _ = pressKey(d, "g")
		_, z := d.mapPane.m.Centre()
		return z
	}
	region, state, county := zoomAt("region"), zoomAt("state"), zoomAt("county")
	if !(region < state && state < county) {
		t.Errorf("region %v, state %v, county %v: want each closer than the last", region, state, county)
	}
	if def := zoomAt(""); def != state {
		t.Errorf("an empty file opens at %v, want the state scale %v", def, state)
	}
	d := mapDash(t, Config{MapScale: "county"})
	d, _ = pressKey(d, "g")
	d = pressCode(d, '-', "-")
	d, _ = pressKey(d, "g")
	d, _ = pressKey(d, "g")
	if _, z := d.mapPane.m.Centre(); z != county {
		t.Errorf("reopened at %v, want the chosen scale %v again", z, county)
	}
}

// TestTheScaleAndNearbyRowsSave is W1.11's round trip: → moves each picker,
// and the group writes the words on close.
func TestTheScaleAndNearbyRowsSave(t *testing.T) {
	d, got := uiDash(t, rowMapScale)
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if d.mapScale != mapScaleCounty {
		t.Errorf("→ from the state scale went to %v, want county", d.mapScale)
	}
	d.setup.focus = rowMapNearby
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if d.mapNearbyKm != 25 {
		t.Errorf("→ from 15 km went to %d, want 25", d.mapNearbyKm)
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapScale != "county" || got.MapNearbyKm != 25 {
		t.Errorf("esc wrote scale %q and nearby %d; want county and 25", got.MapScale, got.MapNearbyKm)
	}
	body, _, _ := d.focusBody(d.opts())
	text := stripANSITest(strings.Join(body, "\n"))
	for _, want := range []string{"Opens at -", "County", "Nearby -"} {
		if !strings.Contains(text, want) {
			t.Errorf("the Maps tab does not show %q:\n%s", want, text)
		}
	}
}

// TestNearbyReadsInTheStationsUnits: the distance is named in the units the
// description speaks.
func TestNearbyReadsInTheStationsUnits(t *testing.T) {
	d, _ := uiDash(t, rowMapNearby)
	d.units = render.UnitC
	if got := d.nearbyLabel(); got != "15 km" {
		t.Errorf("in Celsius nearby reads %q, want 15 km", got)
	}
	d.units = render.UnitF
	if got := d.nearbyLabel(); got != "9 miles (15 km)" {
		t.Errorf("in Fahrenheit nearby reads %q", got)
	}
	if k := mapNearbyByKm(7); k != mapNearbyDefaultKm {
		t.Errorf("a distance not on the list reads as %d, want the default", k)
	}
}

// TestTheNearbySettingReachesTheDescription is W9.2's "nearby" test: an edge
// about 12 km off stops short of the place at the default - M1's 15 km, where
// the library's own 10 would say it lies to one side - and lies to one side
// at 5 km.
func TestTheNearbySettingReachesTheDescription(t *testing.T) {
	for _, c := range []struct {
		km   int
		want string
	}{{0, "Wind Warning in effect for nearby"}, {5, "Wind Warning in effect for the area it covers"}} {
		d := mapDash(t, Config{ASCII: true, MapNearbyKm: c.km, MapFeed: boxFeed(-117.25, -117.0, false)})
		s := placedSnap()
		s.Locations[0].Alerts = []snapshot.Alert{{ID: "w1", Event: "Wind Warning", Severity: "Severe", Expires: windExpires}}
		m, _ := d.Update(SnapshotMsg{Snap: s})
		d = m.(Dashboard)
		d, _ = pressKey(d, "g")
		d = feedAndSettle(t, d)
		if out := unwrapped(stripANSITest(d.View().Content)); !strings.Contains(out, c.want) {
			t.Errorf("nearby %d km: want %q:\n%s", c.km, c.want, out)
		}
	}
}

// TestALayerSwitchedOffIsNotDrawn is W1.11's layers (FR-9.1, R-9.2): a layer
// off draws none of its overlays - they are keyed by the layer's name - and
// the description says the layer is off rather than that nothing is there.
func TestALayerSwitchedOffIsNotDrawn(t *testing.T) {
	d := mapDash(t, Config{MapLayers: alertLayers, MapLayerChoice: map[string]bool{"alert": false}, MapFeed: boxFeed(-117.6, -117.1, false)})
	calls := &[]string{}
	d.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	d = pressCode(d, 'A', "A") // the Area Alerts box, closed on open (D-87)
	for _, c := range *calls {
		if c == "Set" {
			t.Fatal("an overlay of a layer switched off was drawn")
		}
	}
	if text := stripANSITest(strings.Join(d.mapBodyLines(), "\n")); !strings.Contains(text, "Alert areas are switched off") {
		t.Errorf("the description does not say the alert areas are off:\n%s", text)
	}
	on := mapDash(t, Config{MapLayers: alertLayers, MapFeed: boxFeed(-117.6, -117.1, false)})
	on, _ = pressKey(on, "g")
	on = feedAndSettle(t, on)
	if len(on.mapPane.shown) != 1 {
		t.Errorf("with the layer on %d overlays are shown, want 1", len(on.mapPane.shown))
	}
}

// TestTheLayersRowTogglesAndSaves: MAP - LAYERS is a picker a layer (HUM
// LEAD, 2026-09-30); space switches the layer under the cursor, and the
// group writes the choice on close.
func TestTheLayersRowTogglesAndSaves(t *testing.T) {
	d, got := uiDash(t, rowMapLayers)
	d.cfg.MapLayers = alertLayers
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "MAP - LAYERS") || !strings.Contains(text, "Alert areas -") || !strings.Contains(text, "Enabled") {
		t.Errorf("the layers group does not name the layer with its picker:\n%s", text)
	}
	d = d.setupSpace()
	if d.layerOn("alert") {
		t.Error("space did not switch the alert areas off")
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if on, ok := got.MapLayers["alert"]; !ok || on {
		t.Errorf("esc wrote layers %v; want alert off", got.MapLayers)
	}
}

// A LAYER'S ROW IS A PICKER, AND ↑↓ WALK THE LAYERS (HUM LEAD, 2026-09-30):
// ←→ switch the layer under the cursor, as every Enabled / Disabled picker
// does, one layer or many; ↓ walks the layers and past the last leaves the
// group; ↑ from the row below comes back to the last layer.
func TestALayersRowIsAPickerAndTheArrowsWalkThem(t *testing.T) {
	d, _ := uiDash(t, rowMapLayers)
	d.cfg.MapLayers = append(append([]MapLayer(nil), alertLayers...), MapLayer{Key: "quake", Label: "Quakes"})
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	one := m.(Dashboard)
	if one.setupTab() != tabMaps || one.layerOn("alert") {
		t.Fatalf("→ on Alert areas: tab %s, alert on %v; want it switched off, here", one.setupTab().Label(), one.layerOn("alert"))
	}
	m, _ = one.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
	two := m.(Dashboard)
	if two.setup.focus != rowMapLayers || two.setup.layerAt != 1 {
		t.Fatalf("↓ went to row %d, layer %d; want the next layer", two.setup.focus, two.setup.layerAt)
	}
	m, _, _ = two.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if !m.(Dashboard).layerOn("quake") || m.(Dashboard).layerOn("alert") {
		t.Error("← did not switch on the layer under the cursor alone")
	}
	m, _ = two.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
	below := m.(Dashboard)
	if below.setup.focus == rowMapLayers {
		t.Fatal("↓ past the last layer stayed in the group")
	}
	m, _ = below.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if back := m.(Dashboard); back.setup.focus != rowMapLayers || back.setup.layerAt != 1 {
		t.Errorf("↑ from below came to row %d, layer %d; want the last layer", back.setup.focus, back.setup.layerAt)
	}
	m, _ = one.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.(Dashboard).setup.focus == rowMapLayers {
		t.Error("↑ from the first layer stayed in the group")
	}
	above := m.(Dashboard)
	above.setup.layerAt = 1
	m, _ = above.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if into := m.(Dashboard); into.setup.focus != rowMapLayers || into.setup.layerAt != 0 {
		t.Errorf("↓ from above came to row %d, layer %d; want the first layer", into.setup.focus, into.setup.layerAt)
	}
	single, _ := uiDash(t, rowMapLayers)
	single.cfg.MapLayers = alertLayers
	m, _, _ = single.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if got := m.(Dashboard); got.setupTab() != tabMaps || got.layerOn("alert") {
		t.Error("with one layer → did not switch it, here")
	}
	single.cfg.MapLayers = nil
	if single.rowVisible(rowMapLayers) {
		t.Error("with no layer registered, the layers row is still focusable")
	}
}

// TestTheCostWarningsThresholds is W1.14 (FR-9.2, D-43): at 2 MB and 40
// requests nothing is said; one byte or one request more and the station
// says so, in the HUM LEAD's words (D-82), the first sentence in bold.
func TestTheCostWarningsThresholds(t *testing.T) {
	for _, c := range []struct {
		cost MapCost
		want string
	}{
		{MapCost{Bytes: 1_000_000, Requests: 10}, ""},
		{MapCost{Bytes: 3_000_000, Requests: 25}, ""}, // D-149's floor
		{MapCost{Bytes: 3_000_001, Requests: 25}, "Map may experience performance issues at this zoom level. Est. 3.0MB / 25 Requests | Adjust layers/zoom to improve experience."},
		{MapCost{Bytes: 2_200_000, Requests: 211}, "Map may experience performance issues at this zoom level. Est. 2.2MB / 211 Requests | Adjust layers/zoom to improve experience."},
	} {
		head, detail := costWarningParts(c.cost)
		if got := strings.TrimSpace(head + " " + detail); got != c.want {
			t.Errorf("%+v: got %q, want %q", c.cost, got, c.want)
		}
	}
	t.Setenv("TERM", "xterm-256color") // bold shows only where styling does
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	lines := costWarningLines(MapCost{Bytes: 2_200_000, Requests: 211}, 200)
	if len(lines) != 2 || lines[0] != render.Bold("Map may experience performance issues at this zoom level.") || lines[0] == stripANSITest(lines[0]) {
		t.Errorf("the warning's lines are %q; want the first sentence in bold, then the estimate", lines)
	}
}

// TestTheCostWarningShowsBesideTheLayersAndOnTheMap: over the threshold, the
// words are under the layers row and under the map; the estimate is asked
// with the layers as chosen.
func TestTheCostWarningShowsBesideTheLayersAndOnTheMap(t *testing.T) {
	asked := map[bool]int{}
	cost := func(_ MapAsk, on func(string) bool) MapCost {
		asked[on("alert")]++
		if on("alert") {
			return MapCost{Bytes: 5_000_000, Requests: 515}
		}
		return MapCost{}
	}
	d := mapDash(t, Config{MapLayers: alertLayers, MapCost: cost, MapFeed: boxFeed(-117.6, -117.1, false)})
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	if text := stripANSITest(strings.Join(d.mapBodyLines(), "\n")); !strings.Contains(text, "Est. 5.0MB / 515 Requests") {
		t.Errorf("the map does not warn:\n%s", text)
	}
	d, _ = pressKey(d, "esc")
	d = d.openSetupAt(rowMapLayers)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Est. 5.0MB") {
		t.Errorf("Settings does not warn beside the layers:\n%s", text)
	}
	d = d.setupSpace()
	body, _, _ = d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); strings.Contains(text, "Est. 5.0MB") {
		t.Errorf("the warning stayed after the layer went off:\n%s", text)
	}
	if asked[false] == 0 {
		t.Error("the estimate was never asked with the layer off")
	}
}

// TestThePickersGoBackToo: ← walks the scale and the distance the other way,
// and wraps.
func TestThePickersGoBackToo(t *testing.T) {
	d, _ := uiDash(t, rowMapScale)
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := m.(Dashboard).mapScale; got != mapScaleRegion {
		t.Errorf("← from the state scale went to %v, want the region", got)
	}
	d.setup.focus = rowMapNearby
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := m.(Dashboard).mapNearbyKm; got != 10 {
		t.Errorf("← from 15 km went to %d, want 10", got)
	}
}

// TestTheAlertLayersNotesGoWithIt: with the alert areas off, the notes that
// speak for them are not printed either.
func TestTheAlertLayersNotesGoWithIt(t *testing.T) {
	noted := func(ctx context.Context, ask MapAsk) MapFeed {
		f := boxFeed(-117.6, -117.1, true)(ctx, ask)
		f.Notes = []string{"Wind Warning is drawn from 2 of its 3 zones."}
		return f
	}
	d := mapDash(t, Config{MapLayers: alertLayers, MapLayerChoice: map[string]bool{"alert": false}, MapFeed: noted})
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	if len(d.mapPane.notes) != 0 || len(d.mapPane.inMissing) != 0 {
		t.Errorf("with the alert areas off the window keeps notes %v and missing %v", d.mapPane.notes, d.mapPane.inMissing)
	}
}

// TestTheEstimateIsAskedWhereItCanChange: opening Settings asks it, and so
// does the map's new data.
func TestTheEstimateIsAskedWhereItCanChange(t *testing.T) {
	cost := func(ask MapAsk, _ func(string) bool) MapCost {
		s, n := ask.Snap, 0
		if s != nil && len(s.Locations) > 0 {
			n = len(s.Locations[0].Alerts)
		}
		return MapCost{Requests: 100 * n}
	}
	d := mapDash(t, Config{MapLayers: alertLayers, MapCost: cost, MapFeed: boxFeed(-117.6, -117.1, false)})
	d = d.openSetupAt(rowMapLayers)
	if d.mapCost.Requests == 0 {
		t.Error("opening Settings did not ask the estimate")
	}
	d, _ = pressKey(d, "esc")
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	was := d.mapCost.Requests
	s := placedSnap()
	s.Locations[0].Alerts = append(s.Locations[0].Alerts, snapshot.Alert{ID: "x"}, snapshot.Alert{ID: "y"})
	m, _ := d.Update(SnapshotMsg{Snap: s})
	d = feedAndSettle(t, m.(Dashboard))
	if d.mapCost.Requests == was {
		t.Errorf("new data left the estimate at %d requests", was)
	}
}

// TestAnAlertInViewIsDescribedInFull: an alert the station does not hold -
// one of the view's (D-66) - is described with its own name and the time it
// ends, from the feed.
func TestAnAlertInViewIsDescribedInFull(t *testing.T) {
	feed := func(ctx context.Context, ask MapAsk) MapFeed {
		f := boxFeed(-117.6, -117.1, false)(ctx, ask)
		f.InView = []snapshot.Alert{{ID: "w1", Event: "Tornado Warning", Severity: "Extreme", Expires: windExpires}}
		return f
	}
	d := mapDash(t, Config{ASCII: true, MapFeed: feed})
	s := placedSnap()
	s.Locations[0].TZ = "America/Los_Angeles"
	m, _ := d.Update(SnapshotMsg{Snap: s})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	out := unwrapped(stripANSITest(d.View().Content))
	if !strings.Contains(out, "Tornado Warning in effect for this area until") {
		t.Errorf("the alert in view is not described in full:\n%s", out)
	}
}

// TestTheWarningWaitsForTheOverlaysChosen is D-149's floor: 3 MB or 25
// requests of the overlays chosen, and not before.
func TestTheWarningWaitsForTheOverlaysChosen(t *testing.T) {
	for c, want := range map[MapCost]bool{{Bytes: 2_900_000, Requests: 24}: false, {Bytes: 3_000_000, Requests: 25}: false,
		{Bytes: 3_000_001, Requests: 1}: true, {Bytes: 100, Requests: 26}: true} {
		if head, _ := costWarningParts(c); (head != "") != want {
			t.Errorf("%+v warns %v; want %v", c, head != "", want)
		}
	}
}
