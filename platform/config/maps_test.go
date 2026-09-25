package config

import "testing"

// TestTheMapSettingsMigrateAdditively is 0.18.0 W1.10 (FR-9.1): a file from
// before maps gains nothing but defaults - empty words, which read as maps on
// and the description with the picture - and the two words round-trip.
func TestTheMapSettingsMigrateAdditively(t *testing.T) {
	withFixture(t, "0.15.0-full.toml")
	cfg := mustLoad(t)
	if cfg.Maps != "" || cfg.MapDescription != "" {
		t.Errorf("a 0.15.0 file gained map settings it never had: %q %q", cfg.Maps, cfg.MapDescription)
	}
	if err := Mutate(func(c *Config) error { c.Maps, c.MapDescription = "off", "instead"; return nil }); err != nil {
		t.Fatal(err)
	}
	again := mustLoad(t)
	if again.Maps != "off" || again.MapDescription != "instead" {
		t.Errorf("the map settings did not round-trip: %q %q", again.Maps, again.MapDescription)
	}
	if again.Units != "metric" || again.Theme != "catppuccin" {
		t.Errorf("writing the map settings lost the file's others: units %q theme %q", again.Units, again.Theme)
	}
}
