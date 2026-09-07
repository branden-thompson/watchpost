package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// The WATCHPOST UI group: the theme picker that
// replaced a modal, and the two display preferences that had no home.

// uiDash opens Settings on a row of the UI group, with a SetUI hook that
// records what the window writes.
func uiDash(t *testing.T, at setupRowID) (Dashboard, *UIPrefs) {
	t.Helper()
	var got UIPrefs
	d := setupGolden(t, 133, 44, false, at)
	d.cfg.SetUI = func(p UIPrefs) error { got = p; return nil }
	return d, &got
}

// ←→ on the theme row APPLY the theme as they move. A list of names tells a
// listener nothing about which one they want, so the answer is to show them.
func TestThemePickerPreviewsLive(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, _ := uiDash(t, rowTheme)
	before := render.ThemeName()
	m, _, handled := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if !handled {
		t.Fatal("→ operates the theme picker")
	}
	if render.ThemeName() == before {
		t.Errorf("→ applies the next theme live, still on %q", before)
	}
	if name := m.(Dashboard).themeName(); name != render.ThemeName() {
		t.Errorf("the picker shows the theme in force: picker %q, applied %q", name, render.ThemeName())
	}
}

// The theme picker names no preview key. ←→ have already previewed it — the
// whole app repainted — so `p Preview` would name a key for something that has
// already happened.
func TestThemeRowOffersNoPreviewChip(t *testing.T) {
	d, _ := uiDash(t, rowTheme)
	chips := strings.Join(d.setupChips(d.opts()), " ")
	if strings.Contains(chips, "Preview") {
		t.Errorf("the theme previews itself; the chip row must not offer p Preview: %q", chips)
	}
	if !strings.Contains(chips, "Theme") {
		t.Errorf("the chip row names what ←→ do here: %q", chips)
	}
}

// The units and the clock are radio sets: space selects the focused option, and
// the effect is immediate — the units always were live, and the clock now is.
func TestUnitsAndClockSelectLive(t *testing.T) {
	for _, tc := range []struct {
		name  string
		at    setupRowID
		check func(Dashboard) bool
	}{
		{"metric", rowUnitsMetric, func(d Dashboard) bool { return d.units == render.UnitC }},
		{"24hr", rowClock24, func(d Dashboard) bool { return d.clockFmt == render.Clock24 }},
		{"MIL", rowClockMil, func(d Dashboard) bool { return d.clockFmt == render.ClockMil }},
	} {
		d, _ := uiDash(t, tc.at)
		nd := d.setupSpace()
		if !tc.check(nd) {
			t.Errorf("%s: space on the row selects it", tc.name)
		}
		if !nd.setup.uiDirty {
			t.Errorf("%s: the choice is marked for the write on close", tc.name)
		}
	}
}

// Closing WRITES all three together — one hook, one write. Three Loads and
// Saves over the same file in a row is three chances for two of them to
// disagree about what the third wrote.
func TestClosingSettingsWritesTheDisplayPreferences(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, got := uiDash(t, rowClockMil)
	d = d.setupSpace() // MIL
	d.setup.focus = rowUnitsMetric
	d = d.setupSpace() // Metric
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.Clock != "mil" || got.Units != "metric" {
		t.Errorf("esc writes what was chosen, got %+v", *got)
	}
	if got.Theme == "" {
		t.Errorf("the theme is written with them, got %+v", *got)
	}
}

// An untouched group writes NOTHING. Opening Settings to read it must not
// rewrite the file — the same rule the cast group follows.
func TestAnUntouchedUIGroupWritesNothing(t *testing.T) {
	d, _ := uiDash(t, rowTheme)
	writes := 0
	d.cfg.SetUI = func(UIPrefs) error { writes++; return nil }
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if writes != 0 {
		t.Errorf("an untouched window writes nothing, got %d", writes)
	}
}

