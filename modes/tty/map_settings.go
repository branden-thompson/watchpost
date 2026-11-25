package tty

// map_settings.go — the map's Settings rows (0.18.0 W1.8, W1.10, FR-9.1) and
// the one owner of the map's size floor (W1.5, FR-1.4).

import (
	"errors"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// mapMinBody is the smallest map the window draws (FR-1.4, D-16). THE ONLY
// PLACE THE NUMBER IS WRITTEN: the decision and the notice's words both read it.
var mapMinBody = tuimaps.Size{Cols: 69, Rows: 12}

// mapsOffText is what g says with maps off in Settings (FR-1.6): where the
// switch is - the Maps tab's MAP group (D-262).
const mapsOffText = "Maps are off. Turn them on in Settings (s), on the Maps tab."

// mapDescMode is where the description goes: with the picture, first in
// reading order (D-55); instead of it; or not at all.
type mapDescMode int

const (
	mapDescWith mapDescMode = iota // the default: words for everyone, the picture below them
	mapDescInstead
	mapDescOff
)

// Key is the word the file keeps.
func (m mapDescMode) Key() string {
	switch m {
	case mapDescInstead:
		return "instead"
	case mapDescOff:
		return "off"
	}
	return "with"
}

// mapDescByKey reads the file's word; anything unrecognised is the default,
// as every display preference is (a preference is not worth refusing to
// start over).
func mapDescByKey(key string) mapDescMode {
	switch key {
	case "instead":
		return mapDescInstead
	case "off":
		return mapDescOff
	}
	return mapDescWith
}

// cycleMapDesc moves the description's picker: with the picture, instead of
// it, off, and round again. Live, and written with the group on close.
func (mm mapModel) cycleMapDesc(forward bool) (mapModel, mapWant) {
	step := 1
	if !forward {
		step = 2
	}
	mm.mapDesc = (mm.mapDesc + mapDescMode(step)) % 3
	return mm.wantTouched()
}

// Label is the picker's words.
func (m mapDescMode) Label() string {
	switch m {
	case mapDescInstead:
		return "Instead of the map"
	case mapDescOff:
		return "Off"
	}
	return "With the map, on A" // the box waits for A (D-87, D-365)
}

func mapsKey(off bool) string {
	if off {
		return "off"
	}
	return "on"
}

// The MAP group's pickers line up (UAT-1 U1-24): every label padded to the
// widest, "Playback speed -", every picker's value to the longest, a fast
// speed's, so the arrows stand in one column as the other groups' do.
var (
	mapLabelW = len("Playback speed -")
	mapValueW = render.Width(playbackLabel(playbackChoices[len(playbackChoices)-1]))
)

// mapDetailValueW is the second column's value width: its longest value,
// "Essential" or "Disabled", so its arrows line up among themselves (U1-24)
// and the column stays narrow enough to sit beside the first (U1-25).
var mapDetailValueW = len("Essential")

// mapRow is one row of the Maps tab, its label padded to the tab's width.
func (mm mapModel) mapRow(o render.Opts, sv mapSettingsView, lines []string, at int, id setupRowID, label, cell string) ([]string, int) {
	return mm.mapRowW(o, sv, lines, at, id, label, cell, mapLabelW)
}

// mapRowW is mapRow with its group's label width: a group's labels pad to its
// widest, so its pickers line up (HUM LEAD, 2026-09-30).
func (mm mapModel) mapRowW(o render.Opts, sv mapSettingsView, lines []string, at int, id setupRowID, label, cell string, labelW int) ([]string, int) {
	focus := sv.Focus
	if focus == id {
		at = len(lines)
	}
	return append(lines, "  "+setupMark(o, focus == id)+settingLabel(render.PadTo(label, labelW), focus == id)+"  "+cell), at
}

// mapPicker is a picker's cell at the tab's value width.
func (mm mapModel) mapPicker(o render.Opts, sv mapSettingsView, id setupRowID, value string) string {
	return mm.mapPickerW(o, sv, id, value, mapValueW)
}

// mapPickerW is a picker's cell at a column's own value width: each column
// lines its arrows up within itself, and the second is no wider than its
// longest value, or the two columns stop fitting side by side (U1-25).
func (mm mapModel) mapPickerW(o render.Opts, sv mapSettingsView, id setupRowID, value string, w int) string {
	return pickerCellW(value, newArrowChips(o), sv.flashFor(id), w)
}

// mapSettingLines are the MAP group's rows (UAT-1 U1-25: the first of the
// Maps tab's two columns): maps on or off and what the map sends, the
// description's mode, the scale it opens at, the nearby distance, the alerts.
func (mm mapModel) mapSettingLines(o render.Opts, sv mapSettingsView, lines []string, at int, e mapEnv) ([]string, int) {
	if !sv.visible(rowMapsOn) {
		return lines, at
	}
	state := "Enabled"
	if mm.mapsOff {
		state = "Disabled"
	}
	lines, at = mm.mapRow(o, sv, lines, at, rowMapsOn, "Maps -", mm.mapPicker(o, sv, rowMapsOn, state)) // what it contacts is the Status window's (D-75)
	lines, at = mm.mapRow(o, sv, lines, at, rowMapDesc, "Description -", mm.mapPicker(o, sv, rowMapDesc, mm.mapDesc.Label()))
	lines, at = mm.mapRow(o, sv, lines, at, rowMapScale, "Opens at -", mm.mapPicker(o, sv, rowMapScale, mm.mapScale.Label()))
	lines, at = mm.mapRow(o, sv, lines, at, rowMapColours, "Colours -", mm.mapPicker(o, sv, rowMapColours, mapDepthLabel(mm.mapDepth)))             // D-207
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPlayback, "Playback speed -", mm.mapPicker(o, sv, rowMapPlayback, playbackLabel(mm.mapPlayback))) // D-361
	// NO ALERTS SCOPE (D-76): the map draws every alert in view, and what it
	// draws is switched at the map, in the Overlays menu.
	lines, at = mm.mapRow(o, sv, lines, at, rowMapNearby, "Nearby -", mm.mapPicker(o, sv, rowMapNearby, mm.nearbyLabel(e)))
	lines, at = mm.mapRow(o, sv, lines, at, rowMapRadarSource, "Radar -", mm.mapPicker(o, sv, rowMapRadarSource, mm.radarSourceLabel()))                      // D-83
	lines, at = mm.mapRow(o, sv, lines, at, rowMapRadarAhead, "Radar ahead -", mm.mapPicker(o, sv, rowMapRadarAhead, radarAheadLabel(mm.mapRadarAhead)))      // D-114
	lines, at = mm.mapRow(o, sv, lines, at, rowMapQuakes, "Quakes -", mm.mapPicker(o, sv, rowMapQuakes, quakeFeedLabel(mm.mapQuakeFeed)))                     // D-122
	lines, at = mm.mapRow(o, sv, lines, at, rowMapTempSource, "Temperature -", mm.mapPicker(o, sv, rowMapTempSource, mm.tempSourceLabel()))                   // D-93, D-190
	lines, at = mm.mapRow(o, sv, lines, at, rowMapRainDetail, "Rain Day 4+ -", mm.mapPicker(o, sv, rowMapRainDetail, rainDetailLabel(mm.mapRainFull)))        // D-192
	lines, at = mm.mapRow(o, sv, lines, at, rowMapUVCities, "UV cities -", mm.mapPicker(o, sv, rowMapUVCities, strconv.Itoa(mm.mapUVCities)))                 // D-202; its cost is a notice while it is chosen (D-23, D-237)
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPropRadius, "NVIS radius -", mm.mapPicker(o, sv, rowMapPropRadius, mm.propRadiusLabel(e)))                  // D-100: the Propagation mode's near-vertical area
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPropNoise, "RX noise -", mm.mapPicker(o, sv, rowMapPropNoise, propNoiseLabel(mm.mapPropNoise)))             // D-187: the reference circuit's noise
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPropRefresh, "HF refresh -", mm.mapPicker(o, sv, rowMapPropRefresh, mm.propRefreshLabel()))                 // D-120: its cost a notice while it is chosen
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPropReadings, "No readings -", mm.mapPicker(o, sv, rowMapPropReadings, noReadingsLabel(mm.mapPropCorrect))) // D-105: while no station readings are held
	lines, at = mm.mapRow(o, sv, lines, at, rowMapPropStations, "HF stations -", mm.mapPicker(o, sv, rowMapPropStations, stationsLabel(mm.mapPropNoGIRO)))    // D-214: GIRO's terms a notice beside it
	return lines, at
}

