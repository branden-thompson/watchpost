---
title: "0.19.6 DISCOVER — lens AS: the seams in app"
date: 2026-11-14
phase: DISCOVER
lens: AS
status: "FILED — findings checked against e57afeaf"
---

# Lens AS — along which seams `app` splits, and what each cut costs

## Verdict

**`app` has two clusters that cannot be cut further, and everything else is either the composition
root or a leaf.** Set aside the root (`dashboard.go`, `app.go`), the diagnostics (`stats.go`,
`dump.go`, `debug.go`) and about twelve leaf helpers, and treat `livePipelines`'s methods as a seam.
What remains of the package's file graph is exactly two cycles (AS-1):

- **the deck**: 13 files, among them `radio.go`, `arbiter.go`, `mastercontrol.go`, `ticker.go` and
  `cast.go`;
- **the map and the history**: 15 files.

The shared deck is therefore already one unit in the code, and a map that moves it whole is the only
kind the graph allows.

**Recommended: Map I.** It has four packages:

| Package | Files | Lines |
|---|---|---|
| `app`, the composition root | 20 | 4,054 |
| `app/internal/deck`: the deck, its gates, the Director's executors and the Producer | 30 | 9,297 |
| `app/internal/maps`: the map and the history | 22 | 6,616 |
| `app/internal/kit`: leaf helpers | 4 | 217 |

Imports run one way: `app` imports the other three, and `deck` and `maps` import only `kit`. The
cut is not a pure move. It needs:

- two narrower structs in place of 57 of `livePipelines`'s 108 methods;
- `dashboard.go` split by what it wires;
- one lock decision.

It also makes two single-owner rules compile-time facts that today are scans or conventions.
**The strongest argument against it** is that the deck package is still 9,297 lines. Cutting it
further would export the arbiter's and MasterControl's internals (AS-6).

## What I checked and how

- **Read:**
  - the brief and the rulings (D-1 to D-7, A-1 to A-5);
  - both required readings, and `as-built-air-and-deck.md` §§1-3;
  - `docs/architecture.md` and `docs/where-things-happen.md`;
  - `app/doc.go`;
  - the headers of all 76 non-test files;
  - the structs `livePipelines` (`app/dashboard.go:801-925`), `radioDeck` (`app/radio.go:40`),
    `tickerDeck` (`app/ticker.go:99`) and `schedule` (`app/schedule.go:36`);
  - every scan-based test named below.
- **Computed from the shared graph** (`file-edges.tsv`, `symbols.tsv`, `test-reach.tsv`,
  `cross-package-edges.tsv`). The scratch scripts are in
  `<scratchpad>/q196/as/`:
  - `cut.py` assigns files to packages, then counts the symbols that cross each boundary, how many
    of them are unexported, and the cycles. It also counts methods declared in another package than
    their type, which Go does not allow.
  - `seam.py` does the same after a struct seam: each sub-package takes the `livePipelines` fields
    and methods only it uses.
  - `tests2.py` counts the test files whose unexported reach spans packages.
  - `scc.py` and `scc2.py` find the file-level cycles.
  - A small `go/ast` lister (`decls/`) lists every method's receiver, which the graph omits when a
    method is used only in its own file.
- **The record checked for path-dependence:**
  - the 110 mutant anchors of the form `Path("app/…")` in `06_docs/mutants/*.py`;
  - the P10 ledger rows keyed on `app`;
  - `cmd/watchpost/docs_test.go`.
- **Not run:** no test, gate, benchmark or app. The counts are static and come from the graph, at
  file granularity.

## Findings

### AS-1 — Two irreducible clusters, and everything else is the root or a leaf

**Claim.** The file-level graph is one 47-file strongly connected component, but only because of
three things:

- the composition root;
- the diagnostics, which read everything;
- about twelve leaf helpers: `UserAgent`, `userCacheSubdir`, `savePreference`, `lockedCopy`,
  `everyTick`, `lazyMemo`, `keptCopy`, `radioDebugLog`/`On`, `directoryClientConfig` and
  `cacheDir`.

Remove those, and treat `livePipelines`'s methods as the seam they would become. Exactly two cycles
remain:

