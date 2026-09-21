package geo

import (
	"strings"
	"testing"
)

// TestReadGeometryReadsTheThreeShapesTheServiceSends. A hazard arrives as a
// Point, a Polygon or a MultiPolygon, and absent altogether for the four alerts
// in five that name zones instead.
func TestReadGeometryReadsTheThreeShapesTheServiceSends(t *testing.T) {
	for _, c := range []struct {
		what  string
		body  string
		rings int
		verts int
	}{
		{"a point", `{"type":"Point","coordinates":[-85.1,41.1]}`, 1, 1},
		{"a polygon", `{"type":"Polygon","coordinates":[[[-85,41],[-84,41],[-84,42],[-85,41]]]}`, 1, 4},
		{"a polygon with a hole", `{"type":"Polygon","coordinates":[[[-85,41],[-84,41],[-84,42],[-85,41]],[[-84.8,41.2],[-84.7,41.2],[-84.7,41.3],[-84.8,41.2]]]}`, 2, 8},
		{"a multi-polygon", `{"type":"MultiPolygon","coordinates":[[[[-85,41],[-84,41],[-84,42],[-85,41]]],[[[-80,35],[-79,35],[-79,36],[-80,35]]]]}`, 2, 8},
	} {
		got, err := ReadGeometry([]byte(c.body))
		if err != nil {
			t.Errorf("%s: %v", c.what, err)
			continue
		}
		if len(got) != c.rings || got.Vertices() != c.verts {
			t.Errorf("%s: %d rings and %d vertices; want %d and %d", c.what, len(got), got.Vertices(), c.rings, c.verts)
		}
	}
}

// TestAnAbsentGeometryIsNotAnError is the ordinary case, not the exception:
// most alerts carry no geometry at all and must not be read as broken.
func TestAnAbsentGeometryIsNotAnError(t *testing.T) {
	for _, body := range []string{"", "null", `{"type":"Polygon","coordinates":null}`} {
		got, err := ReadGeometry([]byte(body))
		if err != nil {
			t.Errorf("%q was read as an error: %v", body, err)
		}
		if !got.Empty() {
			t.Errorf("%q gave %d vertices", body, got.Vertices())
		}
	}
}

// TestADeeplyNestedGeometryIsRefused is the attack this reader exists to
// survive. The interface decode path recurses once per array level with no
// depth cap, so a hostile `coordinates` would overflow the stack and take the
// process with it (red-team 0.12.0 P4 F2, the reason `geoPoint` streams
// tokens). **It must be refused, not survived by luck.**
func TestADeeplyNestedGeometryIsRefused(t *testing.T) {
	deep := `{"type":"Polygon","coordinates":` + strings.Repeat("[", 20000) + strings.Repeat("]", 20000) + `}`
	if _, err := ReadGeometry([]byte(deep)); err == nil {
		t.Error("twenty thousand levels of nesting were accepted")
	}
}

// TestDamagedGeometryIsAnError, never a guess and never a panic: a position of
// one number, a truncated document, a coordinate that is not a number.
func TestDamagedGeometryIsAnError(t *testing.T) {
	for _, body := range []string{
		`{"type":"Polygon","coordinates":[[[-85],[-84,41]]]}`,
		`{"type":"Polygon","coordinates":[[[-85,41],[-84`,
		`{"type":"Polygon","coordinates":[[["west","north"]]]}`,
		`{"type":"Polygon","coordinates":[[[-181,41],[-84,41]]]}`,
		`{"type":"Polygon","coordinates":[[[-85,91],[-84,41]]]}`,
	} {
		if _, err := ReadGeometry([]byte(body)); err == nil {
			t.Errorf("%.48q was accepted", body)
		}
	}
}

// TestAHugeGeometryIsRefusedRatherThanHeld bounds what one hazard may cost.
// The largest zone measured is twelve thousand positions; a document claiming
// far more is either damage or an attack, and either way it is not drawn.
func TestAHugeGeometryIsRefusedRatherThanHeld(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"type":"Polygon","coordinates":[[`)
	for i := range MaxVertices + 10 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("[-85,41]")
	}
	b.WriteString("]]}")
	if _, err := ReadGeometry([]byte(b.String())); err == nil {
		t.Errorf("a geometry of more than %d positions was accepted", MaxVertices)
	}
}
