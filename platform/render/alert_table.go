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

// The takeover's column spec, from the HUM LEAD (2026-09-12):
//
//	[##.:3][ALERT TYPE: FILL][gutter:3][LOCATION: FIT; TRUNCATABLE]
//
// THREE SHAPES, NOT THREE NUMBERS. `##.` is exactly as wide as it reads; ALERT
// TYPE takes whatever is left; LOCATION is as wide as the widest place name in
// the box and no wider, and is the one that gives way when there is not room.
//
// THE THREE-CELL GUTTER IS THE POINT OF THE SPEC. With ALERT TYPE filling, a
// hazard that used its last cell sat flush against the place name — which is
// what the HUM LEAD saw: "SEVERE THUNDERSTORM WARN…Harper, KS".
const (
	alertNumW = 3 // "##."
	alertGap1 = 1 // between the number and the hazard
	alertGap2 = 3 // between the hazard and the place (the HUM LEAD's spec)
	// alertKindW is the hazard's floor: "T.STORM WATCH" is the shortest name the
	// reference draws, and a column that cannot hold it whole has stopped being
	// useful before the place name has given anything up.
	alertKindW = 14
	// alertLocMin is the narrowest a place name may be cut to before the box
	// would rather show a stub than nothing: "Rancho Pen…" still says WHERE.
	alertLocMin = 10
)

// alertWidths sizes the three columns for a given box and its contents.
//
// LOCATION IS MEASURED, NOT DECLARED, which is what FIT means: the box is about
// fifty cells wide and every cell LOCATION does not need is a cell the hazard
// can use. It is measured over the WHOLE list, so the column does not change
// width as hazards arrive and the operator's eye does not have to re-find it.
func alertWidths(rows []AlertRow, width int) (kind, loc int) {
	for _, r := range rows { // bounded by the burst (P10-02)
		loc = max(loc, displayWidth(r.Location))
	}
	loc = max(loc, displayWidth("LOCATION"))
	room := width - alertNumW - alertGap1 - alertGap2
	// LOCATION GIVES WAY FIRST, AND ONLY AS FAR AS IT MUST. The hazard keeps its
	// floor before the place name is cut at all, which is the reference's own
	// priority: an operator can place "Rancho Pen…" and cannot place "SEV. T.ST…".
	loc = min(loc, max(alertLocMin, room-alertKindW))
	return max(0, room-loc), loc
}

// AlertTable draws the takeover's contents.
//
// THE SAME ASSEMBLY AS THE OTHER TWO, and no group band: this table lives INSIDE
// a box that already names it, so a band over it would be the title said twice.
func (o Opts) AlertTable(rows []AlertRow, width int) string {
	if width <= 0 {
		return ""
	}
	kindW, locW := alertWidths(rows, width)
	cols := alertColumnDefs(kindW, locW)
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter, NoAutoStyle: true}
	for _, r := range rows { // bounded by the burst (P10-02)
		// THE GUTTERS LIVE IN THE COLUMNS AND THE DATA IS CLAMPED WITHOUT THEM.
		// The kit's gutters are positional — it puts none before column three —
		// so a three-column table gets none at all, and a spec that asks for one
		// has to carry it in the width. Clamping the cell to the width WITHOUT
		// its gutter is what keeps the gutter from being eaten by a long hazard.
		data := []string{r.Num, truncate(ShortHazard(r.Kind), kindW), truncate(r.Location, locW)}
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: tableCellStyles(data)})
	}
	dt := studs.NewDataTable(width, def)
	out := []string{o.alertHeader(cols, width)}
	for _, line := range dt.Rows() { // bounded by the rows above (P10-02)
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// alertColumnDefs turns the sized spec into the kit's definitions.
func alertColumnDefs(kind, loc int) []studs.ColumnDefinition {
	return []studs.ColumnDefinition{
		{Name: "num", Header: "##.", Width: alertNumW + alertGap1, Alignment: "left"},
		{Name: "kind", Header: "ALERT TYPE", Width: kind + alertGap2, Alignment: "left"},
		{Name: "loc", Header: "LOCATION", Width: loc, Alignment: "left"},
	}
}

// ShortHazard is the takeover box's own vocabulary for a hazard's name.
//
// HUM LEAD, 2026-09-12: "Shorthand Vocabulary was to ensure the alert type would
// fit in the window; we should keep it for that window."
//
// THE WINDOW IS ABOUT FIFTY CELLS WIDE and a hazard's official name is not built
// for it — "SEVERE THUNDERSTORM WARNING" is twenty-seven on its own. The
// reference writes them short: SEV. T.STORM WARNING, SPEC. WEATHER ADVISORY.
//
// WORD BY WORD, NOT PHRASE BY PHRASE, so a name the reference never drew still
// comes out shortened where it can be — and one nobody has a shorthand for comes
// out whole rather than mangled.
//
// THIS TABLE ONLY. Everywhere else the app has the room and says the hazard's
// real name; a shorthand that leaked would be the app inventing terminology on a
// safety surface. THE SET IS THE REFERENCE'S OWN and grows by ruling, not by
// guess.
func ShortHazard(s string) string {
	out := strings.Fields(s)
	for i, w := range out { // bounded by the name (P10-02)
		if short, ok := hazardShorthand[w]; ok {
			out[i] = short
		}
	}
	return strings.Join(out, " ")
}

// hazardShorthand is the vocabulary, read off `mock-broadcaster-v3.txt`.
var hazardShorthand = map[string]string{
	"SEVERE":       "SEV.",
	"THUNDERSTORM": "T.STORM",
	"SPECIAL":      "SPEC.",
}

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
