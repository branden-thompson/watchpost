package tty

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/term"
)

// propAheadMost is the furthest hour ahead (FR-1.13, go-ionomaps R-1.3).
const propAheadMost = 24

// propAheadMsg is an hour ahead worked out - a snapshot of its own, typical,
// with none of now's measurements - for the snapshot computed then.
type propAheadMsg struct {
	computed time.Time
	k        int
	snap     ionomaps.Snapshot
	err      error
}

// stepPropAhead is the hours' keys in the Propagation mode (FR-1.13, A-48):
// shift+right an hour on, to 24; shift+left an hour back, to now; n now.
// Nothing is fetched: an hour ahead is worked out from the snapshot held.
func (d Dashboard) stepPropAhead(act term.Action) (Dashboard, bool) {
	switch act {
	case actMapOn:
		d.mapPane.propAhead = min(d.mapPane.propAhead+1, propAheadMost)
	case actMapBack:
		d.mapPane.propAhead = max(d.mapPane.propAhead-1, 0)
	case actMapNewest:
		d.mapPane.propAhead = 0
	default:
		return d, false
	}
	d.mapPane.gen++
	return d.setProp().renderMap(), true
}

// armPropAhead asks for the hour ahead shown when it is not yet worked out,
// once, in a command (FR-1.17): about 35 ms, never in Update.
func (d Dashboard) armPropAhead() (Dashboard, tea.Cmd) {
	k, snap := d.mapPane.propAhead, d.mapPane.propSnap
	if k == 0 || snap == nil || !propHasField(*snap) || k == d.mapPane.propAheadAsked {
		return d, nil
	}
	if _, ok := d.mapPane.propAheadSnaps[k]; ok {
		return d, nil
	}
	d.mapPane.propAheadAsked = k
	held := *snap
	return d, func() tea.Msg {
		ahead, err := held.Ahead(k)
		return propAheadMsg{computed: held.Computed, k: k, snap: ahead, err: err}
	}
}

// applyPropAhead keeps an hour ahead of the snapshot held; one of an older
// snapshot is dropped, and a refused one said in diagnostics (D-124).
func (d Dashboard) applyPropAhead(v propAheadMsg) Dashboard {
	if d.mapPane.propSnap == nil || !v.computed.Equal(d.mapPane.propSnap.Computed) {
		return d
	}
	if v.k == d.mapPane.propAheadAsked {
		d.mapPane.propAheadAsked = 0
	}
	if v.err != nil {
		d.problem("Propagation: the hour ahead not worked out - " + v.err.Error())
		return d
	}
	if d.mapPane.propAheadSnaps == nil {
		d.mapPane.propAheadSnaps = map[int]ionomaps.Snapshot{}
	}
	d.mapPane.propAheadSnaps[v.k] = v.snap
	d.mapPane.gen++
	d = d.setProp()
	if d.modal == modalMap {
		return d.renderMap()
	}
	return d
}

// propOnScreen is the snapshot as the window draws it: now's, or the hour
// ahead's own snapshot once worked out - typical, carrying none of now's
// measurements (L-F2) - now's standing in while it is worked out; empty with
// no snapshot.
func (d Dashboard) propOnScreen() ionomaps.Snapshot {
	snap := d.mapPane.propSnap
	if snap == nil || !propHasField(*snap) {
		return ionomaps.Snapshot{}
	}
	if ahead, ok := d.mapPane.propAheadSnaps[d.mapPane.propAhead]; ok && d.mapPane.propAhead > 0 {
		return ahead
	}
	return *snap
}

// propHasField is whether a snapshot holds an hour to draw and answer from:
// go-ionomaps gives one with no hour at all when it has no field
// (ErrNoField).
func propHasField(s ionomaps.Snapshot) bool { return !s.Now.At.IsZero() }

// propShown is the snapshot on screen, and whether it is the hour asked
// for: false while an hour ahead is being worked out, or with no snapshot.
func (d Dashboard) propShown() (ionomaps.Snapshot, bool) {
	shown := d.propOnScreen()
	_, ready := d.mapPane.propAheadSnaps[d.mapPane.propAhead]
	return shown, propHasField(shown) && (d.mapPane.propAhead == 0 || ready)
}

// propTypicalError is the typical error of an hour ahead at the place, in
// the layer drawn (D-134): the distance band of the nearest station to the
// place, or the farthest band where no station is near.
func (d Dashboard) propTypicalError(lat, lon float64) (float32, bool) {
	snap := d.mapPane.propSnap
	if snap == nil || len(snap.TypicalError.Bands) == 0 {
		return 0, false
	}
	bands := snap.TypicalError.Bands
	pick := bands[len(bands)-1]
	if f := snap.Now.FoF2; len(snap.NearestKm) == f.Cols*f.Rows { // the stations' distances, on the fields' grid
		f.Values, f.NoData = snap.NearestKm, ionomaps.Bitset{}
		if km, ok := cellValue(f, lat, lon); ok {
			for _, b := range bands {
				if b.UpToKm > 0 && float32(km) <= b.UpToKm {
					pick = b
					break
				}
			}
		}
	}
	if d.mapPane.propLayer == propFoF2 {
		return pick.FoF2MHz, true
	}
	return pick.MUF3000MHz, true
}

// propAheadLabel is the mode line's words for an hour ahead (FR-1.13,
// D-109): the hour, typical and not a forecast, and its typical error at
// the place; "" for now.
func (d Dashboard) propAheadLabel(lat, lon float64) string {
	k := d.mapPane.propAhead
	if k == 0 || d.mapPane.propSnap == nil {
		return ""
	}
	at := d.mapPane.propSnap.Now.At.Add(time.Duration(k) * time.Hour).UTC()
	label := fmt.Sprintf("+%d h %s UTC · TYPICAL, NOT A FORECAST", k, at.Format("15:04"))
	if _, shown := d.propShown(); !shown {
		label += " · being worked out"
	} else if e, ok := d.propTypicalError(lat, lon); ok {
		label += fmt.Sprintf(" · ±%.1f MHz", e)
	}
	if propStorm(d.mapPane.propSnap.Scales) {
		label += " · STORM: TYPICAL MAY NOT HOLD" // FR-1.18, D-109
	}
	return label
}

// propAheadWords are the words for an hour ahead: typical for this hour,
// not a forecast, its typical error and what that was measured on (D-134).
func (d Dashboard) propAheadWords(lat, lon float64) string {
	k := d.mapPane.propAhead
	if k == 0 || d.mapPane.propSnap == nil {
		return ""
	}
	words := []string{hoursWords(k) + " ahead: typical for this hour, not a forecast"}
	if propStorm(d.mapPane.propSnap.Scales) {
		words = append(words, "a geomagnetic storm is in progress: typical conditions may not hold") // FR-1.18, D-109
	}
	if e, ok := d.propTypicalError(lat, lon); ok {
		words = append(words, fmt.Sprintf("its typical error at the place is %.1f MHz in %s", e, propLayerName(d.mapPane.propLayer)))
	}
	if b := d.mapPane.propSnap.TypicalError.Basis; b != "" {
		words = append(words, b) // in words a listener can be told; the citation is the library's Source (AX-3)
	}
	return strings.Join(words, "; ") + "."
}

// hoursWords is a count of hours said: "1 hour", "3 hours".
func hoursWords(k int) string {
	if k == 1 {
		return "1 hour"
	}
	return strconv.Itoa(k) + " hours"
}
