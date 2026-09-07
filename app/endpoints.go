package app

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
