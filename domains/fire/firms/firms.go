// Package firms reads NASA FIRMS active-fire detections (B5; AI-3): the
// keyed upgrade over HMS — VIIRS NOAA-20/21 near-real-time points minutes
// old, via the area CSV API. A free MAP_KEY (firms.modaps.eosdis.nasa.gov/
// api/map_key) is stored by `watchpost setup`; with no key the provider
// contributes nothing and says nothing (HMS carries the default). Quota
// 5,000 transactions per 10 minutes: one request per location per source,
// cached 10 minutes, keeps a 60-location watchlist at ~120.
package firms

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the credit line (NASA open data; attribute FIRMS/LANCE).
const Attribution = "NASA FIRMS fires, needs key (earthdata.nasa.gov)"

// areaTTL keeps a location's box for the fire tier's cadence.
const areaTTL = 10 * time.Minute

// sources are the VIIRS near-real-time products, one request each.
func sources() []string { return []string{"VIIRS_NOAA20_NRT", "VIIRS_NOAA21_NRT"} }

// keyShape is a MAP_KEY: 32 hex characters — the shape httpx redacts from
// every URL it logs or reports (red-team B5 F1: the redaction must never
// depend on a well-formed paste, so anything else is refused here).
var keyShape = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// ValidKey reports whether k has the MAP_KEY shape.
func ValidKey(k string) bool { return keyShape.MatchString(strings.TrimSpace(k)) }

// CheckKey is the actionable form: nil, or why the key is refused (never
// echoing it).
func CheckKey(k string) error {
	k = strings.TrimSpace(k)
	if k == "" || ValidKey(k) {
		return nil
	}
	return fmt.Errorf("a FIRMS MAP_KEY is 32 hex characters (this one has %d) — copy it again from firms.modaps.eosdis.nasa.gov/api/map_key", len(k))
}

// Provider is the FIRMS snapshot provider. The key may change while the
// dashboard runs (the Setup window stores one — UAT 100), so it sits
// behind a lock and Fetch reads it per call.
type Provider struct {
	client *httpx.Client
	base   string
	rules  fire.Rules

	mu  sync.RWMutex
	key string

	tiles *tileMemo // the parsed-tile memo (Q5, plan §2.6)
}

// New builds the provider; base "" means production; key "" (or a key of
// the wrong shape) leaves it disabled.
func New(client *httpx.Client, base, key string, rules fire.Rules) *Provider {
	if base == "" {
		base = "https://firms.modaps.eosdis.nasa.gov"
	}
	p := &Provider{client: client, base: base, rules: rules, tiles: newTileMemo()}
	_ = p.SetKey(key)
	return p
}

// SetKey installs (or clears) the MAP_KEY; a malformed key is refused with
// the CheckKey reason and the provider stays as it was.
func (p *Provider) SetKey(key string) error {
	key = strings.TrimSpace(key)
	if err := CheckKey(key); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.key = key
	return nil
}

// KeyHint is the stored key's last four characters ("" when none) — enough
// for a user to recognise which key is in place, never enough to use it.
func (p *Provider) KeyHint() string {
	k := p.keyNow()
	if len(k) < 4 {
		return ""
	}
	return k[len(k)-4:]
}

// keyNow reads the key under the lock.
func (p *Provider) keyNow() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.key
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "firms" }

// Domains implements snapshot.Provider.
func (p *Provider) Domains() []string { return []string{"fire"} }

// Enabled reports whether a key is configured.
func (p *Provider) Enabled() bool { return p.keyNow() != "" }

// Fetch implements snapshot.Provider for KindFire: one area request per
// location per source, inside the rules' radius, the last day.
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: p.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindFire, "firms serves only KindFire"); err != nil {
		return frag, err
	}
	if err := p.rules.Valid(); err != nil {
		return frag, err
	}
	key := p.keyNow()
	if key == "" {
		return frag, nil // no key: nothing to add, nothing to report (HMS is the default)
	}
	for _, ref := range req.Locations {
		hs, failed, rejectedKey := p.hotspotsFor(ctx, key, ref, &frag)
		if rejectedKey { // the key itself: say so, once, and stop hitting the quota (P4)
			frag.Err = errors.New("firms: FIRMS rejected the MAP_KEY — open Setup ([s]) and paste it again")
			return frag, nil
		}
		if failed {
			continue // unserved: the scheduler retries it and the location keeps its prior hotspots (P2)
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Fire: &snapshot.FireState{AsOf: frag.FetchedAt, Hotspots: fire.Cluster(hs)}}
	}
	return frag, nil
}

