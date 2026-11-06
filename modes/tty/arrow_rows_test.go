package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// unflashedSettings is the Settings window as drawn, with every key's flash
// cleared so only what the rows say is compared.
func unflashedSettings(d Dashboard) string {
	d.setup.flash = flashNone
	d.volFlash, d.mapPane.flash, d.mapPane.menuFlash = "", "", flashNone
	return stripANSITest(d.View().Content)
}

// TestEveryArrowRowChangesSomething is 0.19.5 Q2.8 (JM-8): every toggle and
// picker row in Settings changes what it says on ← and on →. setupRowKey's
// arrow branches are an ordered chain, and a row whose branch sits where
// it is never reached does nothing at all; this names it.
func TestEveryArrowRowChangesSomething(t *testing.T) {
	d, err := NewDashboard(Config{Voices: func() []string { return []string{"Samantha", "Daniel", "Karen"} }})
	if err != nil {
		t.Fatal(err)
	}
	d.width, d.height = 160, 60
	// The Theme row applies the theme to the process as it moves, so the theme
	// this test found is put back for the tests after it.
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	for _, row := range setupTable() {
		if row.kind != rowToggle && row.kind != rowPicker {
			continue
		}
		at := d.openSetupAt(row.id)
		before := unflashedSettings(at)
		for _, k := range []tea.KeyPressMsg{{Code: tea.KeyLeft}, {Code: tea.KeyRight}} {
			m, _ := at.Update(k)
			if unflashedSettings(m.(Dashboard)) == before {
				t.Errorf("row %d (%v) does not change on %s", row.id, row.kind, k.String())
			}
		}
	}
}
