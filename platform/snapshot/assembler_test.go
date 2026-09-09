package snapshot

import (
	"strings"
	"testing"
	"time"
)

func TestMarineMergesForecastThenBuoy(t *testing.T) {
	// B3 UAT 29: nws-marine supplies swell/wave; ndbc fills water temp (+ its
	// own wave obs only where the forecast left a hole); inland stays null.
	ref := LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	inland := LocationRef{Label: "Phoenix, AZ", Zip: "85001", Lat: 33.4, Lon: -112.0}
	a := NewAssembler([]LocationRef{ref, inland}, []string{"nws", "nws-marine", "ndbc"})
	f := func(v float64) *float64 { return &v }
	a.Apply(Fragment{Provider: "nws-marine", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{SwellHeight: f(0.6), SwellDirDeg: f(280), WavePeriod: f(6), Source: SourceInfo{Provider: "nws-marine"}}},
	}}, nil)
	a.Apply(Fragment{Provider: "ndbc", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{WaveHeight: f(0.9), WavePeriod: f(14), WaterTemp: f(23.3), Buoy: "46224"}},
	}}, nil)
	s := a.Snapshot()
	m := s.Locations[0].Marine
	if m == nil {
		t.Fatal("coastal location must publish a marine section")
	}
	if *m.SwellHeight != 0.6 || *m.WavePeriod != 6 {
		t.Fatalf("forecast provider must win existing fields: %+v", m)
	}
	if *m.WaveHeight != 0.9 || *m.WaterTemp != 23.3 || m.Buoy != "46224" {
		t.Fatalf("buoy must fill the holes: %+v", m)
	}
	if s.Locations[1].Marine != nil {
		t.Fatal("inland location must publish marine: null")
	}
	if len(s.Providers) != 3 {
		t.Fatalf("status must track all three providers, got %d", len(s.Providers))
	}
}

func TestMarineTidesFillFromCoops(t *testing.T) {
	// B3 UAT 61: the tide/current block merges into the buoy section; the
	// event slices never alias assembler state.
	ref := LocationRef{Label: "San Diego, CA", Zip: "92101", Lat: 32.7157, Lon: -117.1611}
	a := NewAssembler([]LocationRef{ref}, []string{"nws", "ndbc", "coops"})
	f := func(v float64) *float64 { return &v }
	when := time.Date(2026, 8, 25, 2, 40, 0, 0, time.UTC)
	a.Apply(Fragment{Provider: "ndbc", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{WaveHeight: f(0.6), WaterTemp: f(23.9), Buoy: "46254"}},
	}}, nil)
	a.Apply(Fragment{Provider: "coops", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{TideLevel: f(1.13), Tides: []TideEvent{{Time: when, Height: 1.73, Type: "H"}},
			TideStation: "San Diego", TideStationKM: f(1.2), Currents: []CurrentEvent{{Time: when, Speed: 0.5, Type: "flood"}}, CurrentStation: "San Diego Bay Entrance"}},
	}}, nil)
	s := a.Snapshot()
	m := s.Locations[0].Marine
	if m == nil || m.Buoy != "46254" || m.TideStation != "San Diego" || len(m.Tides) != 1 || m.Tides[0].Height != 1.73 || len(m.Currents) != 1 || *m.TideLevel != 1.13 {
		t.Fatalf("tides must fill the buoy section: %+v", m)
	}
	m.Tides[0].Height = 99
	if a.Snapshot().Locations[0].Marine.Tides[0].Height != 1.73 {
		t.Fatal("published tide events must not alias assembler state")
	}
	inland := NewAssembler([]LocationRef{ref}, []string{"nws"}).Snapshot()
	if inland.Locations[0].Marine != nil {
		t.Fatal("no marine data stays null")
	}
}

// A PROVIDER'S OWN WORDS REACH A TERMINAL, so they cross the plaintext boundary
// on the way into a warning rather than at each surface that renders one.
//
// Feeds put their error envelope's text straight into a message, and a server
// that controls that text controls a byte stream the terminal will interpret:
// OSC 52 writes the reader's clipboard, OSC 8 paints a hyperlink over honest
// text, a bidi override reverses a line. Cleaning here means `watchpost report`,
// the [S] window and every future surface are safe without knowing they had to be.
func TestWarningTextIsCleanedAtTheBoundary(t *testing.T) {
	const (
		clipboardWrite = "\x1b]52;c;cm0gLXJmIH4K\a"
		hyperlink      = "\x1b]8;;https://evil.example\aCLICK\x1b]8;;\a"
		titleSet       = "\x1b]0;PWNED\a"
		bidiOverride   = "\u202e" // right-to-left override: reverses a rendered line
	)
	hostile := "coops: " + clipboardWrite + titleSet + hyperlink + bidiOverride + " bad station"

	a := NewAssembler([]LocationRef{{Label: "A"}}, nil)
	a.Warn(Warning{Code: WarnProviderError, Provider: "coops", Message: hostile,
		Endpoint: "api.tidesandcurrents.noaa.gov" + titleSet})
	snap := a.Snapshot()
	if len(snap.Warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(snap.Warnings))
	}
	got := snap.Warnings[0]
	for _, sequence := range []struct{ name, raw string }{
		{"clipboard write (OSC 52)", clipboardWrite},
		{"hyperlink (OSC 8)", "\x1b]8;;"},
		{"window title (OSC 0)", titleSet},
		{"bidi override", bidiOverride},
		{"any escape at all", "\x1b"},
	} {
		if strings.Contains(got.Message, sequence.raw) {
			t.Errorf("the %s survived into the message: %q", sequence.name, got.Message)
		}
		if strings.Contains(got.Endpoint, sequence.raw) {
			t.Errorf("the %s survived into the endpoint: %q", sequence.name, got.Endpoint)
		}
	}
	// The words themselves are kept — this sanitises, it does not censor.
	if !strings.Contains(got.Message, "bad station") {
		t.Errorf("the provider's actual words must survive: %q", got.Message)
	}
}

