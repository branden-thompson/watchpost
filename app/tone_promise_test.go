package app

import (
	"context"
	"os"
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
	log := radioDebugTo(t, "1")

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

// A TONE WITH NO WORDS IS REPORTED (FR-9.2).
//
// A TONE IS A PROMISE OF WORDS. The listener hears the attention tone, leans
// in, and gets nothing — heard three times in one burst at UAT 2026-09-06 and
// impossible to diagnose from the outside, because a read that ends early is
// Routed and raises nothing (I-2).
//
// The report goes to the OPERATOR'S surface and is never spoken: an operational
// message over the air is confusing and the audience can do nothing about it
// (HUM LEAD, 2026-09-08).
func TestAToneWithNoWordsIsReported(t *testing.T) {
	v := &silentAfterToneVoice{}
	d := testDirector(v, nil)
	d.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	d.Run(context.Background(), narrateRead, cast.All, true, func(ctx context.Context, s *speaker) {
		readScript(s, lineup.Script{Tone: cast.ClassWarning.Key(), Parts: []lineup.Part{
			{Kind: lineup.PartLine, Text: "a tornado warning", Ref: "a"},
		}}, readHooks{})
	})

	if v.faults == 0 {
		t.Error("a tone sounded and nothing followed it, and nothing said so: the listener hears a " +
			"promise and gets silence, and the operator has no way to know")
	}
}

// AND A READ THAT SPOKE IS NOT REPORTED, however it ended afterwards. A read
// that got a line out and then lost the rest is a different failure with a
// different shape, and calling it this one would train the operator to read
// past both.
func TestAToneFollowedByWordsIsNotReported(t *testing.T) {
	v := &speakingToneVoice{}
	d := testDirector(v, nil)
	d.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	d.Run(context.Background(), narrateRead, cast.All, true, func(ctx context.Context, s *speaker) {
		readScript(s, lineup.Script{Tone: cast.ClassWarning.Key(), Parts: []lineup.Part{
			{Kind: lineup.PartLine, Text: "a tornado warning", Ref: "a"},
		}}, readHooks{})
	})
	if v.faults > 0 {
		t.Errorf("a read that spoke reported a silent tone %d times", v.faults)
	}
}

// AND A CARD WITH NO TONE IS NOT REPORTED, however little it says.
//
// This fault is about a PROMISE — the attention tone — being broken. A card
// that never sounded one and rendered nothing is a different failure, and
// reporting it here would put "an alert tone sounded" on the operator's screen
// when no alert tone did.
func TestACardWithNoToneIsNotReported(t *testing.T) {
	v := &silentAfterToneVoice{}
	d := testDirector(v, nil)
	d.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	d.Run(context.Background(), narrateRead, cast.All, true, func(ctx context.Context, s *speaker) {
		readScript(s, lineup.Script{Parts: []lineup.Part{ // no Tone
			{Kind: lineup.PartLine, Text: "a forecast", Ref: "a"},
		}}, readHooks{})
	})
	if v.faults > 0 {
		t.Errorf("a card that sounded no tone reported a broken tone promise %d times", v.faults)
	}
}

// speakingToneVoice sounds a tone and DOES render — the well-behaved case, so
// the report's absence is measured rather than assumed.
type speakingToneVoice struct{ silentAfterToneVoice }

func (v *speakingToneVoice) render(context.Context, cast.Role, string) (clip, bool) {
	return clip{text: "words", dur: time.Millisecond}, true
}

// silentAfterToneVoice sounds a tone and then renders nothing — the shape of a
// synthesiser that has stopped answering while the tone path still works.
type silentAfterToneVoice struct {
	faults int
}

func (v *silentAfterToneVoice) duck()                         {}
func (v *silentAfterToneVoice) tone(cast.Class) time.Duration { return 10 * time.Millisecond }
func (v *silentAfterToneVoice) render(context.Context, cast.Role, string) (clip, bool) {
	return clip{}, false // every part fails to render, and nothing is cancelled
}
func (v *silentAfterToneVoice) play(clip)    {}
func (v *silentAfterToneVoice) pause()       {}
func (v *silentAfterToneVoice) resume()      {}
func (v *silentAfterToneVoice) stop()        {}
func (v *silentAfterToneVoice) discard()     {}
func (v *silentAfterToneVoice) restore()     {}
func (v *silentAfterToneVoice) fault(string) { v.faults++ }
