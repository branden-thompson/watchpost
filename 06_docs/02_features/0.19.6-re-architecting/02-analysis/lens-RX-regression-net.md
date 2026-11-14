---
title: "0.19.6 DISCOVER lens RX: the regression net under a split"
date: 2026-11-14
phase: DISCOVER
lens: RX
status: "FILED — findings checked against e57afeaf"
---

# Lens RX: the regression net under a split

## Verdict

The net can stay green through the split. Its weakness is somewhere else: most of it is not a set of
per-cluster unit tests. Most of it is **whole-model tests**. 705 of the 1,097 top-level tests in
`modes/tty` build the whole `Dashboard` and drive keys through it, and nearly every map and Propagation
mutant is caught by one of them. So for the clusters that carry most of the mutants, a test cannot
simply "move with its code". It stays at the top of the import graph and reaches down into
internals that a split makes private. Three things decide whether principle 4 can be proved:
1. Where the seams fall relative to those **straddling** tests (RX-3).
2. Whether six directory-bound checks are fixed before anything moves. Today they would go quiet,
   not red (RX-9).
3. Whether each batch's evidence is mechanical: byte-identical goldens and router trace, the same
   test inventory, declarations conserved across the packages, and a targeted verdict sweep compared
   mutant by mutant (RX-10 to RX-13).

The brief's mutant figure is also wrong. **692** mutants anchor in these two packages, not 255
(RX-1).

## What I checked, and how

- **The graph.** The shared graph (`test-reach.tsv`, `symbols.tsv`), grouped by a coarse cluster
  map of my own, by file name:
  - tty: `map-prop` (`map_prop*`), `map` (other `map_*`), `broadcaster` (`broadcaster*`,
    `poolnote`, `relayfault`, `request`), `setup` (`setup*`, `textfield`), `windows` (`alerts`,
    `detail*`, `help_about`, `columns`, `credit_row`, `debug`, `status*`, `severe`,
    `modal_location`, `timing`), `shell` (everything else).
  - app: `deck` (radio, ticker, shared, cast, readers, relays, air scope), `arbiter` (`arbiter`,
    `mastercontrol`), `executors` (`executors`, `pump`, `schedule`), `severe`, `maps` (`map*`,
    `framelog`), `record` (history, stats, dump, inject, timing), `setup` (`setup`, `themes`,
    `voices`, `credits`, `release`), `root` (the rest).
  - TS and AS own the real seams. The scripts take any file-to-cluster map and can be re-run on
    theirs:
    `<scratchpad>/q196/rx/`
    (`analyse.py`, `straddle.py`, `strad_detail.py`, `wholemodel.py`).
- **Mutant targets.** Every `pathlib.Path(...)` literal in `06_docs/mutants/m*.py`, in both quote
  styles. Every anchor, extracted by running each mutant with `write_text` neutered, which is how
  `scripts/quality/mutant-anchors.sh` does it. The tree was checked clean of my changes afterwards
  (`git status`).
- **Killers.** Each mutant's last killer, from `06_docs/mutants/verdicts/since-v0.19.5.log`,
  resolved to its defining file with the same name lookup the sweep uses
  (`scripts/quality/mutant-verdicts.sh:149-152`).
- **Whole-model tests.** Counted by a fixed point over test helpers: a test is whole-model if it
  calls `NewDashboard`/`NewRouter` (tty), or `newStation`/`newExecutors`/`startSchedule`/`newArbiter`
  (app), directly or through a helper, to any depth.
- **Path-bound checks.** Every test that reads source or walks a directory (grep for
  `declset.Files`, `go/parser`, `filepath.Glob`/`Walk`, `os.ReadDir`, hard-coded `app`/`modes/tty`
  paths). Each one was read for whether a move makes it fail or makes it go quiet.
- **Other surfaces.** Goldens, `TestMain`s, Makefile targets that name a package, and the sweep and
  harness scripts.
- **Not run.** No tests and no sweep: the brief forbids them. Each figure below comes from the
  graph, from grep, or from the committed record.

## Findings

### RX-1: 692 mutants anchor in `modes/tty` and `app`, not 255

- **Claim.** D-6 and the brief count 255 (145 tty, 110 app). The corpus holds 539 on `modes/tty` and
  153 on `app`.
