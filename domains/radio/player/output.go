package player

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Output makes players from PCM readers (16-bit LE stereo at OutputRate).
// oto is the production implementation; tests use a fake that drains.
type Output interface {
	NewPlayer(pcm io.Reader) (Player, error)
}

// Player is the subset of oto.Player the engine uses.
type Player interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(v float64)
	Close() error
}

// OtoOutput is the real audio device. The oto context is created once, on
// the first player, and never closed (§10.4 / AI-5: it cannot be
// re-created; S1: ~20 MB stays unallocated until radio is used).
type OtoOutput struct {
	once sync.Once
	ctx  *oto.Context
	err  error
}

// NewPlayer implements Output.
func (o *OtoOutput) NewPlayer(pcm io.Reader) (Player, error) {
	o.once.Do(func() {
		var ready chan struct{}
		o.ctx, ready, o.err = oto.NewContext(&oto.NewContextOptions{
			SampleRate: OutputRate, ChannelCount: 2, Format: oto.FormatSignedInt16LE,
			BufferSize: 200 * time.Millisecond, // underruns are audible gaps; a fifth of a second of slack
		})
		if o.err == nil {
			<-ready
			// oto's unix driver reports "no PulseAudio, no ALSA" only here,
			// after ready, with NewContext's error nil (red-team 0.9.0 Linux
			// F2): unread, a headless box would show PLAYING over silence.
			o.err = o.ctx.Err()
		}
		if o.err != nil {
			o.err = fmt.Errorf("no audio output device — check the system sound settings (%v)", o.err) // F14: what / why / next
		}
	})
	if o.err != nil {
		return nil, o.err
	}
	return o.ctx.NewPlayer(pcm), nil
}
