package app

import (
	"context"
	"fmt"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/render"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/lineup"
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
	if items[0].Category != tty.CatWarning || items[1].Category != tty.CatWatch || items[2].Category != tty.CatDisasters || items[3].Category != tty.CatMarine {
		t.Fatalf("lane mapping: %v", []tty.TickerCategory{items[0].Category, items[1].Category, items[2].Category, items[3].Category})
	}
	// The item carries the FACTS, not a finished line: the TUI writes the times,
	// so a clock change repaints the tape instead of waiting for the next cycle.
	if items[0].Head != "Tornado Warning · the Oklahoma City area" || items[0].Verb != "declared" {
		t.Fatalf("warning head/verb: %q %q", items[0].Head, items[0].Verb)
	}
	if !items[0].At.Equal(declared) || !items[0].Until.Equal(expires) {
		t.Fatalf("warning times: %v %v", items[0].At, items[0].Until)
	}
	// A quake has no active window and its own verb.
	if items[2].Head != "Earthquake · Nepal" || items[2].Verb != "recorded" || !items[2].Until.IsZero() {
		t.Fatalf("quake item: %+v", items[2])
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

func (f *fakeBreakingAudio) stop() {}

func (f *fakeBreakingAudio) duck() {
	f.mu.Lock()
	f.ducked = true
	f.mu.Unlock()
}
func (f *fakeBreakingAudio) pause()   {}
func (f *fakeBreakingAudio) resume()  {}
func (f *fakeBreakingAudio) discard() {}
func (f *fakeBreakingAudio) render(_ context.Context, _ cast.Role, text string) (clip, bool) {
	return clip{text: text, dur: f.dur}, true
}
func (f *fakeBreakingAudio) play(c clip) { f.speak(c.text) }
func (f *fakeBreakingAudio) tone(cast.Class) time.Duration {
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
	// ONE BAND, WIRED ONCE. The takeover cues through the arbiter's effector, so
	// a deck whose arbiter writes elsewhere observes nothing and proves nothing.
	band := func(m tea.Msg) { mu.Lock(); msgs = append(msgs, m); mu.Unlock() }
	nar := testDirector(audio, band)
	d := &tickerDeck{
		send:  band,
		mc:    nar.mc,
		muted: &atomic.Bool{}, // false
		voice: nar,
		seen:  loadSeen(t.TempDir(), time.Hour),
	}
	// LIVE (I-8): a fixed past date meant both alerts had expired, and the
	// station now declines to read an alert whose window has closed. This test
	// asserts the script's SHAPE, never a spoken time.
	declared := time.Now()
	fresh := []globalfeed.Event{
		{ID: "a", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "Cherry, NE", Severity: globalfeed.SevOrange, At: declared, Until: declared.Add(time.Hour)},
		{ID: "b", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: declared, Until: declared.Add(time.Hour)},
	}
	// THROUGH THE RAIL: it sets the read order, and the read performs it. One
	// call now — the producer offers, the Director schedules, the Composer
	// writes and the Reader speaks (T3.10b).
	newStation(t, d).takeover(context.Background(), fresh)

	if audio.overlap {
		t.Fatal("a burst must be one sequential script — speak calls overlapped")
	}
	// tone, the head naming who declared them, each alert (Tornado first — most
	// severe), the ONE closing tail, then restore. SIX steps, and no empty one:
	// an utterance with nothing in it is a render, a play and a hold for nothing.
	if len(audio.script) != 6 || audio.script[0] != "tone" || audio.script[len(audio.script)-1] != "restore" {
		t.Fatalf("script shape: %v", audio.script)
	}
	for i, step := range audio.script {
		if strings.TrimSpace(step) == "" {
			t.Errorf("step %d is empty: %v", i, audio.script)
		}
	}
	if !strings.HasPrefix(audio.script[1], "The following alerts have been declared by the National Weather Service") {
		t.Fatalf("the head names who declared them, once: %q", audio.script[1])
	}
	if !strings.HasPrefix(audio.script[2], "Tornado Warning for OKC") {
		t.Fatalf("most-severe read first, as its title: %q", audio.script[2])
	}
	if audio.script[4] != burstClosingLine(nil, 0, "alerts") {
		t.Fatalf("a burst ends with one closing tail: %q", audio.script[4])
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
	// 35 warnings — more than the 30-cap. THE SEEDING CYCLE settles every one of
	// them, so nothing already live at launch is announced as though it had just
	// happened (P4 F4). What a WARM cycle settles is a different rule, and
	// TestNothingLiveIsSilencedByTheTapeCap covers it.
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
	got := tapeHead(e)
	if strings.ContainsRune(got, 0x1b) || strings.ContainsRune(got, 0x07) {
		t.Fatalf("escape/control bytes survived into the tape line: %q", got)
	}
	// The printable content survives.
	if !strings.Contains(got, "Tornado Warning") || !strings.Contains(got, "Anytown, OK") {
		t.Fatalf("printable text must remain: %q", got)
	}
}

func TestBreakingItemCarriesTheLaneAndLine(t *testing.T) {
	e := globalfeed.Event{ID: "t", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)}
	it := breakingItem(e)
	if it.Category != tty.CatWarning || it.ID != "t" || it.Head == "" || it.Verb == "" || it.At.IsZero() {
		t.Fatalf("breaking item maps to the lane and carries the facts: %+v", it)
	}
}

func TestNarrationLinesSingleBurstAndClosing(t *testing.T) {
	declared := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	expires := time.Date(2026, 8, 27, 16, 15, 0, 0, time.Local)
	e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: declared, Until: expires}

	// A burst line: the sentence + "at <start> until <end>", NO tail. The times
	// are SPELLED OUT — the report says what it means to say rather than handing
	// the synthesiser a layout written for a column (0.14.0).
	if got := eventNarration(e, render.Clock12, sameDay); got != "A Tornado Warning has been declared for the Oklahoma City area at Three Forty-Two PM until Four Fifteen PM" {
		t.Fatalf("event line: %q", got)
	}
	// A single event: the same line + its own broadcast tail.
	if got := alertNarration(nil, e, render.Clock12, sameDay); got != "A Tornado Warning has been declared for the Oklahoma City area at Three Forty-Two PM until Four Fifteen PM. Press W in Watchpost for the full report on this event" {
		t.Fatalf("single narration: %q", got)
	}
	// A quake (no window): the time, no "until".
	q := globalfeed.Event{Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: declared}
	if got := eventNarration(q, render.Clock12, sameDay); got != "An Earthquake has been recorded for Nepal at Three Forty-Two PM" {
		t.Fatalf("quake line (no until): %q", got)
	}
	// The burst closing is a single shared tail, plural: "any of these alerts"
	// (HUM LEAD, UAT 2026-08-30 — it follows a head that called them alerts).
	if !strings.Contains(burstClosingLine(nil, 0, "alerts"), "any of these alerts") {
		t.Fatalf("burst closing: %q", burstClosingLine(nil, 0, "alerts"))
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
	// RELATIVE TO THE REAL CLOCK, because loadSeen prunes against it. Anchored to
	// a fixed date, this fixture aged out: the marks were written 2026-08-27 and
	// the window is seven days, so on 2026-09-03 the reload pruned everything the
	// test then asserted was still there. The test had been passing for a week
	// while the thing it pins had nothing to do with the date it chose.
	now := time.Now()
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
	if s := tapeHead(e); strings.ContainsAny(s, "\n\t") {
		t.Fatalf("tape: %q", s)
	}
}

// The ticker's stop DRAINS ITS OWN CYCLE (REVIEW R5-B-07): it returns only once
// the run loop has finished its in-flight fetches and its seen-store write, so a
// headless run's temp-directory cleanup cannot race them.
//
// THE TAKEOVER HALF OF THIS RULE MOVED AT T3.10b. The ticker no longer runs the
// read on a goroutine of its own — the read is an effect, dispatched by the
// pump, and the pump drains every dispatched effect before it stops. That half
// is pinned against the real pump in TestStopDrainsEveryDispatchedEffect; what
// is left here is the ticker's own loop, which still needs draining.
func TestTickerStopWaitsForItsRunLoop(t *testing.T) {
	d := &tickerDeck{done: make(chan struct{})}
	ended := make(chan struct{})
	go func() {
		time.Sleep(60 * time.Millisecond)
		close(ended)
		close(d.done) // what run's own defer does when the loop returns
	}()
	d.stop()
	select {
	case <-ended:
	default:
		t.Fatal("stop returned before the run loop ended")
	}
}

// sameDay is the fixtures' own day, so the existing expectations keep reading a
// bare clock time. The date only appears when an event is NOT from today, which
// TestTheTapeDatesAnythingNotFromToday covers.
var sameDay = time.Date(2026, 8, 27, 18, 0, 0, 0, time.Local)

// The BURST, end to end: one tone by the most severe
// alert, one head naming who declared them, each alert's title, one tail.
func TestABurstSoundsOneToneNamesItsAgenciesOnceAndReadsTitles(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.Local)
	at := time.Date(2026, 8, 30, 15, 42, 0, 0, time.Local)
	fresh := []globalfeed.Event{
		{ID: "1", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: at},
		{ID: "2", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "Wichita", Severity: globalfeed.SevOrange, At: at},
		{ID: "3", Source: "NIFC", Class: globalfeed.ClassSevereWx, Type: "Red Flag Warning", Location: "San Diego County", Severity: globalfeed.SevOrange, At: at},
		{ID: "4", Source: "USGS", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", Severity: globalfeed.SevRed, At: at},
	}

	// (2) ONE head, each agency once, in first-appearance order, Oxford comma.
	want := "the National Weather Service, the National Interagency Fire Center, and the United States Geological Survey"
	if got := burstAgencies(fresh); got != want {
		t.Errorf("agencies:\n got %q\nwant %q", got, want)
	}
	head := burstHead(nil, fresh)
	if !strings.HasPrefix(head, "The following alerts have been declared by "+want) {
		t.Errorf("head reads %q", head)
	}
	if strings.Count(head, "National Weather Service") != 1 {
		t.Errorf("two alerts from one office name it ONCE: %q", head)
	}

	// (3) Each alert is its TITLE, where and when — the head already said they
	// were declared, so no line repeats it.
	for _, e := range fresh {
		line := breakingLine(nil, e, true, render.Clock12, now)
		if !strings.HasPrefix(line, e.Title()+" for "+e.Location) {
			t.Errorf("burst line for %s reads %q", e.ID, line)
		}
		if strings.Contains(line, "has been declared") || strings.Contains(line, "has been recorded") {
			t.Errorf("the burst line repeats the head's verb: %q", line)
		}
	}

	// (4) One tail, and it is not on any of the lines.
	tail := burstClosingLine(nil, 0, "alerts")
	if !strings.Contains(tail, "press W in Watchpost") {
		t.Errorf("tail reads %q", tail)
	}
	for _, e := range fresh {
		if strings.Contains(breakingLine(nil, e, true, render.Clock12, now), "Watchpost") {
			t.Errorf("%s carries the tail; a burst has ONE, after all of them", e.ID)
		}
	}
}

// (1) ONE tone, by the most severe alert (MVS-D-12) — asked of the WHOLE burst.
//
// `breaking` used to classify the FIRST event, which was right only while the
// burst was sorted by severity. T3.1's ladder orders by rung, so the first card
// can be the milder hazard; `worstOf` is what the rule actually needs.
func TestTheBurstsToneComesFromItsWorstHazardNotItsFirst(t *testing.T) {
	now := time.Now()
	fresh := []globalfeed.Event{
		{ID: "quake", Type: "Earthquake", Severity: globalfeed.SevYellow, At: now}, // leads the READ
		{ID: "warn", Type: "Tornado Warning", Severity: globalfeed.SevRed, At: now},
		{ID: "watch", Type: "Tornado Watch", Severity: globalfeed.SevYellow, At: now},
	}
	if got := worstOf(fresh).ID; got != "warn" {
		t.Fatalf("the worst hazard of the burst is the tornado warning, got %q", got)
	}
	if got := cast.Classify(worstOf(fresh).Type); got == cast.ClassWatch {
		t.Errorf("the burst's tone must not come from the least severe lane, got %v", got)
	}
	// A TIE GOES TO THE LOUDER TONE, NOT THE EARLIER CARD (MVS-D-73, which
	// supersedes the positional rule this test used to assert).
	//
	// These two are the INAUDIBLE case, and it is the one worth pinning: a
	// Tornado Warning and an Earthquake share the dual-tone preset, so a listener
	// cannot tell which was chosen. The rank still decides it, so the answer is
	// the same on every run instead of depending on how the batch arrived.
	tied := []globalfeed.Event{
		{ID: "warning", Type: "Tornado Warning", Severity: globalfeed.SevRed, At: now},
		{ID: "disaster", Type: "Earthquake", Severity: globalfeed.SevRed, At: now},
	}
	if got := worstOf(tied).ID; got != "disaster" {
		t.Errorf("a severity tie is broken by tone rank, got %q", got)
	}
	if a, b := cast.Classify("Tornado Warning"), cast.Classify("Earthquake"); cast.ToneName(a) != cast.ToneName(b) {
		t.Errorf("this case is only inaudible while the two share a preset: %s vs %s", cast.ToneName(a), cast.ToneName(b))
	}
}

// A single event keeps its own sentence and its own tail: the head exists to
// stop FOUR copies of "has been declared for", and one alert is not four.
func TestASingleEventHasNoBurstHead(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.Local)
	e := globalfeed.Event{ID: "1", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", At: now}
	line := breakingLine(nil, e, false, render.Clock12, now)
	if !strings.Contains(line, "has been declared for") {
		t.Errorf("a single event reads its own sentence: %q", line)
	}
	if !strings.Contains(line, "Watchpost") {
		t.Errorf("a single event carries its own tail: %q", line)
	}
}

// THE SCREEN ANSWERS THE TONE (HUM LEAD, UAT 2026-08-30: "a noticeable lag
// between alert tone, the ticker showing the centered takeover and the
// readout").
//
// The centred takeover used to be sent from readBreaking — after the burst head
// had been rendered AND spoken — so the band lagged the sound by a render and a
// whole sentence. The tone is the cue; the frame that answers it must be the
// next thing that happens, before any words are rendered.
func TestTheCentredTakeoverLandsWithTheToneNotAfterTheWords(t *testing.T) {
	audio := &fakeBreakingAudio{dur: time.Millisecond, toneDur: time.Millisecond}
	var mu sync.Mutex
	var order []string // "tone" / "breaking-msg" / each spoken line
	band := func(m tea.Msg) {
		if _, ok := m.(tty.TickerBreakingMsg); ok {
			mu.Lock()
			order = append(order, "breaking-msg")
			mu.Unlock()
		}
	}
	nar := testDirector(audio, band)
	d := &tickerDeck{
		send:  band,
		mc:    nar.mc,
		muted: &atomic.Bool{},
		voice: nar,
		seen:  loadSeen(t.TempDir(), time.Hour),
	}
	at := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	newStation(t, d).takeover(context.Background(), []globalfeed.Event{
		{ID: "a", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: at},
		{ID: "b", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "Cherry, NE", Severity: globalfeed.SevOrange, At: at},
	})

	audio.mu.Lock()
	script := append([]string(nil), audio.script...)
	audio.mu.Unlock()
	mu.Lock()
	sent := len(order)
	mu.Unlock()

	if len(script) == 0 || script[0] != "tone" {
		t.Fatalf("the tone opens the sequence: %v", script)
	}
	if sent == 0 {
		t.Fatal("the takeover was never sent to the band")
	}
	// The band was told BEFORE the head was spoken. script[1] is the head, so a
	// message sent after it would have arrived at least one sentence late.
	if len(script) < 2 || !strings.HasPrefix(script[1], "The following alerts") {
		t.Fatalf("the head follows the tone: %v", script)
	}
}

// THE BAND AND THE RADIO NAME THE SAME MOMENT.
//
// The feeds publish UTC. The spoken line localised and the tape did not, so the
// band read "0026" for an event the radio called "Seventeen Twenty-Six Hours" —
// the same instant, seven hours apart. The two paths format independently, which
// is why they need a test that compares them rather than one that checks each.
func TestTheTapeAndTheNarrationNameTheSameMoment(t *testing.T) {
	// 00:26 UTC — the hour that exposed it, because it is a different DAY in
	// most of the United States.
	at := time.Date(2026, 8, 31, 0, 26, 0, 0, time.UTC)
	e := globalfeed.Event{ID: "x", Source: "NWS", Class: globalfeed.ClassSevereWx,
		Type: "Tornado Warning", Location: "OKC", At: at}
	now := at.Add(30 * time.Minute)

	for _, c := range []render.Clock{render.Clock12, render.Clock24, render.ClockMil} {
		// What the RADIO says, and what the BAND shows, for the same event.
		spoken := eventNarration(e, c, now)
		item := breakingItem(e)
		shown := c.Since(item.At.Local(), now)
		// The band's time must be the local one — the same wall clock the radio
		// spelled out.
		if !strings.Contains(spoken, c.Spoken(at.Local())) {
			t.Errorf("%v: the radio says %q, expected it to name %q", c, spoken, c.Spoken(at.Local()))
		}
		if !strings.Contains(shown, c.Time(at.Local())) {
			t.Errorf("%v: the band shows %q, expected it to name %q", c, shown, c.Time(at.Local()))
		}
	}
}

// breakingDeck is a ticker deck wired for the takeover path: fake audio so a
// read costs a millisecond rather than a hold, and every centred frame the
// takeover sends recorded in the order it was sent.
func breakingDeck(t *testing.T, evs []globalfeed.Event) (*tickerDeck, func() []string) {
	t.Helper()
	return breakingDeckWith(t, evs, &fakeBreakingAudio{dur: time.Millisecond, toneDur: time.Millisecond})
}

// breakingDeckWith is breakingDeck over a chosen voice.
//
// THE VOICE MUST BE CHOSEN BEFORE THE STATION IS BUILT (red team 2026-09-05).
// newStation captures deck.voice into the executors at construction, so a
// caller that swapped d.voice AFTERWARDS was swapping a field nothing read —
// which is how TestNoAdmittedAlertIsDroppedUnread came to run the same fixture
// twice and compare it with itself for fourteen seconds.
func breakingDeckWith(t *testing.T, evs []globalfeed.Event, audio narrationVoice) (*tickerDeck, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var read []string
	band := func(m tea.Msg) {
		if v, ok := m.(tty.TickerBreakingMsg); ok {
			mu.Lock()
			read = append(read, v.Item.Head)
			mu.Unlock()
		}
	}
	nar := testDirector(audio, band)
	d := &tickerDeck{
		send:    band,
		sources: []globalfeed.Source{fakeSource{name: "stub", evs: evs}},
		watch:   func() []snapshot.LocationRef { return nil },
		seen:    loadSeen(t.TempDir(), time.Hour),
		muted:   &atomic.Bool{}, radius: &atomic.Int64{}, done: make(chan struct{}),
		voice: nar,
		mc:    nar.mc,
	}
	d.warm.Store(true) // past the quiet seeding cycle: these events are new
	// THE STATION IS THE REST OF THE PATH (T3.10b). The deck produces arrivals;
	// the Director, the Composer and the Reader turn them into what is said, and
	// the caller drains them by asking what was read aloud.
	st := newStation(t, d)
	return d, func() []string {
		st.drain(context.Background())
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), read...)
	}
}

