package geo

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// LatLon is a place on the globe, in degrees.
type LatLon struct{ Lat, Lon float64 }

// coordinatePart is how one half of a coordinate pair may be written: a
// sign, digits and a point. strconv would also take "NaN", "Inf", exponents
// and hex floats, none of which a place is typed as (FR-4.9).
var coordinatePart = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)$`)

// ParseCoordinates reads a typed latitude and longitude in decimal degrees,
// "33.2, -117.38" (a comma, spaces, or both between them): false unless both
// are finite numbers on the globe (FR-4.9).
func ParseCoordinates(s string) (LatLon, bool) {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(fields) != 2 {
		return LatLon{}, false
	}
	var v [2]float64
	for i, f := range fields {
		if !coordinatePart.MatchString(f) {
			return LatLon{}, false
		}
		x, err := strconv.ParseFloat(f, 64)
		if err != nil || math.IsNaN(x) || math.IsInf(x, 0) {
			return LatLon{}, false
		}
		v[i] = x
	}
	if v[0] < -90 || v[0] > 90 || v[1] < -180 || v[1] > 180 {
		return LatLon{}, false
	}
	return LatLon{Lat: v[0], Lon: v[1]}, true
}
