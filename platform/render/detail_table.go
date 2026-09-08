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

// DetailGutter is the air between a detail table's columns — ONE cell, not the
// [S] window's two.
//
// A detail section is dense and its columns are already separated by their own
// alignment: a right-aligned number ends where the next begins. The second cell
// buys nothing and costs one per column boundary, which across seven columns is
// the difference between the last one fitting and being cut at the 65 cells a
// detail section has (85-wide modal, less the panel and the label gutter).
const DetailGutter = 1

// fitColumns sizes every unsized column to its own widest cell.
//
// THE KIT HAS NO "FIT": Width == 0 means FILL there (data_table_row.go:539), so
// a table of unsized columns stretches to the section's edge and puts a hand's
// width of air in the middle of a row. Measuring here is what "as wide as the
// widest data cell" means, and it pulls the table in to the width of what is
// actually in it — which is also what keeps the last column clear of the scroll
// rail.
//
// A column that asks for a Width or a Fill keeps it: the mark column is one
// cell whatever is in it.
func fitColumns(cols []StatusColumn, rows []StatusRow) []StatusColumn {
	out := append([]StatusColumn(nil), cols...)
	for i := range out {
		if out[i].Width > 0 || out[i].Fill {
			continue
		}
		w := out[i].MinWidth
		for _, r := range rows { // bounded by the table (P10-02)
			if i < len(r.Cells) {
				w = max(w, Width(r.Cells[i]))
			}
		}
		out[i].Width = max(w, 1)
	}
	return out
}

// DetailTable lays out rows at inner cells wide and returns them, no header.
//
// The FILL column absorbs what is left over, so a name is never truncated to
// make room for a number — the HUM LEAD's rule for this section, and the same
// one the seismic section already follows: *"this can not worry about that just
// like the USGS seismic section does not worry about it."*
func (o Opts) DetailTable(cols []StatusColumn, rows []StatusRow, inner, gutter int) []string {
	if len(cols) == 0 || len(rows) == 0 {
		return nil
	}
	// FIT FIRST, THEN THE GUTTERS. statusGutters adds its cell of air to a
	// column's WIDTH, and a column with no width yet gets nothing — which is
	// exactly how a right-aligned fit column ended up touching the name beside
	// it. Sizing first gives it something to add to.
	if gutter > 0 {
		cols, rows = statusGutters(fitColumns(cols, rows), rows)
	} else {
		// GUTTER 0: the caller is placing its own columns to the cell, because
		// it has to line up with something drawn elsewhere. The gaps then live
		// in the widths and the cells, where they can be read off against the
		// section they are matching.
		cols = fitColumns(cols, rows)
	}
	def := &studs.DataTableDefinition{Columns: statusCols(cols, ""), GutterWidth: gutter, NoAutoStyle: true}
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
