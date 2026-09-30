package temperature

// quota.go — a source's spent quota, known and held (W18.1, D-165).
//
// Open-Meteo's free tier is 600 calls a minute, 5,000 an hour and 10,000 a
// day, and each of the map's lattices counts as many. When one is spent it
// answers HTTP 429 with the limit's name in the body - "Daily API request
// limit exceeded" - and every ask after it is refused the same way until the
// period turns. It publishes no reset time and sends no Retry-After.
//
// THE GATE HOLDS A SPENT HOST. Without it every ask of the map goes out,
// waits out the client's retries and comes back refused - counted against
// the quota each time. Held, an ask is refused at once; about once a probe
// interval one goes through, and an answer frees the host. The period's end
// is what the map says for the reset; the probe finds the true one.

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Quota is a spent quota: the host that refused, the limit's period -
// Minutely, Hourly, Daily or Monthly - and when that period turns.
type Quota struct {
	Host   string
	Period string
	Resets time.Time // UTC: the period's end, the best Open-Meteo lets be said
}

// quotaPeriods are the limits Open-Meteo names, in the words of its refusal.
var quotaPeriods = []string{"Minutely", "Hourly", "Daily", "Monthly"}

// quotaProbeEvery is how often a held host is asked again.
const quotaProbeEvery = time.Hour

// QuotaHeldError is an ask the gate refused without asking: the host's quota
// is spent and its probe is not yet due.
type QuotaHeldError struct{ Quota Quota }

func (e *QuotaHeldError) Error() string {
	return e.Quota.Host + ": the " + strings.ToLower(e.Quota.Period) + " request limit is spent; not asked until it resets"
}

// QuotaOf reports whether err is a spent quota: an HTTP 429 whose reason
// names the limit, or an ask the gate held for one.
func QuotaOf(err error, now time.Time) (Quota, bool) {
	var held *QuotaHeldError
	if errors.As(err, &held) {
		return held.Quota, true
	}
	var se *httpx.StatusError
	if !errors.As(err, &se) || se.Status != http.StatusTooManyRequests {
		return Quota{}, false
	}
	for _, p := range quotaPeriods { // four (P10-02)
		if strings.Contains(se.Reason, p+" API request limit exceeded") {
			return Quota{Host: hostOf(se.URL), Period: p, Resets: periodEnd(p, now)}, true
		}
	}
	return Quota{}, false
}

// periodEnd is when a limit's period turns after now, in UTC.
func periodEnd(period string, now time.Time) time.Time {
	now = now.UTC()
	switch period {
	case "Minutely":
		return now.Truncate(time.Minute).Add(time.Minute)
	case "Hourly":
		return now.Truncate(time.Hour).Add(time.Hour)
	case "Monthly":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

// hostOf is a URL's host, or the URL where it has none.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

// QuotaGate is a Getter that holds a host whose quota is spent until its
// probe is due: the sources' client for Open-Meteo.
type QuotaGate struct {
	ask   func(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error) // the getter's GetText, as a value: a call by that name would read to P10 as recursion
	now   func() time.Time
	mu    sync.Mutex
	spent map[string]heldHost
}

// heldHost is a spent host's quota and when it is next asked.
type heldHost struct {
	quota Quota
	probe time.Time
}

// NewQuotaGate wraps get; now is the clock the holds are judged by.
func NewQuotaGate(get Getter, now func() time.Time) *QuotaGate {
	return &QuotaGate{ask: get.GetText, now: now, spent: map[string]heldHost{}}
}

// GetText asks through the gate: refused at once while the host is held and
// its probe not due; otherwise asked, a spent quota holding the host and an
// answer freeing it.
func (g *QuotaGate) GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error) {
	host, now := hostOf(rawURL), g.now()
	g.mu.Lock()
	h, held := g.spent[host]
	g.mu.Unlock()
	if held && now.Before(h.probe) {
		return nil, &QuotaHeldError{Quota: h.quota}
	}
	body, err := g.ask(ctx, rawURL, opts...)
	g.mu.Lock()
	defer g.mu.Unlock()
	if q, ok := QuotaOf(err, now); ok {
		g.spent[host] = heldHost{quota: q, probe: minTime(q.Resets, now.Add(quotaProbeEvery))}
	} else if err == nil {
		delete(g.spent, host)
	}
	return body, err
}

// Refused is the spent quota the map says, if any: the one whose period is
// longest, as it holds longest.
func (g *QuotaGate) Refused() (Quota, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out Quota
	found := false
	for _, h := range g.spent { // a host each: a handful (P10-02)
		if !found || h.quota.Resets.After(out.Resets) {
			out, found = h.quota, true
		}
	}
	return out, found
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
