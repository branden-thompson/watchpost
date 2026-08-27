package render

// table.go — the location table on go-studs DataTable — the ONLY go-studs consumer in the app (column spec, layout, row data, marks, styles, group headers). Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// DayCell is one EXTENDED FORECAST day column (original mock's ultra-wide
// spec: ">125 col, progressive expansion / truncation").
type DayCell struct {
	Date   string // mm/dd, rendered as the column header
	Hi, Lo *float64
}

// LocationRow is one dashboard table row (mock M-V1 columns).
type LocationRow struct {
	Index              int
	Name, Tag, Zip     string   // Tag = user 5-char label (mock LABEL column, hidden)
	Station            string   // observing station id (WX STN column, UAT 60)
	StationKM          *float64 // station distance (DIST column, UAT 60)
	Conditions         string
	Now, Hi, Lo        *float64
	Trend              string // "↗"/"↘"/"" — appended to NOW per mock
	TomorrowConditions string
	TomorrowHi         *float64
	TomorrowLo         *float64
	Extended           []DayCell // days beyond tomorrow (ultra-wide columns)
	Repeat             bool      // UAT 83: playing on repeat wears ∞ instead of ▶
	Fire               int       // B5 / UAT 110: fire events near the row (named incidents, or 1 for unnamed hotspots) — the row wears n◆
	FireHot            bool      // B5: at least one hotspot reads emphasized (FRP ≥ bold threshold)
	HasAlert           bool
	WarnAlert          bool // true = warning/alert-grade; false = advisory (UAT 14.1)
	AlertCount         int  // active alerts: rendered beside the ⚠ (UAT 20.2)
	Loading            bool // data not yet arrived: temps shimmer, not "n/a" (UAT 18.2)
	Playing            bool
	Selected           bool
}

// Base column spec, widths measured from the mock. GutterWidth 2 plus the
// component's prefix-zone rule (no gutter before columns 0-2) reproduces the
// mock's absolute offsets exactly. UAT 110 widened the marks block from 6
// to 11 cells (`›  ▶ 3◆ 2⚠ ` — pointer, two spacers, play, spacer, fire
// count + ◆, spacer, alert count + ⚠, spacer) and took the 5 cells from
// NAME's floor, so idx@11, name@16 and everything from LABEL on keeps its
// offset: label@37,
// zip@46, cond@55, now@69, hi@79, lo@86, tcond@95, thi@109, tlo@117 → 124.
type baseCol struct {
	name, header string
	width        int
	group        int // 0=none, 1=location, 2=today, 3=tomorrow
}

func baseColumns() []baseCol {
	return []baseCol{
		{"marks", "", marksW, 0}, // header icon hidden (UAT 20.1, CLIAmp style); column kept
		{"num", "###.", 5, 1},
		{"name", "NAME", nameMinW, 1},
		{"label", "LABEL", 7, 1},
		{"wxstn", "WX STN", 6, 1}, // observing station (UAT 60); "WX" keeps it apart from the NOAA radio transmitter
		{"dist", "DIST", 6, 1},    // "nnn km" / "nnn mi"
		{"zip", "ZIP", 7, 1},
		{"cond", "CONDITIONS", 12, 2},
		{"now", "NOW", 8, 2},
		{"hi", "HI", 5, 2},
		{"lo", "LOW", 7, 2},
		{"tcond", "CONDITIONS", 12, 3},
		{"thi", "HI", 6, 3},
		{"tlo", "LOW", 7, 3},
	}
}

// Extended-column geometry (original mock's ultra-wide spec: first day cell
// at col 128, 18-cell pitch; the mock's own pitch wobbles 17/18 — the
// component standardizes on 18).
const (
	tableGutter  = 2
	extSpacerW   = 2   // pads the ext group to the mock's col-128 start
	extDayW      = 18  // day column pitch
	extCellW     = 15  // "H 888ºF/888ºF L" (fixed slots: 3-digit temps never stagger, UAT 4.2)
	extMaxDays   = 5   // mock shows five day columns
	marksW       = 11  // the marks block (UAT 110 mock `›  ▶ 3◆ 2⚠ 001.`)
	nameMinW     = 19  // NAME fill floor: the mock's 24 less the 5 cells the marks block grew by (UAT 110); NAME fills again from 120 cols
	minFullW     = 115 // minimal full layout: all groups, LABEL hidden, NAME at floor
	minNoTmrwW   = 84  // minimal layout without the TOMORROW group
	stationColsW = 16  // WX STN + DIST with their gutters (UAT 60)
	minStationW  = minFullW + stationColsW
)

