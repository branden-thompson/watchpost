package app

// maps_basemap_test.go — 0.18.0 W3 (the basemap) with W9.3 and W9.4 folded in
// (D-60). Every request is answered in memory by a recording transport: no
// socket is opened.

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"
)

// recorded is a transport that answers OpenFreeMap's TileJSON from the
// recorded fixture and every tile with real tile bytes, and records what it
// was asked and with which user-agent. offline answers nothing.
type recorded struct {
	mu      sync.Mutex
	asked   []string
	agents  []string
	offline bool
	attrib  string // replaces the fixture's attribution, when set
}

func (r *recorded) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.asked = append(r.asked, req.URL.String())
	r.agents = append(r.agents, req.Header.Get("User-Agent"))
	r.mu.Unlock()
	if r.offline {
		return nil, io.ErrUnexpectedEOF
	}
	var body []byte
	if strings.HasSuffix(req.URL.Path, ".pbf") {
		body, _ = assets.Tile(2, 1, 1) // any tile's bytes decode as any tile
	} else {
		b, err := os.ReadFile(filepath.Join("testdata", "maps", "openfreemap-tilejson.json"))
		if err != nil {
			return nil, err
		}
		if r.attrib != "" {
			b = bytes.Replace(b, []byte(`"attribution":"`), []byte(`"attribution":"`+r.attrib+` `), 1)
		}
		body = b
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)), Header: http.Header{}, Request: req}, nil
}

func (r *recorded) requests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.asked...)
}

var mapNoon = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

// testBuilder is the production builder with the network and the cache
// directory replaced.
func testBuilder(t *testing.T, tr http.RoundTripper) *mapBuilder {
	t.Helper()
	return newMapBuilder("0.18.0-test", filepath.Join(t.TempDir(), "map"), tr, nil)
}

// settleAt draws the map at a size and a clock, and works until nothing is
// pending.
func settleAt(t *testing.T, m *tuimaps.Map, size tuimaps.Size, now time.Time) tuimaps.Frame {
	t.Helper()
	if _, err := m.Render(size, now); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	f, err := m.Render(size, now)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestTheBasemapIsTheClosedListsSource is W3.1, W3.7 and W9.3 (FR-3.1, FR-3.8,
// HR-6, HR-10): the map asks OpenFreeMap and nothing else, as watchpost, and
// the list of sources is FR-3.8's, member for member.
func TestTheBasemapIsTheClosedListsSource(t *testing.T) {
	if len(mapSources) != 1 || mapSources[0].address != "https://tiles.openfreemap.org/planet" {
		t.Fatalf("the basemap sources are %+v; FR-3.8 lists OpenFreeMap's planet alone", mapSources)
	}
	tr := &recorded{}
	m, err := testBuilder(t, tr).build(tuimaps.Size{Cols: 69, Rows: 12})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Recentre(tuimaps.LonLat{Lon: -97, Lat: 38}); err != nil {
		t.Fatal(err)
	}
	if err := m.Zoom(6); err != nil {
		t.Fatal(err)
	}
	f := settleAt(t, m, tuimaps.Size{Cols: 69, Rows: 12}, mapNoon)
	asked := tr.requests()
	if len(asked) < 2 {
		t.Fatalf("asked %v; want the TileJSON and the view's tiles", asked)
	}
	for _, a := range asked {
		if !strings.HasPrefix(a, "https://tiles.openfreemap.org/planet") {
			t.Errorf("asked %s, which is not the closed list's source", a)
		}
	}
	for _, agent := range tr.agents {
		if !strings.Contains(agent, "watchpost/0.18.0-test") {
			t.Errorf("a request's user-agent is %q; it names watchpost and its version (HR-6)", agent)
		}
	}
	if f.Status != tuimaps.Complete {
		t.Errorf("a state view with the source answering is %v, want complete", f.Status)
	}
}

// TestAHostileCreditNeverReachesTheWindow is W3.6 (FR-3.7, R3 InfoSec S5):
// the TileJSON's attribution is remote-controlled, and watchpost never prints
// it; the library draws its own fixed credit.
func TestAHostileCreditNeverReachesTheWindow(t *testing.T) {
	tr := &recorded{attrib: "\x1b[2J\x1b[31mALERT: EVACUATE NOW" + strings.Repeat("x", 500)}
	m, err := testBuilder(t, tr).build(tuimaps.Size{Cols: 69, Rows: 12})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	f := settleAt(t, m, tuimaps.Size{Cols: 69, Rows: 12}, mapNoon)
	all := strings.Join(f.Lines, "\n")
	if strings.Contains(all, "EVACUATE") || strings.Contains(all, "\x1b[2J") {
		t.Error("the TileJSON's attribution reached the frame")
	}
	if !strings.Contains(all, "OpenFreeMap") {
		t.Error("the basemap is not credited on the frame")
	}
}

// TestNoMapRequestBeforeTheListenerAsks is W3.2 (FR-3.2, D-21, D-25): the
// zone outlines are seeded by the first map the listener opens, not at the
// station's start; a second map seeds nothing.
func TestNoMapRequestBeforeTheListenerAsks(t *testing.T) {
	seeded := 0
	b := newMapBuilder("t", filepath.Join(t.TempDir(), "map"), &recorded{offline: true}, func() { seeded++ })
	if seeded != 0 {
		t.Fatal("building the builder seeded the zones")
	}
	for range 2 {
		m, err := b.build(tuimaps.Size{Cols: 69, Rows: 12})
		if err != nil {
			t.Fatal(err)
		}
		m.Close()
	}
	if seeded != 1 {
		t.Errorf("the zones were seeded %d times; want once, on the first map", seeded)
	}
	// And the station's start no longer seeds them: seedZoneShapes is called
	// from the map builder alone.
	fset := token.NewFileSet()
	names, _ := filepath.Glob("*.go")
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "seedZoneShapes" && name != "maps.go" {
				t.Errorf("%s: seedZoneShapes is called outside the map builder, before the listener asked", fset.Position(sel.Pos()))
			}
			return true
		})
	}
}

