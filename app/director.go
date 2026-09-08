package app

// director.go — the narration arbiter: the one owner of the correspondent's voice.
//
// NAMED FOR WHAT IT ARBITRATES, not for the release's Director. platform/lineup
// holds THAT one; this decides who gets the AIR when it frees. The two collide
// in the word and in the file name, and both have a `settle` (red team
// 2026-09-05, R-7/Junior-Dev 5)
// (HUM LEAD 2026-08-28, "build it robust, build it right"). Every spoken
// sequence — a breaking takeover, an event read, whatever comes next — runs
// through Run with a class; the director serialises them, ducks the
// broadcast once for the whole run and restores it once at the end, and lets
// a higher class take the air from a lower one: the lower sequence is
// SUSPENDED — its line in flight pauses, its holds stop counting, a line it
// is still rendering waits — and RESUMES where it left off when the higher
// one ends (HUM LEAD UAT: a read must never collide with a takeover, and
// should carry on afterwards). Equal classes queue in order. Adding a
// narration source is a class constant and a Run call.
//
// Priorities (highest first):
//   narrateBreaking — "<event> has been declared": always first, suspends a read
//   narrateRead     — [space] on an event in the window: ducks the broadcast,
//                     waits behind a takeover, is suspended by one
// The broadcast itself (relay or synth) is not a narration: it is what gets
// ducked.

