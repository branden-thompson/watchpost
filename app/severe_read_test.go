package app

import (
	"context"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/modes/tty"
)

func tornadoRow() tty.SevereRow {
	return tty.SevereRow{Key: "k1", Tab: tty.SevereWarnings, Product: "Tornado Warning", Location: "Olathe, KS",
		Record: tty.SevereRecord{Title: "TORNADO WARNING", Meta: "[Extreme · Immediate · Observed]",
			Timing: "Declared 08/28 08:45 CDT   Expires 08/28 09:30 CDT   (~45m)", Area: "Area: Johnson County, KS · NWS Kansas City",
			Paras: []string{"Wind gusts to 70 mph · Hail to 1.00 in", "At 845 AM CDT, a severe thunderstorm capable of producing a tornado was located near Olathe, moving northeast at 30 mph. HAZARD: Damaging tornado and quarter size hail. SOURCE: Radar indicated rotation.", "Instructions: TAKE COVER NOW! Move to a basement or an interior room on the lowest floor of a sturdy building."}}}
}

func TestEventScriptReadsTheAlertItself(t *testing.T) {
	s := eventScript(nil, tornadoRow())
	if !strings.HasPrefix(s, "This is a Watchpost Severe Weather Notification Report. Notifications may be delayed and are not intended for life safety use. Tornado Warning for") || !strings.HasSuffix(s, " This concludes this Watchpost Severe Weather Notification Report.") {
		t.Errorf("the report opens with the notice and closes with the sign-off:\n%s", s)
	}
	for _, want := range []string{"Tornado Warning for Olathe, KS.", "Extreme, Immediate, Observed.", "In effect for about 45 minutes.", "Wind gusts to 70 mph · Hail to 1.00 in.", "moving northeast at 30 mph.", "Instructions. TAKE COVER NOW!"} {
		if !strings.Contains(s, want) {
			t.Errorf("script lacks %q:\n%s", want, s)
		}
	}
	for _, never := range []string{"Declared 08/28", "Expires 08/28", "Press W", "\x1b"} { // the clock line stays off the air (provider prose may name its own times)
		if strings.Contains(s, never) {
			t.Errorf("script must not carry %q:\n%s", never, s)
		}
	}
	long := tornadoRow()
	long.Record.Paras = []string{strings.Repeat("A sentence of prose. ", 80), "Instructions: " + strings.Repeat("Do this. ", 40)}
	if got := eventScript(nil, long); strings.Count(got, "A sentence of prose.") != 80 || strings.Count(got, "Do this.") != 40 {
		t.Errorf("the whole record is spoken, nothing clipped (UAT 2026-08-28):\n%s", got)
	}
	if got := spokenWindow("Declared x   Expires y   (~1h30m)"); got != "1 hour 30 minutes" {
		t.Errorf("window: %q", got)
	}
	if got := spokenWindow("Recorded 08/28 10:00 UTC"); got != "" {
		t.Errorf("no window: %q", got)
	}
}

func TestEventReaderDucksSpeaksRestoresAndOverlaysThePanel(t *testing.T) {
	var mu sync.Mutex
	var sent []tea.Msg
	var overlay []string
	restored := 0
	v := &scriptVoice{dur: 3 * time.Second}
	nar := testDirector(v, nil)
	var slept time.Duration
	// Deterministic air: the read's FIRST hold step blocks on readGate, so the
	// second Read below is provably made WHILE the first read is in progress.
	// The stubbed voice and stubbed sleep otherwise finish the whole sequence
	// before the next line runs, and then the inertness assertion passes on
	// timing rather than on the guard it is there to hold. It did not, on a
	// Linux runner: the second read ran in full, four reading messages for two.
	readGate := make(chan struct{})
	var sleeps atomic.Int32
	nar.sleep = func(_ context.Context, d time.Duration) bool {
		mu.Lock()
		slept += d
		mu.Unlock()
		if sleeps.Add(1) == 1 {
			<-readGate
		}
		return true
	}
	r := newEventReader(context.Background(), nar, nil, func(key string) (tty.SevereRow, bool) { return tornadoRow(), key == "k1" }, func(m tea.Msg) { mu.Lock(); sent = append(sent, m); mu.Unlock() })
	r.status = func(station, short, detail string, spoken time.Duration) {
		mu.Lock()
		overlay = append(overlay, station+" | "+short+" | "+detail[:16]+" | "+spoken.String())
		mu.Unlock()
	}
	r.restore = func() { mu.Lock(); restored++; mu.Unlock() }
	r.Toggle("k1") // the production entry point: the press owns the mark (T3.2b review)
	r.Read("k1")   // inert while reading — PINNED: the first read is held below
	close(readGate)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := restored
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := v.got(); got != "duck,speak:"+eventScript(nil, tornadoRow())+",restore" {
		t.Fatalf("voice sequence: %s", got)
	}
	mu.Lock()
	sentN, overlayN, restoredN, sleptN := len(sent), len(overlay), restored, slept
	msgs, over := append([]tea.Msg(nil), sent...), append([]string(nil), overlay...)
	mu.Unlock()
	if sentN != 2 || msgs[0].(tty.SevereReadingMsg).Key != "k1" || msgs[1].(tty.SevereReadingMsg).Key != "" {
		t.Fatalf("reading messages: %+v", msgs)
	}
	if overlayN != 1 || !strings.HasPrefix(over[0], "EVENT · Tornado Warning · Olathe, KS | EVENT · TOR · Olathe, KS | This is a Watchp") || !strings.HasSuffix(over[0], "3s") {
		t.Fatalf("overlay: %v", over)
	}
	if restoredN != 1 || sleptN != 3*time.Second {
		t.Fatalf("restore %d, slept %v", restoredN, sleptN)
	}
	r.Read("nope") // an unknown key reads nothing
	waitUntil(t, "the unknown key's no-op", func() bool { r.mu.Lock(); defer r.mu.Unlock(); return !r.busy })
	if got := v.got(); strings.Count(got, "speak") != 1 {
		t.Fatalf("an unknown key must not touch the voice: %v", got)
	}
}

