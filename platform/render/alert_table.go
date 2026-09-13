package render

// alert_table.go — the takeover's contents, as a table (D-103).
//
// HUM LEAD, UAT 2026-09-12: "Alert box is not correct / doesn't match Mock when
// alerts are present (should be a go studs table)."
//
// IT WAS PROSE. `burstBody` drew the Composer's own header sentence and then each
// alert wrapped over two lines — which reads as a paragraph when what the operator
// is doing is SCANNING a list to decide whether to let it interrupt the programme.
// The reference draws two columns and ten numbered rows.

import (
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// AlertRow is one hazard the takeover would read.
type AlertRow struct {
	Num      string
	Kind     string // the hazard, in the operator's shorthand
	Location string
}

// alertColumns is the reference's two columns behind their numbers.
func alertColumns() []baseCol {
	return []baseCol{
		// THE REFERENCE'S OWN OFFSETS: `##.` and one cell, then ALERT TYPE, and
		// LOCATION at column 29 of the table. NOT Observer's five-cell number
		// column — this table has no marks block in front of it, so the number is
		// the first thing on the row and hugs the hazard it numbers.
		{"num", "##.", 4, 1},
		{"kind", "ALERT TYPE", 25, 1},
		{"loc", "LOCATION", 22, 1},
	}
}

// AlertTable draws the takeover's contents.
//
// THE SAME ASSEMBLY AS THE OTHER TWO, and no group band: this table lives INSIDE a
// box that already names it, so a band over it would be the title said twice.
func (o Opts) AlertTable(rows []AlertRow, width int) string {
	if width <= 0 {
		return ""
	}
	cols := alertColumnDefs(width)
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter, NoAutoStyle: true}
	for _, r := range rows { // bounded by the burst (P10-02)
		data := clampCells([]string{r.Num, r.Kind, r.Location}, cols)
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: tableCellStyles(data)})
	}
	dt := studs.NewDataTable(width, def)
	out := []string{o.alertHeader(cols, width)}
	for _, line := range dt.Rows() { // bounded by the rows above (P10-02)
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// alertColumnDefs is the spec, with LOCATION taking the slack.
//
// LOCATION IS SIZED HERE, NOT LEFT TO FILL. A fill column's floor is a MINIMUM,
// and the kit honours a minimum by drawing past the width rather than by cutting
// the cell — which put a long place name through the right wall of a box that is
// only about fifty cells wide to begin with. The same arithmetic the location
// table already does for NAME (`l.nameMin`), done here for the same reason: the
// column that gives way has to be TOLD what it has.
func alertColumnDefs(width int) []studs.ColumnDefinition {
	base := alertColumns()
	fixed := 0
	for _, c := range base { // bounded by the spec (P10-02)
		if c.name != "loc" {
			fixed += c.width
		}
	}
	out := make([]studs.ColumnDefinition, 0, len(base))
	for _, c := range base { // bounded by the spec (P10-02)
		w := c.width
		if c.name == "loc" {
			w = max(alertLocMin, width-fixed)
		}
		out = append(out, studs.ColumnDefinition{Name: c.name, Header: c.header, Width: w, Alignment: "left"})
	}
	return out
}

// alertLocMin is the narrowest a place name may be cut to before the box would
// rather show a stub than nothing: "Rancho Pen…" still says WHERE.
const alertLocMin = 10

// alertHeader is the column titles, unpainted.
//
// NO BAND BEHIND IT, unlike the other two tables: the takeover's box is already
// tinted by the hazard's own category (D-86), and a second ground inside it would
// say the severity twice in two colours.
func (o Opts) alertHeader(cols []studs.ColumnDefinition, width int) string {
	cells := make([]string, 0, len(cols))
	for _, c := range cols { // bounded by the spec (P10-02)
		cells = append(cells, c.Header)
	}
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter, NoAutoStyle: true,
		Rows: []studs.EnhancedTableRow{{Data: clampCells(cells, cols)}}}
	rows := studs.NewDataTable(width, def).Rows()
	if len(rows) == 0 {
		return ""
	}
	return strings.TrimRight(rows[0], " ")
}
