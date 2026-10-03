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
		d.setup.note, d.setup.noteRow, d.setup.noteTone = "Clearing the history…", rowHistoryClear, noticeInfo
		clear := d.cfg.ClearHistory
		return d.settled(), func() tea.Msg { return historyClearedMsg{err: clear()} }
	}
	return d, nil
}

// applyHistoryCleared says, beside the row, that it went.
func (d Dashboard) applyHistoryCleared(v historyClearedMsg) Dashboard {
	note, tone := historyClearedNote, noticeDone
	if v.err != nil {
		note, tone = "Partly cleared - "+v.err.Error(), noticeFail
	}
	d.setup.note, d.setup.noteRow, d.setup.noteTone = note, rowHistoryClear, tone
	return d.settled()
}

// historyClearedNote is what clearing the history says when it all went,
// under the History label.
const historyClearedNote = "Cleared: recording starts again at the next hour."

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

// historyRaise is a longer retention asked about: which preset, from what,
// to what (D-231).
type historyRaise struct {
	trends   bool
	from, to string
	cost     string // said once, when it is asked: the app walks the store to say it
}

// askIfLonger opens the question when a preset moved to a longer window - the
// presets run shortest first - and there is a cost to say; a shorter one is
// taken as it is.
func (d Dashboard) askIfLonger(trends bool, choices []historyChoice, from, to string) Dashboard {
	if d.cfg.HistoryCost == nil || historyAt(choices, to) <= historyAt(choices, from) {
		return d
	}
	shown := choices[historyAt(choices, from)].key // an unset preset is its default (D-175)
	d.setup.raise = &historyRaise{trends: trends, from: from, to: to, cost: d.cfg.HistoryCost(trends, shown, to)}
	return d
}

// confirmRaise answers the question: enter keeps the longer window, esc puts
// the shorter back; every other key waits.
func (d Dashboard) confirmRaise(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		if d.setup.raise.trends {
			d.setup.history.Trends = d.setup.raise.from
		} else {
			d.setup.history.Hours = d.setup.raise.from
		}
		d.setup.raise = nil
		return d.settled(), nil
	case "enter":
		d.setup.raise = nil
		return d.settled(), nil
	}
	return d, nil
}

// raiseLines are the question's words: what changes, what it costs, and that
// what was not recorded cannot be fetched back (D-231).
func (d Dashboard) raiseLines(o render.Opts) []string {
	r := d.setup.raise
	what, choices := "Hourly detail", historyHourChoices
	if r.trends {
		what, choices = "Trends", historyTrendChoices
	}
	from, to := historyLabel(choices, r.from), historyLabel(choices, r.to)
	centre := func(s string) string { return confirmCentre(s, debugConfirmWidth) }
	out := []string{"", centre("KEEP MORE HISTORY?"), ""}
	out = append(out, insetModalLines([]string{
		what + ": " + from + " → " + to + ".",
		r.cost,
		"What was not recorded cannot be fetched back: the longer window fills from now on, while watchpost runs.",
		""}, debugProseWidth(o, debugConfirmWidth))...)
	return append(out, strings.Repeat(" ", modalInset)+o.KeyCap("esc")+"  Keep "+from+"   "+o.KeyCap("enter")+" KEEP "+strings.ToUpper(to), "")
}

// historyLabel is a preset's words.
func historyLabel(choices []historyChoice, key string) string {
	return choices[historyAt(choices, key)].label
}
