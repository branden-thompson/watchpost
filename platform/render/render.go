// Package render is the D-9 pivot seam: the ONLY package allowed to import
// go-studs. The location table rides go-studs' DataTable (spec struct =
// DataTableDefinition/ColumnDefinition, data struct = EnhancedTableRow —
// the reference-CLI pattern, HUM LEAD directive UAT session 2): auto-sizing,
// truncation, and gutters are the component's job; this seam only assembles
// the column spec for the current width (UAT-2D responsive drops + the
// ultra-wide EXTENDED FORECAST columns) and formats watchpost values.
// Rules baked in (architecture §4): severity and health are NEVER color-only
// (text glyphs/labels always present — R-12a); --ascii swaps glyphs without
// dropping signals (RS-14); widths are always explicit; units convert here
// and only here (D-19: f/c live toggle; SI internal).
package render

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Units selects display units (D-19: global, live-swappable).
type Units int

// Unit values. UnitF is the v0.1 default per the mocks.
const (
	UnitF Units = iota
	UnitC
)

// Opts carries the render context every primitive needs.
type Opts struct {
	Width int
	Units Units
	ASCII bool
	Frame int // animation phase (loading dots; ticked by the program loop)
}

// LoadingDots is the loading shimmer (UAT 18.2b): a 4-phase dot sweep shown
// where data has not arrived yet — "n/a" is reserved for data that is truly
// absent after load. Upstream candidate (M6): a go-studs spinner style.
func (o Opts) LoadingDots() string {
	frames := [4]string{"...", "\u00b7..", ".\u00b7.", "..\u00b7"}
	if o.ASCII {
		frames = [4]string{"...", " ..", ". .", ".. "}
	}
	return frames[((o.Frame%4)+4)%4]
}

// Temp renders a Celsius value in the display units; nil renders n/a.
func (o Opts) Temp(c *float64) string {
	if c == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%.0fºC", *c)
	}
	return fmt.Sprintf("%.0fºF", *c*9/5+32)
}

// Distance renders a kilometres value in the DIST column's fixed "nnn km"
// slot (miles under ºF, following Height); blank when unknown.
func (o Opts) Distance(km *float64) string {
	if km == nil {
		return ""
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%3.0f km", *km)
	}
	return fmt.Sprintf("%3.0f mi", *km*0.621371)
}

// TideHeight renders a metres value at tide precision (tenths of a foot
// under ºF, centimetres under ºC — UAT 61) in a fixed 4-cell numeric slot,
// so a negative low ("-0.1 ft") never shifts the column (UAT 62).
func (o Opts) TideHeight(m *float64) string {
	if m == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%4.2f m", *m)
	}
	return fmt.Sprintf("%4.1f ft", *m*3.28084)
}

// Knots renders a m/s current speed in knots — the convention under both
// unit systems (UAT 61) — in the same fixed 4-cell slot as TideHeight.
func (o Opts) Knots(mps *float64) string {
	if mps == nil {
		return "n/a"
	}
	return fmt.Sprintf("%4.1f kt", *mps/0.514444)
}

// Wind renders a m/s value in the display units (mph under ºF, km/h under ºC).
func (o Opts) Wind(mps *float64) string {
	if mps == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%.0f km/h", *mps*3.6)
	}
	return fmt.Sprintf("%.0f mph", *mps*2.23694)
}

// HealthGlyph renders one provider's header status (mock M-V1: ✔/⚠/✘ + name;
// glyph is textual, color additive elsewhere; ASCII fallback keeps the signal).
func (o Opts) HealthGlyph(name, status string) string {
	glyph := "✔"
	if status != snapshot.ProviderOK {
		glyph = "✘"
	}
	if o.ASCII {
		glyph = "OK"
		if status != snapshot.ProviderOK {
			glyph = "XX"
		}
	}
	fg := Tok(ProviderOK) // UAT 4.8: healthy provider reads green
	if status != snapshot.ProviderOK {
		fg = Tok(ProviderDown)
	}
	if status == snapshot.ProviderOff { // not a source right now (FIRMS without a key): neutral, never red (UAT 100)
		glyph, fg = "—", Tok(TextBase)
		if o.ASCII {
			glyph = "--"
		}
	}
	return rendering.WrapSGR(glyph+" "+name, fg)
}

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

