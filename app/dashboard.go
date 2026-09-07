// RunDashboard wires the live pipeline (B3): config locations → NWS provider
// → tiered scheduler → published snapshots streamed into the TTY dashboard.
// M1 instrumentation: the time from start to the first fully-populated
// snapshot is logged to stderr on exit when WATCHPOST_DEBUG_TIMING=1.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// Options are the dashboard's launch switches.
type Options struct {
	OpenSetup bool // open the Setup window at once (`watchpost setup`); it also opens itself on a first run or an empty watchlist (UAT 100)
	ASCII     bool // --ascii: row marks and legend in their ASCII forms (A11-10)
}

// RunDashboard starts the live TUI. Returns after the user quits.
func RunDashboard(version string, opt Options) error {
	start := time.Now()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	refs := refsFromConfig(cfg.Locations)
	openSetup := opt.OpenSetup || cfg.FirstRun || len(refs) == 0
	// MaxRetries 1 (quality pass Q1, plan §2.3): the scheduler already
	// rehydrates at 10/20/40 s, so this client keeps one retry for the
	// sub-second heal of a blip; the per-host memo bounds an outage. The
	// rest of the policy lives in newDataClient (Q5: one owner).
	client, err := newDataClient(1)
	if err != nil {
		return err
	}
	provider := nws.New(client, "")

	keyOverrides := toKeyMap(cfg.Keys)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	applyThemes(cfg.Theme) // built-ins + user theme files; persisted choice (UAT 53)
	tides, tidesClient, err := newCoops()
	if err != nil {
		return err
	}
	fireProvs, firmsProv := fireProviders(client, cfg) // B5
	lp := &livePipelines{ctx: ctx, provider: provider,
		marine: []snapshot.Provider{nws.NewMarine(provider), ndbc.New(client, ""), tides, coops.NewObs(tides)}, // UAT 29 / 61 / 72
		fire:   fireProvs, firms: firmsProv, rules: fireRules(cfg.Fire),
		seismic: seismicProviders(client, cfg),
		clients: []*httpx.Client{client, tidesClient}, weather: provider, tides: tides}
	lp.attachDiagnostics(ctx, start)
	idx, idxErr := geodata.Load()                             // ONCE: the resolver and the seed list share it (Q3, L1-F21/L4-F5)
	resolver, resolverErr := newResolver(client, idx, idxErr) // one resolver serves Resolve and Suggest
	lp.unknownKeys = append([]string(nil), cfg.Unknown...)
	prefs, setRadius := tickerState(cfg) // 0.12.0: the shared mute + alert-radius state and the radius persist hook
	model, err := tty.NewDashboard(lp.ttyConfig(version, opt, openSetup, cfg, keyOverrides, resolver, resolverErr, firmsProv, setRadius, uiHook(prefs.clock)))
	if err != nil {
		return err // e.g. a '?' rebind in [keys] — actionable from term.Merge
	}
	p, deck, stopRadio := attachRadio(model, client, provider, cfg, tty.ParseRadioMode(cfg.Radio.Mode), lp.fireFor, lp.seismicFor, lp.marineFor) // B4 / UAT 97 / 114 / P4 / 0.14.0
	defer stopRadio()
	lp.attachDeck(ctx, p, deck, client, prefs, version, start)

	firstFullNanos := lp.startPipelines(ctx, p, refs, idx, cfg, client, prefs, start)
	// Cancel BEFORE waiting (red-team 0.9.0 C-2): stopAll waits for every
	// in-flight fetch, and a quit during the launch burst or a slow network
	// would otherwise sit through pacing waits and retries.
	defer func() { cancel(); lp.stopAll() }()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("dashboard failed: %w", err)
	}
	reportTiming(time.Duration(firstFullNanos.Load()))
	return nil
}

// tickerState builds the shared [M] mute flag and the alert-radius value, each
// seeded from config, with the hook the UI calls to change and persist it
// (0.12.0) — one owner so RunDashboard stays within its statement budget.
func tickerState(cfg config.Config) (tickerPrefs, func(int)) {
	radius, radiusHook := tickerRadiusState(cfg.TickerRadiusMi)
	clock := &atomic.Int32{}
	clock.Store(int32(render.ClockByKey(cfg.Clock)))
	return tickerPrefs{muted: tickerMuteState(), radius: radius, clock: clock, updateCheck: cfg.UpdateCheck}, radiusHook
}

