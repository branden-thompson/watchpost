package player

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordingOutput hands back players that expose their latest volume and only
// count as playing between Play() and their reader draining — so a finite
// alert clip finishes deterministically without racing Play().
type recordingOutput struct {
	mu      sync.Mutex
	players []*recordingPlayer
}

type recordingPlayer struct {
	started atomic.Bool
	plays   atomic.Int32  // every Play(), so a stream that should never have started is visible
	volAt   atomic.Uint64 // the volume in force the first time Play() was called, float64 bits
	drained atomic.Bool
	vol     atomic.Uint64 // float64 bits
	stop    chan struct{}
	once    sync.Once
	acts    []string // Play/Pause/Close in the ORDER they happened (MVS-D-75)
	actMu   sync.Mutex
}

// act records what was done to this player, so a test can assert the ORDER —
// which is the whole of the stop rule: a close alone leaves buffered audio
// sounding, so the pause must come first.
func (p *recordingPlayer) act(what string) {
	p.actMu.Lock()
	p.acts = append(p.acts, what)
	p.actMu.Unlock()
}

func (p *recordingPlayer) actions() []string {
	p.actMu.Lock()
	defer p.actMu.Unlock()
	return append([]string(nil), p.acts...)
}

func (o *recordingOutput) NewPlayer(pcm io.Reader) (Player, error) {
	p := &recordingPlayer{stop: make(chan struct{})}
	o.mu.Lock()
	o.players = append(o.players, p)
	o.mu.Unlock()
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			if _, err := pcm.Read(buf); err != nil {
				p.drained.Store(true)
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	return p, nil
}

func (p *recordingPlayer) Play() {
	// The volume AT the moment audio starts, sampled here rather than polled:
	// setting the right volume just after Play() still puts a tick of
	// full-volume broadcast over an alert, and a poll can never see that window.
	if p.plays.Load() == 0 {
		p.volAt.Store(p.vol.Load()) // sampled BEFORE the count, so no reader sees a play with no sample
	}
	p.act("play")
	p.plays.Add(1)
	p.started.Store(true)
}

// volumeAtPlay is the volume the player had when it first started.
func (p *recordingPlayer) volumeAtPlay() float64 { return float64(int64(p.volAt.Load())) / 1e6 }
func (p *recordingPlayer) Pause()                { p.act("pause"); p.started.Store(false) }
func (p *recordingPlayer) IsPlaying() bool       { return p.started.Load() && !p.drained.Load() }
func (p *recordingPlayer) SetVolume(v float64) {
	p.vol.Store(uint64(int64(v * 1e6)))
}
func (p *recordingPlayer) volume() float64 { return float64(int64(p.vol.Load())) / 1e6 }
func (p *recordingPlayer) Close() error {
	p.act("close")
	p.once.Do(func() { close(p.stop) })
	return nil
}

// endlessPCM never drains: the "main broadcast" that keeps playing so the
// watch loop keeps re-asserting its (ducked) volume.
type endlessPCM struct{}

func (endlessPCM) Read(b []byte) (int, error) { return len(b), nil }

// mainVol reports the first (main-broadcast) player's latest volume, and false
// until that player exists — safe to poll.
func (o *recordingOutput) mainVol() (float64, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.players) == 0 {
		return 0, false
	}
	return o.players[0].volume(), true
}

// overlayVol reports the second player's volume — the alert overlay mixed over
// the main stream.
func (o *recordingOutput) overlayVol() (float64, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.players) < 2 {
		return 0, false
	}
	return o.players[1].volume(), true
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestDuckAndRestoreScaleTheBroadcastOnly(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80) // 0.80

	// The main broadcast: an endless source, playing, standing in for a relay —
	// a live source dips under an alert where a rendered cycle holds.
	e.StartSource("test", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	e.setLive(true)
	atVol := func(want float64) func() bool {
		return func() bool { v, ok := out.mainVol(); return ok && near(v, want) }
	}
	waitFor(t, "main to reach full volume", atVol(0.80))

	// An alert takes the air: a live broadcast dips.
	e.Suppress()
	waitFor(t, "main to duck", atVol(0.80*alertDuck))

	// An overlay (a narration) plays at the KNOB volume — un-ducked — while the
	// main stream stays dipped under it.
	clip := make([]byte, 8*OutputRate)
	if err := e.Preview(OutputRate, bytes.NewReader(clip)); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.overlayVol(); !ok || !near(v, 0.80) {
		t.Fatalf("the overlay rides at the knob (un-ducked): %v %v", v, ok)
	}

	// Restore: the main broadcast returns.
	e.Restore()
	waitFor(t, "main to restore", atVol(0.80))
}

