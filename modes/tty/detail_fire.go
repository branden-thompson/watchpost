package tty

// detail_fire.go — the FIRE rows of Location Details (hotspots, incidents, marks). Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fireRows is the FIRE section of the detail modal (B5, HUM LEAD 2026-08-25:
// "fire is another alert type and can be a section in the location
// detail"): the hotspots inside the ring — nearest first, each with its
// bearing, distance, strength, satellite and age — the named incidents
// with acres and containment, and the fire-weather alert when one is
// active. Always present, so "none nearby" is said, never implied.
func fireRows(o render.Opts, loc *snapshot.Location, now time.Time, boldMW, ringKm, incidentKm float64, cw int) []string {
	fs := loc.Fire
	if fs.AsOf.IsZero() { // no fire feed has answered yet (cold launch, feeds down): never "none" (red-team B5 P3)
		return []string{detailRow(o, "FIRE", gridRow("Hotspots", "fire feed not yet available", ""))}
	}
	out := []string{detailRow(o, "FIRE", fireSectionHead(o, "Hotspots", ringKm))}
	out = append(out, rows(o, hotspotRows(o, loc, fs.Hotspots, now, boldMW, cw))...)
	out = append(out, detailRow(o, "", ""))
	out = append(out, detailRow(o, "", fireSectionHead(o, "Incidents", incidentKm)))
	out = append(out, rows(o, incidentRows(o, loc, fs.Incidents, now, cw))...)
	for _, a := range loc.Alerts {
		if ev := strings.ToLower(a.Event); strings.Contains(ev, "red flag") || strings.Contains(ev, "fire weather") {
			out = append(out, detailRow(o, "", gridRow("Fire Wx", render.Tint(render.Plain(a.Event), render.Tok(render.AlertDanger)), "")))
			break
		}
	}
	return out
}

// fireSectionHead is "Hotspots - Radius: 16 mi": the list's name and HOW FAR IT
// LOOKED, beside the list itself.
//
// THE RADIUS IS ON THE HEAD BECAUSE THERE ARE TWO OF THEM (UAT 2026-09-07). The
// section drew one label, "Hotspots", and listed satellite detections and named
// incidents under it — two feeds, two rings, one heading. So "none within the
// fire ring" read as a claim about the three named fires printed under it, one
// of them eleven miles away: *"Radio says none within your 16 mile fire ring -
// but there are 2 hotspots at 11 miles."* They were incidents, from the wider
// ring, and nothing on screen said so.
func fireSectionHead(o render.Opts, name string, km float64) string {
	if km <= 0 {
		return gridRow(name, "", "")
	}
	// PadTo the longer of the two names, so the two dashes line up down the
	// section exactly as the mock draws them.
	return render.PadTo(name, len("Incidents")+1) + "- Radius: " + strings.TrimSpace(o.Distance(&km))
}

// detailRailGutter is the air the tables leave on the right for the modal's
// vertical scroll control (HUM LEAD, 2026-09-07: the age column ran up against
// it). The detail report scrolls whenever it is longer than the window, which
// is nearly always, so the gutter is unconditional rather than a guess about
// whether the rail is drawn this frame.
const detailRailGutter = 3

// nearestFirst orders a list by distance, closest at the top (HUM LEAD,
// 2026-09-07).
//
// ON A COPY: the slice belongs to the published snapshot, which every other
// consumer reads, and sorting it in place would reorder the spoken report and
// the row badge from inside a render.
//
// A DISTANCE-LESS ENTRY SORTS LAST rather than first: an unknown distance is
// not zero miles, and a fire of unknown distance at the top of a list read
// closest-first is the one wrong place for it.
func nearestFirst[T any](in []T, km func(T) *float64) []T {
	out := append([]T(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := km(out[i]), km(out[j])
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		}
		return *a < *b
	})
	return out
}

