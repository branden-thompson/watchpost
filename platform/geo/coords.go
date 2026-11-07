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
	if !OnEarth(v[0], v[1]) {
		return LatLon{}, false
	}
	return LatLon{Lat: v[0], Lon: v[1]}, true
}

// OnEarth reports whether a latitude and longitude name a place on Earth:
// within -90 to 90 and -180 to 180, the edges included. Written so that NaN,
// which fails every comparison, is refused too, as an infinity is (D-248). The
// one test of it for every feed and parser (0.19.5 DR-9).
func OnEarth(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
