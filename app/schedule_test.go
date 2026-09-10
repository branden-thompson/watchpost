package app

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// scheduleUnderTest starts a schedule over a silent arbiter and returns it with
// a count of everything the band was told.
func scheduleUnderTest(t *testing.T, ctx context.Context) (*schedule, *atomic.Int64) {
	t.Helper()
	var band atomic.Int64
	nar := testDirector(nil, func(tea.Msg) { band.Add(1) })
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	s := startSchedule(ctx, nar, nil, func() render.Clock { return render.Clock12 }, nil, nil, tick, nil)
	if s == nil {
		t.Fatal("the schedule refused to start over a valid arbiter")
	}
	return s, &band
}

// T3.2a's WHOLE CLAIM: the schedule runs and a listener notices nothing.
//
// With no arrival, the Lineup is empty, `settle` has nothing to promote and no
// effect is produced — so the executors are never reached and the band is never
// written. Asserted rather than assumed, because "it changes nothing" is exactly
// the kind of claim that is believed without being checked.
func TestTheRunningScheduleTouchesNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, band := scheduleUnderTest(t, ctx)

	// Ticks are what it does today, so drive several rather than waiting on the
	// real one-second cadence: the point is that MANY ticks still produce
	// nothing, not that one does.
	for i := 0; i < 50; i++ {
		s.pump.send(ctx, lineup.Tick{Now: time.Now().Add(time.Duration(i) * time.Minute)})
	}
	s.stop()

	if got := band.Load(); got != 0 {
		t.Errorf("the schedule wrote to the band %d times while driving nothing — "+
			"T3.2a's claim is that a listener notices nothing", got)
	}
	// THE INSTRUMENT MUST BE ABLE TO COUNT, or "zero" means nothing. A counter
	// wired to the wrong place reports a perfect silence it cannot hear.
	band.Add(1)
	if band.Load() != 1 {
		t.Fatal("the band counter does not count; the zero above proved nothing")
	}
}

// A SCHEDULE'S WHOLE LIFE LEAVES NO GOROUTINE BEHIND.
//
// WHAT THIS DOES AND DOES NOT PIN, stated because the difference matters. It
// measures that nothing is LEAKED across start-to-stop. It does NOT pin that
// `stop` waits for the tick goroutine specifically: removing that wait leaves
// this green, checked by removing it. The wait is still right — the tick
// goroutine sends to the pump, and a sender outliving what it writes to is the
// shape that turned a 0.12.0 tag red on the Linux race gate — but it is not
// OBSERVABLE here, because `pump.send` already gives up on cancellation, so the
// goroutine returns promptly either way. Claiming this test pins the ordering
// would be claiming coverage that does not exist.
func TestAScheduleLeavesNoGoroutineBehind(t *testing.T) {
	settle := func() int {
		for i := 0; i < 50; i++ {
			runtime.GC()
			time.Sleep(2 * time.Millisecond)
		}
		return runtime.NumGoroutine()
	}
	before := settle()

	// NO EXTERNAL CANCEL — `stop` must be the only thing that shuts this down.
	// An earlier version cancelled the context itself right after stopping, so
	// the goroutines exited on that instead and the count came back to baseline
	// no matter what `stop` did. It stayed green with `stop` gutted entirely,
	// which is a leak detector that cannot detect a leak.
	s, _ := scheduleUnderTest(t, context.Background())
	s.pump.send(context.Background(), lineup.Tick{Now: time.Now()})
	s.stop()

	if after := settle(); after > before {
		t.Errorf("goroutines went from %d to %d across a schedule's whole life — "+
			"something the schedule started was never waited for", before, after)
	}
}

