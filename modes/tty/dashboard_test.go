package tty

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// Spec: mock M-V1 + D-15 (layered keymap; only '?' locked) + D-19 (a/A/ctrl+a
// defaults; f/c live unit toggle) + R-12a. The dashboard reads ONLY the
// Snapshot (import lint). This is the B3 skeleton for the first D-21 UAT.

func f64(v float64) *float64 { return &v }

func snap() *snapshot.Snapshot {
	obs := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	return &snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion, GeneratedAt: obs,
		Locations: []snapshot.Location{{
			Label: "Oceanside, CA", Zip: "92057",
			Harmonized: snapshot.Conditions{Temp: f64(22.8), Condition: "clear",
				Source: snapshot.SourceInfo{Provider: "nws"}},
			Daily: []snapshot.Daily{
				{Date: "2026-08-24", TempMax: f64(23.9), TempMin: f64(17.2), Condition: "clear"},
				{Date: "2026-08-25", TempMax: f64(25.0), TempMin: f64(18.0), Condition: "rain"},
			},
			Alerts: []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe", Headline: "until Friday"}},
		}},
		Providers: []snapshot.ProviderStatus{{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: obs}}, // answered: counts ✔ (REVIEW C5)
	}
}

func dash(t *testing.T) tea.Model {
	t.Helper()
	m, err := NewDashboard(Config{Version: "0.1.0-test"})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44}) // 10-row recent window (chrome 32 + inset 2, UAT 46)
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	return model
}

func TestDashboardRendersMockAnatomy(t *testing.T) {
	v := dash(t).View().Content
	for _, want := range []string{
		"W A T C H P O S T",      // header per mock
		"API: ✔1",                // UAT 102: the masthead counts APIs (the per-provider strip lives in [S])
		"Oceanside, CA", "92057", // location row with zip
		"73ºF",                              // F default (D-19)
		"[severe]",                          // alert banner severity text (R-12a)
		"Watchpost Weather Radio",           // radio panel frame (UAT-3.3, static until B4)
		"R E C E N T   /   S E A R C H E D", // section band (UAT 43)
		"[?] Help",                          // footer line 2; '?' is the locked binding
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, v)
		}
	}
}

func TestUnitToggleIsLive(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if v := m.View().Content; !strings.Contains(v, "23ºC") {
		t.Fatalf("c must live-swap to Celsius:\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if v := m.View().Content; !strings.Contains(v, "73ºF") {
		t.Fatal("f must swap back to Fahrenheit")
	}
}

func TestQuitAndHelpBindings(t *testing.T) {
	m := dash(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q must quit (default binding, D-15-swappable)")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := m2.View().Content; !strings.Contains(v, "Watchpost Help") {
		t.Fatalf("? must open the help modal:\n%s", v)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m3.View().Content; strings.Contains(v, "Watchpost Help") {
		t.Fatal("esc must close the help modal")
	}
}

func TestNavigationMovesSelection(t *testing.T) {
	m := dash(t)
	// Add a second location so navigation has somewhere to go.
	s2 := snap()
	s2.Locations = append(s2.Locations, snapshot.Location{Label: "San Diego, CA", Zip: "92101",
		Harmonized: snapshot.Conditions{Temp: f64(25.0), Source: snapshot.SourceInfo{Provider: "nws"}}})
	m, _ = m.Update(SnapshotMsg{Snap: s2})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	d := m.(Dashboard)
	if d.selected != 1 {
		t.Fatalf("down must move selection, got %d", d.selected)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.(Dashboard).selected != 0 {
		t.Fatal("up must move selection back")
	}
}

func TestKeymapConflictRejectedAtBuild(t *testing.T) {
	// D-15: user overrides merge with validation; a config claiming '?' for
	// anything but help must fail construction, not silently win.
	_, err := NewDashboard(Config{Version: "t", KeyOverrides: term.KeyMap{
		"search": {Keys: []string{"?"}},
	}})
	if err == nil {
		t.Fatal("'?' override must be rejected (R-3)")
	}
}

func TestViewOpensWithTwoBlankLines(t *testing.T) {
	// UAT 10.3 (was UAT-3.1): two blank lines above the header; one after it.
	v := dash(t).View().Content
	lines := strings.Split(v, "\n")
	for i := range 2 {
		if strings.TrimSpace(lines[i]) != "" {
			t.Fatalf("line %d must be blank (top spacing):\n%s", i, v)
		}
	}
	if !strings.HasPrefix(strings.TrimLeft(lines[2], " "), "W A T C H P O S T") {
		t.Fatalf("header must follow the spacing: %q", lines[2])
	}
	if !strings.HasPrefix(lines[2], strings.Repeat(" ", viewPadLeft)+"W") {
		t.Fatalf("viewport must carry the %d-col left padding (UAT 14.3): %q", viewPadLeft, lines[2])
	}
	if strings.TrimSpace(lines[4]) != "" {
		t.Fatalf("blank line required between header and alert section: %q", lines[4])
	}
}

func TestAlertPagingKeys(t *testing.T) {
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = append(s2.Locations[0].Alerts,
		snapshot.Alert{Event: "Flash Flood Warning", Severity: "severe", Headline: "second"})
	m, _ = m.Update(SnapshotMsg{Snap: s2})
	if v := m.View().Content; !strings.Contains(v, "01 / 02 Alerts") {
		t.Fatalf("pager: %s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v := m.View().Content; !strings.Contains(v, "02 / 02 Alerts") || !strings.Contains(v, "FLASH FLOOD WARNING") {
		t.Fatalf("right must page to alert 2:\n%s", v)
	}
}

func TestStaleSnapshotShowsUpdatedStamp(t *testing.T) {
	v := dash(t).View().Content
	if !strings.Contains(v, "Updated:") {
		t.Fatalf("header requires the updated stamp (UAT 24.3 / 102 wording):\n%s", v)
	}
}

func TestRecentSectionSeedsWithRail(t *testing.T) {
	// UAT session 2A: the RECENT/SEARCHED table prepopulates (25 major US
	// cities fed by a slow-cadence pipeline), windows to 10 rows, numbers
	// continue after the priority rows, and the ▲│▼ rail frames the window.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 25 {
		rs.Locations = append(rs.Locations, snapshot.Location{
			Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1),
		})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 133, Height: 28}) // compact: 10-row window (UAT 58 budget)
	v := m.View().Content
	for _, want := range []string{
		"Showing 1-10 of 25 locations",
		"002. City 01", // numbering continues after the 1 priority row
		"011. City 10", // last visible window row
		"▲", "│", "▼",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("recent section missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "City 11") {
		t.Fatalf("rows beyond the 10-row window must not render:\n%s", v)
	}
}

func TestNavigationSpansBothTables(t *testing.T) {
	// UAT 4.4: the focus pointer walks priority rows, then recent rows,
	// auto-scrolling the recent window to keep the focus visible.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 25 {
		rs.Locations = append(rs.Locations, snapshot.Location{
			Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1),
		})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 133, Height: 28}) // 10-row window (UAT 58 budget)
	// 1 priority + 25 recent: step onto the first recent row.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	d := m.(Dashboard)
	if d.selected != 1 {
		t.Fatalf("focus must cross into the recent table, got %d", d.selected)
	}
	if v := m.View().Content; !strings.Contains(v, "› ") || !strings.Contains(v, "City 01") {
		t.Fatalf("pointer must render on the focused recent row:\n%s", v)
	}
	// Walk to recent row 12: the window must scroll to keep focus visible.
	for range 11 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	d = m.(Dashboard)
	if d.selected != 12 || d.recentOff != 2 {
		t.Fatalf("window must follow the focus: selected=%d off=%d", d.selected, d.recentOff)
	}
	if v := m.View().Content; !strings.Contains(v, "Showing 3-12 of 25 locations") {
		t.Fatalf("Showing must track the scrolled window:\n%s", v)
	}
	// And back up beyond the top of the window.
	for range 12 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if d := m.(Dashboard); d.selected != 0 {
		t.Fatalf("focus must walk back to the priority table, got %d", d.selected)
	}
}

func TestAlertAreaHeightIsFixed(t *testing.T) {
	// UAT 5.2: the alert area stays reserved-but-blank when the focused
	// location has no alert, so the layout below it never jumps.
	m := dash(t)
	withAlert := strings.Count(m.View().Content, "\n")
	s2 := snap()
	s2.Locations[0].Alerts = nil
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	withoutAlert := strings.Count(m2.View().Content, "\n")
	if withAlert != withoutAlert {
		t.Fatalf("alert area must reserve its space: %d lines with alert, %d without", withAlert, withoutAlert)
	}
	if v := m2.View().Content; strings.Contains(v, "EXTREME HEAT WATCH") {
		t.Fatal("blank reservation must not render alert text")
	}
}

func TestAlertTitleNamesTheLocation(t *testing.T) {
	// UAT 5.3: the module's inner text matches the focused location.
	v := dash(t).View().Content
	if !strings.Contains(v, "EXTREME HEAT WATCH · Oceanside, CA") {
		t.Fatalf("alert title must carry the focused location:\n%s", v)
	}
}

func TestModuleBlocksAlignWithTables(t *testing.T) {
	// UAT 7.1: alert + radio blocks sit +6 cols left / +4 right of the
	// content span, with bg-padded blank lines top and bottom.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	v := dash(t).View().Content
	var radioLines []string
	for _, l := range strings.Split(v, "\n") {
		if strings.HasPrefix(strings.TrimLeft(l, " "), "\x1b[38;5;250;49m") {
			radioLines = append(radioLines, l)
		}
	}
	if len(radioLines) != 4 { // title/vol/state, station/clock, play line, controls (viz off - UAT 54)
		t.Fatalf("hidden-bg radio module must be 4 flush content lines (UAT 19.1b/54), got %d", len(radioLines))
	}
	for _, l := range radioLines {
		if !strings.HasPrefix(l, strings.Repeat(" ", viewPadLeft)+"\x1b[") {
			t.Fatalf("block must start at the table left edge (UAT 12.3): %q", l)
		}
	}
	// UAT 7.2: green title + VOL fill, white timestamp, bold-grey STOPPED.
	if !strings.Contains(v, "\x1b[38;5;77mWatchpost Weather Radio") {
		t.Fatal("radio title must be green")
	}
	if strings.Contains(v, "00:00 / 00:00") { // UAT 89: the timeline placeholder is gone
		t.Fatal("timestamp must be white")
	}
	if !strings.Contains(v, "\x1b[1;38;5;245m■ STOPPED") {
		t.Fatal("stopped state must be bold grey")
	}
}

func TestHelpFloatsOverDashboard(t *testing.T) {
	// UAT 8.3: '?' composites the help panel over the dashboard instead of
	// replacing the view — the dashboard chrome stays visible around it.
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	v := m2.View().Content
	if !strings.Contains(v, "Watchpost Help") {
		t.Fatalf("help panel missing:\n%s", v)
	}
	if !strings.Contains(v, "W A T C H P O S T") || !strings.Contains(v, "Quit") {
		t.Fatalf("dashboard must stay visible beneath the floating help:\n%s", v)
	}
}

func TestEnterOpensFloatingForecastDetails(t *testing.T) {
	// UAT 10.6: enter floats the forecast window for the focused location.
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v := m2.View().Content
	// location-detail-mock.txt anatomy (UAT 27).
	for _, want := range []string{
		"Oceanside, CA 92057", "Updated:",
		"CURRENTLY │", "TODAY │", "FORECAST │",
		"HIGH", "LOW", "Scroll", "+ Watchlist", "− Watchlist",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("details modal missing %q:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "W A T C H P O S T") {
		t.Fatal("dashboard must stay visible beneath the details window")
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m3.View().Content; strings.Contains(v, "CURRENTLY │") {
		t.Fatal("esc must close the details window")
	}
}

func TestModalScrollRailOnShortTerminals(t *testing.T) {
	// UAT 10.4: on short terminals the modal windows its body and shows the
	// ▲│█▼ rail; up/down scroll the modal, not the tables.
	m := dash(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 133, Height: 19}) // budget: 19-12=7 lines
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	v := m.View().Content
	if !strings.Contains(v, "█") || !strings.Contains(v, "▲") {
		t.Fatalf("scroll rail must be visible on short terminals:\n%s", v)
	}
	before := m.(Dashboard).modalScroll
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if after := m.(Dashboard).modalScroll; after != before+1 {
		t.Fatalf("down must scroll the open modal, got %d -> %d", before, after)
	}
	if m.(Dashboard).selected != 0 {
		t.Fatal("table selection must not move while a modal is open")
	}
}

func TestWindowBackgroundTracksMode(t *testing.T) {
	// UAT 10.2: blue-grey window background, dark/light per terminal mode.
	m := dash(t)
	if bg := m.View().BackgroundColor; bg == nil {
		t.Fatal("window background must be set")
	}
}

func TestAlertBodyWrapsInFixedThreeLineArea(t *testing.T) {
	// UAT 15.2/15.2a: long alert bodies wrap (never truncate with …) inside
	// a constant 3-line body area, so the module height never bounces.
	m := dash(t)
	long := snap()
	long.Locations[0].Alerts = []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe",
		Headline: "dangerously hot conditions with temperatures up to 112 expected across the inland valleys through Friday evening"}}
	m2, _ := m.Update(SnapshotMsg{Snap: long})
	m2, _ = m2.Update(tea.WindowSizeMsg{Width: 100, Height: 60}) // narrow: forces the wrap
	v := m2.View().Content
	if strings.Contains(v, "…") {
		t.Fatalf("alert body must wrap, not truncate:\n%s", v)
	}
	if !strings.Contains(v, "inland valleys") {
		t.Fatalf("wrapped continuation must render:\n%s", v)
	}
	short, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	if a, b := strings.Count(short.View().Content, "\n"), strings.Count(v, "\n"); a != b {
		t.Fatalf("module height must not bounce between short (%d) and wrapped (%d) bodies", a, b)
	}
}

