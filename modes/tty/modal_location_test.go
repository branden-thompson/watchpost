package tty

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestLKeyOpensLookupModal(t *testing.T) {
	// UAT 26.4 (reworded 2026-08-27): [l] Lookup Location floats the search
	// modal; typing builds the query (global bindings must not fire); esc
	// cancels. (ctrl+a is now Favorite — see TestFavoriteChipEnabledOnRecentRowsOnly.)
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if v := m.View().Content; !strings.Contains(v, "Lookup Location") || !strings.Contains(v, "Search:") {
		t.Fatalf("lookup modal missing:\n%s", v)
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
	if v := m.View().Content; strings.Contains(v, "Lookup Location") && strings.Contains(v, "Search:") {
		t.Fatal("esc must close the lookup modal")
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
	if len(h.committed) != 0 || mCancel.(Dashboard).modal == modalRemove {
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
	if d.modal != modalDetails || d.selected != d.numPriority() {
		t.Fatalf("lookup must open details focused on the new recent top: details=%v sel=%d", d.modal == modalDetails, d.selected)
	}
}

func TestFavoriteMutesAtTheWatchlistCap(t *testing.T) {
	// UAT 26.3 (2026-08-27): at the 10-location cap, [ctrl+a] Favorite on a
	// recent row mutes and does nothing — the cap is felt at the chip, not a
	// modal note.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	h := &fakeHooks{}
	m := dashWithHooks(t, h)
	full := snap()
	for i := range 9 {
		full.Locations = append(full.Locations, snapshot.Location{Label: fmt.Sprintf("City %d", i), Zip: fmt.Sprintf("100%02d", i)})
	}
	m, _ = m.Update(SnapshotMsg{Snap: full})
	rs := snap()
	rs.Locations[0].Label, rs.Locations[0].Zip = "Recent City", "99999"
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	for range 10 { // onto the recent row
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	favMuted := false
	for _, l := range strings.Split(m.View().Content, "\n") {
		if k := strings.Index(l, "ctrl+a"); k > 0 {
			if e := strings.LastIndex(l[:k], "\x1b["); e >= 0 && strings.Contains(l[e:k], "48;2;43;43;43") {
				favMuted = true
			}
		}
	}
	if !favMuted {
		t.Fatal("at the cap, Favorite must be muted even on a recent row")
	}
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	_ = drain(t, m2, cmd)
	if len(h.committed) != 0 {
		t.Fatal("ctrl+a must be inert at the cap")
	}
}

// A lookup opens Details on the looked-up location from the FIRST frame —
// blank until its data lands — never on the row that held the index before
// (HUM LEAD UAT 2026-08-28: the modal opened on the old top RECENT row).
func TestLookupOpensDetailsOnTheLookedUpLocationFromTheFirstFrame(t *testing.T) {
	m := dash(t)
	rs := snap()
	rs.Locations[0].Label, rs.Locations[0].Zip = "Ridgecrest, CA", "93555"
	m, _ = m.Update(RecentSnapshotMsg{Snap: rs})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	ref := snapshot.LocationRef{Label: "Escondido, CA", Zip: "92025", Lat: 33.1, Lon: -117.1, TZ: "America/Los_Angeles"}
	m, _ = m.Update(resolvedMsg{mode: "lookup", ref: ref})
	d := m.(Dashboard)
	if d.modal != modalDetails || d.lookupRef == nil {
		t.Fatalf("details open on the lookup: modal=%v pending=%v", d.modal, d.lookupRef != nil)
	}
	if v := stripANSITest(d.detailsModal(d.opts())); !strings.Contains(v, "Escondido, CA 92025") || strings.Contains(v, "Ridgecrest") {
		t.Fatalf("the first frame's modal names the looked-up location, blank — not the old top row:\n%s", v)
	}
	if d.hydrateCmd() != nil {
		t.Fatal("no hydrate while the lookup fetches its own location")
	}
	// The rebuilt RECENT list arrives with the location at the top: the
	// placeholder gives way to the data.
	landed := snap()
	landed.Locations = append([]snapshot.Location{{Label: "Escondido, CA", Zip: "92025", TZ: "America/Los_Angeles"}}, landed.Locations...)
	m, _ = m.Update(RecentSnapshotMsg{Snap: landed})
	d = m.(Dashboard)
	if d.lookupRef != nil || d.selectedLocation() == nil || d.selectedLocation().Label != "Escondido, CA" {
		t.Fatalf("the lookup landed: pending=%v sel=%v", d.lookupRef != nil, d.selectedLocation())
	}
	// Closing the modal before the data lands drops the wait too.
	m, _ = m.Update(resolvedMsg{mode: "lookup", ref: snapshot.LocationRef{Label: "Vista, CA", Zip: "92083"}})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.(Dashboard).lookupRef != nil {
		t.Fatal("esc clears the pending lookup")
	}
}

// A lookup with an EMPTY RECENT list (the first run) keeps its focus through
// a priority publish and lands by identity when the rebuilt list arrives
// (REVIEW R5-C-02: the focus fell to the first favourite and the wait never
// cleared).
func TestLookupWithEmptyRecentSurvivesAPriorityPublish(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	ref := snapshot.LocationRef{Label: "Lookup City", Zip: "99999"}
	m, _ = m.Update(resolvedMsg{mode: "lookup", ref: ref})
	m, _ = m.Update(SnapshotMsg{Snap: snap()}) // a priority publish lands meanwhile
	d := m.(Dashboard)
	if d.lookupRef == nil || d.modal != modalDetails || !strings.Contains(stripANSITest(d.detailsModal(d.opts())), "Lookup City") {
		t.Fatalf("the lookup keeps its focus: pending=%v modal=%v", d.lookupRef != nil, d.modal)
	}
	landed := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, Locations: []snapshot.Location{{Label: "Lookup City", Zip: "99999"}}}
	m, _ = m.Update(RecentSnapshotMsg{Snap: landed})
	d = m.(Dashboard)
	if d.lookupRef != nil || d.selected != d.numPriority() || d.selectedLocation() == nil || d.selectedLocation().Label != "Lookup City" {
		t.Fatalf("landed by identity: pending=%v selected=%d", d.lookupRef != nil, d.selected)
	}
	// Any other window drops the wait (R5-B-09).
	m, _ = m.Update(resolvedMsg{mode: "lookup", ref: snapshot.LocationRef{Label: "Vista, CA", Zip: "92083"}})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.(Dashboard).lookupRef != nil {
		t.Fatal("another window clears the pending lookup")
	}
}