// uiHook is what Settings calls when it closes: it persists the display
// preferences AND updates the clock the running ticker pipeline reads.
//
// Two places, like [M]'s hook. The ticker builds its tape in this package, so a
// tape that kept the launch-time clock would be the one surface in the app still
// writing times the way they were written before the setting existed.
func uiHook(clock *atomic.Int32) func(tty.UIPrefs) error {
	return func(u tty.UIPrefs) error {
		clock.Store(int32(render.ClockByKey(u.Clock)))
		return setUIHook(u)
	}
}

// tickerPrefs are the LIVE preferences the ticker pipeline reads and Settings
// changes underneath it: the [M] mute, the alert radius, and the listener's
// clock.
//
// They travel together because they are one thing — a preference a running
// pipeline must see change — and because threading a third atomic through two
// signatures was one too many. The clock joined them at 0.14.0: the ticker
// builds its tape in this package, so a tape that kept the launch-time clock
// would be the one surface left writing times the old way.
type tickerPrefs struct {
	muted  *atomic.Bool
	radius *atomic.Int64
	clock  *atomic.Int32
	// updateCheck is the listener's opt-in to the hourly release check. Read
	// once at wiring: it decides whether the poller exists at all.
	updateCheck bool
}

// [M]'s PERSIST HOOK WAS HERE, and is deleted (red team 2026-09-05, C-1).
//
// It flipped the tone mode, the deck's live tone state and the file, and it was
// reached through tty.Config.MuteTicker — a field declared and never called.
// MVS-D-48 retired the one-key toggle when [M] became a deep link into the
// Settings tone rows, and the call site went with it; the hook has been dead
// since before this release opened. Settings persists the tone rows through
// saveTones, which is the same three places by the path that is actually taken.
//
// startPipelines launches the priority, recent, and ticker pipelines and marks
// the FIRMS status, returning the CAS-guarded first-full-snapshot timer the
// caller reports on exit (extracted so RunDashboard stays within the P10-04
// statement budget after the 0.12.0 ticker wiring).
func (lp *livePipelines) startPipelines(ctx context.Context, p *tea.Program, refs []snapshot.LocationRef, idx *geodata.Index, cfg config.Config, client *httpx.Client, prefs tickerPrefs, start time.Time) *atomic.Int64 {
	firstFullNanos := &atomic.Int64{} // written by concurrent tier publishes (race-fixed)
	// The favourites ride the client's priority lane (UAT 64): their
	// requests never queue behind the seed pipeline's launch burst.
	lp.severe = newSevereDeck(p.Send) // 0.13.0: the window's index — fed by the ticker cycle and by both publishers' hooks (the snapshot each is about to send)
	lp.severe.radius = prefs.radius   // 0.14.0: ALERTS - EVENTS scopes the WINDOW too, not only the tape
	lp.mu.Lock()
	lp.priority = startPriority(httpx.WithPriority(ctx), p, lp.providers(), refs, func(snap *snapshot.Snapshot) {
		// M1: CAS records only the first fully-populated publish.
		if fullyPopulated(snap) {
			firstFullNanos.CompareAndSwap(0, int64(time.Since(start)))
		}
		lp.severe.SetLocations(0, snap)
	})
	// UAT 48: 50 most-recent; UAT 96: the saved stack comes back on top, the seeds fill below.
	lp.recent = startRecent(ctx, p, lp.providers(), restoreRecent(refsFromConfig(cfg.Recent), refs, seedRecent(idx, refs, tty.RecentCap), tty.RecentCap), func(snap *snapshot.Snapshot) { lp.severe.SetLocations(1, snap) }) // the tty owns the caps (Q6, L3-F11)
	lp.mu.Unlock()
	lp.markFIRMS()                         // unkeyed FIRMS reads "off" in the API status, not "ok" (UAT 100)
	lp.setWatch(refs)                      // seed the live watchlist the ticker ties events to
	lp.director = lp.buildDirector(p.Send) // 0.13.0: one owner of the voice — the ticker's takeovers and the window's event reads
	lp.scripts = script.New(scriptsDir())  // the spoken lines, by file name; the user's config dir may override them
	if lp.deck != nil {
		lp.deck.composer = synth.Composer{Scripts: lp.scripts} // the broadcast, the fire and seismic reports and the voice preview speak from the same tree
	}
	lp.reader = newEventReader(ctx, lp.director, lp.scripts, lp.severe.Row, p.Send) // a read ends with the app (A-08)
	if lp.deck != nil {
		lp.reader.status, lp.reader.restore = lp.deck.overlay, lp.deck.pushStatus
	}
	lp.ticker = startTicker(ctx, p, client, idx, lp.currentWatch, prefs, lp.director, lp.scripts, lp.severe) // 0.12.0: the ticker ties events to the LIVE watchlist (re-homed on every Commit); 0.13.0: and feeds the severe index
	// The schedule runs from here, over the SAME arbiter and effector the ticker
	// was just given. It drives the live alert rail since T3.10b — so a
	// listener notices nothing; T3.2b is the first thing it owns.
	lp.schedule = startSchedule(ctx, lp.director, lp.scripts, lp.ticker.clock, lp.deck, lp.currentWatch, lp.ticker)
	lp.wireDeckWarnings()
	return firstFullNanos
}

