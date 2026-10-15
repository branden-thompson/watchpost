package tty

import "github.com/branden-thompson/watchpost/platform/snapshot"

// mapPlaceKind is whose place a UI mode's map is on (D-195).
type mapPlaceKind uint8

// The kinds. The zero value is unruled: a UI mode added later (Analyst,
// D-27) must say which it is, and TestEveryUIModeHasAMapRole fails until it
// does.
const (
	mapPlaceUnruled mapPlaceKind = iota
	mapOnSelected                // the selected place: the listener's
	mapOnStation                 // the station's: the transmitter set in Settings
)

// mapRole is how a UI mode uses the map (D-193, D-195): whose place it is
// on; whether the station's service area is drawn, the tower marked and the
// near-vertical answers given to the service radius (FR-2.3); whether [ and ]
// step through the places; and its Propagation role.
type mapRole struct {
	place     mapPlaceKind
	zone      bool
	placeKeys bool
	prop      propRole
}

// mapRoles are each UI mode's. Observer is on the selected place, steps
// through the places and explores from the map's centre; the Broadcaster is
// on its station, its service area always drawn, and answers from the
// station, the centre a target.
var mapRoles = [numSurfaces]mapRole{
	SurfaceObserver:    {place: mapOnSelected, placeKeys: true, prop: propRole{origin: propFromCentre}},
	SurfaceBroadcaster: {place: mapOnStation, zone: true, prop: propRole{origin: propFromStation, target: true}},
}

// mapRole is the UI mode on screen's role; an unruled one is Observer's.
func (d Dashboard) mapRole() mapRole {
	if d.surface < 0 || d.surface >= numSurfaces || mapRoles[d.surface].place == mapPlaceUnruled {
		return mapRoles[SurfaceObserver]
	}
	return mapRoles[d.surface]
}

// propRole is the UI mode on screen's Propagation role (D-193).
func (d Dashboard) propRole() propRole { return d.mapRole().prop }

// mapPlace is the place the map is on (D-195): the selected place, or the
// station's; nil with none.
func (d Dashboard) mapPlace() *snapshot.Location {
	if d.mapRole().place == mapOnStation {
		return d.stationPlace()
	}
	return d.selectedLocation()
}

// noTransmitter reports whether no transmitter is set: the selected place
// stands in for the station (D-193).
func (d Dashboard) noTransmitter() bool {
	t := d.cfg.Transmitter
	return t == nil || (t.Lat == 0 && t.Lon == 0)
}

// stationPlace is the station's place (D-193, D-195): the transmitter set in
// Settings - the watchlist's own record where one is at its place, so its
// alerts come with it - or, with none set, the selected place, standing in;
// nil with neither.
func (d Dashboard) stationPlace() *snapshot.Location {
	if d.noTransmitter() {
		return d.selectedLocation()
	}
	t := d.cfg.Transmitter
	if d.snap != nil {
		for i := range d.snap.Locations {
			if l := &d.snap.Locations[i]; l.Lat == t.Lat && l.Lon == t.Lon {
				return l
			}
		}
	}
	return &snapshot.Location{Label: t.Label, Tag: t.Tag, Zip: t.Zip, Lat: t.Lat, Lon: t.Lon, TZ: t.TZ}
}
