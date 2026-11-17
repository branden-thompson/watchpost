package app

// deck_trace_test.go — the deck's record under random acts (0.19.6 design
// AP-7.2, plan B0.6).
//
// WHY A RECORD AND NOT ASSERTIONS. The deck moves whole into its own package at
// 0.19.6, and every rule it holds is pinned somewhere already. What no pin
// holds is the deck's behaviour as one thing: the events the producers tell the
// Director, what the Director does with them, what the executors make of that,
// and what the console is told. This drives the act surface the entry point
// hands out with seeded random acts and records all of it, so a move that
// changes any of it changes the record. The golden was recorded on code
// identical to e57afeaf.
//
// THE DIRECTOR IS STEPPED HERE, NOT BY THE PUMP. The station is the running
// app's wiring with the goroutines taken out: an act's events are collected,
// the Director steps them in this goroutine, and an effect's events go back to
// it until nothing is left. What still runs on a goroutine of the deck's own
// is waited out before the record is taken; see settle.
//
// GOTCHA: an act's record is only stable if nothing outside the act reaches the
// Director while it is recorded. A new goroutine in the deck that tells the
// Director something must be waited for in settle, or the golden flickers.

import (
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var updateDeckTrace = flag.Bool("update-deck-trace", false, "re-record testdata/deck_trace.golden")

// traceStart is the trace's clock at its first act; each act moves it a second.
var traceStart = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

// tracePlaces are the listener's watchlist, fixed: the station at Bonsall and
// four places around it.
func tracePlaces() []snapshot.LocationRef {
	return []snapshot.LocationRef{
		pinRef("Bonsall, CA", 33.2881, -117.2256),
		pinRef("Oceanside, CA", 33.1959, -117.3795),
		pinRef("Vista, CA", 33.2000, -117.2425),
		pinRef("Rainbow, CA", 33.4103, -117.1478),
		pinRef("Fallbrook, CA", 33.3764, -117.2511),
	}
}

// deckTrace is the station the trace drives: the deck, MasterControl, the
// ticker, the severe index and the event reader, wired as startPipelines and
// startSchedule wire them, over an arbiter whose voice is a script.
type deckTrace struct {
	t     *testing.T
	ctx   context.Context
	lp    *livePipelines
	voice *scriptVoice
	dir   lineup.Director
	x     *executors
	now   time.Time

	mu        sync.Mutex
	pending   []lineup.Event    // what the producers told the Director, not yet stepped
	reading   map[string]bool   // main-track cards the reader has started
	last      map[string]string // the last full text of each repeating kind (same)
	onAir     []lineup.Event    // what their reads came home with, held until the clock moves
	published []tea.Msg
	reports   []string

	// The engine's own status line and the relay server's requests: the two
	// things settle waits on (see there).
	voiceFails atomic.Int64
	mountHits  atomic.Int64
}

// newDeckTrace builds the station. The deck is offline: its NWS provider, its
// relay directories and the bed's mounts point at a server of the test's own
// that answers 404, and its platform is Linux with no program, so a synthesised
// read fails at once for want of a voice rather than starting `say`.
func newDeckTrace(t *testing.T) *deckTrace {
	t.Helper()
	asPlatform(t, "linux")
	dt := &deckTrace{t: t, ctx: context.Background(), voice: &scriptVoice{}, now: traceStart, reading: map[string]bool{}, last: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		dt.mountHits.Add(1)
		http.NotFound(w, nil)
	}))
	t.Cleanup(srv.Close)
	client, base := offlineClient(t, http.NotFoundHandler())
	res, err := stream.NewResolver(stream.NewDirectory(client, base, base))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	eng, err := player.New(&heldOutput{}, "test", func(st player.Status) {
		if st.State == player.Failed && strings.HasPrefix(st.Err, "no voice") {
			dt.voiceFails.Add(1)
		}
	})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	t.Cleanup(eng.Halt)
	deck := &radioDeck{
		nws: nws.New(client, base), resolver: res, engine: eng,
		limiter: synth.NewLimiter(2, synth.ReservedSlots), voiceDir: t.TempDir(),
	}
	tbl, err := stream.LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	places := tracePlaces()
	lp := &livePipelines{idx: indexForTest(t), relayTable: tbl}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	lp.bedRadiusMi = config.DefaultBedRadiusMi
	lp.setWatch(places)
	lp.bedStations = traceBed(lp, tbl, srv.URL)
	lp.bedRefresh = func(context.Context) { lp.setBedStations(traceBed(lp, tbl, srv.URL)) }
	lp.deck = deck
	deck.air = func() bool { return lp.owner.get() != tty.SurfaceBroadcaster } // attachDeck's one line (D-74)
	lp.arbiter = testArbiter(dt.voice, dt.publish)
	lp.arbiter.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	lp.scripts = script.New("")
	deck.composer = synth.Composer{Scripts: lp.scripts}
	lp.severe = newSevereDeck(dt.publish)
	lp.reader = newEventReader(dt.ctx, lp.arbiter, lp.scripts, lp.severe.Row, dt.publish)
	lp.reader.status, lp.reader.restore = deck.overlay, deck.pushStatus
	lp.ticker = traceTicker(t, lp, dt.publish)
	mc := lp.masterControl()
	mc.mu.Lock()
	mc.fence = lp.ticker.fence
	mc.silenceProgramme = deck.stopRead
	mc.playBed, mc.stopBed = lp.playBed, lp.stopBed
	mc.carry = dt.queue
	mc.mu.Unlock()
	lp.ticker.setScope(func() airScope {
		return scopeFor(&lp.owner, func() airScope { return listenerScope(lp.ticker.radius, lp.currentWatch) }, lp.currentStation)
	})
	dt.lp = lp
	dt.x = dt.executors()
	lp.ticker.mu.Lock()
	lp.ticker.emit = dt.queue
	lp.ticker.mu.Unlock()
	deck.mu.Lock()
	deck.emit = dt.queue
	deck.mu.Unlock()
	dt.dir = lineup.New(lineup.Settings{Max: defaultBurstMax, Depth: tty.MainTrackSlots}, traceStart)
	snap := &snapshot.Snapshot{Locations: make([]snapshot.Location, 0, len(places))}
	for _, r := range places { // bounded by the watchlist (P10-02)
		snap.Locations = append(snap.Locations, snapshot.Location{Label: r.Label, Lat: r.Lat, Lon: r.Lon, TZ: r.TZ})
	}
	lp.severe.SetLocations(0, snap)
	lp.ticker.cycle(dt.ctx) // the first cycle seeds, as the running app's does
	dt.drain()
	return dt
}

