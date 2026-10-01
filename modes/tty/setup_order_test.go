package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
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

// A GROUP'S PICKERS LINE UP (HUM LEAD, 2026-09-30): where a group's rows have
// ← → pickers, every ← sits in one column - Data → History's Hourly detail
// and Trends were two cells apart, its longer label pushing its picker over.
func TestAGroupsPickersLineUp(t *testing.T) {
	d := setupGolden(t, 133, 400, false, rowMapsOn)
	d.cfg.MapLayers = []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}, {Key: TemperatureLayer, Label: "Temperature"}}
	o := d.opts()
	for _, g := range setupGroups() {
		col := -1
		for _, l := range d.setupBlock(o, g).lines {
			plain := stripANSITest(l)
			i := strings.Index(plain, "←")
			if i < 0 {
				continue
			}
			at := render.Width(plain[:i])
			if col >= 0 && at != col {
				t.Errorf("%s: a ← at column %d, another at %d:\n%s", setupGroupTitle(g), col, at, plain)
			}
			if col < 0 {
				col = at
			}
		}
	}
}
