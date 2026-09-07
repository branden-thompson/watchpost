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

// THE FOCUSED SCENARIO IS ON SCREEN AT THE APP'S FLOOR (FR-5).
//
// modalFocusScroll computed its offset from debugLines' indices, which count
// the body BEFORE floatModalFooter wraps it — and this window's prose is inset
// to an 84-column box, so at 80 columns the panel re-wraps it three columns
// narrower and every index below the first wrapped paragraph moves. Measured:
// the focused scenario sits at unwrapped 11 and wrapped 14, the scroll came out
// 1, and the window drew lines 1..11. The cursor moved and the screen did not
// change — the dead keyboard the relay-fault window was fixed for on
// 2026-09-05, in the window next door, because that fix did its arithmetic in
// the wrong coordinates.
func TestEveryDebugScenarioIsOnScreenWhenFocusedAtTheFloor(t *testing.T) {
	d := debugAtTheFloor(t)

	// THE FIXTURE MUST NOT FIT, or the scroll is legitimately zero and this
	// measures nothing.
	o := d.opts()
	o.Width = min(o.Width, debugWidth)
	body, _, _ := d.debugLines(o)
	if wrapped := d.wrapModal(body, o.Width); len(wrapped) <= d.modalMax() {
		t.Fatalf("the window is %d lines and the budget is %d; it fits", len(wrapped), d.modalMax())
	}

	var model tea.Model = d
	for i, s := range d.debugScenarios() {
		got := stripANSITest(model.(Dashboard).modalView(model.(Dashboard).opts()))
		if !strings.Contains(got, s.Label) {
			t.Errorf("with the cursor on scenario %d, %q is off screen — the listener cannot see what enter would inject:\n%s",
				i, s.Label, got)
		}
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
}

// AND THE WINDOW A RELEASE BUILD SHIPS CAN BE READ AT THE FLOOR (FR-5).
//
// Injection is compiled out of a release binary, so the shipped ctrl+d window
// has no list to focus — and with nothing focused the scroll was pinned at
// zero, ↑↓ did nothing, and at 80x24 the whole of what that window exists to
// say ("INJECTION IS NOT AVAILABLE IN THIS BUILD", and how to get it) sat below
// the fold with no key that would reach it. Five lines unreachable by any
// input, in the build that ships.
func TestTheShippedDiagnosticsWindowIsReadableAtTheFloor(t *testing.T) {
	d := debugAtTheFloor(t)
	d.cfg.InjectAlert, d.cfg.DebugScenarios = nil, nil // a release build

	o := d.opts()
	o.Width = min(o.Width, debugWidth)
	body, _, _ := d.debugLines(o)
	want := d.wrapModal(body, o.Width)
	if len(want) <= d.modalMax() {
		t.Fatalf("the window is %d lines and the budget is %d; it fits", len(want), d.modalMax())
	}

	seen := map[string]bool{}
	var model tea.Model = d
	for i := 0; i <= len(want); i++ { // exhaust the input before reporting a limit
		for _, l := range strings.Split(stripANSITest(model.(Dashboard).modalView(model.(Dashboard).opts())), "\n") {
			seen[modalTextOf(l)] = true
		}
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for _, l := range want {
		if text := modalTextOf(l); text != "" && !seen[text] {
			t.Errorf("%q is in the window and no amount of ↑↓ reaches it at 80x24", text)
		}
	}
}

// debugAtTheFloor is the ctrl+d window at 80x24 — the app's documented floor —
// with the scenarios a debug build offers (app/inject_debug.go).
func debugAtTheFloor(t *testing.T) Dashboard {
	t.Helper()
	d := dash(t).(Dashboard)
	d.cfg.InjectAlert = func(string) {}
	d.cfg.DebugScenarios = []DebugScenario{
		{Label: "One Tornado Warning (single read)", Key: "one"},
		{Label: "A burst of six alerts (head, lines, tail, divert)", Key: "burst"},
		{Label: "An Emergency Order (leads the rail, overruns Max)", Key: "emergency"},
	}
	d.width, d.height = 80, 24
	return d.open(modalDebug)
}

// modalTextOf is a rendered line's TEXT: no ANSI, no box, no scroll rail, no
// list cursor, and runs of spaces collapsed.
//
// THE CURSOR IS NOT TEXT, and dropping it is what makes the comparison sound: a
// row reads "› a burst" while it is focused and "a burst" while it is not, so a
// probe that kept the glyph reported the unfocused form as unreachable — an
// artifact of the instrument, in a measurement whose whole subject is
// instruments that lie.
func modalTextOf(line string) string {
	text := strings.Trim(stripANSITest(line), "│┃▲▼█▓░─━╌┌┐└┘├┤ ")
	text = strings.TrimLeft(text, "›> ")
	return strings.Join(strings.Fields(text), " ")
}
