package app

// executors.go — the Director's effect executors (0.14.0 T2.2): adapters over
// the code that already works. The Director describes work as effects
// (platform/lineup); the pump dispatches them; this is what performs each one
// and reports what it learned as an event. IT DECIDES NOTHING — which card,
// in what order, for how long, is Step's. What lives here is only "how does
// this station do that".
//
// THE SET IS CLOSED (PL-6) AND WALKED IN FULL: every effect is either performed
// or DECLINED BY NAME. The duck, the restore and the tune are PERFORMED — this
// paragraph declined them by name until the BUILD-exit red team (R-3), having
// been written before T2.3 and T3.2 landed. What is still refused is a slot no
// producer proposes: the location report and the severe read. A refusal is
// loud, and a card-effect among them fails its card so the schedule re-plans
// rather than waiting for words that will never come (DR-21).

import (
	"context"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// executors performs effects. Every field is a seam into the running station,
// set once by whoever wires it; the executors hold no state of their own but
// the band record.
type executors struct {
	// voice is today's narration arbiter. Speak runs THROUGH it rather than
	// over render and play directly, so the duck keeps its one owner across
	// Phase 2 (RD-2): the arbiter ducks when a sequence takes the air and
	// restores when it leaves, exactly as the takeover does now.
	voice   *director
	scripts *script.Library // the spoken lines; nil = the built-in tree
	clock   func() render.Clock
	now     func() time.Time // injected (DR-20): a line composes "at Four Fifty PM" against it
	// mc is the one owner of both outputs — the band and the duck. The band was
	// written from here AND from app/ticker.go, two files constructing the same
	// takeover message, which is two carriers of one rule (D-1).
	mc *mastercontrol
	// compose builds a location report's segments for the card path (0.16.0
	// P3). It is the deck's own composition, reached as a SEAM rather than a
	// dependency: the executors know how to turn segments into a script and
	// nothing about how a report is assembled.
	compose func(ctx context.Context, ref string) ([]synth.Segment, error)

	// read performs a MAIN-TRACK card — the programme — and returns whether the
	// words ran out on their own (F-91, BD-9).
	//
	// A SECOND PERFORMER, AND THAT IS THE RULING RATHER THAN A CHOICE. The rail
	// reads through the arbiter because an alert speaks OVER whatever is on;
	// D-33 rules that a chosen read REPLACES the bed, so the programme is a
	// source swap on the broadcast engine and cannot travel the narration path.
	// The fork is `onTheRail`, in one place, and each side names the other.
	//
	// IT BLOCKS FOR THE LENGTH OF THE READ, like the arbiter's Run: the Speak
	// effect comes home Finished when the card has actually been said.
	//
	// Nil is a station with no broadcast engine — the pathless build and the
	// tests that wire no deck — and the executor declines by name.
	read func(ctx context.Context, v lineup.Speak) bool

	// propose asks the Producer what cards COULD exist, so the Director can top
	// the line-up up to its depth (D-40).
	//
	// A SEAM, LIKE compose, and for the same reason: the executors know how to
	// carry an offer and nothing about where locations come from. The producer
	// may offer more than the Director needs, may offer what is already
	// scheduled, and may offer on every publish — none of that costs anything,
	// because THE DIRECTOR HOLDS THE DEPTH. Proposing is cheap by construction:
	// a proposal is a name and a headline, and DR-7 puts the words at standby.
	//
	// Nil is a station with no producer, which offers nothing.
	propose func() []lineup.Proposal

	// publish hands the settled schedule to the console. Nil when no surface
	// is listening, which is every build before 0.16.0 and every test that
	// does not care.
	publish func(tea.Msg)

	// audible is false when there is nothing to hear — no device, no voice. The
	// visuals still run and nothing dips: a station with no sound card still
	// shows the alert on the band.
	audible func() bool

	// muted is the LISTENER saying "do not speak to me" (`[M]`), which is a
	// different question from whether the machine can make a sound.
	//
	// CONFLATING THE TWO WAS THE TRAP. A read is declined when muted, so the
	// alerts stay new and are offered again; a read on a voiceless machine goes
	// ahead, so the band still shows the hazard. One predicate for both would
	// either swallow a hazard on mute or blank the marquee on a laptop with no
	// audio, depending on which way it was pointed.
	muted func() bool

	// alert is the producer's record for ONE of its own ids — an alert, not a
	// card. A takeover is one card made of several alerts (MVS-D-77), so the
	// card cannot be the key; the Director carries the ORDER on the effect
	// (BuildCard.Refs) and this answers what each id IS. The card stays
	// domain-free (DR-1), the way the [space] read keeps its own row.
	alert func(id string) (globalfeed.Event, bool)

	// mark records that an alert was READ ALOUD, so no later cycle offers it
	// again. It is the producer's store, not the schedule's: the Director's own
	// record is the card's state, which ends when the lineup lets the card go.
	//
	// WITHOUT IT EVERY BURST REPEATS. The producer filters its arrivals against
	// this store before it sends them, so an unmarked alert arrives again on the
	// next cycle and is read a second time.
	//
	// IT TAKES THE ID, NOT THE EVENT (red team 2026-09-05). The seen-store reads
	// only the id, and taking an event meant looking one up to obtain the id
	// already in hand — a round trip that can MISS: if the record was evicted
	// between the burst arriving and its read, the alert went unmarked and was
	// read a second time. The eviction bound was reasoned about for the build
	// path and not for this one.
	mark func(id string)

	// readAloud is whether one alert has ALREADY been read — the same store
	// mark writes to, asked in the other direction.
	//
	// TWO CARDS CAN CARRY THE SAME ALERT (red team 2026-09-05, I-7). The
	// producer filters its arrivals against the store before it sends them, and
	// the Director refuses a card whose ID it already holds — but a burst's ID
	// is its LEAD's, so a later burst with a different lead can overlap the one
	// on air, and the store is written line by line AS EACH IS SAID, so an
	// alert queued but not yet spoken is in neither filter. The overlapping
	// alerts would then be read a second time, back to back. On a weather radio
	// an alert that repeats itself is an alert a listener stops trusting.
	//
	// Narrow in Observer, where one read rarely outlives a cycle. STRUCTURAL in
	// Broadcaster: OffAir holds the rail while nothing is marked, so every cycle
	// queues another overlapping burst and the pile reads back on resume.
	readAloud func(id string) bool

	// report sees every effect that could not be performed, with why. It is
	// the DR-23 record's input: a declined effect is the only sign the wiring
	// and the schedule disagree.
	report func(lineup.Effect, string)

	// cutTo cuts the bed over to a location, by the key the Director names.
	//
	// NAMED cutTo, NOT tune, and the reason is the tool rather than the code:
	// P10 resolves by NAME, so a field called `tune` collides with
	// radioDeck.tune — which genuinely sits in a call cycle and carries its own
	// exemption — and the collision reports this seam as recursion it has no
	// part in. Fourth rename in this release for the same false positive.
	//
	// IT MUST NOT LIFT THE DUCK, and that is the whole reason this seam is a
	// function rather than the deck itself. Every tune the Director asks for is
	// AUTOMATIC — the dwell elapsed, a cycle ended — and nobody pressed
	// anything. Lifting the dip here would bring the next location's report in
	// at full volume over a breaking alert still reading, which is the defect a
	// capital letter used to carry (radioDeck.Tune vs tune) and which T2.3
	// removed by giving the duck one owner. The wiring calls the deck's
	// unexported tune, and this comment is here so a future caller does not
	// reach for the exported one.
	cutTo func(ref string)

	// bedLabel turns the bed's ref into the words the console shows for it
	// (F-79). Nil in the modes with no radio, where the bed is never tuned.
	bedLabel func(ref string) string

	// noteBed records whether the bed is carrying, so the relay SELECTOR can
	// publish a truthful row without asking the Director (F-79, D-78). Nil
	// where there is no console.
	noteBed func(carrying bool)

	// escalate is the ONE place a fault reaches a person (DR-21). Nil is not
	// allowed: newExecutors refuses it, because a fault channel wired to
	// nothing is the failure this whole requirement exists to remove — the
	// station stops and nobody is told.
	escalate func(reason string)

	// band is the post-hoc record of what the band was asked (DR-18): a cue
	// is fire-and-trust, so this is what a test and a diagnostic read after.
	band *bandRecord
}

// newExecutors checks every seam once, here, rather than on the first effect —
// where a nil would panic on a worker, be contained, and read as a fault of
// the card instead of the wiring bug it is.
func newExecutors(x executors) *executors {
	if err := invariant.Check(x.voice != nil, "executors are built over a narrator, silent or not"); err != nil {
		return nil
	}
	if err := invariant.Check(x.clock != nil && x.now != nil, "executors are built with a clock and a time"); err != nil {
		return nil
	}
	if err := invariant.Check(x.mc != nil && x.audible != nil, "executors are built with an effector and a mute state"); err != nil {
		return nil
	}
	if err := invariant.Check(x.alert != nil, "executors are built with a producer to ask"); err != nil {
		return nil
	}
	if err := invariant.Check(x.mark != nil, "executors are built with somewhere to record what was read aloud"); err != nil {
		return nil
	}
	if err := invariant.Check(x.readAloud != nil, "executors are built with a way to ask what was already read"); err != nil {
		return nil
	}
	if err := invariant.Check(x.muted != nil, "executors are built with the listener's mute to ask"); err != nil {
		return nil
	}
	if err := invariant.Check(x.report != nil, "executors are built with somewhere to report a declined effect"); err != nil {
		return nil
	}
	if err := invariant.Check(x.cutTo != nil, "executors are built with a bed to cut over"); err != nil {
		return nil
	}
	if err := invariant.Check(x.escalate != nil, "executors are built with somewhere a stopped schedule can be reported"); err != nil {
		return nil
	}
	x.band = &bandRecord{}
	return &x
}

// offer is the producer's answer to a settled schedule, or nothing.
//
// AN EMPTY OFFER IS NOT AN EVENT. Returning `Offered{}` with no proposals would
// be a step the Director takes for no reason, once per publish, for ever.
func (x *executors) offer() []lineup.Event {
	if x.propose == nil {
		return nil // a station with no producer offers nothing
	}
	ps := x.propose()
	if len(ps) == 0 {
		return nil
	}
	return []lineup.Event{lineup.Offered{Proposals: ps}}
}

// run is the pump's runEffect: one effect in, what it learned out. It may
// block for as long as the work takes — it runs on a worker, never the pump.
func (x *executors) run(ctx context.Context, f lineup.Effect) []lineup.Event {
	if err := invariant.Check(f != nil, "an effect is something"); err != nil {
		return nil
	}
	switch v := f.(type) {
	case lineup.BuildCard:
		return x.build(ctx, v)
	case lineup.Speak:
		return x.speak(ctx, v)
	case lineup.CueTicker:
		return x.runCue(v)
	case lineup.ReleaseTicker:
		return x.runRelease(v)
	case lineup.Publish:
		// THE CONSOLE READS THE LINEUP NOW (0.16.0 P2). This executor was
		// deliberately empty through 0.15.0 and said so — the seam existed and
		// had no consumer, which is a very different thing from a no-op that
		// pretends to be wired.
		//
		// BOTH FACTS TRAVEL TOGETHER because the effect carries both. A console
		// told the schedule and the station's state separately can hold a torn
		// pair — a new lineup beside a stale power — and showing what is
		// actually going to air is the console's whole job.
		if x.publish != nil {
			x.publish(tty.LineupMsg{Lineup: v.Lineup})
			x.publish(tty.StationMsg{Power: v.Power})
			// AND WHAT IT IS RIDING ON (F-79, closed at D-78). The third fact
			// D-62 consolidated into one region, published with the other two so
			// the console cannot hold a torn set.
			x.publish(tty.BedMsg{Relay: x.describeBed(v.Bed), Carrying: v.Bed.Carrying})
			if x.noteBed != nil {
				// THE SELECTOR NEEDS TO KNOW TOO. It publishes its own BedMsg
				// when the operator moves, and a message that guessed at
				// `Carrying` would flicker the row between the two publishers.
				x.noteBed(v.Bed.Carrying)
			}
		}
		// AND THE PRODUCER IS ASKED TO TOP THE LINE-UP OFF (D-40). No new
		// effect: `run` already returns what an effect learned, and a publish is
		// the moment the schedule has SETTLED — which is exactly when the
		// producer can see what the line-up still needs.
		//
		// THE CHAIN IS SELF-LIMITING BY THE DEPTH, not by a counter. Publish →
		// Offered → the track fills → settle publishes → Offered again → nothing
		// left to admit → `onOffered` returns NO EFFECTS, so there is no publish
		// and the chain has nowhere to go.
		return x.offer()
	// DUCK AND RESTORE ARE WIRED AND UNREACHED (red team 2026-09-05, I-5).
	//
	// Nothing in production constructs either effect — the Director emits
	// BuildCard, Speak, CueTicker, ReleaseTicker, Publish, Tune and Escalate,
	// and no more — so mastercontrol.held is never true, and hold(), unhold(),
	// takeBack()'s held branch and director.releaseBed() are all inert. That is
	// stated here because those functions carry some of the heaviest comments
	// in the release (the F-D5 critical section, the lock-order warning) in the
	// present tense, and a reader is entitled to know the path is not taken.
	//
	// IT IS HARMLESS TODAY FOR A REASON WORTH WRITING DOWN. MVS-D-67 — one dip
	// per drain, the rail owning the bed until its tail has played — was about
	// a rail of MANY cards bouncing the bed between them. MVS-D-77 made a burst
	// ONE card, so one Speak is one arbiter sequence and the per-sequence
	// giveWay/takeBack dips once anyway. The ruling's problem dissolved rather
	// than being solved by `held`. F-27's design is built on this path, so it
	// is kept rather than deleted (AP-DEAD-01), and whoever wires it must know
	// it has never run.
	case lineup.Duck:
		// The bed gives way. IDEMPOTENT AT THE OWNER, so this and the narration
		// arbiter's own duck cannot dip twice between them.
		x.mc.hold()
		return nil
	case lineup.Restore:
		// THE ASYMMETRY IS DELIBERATE AND LOAD-BEARING — do not "tidy" this to
		// x.mc.unhold(). Acquiring is one critical section inside mastercontrol
		// (hold dips and claims together, which is the F-D5 fix). RELEASING is
		// not mastercontrol's decision to make: the arbiter takes d.mu and then
		// mc.mu, so a release that took mc.mu and then asked the arbiter would
		// invert the lock order and deadlock. director.releaseBed unholds and
		// re-settles under d.mu, which is why the pair reads as two owners and
		// is in fact one owner per lock.
		x.voice.releaseBed()
		return nil
	case lineup.Tune:
		return x.runTune(v)
	case lineup.Escalate:
		// GRADED SURFACING, and the grade was decided by the Director. Anything
		// it could route around never becomes an Escalate at all, so there is no
		// second judgement here — a fault reaching this line has already left
		// the station with nothing to play.
		x.escalate(v.Reason)
		return nil
	}
	// The set is closed, so this is unreachable for anything declared today.
	// It is the safe direction for a member added later without an executor:
	// reported, never silently dropped.
	return x.decline(f, "", "the effect set grew a member with no executor")
}

// build composes a card's words at standby (DR-7).
func (x *executors) build(ctx context.Context, v lineup.BuildCard) []lineup.Event {
	switch v.Slot {
	case lineup.BreakingAlert:
		// THE COMPOSER, ON THE DIRECTOR'S ORDER (MVS-D-77, S-7). The Refs are
		// the burst's alerts in read order — the Director planned them — and
		// this asks the producer what each one IS. Neither half is guessed at:
		// composing from a re-plan here would be a second planner, and holding
		// the events on the card would put the domain inside the schedule.
		evs, ok := x.eventsFor(v.Refs)
		if !ok {
			return x.decline(v, v.ID, "the producer holds no alert for this card")
		}
		// A BURST IS SEVERAL ALERTS, and that is what earns the head and the
		// tail. One alert carries its own broadcast tail inside its line, so it
		// gets neither — the same rule composeTakeover states, asked here
		// because only the effect knows how many alerts this card reads.
		sc := composeTakeover(x.scripts, evs, len(evs) > 1, v.Divert, x.clock(), x.now())
		// A Built with no words trips the Director's own invariant and the
		// card would sit at standby for ever. Today an empty line is a silent
		// hold; under the lineup it is a card that cannot be delivered.
		if sc.Empty() {
			return x.decline(v, v.ID, "the script rendered nothing to say")
		}
		return []lineup.Event{lineup.Built{ID: v.ID, Script: sc}}
	case lineup.LocationReport:
		// THE MAIN TRACK ARRIVED (0.16.0 P3). This decline read "read by the
		// main track, which arrives with T3.2" from 0.14.0 until now.
		//
		// THE EXECUTOR KNOWS NOTHING ABOUT HOW A REPORT IS ASSEMBLED. It asks
		// the composer for segments and turns them into a script; the deck
		// owns what a report IS, exactly as the producer owns what an alert is
		// on the rail path.
		if x.compose == nil {
			return x.decline(v, v.ID, "no composer is wired for the main track")
		}
		segs, err := x.compose(ctx, v.Subject)
		if err != nil {
			return x.decline(v, v.ID, "the report could not be composed: "+err.Error())
		}
		sc := scriptFromSegments(segs)
		if sc.Empty() {
			// A CARD ON THE AIR WITH NO WORDS IS SILENCE under a callout the
			// band has already promised (DR-18) — the same rule the takeover
			// path states two cases above.
			return x.decline(v, v.ID, "the report composed nothing to say")
		}
		return []lineup.Event{lineup.Built{ID: v.ID, Script: sc}}
	case lineup.SevereRead:
		return x.decline(v, v.ID, "read by the severe window's own reader, not the schedule")
	}
	return x.decline(v, v.ID, "a structural card's words are fixed at proposal; nothing builds them")
}

// speak reads a card's words through the narrator and comes home Finished —
// or with nothing, when the context ended mid-read: the pump is stopping, and
// a Finished for words never finished would tell the schedule a read happened
// that did not.
func (x *executors) speak(ctx context.Context, v lineup.Speak) []lineup.Event {
	// Card.To(OnAir) refuses this already; refused again here because this
	// is the last thing between the schedule and a silent hold with a callout
	// already promised (DR-18).
	//
	// ASKED BEFORE THE FORK, because it is true of both readers: a card with no
	// words is dead air whichever thing would have performed it.
	if v.Script.Empty() {
		return x.decline(v, v.ID, "a card took the air with nothing to say")
	}
	// ONLY THE RAIL READS THROUGH THE ARBITER (D-33, HUM LEAD 2026-09-09).
	//
	// The main track was admitted here for one release and it was wrong: a
	// chosen read REPLACES the bed rather than speaking over it, so it is not a
	// narration and it has no business on the narration path. What it IS is the
	// engine Source adapter (BD-9), and that is what `broadcast` performs.
	//
	// THE FORK IS THE TRACK, NOT THE SLOT, and the reason is on the effect: a
	// transition belongs to whichever lane the card it bookends is on.
	if v.Track != lineup.AlertRail {
		return x.broadcast(ctx, v)
	}
	// A CARD IS NEVER CONSUMED IN SILENCE (MVS-D-78). The producer already
	// refuses to send a burst while the listener is muted, but `[M]` can land in
	// the moment between an arrival and its read — and reading it inaudibly
	// would cue the band, MARK EVERY ALERT READ and finish, so a tornado warning
	// would be swallowed and never sounded, which is the exact defect that
	// ruling exists to prevent.
	//
	// DECLINED RATHER THAN HELD. The alerts were never marked, so they are still
	// new and the producer offers them again on its next cycle — the schedule
	// routes around the fault (DR-21) instead of standing by for a listener who
	// may not come back for an hour.
	//
	// ASKED SEPARATELY FROM `audible`, which is about whether the MACHINE can
	// make a sound. A voiceless station still reads: the words play nothing and
	// the band still shows the hazard.
	if x.muted() {
		return x.decline(v, v.ID, "the listener is muted; the alerts stay new and will be offered again")
	}
	read := false
	// The takeover's class and role, for every rail card: the bars keep to the
	// broadcast (vizFor), the breaking correspondent reads (FR-3).
	//
	// THROUGH THE ONE READER (T3.8). It owns MVS-D-72's shape — the tone, the
	// pauses, the overlap that keeps a render out of every gap — and the live
	// takeover reads through the same function. Two implementations of a ruling
	// the HUM LEAD found BY EAR would be two places for it to drift.
	// ONE CLASS REACHES HERE, because one lane does (D-33). The rotation had a
	// class of its own for one release; it was retired with the design that
	// put the programme on the narration path at all.
	class, role := narrateBreaking, cast.Breaking
	x.voice.Run(ctx, class, role, x.audible(), func(ctx context.Context, s *speaker) {
		read = readScript(s, v.Script, readHooks{
			cue: func(ref string) {
				// THE READ'S OWN CUES ARE RECORDED TOO (red team 2026-09-05).
				// The band record's stated job is "was this card cued, and
				// released?", and every cue a live station makes comes from
				// here — so it read as a list of releases with no cues.
				if x.cueFor(ref) {
					x.band.note("cue(" + ref + ")")
				}
			},
			// MARKED AS EACH LINE IS SAID, not when the card ends. A takeover
			// cut short mid-burst has genuinely read the alerts it got to, and
			// they must not be offered again; the ones it did not reach must.
			mark: x.mark,
		})
	})
	if !read {
		// A READ THAT ENDED EARLY SAYS SO, ALWAYS (DR-24).
		//
		// FAILED RATHER THAN FINISHED, because the words did not finish: a
		// Finished would tell the schedule a read happened that did not. And
		// rather than NOTHING, because a card that says nothing stays ON AIR for
		// ever with the band holding a callout for a read that has stopped —
		// DR-24's original defect, exactly.
		//
		// UNCONDITIONAL, including when the pump is stopping. Deciding here
		// whether the schedule is still worth telling is the kind of judgement
		// DR-24 exists to remove: the event is cheap, a stopping pump drops it,
		// and the alternative is a release that fires on all paths but one.
		// ROUTED: a read ends early because the listener pressed esc, a takeover
		// pre-empted it, or the pump is stopping. None of those is the station
		// failing, and a "the relay is silent" modal for any of them would be
		// the noise regression fault.go exists to avoid (I-2).
		return []lineup.Event{lineup.Failed{ID: v.ID, Reason: "the read ended before the words did", Routed: true}}
	}
	return []lineup.Event{lineup.Finished{ID: v.ID}}
}

// broadcast performs a MAIN-TRACK card: the programme, on the broadcast engine
// (F-91, BD-9). It is `speak`'s other half, and the two are one function split
// at `onTheRail` rather than two readers that happen to be called from the same
// place.
//
// THE LISTENER'S MUTE IS NOT ASKED HERE, and the asymmetry with the rail is the
// point. `[M]` is "do not speak ALERTS to me" — the rail path declines under it
// because reading a burst inaudibly would MARK every alert read and swallow a
// tornado warning (MVS-D-78). A location report marks nothing and consumes
// nothing: declining it would silence a station the operator has deliberately
// put ON AIR, over a control that belongs to the other programme. The air is
// what decides who performs, and the Director has already asked it
// (`advances(MainTrack)`).
func (x *executors) broadcast(ctx context.Context, v lineup.Speak) []lineup.Event {
	// A HAZARD IS NEVER READ AS THE PROGRAMME, and this is the one cross-check
	// the track cannot perform on itself: `Track`'s zero value is MainTrack, so
	// an effect built without one arrives HERE by default. A rail card read
	// down this path would lose its attention tone — the engine source plays
	// words, and the Script's tone is the arbiter's to sound — and its
	// per-alert callouts with it: a tornado warning, delivered as the weather.
	//
	// The SLOT is what makes that constructible and therefore testable. It is
	// the direction a mistake would actually go, which is why there is no
	// matching check on the rail's side.
	if onTheRail(v.Slot) {
		return x.decline(v, v.ID, "a rail card reached the programme's reader: its tone and its callouts would be lost")
	}
	if x.read == nil {
		return x.decline(v, v.ID, "no reader for the main track: this station has no broadcast engine")
	}
	if !x.read(ctx, v) {
		// THE SAME VERDICT THE RAIL RETURNS, and for the same reason (DR-24): a
		// Finished would tell the schedule a read happened that did not, and a
		// card that says nothing at all stays ON AIR for ever. Routed, because
		// every way this ends early — the operator went to standby, the pump is
		// stopping, the voice could not render a line — is either deliberate or
		// already reported by the path that raised it (I-2).
		return []lineup.Event{lineup.Failed{ID: v.ID, Reason: "the read ended before the words did", Routed: true}}
	}
	return []lineup.Event{lineup.Finished{ID: v.ID}}
}

// runTune cuts the bed over to the location the Director named (T3.2b).
//
// FIRE AND TRUST, like the cue: the schedule does not wait for a relay to
// resolve, which can take seconds, and a Director that blocked on it would stop
// stepping while a station connected. The deck reports where the bed actually
// landed through a Tuned event, and the dwell starts from THAT rather than from
// here — a resolve charged to the listener's five minutes would cut every turn
// short by however slow the network was.
func (x *executors) runTune(v lineup.Tune) []lineup.Event {
	if v.Ref == "" {
		return x.decline(v, "", "a tune named no location")
	}
	x.cutTo(v.Ref)
	return nil
}

// onTheRail reports whether a slot is one the alert rail reads through the
// narrator. Named here, in the executor that needs the distinction, rather than
// as a registry column nobody else asks for.
//
// IT NO LONGER DECIDES WHO READS (F-91). It answered "is there a reader for
// this at all" for one release, and everything it excluded was declined; both
// lanes perform now, and which one a card is on is the TRACK, carried on the
// effect. What is left here is the question only the SLOT can answer: does this
// kind of card put its own callout up as it reads.
func onTheRail(s lineup.Slot) bool { return s == lineup.BreakingAlert }

// cue asks the band to show the callout for the card taking the air (DR-18).
// Fire and trust: it returns no event, and a band that cannot be cued is a
// diagnostic, never a reason to take the card off the air — the words still
// read.
func (x *executors) runCue(v lineup.CueTicker) []lineup.Event {
	// A TAKEOVER CUES ITSELF, LINE BY LINE (T3.10b).
	//
	// A burst is ONE card made of several alerts (MVS-D-77) and the band shows a
	// different callout as each is spoken, so one callout for the whole card
	// would be wrong for every line after the first. Those cues come from INSIDE
	// the read (cueFor), which is also what keeps them from outliving it: issued
	// from here they would go up for a read the arbiter then cancelled, which is
	// the stale-band defect DR-24 exists to prevent, arrived at from the other
	// side.
	//
	// The card is NOT looked up before this returns. Asking the producer for a
	// burst's id and finding nothing is the normal case, and a report on every
	// takeover would bury the one that means something.
	if onTheRail(v.Slot) {
		return nil
	}
	// THROUGH THE ONE CARRIER (D-1, red team 2026-09-05). This repeated cueFor's
	// three steps — look the record up, give up quietly on a miss, cue through
	// the band's one owner — differing only in the report and the record. Two
	// copies of "put the callout up for an alert id" is two places for it to
	// drift, and the modularity standard says extract at the second caller.
	if !x.cueFor(v.ID) {
		x.report(v, "the producer holds no alert for this card; the band keeps what it shows")
		return nil
	}
	x.band.note(lineup.Describe(v))
	return nil
}

// release gives the band its rotation back — the cue's other half (DR-24).
func (x *executors) runRelease(v lineup.ReleaseTicker) []lineup.Event {
	x.mc.clearBand()
	x.band.note(lineup.Describe(v))
	return nil
}

// decline reports an effect that will not be performed and, when it named a
// card, fails that card so the schedule re-plans around it (DR-21).
func (x *executors) decline(f lineup.Effect, id, why string) []lineup.Event {
	x.report(f, why)
	if id == "" {
		return nil
	}
	// A DECLINE IS ROUTED BY DEFINITION: the executor refused BY NAME and said
	// why, and the producer offers the alerts again. It is not the station
	// going quiet, which is what the fault window is for (I-2).
	return []lineup.Event{lineup.Failed{ID: id, Reason: why, Routed: true}}
}

// bandRecord is what the band was asked, most recent last, bounded (P10-03).
// A ring rather than a log: it answers "was this card cued, and released?"
// for a test and for the diagnostic, and it never grows with the day.
type bandRecord struct {
	mu    sync.Mutex
	lines []string
}

// bandRecordCap is how many band requests the record keeps — a few bursts'
// worth of cue/release pairs, which is as far back as a diagnostic looks.
const bandRecordCap = 32

// note records one request.
func (b *bandRecord) note(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, line)
	// The trim IS the bound; a check beside it saying the same thing would be a
	// second carrier that hides the first going missing (build log, m83).
	if len(b.lines) > bandRecordCap {
		b.lines = b.lines[len(b.lines)-bandRecordCap:]
	}
}

