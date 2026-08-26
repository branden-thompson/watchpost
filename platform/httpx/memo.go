package httpx

// Failure memo, redirect policy and Retry-After (quality pass Q1, plan
// §2.3; red-team PA-1, PF-3, IS-3, IS-4, IS-5, R2-3).
//
// DISCOVER measured ~23,000 attempts/hour during an NWS outage: two retry
// ladders (this client's and the scheduler's) multiplied. The fix is one
// retry layer (Config.MaxRetries = 1 on the dashboard clients) plus a
// per-host memo that lets the normal lane fail fast for a short while
// once a host is known to be down. Two rules keep the memo from becoming
// the outage it prevents:
//
//   - ARMING: a transport error arms it at once (the host did not answer);
//     a 5xx arms it only after memoServerFails consecutive failures on at
//     least two distinct URLs (one station's 500 is that station's
//     problem, not api.weather.gov's); a 4xx or a cancelled context never
//     arms it. A 429/503 Retry-After clamps to memoRetryAfterCap.
//   - CONSULTING: only the normal lane reads the memo. The priority lane
//     (alerts, the favourites' first view) always attempts and clears the
//     memo on any 2xx — it is the half-open probe. So "alerts are never
//     delayed by RECENT failures" is a property of the lane, not of the
//     arming statistics.
//
// Bounds: memoMaxHosts rows (refuse to arm beyond), TTL ≤ memoTransportTTL
// (below the scheduler's second rehydrate at 30 s cumulative), never
// refreshed by memoised fast-fails.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	memoMaxHosts      = 16
	memoTransportTTL  = 20 * time.Second
	memoServerFails   = 3 // consecutive 5xx across ≥ memoDistinctURLs URLs before arming
	memoDistinctURLs  = 2
	memoRetryAfterCap = 5 * time.Minute  // a hostile or huge Retry-After never silences a host longer (IS-5)
	pacingHoldCap     = 30 * time.Second // the all-lane pacing hold a Retry-After imposes (R2-3)
	maxRedirectHops   = 3
)

// hostMemo is one host's failure state.
type hostMemo struct {
	until   time.Time // avoid the host on the normal lane until then
	fails   int       // consecutive 5xx
	urls    int       // distinct URLs among those failures
	lastURL string
}

type failureMemo struct {
	mu    sync.Mutex
	hosts map[string]*hostMemo
}

func newFailureMemo() *failureMemo { return &failureMemo{hosts: map[string]*hostMemo{}} }

// avoiding reports whether host is memoised, and for how much longer.
func (m *failureMemo) avoiding(host string, now time.Time) (time.Duration, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.hosts[host]
	if !ok || !now.Before(h.until) {
		return 0, false
	}
	return h.until.Sub(now), true
}

// clear forgets a host: any 2xx from any lane heals it.
func (m *failureMemo) clear(host string) {
	m.mu.Lock()
	delete(m.hosts, host)
	m.mu.Unlock()
}

// rowLocked finds or creates the host's row; nil when the memo is full
// and the host is new (refuse to arm, never grow — P10-03).
func (m *failureMemo) rowLocked(host string) *hostMemo {
	if h, ok := m.hosts[host]; ok {
		return h
	}
	if len(m.hosts) >= memoMaxHosts {
		return nil
	}
	h := &hostMemo{}
	m.hosts[host] = h
	return h
}

// transportFailure arms the memo at once: the host did not answer.
func (m *failureMemo) transportFailure(host string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h := m.rowLocked(host); h != nil {
		h.until = now.Add(memoTransportTTL)
	}
}

// serverFailure counts a 5xx and arms only past the consecutive/distinct
// thresholds.
func (m *failureMemo) serverFailure(host, rawURL string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := m.rowLocked(host)
	if h == nil {
		return
	}
	h.fails++
	if rawURL != h.lastURL {
		h.urls++
		h.lastURL = rawURL
	}
	if h.fails >= memoServerFails && h.urls >= memoDistinctURLs {
		h.until = now.Add(memoTransportTTL)
	}
}

// retryAfter arms the memo for the server's requested pause (already
// clamped) — the normal lane waits it out; the priority lane still probes.
func (m *failureMemo) retryAfter(host string, d time.Duration, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h := m.rowLocked(host); h != nil && now.Add(d).After(h.until) {
		h.until = now.Add(d)
	}
}

// noteFailure classifies one failed attempt for the memo and the pacing
// hold. A cancelled context is the caller's decision, not the host's fault.
func (c *Client) noteFailure(host, rawURL string, res attemptResult, err error) {
	now := time.Now()
	if res.retryAfter > 0 {
		c.holdPacing(res.retryAfter, now)
		c.memo.retryAfter(host, res.retryAfter, now)
	}
	switch {
	case err != nil && res.status == 0:
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		c.memo.transportFailure(host, now)
	case res.status >= 500:
		c.memo.serverFailure(host, rawURL, now)
	}
}

// holdPacing pushes both lanes' next slot out by min(d, pacingHoldCap):
// a Retry-After is honoured on every lane, without a sleep inside do()
// and without the dashboard going dark (R2-3).
func (c *Client) holdPacing(d time.Duration, now time.Time) {
	until := now.Add(min(d, pacingHoldCap))
	c.mu.Lock()
	defer c.mu.Unlock()
	for l := range c.next {
		if c.next[l].Before(until) {
			c.next[l] = until
		}
	}
}

// parseRetryAfter reads an integer-seconds or HTTP-date Retry-After and
// clamps it to memoRetryAfterCap; unparsable or negative values are 0.
func parseRetryAfter(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	var d time.Duration
	if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
		d = time.Duration(secs) * time.Second
	} else if t, err := http.ParseTime(v); err == nil {
		d = t.Sub(now)
	}
	if d <= 0 {
		return 0
	}
	return min(d, memoRetryAfterCap)
}

// errRedirectRefused marks a redirect this client declined: the caller's
// error, never a reason to memoise the host or retry.
var errRedirectRefused = errors.New("refusing redirect")

// SameOriginRedirect is the redirect policy for every client the app
// builds (IS-3): follow at most maxRedirectHops, and only to the same
// scheme and host the request started on — with the relay directory on
// plain HTTP, an on-path redirect must never send the app to another
// host, and a downgrade must never be followed.
func SameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirectHops {
		return fmt.Errorf("%w: more than %d hops", errRedirectRefused, maxRedirectHops)
	}
	first := via[0].URL
	if req.URL.Scheme != first.Scheme || req.URL.Hostname() != first.Hostname() {
		return fmt.Errorf("%w from %s to %s://%s: only same-origin redirects are followed", errRedirectRefused, RedactURL(first.String()), req.URL.Scheme, req.URL.Hostname())
	}
	return nil
}