// rows puts a table's lines in the detail's continuation column.
//
// NO EXTRA INDENT: the table's first column is one cell wide and carries the
// hotspot's ◆ or an incident's nothing, so both lists' names start in the same
// column — which is what the mock draws.
func rows(o render.Opts, lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines { // bounded by the table (P10-02)
		out = append(out, detailRow(o, "", l))
	}
	return out
}

// fireNone is what a list says when its ring admitted nothing. It names the
// RING, not "the fire ring", because there are two.
func fireNone(o render.Opts) []string {
	return []string{render.Tint("none within this radius", render.Tok(render.TableMuted))}
}

// hotspotRows: up to three hotspots nearest first, then "… and N more".
//
// THE CAP STAYS HERE AND NOT ON THE INCIDENTS, and the difference is what they
// are: there can be 300 hotspots (snapshot.MaxHotspots) and none of them has a
// name to tell it from the next, so three and a count is the whole of what a
// reader can use. A named fire is exactly what someone is asking about, and
// there are as many as the incident radius admits.
//
// A HOTSPOT HAS NO NAME AND NO ACRES. It is a satellite pixel: where it is, how
// hard it is radiating, which bird saw it, when, and how sure. Those are the
// columns, and the mock's NAME and ACRES have no source on this side.
func hotspotRows(o render.Opts, loc *snapshot.Location, hs []snapshot.Hotspot, now time.Time, boldMW float64, cw int) []string {
	if len(hs) == 0 {
		return fireNone(o)
	}
	hs = nearestFirst(hs, func(h snapshot.Hotspot) *float64 { return h.DistanceKm })
	// FIT, NOT FILL (HUM LEAD, 2026-09-07): every column is as wide as its own
	// widest cell, so the table pulls in to the width of what is in it rather
	// than stretching to the section's edge. DISTANCE AND DIRECTION ARE TWO
	// COLUMNS: a number aligns on its right edge and a heading on its left, and
	// together in one cell neither does.
	cols := []render.StatusColumn{
		{Width: 1},    // the ◆
		{Right: true}, // how far
		{},            // which way
		{Right: true}, // radiative power
		{},            // the satellite that saw it
		{Right: true, Truncatable: true, MinWidth: 6}, // how long ago
		{Right: true}, // how sure the feed is
	}
	var rows []render.StatusRow
	for i, h := range hs { // bounded by the cap below (P10-02)
		if i == 3 {
			more := "…"
			if o.ASCII {
				more = "..."
			}
			rows = append(rows, render.StatusRow{Cells: []string{"", "", fmt.Sprintf("%s and %d more", more, len(hs)-3), "", "", "", ""}})
			break
		}
		brg := geo.BearingDeg(loc.Lat, loc.Lon, h.Lat, h.Lon)
		strength, styles := "n/a MW", map[int]string{} // an unmeasured point (HMS GOES often)
		if h.FRPMW != nil {
			strength = fmt.Sprintf("%.0f MW", *h.FRPMW)
			if *h.FRPMW >= boldMW {
				styles[3] = "1;" + render.Tok(render.FireMark) // the FRP cell, which moved when the bearing left it
			}
		}
		age := "age n/a"
		if !h.DetectedAt.IsZero() { // a point without a time is not "2562047h" old (U2)
			age = fixedAgeTrim(now.Sub(h.DetectedAt))
		}
		rows = append(rows, render.StatusRow{Styles: styles, Cells: []string{
			render.Tint(fireGlyph(o), render.Tok(render.FireMark)),
			strings.TrimSpace(o.Distance(h.DistanceKm)),
			compass(&brg),
			strength,
			render.Plain(h.Source.ModelOrStation),
			age,
			render.Plain(h.Confidence),
		}})
	}
	return o.DetailTable(cols, rows, cw-detailRailGutter, render.DetailGutter)
}