// ttyConfig assembles the dashboard config from the wired pipelines and the
// launch switches — the hook set the TTY reads (extracted so RunDashboard stays
// within the P10-04 length budget after the 0.12.0 ticker wiring).
func (lp *livePipelines) ttyConfig(version string, opt Options, openSetup bool, cfg config.Config, keyOverrides term.KeyMap, resolver *locations.Resolver, resolverErr error, firmsProv *firms.Provider, setRadius func(int), setUI func(tty.UIPrefs) error) tty.Config {
	return tty.Config{
		Version: version, KeyOverrides: keyOverrides, ASCII: opt.ASCII,
		Stats:          lp.ttyStats,        // [S] REQUESTS / DUMPS rows (quality pass Q0)
		NarrateEvent:   lp.narrateEvent(),  // 0.13.0: [space] in the severe window; nil without audio, so the chip mutes (R5-B-04)
		EndEventRead:   lp.endEventRead(),  // 0.14.0 MVS-D-75: closing the window stops the read
		AlertRadiusMi:  cfg.TickerRadiusMi, // 0.12.0: the Setup window's Alert Notification Preference
		SetAlertRadius: setRadius,
		RelayDwell:     lp.relayDwell(),
		SetRelayDwell:  lp.setRelayDwell(),
		RelayLang:      lp.relayLang(),
		TuneRelay:      lp.tuneRelay(),
		ReadReport:     lp.readReport(),
		InjectAlert:    lp.injectHook(),  // F-21b: nil in a release build
		DebugScenarios: debugScenarios(), // and empty with it
		SetRelayLang:   lp.setRelayLang(),
		Resolve:        resolveHook(resolver, resolverErr),
		Suggest:        suggestHook(resolver),
		Setup:          lp.setup, // persist the default location + FIRMS key; key the live provider (UAT 100)
		OpenSetup:      openSetup,
		FIRMSKey:       firmsProv.KeyHint, // the Setup window shows a stored key is there (UAT 111)
		Commit:         lp.commit,         // persist watchlist + reconcile both pipelines (UAT 26/69)
		SetTheme:       setThemeHook,
		SetUI:          setUI,
		Units:          cfg.Units,
		Clock:          cfg.Clock,
		Hydrate:        lp.hydrate,                             // hourly forecast on demand for RECENT rows (UAT 72)
		Credits:        credits(),                              // data-source credits, licence obligations included (UAT 75)
		FireBoldMW:     fireRules(cfg.Fire).BoldFRPMW,          // B5: one owner for the emphasis threshold — the [fire] rules
		SeismicDays:    seismicRules(cfg.Seismic).LookbackDays, // 0.11.0: one owner for the lookback window — the [seismic] rules
	}
}

