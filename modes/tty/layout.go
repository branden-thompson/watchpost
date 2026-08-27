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
}

// layout resolves the frame's geometry from the model. The compact
// decision needs the FULL player's height (UAT 49: the full modules stay
// while the table can show tableBreakpoint rows); the player is then
// rendered a second time only when the mode in force is the two-row one.
func (d Dashboard) layout() frameLayout {
	o := d.opts()
	fl := frameLayout{o: o, controlRow: d.controlRow(o), days: d.sharedExtDays()}
	fl.controlRows = strings.Count(fl.controlRow, "\n") + 1
	_, abg := render.AlertBlockTone("", "minor")
	_, rbg := render.RadioBlockTone()
	full := d.radioLines(o, false)
	fullRadioH := render.ModuleHeight(len(full), rbg)
	fullAlertH := render.ModuleHeight(alertContentLines, abg)
	// UAT 49: the full modules stay while the table can show at least
	// tableBreakpoint rows (favourites + recent window, any split); only
	// when the full layout cannot deliver the breakpoint do they minimize.
	rows := max(recentWindow, min(d.numRecent(), tableBreakpoint-d.numPriority()))
	need := chromeLines + (fl.controlRows - 1) + 1 + fullRadioH + 1 + fullAlertH + 1 + d.numPriority() + rows + bottomInset
	fl.compact = d.height < need
	fl.alertH = fullAlertH
	if fl.compact {
		fl.alertH = render.ModuleHeight(1, abg) // one line when compact (UAT 34)
	}
	fl.radioRows = full
	if fl.compact || d.radioMin {
		fl.radioRows = d.radioLines(o, true) // [T] Size: Min or compact = the two-row player
	}
	fl.radioH = render.ModuleHeight(len(fl.radioRows), rbg)
	// UAT 46.1/58: the window expands to fill tall terminals; the content
	// ends exactly bottomInset rows above the bottom.
	chrome := chromeLines + (fl.controlRows - 1) + 1 + fl.radioH + 1 + fl.alertH + 1 + d.numPriority()
	fl.window = max(recentWindow, d.height-chrome-bottomInset)
	return fl
}

// chromeLines are the fixed lines around the modules (UAT 19.1): top pad 2,
// header 2, blank, module gaps 2, priority headers 2 — the band/showing
// rows are counted with the window. bottomInset mirrors the top padding.
const (
	chromeLines = 9
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
