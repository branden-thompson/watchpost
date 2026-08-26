package httpx

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Spec: architecture.md §3/§10 — single client: mandatory User-Agent (AI-1: empty
// UA = 403 from NWS), token bucket ~5 req/s, jittered backoff on 429/5xx,
// last-good semantics live in sched; secrets never appear in errors or logs
// (IS-1/§10.5: keys ride query strings for some providers).

func TestUserAgentAlwaysSent(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.Header.Get("User-Agent"))
	}))
	defer srv.Close()
	c := mustNew(t, Config{UserAgent: "watchpost/test (contact@example.com)", RatePerSec: 100})
	if _, err := c.GetJSON(context.Background(), srv.URL, nil); err != nil {
		t.Fatal(err)
	}
	if ua := got.Load().(string); ua != "watchpost/test (contact@example.com)" {
		t.Fatalf("UA = %q", ua)
	}
}

func mustNew(t *testing.T, cfg Config) *Client {
	t.Helper()
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewRequiresUserAgent(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New must refuse an empty UserAgent (NWS 403s without one)")
	}
}

func TestRetryOn5xxThenSuccess(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := mustNew(t, Config{UserAgent: "t", RatePerSec: 100, RetryBase: time.Millisecond, MaxRetries: 4})
	var out struct{ OK bool }
	if _, err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || n.Load() != 3 {
		t.Fatalf("retries=%d ok=%v", n.Load(), out.OK)
	}
}

func TestRetriesExhaustedReturnsActionableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := mustNew(t, Config{UserAgent: "t", RatePerSec: 100, RetryBase: time.Millisecond, MaxRetries: 2})
	_, err := c.GetJSON(context.Background(), srv.URL, nil)
	if err == nil {
		t.Fatal("exhausted retries must error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("error must name the status: %v", err)
	}
}

func TestSecretsRedactedFromErrors(t *testing.T) {
	c := mustNew(t, Config{UserAgent: "t", RatePerSec: 100, RetryBase: time.Millisecond, MaxRetries: 1})
	// Unreachable host guarantees a transport error embedding the URL.
	_, err := c.GetJSON(context.Background(), "http://127.0.0.1:1/data?appid=SECRET123&lat=1", nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	if strings.Contains(err.Error(), "SECRET123") {
		t.Fatalf("secret leaked into error: %v", err)
	}
	if !strings.Contains(err.Error(), "appid=REDACTED") {
		t.Fatalf("redaction marker missing: %v", err)
	}
	// B0 red-team F1: the transport CAUSE must survive (through redactErr),
	// not a fake "HTTP 0" — this exercises the *url.Error rewrite path.
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("transport cause missing from error: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot reach") {
		t.Fatalf("exhaustion message must use the transport-cause path: %v", err)
	}
}

func TestRedactURLPathSegmentKey(t *testing.T) {
	// FIRMS-style path key (S-1 tripwire for B5): 32-hex segment is masked.
	in := "https://firms.modaps.eosdis.nasa.gov/api/area/csv/0123456789abcdef0123456789abcdef/VIIRS_SNPP_NRT/-125,32,-114,42/1"
	out := RedactURL(in)
	if strings.Contains(out, "0123456789abcdef0123456789abcdef") {
		t.Fatalf("path-segment key survived redaction: %s", out)
	}
	if !strings.Contains(out, "REDACTED") || !strings.Contains(out, "VIIRS_SNPP_NRT") {
		t.Fatalf("redaction shape wrong: %s", out)
	}
}

func TestZeroRetriesIsExpressible(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := mustNew(t, Config{UserAgent: "t", RatePerSec: 100, RetryBase: time.Millisecond, MaxRetries: 0})
	if _, err := c.GetJSON(context.Background(), srv.URL, nil); err == nil {
		t.Fatal("must fail")
	}
	if n.Load() != 1 {
		t.Fatalf("MaxRetries 0 (the zero value) must mean exactly one attempt, got %d", n.Load())
	}
	if _, err := New(Config{UserAgent: "t", MaxRetries: -1}); err == nil {
		t.Fatal("the old -1 encoding is refused (quality pass Q1, PA-7)")
	}
}

func TestRedactURL(t *testing.T) {
	in := "https://api.example.com/v1?key=abc&appid=def&access_key=ghi&token=jkl&lat=33.2"
	out := RedactURL(in)
	for _, s := range []string{"abc", "def", "ghi", "jkl"} {
		if strings.Contains(out, s) {
			t.Fatalf("secret %q survived redaction: %s", s, out)
		}
	}
	if !strings.Contains(out, "lat=33.2") {
		t.Fatalf("non-secret params must survive: %s", out)
	}
}

func TestTokenBucketThrottles(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := mustNew(t, Config{UserAgent: "t", RatePerSec: 20}) // 50ms per token
	start := time.Now()
	for range 5 {
		if _, err := c.GetJSON(context.Background(), srv.URL, nil); err != nil {
			t.Fatal(err)
		}
	}
	// 5 requests at 20/s: first is immediate, the rest wait ≈50ms each ⇒ ≥150ms total.
	if el := time.Since(start); el < 150*time.Millisecond {
		t.Fatalf("token bucket did not throttle: 5 requests in %v", el)
	}
}

func TestPriorityLaneNeverQueuesBehindNormalRequests(t *testing.T) {
	// UAT 64: with the normal lane booked solid for seconds, a priority
	// request still starts at once; the lanes pace independently.
	c, err := New(Config{UserAgent: "t (t@example.com)", RatePerSec: 5})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for range 10 { // book 2 s of normal-lane slots
		if err := c.reserve(ctx); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	if err := c.reserve(WithPriority(ctx)); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("priority request waited %v behind the normal lane", time.Since(start))
	}
	// The priority lane paces itself: a second one waits its interval.
	start = time.Now()
	_ = c.reserve(WithPriority(ctx))
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Fatalf("priority lane must still pace at the configured rate, waited %v", d)
	}
}

