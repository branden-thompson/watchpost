package tty

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/term"
)

// P1(c): the console declares a floor, says so below it, and NEVER renders
// past the terminal (FR-7.3). Silent overflow is a defect, not a degradation
// — F-55 measured Observer rendering 57 cells wide in a 20-cell terminal, and
// this surface must not inherit that.

func bcSized(w, h int) Broadcaster {
	b := NewBroadcaster()
	b.width, b.height = w, h
	return b
}

// widestLine returns the longest rendered line in cells.
func widestLine(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if c := utf8.RuneCountInString(line); c > n {
			n = c
		}
	}
	return n
}

func TestBelowTheFloorTheConsoleSaysSoRatherThanOverflowing(t *testing.T) {
	got := bcSized(60, 20).View().Content
	if !strings.Contains(strings.ToUpper(got), "TOO SMALL") {
		t.Error("below the floor the console must SAY so; the operator gets no notice at 60x20")
	}
	if w := widestLine(got); w > 60 {
		t.Errorf("the notice itself overflowed: %d cells in a 60-cell terminal "+
			"(the red team's own anti-solution for M5 — a notice that overflows reads as zero overflows)", w)
	}
	if h := strings.Count(got, "\n") + 1; h > 20 {
		t.Errorf("the notice is %d lines in a 20-line terminal", h)
	}
}

func TestTheFloorIsFortyFourLines(t *testing.T) {
	// 44 = fixed chrome + one readable card, measured at DISCOVER wave 1.
	if _, rows := bcSized(150, 74).minSize(); rows != 44 {
		t.Errorf("the height floor is 44 lines (wave 1, measured); got %d", rows)
	}
}

func TestAboveTheFloorTheConsoleRendersLanes(t *testing.T) {
	if got := bcSized(150, 74).View().Content; strings.Contains(strings.ToUpper(got), "TOO SMALL") {
		t.Error("at 150x74 — the mock's own size — the console must render, not refuse")
	}
}

// FR-7.1's exit condition: the layout is SELECTED BY the platform breakpoint
// function. A call whose result is discarded does not satisfy this, which is
// why the assertion is that CHANGING the class changes the frame.
func TestTheLayoutIsSelectedByThePlatformBreakpoint(t *testing.T) {
	narrow := bcSized(100, 60).View().Content
	wide := bcSized(150, 60).View().Content
	if term.BreakpointFor(100) == term.BreakpointFor(150) {
		t.Fatal("the fixture is void: both widths classify the same, so this asserts nothing")
	}
	if narrow == wide {
		t.Error("two different breakpoint classes rendered an identical frame — the layout is not " +
			"selected by the breakpoint, and FR-7.1's exit would pass on a discarded call")
	}
}

// INST-1: the sweep DERIVES its widths from the breakpoint boundaries, and is
// UNIONED with a stride so an interior width cannot pass while broken — the
// counter-argument the PLAN red team made against a boundary-only sweep.
func TestNoSizeRendersPastTheTerminal(t *testing.T) {
	seen := map[int]bool{}
	var widths []int
	for w := 20; w <= 200; w++ { // find every boundary by asking the classifier
		if c := term.BreakpointFor(w); !seen[int(c)] {
			seen[int(c)] = true
			widths = append(widths, w-1, w, w+1)
		}
	}
	for w := 20; w <= 200; w += 5 { // ... unioned with a stride
		widths = append(widths, w)
	}
	for _, w := range widths {
		if w < 1 {
			continue
		}
		for _, h := range []int{5, 20, 44, 60, 74} {
			got := bcSized(w, h).View().Content
			if c := widestLine(got); c > w {
				t.Fatalf("%dx%d rendered %d cells wide — past the terminal", w, h, c)
			}
			if l := strings.Count(got, "\n") + 1; l > h {
				t.Fatalf("%dx%d rendered %d lines — past the terminal", w, h, l)
			}
		}
	}
	t.Logf("swept %d widths x 5 heights; blind to a terminal narrower than 20 or wider than 200", len(widths))
}
