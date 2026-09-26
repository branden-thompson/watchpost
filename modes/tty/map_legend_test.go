package tty

// map_legend_test.go — 0.18.0 W1.17 (the legend, D-44), W1.12 (the exposure
// disclosure, FR-9.4) and W3.8 (the clear path, FR-3.9) in the window.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

func legendDash(t *testing.T, cfg Config) Dashboard {
	t.Helper()
	cfg.MapFeed = boxFeed(-117.6, -117.1, false)
	cfg.MapDescription = "off"
	d := mapDash(t, cfg)
	d, _ = pressKey(d, "g")
	return feedAndSettle(t, d)
}

// TestTheLegendOpensOverTheMap is W1.17 (D-44): L opens a legend over the
// map, contextual - the severities drawn, with their digits - and L closes it;
// the map stays drawn around it.
func TestTheLegendOpensOverTheMap(t *testing.T) {
	d := legendDash(t, Config{})
	if strings.Contains(stripANSITest(strings.Join(d.mapBodyLines(), "\n")), "┌─ Legend") {
		t.Fatal("the legend shows before it is opened")
	}
	d = pressCode(d, 'L', "L")
	body := stripANSITest(strings.Join(d.mapBodyLines(), "\n"))
	if !strings.Contains(body, "Legend") || !strings.Contains(body, "3 SEVERE") {
		t.Errorf("the legend does not key the severity drawn:\n%s", body)
	}
	if strings.Contains(body, "4 EXTREME") || strings.Contains(body, "1 MINOR") {
		t.Errorf("the legend keys severities that are not on the map:\n%s", body)
	}
	if strings.IndexFunc(body, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) < 0 {
		t.Error("the legend replaced the map instead of sitting over it")
	}
	for _, l := range d.mapBodyLines() {
		if render.Width(l) > d.mapBodySize().Cols+1 {
			t.Fatalf("a line with the legend over it is %d wide, the body %d", render.Width(l), d.mapBodySize().Cols+1)
		}
	}
	d = pressCode(d, 'L', "L")
	if strings.Contains(stripANSITest(strings.Join(d.mapBodyLines(), "\n")), "┌─ Legend") {
		t.Error("L did not close the legend")
	}
}

// TestTheChipNamesTheLegend is W1.17 (D-44): the window's chrome carries the
// legend's chip, "[ L ] Legend", naming the key as rebound.
func TestTheChipNamesTheLegend(t *testing.T) {
	d := legendDash(t, Config{})
	if s := stripANSITest(d.mapStatusLine()); !strings.Contains(s, "L") || !strings.Contains(s, "Legend") {
		t.Errorf("the status line carries no legend chip: %q", s)
	}
}

// TestSettingsSaysWhatIsSent is W1.12 (FR-9.4) as D-69 amends it: Settings
// says, beside the maps row, what opening the map sends and to whom, with the
// retention (FR-3.9). The map window says nothing of it
// (TestTheMapSaysNothingAboutWhatItSends).
func TestSettingsSaysWhatIsSent(t *testing.T) {
	const told = "Opening the map asks OpenFreeMap for the area shown."
	s, _ := uiDash(t, rowMapsOn)
	s.cfg.MapDisclosure, s.cfg.MapRetention = told, "Kept 7 days."
	body, _, _ := s.focusBody(s.opts())
	text := stripANSITest(strings.Join(body, "\n"))
	if !strings.Contains(text, "OpenFreeMap") || !strings.Contains(text, "Kept 7 days.") {
		t.Errorf("Settings does not say what the map sends and keeps:\n%s", text)
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
