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
	made   propMade // the snapshot as kept (L-F3)
	hour   time.Time
	origin ionomaps.LatLon
	mhz    float64
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
func (mm mapModel) propReachWanted(e mapEnv) (propReachKey, bool) {
	key, ok := mm.propChartWanted(e)
	if !ok || mm.mapPane.propFreqMHz == 0 {
		return propReachKey{}, false
	}
	return propReachKey{made: key.made, hour: key.hour, origin: key.origin, mhz: mm.mapPane.propFreqMHz}, true
}

// armPropReach asks for the reach when what it is worked out from has
// changed (about 0.2 s, never in Update; D-227), one at a time.
func (mm mapModel) armPropReach(e mapEnv) (mapModel, tea.Cmd) {
	if mm.viewSettling() {
		return mm, nil
	}
	if mm.mapPane.propReachAsked != (propReachKey{}) && mm.mapPane.propReachAsked != mm.mapPane.propReach.key {
		return mm, nil // one reach at a time (D-227): the one asked is answered first, then the latest wanted is asked
	}
	key, ok := mm.propReachWanted(e)
	cmd := askAnswer(&mm, key, ok, mm.mapPane.propReach.key, &mm.mapPane.propReachAsked, func(snap ionomaps.Snapshot, key propReachKey) tea.Msg {
		field, err := snap.Reach(key.origin, key.mhz)
		return propReachMsg{reach: propReach{key: key, field: field, err: err}}
	})
	return mm, cmd
}

// propReachShown is the reach for what the window shows now; false while
// none is held, or it is being worked out.
func (mm mapModel) propReachShown(e mapEnv) (ionomaps.ReachField, bool) {
	key, ok := mm.propReachWanted(e)
	r := mm.mapPane.propReach
	return r.field, ok && key == r.key && r.err == nil
}

// propReachOverlay is the reach as a two-class host field over the layer
// drawn, asked for as its edge alone: go-tuiMaps draws it broken, over the
// filled field (L-2.6, D-127; go-tuiMaps v0.3.0 `Type.Edge`).
func propReachOverlay(f ionomaps.ReachField, at time.Time) tuimaps.Overlay {
	g := fieldGrid(f.Field)
	g.Type, g.Lines = tuimaps.Type{Breaks: []float64{0.5}, Edge: true}, false // reached or not; no data where a control point has none
	return tuimaps.Overlay{ID: "propagation/reach", Valid: at, Keeps: time.Hour, Grid: &g}
}

// propFreqLine is the block's line for the frequency held (D-179): its
// value, how far the skip zone runs, and the key that changes it; "" while
// none is held.
func (mm mapModel) propFreqLine(e mapEnv) string {
	if mm.mapPane.propFreqMHz == 0 {
		return ""
	}
	line := "YOUR FREQUENCY " + mm.mapPane.propFreqLabel + " · " + mm.propReachRow(e)
	if keys := mm.mapKeys[actMapFreq].Keys; len(keys) > 0 {
		line += " · " + keys[0] + " change"
	}
	return line
}

// propReachWords say the frequency held's reach (D-178): its skip zone and
// the limit that stops it, that it reaches nowhere and why, or that it is
// being worked out - the words in full, the block's row short (propReachRow).
func (mm mapModel) propReachWords(e mapEnv) string { return mm.propReach(true, e) }

// propReachRow is the reach for the block's row: a reach that goes
// somewhere without its limit, which the words say, so the row keeps its key.
func (mm mapModel) propReachRow(e mapEnv) string { return mm.propReach(false, e) }

// propReach is the reach said, in full or short.
func (mm mapModel) propReach(full bool, e mapEnv) string {
	field, ok := mm.propReachShown(e)
	key, wanted := mm.propReachWanted(e)
	switch {
	case !ok && wanted && key == mm.mapPane.propReach.key && mm.mapPane.propReach.err != nil:
		return mm.propNotAvailable() // AX-9
	case !ok:
		return "being worked out"
	}
	return reachSaid(field, mm.roundDistance(float64(field.SkipKm), e), full)
}

// reachLimitWords are the words for the limit a reach names (D-205): where
// it reaches nowhere, why; where it reaches, what bounds it.
var reachLimitWords = map[ionomaps.Limit][2]string{
	ionomaps.LimitMUF:        {"above the MUF everywhere", "the MUF"},
	ionomaps.LimitAbsorption: {"absorbed everywhere", "absorption"},
	ionomaps.LimitCircuit:    {"too weak at 100 W everywhere", "the signal too weak at 100 W"},
}

