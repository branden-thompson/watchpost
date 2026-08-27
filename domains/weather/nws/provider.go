// Package nws is the National Weather Service provider — the mandatory US
// source and the harmonization tie-break authority (T-E′, OQ-9).
//
// Endpoint flow per AI-1 (live-probed): /points/{lat},{lon} resolves the
// gridpoint, station list, and alert zones (cached for the process lifetime —
// grid data changes ~never; a scheduled daily refresh is a B3 long-running-mode
// task, tracked in the B1 ledger); observations come from the nearest station
// that reports a complete observation (4-station fallback chain — UAT 59);
// forecasts from the gridpoint endpoints; alerts from ONE batched
// /alerts/active?zone=... call covering every location's forecastZone AND
// county UGC (the M3 dual-UGC rule). All values normalize to SI (§10.1):
// wmoUnit conversions here, never at render.
package nws

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the NWS credit line for the About view (OQ-15).
const Attribution = "NOAA National Weather Service (api.weather.gov)" // public domain

// stationCandidates is how many nearest observation stations a location
// keeps as an obs fallback chain (B3 UAT 59: Carlsbad's nearest station is
// a mesonet site with no sky condition and an intermittent temperature).
const stationCandidates = 4

// fetchConcurrency bounds per-location fan-out inside one Fetch (UAT 59).
// The httpx token bucket remains the single politeness governor; this only
// stops one slow station from serializing the whole batch.
const fetchConcurrency = 6

// Provider implements snapshot.Provider for api.weather.gov.
type Provider struct {
	client *httpx.Client
	base   string // e.g. https://api.weather.gov (test servers override)

	mu    sync.Mutex
	cache map[snapshot.LocationKey]*gridInfo
	grids map[string]*gridMemo // the decoded gridpoint max/min series per grid URL, by body hash (Q5b-6); pruned with the cache in Retain
	sf    singleflight.Group   // one points resolution per key across concurrent tiers
	now   func() time.Time     // the clock (tests)

	gridDecodes int // gridpoint bodies decoded since launch
}

// New builds the provider. base "" means the production API.
func New(client *httpx.Client, base string) *Provider {
	if base == "" {
		base = "https://api.weather.gov"
	}
	return &Provider{client: client, base: base, cache: map[snapshot.LocationKey]*gridInfo{}, grids: map[string]*gridMemo{}, now: time.Now}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "nws" }

// Domains implements snapshot.Provider: NWS feeds weather AND alerts.
func (p *Provider) Domains() []string { return []string{"weather", "alerts"} }

// Fetch implements snapshot.Provider for one scheduled unit.
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: "nws", Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(len(req.Locations) > 0, "nws: Fetch requires at least one location"); err != nil {
		return frag, err
	}
	switch req.Kind {
	case snapshot.KindObs:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchObs)
	case snapshot.KindForecast:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchForecast)
	case snapshot.KindForecastHourly:
		frag.PerLocation, frag.Err = snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, p.fetchHourly)
	case snapshot.KindAlerts:
		if err := p.fetchAlerts(ctx, req.Locations, &frag); err != nil {
			frag.Err = err
		}
	default:
		return frag, fmt.Errorf("nws does not serve fetch kind %d", req.Kind)
	}
	return frag, nil
}

// FireWeatherZone resolves the NWS fire weather zone for a location (D-22:
// the app INFERS fire-zone membership from the user's own locations — a
// person cannot be expected to know their zone). "" when unresolvable.
func (p *Provider) FireWeatherZone(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return "" // setup degrades to the generic prompt; never blocks
	}
	return g.fireZone
}

// ForecastZone is the location's forecast zone UGC ("CAZ043") from the
// cached point resolution — the synthesized broadcast reads only that
// zone's block of a product (UAT 81). "" when unresolvable.
func (p *Provider) ForecastZone(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	for _, z := range g.zones {
		if len(z) == 6 && z[2] == 'Z' {
			return z
		}
	}
	return ""
}

// CountyUGC is the location's county UGC ("CAC073") from the cached point
// resolution — the radio resolver turns it into a SAME code (B4). "" when
// unresolvable.
func (p *Provider) CountyUGC(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	for _, z := range g.zones {
		if len(z) == 6 && z[2] == 'C' {
			return z
		}
	}
	return ""
}

// Office is the forecast office (CWA id, e.g. "SGX") that issues the
// location's products — from the cached gridpoint URL (B4 synth). "" when
// unresolvable.
func (p *Provider) Office(ctx context.Context, ref snapshot.LocationRef) string {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(g.gridURL, p.base), "/") // /gridpoints/SGX/54,34
	if len(parts) >= 3 && parts[1] == "gridpoints" {
		return parts[2]
	}
	return ""
}

// CachedGrids reports how many locations' gridpoint resolutions are held
// (a structure the diagnostic dump watches; bounded on removal in Q5 — L4-F7).
func (p *Provider) CachedGrids() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.cache)
}

// rebase rewrites api.weather.gov URLs onto p.base, and passes through URLs
// already relative to a BASE placeholder (test fixtures).
func (p *Provider) rebase(raw string) string {
	if strings.HasPrefix(raw, "BASE/") {
		return p.base + strings.TrimPrefix(raw, "BASE")
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return p.base + u.Path
	}
	return raw
}

func lastSegment(u string) string {
	i := strings.LastIndex(u, "/")
	if i < 0 || i == len(u)-1 {
		return ""
	}
	return u[i+1:]
}
