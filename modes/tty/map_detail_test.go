package tty

// map_detail_test.go — 0.18.0 UAT-1 U1-16 with D-67 (go-tuiMaps D-82): the
// map opens at the Weather detail level; the level is a Settings row and a
// row of the Overlays menu; a layer the level does not draw says the level
// that would.

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestTheDetailLevelIsASettingAndReachesTheMap: Weather's preset - Standard
// (D-144) - by default; → on the row moves it to All, which reaches the
// library and is written; the list marks what the level leaves out.
func TestTheDetailLevelIsASettingAndReachesTheMap(t *testing.T) {
	d, got := uiDash(t, rowMapDetailLevel)
	if d.mapModel.mapDetailLevel.String() != "weather" {
		t.Fatalf("an empty file opens at %v, want weather", d.mapModel.mapDetailLevel)
	}
	body, _, _ := d.focusBody(d.opts())
	text := ttytest.StripANSI(strings.Join(body, "\n"))
	for _, want := range []string{"Detail -", "Standard", "(county zoom)", "(state zoom)"} {
		if !strings.Contains(text, want) {
			t.Errorf("the Maps tab does not show %q:\n%s", want, text)
		}
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m
	if d.mapModel.mapDetailLevel.String() != "full" {
		t.Errorf("→ went to %v, want full - All (D-144)", d.mapModel.mapDetailLevel)
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapDetailLevel != "full" {
		t.Errorf("esc wrote %q, want full", got.MapDetailLevel)
	}
	if detailLevelByKey("standard").String() != "full" {
		t.Error("a saved Standard - the old level between Weather and Full - is not All (D-144)")
	}
	if detailLevelByKey("nonsense").String() != "weather" {
		t.Error("an unknown word is not the default")
	}
}

// TestTheMenusDetailRowStepsTheLevel: space on the menu's level row steps it
// and the library is told at once.
func TestTheMenusDetailRowStepsTheLevel(t *testing.T) {
	calls := &[]string{}
	d := mapDash(t, Config{MapLayers: alertLayers, Map: &Hooks{MapFeed: boxFeed(-117.6, -117.1, false)}})
	d.mapModel.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	d = pressCode(d, 'O', "O")
	var keys []string
	for _, r := range d.mapModel.overlayRows(d.mapEnv()) {
		keys = append(keys, r.key)
	}
	d.mapModel.mapPane.menuAt = slices.Index(keys, detailLevelKey)
	*calls = nil
	m, _, ok := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !ok || m.mapModel.mapDetailLevel.String() != "full" {
		t.Fatalf("space on the level row: handled %v, level %v", ok, m.mapModel.mapDetailLevel)
	}
	if !strings.Contains(strings.Join(*calls, " "), "Layers:rail:on") || !strings.Contains(strings.Join(*calls, " "), "Layers:parks:on") {
		t.Errorf("the level's preset did not reach the library's switches (D-79): %v", *calls)
	}
}

// TestTheLevelIsAPresetAndTheSwitchesAreTheTruth is D-79 (UAT-1 U1-42): the
// library draws all it has and the switches thin it, so a switch on is drawn
// whatever the level; a level sets every switch to its preset; a switch
// changed from the preset makes the level read "Custom"; the minor roads are
// never drawn and never offered.
func TestTheLevelIsAPresetAndTheSwitchesAreTheTruth(t *testing.T) {
	calls := &[]string{}
	d := mapDash(t, Config{Map: &Hooks{MapFeed: boxFeed(-117.6, -117.1, false)}})
	d.mapModel.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	joined := strings.Join(*calls, " ")
	for _, want := range []string{"SetDetail:full", "Layers:minor-roads:off", "Layers:rail:off", "Layers:parks:off", "Layers:roads:on"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the Weather preset was not applied as switches: no %s in %v", want, *calls)
		}
	}
	if d.mapModel.detailLevelShown() != "Standard" {
		t.Errorf("an untouched preset reads %q; want Standard (D-144)", d.mapModel.detailLevelShown())
	}
	*calls = nil
	d.mapModel = d.mapModel.setDetail("rail", true, d.mapEnv())
	if !strings.Contains(strings.Join(*calls, " "), "Layers:rail:on") || strings.Contains(strings.Join(*calls, " "), "SetDetail:weather") {
		t.Errorf("rail switched on at Weather did not reach the library as on: %v", *calls)
	}
	if d.mapModel.detailLevelShown() != "Custom" {
		t.Errorf("a switch off its preset reads %q, want Custom", d.mapModel.detailLevelShown())
	}
	d.mapModel = d.mapModel.cycleDetailLevel(true, d.mapEnv()) // All: rail and parks on by the preset
	if !d.mapModel.detailOn("rail") || !d.mapModel.detailOn("parks") || d.mapModel.detailLevelShown() != "All" {
		t.Errorf("All's preset: rail %v, parks %v, reads %q", d.mapModel.detailOn("rail"), d.mapModel.detailOn("parks"), d.mapModel.detailLevelShown())
	}
	d.mapModel = d.mapModel.cycleDetailLevel(false, d.mapEnv()) // back to Standard: the preset again, not the old switch
	if d.mapModel.detailOn("rail") || d.mapModel.detailLevelShown() != "Standard" {
		t.Errorf("Standard again left rail %v and reads %q", d.mapModel.detailOn("rail"), d.mapModel.detailLevelShown())
	}
	for _, l := range mapDetailLayers() {
		if l.layer == tuimaps.MinorRoadLayer || strings.Contains(l.label, "Minor") {
			t.Errorf("the minor roads are still offered: %+v", l)
		}
	}
}

// TestADetailLevelChangesThePicture: the level reaches the library, so
// Essential draws without the place names Weather draws.
func TestADetailLevelChangesThePicture(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	before := strings.Join(d.mapModel.mapPane.lines, "\n")
	d.mapModel = d.mapModel.cycleDetailLevel(false, d.mapEnv()) // Weather to Essential
	d = settleMap(t, withModel(d, d.mapModel.renderMap(d.mapEnv())))
	if strings.Join(d.mapModel.mapPane.lines, "\n") == before {
		t.Error("Essential drew the picture Weather drew")
	}
}
