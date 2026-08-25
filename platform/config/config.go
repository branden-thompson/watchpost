// Package config owns the watchpost user configuration file.
//
// Contract (architecture.md §10.11, C-3′): TOML at
// $XDG_CONFIG_HOME/watchpost/config.toml (fallback ~/.config), file mode 0600,
// dir 0700, atomic writes (temp + rename). Holds locations, per-provider keys,
// key bindings, radio and playlist settings. Never prompts a keychain; never
// stores secrets anywhere else. Corrupt files fail loudly — they are never
// silently reset (framework error rule).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Location is one configured place to watch (labels always carry a zip — R-2′).
type Location struct {
	Label string  `toml:"label"`
	Tag   string  `toml:"tag,omitempty"` // 5-char short label (M-V3 sets; setup derives)
	Zip   string  `toml:"zip"`
	Lat   float64 `toml:"lat"`
	Lon   float64 `toml:"lon"`
	TZ    string  `toml:"tz"`
}

// Provider holds per-provider settings; Key is the user-supplied API key (T-G).
type Provider struct {
	Key string `toml:"key,omitempty"`
}

// Radio holds tuner settings. Only what 0.9 reads lives here (red-team
// round 2 R2-21): the `--tts-cmd` argv template and a stream override are
// 1.0 items and arrive with their code, not before it.
type Radio struct {
	Mode string `toml:"mode,omitempty"` // "synth" (default) | "relay" — the [m] source pick (UAT 97)
}

// Fire holds the wildfire-proximity rules (B5; HUM LEAD 2026-08-25: "make it
// configurable"). Zero values mean "the default" — see WithDefaults — so a
// user sets only what they want to change:
//
//	[fire]
//	radius_km = 25          # hotspots this close count
//	incident_radius_km = 50 # named incidents this close are listed
//	min_frp_mw = 5          # weaker detections are ignored
//	bold_frp_mw = 50        # this strong reads emphasized
//	min_confidence = "nominal"  # low | nominal | high
type Fire struct {
	RadiusKm         float64 `toml:"radius_km,omitempty"`
	IncidentRadiusKm float64 `toml:"incident_radius_km,omitempty"`
	MinFRPMW         float64 `toml:"min_frp_mw,omitempty"`
	BoldFRPMW        float64 `toml:"bold_frp_mw,omitempty"`
	MinConfidence    string  `toml:"min_confidence,omitempty"`
}

// WithDefaults fills every unset rule with the AI-3 default.
func (f Fire) WithDefaults() Fire {
	if f.RadiusKm <= 0 {
		f.RadiusKm = 25
	}
	if f.IncidentRadiusKm <= 0 {
		f.IncidentRadiusKm = 50
	}
	if f.MinFRPMW <= 0 {
		f.MinFRPMW = 5
	}
	if f.BoldFRPMW <= 0 {
		f.BoldFRPMW = 50
	}
	f.MinConfidence = strings.ToLower(strings.TrimSpace(f.MinConfidence))
	if f.MinConfidence == "" {
		f.MinConfidence = "nominal"
	}
	return f
}

// fireRadiusMax bounds the rings: past this a single location would attach
// the whole continent's detections (red-team B5 F4).
const fireRadiusMax = 500

// Validate checks the [fire] table the way Load reports every other config
// fault — at load, naming the key, before any provider runs (red-team B5
// F3): the confidence label is a closed set (case-folded here), the rings
// bounded, the thresholds non-negative.
func (f Fire) Validate() error {
	if f.RadiusKm < 0 || f.RadiusKm > fireRadiusMax || f.IncidentRadiusKm < 0 || f.IncidentRadiusKm > fireRadiusMax {
		return fmt.Errorf("[fire] radius_km and incident_radius_km must be 0 (default) to %d", fireRadiusMax)
	}
	if f.MinFRPMW < 0 || f.BoldFRPMW < 0 {
		return errors.New("[fire] min_frp_mw and bold_frp_mw must not be negative")
	}
	switch strings.ToLower(strings.TrimSpace(f.MinConfidence)) {
	case "", "low", "nominal", "high":
		return nil
	}
	return fmt.Errorf("[fire] min_confidence must be low, nominal or high (got %q)", f.MinConfidence)
}

// Config is the whole user configuration. FirstRun is derived, never
// persisted. Unknown keys in the file are ignored, so a config written by a
// build that had more fields still loads.
type Config struct {
	Locations []Location          `toml:"locations,omitempty"`
	Recent    []Location          `toml:"recent,omitempty"` // RECENT / SEARCHED stack, newest first (UAT 96) — restored above the seeds
	Providers map[string]Provider `toml:"providers,omitempty"`
	Keys      map[string][]string `toml:"keys,omitempty"` // Action -> key names (D-15)
	Radio     Radio               `toml:"radio,omitempty"`
	Fire      Fire                `toml:"fire,omitempty"`  // wildfire rules (B5)
	Theme     string              `toml:"theme,omitempty"` // active color theme (UAT 53)
	Voice     string              `toml:"voice,omitempty"` // radio correspondent voice (UAT 84)

	FirstRun bool `toml:"-"`
}

// Default returns an empty, valid configuration.
func Default() Config {
	return Config{Providers: map[string]Provider{}, Keys: map[string][]string{}}
}

// Path returns the config file path, honoring $XDG_CONFIG_HOME with the
// conventional ~/.config fallback.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home directory: %w (set XDG_CONFIG_HOME to override)", err)
		}
		base = filepath.Join(home, ".config")
	}
	if err := invariant.Check(base != "", "config base directory must resolve"); err != nil {
		return "", err
	}
	return filepath.Join(base, "watchpost", "config.toml"), nil
}

// Load reads the configuration. A missing file is a valid first run; a corrupt
// file is an error the caller must surface (never silently reset).
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		cfg.FirstRun = true
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("cannot read %s: %w", p, err)
	}
	cfg := Default()
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("config file %s is corrupt: %w — fix it or move it aside and rerun 'watchpost setup'", p, err)
	}
	if err := cfg.Fire.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", p, err)
	}
	return cfg, nil
}

// Save writes the configuration atomically with 0600/0700 permissions (C-3′).
// A FirstRun config is refused: FirstRun means "no file exists"; persisting one
// would make first-run undetectable forever (B0 red-team F5). Callers that want
// an empty config on disk clear FirstRun first; "has locations" checks belong
// on len(cfg.Locations), never FirstRun.
func Save(cfg Config) error {
	if err := invariant.Check(!cfg.FirstRun, "refusing to persist a first-run config — clear FirstRun before Save"); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create config dir %s: %w", dir, err)
	}
	raw, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("cannot stage config write in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: after a successful rename the file is gone and
	// Remove fails with ErrNotExist by design — nothing to handle.
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("cannot set config permissions: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("cannot write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("cannot flush config write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot finish config write: %w", err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		return fmt.Errorf("cannot activate new config: %w", err)
	}
	return nil
}
