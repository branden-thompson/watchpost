package tty

// kit_cursor.go — the cursor both surfaces place: where the focus sits in
// a drawn line, how it is marked, and the slot that holds it for a frame
// (0.19.6 A-21). The kit's cursor.go once B2 moves it.

import (
	"image/color"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/platform/render"
)

// Sides are what may stand before a row's pointer: space and a box's side.
const Sides = " │┃|"

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

// CursorSlot holds a surface's last finished frame and its cursor. A cursor
// handed out is never changed: Bubble Tea keeps the last frame's cursor to
// compare the next against, and sends its colour only when the two differ,
// so a new one is made only when the cursor moves or changes its look, and a
// frame drawn again as it was allocates nothing.
type CursorSlot struct {
	primed  bool
	in, out string
	x, y    int
	blurred bool
	back    color.RGBA
	backed  bool
	look    cursorLook
	held    *tea.Cursor
}

// Finish is a frame and its cursor at a cell of it, nil where there is none:
// under a focused terminal the cell tinted and a bar in the tint, under one
// not focused a block in the glyph's colour, the terminal's own with no
// colour set (D-366, D-367). bg is the frame's background, where a cell sets
// none. From the slot when nothing has changed; a nil slot finishes afresh.
func (s *CursorSlot) Finish(content string, x, y int, ok, blurred bool, bg color.Color) (string, *tea.Cursor) {
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
	*s = CursorSlot{primed: true, in: content, out: out, x: x, y: y, blurred: blurred, back: back, backed: backed, look: look, held: held}
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

// FocusMark is where a line's row pointer is: the pointer with nothing before
// it but space and a box's side, so a tab's pointer between brackets, or a
// legend's among words, is never taken for it.
func FocusMark(raw, pointer string) (int, bool) {
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
		if !strings.ContainsRune(Sides, r) {
			return 0, false
		}
		cells, i = cells+1, i+size
	}
	return cells, true
}

// SurfaceMark is the first row pointer in the surface's frame, its lines read
// in place.
func SurfaceMark(frame, pointer string) (row, col int, ok bool) {
	for row = range len(frame) + 1 { // a line a pass, never more lines than bytes (P10-02)
		line, rest, more := strings.Cut(frame, "\n")
		if c, found := FocusMark(line, pointer); found {
			return row, c, true
		}
		if !more {
			break
		}
		frame = rest
	}
	return 0, 0, false
}
