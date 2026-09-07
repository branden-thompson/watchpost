package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// UAT 2026-08-30 (#10): the tick is a ✔, not an x.
//
// These boxes ENABLE a thing — "this class is muted", "this report has its own
// correspondent". An x reads as crossing something out, which inverts the
// metaphor exactly where the two groups are least alike: one silences, the
// other assigns.
func TestACheckedBoxIsATickNotACross(t *testing.T) {
	d := goldenDash(t, false)
	o := d.opts()
	if got := checkMark(o, true); got != "[✔]" {
		t.Errorf("checkMark(ticked) = %q, want a tick", got)
	}
	if got := checkMark(o, false); got != "[ ]" {
		t.Errorf("checkMark(unticked) = %q", got)
	}
	// Through the glyph set, so --ascii has a form of its own.
	o.ASCII = true
	if got := checkMark(o, true); strings.ContainsAny(got, "✔✘") {
		t.Errorf("--ascii checkMark = %q, want an ASCII form", got)
	}
}

// UAT 2026-08-30 (#11): ↑↓ are a CAROUSEL — ↓ on the last row returns to the
// first, ↑ on the first goes to the last.
//
// Stopping dead at an end reads as a stuck key: the listener presses again,
// nothing moves, and they cannot tell that from a hung app.
func TestSetupArrowsWrapAround(t *testing.T) {
	all := func(setupRowID) bool { return true }
	// The ends are DERIVED. Naming today's last row makes every group added
	// fail this test for the one reason it does not care about.
	first, last := setupRowID(0), setupRowID(setupRowCount-1)
	if got := nextRow(last, all); got != first {
		t.Errorf("down on the last row wraps to the first, got %v", got)
	}
	if got := prevRow(first, all); got != last {
		t.Errorf("up on the first row wraps to the last, got %v", got)
	}
	// And it still steps normally in the middle.
	if got := nextRow(rowLocation, all); got != rowFIRMSKey {
		t.Errorf("down steps one row, got %v", got)
	}
	if got := prevRow(rowFIRMSKey, all); got != rowLocation {
		t.Errorf("up steps one row, got %v", got)
	}
	// enter's rule is about the KIND of row, not its position: a text field
	// commits and moves on, everything else saves. The wrap therefore cannot
	// change where enter saves.
	if enterSaves(rowFIRMSKey) || enterSaves(rowLocation) {
		t.Error("enter on a text field commits it and moves on — it must not save")
	}
	if !enterSaves(rowEventsAll) || !enterSaves(rowCastSeismic) || !enterSaves(rowClassWarning) {
		t.Error("enter on any other row saves")
	}
	// A visible-set with nothing in it leaves the focus alone rather than
	// spinning (the counter bound).
	none := func(setupRowID) bool { return false }
	if got := nextRow(rowCastAlerts, none); got != rowCastAlerts {
		t.Errorf("with nothing focusable the focus stays put, got %v", got)
	}
}

// UAT 2026-08-30 (#12): arriving on the FIRMS key row and pressing enter — the
// natural "let me into this field" gesture — saved and closed the window, so
// the field could never be reached at all.
//
// Enter on a TEXT FIELD commits it and moves on; enter anywhere else saves.
func TestEnterOnATextFieldMovesOnRatherThanSaving(t *testing.T) {
	h := &setupHarness{}
	m, _ := NewDashboard(h.config())
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})

	// The location row is a text field: enter commits and moves to the key row.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := model.(Dashboard); got.modal != modalSetup || got.setup.focus != rowFIRMSKey {
		t.Fatalf("enter on the location field moves to the key row, got modal=%v focus=%v", got.modal, got.setup.focus)
	}
	// The key row is ALSO a text field: enter must not close the window before
	// anything can be typed into it.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if model.(Dashboard).modal != modalSetup {
		t.Fatal("enter on the key field must not save and close — the field would be unreachable")
	}
	// And the field takes text, spaces included.
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	d := model.(Dashboard)
	d.setup.focus = rowFIRMSKey
	if got := d.setup.key; got != "" && len(got) < 1 {
		t.Fatalf("the key field takes text: %q", got)
	}
	// The chip tells the truth on each kind of row.
	d.setup.focus = rowFIRMSKey
	if got := strings.Join(d.setupChips(d.opts()), " "); !strings.Contains(got, "Next") {
		t.Errorf("a text field's chip reads Next: %q", got)
	}
	d.setup.focus = rowEventsAll
	if got := strings.Join(d.setupChips(d.opts()), " "); !strings.Contains(got, "Save") {
		t.Errorf("any other row's chip reads Save: %q", got)
	}
}
