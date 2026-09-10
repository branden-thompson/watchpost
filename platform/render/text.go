package render

// text.go — plain text: wrapping, padding, display width, ANSI stripping, byte formatting. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/plaintext"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

// WrapSegments greedily packs help segments into width-bound lines (UAT
// 6.7: the key-binding footer wraps smartly with terminal width). ANSI-aware.
func WrapSegments(segs []string, width int, sep string) []string {
	var lines []string
	cur := ""
	for _, seg := range segs {
		if cur == "" {
			cur = seg
			continue
		}
		if displayWidth(cur)+displayWidth(sep)+displayWidth(seg) > width {
			lines = append(lines, cur)
			cur = seg
			continue
		}
		cur += sep + seg
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines // one entry per row (Q6, L3-F13: callers used to re-split on "\n")
}

// Thousands groups a whole number ("12,915") — the one owner of the
// acreage format the FIRE rows and the broadcast share (Q6, L3-F8).
func Thousands(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Control is one key-cap chip with its label in a control row.
type Control struct {
	Key, Label string
	Muted      bool // an inert control reads muted (UAT 21.1)
}

// Ctl builds a live control; CtlIf a control that is live only when on.
func Ctl(key, label string) Control            { return Control{Key: key, Label: label} }
func CtlIf(key, label string, on bool) Control { return Control{Key: key, Label: label, Muted: !on} }

// Controls renders a control row — "[key] label" chips joined by gap — the
// one owner of the modal footers' shape (Q6, L3-F12).
func (o Opts) Controls(gap string, items ...Control) string {
	parts := make([]string, 0, len(items))
	for _, c := range items {
		parts = append(parts, o.KeyCapIf(c.Key, !c.Muted)+" "+c.Label)
	}
	return strings.Join(parts, gap)
}

// WrapText word-wraps plain prose to width-bound lines (UAT 15.2: alert
// bodies wrap, never truncate). Single owner - modals and modules share it.
func WrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var lines []string
	cur := ""
	for _, word := range strings.Fields(text) {
		// A WORD WIDER THAN THE LINE is broken across lines rather than left to
		// overflow. Without this the wrap was a wrap only for prose: a provider
		// error carrying a 200-character URL — no spaces in it anywhere — came
		// out as one over-wide line and the panel cut it, losing the half a
		// listener would need to act on it.
		//
		// That is the truncation this function's own contract says callers
		// cannot reintroduce, so it belongs here rather than at the one caller
		// that noticed.
		if displayWidth(word) > width {
			parts := splitCells(word, width)
			if cur != "" {
				lines = append(lines, cur)
			}
			lines = append(lines, parts[:len(parts)-1]...)
			cur = parts[len(parts)-1] // the tail keeps collecting the words after it
			continue
		}
		switch {
		case cur == "":
			cur = word
		case displayWidth(cur)+1+displayWidth(word) > width:
			lines = append(lines, cur)
			cur = word
		default:
			cur += " " + word
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// splitCells breaks s into pieces of at most width display cells.
//
// An escape sequence is copied WHOLE and costs no cells: a break inside one
// would put half an SGR code on each line, which the terminal reads as text.
// Always returns at least one piece, so callers can take its tail.
func splitCells(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	var out []string
	var b strings.Builder
	cells, inEscape := 0, false
	for _, r := range s { // one pass over the runes — no cursor to run away with
		switch {
		case inEscape:
			b.WriteRune(r)
			inEscape = r != 'm' // the sequence ends at its terminator
			continue
		case r == 0x1b:
			b.WriteRune(r)
			inEscape = true
			continue
		}
		if w := RuneCells(r); cells+w > width && cells > 0 {
			out = append(out, b.String())
			b.Reset()
			cells = 0
		}
		b.WriteRune(r)
		cells += RuneCells(r)
	}
	if b.Len() > 0 || len(out) == 0 {
		out = append(out, b.String())
	}
	return out
}

// WrapLines wraps every over-wide body line to width, preserving each
// line's leading indent on continuations (blank lines pass through). THE
// modal-content guarantee: floating windows wrap, never truncate — callers
// cannot reintroduce the truncation class of bug (UAT 25).
func WrapLines(lines []string, width int) []string {
	if width < 8 {
		width = 8
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if displayWidth(line) <= width {
			out = append(out, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		for _, w := range WrapText(strings.TrimLeft(line, " "), width-displayWidth(indent)) {
			out = append(out, indent+w)
		}
	}
	return out
}

// WrapHanging lays one labelled row out: the first line follows head, and every
// line after it is indented to head's DISPLAY width, so a value too long for the
// row reads as one column instead of falling back to the margin.
//
//	›  Recommended:  Tune to WNG712 Coachella / Spanish CA
//	                 162.525 MHz (81 mi)
//
// IT IS THE OTHER HALF OF WrapLines' GUARANTEE. That one keeps a modal's PROSE
// from overflowing and preserves a line's own indent on continuation; a labelled
// row needs the continuation under the VALUE, not under the label, or the wrap
// reads as a new row. Without it the only way to make such a row fit is to cut
// it — which is the truncation class UAT 25 ruled out, and in an error window it
// is the address of the station the listener is being told to tune to.
//
// head is measured ANSI-aware, so a tinted label indents by what it LOOKS like.
func WrapHanging(head, text string, width int) []string {
	room := width - displayWidth(head)
	if room < 8 {
		room = 8 // a head wider than the window still leaves the value somewhere to go
	}
	parts := WrapText(text, room)
	out := make([]string, 0, len(parts))
	pad := strings.Repeat(" ", displayWidth(head))
	for i, p := range parts { // bounded by the wrap (P10-02)
		if i == 0 {
			out = append(out, head+p)
			continue
		}
		out = append(out, pad+p)
	}
	return out
}

// PadBetween left+right-justifies two strings within width (ANSI-aware — key
// chips and styled cells measure by display cells, not runes).
func PadBetween(left, right string, width int) string {
	gap := width - displayWidth(left) - displayWidth(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// Width is the ANSI-aware display width of s (views size compact rows).
func Width(s string) int { return displayWidth(s) }

// PadTo right-pads a line to exactly width display cells (ANSI-aware).
// Exported for the recent section's scroll rail: PadBetween's minimum-1 gap
// pushed the rail glyph right on rows that already filled the row length
// (UAT 6.6 off-by-one).
func PadTo(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-displayWidth(s)))
}

// HumanBytes renders a byte count for a narrow column, at most six cells:
// "0B", "512B", "12.3K", "4.5M", "1023M", "1.2G" (diagnostics rows).
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	v, suffix := float64(n)/unit, "K"
	for _, next := range []string{"M", "G", "T"} {
		if v < unit {
			break
		}
		v, suffix = v/unit, next
	}
	if v < 100 {
		return fmt.Sprintf("%.1f%s", v, suffix)
	}
	return fmt.Sprintf("%.0f%s", v, suffix)
}

// stripANSI removes SGR sequences (width math + tests).
func stripANSI(s string) string { return plaintext.StripSGR(s) }

// Plain is the boundary for text that arrives from outside (plaintext.Text —
// one owner, shared with the snapshot assembler).
func Plain(s string) string { return plaintext.Text(s) }

// TruncateCells cuts s to at most n display cells (a wide rune counts two),
// with no ellipsis — the one owner of "cut to fit" for a row that must not
// overflow (R5-C-10).
//
// IT MEASURES WHAT `Width` MEASURES. An SGR sequence occupies no cells, is
// never cut through, and a span still open at the cut is CLOSED — the three
// things that make a cut safe on styled text.
//
// IT DID NOT, AND THAT IS THE UAT DEFECT OF 2026-09-10 (HUM LEAD): the console
// clamps every row of its frame to the terminal's width through here, the
// masthead's wordmark carries a truecolor escape PER RUNE, and the escapes were
// counted as content. A 150-cell row was cut after ten visible characters and
// through the middle of an escape — so the masthead read "WATCHPOS", lost its
// border, its `Updated:` stamp and its API summary, printed the escape's tail as
// text, and left a span open that painted the padding a colour that MOVED as the
// terminal resized. One measure, six symptoms.
//
// `Width` has stripped ANSI since it was written. Two measures of the same
// quantity disagreeing is what "one canonical way to do a thing" forbids, and
// the disagreement was even NOTED at `status_table.go` and worked around there
// rather than fixed here — a comment is not a fix.
func TruncateCells(s string, n int) string {
	if n <= 0 {
		return ""
	}
	// ONE COUNTING PASS FIRST, so the common case — a row that already fits —
	// returns the string ITSELF and allocates nothing, which is what this did
	// before it learned about escapes. `open` remembers whether the last
	// sequence seen was a reset, and it is read only if the cut happens.
	cut, cells, escAt, open := -1, 0, -1, false
	for i, r := range s {
		if escAt >= 0 { // inside a sequence: it ends at its terminator
			if r == 'm' {
				open = s[escAt:i+1] != sgrReset
				escAt = -1
			}
			continue
		}
		if r == 0x1b {
			escAt = i
			continue
		}
		w := RuneCells(r)
		if cells+w > n {
			cut = i
			break
		}
		cells += w
	}
	if cut < 0 {
		return s
	}
	// CLOSE WHAT WAS OPENED, because the caller pads after cutting and an open
	// span paints the padding.
	if open {
		return s[:cut] + sgrReset
	}
	return s[:cut]
}

// sgrReset ends a span. Spelled once so a truncation and a tint cannot disagree
// about what "off" is.
const sgrReset = "\x1b[0m"

// SpliceCells replaces the display columns [col, col+Width(patch)) of s with
// patch, keeping every escape sequence s carries outside that span and never
// changing the row's width.
//
// THE SAME MEASURE AS `Width` AND `TruncateCells` (D-66). A splice by rune index
// counts an escape's characters as columns and overwrites the escapes it lands
// on — which is how the priority overlay truncated the card underneath it the
// first time a takeover was drawn over a real one (HUM LEAD, UAT 2026-09-10).
//
// THE BASE'S ESCAPES INSIDE THE SPAN ARE KEPT, cells and all discarded. They
// cost nothing to draw and they leave the tone AFTER the span exactly as the row
// intended it — the alternative is guessing what to restore, and a splice that
// guesses is a splice that recolours a row it was only meant to cover.
func SpliceCells(s, patch string, col int) string {
	w := Width(patch)
	if col < 0 || w == 0 {
		return s
	}
	room := Width(s) - col
	if room <= 0 {
		return s // the row ends before the span begins: nothing to cover
	}
	if w > room {
		patch, w = TruncateCells(patch, room), room
	}
	var b strings.Builder
	cells, inEscape, written := 0, false, false
	for _, r := range s { // one pass over the runes, like splitCells
		switch {
		case inEscape:
			b.WriteRune(r)
			inEscape = r != 'm'
			continue
		case r == 0x1b:
			b.WriteRune(r)
			inEscape = true
			continue
		}
		cw := RuneCells(r)
		switch {
		case cells+cw <= col, cells >= col+w:
			b.WriteRune(r)
		case !written:
			// THE PATCH ARRIVES ON A CLEAN SLATE and leaves one, so the tone the
			// row was carrying cannot bleed into it or out of it.
			b.WriteString(sgrReset + patch + sgrReset)
			written = true
		}
		cells += cw
	}
	return b.String()
}

// RuneCells is one rune's display width (a wide rune is two) — the per-rune
// step of a cell-bounded window, with no string built per rune.
func RuneCells(r rune) int { return runewidth.RuneWidth(r) }

// PlainLine is Plain for a field that must stay on ONE line (plaintext.Line).
func PlainLine(s string) string { return plaintext.Line(s) }

// displayWidth measures terminal cells (runewidth; AI-9 glyph policy).
func displayWidth(s string) int { return runewidth.StringWidth(stripANSI(s)) }

// truncate hard-limits a line to the given display width.
func truncate(s string, w int) string {
	if w <= 0 || displayWidth(s) <= w {
		return s
	}
	return runewidth.Truncate(stripANSI(s), w, "…")
}

// FirstFit returns the first form that fits room cells, else the last (the
// narrowest) — the one shape of every "widest form that fits" ladder.
func FirstFit(room int, forms ...string) string {
	for _, f := range forms {
		if Width(f) <= room {
			return f
		}
	}
	return forms[len(forms)-1]
}
