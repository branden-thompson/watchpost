// Package httpx is watchpost's single outbound HTTP client.
//
// Contract (architecture.md §3, §10; AI-1; IS-1): every request carries the
// configured User-Agent (NWS 403s without one); requests flow through a token
// bucket (~5 req/s default) with jittered exponential backoff on 429/5xx; and
// no secret ever appears in an error, log line, or debug string — URLs are
// redacted before they leave this package.
//
// Caching (B3 UAT 71 — see 03-architecture-design/caching.md): every GET is
// memoized by URL. Lifetime, in order: the TTL the caller states (product
// knowledge, e.g. tide predictions are astronomical), else the server's
// Cache-Control max-age / Expires, else none. Entries live in memory and,
// when CacheDir is set, on disk — so a relaunch is warm. Identical
// concurrent GETs share one request (singleflight); a non-retryable 4xx is
// remembered for NegativeTTL so a dead station is not re-hit every cycle.
// Cache hits never consume a pacing token.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// isSecretParam reports whether a query parameter's value must be redacted.
// Covers the keyed providers surveyed in AI-2/AI-3 — extend when a new keyed
// provider lands (scoped as a function per P10-06 minimal scope).
func isSecretParam(name string) bool {
	switch strings.ToLower(name) {
	case "key", "appid", "access_key", "token", "apikey", "api_key", "map_key":
		return true
	}
	return false
}

// Config configures a Client. UserAgent is mandatory.
type Config struct {
	UserAgent  string
	RatePerSec int           // token bucket rate; default 5; max 1000
	RetryBase  time.Duration // backoff base; default 1s; per-attempt backoff caps at 30s
	MaxRetries int           // retries beyond the first attempt; 0 = default (3); -1 = no retries
	Timeout    time.Duration // per-request; default 30s
	CacheDir   string        // on-disk cache tier; "" = memory only
}

// Client is safe for concurrent use.
type Client struct {
	cfg      Config
	http     *http.Client
	cache    *cache
	sf       singleflight.Group // one in-flight request per URL
	inflight [2]chan struct{}   // per-lane cap on requests in flight (UAT 73)

	mu   sync.Mutex
	next [2]time.Time // earliest start per lane (lazy token pacing): [normal, priority]
}

// Resource ceilings (B3 UAT 73 — the adversarial perf pass). The launch
// burst once opened hundreds of connections at once; every one blocked an
// OS thread in a cgo DNS lookup, and Go never retires threads (15 → 137).
// A pure-Go resolver removes the cgo threads; the per-host connection cap
// and the in-flight cap keep the burst to a handful of sockets.
const (
	maxInflight         = 16 // normal lane
	maxInflightPriority = 8  // favourites' lane
	maxConnsPerHost     = 8
	idleConnTimeout     = 90 * time.Second
)

// newTransport builds the shared transport with a pure-Go resolver.
func newTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second,
		Resolver: &net.Resolver{PreferGo: true}}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxConnsPerHost:       maxConnsPerHost,
		MaxIdleConnsPerHost:   maxConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}
}

// Option tunes one request.
type Option func(*reqOpts)

type reqOpts struct {
	ttl     time.Duration
	noCache bool
}

// TTL states how long the response may be reused — product knowledge that
// outranks the server's headers (CO-OPS sends no-store on astronomical
// predictions; NDBC station lists say 60 s for a file that changes daily).
func TTL(d time.Duration) Option { return func(o *reqOpts) { o.ttl = d } }

// NoCache forces a network round trip (and never stores the result).
func NoCache() Option { return func(o *reqOpts) { o.noCache = true } }

// NegativeTTL is how long a non-retryable failure (4xx) is remembered.
const NegativeTTL = 30 * time.Second

// maxBodyBytes bounds a response body (red-team 0.9.0 C-8): the largest
// document this client reads is a gridpoint forecast (~300 KB); anything
// past 32 MB is a misbehaving server, not data.
const maxBodyBytes = 32 << 20

