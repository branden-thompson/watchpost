package sched

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: architecture §3 + §10.8 — tiered cadence through one scheduler; Clock
// interface so the replay harness (M2/M3) can drive time; publish-on-change;
// tier cadence is THE freshness authority.

// fakeClock drives time deterministically.
// fakeClock drives the scheduler's time. IT SYNCHRONISES; IT DOES NOT GUESS
// (F-29, 2026-09-06).
//
// Advance used to fire the due waiters and then `time.Sleep(5 * time.Millisecond)`
// — "give the scheduler goroutines a beat to run their fetch cycle". That beat
// is a WALL-CLOCK GUESS against work of unbounded cost, and under `-race` on a
// busy machine the guess is sometimes wrong: the caller then asserts before the
// work it is waiting for has happened, and the test fails for a reason that has
// nothing to do with the grid arithmetic it is named for. A gate that fails for
// unrelated reasons teaches people to re-run rather than read it.
//
// The synchronisation point is `armed`: a tier loop's only exit from a cycle is
// back into Clock.After (runTier's wait, and fetchWithRetries' retry wait), so
// "every goroutine this Advance woke has registered again" means every one of
// them is quiescent — blocked on the clock, with nothing more to do until time
// moves. That is exactly the state the caller is about to assert against.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
	// rearmed is closed and replaced on every registration, so a caller can
	// wait for the set to grow without polling.
	rearmed chan struct{}
}
type waiter struct {
	at time.Time
	ch chan time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start, rearmed: make(chan struct{})}
}

// wedgeBound is how long Advance and awaitArmed will wait for the scheduler to
// come back round before giving up.
//
// IT IS A FAILURE BOUND, NOT A SUCCESS BOUND, and that is the whole difference
// from the sleep it replaces. The old 5 ms was a guess that the work was DONE;
// this is a limit on how long a genuinely wedged scheduler may hang the suite.
// Reaching it means something is actually broken, so it can be generous.
const wedgeBound = 10 * time.Second

// awaitWaiters blocks until at least n waiters are pending on the clock — n
// quiescent tiers. It is also how to wait for the INITIAL fetches, which no
// Advance triggers: a tier arms for the first time only after its first cycle.
func (c *fakeClock) awaitWaiters(n int) {
	deadline := time.After(wedgeBound)
	for {
		c.mu.Lock()
		got, changed := len(c.waiters), c.rearmed
		c.mu.Unlock()
		if got >= n {
			return
		}
		select {
		case <-changed:
		case <-deadline:
			return // wedged: let the caller's own assertion say so
		}
	}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, waiter{at: c.now.Add(d), ch: ch})
	// REGISTERING IS THE QUIESCENT SIGNAL. A tier that reaches here has finished
	// whatever it woke to do and cannot proceed until time moves again.
	close(c.rearmed)
	c.rearmed = make(chan struct{})
	return ch
}

// Advance moves time forward, firing due waiters.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	var due []waiter
	var rest []waiter
	for _, w := range c.waiters {
		if !w.at.After(now) {
			due = append(due, w)
		} else {
			rest = append(rest, w)
		}
	}
	c.waiters = rest
	// Every tier that was quiescent before must be quiescent again.
	target := len(rest) + len(due)
	c.mu.Unlock()
	for _, w := range due {
		w.ch <- now
	}
	if len(due) == 0 {
		return // nothing was woken: there is nothing to wait for
	}
	c.awaitWaiters(target)
}

// countingProvider records fetches per kind.
type countingProvider struct {
	id       string
	obs      atomic.Int32
	alerts   atomic.Int32
	forecast atomic.Int32
}

func (p *countingProvider) ID() string        { return p.id }
func (p *countingProvider) Domains() []string { return []string{"weather", "alerts"} }
func (p *countingProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	switch req.Kind {
	case snapshot.KindObs:
		p.obs.Add(1)
	case snapshot.KindAlerts:
		p.alerts.Add(1)
	case snapshot.KindForecast:
		p.forecast.Add(1)
	}
	return snapshot.Fragment{Provider: p.id, Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}, nil
}

var refs = []snapshot.LocationRef{{Label: "A", Lat: 33.24, Lon: -117.29}}

