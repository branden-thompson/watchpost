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
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	kept  func(rawURL string) ([]byte, bool)                                             // the getter's cache, read while the host is held; nil for a getter without one
	now   func() time.Time
	mu    sync.Mutex
	spent map[string]heldHost
	state string    // the shared state file, "" for none (NewSharedQuotaGate)
	read  time.Time // when the state was last read
	token string    // this gate's name in the state, to know its own probe's claim
}

// quotaStateEvery is how long a gate trusts its read of the shared state.
const quotaStateEvery = 5 * time.Second

// sharedHold is a host's hold as the shared state keeps it.
type sharedHold struct {
	Period        string
	Resets, Probe time.Time
	Prober        string
}

// NewSharedQuotaGate is a gate whose holds are shared through a state file
// with every other gate on it - one per watchpost instance (design 4b): a
// refusal met by one holds them all, one of them probes, an answer frees
// them all. The file unreadable, the gate holds by its own memory.
func NewSharedQuotaGate(get Getter, now func() time.Time, state string) *QuotaGate {
	g := NewQuotaGate(get, now)
	g.state, g.token = state, strconv.FormatInt(time.Now().UnixNano(), 36)+"-"+strconv.Itoa(os.Getpid())
	return g
}

// DefaultQuotaState is $XDG_STATE_HOME/watchpost/quota.json, by default
// ~/.local/state/...; "" where neither resolves.
func DefaultQuotaState() string {
	base := os.Getenv("XDG_STATE_HOME")
	abs := filepath.IsAbs(base)
	if abs {
		return filepath.Join(base, "watchpost", "quota.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "watchpost", "quota.json")
}

// heldHost is a spent host's quota and when it is next asked.
type heldHost struct {
	quota Quota
	probe time.Time
}

// cachedReader is a getter whose response cache can be read without asking
// the network - the httpx client's (D-217).
type cachedReader interface {
	Cached(rawURL string) ([]byte, bool)
}

// NewQuotaGate wraps get; now is the clock the holds are judged by.
func NewQuotaGate(get Getter, now func() time.Time) *QuotaGate {
	g := &QuotaGate{ask: get.GetText, now: now, spent: map[string]heldHost{}}
	if c, ok := get.(cachedReader); ok {
		g.kept = c.Cached
	}
	return g
}

// GetText asks through the gate. While the host is held, an answer already
// in the cache is served - it is paid for (D-217) - and anything else is
// refused until the probe is due; otherwise asked, a spent quota holding the
// host and an answer from the network freeing it.
func (g *QuotaGate) GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error) {
	host, now := hostOf(rawURL), g.now()
	g.mu.Lock()
	g.syncLocked(now)
	h, held := g.spent[host]
	g.mu.Unlock()
	if held && g.kept != nil {
		if body, ok := g.kept(rawURL); ok {
			return body, nil // served, so the probe is left for an ask that reaches the network
		}
	}
	g.mu.Lock()
	h, held = g.spent[host] // as it stands now: another ask may have freed or held it meanwhile
	probing := held && !now.Before(h.probe) && g.claimProbeLocked(host, now)
	g.mu.Unlock()
	if held && !probing {
		return nil, &QuotaHeldError{Quota: h.quota} // held, or another instance probes
	}
	body, err := g.ask(ctx, rawURL, opts...)
	g.mu.Lock()
	defer g.mu.Unlock()
	if q, ok := QuotaOf(err, now); ok {
		g.spent[host] = heldHost{quota: q, probe: minTime(q.Resets, now.Add(quotaProbeEvery))}
		g.saveLocked()
	} else if err == nil && held {
		delete(g.spent, host)
		g.saveLocked()
	}
	return body, err
}

// syncLocked takes the shared state's holds as the gate's, at most once in
// quotaStateEvery: the state is the machine's word. Unreadable, the gate's
// own memory stands.
func (g *QuotaGate) syncLocked(now time.Time) {
	if g.state == "" || now.Sub(g.read) < quotaStateEvery {
		return
	}
	g.read = now
	holds, ok := readQuotaState(g.state)
	if !ok {
		return
	}
	g.spent = map[string]heldHost{}
	for host, h := range holds { // a host each (P10-02)
		g.spent[host] = heldHost{quota: Quota{Host: host, Period: h.Period, Resets: h.Resets}, probe: h.Probe}
	}
}

// claimProbeLocked takes a due probe for this instance: it moves the probe
// forward in the shared state under its own name and reads it back - true
// only when its name stands. With no state, the probe is its own.
func (g *QuotaGate) claimProbeLocked(host string, now time.Time) bool {
	h := g.spent[host]
	h.probe = now.Add(quotaProbeEvery)
	g.spent[host] = h
	if g.state == "" {
		return true
	}
	holds, _ := readQuotaState(g.state)
	if s, ok := holds[host]; ok && now.Before(s.Probe) {
		return false // another instance has moved it: its probe
	}
	if !g.writeState(host, g.token) {
		return true // the state cannot be written: probe by the gate's own memory
	}
	holds, _ = readQuotaState(g.state)
	return holds[host].Prober == g.token
}

// saveLocked writes the gate's holds to the shared state.
func (g *QuotaGate) saveLocked() {
	if g.state == "" {
		return
	}
	g.writeState("", "")
}

// writeState writes the gate's holds, naming prober as host's probe's
// claimant where host is given; by temp file and rename, so a reader sees a
// whole state.
func (g *QuotaGate) writeState(host, prober string) bool {
	holds := map[string]sharedHold{}
	for h, held := range g.spent { // a host each (P10-02)
		holds[h] = sharedHold{Period: held.quota.Period, Resets: held.quota.Resets, Probe: held.probe}
	}
	if s, ok := holds[host]; ok {
		s.Prober = prober
		holds[host] = s
	}
	body, err := json.Marshal(holds)
	if err != nil {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(g.state), 0o700); err != nil {
		return false
	}
	f, err := os.CreateTemp(filepath.Dir(g.state), ".quota-*")
	if err != nil {
		return false
	}
	tmp := f.Name()
	_, werr := f.Write(body)
	cerr := f.Close()
	if werr != nil || cerr != nil || os.Rename(tmp, g.state) != nil {
		_ = os.Remove(tmp)
		return false
	}
	g.read = g.now()
	return true
}

// readQuotaState is the shared state's holds; false where it cannot be read.
func readQuotaState(path string) (map[string]sharedHold, bool) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var holds map[string]sharedHold
	if json.Unmarshal(body, &holds) != nil {
		return nil, false
	}
	return holds, true
}

// Refused is the spent quota the map says, if any: the one whose period is
// longest, as it holds longest.
func (g *QuotaGate) Refused() (Quota, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.syncLocked(g.now()) // the machine's word: another instance's answer frees this one too
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