func near(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.005
}

// A held line survives the clips that play while it waits (red-team round
// 4, A-01): a read paused for a takeover is not closed when the takeover's
// tone and lines take the flight slot; ResumePreview plays it on; DropHeld
// closes it without playing. The broadcast underneath is untouched.
func TestHeldLineSurvivesTheTakeoversClips(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	e.StartSource("test", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	waitFor(t, "main to play", func() bool { _, ok := out.mainVol(); return ok })
	if err := e.Preview(OutputRate, bytes.NewReader(make([]byte, 60*OutputRate))); err != nil {
		t.Fatal(err)
	}
	out.mu.Lock()
	read := out.players[len(out.players)-1]
	out.mu.Unlock()
	e.PausePreview()
	if err := e.PreviewAside(OutputRate, bytes.NewReader(make([]byte, OutputRate/2))); err != nil { // the takeover's tone
		t.Fatal(err)
	}
	if err := e.PreviewAside(OutputRate, bytes.NewReader(make([]byte, OutputRate/2))); err != nil { // and a line
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond) // the read's watcher must keep waiting
	select {
	case <-read.stop:
		t.Fatal("the takeover's clips closed the held read")
	default:
	}
	if read.IsPlaying() {
		t.Fatal("the held read must stay paused under the takeover")
	}
	e.Volume(40) // the knob moves while it waits
	e.ResumePreview()
	if !read.IsPlaying() {
		t.Fatal("the read must play on after the takeover")
	}
	waitFor(t, "the knob to reach the resumed line", func() bool { return near(read.volume(), 0.4) })
	e.PausePreview()
	e.DropHeld()
	waitFor(t, "the dropped line to close", func() bool {
		select {
		case <-read.stop:
			return true
		default:
			return false
		}
	})
	out.mu.Lock()
	main := out.players[0]
	out.mu.Unlock()
	if !main.IsPlaying() {
		t.Fatal("the broadcast underneath must keep playing")
	}
}

// PausePreview holds the line in flight without closing it; ResumePreview
// lets it play on; the broadcast is untouched throughout.
func TestPauseAndResumePreview(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	e.StartSource("test", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	waitFor(t, "main to play", func() bool { _, ok := out.mainVol(); return ok })
	e.PausePreview() // nothing in flight: inert
	if err := e.Preview(OutputRate, bytes.NewReader(make([]byte, 60*OutputRate))); err != nil {
		t.Fatal(err)
	}
	out.mu.Lock()
	line := out.players[len(out.players)-1]
	out.mu.Unlock()
	e.PausePreview()
	if line.IsPlaying() {
		t.Fatal("the line must be held")
	}
	time.Sleep(150 * time.Millisecond) // the watcher must not close a held line
	select {
	case <-line.stop:
		t.Fatal("a paused line was closed")
	default:
	}
	e.ResumePreview()
	if !line.IsPlaying() {
		t.Fatal("the line must play on")
	}
	out.mu.Lock()
	main := out.players[0]
	out.mu.Unlock()
	if !main.IsPlaying() {
		t.Fatal("the broadcast underneath must keep playing")
	}
}

// A preview alone drives the visualizer's tap (HUM LEAD UAT 2026-08-28:
// an event read is a preview, and the bars must follow it).
func TestPreviewFeedsTheVisualizerTap(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	clip := make([]byte, 4*OutputRate) // one second of a loud square wave, 16-bit LE stereo
	for i := 0; i+3 < len(clip); i += 4 {
		v := int16(20000)
		if (i/4/50)%2 == 1 {
			v = -20000
		}
		clip[i], clip[i+1], clip[i+2], clip[i+3] = byte(v), byte(v>>8), byte(v), byte(v>>8)
	}
	if err := e.Preview(OutputRate, bytes.NewReader(clip)); err != nil {
		t.Fatal(err)
	}
	dst := make([]float64, 1024)
	waitFor(t, "the tap to carry the preview", func() bool {
		if e.Samples(dst) == 0 {
			return false
		}
		for _, v := range dst {
			if v > 0.1 || v < -0.1 {
				return true
			}
		}
		return false
	})
}

// PreviewAside plays without touching the tap: the bars stay at rest.
func TestPreviewAsideLeavesTheTapAlone(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	clip := make([]byte, 4*OutputRate)
	for i := 0; i+3 < len(clip); i += 4 {
		clip[i], clip[i+1], clip[i+2], clip[i+3] = 0x20, 0x4e, 0x20, 0x4e // a loud constant
	}
	if err := e.PreviewAside(OutputRate, bytes.NewReader(clip)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	dst := make([]float64, 1024)
	if n := e.Samples(dst); n != 0 { // nothing at all reached the tap — not merely quiet frames (round 4, A-16d)
		t.Fatalf("an aside preview must not reach the visualizer: %d frames", n)
	}
}

// A voice audition is never the line in flight (REVIEW R5-B-03): with a read
// on air and a sample playing, a pause holds the READ, not the sample.
func TestAuditionIsNeverTheLineInFlight(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Preview(OutputRate, bytes.NewReader(make([]byte, 60*OutputRate))); err != nil {
		t.Fatal(err)
	}
	out.mu.Lock()
	read := out.players[len(out.players)-1]
	out.mu.Unlock()
	if err := e.Audition(OutputRate, bytes.NewReader(make([]byte, 60*OutputRate))); err != nil {
		t.Fatal(err)
	}
	out.mu.Lock()
	sample := out.players[len(out.players)-1]
	out.mu.Unlock()
	e.PausePreview()
	if read.IsPlaying() || !sample.IsPlaying() {
		t.Fatalf("the pause holds the read (playing=%v), not the sample (playing=%v)", read.IsPlaying(), sample.IsPlaying())
	}
	e.ResumePreview()
	if !read.IsPlaying() {
		t.Fatal("the read plays on")
	}
}

// A CHANGEOVER UNDER A DUCK STAYS DUCKED.
//
// The Watchlist advance replaces the main broadcast mid-alert: one location's
// cycle ends, the next one's starts. The new stream must come up AT THE DUCKED
// volume, because the alert reading over it has not finished — and the duck is
// engine state, not stream state, so a stream that started after the duck knows
// nothing about it unless the engine applies it on the way in.
//
// This is the mechanism the app's fix relies on: with the duck lift removed from
// the tune path, an automatic changeover can no longer un-duck, and this is why
// starting a fresh stream underneath does not either.
func TestANewStreamStartedUnderADuckPlaysDucked(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	e.StartSource("first", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	waitFor(t, "the first location to reach full volume", func() bool {
		v, ok := out.mainVol()
		return ok && near(v, 0.80)
	})

	e.Suppress() // an alert takes the air over a rendered cycle: it holds
	waitFor(t, "the first location to hold", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return len(out.players) > 0 && !out.players[0].IsPlaying()
	})

	// The cycle ends and the next favourite tunes — a whole new stream. It must
	// not come up over the alert either.
	e.StartSource("second", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	waitFor(t, "the next location to give way too", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return len(out.players) >= 2 && !out.players[len(out.players)-1].IsPlaying()
	})

	// And it comes up only when the alert is done, not when it started.
	e.Restore()
	waitFor(t, "the next location to come up after the alert", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return near(out.players[len(out.players)-1].volume(), 0.80)
	})
}

// HOLD STOPS THE BROADCAST WHERE IT IS, and a held stream is not a finished one
// (HUM LEAD, UAT 2026-08-30 — approved per mode: hold a synth cycle, duck a
// relay).
//
// The second half is the one that bites. A paused player stops reporting itself
// as playing, which is exactly what a drained one does — so without the guard
// the watch loop would call the cycle over and the watchlist would advance to
// the next location while the alert that held it was still reading. Which is the
// very collision holding was introduced to prevent.
func TestHoldStopsTheBroadcastAndIsNotAnEndedStream(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	e.StartSource("cycle", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	waitFor(t, "the cycle to play", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return len(out.players) > 0 && out.players[0].IsPlaying()
	})

	e.Suppress() // an alert takes the air over a synthesised report
	waitFor(t, "the cycle to stop where it is", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return !out.players[0].IsPlaying()
	})

	// Held is NOT ended: the engine still reports a live stream, so nothing
	// downstream advances the watchlist.
	time.Sleep(200 * time.Millisecond) // several watch ticks
	if st := e.Status(); st.State == Stopped {
		t.Errorf("a held cycle must not read as stopped, got %v", st.State)
	}

	// And it plays on from the same spot when the alert releases it.
	e.Restore()
	waitFor(t, "the cycle to play on", func() bool {
		out.mu.Lock()
		defer out.mu.Unlock()
		return out.players[0].IsPlaying()
	})
}

// A RENDERED REPORT STARTING UNDER AN ALERT IS HELD, NOT PLAYED.
//
// giveWay answers two things — how far to dip, and whether to hold outright —
// and the start path read only the first. A relay came up correctly dipped, but
// a rendered cycle came up PLAYING at full volume over the alert and was only
// held on the watch loop's next tick. The whole reason a rendered report holds
// rather than dips is that its words are lost under an alert; playing its first
// words over one loses exactly what the rule exists to protect.
func TestARenderedReportStartingUnderAnAlertIsHeldFromTheFirstSample(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "test", func(Status) {})
	if err != nil {
		t.Fatal(err)
	}
	e.Suppress() // an alert is already on the air
	e.StartSource("synth", 44100, func(context.Context) io.Reader { return endlessPCM{} })
	defer e.Halt()

	p := waitForPlayer(t, out)
	// Two watch ticks: long enough that a stream started and then paused would
	// still have recorded its Play().
	time.Sleep(120 * time.Millisecond)
	if n := p.plays.Load(); n != 0 {
		t.Errorf("a rendered report must not be played while an alert is on the air, Play() called %d time(s)", n)
	}

	e.Restore() // the alert ends
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && p.plays.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if p.plays.Load() == 0 {
		t.Error("…and it must play on once the alert is off the air — held, not dropped")
	}
}

