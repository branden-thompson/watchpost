package openmeteo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

func TestResolveZipQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "92057" {
			t.Errorf("query name = %q", got)
		}
		b, _ := os.ReadFile("testdata/geocode.json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := NewGeocoder(c, srv.URL).Resolve(context.Background(), "92057")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Label != "Oceanside, CA" {
		t.Fatalf("label = %q", ref.Label)
	}
	if ref.Zip != "92057" {
		t.Fatalf("zip = %q — query-matching zip must win (AI-8 rule)", ref.Zip)
	}
	if ref.TZ != "America/Los_Angeles" || ref.Lat == 0 {
		t.Fatalf("ref = %+v", ref)
	}
}

func TestResolveNoResultIsActionable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	_, err := NewGeocoder(c, srv.URL).Resolve(context.Background(), "xyzzy")
	if err == nil {
		t.Fatal("must error on no results")
	}
}

// TestTheGeocoderQueryIsRedacted is FR-4.9 (W0.3): what was typed never
// appears in an error - not in the words, not in the URL a failed request
// names, where `name` is redacted as a key would be.
func TestTheGeocoderQueryIsRedacted(t *testing.T) {
	const typed = "Zzqxj Private Lane"
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer failing.Close()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"results":[]}`)) }))
	defer empty.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	for name, srv := range map[string]*httptest.Server{"a failed request": failing, "no result": empty} {
		_, err := NewGeocoder(c, srv.URL).Resolve(context.Background(), typed)
		if err == nil {
			t.Fatalf("%s: no error", name)
		}
		for _, part := range []string{typed, "Zzqxj", "Zzqxj+Private", "Zzqxj%20Private"} {
			if strings.Contains(err.Error(), part) {
				t.Errorf("%s: the error quotes what was typed: %v", name, err)
			}
		}
	}
}
