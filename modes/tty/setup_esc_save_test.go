package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// CLOSING THE WINDOW MUST LEAVE THE MODEL AGREEING WITH WHAT IT WROTE.
//
// HUM LEAD, UAT 2026-09-08: "[s] Settings — esc save does not work — changed
// [20] mi radius and spanish choice — esc and re-open did not preserve my
// choices."
//
// The write was never the problem. These settings persist through a setter that
// returns nothing, so nothing wrote the value back into d.cfg — and openSetup
// seeds the form FROM d.cfg, so re-opening showed the OLD choice. The same UAT
// proved the write was real: the [w] window correctly trimmed its events to the
// new radius while Settings still displayed the previous one.
//
// THE SECOND HALF IS WORSE THAN THE FIRST. applyIfChanged compares against
// d.cfg, so with a stale d.cfg the change could not be UNDONE either: selecting
// "All locations" compared 0 against a stale 0, saw no change, and wrote
// nothing. The radius stayed in force with no way back through the window.
func TestClosingSettingsLeavesTheModelAgreeingWithTheWrite(t *testing.T) {
	h := &setupHarness{}
	cfg := h.config()
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})

	toEvents := func(model tea.Model) tea.Model {
		model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		return model
	}

	// First run seeds the location, then the events group.
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model = typeText(model, "20")

	var cmd tea.Cmd
	model, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, model, cmd)

	if d := model.(Dashboard); d.modal != modalNone {
		t.Fatalf("esc closes the window; modal=%v", d.modal)
	}
	if !h.radiusSet || h.radius != 20 {
		t.Fatalf("esc writes the radius: set=%v radius=%d", h.radiusSet, h.radius)
	}
	// THE MODEL, not just the setter. This is the assertion the bug needed.
	if got := model.(Dashboard).cfg.AlertRadiusMi; got != 20 {
		t.Errorf("the model must know what it wrote; cfg.AlertRadiusMi=%d want 20", got)
	}
	if view := stripANSITest(toEvents(model).(Dashboard).View().Content); !strings.Contains(view, "● Within [20") {
		t.Errorf("re-opening shows the radius that was saved:\n%s", view)
	}

	// AND IT CAN BE UNDONE. Selecting All must write 0 — with a stale d.cfg this
	// wrote nothing and the radius could not be cleared from the window at all.
	h.radiusSet, h.radius = false, -1
	model = toEvents(model)
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}) // select the focused radio: All
	model, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, model, cmd)
	if !h.radiusSet || h.radius != 0 {
		t.Errorf("switching back to All must write 0: set=%v radius=%d", h.radiusSet, h.radius)
	}
	if got := model.(Dashboard).cfg.AlertRadiusMi; got != 0 {
		t.Errorf("and the model must follow it back: cfg.AlertRadiusMi=%d want 0", got)
	}
}

// THE RELAY LANGUAGE IS THE SAME BUG, and the UAT named it in the same breath:
// "changed [20] mi radius and spanish choice". It is a PICKER rather than a
// typed field, so it never had the append problem — only the stale model.
func TestTheRelayLanguageSurvivesClosingTheWindow(t *testing.T) {
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	d := model.(Dashboard)
	d.setup.relayLang = "es" // the picker's outcome, without walking the whole window
	var cmd tea.Cmd
	var next tea.Model = d
	next, cmd = next.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, next, cmd)

	if !h.langSet || h.lang != "es" {
		t.Fatalf("esc writes the relay language: set=%v lang=%q", h.langSet, h.lang)
	}
	if got := next.(Dashboard).cfg.RelayLang; got != "es" {
		t.Errorf("the model must know the language it wrote: cfg.RelayLang=%q want \"es\"", got)
	}
}
