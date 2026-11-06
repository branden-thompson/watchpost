package render

import (
	"math/rand/v2"
	"strings"
	"testing"

	runewidth "github.com/mattn/go-runewidth"

	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// widthShipped is displayWidth as 0.19.3 shipped it, the reference the
// segment sum is held to: strip every SGR sequence, then measure.
func widthShipped(s string) int { return runewidth.StringWidth(plaintext.StripSGR(s)) }

// widthPieces is the alphabet the random strings are built from: the pieces of
// an SGR sequence, whole and broken, and every kind of rune that can join a
// grapheme cluster across an escape once the escape is stripped, beside the
// runes the frames draw.
var widthPieces = []string{
	"\x1b[0m", "\x1b[1;38;5;214m", "\x1b[m", "\x1b[38;2;10;20;30m", "\x1b", "[", "m", "3", ";", "\x1b[1",
	"a", "Z", " ", "·", "é", "─", "▲", "▼", "█", "•", "°", "→", "☀",
	"e", "\u0301", "\u20dd", "\u200d", "\u200c", "\ufe0f", "\ufe0e", "\U0001f3fd", "\U000e0067",
	"\U0001f1fa", "\U0001f1f8", "\U0001f468", "\U0001f469", "\u2764",
	"\u1100", "\u1161", "\u11a8", "\uac00", "\ud7b0",
	"\u0600", "\u0915", "\u094d", "\u093e", "\u0d15", "\u0d4d",
	"東", "京", "\u3099", "\r", "\n", "\t", "\u00ad",
	"\xe2", "\x94", "\x80", "\xff",
}

// randomStyled is a string of n pieces drawn from widthPieces.
func randomStyled(r *rand.Rand, n int) string {
	var b strings.Builder
	for range n { // bounded by n (P10-02)
		b.WriteString(widthPieces[r.IntN(len(widthPieces))])
	}
	return b.String()
}

// TestWidthMatchesStripThenMeasure is 0.19.5 Q3.2 (PF-2, RR-17): Width sums the
// plain runs between SGR sequences instead of measuring a stripped copy, and
// gives the same answer: over the frames' own styled rows, a corpus of clusters
// split by an escape, and 200,000 random strings on a fixed seed, so a failing
// cluster fails every run and not one CI leg in a hundred.
func TestWidthMatchesStripThenMeasure(t *testing.T) {
	colourOn(t)
	corpus := []string{
		"", "plain", "\x1b[1mbold\x1b[0m", strings.Split(Opts{Width: 129}.LocationTable(benchRows(3), 0), "\n")[3],
		"e\x1b[1m\u0301", "\U0001f1fa\x1b[0m\U0001f1f8", "\U0001f468\x1b[m\u200d\U0001f469", "\U0001f468\u200d\x1b[m\U0001f469",
		"\u2764\x1b[1m\ufe0f", "\u1100\x1b[0m\u1161\x1b[0m\u11a8", "\u0600\x1b[m\u0915", "\u0d15\u0d4d\x1b[m\u0d15",
		"\U0001f44d\x1b[m\U0001f3fd", "\xe2\x1b[m\x94\x80", "\r\x1b[m\n", "e\x1b[1m\x1b[0m\u0301", "\u1100\x1b[1m\x1b[0m\u1161",
	}
	for _, s := range corpus {
		if got, want := Width(s), widthShipped(s); got != want {
			t.Errorf("Width(%q) = %d; stripped and measured it is %d", s, got, want)
		}
	}
	r := rand.New(rand.NewPCG(195, 2))
	for range 200_000 { // bounded (P10-02)
		s := randomStyled(r, 1+r.IntN(12))
		if got, want := Width(s), widthShipped(s); got != want {
			t.Fatalf("Width(%q) = %d; stripped and measured it is %d", s, got, want)
		}
	}
}

// FuzzWidthMatchesStripThenMeasure is TestWidthMatchesStripThenMeasure's rule
// for the fuzzer (RR-17).
func FuzzWidthMatchesStripThenMeasure(f *testing.F) {
	for _, s := range []string{"e\x1b[1m\u0301", "\U0001f1fa\x1b[0m\U0001f1f8", "\xe2\x1b[m\x94\x80", "\x1b[1mbold\x1b[0m ─▲"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := Width(s), widthShipped(s); got != want {
			t.Errorf("Width(%q) = %d; stripped and measured it is %d", s, got, want)
		}
	})
}

// TestWidthAllocBudget is 0.19.5 Q3.2 (PF-2) at its cost: measuring a styled
// row of the location table allocates nothing, so a change that sends the
// frames' own rows down the strip-and-measure path fails here, not only in the
// beat pin.
func TestWidthAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	colourOn(t)
	row := strings.Split(Opts{Width: 129}.LocationTable(benchRows(3), 0), "\n")[3]
	if n := testing.AllocsPerRun(100, func() { _ = Width(row) }); n != 0 {
		t.Errorf("measuring a styled row allocates %.0f times; want none", n)
	}
}
