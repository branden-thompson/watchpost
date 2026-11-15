---
title: "0.19.6 re-architecting — IMPLEMENTATION PLAN"
date: 2026-11-15
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT for the plan red team, then the PLAN gate. No code: signatures, test descriptions, file paths and order only (0.18.0 D-13)."
---

# Implementation plan

**Goal.** Meet the brief (`../01-objectives/project-brief.md`, D-13, D-18) and carry out the settled
package map (`../02-analysis/package-map.md`) and D-1 to D-26, A-1 to A-27, without a regression. The
layout is drawn in `../03-architecture-design/architecture.md`, and its signatures are in the three
designs beside it.

**Branch.** `feature/re-architecting`, cut from `main` at `e57afeaf` (A-1), squash-merged as
`release/0.19.6` at SHIP.

**Users notice nothing** (R1). No output, sound or timing changes. A golden that changes is a
regression, never a ruling candidate.

## Every batch follows the same order

1. **Re-check** each fact the batch relies on against the code at its `file:line`. A fact that no
   longer holds is recorded in the build log, and the batch's plan is re-checked before code changes.
2. **Instruments first** (R2, C-11, INST-2). Every instrument the batch depends on exists from batch
   0. The move commit first shows them failing on a deliberately partial move: the test inventory, the
   union declaration check, and the graph tool's crossings.
3. **The prep commit, in place** (R3). Receivers re-homed, names exported, seams added, tests changed
   under D-14 and D-22, anchors re-pointed under A-25. Nothing moves package.
4. **The move commit, move only** (R3, B-10). `git mv`, package clauses, imports and qualifiers, the
   `doc.go` file maps, P10 and baseline re-keys, mutants re-pointed by path, the paths documents and
   comments name, and the atlas. The body-hash union check proves nothing else changed (A-12).
5. **The gate.**
   - Every touched package, whole, with `-race`; `make alloc-budget` where a pin's test moved.
   - The pin comparator (C-8, A-10) against `e57afeaf`'s baseline: no count above it, no byte figure
     more than 0.1 % above it, and no baseline pin without a measurement.
   - benchstat on every benchmark in the two trees, paired back to back at `-count 10`, mandatory at
     2a and 2b (M1, B-2).
   - The escape list by function and type, and inlined calls counted by callee across the union of a
     split function (A-26).
   - `make p10` (A-7), its summary in the build log.
   - `make verify`, with MAKE_EXIT and "ALL GATES GREEN" read before the commit, which is chained to the
     verdict. `go run ./tools/ids -write` and `git add -N` on new files first.
6. **The scoped sweep** (C-12, A-11). The batch's mutants, derived (A-11), run with A-3's killers and
   compared mutant by mutant with A-3's record, read from `7c5e5772`. A changed killer fails the batch;
   D-22's drawn-value changes pass only on the same killers.
7. **The changed-line table** (D-14). Every changed test line, marked setup or assertion, each
   assertion with its mutants before and after. It is in the batch's build-log entry.
8. **The record moves with the code**, in the batch's own commits (R6): the as-built map and its
   diagrams, the atlas (`go run ./tools/atlas`), the `doc.go` file maps, and every living document row
   in package map §9.
9. **Push** only when no hosted run is in progress on the branch.

## Before BUILD: the pilot (A-7)

A move of one small file in a scratch copy, through prep and move commits, measured:
- whether `make p10` counts a byte-identical moved function as changed (D-10); the answer is written
  into the re-key table;
- a same-name constructor pair (`deck.New`, `mapdata.New`) and a same-name method pair (an adapter and
  the model method it calls) under P10-01's bare-name rule (CQ F-16, A-22);
- a move with package qualifiers, through the body-hash union check (H-9);
- the generic engine's recursion under the P10 checker (CK, D-23);
- cross-package inlining of a one-line forwarder (`SetMirror`, `Deck.SetLocations`), by `-gcflags=-m`.

A result that differs from this plan is put to the HUM LEAD before batch 0 lands.

## Batch map

