// RunDashboard wires the live pipeline (B3): config locations → NWS provider
// → tiered scheduler → published snapshots streamed into the TTY dashboard.
// M1 instrumentation: the time from start to the first fully-populated
// snapshot is logged to stderr on exit when WATCHPOST_DEBUG_TIMING=1.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// RunDashboard starts the live TUI. Returns after the user quits. openSetup
// opens the Setup window at once (`watchpost setup`); it also opens itself
// on a first run or an empty watchlist (UAT 100).
func RunDashboard(version string, openSetup bool) error {
	startDebugProfiles()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	refs := refsFromConfig(cfg.Locations)
	openSetup = openSetup || cfg.FirstRun || len(refs) == 0
	// RatePerSec 30 (default 5): the per-location seed pipeline fires ~75
	// calls at launch - at 5/s they trickled in over 15s+ (UAT 6.3). 30/s
	// drains the burst in ~2.5s and is still polite to NWS; steady-state
	// request volume is near zero either way.
	client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, CacheDir: cacheDir()})
	if err != nil {
		return err
	}
	provider := nws.New(client, "")

	keyOverrides := toKeyMap(cfg.Keys)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	applyThemes(cfg.Theme) // built-ins + user theme files; persisted choice (UAT 53)
	tides, err := newCoops()
	if err != nil {
		return err
	}
	fireProvs, firmsProv := fireProviders(client, cfg) // B5
	lp := &livePipelines{ctx: ctx, provider: provider,
		marine: []snapshot.Provider{nws.NewMarine(provider), ndbc.New(client, ""), tides, coops.NewObs(tides)}, // UAT 29 / 61 / 72
		fire:   fireProvs, firms: firmsProv, rules: fireRules(cfg.Fire)}
	resolver, resolverErr := newResolver(client) // one resolver serves Resolve and Suggest
	model, err := tty.NewDashboard(tty.Config{
		Version: version, KeyOverrides: keyOverrides,
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
	p, stopRadio := attachRadio(model, client, provider, cfg.Voice, tty.ParseRadioMode(cfg.Radio.Mode), lp.fireFor) // B4 / UAT 97 / 114
	defer stopRadio()
	lp.p = p

	start := time.Now()
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
	lp.recent = startRecent(ctx, p, lp.providers(), restoreRecent(refsFromConfig(cfg.Recent), refs, seedRecent(refs, recentCap), recentCap))
	lp.markFIRMS() // unkeyed FIRMS reads "off" in the API status, not "ok" (UAT 100)
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

// publisher coalesces "new data" notifications into snapshots (UAT 74):
// a burst of provider completions — 200 at launch — becomes one snapshot
// per publishCoalesce window instead of one per completion. Snapshot() is
// the expensive step (deep copy + harmonize + sun times for 60 locations
// under the assembler lock); computing it once per window removed both
// the launch CPU spike and the 140-thread pile-up behind that lock.
type publisher struct {
	mu      sync.Mutex
	pending bool
	run     func() // takes the snapshot and delivers it
}

// publishCoalesce is the window: short enough that rows still fill "as
// they land", long enough to fold a burst.
const publishCoalesce = 50 * time.Millisecond

// Trigger schedules a publish; further triggers inside the window fold in.
func (pb *publisher) Trigger() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if pb.pending {
		return
	}
	pb.pending = true
	time.AfterFunc(publishCoalesce, func() {
		pb.mu.Lock()
		pb.pending = false
		pb.mu.Unlock()
		pb.run()
	})
}

// pipeline is the priority pipeline: one assembler, one batched scheduler.
type pipeline struct {
	asm *snapshot.Assembler
	s   *sched.Scheduler
	pub *publisher
}

// update reconciles the watchlist in place (UAT 69): kept favourites keep
// their data, newcomers fetch at once, and the new order publishes now.
func (pl *pipeline) update(refs []snapshot.LocationRef) {
	pl.asm.SetLocations(refs)
	pl.s.Update(refs)
	pl.pub.Trigger()
}

func (pl *pipeline) stop() { pl.s.Stop() }

// startPriority wires the priority pipeline (fast cadences). onPublish (may
// be nil) observes each publish before it is sent — the M1 timer rides it.
func startPriority(ctx context.Context, p *tea.Program, providers []snapshot.Provider, refs []snapshot.LocationRef, onPublish func(*snapshot.Snapshot)) *pipeline {
	asm := newAssembler(refs, providers)
	pub := &publisher{run: func() {
		snap := asm.Snapshot()
		if onPublish != nil {
			onPublish(snap)
		}
		p.Send(tty.SnapshotMsg{Snap: snap})
	}}
	s, err := sched.New(sched.Config{
		Clock: sched.RealClock{}, Assembler: asm, Locations: refs,
		Providers: providers,
		Tiers: []sched.Tier{
			{Kind: snapshot.KindAlerts, Every: 20 * time.Second},
			{Kind: snapshot.KindObs, Every: 90 * time.Second},
			{Kind: snapshot.KindMarineObs, Every: 10 * time.Minute}, // buoys + gauges: NDBC files turn over every 30 min (UAT 72)
			{Kind: snapshot.KindForecast, Every: 30 * time.Minute},
			{Kind: snapshot.KindForecastHourly, Every: 30 * time.Minute}, // fires with the daily tier (UAT 72)
			{Kind: snapshot.KindMarine, Every: 30 * time.Minute},         // coastal waters (UAT 29); one gridpoint download serves it and the daily fill
			{Kind: snapshot.KindFire, Every: 10 * time.Minute},           // HMS archive refresh + FIRMS cadence (B5, AI-3)
		},
		OnPublish: pub.Trigger,
	})
	if err != nil {
		_ = invariant.Check(false, "priority scheduler misconfigured: "+err.Error())
		return nil
	}
	s.Start(ctx)
	return &pipeline{asm: asm, s: s, pub: pub}
}

// coopsRatePerSec paces CO-OPS on its own token bucket (UAT 64) so tide
// traffic never competes with NWS/NDBC for slots, and stays polite: 30
// concurrent prediction calls were answered 200 in probing, while repeated
// station-list downloads drew 403s. The 60-location launch fires ~150 tide
// calls once (memoized per station afterwards) — ~30 s at 5/s; the
// priority batch reserves its slots up front and lands in seconds.
const coopsRatePerSec = 5

// newCoops builds the tides provider on its paced client (shared by the
// dashboard and report modes — single owner of the pacing knob).
func newCoops() (*coops.Provider, error) {
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: coopsRatePerSec, CacheDir: cacheDir()})
	if err != nil {
		return nil, err
	}
	return coops.New(c, ""), nil
}