func TestTomorrowAndExtendedBandsTouch(t *testing.T) {
	// UAT 15.1: the gutter+spacer gap before EXTENDED is bridged too.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 170, Height: 60})
	band := strings.SplitN(wide.View().Content, "RECENT", 2)[0]
	for _, l := range strings.Split(band, "\n") {
		if strings.Contains(l, "48;2;66;122;122") { // tomorrow band present
			if regexp.MustCompile(`\x1b\[0m +\x1b\[`).MatchString(strings.TrimRight(l, " ")) {
				t.Fatalf("bands must touch across the spacer gap: %q", l)
			}
			return
		}
	}
	t.Fatal("tomorrow band not found on a wide terminal")
}

func TestCtrlAOpensAddLocationModal(t *testing.T) {
	// UAT 16.3: ctrl+a floats the Add Location modal; typing builds the
	// query (global bindings must not fire); esc cancels.
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	v := m.View().Content
	if !strings.Contains(v, "Add Location") || !strings.Contains(v, "Search:") {
		t.Fatalf("add-location modal missing:\n%s", v)
	}
	for _, r := range "qui" { // 'q' must NOT quit while typing
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if q := m.(Dashboard).addQuery; q != "qui" {
		t.Fatalf("typed query = %q, want 'qui'", q)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if q := m.(Dashboard).addQuery; q != "qu" {
		t.Fatalf("backspace: %q", q)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m.View().Content; strings.Contains(v, "Add Location") && strings.Contains(v, "Search:") {
		t.Fatal("esc must close the add modal")
	}
}

func TestAlertsSortMostSevereFirst(t *testing.T) {
	// UAT 16.2: index 0 is always the worst active alert - the module's
	// first page and the name tint agree (Los Angeles case).
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Heat Advisory", Severity: "minor", Headline: "advisory first in feed"},
		{Event: "Flash Flood Warning", Severity: "severe", Headline: "warning buried second"},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	if got := d.snap.Locations[0].Alerts[0].Event; got != "Flash Flood Warning" {
		t.Fatalf("most severe must sort first, got %q", got)
	}
	if v := m2.View().Content; !strings.Contains(v, "FLASH FLOOD WARNING") {
		t.Fatalf("module must open on the severe alert:\n%s", v)
	}
}

func TestPagingChipsMuteWhenInert(t *testing.T) {
	// UAT 21.1: a chip whose press would do nothing reads muted; state
	// flows from the model (ELM) through render.KeyCapIf.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	muted, full := "48;2;43;43;43", "48;2;86;86;86"

	one := dash(t) // single alert: both paging chips inert
	v := one.View().Content
	if got := strings.Count(v, muted); got != 2 {
		t.Fatalf("single-alert view must mute both paging chips, got %d muted", got)
	}
	two := snap()
	two.Locations[0].Alerts = append(two.Locations[0].Alerts,
		snapshot.Alert{Event: "Flash Flood Warning", Severity: "severe", Headline: "second"})
	m, _ := one.Update(SnapshotMsg{Snap: two})
	if v := m.View().Content; strings.Count(v, muted) != 1 || !strings.Contains(v, full) {
		// at index 0 of 2: ← inert, → live
		t.Fatalf("2-alert idx0 must mute only ←:\n%d muted", strings.Count(v, muted))
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v := m.View().Content; strings.Count(v, muted) != 1 {
		t.Fatalf("2-alert idx1 must mute only →: %d muted", strings.Count(v, muted))
	}
}

func TestAKeyOpensAlertDetailsModal(t *testing.T) {
	// UAT 22: [A] floats the full alert record — event + severity/urgency/
	// certainty, timing with duration, area, description, instruction —
	// on a severity-tinted tile.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	onset := time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC)
	ends := onset.Add(30 * time.Hour)
	s2.Locations[0].Alerts = []snapshot.Alert{{
		Event: "Extreme Heat Watch", Severity: "severe", Urgency: "expected", Certainty: "likely",
		Headline: "until Friday", Onset: &onset, Ends: &ends, Effective: onset, Expires: ends,
		AreaDesc:    "San Diego County Coastal Areas",
		Description: "Dangerously hot conditions with temperatures up to 112 possible.",
		Instruction: "Drink plenty of fluids and stay out of the sun.",
	}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	for _, want := range []string{
		"ALERT 1 / 1 · Oceanside, CA",
		"EXTREME HEAT WATCH", "[severe · expected · likely]",
		"Starts", "Ends", "(~30h0m0s)",
		"San Diego County", "Dangerously hot", "Instructions: Drink plenty",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("alert modal missing %q:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "48;2;40;0;0") {
		t.Fatal("warning-grade modal must carry the muted red tile")
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m3.View().Content; strings.Contains(v, "ALERT 1 / 1") {
		t.Fatal("esc must close the alert modal")
	}
}

func TestAlertModalPagesWithTonedTiles(t *testing.T) {
	// UAT 23: left/right page alerts inside the modal (no esc round-trip);
	// the tile re-tints to the FOCUSED alert's class per page.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Heat Advisory", Severity: "minor", Headline: "advisory"},
		{Event: "Flash Flood Warning", Severity: "severe", Headline: "warning"},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2}) // sorts: warning first
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	if !strings.Contains(v, "ALERT 1 / 2") || !strings.Contains(v, "48;2;40;0;0") {
		t.Fatalf("page 1 must be the warning on the red tile:\n%s", v)
	}
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	v = m2.View().Content
	if !strings.Contains(v, "ALERT 2 / 2") || !strings.Contains(v, "48;2;32;28;0") || !strings.Contains(v, "HEAT ADVISORY") {
		t.Fatalf("page 2 must be the advisory on the yellow tile:\n%s", v)
	}
	if !strings.Contains(v, "48;2;43;43;43") {
		t.Fatal("inert → chip must read muted on the last page")
	}
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v := m2.View().Content; !strings.Contains(v, "ALERT 1 / 2") {
		t.Fatal("left must page back without closing")
	}
}

func TestStatusModalAndControlPlacement(t *testing.T) {
	// UAT 24: [S] floats API diagnostics; [+/-] lives in the player line,
	// not the footer; header reads 'Last Updated:' with the [S] chip.
	m := dash(t)
	v := m.View().Content
	if !strings.Contains(v, "Updated:") || strings.Contains(v, "DATA LAST UPDATED") {
		t.Fatalf("header wording (UAT 24.3):\n%s", v)
	}
	head := strings.SplitN(v, "W A T C H P O S T", 2)[1]
	if first := strings.SplitN(head, "\n", 2)[0]; !strings.Contains(first, "Status") || !strings.Contains(first, "API: ") {
		t.Fatal("the title line carries the API summary and the [S] Status chip (UAT 24.2 / 102)")
	}
	if !strings.Contains(v, "VOL") || !strings.Contains(v, "[-]") || !strings.Contains(v, "[+]") {
		t.Fatal("volume control must render VOL [-]bar[+] in the player (UAT 41)")
	}
	if strings.Count(v, "Adjust Radio Volume") != 0 {
		t.Fatal("the [+/-] chip is gone (UAT 41)")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	sv := m2.View().Content
	for _, want := range []string{"API Status", "PROVIDERS", "NWS", "PIPELINES", "Priority", "Recent", "ISSUES"} {
		if !strings.Contains(sv, want) {
			t.Fatalf("status modal missing %q:\n%s", want, sv)
		}
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if strings.Contains(m3.View().Content, "PROVIDERS") {
		t.Fatal("esc must close the status modal")
	}
}

func TestStatusModalWrapsNeverTruncates(t *testing.T) {
	// UAT 25 (the recurring class, now fixed in the component): every modal
	// body line wraps within the tile — no … anywhere in the modal.
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	v := m2.View().Content
	if !strings.Contains(v, "multi-provider work") {
		t.Fatalf("long diagnostic line must survive by wrapping:\n%s", v)
	}
	if strings.Contains(v, "…") {
		t.Fatalf("modal content must never truncate:\n%s", v)
	}
}

// fakeHooks wires deterministic Resolve/Commit for flow tests (UAT 26).
type fakeHooks struct {
	resolved  snapshot.LocationRef
	committed [][2][]snapshot.LocationRef
}

func dashWithHooks(t *testing.T, h *fakeHooks) tea.Model {
	t.Helper()
	m, err := NewDashboard(Config{Version: "t",
		Resolve: func(q string) (snapshot.LocationRef, error) { return h.resolved, nil },
		Commit: func(w, r []snapshot.LocationRef) error {
			h.committed = append(h.committed, [2][]snapshot.LocationRef{w, r})
			return nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	return model
}

func drain(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		m, cmd = m.Update(msg)
	}
	return m
}

func TestAddFlowAppendsToWatchlist(t *testing.T) {
	// UAT 26.3: enter in the Add modal resolves and commits query -> bottom
	// of the watchlist.
	h := &fakeHooks{resolved: snapshot.LocationRef{Label: "Portland, ME", Zip: "04101", Lat: 43.6, Lon: -70.2, TZ: "America/New_York"}}
	m := dashWithHooks(t, h)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	for _, r := range "port" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 = drain(t, m2, cmd)
	if len(h.committed) != 1 {
		t.Fatalf("commit must fire once, got %d", len(h.committed))
	}
	watch := h.committed[0][0]
	if len(watch) != 2 || watch[1].Zip != "04101" {
		t.Fatalf("resolved location must append to the watchlist bottom: %+v", watch)
	}
	if m2.(Dashboard).showAdd {
		t.Fatal("add modal must close on success")
	}
}

func TestRemoveFlowConfirmAndCancel(t *testing.T) {
	// UAT 26.2: shift+del confirms on the #AE7D7E tile; enter removes and
	// moves the location to the top of recent; esc just closes.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	h := &fakeHooks{}
	m := dashWithHooks(t, h)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDelete, Mod: tea.ModShift})
	v := m.View().Content
	if !strings.Contains(v, "Remove Oceanside, CA from the watchlist?") || !strings.Contains(v, "48;2;79;12;12") {
		t.Fatalf("confirmation modal on the confirm tile missing:\n%s", v)
	}
	mCancel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if len(h.committed) != 0 || mCancel.(Dashboard).showRemove {
		t.Fatal("esc must cancel without committing")
	}
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = drain(t, m2, cmd)
	if len(h.committed) != 1 {
		t.Fatalf("confirm must commit once, got %d", len(h.committed))
	}
	watch, recent := h.committed[0][0], h.committed[0][1]
	if len(watch) != 0 {
		t.Fatalf("watchlist must drop the location: %+v", watch)
	}
	if len(recent) == 0 || recent[0].Zip != "92057" {
		t.Fatalf("removed location must top the recent list: %+v", recent)
	}
}

func TestLookupFlowTopsRecentAndOpensDetails(t *testing.T) {
	// UAT 26.4: [l] -> Lookup Location modal; enter tops the recent list and
	// opens the detail report for the looked-up location.
	h := &fakeHooks{resolved: snapshot.LocationRef{Label: "Boise, ID", Zip: "83702", Lat: 43.6, Lon: -116.2, TZ: "America/Boise"}}
	m := dashWithHooks(t, h)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if v := m.View().Content; !strings.Contains(v, "Lookup Location") || !strings.Contains(v, "Lookup") {
		t.Fatalf("lookup modal missing:\n%s", v)
	}
	for _, r := range "boise" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 = drain(t, m2, cmd)
	if len(h.committed) != 1 || len(h.committed[0][1]) == 0 || h.committed[0][1][0].Zip != "83702" {
		t.Fatalf("lookup must top the recent list: %+v", h.committed)
	}
	d := m2.(Dashboard)
	if !d.showDetails || d.selected != d.numPriority() {
		t.Fatalf("lookup must open details focused on the new recent top: details=%v sel=%d", d.showDetails, d.selected)
	}
}

func TestAddModalFullListMutesAndNotes(t *testing.T) {
	// UAT 26.3: at the 10-location cap the Add chip mutes and the modal
	// leads with the performance note.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	h := &fakeHooks{}
	m := dashWithHooks(t, h)
	full := snap()
	for i := range 9 {
		full.Locations = append(full.Locations, snapshot.Location{Label: fmt.Sprintf("City %d", i), Zip: fmt.Sprintf("100%02d", i)})
	}
	m, _ = m.Update(SnapshotMsg{Snap: full})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	v := m.View().Content
	if !strings.Contains(v, "Only 10 locations are allowed") {
		t.Fatalf("cap note missing:\n%s", v)
	}
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = drain(t, m2, cmd)
	if len(h.committed) != 0 {
		t.Fatal("enter must be inert at the cap")
	}
}

func TestAlertBulletFormattingRules(t *testing.T) {
	// location-detail-mock.txt bullet rules: prose indents 2 and bullets 4
	// from the text edge (flush with the section labels, UAT 65);
	// single-line bullets stack tight; multi-line bullets get one blank
	// line above and below.
	desc := "WHAT...Dangerous heat.\n\n* FIRST SHORT BULLET\n* SECOND SHORT BULLET\n* " +
		"A VERY LONG BULLET THAT WILL CERTAINLY WRAP ACROSS MULTIPLE LINES WHEN CONSTRAINED TO A NARROW WIDTH BUDGET FOR THE MODAL\n* TAIL SHORT BULLET"
	out := formatAlertBody(desc, 60)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "  WHAT...Dangerous heat.") {
		t.Fatalf("prose paragraph missing:\n%s", joined)
	}
	i1 := strings.Index(joined, "- FIRST")
	i2 := strings.Index(joined, "- SECOND")
	if strings.Count(joined[i1:i2], "\n") != 1 {
		t.Fatalf("adjacent single-line bullets must stack tight:\n%s", joined)
	}
	iLong := strings.Index(joined, "- A VERY LONG")
	if !strings.Contains(joined[:iLong], "- SECOND SHORT BULLET\n\n") {
		t.Fatalf("multi-line bullet needs a blank above:\n%s", joined)
	}
	if !strings.Contains(joined, "MODAL\n\n    - TAIL") {
		t.Fatalf("multi-line bullet needs a blank below:\n%s", joined)
	}
	for _, l := range out {
		if strings.HasPrefix(l, "    - ") || strings.HasPrefix(l, "      ") || strings.HasPrefix(l, "  ") || l == "" {
			continue
		}
		t.Fatalf("unexpected line shape: %q", l)
	}
}

func TestDetailCtrlAAddsViewedLocation(t *testing.T) {
	// UAT 27: ctrl+a inside the detail view adds the viewed location to the
	// watchlist (inert when already watched).
	h := &fakeHooks{resolved: snapshot.LocationRef{Label: "Boise, ID", Zip: "83702"}}
	m := dashWithHooks(t, h)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion,
		Locations: []snapshot.Location{{Label: "Boise, ID", Zip: "83702", TZ: "America/Boise"}, {Label: "Portland, ME", Zip: "04101"}}}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // focus the recent row
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	_ = drain(t, m2, cmd)
	if len(h.committed) != 1 {
		t.Fatalf("ctrl+a in details must commit once, got %d", len(h.committed))
	}
	watch, recent := h.committed[0][0], h.committed[0][1]
	if len(watch) != 2 || watch[1].Zip != "83702" {
		t.Fatalf("viewed location must append to the watchlist: %+v", watch)
	}
	// UAT 106: promoted, not copied — it leaves RECENT, the rest move up.
	if len(recent) != 1 || recent[0].Zip != "04101" {
		t.Fatalf("promoted location must leave the recent list: %+v", recent)
	}
	// The add modal's resolve path promotes the same way.
	h.committed = nil
	m5, cmd := m.Update(resolvedMsg{mode: "add", ref: snapshot.LocationRef{Label: "Boise, ID", Zip: "83702"}})
	_ = drain(t, m5, cmd)
	if len(h.committed) != 1 || len(h.committed[0][1]) != 1 || h.committed[0][1][0].Zip != "04101" {
		t.Fatalf("add via search must also drop the location from RECENT: %+v", h.committed)
	}
	// Already-watched: inert.
	m3, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp}) // focus Oceanside (watched)
	m3, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	_ = drain(t, m4, cmd)
	if len(h.committed) != 1 {
		t.Fatal("ctrl+a must be inert for an already-watched location")
	}
}

func TestDetailHiLoColumnAlignsAndPads(t *testing.T) {
	// UAT 28.1/28.2: TODAY's HIGH/LOW lands on the forecast rows' column;
	// temps occupy fixed 5-cell slots so 2- and 3-digit values align.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Daily = []snapshot.Daily{
		{Date: "2026-08-24", TempMax: f64(36.7), TempMin: f64(36.7), Condition: "clear"}, // 98ºF / 98ºF
		{Date: "2026-08-25", TempMax: f64(42.2), TempMin: f64(37.8), Condition: "rain"},  // 108ºF / 100ºF
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	var today, fc string
	for _, l := range d.detailLines() {
		if strings.Contains(l, "TODAY │") {
			today = l
		}
		if strings.Contains(l, "FORECAST │") {
			fc = l
		}
	}
	if !strings.Contains(today, "HIGH  98ºF /  98ºF LOW") || !strings.Contains(fc, "HIGH 108ºF / 100ºF LOW") {
		t.Fatalf("fixed 5-cell temp slots:\n%q\n%q", today, fc)
	}
	if strings.Index(today, "HIGH") != strings.Index(fc, "HIGH") {
		t.Fatalf("HIGH/LOW must share one column:\n%q\n%q", today, fc)
	}
}

func TestModalAlertTonesAndBoldTitles(t *testing.T) {
	// UAT 28.3-5: advisory #ACAE7D / warning #BE5454 text in the modals,
	// titles bold.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{
		{Event: "Flash Flood Warning", Severity: "severe", Description: "Warning prose."},
		{Event: "Heat Advisory", Severity: "minor", Description: "Advisory prose."},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v := m2.View().Content
	// The compositor may reorder params (bold before or after the color).
	warnTitle := regexp.MustCompile(`\x1b\[(1;)?38;2;190;84;84(;1)?m⚠ FLASH FLOOD WARNING`)
	advTitle := regexp.MustCompile(`\x1b\[(1;)?38;2;172;174;125(;1)?m⚠ HEAT ADVISORY`)
	if !warnTitle.MatchString(v) {
		t.Fatalf("warning title must be bold #BE5454:\n%s", v)
	}
	if !advTitle.MatchString(v) {
		t.Fatalf("advisory title must be bold #ACAE7D:\n%s", v)
	}
	if !strings.Contains(v, "\x1b[38;2;172;174;125m  Advisory prose.") {
		t.Fatalf("advisory body must carry the advisory tone:\n%s", v)
	}
}

func TestDetailMaritimeSectionForCoastalOnly(t *testing.T) {
	// B3 UAT 29: MARITIME rows render only when marine data exists — swell
	// heading + height in display units, sea state, buoy water temp.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Marine = &snapshot.Marine{
		SwellHeight: f64(0.6096), SwellDirDeg: f64(280), WaveHeight: f64(0.9), WavePeriod: f64(14),
		WaterTemp: f64(23.3), Buoy: "46224", BuoyDistanceKM: f64(9),
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	joined := strings.Join(d.detailLines(), "\n")
	// Grid: labels at 0, values at col 14, secondary values at col 37 (the
	// FORECAST HIGH/LOW column) — UAT 32.2.
	for _, want := range []string{
		"MARITIME │ Conditions    Slight Chop",
		"Water Temp    74ºF             (buoy 46224, 6 mi)", // 9 km in display units (UAT 60.2)
		"Swell         W        2.0 ft  (period 14 s)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("maritime section missing %q:\n%s", want, joined)
		}
	}
	inland := m.(Dashboard)
	inland.showDetails = true
	if strings.Contains(strings.Join(inland.detailLines(), "\n"), "MARITIME") {
		t.Fatal("inland location must not render a MARITIME section")
	}
}

func TestStatusAlignmentAndIssueAggregation(t *testing.T) {
	// UAT 31.1: fixed-width ages line up; warnings fold into issue classes.
	for dur, want := range map[time.Duration]string{
		59*time.Minute + 59*time.Second: "59m 59s",
		1*time.Minute + 5*time.Second:   " 1m  5s",
		55 * time.Second:                "    55s",
		2*time.Hour + 5*time.Minute:     " 2h 05m",
	} {
		if got := fixedAge(dur); got != want {
			t.Fatalf("fixedAge(%v) = %q, want %q", dur, got, want)
		}
	}
	sn := &snapshot.Snapshot{Warnings: []snapshot.Warning{
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "A", Message: "obs 2h old"},
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "B", Message: "obs 3h old"},
		{Code: snapshot.WarnObsStale, Provider: "nws", Location: "B", Message: "obs 3h old again"},
		{Code: snapshot.WarnProviderError, Provider: "ndbc", Message: "cannot reach ndbc"},
	}}
	out := strings.Join(aggregateWarnings(sn, nil), "\n")
	if !strings.HasPrefix(strings.TrimSpace(out), "✘ NDBC") {
		t.Fatalf("provider errors must sort first:\n%s", out)
	}
	if !strings.Contains(out, "obs_stale") || !strings.Contains(out, "×3 (2 locations)") {
		t.Fatalf("stale warnings must fold into one class with counts:\n%s", out)
	}
	if strings.Count(out, "obs_stale") != 1 {
		t.Fatalf("one row per issue class:\n%s", out)
	}
	if aggregateWarnings(nil, nil)[0] != "    none" {
		t.Fatal("no warnings must read none")
	}
}

