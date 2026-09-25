package app

// mapnational.go — the national severe events the map draws in the national
// scope (0.18.0 W5.3, FR-4.3, D-23).
//
// THEY ARE THE TICKER'S, READ AS ALERTS. The national feed is already polled
// for the marquee and the severe window; the map adds no request of its own
// for it. Each event becomes the alert the station's own reader would have
// made of it, so it is resolved, drawn, noted and described by exactly the
// path the station's alerts take - and kept to the selected place's region,
// never wider (D-28).

import (
	"strings"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// nationalAlerts are the national severe events in the place's region, as
// alerts: a superseded event, one of another class, and one outside the
// region are left out. An event with a point is placed by it; a zone-only
// event by its first zone's code.
func nationalAlerts(feed []globalfeed.Event, place *snapshot.Location) []snapshot.Alert {
	if place == nil {
		return nil
	}
	region, ok := geo.RegionOf(place.Lat, place.Lon)
	if !ok {
		return nil
	}
	var out []snapshot.Alert
	for _, e := range feed {
		if e.Class != globalfeed.ClassSevereWx || e.Severe == nil || e.Superseded {
			continue
		}
		zones := make([]string, 0, len(e.Severe.AffectedZones))
		for _, z := range e.Severe.AffectedZones {
			zones = append(zones, afterLastSlash(z))
		}
		if !inRegion(region, e, zones) {
			continue
		}
		out = append(out, snapshot.Alert{ID: afterLastSlash(e.ID), Event: e.Type, Severity: e.Severe.Severity, AreaDesc: e.Place,
			Effective: e.Severe.Effective, Sent: e.Severe.Sent, Expires: e.Until, SenderName: e.Severe.SenderName,
			AffectedZones: zones, Area: e.Severe.Area})
	}
	return out
}

// inRegion reports whether an event lies in the region: by its point, or
// with none by its first zone.
func inRegion(region geo.Region, e globalfeed.Event, zones []string) bool {
	if e.HasPoint {
		return region.Contains(e.Lat, e.Lon)
	}
	if len(zones) == 0 {
		return false
	}
	r, ok := geo.RegionOfZone(zones[0])
	return ok && r.Name == region.Name
}

// afterLastSlash is an address's last part: an alert's id or a zone's code,
// as the station's own reader keeps them.
func afterLastSlash(s string) string { return s[strings.LastIndex(s, "/")+1:] }
