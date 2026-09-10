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

	// playReport plays a composed report and BLOCKS until it ends, reporting
	// whether it reached its sign-off (0.16.0 P3(d), Shape B).
	//
	// NAMED DIFFERENTLY FROM THE DECK METHOD IT REACHES, deliberately. P10
	// resolves by NAME, so a seam here called `readReport` and a method there
	// called `readReport` are read as one node and the pair reports as
	// recursion — the sixth instance of that false positive in this release,
	// and every one of them a method on one type colliding with a method on
	// another. The rename is the cheaper half of that trade.
	//
	// THE ARBITER OWNS WHO SPEAKS; THE SOURCE STILL OWNS HOW A REPORT IS
	// SPOKEN. This runs inside voice.Run, so the air is held for the report's
	// whole length and a takeover gives way exactly as it always has — through
	// the engine, which HOLDS a rendered cycle and DIPS a live relay. Reading a
	// report through the narrator instead would have replaced the player and
	// taken that rule with it.
	playReport func(ctx context.Context, ref string, segs []synth.Segment) bool

	// held is the words a built card is waiting to speak, by card id — the
	// SEGMENTS, which carry the roles, the self-introductions and the pauses
	// that a lineup.Script cannot (red team 2026-09-09, finding 5). The script
	// on the card is what the console DISPLAYS; this is what is voiced, and
	// both come from one composition so the two cannot disagree.
	//
	// BOUNDED (P10-03) by heldSegmentsCap. Preparation runs one ahead and only
	// one, so the live count is the card on the air plus the one standing by;
	// the cap is what stops a card that was built and then dropped from holding
	// its report for the life of the process.
	held *segmentStore

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
// seamsPresent is the wiring contract, checked ONCE and in one place.
//
// SPLIT OUT OF newExecutors at P3(d), when the main track's three seams became
// required and pushed it past P10-04's branch bound. That bound is doing its
// job here: a constructor whose only remaining decision is "was the wiring
// complete" reads better than one that also answers it.
//
// EVERY FAILURE NAMES THE SEAM. A nil seam would panic on the first effect, on
// a worker, where the panic is contained and read as a fault of the CARD rather
// than as the wiring bug it is.
func seamsPresent(x executors) bool {
	// A TABLE, NOT A CHAIN. Eleven `if` statements in a row is eleven branches
	// and P10-04's bound is 15 — the merge's three seams took it past, and
	// splitting the function merely moved the count. Walking a list is ONE
	// branch, and it reads as what it is: the wiring contract, in order, each
	// row saying what is missing in the words the reader needs.
	for _, seam := range []struct {
		present bool
		says    string
	}{
		{x.voice != nil, "executors are built over a narrator, silent or not"},
		{x.clock != nil && x.now != nil, "executors are built with a clock and a time"},
		{x.mc != nil && x.audible != nil, "executors are built with an effector and a mute state"},
		{x.alert != nil, "executors are built with a producer to ask"},
		{x.mark != nil, "executors are built with somewhere to record what was read aloud"},
		{x.readAloud != nil, "executors are built with a way to ask what was already read"},
		{x.muted != nil, "executors are built with the listener's mute to ask"},
		{x.report != nil, "executors are built with somewhere to report a declined effect"},
		{x.cutTo != nil, "executors are built with a bed to cut over"},
		{x.escalate != nil, "executors are built with somewhere a stopped schedule can be reported"},
		// THE MAIN TRACK'S THREE (0.16.0 P3(d)). With the deck's direct path
		// gone, the schedule is the ONLY way a location report reaches the air
		// — so a nil composer, a nil player or a nil store is not a degraded
		// station, it is a station whose rotation is permanently silent while
		// its lineup fills up. They were optional while the direct path still
		// ran; that window closed with it.
		{x.compose != nil, "executors are built with a way to compose a report"},
		{x.held != nil, "executors are built with somewhere to keep a composed report"},
		{x.playReport != nil, "executors are built with something to play a report"},
	} { // bounded by the list (P10-02)
		if err := invariant.Check(seam.present, seam.says); err != nil {
			return false
		}
	}
	return true
}