// attachDeck takes ownership of the player and starts what rides with it: the
// live clock the broadcast reads, this run's launch time, and the hourly
// "is there a newer Watchpost?" check (0.14.0, release.go).
//
// One step because it is one moment — the deck exists, so everything that needs
// it can be wired — and because RunDashboard is at the safety gate's statement
// ceiling and each of these on its own line is a line it does not have.
func (lp *livePipelines) attachDeck(ctx context.Context, p *tea.Program, deck *radioDeck, client *httpx.Client, prefs tickerPrefs, version string, start time.Time) {
	deck.clockPref = prefs.clock // one preference, one holder: the tape and the broadcast read the same value
	lp.p, lp.deck = p, deck
	lp.started, lp.release = start, newReleaseWatch(version, prefs.updateCheck)
	lp.release.start(ctx, client)
}

// attachDiagnostics builds the dump and its triggers (quality pass Q0):
// SIGUSR1 on Unix, and the opt-in loopback server everywhere.
func (lp *livePipelines) attachDiagnostics(ctx context.Context, start time.Time) {
	lp.dump = newDumper(userCacheSubdir("profiles"), start, lp.sources, lp.ttyStats)
	startDumpTrigger(ctx, lp.dump)
	startDebugProfiles(lp.dump)
}

// currentWatch is the live watchlist the ticker ties events to (D5) and centres
// the radius filter on — a fresh slice per Commit, so a read is a stable
// snapshot even if the set changes mid-cycle.
func (lp *livePipelines) currentWatch() []snapshot.LocationRef {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.watchRefs
}

// setWatch replaces the live watchlist (launch + every Commit).
func (lp *livePipelines) setWatch(refs []snapshot.LocationRef) {
	lp.mu.Lock()
	lp.watchRefs = append([]snapshot.LocationRef(nil), refs...)
	lp.mu.Unlock()
}

// tickerAlert is the ticker's breaking-news audio — the radio deck, or nil when
// there is no audio (a nil deck: tests, no device).
// scriptsDir is where the user's script overrides live: <config dir>/scripts
// — the same directory as config.toml, a scripts/ folder beside it. "" when
// the config dir cannot resolve (built-in scripts only).
func scriptsDir() string {
	path, err := config.Path()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "scripts")
}

// buildDirector builds the voice arbiter over the radio deck (a silent one
// without audio: the typed-nil deck must become a nil interface), and the
// effector it performs through.
func (lp *livePipelines) buildDirector(send func(tea.Msg)) *director {
	if lp.deck == nil {
		return newDirector(nil, newMastercontrol(nil, send))
	}
	return newDirector(lp.deck, newMastercontrol(lp.deck, send))
}

// wireDeckWarnings lets the radio deck report a down relay directory as a
// radio_unavailable warning in [S] (Q1, PR-9); a nil deck (no audio) is fine.
func (lp *livePipelines) wireDeckWarnings() {
	if lp.deck != nil && lp.priority != nil {
		lp.deck.warn = lp.priority.asm.Warn
	}
}

// coopsRatePerSec paces CO-OPS on its own token bucket (UAT 64) so tide
// traffic never competes with NWS/NDBC for slots, and stays polite: 30
// concurrent prediction calls were answered 200 in probing, while repeated
// station-list downloads drew 403s. The 60-location launch fires ~150 tide
// calls once (memoized per station afterwards) — ~30 s at 5/s; the
// priority batch reserves its slots up front and lands in seconds.
const coopsRatePerSec = 5

// newCoops builds the tides provider on its paced client (shared by the
// dashboard and report modes — single owner of the pacing knob). The
// client is returned too, so its request counters can be read.
func newCoops() (*coops.Provider, *httpx.Client, error) {
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: coopsRatePerSec, MaxRetries: 1, CacheDir: cacheDir()})
	if err != nil {
		return nil, nil, err
	}
	return coops.New(c, ""), c, nil
}

