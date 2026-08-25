package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Spec: architecture.md §10.11 — TOML at $XDG_CONFIG_HOME/watchpost/config.toml,
// 0600 file / 0700 dir, atomic write, tables [locations] [providers.<id>] [keys]
// [radio] [playlist]. C-3′: no keychain prompts, no secrets beyond this file.

func testDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", d)
	return d
}

func TestPathHonorsXDG(t *testing.T) {
	d := testDir(t)
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(d, "watchpost", "config.toml")
	if p != want {
		t.Fatalf("Path() = %q, want %q", p, want)
	}
}

func TestLoadMissingFileIsFirstRun(t *testing.T) {
	testDir(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing config must not error (first-run), got %v", err)
	}
	if !cfg.FirstRun {
		t.Fatal("missing file must set FirstRun")
	}
	if len(cfg.Locations) != 0 {
		t.Fatal("first-run config must have no locations")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	testDir(t)
	cfg := Default()
	cfg.Locations = []Location{{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29, TZ: "America/Los_Angeles"}}
	cfg.Recent = []Location{{Label: "El Cajon, CA", Zip: "92020", Lat: 32.79, Lon: -116.96, TZ: "America/Los_Angeles"}, {Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2, TZ: "America/Boise"}} // UAT 96
	cfg.Providers = map[string]Provider{"firms": {Key: "sentinel-key-A1"}}
	cfg.Radio.Mode = "relay" // UAT 97
	cfg.Keys = map[string][]string{"help": {"?"}}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstRun {
		t.Fatal("saved config must not be FirstRun")
	}
	if len(got.Locations) != 1 || got.Locations[0].Zip != "92057" {
		t.Fatalf("locations roundtrip failed: %+v", got.Locations)
	}
	if len(got.Recent) != 2 || got.Recent[0].Zip != "92020" || got.Recent[1].Zip != "83702" {
		t.Fatalf("recent stack roundtrip keeps order (UAT 96): %+v", got.Recent)
	}
	if got.Providers["firms"].Key != "sentinel-key-A1" {
		t.Fatal("provider key roundtrip failed")
	}
	if got.Radio.Mode != "relay" {
		t.Fatal("radio mode roundtrip failed")
	}
	if got.Keys["help"][0] != "?" {
		t.Fatal("keymap roundtrip failed")
	}
}

func TestFireRulesDefaultAndOverride(t *testing.T) {
	// B5: unset rules take the AI-3 defaults; a user overrides one at a time.
	d := Fire{}.WithDefaults()
	if d.RadiusKm != 25 || d.IncidentRadiusKm != 50 || d.MinFRPMW != 5 || d.BoldFRPMW != 50 || d.MinConfidence != "nominal" {
		t.Fatalf("defaults: %+v", d)
	}
	o := Fire{RadiusKm: 40, MinConfidence: "high"}.WithDefaults()
	if o.RadiusKm != 40 || o.MinConfidence != "high" || o.MinFRPMW != 5 {
		t.Fatalf("override keeps the rest: %+v", o)
	}
	testDir(t)
	cfg := Default()
	cfg.Locations = []Location{{Label: "x", Zip: "00000"}}
	cfg.Fire = Fire{RadiusKm: 40}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := Load()
	if got.Fire.RadiusKm != 40 {
		t.Fatalf("[fire] roundtrip: %+v", got.Fire)
	}
}

func TestSavePermissions0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permissions")
	}
	testDir(t)
	if err := Save(Default()); err != nil {
		t.Fatal(err)
	}
	p, _ := Path()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("config file mode = %o, want 0600 (C-3')", fi.Mode().Perm())
	}
	di, err := os.Stat(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Fatalf("config dir mode = %o, want 0700", di.Mode().Perm())
	}
}

func TestSaveIsAtomic(t *testing.T) {
	testDir(t)
	if err := Save(Default()); err != nil {
		t.Fatal(err)
	}
	p, _ := Path()
	// No temp remnants next to the config after a successful save.
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "config.toml" {
			t.Fatalf("unexpected file after atomic save: %s", e.Name())
		}
	}
}

func TestSaveRefusesFirstRun(t *testing.T) {
	testDir(t)
	cfg := Default()
	cfg.FirstRun = true
	if err := Save(cfg); err == nil {
		t.Fatal("Save must refuse FirstRun configs — persisting one breaks first-run detection (F5)")
	}
}

func TestLoadCorruptFileFailsActionably(t *testing.T) {
	testDir(t)
	p, _ := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("this is not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("corrupt config must error, not silently reset (framework error rule)")
	}
}

func TestLoadValidatesFireRules(t *testing.T) {
	// Red-team B5 F3/F4: [fire] faults are reported at load, naming the key;
	// the confidence label is case-folded; the rings are bounded.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(body string) error {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load()
		return err
	}
	if err := write("[fire]\nmin_confidence = \"nomnial\"\n"); err == nil || !strings.Contains(err.Error(), "min_confidence") {
		t.Fatalf("a typo is named at load: %v", err)
	}
	if err := write("[fire]\nradius_km = 5000\n"); err == nil || !strings.Contains(err.Error(), "radius_km") {
		t.Fatalf("an absurd ring is refused: %v", err)
	}
	if err := write("[fire]\nmin_confidence = \" High \"\nradius_km = 40\n"); err != nil {
		t.Fatalf("case and space fold: %v", err)
	}
	cfg, _ := Load()
	if f := cfg.Fire.WithDefaults(); f.MinConfidence != "high" || f.RadiusKm != 40 {
		t.Fatalf("folded: %+v", f)
	}
}
