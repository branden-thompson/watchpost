package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Community relays (AI-4): both are volunteer Icecast servers whose terms
// permit direct listening by end users and forbid relaying/harvesting.
// One connection per listener, directory polled at most every 5 minutes,
// 403/404 honoured without retry storms — and the delay disclaimer shown.
//
// weatherUSA is plain HTTP by relay policy (quality pass Q1, DISCOVER
// LR-1 / D1): its directory accepts only RSA-key-exchange TLS suites that
// Go removed, and every mount it advertises is `http://…:80/NWR/*.mp3`
// already — so https:// bought nothing and broke the directory for every
// Go build since 1.22. The app pins mounts to the directory's own host and
// follows only same-origin redirects (httpx.SameOriginRedirect).
const (
	WxradioAttribution    = "NWR audio: wxradio.org & weatherUSA (community)"
	Disclaimer            = "Relayed audio lags; not for life-safety use."
	wxradioStatus         = "https://wxradio.org/status-json.xsl"
	wxradioListen         = "https://wxradio.org/"
	weatherUSAStatus      = "http://radio.weatherusa.net/status-json.xsl"
	weatherUSAListen      = "http://radio.weatherusa.net/"
	directoryTTL          = 5 * time.Minute
	directoryCandidateCap = 3
)

// Mount is one relayed transmitter stream.
type Mount struct {
	Callsign string
	URL      string // listen URL: HTTPS on wxradio.org, plain HTTP on weatherusa.net (relay policy)
	Relay    string // "wxradio.org" | "weatherusa.net"
	Type     string // content type reported by the directory
	Name     string // directory mount name
}

// Status is one relay directory's health after a Mounts call: Err is nil
// when the document was read (or served from cache), else the reason it
// contributed nothing — the deck turns a fresh failure into a
// radio_unavailable warning (Q1, PR-9).
type Status struct {
	Relay string
	Err   error
	Since time.Time // when the failure was first seen ("" while healthy)
}

// Directory reads the relays' Icecast status documents.
type Directory struct {
	client              *httpx.Client
	wxradio, weatherusa string // status URLs (tests override)
	wxListen, wuListen  string
	now                 func() time.Time

	mu   sync.Mutex
	down map[string]Status // relay → remembered failure; a failing directory is asked again only after directoryTTL
}

// NewDirectory builds a directory over the shared client. Empty bases mean production.
func NewDirectory(client *httpx.Client, wxradioBase, weatherusaBase string) *Directory {
	d := &Directory{client: client, wxradio: wxradioStatus, weatherusa: weatherUSAStatus, wxListen: wxradioListen, wuListen: weatherUSAListen, now: time.Now, down: map[string]Status{}}
	if wxradioBase != "" {
		d.wxradio, d.wxListen = wxradioBase+"/status-json.xsl", wxradioBase+"/"
	}
	if weatherusaBase != "" {
		d.weatherusa, d.wuListen = weatherusaBase+"/status-json.xsl", weatherusaBase+"/"
	}
	return d
}

type icestats struct {
	Icestats struct {
		Source json.RawMessage `json:"source"` // one object or an array
	} `json:"icestats"`
}

type source struct {
	ListenURL  string `json:"listenurl"`
	ServerType string `json:"server_type"`
	ServerName string `json:"server_name"`
}

// Mounts returns every relayed transmitter, keyed by callsign (both relays
// merged; a directory that is down contributes nothing rather than an error).
func (d *Directory) Mounts(ctx context.Context) map[string][]Mount {
	m, _ := d.MountsWithStatus(ctx)
	return m
}

// MountsWithStatus is Mounts plus each relay's health.
func (d *Directory) MountsWithStatus(ctx context.Context) (map[string][]Mount, []Status) {
	out := map[string][]Mount{}
	var statuses []Status
	for _, rel := range []struct {
		status, listen, canonical, name string
		callOf                          func(string) string
	}{{d.wxradio, d.wxListen, wxradioListen, "wxradio.org", wxradioCallsign}, {d.weatherusa, d.wuListen, weatherUSAListen, "weatherusa.net", weatherUSACallsign}} {
		mounts, st := d.fetch(ctx, rel.status, rel.listen, rel.canonical, rel.name, rel.callOf)
		statuses = append(statuses, st)
		for _, m := range mounts {
			out[m.Callsign] = append(out[m.Callsign], m)
		}
	}
	return out, statuses
}

