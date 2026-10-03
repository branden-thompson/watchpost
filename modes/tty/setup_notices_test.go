package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// footerText is the Settings footer as drawn: the notice area, then the chips.
func footerText(d Dashboard) []string {
	_, _, foot, _ := d.footerModalChrome(d.opts())
	out := make([]string, 0, len(foot))
	for _, l := range foot {
		out = append(out, stripANSITest(l))
	}
	return out
}

// A SETTINGS NOTICE SITS AT THE BOTTOM, AND MOVES NOTHING (D-237): a note
// such as Rain Day 4+'s is in the notice area above the controls, not in its
// group - the groups draw the same with it or without it, and the area keeps
// its height, so neither the groups nor the controls move.
func TestASettingsNoticeSitsAtTheBottom(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowMapRainDetail)
	d.mapRainFull = false
	coarseBody, coarseFoot := settingsText(d), footerText(d)
	d.mapRainFull = true
	fullBody, fullFoot := settingsText(d), footerText(d)
	if strings.Replace(coarseBody, "Coarse", "Full  ", 1) != fullBody { // the picker's own value aside
		t.Errorf("the groups moved when the note appeared:\n%s\n---\n%s", coarseBody, fullBody)
	}
	if strings.Contains(strings.Join(coarseFoot, "\n"), "Rain Day 4+") {
		t.Error("Rain Day 4+'s notice shows while the coarse density is chosen")
	}
	if strings.Contains(fullBody, rainFullNote[:20]) {
		t.Error("Rain Day 4+'s note is still in its group")
	}
	if len(coarseFoot) != len(fullFoot) {
		t.Errorf("the footer is %d lines without the note and %d with it; want one height", len(coarseFoot), len(fullFoot))
	}
	row := ""
	for _, l := range fullFoot {
		if strings.Contains(l, "Rain Day 4+") {
			row = l
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(row), "! Rain Day 4+") || !strings.Contains(row, " - ") || !strings.Contains(row, "Full density") {
		t.Errorf("the notice reads %q; want \"! Rain Day 4+  - Full density ...\"", row)
	}
	chips := fullFoot[len(fullFoot)-1]
	if !strings.Contains(chips, "Next tab") {
		t.Errorf("the controls are not the footer's last line: %q", chips)
	}
}

// THE AREA HOLDS ITS HEIGHT WHILE THE CURSOR MOVES (D-237): UV cities' note
// shows only while its row is focused; moving onto it and off changes no
// height.
func TestTheNoticeAreaHoldsItsHeightAsTheCursorMoves(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowMapRainDetail)
	before, beforeBody := footerText(d), settingsText(d)
	if strings.Contains(strings.Join(before, "\n"), "UV cities") {
		t.Error("UV cities' note shows while another row is focused")
	}
	d.setup.focus = rowMapUVCities
	on := footerText(d)
	if len(on) != len(before) || len(strings.Split(settingsText(d), "\n")) != len(strings.Split(beforeBody, "\n")) {
		t.Errorf("moving onto UV cities changed the heights: footer %d → %d", len(before), len(on))
	}
	if !strings.Contains(strings.Join(on, "\n"), "UV cities") {
		t.Error("UV cities' note is not in the notice area while its row is focused")
	}
}

// EVERY TAB'S NOTES ARE NOTICES (D-237): the Data tab's history size and its
// clear's outcome, the Radio tab's voice note - each at the bottom, none in a
// group.
func TestEveryTabsNotesAreNotices(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowHistoryHours)
	d.cfg.HistoryUsage = func() string { return "Holds 12 MB, in ~/.local/share/watchpost" }
	if !strings.Contains(strings.Join(footerText(d), "\n"), "Holds 12 MB") || strings.Contains(settingsText(d), "Holds 12 MB") {
		t.Error("the history's size is not a notice, or is still in its group")
	}
	cleared := d.applyHistoryCleared(historyClearedMsg{})
	if !strings.Contains(strings.Join(footerText(cleared), "\n"), "Cleared: recording") {
		t.Error("the clear's outcome is not shown")
	}
	r := setupGolden(t, 133, 44, false, rowCastAlerts)
	r.cfg.VoiceInstalled = func(string) bool { return false }
	r.setup.cast.Names[roleOf(rowCastAlerts)] = "Daniel"
	if !strings.Contains(strings.Join(footerText(r), "\n"), "not installed") || strings.Contains(settingsText(r), "not installed") {
		t.Error("the voice note is not a notice, or is still in its group")
	}
}

// A NOTICE'S MARK SAYS ITS KIND (D-237): a cost to weigh in the advisory
// yellow, an outcome in the providers' green; the words the window's own.
func TestANoticesMarkSaysItsKind(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	warn := noticeRows(settingsNotice{label: "Rain Day 4+", text: "costs calls", tone: noticeWarn}, 90, 14)
	done := noticeRows(settingsNotice{label: "History", text: "cleared", tone: noticeDone}, 90, 14)
	if !strings.Contains(warn[0], render.Tint("! Rain Day 4+", render.Tok(render.AlertLabel))) {
		t.Errorf("a warning reads %q", warn[0])
	}
	if !strings.Contains(done[0], render.Tint("! History", render.Tok(render.ProviderOK))) {
		t.Errorf("an outcome reads %q", done[0])
	}
	long := noticeRows(settingsNotice{label: "Map data", text: strings.Repeat("a long sentence ", 12), tone: noticeInfo}, 90, 14)
	textAt := strings.Index(stripANSITest(long[0]), "a long")
	if second := stripANSITest(long[min(1, len(long)-1)]); len(long) < 2 || len(second)-len(strings.TrimLeft(second, " ")) != textAt {
		t.Errorf("a long notice does not wrap under its words: %q", long)
	}
}

// THE WINDOW KEEPS ONE HEIGHT AS NOTICES COME AND GO (D-237): the window is
// centred, so a footer that grew would move every row in it. Where the tab
// fits, the area's held room keeps the height; where it does not, the held
// room gives way to the body and the window is the screen's height either way.
func TestTheWindowKeepsOneHeightAsNoticesComeAndGo(t *testing.T) {
	for _, at := range []setupRowID{rowMapRainDetail, rowHistoryHours} { // a tab over the screen and one within it (P10-02)
		d := setupGolden(t, 133, 44, false, at)
		d.cfg.HistoryUsage = func() string { return "Holds 12 MB" }
		quiet := strings.Split(stripANSITest(d.renderModal(d.opts())), "\n")
		d.mapRainFull, d.mapTempNDFD = true, false
		d = d.applyHistoryCleared(historyClearedMsg{})
		loud := strings.Split(stripANSITest(d.renderModal(d.opts())), "\n")
		if len(quiet) != len(loud) {
			t.Errorf("%v: the window is %d rows without its notices and %d with them; want one height", at, len(quiet), len(loud))
		}
		if strings.TrimRight(quiet[1], "│ ▲") != strings.TrimRight(loud[1], "│ ▲") { // the scroll rail aside
			t.Errorf("%v: the tab row moved: %q → %q", at, quiet[1], loud[1])
		}
	}
	if got := fitFooter([]string{"", "", "n", "", "c"}, 2, 10, 14); len(got) != 4 || got[0] != "" || got[1] != "n" {
		t.Errorf("a footer one row over gives one held row: %q", got)
	}
	if got := fitFooter([]string{"", "", "n", "", "c"}, 2, 20, 14); len(got) != 3 || got[0] != "n" {
		t.Errorf("held room gives way no further than it holds: %q", got)
	}
}