// Verbatim header lines from mock M-V1 (09-view-mocks L27-28) — the mock is
// the spec, character-for-character (UAT ruling: "the mocks I provided is the
// output that I want exactly"); fidelity tests diff these against the file
// AND against the go-studs-rendered header.
const (
	mockGroupHeader = "           [   L O C A T I O N   /   S T A T I O N    ]     [           T O D A Y             ]     [    T O M O R R O W     ]"
	mockColHeader   = "  ♪        ###. NAME                 LABEL    ZIP      CONDITIONS    NOW       HI     LOW      CONDITIONS    HI      LOW"
)

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

// Color pass (HUM LEAD, UAT session 3): HI temps orange, LO temps cyan;
// chips bold-white on light grey; group headers carry 50%-muted backgrounds
// (terminal cells have no alpha - nearest 256-palette pastels). All styling
// flows through go-studs rendering.WrapSGR, which no-ops when color is off
// (NO_COLOR / pipes / tests), so text signals never depend on color (R-12a).

// sgrRaw wraps text in a raw SGR sequence, gated by go-studs' color switch.
// The sanctioned constructor (SGR/ColorSequence) classifies bare numeric
// codes as 256-palette FOREGROUNDS only — background chips are outside its
// contract (upstream candidate M6: background support in ColorSequence).
// Until that lands, the seam emits the one escape shape it needs.
func sgrRaw(text, params string) string {
	if !colorOn() {
		return text
	}
	return "\x1b[" + params + "m" + text + "\x1b[0m"
}

// colorOn probes the go-studs color gate (single source of truth: NO_COLOR +
// tty detection, test-overridable via SetColorEnabledForTest).
func colorOn() bool { return rendering.ColorsEnabled() }

