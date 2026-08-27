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
func detailRow(label, content string) string {
	// UAT 30: 1-col breathing room each side of the divider (was 3) - the
	// reclaimed width goes to the right gutter beside the scroll rail.
	return fmt.Sprintf("%*s │ %s", detailLabelW, label, content)
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
	lines = append(lines, d.currentlyRows(o, loc)...)
	lines = append(lines, detailRow("", ""))
	lines = append(lines, d.todayRows(o, loc, cw)...)
	lines = append(lines, detailRow("", ""))
	lines = append(lines, d.forecastRows(o, loc)...)
	if loc.Marine != nil {
		lines = append(lines, detailRow("", ""))
		lines = append(lines, maritimeRows(o, loc.Marine, locTZ(loc), time.Now())...) // coastal locations only (UAT 29)
	}
	lines = append(lines, detailRow("", ""))
	lines = append(lines, fireRows(o, loc, time.Now(), d.fireBoldMW())...) // B5: fire is another alert kind
	lines = append(lines, alertBlocks(loc, min(o.Width, d.modalWidth())-11)...)
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
func (d Dashboard) currentlyRows(o render.Opts, loc *snapshot.Location) []string {
	h := loc.Harmonized
	temp := render.Tint(strings.TrimSpace(o.Temp(h.Temp)), render.Tok(render.TextBright)) + o.TrendGlyph(trend(*loc))
	out := []string{detailRow("CURRENTLY", gridRow(prettyCond(h.Condition), temp, ""))}
	feels, hum := "", ""
	if h.Feels != nil && h.Temp != nil {
		feels = fmt.Sprintf("%s   (%+.0fºF)", strings.TrimSpace(o.Temp(h.Feels)), (*h.Feels-*h.Temp)*9/5)
	}
	if h.HumidityPct != nil {
		hum = fmt.Sprintf("Humidity  :  %.0f%%", *h.HumidityPct)
	}
	if feels != "" || hum != "" {
		label := "Feels Like"
		if feels == "" {
			label = ""
		}
		out = append(out, detailRow("", gridRow(label, feels, hum)))
	}
	// UAT 60.2: the observing station and its distance live here at every
	// width — the table's WX STN / DIST columns surface them only when there
	// is room; drilling in one level always reaches them.
	if st := h.Source.ModelOrStation; st != "" {
		dist := ""
		if d := strings.TrimSpace(o.Distance(h.Source.DistanceKm)); d != "" {
			dist = "Distance  :  " + d
		}
		out = append(out, detailRow("", gridRow("Station   :", st, dist)))
	}
	return out
}

// todayRows: today's condition + HIGH/LOW, sunrise/sunset in local time.
func (d Dashboard) todayRows(o render.Opts, loc *snapshot.Location, cw int) []string {
	if len(loc.Daily) == 0 {
		return []string{detailRow("TODAY", o.LoadingDots())}
	}
	_ = cw
	day := loc.Daily[0]
	out := []string{detailRow("TODAY", gridRow(prettyCond(day.Condition), "", hiLo(o, day)))}
	tz := time.Local
	if z, err := zones.Location(loc.TZ); err == nil {
		tz = z
	}
	if !day.Sunrise.IsZero() {
		out = append(out, detailRow("", gridRow("Sunrise:", day.Sunrise.In(tz).Format("1504")+"  Local Time", "")))
	}
	if !day.Sunset.IsZero() {
		out = append(out, detailRow("", gridRow("Sunset :", day.Sunset.In(tz).Format("1504")+"  Local Time", "")))
	}
	return out
}

// forecastHiLoCol is the content column where every HIGH/LOW pair starts
// (date 10 + 4 + cond 13 + 1 + "(nnn%)" 6 + 3): TODAY aligns to it so the
// pairs scan as one column (UAT 28.1).
const forecastHiLoCol = 37

// hiLo renders "HIGH  98ºF /  98ºF LOW" with fixed 5-cell temps so 2- and
// 3-digit values stay aligned (UAT 28.2).
func hiLo(o render.Opts, day snapshot.Daily) string {
	return fmt.Sprintf("HIGH %5s / %5s LOW", o.Temp(day.TempMax), o.Temp(day.TempMin))
}

// forecastRows: up to 10 upcoming days with precip probability.
func (d Dashboard) forecastRows(o render.Opts, loc *snapshot.Location) []string {
	out := []string{}
	label := "FORECAST"
	for i, day := range loc.Daily {
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
		row := fmt.Sprintf("%s    %-13s (%s)   ", date, truncateTo(render.DisplayCondition(day.Condition), 13), pp) + hiLo(o, day)
		out = append(out, detailRow(label, row))
		label = ""
	}
	if len(out) == 0 {
		out = append(out, detailRow("FORECAST", o.LoadingDots()))
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
