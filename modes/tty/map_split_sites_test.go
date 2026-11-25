package tty

// map_split_sites_test.go — the map's split sites (0.19.6 design MS-1; A-31):
// where map code runs a root operation and then reads what it changed. Each
// holds the order the re-homing must keep when the site is cut in two, and
// was seen failing with the two halves swapped.

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// drawnAfter are the sizes the map was drawn at by what ran after the views
// held n draws, and fails if nothing drew it. Every one is held, not the
// last: a draw at the wrong size is redrawn by the update's own steps, so the
// frame left shows nothing of it.
func drawnAfter(t *testing.T, views *[]mapView, n int) []tuimaps.Size {
	t.Helper()
	if len(*views) <= n {
		t.Fatalf("COULD NOT RUN: the map was not drawn (%d draws, %d before)", len(*views), n)
	}
	var out []tuimaps.Size
	for _, v := range (*views)[n:] { // bounded by the draws (P10-02)
		out = append(out, v.size)
	}
	return out
}

// allAt reports whether every size is want.
func allAt(sizes []tuimaps.Size, want tuimaps.Size) bool {
	for _, s := range sizes { // bounded by the draws (P10-02)
		if s != want {
			return false
		}
	}
	return true
}

// TestTheFrequencyEntryClosesBeforeTheMapIsDrawn is split site S2: Enter in
// the frequency entry closes it and then draws the map, so every frame drawn
// is the map window's size, none the entry's (0.19.0 D-178), with a frequency
// typed and with the empty Enter that clears one.
func TestTheFrequencyEntryClosesBeforeTheMapIsDrawn(t *testing.T) {
	for _, typed := range []string{"14.2", ""} { // bounded by the two Enters (P10-02)
		d := chartDash(t)
		views := &[]mapView{}
		d.mapModel.mapPane.views = views
		d = pressMap(t, d, "F")
		entry := d.mapModel.mapBodySize(d.mapEnv())
		for _, r := range typed { // bounded by what is typed (P10-02)
			m, _ := d.Update(mapKeyMsg(t, string(r)))
			d = m.(Dashboard)
		}
		n := len(*views)
		m, _ := d.Update(mapKeyMsg(t, "enter"))
		d = m.(Dashboard)
		if d.modal != modalMap {
			t.Fatalf("Enter on %q left window %v; want the map", typed, d.modal)
		}
		// THE PREMISE: the entry's body is not the map's, so a draw before the close is seen.
		if entry == d.mapModel.mapBodySize(d.mapEnv()) {
			t.Fatalf("COULD NOT RUN: the entry's body %v is the map's", entry)
		}
		if got := drawnAfter(t, views, n); !allAt(got, d.mapModel.mapBodySize(d.mapEnv())) {
			t.Errorf("Enter on %q drew the map at %v; want every draw at the map window's %v, none at the entry's %v", typed, got, d.mapModel.mapBodySize(d.mapEnv()), entry)
		}
	}
}

// TestAFoundPlaceIsDrawnOnceTheFinderHasClosed is split site S3: a place
// found closes the finder and then recentres, settles and draws, so every
// frame drawn is the map window's size, none the finder's (0.19.0 FR-1.10).
func TestAFoundPlaceIsDrawnOnceTheFinderHasClosed(t *testing.T) {
	d := chartDash(t)
	ownHooks(&d).FindPlace = func(context.Context, string) (snapshot.LocationRef, error) { return reykjavik, nil }
	views := &[]mapView{}
	d.mapModel.mapPane.views = views
	finder := mapModel.mapBodySize(envOf(pressMap(t, d, "/")))
	n := len(*views)
	d = typeFind(t, d, "Reykjavik")
	if d.modal != modalMap {
		t.Fatalf("after a place was found the window is %v; want the map", d.modal)
	}
	// THE PREMISE: the finder's body is not the map's, so a draw before the close is seen.
	if finder == d.mapModel.mapBodySize(d.mapEnv()) {
		t.Fatalf("COULD NOT RUN: the finder's body %v is the map's", finder)
	}
	if got := drawnAfter(t, views, n); !allAt(got, d.mapModel.mapBodySize(d.mapEnv())) {
		t.Errorf("a found place drew the map at %v; want every draw at the map window's %v, none at the finder's %v", got, d.mapModel.mapBodySize(d.mapEnv()), finder)
	}
}

