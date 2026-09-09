package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
)

// WATCHPOST_DEBUG_RADIO is the one radio diagnostic: engine statuses and
// the synth's segments share the file, timestamped, so a cycle that ended
// before its tail says which segment it reached (HUM LEAD UAT 2026-08-28).
func TestRadioDebugLogCarriesStatusesAndSegments(t *testing.T) {
	path := radioDebugTo(t, "1")
	d := &radioDeck{}
	d.logStatus(player.Status{State: player.Playing, Title: "x"})
	d.debugLog(`segment key="tail:Samantha" spoken=` + (1500 * time.Millisecond).String())
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `playing      mount="" err="" title="x"`) || !strings.Contains(lines[1], `segment key="tail:Samantha" spoken=1.5s`) {
		t.Fatalf("log: %q", lines)
	}
	for _, l := range lines {
		if _, err := time.Parse(time.RFC3339Nano, strings.Fields(l)[0]); err != nil {
			t.Fatalf("every line is timestamped: %q", l)
		}
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Fatalf("the log is private: %v", fi.Mode())
	}
	if err := os.Unsetenv("WATCHPOST_DEBUG_RADIO"); err != nil {
		t.Fatal(err)
	}
	d.debugLog("nothing")
	if b2, _ := os.ReadFile(path); string(b2) != string(b) {
		t.Fatal("unset: no writes")
	}
}

// radioDebugTo turns the diagnostic on and points the CACHE ROOT at a temp
// directory, so a test never writes into the developer's own cache.
//
// It sets all three of the variables os.UserCacheDir consults, because which
// one applies is the platform's business and a test should not have to know:
// XDG_CACHE_HOME on unix, HOME on macOS, LocalAppData on Windows.
func radioDebugTo(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("LocalAppData", dir)
	t.Setenv("WATCHPOST_DEBUG_RADIO", name)
	path := radioDebugPath()
	if path == "" {
		t.Skip("no cache directory on this platform: the diagnostic is off and there is nothing to read")
	}
	return path
}

// THE ENVIRONMENT PICKS A NAME, NOT A PATH (FR-9.2).
//
// The variable took a path and the writer appended to it, so an unvalidated
// append-anywhere file write was one environment variable away on a process
// that runs all day — and the plan's first revision had it ON BY DEFAULT. What
// it may choose now is WHICH log under the cache root, which is the only part
// of the decision a caller has any business making.
func TestTheRadioDiagnosticTakesANameAndNotAPath(t *testing.T) {
	for _, tc := range []struct{ env, want string }{
		{"1", "radio"},                 // on, default name
		{"soak", "soak"},               // a name
		{"soak-2_a", "soak-2_a"},       // the punctuation a name may carry
		{"../../etc/passwd", "radio"},  // a traversal is not a name
		{"/tmp/anywhere.log", "radio"}, // nor is a path
		{"Soak", "radio"},              // nor an upper case, which a name is not
		{"a.b", "radio"},               // nor a dot: the extension is ours
		{strings.Repeat("x", 33), "radio"},
	} {
		if got := radioDebugName(tc.env); got != tc.want {
			t.Errorf("%q chose the log name %q, want %q", tc.env, got, tc.want)
		}
	}
	// AND THE PATH IS UNDER THE CACHE ROOT, whatever was asked for.
	path := radioDebugTo(t, "../../escape")
	if filepath.Base(path) != "radio.log" || !strings.Contains(path, filepath.Join("watchpost", "debug")) {
		t.Errorf("the log landed at %q", path)
	}
}

// AND IT IS OFF UNLESS IT IS ASKED FOR.
func TestTheRadioDiagnosticIsOffByDefault(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_RADIO", "")
	if radioDebugPath() != "" || radioDebugOn() {
		t.Error("the diagnostic is on with nothing asking for it")
	}
}

// THE LOG IS ROTATED, ONCE. A 24/7 process writing three to ten lines per event
// needs a ceiling; the previous log is kept as .1 and the one before it goes,
// because a diagnostic is read while the thing it describes is still fresh.
func TestTheRadioDiagnosticRotatesPastItsCeiling(t *testing.T) {
	path := radioDebugTo(t, "rot")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, radioDebugMax), 0o600); err != nil {
		t.Fatal(err)
	}
	radioDebugLog("after the ceiling")

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= radioDebugMax {
		t.Errorf("the log was not rotated: %d bytes", len(b))
	}
	if !strings.Contains(string(b), "after the ceiling") {
		t.Errorf("the line that triggered the rotation was lost: %q", b)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("the previous log was not kept: %v", err)
	}
	// AND 0600: it carries station names, mount URLs and the listener's own
	// locations.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the log is %v, want 0600", fi.Mode().Perm())
	}
}
