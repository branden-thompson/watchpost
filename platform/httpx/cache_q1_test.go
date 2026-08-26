package httpx

// Quality pass Q1 (plan §2.2, Q1 task 3): the persistence floor, one
// retention rule, the allow-list sweep with its guard and cap, validator
// storage, the negative-cache cap and the stale-memory disk skip.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cacheDirIn returns a cache directory under a "watchpost" element, as the
// app builds it (<os cache>/watchpost/http): the sweep's guard requires it.
func cacheDirIn(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "watchpost", "http")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func fakeName(i int, ext string) string {
	sum := sha256.Sum256([]byte(fmt.Sprint(i)))
	return hex.EncodeToString(sum[:]) + ext
}

// writeCacheFile plants a cache file with the given expiry and mtime.
func writeCacheFile(t *testing.T, dir, name string, expires, mtime time.Time, body int) {
	t.Helper()
	hdr, _ := json.Marshal(entry{URL: "https://example.test/" + name, Expires: expires})
	if err := os.WriteFile(filepath.Join(dir, name), append(append(hdr, '\n'), strings.Repeat("b", body)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, name), mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func exists(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, name))
	return err == nil
}

func TestSweepIsAnAllowList(t *testing.T) {
	dir := cacheDirIn(t)
	now := time.Now()
	// Things the sweep must never touch.
	_ = os.WriteFile(filepath.Join(dir, "README.txt"), []byte("mine"), 0o600)
	_ = os.Mkdir(filepath.Join(dir, "subdir"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "subdir", "keep"), []byte("x"), 0o600)
	_ = os.Symlink(filepath.Join(dir, "README.txt"), filepath.Join(dir, fakeName(99, ".cache"))) // a symlink wearing a cache name
	// Things it owns.
	writeCacheFile(t, dir, fakeName(1, ".cache"), now.Add(time.Hour), now, 10)                        // fresh
	writeCacheFile(t, dir, fakeName(2, ".cache"), now.Add(-2*time.Hour), now.Add(-2*time.Hour), 10)   // expired, within grace
	writeCacheFile(t, dir, fakeName(3, ".cache"), now.Add(-48*time.Hour), now.Add(-48*time.Hour), 10) // expired beyond grace
	writeCacheFile(t, dir, fakeName(4, ".cache"), now.Add(-48*time.Hour), now.Add(-time.Minute), 10)  // header expired, but touched (a 304 renewal)
	_ = os.WriteFile(filepath.Join(dir, fakeName(5, ".json")), []byte("legacy"), 0o600)               // pre-UAT-73 orphan
	young := fakeName(6, ".cache") + ".123.tmp"
	old := fakeName(7, ".cache") + ".456.tmp"
	_ = os.WriteFile(filepath.Join(dir, young), []byte("t"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, old), []byte("t"), 0o600)
	_ = os.Chtimes(filepath.Join(dir, old), now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	_ = os.WriteFile(filepath.Join(dir, "not-ours.cache"), []byte("x"), 0o600) // wrong name shape
	_ = os.WriteFile(filepath.Join(dir, fakeName(8, ".cache")), []byte("no newline header"), 0o600)

	c := newCache(dir)
	c.sweep(now)

	for _, keep := range []string{"README.txt", "subdir", fakeName(99, ".cache"), fakeName(1, ".cache"), fakeName(2, ".cache"), fakeName(4, ".cache"), young, "not-ours.cache"} {
		if !exists(dir, keep) {
			t.Fatalf("%s must survive the sweep", keep)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "README.txt")); err != nil {
		t.Fatal("the symlink target must be intact")
	}
	for _, gone := range []string{fakeName(3, ".cache"), fakeName(5, ".json"), old, fakeName(8, ".cache")} {
		if exists(dir, gone) {
			t.Fatalf("%s must be swept", gone)
		}
	}
}

func TestSweepRefusesADirectoryWithoutWatchpostElement(t *testing.T) {
	dir := t.TempDir() // no "watchpost" path element: a misconfigured cache home
	now := time.Now()
	writeCacheFile(t, dir, fakeName(1, ".cache"), now.Add(-48*time.Hour), now.Add(-48*time.Hour), 10)
	c := newCache(dir)
	c.sweep(now)
	if !exists(dir, fakeName(1, ".cache")) {
		t.Fatal("the sweep must refuse to delete anything outside a watchpost directory")
	}
}

func TestSweepEnforcesTheDirectoryCapOldestFirst(t *testing.T) {
	dir := cacheDirIn(t)
	now := time.Now()
	for i := range 5 {
		writeCacheFile(t, dir, fakeName(i, ".cache"), now.Add(time.Hour), now.Add(-time.Duration(5-i)*time.Minute), 1000)
	}
	c := newCache(dir)
	c.maxDiskBytes = 2600 // room for two files' bodies + headers
	c.sweep(now)
	survivors := 0
	for i := range 5 {
		if exists(dir, fakeName(i, ".cache")) {
			survivors++
			if i < 3 {
				t.Fatalf("file %d is among the oldest and must go first", i)
			}
		}
	}
	if survivors != 2 {
		t.Fatalf("the cap keeps the newest files that fit, got %d survivors", survivors)
	}
}

func TestPersistenceFloorAndPersistOption(t *testing.T) {
	dir := cacheDirIn(t)
	srv, _ := cacheServer(t, "", 200, 0)
	c := newCached(t, dir)
	ctx := context.Background()
	cases := []struct {
		path string
		opts []Option
		want bool
	}{
		{"/short", []Option{TTL(time.Minute)}, false},
		{"/at-floor", []Option{TTL(5 * time.Minute)}, false},
		{"/above", []Option{TTL(10 * time.Minute)}, true},
		{"/directory", []Option{TTL(5 * time.Minute), Persist()}, true},
	}
	for _, tc := range cases {
		if _, err := c.GetJSON(ctx, srv.URL+tc.path, nil, tc.opts...); err != nil {
			t.Fatal(err)
		}
	}
	c.cache.flush()
	for _, tc := range cases {
		if got := exists(dir, filepath.Base(c.cache.path(srv.URL+tc.path))); got != tc.want {
			t.Fatalf("%s: on disk = %v, want %v (floor %v)", tc.path, got, tc.want, diskFloor)
		}
	}
	if st := c.CacheStats(); st.DiskWrites != 2 {
		t.Fatalf("the disk-write counter must count the two files, got %d", st.DiskWrites)
	}
}

func TestStaleMemoryEntrySkipsTheDiskRead(t *testing.T) {
	dir := cacheDirIn(t)
	srv, hits := cacheServer(t, "", 200, 0)
	c := newCached(t, dir)
	url := srv.URL + "/obs"
	if _, err := c.GetJSON(context.Background(), url, nil, TTL(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	c.cache.flush()
	// Age the memory entry past its expiry while the disk copy stays fresh
	// (a relaunch would read it — but this process knows it is stale).
	c.cache.mu.Lock()
	c.cache.mem[url].Expires = time.Now().Add(-time.Second)
	c.cache.mu.Unlock()
	writeCacheFile(t, dir, filepath.Base(c.cache.path(url)), time.Now().Add(time.Hour), time.Now(), 10)
	if _, err := c.GetJSON(context.Background(), url, nil, TTL(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("a stale memory entry must skip the disk read and refetch, got %d hits", hits.Load())
	}
}

func TestExpiredEntriesWithValidatorsAreRetained(t *testing.T) {
	c := newCache("")
	now := time.Now()
	c.now = func() time.Time { return now }
	c.put("https://x/plain", []byte("b"), now.Add(-time.Minute), validators{}, false)
	c.put("https://x/etag", []byte("b"), now.Add(-time.Minute), validators{etag: `"abc"`}, false)
	c.put("https://x/old", []byte("b"), now.Add(-48*time.Hour), validators{etag: `"old"`}, false)
	c.mu.Lock()
	c.evictLocked()
	_, plain := c.mem["https://x/plain"]
	_, etag := c.mem["https://x/etag"]
	_, old := c.mem["https://x/old"]
	c.mu.Unlock()
	if plain || !etag || old {
		t.Fatalf("expired-without-validators goes, expired-with-validators stays for the grace, beyond the grace goes: %v %v %v", plain, etag, old)
	}
	if _, ok := c.get("https://x/etag"); ok {
		t.Fatal("a retained stale entry is never served as fresh (Q5 revalidates it)")
	}
}

func TestValidatorsAreStoredBounded(t *testing.T) {
	dir := cacheDirIn(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		switch r.URL.Path {
		case "/good":
			w.Header().Set("ETag", `"v1"`)
			w.Header().Set("Last-Modified", "Wed, 26 Aug 2026 12:00:00 GMT")
		case "/huge":
			w.Header().Set("ETag", strings.Repeat("e", maxValidatorSize+1))
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := newCached(t, dir)
	for _, p := range []string{"/good", "/huge"} {
		if _, err := c.GetJSON(context.Background(), srv.URL+p, nil); err != nil {
			t.Fatal(err)
		}
	}
	c.cache.flush()
	c.cache.mu.Lock()
	good, huge := c.cache.mem[srv.URL+"/good"], c.cache.mem[srv.URL+"/huge"]
	c.cache.mu.Unlock()
	if good.ETag != `"v1"` || good.LastModified == "" {
		t.Fatalf("validators must be stored: %+v", good)
	}
	if huge.ETag != "" {
		t.Fatalf("an oversized validator must be dropped: %q", huge.ETag)
	}
	if safeValidator("\"a\x01b\"") != "" || safeValidator("\"ok\"") != `"ok"` { // net/http never delivers one, but the store guards anyway (IS-6)
		t.Fatal("a control-bearing validator must be dropped")
	}
	e, ok := readEntry(c.cache.path(srv.URL + "/good"))
	if !ok || e.ETag != `"v1"` {
		t.Fatalf("validators must survive the disk round trip: %+v %v", e, ok)
	}
}

func TestRenewedEntryOutlivesAnUntouchedOne(t *testing.T) {
	dir := cacheDirIn(t)
	c := newCache(dir)
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	writeCacheFile(t, dir, filepath.Base(c.path("https://x/renewed")), old, old, 10)
	writeCacheFile(t, dir, filepath.Base(c.path("https://x/untouched")), old, old, 10)
	c.renew("https://x/renewed", now.Add(time.Hour))
	c.flush()
	c.sweep(now)
	if !exists(dir, filepath.Base(c.path("https://x/renewed"))) || exists(dir, filepath.Base(c.path("https://x/untouched"))) {
		t.Fatal("a 304-renewed file (fresh mtime) must outlive an untouched expired one")
	}
}

func TestNegativeCacheIsCapped(t *testing.T) {
	c := newCache("")
	for i := range maxNegEntries + 30 {
		c.putNegative(fmt.Sprintf("https://x/%d", i), fmt.Errorf("404"))
	}
	if st := c.stats(); st.Negative != maxNegEntries {
		t.Fatalf("the negative cache must hold at most %d entries, has %d", maxNegEntries, st.Negative)
	}
}

func TestSweepRunsAtWriterStartAndDaily(t *testing.T) {
	dir := cacheDirIn(t)
	now := time.Now()
	writeCacheFile(t, dir, fakeName(1, ".cache"), now.Add(-48*time.Hour), now.Add(-48*time.Hour), 10)
	c := newCache(dir) // the writer sweeps at start
	c.flush()
	deadline := time.Now().Add(2 * time.Second)
	for exists(dir, fakeName(1, ".cache")) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if exists(dir, fakeName(1, ".cache")) {
		t.Fatal("the writer must sweep once at start")
	}
	if c.lastSweepTime().IsZero() {
		t.Fatal("the sweep time must be recorded so the daily rule can fire")
	}
}
