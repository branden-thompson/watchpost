package tty

// cursor_test.go — F-205 AX-3 (FR-7.4, D-362, A-107 (12)): the terminal
// cursor stands at every focus point of both surfaces, steady, found in the
// frame as drawn.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// cursorCell is the character under a frame's cursor and its row, the
// cursor held steady; it fails without one.
func cursorCell(t *testing.T, v tea.View) (cell, row string) {
	t.Helper()
	if v.Cursor == nil {
		t.Fatal("no cursor in the frame")
	}
	if v.Cursor.Blink {
		t.Error("the cursor blinks (D-362: steady)")
	}
	lines := strings.Split(stripANSITest(v.Content), "\n")
	if v.Cursor.Y < 0 || v.Cursor.Y >= len(lines) {
		t.Fatalf("the cursor is on row %d of %d", v.Cursor.Y, len(lines))
	}
	row = lines[v.Cursor.Y]
	at := 0
	for _, r := range row {
		if at == v.Cursor.X {
			return string(r), row
		}
		at += render.RuneCells(r)
	}
	t.Fatalf("the cursor is at column %d past its row %q", v.Cursor.X, row)
	return "", row
}

// TestEveryFocusPointPlacesTheCursor is D-362: on the dashboard's selected
// row, Settings' focused row and the caret of the field it is typing in, the
// Severe Weather window's row, the search's caret, the map's Overlays menu;
// a window with no focus, Help, on its first words; under --ascii as well.
func TestEveryFocusPointPlacesTheCursor(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		fresh := func() Dashboard { // a dashboard of its own for each window: the window memo is shared by copies
			m, _ := mapDash(t, Config{ASCII: ascii}).Update(tea.WindowSizeMsg{Width: 133, Height: 44})
			return m.(Dashboard)
		}
		d := fresh()
		g := d.opts().Glyphs()
		if cell, row := cursorCell(t, d.View()); cell != g.Pointer || !strings.Contains(row, "Oceanside") {
			t.Errorf("ascii %v: on the dashboard the cursor is on %q of %q; want the selected row's pointer", ascii, cell, row)
		}
		for _, c := range []struct {
			row  setupRowID
			mark string
			has  string
		}{{rowTheme, g.Pointer, "Theme -"}, {rowReduceMotion, g.Pointer, "Reduce motion -"}, {rowMapPlayback, g.Pointer, "Playback speed -"}, {rowLocation, g.Cursor, "Search:"}} {
			s := fresh().openSetupAt(c.row)
			if cell, row := cursorCell(t, s.View()); cell != c.mark || !strings.Contains(row, c.has) {
				t.Errorf("ascii %v: Settings on row %d puts the cursor on %q of %q; want %q on the row with %q", ascii, c.row, cell, row, c.mark, c.has)
			}
		}
		help := fresh().open(modalHelp)
		if cell, _ := cursorCell(t, help.View()); strings.TrimSpace(cell) == "" {
			t.Errorf("ascii %v: in Help the cursor is on a blank cell", ascii)
		}
		severe := pressCode(fresh(), 'w', "w")
		if severe.modal != modalSevere {
			t.Fatalf("w opened window %d", severe.modal)
		}
		if cell, _ := cursorCell(t, severe.View()); cell != g.Pointer && strings.TrimSpace(cell) == "" {
			t.Errorf("ascii %v: in the Severe Weather window the cursor is on %q", ascii, cell)
		}
		search := fresh().open(modalAdd)
		if cell, row := cursorCell(t, search.View()); cell != g.Cursor {
			t.Errorf("ascii %v: in the search the cursor is on %q of %q; want the caret", ascii, cell, row)
		}
	}
	menu := pressCode(openMap(t, Config{}, 133, 44), 'O', "O")
	if cell, row := cursorCell(t, menu.View()); cell != menu.opts().Glyphs().Pointer {
		t.Errorf("in the Overlays menu the cursor is on %q of %q; want its focused row's pointer", cell, row)
	}
}

// TestThePropagationModesFocusPointsPlaceTheCursor is D-362 for the four
// D-110 named: the chart's target, the place finder's caret, the
// frequency's caret, and the hour ahead, said on the map's words.
func TestThePropagationModesFocusPointsPlaceTheCursor(t *testing.T) {
	d := chartDash(t)
	g := d.opts().Glyphs()
	chart := pressMap(t, d, "B")
	if cell, row := cursorCell(t, chart.View()); cell != g.Pointer {
		t.Errorf("on the chart the cursor is on %q of %q; want the target's pointer", cell, row)
	}
	for key, want := range map[string]modal{"/": modalPropFind, "F": modalPropFreq} {
		w := pressMap(t, d, key)
		if w.modal != want {
			t.Fatalf("%s opened window %d", key, w.modal)
		}
		if cell, row := cursorCell(t, w.View()); cell != g.Cursor {
			t.Errorf("%s: the cursor is on %q of %q; want the field's caret", key, cell, row)
		}
	}
	ahead, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if cell, row := cursorCell(t, ahead.(Dashboard).View()); strings.TrimSpace(cell) == "" || strings.ContainsFunc(row, isBraille) {
		t.Errorf("an hour ahead, the cursor is on %q of %q; want the map's words", cell, row)
	}
}

