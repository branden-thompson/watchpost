package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Community relays (AI-4): both are volunteer Icecast servers whose terms
// permit direct listening by end users and forbid relaying/harvesting.
// One connection per listener, directory polled at most every 5 minutes,
// 403/404 honoured without retry storms — and the delay disclaimer shown.
const (
	WxradioAttribution    = "NWR audio: wxradio.org & weatherUSA (community)"
	Disclaimer            = "Relayed audio lags; not for life-safety use."
	wxradioStatus         = "https://wxradio.org/status-json.xsl"
	wxradioListen         = "https://wxradio.org/"
	weatherUSAStatus      = "https://radio.weatherusa.net/status-json.xsl"
	weatherUSAListen      = "https://radio.weatherusa.net/"
	directoryTTL          = 5 * time.Minute
	directoryCandidateCap = 3
)

// Mount is one relayed transmitter stream.
type Mount struct {
	Callsign string
	URL      string // HTTPS listen URL
	Relay    string // "wxradio.org" | "weatherusa.net"
	Type     string // content type reported by the directory
	Name     string // directory mount name
}

// Directory reads the relays' Icecast status documents.
type Directory struct {
	client              *httpx.Client
	wxradio, weatherusa string // status URLs (tests override)
	wxListen, wuListen  string
}

// NewDirectory builds a directory over the shared client. Empty bases mean production.
func NewDirectory(client *httpx.Client, wxradioBase, weatherusaBase string) *Directory {
	d := &Directory{client: client, wxradio: wxradioStatus, weatherusa: weatherUSAStatus, wxListen: wxradioListen, wuListen: weatherUSAListen}
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
	out := map[string][]Mount{}
	for _, m := range d.fetch(ctx, d.wxradio, d.wxListen, "wxradio.org", wxradioCallsign) {
		out[m.Callsign] = append(out[m.Callsign], m)
	}
	for _, m := range d.fetch(ctx, d.weatherusa, d.wuListen, "weatherusa.net", weatherUSACallsign) {
		out[m.Callsign] = append(out[m.Callsign], m)
	}
	return out
}

func (d *Directory) fetch(ctx context.Context, statusURL, listenBase, relay string, callOf func(string) string) []Mount {
	var doc icestats
	if _, err := d.client.GetJSON(ctx, statusURL, &doc, httpx.TTL(directoryTTL)); err != nil {
		return nil
	}
	var list []source
	if err := json.Unmarshal(doc.Icestats.Source, &list); err != nil {
		var one source
		if json.Unmarshal(doc.Icestats.Source, &one) != nil {
			return nil
		}
		list = []source{one}
	}
	var out []Mount
	for _, s := range list {
		u, err := url.Parse(s.ListenURL)
		if err != nil || u.Path == "" {
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
