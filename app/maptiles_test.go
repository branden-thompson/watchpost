package app

// maptiles_test.go — 0.18.0 D-150: the basemap's tile fetches counted, and
// the map's own clients joined to the Status window's counters.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

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
