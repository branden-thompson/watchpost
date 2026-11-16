---
title: "0.19.6 PLAN — design AP: the app split (root, app/internal/deck, the maps package, the kit)"
date: 2026-11-15
phase: PLAN
agent: AP
status: "FILED — checked by the lead against e57afeaf; its choices ruled D-21 to D-26 and A-19 to A-27"
---

# Design AP — the `app` split

> **Filed from PLAN's design agent AP.** Its tables are in `design-data/AP/`. `<scratch>` names the
> session's working directory, which is not committed: the computed graph there is committed in batch 0 as
> its own module (A-12). Where a ruling chose differently from this design, the ruling stands.
> **Only the TSV tables were filed.** Where the text says a script, probe or tool is in `design-data/`,
> it was not: they stayed in the session's scratch directory, and B0.12's committed graph tool
> reproduces the tables before BUILD relies on them (A-29).
> **Names (A-20):** this page's `maps`/`app/internal/maps` is `mapdata` (`mapdata.Feeds`), its kit is
> `app/internal/run`, and `app`'s test helpers go to `app/internal/apptest`, not to CK's package (A-24).
> `mutants-AP.tsv` places mutants by file; AP-10's statement-level counts govern. A-23's "unpinned wiring"
> is withdrawn by A-32.

## Verdict

1. **The split closes with no unexported crossing and no cycle.** It needs one deck entry type with
   10 methods, one maps entry type with 11, and a seven-helper leaf package. Imports run root → deck, maps
   and kit, and deck → kit and maps → kit. The deck and the maps never import each other (AP-8).
2. **D-16's lock needs no `…Locked` form.** The shared mutex is taken **25** times in **5** files, not 28 in 6.
   The three `lp.mu.Lock()` calls in `app/mapwaves.go` belong to `landPoints`' own mutex
   (`app/mapwaves.go:57-61`), on a receiver that happens to be named `lp`. After the split, every
   critical section calls only code in its own package (AP-3). The static check and the deadlock test
   still land in batch 0, to keep it so.
3. **D-14: 64 straddling test functions in 39 files.** 30 change setup only, 14 read the same value
   through production API, and 19 move with their code. **1 test cannot be served:**
   `TestThePropagationClientsWireBytesReachTheCounters`, which last-kills m497 and m498. It is a ruling
   (AP-9).
4. **Mutants: 153 anchor in `app`.** After the split, 116 fall in the deck, 13 in the maps package, 23
   in the root and 1 in a moving test file. 130 change path. 13 change anchor text (11 that move,
   2 that stay) (AP-10).
5. **Three findings that change the settled map's details, each within it:**
   - `app/saychildren.go` and `app/hangup.go` stay at the root (their one user is `RunDashboard`).
   - `stationConfig` splits: 5 of its 11 rows read only the root's station.
   - `timing.go` and `lazymemo.go` go to the maps package, not the kit.
6. **Four choices go to the HUM LEAD:**
   - the GOOS seam (AP-6);
   - the one test that cannot be served (AP-9);
   - three root call lines whose mutants move into a child (R-AP-1);
   - whether the maps package repays its cost (AP-4, recommended keep).

## What I computed and how

Every script and table is in `design-data/AP/`.

**The tools:**

| Tool | What it does |
|---|---|
| `aptool/` (`go/packages`, full type information, tests included) | Lists `livePipelines`' fields and methods (`out/fields.tsv`, `out/methods.tsv`), every typed use of them (`out/uses.tsv`), and every lock section on the struct's own `mu` field object, found by type rather than by name (`out/locks.tsv`, `out/lockcontent.tsv`). Also lists the static call graph (`out/calls.tsv`), the composite literals (`out/literals.tsv`) and every `modes/tty` use by enclosing function (`out/ttyuses.tsv`). |
| `declgraph/` | TS's declaration-graph tool, with one fix for generic receivers (`lazyMemo[K, V]`). Run over `./app` without tests (`dg/`) and with them (`dgT/`). |
| `design.py` | **The design as data.** Assigns every declaration and every `livePipelines` field to a package, and prints the unexported crossings it leaves. It also lists, per split function, the foreign uses the entry points absorb. |
| `closure.py` | Follows the transitive static calls under each lock section. |
| `tests.py`, `tfuncs.py`, `classify.py` | The test reach under the design. |
| `mutants.py` | Places every `app` mutant anchor (`ast` over `06_docs/mutants/*.py`) at its function and destination, and joins A-3's killers, read from `git show 7c5e5772:06_docs/mutants/verdicts/since-v0.19.5.log`. |
| `inl-app.txt` | The inlining report (`go build -gcflags=-m ./app`, one package). |

**The tables:**

| Table | Contents |
|---|---|
| `livepipelines-AP.tsv` | Every field and method of `livePipelines`, with its destination |
| `locks-AP.tsv` | The lock inventory |
| `tests-AP.tsv` | Every straddling test function, classified under D-14 (the deliverable AP-9 names) |
| `tests-reach-AP.tsv`, `tests-funcs-AP.tsv` | The raw reach behind that classification |
| `mutants-AP.tsv` | Every `app` mutant with its file, line, function, destination, killer and occurrence count |

**Not run:** no `go test`, no make, no benchmark, no app. Every figure is static, from code identical to
`e57afeaf` (`git diff e57afeaf HEAD -- '*.go'` is empty).

---

## AP-1 — The deck's entry point (AS-4)

### AP-1.1 Two constraints that shape it

**The prep commit holds every new name inside package `app` (R3).** A move commit may change only paths,
package clauses, imports and qualifiers. So every identifier the deck exports must already have its final
name in the prep commit, while it still sits in package `app`. Three names collide there:

- **`Options` is taken.** `app.Options` is the root's public launch switches (`app/dashboard.go:47`),
  used by `cmd/watchpost/root.go`. So the deck's options type is `deck.Config`, and the maps' is
  `mapdata.Config`.
- **`New` cannot be declared twice in one package.** The deck batch and the maps batch are separate
  (A-13), so the deck's `New` has left `app` before the maps' `New` is declared in it.
- **`RenderCast` is the root's front door.** It is `app.RenderCast` (`app/hostfacts.go:126`), called by
  `cmd/watchpost/root.go:108`, which cannot import `internal/`. The deck's function is renamed in the
  prep commit to `CastBlock` (its doc comment already calls it "the cast block"). The root keeps
  `RenderCast`, which forwards to it; the call gains its `deck.` qualifier in the move commit.

**The order of `RunDashboard` is today's order (R1).** The deck's steps today are interleaved with the
root's and the maps' steps:

- `startPipelines` runs the root's priority and recent start, then the deck's arbiter, then the maps'
  history, then the deck's reader, ticker and schedule (`app/dashboard.go:271-337`).
- `stopAll` runs the root's pipelines, then the deck's ticker and reader, then the maps' propagation
  settle, then the deck's schedule (`:1024-1047`).

To change no order, the deck's start and its stop are each two methods. The alternative is a choice for
the lead (C-AP-4).

### AP-1.2 `deck.Config`, the constructor's options

```go
// package deck (app/internal/deck)

// Config is what the deck is given by the composition root.
type Config struct {
	Mu        *sync.Mutex                            // the root's livePipelines.mu (D-16)
	Loaded    config.Config                          // the configuration RunDashboard loaded
	Client    *httpx.Client                          // the data client
	Provider  *nws.Provider                          // the weather provider
	Index     *geodata.Index                         // the embedded location index
	Watch     func() []snapshot.LocationRef          // the listener's live watchlist
	Pool      func() []snapshot.LocationRef          // what the station's Producer may offer
	Station   func() (snapshot.LocationRef, float64) // the station's transmitter and service radius
	Fire      func(snapshot.LocationRef) synth.FireReport
	Seismic   func(snapshot.LocationRef) synth.SeismicReport
	Marine    func(snapshot.LocationRef) synth.MarineReport
	PersistUI func(tty.UIPrefs) error                // Settings' save of the display preferences
}
```

| Field | Comes from today | Read today by |
|---|---|---|
| `Mu` | `livePipelines.mu`, `app/dashboard.go:802` | the bed's 9 sections, `injectHook`, `rebed`, `rememberRequested`, `resolvable` (AP-3) |
| `Loaded` | `config.Load()`, `app/dashboard.go:83` | `tickerState` `:117`; `giveItAStation`'s bed lines `:196-200`; `attachRadio` `:611-631` |
| `Client` | `newDataClient(1)`, `app/dashboard.go:93` | `attachRadio` → `newRadioDeck` `:607`; `startTicker` `:305` |
| `Provider` | `nws.New`, `app/dashboard.go:97` | `newRadioDeck` `:607` |
| `Index` | `loadGeodata`, `app/dashboard.go:116` | `startTicker` `:305` |
| `Watch` | the method value `lp.currentWatch`, `app/dashboard.go:524` | `startTicker` `:305`; the rail's scope `:326`; `startSchedule` `:335`; `testLocation` `app/inject.go:246` |
| `Pool` | `lp.producer()`, `app/pool.go:267` | `startSchedule` `:335`; `resolvable` `app/pool.go:550` |
| `Station` | a new root method, `stationReach`, beside `currentStation` (`app/pool.go:270`); it returns the station's transmitter and radius | `scopeFor` `app/airscope.go:64`; `refreshBedRelays` `app/bedrelay.go:72` |
| `Fire`, `Seismic`, `Marine` | `lp.fireFor` `app/dashboard.go:927`, `lp.seismicFor` `:939`, `lp.marineFor` `app/marine.go:17` | `attachRadio`'s arguments `:124`, assigned at `:611` |
| `PersistUI` | `setUIHook`, `app/themes.go:82` | `uiHook` `app/dashboard.go:235-240` |

**Why `Station` is a func and not today's `stationArea`.** `stationArea` is the root's type
(`app/pool.go:37`), and 8 root functions read its unexported fields (`dg/uses.tsv`). If it moved down to the
deck (AS-5), its fields would have to be exported. The deck reads only the transmitter and the radius.

**The station seam's costs.**

- The prep commit changes the signatures of `scopeFor` and `refreshBedRelays` (the bodies read two
  values in place of a struct).
- No mutant anchors in either function (`mutants-AP.tsv`).
- `currentStation` is not inlinable today, because its `defer` keeps the compiler from inlining it
  (`inl-app.txt`). The seam therefore removes no inlined call.

### AP-1.3 The constructor, the methods and where each sits in `RunDashboard`