// layerLines are the MAP - LAYERS group's rows: a picker each, Enabled or
// Disabled, its label padded to the group's longest so the pickers line up -
// ↑↓ walk them, ←→ and space switch the one the cursor is on (HUM LEAD,
// 2026-09-30) - and what they would cost under them. The focused line is the
// layer under the cursor.
func (mm mapModel) layerLines(o render.Opts, sv mapSettingsView, e mapEnv) ([]string, int) {
	if !sv.visible(rowMapLayers) {
		return nil, 0
	}
	focused := sv.Focus == rowMapLayers
	labelW := 0
	for _, l := range e.Layers { // the registry's (P10-02)
		labelW = max(labelW, render.Width(l.Label+" -"))
	}
	var lines []string
	at := 0
	for i, l := range e.Layers {
		here := focused && i == min(sv.LayerAt, len(e.Layers)-1)
		if here {
			at = len(lines)
		}
		state := "Disabled"
		if mm.layerOn(l.Key, e) {
			state = "Enabled"
		}
		var flash pickerFlash
		if here {
			flash = sv.flashFor(rowMapLayers)
		}
		lines = append(lines, "  "+setupMark(o, here)+settingLabel(render.PadTo(l.Label+" -", labelW), here)+"  "+pickerCellW(state, newArrowChips(o), flash, mapDetailValueW))
	}
	return lines, at
}

