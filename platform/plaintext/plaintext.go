// Package plaintext is the boundary for text that arrives from outside —
// provider prose, relay titles, a config file's labels: escape sequences and
// control characters are dropped so nothing a server or a file sends can
// address the terminal (red-team 0.9.0 S-F6, NFR-6). A leaf: the snapshot
// assembler cleans labels here once, and render delegates its Plain/PlainLine
// to the same code (REVIEW R5-C-05 — one owner, below both).
package plaintext

import (
	"regexp"
	"strings"
	"unicode"
)

var sgrRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripSGR removes SGR sequences (width math + tests).
func StripSGR(s string) string { return sgrRe.ReplaceAllString(s, "") }

// Text is the boundary for text that arrives from outside — relay titles
// and names, provider headlines and product text (red-team 0.9.0 S-F6):
// escape sequences and control characters are dropped so nothing a server
// sends can address the terminal (OSC hyperlinks, clipboard writes). Tabs
// and newlines survive; everything else below 0x20, and 0x7f–0x9f, goes.
func Text(s string) string {
	s = StripSGR(s)
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !dropped(r) {
			return r
		}
		return -1
	}, s)
	return capCombining(s)
}

// Line is Text for a field that must stay on ONE line — a table cell, a
// title, a source name: the newline and tab Plain keeps (prose needs them)
// collapse to a space, so a provider's line break can never split a row
// (R3-B-04).
func Line(s string) string {
	return strings.Join(strings.Fields(Text(s)), " ")
}

// maxCombining bounds a run of combining marks: two is every real accent
// stack; a hundred is a glyph-bomb that stacks off the row (REVIEW R5-C-14).
const maxCombining = 2

// capCombining drops the combining marks past maxCombining in a row. One
// pass, bounded by the runes (P10-02).
func capCombining(s string) string {
	run := 0
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			run++
			if run > maxCombining {
				return -1
			}
			return r
		}
		run = 0
		return r
	}, s)
}

// dropped reports a rune plain text never carries: C0/C1 controls, the
// variation selectors (a terminal may draw "⚠️" as one cell or two — A11-8,
// L5-F12), bidi overrides (they reverse a row on screen — spoofing) and the
// zero-width runes that hide in one (REVIEW R5-C-14).
func dropped(r rune) bool {
	for _, span := range droppedSpans() {
		if r >= span[0] && r <= span[1] {
			return true
		}
	}
	return false
}

// droppedSpans are the inclusive rune ranges dropped. A function, not a
// global (P10-06).
func droppedSpans() [][2]rune {
	return [][2]rune{{0, 0x1f}, {0x7f, 0x9f}, {0xFE0E, 0xFE0F}, {0x202A, 0x202E}, {0x2066, 0x2069}, {0x200B, 0x200F}, {0x2060, 0x2060}, {0xFEFF, 0xFEFF}}
}
