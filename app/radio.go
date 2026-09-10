package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/spectrum"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// radioDeck is the dashboard's Radio hook (B4, architecture §5): resolves
// a location to its NWR transmitter (county SAME from the cached NWS point
// → vendored table → relay directories), tunes the pure-Go player, and
// streams status back to the UI. Everything here runs on tea cmd
// goroutines — never the update loop.
type radioDeck struct {
	p        *tea.Program
	nws      *nws.Provider
	resolver *stream.Resolver
	engine   *player.Engine
	composer synth.Composer // the broadcast\'s spoken text from the script library (0.13.0); zero value = the built-in scripts
	products *synth.Products
	units    render.Units
	// clockPref is the listener's clock, LIVE (Settings changes it while a cycle
	// is composing). It reaches the broadcast because MILITARY changes how a
	// callsign is read, not only how a time is written.
	clockPref *atomic.Int32
	voiceDir  string             // Piper install dir (Linux/Windows)
	analyzer  *spectrum.Analyzer // visualizer bands from the engine's tap (UAT 92)
	vizBuf    []float64          // one analysis window, reused per frame

	// abandonRead ends the read on the air when its audio stops making
	// progress (FR-9). Set where the Director is built, because the deck is
	// constructed first and the Director takes it as its voice; nil in tests
	// and in the pathless build.
	abandonRead func(why string) bool

	persistMode func(tty.RadioMode) error                      // saves the [m] pick (UAT 97); nil in tests
	fire        func(snapshot.LocationRef) synth.FireReport    // the location's fire report for the broadcast (UAT 114); nil = skipped
	seismic     func(snapshot.LocationRef) synth.SeismicReport // the location's seismic report for the broadcast (P4); nil = skipped
	marine      func(snapshot.LocationRef) synth.MarineReport  // the location's coastal block for the maritime report (0.14.0); nil = skipped
	warn        func(snapshot.Warning)                         // a fresh relay-directory failure becomes a radio_unavailable warning (Q1); nil in tests
	dirDown     map[string]bool                                // relays already warned about, so an outage warns once (guarded by mu)
	mountURLs   []string                                       // the tune list in order, so a fault can offer what has not been tried (MVS-D-76)
	mountOwner  map[string]stream.Station                      // the current tune list: mount URL → its station, so the label follows the mount that plays (guarded by mu)

	// cast is who reads what, on this host: the config, the memoised
	// resolutions and problems, and the session's unattended-install budget.
	// Guarded by mu; replaced wholesale so a reader sees a consistent view.
	cast castState

	installMu sync.Mutex // serializes Piper voice installs across concurrent callers (breaking audio + tune) — 0.12.0 P4

	// limiter bounds how many voice renders run at once — ONE per process,
	// here, because "how many say/piper are alive" is a property of the
	// machine, not of a call site (FR-12). Every voice the deck hands out is
	// wrapped by it; nothing else in app calls Say directly.
	limiter *synth.Limiter

	// tuneMu makes "check the epoch, then start the engine" one step, and
	// Stop's "bump the epoch, then halt" another (round 2 N-3): without it a
	// Stop landing between the check and engine.Start left audio playing.
	// Held only around those tails — never across resolving or a voice install.
	tuneMu sync.Mutex

	mu      sync.Mutex
	station string // label of the station being played
	detail  string
	mode    string // "live" | "synth" | ""
	ref     snapshot.LocationRef
	gen     uint64         // tune epoch (red-team 0.9.0 C-3): Tune and Stop bump it; a slow Tune that lost the race must not start playback
	repeat  tty.RepeatMode // [r] Off | One | Watchlist (UAT 83/93)
	// relayDwell is the Settings window's rotation choice; zero = unset.
	relayDwell time.Duration
	// relayLang is the Settings window's language preference; "" = unset.
	relayLang string
	pref      tty.RadioMode          // [m] Synth | Nearest Relay (UAT 97) — the source the user asked for
	queue     []snapshot.LocationRef // Watchlist mode's order (the favourites, from the dashboard)
	// emit hands the Director the bed's facts (T3.2b). It replaced a
	// time.AfterFunc: the dwell is the Director's now, and this deck only
	// reports what it alone can see.
	emit    func(lineup.Event)
	source  *synth.Source // the running synthesized broadcast, if any
	voiceID string        // chosen correspondent (UAT 84); "" = the platform default
	voices  []string      // available correspondents, listed once in the background (UAT 85)
}

// newRadioDeck wires the player. A resolver failure (a broken vendored
// table) is a build bug: surfaced, and the hook stays nil (controls inert).
func newRadioDeck(p *tea.Program, client *httpx.Client, provider *nws.Provider, units render.Units) *radioDeck {
	r, err := stream.NewResolver(stream.NewDirectory(client, "", ""))
	if err != nil {
		return nil
	}
	d := &radioDeck{p: p, nws: provider, resolver: r, units: units, products: synth.NewProducts(client, ""), voiceDir: voiceDir(), limiter: synth.NewLimiter(renderSlots(), synth.ReservedSlots)}
	d.engine, err = player.New(&player.OtoOutput{}, UserAgent, d.onStatus)
	if err != nil {
		return nil
	}
	d.engine.OnSilence(d.onSilence)
	d.engine.OnClipSpent(d.onClipSpent)
	d.engine.Trace(radioDebugLog)
	d.analyzer, err = spectrum.New(player.OutputRate)
	if err != nil {
		return nil
	}
	d.vizBuf = make([]float64, spectrum.FFTSize)
	go d.listVoices() // `say -v ?` takes seconds: never on a key press, never on a render (UAT 85)
	return d
}

// Spectrum is the visualizer feed (UAT 92): the latest window of whatever
// plays — relay or synthesized — as band levels. The one hook that runs on
// the update loop: it is called on the 50 ms visualizer tick and does one
// 2048-point FFT (~20 µs, one allocation — BenchmarkBands), so the loop
// never waits on audio.
func (d *radioDeck) Spectrum() []float64 {
	n := d.engine.Samples(d.vizBuf)
	return d.analyzer.Bands(d.vizBuf[:n])
}