// traceBed is the bed's relays near the station, each with one mount on the
// test's server: what the resolver answers, supplied (D-117, bedPipelines).
func traceBed(lp *livePipelines, tbl *stream.Table, base string) []stream.Station {
	st := lp.currentStation()
	near := tbl.Within(st.transmitter.Lat, st.transmitter.Lon, config.DefaultBedRadiusMi)
	out := make([]stream.Station, 0, len(near))
	for _, n := range near { // bounded by the table (P10-02)
		out = append(out, stream.Station{Transmitter: n.Transmitter, KM: n.KM,
			Mounts: []stream.Mount{{URL: base + "/" + n.Callsign}}})
	}
	return out
}

// traceTicker is startTicker's deck without its sources or its loop: what it
// carries is what an act injects, and settle runs the cycle the loop would.
func traceTicker(t *testing.T, lp *livePipelines, send func(tea.Msg)) *tickerDeck {
	t.Helper()
	return &tickerDeck{
		rescope: make(chan struct{}, 1),
		severe:  lp.severe,
		send:    send,
		mc:      lp.arbiter.mc,
		watch:   lp.currentWatch,
		nearest: nearestMetro(lp.idx),
		seen:    loadSeen(t.TempDir(), tickerSeenWindow),
		alerts:  newAlertStore(),
		muted:   &atomic.Bool{},
		radius:  &atomic.Int64{},
		voice:   lp.arbiter,
		scripts: lp.scripts,
		done:    make(chan struct{}),
	}
}