import (
	"context"
	"fmt"

	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

type narrationClass int

const (
	narrateRead narrationClass = iota
	narrateBreaking

	// numNarrationClasses bounds the set; it is not itself a class.
	//
	// IT EXISTS SO A GUARD CAN WALK THE TYPE. TestNothingOutranksATakeover
	// protects an assumption `holdRest` depends on, and its first version
	// iterated a hand-written list of the classes — so ADDING one, the exact
	// case it was written for, left it green. A sentinel is the difference
	// between a guard and a list that has to be remembered.
	numNarrationClasses
)

// clip is one rendered line: the audio and how long it plays; viz says
// whether the visualizer follows it (set by the director from the class —
// vizFor). text rides along for fakes and diagnostics.
type clip struct {
	text string
	pcm  []byte
	rate int
	dur  time.Duration
	viz  bool
	role cast.Role // whose voice rendered it — set once, by the deck's render
}

// vizFor is the one owner of "does the visualizer follow this narration"
// (HUM LEAD 2026-08-28): every narration does — an event read, a voice
// preview — except a breaking takeover, whose tone and lines play aside.
func vizFor(class narrationClass) bool { return class != narrateBreaking }

// narrationVoice is what a sequence needs from the radio: duck the
// broadcast, sound the attention tone, render a line, play a rendered line,
// pause and resume the line in flight, discard a held line whose sequence
// ended while suspended, and restore the broadcast. The radio deck
// implements it; tests use a fake; nil means no audio (visual sequences
// still run).
type narrationVoice interface {
	duck()
	// tone sounds the attention signal for an alert of this class. It renders
	// from constants and resolves no voice (FR-9).
	tone(class cast.Class) time.Duration
	// render turns a line into audio in the voice the ROLE resolves to.
	render(ctx context.Context, role cast.Role, text string) (clip, bool) // bound to the sequence: a cancelled job renders no further (R5-B-07)
	play(c clip)
	pause()
	resume()
	// stop ends the line in flight. A cancelled read must not play on after
	// its sequence has gone (MVS-D-75).
	stop()
	discard()
	restore()
}

type narrationJob struct {
	class     narrationClass
	role      cast.Role // whose voice reads this sequence (FR-3)
	audible   bool      // a muted or voiceless sequence runs its visuals only and never dips the broadcast
	ctx       context.Context
	cancel    context.CancelFunc // ends THIS job without touching the caller's context
	seq       uint64             // arrival order within a class
	suspended bool               // taken off the air by a higher class; resumes when it ends
	// paused is a LISTENER'S hold, and it is not the same as suspended even
	// though it uses the same stack (MVS-D-74). A suspended job is waiting for
	// something to finish and resumes on its own; a paused one is waiting for a
	// person and resumes only when they ask. Two consequences, both deliberate:
	// settle never promotes it, and it HOLDS NOTHING — the broadcast comes back
	// up while a read is paused, because nobody is speaking over it.
	paused bool
}

// director arbitrates the voice.
type director struct {
	mu        sync.Mutex
	turn      *sync.Cond
	v         narrationVoice
	mc        *mastercontrol // the effector: the one owner of the duck and the band (T2.3)
	onAir     *narrationJob
	suspended []*narrationJob // the jobs a higher class took the air from, innermost last
	waiting   []*narrationJob
	next      uint64
	sleep     func(ctx context.Context, d time.Duration) bool // sleepCtx; tests replace it
}

// newDirector builds the arbiter over a voice and THE effector.
//
// The effector is passed in rather than built here because it is shared: the
// ticker writes the band through the same instance, and a second instance would
// be a second owner of the duck — the exact shape this task exists to remove.
func newDirector(v narrationVoice, mc *mastercontrol) *director {
	d := &director{v: v, mc: mc, sleep: sleepCtx}
	d.turn = sync.NewCond(&d.mu)
	return d
}

// silent reports whether there is no voice at all.
func (d *director) silent() bool { return d == nil || d.v == nil }

// speaker is what a running sequence speaks through: every call honours
// the job's context (an ended sequence hears 0 and false), its suspension
// (a line waits, a hold stops counting) and the audible flag (a muted
// sequence runs its visuals only, and never ducks).
type speaker struct {
	d     *director // nil for a nil director: the sequence runs silently
	job   *narrationJob
	ctx   context.Context
	sleep func(ctx context.Context, d time.Duration) bool
}

// holdStep is how often a hold checks for suspension or an ended context.
const holdStep = 100 * time.Millisecond

// attention sounds the attention tone for an alert of this class; 0 when
// silent, muted or ended.
//
// The class is the CALLER's — the ticker knows its burst's highest-severity
// event, the read knows its row — because only they can classify what they are
// about to say. The Director forwards it; it does not decide it.
func (s *speaker) attention(class cast.Class) time.Duration {
	if !s.live() {
		return 0
	}
	return s.d.v.tone(class)
}

// line renders and plays one line, returning how long it plays; 0 when
// silent, muted or ended (the caller keeps its visual hold). A line rendered
// while the job is suspended waits for the air before it starts, and starts
// under the arbiter's lock: a takeover admitted meanwhile finds it in flight
// and pauses it, never plays over it (red-team round 4, A-04).
func (s *speaker) line(text string) time.Duration {
	c, ok := s.prepare(text)
	if !ok {
		return 0
	}
	return s.deliver(c)
}

// prepare RENDERS a line without putting it on the air, and deliver puts a
// prepared one on. line is the two together, and most callers want that.
//
// They are separable so a sequence can render DURING something that is already
// sounding — the attention tone runs about two seconds and a render takes about
// one, so a takeover that renders its first words while the tone plays starts
// speaking the moment the tone ends instead of a beat later. That gap was
// audible.
func (s *speaker) prepare(text string) (clip, bool) {
	if !s.live() || text == "" {
		return clip{}, false
	}
	c, ok := s.d.v.render(s.ctx, s.role(), text)
	if !ok {
		return clip{}, false
	}
	c.viz = vizFor(s.job.class)
	return c, true
}

func (s *speaker) deliver(c clip) time.Duration {
	if !s.live() {
		return 0
	}
	if !s.awaitAir(func() { s.d.v.play(c) }) {
		return 0
	}
	return c.dur
}

// hold waits d of AIR time — a suspension does not count — returning false
// if the sequence ended meanwhile.
func (s *speaker) hold(d time.Duration) bool {
	steps := int(d/holdStep) + 1 // bounded: one sleep per step of air time (P10-02)
	for i := 0; i < steps && d > 0; i++ {
		if !s.awaitAir(nil) {
			return false
		}
		step := min(d, holdStep)
		if !s.sleep(s.ctx, step) {
			return false
		}
		d -= step
	}
	return true
}

// awaitAir blocks while the job is suspended, then runs then (when given)
// while the arbiter's lock is still held — nothing can be admitted between
// the check and the start of a line; false when the context ended.
func (s *speaker) awaitAir(then func()) bool {
	if s.d == nil || s.job == nil {
		return s.ctx.Err() == nil
	}
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	// A condition-variable wait: one Wait per iteration, ended by the resume
	// or the context ending (the P10-02 ledger row for admit covers the shape).
	for s.job.suspended && s.ctx.Err() == nil {
		s.d.turn.Wait()
	}
	if s.ctx.Err() != nil {
		return false
	}
	if then != nil {
		then()
	}
	return true
}

// role is the job's role, or the root when there is no job (a nil Director's
// silent sequence).
func (s *speaker) role() cast.Role {
	if s.job == nil {
		return cast.All
	}
	return s.job.role
}

func (s *speaker) live() bool {
	return s.job != nil && s.job.audible && !s.d.silent() && s.ctx.Err() == nil
}

// Run runs seq as a narration of this class: it waits for its turn (a
// higher class on air finishes first; a lower one is suspended), ducks the
// broadcast if the sequence is audible and nothing else already did, runs
// seq, and restores the broadcast when nothing is waiting or suspended. It
// returns false when ctx ended before the sequence finished.
// role is whose voice reads the sequence (FR-3); the tone's class travels
// separately, on the speaker's attention call, because only the caller can
// classify the event it is about to read.
func (d *director) Run(ctx context.Context, class narrationClass, role cast.Role, audible bool, seq func(ctx context.Context, s *speaker)) bool {
	// EVERY Director job is priority work (D-R3-1, MVS-D-43): the listener is
	// waiting on it, so it may take one of the two reserved render slots rather
	// than queue behind the broadcast's read-ahead.
	ctx = synth.WithPriority(ctx)
	if d == nil {
		seq(ctx, &speaker{ctx: ctx, sleep: sleepCtx}) // no director at all: run the visuals, in real time
		return ctx.Err() == nil
	}
	// THE JOB GETS ITS OWN CANCEL (FR-9), a child of the caller's context.
	//
	// Without it nothing but the caller can end a read, and the caller is
	// waiting for it: a read whose audio hangs would go on issuing lines over a
	// player that has been closed from under it, holding the arbiter and
	// keeping the bed ducked. Cancelling here unwinds the sequence onto the
	// path a cut-short read already takes — Failed{Routed:true}, the schedule
	// advances, restore() runs — rather than inventing a second way out.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // the child never outlives the job
	job := &narrationJob{class: class, role: role, audible: audible && !d.silent(), ctx: ctx, cancel: cancel}
	// The context's end wakes every wait this job may be parked in — the
	// turn wait in admit, the air wait while suspended (REVIEW R5-B-02: a
	// cancel landed while parked stayed parked until the takeover's release).
	stop := context.AfterFunc(ctx, func() {
		d.mu.Lock()
		d.turn.Broadcast()
		d.mu.Unlock()
	})
	defer stop()
	if !d.admit(job) {
		return false
	}
	seq(ctx, &speaker{d: d, job: job, ctx: ctx, sleep: d.sleep})
	d.release(job)
	return ctx.Err() == nil
}

// admit queues the job, suspends a lower class on air, and waits for the turn.
func (d *director) admit(job *narrationJob) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.next++
	job.seq = d.next
	if on := d.onAir; on != nil && job.class > on.class {
		on.suspended = true // its line pauses, its holds stop counting, a line it is rendering waits
		d.suspended = append(d.suspended, on)
		d.onAir = nil
		if d.v != nil && on.audible {
			d.mc.holdLine()
		}
		narrateDebug("suspend", on)
	}
	narrateDebug("admit", job)
	d.waiting = append(d.waiting, job)
	// A condition-variable wait: one Wait per iteration, ended by the turn
	// arriving or the job's context ending (checked on every wake) — the
	// scheduler tier loops' shape (P10-02 ledger row, app/director.go admit).
	for d.onAir != nil || d.first() != job {
		if job.ctx.Err() != nil { // cancelled while waiting: whoever it outranked may resume
			d.remove(job)
			d.settle()
			return false
		}
		d.turn.Wait()
	}
	d.remove(job)
	d.onAir = job
	if job.audible {
		// THE DUCK HAS ONE OWNER, and it is not this function (D-1). It is
		// idempotent there, so a nested sequence asking again is a no-op rather
		// than a second dip, and the arbiter no longer carries a flag that could
		// disagree with the deck's actual state.
		d.mc.giveWay()
	}
	return true
}