// THE ATTEMPT IS RECORDED FOR THE LOCATIONS THAT WERE ASKED ABOUT (#13).
//
// This is the producer half of the never-resolving lookup. The view can only
// tell "no data yet" from "no data ever" if something records that a fetch
// COVERED a location, and PerLocation cannot: a place the provider had nothing
// for is absent from it in exactly the same way as a place nobody asked about.
//
// Each clause below is a way of getting this wrong that looks right: stamping
// only what came back, stamping on a failed fetch, letting a secondary answer
// a question about the reference, and stamping locations the fetch never
// covered.
func TestTheReferenceFetchRecordsWhichLocationsItCovered(t *testing.T) {
	served := LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.38}
	barren := LocationRef{Label: "Nowhere, XX", Zip: "00000", Lat: 1, Lon: 1}
	unasked := LocationRef{Label: "Elsewhere, YY", Zip: "11111", Lat: 2, Lon: 2}
	at := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

	build := func() *Assembler {
		a := NewAssembler([]LocationRef{served, barren, unasked}, []string{"nws", "ndbc"})
		a.SetAttribution("nws", "reference", "NWS")
		a.SetAttribution("ndbc", "secondary", "NDBC")
		return a
	}
	stamp := func(a *Assembler, ref LocationRef) time.Time {
		for _, l := range a.Snapshot().Locations {
			if l.Label == ref.Label {
				return l.WeatherAsOf
			}
		}
		t.Fatalf("%s missing from the snapshot", ref.Label)
		return time.Time{}
	}
	asked := []LocationKey{Key(served), Key(barren)}

	// BOTH ANSWERING KINDS: a row is loading until it has conditions AND a daily
	// forecast, so either alone leaves it loading (see
	// TestOnlyTheAnsweringFetchesEndTheShimmer).
	a := build()
	for _, k := range []FetchKind{KindObs, KindForecast} {
		a.Apply(Fragment{Provider: "nws", Kind: k, FetchedAt: at,
			PerLocation: map[LocationKey]PartialData{Key(served): {Current: &Conditions{Source: SourceInfo{Provider: "nws"}}}}}, asked)
	}

	if got := stamp(a, served); !got.Equal(at) {
		t.Errorf("a served location is stamped: %v", got)
	}
	// THE WHOLE POINT: asked about, nothing came back, and it says so.
	if got := stamp(a, barren); !got.Equal(at) {
		t.Errorf("a location the feed had NOTHING for is still one it ANSWERED about: %v", got)
	}
	if got := stamp(a, unasked); !got.IsZero() {
		t.Errorf("a location outside the fetch was not attempted; stamping it would report a wait as a fact: %v", got)
	}

	// A FRAGMENT CARRYING AN ERROR STILL RECORDS THE ATTEMPT, and this assertion
	// is the reverse of what it said before (red team, 2026-09-08).
	//
	// FetchEach JOINS per-location errors into one Fragment.Err, so a single
	// location the API cannot serve marks the whole fragment failed — and "the
	// API does not answer for this location" IS issue #13. The old rule skipped
	// the stamp on any error, which meant the one case the field existed for was
	// the one case it never recorded. Reaching Apply means the provider
	// responded; a transport failure returns an error from Fetch and never
	// arrives here.
	// SERVED IS THE PROOF THE PROVIDER WAS REACHABLE. A fragment that carries an
	// error but served somebody says this location has nothing; one that served
	// NOBODY says only that the provider could not be reached, and stamps
	// nothing — see TestAnUnreachableProviderDoesNotAnswerForAnyLocation.
	bad := build()
	for _, k := range []FetchKind{KindObs, KindForecast} {
		bad.Apply(Fragment{Provider: "nws", Kind: k, FetchedAt: at, Err: errFetch,
			PerLocation: map[LocationKey]PartialData{Key(served): {}}}, asked)
	}
	if got := stamp(bad, barren); !got.Equal(at) {
		t.Errorf("a per-location failure alongside a served location is still an answer: %v", got)
	}

	// A SECONDARY cannot answer a question about the reference: its absence is
	// not what makes a row read as loading.
	sec := build()
	for _, k := range []FetchKind{KindObs, KindForecast} {
		sec.Apply(Fragment{Provider: "ndbc", Kind: k, FetchedAt: at,
			PerLocation: map[LocationKey]PartialData{Key(served): {}}}, asked)
	}
	if got := stamp(sec, barren); !got.IsZero() {
		t.Errorf("a secondary stamped the weather attempt: %v", got)
	}
}

