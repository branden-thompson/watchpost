package app

// ticker.go — THE PRODUCER (MVS-D-77, S-7): what arrived, and who is told.
//
// It fetches the national feeds on a 2-minute cadence, scopes them to the
// listener's radius, publishes the marquee's tape, and hands whatever is NEW to
// the Director as arrivals. It owns the alert store and the seen store's use;
// it does NOT own words, and it does not own pacing.
//
// IT HAD NO HEADER AT ALL, and held five separable concerns across 978 lines
// (red team 2026-09-05, Junior-Dev 10). Split on 2026-09-06, purely — nothing
// changed but which file a declaration sits in, and TestDeclarationSetUnchanged
// is the guard that says so:
//
//	burst_words.go  the words a takeover says and the tape's sentences (the
//	                Composer's helpers, which is why they are not here)
//	seen_store.go   the ids already announced, persisted across restarts
//	metro.go        the fuzzy "the <metro> area" tie
//	read_script.go  MVS-D-72's four pause constants, which now sit with the
//	                Reader that reads them and the comment that claims them
//
// The TAPE stays here — itemsOf, laneItems, tapeItems, tickerCategory — because
// the Producer publishes the marquee, so those rows are its own output.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// tickerMuteState is the shared "the listener said do not speak to me" flag.
//
// IT STARTS FALSE, AND THERE IS NO SEED (red team 2026-09-05, C-1). It used to
// be seeded from cfg.TickerMuted, and that was the release's worst defect:
// ticker_muted is a BACK-COMPAT MIRROR that config.Save derives from the tone
// mode so a 0.13.0 binary reading the file still mutes its ticker. It is not
// this binary's state. Reading it back as runtime state meant muting one tone
// class in Settings silenced every spoken alert from the next launch on — and
// nothing could clear it, because MVS-D-48 retired the [M] toggle when the key
// became a deep link into Settings. A listener who found the EAS tone startling
// at night lost the words too, permanently, and their own attempt to undo it
// appeared to do nothing.
//
// THE PARAMETER IS GONE, NOT DEFAULTED (P10-07). A seed nothing may vary is one
// more thing that can be set wrongly, and this one was.
//
// NOTHING SETS IT TODAY, and that is stated rather than implied (AP-DEAD-01,
// the way Power.OffAir is disclosed). MVS-D-26 gives the tone mute the tones
// alone; MVS-D-48 took away the one-key panic mute. So the flag is constant
// false in 0.14.0 and the two gates it drives — the producer's (startTakeover,
// MVS-D-78) and the executor's (runCue) — are correct and pinned but not
// currently reachable. They are the seam a Broadcaster "silence the station"
// control sets; they are deliberately kept, because deriving them again later
// is how this defect was born.
func tickerMuteState() *atomic.Bool {
	flag := &atomic.Bool{}
	return flag
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

// tickerRotate is how long each category lane holds the marquee before it
// rotates to the next non-empty lane (HUM LEAD 2026-08-27, #6).
const tickerRotate = 90 * time.Second

// tickerDeck runs the global event ticker's separate pipeline (0.12.0): fetch
// the three global feeds on a cadence, tie each event to a representative
// location, stack them, publish the marquee, and detect genuinely NEW events
// (the P3 tone/narration will sound those unless muted).
type tickerDeck struct {
	send      func(tea.Msg) // publishes to the dashboard (p.Send in production; a capture in tests)
	sources   []globalfeed.Source
	watch     func() []snapshot.LocationRef // the current watchlist, for the D5 tie
	nearest   globalfeed.NearestCity        // the fuzzy "the <metro> area" resolver
	seen      *seenStore
	warm      atomic.Bool // false until the first cycle seeds quietly (no launch alert storm)
	muted     *atomic.Bool
	radius    *atomic.Int64   // alert-radius filter in miles; 0 = All (global)
	clockPref *atomic.Int32   // how times are written (render.Clock) — Settings changes it live
	voice     *director       // the narration arbiter (app/director.go); a silent one when there is no audio
	mc        *mastercontrol  // the band's and the bed's ONE owner, shared with the arbiter (T2.3)
	inject    injectQueue     // F-21b: a real queue in a debug build, an empty struct in a release one
	scripts   *script.Library // the spoken lines (domains/radio/script); nil = the built-in scripts
	done      chan struct{}   // closed when run returns, so stopAll can drain the ticker before teardown
	severe    *severeDeck     // 0.13.0: the severe-events index (nil = no window, as in the older tests)

	// alerts is the producer's record of what each arrival IS, asked for by the
	// executors when they compose, cue and mark (T3.10b). The card carries only
	// the ids; this is the other half.
	alerts *alertStore

	// mu guards emit, which is wired after the deck is built: the schedule needs
	// the deck to exist before it can be started.
	mu   sync.Mutex
	emit func(lineup.Event) // nil until the schedule is wired
}

// clock is the listener's clock, or the 12-hour default when nothing set one
// (the older tests, which build a deck by hand).
func (t *tickerDeck) clock() render.Clock {
	return clockFrom(t.clockPref)
}

// tickerAudio is the breaking-news sound the ticker drives (0.12.0): duck the
// radio and sound the tone at the start (returning the tone's length), speak a
// line and return how long it will take (so the marquee holds the event until
// its narration finishes — no overlap), restore at the end. The radio deck
// implements it; nil = no audio (the visual takeover still runs on a fixed
// hold).
// startTicker launches the ticker loop; it stops with ctx. nar (nil = a
// silent director) sounds the tone + narration for a genuinely new event
// through the director, so a takeover pre-empts an event read and never
// overlaps one (app/director.go).
func startTicker(ctx context.Context, p *tea.Program, client *httpx.Client, idx *geodata.Index, watch func() []snapshot.LocationRef, prefs tickerPrefs, nar *director, scripts *script.Library, severe *severeDeck) *tickerDeck {
	// ONE INSTANCE, AND NONE BUILT TO BE THROWN AWAY. An earlier shape
	// constructed an effector unconditionally and discarded it whenever an
	// arbiter was supplied — which is every production path. The arbiter's
	// effector IS the band's owner; the fallback exists only so a nil arbiter
	// cannot take the ticker down, and it builds the one instance too.
	if nar == nil {
		nar = newDirector(nil, newMastercontrol(nil, p.Send))
	}
	mc := nar.mc
	t := &tickerDeck{
		severe:  severe,
		send:    p.Send,
		mc:      mc,
		sources: []globalfeed.Source{globalfeed.NewUSGS(client, ""), globalfeed.NewNHC(client, ""), globalfeed.NewNWS(client, "")},
		watch:   watch,
		nearest: nearestMetro(idx),
		seen:    loadSeen(userCacheSubdir("ticker"), tickerSeenWindow),
		alerts:  newAlertStore(),
		muted:   prefs.muted,
		radius:  prefs.radius, clockPref: prefs.clock,
		voice:   nar,
		scripts: scripts,
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
func (t *tickerDeck) stop() {
	<-t.done
}

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
	health := make([]SourceHealth, 0, len(t.sources))
	fetchedAt := time.Now()
	for _, s := range t.sources {
		evs, err := s.Fetch(ctx)
		if err != nil {
			health = append(health, SourceHealth{Name: s.Name(), OK: false}) // a dead source: stated in the window, never hidden
			continue                                                         // a feed outage leaves its events absent this cycle; the others still show
		}
		health = append(health, SourceHealth{Name: s.Name(), OK: true, FetchedAt: fetchedAt})
		events = append(events, evs...)
	}
	// INJECTED EVENTS JOIN HERE, as if a source had returned them (F-21b), so
	// they cross every stage a real alert does. Always nil in a release build —
	// the capability is absent from the binary rather than disabled in it.
	events = append(events, t.takeInjected()...)
	now := time.Now()
	events = globalfeed.Active(events, now) // drop alerts past their active window (#2)
	watch := t.watch()
	// Tie EVERY event to its representative location before any filter
	// (0.13.0): the severe-events window lists the pre-radius set and needs
	// the labels; the tape's radius filter applies below.
	for i := range events {
		events[i].Location = globalfeed.Locate(events[i].HasPoint, events[i].Lat, events[i].Lon, events[i].Place, watch, t.nearest)
	}
	if t.severe != nil {
		t.severe.SetFeed(events, health) // the window's half of the index — its own copy (SetFeed clones)
	}
	events = t.scopeToRadius(events, watch)
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
	t.send(tty.TickerMsg{Items: t.tapeItems(stack)})

	// A SUPERSEDED OR ALREADY-SEEN EVENT IS SETTLED HERE, so an alert dropped
	// from the display cannot resurface later as "new" and fire a takeover for
	// something stale (P4 F4/A1). The fresh ones are not settled here: the
	// takeover marks each as it reads it, and whatever it never reached stays
	// new deliberately — see startTakeover.
	// A GENUINE FIRST RUN SEEDS QUIETLY. A RELAUNCH DOES NOT (C-4).
	//
	// The seed exists so a new listener is not met with every active hazard in
	// the country at once. It used to run on the first cycle of EVERY launch,
	// and that swallowed the one set a returning listener has not heard: the
	// persistent store already stops a restart re-announcing what was announced
	// before, so the only thing the blanket seed added was suppression of the
	// alerts that arrived WHILE THE APP WAS CLOSED. Lid shut at 2pm, tornado
	// warning at 2:40, relaunched at 3:00 — on the tape, never spoken, and
	// filtered by unread for ever after.
	//
	// An empty store is the honest test for "this listener is new here": a
	// store whose entries have all aged past the window is a fresh start too,
	// which is the same thing said a different way. The catch-up case is
	// already bounded by the rail — Max admits the worst few and the tail is
	// DIVERTED with its count spoken (DR-14), so a listener returning to
	// seventeen live hazards is told about the worst and pointed at the rest.
	//
	// Swap is the LEFT operand deliberately: warm must be set on the first
	// cycle whether or not anything is seeded.
	if !t.warm.Swap(true) && t.seen.empty() {
		t.seen.mark(events, now)
		t.seen.save()
		return
	}
	t.startTakeover(fresh)
	t.seen.mark(notNew(events, fresh), now)
	t.seen.save()
}

// startTakeover hands a burst to the DIRECTOR (T3.10b).
//
// THIS IS THE PRODUCER, AND ONLY THE PRODUCER. It decides what has arrived and
// what each arrival IS; the Director decides which of them are read and in what
// order, the Composer decides what is said, and the Reader decides how it
// sounds. This function used to be all four.
//
// WHAT WENT WITH THE SWAP, and why none of it is a loss:
//
//   - The `running` slot. "Only one takeover at a time" is the schedule's
//     invariant now — one card holds the air — and a burst arriving while
//     another reads JOINS THE RAIL instead of being dropped (DR-3). The old
//     behaviour left a hazard unread until the next cycle rediscovered it, and
//     MVS-D-56 named that as the reason for the lineup in the first place.
//   - The `breakers` wait set. The read runs on the pump's worker now, and the
//     pump drains every dispatched effect before it stops (R5-B-07 still holds,
//     one layer down).
//   - `railBurst`. The Director plans from the arrivals, so the app no longer
//     plans at all — one planner rather than two agreeing by luck.
func (t *tickerDeck) startTakeover(fresh []globalfeed.Event) {
	// STANDBY HOLDS THE BURST; IT DOES NOT SPEND IT (MVS-D-78).
	//
	// [M] used to let the takeover run inaudibly: it cued the band, held, and
	// MARKED EACH ALERT READ — so a tornado warning arriving while muted was
	// consumed in silence and never sounded, even on unmuting a minute later.
	// The visual channel still showed it, so nothing was hidden; the audio
	// channel simply swallowed a hazard.
	//
	// IT RETURNS BEFORE ANYTHING IS SENT, which is the same rule one layer up
	// from where it used to sit. A muted burst must not reach the Director at
	// all: admitted to the rail it would be a promise to read (DR-3), and the
	// executors' own mute check would then decline it every time it came round.
	// The tick has already sent the severe index and the ticker tape — both
	// unconditional, both above this call — so nothing is hidden; the burst
	// simply stays new and reads when the listener comes back.
	if t.muted.Load() {
		return
	}
	if len(fresh) == 0 {
		return
	}
	// ALREADY READ ALOUD IS NOT NEW. The executors mark each alert as its line
	// is said, so this is what keeps a burst from being offered twice.
	fresh = unread(fresh, t.seen.set())
	if len(fresh) == 0 {
		return
	}
	// THE RECORDS BEFORE THE ARRIVALS. The Director can describe a build in the
	// same step it receives these, and the executor asks this store for what the
	// card's refs mean — so a store written afterwards would be a race with a
	// build already in flight.
	t.alerts.note(fresh, time.Now())
	t.tell(lineup.Arrived{Arrivals: arrivalsOf(fresh), Fence: t.fence()})
}

// tell hands an event to the Director, or drops it when there is no schedule —
// the older tests build a deck with no station around it.
func (t *tickerDeck) tell(ev lineup.Event) {
	tellUnder(&t.mu, &t.emit, ev)
}

// unread is the events of a burst that no takeover has read aloud yet.
func unread(burst []globalfeed.Event, seen map[string]bool) []globalfeed.Event {
	out := burst[:0:0]
	for _, e := range burst {
		if !seen[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

// arrivalsOf is the producer's translation: domain events in, the Director's
// domain-free arrivals out (DR-1).
//
// THE CHOOSING IS NOT HERE, AND NO LONGER IS ANYWHERE IN THIS FILE (T3.10b).
// This used to plan the burst itself and hand the chosen events to a takeover
// it also ran — a second planner beside the Director's, agreeing with it only
// as long as both were passed the same settings. Now it states what arrived and
// the Director decides the rest, which is the whole point of ONE carrier of the
// ladder (D-1).
func arrivalsOf(fresh []globalfeed.Event) []lineup.Arrival {
	out := make([]lineup.Arrival, 0, len(fresh))
	for _, e := range fresh { // bounded by the cycle's fresh events (P10-02)
		out = append(out, lineup.Arrival{
			ID:       e.ID,
			Category: globalfeed.LaneOf(e),
			Headline: tapeHead(e),
			Subject:  subjectOf(e),
			Severity: int(e.Severity),
			At:       e.At,
			Lat:      e.Lat,
			Lon:      e.Lon,
			HasPoint: e.HasPoint,
			Test:     e.Fabricated, // it takes no real hazard's place (FR-4.4)
			// THE SIGNIFICANCE REACH TRAVELS WITH THE ARRIVAL (BD-6, C-3).
			// Without it Fence.Admits measured every disaster against zero
			// miles of reach, so the ruling's own admit-case — an M7.5 in Los
			// Angeles, ~120 mi, to a listener with a 50-mile radius — was
			// refused by the fence even once the producer stopped dropping it.
			ReachMi: reachMiOf(e),
			Tracked: true, // these events already passed the deck's own scoping
		})
	}
	return out
}

// subjectOf is what an alert is ABOUT, and it is never empty.
//
// THE PLANNER REFUSES A WHOLE BURST OVER ONE BLANK SUBJECT — deliberately, so a
// malformed arrival is caught before the takeover rather than halfway down it.
// That makes this mapping load-bearing: `Location` is the TIED location and an
// alert the locator could not tie has none, so reading it straight would let a
// single untied alert silence every other alert in the batch. The feed's own
// place text is the next best answer, and the hazard type is one a listener can
// still act on.
func subjectOf(e globalfeed.Event) string {
	if e.Location != "" {
		return e.Location
	}
	if e.Place != "" {
		return e.Place
	}
	return e.Type
}

// fence is the listener's service radius as the planner needs it.
//
// It is asked from the SAME radius and origin the deck already scopes with, so
// the ordering rule DR-12 keys off (Fence.InForce) agrees with the scoping that
// chose these events. The events reaching the rail are already inside it, so
// admission here removes nothing further — it decides ORDER.
func (t *tickerDeck) fence() lineup.Fence {
	// A DECK WITHOUT A RADIUS OR A WATCHLIST IS "ALL", not a panic. Both are
	// wired by the pipeline in production; a deck built for one narrow question
	// has neither, and the rail is the one path that would dereference them.
	if t.radius == nil || t.watch == nil {
		return lineup.Fence{}
	}
	r := float64(t.radius.Load())
	if r <= 0 {
		return lineup.Fence{} // All: no radius, and the ladder's unfenced order
	}
	watch := t.watch()
	if len(watch) == 0 {
		return lineup.Fence{}
	}
	return lineup.Fence{RadiusMi: r, Lat: watch[0].Lat, Lon: watch[0].Lon, HasOrigin: true}
}

// notNew is every event whose "is this new?" question this cycle can settle on
// its own — that is, everything except the fresh ones.
//
// A fresh event is settled by the takeover instead, which marks each as it
// reads it. Whatever the takeover never reaches — a burst past its bounds, one
// turned away because a takeover was already on air, a sequence the context cut
// short — is left unmarked, so it is new again next cycle and is announced
// then. Marking it here would be telling the store the listener had been told.
func notNew(events, fresh []globalfeed.Event) []globalfeed.Event {
	if len(fresh) == 0 {
		return events
	}
	isFresh := make(map[string]bool, len(fresh))
	for _, e := range fresh {
		isFresh[e.ID] = true
	}
	out := make([]globalfeed.Event, 0, len(events))
	for _, e := range events {
		if !isFresh[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

// defaultBurstMax is how many alert reads one burst spends with no listener
// setting (read-order-design.md, "the default is 5 alerts").
//
// IT IS THE ONLY BOUND, AND IT APPLIES AT ADMISSION (DR-3). What it replaced
// bounded twice — by count when the burst was chosen and again by TIME while it
// was read — so whichever hazards sorted last were the ones a slow read
// silenced, permanently under sustained arrivals. Two red-team rounds found the
// same defect in that shape and each fix moved the boundary: a per-lane floor,
// then a time budget derived from the count, and a sweep still broke it at
// about 8.1 s a read. The HUM LEAD's ruling was that the ordering is not a
// constant to be tuned but a preference the listener sets, and T4.1 is the
// surface that sets it.
//
// Anything past the Max is DIVERTED, not dropped: the listener is told the
// count, which is why `Plan` states it as a conservation law rather than a
// subtraction.
const defaultBurstMax = 5

// timeRender records how long one render took, under the radio debug flag only.
// A render is the ONLY thing that can block between the tone and the words, so
// its duration is the whole diagnosis of a long gap there.
func timeRender(what string, render func() (clip, bool)) (clip, bool) {
	if !radioDebugOn() {
		return render()
	}
	start := time.Now()
	c, ok := render()
	radioDebugLog(fmt.Sprintf("burst:render:%s=%dms ok=%v", what, time.Since(start).Milliseconds(), ok))
	return c, ok
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
			Head:     tapeHead(e),
			Verb:     e.Verb(),
			At:       e.At,
			Until:    e.Until,
			Severity: tty.TickerSeverity(e.Severity),
			Test:     e.Fabricated, // the band says so (FR-4.4)
		})
	}
	return out
}

// tapeItems is the whole tape: the national feed's stack, plus the two lanes the
// feed cannot fill.
//
// Advisories and Special Weather Statements reach the app only through the
// tracked locations, so they come from the severe index — which this cycle has
// just refreshed — rather than from the stack.
func (t *tickerDeck) tapeItems(stack []globalfeed.Event) []tty.TickerItem {
	items := itemsOf(stack)
	if t.severe == nil {
		return items
	}
	return append(items, laneItems(t.severe.LaneRows())...)
}

// scopeToRadius applies the ALERTS - EVENTS preference to the feed.
//
// "All" is everything. A radius is the events within N miles of the DEFAULT
// location, plus the zone-only alerts the app is already tracking there —
// scopeEvents owns that rule, and the severe window asks it the same question,
// so the two surfaces cannot disagree about one hazard.
//
// Filtered with no default location set shows NOTHING, rather than silently
// falling back to the global stack the UI says is scoped away.
func (t *tickerDeck) scopeToRadius(events []globalfeed.Event, watch []snapshot.LocationRef) []globalfeed.Event {
	// A DECK WITHOUT A RADIUS IS "ALL", not a panic — the same rule fence()
	// states two functions down, and for the same reason: both are wired by the
	// pipeline in production, and a deck built for one narrow question has
	// neither. fence() guarded it and this did not, so the cycle would panic
	// where the fence returned All. A nil dereference in the ticker cycle takes
	// the process with it.
	if t.radius == nil {
		return events
	}
	r := int(t.radius.Load())
	if r <= 0 {
		return events
	}
	if len(watch) == 0 {
		return nil
	}
	var tracked map[string]bool
	if t.severe != nil {
		tracked = t.severe.AlertKeysWithin(watch[0].Lat, watch[0].Lon, float64(r))
	}
	return scopeEvents(events, watch[0].Lat, watch[0].Lon, float64(r), tracked)
}

// laneItems builds the tape items for the location-only categories, in the same
// shape the feed's items take: the product, the location it was tied to, and
// when it was issued.
//
// "issued" rather than the feed classes' declared/recorded/reported: an advisory
// and a statement are issued, which is the word the office itself uses.
func laneItems(rows []severe.Row) []tty.TickerItem {
	out := make([]tty.TickerItem, 0, len(rows))
	for _, r := range rows {
		// EXPLICIT, with no default. A row whose tab has no lane must be
		// dropped, not guessed at: the old form defaulted to Advisory, so a new
		// lane-eligible tab would have been labelled "Advisory" on the band and
		// nothing would have said otherwise.
		var cat tty.TickerCategory
		switch r.Tab {
		case severe.TabEmergency:
			cat = tty.CatEmergency
		case severe.TabStatements:
			cat = tty.CatStatement
		case severe.TabAdvisories:
			cat = tty.CatAdvisory
		default:
			continue
		}
		// Provider text reaches the terminal here too (S-F6), and ONE line: the
		// same defence tapeHead applies, for the same reason.
		out = append(out, tty.TickerItem{
			ID:       r.Key,
			Category: cat,
			Head:     strings.NewReplacer("\n", " ", "\t", " ").Replace(render.Plain(r.Product + " · " + r.Location)),
			Verb:     "issued",
			At:       r.At,
			Until:    r.Until,
			Severity: tty.TickerSeverity(r.Severity),
		})
	}
	return out
}

// tickerCategory is the marquee lane of an event, and it is LaneOf's answer
// unchanged. globalfeed.Lane and tty.TickerCategory are both aliases of
// category.Category, so there is nothing here to translate.
//
// IT USED TO TRANSLATE, AND THAT IS THE BUG (#15). A four-arm switch over the
// lanes, with a default of Warnings, had no arm for LaneEmergency — so an
// Evacuation Immediate was laned Emergency by the feed and relabelled a
// Warning on its way to the band, shown in warning colours beside a
// thunderstorm warning. The window and the read ladder had it right; only the
// screen was wrong. C-2 pinned the ruling where the lane is decided during
// 0.14.0, and this layer, the one that delivers the lane to a listener, was
// never taught it.
//
// A per-lane switch is a producer/consumer pair with nothing checking that the
// consumer knows every value the producer can emit, and a default arm makes
// the gap read as a decision. The identity removes the arms and the default
// together, so a lane added later cannot fall through anything.
func tickerCategory(e globalfeed.Event) tty.TickerCategory { return globalfeed.LaneOf(e) }
