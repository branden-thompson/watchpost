package app

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// I-2 / R-11 — WHAT REACHES A PERSON, AND WHAT DOES NOT.
//
// The RELAY FAULT window is for the fault after which nothing is on the air and
// nothing is coming. It is raised through DR-21's one escalation channel, and
// the grade cannot be "has the schedule emptied" — it ALWAYS has once a rail
// card leaves, because a burst is one card (MVS-D-77) and nothing queues the
// main track, so every deliberate, self-healing decline would raise a modal
// saying the relay was dead.
//
// The composition is tested end to end here: the pieces live in three files,
// and a harness that stubs the escalation with an empty body never joins them.
func TestADeliberateDeclineDoesNotRaiseTheFaultWindow(t *testing.T) {
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	muted := &atomic.Bool{}
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: muted, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
	st := newStation(t, deck)

	// A read declined at the EXECUTOR's mute gate. The producer's gate is one
	// layer up, so the burst is offered first and muted before it is performed —
	// which is the window in which a listener could press [M] in the running app.
	fresh := breakingFixture()
	st.offer(fresh)
	if st.pendingCount() == 0 {
		t.Fatal("the fixture never reached the Director; it poses nothing")
	}
	muted.Store(true)
	st.drain(context.Background())

	if len(st.escalated()) != 0 {
		t.Errorf("a muted read raised the relay-fault window: %q\n"+
			"That modal says the station is dead; here the listener had simply asked for quiet, "+
			"and training them to dismiss it is the noise regression fault.go names.", st.escalated())
	}
}

// THE CHANNEL IS LIVE, which is the other half of R-11: the three pieces of
// "a rail card fails, a person is told" are joined here, with no empty stub to
// keep them apart. An Escalate run through the REAL executors must reach a
// person, carrying the reason the producer gave.
func TestAnEscalationReachesAPersonWithItsReason(t *testing.T) {
	nar := testDirector(&scriptVoice{}, nil)
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
	st := newStation(t, deck)
	st.x.run(context.Background(), lineup.Escalate{ID: "burst:a", Run: 1, Reason: "the schedule stopped"})
	if len(st.escalated()) != 1 {
		t.Fatalf("DR-21's one channel delivered %d escalations, want 1", len(st.escalated()))
	}
	if !strings.Contains(st.escalated()[0], "the schedule stopped") {
		t.Errorf("the escalation lost its reason: %q — a window that says only that the station is quiet "+
			"tells a listener what they already knew", st.escalated()[0])
	}
}