// executors are startSchedule's, over the trace's clock and recorders. The
// main track's reader is a script (0.19.6 A-39): the deck's own reader waits
// on an engine the trace never plays. A read is on the air until the clock
// next moves, so what it comes home with is held until then (drain, release):
// a read that came home at once would play the whole line-up inside one act.
func (dt *deckTrace) executors() *executors {
	lp := dt.lp
	tk := lp.ticker
	x := newExecutors(executors{
		voice:     lp.arbiter,
		publish:   dt.publish,
		scripts:   lp.scripts,
		clock:     func() render.Clock { return render.Clock12 },
		now:       func() time.Time { return dt.now },
		mc:        lp.arbiter.mc,
		audible:   func() bool { return !lp.arbiter.silent() },
		muted:     tk.muted.Load,
		alert:     tk.alerts.get,
		mark:      func(id string) { tk.seen.mark([]globalfeed.Event{{ID: id}}, time.Now()) },
		readAloud: tk.seen.has,
		report: func(f lineup.Effect, why string) {
			dt.mu.Lock()
			dt.reports = append(dt.reports, lineup.Describe(f)+": "+why)
			dt.mu.Unlock()
		},
		cutTo:    tuneTo(lp.deck, lp.currentWatch),
		bedLabel: bedLabelOf(lp.currentWatch),
		selected: lp.selectedRelay,
		noteBed:  lp.noteBedCarrying,
		compose:  composeFor(lp.deck, lp.resolvable),
		read: func(_ context.Context, v lineup.Speak) error {
			dt.mu.Lock()
			dt.reports = append(dt.reports, "read("+v.ID+")")
			dt.reading[v.ID] = true
			dt.mu.Unlock()
			return nil
		},
		propose: proposeFrom(lp.producer()),
	})
	if x == nil {
		dt.t.Fatal("the trace's executors were refused; a seam is missing")
	}
	return x
}

func (dt *deckTrace) queue(ev lineup.Event) {
	dt.mu.Lock()
	dt.pending = append(dt.pending, ev)
	dt.mu.Unlock()
}

func (dt *deckTrace) publish(m tea.Msg) {
	dt.mu.Lock()
	dt.published = append(dt.published, m)
	dt.mu.Unlock()
}

// stepped is one event the Director stepped and what came of it.
type stepped struct {
	ev  lineup.Event
	fx  []lineup.Effect
	why []string // the reports each effect's run left, in the effects' order
}

// drain steps everything told to the Director, performs what it describes,
// and feeds the effects' events back, until nothing is left (station.run).
func (dt *deckTrace) drain() []stepped {
	var out []stepped
	for steps := 0; steps < 10_000; steps++ { // bounded; a cap reached fails below (P10-02)
		dt.mu.Lock()
		if len(dt.pending) == 0 {
			dt.mu.Unlock()
			return out
		}
		ev := dt.pending[0]
		dt.pending = dt.pending[1:]
		dt.mu.Unlock()
		var fx []lineup.Effect
		dt.dir, fx = dt.dir.Step(ev)
		s := stepped{ev: ev, fx: fx}
		for _, f := range fx { // bounded by the step (P10-02)
			dt.mu.Lock()
			from := len(dt.reports)
			dt.mu.Unlock()
			back := dt.x.run(dt.ctx, f)
			dt.mu.Lock()
			s.why = append(s.why, strings.Join(dt.reports[from:], "; "))
			if sp, ok := f.(lineup.Speak); ok && dt.reading[sp.ID] {
				delete(dt.reading, sp.ID)
				dt.onAir = append(dt.onAir, back...) // the read is still going out
			} else {
				dt.pending = append(dt.pending, back...)
			}
			dt.mu.Unlock()
		}
		out = append(out, s)
	}
	var tail []string
	for _, st := range out[len(out)-12:] { // the loop's last steps, to name it (P10-02)
		fx := make([]string, 0, len(st.fx))
		for _, f := range st.fx { // bounded by the step (P10-02)
			fx = append(fx, lineup.Describe(f))
		}
		tail = append(tail, traceEvent(dt.t, st.ev)+" => "+strings.Join(fx, ", "))
	}
	dt.t.Fatalf("the station did not settle in 10000 steps: the record would be of a partial sequence; its last steps:\n%s", strings.Join(tail, "\n"))
	return nil
}

