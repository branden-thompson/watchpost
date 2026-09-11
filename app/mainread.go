package app

// mainread.go — THE MAIN TRACK'S READER (F-91).
//
// BD-9 is the whole design in one sentence: *"A report's speak is the engine
// Source adapter — which IS the main-track absorb."* D-33 says the same thing
// from the operator's side: *a chosen read REPLACES the bed rather than
// speaking over it.* So the programme is not a narration, it does not go
// through the arbiter, and what performs it is the thing that performs every
// other broadcast this station makes — the engine, fed by a synth.Source.
//
// WHAT THAT BUYS, AND IT IS THE REASON THE SHAPE WAS RULED RATHER THAN CHOSEN:
// `StartSource` calls `setLive(false)`, and `giveWayLocked` reads the source
// kind every 50 ms — so a report on the air HOLDS under a breaking alert
// instead of dipping, which is D-24 exactly ("the read PAUSES and resumes
// mid-sentence"). A reader built on the clip path would have had to reimplement
// that rule, and reimplementing it is how the wrong duck decision was made the
// first time.
//
// AND THE BED NEEDS NO HAND-BACK. Main and bed are mutually exclusive (D-11,
// FR-4.2) and the engine has one source, so while the main track is the
// programme the bed is not playing: `advances(MainTrack)` already requires
// `!bed.carries`. There is nothing underneath this to restore.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// readerFor is the executors' reader seam, wired to the deck.
//
// A SEAM, LIKE compose, AND FOR THE SAME REASON: the executors know that a
// card's words have to be performed and nothing about what performs them. A
// station with no deck has no reader, and the executor declines by name rather
// than reaching through a nil.
func readerFor(deck *radioDeck) func(context.Context, lineup.Speak) bool {
	if deck == nil {
		return nil // no audio: newExecutors leaves the seam nil and speak declines
	}
	return func(ctx context.Context, v lineup.Speak) bool {
		return deck.readCard(ctx, v.Headline, segmentsFromScript(v.ID, v.Script))
	}
}

// readSession is one main-track card on the broadcast engine, and the channel
// the reader waits on.
//
// THE ENGINE IS FIRE-AND-FORGET AND THE READER IS NOT. `StartSource` returns
// the moment the goroutine is armed; the executor has to come home Finished or
// Failed, which means it has to know when the words ran out. The only thing
// that observes that is the status callback, so this is what carries the answer
// back across the two goroutines.
type readSession struct {
	done chan struct{}
	// stop ends the read from OUTSIDE it — the operator went to standby, or the
	// air went back to the monitor (F-91).
	//
	// A CANCEL, NOT A HALT, AND THAT IS D-79. `engine.Halt` waits for the audio
	// goroutine, which is in the middle of calling BACK into the program — so a
	// halt reached from `Router.Update` sends to a loop that cannot receive and
	// the app freezes hard enough to need the terminal killed. Cancelling
	// returns immediately, and the WORKER already waiting on this read does the
	// halting, on a goroutine that is allowed to block.
	stop context.CancelFunc
	once sync.Once
	ok   bool // written inside once, before done closes; read after it
}

// finish ends the read, once. A status callback can fire several times for one
// stream — a title change, then the stop — and a second close would panic.
func (r *readSession) finish(ok bool) {
	r.once.Do(func() {
		r.ok = ok
		close(r.done)
	})
}

// noteRead is the status callback's half: it ends the session the moment the
// stream reaches a terminal state.
//
// ENDED IS THE ONLY SUCCESS. A Stopped that is not the sign-off is a Halt —
// the operator went to standby, the pump is stopping, a later read took the
// engine — and a Failed is a render that died. Both are reads that ended before
// the words did, which is DR-24's rule and the same verdict the rail path
// returns for them.
func noteRead(r *readSession, st player.Status, ended bool) {
	if r == nil {
		return
	}
	switch {
	case ended:
		r.finish(true)
	case st.State == player.Stopped || st.State == player.Failed:
		r.finish(false)
	}
}