// Tune implements tty.Radio: the USER picking a location.
//
// IT DOES NOT LIFT THE ALERT DUCK, and that is a reversal of the 0.12.0
// follow-up, which held that a deliberate tune means "play this station now".
// ALERTS ALWAYS HAVE PRIORITY: everything else queues
// behind them or plays under them. A listener who tunes while a warning is
// being read gets the station they asked for, ducked, and hears it come up when
// the alert finishes — which is the same thing the broadcast does.
//
// The duck now has exactly ONE owner, the director, which ducks when a sequence
// takes the air and restores when nothing is waiting or suspended. Nothing else
// touches it but a user STOP, where there is no broadcast left to duck.
//
// That single owner is the point. What broke was not a wrong decision, it was a
// distinction carried by a capital letter: this method lifted the duck and the
// unexported one did not, and the Watchlist advance called the wrong one. A rule
// that lives in the case of an identifier is a rule waiting to be missed.
func (d *radioDeck) Tune(ref snapshot.LocationRef) { d.tune(ref) }

// tune resolves, then plays the first relayed station — or the synthesized
// broadcast when nothing relays this location (B4 step 2: 89 % of
// transmitters). The automatic paths call it directly (no duck lift).
func (d *radioDeck) tune(ref snapshot.LocationRef) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	d.mu.Lock()
	d.ref = ref
	d.gen++
	gen, pref := d.gen, d.pref
	d.mu.Unlock()
	// THE PROGRAMME IS RUNNING, AND THE DIRECTOR HAS TO BE TOLD — HERE, where
	// the listener asked for a location, and NOT wherever the audio happens to
	// begin (red team 2026-09-09, finding 1).
	//
	// IT USED TO RIDE ON setMode's transition edge, which made "the programme
	// is running" a side effect of the DECK changing mode. On the synthesised
	// path setMode is reached only from startSynth, and the merged station does
	// not call startSynth — so the first need arrived at a Director still
	// Stopped, advances(MainTrack) refused the card, no mode ever changed, and
	// the Director was never powered. Every subsequent need was refused the
	// same way: a permanently silent station with a permanently empty lineup
	// and no fault raised, because nothing failed and nothing was ever
	// admitted.
	//
	// THE ASYMMETRY WAS THE DEFECT. Stop is reported from Stop, where the
	// listener acts. Start is now reported from here, for the same reason and
	// in the same terms — and BEFORE the relay/synth fork, so which medium wins
	// cannot change whether the station is on.
	//
	// A REPEATED TUNE IS NOT A SECOND START: onPowered no-ops when the power is
	// already what it is asked for, which is why this needs no transition edge
	// of its own to guard it.
	d.tell(lineup.Powered{To: lineup.Running})
	same := stream.SAMEFromUGC(d.nws.CountyUGC(ctx, ref))
	stations, statuses := d.resolver.ResolveWithStatus(ctx, ref.Lat, ref.Lon, same)
	d.noteDirectories(statuses)
	// UAT 78/97: Synth is the default — a neighbour's broadcast is a
	// neighbour's forecast. [m] Nearest Relay asks for the live station
	// instead: the covering transmitter when relayed, else the nearest
	// relayed one; none in reach still means Synth, with the reason.
	st, live := stream.Station{}, false
	if pref == tty.ModeRelay {
		st, live = chooseNearest(stations, d.watchlistLang())
	}
	if !live {
		d.needsRead(ref, d.synthReason(same, ref, stations), gen)
		return
	}
	// The tune list spans every candidate station in the resolver's order
	// (Q1): a directory lists sources that may not be connected (a 404),
	// and with weatherUSA offered again a transmitter can be "relayed" by a
	// dead mount alone — the engine must fall through to the next live
	// station, as it did when that transmitter was simply not offered.
	urls, owners := tuneList(stations, st)
	d.tuneMu.Lock()
	defer d.tuneMu.Unlock()
	if !d.epoch(gen) {
		return // stopped or re-tuned while resolving: this tune is stale
	}
	d.mu.Lock()
	d.mountOwner, d.mountURLs = owners, urls
	d.mu.Unlock()
	d.setMode("live", d.label(st), st.Mounts[0].Relay)
	d.engine.Start(urls, st.Callsign+" "+st.Site) // the dwell arms when the relay reports Playing (onStatus)
}

// tuneList flattens the candidate stations' mounts in order and remembers
// which station each mount belongs to, so the label can follow the mount
// that actually plays.
func tuneList(stations []stream.Station, first stream.Station) ([]string, map[string]stream.Station) {
	var urls []string
	owners := map[string]stream.Station{}
	add := func(st stream.Station) {
		for _, m := range st.Mounts { // bounded by the station's mounts (P10-02)
			if _, seen := owners[m.URL]; seen {
				continue
			}
			urls = append(urls, m.URL)
			owners[m.URL] = st
		}
	}
	// THE CHOSEN STATION LEADS, and it is not always the resolver's first.
	//
	// The engine starts at urls[0] while the deck is labelled with the station
	// chooseNearest picked. Those were the same station for as long as
	// chooseNearest meant "the first one with a mount". The language preference
	// broke that: it may pick a co-located station further down the order, and
	// the deck would then name the transmitter the listener asked for while the
	// audio came from the one beside it — the label and the sound disagreeing,
	// silently, which is the class of defect this release exists to remove.
	add(first)
	for _, st := range stations { // bounded by the candidate list (P10-02)
		add(st)
	}
	return urls, owners
}

// onSilence raises the fault window: the mount being played is up and
// broadcasting nothing (MVS-D-76).
//
// The candidates it offers are the OTHER stations on this tune list — the ones
// the engine has not tried — because offering the listener the mount that just
// went silent is offering them the fault again. They arrive already in the
// engine's own fall-through order, so "Recommended" is what it would have
// reached next anyway.
func (d *radioDeck) onSilence(mount, _ string) {
	radioDebugLog("relay:silent:" + mount)
	d.p.Send(tty.RelaySilentMsg{Candidates: d.silentCandidates(mount)})
}