// TestTheAcknowledgementAndTheChartScrollAtTheFloor is split site S6: in the
// acknowledgement the Observer's scroll keys, and in the chart the page keys,
// are handed to the window's own scroll, which moves the body (0.19.0 FR-11.2,
// D-256). At the floor, where each window's body runs past its rows.
func TestTheAcknowledgementAndTheChartScrollAtTheFloor(t *testing.T) {
	ack := ackPress(t, openMap(t, Config{Map: &Hooks{SavePropagationAck: func(int) error { return nil }}}, 80, 24), "P")
	m, _ := chartDash(t).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	chart := pressMap(t, m.(Dashboard), "B")
	for _, c := range []struct {
		name, down, up string
		d              Dashboard
		window         modal
	}{{"the acknowledgement", "down", "up", ack, modalPropAck}, {"the chart", "pgdown", "pgup", chart, modalPropChart}} { // bounded by the two windows (P10-02)
		if c.d.modal != c.window {
			t.Fatalf("%s is not open: window %v", c.name, c.d.modal)
		}
		// THE PREMISE: the body runs past the window's rows, so a scroll has somewhere to go.
		if len(c.d.modalLines()) <= c.d.modalMax() {
			t.Fatalf("COULD NOT RUN: %s's %d lines fit its %d rows", c.name, len(c.d.modalLines()), c.d.modalMax())
		}
		m, _ := c.d.Update(mapKeyMsg(t, c.down))
		on := m.(Dashboard)
		if on.modal != c.window || on.modalScroll <= c.d.modalScroll {
			t.Errorf("%s in %s: window %v, scrolled to %d from %d; want it open and scrolled on", c.down, c.name, on.modal, on.modalScroll, c.d.modalScroll)
		}
		m, _ = on.Update(mapKeyMsg(t, c.up))
		if back := m.(Dashboard); back.modal != c.window || back.modalScroll >= on.modalScroll {
			t.Errorf("%s in %s: window %v, scrolled to %d from %d; want it open and scrolled back", c.up, c.name, back.modal, back.modalScroll, on.modalScroll)
		}
	}
}

// TestTheMenusScrollStopsAtTheTextsEnd is split site S7: the map scrolls the
// window to keep the Overlays menu's focus in view without a picture, and
// never past its text's last row, the lines measured as the window draws them
// (0.18.0 D-267). The focus on the menu's last row, the scroll at it.
func TestTheMenusScrollStopsAtTheTextsEnd(t *testing.T) {
	m, _ := screenDash(t, "with", true).Update(tea.WindowSizeMsg{Width: 133, Height: 30})
	d := m.(Dashboard)
	for _, k := range []string{"O", "up"} { // up from the first row is the last (0.19.0 D-199)
		m, _ := d.Update(mapKeyMsg(t, k))
		d = m.(Dashboard)
	}
	lines, rows := d.modalLines(), d.modalMax()
	at := len(lines) - 1
	for at >= 0 && !strings.Contains(ttytest.StripANSI(lines[at]), d.opts().Glyphs().Pointer+" ") { // bounded by the lines (P10-02)
		at--
	}
	// THE PREMISE: the focused row past the last window's top, so a scroll set at it runs past the end.
	if at <= len(lines)-rows {
		t.Fatalf("COULD NOT RUN: the focus at line %d of %d, %d rows: a scroll at it is in bounds", at, len(lines), rows)
	}
	d.modalScroll = at
	if got := d.withMap(d.mapModel.followMenuFocus(d.mapEnv())).modalScroll; got != len(lines)-rows {
		t.Errorf("the menu scrolled to %d; want %d, its text's last window (%d lines, %d rows)", got, len(lines)-rows, len(lines), rows)
	}
}

