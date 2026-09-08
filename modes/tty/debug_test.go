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

// AND IT ASKS BEFORE IT INJECTS (HUM LEAD mock, 2026-09-07).
//
// THE CONFIRMATION IS THE FEATURE. An injected alert enters where a real one
// enters and crosses every stage a real one does, so it cannot be stopped once
// it is under way — and what it produces goes out over the operator's own
// broadcast. enter opens the question; only the answer injects.
func TestTheDebugWindowAsksBeforeItInjects(t *testing.T) {
	fired := ""
	d := wiredDebug(t, func(k string) { fired = k })

	// The window offers the Alert Type question, not a list of rows.
	body := stripANSITest(strings.Join(firstOf(d.debugLines(d.opts())), "\n"))
	for _, want := range []string{"ALERT INJECTION", "Alert Type:", "one alert"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the window does not draw %q:\n%s", want, body)
		}
	}

	// enter opens the confirmation and injects NOTHING.
	m, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	d = m.(Dashboard)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			t.Error("enter produced a command before the operator confirmed")
		}
	}
	if fired != "" {
		t.Fatalf("enter injected %q without asking", fired)
	}
	if !d.debug.confirm {
		t.Fatal("enter did not raise the confirmation")
	}
	if got := stripANSITest(d.View().Content); !strings.Contains(got, "ARE YOU SURE?") ||
		!strings.Contains(got, "YOU CANNOT STOP THIS ACTION") ||
		!strings.Contains(got, "as the operator, responsibility for") {
		t.Errorf("the confirmation does not say what it is asking:\n%s", got)
	}

	// AND THE ANSWER INJECTS WHAT THE PICKER WAS SHOWING.
	m2, cmd2 := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd2 == nil {
		t.Fatal("confirming produced no injection")
	}
	cmd2()
	if fired != "one" {
		t.Errorf("injected %q, want the value the picker was showing", fired)
	}
	if m2.(Dashboard).modal != modalNone {
		t.Error("confirming leaves the window open")
	}
}

// esc TAKES THE CONFIRMATION DOWN AND INJECTS NOTHING, and leaves the window
// where it was — a question answered "no" is not a reason to close the tool.
func TestEscOnTheConfirmationInjectsNothing(t *testing.T) {
	fired := false
	d := wiredDebug(t, func(string) { fired = true })
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.(Dashboard).Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	d = m.(Dashboard)
	if fired {
		t.Error("esc on the confirmation injected")
	}
	if d.debug.confirm {
		t.Error("esc left the confirmation up")
	}
	if d.modal != modalDebug {
		t.Error("esc on the confirmation closed the whole window")
	}
}

// THE PICKER CYCLES, with the arrows and with up/down, and it wraps like every
// other picker in the app.
func TestTheAlertTypePickerCyclesTheBuildsScenarios(t *testing.T) {
	d := wiredDebug(t, func(string) {})
	if got := d.debugPick(); got != 0 {
		t.Fatalf("the window opens on %d", got)
	}
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyDown}, {Code: tea.KeyRight}} {
		before := d.debugPick()
		m, _ := d.Update(key)
		d = m.(Dashboard)
		if d.debugPick() == before {
			t.Errorf("%v did not move the picker off %d", key, before)
		}
	}
	// AND IT WRAPS rather than stopping dead, which reads as a stuck key.
	before := d.debugPick()
	for range len(d.debugScenarios()) {
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		d = m.(Dashboard)
	}
	if got := d.debugPick(); got != before {
		t.Errorf("a full turn of the picker landed on %d, not back on %d", got, before)
	}
}

// THE WHOLE WINDOW FITS THE APP'S FLOOR. It is prose, one question and a
// footer, and at 80x24 nothing about it needs scrolling — which is what makes
// its ONE control reachable without the size contract F-55 asks for.
func TestTheDiagnosticsWindowFitsTheFloor(t *testing.T) {
	d := wiredDebug(t, func(string) {})
	d.width, d.height = 80, 24
	o := d.opts()
	o.Width = min(o.Width, debugWidth)
	body, _, _ := d.debugLines(o)
	if got, budget := len(d.wrapModal(body, o.Width)), d.modalMax()-1; got > budget {
		t.Errorf("the window is %d lines against a %d-line budget at 80x24: its one control needs "+
			"a scroll to reach", got, budget)
	}
}

// wiredDebug is the window as a debug build has it: an injector, and scenarios.
func wiredDebug(t *testing.T, inject func(string)) Dashboard {
	t.Helper()
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d.cfg.InjectAlert = inject
	d.cfg.DebugScenarios = []DebugScenario{{Label: "one alert", Key: "one"}, {Label: "a burst", Key: "burst"}}
	return d.open(modalDebug)
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

// AND THE WINDOW A RELEASE BUILD SHIPS IS READABLE AT THE FLOOR (FR-5).
//
// Injection is compiled out of a release binary, so the shipped ctrl+d window
// is prose and no question. It used to run past the fold at 80x24 with the
// scroll pinned at zero — every line of what it exists to say unreachable by
// any key, in the build that ships. It FITS now, and both halves are asserted:
// that it fits, and that every line of it is on screen.
func TestTheShippedDiagnosticsWindowIsReadableAtTheFloor(t *testing.T) {
	d := debugAtTheFloor(t)
	d.cfg.InjectAlert, d.cfg.DebugScenarios = nil, nil // a release build

	o := d.opts()
	o.Width = min(o.Width, debugWidth)
	body, _, _ := d.debugLines(o)
	want := d.wrapModal(body, o.Width)
	if len(want) > d.modalMax()-1 {
		t.Fatalf("the shipped window is %d lines against a %d-line budget at 80x24", len(want), d.modalMax()-1)
	}
	got := stripANSITest(d.modalView(d.opts()))
	for _, l := range want {
		if text := modalTextOf(l); text != "" && !strings.Contains(stripANSITest(got), text) {
			t.Errorf("%q is in the window and not on the screen at 80x24:\n%s", text, got)
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