// onClipSpent says so when a read's watcher gave up on it (FR-9): the player was
// still claiming to play after ten minutes of AIR time and was closed from under
// it — a read that neither finished nor errored.
//
// IT IS NOT THE RELAY-FAULT WINDOW. That window offers other stations, which is
// the right answer for a mount broadcasting silence and no answer at all for a
// read: there is no other station to switch to, and the broadcast itself is
// fine. This is the station's own detail line, where every other "could not
// read" already goes.
//
// THE LISTENER IS TOLD WHAT HAPPENED, not what it means. Ten minutes of audio
// that never ended has one honest description and no diagnosis this code can
// offer.
func (d *radioDeck) onClipSpent() {
	radioDebugLog("read:clip:spent")
	// THE READ GOES WITH THE CLIP (FR-9). The sequence is still running: it
	// slept the line's computed length minutes ago and has been issuing further
	// lines over a player that has now been closed from under it. Cancelling
	// unwinds it onto the path a cut-short read already takes — the schedule
	// advances and the bed comes back up — instead of leaving the arbiter
	// occupied and the broadcast ducked under nothing.
	//
	// THE LISTENER IS TOLD ONLY IF THEY WERE HEARING IT. abandonRead reports
	// whether there was a read on the air; a spent clip from a read that has
	// already ended is a diagnostic, not something to put on the detail line.
	if d.abandonRead != nil && !d.abandonRead("clip-spent") {
		return
	}
	d.setDetail("a read did not finish and was ended")
}

// silentCandidates is what to offer instead of the mount that went quiet: every
// OTHER station on the tune list, once each, in the engine's own order.
func (d *radioDeck) silentCandidates(mount string) []tty.RelayCandidate {
	d.mu.Lock()
	owners, urls := d.mountOwner, append([]string(nil), d.mountURLs...)
	d.mu.Unlock()
	dead, ok := owners[mount]
	if !ok {
		return nil // a mount from a tune that has already been replaced
	}
	seen := map[string]bool{dead.Callsign: true}
	var out []tty.RelayCandidate
	for _, url := range urls { // bounded by the tune list (P10-02)
		st, known := owners[url]
		if !known || seen[st.Callsign] {
			continue
		}
		seen[st.Callsign] = true
		out = append(out, tty.RelayCandidate{Label: d.label(st), Key: st.Callsign})
	}
	return out
}

// followMount re-labels the deck when the engine has moved on to another
// station's mount (a candidate earlier in the list was refused). Returns
// the label now in force. Callers hold no lock.
func (d *radioDeck) followMount(mount string) {
	if mount == "" {
		return
	}
	d.mu.Lock()
	owner, ok := d.mountOwner[mount]
	current := d.station
	d.mu.Unlock()
	if !ok || d.label(owner) == current {
		return
	}
	relay := ""
	for _, m := range owner.Mounts {
		if m.URL == mount {
			relay = m.Relay
		}
	}
	d.setMode("live", d.label(owner), relay)
}

// epoch reports whether gen is still the current tune (no Stop or newer
// Tune since it began).
func (d *radioDeck) epoch(gen uint64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gen == gen
}

// noteDirectories turns a relay directory's first failure into one
// radio_unavailable warning (Q1, DISCOVER LR-1: the weatherUSA directory
// was unreachable for every Go build since 1.22 and nobody could see it)
// and re-arms when the relay recovers, so an outage is said once.
func (d *radioDeck) noteDirectories(statuses []stream.Status) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dirDown == nil {
		d.dirDown = map[string]bool{}
	}
	for _, st := range statuses {
		switch {
		case st.Err == nil:
			delete(d.dirDown, st.Relay)
		case !d.dirDown[st.Relay]:
			d.dirDown[st.Relay] = true
			if d.warn != nil {
				d.warn(snapshot.Warning{Code: snapshot.WarnRadioUnavailable, Provider: st.Relay,
					Message: fmt.Sprintf("relay directory %s unreachable — its transmitters are not offered until it answers: %v", st.Relay, st.Err)})
			}
		}
	}
}

// SetMode implements tty.Radio (UAT 97): [m] picks the source; a playing
// location re-tunes under the new mode at once.
func (d *radioDeck) SetMode(mode tty.RadioMode) {
	d.mu.Lock()
	d.pref = mode
	ref, playing, persist := d.ref, d.mode != "", d.persistMode
	d.mu.Unlock()
	if persist != nil {
		_ = persist(mode) // a failed save is not a playback failure; the pick still applies for this run
	}
	if playing && d.engine.Status().State != player.Stopped {
		go d.tune(ref) // Watchlist advance is automatic — stays ducked under a takeover
	}
}

// liveDwell is how long Watchlist stays on a live relay before moving on (UAT
// 93): a relay never ends, so one NWR cycle (~5 min) is the "broadcast" we let
// it finish. The deck no longer counts it — it only says what the number is,
// and the Director keeps the deadline.
const liveDwell = 5 * time.Minute

// WATCHPOST_WATCHLIST_DWELL overrides it, for testing the rotation without
// waiting five minutes a station (HUM LEAD, UAT 2026-09-04).
//
// AN ENVIRONMENT VARIABLE IS THE HALF THAT IS NOT A UI DECISION. Where the row
// belongs in Settings, what it is called and what values it offers are the HUM
// LEAD's to specify; making the duration a value rather than a constant is not,
// and it is what makes the rotation testable in thirty seconds. The Settings row
// reads this same function when it arrives.
//
// It takes a Go duration ("30s", "2m"). Anything unparseable or non-positive is
// the default rather than an error: a mistyped variable should not silently stop
// the rotation, which is exactly the failure this exists to help find.
// The Settings window's choice lives ON THE DECK (d.relayDwell), not in a
// package variable. A package-level override is the same setting reachable from
// every test in the package at once: the first version of this was exactly that,
// and its own pin had to save and restore the global to avoid leaking into the
// next test. State that needs a t.Cleanup to be safe is state in the wrong
// place. Zero means nothing has been set this run and the environment or the
// default answers instead.
func (d *radioDeck) watchlistDwell() time.Duration {
	d.mu.Lock()
	set := d.relayDwell
	d.mu.Unlock()
	if set > 0 {
		return set // the listener set it in Settings this run
	}
	return envWatchlistDwell()
}

