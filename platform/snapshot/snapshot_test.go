package snapshot

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Spec: architecture.md §2, §10.1, §10.11 — immutable snapshots published by a
// single assembler; last-write-wins per (provider, location, domain-section);
// failed fragments never overwrite (prior data stands + Warning appended;
// ProviderStatus degrades on Err only — obs_stale never degrades Status);
// every exported field carries a json tag or `json:"-"` (M5 reflection gate).

func loc(label string) LocationRef {
	return LocationRef{Label: label, Lat: 33.24, Lon: -117.29}
}

func f64(v float64) *float64 { return &v }

func TestAssemblerMergesFireFromEveryProvider(t *testing.T) {
	// B5: HMS, WFIGS and FIRMS each contribute a part; the snapshot carries
	// the union — the same fire seen by two feeds once (strongest reading),
	// nearest first; incidents by name, largest first; a provider's later
	// fragment replaces only its own part.
	ref := LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	k := Key(ref)
	a := NewAssembler([]LocationRef{ref}, []string{"hms", "wfigs", "firms"})
	day := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	f := func(v float64) *float64 { return &v }
	a.Apply(Fragment{Provider: "hms", Kind: KindFire, PerLocation: map[LocationKey]PartialData{k: {Fire: &FireState{Hotspots: []Hotspot{
		{Lat: 33.29, Lon: -117.24, DetectedAt: day, FRPMW: f(20), DistanceKm: f(6), Source: SourceInfo{Provider: "hms"}},
		{Lat: 33.10, Lon: -117.10, DetectedAt: day, FRPMW: f(9), DistanceKm: f(22), Source: SourceInfo{Provider: "hms"}},
	}}}}})
	a.Apply(Fragment{Provider: "firms", Kind: KindFire, PerLocation: map[LocationKey]PartialData{k: {Fire: &FireState{Hotspots: []Hotspot{
		{Lat: 33.291, Lon: -117.241, DetectedAt: day.Add(time.Hour), FRPMW: f(61.5), DistanceKm: f(6), Source: SourceInfo{Provider: "firms"}}, // the same fire, stronger
	}}}}})
	a.Apply(Fragment{Provider: "wfigs", Kind: KindFire, PerLocation: map[LocationKey]PartialData{k: {Fire: &FireState{Incidents: []Incident{
		{Name: "Timber", Acres: f(12915), Source: SourceInfo{Provider: "wfigs", IssuedAt: day}},
		{Name: "LAC-1", Source: SourceInfo{Provider: "wfigs", IssuedAt: day}},
	}}}}})
	fs := a.Snapshot().Locations[0].Fire
	if len(fs.Hotspots) != 2 || *fs.Hotspots[0].FRPMW != 61.5 || fs.Hotspots[0].Source.Provider != "firms" || *fs.Hotspots[1].DistanceKm != 22 {
		t.Fatalf("union, deduped to the strongest reading, nearest first: %+v", fs.Hotspots)
	}
	if len(fs.Incidents) != 2 || fs.Incidents[0].Name != "Timber" {
		t.Fatalf("incidents largest first: %+v", fs.Incidents)
	}
	// HMS reports nothing next cycle: only its part goes; FIRMS's stays.
	a.Apply(Fragment{Provider: "hms", Kind: KindFire, PerLocation: map[LocationKey]PartialData{k: {Fire: &FireState{}}}})
	fs = a.Snapshot().Locations[0].Fire
	if len(fs.Hotspots) != 1 || fs.Hotspots[0].Source.Provider != "firms" || len(fs.Incidents) != 2 {
		t.Fatalf("a provider replaces only its own contribution: %+v", fs)
	}
}

func TestAssemblerDedupesLocationsAndBoundsWarnings(t *testing.T) {
	// Red-team 0.9.0 F4: a duplicate ref (a re-run setup, a config edit)
	// used to leave order and refs misaligned and publish an EMPTY snapshot
	// forever. F6: warnings are bounded to the newest maxWarnings.
	twice := LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	a := NewAssembler([]LocationRef{twice, twice, {Label: "Carlsbad, CA", Lat: 33.16, Lon: -117.35}}, []string{"nws"})
	if s := a.Snapshot(); len(s.Locations) != 2 || s.Locations[0].Label != "Oceanside, CA" || s.Locations[1].Label != "Carlsbad, CA" {
		t.Fatalf("duplicates collapse to the first, order kept: %+v", s.Locations)
	}
	for i := 0; i < maxWarnings+50; i++ { // bounded
		a.Warn(Warning{Code: WarnProviderError, Provider: "nws", Message: "down"})
	}
	if n := len(a.Snapshot().Warnings); n != maxWarnings {
		t.Fatalf("warnings bounded to %d, got %d", maxWarnings, n)
	}
}

