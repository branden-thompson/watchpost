package app

import (
	"context"
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
	nar := newNarrator(v)
	var slept time.Duration
	nar.sleep = func(_ context.Context, d time.Duration) bool { mu.Lock(); slept += d; mu.Unlock(); return true }
	r := newEventReader(context.Background(), nar, nil, func(key string) (tty.SevereRow, bool) { return tornadoRow(), key == "k1" }, func(m tea.Msg) { mu.Lock(); sent = append(sent, m); mu.Unlock() })
	r.status = func(station, short, detail string, spoken time.Duration) {
		mu.Lock()
		overlay = append(overlay, station+" | "+short+" | "+detail[:16]+" | "+spoken.String())
		mu.Unlock()
	}
	r.restore = func() { mu.Lock(); restored++; mu.Unlock() }
	r.Read("k1")
	r.Read("k1") // inert while reading (a goroutine may already be running)
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
	nar := newNarrator(v)
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
	ok := nar.Run(context.Background(), narrateBreaking, true, func(ctx context.Context, s *speaker) { s.line("breaking"); s.hold(100 * time.Millisecond) })
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
	nar := newNarrator(nil) // silent: no voice at all
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
	nar := newNarrator(v)
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