- **Evidence.**
  - `grep -l '"modes/tty/'` over `06_docs/mutants/m*.py` finds 145 files. `grep -l "'modes/tty/"`
    finds 394 more. `app` is 110 double-quoted plus 43 single-quoted.
  - The sweep's own timings agree: `dist/mutant-verdicts.timings` holds n=539 for `modes/tty` and
    n=153 for `app`.
  - Every mutant names exactly one target. The tools read both quote styles
    (`06_docs/mutants/mutants_test.go:47`, `scripts/quality/mutant-verdicts.sh:137`), so only the
    figure is wrong.
  - Per cluster:
    - tty: map-prop 203, shell 96, broadcaster 92, map 86, setup 35, windows 27.
    - app: deck 46, executors 35, arbiter 20, root 19, maps 11, severe 10, record 6, setup 6.
- **Consequence.** The re-point and re-vet work (principle 4) is 2.7 times what was put to the HUM
  LEAD. Any PLAN sized on 255 is undersized.
- **Recommendation.** Correct the figure in the record as an A-row (RC). Size the per-batch sweeps
  on 692 (RX-11).
- **Confidence.** High.

### RX-2: Most of the tty net is whole-model tests, and they cannot live below the model

- **Claim.** Most tests named after a cluster's files do not test that cluster alone. They test it
  through the whole model.
- **Evidence.**
  - tty: 705 of 1,097 top-level tests (156 of 213 files) build the `Dashboard` through
    `NewDashboard`/`NewRouter`, via 67 helpers.
  - The central helpers:
    - `mapDash` calls `NewDashboard` (`modes/tty/map_window_test.go:38-46`).
    - `dash` calls it too (`modes/tty/dashboard_test.go:40-42`).
    - `goldenDash` goes through `benchDash` (`modes/tty/golden_test.go:24-26`,
      `bench_test.go:84-89`).
    - `openMap` goes through `mapDash` (`map_uat1_test.go:24-29`).
  - app: 101 of 689 tests build the station (`newStation`, `app/station_test.go:51-76`), and 139
    build the arbiter (`testArbiter`, `app/arbiter_test.go:424-428`).
  - The cause sits in the code. 784 methods are declared on `Dashboard`/`Router` across 39 files.
    382 of them are in `map_*` files (116 in `map_prop*`), 107 in `setup*`, 6 in `broadcaster*`.
    `map_pane.go` alone holds 51, `map_temp.go` 42, `map_boxes.go` 40.
- **Consequence.**
  - Go's in-package tests cannot import a package that imports them, so a test that builds the
    `Dashboard` can only live where the `Dashboard` lives, or above it.
  - Methods on `Dashboard` cannot leave its package. A map move is therefore a re-typing first
    (TS), and the tests that pin map behaviour stay at the top whatever is moved.
- **Recommendation.** Plan the net as two layers:
  - The top package keeps the whole-model tests and their assertions unchanged.
  - Each new package gets only the tests that reach nothing but its own code (RX-3).
  - Do not "move tests with their code" by file name.
- **Confidence.** High. The counts come from a regex fixed point over test bodies. They may be off by
  a few. The shape is not in doubt.

### RX-3: The straddle count per candidate move is the cost of the net, and it marks the real seams

- **Claim.** Take any candidate move and sort each test file of the package by the unexported
  symbols it reaches:
  - **moves intact:** it reaches only the moved code;
  - **stays intact:** it reaches only the rest;
  - **straddles:** it reaches both.

  A straddler compiles on neither side without an export, a wrapper or a rewrite of what it
  observes.
