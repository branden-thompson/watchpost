package tty

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// propCentreStep is how finely the centre is answered: half a degree, so a
// pan is answered and the smallest nudge is not worked out again.
const propCentreStep = 0.5

// propCentreKey is what the centre's answer is worked out from.
type propCentreKey struct {
	computed, hour time.Time
	origin, centre ionomaps.LatLon
	radiusKm       float64
	noise          ionomaps.Noise // the snapshot's (D-187)
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
	step := func(v float64) float64 { return math.Round(v/propCentreStep) * propCentreStep }
	lon := math.Mod(math.Mod(c.Lon+180, 360)+360, 360) - 180
	return ionomaps.LatLon{Lat: step(c.Lat), Lon: step(lon)}, true
}

// propCentreWanted is the centre's answer the window wants: for the hour
// shown, the selected place and the centre; false with nothing to work it
// out from, or the map not the window shown.
func (d Dashboard) propCentreWanted() (propCentreKey, bool) {
	key, ok := d.propChartWanted()
	if !ok || d.modal != modalMap {
		return propCentreKey{}, false // the library is asked for its centre only in this mode (D-41)
	}
	centre, at := d.propCentreAt()
	if !at {
		return propCentreKey{}, false
	}
	return propCentreKey{computed: key.computed, hour: key.hour, origin: key.origin, centre: centre, radiusKm: key.radiusKm, noise: key.noise}, true
}

// armPropCentre asks for the centre's answer when the map has moved, or
// what it is worked out from has changed.
func (d Dashboard) armPropCentre() (Dashboard, tea.Cmd) { return d.armAnswer(answerCentre) }

// workPropCentre is the centre's answer: within the near-vertical radius of
// the place, the area's; farther, the path's.
func workPropCentre(snap ionomaps.Snapshot, key propCentreKey) propCentre {
	c := propCentre{key: key, km: geo.HaversineKM(key.origin.Lat, key.origin.Lon, key.centre.Lat, key.centre.Lon)}
	var got []ionomaps.Answer
	if c.km <= key.radiusKm {
		got, c.err = snap.Bands([]ionomaps.LatLon{key.origin}, key.radiusKm, key.hour)
	} else {
		got, c.err = snap.Path(key.origin, []ionomaps.LatLon{key.centre}, key.hour)
	}
	if c.err == nil && len(got) == 1 {
		c.bands, c.nearest = got[0].Bands, got[0].Nearest
	}
	return c
}

// propCentreLines are the centre's answer: where it is, MUF(3000) and foF2
// there, and from the selected place the distance, the limit and the
// highest band open; nil while there is none for the snapshot shown.
func (d Dashboard) propCentreLines() []string {
	c, loc := d.mapPane.propCentre, d.selectedLocation()
	shown := d.propOnScreen()
	if loc == nil || len(shown.Hours) == 0 || !c.key.computed.Equal(shown.Computed) || c.err != nil || len(c.bands) == 0 {
		return nil
	}
	ns, ew := "N", "E"
	if c.key.centre.Lat < 0 {
		ns = "S"
	}
	if c.key.centre.Lon < 0 {
		ew = "W"
	}
	out := []string{fmt.Sprintf("%.1f°%s %.1f°%s", math.Abs(c.key.centre.Lat), ns, math.Abs(c.key.centre.Lon), ew)}
	muf, okM := cellValue(shown.Hours[0].MUF3000, c.key.centre.Lat, c.key.centre.Lon)
	fo, okF := cellValue(shown.Hours[0].FoF2, c.key.centre.Lat, c.key.centre.Lon)
	if okM && okF {
		out = append(out, fmt.Sprintf("MUF(3000) %.1f · foF2 %.1f MHz", muf, fo))
	}
	if c.km <= c.key.radiusKm {
		out = append(out, "Near-vertical from "+strings.ToUpper(loc.Label)) // within the radius: the area's answer
	} else {
		out = append(out, "From "+strings.ToUpper(loc.Label)+": "+d.roundDistance(c.km))
	}
	if c.bands[0].Status == ionomaps.StatusNoData {
		return append(out, "no data on the path")
	}
	out = append(out, propRowLimitWords(c.bands)+" · up to "+highestOpen(c.bands))
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
	lines := d.propCentreLines()
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
	box := d.propCentreBox()
	if !d.inPropagation() || d.mapPane.menuOn || box == nil || len(lines) == 0 {
		return lines
	}
	lines = spliceBox(lines, []string{"+"}, size.Rows/2, insetCols+size.Cols/2) // the centre answered
	return spliceBox(lines, box, 0, insetCols)
}

// propCentreWords are the centre's answer said (FR-3.1): last.
func (d Dashboard) propCentreWords() string {
	lines := d.propCentreLines()
	if lines == nil {
		return ""
	}
	return "The map's centre, " + strings.Join(lines, "; ") + "."
}