func TestContentModalsStretchTo60Percent(t *testing.T) {
	// UAT 31.2: status/details/alerts widen to 60% of a wide terminal;
	// the base widths remain the floor on narrow ones.
	m := dash(t)
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	d := wide.(Dashboard)
	d.showStatus = true
	if got := d.modalWidth(); got != 120 {
		t.Fatalf("status modal at 200 cols must be 120 wide, got %d", got)
	}
	d.showStatus, d.showDetails = false, true
	if got := d.modalWidth(); got != 120 {
		t.Fatalf("details modal at 200 cols must be 120 wide, got %d", got)
	}
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	n := narrow.(Dashboard)
	n.showDetails = true
	if got := n.modalWidth(); got != 85 {
		t.Fatalf("details floor must hold on narrow terminals, got %d", got)
	}
	n.showDetails, n.showAdd = false, true
	if got := n.modalWidth(); got != 56 {
		t.Fatal("search/confirm modals stay fixed width")
	}
}

func TestDetailGridAlignsCurrentlyWithForecast(t *testing.T) {
	// UAT 32.1/32.5: CURRENTLY temps sit on the FORECAST condition column;
	// humidity sits on the HIGH/LOW column.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Harmonized.Feels = f64(24.4)
	s2.Locations[0].Harmonized.HumidityPct = f64(65)
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	var cur, feels, fc string
	for _, l := range d.detailLines() {
		p := stripANSITest(l)
		switch {
		case strings.Contains(p, "CURRENTLY │"):
			cur = p
		case strings.Contains(p, "Feels Like"):
			feels = p
		case strings.Contains(p, "FORECAST │"):
			fc = p
		}
	}
	content := func(l string) string { return strings.SplitN(l, "│ ", 2)[1] }
	col := func(l, tok string) int { // display column, not byte offset (º is 2 bytes)
		b := strings.Index(l, tok)
		if b < 0 {
			return -1
		}
		return len([]rune(l[:b]))
	}
	if col(content(cur), "73ºF") != colVal || col(content(fc), "RAIN") != colVal {
		t.Fatalf("temps must share the FORECAST condition column:\n%q\n%q", cur, fc)
	}
	if col(content(feels), "Humidity") != forecastHiLoCol || col(content(fc), "HIGH") != forecastHiLoCol {
		t.Fatalf("humidity must share the HIGH/LOW column:\n%q\n%q", feels, fc)
	}
}

// stripANSITest removes SGR sequences for column math in tests.
func stripANSITest(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

func TestDetailDividerEndsBeforeRail(t *testing.T) {
	// UAT 66 (supersedes 33.1): the divider runs to 3 cells before the
	// vertical scroll rail; alert text wraps to that same right edge.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{{Event: "Heat Advisory", Severity: "minor",
		Description: strings.Repeat("Dangerous heat across the valleys and foothills. ", 6) + "\n\n* " + strings.Repeat("BULLET TEXT THAT WRAPS ", 6)}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	inner := min(d.opts().Width, d.modalWidth()) - 7 // ScrollPanel content width; the rail sits at inner+1
	var div string
	longest := 0
	for _, l := range d.detailLines() {
		p := stripANSITest(l)
		if strings.HasPrefix(p, "───") {
			div = p
		}
		if strings.Contains(p, "Dangerous heat") || strings.Contains(p, "BULLET TEXT") {
			longest = max(longest, len([]rune(p)))
		}
	}
	if got := len([]rune(div)); got != inner-2 {
		t.Fatalf("divider must end 3 cells before the rail: %d cols, want %d", got, inner-2)
	}
	if longest > inner-2 || longest < inner-2-24 {
		t.Fatalf("alert text must use the width up to the divider's edge: longest %d, edge %d", longest, inner-2)
	}
}

func TestCompactModulesOnShortTerminals(t *testing.T) {
	// UAT 34: when the full layout cannot fit, the alert module collapses to
	// one line and the radio to two so the footer controls stay visible;
	// tall terminals keep the full modules.
	m := dash(t)
	short, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	v := short.View().Content
	if !short.(Dashboard).compact() {
		t.Fatal("24 rows must select compact mode")
	}
	if !strings.Contains(v, "01/01  ⚠ EXTREME HEAT WATCH · Oceanside, CA") {
		t.Fatalf("compact alert line missing:\n%s", v)
	}
	if !strings.Contains(v, "[space] Play") || !strings.Contains(v, "Repeat:") || strings.Contains(v, "00:00 / 00:00\n") {
		t.Fatalf("compact radio must be the two-row form:\n%s", v)
	}
	if !strings.Contains(v, "Quit") {
		t.Fatal("footer controls must remain visible on short terminals")
	}
	tall, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 60})
	if tall.(Dashboard).compact() {
		t.Fatal("60 rows must keep the full modules")
	}
}

