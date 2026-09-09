package tty

import (
	"strings"
	"testing"
	"time"

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

// TYPING OVER A STORED RADIUS REPLACES IT (HUM LEAD, UAT 2026-09-08).
//
// The window opened reading "[50] mi", the listener typed 20 — the obvious way
// to change it — and got 5020: a five-thousand-mile radius, silently saved.
// That is what "did not preserve my choices" actually was; the choice WAS
// preserved, it just was not the one entered.
//
// No test covered this because both existing radius tests avoid the case: one
// starts from an EMPTY field and types "50", the other uses space to pick All.
// Typing over a value nobody had typed was the untested path.
func TestTypingOverAStoredRadiusReplacesItRatherThanAppending(t *testing.T) {
	open := func(stored int) tea.Model {
		h := &setupHarness{}
		cfg := h.config()
		cfg.AlertRadiusMi = stored
		m, err := NewDashboard(cfg)
		if err != nil {
			t.Fatal(err)
		}
		var model tea.Model = m
		model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
		model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
		model = typeText(model, "oce")
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		return model
	}

	// The reported case, exactly.
	model := open(50)
	if got := model.(Dashboard).setup.radiusMi; got != "50" {
		t.Fatalf("the window opens showing the stored radius; got %q", got)
	}
	model = typeText(model, "20")
	if got := model.(Dashboard).setup.radiusMi; got != "20" {
		t.Errorf("typing 20 over a stored 50 means 20, not %q", got)
	}

	// AND THE DIGITS AFTER THE FIRST STILL APPEND — replacing on every keypress
	// would make a two-digit radius impossible to enter, which is the opposite
	// failure and just as bad.
	model = open(50)
	model = typeText(model, "125")
	if got := model.(Dashboard).setup.radiusMi; got != "125" {
		t.Errorf("only the FIRST digit replaces; got %q want \"125\"", got)
	}

	// A field the listener has already edited belongs to them: backspace then
	// type appends rather than replacing again.
	model = open(50)
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = typeText(model, "7")
	if got := model.(Dashboard).setup.radiusMi; got != "57" {
		t.Errorf("after a backspace the buffer is the listener's; got %q want \"57\"", got)
	}

	// An empty field is unchanged behaviour.
	model = open(0)
	model = typeText(model, "30")
	if got := model.(Dashboard).setup.radiusMi; got != "30" {
		t.Errorf("an empty field still builds normally; got %q", got)
	}
}

// THE TWO GUARD SETS MUST AGREE, AND NOTHING MADE THEM (red team, 2026-09-08).
//
// commitToModel's own comment says it: "The guards match applyIfChanged's
// exactly. If they drift, the model and the file disagree about what is in
// force, which is a worse bug than this one." That was written, and then not
// tested — the shape this release is about.
//
// The drift is silent and asymmetric, which is why it needs a test rather than
// care. If applyIfChanged writes where commitToModel does not, the file moves
// ahead of the model and the window shows a stale value — the defect just
// fixed. If commitToModel updates where applyIfChanged does not write, the model
// moves ahead of the FILE: the window shows a value that was never saved and is
// gone at the next launch, which is worse because nothing on screen is wrong
// until a restart.
func TestTheModelAndTheFileAgreeAboutWhatWasWritten(t *testing.T) {
	// Each case: a form state, and whether the write is expected. The model must
	// change exactly when the write happens, never on one side alone.
	for _, tc := range []struct {
		name    string
		seed    func(*setupState)
		cfg     func(*Config)
		wrote   bool
		reading func(Dashboard) any
		want    any
	}{
		{"radius changes", func(s *setupState) { s.filtered, s.radiusMi = true, "20" },
			func(c *Config) { c.AlertRadiusMi = 5 }, true,
			func(d Dashboard) any { return d.cfg.AlertRadiusMi }, 20},
		{"radius unchanged", func(s *setupState) { s.filtered, s.radiusMi = true, "5" },
			func(c *Config) { c.AlertRadiusMi = 5 }, false,
			func(d Dashboard) any { return d.cfg.AlertRadiusMi }, 5},
		{"radius back to All", func(s *setupState) { s.filtered = false },
			func(c *Config) { c.AlertRadiusMi = 5 }, true,
			func(d Dashboard) any { return d.cfg.AlertRadiusMi }, 0},
		{"language changes", func(s *setupState) { s.relayLang = "es" },
			func(c *Config) { c.RelayLang = "en" }, true,
			func(d Dashboard) any { return d.cfg.RelayLang }, "es"},
		// AN INVALID VALUE WRITES NOTHING AND MOVES NOTHING. This is the case the
		// two guard sets could disagree about without any other test noticing.
		{"language empty is not a choice", func(s *setupState) { s.relayLang = "" },
			func(c *Config) { c.RelayLang = "en" }, false,
			func(d Dashboard) any { return d.cfg.RelayLang }, "en"},
		{"dwell zero is not a choice", func(s *setupState) { s.relayDwell = 0 },
			func(c *Config) { c.RelayDwell = time.Minute }, false,
			func(d Dashboard) any { return d.cfg.RelayDwell }, time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &setupHarness{}
			cfg := h.config()
			tc.cfg(&cfg)
			m, err := NewDashboard(cfg)
			if err != nil {
				t.Fatal(err)
			}
			d := m
			tc.seed(&d.setup)

			cmd := d.applyOnCloseCmds()
			after := d.commitToModel()

			// Did a write actually go out? sequenceWrites returns nil when every
			// member is a no-op, which is the file's answer.
			wrote := cmd != nil
			if wrote != tc.wrote {
				t.Errorf("the FILE: wrote=%v want %v", wrote, tc.wrote)
			}
			if got := tc.reading(after); got != tc.want {
				t.Errorf("the MODEL: got %v want %v", got, tc.want)
			}
		})
	}
}