// release ends a job: the one on air leaves it; one that ended while
// SUSPENDED (its context ran out under a takeover) leaves the suspended
// stack and its held line is discarded, the air untouched (round 4, A-02).
// Then the director settles.
func (d *director) release(job *narrationJob) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.onAir == job {
		d.onAir = nil
		// THE LINE GOES WITH THE JOB (MVS-D-75). Cancelling a read used to end
		// its SEQUENCE and leave its audio playing: the air was released, the
		// suspended location read resumed underneath, and the listener heard
		// both at once. The sound is part of what a job holds, so it is given
		// back on the one path a job leaves the air by.
		//
		// ONLY A JOB THAT WAS CUT SHORT, and the arbiter can tell: a cancelled
		// job's context carries the error, a completed one's does not.
		//
		// The first version stopped unconditionally, on the reasoning that
		// closing a drained player is a no-op. It is not safe to rely on: a
		// part's hold is its own length LESS the work already done, so the
		// sequence can end a hair before the audio finishes draining, and an
		// unconditional close would clip the tail of every read. Two tests
		// caught it by recording a `stop` where a completed read had never had
		// one.
		if radioDebugOn() {
			radioDebugLog(fmt.Sprintf("release:onair cancelled=%v voice=%v audible=%v", job.ctx.Err() != nil, d.v != nil, job.audible))
		}
		if job.ctx.Err() != nil && d.v != nil && job.audible {
			d.mc.stopLine()
		}
	} else if job.suspended {
		d.unsuspend(job)
		if d.v != nil && job.audible {
			d.mc.dropHeld()
		}
	} else {
		return // already discarded by settle: nothing of it remains
	}
	d.settle()
}

