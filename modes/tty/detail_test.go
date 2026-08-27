package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

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
	d.modal = modalDetails
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

func TestDetailGridAlignsCurrentlyWithForecast(t *testing.T) {
	// UAT 32.1/32.5: CURRENTLY temps sit on the FORECAST condition column;
	// humidity sits on the HIGH/LOW column.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Harmonized.Feels = f64(24.4)
	s2.Locations[0].Harmonized.HumidityPct = f64(65)
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
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

func TestDetailDividerEndsBeforeRail(t *testing.T) {
	// UAT 66 (supersedes 33.1): the divider runs to 3 cells before the
	// vertical scroll rail; alert text wraps to that same right edge.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Alerts = []snapshot.Alert{{Event: "Heat Advisory", Severity: "minor",
		Description: strings.Repeat("Dangerous heat across the valleys and foothills. ", 6) + "\n\n* " + strings.Repeat("BULLET TEXT THAT WRAPS ", 6)}}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
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

func TestDetailShowsStationAndDistance(t *testing.T) {
	// UAT 60.2: the observing station and its distance are always one
	// drill-in away, on the CURRENTLY grid (label / col 14 / col 37).
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Harmonized.Source = snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KCRQ", DistanceKm: f64(5.7)}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
	joined := strings.Join(d.detailLines(), "\n")
	want := "Station   :   KCRQ" + strings.Repeat(" ", 19) + "Distance  :  4 mi"
	if !strings.Contains(joined, want) {
		t.Fatalf("CURRENTLY must carry %q:\n%s", want, joined)
	}
	loading := dash(t).(Dashboard)
	loading.modal = modalDetails
	if strings.Contains(strings.Join(loading.detailLines(), "\n"), "Station") {
		t.Fatal("no station row before an observation lands")
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
	if m2.(Dashboard).modal != modalDetails || cmd == nil {
		t.Fatal("details on a recent row without hourly must return the hydrate cmd")
	}
	runCmd(cmd)
	if len(got) != 1 || got[0] != "Recent Town" {
		t.Fatalf("hydrate hook must be called for the recent row, got %v", got)
	}
	// A priority row never hydrates through the hook (the command that
	// comes back is the Details tick — Q3 — not a hydrate).
	d.selected = 0
	_, cmd = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	if len(got) != 1 {
		t.Fatalf("priority rows have their own hourly tier, got %v", got)
	}
}
