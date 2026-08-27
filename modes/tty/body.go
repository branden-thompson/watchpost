package tty

// body.go — the header, the two location tables and their empty states. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// header is the two-line masthead (UAT 102 mock, redesigned for narrow
// terminals and any number of APIs):
//
//	W A T C H P O S T  <version>     Updated: <stamp>     API: ✔8 ⚠0 ✘0 /  8  [S] Status
//	[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit
//
// The stamp is centred in the gap between the title and the API summary;
// the total reserves two columns for growth. Narrow terminals shorten the
// stamp (time only, then no label, then gone — it lives in [S]) before the
// row could overflow; the header never exceeds the width.
func (d Dashboard) header(o render.Opts) string {
	title := render.TitleGradient("W A T C H P O S T") + "  v" + d.cfg.Version // UAT 4.9 gradient; UAT 41: no pipe
	api := d.apiSummary(o) + "  " + o.KeyCap("S") + " Status"                  // UAT 24.2
	stamps := []string{"awaiting first data..."}
	if d.snap != nil {
		at := dataAsOf(d.snap).Local()
		stamps = []string{"Updated: " + at.Format("01/02/2006 15:04:05 MST"), "Updated: " + at.Format("15:04:05"), at.Format("15:04:05")}
	}
	if render.Width(title)+render.Width(api)+2 > o.Width { // very narrow: the chip alone, then no label
		api = d.apiSummary(o) + " " + o.KeyCap("S")
	}
	if render.Width(title)+render.Width(api)+2 > o.Width {
		api = strings.TrimPrefix(d.apiSummary(o), "API: ") + " " + o.KeyCap("S")
	}
	gap := o.Width - render.Width(title) - render.Width(api)
	line1 := render.PadBetween(title, api, o.Width)
	for _, stamp := range stamps {
		if w := render.Width(stamp); w+2 <= gap {
			pad := (gap - w) / 2
			line1 = title + strings.Repeat(" ", pad) + stamp + strings.Repeat(" ", gap-pad-w) + api
			break
		}
	}
	line2 := o.KeyCap("s") + " Setup  " + o.KeyCap("a") + " About  " + o.KeyCap("t") + " Theme  " + o.KeyCap("?") + " Help  " + o.KeyCap("q") + " Quit" // UAT 56/57/100/102
	return line1 + "\n" + line2
}

// apiSummary counts the providers by health: ✔ ok · ⚠ degraded but has
// served (stale) · ✘ degraded and never served (down); providers that are
// not a source right now (off) count in none of them, and a provider that
// has not answered yet counts in the total only (the three never sum past
// it; a shortfall means "still loading"). The total is the active set,
// padded to two columns (UAT 102: "reserve 2 col for growth").
func (d Dashboard) apiSummary(o render.Opts) string {
	ok, stale, down, total := 0, 0, 0, 0
	for _, p := range providersOf(d.snap) {
		switch {
		case p.Status == snapshot.ProviderOff:
			continue
		case p.Status == snapshot.ProviderOK && p.FetchedAt.IsZero():
			// registered, not yet answered (the first frames): in the total, not yet ✔ (REVIEW C5)
		case p.Status == snapshot.ProviderOK:
			ok++
		case p.FetchedAt.IsZero():
			down++
		default:
			stale++
		}
		total++
	}
	gOK, gStale, gDown := "✔", "⚠", "✘"
	if o.ASCII {
		gOK, gStale, gDown = "OK", "!!", "XX"
	}
	return fmt.Sprintf("API: %s %s%d %s%d /%3d",
		render.Tint(gOK+strconv.Itoa(ok), render.Tok(render.ProviderOK)),
		render.Tint(gStale, render.Tok(render.AlertLabel)), stale,
		render.Tint(gDown, render.Tok(render.ProviderDown)), down, total)
}

// body is the frame below the header: radio module, alert area, the
// control row, then the two tables from the memo (Q3).
func (d Dashboard) body(fl frameLayout) string {
	priority, recent := d.tables(fl)
	var b strings.Builder
	d.writeBody(&b, fl, priority, recent)
	return b.String()
}

// writeBody writes the body into the frame's own buffer (Q3: no
// intermediate string).
func (d Dashboard) writeBody(b *strings.Builder, fl frameLayout, priority, recent string) {
	b.WriteString(d.radioPanel(fl)) // UAT 8.1: radio module first
	b.WriteString("\n\n")
	b.WriteString(d.alertArea(fl)) // then the alert area (blanks per UAT 6.1/6.2)
	b.WriteString("\n\n")
	// UAT 26/43: controls live where they act - ABOVE the watchlist's group
	// labels now (right-aligned to the table edge).
	b.WriteString(fl.controlRow)
	b.WriteByte('\n')
	b.WriteString(priority)
	b.WriteByte('\n')
	b.WriteString(recent)
}

