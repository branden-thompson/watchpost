package main

// screen_test.go — the terminal the PTY journey reads (D-271).
//
// THE RENDERER SENDS ONLY THE CELLS THAT CHANGED, so the byte stream after a key
// is a set of cursor moves and fragments, and a word on screen need not appear in
// it contiguously. The journey therefore asserts on a screen, not on the stream:
// screen keeps a grid of cells and applies the stream to it as a terminal would.
// It interprets the movement, erase, insert, delete and scroll sequences an
// xterm renderer emits and ignores the rest (colours, modes, titles, queries),
// which carry no text.

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// screen is a grid of runes with a cursor, a scroll region and the parser's
// state between writes. A width-2 rune takes its cell and the next; widths
// are go-runewidth's, as the renderer's are.
type screen struct {
	rows, cols    int
	cells         [][]rune
	row, col      int
	top, bottom   int // the scroll region, inclusive, 0-based
	savedR, saveC int
	wrap          bool   // a rune filled the last column; the next one wraps
	last          rune   // the last rune printed, for REP
	pending       []byte // an escape sequence or rune split across writes
}

func newScreen(rows, cols int) *screen {
	s := &screen{rows: rows, cols: cols, bottom: rows - 1}
	s.cells = make([][]rune, rows)
	for r := range s.cells {
		s.cells[r] = blankRow(cols)
	}
	return s
}

func blankRow(cols int) []rune {
	row := make([]rune, cols)
	for i := range row {
		row[i] = ' '
	}
	return row
}