// waitForPlayer blocks until the engine has opened its player.
func waitForPlayer(t *testing.T, out *recordingOutput) *recordingPlayer {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		out.mu.Lock()
		n := len(out.players)
		var p *recordingPlayer
		if n > 0 {
			p = out.players[0]
		}
		out.mu.Unlock()
		if p != nil {
			return p
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the engine never opened a player")
	return nil
}

// AN ALERT THAT ENDS BEFORE THE FIRST WATCH TICK STILL LETS THE REPORT PLAY.
//
// A report opened under an alert is not played, so the watch loop must know it
// is holding one. If the loop assumed every stream opened playing, an alert
// clearing inside its first 50 ms would leave nothing to switch: the loop sees
// no change to make, then reads a player that never started as a player that
// has drained, and drops the report entirely.
func TestAnAlertClearingBeforeTheFirstTickStillLetsTheReportPlay(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "test", func(Status) {})
	if err != nil {
		t.Fatal(err)
	}
	e.Suppress()
	e.StartSource("synth", 44100, func(context.Context) io.Reader { return endlessPCM{} })
	defer e.Halt()

	p := waitForPlayer(t, out)
	e.Restore() // the alert ends inside the watch loop's first tick

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && p.plays.Load() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if p.plays.Load() == 0 {
		t.Error("the report must play once the alert clears, however briefly it was held — not be dropped as though it had ended")
	}
}

