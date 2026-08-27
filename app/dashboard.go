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
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
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
		clients: []*httpx.Client{client, tidesClient}, weather: provider, tides: tides}
	lp.attachDiagnostics(ctx, start)
	idx, idxErr := geodata.Load()                             // ONCE: the resolver and the seed list share it (Q3, L1-F21/L4-F5)
	resolver, resolverErr := newResolver(client, idx, idxErr) // one resolver serves Resolve and Suggest
	model, err := tty.NewDashboard(tty.Config{
		Version: version, KeyOverrides: keyOverrides, ASCII: opt.ASCII,
		Stats:      lp.ttyStats, // [S] REQUESTS / DUMPS rows (quality pass Q0)
		Resolve:    resolveHook(resolver, resolverErr),
		Suggest:    suggestHook(resolver),
		Setup:      lp.setup, // persist the default location + FIRMS key; key the live provider (UAT 100)
		OpenSetup:  openSetup,
		FIRMSKey:   firmsProv.KeyHint, // the Setup window shows a stored key is there (UAT 111)
		Commit:     lp.commit,         // persist watchlist + reconcile both pipelines (UAT 26/69)
		SetTheme:   setThemeHook,
		Hydrate:    lp.hydrate,                    // hourly forecast on demand for RECENT rows (UAT 72)
		Credits:    credits(),                     // data-source credits, licence obligations included (UAT 75)
		FireBoldMW: fireRules(cfg.Fire).BoldFRPMW, // B5: one owner for the emphasis threshold — the [fire] rules
	})
	if err != nil {
		return err // e.g. a '?' rebind in [keys] — actionable from term.Merge
	}
	p, deck, stopRadio := attachRadio(model, client, provider, cfg.Voice, tty.ParseRadioMode(cfg.Radio.Mode), lp.fireFor) // B4 / UAT 97 / 114
	defer stopRadio()
	lp.p, lp.deck = p, deck

	var firstFullNanos atomic.Int64 // written by concurrent tier publishes (race-fixed)
	// The favourites ride the client's priority lane (UAT 64): their
	// requests never queue behind the seed pipeline's launch burst.
	lp.priority = startPriority(httpx.WithPriority(ctx), p, lp.providers(), refs, func(snap *snapshot.Snapshot) {
		// M1: CAS records only the first fully-populated publish.
		if fullyPopulated(snap) {
			firstFullNanos.CompareAndSwap(0, int64(time.Since(start)))
		}
	})
	// UAT 48: 50 most-recent; UAT 96: the saved stack comes back on top, the seeds fill below.
	lp.recent = startRecent(ctx, p, lp.providers(), restoreRecent(refsFromConfig(cfg.Recent), refs, seedRecent(idx, refs, tty.RecentCap), tty.RecentCap)) // the tty owns the caps (Q6, L3-F11)
	lp.markFIRMS()                                                                                                                                        // unkeyed FIRMS reads "off" in the API status, not "ok" (UAT 100)
	lp.wireDeckWarnings()
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

// attachDiagnostics builds the dump and its triggers (quality pass Q0):
// SIGUSR1 on Unix, and the opt-in loopback server everywhere.
func (lp *livePipelines) attachDiagnostics(ctx context.Context, start time.Time) {
	lp.dump = newDumper(userCacheSubdir("profiles"), start, lp.sources, lp.ttyStats)
	startDumpTrigger(ctx, lp.dump)
	startDebugProfiles(lp.dump)
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
func attachRadio(model tty.Dashboard, client *httpx.Client, provider *nws.Provider, voice string, mode tty.RadioMode, fire func(snapshot.LocationRef) synth.FireReport) (*tea.Program, *radioDeck, func()) {
	deck := newRadioDeck(nil, client, provider, render.UnitF)
	if deck == nil {
		return tea.NewProgram(model), nil, func() {}
	}
	deck.voiceID, deck.pref, deck.fire = voice, mode, fire
	deck.persistMode = saveRadioMode                                                                                                                  // UAT 97: [m] is a saved preference, like the voice
	model = model.WithRadio(deck).WithRadioMode(mode).WithSpectrum(deck.Spectrum).WithVoices(deck.Voices, deck.VoiceName(), func(name string) error { // UAT 84 / 92 / 97
		deck.SetVoice(name)
		return savePreference(func(cfg *config.Config) { cfg.Voice = name })
	}, deck.PreviewVoice) // UAT 86
	p := tea.NewProgram(model)
	deck.p = p
	return p, deck, deck.Stop
}

// saveRadioMode persists the [m] source pick (UAT 97).
func saveRadioMode(mode tty.RadioMode) error {
	return savePreference(func(cfg *config.Config) { cfg.Radio.Mode = mode.Key() })
}

// savePreference loads, edits and saves the config — the one path for a
// persisted UI preference (voice, radio mode).
func savePreference(edit func(cfg *config.Config)) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	edit(&cfg)
	return config.Save(cfg)
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
	mu       sync.Mutex
	ctx      context.Context
	p        *tea.Program
	provider snapshot.Provider
	marine   []snapshot.Provider // nws-marine + ndbc + coops (UAT 29 / 61)
	fire     []snapshot.Provider // hms + wfigs + firms (B5)
	firms    *firms.Provider     // the keyed one, so the Setup window can turn it on without a relaunch (UAT 100)
	rules    fire.Rules          // the [fire] rings, for the broadcast's fire report (UAT 114)
	priority *pipeline
	recent   *recentPipeline

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

// providers is the full provider set: weather reference + marine and fire secondaries.
func (lp *livePipelines) providers() []snapshot.Provider {
	return append(append([]snapshot.Provider{lp.provider}, lp.marine...), lp.fire...)
}

func (lp *livePipelines) stopAll() {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.priority != nil {
		lp.priority.stop()
	}
	lp.recent.stop()
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
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Locations = configLocations(watch)
	cfg.Recent = configLocations(recent) // UAT 96: the RECENT stack survives a restart
	if err := config.Save(cfg); err != nil {
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
	return nil
}

// hydrate is the dashboard's Hydrate hook.
func (lp *livePipelines) hydrate(ref snapshot.LocationRef) { lp.recent.hydrateHourly(ref) }
