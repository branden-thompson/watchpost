package app

// maps.go — the Observer's map, built at the composition root (0.18.0).
//
// THE MAP IS BUILT HERE AND HANDED TO THE WINDOW, which builds it on the
// listener's g (FR-3.2): nothing is constructed, nothing is fetched and no
// zone outline is seeded before the listener asks for a map. The source is
// the closed list's one basemap (FR-3.8), fetched by the library as
// watchpost and confined by it to that host (HR-6, HR-10); the embedded tiles
// are always passed, so a national view draws offline (FR-3.1).

import (
	"context"
	"net/http"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// mapSource is one entry of FR-3.8's closed list: a name and the exact
// address the library is pointed at. No path takes an address from anywhere
// else.
type mapSource struct {
	name, address string
}

// basemapSources is FR-3.8's basemap half. A custom source is a later ruling
// with its own rules and credit (D-31).
var basemapSources = []mapSource{{name: "OpenFreeMap", address: "https://tiles.openfreemap.org/planet"}}

const (
	// mapDiskBytes caps the tile cache on disk (FR-3.5).
	mapDiskBytes = 256 << 20
	// mapMemoryBytes holds the largest view the window draws with room to
	// spare (FR-3.6): go-tuiMaps measured about 800 KB for a 200x60 view.
	mapMemoryBytes = 4 << 20
	// mapMaxAge is the tiles' stated retention (FR-3.9), kept by the library
	// on disk (HR-8, go-tuiMaps L9.2).
	mapMaxAge = 7 * 24 * time.Hour
	// httpCacheBytes is the existing HTTP cache's cap, which the stated total
	// covers too.
	httpCacheBytes = httpx.DiskCacheBytes
	// statedCacheBytes is the one total the listener is told (FR-3.5). Radar
	// adds its cap with W8.
	statedCacheBytes = mapDiskBytes + httpCacheBytes
)

// mapBuilder builds the window's maps. It holds what outlives one map: the
// memory cache, and whether the zone outlines were seeded.
type mapBuilder struct {
	version   string
	cacheDir  string
	transport http.RoundTripper // nil: the library's own; the tests answer in memory
	seed      func()            // seeds the zone outlines, on the first map only
	once      sync.Once
	sharedErr error
	shared    *tuimaps.Shared
}

func newMapBuilder(version, cacheDir string, transport http.RoundTripper, seed func()) *mapBuilder {
	return &mapBuilder{version: version, cacheDir: cacheDir, transport: transport, seed: seed}
}

// newProductionMapBuilder is the builder the station runs: the OS cache
// directory and the library's own transport.
func newProductionMapBuilder(version string, seed func()) *mapBuilder {
	return newMapBuilder(version, userCacheSubdir("map"), nil, seed)
}

// build makes a map at the window's size: the library moves only a map that
// has a size, so the window's first view can be placed at once. The first
// build seeds the zone outlines (FR-3.2).
func (b *mapBuilder) build(size tuimaps.Size) (*tuimaps.Map, error) {
	b.once.Do(func() {
		b.shared, b.sharedErr = tuimaps.NewShared(mapMemoryBytes)
		if b.seed != nil {
			b.seed()
		}
	})
	if b.sharedErr != nil {
		return nil, b.sharedErr
	}
	m, err := tuimaps.New(tuimaps.WithSize(size.Cols, size.Rows), tuimaps.Embed(assets.Tile, assets.MaxZoom), tuimaps.SharedCaches(b.shared))
	if err != nil {
		return nil, err
	}
	for _, step := range []func() error{
		func() error {
			return m.SetFetchOptions(tuimaps.FetchOptions{UserAgent: "watchpost/" + b.version, Transport: b.transport})
		},
		func() error { return m.CacheRoot(b.cacheDir, mapDiskBytes) },
		func() error { return m.SetCacheMaxAge(mapMaxAge) },
		func() error { return m.Source(basemapSources[0].address) },
	} {
		if err := step(); err != nil {
			m.Close()
			return nil, err
		}
	}
	return m, nil
}

// newMap is the window's map constructor, or nil when the station has no
// builder, which the window says (FR-1.6's "off" arrives with W1.8's Setting).
func (lp *livePipelines) newMap() func(tuimaps.Size) (*tuimaps.Map, error) {
	if lp == nil || lp.maps == nil {
		return nil
	}
	return lp.maps.build
}

// seedOnFirstMap is the builder's seed: the watched places' zone outlines,
// taken when the listener first opens a map and never before (FR-3.2, D-25).
func (lp *livePipelines) seedOnFirstMap(ctx context.Context) func() {
	return func() { lp.seedZoneShapes(ctx, lp.currentWatch()) }
}
