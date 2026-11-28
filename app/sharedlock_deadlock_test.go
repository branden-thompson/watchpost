package app

// sharedlock_deadlock_test.go — the paths that take livePipelines' mutex,
// run together, all finish.
//
// The static check (sharedlock_test.go) reads the sections; this runs them.
// commit, the bed's hooks, the fragment hook and the transmitter's re-station
// all take lp.mu, and stopAll drains the pipelines after releasing it. Run
// together, a section that takes the lock again, or a quit that waits under
// it, is a goroutine that never returns, and this test names it with every
// goroutine's stack rather than leaving it to go test's thirty-minute timeout.

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/app/internal/apptest"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// offlineProvider answers every fetch at once with nothing, so a scheduler
// cycles and its fragment hook runs without a network.
type offlineProvider struct{}

func (offlineProvider) ID() string        { return "nws" }
func (offlineProvider) Domains() []string { return []string{"weather", "alerts"} }
func (offlineProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	return snapshot.Fragment{Provider: "nws", Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}, FetchedAt: time.Now()}, nil
}

// deadlockIterations is how many times each path runs alongside the others.
const deadlockIterations = 200

// deadlockBound is how long the test waits for the next step to finish, an
// iteration of any path or the quit, before the race detector's slowdown. It
// bounds progress, not the whole run (0.19.6 A-35): the paths' own work, the
// station's pool scan and the config file's writes, is outside the lock and
// took 3.4 s for the 200 iterations on the developer's machine, while a
// deadlock stops every step at once.
const deadlockBound = 2 * time.Second

// TestCommitAndStopAllFinishUnderTheSharedLock is 0.19.6 D-16 and design
// AP-3.6. A station that follows the watchlist re-stations on every other
// commit, while the bed steps and reads its relay, the fragment hook takes the
// history store and the transmitter moves; then stopAll. A step that does not
// finish inside the bound of the last one fails the test with every
// goroutine's stack. Seen failing under the vetted mutants mDL1 (a second lock in
// commit's section) and, after the deck batch, mDL2 (the deck's
// StopProducer called in it).
func TestCommitAndStopAllFinishUnderTheSharedLock(t *testing.T) {
	lp := poolPipes(t, nil)
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	lp.idx = idx
	here := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29}
	there := snapshot.LocationRef{Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2}
	lp.station = stationArea{transmitter: here, radiusMi: 25, followsDefault: true}
	lp.setBedStations(bedPipelines(t).bedRelays()) // the resolver's shape, seeded (bedrelay_test.go)
	providers := []snapshot.Provider{offlineProvider{}}
	refs := []snapshot.LocationRef{here}
	s, err := sched.New(sched.Config{Clock: sched.RealClock{}, Assembler: newAssembler(refs, providers), Locations: refs,
		Providers: providers, Tiers: priorityTiers(), OnFragment: lp.recordFragment})
	if err != nil {
		t.Fatal(err)
	}
	s.Start(context.Background())
	lp.priority = &pipeline{asm: newAssembler(refs, providers), s: s, pub: &publisher{run: func() *snapshot.Snapshot { return nil }}}

	paths := []func(i int){
		func(i int) {
			watch := []snapshot.LocationRef{here}
			if i%2 == 1 {
				watch = []snapshot.LocationRef{there}
			}
			_ = lp.commit(watch, nil)
		},
		func(i int) {
			if cmd := lp.stepBedRelay(1); cmd != nil {
				cmd()
			}
			_, _ = lp.selectedRelay(), lp.bedCarrying()
			lp.noteBedCarrying(i%2 == 0)
		},
		func(int) {
			lp.recordFragment(snapshot.Fragment{Provider: "nws", PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}, refs)
		},
		func(i int) {
			to := here
			if i%2 == 1 {
				to = there
			}
			lp.setTransmitter(to)
		},
	}
	var wg sync.WaitGroup
	var steps atomic.Int64
	for _, path := range paths { // the four paths (P10-02)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range deadlockIterations {
				path(i)
				steps.Add(1)
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		lp.stopAll()
		close(done)
	}()
	watchStalls(t, done, &steps, deadlockBound*apptest.RaceSlowdown)
}

// watchStalls waits for done, failing with every goroutine's stack when the
// step count does not move for one bound. Bounded by the steps: each bound
// that passes without a step ends it (P10-02).
func watchStalls(t *testing.T, done <-chan struct{}, steps *atomic.Int64, bound time.Duration) {
	t.Helper()
	last := int64(-1)
	tick := time.NewTicker(bound)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case <-tick.C:
			now := steps.Load()
			if now != last {
				last = now
				continue
			}
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			t.Fatalf("no step of the shared lock's paths or stopAll finished in %v after step %d: a section is waiting on itself or on a quit that holds it.\n%s",
				bound, now, buf[:n])
		}
	}
}
