package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// runEffect performs one effect and returns whatever it learned as events. It
// may block for as long as it likes — a cold card build is 1.03 s of network —
// because it never runs on the pump's own goroutine.
type runEffect func(context.Context, lineup.Effect) []lineup.Event

// pump is the only thing that calls Step.
//
// STATELESS BY DESIGN: it holds the Director because someone must, and it does
// nothing to it but hand it the next event. Every decision is in Step, which is
// pure; every action is in an executor, which runs elsewhere. That leaves the
// pump with one job — keep the two apart — and one rule.
//
// THE RULE IS THAT IT NEVER WAITS FOR WORK (PL-1). An earlier draft of this
// loop ran each effect inline, which would have paid the 1.03 s card build on
// the pump and reintroduced the blocking class inside the approach chosen to
// avoid it. Effects are handed to workers and the loop returns to the select.
//
// It is not "never blocks", and the difference is worth stating precisely: the
// loop parks if the shared-output lane's buffer fills. That the buffer cannot
// fill is a property of the SCHEDULE, not of this file (see laneBacklog), so
// the guarantee here is about work, and the bound behind it lives elsewhere.
type pump struct {
	dir    lineup.Director
	events chan lineup.Event
	run    runEffect

	// jobs is every dispatched effect still in flight, drained at shutdown the
	// way the takeover goroutines are (R5-B-07): a job in flight ends with the
	// context, and is waited for rather than abandoned.
	jobs sync.WaitGroup
	done chan struct{}

	// lane is the one worker every effect touching a SHARED OUTPUT runs on —
	// the marquee and the bed — in the order the Director described it. See
	// runLane.
	lane chan []lineup.Effect

	// onFault sees every contained panic. The Director is told about the ones
	// that fail a card; this is the record of ALL of them, because a fault that
	// fails nothing is still the only sign that an executor is broken (DR-23).
	onFault func(lineup.Effect, any)
}

// newPump wires a pump. It does not start it.
func newPump(d lineup.Director, run runEffect, onFault func(lineup.Effect, any)) *pump {
	if err := invariant.Check(run != nil, "a pump is built with something to run its effects"); err != nil {
		return nil
	}
	if err := invariant.Check(onFault != nil, "a pump is built with somewhere to report a contained panic"); err != nil {
		return nil
	}
	// A DIRECTOR THAT REFUSED TO BE BUILT IS NOT A SCHEDULE (red team
	// 2026-09-05, I-9). lineup.New returns a zero Director on a zero clock or a
	// negative Max, and a zero Director refuses every event for ever — Step
	// fails its own clock invariant and returns unchanged, and Now() fails
	// quiet too, so the state is unobservable from out here. The station would
	// run, tick, and never schedule anything, with nothing anywhere saying so.
	// Unreachable from today's single call site; the second caller is the
	// Broadcaster, which is exactly when nobody will be looking for it.
	if err := invariant.Check(!d.Now().IsZero(), "a pump is built with a Director that has a clock"); err != nil {
		return nil
	}
	return &pump{dir: d, run: run, onFault: onFault,
		events: make(chan lineup.Event, pumpBacklog), done: make(chan struct{}),
		lane: make(chan []lineup.Effect, laneBacklog)}
}

// pumpBacklog is how many events may queue before a producer waits.
//
// It exists so a burst of completions from several workers at once does not
// serialise them against the loop's turnaround. It is NOT a bound on the
// schedule — the Lineup is that — and a producer that fills it WAITS, which is
// backpressure rather than loss.
//
// No event is dropped for want of room. Events ARE dropped when the context
// ends — send, work and dispatch all give up on cancellation — which is the
// station shutting down rather than the queue overflowing.
const pumpBacklog = 64

// laneBacklog is how many runs may queue for the shared-output lane.
//
// IT IS NEVER REACHED, and the reason lives in the Director rather than here:
// takeTheAir refuses a busy air, so at most one card's cue-and-words and the
// outgoing card's release are ever outstanding at once. Measured driving the
// real Director with the lane wedged for 400 rounds, the high-water mark was
// ZERO. The buffer is headroom for a shape the schedule does not produce, and
// a producer that somehow filled it would WAIT — backpressure rather than
// loss, which is the same trade pumpBacklog makes.
const laneBacklog = 64

