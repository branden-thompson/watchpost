package app

// narrate.go — the narrator: the one owner of the correspondent's voice
// (HUM LEAD 2026-08-28, "build it robust, build it right"). Every spoken
// sequence — a breaking takeover, an event read, whatever comes next — runs
// through Run with a class; the narrator serialises them, ducks the
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
	"sync"
	"time"
)

type narrationClass int

const (
	narrateRead narrationClass = iota
	narrateBreaking
)

// clip is one rendered line: the audio and how long it plays; viz says
// whether the visualizer follows it (set by the narrator from the class —
// vizFor). text rides along for fakes and diagnostics.
type clip struct {
	text string
	pcm  []byte
	rate int
	dur  time.Duration
	viz  bool
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
	tone() time.Duration
	render(ctx context.Context, text string) (clip, bool) // bound to the sequence: a cancelled job renders no further (R5-B-07)
	play(c clip)
	pause()
	resume()
	discard()
	restore()
}

type narrationJob struct {
	class     narrationClass
	audible   bool // a muted or voiceless sequence runs its visuals only and never dips the broadcast
	ctx       context.Context
	seq       uint64 // arrival order within a class
	suspended bool   // taken off the air by a higher class; resumes when it ends
}

// narrator arbitrates the voice.
type narrator struct {
	mu        sync.Mutex
	turn      *sync.Cond
	v         narrationVoice
	onAir     *narrationJob
	suspended []*narrationJob // the jobs a higher class took the air from, innermost last
	waiting   []*narrationJob
	ducked    bool
	next      uint64
	sleep     func(ctx context.Context, d time.Duration) bool // sleepCtx; tests replace it
}

func newNarrator(v narrationVoice) *narrator {
	n := &narrator{v: v, sleep: sleepCtx}
	n.turn = sync.NewCond(&n.mu)
	return n
}

// silent reports whether there is no voice at all.
func (n *narrator) silent() bool { return n == nil || n.v == nil }

// speaker is what a running sequence speaks through: every call honours
// the job's context (an ended sequence hears 0 and false), its suspension
// (a line waits, a hold stops counting) and the audible flag (a muted
// sequence runs its visuals only, and never ducks).
type speaker struct {
	n     *narrator // nil for a nil narrator: the sequence runs silently
	job   *narrationJob
	ctx   context.Context
	sleep func(ctx context.Context, d time.Duration) bool
}

// holdStep is how often a hold checks for suspension or an ended context.
const holdStep = 100 * time.Millisecond

// attention sounds the attention tone; 0 when silent, muted or ended.
func (s *speaker) attention() time.Duration {
	if !s.live() {
		return 0
	}
	return s.n.v.tone()
}

// line renders and plays one line, returning how long it plays; 0 when
// silent, muted or ended (the caller keeps its visual hold). A line rendered
// while the job is suspended waits for the air before it starts, and starts
// under the arbiter's lock: a takeover admitted meanwhile finds it in flight
// and pauses it, never plays over it (red-team round 4, A-04).
func (s *speaker) line(text string) time.Duration {
	if !s.live() {
		return 0
	}
	c, ok := s.n.v.render(s.ctx, text)
	if !ok {
		return 0
	}
	c.viz = vizFor(s.job.class)
	if !s.awaitAir(func() { s.n.v.play(c) }) {
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
	if s.n == nil || s.job == nil {
		return s.ctx.Err() == nil
	}
	s.n.mu.Lock()
	defer s.n.mu.Unlock()
	// A condition-variable wait: one Wait per iteration, ended by the resume
	// or the context ending (the P10-02 ledger row for admit covers the shape).
	for s.job.suspended && s.ctx.Err() == nil {
		s.n.turn.Wait()
	}
	if s.ctx.Err() != nil {
		return false
	}
	if then != nil {
		then()
	}
	return true
}

func (s *speaker) live() bool {
	return s.job != nil && s.job.audible && !s.n.silent() && s.ctx.Err() == nil
}

// Run runs seq as a narration of this class: it waits for its turn (a
// higher class on air finishes first; a lower one is suspended), ducks the
// broadcast if the sequence is audible and nothing else already did, runs
// seq, and restores the broadcast when nothing is waiting or suspended. It
// returns false when ctx ended before the sequence finished.
func (n *narrator) Run(ctx context.Context, class narrationClass, audible bool, seq func(ctx context.Context, s *speaker)) bool {
	if n == nil {
		seq(ctx, &speaker{ctx: ctx, sleep: sleepCtx}) // no narrator at all: run the visuals, in real time
		return ctx.Err() == nil
	}
	job := &narrationJob{class: class, audible: audible && !n.silent(), ctx: ctx}
	// The context's end wakes every wait this job may be parked in — the
	// turn wait in admit, the air wait while suspended (REVIEW R5-B-02: a
	// cancel landed while parked stayed parked until the takeover's release).
	stop := context.AfterFunc(ctx, func() {
		n.mu.Lock()
		n.turn.Broadcast()
		n.mu.Unlock()
	})
	defer stop()
	if !n.admit(job) {
		return false
	}
	seq(ctx, &speaker{n: n, job: job, ctx: ctx, sleep: n.sleep})
	n.release(job)
	return ctx.Err() == nil
}

// admit queues the job, suspends a lower class on air, and waits for the turn.
func (n *narrator) admit(job *narrationJob) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.next++
	job.seq = n.next
	if on := n.onAir; on != nil && job.class > on.class {
		on.suspended = true // its line pauses, its holds stop counting, a line it is rendering waits
		n.suspended = append(n.suspended, on)
		n.onAir = nil
		if n.v != nil && on.audible {
			n.v.pause()
		}
	}
	n.waiting = append(n.waiting, job)
	// A condition-variable wait: one Wait per iteration, ended by the turn
	// arriving or the job's context ending (checked on every wake) — the
	// scheduler tier loops' shape (P10-02 ledger row, app/narrate.go admit).
	for n.onAir != nil || n.first() != job {
		if job.ctx.Err() != nil { // cancelled while waiting: whoever it outranked may resume
			n.remove(job)
			n.settle()
			return false
		}
		n.turn.Wait()
	}
	n.remove(job)
	n.onAir = job
	if job.audible && !n.ducked {
		n.v.duck()
		n.ducked = true
	}
	return true
}