// isBraille reports a braille cell, the picture's.
func isBraille(r rune) bool { return r >= 0x2800 && r <= 0x28ff }

// TestTheMapPlacesTheCursorOnItsWords is A-107 (12): with nothing else
// focused, the map window's cursor is on the first line under the picture,
// or on the Description box's first line while it is open.
func TestTheMapPlacesTheCursorOnItsWords(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	v := d.View()
	cell, row := cursorCell(t, v)
	lines := strings.Split(stripANSITest(v.Content), "\n")
	if strings.TrimSpace(cell) == "" || strings.ContainsFunc(row, isBraille) || !strings.ContainsFunc(lines[v.Cursor.Y-1], isBraille) {
		t.Errorf("the cursor is on %q of %q; want the first line under the picture", cell, row)
	}
	box := pressCode(d, 'A', "A")
	if _, row := cursorCell(t, box.View()); !strings.Contains(row, "Oceanside, CA - Currently") {
		t.Errorf("with the Description open the cursor is on %q; want its first line", row)
	}
}

// TestTheConsolePlacesTheCursor is D-362 on the Broadcaster: on the running
// order's focused row, and in an Observer window opened over the console.
func TestTheConsolePlacesTheCursor(t *testing.T) {
	r := consoleRouter(t)
	g := r.broadcaster.opts().Glyphs()
	if cell, row := cursorCell(t, r.View()); cell != g.Pointer {
		t.Errorf("on the console the cursor is on %q of %q; want the focused row's pointer", cell, row)
	}
	d := mapDash(t, Config{ASCII: true})
	m, _ := d.Update(tea.WindowSizeMsg{Width: 150, Height: 74})
	r.observer = m.(Dashboard).openSetupAt(rowTheme)
	if cell, row := cursorCell(t, r.View()); cell != g.Pointer || !strings.Contains(row, "Theme -") {
		t.Errorf("with Settings over the console the cursor is on %q of %q; want its focused row", cell, row)
	}
}

// TestTheCursorIsSteady is D-362: the cursor does not blink, so it adds no
// motion, Reduce motion or not - a bar while the terminal is focused, a
// block while it is not (D-367).
func TestTheCursorIsSteady(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	for _, reduce := range []bool{false, true} {
		for _, blur := range []bool{false, true} {
			d := mapDash(t, Config{ReduceMotion: reduce})
			m, _ := d.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
			m = m.(Dashboard).openSetupAt(rowTheme)
			shape := tea.CursorBar
			if blur {
				m, shape = blurred(m), tea.CursorBlock
			}
			if v := m.View(); v.Cursor == nil || v.Cursor.Blink || v.Cursor.Shape != shape {
				t.Errorf("Reduce motion %v, blurred %v: the cursor is %+v; want a steady %v", reduce, blur, v.Cursor, shape)
			}
		}
	}
}

// cellUnder is the cell under a frame's cursor, as the frame draws it.
func cellUnder(t *testing.T, v tea.View) render.Cell {
	t.Helper()
	lines := strings.Split(v.Content, "\n")
	if v.Cursor == nil || v.Cursor.Y < 0 || v.Cursor.Y >= len(lines) {
		t.Fatalf("no cursor on the frame's rows: %+v", v.Cursor)
	}
	c, ok := render.CellAt(lines[v.Cursor.Y], v.Cursor.X)
	if !ok {
		t.Fatalf("the cursor is past its row: %+v", v.Cursor)
	}
	return c
}

// blurred is a model told the terminal lost focus.
func blurred(m tea.Model) tea.Model {
	m, _ = m.Update(tea.BlurMsg{})
	return m
}

// cursorScenes are the three focus points the cursor's look is held on, each
// a model of its own: the window memo is shared by copies.
func cursorScenes(t *testing.T) map[string]func() tea.Model {
	sized := func() tea.Model {
		m, _ := mapDash(t, Config{}).Update(tea.WindowSizeMsg{Width: 133, Height: 44})
		return m
	}
	return map[string]func() tea.Model{
		"Settings":      func() tea.Model { return sized().(Dashboard).openSetupAt(rowTheme) },
		"the dashboard": sized,
		"the console":   func() tea.Model { return consoleRouter(t) },
	}
}

