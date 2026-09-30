package app

import (
	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/seismic/usgs"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
)

// credits is the About window's "Data Provided by" list (OQ-15, UAT 75) —
// every source the build reads, each line owned by its package. NOAA
// products are public domain; GeoNames and Open-Meteo are CC BY 4.0, so
// this list is a licence obligation, not a courtesy. Add a source here
// when you add a provider or geocoder; the About window renders it as is.
func credits() []string {
	return []string{
		nws.Attribution,
		ndbc.Attribution,
		coops.Attribution,
		locations.Attribution,
		openmeteo.Attribution,
		hms.Attribution,           // wildfire detections (B5)
		wfigs.Attribution,         // wildfire incidents (B5)
		firms.Attribution,         // keyed detections (B5)
		usgs.Attribution,          // earthquakes (0.11.0)
		stream.TableAttribution,   // NWR transmitter list (B4)
		stream.WxradioAttribution, // community audio relays (B4)
	}
}

// aboutNotes are the About window's closing lines, after every credit - the
// station's and the map's (D-148): the relays' condition of use (UAT 103),
// then R-13's safety framing, always last.
func aboutNotes() []string {
	return []string{stream.Disclaimer, SafetyNote, SafetyNext}
}

// mapCredits is the About window's "Maps" list (0.18.0 D-148): every source
// the map draws from, its licence where it has one - the one place the
// credits are said in full; the map says them short, in its badge row's
// chips and the basemap's own credit line on the frame (D-131). Add a source
// here when a layer reads a new one.
func mapCredits() []string {
	return []string{
		basemapAttribution,
		"Radar: NOAA NCEP MRMS and the Iowa Environmental Mesonet (IEM)",
		"Radar ahead: NOAA HRRR, via the Iowa Environmental Mesonet",
		"Forecasts: NWS NDFD (graphical.weather.gov)",
		temperature.OpenMeteoCredit,
		temperature.OpenMeteoRainCredit,
		temperature.OpenMeteoWavesCredit,
		temperature.OpenMeteoAirCredit,
		airquality.Attribution,
		uv.Attribution, // UV's cold start (D-167)
		"Fire: NIFC WFIGS perimeters and incidents; NOAA HMS satellite hotspots",
		usgs.Attribution,
		ndbc.Attribution,
		coops.Attribution,
	}
}

// basemapAttribution is the basemap's credit in full: OpenFreeMap's tiles,
// OpenMapTiles' schema, OpenStreetMap's data under the ODbL - which also asks
// for the credit on the map itself, where the library draws it.
const basemapAttribution = "Basemap: OpenFreeMap, © OpenMapTiles, © OpenStreetMap contributors (ODbL)"

// SafetyNote / SafetyNext are the R-13 safety framing (discover G-3b), two
// About lines: Watchpost shows what the sources publish, with the lag that
// implies — it is not a warning system. Named in the README too.
const (
	SafetyNote = "Not a substitute for official warnings."
	SafetyNext = "For life safety: NOAA Weather Radio and WEA."
)