// layout is the active column set for a width (UAT-2D order as the terminal
// narrows: EXTENDED days leave first, then WX STN + DIST (UAT 60), then
// TOMORROW, then TODAY HI/LOW, and ZIP last — only when NAME would otherwise
// fall below its 10-cell floor; LABEL stays hidden, UAT 11.2).
type layout struct {
	tomorrow, label, hiLo bool
	station, zip          bool
	extDays               int
	nameMin               int // NAME fill floor: 24, compressing below the minimal layout width (UAT 35)
}

func layoutFor(width, days int) layout {
	l := layout{tomorrow: true, label: false, hiLo: true, zip: true} // LABEL hidden (UAT 11.2)
	switch {
	case width >= minStationW:
		// Station columns first (UAT 60), then extended day columns claim the
		// width beyond; the NAME fill absorbs whatever remains (UAT 11.1).
		l.station = true
		for k := 1; k <= min(extMaxDays, days); k++ {
			if width-minStationW >= extSpacerW+tableGutter+extCellW+(k-1)*extDayW {
				l.extDays = k
			}
		}
	case width >= minFullW:
	case width >= minNoTmrwW:
		l.tomorrow = false
	default:
		l.tomorrow, l.hiLo = false, false
	}
	// Very narrow terminals: the fixed columns of the minimal layout are 44
	// cells; let NAME shrink (floor 10) so the table never exceeds the width,
	// and drop ZIP (the last identity column to leave) before NAME breaks.
	l.nameMin = nameMinW
	if fixed := rowLen(l.columns(nil)); width-fixed < nameMinW {
		if width-fixed < 10 {
			l.zip = false
			fixed = rowLen(l.columns(nil))
		}
		l.nameMin = max(10, width-fixed)
	}
	return l
}

// columns assembles the go-studs column spec for a layout.
func (l layout) columns(dates []string) []studs.ColumnDefinition {
	base := baseColumns()
	cols := make([]studs.ColumnDefinition, 0, len(base)+1+extMaxDays)
	for _, c := range base {
		if l.hides(c) {
			continue
		}
		if c.name == "name" {
			// UAT 11.1: NAME is the go-studs fill column - the table always
			// stretches to the full content width (longer names welcome).
			nameMin := l.nameMin
			if nameMin == 0 {
				nameMin = nameMinW // geometry probe before layoutFor sets the floor
			}
			cols = append(cols, studs.ColumnDefinition{Name: c.name, Header: c.header,
				Fill: true, MinWidth: nameMin, Truncatable: true, TruncatedMinWidth: 10, Alignment: "left"})
			continue
		}
		cols = append(cols, studs.ColumnDefinition{Name: c.name, Header: c.header, Width: c.width, Alignment: "left"})
	}
	if l.extDays > 0 {
		cols = append(cols, studs.ColumnDefinition{Name: "extsp", Width: extSpacerW})
		for d := range l.extDays {
			hdr := ""
			if d < len(dates) {
				hdr = "     " + dates[d] // mm/dd at cell offset +5 (mock col 133)
			}
			w := extDayW
			if d == l.extDays-1 {
				w = extCellW // last day sheds its trailing pitch padding
			}
			cols = append(cols, studs.ColumnDefinition{
				Name: fmt.Sprintf("day%d", d), Header: hdr, Width: w, Alignment: "left", NoLeadingGutter: true,
			})
		}
	}
	return cols
}

// hides is the single owner of the column-drop policy (UAT-2D order): a
// base column is absent from the layout when its group or switch is off.
// LABEL stays hidden — ZIP identifies (UAT 11.2); its data is kept.
func (l layout) hides(c baseCol) bool {
	switch c.name {
	case "label":
		return true
	case "wxstn", "dist":
		return !l.station
	case "zip":
		return !l.zip
	case "hi", "lo":
		return !l.hiLo
	}
	return c.group == 3 && !l.tomorrow
}

// rowLen sums the spec's fixed widths + gutters (fill columns count 0).
func rowLen(cols []studs.ColumnDefinition) int {
	n := 0
	for i, c := range cols {
		if i >= 3 && !c.NoLeadingGutter {
			n += tableGutter
		}
		n += c.Width
	}
	return n
}

// colGeom resolves every rendered column's absolute offset AND width for a
// table width, mirroring the component's single-fill math (NAME = width -
// fixed - gutters, MinWidth-floored). Single owner of table geometry -
// group banding, rails, and future tables all derive from it.
type colGeom struct {
	name   string
	off, w int
}