// TestPgDnStopsAtTheWindowsLastPage is split site S7's page keys: PgDn scrolls
// the map window's body a page at a time and stops at its last page, the
// lines counted as the window draws them (0.18.0 D-61). The Propagation
// mode's words in the picture's place, longer than the window.
func TestPgDnStopsAtTheWindowsLastPage(t *testing.T) {
	d := chartDash(t)
	d.mapModel.mapDesc = mapDescInstead
	m, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	d = m.(Dashboard)
	lines, rows := len(d.modalLines()), d.modalMax()
	// THE PREMISE: more than a page and a row, so a page key at the end has a page to overshoot.
	if lines <= rows+1 {
		t.Fatalf("COULD NOT RUN: %d lines in %d rows", lines, rows)
	}
	for range lines/max(rows-1, 1) + 2 { // more presses than pages (P10-02)
		m, _ = d.Update(mapKeyMsg(t, "pgdown"))
		d = m.(Dashboard)
	}
	if d.modalScroll != lines-rows {
		t.Errorf("PgDn scrolled to %d; want %d, the last page (%d lines, %d rows)", d.modalScroll, lines-rows, lines, rows)
	}
}

// TestPgDnKeepsTheMenusFocusInView is split site S7 twice in one call: PgDn
// with the Overlays menu open and no picture scrolls the window a page, and
// the menu's focus, read after that page, brings its row back on screen
// (0.18.0 D-61, D-267). The focus near the top, so the page runs past it.
func TestPgDnKeepsTheMenusFocusInView(t *testing.T) {
	m, _ := screenDash(t, "with", true).Update(tea.WindowSizeMsg{Width: 133, Height: 30})
	d := m.(Dashboard)
	m, _ = d.Update(mapKeyMsg(t, "O"))
	d = m.(Dashboard)
	lines, rows := d.modalLines(), d.modalMax()
	at := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(ttytest.StripANSI(l), d.opts().Glyphs().Pointer+" ") })
	page := min(max(rows-1, 1), max(len(lines)-rows, 0))
	// THE PREMISE: the page lands past the focus, so the focus has to pull it back.
	if at < 0 || d.modalScroll != 0 || page <= at {
		t.Fatalf("COULD NOT RUN: the focus at line %d, the scroll at %d, a page %d (%d lines, %d rows)", at, d.modalScroll, page, len(lines), rows)
	}
	m, _ = d.Update(mapKeyMsg(t, "pgdown"))
	if got := m.(Dashboard).modalScroll; got != at {
		t.Errorf("PgDn with the menu open scrolled to %d; want %d, the menu's focus at the window's top", got, at)
	}
}

// TestAPlaceKeptNeverScrollsPastTheWords is split site S7's place keeping: a
// refresh puts the sentence the reader was at back at the window's top, and
// never scrolls past the words' last page, the lines counted as the window
// draws them (0.19.0 D-211). The reader at the words' last sentence.
func TestAPlaceKeptNeverScrollsPastTheWords(t *testing.T) {
	d := chartDash(t)
	d.mapModel.mapDesc = mapDescInstead
	m, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	d = m.(Dashboard)
	d.modalScroll = len(d.modalLines()) - 1
	d.mapModel.mapPane.wordsAt, _ = d.mapModel.wordsTop(d.mapEnv())
	// THE PREMISE: a place to keep, past the last page's top, so putting it at the window's top overshoots.
	if len(d.mapModel.mapPane.wordsAt.keys) == 0 || d.modalScroll <= len(d.modalLines())-d.modalMax() {
		t.Fatalf("COULD NOT RUN: the place %v at row %d of %d lines, %d rows", d.mapModel.mapPane.wordsAt.keys, d.modalScroll, len(d.modalLines()), d.modalMax())
	}
	d = workChart(t, refreshed(t, d))
	if end := max(len(d.modalLines())-d.modalMax(), 0); d.modalScroll > end {
		t.Errorf("after a refresh the scroll is %d, past the words' last page at %d (%d lines, %d rows)", d.modalScroll, end, len(d.modalLines()), d.modalMax())
	}
}

