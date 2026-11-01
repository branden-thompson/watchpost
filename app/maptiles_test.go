package app

// maptiles_test.go — 0.18.0 D-150: the basemap's tile fetches counted, and
// the map's own clients joined to the Status window's counters.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestTheTilesAreCounted is D-150: every request through the map's transport
// is a try; an answer is counted with its bytes and when it came; a failure
// with when it came - so MAP STATUS can say whether the basemap is working.
func TestTheTilesAreCounted(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer srv.Close()
	c := &tileCounter{}
	client := &http.Client{Transport: c.transport(http.DefaultTransport)}
	get := func() {
		res, err := client.Get(srv.URL + "/1/2/3.pbf")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.ReadAll(res.Body)
		_ = res.Body.Close()
	}
	get()
	fail.Store(true)
	get()
	st := c.stats()
	if len(st.Hosts) != 1 {
		t.Fatalf("%d hosts counted; want the one", len(st.Hosts))
	}
	h := st.Hosts[0]
	if h.Host != "127.0.0.1" || h.Attempts != 2 || h.Net != 1 || h.BytesNet != 10 || h.LastOK.IsZero() || !h.LastFail.After(h.LastOK) {
		t.Errorf("the tiles' counters are %+v; want two tries, one answer of 10 bytes, failing since", h)
	}
	var none *tileCounter
	if none.transport(http.DefaultTransport) != http.DefaultTransport || len(none.stats().Hosts) != 0 {
		t.Error("with no counter the transport is not the base's own")
	}
}

// TestTheMapsClientsReachTheStatusWindow is D-150's wiring: the map's own
// clients' counters - the radar's, the temperature's, the tiles' - are the
// Status window's MapRequests.
func TestTheMapsClientsReachTheStatusWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{}")) }))
	defer srv.Close()
	radarClient, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000})
	if _, err := radarClient.GetJSON(context.Background(), srv.URL+"/radar", nil); err != nil {
		t.Fatal(err)
	}
	tiles := &tileCounter{}
	tiles.add("tiles.openfreemap.org", func(h *httpx.HostStats) { h.Attempts++ })
	lp := &livePipelines{mapClients: []*httpx.Client{radarClient}, tiles: tiles}
	st := lp.ttyStats()
	hosts := map[string]bool{}
	for _, h := range st.MapRequests.Hosts {
		hosts[h.Host] = true
	}
	if !hosts["127.0.0.1"] || !hosts["tiles.openfreemap.org"] {
		t.Errorf("MapRequests holds %v; want the radar client's host and the tiles'", hosts)
	}
}

// closeSpy is a body that says whether it was closed.
type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error { c.closed = true; return nil }

// TestACountedBodyClosesTheBodyItCounts is the counting body's Close: the
// response body underneath is closed with it, so its connection goes back to
// the pool, and the bytes read through it are counted.
func TestACountedBodyClosesTheBodyItCounts(t *testing.T) {
	spy := &closeSpy{Reader: strings.NewReader("twelve bytes")}
	got := 0
	body := countedBody(spy, func(n int) { got += n })
	if _, err := io.ReadAll(body); err != nil {
		t.Fatal(err)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if !spy.closed || got != 12 {
		t.Errorf("closed %v, counted %d; want the body closed and 12 bytes counted", spy.closed, got)
	}
}

// TestAnAbandonedTileIsNotAFailure is 0.19.1 D-354: a tile whose request was
// cancelled - the map moved on, or closed - is a try and nothing more; one
// that ran out of time is the host not answering, a failure.
func TestAnAbandonedTileIsNotAFailure(t *testing.T) {
	asked := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked <- struct{}{}
		<-r.Context().Done() // a tile that never comes
	}))
	defer srv.Close()
	ask := func(c *tileCounter, ctx context.Context) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/9/1/2.pbf", nil)
		if err != nil {
			t.Fatal(err)
		}
		if res, err := c.transport(http.DefaultTransport).RoundTrip(req); err == nil {
			_ = res.Body.Close()
			t.Fatal("a tile that never comes answered")
		}
	}
	gaveUp := &tileCounter{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-asked; cancel() }()
	ask(gaveUp, ctx)
	if h := gaveUp.stats().Hosts[0]; h.Attempts != 1 || !h.LastFail.IsZero() {
		t.Errorf("a cancelled tile counted as %+v; want a try and no failure", h)
	}
	late := &tileCounter{}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	go func() { <-asked }()
	ask(late, ctx)
	if h := late.stats().Hosts[0]; h.Attempts != 1 || h.LastFail.IsZero() {
		t.Errorf("a tile past its deadline counted as %+v; want a failure", h)
	}
}

