package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Plain is a client that hands every answer back as it came - its status, its
// headers and its body, a 304 or a 429 included - and retries nothing, caches
// nothing and paces nothing, for a caller that decides its own back-off, as
// go-ionomaps does (0.19.0 FR-4.4). It shares no pacing hold or failure memo
// with the data client. It is hardened as the radar client is (FR-4.7):
// https only, no private address, a body cap.
type Plain struct {
	http      *http.Client
	ua        string
	maxBody   int64
	httpsOnly bool
	stats     *reqStats // its asks per host, as the data client counts them (D-169)
}

// PlainConfig is a Plain client's settings; the zero hardening values allow
// plain http, private addresses and the package's body cap, for tests.
type PlainConfig struct {
	UserAgent     string
	Timeout       time.Duration // per request; default 30 s
	MaxBodyBytes  int64         // refuse a body past this; 0 = the package's 32 MB
	RefusePrivate bool          // refuse to dial a loopback, private, link-local or unspecified address
	HTTPSOnly     bool          // refuse any address that is not https
}

// Answer is one reply as it came.
type Answer struct {
	Status int
	Header http.Header
	Body   []byte // empty on a 304
}

// failed counts a host's ask that got no answer to use: its latest word.
func failed(h *HostStats) { h.LastFail = time.Now() }

// answered counts a reply on its host as the data client does: a body or a
// 304 is an answer; any other status, a refusal (a 429) included, a failure.
func answered(h *HostStats, status, bytes int) {
	switch status {
	case http.StatusOK:
		h.Net++
		h.BytesNet += int64(bytes)
		h.LastOK = time.Now()
	case http.StatusNotModified:
		h.NotModified++
		h.LastOK = time.Now()
	default:
		h.LastFail = time.Now()
	}
}

// errPlainNotHTTPS is the Plain client's refusal of an address that is not
// https: it names what was wrong, never the reply.
var errPlainNotHTTPS = errors.New("httpx: the address is not https")

// NewPlain is a Plain client.
func NewPlain(cfg PlainConfig) (*Plain, error) {
	if err := invariant.Check(cfg.UserAgent != "", "httpx: UserAgent is mandatory"); err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = maxBodyBytes
	}
	t := clientTransport(Config{RefusePrivate: cfg.RefusePrivate}, http.ProxyFromEnvironment)
	return &Plain{http: &http.Client{Timeout: cfg.Timeout, Transport: t, CheckRedirect: SameOriginRedirect},
		ua: cfg.UserAgent, maxBody: cfg.MaxBodyBytes, httpsOnly: cfg.HTTPSOnly, stats: newReqStats()}, nil
}

// RequestStats reports the client's asks since it was built, per host (D-169).
func (p *Plain) RequestStats() RequestStats { return p.stats.snapshot() }

// Get asks an address once, with the validators of the last answer when it
// has them, and returns the reply whatever its status. An error is a request
// refused here, one that got no reply (a ReachError, after its one attempt),
// or a reply past the cap (a StatusError, 413), each as the data client
// gives it; every one carries the address redacted and never the reply's
// text.
func (p *Plain) Get(ctx context.Context, rawURL, etag, lastModified string) (Answer, error) {
	safe := RedactURL(rawURL)
	if p.httpsOnly && !strings.HasPrefix(rawURL, "https://") {
		return Answer{}, fmt.Errorf("refused %s: %w", safe, errPlainNotHTTPS)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Answer{}, fmt.Errorf("bad request for %s: %w", safe, redactErr(err))
	}
	req.Header.Set("User-Agent", p.ua)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	host := statHost(rawURL)
	p.stats.add(host, func(h *HostStats) { h.Attempts++ })
	resp, err := p.http.Do(req)
	if err != nil {
		p.stats.add(host, failed)
		return Answer{}, &ReachError{URL: safe, Attempts: 1, Err: redactErr(err)}
	}
	defer func() { _ = resp.Body.Close() }() // read in full below; a failed close loses nothing
	body, whole, err := readCapped(resp.Body, p.maxBody)
	if err != nil {
		p.stats.add(host, failed)
		return Answer{}, fmt.Errorf("bad response body from %s: %w", safe, redactErr(err))
	}
	if !whole {
		p.stats.add(host, failed)
		return Answer{}, &StatusError{URL: safe, Status: http.StatusRequestEntityTooLarge}
	}
	p.stats.add(host, func(h *HostStats) { answered(h, resp.StatusCode, len(body)) })
	return Answer{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}
