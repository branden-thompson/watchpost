// Package geodata is the embedded offline geocoding index — the S2 spike's
// measured "compact" representation: two go:embed gzipped TSVs decompressed
// once at Load into single backing byte slices with sorted offset indexes;
// rows parse lazily on access. Measured (B2 exit probes): Load ~50ms on
// pre-sorted data; warm prefix lookups µs-class, single cold call sub-ms;
// ~13 MB RSS; +1.3 MB binary. Sources: the S2 spike
// (06_docs/02_features/watchpost-cli/03-architecture-design/spikes/S2-geodata-memory.md)
// + b2-build-log. Refresh pipeline: tools/geotrim/refresh.sh (updates the
// SHA-256 pins in checksums_test.go — same commit).
//
// Data: GeoNames cities15000 + US postal codes (CC-BY 4.0 — attribution in
// the About view per OQ-15). Snapshot date 2026-08-23; refresh by re-running
// the S2 trim pipeline (documented in the spike).
package geodata

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

//go:embed data/cities_trim.tsv.gz
var citiesGz []byte

//go:embed data/zips_trim.tsv.gz
var zipsGz []byte

// City is one populated place (lazily parsed row).
type City struct {
	Name       string
	ASCII      string
	State      string // admin1 code (US: 2-letter state)
	Country    string
	Lat, Lon   float64
	Population int
	TZ         string
}

// Label renders per D-19: "City, ST" (US) / "City, CC" (non-US).
func (c City) Label() string {
	switch {
	case c.Country == "US" && c.State != "":
		return c.Name + ", " + c.State
	case c.Country != "":
		return c.Name + ", " + c.Country
	default:
		return c.Name
	}
}

// ZipRow is one US postal code centroid.
type ZipRow struct {
	Zip, Place, State string
	Lat, Lon          float64
}

// Index holds the decompressed data and its lookup structures.
type Index struct {
	cities     []byte  // backing TSV
	cityOffs   []int32 // line offsets, sorted by lowercased ASCII name
	zips       []byte
	zipOffs    []int32 // line offsets, sorted by zip
	zipByPlace map[string][]int32
}

// Load decompresses and indexes the embedded data (call once at startup).
func Load() (*Index, error) {
	cities, err := gunzip(citiesGz)
	if err != nil {
		return nil, fmt.Errorf("embedded city data is corrupt: %w — reinstall watchpost", err)
	}
	zips, err := gunzip(zipsGz)
	if err != nil {
		return nil, fmt.Errorf("embedded zip data is corrupt: %w — reinstall watchpost", err)
	}
	idx := &Index{cities: cities, zips: zips, zipByPlace: map[string][]int32{}}
	idx.cityOffs = lineOffsets(cities)
	if err := invariant.Check(len(idx.cityOffs) > 10000, "embedded city index implausibly small"); err != nil {
		return nil, err
	}
	idx.zipOffs = lineOffsets(zips)
	if err := invariant.Check(len(idx.zipOffs) > 10000, "embedded zip index implausibly small"); err != nil {
		return nil, err
	}
	// Data is PRE-SORTED at build time (cities by lowercased ASCII name, zips
	// by code — B2 red-team #5 deleted the runtime O(n log n × parse) sorts).
	// Fail closed if a regenerated payload forgot the sort: an unsorted index
	// silently breaks every binary search.
	if err := invariant.Check(sortedBy(idx.cityOffs, idx.cityKey), "embedded city data is not sorted — regenerate with the sorted trim pipeline"); err != nil {
		return nil, err
	}
	if err := invariant.Check(sortedBy(idx.zipOffs, func(off int32) string { return zipField(zips, off) }), "embedded zip data is not sorted — regenerate with the sorted trim pipeline"); err != nil {
		return nil, err
	}
	for _, off := range idx.zipOffs {
		z := idx.parseZip(off)
		key := strings.ToLower(z.Place) + "|" + z.State
		idx.zipByPlace[key] = append(idx.zipByPlace[key], off)
	}
	return idx, nil
}

