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
	"net/url"
	"strconv"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/domains/weather/nws/zones"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The closed list's basemap (FR-3.8), registered (W1.13). A custom source is
// a later ruling with its own rules and credit (D-31).
func init() {
	registerMapSource(mapSource{name: "OpenFreeMap", address: "https://tiles.openfreemap.org/planet"})
}

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
	// statedCacheBytes is the one total the listener is told (FR-3.5): the
	// tiles, the web cache and the temperature client's disk (D-219). Radar
	// keeps nothing on disk.
	statedCacheBytes = mapDiskBytes + httpCacheBytes + temperature.CacheBytes
)

// mapHTTPDir is where the temperature client keeps what it fetched (D-219):
// the OS cache directory, beside the tiles.
func mapHTTPDir() string { return userCacheSubdir("map-http") }

// newTempClient is the temperature client, keeping its answers at mapHTTPDir.
func newTempClient(userAgent string) (*httpx.Client, error) {
	return temperature.NewClient(userAgent, mapHTTPDir())
}

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
func newProductionMapBuilder(version string, tiles *tileCounter, seed func()) *mapBuilder {
	return newMapBuilder(version, userCacheSubdir("map"), tiles.transport(libraryTransport()), seed)
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
		func() error { return m.SetImageBudget(radarImageBudget) }, // W8.3b: the loop's frames, decoded
		func() error { return m.Source(mapSources[0].address) },    // the window sets nearby from its Setting (W1.11)
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

// clearMapData empties everything the map keeps on this computer that the
// window's live map cannot reach itself (0.18.0 W3.8 with W9.5 folded): the
// tile cache on disk, through the library's own Purge on a map that names no
// source and seeds nothing, so clearing never fetches; the zone outlines the
// store holds; and the HTTP cache's copies of them. The window purges its
// live map's memory itself.
func (lp *livePipelines) clearMapData() tty.MapCleared {
	var out tty.MapCleared
	if lp.maps != nil {
		m, err := tuimaps.New()
		if err == nil {
			if err = m.CacheRoot(lp.maps.cacheDir, mapDiskBytes); err == nil {
				var rep tuimaps.PurgeReport
				rep, err = m.Purge()
				out.Files = rep.Removed
			}
			m.Close()
		}
		out.Err = err
	}
	for _, c := range lp.mapClients { // the map's web caches, memory and disk (D-219, P10-02)
		n, err := c.ForgetPrefix("")
		out.Files += n
		if out.Err == nil {
			out.Err = err
		}
	}
	if lp.temp != nil { // the land Open-Meteo Marine was found to answer nothing for (D-218)
		n, err := lp.temp.land.forget()
		out.Files += n
		if out.Err == nil {
			out.Err = err
		}
	}
	if lp.zoneShapes != nil {
		out.Zones = lp.zoneShapes.Forget()
		n, err := lp.zoneShapes.ForgetCached()
		out.Zones += n
		if out.Err == nil {
			out.Err = err
		}
	}
	return out
}

// mapSourceList is every service the map contacts, its host and the layers
// it serves - the Status window's MAP STATUS rows (D-150) - built from the
// closed list so it names exactly the hosts contacted (FR-9.4 as D-75 and
// D-151 amend it: what each is sent is one line under them).
func mapSourceList() []tty.MapSource {
	var out []tty.MapSource
	for _, s := range mapSources {
		out = append(out, tty.MapSource{Name: s.name, Host: hostOf(s.address), Layers: "basemap"})
	}
	out = append(out, tty.MapSource{Name: "National Weather Service", Host: hostOf(zones.DefaultBase), Layers: "alert areas"})
	out = append(out, radarHosts()...)                                                             // W8: the radar's two sources
	out = append(out, tempHosts()...)                                                              // W10: the temperature's two
	out = append(out, fireHosts()...)                                                              // D-121: the fire's two
	out = append(out, marineHosts()...)                                                            // D-127, D-128: the sea's stations
	out = append(out, airHosts()...)                                                               // D-138: AirNow
	return append(out, tty.MapSource{Name: "USGS", Host: hostOf(quakeFeedBase), Layers: "quakes"}) // D-122
}

// radarImageBudget is the memory the library may hold the radar's pictures
// in (W8.3b, go-tuiMaps D-68): 24 MiB (D-88, UAT-2 U2-5) holds the worst case,
// four boxes of 24 frames, where 6 MiB refuses a single close box's two-hour
// loop. Tuning it down is F-183.
const radarImageBudget = 24 << 20

// mapRetention is how long the map's data is kept, and the one stated total
// (FR-3.5, FR-3.9).
func mapRetention() string {
	return "Map tiles are kept " + strconv.Itoa(int(mapMaxAge.Hours()/24)) + " days; with the web cache, " +
		strconv.Itoa(statedCacheBytes>>20) + " MB in all."
}

// hostOf is an address's host.
func hostOf(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return address
	}
	return u.Host
}