// UAT 71: URL-keyed cache — lifetime rules, singleflight, disk tier,
// negative cache. Hits must never consume a pacing token.

func cacheServer(t *testing.T, cc string, status int, delay time.Duration) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(delay)
		if cc != "" {
			w.Header().Set("Cache-Control", cc)
		}
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"n":%d}`, hits.Load())
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func newCached(t *testing.T, dir string) *Client {
	t.Helper()
	c, err := New(Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0, CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Drain the writer before the temp dir goes: cleanups run last-registered
	// first, so this precedes t.TempDir()'s removal (a CI runner caught the
	// writer landing a file into a directory being removed).
	t.Cleanup(c.cache.flush)
	return c
}

func TestCacheLifetimeRules(t *testing.T) {
	ctx := context.Background()
	var out struct{ N int }
	// 1. No headers, no TTL: every call fetches.
	srv, hits := cacheServer(t, "", 200, 0)
	c := newCached(t, "")
	for range 2 {
		if _, err := c.GetJSON(ctx, srv.URL+"/a", &out); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("uncacheable responses must fetch every time, got %d hits", hits.Load())
	}
	// 2. Server max-age: reused within it.
	srv2, hits2 := cacheServer(t, "public, max-age=60", 200, 0)
	for range 3 {
		if _, err := c.GetJSON(ctx, srv2.URL+"/b", &out); err != nil {
			t.Fatal(err)
		}
	}
	if hits2.Load() != 1 || out.N != 1 {
		t.Fatalf("server max-age must be honoured: %d hits, n=%d", hits2.Load(), out.N)
	}
	// 3. NoCache bypasses even a cacheable response.
	if _, err := c.GetJSON(ctx, srv2.URL+"/b", &out, NoCache()); err != nil || hits2.Load() != 2 {
		t.Fatalf("NoCache must fetch: %v, %d hits", err, hits2.Load())
	}
	// 4. Explicit TTL outranks no-store.
	srv3, hits3 := cacheServer(t, "no-cache, no-store", 200, 0)
	for range 2 {
		if _, err := c.GetJSON(ctx, srv3.URL+"/c", &out, TTL(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if hits3.Load() != 1 {
		t.Fatalf("explicit TTL must outrank server no-store, got %d hits", hits3.Load())
	}
}

func TestCacheDiskTierWarmsARelaunch(t *testing.T) {
	dir := t.TempDir()
	srv, hits := cacheServer(t, "max-age=3600", 200, 0)
	var out struct{ N int }
	first := newCached(t, dir)
	if _, err := first.GetJSON(context.Background(), srv.URL+"/points", &out); err != nil {
		t.Fatal(err)
	}
	first.cache.flush() // disk writes are queued off the request path (UAT 73)
	// A fresh client (a new process) with the same dir is served from disk.
	if _, err := newCached(t, dir).GetJSON(context.Background(), srv.URL+"/points", &out); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 || out.N != 1 {
		t.Fatalf("relaunch must be served from the disk tier: %d hits", hits.Load())
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 || !strings.HasSuffix(files[0].Name(), ".cache") {
		t.Fatalf("one inspectable entry expected, got %v", files)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if !strings.HasPrefix(string(raw), `{"url":"`) || !strings.HasSuffix(string(raw), `{"n":1}`) {
		t.Fatalf("file must be a JSON header line followed by the raw body: %q", raw)
	}
}