// watchlistLang is the language preference the tie-break uses. English until a
// listener says otherwise: the app's own voice is English, so it is the answer
// that surprises fewest people — not a judgement about which is better.
func (d *radioDeck) watchlistLang() string {
	d.mu.Lock()
	set := d.relayLang
	d.mu.Unlock()
	if set != "" {
		return set
	}
	return stream.LangEnglish
}

// envWatchlistDwell is the half that has no deck: the variable and the default.
func envWatchlistDwell() time.Duration {
	v := os.Getenv("WATCHPOST_WATCHLIST_DWELL")
	if v == "" {
		return liveDwell
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		radioDebugLog("watchlist-dwell:ignored:" + v)
		return liveDwell
	}
	return d
}

// tell hands the Director a fact about the bed. Nil until the schedule is wired,
// and on a station with no audio it stays nil — silent rather than a panic on
// the one path that has no deck at all.
func (d *radioDeck) tell(ev lineup.Event) {
	tellUnder(&d.mu, &d.emit, ev)
}

// synthReason explains the Synth default: the unrelayed covering
// transmitter, plus the nearest live broadcast when there is one.
func (d *radioDeck) synthReason(same string, ref snapshot.LocationRef, stations []stream.Station) string {
	reason := d.unrelayedLabel(same, ref)
	if len(stations) > 0 {
		o := render.Opts{Units: d.units}
		km := stations[0].KM
		reason += fmt.Sprintf(" · nearest live: %s %s %s", stations[0].Callsign, stations[0].Site, strings.TrimSpace(o.Distance(&km)))
	}
	return reason
}

// needsRead is THE ONE SEAM at which a location becomes a synthesised read,
// and the merge's whole surface area (0.16.0 P3).
//
// THREE SITES REACH IT, and until now each started audio itself: the ordinary
// tune when nothing live carries this location, the relay that failed while
// playing, and the relay that went silent. They are three different facts with
// one consequence — nobody is carrying this location, so the station must read
// it — and that consequence is what the schedule now owns.
//
// THE DECK REPORTS, THE DIRECTOR DECIDES (T3.2b), which is the rule the
// neighbouring dwell case already states. What travels is the fact; whether it
// becomes a card, where it sits, and whether it is read twice are the
// Director's, and it answers all three from the schedule it holds.
//
// THE STALENESS CHECK GUARDS THE REPORT, not just the audio. startSynth has
// always dropped a fallback that arrived after the listener stopped or re-tuned;
// without the same check here, that stale fallback would still queue a card —
// the listener would have stopped the station and been read to anyway. The
// guard inside startSynth stays: it has its own callers, and it retires with the
// direct path at P3(d).
func (d *radioDeck) needsRead(ref snapshot.LocationRef, why string, gen uint64) {
	stage, fresh := mainTrack(), d.epoch(gen)
	// THE DARK RUN'S ONLY INSTRUMENT (0.16.0 P3).  The whole point of the dark
	// stage is that the producer's decisions can be compared against the live
	// path's, and neither is visible without this: the live path logs its
	// engine transitions and its segments, and the need that produced them was
	// logged nowhere at all.
	//
	// RECORDED BEFORE THE STALENESS CHECK, AND CARRYING ITS ANSWER.  A need
	// dropped as stale is exactly the kind of thing the comparison is looking
	// for — "the live path started a read here and the producer did not" has
	// two possible causes, and this is what tells them apart.  Built only when
	// the diagnostic is on, because the concatenation is pure cost otherwise
	// (the shape radioDebugOn exists for).
	if radioDebugOn() {
		d.debugLog(fmt.Sprintf("needs-read stage=%s fresh=%t ref=%s why=%s", stage, fresh, snapshot.Key(ref), why))
	}
	if !fresh {
		return // the listener stopped, or moved on: this need is about a location nobody is on
	}
	if stage.reports() {
		// THE HEADLINE IS THE LOCATION'S OWN NAME. A card is showable from the
		// moment it exists (DR-7), and at this point there is nothing else true
		// about it: its words are composed at standby, minutes later.
		d.tell(lineup.NeedsRead{Ref: string(snapshot.Key(ref)), Headline: ref.Label})
	}
	// THE DECK STILL PLAYS IT. There is no stage in which it does not: "live"
	// meant the card was read through the ARBITER, and D-33 rules that the
	// programme is not a narration — a chosen read replaces the bed rather
	// than speaking over it. What the schedule eventually takes over is WHEN
	// this happens, not what performs it (BD-9: a report's speak is the engine
	// Source adapter).
	d.startSynth(ref, why, gen)
}

// startSynth voices the location's NWS products (architecture §5 Synth):
// the voice is the built-in `say` on macOS, Piper elsewhere — installed on
// first use with progress shown in the player (HUM LEAD: first-run install).
func (d *radioDeck) startSynth(ref snapshot.LocationRef, why string, gen uint64) {
	if !d.epoch(gen) {
		return // a stale fallback (a relay that failed after the user stopped) must not relabel anything
	}
	d.setMode("synth", "Watchpost Synth · "+ref.Label, why)
	voice, err := d.voice() // may install Piper (minutes): never under tuneMu
	d.tuneMu.Lock()
	defer d.tuneMu.Unlock()
	if !d.epoch(gen) {
		return // stopped while the voice was being found/installed
	}
	if err != nil {
		d.setMode("synth", "Watchpost Synth · "+ref.Label, err.Error())
		d.engine.Fail(err.Error()) // the reason, in the player (F2)
		return
	}
	// The sign-off names whichever voice reaches it (UAT 94: the voice may change mid-cycle).
	src, err := synth.NewSource(voice, func(ctx context.Context) ([]synth.Segment, error) { return d.segments(ctx, ref, synth.VoiceToken) },
		func(seg synth.Segment, spoken time.Duration) {
			d.debugLog(fmt.Sprintf("segment key=%q spoken=%s", seg.Key, spoken.Round(time.Millisecond))) // WATCHPOST_DEBUG_RADIO: which segment the stream reached (UAT 2026-08-28: a cycle that ended before its tail)
			d.setDetailTimed(seg.Text, spoken)
		})
	if err != nil {
		d.engine.Fail(err.Error())
		return
	}
	// The cast: who reads each role, and how correspondents introduce
	// themselves. Both are installed BEFORE the source starts, so the very
	// first segment already resolves through the cast rather than through the
	// root voice the Source was constructed with.
	src.SetResolver(func(role cast.Role) (synth.Voice, error) {
		v, _, err := d.resolveVoice(role)
		return v, err
	})
	src.SetHandoffLine(d.composer.HandoffLine)
	d.mu.Lock()
	src.Loop(d.repeat == tty.RepeatOne) // Watchlist ends the cycle too — then advances (UAT 93)
	d.source = src
	d.mu.Unlock()
	d.engine.StartSource("Watchpost Synth ("+voice.Name()+")", src.Rate(), src.Open)
}

