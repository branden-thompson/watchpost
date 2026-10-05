package app

// maptiles.go — the basemap's tile fetches, counted (0.18.0 D-150): the
// library fetches its own tiles, so the Status window's MAP STATUS could say
// nothing of them. watchpost hands the library its transport - the library's
// own, rebuilt - wrapped to count each host's tries, answers, bytes, and
// when it last answered and failed.

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// tileFirstByte is the library's own wait for a connection and a tile's
// first byte; tileIdle how long it keeps a connection open between requests.
const (
	tileFirstByte = 10 * time.Second
	tileIdle      = 90 * time.Second
)

// libraryTransport is the transport the library would use by itself: the
// environment's proxy, reached wherever it listens; every other connection
// through the library's dialer, which refuses a private address (go-tuiMaps
// L-10.3); TLS 1.2 and up; HTTP/2; the library's timeouts.
// TestTheTileTransportMatchesTheLibrarys holds it to the library's source.
func libraryTransport() http.RoundTripper { return libraryTransportVia(http.ProxyFromEnvironment) }

// libraryTransportVia is libraryTransport with its proxy named by proxyOf.
func libraryTransportVia(proxyOf func(*http.Request) (*url.URL, error)) *http.Transport {
	direct := &net.Dialer{Timeout: tileFirstByte}
	pe := httpx.NewProxyExempt(proxyOf, tuimaps.CheckedDialer(), direct.DialContext)
	return &http.Transport{Proxy: pe.Proxy, DialContext: pe.DialContext,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: tileFirstByte,
		ResponseHeaderTimeout: tileFirstByte, IdleConnTimeout: tileIdle, ForceAttemptHTTP2: true}
}

// tileCounter is the tile fetches' counters, by host.
type tileCounter struct {
	mu    sync.Mutex
	hosts map[string]*httpx.HostStats
}

// transport wraps base to count through; a nil counter counts nothing.
func (c *tileCounter) transport(base http.RoundTripper) http.RoundTripper {
	if c == nil {
		return base
	}
	return countingTransport{base: base, count: c}
}

// add applies f to host's counters.
func (c *tileCounter) add(host string, f func(*httpx.HostStats)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hosts == nil {
		c.hosts = map[string]*httpx.HostStats{}
	}
	h, ok := c.hosts[host]
	if !ok {
		h = &httpx.HostStats{Host: host}
		c.hosts[host] = h
	}
	f(h)
}

// stats are the counters as the Status window reads every client's.
func (c *tileCounter) stats() httpx.RequestStats {
	if c == nil {
		return httpx.RequestStats{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var out httpx.RequestStats
	for _, h := range c.hosts {
		out.Hosts = append(out.Hosts, *h)
	}
	return out
}

// forward hands a request to the transport underneath, as a value, so that
// RoundTrip does not call a method of its own name.
var forward = func(rt http.RoundTripper, req *http.Request) (*http.Response, error) { return rt.RoundTrip(req) }

// countingTransport counts each request through it.
type countingTransport struct {
	base  http.RoundTripper
	count *tileCounter
}

func (t countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	t.count.add(host, func(h *httpx.HostStats) { h.Attempts++ })
	res, err := forward(t.base, req)
	now := time.Now()
	switch {
	case err != nil || res.StatusCode >= 400:
		t.count.add(host, func(h *httpx.HostStats) { h.LastFail = now })
	case res.StatusCode == http.StatusNotModified:
		t.count.add(host, func(h *httpx.HostStats) { h.NotModified++; h.LastOK = now })
	default:
		t.count.add(host, func(h *httpx.HostStats) { h.Net++; h.LastOK = now })
		res.Body = countedBody(res.Body, func(n int) { t.count.add(host, func(h *httpx.HostStats) { h.BytesNet += int64(n) }) })
	}
	return res, err
}

// countedBody is body with the bytes read through it counted: the reads go
// through a TeeReader whose writer only counts, and Close is the body's own.
func countedBody(body io.ReadCloser, add func(n int)) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{io.TeeReader(body, byteCounter(add)), body}
}

// byteCounter is a writer that counts what is written to it and keeps none
// of it.
type byteCounter func(n int)

func (c byteCounter) Write(p []byte) (int, error) {
	c(len(p))
	return len(p), nil
}