// StatusError is a non-retryable HTTP failure (4xx other than 429).
type StatusError struct {
	URL    string // redacted
	Status int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s returned HTTP %d — not retryable; check the request", e.URL, e.Status)
}

// Priority lane (B3 UAT 64): the favourites' pipeline must never queue
// behind the 50-location seed pipeline's launch burst on a shared client.
// Requests whose context carries WithPriority pace on their own lane at the
// same rate — the momentary ceiling is 2x RatePerSec, still polite — so a
// two-location batch lands in seconds instead of minutes.
type laneKey struct{}

// WithPriority marks every request made under ctx for the priority lane.
func WithPriority(ctx context.Context) context.Context {
	return context.WithValue(ctx, laneKey{}, true)
}

func lane(ctx context.Context) int {
	if v, _ := ctx.Value(laneKey{}).(bool); v {
		return 1
	}
	return 0
}

// New builds a Client. An empty UserAgent is refused: it would produce silent
// 403s from NWS at runtime (AI-1 §3). Error-return recovery per P10 rule 1.
func New(cfg Config) (*Client, error) {
	if err := invariant.Check(cfg.UserAgent != "", "httpx: UserAgent is mandatory (NWS rejects empty UA)"); err != nil {
		return nil, err
	}
	if err := invariant.Check(cfg.RatePerSec >= 0 && cfg.RatePerSec <= 1000, "httpx: RatePerSec must be 0..1000"); err != nil {
		return nil, err
	}
	if err := invariant.Check(cfg.MaxRetries >= -1, "httpx: MaxRetries must be >= -1 (-1 = no retries)"); err != nil {
		return nil, err
	}
	if cfg.RatePerSec <= 0 {
		cfg.RatePerSec = 5
	}
	if cfg.RetryBase <= 0 {
		cfg.RetryBase = time.Second
	}
	switch cfg.MaxRetries {
	case -1:
		cfg.MaxRetries = 0
	case 0:
		cfg.MaxRetries = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	// Lazy token pacing: no background goroutine for pacing (B0 red-team
	// F2). Each request reserves the next start slot under the mutex and
	// sleeps outside it.
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout, Transport: newTransport()}, cache: newCache(cfg.CacheDir),
		inflight: [2]chan struct{}{make(chan struct{}, maxInflight), make(chan struct{}, maxInflightPriority)}}, nil
}

