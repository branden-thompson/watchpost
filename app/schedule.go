package app

// schedule.go — the Director, its executors and the pump, wired into the
// running station (T3.2a).
//
// IT DRIVES THE LIVE ALERT RAIL (T3.10b), wired at `tick.emit = s.carry`. The
// producer states what arrived, the Director schedules it, the Composer writes
// it and the Reader performs it. The main track runs through the radio deck.

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
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// scheduleTick is how often the Director is told the time.
//
// The clock is an EVENT (DR-20), so this is the resolution of every deadline
// the schedule keeps. One second is far finer than anything it decides — the
// Watchlist dwell is five minutes — and costs one channel send a second on a
// goroutine that is otherwise asleep.
const scheduleTick = time.Second

// schedule owns the pump's lifetime.
type schedule struct {
	pump   *pump
	ctx    context.Context // the schedule's own: a late producer gives up rather than blocking
	cancel context.CancelFunc
	ticks  chan struct{} // closed by the tick goroutine when it returns

	// executors is what this schedule actually wired, KEPT SO THE WIRING CAN BE
	// DRIVEN (D-140): a call site cannot be asserted, and a field a test reads
	// can. livePipelines.producer is the same remedy for the same reason.
	//
	// A TEST OF THE SET IS NOT A TEST OF THE WIRING: the Composer could resolve
	// against the pool while the operator requests a wider set, and every test that
	// exercises the SET directly would pass without asking what the schedule had
	// been handed.
	executors *executors
}

// bedSeams is what the schedule needs from the console's bed row, and it is ONE
// parameter because it is ONE concern (F-98, D-90).
//
// BOTH HALVES EXIST FOR THE SAME REASON: the row has two publishers — the
// operator's selector and the settle — and each half is how one of them learns
// what the other knows. `note` records the Director's `Carrying` so the selector
// does not guess it; `selected` reports the operator's relay so the settle does
// not overwrite it. Passed separately they would be an eleventh and twelfth
// argument to a function that already has ten, and nothing would say they
// belong together.
type bedSeams struct {
	// note records what the Director says about the bed carrying the programme.
	note func(carrying bool)
	// selected is the relay the operator chose, "" until they choose one.
	selected func() string
}

// startSchedule builds the Director, its executors and the pump, starts both
// goroutines, and wires the producers and MasterControl into it.
//
// Why it is wired this way:
//   - The arbiter and the effector are the ones already running (T2.3): the
//     executors perform through nar and nar.mc, the pair the ticker's
//     takeovers use, so the band and the duck keep one owner each.
//   - The three lists are not one list (D-76, D-140). pool is the station's
//     candidates, what its Producer may offer. resolvable is what its Composer
//     may resolve: the pool plus whatever the operator requested anywhere in
//     the service radius (D-130), which is wider than the capped pool. watch is
//     the listener's, and the bed's cut-over is the monitor's rotation through
//     it, so cutTo resolves against watch, not the pool: resolved against the
//     pool, a watched place outside it would die silently as
//     schedule:tune-unknown.
//   - The console is told, never reads the Director (0.16.0 P2): publish is its
//     only source, called when the schedule settles.
//   - The producer is the ticker (T3.10b): it holds what an alert is and the
//     store of what was read aloud, asked both ways (I-7), and [M] reaches the
//     schedule so a muted read is never consumed in silence.
//   - A card's report kinds travel on the effect (R4b), so compose looks
//     nothing up from the published line-up, which would race the dispatch.
//   - The programme is a source swap on the broadcast engine (F-91, BD-9), read
//     by the deck; a pathless build has no deck, and the executor declines the
//     card by name rather than holding it on the air in silence.
//   - The line-up's depth is the console's slot count (D-40), from the console,
//     so the station never holds cards the operator cannot address.
//   - The spoken transitions' seam is left empty (D-163): ProgrammeReturn and
//     Announcement name no script, so the Director arranges none. To add them,
//     set ProgrammeReturn to transition/resume.txt, rendered by scriptText.
//   - The producers report into the schedule here (T3.10b), not at the caller:
//     left to the caller, the ticker's wiring is one line from being forgotten
//     and no hazard is ever read while every test stays green (D-12's shape).
//   - MasterControl declares ON AIR / STANDBY (MVS-D-78, FR-5.4), the one voice
//     for the operator, and the console is handed the control. Its send runs on
//     its own goroutine because this runs before the program's loop does: a
//     synchronous send would block until the loop reads it, and hangs
//     TestRunWithoutArgsStartsTheDashboard.
func startSchedule(ctx context.Context, nar *arbiter, scripts *script.Library, clock func() render.Clock, deck *radioDeck, pool, resolvable, watch func() []snapshot.LocationRef, tick *tickerDeck, publish func(tea.Msg), bed bedSeams) *schedule {
	if nar == nil {
		return nil // no arbiter, no schedule: there is nothing to perform through
	}
	if tick == nil {
		return nil // no producer, no arrivals: the rail would have nothing to read
	}
	x := newExecutors(executors{
		voice:     nar,
		publish:   publish, // the console's only source
		scripts:   scripts,
		clock:     clock,
		now:       time.Now,
		mc:        nar.mc,
		audible:   func() bool { return !nar.silent() },
		muted:     tick.muted.Load, // [M] reaches the schedule
		alert:     tick.alerts.get, // the producer holds what an alert is
		mark:      func(id string) { tick.seen.mark([]globalfeed.Event{{ID: id}}, time.Now()) },
		readAloud: tick.seen.has, // the same store, asked the other way (I-7)
		report: func(f lineup.Effect, why string) {
			radioDebugLog("schedule:declined:" + lineup.Describe(f) + ":" + why)
		},
		cutTo:    tuneTo(deck, watch), // the monitor's rotation, through the listener's watchlist
		bedLabel: bedLabelOf(watch),   // what the console calls it, from the same list (F-79)
		selected: bed.selected,        // what the operator chose (F-98, D-90)
		noteBed:  bed.note,
		compose:  composeFor(deck, resolvable), // the main track's words, against the pool and the requests (D-140)
		read:     readerFor(deck),              // the programme's reader (F-91, BD-9)
		propose:  proposeFrom(pool),            // what the Producer has to offer (D-40)
	})
	if x == nil {
		return nil // a seam was nil; newExecutors has already said which
	}
	run, cancel := context.WithCancel(ctx)
	p := newPump(lineup.New(lineup.Settings{Max: defaultBurstMax, Depth: tty.MainTrackSlots}, time.Now()), x.run,
		func(f lineup.Effect, v any) { radioDebugLog("schedule:fault:" + lineup.Describe(f)) })
	if p == nil {
		cancel()
		return nil
	}
	s := &schedule{pump: p, ctx: run, cancel: cancel, ticks: make(chan struct{}), executors: x}
	go p.loop(run)
	go s.tick(run)
	tick.mu.Lock()
	tick.emit = s.carry
	tick.mu.Unlock()
	if deck != nil {
		deck.mu.Lock()
		deck.emit = s.carry
		deck.mu.Unlock()
	}
	if nar != nil && nar.mc != nil {
		nar.mc.mu.Lock()
		nar.mc.carry = s.carry
		nar.mc.mu.Unlock()
		if publish != nil {
			go publish(tty.StationControlMsg{Control: nar.mc}) // on its own goroutine: the program's loop has not started (see Why)
		}
	}
	return s
}

