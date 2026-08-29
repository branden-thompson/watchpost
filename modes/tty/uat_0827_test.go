package tty

// HUM LEAD UAT 2026-08-27, items 3a–3c: the Updated stamp's age and colour,
// and the alert module's header-row geometry.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestUpdatedStampCarriesItsAgeAndColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.ResetColorEnabledForTest()
	d := dash(t).(Dashboard)
	at := dataAsOf(d.snap)
	rows := []struct {
		age  time.Duration
		want string
		tone render.Token
	}{
		{20 * time.Second, "(Just Now)", render.ProviderOK},
		{time.Minute + 5*time.Second, "(1 Minute Ago)", render.ProviderOK},
		{2*time.Minute + 30*time.Second, "(2 Minutes Ago)", render.ProviderOK},
		{6 * time.Minute, "(6 Minutes Ago)", render.AlertLabel}, // stale: no successful fetch for > 5 min
		{3 * time.Hour, "(3 Hours Ago)", render.AlertLabel},
	}
	for _, row := range rows {
		d.now = func() time.Time { return at.Add(row.age) }
		line := strings.SplitN(d.header(d.opts()), "\n", 2)[0]
		want := strings.TrimSuffix(render.Tint("Updated: "+at.Local().Format("01/02/2006 15:04:05 MST")+" "+row.want, render.Tok(row.tone)), "\x1b[0m") // the opening tone (the box re-arms its own after)
		if !strings.Contains(line, want) {
			t.Errorf("age %v: want %q in %q", row.age, want, stripANSITest(line))
		}
	}
	// Before the first data: grey.
	d.snap = nil
	if line := strings.SplitN(d.header(d.opts()), "\n", 2)[0]; !strings.Contains(line, strings.TrimSuffix(render.Tint("awaiting first data...", render.Tok(render.TextBase)), "\x1b[0m")) {
		t.Errorf("no data: grey placeholder: %q", stripANSITest(line))
	}
	// A narrow terminal drops the age before it drops the date (the ladder;
	// the rule carries more than the old centred gap did — 80 cols is where
	// the age gives).
	d.snap = snap()
	d.now = func() time.Time { return at.Add(2 * time.Minute) }
	nd, _ := d.Update(tea.WindowSizeMsg{Width: 80, Height: 44})
	if line := stripANSITest(strings.SplitN(nd.(Dashboard).header(nd.(Dashboard).opts()), "\n", 2)[0]); strings.Contains(line, "Ago") || !strings.Contains(line, "Updated:") {
		t.Errorf("narrow: the plain stamp, no age: %q", line)
	}
}

