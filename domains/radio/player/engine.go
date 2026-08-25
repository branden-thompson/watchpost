package player

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/go-mp3"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// State is the engine's externally visible condition (AI-5 §3).
type State string

// errEnded is a stream that ran out — a relay drop (retry) or a finished
// synthesized broadcast (completion).
var errEnded = errors.New("stream ended")

// EndedTitle is the Stopped status Title when a source played to its end
// (UAT 83: Repeat off; UAT 93: the Watchlist advances on it).
const EndedTitle = "broadcast complete"

// ErrMountRefused is a relay answering 403 or 404: the relays' terms say
// honour it — no retries on that mount (red-team 0.9.0; §10.4 ToS).
var ErrMountRefused = errors.New("mount refused")

// Engine states.
const (
	Stopped      State = "stopped"
	Connecting   State = "connecting"
	Playing      State = "playing"
	Reconnecting State = "reconnecting"
	Failed       State = "failed" // every mount exhausted; the caller degrades (Synth/text)
)

// Status is what the UI sees.
type Status struct {
	State  State
	Mount  string // URL being played / attempted
	Name   string // icy-name from the relay
	Title  string // in-band StreamTitle, if any
	Volume int    // 0-100
	Err    string // last error text (redacted URLs only — mounts carry no secrets)
}

// Engine plays one station at a time: tries its mounts in order, reconnects
// with backoff on stalls, and reports status through a callback.
type Engine struct {
	out       Output
	userAgent string
	onStatus  func(Status)

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	status Status
	volume float64

	// startMu serializes Start / StartSource / Halt end to end (red-team
	// 0.9.0 C-1): halting the old stream and installing the new cancel are
	// one step, so no engine goroutine can ever be left without an owner.
	startMu sync.Mutex

	tap *Tap // the visualizer's view of whatever plays (UAT 92)
}

// tapFrames is the tap's ring: one analysis window plus slack (~70 ms).
const tapFrames = 3072

// Backoff bounds for reconnects (§10.4): 1 s → 30 s with ±50 % jitter,
// maxAttempts per mount before moving to the next.
const (
	backoffBase = time.Second
	backoffMax  = 30 * time.Second
	maxAttempts = 3
	preroll     = 12 * 1024 // compressed bytes buffered before decoding starts (~3 s at 32 kbps)
)

// New builds an engine. onStatus may be nil.
func New(out Output, userAgent string, onStatus func(Status)) (*Engine, error) {
	if err := invariant.Check(out != nil && userAgent != "", "player: Output and user agent are required"); err != nil {
		return nil, err
	}
	if onStatus == nil {
		onStatus = func(Status) {}
	}
	tap, err := NewTap(tapFrames)
	if err != nil {
		return nil, err
	}
	return &Engine{out: out, userAgent: userAgent, onStatus: onStatus, volume: 0.55, status: Status{State: Stopped, Volume: 55}, tap: tap}, nil
}

// Samples fills dst with the latest mono samples (±1, oldest first) of
// whatever is playing — relay or synthesized — and returns the count; 0
// when stopped (UAT 92: the visualizer's feed).
func (e *Engine) Samples(dst []float64) int { return e.tap.Samples(dst) }

// Start plays the first mount that works, in order. Any current playback
// halts first. Returns at once; progress arrives via onStatus.
func (e *Engine) Start(mounts []string, name string) {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.halt()
	if len(mounts) == 0 {
		e.set(Status{State: Failed, Err: "no relay carries this transmitter", Volume: e.pct()})
		return
	}
	ctx, done := e.arm()
	go func() {
		defer close(done)
		e.run(ctx, mounts, name)
	}()
}

// arm installs a fresh cancel/done pair for a new engine goroutine (under
// startMu). The goroutine closes its own done — captured here, because
// halt nils the field before the goroutine has exited.
func (e *Engine) arm() (context.Context, chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	e.mu.Lock()
	e.cancel, e.done = cancel, done
	e.mu.Unlock()
	return ctx, done
}

// Fail reports a failure that happened before any stream could start —
// no voice, no source — under the caller's own words (red-team 0.9.0 F2:
// routing it through Start(nil) replaced the reason with "no relay
// carries this transmitter"). Any current playback halts first.
func (e *Engine) Fail(reason string) {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.halt()
	e.set(Status{State: Failed, Err: reason})
}

// Halt ends playback and waits for the engine goroutine.
func (e *Engine) Halt() {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.halt()
}

// halt is Halt under startMu (Start and StartSource call it before arming).
func (e *Engine) halt() {
	e.mu.Lock()
	cancel, done := e.cancel, e.done
	e.cancel, e.done = nil, nil
	e.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	e.tap.Reset() // the bars decay to rest
	e.set(Status{State: Stopped, Volume: e.pct()})
}

// Volume sets 0-100; applies live.
func (e *Engine) Volume(pct int) {
	pct = max(0, min(100, pct))
	e.mu.Lock()
	e.volume = float64(pct) / 100
	e.status.Volume = pct
	st := e.status
	e.mu.Unlock()
	e.onStatus(st)
}

// Status is the current status.
func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

func (e *Engine) pct() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return int(e.volume*100 + 0.5)
}

func (e *Engine) set(st Status) {
	e.mu.Lock()
	st.Volume = int(e.volume*100 + 0.5)
	e.status = st
	e.mu.Unlock()
	e.onStatus(st)
}

