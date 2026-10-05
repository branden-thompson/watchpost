package app

import (
	"context"
	"strings"
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
	r := newPipelineRig(t)
	r.deck.Inject(globalfeed.Event{
		ID: "injected-1", Type: "Tornado Warning", Location: "Olathe, KS",
		Class: globalfeed.ClassSevereWx, Severity: globalfeed.SevRed,
		Source: "NWS", At: time.Now().Add(-time.Minute), Until: time.Now().Add(time.Hour),
	})
	r.cycle()

	// The three consumers a real alert has.
	if len(r.tapes()) == 0 {
		t.Error("the ticker tape never saw it")
	}
	if len(r.bands()) == 0 {
		t.Error("the takeover never cued it — the audio path was not exercised")
	}
	if rows, _ := r.sev.currentRows(); len(rows) == 0 {
		t.Error("the severe index never saw it: the [w] window would be empty")
	}

	// AND THE QUEUE IS DRAINED, NOT HELD. An alert that arrived once does not
	// keep arriving; leaving it queued would re-fire it every cycle.
	if got := r.deck.takeInjected(); len(got) != 0 {
		t.Errorf("the cycle drains the queue, got %d still waiting", len(got))
	}
}

// A TEST EVENT SAYS SO WHEREVER IT GOES (0.18.0 D-152, NFR-2 restated).
//
// The injector ships because nothing it makes can be mistaken for the real
// thing: the rule is no UNMARKED fabricated hazard. So each scenario the
// window offers is sent through the window's own hook and one real cycle, and
// every consumer a real alert has must carry the mark - the tape's items and
// the band's, [w]'s rows, and the words read aloud, which open and close with
// the test script. A new consumer that drops the mark fails here.
func TestATestEventIsMarkedOnEverySurface(t *testing.T) {
	for _, sc := range debugScenarios() {
		r := newPipelineRig(t)
		lp := &livePipelines{ticker: r.deck}
		lp.injectHook()(sc.Key)
		r.cycle()

		tape, band := r.tapes(), r.bands()
		if len(tape) == 0 || len(band) == 0 {
			t.Errorf("%s: never reached the tape (%d) or the band (%d)", sc.Label, len(tape), len(band))
		}
		for _, it := range append(tape, band...) {
			if !it.Test {
				t.Errorf("%s: %q reached the ticker unmarked", sc.Label, it.Head)
			}
		}
		rows, _ := r.sev.currentRows()
		if len(rows) == 0 {
			t.Errorf("%s: never reached [w]", sc.Label)
		}
		for _, row := range rows {
			if !row.Test {
				t.Errorf("%s: [w] lists %q unmarked", sc.Label, row.Product)
			}
		}
		var lines []string
		for _, c := range strings.Split(r.voice.got(), ",") {
			if l, ok := strings.CutPrefix(c, "aside:"); ok {
				lines = append(lines, l)
			}
		}
		if len(lines) < 2 || !strings.HasPrefix(lines[0], "This is a test of the Watchpost alert events system") ||
			!strings.HasPrefix(lines[len(lines)-1], "This concludes the test of the Watchpost alert events system") {
			t.Errorf("%s: the read must open and close with the test script: %q", sc.Label, lines)
		}
	}
}

// pipelineRig is a ticker deck wired as the app wires it - the Director, the
// band's owner, the severe index, the seen store - with no sources, so what is
// injected is the whole feed, and every message the surfaces get is kept.
type pipelineRig struct {
	mu         sync.Mutex
	tape, band []tty.TickerItem
	voice      *scriptVoice
	sev        *severeDeck
	deck       *tickerDeck
	st         *station
}

func newPipelineRig(t *testing.T) *pipelineRig {
	t.Helper()
	r := &pipelineRig{voice: &scriptVoice{}}
	nar := testDirector(r.voice, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	r.sev = newSevereDeck(func(tea.Msg) {})
	r.deck = &tickerDeck{
		send: func(m tea.Msg) {
			r.mu.Lock()
			defer r.mu.Unlock()
			// The BAND does not come through here — it goes through
			// mastercontrol, its one owner (D-1).
			if tm, ok := m.(tty.TickerMsg); ok {
				r.tape = append(r.tape, tm.Items...)
			}
		},
		muted: &atomic.Bool{},
		mc: newMastercontrol(nil, func(m tea.Msg) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if bm, ok := m.(tty.TickerBreakingMsg); ok {
				r.band = append(r.band, bm.Item)
			}
		}),
		voice:  nar,
		seen:   loadSeen(t.TempDir(), time.Hour),
		severe: r.sev,
		watch:  func() []snapshot.LocationRef { return nil },
	}
	r.deck.warm.Store(true) // past the first cycle, or the seed swallows it quietly
	// THE DECK REPORTS INTO A SCHEDULE, or the takeover reaches nobody: since
	// T3.10b the producer hands arrivals to the DIRECTOR through deck.emit.
	r.st = newStation(t, r.deck)
	return r
}

// cycle runs ONE REAL CYCLE - the same one a fetched event goes through - and
// the station's effects, deterministically in this goroutine.
func (r *pipelineRig) cycle() {
	r.deck.cycle(context.Background())
	r.st.drain(context.Background())
}

func (r *pipelineRig) tapes() []tty.TickerItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]tty.TickerItem(nil), r.tape...)
}

func (r *pipelineRig) bands() []tty.TickerItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]tty.TickerItem(nil), r.band...)
}
