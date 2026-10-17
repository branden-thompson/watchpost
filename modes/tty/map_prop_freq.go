package tty

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
	"github.com/branden-thompson/watchpost/platform/units"
)

// actMapFreq opens the frequency entry over the map (D-178).
const actMapFreq term.Action = "map.frequency"

// The frequency entry's range (FR-1.16, go-ionomaps R-3.5).
const (
	propFreqLow  = 1.8
	propFreqHigh = 30.0
)

// errFreqWords are the entry's words for what it cannot take: what is
// accepted, never the text typed (IS-4).
const errFreqWords = "Not a frequency from 1.8 to 30 MHz, or a band from 160m to 10m."

// parsePropFreq reads a frequency in MHz ("7.188", "7.188 MHz") or a band by
// its metres ("40m", "40 m"), a band taken at its lower edge; false for
// anything else, or a frequency outside HF.
func parsePropFreq(text string) (mhz float64, label string, ok bool) {
	t := strings.ToLower(strings.Join(strings.Fields(text), ""))
	if metres, isBand := strings.CutSuffix(t, "m"); isBand && !strings.HasSuffix(t, "mhz") {
		for _, b := range hfBands {
			if strings.TrimSuffix(b.name, "m") == metres {
				return b.low, b.name, true
			}
		}
		return 0, "", false
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(t, "mhz"), 64)
	if err != nil || !(v >= propFreqLow && v <= propFreqHigh) {
		return 0, "", false
	}
	return v, fmt.Sprintf("%.3f MHz", v), true
}

// propReachKey is what a reach is worked out from.
type propReachKey struct {
	computed, hour time.Time
	origin         ionomaps.LatLon
	mhz            float64
}

// propReach is a frequency's reach from the origin (W5.2), or why there is
// none, which is ours to fix (D-124).
type propReach struct {
	key   propReachKey
	field ionomaps.ReachField
	err   error
}

// propReachMsg is a reach worked out.
type propReachMsg struct{ reach propReach }

// propReachWanted is the reach the window wants: for the frequency held,
// the snapshot shown and the selected place; false with no frequency held
// or nothing to work it out from.
func (d Dashboard) propReachWanted() (propReachKey, bool) {
	key, ok := d.propChartWanted()
	if !ok || d.mapPane.propFreqMHz == 0 {
		return propReachKey{}, false
	}
	return propReachKey{computed: key.computed, hour: key.hour, origin: key.origin, mhz: d.mapPane.propFreqMHz}, true
}

// armPropReach asks for the reach when what it is worked out from has
// changed (about 30 ms, never in Update).
func (d Dashboard) armPropReach() (Dashboard, tea.Cmd) { return d.armAnswer(answerReach) }

// propReachShown is the reach for what the window shows now; false while
// none is held, or it is being worked out.
func (d Dashboard) propReachShown() (ionomaps.ReachField, bool) {
	key, ok := d.propReachWanted()
	r := d.mapPane.propReach
	return r.field, ok && key == r.key && r.err == nil
}

// propReachOverlay is the reach as a two-class host field over the layer
// drawn: go-tuiMaps draws its edge broken, over the filled field its edge
// alone (L-2.6, D-127).
func propReachOverlay(f ionomaps.ReachField, at time.Time) tuimaps.Overlay {
	g := fieldGrid(f.Field)
	g.Type, g.Lines = tuimaps.Type{Breaks: []float64{0.5}}, false // reached or not; no data where a control point has none
	return tuimaps.Overlay{ID: "propagation/reach", Valid: at, Keeps: time.Hour, Grid: &g}
}

// propFreqLine is the block's line for the frequency held (D-179): its
// value, how far the skip zone runs, and the key that changes it; "" while
// none is held.
func (d Dashboard) propFreqLine() string {
	if d.mapPane.propFreqMHz == 0 {
		return ""
	}
	line := "YOUR FREQUENCY " + d.mapPane.propFreqLabel + " · " + d.propReachWords()
	if keys := d.mapKeys[actMapFreq].Keys; len(keys) > 0 {
		line += " · " + keys[0] + " change"
	}
	return line
}

