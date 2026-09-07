package app

// station_test.go — the shipped path, driven synchronously (T3.10b).
//
// WHY A HARNESS AND NOT A SHIM. The read used to be `tickerDeck.breaking`, and
// twenty-five tests called it directly. It is now the Director's: the producer
// states what arrived, the Director schedules it, the Composer writes it and the
// Reader performs it. A test-only `breaking` kept alive beside that would be a
// pin on a path nobody ships — which is D-12's defect, three tests passing over
// an inert Settings row because they called the cycle function instead of
// pressing the key.
//
// So this drives the REAL path, end to end, and the only thing it replaces is
// the pump's concurrency: effects run in this goroutine and their events feed
// back. That is what Approach C is for — a test states the events and reads back
// the work, with no sleeps and nothing to synchronise with. The pump's own
// ordering and containment are pinned in pump_test.go, against the real pump.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// station is a ticker deck wired to a Director and its executors, the way
// startSchedule wires them in the running app.
type station struct {
	t       testing.TB
	deck    *tickerDeck
	dir     lineup.Director
	x       *executors
	reports []string
	// escalated is every DR-21 escalation the schedule raised — the thing a
	// LISTENER is told. It was an empty stub, which is what let a nil-deref on
	// this channel (I-1) and a modal for every deliberate decline (I-2) both
	// through: the composition "a rail card fails, a person is told" existed in
	// three pieces and was never joined (red team 2026-09-05, R-11).
	escalated []string

	mu      sync.Mutex
	pending []lineup.Event // what the producer has told the Director, not yet stepped
}

// newStation wires a deck built by a test into a Director and its executors.
// The seams are startSchedule's, so a divergence between the two is a
// divergence a reader can see side by side.
func newStation(t testing.TB, deck *tickerDeck) *station {
	t.Helper()
	if deck.alerts == nil {
		deck.alerts = newAlertStore()
	}
	if deck.mc == nil && deck.voice != nil {
		deck.mc = deck.voice.mc
	}
	s := &station{t: t, deck: deck, dir: lineup.New(lineup.Settings{Max: defaultBurstMax}, time.Now())}
	s.x = newExecutors(executors{
		voice:     deck.voice,
		scripts:   deck.scripts,
		clock:     deck.clock,
		now:       time.Now,
		mc:        deck.mc,
		audible:   func() bool { return deck.voice == nil || !deck.voice.silent() },
		muted:     deck.muted.Load,
		alert:     deck.alerts.get,
		mark:      func(id string) { deck.seen.mark([]globalfeed.Event{{ID: id}}, time.Now()) },
		readAloud: deck.seen.has,
		report:    func(f lineup.Effect, why string) { s.reports = append(s.reports, lineup.Describe(f)+": "+why) },
		cutTo:     func(string) {},
		escalate:  func(reason string) { s.escalated = append(s.escalated, reason) },
	})
	if s.x == nil {
		t.Fatal("the station's executors were refused; a seam is missing")
	}
	// THE DECK REPORTS INTO THE STATION, exactly as it reports into the schedule
	// in the running app. A test that drove `cycle` without this would exercise
	// a producer talking to nobody.
	deck.mu.Lock()
	deck.emit = s.queue
	deck.mu.Unlock()
	return s
}

// queue is the deck's emit: it collects rather than dispatching, so the test
// decides when the Director steps.
func (s *station) queue(ev lineup.Event) {
	s.mu.Lock()
	s.pending = append(s.pending, ev)
	s.mu.Unlock()
}

// drain steps everything the producer has said and performs what it sets off.
func (s *station) drain(ctx context.Context) {
	s.mu.Lock()
	evs := s.pending
	s.pending = nil
	s.mu.Unlock()
	s.run(ctx, evs...)
}

// takeover is one cycle of the alert rail: the producer offers a burst and
// everything it sets off runs to quiescence.
func (s *station) takeover(ctx context.Context, evs []globalfeed.Event) {
	s.offer(evs)
	s.drain(ctx)
}

// offer is the PRODUCER's half alone: what it decided to tell the Director,
// before anything acts on it.
//
// SPLIT OUT SO THE PRODUCER'S OWN DECISIONS ARE OBSERVABLE (red team
// 2026-09-05). MVS-D-78's mute gate lives in startTakeover, and asserting it
// after the drain could not see it: the burst reached the Director anyway,
// speak declined on the SECOND mute gate, the Failed took the card off the rail,
// and the rail read empty either way. Deleting the producer's gate left the
// whole repository green.
func (s *station) offer(evs []globalfeed.Event) {
	s.deck.startTakeover(evs)
}

// pendingCount is how many events the producer has handed over and not yet had
// stepped — the observable the mute gate needs.
func (s *station) pendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}

// run steps the Director and performs what it describes, until nothing is left
// to do. An effect's events go back to the Director, which is the pump's loop
// with the goroutines taken out.
func (s *station) run(ctx context.Context, evs ...lineup.Event) {
	queue := append([]lineup.Event(nil), evs...)
	// BOUNDED (P10-02). A burst is bounded at admission and each card produces a
	// fixed number of effects, so this terminates — the cap guards against a
	// future cycle rather than a real limit.
	//
	// AND IT FAILS LOUDLY, which the comment used to CLAIM while the loop simply
	// returned with work still queued (red team 2026-09-05). A silent truncation
	// here leaves every assertion in the caller running over a PARTIAL sequence,
	// which is this release's own recorded failure mode wearing the harness's
	// clothes.
	steps := 0
	for ; len(queue) > 0 && steps < 10_000; steps++ {
		ev := queue[0]
		queue = queue[1:]
		var fx []lineup.Effect
		s.dir, fx = s.dir.Step(ev)
		for _, f := range fx {
			queue = append(queue, s.x.run(ctx, f)...)
		}
	}
	if len(queue) > 0 {
		s.t.Fatalf("the station did not settle in %d steps, %d event(s) still queued: every assertion "+
			"after this would run over a partial sequence", steps, len(queue))
	}
}

// planned is what the rail would admit from these events and how many it would
// divert — the Director's OWN planner, asked directly so a test can state its
// fixture's shape before asserting on the read.
//
// DERIVED, NOT REIMPLEMENTED. It calls the same lineup.Plan over the same
// arrivalsOf translation the producer uses, with the deck's own fence, so a
// change to the ladder cannot leave a fixture asserting yesterday's order.
func planned(t testing.TB, deck *tickerDeck, evs []globalfeed.Event) ([]globalfeed.Event, int) {
	t.Helper()
	b, err := lineup.Plan(arrivalsOf(evs), lineup.Settings{Max: defaultBurstMax, Fence: deck.fence()}, time.Now())
	if err != nil {
		t.Fatalf("planning the fixture: %v", err)
	}
	byID := make(map[string]globalfeed.Event, len(evs))
	for _, e := range evs {
		byID[e.ID] = e
	}
	out := make([]globalfeed.Event, 0, len(b.Takeover.Refs))
	for _, r := range b.Takeover.Refs { // bounded by the takeover (P10-02)
		if e, ok := byID[r]; ok {
			out = append(out, e)
		}
	}
	return out, b.Divert
}
