package render

// stencil_header_test.go — `##.` hangs its numbers (D-103).
//
// HUM LEAD, UAT 2026-09-12: "'##.' column head misaligned."

import (
	"strings"
	"testing"
)

// THE STENCIL SITS EXACTLY WHERE ITS NUMBERS SIT, on both tables that have one.
//
// CENTRING PUTS IT ONE CELL RIGHT. The column-title row centres each title inside
// its band, which is right for a title that NAMES a column and wrong for one that
// is a picture of its own cells: `##.` is three cells centred in five, so every
// number beneath it hangs one cell left of it.
//
// BOTH TABLES, ONE RULE. The pool's numbers are four cells and the running
// order's are three, so a fix that merely nudged one of them would have broken
// the other — the stencil is anchored to the COLUMN, not to the digits.
func TestTheNumberStencilHangsItsNumbers(t *testing.T) {
	o := Opts{ASCII: true}
	d := 5.0
	for _, tc := range []struct {
		name, table, num string
	}{
		{"running order", o.LineupTable([]LineupRow{{Num: "02.", ReportType: "Location Report", Location: "Vista, CA"}}, 144), "02."},
		{"location pool", o.PoolTable([]LocationRow{{Index: 1, Name: "Oceanside, CA", StationKM: &d}}, 144), "001."},
	} {
		lines := strings.Split(tc.table, "\n")
		var head, row string
		for _, l := range lines { // bounded by the table (P10-02)
			if head == "" && strings.Contains(l, "##.") {
				head = l
			} else if head != "" && strings.Contains(l, tc.num) {
				row = l
				break
			}
		}
		if head == "" || row == "" {
			t.Fatalf("%s: no header or row to compare:\n%s", tc.name, tc.table)
		}
		if h, r := strings.Index(head, "##."), strings.Index(row, tc.num); h != r {
			t.Errorf("%s: the stencil sits at %d and %q at %d:\n%s\n%s", tc.name, h, tc.num, r, head, row)
		}
	}
}

// AND EVERY OTHER HEADER STAYS CENTRED. Observer's shipped tables caption their
// columns, and the finding was about the stencil alone — a fix that left-aligned
// every title would have redrawn a table nobody asked to change.
func TestTheOtherHeadersStayCentred(t *testing.T) {
	d := 5.0
	head := strings.Split(Opts{ASCII: true}.PoolTable([]LocationRow{{Index: 1, Name: "Oceanside, CA", StationKM: &d}}, 144), "\n")[3]
	if !strings.Contains(head, "[ CONDITIONS ]") {
		t.Errorf("CONDITIONS is no longer a centred caption:\n%s", head)
	}
}
