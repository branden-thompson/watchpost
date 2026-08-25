// Package sched is the tiered fetch scheduler (architecture §3, §10.8) — THE
// single freshness authority. Each Tier names a FetchKind and its cadence; the
// scheduler fans each due tier out to the providers that serve it, applies
// Fragments to the Assembler, and publishes a fresh Snapshot after every
// cycle. Time flows through the Clock interface so the alert replay harness
// (M2/M3) can drive it deterministically.
package sched

import (
	"context"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Clock abstracts time for testability (§10.8).
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock is the production Clock.
type RealClock struct{}

// Now implements Clock.
func (RealClock) Now() time.Time { return time.Now() }

// After implements Clock.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Tier is one cadence class: fetch Kind every Every.
type Tier struct {
	Kind  snapshot.FetchKind
	Every time.Duration
}

// Config wires a Scheduler.
type Config struct {
	Clock     Clock
	Assembler *snapshot.Assembler
	Locations []snapshot.LocationRef
	Providers []snapshot.Provider
	Tiers     []Tier
	Hints     map[string]map[string]string // provider id -> FetchReq.Hint
	OnPublish func()                       // "new data applied" — MAY run concurrently from multiple tiers; the app snapshots (coalesced, UAT 74)
	OnWarn    func(snapshot.Warning)       // optional observer (logs, M2 latency)
}

// Scheduler runs the tiers until Stop or context cancellation.
type Scheduler struct {
	cfg  Config
	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup

	mu   sync.Mutex
	locs []snapshot.LocationRef // live location set (Update)
	ctx  context.Context        // set by Start; Update's immediate fetches run under it
}

// New validates and builds a Scheduler. A misconfigured scheduler is refused
// outright (error-return recovery, P10 rule 1) — B1 red-team #5: warn-and-
// continue here guaranteed a later nil-pointer panic in cycle().
func New(cfg Config) (*Scheduler, error) {
	if err := invariant.Check(cfg.Clock != nil, "scheduler requires a Clock"); err != nil {
		cfg.Clock = RealClock{}
	}
	if err := invariant.Check(cfg.Assembler != nil, "scheduler requires an Assembler"); err != nil {
		return nil, err
	}
	if err := invariant.Check(len(cfg.Tiers) > 0, "scheduler requires at least one tier"); err != nil {
		return nil, err
	}
	for _, t := range cfg.Tiers {
		if err := invariant.Check(t.Every > 0, "tier cadence must be positive"); err != nil {
			return nil, err
		}
	}
	// Copy Hints so callers cannot race tier goroutines by mutating after
	// Start (B1 red-team #9); OnPublish may be invoked concurrently from
	// multiple tiers — callers synchronize (documented on Config).
	hints := make(map[string]map[string]string, len(cfg.Hints))
	for id, h := range cfg.Hints {
		hc := make(map[string]string, len(h))
		for k, v := range h {
			hc[k] = v
		}
		hints[id] = hc
	}
	cfg.Hints = hints
	return &Scheduler{cfg: cfg, stop: make(chan struct{}), locs: cfg.Locations}, nil
}

// Start launches one goroutine per tier. Each tier fetches immediately, then
// on its cadence. Publish happens after every completed tier cycle.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
	for _, tier := range s.cfg.Tiers {
		s.wg.Add(1)
		go s.runTier(ctx, tier)
	}
}

// Rehydration (B3 UAT 59): locations a provider could not serve in a cycle
// are re-requested — alone — on a short exponential backoff (retryBase,
// 2×, 4× … capped at the tier cadence) up to maxRetries times, then the
// regular cadence resumes. A transient NWS 5xx therefore heals in seconds,
// not at the next 30-minute forecast tick.
const (
	retryBase  = 10 * time.Second
	maxRetries = 3
)

func (s *Scheduler) runTier(ctx context.Context, tier Tier) {
	defer s.wg.Done()
	for {
		if !s.fetchWithRetries(ctx, tier, s.locations()) {
			return
		}
		if !s.wait(ctx, tier.Every) {
			return
		}
	}
}

// fetchWithRetries runs one tier cycle over refs plus the rehydration
// retries; false when stopped or cancelled mid-way.
func (s *Scheduler) fetchWithRetries(ctx context.Context, tier Tier, refs []snapshot.LocationRef) bool {
	pending := s.cycle(ctx, tier, refs)
	for attempt := 0; attempt < maxRetries && len(pending) > 0; attempt++ {
		if !s.wait(ctx, min(retryBase<<attempt, tier.Every)) {
			return false
		}
		pending = s.cycle(ctx, tier, pending)
	}
	return true
}

// locations is the live location set.
func (s *Scheduler) locations() []snapshot.LocationRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locs
}

