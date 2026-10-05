package httpx

// radar_hardening_test.go — the radar client's hardening (0.18.0 W8.5,
// W8.14a; FR-5.7, RK-11, D-55): a body past its cap is refused as it reads
// and never cached; an address that resolves to a private one is refused at
// the dial; plain http is refused before anything is sent; the box a map asks
// for never appears in an error.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestABodyPastTheCapIsRefusedAndNeverCached(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Cache-Control", "max-age=300")
		_, _ = w.Write(make([]byte, 2<<20))
	}))
	defer srv.Close()
	c, err := New(Config{UserAgent: "test", MaxBodyBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := c.GetText(context.Background(), srv.URL+"/frame.png"); err == nil {
			t.Fatal("a 2 MiB body past a 1 MiB cap was accepted")
		}
	}
	if _, ok := c.cache.get(srv.URL + "/frame.png"); ok {
		t.Error("the refused body was cached")
	}
	if hits.Load() != 1 {
		t.Errorf("the refusal was not remembered for its short while: %d hits", hits.Load()) // a 4xx is kept as a refusal, never as a body
	}
}

func TestAPrivateAddressIsRefusedAtTheDial(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer srv.Close()
	c, err := New(Config{UserAgent: "test", RefusePrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetText(context.Background(), srv.URL); err == nil || hits.Load() != 0 {
		t.Errorf("a server on 127.0.0.1 was reached (%v, %d hits)", err, hits.Load())
	}
	for addr, refused := range map[string]bool{"127.0.0.1:443": true, "10.1.2.3:443": true, "192.168.0.9:443": true,
		"172.16.4.4:443": true, "169.254.169.254:80": true, "[::1]:443": true, "0.0.0.0:443": true, "[fe80::1]:443": true,
		"8.8.8.8:443": false, "[2606:4700::1111]:443": false} {
		if got := refusePrivate(addr) != nil; got != refused {
			t.Errorf("%s: refused %v, want %v", addr, got, refused)
		}
	}
}

func TestPlainHTTPIsRefusedBeforeAnythingIsSent(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer srv.Close()
	c, err := New(Config{UserAgent: "test", HTTPSOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetText(context.Background(), srv.URL); err == nil || hits.Load() != 0 {
		t.Errorf("plain http was sent (%v, %d hits)", err, hits.Load())
	}
}

func TestTheBoxIsNeverInAnError(t *testing.T) {
	got := RedactURL("https://opengeo.ncep.noaa.gov/geoserver/ows?service=WMS&bbox=-118.5,32.5,-116.4,34.1&width=600")
	if strings.Contains(got, "-118.5") || strings.Contains(got, "34.1") {
		t.Errorf("the box survived redaction: %s", got)
	}
}