// HUM LEAD UAT 2026-08-28 facelift: the alert module is ONE row in a heavy
// box on the Alert Details modal's tint — count, glyph, EVENT - place,
// issued, expires; the chips right-aligned. The body lives behind [A].
func TestAlertModuleIsOneBoxedRow(t *testing.T) {
	d := dash(t).(Dashboard)
	sent := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 12, 31, 20, 59, 0, 0, time.UTC)
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: "Air Quality Alert", Severity: "minor", Headline: "until Friday", Sent: sent, Expires: exp}, {Event: "Tornado Warning", Severity: "extreme", Headline: "h"}}
	wide, _ := d.Update(tea.WindowSizeMsg{Width: 160, Height: 44}) // the mock's width: every stamp fits
	d = wide.(Dashboard)
	fl := d.layout()
	if fl.compact {
		t.Fatal("the fixture is the full layout")
	}
	raw := d.alertArea(fl)
	lines := strings.Split(stripANSITest(raw), "\n")
	if len(lines) != render.BoxHeight(1) || fl.alertH != render.BoxHeight(1) || !strings.HasPrefix(lines[0], "┏") || !strings.HasPrefix(lines[2], "┗") {
		t.Fatalf("a one-row heavy box (%d rows): %q", fl.alertH, lines)
	}
	row := lines[1]
	inner := strings.TrimSpace(strings.Trim(row, "┃"))
	if !strings.HasPrefix(inner, "01/02  ⚠ AIR QUALITY ALERT - Oceanside, CA  [minor]  • Issued: 08/26 ") || !strings.Contains(inner, " • Expires: 12/31 ") || !strings.HasSuffix(inner, "Next") { // colour off here: the class in text (B-04)
		t.Fatalf("count · glyph · EVENT - place · issued · expires … chips: %q", inner)
	}
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	place := "\x1b[" + render.Tok(render.ModalTitle) + "mOceanside, CA"
	if raw := d.alertArea(fl); !strings.Contains(raw, render.Tok(render.AlertModalAdvBG)) || strings.Contains(raw, render.Tok(render.AlertModalWarnBG)) ||
		!strings.Contains(raw, "\x1b[1;"+render.Tok(render.AlertModalAdvFG)+"m⚠ AIR QUALITY ALERT") || !strings.Contains(raw, place) {
		t.Fatalf("an advisory sits on the modal's advisory tint, its event bold in the modal's advisory tone, the place bold white: %q", raw)
	}
	d.alertIdx = 1
	if raw := d.alertArea(fl); !strings.Contains(raw, render.Tok(render.AlertModalWarnBG)) || !strings.Contains(stripANSITest(raw), "02/02  ⚠ TORNADO WARNING - Oceanside, CA") ||
		!strings.Contains(raw, "\x1b[1;"+render.Tok(render.AlertModalWarnFG)+"m⚠ TORNADO WARNING") || !strings.Contains(raw, place) {
		t.Fatalf("a warning sits on the modal's warning tint, its event bold in the modal's warning tone: %q", stripANSITest(raw))
	}
	rendering.SetColorEnabledForTest(false)
	// Compact: the same tinted row without the rules (UAT 34).
	if short, _ := d.Update(tea.WindowSizeMsg{Width: 160, Height: 24}); short.(Dashboard).layout().alertH != 1 || !strings.Contains(stripANSITest(short.View().Content), "02/02  ⚠ TORNADO WARNING - Oceanside, CA") {
		t.Fatal("compact keeps the one-row form")
	}
	// Narrow: expires, then issued, then the chip labels give before the
	// title shortens — the row never exceeds the width.
	d.alertIdx = 0
	for _, w := range []int{100, 84, 60} {
		nd, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: 44})
		nfl := nd.(Dashboard).layout()
		row := strings.Split(stripANSITest(nd.(Dashboard).alertArea(nfl)), "\n")[1]
		if render.Width(row) != nfl.o.Width || !strings.Contains(row, "AIR QUALITY") && !strings.Contains(row, "…") {
			t.Fatalf("width %d: the row spans the frame exactly (%d), the title last to give: %q", w, render.Width(row), row)
		}
	}
	// Colour off (and --ascii): the class is text, so a warning and an
	// advisory never read the same (R-12a; round 4 B-04).
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe", Headline: "h"}}
	warn := stripANSITest(d.alertArea(d.layout()))
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "minor", Headline: "h"}}
	adv := stripANSITest(d.alertArea(d.layout()))
	if warn == adv || !strings.Contains(warn, "[severe]") || !strings.Contains(adv, "[minor]") {
		t.Fatalf("colour off: the class in text:\n%s\n%s", warn, adv)
	}
	// No alert: the box stands, muted, so the layout never jumps.
	d.snap.Locations[0].Alerts = nil
	if got := strings.Split(stripANSITest(d.alertArea(d.layout())), "\n"); len(got) != render.BoxHeight(1) || !strings.Contains(got[1], "No active alerts · Oceanside, CA") {
		t.Fatalf("the empty module: %q", got)
	}
}

