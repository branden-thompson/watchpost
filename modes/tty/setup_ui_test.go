package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// The WATCHPOST UI group: the theme picker and the two display preferences.

// uiDash opens Settings on a row of the UI group, with a SetUI hook that
// records what the window writes.
func uiDash(t *testing.T, at setupRowID) (Dashboard, *UIPrefs) {
	t.Helper()
	var got UIPrefs
	d := setupGolden(t, 133, 44, false, at)
	d.cfg.SetUI = func(p UIPrefs) error { got = p; return nil }
	return d, &got
}

// ←→ on the theme row APPLY the theme as they move. A list of names tells a
// listener nothing about which one they want, so the answer is to show them.
func TestThemePickerPreviewsLive(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, _ := uiDash(t, rowTheme)
	before := render.ThemeName()
	m, _, handled := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if !handled {
		t.Fatal("→ operates the theme picker")
	}
	if render.ThemeName() == before {
		t.Errorf("→ applies the next theme live, still on %q", before)
	}
	if name := m.(Dashboard).themeName(); name != render.ThemeName() {
		t.Errorf("the picker shows the theme in force: picker %q, applied %q", name, render.ThemeName())
	}
}

// The theme picker names no preview key. ←→ have already previewed it — the
// whole app repainted — so `p Preview` would name a key for something that has
// already happened.
func TestThemeRowOffersNoPreviewChip(t *testing.T) {
	d, _ := uiDash(t, rowTheme)
	chips := strings.Join(d.setupChips(d.opts()), " ")
	if strings.Contains(chips, "Preview") {
		t.Errorf("the theme previews itself; the chip row must not offer p Preview: %q", chips)
	}
	if !strings.Contains(chips, "Theme") {
		t.Errorf("the chip row names what ←→ do here: %q", chips)
	}
}

// The units and the clock are radio sets: space selects the focused option, and
// the effect is immediate for both.
func TestUnitsAndClockSelectLive(t *testing.T) {
	for _, tc := range []struct {
		name  string
		at    setupRowID
		check func(Dashboard) bool
	}{
		{"metric", rowUnitsMetric, func(d Dashboard) bool { return d.units == render.UnitC }},
		{"24hr", rowClock24, func(d Dashboard) bool { return d.clockFmt == render.Clock24 }},
		{"MIL", rowClockMil, func(d Dashboard) bool { return d.clockFmt == render.ClockMil }},
	} {
		d, _ := uiDash(t, tc.at)
		nd := d.setupSpace()
		if !tc.check(nd) {
			t.Errorf("%s: space on the row selects it", tc.name)
		}
		if !nd.setup.uiDirty {
			t.Errorf("%s: the choice is marked for the write on close", tc.name)
		}
	}
}

// Closing WRITES all three together — one hook, one write. Three Loads and
// Saves over the same file in a row is three chances for two of them to
// disagree about what the third wrote.
func TestClosingSettingsWritesTheDisplayPreferences(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, got := uiDash(t, rowClockMil)
	d = d.setupSpace() // MIL
	d.setup.focus = rowUnitsMetric
	d = d.setupSpace() // Metric
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.Clock != "mil" || got.Units != "metric" {
		t.Errorf("esc writes what was chosen, got %+v", *got)
	}
	if got.Theme == "" {
		t.Errorf("the theme is written with them, got %+v", *got)
	}
}

// An untouched group writes NOTHING. Opening Settings to read it must not
// rewrite the file — the same rule the cast group follows.
func TestAnUntouchedUIGroupWritesNothing(t *testing.T) {
	d, _ := uiDash(t, rowTheme)
	writes := 0
	d.cfg.SetUI = func(UIPrefs) error { writes++; return nil }
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if writes != 0 {
		t.Errorf("an untouched window writes nothing, got %d", writes)
	}
}

// The clock reaches the frame. This is the point of the setting: one owner, and
// every site that shows a time reads it.
func TestTheClockSettingReachesTheFrame(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowClock24)
	if got := d.opts().Clock; got != render.Clock12 {
		t.Fatalf("the fixture starts on the 12-hour clock, got %v", got)
	}
	nd := d.setupSpace()
	if got := nd.opts().Clock; got != render.Clock24 {
		t.Errorf("selecting 24hr reaches the render options, got %v", got)
	}
	// The header stamp is the site a listener sees first.
	twelve := stripANSITest(d.header(d.opts()))
	twentyFour := stripANSITest(nd.header(nd.opts()))
	if twelve == twentyFour {
		t.Errorf("the header stamp follows the clock:\n%s\n%s", twelve, twentyFour)
	}
	if strings.Contains(twentyFour, "AM") || strings.Contains(twentyFour, "PM") {
		t.Errorf("the 24-hour stamp carries no meridiem: %s", twentyFour)
	}
}

