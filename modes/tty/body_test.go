package tty

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

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
		"Showing 1-4 of 25 locations",
		"002. City 01", // numbering continues after the 1 priority row
		"005. City 04", // last visible window row (a 4-row window at 28 rows: the 0.12.0 ticker took two)
		"▲", "│", "▼",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("recent section missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "City 05") {
		t.Fatalf("rows beyond the 6-row window must not render:\n%s", v)
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
	if !strings.Contains(v, strings.TrimSuffix(render.Tint("Watchpost Weather Radio", render.Tok(render.RadioAccent)), "\x1b[0m")) { // the theme token, not a literal (L3-F24)
		t.Fatal("radio title must be green")
	}
	if strings.Contains(v, "00:00 / 00:00") { // UAT 89: the timeline placeholder is gone
		t.Fatal("timestamp must be white")
	}
	if !strings.Contains(v, strings.TrimSuffix(render.Tint("■ STOPPED", render.Tok(render.StateStopped)), "\x1b[0m")) {
		t.Fatal("stopped state must be bold grey")
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
	ctrl, group, band := idx("[ctrl+a] Favorite"), idx("L O C A T I O N"), idx("R E C E N T")
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
		if strings.Contains(l, "▲") && !strings.HasPrefix(strings.TrimSpace(l), "[") { // a band row: the band's bottom row since the three-row bands (UAT 45, nit 2026-08-27)
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
	if md := mid.(Dashboard); md.compact() || !strings.Contains(mid.View().Content, "Showing 1-20 of 25 locations") {
		// UAT 49/57: 50 rows holds the FULL modules (radio 4 rows with viz
		// off, alert 5 since the 2026-08-27 redesign) with a 26-row window
		// now that the footer is gone — every one of the 25 seeds shows.
		t.Fatalf("50 rows: full modules with a 20-row window (three-row bands; the 0.12.0 ticker took two):\n%s", mid.View().Content)
	}
	exact, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 39})
	if !strings.Contains(exact.View().Content, "Showing 1-15 of 25 locations") {
		t.Fatalf("39 rows must yield a 19-row window (compact chrome 18 + inset 2):\n%s", exact.View().Content)
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
	i := strings.Index(lines[0], "Updated:")
	end := i + strings.Index(lines[0][i:], "  ")                                                 // the stamp ends at the first double space (its own words are single-spaced)
	if centre, want := (i+end)/2, render.Width(lines[0])/2; centre < want-1 || centre > want+1 { // the row's global centre (UAT 2026-08-27)
		t.Fatalf("stamp centred on the row: %d vs %d in %q", centre, want, lines[0])
	}
	if !strings.HasPrefix(lines[1], "[s] Setup  [a] About  [t] Theme  [M] Mute Severe Alerts  [?] Help  [q] Quit") {
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
		"then 'ctrl+a' Favorite it",
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
	for _, l := range strings.Split(stripANSITest(nd.body(nd.layout())), "\n") { // the tables' area: the empty states wrap inside the width
		if render.Width(l) > 60 {
			t.Fatalf("line overflows 60 cols: %q", l)
		}
	}
	if nv := stripANSITest(narrow.(Dashboard).View().Content); !strings.Contains(nv, "to your Watchlist") {
		t.Fatalf("narrow: the message wraps, never truncates:\n%s", nv)
	}
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if v := stripANSITest(model.(Dashboard).View().Content); strings.Contains(v, "to your Watchlist") || !strings.Contains(v, "###. NAME") {
		t.Fatalf("data replaces the watchlist empty state:\n%s", v)
	}
}
