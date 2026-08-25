package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: UAT 100 — setup is a window over the dashboard (modes/tty owns the
// questions); the app side is the type-ahead hook and the persist step.
// Re-runs never destroy customizations (B2 red-team #2); the chosen
// location becomes priority[0]; the FIRMS key is stored only when given
// and only in its MAP_KEY shape.

// realResolver drives the hook against the actual embedded index — the
// probe-confirmed "Portland, ME -> Portland, OR" class must be pinned on
// the real qualifier-filtering path, not a fake.
func realResolver(t *testing.T) *locations.Resolver {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	r, err := locations.New(idx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSuggestHookQualifiedCityChoosesTheQualifiedState(t *testing.T) {
	suggest := suggestHook(realResolver(t))
	hints := suggest("Portland, ME", 5)
	if len(hints) == 0 || hints[0].Label != "Portland, ME" {
		t.Fatalf("Portland, ME must lead the hints, got %+v", hints)
	}
	if hints[0].Tag == "" || hints[0].Zip == "" {
		t.Fatalf("hints carry a tag and the zip (R-2'): %+v", hints[0])
	}
	if suggestHook(nil)("Portland", 5) != nil {
		t.Fatal("no resolver: no hints, no panic")
	}
}

func oceanside() snapshot.LocationRef {
	return snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29, TZ: "America/Los_Angeles"}
}

func TestApplySetupRerunPreservesCustomizations(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// An existing config with a theme, keys and a second favourite — a
	// re-run must keep all of it and only replace the default location.
	existing := config.Config{
		Theme: "Monochrome", Keys: map[string][]string{"quit": {"x"}},
		Locations: []config.Location{
			{Label: "El Cajon, CA", Tag: "ELCAJ", Zip: "92020", Lat: 32.79, Lon: -116.96},
			{Label: "Portland, ME", Tag: "PORTL", Zip: "04101", Lat: 43.66, Lon: -70.25},
			{Label: "Oceanside, CA", Tag: "OSIDE", Zip: "92057", Lat: 33.24, Lon: -117.29}, // duplicate of the new default: must drop
		},
		Providers: map[string]config.Provider{"pirate-weather": {Key: "keep-me"}},
	}
	if err := config.Save(existing); err != nil {
		t.Fatal(err)
	}
	if err := applySetup(oceanside(), ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FirstRun {
		t.Fatal("setup ends the first run")
	}
	if cfg.Theme != "Monochrome" || len(cfg.Keys["quit"]) != 1 || cfg.Keys["quit"][0] != "x" || cfg.Providers["pirate-weather"].Key != "keep-me" {
		t.Fatalf("customizations lost: %+v", cfg)
	}
	labels := make([]string, 0, len(cfg.Locations))
	for _, l := range cfg.Locations {
		labels = append(labels, l.Label)
	}
	if got := strings.Join(labels, "|"); got != "Oceanside, CA|Portland, ME" {
		t.Fatalf("default first, old default replaced, duplicate dropped: %s", got)
	}
	if cfg.Locations[0].Tag != "OCEAN" {
		t.Fatalf("tag derived for the default: %q", cfg.Locations[0].Tag)
	}
	if _, ok := cfg.Providers["firms"]; ok {
		t.Fatal("no key given: no firms entry")
	}
}

func TestApplySetupStoresAWellFormedFIRMSKeyOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	key := "0123456789abcdef0123456789ABCDEF"
	if err := applySetup(oceanside(), " "+key+" "); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Providers["firms"].Key != key {
		t.Fatalf("key stored trimmed: %q", cfg.Providers["firms"].Key)
	}
	err = applySetup(oceanside(), "not-a-map-key")
	if err == nil || !strings.Contains(err.Error(), "32 hex") || strings.Contains(err.Error(), "not-a-map-key") {
		t.Fatalf("a malformed key is refused with the shape, never echoed: %v", err)
	}
	if err := applySetup(snapshot.LocationRef{}, ""); err == nil {
		t.Fatal("setup without a location must fail the invariant")
	}
}
