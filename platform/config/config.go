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
	"sync"

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
//
// 0.14.0 adds the correspondent cast and the alert tones. Everything here is
// ADDITIVE and its zero value is 0.13.0's behaviour: an absent [radio.voices]
// table means every role inherits the `voice` key, and an absent [radio.tones]
// table means every class sounds (FR-2, data-shape §2).
type Radio struct {
	Mode string `toml:"mode,omitempty"` // "synth" (default) | "relay" — the [m] source pick (UAT 97)

	// Cast is "" (Single Voice: the root reads everything) or "cast" (the
	// per-role pairs are read). It is a MODE, not a switch that clears the
	// assignments — switching to Single Voice keeps the pairs unread so
	// switching back restores the whole cast (MVS-D-25).
	Cast string `toml:"cast,omitempty"`

	// Voices is one typed pair per assignable role. A struct of structs, not a
	// map: a typo'd role key is ignored rather than minted as a role (NFR-5),
	// and omitempty keeps an untouched file byte-identical.
	Voices Voices `toml:"voices,omitempty"`

	// Tones is the per-class mute state.
	Tones Tones `toml:"tones,omitempty"`
}

// RoleVoice is one role's assignment: the macOS `say -v` name and the Piper
// catalogue KEY (never a display name).
//
// Each OS reads and writes ONLY its own half, so a config file synced between a
// Mac and a Linux box keeps both assignments and neither platform destroys the
// other's work. A half that is empty on this platform simply inherits (FR-7).
type RoleVoice struct {
	MacOS string `toml:"macos,omitempty"`
	Piper string `toml:"piper,omitempty"`
}

// Voices is the nine assignable roles of the cast registry. The FIELD LIST IS
// THE REGISTRY: adding a role is one field here and one constant in
// domains/radio/cast, and an unknown table under [radio.voices] is ignored by
// the decoder rather than becoming a role nobody defined.
type Voices struct {
	Alerts     RoleVoice `toml:"alerts,omitempty"`
	Breaking   RoleVoice `toml:"breaking,omitempty"`
	SevereRead RoleVoice `toml:"severe_read,omitempty"`
	Standard   RoleVoice `toml:"standard,omitempty"`
	Weather    RoleVoice `toml:"weather,omitempty"`
	Maritime   RoleVoice `toml:"maritime,omitempty"`
	Fire       RoleVoice `toml:"fire,omitempty"`
	Seismic    RoleVoice `toml:"seismic,omitempty"`
	Station    RoleVoice `toml:"station,omitempty"`
}

// Tones is the alert-tone mute state (MVS-D-26). Every class always sounds its
// ratified preset; the only choice is whether it sounds at all.
type Tones struct {
	// Mode is "" (all tones on) or "mute".
	Mode string `toml:"mode,omitempty"`
	// Muted holds CLASS KEYS. An empty list under mute mode means every class:
	// the Setup window shows "Mute:" with nothing ticked, and what the screen
	// says is what the ear gets. Unknown keys are ignored and reported once,
	// with a count, by cast.Validate.
	Muted []string `toml:"muted,omitempty"`
}

// radioModes, castModes and toneModes are the closed sets Validate checks. They
// are the SHAPE half of the split with domains/radio/cast: which words are
// legal here, never what a role or a class means.
const (
	castModeSingle = ""
	castModeOn     = "cast"
	toneModeOn     = ""
	toneModeMute   = "mute"
)

