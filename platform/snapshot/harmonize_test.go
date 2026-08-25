package snapshot

import (
	"testing"
	"time"
)

// B1 red-team #2: harmonize/fillFrom shipped without direct tests. These pin
// the OQ-9 rule: NWS wins outright; secondaries fill nil fields only, with
// fill_from provenance; no blending; nws-first order regardless of config
// position (probe-verified order [a,nws,b] -> [nws,a,b]).

func apply(a *Assembler, prov string, k LocationKey, c *Conditions) {
	a.Apply(Fragment{Provider: prov, Kind: KindObs, FetchedAt: time.Now(),
		PerLocation: map[LocationKey]PartialData{k: {Current: c}}})
}

func TestHarmonizeNWSWinsOutright(t *testing.T) {
	ref := LocationRef{Label: "A", Lat: 1, Lon: 2}
	a := NewAssembler([]LocationRef{ref}, []string{"open-meteo", "nws"})
	k := Key(ref)
	apply(a, "open-meteo", k, &Conditions{Temp: f64(30.0), UVIndex: f64(7), Source: SourceInfo{Provider: "open-meteo"}})
	apply(a, "nws", k, &Conditions{Temp: f64(22.8), Source: SourceInfo{Provider: "nws", ModelOrStation: "KOKB"}})

	h := a.Snapshot().Locations[0].Harmonized
	if h.Source.Provider != "nws" {
		t.Fatalf("harmonized source = %s, want nws (tie-break authority)", h.Source.Provider)
	}
	if *h.Temp != 22.8 {
		t.Fatalf("temp = %v — NWS value must win outright, never blend", *h.Temp)
	}
	if h.UVIndex == nil || *h.UVIndex != 7 {
		t.Fatal("nil NWS UVIndex must be filled from open-meteo")
	}
	if h.Source.FillFrom["uv_index"] != "open-meteo" {
		t.Fatalf("fill_from provenance missing: %+v", h.Source.FillFrom)
	}
	if _, filled := h.Source.FillFrom["temp"]; filled {
		t.Fatal("temp had an NWS value — must never appear in fill_from")
	}
}

func TestHarmonizeSecondaryOnlyWhenNWSAbsent(t *testing.T) {
	ref := LocationRef{Label: "B", Lat: 3, Lon: 4}
	a := NewAssembler([]LocationRef{ref}, []string{"nws", "open-meteo"})
	k := Key(ref)
	apply(a, "open-meteo", k, &Conditions{Temp: f64(18.1), Source: SourceInfo{Provider: "open-meteo"}})
	h := a.Snapshot().Locations[0].Harmonized
	if h.Source.Provider != "open-meteo" || *h.Temp != 18.1 {
		t.Fatalf("with no NWS data the secondary must seed harmonized: %+v", h.Source)
	}
}

func TestHarmonizeSeriesNeverSplice(t *testing.T) {
	ref := LocationRef{Label: "C", Lat: 5, Lon: 6}
	a := NewAssembler([]LocationRef{ref}, []string{"nws", "open-meteo"})
	k := Key(ref)
	a.Apply(Fragment{Provider: "open-meteo", Kind: KindForecast, FetchedAt: time.Now(),
		PerLocation: map[LocationKey]PartialData{k: {Hourly: []Hourly{{Temp: f64(9.9)}, {Temp: f64(9.8)}}}}})
	a.Apply(Fragment{Provider: "nws", Kind: KindForecast, FetchedAt: time.Now(),
		PerLocation: map[LocationKey]PartialData{k: {Hourly: []Hourly{{Temp: f64(1.1)}}}}})
	loc := a.Snapshot().Locations[0]
	if len(loc.Hourly) != 1 || *loc.Hourly[0].Temp != 1.1 {
		t.Fatalf("hourly must come wholesale from nws (first in order), never spliced: %+v", loc.Hourly)
	}
}

func TestRehydrateSparseObsFromForecast(t *testing.T) {
	// UAT 59 (Carlsbad): a mesonet station reports no sky condition and a
	// temperature only every few observations. The hour's forecast fills
	// those holes, with provenance, so the row never reads "UNKNOWN n/a".
	now := time.Date(2026, 8, 24, 21, 30, 0, 0, time.UTC)
	l := &Location{
		Harmonized: Conditions{Condition: "unknown", Source: SourceInfo{Provider: "nws", ModelOrStation: "CBDSD"}},
		Hourly: []Hourly{
			{Time: now.Add(-90 * time.Minute), Temp: f64(20), Condition: "cloudy"},
			{Time: now.Add(-30 * time.Minute), Temp: f64(24), Condition: "clear"},
			{Time: now.Add(30 * time.Minute), Temp: f64(26), Condition: "rain"},
		},
	}
	rehydrateFromForecast(l, now)
	if l.Harmonized.Temp == nil || *l.Harmonized.Temp != 24 || l.Harmonized.Condition != "clear" {
		t.Fatalf("expected the covering hour (24 / clear), got %+v", l.Harmonized)
	}
	if l.Harmonized.Source.FillFrom["temp"] != fillForecast || l.Harmonized.Source.FillFrom["condition_code"] != fillForecast {
		t.Fatalf("fills must record forecast provenance: %v", l.Harmonized.Source.FillFrom)
	}
	if l.Harmonized.Source.ModelOrStation != "CBDSD" {
		t.Fatal("the observing station stays the source of record")
	}

	// Observed values are never replaced; a stale forecast never fills.
	obs := &Location{Harmonized: Conditions{Temp: f64(18), Condition: "fog", Source: SourceInfo{Provider: "nws"}},
		Hourly: []Hourly{{Time: now.Add(-30 * time.Minute), Temp: f64(24), Condition: "clear"}}}
	rehydrateFromForecast(obs, now)
	if *obs.Harmonized.Temp != 18 || obs.Harmonized.Condition != "fog" || obs.Harmonized.Source.FillFrom != nil {
		t.Fatalf("observed values must stand: %+v", obs.Harmonized)
	}
	stale := &Location{Harmonized: Conditions{Source: SourceInfo{Provider: "nws"}},
		Hourly: []Hourly{{Time: now.Add(-3 * time.Hour), Temp: f64(24), Condition: "clear"}}}
	rehydrateFromForecast(stale, now)
	if stale.Harmonized.Temp != nil {
		t.Fatal("a forecast hour that does not cover now must not fill")
	}
	none := &Location{Hourly: []Hourly{{Time: now, Temp: f64(24)}}}
	rehydrateFromForecast(none, now)
	if none.Harmonized.Temp != nil {
		t.Fatal("no observation at all is a loading state, not a rehydration case")
	}
}
