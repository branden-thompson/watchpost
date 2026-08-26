package httpx

// Quality pass Q1 (plan §2.3, Q1 task 4): the six fault pins, the memo's
// bounds, Retry-After clamping, the pacing hold and the redirect policy.
// A fault-injecting server plays the host; the lanes are real.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// faultServer answers per path from a mutable status map; every hit counts.
type faultServer struct {
	*httptest.Server
	hits   atomic.Int32
	status atomic.Int32 // 0 = 200
	retry  atomic.Value // Retry-After header value
}

func newFaultServer(t *testing.T) *faultServer {
	t.Helper()
	fs := &faultServer{}
	fs.retry.Store("")
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.hits.Add(1)
		if ra := fs.retry.Load().(string); ra != "" {
			w.Header().Set("Retry-After", ra)
		}
		if st := int(fs.status.Load()); st != 0 {
			w.WriteHeader(st)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(fs.Close)
	return fs
}

// dashClient is the dashboard's configuration: one retry, fast backoff.
func dashClient(t *testing.T) *Client {
	t.Helper()
	return mustNew(t, Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, RetryBase: time.Millisecond, MaxRetries: 1})
}

func TestSingleFiveHundredHealsWithinFifteenSeconds(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(500)
	start := time.Now()
	_, err := c.GetJSON(context.Background(), fs.URL+"/points/1", nil)
	if err == nil {
		t.Fatal("a 500 twice must fail the call")
	}
	fs.status.Store(0)
	if _, err := c.GetJSON(context.Background(), fs.URL+"/points/1", nil, NoCache()); err != nil {
		t.Fatalf("one URL's 5xx must not memoise the host: %v", err)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatal("a single 5xx must heal within one scheduler retry (≤ 15 s)")
	}
}

func TestOneStationFiveHundredNeverDelaysAlerts(t *testing.T) {
	// Three consecutive 5xx on two distinct URLs arm the memo for the
	// normal lane; the priority lane (alerts) is still attempted and its
	// success clears the memo.
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(502)
	for _, p := range []string{"/stations/A", "/stations/B"} {
		_, _ = c.GetJSON(context.Background(), fs.URL+p, nil) // 2 attempts each = 4 consecutive 5xx over 2 URLs
	}
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); !avoided {
		t.Fatal("≥ 3 consecutive 5xx on ≥ 2 URLs must arm the memo")
	}
	before := fs.hits.Load()
	if _, err := c.GetJSON(context.Background(), fs.URL+"/normal", nil); err == nil || !strings.Contains(err.Error(), "being avoided") {
		t.Fatalf("the normal lane must fail fast while memoised, got %v", err)
	}
	if fs.hits.Load() != before {
		t.Fatal("a fast-fail must not touch the network")
	}
	fs.status.Store(0)
	if _, err := c.GetJSON(WithPriority(context.Background()), fs.URL+"/alerts", nil); err != nil {
		t.Fatalf("the priority lane must attempt regardless of the memo: %v", err)
	}
	if fs.hits.Load() != before+1 {
		t.Fatal("the priority request must have reached the server")
	}
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); avoided {
		t.Fatal("a priority-lane 2xx must clear the memo (half-open probe)")
	}
	if h := hostStats(t, c.RequestStats(), "127.0.0.1"); h.FastFail != 1 {
		t.Fatalf("fast-fails are counted, got %+v", h)
	}
}

func TestThreeDistinctRecentFailuresThenAlertsIsAttempted(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(503)
	for _, p := range []string{"/obs/1", "/obs/2", "/obs/3"} {
		_, _ = c.GetJSON(context.Background(), fs.URL+p, nil)
	}
	before := fs.hits.Load()
	_, _ = c.GetJSON(WithPriority(context.Background()), fs.URL+"/alerts/active", nil)
	if fs.hits.Load() <= before {
		t.Fatal("an alerts tick after three distinct RECENT 5xx must still be attempted")
	}
}

func TestOneTransportErrorDelaysNoPriorityRow(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	host := strings.TrimPrefix(fs.URL, "http://")
	// A transport error on the normal lane arms the memo at once…
	if _, err := c.GetJSON(context.Background(), "http://"+host+"/closed", nil, NoCache()); err == nil {
		fs.Close() // force a transport error: the server is gone
	}
	fs.Close()
	_, _ = c.GetJSON(context.Background(), fs.URL+"/x", nil, NoCache())
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); !avoided {
		t.Fatal("a transport error must arm the memo for the normal lane")
	}
	// …and the priority lane still attempts (the server is down, so it
	// fails on the wire — but it was tried, not refused by the memo).
	_, err := c.GetJSON(WithPriority(context.Background()), fs.URL+"/alerts", nil, NoCache())
	if err == nil || strings.Contains(err.Error(), "being avoided") {
		t.Fatalf("the priority lane never consults the memo, got %v", err)
	}
}

