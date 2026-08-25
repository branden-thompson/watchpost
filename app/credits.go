package app

import (
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
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
		stream.TableAttribution,   // NWR transmitter list (B4)
		stream.WxradioAttribution, // community audio relays (B4)
		"",                        // breathing room before the relays' condition of use (UAT 103)
		stream.Disclaimer,         // the relays' condition of use
		SafetyNote, SafetyNext,    // R-13: the app's own framing, always last
	}
}

// SafetyNote / SafetyNext are the R-13 safety framing (discover G-3b), two
// About lines: Watchpost shows what the sources publish, with the lag that
// implies — it is not a warning system. Named in the README too.
const (
	SafetyNote = "Not a substitute for official warnings."
	SafetyNext = "For life safety: NOAA Weather Radio and WEA."
)