- **Evidence.** From `test-reach.tsv`, my clusters (killer = the mutant's last killer):

  | Candidate move | Tests move | Stay | Straddle | Unexported symbols straddlers need | Mutants re-pointed | Killer moves / stays / straddles |
  |---|---|---|---|---|---|---|
  | tty map-prop | 0 | 174 | 26 | 149 | 203 | 0 / 2 / 201 |
  | tty map | 3 | 116 | 81 | 336 | 86 | 2 / 3 / 81 |
  | tty map + map-prop | 3 | 116 | 81 | 485 | 289 | 2 / 4 / 283 |
  | tty broadcaster | 26 | 139 | 35 | 108 | 92 | 46 / 4 / 42 |
  | tty setup | 1 | 152 | 47 | 199 | 35 | 0 / 4 / 29 |
  | tty windows | 1 | 155 | 44 | 93 | 27 | 2 / 6 / 19 |
  | app deck | 17 | 81 | 38 | 187 | 46 | 14 / 0 / 32 |
  | app deck + arbiter + executors | 37 | 77 | 22 | 211 | 101 | 64 / 0 / 35 |
  | app maps | 10 | 88 | 38 | 188 | 11 | 2 / 0 / 9 |
  | app arbiter | 2 | 108 | 26 | 55 | 20 | 2 / 0 / 18 |
  | app severe | 1 | 127 | 8 | 34 | 10 | 0 / 0 / 9 |

- **What the straddlers need.** For map-prop, they reach the `Dashboard`'s own state:
  `dashboard.go:.mapPane` (23 files), `.cfg` (13), `Dashboard.describeLinesAll` (11),
  `Dashboard.opts` (10), `.modal` (9).
- **Hubs** (test files reaching four to six clusters):
  - tty: `map_prop_chart_test.go`, `erase_test.go`, `memo_completeness_test.go`,
    `map_station_test.go`, `map_prop_honesty_test.go`.
  - app: `inject_seam_test.go`, `inject_scenarios_test.go`, `severe_read_test.go` (26 arbiter
    symbols), `schedule_test.go`, `pool_test.go`.
- **Consequence.**
  - A cluster whose tests all straddle is not yet a seam. Moving it either widens the API (against
    principle 2) or rewrites the very tests that pin its mutants (against principle 4).
  - The two moves the net supports best are the tty broadcaster (its own type: `NewBroadcaster`,
    `modes/tty/broadcaster.go:278`) and the app deck kept whole with its arbiter and executors.
    That matches the settled "the shared deck is not split".
- **Recommendation.**
  - Use the straddle count as a seam test in the package map (D-7).
  - Rank candidates by low straddle and high "killer moves".
  - Wherever a seam with straddlers is taken, a prep commit first moves each straddler's subject
    onto an API the production code also uses, inside the old package, with every affected mutant
    re-vetted (RX-11). Only then does the move commit follow.
- **Confidence.** High for the method. Medium for the exact figures: my cluster map is coarse, and the
  graph is blind to build-tagged tests (RX-9).

### RX-4: How tests move, option by option

- **Claim.** Of the four options in the brief, only in-package moves keep the pins exactly as they
  are. The others either cannot reach a straddler or turn it into a new test.
- **Evidence.**
  - **(a) In-package, moved with the code.** The test, its assertions and its mutants' killers are
    unchanged. This works only for the "move intact" column of RX-3.
  - **(b) External `_test` packages.**
    - Every one of the 336 test files that reaches anything reaches unexported code
      (app 136 of 136, tty 200 of 200).
    - The 27 that reach none are mostly source scanners, the path-bound checks of RX-9.
    - Converting a test means rewriting what it observes. It becomes a new test whose pins must be
      re-proved mutant by mutant.
    - Today only `modes/tty/one_band_writer_test.go` is external.
  - **(c) `export_test.go` bridges.** These are compiled into their own package's test binary only.
    A test in `modes/tty` cannot see a bridge in a new `modes/tty/<leaf>`. Bridges help only a leaf's
    own external tests, so they solve none of the straddlers. The tree has none today.
  - **(d) A test-only helper package (`…/internal/…test`).**
    - It sees only exported API.
    - If it builds a `Dashboard`, tty's in-package tests cannot import it (import cycle in test).
    - So it can only hold **pure** helpers: those with no dependency on the model. `stripANSITest`
      (`modes/tty/dashboard_test.go:195`) is one; key-message builders are others.
  - **(e) Not in the brief.** Exported test hooks in production code have a precedent:
    `SetColorEnabledForTest` (`platform/render/sgr.go:36`). This widens the API on purpose.
- **Consequence.** (a) keeps principle 4 provable. (b) and (e) spend principle 2 or principle 4.
  (c) does not reach the problem. (d) is good practice for pure helpers only.
- **Recommendation.**
  - Use (a) for intact movers.
  - Keep straddlers in the top package with their assertions unchanged.
  - Put pure shared helpers in one internal test-support package (GP to confirm the practice).
  - Treat (b) and (e) as a HUM LEAD question (Open questions, 2).
- **Confidence.** High. These are Go language rules.

### RX-5: The shared test fixtures are concentrated, and `_test.go` helpers cannot cross a package

- **Claim.** A split forces a decision about 99 test helpers that are used by more than one cluster.
- **Evidence.**
  - 287 symbols declared in `_test.go` files are used by other test files. 99 of them are used
    from more than one cluster.
  - The heaviest (files that use them, clusters):

    | Helper | Files | Clusters |
    |---|---|---|
    | `stripANSITest` | 123 | 7 |
    | `mapDash` | 43 | 5 |
    | `pressKey` | 31 | 5 |
    | `dash` | 28 | 5 |
    | `feedAndSettle` | 25 | 3 |
    | `snap` | 23 | 6 |
    | `boxFeed` | 22 | 4 |
    | `testArbiter` (app) | 21 | 5 |
    | `openMap` | 20 | 4 |
    | `benchDash` | 18 | 4 |
    | `goldenDash` | 17 | 7 |
    | `scriptVoice` (app) | 16 | 4 |
    | `newStation` (app) | 12 | — |
- **Consequence.** Copying a helper into each new package gives copies that drift. A changed fixture
  in one copy changes what that package's tests pin (principles 2 and 4).
- **Recommendation.**
  - Pure helpers go to one internal test-support package, in the first batch that needs them.
  - Model-building helpers stay with the whole-model tests at the top.
  - A copied helper is not allowed.
- **Confidence.** High.

### RX-6: Each package's `TestMain` holds an invariant that a new package would silently lack

- **Claim.** Both packages set test-wide invariants in a `TestMain`, and the Makefile names `./app`
  for one of them.
- **Evidence.**
  - `modes/tty/main_test.go:20` pins `time.Local = time.UTC`. The goldens' stamps depend on it, and
    `TestNoTestSetsTheZoneMidRun` holds the rule.
  - `app/platform_sim_test.go:26-49`:
    - applies `WATCHPOST_TEST_GOOS` through `setRuntimeGOOS` (`app/cast.go:52`);
    - points `XDG_CONFIG_HOME` and `XDG_STATE_HOME` at a temporary directory, so no test writes
      the developer's config or state.
  - `make test-platforms` runs `./app` only (`Makefile:214-215`).
- **Consequence.** Code moved out of `app` without that `TestMain` has tests that can write real
  config and state files, and it drops out of the platform simulation with no failure anywhere.
  Golden tests moved out of `modes/tty` without the UTC pin pass or fail by machine zone.
- **Recommendation.**
  - Every new package under `app/` and `modes/tty/` carries its parent's `TestMain`, held by one
    test that walks those trees.
  - `test-platforms` names `./app/...` (TG).
- **Confidence.** High.

### RX-7: The goldens are keyed by name and stay byte-identical if they stay put

- **Claim.** A move changes no golden, as long as the test that owns the golden does not move.
- **Evidence.**
  - `modes/tty/testdata` holds 63 `.golden` files and `relay-fault.mock`.
  - `checkGolden` keys each one by an explicit name under the relative `testdata`
    (`modes/tty/golden_test.go:33-35`). `severe_test.go:345` does the same. Nothing keys a golden
    by `t.Name()`, so renaming a test touches no golden.
  - The golden tests build the whole `Dashboard` (RX-2), so they stay at the top. The goldens stay
    where they are.
  - Path couplings:
    - `journey_labels_test.go:125-127` reads only `modes/tty/testdata`. A golden moved elsewhere
      drops out of the journey-label corpus.
    - `cmd/watchpost/journey_test.go:167` and `measure_test.go:221` read
      `app/testdata/maps/m1/...` by path.
    - `app/cadence_test.go:46` and `app/testdata/declset.txt` are relative to `app`.
- **Consequence.** Goldens are the cheapest byte-identical evidence there is, provided no move batch
  re-captures them.
- **Recommendation.**
  - No `-update-golden`, `-update-declset` or `-update-cadences` run in a move commit. Any re-capture
    is a behaviour change with its own ruling.
  - Evidence per batch: `git diff -M` over `*/testdata/*` shows renames at 100 % similarity, or
    nothing at all.
  - `app/testdata/maps/m1` does not move in 0.19.6.
- **Confidence.** High.

### RX-8: The router trace already exists as an instrument for "behaviour byte-identical"; app has none

- **Claim.** One existing test is principle 4 in the form a re-architecture needs.
- **Evidence.**
  - `TestRouterUpdateIsTheSameUnderRandomMessages` (`modes/tty/router_trace_test.go:47`; golden at
    `:66`) runs random message sequences on fixed seeds through the `Router`. After each message it
    records the surface, whether a command came back, and a digest of the frame.
  - It was recorded on the pre-change code for 0.19.5 Q3.1 (`:42`).
  - It is in the top package and builds through `NewRouter`, so no split moves it.
  - Nothing like it exists for app's deck, Director and executors.
  - The PTY journey `TestJourneyMapOnTheRealBinary` (`cmd/watchpost/journey_test.go:72`) runs
    offline in every non-`-short` `go test ./...` and does not depend on layout, but it covers only
    the map path.
  - `make journey` uses live feeds (`Makefile:346-356`), which the quota rule keeps out of per-batch
    evidence.
- **Consequence.** In tty, "the behaviour did not change" can be proved by one byte-identical file
  per batch. In app, today, it can only be argued test by test.
- **Recommendation.**
  - Before any code moves, widen the trace's message set and seeds to reach the map, Propagation,
    Settings and console keys.
  - Build the app equivalent: the deck and the Director step on fixed seeds, effects and published
    messages traced (RM, AS).
  - Record both on code identical to `e57afeaf`, and hold them byte-identical in every batch.
- **Confidence.** High that the instrument fits. Medium on how far its current message set reaches.

### RX-9: Six checks and four Makefile targets go quiet, not red, when code moves into subdirectories

- **Claim.** Most of the source-reading guards follow INST-2 and fail loudly when their subject
  leaves the directory they read. These do not.
- **Evidence.** The quiet ones:
  - `app/logic_test.go:171` (`TestNoListComparesZips`) reads only `.` and `../modes/tty`, skips
    directories, and has no file floor.
  - `modes/tty/map_vision_test.go:166` (`.Ground(`/`.SetPalette(` rule) globs `*.go` and
    `../../app/*.go`, with no floor.
  - `modes/tty/held_clock_test.go:19` globs only this directory's `*_test.go`, with no floor.
  - `modes/tty/map_window_test.go:226-232` (`TestMapTestsReachNoSocket`) globs `map_*_test.go`,
    and its floor is zero.
  - `modes/tty/map_colour_guard_test.go:91-99` has a floor of 40 against 48 files today. Moving 8
    or fewer goes unnoticed.
  - `modes/tty/one_band_writer_test.go:35` matches the qualifier `pkg.Name == "tty"`. If the
    package that declares `TickerBreakingMsg` changes, qualified constructions become invisible.
    Its owner key `app/mastercontrol.go` (`:23`) does fail loudly on a move, so that part works.
  - `cmd/watchpost/docs_test.go:353` (`TestEveryFileIsInItsPackagesMap`) lists `modes/tty` and
    `app` by hand and reads one directory each. New packages go unmapped.
  - Makefile targets that name a package:
    - `property` runs `-run TestThePrintedMapIsAFreshRender ./modes/tty` (`Makefile:526`). If that
      test moves, `go test` matches nothing and exits 0.
    - `vet-tags` (`:62`), `quality-bench` (`:532`) and `test-platforms` (`:214`) name packages the
      same way.
  - The graph cannot see build-tagged tests: `map_property_full_test.go` is `//go:build property`,
    and `race_test.go`/`norace_test.go` are race-tagged.

  The loud ones, for contrast:
  - `app/maintrack_power_test.go:172` (`found != 1`);
  - `app/maintrack_seam_test.go:111`;
  - `app/router_owner_test.go:60`;
  - `app/reachable_test.go:75`;
  - `platform/singleowner`, which walks the whole module.
- **Consequence.** A rule that has stopped being checked still reads green. That is a principle 4
  regression that no gate reports.
- **Recommendation.** Before any move, harden each check test-first (FULL TDD):
  - Plant a file in a subdirectory and show the check misses it today.
  - Then make the check walk the package tree, with a floor taken from the tree.
  - Make the four Makefile targets name `./app/...` and `./modes/tty/...`, or fail on "no tests to
    run" (TG).
- **Confidence.** High. Each check was read.

### RX-10: Nothing proves that a move kept every test and every declaration

- **Claim.** Today no gate shows that a batch moved tests and declarations without losing any.
- **Evidence.**
  - No gate compares the test inventory before and after.
  - The declaration-set pin exists for `app` (`app/declset_test.go:17`, 1,141 declarations in
    `app/testdata/declset.txt`) and four other packages, but not for `modes/tty`.
  - `platform/declset` works per directory. A declaration moved out of `app` shows up as "removed",
    which is loud, but nothing proves it arrived intact in the new package.
  - 0.19.5 used it for an in-package 1→13 split (`platform/declset/declset.go:1-6`).
- **Consequence.** A test file dropped by a build tag or a package clause, or a declaration renamed
  in a "move", passes every check.
- **Recommendation.** Before any move, as two instruments:
  1. A test-inventory record: top-level test names over both trees, with and without
     `-tags property` and `-race`. A move commit keeps the same multiset of names: a test may change
     directory, never vanish or be renamed.
  2. A union declaration check: the declaration sets of the old package and its new packages
     together equal the baseline, modulo a recorded rename table for the identifiers a move exports.

  Add the missing `modes/tty` declset pin. Whether A-4's graph becomes the check of the new
  boundaries is a PLAN question (TG).
- **Confidence.** High.

### RX-11: Re-pointing and re-vetting at 692 mutants; the targeted sweep is the re-vet

- **Claim.** The re-vet rule cannot be met by hand at this scale. The sweep's own scoped mode meets
  it better.
- **Evidence.**
  - The rule: "Every re-pointed mutant is re-vetted by hand" (0.19.5 A-14,
    `../../0.19.5-quality-pass/02-analysis/rulings.md:107`, from RR-10).
  - Every mutant on a moved file needs its path re-pointed. Its anchor text changes too where the
    move changes code:
    - 311 anchors (264 tty, 47 app) name a receiver's field (`d.`, `m.`, `r.`, `b.`), which a
      re-typing (RX-2) rewrites;
    - 83 hold comment text (62 tty, 21 app), which a comment edit in a move commit would drift.
  - The sweep finds a mutant's last killer by its top-level test name anywhere in the tree
    (`mutant-verdicts.sh:149-152`, `:207-219`), so a moved test with an unchanged name is still
    found. Step 2 then runs only the mutant's own package (`:244`).
  - For the 283 map and map-prop mutants whose killers straddle (RX-3), a leaf mutant whose step 1
    misses is caught only at step 3, against `./...`. The record then reads "[caught only against
    ./...]", and each such mutant costs a whole-tree run (PB).
  - 4 of the 7 kept equivalents are in these packages: `m16` (`app/severe.go`), `mAA3`, `mAA4`
    (`modes/tty/setup*.go`), and `mSC3` (`router.go`).
  - Cost, from the timings: 14.9 s per tty mutant and 21.0 s per app mutant. map-prop's 203 take
    about 13 minutes on 4 workers.