func TestNarrowTerminalRowsFitAndModalsCenter(t *testing.T) {
	// UAT 35: on a narrow terminal every rendered row fits the width (the
	// compact radio row degrades: bar, clock, station), controls wrap, the
	// VOL bar shrinks, and modals center on the TERMINAL width.
	m := dash(t)
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})
	v := stripANSITest(narrow.View().Content)
	for i, l := range strings.Split(v, "\n") {
		if w := len([]rune(l)); w > 70 {
			t.Fatalf("row %d exceeds the terminal width (%d): %q", i, w, l)
		}
	}
	if !strings.Contains(v, "[space] Play") || !strings.Contains(v, "Size:") {
		t.Fatal("radio controls must survive by wrapping")
	}
	if !strings.Contains(v, "WWRadio") || strings.Contains(v, "Watchpost Weather Radio") {
		t.Fatal("narrow compact row must use the short title (UAT 36)")
	}
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	wv := stripANSITest(wide.View().Content)
	if strings.Count(v, "█")+strings.Count(v, "░") >= strings.Count(wv, "█")+strings.Count(wv, "░") {
		t.Fatal("VOL bar must shrink on narrow terminals")
	}
	sm, _ := narrow.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	for _, l := range strings.Split(stripANSITest(sm.View().Content), "\n") {
		if strings.Contains(l, "API Status") {
			// Measure the modal by its own box span: base rows can peek out
			// past the modal's right edge on narrow terminals.
			r := []rune(l)
			start, end := strings.IndexRune(l, '┌'), strings.IndexRune(l, '┐')
			startCol := len([]rune(l[:start]))
			modalW := len([]rune(l[:end])) - startCol + 1
			_ = r
			if want := (70 - modalW) / 2; startCol < want-1 || startCol > want+1 {
				t.Fatalf("modal must center on the terminal: starts at %d, want ~%d (modal %d wide)", startCol, want, modalW)
			}
			return
		}
	}
	t.Fatal("status modal title not found")
}

func TestRadioChipLabelsFollowState(t *testing.T) {
	// UAT 37: compact state-driven labels — [r] Repeat: Off|One|Watchlist,
	// [v] Viz: On|Off, [T] Size: Min|Max — and Size: Min renders the two-row
	// player even on a tall terminal. [p] Pin retired (UAT 93).
	m := dash(t)
	v := m.View().Content
	for _, want := range []string{"[space] Play", "Repeat: Off", "Mode: Synth", "Viz: Off", "Size: Max", "STOPPED"} {
		if !strings.Contains(v, want) {
			t.Fatalf("initial label %q missing:\n%s", want, v)
		}
	}
	if strings.Contains(v, "[p]") || strings.Contains(v, "Pin") {
		t.Fatalf("[p] Pin is retired (UAT 93):\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	v = m.View().Content
	for _, want := range []string{"[space] Pause", "PLAYING", "Repeat: One", "Viz: On"} {
		if !strings.Contains(v, want) {
			t.Fatalf("toggled label %q missing:\n%s", want, v)
		}
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	d := m.(Dashboard)
	if !strings.Contains(m.View().Content, "Size: Min") || !d.radioMin || len(d.radioLines(d.opts(), true)) < 2 {
		t.Fatalf("Size: Min must render the two-row player")
	}
}

func TestCompactRadioRowSpansModuleAndKeepsName(t *testing.T) {
	// UAT 40: the compact row always spans the module (tail right-aligned),
	// VOL floors at 10 cells, the clock drops before the location name, and
	// the name reads bold bright yellow.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})
	d := narrow.(Dashboard)
	o := d.opts()
	_, bg := render.RadioBlockTone()
	inner := o.ModuleInnerWidth(bg)
	row := d.radioLines(o, true)[0]
	plain := stripANSITest(row)
	if w := len([]rune(plain)); w != inner {
		t.Fatalf("compact row must span the module (%d), got %d: %q", inner, w, plain)
	}
	if !strings.HasSuffix(plain, "STOPPED") {
		t.Fatalf("tail must right-align: %q", plain)
	}
	if !strings.Contains(plain, "♪ Oceanside") || strings.Contains(plain, "00:00 / 00:00") {
		t.Fatalf("clock drops before the location name (name may shorten, never vanish): %q", plain)
	}
	if n := strings.Count(plain, "█") + strings.Count(plain, "░"); n != 10 {
		t.Fatalf("VOL must floor at 10 cells, got %d", n)
	}
	if !strings.Contains(row, "\x1b[1;38;5;226mOceanside") {
		t.Fatalf("station name must be bold bright yellow: %q", row)
	}
}

func TestVolumeControlStepsAndBlinks(t *testing.T) {
	// UAT 41: VOL [-]bar[+] 55; +/- step the level by 5 (bar cells step at
	// the 10s) and the pressed chip blinks green/red as acknowledgement.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	v := stripANSITest(m.View().Content)
	if !strings.Contains(v, "VOL  -") || !strings.Contains(v, " 55") {
		t.Fatalf("initial volume control missing:\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	raw := m.View().Content
	if m.(Dashboard).radioVolume != 60 || !strings.Contains(stripANSITest(raw), " 60") {
		t.Fatalf("plus must step to 60: %d", m.(Dashboard).radioVolume)
	}
	if !strings.Contains(raw, "\x1b[1;97;48;5;28m + ") {
		t.Fatal("[+] must blink green on press")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	if raw := m.View().Content; !strings.Contains(raw, "\x1b[1;97;48;5;124m - ") || m.(Dashboard).radioVolume != 55 {
		t.Fatal("[-] must blink red and step back to 55")
	}
	short, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	if n := strings.Count(stripANSITest(short.View().Content), "█") + strings.Count(stripANSITest(short.View().Content), "░"); n != 10 {
		t.Fatalf("compact VOL bar must be 10 cells, got %d", n)
	}
}

func TestVolumeLevelReservedAndEdgeChipsMute(t *testing.T) {
	// UAT 42: the level is a fixed 3-cell field; at 100 the [+] chip mutes,
	// at 0 the [-] chip mutes.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	if v := stripANSITest(m.View().Content); !strings.Contains(v, "+   55") {
		t.Fatalf("level must render in a 3-cell field ('+   55'):\n%s", v)
	}
	for range 12 { // 55 -> 100
		m, _ = m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	}
	d := m.(Dashboard)
	d.volFlash = "" // look past the blink
	if d.radioVolume != 100 || !strings.Contains(d.volControl(d.opts(), 10), "48;2;43;43;43m + ") {
		t.Fatalf("at 100 the [+] chip must mute (vol=%d)", d.radioVolume)
	}
	for range 25 { // 100 -> 0
		m, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	}
	d = m.(Dashboard)
	d.volFlash = ""
	if d.radioVolume != 0 || !strings.Contains(d.volControl(d.opts(), 10), "48;2;43;43;43m - ") {
		t.Fatalf("at 0 the [-] chip must mute (vol=%d)", d.radioVolume)
	}
	if !strings.Contains(stripANSITest(d.volControl(d.opts(), 10)), "+    0") {
		t.Fatal("0 must still occupy the 3-cell field")
	}
}

func TestWatchlistControlsAboveAndSectionBandTight(t *testing.T) {
	// UAT 43: controls sit above the watchlist group labels; the RECENT /
	// SEARCHED band replaces the dashed separator with no blank lines
	// around it.
	v := stripANSITest(dash(t).View().Content)
	lines := strings.Split(v, "\n")
	idx := func(sub string) int {
		for i, l := range lines {
			if strings.Contains(l, sub) {
				return i
			}
		}
		return -1
	}
	ctrl, group, band := idx("[ctrl+a] Add"), idx("L O C A T I O N"), idx("R E C E N T")
	if ctrl < 0 || group < 0 || band < 0 || ctrl > group {
		t.Fatalf("controls must sit above the group labels: ctrl=%d group=%d band=%d", ctrl, group, band)
	}
	if strings.TrimSpace(lines[band-1]) == "" {
		t.Fatalf("no blank line above the section band:\n%q\n%q", lines[band-1], lines[band])
	}
	// Below the band: the recent rows when there are any; this fixture has
	// none, so the UAT 104 fallback empty state stands there (its own blank).
	if !strings.Contains(strings.Join(lines[band+1:band+4], "\n"), "NO RECENT LOCATION SEARCHED") {
		t.Fatalf("empty recent list shows the fallback under the band:\n%q", lines[band+1:band+4])
	}
}

func TestRecentTableDropsItsGroupRow(t *testing.T) {
	// UAT 44.1: the section band connects the tables - only the watchlist
	// carries the [LOCATION][TODAY]... group row; the rail's ▲ rides the
	// recent column header.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 25 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	v := stripANSITest(m.View().Content)
	if n := strings.Count(v, "L O C A T I O N"); n != 1 {
		t.Fatalf("exactly one group-label row (the watchlist's), got %d", n)
	}
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "▲") && !strings.Contains(l, "R E C E N T") {
			t.Fatalf("▲ must ride the section band (UAT 45): %q", l)
		}
	}
	if strings.Count(v, "NAME") != 1 {
		t.Fatalf("only the watchlist carries the column header row, got %d", strings.Count(v, "NAME"))
	}
}

func TestRecentWindowExpandsOnTallTerminals(t *testing.T) {
	// UAT 46.1: tall terminals show more recent rows (all 25 here) while the
	// footer stays visible above a 2-row bottom inset; short ones floor at 3.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 25 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	tall, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 100})
	v := tall.View().Content
	if !strings.Contains(v, "Showing 1-25 of 25 locations") {
		t.Fatalf("tall terminal must expand the window to all rows:\n%s", v)
	}
	if lines := strings.Count(v, "\n") + 1; lines > 100-2 {
		t.Fatalf("view must respect the 2-row bottom inset: %d lines on 100 rows", lines)
	}
	if !strings.Contains(v, "[q] Quit") {
		t.Fatal("quit control must stay visible (header)")
	}
	mid, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 50})
	if md := mid.(Dashboard); md.compact() || !strings.Contains(mid.View().Content, "Showing 1-24 of 25 locations") {
		// UAT 49/57: 50 rows holds the FULL modules (radio 4 rows with viz
		// off, alert 7) with a 22-row window now that the footer is gone.
		t.Fatalf("50 rows: full modules with a 22-row window:\n%s", mid.View().Content)
	}
	exact, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 39})
	if !strings.Contains(exact.View().Content, "Showing 1-21 of 25 locations") {
		t.Fatalf("39 rows must yield a 21-row window (compact chrome 16 + inset 2):\n%s", exact.View().Content)
	}
}

func TestModulesHoldUntilTableBreakpoint(t *testing.T) {
	// UAT 49: with 50 recent rows the full modules stay while the table can
	// show >= 20 rows (fav + recent window); the table flexes row-by-row
	// above that; below it the modules minimize (and the table regains rows).
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 50 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	sawFull, sawCompact := false, false
	for h := 120; h >= 20; h-- {
		sized, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: h})
		d := sized.(Dashboard)
		rows := d.numPriority() + min(50, d.windowSize(d.opts()))
		if !d.compact() {
			sawFull = true
			if rows < tableBreakpoint {
				t.Fatalf("full modules must hold only while the table shows >= %d rows: %d rows at %d", tableBreakpoint, rows, h)
			}
		} else {
			sawCompact = true
		}
	}
	if !sawFull || !sawCompact {
		t.Fatalf("both modes must occur across the sweep (full=%v compact=%v)", sawFull, sawCompact)
	}
	// A large terminal shows the FULL modules with a big table.
	big, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 80})
	if big.(Dashboard).compact() {
		t.Fatal("80 rows must render the full modules")
	}
}

func TestRecentListCapsAtFifty(t *testing.T) {
	// UAT 48: 10 favourites + 50 most-recent = 60 tracked locations.
	refs := []snapshot.LocationRef{}
	for i := range 55 {
		refs = prependRef(refs, snapshot.LocationRef{Label: fmt.Sprintf("L%d", i), Zip: fmt.Sprintf("%05d", i)})
	}
	if len(refs) != recentCap || recentCap != 50 {
		t.Fatalf("recent list must cap at 50, got %d", len(refs))
	}
	if refs[0].Zip != "00054" {
		t.Fatal("newest stays on top")
	}
}

