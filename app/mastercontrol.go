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

	// held is whether something owns the bed for longer than one sequence.
	//
	// MVS-D-67: "stays at its lowered volume until the rail is cleared and then
	// either the 'press [w] in watchpost' or the 'for more information go to
	// <website>' tail plays." The narration arbiter gives way per SEQUENCE and
	// takes back the moment nothing is waiting, so a rail of two cards dipped,
	// lifted, dipped and lifted again between them — measured, before this
	// existed. While the bed is held, a per-sequence take-back is refused.
	held bool
}

// newMastercontrol wires the effector. A nil voice is legitimate and means no
// audio; a nil band is not.
func newMastercontrol(v narrationVoice, send func(tea.Msg)) *mastercontrol {
	if err := invariant.Check(send != nil, "mastercontrol is built with a band to write to"); err != nil {
		return nil
	}
	return &mastercontrol{v: v, send: send}
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
// Director emits — `leave` releases only what it cued. Making it conditional
// here would put the rule in two places, and the copy here could not see the
// schedule that decides it.
func (m *mastercontrol) clearBand() {
	if m == nil {
		return
	}
	m.send(tty.TickerBreakingDoneMsg{})
}