// HUM LEAD UAT 2026-08-27: the bands are the last thing to give on a short
// terminal — three rows through compact mode, one row only when the
// RECENT window would otherwise fall below its floor.
func TestBandsCollapseLastOnShortTerminals(t *testing.T) {
	m := dash(t)
	rs := snap()
	for i := range 25 {
		loc := snap().Locations[0]
		loc.Label, loc.Lat = fmt.Sprintf("City %02d", i), 30+float64(i)/10
		rs.Locations = append(rs.Locations, loc)
	}
	rs.Locations = rs.Locations[1:]
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	tall, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	if fl := tall.(Dashboard).layout(); fl.compact || fl.o.ThinBands || fl.o.BandHeight() != 3 {
		t.Fatalf("tall: full modules, thick bands: compact=%v thin=%v", fl.compact, fl.o.ThinBands)
	}
	short, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 30})
	if fl := short.(Dashboard).layout(); !fl.compact || fl.o.ThinBands || fl.window < recentWindow {
		t.Fatalf("short: compact modules, bands still thick, window %d: compact=%v thin=%v", fl.window, fl.compact, fl.o.ThinBands)
	}
	tiny, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: 22})
	fl := tiny.(Dashboard).layout()
	if !fl.compact || !fl.o.ThinBands {
		t.Fatalf("tiny: only when the floor cannot be held do the bands thin: compact=%v thin=%v window=%d", fl.compact, fl.o.ThinBands, fl.window)
	}
	v := stripANSITest(tiny.View().Content)
	if strings.Count(v, "[") < 2 || !strings.Contains(v, "L O C A T I O N") {
		t.Fatalf("thin bands still label the groups:\n%s", v)
	}
}

// HUM LEAD UAT nit 2026-08-27: the rail's ▲ rides the band's bottom row when
// the band is three rows (no gap above the rail line) and the single row
// when the band is thin.
func TestRecentRailTopRidesTheBandsBottomRow(t *testing.T) {
	m := dash(t)
	rs := snap()
	for i := range 25 {
		loc := snap().Locations[0]
		loc.Label, loc.Lat = fmt.Sprintf("City %02d", i), 30+float64(i)/10
		rs.Locations = append(rs.Locations, loc)
	}
	rs.Locations = rs.Locations[1:]
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	for _, h := range []int{44, 22} { // three-row bands, then thin bands
		n, _ := m.Update(tea.WindowSizeMsg{Width: 133, Height: h})
		d := n.(Dashboard)
		lines := strings.Split(stripANSITest(d.recentSection(d.layout())), "\n")
		up := -1
		for i, l := range lines {
			if strings.HasSuffix(strings.TrimRight(l, " "), "▲") {
				up = i
			}
		}
		if up < 0 || up+1 >= len(lines) || !strings.Contains(lines[up+1], "│") && !strings.Contains(lines[up+1], "█") {
			t.Fatalf("h=%d: the ▲ must sit directly above the rail's first row (found at %d):\n%s", h, up, strings.Join(lines[:min(6, len(lines))], "\n"))
		}
		if want := d.layout().o.BandHeight() - 1; up != want {
			t.Fatalf("h=%d: ▲ on the band's bottom row %d, got %d", h, want, up)
		}
	}
}

