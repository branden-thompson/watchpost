package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestARepliesHeadersAreBounded is the REVIEW red team's RI-4: a reply
// whose headers pass maxHeaderBytes is refused, as a body past its cap is,
// never read whole into memory.
func TestARepliesHeadersAreBounded(t *testing.T) {
	pad := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Padding", strings.Repeat("a", pad))
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c, err := New(Config{UserAgent: "t", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	var v struct{}
	pad = 100
	if _, err := c.GetJSON(context.Background(), srv.URL, &v); err != nil {
		t.Fatalf("control: a reply with small headers: %v", err)
	}
	pad = 2 * maxHeaderBytes
	if _, err := c.GetJSON(context.Background(), srv.URL, &v); err == nil {
		t.Errorf("a reply with %d bytes of headers was read", 2*maxHeaderBytes)
	}
}
