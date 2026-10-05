package tty

// window_stack_test.go — 0.18.0 D-106, D-107: the windows form a stack. A
// window opened from another opens over it; esc, or the key that opened it,
// returns to the one below, which resumes. A window already in the stack is
// returned to, never doubled; a search or confirmation window is replaced by
// what it opens, never returned to.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func pressEnter(d Dashboard) Dashboard {
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return m.(Dashboard)
}

// TestDetailsFromTheMapReturnsToTheMap is the HUM LEAD's path (D-106): on the
// map, ] to a place, enter for its details; esc goes back to the map, and so
// does enter again.
func TestDetailsFromTheMapReturnsToTheMap(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d = pressCode(d, ']', "]")
	picked := d.selected
	d = pressEnter(d)
	if d.modal != modalDetails {
		t.Fatalf("enter on the map opened window %d; want the place's details", d.modal)
	}
	if d.selected != picked {
		t.Error("the details are not the place picked on the map")
	}
	d, _ = pressKey(d, "esc")
	if d.modal != modalMap {
		t.Fatalf("esc from the details went to window %d; want back to the map", d.modal)
	}
	d = pressEnter(d)
	if d = pressEnter(d); d.modal != modalMap {
		t.Errorf("enter again from the details went to window %d; want back to the map", d.modal)
	}
	if d, _ = pressKey(d, "esc"); d.modal != modalNone || len(d.under) != 0 {
		t.Errorf("esc from the map, nothing under it, left window %d and %d under", d.modal, len(d.under))
	}
}

// TestAWindowInTheStackIsReturnedToNotDoubled is D-107: map, then Help over
// it, then g - the map again - is the map below, not a second one above.
func TestAWindowInTheStackIsReturnedToNotDoubled(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d, _ = pressKey(d, "?")
	if d.modal != modalHelp || len(d.under) != 1 {
		t.Fatalf("Help over the map: window %d, %d under", d.modal, len(d.under))
	}
	d, _ = pressKey(d, "g")
	if d.modal != modalMap || len(d.under) != 0 {
		t.Errorf("g from Help over the map: window %d, %d under; want the map, nothing under", d.modal, len(d.under))
	}
}

// TestASearchWindowIsNeverReturnedTo is D-107: a search window that opens a
// place's details is done - esc from the details does not reopen it.
func TestASearchWindowIsNeverReturnedTo(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d = d.open(modalAdd).open(modalDetails)
	if len(d.under) != 1 || d.under[0].modal != modalMap {
		t.Fatalf("the stack under the details is %+v; want the map alone", d.under)
	}
	if d = d.close(); d.modal != modalMap {
		t.Errorf("closing the details went to window %d; want the map, the search window gone", d.modal)
	}
}

// TestTheWindowReturnedToResumes is D-107: back on the map it is drawn again
// in Update, and its scroll is as it was left.
func TestTheWindowReturnedToResumes(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d.modalScroll = 3
	d = pressEnter(d)
	if d.modalScroll != 0 {
		t.Error("the details opened at the map's scroll")
	}
	calls := []string{}
	d.mapPane.calls = &calls
	gen := d.mapPane.gen
	d, _ = pressKey(d, "esc")
	if d.modalScroll != 3 {
		t.Errorf("back on the map the scroll is %d; want 3, as left", d.modalScroll)
	}
	if d.mapPane.gen == gen || !strings.Contains(strings.Join(calls, ","), "Render") {
		t.Errorf("back on the map it was not drawn again: %v", calls)
	}
}

// TestTheControlsRowNamesTheMap is D-106: g Maps to the left of ↑↓ Navigate.
func TestTheControlsRowNamesTheMap(t *testing.T) {
	d := mapDash(t, Config{})
	row := stripANSITest(d.controlRow(d.opts()))
	maps, nav := strings.Index(row, "g] Maps"), strings.Index(row, "Navigate")
	if maps < 0 || nav < 0 || maps > nav {
		t.Errorf("the controls row is %q; want g Maps left of Navigate", row)
	}
}
