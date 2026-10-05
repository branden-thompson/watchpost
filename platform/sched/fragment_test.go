package sched

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE SCHEDULER TELLS WHAT IT APPLIED (W22.2): each fetch it applies reaches
// the observer with the locations it was asked for, after the assembler has
// it - the dashboard's own refreshes, so a recorder fetches nothing of its own.
func TestTheSchedulerTellsWhatItApplied(t *testing.T) {
	clk := newFakeClock(time.Now())
	p := &recordingProvider{}
	asm := snapshot.NewAssembler(refs, []string{p.ID()})
	var mu sync.Mutex
	var told []snapshot.Fragment
	var asked [][]snapshot.LocationRef
	s, err := New(Config{Clock: clk, Assembler: asm, Locations: refs, Providers: []snapshot.Provider{p},
		Tiers: []Tier{{Kind: snapshot.KindObs, Every: time.Hour}},
		OnFragment: func(f snapshot.Fragment, locs []snapshot.LocationRef) {
			mu.Lock()
			defer mu.Unlock()
			told, asked = append(told, f), append(asked, locs)
		}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(told) >= 1 }, "the applied fetch told")
	s.Stop()
	mu.Lock()
	defer mu.Unlock()
	if told[0].Provider != "nws" || told[0].Kind != snapshot.KindObs || len(asked[0]) != len(refs) || asked[0][0].Label != refs[0].Label {
		t.Errorf("told %+v for %v; want NWS's observations for the locations asked", told[0], asked[0])
	}
}
