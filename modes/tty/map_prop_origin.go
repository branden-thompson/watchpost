package tty

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/propanswers"
	"math"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// propOriginKind is where a UI mode's Propagation answers come from (D-193).
type propOriginKind uint8

// The kinds. The zero value is unruled: each UI mode's row in mapRoles
// (D-195) says which it is, and TestEveryUIModeHasAMapRole fails until it
// does.
const (
	propOriginUnruled propOriginKind = iota
	propFromCentre                   // the map's centre, as it pans: exploring
	propFromStation                  // the station's: the centre is a target
)

// propRole is how a UI mode uses the Propagation mode (D-193): its origin,
// and whether the map's centre is answered as a target from it. It is a part
// of the UI mode's map role (mapRoles).
type propRole struct {
	origin propOriginKind
	target bool
}

// propPlace is the Propagation mode's origin (D-193), worked out in Update
// with the map's title and read by the frame: where it is and its name.
type propPlace struct {
	at   ionomaps.LatLon
	name string
	ok   bool
}

// propOriginNow is the origin for the UI mode on screen, in Update: the
// map's centre and its name (propCentreName), or the transmitter and its label, or
// the selected place, its name saying it stands in for a transmitter; not
// ok with nothing to answer from.
func (d Dashboard) propOriginNow() propPlace {
	if d.propRole().origin == propFromStation {
		station := d.stationPlace()
		switch {
		case station == nil:
			return propPlace{}
		case d.noTransmitter():
			return propPlace{at: ionomaps.LatLon{Lat: station.Lat, Lon: station.Lon}, name: station.Label + " " + bcNoTransmitter, ok: true}
		}
		return propPlace{at: ionomaps.LatLon{Lat: station.Lat, Lon: station.Lon}, name: d.cfg.Transmitter.Label, ok: true}
	}
	loc := d.selectedLocation()
	if c, ok := d.propCentreAt(); ok {
		return propPlace{at: c, name: d.propCentreName(c), ok: true}
	}
	if loc == nil {
		return propPlace{}
	}
	return propPlace{at: ionomaps.LatLon{Lat: loc.Lat, Lon: loc.Lon}, name: loc.Label, ok: true} // no map yet: the selected place
}

// propCentreName names the centre answered, anywhere on the globe (D-193,
// BUILD red team AX-1): the place found while the centre rests on it, else
// the town nearest it, else its coordinates.
func (d Dashboard) propCentreName(c ionomaps.LatLon) string {
	if f := d.mapPane.propFound; f != nil && propStepped(f.Lat, f.Lon) == c {
		return f.Label
	}
	if d.cfg.PlaceNear != nil {
		if town := d.cfg.PlaceNear(c.Lat, c.Lon); town != "" {
			return "near " + town
		}
	}
	return latLonText(c)
}

// latLonText is a point as the windows write it: "35.5°N 139.5°E".
func latLonText(c ionomaps.LatLon) string {
	ns, ew := "N", "E"
	if c.Lat < 0 {
		ns = "S"
	}
	if c.Lon < 0 {
		ew = "W"
	}
	return fmt.Sprintf("%.1f°%s %.1f°%s", math.Abs(c.Lat), ns, math.Abs(c.Lon), ew)
}

// propTarget is the station's answer for the map's centre (D-193): the
// usable bands from the station and the limits, and how the frequency held
// (D-178) fares on the path - and which way to move it.
func (d Dashboard) propTarget(bands []ionomaps.BandStatus) []string {
	reach := "nothing usable at 100 W"
	switch u := propanswers.Usable(bands); len(u) {
	case 0:
	case 1:
		reach = "usable " + u[0]
	default:
		reach = "usable " + u[0] + "-" + u[len(u)-1]
	}
	out := []string{reach + " · " + propanswers.LimitWords(bands, " MHz")}
	if best := propanswers.BestWords(propanswers.LimitsOf(bands), true); best != "" {
		out = append(out, best) // D-197
	}
	if f := d.mapPane.propFreqMHz; f > 0 {
		out = append(out, propanswers.FreqVerdict(f, d.mapPane.propFreqLabel, propanswers.LimitsOf(bands)))
	}
	return out
}