// TitleGradient renders the app title bold with the reference-CLI interpolated
// truecolor gradient (#DD51D6 -> #378FE9 -> #7CE3B3) — UAT 4.9, standing in
// until theming lands. Plain text when color is off.
func TitleGradient(text string) string {
	if !colorOn() {
		return text
	}
	runes := []rune(text)
	if len(runes) < 2 {
		return text
	}
	r1, g1, b1 := hexRGB(GradStart)
	r2, g2, b2 := hexRGB(GradMid)
	r3, g3, b3 := hexRGB(GradEnd)
	lerp := func(a, b int, t float64) float64 { return float64(a)*(1-t) + float64(b)*t }
	var b strings.Builder
	for i, ch := range runes {
		pos := float64(i) / float64(len(runes)-1)
		var r, g, bl float64
		if pos <= 0.5 {
			t := pos * 2
			r, g, bl = lerp(r1, r2, t), lerp(g1, g2, t), lerp(b1, b2, t)
		} else {
			t := (pos - 0.5) * 2
			r, g, bl = lerp(r2, r3, t), lerp(g2, g3, t), lerp(b2, b3, t)
		}
		fmt.Fprintf(&b, "\x1b[1;38;2;%d;%d;%dm%c", int(r), int(g), int(bl), ch)
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// TintDefault repaints all un-tinted text in the base grey (UAT 4.10): the
// frame opens with the base foreground and every SGR reset re-arms it, so
// explicitly-colored spans keep their colors and everything else drops to
// grey 250 (~9:1 on black — comfortably AA). No-op with color off.
func TintDefault(s string) string {
	if !colorOn() {
		return s
	}
	base := "\x1b[0;38;5;" + Tok(TextBase) + "m"
	return base + strings.ReplaceAll(s, "\x1b[0m", base) + "\x1b[0m"
}

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
func rowMarks(r LocationRow, sel, play, alert string) [marksW]string {
	// UAT 110 mock: `›  ▶ 3◆ 2⚠ 001.` — 0 pointer · 1-2 spacers · 3 play ·
	// 4 spacer · 5 fire count · 6 ◆ · 7 spacer · 8 alert count · 9 ⚠ ·
	// 10 spacer.
	marks := [marksW]string{}
	for i := range marks {
		marks[i] = " "
	}
	if r.Selected {
		marks[0] = Tint(sel, Tok(FocusPointer)) // UAT 50.2: bold white pointer
	}
	if r.Playing {
		marks[3] = Tint(play, Tok(StatePlaying)) // UAT 80: the playing location wears a green ▶
		if r.Repeat {
			marks[3] = Tint(repeatGlyph(play), Tok(StatePlaying)) // UAT 83: ∞ on repeat
		}
	}
	if r.Fire > 0 { // B5 / UAT 110: fire is another alert kind — orange ◆ with its count (bold when a hotspot burns hard)
		glyph := "◆"
		if play == ">" { // --ascii (the play glyph tells)
			glyph = "*"
		}
		tone := Tok(FireMark)
		if r.FireHot {
			tone = "1;" + tone
		}
		marks[5] = Tint(fmt.Sprintf("%d", min(r.Fire, 9)), tone)
		marks[6] = Tint(glyph, tone)
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
	marks[9] = Tint(alert, tone)
	return marks
}

// repeatGlyph is the repeat mark: ∞, or "R" under --ascii (play tells which).
func repeatGlyph(play string) string {
	if play == ">" {
		return "R"
	}
	return "∞"
}

// rowData formats one LocationRow into the spec's cell values.
func (o Opts) rowData(l layout, r LocationRow) []string {
	sel, play, alert := "›", "▶", "⚠"
	if o.ASCII {
		sel, play, alert = ">", ">", "!"
	}
	marks := rowMarks(r, sel, play, alert)
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
	data = append(data, displayCondition(r.Conditions), o.temp5Or(r.Now, r.Loading)+trend)
	if l.hiLo {
		data = append(data, o.temp5Or(r.Hi, r.Loading), o.temp5Or(r.Lo, r.Loading))
	}
	if l.tomorrow {
		data = append(data, displayCondition(r.TomorrowConditions), o.temp5Or(r.TomorrowHi, r.Loading), o.temp5Or(r.TomorrowLo, r.Loading))
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
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: tableGutter}
	for _, r := range rows {
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: clampCells(o.rowData(l, r), cols), CellStyles: rowStyles(cols, r)})
	}
	dt := studs.NewDataTable(o.Width, def)
	out := []string{o.groupHeader(l, cols, o.Width), strings.TrimRight(dt.Header(), " ")}
	for _, line := range dt.Rows() {
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// rowStyles colors one row's cells (UAT 3.3/4.5/4.7): HIs orange, LOs cyan,
// n/a in the base grey, the focused row's name bold yellow - all applied by
// the component after padding, so alignment is untouched.
func rowStyles(cols []studs.ColumnDefinition, r LocationRow) map[int]string {
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
	m := map[int]string{}
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

// TrendGlyph renders a trend arrow in its muted tone (UAT 14.2) - the one
// owner for every view that shows ↗/↘ (tables, forecast modal, future).
func (o Opts) TrendGlyph(trend string) string {
	up, down := "↗", "↘"
	if o.ASCII {
		up, down = "^", "v"
	}
	switch trend {
	case "up":
		return Tint(up, Tok(TrendUp))
	case "down":
		return Tint(down, Tok(TrendDown))
	}
	return ""
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

// displayCondition maps provider condition strings to the mock's vocabulary
// ("P.CLOUDY"): underscores to spaces, PARTLY/MOSTLY abbreviated.
func displayCondition(c string) string {
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

// Band renders a full-width section band in the group-header chip style
// (UAT 43: the RECENT / SEARCHED separator becomes a band like the column
// groups). Bracketed form when color is off.
func Band(title, short string, width int, bg Token) string {
	if colorOn() {
		return sgrRaw(" "+centered(title, short, width-2)+" ", Tok(GroupText)+";"+Tok(bg))
	}
	return bracketTitle(title, short, width)
}

// centered fits a title into inner cells, degrading gracefully: full title,
// then its short form (UAT 16.1: "E X T E N D E D" instead of a truncated
// "EXTENDEDFORECA…"), then unspread lettering, then truncate.
func centered(title, short string, inner int) string {
	if displayWidth(title) > inner && short != "" {
		title = short
	}
	if displayWidth(title) > inner {
		title = strings.ReplaceAll(title, " ", "") // unspread the lettering
	}
	if displayWidth(title) > inner {
		title = truncate(title, max(0, inner))
	}
	pad := inner - displayWidth(title)
	return strings.Repeat(" ", pad/2) + title + strings.Repeat(" ", pad-pad/2)
}

// bracketTitle is the color-off band form: [ title ] across width cells.
func bracketTitle(title, short string, width int) string {
	return "[" + centered(title, short, width-2) + "]"
}

// Exported tint codes for module text (views compose, the seam styles).

// TintRaw wraps text in a full SGR parameter list (truecolor 38;2/48;2 and
// bold composites - the shapes go-studs WrapSGR cannot express), gated by
// the same color switch. Theme tokens with truecolor values ride this.
func TintRaw(text, params string) string { return sgrRaw(text, params) }

// Tint wraps text in a fg code (bare 256 or basic SGR; "1;"-prefixed for
// bold) through the go-studs gate - plain text when color is off.
func Tint(text, code string) string {
	if !colorOn() {
		return text
	}
	return rendering.WrapSGR(text, code)
}

// KeyCapIf renders a key chip in its enabled or muted state (UAT 21.1):
// a control that cannot act in the current model state reads at ~50%
// opacity. THE chip entry point for stateful controls - views pass their
// model-derived enabled flags (ELM: state in, view out) and every future
// control inherits the behavior.
func (o Opts) KeyCapIf(key string, enabled bool) string {
	if enabled {
		return o.KeyCap(key)
	}
	if o.ASCII || !colorOn() {
		return "[" + key + "]" // color-off keeps the textual affordance (RS-14)
	}
	return sgrRaw(" "+key+" ", Tok(KeyChipMuted))
}

// KeyCapWith renders a chip in an explicit tone (composite SGR token) —
// feedback states such as the volume blink (UAT 41). Color-off keeps [key].
func (o Opts) KeyCapWith(key string, tone Token) string {
	if o.ASCII || !colorOn() {
		return "[" + key + "]"
	}
	return sgrRaw(" "+key+" ", Tok(tone))
}

// KeyCap renders a key binding as a CLIAmp-style button: grey background,
// bold white text (UAT session 2B; upstream candidate M6: tui.KeyCap token).
// With color off (NO_COLOR, pipes, tests) it degrades to the mock's [key]
// form so the affordance never disappears (RS-14).
func (o Opts) KeyCap(key string) string {
	if o.ASCII {
		return "[" + key + "]"
	}
	if !colorOn() {
		return "[" + key + "]"
	}
	return sgrRaw(" "+key+" ", Tok(KeyChip))
}

// AlertBanner renders the M-V1 conditional alert module: severity ALWAYS a
// text label; paging "NN / MM" per the mock.
func (o Opts) AlertBanner(a snapshot.Alert, index, total int) string {
	warn := "⚠"
	if o.ASCII {
		warn = "!"
	}
	// Title carries the event (uppercased per mock); the severity label stays
	// a verbatim lowercase token — same shape as report mode ("ALERT [severe]")
	// so both surfaces read identically (R-12a).
	body := fmt.Sprintf("[%s] %s", a.Severity, a.Headline)
	pager := fmt.Sprintf("%02d / %02d Alerts", index, total)
	return o.Panel(warn+" "+strings.ToUpper(a.Event), body+"\n"+pager)
}

// Panel renders a rounded-border box with a title (mock anatomy), width-bound.
func (o Opts) Panel(title, content string) string { return o.PanelColored(title, content, "") }

// PanelColored is Panel with the border + title tinted (bare 256 fg code) —
// the alert module reads red or yellow by statement class (UAT 4.6).
func (o Opts) PanelColored(title, content, fg string) string {
	w := max(o.Width, 20)
	tl, tr, bl, br, hz, vt := "┌", "┐", "└", "┘", "─", "│" // square corners (UAT 10.5)
	if o.ASCII {
		tl, tr, bl, br, hz, vt = "+", "+", "+", "+", "-", "|"
	}
	tint := func(t string) string {
		if fg == "" {
			return t
		}
		return rendering.WrapSGR(t, fg)
	}
	var b strings.Builder
	head := tl + hz + hz + " " + title + " "
	if title == "" {
		head = tl // untitled panel: an unbroken top rule (About window, UAT 68)
	}
	pad := w - displayWidth(head) - 1
	if pad < 0 {
		pad = 0
	}
	b.WriteString(tint(head+strings.Repeat(hz, pad)+tr) + "\n")
	for _, line := range strings.Split(content, "\n") {
		line = truncate(line, w-4)
		b.WriteString(tint(vt) + "  " + line + strings.Repeat(" ", max(0, w-4-displayWidth(line))) + tint(vt) + "\n")
	}
	b.WriteString(tint(bl + strings.Repeat(hz, max(0, w-2)) + br))
	return b.String()
}

// AlertTone maps an alert to its display class (UAT 4.6): warning-grade
// statements read red, watch/advisory-grade read yellow. Returned as the
// bare 256 fg code PanelColored expects.
func AlertTone(event, severity string) string {
	if AlertIsWarning(event, severity) {
		return Tok(AlertDanger)
	}
	return Tok(AlertLabel)
}

// AlertIsWarning is THE warning-vs-advisory classifier (single owner:
// module tones, row-name tints, and future views all ride it).
func AlertIsWarning(event, severity string) bool {
	sev := strings.ToLower(severity)
	return sev == "severe" || sev == "extreme" || strings.Contains(event, "Warning")
}

// Block tones (UAT 5.4): borderless modules carry a ~10%-opacity background
// (truecolor - the 256 palette has no tints that dark) with a matching text
// tone. Dark-terminal variants; the light-bg variants land with theming.

// AlertBlockTone returns the fg/bg pair for an alert module (UAT 5.4a/b).
func AlertBlockTone(event, severity string) (fg, bg string) {
	if AlertIsWarning(event, severity) {
		return Tok(AlertWarnFG), Tok(AlertWarnBG)
	}
	return Tok(AlertAdvFG), Tok(AlertAdvBG)
}

// RadioBlockTone returns the fg/bg pair for the radio module (UAT 5.4c).
func RadioBlockTone() (fg, bg string) { return Tok(RadioFG), Tok(RadioBG) }

// BGVisible reports whether a tone paints a real background ("49" = the
// terminal default = hidden; theming toggles module chrome by value alone).
func BGVisible(bg string) bool { return bg != "" && bg != "49" }

// ModuleInnerWidth is the content width inside a module for its tone:
// visible-bg modules inset 3 cols each side (UAT 19.1a); hidden-bg modules
// run flush with the header edges (19.1b).
func (o Opts) ModuleInnerWidth(bg string) int {
	if BGVisible(bg) {
		return o.Width - 6
	}
	return o.Width
}

// Module renders a module block with the global inset policy (UAT 19.1):
// visible-bg tones get the 3-col left/right inset plus a padded blank line
// top and bottom; hidden-bg tones render flush, no padding lines — the
// single blank between modules comes from the layout (19.1c).
func (o Opts) Module(lines []string, fg, bg string) string {
	if !BGVisible(bg) {
		return o.Block(strings.Join(lines, "\n"), fg, bg)
	}
	padded := make([]string, 0, len(lines)+2)
	padded = append(padded, "")
	for _, l := range lines {
		padded = append(padded, "   "+l)
	}
	padded = append(padded, "")
	return o.Block(strings.Join(padded, "\n"), fg, bg)
}

// ModuleHeight is the rendered line count for a module of n content lines.
func ModuleHeight(n int, bg string) int {
	if BGVisible(bg) {
		return n + 2
	}
	return n
}

// Block renders content as a borderless full-width background block (UAT
// 5.4): every line is padded to the width and painted fg+bg; inner SGR
// resets (chips, temp colors) re-arm the block tone so the background never
// tears mid-line, and every line closes clean so it never bleeds past the
// block. With color off the content passes through untinted (R-12a: the
// module's text signals carry the meaning).
func (o Opts) Block(content, fg, bg string) string {
	lines := strings.Split(content, "\n")
	if !colorOn() {
		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " ")
		}
		return strings.Join(lines, "\n")
	}
	for i, line := range lines {
		line = PadTo(line, o.Width)
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0;"+fg+";"+bg+"m")
		lines[i] = "\x1b[" + fg + ";" + bg + "m" + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}

// WrapSegments greedily packs help segments into width-bound lines (UAT
// 6.7: the key-binding footer wraps smartly with terminal width). ANSI-aware.
func WrapSegments(segs []string, width int, sep string) string {
	var lines []string
	cur := ""
	for _, seg := range segs {
		if cur == "" {
			cur = seg
			continue
		}
		if displayWidth(cur)+displayWidth(sep)+displayWidth(seg) > width {
			lines = append(lines, cur)
			cur = seg
			continue
		}
		cur += sep + seg
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return strings.Join(lines, "\n")
}

// WrapText word-wraps plain prose to width-bound lines (UAT 15.2: alert
// bodies wrap, never truncate). Single owner - modals and modules share it.
func WrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var lines []string
	cur := ""
	for _, word := range strings.Fields(text) {
		switch {
		case cur == "":
			cur = word
		case displayWidth(cur)+1+displayWidth(word) > width:
			lines = append(lines, cur)
			cur = word
		default:
			cur += " " + word
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// WrapLines wraps every over-wide body line to width, preserving each
// line's leading indent on continuations (blank lines pass through). THE
// modal-content guarantee: floating windows wrap, never truncate — callers
// cannot reintroduce the truncation class of bug (UAT 25).
func WrapLines(lines []string, width int) []string {
	if width < 8 {
		width = 8
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if displayWidth(line) <= width {
			out = append(out, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		for _, w := range WrapText(strings.TrimLeft(line, " "), width-displayWidth(indent)) {
			out = append(out, indent+w)
		}
	}
	return out
}

// PadBetween left+right-justifies two strings within width (ANSI-aware — key
// chips and styled cells measure by display cells, not runes).
func PadBetween(left, right string, width int) string {
	gap := width - displayWidth(left) - displayWidth(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// Window + floating-tile backgrounds (UAT 12.4): the base viewport sits on
// a near-black; the blue-grey pair belongs to floating modal tiles only.

// WindowBG returns the viewport background for the terminal's mode (the
// dashboard passes bubbletea's BackgroundColorMsg.IsDark verdict).
func WindowBG(dark bool) color.Color {
	if dark {
		return lipgloss.Color(Tok(WindowBGDark))
	}
	return lipgloss.Color(Tok(WindowBGLight))
}

// ModalTone returns the fg/bg pair for floating modal tiles (UAT 12.4:
// the blue-grey tile treatment; text stays the base grey).
func ModalTone(dark bool) (fg, bg string) {
	if dark {
		return Tok(ModalFG), Tok(ModalBGDark)
	}
	return Tok(ModalFG), Tok(ModalBGLight)
}

// ScrollPanel renders a floating panel whose body windows to maxLines with
// a visible right-edge scroll rail (UAT 10.4: users must SEE that it
// scrolls on short terminals); when everything fits, it expands and the
// rail disappears.
func (o Opts) ScrollPanel(title string, lines []string, scroll, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 1
	}
	if len(lines) <= maxLines {
		return o.PanelColored(title, strings.Join(lines, "\n"), "")
	}
	maxScroll := len(lines) - maxLines
	scroll = max(0, min(scroll, maxScroll))
	win := make([]string, maxLines)
	inner := o.Width - 7 // panel chrome (4) + rail col + gap
	thumb := 1
	if maxLines > 3 {
		thumb = 1 + scroll*(maxLines-3)/max(1, maxScroll)
	}
	for i := range maxLines {
		glyph := "│"
		switch i {
		case 0:
			glyph = "▲"
		case maxLines - 1:
			glyph = "▼"
		case thumb:
			glyph = "█"
		}
		win[i] = PadTo(truncate(lines[scroll+i], inner), inner) + " " + glyph
	}
	return o.PanelColored(title, strings.Join(win, "\n"), "")
}

// Overlay floats modal centered over base via lipgloss v2 Canvas/Layer
// compositing (UAT 8.3: the '?' help floats over the dashboard instead of
// replacing it). The modal's own cells - spaces included - overwrite the
// base, so the panel is opaque.
func Overlay(base, modal string, termWidth int) string {
	// v2.0.2 note: Layer.Draw ignores X/Y (positioning lives in the
	// Compositor) and Layer.Width/Height are unset fields - measure with
	// lipgloss.Width/Height and composite through the Compositor. Centering
	// uses the TERMINAL width (UAT 35.4): the base's widest line is not the
	// viewport, so a long row must never shove the modal off-center.
	if termWidth <= 0 {
		termWidth = lipgloss.Width(base)
	}
	x := max(0, (termWidth-lipgloss.Width(modal))/2)
	y := max(0, (lipgloss.Height(base)-lipgloss.Height(modal))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
	).Render()
}

// Width is the ANSI-aware display width of s (views size compact rows).
func Width(s string) int { return displayWidth(s) }

// PadTo right-pads a line to exactly width display cells (ANSI-aware).
// Exported for the recent section's scroll rail: PadBetween's minimum-1 gap
// pushed the rail glyph right on rows that already filled the row length
// (UAT 6.6 off-by-one).
func PadTo(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-displayWidth(s)))
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR sequences (width math + tests).
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// Plain is the boundary for text that arrives from outside — relay titles
// and names, provider headlines and product text (red-team 0.9.0 S-F6):
// escape sequences and control characters are dropped so nothing a server
// sends can address the terminal (OSC hyperlinks, clipboard writes). Tabs
// and newlines survive; everything else below 0x20, and 0x7f–0x9f, goes.
func Plain(s string) string {
	s = stripANSI(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}

// displayWidth measures terminal cells (runewidth; AI-9 glyph policy).
func displayWidth(s string) int { return runewidth.StringWidth(stripANSI(s)) }

// truncate hard-limits a line to the given display width.
func truncate(s string, w int) string {
	if w <= 0 || displayWidth(s) <= w {
		return s
	}
	return runewidth.Truncate(stripANSI(s), w, "…")
}
