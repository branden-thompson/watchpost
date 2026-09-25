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

// mapSettingLines are the WATCHPOST UI group's map rows, drawn on the
// surfaces that show them (the Observer's).
func (d Dashboard) mapSettingLines(o render.Opts, lines []string, at int) ([]string, int) {
	if !d.rowVisible(rowMapsOn) {
		return lines, at
	}
	focus := d.setup.focus
	state := "Enabled"
	if d.mapsOff {
		state = "Disabled"
	}
	if focus == rowMapsOn {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapsOn)+settingLabel("Maps -", focus == rowMapsOn)+"  "+toggleCell(state, newArrowChips(o), d.pickerFlashFor(rowMapsOn)))
	for _, l := range render.WrapText(d.cfg.MapDisclosure, 56) { // FR-9.4: beside the maps row
		lines = append(lines, "    "+l)
	}
	if focus == rowMapDesc {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapDesc)+settingLabel("Map description -", focus == rowMapDesc)+"  "+
		pickerCellW(d.mapDesc.Label(), newArrowChips(o), d.pickerFlashFor(rowMapDesc), len("Instead of the picture")))
	lines, at = d.mapExtraLines(o, lines, at)
	return lines, at
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
	lines = append(lines, "  "+setupMark(o, focus == rowMapClear)+settingLabel("Clear map data -", focus == rowMapClear)+"  "+o.KeyCap("space")+" clear now")
	for _, l := range render.WrapText(d.cfg.MapRetention, 56) {
		lines = append(lines, "    "+l)
	}
	return lines, at
}