// A breaking takeover suspends a read in progress: the takeover speaks
// alone, the read resumes and finishes (its mark clears at its own end),
// the broadcast is restored once.
func TestEventReadIsSuspendedByABreakingTakeover(t *testing.T) {
	v := &scriptVoice{}
	nar := testDirector(v, nil)
	// Deterministic air: the read's FIRST hold step blocks on readGate (the
	// takeover has not started, so the first sleep is the read's); every
	// later step — the takeover's hold, the read's remaining steps — returns
	// at once. The read cannot finish before the takeover resumes it, so the
	// sequence is the same under any load (it flaked on real time under -race).
	readGate := make(chan struct{})
	var sleeps atomic.Int32
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		if sleeps.Add(1) == 1 {
			<-readGate
		}
		return ctx.Err() == nil
	}
	var mu sync.Mutex
	var sent []tea.Msg
	reading := make(chan struct{}, 1)
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(m tea.Msg) {
		mu.Lock()
		sent = append(sent, m)
		mu.Unlock()
		if v, ok := m.(tty.SevereReadingMsg); ok && v.Key != "" {
			reading <- struct{}{}
		}
	})
	r.status = func(string, string, string, time.Duration) {}
	v.dur = 300 * time.Millisecond // the read's hold (three gated steps)
	r.Read("k1")
	<-reading
	// The mark precedes the first line: wait for the read to be on air (its
	// first hold step parked on the gate) before the takeover.
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline) && sleeps.Load() == 0; {
		time.Sleep(5 * time.Millisecond)
	}
	ok := nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(ctx context.Context, s *speaker) { s.line("breaking"); s.hold(100 * time.Millisecond) })
	close(readGate) // the takeover is over: the read plays on
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(sent)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if !ok || len(sent) != 2 || sent[1].(tty.SevereReadingMsg).Key != "" {
		t.Fatalf("the read must finish after the takeover (its mark clearing at its own end): ok=%v sent=%+v", ok, sent)
	}
	if got := v.got(); !strings.HasPrefix(got, "duck,speak:This is a Watchpost") || !strings.Contains(got, ",pause,aside:breaking,resume,") || !strings.HasSuffix(got, "restore") || strings.Count(got, "restore") != 1 {
		t.Fatalf("voice: %s", got)
	}
}

// An override directory re-words a phrase without touching the app: the
// script files are the contract (domains/radio/script).
func TestEventScriptHonoursAnOverrideDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "event-report"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "event-report", "head.txt"), []byte("Watchpost test bulletin."), 0o600); err != nil {
		t.Fatal(err)
	}
	s := eventScript(script.New(dir), tornadoRow())
	if !strings.HasPrefix(s, "Watchpost test bulletin. Tornado Warning for Olathe, KS.") || !strings.HasSuffix(s, "Notification Report.") {
		t.Fatalf("override head, built-in tail:\n%s", s)
	}
}