// TestEveryArrivalIsEitherReadOrCounted WAS HERE, and is deleted (red team
// 2026-09-05). It called lineup.Plan directly, so it duplicated
// platform/lineup's own TestTheDivertCountIsExactlyWhatWasNotRead — which pins
// read+divert==arrived over a four-row table, against a Plan that ALSO enforces
// it with an invariant. It paid for a temp dir, a fake source, an arbiter and a
// whole station to use one thing from them: d.fence(), which returns the zero
// Fence for a bare deck anyway.
//
// What was app-owned about it — that the rail's Max is defaultBurstMax — is
// asserted by TestNoAdmittedAlertIsDroppedUnread, which reads the bound from
// the planner rather than from a literal.
func TestEveryLiveAlertIsEventuallyReadAloud(t *testing.T) {
	now := time.Now()
	var evs []globalfeed.Event
	for i := range 35 { // one lane, more than the tape holds
		evs = append(evs, globalfeed.Event{ID: fmt.Sprintf("w%d", i), Source: "NWS",
			Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Place: fmt.Sprintf("County %d", i),
			Severity: globalfeed.SevRed, At: now.Add(-time.Duration(i) * time.Minute), Until: now.Add(time.Hour)})
	}
	d, readAloud := breakingDeck(t, evs)
	for range 10 { // ten cycles is more than enough at the Max per takeover
		d.cycle(context.Background())
		readAloud() // drains the schedule the cycle just fed
	}
	spoken := map[string]bool{}
	for _, head := range readAloud() {
		spoken[head] = true
	}
	if len(spoken) != len(evs) {
		t.Errorf("every live alert must be read eventually, cap or no cap: %d of %d were", len(spoken), len(evs))
	}
}