// A SCHEDULE WITHOUT AN ARBITER DOES NOT START, and does not panic.
//
// There is nothing to perform through, so the honest answer is no schedule —
// and `stop` on that nil must be safe, because stopAll calls it unconditionally
// on a station that never had audio.
func TestNoArbiterMeansNoSchedule(t *testing.T) {
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	if s := startSchedule(context.Background(), nil, nil, func() render.Clock { return render.Clock12 }, nil, nil, tick, nil); s != nil {
		t.Error("a schedule was built with nothing to perform through")
	}
	// AND NO PRODUCER MEANS NO SCHEDULE EITHER. The rail would have nothing to
	// read, and a schedule that cannot receive an arrival is a station that
	// silently never sounds a hazard.
	nar := testDirector(nil, func(tea.Msg) {})
	if s := startSchedule(context.Background(), nar, nil, func() render.Clock { return render.Clock12 }, nil, nil, nil, nil); s != nil {
		t.Error("a schedule was built with no producer to hear from")
	}
	var none *schedule
	none.stop() // must not panic: stopAll calls this on an audio-less station
}

// everyTick RUNS ON THE INTERVAL AND STOPS WITH THE CONTEXT.
//
// It carries the single P10-02 exemption that used to be one per poller, so it
// is worth more than the loops it replaced were individually: a defect here is
// a defect in every poller at once.
func TestEveryTickRunsUntilTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var runs atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		everyTick(ctx, time.Millisecond, func(time.Time) { runs.Add(1) })
	}()
	for deadline := time.Now().Add(2 * time.Second); runs.Load() < 3; {
		if time.Now().After(deadline) {
			t.Fatalf("everyTick ran %d times in two seconds on a 1 ms interval", runs.Load())
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("everyTick did not return when the context ended — every poller built on it leaks")
	}
	after := runs.Load()
	time.Sleep(20 * time.Millisecond)
	if got := runs.Load(); got != after {
		t.Errorf("everyTick ran %d more times after returning", got-after)
	}
}

// A NON-POSITIVE INTERVAL IS REFUSED, not spun on.
//
// A zero interval would make this a busy loop burning a core for the life of the
// station — and time.NewTicker panics on it outright, which would take the app
// down from a config value.
func TestEveryTickRefusesANonPositiveInterval(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			everyTick(context.Background(), d, func(time.Time) {
				t.Errorf("everyTick(%v) ran its function", d)
			})
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("everyTick(%v) did not return", d)
		}
	}
}

// THE DIRECTOR'S TUNE NEVER LIFTS THE ALERT DUCK (T3.2b, PD-1).
//
// Every tune the Director asks for is AUTOMATIC — a dwell elapsed, a cycle
// ended — and nobody pressed anything. Lifting the dip on an automatic
// transition brought the next location's report in at full volume over a
// breaking alert that was still reading; the distinction used to live in the
// case of an identifier, `Tune` versus `tune`, and a listener heard the
// difference. T2.3 gave the duck one owner, and this asserts the absorb kept it.
func TestTheDirectorsTuneLeavesTheDuckAlone(t *testing.T) {
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	nar.mc.giveWay() // an alert is reading: the bed is down
	if !nar.mc.givenWay() {
		t.Fatal("the fixture did not dip the bed, so this proves nothing")
	}

	var tuned []string
	x := newExecutors(executors{
		voice: nar, clock: func() render.Clock { return render.Clock12 }, now: time.Now,
		mc: nar.mc, audible: func() bool { return true },
		alert:     func(string) (globalfeed.Event, bool) { return globalfeed.Event{}, false },
		mark:      func(string) {},
		readAloud: func(string) bool { return false },
		muted:     func() bool { return false },
		report:    func(lineup.Effect, string) {},
		cutTo:     func(ref string) { tuned = append(tuned, ref) },
		escalate:  func(string) {},
	})
	if x == nil {
		t.Fatal("the executors refused to build with every seam supplied")
	}

	x.run(context.Background(), lineup.Tune{Ref: "somewhere"})

	if len(tuned) != 1 || tuned[0] != "somewhere" {
		t.Fatalf("the tune did not reach the bed: %v", tuned)
	}
	if !nar.mc.givenWay() {
		t.Error("the Director's tune lifted the alert duck; the next location would come in at full " +
			"volume over an alert that is still reading")
	}
}

