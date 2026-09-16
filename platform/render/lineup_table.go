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

	// Conditions, Now and Trend are the WEATHER where this beat is about (D-116).
	//
	// THEY REPLACED THE CORRESPONDENT (HUM LEAD, 2026-09-13): "I think we change
	// P R E S E N T A T I O N / CORRESPONDENT to C U R R E N T L Y / CONDITIONS
	// NOW … Useful information and doesn't require the correspondents wiring
	// work." The column read `N/A` on every row because nothing calls
	// `Card.WithReadBy` and main-track segments carry no cast role — so it was
	// twenty-eight cells of a table saying nothing, where the operator deciding
	// what to put on the air wanted to know what the weather is doing there.
	Conditions string
	Now        *float64
	Trend      string
	// Loading shimmers the temperature until this location's data lands, exactly
	// as the location table does (UAT 18.2) — an honest "still coming" rather
	// than an "n/a" that reads as "nothing to report".
	Loading bool

	Marks Marks
}

// lineup column widths, measured off the v3 mock.
func lineupColumns() []baseCol {
	// MEASURED TO THE FRAME, NOT TO THE MOCK'S PIXELS. The console's band is 144
	// cells at the reference's 150-wide terminal, and these sum to 128 + eight
	// gutters = 144 exactly.
	//
	// THE MARKS ZONE IS OBSERVER'S THIRTEEN, not the mock's narrower six. Parity
	// is the whole requirement — "the same prefix / alert tags as the Observer
	// table … so the user doesn't have to relearn" — so the seven cells it costs
	// come out of the text columns instead.
	return []baseCol{
		{"marks", "", marksW, 0},
		{"num", "##.", 5, 1},
		{"type", "REPORT TYPE", 20, 1},
		{"loc", "LOCATION", 22, 1},
		{"zip", "ZIP", 7, 1},
		{"dist", "DIST", 6, 1},
		{"prio", "PRIORITY", 10, 2},
		{"req", "REQUESTED BY", 17, 2},
		// CURRENTLY (D-116). The location table's own two columns, at the location
		// table's own widths, because they show the same two facts — and a
		// CONDITIONS that was twelve cells there and ten here would put one
		// vocabulary in two shapes.
		{"cond", "CONDITIONS", 12, 3},
		{"now", "NOW", 8, 3},
	}
}

