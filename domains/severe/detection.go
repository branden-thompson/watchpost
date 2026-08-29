package severe

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// Detection is the window's DETECTION column (HUM LEAD UAT 2026-08-28):
// how the event was established, when the feed says. A CAP alert carries a
// certainty (Observed, Likely, Possible); a warning's description often
// names its source ("SOURCE: Radar indicated rotation.") — that reads more
// usefully than the certainty and wins when present. A quake reads its
// review status (Reviewed, Automatic). Blank when nothing discernible.
func Detection(r Row) string {
	switch {
	case r.Detail.Alert != nil:
		return detectionOf(r.Detail.Alert.Certainty, r.Detail.Alert.Description)
	case r.Detail.Severe != nil:
		return detectionOf(r.Detail.Severe.Certainty, r.Detail.Severe.Description)
	case r.Detail.Quake != nil:
		return cap1(r.Detail.Quake.Status)
	}
	return ""
}

// detectionOf: the SOURCE line's kind when the description has one, else
// the certainty (never "Unknown").
func detectionOf(certainty, description string) string {
	if src := sourceKind(description); src != "" {
		return src
	}
	c := cap1(certainty)
	if c == "Unknown" {
		return ""
	}
	return c
}

// sourceKind reads NWS's "SOURCE: …" sentence into a short kind. The
// sentences are free text; these are the forms the offices use.
func sourceKind(description string) string {
	u := strings.ToUpper(render.Plain(description)) // search and slice the SAME string: upper-casing can change byte lengths (round 4, A-05)
	i := strings.Index(u, "SOURCE:")
	if i < 0 {
		return ""
	}
	src := strings.ToLower(u[i+len("SOURCE:"):])
	if j := strings.IndexAny(src, ".\n"); j >= 0 {
		src = src[:j]
	}
	kinds := []struct{ needle, kind string }{
		{"radar confirmed", "Radar Confirmed"},
		{"radar", "Radar Indicated"},
		{"spotter", "Spotter Reported"},
		{"law enforcement", "Law Enforcement"},
		{"emergency management", "Emergency Mgmt"},
		{"public", "Public Reported"},
		{"satellite", "Satellite"},
		{"observed", "Observed"},
	}
	for _, k := range kinds {
		if strings.Contains(src, k.needle) {
			return k.kind
		}
	}
	return ""
}
