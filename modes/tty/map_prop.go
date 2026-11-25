package tty

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// actMapProp enters and leaves the Propagation mode (D-153).
const actMapProp term.Action = "map.propagation"

// The Propagation mode's words while it has nothing to draw: its badge, its
// line in the radar's place, and its sentence in the description.
const (
	propagationLabel  = "RADIO FREQ. PROPAGATION" // D-167
	propagationStatus = "Propagation · whole world · no propagation data drawn yet"
	propagationWords  = "The Propagation mode, over the whole world: no propagation data is drawn yet."
)

// togglePropagation enters the Propagation mode, or leaves it for the mode it
// came from (D-153). It is never saved (D-154). Entering it takes every
// weather layer off and the region's bound away (D-21); leaving it asks the
// weather mode's data again, as R does.
func (mm mapModel) togglePropagation(e mapEnv) (mapModel, tea.Cmd, mapWant) {
	mm.mapPane.prop = !mm.mapPane.prop
	mm, cmd := mm.enterMode(e)
	if mm.mapMode(e) == modePropagation && mm.propAckDue() {
		return mm, cmd, mm.openPropAck() // before anything of the mode's is fetched (D-81): closing it asks
	}
	mm, update := mm.askPropagation(e)
	return mm, tea.Batch(cmd, update), mapWant{}
}

// enterMode sets the map up for the mode it is now in: its bound, its layers
// and its playback, and asks what the mode draws.
func (mm mapModel) enterMode(e mapEnv) (mapModel, tea.Cmd) {
	mm = mm.resetPlayback()
	mm.mapPane.gen++
	mm = mm.ensureMainOverlay(e)
	mm = mm.followSelection(e).setTemp(e).setProp(e)
	mm = mm.refreshMapCost(e).showStep(e).retimeDrawn(e).requestFeed()
	mm, feed := mm.askFeed(e)
	mm, radar := mm.askRadar(e)
	switch mm.mapMode(e) {
	case modePropagation:
		return mm, tea.Batch(feed, radar, mm.mapWorkCmd(e)) // nothing of the weather's is asked for
	case modeRadar, modeForecast:
		mm = mm.stopPropagation(e) // the Propagation mode's update ends with it (FR-4.2)
	}
	mm, temp := mm.askTemp(e)
	return mm, tea.Batch(feed, radar, temp, mm.mapWorkCmd(e))
}

// propagationBadge is the Propagation mode's badge in the radar badge's place.
func (mm mapModel) propagationBadge() string {
	return mapBadge(render.Tint(propagationLabel, render.Tok(render.ModalTitle)), "", "")
}

// propChipWords are the P chip's words: whether the Propagation mode is on.
func (mm mapModel) propChipWords(e mapEnv) string {
	switch mm.mapMode(e) {
	case modePropagation:
		return "Propagation On"
	case modeRadar, modeForecast:
	}
	return "Propagation Off"
}

// outsideText is the window's words for a place in no region: no weather map
// is drawn for it, and the key that opens the Propagation mode, as bound,
// which covers the whole world (D-21).
func (mm mapModel) outsideText() string {
	text := mm.mapPane.outside + " is outside every region the weather map covers - the contiguous United States, Alaska, Hawaii, Puerto Rico and the Virgin Islands, Guam and the Northern Marianas, and American Samoa - so no weather map is drawn for it."
	if keys := mm.mapKeys[actMapProp].Keys; len(keys) > 0 {
		text += " Press " + keys[0] + " for the Propagation mode, which covers the whole world."
	}
	return text
}

// outsideStated reports whether the window states the place is outside every
// region in place of the map: in a weather mode, for a place in no region.
func (mm mapModel) outsideStated(e mapEnv) bool {
	if mm.mapPane.outside == "" {
		return false
	}
	switch mm.mapMode(e) {
	case modeRadar, modeForecast:
		return true
	case modePropagation:
	}
	return false
}

// The Overlays menu's row for the Propagation mode (D-158, A-37).
const (
	propagationRowKey   = "propagation"
	propagationRowLabel = "Radio Propagation"
)

// radioChangesMode is space or enter on a tint that moves between the modes
// (D-158): the Propagation row is P - into the mode, or, chosen, back out to
// the weather mode it came from; a weather tint chosen in the Propagation
// mode returns to that weather mode with the tint drawn. false for any other
// key or row, which the menu handles as before.
func (mm mapModel) radioChangesMode(key string, e mapEnv) (mapModel, tea.Cmd, mapWant, bool) {
	rows := mm.overlayRows(e)
	if len(rows) == 0 || (key != "space" && key != "enter") {
		return mm, nil, mapWant{}, false
	}
	r := rows[mm.mapPane.menuAt%len(rows)]
	if r.kind != menuRadio {
		return mm, nil, mapWant{}, false
	}
	at := mm.mapPane.menuAt
	switch {
	case r.key == propagationRowKey:
		nd, cmd, w := mm.togglePropagation(e)
		nd.mapPane.menuAt = at    // the tints lead the menu in both modes: the cursor stays on its row
		w.Then = mapStepMenuFocus // over the window the toggle may have opened
		return nd, cmd, w, true
	case mm.mapMode(e) == modePropagation:
		nd, w := mm.chooseTint(r.key, e) // chosen while this mode draws none of it: leaving draws and asks for it, once
		nd, cmd, toggled := nd.timeFrom("overlay", e).togglePropagation(e)
		nd.mapPane.menuAt = at
		w = w.and(toggled)
		w.Settings.Save, w.Settings.SaveLast = true, true // saved after the mode's commands, as the menu saves
		w.Then = mapStepMenuFocus
		return nd, cmd, w, true
	}
	return mm, nil, mapWant{}, false
}

// chooseTint draws a weather tint: chosen, the others not; never cleared,
// as space on a chosen tint would clear it (D-142).
func (mm mapModel) chooseTint(key string, e mapEnv) (mapModel, mapWant) {
	if key == TemperatureLayer && mm.pickFeels(e) {
		key = FeelsLayer
	}
	mm.mapPane.tempAuto = false
	return mm.withChoice(func(c map[string]bool) {
		for _, t := range oneTint {
			c[t] = false
		}
		c[key] = true
	}, e)
}