func newTestSched(clk Clock, p snapshot.Provider) (*Scheduler, *snapshot.Assembler, *atomic.Int32) {
	asm := snapshot.NewAssembler(refs, []string{p.ID()})
	var published atomic.Int32
	s, err := New(Config{
		Clock:     clk,
		Assembler: asm,
		Locations: refs,
		Providers: []snapshot.Provider{p},
		Tiers: []Tier{
			{Kind: snapshot.KindAlerts, Every: 20 * time.Second},
			{Kind: snapshot.KindObs, Every: 60 * time.Second},
		},
		OnPublish: func() { published.Add(1) },
	})
	if err != nil {
		panic(err) // test wiring
	}
	return s, asm, &published
}

func TestForecastTierFires(t *testing.T) {
	// B1 red-team B-1: tier 3 (forecast) was never exercised.
	clk := newFakeClock(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC))
	p := &countingProvider{id: "nws"}
	asm := snapshot.NewAssembler(refs, []string{"nws"})
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: refs,
		Providers: []snapshot.Provider{p},
		Tiers:     []Tier{{Kind: snapshot.KindForecast, Every: 30 * time.Minute}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return p.forecast.Load() == 1 }, "initial forecast fetch")
	clk.Advance(30 * time.Minute)
	waitFor(t, func() bool { return p.forecast.Load() == 2 }, "forecast tier at +30m")
}

func TestNewRefusesMisconfiguration(t *testing.T) {
	if _, err := New(Config{Clock: RealClock{}}); err == nil {
		t.Fatal("nil assembler must be refused, not deferred to a cycle panic (B1 #5)")
	}
}

func TestTierCadence(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC))
	p := &countingProvider{id: "nws"}
	s, _, _ := newTestSched(clk, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	waitFor(t, func() bool { return p.alerts.Load() == 1 && p.obs.Load() == 1 }, "initial fetches")
	clk.Advance(20 * time.Second)
	waitFor(t, func() bool { return p.alerts.Load() == 2 }, "alert tier at +20s")
	if p.obs.Load() != 1 {
		t.Fatalf("obs must not fire at +20s, got %d", p.obs.Load())
	}
	clk.Advance(20 * time.Second)
	clk.Advance(20 * time.Second)
	waitFor(t, func() bool { return p.alerts.Load() == 4 && p.obs.Load() == 2 }, "obs tier at +60s")
}

func TestPublishesAfterEachCycle(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC))
	p := &countingProvider{id: "nws"}
	s, _, published := newTestSched(clk, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return published.Load() >= 2 }, "initial publishes")
	before := published.Load()
	clk.Advance(20 * time.Second)
	waitFor(t, func() bool { return published.Load() > before }, "publish after alert tier")
}

