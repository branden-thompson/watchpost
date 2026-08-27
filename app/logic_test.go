package app

// Quality pass Q2 (L3-F22: app's pure logic was untested — 16 % at
// DISCOVER). Tables for the config ↔ ref conversion, the RECENT restore,
// the coordinate parser, the M1 predicate, the keymap layer, the stale
// warning and the deck's labels.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestParseLatLonTable(t *testing.T) {
	for in, want := range map[string][3]any{
		"33.24,-117.29":    {33.24, -117.29, true},
		" 33.24 , -117.29": {33.24, -117.29, true},
		"91,0":             {0.0, 0.0, false}, // latitude out of range
		"0,181":            {0.0, 0.0, false},
		"33.24":            {0.0, 0.0, false},
		"a,b":              {0.0, 0.0, false},
		"1,2,3":            {0.0, 0.0, false},
	} {
		lat, lon, ok := parseLatLon(in)
		if ok != want[2].(bool) || (ok && (lat != want[0].(float64) || lon != want[1].(float64))) {
			t.Errorf("parseLatLon(%q) = %v %v %v, want %v", in, lat, lon, ok, want)
		}
	}
}

func TestRefsRoundTripDerivesMissingTags(t *testing.T) {
	locs := []config.Location{{Label: "Oceanside, CA", Tag: "OSIDE", Zip: "92057", Lat: 33.24, Lon: -117.29, TZ: "America/Los_Angeles"}, {Label: "Boise, ID", Zip: "83702"}}
	refs := refsFromConfig(locs)
	if len(refs) != 2 || refs[0].Tag != "OSIDE" || refs[0].TZ != "America/Los_Angeles" || refs[1].Tag != "" {
		t.Fatalf("refsFromConfig copies every field and derives nothing: %+v", refs)
	}
	back := configLocations(refs)
	if back[0] != locs[0] {
		t.Fatalf("a tagged location round-trips unchanged: %+v", back[0])
	}
	if back[1].Tag == "" || back[1].Label != "Boise, ID" {
		t.Fatalf("an untagged location gains a derived tag on the way back: %+v", back[1])
	}
}

func TestRestoreRecentSavedFirstThenSeedsDedupedAndCapped(t *testing.T) {
	ref := func(zip string) snapshot.LocationRef { return snapshot.LocationRef{Label: zip, Zip: zip} }
	watch := []snapshot.LocationRef{ref("1")}
	saved := []snapshot.LocationRef{ref("2"), ref("1"), ref("3")} // 1 is a favourite now: dropped
	seeds := []snapshot.LocationRef{ref("3"), ref("4"), ref("5"), ref("6")}
	got := restoreRecent(saved, watch, seeds, 4)
	var zips []string
	for _, r := range got {
		zips = append(zips, r.Zip)
	}
	if strings.Join(zips, ",") != "2,3,4,5" {
		t.Fatalf("saved first (minus favourites), seeds fill, deduped by zip, capped: got %v", zips)
	}
	if got := restoreRecent(nil, nil, nil, 3); len(got) != 0 {
		t.Fatal("nothing saved and no seeds is an empty list")
	}
}

func TestFullyPopulatedNeedsEveryLocationsConditions(t *testing.T) {
	s := &snapshot.Snapshot{}
	if fullyPopulated(s) {
		t.Fatal("no locations is not populated")
	}
	s.Locations = []snapshot.Location{{Harmonized: snapshot.Conditions{Source: snapshot.SourceInfo{Provider: "nws"}}}, {}}
	if fullyPopulated(s) {
		t.Fatal("one location without conditions is not populated")
	}
	s.Locations[1].Harmonized.Source.Provider = "nws"
	if !fullyPopulated(s) {
		t.Fatal("every location with a provider is populated")
	}
}

func TestToKeyMapLayer(t *testing.T) {
	if toKeyMap(nil) != nil {
		t.Fatal("no [keys] table means no override layer")
	}
	km := toKeyMap(map[string][]string{"quit": {"x", "ctrl+c"}})
	if b, ok := km["quit"]; !ok || strings.Join(b.Keys, ",") != "x,ctrl+c" {
		t.Fatalf("each action maps to its bindings: %+v", km)
	}
}

func TestStaleWarningsFlagObservationsOlderThanTwoHours(t *testing.T) {
	ref := snapshot.LocationRef{Label: "Old", Zip: "1"}
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{"nws"})
	old := time.Now().Add(-3 * time.Hour)
	temp := 20.0
	asm.Apply(snapshot.Fragment{Provider: "nws", Kind: snapshot.KindObs, FetchedAt: time.Now(),
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{snapshot.Key(ref): {Current: &snapshot.Conditions{Temp: &temp, ObservedAt: old, Source: snapshot.SourceInfo{Provider: "nws"}}}}})
	staleWarnings(asm, asm.Snapshot())
	warns := asm.Snapshot().Warnings
	if len(warns) != 1 || warns[0].Code != snapshot.WarnObsStale || !strings.Contains(warns[0].Message, "Old") {
		t.Fatalf("a 3-hour-old observation is one obs_stale warning naming the location: %+v", warns)
	}
	fresh := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{"nws"})
	fresh.Apply(snapshot.Fragment{Provider: "nws", Kind: snapshot.KindObs, FetchedAt: time.Now(),
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{snapshot.Key(ref): {Current: &snapshot.Conditions{Temp: &temp, ObservedAt: time.Now(), Source: snapshot.SourceInfo{Provider: "nws"}}}}})
	staleWarnings(fresh, fresh.Snapshot())
	if len(fresh.Snapshot().Warnings) != 0 {
		t.Fatal("a fresh observation warns of nothing")
	}
	staleWarnings(nil, nil) // the guard, never a panic
}

func TestDeckLabelNamesTheTransmitterAndItsReach(t *testing.T) {
	d := &radioDeck{units: render.UnitF}
	st := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC62", Site: "San Diego", State: "CA", FreqMHz: "162.400"}, KM: 40, Covering: true}
	if got := d.label(st); !strings.HasPrefix(got, "KEC62 San Diego CA 162.400 MHz · ") || !strings.HasSuffix(got, " mi") {
		t.Fatalf("the label carries callsign, site, state, frequency and distance in the display units: %q", got)
	}
	st.Covering = false
	if got := d.label(st); !strings.HasSuffix(got, "(nearest relayed)") {
		t.Fatalf("a non-covering transmitter says so: %q", got)
	}
	d.units = render.UnitC
	if got := d.label(st); !strings.Contains(got, "40 km") {
		t.Fatalf("metric units show kilometres: %q", got)
	}
}