func TestAssemblerMergesFragmentsLastWriteWins(t *testing.T) {
	a := NewAssembler([]LocationRef{loc("Oceanside, CA")}, []string{"nws"})
	k := Key(loc("Oceanside, CA"))
	t1 := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)

	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1, PerLocation: map[LocationKey]PartialData{
		k: {Current: &Conditions{Temp: f64(20.0), ObservedAt: t1}},
	}})
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1.Add(time.Minute), PerLocation: map[LocationKey]PartialData{
		k: {Current: &Conditions{Temp: f64(21.5), ObservedAt: t1.Add(time.Minute)}},
	}})

	snap := a.Snapshot()
	if got := snap.Locations[0].ByProvider["nws"].Current.Temp; got == nil || *got != 21.5 {
		t.Fatalf("last write must win, got %v", got)
	}
}

func TestFailedFragmentNeverOverwrites(t *testing.T) {
	a := NewAssembler([]LocationRef{loc("A")}, []string{"nws"})
	k := Key(loc("A"))
	t1 := time.Now().UTC()
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1, PerLocation: map[LocationKey]PartialData{
		k: {Current: &Conditions{Temp: f64(20)}},
	}})
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1.Add(time.Minute), Err: assertErr("boom")})

	snap := a.Snapshot()
	if got := snap.Locations[0].ByProvider["nws"].Current.Temp; got == nil || *got != 20 {
		t.Fatal("failed fragment must not clobber prior data")
	}
	if len(snap.Warnings) == 0 || snap.Warnings[0].Code != WarnProviderError {
		t.Fatalf("failed fragment must append a provider_error warning: %+v", snap.Warnings)
	}
	if st := statusOf(snap, "nws"); st != ProviderDegraded {
		t.Fatalf("provider must degrade on Err, got %s", st)
	}
}

func TestObsStaleWarningDoesNotDegradeStatus(t *testing.T) {
	a := NewAssembler([]LocationRef{loc("A")}, []string{"nws"})
	a.Warn(Warning{Code: WarnObsStale, Message: "KDCA obs 72m old", Location: "A", Provider: "nws"})
	snap := a.Snapshot()
	if st := statusOf(snap, "nws"); st != ProviderOK {
		t.Fatalf("obs_stale must never degrade ProviderStatus (§10.11), got %s", st)
	}
}

func TestSnapshotsAreImmutablePublications(t *testing.T) {
	a := NewAssembler([]LocationRef{loc("A")}, []string{"nws"})
	k := Key(loc("A"))
	s1 := a.Snapshot()
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		k: {Current: &Conditions{Temp: f64(30)}},
	}})
	if s1.Locations[0].ByProvider["nws"].Current != nil {
		t.Fatal("earlier snapshot must not see later writes (immutability)")
	}
}

func TestConcurrentApplyAndReadIsRaceFree(t *testing.T) {
	a := NewAssembler([]LocationRef{loc("A")}, []string{"nws"})
	k := Key(loc("A"))
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			v := float64(n)
			a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
				k: {Current: &Conditions{Temp: &v}},
			}})
		}(i)
		go func() {
			defer wg.Done()
			_ = a.Snapshot().Locations[0].Label
		}()
	}
	wg.Wait()
}

func TestEveryExportedFieldHasJSONTag(t *testing.T) {
	// M5 reflection gate (architecture §2): every exported field of every
	// contract type carries a json tag (or explicit "-").
	types := []any{Snapshot{}, Location{}, Conditions{}, SourceInfo{}, Hourly{}, Daily{},
		Alert{}, FireState{}, Hotspot{}, Incident{}, RadioState{}, ProviderStatus{}, Warning{}}
	for _, v := range types {
		rt := reflect.TypeOf(v)
		for i := range rt.NumField() {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			if _, ok := f.Tag.Lookup("json"); !ok {
				t.Errorf("%s.%s has no json tag", rt.Name(), f.Name)
			}
		}
	}
}

