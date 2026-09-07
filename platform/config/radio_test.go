package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// withFixture points config.Path() at a copy of testdata/<name> and returns the
// path it was written to, so a test can Load, Save and read the bytes back.
func withFixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	p := filepath.Join(dir, "watchpost", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustLoad(t *testing.T) Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// --- Task 1.10: every fixture is pinned by a case ---

// FR-2: a 0.13.0 file names no role, so everything inherits the root voice, and
// its ticker_muted becomes mute mode with an EMPTY set — which means every
// class, so nothing the listener had silenced starts sounding.
func TestFixture0130OnlyInheritsEverythingAndMapsTheMute(t *testing.T) {
	withFixture(t, "0.13.0-only.toml")
	cfg := mustLoad(t)

	if cfg.Voice != "Samantha" {
		t.Errorf("Voice = %q, want the 0.13.0 root", cfg.Voice)
	}
	if cfg.Radio.Cast != castModeSingle {
		t.Errorf("Cast = %q, want single voice", cfg.Radio.Cast)
	}
	if cfg.Radio.Voices != (Voices{}) {
		t.Errorf("Voices = %+v, want the zero value — a 0.13.0 file assigns no roles", cfg.Radio.Voices)
	}
	if cfg.Radio.Tones.Mode != toneModeMute {
		t.Errorf("Tones.Mode = %q, want mute — ticker_muted = true carries over", cfg.Radio.Tones.Mode)
	}
	if len(cfg.Radio.Tones.Muted) != 0 {
		t.Errorf("Tones.Muted = %v, want empty — an empty set under mute means every class", cfg.Radio.Tones.Muted)
	}
	if cfg.TickerRadiusMi != 150 || cfg.Theme != "quattro-dark" {
		t.Error("the rest of the 0.13.0 file must load unchanged")
	}
}

// The mirror runs the other way too: a file with tones ON must not acquire a
// mute from a stale ticker_muted, and Save is the one owner of the mirror.
func TestSaveIsTheOneOwnerOfTheTickerMutedMirror(t *testing.T) {
	p := withFixture(t, "0.13.0-only.toml")
	cfg := mustLoad(t)

	// The listener turns the tones back on. ticker_muted must follow.
	cfg.Radio.Tones.Mode = toneModeOn
	cfg.TickerMuted = true // a caller setting it directly must not survive
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "ticker_muted") {
		t.Errorf("ticker_muted survived a tones-on save — Save derives it from the mode:\n%s", raw)
	}

	// And back the other way.
	cfg.Radio.Tones.Mode = toneModeMute
	cfg.TickerMuted = false
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(p)
	if !strings.Contains(string(raw), "ticker_muted = true") {
		t.Errorf("mute mode must write ticker_muted = true so a 0.13.0 binary still mutes:\n%s", raw)
	}
}

func TestFixtureBothOSRoundTripsWithoutLosingEitherHalf(t *testing.T) {
	p := withFixture(t, "0.14.0-both-os.toml")
	cfg := mustLoad(t)

	if cfg.Radio.Cast != castModeOn {
		t.Fatalf("Cast = %q, want cast mode", cfg.Radio.Cast)
	}
	if got := cfg.Radio.Voices.Alerts; got.MacOS != "Rishi" || got.Piper != "en_US-ryan-medium" {
		t.Errorf("alerts = %+v, want both halves", got)
	}
	if got := cfg.Radio.Tones; got.Mode != toneModeMute || !slices.Equal(got.Muted, []string{"watch", "advisory"}) {
		t.Errorf("tones = %+v", got)
	}

	// This machine edits only its own half; the other must survive the save.
	cfg.Radio.Voices.Alerts.MacOS = "Daniel"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	again := mustLoad(t)
	if got := again.Radio.Voices.Alerts; got.MacOS != "Daniel" || got.Piper != "en_US-ryan-medium" {
		t.Errorf("after a save alerts = %+v, want the edited macOS half and the untouched Piper half", got)
	}
	if _, err := os.ReadFile(p); err != nil {
		t.Fatal(err)
	}
}

