package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// tickerMuteState is the shared [M] mute flag (seeded from config) and the hook
// the dashboard calls on a toggle: it updates the flag the pipeline reads and
// persists the preference.
func tickerMuteState(muted bool) (*atomic.Bool, func(bool)) {
	flag := &atomic.Bool{}
	flag.Store(muted)
	return flag, func(m bool) {
		flag.Store(m)
		_ = savePreference(func(c *config.Config) { c.TickerMuted = m })
	}
}

// tickerRadiusState is the shared alert-radius (miles; 0 = All/global) the
// pipeline reads, and the hook the Setup window calls to change it: it updates
// the value the ticker filters by and persists the preference (0.12.0).
func tickerRadiusState(mi int) (*atomic.Int64, func(int)) {
	v := &atomic.Int64{}
	v.Store(int64(mi))
	return v, func(m int) {
		v.Store(int64(m))
		_ = savePreference(func(c *config.Config) { c.TickerRadiusMi = m })
	}
}

// tickerEvery is the ticker's fetch cadence. The feeds' own TTLs (USGS 5 min /
// NHC 30 min / NWS 2 min) ride the httpx cache, so the fast tick only hits the
// network for the sources due — the byte floor is measured at the P3 gate.
const tickerEvery = 2 * time.Minute

// tickerSeenWindow is how long a seen event id is remembered across restarts so
// it is not re-announced (the 7-day USGS window; HUM LEAD).
const tickerSeenWindow = 7 * 24 * time.Hour

// tickerRotate is how long each category lane holds the marquee before it
// rotates to the next non-empty lane (HUM LEAD 2026-08-27, #6).
const tickerRotate = 90 * time.Second

// tickerDeck runs the global event ticker's separate pipeline (0.12.0): fetch
// the three global feeds on a cadence, tie each event to a representative
// location, stack them, publish the marquee, and detect genuinely NEW events
// (the P3 tone/narration will sound those unless muted).
type tickerDeck struct {
	send    func(tea.Msg) // publishes to the dashboard (p.Send in production; a capture in tests)
	sources []globalfeed.Source
	watch   func() []snapshot.LocationRef // the current watchlist, for the D5 tie
	nearest globalfeed.NearestCity        // the fuzzy "the <metro> area" resolver
	seen    *seenStore
	warm    atomic.Bool // false until the first cycle seeds quietly (no launch alert storm)
	muted   *atomic.Bool
	radius  *atomic.Int64 // alert-radius filter in miles; 0 = All (global)
	audio   tickerAudio   // the breaking-news sound (the radio deck); nil = silent
	running atomic.Bool   // true while a breaking-news sequence holds the marquee (no overlap)
	done    chan struct{} // closed when run returns, so stopAll can drain the ticker before teardown
}

// tickerAudio is the breaking-news sound the ticker drives (0.12.0): duck the
// radio and sound the tone at the start (returning the tone's length), speak a
// line and return how long it will take (so the marquee holds the event until
// its narration finishes — no overlap), restore at the end. The radio deck
// implements it; nil = no audio (the visual takeover still runs on a fixed
// hold).
type tickerAudio interface {
	startBreaking() time.Duration
	speak(text string) time.Duration
	endBreaking()
}

// startTicker launches the ticker loop; it stops with ctx. onNew (nil = no
// audio) sounds the tone + narration for a genuinely new event.
func startTicker(ctx context.Context, p *tea.Program, client *httpx.Client, idx *geodata.Index, watch func() []snapshot.LocationRef, muted *atomic.Bool, radius *atomic.Int64, audio tickerAudio) *tickerDeck {
	t := &tickerDeck{
		send:    p.Send,
		sources: []globalfeed.Source{globalfeed.NewUSGS(client, ""), globalfeed.NewNHC(client, ""), globalfeed.NewNWS(client, "")},
		watch:   watch,
		nearest: nearestMetro(idx),
		seen:    loadSeen(userCacheSubdir("ticker"), tickerSeenWindow),
		muted:   muted,
		radius:  radius,
		audio:   audio,
		done:    make(chan struct{}),
	}
	go t.run(ctx)
	return t
}

