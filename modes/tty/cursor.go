package tty

// cursor.go — where the terminal cursor stands (F-205 AX-3, D-362): at every
// focus point of both surfaces, a steady block that does not blink, so a
// screen reader, which follows the terminal's cursor, follows focus. It is
// found in the topmost layer's own lines - the confirmation, else the open
// window, else the surface - placed where the layer is drawn: on a field's
// caret while one is typed into, else on the focused row's pointer, else on
// the layer's first words (the map window on its words, A-107 (12)). While
// the terminal is focused its cell is tinted with 15 % of the glyph's colour
// over the cell's background, the glyph keeping its colour, under a bar in
// that tint; while it is not, the cursor is a block in the glyph's colour
// (D-366, D-367). The frame is searched on every one drawn, so a line is
// stripped of its colours only where it is read.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/render"
)

// caretIn is where a line's field caret is: its last caret glyph.
func caretIn(raw, caret string) (int, bool) {
	plain := plaintext.StripSGR(raw)
	i := strings.LastIndex(plain, caret)
	if i < 0 {
		return 0, false
	}
	return render.Width(plain[:i]), true
}

// wordStart is where a line's words begin past a box's side, false for a
// line with none.
func wordStart(raw string) (int, bool) {
	plain := plaintext.StripSGR(raw)
	words := strings.TrimLeft(plain, Sides)
	if strings.TrimRight(words, Sides) == "" {
		return 0, false
	}
	return render.Width(plain) - render.Width(words), true
}

// cursorIn is the cursor among a layer's lines: the last field caret while
// one is typed into, else the first row pointer; false with neither.
func cursorIn(lines []string, o render.Opts, typing bool) (row, col int, ok bool) {
	g := o.Glyphs()
	for i := len(lines) - 1; typing && i >= 0; i-- {
		if c, found := caretIn(lines[i], g.Cursor); found {
			return i, c, true
		}
	}
	for i, l := range lines {
		if c, found := FocusMark(l, g.Pointer); found {
			return i, c, true
		}
	}
	return 0, 0, false
}

// firstWords is the first line of a window's body with words on it, past its
// top border, and where they start.
func firstWords(lines []string) (row, col int, ok bool) {
	for i := 1; i < len(lines); i++ {
		if c, found := wordStart(lines[i]); found {
			return i, c, true
		}
	}
	return 0, 0, false
}

// typing reports whether the open window is a field the listener types
// into: the search, the place finder, the frequency.
func (d Dashboard) typing() bool {
	return d.modal == modalAdd || d.modal == modalPropFind || d.modal == modalPropFreq
}

// windowCursor is the cursor in the open window's own lines.
func (d Dashboard) windowCursor(o render.Opts, lines []string) (row, col int, ok bool) {
	if d.modal == modalSetup {
		return setupCursor(o, lines)
	}
	if r, c, found := cursorIn(lines, o, d.typing()); found {
		return r, c, true
	}
	if d.modal == modalMap {
		if r, c, found := d.mapWordsCursor(lines); found {
			return r, c, true
		}
	}
	return firstWords(lines)
}

// setupCursor is the cursor in Settings: the focused row's pointer, or the
// last caret among the lines its row draws - those under it indented deeper
// than its label - so a caret on a row not focused is never taken.
func setupCursor(o render.Opts, lines []string) (row, col int, ok bool) {
	at, c, found := cursorIn(lines, o, false)
	if !found {
		return firstWords(lines)
	}
	label := c + render.Width(o.Glyphs().Pointer) + 1
	end := at + 1
	for ; end < len(lines); end++ {
		if w, words := wordStart(lines[end]); !words || w <= label {
			break // a blank line, or the next row's label: the focused row's lines end
		}
	}
	for i := end - 1; i >= at; i-- {
		if cc, typed := caretIn(lines[i], o.Glyphs().Cursor); typed {
			return i, cc, true
		}
	}
	return at, c, true
}

// mapWordsCursor is the map window's words when nothing else has focus
// (A-107 (12)): the Description box's first line while it is open, else the
// first line under the picture.
func (d Dashboard) mapWordsCursor(lines []string) (row, col int, ok bool) {
	under := -1
	for i, l := range lines {
		if d.mapPane.alertsOn && i+1 < len(lines) && strings.Contains(plaintext.StripSGR(l), "┌─ Description") {
			if c, found := wordStart(lines[i+1]); found {
				return i + 1, c, true
			}
		}
		if strings.ContainsFunc(l, func(r rune) bool { return r >= 0x2800 && r <= 0x28ff }) {
			under = i + 1 // past the picture's last row of braille
		}
	}
	for i := max(under, 1); under >= 0 && i < len(lines); i++ {
		if c, found := wordStart(lines[i]); found {
			return i, c, true
		}
	}
	return 0, 0, false
}

// frameCursor is where the cursor stands in the finished frame: in the
// confirmation over the window, else the window, else the surface, each
// placed where OverlayOrigin lays it across a terminal this wide; false where
// none of them has words.
func (d Dashboard) frameCursor(o render.Opts, base, win string, winY int, box string, termWidth int) (x, y int, ok bool) {
	if box != "" {
		x, y := render.OverlayOrigin(base, box, termWidth, -1)
		lines := strings.Split(box, "\n")
		if r, c, found := cursorIn(lines, o, true); found {
			return x + c, y + r, true
		}
		if r, c, found := firstWords(lines); found {
			return x + c, y + r, true
		}
	}
	if win != "" {
		x, y := render.OverlayOrigin(base, win, termWidth, winY)
		r, c, found := d.windowCursor(o, strings.Split(win, "\n"))
		return x + c, y + r, found
	}
	r, c, found := SurfaceMark(base, o.Glyphs().Pointer)
	return c, r, found
}

// heldCursor is the body memo's cursor slot, nil without a memo.
func (d Dashboard) heldCursor() *CursorSlot {
	if d.memo == nil {
		return nil
	}
	return &d.memo.cursor
}

// consoleCursor is where the console's cursor stands: on its focused row's
// pointer, the line-up's or the pool's.
func (b Broadcaster) consoleCursor(content string) (x, y int, ok bool) {
	r, c, found := SurfaceMark(content, b.opts().Glyphs().Pointer)
	return c, r, found
}

// heldCursor is the console memo's cursor slot, nil without a memo.
func (b Broadcaster) heldCursor() *CursorSlot {
	if b.memo == nil {
		return nil
	}
	return &b.memo.cursor
}
