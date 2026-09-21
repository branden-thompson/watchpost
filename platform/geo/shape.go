package geo

// A shape is what a hazard covers: an alert's own polygon, or the outline of a
// forecast zone. It lives here rather than in the domain that fetches it
// because the drawing side must name it too, and neither `modes/` nor
// `platform/` may import a domain (`scripts/lint-imports.sh`).
//
// **Shapes are kept at the detail they arrive in.** The map library rules that
// hosts pass full detail and it simplifies to what a braille dot can show at
// the zoom in hand (its D-16); simplifying here would throw away detail that
// cannot be recovered, and its budget is two million vertices an overlay
// against the twelve thousand of the largest zone measured (MG-6).

// Point is one position, in degrees. Longitude first is the order GeoJSON
// writes it in; the fields are named so that nothing depends on remembering
// which came first.
type Point struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

// Ring is one closed run of positions: a polygon's outline, or one of its
// holes. The service sends the first position again as the last.
type Ring []Point

// Closed reports whether the ring ends where it began, which is what the
// format promises and what a reader that lost bytes would not produce.
func (r Ring) Closed() bool {
	if len(r) < 2 {
		return false
	}
	return r[0] == r[len(r)-1]
}

// Shape is everything one hazard covers: the rings in the order they were
// read. A zone is often several - an outline and its holes, or a coast and its
// islands - and that order is what says which is which, so nothing here sorts
// them.
type Shape []Ring

// Vertices is how many positions the shape holds in all. It is the figure the
// map library's own budget is counted in, and the one worth logging when a
// shape looks unreasonable.
func (s Shape) Vertices() int {
	n := 0
	for _, r := range s {
		n += len(r)
	}
	return n
}

// Empty reports whether there is nothing to draw. **Most alerts are empty
// here**: four in five carry no polygon of their own and name zones instead,
// so this is the ordinary case and not a failure.
func (s Shape) Empty() bool {
	return s.Vertices() == 0
}
