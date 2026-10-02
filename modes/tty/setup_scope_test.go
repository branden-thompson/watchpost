package tty

// setup_scope_test.go — D-92: a settings row belongs to a surface.
//
// HUM LEAD, 2026-09-12:
//
//	"Settings that are truly shared should be (units / time / theme / etc).
//	 Settings that are common, but can or need different values depending on the
//	 mode should be able to support that.  Settings that are unique and specific
//	 to their mode should only appear in the settings modal of their mode, and
//	 should not be able to leak into the mode."
//
// D-18 ALREADY RULED EVERY SETTING (`02-analysis/config-field-table.md`, approved
// 2026-09-09): 52 paths, 25 units, S / O / B / SPLIT.  This is that ruling made
// structural, and metric M4 — "settings bleed, target 0" — is what it serves.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestEverySettingsRowIsRuledForItsSurface is the closed set.
//
// THE ZERO VALUE IS UNRULED, so a row added without a decision fails here rather
// than inheriting one.  It still RENDERS at runtime (setupScope.shownOn) — a
// settings row that vanishes silently is worse than one that appears where it
// should not, because the first is invisible and the second is reportable.
func TestEverySettingsRowIsRuledForItsSurface(t *testing.T) {
	table := setupTable()
	for id := setupRowID(0); id < setupRowCount; id++ {
		if table[id].scope == scopeUnruled {
			t.Errorf("row %d is in the Settings window and nobody has ruled which surface it belongs "+
				"to.  D-18 rules every setting S / O / B / SPLIT — give it a scope, and cite the row", id)
		}
	}
}

// setupOn is the Settings window as one surface draws it.
func setupOn(t *testing.T, s Surface) (Dashboard, string) {
	t.Helper()
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	d.surface = s
	return d, setupOffers(d)
}

// setupOffers is everything the window OFFERS on this surface, scroll aside.
//
// THE VIEWPORT IS NOT THE QUESTION (D-115). `View().Content` is the window
// SCROLLED to the focused row, so a SHARED group can scroll off the top and read
// as lost when it is two lines up.
//
// Scope is what this file is about, and scope is answered by which blocks the
// surface draws at all.
func setupOffers(d Dashboard) string {
	// THROUGH `setupBlocks`, WHICH IS THE WINDOW'S OWN ASSEMBLY.
	//
	// A HELPER THAT RE-DERIVES PRODUCTION'S ANSWER CANNOT CHECK IT. Looping the
	// groups here and skipping the empty ones with the same `visibleRowOfGroup`
	// call that production makes would have `TestNoSettingsGroupIsDrawnEmpty`
	// filter with the predicate it then asserts on: tautological, and unable to
	// fail. Deleting the skip from `setupBlocks` would change nothing (mutant
	// mAA2), and the console could draw "ALERTS - EVENTS" and "WATCHPOST RADIO -
	// RELAY REPLAY" over no rows with every gate green.
	//
	// AND EVERY TAB (D-62), chosen from the full tab list and not from the tabs
	// the surface shows: a tab wrongly hidden must show up as missing rows here,
	// not be skipped by a helper that asked production which tabs to read.
	var b strings.Builder
	for _, tab := range setupTabs() {
		on := d
		for id := setupRowID(0); id < setupRowCount; id++ {
			if tabOfGroup(setupTable()[id].group) == tab {
				on.setup.focus = id // the tab's first row, whatever this surface draws
				break
			}
		}
		for _, blk := range on.setupBlocks(on.opts()) { // bounded by the group set (P10-02)
			for _, l := range blk.lines {
				b.WriteString(stripANSITest(l) + "\n")
			}
		}
	}
	return b.String()
}

