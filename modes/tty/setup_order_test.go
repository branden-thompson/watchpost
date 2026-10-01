package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ↓ WALKS A GROUP AS IT IS DRAWN (HUM LEAD, 2026-09-30): in the Maps tab ↓
// went from Radar to Temperature, past Radar ahead and Quakes drawn between
// them - focus followed the rows' declared order, the page its drawn one.
// Laid in one column, each ↓ within a group lands on a row drawn below the
// last.
func TestDownWalksAGroupAsItIsDrawn(t *testing.T) {
	d := setupGolden(t, 80, 400, false, rowMapsOn)
	table := setupTable()
	_, at, _ := d.focusBody(d.opts())
	was := d.setup.focus
	seen := map[setupRowID]bool{was: true}
	for range 2 * int(setupRowCount) {
		m, _ := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyDown})
		d = m.(Dashboard)
		now := d.setup.focus
		if seen[now] {
			break
		}
		seen[now] = true
		_, next, _ := d.focusBody(d.opts())
		if table[now].group == table[was].group && next <= at {
			t.Errorf("↓ from row %d (line %d) went to row %d, drawn above it (line %d)", was, at, now, next)
		}
		was, at = now, next
	}
	if !seen[rowMapRadarAhead] || !seen[rowMapQuakes] {
		t.Error("↓ never reached Radar ahead or Quakes")
	}
}
