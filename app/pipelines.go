package app

// pipelines.go — the publisher (coalesced snapshots with counters) and the two pipelines: priority (one batched scheduler) and RECENT (one scheduler per location, batched alerts, staggered start). Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// publisher coalesces "new data" notifications into snapshots (UAT 74):
// a burst of provider completions — 200 at launch — becomes one snapshot
// per publishCoalesce window instead of one per completion. Snapshot() is
// the expensive step (deep copy + harmonize + sun times for 60 locations
// under the assembler lock); computing it once per window removed both
// the launch CPU spike and the 140-thread pile-up behind that lock.
type publisher struct {
	mu      sync.Mutex
	pending bool
	window  time.Duration             // the steady-state coalescing window (publishCoalesce when zero)
	run     func() *snapshot.Snapshot // takes the snapshot, delivers it, returns it

	// The launch shape (follow-up F-1): the very first publish runs at once
	// — the seed rows are on screen as soon as the loop is up (UAT 5.1) —
	// and until launchUntil the window is launchWindow, so the launch
	// burst fills the table cell by cell under the loading shimmer instead
	// of landing all at once after the steady-state window.
	launchWindow time.Duration
	launchUntil  time.Time

	// Counters (quality pass Q0, red-team R2-7): how often this pipeline
	// publishes and how many triggers the window folded — the numbers §1's
	// allocation target and the C3 threshold read. The last snapshot is kept
	// so a dump can size it without marshalling on every publish.
	count  atomic.Int64
	folded atomic.Int64
	last   atomic.Pointer[snapshot.Snapshot]
}

// publishCoalesce is the priority pipeline's window: short enough that
// rows still fill "as they land", long enough to fold a burst.
const publishCoalesce = 50 * time.Millisecond

// recentPublishCoalesce is the RECENT pipeline's window (quality pass Q3,
// plan §2.4 "one publish per tier tick", PF-9): the fifty schedulers'
// tier ticks land as a wave — starts 10 ms apart, the list's fetches
// paced at 30/s, so a wave spans a few seconds — and the 50 ms window
// published ~47 times per wave (Q1 soak: 44 → 91 across one 10-minute
// tick). Five seconds folds a wave into one or two snapshots; a seed row
// still fills within five seconds of its fetch landing.
const recentPublishCoalesce = 5 * time.Second

// The RECENT launch shape (follow-up F-1, HUM LEAD 2026-08-27): for the
// first recentLaunchPhase after start the window is recentLaunchWindow, so
// the seeded rows rehydrate as their fetches land (the launch burst is
// fifty schedulers started 10 ms apart, paced at 30 requests/s: it lands
// over ~30–60 s); the 5 s window takes over for the steady-state waves.
const (
	recentLaunchWindow = 250 * time.Millisecond
	recentLaunchPhase  = 90 * time.Second
)

// Trigger schedules a publish; further triggers inside the window fold in.
// The first publish ever runs immediately (no window: the seed rows).
func (pb *publisher) Trigger() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if pb.pending {
		pb.folded.Add(1)
		return
	}
	pb.pending = true
	window := pb.windowNow()
	if pb.count.Load() == 0 && pb.folded.Load() == 0 {
		window = 0 // the first publish: at once
	}
	time.AfterFunc(window, pb.fire)
}

// fire runs one publish and clears the pending flag.
func (pb *publisher) fire() {
	pb.mu.Lock()
	pb.pending = false
	pb.mu.Unlock()
	if snap := pb.run(); snap != nil {
		pb.last.Store(snap)
	}
	pb.count.Add(1)
}

// windowNow is the window in force: the launch window until launchUntil,
// then the steady-state one (caller holds mu).
func (pb *publisher) windowNow() time.Duration {
	if pb.launchWindow > 0 && time.Now().Before(pb.launchUntil) {
		return pb.launchWindow
	}
	if pb.window == 0 {
		return publishCoalesce
	}
	return pb.window
}