// segments composes one broadcast cycle: the location's current
// observation and alerts (from the provider, served by the client cache)
// plus the office's latest products.
func (d *radioDeck) segments(ctx context.Context, ref snapshot.LocationRef, voiceName string) ([]synth.Segment, error) {
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{d.nws.ID()})
	for _, kind := range []snapshot.FetchKind{snapshot.KindObs, snapshot.KindAlerts} {
		if frag, err := d.nws.Fetch(ctx, snapshot.FetchReq{Kind: kind, Locations: []snapshot.LocationRef{ref}}); err == nil {
			asm.Apply(frag, nil) // a read-driven fetch, not the cycle that answers the row (see pipelines.go)
		}
	}
	snap := asm.Snapshot()
	if len(snap.Locations) == 0 {
		return nil, fmt.Errorf("no location")
	}
	office := d.nws.Office(ctx, ref)
	products, _ := d.products.Latest(ctx, office) // a product outage still leaves the observation and alerts to read
	zone, county := d.nws.ForecastZone(ctx, ref), d.nws.CountyUGC(ctx, ref)
	for i := range products {
		products[i].Text = synth.FilterUGC(products[i].Text, zone, county) // this location's blocks only (UAT 81)
	}
	now := time.Now()
	if z, err := time.LoadLocation(ref.TZ); err == nil && ref.TZ != "" {
		now = now.In(z)
	}
	var fire synth.FireReport
	if d.fire != nil {
		fire = d.fire(ref)
	}
	var seismic synth.SeismicReport
	if d.seismic != nil {
		seismic = d.seismic(ref)
	}
	var maritime synth.MarineReport
	if d.marine != nil {
		maritime = d.marine(ref)
		maritime.Forecast = synth.CoastalForecast(products, zone)
	}
	return d.composer.Compose(snap.Locations[0], products, now, d.units == render.UnitF, voiceName, d.stationFor(county, ref), synth.Reports{Fire: fire, Seismic: seismic, Maritime: maritime}, d.clock()), nil
}

// stationFor names the NWR transmitter the lead points listeners to (UAT
// 112): the one covering the county, else the nearest; none when the table
// is missing.
func (d *radioDeck) stationFor(countyUGC string, ref snapshot.LocationRef) synth.Station {
	if d.resolver == nil {
		return synth.Station{}
	}
	var tx *stream.Transmitter
	for _, c := range d.resolver.CoveringTransmitters(stream.SAMEFromUGC(countyUGC)) {
		tx = c
		break
	}
	if tx == nil {
		tx = d.resolver.NearestTransmitter(ref.Lat, ref.Lon)
	}
	if tx == nil {
		return synth.Station{}
	}
	return synth.Station{Callsign: tx.Callsign, Site: tx.Site, State: tx.State, FreqMHz: tx.FreqMHz}
}

// Stop implements tty.Radio.
func (d *radioDeck) Stop() {
	d.tuneMu.Lock() // one step with the halt: a Tune tail cannot slip in between (N-3)
	defer d.tuneMu.Unlock()
	d.mu.Lock()
	d.gen++     // any Tune still resolving is stale now (C-3)
	d.mode = "" // and no fallback or Watchlist advance follows a user's stop
	d.mu.Unlock()
	// NOTHING FOLLOWS A STOP, and the Director has to be told: it holds the
	// dwell now, and a schedule that never heard about the stop would move the
	// bed on five minutes later and start the station up again by itself.
	d.tell(lineup.Powered{To: lineup.Stopped})
	// NOT Restore(): whether an alert is on the air is the takeover's to say,
	// and it pairs its own. Halt silences the broadcast either way, and a
	// suppression cleared here would let whatever the listener starts next come
	// up over an alert still being read.
	d.engine.Halt()
}

// SetVolume implements tty.Radio.
func (d *radioDeck) SetVolume(pct int) { d.engine.Volume(pct) }

// The radioDeck is the director's voice (0.13.0; app/director.go): duck, tone,
// render, play, pause, resume, discard, restore — the director decides who
// speaks and when.

// duck puts an alert on the air over the broadcast.
//
// WHICH WAY the broadcast gives way — a relay dips, a rendered cycle holds — is
// the engine's, because only the engine knows what is playing at each moment and
// the source can change while the alert is still reading. Asking the deck's mode
// here fixed an answer the audio could outlive.
func (d *radioDeck) duck() { d.engine.Suppress() }

// tone sounds the attention tone once, returning its length so the caller
// waits it out before the first line.
//
// It renders FROM CONSTANTS at synth.ToneRate and RESOLVES NO VOICE (FR-9).
// That is the point of the whole tone path: an alert's sound must reach the
// listener while the correspondent who will read it is still being resolved, or
// still downloading. Before 0.14.0 this went through d.voice(), which on a
// fresh Linux host could block on a ~63 MB install — the alert was silent for
// as long as the download took.
//
// The class parameter arrives with the Station Director in Task 2.6; until then
// every alert sounds the classic tone, exactly as 0.13.0 did.
// clock is the listener's clock, the 12-hour default when nothing set one (the
// older tests, which build a deck by hand).
func (d *radioDeck) clock() render.Clock {
	return clockFrom(d.clockPref)
}

