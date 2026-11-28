package tty

// observer_map.go — the Dashboard's side of the map seam (0.19.6 design
// MS-1, MS-2): the Env it lends the map, the Want it applies, and the
// adapters the window table and the Router name. Not map_-prefixed: it is the
// root's, and stays at the root when the map moves.

import (
	tea "charm.land/bubbletea/v2"
)

// mapEnv is what the map reads of the Dashboard, for one call.
func (d Dashboard) mapEnv() mapEnv {
	return mapEnv{
		Window:       mapModalOf(d.modal),
		Shown:        d.mapShown(),
		Width:        d.width,
		Height:       d.height,
		Opts:         d.opts(),
		ReduceMotion: d.reduceMotion,
		NoColor:      d.cfg.NoColor,
		Surface:      d.surface,
		Pick:         d.pick(),
		Station:      mapStation{Transmitter: d.cfg.Transmitter, RadiusMi: d.cfg.ServiceRadiusMi},
		Layers:       d.cfg.MapLayers,
		Keys:         d.keys,
		Scroll:       d.modalScroll,
		Now:          d.now,
		Ticks:        d.ticks,
	}
}

// mapWindows is the map's windows as the Dashboard holds them, by the map's
// own: the one table both translations read (0.19.6 design MS-2).
var mapWindows = [...]modal{
	mapModalNone:      modalNone,
	mapModalMap:       modalMap,
	mapModalPropAck:   modalPropAck,
	mapModalPropChart: modalPropChart,
	mapModalPropFind:  modalPropFind,
	mapModalPropFreq:  modalPropFreq,
}

// mapModalOf is the Dashboard's window as the map sees it: one of its own,
// or none.
func mapModalOf(m modal) mapModal {
	for w := mapModalMap; int(w) < len(mapWindows); w++ { // the map's five (P10-02)
		if mapWindows[w] == m {
			return w
		}
	}
	return mapModalNone
}

// modal is the map's window as the Dashboard holds it.
func (w mapModal) modal() modal {
	if int(w) < len(mapWindows) {
		return mapWindows[w]
	}
	return modalNone
}

// mapSettings is what the map's Settings rows read of the Settings window.
func (d Dashboard) mapSettings() mapSettingsView {
	return mapSettingsView{
		Focus: d.setup.focus, LayerAt: d.setup.layerAt, Flash: d.pickerFlashFor(d.setup.focus),
		MapsOnShown: d.rowVisible(rowMapsOn), LayersShown: d.rowVisible(rowMapLayers), DetailLevelShown: d.rowVisible(rowMapDetailLevel),
	}
}

// applyMapWant does what the map asked, and then the map's own step and what
// that step asks (0.19.6 design MS-1's split sites): cmd is the call's
// command, which the save joins first or last.
func (d Dashboard) applyMapWant(w mapWant, cmd tea.Cmd) (Dashboard, tea.Cmd) {
	d, cmd = d.applyMapAsks(w, cmd)
	if w.Then == mapStepNone {
		return d, cmd
	}
	mm, then, more := d.mapModel.continueAfter(w.Then, d.mapEnv())
	d.mapModel = mm
	d, then = d.applyMapAsks(more, then) // a step asks for no step after it
	return d, tea.Batch(cmd, then)
}

// applyMapAsks does what the map asked, in the order the map's code asked
// it: the Settings window's state, the window stack, the selection, the
// scroll, the touch, and the save.
func (d Dashboard) applyMapAsks(w mapWant, cmd tea.Cmd) (Dashboard, tea.Cmd) {
	s := w.Settings
	if s.Dirty {
		d.setup.uiDirty = true
	}
	if s.LayerAtSet {
		d.setup.layerAt = s.LayerAt
	}
	if s.NoteSet {
		d.setup.note, d.setup.noteRow, d.setup.noteTone = s.Note, s.NoteRow, s.NoteTone
	}
	if w.Close {
		d = d.close()
	}
	if w.Open != mapModalNone {
		d = d.open(w.Open.modal())
	}
	if w.Nav != "" {
		d = d.handleNav(w.Nav)
	}
	if w.ModalNav != "" {
		d = d.handleModalNav(w.ModalNav)
	}
	if w.ScrollSet {
		d.modalScroll = w.Scroll
	}
	switch {
	case s.Touched:
		d = d.uiTouched()
	case s.Settled:
		d = d.settled()
	}
	if s.Save {
		save := d.uiApplyCmd()
		d.setup.uiDirty = false
		if s.SaveLast {
			cmd = tea.Batch(cmd, save)
		} else {
			cmd = tea.Batch(save, cmd, s.Led[0], s.Led[1], s.Led[2]) // one batch, the save first, as the map's code batched it
		}
	}
	return d, cmd
}

// mapKey is a key in one of the map's windows, put to the map and its
// asks applied: the window table's adapters below, each naming its window.
func (d Dashboard) mapKey(w mapModal, key tea.KeyPressMsg) (Dashboard, tea.Cmd, bool) {
	mm, cmd, want, ok := d.mapModel.press(w, key, d.mapEnv())
	d.mapModel = mm
	d, cmd = d.applyMapWant(want, cmd)
	return d, cmd, ok
}

// handleMapKey is the open map window's keys, as the window table names
// them (window_keys.go).
func (d Dashboard) handleMapKey(key tea.KeyPressMsg) (Dashboard, tea.Cmd, bool) {
	return d.mapKey(mapModalMap, key)
}

// handlePropAckKey is the Propagation acknowledgement's keys.
func (d Dashboard) handlePropAckKey(key tea.KeyPressMsg) (Dashboard, tea.Cmd) {
	d, cmd, _ := d.mapKey(mapModalPropAck, key)
	return d, cmd
}

// handlePropChartKey is the Bands chart's keys.
func (d Dashboard) handlePropChartKey(key tea.KeyPressMsg) (Dashboard, tea.Cmd) {
	d, cmd, _ := d.mapKey(mapModalPropChart, key)
	return d, cmd
}

// handlePropFindKey is the Propagation finder's keys.
func (d Dashboard) handlePropFindKey(key tea.KeyPressMsg) (Dashboard, tea.Cmd) {
	d, cmd, _ := d.mapKey(mapModalPropFind, key)
	return d, cmd
}

// handlePropFreqKey is the frequency entry's keys.
func (d Dashboard) handlePropFreqKey(key tea.KeyPressMsg) (Dashboard, tea.Cmd) {
	d, cmd, _ := d.mapKey(mapModalPropFreq, key)
	return d, cmd
}

// toggleMap opens the map window, or closes it when it is open (0.19.6
// design MS-1's S1): the window is the Dashboard's to open, and the map is
// built in it.
func (d Dashboard) toggleMap() Dashboard {
	if d.modal == modalMap {
		return d.close()
	}
	d = d.open(modalMap)
	mm, cmd, built := d.mapModel.opened(d.mapEnv())
	d.mapModel = mm
	if !built {
		return d
	}
	return d.withCmd(cmd)
}

// withMap is the Dashboard with the map's answer to a Settings row: its
// model, and what it asked of the Settings window. A row's answer carries no
// command, no save and no step: those are the map window's keys'.
func (d Dashboard) withMap(mm mapModel, w mapWant) Dashboard {
	d.mapModel = mm
	d, _ = d.applyMapAsks(w, nil)
	return d
}
