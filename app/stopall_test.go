package app

// stopall_test.go — quitting waits for a fetch that is still landing.
//
// A priority fetch that lands records itself in the history, and the history
// store is read under lp.mu. stopAll waits for the pipelines' fetches, so it
// must not hold lp.mu while it waits: if it does, the landing fetch and the
// quit wait for each other and q leaves the station running. The PTY journey
// (cmd/watchpost/journey_test.go) meets this on the real binary; this pins it
// in the package, with the fetch held in flight on purpose.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// gatedProvider holds every fetch until release closes, and says when the
// first one is in flight.
type gatedProvider struct {
	fetching, release chan struct{}
	once              sync.Once
}

func (g *gatedProvider) ID() string        { return "nws" }
func (g *gatedProvider) Domains() []string { return []string{"weather", "alerts"} }

func (g *gatedProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	g.once.Do(func() { close(g.fetching) })
	<-g.release
	return snapshot.Fragment{Provider: "nws", Kind: req.Kind, PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}, FetchedAt: time.Now()}, nil
}

func TestStopAllWaitsForAFetchThatRecordsItself(t *testing.T) {
	lp := &livePipelines{recent: &recentPipeline{}}
	g := &gatedProvider{fetching: make(chan struct{}), release: make(chan struct{})}
	refs := []snapshot.LocationRef{{Label: "Oak Ridge, TN", Zip: "37830", Lat: 36.0104, Lon: -84.2696}}
	providers := []snapshot.Provider{g}
	s, err := sched.New(sched.Config{Clock: sched.RealClock{}, Assembler: newAssembler(refs, providers), Locations: refs,
		Providers: providers, Tiers: priorityTiers(), OnFragment: lp.recordFragment})
	if err != nil {
		t.Fatal(err)
	}
	s.Start(context.Background())
	lp.priority = &pipeline{s: s}
	select {
	case <-g.fetching:
	case <-time.After(10 * time.Second):
		t.Fatal("the priority pipeline never fetched")
	}

	stopped := make(chan struct{})
	go func() { lp.stopAll(); close(stopped) }()
	// The fetch lands once stopAll holds lp.mu, if it holds it at all: a
	// stopAll that waits under the lock is seen holding it here.
	for range 100 {
		if !lp.mu.TryLock() {
			break
		}
		lp.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	close(g.release)
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("stopAll never returned: it waits for the fetch under lp.mu, and the fetch needs lp.mu to record what it fetched")
	}
}