func (d *radioDeck) tone(class cast.Class) time.Duration {
	if cast.Muted(class, d.tones()) {
		return 0 // the class is muted: no tone. The WORDS always read (MVS-D-26).
	}
	if radioDebugOn() {
		d.debugLog("tone:" + class.Key()) // the live M4 instrument (perf-protocol.md §1 item 2)
	}
	tone := alertTonePCM(class)
	_ = d.engine.PreviewAside(synth.ToneRate, bytes.NewReader(tone)) // the attention tone never drives the bars
	return pcmDuration(tone, synth.ToneRate)
}

// pause holds the line in flight (a read a takeover suspends); resume lets
// it play on from where it stopped.
func (d *radioDeck) pause()  { d.engine.PausePreview() }
func (d *radioDeck) resume() { d.engine.ResumePreview() }

func (d *radioDeck) stop() { d.engine.StopPreview() }

// fault puts what went wrong where the operator reads it, and NOWHERE ELSE
// (FR-9.2; HUM LEAD, 2026-09-08).
//
// NOT SPOKEN, and that is the ruling rather than an omission: an operational
// message over the air is confusing, and the audience can do nothing about it.
// A radio station in this position cuts to "we are experiencing technical
// difficulties" and NOAA Weather Radio names an alternate frequency — both
// infrastructure this app does not have. What it can do is tell the person who
// can act.
func (d *radioDeck) fault(why string) {
	radioDebugLog("read:fault:" + why)
	d.setDetail(why)
}

// discard drops a held line whose sequence ended while it waited.
func (d *radioDeck) discard() { d.engine.DropHeld() }

// render turns a narration line into audio (say/piper blocks ~1 s per
// line, longer for an event read — the director's sequences run on their
// own goroutines, off the UI loop); play puts a rendered line on the air
// over the ducked radio. Split so a line rendered while its sequence is
// suspended never starts under a takeover.
// render turns a narration line into audio IN THE ROLE'S VOICE, and records on
// the player's detail line WHY it failed when it did — a silent takeover must
// never be unexplained.
func (d *radioDeck) render(ctx context.Context, role cast.Role, text string) (clip, bool) {
	v, res, err := d.resolveVoice(role)
	if err != nil {
		d.setDetail(fmt.Sprintf("no voice for %s: %v", role, err))
		return clip{}, false
	}
	pcm, err := synth.AlertNarration(ctx, v, text)
	if err != nil {
		d.setDetail(fmt.Sprintf("%s could not read: %v", res.Spoken, err))
		return clip{}, false
	}
	return clip{text: text, pcm: pcm, rate: v.Rate(), dur: pcmDuration(pcm, v.Rate()), role: role}, true
}

func (d *radioDeck) play(c clip) {
	if c.viz {
		_ = d.engine.Preview(c.rate, bytes.NewReader(c.pcm)) // the bars follow it
		return
	}
	_ = d.engine.PreviewAside(c.rate, bytes.NewReader(c.pcm)) // a takeover: aside, the bars keep to the broadcast
}

// restore lifts the duck — the radio returns after the sequence.
// restore lifts BOTH — a hold and a duck — because the mode can change while a
// sequence is on air and the one that was taken is not always the one that is
// given back.
func (d *radioDeck) restore() { d.engine.Restore() }

// overlay shows a narration on the radio panel while it plays — the event
// being read, its script as the paced marquee — without touching the deck's
// own station/detail; pushStatus puts the true state back afterwards.
func (d *radioDeck) overlay(station, short, detail string, spoken time.Duration) {
	st := d.engine.Status()
	d.p.Send(tty.RadioStatusMsg{State: "playing", Station: station, Short: short, Detail: detail, Spoken: spoken, Volume: st.Volume})
}

// pushStatus re-sends the deck's current state (after an overlay).
func (d *radioDeck) pushStatus() {
	d.mu.Lock()
	detail := d.detail
	d.mu.Unlock()
	d.setDetailTimed(detail, 0)
}

// pcmDuration is the playback length of 16-bit LE stereo PCM at rate (4 bytes a
// frame); resampling preserves it, so it is the true on-air duration.
func pcmDuration(pcm []byte, rate int) time.Duration {
	if rate <= 0 {
		return 0
	}
	frames := len(pcm) / 4
	return time.Duration(frames) * time.Second / time.Duration(rate)
}

// label: "KEC49 Monterey CA 162.550 MHz · 78 mi (nearest relayed)".
func (d *radioDeck) label(st stream.Station) string {
	o := render.Opts{Units: d.units}
	km := st.KM
	s := fmt.Sprintf("%s %s %s %s MHz · %s", st.Callsign, st.Site, st.State, st.FreqMHz, strings.TrimSpace(o.Distance(&km)))
	if !st.Covering {
		s += " (nearest relayed)"
	}
	return s
}

// unrelayedLabel names the covering transmitter nobody relays, so the
// row explains itself: "KEC62 San Diego — not relayed".
func (d *radioDeck) unrelayedLabel(same string, ref snapshot.LocationRef) string {
	for _, tx := range d.resolver.CoveringTransmitters(same) {
		return fmt.Sprintf("%s %s %s %s MHz — not relayed", tx.Callsign, tx.Site, tx.State, tx.FreqMHz)
	}
	return ref.Label + " — no NWR relay in reach"
}

func (d *radioDeck) setMode(mode, station, detail string) {
	// THE POWER IS NOT REPORTED FROM HERE ANY MORE (red team 2026-09-09,
	// finding 1). This used to send Powered{Running} on the transition out of
	// an empty mode, which made "the programme is running" a fact about the
	// DECK's mode string rather than about the listener. `tune` reports it now,
	// where the listener asks for a location and before the relay/synth fork —
	// so a station whose audio is owned by the schedule still starts.
	//
	// The transition variable went with it: it existed only to guard that send,
	// and a local kept "in case" is a reader's question with no answer.
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mode, d.station, d.detail = mode, station, detail
}

