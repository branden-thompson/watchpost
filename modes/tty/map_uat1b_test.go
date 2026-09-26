package tty

// map_uat1b_test.go — 0.18.0 UAT-1's second pass: no disclosure on the map
// (U1-17, D-69); the picture's window never scrolls, so the status and the
// chips are always in it (U1-18, U1-20); the boxes leave the library's
// furniture rows alone - the credit (U1-26) and the top row's time.

import (
	"strings"
	"testing"
)

// TestTheMapSaysNothingAboutWhatItSends is D-69: the first open, box open or
// closed, carries no disclosure; Settings still says it.
func TestTheMapSaysNothingAboutWhatItSends(t *testing.T) {
	const told = "Opening the map asks OpenFreeMap for the area shown."
	for _, mode := range []string{"with", "off", "instead"} {
		d := openMap(t, Config{MapDisclosure: told, MapDescription: mode}, 133, 44)
		if strings.Contains(bodyText(d), "Opening the map") {
			t.Errorf("description %s: the map says what it sends", mode)
		}
	}
	s, _ := uiDash(t, rowMapsOn)
	s.cfg.MapDisclosure = told
	body, _, _ := s.focusBody(s.opts())
	if !strings.Contains(stripANSITest(strings.Join(body, "\n")), "OpenFreeMap") {
		t.Error("Settings no longer says what the map sends")
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
