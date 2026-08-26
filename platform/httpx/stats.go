package httpx

// Request counters (quality pass Q0, plan §2.1): what the client did since
// launch, per host. They answer M2 ("how chatty is it, really") without a
// proxy, feed the [S] modal and every diagnostic dump, and are the numbers
// later batches are measured against. Design rules from the red team:
// keyed by host, never by URL (IS-7 — a URL can carry a key); a fixed set
// of slots plus "other" so the structure has a stated bound (P10-03); plain
// fields under one mutex, as a Client member, never package state (P10-06).

import (
	"net/url"
	"sort"
	"sync"
	"time"
)

// MaxStatHosts is the number of distinct hosts counted individually; every
// further host folds into the OtherHost row. The app talks to seven hosts
// today (NWS, CO-OPS, NDBC, HMS, WFIGS, FIRMS, Open-Meteo); one spare.
const MaxStatHosts = 8

// OtherHost is the overflow row's name.
const OtherHost = "other"

// HostStats is one host's counters since launch.
type HostStats struct {
	Host          string
	Attempts      int64 // network attempts, retries included
	Net           int64 // bodies received from the network (HTTP 200)
	Cache         int64 // served from the memory or disk tier
	Neg           int64 // served from the negative cache (a remembered 4xx)
	FastFail      int64 // refused at once by the per-host failure memo (normal lane, Q1)
	NotModified   int64 // 304 renewals (conditional GETs land in Q5; 0 until then)
	BytesNet      int64 // body bytes received from the network
	Bytes304      int64 // body bytes a 304 saved (Q5)
	H2            int64 // responses that arrived over HTTP/2
	TLSHandshakes int64 // full TLS handshakes (a resumed session does not count)
}

// RequestStats is a point-in-time copy of every host row, busiest first,
// OtherHost last.
type RequestStats struct {
	Uptime time.Duration
	Hosts  []HostStats
}

// reqStats is the live counter set: MaxStatHosts named slots plus the
// overflow slot at index MaxStatHosts.
type reqStats struct {
	mu      sync.Mutex
	started time.Time
	slots   [MaxStatHosts + 1]HostStats
	used    int // named slots assigned so far
}

func newReqStats() *reqStats { return &reqStats{started: time.Now()} }

// add applies f to host's row, assigning a slot on first sight and folding
// into OtherHost once the named slots are spent.
func (s *reqStats) add(host string, f func(*HostStats)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.slotLocked(host))
}

func (s *reqStats) slotLocked(host string) *HostStats {
	for i := range s.used {
		if s.slots[i].Host == host {
			return &s.slots[i]
		}
	}
	if s.used < MaxStatHosts {
		s.slots[s.used] = HostStats{Host: host}
		s.used++
		return &s.slots[s.used-1]
	}
	other := &s.slots[MaxStatHosts]
	other.Host = OtherHost
	return other
}

// snapshot copies the rows that have seen traffic, busiest first, other last.
func (s *reqStats) snapshot() RequestStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := RequestStats{Uptime: time.Since(s.started), Hosts: make([]HostStats, 0, s.used+1)}
	for i := range s.used {
		out.Hosts = append(out.Hosts, s.slots[i])
	}
	sortHosts(out.Hosts)
	if other := s.slots[MaxStatHosts]; other.Host != "" {
		out.Hosts = append(out.Hosts, other)
	}
	return out
}

func sortHosts(hosts []HostStats) {
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].Attempts > hosts[j].Attempts })
}

// statHost is the counter key for a URL: its hostname, port dropped, or
// OtherHost when it does not parse. Never any part of the path or query.
func statHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return OtherHost
	}
	return u.Hostname()
}

// RequestStats reports this client's counters since it was built.
func (c *Client) RequestStats() RequestStats { return c.stats.snapshot() }

// MergeRequestStats sums several clients' counters into one view (the app
// runs one client per provider family): rows with the same host add field
// by field, the result is re-slotted to MaxStatHosts + OtherHost, and the
// uptime is the longest of the inputs.
func MergeRequestStats(all ...RequestStats) RequestStats {
	var s reqStats
	var uptime time.Duration
	for _, st := range all {
		uptime = max(uptime, st.Uptime)
		for _, h := range st.Hosts {
			s.add(h.Host, func(row *HostStats) { addHostStats(row, h) })
		}
	}
	out := s.snapshot()
	out.Uptime = uptime
	return out
}

func addHostStats(dst *HostStats, src HostStats) {
	dst.Attempts += src.Attempts
	dst.Net += src.Net
	dst.Cache += src.Cache
	dst.Neg += src.Neg
	dst.FastFail += src.FastFail
	dst.NotModified += src.NotModified
	dst.BytesNet += src.BytesNet
	dst.Bytes304 += src.Bytes304
	dst.H2 += src.H2
	dst.TLSHandshakes += src.TLSHandshakes
}
