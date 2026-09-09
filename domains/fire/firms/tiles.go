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
	"fmt"
	"math"
	"sync"

	"github.com/branden-thompson/watchpost/platform/bodymemo"
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

// tileMemo is the parsed-tile cache and the per-source pitch rule.
//
// THE CACHE IS platform/bodymemo (F-53). This held its own tick counter, hash
// revalidation and least-recently-used eviction, and domains/seismic/usgs held
// a byte-identical copy of all three — one operation implemented twice, which
// is a defect that has not happened yet: the day one is corrected and the other
// is not, they disagree and both look right in isolation. What stays here is
// what is FIRMS's own: a source whose tiles outgrew the body budget moves to
// the split pitch, and nothing outside this package has that rule.
type tileMemo struct {
	cache *bodymemo.Memo[tileKey, []Point]

	mu    sync.Mutex
	split map[string]bool // sources whose tiles exceeded the body budget: on the split pitch from then on
}

type tileKey struct {
	src  string
	tile tile
}

func newTileMemo() *tileMemo {
	return &tileMemo{cache: bodymemo.New[tileKey, []Point](maxTiles), split: map[string]bool{}}
}

// points returns the tile's parsed points, parsing only when the body changed.
//
// ParseCSV IS PASSED AS A TOP-LEVEL FUNCTION, not wrapped: a closure that
// captures anything allocates on every call, hits included, and
// TestTileMemoHitAllocBudget holds a hit at zero.
func (m *tileMemo) points(k tileKey, raw []byte) ([]Point, error) {
	return m.cache.Parsed(k, raw, ParseCSV)
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
func (m *tileMemo) stats() (tiles, parses int) { return m.cache.Stats() }
