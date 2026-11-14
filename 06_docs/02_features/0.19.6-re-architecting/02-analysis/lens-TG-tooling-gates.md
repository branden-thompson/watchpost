---
title: "0.19.6 DISCOVER — lens TG: tooling, gates and ledgers"
date: 2026-11-14
phase: DISCOVER
lens: TG
status: "FILED — findings checked against e57afeaf"
---

# TG — tooling, gates and ledgers under a split of `modes/tty` and `app`

## Verdict

Most of the tooling keys on a **file path** and breaks loudly when a file moves: the mutant corpus,
the P10 symbol rows, the doc-symbol tests, the lint baseline, the goldens. That is the good direction.
It costs re-pointing, and some of the re-pointing needs the HUM LEAD's ratification.

The danger is the other class: **checks that keep passing while checking less.** There are four
kinds:

- 22 white-box tests that scan their own directory (`"."`) for a rule, and lose sight of any code
  moved into a subpackage. Four of them already log that they are "blind to any … outside package".
- Per-package test harnesses: the `TestMain` functions that isolate XDG state and pin `time.Local`.
  A test that moves into a new package does not take them with it.
- Hard-coded package lists (`vet-tags`, the doc-map test, `test-platforms`).
- Checks that an export silently switches off: `wires`'s `num` sentinel, the window-reply guard,
  the band-writer's `tty.` qualifier, and `unused`.

There is also a structural gap. Today the layering `modes ↛ app` is held only by Go's import-cycle
rule, because `app` imports `modes/tty`. A split takes that protection away for any `app` subpackage
that does not import `tty`. No gate names the rule.

Existing machinery can carry the new checks: `scripts/lint-imports.sh` and its self-test controls,
`platform/declset`, `platform/singleowner`, and the gate oracle. Each new check can be proven by a
planted failure.

## What I checked and how

- Read whole: `Makefile` (all 600 lines), `.github/workflows/ci.yml`, `06_docs/required-gates.txt`,
  `scripts/lint-imports.sh`, `scripts/lint.sh`, `scripts/lint-baseline.txt`, `.golangci.yml`,
  `scripts/quality/p10-unmatched.sh`, `tools/p10floor/main.go`, `tools/authoring/{main,cite,caps}.go`
  and `rules.go` (package-doc rule), `tools/wires/main.go`, `tools/dupes/main.go` (scan, site
  naming, ledger match), `tools/atlas/main.go`, `tools/prepush/main.go`, `platform/declset/declset.go`,
  `platform/singleowner`, `06_docs/mutants/mutants_test.go` (compile step),
  `scripts/quality/mutant-verdicts.sh` (to line 230), `scripts/quality/mutant-anchors.sh`,
  `cmd/watchpost/{docs,record,extending_symbols,gates}_test.go` (the relevant tests), and the P10
  mirror `06_docs/p10-ledger.md`. I read the local `.a2dh-p10-exemptions.yml` only to confirm the row
  format. I did not quote it.
- Grepped every `*.go`, `*.sh`, `Makefile`, `*.yml`, `*.py` and `*.expect` outside `.claude` and
  `06_docs/02_features` for `modes/tty`, `app/…go`, `"../app`, `"modes", "tty"` and the
  `filepath.Join` spellings (154 non-import hits). I then grepped the tests of `app` and `modes/tty` for
  every source-scanning call (`os.ReadDir(".")`, `filepath.Glob`, `declset.Files(".")`, named
  `ParseFile`/`ReadFile` of a `.go` file).
- Counted the mutants by target with the sweep's own rule: the first `pathlib.Path` literal,
  `scripts/quality/mutant-verdicts.sh:141`. I cross-checked the count against
  `dist/mutant-verdicts.timings`, and the per-mutant compile cost against `dist/mutant-check.log`
  (local artefacts, not committed records; say so wherever a figure from them is used).
- `go list` for the module's import edges into `app` and `modes`. Nothing was run beyond `go list`
  and greps. There was no `go test`, no `make`, and no `a2dh`.

**A correction to the brief.** The go-studs import rule is not in `tools/authoring`. It is
`check_studs` in `scripts/lint-imports.sh:23-33`. `tools/authoring` has no import rule at all.
The brief's mutant figure ("255 … 145 tty, 110 app") also does not match the corpus: see TG-9.

---

## Findings

### TG-1 — The layering `modes ↛ app` is held by the cycle rule, not by a gate, and a split removes it

