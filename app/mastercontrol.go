package app

// mastercontrol.go — the effector half of the old director's split (PD-5, T2.3).
//
// The decision half — who gets the air, what suspends what, what is read next —
// is the Director's, and it is a pure function in platform/lineup. What remains
// here is the half that PERFORMS: it puts one card on the air across BOTH
// outputs at once, the voice and the band, which is precisely the thing the HUM
// LEAD's condition asked for — "a system directly responsible for ensuring the
// radio output and the news ticker are properly synced for takeover events".
//
// IT IS THE ONE OWNER OF EACH OUTPUT (D-1). The duck was owned by the narration
// arbiter and the band was written from two places — app/ticker.go for the live
// takeover and app/executors.go for the Director's — so the same message was
// constructed twice, in two files, by two paths. Two carriers of one rule
// disagree silently, and this codebase already paid for that once: the duck was
// lifted by one spelling of `tune` and not the other, and a listener heard the
// next location come up at full volume over a breaking alert still reading.
//
// IT ORDERS NOTHING, AND THAT IS DELIBERATE (D-5). Serialising the band against
// the voice across time is the pump's lane, which is fed from one goroutine so
// that ordering outlives a single step. mastercontrol's lock covers one call, so
// the only guarantee it makes is that two callers do not interleave INSIDE one
// operation. Anything wider is claimed where it is enforced.

