package tty

import (
	"math"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/units"
)

// The station's marks on the map (D-195): its service area's overlay and
// the tower's place.
const (
	serviceAreaID = "station/service-area"
	towerPlaceID  = "station/tower"
)

// serviceAreaKeeps is how long the service area stays current: the most the
// library takes, the area being the station's own and never old. Its valid
// time is the day's start, so the same circle is not handed in again all day
// (U1-28).
const serviceAreaKeeps = 7 * 24 * time.Hour

// serviceKm is the station's service radius in kilometres; zero unset.
func (d Dashboard) serviceKm() float64 {
	return float64(max(d.cfg.ServiceRadiusMi, 0)) * units.KmPerMile
}

// propRadiusKm is the near-vertical answers' radius (D-100, FR-2.3): on a UI
// mode with a service area, the service radius; else the Setting's.
func (d Dashboard) propRadiusKm() int {
	if km := d.serviceKm(); d.mapRole().zone && km > 0 {
		return int(math.Round(km))
	}
	return d.mapPropRadiusKm
}

// propLocalName names the near-vertical area: the service area on a UI mode
// with one (FR-2.3), else NVIS's (D-100).
func (d Dashboard) propLocalName() string {
	if d.mapRole().zone && d.serviceKm() > 0 {
		return "service area"
	}
	return "NVIS"
}

// zoneOverlays are the UI mode's service area (D-195): a circle of the
// service radius around the station in go-tuiMaps' zone role - outlined in
// red, its inside tinted, stippled over a field its tint cannot blend with
// (D-196) - in every mode; none for a UI mode without one, or with no
// station or radius set.
func (d Dashboard) zoneOverlays() []tuimaps.Overlay {
	loc, km := d.mapPlace(), d.serviceKm()
	if !d.mapRole().zone || loc == nil || km <= 0 {
		return nil
	}
	return []tuimaps.Overlay{{ID: serviceAreaID, Valid: d.now().UTC().Truncate(24 * time.Hour), Keeps: serviceAreaKeeps,
		Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}, RadiusKm: km, Role: tuimaps.Zone}}}}
}

// towerPlace is the tower's mark (D-25, D-195): a diamond at the station,
// named for it; false for a UI mode without one, or with no station.
func (d Dashboard) towerPlace() (tuimaps.Place, bool) {
	loc := d.mapPlace()
	if !d.mapRole().zone || loc == nil {
		return tuimaps.Place{}, false
	}
	return tuimaps.Place{ID: towerPlaceID, Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}, Marker: tuimaps.MarkerDiamond}, true
}

// setZone hands the library the service area and the tower's mark as the UI
// mode wants them, and takes off what it no longer does. The role it drew for
// is kept, so a swap of UI modes with the map open puts the map on the new
// role's place.
func (d Dashboard) setZone() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	d.mapPane.zoneGiven, _ = d.reconcile(d.mapPane.zoneGiven, d.zoneOverlays(), func(o tuimaps.Overlay, err error) {
		d.problem("Service area: " + o.ID + " not drawn - " + err.Error()) // ours to fix, never the listener's (D-124)
	})
	want, ok := d.towerPlace()
	if had := d.mapPane.tower; had != want || !ok {
		if had.ID != "" {
			d.mapPane.call("RemovePlace", func() { _, _ = m.RemovePlace(had.ID) })
			d.mapPane.tower = tuimaps.Place{}
		}
		if ok {
			var err error
			d.mapPane.call("AddPlace", func() { _, err = m.AddPlace(want) })
			if err != nil {
				d.problem("Service area: the tower not marked - " + err.Error())
			} else {
				d.mapPane.tower = want
			}
		}
	}
	d.mapPane.roleDrawn = d.surface
	return d
}

// followRole puts an open map on the UI mode's place and marks when the UI
// mode has changed since it was drawn: a swap with the map open.
func (d Dashboard) followRole() Dashboard {
	if d.mapPane.m == nil || d.mapPane.roleDrawn == d.surface {
		return d
	}
	return d.followSelection().setZone().requestFeed()
}

// zoneWords are the words for the service area (D-195), said without the
// picture: its radius around the station, and no colour - the map draws it
// red only where it draws colour (AX-12); none for a UI mode without one.
func (d Dashboard) zoneWords() string {
	loc, mi := d.mapPlace(), d.cfg.ServiceRadiusMi
	if !d.mapRole().zone || loc == nil || mi <= 0 {
		return ""
	}
	radius := d.distanceLabel(int(math.Round(d.serviceKm())))
	return "The circle around " + loc.Label + " is the station's service area: " + radius + "."
}