// stop waits for the ticker's run loop to return. RunDashboard cancels the
// shared context first (its deferred cancel), so this only drains: it blocks
// until the in-flight cycle's feed fetches and the seen-store save have
// finished, leaving the cache directory quiescent before teardown. Without it a
// headless run's t.TempDir cleanup raced the ticker's still-in-flight disk
// write ("directory not empty" under -race on Linux). It must be called with no
// lock the ticker acquires — the cycle's watch tie takes livePipelines.mu — so
// stopAll waits outside that lock.
func (t *tickerDeck) stop() { <-t.done }

// run is the ticker's background event loop: one cycle at once, then one every
// tickerEvery, until ctx ends (the same shape as the scheduler's tier loops).
func (t *tickerDeck) run(ctx context.Context) {
	defer close(t.done) // signal stopAll the last cycle's disk writes are settled
	tk := time.NewTicker(tickerEvery)
	rotate := time.NewTicker(tickerRotate)
	defer tk.Stop()
	defer rotate.Stop()
	t.cycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			t.cycle(ctx)
		case <-rotate.C:
			t.send(tty.TickerAdvanceMsg{}) // the 90s lane rotation; the tty skips it when ≤1 lane is active
		}
	}
}

// cycle fetches every source, ties and stacks the events, publishes the
// marquee, and announces the new ones (after seeding quietly on the first
// cycle so a launch never alert-storms).
func (t *tickerDeck) cycle(ctx context.Context) {
	var events []globalfeed.Event
	for _, s := range t.sources {
		evs, err := s.Fetch(ctx)
		if err != nil {
			continue // a feed outage leaves its events absent this cycle; the others still show
		}
		events = append(events, evs...)
	}
	now := time.Now()
	events = globalfeed.Active(events, now) // drop alerts past their active window (#2)
	watch := t.watch()
	// The alert-radius filter (HUM LEAD): "Filtered to N Mi of my location"
	// scopes the whole ticker to within N miles of the default location; 0 = All.
	// Filtered but no location set → show nothing, never silently fall back to
	// the global stack the UI says is scoped away (red-team 0.12.0 P4 F7).
	if r := int(t.radius.Load()); r > 0 {
		if len(watch) > 0 {
			events = globalfeed.Within(events, watch[0].Lat, watch[0].Lon, float64(r))
		} else {
			events = nil
		}
	}
	for i := range events {
		events[i].Location = globalfeed.Locate(events[i].HasPoint, events[i].Lat, events[i].Lon, events[i].Place, watch, t.nearest)
	}
	// A superseded alert is kept in `events` (so it is seen-marked below and can
	// never resurface as "new" if its replacement drops first — P4 delta A1),
	// but it is excluded from the display and the new-event detection.
	display := events[:0:0]
	for _, e := range events {
		if !e.Superseded {
			display = append(display, e)
		}
	}
	stack, fresh := globalfeed.Merge(display, t.seen.set())
	t.send(tty.TickerMsg{Items: itemsOf(stack)})

	// Mark EVERY active event seen (incl. those past the stack cap and the
	// superseded ones): an event dropped from the display must not resurface
	// later as "new" and fire a breaking takeover for a stale alert (P4 F4/A1).
	if !t.warm.Swap(true) {
		t.seen.mark(events, now) // the first cycle seeds current events quietly — no launch storm
		t.seen.save()
		return
	}
	t.seen.mark(events, now)
	t.seen.save()
	if len(fresh) > 0 {
		// The breaking-news takeover (HUM LEAD 2026-08-27): one sequence at a
		// time; a second burst arriving mid-sequence is still seen-marked and
		// shows in normal rotation, never a doubled takeover.
		if t.running.CompareAndSwap(false, true) {
			go func() { defer t.running.Store(false); t.breaking(ctx, fresh) }()
		}
	}
}

// breakingHold is the fallback centre-hold per event when there is no audio to
// pace it (muted, or no voice) — the visual still steps (HUM LEAD 2026-08-27).
const breakingHold = 5 * time.Second

// breakingGap is the 1 s pause between the events of a burst (HUM LEAD script).
const breakingGap = time.Second

// maxBreaking hard-caps how many events a single takeover reads; breakingCap
// bounds it by time. The overflow shows in normal rotation (all seen-marked).
const (
	maxBreaking = 8
	breakingCap = 30 * time.Second
)