- **Claim.** `lint-imports` forbids only `modes/* → domains/*` and `platform/* → domains/*`
  (`scripts/lint-imports.sh:12`, `:72`, `:81`), plus go-studs outside `platform/render` (`:23-33`).
  No gate states any of these rules:
  - nothing under `modes/`, `platform/` or `domains/` imports `app/...`;
  - `platform/` and `domains/` never import `modes/...`.

  Today they hold only because the reverse edge would be an import cycle. `app → modes/tty` is the
  only edge into `app` or `modes` other than `cmd/watchpost`'s (`go list`: `app -> modes/tty`,
  `cmd/watchpost -> app`, `cmd/watchpost -> modes/report`). After a split, an `app/<sub>` that does
  not import `modes/tty` can be imported by `modes/tty` and it will compile. The same holds for any
  `platform` package that `modes/tty` does not reach today: `sched`, `history`, `config`,
  `closedset`, and the rest outside tty's 16 `platform` deps.
- **Evidence.**
  - `scripts/lint-imports.sh:9-18` is the rule body: a textual grep for `"$MOD/domains/`.
  - `docs/architecture.md` (block 1) draws `app` as "the composition root". It draws modes as "read
    the snapshot, never a domain". It says nothing about modes and app.
  - `go list -deps ./modes/tty` reaches 16 `platform` packages of 24.
- **Consequence.** Principle 2: a split that adds packages without naming their allowed directions
  makes the architecture *less* traceable. The first back-edge compiles silently and nothing reports it.
- **Recommendation.**
  1. Extend `lint-imports.sh`, which is already in `verify` and CI, with its `--self-test` run by
     `gate-controls` (`Makefile:132`). Add `modes/`, `platform/` and `domains/` never import
     `$MOD/app`, and `platform/` and `domains/` never import `$MOD/modes/`. Give each rule a planted
     control in the self-test. The self-test's own "all four controls fired" count rises with it.
  2. For the new subpackages, keep an **allowed-edge ledger**: each package's permitted watchpost
     imports, in a tracked document. A check computes the real graph (`go list -json` or A-4's
     `pkggraph`) and fails on an edge not in the ledger **and** on a ledger edge that no longer
     exists. That second half is the stale-row expiry `wires` uses (`tools/wires/main.go` `triage`).
     Its self-test is a planted graph in each direction. Whether this becomes a required gate is a
     HUM LEAD question (Open question 4).
- **Confidence.** High.

### TG-2 — 22 source-scanning tests are scoped to their own directory, and go blind to code moved below it

- **Claim.** These tests walk `"."`, or a glob in the package directory, to hold a structural rule.
  Code moved to a subpackage leaves their sight without any failure. They split into three kinds:
  - **Absence rules** ("no file does X") pass emptier.
  - **Exactly-one rules** stay green when a second carrier appears in a subpackage.
  - **Presence rules** fail loudly when their subject moves, which is the good direction.
- **Evidence.** Four already state the ceiling in their own log:
  - `app/router_owner_test.go:63` ("blind to any created outside package app");
  - `app/maintrack_seam_test.go:123`;
  - `modes/tty/router_swap_test.go:107` ("blind to any reader outside package tty").

  | Test | Kind | What a move does |
  |---|---|---|
  | `app/cast_test.go:722` `TestNoProductionFileInAppReadsRuntimeGOOSDirectly` | absence | a subpackage reading `runtime.GOOS` is unseen; its control (`runtimeGOOS() != ""`) still passes |
  | `app/cast_test.go:861` `…BuildsANameOnlyVoiceSpec` | absence | same |
  | `app/geodata_test.go:16` `TestGeodataLoadsOncePerEntryPoint` | exact (`dashboard.go`, `app.go`, `:34`) | loud if either file moves; a third `geodata.Load` (36 ms, 19 MB) in a subpackage is unseen |
  | `app/inject_test.go:20` `TestTheInjectorIsInEveryBuild` | absence (tag), presence (`ticker.go`, `:40`) | the tag scan narrows; the `ticker.go` read is loud |
  | `app/logic_test.go:171` `TestNoListComparesZips` | absence over `.` and `../modes/tty` | non-recursive: both packages' children are unseen |
  | `app/maps_basemap_test.go:177` (seed only from `maps.go`, `:187`) | absence | a subpackage call is unseen |
  | `app/reachable_test.go:29` | every `*livePipelines` method referenced in production in-package | a caller moved out reads as dead: loud, false alarm |
  | `app/router_owner_test.go:17` | presence, ≥1 | a second `tea.NewProgram` in a subpackage is unseen |
  | `app/maintrack_power_test.go:125`, `app/maintrack_seam_test.go:69-123` | exactly one | a second caller below is unseen |
  | `modes/tty/main_test.go:30` `TestNoTestSetsTheZoneMidRun` | absence over `*_test.go` | subpackage tests unseen (and see TG-3) |
  | `modes/tty/held_clock_test.go:18` | absence over `*_test.go` | same |
  | `modes/tty/map_window_test.go:225` `TestMapTestsReachNoSocket` | absence, floor 1 file | narrows silently to any one remaining file |
  | `modes/tty/map_colour_guard_test.go:89` | absence over `map_*.go` and `../../app/map*.go`, floor 40 (`:99`); 48 files today (31 tty, 17 app) | loud only once more than 8 files leave |
  | `modes/tty/map_vision_test.go:166` | absence over `*.go` and `../../app/*.go`, **no floor** | silent |
  | `modes/tty/map_mode_test.go:162` | every `mapMode` site in `.` | sites below are unseen |
  | `modes/tty/router_lookup_test.go:139` | every unexported `*Msg` in `.` | unseen below, and see TG-6 |
  | `modes/tty/router_swap_test.go:73` | exactly one power reader | a second reader below is unseen |

  `platform/declset.Files(dir)` (`declset.go:51-73`) is the shared walker. It is deliberately one
  directory (`os.ReadDir`, `IsDir()` skipped).
