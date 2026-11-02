package render

// ink.go — the colours a line draws one of its cells in, read from the
// line's own SGR sequences, and that cell painted: the terminal cursor wears
// the glyph's colour while the terminal is not focused (D-366), and while it
// is, the cell takes a tint of it over its own background, the glyph keeping
// its colour (D-367).

import (
	"image/color"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Cell is what a line draws one of its cells in: its foreground (Ink) and
// background (Paper), each false where the terminal's own stands, and where
// its glyph's bytes lie in the line.
type Cell struct {
	Ink, Paper     color.RGBA
	Inked, Papered bool
	start, end     int
}

// CellAt is the cell of a line at a column, the colours its SGR sequences
// leave set when they reach it; false past the line's end. It reads in
// place: the cursor asks it of every frame drawn.
func CellAt(line string, cell int) (Cell, bool) {
	var pen Cell
	at, i := 0, 0
	for range len(line) { // every pass consumes at least one byte (P10-02)
		if i >= len(line) {
			break
		}
		if line[i] == 0x1b {
			end := strings.IndexByte(line[i:], 'm')
			if end < 2 {
				return Cell{}, false
			}
			pen = pen.apply(line[i+2 : i+end])
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		w := RuneCells(r)
		if cell < at+w {
			pen.start, pen.end = i, i+size
			return pen, true
		}
		at, i = at+w, i+size
	}
	return Cell{}, false
}

// InkAt is the foreground a line leaves set at one of its cells; false where
// the terminal's own stands or past the line's end.
func InkAt(line string, cell int) (color.RGBA, bool) {
	c, ok := CellAt(line, cell)
	return c.Ink, ok && c.Inked
}

// apply is the pen one SGR sequence's parameters leave, from the one before:
// a reset clears both colours, 39 and 49 each one, and a basic, 256 or
// truecolor code sets one; an underline's colour is passed over.
func (pen Cell) apply(params string) Cell {
	p := sgrParams{rest: params}
	for range len(params) + 1 { // a parameter a pass, never more than the bytes (P10-02)
		n, ok := p.next()
		if !ok {
			break
		}
		switch {
		case n == 0:
			pen = Cell{}
		case n == 39:
			pen.Ink, pen.Inked = color.RGBA{}, false
		case n == 49:
			pen.Paper, pen.Papered = color.RGBA{}, false
		case n >= 30 && n <= 37:
			pen.Ink, pen.Inked = paletteInk(n-30), true
		case n >= 90 && n <= 97:
			pen.Ink, pen.Inked = paletteInk(n-90+8), true
		case n >= 40 && n <= 47:
			pen.Paper, pen.Papered = paletteInk(n-40), true
		case n >= 100 && n <= 107:
			pen.Paper, pen.Papered = paletteInk(n-100+8), true
		case n == 38:
			pen.Ink, pen.Inked = p.extended(pen.Ink, pen.Inked)
		case n == 48:
			pen.Paper, pen.Papered = p.extended(pen.Paper, pen.Papered)
		case n == 58:
			p.extended(color.RGBA{}, false)
		}
	}
	return pen
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

// extended reads a 38, 48 or 58 code's colour: 5 and a 256 index, or 2 and
// red, green and blue; anything else leaves the colour as it was.
func (p *sgrParams) extended(was color.RGBA, set bool) (color.RGBA, bool) {
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
	return was, set
}

// paletteInk is a 256-colour index as the xterm palette draws it; the basic
// sixteen are the terminal's own, so theirs is the nearest standard.
func paletteInk(n int) color.RGBA {
	r, g, b := xterm256RGB(n)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}
}

// Blend is one colour laid over another at a share of it, 0 to 1.
func Blend(top, under color.RGBA, share float64) color.RGBA {
	mix := func(t, u uint8) uint8 { return uint8(float64(u) + (float64(t)-float64(u))*share + 0.5) }
	return color.RGBA{R: mix(top.R, under.R), G: mix(top.G, under.G), B: mix(top.B, under.B), A: 0xff}
}

// RGBAOf is a colour as red, green and blue; false for none.
func RGBAOf(c color.Color) (color.RGBA, bool) {
	if c == nil {
		return color.RGBA{}, false
	}
	r, g, b, _ := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0xff}, true
}

// PaintCell is a text with one cell's background set to a colour - the cell
// as CellAt found it on the line starting at byte from - and the background
// it had set back after it; its glyph and foreground are untouched.
func PaintCell(text string, from int, c Cell, paper color.RGBA) string {
	back := "\x1b[49m"
	if c.Papered {
		back = paperSGR(c.Paper)
	}
	var b strings.Builder
	b.Grow(len(text) + 2*len("\x1b[48;2;255;255;255m"))
	b.WriteString(text[:from+c.start])
	b.WriteString(paperSGR(paper))
	b.WriteString(text[from+c.start : from+c.end])
	b.WriteString(back)
	b.WriteString(text[from+c.end:])
	return b.String()
}

// paperSGR is the SGR sequence setting a truecolor background.
func paperSGR(p color.RGBA) string {
	return "\x1b[48;2;" + strconv.Itoa(int(p.R)) + ";" + strconv.Itoa(int(p.G)) + ";" + strconv.Itoa(int(p.B)) + "m"
}