func TestVizChipTogglesVisualizerRows(t *testing.T) {
	// UAT 51: [v] shows/hides the max player's two visualizer rows; in the
	// min player it inserts one visualizer row between status and controls.
	m := dash(t)
	d := m.(Dashboard)
	o := d.opts()
	if n := len(d.radioLines(o, false)); n != 4 || strings.Contains(strings.Join(d.radioLines(o, false), ""), "VISUALIZER") {
		t.Fatalf("viz off: max player is 4 rows without visualizer rows, got %d", n)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	d = m.(Dashboard)
	full := d.radioLines(o, false)
	if len(full) != 7 {
		t.Fatalf("viz on: max player gains a 3-row visualizer frame, got %d: %q", len(full), full)
	}
	for _, row := range full[2:5] { // UAT 92: bracketed rows, blank while nothing plays
		if !strings.HasPrefix(row, "[") || !strings.HasSuffix(row, "]") || strings.TrimSpace(row[1:len(row)-1]) != "" || render.Width(row) != render.Width(full[0]) {
			t.Fatalf("visualizer row is a bracketed frame spanning the module: %q", row)
		}
	}
	mini := d.radioLines(o, true)
	if len(mini) != 3 || !strings.HasPrefix(mini[1], "[") || !strings.HasSuffix(mini[1], "]") {
		t.Fatalf("viz on: min player gets one visualizer row between status and controls: %q", mini)
	}
}

func TestVisualizerAnimatesWhilePlayingAndSettlesAfter(t *testing.T) {
	// UAT 92: with Viz on and the player playing, the dashboard pulls a
	// frame of band levels from the app on a fast tick and draws CLIAmp-style
	// bars in the visualizer rows; when playback stops the bars follow the
	// feed down to rest and the tick ends. Viz off = no tick at all.
	levels := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	fr := &fakeRadio{}
	m, err := NewDashboard(Config{Version: "t", Radio: fr, Spectrum: func() []float64 { return levels }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	model, cmd := model.Update(RadioStatusMsg{State: "playing", Station: "KEC49", Volume: 55})
	if cmd != nil {
		t.Fatal("Viz off: playing never starts the visualizer tick")
	}
	model, cmd = model.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if cmd == nil {
		t.Fatal("Viz on while playing arms the visualizer tick")
	}
	model, cmd = model.Update(vizTickMsg{})
	d := model.(Dashboard)
	rows := d.radioLines(d.opts(), false)[2:5]
	if !strings.Contains(stripANSITest(rows[0]), "█") || !strings.Contains(stripANSITest(rows[2]), "█") {
		t.Fatalf("full bands draw solid blocks on every visualizer row: %q", rows)
	}
	if cmd == nil {
		t.Fatal("the tick re-arms while playing")
	}
	// Stop: the feed decays; the tick keeps going until the bars rest.
	levels = []float64{0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	model, _ = model.Update(RadioStatusMsg{State: "stopped", Volume: 55})
	model, cmd = model.Update(vizTickMsg{})
	if cmd == nil || !strings.Contains(stripANSITest(model.(Dashboard).radioLines(d.opts(), false)[4]), "█") {
		t.Fatal("stopped with bars still up: one more frame, tick re-armed")
	}
	levels = make([]float64, 10)
	model, cmd = model.Update(vizTickMsg{})
	if cmd != nil {
		t.Fatal("bars at rest and nothing playing: the tick ends")
	}
	if row := model.(Dashboard).radioLines(d.opts(), false)[3]; strings.TrimSpace(row[1:len(row)-1]) != "" {
		t.Fatalf("at rest the rows are blank frames: %q", row)
	}
	// A second arm while already ticking never doubles the ticker.
	levels = []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	model, cmd = model.Update(RadioStatusMsg{State: "playing", Volume: 55})
	if cmd == nil {
		t.Fatal("playing again with Viz on re-arms")
	}
	if _, cmd = model.Update(RadioStatusMsg{State: "playing", Volume: 55}); cmd != nil {
		t.Fatal("already ticking: a status update does not add a second ticker")
	}
}

func TestOnStateLabelsEmphasized(t *testing.T) {
	// UAT 52: 'Repeat: On' reads yellow bold, 'Viz: On' green bold.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	v := m.View().Content
	if !strings.Contains(v, "Repeat: \x1b[1;38;5;220mOne") {
		t.Fatalf("Repeat: One must be yellow bold:\n%s", v)
	}
	if !strings.Contains(v, "Viz: \x1b[1;38;5;77mOn") {
		t.Fatalf("Viz: On must be green bold:\n%s", v)
	}
	if off := stripANSITest(dash(t).View().Content); !strings.Contains(off, "Repeat: Off") || !strings.Contains(off, "Viz: Off") {
		t.Fatal("Off states stay plain")
	}
}

func TestThemeChooserAppliesLiveAndFooterDropsTab(t *testing.T) {
	// UAT 53: [t] floats the chooser; ↓ + enter applies the theme live via
	// the app hook; esc closes. The footer no longer carries [tab].
	defer render.SetTheme(render.DefaultThemeName)
	applied := ""
	m, err := NewDashboard(Config{Version: "t", SetTheme: func(n string) error { applied = n; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if strings.Contains(model.View().Content, "Navigate Sections") {
		t.Fatal("[tab] chip must be gone (UAT 53.1)")
	}
	if !strings.Contains(model.View().Content, "[t] Theme") {
		t.Fatal("[t] Theme lives in the header now (UAT 57)")
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	v := model.View().Content
	if !strings.Contains(v, "Themes apply live") || !strings.Contains(v, "High Contrast") {
		t.Fatalf("theme chooser missing:\n%s", v)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if applied != render.ThemeNames()[1] {
		t.Fatalf("enter must apply the highlighted theme via the hook, got %q", applied)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if strings.Contains(model.View().Content, "Themes apply live") {
		t.Fatal("esc must close the chooser")
	}
}

func TestMaxPlayerLayoutPerMock(t *testing.T) {
	// UAT 54: row 1 = title … VOL + state; row 2 = station … clock; both
	// right-aligned to the module edge.
	m := dash(t)
	d := m.(Dashboard)
	o := d.opts()
	rows := d.radioLines(o, false)
	r1, r2 := stripANSITest(rows[0]), stripANSITest(rows[1])
	if !strings.HasPrefix(r1, "Watchpost Weather Radio") || !strings.HasSuffix(r1, "STOPPED") || !strings.Contains(r1, "VOL") {
		t.Fatalf("row 1: %q", r1)
	}
	if !strings.HasPrefix(r2, "♪ Oceanside") || strings.Contains(r2, "00:00 / 00:00") { // UAT 89: no placeholder clock
		t.Fatalf("row 2: %q", r2)
	}
}

func TestAlertModalBodyTextIsWhite(t *testing.T) {
	// UAT 55: in the [A] modal the body text is white; the title keeps its tone.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{{Event: "Heat Advisory", Severity: "minor", Headline: "h",
		AreaDesc: "Coastal Areas", Description: "Hot afternoon.", Instruction: "Hydrate."}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	v := m2.View().Content
	for _, want := range []string{"\x1b[97m  Area: Coastal Areas", "\x1b[97m  Hot afternoon.", "\x1b[97m  Instructions: Hydrate."} {
		if !strings.Contains(v, want) {
			t.Fatalf("body text must be white: missing %q\n%s", want, v)
		}
	}
	if !regexp.MustCompile(`\x1b\[(1;)?38;2;172;174;125(;1)?m`).MatchString(v) {
		t.Fatal("title keeps its advisory tone")
	}
}

func TestControlPlacementUAT56(t *testing.T) {
	// UAT 56: [enter] Location Details leads the watchlist controls, [↑↓]
	// Navigate is right-aligned on that row, [?] Help follows About, and the
	// footer keeps only theme + quit.
	v := stripANSITest(dash(t).View().Content)
	lines := strings.Split(v, "\n")
	var ctrl, head string
	for _, l := range lines {
		if strings.Contains(l, "[ctrl+a] Add") {
			ctrl = l
		}
		if strings.Contains(l, "[a] About") {
			head = l
		}
	}
	if !strings.Contains(ctrl, "[enter] Details") || strings.Index(ctrl, "[enter] Details") > strings.Index(ctrl, "[ctrl+a] Add") {
		t.Fatalf("[enter] Details must lead the control row: %q", ctrl)
	}
	if !strings.HasSuffix(strings.TrimRight(ctrl, " "), "[↑↓] Navigate") {
		t.Fatalf("[↑↓] Navigate must end the control row (right-aligned): %q", ctrl)
	}
	if !strings.Contains(head, "[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit") {
		t.Fatalf("header must carry Setup, About, Theme, Help, Quit in order (UAT 57 / 102): %q", head)
	}
	// UAT 57: no footer - the last content line is the recent section's
	// last row (the Showing line, or the empty state's ▼ row — UAT 104).
	trimmed := strings.TrimRight(v, "\n ")
	if last := trimmed[strings.LastIndex(trimmed, "\n")+1:]; !strings.Contains(last, "Showing") && !strings.HasSuffix(last, "▼") {
		t.Fatalf("view must end at the recent section's last row (no footer): %q", last)
	}
}

func TestViewFillsToBottomInset(t *testing.T) {
	// UAT 58: exactly 2 blank rows top and bottom - the content spans
	// height-2 rows with no stray trailing line.
	m := dash(t)
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}
	for i := range 50 {
		rs.Locations = append(rs.Locations, snapshot.Location{Label: fmt.Sprintf("City %02d", i+1), Zip: fmt.Sprintf("900%02d", i+1)})
	}
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	for _, h := range []int{40, 50, 70} {
		sized, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: h})
		v := sized.View().Content
		lines := strings.Split(v, "\n")
		if len(lines) != h-2 {
			t.Fatalf("h=%d: content must span %d rows, got %d", h, h-2, len(lines))
		}
		if strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[1]) != "" || strings.TrimSpace(lines[2]) == "" {
			t.Fatalf("h=%d: exactly 2 blank rows on top", h)
		}
		if strings.TrimSpace(lines[len(lines)-1]) == "" {
			t.Fatalf("h=%d: no stray blank row at the bottom of the content", h)
		}
	}
}

func TestDetailShowsStationAndDistance(t *testing.T) {
	// UAT 60.2: the observing station and its distance are always one
	// drill-in away, on the CURRENTLY grid (label / col 14 / col 37).
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Harmonized.Source = snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KCRQ", DistanceKm: f64(5.7)}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	joined := strings.Join(d.detailLines(), "\n")
	want := "Station   :   KCRQ" + strings.Repeat(" ", 19) + "Distance  :  4 mi"
	if !strings.Contains(joined, want) {
		t.Fatalf("CURRENTLY must carry %q:\n%s", want, joined)
	}
	loading := dash(t).(Dashboard)
	loading.showDetails = true
	if strings.Contains(strings.Join(loading.detailLines(), "\n"), "Station") {
		t.Fatal("no station row before an observation lands")
	}
}

func TestDetailTideAndCurrentRows(t *testing.T) {
	// B3 UAT 61-64: tides and currents on the MARITIME grid (value col 14 /
	// sub-column 8 / fixed 4-cell number / note at col 37 with HIGH) — trend from the next
	// predicted event, one row per next high and low, local hh:mm; a
	// negative low never shifts "ft"; currents in knots.
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC) // 15:00 PDT
	at := func(h, m int) time.Time { return time.Date(2026, 8, 24, h, m, 0, 0, time.UTC) }
	mar := &snapshot.Marine{
		TideLevel: f64(1.131), TideStation: "La Jolla (Scripps Institution Wharf)", TideStationKM: f64(38.8),
		Tides: []snapshot.TideEvent{
			{Time: at(16, 1), Height: 1.193, Type: "H"}, {Time: at(20, 35), Height: 0.799, Type: "L"},
			{Time: at(26, 40), Height: 1.731, Type: "H"}, {Time: at(33, 49), Height: -0.028, Type: "L"},
		},
		Currents: []snapshot.CurrentEvent{
			{Time: at(20, 9), Speed: 0.712, Type: "flood"}, {Time: at(23, 5), Speed: 0, Type: "slack"}, {Time: at(26, 30), Speed: 0.9, Type: "ebb"},
		},
		CurrentStation: "San Diego Bay Entrance",
	}
	o := render.Opts{Width: 120, Units: render.UnitF}
	rows := strings.Join(maritimeRows(o, mar, tz, now), "\n")
	for _, want := range []string{
		"Tide          Rising   3.7 ft  (La Jolla, 24 mi)",
		"Next High     19:40    5.7 ft",
		"Next Low      02:49   -0.1 ft",
		"Currents      Flood    1.4 kt  (Slack 16:05)",
	} {
		if !strings.Contains(rows, want) {
			t.Fatalf("maritime rows missing %q:\n%s", want, rows)
		}
	}
	// Falling when the next event is a low; slack phase names the next flow.
	falling := &snapshot.Marine{Tides: []snapshot.TideEvent{{Time: at(23, 30), Height: 0.5, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: at(21, 0), Speed: 0, Type: "slack"}, {Time: at(23, 30), Speed: 0.6, Type: "ebb"}}}
	rows = strings.Join(maritimeRows(o, falling, tz, now), "\n")
	if !strings.Contains(rows, "Tide          Falling") || !strings.Contains(rows, "Next Low      16:30    1.6 ft") || strings.Contains(rows, "Next High") || !strings.Contains(rows, "Currents      Slack            (Ebb 16:30)") {
		t.Fatalf("falling / slack rendering:\n%s", rows)
	}
	if got := strings.Join(maritimeRows(render.Opts{Units: render.UnitC}, mar, tz, now), "\n"); !strings.Contains(got, "Rising  1.13 m") || !strings.Contains(got, "1.4 kt") || !strings.Contains(got, "39 km") {
		t.Fatalf("metric tide rows:\n%s", got)
	}
	// No tide data: nothing added (coastal buoy-only sections unchanged).
	if got := maritimeRows(o, &snapshot.Marine{WaterTemp: f64(20)}, tz, now); len(got) != 1 {
		t.Fatalf("buoy-only section must not grow tide rows: %v", got)
	}
}

func TestTideHeightsShareOneColumn(t *testing.T) {
	// UAT 62/63: the height starts at the same cell on the Tide and Next
	// rows, for Rising and Falling, positive and negative values; and no
	// MARITIME row can exceed the details modal's 78-cell wrap budget.
	tz := time.UTC
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, tz)
	for _, tc := range []struct {
		level, next float64
		typ         string
	}{{1.13, 1.73, "H"}, {0.3, -0.03, "L"}} {
		m := &snapshot.Marine{TideLevel: f64(tc.level), Tides: []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: tc.next, Type: tc.typ}}}
		rows := maritimeRows(render.Opts{Units: render.UnitF}, m, tz, now)
		tide, next := stripANSITest(rows[0]), stripANSITest(rows[1])
		ti, ni := strings.Index(tide, " ft"), strings.Index(next, " ft")
		if ti < 0 || ti != ni {
			t.Fatalf("heights must align (%d vs %d):\n%s\n%s", ti, ni, tide, next)
		}
	}
	wide := &snapshot.Marine{WaterTemp: f64(20), Buoy: "46224", BuoyDistanceKM: f64(9), ObservedAt: now.Add(-40 * time.Minute),
		WaveHeight: f64(0.9), WavePeriod: f64(14), SwellDirDeg: f64(200), TideLevel: f64(1), TideStation: "La Jolla (Scripps Institution Wharf) with a very long name", TideStationKM: f64(38.8),
		Tides:    []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: 1.7, Type: "H"}, {Time: now.Add(7 * time.Hour), Height: -0.1, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: now.Add(-time.Hour), Speed: 0.7, Type: "flood"}, {Time: now.Add(time.Hour), Speed: 0, Type: "slack"}}}
	for _, r := range maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now) {
		if w := len([]rune(stripANSITest(r))); w > 78 {
			t.Fatalf("row exceeds the modal wrap budget (%d > 78): %q", w, r)
		}
	}
	// UAT 67: notes share one column, marNoteGap cells past the widest value.
	laid := maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now)
	widest, noteCol := 0, -1
	for _, r := range laid {
		p := stripANSITest(r)[detailPrefixW:]
		if v := strings.TrimRight(strings.SplitN(p, "(", 2)[0], " "); len([]rune(v)) > widest {
			widest = len([]rune(v))
		}
	}
	for _, r := range laid {
		p := stripANSITest(r)[detailPrefixW:]
		if i := strings.Index(p, "("); i >= 0 {
			col := len([]rune(p[:i]))
			if noteCol == -1 {
				noteCol = col
			}
			if col != noteCol || col != widest+marNoteGap {
				t.Fatalf("notes must share one column %d past the widest value (%d): got %d/%d in %q", marNoteGap, widest, col, noteCol, p)
			}
		}
	}
	if got := stripANSITest(strings.Join(maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now), "\n")); !strings.Contains(strings.Split(got, "\n")[0], "MARITIME │ Observed      40m") {
		t.Fatalf("mock order starts with Observed:\n%s", got)
	}
}