// breaking takes the marquee over for the fresh events (HUM LEAD 2026-08-27):
// sorted most-severe first, each shown centred in its lane colour and held
// until its narration finishes, then the next. The tone sounds ONCE up front;
// a single event carries its own "play the broadcast" tail, a burst reads each
// event (1 s apart) then one closing tail. The radio ducks for the whole
// sequence and restores at the end; normal rotation resumes where it was. When
// there is no audio the visual steps on a fixed hold. A burst is bounded by
// breakingCap (time) and maxBreaking (count).
func (t *tickerDeck) breaking(ctx context.Context, fresh []globalfeed.Event) {
	byBreakingPriority(fresh)
	if len(fresh) > maxBreaking {
		fresh = fresh[:maxBreaking]
	}
	burst := len(fresh) > 1
	audible := t.audio != nil && !t.muted.Load()
	if audible {
		toneDur := t.audio.startBreaking() // duck + tone
		defer t.audio.endBreaking()
		if !sleepCtx(ctx, toneDur) { // let the tone finish before the first line
			return
		}
	}
	if !t.readBreaking(ctx, fresh, burst, audible) {
		return // context ended mid-sequence — the deferred restore still fires
	}
	if audible && burst {
		if !sleepCtx(ctx, t.audio.speak(burstClosing)) { // the one closing tail
			return
		}
	}
	t.send(tty.TickerBreakingDoneMsg{}) // resume normal rotation
}

// readBreaking steps the marquee through the events: each is shown centred and
// held until its narration finishes (or a fixed hold when silent), a burst
// pausing breakingGap between them, until the queue or breakingCap runs out.
// Returns false if the context ended mid-sequence.
func (t *tickerDeck) readBreaking(ctx context.Context, fresh []globalfeed.Event, burst, audible bool) bool {
	var elapsed time.Duration
	for i, e := range fresh {
		t.send(tty.TickerBreakingMsg{Item: breakingItem(e)})
		hold := breakingHold
		if audible {
			if d := t.audio.speak(breakingLine(e, burst)); d > 0 { // render + play; hold until it finishes
				hold = d
			}
			// d == 0 means no voice was available at runtime — keep the fixed
			// hold so the event is still readable, don't blit past it (P4 F10).
		}
		if !sleepCtx(ctx, hold) {
			return false
		}
		elapsed += hold
		if burst && i < len(fresh)-1 {
			if !sleepCtx(ctx, breakingGap) {
				return false
			}
			elapsed += breakingGap
		}
		if elapsed >= breakingCap { // the burst has run its budget — stop reading more
			break
		}
	}
	return true
}

// breakingLine is the line to speak for one event: a single event carries its
// own broadcast tail; a burst event's line has none (the tail comes once).
func breakingLine(e globalfeed.Event, burst bool) string {
	if burst {
		return eventNarration(e)
	}
	return alertNarration(e)
}

// sleepCtx waits d, or returns false at once if the context ends.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// byBreakingPriority orders the queue most-severe first, then most-recent — the
// read order of the takeover.
func byBreakingPriority(evs []globalfeed.Event) {
	sort.SliceStable(evs, func(i, j int) bool {
		if evs[i].Severity != evs[j].Severity {
			return evs[i].Severity > evs[j].Severity
		}
		return evs[i].At.After(evs[j].At)
	})
}

// breakingItem is the single marquee item shown for one breaking event.
func breakingItem(e globalfeed.Event) tty.TickerItem { return itemsOf([]globalfeed.Event{e})[0] }

// itemsOf composes the marquee tape item for each event: its lane (category),
// the compact tape line ("<Type> · <Location>  <verb> <t> · expires <t>"), and
// the severity for ordering within the lane.
func itemsOf(evs []globalfeed.Event) []tty.TickerItem {
	out := make([]tty.TickerItem, 0, len(evs))
	for _, e := range evs {
		out = append(out, tty.TickerItem{
			ID:       e.ID,
			Category: tickerCategory(e),
			Text:     tapeText(e),
			Severity: tty.TickerSeverity(e.Severity),
		})
	}
	return out
}

// tickerCategory maps an event to its marquee lane (HUM LEAD 2026-08-27):
// quakes and tropical cyclones to their own lanes; an NWS product splits by
// Watch vs Warning.
func tickerCategory(e globalfeed.Event) tty.TickerCategory {
	switch e.Class {
	case globalfeed.ClassQuake:
		return tty.CatQuake
	case globalfeed.ClassTropical:
		return tty.CatTropical
	default: // ClassSevereWx
		if strings.Contains(e.Type, "Watch") {
			return tty.CatWatch
		}
		return tty.CatWarning
	}
}