// priorityTable is the watchlist table, or its empty state (UAT 104).
func (d Dashboard) priorityTable(fl frameLayout) string {
	o := fl.o
	if d.snap == nil || len(d.snap.Locations) == 0 {
		return emptyState(o.TableRowLen(fl.days), watchlistEmpty, "") // the empty state stands where the table will
	}
	rows := make([]render.LocationRow, 0, len(d.snap.Locations))
	for i := range d.snap.Locations {
		rows = append(rows, d.row(i, &d.snap.Locations[i], d.selected < d.numPriority() && i == d.selected))
	}
	return o.LocationTable(rows, fl.days)
}

// controlRow is the watchlist's control line (UAT 56): [enter] Details,
// [ctrl+a] Add, [shift+del] Remove, [l] Lookup with [↑↓] Navigate right-
// aligned when the row fits; on narrow terminals it smart-wraps instead
// (same WrapSegments as the footer) so no row exceeds the width.
func (d Dashboard) controlRow(o render.Opts) string {
	segs := []string{
		o.KeyCap("enter") + " Details", // UAT 57
		o.KeyCap("ctrl+a") + " Add",
		o.KeyCapIf("shift+del", d.selected < d.numPriority() && d.numPriority() > 0) + " Remove",
		o.KeyCap("l") + " Lookup",
	}
	nav := o.KeyCap("↑↓") + " Navigate"
	line := strings.Join(segs, "   ")
	if render.Width(line)+render.Width(nav)+2 <= o.Width {
		return render.PadBetween(line, nav, o.Width)
	}
	return render.WrapSegments(append(segs, nav), o.Width, "   ")
}

// sharedExtDays pins ONE extended-forecast column count for the priority
// and recent tables (UAT session 3.1: if extended columns display in
// priority, recent must immediately match).
func (d Dashboard) sharedExtDays() int {
	days := 0
	for _, sn := range []*snapshot.Snapshot{d.snap, d.recent} {
		if sn == nil {
			continue
		}
		for _, loc := range sn.Locations {
			days = max(days, min(5, max(0, len(loc.Daily)-2)))
		}
	}
	return days
}

// recentSection renders the RECENT/SEARCHED table — seeded with the top-25
// US cities until real search history lands (UAT session 2A) — windowed to
// recentWindow rows with the mock's ▲│▼ scroll rail.
func (d Dashboard) recentSection(fl frameLayout) string {
	o, days := fl.o, fl.days
	var b strings.Builder
	// UAT 43/45: a full-width section band in the group-label style, no
	// blank lines around it; the rail's ▲ rides the band now that the
	// recent table shows rows only.
	rail := o.TableRowLen(days) + 2 // UAT 9.2: one blank col between the last cell and the rail
	band := render.Band("R E C E N T   /   S E A R C H E D", "R E C E N T", o.TableRowLen(days), render.GroupSectionBG)
	b.WriteString(render.PadTo(band, rail-1) + "▲\n")
	total, base := 0, 0
	if d.recent != nil {
		total = len(d.recent.Locations)
	}
	if d.snap != nil {
		base = len(d.snap.Locations)
	}
	if total == 0 {
		b.WriteString(emptyState(o.TableRowLen(days), recentEmpty, render.PadTo("", rail-o.TableRowLen(days)-1)+"▼") + "\n") // UAT 104 fallback; the rail closes on its last row
		return b.String()
	}
	window := fl.window
	lo := min(d.recentOff, max(0, total-window))
	hi := min(total, lo+window)
	rows := make([]render.LocationRow, 0, hi-lo)
	for i := lo; i < hi; i++ {
		r := d.row(i, &d.recent.Locations[i], base+i == d.selected) // focus spans both tables (UAT 4.4)
		r.Index = base + i + 1                                      // numbering continues after the priority rows (mock: 004.)
		rows = append(rows, r)
	}
	// UAT 44.1/45: the band connects the tables - the recent table renders
	// rows only (both header rows dropped; the watchlist's headers apply).
	table := strings.SplitN(o.LocationTable(rows, days), "\n", 3)[2]
	b.WriteString(railify(table, rail, lo, total, window) + "\n")
	showing := fmt.Sprintf("Showing %d-%d of %d locations", lo+1, hi, total)
	b.WriteString(render.PadBetween("", showing+"  ▼", rail) + "\n")
	return b.String()
}

// Empty states (UAT 104): three to five rows — a blank, the message centred
// on the table span and wrapped for narrow terminals, a blank — standing
// where the table will once a location is added or searched.
const (
	watchlistEmpty = "Run 's' Setup, 'l'ookup a location, or 'ctrl+a' a searched location to add to your Watchlist"
	recentEmpty    = "NO RECENT LOCATION SEARCHED or DATA-SEEDING FAILED"
	emptyWrapAt    = 64 // the mock's two-line break on wide terminals
)