import (
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// mastercontrol performs on the two outputs a card reaches the listener through.
type mastercontrol struct {
	mu sync.Mutex

	// v is the voice — the radio deck in production, a fake in tests, and nil
	// when there is no audio at all, in which case the visual half still runs.
	v narrationVoice

	// send writes to the band. Never nil: a mastercontrol with nowhere to put a
	// callout is refused at construction, because the failure would otherwise
	// surface as a takeover that reads with no headline and no explanation.
	send func(tea.Msg)

	// ducked is whether the broadcast is currently given way to. THE ONE COPY.
	ducked bool

	// carry hands the Director a DECLARATION. Nil until the schedule is wired,
	// and on a station with no schedule it stays nil — silent rather than a
	// panic on the one path that has none.
	//
	// MASTERCONTROL DECLARES ON AIR / STANDBY, and everyone complies, the
	// Director included (MVS-D-78). 0.14.0's role model recorded the gap in as
	// many words — "Today: app/mastercontrol owns the band and the duck. It
	// does NOT own ON AIR / STANDBY" — and this is that sentence closing.
	//
	// IT DECLARES; IT DOES NOT PERFORM. Two entities act on the declaration and
	// neither alone suffices: the Director holds the schedule, because gating
	// only the audio would let cards be marked read and consumed silently, and
	// MasterControl silences the bed, because a Director holding every card
	// still leaves a relay playing and that is not dead air.
	carry func(lineup.Event)

	// held is whether something owns the bed for longer than one sequence.
	//
	// MVS-D-67: "stays at its lowered volume until the rail is cleared and then
	// either the 'press [w] in watchpost' or the 'for more information go to
	// <website>' tail plays." The narration arbiter gives way per SEQUENCE and
	// takes back the moment nothing is waiting, so a rail of two cards dipped,
	// lifted, dipped and lifted again between them — measured, before this
	// existed. While the bed is held, a per-sequence take-back is refused.
	held bool

	// silenceProgramme takes the main track's card off the air (F-91).
	//
	// MASTERCONTROL SILENCES, AND THIS IS THE SECOND THING IT SILENCES. The bed
	// was the first, for the reason `carry` states: the Director holding every
	// card still leaves audio playing, and that is not dead air. A main-track
	// card is the same sentence with a different subject — the card leaves the
	// schedule the moment the power drops, and the WORDS play on regardless,
	// because a read that has started is a worker blocking on an engine.
	//
	// TWO TRIGGERS, AND BOTH ARE THE OPERATOR'S (D-74). The power leaving
	// Running is STANDBY; the air leaving the programme is the operator moving
	// back to Observer — which he ruled comes back "like if Observer was first
	// opened", and Observer first opened is silent.
	//
	// Nil where there is no broadcast engine.
	silenceProgramme func()

	// fence is what the alert rail is scoped to right now (D-75) — the
	// listener's filter or the station's service area, whichever surface has
	// the air. It travels with every `Aired`, so the rail is re-tested when the
	// air moves. Nil is "All", which is a deck built for one narrow question.
	fence func() lineup.Fence
}

// newMastercontrol wires the effector. A nil voice is legitimate and means no
// audio; a nil band is not.
func newMastercontrol(v narrationVoice, send func(tea.Msg)) *mastercontrol {
	if err := invariant.Check(send != nil, "mastercontrol is built with a band to write to"); err != nil {
		return nil
	}
	return &mastercontrol{v: v, send: send}
}

// GoOnAir and GoToStandby are the operator's control (FR-5.4), and they are
// the ONLY producers of a power change from the console.
//
// THE CONSOLE ASKS AND IS TOLD. It does not set a flag of its own: the state it
// draws comes back from the Director through Publish (FR-5.1), so a declaration
// that never reached the schedule shows as a control that did nothing — which
// is the honest failure, rather than a banner that lies.
// GOING ON AIR HANDS THE AIR TO THE PROGRAMME AS WELL (D-74), and it does so
// FIRST, so the settle that follows the power already knows who is carrying.
//
// IT DOES NOT ASSUME THE SWAP. The air follows the surface and the control only
// exists on the console, so in practice the programme already holds it — but
// "in practice" is an invariant maintained somewhere else, and an operator who
// presses ON AIR is owed a station that can actually broadcast. `onAired`
// no-ops when nothing changed, so the usual case costs one refused event.
func (m *mastercontrol) GoOnAir() {
	m.HandAir(lineup.AirProgramme)
	m.declare(lineup.Running)
}

// GoToStandby takes the station to dead air.
func (m *mastercontrol) GoToStandby() { m.declare(lineup.OffAir) }

// silenceTheProgramme stops a main-track card that is mid-read, or does nothing.
//
// IT RUNS BEFORE THE DECLARATION, deliberately: the operator asked for silence
// and the event they triggered is carried to a pump that will get to it. It
// returns at once (the read is CANCELLED, never halted from here — D-79), so
// there is no cost to putting it first and the audible answer is immediate.
func (m *mastercontrol) silenceTheProgramme() {
	if m == nil {
		return
	}
	m.mu.Lock()
	silence := m.silenceProgramme
	m.mu.Unlock()
	if silence != nil {
		silence()
	}
}

// HandAir gives the air to one programme or the other (D-74).
//
// MASTERCONTROL IS THE ONLY DECLARER, which is the rule the power already
// follows one concept along: the air is something the OPERATOR DID — they moved
// to a surface — not something a tune happened to imply. Three declarers of the
// power is exactly how listening on one surface came to put the other ON AIR.
// THE FENCE TRAVELS WITH IT (D-75). The rail is re-tested against it, so a
// hazard admitted under the other surface's fence is held rather than read —
// and released again when the fence widens.
func (m *mastercontrol) HandAir(to lineup.Air) {
	// THE AIR LEAVING THE PROGRAMME SILENCES IT (F-91). The Director stops
	// ADVANCING the main track when the air moves, and a card already reading is
	// not advancing — it is a worker blocking on the engine, and it would play
	// to its end over a surface that says the station is not on air.
	if to != lineup.AirProgramme {
		m.silenceTheProgramme()
	}
	m.tell(lineup.Aired{To: to, Fence: m.railFence()})
}

// railFence is what the rail is scoped to right now.
//
// ASKED, NOT PASSED. `GoOnAir` is reached through `tty.Station`, whose signature
// is the console's contract and has no business carrying a fence — and a fence
// passed by every caller is a fence every caller could get wrong. One source,
// asked at the moment it is needed.
func (m *mastercontrol) railFence() lineup.Fence {
	if m == nil || m.fence == nil {
		return lineup.Fence{} // no rail to scope: All, which is what it has always been
	}
	return m.fence()
}

// CutBed moves the programme between the station's line-up and its bed (D-78).
//
// THE FIRST PRODUCTION CALLER `lineup.CutOver` HAS EVER HAD. The Director has
// modelled it since 0.14.0 — FR-4.2, D-11, D-32 — and nothing emitted it, which
// is why `bed.carries` was false for the life of every process and the pause it
// governs had never once happened.
func (m *mastercontrol) CutBed(toBed bool) { m.tell(lineup.CutOver{ToBed: toBed}) }

// StopMonitor stops the operator's own listening.
//
// IT IS NOT A STATION EVENT. The line-up does not pause, the rail does not hold
// and nothing leaves the air: the operator simply stopped listening, which is
// what taking the air to the console means for them.
func (m *mastercontrol) StopMonitor() { m.tell(lineup.Monitored{Running: false}) }

// tell carries one event to the schedule, or gives up when there is none.
//
// EXTRACTED AT THE SECOND CALLER (`declare` was the first): three producers of
// "carry this to the Director if there is one" would be three places for the
// nil check to be forgotten.
func (m *mastercontrol) tell(ev lineup.Event) {
	if m == nil {
		return
	}
	m.mu.Lock()
	carry := m.carry
	m.mu.Unlock()
	if carry == nil {
		return // no schedule to declare to
	}
	carry(ev)
}

func (m *mastercontrol) declare(p lineup.Power) {
	// STANDBY STOPS THE WORDS, NOT ONLY THE SCHEDULE (F-91). `silenceTheProgramme`
	// on the Director's side takes the card off the air; this is the other half
	// the `carry` field's own comment demands — "a Director holding every card
	// still leaves a relay playing, and that is not dead air."
	if p != lineup.Running {
		m.silenceTheProgramme()
	}
	m.tell(lineup.Powered{To: p})
}

// silent reports whether there is no voice to perform with.
func (m *mastercontrol) silent() bool { return m == nil || m.v == nil }

// giveWay dips the broadcast for something that is about to be read.
//
// IT IS IDEMPOTENT. The arbiter gives way when a sequence takes the air and
// takes it back when nothing is left waiting or suspended, and the Director
// will do the same around a whole rail drain (MVS-D-67) — so a second caller
// asking while it is already down must be a no-op rather than a second dip. An
// earlier version returned whether THIS call won; nothing asked, and a return
// nobody reads is a promise to a caller that does not exist.
func (m *mastercontrol) giveWay() {
	if m.silent() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dip()
}

// dip is THE ONE CARRIER of "put the broadcast down" (D-1). THE CALLER HOLDS
// m.mu AND HAS ALREADY PASSED silent() — it dereferences the voice, so a third
// caller that checks only `m != nil` would nil-deref here. It exists because `hold` must dip AND claim the bed without
// letting go in between; before it did, `hold` called giveWay and then took the
// lock a second time to set `held`, and a take-back landing in that window saw
// held=false over a ducked bed and lifted it. The rail then believed it held a
// bed that was already back up, with nothing left to dip it again.
func (m *mastercontrol) dip() {
	if m.ducked {
		return
	}
	m.v.duck()
	m.ducked = true
}

// takeBack lifts the dip, unless something is HOLDING the bed for longer than
// one sequence. With nothing given way it does nothing, rather than lifting a
// broadcast nobody dipped.
func (m *mastercontrol) takeBack() {
	if m.silent() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.held {
		return // the rail owns the bed until its tail has played (MVS-D-67)
	}
	if !m.ducked {
		return
	}
	m.v.restore()
	m.ducked = false
}

// hold takes the bed for longer than one sequence and dips it.
//
// NOTHING REACHES IT TODAY (red team 2026-09-05, I-5). Its one caller is the
// lineup.Duck executor, and no production path constructs that effect — so
// `held` is never true, and unhold, takeBack's held branch and
// director.releaseBed are inert with it. Everything below describes what the
// mechanism DOES, correctly, on a path the running app does not take. See
// app/executors.go's Duck case for why that is currently harmless and why the
// mechanism is kept rather than deleted. Every
// per-sequence take-back is refused until release, so a rail of many cards is
// ONE dip and ONE lift however many times the arbiter runs inside it.
//
// THE DIP AND THE CLAIM ARE ONE CRITICAL SECTION, and that is the whole point.
// The two halves ran under separate locks once — giveWay, unlock, lock, set
// held — and `hold` is called from the executor's goroutine while `takeBack`
// runs on the arbiter's, sharing only this mutex. A take-back arriving in that
// window read held=false and ducked=true and restored the bed; `held` was then
// set over a broadcast already back at full volume, which is a rail reading its
// whole drain against an undipped bed. It is the same shape as the release-side
// defect mE6 pinned, on the acquire side (F-D5).
func (m *mastercontrol) hold() {
	if m.silent() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dip()
	m.held = true
}

// unhold gives up a hold WITHOUT deciding whether the bed may come back.
//
// THE DECISION IS NOT THIS TYPE'S, and the lock order is why. The arbiter takes
// its own lock and then this one — settle calls takeBack while holding d.mu — so
// a release that took THIS lock and then asked the arbiter would invert the
// order and deadlock. Worse, an earlier version asked outside the lock and acted
// on the stale answer: a job admitted between the question and the lift had the
// bed restored out from under it, which is the broadcast surging to full volume
// over a read in progress. The caller decides under its own lock; see
// director.releaseBed.
func (m *mastercontrol) unhold() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.held = false
	m.mu.Unlock()
}

