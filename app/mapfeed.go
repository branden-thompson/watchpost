package app

// mapfeed.go — what the Observer's map draws (0.18.0 W5).
//
// THIS IS THE ONLY PLACE THAT MAY NAME A DOMAIN AND WHAT DRAWS (architecture,
// 0.17.0's mapgeometry.go): the zone store resolves an alert's ground, and it
// is turned here into the library's overlays, one per alert, and the notes the
// window prints under the map. The window asks for it off its own goroutine.

import (
	"context"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mapFeed is the window's feed: every alert in view, and the station's
// locations' (D-66, D-76), as
// overlays, with the selected place's own zones asked of the weather service
// so a note can say whether the place lies in a missing zone (FR-4.1, FR-4.4).
func (lp *livePipelines) mapFeed(ctx context.Context, ask tty.MapAsk) tty.MapFeed {
	ctx = httpx.WithInteractive(ctx) // the listener is waiting: never behind the station's launch burst (D-156)
	start := time.Now()
	defer lp.timings.stage("whole", start)
	in := lp.mapInputsFetching(ctx, ask)
	lp.timings.stage("inputs", start)
	return lp.mapFeedWith(ctx, in, func(loc snapshot.Location) []string {
		if lp.weather == nil {
			return nil
		}
		return lp.weather.ZonesFor(ctx, snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ})
	})
}

// mapFeedWith is mapFeed with the place's zones given, for the tests.
func (lp *livePipelines) mapFeedWith(ctx context.Context, in mapInputs, placeZones func(snapshot.Location) []string) tty.MapFeed {
	var out tty.MapFeed
	snap := in.drawable()
	if snap == nil {
		return out
	}
	zonesAt := time.Now()
	areas := resolveAlertAreas(ctx, lp.zoneShapes, snap)
	lp.timings.stage("zones", zonesAt)
	defer lp.timings.stage("overlays", time.Now())
	addAlerts(&out, snap, areas, heldAlerts(in.snap), in.place, placeZones)
	now := time.Now()
	for _, o := range fireChosen(fireOverlays(in.fire, in.view, lp.fireRules(), now), in.fireMode) { // D-121, D-145: the fire in view, as chosen
		addNow(&out, o)
	}
	for _, o := range seaStations(in, now) {
		addNow(&out, o)
	}
	airnow, airTimes := airnowOverlays(in.airnow, in.view, in.anchor, now) // D-139: AirNow's monitors
	for _, o := range airnow {
		out.Overlays = append(out.Overlays, o)
		out.Times = timed(out.Times, o.ID, airTimes[o.ID])
	}
	if o, ok := quakeOverlay(in.quakes, now, in.clock); ok { // D-80, D-122, D-123, D-129: the quakes chosen, in view
		addNow(&out, o)
	}
	return out
}

// addNow adds an overlay drawn as now: through the loop, and on Now alone in
// Forecast mode.
func addNow(out *tty.MapFeed, o tuimaps.Overlay) {
	out.Overlays = append(out.Overlays, o)
	out.Times = timed(out.Times, o.ID, tty.TimedOverlay{Happened: true})
}

// heldAlerts is the IDs of the alerts the station holds; the window has the
// rest from the feed.
func heldAlerts(snap *snapshot.Snapshot) map[string]bool {
	held := map[string]bool{}
	if snap == nil {
		return held
	}
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			held[a.ID] = true
		}
	}
	return held
}

// addAlerts adds each alert once, drawn over its area while the mode's moment
// meets it (D-98), named in full where the station does not hold it; and,
// where an area is incomplete, a note of whether it covers the place.
func addAlerts(out *tty.MapFeed, snap *snapshot.Snapshot, areas map[string]geo.Area, held map[string]bool, place *snapshot.Location, placeZones func(snapshot.Location) []string) {
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
				out.Times = timed(out.Times, o.ID, alertTimes(a))
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
			addPartial(out, a, area, place.Label, zonesOfPlace)
		}
	}
}

