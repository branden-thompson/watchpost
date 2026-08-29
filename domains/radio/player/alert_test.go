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
	drained atomic.Bool
	vol     atomic.Uint64 // float64 bits
	stop    chan struct{}
	once    sync.Once
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

func (p *recordingPlayer) Play()           { p.started.Store(true) }
func (p *recordingPlayer) Pause()          { p.started.Store(false) }
func (p *recordingPlayer) IsPlaying() bool { return p.started.Load() && !p.drained.Load() }
func (p *recordingPlayer) SetVolume(v float64) {
	p.vol.Store(uint64(int64(v * 1e6)))
}
func (p *recordingPlayer) volume() float64 { return float64(int64(p.vol.Load())) / 1e6 }
func (p *recordingPlayer) Close() error    { p.once.Do(func() { close(p.stop) }); return nil }

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

	// The main broadcast: an endless source, playing.
	e.StartSource("test", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	atVol := func(want float64) func() bool {
		return func() bool { v, ok := out.mainVol(); return ok && near(v, want) }
	}
	waitFor(t, "main to reach full volume", atVol(0.80))

	// Duck: the main broadcast dips.
	e.Duck()
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
