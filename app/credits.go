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
// Add a source here when a provider or a layer reads a new one.
func creditGroups() []tty.CreditGroup {
	return []tty.CreditGroup{
		{Name: "NATIONAL OCEANIC AND ATMOSPHERIC ADMINISTRATION (NOAA)", Lines: []tty.CreditLine{
			{Abbr: "NWS", What: "National Weather Service", Host: "api.weather.gov"},
			{Abbr: "NDBC", What: "National Data Buoy Center", Host: "ndbc.noaa.gov"},
			{Abbr: "CO-OPS", What: "Tides & Currents", Host: "tidesandcurrents.noaa.gov"},
			{Abbr: "HMS", What: "Wildfire Satellite Hotspots", Host: "ospo.noaa.gov"},
			{Abbr: "NHC", What: "Tropical Storms", Host: "nhc.noaa.gov"},
			{Abbr: "NWR", What: "Transmitter List", Host: "weather.gov/nwr"},
			{Abbr: "MRMS", What: "Current Radar for Maps"},
			{Abbr: "HRRR", What: "Radar Ahead (forecast) for Maps"},
			{Abbr: "NDFD", What: "National Digital Forecast Database", Host: "graphical.weather.gov"},
		}},
		{Name: "NATIONAL AERONAUTICS AND SPACE ADMINISTRATION (NASA)", Lines: []tty.CreditLine{
			{Abbr: "FIRMS", What: "Fire Hotspots, API Key Required", Host: "earthdata.nasa.gov", Note: "LANCE FIRMS, operated by NASA ESDIS"},
		}},
		{Name: "NATIONAL INTERAGENCY FIRE CENTER", Lines: []tty.CreditLine{
			{Abbr: "WFIGS", What: "Wildfire Incidents", Host: "nifc.gov"},
		}},
		{Name: "NATIONAL WEATHER RADIO", Lines: []tty.CreditLine{
			{What: "wxradio.org & weatherUSA (community)"},
		}},
		{Name: "UNITED STATES ENVIRONMENTAL PROTECTION AGENCY", Lines: []tty.CreditLine{
			{Abbr: "AQI", What: "U.S. EPA AirNow", Note: "preliminary data, not fully verified"},
			{Abbr: "UVI", What: "U.S. EPA (Envirofacts)"},
		}},
		{Name: "UNITED STATES GEOLOGICAL SURVEY", Lines: []tty.CreditLine{
			{What: "Earthquake Hazards Program", Host: "earthquake.usgs.gov"},
		}},
		{Name: "IOWA ENVIRONMENTAL MESONET", Lines: []tty.CreditLine{
			{Abbr: "IEM", What: "Radar & Radar Ahead for Maps"},
		}},
		{Name: "OPEN-METEO (CC BY 4.0)", Lines: []tty.CreditLine{
			{What: "Geocoding"},
			{What: "Wind Data, Interpolated"},
			{What: "Supplemental Temperature Data, Interpolated"},
			{What: "Supplemental UV Index Data, Interpolated"},
			{What: "Off-shore Wave Data, Interpolated"},
			{What: "Rain and Snow Total Forecasts, Interpolated"},
		}},
		{Name: "OPENFREEMAP", Lines: []tty.CreditLine{
			{What: "Basemap Tiles", Host: "openfreemap.org", Note: "© OpenMapTiles, © OpenStreetMap contributors (ODbL)"},
		}},
		{Name: "GEONAMES (CC BY 4.0)", Lines: []tty.CreditLine{
			{What: "Cities & Postal Codes, the offline index", Host: "geonames.org"},
		}},
		{Name: "VOICES", Lines: []tty.CreditLine{
			{What: "Piper voices, rhasspy/piper-voices (MIT)", Host: "huggingface.co"},
		}},
	}
}

// aboutWarnings open the About window (W21, D-229): what Watchpost is not, the
// relays' lag, and - R-13's safety framing - where to turn for life safety.
func aboutWarnings() []string {
	return []string{
		"NOT INTENDED AS A SUBSTITUTE FOR OFFICIAL WARNING SOURCES OR DEVICES",
		"WEATHER RELAYS MAY BE DELAYED",
		"NOT INTENDED FOR LIFE SAFETY USE",
		"FOR LIFE SAFETY: NOAA WEATHER RADIO AND WIRELESS EMERGENCY ALERTS",
	}
}
