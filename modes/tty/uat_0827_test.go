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
		want := render.Tint("Updated: "+at.Local().Format("01/02/2006 15:04:05 MST")+" "+row.want, render.Tok(row.tone))
		if !strings.Contains(line, want) {
			t.Errorf("age %v: want %q in %q", row.age, want, stripANSITest(line))
		}
	}
	// Before the first data: grey.
	d.snap = nil
	if line := strings.SplitN(d.header(d.opts()), "\n", 2)[0]; !strings.Contains(line, render.Tint("awaiting first data...", render.Tok(render.TextBase))) {
		t.Errorf("no data: grey placeholder: %q", stripANSITest(line))
	}
	// A narrow terminal drops the age before it drops the date (the ladder).
	d.snap = snap()
	d.now = func() time.Time { return at.Add(2 * time.Minute) }
	nd, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 44})
	if line := stripANSITest(strings.SplitN(nd.(Dashboard).header(nd.(Dashboard).opts()), "\n", 2)[0]); strings.Contains(line, "Ago") || !strings.Contains(line, "Updated:") {
		t.Errorf("narrow: the plain stamp, no age: %q", line)
	}
}

func TestAlertModuleHeaderRowGeometry(t *testing.T) {
	d := dash(t).(Dashboard)
	d.snap.Locations[0].Alerts = []snapshot.Alert{{Event: "Air Quality Alert", Severity: "minor", Headline: "until Friday"}, {Event: "Heat Advisory", Severity: "minor", Headline: "h"}}
	fl := d.layout()
	if fl.compact {
		t.Fatal("the fixture is the full layout")
	}
	lines := strings.Split(stripANSITest(d.alertArea(fl)), "\n")
	_, bg := render.AlertBlockTone("Air Quality Alert", "minor")
	mw := fl.o.ModuleInnerWidth(bg)
	head := strings.TrimRight(lines[0], " ")
	inner := strings.TrimLeft(head, " ")
	if !strings.HasPrefix(inner, "01 / 02 Alerts") || !strings.HasSuffix(inner, "Next") {
		t.Fatalf("count at the left inset, controls at the right: %q", head)
	}
	title := "⚠ AIR QUALITY ALERT · Oceanside, CA"
	i := strings.Index(inner, title)
	if i < 0 {
		t.Fatalf("title row: %q", inner)
	}
	// Centred on the row's global centre — the same axis as the header's
	// stamp (UAT nit 2026-08-27), not the middle of the gap.
	if centre, want := i+render.Width(title)/2, mw/2; centre < want-1 || centre > want+1 {
		t.Fatalf("title centred on the row's centre (%d vs %d): %q", centre, want, inner)
	}
	head1 := stripANSITest(strings.SplitN(d.header(fl.o), "\n", 2)[0])
	si := strings.Index(head1, "Updated:")
	sw := strings.Index(head1[si:], "  ")
	if sc, want := si+sw/2, fl.o.Width/2; sc < want-1 || sc > want+1 {
		t.Fatalf("the stamp sits on the same axis (%d vs %d): %q", sc, want, head1)
	}
	if render.Width(inner) != mw {
		t.Fatalf("the row spans the module width %d: %d", mw, render.Width(inner))
	}
	if len(lines) < 5 || strings.TrimSpace(lines[1]) != "" || !strings.Contains(lines[2], "[minor] until Friday") {
		t.Fatalf("header, blank, body: %q", lines[:min(5, len(lines))])
	}
	if fl.alertH != render.ModuleHeight(5, bg) {
		t.Fatalf("the module is five content lines: %d", fl.alertH)
	}
	// Narrow: the title shortens with an ellipsis before the row overflows.
	nd, _ := d.Update(tea.WindowSizeMsg{Width: 84, Height: 44})
	nfl := nd.(Dashboard).layout()
	row := strings.TrimRight(strings.Split(stripANSITest(nd.(Dashboard).alertArea(nfl)), "\n")[0], " ")
	if render.Width(row) > 84 || !strings.Contains(row, "…") && strings.Contains(row, "AIR QUALITY") && !nfl.compact {
		t.Fatalf("narrow row within width, title ellipsized when needed: %q", row)
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