// givenWay reports whether the broadcast is currently ducked.
func (m *mastercontrol) givenWay() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ducked
}

// holdLine holds the line in flight; resumeLine plays it on from where it
// stopped; dropHeld drops a held line whose sequence ended while it waited.
// They are the arbiter's suspension mechanics, and they live here because the
// voice does.
func (m *mastercontrol) holdLine() {
	if m.silent() {
		radioDebugLog("mc:holdLine:silent")
		return
	}
	radioDebugLog("mc:holdLine")
	m.v.pause()
}

func (m *mastercontrol) resumeLine() {
	if m.silent() {
		return
	}
	m.v.resume()
}

// stopLine ends the line in flight — a read cut short, not one that finished.
//
// THROUGH THE ONE OWNER, like every other write to the outputs (D-1). Closing a
// player that has already drained is a no-op, so this is safe on the single
// path a job leaves the air by and needs no "was it cancelled" flag to decide.
func (m *mastercontrol) stopLine() {
	if m.silent() {
		radioDebugLog("mc:stopLine:silent") // no voice wired: nothing to stop
		return
	}
	radioDebugLog("mc:stopLine")
	m.v.stop()
}

func (m *mastercontrol) dropHeld() {
	if m.silent() {
		return
	}
	m.v.discard()
}

