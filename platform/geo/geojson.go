package geo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxVertices bounds one hazard's shape.
//
// **It is derived, not chosen** (RT-6). Eighty forecast zones were read from
// the live service, weighted towards Alaska and the marine zones because that
// is where the island chains and fjords are: median 252 positions, 95th
// percentile 7,477, largest 15,194 (AKZ324, forty-one areas). This is a little
// over three times that largest, which leaves room for a zone half again as
// intricate as any measured while still refusing a document that is damage or
// an attack.
//
// **Its blind spot, stated** (INST-5): eighty of roughly three thousand zones,
// deliberately drawn from the worst-shaped end, so the median here is far above
// the national one and the maximum is near the true one. That bias is the right
// way round for a cap - it is derived from the tail it has to survive - but it
// is not a census, and a redraw of the zone map could move it.
const MaxVertices = 50_000

// MaxRings bounds how many rings one shape may hold, across all of its areas.
// **Counting positions was not enough** (RT-1): a document of empty rings
// counted none of them and was held without limit, which is the failure this
// reader exists to prevent arriving by another door. The most rings measured in
// one zone is 122 (AKZ735, an island chain), so this is room for thirty such.
const MaxRings = 4_000

// maxDepth bounds how deeply `coordinates` may nest. GeoJSON needs four
// levels for a MultiPolygon - polygons, rings, positions, numbers - and a
// little room above that is generous.
const maxDepth = 8

// ErrGeometry is what every refusal here wraps, so a caller can tell a shape
// it could not read from a failure to reach the service at all.
var ErrGeometry = errors.New("geometry")

// ReadGeometry reads a GeoJSON geometry object into a shape.
//
// **It streams tokens rather than decoding into `any`.** The interface decode
// path recurses once per array level with no cap of its own, so a hostile
// deeply-nested `coordinates` would overflow the stack and take the process
// down with it (red-team 0.12.0 P4 F2). `domains/globalfeed` has read points
// this way since; this reads whole rings the same way.
//
// An absent or null geometry is not an error. Four alerts in five carry none
// and name zones instead, so it is the ordinary answer.
func ReadGeometry(raw []byte) (Shape, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var outer struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGeometry, err)
	}
	if len(bytes.TrimSpace(outer.Coordinates)) == 0 || string(bytes.TrimSpace(outer.Coordinates)) == "null" {
		return nil, nil
	}
	// Where each level sits differs by type, and reading the wrong depth
	// silently gives the wrong shape - so it is taken from the type, not
	// guessed. **Two types with identical nesting can still group
	// differently**: a LineString's positions are one run, a MultiPoint's are
	// unrelated places, and the JSON is the same shape either way (RT-7).
	l, ok := layoutFor(outer.Type)
	if !ok {
		return nil, fmt.Errorf("%w: no geometry of type %q is drawn here", ErrGeometry, outer.Type)
	}
	return walk(outer.Coordinates, l)
}

// layout says at which bracket depth each of the three levels closes. A level
// that shares a depth with the one inside it closes at the same bracket, which
// is how a bare Point is one position, one ring and one area at once.
type layout struct{ position, ring, area int }

// layoutFor is the table of the six geometry types, as a function rather than
// a map so that nothing package-level is mutable (P10-06).
func layoutFor(kind string) (layout, bool) {
	switch kind {
	case "Point":
		return layout{1, 1, 1}, true // one position, and it is the whole of it
	case "MultiPoint":
		return layout{2, 2, 2}, true // each place stands alone (RT-7)
	case "LineString":
		return layout{2, 1, 1}, true // one run of positions
	case "MultiLineString":
		return layout{3, 2, 2}, true // each line stands alone
	case "Polygon":
		return layout{3, 2, 1}, true // an outline and its holes, together
	case "MultiPolygon":
		return layout{4, 3, 2}, true // areas, each with its own outline and holes
	}
	return layout{}, false
}

// walk streams the coordinate array, gathering each innermost run of positions
// into a ring. It never recurses: the depth is a counter, which is the whole
// point of reading it this way.
func walk(raw json.RawMessage, l layout) (Shape, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var (
		out   Shape
		area  Polygon
		ring  Ring
		nums  []float64
		depth int
		total int
		rings int
	)
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("%w: %v", ErrGeometry, err)
		}
		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '[':
				depth++
				if depth > maxDepth {
					return nil, fmt.Errorf("%w: nested more than %d deep", ErrGeometry, maxDepth)
				}
			case ']':
				// The three levels are closed innermost first, and a type
				// whose levels share a depth closes them all at one bracket.
				//
				// Leaving a position: the two numbers gathered are one point.
				// The `[` that opened it already counted, so inside a position
				// the depth IS l.position.
				if depth == l.position {
					if len(nums) < 2 {
						return nil, fmt.Errorf("%w: a position of %d numbers", ErrGeometry, len(nums))
					}
					p := Point{Lon: nums[0], Lat: nums[1]}
					if p.Lon < -180 || p.Lon > 180 || p.Lat < -90 || p.Lat > 90 {
						return nil, fmt.Errorf("%w: %v is not a place on the world", ErrGeometry, p)
					}
					ring = append(ring, p)
					total++
					if total > MaxVertices {
						return nil, fmt.Errorf("%w: more than %d positions", ErrGeometry, MaxVertices)
					}
					nums = nums[:0]
				}
				// Leaving a ring: keep it, in the order it was read. **A ring
				// with no positions is not geometry** and is refused rather
				// than kept, so that emptiness cannot be used to fill memory
				// while every other count stays at zero (RT-1).
				if depth == l.ring {
					if len(ring) == 0 {
						return nil, fmt.Errorf("%w: a ring with no positions", ErrGeometry)
					}
					rings++
					if rings > MaxRings {
						return nil, fmt.Errorf("%w: more than %d rings", ErrGeometry, MaxRings)
					}
					area = append(area, ring)
					ring = nil
				}
				// Leaving an area: its outline and its holes, kept together.
				// **An area with no rings is the same emptiness one level up**
				// and is refused for the same reason - a door RT-1's fix would
				// not have covered, because this level did not exist then.
				if depth == l.area {
					if len(area) == 0 {
						return nil, fmt.Errorf("%w: an area with no rings", ErrGeometry)
					}
					out = append(out, area)
					area = nil
				}
				depth--
			default:
				return nil, fmt.Errorf("%w: an object where coordinates were expected", ErrGeometry)
			}
		case float64:
			nums = append(nums, v)
			if len(nums) > 3 { // lon, lat, and an elevation some sources add
				return nil, fmt.Errorf("%w: a position of more than three numbers", ErrGeometry)
			}
		default:
			return nil, fmt.Errorf("%w: %T where a coordinate was expected", ErrGeometry, tok)
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("%w: the coordinates end part-way through", ErrGeometry)
	}
	return out, nil
}
