package app

import "github.com/branden-thompson/watchpost/modes/tty"

// endpoints.go — which host each provider talks to (0.14.0).
//
// [S] shows one row per ENDPOINT with the provider(s) that use it, because the
// request counters are per host: a host has exactly one set of them and a
// provider does not. Several providers share a host — nws and nws-marine are
// both api.weather.gov — so keying the table the other way would either repeat
// the same numbers on two rows or invent an attribution that is not in the data.
//
// WRITTEN DOWN, not derived. The snapshot's provider status carries no host, and
// the attribution strings cannot supply one: FIRMS credits earthdata.nasa.gov
// while its API is firms.modaps.eosdis.nasa.gov, so a table built by parsing
// them would be confidently wrong about the one provider a listener is most
// likely to be debugging.
//
// TestEveryProviderHasAnEndpoint pins it against the registered set, so a
// provider added without a host here fails rather than showing a blank column.

// providerEndpoints is the map, one entry per registered provider id.
func providerEndpoints() map[string][]string {
	return map[string][]string{
		"nws":        {"api.weather.gov"},
		"nws-marine": {"api.weather.gov"},
		"ndbc":       {"www.ndbc.noaa.gov"},
		"coops":      {"api.tidesandcurrents.noaa.gov"},
		"coops-obs":  {"api.tidesandcurrents.noaa.gov"},
		"hms":        {"www.ospo.noaa.gov"},
		"wfigs":      {"services3.arcgis.com"},
		"firms":      {"firms.modaps.eosdis.nasa.gov"},
		"usgs":       {"earthquake.usgs.gov"},
	}
}

// providerSources are what each provider serves and its badge (0.19.0
// D-170): the Status window groups its hosts by them and names each by its
// badge, About's chip for the source. TestEveryStatusSourceIsGroupedAndCredited
// holds every provider to one, and every badge to About's credits.
func providerSources() map[string]tty.StatusSource {
	return map[string]tty.StatusSource{
		"nws":        {Feature: tty.FeatureWeather, Badge: "NWS"},
		"nws-marine": {Feature: tty.FeatureWeather, Badge: "NWS"},
		"ndbc":       {Feature: tty.FeatureWeather, Badge: "NDBC"},
		"coops":      {Feature: tty.FeatureWeather, Badge: "CO-OPS"},
		"coops-obs":  {Feature: tty.FeatureWeather, Badge: "CO-OPS"},
		"hms":        {Feature: tty.FeatureAdverse, Badge: "HMS"},
		"wfigs":      {Feature: tty.FeatureAdverse, Badge: "NIFC"},
		"firms":      {Feature: tty.FeatureAdverse, Badge: "FIRMS"},
		"usgs":       {Feature: tty.FeatureAdverse, Badge: "USGS"},
	}
}

// hostSources are the hosts the station's client asks that no provider
// claims - the ticker's storm feed, the geocoder - with what each serves and
// its badge (D-170). A host the window counts that is named nowhere is drawn
// under OTHER DATA.
func hostSources() map[string]tty.StatusSource {
	return map[string]tty.StatusSource{
		"www.nhc.noaa.gov":             {Feature: tty.FeatureAdverse, Badge: "NHC"},
		"geocoding-api.open-meteo.com": {Feature: tty.FeatureWeather, Badge: "O-METEO"},
	}
}

// mapBadges are the map's sources' badges, by the name each is listed under
// (D-170): About's chip for the source.
var mapBadges = map[string]string{
	"OpenFreeMap": "OFM", "National Weather Service": "NWS", "USGS": "USGS",
	"NOAA / NCEP (MRMS)": "MRMS", "Iowa Environmental Mesonet": "IEM",
	"NWS NDFD": "NDFD", "Open-Meteo": "O-METEO", "Open-Meteo Marine": "O-METEO",
	"NIFC WFIGS": "NIFC", "NOAA HMS": "HMS", "NOAA NDBC": "NDBC", "NOAA CO-OPS": "CO-OPS",
	"EPA AirNow": "AIRNOW", "EPA Envirofacts": "EPA",
	"NOAA Space Weather Prediction Center": "SWPC", "GIRO": "GIRO",
}