// propReachWords say the frequency held's reach (D-178): its skip zone,
// that it reaches nowhere, or that it is being worked out.
func (d Dashboard) propReachWords() string {
	field, ok := d.propReachShown()
	key, wanted := d.propReachWanted()
	switch {
	case !ok && wanted && key == d.mapPane.propReach.key && d.mapPane.propReach.err != nil:
		return d.propNotAvailable() // AX-9
	case !ok:
		return "being worked out"
	case !field.Reaches:
		return "above the MUF everywhere: it reaches nowhere"
	case field.SkipKm == 0:
		return "no skip zone"
	}
	return "skip zone to ~" + d.roundDistance(float64(field.SkipKm))
}

// roundDistance is a distance in the units the description speaks, to the
// nearest hundred.
func (d Dashboard) roundDistance(km float64) string {
	unit := "km"
	if d.units == render.UnitF {
		km, unit = units.MilesOf(km), "miles"
	}
	n := int(math.Round(km/100) * 100)
	s := strconv.Itoa(n)
	if n >= 1000 {
		s = strconv.Itoa(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
	}
	return s + " " + unit
}

// openPropFreq is F: the frequency entry over the map, in the Propagation
// mode alone, holding the frequency already held.
func (d Dashboard) openPropFreq() Dashboard {
	d.mapPane.propFreqQuery, d.mapPane.propFreqErr = "", ""
	return d.openInPropagation(modalPropFreq)
}

// openInPropagation opens one of the Propagation mode's entries over the
// map: in that mode alone.
func (d Dashboard) openInPropagation(m modal) Dashboard {
	if !d.inPropagation() {
		return d
	}
	return d.open(m)
}

// propFreqWidth is the entry's width: within the 80-column floor.
const propFreqWidth = 60

// propFreqLines are the entry's lines (D-178): what is typed, why it was not
// taken, what is accepted, and its keys.
func (d Dashboard) propFreqLines(o render.Opts) []string {
	lines := []string{"", "  MHz or band: " + d.mapPane.propFreqQuery + o.Glyphs().Cursor, ""}
	if d.mapPane.propFreqErr != "" {
		lines = append(lines, "  "+o.Glyphs().Alert+" "+d.mapPane.propFreqErr, "")
	}
	if d.mapPane.propFreqMHz != 0 {
		lines = append(lines, "  Now drawn: "+d.mapPane.propFreqLabel+". An empty Enter clears it.", "")
	}
	lines = append(lines, "  1.8 to 30 MHz, or a band: 160m to 10m.", "")
	return append(lines, "  "+o.Controls("   ", render.Ctl("enter", "Draw"), render.Ctl("esc", "Cancel")))
}

// handlePropFreqKey is the entry's keyboard: printable keys build what is
// typed, so no binding fires mid-typing; Enter draws it, or with nothing
// typed clears the frequency held; Esc leaves it as it was.
func (d Dashboard) handlePropFreqKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		q := strings.TrimSpace(d.mapPane.propFreqQuery)
		if q == "" {
			d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = 0, ""
			return d.close().setProp().renderMap(), nil
		}
		mhz, label, ok := parsePropFreq(q)
		if !ok {
			d.mapPane.propFreqErr = errFreqWords
			return d, nil
		}
		d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = mhz, label
		return d.close().renderMap(), nil
	case "esc":
		return d.close(), nil
	case "backspace":
		if r := []rune(d.mapPane.propFreqQuery); len(r) > 0 {
			d.mapPane.propFreqQuery = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" && len([]rune(d.mapPane.propFreqQuery)) < 16 {
			d.mapPane.propFreqQuery += key.Text
		}
	}
	d.mapPane.propFreqErr = ""
	return d, nil
}