// TestTheTickerCuesThroughTheOneOwnerNotItsOwnSend — D-1, pinned for the LIVE
// takeover path.
//
// The band's takeover message used to be constructed here AND in
// app/executors.go, so one rule had two carriers in two files and they could
// drift into sending different things for one event. Asserting that a cue
// "reaches the band" cannot see the difference, because both writers reach the
// same band in production. So this deck's own send and its effector's band are
// DIFFERENT captures, and the cue must land on the effector's.
func TestTheTickerCuesThroughTheOneOwnerNotItsOwnSend(t *testing.T) {
	var mu sync.Mutex
	var direct, throughOwner []string
	audio := &fakeBreakingAudio{dur: time.Millisecond, toneDur: time.Millisecond}
	owner := func(m tea.Msg) {
		if v, ok := m.(tty.TickerBreakingMsg); ok {
			mu.Lock()
			throughOwner = append(throughOwner, v.Item.Head)
			mu.Unlock()
		}
	}
	nar := testDirector(audio, owner)
	d := &tickerDeck{
		send: func(m tea.Msg) {
			if v, ok := m.(tty.TickerBreakingMsg); ok {
				mu.Lock()
				direct = append(direct, v.Item.Head)
				mu.Unlock()
			}
		},
		mc:    nar.mc,
		muted: &atomic.Bool{},
		voice: nar,
		seen:  loadSeen(t.TempDir(), time.Hour),
	}
	at := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	newStation(t, d).takeover(context.Background(), []globalfeed.Event{
		{ID: "a", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: at},
	})
	mu.Lock()
	defer mu.Unlock()
	if len(throughOwner) != 1 {
		t.Errorf("the band's owner saw %v, want the one takeover", throughOwner)
	}
	if len(direct) != 0 {
		t.Errorf("the ticker wrote the band itself: %v — that is the second carrier this task removed", direct)
	}
}