// loop is the pump. One goroutine, one event at a time, and no work.
func (p *pump) loop(ctx context.Context) {
	defer close(p.done)
	// The lane is closed HERE, by the only goroutine that sends to it, so it
	// retires with the loop rather than outliving it: cancelling the context
	// used to leave it running until someone called stop.
	defer close(p.lane)
	go p.runLane(ctx)
	for { // bounded by the context (P10-02): every exit is a cancel
		select {
		case <-ctx.Done():
			return
		case ev := <-p.events:
			var fx []lineup.Effect
			p.dir, fx = p.dir.Step(ev)
			trace(ev, fx, p.dir.Lineup())
			p.dispatch(ctx, fx)
		}
	}
}

// stop drains. The caller cancels the context first; this waits for the loop to
// return and for every dispatched effect to finish, so nothing is still writing
// when teardown starts.
func (p *pump) stop() {
	<-p.done // the loop has returned and closed the lane, which drains what is queued
	p.jobs.Wait()
}

// send offers an event to the pump, or gives up when the context ends. It is how
// everything outside reaches the schedule.
func (p *pump) send(ctx context.Context, ev lineup.Event) {
	select {
	case p.events <- ev:
	case <-ctx.Done():
	}
}

// dispatch hands a step's effects to workers and returns without waiting for
// any of them to finish (PL-1).
//
// Effects about ONE CARD keep the order Step gave them, because the cue must
// reach the band before that card's words are spoken (DR-18). Everything else
// may run at once, so a 1.03 s build for the next card never waits behind a
// read.
//
// ANYTHING TOUCHING A SHARED OUTPUT GOES TO ONE LANE INSTEAD, and it is
// enqueued HERE — on the pump's own goroutine, which is serial — so the lane
// receives that work in exactly the order the Director described it.
//
// It parks only if the lane's buffer is full, which the schedule does not
// produce (see laneBacklog); the send is preferred over the cancel so a run
// that could be queued is not discarded while there is room for it.
func (p *pump) dispatch(ctx context.Context, fx []lineup.Effect) {
	for _, group := range runsOf(fx) { // bounded by the step's effects (P10-02)
		p.jobs.Add(1)
		if !sharesAnOutput(group) {
			go p.work(ctx, group)
			continue
		}
		select {
		case p.lane <- group:
		default:
			// Both arms below are usually ready at once, and select picks
			// between ready arms at random — so without the attempt above, a
			// run stop would have drained was discarded on a coin flip, and
			// the run in question is the RELEASE.
			select {
			case p.lane <- group:
			case <-ctx.Done():
				p.jobs.Done()
			}
		}
	}
}

// runLane performs everything touching a shared output, ONE AT A TIME AND IN
// THE ORDER IT WAS DESCRIBED.
//
// The band shows one callout at a time and the bed is one broadcast, and both
// are only correct in order: a release that overtakes the next card's cue
// clears a callout that has just gone up, and a duck that lands after the read
// has started is the duck-lift bug in a new costume (RD-2). Grouping alone
// cannot give either, because the effects concerned belong to DIFFERENT cards
// or to no card at all, and they routinely fall in DIFFERENT steps — a card
// finishes while the next card's 1.03 s build is still out. Ordering has to
// outlive the step, and one lane fed from the serial pump is the whole of it.
//
// It ends when the loop closes the channel, having finished what was queued:
// no dispatch can follow, because the loop is the only sender.
func (p *pump) runLane(ctx context.Context) {
	for group := range p.lane { // bounded by the channel (P10-02)
		p.work(ctx, group)
	}
}

// sharesAnOutput reports whether any effect in a run touches something only one
// effect may have at a time — the band, or the bed. Such a run rides the lane;
// everything else is free to run at once.
func sharesAnOutput(group []lineup.Effect) bool {
	for _, f := range group { // bounded by the run (P10-02)
		for _, r := range lineup.Holds(f) { // bounded by the effect (P10-02)
			if r == lineup.TheBand || r == lineup.TheBed {
				return true
			}
		}
	}
	return false
}

