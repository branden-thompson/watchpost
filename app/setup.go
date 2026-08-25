// Setup (UAT 100, HUM LEAD 2026-08-25): the first-run questions live in a
// window over the dashboard like every other modal (modes/tty setupLines);
// this file is the app side — the type-ahead hook and the persist step.
// Re-runs never destroy customizations (B2 red-team #2, probe-confirmed
// data loss): the existing config is loaded and only what setup owns is
// touched — the default location and, when given, the FIRMS key.
package app

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// suggestHook is the Setup window's type-ahead: the embedded index only
// (never the network per keystroke — AI-8 ToS), tags derived like Resolve.
func suggestHook(r *locations.Resolver) func(query string, limit int) []snapshot.LocationRef {
	return func(query string, limit int) []snapshot.LocationRef {
		if r == nil {
			return nil
		}
		hints := r.TypeAhead(query, limit)
		out := make([]snapshot.LocationRef, 0, len(hints))
		for _, h := range hints {
			ref := h.Ref
			if ref.Tag == "" {
				ref.Tag = deriveTag(ref.Label)
			}
			out = append(out, ref)
		}
		return out
	}
}

// applySetup persists the Setup window's answers: the chosen location
// becomes the default (first priority entry; any other entry for the same
// place drops out — F4: a duplicate would leave the dashboard empty), the
// FIRMS key is stored when given, and first-run is over.
func applySetup(def snapshot.LocationRef, firmsKey string) error {
	if err := invariant.Check(def.Label != "", "setup must carry a location"); err != nil {
		return err
	}
	firmsKey = strings.TrimSpace(firmsKey)
	if err := firms.CheckKey(firmsKey); err != nil {
		return err // refused before anything is written — the window shows the reason
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot read existing config before saving setup: %w", err)
	}
	cfg.FirstRun = false
	loc := config.Location{Label: def.Label, Tag: def.Tag, Zip: def.Zip, Lat: def.Lat, Lon: def.Lon, TZ: def.TZ}
	if loc.Tag == "" {
		loc.Tag = deriveTag(def.Label)
	}
	locs := []config.Location{loc}
	for i, l := range cfg.Locations {
		if i == 0 || (l.Zip == loc.Zip && l.Zip != "") || (l.Lat == loc.Lat && l.Lon == loc.Lon) {
			continue
		}
		locs = append(locs, l)
	}
	cfg.Locations = locs
	if firmsKey != "" {
		if cfg.Providers == nil {
			cfg.Providers = map[string]config.Provider{}
		}
		cfg.Providers["firms"] = config.Provider{Key: firmsKey}
	}
	return config.Save(cfg)
}

// deriveTag builds the default 5-char short label from a location name
// (mock LABEL column; users customize it in the add-location modal, M-V3).
func deriveTag(label string) string {
	var out []rune
	for _, r := range strings.ToUpper(label) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
		if len(out) == 5 {
			break
		}
	}
	return string(out)
}
