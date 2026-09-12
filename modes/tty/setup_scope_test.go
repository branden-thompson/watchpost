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
	return d, stripANSITest(d.View().Content)
}

// THE LISTENER'S SETTINGS DO NOT REACH THE OPERATOR OF A STATION, and the
// SHARED ones still do — the metric counts failures in BOTH directions (M4).
func TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt(t *testing.T) {
	_, console := setupOn(t, SurfaceBroadcaster)
	_, observer := setupOn(t, SurfaceObserver)

	// D-18 row 1 (the listener's default location) and the monitor's own
	// rotation pacing, which cannot even advance while the console holds the air.
	for _, gone := range []string{"ALERTS - EVENTS", "WATCHPOST RADIO - RELAY REPLAY"} {
		if strings.Contains(console, gone) {
			t.Errorf("the console's Settings window still offers %q", gone)
		}
		if !strings.Contains(observer, gone) {
			t.Errorf("%q vanished from OBSERVER too: the scope hid a row from the surface that owns it", gone)
		}
	}
	// D-18 rows 3, 19, 21, 22, 6, 7 — shared, and shared means BOTH.
	for _, kept := range []string{"DATA", "WATCHPOST UI", "ALERTS - TONE", "WATCHPOST RADIO - CORRESPONDENTS"} {
		if !strings.Contains(console, kept) {
			t.Errorf("the console's Settings window lost %q, which D-18 rules SHARED", kept)
		}
	}
	// DATA IS THE ONE MIXED GROUP: the provider key is shared, the default
	// location is not, so the group is half-drawn rather than skipped.
	//
	// "Default location:" WITH A LOWER-CASE L IS THE ROW. "Default Location" is
	// the EVENTS row's own wording ("Within [ ] mi of Default Location"), and
	// asserting that instead let a mutant through: the case is the only thing
	// telling the two apart.
	for _, gone := range []string{"Default location:", `City Name, "City, ST", or Zip`} {
		if strings.Contains(console, gone) {
			t.Errorf("the console still draws the LISTENER's default location row: %q", gone)
		}
		if !strings.Contains(observer, gone) {
			t.Errorf("%q vanished from OBSERVER too", gone)
		}
	}
	if !strings.Contains(console, "NASA FIRMS key") {
		t.Errorf("the console lost the provider key, which D-18 row 3 rules SHARED — DATA is a MIXED " +
			"group and must be half-drawn, not skipped")
	}
}

// A HEADING OVER NO ROWS is worse than an absent group: it says a category of
// settings exists here and then shows none of it.
func TestNoSettingsGroupIsDrawnEmpty(t *testing.T) {
	for _, s := range []Surface{SurfaceObserver, SurfaceBroadcaster} {
		d, out := setupOn(t, s)
		for _, g := range setupGroups() {
			title := setupGroupTitle(g)
			_, drawn := visibleRowOfGroup(g, d.rowVisible)
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
// EVERY KEY IS PRESSED, NOT CALLED. The first draft walked `stepGroup` directly
// and a mutant that broke the CALL SITE survived it — the helper was right and the
// wiring was not, which is a distinction only the real key can make. The draft
// also called `openSetup` on an already-open window, which TOGGLES it shut: the
// keys then reached no modal and the test passed by measuring nothing.
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