// runsOf splits a step's effects into the runs that must stay in order: ALL the
// effects about one card, in the order they were described.
//
// CONNECTED, NOT ADJACENT. A card's effects join its run wherever they fall,
// even when something that names no card was described between them — the cue
// and the words of one card must not land on separate workers merely because a
// publish sat between them. Effects that name no card each get a run of their
// own and run at once.
func runsOf(fx []lineup.Effect) [][]lineup.Effect {
	var runs [][]lineup.Effect
	of := map[string]int{} // card -> the run its effects share
	for _, f := range fx { // bounded by the step's effects (P10-02)
		if id, ofCard := lineup.CardOf(f); ofCard {
			if i, seen := of[id]; seen {
				runs[i] = append(runs[i], f)
				continue
			}
			of[id] = len(runs)
		}
		runs = append(runs, []lineup.Effect{f})
	}
	// EVERY EFFECT APPEARS EXACTLY ONCE, which is what dispatch's one Add per
	// run depends on: an effect dropped here is work nobody does and a card
	// that waits for ever, and one duplicated is a read heard twice. The
	// obvious `len(runs) <= len(fx)` cannot fail — runs grows by at most one
	// per effect — so it says nothing.
	placed := 0
	for _, r := range runs { // bounded by the runs (P10-02)
		placed += len(r)
	}
	if err := invariant.Check(placed == len(fx), "every effect is placed in exactly one run"); err != nil {
		return nil
	}
	return runs
}

// work runs one ordered group and feeds what it learns back as events.
func (p *pump) work(ctx context.Context, group []lineup.Effect) {
	defer p.jobs.Done()
	for _, f := range group { // bounded by the group (P10-02)
		for _, ev := range p.contained(ctx, f) {
			select {
			case p.events <- ev:
			case <-ctx.Done():
				return
			}
		}
	}
}

// contained runs one effect and turns a panic into a fault (DR-22).
//
// UNDER APPROACH C THE PUMP IS THE SINGLE OWNER, so an unguarded panic would
// take both tracks and the bed with it — a strictly larger blast radius than
// today, where startTakeover guards a panic precisely because one "would wedge
// the marquee for the life of the process" and that is ONE surface.
//
// The recover is scoped to this function alone, and it is deliberately narrow:
// a panic inside the pump's own loop must never be swallowed here and read as
// "the effect returned nothing". It reports every containment, and additionally
// FAILS THE CARD when the effect named one — so the schedule re-plans around it
// rather than waiting for a completion that will never come.
func (p *pump) contained(ctx context.Context, f lineup.Effect) (out []lineup.Event) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		p.onFault(f, r)
		id, ofCard := lineup.CardOf(f)
		if !ofCard {
			out = nil // nothing to fail: the schedule is unharmed, the record stands
			return
		}
		// NOT Routed (I-2): a panicking executor is the station unable to do the
		// work, and nothing is going to fix it. This is the case DR-21's modal
		// is for, and leaving the field false is what surfaces it.
		out = []lineup.Event{lineup.Failed{ID: id, Reason: fmt.Sprintf("%s panicked: %v", lineup.Describe(f), r)}}
	}()
	return p.run(ctx, f)
}

// trace writes the Director's timeline: what it was told, what it decided, and
// the schedule that left behind (DR-23).
//
// ONE PLACE, because the pump is the only one that sees both sides. The pure
// core cannot write — that is the point of it — so a line per state transition
// at the transition is not available; the schedule AFTER the step is, and it
// cannot disagree with itself the way a hand-written transition line can.
//
// A TICK THAT CHANGED NOTHING GETS NO LINE. Ticks arrive every second, and a
// timeline where the interesting lines are one in three hundred is not a
// timeline anyone reads. Every other event is logged whether it decided
// anything or not: those are rare, and "the Director was told and did nothing"
// is exactly the kind of thing this log exists to show.
func trace(ev lineup.Event, fx []lineup.Effect, l lineup.Lineup) {
	if !radioDebugOn() {
		return
	}
	if _, isTick := ev.(lineup.Tick); isTick && len(fx) == 0 {
		return
	}
	radioDebugLog("director:ev:" + lineup.DescribeEvent(ev))
	for _, f := range fx { // bounded by one step's effects (P10-02)
		radioDebugLog("director:fx:" + lineup.Describe(f))
	}
	radioDebugLog("director:cards:" + l.Trace())
}
