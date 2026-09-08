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
	preview   Player          // the preview line in flight; nil between lines
	held      map[Player]bool // lines held by PausePreview: their watchers wait, later clips never displace them
	heldOrder []Player        // the held lines in the order held — ResumePreview takes the last
	userAgent string
	onStatus  func(Status)
	onSilence func(mount, name string) // MVS-D-76: the relay is up and silent
	trace     func(string)             // DR-23: the engine's line in the radio timeline
	clips     int                      // clips started, so the timeline can name which one
	budget    int                      // the clip watcher's ceiling in polls; 0 = defaultClipBudget

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	status Status
	volume float64
	// suppressed is "an alert is on air over the broadcast". HOW the broadcast
	// gives way is decided from live, every watch tick — see Suppress.
	suppressed bool
	live       bool // the current source is a relay (live radio), not a rendered cycle

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

// SilenceAfter is how long a relay may broadcast nothing before the listener is
// told. Five seconds, ratified by the HUM LEAD (MVS-D-76): a working NWR mount
// measured zero quiet windows in 102 seconds, so this is twenty times anything
// a live transmitter produces.
const SilenceAfter = 5 * time.Second

// OnSilence is called when the mount being played has been quiet for
// SilenceAfter. It runs on the AUDIO GOROUTINE, so it must not block — the app
// hands it straight to the UI as a message.
func (e *Engine) OnSilence(fn func(mount, name string)) {
	e.mu.Lock()
	e.onSilence = fn
	e.mu.Unlock()
}

// reportSilence tells whoever asked, once, off the read path.
func (e *Engine) reportSilence(mount, name string) {
	e.mu.Lock()
	fn := e.onSilence
	e.mu.Unlock()
	if fn != nil {
		go fn(mount, name) // never on the audio goroutine
	}
}

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
	return &Engine{out: out, userAgent: userAgent, onStatus: onStatus, volume: 0.55, status: Status{State: Stopped, Volume: 55}, tap: tap, held: map[Player]bool{}}, nil
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
	e.setLive(true) // a relay: live radio, which dips rather than pauses
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
	// THE SAMPLES ARE WATCHED, not just the connection. A relay that is up and
	// broadcasting nothing satisfies every other check (MVS-D-76); this is the
	// only place that can tell the difference, and it sits before the tap so it
	// sees exactly what the listener hears.
	pcm := newSilenceReader(newResampler(dec, dec.SampleRate()), OutputRate, SilenceAfter, func() { e.reportSilence(mount, label) })
	return e.playPCM(ctx, pcm, Status{State: Playing, Mount: mount, Name: label}, s.Title)
}

// ClipEnd is one clip's ending, reported once when its watcher stops.
//
// SPENT SAYS WHICH ENDING IT WAS: false when the player ran out of audio —
// the clip was heard to its end — and true when the watcher's own budget ran
// out first, which means the player was still claiming to play after ten
// minutes of AIR time and was closed from under it. The second is the shape
// FR-9 calls a fault: the read neither finished nor errored.
type ClipEnd struct {
	Spent bool
}

// Preview plays a short PCM clip (16-bit LE stereo at rate) on its own
// player, mixed over whatever is playing, without touching the engine's
// state (UAT 86: the voice chooser's sample). Returns when it has started.
func (e *Engine) Preview(rate int, pcm io.Reader) error {
	_, err := e.playClip(rate, pcm, true, true)
	return err
}

// PreviewWatched is Preview with the clip's ending reported on a channel
// (FR-9). The channel carries exactly one value and is then closed, so a
// caller that stops listening cannot block the watcher.
func (e *Engine) PreviewWatched(rate int, pcm io.Reader) (<-chan ClipEnd, error) {
	return e.playClip(rate, pcm, true, true)
}

// PreviewAsideWatched is PreviewAside with the same report.
func (e *Engine) PreviewAsideWatched(rate int, pcm io.Reader) (<-chan ClipEnd, error) {
	return e.playClip(rate, pcm, false, true)
}

