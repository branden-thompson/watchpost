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
func (d Dashboard) openPropFind() Dashboard {
	d.mapPane.propFindQuery, d.mapPane.propFindErr = "", ""
	return d.openInPropagation(modalPropFind)
}

// propFindLines are the entry's lines: what is typed, that it is being
// found, why it was not, the place found now, that none is kept, and the
// keys.
func (d Dashboard) propFindLines(o render.Opts) []string {
	lines := []string{"", "  City, ZIP, coordinates or grid locator:", "  " + d.mapPane.propFindQuery + o.Glyphs().Cursor, ""}
	switch {
	case d.mapPane.propFindBusy:
		lines = append(lines, "  Finding it...", "")
	case d.mapPane.propFindErr != "":
		for _, l := range render.WrapText(o.Glyphs().Alert+" "+d.mapPane.propFindErr+". Change it, or press Enter to try again.", propFindWidth-8) { // a lookup that did not answer is tried again as it is (AX-10)
			lines = append(lines, "  "+l)
		}
		lines = append(lines, "")
	}
	if f := d.mapPane.propFound; f != nil {
		lines = append(lines, "  Marked now: "+render.TruncateCells(f.Label, propFindWidth-22)+".", "  An empty Enter takes the mark off.", "")
	}
	lines = append(lines, "  Anywhere on the globe. A found place is", "  not kept: find it again at any time.", "")
	return append(lines, "  "+o.Controls("   ", render.Ctl("enter", "Find"), render.Ctl("esc", "Cancel")))
}

// handlePropFindKey is the entry's keyboard: printable keys build what is
// typed, so no binding fires mid-typing; Enter finds it - off the UI
// goroutine, the lookup may go online - or with nothing typed takes the
// mark off; Esc leaves everything as it was.
func (d Dashboard) handlePropFindKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		q := strings.TrimSpace(d.mapPane.propFindQuery)
		if q == "" {
			return d.markFound(nil).close().renderMap(), nil
		}
		find := d.cfg.FindPlace
		if find == nil || d.mapPane.propFindBusy {
			return d, nil
		}
		d.mapPane.propFindSeq++
		d.mapPane.propFindBusy, d.mapPane.propFindErr = true, ""
		seq := d.mapPane.propFindSeq
		ctx, cancel := context.WithCancel(context.Background())
		d.mapPane.propFindCancel = cancel
		return d, func() tea.Msg {
			defer cancel()
			ref, err := find(ctx, q)
			return propFoundMsg{seq: seq, ref: ref, err: err}
		}
	case "esc":
		return d.dropFind().close(), nil
	case "backspace":
		if r := []rune(d.mapPane.propFindQuery); len(r) > 0 {
			d.mapPane.propFindQuery = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" && len([]rune(d.mapPane.propFindQuery)) < propFindMost {
			d.mapPane.propFindQuery += key.Text
		}
	}
	d.mapPane.propFindErr = ""
	return d, nil
}

// dropFind cancels the find in flight, if one is: its answer is dropped
// (AX-11) and its request stopped with it - a query the listener took back,
// or one from a session since ended (D-27), is not sent on.
func (d Dashboard) dropFind() Dashboard {
	if !d.mapPane.propFindBusy {
		return d
	}
	d.mapPane.propFindSeq++
	d.mapPane.propFindBusy = false
	if d.mapPane.propFindCancel != nil {
		d.mapPane.propFindCancel()
	}
	return d
}

// applyPropFound takes a find's answer: a failure is said in the entry,
// which stays open to change what was typed; a place is marked, and centred
// on the Broadcaster alone (D-204); the entry closes and the map has the
// focus again.
func (d Dashboard) applyPropFound(v propFoundMsg) (tea.Model, tea.Cmd) {
	if v.seq != d.mapPane.propFindSeq {
		return d, nil // a later find's answer is the one wanted
	}
	d.mapPane.propFindBusy = false
	if v.err != nil {
		d.mapPane.propFindErr = v.err.Error()
		if cause := errors.Unwrap(v.err); cause != nil {
			d.problem("Propagation finder: " + cause.Error()) // the cause for the diagnostics; the words above are the listener's (D-124)
		}
		return d, nil
	}
	ref := v.ref
	d = d.markFound(&ref)
	if d.modal == modalPropFind {
		d = d.close() // focus back on the map (FR-1.10)
	}
	if m := d.mapPane.m; m != nil && d.propRole().target { // the Broadcaster's target is the centre; on the Observer the place is a target row and the map stays (D-204)
		d.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: ref.Lon, Lat: ref.Lat}) })
	}
	d, settle := d.viewMoved()
	d = d.renderMap()
	return d, tea.Batch(d.mapWorkCmd(), settle)
}

// markFound marks the place found, or with nil takes the mark off: a ring,
// named for it, on the map's places (D-147: the map's own marks).
func (d Dashboard) markFound(ref *snapshot.LocationRef) Dashboard {
	d.mapPane.propFound = ref
	m := d.mapPane.m
	if m == nil {
		return d
	}
	if d.mapPane.foundMarked {
		d.mapPane.call("RemovePlace", func() { _, _ = m.RemovePlace(foundPlaceID) })
		d.mapPane.foundMarked = false
	}
	if ref == nil {
		return d
	}
	var err error
	d.mapPane.call("AddPlace", func() {
		_, err = m.AddPlace(tuimaps.Place{ID: foundPlaceID, Name: ref.Label, At: tuimaps.LonLat{Lon: ref.Lon, Lat: ref.Lat}, Marker: tuimaps.MarkerRing})
	})
	if err != nil {
		d.problem("Propagation: the found place not marked - " + err.Error()) // ours to fix, never the listener's (D-124)
		return d
	}
	d.mapPane.foundMarked = true
	return d
}

// propFoundWords say the place found, while it is marked (FR-1.10).
func (d Dashboard) propFoundWords() string {
	f := d.mapPane.propFound
	if f == nil {
		return ""
	}
	if !d.propRole().target {
		return "Found " + f.Label + ": marked on the map and answered with the targets, and not kept." // a target row, the map where it was (D-204)
	}
	if c, ok := d.propCentreAt(); ok && propStepped(f.Lat, f.Lon) != c {
		return "Found " + f.Label + ": marked on the map, and not kept." // panned away: the centre is another place (AX-1)
	}
	return "Found " + f.Label + ": marked at the map's centre, and not kept."
}
