package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// clampShipped is Broadcaster.clamp as 0.19.3 shipped it: the reference the
// one-pass finisher is held to, byte for byte.
func clampShipped(b Broadcaster, lines []string) string {
	if b.height > 0 && len(lines) > b.height {
		lines = lines[:b.height]
	}
	if b.height > 0 {
		for range max(0, b.height-len(lines)) {
			lines = append(lines, "")
		}
	}
	if b.height > 0 && len(lines) > b.height {
		lines = lines[:b.height]
	}
	if b.width > 0 {
		for i, l := range lines {
			lines[i] = render.PadTo(bcLeftInset+render.TruncateCells(l, b.frameWidth()), b.width)
		}
	}
	out := strings.Join(lines, "\n")
	if render.ColorOn() {
		out = render.TintKeeping(out, render.FgSGR(render.Tok(render.TextBase)))
	}
	return out
}

// TestTheConsolesFinisherIsByteIdentical is 0.19.5 Q3.4 (PF-4, RR-8): the
// console's frame is finished in one pass, truncated, inset, padded and armed
// with the theme's foreground, and comes out the same bytes as the three-copy
// finisher it replaces: at four sizes, colour off and on, the Light theme, a
// frame taller than the terminal, a row wider than the frame, and width and
// height zero.
func TestTheConsolesFinisherIsByteIdentical(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Cleanup(rendering.ResetColorEnabledForTest)
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	c := loadedConsole(t, loadedPoolSize)
	wide := strings.Repeat("\x1b[38;5;214mWIDE\x1b[0m ", 80)
	for _, theme := range []string{render.DefaultThemeName, "Watchpost Light"} {
		if !render.SetTheme(theme) {
			t.Fatalf("no theme %q", theme)
		}
		for _, colour := range []bool{false, true} {
			rendering.SetColorEnabledForTest(colour)
			for _, sz := range [][2]int{{100, 44}, {133, 44}, {160, 74}, {220, 60}, {0, 44}, {133, 0}} {
				c.width, c.height = sz[0], sz[1]
				frame := c.lanes()
				for _, lines := range [][]string{frame, append(append([]string{wide}, frame...), frame...)} {
					got := c.clamp(append([]string(nil), lines...))
					if want := clampShipped(c, append([]string(nil), lines...)); got != want {
						line, w, g := firstDiffLine(want, got)
						t.Fatalf("%s, colour %v, %dx%d: the frame differs at line %d:\n want %q\n  got %q", theme, colour, sz[0], sz[1], line, w, g)
					}
				}
			}
		}
	}
}