// lineupGroups names the three questions a row answers.
func lineupGroups() []groupSpec {
	return []groupSpec{
		{"R E A D   O U T S", "R E A D   O U T S", GroupLocationBG, []string{"marks", "num", "type", "loc", "zip", "dist"}},
		{"D I R E C T I O N", "D I R E C T I O N", GroupTodayBG, []string{"prio", "req"}},
		{"C U R R E N T L Y", "C U R R E N T L Y", GroupTomorrowBG, []string{"cond", "now"}},
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
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: lineupRowStyles(cols, r, data)})
	}
	dt := studs.NewDataTable(width, def)
	groups := lineupGroups()
	out := []string{o.groupHeader(groups, cols, width), o.columnHeader(groups, cols, width)}
	for _, line := range dt.Rows() { // bounded by the rows above (P10-02)
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// lineupColumnDefs turns the spec into the kit's definitions.
//
// LOCATION AND REPORT TYPE TAKE THE SLACK, and the kit's own `Fill` is what
// grants it — so this needs no width. It TOOK one and never read it (P10-07).
// The HUM LEAD ruled the second fill column on 2026-09-16 (F-112), which is what
// made the doc line above — "giving the two WIDEST COLUMNS the slack" — true
// rather than something to correct away.
// lineupNaturalWidth is the width this table occupies with every column at its
// declared size and no surplus to share.
//
// IT ASKS `rowLen`, WHICH OWNS THE GEOMETRY. A gutter precedes columns 3..last
// and the category spacer widens two of them, so the sum is not the column
// widths plus one gutter per pair — re-deriving that here produced 138 for a
// table that occupies 140, and a surplus computed from a wrong base mis-splits
// the day a column changes group or width.
func lineupNaturalWidth() int {
	cols := lineupColumns()
	defs := make([]studs.ColumnDefinition, 0, len(cols))
	for _, c := range cols { // bounded by the spec (P10-02)
		defs = append(defs, studs.ColumnDefinition{Name: c.name, Header: c.header, Width: c.width})
	}
	return rowLen(spaceCategories(defs, lineupGroups()))
}

func lineupColumnDefs(width int) []studs.ColumnDefinition {
	out := make([]studs.ColumnDefinition, 0, len(lineupColumns()))
	for _, c := range lineupColumns() { // bounded by the spec (P10-02)
		// LOCATION AND REPORT TYPE SHARE THE SLACK (F-112, HUM LEAD 2026-09-16:
		// "give fill to both report type and location").
		//
		// LOCATION IS THE FILL COLUMN, as NAME is on Observer's table and for the
		// same reason: a place name is the one cell whose length nobody controls,
		// so it takes the slack and gives way first.
		//
		// AND REPORT TYPE TAKES HALF THE SURPLUS AS A FIXED WIDTH, computed here
		// rather than by marking it `Fill` as well. THE KIT CANNOT DO TWO FILL
		// COLUMNS CORRECTLY: `data_table_row.go` computes the space to share as
		// `terminalWidth - usedWidth` where `usedWidth` counts the FIXED columns
		// and NOT the gutters, so the gutter allowance is handed out once per
		// fill column. With one that error is absorbed; with two it is counted
		// twice — measured, a 144-cell band rendered 190. An upstream candidate
		// (M6); patched around here rather than reimplemented, which is the
		// standing rule for this kit.
		//
		// SO THE SPLIT IS OURS AND THE FILL IS THEIRS: REPORT TYPE grows by half
		// the surplus, LOCATION fills whatever is left, and the total stays the
		// band the caller asked for.
		if c.name == "type" {
			w := c.width
			if extra := (width - lineupNaturalWidth()) / 2; extra > 0 {
				w += extra
			}
			out = append(out, studs.ColumnDefinition{Name: c.name, Header: c.header,
				Width: w, Alignment: "left"})
			continue
		}
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
	return spaceCategories(out, lineupGroups())
}

// lineupRowStyles paints the focused row the way the location table paints its
// own (D-105).
//
// HUM LEAD, UAT 2026-09-12: "Rows in the Scheduled Line up should highlight just
// like the location pool table."
//
// THE POINTER WALKS BOTH TABLES AND ONLY ONE OF THEM ANSWERED. The pool's focused
// row reads light blue with its name picked out; the running order's wore the
// pointer glyph and nothing else, so the operator's eye had one cell to find on a
// fifteen-row list — and the two halves of one pointer looked like two pointers.
//
// LOCATION IS THE NAME HERE, which is what `rowStyles` picks out on the other
// table: the cell that says WHICH row this is.
func lineupRowStyles(cols []studs.ColumnDefinition, r LineupRow, data []string) map[int]string {
	m := tableCellStyles(data)
	if !r.Marks.Selected {
		return m
	}
	for i, c := range cols { // bounded by the spec (P10-02)
		switch c.Name {
		case "marks":
		case "loc":
			m[i] = Tok(FocusName)
		default:
			m[i] = Tok(FocusCell)
		}
	}
	return m
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
		// THE LOCATION TABLE'S OWN FORMATTERS (D-116), so a place's conditions
		// read the same on the running order as they do in the pool below it:
		// `DisplayCondition` is the one vocabulary ("P.CLOUDY") and `temp5Or` is
		// the one that shimmers while the data is still coming.
		DisplayCondition(r.Conditions),
		o.temp5Or(r.Now, r.Loading) + o.TrendGlyph(r.Trend),
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