// abandonOnAir ends the read that is on the air, and reports whether there was
// one (FR-9).
//
// IT CANCELS RATHER THAN STOPS. Stopping the audio would leave the sequence
// running and it would simply speak the next line; cancelling unwinds it onto
// the one path a cut-short read already takes, which the arbiter, the schedule
// and the bed all already understand.
func (d *director) abandonOnAir(why string) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	job := d.onAir
	d.mu.Unlock()
	if job == nil || job.cancel == nil {
		return false // nothing on the air: whatever failed has already gone
	}
	radioDebugLog("read:abandoned:" + why)
	job.cancel()
	return true
}

// settle decides who has the air when it frees (a release, a cancelled wait):
// a waiting job of a class above the innermost suspended one goes first,
// else the suspended job resumes where it stopped; the broadcast is restored
// only when nothing waits or is suspended. Callers hold d.mu.
func (d *director) settle() {
	defer d.turn.Broadcast()
	if d.onAir != nil {
		return
	}
	// A suspended job whose context ended while it waited is discarded, never
	// resumed (R5-B-02: its held line played out while the duck was already
	// lifted); the next one down gets the air. Bounded by the stack (P10-02).
	//
	// IT IS PROMPTNESS, NOT CORRECTNESS, AND ITS MUTANT IS EXPECTED TO SURVIVE.
	// Disabling this loop entirely leaves the whole package green — checked —
	// because the job's OWN goroutine cleans it up: its context ending wakes
	// awaitAir, Run returns, and `release` does the same unsuspend and dropHeld.
	// Since round 4 gave innermostResumable the liveness rule, a corpse left here
	// is also never promoted and never holds the bed. What this loop buys is
	// dropping the held line NOW rather than when the goroutine is scheduled.
	//
	// So it is kept and documented rather than pinned: D-2 says a check with no
	// failing input should go, and this is not a check — it is the same work,
	// sooner. The liveness question it asks goes through `live` so that a future
	// clause reaches it too, which is the part that would otherwise drift.
	for s := d.innermostSuspended(); s != nil && !live(s); s = d.innermostSuspended() {
		d.unsuspend(s)
		if d.v != nil && s.audible {
			d.mc.dropHeld()
		}
	}
	if s := d.innermostResumable(); s != nil && (d.first() == nil || d.first().class <= s.class) {
		d.unsuspend(s)
		d.onAir = s
		if d.v != nil && s.audible {
			// THE BED MAY HAVE COME BACK WHILE IT WAITED, and only a listener's
			// pause does that — a job suspended by a takeover keeps the dip, so
			// this is a no-op there (giveWay is idempotent). Without it a resumed
			// read speaks over a broadcast at full volume, which is the defect
			// the duck's one owner exists to prevent, arriving from the one path
			// that lifts the dip while a job is still alive.
			d.mc.giveWay()
			d.mc.resumeLine()
		}
		return
	}
	// A PAUSED READ HOLDS NOTHING. The bed comes back up while it waits for a
	// person, because nobody is speaking over it — counting it here would leave
	// the broadcast dipped for as long as a listener left the read paused.
	// A PAUSED WAITING READ HOLDS NOTHING EITHER, and `first()` is the one that
	// knows: it already skips them, so asking it rather than counting the queue
	// keeps this guard and the promotion rule reading the same state. Counting
	// the queue left the broadcast dipped for as long as a listener kept a
	// queued read paused — the exact failure the comment above forbids, in the
	// one place that had not learned the rule.
	if d.first() == nil && d.innermostResumable() == nil {
		d.mc.takeBack() // a no-op when nothing is ducked
	}
}

