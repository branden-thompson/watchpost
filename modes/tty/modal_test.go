package tty

// Quality pass Q6 (plan Q6.1, red-team L3-F15, A11-6): modal exclusivity,
// asserted on the RENDERED frame — for every open modal and every opener
// (each key that opens a window, and the voice-error message that reopens
// the Voice chooser), View() carries at most one modal, and esc never moves
// the focus. Written before the `type modal int` refactor; it is the pin
// that refactor satisfies.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// modalMarkers are the strings that appear in a frame only while that
// modal is open (the panel title as the frame draws it, or a body line no
// other window shows).
var modalMarkers = map[string]string{
	"help":    "─ Watchpost Help ─",
	"details": "CURRENTLY │",
	"add":     "─ Add Location ─",
	"lookup":  "─ Lookup Location ─",
	"remove":  "─ Remove Location ─",
	"alerts":  "─ ALERT",
	"status":  "─ Watchpost Status ─",
	"about":   "Data Provided by:",
	"setup":   "─ Settings ─",
	"severe":  "NOTABLE EVENTS AND FORECASTS",
}

// openers are every way a window opens.
var openers = map[string]tea.Msg{
	"help":    tea.KeyPressMsg{Code: '?', Text: "?"},
	"details": tea.KeyPressMsg{Code: tea.KeyEnter},
	"lookup":  tea.KeyPressMsg{Code: 'l', Text: "l"}, // ctrl+a Add is row-gated; lookup covers modalAdd here
	"remove":  tea.KeyPressMsg{Code: tea.KeyDelete, Mod: tea.ModShift},
	"alerts":  tea.KeyPressMsg{Code: 'A', Text: "A"},
	"status":  tea.KeyPressMsg{Code: 'S', Text: "S"},
	"about":   tea.KeyPressMsg{Code: 'a', Text: "a"},
	"setup":   tea.KeyPressMsg{Code: 's', Text: "s"},
	"severe":  tea.KeyPressMsg{Code: 'w', Text: "w"},
}

// modalsIn lists the markers present in a frame.
func modalsIn(frame string) []string {
	var found []string
	for name, marker := range modalMarkers {
		if strings.Contains(frame, marker) {
			found = append(found, name)
		}
	}
	return found
}

func modalFixture(t *testing.T) tea.Model {
	t.Helper()
	d := dash(t).(Dashboard).WithVoices(func() []string { return []string{"Alex", "Samantha"} }, "Alex", func(string) error { return nil }, func(string) {})
	return d
}

func TestExactlyOneModalRendersWhateverOpensOverWhat(t *testing.T) {
	for first, open := range openers {
		for second, next := range openers {
			m := modalFixture(t)
			m, _ = m.Update(open)
			before := m.(Dashboard).selected
			if got := modalsIn(stripANSITest(m.View().Content)); len(got) > 1 {
				t.Errorf("%s alone: %v modals rendered", first, got)
			}
			m, _ = m.Update(next)
			frame := stripANSITest(m.View().Content)
			if got := modalsIn(frame); len(got) > 1 {
				t.Errorf("%s then %s: stacked modals %v", first, second, got)
			}
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if d := m.(Dashboard); d.selected != before {
				t.Errorf("%s then %s: esc moved the focus %d -> %d", first, second, before, d.selected)
			}
		}
	}
}

// [t] and [M] no longer open windows of their own: both DEEP-LINK into Settings
// (0.14.0). They are exercised here rather than in openers, which is keyed by
// the window a key opens, because they open a window someone else already owns.
func TestTheRetiredChoosersDeepLinkIntoSettings(t *testing.T) {
	for _, tc := range []struct{ name, key string }{{"theme", "t"}, {"mute", "M"}} {
		m := modalFixture(t)
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(tc.key[0]), Text: tc.key})
		frame := stripANSITest(m.View().Content)
		if got := modalsIn(frame); len(got) != 1 || got[0] != "setup" {
			t.Errorf("%s opens Settings and nothing else, got %v", tc.name, got)
		}
		if !strings.Contains(frame, "› ") {
			t.Errorf("%s: Settings opens with a row focused", tc.name)
		}
	}
}

func TestEveryModalHasItsMarker(t *testing.T) {
	for name, open := range openers {
		m := modalFixture(t)
		m, _ = m.Update(open)
		got := modalsIn(stripANSITest(m.View().Content))
		if len(got) != 1 || got[0] != name {
			t.Errorf("%s: rendered %v", name, got)
		}
	}
}
