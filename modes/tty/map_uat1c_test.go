package tty

// map_uat1c_test.go — 0.18.0 UAT-1's fourth pass, the Settings window: air
// between the tabs and the page (U1-30), the map's detail as a row per layer
// in the "← Enabled →" pattern (U1-35), and a window no wider than 80% of the
// terminal with its two columns still side by side (U1-36).

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestABlankRowSetsTheTabsOffThePage is U1-30: where the window has the room,
// the row under the tabs is empty.
func TestABlankRowSetsTheTabsOffThePage(t *testing.T) {
	d, _ := uiDash(t, rowMapsOn)
	lines := strings.Split(stripANSITest(d.renderModal(d.opts())), "\n")
	if !strings.Contains(lines[1], "Maps ]") {
		t.Fatalf("the tabs are not the first row: %q", lines[1])
	}
	if strings.Trim(lines[2], "│ ") != "" {
		t.Errorf("the row under the tabs is %q, want it empty", lines[2])
	}
}

// TestEachDetailLayerIsAnEnabledRow is U1-35: each detail layer is a row of
// its own, "← Enabled →" as every other on/off row; ←, → and space each
// switch it; the choice reaches the saved Settings.
func TestEachDetailLayerIsAnEnabledRow(t *testing.T) {
	for i, l := range mapDetailLayers() {
		id := rowMapDetailBorders + setupRowID(i)
		d, got := uiDash(t, id)
		body, _, _ := d.focusBody(d.opts())
		row := ""
		for _, line := range body {
			if s := stripANSITest(line); strings.Contains(s, l.label+" -") {
				row = s
			}
		}
		was := d.detailOn(l.key) // the Weather preset's (D-79)
		words := map[bool]string{true: "Enabled", false: "Disabled"}
		if !strings.Contains(row, "←") || !strings.Contains(row, words[was]) || !strings.Contains(row, "→") {
			t.Fatalf("%s: the row is %q, not the picker at %s", l.key, row, words[was])
		}
		for _, key := range []tea.KeyPressMsg{{Code: tea.KeyRight}, {Code: tea.KeyLeft}} {
			m, _, _ := d.setupRowKey(key)
			if m.(Dashboard).detailOn(l.key) == was {
				t.Errorf("%s: %s left it as it was", l.key, key.String())
			}
		}
		d = d.setupSpace()
		if d.detailOn(l.key) == was {
			t.Errorf("%s: space left it as it was", l.key)
		}
		body, _, _ = d.focusBody(d.opts())
		if !strings.Contains(stripANSITest(strings.Join(body, "\n")), words[!was]) {
			t.Errorf("%s: switched, the row does not say %s", l.key, words[!was])
		}
		if m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight}); m.(Dashboard).detailOn(l.key) != was {
			t.Errorf("%s: → did not switch it back", l.key)
		}
		m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
		drain(t, m, cmd)
		if on, ok := got.MapDetail[l.key]; !ok || on == was {
			t.Errorf("%s: esc wrote %v", l.key, got.MapDetail)
		}
	}
}

// TestSettingsIsNoWiderThanFourFifths is U1-36: at every size the window is
// no wider than 80% of the terminal, or the one-column floor on a terminal
// too small for that; at 133 columns every tab that was laid in two columns
// still is, the Maps tab among them.
func TestSettingsIsNoWiderThanFourFifths(t *testing.T) {
	for _, size := range [][2]int{{133, 44}, {160, 50}, {100, 30}, {80, 24}} {
		d, _ := uiDash(t, rowMapsOn)
		m, _ := d.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		d = m.(Dashboard)
		for _, tab := range d.tabsShown() {
			on := d
			if id, ok := d.firstRowOfTab(tab); ok {
				on.setup.focus = id
			}
			if w, most := on.modalWidth(), max(setupOneColWidth, size[0]*80/100); w > most {
				t.Errorf("%dx%d tab %d: the window is %d wide, over %d", size[0], size[1], tab, w, most)
			}
		}
	}
	d, _ := uiDash(t, rowMapsOn)
	for _, tab := range []setupTab{tabData, tabRadio, tabMaps} {
		on := d
		if id, ok := d.firstRowOfTab(tab); ok {
			on.setup.focus = id
		}
		if _, two := on.columnPlan(on.setupBlocks(on.opts()), on.opts()); !two {
			t.Errorf("at 133 columns tab %d is stacked; its two columns no longer fit", tab)
		}
	}
}

