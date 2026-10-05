package temperature

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// totalsLattice is ndfd-totals.xml's 3x2 lattice: the Alaska Range.
var totalsLattice = Lattice{Name: "totals", Box: geo.Box{W: -150, S: 63, E: -146, N: 63.5}, Cols: 3, Rows: 2}

// totalsCaptured is when ndfd-totals.xml was captured: 15:34 in Alaska.
var totalsCaptured = time.Date(2026, 9, 30, 23, 34, 38, 0, time.UTC)

// NDFD'S TOTALS ARE EACH DAY'S SUM (W18.5, D-168): its six-hour rain,
// liquid-equivalent, in mm and its snow in cm, each period counted on the
// local date it starts - today's from the period the answer begins with - a
// day none reaches missing, never dry; and a day with periods all zero is
// dry, not missing.
func TestNDFDsTotalsAreEachDaysSum(t *testing.T) {
	var asked []string
	got, err := NewNDFD(fixedGet{t: t, name: "ndfd-totals.xml", asked: &asked}, "").Totals(context.Background(), totalsLattice, totalsCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || !strings.Contains(asked[0], "qpf=qpf") || !strings.Contains(asked[0], "snow=snow") {
		t.Errorf("asked %v; want one ask for qpf and snow", asked)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-6 }
	for _, c := range []struct {
		what      string
		got, want float64
	}{
		{"today's rain at 63.5N 146W", got.QPF[0][2], (0.09 + 0.11 + 0.17) * 25.4},
		{"today's snow at 63.5N 146W", got.Snow[0][2], (1.42 + 1.61 + 2.80) * 2.54},
		{"tomorrow's rain at 63.5N 146W", got.QPF[1][2], (0.19 + 0.14 + 0.13 + 0.03) * 25.4},
		{"tomorrow's snow at 63.5N 146W", got.Snow[1][2], (3.19 + 2.52 + 2.28 + 0.51) * 2.54},
		{"the third day's rain at 63N 146W, two periods", got.QPF[3][5], 0.01 * 25.4},
		{"today's snow at 63N 150W, none", got.Snow[0][3], 0},
	} {
		if !near(c.got, c.want) {
			t.Errorf("%s is %v; want %v", c.what, c.got, c.want)
		}
	}
	for k := 4; k < Days; k++ {
		if !math.IsNaN(got.QPF[k][0]) || !math.IsNaN(got.Snow[k][0]) {
			t.Errorf("day %d, past NDFD's reach, is %v / %v; want missing", k, got.QPF[k][0], got.Snow[k][0])
		}
	}
}