func TestAboutWindowMatchesMock(t *testing.T) {
	// UAT 68/70/75: [a] floats the About window (60 cols): centred title +
	// version, the app-owned credits list + licence notice and the build
	// stack inset 3 from the frame, maker lines centred; esc closes.
	m, err := NewDashboard(Config{Version: "0.1.0-test", Credits: []string{"NOAA National Weather Service (api.weather.gov)", "GeoNames.org cities & postal codes (CC BY 4.0)"}})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	m2, _ := model.Update(SnapshotMsg{Snap: snap()})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	d := m2.(Dashboard)
	if !d.showAbout {
		t.Fatal("[a] must open the About window")
	}
	v := stripANSITest(d.View().Content)
	var frame []string
	for _, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, "│"); i >= 0 && strings.Contains(l, "│") && strings.Count(l, "│") >= 2 {
			frame = append(frame, l[i:])
		}
	}
	body := strings.Join(frame, "\n")
	for _, want := range []string{
		"│                    W A T C H P O S T                     │",
		"│                       v 0.1.0-test                       │", // 12 chars centred on the 58-cell interior (the mock's {v 0.0.0-dev} is 13)
		"│   Data Provided by:                                      │",
		"│   NOAA National Weather Service (api.weather.gov)        │",
		"│   GeoNames.org cities & postal codes (CC BY 4.0)         │", // the app's list renders as given (UAT 75)
		"│   All sources free to use with attribution.              │",
		"│   Built with:                                            │",
		"│   STUDS - Stylized Terminal UI Design System             │",
		"│            Made with ♥ by Branden R. Thompson            │",
		"│                 github: branden-thompson                 │",
		"│             Make CLIs Great for Humans Again             │",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("About window missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "│   GO "+strings.TrimPrefix(runtime.Version(), "go")+" | BubbleTea | LipGloss |") {
		t.Fatalf("build line must carry the running Go version:\n%s", body)
	}
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "┌") && strings.Contains(l, "┐") && strings.Contains(l, "─ ") {
			t.Fatalf("About window has no title: %q", l)
		}
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m3.(Dashboard).showAbout {
		t.Fatal("esc must close the About window")
	}
}

func TestHelpModalControlsAreChips(t *testing.T) {
	// UAT 68.2: the help window's controls use the same KeyCap chips as
	// every other modal (not literal brackets).
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	d := m2.(Dashboard)
	if !d.showHelp {
		t.Fatal("? opens help")
	}
	o := d.opts()
	lines := d.helpLines(o)
	last := lines[len(lines)-1]
	if want := "  " + o.KeyCap("esc") + " Close   " + o.KeyCap("↑↓") + " Scroll"; last != want || strings.Contains(last, "[esc]") {
		t.Fatalf("help controls must be chips:\n got %q\nwant %q", last, want)
	}
	if !strings.Contains(last, "48;2;86;86;86") { // the chip background — colour is on
		t.Fatalf("chips must carry the KeyChip tone: %q", last)
	}
}

func TestDetailsOnRecentRowHydratesHourly(t *testing.T) {
	// UAT 72: the RECENT list skips the hourly forecast on its cadence;
	// opening Details on a recent row without one asks the app for it, once
	// per open; priority rows (which have their own hourly tier) never do.
	var got []string
	m, err := NewDashboard(Config{Version: "t", Hydrate: func(ref snapshot.LocationRef) { got = append(got, ref.Label) }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	rs := snap()
	rs.Locations[0].Label, rs.Locations[0].Lat = "Recent Town", 35.0
	rs.Locations[0].Hourly = nil
	model, _ = model.Update(RecentSnapshotMsg{Snap: rs})
	d := model.(Dashboard)
	d.selected = d.numPriority() // first recent row
	m2, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m2.(Dashboard).showDetails || cmd == nil {
		t.Fatal("details on a recent row without hourly must return the hydrate cmd")
	}
	cmd()
	if len(got) != 1 || got[0] != "Recent Town" {
		t.Fatalf("hydrate hook must be called for the recent row, got %v", got)
	}
	// A priority row never hydrates through the hook.
	d.selected = 0
	if _, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("priority rows have their own hourly tier")
	}
}

type fakeRadio struct {
	mu     sync.Mutex
	tuned  []string
	stops  int
	vol    int
	repeat RepeatMode
	queue  []snapshot.LocationRef
	mode   RadioMode
}

func (f *fakeRadio) Tune(ref snapshot.LocationRef) {
	f.mu.Lock()
	f.tuned = append(f.tuned, ref.Label)
	f.mu.Unlock()
}
func (f *fakeRadio) Stop()             { f.mu.Lock(); f.stops++; f.mu.Unlock() }
func (f *fakeRadio) SetVolume(pct int) { f.mu.Lock(); f.vol = pct; f.mu.Unlock() }
func (f *fakeRadio) SetRepeat(mode RepeatMode, queue []snapshot.LocationRef) {
	f.mu.Lock()
	f.repeat, f.queue = mode, queue
	f.mu.Unlock()
}
func (f *fakeRadio) SetMode(mode RadioMode) { f.mu.Lock(); f.mode = mode; f.mu.Unlock() }

func TestRadioModeChipTogglesSynthAndNearestRelay(t *testing.T) {
	// UAT 97: [m] Mode: Synth | Nearest Relay — the persisted mode seeds the
	// chip; each press flips it and pushes it to the player; help lists it.
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m.WithRadioMode(ParseRadioMode("relay"))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if v := stripANSITest(model.View().Content); !strings.Contains(v, "[m] Mode: Nearest Relay") {
		t.Fatalf("the persisted mode seeds the chip:\n%s", v)
	}
	model, cmd := model.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("[m] pushes the mode to the player")
	}
	cmd()
	if fr.mode != ModeSynth || !strings.Contains(stripANSITest(model.View().Content), "[m] Mode: Synth") {
		t.Fatalf("one press: Synth, got %v", fr.mode)
	}
	model, cmd = model.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	cmd()
	if fr.mode != ModeRelay {
		t.Fatalf("second press: Nearest Relay, got %v", fr.mode)
	}
	if ModeRelay.Key() != "relay" || ModeSynth.Key() != "synth" || ParseRadioMode("bogus") != ModeSynth {
		t.Fatal("persisted forms")
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := stripANSITest(model.View().Content); !strings.Contains(v, "Radio Mode") {
		t.Fatalf("help lists [m]:\n%s", v)
	}
}

func TestRepeatCyclesOffOneWatchlistAndTheRowFollowsTheDeck(t *testing.T) {
	// UAT 93: [r] cycles Off → One → Watchlist → Off; Watchlist hands the
	// player the favourites in order as its queue; when the player advances
	// it reports the location and the ▶ row follows; a watchlist change
	// under Watchlist mode re-sends the queue; Off clears the ∞.
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	two := snap()
	two.Locations = append(two.Locations, snapshot.Location{Label: "Carlsbad, CA", Zip: "92008", Lat: 33.16, Lon: -117.35})
	model, _ = model.Update(SnapshotMsg{Snap: two})
	press := func(r rune) {
		var cmd tea.Cmd
		model, cmd = model.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		if cmd != nil {
			cmd()
		}
	}
	press('r')
	press('r')
	if fr.repeat != RepeatWatchlist || !strings.Contains(stripANSITest(model.View().Content), "Repeat: Watchlist") {
		t.Fatalf("two presses: Watchlist, got %v", fr.repeat)
	}
	want := refsOf(two)
	if !sameRefs(fr.queue, want) || len(want) != 2 {
		t.Fatalf("Watchlist queue is the favourites in order: %v", fr.queue)
	}
	// The deck advances to the second favourite: the playing mark (∞ while
	// repeating, in the ▶ column) moves from row 001 to row 002.
	model, _ = model.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · " + want[1].Label, Location: snapshot.Key(want[1]), Volume: 55})
	rows := strings.Split(stripANSITest(model.View().Content), "\n")
	var first, second string
	for _, line := range rows {
		if strings.Contains(line, "001.") {
			first = string([]rune(line)[:12])
		}
		if strings.Contains(line, "002.") {
			second = string([]rune(line)[:12])
		}
	}
	if !strings.Contains(second, "∞") || strings.Contains(first, "∞") || strings.Contains(first, "▶") {
		t.Fatalf("the playing mark follows the deck to row 002: first %q second %q", first, second)
	}
	// A changed watchlist re-sends the queue while in Watchlist mode.
	fr.queue = nil
	grown := snap() // a fresh snapshot (the model holds `two` — never mutate what it sees)
	grown.Locations = append(append([]snapshot.Location(nil), two.Locations...), snapshot.Location{Label: "Julian, CA", Zip: "92036", Lat: 33.08, Lon: -116.60})
	var cmd tea.Cmd
	model, cmd = model.Update(SnapshotMsg{Snap: grown})
	if cmd == nil {
		t.Fatal("a watchlist change under Watchlist mode re-sends the queue")
	}
	cmd()
	if len(fr.queue) != len(want)+1 {
		t.Fatalf("queue follows the watchlist: %d", len(fr.queue))
	}
	if _, cmd = model.Update(SnapshotMsg{Snap: grown}); cmd != nil {
		t.Fatal("an unchanged watchlist sends nothing")
	}
	press('r')
	if fr.repeat != RepeatOff || !strings.Contains(stripANSITest(model.View().Content), "Repeat: Off") {
		t.Fatalf("third press: Off, got %v", fr.repeat)
	}
	if strings.Contains(stripANSITest(model.View().Content), "∞") {
		t.Fatal("Off: no row wears ∞")
	}
	// Help lists no [p] any more.
	model, _ = model.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := stripANSITest(model.View().Content); strings.Contains(v, "Pin") {
		t.Fatalf("help must not list the retired Pin control:\n%s", v)
	}
}

func TestFailSoftUXRedTeam09(t *testing.T) {
	// Red-team 0.9.0 F1/F4/F10: a failed radio shows its reason where the
	// station was; adding a location already on the watchlist is refused
	// with a reason; a commit that fails to save reopens the modal with the
	// error instead of vanishing.
	fr := &fakeRadio{}
	commits := 0
	commitErr := fmt.Errorf("cannot write config.toml: permission denied — check the file's permissions")
	m, _ := NewDashboard(Config{Version: "t", Radio: fr, Commit: func([]snapshot.LocationRef, []snapshot.LocationRef) error { commits++; return nil }})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	// F1: the reason on the station line, in both player sizes.
	model, _ = model.Update(RadioStatusMsg{State: "failed", Detail: "no voice for linux/arm: install Piper", Volume: 55})
	d := model.(Dashboard)
	for _, compact := range []bool{false, true} {
		if v := stripANSITest(strings.Join(d.radioLines(d.opts(), compact), "\n")); !strings.Contains(v, "✘ no voice for linux/arm: install Piper") {
			t.Fatalf("failed state shows the reason (compact=%v):\n%s", compact, v)
		}
	}
	// F4: a duplicate add is refused before it reaches the commit.
	model, _ = model.Update(resolvedMsg{mode: "add", ref: refsOf(snap())[0]})
	if d := model.(Dashboard); !strings.Contains(d.addErr, "already on the watchlist") || commits != 0 {
		t.Fatalf("duplicate add refused with a reason: %q, commits %d", d.addErr, commits)
	}
	// F10: a failed save comes back into view.
	model, _ = model.Update(committedMsg{err: commitErr, what: "remove"})
	if d := model.(Dashboard); !d.showAdd || !strings.Contains(d.addErr, "remove failed: cannot write") || d.addMode != "add" {
		t.Fatalf("a failed commit reopens the modal with the reason: showAdd=%v err=%q", d.showAdd, d.addErr)
	}
}

func TestRadioSpaceTunesFocusedLocationAndStatusDrivesLabels(t *testing.T) {
	// B4: [space] asks the app to tune the focused location; the player's
	// status message drives the state label and the station line; [space]
	// again stops; [+]/[-] push the volume.
	fr := &fakeRadio{}
	m, err := NewDashboard(Config{Version: "t", Radio: fr})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	m2, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("space must queue the tune command")
	}
	cmd()
	if len(fr.tuned) != 1 || fr.tuned[0] != "Oceanside, CA" {
		t.Fatalf("tune calls = %v", fr.tuned)
	}
	if v := stripANSITest(m2.View().Content); !strings.Contains(v, "CONNECTING") {
		t.Fatalf("state must read CONNECTING while the app resolves:\n%s", v)
	}
	m3, _ := m2.Update(RadioStatusMsg{State: "playing", Station: "KEC49 Monterey CA 162.550 MHz · 78 mi", Detail: "wxradio.org", Volume: 55, Live: true})
	v := stripANSITest(m3.View().Content)
	if !strings.Contains(v, "▶ PLAYING") || !strings.Contains(v, "♪ KEC49 Monterey CA 162.550 MHz") || !strings.Contains(v, "[space] Pause") && !strings.Contains(v, " space  Pause") {
		t.Fatalf("playing status must drive the label, station line and control:\n%s", v)
	}
	if !strings.Contains(v, "LIVE RADIO") || strings.Contains(v, "00:00 / 00:00") { // UAT 79: a relay has no timeline
		t.Fatalf("live relay must read LIVE RADIO instead of a timeline:\n%s", v)
	}
	// UAT 80: the playing location's row wears ▶ in the radio column; others do not.
	playingRows := 0
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "001.") && strings.Contains(line, "Oceanside, CA") {
			if r := []rune(line); len(r) > 5 && !strings.Contains(string(r[:8]), "▶") {
				t.Fatalf("playing row must show ▶ in the radio column: %q", string(r[:12]))
			}
			playingRows++
		} else if strings.Contains(line, "▶") && strings.Contains(line, "ºF") {
			t.Fatalf("only the playing location shows ▶: %q", line)
		}
	}
	if playingRows == 0 {
		t.Fatal("playing row not found")
	}
	synthM, _ := m2.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: "Tonight. Mostly clear.", Volume: 55})
	if sv := stripANSITest(synthM.View().Content); !strings.Contains(sv, "Tonight. Mostly clear.") || strings.Contains(sv, "LIVE RADIO") {
		t.Fatalf("synth shows the narration line:\n%s", sv)
	}
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	cmd()
	if fr.stops != 1 {
		t.Fatal("second space must stop")
	}
	_, cmd = m4.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	cmd()
	if fr.vol != 60 {
		t.Fatalf("volume must push to the player: %d", fr.vol)
	}
	m5, _ := m4.Update(RadioStatusMsg{State: "failed", Station: "KEC62 San Diego", Detail: "not relayed", Volume: 60})
	if v := stripANSITest(m5.View().Content); !strings.Contains(v, "NO STREAM") {
		t.Fatalf("failed status must read NO STREAM:\n%s", v)
	}
}