// Audition plays a clip that is NEVER the line in flight — the voice
// chooser's sample: it drives the bars like a narration but a takeover's
// pause holds the narration's line, not it (REVIEW R5-B-03).
func (e *Engine) Audition(rate int, pcm io.Reader) error {
	_, err := e.playClip(rate, pcm, true, false)
	return err
}

// PreviewAside plays a clip beside the broadcast WITHOUT feeding the
// visualizer's tap — a takeover's tone and lines (the app's director
// decides which narrations the bars follow; HUM LEAD 2026-08-28).
func (e *Engine) PreviewAside(rate int, pcm io.Reader) error {
	_, err := e.playClip(rate, pcm, false, true)
	return err
}

// playClip is the one preview path: through the visualizer's tap like the
// broadcast when tapped (the bars sat flat during an event read before —
// the read plays here, not on the broadcast path), after the resampler so
// the tap reads OutputRate. The clip becomes the line in flight; a HELD
// line (PausePreview) is not displaced by it — a takeover's tone and lines
// play while the read waits (red-team round 4, A-01).
func (e *Engine) playClip(rate int, pcm io.Reader, tapped, inFlight bool) (<-chan ClipEnd, error) {
	src := io.Reader(newResampler(pcm, rate))
	if tapped {
		src = e.tap.Wrap(src)
	}
	p, err := e.out.NewPlayer(src)
	if err != nil {
		return nil, fmt.Errorf("audio output: %w", err)
	}
	// BUFFERED AND CLOSED, so the watcher never waits on a listener. A caller
	// that walks away — the sequence ended, the operator pressed esc — must not
	// pin the goroutine that closes the player.
	done := make(chan ClipEnd, 1)
	e.mu.Lock()
	p.SetVolume(e.volume)
	if inFlight {
		e.preview = p // the line in flight
	}
	e.clips++
	n := e.clips
	e.mu.Unlock()
	e.debugf("player:clip:start n=%d tapped=%v inFlight=%v", n, tapped, inFlight)
	p.Play()
	go func() {
		// Bounded per P10-02: 10 min of PLAY is the ceiling (an event read
		// speaks a whole record — minutes, not seconds); a held line does
		// not spend its budget.
		//
		// THE BUDGET IS ALSO FR-9's BOUND, and it was already the right shape:
		// held-excluded elapsed time, on the read's own player, never on the
		// live stream. What was missing was anyone being told how it ended.
		spent := true
		budget := e.clipBudget()
		i := 0
		for i < budget {
			e.mu.Lock()
			held := e.held[p]
			if !held {
				p.SetVolume(e.volume) // the knob follows a minutes-long read (round 4, A-07)
			}
			e.mu.Unlock()
			if !held && !p.IsPlaying() {
				spent = false // the player ran out of audio: the clip was heard
				break
			}
			time.Sleep(50 * time.Millisecond)
			if !held {
				i++
			}
		}
		e.mu.Lock()
		if e.preview == p {
			e.preview = nil
		}
		delete(e.held, p)
		e.mu.Unlock()
		e.debugf("player:clip:end n=%d spent=%v", n, spent)
		_ = p.Close()
		done <- ClipEnd{Spent: spent}
		close(done)
	}()
	return done, nil
}

// defaultClipBudget is the watcher's ceiling in 50 ms polls: ten minutes of AIR
// time. An event read speaks a whole record — minutes, not seconds — and a held
// line does not spend it.
const defaultClipBudget = 12000

// clipBudget is that ceiling, OVERRIDABLE so the fault path can be tested.
//
// A BOUND NOBODY CAN REACH IN A TEST IS A BOUND NOBODY HAS WATCHED FAIL. Ten
// minutes of air time is right for a listener and impossible for a suite, so
// the number lives in a field the package's own tests can shorten — which is
// the difference between "the fault fires" being a claim and a measurement.
func (e *Engine) clipBudget() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.budget > 0 {
		return e.budget
	}
	return defaultClipBudget
}

