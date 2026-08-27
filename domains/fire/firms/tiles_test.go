package firms

// Quality pass Q5 (plan §2.6, R2-5): the tile grid — shared requests,
// the straddle rule, identical hotspots, the parsed-tile memo and its bound,
// the body-budget split.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func tileServer(t *testing.T, body string) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &paths
}

func TestTilesForCoversTheBoxWithAtMostFour(t *testing.T) {
	rows := []struct {
		name       string
		lat, lon   float64
		wantTiles  int
		wantSouthW [2]int
	}{
		{"inside one tile", 33.24, -117.29, 1, [2]int{-24, 6}},
		{"straddles a meridian", 33.24, -120.1, 2, [2]int{-25, 6}},
		{"straddles a parallel", 35.1, -117.29, 2, [2]int{-24, 6}},
		{"a corner: four", 35.1, -120.1, 4, [2]int{-25, 6}},
	}
	for _, row := range rows {
		w, s, e, n := fire.Bounds(row.lat, row.lon, 25)
		ts := tilesFor(w, s, e, n, tileDeg)
		if len(ts) != row.wantTiles || ts[0].x != row.wantSouthW[0] || ts[0].y != row.wantSouthW[1] {
			t.Errorf("%s: tiles %v, want %d starting at %v", row.name, ts, row.wantTiles, row.wantSouthW)
		}
		for _, tl := range ts {
			tw, tsouth, te, tn := tl.bounds()
			if te-tw != tileDeg || tn-tsouth != tileDeg || tw > e || te < w || tsouth > n || tn < s {
				t.Errorf("%s: tile %v does not touch the box (%v %v %v %v)", row.name, tl, w, s, e, n)
			}
		}
	}
	if got := tilesFor(-125, 24, 125, 49, tileDeg); len(got) != maxTilesPerBox {
		t.Fatalf("a pathological box is bounded at %d tiles, got %d", maxTilesPerBox, len(got))
	}
}

func TestLocationsInOneTileShareOneRequestAndSeeTheSameHotspots(t *testing.T) {
	srv, paths := tileServer(t, csvBody)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	key := "0123456789abcdef0123456789abcdef"
	p := New(c, srv.URL, key, fire.DefaultRules())
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	carlsbad := snapshot.LocationRef{Label: "Carlsbad, CA", Lat: 33.16, Lon: -117.35}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside, carlsbad}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	if len(*paths) != 2 {
		t.Fatalf("two locations in one tile: one request per source, got %d: %v", len(*paths), *paths)
	}
	if !strings.Contains((*paths)[0], "/-120.000,30.000,-115.000,35.000/1") {
		t.Fatalf("the request is the tile, not the box: %s", (*paths)[0])
	}
	fs := frag.PerLocation[snapshot.Key(oceanside)].Fire
	if fs == nil || len(fs.Hotspots) != 1 || *fs.Hotspots[0].FRPMW != 61.5 {
		t.Fatalf("the same near/confident hotspot as the per-box request: %+v", fs)
	}
	if tiles, parses := p.MemoStats(); tiles != 2 || parses != 2 {
		t.Fatalf("one parse per (source, tile): tiles=%d parses=%d", tiles, parses)
	}
	// A second fetch: the client cache serves the tiles, the memo serves the parse.
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside}}); err != nil {
		t.Fatal(err)
	}
	if _, parses := p.MemoStats(); parses != 2 || len(*paths) != 2 {
		t.Fatalf("no re-parse, no re-request: parses=%d requests=%d", parses, len(*paths))
	}
}

func TestStraddlingBoxFetchesEveryTouchedTile(t *testing.T) {
	srv, paths := tileServer(t, csvBody)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	p := New(c, srv.URL, "0123456789abcdef0123456789abcdef", fire.DefaultRules())
	corner := snapshot.LocationRef{Label: "corner", Lat: 35.1, Lon: -120.1}
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{corner}}); err != nil {
		t.Fatal(err)
	}
	if len(*paths) != 8 {
		t.Fatalf("a corner box touches four tiles per source: %d requests", len(*paths))
	}
}

func TestTileMemoIsBoundedAndSplitsAnOversizedSource(t *testing.T) {
	m := newTileMemo()
	for i := range maxTiles + 5 {
		if _, err := m.points(tileKey{src: "a", tile: tile{x: i, y: 0, pitch: tileDeg}}, []byte(csvBody)); err != nil {
			t.Fatal(err)
		}
	}
	if n, _ := m.stats(); n != maxTiles {
		t.Fatalf("the memo holds at most %d tiles, got %d", maxTiles, n)
	}
	if m.pitchFor("a") != tileDeg {
		t.Fatal("the full pitch until a body exceeds the budget")
	}
	m.noteBody("a", tileBodyBudget+1)
	if m.pitchFor("a") != splitTileDeg || m.pitchFor("b") != tileDeg {
		t.Fatal("an oversized tile switches its source — only its source — to the split pitch")
	}
	if got := tilesFor(-117.5, 33.0, -117.0, 33.5, splitTileDeg); len(got) != 1 || got[0].pitch != splitTileDeg {
		t.Fatalf("split tiles: %v", got)
	}
}
