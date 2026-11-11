package tty

import (
	"fmt"
	"strings"
	"testing"
)

// TestTheFrameNeverExceedsTheTerminal is 0.19.5 D-28 and D-30 (DEF-8): the
// Observer's frame fits the terminal at every size down to 80x24, whatever
// the watchlist holds; on a short terminal the priority table scrolls and
// keeps the focused favourite in view. Under the alt screen Bubble Tea draws
// only the terminal's height, so a taller frame loses its bottom rows.
func TestTheFrameNeverExceedsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {80, 30}, {100, 30}, {133, 44}, {200, 60}} {
		for _, keep := range []int{0, 3, WatchCap} {
			d := benchDash(t, size[0], size[1]).(Dashboard)
			if keep < len(d.snap.Locations) {
				snap := *d.snap
				snap.Locations = snap.Locations[:keep]
				d.snap = &snap
			}
			for sel := range max(1, d.numPriority()) {
				d.selected = sel
				frame := stripANSITest(d.View().Content)
				if rows := strings.Count(frame, "\n") + 1; rows > size[1] {
					t.Fatalf("%dx%d with %d favourites: the frame is %d rows", size[0], size[1], keep, rows)
				}
				if keep > 0 && !strings.Contains(frame, fmt.Sprintf("%03d. %s", sel+1, d.snap.Locations[sel].Label)) {
					t.Fatalf("%dx%d with %d favourites: favourite %d has the focus and is not on screen", size[0], size[1], keep, sel+1)
				}
			}
		}
	}
}