// fetch reads one directory. A failure is remembered for directoryTTL so
// Tune/advance/SetMode — each of which resolves — never hit a down relay
// more than once per window (the ToS cadence holds on failure too).
func (d *Directory) fetch(ctx context.Context, statusURL, listenBase, canonical, relay string, callOf func(string) string) ([]Mount, Status) {
	now := d.now()
	d.mu.Lock()
	if st, ok := d.down[relay]; ok && now.Sub(st.Since) < directoryTTL {
		d.mu.Unlock()
		return nil, st
	}
	d.mu.Unlock()
	var doc icestats
	if _, err := d.client.GetJSON(ctx, statusURL, &doc, httpx.TTL(directoryTTL), httpx.Persist()); err != nil {
		return nil, d.markDown(relay, err, now)
	}
	list, ok := sources(doc)
	if !ok {
		return nil, d.markDown(relay, fmt.Errorf("%s directory: unreadable status document", relay), now)
	}
	d.mu.Lock()
	delete(d.down, relay)
	d.mu.Unlock()
	return mountsOf(list, listenBase, canonical, relay, callOf), Status{Relay: relay}
}

func (d *Directory) markDown(relay string, err error, now time.Time) Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	st, ok := d.down[relay]
	if !ok {
		st = Status{Relay: relay, Since: now}
	}
	st.Err = err
	d.down[relay] = st
	return st
}

// sources unpacks the document's one-or-many source field.
func sources(doc icestats) ([]source, bool) {
	var list []source
	if err := json.Unmarshal(doc.Icestats.Source, &list); err == nil {
		return list, true
	}
	var one source
	if json.Unmarshal(doc.Icestats.Source, &one) != nil {
		return nil, false
	}
	return []source{one}, true
}

// mountsOf turns directory sources into mounts. The listen URL is rebuilt
// from the directory's own base (scheme and host pinned — the document
// cannot point the player anywhere else), and a source whose advertised
// host is neither the relay's canonical host nor the base it was fetched
// from (tests) is dropped (RT-9); the port is ignored because Icecast
// advertises `:8000` while the relay serves on 80/443.
func mountsOf(list []source, listenBase, canonical, relay string, callOf func(string) string) []Mount {
	base, err := url.Parse(listenBase)
	if err != nil {
		return nil
	}
	canon, err := url.Parse(canonical)
	if err != nil {
		return nil
	}
	var out []Mount
	for _, s := range list {
		u, err := url.Parse(s.ListenURL)
		if err != nil || u.Path == "" || !(strings.EqualFold(u.Hostname(), base.Hostname()) || strings.EqualFold(u.Hostname(), canon.Hostname())) {
			continue
		}
		name := u.Path[strings.LastIndex(u.Path, "/")+1:]
		call := callOf(name)
		// MP3 only in v0.1 (AI-5): a missing server_type (weatherUSA omits it
		// on some mounts) is taken as MP3 — the mount names say so.
		if call == "" || (s.ServerType != "" && !strings.HasPrefix(s.ServerType, "audio/mp")) {
			continue
		}
		out = append(out, Mount{Callsign: call, URL: strings.TrimSuffix(listenBase, "/") + u.Path, Relay: relay, Type: s.ServerType, Name: name})
	}
	return out
}

// wxradioCallsign: mounts are "ST-City-CALLSIGN".
func wxradioCallsign(mount string) string {
	i := strings.LastIndex(mount, "-")
	if i < 0 {
		return ""
	}
	return strings.ToUpper(mount[i+1:])
}

// weatherUSACallsign: mounts are "CALLSIGN.mp3" or "CALLSIGN_2.mp3".
func weatherUSACallsign(mount string) string {
	name := strings.TrimSuffix(mount, ".mp3")
	if i := strings.Index(name, "_"); i > 0 {
		name = name[:i]
	}
	if name == mount { // no .mp3 suffix: not a weatherUSA NWR mount
		return ""
	}
	return strings.ToUpper(name)
}

// String describes a mount for logs.
func (m Mount) String() string { return fmt.Sprintf("%s via %s", m.Callsign, m.Relay) }