```go
// package deck

// New readies the deck before the console's configuration is built.
func New(c Config) *Deck

// FillConfig fills the deck's rows of the console's configuration.
func (dk *Deck) FillConfig(c *tty.Config)

// Dress builds the radio deck and gives the model its radio, voices and cast,
// returning the model and the player's stop.
func (dk *Deck) Dress(model tty.Dashboard) (tty.Dashboard, func())

// Attach hands the deck the program and builds the severe index.
func (dk *Deck) Attach(p *tea.Program)

// StartVoice builds the arbiter and the scripts and asks for the bed's relays.
func (dk *Deck) StartVoice(ctx context.Context)

// StartDirector starts the event reader, the ticker and the schedule.
func (dk *Deck) StartDirector(ctx context.Context, warn func(snapshot.Warning))

// SetLocations feeds the severe index a tier's publish.
func (dk *Deck) SetLocations(tier int, snap *snapshot.Snapshot)

// StationMoved re-fences the rail and re-resolves the bed after the station moves.
func (dk *Deck) StationMoved(ctx context.Context)

// StopProducer stops the ticker and ends a severe-event read.
func (dk *Deck) StopProducer()

// StopDirector stops the schedule, last.
func (dk *Deck) StopDirector()

// SynthSource is the running synthesized broadcast, for the diagnostics.
func (dk *Deck) SynthSource() interface{ Cached() (int, int) }

// CastBlock is `watchpost report --verbose`'s cast block.
func CastBlock(ctx context.Context) string

// VoiceDir is where the voices are installed.
func VoiceDir() string
```

