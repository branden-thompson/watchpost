package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestItemsOfComposesTapeItemsWithLanesAndTimes(t *testing.T) {
	declared := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	expires := time.Date(2026, 8, 27, 16, 15, 0, 0, time.Local)
	evs := []globalfeed.Event{
		{ID: "a", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: declared, Until: expires},
		{ID: "b", Class: globalfeed.ClassSevereWx, Type: "Tornado Watch", Location: "the Dallas area", Severity: globalfeed.SevYellow, At: declared, Until: expires},
		{ID: "c", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", Severity: globalfeed.SevYellow, At: declared},
		{ID: "d", Class: globalfeed.ClassTropical, Type: "Hurricane", Location: "the Atlantic", Severity: globalfeed.SevRed, At: declared},
	}
	items := itemsOf(evs)
	// Lane mapping: Warning / Watch split by the NWS product; quake and tropical own lanes.
	if items[0].Category != tty.CatWarning || items[1].Category != tty.CatWatch || items[2].Category != tty.CatQuake || items[3].Category != tty.CatTropical {
		t.Fatalf("lane mapping: %v", []tty.TickerCategory{items[0].Category, items[1].Category, items[2].Category, items[3].Category})
	}
	// The tape item: type · location  <verb> <t> · expires <t> (warning has a window).
	if items[0].Text != "Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires 4:15 PM" {
		t.Fatalf("warning tape line: %q", items[0].Text)
	}
	// A quake has no active window — no "expires".
	if items[2].Text != "Earthquake · Nepal  recorded 3:42 PM" {
		t.Fatalf("quake tape line (no expires): %q", items[2].Text)
	}
}

// fakeBreakingAudio records the script and proves speak calls never overlap.
type fakeBreakingAudio struct {
	mu        sync.Mutex
	script    []string // "tone", then each spoken line, then "restore"
	inSpeak   bool
	overlap   bool
	dur       time.Duration
	toneDur   time.Duration
	ducked    bool
	restoredN int
}

func (f *fakeBreakingAudio) duck() {
	f.mu.Lock()
	f.ducked = true
	f.mu.Unlock()
}
func (f *fakeBreakingAudio) pause()   {}
func (f *fakeBreakingAudio) resume()  {}
func (f *fakeBreakingAudio) discard() {}
func (f *fakeBreakingAudio) render(_ context.Context, text string) (clip, bool) {
	return clip{text: text, dur: f.dur}, true
}
func (f *fakeBreakingAudio) play(c clip) { f.speak(c.text) }
func (f *fakeBreakingAudio) tone() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.script = append(f.script, "tone")
	f.ducked = true
	return f.toneDur
}
func (f *fakeBreakingAudio) speak(text string) time.Duration {
	f.mu.Lock()
	if f.inSpeak {
		f.overlap = true // a second speak began before the previous returned
	}
	f.inSpeak = true
	f.script = append(f.script, text)
	f.mu.Unlock()
	f.mu.Lock()
	f.inSpeak = false
	f.mu.Unlock()
	return f.dur
}
func (f *fakeBreakingAudio) restore() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.script = append(f.script, "restore")
	f.restoredN++
	f.ducked = false
}