// addPartial adds an incomplete area's note, and marks the alert as covering
// the place by a missing zone where one of the place's zones is missing.
func addPartial(out *tty.MapFeed, a snapshot.Alert, area geo.Area, label string, zonesOfPlace map[string]bool) {
	out.Notes = append(out.Notes, partialNote(a, area, label, zonesOfPlace))
	for _, id := range area.Missing {
		if zonesOfPlace[id] {
			if out.InMissing == nil {
				out.InMissing = map[string]bool{}
			}
			out.InMissing[a.ID] = true // the description says it covers the place by that zone
		}
	}
}

// seaStations is the sea's stations in view (D-127, D-128): the buoys and
// the tide stations, each with its next tide.
func seaStations(in mapInputs, now time.Time) []tuimaps.Overlay {
	var marine []tuimaps.Overlay
	if o, ok := buoyOverlay(in.buoys, in.view, now, in.imperial); ok {
		marine = append(marine, o)
	}
	var stations []coops.Station
	nexts := map[string]tideMark{}
	for _, m := range in.tides {
		stations, nexts[m.station.ID] = append(stations, m.station), m
	}
	if o, ok := tideOverlay(stations, in.view, now, in.imperial, in.clock, now.Location(), func(id string) (snapshot.TideEvent, bool) {
		m := nexts[id]
		return m.next, m.known
	}); ok {
		marine = append(marine, o)
	}
	return marine
}

// alertTimes is when an alert is (D-98): from its onset - or, with none
// given, when it took effect - to when it ends, or with no end given, when
// it expires.
func alertTimes(a snapshot.Alert) tty.TimedOverlay {
	t := tty.TimedOverlay{From: a.Effective, Until: a.Expires}
	if t.From.IsZero() {
		t.From = a.Sent
	}
	if a.Onset != nil && !a.Onset.IsZero() {
		t.From = *a.Onset
	}
	if a.Ends != nil && !a.Ends.IsZero() {
		t.Until = *a.Ends
	}
	return t
}

// timed adds one overlay's times.
func timed(times map[string]tty.TimedOverlay, id string, t tty.TimedOverlay) map[string]tty.TimedOverlay {
	if times == nil {
		times = map[string]tty.TimedOverlay{}
	}
	times[id] = t
	return times
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
	registerMapLayer(mapLayer{key: alertLayerKey, label: "Alert areas", on: true, cost: alertLayerCost, chips: []string{"NWS"}})
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
			if cat, _ := alertCategory(a); in.switchedOn != nil && !in.switchedOn(tty.AlertCategorySwitch(cat)) {
				continue // its category unchecked: nothing of it is fetched or drawn (D-149)
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

// overlayStep is the step a point overlay's Valid is stamped to (W14, P-6).
// Stamped with the moment of each ask, the same data a minute later would be a
// new overlay - handed to the library, and prepared, again on every answer. To
// the step, unchanged data compares the same across asks; its currency moves
// at most a step early, well inside every Keeps (an hour at the least), and
// the next ask stamps it anew. Ages - a buoy's reading, a quake's NEW - still
// read the moment itself.
const overlayStep = 10 * time.Minute

// overlayStamp is the Valid a point overlay built now carries.
func overlayStamp(now time.Time) time.Time { return now.Truncate(overlayStep) }

// drawable is the snapshot the feed and the estimate walk: withInView's, with
// only the alerts the map draws - never a forecast (D-80), nor an alert of a
// category unchecked or with the layer off (D-160) - so no zone is fetched for
// one. A copy: the station's snapshot is shared.
func (in mapInputs) drawable() *snapshot.Snapshot {
	all := in.withInView()
	if all == nil {
		return nil
	}
	off := map[string]bool{}
	for _, c := range in.alertCategoriesOff {
		off[c] = true
	}
	out := *all
	out.Locations = make([]snapshot.Location, len(all.Locations))
	for i, loc := range all.Locations {
		kept := make([]snapshot.Alert, 0, len(loc.Alerts))
		for _, a := range loc.Alerts {
			if cat, ok := alertCategory(a); ok && !in.alertsOff && !off[cat] {
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