func gunzip(gz []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// sortedBy reports whether offsets are ordered by key (O(n) spot walk).
func sortedBy(offs []int32, key func(int32) string) bool {
	for n := 1; n < len(offs); n++ {
		if key(offs[n]) < key(offs[n-1]) {
			return false
		}
	}
	return true
}

// lineOffsets records the start of every non-empty line.
func lineOffsets(b []byte) []int32 {
	if invariant.Check(len(b) < (1<<31)-1, "embedded dataset exceeds int32 offsets — widen the index type") != nil {
		return nil // fails the plausibility check in Load (red-team #6)
	}
	offs := make([]int32, 0, 40000)
	start := int32(0)
	for i, c := range b {
		if c == '\n' {
			if int32(i) > start {
				offs = append(offs, start)
			}
			start = int32(i) + 1
		}
	}
	if int(start) < len(b) {
		offs = append(offs, start)
	}
	return offs
}

// cityKey returns the lowercased ASCII name (column 2) at a line offset.
func (i *Index) cityKey(off int32) string {
	return strings.ToLower(field(i.cities, off, 1))
}

// field extracts TSV column n (0-based) from the line at off, allocation-free
// until the final string cast.
func field(b []byte, off int32, n int) string {
	col := 0
	start := int(off)
	i := start
	// Bounded counter form (P10-02): i walks to len(b) at most.
	for ; i < len(b); i++ {
		if b[i] == '\n' {
			break
		}
		if b[i] == '\t' {
			if col == n {
				return string(b[start:i])
			}
			col++
			start = i + 1
		}
	}
	if col == n {
		return string(b[start:i])
	}
	return ""
}

// parseCity parses the full row at a line offset.
// Columns: name, ascii, admin1, country, lat, lon, population, tz.
func (i *Index) parseCity(off int32) City {
	f := func(n int) string { return field(i.cities, off, n) }
	lat, _ := strconv.ParseFloat(f(4), 64)
	lon, _ := strconv.ParseFloat(f(5), 64)
	pop, _ := strconv.Atoi(f(6))
	return City{Name: f(0), ASCII: f(1), State: f(2), Country: f(3), Lat: lat, Lon: lon, Population: pop, TZ: f(7)}
}

// parseZip parses a zip row: zip, place, state, lat, lon.
func (i *Index) parseZip(off int32) ZipRow {
	f := func(n int) string { return field(i.zips, off, n) }
	lat, _ := strconv.ParseFloat(f(3), 64)
	lon, _ := strconv.ParseFloat(f(4), 64)
	return ZipRow{Zip: f(0), Place: f(1), State: f(2), Lat: lat, Lon: lon}
}

func zipField(b []byte, off int32) string { return field(b, off, 0) }

// Cities returns the city row count.
func (i *Index) Cities() int { return len(i.cityOffs) }

// Zips returns the zip row count.
func (i *Index) Zips() int { return len(i.zipOffs) }

// PrefixSearch returns up to limit cities whose ASCII name starts with the
// query (case-insensitive), ranked by population descending — the type-ahead
// primitive (R-2′; <10ms per-call budget — test-enforced on TypeAhead).
func (i *Index) PrefixSearch(query string, limit int) []City {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" || limit <= 0 {
		return nil
	}
	lo := sort.Search(len(i.cityOffs), func(n int) bool { return i.cityKey(i.cityOffs[n]) >= q })
	// Broad prefixes ("a") match thousands of rows: parse ONLY the population
	// during the scan and keep a running top-limit selection; full rows parse
	// for the winners alone (B2 budget test caught the 12ms full-parse+sort).
	type cand struct {
		off int32
		pop int
	}
	var top []cand
	for n := lo; n < len(i.cityOffs); n++ {
		off := i.cityOffs[n]
		if !strings.HasPrefix(i.cityKey(off), q) {
			break
		}
		pop, _ := strconv.Atoi(field(i.cities, off, 6))
		if len(top) < limit {
			top = append(top, cand{off, pop})
			sort.Slice(top, func(a, b int) bool { return top[a].pop > top[b].pop })
			continue
		}
		if pop > top[len(top)-1].pop {
			top[len(top)-1] = cand{off, pop}
			sort.Slice(top, func(a, b int) bool { return top[a].pop > top[b].pop })
		}
	}
	hits := make([]City, 0, len(top))
	for _, c := range top {
		hits = append(hits, i.parseCity(c.off))
	}
	return hits
}

// TopUS returns the n most-populous US cities, population-descending — the
// RECENT/SEARCHED seed list (B3 UAT session 2: prepopulate the table so the
// layout is judgeable before real search history exists). Same scan shape as
// PrefixSearch: population-only parse, full rows for the winners alone.
func (i *Index) TopUS(n int) []City {
	if n <= 0 {
		return nil
	}
	type cand struct {
		off int32
		pop int
	}
	var top []cand
	for _, off := range i.cityOffs {
		if field(i.cities, off, 3) != "US" {
			continue
		}
		pop, _ := strconv.Atoi(field(i.cities, off, 6))
		if len(top) == n && pop <= top[len(top)-1].pop {
			continue
		}
		if len(top) < n {
			top = append(top, cand{off, pop})
		} else {
			top[len(top)-1] = cand{off, pop}
		}
		sort.Slice(top, func(a, b int) bool { return top[a].pop > top[b].pop })
	}
	hits := make([]City, 0, len(top))
	for _, c := range top {
		hits = append(hits, i.parseCity(c.off))
	}
	return hits
}

// Zip resolves an exact US postal code.
func (i *Index) Zip(zip string) (ZipRow, bool) {
	n := sort.Search(len(i.zipOffs), func(n int) bool { return zipField(i.zips, i.zipOffs[n]) >= zip })
	if n < len(i.zipOffs) && zipField(i.zips, i.zipOffs[n]) == zip {
		return i.parseZip(i.zipOffs[n]), true
	}
	return ZipRow{}, false
}

// RepresentativeZipFast is the type-ahead variant: place-name lookup only
// ("" on miss) — never the O(41k) centroid scan (B2 red-team #3: the scan
// measured 6.2ms per miss, 29.5ms per keystroke with 5 missing rows).
func (i *Index) RepresentativeZipFast(c City) string {
	if c.Country != "US" {
		return ""
	}
	key := strings.ToLower(c.ASCII) + "|" + c.State
	offs := i.zipByPlace[key]
	if len(offs) == 0 {
		key = strings.ToLower(c.Name) + "|" + c.State
		offs = i.zipByPlace[key]
	}
	best := ""
	for _, off := range offs {
		z := i.parseZip(off)
		if best == "" || z.Zip < best {
			best = z.Zip
		}
	}
	return best
}

// RepresentativeZip picks the deterministic zip for a US city (AI-8 §3):
// lowest-numbered zip whose place name matches the city; else the
// nearest-centroid zip (measured 6.2ms — Resolve-path only); "" for non-US.
func (i *Index) RepresentativeZip(c City) string {
	if best := i.RepresentativeZipFast(c); best != "" || c.Country != "US" {
		return best
	}
	best := ""
	// Nearest centroid fallback (measured 6.2ms — Resolve-path only).
	bestDist := 1e18
	for _, off := range i.zipOffs {
		z := i.parseZip(off)
		d := (z.Lat-c.Lat)*(z.Lat-c.Lat) + (z.Lon-c.Lon)*(z.Lon-c.Lon)
		if d < bestDist {
			bestDist = d
			best = z.Zip
		}
	}
	return best
}