// TestTheWaterSwitchIsTheLakes is UAT-1 U1-39 (go-tuiMaps D-85): the switch
// takes the lakes and inland water, and says so; the sea and its coast are
// the library's to keep, so no row offers them.
func TestTheWaterSwitchIsTheLakes(t *testing.T) {
	for _, l := range mapDetailLayers() {
		if l.key == "water" && l.label != "Lakes" {
			t.Errorf("the water switch is labelled %q; it takes the lakes, never the sea", l.label)
		}
		if strings.Contains(strings.ToLower(l.label), "ocean") || strings.Contains(strings.ToLower(l.label), "sea") {
			t.Errorf("a detail row offers the sea: %q", l.label)
		}
	}
}

// TestTheAlertCategoriesSwitchTheirAlerts is D-80 (UAT-1 U1-43): the Overlays
// menu lists [w]'s categories under the alert areas - not Disasters, not
// Forecasts - and a category switched off takes its alerts off the map, and
// only its alerts; the choice is written with the map's settings.
func TestTheAlertCategoriesSwitchTheirAlerts(t *testing.T) {
	var labels []string
	for _, c := range AlertCategories() {
		labels = append(labels, c.Label)
	}
	if got := strings.Join(labels, ", "); got != "Emergency, Warnings, Watches, Advisories, Spec. Statements, Marine" {
		t.Errorf("the categories are %s", got)
	}
	feed := func(ctx context.Context, ask MapAsk) MapFeed {
		f := MapFeed{}
		for _, o := range []tuimaps.Overlay{alertAt("w", "Wind Warning", -117.5, 33.1, tuimaps.SeveritySevere), alertAt("a", "Small Craft Advisory", -117.9, 33.0, tuimaps.SeverityMinor)} {
			cat := map[string]string{"alert/w": "warnings", "alert/a": "advisories"}[o.ID]
			o.ID = AlertLayer + "/" + cat + "/" + strings.TrimPrefix(o.ID, "alert/")
			f.Overlays = append(f.Overlays, o)
		}
		return f
	}
	var saved UIPrefs
	d := mapDash(t, Config{MapLayers: alertLayers, MapFeed: feed})
	d.cfg.SetUI = func(p UIPrefs) error { saved = p; return nil }
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	if len(d.mapPane.given) != 2 {
		t.Fatalf("both categories on, %d overlays drawn", len(d.mapPane.given))
	}
	d = pressCode(d, 'O', "O")
	var keys []string
	for _, r := range d.overlayRows() {
		keys = append(keys, r.key)
	}
	d.mapPane.menuAt = indexOf(keys, categoryChoice("advisories"))
	if d.mapPane.menuAt < 0 {
		t.Fatalf("the menu has no Advisories row: %v", keys)
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = feedAndSettle(t, m.(Dashboard))
	if _, ok := d.mapPane.given["alert/advisories/a"]; ok {
		t.Error("the advisory is still drawn with Advisories off")
	}
	if _, ok := d.mapPane.given["alert/warnings/w"]; !ok {
		t.Error("switching Advisories off took the warning too")
	}
	for _, msg := range msgsOf(cmd) {
		if _, ok := msg.(uiSavedMsg); ok && saved.MapLayers != nil && !saved.MapLayers[categoryChoice("advisories")] {
			return
		}
	}
	t.Errorf("the switch was not written: %+v", saved.MapLayers)
}
