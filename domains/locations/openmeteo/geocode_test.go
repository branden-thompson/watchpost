package openmeteo

import (
	"context"
	"errors"
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
		for _, err := range []error{second(NewGeocoder(c, srv.URL).Resolve(context.Background(), typed)), second(NewGeocoder(c, srv.URL).ResolveWorld(context.Background(), typed))} {
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
}

// second is a call's error.
func second[T any](_ T, err error) error { return err }

// TestTheWorldsFailuresSayItsOwnFix is BUILD red team AX-10: the Propagation
// mode's finder, which finds anywhere, is never told to try 'City, ST' or a
// zip code; a lookup that did not answer says what is found without it.
func TestTheWorldsFailuresSayItsOwnFix(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer failing.Close()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"results":[]}`)) }))
	defer empty.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	_, err := NewGeocoder(c, empty.URL).ResolveWorld(context.Background(), "nowhere")
	if err == nil || strings.Contains(err.Error(), "City, ST") || !strings.Contains(err.Error(), "its country, coordinates or a grid locator") {
		t.Errorf("no result anywhere is said %v", err)
	}
	_, err = NewGeocoder(c, failing.URL).ResolveWorld(context.Background(), "nowhere")
	if err == nil || !strings.Contains(err.Error(), "found without it") || strings.Contains(err.Error(), "http") || errors.Unwrap(err) == nil {
		t.Errorf("a lookup that did not answer is said %v: the words alone, the cause kept for the diagnostics", err)
	}
}

// TestTheOnlineFallbackIsUncached is 0.19.0 FR-1.10 (D-61): the Propagation
// mode's lookup answers a place in any country, and keeps no answer - the
// same query asked twice is asked of the geocoder twice, where the weather's
// lookup is answered the second time from its cache.
func TestTheOnlineFallbackIsUncached(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "max-age=3600")
		_, _ = w.Write([]byte(`{"results":[{"name":"Reykjavik","latitude":64.1,"longitude":-21.9,"country_code":"IS","timezone":"Atlantic/Reykjavik"}]}`))
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	g := NewGeocoder(c, srv.URL)
	for range 2 {
		ref, err := g.ResolveWorld(context.Background(), "Reykjavik")
		if err != nil || ref.Label != "Reykjavik, IS" || ref.Lat != 64.1 {
			t.Fatalf("a place in Iceland: %+v, %v", ref, err)
		}
	}
	if hits != 2 {
		t.Errorf("the world lookup asked %d times for two lookups; never kept, it asks each time", hits)
	}
	if _, err := g.Resolve(context.Background(), "Reykjavik"); err == nil {
		t.Error("control: the weather's lookup took a place it does not cover")
	}
}

// TestTheGeocodersTextIsCleaned is FR-4.9 at the terminal (PN2-6): a
// place's name and region from the geocoder reach the terminal and the
// words, so an escape sequence or a control character in them is dropped
// before either sees it.
func TestTheGeocodersTextIsCleaned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"name":"Reyk\u001b]52;c;aGk=\u0007javik\n","latitude":64.1,"longitude":-21.9,"country_code":"I\u0007S","timezone":"Atlantic/Reykjavik"}]}`))
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := NewGeocoder(c, srv.URL).ResolveWorld(context.Background(), "Reykjavik")
	if err != nil || strings.ContainsAny(ref.Label, "\x1b\x07\n") {
		t.Errorf("the place found is labelled %q (%v); no escape or control character", ref.Label, err)
	}
}
