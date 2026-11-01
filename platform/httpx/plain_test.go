package httpx

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// plainFor is a Plain client for a local test server: plain http and a
// loopback address allowed, as only a test may.
func plainFor(t *testing.T, maxBody int64) *Plain {
	t.Helper()
	p, err := NewPlain(PlainConfig{UserAgent: "watchpost-test", MaxBodyBytes: maxBody})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestThePropagationClientNeverRetriesA429 is 0.19.0 FR-4.4: a 429 is asked
// once and handed back with its status and Retry-After, never retried, never
// held against other hosts.
func TestThePropagationClientNeverRetriesA429(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	a, err := plainFor(t, 0).Get(context.Background(), srv.URL, "", "")
	if err != nil || a.Status != http.StatusTooManyRequests || a.Header.Get("Retry-After") != "120" {
		t.Fatalf("a 429 came back as %d, %v, %v", a.Status, a.Header, err)
	}
	if n := asked.Load(); n != 1 {
		t.Errorf("the 429 was asked %d times", n)
	}
}

// TestThePropagationClientReturnsEveryOutcome is FR-4.4: each answer comes
// back with its status, headers and validators sent - a 200 with its body, a
// 304 without one, a 500 as it is - and nothing is cached: the same address
// asked twice is asked twice.
func TestThePropagationClientReturnsEveryOutcome(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		switch {
		case r.URL.Path == "/fail":
			w.WriteHeader(http.StatusInternalServerError)
		case r.Header.Get("If-None-Match") == `"v1"` && r.Header.Get("If-Modified-Since") == "Fri, 09 Oct 2026 12:00:00 GMT":
			w.WriteHeader(http.StatusNotModified)
		default:
			w.Header().Set("ETag", `"v1"`)
			_, _ = w.Write([]byte("grid"))
		}
	}))
	defer srv.Close()
	p := plainFor(t, 0)
	ok, err := p.Get(context.Background(), srv.URL, "", "")
	if err != nil || ok.Status != 200 || string(ok.Body) != "grid" || ok.Header.Get("ETag") != `"v1"` {
		t.Fatalf("a 200: %d %q %v %v", ok.Status, ok.Body, ok.Header, err)
	}
	same, err := p.Get(context.Background(), srv.URL, `"v1"`, "Fri, 09 Oct 2026 12:00:00 GMT")
	if err != nil || same.Status != 304 || len(same.Body) != 0 {
		t.Errorf("a 304: %d %q %v", same.Status, same.Body, err)
	}
	if bad, err := p.Get(context.Background(), srv.URL+"/fail", "", ""); err != nil || bad.Status != 500 {
		t.Errorf("a 500: %d %v", bad.Status, err)
	}
	if n := asked.Load(); n != 3 {
		t.Errorf("three asks reached the server %d times", n)
	}
}