// readCard puts one card's words on the air and waits for them to run out.
//
// IT BLOCKS, DELIBERATELY. `executors.run` is documented to block for as long
// as the work takes — it runs on a worker, never the pump — and a Speak that
// returned before the words did would let the Director step the next card onto
// an engine still carrying this one.
func (d *radioDeck) readCard(ctx context.Context, label string, segs []synth.Segment) bool {
	if d == nil || len(segs) == 0 {
		return false // nothing to say: the executor's empty-script check already refused this
	}
	voice, err := d.voice() // may install Piper (minutes): never under a lock
	if err != nil {
		d.engine.Fail(err.Error()) // the reason, in the player (F2)
		return false
	}
	// THE READ'S OWN CONTEXT, so it can be ended without ending the effect's.
	// It is a CHILD of the pump's, so a stopping schedule still takes the words
	// off the air.
	rctx, stop := context.WithCancel(ctx)
	defer stop()
	sess := &readSession{done: make(chan struct{}), stop: stop}
	// ONE CYCLE, NOT A ROTATION. The Source's `next` is asked once because
	// Loop is off, and a card is a fixed set of words: there is no second cycle
	// to plan and nothing to re-ask for. That is also what ends the stream, and
	// the end is how this read reports itself finished.
	src, err := synth.NewSource(voice, func(context.Context) ([]synth.Segment, error) { return segs, nil },
		func(seg synth.Segment, spoken time.Duration) {
			d.debugLog(fmt.Sprintf("read segment key=%q spoken=%s", seg.Key, spoken.Round(time.Millisecond)))
			d.setDetailTimed(seg.Text, spoken)
		})
	if err != nil {
		d.engine.Fail(err.Error())
		return false
	}
	// THE SAME CAST AS THE ROTATION'S. Every segment this card carries is
	// cast.All today (segmentsFromScript), so the resolver answers with the
	// root correspondent — but installing it here rather than skipping it is
	// what keeps ONE answer to "who reads this station", so the day a card
	// carries a role it is already right.
	src.SetResolver(func(role cast.Role) (synth.Voice, error) {
		v, _, err := d.resolveVoice(role)
		return v, err
	})
	src.SetHandoffLine(d.composer.HandoffLine)
	src.Loop(false)
	d.setMode("read", label, "on the main track")
	// ONE STEP WITH THE START, the discipline Stop and startSynth already
	// keep (N-3): without it a Stop landing between the record and the engine
	// leaves a session nothing will ever finish, and the executor waits on it
	// until the pump's context ends.
	d.tuneMu.Lock()
	d.mu.Lock()
	d.source, d.read = src, sess
	d.mu.Unlock()
	// THE LOCATION IS NOT TOUCHED, and that is not an oversight. `ref` is the
	// MONITOR's row — which location the dashboard highlights as playing — and
	// the monitor does not have the air while this reads (D-74). Pointing it at
	// a pool location the listener never asked for would light up a row in the
	// other mode's list.
	d.engine.StartSource("Watchpost Programme · "+label, src.Rate(), src.Open)
	d.tuneMu.Unlock()
	select {
	case <-sess.done:
		return sess.ok
	case <-rctx.Done():
		// THE STATION STOPPED BROADCASTING THIS — the pump is shutting down, the
		// operator went to standby, or the air went back to the monitor. Halt,
		// rather than leaving the words playing under a station that has gone:
		// nothing else would ever take them off.
		//
		// ON THIS GOROUTINE, which is a worker and may block. That is the whole
		// reason `stop` is a cancel: whoever asked for silence did not have to
		// wait for the audio device, and in the one case that matters could not
		// have afforded to (D-79).
		d.engine.Halt()
		return false
	}
}

// stopRead takes a main-track card off the air, and returns at once.
//
// INERT WITH NOTHING READING, which is most of the time: the monitor's own
// rotation is not a card, and silencing it is `Stop`. Whoever asks for silence
// asks BOTH, because they are two different things playing through one engine.
func (d *radioDeck) stopRead() {
	if d == nil {
		return
	}
	d.mu.Lock()
	r := d.read
	d.mu.Unlock()
	if r == nil || r.stop == nil {
		return
	}
	r.stop()
}
