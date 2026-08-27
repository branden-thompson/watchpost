package render

// text.go — plain text: wrapping, padding, display width, ANSI stripping, byte formatting. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"regexp"
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

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR sequences (width math + tests).
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// Plain is the boundary for text that arrives from outside — relay titles
// and names, provider headlines and product text (red-team 0.9.0 S-F6):
// escape sequences and control characters are dropped so nothing a server
// sends can address the terminal (OSC hyperlinks, clipboard writes). Tabs
// and newlines survive; everything else below 0x20, and 0x7f–0x9f, goes.
func Plain(s string) string {
	s = stripANSI(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		if r == 0xFE0E || r == 0xFE0F {
			return -1 // variation selectors: a terminal may draw "⚠️" as one cell or two; plain text carries neither (A11-8, L5-F12)
		}
		return r
	}, s)
}

// displayWidth measures terminal cells (runewidth; AI-9 glyph policy).
func displayWidth(s string) int { return runewidth.StringWidth(stripANSI(s)) }

// truncate hard-limits a line to the given display width.
func truncate(s string, w int) string {
	if w <= 0 || displayWidth(s) <= w {
		return s
	}
	return runewidth.Truncate(stripANSI(s), w, "…")
}
