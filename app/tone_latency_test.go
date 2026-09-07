package app

import (
	"context"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
)

// The time-to-tone-start instrument (07-readiness/perf-protocol.md §1 item 1 is
// the one owner of the method; this file only implements it). It measures the
// Go path a breaking takeover walks from breaking()'s entry to the moment the
// director asks for the attention tone — the arbiter's admission — and nothing
// of the audio behind it: the fake voice stamps the tone() call and ends the
// sequence there.
//
// It runs on BOTH trees. At P0 the same file is copied into a worktree of the
// tag v0.13.0 to record the "before" number; in 0.14.0 it is the regression pin
// for the path the cast resolution is added to. The pass rule is a comparison
// of medians, so the two runs must measure the same thing: keep this file's
// shape identical on both trees and change only what the compiler forces.

// toneStampVoice is a narrationVoice that records when the tone was asked for
// and cancels the sequence at that instant, so nothing after the tone is timed
// and no iteration waits out a hold.
type toneStampVoice struct {
	at     time.Time
	cancel context.CancelFunc
}

func (v *toneStampVoice) duck() {}

func (v *toneStampVoice) tone(cast.Class) time.Duration {
	v.at = time.Now()
	v.cancel() // the measured path ends here; the rest of the sequence must not run
	return 0
}

func (v *toneStampVoice) render(context.Context, cast.Role, string) (clip, bool) {
	return clip{}, false
}
func (v *toneStampVoice) play(clip) {}
func (v *toneStampVoice) pause()    {}
func (v *toneStampVoice) resume()   {}
func (v *toneStampVoice) stop()     {}
func (v *toneStampVoice) discard()  {}
func (v *toneStampVoice) restore()  {}

// breakingFixture is the event a measured takeover reads: one severe-weather
// warning, so the single-event path runs (a burst adds a sort and a second
// marquee send that are not part of the tone path).
func breakingFixture() []globalfeed.Event {
	// LIVE, NOT A FIXED PAST DATE (red team 2026-09-05, I-8). This carried
	// 2026-08-27, so every alert it posed had expired — and once eventsFor
	// started re-asking whether an alert was still active before composing it,
	// the fixture was asking the station to read a warning nine days dead.
	// Nothing here asserts a spoken time, so `now` costs no determinism.
	declared := time.Now()
	return []globalfeed.Event{{
		ID:       "tone-latency",
		Class:    globalfeed.ClassSevereWx,
		Type:     "Tornado Warning",
		Location: "Norfolk, VA",
		Severity: globalfeed.SevRed,
		At:       declared,
		Until:    declared.Add(time.Hour),
	}}
}

// BenchmarkTimeToToneStart reports ns/tone: the wall-clock nanoseconds from
// breaking()'s entry to the tone being asked for, averaged over the run. Read
// it with `go test ./app -run '^$' -bench TimeToToneStart -count=10` and take
// the median and p95 of the ten ns/tone values — ns/op includes the teardown
// after the stamp and is not the number of record.
func BenchmarkTimeToToneStart(b *testing.B) {
	fresh := breakingFixture()
	var total time.Duration

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ctx, cancel := context.WithCancel(context.Background())
		voice := &toneStampVoice{cancel: cancel}
		deck := &tickerDeck{
			send:  func(tea.Msg) {},
			muted: &atomic.Bool{}, // not muted: an audible takeover sounds the tone
			voice: testDirector(voice, nil),
			seen:  loadSeen(b.TempDir(), time.Hour),
		}
		b.StartTimer()

		start := time.Now()
		newStation(b, deck).takeover(ctx, fresh)

		b.StopTimer()
		if voice.at.IsZero() {
			b.Fatal("the takeover never asked for the tone — the instrument is measuring nothing")
		}
		total += voice.at.Sub(start)
		cancel()
		b.StartTimer()
	}

	b.ReportMetric(float64(total.Nanoseconds())/float64(b.N), "ns/tone")
}