// TestThePropagationClientIsHardened is FR-4.7: plain http is refused when
// https is required, a private address is refused, a body past the cap is
// refused, and no error carries the reply's text.
func TestThePropagationClientIsHardened(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2048) + " 203.0.113.9"))
	}))
	defer srv.Close()
	strict, err := NewPlain(PlainConfig{UserAgent: "watchpost-test", HTTPSOnly: true, RefusePrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strict.Get(context.Background(), srv.URL, "", ""); err == nil {
		t.Error("plain http was asked with https required")
	}
	private, err := NewPlain(PlainConfig{UserAgent: "watchpost-test", RefusePrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := private.Get(context.Background(), srv.URL, "", ""); err == nil {
		t.Error("a loopback address was dialled with private addresses refused")
	}
	_, err = plainFor(t, 1024).Get(context.Background(), srv.URL, "", "")
	var se *StatusError
	if !errors.As(err, &se) || se.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("a body past the cap is not the data client's 413: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "203.0.113.9") {
		t.Errorf("an error carries the reply's text: %v", err)
	}
	if _, err := NewPlain(PlainConfig{}); err == nil {
		t.Error("a client with no user agent was made")
	}
}

// TestThePropagationClientsErrorsCarryNoSecret is IS-1, as the data client
// keeps it: an address that got no reply is said redacted, after its one
// attempt.
func TestThePropagationClientsErrorsCarryNoSecret(t *testing.T) {
	_, err := plainFor(t, 1024).Get(context.Background(), "http://127.0.0.1:1/x?key=secret-token", "", "")
	var re *ReachError
	if !errors.As(err, &re) || re.Attempts != 1 {
		t.Fatalf("an address with no reply: %v; want a ReachError after one attempt", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Errorf("the error carries the address's secret: %v", err)
	}
}

// TestThePropagationClientCountsItsRequests is D-169: each ask is counted on
// its host as the data client counts it - an answer, a 304, a refusal, no
// reply - so the Status window shows the host's activity.
func TestThePropagationClientCountsItsRequests(t *testing.T) {
	statuses := []int{http.StatusOK, http.StatusNotModified, http.StatusTooManyRequests}
	i := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statuses[i%len(statuses)])
		if statuses[i%len(statuses)] == http.StatusOK {
			_, _ = w.Write([]byte("grid"))
		}
		i++
	}))
	defer srv.Close()
	p := plainFor(t, 1024)
	for range statuses {
		if _, err := p.Get(context.Background(), srv.URL, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	host := statHost(srv.URL)
	var h HostStats
	for _, x := range p.RequestStats().Hosts {
		if x.Host == host {
			h = x
		}
	}
	if h.Attempts != 3 || h.Net != 1 || h.NotModified != 1 || h.BytesNet != 4 || h.LastOK.IsZero() {
		t.Errorf("three asks counted as %+v", h)
	}
	if !h.LastFail.After(h.LastOK) {
		t.Error("a 429 after the answers is not the host's latest word")
	}
	silent := plainFor(t, 1024)
	if _, err := silent.Get(context.Background(), "http://127.0.0.1:1/x", "", ""); err == nil {
		t.Fatal("an address with no listener answered")
	}
	if hs := silent.RequestStats().Hosts; len(hs) != 1 || hs[0].Attempts != 1 || hs[0].LastFail.IsZero() || !hs[0].LastOK.IsZero() {
		t.Errorf("an ask with no reply counted as %+v", hs)
	}
}

// TestEachSourceHasItsOwnCap is 0.19.0 W0.2 (FR-4.4, D-201): a host's own
// cap, tighter than the client's, refuses a body past it as the client's cap
// does; another host keeps the client's.
func TestEachSourceHasItsOwnCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	defer srv.Close()
	host := strings.TrimPrefix(strings.Split(srv.URL, ":")[1], "//")
	capped, err := NewPlain(PlainConfig{UserAgent: "watchpost-test", MaxBodyBytes: 1 << 20, MaxBodyByHost: map[string]int64{host: 1024}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = capped.Get(context.Background(), srv.URL, "", "")
	var se *StatusError
	if !errors.As(err, &se) || se.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("a body past the host's own cap: %v; the client's 413", err)
	}
	other, err := NewPlain(PlainConfig{UserAgent: "watchpost-test", MaxBodyBytes: 1 << 20, MaxBodyByHost: map[string]int64{"elsewhere.example": 1024}})
	if err != nil {
		t.Fatal(err)
	}
	if a, err := other.Get(context.Background(), srv.URL, "", ""); err != nil || len(a.Body) != 2048 {
		t.Errorf("a host with no cap of its own was held to another's: %v", err)
	}
}

// TestAGIRORefusalNeverPausesWeather is 0.19.0 FR-4.4 (D-39, IS-1): a 429
// with a long Retry-After to the propagation client holds nothing the data
// client asks, on that host or another - the two share no pacing hold and no
// host avoidance.
func TestAGIRORefusalNeverPausesWeather(t *testing.T) {
	refused := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refused {
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	data, err := New(Config{UserAgent: "watchpost-test", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if a, err := plainFor(t, 0).Get(context.Background(), srv.URL+"/giro", "", ""); err != nil || a.Status != http.StatusTooManyRequests {
		t.Fatalf("control: the 429 was not handed back: %+v %v", a, err)
	}
	refused = false
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var got struct{ OK bool }
	if _, err := data.GetJSON(ctx, srv.URL+"/weather", &got); err != nil || !got.OK {
		t.Errorf("after the propagation client was refused, the data client waited or failed on the same host: %v", err)
	}
	// Control: the data client does hold after its own 429, so the test above
	// is about the two clients being apart, not about a hold that never happens.
	held, err := New(Config{UserAgent: "watchpost-test", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	refused = true
	_, _ = held.GetJSON(context.Background(), srv.URL+"/weather", &got)
	refused = false
	short, stop := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer stop()
	if _, err := held.GetJSON(short, srv.URL+"/weather", &got); err == nil {
		t.Fatal("control: the data client does not hold after its own 429, so this proves nothing")
	}
}

// TestThePropagationClientCountsItsWireBytes is 0.19.0 G1's downloads (D-141,
// DN-12): what the client read from the wire, a compressed body compressed,
// not the body it handed up; and every connection counted, a second ask
// adding its own.
func TestThePropagationClientCountsItsWireBytes(t *testing.T) {
	body := strings.Repeat("a grid of numbers, much repeated; ", 6000)
	var packed bytes.Buffer
	zw := gzip.NewWriter(&packed)
	_, _ = zw.Write([]byte(body))
	_ = zw.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") != "gzip" {
			t.Errorf("the client asked with Accept-Encoding %q", r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(packed.Bytes())
	}))
	defer srv.Close()
	p := plainFor(t, int64(2*len(body)))
	if p.WireBytes() != 0 {
		t.Fatalf("a new client has read %d bytes", p.WireBytes())
	}
	got, err := p.Get(context.Background(), srv.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Body) != body {
		t.Fatalf("the body handed up is %d bytes, want %d", len(got.Body), len(body))
	}
	one := p.WireBytes()
	if one < int64(packed.Len()) || one > int64(packed.Len())+1024 {
		t.Errorf("one ask read %d bytes off the wire; the compressed body is %d and the body %d", one, packed.Len(), len(body))
	}
	if _, err := p.Get(context.Background(), srv.URL, "", ""); err != nil {
		t.Fatal(err)
	}
	if two := p.WireBytes(); two < one+int64(packed.Len()) {
		t.Errorf("two asks read %d bytes, the first alone %d", two, one)
	}
}

// TestAnAbandonedPlainAskIsNotAFailure is D-354 for the propagation client:
// an ask its caller cancelled is a try and not the host failing; one past its
// deadline is a failure.
func TestAnAbandonedPlainAskIsNotAFailure(t *testing.T) {
	srv, asked := neverAnswers(t)
	ask := func(ctx context.Context) HostStats {
		p := plainFor(t, 1024)
		if _, err := p.Get(ctx, srv.URL, "", ""); err == nil {
			t.Fatal("a server that never answers answered")
		}
		return hostStats(t, p.RequestStats(), statHost(srv.URL))
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-asked; cancel() }()
	if h := ask(ctx); h.Attempts != 1 || !h.LastFail.IsZero() {
		t.Errorf("a cancelled ask counted as %+v; want a try and no failure", h)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	go func() { <-asked }()
	if h := ask(ctx); h.Attempts != 1 || h.LastFail.IsZero() {
		t.Errorf("an ask past its deadline counted as %+v; want a failure", h)
	}
}
