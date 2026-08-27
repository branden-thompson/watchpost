package tty

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

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
