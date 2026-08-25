package snapshot

import (
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
