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
func (l Lattice) Interpolate(values []float64) Field { return l.interpolate(values, false) }

// InterpolateWide is Interpolate for NDFD's temperature, feels like and wind
// (D-201, UAT-2 U2-50): a cell is missing only where all four of its points
// are. Land beside a point over the sea, Mexico or Canada - where NDFD
// answers nothing, and the nearest rule would leave the Florida panhandle,
// Big Bend and Boston blank - is weighed from the points that answered; past
// the border it runs at most one cell of NDFD's dense lattice.
func (l Lattice) InterpolateWide(values []float64) Field { return l.interpolate(values, true) }

// fine is how many cells a lattice cell is split into each way: Fine, or half
// as many for a lattice past twice MaxPoints (NDFD's, D-201) - so a dense
// lattice's field has about as many cells as a coarse one's.
func (l Lattice) fine() int {
	if l.Cols*l.Rows > 2*MaxPoints {
		return Fine / 2
	}
	return Fine
}

func (l Lattice) interpolate(values []float64, wide bool) Field {
	if l.Cols < 2 || l.Rows < 2 || len(values) != l.Cols*l.Rows {
		return Field{}
	}
	fine := l.fine()
	cols, rows := (l.Cols-1)*fine, (l.Rows-1)*fine
	out := Field{Box: l.Box, Cols: cols, Rows: rows, Values: make([]float64, cols*rows)}
	for r := range rows {
		y := (float64(r) + 0.5) / float64(fine) // in lattice rows from the north
		r0 := min(int(y), l.Rows-2)
		fy := y - float64(r0)
		for c := range cols {
			x := (float64(c) + 0.5) / float64(fine)
			c0 := min(int(x), l.Cols-2)
			fx := x - float64(c0)
			out.Values[r*cols+c] = math.NaN()
			nr, nc := r0+int(math.Round(fy)), c0+int(math.Round(fx))
			if !wide && math.IsNaN(values[nr*l.Cols+nc]) {
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

// Resample is values on this lattice put on another's points, each weighed
// from the four around it under InterpolateWide's rule (D-201): Open-Meteo's
// day put on NDFD's denser lattice, where a day NDFD left empty is filled
// (D-189). A point outside this lattice's box is missing; the same lattice is
// a copy.
func (l Lattice) Resample(values []float64, to Lattice) []float64 {
	out := make([]float64, to.Cols*to.Rows)
	if l == to && len(values) == len(out) {
		copy(out, values)
		return out
	}
	for i, p := range to.Points() { // the other lattice's points (P10-02)
		out[i] = math.NaN()
		if l.Cols < 2 || l.Rows < 2 || len(values) != l.Cols*l.Rows || !l.Box.Contains(p.Lat, p.Lon) {
			continue
		}
		x := (p.Lon - l.Box.W) / (l.Box.E - l.Box.W) * float64(l.Cols-1)
		y := (l.Box.N - p.Lat) / (l.Box.N - l.Box.S) * float64(l.Rows-1)
		c0, r0 := min(int(x), l.Cols-2), min(int(y), l.Rows-2)
		fx, fy := x-float64(c0), y-float64(r0)
		sum, weight := 0.0, 0.0
		for _, k := range [4]struct {
			dr, dc int
			w      float64
		}{{0, 0, (1 - fx) * (1 - fy)}, {0, 1, fx * (1 - fy)}, {1, 0, (1 - fx) * fy}, {1, 1, fx * fy}} {
			if v := values[(r0+k.dr)*l.Cols+c0+k.dc]; !math.IsNaN(v) && k.w > 0 {
				sum, weight = sum+v*k.w, weight+k.w
			}
		}
		if weight > 0 {
			out[i] = sum / weight
		}
	}
	return out
}

// ResampleNearest is Resample for what cannot be blended - a direction, whose
// 350 and 10 degrees would average to 180: each point takes its nearest
// point's value (D-201).
func (l Lattice) ResampleNearest(values []float64, to Lattice) []float64 {
	out := make([]float64, to.Cols*to.Rows)
	for i, p := range to.Points() { // the other lattice's points (P10-02)
		out[i] = math.NaN()
		if l.Cols < 2 || l.Rows < 2 || len(values) != l.Cols*l.Rows || !l.Box.Contains(p.Lat, p.Lon) {
			continue
		}
		c := int(math.Round((p.Lon - l.Box.W) / (l.Box.E - l.Box.W) * float64(l.Cols-1)))
		r := int(math.Round((l.Box.N - p.Lat) / (l.Box.N - l.Box.S) * float64(l.Rows-1)))
		out[i] = values[min(max(r, 0), l.Rows-1)*l.Cols+min(max(c, 0), l.Cols-1)]
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
	return l.interpolateWind(speed, from, false)
}

// InterpolateWindWide is InterpolateWind under InterpolateWide's rule (D-201):
// NDFD's wind, blank only where all four points are.
func (l Lattice) InterpolateWindWide(speed, from []float64) (Field, []float64) {
	return l.interpolateWind(speed, from, true)
}

func (l Lattice) interpolateWind(speed, from []float64, wide bool) (Field, []float64) {
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
	e, no := l.interpolate(east, wide), l.interpolate(north, wide)
	sp := l.interpolate(speed, wide) // the speed interpolated itself: averaging vectors would slow a wind that turns
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
