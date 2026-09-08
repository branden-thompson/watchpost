package tty

// detail.go — the Location Details modal: currently / today / forecast rows and their layout. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// Detail view (location-detail-mock.txt): labeled section rows with a
// divider column, 10-day forecast, alert blocks with the mock's bullet
// rules. MARITIME renders only when marine data exists — the marine
// provider (NWS coastal-waters / Open-Meteo Marine) is queued work.
const detailLabelW = 10

// detailPrefixW is the width of a detail row's section prefix
// ("{LABEL:10} │ "): the single owner every row-budget derives from.
const detailPrefixW = detailLabelW + 3

// detailRow renders one "{LABEL} │ {content}" line (label right-aligned;
// empty label for continuations). No lead (UAT 65): the section label
// column starts flush with the modal's header label; the freed cells are
// spacing on the right.
func detailRow(o render.Opts, label, content string) string {
	// UAT 30: 1-col breathing room each side of the divider (was 3) - the
	// reclaimed width goes to the right gutter beside the scroll rail.
	return fmt.Sprintf("%*s %s %s", detailLabelW, label, o.Glyphs().Rail, content)
}

func (d Dashboard) detailLines() []string {
	loc := d.selectedLocation()
	if loc == nil {
		return []string{"No location selected."}
	}
	o := d.opts()
	// Rows must fit INSIDE the modal wrap budget (width-7): WrapLines
	// collapses interior spacing on over-wide lines, which would tear the
	// divider column. 15 = the detailRow chrome left of the content.
	cw := min(o.Width, d.modalWidth()) - 7 - detailPrefixW
	lines := []string{""}
	lines = append(lines, d.currentlyRows(o, loc, cw)...)
	lines = append(lines, detailRow(o, "", ""))
	lines = append(lines, d.todayRows(o, loc, cw)...)
	lines = append(lines, detailRow(o, "", ""))
	lines = append(lines, d.forecastRows(o, loc, cw)...)
	if loc.Marine != nil {
		lines = append(lines, detailRow(o, "", ""))
		lines = append(lines, maritimeRows(o, loc.Marine, locTZ(loc), d.now(), cw)...) // coastal locations only (UAT 29)
	}
	lines = append(lines, detailRow(o, "", ""))
	lines = append(lines, fireRows(o, loc, d.now(), d.fireBoldMW(), d.cfg.FireRadiusKm, d.cfg.FireIncidentRadiusKm, cw)...) // B5: fire is another alert kind
	lines = append(lines, detailRow(o, "", ""))
	lines = append(lines, seismicRows(o, loc, d.now(), cw, d.seismicLookbackDays())...) // 0.11.0: earthquakes are another alert kind
	lines = append(lines, alertBlocks(o, loc, min(o.Width, d.modalWidth())-11)...)
	// UAT 101: one consolidated chip row; + / − Watchlist enabled by membership.
	controls := o.KeyCap("↑↓") + " Scroll  " + o.KeyCap("esc") + " Close  " +
		o.KeyCapIf("ctrl+a", d.canAddFocused()) + " + Watchlist  " +
		o.KeyCapIf("shift+del", d.canRemoveFocused()) + " − Watchlist"
	return append(lines, "", controls)
}

// Content column grid (UAT 32): labels at 0, primary values at colVal
// (the FORECAST condition column), secondary values at forecastHiLoCol so
// CURRENTLY / TODAY / FORECAST / MARITIME share two vertical scan lines.
const colVal = 14

// gridRow places a label, primary value, and optional secondary value on
// the grid (values never overrun: a long primary pushes the secondary).
func gridRow(label, primary, secondary string) string {
	line := render.PadTo(label, colVal) + primary
	if secondary != "" {
		line = render.PadTo(line, forecastHiLoCol) + secondary
	}
	return line
}