// The clock reaches the frame. This is the point of the setting: one owner, and
// every site that shows a time reads it.
func TestTheClockSettingReachesTheFrame(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowClock24)
	if got := d.opts().Clock; got != render.Clock12 {
		t.Fatalf("the fixture starts on the 12-hour clock, got %v", got)
	}
	nd := d.setupSpace()
	if got := nd.opts().Clock; got != render.Clock24 {
		t.Errorf("selecting 24hr reaches the render options, got %v", got)
	}
	// The header stamp is the site a listener sees first.
	twelve := stripANSITest(d.header(d.opts()))
	twentyFour := stripANSITest(nd.header(nd.opts()))
	if twelve == twentyFour {
		t.Errorf("the header stamp follows the clock:\n%s\n%s", twelve, twentyFour)
	}
	if strings.Contains(twentyFour, "AM") || strings.Contains(twentyFour, "PM") {
		t.Errorf("the 24-hour stamp carries no meridiem: %s", twentyFour)
	}
}

// Every form round-trips through its config word, and an unknown word reads as
// the default rather than failing a load — a display preference is not worth
// refusing to start over.
func TestClockAndUnitsRoundTripTheirConfigWords(t *testing.T) {
	for _, c := range render.ClockOrder() {
		if got := render.ClockByKey(c.Key()); got != c {
			t.Errorf("clock %v round-trips through %q, got %v", c, c.Key(), got)
		}
	}
	for _, u := range render.UnitsOrder() {
		if got := render.UnitsByKey(u.Key()); got != u {
			t.Errorf("units %v round-trip through %q, got %v", u, u.Key(), got)
		}
	}
	if got := render.ClockByKey("half past"); got != render.Clock12 {
		t.Errorf("an unknown clock word reads as the default, got %v", got)
	}
	if got := render.UnitsByKey("furlongs"); got != render.UnitF {
		t.Errorf("an unknown units word reads as the default, got %v", got)
	}
}

// The two columns BALANCE THEMSELVES.
//
// The groups used to be assigned to columns by hand, and every group added made
// that worse — by 0.14.0 four stood against one, so the right column ended a
// dozen rows short and the window was a dozen rows taller than it needed to be.
// The point of computing the split is that adding the NEXT group needs no
// tweak, so that is what this asserts: a property, not today's arrangement.
func TestSettingsColumnsBalanceThemselves(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	o := d.opts()
	blocks := d.setupBlocks(o)
	plan, ok := d.columnPlan(blocks, o)
	if !ok {
		t.Fatal("at 133x44 the groups fit side by side")
	}
	left, right := blockHeight(blocks[:plan.split]), blockHeight(blocks[plan.split:])
	// No OTHER split point may be more even. This is the whole rule, and it
	// holds whatever the groups happen to be.
	best := abs(left - right)
	for k := 1; k < len(blocks); k++ {
		if got := abs(blockHeight(blocks[:k]) - blockHeight(blocks[k:])); got < best {
			// A more even split is allowed to lose only by not FITTING.
			lw, rw := widestBlock(blocks[:k]), widestBlock(blocks[k:])
			if twoColumnsWidth(lw, rw, panelChromeFor(blockHeight(blocks[:k])+4, d.modalMax())) <= o.Width {
				t.Errorf("split at %d is off by %d lines; a split at %d is off by %d and fits", plan.split, best, k, got)
			}
		}
	}
	// Reading order survives it: groups fill the left column, then the right.
	if plan.split < 1 || plan.split >= len(blocks) {
		t.Errorf("both columns carry at least one group, split=%d of %d", plan.split, len(blocks))
	}
}

// The balance has to pay for itself: a window that fits its budget needs no
// scroll rail, and that is what taking a dozen wasted rows out of it buys.
func TestSettingsFitsWithoutScrollingAtTheMockWidth(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	lines, _, _ := d.setupBody(d.opts())
	if len(lines) > d.modalMax() {
		t.Errorf("the balanced window fits its %d-line budget without a rail, got %d lines", d.modalMax(), len(lines))
	}
}

// The scroll must read the layout that is DRAWN. The focused row moves between
// columns as the split moves, and a focus line computed against the other
// column puts the mark off the edge — which reads as a dead keyboard.
func TestSettingsFocusLineFollowsTheBalancedLayout(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{{"two columns", 133, 44}, {"stacked", 80, 24}} {
		for id := setupRowID(0); id < setupRowCount; id++ {
			d := setupGolden(t, size.w, size.h, false, id)
			lines, at, end := d.setupBody(d.opts())
			if at < 0 || at >= len(lines) || end < at || end > len(lines) {
				t.Fatalf("%s: row %v spans [%d,%d] of %d lines", size.name, id, at, end, len(lines))
			}
			if !strings.Contains(lines[at], "›") {
				t.Errorf("%s: row %v — the focus mark is not on line %d: %q", size.name, id, at, stripANSITest(lines[at]))
			}
		}
	}
}