// tuneTo resolves the key the Director named back to a location and cuts the
// bed over to it.
//
// THE UNEXPORTED tune, DELIBERATELY. Every tune the Director asks for is
// automatic — a dwell elapsed, a cycle ended — and lifting the alert duck on an
// automatic transition would bring the next location's report in at full volume
// over a breaking alert still reading. The duck has ONE OWNER (T2.3) rather
// than that distinction living in the case of an identifier, and this calls
// the path that never lifts it.
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
// ONE RESOLVER FOR BOTH CALLERS: the cut-over and the main-track composer ask
// the same question — two walks of the watchlist comparing the same key would
// be two places for "what is a location's identity" to drift.
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
// WHAT IS SHARED IS THE TEXT, NOT THE DELIVERY. A synth.Segment
// carries Key, Text, Role, SelfIntro and Pause; scriptFromSegments keeps Text
// and drops the rest, because lineup.Part has nowhere to put them. The source
// consumes all five. So the two paths cannot say different WORDS — and can
// still differ in which correspondent says them, whether an introduction is
// suppressed, and how long the pauses are.
//
// That is a live question for the flip and it is recorded there, not resolved
// here (04-development/p3-flip-design.md, G-7).
func composeFor(deck *radioDeck, watch func() []snapshot.LocationRef) func(context.Context, string, report.Set) ([]synth.Segment, error) {
	return func(ctx context.Context, ref string, want report.Set) ([]synth.Segment, error) {
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
		// EMPTY MEANS THE WHOLE REPORT. Every card the Director makes for itself
		// is in that state — only an operator's request names a subset — so the
		// zero value is the ordinary case, and reading it the other way would
		// compose a frame with nothing in it (mAX3).
		if want.Empty() {
			want = report.Everything()
		}
		return deck.segments(ctx, r, synth.VoiceToken, want)
	}
}

// carry hands the Director something that happened, from anywhere outside it.
//
// NAMED carry, NOT send, for the tool rather than the code: P10 resolves by
// NAME, so a `send` here collides with pump.send — which is genuinely in a call
// cycle — and reports this as recursion it has no part in. Those collisions are
// always between a method on one type and a method on another.
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
// returned — a fire-and-forget goroutine outliving the thing it writes to,
// which the Linux race gate fails.
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
