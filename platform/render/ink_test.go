package render

import (
	"image/color"
	"testing"
)

// TestInkAtReadsTheCellsForeground is D-366: the colour a line draws a cell
// in, from its SGR sequences - basic, 256 and truecolor, bold beside it, a
// background passed over, cleared by a reset or 39, a wide glyph's second
// cell its own - and none where the terminal's own colour stands.
func TestInkAtReadsTheCellsForeground(t *testing.T) {
	yellow := color.RGBA{R: 255, G: 215, B: 0, A: 0xff} // xterm 220
	for _, c := range []struct {
		name  string
		line  string
		cell  int
		ink   color.RGBA
		inked bool
	}{
		{"a 256 pointer", "  \x1b[1;38;5;220m>\x1b[0m Theme", 2, yellow, true},
		{"past its reset", "  \x1b[1;38;5;220m>\x1b[0m Theme", 4, color.RGBA{}, false},
		{"a bare reset", "\x1b[38;5;220m>\x1b[m x", 2, color.RGBA{}, false},
		{"39 clears it", "\x1b[38;5;220m>\x1b[39m x", 2, color.RGBA{}, false},
		{"truecolor", "\x1b[38;2;10;20;30mab", 1, color.RGBA{R: 10, G: 20, B: 30, A: 0xff}, true},
		{"a bright basic", "\x1b[1;97m>", 0, color.RGBA{R: 255, G: 255, B: 255, A: 0xff}, true},
		{"a basic", "\x1b[33m>", 0, color.RGBA{R: 128, G: 128, A: 0xff}, true},
		{"a background passed over", "\x1b[38;5;220;48;2;1;2;3m>", 0, yellow, true},
		{"a background alone", "\x1b[48;5;240m>", 0, color.RGBA{}, false},
		{"a wide glyph's second cell", "\x1b[38;5;220m世>", 1, yellow, true},
		{"past the line's end", "\x1b[38;5;220m>", 3, color.RGBA{}, false},
		{"no colour at all", "> Theme", 0, color.RGBA{}, false},
	} {
		ink, inked := InkAt(c.line, c.cell)
		if ink != c.ink || inked != c.inked {
			t.Errorf("%s: InkAt(%q, %d) = %v, %v; want %v, %v", c.name, c.line, c.cell, ink, inked, c.ink, c.inked)
		}
	}
}
