package tty

// map_legend_test.go — 0.18.0 W1.17 (the legend, D-44), W1.12 (the exposure
// disclosure, FR-9.4) and W3.8 (the clear path, FR-3.9) in the window.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func legendDash(t *testing.T, cfg Config) Dashboard {
	t.Helper()
	cfg.MapFeed = boxFeed(-117.6, -117.1, false)
	cfg.MapDescription = "off"
	d := mapDash(t, cfg)
	d, _ = pressKey(d, "g")
	return feedAndSettle(t, d)
}

// TestTheLegendBoxIsRetired is D-103: the map says everything the legend
// box did - the colour rows under it, the severity digits on the outlines -
// so the box is gone and L is free: no binding, no chip, no Help row.
func TestTheLegendBoxIsRetired(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	if _, bound := d.mapKeys.Lookup("L"); bound {
		t.Error("L is still bound in the map window")
	}
	if strings.Contains(stripANSITest(d.mapStatusLine()), "Legend") {
		t.Error("the chips still offer the legend")
	}
	for _, r := range mapHelpRows(d.mapKeys, false) {
		if strings.Contains(r.help, "Legend") {
			t.Errorf("Help still lists %q", r.help)
		}
	}
}

// TestSettingsStatesTheRetention is W3.8 (FR-3.9): beside Clear map data,
// Settings says how long the map's data is kept. What the map sends is the
// Status window's (D-75, TestWhatTheMapContactsIsInTheStatusWindow).
func TestSettingsStatesTheRetention(t *testing.T) {
	s, _ := uiDash(t, rowMapClear)
	s.cfg.MapRetention = "Kept 7 days."
	body, _, _ := s.focusBody(s.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Kept 7 days.") {
		t.Errorf("Settings does not say how long the map keeps its data:\n%s", text)
	}
}

// TestClearMapDataEmptiesTheLiveMapAndAsksTheApp is W3.8 with W9.5 folded:
// the Settings action purges the window's live map through the library and
// asks the app for everything else, then says what went.
func TestClearMapDataEmptiesTheLiveMapAndAsksTheApp(t *testing.T) {
	asked := 0
	d := legendDash(t, Config{ClearMapData: func() MapCleared { asked++; return MapCleared{Files: 7, Zones: 3} }})
	calls := &[]string{}
	d.mapPane.calls = calls
	d, _ = pressKey(d, "esc")
	d = d.openSetupAt(rowMapClear)
	m, cmd, handled := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !handled || cmd == nil {
		t.Fatal("space on Clear map data did nothing")
	}
	d = m.(Dashboard)
	if !strings.Contains(strings.Join(*calls, " "), "Purge") {
		t.Errorf("the live map was not purged: %v", *calls)
	}
	m, _ = d.Update(cmd())
	d = m.(Dashboard)
	if asked != 1 {
		t.Errorf("the app was asked %d times", asked)
	}
	if !strings.Contains(d.setup.note, "7") || !strings.Contains(d.setup.note, "cleared") {
		t.Errorf("Settings says %q after clearing", d.setup.note)
	}
}