// reachSaid says a reach worked out (D-178, FR-1.16, D-205): how far its
// skip zone runs and, in full, the limit that bounds it; or that it reaches
// nowhere and which limit stops it - never the MUF when the absorption or
// the circuit does.
func reachSaid(f ionomaps.ReachField, skip string, full bool) string {
	words, named := reachLimitWords[f.Limit]
	if !named {
		words = reachLimitWords[ionomaps.LimitMUF]
	}
	reached := "skip zone to ~" + skip
	if f.SkipKm == 0 {
		reached = "no skip zone"
	}
	switch {
	case !f.Reaches:
		return "it reaches nowhere: " + words[0]
	case !full:
		return reached
	}
	return reached + ", limited by " + words[1]
}

// roundDistance is a distance in the units the description speaks, to the
// nearest hundred.
func (mm mapModel) roundDistance(km float64, e mapEnv) string {
	unit := "km"
	if e.Opts.Units == render.UnitF {
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
func (mm mapModel) openPropFreq(e mapEnv) (mapModel, mapWant) {
	return mm.openPropEntry(mapModalPropFreq, e)
}

// openPropEntry opens one of the Propagation mode's two entries over the map,
// its field emptied: in that mode alone.
func (mm mapModel) openPropEntry(w mapModal, e mapEnv) (mapModel, mapWant) {
	if w == mapModalPropFind {
		mm.mapPane.propFindQuery, mm.mapPane.propFindErr = "", ""
	} else {
		mm.mapPane.propFreqQuery, mm.mapPane.propFreqErr = "", ""
	}
	if !mm.inPropagation(e) {
		return mm, mapWant{} // the Propagation mode's entry, in that mode alone
	}
	return mm, mapWant{Open: w}
}

// propFreqWidth is the entry's width: within the 80-column floor.
const propFreqWidth = 60

// propFreqLines are the entry's lines (D-178): what is typed, why it was not
// taken, what is accepted, and its keys.
func (mm mapModel) propFreqLines(o render.Opts) []string {
	lines := []string{"", "  MHz or band: " + mm.mapPane.propFreqQuery + o.Glyphs().Cursor, ""}
	if mm.mapPane.propFreqErr != "" {
		lines = append(lines, "  "+o.Glyphs().Alert+" "+mm.mapPane.propFreqErr, "")
	}
	if mm.mapPane.propFreqMHz != 0 {
		lines = append(lines, "  Now drawn: "+mm.mapPane.propFreqLabel+". An empty Enter clears it.", "")
	}
	lines = append(lines, "  1.8 to 30 MHz, or a band: 160m to 10m.", "")
	return append(lines, "  "+o.Controls("   ", render.Ctl("enter", "Draw"), render.Ctl("esc", "Cancel")))
}

// onPropFreqKey is the entry's keyboard: printable keys build what is
// typed, so no binding fires mid-typing; Enter draws it, or with nothing
// typed clears the frequency held; Esc leaves it as it was.
func (mm mapModel) onPropFreqKey(key tea.KeyPressMsg) (mapModel, tea.Cmd, mapWant) {
	switch key.String() {
	case "enter":
		q := strings.TrimSpace(mm.mapPane.propFreqQuery)
		if q == "" {
			mm.mapPane.propFreqMHz, mm.mapPane.propFreqLabel = 0, ""
			return mm, nil, mapWant{Close: true, Then: mapStepSetPropRender}
		}
		mhz, label, ok := parsePropFreq(q)
		if !ok {
			mm.mapPane.propFreqErr = errFreqWords
			return mm, nil, mapWant{}
		}
		mm.mapPane.propFreqMHz, mm.mapPane.propFreqLabel = mhz, label
		return mm, nil, mapWant{Close: true, Then: mapStepRender}
	case "esc":
		return mm, nil, mapWant{Close: true}
	case "backspace":
		mm.mapPane.propFreqQuery = eraseRune(mm.mapPane.propFreqQuery)
	default:
		if key.Text != "" && len([]rune(mm.mapPane.propFreqQuery)) < 16 {
			mm.mapPane.propFreqQuery += key.Text
		}
	}
	mm.mapPane.propFreqErr = ""
	return mm, nil, mapWant{}
}