func tableGeom(cols []studs.ColumnDefinition, tableW int) []colGeom {
	nameMin := nameMinW
	for _, c := range cols {
		if c.Fill || c.Width == 0 {
			nameMin = c.MinWidth // the layout's (possibly compressed) floor
		}
	}
	nameW := max(nameMin, tableW-rowLen(cols))
	off := 0
	geom := make([]colGeom, 0, len(cols))
	for i, c := range cols {
		if i >= 3 && !c.NoLeadingGutter {
			off += tableGutter
		}
		w := c.Width
		if c.Fill || w == 0 {
			w = nameW
		}
		geom = append(geom, colGeom{c.Name, off, w})
		off += w
	}
	return geom
}

// TableRowLen exposes the current table width: with the NAME fill the table
// spans the full width whenever it exceeds the layout's minimum.
func (o Opts) TableRowLen(days int) int {
	l := layoutFor(o.Width, days)
	return max(rowLen(l.columns(nil))+l.nameMin, o.Width)
}

// temp5 right-justifies a temperature in the mock's 5-cell field (the mock's
// " 72ºF"/" 00ºF" rows pin right-justification; ºF stays column-aligned).
func (o Opts) temp5(c *float64) string { return fmt.Sprintf("%5s", o.Temp(c)) }

// temp5Or renders the 5-cell temp slot, shimmering while the row is still
// loading (UAT 18.2b) instead of a misleading "n/a".
func (o Opts) temp5Or(c *float64, loading bool) string {
	if c == nil && loading {
		return fmt.Sprintf("%5s", o.LoadingDots())
	}
	return o.temp5(c)
}

// rowMarks builds the 6-cell prefix: pointer, play mark, and the alert
// badge — count beside the glyph ('›  2⚠'), both toned by the most severe
// alert (yellow advisory-grade, red warning-grade; UAT 20.2/20.3). Split
// from rowData (P10-04).
func rowMarks(r LocationRow, g Glyphs) [marksW]string {
	// UAT 110 mock: `›  ▶ 3◆ 2⚠ 001.` — 0 pointer · 1-2 spacers · 3 play ·
	// 4 spacer · 5 fire count · 6 ◆ · 7 spacer · 8 alert count · 9 ⚠ ·
	// 10 spacer.
	marks := [marksW]string{}
	for i := range marks {
		marks[i] = " "
	}
	if r.Selected {
		marks[0] = Tint(g.Pointer, Tok(FocusPointer)) // UAT 50.2: bold white pointer
	}
	if r.Playing {
		marks[3] = Tint(g.Play, Tok(StatePlaying)) // UAT 80: the playing location wears a green ▶
		if r.Repeat {
			marks[3] = Tint(g.Repeat, Tok(StatePlaying)) // UAT 83: ∞ on repeat
		}
	}
	if r.Fire > 0 { // B5 / UAT 110: fire is another alert kind — orange ◆ with its count (bold when a hotspot burns hard)
		tone := Tok(FireMark)
		if r.FireHot {
			tone = "1;" + tone
		}
		marks[5] = Tint(fmt.Sprintf("%d", min(r.Fire, 9)), tone)
		marks[6] = Tint(g.Fire, tone)
	}
	if !r.HasAlert {
		return marks
	}
	tone := Tok(AlertLabel)
	if r.WarnAlert {
		tone = Tok(AlertDanger)
	}
	if n := min(r.AlertCount, 9); n > 0 { // single glyph slot; 9 caps the badge
		marks[8] = Tint(fmt.Sprintf("%d", n), tone)
	}
	marks[9] = Tint(g.Alert, tone)
	return marks
}