// NO ADMITTED ALERT IS EVER DROPPED UNREAD (T3.1, DR-3).
//
// This is the defect's removal, stated as a property rather than a boundary.
// Bounds apply at ADMISSION: once an alert is on the rail it is read. The old
// arrangement bounded twice — by count when the burst was chosen, and again by
// TIME while it was being read — so whichever hazards sorted last were the ones
// a slow read silenced. Every previous fix moved that boundary (a per-lane
// floor, then a derived time budget) and a reviewer's sweep still found the
// break at about 8.1 s a read.
//
// THE ASSERTION IS DELIBERATELY INDEPENDENT OF THE BOUND'S VALUE. It reads the
// same burst twice, once with instant narration and once with narration well
// past that break point, and requires the same alerts to be read both times. A
// bound that lives at admission cannot care how long a read takes; a bound that
// cuts mid-burst must. Pinning "5 are read" instead would go green the moment
// somebody moved the budget again, which is exactly the history here.
func TestNoAdmittedAlertIsDroppedUnread(t *testing.T) {
	now := time.Now()
	outbreak := func() []globalfeed.Event {
		var evs []globalfeed.Event
		for i := range 10 {
			evs = append(evs, globalfeed.Event{ID: fmt.Sprintf("warn%d", i), Source: "NWS",
				Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: fmt.Sprintf("County %d", i),
				Severity: globalfeed.SevRed, At: now.Add(-time.Duration(i) * time.Minute), Until: now.Add(time.Hour)})
		}
		return evs
	}
	// Virtual time: narration REPORTS a realistic length while the director's
	// sleep returns at once, so a budget binds without the test taking as long
	// as the broadcast would.
	readWith := func(dur time.Duration) []string {
		// THE VOICE GOES IN AT CONSTRUCTION. Assigning d.voice afterwards
		// swapped a field the station had already captured, so both runs were
		// the 1 ms one and the comparison below was a slice against itself —
		// the whole named property measured nothing (red team 2026-09-05).
		d, readAloud := breakingDeckWith(t, outbreak(), &fakeBreakingAudio{dur: dur, toneDur: time.Second})
		d.voice.sleep = func(context.Context, time.Duration) bool { return true }
		d.cycle(context.Background())
		return readAloud()
	}
	quick := readWith(time.Millisecond)
	slow := readWith(12 * time.Second) // past the measured break point

	// EVERYTHING ADMITTED IS READ, which is the property itself. Comparing the
	// two runs alone would miss a bound that does not depend on time — a loop
	// that simply stopped after three would read three both ways and look
	// perfectly consistent. Asked against the rail's own admission rather than
	// against a literal, so moving the Max cannot make this go quietly green.
	probe, _ := breakingDeck(t, outbreak())
	admitted, _ := planned(t, probe, outbreak())
	for _, run := range []struct {
		what string
		got  []string
	}{{"instant narration", quick}, {"slow narration", slow}} {
		if len(run.got) != len(admitted) {
			t.Errorf("with %s the rail admitted %d alerts and only %d were read — "+
				"%d admitted alert(s) were dropped unread:\n  read: %v",
				run.what, len(admitted), len(run.got), len(admitted)-len(run.got), run.got)
		}
	}

	if len(quick) != len(slow) {
		t.Errorf("the same burst read %d alerts with instant narration and %d with slow narration — "+
			"a bound that cuts mid-burst silenced %d admitted alert(s) that a faster read would have spoken.\n"+
			"  quick: %v\n  slow:  %v",
			len(quick), len(slow), len(quick)-len(slow), quick, slow)
	}
}

