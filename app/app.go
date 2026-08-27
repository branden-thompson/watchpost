// Package app is the composition root (Option C): the ONLY place that wires
// domains, platform, and modes together. cmd/watchpost stays thin.
package app

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// UserAgent identifies watchpost to providers (AI-1: NWS requires contact
// info; a reachable project URL satisfies it). The tree is public, so no
// personal address rides here (b1 red-team S-1, closed at the 0.9.0 exit).
const UserAgent = "watchpost/0.9 (+https://github.com/branden-thompson/watchpost)"

// ReportOnce runs the one-shot fetch pipeline for `watchpost report <query>`:
// resolve the location, fetch obs+forecast+alerts from NWS, and return the
// published snapshot (B1a scope; scheduler-driven live mode arrives B1b/B3).
func ReportOnce(ctx context.Context, query string) (*snapshot.Snapshot, error) {
	snap, _, err := ReportOnceWithStats(ctx, query)
	return snap, err
}

// ReportOnceWithStats is ReportOnce plus the request counters of the
// clients the run used (`report --verbose`, quality pass Q0).
func ReportOnceWithStats(ctx context.Context, query string) (*snapshot.Snapshot, httpx.RequestStats, error) {
	var none httpx.RequestStats
	if err := invariant.Check(query != "", "report requires a location query"); err != nil {
		return nil, none, err
	}
	client, err := newDataClient(3) // one-shot: no scheduler behind it, so it keeps the full ladder (plan §2.3)
	if err != nil {
		return nil, none, err
	}
	ref, fellBack, err := resolveQuery(ctx, client, query)
	if err != nil {
		return nil, none, err
	}
	provider := nws.New(client, "")
	// Same provider set as the dashboard (M5 parity): weather reference +
	// coastal-waters secondaries (UAT 29).
	tides, tidesClient, err := newCoops()
	if err != nil {
		return nil, none, err
	}
	snap, err := reportFetch(ctx, client, provider, tides, ref, fellBack, query)
	if err != nil {
		return nil, none, err
	}
	return snap, requestStats([]*httpx.Client{client, tidesClient}), nil
}

// reportFetch runs every kind through every provider that serves it and
// publishes once (split from ReportOnceWithStats for P10-04).
func reportFetch(ctx context.Context, client *httpx.Client, provider *nws.Provider, tides *coops.Provider, ref snapshot.LocationRef, fellBack bool, query string) (*snapshot.Snapshot, error) {
	cfg, err := config.Load() // rules and the FIRMS key; a missing config is the defaults (report needs none)
	if err != nil {
		return nil, err
	}
	fireProvs, firmsProv := fireProviders(client, cfg)
	providers := append([]snapshot.Provider{provider, nws.NewMarine(provider), ndbc.New(client, ""), tides, coops.NewObs(tides)}, fireProvs...)
	asm := newAssembler([]snapshot.LocationRef{ref}, providers)
	_ = asm.SetInactive(firmsProv.ID(), !firmsProv.Enabled()) // unkeyed FIRMS reads "off", not "ok" (UAT 100)
	if fellBack {
		asm.Warn(snapshot.Warning{Code: snapshot.WarnGeocodeFallback, Location: ref.Label,
			Message: "offline index had no match for " + query + "; resolved via Open-Meteo geocoding"})
	}

	refs := []snapshot.LocationRef{ref}
	// Every kind, every provider that serves it (discovered, not enumerated),
	// the kinds in parallel (Q5, L2-F8: the serial fan-out took 4–10 s cold
	// at NWS's pacing; the assembler applies under its own lock).
	kinds := []snapshot.FetchKind{snapshot.KindObs, snapshot.KindForecast, snapshot.KindForecastHourly, snapshot.KindAlerts, snapshot.KindMarine, snapshot.KindMarineObs, snapshot.KindFire}
	g, gctx := errgroup.WithContext(ctx)
	for _, kind := range kinds {
		g.Go(func() error {
			for _, pr := range providers {
				if !sched.Serves(pr, kind) {
					continue
				}
				frag, err := pr.Fetch(gctx, snapshot.FetchReq{Kind: kind, Locations: refs})
				if err != nil {
					return err // contract violation, not a data failure
				}
				asm.Apply(frag)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	staleWarnings(asm, asm.Snapshot())
	return asm.Snapshot(), nil
}

// newDataClient is the one owner of the data client's policy (Q5, L2-F8):
// the shared user agent, 30 requests/s (the launch burst drains in seconds
// and is still polite to NWS — UAT 6.3), the disk cache, and the retry
// budget the caller's shape allows — 1 under a scheduler that rehydrates
// at 10/20/40 s, 3 for the one-shot report.
func newDataClient(maxRetries int) (*httpx.Client, error) {
	return httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, MaxRetries: maxRetries, CacheDir: cacheDir()})
}

// staleWarnings appends obs_stale warnings for observations older than 2h
// (AI-1 §5: ASOS obs can lag; honesty beats silence — RS-10).
func staleWarnings(asm *snapshot.Assembler, snap *snapshot.Snapshot) {
	if err := invariant.Check(asm != nil && snap != nil, "staleWarnings requires assembler and snapshot"); err != nil {
		return
	}
	for _, loc := range snap.Locations {
		c := loc.Harmonized
		if c.Source.Provider == "" || c.ObservedAt.IsZero() {
			continue
		}
		if age := time.Since(c.ObservedAt); age > 2*time.Hour {
			asm.Warn(snapshot.Warning{
				Code:     snapshot.WarnObsStale,
				Message:  fmt.Sprintf("observation for %s is %s old", loc.Label, age.Round(time.Minute)),
				Location: loc.Label, Provider: c.Source.Provider,
			})
		}
	}
}

// resolveQuery accepts "lat,lon" directly, else resolves embedded-first with
// the Open-Meteo online fallback (AI-8 hybrid; B2).
func resolveQuery(ctx context.Context, client *httpx.Client, query string) (snapshot.LocationRef, bool, error) {
	if err := invariant.Check(client != nil, "resolveQuery requires a client"); err != nil {
		return snapshot.LocationRef{}, false, err
	}
	if lat, lon, ok := parseLatLon(query); ok {
		return snapshot.LocationRef{Label: query, Lat: lat, Lon: lon}, false, nil
	}
	idx, err := geodata.Load()
	if err != nil {
		return snapshot.LocationRef{}, false, err
	}
	resolver, err := locations.New(idx, openmeteo.NewGeocoder(client, ""))
	if err != nil {
		return snapshot.LocationRef{}, false, err
	}
	ref, fellBack, err := resolver.Resolve(ctx, query)
	if err != nil {
		return snapshot.LocationRef{}, false, err
	}
	if err := invariant.Check(ref.Lat != 0 || ref.Lon != 0, "resolver returned a zero coordinate for "+query); err != nil {
		return snapshot.LocationRef{}, false, err
	}
	return ref, fellBack, nil
}

func parseLatLon(s string) (lat, lon float64, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, false
	}
	return lat, lon, true
}
