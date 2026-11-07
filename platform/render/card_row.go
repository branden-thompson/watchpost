package render

// card_row.go — the console's card title row on the go-studs table kit, so
// the kit stays behind this package (0.19.5 D-10).

import (
	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// CardRow is a card's title row: an inset, or in its place a fabricated
// event's mark, then the headline, left-aligned and truncating, and a badge
// right-anchored to the row.
//
// A GO-STUDS ROW, per the standing rule, and it earns its place rather than
// merely satisfying it: the fill column is sized with the badge's width ALREADY
// RESERVED, so a title cannot run into the badge. A title can fit BY LENGTH and
// still collide by POSITION — "…(COASTAL)D•". Here that is structural rather
// than policed.
type CardRow struct {
	plain *studs.DataTableRow
	// marked is the same row with a leading, NON-TRUNCATABLE column for the
	// mark (D-55). TWO ROWS RATHER THAN ONE, because a fixed column present on
	// every card would reserve its width on every card — the real hazards would
	// all sit 14 cells off-centre to make room for a label they never carry.
	//
	// IT IS A COLUMN AND NOT `SetPrefix` BECAUSE THE COMPONENT DOES NOT RENDER
	// ONE ON DATA ROWS. `SetPrefix` is accounted for in the width calculation
	// (`prefixWidth`, data_table_row.go:521) and written only by `RenderHeader`
	// — so a prefix set here reserves its space and prints nothing. The gap is
	// an M6 upstream candidate rather than patched locally: the dependency is
	// not re-implemented.
	marked *studs.DataTableRow
}

// NewCardRow is the title row for a card width cells wide, with an inset of
// inset cells, room for a mark markWidth cells wide, and ellipsis to show a
// truncated headline.
func NewCardRow(width, inset, markWidth int, ellipsis string) CardRow {
	headline := studs.ColumnDefinition{
		Name: "headline",
		Fill: true,
		// LEFT, FROM THE v2 REFERENCE (D-87). A centred title floats between
		// the badge and the border while every other row of the card begins at
		// the same inset — so the one line naming the card would be the one line
		// that does not line up with it.
		Alignment:         "left",
		Truncatable:       true,
		TruncatedMinWidth: 8,
		TruncationTail:    ellipsis,
	}
	// THE INSET IS A COLUMN RATHER THAN A PREFIX, because the badge is
	// right-anchored to the ROW: prefixing would push the row three cells wide
	// and take the handle with it.
	in := studs.ColumnDefinition{Name: "inset", Width: inset}
	mark := studs.ColumnDefinition{
		Name:  "mark",
		Width: markWidth,
		// NEVER TRUNCATABLE. A half-eaten mark is worse than none: "**TEST EV…"
		// beside a tornado warning reads as damage rather than as a label.
		//
		// FALSE IS THE ZERO VALUE, so this line is documentation with syntax and
		// its mutant SURVIVES BY DESIGN — the component never needs to shrink
		// this column at any width the row can be built at, because the fill
		// column absorbs the squeeze first. It is stated because the guarantee
		// belongs to a dependency whose sizing this package does not own, and
		// the console's own backstop is what enforces the rule.
		Truncatable: false,
	}
	return CardRow{
		plain:  studs.NewDataTableRow(width, []studs.ColumnDefinition{in, headline}),
		marked: studs.NewDataTableRow(width, []studs.ColumnDefinition{mark, headline}),
	}
}

// Ready reports whether the row was built, for a width above nothing. Both
// rows are built together, so one answer covers both.
func (r CardRow) Ready() bool { return r.plain != nil && r.marked != nil }

// Render draws the row: the headline, the badge two cells from the headline's
// room, and, when mark is not empty, the mark in place of the inset.
func (r CardRow) Render(headline, mark, badge string) string {
	row, data := r.plain, map[string]string{"headline": headline}
	if mark != "" {
		row, data["mark"] = r.marked, mark
	}
	row.SetBadge(badge, 2)
	return row.RenderRow(data)
}
