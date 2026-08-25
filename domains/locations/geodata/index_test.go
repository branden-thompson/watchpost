package geodata

import (
	"sync"
	"testing"
	"time"
)

// Spec: S2 spike (compact representation b — backing []byte + sorted offset
// index, lazy row parse, prefix lookups <10ms with 7µs measured) + AI-8 rules
// (representative zip: query-matching zip wins, else lowest-numbered for the
// place; population ranks hints) + D-19 (labels "City, ST (zip)").

var (
	sharedIdx  *Index
	sharedErr  error
	sharedOnce sync.Once
)

// idx loads the index once for the whole package (Load is ~70ms; the index is
// read-only after Load, so sharing is race-safe — asserted by -race runs).
func idx(t *testing.T) *Index {
	t.Helper()
	sharedOnce.Do(func() { sharedIdx, sharedErr = Load() })
	if sharedErr != nil {
		t.Fatal(sharedErr)
	}
	return sharedIdx
}

func TestLoadCounts(t *testing.T) {
	i := idx(t)
	// Dataset facts from the S2 spike (2026-08-23 GeoNames snapshot).
	if i.Cities() < 30000 || i.Cities() > 40000 {
		t.Fatalf("cities = %d, want ~34k", i.Cities())
	}
	if i.Zips() < 35000 || i.Zips() > 50000 {
		t.Fatalf("zips = %d, want ~41k", i.Zips())
	}
}

func TestPrefixSearchRanksByPopulation(t *testing.T) {
	i := idx(t)
	hits := i.PrefixSearch("spring", 5)
	if len(hits) == 0 {
		t.Fatal("no hits for 'spring'")
	}
	for n := 1; n < len(hits); n++ {
		if hits[n].Population > hits[n-1].Population {
			t.Fatalf("hits must rank by population desc: %v", hits)
		}
	}
	// Case-insensitive.
	if len(i.PrefixSearch("SPRING", 5)) != len(hits) {
		t.Fatal("prefix search must be case-insensitive")
	}
}

func TestPrefixSearchLatencyBudget(t *testing.T) {
	i := idx(t)
	start := time.Now()
	const n = 200
	for range n {
		_ = i.PrefixSearch("san", 8)
	}
	per := time.Since(start) / n
	if per > 10*time.Millisecond {
		t.Fatalf("type-ahead budget blown: %v per lookup (want <10ms)", per)
	}
}

func TestZipLookup(t *testing.T) {
	i := idx(t)
	z, ok := i.Zip("92057")
	if !ok {
		t.Fatal("92057 must resolve")
	}
	if z.Place == "" || z.State != "CA" || z.Lat == 0 {
		t.Fatalf("zip row: %+v", z)
	}
}

func TestRepresentativeZip(t *testing.T) {
	i := idx(t)
	hits := i.PrefixSearch("oceanside", 10)
	var oceanside *City
	for n := range hits {
		if hits[n].State == "CA" {
			oceanside = &hits[n]
			break
		}
	}
	if oceanside == nil {
		t.Fatal("Oceanside, CA not found")
	}
	zip := i.RepresentativeZip(*oceanside)
	if zip == "" {
		t.Fatalf("no representative zip for %+v", oceanside)
	}
	z, _ := i.Zip(zip)
	if z.State != "CA" {
		t.Fatalf("representative zip %s resolves to %s, want CA", zip, z.State)
	}
}

func TestCityLabel(t *testing.T) {
	i := idx(t)
	hits := i.PrefixSearch("oceanside", 10)
	for _, h := range hits {
		if h.Country == "US" && h.State == "CA" {
			if got := h.Label(); got != "Oceanside, CA" {
				t.Fatalf("label = %q", got)
			}
			return
		}
	}
	t.Fatal("no US Oceanside hit")
}

func TestNoMatch(t *testing.T) {
	i := idx(t)
	if hits := i.PrefixSearch("zzzxqv", 5); len(hits) != 0 {
		t.Fatalf("impossible prefix returned %d hits", len(hits))
	}
	if _, ok := i.Zip("00000"); ok {
		t.Fatal("00000 must not resolve")
	}
}
