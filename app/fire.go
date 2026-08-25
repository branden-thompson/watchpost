package app

import (
	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fireProviders is the wildfire set (B5, D-10/OQ-19): HMS satellite
// detections and WFIGS named incidents for everyone, FIRMS when the user
// stored a MAP_KEY in setup. The rules come from `[fire]` in config with
// the AI-3 defaults filled in — one place, shared by the dashboard and the
// one-shot report (M5 parity).
func fireProviders(client *httpx.Client, cfg config.Config) ([]snapshot.Provider, *firms.Provider) {
	rules := fireRules(cfg.Fire)
	f := firms.New(client, "", cfg.Providers["firms"].Key, rules) // always registered: the Setup window can key it while the dashboard runs (UAT 100); unkeyed it reads "off"
	return []snapshot.Provider{hms.New(client, "", rules), wfigs.New(client, "", rules), f}, f
}

// fireSourceNames are the feeds as the broadcast says them (UAT 114).
func fireSourceNames(firmsKeyed bool) []string {
	names := []string{"NOAA's Hazard Mapping System", "the National Interagency Fire Center"}
	if firmsKeyed {
		names = append(names, "NASA FIRMS")
	}
	return names
}

// fireReportOf builds the broadcast's fire report (UAT 114) from a
// location's merged fire state: Known only once a fire feed has answered;
// the rings and the contributing feeds ride along for the script. FIRMS is
// credited only when it answered ok (REVIEW C4).
func fireReportOf(fs snapshot.FireState, lat, lon float64, rules fire.Rules, firmsOK bool) synth.FireReport {
	if fs.AsOf.IsZero() {
		return synth.FireReport{}
	}
	return synth.FireReport{Known: true, State: fs, RadiusKm: rules.RadiusKm, IncidentRadiusKm: rules.IncidentRadiusKm,
		Sources: fireSourceNames(firmsOK), Lat: lat, Lon: lon}
}

// fireRules maps the config section onto the domain's rules.
func fireRules(f config.Fire) fire.Rules {
	f = f.WithDefaults()
	return fire.Rules{RadiusKm: f.RadiusKm, IncidentRadiusKm: f.IncidentRadiusKm, MinFRPMW: f.MinFRPMW, BoldFRPMW: f.BoldFRPMW, MinConfidence: f.MinConfidence}
}