func TestStopIsIdempotentAndHalts(t *testing.T) {
	clk := newFakeClock(time.Now())
	p := &countingProvider{id: "nws"}
	s, _, _ := newTestSched(clk, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	waitFor(t, func() bool { return p.alerts.Load() >= 1 }, "started")
	s.Stop()
	s.Stop() // idempotent
	n := p.alerts.Load()
	clk.Advance(time.Minute)
	time.Sleep(10 * time.Millisecond)
	if p.alerts.Load() != n {
		t.Fatal("no fetches after Stop")
	}
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func TestServesMarineKind(t *testing.T) {
	// B3 UAT 29: providers declaring the "marine" domain receive KindMarine.
	if !serves(domainsOnly{"marine"}, snapshot.KindMarine) || serves(domainsOnly{"weather"}, snapshot.KindMarine) {
		t.Fatal("marine domain must map to KindMarine exclusively")
	}
	// UAT 72: observations are their own kind/domain; hourly is a weather kind.
	if !serves(domainsOnly{"marine-obs"}, snapshot.KindMarineObs) || serves(domainsOnly{"marine"}, snapshot.KindMarineObs) || serves(domainsOnly{"marine-obs"}, snapshot.KindMarine) {
		t.Fatal("marine-obs domain must map to KindMarineObs exclusively")
	}
	if !serves(domainsOnly{"weather"}, snapshot.KindForecastHourly) {
		t.Fatal("weather domain serves the hourly forecast kind")
	}
}

// domainsOnly is a minimal Provider for the serves() check.
type domainsOnly []string

func (d domainsOnly) ID() string        { return "x" }
func (d domainsOnly) Domains() []string { return d }
func (d domainsOnly) Fetch(context.Context, snapshot.FetchReq) (snapshot.Fragment, error) {
	return snapshot.Fragment{}, nil
}

// flakyProvider fails one location on its first obs fetch and records the
// locations of every request (UAT 59 rehydration).
type flakyProvider struct {
	mu    sync.Mutex
	calls [][]snapshot.LocationRef
}

func (p *flakyProvider) ID() string        { return "nws" }
func (p *flakyProvider) Domains() []string { return []string{"weather"} }
func (p *flakyProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, append([]snapshot.LocationRef(nil), req.Locations...))
	frag := snapshot.Fragment{Provider: "nws", Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	for _, r := range req.Locations {
		if r.Label == "B" && len(p.calls) == 1 {
			frag.Err = assertErr("B: HTTP 500")
			continue
		}
		frag.PerLocation[snapshot.Key(r)] = snapshot.PartialData{}
	}
	return frag, nil
}

func (p *flakyProvider) snapshotCalls() [][]snapshot.LocationRef {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([][]snapshot.LocationRef(nil), p.calls...)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func TestFailedLocationsRetryBeforeCadence(t *testing.T) {
	// UAT 59: a location a provider could not serve is re-requested on a
	// short backoff — alone — instead of waiting out the tier cadence.
	two := []snapshot.LocationRef{{Label: "A", Lat: 33.24, Lon: -117.29}, {Label: "B", Lat: 32.72, Lon: -117.16}}
	clk := newFakeClock(time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	p := &flakyProvider{}
	asm := snapshot.NewAssembler(two, []string{"nws"})
	var published atomic.Int32
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: two, Providers: []snapshot.Provider{p},
		Tiers:     []Tier{{Kind: snapshot.KindObs, Every: 10 * time.Minute}},
		OnPublish: func() { published.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 1 }, "initial fetch")
	clk.Advance(retryBase)
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 2 }, "retry after the backoff")
	calls := p.snapshotCalls()
	if len(calls[1]) != 1 || calls[1][0].Label != "B" {
		t.Fatalf("retry must re-request only the failed location, got %v", calls[1])
	}
	if published.Load() < 2 {
		t.Fatal("the retry cycle must publish so the row fills")
	}
	clk.Advance(retryBase * 2)
	time.Sleep(10 * time.Millisecond)
	if n := len(p.snapshotCalls()); n != 2 {
		t.Fatalf("nothing pending — no further retries, got %d calls", n)
	}
}

func TestRetriesAreBounded(t *testing.T) {
	// A location that keeps failing gets maxRetries attempts, then waits for
	// the cadence (never a hot loop).
	one := []snapshot.LocationRef{{Label: "B", Lat: 32.72, Lon: -117.16}}
	clk := newFakeClock(time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	p := &alwaysFailing{}
	asm := snapshot.NewAssembler(one, []string{"nws"})
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: one, Providers: []snapshot.Provider{p},
		Tiers: []Tier{{Kind: snapshot.KindObs, Every: time.Hour}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return p.n.Load() == 1 }, "initial fetch")
	for i := range maxRetries {
		clk.Advance(retryBase << i)
		waitFor(t, func() bool { return p.n.Load() == int32(i+2) }, "bounded retry")
	}
	clk.Advance(30 * time.Minute)
	time.Sleep(10 * time.Millisecond)
	if p.n.Load() != int32(maxRetries+1) {
		t.Fatalf("retries must stop at maxRetries until the cadence, got %d calls", p.n.Load())
	}
}

type alwaysFailing struct{ n atomic.Int32 }

func (p *alwaysFailing) ID() string        { return "nws" }
func (p *alwaysFailing) Domains() []string { return []string{"weather"} }
func (p *alwaysFailing) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	p.n.Add(1)
	return snapshot.Fragment{Provider: "nws", Kind: req.Kind, Err: assertErr("down"),
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}, nil
}

// slowProvider blocks Fetch until released (UAT 64).
type slowProvider struct{ release chan struct{} }

func (p *slowProvider) ID() string        { return "slow" }
func (p *slowProvider) Domains() []string { return []string{"weather"} }
func (p *slowProvider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	select {
	case <-p.release:
	case <-ctx.Done():
	}
	return snapshot.Fragment{Provider: "slow", Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}, nil
}