// stats is the counters' point-in-time copy.
func (pb *publisher) stats() tty.PipelineStats {
	return tty.PipelineStats{Publishes: pb.count.Load(), Folded: pb.folded.Load()}
}

// lastSnapshot is the most recent published snapshot (nil before the first).
func (pb *publisher) lastSnapshot() *snapshot.Snapshot { return pb.last.Load() }

// The cadence tables — the one owner (L3-F14). TestCadenceTableIsTheDoc
// prints them as the markdown table architecture.md §11.1 carries, so the
// document is generated from these, never re-typed. Every cadence change
// needs a §0.3 freshness row in the batch's build log.

// priorityTiers is the favourites' pipeline: one batched scheduler.
func priorityTiers() []sched.Tier {
	return []sched.Tier{
		{Kind: snapshot.KindAlerts, Every: 20 * time.Second},
		{Kind: snapshot.KindObs, Every: 90 * time.Second},
		{Kind: snapshot.KindMarineObs, Every: 10 * time.Minute}, // buoys + gauges: NDBC files turn over every 30 min (UAT 72)
		{Kind: snapshot.KindForecast, Every: 30 * time.Minute},
		{Kind: snapshot.KindForecastHourly, Every: 30 * time.Minute}, // fires with the daily tier (UAT 72)
		{Kind: snapshot.KindMarine, Every: 30 * time.Minute},         // coastal waters (UAT 29); one gridpoint download serves it and the daily fill
		{Kind: snapshot.KindFire, Every: 10 * time.Minute},           // HMS archive refresh + FIRMS cadence (B5, AI-3)
		{Kind: snapshot.KindSeismic, Every: 5 * time.Minute},         // USGS max-age=60; "did my area shake" needs no sub-minute latency — the fire cadence, regional like it (seismic P2 §0.3)
	}
}

// recentTiers is one RECENT location's scheduler: no alerts (batched
// across the list at recentAlertsEvery) and no hourly (hydrated on demand) — UAT 72.
func recentTiers() []sched.Tier {
	return []sched.Tier{
		{Kind: snapshot.KindObs, Every: 10 * time.Minute},
		{Kind: snapshot.KindMarineObs, Every: 10 * time.Minute},
		{Kind: snapshot.KindForecast, Every: time.Hour},
		{Kind: snapshot.KindMarine, Every: time.Hour},
		{Kind: snapshot.KindFire, Every: 15 * time.Minute},    // the archive is shared through the client cache (B5)
		{Kind: snapshot.KindSeismic, Every: 15 * time.Minute}, // the regional box is shared through the client cache; RECENT half the priority cadence (seismic P2 §0.3)
	}
}

// pipeline is the priority pipeline: one assembler, one batched scheduler.
type pipeline struct {
	asm *snapshot.Assembler
	s   *sched.Scheduler
	pub *publisher
}

// update reconciles the watchlist in place (UAT 69): kept favourites keep
// their data, newcomers fetch at once, and the new order publishes now.
func (pl *pipeline) update(refs []snapshot.LocationRef) {
	pl.asm.SetLocations(refs)
	pl.s.Update(refs)
	pl.pub.Trigger()
}

func (pl *pipeline) stop() { pl.s.Stop() }

// startPriority wires the priority pipeline (fast cadences). onPublish (may
// be nil) observes each publish before it is sent — the M1 timer rides it.
func startPriority(ctx context.Context, p *tea.Program, providers []snapshot.Provider, refs []snapshot.LocationRef, onPublish func(*snapshot.Snapshot)) *pipeline {
	asm := newAssembler(refs, providers)
	pub := &publisher{run: func() *snapshot.Snapshot {
		snap := asm.Snapshot()
		dropSuperseded(snap) // an alert's own update replaces it in the tables too (R5-A-08)
		if onPublish != nil {
			onPublish(snap)
		}
		p.Send(tty.SnapshotMsg{Snap: snap})
		return snap
	}}
	s, err := sched.New(sched.Config{
		Clock: sched.RealClock{}, Assembler: asm, Locations: refs,
		Providers: providers,
		Tiers:     priorityTiers(),
		OnPublish: pub.Trigger,
	})
	if err != nil {
		_ = invariant.Check(false, "priority scheduler misconfigured: "+err.Error())
		return nil
	}
	s.Start(ctx)
	return &pipeline{asm: asm, s: s, pub: pub}
}

