package firms

// tiles.go — the FIRMS request grid (quality pass Q5, plan §2.6, red-team
// R2-5 / SC-8 / PR-8). Fetch is called per location, so "merge boxes" has
// nothing to merge; instead every request is a fixed tile of the globe:
// the URL — the cache key and the singleflight key — is the tile, every
// location inside it shares one request, and a request never exceeds one
// tile. A location's 25 km box that straddles an edge fetches every tile
// it touches (at most four). fire.Near decides membership afterwards, so
// the hotspots a location sees are byte-identical to the per-box request.

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sync"
)

// tileDeg is the grid pitch. A 5° tile is about 1/40 of CONUS: on a peak
// day (~50k detections, ~100 B each in CSV) that is ~125 KB, far under the
// body budget below; the split pitch is the fallback when a tile exceeds it.
const (
	tileDeg        = 5.0
	splitTileDeg   = 2.5
	tileBodyBudget = 2 << 20 // a tile body past this size switches its source to the split pitch
	maxTiles       = 240     // parsed-tile memo bound: the distinct tiles the current location set can touch (60 locations × ≤ 4)
)

// tile is one grid cell, named by its south-west corner in units of its pitch.
type tile struct {
	x, y  int
	pitch float64
}

// bounds is the tile's west, south, east, north edges in degrees.
func (t tile) bounds() (w, s, e, n float64) {
	w, s = float64(t.x)*t.pitch, float64(t.y)*t.pitch
	return w, s, w + t.pitch, s + t.pitch
}

// tilesFor lists the tiles a box touches, west to east then south to
// north. A box narrower than a tile touches ≤ 4; the loop is bounded by
// the box's extent either way (P10-02).
func tilesFor(w, s, e, n float64, pitch float64) []tile {
	x0, x1 := int(math.Floor(w/pitch)), int(math.Floor((e-1e-9)/pitch))
	y0, y1 := int(math.Floor(s/pitch)), int(math.Floor((n-1e-9)/pitch))
	y1 = min(y1, int(math.Floor((90-1e-9)/pitch)))
	out := make([]tile, 0, 4)
	for y := y0; y <= y1 && len(out) < maxTilesPerBox; y++ {
		for x := x0; x <= x1 && len(out) < maxTilesPerBox; x++ {
			out = append(out, tile{x: x, y: y, pitch: pitch})
		}
	}
	return out
}

// maxTilesPerBox bounds a box's tile walk: a 25 km box (≈ 0.45°) touches
// at most four; a pathological rules radius still cannot fan out.
const maxTilesPerBox = 16

// tileURL is the area request for one tile and source.
func (p *Provider) tileURL(key, src string, t tile) string {
	w, s, e, n := t.bounds()
	return fmt.Sprintf("%s/api/area/csv/%s/%s/%.3f,%.3f,%.3f,%.3f/1", p.base, key, src, w, s, e, n) // the key is a 32-hex path segment: httpx redacts it from every error and log line
}

// tileMemo holds the parsed points of the tiles most recently fetched,
// keyed by (source, tile) and revalidated by body hash — a peak-season tile
// is parsed once per body change, not once per location per cycle.
// Bound: maxTiles entries, least-recently-used out (R2-5).
type tileMemo struct {
	mu     sync.Mutex
	tick   uint64
	items  map[tileKey]*tileEntry
	parses int
	split  map[string]bool // sources whose tiles exceeded the body budget: on the split pitch from then on
}

type tileKey struct {
	src  string
	tile tile
}

type tileEntry struct {
	sum  [sha256.Size]byte
	pts  []Point
	used uint64
}

func newTileMemo() *tileMemo {
	return &tileMemo{items: make(map[tileKey]*tileEntry, 64), split: map[string]bool{}}
}

// points returns the tile's parsed points, parsing only when the body changed.
func (m *tileMemo) points(k tileKey, raw []byte) ([]Point, error) {
	sum := sha256.Sum256(raw)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tick++
	if e, ok := m.items[k]; ok && e.sum == sum {
		e.used = m.tick
		return e.pts, nil
	}
	pts, err := ParseCSV(raw)
	if err != nil {
		return nil, err
	}
	m.parses++
	if _, ok := m.items[k]; !ok && len(m.items) >= maxTiles {
		m.evictLocked()
	}
	m.items[k] = &tileEntry{sum: sum, pts: pts, used: m.tick}
	return pts, nil
}

// evictLocked drops the least-recently-used tile (caller holds mu).
func (m *tileMemo) evictLocked() {
	var victim tileKey
	oldest := uint64(math.MaxUint64)
	for k, e := range m.items {
		if e.used < oldest {
			victim, oldest = k, e.used
		}
	}
	delete(m.items, victim)
}

// pitchFor is the grid pitch a source uses: the split pitch once one of its
// tiles has exceeded the body budget.
func (m *tileMemo) pitchFor(src string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.split[src] {
		return splitTileDeg
	}
	return tileDeg
}

// noteBody records a tile body's size; past the budget the source splits.
func (m *tileMemo) noteBody(src string, n int) {
	if n <= tileBodyBudget {
		return
	}
	m.mu.Lock()
	m.split[src] = true
	m.mu.Unlock()
}

// stats is the memo's size and parse count (the diagnostic gauges).
func (m *tileMemo) stats() (tiles, parses int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items), m.parses
}
