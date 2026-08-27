package tty

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

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

func TestRecentListCapsAtFifty(t *testing.T) {
	// UAT 48: 10 favourites + 50 most-recent = 60 tracked locations.
	refs := []snapshot.LocationRef{}
	for i := range 55 {
		refs = prependRef(refs, snapshot.LocationRef{Label: fmt.Sprintf("L%d", i), Zip: fmt.Sprintf("%05d", i)})
	}
	if len(refs) != RecentCap || RecentCap != 50 {
		t.Fatalf("recent list must cap at 50, got %d", len(refs))
	}
	if refs[0].Zip != "00054" {
		t.Fatal("newest stays on top")
	}
}
