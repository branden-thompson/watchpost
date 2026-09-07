package app

import (
	"context"
	"testing"
	"time"
)

// holdRest's `rest <= 0` BRANCH IS THE AIR CHECK, and nothing pinned it.
//
// Found the hard way (2026-09-06): mH0 disables that branch, the mutation was
// committed by accident, and the whole suite — including `-race` — went green
// over it. The mutant's own CAUGHT verdict had been a false positive from a
// flaky test failing under load during the run.
//
// What the branch is for: `hold(d)` loops `for i < steps && d > 0`, so a
// non-positive d never enters the loop and never reaches awaitAir — it returns
// true immediately. When the work outran the sound there is nothing left to
// WAIT for, but there is still something to CHECK: whether the sequence is
// still on the air. Without the branch a takeover whose sequence ended during
// an overlapped render carries on to the next line and cues the band for a read
// that never happens.
//
// Reachable on every muted-class burst, where the tone's duration is zero and
// the first remainder is already non-positive.
func TestHoldRestChecksTheAirWhenTheWorkOutranTheSound(t *testing.T) {
	live := func() (*speaker, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		return &speaker{ctx: ctx, sleep: func(context.Context, time.Duration) bool { return true }}, cancel
	}
	overran := time.Now().Add(-time.Second) // work that took a second longer than the sound

	// THE RULE: the sequence has ended, and the work outran the sound. There is
	// nothing to wait for — and the answer is still NO, because the air is gone.
	s, cancel := live()
	cancel()
	if holdRest(s, time.Millisecond, overran) {
		t.Error("a read whose sequence had ended carried on because the work outran the sound: " +
			"nothing left to wait is not the same as nothing to check")
	}

	// CONTROL 1 — the same non-positive remainder, but the sequence is still on
	// air: it proceeds. Without this the assertion above is satisfied by a
	// holdRest that always refuses.
	s2, cancel2 := live()
	defer cancel2()
	if !holdRest(s2, time.Millisecond, overran) {
		t.Error("a live sequence must carry on when the work outran the sound")
	}

	// CONTROL 2 — an ended sequence with time still to wait also stops, so the
	// rule above is about the AIR and not about the remainder's sign.
	s3, cancel3 := live()
	cancel3()
	if holdRest(s3, time.Millisecond, time.Now()) {
		t.Error("an ended sequence must not hold")
	}
}