- **Deck:** `airscope.go`, `arbiter.go`, `burst_words.go`, `cast.go`, `inject.go`, `mainread.go`,
  `mastercontrol.go`, `pool.go`, `radio.go`, `radio_queue.go`, `shared.go`, `ticker.go`,
  `voices.go`.
- **Map and history:** `eventrecord.go`, `history.go`, `stationrecord.go`, `uvknown.go`, and
  `map{airuv,feed,fire,layers,marine,quakes,radar,rain,s,temp,waves}.go`.

The other 30 files are acyclic satellites. Examples are `executors.go`, `schedule.go`, `pump.go`,
`read_script.go`, `severe_read.go`, `mapgeometry.go`, `mapprop.go` and `framelog.go`.

**Evidence.**
- The two cycles are the output of `scc2.py` over `file-edges.tsv`.
- Within the deck, the pairs call each other both ways:
  - `radio.go` → `arbiter.go`: 15 uses (`clip` and its fields). The reverse is 5 uses
    (`radioDebugLog`, `radioDebugOn`).
  - `arbiter.go` → `mastercontrol.go`: 12 uses. The reverse is 8 (`narrationVoice.*`).
- `ticker.go` builds an arbiter over a MasterControl (`app/ticker.go:159-167`) and holds both
  (`:117-118`).
- `pool.go` joins the cycle only through `airscope.go`'s use of its `stationArea`
  (`app/pool.go:37`).

**Consequence.** Principle 2: the map can follow seams the code already has, instead of inventing
them. Any package boundary drawn through either cycle forces new exported interfaces or a compile
error.

**Recommendation.** Draw the packages around these two cycles. Place the satellites by which cycle
they lean on (AS-3), and lift the leaf helpers into one leaf package.

**Confidence.** High. This is computed, and the edges in the claim were read.

### AS-2 — `livePipelines` is the hub, and its methods are spread over 21 files

**Claim.** The one struct that stops a plain move is `livePipelines`. Its definition:

- 55 fields (`app/dashboard.go:801-925`);
- 108 methods, declared in 21 files: `dashboard.go` 30, `pool.go` 18, `bedrelay.go` 13,
  `history.go` 10, the `map*.go` files 27, and others;
- used by 20 non-test files and 55 test files (`symbols.tsv`);
- built as a literal 124 times in 55 test files.

Go requires a method to be declared in its type's package. So every map or history file that moves
takes its methods with it onto a new struct.

**Evidence.**
- The method counts come from the `go/ast` lister.
- Under Map I, 57 methods would be declared away from their type: 38 in map and history files, 19
  in deck files (`air_boundary.go`, `airscope.go`, `bedrelay.go`, `inject.go`). One `radioDeck`
  method is in `stats.go:250`.
- The fields fall into clean groups by user (`symbols.tsv` `using_files`):
  - **map and history:** 25 fields, among them `zoneShapes`, `temp`, `radar`, `tempAnswers`,
    `airGrids`, `areaAlerts`, `areaMemo`, `areasOf`, `mapQuakes`, `airnow`, `mapClients`, `tiles`,
    `problems`, `propagation`, `lastMapRegion`, `history` and `usage`;
  - **bed and air:** 15 fields, among them `bedCall`, `bedOn`, `bedPick`, `bedRadiusMi`,
    `bedRelay`, `bedStations`, `owner`, `arbiter`, `deck` and `ticker`;
  - **the root's own:** the rest.

**Consequence.**
- Principle 2: one struct holding five concerns is what makes the package hard to navigate. The
  split's biggest gain is replacing it.
- Principle 4: the 124 test literals are the bulk of the test churn (RX).

**Recommendation.** This is a new seam, not a move. Each sub-package gets a narrower concrete struct
for its concern. The root builds it and keeps a pointer. The shared reads (`currentStation`,
`currentWatch`, `ctx`, `idx`) are passed in as values or func fields at construction. Do not use
interfaces here (AS-9).

**Confidence.** High for the counts. Medium for the exact field grouping: the graph counts a field's
use by file, and `dashboard.go` assigns most fields.

### AS-3 — The deck, whole: what it is, file by file

