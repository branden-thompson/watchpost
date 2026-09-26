package tty

// map_settings.go — the map's two Settings (0.18.0 W1.8, W1.10, FR-9.1) and
// the one owner of the map's size floor (W1.5, FR-1.4).

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// mapMinBody is the smallest map the window draws (FR-1.4, D-16). THE ONLY
// PLACE THE NUMBER IS WRITTEN: the decision and the notice's words both read it.
var mapMinBody = tuimaps.Size{Cols: 69, Rows: 12}

// mapsOffText is what g says with maps off in Settings (FR-1.6).
const mapsOffText = "Maps are off. Turn them on in Settings (s), under WATCHPOST UI."

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
		return "Instead of the picture"
	case mapDescOff:
		return "Off"
	}
	return "With the picture"
}

func mapsKey(off bool) string {
	if off {
		return "off"
	}
	return "on"
}

// The Maps tab's two columns line up (UAT-1 U1-24): every label padded to
// the widest, every picker's value to the longest, so the arrows stand in two
// columns as the other tabs' do.
var (
	mapLabelW = len("Map description -")
	mapValueW = len("Plus national severe events")
)

// mapNoteW is how wide a note under a row wraps in its column.
const mapNoteW = 46

// mapRow is one row of the Maps tab, its label padded to the tab's width.
func (d Dashboard) mapRow(o render.Opts, lines []string, at int, id setupRowID, label, cell string) ([]string, int) {
	focus := d.setup.focus
	if focus == id {
		at = len(lines)
	}
	return append(lines, "  "+setupMark(o, focus == id)+settingLabel(render.PadTo(label, mapLabelW), focus == id)+"  "+cell), at
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
	lines, at = d.mapRow(o, lines, at, rowMapsOn, "Maps -", d.mapPicker(o, rowMapsOn, state))
	for _, l := range render.WrapText(d.cfg.MapDisclosure, mapNoteW) { // FR-9.4 as D-69 amends it: said here alone
		lines = append(lines, "    "+settingSupport(l))
	}
	lines, at = d.mapRow(o, lines, at, rowMapDesc, "Map description -", d.mapPicker(o, rowMapDesc, d.mapDesc.Label()))
	lines, at = d.mapRow(o, lines, at, rowMapScale, "Default scale -", d.mapPicker(o, rowMapScale, d.mapScale.Label()))
	lines, at = d.mapRow(o, lines, at, rowMapNearby, "Nearby -", d.mapPicker(o, rowMapNearby, d.nearbyLabel()))
	return d.mapRow(o, lines, at, rowMapScope, "Alerts -", d.mapPicker(o, rowMapScope, d.mapScope.Label()))
}

// mapLayerLines are the MAP - LAYERS AND DETAIL group's rows (the second
// column): the weather layers and what they would cost, the map's own detail
// as a list, and the action that empties the map's data.
func (d Dashboard) mapLayerLines(o render.Opts, lines []string, at int) ([]string, int) {
	if !d.rowVisible(rowMapLayers) {
		return lines, at
	}
	focus := d.setup.focus
	lines, at = d.mapRow(o, lines, at, rowMapLayers, "Layers -", d.layersCell(o, focus == rowMapLayers))
	for _, l := range render.WrapText(costWarning(d.mapCost), mapNoteW) {
		lines = append(lines, "    "+settingSupport(l))
	}
	on := 0
	for _, l := range mapDetailLayers() {
		if d.detailOn(l.key) {
			on++
		}
	}
	lines, at = d.mapRow(o, lines, at, rowMapDetailLevel, "Detail level -", d.mapPickerW(o, rowMapDetailLevel, detailLevelLabel(d.mapDetailLevel), len("Essential")))
	lines, at = d.mapRow(o, lines, at, rowMapDetail, "Map detail -",
		settingSupport(strconv.Itoa(on)+" of "+strconv.Itoa(len(mapDetailLayers()))+" on"))
	for i, l := range mapDetailLayers() { // THE WHOLE LIST: the column has the room (U1-25)
		cursor := focus == rowMapDetail && i == d.setup.detailAt%len(mapDetailLayers())
		lines = append(lines, "      "+setupMark(o, cursor)+checkMark(o, d.detailOn(l.key))+" "+l.label+settingSupport(d.beyondLevel(l)))
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

// clearMapData empties the window's live map through the library - its
// memory and its disk cache - and asks the app for the rest, off the UI
// goroutine: the tile files a map built later would read, the zone outlines
// held and cached (0.18.0 W3.8, W9.5 folded).
func (d Dashboard) clearMapData() (tea.Model, tea.Cmd) {
	if m := d.mapPane.m; m != nil {
		d.mapPane.call("Purge", func() { _, _ = m.Purge() })
	}
	d.setup.note, d.setup.noteRow = "Clearing map data…", rowMapClear
	clear := d.cfg.ClearMapData
	if clear == nil {
		return d.settled(), nil
	}
	return d.settled(), func() tea.Msg { return mapClearedMsg{r: clear()} }
}

// applyMapCleared says, beside the row, what went.
func (d Dashboard) applyMapCleared(v mapClearedMsg) Dashboard {
	note := "Map data cleared: " + strconv.Itoa(v.r.Files) + " tile files and " + strconv.Itoa(v.r.Zones) + " zone outlines removed."
	if v.r.Err != nil {
		note = "Map data partly cleared - " + v.r.Err.Error()
	}
	d.setup.note, d.setup.noteRow = note, rowMapClear
	return d.settled()
}

// mapExtraLines are the maps rows' words: what the map sends, how long it
// keeps its data, and the action that empties it (FR-9.4, FR-3.9).
func (d Dashboard) mapExtraLines(o render.Opts, lines []string, at int) ([]string, int) {
	focus := d.setup.focus
	if focus == rowMapClear {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapClear)+settingLabel(render.PadTo("Clear map data -", mapLabelW), focus == rowMapClear)+"  "+o.KeyCap("space")+" clear now")
	for _, l := range render.WrapText(d.cfg.MapRetention, mapNoteW) {
		lines = append(lines, "    "+settingSupport(l))
	}
	return lines, at
}