// A TUNE THAT NAMES NOTHING IS DECLINED, not guessed at.
func TestATuneWithNoLocationIsDeclined(t *testing.T) {
	nar := testDirector(&scriptVoice{dur: time.Millisecond}, nil)
	var declined int
	var tuned int
	x := newExecutors(executors{
		voice: nar, clock: func() render.Clock { return render.Clock12 }, now: time.Now,
		mc: nar.mc, audible: func() bool { return true },
		alert:     func(string) (globalfeed.Event, bool) { return globalfeed.Event{}, false },
		mark:      func(string) {},
		readAloud: func(string) bool { return false },
		muted:     func() bool { return false },
		report:    func(lineup.Effect, string) { declined++ },
		cutTo:     func(string) { tuned++ },
		escalate:  func(string) {},
	})
	x.run(context.Background(), lineup.Tune{})
	if tuned != 0 {
		t.Error("a tune naming no location reached the bed")
	}
	if declined != 1 {
		t.Errorf("a tune naming no location was not reported as declined (%d)", declined)
	}
}

// TestTheProducersReachTheSchedule — the wiring, asserted where it lives.
//
// D-12: A PIN STARTS WHERE THE HUMAN STARTS. Every other test in this package
// builds its own Director and executors and drives them directly, so all of them
// stay green if startSchedule never connects the real producers — and the
// station would then run with a rail nothing ever reaches, reading no hazard at
// all, silently. That is the inert-Settings-row defect with a much worse blast
// radius, so it is pinned against the wiring function itself.
func TestTheProducersReachTheSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nar := testDirector(nil, func(tea.Msg) {})
	// A voiceless station falls back to breakingHold (5 s) per part, and this
	// test is about the WIRE, not the pacing.
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	deck := &radioDeck{}
	s := startSchedule(ctx, nar, nil, func() render.Clock { return render.Clock12 }, deck, nil, tick, nil)
	if s == nil {
		t.Fatal("the schedule refused to start over a valid arbiter and producer")
	}
	defer s.stop()

	tick.mu.Lock()
	toTicker := tick.emit
	tick.mu.Unlock()
	if toTicker == nil {
		t.Error("the ticker has nowhere to send its arrivals; the alert rail would never read a hazard")
	}
	deck.mu.Lock()
	toDeck := deck.emit
	deck.mu.Unlock()
	if toDeck == nil {
		t.Error("the deck has nowhere to report the bed; the Watchlist advance would silently stop")
	}

	// AND AN ARRIVAL ACTUALLY REACHES THE READ.
	//
	// RED TEAM 2026-09-05: this test asserted only that emit was non-nil, and
	// `schedule.carry`'s body could be replaced with `_ = ev` — every producer
	// event dropped, the rail dead, no hazard ever read — with the whole app
	// package still green. A NON-NIL POINTER TO A NO-OP IS THE SAME SILENCE
	// this test was written to prevent, which is D-12 one layer further out
	// than the version that caught the missing wiring.
	//
	// Driven from the PRODUCER, so the assertion covers the whole wire:
	// startTakeover -> carry -> pump -> Director -> build -> compose -> speak,
	// and the mark at the end of it is the producer's own store closing the
	// loop. Nothing short of the full path can turn that flag true.
	ev := globalfeed.Event{ID: "rt1", Source: "NWS", Class: globalfeed.ClassSevereWx,
		Type: "Tornado Warning", Location: "Bonsall, CA", Severity: globalfeed.SevRed,
		At: time.Now().Add(-time.Minute), Until: time.Now().Add(time.Hour)}
	tick.startTakeover([]globalfeed.Event{ev})
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		if tick.seen.set()[ev.ID] {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("an arrival never reached the read: the producer offered it, and nothing in the schedule acted on it")
}
