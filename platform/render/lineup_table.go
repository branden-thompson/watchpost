package render

// lineup_table.go — the Broadcaster's running order, as a table (D-94).
//
// HUM LEAD, 2026-09-12, after UAT: the console's line-up becomes a table matching
// Observer's, so that "the experiences are more consistent and the user doesn't
// have to relearn what certain things mean in between experiences".
//
// IT LIVES BESIDE THE LOCATION TABLE, in the one file's package, because
// `table.go` is "the ONLY go-studs consumer in the app" and a second consumer
// somewhere else would end that. What the two share — the marks block, the header
// tone, the group-header shape — is shared HERE rather than copied.
//
// WHAT IT IS NOT: a second answer to the running order. The console draws what
// the Director published; every cell below is a fact the schedule already holds
// or a join the console makes against the pool it was told (D-93).

import (
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// LineupRow is one slot of the running order, as the operator reads it.
//
// THE COLUMNS ARE THE MOCK'S, and they are grouped the way it groups them: what
// will be READ, how the Director is DIRECTING it, and who PRESENTS it. A row is
// three questions, and the group headers say which is which.
type LineupRow struct {
	// Slot is the address the operator keys, and Num is what is drawn — they
	// differ because the table starts at 2 while the frame counts from 0.
	Slot int
	Num  string

	// ReportType is what KIND of read this is, from the slot registry — never
	// invented here.  D-31's finer taxonomy lands in that registry, and this
	// column needs no change when it does.
	ReportType string

	Location string
	Zip      string
	DistMi   *float64

	// Priority is the badge the card already wears; RequestedBy is its ORIGIN in
	// the operator's words.  D-86 tinted the card by origin and the HUM LEAD's v3
	// replaces that with this column — a shade implied it, a column says it.
	Priority    string
	RequestedBy string

	// Correspondent is who reads this beat.  The per-card presenter wins over the
	// role cast (2026-09-12: "In Broadcaster it's beats").
	Correspondent string

	Marks Marks
}

// lineup column widths, measured off the v3 mock.
func lineupColumns() []baseCol {
	return []baseCol{
		{"marks", "", marksW, 0},
		{"num", "##.", 5, 1},
		{"type", "REPORT TYPE", 22, 1},
		{"loc", "LOCATION", 24, 1},
		{"zip", "ZIP", 7, 1},
		{"dist", "DIST", 7, 1},
		{"prio", "PRIORITY", 11, 2},
		{"req", "REQUESTED BY", 18, 2},
		{"corr", "CORRESPONDENT", 34, 3},
	}
}

// lineupGroups names the three questions a row answers.
func lineupGroups() []groupSpec {
	return []groupSpec{
		{"R E A D   O U T S", "R E A D   O U T S", GroupLocationBG, []string{"marks", "num", "type", "loc", "zip", "dist"}},
		{"D I R E C T I O N", "D I R E C T I O N", GroupTodayBG, []string{"prio", "req"}},
		{"P R E S E N T A T I O N", "P R E S E N T A T I O N", GroupTomorrowBG, []string{"corr"}},
	}
}

// LineupTable draws the running order.
//
// THE SAME ASSEMBLY AS THE LOCATION TABLE, deliberately: our own two header rows
// over the kit's rows, `NoAutoStyle` so the THEME owns every colour (Q4a-004),
// and the trailing blanks trimmed.  A reader who knows one knows the other.
func (o Opts) LineupTable(rows []LineupRow, width int) string {
	if width <= 0 {
		return ""
	}
	cols := lineupColumnDefs(width)
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter, NoAutoStyle: true}
	for _, r := range rows { // bounded by the running order (P10-02)
		data := clampCells(o.lineupRowData(r), cols)
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: tableCellStyles(data)})
	}
	dt := studs.NewDataTable(width, def)
	groups := lineupGroups()
	out := []string{o.groupHeader(groups, cols, width), o.columnHeader(groups, cols, width)}
	for _, line := range dt.Rows() { // bounded by the rows above (P10-02)
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// lineupColumnDefs turns the spec into the kit's definitions, giving the two
// widest columns the slack when the terminal is wider than the mock.
func lineupColumnDefs(width int) []studs.ColumnDefinition {
	out := make([]studs.ColumnDefinition, 0, len(lineupColumns()))
	for _, c := range lineupColumns() { // bounded by the spec (P10-02)
		// LOCATION IS THE FILL COLUMN, as NAME is on Observer's table and for the
		// same reason: a place name is the one cell whose length nobody controls,
		// so it takes the slack and gives way first.
		if c.name == "loc" {
			out = append(out, studs.ColumnDefinition{Name: c.name, Header: c.header,
				Fill: true, MinWidth: c.width, Truncatable: true, TruncatedMinWidth: 10, Alignment: "left"})
			continue
		}
		align := "left"
		if c.name == "zip" { // an identifier of one length, not arithmetic — centred, as Observer centres it
			align = "center"
		}
		out = append(out, studs.ColumnDefinition{Name: c.name, Header: c.header, Width: c.width, Alignment: align})
	}
	return out
}

// lineupRowData formats one row into the spec's cells.
func (o Opts) lineupRowData(r LineupRow) []string {
	g := o.Glyphs()
	marks := markCells(r.Marks, g)
	return []string{
		strings.Join(marks[:], ""),
		r.Num,
		r.ReportType,
		r.Location,
		r.Zip,
		o.StationDistance(kmOf(r.DistMi)),
		r.Priority,
		r.RequestedBy,
		r.Correspondent,
	}
}

// kmOf converts the row's miles back to the kilometres StationDistance takes, so
// the DIST column reads in the operator's own units through the ONE formatter
// every other distance in the app already uses.
func kmOf(mi *float64) *float64 {
	if mi == nil {
		return nil
	}
	km := *mi / 0.621371
	return &km
}
