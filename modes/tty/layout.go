package tty

// layout.go — the frame's geometry, computed once per View and once per key
// event (quality pass Q3, plan §2.5, L5-F6): compact mode, the player rows,
// module heights, the control row and the RECENT window. Before Q3 the
// same facts were recomputed about eight times a frame, each time rendering
// the radio module and the control row again to count their lines.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// frameLayout is what one frame's geometry resolves to. The field values
// (not the row strings) form part of the body memo's key: a change in any
// of them changes the table bytes (R2-4).
type frameLayout struct {
	o           render.Opts
	compact     bool     // the short-terminal mode (UAT 34/47)
	radioRows   []string // the player rows for the mode in force — rendered once, drawn once
	radioH      int      // the radio module's height as drawn
	alertH      int      // the alert module's height as drawn
	controlRow  string   // the watchlist control line (wrapped on narrow terminals)
	controlRows int      // its line count
	window      int      // RECENT rows visible
	days        int      // EXTENDED columns shared by both tables
	chrome      int      // every fixed line around the RECENT window
}

// layout resolves the frame's geometry from the model. The compact
// decision needs the FULL player's height (UAT 49: the full modules stay
// while the table can show tableBreakpoint rows); the player is then
// rendered a second time only when the mode in force is the two-row one.
func (d Dashboard) layout() frameLayout {
	o := d.opts()
	// The rows the geometry measures are built ONCE per frame and handed to
	// both resolutions (round 4, B-06: they were built four times a tick).
	rows := layoutRows{full: d.radioLines(o, false), compact: d.radioLines(o, true), controlRow: d.controlRow(o)}
	fl := d.layoutWith(o, rows)
	// The bands are the last thing to give (UAT 2026-08-27): only when the
	// compact layout still cannot hold the table's floor do they fall back
	// to one row, and the geometry is resolved again on that basis.
	if fl.compact && d.height-fl.chrome-bottomInset < recentWindow {
		o.ThinBands = true
		fl = d.layoutWith(o, rows)
	}
	return fl
}

// layoutRows are the rendered rows a layout measures: the player in both
// modes and the control row (width-bound; the bands' height is not theirs).
type layoutRows struct {
	full, compact []string
	controlRow    string
}

// layoutWith resolves the geometry for a given set of options.
func (d Dashboard) layoutWith(o render.Opts, built layoutRows) frameLayout {
	fl := frameLayout{o: o, controlRow: built.controlRow, days: d.sharedExtDays()}
	fl.controlRows = strings.Count(fl.controlRow, "\n") + 1
	full := built.full
	fullRadioH := render.BoxHeight(len(full)) // the player is a box: its two rules whatever the tone
	fullAlertH := render.BoxHeight(1)         // the alert module is a one-row box (facelift 2026-08-28)
	// UAT 49: the full modules stay while the table can show at least
	// tableBreakpoint rows (favourites + recent window, any split); only
	// when the full layout cannot deliver the breakpoint do they minimize.
	rows := max(recentWindow, min(d.numRecent(), tableBreakpoint-d.numPriority()))
	bandExtra := 2 * (o.BandHeight() - 1) // the two bands' rows beyond the one chromeLines counts each
	need := chromeLines + bandExtra + (fl.controlRows - 1) + 1 + fullRadioH + 1 + fullAlertH + 1 + d.numPriority() + rows + bottomInset
	fl.compact = d.height < need
	fl.alertH = fullAlertH
	if fl.compact {
		fl.alertH = 1 // the same row without the rules when compact (UAT 34)
	}
	fl.radioRows = full
	if fl.compact || d.radioMin {
		fl.radioRows = built.compact // [T] Size: Min or compact = the two-row player
	}
	fl.radioH = render.BoxHeight(len(fl.radioRows))
	// UAT 46.1/58: the window expands to fill tall terminals; the content
	// ends exactly bottomInset rows above the bottom.
	fl.chrome = chromeLines + bandExtra + (fl.controlRows - 1) + 1 + fl.radioH + 1 + fl.alertH + 1 + d.numPriority()
	fl.window = max(recentWindow, d.height-fl.chrome-bottomInset)
	if fl.chrome+fl.window > d.height { // the floor: the inset goes first, then the window shrinks to what fits — the frame never exceeds the terminal (round 4, B-07)
		fl.window = max(1, d.height-fl.chrome)
	}
	return fl
}

// chromeLines are the fixed lines around the modules (UAT 19.1): top pad 2,
// the header box 3, blank, the alert→controls gap 1 (the radio and alert boxes
// touch — HUM LEAD UAT 2026-08-28), priority headers 2, and the 0.12.0
// global ticker row + its blank (2) — the band/showing rows are counted
// with the window. bottomInset mirrors the top padding.
const (
	chromeLines = 11
	bottomInset = 2
)

// compact reports whether the terminal is too short for the full modules
// (UAT 34/47) — the layout's decision, exposed for the tests and the
// modal paths that ask it alone.
func (d Dashboard) compact() bool { return d.layout().compact }

// windowSize is the height-aware recent viewport (UAT 8.2); the Opts
// argument is the caller's own resolution of the same width (kept so the
// render sites read as before Q3).
func (d Dashboard) windowSize(render.Opts) int { return d.layout().window }