// attachRadio wires the player (B4): the model needs the player and the
// player needs the program to send status back, so the deck is attached
// to the model first and given the program after. Returns the program,
// the deck (nil when it could not be built) and a stop func.
func attachRadio(model tty.Dashboard, client *httpx.Client, provider *nws.Provider, cfg config.Config, mode tty.RadioMode, fire func(snapshot.LocationRef) synth.FireReport, seismic func(snapshot.LocationRef) synth.SeismicReport, marine func(snapshot.LocationRef) synth.MarineReport) (*tea.Program, *radioDeck, func()) {
	deck := newRadioDeck(nil, client, provider, render.UnitF)
	if deck == nil {
		return tea.NewProgram(model), nil, func() {}
	}
	deck.voiceID, deck.pref, deck.fire, deck.seismic, deck.marine = cfg.Voice, mode, fire, seismic, marine
	// The whole cast, from the config: setCast validates it against this host
	// OUTSIDE the deck's lock and memoises the problems for [S] (P4).
	deck.setCast(castLoaded(cfg))
	deck.persistMode = saveRadioMode                                                                                                                  // UAT 97: [m] is a saved preference, like the voice
	model = model.WithRadio(deck).WithRadioMode(mode).WithSpectrum(deck.Spectrum).WithVoices(deck.Voices, deck.VoiceName(), func(name string) error { // UAT 84 / 92 / 97
		deck.SetVoice(name)
		if err := savePreference(func(cfg *config.Config) { cfg.Voice = name }); err != nil {
			return err
		}
		// A saved root is a cast change: re-validate and re-resolve from the
		// file, so [S] and the broadcast agree with what is on disk.
		return reloadCast(deck)
	}, deck.PreviewVoice) // UAT 86
	// The cast reaches the TUI as strings (modes/tty imports no domain): the
	// assignments, the tone state, the class list, and the two save hooks.
	// SetCast re-casts the deck — the listener is waiting to hear it. SetTones
	// deliberately does NOT: [M] and the checkboxes must not disturb a
	// broadcast in flight.
	model = model.WithCast(castView(castLoaded(cfg)), toneView(castLoaded(cfg).Tones), toneClasses(),
		func(v tty.CastView) error { return saveCast(deck, v) },
		func(t tty.ToneState) error { return saveTones(deck, toneFromView(t)) },
		deck.Installed)
	p := tea.NewProgram(model)
	deck.p = p
	return p, deck, deck.Stop
}

// saveCast persists the cast the Setup window shows and re-casts the deck.
//
// It re-reads the config first so THIS platform's half is written over
// whatever is on disk while the other platform's half — and every key this
// build does not know — survives (FR-8, NFR-5).
func saveCast(deck *radioDeck, v tty.CastView) error {
	// `next` ESCAPES THE TRANSACTION by capture, because the deck needs the
	// value the write produced. It is the one caller of the six that cannot be
	// expressed as edit(*Config) error alone — a second signature for one
	// caller would be a second write path, which is what FR-1.1 removes.
	var next cast.Config
	if err := config.Mutate(func(cfg *config.Config) error {
		next = castFromView(v, castLoaded(*cfg))
		*cfg = castToConfig(next, *cfg)
		return nil
	}); err != nil {
		return err
	}
	if deck != nil {
		deck.setCast(next)
	}
	return nil
}

// reloadCast re-reads the config and installs it on the deck. One reload, not a
// field-by-field patch: the file is the source of truth after a save, and
// patching would let the deck and the file disagree about a key this build
// preserved but does not read.
func reloadCast(deck *radioDeck) error {
	if deck == nil {
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	deck.setCast(castLoaded(cfg))
	return nil
}

// saveTones persists the per-class tone mute and tells the deck, WITHOUT a
// recast: [M] must be instant and must not disturb a broadcast in flight — no
// config reload on the deck, no re-resolution, no install pass (Task 2.10).
func saveTones(deck *radioDeck, t cast.Tones) error {
	if deck != nil {
		deck.setTones(t)
	}
	return savePreference(func(cfg *config.Config) {
		cfg.Radio.Tones = config.Tones{Mode: t.Mode, Muted: t.Muted}
		// Save is the one owner of the ticker_muted mirror; nothing here writes it.
	})
}

// saveRadioMode persists the [m] source pick (UAT 97).
func saveRadioMode(mode tty.RadioMode) error {
	return savePreference(func(cfg *config.Config) { cfg.Radio.Mode = mode.Key() })
}

// savePreference loads, edits and saves the config — the one path for a
// persisted UI preference (voice, radio mode).
// savePreference is the no-error convenience over config.Mutate, for the
// preference writes that cannot fail. It is an ADAPTER now, not a second write
// path: it once called itself "the one path" while five siblings bypassed it,
// which is the defect FR-1.1 closes.
func savePreference(edit func(cfg *config.Config)) error {
	return config.Mutate(func(cfg *config.Config) error {
		edit(cfg)
		return nil
	})
}

// cacheDir is the on-disk tier of the HTTP cache (UAT 71): the OS cache
// directory ($XDG_CACHE_HOME or ~/Library/Caches on macOS), never the config
// dir — it holds only public weather products and is safe to delete. ""
// (memory only) when the OS gives us no cache directory.
func cacheDir() string { return userCacheSubdir("http") }

// userCacheSubdir is <OS cache dir>/watchpost/<name> — the one builder for
// the HTTP cache and the voice install (round 2 R2-23); "" when the OS has
// no cache directory (the callers degrade: memory only, no voice).
func userCacheSubdir(name string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "watchpost", name)
}