// NOTHING OUTRANKS A TAKEOVER — and `holdRest` depends on it.
//
// `s.hold` waits AIR time, which stops while a sequence is suspended, but
// `holdRest` subtracts WALL time from the budget. If a takeover could ever be
// parked mid-line, the suspension would be charged against that line's own hold
// and the read would be cut short — a defect that would look like alerts
// truncating at random under load.
//
// It cannot happen while a takeover is the highest class, because `admit`
// suspends what is on air only for a STRICTLY higher class. This is the guard on
// that assumption: adding a class above narrateBreaking makes the hazard real,
// and this test is what says so before a listener finds out.
func TestNothingOutranksATakeover(t *testing.T) {
	// WALKS THE TYPE, NOT A LIST. The first version iterated
	// []narrationClass{narrateRead, narrateBreaking}, so a class ADDED to the
	// const block was simply not in it and the guard stayed green through the
	// one change it exists to catch — found by review, which added a class and
	// watched this pass.
	for c := narrationClass(0); c < numNarrationClasses; c++ {
		if c > narrateBreaking {
			t.Fatalf("a narration class outranks a takeover (%d > %d), so a takeover can now be "+
				"suspended mid-line — holdRest subtracts wall time and must be reworked to "+
				"count air time before this ships", c, narrateBreaking)
		}
	}
}