// currentlyRows: condition + temp/trend; feels-like + delta; humidity
// aligned to the HIGH/LOW column.
func (d Dashboard) currentlyRows(o render.Opts, loc *snapshot.Location, cw int) []string {
	h := loc.Harmonized
	temp := render.Tint(strings.TrimSpace(o.Temp(h.Temp)), render.Tok(render.TextBright)) + o.TrendGlyph(trend(*loc))
	out := []string{detailRow(o, "CURRENTLY", gridRow(prettyCond(h.Condition), temp, ""))}
	// WHERE THE NUMBER CAME FROM, WHEN IT DID NOT COME FROM A STATION.
	//
	// A location with no observing station within twenty miles is filled from
	// the NWS hourly grid for its own point (UAT 2026-09-05). Saying so does
	// two jobs: it sets the expectation that this is modelled rather than
	// measured, and it explains why the rows below — feels like, station,
	// distance — are simply absent rather than broken.
	//
	// It reads as an aside because it is one: the number above is the most
	// accurate available, not a degraded one.
	if h.Source.Provider != "" && render.PlainLine(h.Source.ModelOrStation) == "" {
		out = append(out, detailRow(o, "", render.Italic("Data from NWS hourly grid forecast for this location")))
	}
	// THE SECOND LABEL IS A COLUMN, not a string with spaces in it (0.15.0).
	//
	// These rows carry two label/value pairs each — feels-like and humidity,
	// station and distance — and the second pair was built as
	// "Humidity  :  %.0f%%", a hand-laid column inside a cell. The two rows
	// lined up only because "Humidity  :" and "Distance  :" happen to be the
	// same length; a third pair of a different width would have gone unnoticed
	// until someone read it.
	var grid []render.StatusRow
	if h.Feels != nil && h.Temp != nil {
		hum := []string{"", ""}
		if h.HumidityPct != nil {
			hum = []string{"Humidity  :", secondGap + fmt.Sprintf("%.0f%%", *h.HumidityPct)}
		}
		grid = append(grid, render.StatusRow{Cells: append([]string{"Feels Like",
			fmt.Sprintf("%s   (%+.0f°F)", strings.TrimSpace(o.Temp(h.Feels)), (*h.Feels-*h.Temp)*9/5)}, hum...)})
	} else if h.HumidityPct != nil {
		grid = append(grid, render.StatusRow{Cells: []string{"", "", "Humidity  :", secondGap + fmt.Sprintf("%.0f%%", *h.HumidityPct)}})
	}
	// UAT 60.2: the observing station and its distance live here at every
	// width — the table's WX STN / DIST columns surface them only when there
	// is room; drilling in one level always reaches them.
	st := render.PlainLine(h.Source.ModelOrStation) // a provider name never addresses the terminal (NFR-6, R5-C-05)
	if st != "" {
		dist := []string{"", ""}
		if v := strings.TrimSpace(o.Distance(h.Source.DistanceKm)); v != "" {
			dist = []string{"Distance  :", secondGap + v}
		}
		grid = append(grid, render.StatusRow{Cells: append([]string{"Station   :", st}, dist...)})
	}
	out = append(out, currentlyGrid(o, grid, cw)...)
	if st != "" {
		// NOT YOUR LOCAL STATION (HUM LEAD, UAT 2026-09-05).
		//
		// Lone Pine read 86 °F at half past six because the observation came
		// from Death Valley, 110 km away — real, current, and not this
		// location's weather. Beyond twenty miles the provider now refuses the
		// reading outright; between ten and twenty it is used, and the listener
		// is told, because a measurement from the far side of a ridge is a
		// different microclimate and the number may simply not be theirs.
		//
		// It says "may vary" rather than naming a doubt it cannot quantify: how
		// wrong the reading is depends on terrain this app does not model.
		if d := h.Source.DistanceKm; d != nil && *d > render.StationFarKM {
			out = append(out, detailRow(o, "", render.Italic(render.Tint("This is not your local station - actual temp may vary", render.Tok(render.NameWarning)))))
		}
	}
	return out
}

// secondGap is the air between the second pair's label and its value. It lives
// in the CELL because every column here has the report's own geometry and takes
// no gutter of its own.
const secondGap = "  "

// currentlyGrid draws CURRENTLY's two-pair rows: the report's own label column
// (colVal), a value column that reaches the column every secondary starts at
// (forecastHiLoCol), and the second pair beyond it.
//
// GUTTER 0: those two columns are the report's, shared with every other
// section, so the gaps are in the widths rather than in a uniform gutter.
func currentlyGrid(o render.Opts, rows []render.StatusRow, cw int) []string {
	cols := []render.StatusColumn{
		{Width: colVal, NoGutter: true},                   // the row's name
		{Width: forecastHiLoCol - colVal, NoGutter: true}, // its value
		{NoGutter: true},                                  // the second name
		{NoGutter: true},                                  // and its value
	}
	out := make([]string, 0, len(rows))
	for _, l := range o.DetailTable(cols, rows, cw-detailRailGutter, 0) { // bounded by the rows (P10-02)
		out = append(out, detailRow(o, "", l))
	}
	return out
}

