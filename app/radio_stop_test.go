package app

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
)

// heldOutput hands out players that count how many times they were told to
// play, which is what distinguishes a stream held under an alert from one that
// came up over it.
type heldOutput struct {
	mu      sync.Mutex
	players []*heldPlayer
}

type heldPlayer struct {
	plays   atomic.Int32
	started atomic.Bool
	stop    chan struct{}
	once    sync.Once
}

func (o *heldOutput) NewPlayer(io.Reader) (player.Player, error) {
	p := &heldPlayer{stop: make(chan struct{})}
	o.mu.Lock()
	o.players = append(o.players, p)
	o.mu.Unlock()
	return p, nil
}

func (p *heldPlayer) Play()             { p.plays.Add(1); p.started.Store(true) }
func (p *heldPlayer) Pause()            { p.started.Store(false) }
func (p *heldPlayer) IsPlaying() bool   { return p.started.Load() }
func (p *heldPlayer) SetVolume(float64) {}
func (p *heldPlayer) Close() error      { p.once.Do(func() { close(p.stop) }); return nil }
func (o *heldOutput) latest() *heldPlayer {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.players) == 0 {
		return nil
	}
	return o.players[len(o.players)-1]
}

type endlessSilence struct{}

func (endlessSilence) Read(b []byte) (int, error) { return len(b), nil }

// A USER'S STOP DOES NOT TAKE AN ALERT OFF THE AIR.
//
// Stopping the radio lifted the alert suppression, which the takeover owns and
// pairs with its own restore. Nothing was playing at that moment, so it looked
// harmless — but the alert was still being read, and the next thing the
// listener started came up at full volume over the top of it. Whether an alert
// is on the air is not the stop button's to answer.
func TestAUserStopDoesNotTakeAnAlertOffTheAir(t *testing.T) {
	out := &heldOutput{}
	eng, err := player.New(out, "test", func(player.Status) {})
	if err != nil {
		t.Fatal(err)
	}
	d := &radioDeck{engine: eng}
	eng.Suppress() // a takeover is reading an alert
	defer eng.Restore()

	d.Stop() // the listener presses stop while the alert is still being read

	eng.StartSource("synth", 44100, func(context.Context) io.Reader { return endlessSilence{} })
	defer eng.Halt()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && out.latest() == nil {
		time.Sleep(5 * time.Millisecond)
	}
	p := out.latest()
	if p == nil {
		t.Fatal("the engine never opened a player")
	}
	time.Sleep(120 * time.Millisecond) // two watch ticks
	if n := p.plays.Load(); n != 0 {
		t.Errorf("what the listener starts next must still give way to the alert, Play() called %d time(s)", n)
	}

	// AND THE POSITIVE HALF. Without it this passes on an engine that never
	// plays anything at all — a held stream and a broken one look identical
	// from the outside, and only one of them is the behaviour being asserted.
	eng.Restore()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && p.plays.Load() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if p.plays.Load() == 0 {
		t.Error("…and once the alert is off the air the held stream must play")
	}
}
