package tty

// map_prop_find.go — finding a place in the Propagation mode (0.19.0 W5.3,
// FR-1.10; D-25, D-27, D-61, D-79): a city, a ZIP, coordinates or a grid
// locator, anywhere on the globe. The place found is marked and said; on the
// Observer it joins the chart as a target, answered from the origin, and the
// map stays where it is; on the Broadcaster it is centred, its target being
// the centre (D-193, D-204). It is never kept, and can always be found again
// (D-27). Focus returns to the map.

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// actMapFind is /: find a place, in the Propagation mode.
const actMapFind term.Action = "map.find"

// foundPlaceID is the found place's mark on the map.
const foundPlaceID = "propagation/found"

// propFindWidth is the entry's width: within the 80-column floor.
const propFindWidth = 60

// propFindMost is the most a query may hold: inside the entry's width.
const propFindMost = 48

// propFoundMsg is a find's answer: the place, or why there is none; seq is
// the find it answers, so a later find's answer wins.
type propFoundMsg struct {
	seq uint64
	ref snapshot.LocationRef
	err error
}

// openPropFind is /: the find entry over the map, in the Propagation mode
// alone.
func (mm mapModel) openPropFind(e mapEnv) (mapModel, mapWant) {
	return mm.openPropEntry(mapModalPropFind, e)
}

// propFindLines are the entry's lines: what is typed, that it is being
// found, why it was not, the place found now, that none is kept, and the
// keys.
func (mm mapModel) propFindLines(o render.Opts) []string {
	lines := []string{"", "  City, ZIP, coordinates or grid locator:", "  " + mm.mapPane.propFindQuery + o.Glyphs().Cursor, ""}
	switch {
	case mm.mapPane.propFindBusy:
		lines = append(lines, "  Finding it...", "")
	case mm.mapPane.propFindErr != "":
		for _, l := range render.WrapText(o.Glyphs().Alert+" "+mm.mapPane.propFindErr+". Change it, or press Enter to try again.", propFindWidth-8) { // a lookup that did not answer is tried again as it is (AX-10)
			lines = append(lines, "  "+l)
		}
		lines = append(lines, "")
	}
	if f := mm.mapPane.propFound; f != nil {
		lines = append(lines, "  Marked now: "+render.TruncateCells(f.Label, propFindWidth-22)+".", "  An empty Enter takes the mark off.", "")
	}
	lines = append(lines, "  Anywhere on the globe. A found place is", "  not kept: find it again at any time.", "")
	return append(lines, "  "+o.Controls("   ", render.Ctl("enter", "Find"), render.Ctl("esc", "Cancel")))
}

// onPropFindKey is the entry's keyboard: printable keys build what is
// typed, so no binding fires mid-typing; Enter finds it - off the UI
// goroutine, the lookup may go online - or with nothing typed takes the
// mark off; Esc leaves everything as it was.
func (mm mapModel) onPropFindKey(key tea.KeyPressMsg, e mapEnv) (mapModel, tea.Cmd, mapWant) {
	switch key.String() {
	case "enter":
		q := strings.TrimSpace(mm.mapPane.propFindQuery)
		if q == "" {
			return mm.markFound(nil, e), nil, mapWant{Close: true, Then: mapStepRender}
		}
		find := mm.mapHooks.FindPlace
		if find == nil || mm.mapPane.propFindBusy {
			return mm, nil, mapWant{}
		}
		mm.mapPane.propFindSeq++
		mm.mapPane.propFindBusy, mm.mapPane.propFindErr = true, ""
		seq := mm.mapPane.propFindSeq
		ctx, cancel := context.WithCancel(context.Background())
		mm.mapPane.propFindCancel = cancel
		return mm, func() tea.Msg {
			defer cancel()
			ref, err := find(ctx, q)
			return propFoundMsg{seq: seq, ref: ref, err: err}
		}, mapWant{}
	case "esc":
		return mm.dropFind(), nil, mapWant{Close: true}
	case "backspace":
		mm.mapPane.propFindQuery = eraseRune(mm.mapPane.propFindQuery)
	default:
		if key.Text != "" && len([]rune(mm.mapPane.propFindQuery)) < propFindMost {
			mm.mapPane.propFindQuery += key.Text
		}
	}
	mm.mapPane.propFindErr = ""
	return mm, nil, mapWant{}
}

