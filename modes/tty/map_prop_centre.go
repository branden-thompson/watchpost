package tty

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/propanswers"
)

// propCentreStep is how finely the centre is answered: half a degree, so a
// pan is answered and the smallest nudge is not worked out again.
const propCentreStep = 0.5

// propCentreKey is what the centre's answer is worked out from.
type propCentreKey struct {
	propChartKey // what the chart is worked out from: the snapshot, hour, origin, radius and noise
	centre       ionomaps.LatLon
}

// propCentre is the map centre's answer (FR-1.11, D-180): the bands from
// the selected place to it and how far it is; or why there is none, which
// is ours to fix (D-124).
type propCentre struct {
	key     propCentreKey
	bands   []ionomaps.BandStatus
	nearest ionomaps.Nearest // the answer's nearest station (R-2.11)
	km      float64
	err     error
}

// propCentreMsg is a centre's answer worked out.
type propCentreMsg struct{ centre propCentre }

// propCentreAt is the map's centre, to the step, its longitude wrapped;
// false with no map.
func (d Dashboard) propCentreAt() (ionomaps.LatLon, bool) {
	m := d.mapPane.m
	if m == nil {
		return ionomaps.LatLon{}, false
	}
	var c tuimaps.LonLat
	d.mapPane.call("Centre", func() { c, _ = m.Centre() })
	return propStepped(c.Lat, c.Lon), true
}

// propStepped is a point to the centre's step, its longitude wrapped.
func propStepped(lat, lon float64) ionomaps.LatLon {
	step := func(v float64) float64 { return math.Round(v/propCentreStep) * propCentreStep }
	lon = math.Mod(math.Mod(lon+180, 360)+360, 360) - 180
	return ionomaps.LatLon{Lat: step(lat), Lon: step(lon)}
}

// propCentreWanted is the centre's answer the window wants: for the hour
// shown, the selected place and the centre; false with nothing to work it
// out from, or the map not the window shown.
func (d Dashboard) propCentreWanted() (propCentreKey, bool) {
	key, ok := d.propChartWanted()
	if !ok || d.modal != modalMap || !d.propRole().target {
		return propCentreKey{}, false // the library is asked for its centre only in this mode (D-41); on Observer the block is the centre's (D-193)
	}
	centre, at := d.propCentreAt()
	if !at {
		return propCentreKey{}, false
	}
	return propCentreKey{propChartKey: key, centre: centre}, true
}

// armPropCentre asks for the centre's answer when the map has moved, or
// what it is worked out from has changed.
func (d Dashboard) armPropCentre() (Dashboard, tea.Cmd) { return d.armAnswer(answerCentre) }

// workPropCentre is the centre's answer off the UI goroutine: the answers
// package's (D-212).
func workPropCentre(snap ionomaps.Snapshot, key propCentreKey) propCentre {
	got, err := propanswers.WorkCentre(snap, key.origin, key.centre, key.radiusKm)
	return propCentre{key: key, bands: got.Bands, nearest: got.Nearest, km: got.Km, err: err}
}

// propCentreLines are the Broadcaster's answer for the centre, the target
// (D-193): where it is, MUF(3000) and foF2 there, and from the station - by
// the name given, the box's in capitals, the words' as it is (AX-8) - the
// distance, the bands usable and the limits, and the frequency held; nil on
// Observer, or while there is none for the snapshot shown.
func (d Dashboard) propCentreLines(from string) []string {
	c, origin := d.mapPane.propCentre, d.mapPane.propOrigin
	shown := d.propOnScreen()
	if key, ok := d.propCentreWanted(); !ok || key != c.key { // the answer for what is shown now: the hour, the noise, the station (P-6)
		return nil
	}
	if !d.propRole().target || !origin.ok || !propHasField(shown) || c.err != nil || len(c.bands) == 0 {
		return nil
	}
	out := []string{latLonText(c.key.centre)}
	muf, okM := cellValue(shown.Now.MUF3000, c.key.centre.Lat, c.key.centre.Lon)
	fo, okF := cellValue(shown.Now.FoF2, c.key.centre.Lat, c.key.centre.Lon)
	if okM && okF {
		out = append(out, fmt.Sprintf("MUF(3000) %.1f · foF2 %.1f MHz", muf, fo))
	}
	if c.km <= c.key.radiusKm {
		out = append(out, "Near-vertical from "+from) // within the radius: the area's answer
	} else {
		out = append(out, "From "+from+": "+d.roundDistance(c.km))
	}
	if c.bands[0].Status == ionomaps.StatusNoData {
		return append(out, "no data on the path")
	}
	out = append(out, d.propTarget(c.bands)...)
	return append(out, d.propNearestWords(c.nearest))
}

// propNearestWords say how far the answer's nearest station is, and when
// its readings are unscored, to verify on air (D-181); or that none fed it.
func (d Dashboard) propNearestWords(n ionomaps.Nearest) string {
	if !n.Known {
		return "No station fed this answer"
	}
	w := "Nearest station " + d.roundDistance(float64(n.Km))
	if n.Unscored {
		w += ", unscored: verify on air"
	}
	return w
}

// propCentreBox is the centre's answer boxed for the map's upper left
// (D-180); nil while there is none.
func (d Dashboard) propCentreBox() []string {
	lines := d.propCentreLines(strings.ToUpper(d.mapPane.propOrigin.name))
	if lines == nil {
		return nil
	}
	inner := 0
	for i, l := range lines {
		lines[i] = " " + l
		inner = max(inner, len([]rune(lines[i]))+1)
	}
	return boxed("Centre", lines, inner)
}

// withPropCentre lays the centre's box over the map's upper left and marks
// the centre, in the Propagation mode, while the menu leaves the corner
// free.
func (d Dashboard) withPropCentre(lines []string, size tuimaps.Size) []string {
	if !d.inPropagation() || d.mapPane.menuOn || len(lines) == 0 {
		return lines
	}
	lines = spliceBox(lines, []string{"+"}, size.Rows/2, insetCols+size.Cols/2) // the centre: the origin on Observer, the target on the Broadcaster (D-193)
	if box := d.propCentreBox(); box != nil {
		lines = spliceBox(lines, box, 0, insetCols)
	}
	return lines
}

// propCentreWords are the centre's answer said (FR-3.1): the last of the
// mode's words, the station named as it is, and each clause begun in the
// sentence's lower case.
func (d Dashboard) propCentreWords() string {
	o := d.mapPane.propOrigin
	lines := d.propCentreLines(geo.SpokenPlace(o.name, o.country)) // the station abroad by its country (D-231)
	if c := d.mapPane.propCentre; lines == nil && c.err != nil {
		if key, ok := d.propCentreWanted(); ok && key == c.key {
			return "The map's centre is " + d.propNotAvailable() + "." // AX-9
		}
	}
	if lines == nil {
		return ""
	}
	for i, l := range lines {
		for _, lead := range propCentreLeads {
			if strings.HasPrefix(l, lead) {
				lines[i] = strings.ToLower(lead[:1]) + l[1:]
			}
		}
	}
	return "The map's centre, " + strings.Join(lines, "; ") + "."
}

// propCentreLeads are the box's lines that begin with a capital as a line
// does, and inside the words' sentence do not.
var propCentreLeads = []string{"Near-vertical from ", "From ", "No station ", "Nearest station "}