- **Consequence.** Principle 4: these guard defects that have already happened (the `runtime.GOOS`
  Linux panic in `Makefile:200-203`, #23's ZIP identity, F-197's colour literals). After the split
  they stay green over less code, and nothing reports the loss.
- **Recommendation.**
  - Add a tree-scoped walk beside `declset.Files`: a package directory and every package below it,
    test files still excluded. Re-point every absence and exactly-one rule above to it, rooted at
    the *original* root (`app`, `modes/tty`). The rule then keeps its scope whatever the split does.
  - Give each a floor or a planted specimen where it lacks one (`map_vision_test.go` has neither).
  - Hold the pattern with a cmd-level check that no test outside `platform/declset` calls
    `os.ReadDir(".")` or `declset.Files(".")` for a structural rule. A planted file proves it.
  - Decide this at PLAN, before the first move: converting afterwards cannot show the rule held
    across the move.
- **Confidence.** High: every row was read at the cited line.

### TG-3 — Per-package test harnesses do not travel with the tests

- **Claim.** Two package-level `TestMain` functions carry invariants that exist nowhere else:
  - `app`'s isolates the developer's config and state. It sets `XDG_CONFIG_HOME` and
    `XDG_STATE_HOME` to a temp directory (`app/platform_sim_test.go:38-43`) and steers the platform
    seam from `WATCHPOST_TEST_GOOS` (`:27-29`).
  - `modes/tty`'s pins `time.Local = time.UTC` before any test (`modes/tty/main_test.go:21`).
  
  A test moved into a new package runs without either. The XDG case is the worst: a moved test that
  saves a preference writes the HUM LEAD's real `~/.config/watchpost` and passes. The tty case shows
  up as a race-only, one-in-three flake (the W14 history at `main_test.go:12-19`).
  `make test-platforms` names `./app` alone (`Makefile:213-215`), so a platform branch moved below
  `app` is never run "as linux".
- **Evidence.** Cited above. `TestMain` exists only in `app/platform_sim_test.go` and
  `modes/tty/main_test.go`, from a grep of both packages' tests.
- **Consequence.** Principle 4. The behaviour itself does not regress, but the harness that makes the
  tests trustworthy does, and the developer's machine state is put at risk.
- **Recommendation.**
  - Move both invariants into one shared test-support helper that every new package's `TestMain`
    calls.
  - Hold it with a cmd-level test that derives every package under `app/` and `modes/` and fails
    when one with tests has no such `TestMain`. A planted package proves it.
  - Derive `test-platforms`'s package list from the packages whose tests use the GOOS seam.
  
  Touches RX.
- **Confidence.** High on the mechanism. Which moved tests would actually write state depends on the
  package map (TS/AS).

### TG-4 — AP-CITE-01 keys known citations by directory: every moved comment with a bare ID fails `lint-authoring`

- **Claim.** `baseCitations` records each ID against the *directory* of the file that carried it at
  the merge-base (`tools/authoring/cite.go:38`). `checkCitations` then looks a comment's IDs up under
  its *current* directory (`:82`). A comment moved from `app/` to `app/<sub>/` finds an empty set at
  the base, so every bare ID in it (`D-139`, `F-22`) becomes "a new citation with no release". There
  is no rename handling. CI checks out with `fetch-depth: 0` for this rule (`ci.yml:41`), so it fires
  in CI too.
- **Evidence.**
  - `cite.go:23-45` and `:78-104`.
  - Scale, as an upper bound (a grep, not the tool): 1,285 comment lines in `app` and 2,480 in
    `modes/tty` carry an ID-shaped token with no release number on the same line. A release earlier
    in the sentence, or an `MVS-` prefix, qualifies some of them. The exact figure needs the tool run
    on a pilot move.
- **Consequence.** It fails loudly, which is the right direction. But the volume would push a batch
  toward one of two bad fixes: editing thousands of comments as part of a "pure move", or loosening
  the rule under pressure. Either works against principle 2 and the authoring standard.
- **Recommendation.** Re-key before the first move. The known set for a file becomes its directory's
  base set **joined with the base sets of the directories its content came from**.
  - For 0.19.6 the simplest sound key is the package *root*: an ID cited anywhere under `app/` at
    the base is known anywhere under `app/`, and the same for `modes/tty/`. Inside the moved code
    that is no wider than today's rule.
  - Add a self-test specimen: a moved comment passes, and a genuinely new bare ID still fails.
  - Record it as an A-row: it is a tool fix (0.19.5 D-13).
- **Confidence.** High.

### TG-5 — P10: 16 file-keyed rows break `verify` on a move, density rows silently cover subpackages, and `make p10` is not per batch

- **Claim.**
  1. **Symbol rows.** 16 ratified rows are keyed to files in the two packages: 10 in `modes/tty` and
     6 in `app` (`06_docs/p10-ledger.md:18`, `:49-56`, `:63`, `:172`, `:190`, `:212-213`).
     `TestEveryRatifiedP10RowNamesCodeThatExists` (`cmd/watchpost/docs_test.go:288`) runs in every
     `go test`, so in `make race`, `verify` and CI. It fails the moment such a file moves. The ledger
     changes only by HUM ratification (D-4), so **the re-key has to be ratified in the same batch as
     the move.** Seven of the rows cite a proof that depends on package structure: the router's
     `surfaceView` "whose only implementers are Broadcaster and Dashboard" (`:49-56`). A split that
     moves either surface changes the proof, not just the key.
  2. **Density rows.** The `app` row (FLOOR 0.61, `:122`) and the `modes/tty` row (FLOOR 0.42,
     `:129`) match by **directory prefix** (`scripts/quality/p10-unmatched.sh:113-115`, "mirrors the
     tool … for P10-05 only"). A new package under `app/` or `modes/tty/` inherits its parent's
     ratification and floor with no ledger change. A package placed anywhere else (say
     `modes/broadcaster`) has no row, so its live finding fails `make p10`. `p10floor` then holds every
     inherited finding to the parent's floor (`tools/p10floor/main.go:90-120`). A thin subpackage (the
     Bubble Tea plumbing the rows name as the dilution) fails loudly. A thick one passes under a
     ratification that was given for a different package: the case D-264 was written against.
  3. **Cadence.** `p10` is a phase-exit gate, not in `verify` or CI (`Makefile:323`,
     `required-gates.txt:26-30`). A batch can be green in `verify` and red in P10 until BUILD exit.
- **Evidence.** Cited above.
  - The last local run (`dist/p10.json`, 2026-11-10, base `7ca44489`) put the density findings at
    `app/airscope.go` (0.66 over 77 changed functions) and `modes/tty/body.go` (0.46 over 169). The
    margin over `app`'s 0.61 floor is 0.05.
  - **UNVERIFIED.** Whether `a2dh`'s density and analyzer passes count a function as changed when it
    moved byte-identical between files (git rename detection). If they do, every moved function is
    re-judged under every P10 rule, and debt in code that has not been touched since its rule began
    surfaces in the move batch.
- **Consequence.** Principle 3 (FULL P10). The prefix inheritance is the significant one: it extends
  a HUM ratification's scope without a HUM decision.
- **Recommendation.**
  - Put the inheritance question to the HUM LEAD (Open question 1).
  - Run `make p10` at the end of each 0.19.6 batch: an A-row for cadence, not a gate-list change.
  - At PLAN, dry-run one pilot move (one file into one new package) through `make p10` and
    `lint-authoring`. That settles the UNVERIFIED point and gives TG-4 its real figure. (0.18.0 L1:
    instrument dry-run at PLAN.)
  - Batch the 16 re-keys with their moves, as ratification requests.
- **Confidence.** High on 1 to 3. The changed-function semantics are UNVERIFIED and do not count.

### TG-6 — Exporting a symbol for a new boundary silently disarms four checks

- **Claim.** A split exports what crosses a package line, and four checks treat an exported, or
  differently qualified, name as out of scope:
  - **`wires`** recognises a closed set only by a sentinel starting with lowercase `num`
    (`tools/wires/main.go:311`), and fails only when it finds no sets at all (`:149`). Five of
    today's 13 sets live in the two packages:
    - `numMainTrackStages` (`app/maintrack.go:38`);
    - `numNarrationClasses` (`app/arbiter.go:50`);
    - `numRequestFields` (`modes/tty/request.go:39`);
    - `numModals` (`modes/tty/dashboard.go:1587`);
    - `numSurfaces` (`modes/tty/router.go:31`).

    Export one as `NumModals` and the set leaves the producer/consumer check with a green run.
  - **`TestEveryWindowReplyIsRoutedBackToTheWindow`** skips exported `*Msg` types
    (`modes/tty/router_lookup_test.go:152`). A reply message exported so a subpackage can send it
    leaves the guard.
  - **`TestOnlyMastercontrolWritesTheBand`** matches a qualified construction only when the qualifier
    is the identifier `tty` (`modes/tty/one_band_writer_test.go:35`). If `TickerBreakingMsg` moves
    to a package with another name, a non-owner's construction is unmatched. The guard still fails
    loudly if the *owner's* construction also stops matching (`platform/singleowner`'s
    blind-matcher rule), so the hole is a stranger's construction while the owner's still matches.
  - **`unused`** (golangci's standard set; `.golangci.yml:1-6` names no linters) reports unexported
    dead code only. Every symbol exported for the split leaves dead-code detection. The graph's 3,171
    unexported symbols used across files bound how many could; the package map decides how many do
    (TS, AS). After the split, only `*livePipelines` keeps a reachability check
    (`app/reachable_test.go`).
- **Evidence.** Cited above.
- **Consequence.** Principles 2 and 4: green gates over a smaller subject, with no failure to show it.
- **Recommendation.**
  - `wires`: accept `Num<Set>` as a sentinel too, and pin the set list: the ledger names every closed
    set, a set that vanishes fails, and a self-test plants an exported sentinel.
  - `router_lookup`: judge every `*Msg` the package tree declares, exported or not.
  - `one_band_writer`: match the qualifier through the import's path, not the identifier `tty`.
  - Dead code: put exported-for-split symbols under `internal/` so they are not public API (`pkg/` is
    the only public tree). Consider a pinned whole-program `deadcode` run
    (`golang.org/x/tools/cmd/deadcode`) as a ratchet in the shape of `scripts/lint.sh`. The
    `internal/` choice is an API decision (Open question 3). Touches GP.
- **Confidence.** High on `wires`, `router_lookup` and `one_band_writer`, all read at the line.
  High on the `unused` semantics. The size of that last effect depends on the cut.

### TG-7 — `declset` proves a pure move inside one directory only; `modes/tty` has no golden

- **Claim.** `declset.Compare(dir)` holds one directory's top-level declarations against
  `dir/testdata/declset.txt` (`platform/declset/declset.go:146`, `:165-183`). A move from `app` to
  `app/<sub>` shows as `removed` in one and as a missing golden in the other. The test's own way out
  is `-update-declset` (`app/declset_test.go:15-24`), which re-captures in silence. So the instrument
  0.19.5 built for exactly this question (red-team R2-22: `git diff --stat` cannot certify a split)
  cannot certify a cross-package one. `modes/tty`, the larger package, has no golden at all. Only
  `app`, `platform/{snapshot,render,history}` and `domains/weather/nws` do.
- **Evidence.** Cited above, and `find -name declset.txt`.
- **Consequence.** Principle 4: "behaviour byte-identical" (D-1, option A) goes without its
  mechanical proof for the moves.
- **Recommendation.**
  - Capture a `modes/tty` golden on `e57afeaf`'s code before any move.
  - Add a union mode: the declarations of a root and every package below it, compared against the
    root's base golden under an explicit, reviewed rename list (each export is a rename).
  - Move goldens and fixtures with `git mv` and check them as R100 renames in move batches, never
    regenerated with `-update-golden`.
  
  Touches RX.
- **Confidence.** High.

### TG-8 — Hard-coded package lists go stale after a split; some are guarded, some are not

- **Claim.** INST-1 asks that subjects be derived. These lists are written out:

  | Where | Keys on | After a move | Guarded by |
  |---|---|---|---|
  | `Makefile:62` `vet-tags` | `./modes/tty ./platform/history` | moving `modes/tty/map_property_full_test.go` leaves `vet-tags` green, vetting no tagged file there | nothing: **silent** |
  | `Makefile:526` `property` | `-run TestThePrintedMapIsAFreshRender ./modes/tty` | selects nothing | `TestEveryRunSelectorSelectsATest` (`cmd/watchpost/gates_test.go:746`): loud |
  | `tools/prepush/main.go:28` | `./app` runs `TestTheAsBuiltMapIsKeptInStep` | selects nothing | `TestThePrePushHookRunsTheRecordsTests` (`record_test.go:326`): loud |
  | `cmd/watchpost/docs_test.go:353` `TestEveryFileIsInItsPackagesMap` | `{"modes/tty", "app"}`, floor 10 rows (`:366`) | every moved file leaves a stale row: loud; **new packages' `doc.go` maps are never checked** | partly silent |
  | `Makefile:213-215` `test-platforms` | `./app` | see TG-3 | exempt as a manual probe (`gates_test.go:79`) |
  | `Makefile:532` `quality-bench` | `./modes/tty …` | moved benchmarks are not run | a measurement, not a gate; PB |
- **Evidence.** Cited in the table.
- **Consequence.** Principle 2 (the package maps are the junior's front door: JN) and principle 3.
- **Recommendation.**
  - `vet-tags` becomes `go vet -tags property ./...`.
  - The doc-map test derives its list: every package under `app/` and `modes/` with more than a set
    number of files, or every one, carries a `doc.go` map.
  - Derive `test-platforms` as in TG-3.
  - Each change keeps a planted failure.
- **Confidence.** High.

### TG-9 — The mutant corpus: 692 mutants over 110 files break loudly on a move and need vetted re-anchoring

- **Claim.** Every mutant names its target as a repository-relative path in its first
  `pathlib.Path`. It holds a literal anchor (`06_docs/mutants/m48_cue_send_deleted.py`). A moved file
  reads `MISSING` and a changed line reads `DRIFTED` in `mutant-anchors`
  (`scripts/quality/mutant-anchors.sh:60-66`). That runs in `verify` and on every CI push
  (`ci.yml:82`), so a move cannot hide.
  - An export rename (`foo` → `Foo`) inside an anchor also drifts.
  - The sweep derives each mutant's step-2 package from its target's directory
    (`mutant-verdicts.sh:141-148`), and its last killer from the test name, wherever that is defined
    (`:155-162`). Verdicts stay sound across moves.
  - A mutant whose detector stays in the parent while its code moves into a child is not caught at
    step 2 (the child's own suite). It reaches step 3 (`./...`), which still catches it.
- **Evidence.**
  - Counted with the sweep's own rule: **539 mutants target `modes/tty` and 153 target `app` (692),
    over 72 and 38 distinct files (110).** `dist/mutant-verdicts.timings` agrees (539 and 153). The
    brief's "255 (145 tty, 110 app)" is not this corpus.
  - The most-anchored files: `modes/tty/map_prop_update.go` 56, `router.go` 37, `map_prop_words.go`
    31, `broadcaster.go` 31, `app/executors.go` 19, `app/radio.go` 15.
- **Consequence.** Principle 4 is protected, at a cost: each re-anchor is re-vetted (the required
  reading's rule 4).
- **Recommendation.**
  - Each batch's plan lists the mutants its files carry, by this count.
  - Re-point anchors in the move commit, with `mutant-anchors` before every commit.
  - Have the closing sweep compare by mutant file against `since-v0.19.5.log` (A-3).
  - Prefer moves that carry a file's white-box tests with it, so step 2 stays the catcher (PB).
- **Confidence.** High.

### TG-10 — Path-keyed record checks fail loudly on a move; one resolves bare file names anywhere

- **Claim.** These need re-pointing in the move commit. All are loud:
  - `docs/where-things-happen.md`: 168 `path:Symbol` refs into `app` or `modes/tty`
    (`TestWhereThingsHappenNamesRealSymbols`, `docs_test.go:62`).
  - `docs/accepted-costs.md`: 5 (`:144`).
  - `docs/extending.md`: 60 paths; `docs/architecture.md`: 11 (`extending_symbols_test.go:249-258`).
  - 0.19.0's living documents: 28 `app`/`tty` symbol citations, 24 of them in its
    `implementation-plan.md` (`record_test.go:271`).
  - `scripts/lint-baseline.txt:1,2,5`: the ratchet fails both ways, `scripts/lint.sh:78-88`.
  - The singleowner owner map: `"app/mastercontrol.go"`, `one_band_writer_test.go:23`.
  - `segments_completeness_test.go`: `radio.go`, and `../platform/report/report.go`, which is
    depth-relative.
  - `memo_positions_test.go:61`.
  - 64 goldens in `modes/tty/testdata`.
  - `cmd/watchpost/{measure,journey}_test.go`, which read `app/testdata/maps/…`.

  Two are silent:
  - `indexModule` registers every `.go` file by its **base name** too
    (`extending_symbols_test.go:57`), so a page naming `credits.go` without a directory resolves
    wherever the file goes. The control itself hard-codes `app/credits.go` (`:353`), which is loud.
  - The record checks read the release record at `propagation-overlays` (`record_test.go:20`), so
    0.19.6's own living documents are not held by them. That is RC's to weigh.
- **Evidence.** Counts by grep over the named pages. Lines cited.
- **Consequence.** Principle 2 (docs reflect current reality): the cost is mechanical and the checks
  hold it.
- **Recommendation.**
  - Re-point in the same commit as the move ("docs move with code").
  - Make the bare-base-name match count only when the base name is unique in the tree.
  - Re-point `releaseRecord` at the release in flight when 0.19.6 opens its plan (RC).
- **Confidence.** High.

### TG-11 — Name-keyed ledgers survive moves; export renames are what break them

- **Claim.** These are robust to a move:
  - `dupes`, which matches a fingerprint and the exact site set by `Type.Method`, with no package in
    the key (`tools/dupes/main.go:283-291`, `:109`; `duplicates-ratified.md:39`);
  - `wires`'s member names (bare identifiers);
  - the follow-ups and roster checks (test names, `docs_test.go:188`, `:252`);
  - `alloc-budget` (unanchored across `./...`, its floor of 9 at `gates_test.go:386`).

  An export rename of a ratified site breaks its `dupes` row, which is loud and needs
  re-ratification. `dupes` also catches split-by-copy, a helper duplicated to avoid an import, but
  only at 25 AST nodes or more. A small copied helper passes.
- **Evidence.** Cited above.
- **Consequence.** Low risk. One HUM touchpoint: a renamed site in `duplicates-ratified.md:44`
  (`livePipelines.bedRelays` · `mastercontrol.givenWay`) needs re-ratifying if either type is
  exported.
- **Recommendation.** No change. List affected ratified rows per batch.
- **Confidence.** High.

### TG-12 — Nothing ties the drawn architecture to the import graph, and 0.19.6's batches are not held in step

- **Claim.** `tools/atlas` regenerates the page from the documents (`tools/atlas/main.go:1-16`).
  `TestTheAtlasMatchesAFreshRender` holds the page to the documents, never the documents to the code.
  `TestTheAsBuiltMapIsKeptInStep` (`app/maps_test.go:56`) holds the as-built page's batch count only
  for `observer-maps` and `propagation-overlays`. Under FULL DIAGRAMS (D-4), 0.19.6's diagrams could
  lag its batches with every gate green. The package diagram in `docs/architecture.md` (block 1) has
  no check that its nodes are the packages that exist.
- **Evidence.** Cited above.
- **Consequence.** Principle 2.
- **Recommendation.**
  - Add 0.19.6 to `TestTheAsBuiltMapIsKeptInStep`'s table when its as-built page opens. The test
    lives in `app` and reads `../06_docs` (`:57`), so it fails loudly if it moves without
    re-pointing.
  - Consider a check that the `docs/architecture.md` package diagram names every package under
    `app/` and `modes/` that `go list` reports, and no other. A planted missing package proves it.
- **Confidence.** Medium: the second check's value depends on how the diagram is drawn at PLAN.

### TG-13 — Gate time is dominated by `mutant-check`, which scales with the mutant count, not package size

- **Claim.** A split changes `race` (the critical path) and the sweep's step 2. It barely changes
  `mutant-check`, which is most of a `verify`.
- **Evidence.**
  - Green `verify` runs took 2,061 to 2,743 s (`06_docs/verify-runs.md:235-244`).
  - `mutant-check` alone took 1,572 s in the 2026-11-11 local log (`dist/mutant-check.log`, not a
    committed record).
  - Per mutant, `TestEveryMutantAppliesAndCompiles` runs `go build ./...` and then `go vet` of the
    target's package (`06_docs/mutants/mutants_test.go:153`, `:160`). Its mean subtest time is
    26.9 s for tty mutants, 27.1 s for app and 25.4 s for `platform/lineup`. The whole-tree build
    dominates, and a split moves this little.
  - `race`: `modes/tty` is the slowest package under `-race`, at 226 s local and 290 to 489 s hosted
    (`Makefile:35-41`). 0 of 1,085 tty tests and 1 of 687 app tests call `t.Parallel`, so the package
    is the unit of parallelism. Split pieces run concurrently under `go test`'s `-p`, bounded by the
    runner's cores and offset by more test-binary links.
  - The sweep's step 2 runs the target package whole: 14.9 s per tty mutant and 21.0 s per app
    mutant on average (`dist/mutant-verdicts.timings`: 8,013 s / 539, 3,213 s / 153). Smaller
    packages shorten it, unless detectors are left behind (TG-9).
  - There is no coverage step in `verify` or CI.
- **Consequence.** Principle 1: the gain sits in `race` and the sweep, not in `verify`'s largest
  step. Do not promise it from intuition.
- **Recommendation.** PB measures `race` per package and a pilot split's effect on both. The
  `race -timeout 30m` comment ("modes/tty, the slowest package") is re-recorded when the slowest
  package changes.
- **Confidence.** Medium. The figures come from local artefacts, as stated. The direction is high.

### TG-14 — New checks that would hold the new architecture, and how each is shown to fail

| Check | Holds | Machinery to carry it | Watched failure |
|---|---|---|---|
| Import direction: `modes`, `platform`, `domains` ↛ `app`; `platform`, `domains` ↛ `modes` (TG-1) | the layering | `scripts/lint-imports.sh` (`verify`, CI), with its self-test in `gate-controls` | a planted file per rule in `--self-test`; the gate oracle (`TestEveryRequiredGateCanFail`, `gates_test.go:478`) proves the exit code is wired |
| Allowed-edge ledger among the new packages (TG-1) | each package's permitted dependencies, stale rows expiring | a new `tools/layers` built from A-4's `pkggraph` or `go list -json`, with a ledger in the shape of `wires-ratified.md` | a planted graph with an unlisted edge, and one with a stale row |
| Tree-scoped structural scans (TG-2) | absence and exactly-one rules across a root and its children | a tree walk beside `platform/declset.Files` | each rule's planted specimen in a child directory |
| Shared `TestMain` invariants (TG-3) | XDG isolation, the GOOS seam, a UTC clock in every package's tests | a test-support helper, plus a cmd-level presence check | a planted package with tests and no `TestMain` |
| Cross-package pure-move proof (TG-7) | no declaration added, dropped or renamed except by a listed export | `platform/declset` in union mode | a planted dropped declaration |
| Package size ceiling | D-1's reason for the release cannot quietly come back | a cmd-level test over `go list`, ceilings recorded when the split ends (figures: Open question 2) | a fixture over its ceiling |
| Ownership by visibility | "only X does Y" | unexported in the owner package (the compiler is the gate); where an export is forced, `platform/singleowner` keyed on the package path (TG-6) | `singleowner`'s existing fixtures (`TestCheckInFailsForANonOwner`) |
| Exactly one package doc per package | every new package has a front door | `tools/authoring` SN-03, which flags two or more today and passes zero (`rules.go:364`) | a planted package with none (JN) |

Wire each into `required-gates.txt`, `verify-gates` and `ci.yml` together:
`TestTheThreeGateListsAgree` (`gates_test.go:187`) refuses a gate that is in only some of the three.

---

## Open questions for the HUM LEAD

1. **P10 density for new packages.** Does a package created under `app/` or `modes/tty/` ride its
   parent's ratified density row and floor (today's prefix matching, no ledger change), or does each
   new package get its own row, ratified at its own measured floor? (TG-5; D-4 FULL P10.)
2. **A package size ceiling.** Should 0.19.6 end with a ceiling per package that a test holds, and
   at what figure? (TG-14; it sets a standard the release has not set.)
3. **`internal/` for symbols exported only for the split.** Should they live under `internal/` so
   they are not part of the public module API? (`pkg/` is the public tree today.) (TG-6; an API
   question.)
4. **The allowed-edge ledger as a required gate.** Should it join `required-gates.txt`, a list
   that changes by HUM ruling? (TG-1, TG-14.)

These are A-row candidates, not questions: the AP-CITE-01 re-key (TG-4), `make p10` at the end of
each batch (TG-5), the `wires` sentinel and set pinning (TG-6), `vet-tags ./...` (TG-8), and the
tree-scoped walker (TG-2).

## Touches other lenses

- **RX:** TG-2 (absence guards going blind), TG-3 (harness invariants), TG-7 (pure-move proof),
  TG-9 (re-anchoring 692 mutants).
- **RC:** TG-10 (path-keyed docs; `releaseRecord` pinned to `propagation-overlays`), TG-12
  (diagrams).
- **PB:** TG-13 (where gate time goes); TG-9 (detectors left in a parent push mutants to `./...`).
- **TS / AS:** TG-6 and TG-1. How many symbols a cut exports, and which edges it creates, decide how
  much these findings cost.
- **GP:** TG-6 (`internal/`, `deadcode`).
- **JN:** TG-8 (`doc.go` maps for new packages), TG-14 (one package doc each).
- **RM:** TG-6 (the band's single owner, `app/mastercontrol.go`, keyed by file).
