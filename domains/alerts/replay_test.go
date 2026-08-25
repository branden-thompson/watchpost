// Package alerts_test hosts the M2/M3 replay harness (architecture §7/§10.8):
// recorded alert-feed states replayed through the real scheduler+assembler
// with a fake clock, asserting 100% coverage (M3) and issuance→render ≤60s
// (M2). Fixtures: testdata/replays/*.jsonl (tools/alertrec records real feeds).
package alerts_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

type feedState struct {
	At     time.Time `json:"at"`
	Alerts []struct {
		ID       string    `json:"id"`
		Event    string    `json:"event"`
		Severity string    `json:"severity"`
		Sent     time.Time `json:"sent"`
		Expires  time.Time `json:"expires"`
	} `json:"alerts"`
}

func loadReplay(t *testing.T, name string) []feedState {
	t.Helper()
	f, err := os.Open("testdata/replays/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var states []feedState
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var st feedState
		if err := json.Unmarshal(sc.Bytes(), &st); err != nil {
			t.Fatal(err)
		}
		states = append(states, st)
	}
	return states
}

// replayClock is a fake clock the harness drives along the fixture timeline.
type replayClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []struct {
		at time.Time
		ch chan time.Time
	}
}

func (c *replayClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *replayClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, struct {
		at time.Time
		ch chan time.Time
	}{c.now.Add(d), ch})
	return ch
}
func (c *replayClock) advanceTo(t time.Time) {
	c.mu.Lock()
	c.now = t
	var due []chan time.Time
	rest := c.waiters[:0]
	for _, w := range c.waiters {
		if !w.at.After(t) {
			due = append(due, w.ch)
		} else {
			rest = append(rest, w)
		}
	}
	c.waiters = rest
	c.mu.Unlock()
	for _, ch := range due {
		ch <- t
	}
	time.Sleep(5 * time.Millisecond) // let tier goroutines run their cycle
}

// feedProvider serves the fixture's state as of the fake clock's now.
type feedProvider struct {
	clk    *replayClock
	states []feedState
	key    snapshot.LocationKey
}

func (p *feedProvider) ID() string        { return "nws" }
func (p *feedProvider) Domains() []string { return []string{"alerts"} }
func (p *feedProvider) Fetch(_ context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	now := p.clk.Now()
	var current *feedState
	for i := range p.states {
		if !p.states[i].At.After(now) {
			current = &p.states[i]
		}
	}
	al := []snapshot.Alert{}
	if current != nil {
		for _, a := range current.Alerts {
			al = append(al, snapshot.Alert{ID: a.ID, Event: a.Event, Severity: a.Severity, Sent: a.Sent, Expires: a.Expires,
				Source: snapshot.SourceInfo{Provider: "nws", IssuedAt: a.Sent}})
		}
	}
	return snapshot.Fragment{Provider: "nws", Kind: snapshot.KindAlerts, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{p.key: {Alerts: al}}}, nil
}

func TestReplayM2M3(t *testing.T) {
	states := loadReplay(t, "basic.jsonl")
	ref := snapshot.LocationRef{Label: "A", Lat: 33.24, Lon: -117.29}
	clk := &replayClock{now: states[0].At.Add(-30 * time.Second)}
	provider := &feedProvider{clk: clk, states: states, key: snapshot.Key(ref)}
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{"nws"})

	var mu sync.Mutex
	firstRender := map[string]time.Time{} // alert id -> fake-clock render time
	s, err := sched.New(sched.Config{
		Clock: clk, Assembler: asm,
		Locations: []snapshot.LocationRef{ref},
		Providers: []snapshot.Provider{provider},
		Tiers:     []sched.Tier{{Kind: snapshot.KindAlerts, Every: 20 * time.Second}},
		OnPublish: func() { // UAT 74: a notification; the observer snapshots
			snap := asm.Snapshot()
			mu.Lock()
			defer mu.Unlock()
			for _, a := range snap.Locations[0].Alerts {
				if _, seen := firstRender[a.ID]; !seen {
					firstRender[a.ID] = clk.Now()
				}
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// Drive the fake clock in 20s scheduler steps across the fixture window.
	end := states[len(states)-1].At.Add(40 * time.Second)
	for now := clk.Now(); now.Before(end); now = clk.Now() {
		clk.advanceTo(now.Add(20 * time.Second))
	}

	// M3: 100% coverage — every alert that ever appeared in the feed rendered.
	want := map[string]time.Time{}
	for _, st := range states {
		for _, a := range st.Alerts {
			if _, ok := want[a.ID]; !ok {
				want[a.ID] = a.Sent
			}
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for id, sent := range want {
		rendered, ok := firstRender[id]
		if !ok {
			t.Errorf("M3 FAIL: alert %s never rendered", id)
			continue
		}
		// M2: issuance -> first render ≤60s. Feed visibility bounds this too:
		// latency = (feed appearance - sent) + poll delay; assert the metric.
		if lat := rendered.Sub(sent); lat > 60*time.Second {
			t.Errorf("M2 FAIL: alert %s rendered %v after issuance (>60s)", id, lat)
		}
	}
	// Cancellation honesty: alert-1 left the feed at 12:02; a snapshot after
	// the final state must not carry it.
	final := asm.Snapshot()
	for _, a := range final.Locations[0].Alerts {
		if a.ID == "alert-1" {
			t.Error("cancelled alert-1 must not persist after leaving the feed")
		}
	}
}
