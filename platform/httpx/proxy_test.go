package httpx

// proxy_test.go — QA-14 and IS-4: a client that refuses private addresses
// still reaches the proxy the environment names, wherever it listens, and the
// refusal covers every range that is not the public internet.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// TestAPrivateProxyIsReachedUnderRefusePrivate is QA-14: behind a local proxy
// (HTTPS_PROXY=http://127.0.0.1:<port>) the hardened client dialled the proxy
// through the private-address check and was refused. The proxy the transport
// resolved is the one address it may reach unchecked.
func TestAPrivateProxyIsReachedUnderRefusePrivate(t *testing.T) {
	var hits atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("through the proxy"))
	}))
	defer proxy.Close()
	cfg := Config{UserAgent: "test", RefusePrivate: true}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	via, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = clientTransport(cfg, func(*http.Request) (*url.URL, error) { return via, nil })
	body, err := c.GetText(context.Background(), "http://public.example/x")
	if err != nil || string(body) != "through the proxy" || hits.Load() != 1 {
		t.Fatalf("the request did not reach the local proxy: %q, %v, %d hits", body, err, hits.Load())
	}
}

// TestADirectDialStillMeetsTheCheckBesideAProxy is the exemption's bound: it
// is the proxy's address alone, so a request the proxy does not carry is still
// refused at the dial.
func TestADirectDialStillMeetsTheCheckBesideAProxy(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer srv.Close()
	cfg := Config{UserAgent: "test", RefusePrivate: true}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	via, _ := url.Parse("http://127.0.0.1:1")
	c.http.Transport = clientTransport(cfg, func(r *http.Request) (*url.URL, error) {
		if r.URL.Host == "public.example" {
			return via, nil
		}
		return nil, nil
	})
	_, _ = c.GetText(context.Background(), "http://public.example/x") // names the proxy, so 127.0.0.1:1 is exempt from now on
	if _, err := c.GetText(context.Background(), srv.URL); err == nil || hits.Load() != 0 {
		t.Errorf("a direct dial to 127.0.0.1 beside a proxy was let through (%v, %d hits)", err, hits.Load())
	}
}

// TestTheReservedRangesAreRefused is IS-4: shared address space, "this"
// network, the benchmarking range, the documentation ranges, class E and NAT64
// all reach something other than the public internet.
func TestTheReservedRangesAreRefused(t *testing.T) {
	for addr, refused := range map[string]bool{
		"100.64.0.1:443": true, "100.127.255.254:443": true, "0.1.2.3:443": true, "198.18.0.1:443": true,
		"198.19.255.255:443": true, "[64:ff9b::a00:1]:443": true, "[64:ff9b::808:808]:443": true,
		"[::ffff:10.0.0.1]:443": true, "[fd00::1]:443": true, "240.0.0.1:443": true, "255.255.255.255:443": true,
		"192.0.2.1:443": true, "[2001:db8::1]:443": true, "[::]:443": true,
		"100.128.0.1:443": false, "198.20.0.1:443": false, "1.1.1.1:443": false, "[2606:4700::1111]:443": false,
	} {
		if got := refusePrivate(addr) != nil; got != refused {
			t.Errorf("%s: refused %v, want %v", addr, got, refused)
		}
	}
}