// A TIE IN SEVERITY IS BROKEN BY THE LOUDER TONE, NOT BY POSITION (MVS-D-73).
//
// `Severity` is a three-value colour tier, so hazards of different kinds share
// one constantly. Ranking on it alone left the tone to whatever order the rail
// happened to produce: a red hurricane (Marine, low sweep) and a red tornado
// warning (Warnings, EAS dual-tone) in one burst sounded whichever came first.
// That is the positional mechanism the tone fix set out to remove, surviving
// inside the tie.
func TestASeverityTieIsBrokenByTheLouderTone(t *testing.T) {
	now := time.Now()
	hurricane := globalfeed.Event{ID: "h", Source: "NHC", Class: globalfeed.ClassTropical,
		Type: "Hurricane", Location: "Gulf of Mexico", Severity: globalfeed.SevRed, At: now}
	tornado := globalfeed.Event{ID: "t", Source: "NWS", Class: globalfeed.ClassSevereWx,
		Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: now}

	// THE FIXTURE MUST ACTUALLY TIE, and the quieter tone must be the one that
	// would win on position — or this passes without exercising the rule.
	if hurricane.Severity != tornado.Severity {
		t.Fatalf("the fixture no longer ties on severity: %v vs %v", hurricane.Severity, tornado.Severity)
	}
	if cast.Classify(hurricane.Type).ToneRank() >= cast.Classify(tornado.Type).ToneRank() {
		t.Fatalf("the fixture needs the hurricane to carry the QUIETER tone, got %d vs %d",
			cast.Classify(hurricane.Type).ToneRank(), cast.Classify(tornado.Type).ToneRank())
	}
	// BOTH ORDERS, because a rule decided by position passes one of them.
	for _, order := range [][]globalfeed.Event{{hurricane, tornado}, {tornado, hurricane}} {
		if got := worstOf(order).ID; got != "t" {
			t.Errorf("with order %s,%s the tone came from %q — a severity tie must sound the "+
				"louder tone, not whichever card the schedule put first",
				order[0].ID, order[1].ID, got)
		}
	}
}