// tapeText is one alert on the ticker tape: the specific type and tied
// location, when it happened (the class verb — declared/recorded/reported), and
// its active window's end when it has one (HUM LEAD 2026-08-27, #1). The lane
// label already names the category, so the line names the specific alert.
func tapeText(e globalfeed.Event) string {
	s := e.Type + " · " + e.Location + "  " + e.Verb() + " " + clock(e.At)
	if !e.Until.IsZero() {
		s += " · expires " + clock(e.Until)
	}
	// Feed text reaches the terminal here — strip any escape/control sequences a
	// hostile or compromised feed could smuggle in (OSC-52 clipboard, title
	// spoof, CSI frame corruption), the same defence the snapshot path applies
	// (red-team 0.12.0 P4 F1 — S-F6).
	return render.Plain(s)
}

// eventNarration is one event's spoken line: the sentence, when it happened,
// and (for an alert with a window) until when — no tail (HUM LEAD script).
// ExpandStates in AlertNarration reads "VA" as "Virginia".
func eventNarration(e globalfeed.Event) string {
	s := e.Sentence() + " at " + clock(e.At)
	if !e.Until.IsZero() {
		s += " until " + clock(e.Until)
	}
	return s
}

// alertNarration is a SINGLE event's full narration: its line plus the radio
// tail directing the listener to that location's broadcast.
func alertNarration(e globalfeed.Event) string {
	return eventNarration(e) + ". Play the Watchpost Radio Broadcast of that location for more details"
}

// burstClosing is the one tail after a multi-event burst — the events are read
// in turn, then this points the listener at the broadcasts (HUM LEAD script).
const burstClosing = "For more information regarding any of these events, play the Watchpost Radio broadcast at that location for details."

// clock renders a time in the local zone as a 12-hour clock ("3:42 PM"), the
// same wall-clock convention the rest of the dashboard uses.
func clock(t time.Time) string { return t.Local().Format("3:04 PM") }

// metroCapKm bounds the fuzzy tie: past this the nearest US metro is not a
// meaningful description, so an event out at sea or overseas isn't asserted to
// be near a US city (red-team 0.12.0 P4 F15).
const metroCapKm = 400

// nearestMetro resolves the nearest major US metro to a point (for the fuzzy
// "the <metro> area" tie); it searches the top cities so the scan is bounded.
func nearestMetro(idx *geodata.Index) globalfeed.NearestCity {
	if idx == nil {
		return nil
	}
	top := idx.TopUS(300)
	return func(lat, lon float64) string {
		best, bestKm := "", 0.0
		for _, c := range top {
			km := geo.HaversineKM(lat, lon, c.Lat, c.Lon)
			if best == "" || km < bestKm {
				best, bestKm = c.Name, km
			}
		}
		if bestKm > metroCapKm {
			return "" // nothing close enough to name
		}
		return best
	}
}

// seenStore persists the ids the ticker has already announced (id -> first
// seen), pruned to a window, so a restart does not re-announce a still-active
// event. Bounded by the window and the feeds' own sizes (P10).
type seenStore struct {
	mu     sync.Mutex
	path   string
	window time.Duration
	ids    map[string]time.Time
}

func loadSeen(dir string, window time.Duration) *seenStore {
	s := &seenStore{path: filepath.Join(dir, "seen.json"), window: window, ids: map[string]time.Time{}}
	if b, err := os.ReadFile(s.path); err == nil {
		var raw map[string]time.Time
		if json.Unmarshal(b, &raw) == nil {
			cutoff := time.Now().Add(-window)
			for id, at := range raw {
				if at.After(cutoff) { // drop the stale on load
					s.ids[id] = at
				}
			}
		}
	}
	return s
}

// set is a snapshot of the seen ids as a presence set (for Merge).
func (s *seenStore) set() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.ids))
	for id := range s.ids {
		out[id] = true
	}
	return out
}

// mark records events as seen at t and prunes the stale.
func (s *seenStore) mark(evs []globalfeed.Event, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range evs {
		if _, ok := s.ids[e.ID]; !ok {
			s.ids[e.ID] = t
		}
	}
	cutoff := t.Add(-s.window)
	for id, at := range s.ids {
		if at.Before(cutoff) {
			delete(s.ids, id)
		}
	}
}

func (s *seenStore) save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, err := json.Marshal(s.ids); err == nil {
		_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
		_ = os.WriteFile(s.path, b, 0o644)
	}
}