// todayRows: today's condition + HIGH/LOW, sunrise/sunset in local time.
func (d Dashboard) todayRows(o render.Opts, loc *snapshot.Location, cw int) []string {
	if len(loc.Daily) == 0 {
		return []string{detailRow(o, "TODAY", o.LoadingDots())}
	}
	_ = cw
	day := loc.Daily[0]
	out := []string{detailRow(o, "TODAY", gridRow(prettyCond(day.Condition), "", hiLo(o, day)))}
	tz := time.Local
	if z, err := zones.Location(loc.TZ); err == nil {
		tz = z
	}
	if !day.Sunrise.IsZero() {
		out = append(out, detailRow(o, "", gridRow("Sunrise:", o.Clock.Time(day.Sunrise.In(tz))+"  Local Time", "")))
	}
	if !day.Sunset.IsZero() {
		out = append(out, detailRow(o, "", gridRow("Sunset :", o.Clock.Time(day.Sunset.In(tz))+"  Local Time", "")))
	}
	return append(out, d.hourlyRows(o, loc, tz, cw)...)
}

// hourlyWindow is how many hours the section shows, rolling from now.
//
// TWELVE, AND CONSTANT: half a day is enough to plan around, and a fixed count
// keeps the section — and the sections under it — in the same place all day.
const hourlyWindow = 12

// hourlyRows is the coming hours, hour by hour (HUM LEAD, 2026-09-07).
//
// The heading carries the count rather than the constant, because a feed short
// of twelve periods should say what it actually has.
//
// AN EMPTY LIST DRAWS NOTHING. The hourly tier hydrates on demand (UAT 72: 162
// KB per location), so a RECENT row has none until it is opened — and a heading
// over no rows reads as a fault rather than as a fetch that has not happened.
func (d Dashboard) hourlyRows(o render.Opts, loc *snapshot.Location, tz *time.Location, cw int) []string {
	hrs := nextHours(loc.Hourly, d.now(), tz, hourlyWindow)
	if len(hrs) == 0 {
		return nil
	}
	local := d.now().In(tz)
	rows := make([]render.StatusRow, 0, len(hrs))
	for _, h := range hrs { // bounded by the hours left in the day (P10-02)
		pp := " --%"
		if h.PrecipProb != nil {
			pp = fmt.Sprintf("%3.0f%%", *h.PrecipProb)
		}
		// A ROLLING WINDOW CROSSES MIDNIGHT, so the day is named when it turns
		// over: without it the column reads 22:00, 23:00, 00:00, 01:00 and a
		// reader takes the times as going backwards into this morning.
		// THE DAY IS NAMED WHEN IT TURNS OVER: without it the column reads
		// 22:00, 23:00, 00:00, 01:00 and a reader takes the times as going
		// backwards into this morning.
		at := h.Time.In(tz)
		day := ""
		if at.Day() != local.Day() {
			day = at.Format("Mon")
		}
		rows = append(rows, render.StatusRow{Cells: []string{
			day, o.Clock.Time(at), "    " + render.DisplayCondition(h.Condition),
			"(" + pp + ")", "   " + strings.TrimSpace(o.Temp(h.Temp)),
		}})
	}
	out := []string{detailRow(o, "", ""), detailRow(o, "", gridRow(fmt.Sprintf("Next %d Hours:", len(hrs)), "", ""))}
	for _, l := range o.DetailTable(hourCols(), rows, cw-detailRailGutter, 0) { // bounded by the table (P10-02)
		out = append(out, detailRow(o, "", l))
	}
	return out
}

// nextHours is a ROLLING WINDOW of the coming hours, from the hour the listener
// is standing in (HUM LEAD, 2026-09-07).
//
// A WINDOW, NOT "THE REST OF TODAY", and the reason is the layout: the rest of
// today is 23 rows at one in the morning and one row at eleven at night, so the
// section — and everything below it — moves down the report all day. A fixed
// window is the same height whenever it is read.
//
// NOT THE WHOLE FEED EITHER: NWS /forecast/hourly returns about 156 periods and
// nothing caps them on the way in, so "every hour available" is a 156-row table
// inside a section headed TODAY.
//
// FROM THE CURRENT HOUR, not the next one: the hour someone is standing in is
// the one they are asking about, and the feed's period for it is the forecast
// for the rest of it.
func nextHours(hrs []snapshot.Hourly, now time.Time, tz *time.Location, want int) []snapshot.Hourly {
	local := now.In(tz)
	from := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, tz)
	out := make([]snapshot.Hourly, 0, want)
	for _, h := range hrs { // bounded by the feed's periods (P10-02)
		if h.Time.In(tz).Before(from) {
			continue
		}
		if out = append(out, h); len(out) == want {
			break
		}
	}
	return out
}

