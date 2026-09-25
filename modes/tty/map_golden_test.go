package tty

// map_golden_test.go — 0.18.0 W2.8 (FR-8.5): the map window's goldens at 80,
// 120 and 133 columns, each carrying the width invariant.

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// checkWidthGolden is checkGolden with the frame's width invariants: no line
// is wider than the terminal, and the window's right border stands in one
// column from its top corner to its bottom one - a map line one cell too wide
// would push it, and the rows beside it would shear.
func checkWidthGolden(t *testing.T, name, frame string, width int) {
	t.Helper()
	if problem := widthProblem(frame, width); problem != "" {
		t.Fatalf("%s: %s", name, problem)
	}
	checkGolden(t, name, frame)
}

// widthProblem is what breaks the invariants, or nothing.
func widthProblem(frame string, width int) string {
	lines := strings.Split(frame, "\n")
	top, border := -1, -1
	for i, line := range lines {
		if w := render.Width(line); w > width {
			return "line " + strconv.Itoa(i) + " is " + strconv.Itoa(w) + " cells wide, the terminal " + strconv.Itoa(width)
		}
		if top < 0 && strings.Contains(line, "┐") && strings.Contains(line, "┌") {
			top, border = i, cellOf(line, '┐')
		}
	}
	if top < 0 {
		return "no window in the frame"
	}
	for i := top + 1; i < len(lines); i++ {
		cells := []rune(padCells(lines[i], border+1))
		if cells[border] == '┘' {
			return ""
		}
		if cells[border] != '│' {
			return "line " + strconv.Itoa(i) + " has " + strconv.QuoteRune(cells[border]) + " where the window's right border stands (column " + strconv.Itoa(border) + ")"
		}
	}
	return "the window has no bottom corner"
}

// TestTheWidthInvariantCatchesAShear: a frame whose window row is one cell
// too wide, or wider than the terminal, is refused; a straight one passes.
func TestTheWidthInvariantCatchesAShear(t *testing.T) {
	straight := "x ┌──┐\n  │ab│\n  └──┘"
	if p := widthProblem(straight, 7); p != "" {
		t.Errorf("a straight window was refused: %s", p)
	}
	if widthProblem("x ┌──┐\n  │abc│\n  └──┘", 7) == "" {
		t.Error("a sheared border passed")
	}
	if widthProblem(straight, 5) == "" {
		t.Error("a frame wider than the terminal passed")
	}
}

// cellOf is the cell column of a rune's first appearance in a plain line.
func cellOf(line string, r rune) int {
	col := 0
	for _, c := range line {
		if c == r {
			return col
		}
		col += render.Width(string(c))
	}
	return -1
}

// padCells is a plain line as one rune per cell, at least n cells long.
func padCells(line string, n int) string {
	var b strings.Builder
	cells := 0
	for _, c := range line {
		b.WriteRune(c)
		cells++
		if render.Width(string(c)) == 2 {
			b.WriteRune(' ')
			cells++
		}
	}
	for ; cells < n; cells++ {
		b.WriteRune(' ')
	}
	return b.String()
}

// TestMapWindowGoldens pins the window at each width the plan names, with
// one alert on the map, every piece of work landed, and the boxes over it as
// they open (UAT-1: Area Alerts, the controls).
func TestMapWindowGoldens(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {120, 40}, {133, 44}} {
		t.Run(strconv.Itoa(size.w), func(t *testing.T) {
			d := goldenDash(t, false)
			d.cfg.NewMap, d.cfg.MapFeed = embeddedMap, boxFeed(-117.6, -117.1, false)
			m, _ := d.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			d = m.(Dashboard)
			m, _ = d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
			d = feedAndSettle(t, m.(Dashboard))
			d.mapPane.disclose = false
			frame := stripANSITest(d.View().Content)
			if strings.IndexFunc(frame, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) < 0 {
				t.Fatal("no map in the frame: this pins nothing")
			}
			checkWidthGolden(t, "map-"+strconv.Itoa(size.w)+"x"+strconv.Itoa(size.h)+".golden", frame, size.w)
		})
	}
}
