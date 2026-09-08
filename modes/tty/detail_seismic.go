package tty

// detail_seismic.go — the SEISMIC rows of Location Details (0.11.0): recent
// nearby earthquakes from USGS, largest-then-nearest, each glyphed by
// felt-likelihood. Always present, so a quiet or cold feed is stated, never
// implied (the FIRE AsOf precedent, B5-P3). The map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// seismicLookbackDays is the window the header words, from the [seismic] rules
// (the one owner is the config; 0 = the ratified 7-day default).
func (d Dashboard) seismicLookbackDays() int {
	if d.cfg.SeismicDays > 0 {
		return d.cfg.SeismicDays
	}
	return 7
}

// seismicRows is the SEISMIC section of the detail modal (0.11.0): the recent
// quakes within the magnitude-graduated bands, each with its magnitude,
// distance + bearing, depth, age and felt-likelihood. cw is the content width
// (for the right-aligned attribution). lookbackDays words the window.
func seismicRows(o render.Opts, loc *snapshot.Location, now time.Time, cw, lookbackDays int) []string {
	ss := loc.Seismic
	if ss == nil || ss.AsOf.IsZero() { // no feed has answered (cold/down): "unavailable", never a fake "none" (FIRE AsOf precedent)
		return []string{detailRow("SEISMIC", seismicHead("seismic data unavailable", cw))}
	}
	n := len(ss.Quakes)
	if n == 0 { // the feed answered and nothing was in reach: the honest quiet answer
		return []string{detailRow("SEISMIC", seismicHead("no recent seismic activity", cw))}
	}
	head := fmt.Sprintf("%d nearby in the last %d days", n, lookbackDays)
	// The whole list (the provider caps it at 20, so this is bounded): the radio
	// broadcast reads only the strongest three and sends listeners here for the
	// rest (P4, HUM LEAD). Pre-sized: header + one row per quake.
	out := make([]string, 0, n+1)
	out = append(out, detailRow("SEISMIC", seismicHead(head, cw)))
	for _, l := range seismicTable(o, ss.Quakes, now, cw-detailRailGutter) { // bounded by the provider's cap of 20 (P10-02)
		out = append(out, detailRow("", l))
	}
	return out
}

// seismicTable is the quakes, through the kit (0.15.0).
//
// THE COLUMNS WERE PadTo LITERALS, which is the same table written by hand: the
// widths were counted once and every change since has had to re-count them, and
// a value one cell too long pushed the rest of its row right. FIT sizes each
// column to its own widest cell, so the table is as wide as what is in it.
//
// DISTANCE AND DIRECTION ARE TWO COLUMNS, as they are in the fire tables: a
// number aligns on its right edge and a heading on its left, and together in
// one cell neither does.
//
// THE FELT LABEL IS THE ONE TRUNCATABLE COLUMN. "Tsunami — Almost certainly
// felt" is 31 cells and the section has about 62 for everything; it is also the
// only column whose meaning survives being cut, because the words that matter
// are at the front. Everything left of it is a measurement, and half a
// measurement is worse than none.
func seismicTable(o render.Opts, qs []snapshot.Quake, now time.Time, inner int) []string {
	cols := []render.StatusColumn{
		{Width: 1},                       // the felt-likelihood glyph
		{},                               // the magnitude
		{Right: true},                    // how far
		{},                               // which way
		{Right: true},                    // how deep
		{Right: true},                    // how long ago
		{Truncatable: true, MinWidth: 8}, // what it would have felt like
	}
	rows := make([]render.StatusRow, 0, len(qs))
	for _, q := range qs { // bounded by the provider's cap of 20 (P10-02)
		glyph, label := seismicBand(o, q.Mag)
		tone := render.Tok(render.SeismicMark)
		if q.Tsunami || seismicAlertSevere(q.Alert) {
			tone = render.Tok(render.AlertDanger) // tsunami / orange-red PAGER: warning tone, any magnitude
			label = seismicWarnLabel(o, q, label)
		}
		rows = append(rows, render.StatusRow{Cells: []string{
			render.Tint(glyph, tone),
			fmt.Sprintf("M%.1f", q.Mag),
			strings.TrimSpace(o.Distance(&q.DistanceKm)),
			q.Bearing,
			fmt.Sprintf("depth %.0f km", q.DepthKm),
			seismicAge(now.Sub(q.At)),
			label,
		}})
	}
	return o.DetailTable(cols, rows, inner, render.DetailGutter)
}

// seismicHead renders the section header content with the "(USGS)" credit
// right-aligned to the content width (the mock's top-right attribution).
func seismicHead(head string, cw int) string {
	const attr = "(USGS)"
	head = gridRow(head, "", "") // start at the value column, like the other sections' heads
	target := cw - render.Width(attr)
	if target <= render.Width(head) {
		return head + "  " + attr // too narrow to right-align: keep a gap
	}
	return render.PadTo(head, target) + attr
}

// seismicRowLevel is the felt-band level the main-table row mark wears
// (0.11.0, HUM LEAD): the strongest recent quake's level, or 0 for none. The
// quakes are sorted largest-first, so the strongest is the first.
func seismicRowLevel(ss *snapshot.SeismicState) int {
	if ss == nil || len(ss.Quakes) == 0 {
		return 0
	}
	return render.SeismicLevel(ss.Quakes[0].Mag)
}

// seismicBand returns the felt-likelihood glyph and label for a magnitude. The
// glyph is the shared ramp (○●◉ / .oO — one owner in render.Glyphs, so the
// table mark and this section cannot disagree); the label carries the finer
// four-way felt wording the section shows (objectives §4).
func seismicBand(o render.Opts, mag float64) (glyph, label string) {
	glyph = o.Glyphs().Seismic[render.SeismicLevel(mag)-1]
	switch {
	case mag >= 5.0:
		label = "Significant"
	case mag >= 4.5:
		label = "Almost certainly felt"
	case mag >= 3.5:
		label = "Might feel it"
	default:
		label = "Below feeling"
	}
	return glyph, label
}

// seismicAlertSevere reports a PAGER alert level that warrants the warning
// tone (USGS levels green/yellow/orange/red — orange and red are the
// damaging ones).
func seismicAlertSevere(alert string) bool {
	switch strings.ToLower(alert) {
	case "orange", "red":
		return true
	}
	return false
}

// seismicWarnLabel prefixes the felt label with the reason it reads emphasised.
// The dash follows the glyph set: an em-dash normally, an ASCII hyphen under
// --ascii (so the seismic path never leaks a unicode character — A11-10).
func seismicWarnLabel(o render.Opts, q snapshot.Quake, label string) string {
	dash := " — "
	if o.ASCII {
		dash = " - "
	}
	if q.Tsunami {
		return "Tsunami" + dash + label
	}
	return "PAGER" + dash + label
}

// seismicAge words a quake's age coarsely (minutes / hours / days) — the
// window is up to a week, which the frame's fixed "Hh Mm" ages do not span.
func seismicAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
}
