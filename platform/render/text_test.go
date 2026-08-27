package render

import (
	"strings"
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