- **Consequence.** Re-vetting 692 by hand is not credible. A re-vet that skips any is RR-10's
  "a pin silently lost".
- **Recommendation.**
  - The re-vet of a batch is `scripts/quality/mutant-verdicts.sh` with `MUTANT_ONLY=<that batch's
    mutants>` and `MUTANT_KILLERS=since-v0.19.5.log`. That run is stronger than a hand vet: the catch
    must repeat, and attribution is checked.
  - Search `06_docs/mutants` for anchors before code moves.
  - A move commit touches no comment.
  - The 4 kept equivalents stay SURVIVED, or each one is re-ruled.
- **Confidence.** High.

### RX-12: Nothing compares a sweep with the baseline, and the baseline is uncommitted

- **Claim.** "Judged against since-v0.19.5.log" has no mechanism and, right now, no committed
  record.
- **Evidence.**
  - No script or test compares two verdict records. The only reader of that path is the promotion
    target (`Makefile:378-417`).
  - `git status` shows `06_docs/mutants/verdicts/since-v0.19.5.log` untracked on
    `feature/re-architecting`.
  - The log names each killer by its top-level name only.
- **Consequence.** The closing judgement would be read by eye across 1,058 lines.
- **Recommendation.**
  - Commit the baseline record before any code moves (RC).
  - Build a comparator that checks, for each mutant, that its class is equal and that its killer's
    name is equal or listed in the batch's re-map, with no NO EVIDENCE allowed. Prove it can fail by
    planting a flipped verdict.
  - Use the comparator on every targeted sweep and on the closing one.