// mapLayerLines are the MAP - DETAIL group's rows: the map's own detail as a
// list, and the action that empties the map's data.
func (mm mapModel) mapLayerLines(o render.Opts, sv mapSettingsView, lines []string, at int) ([]string, int) {
	if !sv.visible(rowMapDetailLevel) {
		return lines, at
	}
	lines, at = mm.mapRow(o, sv, lines, at, rowMapDetailLevel, "Detail -", mm.mapPickerW(o, sv, rowMapDetailLevel, mm.detailLevelShown(), mapDetailValueW))
	for i, l := range mapDetailLayers() { // A ROW EACH, "← Enabled →" (U1-35)
		state := "Disabled"
		if mm.detailOn(l.key) {
			state = "Enabled"
		}
		lines, at = mm.mapRow(o, sv, lines, at, rowMapDetailBorders+setupRowID(i), l.label+" -",
			mm.mapPickerW(o, sv, rowMapDetailBorders+setupRowID(i), state, mapDetailValueW)+settingSupport(strings.TrimPrefix(detailShows(l), " ")))
	}
	return mm.mapExtraLines(o, sv, lines, at)
}

// belowFloorText names the size the map needs and the size it has (FR-1.4).
func belowFloorText(has tuimaps.Size) string {
	return "The map needs " + strconv.Itoa(mapMinBody.Cols) + " × " + strconv.Itoa(mapMinBody.Rows) +
		" cells and this window has " + strconv.Itoa(has.Cols) + " × " + strconv.Itoa(has.Rows) + ". What the map would show:"
}

// MapCleared is what "Clear map data" removed: tile files, and zone outlines
// held or cached (0.18.0 W3.8).
type MapCleared struct {
	Files, Zones int
	Err          error
}

// mapClearedMsg is the app's answer to "Clear map data".
type mapClearedMsg struct{ r MapCleared }

