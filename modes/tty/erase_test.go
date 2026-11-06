package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// backspace is the key every text field erases on.
var backspace = tea.KeyPressMsg{Code: tea.KeyBackspace}

// TestBackspaceErasesALetterInEveryField is 0.19.5 D-5 (DEF-1, RN-4): backspace
// takes one letter, never one byte, from every field that takes text, and each
// field keeps the side effect it has beside the erase. "Peñ" ends in a
// two-byte letter, so a field that cut a byte would be left holding half of it.
func TestBackspaceErasesALetterInEveryField(t *testing.T) {
	var asked []string
	base, err := NewDashboard(Config{Suggest: func(q string, _ int) []snapshot.LocationRef {
		asked = append(asked, q)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		// erase presses backspace on a field holding the given text, and returns
		// what the field holds after, with an error for a lost side effect.
		erase func(text string) (string, string)
		text  string
		want  string
	}{
		{"the request's location", func(s string) (string, string) {
			d := base.openRequest()
			d.request.field, d.request.query = requestLocation, s
			m, _ := d.handleRequestKey(backspace)
			out := m.(Dashboard)
			if out.request.locate.query != out.request.query {
				return out.request.query, "the locate pause was not restarted on the new query"
			}
			return out.request.query, ""
		}, "Peñ", "Pe"},
		{"the request's position", func(s string) (string, string) {
			d := base.openRequest()
			d.request.field, d.request.slot = requestPosition, s
			m, _ := d.handleRequestKey(backspace)
			return m.(Dashboard).request.slot, ""
		}, "12", "1"},
		{"the Lookup window", func(s string) (string, string) {
			d := base
			d.addQuery = s
			m, _ := d.handleAddKey(backspace)
			return m.(Dashboard).addQuery, ""
		}, "Peñ", "Pe"},
		{"Settings' place", func(s string) (string, string) {
			d := base
			d.setup.query, asked = s, nil
			m, _ := d.setupLocationKey(backspace)
			out := m.(Dashboard)
			if len(asked) != 1 || asked[0] != out.setup.query {
				return out.setup.query, "the suggestions were not asked again for the shorter query"
			}
			return out.setup.query, ""
		}, "Peñ", "Pe"},
		{"Settings' service radius", func(s string) (string, string) {
			d := base
			d.setup.serviceMi, d.setup.serviceSeeded = s, true
			m, _ := d.setupServiceKey(backspace)
			out := m.(Dashboard)
			if out.setup.serviceSeeded {
				return out.setup.serviceMi, "the radius is still marked as the stored one"
			}
			return out.setup.serviceMi, ""
		}, "25", "2"},
		{"Settings' FIRMS key", func(s string) (string, string) {
			d := base
			d.setup.key = s
			m, _ := d.setupKeyKey(backspace)
			return m.(Dashboard).setup.key, ""
		}, "Peñ", "Pe"},
		{"Settings' alert radius", func(s string) (string, string) {
			d := base
			d.setup.radiusMi, d.setup.radiusSeeded = s, true
			m, _ := d.setupAlertKey(backspace)
			out := m.(Dashboard)
			if out.setup.radiusSeeded {
				return out.setup.radiusMi, "the radius is still marked as the stored one"
			}
			return out.setup.radiusMi, ""
		}, "25", "2"},
		{"the Propagation finder", func(s string) (string, string) {
			d := base
			d.mapPane.propFindQuery = s
			m, _ := d.handlePropFindKey(backspace)
			return m.(Dashboard).mapPane.propFindQuery, ""
		}, "Peñ", "Pe"},
		{"the frequency's field", func(s string) (string, string) {
			d := base
			d.mapPane.propFreqQuery = s
			m, _ := d.handlePropFreqKey(backspace)
			return m.(Dashboard).mapPane.propFreqQuery, ""
		}, "14.07ñ", "14.07"},
		{"the card's new position", func(s string) (string, string) {
			r := NewRouter(base)
			r.observer.cardMoveTo, r.observer.cardErr = s, "positions 1 to 8"
			out, _ := r.moveWindowKey(backspace)
			if out.observer.cardErr != "" {
				return out.observer.cardMoveTo, "the error line was not cleared"
			}
			return out.observer.cardMoveTo, ""
		}, "12", "1"},
	} {
		got, lost := c.erase(c.text)
		if got != c.want {
			t.Errorf("%s: backspace on %q leaves %q, want %q", c.name, c.text, got, c.want)
		}
		if lost != "" {
			t.Errorf("%s: %s", c.name, lost)
		}
	}
}

// TestTheRequestWindowsKeysDoWhatTheyClaim is 0.19.5 RN-4: every key the
// Line-Up Request window offers does what its hints say, so a change to the
// window's keys cannot pass unseen.
func TestTheRequestWindowsKeysDoWhatTheyClaim(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)
	space := tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	step := func(d Dashboard, k tea.KeyPressMsg) Dashboard {
		t.Helper()
		m, _ := d.handleRequestKey(k)
		return m.(Dashboard)
	}

	if up := step(d, tea.KeyPressMsg{Code: tea.KeyUp}); up.request.field != requestPosition {
		t.Errorf("up from the location lands on field %v, want the position: the form wraps", up.request.field)
	}

	tabbed := step(d, keyPress(t, "tab"))
	if tabbed.request.field != requestReports || tabbed.request.at != 0 {
		t.Errorf("tab from the location lands on field %v row %d, want the first report row",
			tabbed.request.field, tabbed.request.at)
	}
	if again := step(tabbed, keyPress(t, "tab")); again.request.field != requestPosition {
		t.Errorf("tab from the reports lands on field %v, want the position: tab walks fields, not rows",
			again.request.field)
	}

	for row := range requestRows() {
		at := tabbed
		at.request.at = row
		if out := step(at, keyPress(t, "tab")); out.request.field != requestPosition || out.request.at != 0 {
			t.Errorf("tab from report row %d lands on field %v row %d, want the position", row, out.request.field, out.request.at)
		}
	}

	if typed := step(d, space); typed.request.query != " " {
		t.Errorf("space in the location leaves %q, want a space: it is a letter of a place's name", typed.request.query)
	}

	first := requestRows()[0]
	before := tabbed.request.chosen.Has(first)
	if toggled := step(tabbed, space); toggled.request.chosen.Has(first) == before {
		t.Errorf("space on the %v row leaves it chosen=%v", first, before)
	}

	pos := d
	pos.request.field, pos.request.slot = requestPosition, ""
	flipped := step(pos, space)
	if flipped.request.prioritize == pos.request.prioritize {
		t.Error("space on the position does not flip PRIORITIZE")
	}
	typed := step(step(flipped, tea.KeyPressMsg{Code: '4', Text: "4"}), tea.KeyPressMsg{Code: 'x', Text: "x"})
	if typed.request.slot != "4" {
		t.Errorf("typing 4 then x into the position leaves %q, want \"4\": digits only", typed.request.slot)
	}
	if typed.request.prioritize {
		t.Error("typing a slot leaves PRIORITIZE chosen: typing a slot is choosing the slot")
	}
}