// recentStartDelay holds the seed pipeline back just long enough for the
// priority pipeline to own the first second (M1 warm budget).
const recentStartDelay = time.Second

// recentPipeline feeds the RECENT/SEARCHED list: one shared assembler,
// one scheduler per location so each row publishes the moment its own
// fetches land (UAT 5.1); the launch burst is ~5 requests per location
// once (a disk cache for warm launches is the queued "NWS cache refresh").
type recentPipeline struct {
	ctx       context.Context
	asm       *snapshot.Assembler
	providers []snapshot.Provider
	alerts    *sched.Scheduler // ONE batched alerts call for the whole list (UAT 72)
	newFor    func(ref snapshot.LocationRef) *sched.Scheduler
	publish   func()
	pub       *publisher // the coalescer behind publish (counters; nil when the list is empty)

	mu      sync.Mutex // guards scheds and started (red-team 0.9.0 C-6: the staggered starter and a commit could touch the map together)
	scheds  map[snapshot.LocationKey]*sched.Scheduler
	started bool // the staggered start has run: newcomers start themselves from now on
}

// recentAlertsEvery is the RECENT list's alert cadence: one batched
// /alerts/active call covering every recent zone (was 50 per-location
// calls every 2 minutes — 25 of the app's ~40 requests per minute, UAT 72).
const recentAlertsEvery = 2 * time.Minute

// update reconciles the list in place (UAT 69): removed locations stop
// their schedulers, newcomers get one (their first cycle fetches at once),
// kept rows keep their data, and the new order publishes immediately.
func (rp *recentPipeline) update(refs []snapshot.LocationRef) {
	if rp.asm == nil {
		return // nothing was seeded; the list is a nicety
	}
	added, removed := rp.asm.SetLocations(refs)
	rp.mu.Lock()
	for _, r := range removed {
		if s := rp.scheds[snapshot.Key(r)]; s != nil {
			go s.Stop() // Stop waits for in-flight fetches; never on the commit path
			delete(rp.scheds, snapshot.Key(r))
		}
	}
	for _, r := range added {
		if s := rp.newFor(r); s != nil {
			rp.scheds[snapshot.Key(r)] = s
			if rp.started {
				s.Start(rp.ctx) // before that, the staggered starter picks it up
			}
		}
		go rp.hydrateHourly(r) // a looked-up location is being looked at: it earns its hourly now
	}
	rp.mu.Unlock()
	if rp.alerts != nil {
		rp.alerts.Update(refs) // newcomers' alerts fetch at once; the batch continues on cadence
	}
	rp.publish()
}

// hydrateHourly fetches the hourly forecast for one RECENT location on
// demand (UAT 72) and publishes; safe to call from any goroutine.
func (rp *recentPipeline) hydrateHourly(ref snapshot.LocationRef) {
	if rp.asm == nil {
		return
	}
	for _, pr := range rp.providers {
		if !sched.Serves(pr, snapshot.KindForecastHourly) {
			continue
		}
		if frag, err := pr.Fetch(rp.ctx, snapshot.FetchReq{Kind: snapshot.KindForecastHourly, Locations: []snapshot.LocationRef{ref}}); err == nil {
			rp.asm.Apply(frag)
		}
	}
	rp.publish()
}

func (rp *recentPipeline) stop() {
	rp.mu.Lock()
	scheds := make([]*sched.Scheduler, 0, len(rp.scheds))
	for _, s := range rp.scheds {
		scheds = append(scheds, s)
	}
	rp.mu.Unlock() // a commit still in flight at quit must not race the map (round 2 N-6)
	for _, s := range scheds {
		s.Stop()
	}
	if rp.alerts != nil {
		rp.alerts.Stop()
	}
}

