package app

import (
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/seismic"
	"github.com/branden-thompson/watchpost/domains/seismic/usgs"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// seismicReportOf builds the broadcast's seismic report (P4) from a location's
// merged seismic state: Known only once the USGS feed has answered (a nil or
// cold state stays unknown, so the report is skipped). The quakes ride along
// already filtered, sorted largest-first and capped by the provider.
func seismicReportOf(ss *snapshot.SeismicState, lat, lon float64) synth.SeismicReport {
	if ss == nil || ss.AsOf.IsZero() {
		return synth.SeismicReport{}
	}
	return synth.SeismicReport{Known: true, State: *ss, Lat: lat, Lon: lon}
}

// seismicProviders is the earthquake set (0.11.0): the USGS FDSN feed, keyless
// and public-domain. The rules come from `[seismic]` in config with the
// ratified defaults filled in — one place, shared by the dashboard and the
// one-shot report (the fireProviders sibling, M5 parity).
func seismicProviders(client *httpx.Client, cfg config.Config) []snapshot.Provider {
	return []snapshot.Provider{usgs.New(client, "", seismicRules(cfg.Seismic))}
}

// seismicRules maps the config section onto the domain's rules: the ascending
// [upperMag, miles] bands, the lookback window and the type allowlist, with
// the ratified defaults filled in for a missing or partial section.
func seismicRules(s config.Seismic) seismic.Rules {
	s = s.WithDefaults()
	bands := make([]seismic.Band, 0, len(s.RadiusBandsMi))
	for _, b := range s.RadiusBandsMi {
		if len(b) >= 2 {
			bands = append(bands, seismic.Band{UpperMag: b[0], RadiusMi: b[1]})
		}
	}
	return seismic.Rules{Bands: bands, LookbackDays: s.LookbackDays, Types: s.Types}
}