// Validate checks the [radio] tables the way Load reports every other config
// fault — at load, naming the key.
//
// It checks TABLE SHAPES AND CLOSED WORD SETS ONLY. Whether an assigned voice
// can actually speak on this host, and which tone-class keys exist, are
// domains/radio/cast's (cast.Validate, P1 Task 1.5): this package must not
// import a domain, and duplicating the rule is how the loader, the screen and
// the deck drift apart.
//
// Note the asymmetry with Fire.Validate: a bad [fire] radius is an ERROR that
// stops the load, because acting on it would attach the wrong hotspots. A bad
// cast mode is reported and read as the default instead, because refusing to
// start over a typo in a voice preference would be worse than reading it
// conservatively — and cast.Validate surfaces it in [S] either way.
func (r Radio) Validate() error {
	switch r.Cast {
	case castModeSingle, castModeOn:
	default:
		return fmt.Errorf("[radio] cast must be %q (single voice) or %q (got %q)", castModeSingle, castModeOn, r.Cast)
	}
	switch r.Tones.Mode {
	case toneModeOn, toneModeMute:
	default:
		return fmt.Errorf("[radio.tones] mode must be %q (all tones on) or %q (got %q)", toneModeOn, toneModeMute, r.Tones.Mode)
	}
	return nil
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

// Seismic are the earthquake rules (0.11.0): the magnitude→radius step
// function (a quake shows iff its distance ≤ the miles of the first band
// whose upper magnitude exceeds the quake's), the lookback window, and the
// USGS event types shown. Zero values mean "the default" (WithDefaults).
type Seismic struct {
	Enabled       *bool       `toml:"enabled,omitempty"`
	LookbackDays  int         `toml:"lookback_days,omitempty"`
	Types         []string    `toml:"types,omitempty"`           // USGS event types; default ["earthquake"] (D4)
	RadiusBandsMi [][]float64 `toml:"radius_bands_mi,omitempty"` // ascending [upperMag, miles] bands
}

// WithDefaults fills every unset seismic rule with the ratified default.
func (s Seismic) WithDefaults() Seismic {
	if s.Enabled == nil {
		on := true
		s.Enabled = &on
	}
	if s.LookbackDays <= 0 {
		s.LookbackDays = 7
	}
	if len(s.Types) == 0 {
		s.Types = []string{"earthquake"}
	}
	if len(s.RadiusBandsMi) == 0 {
		s.RadiusBandsMi = [][]float64{{1.0, 3}, {2.5, 10}, {3.5, 20}, {4.0, 40}, {4.5, 100}, {5.0, 150}, {6.0, 400}, {7.0, 500}, {99, 1000}}
	}
	return s
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
	Fire      Fire                `toml:"fire,omitempty"`    // wildfire rules (B5)
	Seismic   Seismic             `toml:"seismic,omitempty"` // earthquake rules (0.11.0)
	Theme     string              `toml:"theme,omitempty"`   // active color theme (UAT 53)
	Voice     string              `toml:"voice,omitempty"`   // radio correspondent voice (UAT 84)

	// Display preferences — 0.14.0's WATCHPOST UI group. Both were live-only
	// before: [f]/[c] swapped the units for the session and nothing remembered
	// it, and the clock was whatever each site had hard-coded. Empty means the
	// default, and an unrecognised word reads as the default too (render's
	// UnitsByKey / ClockByKey) — a display preference is not worth refusing to
	// start over.
	Units string `toml:"units,omitempty"` // "imperial" (default) | "metric"
	Clock string `toml:"clock,omitempty"` // "12h" (default) | "24h" | "mil"

	// UpdateCheck asks the app, once an hour, whether a newer release is
	// published. OPT-IN: the app makes no unattended outbound request the
	// listener did not ask for, and the check is not needed to read weather.
	UpdateCheck bool `toml:"update_check,omitempty"`

	TickerMuted    bool `toml:"ticker_muted,omitempty"`     // [M]: the global event ticker's tone/narration muted (0.12.0)
	TickerRadiusMi int  `toml:"ticker_radius_mi,omitempty"` // 0.12.0: the alert-notification radius — 0 = All (global), >0 = only alerts within N miles of the default location

	FirstRun bool `toml:"-"`

	// Unknown lists the [radio.*] keys in the file that this build does not
	// know, as display strings (Task 1.11). Derived at load, shown in [S],
	// NEVER persisted — the keys themselves are preserved by Save's merge
	// (keep.go), which is a separate mechanism from reporting them.
	Unknown []string `toml:"-"`
}

// Default returns an empty, valid configuration.
func Default() Config {
	return Config{Providers: map[string]Provider{}, Keys: map[string][]string{}}
}

// Path returns the config file path, honoring $XDG_CONFIG_HOME with the
// conventional ~/.config fallback.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	// A RELATIVE XDG_CONFIG_HOME IS REFUSED — the same failure S-F5 already
	// fixed for the Piper binary, on the tree that decides WHAT THE RADIO SAYS
	// (red team 2026-09-05, S-4). With a relative value, launching watchpost
	// from an untrusted directory loaded ./watchpost/config.toml — locations,
	// the FIRMS key — and ./watchpost/scripts/. The spec says the variable is an
	// absolute base directory; anything else falls back to the conventional
	// path rather than resolving against the working directory.
	if base != "" && !filepath.IsAbs(base) {
		base = ""
	}
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
	if err := cfg.Radio.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w — fix it or move it aside and rerun 'watchpost setup'", p, err)
	}
	cfg = cfg.withToneCompat()
	cfg.Unknown = radioUnknowns(unknownKeys(raw))
	return cfg, nil
}

