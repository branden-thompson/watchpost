package geo

import "testing"

// TestARingKnowsWhetherItIsClosed is the shape of the thing every other test
// here leans on. A ring the service sends is closed — its last position repeats
// its first — and a ring that is not closed is a ring we did not read properly.
func TestARingKnowsWhetherItIsClosed(t *testing.T) {
	closed := Ring{{Lon: -85, Lat: 41}, {Lon: -84, Lat: 41}, {Lon: -84, Lat: 42}, {Lon: -85, Lat: 41}}
	if !closed.Closed() {
		t.Error("a ring whose last position repeats its first is not closed")
	}
	open := Ring{{Lon: -85, Lat: 41}, {Lon: -84, Lat: 41}, {Lon: -84, Lat: 42}}
	if open.Closed() {
		t.Error("a ring whose ends differ is closed")
	}
	if (Ring{}).Closed() {
		t.Error("an empty ring is closed")
	}
}

// TestAShapeKeepsItsAreasAndTheirRingsInOrder matters because **order carries
// meaning at both levels**: within an area the first ring is the outline and
// the rest are its holes, and across areas the order is the service's own.
// Reordering either changes what is drawn.
func TestAShapeKeepsItsAreasAndTheirRingsInOrder(t *testing.T) {
	s := Shape{
		{
			{{Lon: -85, Lat: 41}, {Lon: -84, Lat: 41}, {Lon: -84, Lat: 42}, {Lon: -85, Lat: 41}},                 // outline
			{{Lon: -84.8, Lat: 41.2}, {Lon: -84.6, Lat: 41.2}, {Lon: -84.6, Lat: 41.4}, {Lon: -84.8, Lat: 41.2}}, // its hole
		},
		{{{Lon: -80, Lat: 35}, {Lon: -79, Lat: 35}, {Lon: -79, Lat: 36}, {Lon: -80, Lat: 35}}}, // separate land
	}
	if len(s) != 2 {
		t.Fatalf("a shape of two areas has %d", len(s))
	}
	if len(s[0]) != 2 || len(s[1]) != 1 {
		t.Errorf("the areas hold %d and %d rings; they were given two and one", len(s[0]), len(s[1]))
	}
	if s[0][0][0].Lon != -85 {
		t.Errorf("the first area's outline is not the ring given first: %v", s[0][0][0])
	}
	if got := s.Vertices(); got != 12 {
		t.Errorf("three four-position rings count %d vertices", got)
	}
	if got := s.Rings(); got != 3 {
		t.Errorf("two areas of two rings and one count %d rings", got)
	}
}

// TestTheZeroShapeIsUsable: a zone-only alert has no shape at all, and every
// caller here has to be able to hold that without a nil check at each use.
func TestTheZeroShapeIsUsable(t *testing.T) {
	var s Shape
	if s.Vertices() != 0 {
		t.Errorf("the zero shape counts %d vertices", s.Vertices())
	}
	if !s.Empty() {
		t.Error("the zero shape is not empty")
	}
	full := Shape{{{{Lon: 1, Lat: 2}}}}
	if full.Empty() {
		t.Error("a shape with a position in it is empty")
	}
}