// Every form round-trips through its config word, and an unknown word reads as
// the default rather than failing a load — a display preference is not worth
// refusing to start over.
func TestClockAndUnitsRoundTripTheirConfigWords(t *testing.T) {
	for _, c := range render.ClockOrder() {
		if got := render.ClockByKey(c.Key()); got != c {
			t.Errorf("clock %v round-trips through %q, got %v", c, c.Key(), got)
		}
	}
	for _, u := range render.UnitsOrder() {
		if got := render.UnitsByKey(u.Key()); got != u {
			t.Errorf("units %v round-trip through %q, got %v", u, u.Key(), got)
		}
	}
	if got := render.ClockByKey("half past"); got != render.Clock12 {
		t.Errorf("an unknown clock word reads as the default, got %v", got)
	}
	if got := render.UnitsByKey("furlongs"); got != render.UnitF {
		t.Errorf("an unknown units word reads as the default, got %v", got)
	}
}

// TestEveryTabIsOneColumn is F-205 AX-7 (D-364): every tab, on both
// surfaces and at every size, lays its groups end to end, so no line holds
// two settings for a line reader to hear together.
func TestEveryTabIsOneColumn(t *testing.T) {
	for _, size := range [][2]int{{133, 44}, {200, 60}, {80, 24}} {
		for _, surface := range []Surface{SurfaceObserver, SurfaceBroadcaster} {
			base := setupGolden(t, size[0], size[1], false, rowCastAlerts)
			base.surface = surface
			for _, tab := range base.tabsShown() {
				d := base
				d.setup.focus, _ = base.firstRowOfTab(tab)
				lines, _, _ := d.setupPage(d.opts())
				for _, l := range lines {
					plain := stripANSITest(l)
					if strings.Count(plain, "←") > 1 || strings.Count(plain, "●")+strings.Count(plain, "○") > 1 {
						t.Errorf("%dx%d surface %v tab %s: a line holds two settings: %q", size[0], size[1], surface, tab.Label(), plain)
					}
				}
			}
		}
	}
}

// TestTheMapLayersTabHoldsDetailAndLayers is D-364: MAP is the Maps tab's,
// MAP - DETAIL and MAP - LAYERS the Map Layers tab's, each one page.
func TestTheMapLayersTabHoldsDetailAndLayers(t *testing.T) {
	if tabOfGroup(groupMap) != tabMaps || tabOfGroup(groupLayers) != tabMapLayers || tabOfGroup(groupMapLayers) != tabMapLayers {
		t.Fatal("the map's groups are not split between Maps and Map Layers")
	}
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	page := func(tab setupTab) string {
		on := d
		on.setup.focus, _ = d.firstRowOfTab(tab)
		lines, _, _ := on.setupPage(on.opts())
		return stripANSITest(strings.Join(lines, "\n"))
	}
	maps, layers := page(tabMaps), page(tabMapLayers)
	if !strings.Contains(maps, "MAP") || strings.Contains(maps, "MAP - DETAIL") || strings.Contains(maps, "MAP - LAYERS") {
		t.Errorf("the Maps tab is not MAP alone:\n%s", maps)
	}
	if !strings.Contains(layers, "MAP - DETAIL") || !strings.Contains(layers, "MAP - LAYERS") {
		t.Errorf("the Map Layers tab lacks DETAIL or LAYERS:\n%s", layers)
	}
}

// TestTheTabRowFitsTheNarrowWindow is A-107 (11): the tab row fits its
// window in one line, short names where the full ones do not fit, every name
// whole and every line within the window.
func TestTheTabRowFitsTheNarrowWindow(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {133, 44}} {
		d := setupGolden(t, size[0], size[1], false, rowCastAlerts)
		o := d.opts()
		rows := d.setupTabRows(o, o.Width-4)
		all := stripANSITest(strings.Join(rows, "\n"))
		if len(rows) != 1 {
			t.Errorf("%dx%d: the tab row takes %d lines", size[0], size[1], len(rows))
		}
		for _, r := range rows {
			if w := render.Width(r); w > o.Width-4 {
				t.Errorf("%dx%d: a tab row is %d wide, over %d: %q", size[0], size[1], w, o.Width-4, stripANSITest(r))
			}
		}
		for _, tab := range d.tabsShown() {
			if !strings.Contains(all, tab.Label()) && !strings.Contains(all, tab.Short()) {
				t.Errorf("%dx%d: the tab row cuts %q:\n%s", size[0], size[1], tab.Label(), all)
			}
		}
	}
}

