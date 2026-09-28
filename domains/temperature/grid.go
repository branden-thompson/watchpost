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
// bilinear from the four points around its centre. A SOURCE IS DRAWN ONLY AS
// FAR AS ITS OWN POINTS REACH (D-101): a cell whose nearest point has no value
// is missing - extrapolating from a farther one left square patches past
// NDFD's grid (UAT-2 U2-17) - and otherwise a point with no value is left out
// and the others weighed up to one.
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
			out.Values[r*cols+c] = math.NaN()
			nr, nc := r0+int(math.Round(fy)), c0+int(math.Round(fx))
			if math.IsNaN(values[nr*l.Cols+nc]) {
				continue // its nearest point has nothing: past the source's reach
			}
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
			if weight > 0 {
				out.Values[r*cols+c] = sum / weight
			}
		}
	}
	return out
}

// InterpolateOut is Interpolate for a field its drawing keeps to its own
// ground - the waves, which the map draws over the sea alone (UAT-2 U2-31):
// a point with no value first takes the mean of its neighbours that have
// one, round after round, so every cell by the coast has the sea's value
// and none is blank because its nearest point was ashore. Nothing anywhere
// stays nothing.
func (l Lattice) InterpolateOut(values []float64) Field {
	if l.Cols < 2 || l.Rows < 2 || len(values) != l.Cols*l.Rows {
		return Field{}
	}
	v := append([]float64(nil), values...)
	for changed := true; changed; {
		changed = false
		next := append([]float64(nil), v...)
		for r := range l.Rows {
			for c := range l.Cols {
				if !math.IsNaN(v[r*l.Cols+c]) {
					continue
				}
				sum, n := 0.0, 0
				for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					rr, cc := r+d[0], c+d[1]
					if rr >= 0 && rr < l.Rows && cc >= 0 && cc < l.Cols && !math.IsNaN(v[rr*l.Cols+cc]) {
						sum, n = sum+v[rr*l.Cols+cc], n+1
					}
				}
				if n > 0 {
					next[r*l.Cols+c], changed = sum/float64(n), true
				}
			}
		}
		v = next
	}
	return l.Interpolate(v)
}

// InterpolateWind is a lattice's wind spread over the box as Interpolate
// spreads its temperatures: speed and the direction it blows from. THE
// DIRECTION IS INTERPOLATED AS A VECTOR, by its east and north parts - as a
// number, 359 degrees and 1 would meet at 180, the wind turned round.
func (l Lattice) InterpolateWind(speed, from []float64) (Field, []float64) {
	n := len(speed)
	if len(from) != n {
		return Field{}, nil
	}
	east, north := make([]float64, n), make([]float64, n)
	for i := range speed {
		if math.IsNaN(speed[i]) || math.IsNaN(from[i]) {
			east[i], north[i] = math.NaN(), math.NaN()
			continue
		}
		rad := from[i] * math.Pi / 180
		east[i], north[i] = speed[i]*math.Sin(rad), speed[i]*math.Cos(rad) // the from-vector: its angle is the direction's
	}
	e, no := l.Interpolate(east), l.Interpolate(north)
	sp := l.Interpolate(speed) // the speed interpolated itself: averaging vectors would slow a wind that turns
	dirs := make([]float64, len(sp.Values))
	for i := range dirs {
		if math.IsNaN(e.Values[i]) || math.IsNaN(no.Values[i]) || math.IsNaN(sp.Values[i]) {
			dirs[i], sp.Values[i] = math.NaN(), math.NaN()
			continue
		}
		dirs[i] = math.Mod(math.Atan2(e.Values[i], no.Values[i])*180/math.Pi+360, 360)
	}
	return sp, dirs
}
