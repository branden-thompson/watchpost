---
title: "0.19.6 re-architecting — IMPLEMENTATION PLAN"
date: 2026-11-15
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "APPROVED at the PLAN gate (D-33), after the plan red team (`../08-reports/red-team-plan.md`) and the pilot (`../08-reports/pilot-report.md`). No code: signatures, test descriptions, file paths and order only (0.18.0 D-13)."
---

# Implementation plan

## In short

0.19.6 splits `modes/tty` and `app` into eleven packages in ten batches, B0 to B9, and users notice
nothing. Batch 0 builds every instrument before any code moves; B1 fixes the docs M4's before found
wrong; B2 to B8 each move one seam in a prep commit and a move commit; B9 runs the closing sweep. About
24 full gates, about 16 hours at the 2,408-second median (restated from D-18's 18 by the red team,
BQ-11).

**For ratification at this gate** (the tables are at the end):
- M3's targets, as D-27 set them;
- the P10 re-key table, with the pilot's results written in (D-10, D-31, D-32);
- this plan, with the red team's dispositions (A-29 to A-33).

**Goal.** Meet the brief (`../01-objectives/project-brief.md`, D-13, D-18) and carry out the settled
package map (`../02-analysis/package-map.md`) and D-1 to D-32, A-1 to A-33, without a regression. The
layout is drawn in `../03-architecture-design/architecture.md`, and its signatures are in the three
designs beside it; where a later ruling chose differently, the ruling stands.

**Branch.** `feature/re-architecting`, cut from `main` at `e57afeaf` (A-1), squash-merged as
`release/0.19.6` at SHIP. Each push's remote ref is recorded in the build log (PH F-1).

**Users notice nothing** (R1). No output, sound or timing changes. A golden that changes outside B0's
listed re-recordings is a regression, never a ruling candidate (A-29).

## Every batch follows the same order

1. **Re-check** each fact the batch relies on against the code at its `file:line`. A fact that no
   longer holds is recorded in the build log, and the batch's plan is re-checked before code changes.
2. **Instruments first** (R2, C-11, INST-2). Every instrument the batch depends on exists from batch
   0. A test the batch's change must keep green (the split sites, the root wiring) lands in a commit
   before the change, green on today's code (A-31, A-32). The move commit first shows the instruments
   failing on a deliberately partial move: the test inventory, the union declaration check, the graph
   tool's crossings.
3. **The prep commit, in place** (R3). Receivers re-homed, names exported, seams added, tests changed
   under D-14 and D-22, anchors re-pointed under A-25. In the same commit: the P10 ledger's symbol
   re-keys for every row whose symbol is renamed, the mirror regenerated from the local ledger, and the
   declaration-set goldens re-captured (the pilot). Nothing moves package.