- **Confidence.** High.

### RX-13: The sequencing that keeps the net green and meaningful at every commit

- **Claim.** The order of work decides whether each step can be proved.
- **Batch 0** (no code moves; `make verify` green):
  - A-2.
  - RX-9's hardening, test-first.
  - RX-10's inventory and union declaration check, plus a `modes/tty` declset.
  - RX-8's widened router trace and the new app trace, recorded on code identical to `e57afeaf`.
  - RX-12's comparator, and the committed baseline.
  - The internal test-support package for pure helpers (RX-5).
- **One cluster per batch**, leaves first, best-supported first: the tty broadcaster, then the app
  deck with arbiter and executors whole. Map and map-prop come last, or after their re-typing (TS).
  Each batch has two commits:
  1. **Prep, in place.** Receivers re-homed, exports added, straddlers' subjects moved onto
     production API. Tests change only identifiers. Mutants are re-pointed by text. Full gate, plus
     a targeted sweep.
  2. **Move only.** `git mv` of source, tests and any testdata; package clauses and imports only;
     mutants re-pointed by path only; no comment edits. `git diff -M` shows renames plus clause and
     import lines.
- **Evidence each batch shows:**
  - goldens and both traces byte-identical;
  - the same test inventory;
  - the union declaration set conserved;
  - `mutant-anchors` and `mutant-check` green;
  - a targeted sweep of the batch's mutants, compared with the baseline: same class, and killers by
    name;
  - `make verify` green, MAKE_EXIT and ALL GATES GREEN read;
  - the re-map list written to the build log.
