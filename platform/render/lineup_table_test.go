package render

// lineup_table_test.go — D-94: the console's running order, as a table.
//
// THE REQUIREMENT IS PARITY, not resemblance (HUM LEAD, 2026-09-12): "the
// experiences are more consistent and the user doesn't have to relearn what
// certain things mean in between experiences."  So the tests below ask whether
// the two tables agree, not whether this one looks plausible on its own.

import (
	"strings"
	"testing"
)

func lineupFixture() []LineupRow {
	mi := func(v float64) *float64 { return &v }
	return []LineupRow{
		{Slot: 2, Num: "02.", ReportType: "Location Report", Location: "Fallbrook, CA", Zip: "92028",
			DistMi: mi(5), Priority: "Standard", RequestedBy: "Producer", Correspondent: "System Voice",
			Marks: Marks{Selected: true}},
		{Slot: 3, Num: "03.", ReportType: "Location Report", Location: "Carlsbad, CA", Zip: "92008",
			DistMi: mi(10), Priority: "Priority", RequestedBy: "Station Operator", Correspondent: "Samantha",
			Marks: Marks{Playing: true, HasAlert: true, WarnAlert: true, AlertCount: 8}},
	}
}

// THE MARKS ARE OBSERVER'S OWN, through the one drawer.
//
// NOT "THEY LOOK SIMILAR" — the assertion is that the SAME mark state produces
// the SAME cells on both tables, which is the only form of this claim that can
// fail when someone edits one of them.
func TestTheLineupWearsTheSameMarksAsTheLocationTable(t *testing.T) {
	o := Opts{Width: 150}
	m := Marks{Selected: true, Playing: true, HasAlert: true, WarnAlert: true, AlertCount: 3, Fire: 2, Seismic: 1}

	loc := rowMarks(LocationRow{
		Selected: m.Selected, Playing: m.Playing, HasAlert: m.HasAlert, WarnAlert: m.WarnAlert,
		AlertCount: m.AlertCount, Fire: m.Fire, Seismic: m.Seismic,
	}, o.Glyphs())
	lin := markCells(m, o.Glyphs())

	if loc != lin {
		t.Errorf("the same marks render differently on the two tables:\n  location %q\n  line-up  %q",
			strings.Join(loc[:], ""), strings.Join(lin[:], ""))
	}
}

// THE TABLE CARRIES WHAT THE OPERATOR DECIDES ON, and the mock's three groups say
// which question each column answers.
func TestTheLineupTableDrawsItsColumnsAndGroups(t *testing.T) {
	o := Opts{Width: 150}
	got := o.LineupTable(lineupFixture(), 150)

	for _, want := range []string{
		"R E A D   O U T S", "D I R E C T I O N", "P R E S E N T A T I O N",
		"REPORT TYPE", "LOCATION", "ZIP", "DIST", "PRIORITY", "REQUESTED BY", "CORRESPONDENT",
		"02.", "Fallbrook, CA", "92028", "Station Operator", "Samantha",
	} {
		if !strings.Contains(StripSGRForTest(got), want) {
			t.Errorf("the line-up table is missing %q:\n%s", want, StripSGRForTest(got))
		}
	}
}

// THE GROUP HEADER IS THE ONE OBSERVER DRAWS, not a copy of it.
//
// `groupHeader` and `columnHeader` took a LOCATION layout until D-94 and now take
// the group spec, which is the only thing they ever used it for.  A copy would
// have drifted the first time either was touched.
func TestBothTablesDrawTheirHeadersThroughTheSameFunction(t *testing.T) {
	o := Opts{Width: 150}
	groups := lineupGroups()
	cols := lineupColumnDefs(150)

	head := StripSGRForTest(o.groupHeader(groups, cols, 150))
	if !strings.Contains(head, "R E A D   O U T S") {
		t.Fatalf("the shared group header drew nothing for the line-up: %q", head)
	}
	// AND IT STILL DRAWS OBSERVER'S — the same call, the other spec.
	l := layoutFor(150, 0)
	locHead := StripSGRForTest(o.groupHeader(groupsFor(l), l.columns(nil), 150))
	if !strings.Contains(locHead, "L O C A T I O N") {
		t.Errorf("the shared group header stopped drawing Observer's: %q", locHead)
	}
}
