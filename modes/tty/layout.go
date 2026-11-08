package tty

// layout.go — the frame's geometry, computed once per View and once per key
// event (quality pass Q3, plan §2.5, L5-F6): compact mode, the player rows,
// module heights, the control row and the RECENT window. Each recomputation of
// these facts would render the radio module and the control row again to count
// their lines.

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
	radioRows   []string // the player rows for the mode in force — the only player rendered, drawn once
	radioH      int      // the radio module's height as drawn
	alertH      int      // the alert module's height as drawn
	controlRow  string   // the watchlist control line (wrapped on narrow terminals)
	controlRows int      // its line count
	window      int      // RECENT rows visible
	// priorityRows is how many favourites the priority table shows: all of
	// them while the frame has room, fewer on a short terminal, which then
	// scrolls with the focus (0.19.5 D-30).
	priorityRows int
	days         int // EXTENDED columns shared by both tables
	chrome       int // every fixed line around the RECENT window
}

// layout resolves the frame's geometry from the model. The compact
// decision needs the FULL player's height (UAT 49: the full modules stay
// while the table can show tableBreakpoint rows); the player is then
// rendered a second time only when the mode in force is the two-row one.
func (d Dashboard) layout() frameLayout {
	o := d.opts()
	// What the geometry measures is built ONCE per frame and handed to both
	// resolutions: the player's plan, which gives both
	// players' heights, and the control row. Only the player drawn is
	// rendered (0.19.5 PF-3).
	plan, drawn := d.planRadio(o), o
	rows := layoutRows{radio: plan, viz: d.radioViz, controlRow: d.controlRow(o)}
	fl := d.layoutWith(o, rows)
	// The bands are the last thing to give (UAT 2026-08-27): only when the
	// compact layout still cannot hold the table's floor do they fall back
	// to one row, and the geometry is resolved again on that basis.
	if fl.compact && d.height-fl.chrome-bottomInset < recentWindow {
		o.ThinBands = true
		fl = d.layoutWith(o, rows)
	}
	fl.radioRows = d.radioRender(drawn, plan, fl.compact)
	return d.fitPriority(fl)
}

// minPriorityRows is the fewest favourites a short terminal shows (0.19.5 D-30).
const minPriorityRows = 3

// fitPriority is the last thing to give on a short terminal, after the
// compact player and the thin bands (0.19.5 D-30): when even one RECENT row
// would push the frame past the terminal's bottom, the priority table shows
// fewer favourites, never fewer than minPriorityRows, and scrolls with the
// focus. The frame then never exceeds the terminal down to 80x24.
func (d Dashboard) fitPriority(fl frameLayout) frameLayout {
	over := fl.chrome + 1 - d.height
	if over <= 0 || d.numPriority() == 0 || fl.priorityRows <= minPriorityRows {
		return fl
	}
	shown := max(minPriorityRows, fl.priorityRows-over)
	fl.chrome -= fl.priorityRows - shown
	fl.priorityRows = shown
	fl.window = fitWindow(fl.chrome, d.height)
	return fl
}

// priorityHeight is the rows the priority table's body takes: one a
// favourite, or the empty state's block when there are none, which the frame
// has to count as surely as rows.
func (d Dashboard) priorityHeight(o render.Opts, days int) int {
	if n := d.numPriority(); n > 0 {
		return n
	}
	return strings.Count(emptyState(o.TableRowLen(days), watchlistEmpty, ""), "\n") + 1
}

// fitWindow is how many RECENT rows fit under chrome fixed rows: as many as
// the terminal holds above the bottom inset, at least recentWindow, and on a
// terminal too short for that, the inset gives way first, then the window.
func fitWindow(chrome, height int) int {
	window := max(recentWindow, height-chrome-bottomInset)
	if chrome+window > height {
		window = max(1, height-chrome)
	}
	return window
}

// layoutRows are what a layout measures: the player's plan, with the
// visualiser on or off, and the rendered control row (width-bound; the bands'
// height is not theirs).
type layoutRows struct {
	radio      radioPlan
	viz        bool
	controlRow string
}

// layoutWith resolves the geometry for a given set of options.
func (d Dashboard) layoutWith(o render.Opts, built layoutRows) frameLayout {
	fl := frameLayout{o: o, controlRow: built.controlRow, days: d.sharedExtDays()}
	fl.controlRows = strings.Count(fl.controlRow, "\n") + 1
	fullRadioH := render.BoxHeight(built.radio.rows(false, built.viz)) // the player is a box: its two rules whatever the tone
	fullAlertH := render.BoxHeight(1)                                  // the alert module is a one-row box (facelift 2026-08-28)
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
	fl.radioH = render.BoxHeight(built.radio.rows(fl.compact, built.viz)) // a short frame takes the two-row player
	// UAT 46.1/58: the window expands to fill tall terminals; the content
	// ends exactly bottomInset rows above the bottom.
	fl.priorityRows = d.numPriority()
	fl.chrome = chromeLines + bandExtra + (fl.controlRows - 1) + 1 + fl.radioH + 1 + fl.alertH + 1 + d.priorityHeight(o, fl.days)
	fl.window = fitWindow(fl.chrome, d.height)
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
	topInset    = 2 // the frame's top padding (UAT 10.3), which bottomInset mirrors
)

// compact reports whether the terminal is too short for the full modules
// (UAT 34/47) — the layout's decision, exposed for the tests and the
// modal paths that ask it alone.
func (d Dashboard) compact() bool { return d.layout().compact }

// windowSize is the height-aware recent viewport (UAT 8.2); the Opts
// argument is the caller's own resolution of the same width.
func (d Dashboard) windowSize(render.Opts) int { return d.layout().window }