// acquire takes an in-flight slot on the request's lane (or fails on ctx).
func (c *Client) acquire(ctx context.Context) (func(), error) {
	slot := c.inflight[lane(ctx)]
	select {
	case slot <- struct{}{}:
		return func() { <-slot }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// reserve blocks until this request's pacing slot arrives (or ctx ends).
func (c *Client) reserve(ctx context.Context) error {
	interval := time.Second / time.Duration(c.cfg.RatePerSec)
	l := lane(ctx)
	c.mu.Lock()
	now := time.Now()
	start := c.next[l]
	if start.Before(now) {
		start = now
	}
	c.next[l] = start.Add(interval)
	c.mu.Unlock()
	wait := time.Until(start)
	if wait <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

// RedactURL replaces the values of known secret query parameters with REDACTED.
// Unparseable input is fully masked rather than leaked.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable url REDACTED>"
	}
	q := u.Query()
	for name := range q {
		if isSecretParam(name) {
			q.Set(name, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	// Path-segment keys (B0 red-team S-1 tripwire): NASA FIRMS carries its
	// MAP_KEY as a path segment (/api/area/csv/{MAP_KEY}/...). Mask the
	// segment following a "csv"/"json" area-API segment on FIRMS hosts, and
	// any 32-hex path segment anywhere (the FIRMS key shape).
	segs := strings.Split(u.Path, "/")
	for i, seg := range segs {
		if len(seg) == 32 && isHex(seg) {
			segs[i] = "REDACTED"
		}
	}
	u.Path = strings.Join(segs, "/")
	return u.String()
}

// isHex reports whether s is entirely lowercase/uppercase hex.
func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return len(s) > 0
}

// GetJSON fetches a URL (cache first) and decodes the JSON body into out
// (out may be nil). Errors never contain secrets. The returned headers are
// nil on a cache hit.
func (c *Client) GetJSON(ctx context.Context, rawURL string, out any, opts ...Option) (http.Header, error) {
	if err := invariant.Check(rawURL != "", "GetJSON requires a URL"); err != nil {
		return nil, err
	}
	body, hdr, err := c.fetch(ctx, rawURL, opts)
	if err != nil {
		return nil, err
	}
	switch v := out.(type) {
	case nil:
	case *[]byte:
		*v = append([]byte(nil), body...)
	default:
		if err := json.Unmarshal(body, out); err != nil {
			// A poisoned cache entry (truncated file, torn write) decodes badly
			// forever (red-team 0.9.0 F8): forget it and fetch once more.
			c.cache.forget(rawURL)
			if body, hdr, err = c.fetch(ctx, rawURL, append(opts, NoCache())); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(body, out); err != nil {
				return nil, fmt.Errorf("bad response body from %s: %w", RedactURL(rawURL), redactErr(err))
			}
		}
	}
	return hdr, nil
}

// GetText fetches a URL and returns the raw body (text products such as
// NDBC realtime files) through the same pacing, retry, cache and redaction path.
// Forget drops a URL from the cache: a caller whose parser rejected the
// body must not be served it again for the rest of its TTL (the GetJSON
// poison guard, made available to GetText callers — red-team B5 P6).
func (c *Client) Forget(rawURL string) { c.cache.forget(rawURL) }

func (c *Client) GetText(ctx context.Context, rawURL string, opts ...Option) ([]byte, error) {
	var body []byte
	if _, err := c.GetJSON(ctx, rawURL, &body, opts...); err != nil {
		return nil, err
	}
	return body, nil
}

// fetch serves a GET from the cache, else performs it once for every
// concurrent caller (singleflight) and stores it per the lifetime rules.
func (c *Client) fetch(ctx context.Context, rawURL string, opts []Option) ([]byte, http.Header, error) {
	var ro reqOpts
	for _, o := range opts {
		o(&ro)
	}
	if !ro.noCache {
		if body, ok := c.cache.get(rawURL); ok {
			return body, nil, nil
		}
		if err, ok := c.cache.negative(rawURL); ok {
			return nil, nil, err
		}
	}
	type result struct {
		body []byte
		hdr  http.Header
	}
	v, err, _ := c.sf.Do(rawURL, func() (any, error) {
		if !ro.noCache {
			if body, ok := c.cache.get(rawURL); ok { // filled while we waited
				return result{body: body}, nil
			}
		}
		body, hdr, err := c.do(ctx, rawURL)
		if err != nil {
			var se *StatusError
			if !ro.noCache && errors.As(err, &se) && se.Status != http.StatusTooManyRequests {
				c.cache.putNegative(rawURL, err)
			}
			return nil, err
		}
		if ttl := lifetime(ro, hdr); ttl > 0 && !ro.noCache {
			c.cache.put(rawURL, body, time.Now().Add(ttl))
		}
		return result{body: body, hdr: hdr}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	r := v.(result)
	return r.body, r.hdr, nil
}

// lifetime applies the cache-lifetime rules: caller TTL, else server max-age
// / Expires, else none.
func lifetime(ro reqOpts, hdr http.Header) time.Duration {
	if ro.ttl > 0 {
		return ro.ttl
	}
	return serverTTL(hdr)
}

// do performs a GET with pacing and retries, returning the body bytes.
func (c *Client) do(ctx context.Context, rawURL string) ([]byte, http.Header, error) {
	if err := invariant.Check(c.cfg.MaxRetries >= 0, "retry budget must be non-negative"); err != nil {
		return nil, nil, err
	}
	safe := RedactURL(rawURL)
	release, err := c.acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("request cancelled: %s: %w", safe, err)
	}
	defer release()
	var lastStatus int
	var lastErr error
	// Bounded per P10-02: at most 1 + MaxRetries attempts.
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if err := c.reserve(ctx); err != nil {
			return nil, nil, fmt.Errorf("request cancelled: %s: %w", safe, err)
		}
		body, hdr, status, err := c.doAttempt(ctx, rawURL, safe)
		if err == nil && status == http.StatusOK {
			return body, hdr, nil
		}
		if err != nil && status != 0 {
			return nil, nil, err // non-retryable failure, already actionable + redacted
		}
		lastStatus, lastErr = status, err
		if attempt == c.cfg.MaxRetries {
			break
		}
		if err := c.sleepBackoff(ctx, attempt, safe); err != nil {
			if lastErr != nil {
				return nil, nil, fmt.Errorf("%w (last failure: %v)", err, redactErr(lastErr)) // the cause survives the cancellation (F12)
			}
			return nil, nil, err
		}
	}
	if lastErr != nil {
		// Transport-level cause survives, redacted (B0 red-team F1).
		return nil, nil, fmt.Errorf("cannot reach %s after %d attempts: %w", safe, c.cfg.MaxRetries+1, redactErr(lastErr))
	}
	return nil, nil, fmt.Errorf("%s kept failing (last HTTP %d) after %d attempts — provider degraded; serving last-good data upstream", safe, lastStatus, c.cfg.MaxRetries+1)
}

// doAttempt performs one request. Returns (body, headers, 200, nil) on
// success; (nil, nil, status, nil) on a retryable HTTP failure; (nil, nil,
// 0, err) on a transport error (retryable); (nil, nil, status, err) on a
// non-retryable, final error.
func (c *Client) doAttempt(ctx context.Context, rawURL, safe string) ([]byte, http.Header, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("bad request for %s: %w", safe, redactErr(err))
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/geo+json, application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		// Transport error: retryable. Return the cause so the exhaustion
		// message can name it (redacted) instead of a fake HTTP 0 (F1).
		return nil, nil, 0, err
	}
	// Close errors on a fully-drained GET response carry no recoverable
	// information — drain completes the read; ignore explicitly.
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		if !retryable(resp.StatusCode) {
			return nil, nil, resp.StatusCode, &StatusError{URL: safe, Status: resp.StatusCode}
		}
		return nil, nil, resp.StatusCode, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("bad response body from %s: %w", safe, redactErr(err))
	}
	if len(body) > maxBodyBytes {
		return nil, nil, resp.StatusCode, fmt.Errorf("response from %s exceeds %d MB — refused", safe, maxBodyBytes>>20)
	}
	return body, resp.Header, http.StatusOK, nil
}

// sleepBackoff waits base·2^attempt ± 50% jitter, honoring cancellation.
func (c *Client) sleepBackoff(ctx context.Context, attempt int, safe string) error {
	if err := invariant.Check(attempt >= 0, "attempt index must be non-negative"); err != nil {
		return err
	}
	back := min(c.cfg.RetryBase<<min(attempt, 10), 30*time.Second) // cap prevents overflow and multi-day sleeps (F4)
	back += time.Duration(rand.Int63n(int64(back))) - back/2
	select {
	case <-ctx.Done():
		return fmt.Errorf("request cancelled during backoff: %s: %w", safe, ctx.Err())
	case <-time.After(back):
		return nil
	}
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// redactErr rewrites any URL embedded in a transport error so secrets in query
// strings never surface (IS-1). *url.Error is the only stdlib error that
// embeds the full URL.
func redactErr(err error) error {
	if ue, ok := err.(*url.Error); ok {
		return fmt.Errorf("%s %s: %w", ue.Op, RedactURL(ue.URL), ue.Err)
	}
	return err
}

// CacheStats reports the client's memory cache tier.
func (c *Client) CacheStats() Stats { return c.cache.stats() }