// PausePreview holds the line in flight (a lower-priority narration while a
// takeover speaks — the app's director): it leaves the flight slot, so the
// takeover's own clips never displace it, and its watcher waits. ResumePreview
// puts the most recently held line back in flight and lets it go on from
// where it stopped; DropHeld closes every held line without playing it (a
// suspended narration that ended). All inert with nothing held.
func (e *Engine) PausePreview() {
	e.mu.Lock()
	p := e.preview
	if p != nil {
		e.held[p] = true
		e.heldOrder = append(e.heldOrder, p)
		e.preview = nil
	}
	e.mu.Unlock()
	if p != nil {
		p.Pause()
	}
}

func (e *Engine) ResumePreview() {
	e.mu.Lock()
	var p Player
	if n := len(e.heldOrder); n > 0 {
		p = e.heldOrder[n-1]
		e.heldOrder = e.heldOrder[:n-1]
		delete(e.held, p)
		e.preview = p
	}
	e.mu.Unlock()
	if p != nil {
		p.Play()
	}
}

func (e *Engine) DropHeld() {
	e.mu.Lock()
	held := e.heldOrder
	e.heldOrder = nil
	for p := range e.held {
		delete(e.held, p)
	}
	e.mu.Unlock()
	for _, p := range held {
		_ = p.Close() // its watcher sees !IsPlaying and finishes
	}
}

// alertDuck is how far the main broadcast dips while an alert sounds over it
// (to 15 % — audibly under the alert, still present so the broadcast is not
// mistaken for stopped). HUM LEAD 2026-08-27: duck, not interrupt.
//
// 0.25 until MVS-D-70 (UAT 2026-09-03): heard on a live relay, a quarter was
// "still loud enough to be distracting". The floor of the range and the reason
// it is not zero are both in TestTheDipDepthIsPinned, which is the one place
// this number is allowed to be argued about.
const alertDuck = 0.15

// Suppress puts an alert on the air over the broadcast: the watch loop gives
// way within 50 ms. Inert when nothing is playing. Pair every Suppress with a
// Restore.
//
// HOW the broadcast gives way is not decided here, and that is the point. A
// relay DIPS: it is live radio, and a paused relay resumes into audio that is
// minutes stale — a listener would hear a warning that had already expired. A
// rendered cycle HOLDS: it has nowhere to be, so dipping loses the words under
// the alert for good where holding means the whole report is heard afterwards.
//
// Deciding that at the moment the alert arrives fixes an answer that the source
// can outlive: a relay that falls back to synth mid-alert would keep playing
// dipped, and a stream started while an alert was already on air would take
// whichever treatment the previous source earned. The watch loop re-reads the
// source kind every tick, so the answer follows the audio.
func (e *Engine) Suppress() {
	e.mu.Lock()
	e.suppressed = true
	e.mu.Unlock()
}

// Restore takes the alert off the air — the broadcast returns to the knob, or
// plays on from where it was held, within 50 ms.
func (e *Engine) Restore() {
	e.mu.Lock()
	e.suppressed = false
	e.mu.Unlock()
}

// setLive records what kind of source is playing, for Suppress to read.
func (e *Engine) setLive(relay bool) {
	e.mu.Lock()
	e.live = relay
	e.mu.Unlock()
}

// giveWay is how the current source yields to an alert: the volume scale to
// apply, and whether to pause outright. Caller holds mu.
func (e *Engine) giveWayLocked() (scale float64, pause bool) {
	switch {
	case !e.suppressed:
		return 1, false
	case e.live:
		return alertDuck, false // dip: live radio plays on under the alert
	}
	return 1, true // hold: a rendered report waits and is heard in full afterwards
}

