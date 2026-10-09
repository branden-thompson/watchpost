package tty

// window_keys_test.go — F-184's guards: every window declares the keys it
// owns (window_keys.go), and the declarations hold. The windows are derived
// from the modal constants, never listed by hand: a new window without a
// declaration fails here the moment it is added.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// eachWindow runs a check for every window there is.
func eachWindow(t *testing.T, check func(t *testing.T, m modal, w windowKeys)) {
	t.Helper()
	for m := modalNone + 1; m < numModals; m++ {
		w, ok := windowKeysOf(m)
		if !ok {
			t.Errorf("window %d declares no keys: add it to windowKeysOf", m)
			continue
		}
		check(t, m, w)
	}
}

// TestEveryWindowDeclaresItsKeys: a window that takes every key names what
// takes them; one that binds some names them and what takes them.
func TestEveryWindowDeclaresItsKeys(t *testing.T) {
	eachWindow(t, func(t *testing.T, m modal, w windowKeys) {
		switch w.claim {
		case claimAll:
			if w.all == nil {
				t.Errorf("window %d takes every key and names nothing to take them", m)
			}
		case claimBound:
			if w.bound == nil || w.onKey == nil {
				t.Errorf("window %d binds keys and does not say which, or what takes them", m)
			}
		}
	})
}

// TestNoWindowsArrowsReachTheTableUnderIt: a window whose arrows walk nothing
// of its own lets them reach past it to the table underneath. Every window
// takes every key, binds the arrows itself, or walks its own with nav.
func TestNoWindowsArrowsReachTheTableUnderIt(t *testing.T) {
	d := mapDash(t, Config{})
	eachWindow(t, func(t *testing.T, m modal, w windowKeys) {
		switch {
		case w.claim == claimAll, w.nav != nil:
		case w.claim == claimBound:
			keys := w.bound(d)
			up, down := false, false
			for _, b := range keys {
				for _, k := range b.Keys {
					up, down = up || k == "up", down || k == "down"
				}
			}
			if !up || !down {
				t.Errorf("window %d binds keys but not the arrows, and walks nothing of its own", m)
			}
		default:
			t.Errorf("window %d takes none of the keys and walks nothing of its own: its arrows reach the table under it", m)
		}
	})
}

// TestEveryWindowCanBeLeft is D-62's "always an escape key": esc on every
// window shown leaves it.
func TestEveryWindowCanBeLeft(t *testing.T) {
	eachWindow(t, func(t *testing.T, m modal, w windowKeys) {
		d := mapDash(t, Config{})
		if m == modalMap {
			d, _ = pressKey(d, "g")
		} else {
			d = d.open(m)
		}
		if d.modal != m {
			t.Fatalf("window %d did not open", m)
		}
		if d, _ = pressKey(d, "esc"); d.modal == m {
			t.Errorf("esc on window %d left it shown: a window with no way out", m)
		}
	})
}

// TestNoWindowBindsAKeyTwice: within a window's own keys, one key does one
// thing.
func TestNoWindowBindsAKeyTwice(t *testing.T) {
	d := mapDash(t, Config{})
	eachWindow(t, func(t *testing.T, m modal, w windowKeys) {
		if w.bound == nil {
			return
		}
		seen := map[string]string{}
		for act, b := range w.bound(d) {
			for _, k := range b.Keys {
				if other, dup := seen[k]; dup {
					t.Errorf("window %d binds %q to both %s and %s", m, k, other, act)
				}
				seen[k] = string(act)
			}
		}
	})
}

// TestHelpListsEachWindowsOwnGroup: Help's groups for a window's own keys
// come from its declaration - the map's among them, a row an idea.
func TestHelpListsEachWindowsOwnGroup(t *testing.T) {
	groups := helpGroups(SurfaceObserver)
	eachWindow(t, func(t *testing.T, m modal, w windowKeys) {
		if w.helpTitle == "" {
			return
		}
		found := false
		for _, g := range groups {
			found = found || (g.name == w.helpTitle && g.rows != nil)
		}
		if !found {
			t.Errorf("window %d declares Help's %s group and Help does not list it", m, w.helpTitle)
		}
	})
	help := stripANSITest(strings.Join(helpTextLines(t), "\n"))
	if !strings.Contains(help, "Radar / Propagation / Forecast Hi-Lo") {
		t.Error("Help lost the map's rows")
	}
}

// helpTextLines is Help as drawn on Observer.
func helpTextLines(t *testing.T) []string {
	t.Helper()
	d := goldenDash(t, false)
	return d.helpLines(d.opts())
}

// TestTheWindowShownTakesItsKeysFirst: a key the window shown owns is its,
// though the Observer binds it too - A on the map is the map's Area Alerts,
// not the Observer's alert details.
func TestTheWindowShownTakesItsKeysFirst(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	m, _ := d.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	if d = m.(Dashboard); d.modal != modalMap || !d.mapPane.alertsOn {
		t.Errorf("A on the map opened window %d (alerts box %v); want the map's Area Alerts box", d.modal, d.mapPane.alertsOn)
	}
}

// TestAFormTakesEveryKey is claimAll's reason: in a form or a search -
// Setup, Add, Remove, Request, the windows D-107 names - a key the Observer
// binds is the form's, never the Observer's. g types into the search; it does
// not open the map, and q does not quit.
func TestAFormTakesEveryKey(t *testing.T) {
	for _, m := range []modal{modalSetup, modalAdd, modalRemove, modalRequest} {
		if w, _ := windowKeysOf(m); w.claim != claimAll {
			t.Errorf("window %d is a form and does not take every key", m)
		}
		d := mapDash(t, Config{}).open(m)
		for _, k := range []string{"g", "q", "?"} {
			next, cmd := pressKey(d, k)
			if next.modal != m {
				t.Errorf("%q in window %d went to window %d: a stray key left the form", k, m, next.modal)
			}
			if cmd != nil {
				if _, quit := cmd().(tea.QuitMsg); quit {
					t.Errorf("%q in window %d quit the app", k, m)
				}
			}
		}
	}
}