// Without a voice the overlay still shows for a reading-length hold — the
// panel says what is being "read" even in silence (round 4, C-11b).
func TestEventReadWithoutAVoiceHoldsTheOverlay(t *testing.T) {
	nar := testDirector(nil, nil) // silent: no voice at all
	nar.sleep = func(context.Context, time.Duration) bool { return true }
	var held time.Duration
	var mu sync.Mutex
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(tea.Msg) {})
	r.status = func(_, _, _ string, d time.Duration) { mu.Lock(); held = d; mu.Unlock() }
	r.Read("k1")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		h := held
		mu.Unlock()
		if h > 0 {
			if h > readingHoldMax {
				t.Fatalf("the silent hold is bounded: %v", h)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the overlay never showed")
}

// A read in progress ends with the app (round 4, A-08): End cancels it —
// its hold ends, its mark clears, the broadcast is restored — and waits
// for it; inert with none in progress.
func TestEventReadEndsWithTheApp(t *testing.T) {
	v := &scriptVoice{dur: time.Minute} // a long read
	nar := testDirector(v, nil)
	nar.sleep = sleepCtx
	var mu sync.Mutex
	var sent []tea.Msg
	reading := make(chan struct{}, 1)
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(m tea.Msg) {
		mu.Lock()
		sent = append(sent, m)
		mu.Unlock()
		if v, ok := m.(tty.SevereReadingMsg); ok && v.Key != "" {
			reading <- struct{}{}
		}
	})
	r.End() // nothing in progress: inert
	r.Read("k1")
	<-reading
	waitUntil(t, "the read on air", func() bool { return strings.Contains(v.got(), "speak:") })
	start := time.Now()
	r.End()
	if took := time.Since(start); took > time.Second {
		t.Fatalf("Stop waited %v for a cancelled read", took)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 2 || sent[1].(tty.SevereReadingMsg).Key != "" {
		t.Fatalf("the mark clears at the stop: %+v", sent)
	}
	if got := v.got(); !strings.HasSuffix(got, ",restore") {
		t.Fatalf("the broadcast is restored: %s", got)
	}
	r.mu.Lock()
	busy := r.busy
	r.mu.Unlock()
	if busy {
		t.Fatal("the reader is free again")
	}
}

// TestASpaceReadSoundsNoTone is MVS-D-69, and the reason is worth stating
// where the test is: a tone is an ATTENTION SIGNAL, and a listener who pressed
// [space] on a row they are looking at has already given theirs.
//
// It was also four seconds of delay. Every tone carries a two-second trailing
// silence so a takeover has a beat between the signal and "…has been declared",
// and the read held for the whole buffer — 2.8 to 4.0 seconds by class, of which
// under a second and a half is audible. Measured at UAT as: tone, four seconds,
// words.
//
// THE TAKEOVER'S TONE IS ASSERTED IN THE SAME TEST, because the ruling is about
// who asked for the read and not about tones: an unattended alert still has to
// fetch the listener.
func TestASpaceReadSoundsNoTone(t *testing.T) {
	// dur is short so the read finishes at once: with a zero-length line the
	// reader falls back to a hold proportional to the script, which for a
	// tornado warning is seconds of real time and would make this test slow for
	// a reason unrelated to what it pins. toneDur is deliberately LONG — if a
	// tone were still sounded, the read would take four seconds and this test
	// would notice by timing out as well as by the sequence.
	v := &scriptVoice{dur: time.Millisecond, toneDur: 4 * time.Second}
	nar := testDirector(v, nil)
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) {
		return tornadoRow(), true
	}, func(tea.Msg) {})
	r.Read("k")
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(v.got(), "restore") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !strings.Contains(v.got(), "restore") {
		t.Fatalf("the read never finished: %q — it must not be waiting on a tone", v.got())
	}

	if got := v.got(); strings.Contains(got, "tone") {
		t.Errorf("the [space] read sounded a tone: %q — the listener who pressed it is already here", got)
	}
	if !strings.HasPrefix(v.got(), "duck,speak:") {
		t.Errorf("the read opened with %q, want the words straight after the duck", v.got())
	}
}

// A READ PAUSES AND RESUMES ON [space], AND THE BED COMES BACK WHILE IT WAITS
// (MVS-D-74).
//
// Two properties, and the second is the one that is easy to miss: a paused read
// HOLDS NOTHING. The arbiter dips the broadcast when a sequence takes the air
// and restores it when nothing is waiting or suspended — and a paused job sits
// on the suspended stack, so counting it there would leave the broadcast dipped
// for as long as a listener left the read paused, with nobody speaking over it.
func TestAPausedReadReleasesTheBedAndResumesOnRequest(t *testing.T) {
	v := &scriptVoice{dur: time.Hour} // long enough that the read is still on air
	nar := testDirector(v, nil)
	onAir := make(chan struct{})
	go nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(_ context.Context, s *speaker) {
		close(onAir)
		s.hold(time.Hour)
	})
	<-onAir
	waitUntil(t, "the read to take the air", func() bool { return nar.mc.givenWay() })

	if !nar.pauseRead() {
		t.Fatal("there was no read to pause")
	}
	waitUntil(t, "the bed to come back while paused", func() bool { return !nar.mc.givenWay() })
	if !nar.readPaused() {
		t.Error("the read is paused but does not report itself so; the window cannot draw the mark")
	}
	if nar.pauseRead() {
		t.Error("an already-paused read was paused again")
	}

	if !nar.resumeRead() {
		t.Fatal("the paused read would not resume")
	}
	waitUntil(t, "the bed to dip again on resume", func() bool { return nar.mc.givenWay() })
	if nar.readPaused() {
		t.Error("the read still reports itself paused after resuming")
	}
	if nar.resumeRead() {
		t.Error("a running read was resumed again")
	}
}

// A PAUSED READ IS NEVER PROMOTED BACK BY THE ARBITER.
//
// It shares the suspended stack with a job a takeover displaced, and those DO
// resume on their own. If settle treated them alike, a paused read would start
// speaking again the moment an alert finished — the listener's hold silently
// undone by an unrelated event.
func TestATakeoverEndingDoesNotResumeAPausedRead(t *testing.T) {
	v := &scriptVoice{dur: time.Hour}
	nar := testDirector(v, nil)
	onAir := make(chan struct{})
	go nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(_ context.Context, s *speaker) {
		close(onAir)
		s.hold(time.Hour)
	})
	<-onAir
	if !nar.pauseRead() {
		t.Fatal("there was no read to pause")
	}

	// A takeover comes and goes while the read is held.
	done := make(chan struct{})
	go func() {
		defer close(done)
		nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(context.Context, *speaker) {})
	}()
	<-done

	if !nar.readPaused() {
		t.Error("the takeover ending resumed a read the listener had paused")
	}
}