// run walks the mounts: each gets maxAttempts connections with backoff;
// a stream that stalls or ends reconnects to the same mount first.
func (e *Engine) run(ctx context.Context, mounts []string, name string) {
	var lastErr error
	for _, mount := range mounts {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if ctx.Err() != nil {
				return
			}
			state := Connecting
			if attempt > 0 {
				state = Reconnecting
			}
			e.set(Status{State: state, Mount: mount, Name: name, Err: errText(lastErr)})
			err := e.playOnce(ctx, mount, name)
			if ctx.Err() != nil {
				return // stopped on purpose
			}
			lastErr = err
			if errors.Is(err, ErrMountRefused) {
				break // 403/404: the relay said no — next mount, no backoff retries
			}
			if !e.sleep(ctx, backoff(attempt)) {
				return
			}
		}
	}
	e.set(Status{State: Failed, Err: errText(lastErr)})
}

// playOnce plays one relay connection until it ends or fails.
func (e *Engine) playOnce(ctx context.Context, mount, name string) error {
	s, err := Open(ctx, e.userAgent, mount)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }() // a closed stream's error carries nothing recoverable
	dec, err := mp3.NewDecoder(&prerollReader{r: s, want: preroll})
	if err != nil {
		return fmt.Errorf("decoder: %w", err)
	}
	label := s.Name
	if label == "" {
		label = name
	}
	return e.playPCM(ctx, newResampler(dec, dec.SampleRate()), Status{State: Playing, Mount: mount, Name: label}, s.Title)
}

// Preview plays a short PCM clip (16-bit LE stereo at rate) on its own
// player, mixed over whatever is playing, without touching the engine's
// state (UAT 86: the voice chooser's sample). Returns when it has started.
func (e *Engine) Preview(rate int, pcm io.Reader) error {
	p, err := e.out.NewPlayer(newResampler(pcm, rate))
	if err != nil {
		return fmt.Errorf("audio output: %w", err)
	}
	e.mu.Lock()
	p.SetVolume(e.volume)
	e.mu.Unlock()
	p.Play()
	go func() {
		// Bounded per P10-02: a preview is seconds long; 30 s is the ceiling.
		for i := 0; i < 600 && p.IsPlaying(); i++ {
			time.Sleep(50 * time.Millisecond)
		}
		_ = p.Close()
	}()
	return nil
}

// StartSource plays a generic PCM source (16-bit LE stereo at rate) until
// it ends or Halt — the synthesized broadcast (B4 step 2). open is called
// on the engine goroutine with a context that Halt cancels.
func (e *Engine) StartSource(name string, rate int, open func(ctx context.Context) io.Reader) {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.halt()
	ctx, done := e.arm()
	go func() {
		defer close(done)
		e.set(Status{State: Connecting, Name: name})
		err := e.playPCM(ctx, newResampler(open(ctx), rate), Status{State: Playing, Name: name}, func() string { return "" })
		switch {
		case ctx.Err() != nil:
		case errors.Is(err, errEnded):
			e.set(Status{State: Stopped, Name: name, Title: EndedTitle}) // UAT 83: Repeat off
		default:
			e.set(Status{State: Failed, Name: name, Err: errText(err)})
		}
	}()
}

// playPCM feeds a PCM reader to a new player and watches it.
func (e *Engine) playPCM(ctx context.Context, pcm io.Reader, playing Status, title func() string) error {
	e.tap.Reset()
	p, err := e.out.NewPlayer(e.tap.Wrap(pcm)) // before the player's volume: the bars show the broadcast, not the knob
	if err != nil {
		return fmt.Errorf("audio output: %w", err)
	}
	defer func() { _ = p.Close() }()
	e.mu.Lock()
	p.SetVolume(e.volume)
	e.mu.Unlock()
	p.Play()
	playing.Title = title()
	e.set(playing)
	return e.watch(ctx, p, title)
}

// watch keeps the volume applied and reports title changes until the
// player drains (stream ended/stalled) or the context ends.
func (e *Engine) watch(ctx context.Context, p Player, title func() string) error {
	tick := time.NewTicker(50 * time.Millisecond) // stop latency ≤ 50 ms
	defer tick.Stop()
	for range tick.C { // bounded by the ticker; exits on ctx or stream end
		if ctx.Err() != nil {
			p.Pause() // silence at once; Close follows (UAT 81: stop must feel instant)
			return ctx.Err()
		}
		e.mu.Lock()
		p.SetVolume(e.volume)
		st := e.status
		e.mu.Unlock()
		if !p.IsPlaying() {
			return errEnded
		}
		if t := title(); t != st.Title {
			st.Title = t
			e.set(st)
		}
	}
	return nil
}

func (e *Engine) sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func backoff(attempt int) time.Duration {
	d := min(backoffBase<<min(attempt, 5), backoffMax)
	return d + time.Duration(rand.Int63n(int64(d))) - d/2
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// prerollReader delays the first byte until want bytes are buffered, so
// the decoder never starves right after connecting (AI-5 §3: 2–3 s at
// 32 kbps), then passes reads straight through.
type prerollReader struct {
	r    io.Reader
	want int
	buf  []byte
	done bool
}

func (p *prerollReader) Read(b []byte) (int, error) {
	if !p.done {
		// Bounded per P10-02: at most want/64 reads (a relay sends ≥64 B per read).
		for i := 0; i < p.want/64+1 && len(p.buf) < p.want; i++ {
			chunk := make([]byte, 4096)
			n, err := p.r.Read(chunk)
			p.buf = append(p.buf, chunk[:n]...)
			if err != nil {
				break
			}
		}
		p.done = true
	}
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}
	return p.r.Read(b)
}
