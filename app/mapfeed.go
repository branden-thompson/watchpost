package app

// mapfeed.go — what the Observer's map draws (0.18.0 W5).
//
// THIS IS THE ONLY PLACE THAT MAY NAME A DOMAIN AND WHAT DRAWS (architecture,
// 0.17.0's mapgeometry.go): the zone store resolves an alert's ground, and it
// is turned here into the library's overlays, one per alert, and the notes the
// window prints under the map. The window asks for it off its own goroutine.

import (
	"context"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mapFeed is the window's feed: every alert of the station's locations, as
// overlays, with the selected place's own zones asked of the weather service
// so a note can say whether the place lies in a missing zone (FR-4.1, FR-4.4).
func (lp *livePipelines) mapFeed(ctx context.Context, snap *snapshot.Snapshot, place *snapshot.Location) tty.MapFeed {
	return lp.mapFeedWith(ctx, snap, place, func(loc snapshot.Location) []string {
		if lp.weather == nil {
			return nil
		}
		return lp.weather.ZonesFor(ctx, snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ})
	})
}

// mapFeedWith is mapFeed with the place's zones given, for the tests.
func (lp *livePipelines) mapFeedWith(ctx context.Context, snap *snapshot.Snapshot, place *snapshot.Location, placeZones func(snapshot.Location) []string) tty.MapFeed {
	var out tty.MapFeed
	if snap == nil {
		return out
	}
	areas := resolveAlertAreas(ctx, lp.zoneShapes, snap)
	seen := map[string]bool{}
	var zonesOfPlace map[string]bool
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			if seen[a.ID] {
				continue // the same alert is attached to every location it covers
			}
			seen[a.ID] = true
			area := areas[a.ID]
			if o, ok := alertOverlay(a, area); ok {
				out.Overlays = append(out.Overlays, o)
			}
			if area.Complete() || place == nil {
				continue
			}
			if zonesOfPlace == nil {
				zonesOfPlace = map[string]bool{}
				for _, z := range placeZones(*place) {
					zonesOfPlace[z] = true
				}
			}
			out.Notes = append(out.Notes, partialNote(a, area, place.Label, zonesOfPlace))
			for _, id := range area.Missing {
				if zonesOfPlace[id] {
					if out.InMissing == nil {
						out.InMissing = map[string]bool{}
					}
					out.InMissing[a.ID] = true // the description says it covers the place by that zone
				}
			}
		}
	}
	return out
}

// alertOverlay is one alert as the library draws it (FR-4.2, D-55): one
// feature per area, the first ring the outline and the rest holes; the
// severity as data and as the role; its times and its id, which Report
// returns so a description can join back to the alert. A partial area is
// drawn as found and its label says how much (FR-4.4).
func alertOverlay(a snapshot.Alert, area geo.Area) (tuimaps.Overlay, bool) {
	if area.Shape.Empty() {
		return tuimaps.Overlay{}, false
	}
	sev, role := severityOf(a.Severity)
	label := a.Event
	if total := len(a.AffectedZones); !area.Complete() && total > 0 {
		label += " (" + strconv.Itoa(total-len(area.Missing)) + " of " + strconv.Itoa(total) + " zones)"
	}
	valid := a.Effective
	if valid.IsZero() {
		valid = a.Sent
	}
	keeps := time.Hour
	if d := a.Expires.Sub(valid); d > 0 {
		keeps = d
	}
	o := tuimaps.Overlay{ID: "alert/" + a.ID, Valid: valid, Keeps: keeps}
	for _, poly := range area.Shape {
		o.Features = append(o.Features, tuimaps.Feature{Kind: tuimaps.Polygon, Rings: tuimaps.Rings(poly),
			Role: role, Label: label, Severity: sev, Valid: valid, Expires: a.Expires, ID: a.ID})
	}
	return o, true
}

// severityOf is CAP's severity as the library's, and the role it is drawn in:
// one to one (FR-4.2).
func severityOf(s string) (tuimaps.Severity, tuimaps.Token) {
	switch strings.ToLower(s) {
	case "extreme":
		return tuimaps.SeverityExtreme, tuimaps.AlertExtreme
	case "severe":
		return tuimaps.SeveritySevere, tuimaps.AlertSevere
	case "moderate":
		return tuimaps.SeverityModerate, tuimaps.AlertModerate
	case "minor":
		return tuimaps.SeverityMinor, tuimaps.AlertMinor
	}
	return tuimaps.SeverityUnknown, tuimaps.AlertUnknown
}

// partialNote names what an alert's area is missing, in words from the
// alert's own area description, never by code (D-55), and says whether the
// selected place lies in it: by the place's own zone and county codes,
// because a missing zone has no ground to test against (FR-4.4, D-42).
func partialNote(a snapshot.Alert, area geo.Area, place string, placeZones map[string]bool) string {
	names := strings.Split(a.AreaDesc, "; ")
	var missing []string
	inIt := false
	for _, id := range area.Missing {
		if placeZones[id] {
			inIt = true
		}
		if len(names) == len(a.AffectedZones) {
			for i, z := range a.AffectedZones {
				if z == id {
					missing = append(missing, names[i])
				}
			}
		}
	}
	what := strings.Join(missing, ", ")
	it := "it"
	switch {
	case len(missing) != len(area.Missing) || len(missing) == 0:
		what = strconv.Itoa(len(area.Missing)) + " of its zones"
		it = "them"
	case len(missing) > 1:
		it = "them"
	}
	total := len(a.AffectedZones)
	note := a.Event + ": " + what + " could not be drawn (" + strconv.Itoa(total-len(area.Missing)) + " of " + strconv.Itoa(total) + " zones shown). "
	if inIt {
		return note + place + " lies in " + it + "."
	}
	return note + place + " does not lie in " + it + "."
}