// CLOSING THE WINDOW NEVER WAITS FOR THE READ (MVS-D-75, UAT 2026-09-03).
//
// `close` runs on Bubbletea's UPDATE goroutine, so anything it blocks on is a
// frame that does not redraw. The first version called `End`, which waits for
// the read's goroutine — right at shutdown, and about half a second of visible
// hesitation on a keypress. *"Key controls for showing/hiding MUST feel instant
// — that's the whole point of using a TUI."*
//
// The pin is the DIFFERENCE between the two, not a stopwatch on one: `Cancel`
// returns while the read is still running, `End` returns only once it has
// finished. A timing threshold alone would pass on a fast machine with the
// blocking call still in place.
func TestClosingTheWindowStopsTheReadWithoutWaitingForIt(t *testing.T) {
	release := make(chan struct{})
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	// The read parks until the test lets it go, so "still running" is a fact
	// rather than a race with a fast fake.
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return ctx.Err() == nil
	}
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(tea.Msg) {})
	r.Read("k1")
	waitUntil(t, "the read to start", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.busy
	})

	r.mu.Lock()
	done := r.done
	r.mu.Unlock()

	r.Cancel() // the keypress path
	select {
	case <-done:
		t.Fatal("Cancel waited for the read to finish; on the update goroutine that is a frame that does not redraw")
	default: // still running, which is the point: Cancel cancelled and returned
	}
	close(release)
}

// AND THE SHUTDOWN PATH DOES WAIT — the distinction the split exists for.
//
// A SEPARATE READ, because Cancel FREES the reader: asking End afterwards would
// find nothing left to wait for and return at once, which looks exactly like the
// defect while being the right answer to a different question. The first version
// of this test did that and failed for that reason.
func TestShutdownWaitsForTheReadItEnds(t *testing.T) {
	release := make(chan struct{})
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return ctx.Err() == nil
	}
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(tea.Msg) {})
	r.Read("k1")
	waitUntil(t, "the read to start", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.busy
	})
	r.mu.Lock()
	done := r.done
	r.mu.Unlock()

	close(release)
	r.End()
	select {
	case <-done:
	default:
		t.Error("End returned while the read was still running; shutdown would race its cache writes")
	}
}

// AND THE WINDOW'S HOOK IS WIRED TO THE ONE THAT DOES NOT WAIT.
//
// The test above pins `Cancel`'s behaviour and cannot see which of the two the
// window actually calls — swapping the hook back to `End` left it green,
// checked. This asserts the WIRING, which is where the half-second lived.
func TestTheWindowsCloseHookDoesNotWaitForTheRead(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return ctx.Err() == nil
	}
	lp := &livePipelines{}
	lp.reader = newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(tea.Msg) {})
	lp.reader.Read("k1")
	waitUntil(t, "the read to start", func() bool {
		lp.reader.mu.Lock()
		defer lp.reader.mu.Unlock()
		return lp.reader.busy
	})
	lp.reader.mu.Lock()
	done := lp.reader.done
	lp.reader.mu.Unlock()

	lp.endEventRead()() // exactly what tty's close() calls, on the update goroutine
	select {
	case <-done:
		t.Error("the window's close hook waited for the read to finish — that is the frame that did not redraw")
	default:
	}
}

// FAST INPUT NEVER STACKS READS (UAT 2026-09-03).
//
// The sequence a listener actually performed: open, [space], [esc], reopen,
// [space] on a DIFFERENT row. The first read's cleanup ran after the second had
// started, cleared `busy` while it was playing and took its mark down — so the
// next press launched a third read, and the window no longer knew which one
// [space] should pause. *"It will then be playing two reports — and the program
// gets confused when I hit space which report to pause."*
//
// The reader is generational now: a read winding down clears nothing that
// belongs to the one that replaced it (the radioDeck.epoch pattern).
func TestFastInputNeverLeavesTwoReadsRunning(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	var mu sync.Mutex
	var marks []string
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	// Every read parks, so each one is still in flight when the next arrives —
	// which is the whole scenario. A fake that finished instantly would never
	// overlap and the pin would prove nothing.
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return ctx.Err() == nil
	}
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true },
		func(m tea.Msg) {
			if v, ok := m.(tty.SevereReadingMsg); ok {
				mu.Lock()
				marks = append(marks, v.Key)
				mu.Unlock()
			}
		})

	r.Toggle("k1") // open, [space]
	waitUntil(t, "the first read", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.busy
	})
	r.Cancel()     // [esc]
	r.Toggle("k2") // reopen, navigate, [space] on another row
	waitUntil(t, "the second read", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.busy && r.key == "k2"
	})

	// THE READER KNOWS EXACTLY ONE READ IS CURRENT, and knows which.
	r.mu.Lock()
	busy, key := r.busy, r.key
	r.mu.Unlock()
	if !busy || key != "k2" {
		t.Fatalf("after [esc] and a second press the reader holds busy=%v key=%q, want the second read", busy, key)
	}

	// ON AIR, not merely launched: `busy` is set when the goroutine starts, and
	// the job still has to take the air from the cancelled one. Toggling before
	// then asks the arbiter to pause something that is not playing.
	waitUntil(t, "the second read to take the air", func() bool { return nar.mc.givenWay() })

	// And a further press is a PAUSE of that read, never a third read.
	r.Toggle("k2")
	waitUntil(t, "the second read to pause", func() bool { return nar.readPaused() })
	r.mu.Lock()
	stillOne := r.busy && r.key == "k2"
	r.mu.Unlock()
	if !stillOne {
		t.Error("pausing the current read lost track of it; the next press would start another")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(marks) == 0 {
		t.Fatal("no marks were sent, so this proves nothing about which row the window shows")
	}
	if last := marks[len(marks)-1]; last != "k2" {
		t.Errorf("the window's last mark is %q, not the read that is actually running: %v", last, marks)
	}
}

