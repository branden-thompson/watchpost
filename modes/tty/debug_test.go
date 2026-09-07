package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// THE WINDOW CANNOT OFFER WHAT THE BUILD DOES NOT HAVE (F-21).
//
// A release build compiles the injector out, so the app supplies no hook and no
// scenarios. The window must then say so plainly rather than render a row that
// does nothing — a disabled control is a control someone will try, and this one
// fabricates tornado warnings.
func TestTheDebugWindowOffersNoInjectionWithoutAHook(t *testing.T) {
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d = d.open(modalDebug)

	got := stripANSITest(strings.Join(firstOf(d.debugLines(d.opts())), "\n"))
	if !strings.Contains(got, "NOT AVAILABLE IN THIS BUILD") {
		t.Errorf("a build with no injector says so:\n%s", got)
	}
	if strings.Contains(got, "INJECT AN ALERT") {
		t.Error("a build with no injector must offer no injection")
	}
	if len(d.debugScenarios()) != 0 {
		t.Error("no hook means no scenarios")
	}

	// AND SCENARIOS WITHOUT A HOOK OFFER NOTHING EITHER. A build that listed
	// them and wired no injector would render rows that silently do nothing —
	// and a row that fabricates a tornado warning is the last place to leave a
	// control that might or might not be connected. The first version of this
	// test left both nil, so a mutant deleting the guard SURVIVED it.
	d.cfg.DebugScenarios = []DebugScenario{{Label: "one alert", Key: "one"}}
	if got := d.debugScenarios(); len(got) != 0 {
		t.Errorf("scenarios without an injector must not be offered, got %v", got)
	}
	if body := stripANSITest(strings.Join(firstOf(d.debugLines(d.opts())), "\n")); strings.Contains(body, "one alert") {
		t.Errorf("an unwired scenario must not be drawn:\n%s", body)
	}
}

// AND IT OFFERS THEM WHEN THE BUILD DOES, with the focus mark that every other
// list in the app uses.
func TestTheDebugWindowFiresTheFocusedScenario(t *testing.T) {
	fired := ""
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d.cfg.InjectAlert = func(k string) { fired = k }
	d.cfg.DebugScenarios = []DebugScenario{{Label: "one alert", Key: "one"}, {Label: "a burst", Key: "burst"}}
	d = d.open(modalDebug)

	got := stripANSITest(strings.Join(firstOf(d.debugLines(d.opts())), "\n"))
	if !strings.Contains(got, "INJECT AN ALERT") || !strings.Contains(got, "a burst") {
		t.Fatalf("the scenarios must be listed:\n%s", got)
	}
	// ↑↓ wrap like every other list, and enter fires what is focused.
	d = d.handleDebugNav("nav-down")
	if d.debug.focus != 1 {
		t.Fatalf("focus moved to %d", d.debug.focus)
	}
	next, cmd := d.chooseDebug()
	if cmd == nil {
		t.Fatal("choosing must produce the injection")
	}
	cmd()
	if fired != "burst" {
		t.Errorf("fired %q, want the focused scenario", fired)
	}
	if next.modal != modalNone {
		t.Error("choosing closes the window")
	}
}

// esc closes it and injects nothing.
func TestEscClosesTheDebugWindowWithoutInjecting(t *testing.T) {
	fired := false
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d.cfg.InjectAlert = func(string) { fired = true }
	d.cfg.DebugScenarios = []DebugScenario{{Label: "one", Key: "one"}}
	var m tea.Model = d.open(modalDebug)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.(Dashboard).modal != modalNone {
		t.Error("esc closes it")
	}
	if fired {
		t.Error("esc injects nothing")
	}
}

// firstOf is the lines half of debugLines' (lines, focusAt, focusEnd).
func firstOf(lines []string, _, _ int) []string { return lines }