// attachRadio wires the player (B4): the model needs the player and the
// player needs the program to send status back, so the deck is attached
// to the model first and given the program after. Returns the program and
// a stop func (a no-op when the deck could not be built).
func attachRadio(model tty.Dashboard, client *httpx.Client, provider *nws.Provider, voice string, mode tty.RadioMode, fire func(snapshot.LocationRef) synth.FireReport) (*tea.Program, func()) {
	deck := newRadioDeck(nil, client, provider, render.UnitF)
	if deck == nil {
		return tea.NewProgram(model), func() {}
	}
	deck.voiceID, deck.pref, deck.fire = voice, mode, fire
	deck.persistMode = saveRadioMode                                                                                                                  // UAT 97: [m] is a saved preference, like the voice
	model = model.WithRadio(deck).WithRadioMode(mode).WithSpectrum(deck.Spectrum).WithVoices(deck.Voices, deck.VoiceName(), func(name string) error { // UAT 84 / 92 / 97
		deck.SetVoice(name)
		return savePreference(func(cfg *config.Config) { cfg.Voice = name })
	}, deck.PreviewVoice) // UAT 86
	p := tea.NewProgram(model)
	deck.p = p
	return p, deck.Stop
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

// applyThemes registers user theme files (<config dir>/themes/*.json,
// {"tokens": {"temp.hi": "208", ...}}; unlisted tokens inherit the default)
// and activates the persisted choice (UAT 53).
func applyThemes(chosen string) {
	if path, err := config.Path(); err == nil {
		loadUserThemes(filepath.Join(filepath.Dir(path), "themes"))
	}
	if chosen != "" && !render.SetTheme(chosen) {
		_ = invariant.Check(false, "configured theme not found: "+chosen+" (falling back to "+render.DefaultThemeName+")")
	}
}

// loadUserThemes registers every readable theme file in dir; a bad file is
// skipped with a dev-visible invariant, never a startup failure.
func loadUserThemes(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // no user themes
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			continue
		}
		var doc struct {
			Tokens map[string]string `json:"tokens"`
		}
		if jerr := json.Unmarshal(raw, &doc); jerr != nil || len(doc.Tokens) == 0 {
			_ = invariant.Check(false, "theme file unreadable or empty: "+e.Name())
			continue
		}
		over := make(map[render.Token]string, len(doc.Tokens))
		for k, v := range doc.Tokens {
			over[render.Token(k)] = v
		}
		render.RegisterTheme(strings.TrimSuffix(e.Name(), ".json"), over)
	}
}