// has reports whether the record holds this request.
func (b *bandRecord) has(line string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, l := range b.lines { // bounded by the cap (P10-02)
		if l == line {
			return true
		}
	}
	return false
}

// recent is the record, oldest first, as a copy.
func (b *bandRecord) recent() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.lines...)
}

// cueFor puts the band's callout up for the producer's record behind a part.
// The record is asked for, never constructed here: the band has ONE owner and
// the producer owns what an alert IS (D-1).
func (x *executors) cueFor(ref string) bool {
	if ref == "" {
		return false
	}
	e, ok := x.alert(ref)
	if !ok {
		return false
	}
	x.mc.cue(breakingItem(e))
	return true
}

// eventsFor is the producer's records for a card's refs, in the card's order.
//
// ALL OR NOTHING. A burst missing one of its alerts would read as a shorter
// burst than the one the schedule planned and the operator saw — quieter than
// the truth, with nothing to say a line went absent. A card that cannot be
// composed in full is declined and reported.
//
// AND IT RE-ASKS WHETHER THE ALERT IS STILL LIVE (red team 2026-09-05, I-8).
// The record is snapshotted when the burst ARRIVES and the card is composed
// later, so an alert that expired in between was read aloud as current, with
// its original "until" time in the sentence. lineup.StaleAfter does not cover
// this: it bounds BuiltAt — how old the composed WORDS are — and firstStale
// skips any card whose BuiltAt is zero, which is every card still at Admitted.
// It bounded the sentence and not the fact. Seconds of exposure in Observer;
// unbounded behind an emergency overrun, a [space] read, or Broadcaster's
// OffAir. stale.go's own header is the argument: "a station that says something
// false with confidence is the failure this whole release is about."
//
// DECLINING IS SELF-HEALING, which is why it is the same all-or-nothing exit:
// nothing was marked, so the producer offers the burst again next cycle and
// globalfeed.Active has dropped the expired alert by then.
func (x *executors) eventsFor(refs []string) ([]globalfeed.Event, bool) {
	if len(refs) == 0 {
		return nil, false
	}
	now := x.now()
	out := make([]globalfeed.Event, 0, len(refs))
	for _, ref := range refs { // bounded by the card (P10-02)
		e, ok := x.alert(ref)
		if !ok {
			return nil, false
		}
		// THE SAME TEST globalfeed.Active APPLIES, said the same way: an alert
		// with no Until (a quake's instant) never expires, and one expiring
		// exactly now is KEPT — the boundary errs towards telling the listener.
		if !e.Until.IsZero() && now.After(e.Until) {
			return nil, false
		}
		// ALREADY SAID IS NOT SAID AGAIN (I-7). Declining is the same
		// self-healing exit as the two above: nothing was marked, so the
		// producer re-offers and its own unread filter drops the overlap.
		if x.readAloud(ref) {
			return nil, false
		}
		out = append(out, e)
	}
	return out, true
}

// describeBed turns the bed's ref into the row the operator reads.
//
// THE APP IS WHERE A REF BECOMES A PLACE AGAIN (DR-1), which is `refFor`'s own
// rule: the card and the bed carry identifiers, and what an identifier MEANS is
// the app's. A Director that knew a relay's call sign would be a Director that
// knew about radios.
//
// NOTHING TUNED READS AS NOTHING, not as a blank: the console has its own words
// for that, and inventing a second set here would be two answers to one state.
func (x *executors) describeBed(b lineup.BedState) string {
	if b.Ref == "" || x.bedLabel == nil {
		return ""
	}
	return x.bedLabel(b.Ref)
}