// A note NEVER RESIZES THE WINDOW.
//
// The correspondent notes were wrapped to a constant that happened to be wider
// than the rows they sit under, so focusing a row with a reason nudged the whole
// window out. A window that changes size when you move the cursor is the layout
// telling you it does not know its own mind.
func TestACorrespondentNoteNeverWidensTheWindow(t *testing.T) {
	base := setupGolden(t, 133, 44, false, rowLocation).setupWidth()
	for _, id := range castRowOrder() {
		d := setupGolden(t, 133, 44, false, id)
		if d.castNote(id) == "" {
			continue
		}
		if got := d.setupWidth(); got != base {
			t.Errorf("focusing %v changed the window from %d to %d cells", id, base, got)
		}
	}
}

// The note sits FLUSH with the label of the row that raised it, not inset under
// it: every other group's support line does, and this was the one that did not.
func TestACorrespondentNoteIsFlushWithItsRow(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowCastAlerts)
	note := d.castNote(rowCastAlerts)
	if note == "" {
		t.Skip("the fixture's focused row has no note")
	}
	block := d.setupBlock(d.opts(), groupCast)
	var rowAt, noteAt = -1, -1
	for _, l := range block.lines {
		plain := stripANSITest(l)
		trimmed := strings.TrimLeft(plain, " ")
		if trimmed == "" {
			continue
		}
		indent := len(plain) - len(trimmed)
		// An UNFOCUSED row: the focused one carries the › in its indent, and the
		// note aligns with the label, not with the mark.
		if strings.HasPrefix(trimmed, "Location Report") {
			rowAt = indent
		}
		if strings.HasPrefix(trimmed, strings.Fields(note)[0]) && rowAt >= 0 && noteAt < 0 {
			noteAt = indent
		}
	}
	if rowAt < 0 || noteAt < 0 {
		t.Fatalf("could not find the row and its note:\n%s", strings.Join(block.lines, "\n"))
	}
	if noteAt != rowAt {
		t.Errorf("the note starts at %d, its row's label at %d — flush, not inset", noteAt, rowAt)
	}
}

// BOTH EXITS WRITE THE SAME GROUPS. enter on a WATCHPOST UI row saves and
// closes, and it must carry the display preferences with it: the window state
// is cleared on the way out, so a group left off this path is a group the
// listener chose and the file never received.
func TestEnterSavingFromAUIRowWritesTheDisplayPreferences(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	d, saved := uiDash(t, rowClockMil)
	d = d.setupSpace() // MIL
	d.setup.focus = rowUnitsMetric
	d = d.setupSpace() // Metric

	d.setup.focus = rowClockMil
	model, cmd := d.setupAdvance() // enter on a UI row: saves and closes
	drain(t, model, cmd)

	if saved.Clock != "mil" || saved.Units != "metric" {
		t.Errorf("enter must write the display preferences it was shown, got %+v", *saved)
	}
}

// The two exits agree. Whatever one route writes, the other writes too —
// otherwise which key a listener happens to press decides which of their
// answers survive.
func TestTheTwoSettingsExitsWriteTheSameGroups(t *testing.T) {
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	choose := func(t *testing.T, exit func(Dashboard) (tea.Model, tea.Cmd)) UIPrefs {
		t.Helper()
		d, saved := uiDash(t, rowClockMil)
		d = d.setupSpace()
		d.setup.focus = rowUnitsMetric
		d = d.setupSpace()
		d.setup.focus = rowClockMil
		model, cmd := exit(d)
		drain(t, model, cmd)
		return *saved
	}
	viaEsc := choose(t, func(d Dashboard) (tea.Model, tea.Cmd) {
		return d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	})
	viaEnter := choose(t, func(d Dashboard) (tea.Model, tea.Cmd) { return d.setupAdvance() })
	if viaEsc.Clock != viaEnter.Clock || viaEsc.Units != viaEnter.Units {
		t.Errorf("the exits disagree: esc wrote %+v, enter wrote %+v", viaEsc, viaEnter)
	}
}