// rowData formats one LocationRow into the spec's cell values.
func (o Opts) rowData(l layout, r LocationRow) []string {
	marks := rowMarks(r, o.Glyphs())
	trend := o.TrendGlyph(r.Trend)
	data := []string{strings.Join(marks[:], ""), fmt.Sprintf("%03d.", r.Index), r.Name}
	if l.label {
		data = append(data, r.Tag)
	}
	if l.station {
		data = append(data, r.Station, o.Distance(r.StationKM))
	}
	if l.zip {
		data = append(data, r.Zip)
	}
	data = append(data, DisplayCondition(r.Conditions), o.temp5Or(r.Now, r.Loading)+trend)
	if l.hiLo {
		data = append(data, o.temp5Or(r.Hi, r.Loading), o.temp5Or(r.Lo, r.Loading))
	}
	if l.tomorrow {
		data = append(data, DisplayCondition(r.TomorrowConditions), o.temp5Or(r.TomorrowHi, r.Loading), o.temp5Or(r.TomorrowLo, r.Loading))
	}
	if l.extDays > 0 {
		data = append(data, "") // spacer
		for d := range l.extDays {
			cell := ""
			if d < len(r.Extended) && (r.Extended[d].Hi != nil || r.Extended[d].Lo != nil) {
				// Fixed 5-cell slots (right-just hi / left-just lo) so 2- and
				// 3-digit temps never stagger the slash (UAT 4.2); HI orange,
				// LO cyan, n/a in the base grey (UAT 4.5).
				cell = "H " + tempTint(o.temp5(r.Extended[d].Hi), Tok(TempHi), r.Extended[d].Hi) +
					"/" + tempTint(fmt.Sprintf("%-5s", o.Temp(r.Extended[d].Lo)), Tok(TempLo), r.Extended[d].Lo) + " L"
			}
			data = append(data, cell)
		}
	}
	return data
}