// TestAWeatherTintChosenInThePropagationModeIsSavedLast is split site S9: a
// weather tint chosen from the Overlays menu in the Propagation mode leaves
// the mode with the tint drawn, and saves the UI while the choice is unsaved,
// the save last in its batch, after the mode's own commands (0.19.0 D-158).
func TestAWeatherTintChosenInThePropagationModeIsSavedLast(t *testing.T) {
	var saved []UIPrefs
	var asks []MapAsk
	d := openTempMap(t, false, &asks) // Forecast mode, temperature on, UV off
	ownHooks(&d).PropagationUpdate = (&seam{answer: PropagationResult{Err: errors.New("offline")}}).update
	d.mapModel.ackSeen = propAckVersion
	d.cfg.SetUI = func(p UIPrefs) error { saved = append(saved, p); return nil }
	tint := UVLayer
	d = menuOn(t, pressMap(t, d, "P"), tint)
	m, cmd, ok := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !ok || m.mapModel.mapMode(m.mapEnv()) == modePropagation {
		t.Fatalf("space on %s: handled %v, mode %v; want the weather mode", tint, ok, m.mapModel.mapMode(m.mapEnv()))
	}
	msgs := msgsOf(t, cmd)
	// THE PREMISE: the mode's commands beside the save, so where the save stands is seen.
	if len(msgs) < 2 {
		t.Fatalf("COULD NOT RUN: %d messages; want the mode's and the save", len(msgs))
	}
	if _, ok := msgs[len(msgs)-1].(uiSavedMsg); !ok {
		t.Errorf("the batch's last message is %T; want the save, after the mode's commands", msgs[len(msgs)-1])
	}
	if len(saved) != 1 || !saved[0].MapLayers[tint] {
		t.Errorf("saved %d times, %s %v in the last; want once, with %s on", len(saved), tint, saved, tint)
	}
}

// savedFirst checks a batch's messages: the save first, ahead of the others,
// and written once with the layer as want.
func savedFirst(t *testing.T, what string, msgs []tea.Msg, saved []UIPrefs, layer string, want bool) {
	t.Helper()
	// THE PREMISE: other commands beside the save, so where the save stands is seen.
	if len(msgs) < 2 {
		t.Fatalf("COULD NOT RUN: %s gave %d messages; want the save and the rest", what, len(msgs))
	}
	if _, ok := msgs[0].(uiSavedMsg); !ok {
		t.Errorf("%s: the batch's first message is %T; want the save, ahead of the rest", what, msgs[0])
	}
	if len(saved) != 1 || saved[0].MapLayers[layer] != want {
		t.Errorf("%s: saved %d times (%v); want once, with %s %v", what, len(saved), saved, layer, want)
	}
}

// TestAMenuSwitchIsSavedFirst is split site S8: a switch made in the Overlays
// menu draws the map and then saves the UI while the switch is unsaved, the
// save first in its batch (0.18.0 D-65).
func TestAMenuSwitchIsSavedFirst(t *testing.T) {
	var saved []UIPrefs
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	d.cfg.SetUI = func(p UIPrefs) error { saved = append(saved, p); return nil }
	d = menuOn(t, d, WindLayer)
	_, cmd, ok := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !ok {
		t.Fatal("space in the menu was not handled")
	}
	savedFirst(t, "the wind switched on", msgsOf(t, cmd), saved, WindLayer, true)
}

// TestTheModeSwitchIsSavedFirst is split site S10: R saves the radar's switch
// while it is unsaved, the save first in its batch, from Forecast mode and
// from the Propagation mode (0.18.0 D-94; 0.19.0 D-153).
func TestTheModeSwitchIsSavedFirst(t *testing.T) {
	var saved []UIPrefs
	var asks []MapAsk
	d := openTempMap(t, false, &asks) // Forecast mode
	ownHooks(&d).PropagationUpdate = (&seam{answer: PropagationResult{Err: errors.New("offline")}}).update
	d.mapModel.ackSeen = propAckVersion
	d.cfg.SetUI = func(p UIPrefs) error { saved = append(saved, p); return nil }
	mm, cmd, w := d.mapModel.switchMode(d.mapEnv())
	_, cmd = withModel(d, mm).applyMapWant(w, cmd)
	savedFirst(t, "R from Forecast mode", msgsOf(t, cmd), saved, RadarLayer, true)
	saved = nil
	inProp := pressMap(t, d, "P")
	mm, cmd, w = inProp.mapModel.switchMode(inProp.mapEnv())
	_, cmd = withModel(inProp, mm).applyMapWant(w, cmd)
	savedFirst(t, "R from the Propagation mode", msgsOf(t, cmd), saved, RadarLayer, true)
}