4. **The move commit, move only** (R3, B-10). `git mv` of the code and its tests, package clauses,
   imports and qualifiers (tests' included), the `doc.go` file maps, the P10 ledger's path re-keys,
   the parent's declaration-set golden, mutants re-pointed by path, the paths documents and comments
   name, and the atlas. The body-hash union check proves nothing else changed (A-12). `git diff -M
   --name-status main` lists the moved pairs: an unpaired or `R100` file is judged whole by `make p10`
   (the pilot).
5. **The gate.**
   - Every touched package, whole, with `-race`; `make alloc-budget` where a pin's test moved.
   - The pin comparator (C-8, A-10) against `e57afeaf`'s baseline: no count above it, no byte figure
     more than 0.1 % above it, no baseline pin without a measurement, and a pin whose test or fixture
     changed equal to its baseline (D-28).
   - `make quality-bench`, benchstat pinned: two back-to-back paired runs at `-count 10` on a quiet
     machine, every baseline benchmark paired after a move; both runs showing a significant slowdown
     (p < 0.05) of 2 % or more fails the batch and goes to the HUM LEAD (D-28).
   - The escape list by function and type, and inlined calls counted by callee, both keyed through the
     re-home and move maps, with a per-package total by type as a backstop (A-26, RR-12). Model sizes
     not larger (D-28).
   - `make p10` (A-7), green at the end of every batch, its summary in the build log. A new package's
     density row and any `New`-to-`New` P10-01 row are put to the HUM LEAD in the batch that creates
     them (D-31, D-32).
   - `make verify`, the commit gate, with MAKE_EXIT and "ALL GATES GREEN" read before the commit, which
     is chained to the verdict. Its mutant check judges the mutants the commit's change reaches (D-38).
     `go run ./tools/ids -write` and `git add -N` on new files first, and nothing written into the tree
     while it runs.
   - `make verify-full` at the batch's close, on its last commit: the whole mutant corpus (D-38). A
     batch is not closed, and nothing merges, without it.
   - Every heavy run at a low priority, on the efficiency cores: `taskpolicy -b nice -n 19` with
     `GOFLAGS=-p=2` and `GOMAXPROCS=4` (D-37 to D-39, the 2026-11-16 overheating).
6. **The scoped sweep** (C-12, A-11, A-30), `make mutant-scoped` (D-30), run locally (D-39). Its list is derived: mutants
   in a moving file, mutants whose last killer's file moves, mutants whose enclosing function's body
   hash changed, and mutants whose last killer has a row in the changed-line table. It runs with A-3's
   killers and is compared mutant by mutant with A-3's record, read from `7c5e5772`; a changed killer
   (by package and name), or an anchor whose enclosing function does not map through the re-home and
   move maps, fails the batch.
7. **The changed-line table** (D-14, D-22). Every changed test line, marked setup or assertion, each
   assertion with its mutants before and after; a D-22 assertion with no mutant gets a vetted mutant
   before it changes (A-30). The test inventory's per-batch delta (a test's identity is its name and
   package, mapped through the move map; a split's renamed half is a row) beside it. Both are in the
   batch's build-log entry.
8. **The record moves with the code**, in the batch's own commits (R6): the as-built map and its
   diagrams, the atlas (`go run ./tools/atlas`), the `doc.go` file maps, every living document row in
   package map §9, and a comment pass that fixes comments the move made false and gives a bare D-ID in
   moved code its release (A-33).
9. **Push** only when no hosted run is in progress on the branch.

## The pilot (A-7), done

`../08-reports/pilot-report.md`, on a scratch copy of `e57afeaf`, moved `everyTick` to
`app/internal/run` as `EveryTick` in two commits:
- **P10 and a moved function:** a cross-package move that git pairs is judged by changed line, so a
  byte-identical body is not judged again, and a function whose signature the prep changed is; a
  same-package rename with no edit, a split, and an unpaired file are judged whole. Every row is
  re-keyed either way: a stale key fails `make p10` in the second case and
  `TestEveryRatifiedP10RowNamesCodeThatExists` in both. The symbol re-key belongs in the prep commit.
- **Density:** a new package does not ride its parent's row (D-31).
- **Same names:** P10-01 flags every same-name call, across packages or within one, and generic
  recursion (D-23, D-32).
- **The move diff:** one package clause, three imports, four qualifier lines, plus the test callers
  and the declaration-set golden; nothing else.
- **Cost:** the forwarder inlines across the boundary; escapes unchanged; `go test ./app/...` within
  noise.

## Batch map

| Batch | What | Commits, full gates | Why it stands alone |
|---|---|---|---|
| B0 | the instruments (package map §4, A-12, A-29), go 1.26.11 and the floor scan (D-12) | 13, 12 and the closing `verify-full` (planned 4, 4: D-37 and D-38 split the gate, and B0.6, B0.7 and B0.12 each took more than one commit) | nothing moves until every instrument is seen failing and then passing |
| B1 | RM's slices 1 and 3, the doc errors, M4's doc findings; then M4's midpoint trace (D-29) and D-50's ID families | 6, 5 and the closing `verify-full` (planned 2, 2: B1.1 to B1.4 each took a commit of their own, and the midpoint's grade added B1.6) | changes no production behaviour; fixes docs before any move makes them harder to fix |
| B2 | `modes/tty`: the kit (`internal/frame`) and the test support (`internal/ttytest`) | 6, 5 and the closing `verify-full` (planned 2, 2: the instruments took a commit first (A-49), the move's first build found what the prep had left, a second prep (D-62, D-63), and the closing sweep a trace that had stopped reaching three paths since B0.6 (D-67, D-68)) | the console's prerequisite; the first new packages |
| B3 | `modes/tty`: the console (`modes/tty/console`) | 6, 5 and the closing `verify-full` (planned 2, 2: the instruments took two commits, one before the prep (A-57) and one before the move (A-60), and the prep two, the console's API in place (A-58) and then the tests across its boundary (A-59)) | a move with a narrow seam; its killers sit in its own tests (RX-3) |
| B4 | `modes/tty`: the map's 2a, in place | 4, 4 | the largest re-homing, with every test still in its package; 2b's straddle count re-measured |
| B5 | `app`: the kit (`internal/run`) and the test helpers (`internal/apptest`) | 2, 2 | the deck's and the map data's prerequisite |
| B6 | `app`: the deck (`internal/deck`), whole, with RM's slice 2 | 2, 2 | the shared deck moves as one unit (K-3) |
| B7 | `app`: the map data (`internal/mapdata`) | 2, 2 | after the deck, so the shared lock's seams are proved once |
| B8 | `modes/tty`: the map's 2b and the controls (`mapview`, `internal/controls`) | 3, 3 | last: its net is the most rewritten (RR F12) |
| B9 | the closing sweep, the figures, the REVIEW inputs | 1, 1 | the release's verdict against A-3 |

24 full gates, plus a scoped sweep in each code batch and the closing sweep. RM's slice 4 is done:
package map §8 records RM-1's table and RM-5's end state (`../02-analysis/lens-RM-role-model.md:438`).

---

## B0: the instruments

Each item is written test-first: seen failing on a planted case, then passing on today's tree. Each
records a fixed floor, never one taken from the tree it scans (CQ F-18). Every instrument that can be
a Go test lives in the suites `make verify` and CI already run (D-30).

- **B0.1 The sweep's killers after a release (A-2).** `make mutant-verdicts` orders the killer records
  by the version in their names, prints the record it used, and fails when a named record is missing
  (H-4).
- **B0.2 The verdict comparator and the scoped sweep (RX-12, A-9, A-11, A-30, D-30).** The comparator
  reads A-3's record by `git show 7c5e5772:06_docs/mutants/verdicts/since-v0.19.5.log`, judges mutant
  by mutant, asserts it judged exactly the derived list, and records each anchor's enclosing function
  at baseline. `make mutant-scoped` feeds the derived list, writes its own log, never promotes, runs
  `cache-clean`, and is classified in `gates_test.go`'s exempt table by D-30. Seen failing on a
  planted flipped verdict and on an anchor re-pointed into another function.
- **B0.3 Quiet guards made loud (RX-9, TG-2, RC-3, GP-7, AS-8, TS-12).** Each of the 22
  directory-scoped scans walks its package tree; the file-map test derives its packages; `property`,
  `vet-tags`, `quality-bench` and `test-platforms` name `./...` trees or fail on no tests run; the
  band-writer check stops matching the qualifier `tty`; the `wires` sentinel and TG-6's three
  export-sensitive checks are pinned. The reply-routing guard fails on a stale excuse, such as
  `"tickMsg"` once the kit moves it (`modes/tty/router_lookup_test.go:TestEveryWindowReplyIsRoutedBackToTheWindow`). `scripts/quality/p10-unmatched.sh`'s
  header says its prefix rule is its own, not the tool's (D-31).
- **B0.4 `TestMain` invariants that travel (TG-3, RX-6, CK-7, D-24, A-29).** B0 creates
  `modes/tty/internal/ttytest` holding `Main` alone and `app/internal/apptest` holding the invariants
  helper alone (the temporary config and state directories, UTC, `WATCHPOST_TEST_GOOS`); both are
  imported by `_test.go` files only. `TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain` in
  `cmd/watchpost` requires every package under `modes/tty/` to call `ttytest.Main` and every package
  under `app/` to call `apptest`'s helper, and every package whose production code reads `runtimeGOOS`
  to apply `WATCHPOST_TEST_GOOS`. Its floor is the fixed list of the packages that exist, raised in
  each batch that adds one; a planted package without one is reported. `TestNoTestSetsTheZoneMidRun`
  walks the tree, not `.`.
- **B0.5 Proof that a move lost nothing (RX-10, TG-7, RC-8, A-12, A-29, A-30).** The test inventory
  over both trees records each test's outcome, PASS or SKIP, subtests included, with and without
  `-tags property` and `-race`, a test's identity its name and package. The union declaration check
  hashes bodies with package qualifiers normalised. The golden gate's baseline is B0's closing commit:
  B0's re-recorded and added goldens are listed exactly, and a file under `testdata/` that differs
  from it, or is added outside that list or a later batch's named list (A-33), fails; a pure rename
  passes. Seen failing on a planted re-capture and on a planted addition.
- **B0.6 The byte-identical traces (RX-8, D-21, A-12, A-29).**
  - The router trace's new form (design CK-6), widened as RX-8 asked: its Dashboard is built with a
    real map (`embeddedMap`, offline feeds, a recording `SetUI`, a Propagation seam, as
    `map_golden_test.go` and `map_alloc_test.go` build theirs); messages are keyed by index into a
    widened, fixed alphabet that reaches the map, Propagation and Settings; each returned command is
    run, bounded and synchronous, and named by the type of message it returns; the clock is set
    through `NewRouter` once B3 lands D-21's `Opening.Now` (B0 keeps today's console-clock line, and B3
    deletes it with the golden unchanged). Recorded on code identical to `e57afeaf`; the re-recorded
    golden is on B0's list.
  - The `app` trace, `TestTheDeckIsTheSameUnderRandomActs` (design AP-7.2): four fixed seeds, 150 acts
    each, every act and effect by stable key, the golden recorded on code identical to `e57afeaf`.
- **B0.7 The performance instruments (PB-1, PB-2, PB-7, C-8, A-10, A-17, D-28, A-31).** The pin
  comparator, its baseline the maximum over 10 runs on `e57afeaf` with the count recorded, the window
  pins with byte figures; new pins for a beat with Settings open and for `Update` with the map open;
  the three race-skipped pins under `make alloc-budget`, with a gate test that refuses one left
  outside; the tone-latency benchmark's 185 allocations as a pin; benchstat pinned to a version; the
  escape probe by function and type, the model-size probe ("not larger"), the inlining count per
  callee (A-26). Startup and binary size recorded.
- **B0.8 The order of things that must not move.** `windowLayers()`'s order (`app/maplayers.go:windowLayers`,
  PB-4); a test-held list of the post-update steps, parsed from `Dashboard.update`
  (`modes/tty/dashboard.go:update`) as the listed names in order, each exactly once, so 2a's added
  `Env` build does not edit it; seen failing on a swapped pair; a behaviour test that a non-map
  message still makes `armPropOptions` re-ask (design MS-5); the pinned set of message types across the
  tree (R8, RR F7).
- **B0.9 The import rules (TG-1, TG-14, DQ-6, D-15, A-24).** `scripts/lint-imports.sh` holds:
  `modes`, `platform` and `domains` never import `app`; `platform` never imports `modes`; `deck` and
  `mapdata` never import each other; `ttytest` and `apptest` imported by `_test.go` files only. Each
  rule has a planted specimen in its `--self-test`. No type alias that points into the split trees
  survives a batch (GP-11); `SevereTab`, `MapView` and `TickerCategory` exist today and are listed.
- **B0.10 The citation lint keyed on the package root (TG-4, RC-5)**, and "`git log` on the file"
  read as `git log --follow` (`docs/where-things-happen.md:147-149`, `06_docs/ids.md`).
  `cmd/watchpost/extending_symbols_test.go:indexModule` matches by path, not base name (TG-10).
- **B0.11 The shared lock (D-16, A-19, A-32).** `TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage`,
  floor 25 sections, reading the planned file-to-package map (an unmapped file fails; the map is
  deleted once B7 lands): any call under the lock that crosses a planned package fails, and a call
  through a func value passes only for the four declarations recorded at B0. Seen failing on a planted
  fixture under `app/testdata/sharedlock/` (on B0's golden list).
  `TestCommitAndStopAllFinishUnderTheSharedLock`, 200 iterations, each step bound to 2 s of the last,
  scaled under `-race` (A-35), seen failing under vetted mutants `mDL1` and `mDL2` (design AP-3.5, AP-3.6).
- **B0.12 The surfaces held, and the graph tool (A-12, A-29, D-30).** The deck's exported surface,
  tiers 2 and 3 against `app` (design AP-1.5); the exported-name lists of the `modes/tty` and `app`
  roots, which grow only by the new packages' entry points (M5, P2-4); the three pre-existing exports
  of D-26, listed exactly. The graph tool, committed as its own module under `tools/`, counts reach
  across packages as within one, through `Env` and the hooks, and classifies each straddling test
  under D-14 and D-22 (it replaces the designs' scratch scripts). `make graph` is a required gate in
  all three lists, its module named in vet, lint, test, vuln and the licence script (D-30). Before
  M3's targets bind it reproduces on `e57afeaf` every "today" figure (69, 72, 93, 102, 752 in 68, 13)
  and MS-7's 288 and 120; a difference sends the targets back to the HUM LEAD. Its `e57afeaf` outputs
  are committed. Seen failing on a planted pass-through.
- **B0.13 The release record and the floor (A-12, A-29, D-12).** `releaseRecord` points at 0.19.6
  (`cmd/watchpost/record_test.go:releaseRecord`), and each record test gets its 0.19.6 form: the brief stands
  in for `requirements.md`, the FR-7 test stays on 0.19.0's folder, and the plan's and the
  architecture's line citations become symbol citations. `govulncheck` runs at the `go` floor in CI and
  the release job, seen failing at 1.26.10; then the floor is raised to go 1.26.11, with
  `README.md:587` restated and a CHANGELOG entry.
- **B0.14 The testing infrastructure (D-37 to D-39).** The mutant check builds each mutant in place
  through an overlay, its judge held to every verdict on a planted module. `make verify` is the commit
  gate, its mutant check scoped by `inScope` to what the change reaches; `make verify-full` is the full
  gate, CI's and the release job's tier. Before B9, the full sweep becomes a dispatched GitHub workflow
  in 8 Linux shards, a macOS job re-judging each Linux survivor, merged into one committed record.

## B1: RM's slices 1 and 3, the doc errors, and M4's midpoint

- **B1.1 F-158's owed test** (design AP-7.1): `TestACutBackUnderAHazardReadKeepsTheBedDown`, written
  as `app/cutback_hazard_test.go`, its reach deck-only by the graph tool; seen failing under the new
  vetted mutant `mF158` at `app/executors.go:run`. Its first assertion is a precondition that fails
  with "no longer exercises F-158" when 0.19.7's D-2 changes the step.
- **B1.2 The vocabulary** (RM slice 3; JN-5): the misleading names RM-8 and JN-5 found, as A-rows; the
  `doc.go` and `docs/` rows.
- **B1.3 RC-7's six disagreements** (package map §9 row 2): one window where the code has a stack
  (`docs/where-things-happen.md:15`, `:95`, `:128`; `docs/extending.md:291-294`; the comments at
  `modes/tty/dashboard.go:Dashboard`, `:open`).
- **B1.4 M4's before, its doc findings** (`../08-reports/m4-before.md`): Walkthrough 6's golden for a
  map row (`docs/extending.md:190`) and its "Search for" sentences (`:173-174`);
  `docs/where-things-happen.md:67`'s time row, naming `cardPulledShort` and `spokenWhen` as the places
  that choose a date; `UIPrefs.MapPropRefresh`'s comment (`modes/tty/dashboard.go:MapPropRefresh`);
  `modes/tty/console/broadcaster_manifest.go:cardPulledShort`'s cell count; a CHANGELOG `[Unreleased]` heading, and a
  walkthrough line that every user-visible change gets an entry.
- **B1.5 M4's midpoint (D-29).** A fresh blind reviewer traces the four tasks on the unchanged layout
  with B1's docs, on a copy without this release's folder; the grade is filed beside the before.
- **B1.6 D-50's ID families** (A-47): `tools/ids` indexes the UAT rounds' findings, the plans' tasks,
  PD-, S-, SAM-D- and 0.9.0's S-F, each held by `TestEveryIDOfD50sFamiliesTheCodeCitesIsIndexed`, and
  AP-CITE-01 asks a new citation of any of them to name its release.

## B2: the kit and the test support (`modes/tty`)

- **Before:** the console's nine non-drawn references (design CK-5's class X less D-22's drawn
  values: `cardTitle`/`detailBody`, `cardTone`, `poolRows`/`locIndex`, `fireHot`, `fireCount`,
  `.ascii`, `staleAfter`) come to the HUM LEAD singly, with their killers (D-22, A-33): ruled D-52 to
  D-58, two splits by package (D-52, D-53), two drawn values (D-54, D-57), `frame.WeatherRow` for the
  two kit reads (D-55, D-56) and CK-4's kit-level masthead test (D-58).
- **Prep:** TS-4's closure cut into the kit's files in place (design CK-1: `masthead.go`, `row.go`,
  `cursor.go`, `tick.go`, `memo.go`, `text.go`); 25 names and 5 methods exported; `MemoStats` with its
  embedded mutex, `Hit` and `Miss` documented as running with the lock held (A-21); `NoTransmitter`
  into the kit (A-31). The engine renamed only (`PerturbEach`); the pure test helpers gathered into
  `ttytest` beside B0's `Main`. `raceEnabled` stays a `_test.go` pair per package; the golden helper
  takes its update choice as a parameter (A-33).
- **Move:** `modes/tty/internal/frame`, and the engine into `modes/tty/internal/ttytest`.
- **P10:** D-15's P10-09 row and D-23's two P10-01 rows written for ratification; the kit's and the test
  support's density rows (D-31).
- **Tests:** the masthead guard split by surface, with the named inputs in a kit-level test (design
  CK-4); the two kit-internal references (`fireHot`, `fireCount`) as ruled before the batch;
  `kit-CK.tsv` and `tests-CK-refs.tsv` are the batch's lists, regenerated by B0.12's tool.
- **Mutants:** 12 kit anchors re-point by path; 12 anchors that name a kit symbol re-point under A-25.
- **Record:** 0.19.6's build log and its as-built map are created; `TestTheAsBuiltMapIsKeptInStep`
  learns the feature (`app/maps_test.go:TestTheAsBuiltMapIsKeptInStep`).

## B3: the console (`modes/tty/console`)

- **Prep:** `console.New(Opening)`, `SetMirror(Mirror)`, `SetListenerMuted`, `WithSnapshot`,
  `WithRecent`, `Scoped`, the 15 Router calls exported and the 5 getters added (design CK-2, A-21); the
  card window cut into `card_window.go`; D-21's `Opening.Now`, with the router trace's console-clock
  line deleted and its golden unchanged.
- **Tests (D-14, D-22):** `tests-CK.tsv`: 73 setup-only references, 23 same-value, 12 moved, the 11
  drawn values under D-22 and the remaining singles as ruled before B2, each in the changed-line table
  with its killers. A console golden with a populated DATA PULL line, on the batch's golden list
  (A-33).
- **Move:** the twelve `broadcaster*.go` files and their tests; `app`'s 11 non-test sites re-qualified
  for the six messages (C1).
- **Mutants:** 77 re-point by path; 14 anchors per A-25.
- **P10:** `modes/tty/broadcaster_rail.go`'s row re-keys; the console's density row (D-31). The seven
  `Router` rows and `Dashboard.View`'s are re-proved per surface: the console's `Update` and `View` are
  in another package, the Dashboard's are unchanged, and neither surface holds a Router field.
- **Docs:** `0.16.0-broadcaster-ui/00-REQUIRED-READING.md`'s `broadcaster_rail.go` row; the vocabulary
  row's `console.New`, in the new `modes/tty/console/broadcaster.go`.

## B4: the map's 2a, in place, four gated commits

1. **The guards' tests.** The 11 split sites' tests and the Overlays menu's guards (its hand-listed
   rows; `switchRow`'s fall-through, a guard only) land here, green on today's code, with no
   production change (A-31, A-33, D-76).
2. **The hooks.** `Config.Map *Hooks`, exported under its final name, held once on `Dashboard` for
   this commit; `app/dashboard.go:mapConfig`'s hook assignments become one keyed literal. The Router's
   `Timed` is renamed so a stale `cfg.Timed` write fails to compile; tests change hooks only by
   copy-on-write, held by a static guard (A-31, D-77).
3. **The model, its fields, its methods and the seam's values.** `mapModel` declared and embedded in
   `Dashboard`, the map's fields moved onto it and the hooks pointer into it, in the same commit as the
   methods, so no commit compiles a promoted field one selector deeper with its methods still on
   `Dashboard` (D-76). `Env` (plain values, no derived sizes, no hook), `Want`, `Step` and `Window`
   land here with their callers, their 2a names put to the HUM LEAD first (D-77).
   The 387 methods re-homed, in file groups; S8 to S10 cut at the save, its batch
   position carried in `Want`; `mapEnv`'s and `applyMapWant`'s switches held by a round-trip test and
   vetted mutants. Assertions that call a method gaining a parameter change under D-14, rows of the
   changed-line table (A-31). The seam's types unexported and `map`-prefixed (D-79), the model's
   receiver `mm` (D-80). `mapModel` a named field of `Dashboard` from this commit, 2b's first step
   brought forward, so the compiler emits no wrapper for the map's methods on the `Dashboard`; the root's
   memo guard calls its engine twice here, as B8 had it (D-81). `gcprobe` holds what no rename can
   carry by a reviewed list, `06_docs/perf/excused-B4.tsv`, and no method falls over the inline budget
   (D-82).
4. **The audit and the re-measure.** The 41 hook writes made after construction audited for the shared
   pointer; 2b's class C4 re-measured by B0.12's tool and classified against D-22, the residual put to
   the HUM LEAD singly before B5 (A-33).

- **Mutants:** 146 anchors re-pointed in the prep commits, reviewed as A-11's old-and-new table, then
  the scoped sweep.
- **Gate:** benchstat (D-28); the sizes probe records `Dashboard`'s and the model's bytes.

## B5: the kit and the test helpers (`app`)

- **Prep:** the seven shared helpers gathered and exported (`EveryTick`, `LockedCopy`,
  `SavePreference`, `UserCacheSubdir`, `CacheDir`, `UserAgent`, `RequestStats`); `app`'s pure test
  helpers gathered into `apptest` beside B0's helper; the local `run` in `startSchedule`
  (`app/schedule.go:startSchedule`) renamed; `app/every.go`'s P10-02 row re-keyed by symbol.
- **Move:** `app/internal/run` with its tests; `app/schedule_test.go`'s callers re-qualified.
- **P10:** the row re-keyed by path; `run`'s density row (D-31; the pilot's hand count puts it near
  0.5).

## B6: the deck (`app/internal/deck`), whole

- **Before:** the root wiring tests (A-32): a root test drives `SetTransmitter` on a `livePipelines`
  holding the deck and reads the re-fence through the deck's held surface, and one does the same for
  `Attach`, each green on today's code with a vetted mutant on the call.
- **Prep** (design AP-1, AP-2, A-23, A-32): `deck.Config`, `deck.New`, the ten methods; `RunDashboard`'s
  order kept step for step; `stationConfig` split; the station seam (`Watch`, `Pool`, `Station`); the
  deck-side fields and 35 methods onto `Deck` (its radio player field named `player`); `CastBlock`
  behind the root's `RenderCast`; a root test that the deck's `mu` is `&lp.mu`.
- **D-24:** the deck's own `TestMain` applies `WATCHPOST_TEST_GOOS`; `make test-platforms` over
  `./app/...` is in this batch's gate.
- **Tests:** `tests-AP.tsv`'s deck rows; the deck's exported surface test gains tier 1.
- **Move:** the 29 deck files (A-23: `saychildren.go` and `hangup.go` stay at the root) and their
  tests; the two traces' goldens renamed with them.
- **Mutants:** 116 re-point (design AP-10's statement-level count; `mutants-AP.tsv` is file-level);
  11 anchors change text.
- **P10:** `app/cast.go`'s row re-keys to the deck (D-24); the deck's density row and its `New` row
  (D-31, D-32).
- **Docs:** `0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md`; `follow-ups.md`'s
  F-158 paths.

## B7: the map data (`app/internal/mapdata`)

- **Before:** the `FillStats` root test and its mutant, green on today's code (A-32).
- **Prep:** `mapdata.Config`, `mapdata.New`, the eleven methods (design AP-4); the maps-side fields and
  40 methods onto `Feeds`; `lazymemo.go` and `timing.go` with them; D-22's one test (m497, m498)
  served by the `diagSources` interface member; a root test that the map data's `mu` is `&lp.mu`.
- **Move:** the 22 files and their tests; the lock check's planned file map deleted.
- **P10:** the map data's density row and its `New` row (D-31, D-32).
- **Docs:** `map-ready-geometry/03-architecture-design/as-built.md`; `observer-maps`' as-built map.

## B8: the map's 2b and the controls, three commits

1. **The straddling tests re-wired in place** onto the API that survives (D-22, as ruled before B5):
   B4.4's 135 by `06_docs/graph/ms7-c4-B4.tsv`'s classes, and the 9 unserved split at the package line,
   the map's half in `mapview` with the instruments it reads, a static guard that the root reaches the
   map only through MS-6's drawing methods (D-83).
   The root's memo guard already calls the unchanged engine twice, on `Dashboard` with the model
   excused and then on the model, each perturbed model put back into a copy (A-31; B4.3, D-81); D-23's
   rows hold.
2. **The straddle count re-measured**, and the result reported.
3. **The move:** `modes/tty/mapview` and `modes/tty/internal/controls` (D-25); `app`'s 14 files
   re-qualified from `tty.` to `mapview.`.

- **Mutants:** 289 re-point by path; 7 anchors change text in the first commit.
- **Gate:** benchstat (D-28).
- **P10:** the two packages' density rows and `mapview.New`'s row if it calls another `New` (D-31,
  D-32).
- **Docs:** the map package's `doc.go` names its four under-the-map rows with one word for the legend
  (JN-3), and the Overlays menu's rows; Walkthrough 6 covers a map Setting that is also an Overlays
  row; a golden that draws a legend row, on the batch's golden list (A-33).

## B9: the closing sweep and the REVIEW inputs

- The full sweep against A-3's record: every A-3 mutant keeps its class or is re-ruled; every new
  mutant is caught; none has no evidence (M7, B-9).
- M1 to M6's figures against the baseline; `README.md:597` and `Makefile:35-41` restated with the
  measured slowest package.
- Hosted full CI green on both systems, named as full and not quick (M7; D-128, D-129).
- M4's blind trace on the new layout, against a fresh key built by leave-one-out before REVIEW, its
  delta read from B1.5's midpoint (D-29).

---

## For ratification at the PLAN gate

### M3's targets (D-27)

| Figure | Today | Target | Blind spot |
|---|---|---|---|
| Hub fan-in: files using `Dashboard` | 69 | at most 39 | counted by file, so a file that keeps one use still counts |
| Settings' reach into the model (JN's scope) | 102 | at most 83 | Settings reaches the map's rows through about ten `mapview` entry points instead |
| The map's reach into the model (JN's scope) | 72 | at most 23 | holds under D-25 |
| The same, by type | 93 | at most 40 | the same |
| `Dashboard`'s methods and files | 752 in 68 | at most 374 in 38 | in 2a the model's methods are still promoted into its method set; 2b ends that |
| The console's reach into the Observer | 13 | 0 | — |
| W8, a console card line | 1 package | 1 package | — |
| The legend-key task | at least 5 files, 1 package | at most 3 files, 1 package | holds only if 2b's `doc.go` carries the row list |
| W6 for a map Setting (M4's T1) | 3 packages | 4 packages, recorded | the one rise; D-25 is its reason |

Every "today" figure is reproduced by B0.12's committed tool on `e57afeaf` before the targets bind.

### The P10 re-key table (D-10, D-31, D-32)

Keyed by file and symbol. Re-keys are written to the local ledger and the mirror regenerated; a
symbol re-key goes in the prep commit, a path re-key in the move commit.

| Ledger row (file, symbol) | After | Batch | Proof |
|---|---|---|---|
| `app/dump_windows.go`, `app/dump_unix.go` (`package`) | unchanged | — | — |
| `modes/tty/router.go`: `Router.Init`, `.Update`, `.update`, `.scopedMessage`, `.routeKey`, `.keyAction`, `.throughToObserver` (P10-01) | unchanged | B3, B8 | re-proved per surface: the console's methods are in another package, the Dashboard's are unchanged, and neither surface holds a Router field |
| `modes/tty/view.go`, `Dashboard.View` (P10-01) | unchanged | B4, B8 | re-proved after the model's methods leave; the map's methods share no bare name with a Dashboard method they are called from (D-32) |
| `app/every.go`, `everyTick` (P10-02) | `app/internal/run/every.go`, `EveryTick` | B5 | the reason unchanged |
| `app`, `package` (density) | unchanged for the root; each child its own row | B5 to B7 | D-31 |
| `modes/tty`, `package` (density) | unchanged for the root; each child its own row | B2 to B8 | D-31 |
| `app/cast.go`, `package` (`runtimeGOOS`) | `app/internal/deck/cast.go` | B6 | D-24 |
| `modes/tty/broadcaster_rail.go`, `package` | `modes/tty/console/broadcaster_rail.go` | B3 | the reason unchanged |
| new: `modes/tty/internal/ttytest/perturb.go`, `package` (P10-09: `unsafe`, `reflect`) | — | B2 | D-15 |
| new: `walkAt`, `bump` (P10-01) | — | B2 | D-23 |
| new: each child package's density row; each `New` that calls another `New` (P10-01) | — | the batch that creates it | D-31, D-32 |

A cross-package move that git pairs is judged by changed line; a byte-identical body is not judged
again, and a function whose signature the prep changed is. A same-package rename with no edit, a
split, and an unpaired file are judged whole. Every row is re-keyed in either case (the pilot).

## M4's four tasks

Fixed at PLAN by the key builder (A-28), and traced blind on `e57afeaf` as M4's before (B-6): 0 of 4,
in `../08-reports/m4-before.md`, the keys in `../08-reports/m4-key/`. B1.5 traces the midpoint (D-29).
Each trace runs on a copy with this release's folder deleted, and `06_docs/ids.md` is checked to hold no
path into `m4-key/` (0.19.5 batch 43).

## What this plan does not do

D-2 proper and everything A-8 lists (0.19.7, D-11); any Charm or Bubble Tea upgrade (C-9); any
behaviour change (R1); any package outside `modes/tty` and `app` (scope).