// nothingBack is a transport that breaks its contract: no reply, no error.
type nothingBack struct{}

func (nothingBack) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

// TestATransportGivingNothingBackIsRefused is the counter's contract with the
// transport underneath: a reply or an error, never neither - which it refuses
// as an error rather than reading a reply that is not there.
func TestATransportGivingNothingBackIsRefused(t *testing.T) {
	c := &tileCounter{}
	req, err := http.NewRequest(http.MethodGet, "https://tiles.openfreemap.org/planet/1/2/3.pbf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := c.transport(nothingBack{}).RoundTrip(req); res != nil || err == nil {
		t.Fatalf("nothing back came through as %v, %v; want an error", res, err)
	}
	if h := tileHost(c); h.Attempts != 1 || !h.LastOK.IsZero() {
		t.Errorf("nothing back counted as %+v; want a try and no answer", h)
	}
}

// heldTiles answers the TileJSON as recorded does and holds every tile until
// its request is given up, saying when the first is asked.
type heldTiles struct {
	recorded
	once  sync.Once
	asked chan struct{}
}

func (h *heldTiles) RoundTrip(req *http.Request) (*http.Response, error) {
	if !strings.HasSuffix(req.URL.Path, ".pbf") {
		return h.recorded.RoundTrip(req)
	}
	h.once.Do(func() { close(h.asked) })
	<-req.Context().Done()
	return nil, req.Context().Err()
}

// counted says when the first tile through it has come back, its outcome
// counted by the transport underneath.
type counted struct {
	base   http.RoundTripper
	once   sync.Once
	gaveUp chan struct{}
}

func (c *counted) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := c.base.RoundTrip(req)
	if strings.HasSuffix(req.URL.Path, ".pbf") {
		c.once.Do(func() { close(c.gaveUp) })
	}
	return res, err
}

// TestATileTheMapGivesUpIsNotAFailure is D-354 through the library: a view
// moved while its tile is on the way has go-tuiMaps cancel the tile, and the
// Status window's row stays as it was - the FAIL a quick zoom left on a
// complete map (reproduced 3 of 3 against OpenFreeMap).
func TestATileTheMapGivesUpIsNotAFailure(t *testing.T) {
	tiles := &tileCounter{}
	held := &heldTiles{asked: make(chan struct{})}
	back := &counted{base: tiles.transport(held), gaveUp: make(chan struct{})}
	size := tuimaps.Size{Cols: 69, Rows: 12}
	m, err := testBuilder(t, back).build(size)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Recentre(tuimaps.LonLat{Lon: -97, Lat: 38}); err != nil {
		t.Fatal(err)
	}
	if err := m.Zoom(9); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Render(size, mapNoon); err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	worked := make(chan struct{})
	go func() {
		defer close(worked)
		for ctx.Err() == nil {
			_, _ = m.Work(ctx)
		}
	}()
	<-held.asked
	if err := m.Recentre(tuimaps.LonLat{Lon: 10, Lat: 50}); err != nil { // another continent: the tile on the way is no longer wanted
		t.Fatal(err)
	}
	if _, err := m.Render(size, mapNoon); err != nil {
		t.Fatal(err)
	}
	select {
	case <-back.gaveUp: // the map's own cancel, before any of the test's
	case <-time.After(5 * time.Second):
		t.Fatal("the map never gave up the tile its old view wanted")
	}
	h := tileHost(tiles)
	stop()
	<-worked
	if !h.LastFail.IsZero() {
		t.Errorf("the tiles counted %+v; a tile the map gave up is not the host failing", h)
	}
}

// tileHost is the tile counter's one host.
func tileHost(c *tileCounter) httpx.HostStats {
	for _, h := range c.stats().Hosts {
		return h
	}
	return httpx.HostStats{}
}
