package tty

// cursor.go — where the terminal cursor stands (F-205 AX-3, D-362): at every
// focus point of both surfaces, a steady block that does not blink, so a
// screen reader, which follows the terminal's cursor, follows focus. It is
// found in the topmost layer's own lines - the confirmation, else the open
// window, else the surface - placed where the layer is drawn: on a field's
// caret while one is typed into, else on the focused row's pointer, else on
// the layer's first words (the map window on its words, A-107 (12)). It
// wears the colour of the cell it covers, so a yellow pointer under it stays
// yellow, the glyph drawn in the background's colour (D-366). The frame is
// searched on every one drawn, so a line is stripped of its colours only
// where it is read.

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

// cursorSlot holds the last cursor a surface drew. One handed out is never
// changed: Bubble Tea keeps the last frame's cursor to compare the next
// against, and sends its colour only when the two differ, so a new one is
// made only when the cursor moves or changes colour, and a frame where
// neither happens allocates nothing for it.
type cursorSlot struct {
	held  *tea.Cursor
	ink   color.RGBA
	inked bool
}

// at is the cursor on a cell of the frame: a block, not blinking, so it adds
// no motion (D-362), in the colour the layer's line draws its cell in
// (D-366), the terminal's own where it sets none; from the slot when nothing
// has changed. A nil slot makes one each time.
func (s *cursorSlot) at(x, y int, line string, col int) *tea.Cursor {
	ink, inked := render.InkAt(line, col)
	if s != nil && s.held != nil && s.held.X == x && s.held.Y == y && s.ink == ink && s.inked == inked {
		return s.held
	}
	c := &tea.Cursor{Position: tea.Position{X: x, Y: y}, Shape: tea.CursorBlock}
	if inked {
		c.Color = ink
	}
	if s != nil {
		s.held, s.ink, s.inked = c, ink, inked
	}
	return c
}

// steadyCursor is the cursor on a cell of the frame, from the body memo's
// slot where there is one.
func (d Dashboard) steadyCursor(x, y int, line string, col int) *tea.Cursor {
	var slot *cursorSlot
	if d.memo != nil {
		slot = &d.memo.cursor
	}
	return slot.at(x, y, line, col)
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

// surfaceMark is the first row pointer in the surface's frame and the line it
// is on, its lines read in place.
func surfaceMark(frame, pointer string) (row, col int, line string, ok bool) {
	for row = range len(frame) + 1 { // a line a pass, never more lines than bytes (P10-02)
		l, rest, more := strings.Cut(frame, "\n")
		if c, found := focusMark(l, pointer); found {
			return row, c, l, true
		}
		if !more {
			break
		}
		frame = rest
	}
	return 0, 0, "", false
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

// frameCursor is the cursor in the finished frame: in the confirmation over
// the window, else the window, else the surface, each placed where
// OverlayOrigin lays it across a terminal this wide; nil where none of them
// has words.
func (d Dashboard) frameCursor(o render.Opts, base, win string, winY int, box string, termWidth int) *tea.Cursor {
	if box != "" {
		x, y := render.OverlayOrigin(base, box, termWidth, -1)
		lines := strings.Split(box, "\n")
		if r, c, ok := cursorIn(lines, o, true); ok {
			return d.steadyCursor(x+c, y+r, lines[r], c)
		}
		if r, c, ok := firstWords(lines); ok {
			return d.steadyCursor(x+c, y+r, lines[r], c)
		}
	}
	if win != "" {
		x, y := render.OverlayOrigin(base, win, termWidth, winY)
		lines := strings.Split(win, "\n")
		if r, c, ok := d.windowCursor(o, lines); ok {
			return d.steadyCursor(x+c, y+r, lines[r], c)
		}
		return nil
	}
	if r, c, line, ok := surfaceMark(base, o.Glyphs().Pointer); ok {
		return d.steadyCursor(c, r, line, c)
	}
	return nil
}

// OverlayCursor is the cursor of the window OverlayWindow lays over another
// surface's frame - the console's - as it lays it, centred.
func (d Dashboard) OverlayCursor(base string, termWidth int) *tea.Cursor {
	if !d.ModalOpen() {
		return nil
	}
	o := d.layout().o
	return d.frameCursor(o, base, d.modalView(o), -1, d.confirmOverlay(o), termWidth)
}

// consoleCursor is the console's cursor: on its focused row's pointer, the
// line-up's or the pool's, kept in the console's memo slot.
func (b Broadcaster) consoleCursor(frame string) *tea.Cursor {
	r, c, line, ok := surfaceMark(frame, b.opts().Glyphs().Pointer)
	if !ok {
		return nil
	}
	var slot *cursorSlot
	if b.memo != nil {
		slot = &b.memo.cursor
	}
	return slot.at(c, r, line, c)
}
