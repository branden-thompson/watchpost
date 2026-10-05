package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// CACHED NEVER ASKS THE NETWORK (D-217): it answers from a fresh entry, memory
// or disk, or says there is none - an expired entry is none.
func TestCachedNeverAsksTheNetwork(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	c, err := New(Config{UserAgent: "watchpost/test", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	u := srv.URL + "/x"
	if _, ok := c.Cached(u); ok || hits.Load() != 0 {
		t.Fatalf("an unasked address read as cached, or was asked (%d)", hits.Load())
	}
	if _, err := c.GetText(context.Background(), u, TTL(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if b, ok := c.Cached(u); !ok || string(b) != "body" || hits.Load() != 1 {
		t.Errorf("the cached body is %q, %v after %d asks; want it, after one", b, ok, hits.Load())
	}
	c.cache.flush()
	again, err := New(Config{UserAgent: "watchpost/test", CacheDir: c.cfg.CacheDir})
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := again.Cached(u); !ok || string(b) != "body" {
		t.Errorf("the disk's fresh copy was not read: %q, %v", b, ok)
	}
	later := time.Now().Add(2 * time.Hour)
	c.cache.now = func() time.Time { return later }
	if _, ok := c.Cached(u); ok || hits.Load() != 1 {
		t.Errorf("an expired entry read as cached, or the network was asked (%d)", hits.Load())
	}
}

// THE TIERS ARE SIZED BY THE CONFIG (D-219): a client may hold more in memory
// than the package's default, and cap its disk below it; zero is the default.
func TestTheTiersAreSizedByTheConfig(t *testing.T) {
	c, err := New(Config{UserAgent: "watchpost/test", MemBytes: 24 << 20, DiskBytes: 64 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if c.cache.memMax != 24<<20 || c.cache.memEntryMax != 6<<20 || c.cache.maxDiskBytes != 64<<20 {
		t.Errorf("the tiers are %d (entry %d) in memory, %d on disk; want 24 MiB (6), 64 MiB", c.cache.memMax, c.cache.memEntryMax, c.cache.maxDiskBytes)
	}
	d, err := New(Config{UserAgent: "watchpost/test"})
	if err != nil {
		t.Fatal(err)
	}
	if d.cache.memMax != maxMemBytes || d.cache.memEntryMax != maxMemEntry || d.cache.maxDiskBytes != maxDiskBytes {
		t.Errorf("the default tiers are %d (entry %d), %d", d.cache.memMax, d.cache.memEntryMax, d.cache.maxDiskBytes)
	}
	// AND THE SIZE IS OBEYED: three bodies of 3 MiB fit a 24 MiB tier, where the
	// default's 2 MiB entry limit would send each to the large tier.
	body := make([]byte, 3<<20)
	for i, u := range []string{"https://a.example/1", "https://a.example/2", "https://a.example/3"} {
		c.cache.remember(u, entry{URL: u, Body: body, Expires: time.Now().Add(time.Hour)})
		if _, ok := c.cache.mem[u]; !ok {
			t.Fatalf("body %d is not in the sized small tier", i)
		}
	}
	small := newCacheSized("", maxDiskBytes, 4<<20)
	for _, u := range []string{"https://a.example/1", "https://a.example/2"} {
		small.remember(u, entry{URL: u, Body: make([]byte, 1<<20), Expires: time.Now().Add(time.Hour)})
	}
	small.remember("https://a.example/3", entry{URL: "https://a.example/3", Body: make([]byte, 1<<20), Expires: time.Now().Add(time.Hour)})
	small.remember("https://a.example/4", entry{URL: "https://a.example/4", Body: make([]byte, 1<<20), Expires: time.Now().Add(time.Hour)})
	small.remember("https://a.example/5", entry{URL: "https://a.example/5", Body: make([]byte, 1<<20), Expires: time.Now().Add(time.Hour)})
	if small.bytes > 4<<20 {
		t.Errorf("a 4 MiB tier holds %d bytes", small.bytes)
	}
}