// dropFind cancels the find in flight, if one is: its answer is dropped
// (AX-11) and its request stopped with it - a query the listener took back,
// or one from a session since ended (D-27), is not sent on.
func (mm mapModel) dropFind() mapModel {
	if !mm.mapPane.propFindBusy {
		return mm
	}
	mm.mapPane.propFindSeq++
	mm.mapPane.propFindBusy = false
	if mm.mapPane.propFindCancel != nil {
		mm.mapPane.propFindCancel()
	}
	return mm
}

// applyPropFound takes a find's answer: a failure is said in the entry,
// which stays open to change what was typed; a place is marked, and centred
// on the Broadcaster alone (D-204); the entry closes and the map has the
// focus again.
func (mm mapModel) applyPropFound(v propFoundMsg, e mapEnv) (mapModel, tea.Cmd, mapWant) {
	if v.seq != mm.mapPane.propFindSeq {
		return mm, nil, mapWant{} // a later find's answer is the one wanted
	}
	mm.mapPane.propFindBusy = false
	if v.err != nil {
		mm.mapPane.propFindErr = v.err.Error()
		if cause := errors.Unwrap(v.err); cause != nil {
			mm.problem("Propagation finder: "+cause.Error(), e) // the cause for the diagnostics; the words above are the listener's (D-124)
		}
		return mm, nil, mapWant{}
	}
	ref := v.ref
	mm = mm.markFound(&ref, e)
	if e.Window == mapModalPropFind {
		return mm, nil, mapWant{Close: true, Then: mapStepFoundRecentre} // focus back on the map (FR-1.10), then the place
	}
	mm, cmd := mm.foundRecentre(e)
	return mm, cmd, mapWant{}
}

// foundRecentre is the map on the place found (0.19.6 design MS-1's S3),
// the finder closed: recentred where the role centres on it, and drawn.
func (mm mapModel) foundRecentre(e mapEnv) (mapModel, tea.Cmd) {
	ref := *mm.mapPane.propFound
	if m := mm.mapPane.m; m != nil && mm.propRole(e).target { // the Broadcaster's target is the centre; on the Observer the place is a target row and the map stays (D-204)
		mm.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: ref.Lon, Lat: ref.Lat}) })
	}
	mm, settle := mm.viewMoved(e)
	mm = mm.renderMap(e)
	return mm, tea.Batch(mm.mapWorkCmd(e), settle)
}

// markFound marks the place found, or with nil takes the mark off: a ring,
// named for it, on the map's places (D-147: the map's own marks).
func (mm mapModel) markFound(ref *snapshot.LocationRef, e mapEnv) mapModel {
	mm.mapPane.propFound = ref
	m := mm.mapPane.m
	if m == nil {
		return mm
	}
	if mm.mapPane.foundMarked {
		mm.mapPane.call("RemovePlace", func() { _, _ = m.RemovePlace(foundPlaceID) })
		mm.mapPane.foundMarked = false
	}
	if ref == nil {
		return mm
	}
	var err error
	mm.mapPane.call("AddPlace", func() {
		_, err = m.AddPlace(tuimaps.Place{ID: foundPlaceID, Name: ref.Label, At: tuimaps.LonLat{Lon: ref.Lon, Lat: ref.Lat}, Marker: tuimaps.MarkerRing})
	})
	if err != nil {
		mm.problem("Propagation: the found place not marked - "+err.Error(), e) // ours to fix, never the listener's (D-124)
		return mm
	}
	mm.mapPane.foundMarked = true
	return mm
}

// propFoundWords say the place found, while it is marked (FR-1.10).
func (mm mapModel) propFoundWords(e mapEnv) string {
	f := mm.mapPane.propFound
	if f == nil {
		return ""
	}
	if !mm.propRole(e).target {
		return "Found " + f.Label + ": marked on the map and answered with the targets, and not kept." // a target row, the map where it was (D-204)
	}
	if c, ok := mm.propCentreAt(); ok && propStepped(f.Lat, f.Lon) != c {
		return "Found " + f.Label + ": marked on the map, and not kept." // panned away: the centre is another place (AX-1)
	}
	return "Found " + f.Label + ": marked at the map's centre, and not kept."
}
