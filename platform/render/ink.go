package render

// ink.go — the colour a line draws one of its cells in, read from the line's
// own SGR sequences: the terminal cursor wears it, so the glyph it covers
// keeps its colour (D-366).

import (
	"image/color"
	"strconv"
	"strings"
	"unicode/utf8"
)

// InkAt is the foreground a line leaves set when it reaches one of its
// cells; false where the cell takes the terminal's own (none set, a reset,
// or past the line's end). It reads in place: the cursor asks it of every
// frame drawn.
func InkAt(line string, cell int) (color.RGBA, bool) {
	var ink color.RGBA
	inked, at, i := false, 0, 0
	for range len(line) { // every pass consumes at least one byte (P10-02)
		if i >= len(line) {
			break
		}
		if line[i] == 0x1b {
			end := strings.IndexByte(line[i:], 'm')
			if end < 2 {
				return color.RGBA{}, false
			}
			ink, inked = sgrInk(line[i+2:i+end], ink, inked)
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		w := RuneCells(r)
		if cell < at+w {
			return ink, inked
		}
		at, i = at+w, i+size
	}
	return color.RGBA{}, false
}

// sgrInk is the foreground one SGR sequence's parameters leave, from the one
// set before it: a reset or 39 clears it, a basic, 256 or truecolor code sets
// it, and a background's values are passed over.
func sgrInk(params string, ink color.RGBA, inked bool) (color.RGBA, bool) {
	p := sgrParams{rest: params}
	for range len(params) + 1 { // a parameter a pass, never more than the bytes (P10-02)
		n, ok := p.next()
		if !ok {
			break
		}
		switch {
		case n == 0 || n == 39:
			ink, inked = color.RGBA{}, false
		case n >= 30 && n <= 37:
			ink, inked = paletteInk(n-30), true
		case n >= 90 && n <= 97:
			ink, inked = paletteInk(n-90+8), true
		case n == 38:
			ink, inked = p.extended(ink, inked)
		case n == 48 || n == 58:
			p.extended(ink, inked)
		}
	}
	return ink, inked
}

// sgrParams walks an SGR sequence's parameters in place.
type sgrParams struct {
	rest string
	done bool
}

// next is the next parameter: an empty one is 0, as the terminal reads it,
// and one that is not a number is -1, which sets nothing.
func (p *sgrParams) next() (int, bool) {
	if p.done {
		return 0, false
	}
	field, rest, more := strings.Cut(p.rest, ";")
	p.rest, p.done = rest, !more
	if field == "" {
		return 0, true
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return -1, true
	}
	return n, true
}

// extended reads a 38 (or 48, 58) code's colour: 5 and a 256 index, or 2 and
// red, green and blue; anything else leaves the ink as it was.
func (p *sgrParams) extended(ink color.RGBA, inked bool) (color.RGBA, bool) {
	switch kind, _ := p.next(); kind {
	case 5:
		if n, _ := p.next(); n >= 0 && n <= 255 {
			return paletteInk(n), true
		}
	case 2:
		r, _ := p.next()
		g, _ := p.next()
		b, _ := p.next()
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}, true
	}
	return ink, inked
}

// paletteInk is a 256-colour index as the xterm palette draws it; the basic
// sixteen are the terminal's own, so theirs is the nearest standard.
func paletteInk(n int) color.RGBA {
	r, g, b := xterm256RGB(n)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}
}