// withToneCompat maps a 0.13.0 file's ticker_muted onto the 0.14.0 tone state.
//
// 0.13.0's [M] muted the ticker's tone AND its narration; 0.14.0's mutes the
// TONES ONLY — the words always read (MVS-D-26). A listener upgrading with
// ticker_muted = true therefore starts in mute mode with an EMPTY set, which
// means every class (the empty-set rule), so every tone they had silenced stays
// silenced. They will hear the words again; that is the accepted, CHANGELOG'd
// behaviour change (MVS-D-34, E-6).
//
// The condition is "ticker_muted is set and the tones table says nothing",
// which is exactly a 0.13.0 file: from 0.14.0 on, Save mirrors ticker_muted
// from the mode, so the two can never disagree.
func (c Config) withToneCompat() Config {
	if c.TickerMuted && c.Radio.Tones.Mode == toneModeOn && len(c.Radio.Tones.Muted) == 0 {
		c.Radio.Tones.Mode = toneModeMute
	}
	return c
}

// mergeUnknown copies the keys this build does not know from the file at path
// into the freshly marshalled document, returning the re-marshalled bytes and
// whether anything was kept (NFR-5, keep.go).
//
// It re-marshals ONLY when something was kept, so an ordinary save of a file
// with no unknown keys is byte-for-byte what the typed marshal produced. Every
// failure along the way — an unreadable file, an unparsable one, an encode that
// will not round-trip — reports "kept nothing" and leaves the typed document
// alone: preserving keys must never be able to corrupt a save.
func mergeUnknown(marshalled []byte, path string) ([]byte, bool) {
	old, err := os.ReadFile(path)
	if err != nil {
		return nil, false // no previous file (a first save): nothing to preserve
	}
	paths := unknownKeys(old)
	if len(paths) == 0 {
		return nil, false
	}
	var oldDoc, newDoc map[string]any
	if err := toml.Unmarshal(old, &oldDoc); err != nil {
		return nil, false // an unparsable old file is not merged
	}
	if err := toml.Unmarshal(marshalled, &newDoc); err != nil {
		return nil, false
	}
	if keepUnknown(newDoc, oldDoc, paths) == 0 {
		return nil, false
	}
	merged, err := toml.Marshal(newDoc)
	if err != nil {
		return nil, false
	}
	return merged, true
}

// Save writes the configuration atomically with 0600/0700 permissions (C-3′).
// A FirstRun config is refused: FirstRun means "no file exists"; persisting one
// would make first-run undetectable forever (B0 red-team F5). Callers that want
// an empty config on disk clear FirstRun first; "has locations" checks belong
// on len(cfg.Locations), never FirstRun.
// Mutate is the ONE write path for the config file: it loads, applies edit, and
// saves, with the whole sequence held under one lock.
//
// WHAT THE LOCK BUYS, precisely. Save is already atomic on disk — CreateTemp
// plus Rename — so a reader can never see a torn file. What was unprotected was
// the READ-MODIFY-WRITE: six owners each did Load, edited their own field, and
// Saved, so two owners interleaving lost whichever edit landed first. This
// closes that window and nothing else.
//
// WHAT IT DOES NOT BUY: the lock is process-local. Two Watchpost instances on
// one machine still race, and last writer still wins — app/debug.go already
// anticipates two instances, so this is a stated limit rather than an oversight.
//
// edit RETURNS AN ERROR so a caller can abort inside the transaction: applySetup
// validates before writing and livePipelines.commit checks the watchlist cap,
// and neither can refuse through a signature that cannot say no.
//
// edit MUST NOT call Load, Save or Mutate. A Go mutex is not reentrant, so a
// nested call deadlocks the config path for the life of the process.
func Mutate(edit func(*Config) error) error {
	mu.Lock()
	defer mu.Unlock() // deferred: a panic inside edit must not strand every later write
	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := edit(&cfg); err != nil {
		return err
	}
	return Save(cfg)
}

// mu serialises Mutate's read-modify-write. It guards the SEQUENCE, not the
// file: Save's rename is what makes the file safe.
var mu sync.Mutex

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
	// Save is the ONE owner of the ticker_muted mirror: it is derived from the
	// tone mode on every write so a 0.13.0 binary reading this file still mutes
	// its ticker (data-shape §2). No caller sets it, and nothing else may.
	cfg.TickerMuted = cfg.Radio.Tones.Mode == toneModeMute
	raw, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	if merged, ok := mergeUnknown(raw, p); ok {
		raw = merged
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
