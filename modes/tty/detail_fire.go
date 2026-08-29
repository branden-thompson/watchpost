package tty

// detail_fire.go — the FIRE rows of Location Details (hotspots, incidents, marks). Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strconv"
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
func fireRows(o render.Opts, loc *snapshot.Location, now time.Time, boldMW float64) []string {
	fs := loc.Fire
	if fs.AsOf.IsZero() { // no fire feed has answered yet (cold launch, feeds down): never "none" (red-team B5 P3)
		return []string{detailRow("FIRE", gridRow("Hotspots", "fire feed not yet available", ""))}
	}
	head := "none within the fire ring"
	if n := len(fs.Hotspots); n > 0 {
		head = fmt.Sprintf("%s hotspot%s nearby", hotspotCount(n), plural(n))
	}
	out := []string{detailRow("FIRE", gridRow("Hotspots", render.Tint(head, fireTone(len(fs.Hotspots) > 0)), ""))}
	out = append(out, hotspotRows(o, loc, fs.Hotspots, now, boldMW)...)
	out = append(out, incidentRows(o, fs.Incidents)...)
	for _, a := range loc.Alerts {
		if ev := strings.ToLower(a.Event); strings.Contains(ev, "red flag") || strings.Contains(ev, "fire weather") {
			out = append(out, detailRow("", gridRow("Fire Wx", render.Tint(render.Plain(a.Event), render.Tok(render.AlertDanger)), "")))
			break
		}
	}
	return out
}

// hotspotCount words the count; at the cap it is "300+" (snapshot.MaxHotspots).
func hotspotCount(n int) string {
	if n >= snapshot.MaxHotspots {
		return fmt.Sprintf("%d+", snapshot.MaxHotspots)
	}
	return strconv.Itoa(n)
}

// fireSeparators are the glyphs the FIRE rows join with, per glyph set.
func fireSeparators(o render.Opts) (dot, more string) {
	if o.ASCII {
		return " - ", "..."
	}
	return " " + o.Glyphs().Dot + " ", "…"
}

// hotspotRows: up to three hotspots nearest first, then "… and N more".
func hotspotRows(o render.Opts, loc *snapshot.Location, hs []snapshot.Hotspot, now time.Time, boldMW float64) []string {
	dot, more := fireSeparators(o)
	var out []string
	for i, h := range hs {
		if i == 3 {
			out = append(out, detailRow("", gridRow("", fmt.Sprintf("%s and %d more", more, len(hs)-3), "")))
			break
		}
		brg := geo.BearingDeg(loc.Lat, loc.Lon, h.Lat, h.Lon)
		where := "  " + fireGlyph(o) + " " + strings.TrimSpace(o.Distance(h.DistanceKm)) + " " + compass(&brg) + " " // the trailing space keeps a gap when the label fills the column (red-team B5 U6)
		strength := "n/a MW"                                                                                         // an unmeasured point (HMS GOES often) — short, so the age column never collides (U1)
		if h.FRPMW != nil {
			strength = fmt.Sprintf("%.0f MW", *h.FRPMW)
			if *h.FRPMW >= boldMW {
				strength = render.Tint(strength, "1;"+render.Tok(render.FireMark))
			}
		}
		if sat := render.Plain(h.Source.ModelOrStation); sat != "" {
			strength += dot + sat
		}
		age := "age n/a"
		if !h.DetectedAt.IsZero() { // a point without a time is not "2562047h" old (U2)
			age = fixedAgeTrim(now.Sub(h.DetectedAt))
		}
		out = append(out, detailRow("", gridRow(where, strength, age)))
	}
	return out
}

// incidentRows: up to three named incidents, largest first.
func incidentRows(o render.Opts, ins []snapshot.Incident) []string {
	dot, _ := fireSeparators(o)
	var out []string
	for i, in := range ins {
		if i == 3 {
			break
		}
		facts := strings.TrimSpace(o.Distance(in.Source.DistanceKm))
		if in.Acres != nil {
			facts += dot + render.Thousands(*in.Acres) + " ac"
		}
		contained := ""
		if in.PercentContained != nil {
			contained = fmt.Sprintf("%.0f%% contained", *in.PercentContained)
		}
		out = append(out, detailRow("", gridRow("  "+ellipsize(render.Plain(in.Name), 11, o.ASCII)+" ", facts, contained)))
	}
	return out
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

// fireTone: the count reads in the fire colour when there is fire.
func fireTone(on bool) string {
	if on {
		return render.Tok(render.FireMark)
	}
	return render.Tok(render.TextBase)
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
