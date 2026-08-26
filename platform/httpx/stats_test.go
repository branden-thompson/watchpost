package httpx

// Quality pass Q0 (03-architecture-design/quality-pass-plan.md §2.1): the
// request counters the [S] modal, the plain report (--verbose) and every
// diagnostic dump read. Since launch, keyed by host — never by URL, so a
// key can never carry a secret path segment (red-team IS-7).

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func hostStats(t *testing.T, st RequestStats, host string) HostStats {
	t.Helper()
	for _, h := range st.Hosts {
		if h.Host == host {
			return h
		}
	}
	t.Fatalf("no stats for host %q in %+v", host, st.Hosts)
	return HostStats{}
}

func TestRequestStatsCountByHost(t *testing.T) {
	srv, _ := cacheServer(t, "max-age=60", 200, 0)
	dead, _ := cacheServer(t, "", 404, 0)
	c := newCached(t, "")
	ctx := context.Background()
	for range 2 {
		if _, err := c.GetJSON(ctx, srv.URL+"/a", nil); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if _, err := c.GetJSON(ctx, dead.URL+"/x", nil); err == nil {
			t.Fatal("404 must error")
		}
	}
	st := c.RequestStats()
	if st.Uptime <= 0 {
		t.Fatalf("uptime must run from launch, got %v", st.Uptime)
	}
	h := hostStats(t, st, "127.0.0.1") // both servers are loopback: one slot
	if h.Attempts != 2 || h.Net != 1 || h.Cache != 1 || h.Neg != 1 {
		t.Fatalf("expected 2 attempts / 1 net / 1 cache hit / 1 negative hit, got %+v", h)
	}
	if h.BytesNet != int64(len(`{"n":1}`)) {
		t.Fatalf("BytesNet must be the network body size, got %d", h.BytesNet)
	}
	if cs := c.CacheStats(); cs.Negative != 1 || cs.Entries != 1 {
		t.Fatalf("CacheStats reports both tiers' sizes, got %+v", cs)
	}
	for _, hs := range st.Hosts {
		if strings.ContainsAny(hs.Host, "/?") {
			t.Fatalf("a counter key must be a host, never a URL: %q", hs.Host)
		}
	}
}

func TestRequestStatsCountEveryAttempt(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, err := New(Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, RetryBase: 1, MaxRetries: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetJSON(context.Background(), srv.URL+"/r", nil); err != nil {
		t.Fatal(err)
	}
	h := hostStats(t, c.RequestStats(), "127.0.0.1")
	if h.Attempts != 3 || h.Net != 1 {
		t.Fatalf("retries count as attempts, one net success: got %+v", h)
	}
}

func TestRequestStatsSlotsOverflowToOther(t *testing.T) {
	var s reqStats
	for i := range MaxStatHosts + 3 {
		s.add(fmt.Sprintf("host%d.example", i), func(h *HostStats) { h.Attempts++ })
	}
	st := s.snapshot()
	if len(st.Hosts) != MaxStatHosts+1 {
		t.Fatalf("expected %d host slots + other, got %d", MaxStatHosts, len(st.Hosts))
	}
	last := st.Hosts[len(st.Hosts)-1]
	if last.Host != OtherHost || last.Attempts != 3 {
		t.Fatalf("hosts beyond the fixed slots fold into %q: got %+v", OtherHost, last)
	}
}

func TestRequestStatsBusiestFirstOtherLast(t *testing.T) {
	var s reqStats
	s.add("quiet.example", func(h *HostStats) { h.Attempts++ })
	s.add("busy.example", func(h *HostStats) { h.Attempts += 5 })
	for i := range MaxStatHosts + 2 {
		s.add(fmt.Sprintf("h%d", i), func(h *HostStats) { h.Attempts += 10 }) // overflow lands in other with the biggest count
	}
	st := s.snapshot()
	if st.Hosts[0].Host != "h0" || st.Hosts[len(st.Hosts)-1].Host != OtherHost {
		t.Fatalf("busiest first, other last regardless of count: %+v", st.Hosts)
	}
}

func TestMergeRequestStats(t *testing.T) {
	a := RequestStats{Uptime: 10, Hosts: []HostStats{{Host: "x", Attempts: 1, Net: 1, BytesNet: 5}}}
	b := RequestStats{Uptime: 20, Hosts: []HostStats{{Host: "x", Attempts: 2, Cache: 1}, {Host: "y", Attempts: 1}}}
	m := MergeRequestStats(a, b)
	if m.Uptime != 20 {
		t.Fatalf("merged uptime is the longest, got %v", m.Uptime)
	}
	x := hostStats(t, m, "x")
	if x.Attempts != 3 || x.Net != 1 || x.Cache != 1 || x.BytesNet != 5 {
		t.Fatalf("host rows sum field by field, got %+v", x)
	}
	if hostStats(t, m, "y").Attempts != 1 {
		t.Fatal("a host in one input survives the merge")
	}
}

func TestRequestStatsSeeHTTP2AndTLSHandshakes(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte(`{}`))
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()
	c := newCached(t, "")
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	c.http.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	if _, err := c.GetJSON(context.Background(), srv.URL+"/h2", nil); err != nil {
		t.Fatal(err)
	}
	h := hostStats(t, c.RequestStats(), "127.0.0.1")
	if h.H2 != 1 || h.TLSHandshakes != 1 {
		t.Fatalf("one h2 response over one TLS handshake expected, got %+v", h)
	}
}
