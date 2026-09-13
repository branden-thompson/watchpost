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

// THE HUM LEAD'S SPEC, TO THE CELL (2026-09-12):
//
//	[##.:3][ALERT TYPE: FILL][gutter:3][LOCATION: FIT; TRUNCATABLE]
//
// THE OFFSETS ARE NOT CONSTANTS ANY MORE, and that is the spec, not a loss: with
// ALERT TYPE filling and LOCATION fitting, where the place name starts is a
// function of the longest place name in the box. What is fixed is the SHAPE —
// the number's four cells, and the three that must always stand between the
// hazard and the place.
func TestTheAlertTableFollowsTheColumnSpec(t *testing.T) {
	rows := []AlertRow{{"01.", "SEVERE THUNDERSTORM WARNING", "Long LocationName, CA"}}
	got := alertLines(t, rows, 51)

	if at := strings.Index(got[0], "##."); at != 0 {
		t.Errorf("the number starts at %d, not 0", at)
	}
	for _, l := range got {
		if at := strings.Index(l, strings.TrimSpace(l[4:])[:1]); at != 4 {
			t.Errorf("ALERT TYPE starts at %d, not %d:\n%s", at, alertNumW+alertGap1, l)
		}
	}
	// LOCATION IS AS WIDE AS THE LONGEST PLACE NAME AND NO WIDER (FIT), and the
	// three-cell gutter stands in front of it whatever the hazard's length.
	loc := strings.Index(got[1], "Long LocationName, CA")
	if head := strings.Index(got[0], "LOCATION"); head != loc {
		t.Errorf("the LOCATION header sits at %d and its cells at %d", head, loc)
	}
	if gap := got[1][loc-alertGap2 : loc]; gap != "   " {
		t.Errorf("the hazard and the place are %q apart; the spec says three cells", gap)
	}
	if _, w := alertWidths(rows, 51); w != len("Long LocationName, CA") {
		t.Errorf("LOCATION is %d cells for a %d-cell name; FIT means neither more nor less",
			w, len("Long LocationName, CA"))
	}
}

// THE GUTTER SURVIVES A HAZARD THAT FILLS ITS COLUMN, which is the whole reason
// the HUM LEAD wrote a number on it: "SEVERE THUNDERSTORM WARN…Harper, KS" is
// what a fill column with no gutter draws.
func TestTheHazardNeverTouchesThePlace(t *testing.T) {
	rows := []AlertRow{{"01.", "A VERY LONG HAZARD NAME THAT WILL NOT FIT IN ANY COLUMN", "Harper, KS"}}
	got := alertLines(t, rows, 44)
	loc := strings.Index(got[1], "Harper, KS")
	if loc < 0 {
		t.Fatalf("the place name is missing:\n%s", strings.Join(got, "\n"))
	}
	if gap := got[1][loc-alertGap2 : loc]; gap != "   " {
		t.Errorf("a hazard that fills its column left %q before the place:\n%s", gap, got[1])
	}
}

// THE SHORTHAND IS THE REFERENCE'S, and it belongs to THIS table.
//
// HUM LEAD, 2026-09-12: "Shorthand Vocabulary was to ensure the alert type would
// fit in the window; we should keep it for that window."
func TestTheTakeoverSpeaksTheReferencesShorthand(t *testing.T) {
	for in, want := range map[string]string{
		"SEVERE THUNDERSTORM WARNING": "SEV. T.STORM WARNING",
		"THUNDERSTORM WATCH":          "T.STORM WATCH",
		"SPECIAL WEATHER ADVISORY":    "SPEC. WEATHER ADVISORY",
		// A NAME WITH NO SHORTHAND COMES OUT WHOLE. Inventing one on a safety
		// surface is worse than a long cell.
		"FLASH FLOOD WARNING": "FLASH FLOOD WARNING",
	} {
		if got := ShortHazard(in); got != want {
			t.Errorf("ShortHazard(%q) = %q, want %q", in, got, want)
		}
	}
	// AND THE TABLE APPLIES IT, so a caller cannot forget to.
	if got := alertLines(t, []AlertRow{{"01.", "SEVERE THUNDERSTORM WARNING", "Harper, KS"}}, 51); !strings.Contains(got[1], "SEV. T.STORM WARNING") {
		t.Errorf("the table drew the long form:\n%s", strings.Join(got, "\n"))
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

// THE LOCATION COLUMN GIVES WAY, and only after the hazard has its floor: an
// operator can place "Rancho Pen…" and cannot place "SEV. T.ST…".
func TestTheAlertTablesLocationGivesWayLast(t *testing.T) {
	rows := []AlertRow{{"01.", "T.STORM WATCH", "Rancho Penasquitos, CA"}}
	if !strings.Contains(alertLines(t, rows, 51)[1], "Rancho Penasquitos, CA") {
		t.Error("the name was cut at 51 cells, where it fits")
	}
	narrow := alertLines(t, rows, 30)
	if strings.Contains(narrow[1], "Rancho Penasquitos, CA") {
		t.Errorf("nothing gave way at 30 cells:\n%s", strings.Join(narrow, "\n"))
	}
	// AND THE HAZARD KEPT ITS FLOOR while the place name was cut.
	if !strings.Contains(narrow[1], "T.STORM WATCH") {
		t.Errorf("the hazard gave way before the place did:\n%s", strings.Join(narrow, "\n"))
	}
}