// text is the screen as lines, trailing blanks trimmed; a wide rune's second
// cell is not a character and is left out.
func (s *screen) text() string {
	var b strings.Builder
	for _, row := range s.cells {
		b.WriteString(strings.TrimRight(strings.ReplaceAll(string(row), "\x00", ""), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// write applies bytes from the terminal's output.
func (s *screen) write(p []byte) {
	buf := append(s.pending, p...)
	s.pending = nil
	for i := 0; i < len(buf); {
		n, done := s.step(buf[i:])
		if !done {
			s.pending = append([]byte(nil), buf[i:]...)
			return
		}
		i += n
	}
}

// step consumes one control, sequence or rune; done is false when buf ends
// inside one.
func (s *screen) step(buf []byte) (int, bool) {
	if c := buf[0]; c < 0x20 || c == 0x7f {
		s.wrap = false // any control or sequence ends a pending wrap
	}
	switch c := buf[0]; {
	case c == 0x1b:
		return s.escape(buf)
	case c == '\r':
		s.col = 0
	case c == '\n', c == 0x0b, c == 0x0c:
		s.lineFeed()
	case c == '\b':
		s.col = max(s.col-1, 0)
	case c == '\t':
		s.col = min((s.col/8+1)*8, s.cols-1)
	case c < 0x20 || c == 0x7f:
	default:
		if !utf8.FullRune(buf) {
			return 0, false
		}
		r, n := utf8.DecodeRune(buf)
		s.put(r)
		return n, true
	}
	return 1, true
}

// put prints r at the cursor. A rune in the last column leaves the cursor
// there with a wrap pending, which the next rune takes, as a terminal does.
func (s *screen) put(r rune) {
	w := max(runewidth.RuneWidth(r), 1)
	if s.wrap || s.col+w > s.cols {
		s.col, s.wrap = 0, false
		s.lineFeed()
	}
	s.cells[s.row][s.col] = r
	if w == 2 && s.col+1 < s.cols {
		s.cells[s.row][s.col+1] = 0
	}
	if s.col+w >= s.cols {
		s.col, s.wrap = s.cols-1, true
	} else {
		s.col += w
	}
	s.last = r
}

func (s *screen) lineFeed() {
	if s.row == s.bottom {
		s.scrollUp(1)
		return
	}
	s.row = min(s.row+1, s.rows-1)
}

// scrollUp moves the scroll region's lines up n, blanking the bottom.
func (s *screen) scrollUp(n int) {
	for range min(n, s.bottom-s.top+1) {
		copy(s.cells[s.top:s.bottom], s.cells[s.top+1:s.bottom+1])
		s.cells[s.bottom] = blankRow(s.cols)
	}
}

// scrollDown moves the scroll region's lines down n, blanking the top.
func (s *screen) scrollDown(n int) {
	for range min(n, s.bottom-s.top+1) {
		copy(s.cells[s.top+1:s.bottom+1], s.cells[s.top:s.bottom])
		s.cells[s.top] = blankRow(s.cols)
	}
}

// escape consumes an ESC sequence: CSI, a string (OSC, DCS, APC, PM, SOS), or
// a two- or three-byte escape.
func (s *screen) escape(buf []byte) (int, bool) {
	if len(buf) < 2 {
		return 0, false
	}
	switch buf[1] {
	case '[':
		return s.csi(buf)
	case ']', 'P', '_', '^', 'X':
		for i := 2; i < len(buf); i++ { // ends at BEL or ST (ESC \)
			if buf[i] == 0x07 {
				return i + 1, true
			}
			if buf[i] == 0x1b && i+1 < len(buf) && buf[i+1] == '\\' {
				return i + 2, true
			}
		}
		return 0, false
	case '(', ')', '*', '+', '#', '%', ' ':
		if len(buf) < 3 {
			return 0, false
		}
		return 3, true
	case '7':
		s.savedR, s.saveC = s.row, s.col
	case '8':
		s.row, s.col = s.savedR, s.saveC
	case 'D':
		s.lineFeed()
	case 'E':
		s.col = 0
		s.lineFeed()
	case 'M':
		if s.row == s.top {
			s.scrollDown(1)
		} else {
			s.row = max(s.row-1, 0)
		}
	case 'c':
		*s = *newScreen(s.rows, s.cols)
	}
	return 2, true
}

// csi consumes ESC [ params intermediates final.
func (s *screen) csi(buf []byte) (int, bool) {
	i := 2
	for i < len(buf) && buf[i] >= 0x30 && buf[i] <= 0x3f {
		i++
	}
	params := string(buf[2:i])
	for i < len(buf) && buf[i] >= 0x20 && buf[i] <= 0x2f {
		i++
	}
	if i >= len(buf) {
		return 0, false
	}
	final := buf[i]
	if params != "" && strings.IndexAny(params[:1], "<=>?") == 0 {
		return i + 1, true // a private mode, a query or a report: no text moves
	}
	if i > 2 && buf[i-1] >= 0x20 && buf[i-1] <= 0x2f {
		return i + 1, true // an intermediate byte: a cursor style or the like
	}
	s.apply(final, csiParams(params))
	return i + 1, true
}

// csiParams parses the numeric parameters; a colon sub-parameter is dropped.
func csiParams(p string) []int {
	if p == "" {
		return nil
	}
	parts := strings.Split(p, ";")
	out := make([]int, len(parts))
	for i, part := range parts {
		part, _, _ = strings.Cut(part, ":")
		out[i], _ = strconv.Atoi(part) // an empty or odd parameter is its default, 0
	}
	return out
}

// arg is parameter i, or def when absent or zero.
func arg(ps []int, i, def int) int {
	if i < len(ps) && ps[i] > 0 {
		return ps[i]
	}
	return def
}

// apply runs one CSI command.
func (s *screen) apply(final byte, ps []int) {
	n := arg(ps, 0, 1)
	switch final {
	case 'A':
		s.row = max(s.row-n, 0)
	case 'B', 'e':
		s.row = min(s.row+n, s.rows-1)
	case 'C', 'a':
		s.col = min(s.col+n, s.cols-1)
	case 'D':
		s.col = max(s.col-n, 0)
	case 'E':
		s.row, s.col = min(s.row+n, s.rows-1), 0
	case 'F':
		s.row, s.col = max(s.row-n, 0), 0
	case 'G', '`':
		s.col = clamp(n-1, s.cols)
	case 'd':
		s.row = clamp(n-1, s.rows)
	case 'I': // forward n tab stops, every 8 columns
		s.col = min((s.col/8+n)*8, s.cols-1)
	case 'Z': // back n tab stops
		s.col = max(((s.col+7)/8-n)*8, 0)
	case 'H', 'f':
		s.row, s.col = clamp(arg(ps, 0, 1)-1, s.rows), clamp(arg(ps, 1, 1)-1, s.cols)
	case 'J':
		s.eraseDisplay(arg(ps, 0, 0))
	case 'K':
		s.eraseLine(arg(ps, 0, 0))
	case 'X':
		for c := s.col; c < min(s.col+n, s.cols); c++ {
			s.cells[s.row][c] = ' '
		}
	case 'b':
		for range n {
			s.put(s.last)
		}
	case '@':
		row := s.cells[s.row]
		n = min(n, s.cols-s.col)
		copy(row[s.col+n:], row[s.col:s.cols-n])
		for c := s.col; c < s.col+n; c++ {
			row[c] = ' '
		}
	case 'P':
		row := s.cells[s.row]
		n = min(n, s.cols-s.col)
		copy(row[s.col:], row[s.col+n:])
		for c := s.cols - n; c < s.cols; c++ {
			row[c] = ' '
		}
	case 'L', 'M':
		if s.row < s.top || s.row > s.bottom {
			return
		}
		top := s.top
		s.top = s.row
		if final == 'L' {
			s.scrollDown(n)
		} else {
			s.scrollUp(n)
		}
		s.top = top
	case 'S':
		s.scrollUp(n)
	case 'T':
		s.scrollDown(n)
	case 'r':
		s.top, s.bottom = clamp(arg(ps, 0, 1)-1, s.rows), clamp(arg(ps, 1, s.rows)-1, s.rows)
		s.row, s.col = 0, 0
	case 's':
		s.savedR, s.saveC = s.row, s.col
	case 'u':
		s.row, s.col = s.savedR, s.saveC
	}
}

func clamp(v, n int) int { return min(max(v, 0), n-1) }

func (s *screen) eraseDisplay(mode int) {
	switch mode {
	case 0:
		s.eraseLine(0)
		for r := s.row + 1; r < s.rows; r++ {
			s.cells[r] = blankRow(s.cols)
		}
	case 1:
		s.eraseLine(1)
		for r := 0; r < s.row; r++ {
			s.cells[r] = blankRow(s.cols)
		}
	default:
		for r := range s.cells {
			s.cells[r] = blankRow(s.cols)
		}
	}
}

func (s *screen) eraseLine(mode int) {
	row := s.cells[s.row]
	from, to := s.col, s.cols
	switch mode {
	case 1:
		from, to = 0, s.col+1
	case 2:
		from = 0
	}
	for c := from; c < min(to, s.cols); c++ {
		row[c] = ' '
	}
}

// TestTheJourneyScreenAppliesWhatTheRendererSends holds the emulator to the
// sequences the renderer emits, each on a 3x10 screen. A sequence applied
// wrongly puts the journey's words on the wrong cells, and a check then passes
// or fails for the emulator's reason rather than the station's.
func TestTheJourneyScreenAppliesWhatTheRendererSends(t *testing.T) {
	for _, c := range []struct {
		name   string
		writes []string
		want   string
	}{
		{"CUP places the cursor", []string{"\x1b[2;3Hab"}, "\n  ab\n\n"},
		{"ECH blanks cells in place", []string{"abcdef\x1b[1;2H\x1b[3X"}, "a   ef\n\n\n"},
		{"EL erases to the line's end", []string{"abcd\x1b[1;3H\x1b[K"}, "ab\n\n\n"},
		{"CBT moves back to the tab stop", []string{"\x1b[1;10H\x1b[Zx"}, "        x\n\n\n"},
		{"HT moves on to the tab stop", []string{"a\tx"}, "a       x\n\n\n"},
		{"REP repeats the last rune", []string{"a\x1b[3b"}, "aaaa\n\n\n"},
		{"the last column wraps on the next rune", []string{"0123456789x"}, "0123456789\nx\n\n"},
		{"a wide rune takes two cells", []string{"界x\x1b[1;4Hy"}, "界xy\n\n\n"},
		{"a sequence split across writes", []string{"\x1b[1;", "3Hz"}, "  z\n\n\n"},
		{"a rune split across writes", []string{"\xe2\xa0", "\x80x"}, "⠀x\n\n\n"},
		{"a line feed at the bottom scrolls", []string{"a\r\nb\r\nc\r\nd"}, "b\nc\nd\n"},
		{"colours, modes and titles print nothing", []string{"\x1b[38;5;250mA\x1b[m\x1b[?25l\x1b]0;title\x07B"}, "AB\n\n\n"},
	} {
		s := newScreen(3, 10)
		for _, w := range c.writes {
			s.write([]byte(w))
		}
		if got := s.text(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
