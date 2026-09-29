package app

// maptiles.go — the basemap's tile fetches, counted (0.18.0 D-150): the
// library fetches its own tiles, so the Status window's MAP STATUS could say
// nothing of them. watchpost hands the library its transport - the library's
// own, rebuilt - wrapped to count each host's tries, answers, bytes, and
// when it last answered and failed.

import (
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// tileFirstByte is the library's own wait for a tile's first byte.
const tileFirstByte = 10 * time.Second

// libraryTransport is the transport the library would use by itself: its
// dialer, which refuses a private address (go-tuiMaps L-10.3), the
// environment's proxy, TLS 1.2 and up, HTTP/2.
func libraryTransport() http.RoundTripper {
	return &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: tuimaps.CheckedDialer(),
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: tileFirstByte,
		ResponseHeaderTimeout: tileFirstByte, ForceAttemptHTTP2: true}
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

// countingTransport counts each request through it.
type countingTransport struct {
	base  http.RoundTripper
	count *tileCounter
}

func (t countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	t.count.add(host, func(h *httpx.HostStats) { h.Attempts++ })
	res, err := t.base.RoundTrip(req)
	now := time.Now()
	switch {
	case err != nil || res.StatusCode >= 400:
		t.count.add(host, func(h *httpx.HostStats) { h.LastFail = now })
	case res.StatusCode == http.StatusNotModified:
		t.count.add(host, func(h *httpx.HostStats) { h.NotModified++; h.LastOK = now })
	default:
		t.count.add(host, func(h *httpx.HostStats) { h.Net++; h.LastOK = now })
		res.Body = countedBody{ReadCloser: res.Body, add: func(n int) { t.count.add(host, func(h *httpx.HostStats) { h.BytesNet += int64(n) }) }}
	}
	return res, err
}

// countedBody counts the bytes read through it.
type countedBody struct {
	io.ReadCloser
	add func(n int)
}

func (b countedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.add(n)
	}
	return n, err
}
