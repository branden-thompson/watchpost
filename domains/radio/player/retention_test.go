package player

import (
	"bytes"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// retainingOutput's players HOLD their PCM and never drain it, which is what a
// real device does for a minutes-long read: it consumes in real time, so the
// buffer stays reachable for as long as the engine keeps the player.
//
// fakeOutput CANNOT be used here, and finding that out is what made this probe
// trustworthy: it drains at memory speed, so an 8 MB clip is finished in
// microseconds and released whether or not the engine holds it. The control
// below caught exactly that — the first version of this test "passed" while
// proving nothing.
type retainingOutput struct{}

type retainingPlayer struct {
	pcm    io.Reader // held deliberately: this is the reference under test
	closed atomic.Bool
	once   sync.Once
}

func (retainingOutput) NewPlayer(pcm io.Reader) (Player, error) {
	return &retainingPlayer{pcm: pcm}, nil
}
func (p *retainingPlayer) Play()             {}
func (p *retainingPlayer) Pause()            {}
func (p *retainingPlayer) SetVolume(float64) {}
func (p *retainingPlayer) IsPlaying() bool   { return !p.closed.Load() }
func (p *retainingPlayer) Close() error      { p.once.Do(func() { p.closed.Store(true) }); return nil }

// bigClip is a PCM source that can say when it has been let go of. The
// finalizer fires only once nothing reachable holds it — the engine, the
// player, or the player's watcher goroutine.
type bigClip struct{ *bytes.Reader }

// freedAfter reports whether the clip became unreachable within d, polling a
// forced GC. It is a FAILURE bound, not a success bound: it returns as soon as
// the answer is yes, and only spends the full time when the answer is no.
//
// THE FLAG IS ATOMIC BECAUSE A FINALIZER RUNS ON THE COLLECTOR'S GOROUTINE.
// A plain bool here is a genuine data race — written by the finalizer, read by
// this loop — and `make race` caught it on the first full run after this test
// was written.
func freedAfter(freed *atomic.Bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		runtime.GC()
		if freed.Load() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return freed.Load()
}

// F-28 — A STOPPED READ LETS GO OF ITS AUDIO.
//
// Resident memory after a long event read was reported at 133–175 MB across UAT
// sessions. Nobody had established whether that is a leak or simply what a
// minutes-long read costs: 44.1 kHz stereo 16-bit is ~10 MB per minute of
// decoded PCM, so those numbers are plausible with nothing wrong. The row asked
// for a pprof session; the question is narrower than that and answerable here —
// whether a stopped read's buffer and its player are RELEASED — and as a test it
// keeps answering after this release, which a one-off session does not.
//
// [esc] is the path that matters: MVS-D-75 made closing the window stop the
// read, and Pause-before-Close is new in 0.14.0.
func TestAStoppedReadReleasesItsBufferAndPlayer(t *testing.T) {
	e, err := New(retainingOutput{}, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	var freed atomic.Bool
	// Scoped so the test's own reference dies with the block; only the engine
	// and its player can keep the clip alive past it.
	func() {
		clip := &bigClip{bytes.NewReader(make([]byte, 8<<20))} // 8 MB, ~48 s of audio
		runtime.SetFinalizer(clip, func(*bigClip) { freed.Store(true) })
		if err := e.PreviewAside(OutputRate, clip); err != nil {
			t.Fatal(err)
		}
	}()

	e.StopPreview() // what [esc] does

	if !freedAfter(&freed, 5*time.Second) {
		t.Error("a stopped read still holds its decoded audio and its player: the buffer is reachable " +
			"after StopPreview, so a minutes-long report stays resident until the process exits")
	}
}

// THE CONTROL, and without it the test above passes against a finalizer that
// fires for reasons of its own. A read still ON AIR must NOT be collected —
// if this reports freed, the instrument cannot tell retention from release and
// the result above means nothing.
func TestTheRetentionProbeCanSeeAClipThatIsStillHeld(t *testing.T) {
	e, err := New(retainingOutput{}, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer e.StopPreview()
	var freed atomic.Bool
	func() {
		clip := &bigClip{bytes.NewReader(make([]byte, 8<<20))}
		runtime.SetFinalizer(clip, func(*bigClip) { freed.Store(true) })
		if err := e.PreviewAside(OutputRate, clip); err != nil {
			t.Fatal(err)
		}
	}()
	if freedAfter(&freed, time.Second) {
		t.Error("the probe reported a clip freed while the engine was still playing it; " +
			"it cannot distinguish retention from release, so it proves nothing either way")
	}
}

// AND THE PAUSED READ — WHOSE RELEASE IS A DIFFERENT CALL, deliberately.
//
// A takeover suspends a `[w]` read by PAUSING it: the line leaves the flight
// slot and joins heldOrder. StopPreview closes what is IN FLIGHT and does not
// touch heldOrder — it only counts it for the debug line. So at this layer a
// held line survives a stop, and DropHeld is what lets it go.
//
// THAT IS THE CONTRACT, NOT A LEAK, and the distinction cost a wrong test to
// find: written as "a paused read that is stopped must not strand its audio"
// this failed, and it looked like a defect. It is not, because the app never
// leaves a held line behind — app/director.go calls dropHeld() whenever a
// suspended job is released or its context ends, on both paths. That guarantee
// lives in the ARBITER, so it is pinned there
// (TestASuspendedReadThatEndsDropsItsHeldAudio), and what is pinned here is the
// split itself. A future StopPreview that quietly started draining heldOrder
// would make the arbiter's call dead code; a future one that stopped would
// strand a report.
func TestAHeldLineSurvivesAStopAndIsReleasedByDropHeld(t *testing.T) {
	e, err := New(retainingOutput{}, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	var freed atomic.Bool
	func() {
		clip := &bigClip{bytes.NewReader(make([]byte, 8<<20))}
		runtime.SetFinalizer(clip, func(*bigClip) { freed.Store(true) })
		if err := e.PreviewAside(OutputRate, clip); err != nil {
			t.Fatal(err)
		}
	}()
	e.PausePreview() // a takeover suspends the read
	e.StopPreview()  // and the window closes

	if freedAfter(&freed, time.Second) {
		t.Error("a HELD line was released by StopPreview: that is DropHeld's job, and the arbiter's " +
			"dropHeld() call is now dead code that nothing will notice has stopped mattering")
	}

	e.DropHeld() // what the arbiter does when the suspended job ends
	if !freedAfter(&freed, 3*time.Second) {
		t.Error("DropHeld did not release the held line's audio: a suspended report stays resident " +
			"for the life of the process")
	}
}