// hotspotsFor gathers one location's detections from every tile its box
// touches, per source (Q5 tiles: the tile URL is the cache and singleflight
// key, so locations in one tile share one request; the parsed tile is
// memoised by body hash). failed means some tile could not be served;
// rejectedKey means FIRMS refused the MAP_KEY.
func (p *Provider) hotspotsFor(ctx context.Context, key string, ref snapshot.LocationRef, frag *snapshot.Fragment) (hs []snapshot.Hotspot, failed, rejectedKey bool) {
	w, s, e, n := fire.Bounds(ref.Lat, ref.Lon, p.rules.RadiusKm)
	for _, src := range sources() {
		for _, t := range tilesFor(w, s, e, n, p.tiles.pitchFor(src)) {
			u := p.tileURL(key, src, t)
			raw, err := p.client.GetText(ctx, u, httpx.TTL(areaTTL)) // read-only (httpx.GetText contract)
			if err != nil {
				if rejected(err) {
					return nil, true, true
				}
				frag.Err = fmt.Errorf("firms: %w", redactKey(err, key)) // belt and braces: never trust one layer with a secret
				failed = true
				continue
			}
			p.tiles.noteBody(src, len(raw))
			pts, err := p.tiles.points(tileKey{src: src, tile: t}, raw)
			if err != nil {
				p.client.Forget(u)
				frag.Err = fmt.Errorf("firms: %w", err)
				failed = true
				continue
			}
			hs = append(hs, p.near(ref, pts)...)
		}
	}
	return hs, failed, false
}

// near keeps the points inside the rules' radius and confidence.
func (p *Provider) near(ref snapshot.LocationRef, pts []Point) []snapshot.Hotspot {
	var hs []snapshot.Hotspot
	for _, pt := range pts {
		km, ok := fire.Near(ref, pt.Lat, pt.Lon, p.rules.RadiusKm)
		if !ok || !p.rules.Keep(pt.Confidence, pt.FRPMW) {
			continue
		}
		d := km
		hs = append(hs, snapshot.Hotspot{Lat: pt.Lat, Lon: pt.Lon, DetectedAt: pt.At, Confidence: pt.Confidence, FRPMW: pt.FRPMW, DistanceKm: &d,
			Source: snapshot.SourceInfo{Provider: p.ID(), ModelOrStation: pt.Satellite, IssuedAt: pt.At}})
	}
	return hs
}

// MemoStats is the parsed-tile memo's size and parse count since launch
// (the diagnostic dump's gauges).
func (p *Provider) MemoStats() (tiles, parses int) { return p.tiles.stats() }

// rejected reports a FIRMS answer that means the key, not the network:
// the area API returns 400 "Invalid MAP_KEY." (live 2026-08-25), 401/403
// for a revoked one.
func rejected(err error) bool {
	var se *httpx.StatusError
	if !errors.As(err, &se) {
		return false
	}
	return se.Status == 400 || se.Status == 401 || se.Status == 403
}

// Point is one VIIRS detection.
type Point struct {
	Lat, Lon   float64
	At         time.Time
	Satellite  string
	Confidence string // low | nominal | high
	FRPMW      *float64
	Night      bool
}

// ParseCSV reads the area API's CSV (header-driven; VIIRS columns:
// latitude, longitude, bright_ti4, scan, track, acq_date, acq_time,
// satellite, instrument, confidence, version, bright_ti5, frp, daynight).
func ParseCSV(raw []byte) ([]Point, error) {
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.TrimSpace(h)] = i
	}
	for _, need := range []string{"latitude", "longitude", "acq_date", "acq_time", "confidence"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("csv: column %q missing (an error page instead of data?)", need)
		}
	}
	get := func(r []string, name string) string {
		if i, ok := col[name]; ok && i < len(r) {
			return strings.TrimSpace(r[i])
		}
		return ""
	}
	var out []Point
	for _, r := range rows[1:] {
		var pt Point
		var err1, err2 error
		pt.Lat, err1 = strconv.ParseFloat(get(r, "latitude"), 64)
		pt.Lon, err2 = strconv.ParseFloat(get(r, "longitude"), 64)
		if err1 != nil || err2 != nil {
			continue
		}
		if t, err := time.Parse("2006-01-02 1504", get(r, "acq_date")+" "+fmt.Sprintf("%04s", get(r, "acq_time"))); err == nil {
			pt.At = t.UTC()
		}
		pt.Satellite = satelliteName(get(r, "satellite"))
		pt.Confidence = confidence(get(r, "confidence"))
		if v, err := strconv.ParseFloat(get(r, "frp"), 64); err == nil {
			pt.FRPMW = &v
		}
		pt.Night = get(r, "daynight") == "N"
		out = append(out, pt)
	}
	return out, nil
}

// confidence maps VIIRS l/n/h (and MODIS 0–100) to the shared scale.
func confidence(c string) string {
	switch strings.ToLower(c) {
	case "l", "low":
		return "low"
	case "h", "high":
		return "high"
	case "n", "nominal":
		return "nominal"
	}
	if v, err := strconv.Atoi(c); err == nil {
		switch {
		case v >= 80:
			return "high"
		case v >= 30:
			return "nominal"
		}
		return "low"
	}
	return "nominal"
}

// redactKey scrubs the key from an error's text, whatever produced it.
func redactKey(err error, key string) error {
	if err == nil || key == "" || !strings.Contains(err.Error(), key) {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), key, "[redacted]"))
}

// satelliteName spells the API's codes the way HMS does (U7): N → Suomi
// NPP, N20/N21 → NOAA-20/21; anything else passes through.
func satelliteName(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "N":
		return "Suomi NPP"
	case "N20":
		return "NOAA-20"
	case "N21":
		return "NOAA-21"
	}
	return code
}