func TestMarqueeFollowsTheVoiceAndRepeatWiresThrough(t *testing.T) {
	// UAT 83: the marquee window tracks spoken progress; [r] pushes repeat
	// to the player and the playing row wears ∞.
	text := "This is a long narration line that will not fit the window and must scroll with the voice as it is spoken aloud."
	if got := marquee(text, 30, 0); got != text[:30] {
		t.Fatalf("start: %q", got)
	}
	mid := marquee(text, 30, 0.5)
	if mid == text[:30] || len([]rune(mid)) != 30 || !strings.Contains(text, mid) {
		t.Fatalf("midway the window must have moved: %q", mid)
	}
	if got := marquee(text, 30, 1); got != text[len(text)-30:] {
		t.Fatalf("end: %q", got)
	}
	if got := marquee("short", 30, 0.7); got != "short" {
		t.Fatal("short lines are static")
	}
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	m2, cmd := model.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("[r] must push repeat to the player")
	}
	cmd()
	if fr.repeat != RepeatOne {
		t.Fatal("one press: Repeat: One")
	}
	m3, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	cmd()
	m4, _ := m3.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: "Tonight. Mostly clear.", Volume: 55, Spoken: 3 * time.Second})
	v := stripANSITest(m4.View().Content)
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "001.") && strings.Contains(line, "Oceanside, CA") {
			if !strings.Contains(string([]rune(line)[:8]), "∞") {
				t.Fatalf("repeat on: the playing row wears ∞: %q", string([]rune(line)[:12]))
			}
		}
	}
}

func TestVoiceChooserUAT84(t *testing.T) {
	// [V] opens the correspondent chooser after [v] Viz; ↓ + enter applies
	// through the hook (persisted by the app) and the chip shows the name.
	var chosen, previewed string
	hookCalls := 0
	m, err := NewDashboard(Config{Version: "t", Voice: "Samantha",
		Voices:       func() []string { hookCalls++; return []string{"Samantha", "Alex", "Daniel"} },
		SetVoice:     func(name string) error { chosen = name; return nil },
		PreviewVoice: func(name string) { previewed = name }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	v := stripANSITest(model.View().Content)
	if i, j := strings.Index(v, "[v] Viz: Off"), strings.Index(v, "[V] Voice: Samantha"); i < 0 || j < 0 || j < i {
		t.Fatalf("[V] Voice: Samantha must follow [v] Viz:\n%s", v)
	}
	m2, _ := model.Update(tea.KeyPressMsg{Code: 'V', Text: "V", Mod: tea.ModShift})
	d := m2.(Dashboard)
	if !d.showVoice {
		t.Fatal("V opens the chooser")
	}
	if mv := stripANSITest(d.View().Content); !strings.Contains(mv, "Correspondent Voice") || !strings.Contains(mv, "✔ Samantha") || !strings.Contains(mv, "Daniel") || !strings.Contains(mv, "[p] Preview") || !strings.Contains(mv, "[enter] Select Voice") || !strings.Contains(mv, "[esc] Cancel") {
		t.Fatalf("chooser lists the voices with the current one marked and the chip controls:\n%s", mv)
	}
	for range 5 {
		_ = d.View() // UAT 85: rendering never calls the hook
	}
	if hookCalls != 1 {
		t.Fatalf("the voice hook must run once per open, got %d", hookCalls)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if mv := stripANSITest(m3.View().Content); !strings.Contains(mv, "› ") || !strings.Contains(mv, "›   Alex") && !strings.Contains(mv, "›  Alex") {
		t.Fatalf("↓ must move the pointer to Alex:\n%s", mv)
	}
	if _, cmd := m3.Update(tea.KeyPressMsg{Code: 'p', Text: "p"}); cmd == nil { // UAT 86
		t.Fatal("[p] must queue the preview")
	} else {
		cmd()
	}
	if previewed != "Alex" {
		t.Fatalf("preview must speak the highlighted voice, got %q", previewed)
	}
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter runs the voice hook off the update loop (red-team 0.9.0 C-5)")
	}
	cmd()
	d4 := m4.(Dashboard)
	if chosen != "Alex" || d4.radioVoice != "Alex" || d4.showVoice {
		t.Fatalf("enter applies the highlighted voice: chosen=%q chip=%q open=%v", chosen, d4.radioVoice, d4.showVoice)
	}
	if v := stripANSITest(d4.View().Content); !strings.Contains(v, "[V] Voice: Alex") {
		t.Fatalf("chip must show the new voice:\n%s", v)
	}
}

func TestMaxPlayerMarqueeFillsTheRowAfterTheLocation(t *testing.T) {
	// UAT 90: max player line 2 = full location name + 4 cells + the marquee
	// filling the rest of the row; the min player carries no marquee.
	m, _ := NewDashboard(Config{Version: "t", Radio: &fakeRadio{}})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	long := strings.Repeat("Tonight mostly clear with patchy fog overnight and lows near seventy. ", 4)
	model, _ = model.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: long, Volume: 55, Spoken: 20 * time.Second})
	d := model.(Dashboard)
	o := d.opts()
	_, bg := render.RadioBlockTone()
	inner := o.ModuleInnerWidth(bg)
	rows := d.radioLines(o, false)
	l2 := stripANSITest(rows[1])
	if !strings.HasPrefix(l2, "♪ Watchpost Synth · Oceanside, CA    Tonight mostly clear") {
		t.Fatalf("location, 4-cell gap, then the marquee: %q", l2)
	}
	if w := len([]rune(l2)); w != inner {
		t.Fatalf("the marquee must fill the row exactly (%d of %d)", w, inner)
	}
	min := stripANSITest(strings.Join(d.radioLines(o, true), "\n"))
	if strings.Contains(min, "Tonight mostly") {
		t.Fatalf("the min player has no marquee:\n%s", min)
	}
}