// clearMapData empties the window's live map's memory on the UI goroutine
// and asks the app for the rest off it: the tile files on disk, the zone
// outlines held and cached (0.18.0 W3.8, W9.5 folded). THE DISK IS NEVER
// TOUCHED HERE (PF-4): the live map lets its disk cache go first, so its
// Purge empties memory alone, and it names the cache again when the app has
// answered. What the live map refused is kept for the answer (IS-M5).
func (mm mapModel) clearMapData() (mapModel, tea.Cmd, mapWant) {
	mm.mapPane.clearErr = nil
	if m := mm.mapPane.m; m != nil {
		var off, purged error
		mm.mapPane.call("CacheRoot:off", func() { off = m.CacheRoot("", 0) })
		mm.mapPane.call("Purge", func() { _, purged = m.Purge() })
		if kind, ok := tuimaps.KindOf(purged); ok && kind == tuimaps.CacheRefused {
			purged = nil // the disk was let go first: "no disk to purge" is the memory emptied alone
		}
		mm.mapPane.clearErr, mm.mapPane.diskOff = errors.Join(off, purged), off == nil
	}
	w := mapWant{Settings: mapSettingsWant{Note: "Clearing map data…", NoteRow: rowMapClear, NoteTone: noticeInfo, NoteSet: true, Settled: true}}
	clear := mm.mapHooks.ClearMapData
	if clear == nil {
		return mm.diskBack(), nil, w
	}
	return mm, func() tea.Msg { return mapClearedMsg{r: clear()} }, w
}

// diskBack names the live map's disk cache again after a clear let it go.
func (mm mapModel) diskBack() mapModel {
	m, disk := mm.mapPane.m, mm.mapHooks.MapDisk
	if !mm.mapPane.diskOff || m == nil || disk == nil {
		return mm
	}
	var err error
	mm.mapPane.call("CacheRoot:on", func() { err = disk(m) })
	mm.mapPane.diskOff, mm.mapPane.clearErr = false, errors.Join(mm.mapPane.clearErr, err)
	return mm
}

// applyMapCleared says, beside the row, what went. What could not be
// cleared is a count and the way to the diagnostics, which are told each
// error (IS-M8, D-124): an OS error names paths in the listener's home.
func (mm mapModel) applyMapCleared(v mapClearedMsg, e mapEnv) (mapModel, mapWant) {
	mm = mm.diskBack()
	failed := leafErrors(errors.Join(v.r.Err, mm.mapPane.clearErr))
	mm.mapPane.clearErr = nil
	note, tone := mapClearedNote(v.r), noticeDone
	if partly, ok := mm.partlyCleared("Clear map data", failed, e); ok {
		note, tone = partly, noticeFail
	}
	return mm, mapWant{Settings: mapSettingsWant{Note: note, NoteRow: rowMapClear, NoteTone: tone, NoteSet: true, Settled: true}}
}

// partlyCleared is what a clear that left some behind says: a count and the
// way to the diagnostics, which are told each error (IS-M8, D-124) - an OS
// error names paths in the listener's home. ok is false when nothing failed.
func (mm mapModel) partlyCleared(what string, failed []error, e mapEnv) (string, bool) {
	if len(failed) == 0 {
		return "", false
	}
	for _, err := range failed { // each to the diagnostics (P10-02)
		mm.problem(what+": "+err.Error(), e)
	}
	key := "ctrl+d"
	if keys := e.Keys["debug"].Keys; len(keys) > 0 {
		key = keys[0]
	}
	return "Partly cleared: " + strconv.Itoa(len(failed)) + " could not be removed - see Diagnostics (" + key + ").", true
}

// maxLeafSteps bounds leafErrors' walk: a clear joins a few errors a store,
// never a thousand.
const maxLeafSteps = 1024

// leafErrors is every error an error holds, joined ones one by one.
func leafErrors(err error) []error {
	var out []error
	stack := []error{err}
	for steps := 0; len(stack) > 0 && steps < maxLeafSteps; steps++ { // each part once (P10-02)
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		joined, ok := e.(interface{ Unwrap() []error })
		switch {
		case ok:
			parts := joined.Unwrap()
			for i := len(parts) - 1; i >= 0; i-- { // pushed last first: taken in order
				stack = append(stack, parts[i])
			}
		case e != nil:
			out = append(out, e)
		}
	}
	return out
}

// mapExtraLines are the maps rows' words: what the map sends, how long it
// keeps its data, and the action that empties it (FR-9.4, FR-3.9).
func (mm mapModel) mapExtraLines(o render.Opts, sv mapSettingsView, lines []string, at int) ([]string, int) {
	focus := sv.Focus
	if focus == rowMapClear {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapClear)+settingLabel(render.PadTo("Map data -", mapLabelW), focus == rowMapClear)+"  "+o.KeyCap("space")+" clear now")
	return lines, at
}