// TestTheCursorWearsTheColourItCovers is D-366: while the terminal is not
// focused the cursor is a block in the colour the frame draws its cell in -
// Settings' yellow pointer, the dashboard's row, the console's - and with
// colour off it takes the terminal's own.
func TestTheCursorWearsTheColourItCovers(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	yellow, _ := render.InkAt(render.Tint(">", render.Tok(render.ListPointer)), 0)
	for name, scene := range cursorScenes(t) {
		v := blurred(scene()).View()
		c := cellUnder(t, v)
		if !c.Inked || v.Cursor.Color != c.Ink || v.Cursor.Shape != tea.CursorBlock {
			t.Errorf("on %s the cursor is %+v; want a block in its cell's colour, %v", name, v.Cursor, c.Ink)
		}
		if name == "Settings" && c.Ink != yellow {
			t.Errorf("on Settings the cursor's cell is drawn in %v; want the pointer's %v", c.Ink, yellow)
		}
	}
	rendering.SetColorEnabledForTest(false)
	for _, v := range []tea.View{blurred(cursorScenes(t)["Settings"]()).View(), cursorScenes(t)["Settings"]().View()} {
		if v.Cursor == nil || v.Cursor.Color != nil || v.Cursor.Shape != tea.CursorBlock {
			t.Errorf("with colour off the cursor is %+v; want a block in the terminal's own colour, focused or not", v.Cursor)
		}
	}
}

// TestAFocusedTerminalTintsTheCellUnderTheCursor is D-367: while the terminal
// is focused the cell under the cursor takes 15 % of its glyph's colour over
// the background it had - its own, else the frame's - the glyph keeps its
// colour, and the cursor is a steady bar in the tint, on every theme; told
// the terminal lost focus and regained it, the tint returns.
func TestAFocusedTerminalTintsTheCellUnderTheCursor(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	was := render.ThemeName()
	defer render.SetTheme(was)
	for _, theme := range render.ThemeNames() {
		render.SetTheme(theme)
		for name, scene := range cursorScenes(t) {
			before := cellUnder(t, blurred(scene()).View())
			v := scene().View()
			under, ok := before.Paper, before.Papered
			if !ok {
				under, ok = render.RGBAOf(v.BackgroundColor)
			}
			if !before.Inked || !ok {
				t.Errorf("%s, %s: the cell has no colour to tint (%+v)", theme, name, before)
				continue
			}
			tint := render.Blend(before.Ink, under, 0.15)
			if c := cellUnder(t, v); c.Ink != before.Ink || !c.Papered || c.Paper != tint {
				t.Errorf("%s, %s: the cell is drawn %v on %v; want %v on %v", theme, name, c.Ink, c.Paper, before.Ink, tint)
			}
			if v.Cursor.Color != tint || v.Cursor.Shape != tea.CursorBar || v.Cursor.Blink {
				t.Errorf("%s, %s: the cursor is %+v; want a steady bar in %v", theme, name, v.Cursor, tint)
			}
			back, _ := blurred(scene()).Update(tea.FocusMsg{})
			if again := back.View(); cellUnder(t, again).Paper != tint || again.Cursor.Shape != tea.CursorBar {
				t.Errorf("%s, %s: focus regained and the cursor is %+v; want the tint back", theme, name, again.Cursor)
			}
		}
	}
}

// TestTheRouterAsksForFocusReports is D-367: the terminal is asked to say
// when it gains and loses focus, on both surfaces.
func TestTheRouterAsksForFocusReports(t *testing.T) {
	r := consoleRouter(t)
	if !r.View().ReportFocus {
		t.Error("the console's frame does not ask for focus reports")
	}
	r.active = SurfaceObserver
	if !r.View().ReportFocus {
		t.Error("the Observer's frame does not ask for focus reports")
	}
}

// TestACursorHandedOutIsNeverChanged is D-366: Bubble Tea compares a frame's
// cursor with the last one's, its colour above all, so a cursor already
// handed out keeps its place and look when the focus moves, and a frame
// drawn again as it was hands out the same one.
func TestACursorHandedOutIsNeverChanged(t *testing.T) {
	m, _ := mapDash(t, Config{}).Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	d := m.(Dashboard).openSetupAt(rowTheme)
	first := d.View().Cursor
	was := *first
	next, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	moved := next.(Dashboard)
	second := moved.View().Cursor
	if *first != was || second == first {
		t.Errorf("the focus moved and the first cursor became %+v (was %+v); want it unchanged and a new one", *first, was)
	}
	if again := moved.View().Cursor; again != second {
		t.Error("a frame drawn again as it was made a new cursor; want the one held")
	}
}