| Batch | What | Why it stands alone |
|---|---|---|
| B0 | the instruments (package map §4, A-12, D-14 to D-24's checks), go 1.26.11 and the floor scan (D-12) | nothing moves until every instrument is seen failing and then passing |
| B1 | RM's slices 1 and 3: F-158's owed test, the vocabulary, RC-7's and JN-5's doc errors | changes no production behaviour; fixes docs before any move makes them harder to fix |
| B2 | `modes/tty`: the kit (`internal/frame`) and the test support (`internal/ttytest`) | the console's prerequisite; the first new packages |
| B3 | `modes/tty`: the console (`modes/tty/console`) | a move with a narrow seam; its killers sit in its own tests (RX-3) |
| B4 | `modes/tty`: the map's 2a, in place | the largest re-homing, with every test still in its package; 2b's straddle count re-measured |
| B5 | `app`: the kit (`internal/run`) and the test helpers (`internal/apptest`) | the deck's and the maps' prerequisite |
| B6 | `app`: the deck (`internal/deck`), whole, with RM's slice 2 | the shared deck moves as one unit (K-3) |
| B7 | `app`: the map data (`internal/mapdata`) | after the deck, so the shared lock's seams are proved once |
| B8 | `modes/tty`: the map's 2b and the controls (`mapview`, `internal/controls`), three commits | last: its net is the most rewritten (RR F12) |
| B9 | the closing sweep, the figures, the REVIEW inputs | the release's verdict against A-3 |

Eight code batches after B0 (B1 to B8), about 18 full gates (D-18).

---

## B0: the instruments

Each item is written test-first: seen failing on a planted case, then passing on today's tree. Each
records a fixed floor, never one taken from the tree it scans (CQ F-18).

- **B0.1 The sweep's killers after a release (A-2).** `make mutant-verdicts` orders the killer records
  by the version in their names, prints the record it used, and fails when a named record is missing
  (H-4).
- **B0.2 The verdict comparator (RX-12, A-9, A-11).** Compares a sweep's verdicts with A-3's, read by
  `git show 7c5e5772:06_docs/mutants/verdicts/since-v0.19.5.log`, mutant by mutant; asserts it judged
  exactly the derived list; reports a change in a mutant's target file or anchor-occurrence count. Seen
  failing on a planted flipped verdict.
- **B0.3 Quiet guards made loud (RX-9, TG-2, RC-3, GP-7, AS-8, TS-12).** Each of the 22
  directory-scoped scans walks its package tree; the file-map test derives its packages; `property`,
  `vet-tags`, `quality-bench` and `test-platforms` name `./...` trees or fail on no tests run; the
  band-writer check stops matching the qualifier `tty`; the `wires` sentinel and TG-6's three
  export-sensitive checks are pinned. The reply-routing guard fails on a stale excuse, such as
  `"tickMsg"` once the kit moves it (`modes/tty/router_lookup_test.go:134-137`).
- **B0.4 `TestMain` invariants that travel (TG-3, RX-6, CK-7, D-24).**
  `TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain` in `cmd/watchpost`: every package
  under `modes/tty/` calls `ttytest.Main`, every package under `app/` calls `apptest`'s invariants
  helper, and every package whose production code reads `runtimeGOOS` applies `WATCHPOST_TEST_GOOS`.
  Floor 2 packages; a planted package without one is reported. `TestNoTestSetsTheZoneMidRun` walks the
  tree, not `.`.
- **B0.5 Proof that a move lost nothing (RX-10, TG-7, RC-8, A-12).** The test inventory over both
  trees records each test's outcome, PASS or SKIP, subtests included, with and without `-tags
  property` and `-race`. The union declaration check hashes bodies with package qualifiers
  normalised. A golden gate fails when a file under `testdata/` differs from `e57afeaf` except as a
  pure rename, seen failing on a planted re-capture.
- **B0.6 The byte-identical traces (RX-8, D-21, A-12).**
  - The router trace's new form (design CK-6): messages keyed by index into a widened, fixed alphabet;
    the clock set through `NewRouter` once B3 lands D-21's `Opening.Now` (B0 keeps today's console
    clock line, and B3 deletes it with the golden unchanged); commands described by shape (`tick`,
    `every`, `batch(…)`, `seq(…)`, `fn`, `-`).
  - The `app` trace, `TestTheDeckIsTheSameUnderRandomActs` (design AP-7.2): four fixed seeds, 150 acts
    each, every act and effect by stable key, the golden recorded on code identical to `e57afeaf`.
- **B0.7 The performance instruments (PB-1, PB-2, PB-7, C-8, A-10, A-17).** The pin comparator, its
  baseline the maximum over 10 runs on `e57afeaf` with the count recorded, the window pins with byte
  figures; the three race-skipped pins under `make alloc-budget`, with a gate test that refuses one left
  outside; the tone-latency benchmark's 185 allocations as a pin; the escape probe by function and
  type, the model-size probe, the inlining count per callee (A-26). Startup and binary size recorded.
- **B0.8 The order of things that must not move.** `windowLayers()`'s order (`app/maplayers.go:205-211`,
  PB-4); a test-held list of the post-update steps, parsed from `Dashboard.update`
  (`modes/tty/dashboard.go:1156-1168`), seen failing on a swapped pair; a behaviour test that a
  non-map message still makes `armPropOptions` re-ask (design MS-5); the pinned set of message types
  across the tree (R8, RR F7).
- **B0.9 The import rules (TG-1, TG-14, DQ-6, D-15, A-24).** `scripts/lint-imports.sh` holds:
  `modes`, `platform` and `domains` never import `app`; `platform` never imports `modes`; `deck` and
  `mapdata` never import each other; `ttytest` and `apptest` imported by `_test.go` files only. Each
  rule has a planted specimen in its `--self-test`. A check that no type alias survives (GP-11).
- **B0.10 The citation lint keyed on the package root (TG-4, RC-5)**, and "`git log` on the file"
  read as `git log --follow` (`docs/where-things-happen.md:147-149`, `06_docs/ids.md`).
  `cmd/watchpost/extending_symbols_test.go:57` matches by path, not base name (TG-10).
- **B0.11 The shared lock (D-16, A-19).** `TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage`,
  floor 25 sections, seen failing on a planted fixture under `app/testdata/sharedlock/`; and
  `TestCommitAndStopAllFinishUnderTheSharedLock`, 200 iterations against a 2 s bound scaled under
  `-race`, seen failing under vetted mutants `mDL1` and `mDL2` (design AP-3.5, AP-3.6).
- **B0.12 The surfaces held.** The deck's exported surface, tiers 2 and 3 against `app` (design
  AP-1.5); the three pre-existing exports of D-26, listed exactly; the graph tool committed as its own
  module under `tools/`, run by a make target, counting reach across packages as within one, through
  `Env` and the hooks, seen failing on a planted pass-through, its `e57afeaf` outputs committed (A-12).
- **B0.13 The release record and the floor (A-12, D-12).** `releaseRecord` points at 0.19.6
  (`cmd/watchpost/record_test.go:19-20`); `govulncheck` at the `go` floor in CI and the release job,
  seen failing at 1.26.10, then the floor raised to go 1.26.11.

## B1: RM's slices 1 and 3, and the doc errors

- **B1.1 F-158's owed test** (design AP-7.1): `TestACutBackUnderAHazardReadKeepsTheBedDown`, written
  as `app/cutback_hazard_test.go`, its reach deck-only by the graph tool; seen failing under the new
  vetted mutant `mF158` at `app/executors.go:301`. Its first assertion is a precondition that fails
  with "no longer exercises F-158" when 0.19.7's D-2 changes the step.
- **B1.2 The vocabulary** (RM slice 3; JN-5): the misleading names RM-8 and JN-5 found, as A-rows; the
  `doc.go` and `docs/` rows.
- **B1.3 RC-7's six disagreements** (package map §9 row 2): one window where the code has a stack
  (`docs/where-things-happen.md:15`, `:95`, `:128`; `docs/extending.md:291-294`; the comments at
  `modes/tty/dashboard.go:641`, `:1590`).

- **B1.4 M4's before, its doc findings** (`../08-reports/m4-before.md`): Walkthrough 6's golden for a
  map row (`docs/extending.md:190`) and its "Search for" sentences (`:173-174`);
  `docs/where-things-happen.md:67`'s time row, naming `cardPulledShort` and `spokenWhen` as the places
  that choose a date; `UIPrefs.MapPropRefresh`'s comment (`modes/tty/dashboard.go:458`);
  `modes/tty/broadcaster_manifest.go:121-122`'s cell count; a CHANGELOG `[Unreleased]` heading, and a
  walkthrough line that every user-visible change gets an entry.

## B2: the kit and the test support (`modes/tty`)

- **Prep:** TS-4's closure cut into the kit's files in place (design CK-1: `masthead.go`, `row.go`,
  `cursor.go`, `tick.go`, `memo.go`, `text.go`); 25 names and 5 methods exported; `MemoStats` with its
  embedded mutex (A-21). The engine's helpers renamed only (`PerturbEach`); the pure test helpers
  gathered (`StripANSI`, `FirstDiffLine`, `Card`, `MastheadSnap`, `LoadedPoolSize`, `Main`).
- **Move:** `modes/tty/internal/frame` and `modes/tty/internal/ttytest`.
- **P10:** D-15's P10-09 row and D-23's two P10-01 rows written for ratification in this batch.
- **Tests:** the masthead guard split by surface, with the named inputs in a kit-level test (design
  CK-4); `kit-CK.tsv` and `tests-CK-refs.tsv` are the batch's lists.
- **Mutants:** 12 kit anchors re-point by path; 12 anchors that name a kit symbol re-point under A-25.

## B3: the console (`modes/tty/console`)

- **Prep:** `console.New(Opening)`, `SetMirror(Mirror)`, `SetListenerMuted`, `WithSnapshot`,
  `WithRecent`, `Scoped`, the 15 Router calls exported and the 5 getters added (design CK-2, A-21); the
  card window cut into `card_window.go`; D-21's `Opening.Now`, with the router trace's console-clock
  line deleted and its golden unchanged.
- **Tests (D-14, D-22):** `tests-CK.tsv`: 73 setup-only references, 23 same-value, 12 moved, and the 20
  of D-22's class, each listed in the changed-line table with its killers.
- **Move:** the twelve `broadcaster*.go` files and their tests; `app`'s 11 non-test sites re-qualified
  for the six messages (C1).
- **Mutants:** 77 re-point by path; 14 anchors per A-25.
- **P10:** `modes/tty/broadcaster_rail.go`'s row re-keys; the eight `Router` and `Dashboard.View`
  P10-01 rows re-proved with the console in its own package.
- **Docs:** `0.16.0-broadcaster-ui/00-REQUIRED-READING.md`'s `broadcaster_rail.go` row; the vocabulary
  row's `console/broadcaster.go:New`.

## B4: the map's 2a, in place

- **Prep, in steps** (design MS-1, A-22):
  1. `mapModel` declared and embedded in `Dashboard`; the map's fields moved onto it; no method moves.
  2. `Env`, `Want`, `Step` and `Config.Map` (the hooks held once); `app/dashboard.go:424-466`'s
     assignments become one keyed literal.
  3. The 387 methods re-homed, in file groups; each of the 11 split sites with its own test, seen
     failing on a wrong order.
  4. The 41 hook writes made after construction audited for the shared pointer.
- **Tests:** 798 call references in 68 files re-pointed; no assertion changes (design MS-7).
- **Mutants:** 146 anchors re-pointed in the prep commit, reviewed as A-11's old-and-new table, then
  the scoped sweep.
- **Re-measure:** 2b's class C4 by `tests_fn_ms.py` on 2a's code, reported under D-22 before B8.
- **Gate:** benchstat mandatory (B-2); the sizes probe records `Dashboard`'s and the model's bytes.

## B5: the kit and the test helpers (`app`)

- **Prep:** the seven shared helpers gathered and exported (`EveryTick`, `LockedCopy`,
  `SavePreference`, `UserCacheSubdir`, `CacheDir`, `UserAgent`, `RequestStats`); `app`'s pure test
  helpers gathered.
- **Move:** `app/internal/run` and `app/internal/apptest`.
- **P10:** `app/every.go`'s P10-02 row re-keys by path and by symbol (`everyTick` to `EveryTick`).

## B6: the deck (`app/internal/deck`), whole

- **Prep** (design AP-1, AP-2, A-23): `deck.Config`, `deck.New`, the ten methods; `RunDashboard`'s
  order kept step for step; `stationConfig` split; the station seam (`Watch`, `Pool`, `Station`);
  the deck-side fields and 35 methods onto `Deck`; `CastBlock` behind the root's `RenderCast`.
- **D-24:** the deck's own `TestMain` applies `WATCHPOST_TEST_GOOS`; `make test-platforms` over
  `./app/...`.
- **Tests:** `tests-AP.tsv`'s deck rows; the deck's exported surface test gains tier 1.
- **Move:** the 29 deck files (A-23: `saychildren.go` and `hangup.go` stay at the root) and their
  tests; the two traces' goldens renamed with them.
- **Mutants:** 116 re-point; 11 anchors change text.
- **P10:** `app/cast.go`'s row re-keys to the deck (D-24).
- **Docs:** `0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md`; `follow-ups.md`'s
  F-158 paths.

## B7: the map data (`app/internal/mapdata`)

- **Prep:** `mapdata.Config`, `mapdata.New`, the eleven methods (design AP-4); the maps-side fields
  and 40 methods onto `Feeds`; `lazymemo.go` and `timing.go` with them; D-22's one test (m497, m498)
  served by the `diagSources` interface member; the `FillStats` call pinned at the root with a new
  mutant (A-23).
- **Move:** the 22 files and their tests.
- **Docs:** `map-ready-geometry/03-architecture-design/as-built.md`; `observer-maps`' as-built map.

## B8: the map's 2b and the controls, three commits

1. **The straddling tests re-wired in place** onto the API that survives (D-22), the guard engine's
   named sub-model rule added and seen failing on a planted omission (A-22).
2. **The straddle count re-measured**, and the result reported.
3. **The move:** `modes/tty/mapview` and `modes/tty/internal/controls` (D-25); `app`'s 14 files
   re-qualified from `tty.` to `mapview.`.

- **Mutants:** 289 re-point by path; 7 anchors change text in the first commit.
- **Gate:** benchstat mandatory.
- **Docs:** the map package's `doc.go` names its four under-the-map rows with one word for the legend
  (JN-3).

## B9: the closing sweep and the REVIEW inputs

- The full sweep against A-3's record: every A-3 mutant keeps its class or is re-ruled; every new
  mutant is caught; none has no evidence (M7, B-9).
- M1 to M6's figures against the baseline; `README.md:597` and `Makefile:35-41` restated with the
  measured slowest package.
- M4's blind trace on the new layout, against a fresh key built by leave-one-out before the docs are
  rewritten (M4, D-24 of 0.19.5).

---

## For ratification at PLAN exit

### M3's targets (D-13; designs MS-9, CK-9)

| Figure | Today | Target |
|---|---|---|
| The map's reach into the model (JN's scope) | 72 | at most 23 |
| The same, by type | 93 | at most 40 |
| `Dashboard`'s methods and files | 752 in 68 | at most 374 in 38 |
| The console's reach into the Observer | 13 | 0 |
| The legend-key task | at least 5 files, 1 package | at most 3 files, 1 package |

### The P10 re-key table (D-10)

| Ledger row (`06_docs/p10-ledger.md`) | Today | After | Batch | Proof |
|---|---|---|---|---|
| `:18`, `:212`, `:213` | `app/dump_windows.go`, `app/dump_unix.go` | unchanged | — | — |
| `:49` to `:55` | `modes/tty/router.go`, seven `Router` methods (P10-01) | unchanged path | B3, B8 | re-proved: the surfaces' `Update` and `View` are in other packages, and neither holds a Router field |
| `:56` | `modes/tty/view.go`, `Dashboard.View` (P10-01) | unchanged path | B4, B8 | re-proved after the model's methods leave |
| `:63` | `app/every.go`, `everyTick` (P10-02) | `app/internal/run/every.go`, `EveryTick` | B5 | the reason unchanged |
| `:122` | `app` density | `app` density; its children ride it by prefix until BUILD exit (D-10) | B5 to B7 | floors measured on the finished code |
| `:129` | `modes/tty` density | the same, for `modes/tty`'s children | B2 to B8 | the same |
| `:172` | `app/cast.go` (`runtimeGOOS`) | `app/internal/deck/cast.go` | B6 | D-24 |
| `:190` | `modes/tty/broadcaster_rail.go` | `modes/tty/console/broadcaster_rail.go` | B3 | the reason unchanged |
| new | — | `modes/tty/internal/ttytest/perturb.go`, P10-09 (`unsafe`, `reflect`) | B2 | D-15 |
| new | — | `walkAt` and `bump`, P10-01 | B2 | D-23 |

Whether a byte-identical moved function counts as changed comes from the pilot, and is written here
before the table is ratified.

## M4's four tasks

Fixed at PLAN by the key builder, and traced blind on `e57afeaf` as M4's before (B-6). The tasks and
the before's grade, 0 of 4, are in `../08-reports/m4-before.md`, the keys in `../08-reports/m4-key/`.
The REVIEW trace runs on a copy with this release's folder deleted, and `06_docs/ids.md` is checked to
hold no path into `m4-key/` (0.19.5 batch 43).

## What this plan does not do

D-2 proper and everything A-8 lists (0.19.7, D-11); any Charm or Bubble Tea upgrade (C-9); any
behaviour change (R1); any package outside `modes/tty` and `app` (scope).
