package tty

// clip_ansi_test.go — clipping must count CELLS, not bytes.
//
// `clipToWidth` MUST SKIP THE ESCAPE, not measure it. Walking runes and calling
// render.Width per rune, a lone \x1b is 0 but '[', '1' and 'm' each measure 1 —
// so every bold span costs eight phantom cells. Text that fits is cut anyway, and
// the cut lands mid-escape, leaving the SGR unterminated so the weight bleeds
// into whatever follows.
//
// IT IS D-138's DEFECT ONE FUNCTION LATER. `centerText` takes the clip branch
// whenever the text measures at least the width — exactly what `WrapText`
// produces on a full line — so the held-hazard band lost its tail at the widths
// where it fits most tightly. The band's own test turns colour ON and is right
// in shape; it sampled four widths and missed it.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestClippingCountsCellsAndNotEscapeBytes(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	// TWELVE VISIBLE CELLS, wearing a bold span in the middle.
	text := "ab " + render.Bold("HAZARD") + " cd"
	if got := render.Width(stripANSITest(text)); got != 12 {
		t.Fatalf("fixture is %d cells, expected 12", got)
	}
	// A WIDTH THAT FITS IT EXACTLY must lose nothing.
	if got := stripANSITest(clipToWidth(text, 12)); got != "ab HAZARD cd" {
		t.Errorf("clipping at the exact width dropped visible text: %q", got)
	}
	// AND A NARROWER ONE must keep exactly that many cells — not fewer because
	// the escapes were charged against the budget.
	for _, w := range []int{4, 8, 11} {
		if got := render.Width(stripANSITest(clipToWidth(text, w))); got != w {
			t.Errorf("clipping to %d kept %d visible cells", w, got)
		}
	}
}

// AND THE STYLING IS CLOSED. A cut that lands inside or after an open span must
// not leave the weight running into the rest of the frame.
func TestClippingNeverLeavesTheStylingOpen(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	text := "ab " + render.Bold("HAZARD") + " cd"
	for w := 1; w <= 12; w++ {
		got := clipToWidth(text, w)
		if strings.Count(got, "\x1b[1m") > strings.Count(got, "\x1b[0m") {
			t.Errorf("clipping to %d left the bold open: %q", w, got)
		}
	}
}

// AND THE BAND THAT REVEALED IT KEEPS ITS TAIL AT EVERY WIDTH, not at four
// sampled ones — a sweep, because the failure was width-dependent and the spot
// checks fell between the cracks.
func TestTheHeldBandKeepsItsTailAcrossEveryWidth(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for w := 100; w <= 220; w++ {
		b := bcStandby(t, true, 20*time.Minute)
		b.width = w
		// WHITESPACE COLLAPSED FIRST. The band CENTRES each wrapped line, so at
		// widths where the wrap falls between "dropped" and "unread." the two
		// words are separated by a run of padding — the message is intact and a
		// contiguous-substring check is not. My first sweep reported width 152
		// as truncated for exactly this reason, and the band was correct.
		got := strings.Join(strings.Fields(stripANSITest(strings.Join(b.heldNotice(), " "))), " ")
		if !strings.Contains(got, "dropped unread") {
			t.Fatalf("width %d: the severest rung is cut short:\n%q", w, got)
		}
	}
}