func TestPublishesPerProviderNotPerTier(t *testing.T) {
	// UAT 64: the priority marine tier sat blank while CO-OPS waited on the
	// shared pacing bucket — NDBC's data must reach the screen as soon as
	// its fragment applies, not when the slowest provider finishes.
	clk := newFakeClock(time.Now())
	fast := &countingProvider{id: "nws"}
	slow := &slowProvider{release: make(chan struct{})}
	asm := snapshot.NewAssembler(refs, []string{"nws", "slow"})
	var published atomic.Int32
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: refs, Providers: []snapshot.Provider{fast, slow},
		Tiers: []Tier{{Kind: snapshot.KindObs, Every: time.Hour}}, OnPublish: func() { published.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	waitFor(t, func() bool { return published.Load() >= 1 }, "publish after the fast provider while the slow one is still fetching")
	close(slow.release)
	waitFor(t, func() bool { return published.Load() >= 2 }, "publish after the slow provider")
	s.Stop()
}

// recordingProvider records every request's locations per kind.
type recordingProvider struct {
	mu    sync.Mutex
	calls []snapshot.FetchReq
}

func (p *recordingProvider) ID() string        { return "nws" }
func (p *recordingProvider) Domains() []string { return []string{"weather", "alerts"} }
func (p *recordingProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, req)
	frag := snapshot.Fragment{Provider: "nws", Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	for _, r := range req.Locations {
		frag.PerLocation[snapshot.Key(r)] = snapshot.PartialData{}
	}
	return frag, nil
}

func (p *recordingProvider) snapshotCalls() []snapshot.FetchReq {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]snapshot.FetchReq(nil), p.calls...)
}

func TestUpdateFetchesOnlyTheNewcomersNow(t *testing.T) {
	// UAT 69: a lookup adds one location — it is fetched at once on every
	// tier, alone; the kept locations wait for their regular cadence.
	a := snapshot.LocationRef{Label: "A", Lat: 33.24, Lon: -117.29}
	b := snapshot.LocationRef{Label: "B", Lat: 32.72, Lon: -117.16}
	clk := newFakeClock(time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	p := &recordingProvider{}
	asm := snapshot.NewAssembler([]snapshot.LocationRef{a}, []string{"nws"})
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: []snapshot.LocationRef{a}, Providers: []snapshot.Provider{p},
		Tiers: []Tier{{Kind: snapshot.KindAlerts, Every: time.Minute}, {Kind: snapshot.KindObs, Every: time.Hour}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 2 }, "initial fetch on both tiers")
	asm.SetLocations([]snapshot.LocationRef{b, a})
	s.Update([]snapshot.LocationRef{b, a})
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 4 }, "newcomer fetched on both tiers")
	for _, req := range p.snapshotCalls()[2:] {
		if len(req.Locations) != 1 || req.Locations[0].Label != "B" {
			t.Fatalf("immediate fetch must cover only the newcomer, got %v", req.Locations)
		}
	}
	clk.Advance(time.Minute)
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 5 }, "alerts cadence over the whole set")
	if got := p.snapshotCalls()[4].Locations; len(got) != 2 || got[0].Label != "B" {
		t.Fatalf("cadence must cover the updated set in order, got %v", got)
	}
	s.Update([]snapshot.LocationRef{a}) // pure removal: nothing to fetch
	time.Sleep(10 * time.Millisecond)
	if len(p.snapshotCalls()) != 5 {
		t.Fatal("a removal must not fetch anything")
	}
}

// clockProvider spends fetchTime of fake-clock time in every alerts Fetch
// (the one tier the test watches; the other tier's fetch stays instant so
// the clock moves only where the test can account for it).
type clockProvider struct {
	countingProvider
	clk       *fakeClock
	fetchTime time.Duration
}

func (p *clockProvider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	if req.Kind == snapshot.KindAlerts {
		p.clk.Advance(p.fetchTime)
	}
	return p.countingProvider.Fetch(ctx, req)
}

