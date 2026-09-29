package tty

// map_same_test.go — 0.18.0 D-136 (and UAT-2 U2-13): an overlay the window
// already handed in is not handed in again. A missing value is NaN, and NaN
// is never equal to itself, so a wind grid whose gusts say "none here" was
// new at every answer, handed in again, and blinked.

import (
	"math"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestAGridWithMissingValuesIsTheSameGridAgain is D-136 with U2-13: two
// answers with the same grid - its missing values, its gusts said nowhere -
// are the same overlay; one that differs anywhere is not.
func TestAGridWithMissingValuesIsTheSameGridAgain(t *testing.T) {
	grid := func(gust float64) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 1, Values: []float64{15, math.NaN()}, Gusts: []float64{gust, math.NaN()}}
		return tuimaps.WindGrid("wind/us/now", g, []float64{270, math.NaN()}, tuimaps.MilesPerHour, time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC))
	}
	if !SameOverlay(grid(math.NaN()), grid(math.NaN())) {
		t.Error("the same grid, its missing values missing in both, is new")
	}
	if !SameOverlay(grid(30), grid(30)) {
		t.Error("the same grid with a gust is new")
	}
	if SameOverlay(grid(30), grid(math.NaN())) || SameOverlay(grid(30), grid(31)) {
		t.Error("grids whose gusts differ are the same")
	}
	a, b := grid(30), grid(30)
	b.ID = "wind/us/d1"
	if SameOverlay(a, b) {
		t.Error("overlays that differ outside the grid are the same")
	}
}
