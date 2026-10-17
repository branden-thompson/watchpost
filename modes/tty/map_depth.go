package tty

// map_depth.go — the map's colour depth, a Map Setting (0.19.0 D-207):
// Automatic by default, following the terminal as go-tuiMaps reads it
// (its L-7.2: COLORTERM and TERM), or a depth chosen; NO_COLOR still means
// none whatever is chosen.

import tuimaps "github.com/branden-thompson/go-tuimaps"

// mapDepth is the Setting's choice.
type mapDepth uint8

// The choices, the default first.
const (
	depthAuto mapDepth = iota // the terminal's (go-tuiMaps TerminalDepth)
	depthTruecolor
	depth256
	depth16
	depthNone
)

// mapDepthChoices are the Setting's choices, in its order.
var mapDepthChoices = []mapDepth{depthAuto, depthTruecolor, depth256, depth16, depthNone}

// mapDepthKeys are the file's words for them; the default is written empty.
var mapDepthKeys = map[mapDepth]string{depthAuto: "", depthTruecolor: "truecolor", depth256: "256", depth16: "16", depthNone: "none"}

// mapDepthByKey reads the file's word; anything not offered is Automatic.
func mapDepthByKey(key string) mapDepth {
	for c, k := range mapDepthKeys {
		if k == key {
			return c
		}
	}
	return depthAuto
}

// mapDepthKey is the file's word for a choice.
func mapDepthKey(c mapDepth) string { return mapDepthKeys[c] }

// mapDepthLabel is the choice as the Settings say it.
func mapDepthLabel(c mapDepth) string {
	switch c {
	case depthTruecolor:
		return "Truecolor"
	case depth256:
		return "256"
	case depth16:
		return "16"
	case depthNone:
		return "None"
	}
	return "Automatic"
}

// cycleMapDepth moves the colour-depth picker, and round again; the map
// takes it at its next frame.
func (d Dashboard) cycleMapDepth(forward bool) Dashboard {
	d.mapDepth = cycleIn(mapDepthChoices, itself, d.mapDepth, forward)
	return d.uiTouched()
}

// mapDepthHint is the depth the map is hinted: the choice, or with NO_COLOR
// set or Automatic chosen, go-tuiMaps' own reading of the terminal, which
// honours NO_COLOR (D-207).
func (d Dashboard) mapDepthHint() tuimaps.Depth {
	if d.cfg.NoColor {
		return tuimaps.TerminalDepth
	}
	switch d.mapDepth {
	case depthTruecolor:
		return tuimaps.Truecolor
	case depth256:
		return tuimaps.Colours256
	case depth16:
		return tuimaps.Colours16
	case depthNone:
		return tuimaps.NoColour
	}
	return tuimaps.TerminalDepth
}
