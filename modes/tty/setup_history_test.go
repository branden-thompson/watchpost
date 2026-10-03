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
	d.cfg.HistoryCost = func(bool, string, string) string { return "About 10 MB more." }
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	m, _ = m.(Dashboard).handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEnter}) // a longer window: kept (D-231)
	d = m.(Dashboard)
	d.setup.focus = rowHistoryTrends
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft}) // 30 days wraps to 5 years: longer too
	m, _ = m.(Dashboard).handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEnter})
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
	if cleared != 1 || !strings.Contains(m.(Dashboard).setup.note, "Cleared: recording") {
		t.Errorf("cleared %d times, the note %q", cleared, m.(Dashboard).setup.note)
	}
	d.cfg.ClearHistory = func() error { return errors.New("permission denied") }
	m, _ = d.Update(historyClearedMsg{err: errors.New("permission denied")})
	if !strings.Contains(m.(Dashboard).setup.note, "Partly cleared") {
		t.Errorf("a failed clear says %q", m.(Dashboard).setup.note)
	}
}

// WHAT THE HISTORY HOLDS IS SAID IN THE DATA TAB'S NOTICES (D-175, D-237):
// its size and its place.
func TestTheHistorySaysWhatItHolds(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryHours)
	d.cfg.HistoryUsage = func() string { return "3.2 MB in ~/.local/share/watchpost/weather/history" }
	if notes := strings.Join(footerText(d), "\n"); !strings.Contains(notes, "3.2 MB") || !strings.Contains(notes, "~/.local/share/watchpost/weather/history") {
		t.Errorf("the notices do not say what the history holds and where:\n%s", notes)
	}
}

// settingsText is the Settings window's body, plain.
func settingsText(d Dashboard) string {
	lines, _, _ := d.setupBody(d.opts())
	return stripANSITest(strings.Join(lines, "\n"))
}

// RAISING A RETENTION SAYS WHAT IT COSTS FIRST (D-231): a longer window opens
// a question before it is kept - what it will take on disk, and that what was
// not recorded cannot be fetched back; enter keeps it, esc puts the shorter one
// back. A shorter window asks nothing.
func TestRaisingARetentionSaysWhatItCosts(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryHours)
	var asked []string
	d.cfg.HistoryCost = func(trends bool, from, to string) string {
		asked = append(asked, from+">"+to)
		return "About 120 MB more on disk, an estimate."
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	body := stripANSITest(d.confirmOverlay(d.opts()))
	if d.setup.raise == nil || !strings.Contains(body, "About 120 MB more") || !strings.Contains(body, "cannot be fetched back") || !strings.Contains(body, "7 days") {
		t.Fatalf("raising 72 hours to 7 days did not ask, or the question lacks its cost or its warning:\n%s", body)
	}
	_ = d.confirmOverlay(d.opts()) // drawn again: the cost is not asked again
	if len(asked) != 1 || asked[0] != "72h>7d" {
		t.Errorf("the cost was asked for %v; want 72h>7d", asked)
	}
	m, _ = d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if d2 := m.(Dashboard); d2.setup.raise == nil || d2.setup.focus != rowHistoryHours {
		t.Error("a key the question does not take moved the focus under it")
	}
	m, _ = d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	d = m.(Dashboard)
	if d.setup.raise != nil || historyLabel(historyHourChoices, d.setup.history.Hours) != "72 hours" {
		t.Fatalf("esc left the window open, or kept %q; want 72 hours back", d.setup.history.Hours)
	}
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	m, _ = m.(Dashboard).handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	d = m.(Dashboard)
	if d.setup.raise != nil || d.setup.history.Hours != "7d" {
		t.Fatalf("enter did not keep 7 days (%q)", d.setup.history.Hours)
	}
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if d = m.(Dashboard); d.setup.raise != nil || historyLabel(historyHourChoices, d.setup.history.Hours) != "72 hours" {
		t.Error("a shorter window asked, or was not taken")
	}
}
