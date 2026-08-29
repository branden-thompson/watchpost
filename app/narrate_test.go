package app

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scriptVoice records the narrator's calls in order.
type scriptVoice struct {
	mu    sync.Mutex
	calls []string
	dur   time.Duration
}

func (v *scriptVoice) duck()               { v.rec("duck") }
func (v *scriptVoice) tone() time.Duration { v.rec("tone"); return v.dur }
func (v *scriptVoice) render(_ context.Context, t string) (clip, bool) {
	return clip{text: t, dur: v.dur}, true
}
func (v *scriptVoice) play(c clip) {
	if c.viz {
		v.rec("speak:" + c.text)
		return
	}
	v.rec("aside:" + c.text) // a takeover's line: the visualizer does not follow it
}
func (v *scriptVoice) pause()   { v.rec("pause") }
func (v *scriptVoice) resume()  { v.rec("resume") }
func (v *scriptVoice) discard() { v.rec("discard") }
func (v *scriptVoice) restore() { v.rec("restore") }
func (v *scriptVoice) rec(s string) {
	v.mu.Lock()
	v.calls = append(v.calls, s)
	v.mu.Unlock()
}
func (v *scriptVoice) got() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return strings.Join(v.calls, ",")
}

// A single sequence: duck once, its lines, restore once.
func TestNarratorDucksOnceAndRestoresOnce(t *testing.T) {
	v := &scriptVoice{dur: time.Second}
	n := newNarrator(v)
	n.sleep = func(context.Context, time.Duration) bool { return true }
	ok := n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) {
		s.hold(s.attention())
		s.hold(s.line("a"))
		s.hold(s.line("b"))
	})
	if !ok || v.got() != "duck,tone,aside:a,aside:b,restore" {
		t.Fatalf("ok=%v calls=%s", ok, v.got())
	}
}

// Breaking takes the air from a read: the read's line pauses, its hold
// stops counting, breaking runs, the read resumes where it stopped and
// finishes; the broadcast is restored once at the very end (HUM LEAD UAT:
// never a collision, and the read carries on).
func TestNarratorBreakingSuspendsAndResumesARead(t *testing.T) {
	v := &scriptVoice{}
	n := newNarrator(v)
	n.sleep = sleepCtx
	readStarted, readDone := make(chan struct{}), make(chan bool, 1)
	go n.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) {
		s.line("read-line")
		close(readStarted)
		readDone <- s.hold(300 * time.Millisecond) // air time: the suspension does not count
	})
	<-readStarted
	inBreaking := time.Now()
	ok := n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) {
		s.line("breaking-line")
		s.hold(500 * time.Millisecond) // longer than the read's whole hold
	})
	if !ok || v.got() != "duck,speak:read-line,pause,aside:breaking-line,resume" {
		t.Fatalf("at the takeover's end the read resumes, nothing restored yet: ok=%v calls=%s", ok, v.got())
	}
	select {
	case finished := <-readDone:
		if !finished {
			t.Fatal("the read must finish, not be cut")
		}
		if since := time.Since(inBreaking); since < 700*time.Millisecond { // 500 ms takeover + the read's remaining ≥ 200 ms
			t.Fatalf("the read's hold must not count the suspension: finished %v after the takeover began", since)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the read never finished")
	}
	waitUntil(t, "the read's release", func() bool { return strings.HasSuffix(v.got(), "restore") })
	if got := v.got(); got != "duck,speak:read-line,pause,aside:breaking-line,resume,restore" {
		t.Fatalf("one restore, at the very end: %s", got)
	}
}

// A line still rendering when the takeover lands never starts under it: it
// waits for the air (the UAT collision).
func TestNarratorALineRenderedUnderSuspensionWaitsForTheAir(t *testing.T) {
	v := &scriptVoice{}
	n := newNarrator(v)
	n.sleep = sleepCtx
	rendering, release := make(chan struct{}), make(chan struct{})
	slow := &slowVoice{scriptVoice: v, rendering: rendering, release: release}
	n.v = slow
	go n.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) { s.line("read-line") })
	<-rendering // the read is rendering its (long) script
	breakingDone := make(chan struct{})
	go func() {
		n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) {
			s.line("breaking-line")
			close(release) // the read's render finishes while the takeover is on air…
			s.hold(200 * time.Millisecond)
		})
		close(breakingDone)
	}()
	<-breakingDone
	waitUntil(t, "the read's line and release", func() bool { return strings.HasSuffix(v.got(), "restore") })
	if got := v.got(); got != "duck,pause,aside:breaking-line,resume,speak:read-line,restore" {
		t.Fatalf("…and its line plays only after: %s", got)
	}
}