// A RELAY OPENED UNDER AN ALERT COMES UP DIPPED, NOT AT FULL VOLUME.
//
// giveWay answers two things at once — how far to dip and whether to hold — and
// the open path has to honour BOTH. Holding was the half that was wrong before;
// this is the other half at the same moment. A relay that opened at the knob's
// volume would talk over the alert for a whole watch tick before the loop
// pulled it down, which is the same defect wearing the other hat.
//
// A real relay, because `live` is what decides dip-versus-hold and only the
// relay path sets it.
func TestARelayOpenedUnderAnAlertComesUpDipped(t *testing.T) {
	srv := mp3Server(t, nil)
	defer srv.Close()
	out := &recordingOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", func(Status) {})
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(80)
	e.Suppress() // an alert is already on the air
	e.Start([]string{srv.URL + "/live"}, "KEC49 Monterey")
	defer e.Halt()

	p := waitForPlayer(t, out)
	// The FIRST volume the player is ever given, before any watch tick could
	// correct it: dipped, and playing, because live radio does not wait.
	waitFor(t, "the relay to be playing", func() bool { return p.plays.Load() > 0 })
	if got, want := p.volumeAtPlay(), 0.80*alertDuck; !near(got, want) {
		t.Errorf("a relay opened under an alert must be dipped BEFORE it plays: volume at play %.4f, want %.4f", got, want)
	}

	// …and it returns to the knob when the alert ends, so the dip is not a
	// one-way trip. Without this the test would pass on an engine that simply
	// never raised the volume again.
	e.Restore()
	waitFor(t, "the relay to return to full volume", func() bool { return near(p.volume(), 0.80) })
}
