package tty

// cursor_test.go — F-205 AX-3 (FR-7.4, D-362, A-107 (12)): the terminal
// cursor stands at every focus point of both surfaces, steady, found in the
// frame as drawn.

import (
	"image/color"
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

// TestTheCursorIsSteady is D-362: the cursor is a block that does not blink,
// so it adds no motion, Reduce motion or not.
func TestTheCursorIsSteady(t *testing.T) {
	for _, reduce := range []bool{false, true} {
		d := mapDash(t, Config{ReduceMotion: reduce})
		m, _ := d.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
		v := m.(Dashboard).openSetupAt(rowTheme).View()
		if v.Cursor == nil || v.Cursor.Blink || v.Cursor.Shape != tea.CursorBlock {
			t.Errorf("Reduce motion %v: the cursor is %+v; want a steady block", reduce, v.Cursor)
		}
	}
}

// inkUnder is the colour the frame draws the cell under its cursor in.
func inkUnder(v tea.View) (color.Color, bool) {
	lines := strings.Split(v.Content, "\n")
	if v.Cursor == nil || v.Cursor.Y < 0 || v.Cursor.Y >= len(lines) {
		return nil, false
	}
	ink, inked := render.InkAt(lines[v.Cursor.Y], v.Cursor.X)
	return ink, inked
}

// TestTheCursorWearsTheColourItCovers is D-366: the cursor takes the colour
// the frame draws its cell in, so Settings' yellow pointer stays yellow under
// it - the dashboard's row, Settings' row and the console's too - and with
// colour off it takes the terminal's own.
func TestTheCursorWearsTheColourItCovers(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m, _ := mapDash(t, Config{}).Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	settings := m.(Dashboard).openSetupAt(rowTheme).View()
	yellow, _ := render.InkAt(render.Tint(">", render.Tok(render.ListPointer)), 0)
	if settings.Cursor == nil || settings.Cursor.Color != yellow {
		t.Errorf("on Settings' focused row the cursor is %+v; want the pointer's %v", settings.Cursor, yellow)
	}
	m, _ = mapDash(t, Config{}).Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	for name, v := range map[string]tea.View{"Settings": settings, "the dashboard": m.(Dashboard).View(), "the console": consoleRouter(t).View()} {
		ink, inked := inkUnder(v)
		if !inked || v.Cursor.Color != ink {
			t.Errorf("on %s the cursor is %+v; want the colour of its cell, %v", name, v.Cursor, ink)
		}
	}
	rendering.SetColorEnabledForTest(false)
	if v := m.(Dashboard).openSetupAt(rowTheme).View(); v.Cursor == nil || v.Cursor.Color != nil {
		t.Errorf("with colour off the cursor is %+v; want the terminal's own colour", v.Cursor)
	}
}

// TestACursorHandedOutIsNeverChanged is D-366: Bubble Tea compares a frame's
// cursor with the last one's, its colour above all, so a cursor already
// handed out keeps its place and colour when the focus moves, and a frame
// where nothing moved hands out the same one.
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
		t.Error("a frame where nothing moved made a new cursor; want the one held")
	}
}
