package render

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
	"testing"
)

func TestPadBetweenIsANSIAware(t *testing.T) {
	line := PadBetween("\x1b[1mleft\x1b[0m", "right", 20)
	if displayWidth(line) != 20 {
		t.Fatalf("padded width = %d, want 20: %q", displayWidth(line), line)
	}
}

func TestWrapLinesPreservesIndent(t *testing.T) {
	// UAT 25: the modal component's wrap guarantee — over-wide lines wrap
	// with their indent, blanks pass through, nothing truncates.
	out := WrapLines([]string{
		"",
		"  short",
		"    a very long indented diagnostic line that certainly exceeds the width budget",
	}, 30)
	if len(out) != 6 {
		t.Fatalf("want 6 lines after wrapping, got %d: %q", len(out), out)
	}
	for i, l := range out[2:] {
		if !strings.HasPrefix(l, "    ") {
			t.Fatalf("continuation %d must keep the indent: %q", i, l)
		}
		if displayWidth(l) > 30 {
			t.Fatalf("wrapped line still over-wide: %q", l)
		}
	}
	if strings.Contains(strings.Join(out, " "), "…") {
		t.Fatal("wrap must never truncate")
	}
}

// Quality pass Q0: the byte column of the [S] REQUESTS rows never exceeds
// six cells, whatever the count.
func TestHumanBytesFitsSixCells(t *testing.T) {
	for n, want := range map[int64]string{0: "0B", 1023: "1023B", 1024: "1.0K", 12_900_000: "12.3M", 1023 << 20: "1023M", 1 << 30: "1.0G"} {
		if got := HumanBytes(n); got != want || len(got) > 6 {
			t.Fatalf("HumanBytes(%d) = %q, want %q (≤ 6 cells)", n, got, want)
		}
	}
}

// Quality pass Q6 (L3-F8, L3-F12, L3-F13).
func TestThousandsControlsAndWrapSegmentsRows(t *testing.T) {
	for v, want := range map[float64]string{0: "0", 999: "999", 1000: "1,000", 12915: "12,915", 1234567: "1,234,567", 42.6: "43"} {
		if got := Thousands(v); got != want {
			t.Errorf("Thousands(%v) = %q, want %q", v, got, want)
		}
	}
	o := Opts{Width: 80}
	if got, want := o.Controls("   ", Ctl("esc", "Close"), Ctl("↑↓", "Scroll")), o.KeyCap("esc")+" Close   "+o.KeyCap("↑↓")+" Scroll"; got != want {
		t.Errorf("Controls: %q, want %q", got, want)
	}
	if got, want := o.Controls("  ", CtlIf("enter", "Add", false)), o.KeyCapIf("enter", false)+" Add"; got != want {
		t.Errorf("a muted control: %q, want %q", got, want)
	}
	rows := WrapSegments([]string{"aaaa", "bbbb", "cccc"}, 10, "  ")
	if len(rows) != 2 || rows[0] != "aaaa  bbbb" || rows[1] != "cccc" {
		t.Errorf("rows: %q", rows)
	}
}

// Bidi overrides and zero-width runes never reach the frame (REVIEW
// R5-C-14), and TruncateCells cuts by display cells (R5-C-10).
func TestPlainDropsBidiAndZeroWidthAndTruncateCellsCountsCells(t *testing.T) {
	if got := Plain("a\u202Eb\u200Bc\u2066d\uFEFFe"); got != "abcde" {
		t.Fatalf("plain: %q", got)
	}
	if got := Plain("e\u0301\u0301\u0301\u0301x"); got != "e\u0301\u0301x" { // a combining run is capped at two (R5-C-14)
		t.Fatalf("plain: %q", got)
	}
	if got := TruncateCells("日本語テキスト", 5); got != "日本" {
		t.Fatalf("two wide runes fit five cells: %q", got)
	}
	if got := TruncateCells("abc", 10); got != "abc" {
		t.Fatalf("short stays: %q", got)
	}
}

