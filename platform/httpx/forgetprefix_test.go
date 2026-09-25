package httpx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForgetPrefixRemovesOnlyThoseEntries is 0.18.0 W3.8 (FR-3.9, FR-3.10):
// "Clear map data" forgets the zone outlines the HTTP cache holds - in memory
// and on disk - and nothing else the station has cached.
func TestForgetPrefixRemovesOnlyThoseEntries(t *testing.T) {
	ctx := context.Background()
	srv, hits := cacheServer(t, "public, max-age=426444", 200, 0)
	dir := t.TempDir()
	c := newCached(t, dir)
	var out struct{ N int }
	for _, p := range []string{"/zones/forecast/TXZ277", "/zones/county/TXC043", "/gridpoints/SGX/1,2/forecast"} {
		if _, err := c.GetJSON(ctx, srv.URL+p, &out); err != nil {
			t.Fatal(err)
		}
	}
	c.cache.flush()
	onDisk := func() int {
		n, _ := filepath.Glob(filepath.Join(dir, "*.cache"))
		return len(n)
	}
	if onDisk() != 3 {
		t.Fatalf("%d entries on disk, want 3, so this proves nothing", onDisk())
	}
	removed, err := c.ForgetPrefix(srv.URL + "/zones/")
	if err != nil || removed != 2 {
		t.Errorf("forgot %d (%v), want the 2 zone entries", removed, err)
	}
	if onDisk() != 1 {
		t.Errorf("%d entries left on disk, want the forecast alone", onDisk())
	}
	before := hits.Load()
	if _, err := c.GetJSON(ctx, srv.URL+"/gridpoints/SGX/1,2/forecast", &out); err != nil || hits.Load() != before {
		t.Error("the forecast was forgotten too")
	}
	if _, err := c.GetJSON(ctx, srv.URL+"/zones/forecast/TXZ277", &out); err != nil || hits.Load() != before+1 {
		t.Error("a forgotten zone was still served from the cache")
	}
	for _, f := range mustGlob(t, dir) {
		b, _ := os.ReadFile(f)
		if strings.Contains(string(b), "/zones/county/") {
			t.Errorf("a forgotten entry is still on disk: %s", f)
		}
	}
}

func mustGlob(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "*.cache"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}