// MVS-D-26 — THE TONE MUTE SILENCES TONES. IT DOES NOT SILENCE THE STATION.
//
// This starts where the LISTENER starts (D-12): they tick a class in the ALERT
// TONES group, which calls saveTones, and then they relaunch. Every earlier
// mute test began at the atomic flag and so could not see the seam that set it.
//
// The chain that broke it: Save derives ticker_muted from the tone mode, and
// tickerState seeded the station's "do not speak to me" flag from that field.
// ticker_muted is a BACK-COMPAT MIRROR, written so a 0.13.0 binary reading this
// file still mutes its ticker; it is not this binary's state. Reading it back
// as runtime state meant muting one tone class silenced every spoken alert from
// the next launch on — and nothing could clear it, because MVS-D-48 took the
// toggle away when [M] became a deep link into Settings.
//
// BOTH HALVES ARE ASSERTED. The seam, so the defect cannot come back by a
// different route, and the CONSEQUENCE — words in the air — so deleting the
// seed alone cannot leave the listener silent for some other reason.
func TestMutingAToneClassDoesNotSilenceTheStationAtTheNextLaunch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// A STATION THAT HAS BEEN SET UP. saveTones edits the file in place, and a
	// first-run config is refused, so there has to be one to edit.
	if err := config.Save(config.Config{Locations: []config.Location{
		{Label: "Oceanside, CA", Tag: "OSIDE", Zip: "92057", Lat: 33.24, Lon: -117.29},
	}}); err != nil {
		t.Fatalf("seeding the config: %v", err)
	}
	// THE LISTENER'S ACTION, through the code Settings actually calls.
	if err := saveTones(nil, cast.Tones{Mode: cast.ModeTonesMute, Muted: []string{cast.ClassWarning.Key()}}); err != nil {
		t.Fatalf("saving the tone preference: %v", err)
	}
	cfg, err := config.Load() // the next launch
	if err != nil {
		t.Fatalf("loading the saved config: %v", err)
	}
	// THE FIXTURE IS ASSERTED VALID FIRST. If the file did not carry the mute,
	// or the mirror were not written, this test would pose nothing and pass.
	if cfg.Radio.Tones.Mode != cast.ModeTonesMute {
		t.Fatalf("the fixture did not persist the tone mute: %+v", cfg.Radio.Tones)
	}
	if !cfg.TickerMuted {
		t.Fatal("the fixture did not write the 0.13.0 ticker_muted mirror; this test poses nothing without it")
	}

	prefs, _ := tickerState(cfg)
	if prefs.muted.Load() {
		t.Error("muting a tone class left the station muted at launch: the tone preference is not the listener " +
			"saying 'do not speak to me', and nothing in 0.14.0 can clear this flag once it is set")
	}

	// AND THE WORDS REACH THE AIR. The seam above is the cause; this is what
	// the listener actually loses.
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	seen := loadSeen(t.TempDir(), time.Hour)
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: prefs.muted, voice: nar, seen: seen}
	fresh := breakingFixture()
	newStation(t, deck).takeover(context.Background(), fresh)
	if !seen.set()[fresh[0].ID] {
		t.Error("with a tone class muted, the burst was never read aloud: a hazard the listener is never told about")
	}
}

