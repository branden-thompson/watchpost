package tty

// detail_marine.go — the marine rows of Location Details (buoys, tides, currents, swell). Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// MARITIME grid (UAT 63/64/66 mock): labels on the modal's shared value
// column (col 14, like CURRENTLY/TODAY), a first sub-column of 8
// (direction / trend / time / phase), a fixed 4-cell number + unit, and
// the provenance notes in ONE column 2 cells past the section's widest
// value (UAT 66/67 — scannable, yet never further right than the data
// needs, so the section never pushes toward the scroll rail). Every row
// fits the details modal's 78-cell wrap budget with its section prefix.
//
//	Observed      39m 22s ago
//	Conditions    Slight Chop
//	Water Temp    75°F             (buoy 46224, 11 mi)
//	Swell         SSW      3.0 ft  (period 14 s)
//	Tide          Rising   3.7 ft  (La Jolla, 24 mi)
//	Next High     19:40    5.7 ft
//	Next Low      02:49   -0.1 ft
//	Currents      Flood    1.4 kt  (Slack 16:05)
const (
	marLabelW  = colVal
	marFirstW  = 8
	marNoteGap = 2                                                                 // UAT 67: 2 cells past the widest value (Los Angeles-length names)
	marNumW    = 7                                                                 // "%4.1f ft"
	marNoteMax = 78 - detailPrefixW - marLabelW - marFirstW - marNumW - marNoteGap // 32: wrap budget at the 85-col modal floor after the widest value
)

// marineCell is one MARITIME row before layout: label, value, note.
type marineCell struct{ label, value, note string }

// marineRow collects one row for layoutMarine.
func marineRow(label, value, note string) marineCell { return marineCell{label, value, note} }

// layoutMarine lays the rows through the kit: label, value, note.
//
// IT WAS THIS TABLE WRITTEN BY HAND — a pass for the widest value, a PadTo for
// the label, a PadTo to the note column. That pass IS fit, and the kit does it
// now: the note column starts marNoteGap past the widest value in the section
// (UAT 67), which is what a fit value column plus a two-cell prefix means.
//
// GUTTER 0: the gaps are in the widths and the cells, because this section's
// label column is the report's own (colVal) and its note gap is a ruling.
//
// THE VALUE IS ONE COLUMN, not the two that marinePair draws inside it. Most
// rows carry a pair — a compass and a height, a time and a height — but
// "Moderate Chop" and "9 mph" are single values, and splitting the column would
// size the second one against a first that half the rows do not have.
func layoutMarine(o render.Opts, cells []marineCell, inner int) []string {
	cols := []render.StatusColumn{
		{Width: marLabelW, NoGutter: true}, // the row's name
		{NoGutter: true},                   // the value, whole
		{NoGutter: true},                   // the provenance or the second fact
	}
	rows := make([]render.StatusRow, 0, len(cells))
	for _, c := range cells { // bounded by the section's rows (P10-02)
		note := c.note
		if note != "" {
			note = strings.Repeat(" ", marNoteGap) + note
		}
		rows = append(rows, render.StatusRow{Cells: []string{c.label, c.value, note}})
	}
	return o.DetailTable(cols, rows, inner, 0)
}

// marinePair is the two-part value: first sub-column + fixed-width number.
func marinePair(first, num string) string { return render.PadTo(first, marFirstW) + num }

// maritimeRows renders the coastal-waters section in the mock's scan order
// (UAT 29/32/61/63): observation age, sea state, water temperature, swells,
// then tides and currents.
func maritimeRows(o render.Opts, m *snapshot.Marine, tz *time.Location, now time.Time, cw int) []string {
	rows := []marineCell{}
	if !m.ObservedAt.IsZero() && m.Buoy != "" {
		rows = append(rows, marineRow("Observed", fixedAgeTrim(now.Sub(m.ObservedAt))+" ago", ""))
	}
	if m.WaveHeight != nil {
		rows = append(rows, marineRow("Conditions", render.SeaState(*m.WaveHeight), ""))
	}
	if m.WaterTemp != nil {
		rows = append(rows, marineRow("Water Temp", strings.TrimSpace(o.Temp(m.WaterTemp)), buoyNote(o, m)))
	}
	rows = append(rows, swellRows(o, m)...)
	rows = append(rows, tideRows(o, m, tz, now)...)
	laid := layoutMarine(o, rows, cw-detailRailGutter)
	out := make([]string, 0, len(laid))
	for i, r := range laid {
		label := ""
		if i == 0 {
			label = "MARINE"
		}
		out = append(out, detailRow(label, r))
	}
	return out
}

// buoyNote is the "(buoy id, distance)" provenance note.
func buoyNote(o render.Opts, m *snapshot.Marine) string {
	if m.Buoy == "" {
		return ""
	}
	note := "(buoy " + render.PlainLine(m.Buoy)
	if d := strings.TrimSpace(o.Distance(m.BuoyDistanceKM)); d != "" {
		note += ", " + d // display units, one formatter (UAT 60.2)
	}
	return note + ")"
}