var errFetch = errString("the provider did not answer")

type errString string

func (e errString) Error() string { return string(e) }

// AN UNREACHABLE PROVIDER IS NOT AN ANSWER (red team, 2026-09-08).
//
// The stamp's first version fired on any fragment reaching Apply, justified by a
// comment claiming "reaching Apply means the provider responded". That is false:
// FetchEach joins per-location errors into Fragment.Err and returns a NIL error,
// so a connection refused arrives as a fragment carrying an error and serving
// nothing — measured against 127.0.0.1:1, Fetch returns err=nil, frag.Err set,
// PerLocation=0.
//
// The consequence was a regression a listener would see: cold-start with no
// network and every row stopped shimmering and read "n/a" — the product saying
// "we asked and there is nothing for your area" when the truth was "we cannot
// reach the weather service". Before the change those rows kept shimmering.
func TestAnUnreachableProviderDoesNotAnswerForAnyLocation(t *testing.T) {
	a := LocationRef{Label: "A", Lat: 33.2, Lon: -117.38}
	b := LocationRef{Label: "B", Lat: 32.7, Lon: -117.16}
	at := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	asked := []LocationKey{Key(a), Key(b)}

	build := func() *Assembler {
		asm := NewAssembler([]LocationRef{a, b}, []string{"nws"})
		asm.SetAttribution("nws", "reference", "NWS")
		return asm
	}
	stamped := func(asm *Assembler) int {
		n := 0
		for _, l := range asm.Snapshot().Locations {
			if !l.WeatherAsOf.IsZero() {
				n++
			}
		}
		return n
	}

	// THE OUTAGE: an error, and nothing served. Proves nothing about any location.
	out := build()
	for _, k := range []FetchKind{KindObs, KindForecast} {
		out.Apply(Fragment{Provider: "nws", Kind: k, FetchedAt: at, Err: errFetch}, asked)
	}
	if n := stamped(out); n != 0 {
		t.Errorf("an unreachable provider answered for %d location(s); rows must keep shimmering", n)
	}

	// THE PARTIAL: an error, but A was served — so the provider IS reachable and
	// B is a place it has nothing for. That is issue #13 and it must still work.
	part := build()
	for _, k := range []FetchKind{KindObs, KindForecast} {
		part.Apply(Fragment{Provider: "nws", Kind: k, FetchedAt: at, Err: errFetch,
			PerLocation: map[LocationKey]PartialData{Key(a): {}}}, asked)
	}
	if n := stamped(part); n != 2 {
		t.Errorf("a partial answer covers every location asked about; stamped %d of 2", n)
	}
}

// ONLY THE FETCHES THAT ANSWER THE ROW END ITS SHIMMER (red team, 2026-09-08).
//
// The alerts tier is a single GET and starts at the same instant as obs, which
// is three chained GETs, so it lands first. Stamping on it ended the shimmer
// across the whole board a second into every cold start — flashing "n/a" for
// temperatures that were on their way. app/pipelines.go already refused to stamp
// on the supplementary hourly fetch for exactly this reason; the same hazard was
// left standing on the path that matters.
func TestOnlyTheAnsweringFetchesEndTheShimmer(t *testing.T) {
	ref := LocationRef{Label: "A", Lat: 33.2, Lon: -117.38}
	at := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	asked := []LocationKey{Key(ref)}
	served := map[LocationKey]PartialData{Key(ref): {}}

	stamp := func(kinds ...FetchKind) time.Time {
		a := NewAssembler([]LocationRef{ref}, []string{"nws"})
		a.SetAttribution("nws", "reference", "NWS")
		for _, k := range kinds {
			a.Apply(Fragment{Provider: "nws", Kind: k, FetchedAt: at, PerLocation: served}, asked)
		}
		return a.Snapshot().Locations[0].WeatherAsOf
	}

	for _, k := range []FetchKind{KindAlerts, KindFire, KindMarine, KindSeismic, KindForecastHourly} {
		if got := stamp(k); !got.IsZero() {
			t.Errorf("kind %v does not answer the row and must not end its shimmer: %v", k, got)
		}
	}
	// EITHER ALONE IS HALF A ROW: obs without a forecast is a temperature with no
	// high/low, which would read "n/a" while the forecast is still in flight.
	if got := stamp(KindObs); !got.IsZero() {
		t.Errorf("conditions alone leave the forecast outstanding: %v", got)
	}
	if got := stamp(KindForecast); !got.IsZero() {
		t.Errorf("a forecast alone leaves the conditions outstanding: %v", got)
	}
	if got := stamp(KindObs, KindForecast); got.IsZero() {
		t.Error("both answering fetches completed; the row is no longer loading")
	}
}
