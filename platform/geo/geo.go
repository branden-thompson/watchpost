// Package geo holds the pure great-circle helpers shared by providers that
// pick stations by proximity (NDBC buoys, NWS observation stations — UAT 60).
package geo

import "math"

// EarthRadiusKM is the mean Earth radius used for haversine distances.
const EarthRadiusKM = 6371.0

// HaversineKM is the great-circle distance between two lat/lon points.
func HaversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLon := toRad(lat2-lat1), toRad(lon2-lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * EarthRadiusKM * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// BearingDeg is the initial great-circle bearing from point 1 to point 2,
// degrees true in [0, 360) — the FIRE rows say which way a hotspot lies (B5).
func BearingDeg(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	φ1, φ2, dλ := toRad(lat1), toRad(lat2), toRad(lon2-lon1)
	y := math.Sin(dλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(dλ)
	deg := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(deg+360, 360)
}
