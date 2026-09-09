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

// shrinkToFit takes an overflow out of the columns that said they could lose
// it, and leaves the rest alone.
//
// FIT SIZES A COLUMN TO ITS CONTENT, which means a long value makes a wide
// column and the kit — believing every column fits — never truncates anything.
// The row then runs past the section and is CUT, with no tail, in the middle of
// a word: "PAGER — Almost certainly" for "PAGER — Almost certainly felt". A cut
// is what a clamp does to a line; a truncation is what a column does to a
// value, and only the second one leaves a mark saying so.
//
// The excess comes off the Truncatable columns in order, each floored at its
// MinWidth. A table with none keeps its width, and the caller's clamp is then
// the honest outcome: nothing in it volunteered to be shortened.
func shrinkToFit(cols []StatusColumn, rows []StatusRow, inner, gutter int) ([]StatusColumn, []StatusRow) {
	out := append([]StatusColumn(nil), cols...)
	natural := 0
	for i, c := range out {
		natural += c.Width
		if i >= 3 && !c.NoGutter { // the kit's own rule: a gutter precedes columns 3..last
			natural += gutter
		}
	}
	over := natural - inner
	drop := map[int]bool{}
	for i := range out { // bounded by the columns (P10-02)
		if over <= 0 || !out[i].Truncatable {
			continue
		}
		// SHRINK TO ITS FLOOR, THEN DROP IT ENTIRELY. MinWidth on a truncatable
		// column is the width below which its value stops meaning anything: an
		// age cut to three cells renders "...", a cell shaped like a value that
		// says nothing. Past that the column goes, because an absent column
		// reads better than an empty one.
		if take := min(over, out[i].Width-out[i].MinWidth); take > 0 {
			out[i].Width -= take
			over -= take
		}
		if over > 0 {
			over -= out[i].Width + gutter
			drop[i] = true
		}
	}
	if len(drop) == 0 {
		return out, rows
	}
	// REMOVED, NOT ZEROED. A zero width means FILL in the kit
	// (data_table_row.go:539), so a "dropped" column would stretch to take
	// everything that is left — the exact opposite of dropping it.
	kept := make([]StatusColumn, 0, len(out))
	shift := make([]int, len(out)) // old index -> new, for the per-cell styles
	for i, c := range out {
		if drop[i] {
			shift[i] = -1
			continue
		}
		shift[i] = len(kept)
		kept = append(kept, c)
	}
	trimmed := make([]StatusRow, len(rows))
	for j, r := range rows {
		cells := make([]string, 0, len(kept))
		var styles map[int]string
		for i, cell := range r.Cells {
			if i >= len(shift) || shift[i] < 0 {
				continue
			}
			if s, ok := r.Styles[i]; ok {
				if styles == nil {
					styles = map[int]string{}
				}
				styles[shift[i]] = s
			}
			cells = append(cells, cell)
		}
		trimmed[j] = StatusRow{Cells: cells, Styles: styles}
	}
	return kept, trimmed
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
		cols, rows = shrinkToFit(cols, rows, inner, gutter)
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