// LocationTable renders the borderless M-V1 table through go-studs'
// DataTable (UAT ruling: use the component structure, not hand-rolled rows).
// days pins the EXTENDED column count so sibling tables (priority + recent)
// always share one layout (UAT session 3.1) - pass the max across both.
func (o Opts) LocationTable(rows []LocationRow, days int) string {
	var dates []string
	for _, r := range rows {
		if len(r.Extended) == 0 {
			continue
		}
		for _, dc := range r.Extended {
			dates = append(dates, dc.Date)
		}
		break
	}
	l := layoutFor(o.Width, days)
	cols := l.columns(dates)
	// The theme owns every colour in the table (Q4a-004, L5-F4): headers
	// through HeaderColor, cells through CellStyles, and the kit's own
	// $TERM-gated palette is switched off — so NO_COLOR is honoured by the
	// one gate (WrapSGR) and the `t` chooser restyles the whole table.
	header := Tok(TableHeader)
	for i := range cols {
		if cols[i].Header != "" {
			cols[i].HeaderColor = header
		}
	}
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter, NoAutoStyle: true}
	for _, r := range rows {
		data := clampCells(o.rowData(l, r), cols)
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: rowStyles(cols, r, data)})
	}
	dt := studs.NewDataTable(o.Width, def)
	out := []string{o.groupHeader(l, cols, o.Width), strings.TrimRight(dt.Header(), " ")}
	for _, line := range dt.Rows() {
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// tableCellStyles is the theme's default for every non-blank cell (Q4a-004):
// the row number and the attribute cells muted, the name bright; blank
// cells stay bare, as the kit left them; the marks column paints itself.
func tableCellStyles(data []string) map[int]string {
	m := make(map[int]string, len(data))
	for i := 1; i < len(data); i++ {
		if strings.TrimSpace(data[i]) == "" {
			continue
		}
		m[i] = Tok(TableMuted)
		if i == 2 {
			m[i] = Tok(TableName)
		}
	}
	return m
}

// rowStyles colors one row's cells (UAT 3.3/4.5/4.7): HIs orange, LOs cyan,
// n/a in the base grey, the focused row's name bold yellow - all applied by
// the component after padding, so alignment is untouched. Every other
// non-blank cell takes the theme's table tokens (Q4a-004): the row number
// and the attribute cells muted, the name bright — blank cells stay bare,
// as the kit left them.
func rowStyles(cols []studs.ColumnDefinition, r LocationRow, data []string) map[int]string {
	base := Tok(TextBase)
	if r.Selected {
		base = Tok(FocusCell) // UAT 50.1: focused row's grey cells read light blue
	}
	pick := func(v *float64, on string) string {
		if v == nil {
			return base
		}
		return on
	}
	m := tableCellStyles(data)
	for i, c := range cols {
		switch c.Name {
		case "num", "zip", "cond", "tcond", "now":
			if r.Selected {
				m[i] = base
			}
		case "hi":
			m[i] = pick(r.Hi, Tok(TempHi))
		case "thi":
			m[i] = pick(r.TomorrowHi, Tok(TempHi))
		case "lo":
			m[i] = pick(r.Lo, Tok(TempLo))
		case "tlo":
			m[i] = pick(r.TomorrowLo, Tok(TempLo))
		case "name":
			switch {
			case r.Selected:
				m[i] = Tok(FocusName) // focus outranks alert tinting
			case r.HasAlert && r.WarnAlert:
				m[i] = Tok(NameWarning)
			case r.HasAlert:
				m[i] = Tok(NameAdvisory)
			}
		}
	}
	return m
}

// tempTint wraps a padded temp slot in its color, or the base grey for n/a.
func tempTint(padded, fg string, v *float64) string {
	if v == nil {
		fg = Tok(TextBase)
	}
	return rendering.WrapSGR(padded, fg)
}

// clampCells hard-limits every cell to its column width: go-studs returns
// over-wide non-truncatable cells as-is, which shifts the whole row (UAT
// session 3.2 - "PARTLY_CLOUDY" at 13 cells broke a 12-cell column).
func clampCells(data []string, cols []studs.ColumnDefinition) []string {
	for i, v := range data {
		if i < len(cols) && cols[i].Width > 0 && displayWidth(v) > cols[i].Width {
			data[i] = truncate(v, cols[i].Width) // fill cols truncate via the component
		}
	}
	return data
}

// DisplayCondition maps provider condition strings to the mock's vocabulary
// ("P.CLOUDY"): underscores to spaces, PARTLY/MOSTLY abbreviated — the
// table's and the Details modal's one vocabulary (Q6, L3-F10).
func DisplayCondition(c string) string {
	c = strings.ToUpper(strings.ReplaceAll(c, "_", " "))
	c = strings.ReplaceAll(c, "PARTLY ", "P.")
	c = strings.ReplaceAll(c, "MOSTLY ", "M.")
	return c
}

// groupSpec is one header band: title, chip tone, member column names.
// Declarative and orderable - future tables declare their own list
// (UAT 14.4 modularity note).
type groupSpec struct {
	title   string
	short   string // narrow-band fallback title (UAT 16.1)
	bg      Token
	members []string
}

func groupsFor(l layout) []groupSpec {
	today := []string{"cond", "now"}
	if l.hiLo {
		today = append(today, "hi", "lo")
	}
	g := []groupSpec{
		{"L O C A T I O N", "L O C A T I O N", GroupLocationBG, []string{"marks", "num", "name", "wxstn", "dist", "zip"}}, // '/ STATION' dropped: radio is location-based (UAT 44.3)
		{"T O D A Y", "", GroupTodayBG, today},
	}
	if l.tomorrow {
		g = append(g, groupSpec{"T O M O R R O W", "", GroupTomorrowBG, []string{"tcond", "thi", "tlo"}})
	}
	if l.extDays > 0 {
		days := make([]string, 0, l.extDays)
		for d := range l.extDays {
			days = append(days, fmt.Sprintf("day%d", d))
		}
		g = append(g, groupSpec{"E X T E N D E D   F O R E C A S T", "E X T E N D E D", GroupExtendedBG, days})
	}
	return g
}

// groupHeader renders the group bands from the resolved column geometry.
// Bands extend to MEET at gutter midpoints (UAT 14.4: labels read as one
// continuous strip while the columns beneath keep their spacing); the
// column set and fill width drive every span.
func (o Opts) groupHeader(l layout, cols []studs.ColumnDefinition, tableW int) string {
	geom := tableGeom(cols, tableW)
	span := func(names []string) (int, int) {
		lo, hi := -1, -1
		for _, g := range geom {
			for _, n := range names {
				if g.name != n {
					continue
				}
				if lo < 0 || g.off < lo {
					lo = g.off
				}
				hi = max(hi, g.off+g.w-1)
			}
		}
		return lo, hi
	}
	groups := groupsFor(l)
	// Resolve every band's raw span, then stretch adjacent bands to MEET at
	// the midpoint of whatever separates them (plain gutters AND the wider
	// gutter+spacer gap before EXTENDED - UAT 15.1).
	type band struct {
		lo, hi int
		g      groupSpec
	}
	bands := make([]band, 0, len(groups))
	for _, g := range groups {
		if lo, hi := span(g.members); lo >= 0 {
			bands = append(bands, band{lo, hi, g})
		}
	}
	for i := 1; i < len(bands); i++ {
		mid := (bands[i-1].hi + bands[i].lo) / 2
		bands[i-1].hi, bands[i].lo = mid, mid+1
	}
	var b strings.Builder
	for _, bd := range bands {
		b.WriteString(strings.Repeat(" ", max(0, bd.lo-displayWidth(b.String()))))
		if colorOn() {
			b.WriteString(sgrRaw(" "+centered(bd.g.title, bd.g.short, bd.hi-bd.lo-1)+" ", Tok(GroupText)+";"+Tok(bd.g.bg)))
		} else {
			b.WriteString(bracketTitle(bd.g.title, bd.g.short, bd.hi-bd.lo+1))
		}
	}
	return b.String()
}