// Quality pass Q3 (PF-9): the cadence is a fixed grid from Start — a
// tier's own fetch time does not push its phase later each cycle, so fifty
// schedulers started together stay together.
func TestTierCadenceIsAFixedGrid(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC))
	p := &clockProvider{countingProvider: countingProvider{id: "nws"}, clk: clk, fetchTime: 3 * time.Second}
	s, _, _ := newTestSched(clk, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	// NOT waitFor (F-29). Both bounds in this test were 2 s of REAL time against
	// work of unbounded cost, so under `-race` on a busy machine the assertion
	// could fire before the scheduler had acted — and the test then failed for a
	// reason unrelated to the grid arithmetic it is named for. Both tiers arm the
	// clock only after their first cycle, so two pending waiters IS "the initial
	// fetches are done"; the clock waits on that, and the counters are then
	// asserted directly rather than polled towards.
	clk.awaitWaiters(2)
	if p.alerts.Load() != 1 || p.obs.Load() != 1 {
		t.Fatalf("initial fetches: alerts=%d obs=%d, want 1 and 1", p.alerts.Load(), p.obs.Load())
	}
	// The initial alerts fetch cost 3 s of clock; the tier's next slot is
	// still start+20 s — 17 s away, not 20 (the pre-Q3 loop fired at +23).
	clk.Advance(16 * time.Second)
	if p.alerts.Load() != 1 {
		t.Fatalf("alerts must not fire before its grid point, got %d", p.alerts.Load())
	}
	clk.Advance(2 * time.Second) // +21 s: past the grid point, short of the drifted one
	// DIRECT, because Advance now returns only once every tier it woke is
	// quiescent again. If this ever needs a wait, the clock has stopped
	// synchronising and that is the bug to fix — not the bound to raise.
	if got := p.alerts.Load(); got != 2 {
		t.Fatalf("the alert tier fires on the +20 s grid point despite 3 s of fetch time: alerts=%d, want 2", got)
	}
}

// F-29 — ADVANCE RETURNS ONLY ONCE THE SCHEDULER HAS ACTED.
//
// This is the property the cadence test rests on, pinned on its own so that
// losing it fails HERE, with a message about the clock, rather than surfacing as
// an occasional unexplained timeout in a test about grid arithmetic.
//
// The old Advance fired the due waiters and slept 5 ms. That is a guess, and no
// number of passing runs makes it not one; this asserts with no wait at all.
func TestAdvanceReturnsOnlyOnceTheTierHasActed(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC))
	p := &clockProvider{countingProvider: countingProvider{id: "nws"}, clk: clk, fetchTime: 3 * time.Second}
	s, _, _ := newTestSched(clk, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	clk.awaitWaiters(2)
	before := p.alerts.Load()
	// Past the alerts tier's next grid point in one step.
	clk.Advance(20 * time.Second)
	if got := p.alerts.Load(); got != before+1 {
		t.Fatalf("Advance returned before the tier it woke had fetched: alerts=%d, want %d — "+
			"the clock is guessing again, and every assertion after an Advance is racing it", got, before+1)
	}
}

// THE SCHEDULER RECORDS WHICH LOCATIONS IT ASKED ABOUT (#13, red team 2026-09-08).
//
// This is the dashboard's ONLY refresh path, and it was the one call site that
// did not pass the asked set — so no location ever recorded an attempt, and a
// row the feed cannot serve shimmered for ever. The fix ran only in
// `watchpost report`, which is not a surface a listener can see.
//
// IT IS TESTED HERE, THROUGH THE REAL SCHEDULER, and that is the point. The
// assembler's own test hand-built the fragment and set the asked set itself, so
// it proved the assembler works and could say nothing about whether anybody
// calls it that way. A test that supplies the producer's output cannot detect a
// missing producer.
func TestTheSchedulerRecordsTheLocationsItAskedAbout(t *testing.T) {
	two := []snapshot.LocationRef{{Label: "A", Lat: 33.24, Lon: -117.29}, {Label: "B", Lat: 32.72, Lon: -117.16}}
	clk := newFakeClock(time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	p := &flakyProvider{} // serves A, fails B — the issue #13 shape exactly
	asm := snapshot.NewAssembler(two, []string{"nws"})
	asm.SetAttribution("nws", "reference", "NWS")
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: two, Providers: []snapshot.Provider{p},
		Tiers: []Tier{{Kind: snapshot.KindObs, Every: 10 * time.Minute}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	waitFor(t, func() bool { return len(p.snapshotCalls()) == 1 }, "initial fetch")

	waitFor(t, func() bool {
		for _, l := range asm.Snapshot().Locations {
			if l.Label == "B" && !l.WeatherAsOf.IsZero() {
				return true
			}
		}
		return false
	}, "B — which the provider could NOT serve — must still record that it was asked about; "+
		"without it the row shimmers for ever and the listener is told the data is still coming")

	for _, l := range asm.Snapshot().Locations {
		if l.WeatherAsOf.IsZero() {
			t.Errorf("%s was in the fetch and recorded no attempt", l.Label)
		}
	}
}