func TestDetailFireSectionAlwaysPresent(t *testing.T) {
	// B5 (HUM LEAD 2026-08-25): fire is another alert kind — a FIRE section
	// in the detail modal: hotspots nearest first with bearing, distance,
	// strength (bold at the threshold), satellite and age; named incidents
	// with acres and containment; the fire-weather alert when one is active.
	// With nothing burning it still says so.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	s2 := snap()
	loc := &s2.Locations[0]
	now := time.Now()
	loc.Fire = snapshot.FireState{
		AsOf: now,
		Hotspots: []snapshot.Hotspot{
			{Lat: loc.Lat + 0.09, Lon: loc.Lon, DetectedAt: now.Add(-2 * time.Hour), FRPMW: f64(62), DistanceKm: f64(10), Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "GOES-WEST"}},
			{Lat: loc.Lat, Lon: loc.Lon + 0.2, DetectedAt: now.Add(-40 * time.Minute), FRPMW: f64(8), DistanceKm: f64(18), Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "NOAA-20"}},
		},
		Incidents: []snapshot.Incident{{Name: "Timber", Acres: f64(12915), PercentContained: f64(26), Discovered: now.Add(-72 * time.Hour), Source: snapshot.SourceInfo{Provider: "wfigs", DistanceKm: f64(30)}}},
	}
	loc.Alerts = append(loc.Alerts, snapshot.Alert{ID: "rfw", Event: "Red Flag Warning", Severity: "severe", Expires: now.Add(6 * time.Hour)})
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	raw := strings.Join(d.detailLines(), "\n")
	joined := stripANSITest(raw)
	for _, want := range []string{
		"FIRE │ Hotspots      2 hotspots nearby",
		"◆ 6 mi N    62 MW · GOES-WEST",
		"2h 00m",
		"◆ 11 mi E   8 MW · NOAA-20",
		"Timber      19 mi · 12,915 ac",
		"26% contained",
		"Fire Wx       Red Flag Warning",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("FIRE section missing %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(raw, render.Tint("62 MW", "1;"+render.Tok(render.FireMark))) {
		t.Fatalf("62 MW must read bold at the 50 MW threshold:\n%q", raw)
	}
	if strings.Contains(raw, render.Tint("8 MW", "1;"+render.Tok(render.FireMark))) {
		t.Fatal("8 MW must not read bold")
	}
	cold := m.(Dashboard)
	cold.showDetails = true
	if q := stripANSITest(strings.Join(cold.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      fire feed not yet available") {
		t.Fatalf("before any fire feed answers it must say so, never 'none':\n%s", q)
	}
	s3 := snap()
	s3.Locations[0].Fire = snapshot.FireState{AsOf: now}
	m3, _ := m.Update(SnapshotMsg{Snap: s3})
	quiet := m3.(Dashboard)
	quiet.showDetails = true
	if q := stripANSITest(strings.Join(quiet.detailLines(), "\n")); !strings.Contains(q, "FIRE │ Hotspots      none within the fire ring") {
		t.Fatalf("no fire must still be said:\n%s", q)
	}
}

// setupHarness wires the Setup window's hooks to recorders (UAT 100).
type setupHarness struct {
	def    *snapshot.LocationRef
	key    string
	watch  []snapshot.LocationRef
	setups int
}

func (h *setupHarness) config() Config {
	return Config{
		Version: "0.1.0-test",
		Suggest: func(q string, limit int) []snapshot.LocationRef {
			if strings.HasPrefix(strings.ToLower(q), "oce") {
				return []snapshot.LocationRef{{Label: "Oceanside, CA", Tag: "OCEAN", Zip: "92057", Lat: 33.24, Lon: -117.29}}
			}
			return nil
		},
		Setup: func(def snapshot.LocationRef, key string) error {
			h.def, h.key, h.setups = &def, key, h.setups+1
			return nil
		},
		Commit: func(watch, recent []snapshot.LocationRef) error { h.watch = watch; return nil },
	}
}

func typeText(m tea.Model, text string) tea.Model {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestSetupWindowOpensOnSAndAtLaunch(t *testing.T) {
	// UAT 100: [s] Setup at any time; first run / `watchpost setup` opens it
	// at once, over the dashboard like every other modal.
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	d := m.(Dashboard)
	if !d.showSetup {
		t.Fatal("[s] must open the Setup window")
	}
	if view := stripANSITest(d.View().Content); !strings.Contains(view, "Setup") || !strings.Contains(view, "1. Your default location") {
		t.Fatalf("Setup window missing:\n%s", view)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.(Dashboard).showSetup {
		t.Fatal("esc closes it without saving")
	}
	first, err := NewDashboard(Config{Version: "t", OpenSetup: true})
	if err != nil || !first.showSetup {
		t.Fatalf("OpenSetup opens the window at launch (%v)", err)
	}
	if view := stripANSITest(first.View().Content); !strings.Contains(view, "1. Your default location") {
		t.Fatalf("an empty dashboard with the Setup window must render:\n%s", view)
	}
}

func TestHeaderMastheadCentresTheStampAndCountsAPIs(t *testing.T) {
	// UAT 102 mock: title left, stamp centred in the gap, API summary +
	// [S] right; chips on line 2 starting with [s] Setup; narrow widths
	// shorten the stamp before the row could overflow.
	m := dash(t)
	s2 := snap()
	s2.Providers = []snapshot.ProviderStatus{
		{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: time.Now()},
		{ID: "ndbc", Status: snapshot.ProviderDegraded, FetchedAt: time.Now()},
		{ID: "coops", Status: snapshot.ProviderDegraded},
		{ID: "firms", Status: snapshot.ProviderOff},
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	lines := strings.Split(stripANSITest(d.header(d.opts())), "\n")
	if !strings.Contains(lines[0], "API: ✔1 ⚠1 ✘1 /  3") {
		t.Fatalf("api summary (off excluded, total two columns): %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "[S] Status") || !strings.Contains(lines[0], "Updated: ") || render.Width(lines[0]) > 133 {
		t.Fatalf("title line: %q", lines[0])
	}
	i, j := strings.Index(lines[0], "Updated:"), strings.Index(lines[0], "API:")
	left := strings.Index(lines[0], "v0.1.0-test") + len("v0.1.0-test")
	if before, after := i-left, j-(i+len("Updated: 01/02/2006 15:04:05 MST")); before < after-3 || before > after+3 {
		t.Fatalf("stamp centred between title and API block: %d vs %d in %q", before, after, lines[0])
	}
	if !strings.HasPrefix(lines[1], "[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit") {
		t.Fatalf("chip line: %q", lines[1])
	}
	for _, w := range []int{100, 80, 60} {
		n, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: 44})
		nd := n.(Dashboard)
		for _, l := range strings.Split(stripANSITest(nd.header(nd.opts())), "\n") {
			if render.Width(l) > w {
				t.Fatalf("width %d: header line overflows (%d): %q", w, render.Width(l), l)
			}
		}
	}
}

func TestEmptyStatesStandWhereTheTablesWill(t *testing.T) {
	// UAT 104: no locations → a centred, wrapped message in place of the
	// watchlist table and of the recent rows; both give way to the tables
	// as soon as data lands; nothing exceeds a narrow width.
	m, _ := NewDashboard(Config{Version: "t"})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	v := stripANSITest(model.(Dashboard).View().Content)
	for _, want := range []string{
		"                                     Run 's' Setup, 'l'ookup a location, or 'ctrl+a' a searched",
		"                                                 location to add to your Watchlist",
		"NO RECENT LOCATION SEARCHED or DATA-SEEDING FAILED",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("empty state missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "###. NAME") || strings.Contains(v, "Showing 0-0") {
		t.Fatalf("empty state replaces the table and the Showing line:\n%s", v)
	}
	narrow, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 44})
	nd := narrow.(Dashboard)
	for _, l := range strings.Split(stripANSITest(nd.body(nd.opts())), "\n") { // the tables' area: the empty states wrap inside the width
		if render.Width(l) > 60 {
			t.Fatalf("line overflows 60 cols: %q", l)
		}
	}
	if nv := stripANSITest(narrow.(Dashboard).View().Content); !strings.Contains(nv, "searched location to add to your Watchlist") {
		t.Fatalf("narrow: the message wraps, never truncates:\n%s", nv)
	}
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if v := stripANSITest(model.(Dashboard).View().Content); strings.Contains(v, "to add to your Watchlist") || !strings.Contains(v, "###. NAME") {
		t.Fatalf("data replaces the watchlist empty state:\n%s", v)
	}
}

func TestSetupFormNoKeyIsTheDefaultDataSet(t *testing.T) {
	// UAT 100 / 111.3: one form, both questions visible; enter on question 1
	// takes the pick and moves the focus to question 2; enter there with an
	// empty key saves — the default data set, no key.
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()}) // an existing watchlist stays, below the new default
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	view := stripANSITest(model.(Dashboard).View().Content)
	for _, want := range []string{"› 1. Your default location", "  2. NASA FIRMS key (optional", "Key: ▌", "[tab] Next question", "[enter] Next"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the form shows every question at once, missing %q:\n%s", want, view)
		}
	}
	// UAT 111.4: the chip row wraps by chip inside the inset — no line
	// runs to the window's edge, no chip is split.
	d0 := model.(Dashboard)
	for _, l := range d0.setupLines(d0.opts()) {
		if w := render.Width(stripANSITest(l)); w > d0.modalWidth()-7 {
			t.Fatalf("setup line runs past the inset (%d): %q", w, stripANSITest(l))
		}
	}
	if !strings.Contains(view, "[ctrl+r] Reveal key") {
		t.Fatalf("a chip must never split across lines:\n%s", view)
	}
	model = typeText(model, "oce")
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "› Oceanside, CA (92057)") {
		t.Fatalf("type-ahead hint missing:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view = stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "Chosen: Oceanside, CA (92057)") || !strings.Contains(view, "› 2. NASA FIRMS key") || !strings.Contains(view, "[enter] Save") {
		t.Fatalf("enter accepts the pick and focuses question 2:\n%s", view)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on the last question saves")
	}
	model, _ = model.Update(cmd())
	d := model.(Dashboard)
	if d.showSetup || h.setups != 1 || h.key != "" || h.def == nil || h.def.Zip != "92057" {
		t.Fatalf("no key: Setup hook once with the location and an empty key: %+v key=%q open=%v", h.def, h.key, d.showSetup)
	}
	if len(h.watch) != 1 || h.watch[0].Zip != "92057" || d.selected != 0 { // the fixture's only favourite IS Oceanside: kept once, on top
		t.Fatalf("the default leads the committed watchlist, never duplicated, focus on it: %+v", h.watch)
	}
	// tab moves between the questions both ways; saving without a location
	// on a first run sends the focus back with the reason.
	first, _ := NewDashboard(h.config())
	var fm tea.Model = first
	fm, _ = fm.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !strings.Contains(stripANSITest(fm.(Dashboard).View().Content), "› 2. NASA FIRMS key") {
		t.Fatal("tab moves to question 2")
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if fd := fm.(Dashboard); fd.setup.focus != focusLocation || !strings.Contains(stripANSITest(fd.View().Content), "choose your default location first") {
		t.Fatalf("saving without a location goes back to question 1 with the reason:\n%s", stripANSITest(fd.View().Content))
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if fm.(Dashboard).setup.focus != focusKey {
		t.Fatal("shift+tab moves back the other way")
	}
}

func TestSetupFormKeyMasksAndStoresIt(t *testing.T) {
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	key := "0123456789abcdef0123456789abcdef"
	model = typeText(model, key)
	view := stripANSITest(model.(Dashboard).View().Content)
	if strings.Contains(view, key) || !strings.Contains(view, "Key: "+strings.Repeat("•", 32)) {
		t.Fatalf("the key is masked while typed:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "Key: "+key) {
		t.Fatalf("ctrl+r reveals it:\n%s", view)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(cmd())
	if model.(Dashboard).showSetup || h.key != key {
		t.Fatalf("enter stores the key: %q", h.key)
	}
	// A refused key (the hook's word) keeps the window open with the reason.
	h2 := &setupHarness{}
	cfg := h2.config()
	cfg.Setup = func(snapshot.LocationRef, string) error { return fmt.Errorf("a FIRMS MAP_KEY is 32 hex characters") }
	m2, _ := NewDashboard(cfg)
	model = m2
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(cmd())
	d := model.(Dashboard)
	if !d.showSetup || !strings.Contains(stripANSITest(d.View().Content), "setup failed: a FIRMS MAP_KEY is 32 hex") {
		t.Fatalf("a failed setup stays open with the reason:\n%s", stripANSITest(d.View().Content))
	}
}

func TestSetupFormShowsAStoredFIRMSKeyAndItsHealth(t *testing.T) {
	// UAT 111: on a re-run the form shows the current default and the stored
	// key's tail with how the provider is doing, from the first screen; a
	// bare enter keeps the default, an empty key keeps the stored key.
	h := &setupHarness{}
	cfg := h.config()
	cfg.FIRMSKey = func() string { return "cdef" }
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	s2 := snap()
	s2.Providers = append(s2.Providers, snapshot.ProviderStatus{ID: "firms", Status: snapshot.ProviderOK, FetchedAt: time.Now()})
	model, _ = model.Update(SnapshotMsg{Snap: s2})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	first := stripANSITest(model.(Dashboard).View().Content)
	for _, want := range []string{"Current: Oceanside, CA (92057)", "2. NASA FIRMS key: stored (…cdef) — ✔ working", "empty keeps it"} {
		if !strings.Contains(first, want) {
			t.Fatalf("re-run first screen missing %q:\n%s", want, first)
		}
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // keeps the default, focus to the key
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "Chosen: Oceanside, CA (92057)") || !strings.Contains(view, "› 2. NASA FIRMS key") {
		t.Fatalf("bare enter keeps the default:\n%s", view)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(cmd())
	if h.key != "" || h.setups != 1 || h.def == nil || h.def.Zip != "92057" {
		t.Fatalf("empty key keeps the stored one; the kept default is saved: %q %+v", h.key, h.def)
	}
	// A rejected key says so, in the window, before the user hunts through [S].
	s3 := snap()
	s3.Providers = append(s3.Providers, snapshot.ProviderStatus{ID: "firms", Status: snapshot.ProviderDegraded})
	s3.Warnings = []snapshot.Warning{{Code: "provider_error", Provider: "firms", Message: "firms: FIRMS rejected the MAP_KEY — open Setup ([s]) and paste it again"}}
	model, _ = model.Update(SnapshotMsg{Snap: s3})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "stored (…cdef) — ✘ rejected — replace it") {
		t.Fatalf("rejected key must be said in the window:\n%s", view)
	}
}

func TestVoiceChooserShowsTheWaitAndItsProgress(t *testing.T) {
	// UAT 119: p says at once what is about to happen; the deck's notes
	// (download progress, loading) replace it; "" clears; opening the
	// chooser starts clean.
	previewed := ""
	m, err := NewDashboard(Config{Version: "t", Voices: func() []string { return []string{"Lessac", "Amy"} }, PreviewVoice: func(n string) { previewed = n }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "… preparing Amy… (a first use downloads the voice") {
		t.Fatalf("the wait must be explained at once:\n%s", view)
	}
	_ = drain(t, model, cmd)
	if previewed != "Amy" {
		t.Fatalf("preview hook called with %q", previewed)
	}
	model, _ = model.Update(VoiceNoteMsg{Text: "installing Amy voice… 40% (25 MB)"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "… installing Amy voice… 40% (25 MB)") || strings.Contains(view, "preparing Amy") {
		t.Fatalf("the deck's progress replaces the first line:\n%s", view)
	}
	model, _ = model.Update(VoiceNoteMsg{Text: ""})
	if strings.Contains(stripANSITest(model.(Dashboard).View().Content), "… ") {
		t.Fatal("an empty note clears the line")
	}
	model, _ = model.Update(VoiceNoteMsg{Text: "loading Amy…"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	if strings.Contains(stripANSITest(model.(Dashboard).View().Content), "loading Amy") {
		t.Fatal("reopening the chooser starts clean")
	}
}