// whenTail is the three columns the hours and the days share, and the reason
// they scan as one column instead of as two tables that happen to be adjacent
// (HUM LEAD, 2026-09-07). The alternative was tuning one to the other by eye,
// which holds until either changes.
//
// THE GAPS ARE IN THE WIDTHS, and these tables are drawn with no gutter of
// their own, because the columns have to land where the report's OTHER sections
// already put theirs: CURRENTLY's value shares the condition column (colVal),
// and every HIGH/LOW pair in the report starts at forecastHiLoCol. A uniform
// gutter cannot reproduce four, one and three.
func whenTail() []render.StatusColumn {
	return []render.StatusColumn{
		{Width: colVal + 3, NoGutter: true},     // 4 spaces + the condition's 13
		{Width: 7, Right: true, NoGutter: true}, // 1 space + "( nn%)"
		{NoGutter: true},                        // 3 spaces + the temperature, or the pair
	}
}

// hourCols is whenTail under a DAY and an HOUR (HUM LEAD, 2026-09-07).
//
// THE DAY IS ITS OWN COLUMN so that the hour does not move when it appears: a
// rolling window crosses midnight, and a day name folded into the time cell
// pushed every column right on that one row. Blank on every other row, four
// cells wide against the kit's one of gutter, which is the two the mock draws
// between "Tue" and the hour.
//
// FOUR AND FIVE AND THE GUTTER MAKE TEN — the width the days' date occupies —
// so the condition column is in the same place in both tables.
func hourCols() []render.StatusColumn {
	return append([]render.StatusColumn{
		{Width: 5, NoGutter: true}, // the day, when it turns over
		{Width: 5, NoGutter: true}, // the hour
	}, whenTail()...)
}

// dayCols is whenTail under a date.
func dayCols() []render.StatusColumn {
	return append([]render.StatusColumn{{Width: 10, NoGutter: true}}, whenTail()...)
}

// forecastHiLoCol is the content column where every HIGH/LOW pair starts
// (date 10 + 4 + cond 13 + 1 + "(nnn%)" 6 + 3): TODAY aligns to it so the
// pairs scan as one column (UAT 28.1).
const forecastHiLoCol = 37

// hiLo renders "HIGH  98°F /  98°F LOW" with fixed 5-cell temps so 2- and
// 3-digit values stay aligned (UAT 28.2).
func hiLo(o render.Opts, day snapshot.Daily) string {
	return fmt.Sprintf("HIGH %5s / %5s LOW", o.Temp(day.TempMax), o.Temp(day.TempMin))
}

// forecastRows: up to 10 upcoming days with precip probability, through the
// same table the hours above are drawn with (whenCols).
func (d Dashboard) forecastRows(o render.Opts, loc *snapshot.Location, cw int) []string {
	var rows []render.StatusRow
	for i, day := range loc.Daily { // bounded at ten days below (P10-02)
		if i == 0 {
			continue // today has its own section
		}
		if i > 10 {
			break
		}
		pp := " --%"
		if day.PrecipProb != nil {
			pp = fmt.Sprintf("%3.0f%%", *day.PrecipProb)
		}
		date := day.Date
		if t, err := time.Parse("2006-01-02", day.Date); err == nil {
			date = t.Format("01/02/2006")
		}
		rows = append(rows, render.StatusRow{Cells: []string{
			date, "    " + render.DisplayCondition(day.Condition), "(" + pp + ")", "   " + hiLo(o, day),
		}})
	}
	if len(rows) == 0 {
		return []string{detailRow(o, "FORECAST", o.LoadingDots())}
	}
	var out []string
	label := "FORECAST"
	for _, l := range o.DetailTable(dayCols(), rows, cw-detailRailGutter, 0) { // bounded by the days (P10-02)
		out = append(out, detailRow(o, label, l))
		label = ""
	}
	return out
}

// locTZ resolves a location's zone for local-time rows (local fallback).
func locTZ(loc *snapshot.Location) *time.Location {
	if z, err := zones.Location(loc.TZ); err == nil && loc.TZ != "" {
		return z
	}
	return time.Local
}
