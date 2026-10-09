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
	propagationLabel  = "PROPAGATION"
	propagationStatus = "Propagation · whole world · no propagation data drawn yet"
	propagationWords  = "This is the Propagation mode, over the whole world. No propagation data is drawn yet."
)

// togglePropagation enters the Propagation mode, or leaves it for the mode it
// came from (D-153). It is never saved (D-154). Entering it takes every
// weather layer off and the region's bound away (D-21); leaving it asks the
// weather mode's data again, as R does.
func (d Dashboard) togglePropagation() (Dashboard, tea.Cmd) {
	d.mapPane.prop = !d.mapPane.prop
	return d.enterMode()
}

// enterMode sets the map up for the mode it is now in: its bound, its layers
// and its playback, and asks what the mode draws.
func (d Dashboard) enterMode() (Dashboard, tea.Cmd) {
	d.mapPane.fcStep, d.mapPane.fcPlaying = 0, false
	d.mapPane.fcGen++
	d.mapPane.tempAuto = false
	d.mapPane.gen++
	d = d.ensureMainOverlay()
	d = d.followSelection().setTemp()
	d = d.refreshMapCost().showStep().retimeDrawn().requestFeed()
	d, feed := d.askFeed()
	d, radar := d.askRadar()
	switch d.mapMode() {
	case modePropagation:
		return d, tea.Batch(feed, radar, d.mapWorkCmd()) // nothing of the weather's is asked for
	case modeRadar, modeForecast:
	}
	d, temp := d.askTemp()
	return d, tea.Batch(feed, radar, temp, d.mapWorkCmd())
}

// propagationBadge is the Propagation mode's badge in the radar badge's place.
func (d Dashboard) propagationBadge() string {
	return mapBadge(render.Tint(propagationLabel, render.Tok(render.ModalTitle)), "", "")
}

// propChipWords are the P chip's words: whether the Propagation mode is on.
func (d Dashboard) propChipWords() string {
	switch d.mapMode() {
	case modePropagation:
		return "Propagation On"
	case modeRadar, modeForecast:
	}
	return "Propagation Off"
}

// outsideText is the window's words for a place in no region: no weather map
// is drawn for it, and the key that opens the Propagation mode, as bound,
// which covers the whole world (D-21).
func (d Dashboard) outsideText() string {
	text := d.mapPane.outside + " is outside every region the weather map covers - the contiguous United States, Alaska, Hawaii, Puerto Rico and the Virgin Islands, Guam and the Northern Marianas, and American Samoa - so no weather map is drawn for it."
	if keys := d.mapKeys[actMapProp].Keys; len(keys) > 0 {
		text += " Press " + keys[0] + " for the Propagation mode, which covers the whole world."
	}
	return text
}

// outsideStated reports whether the window states the place is outside every
// region in place of the map: in a weather mode, for a place in no region.
func (d Dashboard) outsideStated() bool {
	if d.mapPane.outside == "" {
		return false
	}
	switch d.mapMode() {
	case modeRadar, modeForecast:
		return true
	case modePropagation:
	}
	return false
}
