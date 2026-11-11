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
	"image/color"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/render"
)

// sides are what may stand before a row's pointer: space and a box's side.
const sides = " │┃|"

// focusShare is how much of the glyph's colour tints its cell under a
// focused terminal's cursor, over the cell's own background (D-367).
const focusShare = 0.15

// cursorLook is how the terminal draws the cursor: its colour, the
// terminal's own where none, and its shape.
type cursorLook struct {
	colour   color.RGBA
	coloured bool
	shape    tea.CursorShape
}

// cursorSlot holds a surface's last finished frame and its cursor. A cursor
// handed out is never changed: Bubble Tea keeps the last frame's cursor to
// compare the next against, and sends its colour only when the two differ,
// so a new one is made only when the cursor moves or changes its look, and a
// frame drawn again as it was allocates nothing.
type cursorSlot struct {
	primed  bool
	in, out string
	x, y    int
	blurred bool
	back    color.RGBA
	backed  bool
	look    cursorLook
	held    *tea.Cursor
}

// finish is a frame and its cursor at a cell of it, nil where there is none:
// under a focused terminal the cell tinted and a bar in the tint, under one
// not focused a block in the glyph's colour, the terminal's own with no
// colour set (D-366, D-367). bg is the frame's background, where a cell sets
// none. From the slot when nothing has changed; a nil slot finishes afresh.
func (s *cursorSlot) finish(content string, x, y int, ok, blurred bool, bg color.Color) (string, *tea.Cursor) {
	if !ok {
		return content, nil
	}
	back, backed := render.RGBAOf(bg)
	if s != nil && s.primed && s.x == x && s.y == y && s.blurred == blurred && s.back == back && s.backed == backed && s.in == content {
		return s.out, s.held
	}
	out, look := focusCell(content, x, y, blurred, bg)
	if s == nil {
		return out, look.cursor(x, y)
	}
	held := s.held
	if held == nil || held.X != x || held.Y != y || s.look != look {
		held = look.cursor(x, y)
	}
	*s = cursorSlot{primed: true, in: content, out: out, x: x, y: y, blurred: blurred, back: back, backed: backed, look: look, held: held}
	return out, held
}

// cursor is a steady cursor of this look at a cell: it never blinks, so it
// adds no motion (D-362).
func (l cursorLook) cursor(x, y int) *tea.Cursor {
	c := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: l.shape}
	if l.coloured {
		c.Color = l.colour
	}
	return c
}

// focusCell is the frame with the cell under the cursor finished, and the
// cursor's look: the glyph's colour, a block; and while the terminal is
// focused, the cell tinted with focusShare of it over its background - its
// own, else the frame's - under a bar in the tint. A cell with no colour
// set, or none to tint over, is left as drawn under the terminal's own.
func focusCell(content string, x, y int, blurred bool, bg color.Color) (string, cursorLook) {
	from, line := lineAt(content, y)
	cell, found := render.CellAt(line, x)
	if !found || !cell.Inked {
		return content, cursorLook{shape: tea.CursorBlock}
	}
	look := cursorLook{colour: cell.Ink, coloured: true, shape: tea.CursorBlock}
	paper, papered := cell.Paper, cell.Papered
	if !papered {
		paper, papered = render.RGBAOf(bg)
	}
	if blurred || !papered {
		return content, look
	}
	tint := render.Blend(cell.Ink, paper, focusShare)
	return render.PaintCell(content, from, cell, tint), cursorLook{colour: tint, coloured: true, shape: tea.CursorBar}
}

// lineAt is a frame's line by its row and the byte it starts at, read in
// place; "" past the frame's end.
func lineAt(frame string, row int) (from int, line string) {
	for at := range len(frame) + 1 { // a line a pass, never more lines than bytes (P10-02)
		l, _, more := strings.Cut(frame[from:], "\n")
		if at == row {
			return from, l
		}
		if !more {
			break
		}
		from += len(l) + 1
	}
	return 0, ""
}

// focusMark is where a line's row pointer is: the pointer with nothing before
// it but space and a box's side, so a tab's pointer between brackets, or a
// legend's among words, is never taken for it.
func focusMark(raw, pointer string) (int, bool) {
	i := strings.Index(raw, pointer)
	if i < 0 {
		return 0, false
	}
	return sidesOnly(raw[:i])
}

// sidesOnly is how many cells a run holds when it is nothing but space and a
// box's side between its colours, read in place: the surface's frame is
// searched on every one drawn, and a copy without colours would cost one.
func sidesOnly(run string) (int, bool) {
	cells, i := 0, 0
	for range len(run) { // every pass consumes at least one byte (P10-02)
		if i >= len(run) {
			break
		}
		if run[i] == 0x1b {
			end := strings.IndexByte(run[i:], 'm')
			if end < 0 {
				return 0, false
			}
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(run[i:])
		if !strings.ContainsRune(sides, r) {
			return 0, false
		}
		cells, i = cells+1, i+size
	}
	return cells, true
}

// surfaceMark is the first row pointer in the surface's frame, its lines read
// in place.
func surfaceMark(frame, pointer string) (row, col int, ok bool) {
	for row = range len(frame) + 1 { // a line a pass, never more lines than bytes (P10-02)
		line, rest, more := strings.Cut(frame, "\n")
		if c, found := focusMark(line, pointer); found {
			return row, c, true
		}
		if !more {
			break
		}
		frame = rest
	}
	return 0, 0, false
}

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
	words := strings.TrimLeft(plain, sides)
	if strings.TrimRight(words, sides) == "" {
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
		if c, found := focusMark(l, g.Pointer); found {
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
	r, c, found := surfaceMark(base, o.Glyphs().Pointer)
	return c, r, found
}

// heldCursor is the body memo's cursor slot, nil without a memo.
func (d Dashboard) heldCursor() *cursorSlot {
	if d.memo == nil {
		return nil
	}
	return &d.memo.cursor
}

// consoleCursor is where the console's cursor stands: on its focused row's
// pointer, the line-up's or the pool's.
func (b Broadcaster) consoleCursor(frame string) (x, y int, ok bool) {
	r, c, found := surfaceMark(frame, b.opts().Glyphs().Pointer)
	return c, r, found
}

// heldCursor is the console memo's cursor slot, nil without a memo.
func (b Broadcaster) heldCursor() *cursorSlot {
	if b.memo == nil {
		return nil
	}
	return &b.memo.cursor
}
