package httpx

// Quality pass Q5 (plan §2.2, the send side): an expired entry with
// validators is offered to the server; a 304 renews it without a body.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// validatingServer answers 200 with validators and Cache-Control max-age=1,
// then 304 when the request carries the right validator — unless changed
// is set, when it answers a new body.
type validatingServer struct {
	*httptest.Server
	hits, cond atomic.Int32
	changed    atomic.Bool
	lastIMS    atomic.Value // the If-Modified-Since seen last
	lastINM    atomic.Value
}

func newValidatingServer(t *testing.T, etag, lastModified string) *validatingServer {
	t.Helper()
	vs := &validatingServer{}
	vs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vs.hits.Add(1)
		ims, inm := r.Header.Get("If-Modified-Since"), r.Header.Get("If-None-Match")
		vs.lastIMS.Store(ims)
		vs.lastINM.Store(inm)
		if ims != "" || inm != "" {
			vs.cond.Add(1)
		}
		if !vs.changed.Load() && ((lastModified != "" && ims == lastModified) || (etag != "" && inm == etag)) {
			w.Header().Set("Cache-Control", "max-age=1")
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if etag != "" {
			w.Header().Set("ETag", etag)
		}
		if lastModified != "" {
			w.Header().Set("Last-Modified", lastModified)
		}
		w.Header().Set("Cache-Control", "max-age=1")
		body := "the product body"
		if vs.changed.Load() {
			body = "a newer product body"
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(vs.Close)
	return vs
}

func condClient(t *testing.T, dir string) (*Client, *time.Time) {
	t.Helper()
	c, err := New(Config{UserAgent: "t", RatePerSec: 1000, CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	c.cache.flush() // the start sweep reads the clock on the writer goroutine: let it finish before the clock is replaced
	now := time.Now()
	c.cache.now = func() time.Time { return now }
	t.Cleanup(c.cache.flush)
	return c, &now
}

func TestConditionalGETRenewsOnA304WithIfModifiedSinceFirst(t *testing.T) {
	vs := newValidatingServer(t, `"v1"`, "Tue, 26 Aug 2026 12:00:00 GMT")
	c, now := condClient(t, "")
	url := vs.URL + "/product"
	body, err := c.GetText(context.Background(), url)
	if err != nil || string(body) != "the product body" {
		t.Fatalf("first fetch: %v %q", err, body)
	}
	*now = now.Add(2 * time.Second) // past max-age=1: stale, with validators
	body, err = c.GetText(context.Background(), url)
	if err != nil || string(body) != "the product body" {
		t.Fatalf("revalidated fetch serves the stored body: %v %q", err, body)
	}
	if vs.hits.Load() != 2 || vs.cond.Load() != 1 {
		t.Fatalf("one plain GET, one conditional: hits=%d cond=%d", vs.hits.Load(), vs.cond.Load())
	}
	if vs.lastIMS.Load() != "Tue, 26 Aug 2026 12:00:00 GMT" || vs.lastINM.Load() != `"v1"` {
		t.Fatalf("both validators are sent, If-Modified-Since first: %v / %v", vs.lastIMS.Load(), vs.lastINM.Load())
	}
	st := c.RequestStats().Hosts[0]
	if st.NotModified != 1 || st.Bytes304 != int64(len("the product body")) || st.Net != 1 {
		t.Fatalf("the 304 is counted with the bytes it saved: %+v", st)
	}
	// Renewed: a third read inside the new lifetime is a cache hit.
	if _, err := c.GetText(context.Background(), url); err != nil || vs.hits.Load() != 2 {
		t.Fatalf("renewed entry serves from cache: hits=%d err=%v", vs.hits.Load(), err)
	}
}

func TestConditionalGETTakesTheNewBodyOnA200(t *testing.T) {
	vs := newValidatingServer(t, "", "Tue, 26 Aug 2026 12:00:00 GMT")
	c, now := condClient(t, "")
	url := vs.URL + "/product"
	_, _ = c.GetText(context.Background(), url)
	*now = now.Add(2 * time.Second)
	vs.changed.Store(true)
	body, err := c.GetText(context.Background(), url)
	if err != nil || string(body) != "a newer product body" {
		t.Fatalf("a 200 replaces the stored body: %v %q", err, body)
	}
	if st := c.RequestStats().Hosts[0]; st.NotModified != 0 || st.Net != 2 {
		t.Fatalf("no 304 counted: %+v", st)
	}
}

func TestNoValidatorsMeansAPlainGET(t *testing.T) {
	vs := newValidatingServer(t, "", "")
	c, now := condClient(t, "")
	url := vs.URL + "/product"
	_, _ = c.GetText(context.Background(), url)
	*now = now.Add(2 * time.Second)
	_, _ = c.GetText(context.Background(), url)
	if vs.hits.Load() != 2 || vs.cond.Load() != 0 {
		t.Fatalf("without validators there is nothing to revalidate: hits=%d cond=%d", vs.hits.Load(), vs.cond.Load())
	}
}

func TestA304ToAnUnconditionalGETIsRefetchedOnce(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusNotModified) // a broken server: 304 with nothing to renew
			return
		}
		_, _ = w.Write([]byte("body at last"))
	}))
	defer srv.Close()
	c, _ := condClient(t, "")
	body, err := c.GetText(context.Background(), srv.URL+"/x")
	if err != nil || string(body) != "body at last" || hits.Load() != 2 {
		t.Fatalf("one bounded refetch, never a loop: %v %q hits=%d", err, body, hits.Load())
	}
}

func TestA304RenewsAnEntryEvictedToDisk(t *testing.T) {
	// PF-4: the memory budget dropped the entry; the file with its
	// validators is what the conditional GET revalidates, and the renewal
	// moves the file's mtime (the sweep's `max(expires, mtime) + grace`).
	vs := newValidatingServer(t, `"v1"`, "Tue, 26 Aug 2026 12:00:00 GMT")
	dir := t.TempDir()
	c, now := condClient(t, dir)
	url := vs.URL + "/product"
	if _, err := c.GetText(context.Background(), url, TTL(10*time.Minute)); err != nil { // past the persistence floor: on disk
		t.Fatal(err)
	}
	c.cache.flush()
	path := c.cache.path(url)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	c.cache.mu.Lock()
	delete(c.cache.mem, url) // evicted from memory (the budget), file kept
	c.cache.mu.Unlock()
	*now = now.Add(11 * time.Minute)
	body, err := c.GetText(context.Background(), url, TTL(10*time.Minute))
	if err != nil || string(body) != "the product body" || vs.cond.Load() != 1 {
		t.Fatalf("the disk entry's validators revalidate: %v %q cond=%d", err, body, vs.cond.Load())
	}
	c.cache.flush()
	st, err := os.Stat(path)
	if err != nil || !st.ModTime().After(old.Add(time.Hour)) {
		t.Fatalf("the 304 moved the file's mtime (Chtimes on the writer): %v %v", err, st.ModTime())
	}
	c.cache.mu.Lock()
	_, inMem := c.cache.mem[url]
	c.cache.mu.Unlock()
	if !inMem {
		t.Fatal("the renewed entry is back in memory")
	}
}