// release ends a job: the one on air leaves it; one that ended while
// SUSPENDED (its context ran out under a takeover) leaves the suspended
// stack and its held line is discarded, the air untouched (round 4, A-02).
// Then the narrator settles.
func (n *narrator) release(job *narrationJob) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.onAir == job {
		n.onAir = nil
	} else if job.suspended {
		n.unsuspend(job)
		if n.v != nil && job.audible {
			n.v.discard()
		}
	} else {
		return // already discarded by settle: nothing of it remains
	}
	n.settle()
}

// settle decides who has the air when it frees (a release, a cancelled wait):
// a waiting job of a class above the innermost suspended one goes first,
// else the suspended job resumes where it stopped; the broadcast is restored
// only when nothing waits or is suspended. Callers hold n.mu.
func (n *narrator) settle() {
	defer n.turn.Broadcast()
	if n.onAir != nil {
		return
	}
	// A suspended job whose context ended while it waited is discarded, never
	// resumed (R5-B-02: its held line played out while the duck was already
	// lifted); the next one down gets the air. Bounded by the stack (P10-02).
	for s := n.innermostSuspended(); s != nil && s.ctx.Err() != nil; s = n.innermostSuspended() {
		n.unsuspend(s)
		if n.v != nil && s.audible {
			n.v.discard()
		}
	}
	if s := n.innermostSuspended(); s != nil && (n.first() == nil || n.first().class <= s.class) {
		n.unsuspend(s)
		n.onAir = s
		if n.v != nil && s.audible {
			n.v.resume()
		}
		return
	}
	if len(n.waiting) == 0 && len(n.suspended) == 0 && n.ducked && n.v != nil {
		n.v.restore()
		n.ducked = false
	}
}

// unsuspend takes job off the suspended stack. Callers hold n.mu.
func (n *narrator) unsuspend(job *narrationJob) {
	job.suspended = false
	for i, j := range n.suspended {
		if j == job {
			n.suspended = append(n.suspended[:i], n.suspended[i+1:]...)
			return
		}
	}
}

func (n *narrator) innermostSuspended() *narrationJob {
	if len(n.suspended) == 0 {
		return nil
	}
	return n.suspended[len(n.suspended)-1]
}

// first is the waiting job that goes next: the highest class, then arrival.
func (n *narrator) first() *narrationJob {
	var best *narrationJob
	for _, j := range n.waiting {
		if best == nil || j.class > best.class || (j.class == best.class && j.seq < best.seq) {
			best = j
		}
	}
	return best
}

func (n *narrator) remove(job *narrationJob) {
	for i, j := range n.waiting {
		if j == job {
			n.waiting = append(n.waiting[:i], n.waiting[i+1:]...)
			return
		}
	}
}
