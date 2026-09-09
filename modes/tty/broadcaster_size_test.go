package tty

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
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

// FR-7 / RT-23: the console must survive --ascii. This was flagged at the
// DISCOVER red team as "no --ascii rendering exists, so the design is not
// shown to survive that mode", carried through PLAN, and is closed here.
//
// THE SUBJECT LIST IS DERIVED (INST-1): it asks the glyph set what the
// non-ASCII forms ARE, rather than naming a few by hand. A hand-written list
// is stale the day someone adds a glyph.
func TestTheConsoleCarriesNoNonASCIIUnderASCII(t *testing.T) {
	// SWEEP EVERY POWER STATE. The first version used ONE fixture, which left
	// the station STOPPED — and the ON AIR banner's separator slipped past it.
	//
	// DERIVED, not listed (INST-1): Power.String() returns "" for a value
	// outside the declared set, so the walk asks the type where it ends
	// rather than naming three constants that go stale when a fourth lands.
	for p := lineup.Power(0); p.String() != ""; p++ {
		b := bcWith(t, card(t, "a", "OCEANSIDE"))
		b.width, b.height = 150, 74
		b.ascii = true
		b, _ = b.Update(StationMsg{Power: p})
		assertASCIIOnly(t, b.View().Content, p)
	}
}

func assertASCIIOnly(t *testing.T, got string, p lineup.Power) {
	t.Helper()
	rich := render.Opts{}.Glyphs() // the non-ASCII set, asked for rather than listed
	for _, g := range []string{
		rich.Bullet, rich.Pointer, rich.Play, rich.Pause, rich.Alert, rich.OK, rich.Fail,
		rich.Rail, rich.RailCar, rich.Fill, rich.Up, rich.Down, rich.Stop, rich.Rule, rich.Dot,
	} {
		if g != "" && strings.Contains(got, g) {
			t.Errorf("--ascii (power=%v): the frame carries %q, which has an ASCII form the glyph set already defines", p, g)
		}
	}
	for _, r := range got {
		if r > 127 && r != '\n' {
			t.Errorf("--ascii (power=%v): the frame carries the non-ASCII rune %q", p, r)
			break
		}
	}
}
