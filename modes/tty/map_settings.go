package tty

// map_settings.go — the map's two Settings (0.18.0 W1.8, W1.10, FR-9.1) and
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
func (d Dashboard) cycleMapDesc(forward bool) Dashboard {
	step := 1
	if !forward {
		step = 2
	}
	d.mapDesc = (d.mapDesc + mapDescMode(step)) % 3
	return d.uiTouched()
}

// Label is the picker's words.
func (m mapDescMode) Label() string {
	switch m {
	case mapDescInstead:
		return "Instead of the map"
	case mapDescOff:
		return "Off"
	}
	return "With the map"
}

func mapsKey(off bool) string {
	if off {
		return "off"
	}
	return "on"
}

// The Maps tab's two columns line up (UAT-1 U1-24): every label padded to
// the widest, every picker's value to the longest, so the arrows stand in two
// columns as the other tabs' do. THE WORDS ARE CUT TO FIT (U1-36): the two
// columns sit side by side inside 80% of a 133-column terminal, so a label is
// at most "Description -" and a value at most "Instead of the map".
var (
	mapLabelW = len("Description -")
	mapValueW = len("Instead of the map")
)

// mapDetailValueW is the second column's value width: its longest value,
// "Essential" or "Disabled", so its arrows line up among themselves (U1-24)
// and the column stays narrow enough to sit beside the first (U1-25).
var mapDetailValueW = len("Essential")

// mapRow is one row of the Maps tab, its label padded to the tab's width.
func (d Dashboard) mapRow(o render.Opts, lines []string, at int, id setupRowID, label, cell string) ([]string, int) {
	return d.mapRowW(o, lines, at, id, label, cell, mapLabelW)
}

// mapRowW is mapRow with its group's label width: a group's labels pad to its
// widest, so its pickers line up (HUM LEAD, 2026-09-30).
func (d Dashboard) mapRowW(o render.Opts, lines []string, at int, id setupRowID, label, cell string, labelW int) ([]string, int) {
	focus := d.setup.focus
	if focus == id {
		at = len(lines)
	}
	return append(lines, "  "+setupMark(o, focus == id)+settingLabel(render.PadTo(label, labelW), focus == id)+"  "+cell), at
}

// mapPicker is a picker's cell at the tab's value width.
func (d Dashboard) mapPicker(o render.Opts, id setupRowID, value string) string {
	return d.mapPickerW(o, id, value, mapValueW)
}

// mapPickerW is a picker's cell at a column's own value width: each column
// lines its arrows up within itself, and the second is no wider than its
// longest value, or the two columns stop fitting side by side (U1-25).
func (d Dashboard) mapPickerW(o render.Opts, id setupRowID, value string, w int) string {
	return pickerCellW(value, newArrowChips(o), d.pickerFlashFor(id), w)
}

// mapSettingLines are the MAP group's rows (UAT-1 U1-25: the first of the
// Maps tab's two columns): maps on or off and what the map sends, the
// description's mode, the scale it opens at, the nearby distance, the alerts.
func (d Dashboard) mapSettingLines(o render.Opts, lines []string, at int) ([]string, int) {
	if !d.rowVisible(rowMapsOn) {
		return lines, at
	}
	state := "Enabled"
	if d.mapsOff {
		state = "Disabled"
	}
	lines, at = d.mapRow(o, lines, at, rowMapsOn, "Maps -", d.mapPicker(o, rowMapsOn, state)) // what it contacts is the Status window's (D-75)
	lines, at = d.mapRow(o, lines, at, rowMapDesc, "Description -", d.mapPicker(o, rowMapDesc, d.mapDesc.Label()))
	lines, at = d.mapRow(o, lines, at, rowMapScale, "Opens at -", d.mapPicker(o, rowMapScale, d.mapScale.Label()))
	// NO ALERTS SCOPE (D-76): the map draws every alert in view, and what it
	// draws is switched at the map, in the Overlays menu.
	lines, at = d.mapRow(o, lines, at, rowMapNearby, "Nearby -", d.mapPicker(o, rowMapNearby, d.nearbyLabel()))
	lines, at = d.mapRow(o, lines, at, rowMapRadarSource, "Radar -", d.mapPicker(o, rowMapRadarSource, d.radarSourceLabel()))                 // D-83
	lines, at = d.mapRow(o, lines, at, rowMapRadarAhead, "Radar ahead -", d.mapPicker(o, rowMapRadarAhead, radarAheadLabel(d.mapRadarAhead))) // D-114
	lines, at = d.mapRow(o, lines, at, rowMapQuakes, "Quakes -", d.mapPicker(o, rowMapQuakes, quakeFeedLabel(d.mapQuakeFeed)))                // D-122
	lines, at = d.mapRow(o, lines, at, rowMapTempSource, "Temperature -", d.mapPicker(o, rowMapTempSource, d.tempSourceLabel()))              // D-93, D-190
	lines, at = d.mapRow(o, lines, at, rowMapRainDetail, "Rain Day 4+ -", d.mapPicker(o, rowMapRainDetail, rainDetailLabel(d.mapRainFull)))   // D-192
	lines, at = d.mapRow(o, lines, at, rowMapUVCities, "UV cities -", d.mapPicker(o, rowMapUVCities, strconv.Itoa(d.mapUVCities)))            // D-202; its cost is a notice while it is chosen (D-23, D-237)
	return lines, at
}