- **Close:** a full sweep compared with `since-v0.19.5.log`: 1,051 caught, the 7 equivalents
  surviving or re-ruled, 0 no evidence.
- **Consequence.** Each commit is either a refactor in place, proved by the gate and its mutants, or
  a move, proved mechanically. No commit is both.
- **Confidence.** Medium-high. The order of clusters is TS's and AS's to set.

## Open questions for the HUM LEAD

1. **FULL TDD for a move-only commit (D-4).** A pure move adds no behaviour, so it has no
   behavioural test to see fail first. RX proposes that the failing test be the batch-0 instruments:
   on a half-done move, the inventory or the union declaration check fails. Is that acceptable, or
   should a move commit carry something else?
2. **API widened for tests.** Where a chosen seam has straddlers, there are three ways forward:
   - export a narrow read surface the tests can use (precedent: `SetColorEnabledForTest`);
   - rewrite those tests against existing API and re-prove them;
   - leave that seam out of 0.19.6.

   Which should it be?
3. **Map and Propagation scope.** 289 mutants and 81 straddling test files, and the move needs a
   re-typing of 382 `Dashboard` methods first. Is that cluster in 0.19.6, or a later release?

## Touches other lenses

- **TS:** clusters, the `Dashboard` receivers (RX-2, RX-3).
- **AS:** the deck, arbiter and executors as one move (RX-3).
- **PB:** step-3 sweep cost for leaf mutants with top-level killers (RX-11).
- **TG:** Makefile package names, comparator, graph as a check (RX-9, RX-10, RX-12).
- **GP:** test-support package and `TestMain` practice (RX-4 to RX-6).
- **RC:** the 255→692 correction and the uncommitted baseline (RX-1, RX-12).
- **RM:** the app trace instrument (RX-8).
- **JN:** where a junior finds a cluster's tests when they stay at the top (RX-2).
