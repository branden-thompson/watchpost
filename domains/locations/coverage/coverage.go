// Package coverage answers one question for the location resolvers: does
// the National Weather Service serve this place? Both the offline index
// (domains/locations) and the online geocoder (domains/locations/openmeteo)
// need the same answer, and neither may import the other — so it lives
// here, and the two paths cannot disagree about a place.
package coverage

// NWS reports whether NWS forecasts a GeoNames country code: the states
// plus the territories with NWS offices.
func NWS(country string) bool {
	switch country {
	case "US", "PR", "VI", "GU", "AS", "MP":
		return true
	}
	return false
}

// Outside is the user-facing refusal for a place NWS does not serve.
func Outside(place string) string {
	return place + " is outside NWS coverage — Watchpost 0.9 is US-only (a global provider is planned for 1.0)"
}