func TestBreakingBurstIsOneSequentialScriptNoOverlap(t *testing.T) {
	audio := &fakeBreakingAudio{dur: time.Millisecond, toneDur: time.Millisecond}
	var mu sync.Mutex
	var msgs []tea.Msg
	d := &tickerDeck{
		send:  func(m tea.Msg) { mu.Lock(); msgs = append(msgs, m); mu.Unlock() },
		muted: &atomic.Bool{}, // false
		voice: newNarrator(audio),
	}
	declared := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	fresh := []globalfeed.Event{
		{ID: "a", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "Cherry, NE", Severity: globalfeed.SevOrange, At: declared, Until: declared.Add(time.Hour)},
		{ID: "b", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: declared, Until: declared.Add(time.Hour)},
	}
	d.breaking(context.Background(), fresh)

	if audio.overlap {
		t.Fatal("a burst must be one sequential script — speak calls overlapped")
	}
	// tone, then each event line (Tornado first — most severe), then the ONE
	// closing tail, then restore.
	if len(audio.script) != 5 || audio.script[0] != "tone" || audio.script[len(audio.script)-1] != "restore" {
		t.Fatalf("script shape: %v", audio.script)
	}
	if !strings.HasPrefix(audio.script[1], "A Tornado Warning") {
		t.Fatalf("most-severe read first: %q", audio.script[1])
	}
	if audio.script[3] != burstClosingLine(nil) {
		t.Fatalf("a burst ends with one closing tail: %q", audio.script[3])
	}
	if audio.ducked || audio.restoredN != 1 {
		t.Fatalf("the duck is restored exactly once: ducked=%v restored=%d", audio.ducked, audio.restoredN)
	}
	// A BreakingDone resumes normal rotation.
	var done bool
	for _, m := range msgs {
		if _, ok := m.(tty.TickerBreakingDoneMsg); ok {
			done = true
		}
	}
	if !done {
		t.Fatal("the sequence ends with TickerBreakingDoneMsg")
	}
}

type fakeSource struct {
	name string
	evs  []globalfeed.Event
}

func (f fakeSource) Name() string                                      { return f.name }
func (f fakeSource) Fetch(context.Context) ([]globalfeed.Event, error) { return f.evs, nil }