// layerLines are the MAP - LAYERS group's rows: a picker each, Enabled or
// Disabled, its label padded to the group's longest so the pickers line up -
// ↑↓ walk them, ←→ and space switch the one the cursor is on (HUM LEAD,
// 2026-09-30) - and what they would cost under them. The focused line is the
// layer under the cursor.
func (d Dashboard) layerLines(o render.Opts) ([]string, int) {
	if !d.rowVisible(rowMapLayers) {
		return nil, 0
	}
	focused := d.setup.focus == rowMapLayers
	labelW := 0
	for _, l := range d.cfg.MapLayers { // the registry's (P10-02)
		labelW = max(labelW, render.Width(l.Label+" -"))
	}
	var lines []string
	at := 0
	for i, l := range d.cfg.MapLayers {
		here := focused && i == min(d.setup.layerAt, len(d.cfg.MapLayers)-1)
		if here {
			at = len(lines)
		}
		state := "Disabled"
		if d.layerOn(l.Key) {
			state = "Enabled"
		}
		var flash pickerFlash
		if here {
			flash = d.pickerFlashFor(rowMapLayers)
		}
		lines = append(lines, "  "+setupMark(o, here)+settingLabel(render.PadTo(l.Label+" -", labelW), here)+"  "+pickerCellW(state, newArrowChips(o), flash, mapDetailValueW))
	}
	return lines, at
}

// mapLayerLines are the MAP - DETAIL group's rows: the map's own detail as a
// list, and the action that empties the map's data.
func (d Dashboard) mapLayerLines(o render.Opts, lines []string, at int) ([]string, int) {
	if !d.rowVisible(rowMapDetailLevel) {
		return lines, at
	}
	lines, at = d.mapRow(o, lines, at, rowMapDetailLevel, "Detail -", d.mapPickerW(o, rowMapDetailLevel, d.detailLevelShown(), mapDetailValueW))
	for i, l := range mapDetailLayers() { // A ROW EACH, "← Enabled →" (U1-35)
		state := "Disabled"
		if d.detailOn(l.key) {
			state = "Enabled"
		}
		lines, at = d.mapRow(o, lines, at, rowMapDetailBorders+setupRowID(i), l.label+" -",
			d.mapPickerW(o, rowMapDetailBorders+setupRowID(i), state, mapDetailValueW)+settingSupport(strings.TrimPrefix(detailShows(l), " ")))
	}
	return d.mapExtraLines(o, lines, at)
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
func (d Dashboard) clearMapData() (tea.Model, tea.Cmd) {
	d.mapPane.clearErr = nil
	if m := d.mapPane.m; m != nil {
		var off, purged error
		d.mapPane.call("CacheRoot:off", func() { off = m.CacheRoot("", 0) })
		d.mapPane.call("Purge", func() { _, purged = m.Purge() })
		if kind, ok := tuimaps.KindOf(purged); ok && kind == tuimaps.CacheRefused {
			purged = nil // the disk was let go first: "no disk to purge" is the memory emptied alone
		}
		d.mapPane.clearErr, d.mapPane.diskOff = errors.Join(off, purged), off == nil
	}
	d.setup.note, d.setup.noteRow, d.setup.noteTone = "Clearing map data…", rowMapClear, noticeInfo
	clear := d.cfg.ClearMapData
	if clear == nil {
		return d.diskBack().settled(), nil
	}
	return d.settled(), func() tea.Msg { return mapClearedMsg{r: clear()} }
}

// diskBack names the live map's disk cache again after a clear let it go.
func (d Dashboard) diskBack() Dashboard {
	m, disk := d.mapPane.m, d.cfg.MapDisk
	if !d.mapPane.diskOff || m == nil || disk == nil {
		return d
	}
	var err error
	d.mapPane.call("CacheRoot:on", func() { err = disk(m) })
	d.mapPane.diskOff, d.mapPane.clearErr = false, errors.Join(d.mapPane.clearErr, err)
	return d
}

// applyMapCleared says, beside the row, what went. What could not be
// cleared is a count and the way to the diagnostics, which are told each
// error (IS-M8, D-124): an OS error names paths in the listener's home.
func (d Dashboard) applyMapCleared(v mapClearedMsg) Dashboard {
	d = d.diskBack()
	failed := leafErrors(errors.Join(v.r.Err, d.mapPane.clearErr))
	d.mapPane.clearErr = nil
	note, tone := mapClearedNote(v.r), noticeDone
	if partly, ok := d.partlyCleared("Clear map data", failed); ok {
		note, tone = partly, noticeFail
	}
	d.setup.note, d.setup.noteRow, d.setup.noteTone = note, rowMapClear, tone
	return d.settled()
}

// partlyCleared is what a clear that left some behind says: a count and the
// way to the diagnostics, which are told each error (IS-M8, D-124) - an OS
// error names paths in the listener's home. ok is false when nothing failed.
func (d Dashboard) partlyCleared(what string, failed []error) (string, bool) {
	if len(failed) == 0 {
		return "", false
	}
	for _, err := range failed { // each to the diagnostics (P10-02)
		d.problem(what + ": " + err.Error())
	}
	key := "ctrl+d"
	if keys := d.keys["debug"].Keys; len(keys) > 0 {
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
func (d Dashboard) mapExtraLines(o render.Opts, lines []string, at int) ([]string, int) {
	focus := d.setup.focus
	if focus == rowMapClear {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapClear)+settingLabel(render.PadTo("Map data -", mapLabelW), focus == rowMapClear)+"  "+o.KeyCap("space")+" clear now")
	return lines, at
}
