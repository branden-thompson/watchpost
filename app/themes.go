package app

// themes.go — theme files: registering user theme JSON and the live theme switch. Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
)

// applyThemes registers user theme files (<config dir>/themes/*.json,
// {"tokens": {"temp.hi": "208", ...}}; unlisted tokens inherit the default)
// and activates the persisted choice (UAT 53).
func applyThemes(chosen string) {
	if path, err := config.Path(); err == nil {
		loadUserThemes(filepath.Join(filepath.Dir(path), "themes"))
	}
	if chosen != "" && !render.SetTheme(chosen) {
		_ = invariant.Check(false, "configured theme not found: "+chosen+" (falling back to "+render.DefaultThemeName+")")
	}
}

// loadUserThemes registers every readable theme file in dir; a bad file is
// skipped with a dev-visible invariant, never a startup failure.
func loadUserThemes(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // no user themes
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			continue
		}
		var doc struct {
			Tokens map[string]string `json:"tokens"`
		}
		if jerr := json.Unmarshal(raw, &doc); jerr != nil || len(doc.Tokens) == 0 {
			_ = invariant.Check(false, "theme file unreadable or empty: "+e.Name())
			continue
		}
		over := make(map[render.Token]string, len(doc.Tokens))
		for k, v := range doc.Tokens {
			over[render.Token(k)] = v
		}
		render.RegisterTheme(strings.TrimSuffix(e.Name(), ".json"), over)
	}
}

// setThemeHook activates a theme live and persists the choice.
func setThemeHook(name string) error {
	if !render.SetTheme(name) {
		return fmt.Errorf("unknown theme %q", name)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Theme = name
	return config.Save(cfg)
}
