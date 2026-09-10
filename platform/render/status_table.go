package render

// status_table.go — the [S] window's tables, laid out by the KIT (0.14.0).
//
// THE RULE: when we need a table, we use a go-studs table. That is what it was
// vendored for. These three were hand-rolled with
// fmt.Sprintf and column widths measured by hand, which reimplemented — worse —
// what ColumnDefinition already does: Width for a fixed column, Fill for the one
// that takes the slack, Truncatable with its own floor and tail, and
// NoLeadingGutter to pair a mark cell with the column it belongs to.
//
// The seam is the one SevereTable already uses: modes/tty builds the CELLS,
// this package lays them out. A mode never touches the kit.

import (
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// StatusColumn is one column of a [S] table, in the vocabulary a caller needs:
// a name, a width or a fill, and whether it may be cut when the room runs out.
type StatusColumn struct {
	Header      string
	Width       int  // 0 with Fill takes the slack; 0 without it sizes to content
	Fill        bool // exactly one column per table should carry this
	Right       bool // numbers align right, names left
	Truncatable bool
	MinWidth    int
	NoGutter    bool // pairs a mark cell with the column it marks
}

// StatusRow is one row: its cells, and a style per cell index.
type StatusRow struct {
	Cells  []string
	Styles map[int]string
}

// StatusTable lays out a [S] table at inner cells wide and returns its header
// followed by its rows.
//
// The FILL column absorbs whatever the window has left over, so the table spans
// the inner width instead of ending short of it — and it is the only column that
// stretches, because padding a measurement puts air between a number and its own
// heading.
func (o Opts) StatusTable(cols []StatusColumn, rows []StatusRow, inner int, headerTone string) []string {
	if len(cols) == 0 {
		return nil
	}
	cols, rows = statusGutters(cols, rows)
	def := &studs.DataTableDefinition{Columns: statusCols(cols, headerTone), GutterWidth: statusGutter, NoAutoStyle: true}
	for _, r := range rows {
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: r.Cells, CellStyles: r.Styles})
	}
	dt := studs.NewDataTableRowFromLayout(inner, def)
	// EVERY LINE EXACTLY inner WIDE, the contract SevereTable states too. The
	// kit's header path and its row path size a fill column from different
	// inputs — an empty row for the header, the row's own cells for a row — and
	// come out a cell apart; the window needs a rectangle, so this is where it
	// becomes one, padded up and clamped down by OUR measure.
	// TruncateCells, WHICH NOW MEASURES WHAT `Width` MEASURES. This used to reach
	// for `splitCells` because "TruncateCells counts an escape sequence's bytes
	// as content and will cut through the middle of one" — true when it was
	// written, and worked around HERE instead of fixed THERE. The console's
	// masthead then hit the same bug through the same function and lost half its
	// content (HUM LEAD, UAT 2026-09-10). A known-wrong shared function with a
	// local detour around it is the shape "one canonical way" exists to stop.
	fit := func(line string) string { return PadTo(TruncateCells(line, inner), inner) }
	out := []string{fit(dt.RenderHeader())}
	for _, r := range def.Rows {
		out = append(out, fit(dt.FormatEnhancedTableRow(r)))
	}
	return out
}

// statusGutter is the air between columns — two cells, the spacing these tables
// were drawn with.
const statusGutter = 2

// statusGutters supplies the gutters the kit does not.
//
// go-studs puts air before a column only from INDEX THREE (gutterBefore is
// `i >= 3`), because the dashboard tables it was written for lead with a
// prefix, a number and a name that butt together on purpose. Ours do not: every
// column here is a separate reading and the first three ran into each other —
// "COOPS · COOPS-OBSREF OK".
//
// Rather than fork the kit for a layout preference, the air is folded into the
// columns it is missing from: the width grows by a gutter, and a LEFT-aligned
// column also carries the spaces at the front of its text. A right-aligned one
// needs nothing more, since the extra width lands on its left already.
func statusGutters(cols []StatusColumn, rows []StatusRow) ([]StatusColumn, []StatusRow) {
	out := append([]StatusColumn(nil), cols...)
	rs := make([]StatusRow, len(rows))
	for i, r := range rows {
		rs[i] = StatusRow{Cells: append([]string(nil), r.Cells...), Styles: r.Styles}
	}
	lead := strings.Repeat(" ", statusGutter)
	for i := 1; i < len(out) && i < 3; i++ {
		if out[i].NoGutter {
			continue
		}
		if out[i].Width > 0 {
			out[i].Width += statusGutter
		}
		if out[i].MinWidth > 0 {
			out[i].MinWidth += statusGutter
		}
		if out[i].Right {
			continue
		}
		out[i].Header = lead + out[i].Header
		for j := range rs {
			if i < len(rs[j].Cells) {
				rs[j].Cells[i] = lead + rs[j].Cells[i]
			}
		}
	}
	return out, rs
}

// StatusNaturalWidth is what a column set occupies with nothing stretched — the
// fill column at its floor. It is what a caller measures a form against before
// deciding the table fits, and it counts the gutters statusGutters supplies as
// well as the ones the kit does.
func StatusNaturalWidth(cols []StatusColumn) int {
	w := 0
	for i, c := range cols {
		if i > 0 && !c.NoGutter {
			w += statusGutter
		}
		if c.Width > 0 {
			w += c.Width
			continue
		}
		w += c.MinWidth
	}
	return w
}

// statusCols maps the caller's vocabulary onto the kit's.
func statusCols(cols []StatusColumn, headerTone string) []studs.ColumnDefinition {
	out := make([]studs.ColumnDefinition, 0, len(cols))
	for _, c := range cols {
		align := "left"
		if c.Right {
			align = "right"
		}
		out = append(out, studs.ColumnDefinition{
			Name: c.Header, Header: c.Header, Width: c.Width, MinWidth: c.MinWidth,
			Fill: c.Fill, Alignment: align, HeaderColor: headerTone,
			Truncatable: c.Truncatable, TruncatedMinWidth: c.MinWidth, TruncationTail: "…",
			NoLeadingGutter: c.NoGutter,
		})
	}
	return out
}
