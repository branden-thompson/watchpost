package tty

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// M2 — THE KEYPRESSES FROM THE DASHBOARD TO "ROLE X SPEAKS IN VOICE Y".
//
// MEASURED HERE, NOT IN THE PTY JOURNEY, because the journey could not measure
// it. Its counter incremented once per scripted `send` and asserted 11 <= 11 —
// arithmetic, incapable of failing (F-6). Rewritten to stop when the window
// closed, it read a late redraw as "closed" and reported 2, then 3. An
// app-level probe settled that: Settings is demonstrably still open after seven
// keys, so the PTY's signal was wrong, not the app.
//
// A PTY is the right instrument for "does this journey work against the real
// binary" and the wrong one for "how many keys does it take". This drives the
// same Update loop the terminal drives and cannot be fooled by redraw timing.
//
// ESC IS THE SAVE. A picker row cycles with the arrows for as long as you like
// and commits nothing; the window has exactly two exits, and both write through
// sequenceWrites (setup.go: "no group can be saved by one route and dropped by
// the other"). Enter ADVANCES from a picker row rather than saving, which is
// why a path built out of Enters never terminated — and why the original ≤ 11
// was never measuring a completed assignment either.
func TestM2TheKeypressesToAssignACorrespondent(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	saved := 0
	var saw CastView
	d := m
	d.cfg.Voices = func() []string { return []string{"System Voice", "Daniel", "Karen"} }
	d.cfg.VoiceInstalled = func(string) bool { return true }
	d.cfg.PreviewVoice = func(string) {}
	d.cfg.SetCast = func(c CastView) error { saved++; saw = c; return nil }
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC) }

	var model tea.Model = d
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})

	// From the DASHBOARD: V deep-links into Settings at the correspondents
	// (MVS-D-3), the cast switch goes on, the Alerts row takes a voice, and esc
	// closes-and-writes.
	path := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"V", tea.KeyPressMsg{Code: 'V', Text: "V"}},
		{"down", tea.KeyPressMsg{Code: tea.KeyDown}},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}},
		{"down", tea.KeyPressMsg{Code: tea.KeyDown}},
		{"right", tea.KeyPressMsg{Code: tea.KeyRight}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}},
	}
	keys, dirtied := 0, false
	var preClose Dashboard
	for _, k := range path {
		preClose = model.(Dashboard)
		model, _ = model.Update(k.key)
		keys++
		if model.(Dashboard).setup.castDirty {
			dirtied = true
		}
		if model.(Dashboard).modal != modalSetup {
			break
		}
	}
	if !dirtied {
		t.Fatalf("no assignment was made in %d keypresses, so there is nothing to persist", keys)
	}
	if model.(Dashboard).modal == modalSetup {
		t.Fatalf("Settings never closed in %d keypresses; the path does not reach a save", keys)
	}

	// THE SAVE IS A COMMAND THE RUNTIME UNWRAPS. esc returns
	// tea.Sequence(applyOnCloseCmds()...), and calling a Sequence once does not
	// run what is inside it — bubbletea does that. So the write is invoked here
	// the way the close composes it, from the state the close saw: this is
	// castApplyCmd, which applyOnCloseCmds sequences first.
	write := preClose.castApplyCmd()
	if write == nil {
		t.Fatal("closing Settings composed no cast write, so the assignment would be dropped on exit")
	}
	if msg := write(); msg != nil {
		if v, ok := msg.(castSavedMsg); ok && v.err != nil {
			t.Fatalf("the cast write failed: %v", v.err)
		}
	}
	if saved == 0 {
		t.Fatal("the closing write did not persist the cast: SetCast was never called")
	}
	// "role X speaks in voice Y" is the metric's own wording, so the saved view
	// must actually name a voice for a role — not merely have been written.
	if saw.Mode != castModeOn {
		t.Errorf("the cast was saved in mode %q, want %q: the switch never went on, so no role has its own voice", saw.Mode, castModeOn)
	}
	if len(saw.Names) == 0 {
		t.Error("the cast was saved with no role named; the path closed the window without assigning a voice")
	}

	t.Logf("M2: assigning a correspondent takes %d keypresses (pin <= 11; the project brief's target is 5)", keys)
	if keys > 11 {
		t.Errorf("M2: %d keypresses, pin <= 11 (AM-18 / MVS-D-40)", keys)
	}
}
