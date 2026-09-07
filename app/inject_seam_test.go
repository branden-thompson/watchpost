//go:build watchpost_debug

package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// AN INJECTED ALERT CROSSES WHAT A REAL ONE CROSSES (F-21b).
//
// This is the test the whole tool stands on. An injector that handed a card
// straight to the reader would make a takeover sound on demand and prove
// NOTHING about the wiring — which is the failure this session keeps finding,
// most recently three tests that passed over an inert Settings row by calling
// its cycle function instead of pressing a key (D-12).
//
// So the assertion is not "audio happened". It is that the event reached the
// severe index, the ticker tape AND the takeover — the three consumers a real
// alert has — from a single Inject call.
func TestAnInjectedAlertCrossesTheWholePipeline(t *testing.T) {
	var mu sync.Mutex
	var tape, breaking int
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	sev := newSevereDeck(func(tea.Msg) {})
	deck := &tickerDeck{
		send: func(m tea.Msg) {
			mu.Lock()
			defer mu.Unlock()
			// The BAND does not come through here — it goes through
			// mastercontrol, its one owner (D-1), which is why the cue is
			// counted there.
			if _, ok := m.(tty.TickerMsg); ok {
				tape++
			}
		},
		muted: &atomic.Bool{},
		mc: newMastercontrol(nil, func(m tea.Msg) {
			mu.Lock()
			defer mu.Unlock()
			if _, ok := m.(tty.TickerBreakingMsg); ok {
				breaking++
			}
		}),
		voice:  nar,
		seen:   loadSeen(t.TempDir(), time.Hour),
		severe: sev,
		watch:  func() []snapshot.LocationRef { return nil },
	}
	deck.warm.Store(true) // past the first cycle, or the seed swallows it quietly

	// THE DECK REPORTS INTO A SCHEDULE, or the takeover reaches nobody. Since
	// T3.10b the producer hands arrivals to the DIRECTOR through deck.emit; a
	// deck with no schedule wired drops them on the floor, and this test's
	// takeover assertion would have been measuring nothing. It waited on
	// deck.breakers instead — a WaitGroup T3.10b deleted along with the
	// goroutine — so the file stopped compiling and no gate ran the tag.
	st := newStation(t, deck)

	deck.Inject(globalfeed.Event{
		ID: "injected-1", Type: "Tornado Warning", Location: "Olathe, KS",
		Class: globalfeed.ClassSevereWx, Severity: globalfeed.SevRed,
		Source: "NWS", At: time.Now().Add(-time.Minute), Until: time.Now().Add(time.Hour),
	})

	// ONE REAL CYCLE. The deck has no sources, so the injected event is the
	// only thing in the feed — and it goes through the SAME cycle a fetched one
	// would, which is the whole claim.
	deck.cycle(context.Background())
	// DETERMINISTIC, with nothing to synchronise on: the station steps the
	// Director and performs every effect in this goroutine (station_test.go).
	st.drain(context.Background())

	mu.Lock()
	gotTape, gotBreaking := tape, breaking
	mu.Unlock()

	// The three consumers a real alert has.
	if gotTape == 0 {
		t.Error("the ticker tape never saw it")
	}
	if gotBreaking == 0 {
		t.Error("the takeover never cued it — the audio path was not exercised")
	}
	if rows, _ := sev.currentRows(); len(rows) == 0 {
		t.Error("the severe index never saw it: the [w] window would be empty")
	}

	// AND THE QUEUE IS DRAINED, NOT HELD. An alert that arrived once does not
	// keep arriving; leaving it queued would re-fire it every cycle.
	if got := deck.takeInjected(); len(got) != 0 {
		t.Errorf("the cycle drains the queue, got %d still waiting", len(got))
	}
}
