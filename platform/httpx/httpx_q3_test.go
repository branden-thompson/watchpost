package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
