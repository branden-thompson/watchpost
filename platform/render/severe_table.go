package render

// severe_table.go — the Severe Weather / Disaster Events browse table on
// go-studs DataTable (0.13.0, SAM-D-26 AX-5): the six-column layout the mock
// pins (02-analysis/mocks/mock.py), its width ladder, the bracketed spaced-
// letter headers, and the one-column scroll rail every scrolling table in the
// app shares (Railify). The seam owns the layout; modes/tty supplies the
// cells and composes the window around it.

import (
	"fmt"
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// SevereCell is one browse row as the window supplies it: the four data
// cells, the row's absolute number, and its marks.
type SevereCell struct {
	Num                                           int // 1-based, absolute within the tab
	Event, Location, Detection, Declared, Expires string
	Focused                                       bool // the › pointer (the focused row)
	Playing                                       bool // the green ▶: this event is the one being read over the radio (UAT items 11–12)
}

// SevereTableTokens are the tokens the table paints on the category tints —
// the contrast gate iterates exactly these (NFR-14): white-family tokens,
// because the muted table greys and the alert reds fall below AA on the
// tints (R3-B-03). The column headers are group bands on their own lifted
// tint (SevereHeaderTone), checked separately by the same test.
func SevereTableTokens() []Token {
	return []Token{AlertModalText, ModalTitle, FocusPointer, StatePlaying}
}

// severeHeader paints one column header as the dashboard paints its group
// headers (Band): bold white on the band tone with colour on, the bracketed
// title with colour off — one style, not a new one. EVENT is the group
// header of the marks + number + event span and keeps the spaced lettering
// of a group band; the data columns read as plain column titles (HUM LEAD
// UAT 2026-08-28, items 7–8).
func severeHeader(name string, w int, spread bool, bg string) string {
	title := name
	if spread {
		title = strings.Join(strings.Split(name, ""), " ")
	}
	if colorOn() {
		return sgrRaw(" "+centered(title, "", w-2)+" ", severeHeaderFG(bg)+";"+bg)
	}
	return bracketTitle(title, "", w)
}

// severeHeaderFG is the band text for a band tone: bold pure white (fixed,
// like the dark themes' tints — a theme's ModalTitle is tuned for its own
// tile, not the band) unless the theme's ModalTitle reads better on it —
// Watchpost Light, whose tints are pale and whose title is dark.
func severeHeaderFG(bg string) string {
	if Contrast(Tok(ModalTitle), bg) > Contrast("1;97", bg) {
		return Tok(ModalTitle)
	}
	return "1;97"
}

// severeHeaderLift is how far the header band's tone is lifted from the
// category hue toward white: a lighter tint of the same hue, distinct from
// the key chips and the tile, themed with the hue token (HUM LEAD UAT
// 2026-08-28, item 9). Bold white on it clears AA on every hue
// (TestCategoryToneContrastAA).
const severeHeaderLift = 0.2

// SevereHeaderTone is the header band's background for a category hue: the
// hue lifted toward white when it is dark, toward black when it is light
// (a light hue lifted lighter loses the band's white text — UAT 2026-08-28,
// the brighter Watches / Statements tints), so the band reads as a distinct
// tint of the same hue either way.
func SevereHeaderTone(hue Token) string {
	r, g, b, ok := bgRGB(Tok(hue))
	if !ok {
		return Tok(hue) // not truecolor (a theme's 256-index hue): the band is the hue itself, never a white slab (round 4, B-17)
	}
	if 0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(b) > 96 { // a light hue (perceived brightness past ~37 %)
		return mixBG("48;2;0;0;0", Tok(hue), severeHeaderLift)
	}
	return mixBG("48;2;255;255;255", Tok(hue), severeHeaderLift)
}

// severeHeaderLine composes the header row: the EVENT band over the marks,
// number and event columns, then each data column's band — the bands MEET
// in the gutters, each taking half, as the dashboard's group bands do (HUM
// LEAD UAT 2026-08-28, item 10); the rows below keep their gutters (10b).
func severeHeaderLine(ev int, data []severeCol, bg string, ascii bool) string {
	half := severeGutter / 2
	widths := []int{severeMarksW + severeNumW + ev}
	names, spread := []string{"EVENT"}, []bool{!ascii} // plain EVENT under --ascii: a screen reader spells a spread word letter by letter (FR-13, HUM LEAD ruling 2026-08-29)
	for _, c := range data {
		widths = append(widths, c.width)
		names, spread = append(names, c.name), append(spread, false)
	}
	var b strings.Builder
	for i, w := range widths {
		if i > 0 {
			w += half // the left half of the gutter before it
		}
		if i < len(widths)-1 {
			w += severeGutter - half // the right half of the gutter after it
		}
		b.WriteString(severeHeader(names[i], w, spread[i], bg))
	}
	return b.String()
}

// The mock's fixed widths: marks 5 · number 5 · LOCATION 22 (16 when
// squeezed) · DECLARED 15 · EXPIRES 15; gutter 4 between the data columns
// (2 in the mock; HUM LEAD UAT 2026-08-28 item 13).
// (Marks were 7 with a severity glyph; HUM LEAD dropped the glyph at UAT
// 2026-08-28 — every row in a tab wore the same one — and EVENT took the cells.)
const (
	severeMarksW     = 5
	severeNumW       = 5
	severeLocW       = 22
	severeDetectW    = 17 // "Spotter Reported" (UAT 2026-08-28)
	severeLocMinW    = 16
	severeTimeW      = 15
	severeGutter     = 4
	severeEventMin   = 22 // the EVENT column must keep at least this before a column drops
	severeEventFloor = 14
)

type severeCol struct {
	name  string
	width int
}

// severeColumns is the width ladder (NFR-11): EXPIRES drops first, then
// DETECTION, then DECLARED, then LOCATION squeezes to 16 — EVENT keeps ≥ 22
// cols until nothing is left to drop.
func severeColumns(inner int) (event int, cols []severeCol) {
	cols = []severeCol{{"LOCATION", severeLocW}, {"DETECTION", severeDetectW}, {"DECLARED", severeTimeW}, {"EXPIRES", severeTimeW}}
	ev := func() int {
		w := inner - severeMarksW - severeNumW
		for _, c := range cols {
			w -= severeGutter + c.width
		}
		return w
	}
	for _, drop := range []string{"EXPIRES", "DETECTION", "DECLARED"} {
		if ev() >= severeEventMin {
			break
		}
		cols = without(cols, drop)
	}
	if ev() < severeEventMin {
		cols = []severeCol{{"LOCATION", severeLocMinW}}
	}
	return max(severeEventFloor, ev()), cols
}

// without drops the named column, keeping the order of the rest.
func without(cols []severeCol, name string) []severeCol {
	out := cols[:0:0]
	for _, c := range cols {
		if c.name != name {
			out = append(out, c)
		}
	}
	return out
}

// severeMarks is the 5-col marks cell: the bold-white › on the focused row,
// the green ▶ on the row being read over the radio (as the dashboard's
// playing location wears it — UAT 80), blank otherwise. Tinted here, so the
// two marks keep their own tones inside one cell.
func severeMarks(o Opts, c SevereCell) string {
	g := o.Glyphs()
	ptr, play := " ", " "
	if c.Focused {
		ptr = Tint(g.Pointer, Tok(FocusPointer))
	}
	if c.Playing {
		play = Tint(g.Play, Tok(StatePlaying))
	}
	return ptr + "  " + play + " "
}

// SevereTable renders the header and the rows for a body `inner` columns
// wide, every line exactly inner wide for inner ≥ 66 (the 80-col floor; the
// ladder bottoms out at EVENT 14 + LOCATION 16 below inner 44 — outside the
// spec, pinned by TestSevereColumnsLadder). Cells are truncated with the
// ellipsis before the kit lays them out, so the kit never wraps.
func (o Opts) SevereTable(cells []SevereCell, inner int, hue Token) []string {
	ev, data := severeColumns(inner)
	header, text := Tok(ModalTitle), Tok(AlertModalText)
	cols := []studs.ColumnDefinition{
		{Name: "marks", Width: severeMarksW},
		{Name: "num", Width: severeNumW, NoLeadingGutter: true},
		{Name: "event", Width: ev, NoLeadingGutter: true},
	}
	for _, c := range data {
		cols = append(cols, studs.ColumnDefinition{Name: strings.ToLower(c.name), Width: c.width})
	}
	def := &studs.DataTableDefinition{Columns: cols, GutterWidth: severeGutter, NoAutoStyle: true}
	for _, c := range cells {
		row := []string{severeMarks(o, c), fmt.Sprintf("%03d. ", c.Num), c.Event}
		vals := map[string]string{"LOCATION": c.Location, "DETECTION": c.Detection, "DECLARED": c.Declared, "EXPIRES": c.Expires}
		for _, dc := range data {
			row = append(row, vals[dc.name])
		}
		row = clampCells(row, cols)
		styles := map[int]string{}
		for i := range row {
			styles[i] = text
		}
		delete(styles, 0) // the marks cell carries its own tones
		if c.Focused {
			styles[2] = header
		}
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: row, CellStyles: styles})
	}
	dt := studs.NewDataTable(inner, def)
	out := []string{PadTo(severeHeaderLine(ev, data, SevereHeaderTone(hue), o.ASCII), inner)} // the kit lays the rows out; the header is the group-band line
	for _, line := range dt.Rows() {
		out = append(out, PadTo(line, inner))
	}
	return out
}

// RailGlyphs is a scroll rail's glyph set; RailGlyphsFor picks the mock's
// forms per --ascii.
type RailGlyphs struct{ Up, Down, Thumb, Bar string }

func RailGlyphsFor(ascii bool) RailGlyphs {
	if ascii {
		return RailGlyphs{"^", "v", "#", "|"}
	}
	return RailGlyphs{"▲", "▼", "█", "│"}
}

// Railify appends the one-column scroll rail to lines: each padded to
// width-1 then the bar, with the thumb on the line that tracks the scroll
// position (lo of total, window visible) — the thumb ranges over ALL the
// lines given, so a caller that draws ▼ on its last visible row passes the
// rows above it (and total-1 / window-1). PadTo, not PadBetween: a full-width
// line must never push the rail right (UAT 6.6). Callers draw ▲/▼ themselves.
func Railify(lines []string, width, lo, total, window int, g RailGlyphs) []string {
	thumb := 0
	if maxLo := total - window; maxLo > 0 && window > 1 {
		thumb = lo * (window - 1) / maxLo
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		glyph := g.Bar
		if i == thumb {
			glyph = g.Thumb
		}
		out[i] = PadTo(line, width-1) + glyph
	}
	return out
}