// newAssembler registers every provider with its attribution line — the
// weather provider is the reference; marine providers are secondaries.
func newAssembler(refs []snapshot.LocationRef, providers []snapshot.Provider) *snapshot.Assembler {
	ids := make([]string, 0, len(providers))
	for _, pr := range providers {
		ids = append(ids, pr.ID())
	}
	asm := snapshot.NewAssembler(refs, ids)
	for _, pr := range providers {
		switch pr.ID() {
		case "nws":
			asm.SetAttribution(pr.ID(), "reference", nws.Attribution)
		case "nws-marine":
			asm.SetAttribution(pr.ID(), "secondary", nws.Attribution)
		case "ndbc":
			asm.SetAttribution(pr.ID(), "secondary", ndbc.Attribution)
		case "coops", "coops-obs":
			asm.SetAttribution(pr.ID(), "secondary", coops.Attribution)
		case "hms":
			asm.SetAttribution(pr.ID(), "secondary", hms.Attribution)
		case "wfigs":
			asm.SetAttribution(pr.ID(), "secondary", wfigs.Attribution)
		case "firms":
			asm.SetAttribution(pr.ID(), "secondary", firms.Attribution)
		}
	}
	return asm
}

// livePipelines reconciles the running pipelines when the watchlist
// changes (UAT 26, incremental since UAT 69): persist, then add/remove
// exactly the changed locations — a lookup is one location's requests,
// never a rebuild. Serialized — commits arrive from tea cmd goroutines.
type livePipelines struct {
	mu  sync.Mutex
	ctx context.Context

	// started is this run's launch, for [S]'s uptime; release is the hourly
	// "is there a newer Watchpost?" check (0.14.0).
	started time.Time
	release *releaseWatch

	// unknownKeys are the [radio.*] keys the config carried that this build
	// does not know (NFR-5). They are PRESERVED on save by platform/config;
	// this is only the reporting half, for [S].
	unknownKeys []string

	p        *tea.Program
	provider snapshot.Provider
	marine   []snapshot.Provider // nws-marine + ndbc + coops (UAT 29 / 61)
	fire     []snapshot.Provider // hms + wfigs + firms (B5)
	firms    *firms.Provider     // the keyed one, so the Setup window can turn it on without a relaunch (UAT 100)
	seismic  []snapshot.Provider // usgs (0.11.0)
	rules    fire.Rules          // the [fire] rings, for the broadcast's fire report (UAT 114)
	priority *pipeline
	recent   *recentPipeline
	ticker   *tickerDeck     // 0.12.0: waited at shutdown so its cache writes settle before teardown
	severe   *severeDeck     // 0.13.0: the severe-events index the window lists
	director *director       // 0.13.0: the voice arbiter (app/director.go)
	scripts  *script.Library // 0.13.0: the spoken lines (domains/radio/script)
	reader   *eventReader    // 0.13.0: [space] in the window
	schedule *schedule       // 0.14.0 T3.2a: the Director, its executors and the pump — running, driving nothing yet

	watchRefs []snapshot.LocationRef // the live watchlist the ticker ties events to; updated on Commit (0.12.0 follow-up)

	// Diagnostics (quality pass Q0): the clients whose counters the [S]
	// modal sums, the typed providers whose memos the dump gauges, the
	// radio deck, and the dumper itself.
	clients []*httpx.Client
	weather *nws.Provider
	tides   *coops.Provider
	deck    *radioDeck
	dump    *dumper
}