// cue shows the band the callout for the card taking the air (DR-18).
//
// FIRE AND TRUST: the voice never waits on the band, so this returns nothing to
// wait for. Its ORDER against the words is the pump's lane, not this call.
func (m *mastercontrol) cue(item tty.TickerItem) {
	if m == nil {
		return // refused at construction; never a panic on the hazard path
	}
	m.send(tty.TickerBreakingMsg{Item: item})
}

// clearBand gives the band its rotation back.
//
// IT IS UNCONDITIONAL, deliberately and for now. DR-24 pairs a release with the
// CUE rather than with the card, and that pairing is a property of what the
// Director emits. Making it conditional here would put the rule in two places,
// and the copy here could not see the schedule that decides it.
//
// THE SENTENCE THAT USED TO FINISH THAT PARAGRAPH — "`leave` releases only what
// it cued" — WAS WRONG, and 0.16.0 P3 made it visibly so (red team 2026-09-09,
// finding 7). `leave` releases whatever was ON THE AIR, and a LocationReport is
// routinely on the air without a cue: runCue asks the producer for an alert
// under `read:<location>`, finds none, and cues nothing. So every rotation turn
// issued a release for a cue that never happened.
//
// F-71, AND IT IS CLOSED (D-82) — by the caller, exactly as this paragraph
// argued it had to be. The trigger it named was "the moment the band gets a
// second writer"; what arrived instead was a second card on the air, when the
// rail gained the ability to interrupt a report. `runRelease` now asks the LANE,
// which rides on the effect, so the rule stays where the schedule decides it and
// this function stays the one unconditional carrier of "give the band back".
func (m *mastercontrol) clearBand() {
	if m == nil {
		return
	}
	m.send(tty.TickerBreakingDoneMsg{})
}
