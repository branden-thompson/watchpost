package app

import "github.com/branden-thompson/watchpost/modes/tty"

// creditGroups are the About window's data sets (W21, the HUM LEAD's mock in
// about-credits-mock.md): a group a provider, each source the app reads named
// once - the station's sources and the map's alike - with what its terms ask.
// NOAA's products are public domain; Open-Meteo and GeoNames are CC BY 4.0,
// which asks for the credit and for a change to be said, so every grid drawn
// from Open-Meteo says it is interpolated; OpenStreetMap's data is the ODbL,
// whose credit is named; AirNow asks its readings be called preliminary, and
// NASA that FIRMS data be credited to LANCE FIRMS. The
// voices are credited under the licence they are published under (D-228).
// Every line is a chip, a phrase, an endpoint and a note where there is one
// (D-235): the credit-row component draws them all.
// Add a source here when a provider or a layer reads a new one.
func creditGroups() []tty.CreditGroup {
	return []tty.CreditGroup{
		{Name: "NATIONAL OCEANIC AND ATMOSPHERIC ADMINISTRATION (NOAA)", Lines: []tty.CreditLine{
			{Badge: "NWS", What: "National Weather Service", Host: "api.weather.gov"},
			{Badge: "NDBC", What: "National Data Buoy Center", Host: "ndbc.noaa.gov"},
			{Badge: "CO-OPS", What: "Tides & Currents", Host: "tidesandcurrents.noaa.gov"},
			{Badge: "HMS", What: "Wildfire Satellite Hotspots", Host: "ospo.noaa.gov"},
			{Badge: "NHC", What: "Tropical Storms", Host: "nhc.noaa.gov"},
			{Badge: "NWR", What: "Transmitter List", Host: "weather.gov/nwr"},
			{Badge: "MRMS", What: "Current Radar for Maps"},
			{Badge: "HRRR", What: "Radar Ahead (forecast) for Maps"},
			{Badge: "NDFD", What: "Gridded Forecast Data", Host: "graphical.weather.gov"},
		}},
		{Name: "NATIONAL AERONAUTICS AND SPACE ADMINISTRATION (NASA)", Lines: []tty.CreditLine{
			{Badge: "FIRMS", What: "Fire Hotspots, API Key Required", Host: "earthdata.nasa.gov", Note: "LANCE FIRMS, operated by NASA ESDIS"},
		}},
		{Name: "NATIONAL INTERAGENCY FIRE CENTER", Lines: []tty.CreditLine{
			{Badge: "NIFC", What: "WFIGS Wildfire Incidents", Host: "nifc.gov"},
		}},
		{Name: "NATIONAL WEATHER RADIO", Lines: []tty.CreditLine{
			{Badge: "RELAYS", What: "Community Audio Relays", Host: "wxradio.org", Note: "wxradio.org & weatherUSA (community)"},
		}},
		{Name: "UNITED STATES ENVIRONMENTAL PROTECTION AGENCY", Lines: []tty.CreditLine{
			{Badge: "AIRNOW", What: "AirNow Air Quality (AQI)", Host: "airnow.gov", Note: "Preliminary data, not fully verified"},
			{Badge: "EPA", What: "UV Index (Envirofacts)", Host: "data.epa.gov"},
		}},
		{Name: "UNITED STATES GEOLOGICAL SURVEY", Lines: []tty.CreditLine{
			{Badge: "USGS", What: "Earthquake Hazards Program", Host: "earthquake.usgs.gov"},
		}},
		{Name: "IOWA ENVIRONMENTAL MESONET", Lines: []tty.CreditLine{
			{Badge: "IEM", What: "Radar & Radar Ahead", Host: "mesonet.agron.iastate.edu"},
		}},
		{Name: "OPEN-METEO (CC BY 4.0)", Lines: []tty.CreditLine{
			{Badge: "O-METEO", What: "Geocoding", Host: "geocoding-api.open-meteo.com"},
			{Badge: "O-METEO", What: "Wind Data", Host: "api.open-meteo.com", Note: "Interpolated"},
			{Badge: "O-METEO", What: "Supplemental Temperature Data", Host: "api.open-meteo.com", Note: "Interpolated"},
			{Badge: "O-METEO", What: "Supplemental UV Index Data", Host: "api.open-meteo.com", Note: "Interpolated"},
			{Badge: "O-METEO", What: "Off-shore Wave Data", Host: "marine-api.open-meteo.com", Note: "Interpolated"},
			{Badge: "O-METEO", What: "Rain and Snow Total Forecasts", Host: "api.open-meteo.com", Note: "Interpolated"},
		}},
		{Name: "OPENFREEMAP", Lines: []tty.CreditLine{
			{Badge: "OFM", What: "Basemap Tiles", Host: "openfreemap.org", Note: "© OpenMapTiles, © OpenStreetMap contributors (ODbL)"},
		}},
		{Name: "GEONAMES (CC BY 4.0)", Lines: []tty.CreditLine{
			{Badge: "GEONAMES", What: "Cities & Postal Codes", Host: "geonames.org", Note: "The offline place index"},
		}},
		{Name: "VOICES", Lines: []tty.CreditLine{
			{Badge: "PIPER", What: "Piper Voices (MIT)", Host: "huggingface.co", Note: "rhasspy/piper-voices"},
		}},
	}
}

// aboutWarnings open the About window (D-232): what Watchpost is not, where to
// turn for life safety - R-13's framing - and the relays' gaps and lag.
func aboutWarnings() []string {
	return []string{
		"NOT INTENDED TO SUBSTITUTE OFFICIAL WARNING SOURCES, DEVICES, OR FOR LIFE SAFETY USE.",
		"FOR LIFE SAFETY, USE NOAA WEATHER RADIO AND COMPATIBLE DEVICES.",
		"WEATHER RELAYS MAY BE INCOMPLETE OR DELAYED.",
	}
}