// A single-OS file is not wrong on the other platform: the halves it does not
// carry are empty, which means inherit (FR-7). The falling back itself is
// domains/radio/cast's; here we pin that the file loads and keeps its shape.
func TestFixtureSingleOSFilesLoadWithTheOtherHalfEmpty(t *testing.T) {
	for _, tc := range []struct{ name, macos, piper string }{
		{"0.14.0-macos-only.toml", "Rishi", ""},
		{"0.14.0-piper-only.toml", "", "en_US-ryan-medium"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withFixture(t, tc.name)
			cfg := mustLoad(t)
			if got := cfg.Radio.Voices.Alerts; got.MacOS != tc.macos || got.Piper != tc.piper {
				t.Errorf("alerts = %+v, want macos %q piper %q", got, tc.macos, tc.piper)
			}
			if cfg.Radio.Voices.Weather != (RoleVoice{}) {
				t.Error("an unassigned role must stay the zero pair")
			}
		})
	}
}

// An empty pair is omitted, so an untouched file does not grow nine empty
// tables the listener never asked for.
func TestAnEmptyPairIsOmittedAndAnAssignedOneIsATable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Cast = castModeOn
	cfg.Radio.Voices.Fire = RoleVoice{MacOS: "Daniel"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "watchpost", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "[radio.voices.fire]") || !strings.Contains(got, `macos = 'Daniel'`) && !strings.Contains(got, `macos = "Daniel"`) {
		t.Errorf("the assigned pair must be written as a table:\n%s", got)
	}
	for _, absent := range []string{"[radio.voices.alerts]", "[radio.voices.weather]", "[radio.voices.station]"} {
		if strings.Contains(got, absent) {
			t.Errorf("%s was written for an empty pair:\n%s", absent, got)
		}
	}
}

// --- Task 1.8: Validate names the key it rejects ---

func TestLoadNamesABadRadioValue(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"wrong-type.toml", "corrupt"},
		{"bad-tone.toml", "[radio.tones] mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withFixture(t, tc.name)
			_, err := Load()
			if err == nil {
				t.Fatal("want an error naming the key")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err, tc.want)
			}
		})
	}
}

func TestRadioValidateChecksClosedSetsOnly(t *testing.T) {
	if err := (Radio{Cast: "casr"}).Validate(); err == nil || !strings.Contains(err.Error(), "cast") {
		t.Errorf("a bad cast mode must be named, got %v", err)
	}
	if err := (Radio{Tones: Tones{Mode: "muted"}}).Validate(); err == nil || !strings.Contains(err.Error(), "tones") {
		t.Errorf("a bad tone mode must be named, got %v", err)
	}
	// An unknown CLASS key is NOT this package's business — domains/radio/cast
	// owns what a class is, and duplicating the set is how the two drift.
	if err := (Radio{Tones: Tones{Mode: "mute", Muted: []string{"not-a-class"}}}).Validate(); err != nil {
		t.Errorf("class keys are cast.Validate's, not config's: %v", err)
	}
	// A voice name is likewise not checked here: whether it can speak depends
	// on the host, which this package knows nothing about.
	if err := (Radio{Cast: "cast", Voices: Voices{Alerts: RoleVoice{MacOS: "Nobody At All"}}}).Validate(); err != nil {
		t.Errorf("voice names are the host's business: %v", err)
	}
}

// --- Task 1.9: Save preserves keys this build does not know ---

func TestSavePreservesKeysFromAFutureBuild(t *testing.T) {
	p := withFixture(t, "unknown-key.toml")
	cfg := mustLoad(t)
	cfg.Voice = "Daniel" // an ordinary edit, as Setup would make
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{"future_top_level", "future_scalar", "future_half", "future_table", "future_root_table"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q was dropped on save — a 0.13.0 binary would delete a 0.14.0 cast this way:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "Daniel") {
		t.Error("the edit itself must survive too")
	}
	// The known keys are still known: the merge must not turn the document
	// into an untyped blob.
	again := mustLoad(t)
	if again.Voice != "Daniel" || again.Radio.Mode != "synth" || again.Radio.Voices.Alerts.MacOS != "Rishi" {
		t.Errorf("the typed keys did not survive the merge: %+v", again.Radio)
	}
}

// A key this build DOES know, cleared by the listener, must stay cleared — the
// merge preserves the unknown, never resurrects the known.
func TestSaveDoesNotResurrectAClearedKnownKey(t *testing.T) {
	p := withFixture(t, "0.14.0-both-os.toml")
	cfg := mustLoad(t)
	cfg.Radio.Voices.Alerts = RoleVoice{}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if strings.Contains(string(raw), "Rishi") {
		t.Errorf("a cleared assignment came back:\n%s", raw)
	}
}