| Step in `RunDashboard`, today's order | Today | After |
|---|---|---|
| `tickerState(cfg)` | `app/dashboard.go:117` | inside `deck.New`. It moves one step later (pure allocation of three atomics, `:222-227`). |
| `giveItAStation`: station, then bed | `:118` (`:189-201`) | The root keeps the unknown keys, the index and the station (`:190-192`). `deck.New` follows `setStation` and takes the relay table, the bed radius and the kept relay (`:196-200`). |
| `tty.NewDashboard(lp.ttyConfig(…))` | `:120` | `ttyConfig` calls `dk.FillConfig`. It must run while the radio deck is nil, as today: `relayDwell()` and `relayLang()` are read at configuration time and answer the env default and English (`:1133-1138`, `:1165-1170`). |
| `attachRadio`, the deck and the model | `:124`, `:606-634` | `dk.Dress(model)`: lines `:607`, `:611-634`, in order |
| `tea.NewProgram(tty.NewRouter(model))` | `:609` and `:635`, two call sites | **the root, one call site**: the program is the composition root's (`TestTheProgramsModelIsAlwaysTheRouter`, `app/router_owner_test.go:17`, scans the root only) |
| `defer stopRadio()` | `:125` | the root defers the stop `Dress` returned (`deck.Stop`, `:637`) |
| `attachDeck` | `:126`, `:499-511` | The root keeps `:500-502` (program, launch time, release watch). `dk.Attach(p)` then does `:636` (the radio deck's program), the severe index (`:269-270`, a pure allocation, `app/severe.go:79-81`), the nil-radio-deck return `:503-505` and `:506-510`. |
| `startPipelines`, the severe index | `:269-270` | in `Attach`, above |
| the priority and recent pipelines under the lock | `:271-287` | root |
| `markFIRMS`, `setWatch` | `:288-289` | root |
| arbiter, scripts, composer, `go rebed` | `:290-299` | `dk.StartVoice(ctx)` |
| `startHistory` | `:300` | the maps' `StartHistory` |
| reader, ticker, MasterControl's wiring, the rail's scope, the schedule, the warnings | `:301-337` | `dk.StartDirector(ctx, warn)`. The root passes `warn` as the priority assembler's `Warn` (`:565-567`), or nil when there is no priority pipeline. |
| `stopAll` | `:1024-1047` | the root's lock section reads only `priority` and `recent`. Then the root stops the pipelines (`:1032-1035`), then `dk.StopProducer()` (`:1036-1041`), then the maps' `Settle()` (`:1042`), then `dk.StopDirector()` (`:1046`). |

`StopProducer` reads `ticker` under the shared mutex, as `stopAll` does at `:1026`. The value is the
same. `ticker` is written once, without the lock, at `:305`. `stopAll` is deferred at `:132`, after
`startPipelines` returns at `:128`. **This is the only new lock section (AP-3, row 26).**

`SetLocations` serves the two publish callbacks (`:277`, `:286`), which today read `lp.severe` when
called. It forwards to `severeDeck.SetLocations` and is small enough to inline across the package
boundary.

`StationMoved` serves the root's `restationTo`. It carries `app/pool.go:207-219`: MasterControl's
`Refence`, then the nil-context guard and the goroutine start of `rebed`.

### AP-1.4 `FillConfig`'s rows

**`ttyConfig`'s rows (12):**

| Row | `app/dashboard.go` |
|---|---|
| `NarrateEvent` | `:347` |
| `EndEventRead` | `:348` |
| `SetAlertRadius` | `:350`; the hook `tickerState` returns |
| `RelayDwell` | `:351` |
| `SetRelayDwell` | `:352` |
| `RelayLang` | `:358` |
| `TuneRelay` | `:359` |
| `ReadReport` | `:360` |
| `InjectAlert` | `:361` |
| `DebugScenarios` | `:362` |
| `SetRelayLang` | `:363` |
| `SetUI` | `:373`; `uiHook` over `Config.PersistUI` |

**`stationConfig`'s rows (6):**

| Row | `app/dashboard.go` |
|---|---|
| `OnSurface` | `:398` |
| `StepBedRelay` | `:399` |
| `ToggleBedRelay` | `:400` |
| `MoveCard`, `DropCard` | `:404` |
| `RequestCard` | `:406`, its second half |

**`stationConfig` splits; it does not move whole.** AS-4 moved it whole. Its other five rows read only
the root's station:

- `LocateInRadius` (`:406`, over `lookInPool` and `currentStation`);
- `Transmitter` and `SetTransmitter` (`:411`);
- `ServiceRadiusMi` and `SetServiceRadius` (`:412`);
- the two bounds (`:415`, mutant mAS1);
- `StationArea` (`:416-420`).

They stay at the root.

### AP-1.5 The deck's whole exported surface (AS-7, A-12)

| Tier | Names | Count |
|---|---|---|
| **1. Reachable by the root.** Package-level names, and the methods of exported types. | `Deck`, `Config`, `New`, `CastBlock`, `VoiceDir`; `Deck.FillConfig`, `.Dress`, `.Attach`, `.StartVoice`, `.StartDirector`, `.SetLocations`, `.StationMoved`, `.StopProducer`, `.StopDirector`, `.SynthSource`; three types moved as they are, `CastReport`, `CastRow` (`app/hostfacts.go`) and `SourceHealth` (`app/severe.go`), with `CastReport.ToneSummary` | 19 (plus `Config`'s 12 exported fields) |
| **2. Handed out through an interface or a method value.** Exported methods on unexported types. | `radioDeck` as `tty.Radio`: `Tune`, `Stop`, `SetVolume`, `SetRepeat`, `SetMode` (`modes/tty/dashboard.go:479-485`). `radioDeck`'s method values passed by `Dress` to `tty.Dashboard`'s `With…` setters: `Spectrum`, `Voices`, `VoiceName`, `SetVoice`, `PreviewVoice`, `Installed` (`app/dashboard.go:617-634`). `radioDeck` as `cast.Host` (`domains/radio/cast/resolve.go:13`): `Platform`, `Discovered`, `Default`. `mastercontrol` as `tty.Station` (`modes/tty/router.go:137`, handed out in `StationControlMsg`, `app/schedule.go:164`): `GoOnAir`, `GoToStandby`, `CutBed`. `hostFacts` as `cast.Host`: `Platform`, `Discovered`, `Installed`, `Default`. | 21 |
| **3. Exported, but used only inside the deck.** | `mastercontrol`: `HandAir`, `MoveCard`, `DropCard`, `RequestCard`, `Refence`, `StopMonitor`. Their root callers, `app/pool.go:92`, `:99`, `:208` and `:490`, move into the deck. `severeDeck`: `AlertKeysWithin`, `LaneRows`, `Row`, `SetFeed`, `SetLocations`. `eventReader`: `Cancel`, `End`, `Read`, `Toggle`. `tickerDeck.Inject`, `arbiter.Run`, `bedReach.Line`, `mainTrackStage.String`. `radioDeck`'s `InstallsRemaining`, `Resolutions`, `Problems` (`app/cast.go:438`, `:455`, `:465`). | 22 |
| **Total** | | **62** |

**Finding.** `InstallsRemaining`, `Resolutions` and `Problems` have test users only, today
(`app/cast_test.go:363-439`). They are not counted by M5's P2-5, because their users are inside their own
package. The held-surface test records them as they are; 0.19.6 changes nothing about them.

**The held-surface test.** `TestTheDecksExportedSurfaceIsHeld` lives in the deck package. It walks the
deck's non-test files, lists every exported identifier and every exported method by receiver type, and
compares the list with the 62 rows above. It also fails when:

- `radioDeck`'s exported methods are more than `tty.Radio` ∪ the six method values ∪ `cast.Host` ∪ the
  three listed;
- `mastercontrol`'s are more than `tty.Station` ∪ the six listed.

Batch 0 writes it against `app` for tiers 2 and 3. The deck's prep commit adds tier 1.

### AP-1.6 What the 12 symbols other root files use become

| Root file | Today (AS-4) | After |
|---|---|---|
| `pool.go` (7) | `masterControl`, `.mc`, `refreshBedRelays`, `Refence` | `Deck.StationMoved` |
| | `MoveCard`, `DropCard`, `RequestCard` | nothing: `moveCard`, `dropCard`, `requestCard`, `rememberRequested`, `resolvable` and `rebed` move into the deck, with `requestedRefs` and `bedRefresh` |
| `stats.go` (4) | `.mu`, `.source`, `radioDeck` | `Deck.SynthSource`. `synthSource` (`app/stats.go:250`) moves into the deck; `diagSources.deck` becomes a `*deck.Deck`. |
| | `voiceDir` | `deck.VoiceDir` |
| `pipelines.go` (1) | `dropSuperseded` | nothing: it moves to `app/pipelines.go`, its one user (`:189`, `:329`; `app/severe.go:530`) |

**12 symbols become 3.** The root's whole deck surface is the 10 `Deck` methods, `New`, `Config` and
`Deck`, plus `CastBlock` and `VoiceDir`.

### AP-1.7 `app/dashboard.go`'s functions, AS-4's list checked

| AS-4's list | Check |
|---|---|
| `attachRadio` | **moves, without its two `tea.NewProgram` calls**, as `Dress` |
| `saveCast`, `reloadCast`, `saveTones`, `keptRadio`, `radioPrefsFrom`, `relayPrefsFrom`, `buildArbiter` | move |
| `attachDeck` | **splits** (AP-1.3) |
| `wireDeckWarnings` | moves, and takes `warn` as an argument |
| `stationConfig` | **splits** (AP-1.4) |
| the relay hooks (`relayDwell`, `tuneRelay`, `readReport`, `relayLang`, `setRelayLang`, `setRelayDwell`) | move |

**Also moving, and not in AS-4's list:**

- `saveRadioMode`, `saveRadioPrefs`, `defaultVolume`;
- `tickerState` and `tickerPrefs`, without `updateCheck`: the release watch reads `cfg.UpdateCheck` at
  the root, the same value `:226` copies;
- `uiHook`, over `PersistUI`;
- `scriptsDir`, `endEventRead`, `narrateEvent`.

**To the kit:** `savePreference`, `cacheDir`, `userCacheSubdir`.

About 221 declaration lines move from `app/dashboard.go` to the deck (by `dg/decls.tsv`), plus their
doc comments and the carved halves of `startPipelines`, `stopAll` and `attachDeck`.

**Cost and confidence.**

- Principle 2: the root reaches the deck through 15 names instead of 109 symbols.
- Principle 4: no order changes; one extra uncontended lock at shutdown (row 26).
- Principle 1: no new interface, no message, no allocation on the beat (AP-12).
- Confidence: high for the shape; medium for the exact line spans of the carved halves.

---

## AP-2 — `livePipelines`, split (55 fields + the embedded `providerFamilies`, 108 methods)

The struct is `app/dashboard.go:801-922`. AS wrote 801-925; it closes at `:922`. `out/fields.tsv` has
56 entries: `mu`, 54 other named fields and the embedded `providerFamilies`.

### AP-2.1 Every field

| Field (`app/dashboard.go`) | Goes to | Note |
|---|---|---|
| `mu` :802 | root | owner. The deck's and the maps' `mu` hold `&lp.mu` (D-16). |
| `ctx` :803 | root | the maps get their own copy, for `propagationUpdate` (`app/mapprop.go:112`) |
| `started` :807, `release` :808, `unknownKeys` :813 | root | |
| `zoneShapes` :815, `radar` :816, `temp` :817, `tempAnswers` :819, `airGrids` :821, `history` :822, `usage` :823, `lastMapRegion` :826, `maps` :827 | maps | |
| `program` :828 | root | The deck holds its own copy, set by `Attach`, read under `mu` by the bed (`app/bedrelay.go:118`, `:171`). `radioDeck.program` is already a second holder (`:636`). |
| `provider` :829, `providerFamilies` :830, `firms` :831 | root | the maps get `fire` and `marine` as slices |
| `rules` :832 | root | the maps get a copy (`fireRules`, `app/mapfire.go:48`) |
| `priority` :833, `recent` :834 | root | |
| `ticker` :835, `severe` :836 | deck | |
| `mapQuakes` :837, `airnow` :838, `mapClients` :839, `tiles` :840, `problems` :841, `propagation` :842 | maps | |
| `timings` :843 | maps | with `timing.go` (AP-5) |
| `arbiter` :844, `scripts` :845, `reader` :846, `schedule` :847 | deck | |
| `watchRefs` :849, `idx` :855, `station` :856, `poolRefs` :857 | root | `idx`: the deck gets it in `Config.Index`, the maps by `WireUVCities` |
| `requestedRefs` :862 | deck | |
| `relayTable` :867, `bedRadiusMi` :868, `bedPick` :869, `bedOn` :870, `bedStations` :880, `bedRefresh` :884, `bedRelay` :895, `bedCall` :900, `owner` :904 | deck | `relayTable` is written once (`:197`) and read only by tests |
| `clients` :909, `weather` :910 | root | `weather`: the maps hold the same pointer |
| `areaAlerts` :913, `areaMemo` :914, `areasOf` :918 | maps | |
| `tides` :919 | root | the maps hold the pointer (`forgetTidesAndUVCities`) |
| `deck` :920 | deck | |
| `dump` :921 | root | |

**Totals:** root 20, deck 17, maps 19, the 56 entries. The root then holds two new pointers:
`deck *deck.Deck` and `maps *mapdata.Feeds`.

### AP-2.2 Every method (108), by declaring file

| File | Destination (method, line) |
|---|---|
| `app/air_boundary.go` | **deck** (1): `monitorMayReachTheAir` :53 |
| `app/airscope.go` | **deck** (3): `takeTheAir` :117, `masterControl` :151, `silenceMonitor` :166 |
| `app/bedrelay.go` | **deck** (13): `bedRelays` :36, `refreshBedRelays` :68, `setBedStations` :106, `bedFenceMi` :131, `stepBedRelay` :154, `keepBedRelay` :188, `selectedStation` :196, `toggleBedRelay` :209, `playBed` :230, `stopBed` :239, `selectedRelay` :255, `bedCarrying` :265, `noteBedCarrying` :284 |
| `app/dashboard.go` | **root** (18): `giveItAStation` :189 (split), `startPipelines` :265 (split), `ttyConfig` :343 (split), `stationConfig` :395 (split), `attachMaps` :477 (split), `attachDeck` :499 (split), `attachDiagnostics` :515, `currentWatch` :524, `setWatch` :531, `fireFor` :927, `seismicFor` :939, `markFIRMS` :949, `assemblers` :961, `setup` :975, `providers` :990, `stopAll` :1024 (split), `commit` :1049, `hydrate` :1122. **deck** (10): `buildArbiter` :551, `wireDeckWarnings` :564, `relayDwell` :1133, `tuneRelay` :1143, `readReport` :1155, `relayLang` :1165, `setRelayLang` :1176, `setRelayDwell` :1195, `endEventRead` :1211, `narrateEvent` :1219. **maps** (2): `mapConfig` :424 (becomes `FillConfig`), `wireUVCities` :485 (becomes `WireUVCities`). |
| `app/history.go` | **maps** (10): `startHistory` :173 (exported), `historyRegions` :205, `historyStore` :514 (exported), `applyHistory` :521, `setHistory` :533, `clearHistory` :545, `historyUsage` :569, `forgetUsage` :580, `readHistoryUsage` :588, `historyCost` :617 |
| `app/inject.go` | **deck** (2): `injectHook` :217, `testLocation` :246 |
| `app/mapairuv.go` | **maps** (1): `airnowIn` :238 |
| `app/mapfeed.go` | **maps** (2): `mapFeed` :31, `mapFeedWith` :53 |
| `app/mapfire.go` | **maps** (2): `fireRules` :48, `fireIn` :58 |
| `app/mapgeometry.go` | **maps** (2): `seedZoneShapes` :100, `mapClosed` :121 |
| `app/mapinview.go` | **maps** (1): `viewAlerts` :51 |
| `app/maplayers.go` | **maps** (5): `mapInputs` :73, `mapInputsFetching` :79, `inputsFor` :86, `fetchInputs` :114, `mapCost` :164 |
| `app/mapmarine.go` | **maps** (4): `marineProviders` :52, `buoysIn` :67, `tidesIn` :81, `tidesHeld` :105 |
| `app/mapprop.go` | **maps** (1): `propagationUpdate` :112 |
| `app/mapradar.go` | **maps** (1): `mapRadar` :174 |
| `app/maps.go` | **maps** (5): `newMap` :125, `seedOnFirstMap` :134, `clearMapData` :146, `forgetTidesAndUVCities` :183, `mapDisk` :201 |
| `app/maptemp.go` | **maps** (3): `mapTemperature` :163, `tempKeyFor` :224, `tempCoreFor` :236 |
| `app/marine.go` | **root** (1): `marineFor` :17 |
| `app/pool.go` | **root** (12): `setStation` :75, `setTransmitter` :149, `setServiceRadius` :169, `restationTo` :186 (split: `:207-219` become `StationMoved`), `poolFor` :247, `currentPool` :253, `producer` :267, `currentStation` :270, `reStation` :294, `lookInPool` :346, `locateInRadius` :411, `lookupCtx` :456. **deck** (6): `moveCard` :88, `dropCard` :95, `rebed` :109, `requestCard` :482, `rememberRequested` :520, `resolvable` :546. |
| `app/stationrecord.go` | **maps** (1): `recordFragment` :86 (exported as `RecordFragment`) |
| `app/stats.go` | **root** (2): `ttyStats` :181 (split: `FillStats`), `sources` :210 (split) |

**Totals:** root 33 (and one new, `stationReach`), deck 35, maps 40. `livepipelines-AP.tsv` has each row
with its note.

**Bodies that change in the prep commit**, each reading a seam in place of a root field, with no
behaviour change:

- `testLocation` reads `Watch` (`app/inject.go:246-253`). The slice is replaced whole on every
  write (`app/dashboard.go:533`, `:1095`), so reading element 0 outside the lock is the same value.
- `refreshBedRelays` reads `Station` (`app/bedrelay.go:72`).
- `resolvable` reads `Pool` (`app/pool.go:550`).
- `scopeFor` reads `Station` (`app/airscope.go:64-73`).
- `historyRegions` reads the maps' `Station` (`app/history.go:207`).
- `seedOnFirstMap` reads the maps' `Watch` (`app/maps.go:134`).
- `fireIn` and `marineProviders` read the maps' own slices.

### AP-2.3 The two structs

```go
// package deck

// Deck is the shared deck, whole: the radio deck, the arbiter and
// MasterControl, the bed, the Director's executors and the Producer.
type Deck struct {
	mu      *sync.Mutex  // the root's livePipelines.mu (D-16); guards the fields marked G
	program *tea.Program // set by Attach; read under mu by the bed (G)
	deck    *radioDeck
	arbiter *arbiter
	scripts *script.Library
	reader  *eventReader
	schedule *schedule
	ticker  *tickerDeck // written once by StartDirector; read under mu (G)
	severe  *severeDeck
	prefs     tickerPrefs // the [M] mute, the alert radius, the clock
	setRadius func(int)   // tickerState's persist hook, SetAlertRadius's row
	owner   airOwner

	relayTable    *stream.Table
	bedRadiusMi   float64          // written by New, read under mu (G)
	bedPick       int              // G
	bedOn         bool             // G
	bedStations   []stream.Station // G
	bedRefresh    func(context.Context) // G; nil is the real resolve (the test seam of app/dashboard.go:882-884)
	bedRelay      string           // G
	bedCall       string           // G
	requestedRefs []snapshot.LocationRef // G

	watch   func() []snapshot.LocationRef
	pool    func() []snapshot.LocationRef
	station func() (snapshot.LocationRef, float64)
	fire    func(snapshot.LocationRef) synth.FireReport
	seismic func(snapshot.LocationRef) synth.SeismicReport
	marine  func(snapshot.LocationRef) synth.MarineReport
	persistUI func(tty.UIPrefs) error
	client   *httpx.Client
	provider *nws.Provider
	idx      *geodata.Index
	loaded   config.Config
}
```

```go
// package mapdata (app/internal/mapdata; name: AP-11)

// Config is what the map's side of the run is given by the composition root.
type Config struct {
	Mu      *sync.Mutex                            // the root's livePipelines.mu (D-16)
	Ctx     context.Context                        // RunDashboard's context
	Client  *httpx.Client                          // the data client
	Weather *nws.Provider                          // the weather provider
	Tides   *coops.Provider                        // the tides provider
	Fire    []snapshot.Provider                    // the fire family
	Marine  []snapshot.Provider                    // the marine family
	Rules   fire.Rules                             // the [fire] rings
	Watch   func() []snapshot.LocationRef          // the listener's live watchlist
	Station func() (snapshot.LocationRef, float64) // the station's transmitter and radius
}

// Feeds is the map's side of the run: its feeds and memos, the zone shapes,
// the propagation client, the timing instrument and the history store.
type Feeds struct {
	mu      *sync.Mutex // the root's livePipelines.mu (D-16); guards history (G)
	ctx     context.Context
	weather *nws.Provider
	tides   *coops.Provider
	fire    []snapshot.Provider
	marine  []snapshot.Provider
	rules   fire.Rules
	idx     *geodata.Index
	watch   func() []snapshot.LocationRef
	station func() (snapshot.LocationRef, float64)

	zoneShapes    *zones.Store
	radar         *radarSources
	temp          *tempSources
	tempAnswers   lazyMemo[tempKey, tempCore]
	airGrids      airGrids
	history       *history.Store // G
	usage         usageCache
	lastMapRegion atomic.Value
	maps          *mapBuilder
	mapQuakes     *mapQuakes
	airnow        *airquality.Provider
	mapClients    []*httpx.Client
	tiles         *tileCounter
	problems      mapProblems
	propagation   propagationHolder
	timings       *timingLog
	areaAlerts    func(ctx context.Context, areas []string) ([]snapshot.Alert, error)
	areaMemo      lazyMemo[string, []snapshot.Alert]
	areasOf       lastViewAreas
}
```

**What is left at the root.** `livePipelines` keeps 20 entries:

- `mu`, `ctx`, `started`, `release`, `unknownKeys`, `program`, `provider`;
- the embedded `providerFamilies`;
- `firms`, `rules`, `priority`, `recent`, `watchRefs`, `idx`, `station`, `poolRefs`, `clients`,
  `weather`, `tides`, `dump`.

It adds `deck *deck.Deck` and `maps *mapdata.Feeds`, and has 34 methods.

**Test literals of the new structs set `mu` in their setup.** A zero `Deck` or `Feeds` has a nil `mu`, where
a zero `livePipelines` had a usable mutex. Every test literal therefore sets `mu` (setup only, D-14), or
uses an in-package builder.

### AP-2.4 The 124 test literals (55 files), by family

Computed from `out/literals.tsv`, the fields each sets, and the home of its file after the split.

| Family | Literals / files | How it is served (D-14) |
|---|---|---|
| maps fields only, maps test | 34 / 16 | `&Feeds{mu: …, same fields}`. The type and the field path change; setup only. |
| empty, maps test | 19 / 11 | `&Feeds{mu: …}`; setup only |
| root fields only, root test | 17 / 11 | unchanged |
| deck fields only, deck test | 13 / 7 | `&Deck{mu: …, same fields}`; setup only |
| root and maps fields, maps test (`ctx`, `weather`, `idx`, `rules`, `providerFamilies`) | 10 / 7 | the maps' own copies: `ctx`, `weather`, `idx`, `rules`, `fire`, `marine`; setup only |
| empty, deck test | 9 / 7 | `&Deck{mu: …}`; setup only |
| root fields only (`ctx`, `idx`, `providerFamilies`, `rules`), maps test | 8 / 5 | the maps' own copies; setup only |
| empty, root test | 5 / 4 | unchanged |
| root fields (`idx`, `poolRefs`), deck test | 4 / 2 | the `station` and `pool` seams set to literals; setup only |
| root and deck fields, deck test (`idx`+`relayTable`; `ticker`+`watchRefs`) | 2 / 2 | the `station` and `watch` seams; setup only |
| deck fields, root test (`arbiter`, `reader`) | 2 / 2 | the two tests move with their code (AP-9) |
| maps field (`history`), root test | 1 / 1 | a `diagSources` literal; setup only (AP-9) |
| **Total** | **124 / 55** | 99 become the child's own literal; 22 are unchanged; 2 move with their test; 1 changes to a `diagSources` literal. None needs a test-only export. |

**Cost.** Principle 2: this is the release's largest navigation gain in `app`. Each struct holds one
concern, and the compiler holds the line between them. Principle 4: setup only, except where AP-9 says
otherwise. **Confidence:** high; the destination of every field and method is computed and was checked
against its body.

---

## AP-3 — The lock (D-16): the inventory owed to PLAN

### AP-3.1 The count, corrected

**`livePipelines.mu` is locked 25 times, in 5 files.** They are found by type, on the field object
(`aptool`, `out/locks.tsv`):

| File | Locks |
|---|---|
| `app/bedrelay.go` | 9 |
| `app/pool.go` | 7 |
| `app/dashboard.go` | 5 |
| `app/history.go` | 2 |
| `app/inject.go` | 2 |

**D-16's question said 28 in 6 files.** The difference is `app/mapwaves.go`:

- **Its three `lp.mu.Lock()` take `landPoints`' own mutex.** The struct is `app/mapwaves.go:57-62`, its
  `mu` is at `:60`, and the receiver `lp *landPoints` is at `:98`, `:109` and `:132`. `lp.land` and
  `lp.path` are `landPoints` fields.
- **They are covered, not missed: no shared-lock concern.** They move with `mapwaves.go` into the maps
  package, under their own lock, which no other section takes.

**RR F9's per-file counts are line counts, not sections.**

- **"18 sites in `bedrelay.go`"** counts Lock and Unlock lines; that file has 9 sections.
- **"`history.go` 4"** adds `usageCache.mu` (`app/history.go:570`, `:581`), which is that cache's own
  mutex and moves with it.

### AP-3.2 Every section

The full inventory is `locks-AP.tsv`. "Calls under the lock" counts only calls that run inside the
section: not those in function literals or `go` statements.

| # | Section | Fields touched (package) | Section lands in | Calls under the lock into `app` code | Crossing calls after the split |
|---|---|---|---|---|---|
| 1 | `app/bedrelay.go:40-43` `bedRelays` | `bedStations` (deck) | deck | none | none |
| 2 | `:107-119` `setBedStations` | `bedStations`, `bedPick`, `bedRelay`, `bedCall`, `bedOn` (deck), `program` (the deck's copy) | deck | `relayLine` (deck) | none |
| 3 | `:132-138` `bedFenceMi` | `bedRadiusMi` (deck) | deck | none | none |
| 4 | `:159-172` `stepBedRelay` | `bedPick`, `bedRelay`, `bedCall` (deck), `program` (the deck's copy) | deck | `relayLine` | none |
| 5 | `:189-192` `keepBedRelay` | `bedCall` | deck (in `New`) | none | none |
| 6 | `:197-203` `selectedStation` | `bedRelay`, `bedPick`, `bedStations` | deck | none | none |
| 7 | `:259-262` `selectedRelay` | `bedRelay` | deck | none | none |
| 8 | `:266-269` `bedCarrying` | `bedOn` | deck | none | none |
| 9 | `:285-287` `noteBedCarrying` | `bedOn` | deck | none | none |
| 10 | `app/dashboard.go:271-287` `startPipelines` | `priority`, `recent`, `poolRefs` (root). `severe` appears only in the two callbacks, which run on the publisher's timer after the lock is released (`app/pipelines.go:95`). | root | `providers`, `startPriority`, `startRecent`, `withPool`, `refsFromConfig`, `restoreRecent`, `seedRecent` (all root); `lp.recordFragment` is passed, not called | none. The callbacks become root closures over `Deck.SetLocations`, and the fragment hook `Feeds.RecordFragment`; both are passed, not called under the lock. |
| 11 | `:525-528` `currentWatch` | `watchRefs` | root | none | none |
| 12 | `:532-534` `setWatch` | `watchRefs` | root | none | none |
| 13 | `:1025-1027` `stopAll` | `priority`, `recent` (root), `ticker` (deck) | root, without `ticker` | none | none: `ticker` moves to row 26 |
| 14 | `:1074-1119` `commit` | `watchRefs`, `station`, `poolRefs`, `priority`, `recent`, `program` (root) | root | `pipeline.update`, `recentPipeline.update`, `withPool` (root); `config.Mutate` (the config's lock, "LOCK ORDER: lp.mu -> config's", `:1079-1082`) | none |
| 15 | `app/history.go:186-188` `startHistory` | `history` (maps) | maps | none | none |
| 16 | `:515-518` `historyStore` | `history` | maps | none | none |
| 17 | `app/inject.go:219-221` `injectHook` | `ticker` (deck) | deck | none | none |
| 18 | `:247-253` `testLocation` | `watchRefs` (root) | **removed.** It reads `Config.Watch`, which takes row 11. | none | none |
| 19 | `app/pool.go:77-79` `setStation` | `station`, `poolRefs` | root | none | none |
| 20 | `:110-112` `rebed` | `bedRefresh` (deck) | deck | none | none |
| 21 | `:188-190` `restationTo` | `program`, `poolRefs`, `ctx` | root | none | none |
| 22 | `:254-257` `currentPool` | `poolRefs` | root | none | none |
| 23 | `:271-274` `currentStation` | `station` | root | none | none |
| 24 | `:524-536` `rememberRequested` | `requestedRefs` (deck) | deck | none | none |
| 25 | `:551-556` `resolvable` | `requestedRefs` | deck. `Pool` is called before the lock, as `currentPool` is today (`:550`). | none | none |
| 26 (new) | `Deck.StopProducer` | `ticker` | deck | none | none |

**Sections after the split:** root 9, deck 14, maps 2, which is 25.

**The transitive closure under the lock** (`closure.py`). Rows 10 and 14 reach 11 and 5 functions, all in
root files: `app/pipelines.go`, `app/pool.go`, `app/refs.go`, `app/setup.go` and `app/dashboard.go`.
No section reaches a function that takes the shared mutex.

**Under the lock, four calls go through a func value**, all in row 14's closure:

- `rp.publish`, which is `publisher.Trigger` (`app/pipelines.go:337`);
- `pb.after`, nil in production, so `time.AfterFunc` (`:92-95`);
- `rp.newFor`, a scheduler constructor (`:338`);
- `now`.

The production targets are asynchronous or lock-free. Their names are the static check's fixed
allowance (AP-3.5).

**No `…Locked` form is needed.** No critical section calls into another package after the split. The rule
stays, and the check holds it.

### AP-3.3 The lock order

| Order | Where |
|---|---|
| `lp.mu` → the config's lock | `commit` |
| `lp.mu` → `publisher.mu`, the assembler's locks, the scheduler's | inside `commit`'s and `startPipelines`' root calls |
| `mastercontrol.mu`, `radioDeck.mu`, `severeDeck.mu` | each held without `lp.mu`. MasterControl reads its seams under its own lock and calls them after releasing it (`app/mastercontrol.go:154-159`, `:265-270`, `:289-295`), so `playBed` and `stopBed` take `lp.mu` with nothing held. |
| no lock → `lp.mu` | the deck's and the maps' seams into the root (`Watch`, `Pool`, `Station`). Each is called with no lock held, as today. |

### AP-3.4 The guarded set, written on the field

**On `livePipelines.mu`:**

- the owner is the root; the deck's `mu` and the maps' `mu` are this mutex, by pointer (D-16);
- the lock order above;
- "never call `tea.Program.Send` under it" (today's rule, `app/dashboard.go:1064-1067`);
- the guarded set, as two lists:
  - **read and written under it:** `watchRefs`, `station`, `poolRefs`; the deck's `bedStations`,
    `bedPick`, `bedRelay`, `bedCall`, `bedOn` and `requestedRefs`; the maps' `history`;
  - **written once before any reader runs, and read under it:** `priority` and `recent` (written under
    it, `:272`, `:286`); `program` (`:500`) and the deck's copy; `ctx`; the deck's `ticker` (`:305`),
    `bedRadiusMi` (`:199`) and `bedRefresh` (tests only).

**On the deck's and the maps' `mu`:** the owner (`app`'s `livePipelines.mu`), the full guarded set, the
lock order, and "a method called across a package under it is a non-locking `…Locked` form".

### AP-3.5 The static check (batch 0)

**`TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage`**, in `app/sharedlock_test.go`.

**What it scans.** Every non-test `.go` file under `app/`, recursively, with the planned file-to-package map
(`design.py`'s, committed as the graph tool's input, A-12). It treats as the shared mutex
`livePipelines.mu`, `Deck.mu` and `Feeds.mu`, found through the receiver's declared type. A section runs
from a `Lock()` on it to the matching `Unlock()`, or to the end of the enclosing function when the unlock
is deferred. Function literals and `go` statements inside a section are excluded, as they run later. The
check follows the static calls of each section through the module's `app` tree, transitively.

**It fails when:**

1. a function reached from a section takes the shared mutex, or reaches one that does;
2. a call from a section crosses a package of the planned map to a callee whose name does not end in
   `Locked`;
3. a function named `…Locked` takes the shared mutex, or reaches one that does;
4. a call through a func value under a section is not in the fixed allowance. The allowance is four
   names, recorded at batch 0 (CQ F-18): `rp.publish`, `pb.after`, `rp.newFor`, `now`;
5. it finds fewer sections than its fixed floor of 25 (INST-2: silence is a verdict).

**Seen failing** on a planted fixture under `app/testdata/sharedlock/`. The fixture has a section that
calls a locking method, a section that calls a non-`Locked` method across the planned map, and a section
that calls an unlisted func value. Each must be reported.

### AP-3.6 The deadlock test (batch 0)

**`TestCommitAndStopAllFinishUnderTheSharedLock`**, in `app/sharedlock_deadlock_test.go`.

**Setup.** A `livePipelines` with a priority and a recent pipeline over offline providers (as
`TestStopAllWaitsForAFetchThatRecordsItself` builds them, `app/stopall_test.go:38-44`). After the deck
and maps batches, a deck from `deck.New` and the maps from `mapdata.New`, each with `Mu: &lp.mu` and the
root's seams. In batch 0 it is the same struct with its deck and maps fields set in place.

**Acts.** Concurrently, a fixed number of iterations (200) on separate goroutines:

- `commit` with alternating watchlists, so the station re-stations;
- the bed's hooks: `StepBedRelay`, whose command is run; the selected-relay and carrying reads; and the
  note `noteBedCarrying` makes;
- the fragment hook, which takes `historyStore`;
- `SetTransmitter`, which runs `restationTo`.

Then `stopAll`.

**Assertion.**

- Every goroutine and `stopAll` returns within 2 s, scaled by `raceSlowdown` under `-race`.
- On a timeout it fails with every goroutine's stack, so the stuck section is named.
- It never falls back to the 30-minute `go test` timeout (`Makefile:35-38`).

**Seen failing** under two vetted mutants:

- `mDL1`, a second lock in `commit`'s section (a call to `currentPool` after `:1095`), which deadlocks
  on itself;
- `mDL2`, after the deck batch, a call to `Deck.StopProducer` inside the same section. It takes the same
  mutex from another package. Both the static check and this test fail.

`TestStopAllWaitsForAFetchThatRecordsItself` stays as the single-path pin it is.

**Cost.** Principle 4: no lock and no order changes. One section is added (row 26) and one removed
(row 18). Principle 3: the rule is held by a gate, not a comment. **Confidence:** high. The sections are
typed and their closures computed. The allowance list is the one judgement in the check.

---

## AP-4 — The maps entry point and the history; whether `app/internal/maps` repays its cost

### AP-4.1 The entry point

```go
// package mapdata

// New builds the map's clients and memos.
func New(c Config) *Feeds

// Attach readies the tiles' counters and the map builder.
func (fd *Feeds) Attach(ctx context.Context, version string)

// WireUVCities keeps the index and gives UV's markers the cities in view.
func (fd *Feeds) WireUVCities(idx *geodata.Index)

// FillConfig fills the map's and the history's rows of the console's configuration.
func (fd *Feeds) FillConfig(c *tty.Config, cfg config.Config)

// StartHistory opens the history store and records while ctx lives.
func (fd *Feeds) StartHistory(ctx context.Context, keep tty.HistoryRetention)

// RecordFragment is the priority pipeline's fragment hook.
func (fd *Feeds) RecordFragment(f snapshot.Fragment, refs []snapshot.LocationRef)

// Settle waits, bounded, for a propagation update in flight.
func (fd *Feeds) Settle()

// Note records a problem the listener cannot act on, for the diagnostics.
func (fd *Feeds) Note(problem string)

// FillStats fills the Status window's map rows.
func (fd *Feeds) FillStats(st *tty.Stats)

// WireBytes is the propagation client's wire bytes, for the counters.
func (fd *Feeds) WireBytes() int64

// HistoryStore is the store, nil while none is open.
func (fd *Feeds) HistoryStore() *history.Store

// TimingRecords is the timing instrument's intervals, for the dump.
func (fd *Feeds) TimingRecords() []TimingRecord
```

`TimingRecord` is today's `timingRecord` (`app/timing.go:24`), exported because `dumpRecord.Timings`
(`app/dump.go:228`) holds it.

| Step | Today | After |
|---|---|---|
| the map's clients and memos | `newLivePipelines`, `app/dashboard.go:63-76` | `mapdata.New`, called after the root's literal (`:64-70`), so the provider families exist. It builds `timings` (`:64`), `zoneShapes` (`:63`), `areaAlerts` (`:68`), `mapQuakes` (`:69`), `airnow` (`:70`), and the radar and temperature with their clients (`:71-76`). This puts `newTimingLog` and `zones.New` after `newProviderFamilies`. They are constructors; that they do no I/O is UNVERIFIED for `zones.New` and must be read before the prep commit. |
| the sweep's words | `:112-113` | `Note` |
| `attachMaps` | `:115`, `:477-481` | The root keeps `attachDiagnostics` (`:478`), then calls `fd.Attach(ctx, version)` (`:479-480`). |
| `wireUVCities` | `:119`, `:485-490` | `WireUVCities(idx)`, which also keeps the index for `mapConfig`'s namers (`:461-462`) and `viewAlerts` |
| `mapConfig` and the history's 4 rows | `:389`, `:424-466`; `:354-357` | `FillConfig` |
| `startHistory` | `:300` | `StartHistory` |
| `recordFragment` | `:278`, `app/stationrecord.go:86` | `RecordFragment`, a method value |
| `propagation.settle` | `:1042` | `Settle` |
| the map rows of `ttyStats` | `app/stats.go:184`, `:193-196`, `:199` | `FillStats`, called at `:184`'s place. This moves two independent counter reads earlier within one `[S]` refresh; it is a diagnostics read and changes no value. `requestStats` goes to the kit. |
| the dump's wire bytes and timings | `app/dump.go:228-229`; `diagSources.propagation`, `.timings` | `diagSources.maps *mapdata.Feeds`; `WireBytes`, `TimingRecords` |
| the gauges' history | `app/stats.go:211` | `HistoryStore`, nil-safe on a nil `*Feeds` |

**AS-4's 47 + 5 + 1 crossings become 15 names:** `Feeds`, `Config`, `New`, `TimingRecord` and the 11
methods. Three exported methods move as they are, on unexported types, each implementing a standard
interface: `answeredFor.Fetch`, `byteCounter.Write`, `countingTransport.RoundTrip`.

### AP-4.2 Whether it repays its cost (BQ 4.3)

| | Keep (`mapdata`) | Fold back (the map and history stay in the root) |
|---|---|---|
| Lines | about 6,890: Map I's 6,616, plus `timing.go` 82, `lazymemo.go` 57, and the moved `mapConfig`, `savePropagationAck`, `wireUVCities` and `mapBadges` (`app/endpoints.go:68`) | root about 10,300 |
| Users | the root only, through 15 names | — |
| Mutants | 13 (Map I's 12, and m183) | in the root |
| Tests | 52 test files move with it; 16 maps-home files straddle (AP-9), one of them the only unservable test | none straddle; no X |
| `livePipelines` | loses 19 fields and 40 methods; the root keeps 20 and 34 | keeps 39 and 73 |
| Compile-time rule | the root cannot reach the temperature memo, the zone store, the propagation holder or the history store except through its 15 names | none |
| `modes/tty` use | 612 uses of 168 symbols, the map pane's types: the import MS's map package would take over (AS-9) | stays in the root |
| Gate | a maps mutant re-runs about 52 test files instead of about 90 | — |

**Recommendation: keep it.** It is where half of the hub's fields and 37 % of its methods go. That
is principle 2's measured gain, and the one that makes `livePipelines` the root's own struct.

**Strongest counter-argument.** It is the cut with the worst cost-to-gate ratio: 13 mutants. It creates
the one unservable test, and one root call line (`FillStats`) whose mutant moves into it (R-AP-1). The
gain is principle 2, not the gate. That alone is the reason, and the HUM LEAD should weigh it as such.

**Confidence:** high for the figures; the lines are approximate (doc comments counted by a factor).

---

## AP-5 — `app/internal/kit`

The users come from the declaration graph (`dg/uses.tsv`) under the design. A-14 dissolves a helper where
the graph shows one user (CQ F-6).

| Helper (today) | Users after the split | Where it lives | Why |
|---|---|---|---|
| `everyTick` (`app/every.go:27`) | deck (`schedule.tick`), maps (`startHistory`, `app/history.go:199`) | kit | shared. It stays one function for its P10-02 row (`06_docs/p10-ledger.md:63`), re-keyed to the new path and to `EveryTick`. |
| `lazyMemo`, `keptCopy` (`app/lazymemo.go`) | maps only: `mapairuv.go`, `mapradar.go`, `maptemp.go`, `mapinview.go`. The root's two `lazyMemo` fields move to the maps. | maps | one user (CQ F-6) |
| `timingLog` family (`app/timing.go`) | maps (`stage`, `hook`); the root's dump only through `TimingRecords` | maps | one owner; the root reads its records |
| `endOnHangUp` (`app/hangup.go`) | `sayChildren` only | root, with `saychildren.go` (AP-1, C-AP-5) | one user |
| `lockedCopy` (`app/severe.go:442`) | deck (`severeDeck.LaneRows`, `bandRecord.recent`), maps (`mapProblems.last`, `timingLog.last`) | kit | shared; the only `maps → deck` crossing Map I left (`x1.py`) |
| `savePreference` (`app/dashboard.go:744`) | deck (7 callers), maps (`savePropagationAck`) | kit | shared. CQ F-6's "deck only" missed `savePropagationAck`. |
| `userCacheSubdir` (`:760`) | root (`:102`, `:516`, `stats.go:212`), deck (`radio.go:1241`, `voiceDir`, `ticker.go:178`), maps (`maps.go:81`, `mapHTTPDir`) | kit | shared by all three |
| `cacheDir` (`:755`) | root (`dataClientConfig`, `coopsClientConfig`, `sources`), deck (`directoryClientConfig`) | kit | shared |
| `UserAgent` (`app/app.go:31`) | root (`app.go:130`, `dashboard.go:593`), deck (`radio.go:153`, `voices.go:121`), maps (`mapprop.go:37`, the radar and temperature clients) | kit | shared. It leaves `app`'s exported API, where nothing outside `app` uses it (checked across the tree). |
| `requestStats` (`app/stats.go`) | root (`ttyStats`, `ReportOnceWithStats`), maps (`FillStats`) | kit | shared |
| `directoryClientConfig` (`app/app.go:136`) | deck only (`radio.go:144`) | deck | one user |
| `dropSuperseded` (`app/severe.go:530`) | root only (`pipelines.go:189`, `:329`) | root | one user |

**The kit survives, with seven helpers in about 105 lines.** It holds `EveryTick`, `LockedCopy`,
`SavePreference`, `UserCacheSubdir`, `CacheDir`, `UserAgent` and `RequestStats`, exported because they
are shared. The kit batch's prep commit gathers them into one file beside `every.go`, still in package
`app`. Every helper in it has users in at least two packages (graph-held). Map I's other two kit files
leave: `lazymemo.go` to the maps, `hangup.go` to the root. Its name is in AP-11.

**Alternatives considered.**

- **A platform package.** Out of scope: "Splitting any other package" (`project-brief.md:109`).
- **A copy per package.** Forbidden (K-8, RX-5).

**Cost.** Principle 2: the kit is the only package named for a property (sharing) and not a role; A-14's
name has to carry it. Principle 3: one P10 row is re-keyed by both path and name, `everyTick` to
`EveryTick`. That goes into D-10's re-key table (TG's lane). **Confidence:** high.

---

## AP-6 — The GOOS seam (PH H-6), owed to the HUM LEAD

**Facts.**

- The seam is `goosSeam`, `runtimeGOOS` and `setRuntimeGOOS` (`app/cast.go:44-52`; "Tests only",
  `:51`).
- **Every production reader goes to the deck:** `app/cast.go` (`:213`, `:262`, `:274`, `:360`, `:587`,
  `:595`), `app/voices.go` (`:36`, `:59`, `:80`, `:83`, `:179`, `:250`, `:327`) and `app/hostfacts.go`
  (`:31`, `:32`). No root or maps production code reads it.

**Every test that uses it, and where it lands:**

| Use | `file:line` | Lands in |
|---|---|---|
| `TestMain` applies `WATCHPOST_TEST_GOOS` | `app/platform_sim_test.go:26-29` | root, and must exist in the deck too (below) |
| `asPlatform`, the per-test steer | `app/cast_test.go:26-30` | deck |
| `asPlatform` callers | `app/cast_test.go` (19 calls), `app/cast_reconcile_test.go:80`, `app/deck_voice_test.go:62`, `app/director_pins_test.go:78` | deck |
| readers of `runtimeGOOS()` | `app/cast_test.go:28`, `:753`; `app/radio_test.go:65` | deck |
| the guard against direct `runtime.GOOS` reads | `TestNoProductionFileInAppReadsRuntimeGOOSDirectly`, `app/cast_test.go:722-755` | deck |

**Which tests the seam reaches.** By the static call closure, 57 deck-home, 6 root-home and 10 maps-home
test functions reach `runtimeGOOS` today. Under the design, all 6 root-home functions move to the deck
or are split (AP-9). The 10 maps ones reach it only through `lp.ttyConfig`, which becomes
`Feeds.FillConfig` and reaches no deck code.

**After the split, only the root's `TestMain` is outside the deck, and it no longer needs the seam.**

| Option | What it is | Verdict |
|---|---|---|
| **A** | **The seam and its tests stay in the deck.** The deck gets its own `TestMain` (`platform_sim_test.go` in the deck), which applies `WATCHPOST_TEST_GOOS` in-package and calls batch 0's invariants helper for the temporary config and state directories. The root's and the maps' `TestMain` keep the directories only. `make test-platforms` names `./app/...` (`Makefile:213-215`). Batch 0's `TestMain` check gains a rule: a package whose production code calls `runtimeGOOS` applies `WATCHPOST_TEST_GOOS` in its `TestMain`. The `runtime.GOOS` guard widens to `app/...` (AS-8). | **recommended** |
| B | GOOS as a field of `deck.Config` | No: production would pass `runtime.GOOS`, and only tests would set anything else (a test-only option in production API, M5 P2-5). `CastBlock` has no `Deck` to read it from. |
| C | an exported setter in the deck | No: forbidden (D-14) |
| D | the deck reads the environment variable behind a build tag only tests compile (PH's first option) | No: test logic in a production file, and every run without the tag drops the simulation silently |
| E | nothing | No: the deck's suite would run on the host's OS under `make test-platforms` and pass (H-6's quiet case) |

**Recommendation: A.**

**Strongest counter-argument.** Two packages (later three, with the maps and the console) each carry a
`TestMain`, and the deck's carries one line the others do not. Only batch 0's check stops them
drifting. The answer is that the line belongs where the seam is, and the check is already owed (batch 0
item 4).

---

## AP-7 — F-158's owed test (RR F13) and the `app` trace (RR F8)

### AP-7.1 F-158's test

| | |
|---|---|
| **Name** | `TestACutBackUnderAHazardReadKeepsTheBedDown` |
| **File** | written in RM slice 1 as `app/cutback_hazard_test.go`. Its reach must be deck-only under the planned map, checked by the committed graph tool. It therefore moves intact to `app/internal/deck/cutback_hazard_test.go`. |
| **Path** | the shipping one: `WATCHPOST_MAINTRACK` unset, so `mainTrack()` is off (`app/maintrack.go:49-53`), set explicitly by the test and not parallel. This is F-158's direct path: the arbiter reads the hazard, the Director governs the bed. |
| **Entry** | the act enters where the operator's `[b]` does: `tty.Station.CutBed(false)`, MasterControl's `CutBed` (`app/mastercontrol.go:253`, tier 2 of AP-1.5). It carries `CutOver{ToBed:false}` to the Director. |

**Inputs:**

- the station harness (`newStation`, `app/station_test.go`);
- an arbiter over a script voice whose hazard read blocks until the test releases it;
- a fixed clock;
- the sequence: power `Running`, air `AirProgramme`, `CutBed(true)` (the bed carries), a Tornado
  Warning admitted to the rail (the Director emits `Duck`, the executors `hold`), the read started and
  blocked, then `CutBed(false)`.

**Assertions:**

1. **Precondition, not the pinned behaviour.** The Director's step after the cut-back includes
   `Restore` (`platform/lineup/bed.go:330-336`, `:392-402`), recorded by `lineup.Describe`. If it does
   not, the test fails with "no longer exercises F-158", so 0.19.7 must re-author it rather than find it
   passing.
2. With `Restore` executed (`app/executors.go:300-302` → `arbiter.releaseBed`, `app/arbiter.go:675-683`;
   `settle` returns early while a job is on the air, `:415-418`), `mastercontrol.givenWay()`
   (`app/mastercontrol.go:409`) is true while the hazard's job is on the air.
3. After the read is released and the arbiter settles, `givenWay()` is false: the bed returns by the
   ordinary path.

It pins today's safe outcome and never the hazard.

**FULL TDD.** Seen failing under a new vetted mutant, `mF158`, anchored at `app/executors.go:301`. The
mutant replaces `releaseBed` in the `Restore` case with MasterControl's unhold and take-back (RM-4's
mutant). It re-points with the deck.

### AP-7.2 The `app` trace (batch 0 item 6)

| | |
|---|---|
| **Name** | `TestTheDeckIsTheSameUnderRandomActs` |
| **File** | batch 0's `app/deck_trace_test.go`. Its reach is deck-only under the planned map (checked), so it moves intact in the deck's move commit. |
| **Golden** | `app/testdata/deck_trace.golden`, renamed to `app/internal/deck/testdata/` at the move: a pure rename, which A-12's golden gate allows |
| **Recorded** | on code identical to `e57afeaf` |

**The driver is the entry point's act surface,** so the inputs are the same before and after the move:

- the hook rows `FillConfig` fills (AP-1.4);
- `tty.Radio` and `tty.Station` as the deck hands them out;
- `StationMoved`;
- the severe index's `SetLocations`.

In batch 0 the hooks are assembled from the same `livePipelines` methods; in the deck's prep commit they
come from `FillConfig`. That is a setup-only change (D-14), and the golden must not change.

**The build** is the in-package harness: the station, the executors and the arbiter over a script voice
with zero durations, an offline client and a seen store in a temporary directory. It is not
`StartDirector`'s goroutines. The Director is stepped synchronously after every act, so the record is
deterministic.

**Inputs:**

- **Seeds:** four, fixed (1, 7, 42, 1066); 150 acts each.
- **Clock:** starts at 2026-11-14T12:00:00Z and advances one second per act.
- **Locations:** five fixed refs (Bonsall, Oceanside, Vista, Rainbow, Fallbrook).
- **The act set is closed, each act with a stable key:**
  - `on-air`, `standby`, `cut-bed`, `cut-back`;
  - `surface:observer`, `surface:broadcaster`;
  - `tune:<i>`, `stop`, `repeat:<mode>`, `mode:<synth|relay>`, `volume:<n>`;
  - `alert:<scenario key>` (the `InjectAlert` hook's keys, `debugScenarios`), `lapse`;
  - `move:<i>:<to>`, `drop:<i>`, `request:<i>`;
  - `bed-step:<±1>`, `bed-toggle` (the returned command run inline);
  - `narrate:<key>`, `end-read`;
  - `station-moved`, `tick`.

**The record, one line per act, every value by stable key:**

- the seed and step;
- the act's key;
- the events emitted to the Director, by type and fields;
- the Director's effects, by `lineup.Describe`;
- the effects executed, with their outcome through the executors' report seam;
- the messages published, through a closed table of per-type formatters (times as offsets from the
  clock's start, no pointers). An unformatted message type fails the trace, which also holds R8;
- the voice calls recorded since the last act;
- MasterControl's state: power, air, held, given way.

**Assertion.** The record equals the golden byte for byte. No `-update` flag runs in a move commit
(RX-7). The `station-moved` act puts `StationMoved`'s effects (re-fence, re-resolve) in the trace,
which is the deck-side half of R-AP-1.

**KB's lane:** the key format and the per-type formatter table.

---

## AP-8 — Crossings projected under the design

Distinct symbols used across a boundary by non-test code, after the design (`design.py`; the residual
unexported crossing set is empty once the listed entry points replace the split functions' uses):

| Edge | Exported | Unexported | Symbols |
|---|---|---|---|
| root → deck | 15 (+ 12 `Config` fields) | 0 | `Deck`, `Config`, `New`, `CastBlock`, `VoiceDir`, the 10 methods |
| root → maps | 15 (+ 10 `Config` fields) | 0 | `Feeds`, `Config`, `New`, `TimingRecord`, the 11 methods |
| root → kit | 4 | 0 | `UserAgent`, `CacheDir`, `UserCacheSubdir`, `RequestStats` |
| deck → kit | 6 | 0 | `EveryTick`, `LockedCopy`, `SavePreference`, `UserCacheSubdir`, `CacheDir`, `UserAgent` |
| maps → kit | 6 | 0 | `EveryTick`, `LockedCopy`, `SavePreference`, `UserCacheSubdir`, `UserAgent`, `RequestStats` |
| deck ↔ maps | 0 | 0 | none. Map I's one `maps → deck` edge (`lockedCopy`, `app/mapproblems.go:38`) goes to the kit. Map I's `kit → deck` edge (`timing.go` → `lockedCopy`) goes with `timing.go` to the maps. |
| kit → anything in `app` | 0 | 0 | none |

**No cycle** (DQ-6). `lint-imports` holds deck ↛ maps and maps ↛ deck (A-12).

Map I's file-for-file crossings were 109 (root → deck), 52 (root → maps), 26 (deck → root) and 33
(maps → root) (`x1.py`). The design removes the last two by the seams (`Watch`, `Pool`, `Station`) and
by the maps' own copies.

**`app`'s use of `modes/tty`** (1,160 uses of 353 symbols in non-test code, `out/ttyuses.tsv`; AS-9's
total):

| Package | Uses | Distinct symbols | What |
|---|---|---|---|
| root | 300 | 97 | `tty.Config` rows, `NewDashboard`, `NewRouter`, messages |
| deck | 248 | 125 | radio and ticker messages, the `Radio`/`Station` surface, `Surface`, the cast views |
| maps | 612 | 168 | the map pane's types (`MapAsk`, `MapView`, `Region`, `Overlays`, …) |
| kit | 0 | 0 | `timing.go`'s `tty.Timing` leaves with it |

---

## AP-9 — Tests under D-14

`tests-AP.tsv` has one row per straddling test function. Each row carries:

- its home after the split;
- its class, and where it is served;
- the mutants it last-kills in A-3's record;
- how it is served;
- its foreign reach.

`tests-reach-AP.tsv` is the per-file view. Homes are computed from the file's name stem, or by majority
of reach. Three helper files are homed by subject: `mapcost_standin_test.go` and
`mapconcurrent_test.go` in the maps, `test_event_helpers_test.go` in the deck.

**Homes: 150 test files.**

| Home | Files |
|---|---|
| deck | 58 |
| maps | 52 |
| root | 38 |
| kit | 2 |

**Straddlers: 39 files, 64 functions** (AS's Map I figure was 42 files).

| Class | Functions | Files |
|---|---|---|
| setup only | 30 | 20 |
| same value through production API | 14 | 11 |
| moves with the code (the function moves, or the file splits by function) | 19 | 13 |
| **cannot be served without a test-only export** | **1** | **1** |

A file can hold functions of more than one class.

**Setup only (30).**

- A pure helper goes to CK's test-support package: `withConfigFile`, `waitUntil`, `raceSlowdown`,
  `oceanside`, `nwsFixtures`, `offlineClient`, `bonsallCfg`, `indexForTest`.
- A child literal takes the root's seams in place of root fields.
- A `diagSources` literal replaces a root-built one (`TestTheHistoryStoresCachesAreGauged`; its mutant,
  m520, stays at the root with the same killer).

**Same value through production API (14).** The production API each reads through:

| API | Tests | What changes |
|---|---|---|
| `Feeds.FillConfig` | 9 | the map, history, sources and timing hooks are read from the production filler, not `lp.ttyConfig`. `TestTheTimingsReachTheCounters` also uses the `Timed` hook; that it needs no internal timing log is UNVERIFIED. |
| each half's own lane constant (the map feed, `lookupContext`) | 1 | `TestTheMapAndTheLookupAskOnTheListenersLane` splits; that both halves name one constant is UNVERIFIED |
| `Feeds.FillStats` | 3 | the Status window's map rows; m183 re-points into `FillStats` |
| `deck.VoiceDir` | 1 | — |

**Moves with the code (19).**

- **Whole:** the narrate, preview, label and `dropSuperseded` tests; the `currentWatch` test; the two
  station-move tests (mAK4, mRF3 re-point into `StationMoved`).
- **Split by package:** `TestARunWithNoDeckStarts` (m450), the clients-hardening test, the
  kept-settings test, the map Settings window-and-file tests.

**One naming rule for a split test.** The half in the package where its mutant lands keeps the name. The
other half takes a new one, because the sweep finds killers by top-level test name across the tree
(`mutant-verdicts.sh:149-152`).

**The one that cannot be served: `TestThePropagationClientsWireBytesReachTheCounters`**
(`app/mapprop_test.go`). It last-kills m497 (`app/dump.go:229`) and m498 (`app/stats.go:211`), both
root lines.

- Its setup stores a client in the propagation holder's unexported `client` (white-box). That holder is
  the maps' after the split.
- The only production path that sets it is a live GIRO update (`propagationUpdate`,
  `app/mapprop.go:112`).
- A root test cannot give the maps a client with wire bytes without a test-only export.

**Options for the HUM LEAD (C-AP-2):**

- **(i)** The dump reads the counter through a `diagSources` member typed as an interface over the
  client's own `WireBytes` (`httpx`'s production method). m497 is then served at the root with a
  `diagSources` built from the test's client, through `newDumper`'s sources parameter (setup only). m498
  is served by a root assertion that `sources()` hands the maps. That is a weaker observation (the
  holder's identity, not the bytes flowing), so it is the ruling.
- **(ii)** Fold the maps back (AP-4), which removes the case.
- **(iii)** A test-only export: forbidden by D-14 itself.

**Recommended: (i).** **Strongest counter-argument:** it rewrites one assertion to a weaker one where
two mutants sit, which is exactly the loss D-14's proof is there to prevent.

**R-AP-1, three root call lines whose mutants move into a child.** Four anchored statements move from a
root function into a child entry point:

| Mutant | From | Into |
|---|---|---|
| m183 | `ttyStats` | `FillStats` |
| m450 | `attachDeck` | `Attach` |
| mAK4, mRF3 | `restationTo` | `StationMoved` |

Their killers move with them, so M7's scoped sweep matches A-3's. But the root's new call lines carry no
mutant and no root test.

**Today's wiring.** No mutant anchors in `startPipelines`' wiring either (`mutants-AP.tsv`), so this
matches today's wiring coverage everywhere except these three, which were pinned.

**What production API can pin at the root:**

- **`FillStats`:** can be pinned at the root: `Note` a problem, read `ttyStats().MapProblems` (same value).
- **`Attach` and `StationMoved`:** have no root-observable effect without a running program.

**Recommended (C-AP-3):**

- a root test, and a new vetted mutant, for the `FillStats` call;
- the trace's `station-moved` act for `StationMoved`'s effects in the deck;
- the two remaining root lines (`Attach`, `StationMoved`) recorded as wiring with no pin, as
  `startPipelines`' wiring is today.

**Confidence.** High for the counts and the one unservable test. Medium for the class of the 14 "same
value" tests: each was judged from its foreign reach and the test's own lines, and two are marked
UNVERIFIED in the TSV.

---

## AP-10 — Mutants

**153 anchor in `app`** (A-6). The check: `mutants.py` parses every mutant file with `ast` (the anchor
strings, both quote styles and triple quotes) and finds each in its file. All 153 were located, 1 anchor
each.

**Map I, file level:** deck 112, maps 12, root 28, plus 1 in a test file (`app/air_boundary_test.go`),
which A-6 counted with the root (29). **Confirmed.**

**Under this design** (statement level, the split functions' carved lines followed):

| Package | Mutants |
|---|---|
| deck | 116 |
| maps | 13 |
| root | 23 |
| the test-file anchor, which moves with the deck | 1 |

The moves from Map I's figures:

| Mutant | From | To | Because |
|---|---|---|---|
| m450 | root | deck | the nil return goes into `Attach` |
| mAK4, mRF3 | root | deck | `StationMoved` |
| mJ2, mM5 | root | deck | `endEventRead`, `setRelayDwell` |
| m183 | root | maps | `FillStats` |
| m377 | deck | root | `saychildren.go` stays at the root |

**Re-homing, with receivers renamed** (`dk` on `Deck`, `fd` on `Feeds`):

| What | Count |
|---|---|
| mutants that change path | **130** (116 + 13 + 1) |
| of those, also change anchor text | **11** |
| root mutants that change anchor text only | **2** |
| mutants that change path only | **119** |
| unchanged | **21** |

**The 11 that change anchor text, by file:**

| File | Mutants |
|---|---|
| `app/bedrelay.go` | 3: mAK1 (`toggleBedRelay`), mY2 (`stepBedRelay`), mY4 (`selectedRelay`) |
| `app/dashboard.go` | 3: m450 (the comment names `attachRadio`), mJ2, mM5 |
| `app/pool.go` | 2: mAK4, mRF3 |
| `app/airscope.go` | 1: mZ3 |
| `app/mapprop.go` | 1: m575 |
| `app/stats.go` | 1: m183 |

The 2 root mutants that change text are m497 (`app/dump.go:229`) and m498 (`app/stats.go:211`): the
diagnostics' member names change.

**The anchors of three body-changed functions keep their text** and were checked: m520
(`diagSources.gauges`), m723 (`sources`) and mAS1 (`stationConfig`'s root bounds row).

**No anchor names a renamed kit helper,** except m183 (`requestStats`, in the moved line).

**If the re-homed methods kept the receiver name `lp`,** the text changes would fall from 13 to 3 (m450,
m497, m498), at the price of a receiver that names the wrong struct on 75 methods. That is a choice
(C-AP-8). A-11's scoped sweep re-vets every moved mutant either way; the text changes cost only the
old-and-new anchor table.

**New mutants** this design asks for:

| Mutant | Pins |
|---|---|
| `mF158` | AP-7.1 |
| `mDL1`, `mDL2` | AP-3.6 |
| one on the root's `FillStats` call | R-AP-1 |

---

## AP-11 — Names (A-14)

| Package | Recommended | Alternatives | Why |
|---|---|---|---|
| the deck | `app/internal/deck`, entry type `deck.Deck`, receiver `dk` | `deck.Air` | The HUM LEAD's own word for the shared deck (`package-map.md` §2). A `deck.Deck` is the package's one central type, as `list.List` is in the standard library. The doc comment says what the record's "the deck" means: the whole, with `radioDeck` its player. `deck.Air` names a concept the deck serves, not what it is. |
| the maps | `app/internal/mapdata`, entry type `mapdata.Feeds`, receiver `fd` | `mapfeeds`, `layers` | `maps` shadows the standard library's package, imported at `app/cast.go:10` (forbidden). "Map data" is the listener's own phrase: Settings' "Clear map data" (`app/dashboard.go:429`, FR-3.10). The package also holds the history the Data tab keeps. `mapfeeds` collides with `mapfeed.go`'s narrower meaning; `layers` leaves out the history. |
| the kit | `app/internal/run` | `launch` | What one run of watchpost shares: how it names itself (`UserAgent`), where it caches, how it saves a preference, its one clock-driven loop, and the locked copy. "Run" is the code's word for one launch ("this run's children", `app/saychildren.go:24`; `RunDashboard`). It shadows no standard-library package and is not a keyword. |
| the constructors' options | `deck.Config`, `mapdata.Config` | `Options` | `Options` is taken in package `app` during the prep commit (AP-1.1) |
| the cast block | `deck.CastBlock` | — | `RenderCast` is the root's front door |

Each needs a vocabulary row (M5).

---

## AP-12 — Performance

**Where the split could add an allocation or an escape on the beat, and why it does not.**

| Measure | Today | Why the design leaves it the same |
|---|---|---|
| Radio beat pin (327 allocations / 68,475 B, `modes/tty/beat_alloc_test.go:60`) and publish pin (203 / 82,322, `:106`) | measured in `modes/tty` over `tty`'s own fakes | The `app` split cannot move them. The one `app` value on the 50 ms tick is `radioDeck.Spectrum`, handed to `tty` as a method value (`app/dashboard.go:617`, AS-9). `Dress` hands the same method value, built once at wiring, as today. |
| The publish callbacks (`app/dashboard.go:277`, `:286`) | dereference `lp.severe` when called | They call `Deck.SetLocations`, a one-line forwarder that inlines across the package. No closure and no interface are added; the closures are built once, at start, as today. |
| The tone-latency benchmark (185 allocations, `app/tone_latency_test.go:83`) | the harness, the arbiter, the executors, MasterControl | None of the bodies on its path changes. Its file moves with the deck intact. A-12's pin holds it. |
| `BenchmarkSevereDeckTrigger` (`app/severe_test.go:188`); the names memo's `AllocsPerRun` pin (`app/mapnames_test.go:134`) | — | They move with their code; their paths are unchanged. |
| Seams (`Watch`, `Pool`, `Station`) | `currentWatch` and `currentStation` are already passed as func values to the ticker and to `scopeFor` (`:305`, `:326`). `currentPool`, `currentStation`, `currentWatch` and `historyStore` are not inlinable today (`defer`; `inl-app.txt`). | No inlined call site is lost. None is read on the frame path (R9). `masterControl` and `producer`, inlined today at `app/airscope.go:132`, `app/dashboard.go:313`, `:335` and `app/pool.go:207`, keep same-package or cross-package inlining. |
| New interfaces | — | None on `Update`, `View` or any hot path (PB-2, GP-10). `SynthSource` returns the interface `synthSource` returns today, on the diagnostics path. |
| Model sizes | — | Unchanged: no `modes/tty` type changes |

**One thing the batch's instruments must allow for.** 122 inlined call sites sit today in the functions
this design splits: `startPipelines` 26, `ttyConfig` 15, `dumper.record` 12, `newLivePipelines` 10,
`RunDashboard` 9, `attachRadio` 8, `ttyStats` 8, `mapConfig` 7, `stopAll` 6, `restationTo` 6, and the
rest (`inl-app.txt`). A-12's "inlined calls counted per function, any fewer needs a HUM LEAD ruling"
must therefore match by callee and call-site text across the union of the old function and its carved
entry points, not by enclosing function name. Otherwise every split reads as a loss. **PB's and KB's
lane.**

**Confidence:** high for the beat. The cross-package inlining of `SetLocations` and the other forwarders
is to be confirmed by the batch's `-gcflags=-m` diff.

---

## Choices for the lead

| # | Question | Options | Recommendation | Strongest counter-argument |
|---|---|---|---|---|
| C-AP-1 | **The GOOS seam (AP-6), a ruling for the HUM LEAD** | A to E | **A**: the seam and its tests in the deck, the deck's own `TestMain`, `test-platforms` over `./app/...`, held by batch 0's `TestMain` check | `TestMain`s that differ by one line drift without the check |
| C-AP-2 | **The one test that cannot be served (AP-9), a ruling with its count: 1 test, 2 mutants (m497, m498)** | (i) the interface member, m497 by setup, m498 by a holder-identity assertion; (ii) fold the maps back; (iii) a test-only export (forbidden) | **(i)** | a weaker assertion where two mutants sit |
| C-AP-3 | **R-AP-1: root call lines whose mutants move into a child** (m183, m450, mAK4, mRF3) | (a) accept, the killers kept in the child; (b) a root test and mutant where a production effect is observable (`FillStats`), the trace's `station-moved` act for the deck side, the other two recorded as unpinned wiring; (c) a root-level trace through a running program | **(b)** | two root lines that were pinned are no longer pinned at the root |
| C-AP-4 | The deck's start and stop as two methods each, today's exact order | (a) exact order: `StartVoice`, `StartDirector`, `StopProducer`, `StopDirector`; (b) one `Start` and one `Stop`, moving `startHistory` (`:300`) and `propagation.settle` (`:1042`) out from between them | **(a)**, R1 | two more exported names for one boundary |
| C-AP-5 | `saychildren.go` and `hangup.go` at the root, not the deck and the kit (an A-row) | root (one user, `RunDashboard`; no deck symbol); deck (Map I) | **root** | Map I's file list changes (deck 29 files); m377 changes package |
| C-AP-6 | Does `app/internal/maps` repay its cost (BQ 4.3)? | keep; fold back | **keep** (AP-4) | 13 mutants, the X test and one orphaned call exist only because of it |
| C-AP-7 | The kit's seven helpers, under one name (an A-row) | `app/internal/run`; `launch`; dissolve (out of scope) | **`run`** | it is still a package of shared things, not a role |
| C-AP-8 | Names (A-rows): `deck`/`Deck`, `mapdata`/`Feeds`, `deck.Config`/`mapdata.Config`, `CastBlock`; receivers renamed `dk`/`fd` | rename receivers (13 anchors change text); keep `lp` (3 change) | **rename** | 10 more anchor-text changes, each re-vetted anyway by A-11's sweep |
| C-AP-9 | `stationConfig` splits (5 rows stay at the root), unlike AS-4 (an A-row) | split; move whole and export the root's station API | **split** | none of weight |

## Touches other agents

- **CK, the test-support package and the `TestMain` helper.**
  - CK receives the pure helpers AP-9 moves: `withConfigFile`, `waitUntil`, `raceSlowdown`,
    `oceanside`, `nwsFixtures`, `offlineClient`, `bonsallCfg`, `indexForTest`.
  - The `TestMain` invariants helper must be callable from the deck's `TestMain`, which also applies the
    GOOS seam in-package (AP-6).
- **MS, the map seam.** `app/internal/mapdata` makes 612 of `app`'s `modes/tty` uses, all on the map pane's
  types. If MS groups `tty.Config`'s map hooks into a struct of the map package (GP-10),
  `Feeds.FillConfig` fills that struct instead.
- **KB, the key builder.** AP-7.2's act keys and per-type message formatters; AP-7.1's
  `lineup.Describe` precondition; the new mutants `mF158`, `mDL1`, `mDL2` and the `FillStats` one.
- **TG / D-10's re-key table.** The ledger row at `06_docs/p10-ledger.md:63` is re-keyed by path and by
  symbol (`everyTick` to `EveryTick`). `docs_test.go:353`'s package list gains the four packages.
- **PB, the per-function inlining comparison** across carved functions (AP-12).
- **RX, two batch-0 checks.**
  - The committed graph tool's crossings, computed with the planned file map, are the prep commit's
    acceptance check: zero unexported crossings before the move.
  - The trace's and F-158's test files are checked to be deck-only in reach.