// traceAct is one act: its stable key and what it does.
type traceAct struct {
	key string
	do  func()
}

// pick draws the next act. The set is closed (design AP-7.2), and every act's
// key names what it was given, so the record can be read without this file.
func (dt *deckTrace) pick(r *rand.Rand) traceAct {
	lp, deck, mc := dt.lp, dt.lp.deck, dt.lp.masterControl()
	places := tracePlaces()
	modes := []tty.RepeatMode{tty.RepeatOff, tty.RepeatOne, tty.RepeatWatchlist}
	switch r.IntN(24) {
	case 0:
		return traceAct{"on-air", mc.GoOnAir}
	case 1:
		return traceAct{"standby", mc.GoToStandby}
	case 2:
		return traceAct{"cut-bed", func() { mc.CutBed(true) }}
	case 3:
		return traceAct{"cut-back", func() { mc.CutBed(false) }}
	case 4:
		return traceAct{"surface:observer", func() { dt.runCmd(lp.takeTheAir(tty.SurfaceObserver)) }}
	case 5:
		return traceAct{"surface:broadcaster", func() { dt.runCmd(lp.takeTheAir(tty.SurfaceBroadcaster)) }}
	case 6:
		i := r.IntN(len(places))
		return traceAct{"tune:" + strconv.Itoa(i), func() { deck.Tune(places[i]) }}
	case 7:
		return traceAct{"stop", deck.Stop}
	case 8:
		m := modes[r.IntN(len(modes))]
		return traceAct{"repeat:" + strconv.Itoa(int(m)), func() { deck.SetRepeat(m, lp.currentWatch()) }}
	case 9:
		m, name := tty.ModeSynth, "synth"
		if r.IntN(2) == 1 {
			m, name = tty.ModeRelay, "relay"
		}
		return traceAct{"mode:" + name, func() { deck.SetMode(m) }}
	case 10:
		n := r.IntN(101)
		return traceAct{"volume:" + strconv.Itoa(n), func() { deck.SetVolume(n) }}
	case 11, 12:
		sc := debugScenarios()
		k := sc[r.IntN(len(sc))].Key
		return traceAct{"alert:" + k, func() { lp.injectHook()(k) }}
	case 13:
		return traceAct{"lapse", func() { dt.now = dt.now.Add(10 * time.Minute); dt.release(); dt.queue(lineup.Tick{Now: dt.now}) }}
	case 14:
		i, to := r.IntN(tty.MainTrackSlots), r.IntN(tty.MainTrackSlots)
		return traceAct{fmt.Sprintf("move:%d:%d", i, to), func() { lp.moveCard(dt.mainCard(i), to) }}
	case 15:
		i := r.IntN(tty.MainTrackSlots)
		return traceAct{"drop:" + strconv.Itoa(i), func() { lp.dropCard(dt.mainCard(i)) }}
	case 16:
		pool := lp.currentPool()
		i := r.IntN(len(pool))
		return traceAct{"request:" + strconv.Itoa(i), func() { lp.requestCard(pool[i], report.Set(0), 1) }}
	case 17:
		by := []int{-1, 1}[r.IntN(2)]
		return traceAct{"bed-step:" + strconv.Itoa(by), func() { dt.runCmd(lp.stepBedRelay(by)) }}
	case 18:
		return traceAct{"bed-toggle", func() { dt.runCmd(lp.toggleBedRelay()) }}
	case 19:
		i := r.IntN(8)
		return traceAct{"narrate:" + strconv.Itoa(i), func() {
			if k, ok := dt.severeKey(i); ok {
				lp.narrateEvent()(k)
			}
		}}
	case 20:
		return traceAct{"end-read", lp.endEventRead()}
	case 21:
		return traceAct{"station-moved", func() {
			if mc := lp.masterControl(); mc != nil {
				mc.Refence()
			}
			lp.rebed(dt.ctx)
		}}
	}
	return traceAct{"tick", func() { dt.release(); dt.queue(lineup.Tick{Now: dt.now}) }}
}

