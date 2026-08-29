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
	path := filepath.Join(t.TempDir(), "radio.log")
	t.Setenv("WATCHPOST_DEBUG_RADIO", path)
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
