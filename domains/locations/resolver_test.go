package locations

import (
	"context"
	"errors"
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
	budget := 10 * time.Millisecond
	if raceEnabled {
		budget = 60 * time.Millisecond // the detector alone is a 5–10× slowdown; shared CI runners add jitter (release job, 2026-08-25: 13.9 ms)
	}
	for _, q := range []string{"New", "San", "Spring", "Portland, ME", "a"} {
		start := time.Now()
		_ = r.TypeAhead(q, 5)
		if el := time.Since(start); el > budget {
			t.Fatalf("TypeAhead(%q) took %v (>%v budget)", q, el, budget)
		}
	}
}

// offlineOnly is a fallback that fails the test when asked: nothing typed as
// coordinates or a locator may go online (FR-4.9).
type offlineOnly struct{ t *testing.T }

func (o offlineOnly) Resolve(_ context.Context, q string) (snapshot.LocationRef, error) {
	o.t.Errorf("%q went to the online geocoder", q)
	return snapshot.LocationRef{}, errors.New("asked")
}

// TestEveryResolverParsesCoordinatesLocally is FR-4.9 (W0.3): the one
// resolver every lookup goes through - the Observer's, the console's,
// `watchpost report` - reads coordinates itself, offers them as you type and
// never sends them online; a pair that is not two finite numbers on the
// globe is refused, offline, saying what one is.
func TestEveryResolverParsesCoordinatesLocally(t *testing.T) {
	r := testResolver(t, offlineOnly{t})
	ref, fell, err := r.Resolve(context.Background(), " 33.2 , -117.38 ")
	if err != nil || fell || ref.Lat != 33.2 || ref.Lon != -117.38 || ref.Label != "33.2000, -117.3800" {
		t.Fatalf("coordinates resolved to %+v (fell back %v): %v", ref, fell, err)
	}
	if s := r.TypeAhead("33.2,-117.38", 5); len(s) != 1 || s[0].Ref != ref {
		t.Errorf("the type-ahead offers %+v for coordinates", s)
	}
	for _, q := range []string{"NaN,NaN", "Inf, 0", "33.2, 400", "1e3,2"} {
		if _, _, err := r.Resolve(context.Background(), q); err == nil || !strings.Contains(err.Error(), "latitude") {
			t.Errorf("%q: %v; refused, saying what a coordinate pair is", q, err)
		}
	}
}

// TestCoordinatesAndLocatorsNeverGoOnline is FR-4.9 and FR-1.10 (W0.3): a
// Maidenhead locator is read locally to the centre of its square; a place
// typed either way in no region the weather covers is refused, offline, as a
// city there is.
func TestCoordinatesAndLocatorsNeverGoOnline(t *testing.T) {
	r := testResolver(t, offlineOnly{t})
	ref, fell, err := r.Resolve(context.Background(), "dm13AF")
	if err != nil || fell || ref.Label != "DM13af" || ref.Lat < 33 || ref.Lat > 33.25 || ref.Lon < -118 || ref.Lon > -117.9 {
		t.Fatalf("the locator resolved to %+v (fell back %v): %v", ref, fell, err)
	}
	for _, q := range []string{"51.5,-0.12", "IO91wm"} { // London
		if _, _, err := r.Resolve(context.Background(), q); err == nil || !strings.Contains(err.Error(), "outside") {
			t.Errorf("%q, outside every region: %v", q, err)
		}
	}
}

// TestNoLookupErrorQuotesTheQuery is FR-4.9 (W0.3): a place not found is
// said without quoting what was typed.
func TestNoLookupErrorQuotesTheQuery(t *testing.T) {
	const typed = "Zzqxj Private Lane"
	_, _, err := testResolver(t, nil).Resolve(context.Background(), typed)
	if err == nil || strings.Contains(err.Error(), "Zzqxj") {
		t.Errorf("offline, no match: %v", err)
	}
}

// worldFallback counts the online asks the Propagation mode's lookup makes.
type worldFallback struct {
	fakeFallback
	world int
}

func (w *worldFallback) ResolveWorld(_ context.Context, q string) (snapshot.LocationRef, error) {
	w.world++
	return snapshot.LocationRef{Label: "Online Place, XX", Lat: 10, Lon: 20}, nil
}

// TestALocalMatchSendsNothing is 0.19.0 FR-1.10 (D-61, the mode-aware
// coverage gate): the Propagation mode's lookup answers a world city from
// the embedded index, and coordinates and a locator anywhere on the globe,
// sending nothing; only a query nothing local matches goes online - where
// the weather's lookup refuses what it does not cover.
func TestALocalMatchSendsNothing(t *testing.T) {
	fb := &worldFallback{}
	r := testResolver(t, fb)
	for q, wantLat := range map[string]float64{"London": 51.5, "64.1466, -21.9426": 64.1466, "IO91wm": 51.5} {
		ref, fellBack, err := r.ResolveAnywhere(context.Background(), q)
		if err != nil || fellBack || ref.Lat < wantLat-0.5 || ref.Lat > wantLat+0.5 {
			t.Errorf("%q: %+v, fell back %v, %v", q, ref, fellBack, err)
		}
	}
	if fb.world != 0 || fb.called {
		t.Errorf("local matches went online: %d world asks, weather's %v", fb.world, fb.called)
	}
	if _, _, err := r.Resolve(context.Background(), "64.1466, -21.9426"); err == nil {
		t.Error("control: the weather's lookup took coordinates it does not cover")
	}
	if ref, fellBack, err := r.ResolveAnywhere(context.Background(), "Zzyzxville Nowhere"); err != nil || !fellBack || fb.world != 1 || ref.Label != "Online Place, XX" {
		t.Errorf("a query nothing local matches: %+v, fell back %v, %d asks, %v", ref, fellBack, fb.world, err)
	}
	offline := testResolver(t, nil)
	if _, _, err := offline.ResolveAnywhere(context.Background(), "Zzyzxville Nowhere"); err == nil {
		t.Error("with no fallback, a miss was answered")
	}
}

// TestAPositionInAnotherFormNeverGoesOnline is the BUILD red team's I-2
// (FR-4.9): a position written in a form the parsers do not read is refused
// here, with how to write it, by every lookup - never sent to the geocoder;
// a ZIP with its four, a place and not a position, is not refused as one.
func TestAPositionInAnotherFormNeverGoesOnline(t *testing.T) {
	r := testResolver(t, offlineOnly{t})
	for _, q := range []string{"33.2N 117.4W", "33°12'N 117°23'W", "33.2;-117.38", "33.2, -117.38, 10", "DM13af45xy", "DZ13", "33 12 117 23",
		"33°12′N 117°23′W", "33º12'N", "33.2, −117.38", "(33.2, -117.38)", "lat 33.2 lon -117.38", "geo:33.2,-117.38", "３３.２ １１７.３８"} { // round 2 PN2-4: the forms a list of spellings missed
		for name, resolve := range map[string]func(context.Context, string) (snapshot.LocationRef, bool, error){"the weather's": r.Resolve, "the Propagation mode's": r.ResolveAnywhere} {
			if _, fell, err := resolve(context.Background(), q); err == nil || fell || !strings.Contains(err.Error(), "latitude then longitude") {
				t.Errorf("%s lookup of %q: fell back %v, %v", name, q, fell, err)
			}
		}
	}
	if positionShaped("92003-1234") || positionShaped("29 Palms") || positionShaped("92003") || positionShaped("Area 51") {
		t.Error("a ZIP, a ZIP with its four or a town with a figure in its name reads as a position")
	}
}
