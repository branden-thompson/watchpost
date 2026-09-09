package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
)

// P3(c): the rotation reads as its OWN class, below the severe read.
//
// THE ORDERING IS THE POINT. Once the rotation travels the card path it is a
// narration job like any other, and the arbiter decides who yields to whom by
// class. A rotation read must give way to BOTH a severe read and a takeover —
// it is the programme, and both of those interrupt the programme.

func TestARotationReadIsSuspendedByASevereRead(t *testing.T) {
	v := &scriptVoice{}
	n := testDirector(v, nil)
	n.sleep = sleepCtx
	started, done := make(chan struct{}), make(chan bool, 1)
	go n.Run(context.Background(), narrateRotation, cast.Standard, true, func(ctx context.Context, s *speaker) {
		s.line("rotation-line")
		close(started)
		done <- s.hold(300 * time.Millisecond)
	})
	<-started
	ok := n.Run(context.Background(), narrateRead, cast.SevereRead, true, func(ctx context.Context, s *speaker) {
		s.line("read-line")
		s.hold(200 * time.Millisecond)
	})
	if !ok {
		t.Fatal("the severe read must take the air over the rotation")
	}
	// PAUSE then RESUME around the read is the suspension. It is NOT an
	// "aside": that marks a TAKEOVER's line, whose visualizer does not follow
	// it — my first assertion said aside and the code was right, not the test.
	if got := v.got(); !strings.Contains(got, "pause,speak:read-line,resume") {
		t.Fatalf("the rotation must be SUSPENDED around the severe read (pause ... resume), got: %s", got)
	}
	select {
	case finished := <-done:
		if !finished {
			t.Fatal("the rotation must RESUME after the read, not be cut — it is the programme")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the rotation never resumed")
	}
}

func TestARotationReadIsSuspendedByATakeover(t *testing.T) {
	v := &scriptVoice{}
	n := testDirector(v, nil)
	n.sleep = sleepCtx
	started, done := make(chan struct{}), make(chan bool, 1)
	go n.Run(context.Background(), narrateRotation, cast.Standard, true, func(ctx context.Context, s *speaker) {
		s.line("rotation-line")
		close(started)
		done <- s.hold(300 * time.Millisecond)
	})
	<-started
	if ok := n.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(ctx context.Context, s *speaker) {
		s.line("breaking-line")
		s.hold(200 * time.Millisecond)
	}); !ok {
		t.Fatal("a takeover must take the air over the rotation")
	}
	// ASSERT THE SUSPENSION, not merely that the rotation finished.
	//
	// WRITTEN BECAUSE A PLANT SURVIVED: disabling the arbiter's suspend
	// entirely failed nothing here, because a rotation that is never paused
	// finishes just as happily as one that is paused and resumed. "It
	// completed" was never evidence that anything gave way to the takeover.
	if got := v.got(); !strings.Contains(got, "pause,aside:breaking-line,resume") {
		t.Fatalf("the rotation must be SUSPENDED around the takeover (pause ... resume), got: %s", got)
	}
	select {
	case finished := <-done:
		if !finished {
			t.Fatal("the rotation must resume after the takeover")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the rotation never resumed")
	}
}

// The rotation is the LOWEST class: nothing yields to it.
//
// DERIVED (INST-1): it walks the class type by its sentinel rather than
// naming the classes, so a class added later is covered without editing this.
func TestTheRotationIsTheLowestClass(t *testing.T) {
	for c := narrationClass(0); c < numNarrationClasses; c++ {
		if c == narrateRotation {
			continue
		}
		if c < narrateRotation {
			t.Errorf("class %d outranks the rotation the wrong way: the rotation is the programme "+
				"and everything else interrupts it", c)
		}
	}
	if narrateRotation != 0 {
		t.Errorf("the rotation must be the lowest class; it is %d", narrateRotation)
	}
}