// StartSource plays a generic PCM source (16-bit LE stereo at rate) until
// it ends or Halt — the synthesized broadcast (B4 step 2). open is called
// on the engine goroutine with a context that Halt cancels.
func (e *Engine) StartSource(name string, rate int, open func(ctx context.Context) io.Reader) {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.halt()
	e.setLive(false) // a rendered cycle: it waits rather than plays on under an alert
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
	scale, hold := e.giveWayLocked() // a stream starting under an alert comes up already giving way
	p.SetVolume(e.volume * scale)
	e.mu.Unlock()
	if !hold { // a rendered report waits for the alert rather than opening under it
		p.Play()
	}
	playing.Title = title()
	e.set(playing)
	return e.watch(ctx, p, title, hold)
}

// watch keeps the volume applied and reports title changes until the
// player drains (stream ended/stalled) or the context ends.
//
// held is how the stream was opened, and it is NOT defensive. Pass a constant
// false and an alert clearing inside the first tick leaves want == held, so no
// Play() ever fires; the drain check below then reads a player that never
// started as one that has finished, and the whole report is dropped without a
// word. TestAnAlertClearingBeforeTheFirstTickStillLetsTheReportPlay is what
// holds that shut.
func (e *Engine) watch(ctx context.Context, p Player, title func() string, held bool) error {
	tick := time.NewTicker(50 * time.Millisecond) // stop latency ≤ 50 ms
	defer tick.Stop()
	for range tick.C { // bounded by the ticker; exits on ctx or stream end
		if ctx.Err() != nil {
			p.Pause() // silence at once; Close follows (UAT 81: stop must feel instant)
			return ctx.Err()
		}
		e.mu.Lock()
		scale, want := e.giveWayLocked() // re-read every tick: the answer follows the audio
		p.SetVolume(e.volume * scale)
		st := e.status
		e.mu.Unlock()
		if want != held {
			if want {
				p.Pause()
			} else {
				p.Play()
			}
			held = want
		}
		// A HELD player is not a finished one. It stops reporting itself as
		// playing, which is exactly what a drained stream does, and reading that
		// as the end of the cycle would advance the watchlist while the alert
		// that held it was still reading.
		if !held && !p.IsPlaying() {
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

// StopPreview ends the line currently in flight.
//
// THE ONE THING THE ENGINE COULD NOT DO. It could pause a line (PausePreview),
// resume one (ResumePreview) and close the HELD ones (DropHeld) — but nothing
// stopped the line that was actually sounding. So cancelling a read ended its
// SEQUENCE while its audio played on: the arbiter released the air, resumed
// whatever the read had suspended, and the listener heard both at once
// (MVS-D-75, UAT 2026-09-05).
//
// Closing a player that has already drained is a no-op, so a sequence that
// finished normally needs no special case — which is what lets one call sit on
// the one path a job leaves the air by.
func (e *Engine) StopPreview() {
	e.mu.Lock()
	p, held := e.preview, len(e.heldOrder)
	e.preview = nil
	e.mu.Unlock()
	e.debugf("player:stopPreview found=%v held=%d", p != nil, held)
	if p != nil {
		// PAUSE BEFORE CLOSE. Close releases the player; it is Pause that
		// guarantees the device stops emitting what is already buffered. A
		// close alone left a read audible after it had been stopped
		// (UAT 2026-09-05), and the buffered tail of a minutes-long read is not
		// a tail — it is the rest of the report.
		p.Pause()
		_ = p.Close() // its watcher sees !IsPlaying and finishes
	}
}

// debugf is the engine's own line in the radio timeline. It writes only when a
// hook is wired, so the player stays free of the app's logging.
func (e *Engine) debugf(format string, args ...any) {
	if e.trace == nil {
		return
	}
	e.trace(fmt.Sprintf(format, args...))
}

// Trace wires the engine's debug line to the app's log (DR-23).
func (e *Engine) Trace(fn func(string)) {
	e.mu.Lock()
	e.trace = fn
	e.mu.Unlock()
}