func TestFirstViewUnchangedUnderOneURLFiveHundred(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(500)
	_, _ = c.GetJSON(context.Background(), fs.URL+"/points/only", nil) // 2 attempts, ONE url
	fs.status.Store(0)
	before := fs.hits.Load()
	if _, err := c.GetJSON(context.Background(), fs.URL+"/forecast", nil); err != nil || fs.hits.Load() != before+1 {
		t.Fatalf("one URL's failures must not memoise the host: err=%v hits=%d", err, fs.hits.Load()-before)
	}
}

func TestMemoNeverArmsOnFourXXOrCancel(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(404)
	for i := range 4 {
		_, _ = c.GetJSON(context.Background(), fmt.Sprintf("%s/missing/%d", fs.URL, i), nil)
	}
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); avoided {
		t.Fatal("4xx must never arm the memo")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = c.GetJSON(ctx, fs.URL+"/cancelled", nil, NoCache())
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); avoided {
		t.Fatal("a cancelled context must never arm the memo")
	}
}

func TestMemoRefusesToArmBeyondSixteenHosts(t *testing.T) {
	m := newFailureMemo()
	now := time.Now()
	for i := range memoMaxHosts + 2 {
		m.transportFailure(fmt.Sprintf("h%d", i), now)
	}
	if len(m.hosts) != memoMaxHosts {
		t.Fatalf("the memo must hold at most %d hosts, has %d", memoMaxHosts, len(m.hosts))
	}
	if _, avoided := m.avoiding("h17", now); avoided {
		t.Fatal("a host beyond the cap is never avoided (refuse to arm, never grow)")
	}
}

func TestRetryAfterIsClampedAndHoldsBothLanes(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	for v, want := range map[string]time.Duration{"": 0, "x": 0, "-5": 0, "7": 7 * time.Second, "315360000": memoRetryAfterCap,
		"Thu, 01 Jan 2036 00:00:00 GMT": memoRetryAfterCap, "Wed, 26 Aug 2026 12:00:30 GMT": 30 * time.Second} {
		if got := parseRetryAfter(v, now); got != want {
			t.Fatalf("Retry-After %q → %v, want %v", v, got, want)
		}
	}
	fs := newFaultServer(t)
	c := dashClient(t)
	fs.status.Store(429)
	fs.retry.Store("300")
	_, _ = c.GetJSON(context.Background(), fs.URL+"/limited", nil)
	if wait, avoided := c.memo.avoiding("127.0.0.1", time.Now()); !avoided || wait > memoRetryAfterCap || wait < 4*time.Minute {
		t.Fatalf("a 429 Retry-After feeds the normal-lane memo, clamped to %v: got %v %v", memoRetryAfterCap, wait, avoided)
	}
	c.mu.Lock()
	hold0, hold1 := time.Until(c.next[0]), time.Until(c.next[1])
	c.mu.Unlock()
	if hold0 > pacingHoldCap || hold1 > pacingHoldCap || hold0 < 25*time.Second || hold1 < 25*time.Second {
		t.Fatalf("both lanes hold for min(Retry-After, %v): got %v / %v", pacingHoldCap, hold0, hold1)
	}
}

func TestRedirectPolicy(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"host":"other"}`)) }))
	defer other.Close()
	var hops atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/cross":
			http.Redirect(w, r, strings.Replace(other.URL, "127.0.0.1", "localhost", 1)+"/x", http.StatusFound) // another hostname, same box
		case "/loop":
			hops.Add(1)
			http.Redirect(w, r, fmt.Sprintf("/loop%d", hops.Load()), http.StatusFound)
		default:
			if strings.HasPrefix(r.URL.Path, "/loop") {
				http.Redirect(w, r, r.URL.Path+"x", http.StatusFound)
				return
			}
			_, _ = w.Write([]byte(`{"host":"same"}`))
		}
	}))
	defer srv.Close()
	c := dashClient(t)
	var out struct{ Host string }
	if _, err := c.GetJSON(context.Background(), srv.URL+"/same", &out); err != nil || out.Host != "same" {
		t.Fatalf("a same-origin redirect is followed: %v %+v", err, out)
	}
	if _, err := c.GetJSON(context.Background(), srv.URL+"/cross", nil, NoCache()); err == nil || !strings.Contains(err.Error(), "same-origin") {
		t.Fatalf("a cross-host redirect must be refused: %v", err)
	}
	if _, err := c.GetJSON(context.Background(), srv.URL+"/loop", nil, NoCache()); err == nil || !strings.Contains(err.Error(), "hops") {
		t.Fatalf("more than %d hops must be refused: %v", maxRedirectHops, err)
	}
}