// slowVoice renders its FIRST line on a signal, so a test can hold that
// line in the renderer; later lines render at once.
type slowVoice struct {
	*scriptVoice
	rendering, release chan struct{}
	once               sync.Once
}

func (v *slowVoice) render(_ context.Context, t string) (clip, bool) {
	first := false
	v.once.Do(func() { first = true })
	if first {
		close(v.rendering)
		<-v.release
	}
	return clip{text: t, dur: v.dur}, true
}

// A read submitted during a takeover waits for it; equal classes go in order.
func TestNarratorQueuesBehindAHigherClassAndInOrder(t *testing.T) {
	v := &scriptVoice{}
	n := newNarrator(v)
	n.sleep = sleepCtx
	inBreaking := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) {
			s.line("breaking")
			close(inBreaking)
			<-release
		})
	}()
	<-inBreaking
	var order []string
	var omu sync.Mutex
	for i, name := range []string{"read-1", "read-2"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			n.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) {
				omu.Lock()
				order = append(order, name)
				omu.Unlock()
				s.line(name)
			})
		}(name)
		waitUntil(t, "arrival order", func() bool { n.mu.Lock(); defer n.mu.Unlock(); return len(n.waiting) == i+1 }) // queued before the next arrives
	}
	if got := v.got(); got != "duck,aside:breaking" {
		t.Fatalf("reads must wait for the takeover: %s", got)
	}
	close(release)
	wg.Wait()
	if got := v.got(); got != "duck,aside:breaking,speak:read-1,speak:read-2,restore" {
		t.Fatalf("one duck for the whole run, reads in arrival order, one restore: %s", got)
	}
}

// Muted or silent sequences run their visuals and never touch the broadcast.
func TestNarratorMutedAndSilentNeverDuck(t *testing.T) {
	v := &scriptVoice{}
	n := newNarrator(v)
	n.sleep = func(context.Context, time.Duration) bool { return true }
	ran := 0
	n.Run(context.Background(), narrateBreaking, false, func(ctx context.Context, s *speaker) { ran++; s.hold(s.line("x")) })
	if v.got() != "" || ran != 1 {
		t.Fatalf("muted: calls=%q ran=%d", v.got(), ran)
	}
	silent := newNarrator(nil)
	silent.sleep = func(context.Context, time.Duration) bool { return true }
	if !silent.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) { ran++; s.hold(s.line("x")) }) || ran != 2 {
		t.Fatal("silent: the visual sequence still runs")
	}
	var nilN *narrator
	if !nilN.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) { ran++ }) || ran != 3 {
		t.Fatal("a nil narrator runs the sequence")
	}
}

// The visualizer follows every narration but a takeover — one owner.
func TestVizFollowsEveryNarrationButATakeover(t *testing.T) {
	if vizFor(narrateBreaking) || !vizFor(narrateRead) {
		t.Fatal("vizFor: a read drives the bars, a takeover plays aside")
	}
}

// A read cancelled while SUSPENDED under a takeover leaves the stack and
// its held line is discarded; the takeover keeps the air, and the next read
// is admitted after it (red-team round 4, A-02 — the wedge).
func TestReadCancelledWhileSuspendedDoesNotWedgeTheNarrator(t *testing.T) {
	v := &scriptVoice{dur: 50 * time.Millisecond}
	n := newNarrator(v)
	n.sleep = sleepCtx
	ctx, cancel := context.WithCancel(context.Background())
	readDone := make(chan bool, 1)
	suspended := make(chan struct{})
	go func() {
		readDone <- n.Run(ctx, narrateRead, true, func(ctx context.Context, s *speaker) {
			s.line("read")
			close(suspended)
			s.hold(5 * time.Second) // suspended by the takeover during this hold
		})
	}()
	<-suspended
	tookOver := make(chan struct{})
	release := make(chan struct{})
	go func() {
		n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) {
			s.line("breaking")
			close(tookOver)
			<-release
		})
	}()
	<-tookOver
	cancel() // the read ends while suspended
	if ok := <-readDone; ok {
		t.Fatal("a cancelled read reports false")
	}
	close(release)
	admitted := make(chan bool, 1)
	go func() {
		admitted <- n.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) { s.line("next") })
	}()
	select {
	case ok := <-admitted:
		if !ok {
			t.Fatal("the next read must run")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("the next read was never admitted (wedged): %s", v.got())
	}
	// The next read is admitted either after the takeover's restore (its own
	// duck) or before it (the duck still standing): both end on its line and
	// one restore; the cancelled read never resumes.
	got := v.got()
	if !strings.Contains(got, ",pause,aside:breaking,discard,") || strings.Contains(got, "resume") || !strings.HasSuffix(got, "speak:next,restore") {
		t.Fatalf("voice: %s", got)
	}
}