// A CUT NEVER LANDS INSIDE AN ESCAPE SEQUENCE, and never counts one as content.
//
// THE UAT DEFECT THIS IS BUILT FROM (HUM LEAD, 2026-09-10): the console's
// masthead read "WATCHPOS" and then stopped — no top border, no `Updated:`, no
// API summary — and its colours CHANGED AS THE TERMINAL WAS RESIZED. One cause
// for all of it: the frame clamps every row to the terminal's width through
// this function, the wordmark carries a truecolor escape per rune, and the
// escapes were counted as cells. So a 150-cell row was cut after ten visible
// characters, THROUGH the middle of an escape — which the terminal then printed
// as text and left the span open, so the tone bled and moved with the width.
//
// `Width` has always stripped ANSI. This is the other half of the same measure,
// and the two disagreeing is what "one canonical way to do a thing" forbids.
func TestTruncateCellsDoesNotCountOrCutEscapes(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	styled := Tint("abcdef", "31")
	if Width(styled) != 6 {
		t.Fatalf("precondition: Width already strips ANSI, got %d", Width(styled))
	}
	got := TruncateCells(styled, 6)
	if StripSGRForTest(got) != "abcdef" {
		t.Errorf("a row that already fits is not cut: %q", StripSGRForTest(got))
	}
	got = TruncateCells(styled, 3)
	if StripSGRForTest(got) != "abc" {
		t.Errorf("cut by CELLS, not bytes: %q", StripSGRForTest(got))
	}
	if strings.Contains(StripSGRForTest(got), "\x1b") {
		t.Errorf("an escape survived the strip, so the cut landed inside one: %q", got)
	}
	// AND IT CLOSES WHAT IT OPENED. The frame pads to width after cutting, so a
	// span left open paints the padding — which is the colour bleed the HUM LEAD
	// saw travel as the window resized.
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("a cut inside a coloured span closes it: %q", got)
	}
}

// A WORD WIDER THAN THE LINE IS BROKEN, NOT LEFT TO OVERFLOW (HUM LEAD, UAT
// 2026-08-30).
//
// This function's contract is that a floating window wraps and never truncates,
// so callers cannot reintroduce that class of bug. It held only for prose: a
// provider error carrying a 200-character URL — no spaces in it anywhere — came
// out as one over-wide line and the panel cut it, losing the half somebody would
// need to act on it.
func TestWrapBreaksAWordWiderThanTheLine(t *testing.T) {
	url := "https://api.weather.gov/alerts/active?status=actual&zone=CAC017%2CCAC027%2CCAC065%2CCAC073%2CCAZ043"
	got := WrapText("latest: "+url+" kept failing (last HTTP 502)", 40)
	for i, l := range got {
		if Width(l) > 40 {
			t.Errorf("line %d is %d cells: %q", i, Width(l), l)
		}
	}
	// EVERY character survives: the point is that nothing is lost.
	var joined string
	for _, l := range got {
		joined += l
	}
	if !strings.Contains(strings.ReplaceAll(joined, " ", ""), strings.ReplaceAll(url, " ", "")) {
		t.Errorf("the URL must survive the break intact:\n%v", got)
	}
	// The words AFTER the broken one keep flowing rather than each taking a line.
	if last := got[len(got)-1]; !strings.Contains(last, "502") {
		t.Errorf("the tail keeps collecting: %q", last)
	}
}

// An escape sequence is never split: half an SGR code on each line is read by
// the terminal as text.
func TestWrapNeverBreaksInsideAnEscape(t *testing.T) {
	tinted := Tint("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Tok(TextBright))
	for _, l := range WrapText(tinted, 12) {
		if n := strings.Count(l, "\x1b"); n > 0 {
			for _, part := range strings.Split(l, "\x1b")[1:] {
				if !strings.Contains(part, "m") {
					t.Errorf("a half escape survived: %q", l)
				}
			}
		}
		if Width(l) > 12 {
			t.Errorf("%d cells: %q", Width(l), l)
		}
	}
}