// CLOSING THE WINDOW NEVER SENDS ON THE CALLER'S GOROUTINE (UAT 2026-09-03).
//
// `send` is tea.Program.Send and `close` runs on Bubbletea's UPDATE goroutine.
// A send from there puts a message into the channel that the blocked loop is the
// only one able to drain — the app freezes, no key works, and the listener has
// to kill the terminal. *"Soft lock is so bad I actually had to kill the entire
// terminal to fix."*
//
// THE PIN IS A SEND THAT NEVER RETURNS, which is what a full channel looks like
// from the caller's side. If the close path sends, this hangs — exactly as the
// app did. A test that merely counted sends would pass while the deadlock
// remained, because the count would be right and the goroutine wrong.
func TestTheClosePathNeverSendsAndSoCannotSoftlock(t *testing.T) {
	stuck := make(chan struct{})
	defer close(stuck)
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true },
		// A send that never returns — but only for the CLEAR, which is the message
		// the close path would send. Blocking every send would wedge this test's
		// own Read in its setup, which is what it did when the set-mark moved
		// into Read: the fixture hung before reaching a single assertion.
		func(m tea.Msg) {
			if v, ok := m.(tty.SevereReadingMsg); ok && v.Key == "" {
				<-stuck
			}
		})
	r.Read("k1")
	waitUntil(t, "the read to start", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.busy
	})

	for _, tc := range []struct {
		what string
		call func()
	}{
		{"Cancel", r.Cancel},
		{"the window's close hook", func() { lpFor(r).endEventRead()() }},
	} {
		returned := make(chan struct{})
		go func() { defer close(returned); tc.call() }()
		select {
		case <-returned:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s blocked on a send; on Bubbletea's update goroutine that is the softlock", tc.what)
		}
	}
}

// lpFor is the production wiring around one reader, so a test can call the hook
// the window actually calls rather than a stand-in for it.
func lpFor(r *eventReader) *livePipelines { return &livePipelines{reader: r} }

// A READ CAN BE PAUSED WHILE A TAKEOVER HAS THE AIR — the other ordering.
//
// `TestATakeoverEndingDoesNotResumeAPausedRead` pauses first and then lets an
// alert arrive. This is the reverse, and it was broken: while the takeover
// speaks the read is on the SUSPENDED stack rather than on the air, so
// `pauseRead` looked only at `onAir`, found a takeover, and did nothing. The
// listener pressed [space] and got no response — and when the alert finished,
// `settle` promoted the read back, because nothing had marked it as their hold.
//
// Found by review. The window is the whole duration of any takeover, against a
// read that runs for minutes.
func TestAReadCanBePausedWhileATakeoverIsSpeaking(t *testing.T) {
	v := &scriptVoice{dur: time.Hour}
	nar := testDirector(v, nil)
	readOn := make(chan struct{})
	go nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(_ context.Context, s *speaker) {
		close(readOn)
		s.hold(time.Hour)
	})
	<-readOn
	waitUntil(t, "the read on air", func() bool { return nar.mc.givenWay() })

	// A takeover displaces it and holds the air until released.
	release := make(chan struct{})
	takeoverDone := make(chan struct{})
	go func() {
		defer close(takeoverDone)
		nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(context.Context, *speaker) { <-release })
	}()
	waitUntil(t, "the read to be displaced", func() bool {
		nar.mu.Lock()
		defer nar.mu.Unlock()
		return len(nar.suspended) == 1
	})

	if !nar.pauseRead() {
		t.Fatal("[space] while a takeover is speaking did nothing; the listener's press was ignored")
	}
	if !nar.readPaused() {
		t.Error("the read does not report itself paused, so the chip would still offer Pause")
	}

	close(release)
	<-takeoverDone
	// THE HOLD SURVIVES THE TAKEOVER. This is what was actually broken: the read
	// went back on the air the moment the alert finished.
	waitUntil(t, "the takeover to leave the air", func() bool {
		nar.mu.Lock()
		defer nar.mu.Unlock()
		return nar.onAir == nil
	})
	if !nar.readPaused() {
		t.Error("the takeover ending resumed a read the listener had paused")
	}
}

