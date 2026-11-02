package render

import (
	"image/color"
	"strings"
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

// TestCellAtReadsTheCellsBackground is D-367: the background a line draws a
// cell on - 256, truecolor and basic, beside a foreground - cleared by 49 or
// a reset, 39 leaving it, and none where the terminal's own stands.
func TestCellAtReadsTheCellsBackground(t *testing.T) {
	grey := color.RGBA{R: 88, G: 88, B: 88, A: 0xff} // xterm 240
	for _, c := range []struct {
		name    string
		line    string
		cell    int
		paper   color.RGBA
		papered bool
	}{
		{"a 256 background", "\x1b[38;5;220;48;5;240m>", 0, grey, true},
		{"a truecolor one", "\x1b[48;2;1;2;3m>", 0, color.RGBA{R: 1, G: 2, B: 3, A: 0xff}, true},
		{"a basic one", "\x1b[41m>", 0, color.RGBA{R: 128, A: 0xff}, true},
		{"a bright basic one", "\x1b[107m>", 0, color.RGBA{R: 255, G: 255, B: 255, A: 0xff}, true},
		{"49 clears it", "\x1b[48;5;240m>\x1b[49m>", 1, color.RGBA{}, false},
		{"a reset clears it", "\x1b[48;5;240m>\x1b[0m>", 1, color.RGBA{}, false},
		{"39 leaves it", "\x1b[38;5;220;48;5;240m>\x1b[39m>", 1, grey, true},
		{"an underline's colour is not it", "\x1b[58;5;240m>", 0, color.RGBA{}, false},
		{"none", "> Theme", 0, color.RGBA{}, false},
	} {
		cell, ok := CellAt(c.line, c.cell)
		if !ok || cell.Paper != c.paper || cell.Papered != c.papered {
			t.Errorf("%s: CellAt(%q, %d) has paper %v, %v; want %v, %v", c.name, c.line, c.cell, cell.Paper, cell.Papered, c.paper, c.papered)
		}
	}
}

// TestBlendLaysAShareOfOneColourOverAnother is D-367's 15 %: none of the top
// colour at 0, all of it at 1, and between, rounded.
func TestBlendLaysAShareOfOneColourOverAnother(t *testing.T) {
	yellow, under := color.RGBA{R: 255, G: 215, A: 0xff}, color.RGBA{R: 40, G: 40, B: 40, A: 0xff}
	for _, c := range []struct {
		share float64
		want  color.RGBA
	}{
		{0, under}, {1, color.RGBA{R: 255, G: 215, A: 0xff}}, {0.15, color.RGBA{R: 72, G: 66, B: 34, A: 0xff}},
	} {
		if got := Blend(yellow, under, c.share); got != c.want {
			t.Errorf("Blend at %v = %v; want %v", c.share, got, c.want)
		}
	}
}

// TestPaintCellSetsOnlyThatCellsBackground is D-367: the cell takes the
// colour, its glyph and foreground untouched, and the background it had is
// set back after it - its own, or the terminal's - the line no wider.
func TestPaintCellSetsOnlyThatCellsBackground(t *testing.T) {
	tint := color.RGBA{R: 72, G: 66, B: 34, A: 0xff}
	for _, c := range []struct {
		name, text string
		cell       int
	}{
		{"on the terminal's background", "top\n  \x1b[1;38;5;220m>\x1b[0m Theme", 2},
		{"on a window's", "top\n\x1b[48;5;240m  \x1b[38;5;220m>\x1b[39m Theme\x1b[0m", 2},
	} {
		from := strings.Index(c.text, "\n") + 1
		before, _ := CellAt(c.text[from:], c.cell)
		out := PaintCell(c.text, from, before, tint)
		line := out[from:]
		after, _ := CellAt(line, c.cell)
		if after.Ink != before.Ink || after.Inked != before.Inked || after.Paper != tint || !after.Papered {
			t.Errorf("%s: the cell is %+v; want %v on %v", c.name, after, before.Ink, tint)
		}
		next, _ := CellAt(line, c.cell+1)
		if was, _ := CellAt(c.text[from:], c.cell+1); next.Paper != was.Paper || next.Papered != was.Papered {
			t.Errorf("%s: the next cell is on %v, %v; want its own %v, %v", c.name, next.Paper, next.Papered, was.Paper, was.Papered)
		}
		if Width(line) != Width(c.text[from:]) || !strings.HasPrefix(out, "top\n") {
			t.Errorf("%s: painted %q from %q", c.name, out, c.text)
		}
	}
}
