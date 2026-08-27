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
//	Water Temp    75ºF             (buoy 46224, 11 mi)
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

// layoutMarine lays the rows on the grid: notes share one column,
// marNoteGap cells past the widest value in the section (UAT 67).
func layoutMarine(cells []marineCell) []string {
	widest := 0
	for _, c := range cells {
		widest = max(widest, render.Width(c.value))
	}
	noteCol := marLabelW + widest + marNoteGap
	out := make([]string, 0, len(cells))
	for _, c := range cells {
		line := render.PadTo(c.label, marLabelW) + c.value
		if c.note != "" {
			line = render.PadTo(line, noteCol) + c.note
		}
		out = append(out, line)
	}
	return out
}

// marinePair is the two-part value: first sub-column + fixed-width number.
func marinePair(first, num string) string { return render.PadTo(first, marFirstW) + num }

// maritimeRows renders the coastal-waters section in the mock's scan order
// (UAT 29/32/61/63): observation age, sea state, water temperature, swells,
// then tides and currents.
func maritimeRows(o render.Opts, m *snapshot.Marine, tz *time.Location, now time.Time) []string {
	rows := []marineCell{}
	if !m.ObservedAt.IsZero() && m.Buoy != "" {
		rows = append(rows, marineRow("Observed", fixedAgeTrim(now.Sub(m.ObservedAt))+" ago", ""))
	}
	if m.WaveHeight != nil {
		rows = append(rows, marineRow("Conditions", seaState(*m.WaveHeight), ""))
	}
	if m.WaterTemp != nil {
		rows = append(rows, marineRow("Water Temp", strings.TrimSpace(o.Temp(m.WaterTemp)), buoyNote(o, m)))
	}
	rows = append(rows, swellRows(o, m)...)
	rows = append(rows, tideRows(o, m, tz, now)...)
	laid := layoutMarine(rows)
	out := make([]string, 0, len(laid))
	for i, r := range laid {
		label := ""
		if i == 0 {
			label = "MARITIME"
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
	note := "(buoy " + m.Buoy
	if d := strings.TrimSpace(o.Distance(m.BuoyDistanceKM)); d != "" {
		note += ", " + d // display units, one formatter (UAT 60.2)
	}
	return note + ")"
}

// swellRows: primary/secondary swell with direction + period, wind waves,
// and the buoy wind.
func swellRows(o render.Opts, m *snapshot.Marine) []marineCell {
	var rows []marineCell
	if h := firstOf(m.SwellHeight, m.WaveHeight); h != nil {
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
	nh, nl := nextTide(m.Tides, "H", now), nextTide(m.Tides, "L", now)
	if m.TideStation != "" || m.TideLevel != nil || nh != nil || nl != nil {
		val := tideTrend(nh, nl)
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
	return marinePair(e.Time.In(tz).Format("15:04"), o.TideHeight(&e.Height))
}

// tideTrend reads the direction from whichever extreme comes next.
func tideTrend(nh, nl *snapshot.TideEvent) string {
	switch {
	case nh != nil && (nl == nil || nh.Time.Before(nl.Time)):
		return "Rising"
	case nl != nil:
		return "Falling"
	}
	return ""
}

// nextTide is the first event of a type after now (events are time-ordered).
func nextTide(events []snapshot.TideEvent, typ string, now time.Time) *snapshot.TideEvent {
	for i := range events {
		if events[i].Type == typ && events[i].Time.After(now) {
			return &events[i]
		}
	}
	return nil
}

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
		note = "(" + titleWord(next.Type) + " " + next.Time.In(tz).Format("15:04") + ")"
	}
	return marineRow("Currents", val, note), true
}

// stationNote is the "(name, distance)" provenance note for the tide row;
// the name is cut at its parenthetical qualifier and capped so the note
// fits the grid's note budget.
func stationNote(o render.Opts, name string, km *float64) string {
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

// firstOf returns the first non-nil value.
func firstOf(vals ...*float64) *float64 {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
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

// seaState words a significant wave height (Douglas sea-state bands).
func seaState(m float64) string {
	switch {
	case m < 0.1:
		return "Calm (glassy)"
	case m < 0.5:
		return "Smooth"
	case m < 1.25:
		return "Slight Chop"
	case m < 2.5:
		return "Moderate Chop"
	case m < 4:
		return "Rough"
	}
	return "Very Rough"
}
