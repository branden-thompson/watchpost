package synth

import (
	"context"
	"errors"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// The one owner of "how many voice renders run at once" (FR-12, plan §1.3).
// Nothing bounded this before 0.14.0: a broadcast reading ahead while a
// takeover renders and a Setup preview plays could have six `say`/`piper`
// processes alive at once, and on Linux each one is a ~63 MB model load.
//
// Slots are N ordinary plus TWO reserved. The reserved pair belongs to the
// Station Director, which sets WithPriority on every job it runs, so a line a
// listener is waiting for never queues behind the broadcast's read-ahead.
// There are two and not one because a takeover is admitted while the read it
// suspended may still hold a reserved slot for an in-flight render
// (MVS-D-43 — it amends the ratified AX-4, which said one).
const (
	// ReservedSlots is the size of the Director's private pool.
	ReservedSlots = 2

	// DefaultBound caps a single Say end to end — the wait for a slot AND the
	// render. Bounding only the render would let a Say queued behind wedged
	// renders wait N x bound before its own clock even started.
	DefaultBound = 90 * time.Second
)

// ErrVoiceUnwired reports a Limited voice built with a nil voice or a nil
// limiter. It fails closed: the wiring error is returned from Say, because
// handing back an uncapped voice is the failure this type exists to prevent.
var ErrVoiceUnwired = errors.New("synth: limited voice is not wired (nil voice or nil limiter)")

// Limiter admits renders: an ordinary pool and the Director's reserved pool,
// each a buffered channel used as a counting semaphore. The zero value is not
// usable — build one with NewLimiter.
type Limiter struct {
	ordinary chan struct{}
	reserved chan struct{}
}

// NewLimiter returns a limiter with ordinary general slots and reserved
// priority slots. Both counts are floored at 1: a limiter that admits nothing
// would silence the station, which is worse than an unbounded one.
func NewLimiter(ordinary, reserved int) *Limiter {
	ordinary = max(ordinary, 1)
	reserved = max(reserved, 1)
	return &Limiter{
		ordinary: make(chan struct{}, ordinary),
		reserved: make(chan struct{}, reserved),
	}
}

// priorityKey types the context value; an unexported struct key cannot
// collide with another package's.
type priorityKey struct{}

// WithPriority marks ctx as a Station Director job, admitting it to the
// reserved pool when the ordinary one is full.
func WithPriority(ctx context.Context) context.Context {
	return context.WithValue(ctx, priorityKey{}, true)
}

// HasPriority reports whether ctx was marked by WithPriority.
func HasPriority(ctx context.Context) bool {
	got, ok := ctx.Value(priorityKey{}).(bool)
	return ok && got
}

// acquire takes a slot, returning the function that gives it back. An ordinary
// job waits on the ordinary pool alone. A priority job prefers an ordinary
// slot when one is free — the reserved pair is a fallback, not a fast lane —
// and waits on both otherwise. The error is ctx's: a bound that fires while
// waiting is reported as the wait it was, not as a render failure.
func (l *Limiter) acquire(ctx context.Context) (func(), error) {
	if err := invariant.Check(l != nil && l.ordinary != nil && l.reserved != nil, "synth: limiter is not initialised"); err != nil {
		return nil, err
	}
	select { // an ordinary slot, if one is free right now, for either kind of job
	case l.ordinary <- struct{}{}:
		return func() { <-l.ordinary }, nil
	default:
	}
	if !HasPriority(ctx) {
		select {
		case l.ordinary <- struct{}{}:
			return func() { <-l.ordinary }, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	select {
	case l.ordinary <- struct{}{}:
		return func() { <-l.ordinary }, nil
	case l.reserved <- struct{}{}:
		return func() { <-l.reserved }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// limitedVoice is the decorator Limited returns. Name and Rate pass straight
// through — only Say is bounded — so a caller can still ask who is speaking
// while every slot is held.
type limitedVoice struct {
	v     Voice
	l     *Limiter
	bound time.Duration
}

// Limited wraps v so every Say is admitted by l and bounded by DefaultBound.
// A nil voice or a nil limiter yields a voice whose Say reports
// ErrVoiceUnwired; it never yields v itself, so a mis-wired caller loses its
// audio rather than its cap.
func Limited(v Voice, l *Limiter) Voice {
	return limitedVoice{v: v, l: l, bound: DefaultBound}
}

// Name implements Voice.
func (lv limitedVoice) Name() string {
	if lv.v == nil {
		return ""
	}
	return lv.v.Name()
}

// Rate implements Voice.
func (lv limitedVoice) Rate() int {
	if lv.v == nil {
		return 0
	}
	return lv.v.Rate()
}

// Say implements Voice: wait for a slot and render, the two together inside
// one bound.
func (lv limitedVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if lv.v == nil || lv.l == nil {
		return nil, ErrVoiceUnwired
	}
	ctx, cancel := context.WithTimeout(ctx, lv.bound)
	defer cancel()
	release, err := lv.l.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return lv.v.Say(ctx, text)
}