func TestCacheMemoryBudget(t *testing.T) {
	// UAT 73: the memory tier stays within maxMemBytes (LRU eviction) and
	// bodies larger than a quarter of it are disk-only; both remain
	// servable from disk.
	dir := t.TempDir()
	big := strings.Repeat("x", maxMemEntry+1)
	var served atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		w.Header().Set("Cache-Control", "max-age=600")
		if strings.HasPrefix(r.URL.Path, "/big") {
			_, _ = w.Write([]byte(big))
			return
		}
		_, _ = w.Write(bytes.Repeat([]byte("y"), 1<<20)) // 1 MB
	}))
	defer srv.Close()
	c := newCached(t, dir)
	if _, err := c.GetText(context.Background(), srv.URL+"/big"); err != nil {
		t.Fatal(err)
	}
	if st := c.CacheStats(); st.Entries != 0 {
		t.Fatalf("oversized bodies are disk-only, got %+v", st)
	}
	for i := range 12 {
		if _, err := c.GetText(context.Background(), fmt.Sprintf("%s/mb/%d", srv.URL, i)); err != nil {
			t.Fatal(err)
		}
	}
	if st := c.CacheStats(); st.Bytes > maxMemBytes || st.Entries >= 12 {
		t.Fatalf("memory tier must evict to its budget, got %+v", st)
	}
	c.cache.flush()
	before := served.Load()
	if body, err := c.GetText(context.Background(), srv.URL+"/big"); err != nil || len(body) != len(big) {
		t.Fatal("disk-only entry must still be served", err)
	}
	if _, err := c.GetText(context.Background(), srv.URL+"/mb/0"); err != nil { // evicted from memory, on disk
		t.Fatal(err)
	}
	if served.Load() != before {
		t.Fatal("evicted and oversized entries are served from disk, not refetched")
	}
}

func TestCacheSingleflightAndNegative(t *testing.T) {
	srv, hits := cacheServer(t, "max-age=60", 200, 30*time.Millisecond)
	c := newCached(t, "")
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.GetJSON(context.Background(), srv.URL+"/g", nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if hits.Load() != 1 {
		t.Fatalf("concurrent identical GETs must share one request, got %d", hits.Load())
	}
	dead, deadHits := cacheServer(t, "", 404, 0)
	for range 3 {
		if _, err := c.GetJSON(context.Background(), dead.URL+"/x", nil); err == nil {
			t.Fatal("404 must error")
		}
	}
	if deadHits.Load() != 1 {
		t.Fatalf("a non-retryable failure is remembered for NegativeTTL, got %d hits", deadHits.Load())
	}
}
