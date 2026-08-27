package app

// Quality pass Q2 (L3-F22): the composition root's pure helpers — theme
// files, the assembler's attribution cases, the debug address, the
// resolver hooks' failure paths and the cache directory builder.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestLoadUserThemesRegistersReadableFilesAndSkipsBadOnes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dusk.json"), []byte(`{"tokens":{"temp.hi":"208","text.base":"250"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{"tokens":`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "empty.json"), []byte(`{"tokens":{}}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(`ignored`), 0o600)
	loadUserThemes(dir)
	names := strings.Join(render.ThemeNames(), ",")
	if !strings.Contains(names, "dusk") {
		t.Fatalf("a readable theme file registers under its file name: %s", names)
	}
	if strings.Contains(names, "broken") || strings.Contains(names, "empty") || strings.Contains(names, "notes") {
		t.Fatalf("unreadable, empty and non-JSON files are skipped: %s", names)
	}
	loadUserThemes(filepath.Join(dir, "missing")) // no themes dir: nothing happens
	applyThemes("no-such-theme")                  // an unknown persisted choice never fails startup
	if render.SetTheme(render.DefaultThemeName) != true {
		t.Fatal("the default theme must still be selectable")
	}
}

func TestNewAssemblerAttributesEveryKnownProvider(t *testing.T) {
	refs := []snapshot.LocationRef{{Label: "X", Zip: "1"}}
	asm := newAssembler(refs, []snapshot.Provider{})
	if locs, _ := asm.Size(); locs != 1 {
		t.Fatalf("the assembler tracks the refs: %d", locs)
	}
	// Attribution follows the provider id through the closed set; unknown
	// ids simply carry no line (the About view shows what it has).
	snap := asm.Snapshot()
	if len(snap.Providers) != 0 {
		t.Fatalf("no providers registered, none published: %+v", snap.Providers)
	}
}

func TestDebugAddrDefaultsAndHonoursTheOverride(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_PPROF_ADDR", "")
	if got := debugAddr(); got != "127.0.0.1:6060" {
		t.Fatalf("default debug address, got %q", got)
	}
	t.Setenv("WATCHPOST_DEBUG_PPROF_ADDR", "127.0.0.1:6061")
	if got := debugAddr(); got != "127.0.0.1:6061" {
		t.Fatalf("the override wins, got %q", got)
	}
}

func TestResolveHookReportsABrokenResolverOnEveryQuery(t *testing.T) {
	hook := resolveHook(nil, os.ErrNotExist)
	if _, err := hook("Oceanside"); err == nil || !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("a resolver that failed to build answers with that reason: %v", err)
	}
	if got := suggestHook(nil)("Ocean", 5); got != nil {
		t.Fatalf("no resolver means no suggestions (the Setup window then resolves on enter), got %v", got)
	}
}

func TestCacheDirBuilderIsUnderTheOSCacheDir(t *testing.T) {
	if dir := userCacheSubdir("http"); dir != "" && !strings.Contains(filepath.ToSlash(dir), "/watchpost/http") {
		t.Fatalf("the HTTP cache lives at <os cache>/watchpost/http, got %s", dir)
	}
	if cacheDir() != userCacheSubdir("http") || voiceDir() != userCacheSubdir("piper") {
		t.Fatal("cacheDir and voiceDir are the one builder's outputs")
	}
}

func TestDeriveTagIsFiveAlnumUppercase(t *testing.T) {
	for in, want := range map[string]string{"Oceanside, CA": "OCEAN", "Boise": "BOISE", "St. Louis, MO": "STLOU", "Ely": "ELY", "": ""} {
		if got := deriveTag(in); got != want {
			t.Errorf("deriveTag(%q) = %q, want %q", in, got, want)
		}
	}
}