// startRecent wires the slow-cadence background pipeline that feeds live
// weather to the seeded RECENT/SEARCHED list. The seed snapshot publishes
// once the program loop is up (names/zips render instantly; temps stream
// in). An empty seed list yields an inert pipeline.
func startRecent(ctx context.Context, p *tea.Program, providers []snapshot.Provider, refs []snapshot.LocationRef, onPublish func(*snapshot.Snapshot)) *recentPipeline {
	rp := &recentPipeline{ctx: ctx, providers: providers, scheds: map[snapshot.LocationKey]*sched.Scheduler{}}
	if len(refs) == 0 {
		rp.publish = func() {}
		return rp
	}
	rp.asm = newAssembler(refs, providers)
	pub := &publisher{window: recentPublishCoalesce, launchWindow: recentLaunchWindow, launchUntil: time.Now().Add(recentLaunchPhase), run: func() *snapshot.Snapshot {
		// Always publish the SHARED assembler's view: every scheduler's
		// progress lands in one snapshot regardless of which one cycled.
		// Snapshot() deep-copies, so the delivered value is the receiver's
		// own — the tty sorts its alerts in place (L3-F16).
		snap := rp.asm.Snapshot()
		dropSuperseded(snap) // R5-A-08
		if onPublish != nil {
			onPublish(snap) // 0.13.0: the severe deck's Trigger rides every publish
		}
		p.Send(tty.RecentSnapshotMsg{Snap: snap})
		return snap
	}}
	rp.pub = pub
	rp.publish = pub.Trigger
	rp.newFor = func(ref snapshot.LocationRef) *sched.Scheduler {
		s, err := sched.New(sched.Config{
			Clock: sched.RealClock{}, Assembler: rp.asm, Locations: []snapshot.LocationRef{ref},
			Providers: providers,
			Tiers:     recentTiers(),
			OnPublish: rp.publish,
		})
		if err != nil {
			// A misassembled seed pipeline never blocks the dashboard: surface
			// the wiring bug loudly in dev, drop the nicety in release.
			_ = invariant.Check(false, "recent seed scheduler misconfigured: "+err.Error())
			return nil
		}
		return s
	}
	for _, ref := range refs {
		if s := rp.newFor(ref); s != nil {
			rp.scheds[snapshot.Key(ref)] = s
		}
	}
	if s, err := sched.New(sched.Config{Clock: sched.RealClock{}, Assembler: rp.asm, Locations: refs, Providers: providers,
		Tiers: []sched.Tier{{Kind: snapshot.KindAlerts, Every: recentAlertsEvery}}, OnPublish: rp.publish}); err == nil {
		rp.alerts = s
	} else {
		_ = invariant.Check(false, "recent alerts scheduler misconfigured: "+err.Error())
	}
	go rp.startStaggered(ctx)
	return rp
}

// startStaggered publishes the seed snapshot once the program loop is up
// (p.Send blocks until Run starts — never call it pre-Run on the main
// goroutine, caught by the cmd test hang), waits recentStartDelay, then
// starts the schedulers recentStartStagger apart (UAT 74: 50 schedulers
// starting in the same instant made a 200-goroutine burst that cost ~90
// OS threads; 10 ms apart spreads the launch over half a second).
func (rp *recentPipeline) startStaggered(ctx context.Context) {
	rp.publish()
	select {
	case <-ctx.Done():
		return
	case <-time.After(recentStartDelay):
	}
	rp.mu.Lock()
	toStart := make([]*sched.Scheduler, 0, len(rp.scheds))
	for _, s := range rp.scheds {
		toStart = append(toStart, s)
	}
	rp.started = true // from here a commit's newcomers start themselves
	rp.mu.Unlock()
	for _, s := range toStart {
		s.Start(ctx)
		time.Sleep(recentStartStagger)
	}
	if rp.alerts != nil {
		rp.alerts.Start(ctx)
	}
}

// recentStartStagger spaces the recent schedulers' starts.
const recentStartStagger = 10 * time.Millisecond
