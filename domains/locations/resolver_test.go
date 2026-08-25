package locations

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var (
	gIdx  *geodata.Index
	gOnce sync.Once
)

func testResolver(t *testing.T, fb Fallback) *Resolver {
	t.Helper()
	gOnce.Do(func() {
		var err error
		gIdx, err = geodata.Load()
		if err != nil {
			t.Fatal(err)
		}
	})
	r, err := New(gIdx, fb)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type fakeFallback struct{ called bool }

func (f *fakeFallback) Resolve(_ context.Context, q string) (snapshot.LocationRef, error) {
	f.called = true
	return snapshot.LocationRef{Label: "Fallback City", Lat: 1, Lon: 2}, nil
}

func TestResolveZipOffline(t *testing.T) {
	fb := &fakeFallback{}
	r := testResolver(t, fb)
	ref, fell, err := r.Resolve(context.Background(), "92057")
	if err != nil {
		t.Fatal(err)
	}
	if fell || fb.called {
		t.Fatal("known zip must resolve offline")
	}
	if ref.Zip != "92057" || !strings.Contains(ref.Label, "CA") {
		t.Fatalf("ref = %+v", ref)
	}
	if ref.TZ != "America/Los_Angeles" {
		t.Fatalf("zip resolve must backfill TZ from the city index (B2 PTY finding): %+v", ref)
	}
}

func TestResolveCityWithStateQualifier(t *testing.T) {
	r := testResolver(t, nil)
	ref, fell, err := r.Resolve(context.Background(), "Portland, ME")
	if err != nil {
		t.Fatal(err)
	}
	if fell {
		t.Fatal("offline hit expected")
	}
	if ref.Label != "Portland, ME" {
		t.Fatalf("state qualifier must disambiguate: %+v", ref)
	}
	// Without qualifier, the bigger Portland wins by population.
	ref2, _, err := r.Resolve(context.Background(), "Portland")
	if err != nil {
		t.Fatal(err)
	}
	if ref2.Label != "Portland, OR" {
		t.Fatalf("population rank must pick Portland, OR: %+v", ref2)
	}
}

func TestResolveMissFallsBack(t *testing.T) {
	fb := &fakeFallback{}
	r := testResolver(t, fb)
	ref, fell, err := r.Resolve(context.Background(), "Tinytown Nowhere")
	if err != nil {
		t.Fatal(err)
	}
	if !fell || !fb.called || ref.Label != "Fallback City" {
		t.Fatalf("miss must fall back: fell=%v called=%v ref=%+v", fell, fb.called, ref)
	}
}

func TestResolveMissWithoutFallbackIsActionable(t *testing.T) {
	r := testResolver(t, nil)
	_, _, err := r.Resolve(context.Background(), "Tinytown Nowhere")
	if err == nil || !strings.Contains(err.Error(), "City, ST") {
		t.Fatalf("offline miss must guide the user: %v", err)
	}
}

func TestTypeAheadZipAdornedAndRanked(t *testing.T) {
	r := testResolver(t, nil)
	hints := r.TypeAhead("san", 5)
	if len(hints) == 0 {
		t.Fatal("no hints")
	}
	for _, h := range hints {
		if h.Ref.Label == "" {
			t.Fatalf("hint without label: %+v", h)
		}
		if h.Ref.Zip != "" && !strings.Contains(h.Display, "("+h.Ref.Zip+")") {
			t.Fatalf("hint with a zip must show it (R-2'): %q", h.Display)
		}
	}
	// Zip query gives exactly one suggestion.
	zh := r.TypeAhead("92057", 5)
	if len(zh) != 1 || !strings.Contains(zh[0].Display, "(92057)") {
		t.Fatalf("zip type-ahead: %+v", zh)
	}
	// Never hits the network.
	fb := &fakeFallback{}
	r2 := testResolver(t, fb)
	r2.TypeAhead("qqqqqzzz", 5)
	if fb.called {
		t.Fatal("type-ahead must never call the online fallback (AI-8 ToS)")
	}
}

func TestTypeAheadSingleCallBudget(t *testing.T) {
	// B2 red-team #3: the old budget test averaged warm PrefixSearch and never
	// exercised TypeAhead's zip-adornment path. Enforce the <10ms budget on
	// single cold-ish TypeAhead calls including qualifier filtering.
	r := testResolver(t, nil)
	for _, q := range []string{"New", "San", "Spring", "Portland, ME", "a"} {
		start := time.Now()
		_ = r.TypeAhead(q, 5)
		if el := time.Since(start); el > 10*time.Millisecond {
			t.Fatalf("TypeAhead(%q) took %v (>10ms budget)", q, el)
		}
	}
}
