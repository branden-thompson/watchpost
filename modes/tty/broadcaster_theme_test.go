package tty

// broadcaster_theme_test.go — the console reads the THEME's palette, not the
// terminal's (D-108).
//
// HUM LEAD, UAT 2026-09-12: "Some of the Broadcaster UIs are not using themeable
// token values for colors - particularly in the 'UP NEXT' section - Watchpost
// Light has white lines and text when it should be inverted appropriately."

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// EVERY ROW OPENS ON A COLOUR THE THEME CHOSE.
//
// NOTHING WAS PAINTING THEM WHITE — they were painted by NOBODY, and took the
// TERMINAL's default foreground. On a dark terminal that is white, which looks
// right under the dark themes and made the whole surface read the terminal's
// palette while calling it the theme's. The UP NEXT box is where it showed,
// because its border is the largest run of untinted glyphs on the frame.
func TestTheConsoleArmsTheThemesForeground(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	// UNDER THE LIGHT THEME, which is the theme the finding was reported against
	// AND the one whose `TextBase` is TRUECOLOR. Asked of a palette-index theme
	// this test cannot see the difference between arming the token properly and
	// arming it through a hard-coded `38;5;` prefix — which is exactly what a
	// mutant proved by making that substitution and surviving.
	was := render.ThemeName()
	if !render.SetTheme(render.LightThemeName) {
		t.Fatalf("the Light theme is not registered")
	}
	defer render.SetTheme(was)
	if !strings.HasPrefix(render.Tok(render.TextBase), "38;2;") {
		t.Fatalf("the Light theme's TextBase is %q, not truecolor; this test measures nothing",
			render.Tok(render.TextBase))
	}

	b := manyPool(t, 25)
	b.width, b.height = 150, 58
	got := b.View().Content
	want := "\x1b[0;" + render.FgSGR(render.Tok(render.TextBase)) + "m"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("the frame does not open on the theme's text colour:\n%q", got[:min(80, len(got))])
	}
	// AND NO RESET HANDS THE FRAME BACK TO THE TERMINAL. `TintKeeping`'s whole
	// contract is that every inner reset falls back to THESE parameters, so the
	// only bare reset in the frame is the one that closes it — and a chip, a card
	// ground or a table cell can style itself without leaving the rest of its row
	// in whatever colour the terminal felt like.
	if n := strings.Count(got, "\x1b[0m"); n != 1 {
		t.Errorf("the frame carries %d bare resets; each one drops a row back to the terminal", n)
	}
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Error("and the frame closes on a reset, so the tint does not leak past it")
	}
	// THE UP NEXT BOX IS THE CASE THAT WAS REPORTED: its border is the largest run
	// of glyphs on the frame that nothing else tints, so before this it was drawn
	// in the terminal's default and vanished on the Light theme.
	if !strings.Contains(got, render.HeavyBox(false).T) {
		t.Error("the frame draws no UP NEXT box, so this proves nothing")
	}
}