func newExecutors(x executors) *executors {
	if !seamsPresent(x) {
		return nil // seamsPresent has already named the one that was missing
	}
	x.band = &bandRecord{}
	return &x
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
		}
		return nil
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
		segs, err := x.compose(ctx, v.Subject)
		if err != nil {
			return x.decline(v, v.ID, "the report could not be composed: "+err.Error())
		}
		// COMPOSED ONCE, USED TWICE — as the card's script, which the console
		// DISPLAYS, and as the segments, which the source VOICES. The script
		// keeps only the text; the segments carry the roles, the
		// self-introductions and the pauses that a lineup.Part has nowhere to
		// put. Re-composing at the air would be eleven more requests and a
		// second answer to a question already asked.
		x.held.put(v.ID, segs)
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
	if !onTheRail(v.Slot) && v.Slot != lineup.LocationReport {
		return x.decline(v, v.ID, "no reader for this slot: only the rail and the main track read")
	}
	// Card.To(OnAir) refuses this already; refused again here because this
	// is the last thing between the schedule and a silent hold with a callout
	// already promised (DR-18).
	if v.Script.Empty() {
		return x.decline(v, v.ID, "a card took the air with nothing to say")
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
	//
	// THE RAIL'S RULE, AND ONLY THE RAIL'S (0.16.0 P3(d)). Every word above is
	// about alerts, and the rotation is not one: `[M]` is "do not read me
	// hazards", and it has never silenced the broadcast — the radio plays
	// through it today. Applying it to a location report would have made the
	// merge turn the mute key into a stop button.
	if onTheRail(v.Slot) && x.muted() {
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
	// THE CLASS AND THE VOICE FOLLOW THE LANE (0.16.0 P3).
	//
	// A takeover reads as narrateBreaking in the breaking correspondent's
	// voice; the rotation reads as narrateRotation — the LOWEST class — in the
	// standard voice, because it is the programme and everything interrupts
	// it. That is the whole reason the class was added, and the arbiter needed
	// nothing: it already suspends a lower class for a higher one and resumes
	// it after.
	class, role := narrateBreaking, cast.Breaking
	if v.Slot == lineup.LocationReport {
		class, role = narrateRotation, cast.Standard
	}
	// THE ROTATION IS PLAYED BY ITS SOURCE, UNDER THE ARBITER (0.16.0 P3(d),
	// Shape B, ratified). The air is taken here and held for the report's whole
	// length, so the schedule owns WHO speaks — and the report is still voiced
	// by the synth source, so everything about HOW it is spoken is unchanged:
	// the per-segment marquee, the cast, the correspondent handoffs,
	// repeat-one, the player row, and the engine's give-way rule, which HOLDS a
	// rendered cycle and DIPS a live relay.
	//
	// ITS WORDS ARE THE SEGMENTS, NOT THE SCRIPT. The script is what the
	// console displays; both come from the one composition the build made.
	if v.Slot == lineup.LocationReport {
		segs, ok := x.held.take(v.ID)
		if !ok || x.playReport == nil {
			// A card whose report is gone, or a station with no audio deck.
			// DECLINED rather than held: the schedule routes around it and
			// offers the location again on the next turn (DR-21).
			return x.decline(v, v.ID, "no composed report is waiting for this card")
		}
		x.voice.Run(ctx, class, role, x.audible(), func(ctx context.Context, _ *speaker) {
			read = x.playReport(ctx, v.Subject, segs)
		})
		return x.leaveTheAir(v, read)
	}
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
	return x.leaveTheAir(v, read)
}

// leaveTheAir is how a read comes home, whichever path performed it.
//
// EXTRACTED AT THE SECOND CALLER (0.16.0 P3(d)): the rotation is played by its
// source and a takeover is read line by line, and both have to end the card the
// same way. Two copies of DR-24's rule is two places for it to drift, and the
// half that drifts is the half that fires rarely.
func (x *executors) leaveTheAir(v lineup.Speak, read bool) []lineup.Event {
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
// as a registry column nobody else asks for: the main track's slots arrive
// with the absorb that reads them (T3.2), and this list shrinks then.
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

// heldSegmentsCap bounds the composed-report store. Preparation runs one ahead
// and only one, so two is the working count; the slack is for a card dropped
// between its build and its read.
const heldSegmentsCap = 8

// segmentStore keeps a built card's composed report until it is read.
//
// TAKEN EXACTLY ONCE. A report belongs to one card and one read; leaving it
// behind would let a later card with a recycled id speak last week's weather,
// and the rotation recycles ids by design (ReadID is a pure function of the
// location).
type segmentStore struct {
	mu   sync.Mutex
	segs map[string][]synth.Segment
}

func newSegmentStore() *segmentStore { return &segmentStore{segs: map[string][]synth.Segment{}} }

// put files a card's composed report.
func (s *segmentStore) put(id string, segs []synth.Segment) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.segs) >= heldSegmentsCap {
		// THE BOUND IS THE POINT, not which entry goes. Everything here is a
		// report for a card that was built and never read, and the cost of
		// clearing is that such a card declines at the air and the schedule
		// routes around it — which is what it would do anyway.
		clear(s.segs)
	}
	s.segs[id] = segs
}

// take is a card's composed report, removed.
func (s *segmentStore) take(id string) ([]synth.Segment, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	segs, ok := s.segs[id]
	delete(s.segs, id)
	return segs, ok
}
