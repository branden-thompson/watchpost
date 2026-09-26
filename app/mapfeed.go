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

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mapFeed is the window's feed: every alert in view, and the station's
// locations' (D-66, D-76), as
// overlays, with the selected place's own zones asked of the weather service
// so a note can say whether the place lies in a missing zone (FR-4.1, FR-4.4).
func (lp *livePipelines) mapFeed(ctx context.Context, ask tty.MapAsk) tty.MapFeed {
	return lp.mapFeedWith(ctx, lp.mapInputsFetching(ctx, ask), func(loc snapshot.Location) []string {
		if lp.weather == nil {
			return nil
		}
		return lp.weather.ZonesFor(ctx, snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ})
	})
}

// mapFeedWith is mapFeed with the place's zones given, for the tests.
func (lp *livePipelines) mapFeedWith(ctx context.Context, in mapInputs, placeZones func(snapshot.Location) []string) tty.MapFeed {
	var out tty.MapFeed
	snap, place := in.drawable(), in.place
	if snap == nil {
		return out
	}
	held := map[string]bool{} // the station holds these; the window has the rest from the feed
	if in.snap != nil {
		for _, loc := range in.snap.Locations {
			for _, a := range loc.Alerts {
				held[a.ID] = true
			}
		}
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
				if !held[a.ID] {
					out.InView = append(out.InView, a) // the window names it in full
				}
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
	out.Overlays = append(out.Overlays, quakeOverlays(in.quakes)...) // D-80: the ticker's quakes in view
	return out
}

// alertLayerKey is the alert areas' key in the registry, and their overlays'
// ids' first part (W1.13).
const alertLayerKey = tty.AlertLayer

// zoneShapeBytes is one zone's outline on the wire, measured: a mean of 9.6
// KB over 159 cached land and marine zones (2026-09-25; marine zones run to
// 640 KB, wave 1). The estimate's unit for the alert areas (W1.14).
const zoneShapeBytes = 10_000

// The alert areas, registered (W1.13, R-9.2): on by default.
func init() {
	registerMapLayer(mapLayer{key: alertLayerKey, label: "Alert areas", on: true, cost: alertLayerCost})
}

// alertLayerCost is what the alert areas fetch in a refresh, as if nothing
// were held: every zone the alerts name, once - the station's and the
// view's - and an alert with its own
// polygon fetches nothing (FR-9.2).
func alertLayerCost(in mapInputs) (int64, int) {
	snap := in.drawable()
	if snap == nil {
		return 0, 0
	}
	zones := map[string]bool{}
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			if !a.Area.Empty() {
				continue
			}
			for _, z := range a.AffectedZones {
				zones[z] = true
			}
		}
	}
	return int64(len(zones)) * zoneShapeBytes, len(zones)
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
	cat, ok := alertCategory(a)
	if !ok {
		return tuimaps.Overlay{}, false // a forecast, or a product [w] does not show (D-80)
	}
	o := tuimaps.Overlay{ID: alertLayerKey + "/" + cat + "/" + a.ID, Valid: valid, Keeps: keeps} // the category switches it (D-80)
	for _, poly := range area.Shape {
		o.Features = append(o.Features, tuimaps.Feature{Kind: tuimaps.Polygon, Rings: tuimaps.Rings(poly),
			Role: role, Label: label, Severity: sev, Valid: valid, Expires: a.Expires, ID: a.ID})
	}
	return o, true
}

// alertCategory is the alert's category as the [w] window files it - the
// same Classify - by the key the window switches it with, and false for a
// forecast or a product [w] does not show: the map does not draw those (D-80).
func alertCategory(a snapshot.Alert) (string, bool) {
	tab, ok := severe.Classify(globalfeed.ClassSevereWx, a.Event)
	if !ok {
		return "", false
	}
	return tty.AlertCategoryKey(tab)
}

// drawable is the snapshot the feed and the estimate walk: withInView's, with
// only the alerts the map draws, so no zone is fetched for a forecast (D-80).
// A copy: the station's snapshot is shared.
func (in mapInputs) drawable() *snapshot.Snapshot {
	all := in.withInView()
	if all == nil {
		return nil
	}
	out := *all
	out.Locations = make([]snapshot.Location, len(all.Locations))
	for i, loc := range all.Locations {
		kept := make([]snapshot.Alert, 0, len(loc.Alerts))
		for _, a := range loc.Alerts {
			if _, ok := alertCategory(a); ok {
				kept = append(kept, a)
			}
		}
		loc.Alerts = kept
		out.Locations[i] = loc
	}
	return &out
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