// emptyState renders the block; tail rides the last row (the rail's ▼).
func emptyState(span int, text, tail string) string {
	lines := render.WrapText(text, max(10, min(span-4, emptyWrapAt)))
	out := []string{""}
	for _, l := range lines {
		pad := max(0, (span-render.Width(l))/2)
		out = append(out, strings.Repeat(" ", pad)+l)
	}
	out = append(out, "")
	if tail != "" {
		out[len(out)-1] = render.PadTo("", span) + tail
	}
	return strings.Join(out, "\n")
}

// railify appends the scroll rail: ▲ on the column-header line, │ on each
// row line with a █ thumb tracking the scroll position (UAT 11.3), ▼ on the
// Showing line.
func railify(table string, width, lo, total, window int) string {
	lines := strings.Split(table, "\n") // rows only (UAT 45); ▲ rides the band, ▼ the Showing line
	thumb := 0
	if maxLo := total - window; maxLo > 0 && window > 1 {
		thumb = lo * (window - 1) / maxLo
	}
	for i := range lines {
		glyph := "│"
		if i == thumb {
			glyph = "█"
		}
		// PadTo, not PadBetween: full-width rows must not push the rail right
		// (UAT 6.6 off-by-one vs the Showing line's ▼).
		lines[i] = render.PadTo(lines[i], width-1) + glyph
	}
	return strings.Join(lines, "\n")
}

// row converts a snapshot location to a table row.
func (d Dashboard) row(i int, loc *snapshot.Location, selected bool) render.LocationRow {
	row := render.LocationRow{
		Index: i + 1, Name: loc.Label, Tag: loc.Tag, Zip: loc.Zip,
		Station:    loc.Harmonized.Source.ModelOrStation,                                                           // WX STN / DIST (UAT 60)
		Playing:    d.radioPlaying && d.radioKey == snapshot.Key(snapshot.LocationRef{Lat: loc.Lat, Lon: loc.Lon}), // UAT 80
		Repeat:     d.radioRepeat != RepeatOff,                                                                     // UAT 83/93: ∞ when the row will come round again
		StationKM:  loc.Harmonized.Source.DistanceKm,
		Conditions: loc.Harmonized.Condition, // display mapping in the seam (P.CLOUDY etc)
		Now:        loc.Harmonized.Temp,
		Trend:      trend(*loc),
		HasAlert:   len(loc.Alerts) > 0,
		AlertCount: len(loc.Alerts),                            // ⚠ badge (UAT 20.2)
		Fire:       fireCount(loc.Fire),                        // B5 / UAT 110: n◆
		FireHot:    fireHot(loc.Fire.Hotspots, d.fireBoldMW()), // B5
		Selected:   selected,
		// UAT 18.2: shimmer until this location's data lands (obs or daily
		// still pending); post-load nils stay honest "n/a".
		Loading: rowLoading(loc),
	}
	for _, al := range loc.Alerts {
		if render.AlertIsWarning(al.Event, al.Severity) {
			row.WarnAlert = true // warning-grade outranks advisory (UAT 14.1)
			break
		}
	}
	if len(loc.Daily) > 0 {
		row.Hi, row.Lo = loc.Daily[0].TempMax, loc.Daily[0].TempMin
	}
	if len(loc.Daily) > 1 {
		row.TomorrowConditions = loc.Daily[1].Condition
		row.TomorrowHi, row.TomorrowLo = loc.Daily[1].TempMax, loc.Daily[1].TempMin
	}
	for _, day := range loc.Daily[min(2, len(loc.Daily)):] {
		if len(row.Extended) == 5 {
			break
		}
		row.Extended = append(row.Extended, render.DayCell{Date: mmdd(day.Date), Hi: day.TempMax, Lo: day.TempMin})
	}
	return row
}

// mmdd shortens an ISO date to the mock's mm/dd column header.
func mmdd(iso string) string {
	if len(iso) < 10 {
		return iso
	}
	return iso[5:7] + "/" + iso[8:10]
}

// trend derives the NOW arrow from the next forecast hour vs current temp
// (mock: 888ºF↗; ±0.3ºC deadband so noise doesn't flicker the arrow).
func trend(loc snapshot.Location) string {
	if loc.Harmonized.Temp == nil || len(loc.Hourly) == 0 || loc.Hourly[0].Temp == nil {
		return ""
	}
	delta := *loc.Hourly[0].Temp - *loc.Harmonized.Temp
	switch {
	case delta > 0.3:
		return "up"
	case delta < -0.3:
		return "down"
	}
	return ""
}