// Update replaces the location set (B3 UAT 69): the regular cadence
// continues over the new set, and ONLY the newcomers are fetched now, on
// every tier, so a looked-up location fills in seconds without the rest
// of the list re-requesting. No-op for the newcomers before Start.
func (s *Scheduler) Update(refs []snapshot.LocationRef) {
	s.mu.Lock()
	had := map[snapshot.LocationKey]bool{}
	for _, r := range s.locs {
		had[snapshot.Key(r)] = true
	}
	s.locs = refs
	ctx := s.ctx
	s.mu.Unlock()
	var added []snapshot.LocationRef
	for _, r := range refs {
		if !had[snapshot.Key(r)] {
			added = append(added, r)
		}
	}
	if len(added) == 0 || ctx == nil {
		return
	}
	select {
	case <-s.stop:
		return
	default:
	}
	for _, tier := range s.cfg.Tiers {
		s.wg.Add(1)
		go func(tier Tier) {
			defer s.wg.Done()
			s.fetchWithRetries(ctx, tier, added)
		}(tier)
	}
}

// wait sleeps d on the scheduler clock; false when stopped or cancelled.
func (s *Scheduler) wait(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-s.stop:
		return false
	case <-s.cfg.Clock.After(d):
		return true
	}
}

// cycle fans one tier out to every provider serving its kind over refs,
// publishing after each. It returns the locations left unserved by a FAILED fragment —
// the rehydration set (a provider that simply has nothing for a location,
// e.g. marine inland, reports no error and is not retried).
func (s *Scheduler) cycle(ctx context.Context, tier Tier, refs []snapshot.LocationRef) []snapshot.LocationRef {
	select {
	case <-s.stop:
		return nil
	case <-ctx.Done():
		return nil
	default:
	}
	if err := invariant.Check(len(refs) > 0, "scheduler cycle with no locations — nothing to fetch"); err != nil {
		return nil
	}
	pending := map[snapshot.LocationKey]snapshot.LocationRef{}
	for _, p := range s.cfg.Providers {
		if p == nil || !serves(p, tier.Kind) {
			continue
		}
		req := snapshot.FetchReq{Kind: tier.Kind, Locations: refs}
		if h, ok := s.cfg.Hints[p.ID()]; ok {
			req.Hint = h
		}
		frag, err := p.Fetch(ctx, req)
		if err != nil {
			// Contract violation (not a data failure): surface as a warning.
			s.cfg.Assembler.Warn(snapshot.Warning{Code: snapshot.WarnProviderError, Provider: p.ID(), Message: err.Error()})
			continue
		}
		s.cfg.Assembler.Apply(frag)
		s.publish() // per provider (UAT 64): a slow provider never holds the others' data off screen
		if frag.Err != nil {
			for _, r := range unserved(refs, frag) {
				pending[snapshot.Key(r)] = r
			}
		}
	}
	out := make([]snapshot.LocationRef, 0, len(pending))
	for _, r := range refs { // requested order, deduped
		if _, ok := pending[snapshot.Key(r)]; ok {
			out = append(out, r)
			delete(pending, snapshot.Key(r))
		}
	}
	return out
}

func (s *Scheduler) publish() {
	if s.cfg.OnPublish != nil {
		s.cfg.OnPublish()
	}
}

// unserved lists the requested refs a fragment carries no data for.
func unserved(refs []snapshot.LocationRef, frag snapshot.Fragment) []snapshot.LocationRef {
	var out []snapshot.LocationRef
	for _, r := range refs {
		if _, ok := frag.PerLocation[snapshot.Key(r)]; !ok {
			out = append(out, r)
		}
	}
	return out
}

// domainFor names the provider domain that serves a fetch kind — one
// mapping per kind, no globals (P10-06); a new kind is one line.
func domainFor(kind snapshot.FetchKind) string {
	switch kind {
	case snapshot.KindObs, snapshot.KindForecast, snapshot.KindForecastHourly:
		return "weather"
	case snapshot.KindAlerts:
		return "alerts"
	case snapshot.KindFire:
		return "fire"
	case snapshot.KindProducts:
		return "radio"
	case snapshot.KindMarine:
		return "marine"
	case snapshot.KindMarineObs:
		return "marine-obs"
	}
	return ""
}

// Serves reports whether a provider handles a fetch kind (derived from its
// declared domains — discovered, not enumerated). Exported for one-off
// hydration outside a tier (UAT 72).
func Serves(p snapshot.Provider, kind snapshot.FetchKind) bool {
	want := domainFor(kind)
	if want == "" {
		return false
	}
	for _, d := range p.Domains() {
		if d == want {
			return true
		}
	}
	return false
}

func serves(p snapshot.Provider, kind snapshot.FetchKind) bool { return Serves(p, kind) }

// Stop halts all tiers (idempotent) and waits for them to exit.
func (s *Scheduler) Stop() {
	s.once.Do(func() { close(s.stop) })
	s.wg.Wait()
}
