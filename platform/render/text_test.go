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

// HUM LEAD UAT nit 2026-08-27: the middle text sits on the row's global
// centre whatever the side blocks measure; it shifts only to clear them.
func TestCentreBetweenUsesTheGlobalCentre(t *testing.T) {
	row := CentreBetween("ab", "MID", "rightblock", 40)
	if Width(row) != 40 || !strings.HasPrefix(row, "ab") || !strings.HasSuffix(row, "rightblock") {
		t.Fatalf("row: %q", row)
	}
	if i := strings.Index(row, "MID"); i != (40-3)/2 { // "MID" straddles the row's centre, whatever the side blocks measure
		t.Fatalf("global centre: MID starts at %d in %q", i, row)
	}
	// A wide left block pushes it right the least it must.
	row = CentreBetween(strings.Repeat("L", 18), "MID", "r", 40)
	if i := strings.Index(row, "MID"); i != 20 {
		t.Fatalf("clamped clear of the left block: %d in %q", i, row)
	}
	// No room: ellipsis, then nothing.
	if row := CentreBetween("LLLLLLLLLL", "a long middle text", "RRRRRRRRRR", 30); !strings.Contains(row, "…") || Width(row) != 30 {
		t.Fatalf("ellipsis: %q", row)
	}
	if row := CentreBetween("LLLLLLLLLLLLL", "middle", "RRRRRRRRRRRRR", 30); strings.Contains(row, "middle") || Width(row) != 30 {
		t.Fatalf("dropped: %q", row)
	}
}