// setThemeHook activates a theme live and persists the choice.
func setThemeHook(name string) error {
	if !render.SetTheme(name) {
		return fmt.Errorf("unknown theme %q", name)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Theme = name
	return config.Save(cfg)
}

// resolveHook adapts the offline-first resolver (embedded geodata, online
// geocoder fallback) for the dashboard's search modal (UAT 26).
func newResolver(client *httpx.Client) (*locations.Resolver, error) {
	idx, err := geodata.Load()
	if err != nil {
		return nil, fmt.Errorf("location data unavailable: %w", err)
	}
	r, err := locations.New(idx, openmeteo.NewGeocoder(client, ""))
	if err != nil {
		return nil, fmt.Errorf("resolver unavailable: %w", err)
	}
	return r, nil
}

// resolveHook turns a typed query into a ref; a resolver that failed to
// build answers every query with that reason (actionable, never a panic).
func resolveHook(r *locations.Resolver, buildErr error) func(string) (snapshot.LocationRef, error) {
	if buildErr != nil {
		return func(string) (snapshot.LocationRef, error) { return snapshot.LocationRef{}, buildErr }
	}
	return func(query string) (snapshot.LocationRef, error) {
		rctx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		ref, _, err := r.Resolve(rctx, query)
		if err != nil {
			return snapshot.LocationRef{}, err
		}
		if ref.Tag == "" {
			ref.Tag = deriveTag(ref.Label)
		}
		return ref, nil
	}
}

// startDebugProfiles serves Go's runtime profiles on 127.0.0.1:6060 when
// WATCHPOST_DEBUG_PPROF=1 (UAT 73/74): threadcreate, goroutine, heap —
// the way to read a live process rather than guess. Loopback only; off by
// default; never in release notes as a feature.
func startDebugProfiles() {
	if os.Getenv("WATCHPOST_DEBUG_PPROF") != "1" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	go func() { _ = http.ListenAndServe("127.0.0.1:6060", mux) }()
}

// reportTiming prints the M1 launch->full-view measurement when
// WATCHPOST_DEBUG_TIMING=1 (split from RunDashboard, P10-04).
func reportTiming(firstFull time.Duration) {
	if os.Getenv("WATCHPOST_DEBUG_TIMING") != "1" {
		return
	}
	if err := invariant.Check(firstFull > 0, "M1 timer never fired — no fully-populated snapshot"); err != nil {
		fmt.Fprintln(os.Stderr, "watchpost timing:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "watchpost timing: M1 launch->full view = %s (target warm<=3s cold<=8s)\n", firstFull.Round(10*time.Millisecond))
}

// refsFromConfig is the config → request-identity conversion (one builder;
// favourites and the recent stack both ride it, UAT 96).
func refsFromConfig(locs []config.Location) []snapshot.LocationRef {
	refs := make([]snapshot.LocationRef, 0, len(locs))
	for _, l := range locs {
		refs = append(refs, snapshot.LocationRef{Label: l.Label, Tag: l.Tag, Zip: l.Zip, Lat: l.Lat, Lon: l.Lon, TZ: l.TZ})
	}
	return refs
}

// configLocations is the reverse conversion; a missing tag is derived.
func configLocations(refs []snapshot.LocationRef) []config.Location {
	out := make([]config.Location, 0, len(refs))
	for _, r := range refs {
		if r.Tag == "" {
			r.Tag = deriveTag(r.Label)
		}
		out = append(out, config.Location{Label: r.Label, Tag: r.Tag, Zip: r.Zip, Lat: r.Lat, Lon: r.Lon, TZ: r.TZ})
	}
	return out
}

// restoreRecent rebuilds the RECENT / SEARCHED stack at launch (UAT 96):
// the saved stack first (newest first, as saved; anything now a favourite
// drops out), then the seeds fill the room below, deduped by zip, capped.
func restoreRecent(saved, watch, seeds []snapshot.LocationRef, n int) []snapshot.LocationRef {
	used := make(map[string]bool, len(watch))
	for _, r := range watch {
		used[r.Zip] = true
	}
	out := make([]snapshot.LocationRef, 0, n)
	for _, r := range append(append([]snapshot.LocationRef(nil), saved...), seeds...) {
		if used[r.Zip] || len(out) == n {
			continue
		}
		used[r.Zip] = true
		out = append(out, r)
	}
	return out
}

// seedRecent builds the top-N major-city refs for the RECENT/SEARCHED table
// (UAT session 2A: prepopulate so the table is judgeable before real search
// history exists), skipping zips already configured as priority locations.
func seedRecent(priority []snapshot.LocationRef, n int) []snapshot.LocationRef {
	idx, err := geodata.Load()
	if err != nil {
		return nil // the seed list is a nicety; the dashboard renders without it
	}
	used := make(map[string]bool, len(priority))
	for _, r := range priority {
		used[r.Zip] = true
	}
	out := make([]snapshot.LocationRef, 0, n)
	for _, ref := range locations.Seeds(idx, n+len(priority)) {
		if used[ref.Zip] || len(out) == n {
			continue
		}
		ref.Tag = deriveTag(ref.Label)
		out = append(out, ref)
	}
	return out
}

// recentCap is the RECENT / SEARCHED list size (UAT 48: 10 favourites +
// 50 most-recent = 60 tracked locations). The launch burst grows with it
// (~5 calls per location); UAT 59 landed bounded parallel fetch, singleflight
// points resolution and a shared NDBC station cache — the warm-launch disk
// cache is the mitigation still queued.
const recentCap = 50

// recentStartDelay holds the seed pipeline back just long enough for the
// priority pipeline to own the first second (M1 warm budget).
const recentStartDelay = time.Second

// recentPipeline feeds the RECENT/SEARCHED list: one shared assembler,
// one scheduler per location so each row publishes the moment its own
// fetches land (UAT 5.1); the launch burst is ~5 requests per location
// once (a disk cache for warm launches is the queued "NWS cache refresh").
type recentPipeline struct {
	ctx       context.Context
	asm       *snapshot.Assembler
	providers []snapshot.Provider
	alerts    *sched.Scheduler // ONE batched alerts call for the whole list (UAT 72)
	newFor    func(ref snapshot.LocationRef) *sched.Scheduler
	publish   func()

	mu      sync.Mutex // guards scheds and started (red-team 0.9.0 C-6: the staggered starter and a commit could touch the map together)
	scheds  map[snapshot.LocationKey]*sched.Scheduler
	started bool // the staggered start has run: newcomers start themselves from now on
}

// recentAlertsEvery is the RECENT list's alert cadence: one batched
// /alerts/active call covering every recent zone (was 50 per-location
// calls every 2 minutes — 25 of the app's ~40 requests per minute, UAT 72).
const recentAlertsEvery = 2 * time.Minute

// update reconciles the list in place (UAT 69): removed locations stop
// their schedulers, newcomers get one (their first cycle fetches at once),
// kept rows keep their data, and the new order publishes immediately.
func (rp *recentPipeline) update(refs []snapshot.LocationRef) {
	if rp.asm == nil {
		return // nothing was seeded; the list is a nicety
	}
	added, removed := rp.asm.SetLocations(refs)
	rp.mu.Lock()
	for _, r := range removed {
		if s := rp.scheds[snapshot.Key(r)]; s != nil {
			go s.Stop() // Stop waits for in-flight fetches; never on the commit path
			delete(rp.scheds, snapshot.Key(r))
		}
	}
	for _, r := range added {
		if s := rp.newFor(r); s != nil {
			rp.scheds[snapshot.Key(r)] = s
			if rp.started {
				s.Start(rp.ctx) // before that, the staggered starter picks it up
			}
		}
		go rp.hydrateHourly(r) // a looked-up location is being looked at: it earns its hourly now
	}
	rp.mu.Unlock()
	if rp.alerts != nil {
		rp.alerts.Update(refs) // newcomers' alerts fetch at once; the batch continues on cadence
	}
	rp.publish()
}

// hydrateHourly fetches the hourly forecast for one RECENT location on
// demand (UAT 72) and publishes; safe to call from any goroutine.
func (rp *recentPipeline) hydrateHourly(ref snapshot.LocationRef) {
	if rp.asm == nil {
		return
	}
	for _, pr := range rp.providers {
		if !sched.Serves(pr, snapshot.KindForecastHourly) {
			continue
		}
		if frag, err := pr.Fetch(rp.ctx, snapshot.FetchReq{Kind: snapshot.KindForecastHourly, Locations: []snapshot.LocationRef{ref}}); err == nil {
			rp.asm.Apply(frag)
		}
	}
	rp.publish()
}

func (rp *recentPipeline) stop() {
	rp.mu.Lock()
	scheds := make([]*sched.Scheduler, 0, len(rp.scheds))
	for _, s := range rp.scheds {
		scheds = append(scheds, s)
	}
	rp.mu.Unlock() // a commit still in flight at quit must not race the map (round 2 N-6)
	for _, s := range scheds {
		s.Stop()
	}
	if rp.alerts != nil {
		rp.alerts.Stop()
	}
}

// hydrate is the dashboard's Hydrate hook.
func (lp *livePipelines) hydrate(ref snapshot.LocationRef) { lp.recent.hydrateHourly(ref) }

// startRecent wires the slow-cadence background pipeline that feeds live
// weather to the seeded RECENT/SEARCHED list. The seed snapshot publishes
// once the program loop is up (names/zips render instantly; temps stream
// in). An empty seed list yields an inert pipeline.
func startRecent(ctx context.Context, p *tea.Program, providers []snapshot.Provider, refs []snapshot.LocationRef) *recentPipeline {
	rp := &recentPipeline{ctx: ctx, providers: providers, scheds: map[snapshot.LocationKey]*sched.Scheduler{}}
	if len(refs) == 0 {
		rp.publish = func() {}
		return rp
	}
	rp.asm = newAssembler(refs, providers)
	pub := &publisher{run: func() {
		// Always publish the SHARED assembler's view: every scheduler's
		// progress lands in one snapshot regardless of which one cycled.
		p.Send(tty.RecentSnapshotMsg{Snap: rp.asm.Snapshot()})
	}}
	rp.publish = pub.Trigger
	rp.newFor = func(ref snapshot.LocationRef) *sched.Scheduler {
		s, err := sched.New(sched.Config{
			Clock: sched.RealClock{}, Assembler: rp.asm, Locations: []snapshot.LocationRef{ref},
			Providers: providers,
			Tiers: []sched.Tier{ // no alerts (batched across the list) and no hourly (hydrated on demand) — UAT 72
				{Kind: snapshot.KindObs, Every: 10 * time.Minute},
				{Kind: snapshot.KindMarineObs, Every: 10 * time.Minute},
				{Kind: snapshot.KindForecast, Every: time.Hour},
				{Kind: snapshot.KindMarine, Every: time.Hour},
				{Kind: snapshot.KindFire, Every: 15 * time.Minute}, // the archive is shared through the client cache (B5)
			},
			OnPublish: rp.publish,
		})
		if err != nil {
			// A misassembled seed pipeline never blocks the dashboard: surface
			// the wiring bug loudly in dev, drop the nicety in release.
			_ = invariant.Check(false, "recent seed scheduler misconfigured: "+err.Error())
			return nil
		}
		return s
	}
	for _, ref := range refs {
		if s := rp.newFor(ref); s != nil {
			rp.scheds[snapshot.Key(ref)] = s
		}
	}
	if s, err := sched.New(sched.Config{Clock: sched.RealClock{}, Assembler: rp.asm, Locations: refs, Providers: providers,
		Tiers: []sched.Tier{{Kind: snapshot.KindAlerts, Every: recentAlertsEvery}}, OnPublish: rp.publish}); err == nil {
		rp.alerts = s
	} else {
		_ = invariant.Check(false, "recent alerts scheduler misconfigured: "+err.Error())
	}
	go func() {
		// p.Send blocks until Run starts - never call it pre-Run on the main
		// goroutine (caught by the cmd test hang).
		rp.publish()
		select {
		case <-ctx.Done():
			return
		case <-time.After(recentStartDelay):
			// Staggered (UAT 74): 50 schedulers starting in the same instant
			// made a 200-goroutine burst that cost ~90 OS threads; 10 ms apart
			// spreads the launch over half a second with no visible delay.
			rp.mu.Lock()
			toStart := make([]*sched.Scheduler, 0, len(rp.scheds))
			for _, s := range rp.scheds {
				toStart = append(toStart, s)
			}
			rp.started = true // from here a commit's newcomers start themselves
			rp.mu.Unlock()
			for _, s := range toStart {
				s.Start(ctx)
				time.Sleep(recentStartStagger)
			}
			if rp.alerts != nil {
				rp.alerts.Start(ctx)
			}
		}
	}()
	return rp
}

// recentStartStagger spaces the recent schedulers' starts.
const recentStartStagger = 10 * time.Millisecond

// fullyPopulated reports whether every location has current conditions —
// the M1 "fully-populated multi-location view" definition (brief M1).
func fullyPopulated(s *snapshot.Snapshot) bool {
	if len(s.Locations) == 0 {
		return false
	}
	for _, loc := range s.Locations {
		if loc.Harmonized.Source.Provider == "" {
			return false
		}
	}
	return true
}

// toKeyMap converts the config [keys] table to a term.KeyMap override layer
// (Help text comes from the defaults — B3 ledger item: preserve Help on
// override merge lands with the full keymap config work).
func toKeyMap(keys map[string][]string) term.KeyMap {
	if len(keys) == 0 {
		return nil
	}
	out := term.KeyMap{}
	for action, bindings := range keys {
		out[term.Action(action)] = term.Binding{Keys: bindings}
	}
	return out
}