**Claim.** The deck and its gates, moved whole, are the following 30 files, 9,297 lines:

- **The deck:** `radio.go`, `radio_queue.go`, `cast.go`, `voices.go`, `saychildren.go`,
  `livesource.go`, `mainread.go`, `maintrack.go`, `rotation_script.go`, `read_script.go`,
  `shared.go`, `hostfacts.go`.
- **The gates:**
  - `arbiter.go`, the voice's one owner;
  - `mastercontrol.go`, the air's effector and the band's one writer;
  - `air_boundary.go` and `airscope.go`: who may reach the air, and the alert rail's fence;
  - `bedrelay.go` and `bedfence.go`: the station's bed.
- **The Director's executors:** `schedule.go`, `executors.go`, `pump.go`.
- **The Producer and Composer:** `ticker.go`, `seen_store.go`, `alert_store.go`, `burst_words.go`,
  `compose_takeover.go`, `severe.go`, `severe_read.go`, `metro.go`, `inject.go`.

The type-level contents are:

- `radioDeck`: 88 methods, in `radio.go`, `radio_queue.go`, `cast.go`, `voices.go` and
  `mainread.go`, plus `stats.go:250` (`synthSource`), which must move into the deck;
- `arbiter`, `speaker`, `clip`, `narrationVoice` and `narrationClass`;
- `mastercontrol`, with 27 methods;
- `schedule`, `executors`, `pump` and `bedSeams`;
- `tickerDeck` (15 methods), `seenStore`, `alertStore`, `severeDeck` and `eventReader`;
- `airOwner`, `airScope` and `listenerScope`;
- the bed's state and methods.

`hostfacts.go` goes with the deck because it is the cast without a deck. It uses `castLoaded`,
`discoverMacVoices`, `discoverTimeout`, `piperInstallFor`, `runtimeGOOS`, `voiceDir`,
`defaultVoiceFor` and `discoveredIn`.

**Evidence.**
- The files and counts come from `cut.py` (Map I).
- The air's predicates:
  - `radioDeck.monitorHasTheAir` (`app/radio.go:1341`) asks `d.air`;
  - the root sets `d.air` (`app/dashboard.go:510`) from `lp.owner`;
  - `livePipelines.monitorMayReachTheAir` (`app/air_boundary.go`) asks the same `lp.owner`;
  - `takeTheAir` moves it (`app/airscope.go:117`).
- `advances` and `advancesMonitor` themselves live in `platform/lineup`, which is not moved.
  `TestOnlyOneProgrammeCanEverAdvance` (`app/air_test.go:40`) reaches `takeTheAir`, `.arbiter`,
  `.deck`, `.owner`, `.ticker` and MasterControl, so it moves with the deck.

**Consequence.** This holds "the shared deck is not split" (0.16.0 required reading): one deck, the
Director's executors beside it, and MasterControl gating it, all in one package. D-2's dissolution of
the arbiter (RM) then happens inside one package and moves no boundary.

**Recommendation.** Move this set as one unit. Moving `bedrelay.go`, `airscope.go`,
`air_boundary.go` and `inject.go` into the deck needs a deck-side struct for the bed state and the
surface owner, which their 19 `livePipelines` methods become methods of. It also needs a decision
on `lp.mu` (AS-5). `pool.go` stays with the root (AS-5).

**Confidence.** High.

### AS-4 — Most of the cost sits in `dashboard.go`, so it splits by what it wires

**Claim.** Under Map I, 112 symbols would cross from the root into the deck. 99 of them come from
`dashboard.go`'s deck wiring:

- `attachRadio`, `saveCast`, `reloadCast`, `saveTones`, `keptRadio`, `radioPrefsFrom`,
  `relayPrefsFrom`;
- `buildArbiter`, `attachDeck`, `wireDeckWarnings`, `stationConfig`;
- the relay hooks.

Moved file for file, the cut would export about 90 identifiers that are unexported today, 19 of them
`radioDeck` fields assigned at construction (`.abandonRead`, `.air`, `.clockPref`, `.composer` and
others).

The other root files reach far less:

| Root file | Symbols reached in the deck |
|---|---|
| `pool.go` | 7 (`mastercontrol.DropCard`/`MoveCard`/`RequestCard`/`Refence`, `.mc`, and two bed methods) |
| `stats.go` | 4 (`.source`, `.mu`, `radioDeck`, `voiceDir`) |
| `pipelines.go` | 1 (`dropSuperseded`) |

The same holds for the maps: 47 of 53 root crossings come from `dashboard.go` (`mapConfig`,
`attachMaps`, `wireUVCities`). The rest are `stats.go`'s 5 and `dump.go`'s 1.

**Evidence.**
- The counts come from `cut.py` and a per-file pass over the Map I crossings.
- A keyword pass over `dashboard.go`'s functions puts about 500 of its 1,234 lines in functions that
  name only the deck, and about 450 in the mixed composition (`RunDashboard`, `startPipelines`,
  `ttyConfig`). This split is a heuristic, not a proof.

**Consequence.**
- Principle 2: an export of each field would make the deck's exported surface mean "whatever the
  root once touched".
- AS-7 shows why that undoes a rule the code relies on.

**Recommendation.** The deck's wiring half of `dashboard.go` moves into the deck as one entry point:

- a constructor that takes one options struct of the deck's dependencies;
- a method that fills `tty.Config`'s radio and station hooks;
- a stop.

The map half does the same in `maps`. The root keeps `RunDashboard`, the pipelines, the pool,
`commit`, `stopAll`, `ttyConfig`'s non-deck rows and the diagnostics. The root-to-deck surface then
shrinks to the entry point plus the 12 symbols the other root files use. Signatures belong to PLAN.

**Confidence.** High for the counts. Medium for the line split.

### AS-5 — One lock guards three concerns, and only one of them can leave

**Claim.** `livePipelines.mu` is taken 28 times. It guards:

- the watchlist (`watchRefs`);
- the station's pool (`station`, `poolRefs`, `requestedRefs`, `bedRefresh`);
- the bed (`bed*`);
- the history store pointer;
- the pipelines (`priority`, `recent`, `ticker`).

`commit` updates the watchlist, the station, the pool and the RECENT pipeline's set in one critical
section (`app/dashboard.go:1074-1116`; the assignments are at `:1095`, `:1104` and `:1116`). Giving
the pool its own lock would break that atomicity.

The bed's sections touch only bed fields and `program`:

- `bedrelay.go`'s locked blocks;
- `bedRelays`, `bedFenceMi`, `keepBedRelay`, `selectedStation`, `selectedRelay` and
  `bedCarrying`.

The history's sections touch only `lp.history` (`app/history.go:186`, `:515`).

**Evidence.** The `lp.mu` sites were read, and the fields each section touches were extracted. The
lock-order warnings are at `app/dashboard.go:1028-1031` and in `commit`'s "LOCK ORDER: lp.mu ->
config's" comment.

**Consequence.**
- Principle 4: moving the bed or the history out of `livePipelines` adds a lock and a lock order.
  It is behaviour-neutral only if no caller holds `lp.mu` while it takes the new lock, or the
  reverse.
- Moving the pool out would change what `commit` makes atomic.

**Recommendation.**
- The pool, the station and the watchlist stay in the root, under `lp.mu`.
- The deck reads them through funcs it is given. `startTicker` already takes `watch func()`
  (`app/ticker.go:159`).
- `stationArea` (`app/pool.go:37`) moves down to the deck, which breaks the only reason `pool.go`
  is in the deck's cycle (AS-1).
- The bed and the history each take their own mutex. PLAN must list every call path between the
  two locks first, and RX must run each touched package whole under `-race`.

**Confidence.** Medium. The sections were read, but call paths that take `lp.mu` and then call into
the bed were not traced exhaustively.

### AS-6 — The deck cannot be cut further without exporting the duck and the band

**Claim.** Two smaller deck maps were measured. Both cross the arbiter's and MasterControl's inner
API.

