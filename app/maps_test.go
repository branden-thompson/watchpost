package app

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

// TestTheAppHandsTheWindowItsMap is 0.18.0 W1.1 and W2's wiring (rule 8: a
// test of a thing is not a test of its wiring): the composition root hands the
// map window a constructor, and what it builds draws the basemap from the
// embedded tiles, sized as asked, reaching nothing (FR-3.2).
func TestTheAppHandsTheWindowItsMap(t *testing.T) {
	lp := &livePipelines{maps: newMapBuilder("t", t.TempDir(), &recorded{offline: true}, nil)}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	if cfg.NewMap == nil {
		t.Fatal("the window is handed no map constructor, so g would open a window with no map")
	}
	m, err := cfg.NewMap(tuimaps.Size{Cols: 69, Rows: 12})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	f, err := m.Render(tuimaps.Size{Cols: 69, Rows: 12}, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Lines) != 12 {
		t.Errorf("the map drew %d lines, want 12", len(f.Lines))
	}
	if (&livePipelines{}).newMap() != nil {
		t.Error("a station with no builder handed the window a constructor")
	}
}

// TestTheAsBuiltMapIsKeptInStep holds the HUM LEAD's rule (2026-09-25): the
// diagrams move with the code. The as-built page names the batches it draws,
// and the build log names the batches that landed; a batch that lands without
// its diagrams fails here.
func TestTheAsBuiltMapIsKeptInStep(t *testing.T) {
	dir := filepath.Join("..", "06_docs", "02_features", "observer-maps")
	log, err := os.ReadFile(filepath.Join(dir, "04-development", "build-log.md"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(dir, "03-architecture-design", "as-built-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	latest := 0
	for _, m := range regexp.MustCompile(`(?m)^## Batch (\d+)`).FindAllStringSubmatch(string(log), -1) {
		latest = max(latest, atoiOrZero(m[1]))
	}
	drawn := regexp.MustCompile(`Batches 1–(\d+)`).FindStringSubmatch(string(page))
	if latest == 0 || drawn == nil {
		t.Fatalf("the build log names batch %d and the as-built page %v: this measures nothing", latest, drawn)
	}
	if atoiOrZero(drawn[1]) != latest {
		t.Errorf("the build log has reached batch %d and the as-built map draws batches 1–%s: redraw it with the batch", latest, drawn[1])
	}
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// TestTheMapSettingsReachTheWindowAndTheFile is W1.10's wiring (FR-9.1): the
// file's words reach the window, and the window's save writes them back.
func TestTheMapSettingsReachTheWindowAndTheFile(t *testing.T) {
	lp := &livePipelines{}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{Maps: "off", MapDescription: "instead"}, nil, nil, nil, nil, nil, nil)
	if cfg.Maps != "off" || cfg.MapDescription != "instead" {
		t.Errorf("the window is handed %q %q", cfg.Maps, cfg.MapDescription)
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "watchpost"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "watchpost", "config.toml"), []byte("units = \"imperial\"\n"), 0o600); err != nil {
		t.Fatal(err) // Settings saves over a file the station already has; a first run's is refused by design
	}
	if err := setUIHook(tty.UIPrefs{Units: "metric", Clock: "24h", Maps: "off", MapDescription: "off"}); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Maps != "off" || got.MapDescription != "off" {
		t.Errorf("the save wrote %q %q", got.Maps, got.MapDescription)
	}
}