// fireFor is the radio deck's fire hook (UAT 114): the location's fire
// state from whichever pipeline carries it (favourites first), with the
// rings and the contributing feeds. Not Known → the report is skipped.
func (lp *livePipelines) fireFor(ref snapshot.LocationRef) synth.FireReport {
	var asms []*snapshot.Assembler
	if lp.priority != nil {
		asms = append(asms, lp.priority.asm)
	}
	if lp.recent != nil {
		asms = append(asms, lp.recent.asm)
	}
	for _, asm := range asms {
		if fs, lat, lon, ok := asm.FireFor(ref); ok { // the narrow read: no snapshot clone per cycle (REVIEW C2)
			return fireReportOf(fs, lat, lon, lp.rules, asm.ProviderStatus(lp.firms.ID()) == snapshot.ProviderOK)
		}
	}
	return synth.FireReport{}
}

// seismicFor is the radio deck's seismic hook (P4): the location's recent
// quakes from whichever pipeline carries it (favourites first). No state → the
// report is skipped.
func (lp *livePipelines) seismicFor(ref snapshot.LocationRef) synth.SeismicReport {
	var asms []*snapshot.Assembler
	if lp.priority != nil {
		asms = append(asms, lp.priority.asm)
	}
	if lp.recent != nil {
		asms = append(asms, lp.recent.asm)
	}
	for _, asm := range asms {
		if ss, lat, lon, ok := asm.SeismicFor(ref); ok { // the narrow read: no snapshot clone per cycle (REVIEW C2)
			return seismicReportOf(ss, lat, lon)
		}
	}
	return synth.SeismicReport{}
}

// markFIRMS mirrors the FIRMS key state into both assemblers' status rows.
func (lp *livePipelines) markFIRMS() {
	if lp.firms == nil {
		return
	}
	off := !lp.firms.Enabled()
	if lp.priority != nil {
		_ = lp.priority.asm.SetInactive(lp.firms.ID(), off)
	}
	if lp.recent != nil {
		_ = lp.recent.asm.SetInactive(lp.firms.ID(), off)
	}
}

// setup is the Setup window's finish (UAT 100): persist, then key the live
// FIRMS provider so fire data upgrades without a relaunch. The watchlist
// itself follows through the Commit hook (the window calls both, in order).
func (lp *livePipelines) setup(def snapshot.LocationRef, firmsKey string) error {
	if err := applySetup(def, firmsKey); err != nil {
		return err
	}
	if firmsKey != "" && lp.firms != nil {
		if err := lp.firms.SetKey(firmsKey); err != nil {
			return err
		}
		lp.markFIRMS()
	}
	return nil
}

// providers is the full provider set: weather reference + marine, fire and
// seismic secondaries.
func (lp *livePipelines) providers() []snapshot.Provider {
	out := append([]snapshot.Provider{lp.provider}, lp.marine...)
	out = append(out, lp.fire...)
	return append(out, lp.seismic...)
}

func (lp *livePipelines) stopAll() {
	lp.mu.Lock()
	if lp.priority != nil {
		lp.priority.stop()
	}
	lp.recent.stop()
	ticker := lp.ticker
	lp.mu.Unlock()
	// The ticker's cycle takes lp.mu (the watchlist tie), so drain it AFTER
	// releasing the lock — waiting under it would deadlock against that tie.
	if ticker != nil {
		ticker.stop()
	}
	if lp.reader != nil {
		lp.reader.End() // a [space] read in progress ends with the app, its goroutine waited for (A-08)
	}
	// LAST, and waited for. The schedule's tick goroutine writes to the pump, so
	// both have an owner here rather than being abandoned to the context — the
	// 0.12.0 lesson, where a fire-and-forget goroutine outlived what it wrote to
	// and turned a release tag red on the Linux race gate.
	lp.schedule.stop()
}

