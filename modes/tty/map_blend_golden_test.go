package tty

// map_blend_golden_test.go — 0.18.0 W8.11 (D-263): radar over an alert area,
// the scene of go-tuiMaps' specimen 29, at each colour depth. The library's
// own goldens hold its blend (WP-L3); these hold what watchpost hands it and
// what comes back: every outline, label, severity digit and credit drawn at
// every depth, never erased by the radar under it. watchpost hints no depth
// in 0.18.0 (FR-7.1, cut by D-247): truecolor, or none under NO_COLOR, is
// what a listener sees; 256 and 16 are held for the hint to come.

import (
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

func TestRadarOverAnAlertAtEachDepth(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	size := d.mapBodySize()
	at := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	depths := []struct {
		name  string
		depth tuimaps.Depth
	}{{"truecolor", tuimaps.Truecolor}, {"256", tuimaps.Colours256}, {"16", tuimaps.Colours16}, {"none", tuimaps.NoColour}}
	var furniture []string // truecolor's characters: radar is colour alone there, so this is everything else
	for _, dp := range depths {
		d.mapPane.m.ColourDepth(dp.depth)
		f, err := d.mapPane.m.Render(size, at)
		if err != nil {
			t.Fatal(err)
		}
		frame := strings.Join(f.Lines, "\n")
		plain := strings.Split(stripANSITest(frame), "\n")
		text := strings.Join(plain, "\n")
		for _, w := range []string{"Wind Warning · SEVERE", "OpenFreeMap", "100 km"} {
			if !strings.Contains(text, w) {
				t.Errorf("%s: %q is not drawn", dp.name, w)
			}
		}
		if dp.depth == tuimaps.Truecolor {
			if !strings.Contains(frame, "\x1b[48;2;") {
				t.Fatal("truecolor: no radar under the map: this pins nothing")
			}
			furniture = plain
		}
		if n := strings.Count(text, "3"); n < 8 {
			t.Errorf("%s: %d severity digits on the outline; want them all", dp.name, n)
		}
		for row, line := range furniture {
			want, got := []rune(line), []rune(plain[row])
			for col, r := range want {
				if r != '⠀' && r != ' ' && (col >= len(got) || got[col] != r) {
					t.Errorf("%s: row %d col %d is %q where truecolor draws %q: the radar erased the map's furniture", dp.name, row, col, string(got[col]), string(r))
				}
			}
		}
		checkGolden(t, "map-radar-over-alert-"+dp.name+".golden", frame)
	}
}
