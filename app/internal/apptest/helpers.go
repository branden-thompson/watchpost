package apptest

// helpers.go — the pure helpers the app tree's tests share (0.19.6 design
// AP-9, A-24): the config file, the poll, the places, the offline client and
// the NWS fixtures. None reads a package's own state.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/app/internal/run"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// WithConfigFile points the config at a file the station already has:
// Settings saves over one, and a first run's is refused by design.
func WithConfigFile(t *testing.T) {
	t.Helper()
	WriteConfigFile(t, "units = \"imperial\"\n")
}

// WriteConfigFile is WithConfigFile with the file's words given.
func WriteConfigFile(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "watchpost"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "watchpost", "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// PollLimit is how long a test polls for a state before failing. A poll
// returns the moment its condition holds, so the limit costs nothing on a
// passing run; it has to outlast a loaded CI runner under the race detector,
// which two seconds does not.
const PollLimit = 10 * time.Second

// WaitUntil polls cond (no fixed sleeps: the release, a queue position) and
// fails after pollLimit.
func WaitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(PollLimit); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// Oceanside is a resolved place the tests save and read back.
func Oceanside() snapshot.LocationRef {
	return snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29, TZ: "America/Los_Angeles"}
}

// BonsallCfg is a configured location with its zone.
var BonsallCfg = config.Location{Label: "Bonsall, CA", Zip: "92003", Lat: 33.2881, Lon: -117.2256, TZ: "America/Los_Angeles"}

// IndexForTest is the embedded place index, loaded once per call.
func IndexForTest(t *testing.T) *geodata.Index {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// OfflineClient is an httpx client over a server of the test's own, and the
// server's URL for the providers to point at. The handler is what the test
// wants upstream to say — nothing at all for a deck that must run offline, the
// NWS fixtures for a seam that needs a location to exist — and no unit test
// reaches api.weather.gov. The cache is memory-only on purpose: a disk tier
// starts a writer goroutine that outlives the test and races the removal of
// t.TempDir(), which is 0.16.0 F-102's mechanism.
func OfflineClient(t *testing.T, upstream http.Handler) (*httpx.Client, string) {
	t.Helper()
	srv := httptest.NewServer(upstream)
	t.Cleanup(srv.Close)
	client, err := httpx.New(httpx.Config{UserAgent: run.UserAgent, RatePerSec: 1000, MaxRetries: 1})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client, srv.URL
}

// NWSFixtures serves the NWS package's recorded responses for the point,
// station, observation and alert routes, and nothing for any other.
func NWSFixtures(t *testing.T) http.Handler {
	t.Helper()
	dir := ModuleDir(t, "domains", "weather", "nws", "testdata")
	serve := func(w http.ResponseWriter, name string) {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("COULD NOT RUN — the NWS fixture %s: %v", name, err)
			http.Error(w, "no fixture", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/points/"):
			serve(w, "points.json")
		case strings.HasSuffix(p, "/stations"):
			serve(w, "stations.json")
		case strings.HasSuffix(p, "/observations/latest"):
			serve(w, "obs.json")
		case p == "/alerts/active":
			serve(w, "alerts.json")
		default:
			http.NotFound(w, r)
		}
	})
}

// ModuleDir is a directory of the module by its path from the module's root,
// found from the test's working directory up to go.mod, so a helper reads the
// same fixture from whichever package's tests call it.
func ModuleDir(t *testing.T, elem ...string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ { // bounded by the tree's depth (P10-02)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(append([]string{dir}, elem...)...)
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("COULD NOT RUN: no go.mod above the test's working directory")
	return ""
}
