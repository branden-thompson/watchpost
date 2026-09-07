// Package plaintext is the boundary for text that arrives from outside —
// provider prose, relay titles, a config file's labels: escape sequences and
// control characters are dropped so nothing a server or a file sends can
// address the terminal (red-team 0.9.0 S-F6, NFR-6). A leaf: the snapshot
// assembler cleans labels here once, and render delegates its Plain/PlainLine
// to the same code (REVIEW R5-C-05 — one owner, below both).
package plaintext

import (
	"strings"
	"unicode"
)

// StripSGR removes SGR sequences (width math + tests).
//
// A SCANNER, NOT A REGEXP, and the fast path returns the input itself.
//
// This sits under render.Width, which measures every cell of every row of every
// frame; since 0.12.0's ticker the frame draws continuously, so it runs about
// three times a second for as long as the app is up. Regexp.ReplaceAllString
// builds a fresh string EVEN WHEN NOTHING MATCHES, and almost every cell
// measured is plain text — ≈3.2 MB/min of copies of strings that had nothing to
// strip (perf pass, 2026-08-30; flagged as item 26 of the L4 caching lens and
// left to the render lens, which never picked it up).
//
// TestStripSGRMatchesTheRegexpItReplaced holds this byte-identical to
// `\x1b\[[0-9;]*m` over a corpus and 20,000 random strings built from the
// alphabet that can form and malform a sequence, so the grammar below is pinned
// rather than described: ESC, '[', digits and semicolons, 'm'. Anything else —
// a truncated sequence at the end of the string, an OSC hyperlink, a CSI with
// another final byte — is left exactly where it is.
func StripSGR(s string) string {
	i := strings.IndexByte(s, escByte)
	if i < 0 {
		return s // the overwhelmingly common case: no escape, no copy
	}
	var b strings.Builder
	b.Grow(len(s))
	b.WriteString(s[:i])
	// BOUNDED BY THE STRING (P10-02): every pass consumes at least one byte, so
	// there can be at most len(s) of them; the loop counter is the proof and the
	// guard is what stops it.
	for range len(s) {
		if i >= len(s) {
			break
		}
		if n := sgrLen(s[i:]); n > 0 {
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	// POSTCONDITION: stripping only ever removes bytes. A longer result would
	// mean the scan had gone backwards, and the input is the safe answer —
	// text that keeps an escape is a cosmetic fault; text that grows without
	// bound on the render path is not.
	if b.Len() > len(s) {
		return s
	}
	return b.String()
}

// escByte is ESC, the byte every sequence starts with.
const escByte = 0x1b

// sgrLen is the length of the SGR sequence at the head of s, or 0 when there is
// not a complete one there.
func sgrLen(s string) int {
	if len(s) < 3 || s[0] != escByte || s[1] != '[' {
		return 0 // the shortest possible sequence is ESC [ m
	}
	for i := 2; i < len(s); i++ { // bounded by the string (P10-02)
		switch c := s[i]; {
		case c == 'm':
			if i+1 > len(s) {
				return 0 // a length past the end would slice out of range upstream
			}
			return i + 1
		case c >= '0' && c <= '9', c == ';':
		default:
			return 0 // another CSI final byte, or a stray: not ours to remove
		}
	}
	return 0 // ran off the end mid-sequence
}

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

// The bounds on text that arrives from outside, and the one place they are set.
//
// THEY WERE SET TWICE. domains/globalfeed and domains/weather/nws each declared
// maxListLen = 50 and maxFieldRunes = 120, and each carried its own pair of
// helpers to apply them — clampSlice/clampField and clampList/clampRunes. Four
// names, two declarations, one policy. They agreed, which is exactly the state
// issue #7 was in before it did not: a limit raised in one feed and not the
// other would sanitise two providers' prose to different lengths, silently.
//
// This package already says why that belongs here: it is the boundary for text
// from outside, and a leaf, so "one owner, below both" (R5-C-05).
const (
	// MaxFieldRunes bounds one field of provider prose.
	MaxFieldRunes = 120
	// MaxListLen bounds how many of them are kept.
	MaxListLen = 50
)

// ClampRunes truncates to n runes, never bytes: cutting a multi-byte rune in
// half produces invalid UTF-8, which is a different kind of unsafe.
func ClampRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// ClampField bounds one field at MaxFieldRunes.
func ClampField(s string) string { return ClampRunes(s, MaxFieldRunes) }

// ClampList bounds a list at MaxListLen and every field in it at
// MaxFieldRunes, returning a new slice.
func ClampList(s []string) []string {
	if len(s) > MaxListLen {
		s = s[:MaxListLen]
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = ClampField(v)
	}
	return out
}
