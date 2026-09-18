package render

// pool_columns_test.go — what the pool gives up when it runs out of room, and in
// what order (D-106).

import (
	"strings"
	"testing"
)

func poolAt(t *testing.T, w int) []string {
	t.Helper()
	d := 5.0
	return strings.Split(Opts{ASCII: true}.PoolTable([]LocationRow{
		{Index: 1, Name: "Oceanside, CA", Zip: "92057", Station: "KOKB", StationKM: &d,
			Population: 1200000, Conditions: "CLEAR"},
	}, w), "\n")
}

// WX STN GOES FIRST (HUM LEAD, UAT 2026-09-12: "WX STN should be the first col to
// get hidden if something doesnt fit").
//
// NOT ZIP, and the difference is what the operator is doing: the pool's job is
// deciding whether a place is worth scheduling, and which observing station
// reported it is the least of what that takes. A postcode identifies the place.
func TestThePoolGivesUpWxStnBeforeZip(t *testing.T) {
	got := strings.Join(poolAt(t, 143), "\n")
	if strings.Contains(got, "WX STN") {
		t.Errorf("at 143 cells the pool still carries WX STN:\n%s", got)
	}
	if !strings.Contains(got, "ZIP") || !strings.Contains(got, "92057") {
		t.Errorf("and it kept ZIP, which identifies the place:\n%s", got)
	}
	// AND A WIDE ENOUGH TABLE KEEPS BOTH, or this proves only that the column can
	// be dropped and not that it is dropped for a reason.
	wide := strings.Join(poolAt(t, 170), "\n")
	if !strings.Contains(wide, "WX STN") || !strings.Contains(wide, "ZIP") {
		t.Errorf("at 170 cells the pool has room for both:\n%s", wide)
	}
}

// POPULATION SETS AGAINST ITS OWN WORD (HUM LEAD, UAT 2026-09-12: "right aligned
// lining up with 'N' in population").
//
// THE CATEGORY GUTTER LIVES IN THAT COLUMN'S WIDTH (D-106), so a right-ALIGNED
// column would set the digits against the far side of the gutter — three cells
// out from under the heading they belong to.
func TestPopulationSetsUnderItsOwnHeading(t *testing.T) {
	rows := poolAt(t, 143)
	var head, data string
	for _, l := range rows { // bounded by the table (P10-02)
		if strings.Contains(l, "POPULATION") {
			head = l
		}
		if strings.Contains(l, "1,200,000") {
			data = l
		}
	}
	if head == "" || data == "" {
		t.Fatalf("no POPULATION column to measure:\n%s", strings.Join(rows, "\n"))
	}
	if h, d := strings.Index(head, "POPULATION")+len("POPULATION"), strings.Index(data, "1,200,000")+len("1,200,000"); h != d {
		t.Errorf("POPULATION ends at %d and its digits at %d; they set against one edge", h, d)
	}
}

// AND THE FIVE-CELL CATEGORY GUTTER IS WHAT THE BANDS MEET IN (D-106).
func TestTheCategoryGutterIsFiveCells(t *testing.T) {
	rows := poolAt(t, 143)
	var head, data string
	for _, l := range rows { // bounded by the table (P10-02)
		if strings.Contains(l, "POPULATION") {
			head = l
		}
		if strings.Contains(l, "1,200,000") {
			data = l
		}
	}
	pop := strings.Index(data, "1,200,000") + len("1,200,000")
	cond := strings.Index(data, "CLEAR")
	if got := cond - pop; got != tableCatExtra+tableGutter {
		t.Errorf("the categories are %d cells apart; the spec says %d", got, tableCatExtra+tableGutter)
	}
	if !strings.Contains(head, "CONDITIONS") {
		t.Errorf("no CONDITIONS heading to sit over it:\n%s", head)
	}
}
