package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// CLEARING EMPTIES THE MAP'S WEB CACHE (D-219, FR-3.10): the map's clients keep
// what they fetched on disk now, so "Clear map data" removes it there and in
// memory - a later session finds nothing - and asks the network nothing.
func TestClearingEmptiesTheMapsWebCache(t *testing.T) {
	asked := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked++
		_, _ = w.Write([]byte("hour"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	u := srv.URL + "/forecast"
	if _, err := c.GetText(context.Background(), u, httpx.TTL(time.Hour)); err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{mapClients: []*httpx.Client{c}}
	got := lp.clearMapData()
	if got.Err != nil || got.Files == 0 {
		t.Errorf("cleared %+v; want the web cache's entry counted", got)
	}
	if _, ok := c.Cached(u); ok {
		t.Error("the entry is still in memory")
	}
	later, err := httpx.New(httpx.Config{UserAgent: UserAgent, CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := later.Cached(u); ok {
		t.Error("a later session finds the entry on disk")
	}
	if asked != 1 {
		t.Errorf("clearing asked the network (%d asks)", asked)
	}
}

// THE TEMPERATURE CLIENT KEEPS ITS ANSWERS ON DISK (D-219): the production
// client is built over the map's web-cache directory, which building it makes.
func TestTheTemperatureClientKeepsItsAnswersOnDisk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	if !strings.HasPrefix(mapHTTPDir(), home) {
		t.Fatalf("the directory %q is not under the test's home", mapHTTPDir())
	}
	if _, err := newTempClient(UserAgent); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(mapHTTPDir()); err != nil || !info.IsDir() {
		t.Errorf("the temperature client keeps nothing at %q: %v", mapHTTPDir(), err)
	}
}
