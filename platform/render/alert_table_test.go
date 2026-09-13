package render

// alert_table_test.go — the takeover's table against the reference (D-103).

import (
	"strings"
	"testing"
)

// alertLines is the table split, for reading offsets off it.
func alertLines(t *testing.T, rows []AlertRow, w int) []string {
	t.Helper()
	out := strings.Split(Opts{ASCII: true}.AlertTable(rows, w), "\n")
	if len(out) != len(rows)+1 {
		t.Fatalf("got %d lines for %d rows and a header:\n%s", len(out), len(rows), strings.Join(out, "\n"))
	}
	return out
}

// THE REFERENCE'S OFFSETS, TO THE CELL (HUM LEAD, mock-broadcaster-v3):
//
//	##. ALERT TYPE               LOCATION
//	01. SEV. T.STORM WARNING     Long LocationName, CA
//
// A TABLE THAT MERELY HAS THREE COLUMNS IS NOT THE MOCK. The offsets are what
// makes the two boxes on this row scan as one instrument, and they are the thing
// a width change silently moves — so they are asserted as NUMBERS rather than as
// "contains the words".
func TestTheAlertTableSitsOnTheReferencesOffsets(t *testing.T) {
	rows := []AlertRow{{"01.", "SEV. T.STORM WARNING", "Long LocationName, CA"}}
	got := alertLines(t, rows, 51)

	for _, tc := range []struct {
		line int
		text string
		at   int
	}{
		{0, "##.", 0}, {0, "ALERT TYPE", 4}, {0, "LOCATION", 29},
		{1, "01.", 0}, {1, "SEV. T.STORM WARNING", 4}, {1, "Long LocationName, CA", 29},
	} {
		if at := strings.Index(got[tc.line], tc.text); at != tc.at {
			t.Errorf("%q sits at %d, not %d:\n%s", tc.text, at, tc.at, strings.Join(got, "\n"))
		}
	}
}

// AN EMPTY SLOT IS STILL A SLOT. The reference numbers ten rows whether or not
// ten hazards have been declared, because a row is an ADDRESS — and a box that
// listed only what had arrived would change height as hazards did.
func TestAnUnfilledAlertRowKeepsItsNumber(t *testing.T) {
	got := alertLines(t, []AlertRow{{Num: "07."}}, 51)
	if strings.TrimSpace(got[1]) != "07." {
		t.Errorf("an empty slot drew %q; it draws its number and nothing else", got[1])
	}
}

// THE LOCATION COLUMN GIVES WAY, as it does on every other table here: a place
// name is the one cell whose length nobody controls.
func TestTheAlertTablesLocationTakesTheSlack(t *testing.T) {
	rows := []AlertRow{{"01.", "T.STORM WATCH", "Rancho Penasquitos, CA"}}
	wide := alertLines(t, rows, 80)
	if !strings.Contains(wide[1], "Rancho Penasquitos, CA") {
		t.Errorf("the name was cut at 80 cells:\n%s", strings.Join(wide, "\n"))
	}
	narrow := alertLines(t, rows, 44)
	if strings.Contains(narrow[1], "Rancho Penasquitos, CA") {
		t.Errorf("nothing gave way at 44 cells:\n%s", strings.Join(narrow, "\n"))
	}
	// AND THE COLUMNS IN FRONT OF IT DO NOT MOVE while it does.
	if at := strings.Index(narrow[1], "T.STORM WATCH"); at != 4 {
		t.Errorf("ALERT TYPE moved to %d when LOCATION gave way", at)
	}
}
