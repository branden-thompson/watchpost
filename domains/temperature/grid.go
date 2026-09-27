package temperature

import (
	"math"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// Fine is how many grid cells a lattice cell is split into each way: the map
// library draws a grid cell by cell, nearest, so a lattice handed in as it is
// would draw in blocks a degree and a half across.
const Fine = 8

// Field is values over a box, rows from the north, each west to east - the
// shape the map library draws. Missing is NaN.
type Field struct {
	Box        geo.Box
	Cols, Rows int
	Values     []float64
}

// Interpolate is the lattice's values spread over the box: each cell's value
// bilinear from the four points around its centre. A point with no value is
// left out and the others weighed up to one, so the land between an inland
// point and a missing offshore one keeps its temperature; a cell with no
// point around it at all is missing, never filled from farther away.
func (l Lattice) Interpolate(values []float64) Field {
	if l.Cols < 2 || l.Rows < 2 || len(values) != l.Cols*l.Rows {
		return Field{}
	}
	cols, rows := (l.Cols-1)*Fine, (l.Rows-1)*Fine
	out := Field{Box: l.Box, Cols: cols, Rows: rows, Values: make([]float64, cols*rows)}
	for r := range rows {
		y := (float64(r) + 0.5) / Fine // in lattice rows from the north
		r0 := min(int(y), l.Rows-2)
		fy := y - float64(r0)
		for c := range cols {
			x := (float64(c) + 0.5) / Fine
			c0 := min(int(x), l.Cols-2)
			fx := x - float64(c0)
			sum, weight := 0.0, 0.0
			for _, k := range [4]struct {
				dr, dc int
				w      float64
			}{{0, 0, (1 - fx) * (1 - fy)}, {0, 1, fx * (1 - fy)}, {1, 0, (1 - fx) * fy}, {1, 1, fx * fy}} {
				v := values[(r0+k.dr)*l.Cols+c0+k.dc]
				if math.IsNaN(v) || k.w == 0 {
					continue
				}
				sum, weight = sum+v*k.w, weight+k.w
			}
			out.Values[r*cols+c] = math.NaN()
			if weight > 0 {
				out.Values[r*cols+c] = sum / weight
			}
		}
	}
	return out
}