// setDetail updates the detail line and pushes it to the UI at once
// (install progress, the sentence being narrated).
func (d *radioDeck) setDetail(detail string) { d.setDetailTimed(detail, 0) }

// setDetailTimed also carries how long the line will be spoken, so the
// marquee can pace itself to the voice (UAT 83).
func (d *radioDeck) setDetailTimed(detail string, spoken time.Duration) {
	d.mu.Lock()
	d.detail = detail
	station, mode, ref, st := d.station, d.mode, d.ref, d.engine.Status()
	d.mu.Unlock()
	d.p.Send(tty.RadioStatusMsg{State: string(st.State), Station: station, Detail: detail, Spoken: spoken, Volume: st.Volume, Live: mode == "live", Location: snapshot.Key(ref)})
}

// onStatus forwards engine status to the dashboard; a relay that fails
// outright falls back to the synthesized broadcast (§5: Live → Synth).
func (d *radioDeck) onStatus(st player.Status) {
	d.logStatus(st)         // WATCHPOST_DEBUG_RADIO: the transitions, for a relay that plays nothing (follow-up F-2)
	d.followMount(st.Mount) // a later candidate's mount is playing: the label says which (Q1)
	d.mu.Lock()
	station, detail, mode, ref, src, gen := d.station, d.detail, d.mode, d.ref, d.source, d.gen
	d.mu.Unlock()
	state := st.State
	if st.Title != "" {
		detail = st.Title
	}
	if st.Err != "" && st.State == player.Failed {
		detail = st.Err
	}
	ended, voiceErr := d.cycleEnded(st, src)
	if voiceErr != "" {
		// The stream ended because the voice could not render, not because
		// the broadcast finished (C-4/F3): say so, and never advance on it.
		state, detail = player.Failed, voiceErr+" — check your correspondents in Settings, or reinstall the voice"
	}
	if state == player.Stopped {
		station = "" // the row falls back to the focused location's name
	}
	d.p.Send(tty.RadioStatusMsg{State: string(state), Station: station, Detail: detail, Volume: st.Volume, Live: mode == "live", Location: snapshot.Key(ref)})
	// Off the engine goroutine (it is finishing this very status): Halt
	// inside Tune waits for it. Nothing follows a user's Stop (mode == "").
	if st.State == player.Failed && mode == "live" {
		go d.needsRead(ref, "relay unavailable — "+st.Err, gen)
	}
	// THE DECK REPORTS, THE DIRECTOR DECIDES (T3.2b). What used to be a
	// time.AfterFunc here — armDwell setting a five-minute timer, advanceQueue
	// firing on it — is now two facts the deck is the only thing able to
	// observe: that a synthesised cycle ran to its end, and where the bed landed
	// and when it actually started playing. Whether either moves the rotation on
	// is the Director's, and it is a pure function of those facts, the
	// listener's settings and the clock.
	//
	// WATCHLIST IS NOT ASKED HERE ANY MORE. It was the deck's test before; the
	// Director holds the rotation now and a zero dwell is how "not Watchlist"
	// reaches it, so reporting the fact unconditionally is right and filtering
	// it here would be a second copy of the rule.
	if ended && mode != "" {
		d.tell(lineup.Ended{})
	}
	if st.State == player.Playing {
		// THE DWELL STARTS FROM HERE, which is why this is reported at Playing
		// rather than when the tune was asked for: a resolve and a connect can
		// take seconds, and charging those to the listener's turn would cut
		// every one short by however slow the network was that time.
		d.tell(lineup.Tuned{Ref: string(snapshot.Key(ref)), Live: mode == "live"})
	}
}

// cycleEnded reads a Stopped status: ended is a synth cycle that played to
// its sign-off; voiceErr is the voice's own failure when the stream ended
// because a segment could not render (then ended is false — never an
// advance on it). The diagnostic log records which it was.
func (d *radioDeck) cycleEnded(st player.Status, src *synth.Source) (ended bool, voiceErr string) {
	if st.State != player.Stopped || st.Title != player.EndedTitle || src == nil {
		return st.State == player.Stopped && st.Title == player.EndedTitle, ""
	}
	if err := src.Err(); err != nil {
		voiceErr = err.Error()
	}
	d.debugLog(fmt.Sprintf("cycle-end source-err=%q", voiceErr))
	return voiceErr == "", voiceErr
}

// logStatus appends one line per engine status to the WATCHPOST_DEBUG_RADIO
// log: the state, the mount, the error and the title — what "nothing after
// 90 seconds" needs to become a cause. Never a secret: mounts are public URLs.
func (d *radioDeck) logStatus(st player.Status) {
	d.debugLog(fmt.Sprintf("%-12s mount=%q err=%q title=%q vol=%d", st.State, st.Mount, st.Err, st.Title, st.Volume))
}

// debugLog appends one timestamped line to the file named by
// WATCHPOST_DEBUG_RADIO (diagnostic, opt-in, off by default): engine
// statuses, the synth's segments as they reach the air, and why a cycle
// ended. The one writer for the radio diagnostic.
func (d *radioDeck) debugLog(line string) { radioDebugLog(line) }

// radioDebugLog is the package-level form. The ticker writes the `breaking:`
// entry of the M4 live instrument (perf-protocol.md §1 item 2) and has no deck,
// so the body lives here and the deck method delegates — one writer, two
// callers, rather than two implementations of the same file format.
func radioDebugLog(line string) {
	if path := radioDebugPath(); path != "" {
		writeRadioDebug(path, line)
	}
}

// radioDebugOn reports whether the diagnostic is enabled, so a caller on a
// latency-sensitive path can skip BUILDING its line.
//
// It matters: the takeover path's entry log is one string concatenation, and
// that concatenation ran on every takeover whether or not anybody was
// collecting the log. The diagnostic is off by default, so the allocation was
// pure cost on the one path M4 measures.
func radioDebugOn() bool { return os.Getenv(radioDebugEnv) != "" }

