package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
)

// scrollWindowShipped is scrollWindow as 0.19.3 shipped it: the reference the
// walk in place is held to, byte for byte.
func scrollWindowShipped(text string, offset, width int, gap string) string {
	if width <= 0 {
		return ""
	}
	if render.Width(text) <= width {
		return render.PadTo(text, width)
	}
	loop := []rune(text + gap)
	n := len(loop)
	off := ((offset % n) + n) % n
	var b strings.Builder
	cells := 0
	for i := 0; i < n && cells < width; i++ {
		r := loop[(off+i)%n]
		w := render.RuneCells(r)
		if cells+w > width {
			break
		}
		b.WriteRune(r)
		cells += w
	}
	return render.PadTo(b.String(), width)
}

// TestTheMarqueeWindowIsByteIdentical is 0.19.5 Q3.5 (PF-5): the marquee walks
// the tape in place instead of copying it into a rune slice every frame, and
// draws the same bytes at every offset, width and gap, with wide runes and
// invalid UTF-8 in the tape.
func TestTheMarqueeWindowIsByteIdentical(t *testing.T) {
	texts := []string{
		"Severe Thunderstorm Warning · Benchmark County 01, KS  issued 12:01 AM   •   Tornado 東京 Warning é",
		"short",
		"東京東京東京東京東京",
		"a\xffb\xfe broken utf8 that is long enough to scroll around",
	}
	for _, tx := range texts {
		for _, gap := range []string{tickerBulletDot, tickerBulletASCII, ""} {
			for _, w := range []int{0, 1, 2, 5, 10, 40, 120} {
				for off := -3; off < 260; off++ {
					if got, want := scrollWindow(tx, off, w, gap), scrollWindowShipped(tx, off, w, gap); got != want {
						t.Fatalf("%q at %d, width %d, gap %q: %q; the shipped window draws %q", tx, off, w, gap, got, want)
					}
				}
			}
		}
	}
}
