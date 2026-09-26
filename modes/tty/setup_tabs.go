package tty

// setup_tabs.go — Settings in tabs (0.18.0, D-62).
//
// ONE WINDOW, MANY TABS, as the [w] window is. The tab is not state of its
// own: it is the tab of the focused row, so every path that moves the focus -
// a save that sends the listener back to the location, a deep link that opens
// Settings at the voices - lands on the right tab without being told.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// setupTab is one of Settings' tabs.
type setupTab int

const (
	tabData setupTab = iota // D-71: General split into Data and Watchpost UI
	tabUI
	tabRadio
	tabBroadcaster
	tabMaps
)

// setupTabs is every tab, in the order the tab row draws them.
func setupTabs() []setupTab { return []setupTab{tabData, tabUI, tabRadio, tabBroadcaster, tabMaps} }

// Label is the tab's name on the tab row.
func (t setupTab) Label() string {
	switch t {
	case tabRadio:
		return "Watchpost Radio"
	case tabBroadcaster:
		return "Broadcaster"
	case tabMaps:
		return "Maps"
	case tabUI:
		return "Watchpost UI"
	}
	return "Data"
}

// tabOfGroup is where a group lives (D-62).
func tabOfGroup(g setupGroupID) setupTab {
	switch g {
	case groupTone, groupCast, groupRelay:
		return tabRadio
	case groupStation:
		return tabBroadcaster
	case groupMap, groupMapLayers:
		return tabMaps
	case groupUI:
		return tabUI
	}
	return tabData // DATA and ALERTS - EVENTS: what arrives (D-71)
}

// setupTab is the tab the window shows: the focused row's.
func (d Dashboard) setupTab() setupTab { return tabOfGroup(setupTable()[d.setup.focus].group) }

// onFocusedTab reports whether a row is on the tab shown.
func (d Dashboard) onFocusedTab(id setupRowID) bool {
	return tabOfGroup(setupTable()[id].group) == d.setupTab()
}

// shownOnSurface is a row the surface draws on some tab: every row, on every
// surface (D-70, superseding D-92's visibility - each row still writes only
// what it wrote).
func (d Dashboard) shownOnSurface(id setupRowID) bool {
	return id >= 0 && id < setupRowCount
}

// tabsShown are the tabs this surface has: a tab with no row the surface
// draws is not shown - Observer has no Broadcaster tab, the console no Maps.
func (d Dashboard) tabsShown() []setupTab {
	var out []setupTab
	for _, t := range setupTabs() {
		if _, ok := d.firstRowOfTab(t); ok {
			out = append(out, t)
		}
	}
	return out
}

// firstRowOfTab is a tab's first row on this surface.
func (d Dashboard) firstRowOfTab(t setupTab) (setupRowID, bool) {
	for id := setupRowID(0); id < setupRowCount; id++ { // bounded by the table (P10-02)
		if tabOfGroup(setupTable()[id].group) == t && d.shownOnSurface(id) {
			return id, true
		}
	}
	return rowLocation, false
}

// stepTab is the first row of the next (or previous) tab, wrapping.
func (d Dashboard) stepTab(step int) setupRowID {
	tabs := d.tabsShown()
	if len(tabs) == 0 {
		return d.setup.focus
	}
	at := 0
	for i, t := range tabs {
		if t == d.setupTab() {
			at = i
		}
	}
	next := tabs[((at+step)%len(tabs)+len(tabs))%len(tabs)]
	if id, ok := d.firstRowOfTab(next); ok {
		return id
	}
	return d.setup.focus
}

// rowTakesLeftRight reports whether the focused row operates with the arrows
// (a picker, a toggle), which then keeps them (D-62: "the one that just works").
func (d Dashboard) rowTakesLeftRight() bool {
	row := setupTable()[d.setup.focus]
	if d.setup.focus == rowMapLayers {
		return len(d.cfg.MapLayers) > 1 // one layer has nothing to walk to: the arrows switch tabs
	}
	if d.setup.focus == rowMapDetail {
		return true // seven boxes to walk
	}
	return row.picker || row.kind == rowToggle || row.kind == rowPicker
}

// setupTabRow is the tab row, [w]'s shape: the open tab marked with the
// pointer, in the widest form that fits.
func (d Dashboard) setupTabRow(o render.Opts, width int) string {
	ptr := o.Glyphs().Pointer
	var wide, narrow []string
	for _, t := range d.tabsShown() {
		if t == d.setupTab() {
			wide = append(wide, "[ "+ptr+" "+t.Label()+" ]")
			narrow = append(narrow, "["+ptr+t.Label()+"]")
			continue
		}
		wide = append(wide, "[ "+t.Label()+" ]")
		narrow = append(narrow, "["+t.Label()+"]")
	}
	if row := "  " + strings.Join(wide, "  "); render.Width(row) <= width {
		return row
	}
	return render.TruncateCells("  "+strings.Join(narrow, " "), width)
}