func TestNilFieldsMarshalNull(t *testing.T) {
	// §10.11 null-parity rule: nil pointer -> JSON null (renders n/a).
	b, err := json.Marshal(Conditions{ObservedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"temp":null`) {
		t.Fatalf("nil Temp must marshal as null: %s", b)
	}
}

func TestKeyIsStable(t *testing.T) {
	a := Key(LocationRef{Label: "X", Lat: 33.2405, Lon: -117.2912})
	b := Key(LocationRef{Label: "Y", Lat: 33.24051, Lon: -117.29119})
	if a != b {
		t.Fatalf("keys must normalize to 4dp: %s vs %s", a, b)
	}
}

// helpers
func statusOf(s *Snapshot, id string) string {
	for _, p := range s.Providers {
		if p.ID == id {
			return p.Status
		}
	}
	return ""
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func TestFailedFragmentStillAppliesPartialData(t *testing.T) {
	// UAT 59: one bad location must not blank the others — a Fragment that
	// carries Err AND PerLocation merges what it has (Err degrades + warns).
	b := LocationRef{Label: "B", Lat: 32.72, Lon: -117.16}
	a := NewAssembler([]LocationRef{loc("A"), b}, []string{"nws"})
	ka, kb := Key(loc("A")), Key(b)
	t1 := time.Now().UTC()
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1, PerLocation: map[LocationKey]PartialData{
		kb: {Current: &Conditions{Temp: f64(15)}},
	}})
	a.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: t1.Add(time.Minute), Err: assertErr("B: 404"),
		PerLocation: map[LocationKey]PartialData{ka: {Current: &Conditions{Temp: f64(20)}}}})
	snap := a.Snapshot()
	if got := snap.Locations[0].ByProvider["nws"].Current; got == nil || *got.Temp != 20 {
		t.Fatal("the successful location in a partially failed fragment must land")
	}
	if got := snap.Locations[1].ByProvider["nws"].Current; got == nil || *got.Temp != 15 {
		t.Fatal("the failed location keeps its prior data")
	}
	if st := statusOf(snap, "nws"); st != ProviderDegraded {
		t.Fatalf("partial failure still degrades the provider, got %s", st)
	}
}

func TestSetLocationsKeepsDataForKeptLocations(t *testing.T) {
	// UAT 69: reconciling the tracked set never blanks what is already
	// loaded — a lookup prepends one new (empty) location, drops the oldest,
	// and every kept row keeps its sections in the new order.
	a, b, c := loc("A"), LocationRef{Label: "B", Lat: 32.72, Lon: -117.16}, LocationRef{Label: "C", Lat: 34.05, Lon: -118.24}
	asm := NewAssembler([]LocationRef{a, b}, []string{"nws"})
	asm.Apply(Fragment{Provider: "nws", Kind: KindObs, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{
		Key(a): {Current: &Conditions{Temp: f64(20)}}, Key(b): {Current: &Conditions{Temp: f64(15)}},
	}})
	added, removed := asm.SetLocations([]LocationRef{c, a}) // C on top, B dropped
	if len(added) != 1 || added[0].Label != "C" || len(removed) != 1 || removed[0].Label != "B" {
		t.Fatalf("added %v removed %v", added, removed)
	}
	s := asm.Snapshot()
	if len(s.Locations) != 2 || s.Locations[0].Label != "C" || s.Locations[1].Label != "A" {
		t.Fatalf("order must follow refs: %v", s.Locations)
	}
	if got := s.Locations[1].ByProvider["nws"].Current; got == nil || *got.Temp != 20 {
		t.Fatal("kept location must keep its data")
	}
	if s.Locations[0].Harmonized.Source.Provider != "" {
		t.Fatal("new location starts empty (loading)")
	}
	if added, removed := asm.SetLocations([]LocationRef{a, c}); len(added) != 0 || len(removed) != 0 {
		t.Fatalf("a pure reorder adds and removes nothing: %v %v", added, removed)
	}
	if asm.Snapshot().Locations[0].Label != "A" {
		t.Fatal("reorder must apply")
	}
}

func TestSetInactivePublishesOffAndNeverExit2(t *testing.T) {
	// UAT 100: a registered provider that is not a source right now (FIRMS
	// without a key) reads "off" whatever its fragments say; unknown ids
	// are refused.
	a := NewAssembler([]LocationRef{{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.3}}, []string{"nws", "firms"})
	if err := a.SetInactive("nope", true); err == nil {
		t.Fatal("unregistered id must fail the invariant")
	}
	if err := a.SetInactive("firms", true); err != nil {
		t.Fatal(err)
	}
	status := func() map[string]string {
		out := map[string]string{}
		for _, p := range a.Snapshot().Providers {
			out[p.ID] = p.Status
		}
		return out
	}
	if st := status(); st["firms"] != ProviderOff || st["nws"] != ProviderOK {
		t.Fatalf("firms off, nws ok: %v", st)
	}
	if err := a.SetInactive("firms", false); err != nil {
		t.Fatal(err)
	}
	if st := status(); st["firms"] != ProviderOK {
		t.Fatalf("keyed again: back to ok, got %v", st)
	}
}

func TestFireForAndProviderStatusReadNarrowly(t *testing.T) {
	// REVIEW C2: the radio deck reads one location's fire state and a
	// provider's status without cloning the snapshot.
	ref := LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2, Lon: -117.3}
	a := NewAssembler([]LocationRef{ref}, []string{"hms", "firms"})
	if _, _, _, ok := a.FireFor(LocationRef{Zip: "00000"}); ok {
		t.Fatal("untracked ref: not found")
	}
	fs, lat, lon, ok := a.FireFor(ref)
	if !ok || !fs.AsOf.IsZero() || lat != 33.2 || lon != -117.3 {
		t.Fatalf("tracked, no feed yet: %v %+v %v %v", ok, fs, lat, lon)
	}
	at := time.Now().UTC()
	a.Apply(Fragment{Provider: "hms", Kind: KindFire, FetchedAt: at, PerLocation: map[LocationKey]PartialData{
		Key(ref): {Fire: &FireState{AsOf: at, Hotspots: []Hotspot{{Lat: 33.3, Lon: -117.3, DistanceKm: f64(11)}}}}}})
	if fs, _, _, _ := a.FireFor(ref); fs.AsOf.IsZero() || len(fs.Hotspots) != 1 {
		t.Fatalf("after a fire fragment: %+v", fs)
	}
	if a.ProviderStatus("firms") != ProviderOK || a.ProviderStatus("nope") != "" {
		t.Fatal("provider status")
	}
	_ = a.SetInactive("firms", true)
	if a.ProviderStatus("firms") != ProviderOff {
		t.Fatal("inactive reads off")
	}
}

// The seismic seam (0.11.0): a provider fragment's SeismicState reaches the
// published Location, a later fragment replaces it, and the snapshot is a deep
// copy (mutating the published value cannot reach back into the assembler).
func TestAssemblerMergesSeismicAndIsolates(t *testing.T) {
	ref := LocationRef{Label: "Ridgecrest, CA", Lat: 35.62, Lon: -117.67}
	k := Key(ref)
	a := NewAssembler([]LocationRef{ref}, []string{"usgs"})
	now := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	felt := 42
	a.Apply(Fragment{Provider: "usgs", Kind: KindSeismic, PerLocation: map[LocationKey]PartialData{k: {Seismic: &SeismicState{
		AsOf: now, Quakes: []Quake{{Mag: 4.0, Place: "Coso", DistanceKm: 52, Bearing: "NNW", Felt: &felt}},
	}}}})
	ss := a.Snapshot().Locations[0].Seismic
	if ss == nil || ss.AsOf != now || len(ss.Quakes) != 1 || ss.Quakes[0].Mag != 4.0 || *ss.Quakes[0].Felt != 42 {
		t.Fatalf("the seismic fragment must reach the published location: %+v", ss)
	}
	// Deep copy: mutating the published value leaves the next snapshot intact.
	ss.Quakes[0].Mag = 9.9
	*ss.Quakes[0].Felt = 0
	if again := a.Snapshot().Locations[0].Seismic; again.Quakes[0].Mag != 4.0 || *again.Quakes[0].Felt != 42 {
		t.Fatalf("the snapshot must not alias assembler state: %+v", again.Quakes[0])
	}
	// A later fragment replaces the state (one provider, latest wins).
	a.Apply(Fragment{Provider: "usgs", Kind: KindSeismic, PerLocation: map[LocationKey]PartialData{k: {Seismic: &SeismicState{AsOf: now.Add(time.Hour)}}}})
	if ss := a.Snapshot().Locations[0].Seismic; ss == nil || len(ss.Quakes) != 0 {
		t.Fatalf("a later fragment replaces the state (quiet now): %+v", ss)
	}
}