// TWO FAST PRESSES NEVER MISROUTE THE PAUSE OR THE MARK.
//
// Bubbletea runs each Cmd on its own goroutine, so two `[space]` presses are
// genuinely concurrent Toggles — the update loop does not serialise them. Each
// read the reader's state, released the lock, and acted on a snapshot the other
// had already invalidated: the loser could pause the read the winner had just
// started, and marked the row with the OLD key, putting the ▶ on a row that was
// not reading.
//
// INVISIBLE TO THE RACE DETECTOR — every access is properly locked; it is the
// DECISION that was racing, not the memory. Found by review at ~2 % of presses;
// this drives enough rounds to make that a near-certainty.
func TestConcurrentPressesNeverMarkARowThatIsNotReading(t *testing.T) {
	var mu sync.Mutex
	var bad []string
	v := &scriptVoice{dur: time.Millisecond}
	nar := testDirector(v, nil)
	nar.sleep = func(context.Context, time.Duration) bool { return true }
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tornadoRow(), true },
		func(m tea.Msg) {
			v, ok := m.(tty.SevereReadingMsg)
			if !ok || v.Key == "" {
				return // a clear names no row and cannot disagree with one
			}
			mu.Lock()
			defer mu.Unlock()
			bad = append(bad, v.Key)
		})

	// SIZED FOR THE RACE DETECTOR, WHICH IS HOW THE GATE RUNS IT. Measured by a
	// reviewer against the broken code: 20/20 under -race, but only ~81/100
	// without it, with batch-to-batch swings from 50 % to 92 %. So a bare
	// `go test ./app -run ...` during development can miss this — the guarantee
	// is `make race`, and 400 rounds bought no extra certainty while pushing the
	// package's -race run from 90 s to 600 s.
	marked := 0
	for round := 0; round < 60; round++ {
		var wg sync.WaitGroup
		for _, k := range []string{"k1", "k2"} {
			wg.Add(1)
			go func(k string) { defer wg.Done(); r.Toggle(k) }(k)
		}
		wg.Wait()

		// THE READER AND THE WINDOW MUST AGREE. Whatever won, the last row the
		// window was told about must be the row the reader holds.
		r.mu.Lock()
		busy, key := r.busy, r.key
		r.mu.Unlock()
		mu.Lock()
		var last string
		if len(bad) > 0 {
			last = bad[len(bad)-1]
		}
		bad = nil
		mu.Unlock()
		if busy && last != "" && last != key {
			t.Fatalf("round %d: the window shows %q while the reader holds %q", round, last, key)
		}
		if last != "" {
			marked++
		}
		r.Cancel()
	}
	// THE FIXTURE MUST HAVE MARKED SOMETHING. With no marks sent at all the
	// disagreement check above can never fire and the test passes proving
	// nothing — which it did, when a mutation removed the mark entirely rather
	// than misrouting it.
	if marked == 0 {
		t.Fatal("no row was ever marked, so this proves nothing about which row the window shows")
	}
}

// A READ FOR A ROW THAT IS GONE LEAVES NO MARK BEHIND.
//
// The set-mark happens when the read STARTS, before the goroutine has looked the
// row up — and a row can vanish from the feed between the frame the listener saw
// and the key they pressed. The clear lived inside the narration sequence, which
// that path never reaches, so the window kept a play mark for a read that never
// began and nothing could take it down.
//
// Found by review. The existing fixture called `Read("nope")` and asserted
// nothing about the messages afterwards, so it walked straight past this.
func TestAReadForAMissingRowLeavesNoMark(t *testing.T) {
	var mu sync.Mutex
	var marks []tty.SevereReadingMsg
	nar := testDirector(&scriptVoice{dur: time.Millisecond}, nil)
	r := newEventReader(context.Background(), nar, nil,
		func(string) (tty.SevereRow, bool) { return tty.SevereRow{}, false }, // the row has gone
		func(m tea.Msg) {
			if v, ok := m.(tty.SevereReadingMsg); ok {
				mu.Lock()
				marks = append(marks, v)
				mu.Unlock()
			}
		})

	r.Toggle("gone")
	waitUntil(t, "the read to finish", func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.busy
	})
	waitUntil(t, "the mark to come down", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(marks) > 0 && marks[len(marks)-1].Key == ""
	})

	mu.Lock()
	defer mu.Unlock()
	if last := marks[len(marks)-1]; last.Key != "" {
		t.Errorf("the window still shows %q reading, for a row that was never found: %v", last.Key, marks)
	}
}

// A READ ASKED FOR *DURING* AN ALERT CAN BE PAUSED BEFORE IT EVER SPEAKS.
//
// The third ordering, and the third time this defect arrived. A read requested
// while a takeover has the air is QUEUED — not on air, not suspended — so a
// pause that looked only at those two did nothing, and the read started on its
// own when the alert ended. The listener pressed [space] twice and got a read
// they had told to wait.
func TestAQueuedReadCanBePausedBeforeItTakesTheAir(t *testing.T) {
	nar := testDirector(&scriptVoice{dur: time.Hour}, nil)
	release := make(chan struct{})
	onAir := make(chan struct{})
	go nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(context.Context, *speaker) {
		close(onAir)
		<-release
	})
	<-onAir

	// The read queues behind the alert.
	go nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(_ context.Context, s *speaker) {
		s.hold(time.Hour)
	})
	waitUntil(t, "the read to queue", func() bool {
		nar.mu.Lock()
		defer nar.mu.Unlock()
		return len(nar.waiting) == 1
	})

	if !nar.pauseRead() {
		t.Fatal("a read waiting behind an alert could not be paused; the press was ignored")
	}
	if !nar.readPaused() {
		t.Error("the read does not report itself paused, so the chip would still offer Pause")
	}

	close(release)
	waitUntil(t, "the alert to leave the air", func() bool {
		nar.mu.Lock()
		defer nar.mu.Unlock()
		return nar.onAir == nil
	})
	// THE HOLD SURVIVES: the queued read must not take the air it was told to wait for.
	nar.mu.Lock()
	started := nar.onAir != nil
	nar.mu.Unlock()
	if started {
		t.Error("the alert ended and the paused read started anyway")
	}
}