// A line never starts under a takeover: the start happens under the
// arbiter's lock, so a takeover admitted meanwhile finds it in flight and
// pauses it (red-team round 4, A-04).
func TestALineNeverStartsUnderATakeover(t *testing.T) {
	var mu sync.Mutex
	collisions := 0
	n := newNarrator(nil)
	v := &probeVoice{onPlay: func(c clip) {
		if c.viz && n.onAir != nil && n.onAir.class == narrateBreaking { // called under n.mu
			mu.Lock()
			collisions++
			mu.Unlock()
		}
	}}
	n.v = v
	n.sleep = sleepCtx
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			n.Run(context.Background(), narrateRead, true, func(ctx context.Context, s *speaker) { s.line("r") })
		}
		close(stop)
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) { s.line("b"); s.hold(time.Millisecond) })
			}
		}
	}()
	wg.Wait()
	if collisions != 0 {
		t.Fatalf("%d read lines started under a takeover", collisions)
	}
}

// probeVoice is a narrationVoice whose play hook sees the narrator's state.
type probeVoice struct{ onPlay func(clip) }

func (p *probeVoice) duck()                                           {}
func (p *probeVoice) tone() time.Duration                             { return 0 }
func (p *probeVoice) render(_ context.Context, t string) (clip, bool) { return clip{text: t}, true }
func (p *probeVoice) play(c clip)                                     { p.onPlay(c) }
func (p *probeVoice) pause()                                          {}
func (p *probeVoice) resume()                                         {}
func (p *probeVoice) discard()                                        {}
func (p *probeVoice) restore()                                        {}

// waitUntil polls cond (no fixed sleeps: the release, a queue position) and
// fails after two seconds.
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// A read cancelled while PARKED suspended (in the air wait, not in a sleep)
// returns at once and is discarded — never resumed after the takeover
// (REVIEW R5-B-02: the cancel did not wake the cond var; settle resumed it).
func TestReadCancelledWhileParkedSuspendedReturnsAtOnce(t *testing.T) {
	v := &scriptVoice{dur: 50 * time.Millisecond}
	n := newNarrator(v)
	gate := make(chan struct{})
	var sleeps atomic.Int32
	n.sleep = func(ctx context.Context, _ time.Duration) bool {
		if sleeps.Add(1) == 1 {
			<-gate // the read's first hold step waits for the takeover to be on air
		}
		return ctx.Err() == nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	readDone := make(chan bool, 1)
	go func() {
		readDone <- n.Run(ctx, narrateRead, true, func(ctx context.Context, s *speaker) { s.line("read"); s.hold(time.Hour) })
	}()
	waitUntil(t, "the read's hold", func() bool { return sleeps.Load() == 1 })
	onAir, release := make(chan struct{}), make(chan struct{})
	go n.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) { s.line("breaking"); close(onAir); <-release })
	<-onAir
	close(gate) // the read's step ends: its next air wait PARKS (suspended)
	waitUntil(t, "the read parked", func() bool {
		n.mu.Lock()
		defer n.mu.Unlock()
		return len(n.suspended) == 1 && n.suspended[0].suspended
	})
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case ok := <-readDone:
		if ok {
			t.Fatal("a cancelled read reports false")
		}
	case <-time.After(time.Second):
		t.Fatalf("the parked read never woke on its cancel: %s", v.got())
	}
	close(release)
	waitUntil(t, "the takeover's release", func() bool { return strings.HasSuffix(v.got(), "restore") })
	if got := v.got(); strings.Contains(got, "resume") || !strings.Contains(got, ",discard,") || strings.Count(got, "restore") != 1 {
		t.Fatalf("discarded, never resumed, one restore: %s", got)
	}
}