func TestCycleSeenMarksAllActiveAndRadiusEmptyShowsNothing(t *testing.T) {
	// 35 warnings — more than the 30-cap. Every one must be seen-marked so an
	// event dropped past the cap can't resurface as "new" and fire a breaking
	// takeover for a stale alert (P4 F4).
	base := time.Now()
	var evs []globalfeed.Event
	for i := 0; i < 35; i++ {
		evs = append(evs, globalfeed.Event{ID: fmt.Sprintf("w%d", i), Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", At: base.Add(time.Duration(i) * time.Minute)})
	}
	var mu sync.Mutex
	var msgs []tea.Msg
	d := &tickerDeck{
		send:    func(m tea.Msg) { mu.Lock(); msgs = append(msgs, m); mu.Unlock() },
		sources: []globalfeed.Source{fakeSource{"NWS", evs}},
		watch:   func() []snapshot.LocationRef { return nil },
		seen:    loadSeen(t.TempDir(), tickerSeenWindow),
		muted:   &atomic.Bool{},
		radius:  &atomic.Int64{}, // 0 = All
	}
	d.cycle(context.Background()) // warm-up seeds all quietly
	if got := len(d.seen.set()); got != 35 {
		t.Fatalf("all 35 active events seen-marked (not just the top 30), got %d", got)
	}

	// Filtered to a radius but no watchlist location → show nothing, not the
	// global stack the UI claims is scoped away (P4 F7).
	d.radius.Store(50)
	d.cycle(context.Background())
	mu.Lock()
	var last tty.TickerMsg
	var found bool
	for _, m := range msgs {
		if tm, ok := m.(tty.TickerMsg); ok {
			last, found = tm, true
		}
	}
	mu.Unlock()
	if !found || len(last.Items) != 0 {
		t.Fatalf("filtered + no location → empty ticker, got found=%v items=%d", found, len(last.Items))
	}
}

func TestCurrentWatchIsALiveStableSnapshot(t *testing.T) {
	lp := &livePipelines{}
	lp.setWatch([]snapshot.LocationRef{{Label: "A"}})
	snap := lp.currentWatch()
	if len(snap) != 1 || snap[0].Label != "A" {
		t.Fatalf("currentWatch returns the set: %v", snap)
	}
	// A Commit-style replacement re-homes the tie; a prior read stays stable.
	lp.setWatch([]snapshot.LocationRef{{Label: "B"}})
	if snap[0].Label != "A" {
		t.Fatal("an earlier snapshot must not change under a later setWatch")
	}
	if lp.currentWatch()[0].Label != "B" {
		t.Fatal("currentWatch reflects the latest watchlist (live tie)")
	}
}

func TestCycleSeenMarksSupersededButHidesIt(t *testing.T) {
	now := time.Now()
	evs := []globalfeed.Event{
		{ID: "old", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", At: now, Superseded: true},
		{ID: "new", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", At: now.Add(time.Minute)},
	}
	var mu sync.Mutex
	var msgs []tea.Msg
	d := &tickerDeck{
		send:    func(m tea.Msg) { mu.Lock(); msgs = append(msgs, m); mu.Unlock() },
		sources: []globalfeed.Source{fakeSource{"NWS", evs}},
		watch:   func() []snapshot.LocationRef { return nil },
		seen:    loadSeen(t.TempDir(), tickerSeenWindow),
		muted:   &atomic.Bool{},
		radius:  &atomic.Int64{},
	}
	d.cycle(context.Background())
	// The superseded id is seen-marked (so it can't resurface as new), but the
	// displayed stack shows only the update.
	if s := d.seen.set(); !s["old"] || !s["new"] {
		t.Fatalf("both ids seen-marked (superseded incl.): %v", s)
	}
	mu.Lock()
	var last tty.TickerMsg
	for _, m := range msgs {
		if tm, ok := m.(tty.TickerMsg); ok {
			last = tm
		}
	}
	mu.Unlock()
	if len(last.Items) != 1 || last.Items[0].ID != "new" {
		t.Fatalf("superseded hidden from display, only the update shown: %+v", last.Items)
	}
}

func TestTapeTextStripsTerminalEscapes(t *testing.T) {
	// A hostile/compromised feed must not smuggle escape/control sequences into
	// the rendered band (OSC-52 clipboard, title spoof, CSI) — the ticker text
	// goes through render.Plain like the rest of the app (P4 F1).
	e := globalfeed.Event{
		Class:    globalfeed.ClassSevereWx,
		Type:     "Tornado Warning\x1b]52;c;cGVvd25lZA==\x07",
		Location: "\x1b]0;PWNED\x07Anytown, OK\x1b[2J",
		At:       time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local),
	}
	got := tapeText(e)
	if strings.ContainsRune(got, 0x1b) || strings.ContainsRune(got, 0x07) {
		t.Fatalf("escape/control bytes survived into the tape line: %q", got)
	}
	// The printable content survives.
	if !strings.Contains(got, "Tornado Warning") || !strings.Contains(got, "Anytown, OK") {
		t.Fatalf("printable text must remain: %q", got)
	}
}

func TestByBreakingPriorityMostSevereThenRecent(t *testing.T) {
	old := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	evs := []globalfeed.Event{
		{ID: "yellow-new", Severity: globalfeed.SevYellow, At: old.Add(time.Hour)},
		{ID: "red-old", Severity: globalfeed.SevRed, At: old},
		{ID: "red-new", Severity: globalfeed.SevRed, At: old.Add(time.Hour)},
	}
	byBreakingPriority(evs)
	if evs[0].ID != "red-new" || evs[1].ID != "red-old" || evs[2].ID != "yellow-new" {
		t.Fatalf("most-severe first, then most-recent: %s %s %s", evs[0].ID, evs[1].ID, evs[2].ID)
	}
}

func TestBreakingItemCarriesTheLaneAndLine(t *testing.T) {
	e := globalfeed.Event{ID: "t", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)}
	it := breakingItem(e)
	if it.Category != tty.CatWarning || it.ID != "t" || it.Text == "" {
		t.Fatalf("breaking item maps to the lane + line: %+v", it)
	}
}

func TestNarrationLinesSingleBurstAndClosing(t *testing.T) {
	declared := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	expires := time.Date(2026, 8, 27, 16, 15, 0, 0, time.Local)
	e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: declared, Until: expires}

	// A burst line: the sentence + "at <start> until <end>", NO tail.
	if got := eventNarration(e); got != "A Tornado Warning has been declared for the Oklahoma City area at 3:42 PM until 4:15 PM" {
		t.Fatalf("event line: %q", got)
	}
	// A single event: the same line + its own broadcast tail.
	if got := alertNarration(nil, e); got != "A Tornado Warning has been declared for the Oklahoma City area at 3:42 PM until 4:15 PM. Press W in Watchpost for the full report on this event" {
		t.Fatalf("single narration: %q", got)
	}
	// A quake (no window): the time, no "until".
	q := globalfeed.Event{Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: declared}
	if got := eventNarration(q); got != "An Earthquake has been recorded for Nepal at 3:42 PM" {
		t.Fatalf("quake line (no until): %q", got)
	}
	// The burst closing is a single shared tail (plural, "any of these events").
	if !strings.Contains(burstClosingLine(nil), "any of these events") {
		t.Fatalf("burst closing: %q", burstClosingLine(nil))
	}
}

func TestPCMDuration(t *testing.T) {
	// 1 s of 16-bit LE stereo at 8000 Hz = 8000 frames × 4 bytes.
	if d := pcmDuration(make([]byte, 8000*4), 8000); d != time.Second {
		t.Fatalf("1 s of PCM = %v", d)
	}
	if d := pcmDuration(make([]byte, 100), 0); d != 0 {
		t.Fatalf("a non-positive rate = 0, got %v", d)
	}
}

func TestSeenStoreColdStartPruneAndPersist(t *testing.T) {
	dir := t.TempDir()
	s := loadSeen(dir, tickerSeenWindow)
	if len(s.set()) != 0 {
		t.Fatal("a fresh cache is empty")
	}
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	// Mark two events, one already 8 days old → pruned on the next mark.
	s.ids["old"] = now.Add(-8 * 24 * time.Hour)
	s.mark([]globalfeed.Event{{ID: "a"}, {ID: "b"}}, now)
	set := s.set()
	if !set["a"] || !set["b"] || set["old"] {
		t.Fatalf("marks recorded; the 8-day-old id pruned: %v", set)
	}
	s.save()

	// Reload: the persisted ids come back (still within the window).
	s2 := loadSeen(dir, tickerSeenWindow)
	if !s2.set()["a"] || !s2.set()["b"] {
		t.Fatalf("the seen ids persist across a reload: %v", s2.set())
	}
}

func TestSeenStoreLoadDropsStale(t *testing.T) {
	dir := t.TempDir()
	s := loadSeen(dir, tickerSeenWindow)
	s.ids["fresh"] = time.Now()
	s.ids["stale"] = time.Now().Add(-30 * 24 * time.Hour)
	s.save()
	if got := loadSeen(dir, tickerSeenWindow).set(); !got["fresh"] || got["stale"] {
		t.Fatalf("a stale id is dropped on load: %v", got)
	}
}

// The tape is ONE line whatever a feed puts in a name (REVIEW R5-C-01): a
// newline or tab collapses to a space.
func TestTapeTextIsOneLine(t *testing.T) {
	e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: "Tornado\nWarning", Location: "Olathe,\tKS", At: time.Now()}
	if s := tapeText(e); strings.ContainsAny(s, "\n\t") {
		t.Fatalf("tape: %q", s)
	}
}

// A takeover goroutine is in the shutdown wait set (REVIEW R5-B-07): stop
// returns only after the sequence in flight has ended.
func TestTickerStopWaitsForATakeoverInFlight(t *testing.T) {
	d := &tickerDeck{done: make(chan struct{})}
	close(d.done)
	d.breakers.Add(1)
	ended := make(chan struct{})
	go func() {
		time.Sleep(60 * time.Millisecond)
		close(ended)
		d.breakers.Done()
	}()
	d.stop()
	select {
	case <-ended:
	default:
		t.Fatal("stop returned before the takeover ended")
	}
}