// A PAUSE NEVER CLAIMS A READ THAT IS ALREADY DEAD.
//
// A read Cancel has killed stays on the suspended stack until its goroutine
// unwinds, and a walk that matched on class alone found that corpse first. It
// reported success — so the chip said Paused — while the read the listener could
// actually hear carried on. A pause that LIES is worse than one that refuses.
//
// ASSERTED ON `pausable` DIRECTLY, because driving it through Run is a race with
// the goroutine unwinding: the first version of this pin did that and stayed
// green against the defect about half the time, which is a pin that reports
// clean on broken code.
func TestAPauseNeverClaimsACancelledRead(t *testing.T) {
	nar := testDirector(&scriptVoice{dur: time.Millisecond}, nil)
	dead, killed := context.WithCancel(context.Background())
	killed()
	live := context.Background()

	nar.mu.Lock()
	nar.suspended = []*narrationJob{
		{class: narrateRead, ctx: dead}, // cancelled, not yet unwound
	}
	nar.waiting = []*narrationJob{
		{class: narrateRead, ctx: live}, // the one the listener can still hear
	}
	got := nar.reads()
	nar.mu.Unlock()

	if len(got) != 1 {
		t.Fatalf("want exactly the live read, got %d candidates", len(got))
	}
	if got[0].ctx.Err() != nil {
		t.Error("a cancelled read was offered as pausable; the chip would say Paused with nothing held")
	}
}

// EVERY ASKER AGREES ABOUT WHERE A READ IS, AND ABOUT DEAD ONES.
//
// Five functions need to know where a read can be and whether it still counts —
// pausing, resuming, reporting a hold, choosing what to promote, and deciding
// whether the bed may come back. Each used to walk the state itself, and they
// drifted exactly as carriers of one rule always do. Four review rounds went
// into finding the places one at a time; `live` and `reads` are the answer
// written once.
//
// EACH CELL CARRIES ITS OWN EXPECTATION. An earlier version asserted "nothing is
// held" for every cell, which is only true where every read is dead — it failed
// on the one cell that has a live paused read in it, and the failure was the
// test being wrong rather than the code.
func TestThePauseAndTheReportNeverDisagree(t *testing.T) {
	dead, kill := context.WithCancel(context.Background())
	kill()
	alive := context.Background()

	for _, tc := range []struct {
		what       string
		onAir      *narrationJob
		suspended  []*narrationJob
		waiting    []*narrationJob
		wantHeld   bool // readPaused: a listener is holding something
		wantResume bool // resumeRead: there is a held read to give the air back to
	}{
		{what: "a cancelled read on the air",
			onAir: &narrationJob{class: narrateRead, ctx: dead}},
		{what: "a cancelled read suspended",
			suspended: []*narrationJob{{class: narrateRead, ctx: dead}}},
		{what: "a cancelled read PAUSED and suspended",
			suspended: []*narrationJob{{class: narrateRead, ctx: dead, paused: true}}},
		{what: "a cancelled read paused and waiting",
			waiting: []*narrationJob{{class: narrateRead, ctx: dead, paused: true}}},
		{what: "a cancelled read waiting, not paused",
			waiting: []*narrationJob{{class: narrateRead, ctx: dead}}},
		// THE CELL THAT PUT A CORPSE ON THE AIR. A paused read on top of the
		// stack shields the dead one underneath from settle's drain, which stops
		// at the first live job — so the dead one sat there waiting to be
		// promoted, and was, with the bed ducked under it.
		{what: "a cancelled read beneath a live paused one",
			suspended: []*narrationJob{
				{class: narrateRead, ctx: dead},
				{class: narrateRead, ctx: alive, paused: true},
			},
			wantHeld: true, wantResume: true},
	} {
		t.Run(tc.what, func(t *testing.T) {
			nar := testDirector(&scriptVoice{dur: time.Millisecond}, nil)
			nar.mu.Lock()
			nar.onAir, nar.suspended, nar.waiting = tc.onAir, tc.suspended, tc.waiting
			nar.mu.Unlock()

			// NOTHING DEAD IS EVER PROMOTED. settle would put it on the air and
			// duck the bed under a read that had already been cancelled.
			nar.mu.Lock()
			promoted := nar.innermostResumable()
			nar.mu.Unlock()
			if promoted != nil && promoted.ctx.Err() != nil {
				t.Error("a dead read was offered for promotion; settle would put a corpse on the air")
			}

			// THE PAUSE NEVER CLAIMS A CORPSE. Where every read is dead there is
			// nothing to claim at all.
			if nar.pauseRead() {
				nar.mu.Lock()
				held := nar.heldRead()
				nar.mu.Unlock()
				if held == nil || held.ctx.Err() != nil {
					t.Error("pauseRead reported success over a dead read; the chip would say Paused with nothing held")
				}
			}

			// AND THE REPORT AGREES WITH BOTH.
			nar.mu.Lock()
			nar.onAir, nar.suspended, nar.waiting = tc.onAir, tc.suspended, tc.waiting
			nar.mu.Unlock()
			if got := nar.readPaused(); got != tc.wantHeld {
				t.Errorf("readPaused = %v, want %v — the chip and the arbiter disagree", got, tc.wantHeld)
			}
			if got := nar.resumeRead(); got != tc.wantResume {
				t.Errorf("resumeRead = %v, want %v", got, tc.wantResume)
			}
		})
	}
}

