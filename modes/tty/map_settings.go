package tty

// map_settings.go — the map's two Settings (0.18.0 W1.8, W1.10, FR-9.1) and
// the one owner of the map's size floor (W1.5, FR-1.4).

import (
	"strconv"

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
	lines = append(lines, "")
	if focus == rowMapsOn {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapsOn)+settingLabel("Maps -", focus == rowMapsOn)+"  "+toggleCell(state, newArrowChips(o), d.pickerFlashFor(rowMapsOn)))
	if focus == rowMapDesc {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowMapDesc)+settingLabel("Map description -", focus == rowMapDesc)+"  "+
		pickerCellW(d.mapDesc.Label(), newArrowChips(o), d.pickerFlashFor(rowMapDesc), len("Instead of the picture")))
	return lines, at
}

// belowFloorText names the size the map needs and the size it has (FR-1.4).
func belowFloorText(has tuimaps.Size) string {
	return "The map needs " + strconv.Itoa(mapMinBody.Cols) + " × " + strconv.Itoa(mapMinBody.Rows) +
		" cells and this window has " + strconv.Itoa(has.Cols) + " × " + strconv.Itoa(has.Rows) + ". What the map would show:"
}