const (
	radioDebugEnv = "WATCHPOST_DEBUG_RADIO"

	// radioDebugMax is the size one log may reach before it is rotated. A 24/7
	// process writing three to ten lines per event needs a ceiling, and eight
	// mebibytes is days of them.
	radioDebugMax = 8 << 20

	// radioDebugNameMax bounds the name the environment may choose.
	radioDebugNameMax = 32
)

// radioDebugName is the file name the environment asked for, and "" when what
// it asked for is not a name.
//
// A NAME, NOT A PATH (FR-9.2). The variable took a path and the writer appended
// to it, so an unvalidated append-anywhere file write was one environment
// variable away on a process that runs all day — and the plan's first revision
// had it ON BY DEFAULT. The variable now picks WHICH log under the cache root,
// which is the only part of the decision a caller has any business making.
// Anything else falls back to the default name rather than failing, because a
// diagnostic that refuses to start is a diagnostic nobody collects.
//
// The rule is debugAddr's, one file down: the environment may choose the
// harmless half of the decision and nothing else.
func radioDebugName(v string) string {
	if v == "" || v == "1" || len(v) > radioDebugNameMax {
		return "radio"
	}
	for _, r := range v { // bounded by the name (P10-02)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return "radio" // a separator, an upper case, a dot: not a name
		}
	}
	return v
}

// radioDebugPath is where the diagnostic goes, or "" when it is off:
// <OS cache dir>/watchpost/debug/<name>.log.
//
// UNDER THE CACHE ROOT, with the other things this app writes and deletes
// freely. "" when the OS gives no cache directory, which turns the diagnostic
// off rather than guessing at a location.
func radioDebugPath() string {
	v, ok := os.LookupEnv(radioDebugEnv)
	if !ok || v == "" {
		return ""
	}
	dir := userCacheSubdir("debug")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, radioDebugName(v)+".log")
}

// writeRadioDebug appends one timestamped line, rotating the log once it has
// grown past its ceiling.
//
// ONE GENERATION. The previous log is kept as .1 and the one before it is
// dropped: a diagnostic is read while the thing it describes is still fresh,
// and keeping more is disk nobody asked for on a process that runs all day.
//
// 0600 ON BOTH THE FILE AND THE DIRECTORY. It carries station names, mount
// URLs and the listener's own locations.
func writeRadioDebug(path, line string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	if fi, err := os.Stat(path); err == nil && fi.Size() >= radioDebugMax {
		_ = os.Rename(path, path+".1") // best effort: a failed rotation must not stop the log
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339Nano), line)
}

// tuneCallsign plays one named transmitter from the current tune list.
//
// It plays only mounts ALREADY RESOLVED for this location: the window offers
// what the resolver found, so answering it is choosing among those rather than
// starting a new search. Its own mounts lead, and the rest follow, so a station
// the listener picked that is itself dead still falls through.
func (d *radioDeck) tuneCallsign(callsign string) {
	d.mu.Lock()
	owners, urls := d.mountOwner, append([]string(nil), d.mountURLs...)
	d.mu.Unlock()
	var lead, rest []string
	for _, u := range urls { // bounded by the tune list (P10-02)
		if st, ok := owners[u]; ok && st.Callsign == callsign {
			lead = append(lead, u)
			continue
		}
		rest = append(rest, u)
	}
	if len(lead) == 0 {
		return // it is not on this list any more; the tune moved on
	}
	d.setMode("live", d.label(owners[lead[0]]), owners[lead[0]].Mounts[0].Relay)
	d.engine.Start(append(lead, rest...), callsign)
}

// readSynth falls through to the synthesized report for the current location —
// the relay-fault window's default when nobody answers it (MVS-D-76).
func (d *radioDeck) readSynth() {
	d.mu.Lock()
	ref, gen := d.ref, d.gen
	d.mu.Unlock()
	d.needsRead(ref, "the relay was silent", gen)
}

// escalate raises the fault window for a schedule that has stopped (DR-21).
//
// ONE WINDOW FOR BOTH FAULTS. A silent relay and a schedule with nothing left
// are different causes with one consequence — the station is quiet — and one
// surface for that is what keeps the window meaningful. A second error modal
// would be a second thing to learn and a second thing to dismiss.
func (d *radioDeck) escalate(reason string) {
	// A STATION WITH NO AUDIO IS A SUPPORTED CONFIGURATION, and this is the one
	// channel that tells a listener the station has gone quiet — so a nil deck
	// must return, not dereference (red team 2026-09-05, I-1). buildDirector
	// treats a nil deck as fine and nine call sites guard it; startSchedule's
	// escalate closure did not, while the SAME function guards it for cutTo
	// twenty-eight lines later. The invariant in newExecutors could not see it:
	// the closure is non-nil while the channel behind it is dead. The pump then
	// contained the panic and Escalate names no card, so no Failed was emitted
	// and nothing anywhere said the station had stopped — verbatim the outcome
	// DR-21 exists to remove. Same idiom and same reason as mastercontrol.cue.
	if d == nil || d.p == nil {
		return
	}
	radioDebugLog("schedule:escalate:" + reason)
	// The candidates are whatever the current tune still offers. A schedule that
	// stopped for a reason unrelated to the bed leaves none, and the window then
	// shows the fall-through alone — which is the honest answer: read the
	// report, because there is nothing else to tune to.
	d.p.Send(tty.RelaySilentMsg{Candidates: d.silentCandidates(d.engine.Status().Mount)})
}

// alertTonePCM is a class's attention signal, whole.
//
// A class sounds its ratified preset (MVS-D-26) — and, since #18, sounds it a
// ratified NUMBER OF TIMES. An evacuation order is three dual tones where a
// warning is one, so a listener with no screen hears how many and knows what
// kind of thing is coming before a word is spoken.
func alertTonePCM(class cast.Class) []byte {
	one := synth.AlertTone(synth.PresetByName(cast.ToneName(class)), synth.ToneRate)
	n := cast.ToneRepeats(class)
	if n <= 1 || len(one) == 0 {
		return one
	}
	out := make([]byte, 0, len(one)*n)
	for range n { // bounded by the class's ratified count (P10-02)
		out = append(out, one...)
	}
	return out
}