// A PAUSED READ WAITING BEHIND AN ALERT GIVES THE BED BACK.
//
// `first()` learned to skip paused jobs; the take-back guard counted the queue
// instead of asking it, so a queued read the listener had paused kept the
// broadcast dipped for as long as they left it — with nobody speaking over it.
// That is verbatim what the guard's own comment forbids, in the one place that
// had not learned the rule.
func TestAPausedQueuedReadGivesTheBedBack(t *testing.T) {
	nar := testDirector(&scriptVoice{dur: time.Hour}, nil)
	release, onAir := make(chan struct{}), make(chan struct{})
	go nar.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(context.Context, *speaker) {
		close(onAir)
		<-release
	})
	<-onAir
	waitUntil(t, "the alert to duck the bed", func() bool { return nar.mc.givenWay() })

	go nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(_ context.Context, s *speaker) {
		s.hold(time.Hour)
	})
	waitUntil(t, "the read to queue", func() bool {
		nar.mu.Lock()
		defer nar.mu.Unlock()
		return len(nar.waiting) == 1
	})
	if !nar.pauseRead() {
		t.Fatal("the queued read could not be paused")
	}

	close(release) // the alert ends; nothing is speaking now
	waitUntil(t, "the bed to come back", func() bool { return !nar.mc.givenWay() })
}

// A READ THAT IS ALREADY GONE DOES NOT HOLD THE BED.
//
// The take-back guard asks `first()`, which learned to skip a listener's hold
// but not a job whose context had ended. A cancelled read still sits in the
// queue until its goroutine unwinds, so the broadcast stayed dipped for a read
// nobody was going to hear — the same corpse window that made the pause lie,
// arriving at the other half of the same guard.
func TestADeadWaitingReadDoesNotHoldTheBed(t *testing.T) {
	nar := testDirector(&scriptVoice{dur: time.Millisecond}, nil)
	dead, kill := context.WithCancel(context.Background())
	kill()

	nar.mc.giveWay() // something was speaking; the bed is down
	if !nar.mc.givenWay() {
		t.Fatal("the fixture did not dip the bed, so this proves nothing")
	}

	nar.mu.Lock()
	nar.waiting = []*narrationJob{{class: narrateRead, ctx: dead}}
	nar.settle()
	nar.mu.Unlock()

	if nar.mc.givenWay() {
		t.Error("the bed is still down for a read whose context has already ended")
	}
}

// MVS-D-75 — CLOSING THE WINDOW STOPS THE READ, AUDIO INCLUDED.
//
// UAT 2026-09-05, and the ruling already existed: "closing the window stops the
// read". What it did was end the read's SEQUENCE. The arbiter then released the
// air, settled, and resumed the location read the `[w]` read had suspended —
// while the `[w]` clip played on, because nothing stopped it. Two reports at
// once, and the window showed no play mark on either, since the reader
// correctly believed it had finished.
//
// The engine could pause a line, resume one, and close the HELD ones. It could
// not stop the line that was actually sounding.
func TestClosingTheWindowStopsTheReadsAudioNotJustItsSequence(t *testing.T) {
	v := &scriptVoice{}
	nar := testDirector(v, nil)
	// THE READ MUST STILL BE READING WHEN IT IS CANCELLED. A sleep that returns
	// at once makes the read FINISH immediately whatever its clip's length, and
	// a finished read is correctly never stopped — so the assertion below would
	// have been about the wrong thing. It passed in isolation and failed in the
	// suite, which is the tell: the ordering was luck, not design.
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		<-ctx.Done() // park in the hold until [esc] cancels it
		return false
	}

	reading := make(chan struct{}, 1)
	r := newEventReader(context.Background(), nar, nil, func(string) (tty.SevereRow, bool) { return tornadoRow(), true }, func(m tea.Msg) {
		if v, ok := m.(tty.SevereReadingMsg); ok && v.Key != "" {
			select {
			case reading <- struct{}{}:
			default:
			}
		}
	})
	r.status = func(string, string, string, time.Duration) {}
	v.dur = time.Hour // a read long enough that it CANNOT have finished on its own
	r.Read("k1")
	<-reading
	// WAIT UNTIL IT IS ACTUALLY SPEAKING. The mark is sent BEFORE the read's
	// goroutine starts, so cancelling on the mark alone can arrive before the
	// line ever plays — and then "no stop was recorded" would be true of a
	// perfect implementation too, which is this release's most repeated
	// mistake.
	for deadline := time.Now().Add(2 * time.Second); !strings.Contains(v.got(), "speak:"); {
		if time.Now().After(deadline) {
			t.Fatalf("the read never reached its line, so this proves nothing: %s", v.got())
		}
		time.Sleep(5 * time.Millisecond)
	}

	r.Cancel() // what [esc] does
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if strings.Contains(v.got(), "stop") {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("a read cut short must stop its line, or it plays on under whatever resumes: %s", v.got())
}

// AND A READ THAT FINISHES ON ITS OWN IS NEVER STOPPED. A part's hold is its
// own length LESS the work already done, so the sequence can end a hair before
// the audio drains — closing then would clip the tail of every read.
func TestAReadThatFinishesIsNeverCutOff(t *testing.T) {
	v := &scriptVoice{}
	nar := testDirector(v, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	ok := nar.Run(context.Background(), narrateRead, cast.SevereRead, true, func(ctx context.Context, s *speaker) {
		s.line("a complete read")
	})
	if !ok {
		t.Fatal("the read must complete, or this proves nothing about completion")
	}
	if strings.Contains(v.got(), "stop") {
		t.Errorf("a completed read is not stopped: %s", v.got())
	}
}