// TestEverySurfaceOffersEverySetting is D-70 (HUM LEAD, UAT-1, 2026-09-25):
// "settings show now be the same across ALL UI modes": the console and
// Observer offer the same groups and the same rows. Each row writes its own
// scope's values - the scope in the table says whose - so nothing leaks into a
// mode.
func TestEverySurfaceOffersEverySetting(t *testing.T) {
	_, console := setupOn(t, SurfaceBroadcaster)
	_, observer := setupOn(t, SurfaceObserver)
	for _, want := range []string{"DATA", "ALERTS - EVENTS", "WATCHPOST UI", "ALERTS - TONE",
		"WATCHPOST RADIO - CORRESPONDENTS", "WATCHPOST RADIO - RELAY REPLAY", "STATION", "MAP", "MAP - LAYERS", "MAP - DETAIL", "HISTORY",
		"Default location:", "NASA FIRMS key"} {
		if !strings.Contains(console, want) {
			t.Errorf("the console's Settings window does not offer %q", want)
		}
		if !strings.Contains(observer, want) {
			t.Errorf("Observer's Settings window does not offer %q", want)
		}
	}
}

// A HEADING OVER NO ROWS is worse than an absent group: it says a category of
// settings exists here and then shows none of it.
func TestNoSettingsGroupIsDrawnEmpty(t *testing.T) {
	for _, s := range []Surface{SurfaceObserver, SurfaceBroadcaster} {
		d, out := setupOn(t, s)
		for _, g := range setupGroups() {
			title := setupGroupTitle(g)
			_, drawn := visibleRowOfGroup(g, d.shownOnSurface) // on any tab (D-62): the surface's rows, not the open tab's
			if strings.Contains(out, title) && !drawn {
				t.Errorf("surface %v draws the heading %q with no rows under it", s, title)
			}
		}
	}
}

// THE KEYBOARD NEVER LANDS ON A ROW NOBODY CAN SEE — not on open, not on ↓, not
// on tab.  A focus on a hidden row is a window whose keys appear dead.
// THE WINDOW OPENS ON A ROW THIS SURFACE DRAWS. `setupState{}` focuses row zero,
// which is the listener's default LOCATION — hidden on the console. A window whose
// focus starts off screen is a window whose first keystroke appears to do nothing.
func TestTheWindowOpensOnARowThisSurfaceDraws(t *testing.T) {
	d := dash(t).(Dashboard)
	d.surface = SurfaceBroadcaster
	d = d.openSetup()

	if d.modal != modalSetup {
		t.Fatalf("the fixture did not open the window (modal=%v), so this measures nothing", d.modal)
	}
	if !d.rowVisible(d.setup.focus) {
		t.Errorf("the window opened focused on row %d, which this surface does not draw", d.setup.focus)
	}
}

// THE KEYBOARD NEVER LANDS ON A ROW NOBODY CAN SEE — not on ↓, not on ↑, not on
// tab. A focus on a hidden row is a window whose keys appear dead.
//
// EVERY KEY IS PRESSED, NOT CALLED. Walking `stepGroup` directly lets a mutant
// that breaks the CALL SITE survive — the helper can be right and the wiring not,
// which is a distinction only the real key can make. And `openSetup` on an
// already-open window TOGGLES it shut, after which the keys reach no modal and
// the test passes by measuring nothing.
func TestTheKeyboardNeverFocusesAHiddenRow(t *testing.T) {
	d, _ := setupOn(t, SurfaceBroadcaster)
	if d.modal != modalSetup {
		t.Fatalf("the fixture's window is not open (modal=%v), so no key reaches it", d.modal)
	}

	press := func(t *testing.T, m tea.Model, key tea.KeyPressMsg, n int, what string) tea.Model {
		t.Helper()
		for range n {
			m, _ = m.Update(key)
			cur := m.(Dashboard)
			if cur.modal != modalSetup {
				t.Fatalf("%s closed the window; the walk is no longer measuring focus", what)
			}
			if !cur.rowVisible(cur.setup.focus) {
				t.Fatalf("%s landed on row %d, which this surface does not draw", what, cur.setup.focus)
			}
		}
		return m
	}

	var m tea.Model = d
	rows, groups := 2*int(setupRowCount)+2, 2*len(setupGroups())+2
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyDown}, rows, "down")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyUp}, rows, "up")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyTab}, groups, "tab")
	_ = press(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, groups, "shift+tab")
}