func (lp *livePipelines) commit(watch, recent []snapshot.LocationRef) error {
	if lp.weather != nil {
		lp.weather.Retain(append(append([]snapshot.LocationRef(nil), watch...), recent...)) // the grid cache follows the location set (Q5, L4-F7)
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if err := invariant.Check(len(watch) <= 10, "watchlist cap is 10 (R-4)"); err != nil {
		return err
	}
	// LOCK ORDER: lp.mu -> config's. Held in that order here and nowhere in the
	// reverse, and the edit below must never reach config.Load or config.Save —
	// the config mutex is not reentrant. reloadCast's bare Load is the one to
	// keep out of any edit closure.
	if err := config.Mutate(func(cfg *config.Config) error {
		cfg.Locations = configLocations(watch)
		cfg.Recent = configLocations(recent) // UAT 96: the RECENT stack survives a restart
		return nil
	}); err != nil {
		return err
	}
	// UAT 69: incremental — only the changed locations move; nothing that
	// is already loaded re-requests, and the regular cadences continue.
	if lp.priority != nil {
		lp.priority.update(watch)
	}
	if lp.recent != nil {
		lp.recent.update(recent)
	}
	lp.watchRefs = append([]snapshot.LocationRef(nil), watch...) // re-home the ticker's tie to the new watchlist (under lp.mu, already held)
	return nil
}

// hydrate is the dashboard's Hydrate hook.
func (lp *livePipelines) hydrate(ref snapshot.LocationRef) { lp.recent.hydrateHourly(ref) }

// narrateEvent is the window's [space] hook — a play/PAUSE now (MVS-D-74). The deck is attached AFTER the
// dashboard is built (attachRadio needs the model), so the hook decides at
// the press, not at wiring: with no deck to speak through the press is inert
// — no ▶ mark, no busy reader for a silent record (R5-B-04; VALIDATE
// 2026-08-29 found the wiring-time check had muted the chip for everyone).
// relayDwell is what the Settings window opens showing. Nil-safe: the deck is
// not built in every mode, and a window that cannot show the setting is better
// than one that cannot open.
func (lp *livePipelines) relayDwell() time.Duration {
	if lp.deck == nil {
		return envWatchlistDwell()
	}
	return lp.deck.watchlistDwell()
}

// tuneRelay tunes to one named transmitter — the listener's answer to the
// relay-fault window (MVS-D-76). It goes through the deck's own tune so the
// duck is not lifted: an alert may be reading while they answer.
func (lp *livePipelines) tuneRelay() func(string) {
	return func(callsign string) {
		if lp.deck == nil || callsign == "" {
			return
		}
		lp.deck.tuneCallsign(callsign)
	}
}

// readReport is the fall-through: stop trying relays and read the synthesized
// report for wherever the deck is pointed. It is what the countdown takes, so
// it must work with nobody at the keyboard.
func (lp *livePipelines) readReport() func() {
	return func() {
		if lp.deck == nil {
			return
		}
		lp.deck.readSynth()
	}
}

// relayLang is what the Settings window opens showing.
func (lp *livePipelines) relayLang() string {
	if lp.deck == nil {
		return stream.LangEnglish
	}
	return lp.deck.watchlistLang()
}

// setRelayLang records which language wins a co-located tie. It takes effect on
// the NEXT tune rather than at once: re-tuning mid-sentence to swap languages
// would cut off whatever is being said, and the listener changing this is
// setting a preference, not asking for an interruption.
func (lp *livePipelines) setRelayLang() func(string) {
	return func(lang string) {
		if lp.deck == nil {
			return
		}
		lp.deck.mu.Lock()
		lp.deck.relayLang = lang
		lp.deck.mu.Unlock()
	}
}

// setRelayDwell applies a new rotation interval at once.
//
// IT GOES STRAIGHT TO THE DIRECTOR, through the same SetRepeat the [r] key uses,
// because the Director holds the dwell and a setting it has not been told is a
// setting that does not apply. Re-sending the current repeat mode is how the new
// number reaches it — the deck translates the mode to a dwell in one place, and
// this asks it to do that again with the new value.
func (lp *livePipelines) setRelayDwell() func(time.Duration) {
	return func(d time.Duration) {
		if lp.deck == nil {
			return
		}
		lp.deck.mu.Lock()
		lp.deck.relayDwell = d
		mode, queue := lp.deck.repeat, lp.deck.queue
		lp.deck.mu.Unlock()
		lp.deck.SetRepeat(mode, queue)
	}
}

// endEventRead is the window's close hook: a read belongs to the window that
// started it (MVS-D-75).
func (lp *livePipelines) endEventRead() func() {
	return func() {
		if lp.reader != nil {
			lp.reader.Cancel() // a keypress never waits: the frame redraws now
		}
	}
}

func (lp *livePipelines) narrateEvent() func(string) {
	return func(key string) {
		if lp.deck == nil {
			return
		}
		lp.reader.Toggle(key)
	}
}