// release hands the Director what the reads on the air came home with: the
// clock moved, so they have finished.
func (dt *deckTrace) release() {
	dt.mu.Lock()
	dt.pending = append(dt.pending, dt.onAir...)
	dt.onAir = nil
	dt.mu.Unlock()
}

// runCmd runs a command an act returned, inline, as Bubble Tea would on its
// own goroutine; its message, if any, is published.
func (dt *deckTrace) runCmd(c tea.Cmd) {
	if c == nil {
		return
	}
	if m := c(); m != nil {
		dt.publish(m)
	}
}

// mainCard is the i-th card of the main track, or "" past its end: the
// operator's act on a slot with nothing in it is still an act.
func (dt *deckTrace) mainCard(i int) string {
	cards := dt.dir.Lineup().Cards(lineup.MainTrack)
	if i >= len(cards) {
		return ""
	}
	return cards[i].ID
}

// severeKey is the i-th row of the severe window, in key order, wrapping.
func (dt *deckTrace) severeKey(i int) (string, bool) {
	s := dt.lp.severe
	s.mu.Lock()
	keys := make([]string, 0, len(s.rows))
	for k := range s.rows { // bounded by the window (P10-02)
		keys = append(keys, k)
	}
	s.mu.Unlock()
	if len(keys) == 0 {
		return "", false
	}
	slices.Sort(keys)
	return keys[i%len(keys)], true
}

// act performs one act and waits for what it set off on the deck's own
// goroutines, then steps the Director.
func (dt *deckTrace) act(a traceAct) []stepped {
	deck := dt.lp.deck
	urls := tuneListOf(deck)
	retune := retunesOnMode(deck)
	fails, hits := dt.voiceFails.Load(), dt.mountHits.Load()
	a.do()
	dt.settle(a.key, urls, retune, fails, hits)
	return dt.drain()
}

// settle waits for the goroutines an act starts, so the record is the act's
// whole effect and never a part of it:
//
//   - SetMode re-tunes on a goroutine when the monitor has the air and is
//     playing; the tune ends with the engine failing for want of a voice.
//   - A relay started on the engine walks its mounts, each answered 404, and
//     ends Failed.
//   - The event reader reads on a goroutine; its done channel closes when it
//     has finished.
//   - The ticker's loop answers an injected event and a re-scope with a
//     cycle; the trace runs that cycle here.
func (dt *deckTrace) settle(key string, urls []string, retune bool, fails, hits int64) {
	deck := dt.lp.deck
	if strings.HasPrefix(key, "mode:") && retune {
		dt.await("the re-tune", func() bool {
			if dt.voiceFails.Load() <= fails || !deck.tuneMu.TryLock() { // the tune's tail holds it until it returns
				return false
			}
			deck.tuneMu.Unlock()
			return true
		})
	}
	if now := tuneListOf(deck); len(now) > 0 && (len(urls) == 0 || &now[0] != &urls[0]) {
		dt.await("the relay's mounts", func() bool {
			return dt.mountHits.Load() >= hits+int64(len(now)) && deck.engine.Status().State == player.Failed
		})
	}
	r := dt.lp.reader
	r.mu.Lock()
	done := r.done
	r.mu.Unlock()
	if done != nil {
		dt.await("the event read", func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		})
	}
	tk := dt.lp.ticker
	select {
	case <-tk.inject.wake():
		tk.cycle(dt.ctx)
	default:
	}
	select {
	case <-tk.rescope:
		tk.cycle(dt.ctx)
	default:
	}
}