// swellRows: primary/secondary swell with direction + period, wind waves,
// and the buoy wind.
func swellRows(o render.Opts, m *snapshot.Marine) []marineCell {
	var rows []marineCell
	if h := render.FirstOf(m.SwellHeight, m.WaveHeight); h != nil {
		rows = append(rows, marineRow("Swell", marinePair(compass(m.SwellDirDeg), o.TideHeight(h)), period(m.WavePeriod)))
	}
	if m.SecondarySwellHeight != nil {
		rows = append(rows, marineRow("Swell 2", marinePair(compass(m.SecondarySwellDirDeg), o.TideHeight(m.SecondarySwellHeight)), period(m.SecondaryPeriod)))
	}
	if m.WindWaveHeight != nil {
		rows = append(rows, marineRow("Wind Waves", marinePair("", o.TideHeight(m.WindWaveHeight)), ""))
	}
	if m.WindSpeed != nil {
		gust := ""
		if m.WindGust != nil {
			gust = "(gusts " + o.Wind(m.WindGust) + ")"
		}
		rows = append(rows, marineRow("Buoy Wind", o.Wind(m.WindSpeed), gust))
	}
	return rows
}

// tideRows renders tides and currents (UAT 61/63, NOAA CO-OPS): trend from
// the next predicted event, one row per next high / low, local hh:mm.
func tideRows(o render.Opts, m *snapshot.Marine, tz *time.Location, now time.Time) []marineCell {
	var rows []marineCell
	nh, nl := render.NextTide(m.Tides, "H", now), render.NextTide(m.Tides, "L", now)
	if m.TideStation != "" || m.TideLevel != nil || nh != nil || nl != nil {
		val := render.TideTrend(nh, nl)
		if m.TideLevel != nil {
			val = marinePair(val, o.TideHeight(m.TideLevel))
		}
		rows = append(rows, marineRow("Tide", val, stationNote(o, m.TideStation, m.TideStationKM)))
	}
	if nh != nil {
		rows = append(rows, marineRow("Next High", tideEvent(o, nh, tz), ""))
	}
	if nl != nil {
		rows = append(rows, marineRow("Next Low", tideEvent(o, nl, tz), ""))
	}
	if row, ok := currentRow(o, m.Currents, tz, now); ok {
		rows = append(rows, row)
	}
	return rows
}

// tideEvent: "19:40    5.7 ft" (local time, fixed-width height).
func tideEvent(o render.Opts, e *snapshot.TideEvent, tz *time.Location) string {
	return marinePair(o.Clock.Time(e.Time.In(tz)), o.TideHeight(&e.Height))
}

// tideTrend reads the direction from whichever extreme comes next.

// currentRow: the phase in force (last predicted extreme before now, with
// its max speed) and the next predicted event as the note.
func currentRow(o render.Opts, events []snapshot.CurrentEvent, tz *time.Location, now time.Time) (marineCell, bool) {
	var cur, next *snapshot.CurrentEvent
	for i := range events {
		if !events[i].Time.After(now) {
			cur = &events[i]
		} else if next == nil {
			next = &events[i]
		}
	}
	if cur == nil && next == nil {
		return marineCell{}, false
	}
	val, note := "Slack", ""
	if cur != nil && cur.Type != "slack" {
		val = marinePair(titleWord(cur.Type), o.Knots(&cur.Speed))
	}
	if next != nil {
		note = "(" + titleWord(next.Type) + " " + o.Clock.Time(next.Time.In(tz)) + ")"
	}
	return marineRow("Currents", val, note), true
}

// stationNote is the "(name, distance)" provenance note for the tide row;
// the name is cut at its parenthetical qualifier and capped so the note
// fits the grid's note budget.
func stationNote(o render.Opts, name string, km *float64) string {
	name = render.PlainLine(name) // a station name is provider text (NFR-6, R5-C-05)
	if name == "" {
		return ""
	}
	if i := strings.Index(name, " ("); i > 0 {
		name = name[:i]
	}
	dist := strings.TrimSpace(o.Distance(km))
	room := marNoteMax - 2 // parentheses
	if dist != "" {
		room -= len(dist) + 2
	}
	note := "(" + truncateTo(name, room)
	if dist != "" {
		note += ", " + dist
	}
	return note + ")"
}

func titleWord(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// period formats a wave period for the secondary column.
func period(s *float64) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("(period %.0f s)", *s)
}

// fixedAgeTrim is fixedAge without the alignment padding.
func fixedAgeTrim(d time.Duration) string { return strings.TrimSpace(fixedAge(d)) }

// compass maps degrees true to a 16-point heading.
func compass(deg *float64) string {
	if deg == nil {
		return "--"
	}
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return pts[geo.CompassIndex(*deg, 16)]
}
