package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// A TONE IS A PROMISE OF WORDS (F-43).
//
// On a weather radio the attention tone means "something is about to be said".
// Sounding one and saying nothing spends the listener's attention on nothing,
// and teaches them the tone can be ignored — which is the one thing it must
// never mean.
//
// NOTHING PINNED THIS. app/tone_latency_test.go measures how FAST the tone
// starts, not that words follow it, so the suite would have watched a burst
// sound three tones in silence — which is what the HUM LEAD heard at UAT
// 2026-09-06 — and reported nothing.
func TestASoundedToneIsFollowedByWords(t *testing.T) {
	v := &classVoice{}
	nar := testDirector(v, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	seen := loadSeen(t.TempDir(), time.Hour)
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: seen}

	fresh := breakingFixture()
	newStation(t, deck).takeover(context.Background(), fresh)

	classes, _ := v.seen()
	if len(classes) == 0 {
		t.Fatal("no tone sounded, so this pins nothing about what follows one")
	}
	// The promise: something was said. The seen store is marked line by line AS
	// EACH IS SPOKEN, so a marked alert is a spoken one.
	if !seen.set()[fresh[0].ID] {
		t.Errorf("a tone sounded and nothing was said: %d tone(s), no alert read aloud. "+
			"On a weather radio that teaches a listener the tone can be ignored.", len(classes))
	}
}

// AND WHEN THE READ DOES GIVE UP AFTER THE TONE, IT SAYS SO.
//
// The cut-short is Routed (I-2), so it raises no window — correctly, because
// pressing esc must not claim the relay is dead. The cost is that this failure
// is otherwise silent, and it took a reconstruction from a UAT report to place
// it. The timeline now carries the line.
// tonedVoice sounds a tone with REAL duration, so the hold that follows it is a
// hold and not an instant return. classVoice returns 0, which sends holdRest
// down its awaitAir branch and never poses the window F-43 is about.
type tonedVoice struct {
	scriptVoice
	dur time.Duration
}

func (v *tonedVoice) tone(cast.Class) time.Duration { return v.dur }

// AND WHEN THE READ DOES GIVE UP AFTER THE TONE, IT SAYS SO.
//
// The cut-short is Routed (I-2), so it raises no window — correctly, because
// pressing esc must not claim the relay is dead. The cost is that this failure
// is otherwise SILENT, and placing it took a reconstruction from a UAT report.
// The timeline carries the line now.
func TestGivingUpAfterTheToneIsTraced(t *testing.T) {
	log := filepath.Join(t.TempDir(), "radio.log")
	t.Setenv("WATCHPOST_DEBUG_RADIO", log)

	v := &tonedVoice{dur: 300 * time.Millisecond}
	nar := testDirector(v, nil)
	ctx, cancel := context.WithCancel(context.Background())
	// The sequence ends DURING the tone's hold — the window F-43 names, and the
	// only one in which a tone is sounded and nothing follows it.
	nar.sleep = func(c context.Context, d time.Duration) bool {
		cancel()
		return c.Err() == nil
	}
	ok := nar.Run(ctx, narrateBreaking, cast.Breaking, true, func(_ context.Context, s *speaker) {
		readScript(s, lineup.Script{
			Tone:  cast.ClassWarning.Key(),
			Parts: []lineup.Part{{Kind: lineup.PartLine, Text: "a tornado warning", Ref: "a"}},
		}, readHooks{})
	})
	_ = ok
	cancel()

	b, _ := os.ReadFile(log)
	if !strings.Contains(string(b), "read:gaveup:after-tone") {
		t.Errorf("a read that gave up after the tone left no trace, so the next sighting needs the "+
			"same reconstruction this one did:\n%s", string(b))
	}
}
