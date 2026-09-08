package render

// detail_table.go — a table INSIDE a report section (UAT 2026-09-07).
//
// THROUGH THE KIT, LIKE EVERY OTHER TABLE. The detail report's sections lay
// their columns out by hand with PadTo, which is why adding a column means
// re-counting every literal in the function — and why the fire section's
// columns did not survive being asked to carry one more thing. A table with a
// column spec can gain or lose a column without anyone re-counting anything.
//
// ROWS ONLY, NO HEADER ROW: a detail section already has a heading of its own,
// and a second bar of column names inside a report that is read top-to-bottom
// is noise. StatusTable, next door, is the same kit with a header, for the [S]
// window where the columns need naming.

import studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"

// DetailTable lays out rows at inner cells wide and returns them, no header.
//
// The FILL column absorbs what is left over, so a name is never truncated to
// make room for a number — the HUM LEAD's rule for this section, and the same
// one the seismic section already follows: *"this can not worry about that just
// like the USGS seismic section does not worry about it."*
func (o Opts) DetailTable(cols []StatusColumn, rows []StatusRow, inner int) []string {
	if len(cols) == 0 || len(rows) == 0 {
		return nil
	}
	cols, rows = statusGutters(cols, rows)
	def := &studs.DataTableDefinition{Columns: statusCols(cols, ""), GutterWidth: statusGutter, NoAutoStyle: true}
	for _, r := range rows {
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: r.Cells, CellStyles: r.Styles})
	}
	dt := studs.NewDataTableRowFromLayout(inner, def)
	out := make([]string, 0, len(rows))
	for _, r := range def.Rows {
		line := dt.FormatEnhancedTableRow(r)
		// The same rectangle rule StatusTable states: the kit sizes a fill
		// column from the row's own cells, so lines come out a cell apart and
		// the section needs them square.
		if Width(line) > inner {
			line = splitCells(line, inner)[0]
		}
		out = append(out, PadTo(line, inner))
	}
	return out
}
