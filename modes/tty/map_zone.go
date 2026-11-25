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
func (mm mapModel) serviceKm(e mapEnv) float64 {
	return float64(max(e.Station.RadiusMi, 0)) * units.KmPerMile
}

// propRadiusKm is the near-vertical answers' radius (D-100, FR-2.3): on a UI
// mode with a service area, the service radius; else the Setting's.
func (mm mapModel) propRadiusKm(e mapEnv) int {
	if km := mm.serviceKm(e); mm.mapRole(e).zone && km > 0 {
		return int(math.Round(km))
	}
	return mm.mapPropRadiusKm
}

// propLocalName names the near-vertical area: the service area on a UI mode
// with one (FR-2.3), else NVIS's (D-100).
func (mm mapModel) propLocalName(e mapEnv) string {
	if mm.mapRole(e).zone && mm.serviceKm(e) > 0 {
		return "service area"
	}
	return "NVIS"
}

// zoneOverlays are the UI mode's service area (D-195): a circle of the
// service radius around the station in go-tuiMaps' zone role - outlined in
// red, its inside tinted, stippled over a field its tint cannot blend with
// (D-196) - in every mode; none for a UI mode without one, or with no
// station or radius set.
func (mm mapModel) zoneOverlays(e mapEnv) []tuimaps.Overlay {
	loc, km := mm.mapPlace(e), mm.serviceKm(e)
	if !mm.mapRole(e).zone || loc == nil || km <= 0 {
		return nil
	}
	return []tuimaps.Overlay{{ID: serviceAreaID, Valid: e.Now().UTC().Truncate(24 * time.Hour), Keeps: serviceAreaKeeps,
		Features: []tuimaps.Feature{{Kind: tuimaps.Circle, Centre: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}, RadiusKm: km, Role: tuimaps.Zone}}}}
}

// towerPlace is the tower's mark (D-25, D-195): a diamond at the station,
// named for it; false for a UI mode without one, or with no station.
func (mm mapModel) towerPlace(e mapEnv) (tuimaps.Place, bool) {
	loc := mm.mapPlace(e)
	if !mm.mapRole(e).zone || loc == nil {
		return tuimaps.Place{}, false
	}
	return tuimaps.Place{ID: towerPlaceID, Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}, Marker: tuimaps.MarkerDiamond}, true
}

// setZone hands the library the service area and the tower's mark as the UI
// mode wants them, and takes off what it no longer does. The role it drew for
// is kept, so a swap of UI modes with the map open puts the map on the new
// role's place.
func (mm mapModel) setZone(e mapEnv) mapModel {
	m := mm.mapPane.m
	if m == nil {
		return mm
	}
	mm.mapPane.zoneGiven, _ = mm.reconcile(mm.mapPane.zoneGiven, mm.zoneOverlays(e), func(o tuimaps.Overlay, err error) {
		mm.problem("Service area: "+o.ID+" not drawn - "+err.Error(), e) // ours to fix, never the listener's (D-124)
	})
	want, ok := mm.towerPlace(e)
	if had := mm.mapPane.tower; had != want || !ok {
		if had.ID != "" {
			mm.mapPane.call("RemovePlace", func() { _, _ = m.RemovePlace(had.ID) })
			mm.mapPane.tower = tuimaps.Place{}
		}
		if ok {
			var err error
			mm.mapPane.call("AddPlace", func() { _, err = m.AddPlace(want) })
			if err != nil {
				mm.problem("Service area: the tower not marked - "+err.Error(), e)
			} else {
				mm.mapPane.tower = want
			}
		}
	}
	mm.mapPane.roleDrawn = e.Surface
	return mm
}

// followRole puts an open map on the UI mode's place and marks when the UI
// mode has changed since it was drawn: a swap with the map open.
func (mm mapModel) followRole(e mapEnv) mapModel {
	if mm.mapPane.m == nil || mm.mapPane.roleDrawn == e.Surface {
		return mm
	}
	return mm.followSelection(e).setZone(e).requestFeed()
}

// zoneWords are the words for the service area (D-195), said without the
// picture: its radius around the station, and no colour - the map draws it
// red only where it draws colour (AX-12); none for a UI mode without one.
func (mm mapModel) zoneWords(e mapEnv) string {
	loc, mi := mm.mapPlace(e), e.Station.RadiusMi
	if !mm.mapRole(e).zone || loc == nil || mi <= 0 {
		return ""
	}
	radius := mm.distanceLabel(int(math.Round(mm.serviceKm(e))), e)
	return "The circle around " + loc.Label + " is the station's service area: " + radius + "."
}
