package geo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxVertices bounds one hazard's shape. The largest forecast zone measured is
// twelve thousand positions (Glacier Bay, thirty-two rings); a document
// claiming far more than that is damage or an attack, and neither is drawn.
const MaxVertices = 50_000

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
	// How deep the positions sit differs by type, and reading the wrong depth
	// silently gives the wrong shape - so it is taken from the type, not guessed.
	var want int
	switch outer.Type {
	case "Point":
		want = 1
	case "MultiPoint", "LineString":
		want = 2
	case "Polygon", "MultiLineString":
		want = 3
	case "MultiPolygon":
		want = 4
	default:
		return nil, fmt.Errorf("%w: no geometry of type %q is drawn here", ErrGeometry, outer.Type)
	}
	return walk(outer.Coordinates, want)
}

// walk streams the coordinate array, gathering each innermost run of positions
// into a ring. It never recurses: the depth is a counter, which is the whole
// point of reading it this way.
func walk(raw json.RawMessage, positionsAt int) (Shape, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var (
		out   Shape
		ring  Ring
		nums  []float64
		depth int
		total int
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
				// Leaving a position: the two numbers gathered are one point.
				// The `[` that opened it already counted, so inside a position
				// the depth IS positionsAt.
				if depth == positionsAt {
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
				// Leaving a ring: keep it, in the order it was read.
				if depth == positionsAt-1 {
					out = append(out, ring)
					ring = nil
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
	// A Point has no ring of its own; what was gathered is the whole of it.
	if positionsAt == 1 && len(ring) > 0 {
		out = append(out, ring)
	}
	if depth != 0 {
		return nil, fmt.Errorf("%w: the coordinates end part-way through", ErrGeometry)
	}
	return out, nil
}