// The balance has to pay for itself: a window that fits its budget needs no
// scroll rail, and that is what taking a dozen wasted rows out of it buys.
func TestSettingsFitsWithoutScrollingAtTheMockWidth(t *testing.T) {
	// EVERY TAB, on both surfaces (D-62: the one-page rule is one per tab).
	for _, surface := range []Surface{SurfaceObserver, SurfaceBroadcaster} {
		base := setupGolden(t, 133, 44, false, rowCastAlerts)
		base.surface = surface
		for _, tab := range base.tabsShown() {
			d := base
			d.setup.focus, _ = base.firstRowOfTab(tab)
			d.cfg.MapRetention = "Map tiles are kept 7 days; 512 MB in all with the web cache."
			lines, _, _ := d.setupBody(d.opts())
			if len(lines) > d.modalMax() {
				t.Errorf("surface %v, tab %s: fits its %d-line budget without a rail, got %d lines", surface, tab.Label(), d.modalMax(), len(lines))
			}
		}
	}
}

// The scroll must read the layout that is DRAWN. The focused row moves between
// columns as the split moves, and a focus line computed against the other
// column puts the mark off the edge — which reads as a dead keyboard.
func TestSettingsFocusLineFollowsTheBalancedLayout(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{{"two columns", 133, 44}, {"stacked", 80, 24}} {
		for id := setupRowID(0); id < setupRowCount; id++ {
			d := setupGolden(t, size.w, size.h, false, id)
			// THE ROWS THIS SURFACE DRAWS (D-92, D-115). A focus line for a row
			// the surface does not draw is meaningless — and since the station's
			// transmitter and service radius are Broadcaster-only, walking every
			// id on Observer would ask where a mark is for two rows that are not
			// there.
			if !d.rowVisible(id) {
				continue
			}
			lines, at, end := d.setupBody(d.opts())
			if at < 0 || at >= len(lines) || end < at || end > len(lines) {
				t.Fatalf("%s: row %v spans [%d,%d] of %d lines", size.name, id, at, end, len(lines))
			}
			if !strings.Contains(lines[at], "›") {
				t.Errorf("%s: row %v — the focus mark is not on line %d: %q", size.name, id, at, stripANSITest(lines[at]))
			}
		}
	}
}

// A note NEVER RESIZES THE WINDOW.
//
// A note wrapped to a constant wider than the rows it sits under would nudge the
// whole window out whenever a row with a reason is focused. A window that
// changes size when you move the cursor is the layout
// telling you it does not know its own mind.
func TestACorrespondentNoteNeverWidensTheWindow(t *testing.T) {
	base := setupGolden(t, 133, 44, false, rowLocation).setupWidth()
	for _, id := range castRowOrder() {
		d := setupGolden(t, 133, 44, false, id)
		if d.castNote(id) == "" {
			continue
		}
		if got := d.setupWidth(); got != base {
			t.Errorf("focusing %v changed the window from %d to %d cells", id, base, got)
		}
	}
}

// BOTH EXITS WRITE THE SAME GROUPS. enter on a WATCHPOST UI row saves and
// closes, and it must carry the display preferences with it: the window state
// is cleared on the way out, so a group left off this path is a group the
// listener chose and the file never received.
func TestEnterSavingFromAUIRowWritesTheDisplayPreferences(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, saved := uiDash(t, rowClockMil)
	d = d.setupSpace() // MIL
	d.setup.focus = rowUnitsMetric
	d = d.setupSpace() // Metric

	d.setup.focus = rowClockMil
	model, cmd := d.setupAdvance() // enter on a UI row: saves and closes
	drain(t, model, cmd)

	if saved.Clock != "mil" || saved.Units != "metric" {
		t.Errorf("enter must write the display preferences it was shown, got %+v", *saved)
	}
}

// The two exits agree. Whatever one route writes, the other writes too —
// otherwise which key a listener happens to press decides which of their
// answers survive.
func TestTheTwoSettingsExitsWriteTheSameGroups(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	choose := func(t *testing.T, exit func(Dashboard) (tea.Model, tea.Cmd)) UIPrefs {
		t.Helper()
		d, saved := uiDash(t, rowClockMil)
		d = d.setupSpace()
		d.setup.focus = rowUnitsMetric
		d = d.setupSpace()
		d.setup.focus = rowClockMil
		model, cmd := exit(d)
		drain(t, model, cmd)
		return *saved
	}
	viaEsc := choose(t, func(d Dashboard) (tea.Model, tea.Cmd) {
		return d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	})
	viaEnter := choose(t, func(d Dashboard) (tea.Model, tea.Cmd) { return d.setupAdvance() })
	if viaEsc.Clock != viaEnter.Clock || viaEsc.Units != viaEnter.Units {
		t.Errorf("the exits disagree: esc wrote %+v, enter wrote %+v", viaEsc, viaEnter)
	}
}
