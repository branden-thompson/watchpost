package tty

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// THE HISTORY'S RETENTION IS THE DATA TAB'S PRESETS, WRITTEN ON CLOSE (D-175):
// ←→ cycle hourly detail and trends through their presets, and closing the
// window writes the choice once - and the next open shows it.
func TestTheHistorysRetentionIsChosenOnTheDataTab(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryHours)
	var got []HistoryRetention
	d.cfg.SetHistory = func(r HistoryRetention) { got = append(got, r) }
	if tabOfGroup(groupHistory) != tabData {
		t.Error("the HISTORY group is not on the Data tab")
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	d.setup.focus = rowHistoryTrends
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	d = m.(Dashboard)
	if body := settingsText(d); !strings.Contains(body, "7 days") || !strings.Contains(body, "5 years") {
		t.Errorf("the pickers do not show 7 days and 5 years:\n%s", body)
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if len(got) != 1 || got[0] != (HistoryRetention{Hours: "7d", Trends: "5y"}) {
		t.Errorf("closing wrote %+v; want one write of 7 days and 5 years", got)
	}
	if next := m.(Dashboard).cfg.History; next != (HistoryRetention{Hours: "7d", Trends: "5y"}) {
		t.Errorf("the next open would show %+v", next)
	}
}

// CLEAR HISTORY ASKS FIRST (D-177): space opens the ARE YOU SURE; while it is
// open it owns the keys; esc leaves everything recorded; enter clears, and
// the row says so.
func TestClearingTheHistoryAsksFirst(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryClear)
	cleared := 0
	d.cfg.ClearHistory = func() error { cleared++; return nil }
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: ' ', Text: " "})
	d = m.(Dashboard)
	if !d.setup.confirmClear || !strings.Contains(stripANSITest(d.confirmOverlay(d.opts())), "ARE YOU SURE?") {
		t.Fatal("space on Clear history did not ask ARE YOU SURE")
	}
	m, _ = d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.(Dashboard).setup.focus != rowHistoryClear || !m.(Dashboard).setup.confirmClear {
		t.Error("a key the question does not take moved the focus under it")
	}
	m, _ = d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	d = m.(Dashboard)
	if d.setup.confirmClear || cleared != 0 {
		t.Fatalf("esc did not cancel (cleared %d)", cleared)
	}
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, cmd := m.(Dashboard).handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter did not clear")
	}
	m, _ = m.(Dashboard).Update(cmd())
	if cleared != 1 || !strings.Contains(m.(Dashboard).setup.note, "History cleared") {
		t.Errorf("cleared %d times, the note %q", cleared, m.(Dashboard).setup.note)
	}
	d.cfg.ClearHistory = func() error { return errors.New("permission denied") }
	m, _ = d.Update(historyClearedMsg{err: errors.New("permission denied")})
	if !strings.Contains(m.(Dashboard).setup.note, "partly cleared") {
		t.Errorf("a failed clear says %q", m.(Dashboard).setup.note)
	}
}

// WHAT THE HISTORY HOLDS IS SAID UNDER IT (D-175): its size and its place.
func TestTheHistorySaysWhatItHolds(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryHours)
	d.cfg.HistoryUsage = func() string { return "3.2 MB in ~/.local/share/watchpost/weather/history" }
	if body := settingsText(d); !strings.Contains(body, "3.2 MB") || !strings.Contains(body, "~/.local/share/watchpost/weather/history") {
		t.Errorf("the group does not say what the history holds and where:\n%s", body) // wrapped at the notes' width
	}
}

// settingsText is the Settings window's body, plain.
func settingsText(d Dashboard) string {
	lines, _, _ := d.setupBody(d.opts())
	return stripANSITest(strings.Join(lines, "\n"))
}
