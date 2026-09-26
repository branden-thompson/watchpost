package geo

// box.go — a box in degrees: a state's extent over its cities (the map's
// title, 0.18.0 D-64) and the map's view (Alerts in view, D-66). ONE OWNER:
// the two were written apart and the dupes gate found them identical.

// Box is a box in degrees, west, south, east and north.
type Box struct{ W, S, E, N float64 }

// Contains reports whether a place is inside the box, its edges included.
func (b Box) Contains(lat, lon float64) bool {
	return lat >= b.S && lat <= b.N && lon >= b.W && lon <= b.E
}
