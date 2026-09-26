package tty

// map_uat1b_test.go — 0.18.0 UAT-1's second pass: no disclosure on the map
// (U1-17, D-69); the picture's window never scrolls, so the status and the
// chips are always in it (U1-18, U1-20); the boxes leave the library's
// furniture rows alone - the credit (U1-26) and the top row's time.

import (
	"strings"
	"testing"
)

// TestWhatTheMapContactsIsInTheStatusWindow is D-69 and D-75: neither the
// map nor Settings carries words about what the map sends; the Status window
// lists each source the map contacts, its host and what it is sent.
func TestWhatTheMapContactsIsInTheStatusWindow(t *testing.T) {
	sources := []MapSource{{Name: "OpenFreeMap", Host: "tiles.openfreemap.org", Use: "the map's tiles, for the area shown"},
		{Name: "National Weather Service", Host: "api.weather.gov", Use: "alert zone outlines; alerts of the states and marine areas in view"}}
	for _, mode := range []string{"with", "off", "instead"} {
		d := openMap(t, Config{MapSources: sources, MapDescription: mode}, 133, 44)
		if strings.Contains(bodyText(d), "tiles.openfreemap.org") {
			t.Errorf("description %s: the map says what it sends", mode)
		}
	}
	s, _ := uiDash(t, rowMapsOn)
	s.cfg.MapSources = sources
	body, _, _ := s.focusBody(s.opts())
	if strings.Contains(stripANSITest(strings.Join(body, "\n")), "tiles.openfreemap.org") {
		t.Error("Settings still carries what the map sends (U1-33)")
	}
	s.cfg.MapSources = sources
	status := stripANSITest(strings.Join(s.statusLines(), "\n"))
	for _, want := range []string{"MAP", "OpenFreeMap", "tiles.openfreemap.org", "api.weather.gov", "states and marine areas in view"} {
		if !strings.Contains(status, want) {
			t.Errorf("the Status window does not name %q:\n%s", want, status)
		}
	}
}

// TestThePicturesWindowNeverScrolls is U1-18 and U1-20: with the picture,
// box open or closed, at every size the floor allows, the body is exactly the
// window - so there is nothing to scroll and no rail - and its last line is
// the chips.
func TestThePicturesWindowNeverScrolls(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {133, 44}, {200, 60}} {
		for _, open := range []bool{true, false} {
			d := openMap(t, Config{MapLayers: alertLayers}, size.w, size.h)
			if !open {
				d = pressCode(d, 'A', "A")
			}
			if !d.mapFits() {
				t.Fatalf("%dx%d: under the floor", size.w, size.h)
			}
			lines := d.modalLines()
			if len(lines) > d.modalMax() {
				t.Errorf("%dx%d box %v: %d lines in a %d-line window - it scrolls", size.w, size.h, open, len(lines), d.modalMax())
			}
			if last := stripANSITest(lines[len(lines)-1]); !strings.Contains(last, "Legend") {
				t.Errorf("%dx%d box %v: the last line is %q, not the chips", size.w, size.h, open, last)
			}
		}
	}
}

// TestTheBoxesLeaveTheFurnitureRowsAlone is U1-26: the controls stand above
// the credit row, and the legend below the top row, where the library writes
// its time.
func TestTheBoxesLeaveTheFurnitureRowsAlone(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	size := d.mapBodySize()
	lines := d.withControls(d.mapPane.lines, size)
	if last := stripANSITest(lines[size.Rows-1]); strings.Contains(last, "Controls") || strings.Contains(last, "┘") && !strings.Contains(stripANSITest(d.mapPane.lines[size.Rows-1]), "┘") {
		t.Errorf("the controls cover the credit row: %q", last)
	}
	if got, want := stripANSITest(lines[size.Rows-1]), stripANSITest(d.mapPane.lines[size.Rows-1]); got != want {
		t.Errorf("the credit row changed under the controls:\n got %q\nwant %q", got, want)
	}
	d = pressCode(d, 'L', "L")
	legend := d.withLegend(d.mapPane.lines)
	if stripANSITest(legend[0]) != stripANSITest(d.mapPane.lines[0]) {
		t.Errorf("the legend covers the top row: %q", stripANSITest(legend[0]))
	}
}

// TestTheMapRunsBorderToBorder is U1-27: the map window has no inset - the
// map's cells begin at the left border and end at the right one, and the map
// is as wide as the window less its borders.
func TestTheMapRunsBorderToBorder(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	if got, want := d.mapBodySize().Cols, d.modalWidth()-2; got != want {
		t.Errorf("the map is %d wide in a window of %d; want %d", got, d.modalWidth(), want)
	}
	braille := func(r rune) bool { return r >= 0x2800 && r <= 0x28ff }
	for _, row := range strings.Split(stripANSITest(d.renderModal(d.opts())), "\n") {
		cells := []rune(strings.TrimSpace(row))
		// A row of picture at both borders (the top row ends in the library's
		// own furniture, the stale word and the time, which is fine).
		if len(cells) > 3 && cells[0] == '│' && braille(cells[1]) && braille(cells[len(cells)-2]) {
			return
		}
	}
	t.Fatal("no map row runs from the left border to the right one")
}
