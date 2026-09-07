package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// S-4 — A RELATIVE XDG_CONFIG_HOME NEVER RESOLVES AGAINST THE WORKING DIRECTORY.
//
// domains/radio/synth already refuses a relative install dir by name (S-F5:
// "$HOME unset must not exec ./piper/piper"). This path is the one that decides
// what the radio SAYS — the watchlist, the FIRMS key and the script overrides —
// and it had no such guard, so launching from an untrusted directory read that
// directory's config.
func TestARelativeConfigHomeIsRefused(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/abs-base")
	abs, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if abs != filepath.Join("/tmp/abs-base", "watchpost", "config.toml") {
		t.Fatalf("an absolute base must be honoured, got %q", abs)
	}
	for _, rel := range []string{"watchpost-conf", "./conf", "../conf"} {
		t.Setenv("XDG_CONFIG_HOME", rel)
		got, err := Path()
		if err != nil {
			continue // no home directory to fall back to is a fine refusal
		}
		if !filepath.IsAbs(got) {
			t.Errorf("XDG_CONFIG_HOME=%q resolved to the relative path %q", rel, got)
		}
		if strings.HasPrefix(got, rel) {
			t.Errorf("XDG_CONFIG_HOME=%q was resolved against the working directory: %q", rel, got)
		}
	}
}
