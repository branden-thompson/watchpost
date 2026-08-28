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