// An ordinary save of a file with nothing unknown in it must be byte-for-byte
// what the typed marshal produces: the merge re-marshals only when it kept
// something, so preserving keys costs nothing on the common path.
func TestAnOrdinarySaveIsByteStable(t *testing.T) {
	p := withFixture(t, "0.14.0-both-os.toml")
	cfg := mustLoad(t)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(p)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Errorf("two identical saves differ:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// The fixture that forced the dependency bump: on go-toml v2.2.4 the strict
// decode panicked here, the recover masked it, and every unknown key was
// dropped in silence.
func TestQuotedEscapeKeyNeitherPanicsNorAliasesAKnownKey(t *testing.T) {
	p := withFixture(t, "quoted-escape-key.toml")
	cfg := mustLoad(t)
	if cfg.Radio.Mode != "synth" {
		t.Fatalf("Radio.Mode = %q — a quoted top-level key aliased the real nested one", cfg.Radio.Mode)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	got := string(raw)
	if !strings.Contains(got, "not the real one") || !strings.Contains(got, "also not the real one") {
		t.Errorf("the quoted keys were dropped — this is exactly the silent NFR-5 failure:\n%s", got)
	}
	again := mustLoad(t)
	if again.Radio.Mode != "synth" {
		t.Errorf("after the merge Radio.Mode = %q, want synth", again.Radio.Mode)
	}
}

// The recorded NFR-5 limitation: a path through an array of tables is not
// preserved, because indices are not stable across a save that reorders them
// and copying by index could attach a key to the wrong entry.
func TestUnknownKeysInsideAnArrayOfTablesAreNotPreserved(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "watchpost", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "voice = \"Samantha\"\n\n[[locations]]\nlabel = \"Oceanside, CA\"\nfuture_field = \"dropped by design\"\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := mustLoad(t)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if strings.Contains(string(raw), "future_field") {
		t.Error("this is the RECORDED LIMITATION — if it now survives, NFR-5's note must be updated, not this test deleted")
	}
	if !strings.Contains(string(raw), "Oceanside") {
		t.Error("the entry itself must survive")
	}
}

// A corrupt previous file must not be able to corrupt the save: the merge
// reports "kept nothing" and the typed document is written unchanged.
func TestAnUnparsableOldFileIsNotMerged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "watchpost", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("this is not = = toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	cfg.Voice = "Samantha"
	if err := Save(cfg); err != nil {
		t.Fatalf("a corrupt old file must not fail the save: %v", err)
	}
	if got := mustLoad(t); got.Voice != "Samantha" {
		t.Errorf("Voice = %q after saving over a corrupt file", got.Voice)
	}
}

// --- Task 1.11: [S] lists the unknown [radio.*] keys, rendered and capped ---

func TestUnknownListsRadioKeysOnlyAndIsNeverPersisted(t *testing.T) {
	p := withFixture(t, "unknown-key.toml")
	cfg := mustLoad(t)

	if !slices.Contains(cfg.Unknown, "radio.future_scalar") {
		t.Errorf("Unknown = %v, want the radio scalar", cfg.Unknown)
	}
	for _, absent := range []string{"future_top_level", "future_root_table"} {
		if slices.Contains(cfg.Unknown, absent) {
			t.Errorf("%q is not a [radio.*] key and must not be listed (it is still PRESERVED)", absent)
		}
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if strings.Contains(string(raw), "Unknown") {
		t.Errorf("Unknown is derived, never persisted:\n%s", raw)
	}
}

// A key from a hostile file reaches [S] only through plaintext.Line: this is
// the one place that rendering happens, so nothing downstream has to remember.
func TestUnknownRendersHostileKeysPlain(t *testing.T) {
	withFixture(t, "hostile-name.toml")
	cfg := mustLoad(t)
	for _, s := range cfg.Unknown {
		if strings.ContainsAny(s, "\x1b\x07\r\n\t") {
			t.Errorf("Unknown carries a control character: %q", s)
		}
	}
	// The hostile fixture's VALUES are not this package's to sanitise — they
	// are rendered at the seam that displays or speaks them (NFR-6) — but they
	// must survive load intact so that seam sees what is really there.
	if cfg.Radio.Voices.Alerts.MacOS == "" {
		t.Error("a hostile name must still load; sanitising happens where it is used")
	}
}

func TestUnknownIsCapped(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "watchpost", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("[radio]\nmode = \"synth\"\n")
	for i := range 200 {
		fmt.Fprintf(&b, "future_key_%d = 1\n", i)
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := mustLoad(t)
	if len(cfg.Unknown) > maxUnknownReported {
		t.Errorf("Unknown has %d entries, cap is %d — [S] shows a list a person reads", len(cfg.Unknown), maxUnknownReported)
	}
}