// incidentRows lists EVERY named fire the row counts (UAT 2026-09-07).
//
// It broke at three and said nothing about the rest, so a location wearing 5◆
// listed three in the one place the app sends people for more detail — while
// the spoken report named all five. Three surfaces, three answers to "which
// fires are near me", and the most complete was the one you cannot re-read.
//
// AN INCIDENT HAS NO RADIATIVE POWER. It is a reported fire: its name, where it
// is, how big, how contained, and when it was found. The mock's MWRP column has
// no source on this side.
//
// THE NAME IS NOT TRUNCATED, by HUM LEAD ruling: *"this can not worry about
// that just like the USGS seismic section does not worry about it."*
func incidentRows(o render.Opts, loc *snapshot.Location, ins []snapshot.Incident, now time.Time, cw int) []string {
	if len(ins) == 0 {
		return fireNone(o)
	}
	ins = nearestFirst(ins, func(in snapshot.Incident) *float64 { return in.Source.DistanceKm })
	cols := []render.StatusColumn{
		{Width: 1},    // no glyph: a named fire is not a detection
		{},            // the name, whole and never truncated
		{Right: true}, // how far
		{},            // which way
		{Right: true}, // acres
		{Right: true}, // containment
		// THE AGE IS THE COLUMN THAT VOLUNTEERS (HUM LEAD, 2026-09-07: "Age can
		// be truncatable - containment is more important"). The name is never
		// shortened, so an unusually long one has to come out of something —
		// and it used to come out of the right edge, silently, taking the
		// containment with it.
		{Right: true, Truncatable: true, MinWidth: 6}, // when it was found — "8d ago" or nothing
	}
	var rows []render.StatusRow
	for _, in := range ins { // bounded by the incident radius (P10-02)
		acres := ""
		if in.Acres != nil {
			acres = render.Thousands(*in.Acres) + " acres"
		}
		contained := ""
		if in.PercentContained != nil {
			contained = fmt.Sprintf("%.0f%% contained", *in.PercentContained)
		}
		found := ""
		if !in.Discovered.IsZero() {
			// THE COARSE FORM, not the marine clock's: a fire discovered 200
			// hours ago reads "8d ago", which is both shorter and what a person
			// would say. The precise form is for something that moves.
			found = seismicAge(now.Sub(in.Discovered))
		}
		dir := ""
		if in.Lat != 0 || in.Lon != 0 {
			brg := geo.BearingDeg(loc.Lat, loc.Lon, in.Lat, in.Lon)
			dir = compass(&brg)
		}
		rows = append(rows, render.StatusRow{Cells: []string{
			"", render.Plain(in.Name), strings.TrimSpace(o.Distance(in.Source.DistanceKm)), dir,
			acres, contained, found,
		}})
	}
	return o.DetailTable(cols, rows, cw-detailRailGutter, render.DetailGutter)
}

// fireCount is the row badge's number (UAT 110): the named incidents
// nearby when there are any, else 1 for unnamed hotspots, 0 for no fire.
func fireCount(fs snapshot.FireState) int {
	switch {
	case len(fs.Incidents) > 0:
		return len(fs.Incidents)
	case len(fs.Hotspots) > 0:
		return 1
	}
	return 0
}

// fireHot reports whether any hotspot reads emphasized.
func fireHot(hs []snapshot.Hotspot, boldMW float64) bool {
	for _, h := range hs {
		if h.FRPMW != nil && *h.FRPMW >= boldMW {
			return true
		}
	}
	return false
}

// ellipsize cuts a name to n cells with a visible ellipsis (U5: a silent
// cut hid that "Cottonwood Creek Complex" was cut at all).
func ellipsize(s string, n int, ascii bool) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if ascii {
		return string(r[:max(0, n-3)]) + "..."
	}
	return string(r[:n-1]) + "…"
}

// fireGlyph is the fire mark for the current glyph set (◆ like the row
// mark — UAT 110/121 — or * under --ascii).
func fireGlyph(o render.Opts) string {
	if o.ASCII {
		return "*"
	}
	return "◆"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// fireBoldMW is the emphasis threshold for the FIRE rows (Config, default 50).
func (d Dashboard) fireBoldMW() float64 {
	if d.cfg.FireBoldMW > 0 {
		return d.cfg.FireBoldMW
	}
	return 50
}