// await polls a condition the deck's goroutine makes true, and fails the trace
// if it does not within the bound (times the race build's slowdown).
func (dt *deckTrace) await(what string, ok func() bool) {
	end := time.Now().Add(10 * time.Second * raceSlowdown)
	for !ok() { // bounded by the deadline (P10-02)
		if time.Now().After(end) {
			dt.t.Fatalf("%s did not finish: the record would be of a partial act", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// tuneListOf is the tune list the deck's last relay start recorded.
func tuneListOf(d *radioDeck) []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.mountURLs
}

// retunesOnMode is SetMode's own condition for re-tuning on a goroutine.
func retunesOnMode(d *radioDeck) bool {
	air := d.monitorHasTheAir()
	d.mu.Lock()
	playing := d.mode != ""
	d.mu.Unlock()
	return air && playing && d.engine.Status().State != player.Stopped
}

// line is one act's record: what was done, what the Director was told and did,
// what the executors made of it, what the console was told, what the voice
// was asked, and where MasterControl stands.
func (dt *deckTrace) line(seed uint64, step int, key string, got []stepped, pubFrom, voiceFrom int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "seed=%d step=%03d act=%s", seed, step, key)
	for _, s := range got { // bounded by the act's steps (P10-02)
		b.WriteString("\n  ev " + dt.same("offered[", traceEvent(dt.t, s.ev)))
		for i, f := range s.fx { // bounded by the step (P10-02)
			d := lineup.Describe(f)
			if d == "" {
				dt.t.Fatalf("an effect the record cannot name: %T", f)
			}
			b.WriteString("\n    fx " + dt.same("publish(", d))
			if s.why[i] != "" {
				b.WriteString(" -> " + s.why[i])
			}
		}
	}
	dt.mu.Lock()
	pub := slices.Clone(dt.published[pubFrom:])
	dt.mu.Unlock()
	for _, m := range pub { // bounded by the act's messages (P10-02)
		b.WriteString("\n  pub " + dt.same("lineup ", traceMsg(dt.t, m)))
	}
	dt.voice.mu.Lock()
	calls := slices.Clone(dt.voice.calls[voiceFrom:])
	dt.voice.mu.Unlock()
	if len(calls) > 0 {
		b.WriteString("\n  voice " + strings.Join(calls, ","))
	}
	mc := dt.lp.masterControl()
	mc.mu.Lock()
	held := mc.held
	mc.mu.Unlock()
	fmt.Fprintf(&b, "\n  mc power=%s air=%s held=%t given-way=%t", dt.dir.Power(), dt.dir.Air(), held, mc.givenWay())
	return b.String()
}

// same is text of a repeating kind, said in full only when it differs from the
// last of that kind: the proposals and the line-up are published after almost
// every step, and in full each time they made the golden 1.5 MB. Text of any
// other kind is returned as it is.
func (dt *deckTrace) same(kind, text string) string {
	if !strings.HasPrefix(text, kind) {
		return text
	}
	if dt.last[kind] == text {
		return strings.TrimRight(kind, "[( ") + " (same)"
	}
	dt.last[kind] = text
	return text
}

// traceEvent names an event and its fields. The set is closed: an event the
// record cannot name fails the trace, so a new one is added here on purpose.
func traceEvent(t *testing.T, ev lineup.Event) string {
	switch v := ev.(type) {
	case lineup.Arrived:
		ids := make([]string, 0, len(v.Arrivals))
		for _, a := range v.Arrivals { // bounded by the burst (P10-02)
			ids = append(ids, a.ID+"/"+a.Headline)
		}
		return fmt.Sprintf("arrived[%s] fence=%s", strings.Join(ids, " "), traceFence(v.Fence))
	case lineup.Tick:
		return "tick+" + v.Now.Sub(traceStart).String()
	case lineup.Built:
		return fmt.Sprintf("built(%s parts=%d contents=%d)", v.ID, len(v.Script.Parts), len(v.Contents))
	case lineup.Finished:
		return "finished(" + v.ID + ")"
	case lineup.Failed:
		return fmt.Sprintf("failed(%s routed=%t): %s", v.ID, v.Routed, v.Reason)
	case lineup.Powered:
		return "powered(" + v.To.String() + ")"
	case lineup.Aired:
		return "aired(" + v.To.String() + ") fence=" + traceFence(v.Fence)
	case lineup.Refenced:
		return "refenced fence=" + traceFence(v.Fence)
	case lineup.Monitored:
		return fmt.Sprintf("monitored(running=%t)", v.Running)
	case lineup.Tuned:
		return fmt.Sprintf("tuned(%s live=%t)", v.Ref, v.Live)
	case lineup.Programme:
		return fmt.Sprintf("programme([%s] dwell=%s)", strings.Join(v.Watchlist, " "), v.Dwell)
	case lineup.Ended:
		return "ended"
	case lineup.CutOver:
		return fmt.Sprintf("cut-over(bed=%t)", v.ToBed)
	case lineup.Moved:
		return fmt.Sprintf("moved(%s to=%d)", v.ID, v.To)
	case lineup.Dropped:
		return "dropped(" + v.ID + ")"
	case lineup.Requested:
		return fmt.Sprintf("requested(%s to=%d)", v.Card.ID, v.To)
	case lineup.NeedsRead:
		return "needs-read(" + v.Ref + " " + v.Headline + ")"
	case lineup.Offered:
		ids := make([]string, 0, len(v.Proposals))
		for _, p := range v.Proposals { // bounded by the offer (P10-02)
			ids = append(ids, fmt.Sprint(p))
		}
		return "offered[" + strings.Join(ids, " ") + "]"
	}
	t.Fatalf("an event the record cannot name: %T", ev)
	return ""
}

func traceFence(f lineup.Fence) string {
	if !f.InForce() {
		return "all"
	}
	return fmt.Sprintf("%.4f,%.4f/%.0fmi", f.Lat, f.Lon, f.RadiusMi)
}

// traceMsg names a published message and its fields, with no times and no
// pointers. The table is closed, as traceEvent's is.
func traceMsg(t *testing.T, m tea.Msg) string {
	switch v := m.(type) {
	case tty.LineupMsg:
		return "lineup " + v.Lineup.Trace()
	case tty.StationMsg:
		return "station power=" + v.Power.String()
	case tty.StationFaultMsg:
		return fmt.Sprintf("fault run=%d %s", v.Run, v.Reason)
	case tty.BedMsg:
		return fmt.Sprintf("bed relay=%q carrying=%t", v.Relay, v.Carrying)
	case tty.TickerMsg:
		items := make([]string, 0, len(v.Items))
		for _, it := range v.Items { // bounded by the tape (P10-02)
			items = append(items, it.ID+"/"+it.Head)
		}
		return "tape[" + strings.Join(items, " ") + "]"
	case tty.TickerBreakingMsg:
		return "band " + v.Item.ID + "/" + v.Item.Head
	case tty.TickerBreakingDoneMsg:
		return "band-done"
	case tty.TickerAdvanceMsg:
		return "tape-advance"
	case tty.SevereReadingMsg:
		return fmt.Sprintf("severe-reading key=%q paused=%t", v.Key, v.Paused)
	case tty.SevereMsg:
		keys := make([]string, 0, len(v.Rows))
		for _, r := range v.Rows { // bounded by the window (P10-02)
			keys = append(keys, r.Key)
		}
		return "severe[" + strings.Join(keys, " ") + "]"
	}
	t.Fatalf("a message the record cannot name: %T", m)
	return ""
}

// injectedID is a fabricated event's id, which carries the wall clock; the
// record names each by its order of first appearance instead.
var injectedID = regexp.MustCompile(`watchpost-injected-([a-z0-9-]+?)-(\d{6,})-(\d+)`)

// stableKeys rewrites every fabricated id in the record to its stable key.
func stableKeys(rec string) string {
	seen := map[string]int{}
	return injectedID.ReplaceAllStringFunc(rec, func(id string) string {
		m := injectedID.FindStringSubmatch(id)
		n, ok := seen[m[2]]
		if !ok {
			n = len(seen) + 1
			seen[m[2]] = n
		}
		return fmt.Sprintf("inj:%s#%d.%s", m[1], n, m[3])
	})
}

// poolKeys names each place of the station's pool by its index, p0 first, and
// is the record's legend: a card's id is its place's coordinates, which said in
// full were the largest part of the golden.
func poolKeys(pool []snapshot.LocationRef) (*strings.Replacer, string) {
	pairs := make([]string, 0, 2*len(pool))
	legend := make([]string, 0, len(pool))
	for i, r := range pool { // bounded by the pool (P10-02)
		k := string(snapshot.Key(r))
		pairs = append(pairs, k, "p"+strconv.Itoa(i))
		legend = append(legend, fmt.Sprintf("p%d=%s(%s)", i, r.Label, k))
	}
	return strings.NewReplacer(pairs...), "pool " + strings.Join(legend, " ")
}

// TestTheDeckIsTheSameUnderRandomActs is the deck's record under four fixed
// seeds of 150 acts each (0.19.6 design AP-7.2). It equals the golden byte for
// byte; a move commit never re-records it.
func TestTheDeckIsTheSameUnderRandomActs(t *testing.T) {
	var all strings.Builder
	var names *strings.Replacer
	var legend string
	for _, seed := range []uint64{1, 7, 42, 1066} { // the four seeds (P10-02)
		dt := newDeckTrace(t)
		if names == nil {
			names, legend = poolKeys(dt.lp.currentPool())
		}
		r := rand.New(rand.NewPCG(seed, seed))
		for step := range 150 {
			dt.now = traceStart.Add(time.Duration(step) * time.Second)
			a := dt.pick(r)
			dt.mu.Lock()
			pubFrom := len(dt.published)
			dt.mu.Unlock()
			dt.voice.mu.Lock()
			voiceFrom := len(dt.voice.calls)
			dt.voice.mu.Unlock()
			got := dt.act(a)
			all.WriteString(dt.line(seed, step, a.key, got, pubFrom, voiceFrom) + "\n")
		}
	}
	rec := legend + "\n" + stableKeys(names.Replace(all.String()))
	path := filepath.Join("testdata", "deck_trace.golden")
	if *updateDeckTrace {
		if err := os.WriteFile(path, []byte(rec), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the golden: %v (record it with -update-deck-trace on code identical to e57afeaf)", err)
	}
	if rec != string(want) {
		t.Errorf("the deck's record changed:\n%s", firstDifference(string(want), rec))
	}
}

// firstDifference is the first line where two records part, with its act.
func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	act := ""
	for i := range min(len(w), len(g)) { // bounded by the shorter record (P10-02)
		if strings.HasPrefix(w[i], "seed=") {
			act = w[i]
		}
		if w[i] != g[i] {
			return fmt.Sprintf("line %d, in %s:\n  want %s\n  got  %s", i+1, act, w[i], g[i])
		}
	}
	return fmt.Sprintf("the records are %d and %d lines long", len(w), len(g))
}
