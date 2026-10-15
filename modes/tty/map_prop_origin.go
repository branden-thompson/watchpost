package tty

import (
	"strconv"

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
// map's centre and the view's name, or the transmitter and its label, or
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
		return propPlace{at: c, name: d.viewName(), ok: true}
	}
	if loc == nil {
		return propPlace{}
	}
	return propPlace{at: ionomaps.LatLon{Lat: loc.Lat, Lon: loc.Lon}, name: loc.Label, ok: true} // no map yet: the selected place
}

// propLimits are the limits an answer's bands name (R-2.6): the upper
// limit, the LUF and whether the circuit is too weak there rather than
// absorbed (D-70), and D-RAP's 1 dB frequency; zero where none decides.
type propLimits struct {
	upper, luf, drap float64
	weak             bool
}

// limitsOf reads an answer's limits from its bands.
func limitsOf(bands []ionomaps.BandStatus) propLimits {
	var l propLimits
	for _, b := range bands {
		switch b.Limit {
		case ionomaps.LimitMUF, ionomaps.LimitFoF2:
			if l.upper == 0 {
				l.upper = float64(b.LimitMHz)
			}
		case ionomaps.LimitAbsorption, ionomaps.LimitCircuit:
			l.luf, l.weak = float64(b.LimitMHz), l.weak || b.Limit == ionomaps.LimitCircuit
		case ionomaps.LimitDRAP:
			l.drap = float64(b.LimitMHz)
		}
	}
	return l
}

// usableBands are the bands an answer can use: open or disturbed, by name.
func usableBands(bands []ionomaps.BandStatus) []string {
	var out []string
	for i, b := range bands {
		if (b.Status == ionomaps.Open || b.Status == ionomaps.Disturbed) && i < len(hfBands) {
			out = append(out, hfBands[i].name)
		}
	}
	return out
}

// mhzWords is a frequency as the windows say it: to a tenth of a megahertz.
func mhzWords(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

// propTarget is the station's answer for the map's centre (D-193): the
// usable bands from the station and the limits, and how the frequency held
// (D-178) fares on the path - and which way to move it.
func (d Dashboard) propTarget(bands []ionomaps.BandStatus) []string {
	reach := "nothing usable at 100 W"
	switch u := usableBands(bands); len(u) {
	case 0:
	case 1:
		reach = "usable " + u[0]
	default:
		reach = "usable " + u[0] + "-" + u[len(u)-1]
	}
	out := []string{reach + " · " + propRowLimitWords(bands)}
	if f := d.mapPane.propFreqMHz; f > 0 {
		out = append(out, propFreqVerdict(f, d.mapPane.propFreqLabel, limitsOf(bands)))
	}
	return out
}

// propFreqVerdict is a held frequency on a path (D-193): with the LUF at or
// over the MUF, no frequency serves - too weak at 100 W; above the MUF, come
// down under it; under the LUF, go up to it (absorbed, or too weak); under
// D-RAP's 1 dB frequency, disturbed; else open.
func propFreqVerdict(f float64, label string, l propLimits) string {
	held := label + ": "
	switch {
	case l.upper > 0 && l.luf >= l.upper:
		return held + "too weak at 100 W on this path"
	case l.upper > 0 && f >= l.upper:
		return held + "above the MUF - come down under " + mhzWords(l.upper)
	case l.luf > 0 && f < l.luf && l.weak:
		return held + "too weak at 100 W - go up to " + mhzWords(l.luf) + "+"
	case l.luf > 0 && f < l.luf:
		return held + "absorbed - go up to " + mhzWords(l.luf) + "+"
	case l.drap > 0 && f < l.drap:
		return held + "disturbed (D-RAP to " + mhzWords(l.drap) + ")"
	}
	return held + "open"
}
