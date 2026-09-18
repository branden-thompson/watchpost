package config

import (
	"os"
	"slices"
	"testing"
)

// FR-6.3 — THE MIGRATION IS PURELY ADDITIVE, proved against a file 0.15.0
// itself wrote: testdata/0.15.0-full.toml was captured through v0.15.0's own
// Save with every field populated. It must decode unchanged, gain only 0.16.0's
// defaults, and survive a 0.16.0 Save without losing a value (RS-6).
func TestA0150ConfigRoundTripsUnchangedAndGainsOnlyDefaults(t *testing.T) {
	p := withFixture(t, "0.15.0-full.toml")
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	check := func(cfg Config, when string) {
		t.Helper()
		if len(cfg.Locations) != 2 || cfg.Locations[0].Label != "Bonsall, CA" || cfg.Locations[0].Tag != "BNSL" ||
			cfg.Locations[0].Lat != 33.2889 || cfg.Locations[1].Zip != "92054" || cfg.Locations[1].TZ != "America/Los_Angeles" {
			t.Errorf("%s: locations = %+v", when, cfg.Locations)
		}
		if len(cfg.Recent) != 1 || cfg.Recent[0].Label != "Julian, CA" {
			t.Errorf("%s: recent = %+v", when, cfg.Recent)
		}
		if cfg.Providers["firms"].Key != "firms-key-for-the-fixture-not-a-credential" {
			t.Errorf("%s: providers = %+v", when, cfg.Providers)
		}
		if !slices.Equal(cfg.Keys["quit"], []string{"q", "ctrl+c"}) || !slices.Equal(cfg.Keys["lookup"], []string{"l"}) {
			t.Errorf("%s: keys = %+v", when, cfg.Keys)
		}
		r := cfg.Radio
		if r.Mode != "relay" || r.Cast != castModeOn || r.Voices.Alerts != (RoleVoice{MacOS: "Rishi", Piper: "en_US-ryan-medium"}) ||
			r.Voices.Weather != (RoleVoice{MacOS: "Samantha", Piper: "en_US-amy-medium"}) || r.Voices.Station.MacOS != "Daniel" ||
			r.Tones.Mode != toneModeMute || !slices.Equal(r.Tones.Muted, []string{"watch", "advisory"}) {
			t.Errorf("%s: radio = %+v", when, r)
		}
		if f := cfg.Fire; f.RadiusKm != 80 || f.IncidentRadiusKm != 120 || f.MinFRPMW != 5 || f.BoldFRPMW != 50 || f.MinConfidence != "nominal" {
			t.Errorf("%s: fire = %+v", when, f)
		}
		if s := cfg.Seismic; s.Enabled == nil || !*s.Enabled || s.LookbackDays != 14 || !slices.Equal(s.Types, []string{"earthquake", "quarry blast"}) {
			t.Errorf("%s: seismic = %+v", when, s)
		}
		if cfg.Theme != "catppuccin" || cfg.Voice != "Rishi" || cfg.Units != "metric" || cfg.Clock != "24h" ||
			!cfg.UpdateCheck || !cfg.TickerMuted || cfg.TickerRadiusMi != 75 {
			t.Errorf("%s: scalars = theme %q voice %q units %q clock %q update %v muted %v radius %d", when,
				cfg.Theme, cfg.Voice, cfg.Units, cfg.Clock, cfg.UpdateCheck, cfg.TickerMuted, cfg.TickerRadiusMi)
		}
		// ONLY DEFAULTS ARE GAINED: 0.16.0's station settings are unset, and the
		// station falls back to the listener's default location (D-72).
		if b := cfg.Broadcaster; b.Transmitter != (Location{}) || b.ServiceRadiusMi != 0 || b.BedRadiusMi != 0 {
			t.Errorf("%s: a 0.15.0 file gained a station it never had: %+v", when, b)
		}
		if st, ok := cfg.Station(); !ok || st.Label != "Bonsall, CA" {
			t.Errorf("%s: Station() = %+v, %v — want the default location", when, st, ok)
		}
	}
	cfg := mustLoad(t)
	check(cfg, "after Load")
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	check(mustLoad(t), "after a 0.16.0 Save and Load")
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == string(before) {
		t.Logf("a 0.16.0 Save reproduced the 0.15.0 bytes exactly")
	}
}
