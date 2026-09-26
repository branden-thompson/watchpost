package tty

// map_detail_test.go — 0.18.0 UAT-1 U1-16 with D-67 (go-tuiMaps D-82): the
// map opens at the Weather detail level; the level is a Settings row and a
// row of the Overlays menu; a layer the level does not draw says the level
// that would.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheDetailLevelIsASettingAndReachesTheMap: Weather by default; → on the
// row moves it to Standard, which reaches the library and is written; the
// list marks what the level leaves out.
func TestTheDetailLevelIsASettingAndReachesTheMap(t *testing.T) {
	d, got := uiDash(t, rowMapDetailLevel)
	if d.mapDetailLevel.String() != "weather" {
		t.Fatalf("an empty file opens at %v, want weather", d.mapDetailLevel)
	}
	body, _, _ := d.focusBody(d.opts())
	text := stripANSITest(strings.Join(body, "\n"))
	for _, want := range []string{"Detail level -", "Weather", "Minor roads (at Full)", "Rail (at Standard)"} {
		if !strings.Contains(text, want) {
			t.Errorf("the Maps tab does not show %q:\n%s", want, text)
		}
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if d.mapDetailLevel.String() != "standard" {
		t.Errorf("→ went to %v, want standard", d.mapDetailLevel)
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapDetailLevel != "standard" {
		t.Errorf("esc wrote %q, want standard", got.MapDetailLevel)
	}
	if detailLevelByKey("nonsense").String() != "weather" {
		t.Error("an unknown word is not the default")
	}
}

// TestTheMenusDetailRowStepsTheLevel: space on the menu's level row steps it
// and the library is told at once.
func TestTheMenusDetailRowStepsTheLevel(t *testing.T) {
	calls := &[]string{}
	d := mapDash(t, Config{MapLayers: alertLayers, MapFeed: boxFeed(-117.6, -117.1, false)})
	d.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	d = pressCode(d, 'O', "O")
	var keys []string
	for _, r := range d.overlayRows() {
		keys = append(keys, r.key)
	}
	d.mapPane.menuAt = indexOf(keys, detailLevelKey)
	*calls = nil
	m, _, ok := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !ok || m.(Dashboard).mapDetailLevel.String() != "standard" {
		t.Fatalf("space on the level row: handled %v, level %v", ok, m.(Dashboard).mapDetailLevel)
	}
	if !strings.Contains(strings.Join(*calls, " "), "SetDetail:standard") {
		t.Errorf("the library was not told: %v", *calls)
	}
}

// TestADetailLevelChangesThePicture: the level reaches the library, so
// Essential draws without the place names Weather draws.
func TestADetailLevelChangesThePicture(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	before := strings.Join(d.mapPane.lines, "\n")
	d = d.cycleDetailLevel(false) // Weather to Essential
	d = settleMap(t, d.renderMap())
	if strings.Join(d.mapPane.lines, "\n") == before {
		t.Error("Essential drew the picture Weather drew")
	}
}