// HUM LEAD UAT 2026-08-27: [ctrl+a] Favorite is the mirror of [shift+del]
// Unfavorite — enabled on a recent/searched row, dimmed and inert on a
// watchlist row; on a recent row it adds that location to the watchlist.
func TestFavoriteChipEnabledOnRecentRowsOnly(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.ResetColorEnabledForTest()
	muted, full := "48;2;43;43;43", "48;2;86;86;86"
	tone := func(v string) string {
		for _, l := range strings.Split(v, "\n") {
			if i := strings.Index(l, "ctrl+a"); i > 0 {
				if j := strings.LastIndex(l[:i], "\x1b["); j >= 0 {
					return l[j:i]
				}
			}
		}
		return ""
	}
	fr := &fakeHooks{}
	m := dashWithHooks(t, fr)
	rs := snap()
	rs.Locations[0].Label, rs.Locations[0].Lat, rs.Locations[0].Zip = "Recent City", 40.0, "99999"
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	// Watchlist row (selected 0): Favorite muted, ctrl+a inert.
	if c := tone(m.View().Content); !strings.Contains(c, muted) || strings.Contains(c, full) {
		t.Fatalf("watchlist row: Favorite muted: %q", c)
	}
	n, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	_ = drain(t, n, cmd)
	if len(fr.committed) != 0 {
		t.Fatal("ctrl+a on a watchlist row favorites nothing")
	}
	// Recent row: Favorite bright; ctrl+a favorites it.
	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if c := tone(down.View().Content); !strings.Contains(c, full) {
		t.Fatalf("recent row: Favorite enabled: %q", c)
	}
	fav, cmd := down.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	_ = drain(t, fav, cmd)
	if len(fr.committed) != 1 || fr.committed[0][0][len(fr.committed[0][0])-1].Label != "Recent City" {
		t.Fatalf("ctrl+a on a recent row favorites it: %+v", fr.committed)
	}
}

// The header's inner ladder (round 4, C-11a): as the row narrows the mute
// label shortens, then the mute chip drops, then the API summary loses its
// label, then the chips stand alone, then the summary leaves — in that
// order, the row never wider than the box.
func TestHeaderRowLaddersInOrder(t *testing.T) {
	d := dash(t).(Dashboard)
	seen := map[string]int{}
	prev := ""
	for w := 160; w >= 40; w-- {
		nd, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: 44})
		row := strings.TrimSpace(strings.Trim(strings.Split(stripANSITest(nd.(Dashboard).header(nd.(Dashboard).opts())), "\n")[1], "┃"))
		if render.Width(row) > nd.(Dashboard).opts().Width-8 {
			t.Fatalf("width %d: the row exceeds the box: %q", w, row)
		}
		form := "chips-only"
		switch {
		case strings.Contains(row, "[M] Mute Severe Alerts"):
			form = "full"
		case strings.Contains(row, "[M] Mute "):
			form = "mute-short"
		case strings.Contains(row, "[s] Setup"):
			form = "no-mute"
		}
		if strings.Contains(row, "API: ") {
			form += "+api"
		} else if strings.Contains(row, "✔") {
			form += "+count"
		}
		if form != prev {
			seen[form] = w
			prev = form
		}
	}
	order := []string{"full+api", "mute-short+api", "no-mute+api", "no-mute+count", "chips-only+count", "chips-only"}
	last := 999
	for _, f := range order {
		w, ok := seen[f]
		if !ok || w > last {
			t.Fatalf("ladder order %v, first widths %v", order, seen)
		}
		last = w
	}
}

// The module counts and chips follow the page as SHOWN when the alert list
// shrinks under the raw index (REVIEW R5-C-09), and a wide-rune event never
// widens the compact row (R5-C-10).
func TestAlertModulePageFollowsAShrunkListAndWideRunesFit(t *testing.T) {
	d := dash(t).(Dashboard)
	d.alertIdx = 2
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: "Heat Advisory", Severity: "minor", Headline: "h"}}
	row := stripANSITest(strings.Split(d.alertArea(d.layout()), "\n")[1])
	if !strings.Contains(row, "01/01") {
		t.Fatalf("one page: %q", row)
	}
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	if muted := strings.Count(d.alertArea(d.layout()), "48;2;43;43;43"); muted != 2 {
		t.Fatalf("both paging chips mute on a single page, got %d", muted)
	}
	rendering.SetColorEnabledForTest(false)
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: strings.Repeat("日", 60), Severity: "minor", Headline: "h"}}
	short, _ := d.Update(tea.WindowSizeMsg{Width: 133, Height: 24})
	for _, l := range strings.Split(stripANSITest(short.View().Content), "\n") {
		if render.Width(l) > 133 {
			t.Fatalf("a wide-rune event overflows the compact row (%d): %q", render.Width(l), l)
		}
	}
}
