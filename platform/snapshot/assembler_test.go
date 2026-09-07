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
	}})
	a.Apply(Fragment{Provider: "ndbc", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{WaveHeight: f(0.9), WavePeriod: f(14), WaterTemp: f(23.3), Buoy: "46224"}},
	}})
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
	}})
	a.Apply(Fragment{Provider: "coops", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(ref): {Marine: &Marine{TideLevel: f(1.13), Tides: []TideEvent{{Time: when, Height: 1.73, Type: "H"}},
			TideStation: "San Diego", TideStationKM: f(1.2), Currents: []CurrentEvent{{Time: when, Speed: 0.5, Type: "flood"}}, CurrentStation: "San Diego Bay Entrance"}},
	}})
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
