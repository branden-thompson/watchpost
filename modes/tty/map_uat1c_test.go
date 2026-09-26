package tty

// map_uat1c_test.go — 0.18.0 UAT-1's fourth pass, the Settings window: air
// between the tabs and the page (U1-30), the map's detail as a row per layer
// in the "← Enabled →" pattern (U1-35), and a window no wider than 80% of the
// terminal with its two columns still side by side (U1-36).

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
		if !strings.Contains(row, "←") || !strings.Contains(row, "Enabled") || !strings.Contains(row, "→") {
			t.Fatalf("%s: the row is %q, not the picker", l.key, row)
		}
		for _, key := range []tea.KeyPressMsg{{Code: tea.KeyRight}, {Code: tea.KeyLeft}} {
			m, _, _ := d.setupRowKey(key)
			if m.(Dashboard).detailOn(l.key) {
				t.Errorf("%s: %s left it on", l.key, key.String())
			}
		}
		d = d.setupSpace()
		if d.detailOn(l.key) {
			t.Errorf("%s: space left it on", l.key)
		}
		body, _, _ = d.focusBody(d.opts())
		if !strings.Contains(stripANSITest(strings.Join(body, "\n")), "Disabled") {
			t.Errorf("%s: switched off, the row does not say Disabled", l.key)
		}
		if m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight}); !m.(Dashboard).detailOn(l.key) {
			t.Errorf("%s: → did not switch it back on", l.key)
		}
		m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
		drain(t, m, cmd)
		if on, ok := got.MapDetail[l.key]; !ok || on {
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