// MVS-D-78 — STANDBY HOLDS A BURST; IT DOES NOT SPEND IT.
//
// [M] used to let the takeover run inaudibly: it cued the band, held, and
// marked each alert READ. A tornado warning arriving while muted was consumed
// in silence and never sounded, even on unmuting a minute later.
//
// The visual half is asserted alongside, because the whole ruling turns on it:
// muting silences the AUDIO channel and leaves the visual one alone (the TV
// analogy — picture live, subtitles advancing).
func TestMutingHoldsABurstRatherThanSpendingIt(t *testing.T) {
	fresh := breakingFixture()
	seen := loadSeen(t.TempDir(), time.Hour)
	muted := &atomic.Bool{}
	muted.Store(true)

	// DETERMINISTIC AIR. A fake clip has no duration, so the reader falls back
	// to breakingHold (5 s) per part and the unmuted half below would race the
	// test's own deadline rather than measure anything.
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	var mu sync.Mutex
	var sent []tea.Msg
	deck := &tickerDeck{
		send:  func(m tea.Msg) { mu.Lock(); sent = append(sent, m); mu.Unlock() },
		muted: muted,
		voice: nar,
		seen:  seen,
	}
	st := newStation(t, deck)
	// THE PRODUCER'S OWN DECISION, ASSERTED BEFORE ANYTHING DRAINS.
	//
	// RED TEAM 2026-09-05: deleting the mute gate from startTakeover left the
	// WHOLE REPOSITORY green. Asserting after the drain cannot see it — the
	// burst reaches the Director, speak declines on the second mute gate, the
	// Failed takes the card off the rail, and the rail reads empty either way.
	// The gate's whole claim is that the burst never reaches the schedule.
	st.offer(fresh)
	if n := st.pendingCount(); n != 0 {
		t.Errorf("a muted burst told the Director %d time(s); standby holds it, it does not offer it", n)
	}
	st.drain(context.Background())

	// NOTHING WAS SPENT. Every alert is still new, so it reads when the
	// listener comes back — or expires quietly if it went stale meanwhile.
	for _, e := range fresh {
		if seen.set()[e.ID] {
			t.Errorf("%s was marked read while muted: a hazard consumed in silence", e.ID)
		}
	}
	// AND THE BAND WAS NEVER CUED. A callout for a read that never happens is
	// the stale-band defect from the other direction.
	mu.Lock()
	held := append([]tea.Msg(nil), sent...)
	mu.Unlock()
	for _, m := range held {
		if _, ok := m.(tty.TickerBreakingMsg); ok {
			t.Error("a muted takeover must not cue the band for a read it will not perform")
		}
	}
	// AND NOTHING WAS ADMITTED. Under the Director a queued card is a PROMISE to
	// read (DR-3), so a muted burst must not reach the schedule at all — held on
	// the rail it would sit there until the listener came back, and read from
	// there it would be consumed. Neither is what "standby holds it" means.
	if got := st.dir.Lineup().Cards(lineup.AlertRail); len(got) != 0 {
		t.Errorf("a muted burst put %d cards on the rail; standby holds it, it does not queue it", len(got))
	}

	// UNMUTED, THE SAME BURST IS SPENT. Without this the test passes against a
	// takeover that never runs at all.
	muted.Store(false)
	st.takeover(context.Background(), fresh)
	if !seen.set()[fresh[0].ID] {
		t.Error("unmuted, the burst must be read and marked")
	}
}
