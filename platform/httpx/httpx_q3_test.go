package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Quality pass Q3 (L1-F9): GetText hands out the cache's own slice — no
// copy per call. The read-only side of the contract is pinned in every
// consumer package (TestGetTextCallersMustNotMutate).
func TestGetTextAliasesTheCachedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte("a text product"))
	}))
	defer srv.Close()
	c, err := New(Config{UserAgent: "t"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := c.GetText(context.Background(), srv.URL+"/p")
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.GetText(context.Background(), srv.URL+"/p")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || &first[0] != &second[0] {
		t.Fatal("a cache hit returns the stored body itself, not a copy (read-only by contract)")
	}
}

// Quality pass Q5 (L4-F13, L2-F11): the app-wide transport keeps an idle
// connection across a 10-minute tier, so the tick reuses the session
// instead of paying a TLS handshake per host per tick (the 24 h soak's
// counters: FIRMS 178, CO-OPS 140, NDBC 91 handshakes in nine hours at 90 s).
func TestNewTransportOutlivesATenMinuteTier(t *testing.T) {
	tr := NewTransport()
	if tr.IdleConnTimeout <= 10*time.Minute || !tr.ForceAttemptHTTP2 || tr.MaxConnsPerHost != maxConnsPerHost {
		t.Fatalf("idle %v h2 %v conns %d", tr.IdleConnTimeout, tr.ForceAttemptHTTP2, tr.MaxConnsPerHost)
	}
	tr.MaxConnsPerHost = 2 // a caller's copy is its own
	if NewTransport().MaxConnsPerHost != maxConnsPerHost {
		t.Fatal("each caller gets a fresh transport")
	}
}
