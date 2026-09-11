package app

// schedule.go — the Director, its executors and the pump, wired into the
// running station (T3.2a).
//
// IT DRIVES THE LIVE ALERT RAIL (T3.10b). It did not when this file was
// written — the header claimed, correctly then and falsely for the rest of the
// release, that no arrival reached it — and the sentence stayed while the wiring
// changed eighteen lines below, at `tick.emit = s.carry`. A file header telling
// the next maintainer that the file wiring the Director into a weather radio is
// dead is the most expensive comment in the tree (red team 2026-09-05, R-3).
//
// So: the producer states what arrived, the Director schedules it, the Composer
// writes it and the Reader performs it. The main track still runs through the
// radio deck.
//
// It is split from the absorb it enables (T3.2b) deliberately. Moving the
// Watchlist dwell onto a Director that nothing drives would have deleted a
// working five-minute advance and replaced it with something that never fires;
// splitting isolates the risky half behind a claim a test can state — nothing
// changes — and every later Phase 3 task needs this half regardless.

import (
	tea "charm.land/bubbletea/v2"

	"context"
	"errors"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// scheduleTick is how often the Director is told the time.
//
// The clock is an EVENT (DR-20), so this is the resolution of every deadline
// the schedule keeps. One second is far finer than anything it decides today —
// the Watchlist dwell it absorbs at T3.2b is five minutes — and costs one
// channel send a second on a goroutine that is otherwise asleep.
const scheduleTick = time.Second

// schedule owns the pump's lifetime.
type schedule struct {
	pump   *pump
	ctx    context.Context // the schedule's own: a late producer gives up rather than blocking
	cancel context.CancelFunc
	ticks  chan struct{} // closed by the tick goroutine when it returns
}

// startSchedule builds the Director, its executors and the pump, and starts
// both goroutines.
//
// THE ARBITER AND THE EFFECTOR ARE THE ONES ALREADY RUNNING, not new ones. The
// executors perform through `nar` and `nar.mc`, which is the same pair the
// ticker's takeovers use — building a second effector here would put the band
// and the duck back under two owners, which is the defect T2.3 removed and the
// one this file would be the easiest place to reintroduce.
// THE TWO LISTS ARE NOT ONE LIST (D-76). `pool` is the STATION's candidates —
// what its Producer may offer and what its Composer resolves against; `watch` is
// the LISTENER's, and the bed's cut-over is the MONITOR's rotation moving
// through it.
//
// D-72 MOVED ALL THREE TOGETHER AND THAT WAS TWO-THIRDS RIGHT. The reasoning
// was that a Director scheduling a location its own Composer cannot resolve gets
// it benched by D-67's cool-off — true of `propose` and `compose`, and NOT of
// `cutTo`, which serves a rotation through the listener's own watchlist. A
// watched location outside the station's pool stopped resolving and the tune
// died as `schedule:tune-unknown`, silently. Found by DRAWING THE FLOW for the
// HUM LEAD rather than by a gate.
func startSchedule(ctx context.Context, nar *director, scripts *script.Library, clock func() render.Clock, deck *radioDeck, pool, watch func() []snapshot.LocationRef, tick *tickerDeck, publish func(tea.Msg), noteBed func(bool)) *schedule {
	if nar == nil {
		return nil // no arbiter, no schedule: there is nothing to perform through
	}
	if tick == nil {
		return nil // no producer, no arrivals: the rail would have nothing to read
	}
	x := newExecutors(executors{
		voice: nar,
		// THE CONSOLE'S ONLY SOURCE (0.16.0 P2). It never reads the Director
		// directly — it is told, at the moment the schedule settles.
		publish: publish,
		scripts: scripts,
		clock:   clock,
		now:     time.Now,
		mc:      nar.mc,
		audible: func() bool { return !nar.silent() },
		// MUTE REACHES THE SCHEDULE (T3.10b). `[M]` is the listener's "do not
		// speak to me"; a read that ignored it would consume a hazard in silence.
		muted: tick.muted.Load,
		// THE PRODUCER IS THE TICKER (T3.10b). It holds what an alert IS and the
		// store of what has been read aloud; the card carries only the ids.
		alert: tick.alerts.get,
		mark:  func(id string) { tick.seen.mark([]globalfeed.Event{{ID: id}}, time.Now()) },
		// THE SAME STORE, ASKED THE OTHER WAY (I-7): two bursts with different
		// leads can carry the same alert, and neither the producer's filter nor
		// the Director's ID check can see the overlap.
		readAloud: tick.seen.has,
		report: func(f lineup.Effect, why string) {
			radioDebugLog("schedule:declined:" + lineup.Describe(f) + ":" + why)
		},
		// THE MONITOR'S ROTATION MOVES THROUGH THE LISTENER'S OWN WATCHLIST
		// (D-76). It is `advanceBed` that emits this, and `advanceBed` is the
		// operator's own listening — not the station's.
		cutTo: tuneTo(deck, watch),
		// AND WHAT TO CALL IT ON THE CONSOLE (F-79). The same list the cut-over
		// resolves against, so the row cannot name a place the tune did not go.
		bedLabel: bedLabelOf(watch),
		noteBed:  noteBed,
		// THE MAIN TRACK'S WORDS (0.16.0 P3). Required from P3(d): a station
		// whose rotation is owned by the schedule and has no composer wired
		// would queue every report and read none.
		compose: composeFor(deck, pool),
		// AND WHAT PERFORMS THEM (F-91, BD-9). The rail reads through the
		// arbiter above; the programme is a source swap on the broadcast
		// engine, and this is the deck that owns it. Nil with no deck, which is
		// every pathless build — the executor declines the card by name rather
		// than holding it on the air in silence.
		read: readerFor(deck),
		// WHAT THE PRODUCER HAS TO OFFER (0.16.0 P4, D-40). The Director asks
		// on every publish and takes only what the line-up still needs.
		propose: proposeFrom(pool),
		// DR-21's one escalation channel. It reuses the relay-fault window
		// rather than adding a second error surface: from the listener's chair
		// "the relay is silent" and "the schedule stopped" are the same event —
		// the station has gone quiet and they are being offered the way back.
		escalate: func(reason string) { deck.escalate(reason) },
	})
	if x == nil {
		return nil // a seam was nil; newExecutors has already said which
	}
	run, cancel := context.WithCancel(ctx)
	// THE DEPTH IS THE CONSOLE'S SLOT COUNT, FROM THE CONSOLE (D-40). A second
	// constant here would agree with it today and drift silently: the station
	// would hold cards the operator cannot address, or leave slots empty for
	// ever, and neither reads as a bug from either side.
	p := newPump(lineup.New(lineup.Settings{Max: defaultBurstMax, Depth: tty.MainTrackSlots}, time.Now()), x.run,
		func(f lineup.Effect, v any) { radioDebugLog("schedule:fault:" + lineup.Describe(f)) })
	if p == nil {
		cancel()
		return nil
	}
	s := &schedule{pump: p, ctx: run, cancel: cancel, ticks: make(chan struct{})}
	go p.loop(run)
	go s.tick(run)
	// THE PRODUCERS REPORT INTO IT, WIRED HERE (T3.10b).
	//
	// Both were the caller's job, and the ticker's was one line away from being
	// forgotten: without it the rail receives nothing and NO HAZARD IS EVER
	// READ, while every test stays green because each wires its own schedule.
	// That is the shape of D-12 — three tests passing over an inert Settings row
	// because they called the cycle function instead of pressing the key — so
	// the wiring lives with the thing that needs it and a test can assert it.
	//
	// There is nothing mutual about it any more: both are parameters.
	tick.mu.Lock()
	tick.emit = s.carry
	tick.mu.Unlock()
	if deck != nil {
		deck.mu.Lock()
		deck.emit = s.carry
		deck.mu.Unlock()
	}
	// AND MASTERCONTROL DECLARES ON AIR / STANDBY (MVS-D-78, FR-5.4). It is the
	// third thing that speaks to the Director and the only one that speaks for
	// the OPERATOR — the ticker reports arrivals, the deck reports the bed, and
	// this carries a decision a human made.
	if nar != nil && nar.mc != nil {
		nar.mc.mu.Lock()
		nar.mc.carry = s.carry
		nar.mc.mu.Unlock()
		// AND THE CONSOLE IS HANDED THE CONTROL (FR-5.4). Without this the
		// operator's ON AIR / STANDBY key reaches a nil seam and does nothing,
		// which is the console being a surface with no controls — the trap
		// F-72 recorded, arriving by a different route.
		//
		// ON ITS OWN GOROUTINE, AND THAT IS NOT STYLE. `publish` is the
		// program's Send, and THIS RUNS BEFORE THE PROGRAM'S LOOP DOES: a
		// synchronous send here blocks until something reads it, and nothing
		// will, because the reader is the loop this function returns to start.
		// Measured: `TestRunWithoutArgsStartsTheDashboard` hung for the full
		// ten-minute test timeout, with the trace pointing at this line.
		//
		// It is the shape every other startup-time sender already has —
		// severeDeck publishes from its own goroutine for the same reason — and
		// `Send` selects on the program's context, so a program that never
		// starts unblocks it at shutdown rather than leaking.
		if publish != nil {
			go publish(tty.StationControlMsg{Control: nar.mc})
		}
	}
	return s
}

// tuneTo resolves the key the Director named back to a location and cuts the
// bed over to it.
//
// THE UNEXPORTED tune, DELIBERATELY. Every tune the Director asks for is
// automatic — a dwell elapsed, a cycle ended — and lifting the alert duck on an
// automatic transition brought the next location's report in at full volume over
// a breaking alert still reading. That distinction used to live in the case of an
// identifier; T2.3 gave the duck one owner instead, and this calls the path that
// has never lifted it.
//
// A KEY THAT NAMES NOTHING IS DROPPED, not guessed at. The listener can remove a
// location from the watchlist between the Director planning a move and the move
// running, and tuning "the nearest thing to what they asked for" would put a
// station on the air they had just taken away.
func tuneTo(deck *radioDeck, watch func() []snapshot.LocationRef) func(string) {
	return func(ref string) {
		if deck == nil {
			return // no audio
		}
		r, ok := refFor(watch, ref)
		if !ok {
			radioDebugLog("schedule:tune-unknown:" + ref)
			return
		}
		deck.tune(r)
	}
}

// refFor resolves the key a card carries back to the location it names.
//
// THE CARD STAYS DOMAIN-FREE (DR-1), so what travels through the schedule is an
// identifier and nothing more, and the app is where it becomes a place again.
// EXTRACTED AT THE SECOND CALLER: the cut-over asked this question first, and
// the main-track composer asks the same one — two walks of the watchlist
// comparing the same key would be two places for "what is a location's identity"
// to drift.
func refFor(watch func() []snapshot.LocationRef, ref string) (snapshot.LocationRef, bool) {
	if watch == nil {
		return snapshot.LocationRef{}, false // no watchlist to resolve against
	}
	for _, r := range watch() { // bounded by the watchlist (P10-02)
		if string(snapshot.Key(r)) == ref {
			return r, true
		}
	}
	return snapshot.LocationRef{}, false
}

// composeFor is how a main-track card gets its words (0.16.0 P3).
//
// THE EXECUTOR KNOWS NOTHING ABOUT HOW A REPORT IS ASSEMBLED, and this is the
// other side of that: the deck owns what a location report IS — the observation,
// the alerts, the office products, the sign-off — and hands back the segments it
// would have voiced. It is the same call startSynth makes for its own source.
//
// WHAT IS SHARED IS THE TEXT, NOT THE DELIVERY, and the first draft of this
// comment overclaimed it (red team 2026-09-09, finding 5). A synth.Segment
// carries Key, Text, Role, SelfIntro and Pause; scriptFromSegments keeps Text
// and drops the rest, because lineup.Part has nowhere to put them. The source
// consumes all five. So the two paths cannot say different WORDS — and can
// still differ in which correspondent says them, whether an introduction is
// suppressed, and how long the pauses are.
//
// That is a live question for the flip and it is recorded there, not resolved
// here (04-development/p3-flip-design.md, G-7).
func composeFor(deck *radioDeck, watch func() []snapshot.LocationRef) func(context.Context, string) ([]synth.Segment, error) {
	return func(ctx context.Context, ref string) ([]synth.Segment, error) {
		if deck == nil {
			return nil, errors.New("no audio deck to compose a report")
		}
		r, ok := refFor(watch, ref)
		if !ok {
			// NAMED, NOT EMPTY. The executor turns this into a decline that
			// says why, and a card for a location the listener has since
			// removed is exactly the case that produces it.
			return nil, errors.New("no watched location for " + ref)
		}
		return deck.segments(ctx, r, synth.VoiceToken)
	}
}

// carry hands the Director something that happened, from anywhere outside it.
//
// NAMED carry, NOT send, for the tool rather than the code: P10 resolves by
// NAME, so a `send` here collides with pump.send — which is genuinely in a call
// cycle — and reports this as recursion it has no part in. Fifth rename in this
// release for the same false positive, and the collisions are all between a
// method on one type and a method on another.
//
// IT CARRIES THE SCHEDULE'S OWN CONTEXT, so a producer that outlives the station
// — the deck reports a status after shutdown has begun — gives up rather than
// blocking on a pump that has stopped reading.
func (s *schedule) carry(ev lineup.Event) {
	if s == nil {
		return
	}
	s.pump.send(s.ctx, ev)
}

// tick tells the Director the time until the station stops.
func (s *schedule) tick(ctx context.Context) {
	defer close(s.ticks)
	everyTick(ctx, scheduleTick, func(now time.Time) {
		s.pump.send(ctx, lineup.Tick{Now: now})
	})
}

// stop ends the schedule and WAITS for both goroutines.
//
// Both, and in this order. The tick goroutine sends to the pump, so a stop that
// waited only for the pump would leave a sender writing to a loop that had
// returned — the shape that made a 0.12.0 tag go red on the Linux race gate,
// where a fire-and-forget goroutine outlived the thing it wrote to.
func (s *schedule) stop() {
	if s == nil {
		return
	}
	s.cancel()
	<-s.ticks
	s.pump.stop()
}

// proposeFrom turns the listener's watched locations into what the Producer can
// offer the Director (D-40).
//
// THE PRODUCER PROPOSES; THE DIRECTOR CHOOSES. It offers everything it has, in
// the listener's own order, and does not look at the schedule at all — whether
// any of it is scheduled, and which, is the Director's, decided from the depth
// and the watchlist it already holds. That is the role split the HUM LEAD drew:
// "it's the Producer's job to PROPOSE … the DIRECTOR, as the owner of the
// lineup, then is the one who gets to choose which card gets the slot."
//
// IT KEYS EACH PROPOSAL THE WAY THE ROTATION DOES — `snapshot.Key`, the same
// ref `radioDeck.needsRead` reports — so `ReadID` gives a location one identity
// across both paths and the lineup's own refusal of a duplicate is what stops a
// place being read twice (FR-2.5).
func proposeFrom(watch func() []snapshot.LocationRef) func() []lineup.Proposal {
	return func() []lineup.Proposal {
		if watch == nil {
			return nil
		}
		refs := watch()
		out := make([]lineup.Proposal, 0, len(refs))
		for _, r := range refs { // bounded by the watchlist (P10-02)
			key := string(snapshot.Key(r))
			if key == "" || r.Label == "" {
				continue // a location with no key or no name cannot become a card
			}
			out = append(out, lineup.Proposal{Ref: key, Headline: r.Label, Slot: lineup.LocationReport})
		}
		return out
	}
}

// bedLabelOf turns the bed's ref into the words the console shows.
//
// THE LOCATION'S OWN NAME, which is all the schedule can honestly say today: the
// relay's call sign, frequency and distance live on the deck's resolved station,
// and the bed's ref is a LOCATION key. Naming the place the bed is carrying is
// true; inventing a call sign from a key would not be.
func bedLabelOf(watch func() []snapshot.LocationRef) func(string) string {
	return func(ref string) string {
		r, ok := refFor(watch, ref)
		if !ok {
			return ""
		}
		return r.Label
	}
}
