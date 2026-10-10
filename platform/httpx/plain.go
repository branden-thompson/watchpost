package httpx

import (
	"context"
	"errors"
	"io"
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

// The Plain client's refusals: each names what was wrong, never the reply.
var (
	errPlainNotHTTPS = errors.New("httpx: the address is not https")
	errPlainTooLarge = errors.New("httpx: the reply is larger than this source's cap")
)

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
		ua: cfg.UserAgent, maxBody: cfg.MaxBodyBytes, httpsOnly: cfg.HTTPSOnly}, nil
}

// Get asks an address once, with the validators of the last answer when it
// has them, and returns the reply whatever its status. An error is a request
// that got no reply, or one refused here; it carries the address redacted
// and never the reply's text.
func (p *Plain) Get(ctx context.Context, rawURL, etag, lastModified string) (Answer, error) {
	if p.httpsOnly && !strings.HasPrefix(rawURL, "https://") {
		return Answer{}, &ReachError{URL: RedactURL(rawURL), Err: errPlainNotHTTPS}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Answer{}, &ReachError{URL: RedactURL(rawURL), Err: errors.New("httpx: the address does not parse")}
	}
	req.Header.Set("User-Agent", p.ua)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return Answer{}, &ReachError{URL: RedactURL(rawURL), Err: err}
	}
	defer func() { _ = resp.Body.Close() }() // read in full below; a failed close loses nothing
	body, err := io.ReadAll(io.LimitReader(resp.Body, p.maxBody+1))
	if err != nil {
		return Answer{}, &ReachError{URL: RedactURL(rawURL), Err: err}
	}
	if int64(len(body)) > p.maxBody {
		return Answer{}, &ReachError{URL: RedactURL(rawURL), Err: errPlainTooLarge}
	}
	return Answer{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}