- **Map IIb: the Director's executors as a package above the deck.** This means `schedule.go`,
  `executors.go`, `pump.go`, `read_script.go`, `rotation_script.go`, `compose_takeover.go` and
  `severe_read.go`: 7 files, 2,290 lines.
  - It crosses 58 symbols into the deck, 57 of them unexported. Among them are `speaker.awaitAir`,
    `speaker.deliver`, `speaker.hold`, `arbiter.releaseBed`, `mastercontrol.cue`,
    `mastercontrol.hold`, `mastercontrol.clearBand`, `.carry`, `.mu`, and `tickerDeck`'s
    `.alerts`, `.emit`, `.mu`, `.muted` and `.seen`.
  - It leaves 56 test files spanning packages.
- **Map II: the Producer and Composer as a package beside the deck.** This means `ticker.go`, the
  stores, `burst_words.go`, `compose_takeover.go`, `severe.go`, `severe_read.go` and `metro.go`:
  2,257 lines.
  - It forms a cycle: 20 symbols cross from the Producer into the deck (`arbiter.Run`, `speaker.*`,
    `newMastercontrol`) and 15 cross back (`composeTakeover`, `alertStore.get`, `seenStore.mark`,
    `tickerDeck` fields).
  - Breaking the cycle needs interfaces.
  - It leaves 72 test files spanning packages.

**Evidence.** Both maps were run through `cut.py` and `tests2.py`.

