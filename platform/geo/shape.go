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
// against the fifteen thousand of the largest zone measured (MG-6, RT-6).

// Point is one position, in degrees. Longitude first is the order GeoJSON
// writes it in; the fields are named so that nothing depends on remembering
// which came first.
type Point struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

// LonLat says where this is, in plain numbers.
//
// **It is what lets the map library take these points as they are.** That
// library asks a host's point to SAY its position rather than matching on how
// it is laid out - which is what makes this work at all, because a type
// carrying struct tags is a different type from one without, and these fields
// are tagged because they are also serialised. Nothing of the library appears
// in this signature, so this package still names no renderer.
func (p Point) LonLat() (lon, lat float64) { return p.Lon, p.Lat }

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

// Polygon is one filled area: its outline first, then any holes in it.
//
// **The grouping is the whole point of this type.** What draws a shape reads
// the first ring as the outline and every ring after it as a hole, so rings
// from different areas cannot share a list. A zone that is thirty-two separate
// islands, flattened, becomes one island with thirty-one holes punched in it -
// and the flattening cannot be undone afterwards, because an island and a hole
// are the same list of positions (RT-8).
type Polygon []Ring

// Vertices is how many positions this one area holds, outline and holes alike.
func (p Polygon) Vertices() int {
	n := 0
	for _, r := range p {
		n += len(r)
	}
	return n
}

// Shape is everything one hazard covers: one area, or several. A county is
// one; a coast is the mainland and each of its islands, each standing alone.
// Nothing here sorts them - within an area the order says which ring is the
// outline, and across areas the order is the service's own.
type Shape []Polygon

// Vertices is how many positions the shape holds in all. It is the figure the
// map library's own budget is counted in, and the one worth logging when a
// shape looks unreasonable.
func (s Shape) Vertices() int {
	n := 0
	for _, p := range s {
		n += p.Vertices()
	}
	return n
}

// Rings is how many rings the shape holds across all its areas. It is what the
// reader's own cap is counted in, and it is not the same number as the count
// of areas - which is the distinction RT-8 turned on.
func (s Shape) Rings() int {
	n := 0
	for _, p := range s {
		n += len(p)
	}
	return n
}

// Empty reports whether there is nothing to draw. **Most alerts are empty
// here**: four in five carry no polygon of their own and name zones instead,
// so this is the ordinary case and not a failure.
func (s Shape) Empty() bool {
	return s.Vertices() == 0
}

// Area is a hazard's ground, and whether all of it is known.
//
// **A shape built from two of a hazard's nine parts and a shape built from all
// nine are the same Shape**, and that is the distinction this type exists to
// keep. An alert is matched to a place by the full list of areas it names, and
// drawn from the parts that could be fetched - so if the part that failed is
// the listener's own, a map would shade up to their county line and stop, and
// they would read that as "not me".
//
// Whether to draw a partly-known area is a question for whatever has a view to
// answer it with, and nothing here has one (MG-10). That decision is deferred;
// **the information it needs is not**, which is what was wrong before: the
// missing ids were fetched, reported by the store, and then dropped by its
// only caller.
type Area struct {
	Shape Shape
	// Missing names the parts that could not be got, the way the source names
	// them - a zone id, so that a description can say which place is unknown
	// rather than only that something is.
	Missing []string
}

// Complete reports whether every part of the hazard's ground is here.
func (a Area) Complete() bool { return len(a.Missing) == 0 }

// Empty reports whether there is nothing to draw. An area may be empty AND
// incomplete - nothing was got - or empty and complete, which is the ordinary
// alert that names no ground at all.
func (a Area) Empty() bool { return a.Shape.Empty() }