// unsuspend takes job off the suspended stack. Callers hold d.mu.
func (d *director) unsuspend(job *narrationJob) {
	job.suspended = false
	narrateDebug("resume", job)
	for i, j := range d.suspended {
		if j == job {
			d.suspended = append(d.suspended[:i], d.suspended[i+1:]...)
			return
		}
	}
}

func (d *director) innermostSuspended() *narrationJob {
	if len(d.suspended) == 0 {
		return nil
	}
	return d.suspended[len(d.suspended)-1]
}

// innermostResumable is the innermost suspended job that would resume ON ITS
// OWN — so, not one a listener has paused. Callers hold d.mu.
func (d *director) innermostResumable() *narrationJob {
	for i := len(d.suspended) - 1; i >= 0; i-- { // bounded by the stack (P10-02)
		// A DEAD JOB IS NOT RESUMABLE. Promoting one put a cancelled read back on
		// the air with the bed ducked under it, while the chip said Paused — and
		// a paused job above it on the stack shields it from settle's drain, so
		// it can sit there indefinitely waiting to be promoted.
		if j := d.suspended[i]; live(j) && !j.paused {
			return j
		}
	}
	return nil
}

// pauseRead takes a read off the air at the LISTENER'S request (MVS-D-74),
// reporting whether there was one to pause.
//
// It goes on the suspended stack, so a takeover arriving meanwhile behaves
// exactly as it would have — but marked paused, so settle never promotes it
// back and the bed is not held down waiting for a person.
func (d *director) pauseRead() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// A READ IS PAUSED WHEREVER IT IS, and there are three places — which took
	// three attempts to learn. On the air is the obvious one. SUSPENDED is where
	// a takeover put it, and missing that meant [space] did nothing while an
	// alert spoke, then settle promoted the read back when the alert ended.
	// WAITING is where a read ASKED FOR during an alert sits, and missing that
	// was the same failure again in a third ordering: the press ignored, the
	// read starting on its own.
	//
	// A suspended job is already held — the takeover's suspension called
	// holdLine — so it is not held twice. A waiting job has never been on the
	// air and holds nothing at all.
	//
	// AND NEVER A DEAD ONE. A read Cancel has killed is still on the stack until
	// its goroutine unwinds, and claiming that corpse reported success while the
	// live read carried on: the chip said Paused and nothing was.
	for _, j := range d.reads() { // bounded by reads (P10-02)
		if j.paused {
			continue // an already-held read is not paused twice
		}
		j.paused = true
		// LEAVING THE AIR IS THE ONLY DIFFERENCE between the three places, and
		// it is here rather than in a branch of its own: a second branch meant a
		// second copy of "which reads count", and the copy in reads() went dead
		// while the live rule sat here — a decorative enumeration.
		if j == d.onAir {
			j.suspended = true
			d.suspended = append(d.suspended, j)
			d.onAir = nil
			if d.v != nil && j.audible {
				d.mc.holdLine()
			}
		}
		d.settle() // it holds nothing now, so the bed may come back
		return true
	}
	return false
}

// live reports whether a job is still a candidate for anything.
//
// THE ONE CARRIER OF THAT QUESTION (D-1), and it took four review rounds to get
// here. A job whose context has ended is still in its list until its goroutine
// unwinds, and every asker was wrong about it in a different way: the pause
// claimed the corpse and reported success, the report said a corpse was held,
// the promotion put one back ON THE AIR with the bed ducked under it, and the
// take-back guard kept the bed down for a read that was already gone.
//
// Five functions ask it. When it was written out five times, five of them
// disagreed.
func live(j *narrationJob) bool { return j != nil && j.ctx.Err() == nil }