**Consequence.**
- Exporting `cue`, `hold`, `giveWay` and `takeBack` lets any importer write the band and the duck.
  Their one owner is the reason those methods are unexported (`app/mastercontrol.go` header: "IT IS
  THE ONE OWNER OF EACH OUTPUT").
- Interfaces between the Producer and the arbiter would be built now and then rewritten by D-2,
  which changes exactly that coupling (RM-1, RM-7).

**Recommendation.** No sub-cut of the deck in 0.19.6. If the deck's size is still a problem after
D-2 is ruled, the Producer is the candidate cut, measured again against the code D-2 leaves.

**Confidence.** High.

### AS-7 — What a boundary turns from a scan into a compile-time fact, and what it does not

**Claim.** Under Map I, two ownership rules become compile-time for every caller outside the deck
package:

- **The deck's lower-case internals can no longer be called from the root or the map.**
  `air_boundary.go:39-40` states the rule today as a naming convention: "The EXPORTED `tty.Radio`
  methods are the monitor's control surface; the lower-case internals are shared, and that split
  is the seam." After the split, `startSynth`, `tune` and `tuneResolved` are unreachable from
  outside the deck. So is the guard-free half of the deck.
- **Only the deck package can reach MasterControl's and the arbiter's unexported writers**:
  `cue`, `clearBand`, `hold`, `giveWay` and `takeBack`.

Four things do not change:

- **`TestOnlyMastercontrolWritesTheBand`** (`modes/tty/one_band_writer_test.go:19-23`) stays a scan.
  It polices construction of `tty.TickerBreakingMsg`, which `modes/tty` must export, and Go has no
  friend packages. It is module-wide (`singleowner.Check`) and names its owner by path
  (`"app/mastercontrol.go"`, `:23`). A move without re-pointing that path fails loudly, as it
  should.
- **The deck's own scans still apply inside the deck.** `TestEveryPathToASynthesisedReadGoesThroughTheOneSeam`
  (`app/maintrack_seam_test.go:105`) still has to police callers within it.
- **The rule survives only if the deck's exported surface stays small.** If the 17 lower-case
  `radioDeck` methods the root calls today are exported to make the cut (AS-4), "exported" stops
  meaning "monitor surface". The guard boundary would then be weaker than it is now. Those methods
  are `overlay`, `pushStatus`, `readSynth`, `tuneCallsign`, `watchlistDwell`, `watchlistLang`,
  `stopRead`, `setCast`, `setTones`, `live`, `playing`, `reading`, `resolveAt`, `stopStation`,
  `tuneResolved` and `stopMonitor`, with `Installed` already exported.
- **`TestEverySurfaceSeamIsClassifiedForTheAir`** (`app/air_boundary_test.go:218`, `:234`) reflects
  over `tty.Config` and `tty.Radio`. It is independent of `app`'s package layout and survives
  unchanged.

**Evidence.** The files and lines are cited above. The root's lower-case calls are in the Map I
crossing detail from `cut.py`.

**Consequence.**
- Principle 2, and principle 4's guard: the split can strengthen the air boundary. It does so only if
  AS-4's entry point keeps the deck's internals unexported.

**Recommendation.**
- Make it a PLAN acceptance check: the deck package's exported methods on `radioDeck` equal
  `tty.Radio`'s method set plus the entry point.
- Re-point the owner path in `one_band_writer_test.go`.

**Confidence.** High.

### AS-8 — Ten package-scoped scans narrow silently under any split

**Claim.** Ten tests in `app` police a rule by scanning only their own package's directory. After a
split, each one either moves with the code it polices, or stays in the root. Either way it no longer
sees the other packages. Seven of them would still pass while blind to the new packages:

| Test | Scans | After Map I |
|---|---|---|
| `TestTheProgramsModelIsAlwaysTheRouter` (`router_owner_test.go:17`) | `declset.Files(".")` | stays in the root; blind to the deck and the map (its own log says so, `:63`) |
| `TestTheNeedIsReportedFromTheOneSeam` (`maintrack_seam_test.go:130`) | `"."` for `lineup.NeedsRead` literals | moves to the deck; blind to the root and the map. `NeedsRead` is exported from `platform/lineup`, so no boundary holds it |
| `TestGeodataLoadsOncePerEntryPoint` (`geodata_test.go:16`) | `"."` | stays in the root; blind to the map (`mapnames.go` reads `geodata`) |
| `TestNoProductionFileInAppReadsRuntimeGOOSDirectly`, `…BuildsANameOnlyVoiceSpec` (`cast_test.go:722`, `:861`) | `"."` | move to the deck; blind to the rest |
| `TestNoListComparesZips` (`logic_test.go:167`) | `"."` and `"../modes/tty"` (`:171`), non-recursive | blind to every new subdirectory, on both the `app` side and the `tty` side (TS) |
| `TestEveryLivePipelinesMethodIsReachedFromProductionCode` (`reachable_test.go:29`) | `"."` | the 57 re-homed methods leave its scope |
| `TestTheInjectorIsInEveryBuild` (`inject_test.go:20`) | `"."` and `ticker.go` by name | moves with the ticker; blind to the root |

The other three fail loudly if they are left behind:

- `TestEveryPathToASynthesisedReadGoesThroughTheOneSeam` (`:105`, zero callers is fatal);
- `TestThePowerReportIsNotInsideABranch` (`maintrack_power_test.go:125`);
- `TestEveryReportKindReachesTheComposer` (`segments_completeness_test.go:39`, `:111`), which parses
  `"radio.go"` by name and `"../platform/report/report.go"` by relative depth.

**Evidence.** Each test was read at the cited line. The `grep` was for `declset.Files(`,
`os.ReadDir(".")` and `parser.ParseFile(fset, "` across `app/*_test.go`.

**Consequence.** Principle 4: a no-regression net that narrows silently is the INST-2 failure the
code's own comments warn against ("SILENCE IS A DISTINCT VERDICT").

**Recommendation.**
- Before any code moves, widen each scan to the `app/...` tree. `singleowner.Check`'s module walk is
  the existing shape, with owners named by path.
- Each widening is a TDD step: first seen failing on a fixture file in a second directory.
- Then the moves re-point owner paths, and a stale path fails loudly. (RX, TG)

**Confidence.** High.

### AS-9 — `modes/tty` use, import direction, and per-beat risk

**Claim.**
- **Imports.** Under Map I the root imports `deck`, `maps` and `kit`. `deck` and `maps` import only
  `kit`, never each other, and no child can import the root (Go forbids the cycle).
  `cmd/watchpost` still imports only `app` (`cmd/watchpost/root.go`).
- **`internal/`.** Placing the children under `app/internal/` makes "only the composition root wires
  them" a compiler rule. Nothing in the compiler stops `deck` and `maps` importing each other,
  though. That needs a `lint-imports` row (TG).
- **`modes/tty` use** goes from one importer to three. In non-test files: root 404 uses, deck 199,
  maps 548, kit 9 (`timing.go`).
  - The map's use is almost all the map pane's types: 220 uses into `modes/tty/map_temp.go`, 151
    into `map_prefs.go`, 62 into `status.go`, 40 into `map_radar.go`.
  - The deck's use is the radio and ticker messages: `severe.go` 59, `dashboard.go` 39, `ticker.go`
    33, `setup_cast.go` 17.
  - If TS moves the map pane's types into their own package, `app/internal/maps` can import that
    package instead of all of `tty`.
- **Per-beat.**
  - The one `app` hook on the 50 ms visualiser tick is `radioDeck.Spectrum` (`app/radio.go:171-177`).
    It is handed to `tty` as a method value and is unchanged by a move.
  - `tty.Radio` is already an interface.
  - Map I's new seams are concrete structs and func fields, with no new interface on any hot path.

**Evidence.** The figures are from `cross-package-edges.tsv`, aggregated per candidate package.

**Consequence.** Principle 1 holds if the seams stay concrete. Cross-package calls inline as
same-package ones do.

**Recommendation.** Concrete structs, not interfaces, at every Map I seam. PB to confirm with the
existing allocation pins after the move.

**Confidence.** High for direction and figures. Medium for per-beat, which is static reasoning
pending PB.

### AS-10 — What the cut does to the gate's time, and to the record

**Claim.**
- **Mutants.** Of the 110 `app` mutant anchors, Map I puts 90 in the deck package, 18 in the root,
  1 in `maps`, and 1 in a test file (`air_boundary_test.go`). Map III leaves 108 in the root. Each
  mutant re-runs its own package's suite:
  - Map I: the deck's suite is 56 test files and 14,891 lines;
  - Map III: the root's suite is 101 test files and 19,681 lines;
  - today: `app`'s whole suite is 150 test files and 28,477 lines.
- **P10.** The ratified `app` package-density row (`06_docs/p10-ledger.md:122`) and the
  `app/every.go` `everyTick` row ("THE ONE CLOCK-DRIVEN LOOP IN app/", `:63`) are keyed by path.
  Each new package is measured for density on its own. `everyTick` must stay one function in `kit`
  for the "0 / 1 / n" ruling in that row to hold.
- **Fixed lists and paths.**
  - `TestEveryFileIsInItsPackagesMap` holds only `{"modes/tty", "app"}`
    (`cmd/watchpost/docs_test.go:353`), so new packages are unheld until that list is extended.
  - `docs/where-things-happen.md` names `app/*.go` 97 times, and `docs/extending.md` 26 times.
  - 26 `"../…"` relative paths in 9 `app` test files break on a depth change. They break loudly.

**Evidence.**
- The mutant anchors were grepped (`Path("app/…")`).
- The ledger and `docs_test.go` were read at the cited lines.
- The test sizes are from `cut.py`.

**Consequence.**
- Principle 1, for the gate: Map I roughly halves the test code each `app` mutant re-runs. PB
  measures the real seconds; sweep time does not scale exactly with lines.
- Principle 3: per-package density outcomes need ratification if one fails. Whether each new
  package passes density is **UNVERIFIED**, because `make p10` was not run.

**Recommendation.** TG to enumerate every path-keyed row before PLAN. Mutant re-pointing follows the
required reading's rule 4.

**Confidence.** High for the counts. The density result is UNVERIFIED.

## The three maps, side by side

All figures are computed from the graph. "Crossing" means distinct package-level symbols used across
a boundary by non-test files. "After seams" means AS-2's struct seams are applied.

| | **Map I: deck out, maps out** (recommended) | **Map II: role packages** | **Map III: periphery only** |
|---|---|---|---|
| Packages (non-test files / lines) | `app` 20 / 4,054; `deck` 30 / 9,297; `maps` 22 / 6,616; `kit` 4 / 217 | `app` 25 / 5,035; `deck` 17 / 6,059; `alerts` 8 / 2,257; `maps` 19 / 5,454; `history` 3 / 1,162; `kit` 4 / 217 | `app` 50 / 13,351; `maps` 22 / 6,616; `kit` 4 / 217 |
| Crossing, file-for-file move | 226 (207 unexported) | 277 (253) | 97 (96) |
| Crossing after seams | 198; with AS-4's entry points, root→deck falls from 112 to about 12 plus the entry point | 262 | 81 |
| `livePipelines` methods re-homed | 57 (deck 19, maps 38) | 38 (maps 27, history 11) | 38 |
| Cycles left after seams | none beyond about 17 leaf symbols that move to `kit` or are injected | **inherent**: deck↔alerts (20 / 15 symbols) and history↔maps (10 / 18), each needing interfaces | none beyond leaf symbols |
| Test files whose reach spans packages | 42 | 72 | 32 |
| `app` mutants per package | deck 90, app 18, maps 1 | deck 75, alerts 15, app 18, maps 1 | app 108, maps 1 |
| `tty` importers | 3 (+`kit`) | 5 (+`kit`) | 2 (+`kit`) |
| Compile-time ownership gained | the deck's internals, and the band and duck writers (AS-7) | the same, but the Producer↔arbiter seam exports part of it | none: the deck stays in the root |
| D-2 | happens inside one package | crosses the deck/alerts boundary it would create | happens in the root |

**Map II, a variant.** Map IIb puts the executors above the deck instead. It is worse on the measure
that matters: 57 unexported symbols must be exported (AS-6).

**Map III** is the cheapest and the safest. It moves only the cycle that touches no audio. It leaves
a 13,351-line root that still holds the deck, so principle 2's gain is small, and the deck's rules
stay conventions.

**Recommended: Map I.**
- It follows the two cycles the code already has (AS-1).
- It moves the deck whole (AS-3).
- It turns the deck's private/public convention into a compiler rule (AS-7).
- It puts 90 of 110 mutants on a suite half the size (AS-10).
- It leaves D-2 a single-package change (AS-6).

**The strongest argument against Map I:**
- The deck package is still 9,297 lines with 14,891 of tests, which is 1.55 times `platform/render`.
- It is not a pure move. It needs:
  - two new structs and 57 re-homed methods (AS-2);
  - the `dashboard.go` split (AS-4);
  - new locks for the bed and the history (AS-5);
  - 42 test files split by assertion;
  - most of the 124 `livePipelines` test literals rewritten.
- D-1's "byte-identical behaviour" rests on those seams being done right. Map III defers all of it.
- My answer: the size cannot be reduced without the exports AS-6 rules out. The seams are the same
  ones any later split would need. Map III only postpones them, behind a root that keeps growing.

## Open questions for the HUM LEAD

1. **The package map** (D-1's fork): Map I, Map II or Map III. This lens recommends Map I.
2. **Whether the deck's further cut, the Producer, waits for D-2's ruling** (AS-6). This lens
   recommends that it waits. It is a scope question because it sets which release carries the
   remaining size.
3. **The bed's and the history's own locks** (AS-5). The aim is no behavioural change. It is raised
   only because it changes a concurrency shape the 0.16.0 record drew, and the HUM LEAD may want it
   ruled rather than taken as an A-row.

## Touches other lenses

- **TS**: `TestNoListComparesZips` scans `../modes/tty` non-recursively (AS-8). The map pane's types
  decide what `app/internal/maps` imports (AS-9).
- **RM**: this lens agrees that the arbiter, MasterControl and the Reader are inseparable, and
  measures why: 57 unexported symbols if the executors leave the deck (AS-6). D-2 then stays inside
  one package.
- **RX**: the 42 test files whose reach spans packages, the 124 test literals, the 10 narrowing
  scans, and `-race` over the new locks (AS-2, AS-5, AS-8).
- **TG**: owner paths, `declset.txt` per package, `docs_test.go:353`'s list, the P10 rows,
  `where-things-happen.md`'s 97 paths, mutant anchors, and a `lint-imports` row between the children
  (AS-7, AS-9, AS-10).
- **PB**: the per-mutant suite size under each map, and the allocation pins after the move (AS-9,
  AS-10).
- **GP**: concrete structs over interfaces at the seams, and `internal/` placement (AS-2, AS-9).
- **JN**: replacing `livePipelines` is the largest navigation gain in `app` (AS-2).
- **RC**: `startSynth` and `maintrack.go`'s disposition ("Instruct: it stays where it is") is about
  the role's place, not the file's path. Moving it with the deck whole respects it; RC should
  confirm that reading.
