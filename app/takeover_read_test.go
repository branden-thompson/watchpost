package app

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// A TAKEOVER'S TONE NEVER SOUNDS OVER A LIVE READ.
//
// UAT 2026-09-05: the HUM LEAD heard a weather alert and a `[w]` read at once.
// TestEventReadIsSuspendedByABreakingTakeover already pins the suspension and
// passes — but it drives the takeover with `s.line()` directly, so no test had
// ever put a REAL takeover opening against a read in progress, and the tone is
// the first thing a listener hears.
//
// This closes that gap: the takeover runs through readScript, tone and all. The
// arbiter is correct at this layer — the read is paused before the tone sounds —
// which is worth pinning precisely because it narrows where the reported defect
// can be: not in the ranking, and not in the ordering of pause against tone.
func TestATakeoversToneNeverSoundsOverALiveRead(t *testing.T) {
	v := &scriptVoice{}
	nar := testDirector(v, nil)
	readGate := make(chan struct{})
	var sleeps atomic.Int32
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		if sleeps.Add(1) == 1 {
			<-readGate
		}
		return ctx.Err() == nil
	}
	var mu sync.Mutex
	reading := make(chan struct{}, 1)
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(m tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := m.(tty.SevereReadingMsg); ok && v.Key != "" {
			reading <- struct{}{}
		}
	})
	r.status = func(string, string, string, time.Duration) {}
	v.dur, v.toneDur = 300*time.Millisecond, 50*time.Millisecond
	r.Read("k1")
	<-reading
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline) && sleeps.Load() == 0; {
		time.Sleep(5 * time.Millisecond)
	}

	// THE REAL TAKEOVER OPENING: a tone, then the words, through readScript.
	sc := lineup.Script{Tone: cast.Classify("Tornado Warning").Key(), Parts: []lineup.Part{
		{Kind: lineup.PartLine, Text: "breaking", Ref: "e1"},
	}}
	nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(ctx context.Context, s *speaker) {
		readScript(s, sc, readHooks{})
	})
	close(readGate)
	time.Sleep(200 * time.Millisecond)

	got := v.got()
	t.Logf("voice sequence: %s", got)
	// The read must be PAUSED before the takeover makes any sound at all —
	// including its tone, which is the first thing a listener hears.
	pause, tone := strings.Index(got, "pause"), strings.Index(got, "tone")
	if pause == -1 {
		t.Fatalf("the read was never paused: %s", got)
	}
	if tone != -1 && tone < pause {
		t.Errorf("the tone sounded BEFORE the read was paused — both are audible: %s", got)
	}
}
