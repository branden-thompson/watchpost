package plaintext

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

// StripSGR must be BYTE-IDENTICAL to the regexp it replaced, on everything.
//
// It stopped being a regexp because it sits under render.Width, which measures
// every cell of every row of every frame — and Regexp.ReplaceAllString builds a
// fresh string even when nothing matches. Since 0.12.0's ticker the frame draws
// continuously, so that ran ~3.3 times a second forever: ≈3.2 MB/min plus its
// share of the builder growth (perf pass, 2026-08-30; the L4 lens flagged it as
// item 26 and left it to the render lens).
//
// The regexp is rebuilt HERE, in the test, so the two implementations are
// compared rather than the new one being compared to itself.
func TestStripSGRMatchesTheRegexpItReplaced(t *testing.T) {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	const esc = "\x1b"
	corpus := []string{
		"", "plain text", "no escapes at all — just prose, °F and ✔",
		esc + "[0m", esc + "[m", esc + "[1;38;2;255;0;0mred" + esc + "[0m",
		"lead" + esc + "[31m" + "mid" + esc + "[0m" + "tail",
		esc + "[", esc + "[999", esc, esc + esc, esc + "[0", // truncated: nothing to strip
		esc + "]8;;http://example.com" + esc + "\\link", // OSC: not SGR, must survive
		esc + "[2J", esc + "[1A", esc + "[38;5;242K", // other CSI finals: must survive
		esc + "[0;m" + esc + "[;;m", esc + "[m" + esc + "[m" + esc + "[m",
		esc + "[31mAB" + esc + "[m" + esc + "[32mCD",
		"✔ " + esc + "[38;2;190;25;25mapi.weather.gov" + esc + "[0m", // a real status cell
		esc + "[1m" + "wide 漢字 and combining é" + esc + "[0m",
		"a" + esc + "[3" + "m", "trailing esc" + esc,
	}
	for _, s := range corpus {
		if got, want := StripSGR(s), re.ReplaceAllString(s, ""); got != want {
			t.Errorf("StripSGR(%q) = %q, the regexp gives %q", s, got, want)
		}
	}
	// And randomly, over the alphabet that can form and malform a sequence.
	alphabet := []string{esc, "[", "]", "m", "0", "1", ";", "9", "K", "A", "x", " ", "漢"}
	rnd := rand.New(rand.NewSource(20260830))
	for i := 0; i < 20000; i++ {
		var b strings.Builder
		for n := rnd.Intn(12); n > 0; n-- {
			b.WriteString(alphabet[rnd.Intn(len(alphabet))])
		}
		s := b.String()
		if got, want := StripSGR(s), re.ReplaceAllString(s, ""); got != want {
			t.Fatalf("StripSGR(%q) = %q, the regexp gives %q", s, got, want)
		}
	}
}

// The fast path returns the INPUT STRING ITSELF when there is no escape at all
// — not a copy. That is the whole point: almost every cell measured is plain.
func TestStripSGRDoesNotCopyPlainText(t *testing.T) {
	s := strings.Repeat("api.tidesandcurrents.noaa.gov ", 8)
	if n := testing.AllocsPerRun(200, func() { _ = StripSGR(s) }); n != 0 {
		t.Errorf("plain text cost %v allocations, want 0", n)
	}
}