// reads is EVERY LIVE READ THE DIRECTOR IS HOLDING, innermost first: on the air,
// then suspended by a takeover, then waiting behind one. Callers hold d.mu.
//
// TWO CARRIERS, TWO QUESTIONS, AND THEY ARE NOT THE SAME ONE (D-1). `reads()`
// answers WHERE a read can be — the three places, class-filtered. `live()`
// answers whether a job is still a candidate at all, and it is asked by more
// than reads(): also by innermostResumable, by first, and by settle's drain.
//
// Its callers are `pauseRead` and `heldRead` — and `resumeRead` and `readPaused`
// through heldRead. The bed's take-back guard does NOT ask it: that asks first()
// and innermostResumable, which carry the liveness rule directly.
//
// THE COUNT IS STATED BECAUSE MISCOUNTING IT IS THE DEFECT. Four review rounds
// went into discovering that a rule was written in more places than I had
// counted — three places found one at a time, then a fifth asker of liveness,
// then a sixth in settle's drain. A comment here claiming the wrong number is
// how the seventh gets written.
//
// DEAD READS ARE NOT HERE. A read Cancel has killed stays in its list until the
// goroutine unwinds, and every asker was wrong about it in a different way — the
// pause claimed the corpse and reported success, the report said a corpse was
// held, and the window showed Paused over a read that was playing.
func (d *director) reads() []*narrationJob {
	var out []*narrationJob
	if j := d.onAir; live(j) && j.class == narrateRead {
		out = append(out, j)
	}
	for i := len(d.suspended) - 1; i >= 0; i-- { // bounded by the stack (P10-02)
		if j := d.suspended[i]; live(j) && j.class == narrateRead {
			out = append(out, j)
		}
	}
	for _, j := range d.waiting { // bounded by the queue (P10-02)
		if live(j) && j.class == narrateRead {
			out = append(out, j)
		}
	}
	return out
}

// heldRead is the read a listener has paused, or nil. Callers hold d.mu.
func (d *director) heldRead() *narrationJob {
	for _, j := range d.reads() { // bounded by reads (P10-02)
		if j.paused {
			return j
		}
	}
	return nil
}

// resumeRead gives a paused read the air back when it is free, reporting
// whether there was one to resume.
//
// It only clears the flag: settle decides WHEN, exactly as it does for a job
// suspended by a takeover, so a resume pressed while an alert is reading waits
// its turn instead of speaking over it.
func (d *director) resumeRead() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	job := d.heldRead()
	if job == nil {
		return false
	}
	job.paused = false
	d.settle()
	return true
}

// readPaused reports whether a listener has a read paused.
func (d *director) readPaused() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.heldRead() != nil
}

// first is the waiting job that goes next: the highest class, then arrival.
func (d *director) first() *narrationJob {
	var best *narrationJob
	for _, j := range d.waiting {
		if j.paused || !live(j) {
			continue // a listener's hold, or a job already gone: neither takes a turn
		}
		if best == nil || j.class > best.class || (j.class == best.class && j.seq < best.seq) {
			best = j
		}
	}
	return best
}

func (d *director) remove(job *narrationJob) {
	for i, j := range d.waiting {
		if j == job {
			d.waiting = append(d.waiting[:i], d.waiting[i+1:]...)
			return
		}
	}
}

// releaseBed hands the bed back after a rail drain (MVS-D-67).
//
// IT DECIDES UNDER THE ARBITER'S OWN LOCK, and then defers to settle — which
// already owns "when may the bed come back" and is the only thing that should.
// A job on the air, waiting or suspended leaves the bed down, and whichever
// sequence finishes last lifts it by the ordinary path.
//
// AN EARLIER VERSION ASKED AND THEN ACTED, with the question answered outside
// the effector's lock and the lift taken after it: a job admitted in between had
// the bed restored out from under it, which is the broadcast surging to full
// volume over a read in progress. Holding the effector's lock across the
// question instead would invert the lock order every other path takes — settle
// calls into the effector while holding this lock — and deadlock.
func (d *director) releaseBed() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mc.unhold()
	d.settle()
}

// narrateDebug records who has the air and who was displaced (DR-23).
//
// THE ARBITER IS THE ONE THING THE TIMELINE COULD NOT SEE. A takeover reading
// over a live `[w]` report was reported at UAT 2026-09-05 and could not be
// reproduced: the ranking is right, and the read is paused before the tone
// sounds (TestATakeoversToneNeverSoundsOverALiveRead). What no test reaches is
// the audio layer underneath — whether the pause actually stopped the sound. A
// line here says which job held the air and which was displaced, so the next
// occurrence says whether the arbiter decided wrongly or the decision did not
// take effect. Those are different defects with the same symptom.
func narrateDebug(what string, job *narrationJob) {
	if !radioDebugOn() || job == nil {
		return
	}
	class := "read"
	if job.class == narrateBreaking {
		class = "breaking"
	}
	radioDebugLog("narrate:" + what + ":" + class + ":" + job.role.String())
}
