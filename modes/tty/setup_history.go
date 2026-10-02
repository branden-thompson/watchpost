package tty

// setup_history.go — the Data tab's HISTORY group (W18; D-171, D-175, D-177):
// how long the local history keeps its hours and its trends, what it holds on
// disk, and Clear history behind the app's ARE YOU SURE.
//
// THE CHOICES ARE PRESETS (D-175), each a picker of the app's own (D-147):
// hourly detail 72 hours (the default) / 7 days / 30 days / 1 year; trends -
// the days rolled up past the hours - 30 days (the default) / 90 days /
// 1 year / 5 years. Written on close, as every setting is, and applied to the
// running store at once.

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// HistoryRetention is the history's retention as the Data tab chose it: the
// hours' key and the trends' key ("" is the default, D-175).
type HistoryRetention struct{ Hours, Trends string }

// historyChoice is one preset: its key in the file and its words.
type historyChoice struct{ key, label string }

// historyHourChoices and historyTrendChoices are the Data tab's presets, the
// default first (D-175).
var (
	historyHourChoices  = []historyChoice{{"72h", "72 hours"}, {"7d", "7 days"}, {"30d", "30 days"}, {"1y", "1 year"}}
	historyTrendChoices = []historyChoice{{"30d", "30 days"}, {"90d", "90 days"}, {"1y", "1 year"}, {"5y", "5 years"}}
)

// historyValueW is the history pickers' value width: their longest, "72 hours".
const historyValueW = 8

// historyLabelW is the group's label width: its longest label, so its
// pickers line up.
const historyLabelW = len("Hourly detail -")

// historyAt is a key's place among choices; the default's where it is not one.
func historyAt(choices []historyChoice, key string) int {
	if key == "" {
		return 0
	}
	for i, c := range choices { // four (P10-02)
		if c.key == key {
			return i
		}
	}
	return 0
}

// cycleHistory is the next or previous preset's key.
func cycleHistory(choices []historyChoice, key string, forward bool) string {
	if len(choices) == 0 {
		return key
	}
	return cycleIn(choices, func(c historyChoice) string { return c.key }, key, forward).key
}

// historyLines are the HISTORY group's rows: the two presets, the store's
// size and place, and Clear history.
func (d Dashboard) historyLines(o render.Opts) ([]string, int) {
	var lines []string
	at := 0
	focus := d.setup.focus
	// AS NARROW AS THEIR VALUES: a picker as wide as the map's would not let the
	// Data tab's two columns fit beside each other in ASCII, and the whole
	// window would narrow to one column (every tab's width is the widest's).
	lines, at = d.mapRowW(o, lines, at, rowHistoryHours, "Hourly detail -", d.mapPickerW(o, rowHistoryHours, historyHourChoices[historyAt(historyHourChoices, d.setup.history.Hours)].label, historyValueW), historyLabelW)
	lines, at = d.mapRowW(o, lines, at, rowHistoryTrends, "Trends -", d.mapPickerW(o, rowHistoryTrends, historyTrendChoices[historyAt(historyTrendChoices, d.setup.history.Trends)].label, historyValueW), historyLabelW)
	if focus == rowHistoryClear {
		at = len(lines)
	}
	lines = append(lines, "  "+setupMark(o, focus == rowHistoryClear)+settingLabel(render.PadTo("History -", historyLabelW), focus == rowHistoryClear)+"  "+o.KeyCap("space")+" clear history")
	if d.cfg.HistoryUsage != nil {
		for _, l := range render.WrapText(d.cfg.HistoryUsage(), mapNoteW) {
			lines = append(lines, "    "+settingSupport(l))
		}
	}
	return lines, at
}

// historyApplyCmd writes the retention on close, when it moved (D-175).
func (d Dashboard) historyApplyCmd() tea.Cmd {
	return applyIfChanged(d.cfg.SetHistory, d.setup.history, d.cfg.History, nil)
}

// askClearHistory opens the ARE YOU SURE over Settings (D-177): clearing the
// history cannot be taken back.
func (d Dashboard) askClearHistory() (tea.Model, tea.Cmd) {
	if d.cfg.ClearHistory == nil {
		return d.settled(), nil
	}
	d.setup.confirmClear = true
	return d.settled(), nil
}

// historyClearedMsg is the app's answer to Clear history.
type historyClearedMsg struct{ err error }

// confirmClearHistory answers the ARE YOU SURE: enter clears, off the UI
// goroutine; esc cancels; every other key waits.
func (d Dashboard) confirmClearHistory(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		d.setup.confirmClear = false
		return d.settled(), nil
	case "enter":
		d.setup.confirmClear = false
		d.setup.note, d.setup.noteRow = "Clearing the history…", rowHistoryClear
		clear := d.cfg.ClearHistory
		return d.settled(), func() tea.Msg { return historyClearedMsg{err: clear()} }
	}
	return d, nil
}

// applyHistoryCleared says, beside the row, that it went.
func (d Dashboard) applyHistoryCleared(v historyClearedMsg) Dashboard {
	note := "History cleared: recording starts again at the next hour."
	if v.err != nil {
		note = "History partly cleared - " + v.err.Error()
	}
	d.setup.note, d.setup.noteRow = note, rowHistoryClear
	return d.settled()
}

// historyConfirmLines are the ARE YOU SURE's words, as ctrl+d's (D-152).
func (d Dashboard) historyConfirmLines(o render.Opts) []string {
	centre := func(s string) string { return confirmCentre(s, debugConfirmWidth) }
	out := []string{"", centre("ARE YOU SURE?"), centre("*** ONCE CONFIRMED, YOU CANNOT UNDO THIS ACTION ***"), ""}
	out = append(out, insetModalLines([]string{
		"Clear history removes everything watchpost has recorded - the hours a loop replays when a " +
			"source fails, and the days kept for trends. Recording starts again at the next hour.",
		""}, debugProseWidth(o, debugConfirmWidth))...)
	return append(out, strings.Repeat(" ", modalInset)+o.KeyCap("esc")+"  Cancel   "+o.KeyCap("enter")+" CLEAR HISTORY", "")
}
