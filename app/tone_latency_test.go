package app

import (
	"context"
	"github.com/branden-thompson/watchpost/app/internal/apptest"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/internal/allocpin"
)

// The time-to-tone-start instrument (07-readiness/perf-protocol.md §1 item 1 is
// the one owner of the method; this file only implements it). It measures the
// Go path a breaking takeover walks from breaking()'s entry to the moment the
// arbiter asks for the attention tone — its admission — and nothing
// of the audio behind it: the fake voice stamps the tone() call and ends the
// sequence there.
//
// It runs on BOTH trees: copied into a worktree of the tag v0.13.0 it records
// the "before" number, and here it is the regression pin for the path the cast
// resolution sits on. The pass rule is a comparison
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

// fault is inert here: the seam exists so a read can report a tone with no
// words (FR-9.2).
func (v *toneStampVoice) fault(string) {}

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
	// LIVE, NOT A FIXED PAST DATE (red team 2026-09-05, I-8). eventsFor asks
	// whether an alert is still active before composing it, so a fixed past
	// date would ask the station to read a warning long expired.
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
			voice: testArbiter(voice, nil),
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

// TestTimeToToneAllocBudget holds the time-to-tone path's allocations, the
// span BenchmarkTimeToToneStart times, from the station's wiring to the tone
// being asked for (0.19.6 A-17, M1): 185 allocations a takeover at 0.19.6
// DISCOVER. The wall time stays the benchmark's, reconciled with D-2 in 0.19.7
// (PB-8); the count is deterministic, so it is held now. Each run builds its
// deck outside the window, as the benchmark builds it with the timer stopped.
func TestTimeToToneAllocBudget(t *testing.T) {
	if apptest.RaceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	fresh := breakingFixture()
	var cancels []context.CancelFunc
	defer func() {
		for _, cancel := range cancels { // bounded by the runs (P10-02)
			cancel()
		}
	}()
	var voices []*toneStampVoice
	// A TAKEOVER LEAVES GOROUTINES that end after its context is cancelled, and
	// one still running allocates inside the next run's window. Without the
	// wait the count reads 186 in some runs and the bytes move 0.5 % between
	// them. idle is the goroutines before any takeover.
	idle := runtime.NumGoroutine()
	takeover := func() func() {
		for deadline := time.Now().Add(time.Second); runtime.NumGoroutine() > idle; { // bounded by the deadline (P10-02)
			if time.Now().After(deadline) {
				t.Fatalf("a takeover's goroutines did not end: %d running, %d before any", runtime.NumGoroutine(), idle)
			}
			time.Sleep(time.Millisecond)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		voice := &toneStampVoice{cancel: cancel}
		voices = append(voices, voice)
		deck := &tickerDeck{
			send:  func(tea.Msg) {},
			muted: &atomic.Bool{}, // not muted: an audible takeover sounds the tone
			voice: testArbiter(voice, nil),
			seen:  loadSeen(t.TempDir(), time.Hour),
		}
		return func() { newStation(t, deck).takeover(ctx, fresh) }
	}
	// The first takeover in a process carries one-time costs, so one runs
	// before the window and the pin is the path every later takeover walks.
	takeover()()
	c := allocpin.MeasureEach(t, 20, takeover)
	allocpin.ReportCount(t, "time to tone", c.Allocs, "the fixture is declared at the wall clock and they fell 0.2 % over ten runs")
	for i, v := range voices { // bounded by the runs (P10-02)
		if v.at.IsZero() {
			t.Fatalf("run %d never asked for the tone: the pin is measuring nothing", i)
		}
	}
	// Measured 185 allocations at 0.19.6 B0: the budget is the measurement ×
	// 1.05, as the other pins' are, and tools/pins holds the count to its
	// baseline's maximum. THE BYTES ARE NOT HELD: the fixture is declared at
	// the wall clock (I-8), and they fell from 21,726 to 21,680 over ten runs
	// in ten minutes, past the 0.1 % the comparator allows. 0.19.6 A-17 holds
	// the count.
	const allocs = 185 * 1.05
	if float64(c.Allocs) > allocs {
		t.Errorf("a takeover to its tone costs %d allocations; its budget is %.0f", c.Allocs, allocs)
	}
}