// TestTheMapCachesAreStated is W3.4, W3.5 and W9.4 (FR-3.5, FR-3.6, FR-3.9,
// HR-8): the tiles are kept under the OS cache directory with one stated
// total, the memory cache holds the largest view, and a tile older than seven
// days is fetched again.
func TestTheMapCachesAreStated(t *testing.T) {
	if want := mapDiskBytes + httpCacheBytes; statedCacheBytes != want {
		t.Errorf("the stated total is %d; the configured caps sum to %d", statedCacheBytes, want)
	}
	if prod := newProductionMapBuilder("t", nil); prod.cacheDir != userCacheSubdir("map") {
		t.Errorf("the tile cache is at %q, want the OS cache directory's map folder", prod.cacheDir)
	}
	tr := &recorded{}
	dir := filepath.Join(t.TempDir(), "map")
	m, err := newMapBuilder("t", dir, tr, nil).build(tuimaps.Size{Cols: 149, Rows: 38})
	if err != nil {
		t.Fatal(err)
	}
	if limit := m.CacheUse().Tiles.Limit; limit != mapMemoryBytes {
		t.Errorf("the memory tile cache holds %d, want %d", limit, mapMemoryBytes)
	}
	if err := m.Recentre(tuimaps.LonLat{Lon: -97, Lat: 38}); err != nil {
		t.Fatal(err)
	}
	if err := m.Zoom(6); err != nil {
		t.Fatal(err)
	}
	settleAt(t, m, tuimaps.Size{Cols: 69, Rows: 12}, mapNoon)
	m.Close()
	first := len(tr.requests())

	// Six days on, the tiles are served from disk; eight days on, fetched again.
	// Each is a later session: a new builder, whose memory holds nothing, over
	// the same directory. The maximum age is the disk's (go-tuiMaps L9.2).
	for _, c := range []struct {
		days  int
		again bool
	}{{6, false}, {8, true}} {
		m, err := newMapBuilder("t", dir, tr, nil).build(tuimaps.Size{Cols: 69, Rows: 12})
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Recentre(tuimaps.LonLat{Lon: -97, Lat: 38}); err != nil {
			t.Fatal(err)
		}
		if err := m.Zoom(6); err != nil {
			t.Fatal(err)
		}
		before := len(tr.requests())
		settleAt(t, m, tuimaps.Size{Cols: 69, Rows: 12}, mapNoon.Add(time.Duration(c.days)*24*time.Hour))
		m.Close()
		tiles := 0
		for _, a := range tr.requests()[before:] {
			if strings.HasSuffix(a, ".pbf") {
				tiles++
			}
		}
		if (tiles > 0) != c.again {
			t.Errorf("%d days on: %d tiles fetched again (first open fetched %d requests); want fetched again = %v", c.days, tiles, first, c.again)
		}
	}
}
