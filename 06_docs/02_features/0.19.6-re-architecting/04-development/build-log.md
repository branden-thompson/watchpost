---
title: "0.19.6 re-architecting — BUILD LOG"
date: 2026-11-16
phase: BUILD
sev: SEV-0
status: "In progress: one entry per batch commit, in order"
---

# Build log

Each entry names its plan items (`implementation-plan.md`), what changed, how each instrument was seen
failing before it passed, and the gate it landed on. Each push's remote ref is recorded (A-33).

## B0.1 and B0.2: the sweep's killers, the comparator, the scoped sweep

**What.**
- **The killer record (A-2, H-4).** `make mutant-verdicts` reads its killers through
  `go run ./tools/mutantmap killers`: the release's own record when it exists, else the newest
  `since-v*.log` by the version in its name, so `since-v0.19.10.log` follows `since-v0.19.5.log`. The
  sweep's log names the record it read. A record the sweep is told to read and cannot find stops the
  run at once (`scripts/quality/mutant-verdicts.sh`).
- **`MUTANT_LIST`.** The sweep judges exactly the mutants a file lists, and refuses one the corpus
  lacks. A listed run, like a `MUTANT_ONLY` run, records no timings.
- **`tools/mutantmap`.** One tool, three answers, each from git:
  - `sites <rev>`: where each mutant edits, found by running the mutant with its read served from the
    commit and its write caught, so the corpus has one parser, the harness's own. At `e57afeaf` all
    1,058 locate; five sit outside a function (three in the `Makefile`, one in the release workflow,
    one in `modes/tty/doc.go`). The baseline is committed as
    `06_docs/mutants/verdicts/anchors-e57afeaf.tsv` (A-30).
  - `derive -base <rev>`: the batch's re-vet list (A-11, A-30): mutants whose anchor's file or
    function changed, whose last killer's declaration changed, whose own file changed, and new ones.
    From `e57afeaf` to this branch it derives none, since only documents changed: 51 s over the
    corpus.
  - `compare`: a scoped sweep against the baseline record: the derived list judged exactly, each class
    and killer kept (by package and name), each anchor where the batch's maps say it went.
- **`make mutant-scoped BATCH_BASE=<rev>`** (D-30): derive, sweep with A-3's killers, compare, clean
  the build cache, never promote. Its verdict is the comparator's, because the sweep exits 1 for the
  seven kept equivalents that survive by ruling. Listed in `gates_test.go`'s exempt table.

**Seen failing first.**
- `tools/mutantmap`'s tests plant each loss in a throwaway repository: a mutant not judged, one judged
  but not derived, a catch lost, another killer, an anchor that moved function, a killer that moved
  package, and the same moves passing once the maps name them. The first run failed on two of them,
  a missing-file message git words two ways and a map that moved a test file's staying tests, both
  fixed before the tool was used.
- `TestTheSweepRefusesInputsItCannotHonour`: the committed script, given a missing killer record,
  reported only "no mutant matches" and said nothing of the record; the edited one refuses it by name.
  The list case was not run against the old script, which does not read `MUTANT_LIST` and would have
  started a full sweep.
- A smoke run on two mutants: m105 CAUGHT by `TestTheNextCardIsBuiltWhileThisOneReads`, m16 SURVIVED
  (a kept equivalent); the sweep exited 1, the comparator 0, as the target now reads it.

**P10.** `make p10` found one live finding, a scanner loop with no visible bound in
`tools/mutantmap/sites.go` (P10-02); it reads the driver's lines by `range` now. Then 0 live, 0
unmatched, 0 unratified.

Landed as `b1327208` on a full gate (2,152 s, green); pushed to `feature/re-architecting` on GitHub.

## B0.3: the quiet guards made loud

**What.**
- **`declset.Tree`, `SourcePaths` and `TestPaths`** walk a package and every package below it, testdata,
  hidden directories and nested modules left out. 18 structural rules that walked one directory now
  walk their tree, rooted where they were written: the two `runtime.GOOS` and voice-spec rules, the
  geodata loads, the debug tag, the power report, `startSynth`'s seam, the reachability walk, the
  program's model, the ZIP identity, the zone seeding (`app`); the held clock, the zone write, the map
  modes, the map tests' sockets, the colour literals, the vision ramps, the reply routing and the
  power reader (`modes/tty`). Each that had no floor has a fixed one now: 213 test files, 72 map test
  files, 48 map files, 160 production files.
- **`TestNoStructuralRuleScansOnlyItsOwnDirectory`** (`cmd/watchpost`) fails any test under the two
  trees that walks `"."` by `os.ReadDir`, `declset.Files`, a local glob, or a list that starts with one.
- **The file-map test derives its packages:** every package under `app` and `modes/tty` carries a
  `doc.go` with its file map, the two of today the floor.
- **`property`, `vet-tags`, `quality-bench` and `test-platforms`** name `./...` trees, and
  `property`'s tests are anchored by name. A `-run` that matches nothing passes empty, so
  `TestThePropertyGatesTestsExist` fails when a test the recipe names is declared nowhere under the
  tree it is run over. The first form checked in the recipe that each test printed PASS; the gate
  oracle runs every recipe with its tools stubbed to prove a gate can fail, and a stubbed `go` prints
  no PASS, so that `verify` FAILED (exit 2, 248 s, row kept) with `TestEveryRequiredGateCanFail`,
  `TestVerifyCanFail`, `TestEveryControlIsReached` and `TestNoFileSilencesARequiredGate` calling
  `property` unjudgeable. The check moved into the Go test, and the recipe is a plain `go test` again.
  The next `verify` FAILED too (exit 2, 387 s, row kept), at `dupes`: the pinned-set comparison in
  `tools/wires` was the same 65-node shape as `declset.Diff`. It calls `declset.Diff` now.
  `quality-bench` covers every benchmark in the two trees, benchstat pinned to the installed
  `v0.0.0-20261009192801-be2c69fb417e` (D-28).
- **TG-6's three export-sensitive checks.** `wires` reads `Num<Set>` as a sentinel too, and pins its
  17 closed sets (TG-6 counted 13; the tool finds 17), so a set that vanishes fails. The reply guard
  judges exported messages too, each named with the routing that delivers it, and fails on an excuse
  for a type that no longer exists. The band-writer check matches a construction under any qualifier.
- **`scripts/quality/p10-unmatched.sh`'s header** says its prefix rule for density rows is its own; the
  tool matches the exact directory (D-31).

**Seen failing first.**
- The new check listed all 18 rules before they were re-pointed, and passes after; its detector
  finds its five planted shapes.
- `wires -self-test` with the old lower-case-only sentinel: the planted `NumModes` scenario FAILED;
  with the change it passes. The file was restored by copy.
- The band-writer check with the old `tty`-only qualifier: its planted `deck.TickerBreakingMsg{}`
  control FAILED; restored by copy, it passes.
- `TestThePropertyGatesTestsExist` with the recipe naming `TestNoSuchPropertyTest`: FAILED, naming it;
  the recipe restored by copy, it passes.

**P10.** `make p10` shows no live finding, but two density rows now fire with no floor stated
(D-277): `platform/declset` measures 1.00 over 5 changed functions and `tools/wires` 0.20 over 5.
Writing a floor changes the ledger, which only the HUM LEAD ratifies. The HUM LEAD ratified both
and writes them into the ledger (D-34); the regenerated mirror lands with the commit after the edit.

Landed as `4d948def` on a full gate (2,129 s, green), after the two failed runs above; pushed.

## B0.4, B0.9 and B0.10: the TestMain ground, the import layers, the citations

B0 lands in five commits rather than the plan's four: this one carries the three lighter items, so a
failure here does not hold up the inventory, the order tests and the lock check (one more full gate).

**What.**
- **B0.4.** `modes/tty/internal/ttytest` holds `Main`, which pins the zone to UTC; `app/internal/apptest`
  holds `Main`, which gives a run its own config and state directories. Both are imported by test files
  only. `modes/tty`'s and `app`'s `TestMain` call them; `app`'s applies `WATCHPOST_TEST_GOOS` first, as
  D-24 asks of every package whose code reads the platform seam.
  `TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain` (`cmd/watchpost`) holds every package with
  tests under each tree to its helper, and the GOOS rule; the two packages with tests at B0 are its
  floor. Each new package carries a `doc.go` with its file map, which the derived file-map test holds.
- **B0.9.** `scripts/lint-imports.sh` holds the 0.19.6 layers: `modes`, `platform` and `domains`
  never import `app`; `platform` never imports `modes`; the deck and the map data never import each
  other; `ttytest` and `apptest` have test importers only. Each rule refuses a planted file in the
  self-test, 13 controls in all.
- **B0.10.** AP-CITE-01 keys a file under `app` or `modes/tty` by the tree's root (`tools/authoring`,
  `citeKey`), so a comment moved below the root keeps the citations its tree carried at the base, and
  a new bare ID still fails. "`git log` on the file" reads `git log --follow` in
  `docs/where-things-happen.md` and `06_docs/ids.md`. The docs' symbol check resolves a bare file name
  only when one file has it: it found four bare names in `docs/extending.md` that already named two
  or three files (`report_test.go`, `themes.go` twice, `settings_kept_test.go`), now given their
  directories.

**Seen failing first.**
- The TestMain check on the committed `TestMain`s: both reported as not calling their helper; and its
  planted package without one.
- Every new import rule refused its planted file in the self-test.
- The authoring self-test's moved-comment specimen: MISSED on the committed `cite.go`, which looked
  the comment up under `app/internal/deck`; caught after.
- The bare-name rule: four real ambiguous names in `docs/extending.md`.

**P10.** The two new packages fire P10-05 at 0.00 (`ttytest` over 1 changed function, `apptest` over
2): each sets the zone, the directories and the exit, with no state worth a check, and none is added
for the count (0.19.5 D-27). Their density rows go to the HUM LEAD in this batch (D-31).

Landed as `fc39a8a7` on a full gate (2,899 s, green); pushed. The two rows are ratified (D-35); the
HUM LEAD adds them to the ledger with D-34's floors.

## B0.5, B0.8 and B0.11: the inventory and the union, the order that must not move, the shared lock

**What.**
- **B0.5.** `tools/testinv` records app's and modes/tty's tests and subtests with their outcomes from
  `go test -json` (`run -mode plain|property|race`), and compares a run with the committed baseline
  through the batch's move map and its reviewed renamings (`compare`). `make test-inventory` runs the
  three modes; it is the third exempt probe (D-36). The baseline in `06_docs/testinv/` is captured at
  B0's close. `TestTheTwoTreesDeclarationsAreConserved` (`cmd/watchpost`) holds every top-level
  declaration under the two trees, 3,319 at B0, with a hash of its tokens, comments and layout
  dropped and a split package's qualifier dropped, against `cmd/watchpost/testdata/union-decls.txt`:
  a move changes none, and a prep commit re-captures it with `-update-union` and lists the diff in the
  changed-line table.
- **B0.8.** `TestThePostUpdateStepsRunInTodaysOrder` holds `Dashboard.update`'s twelve steps, in order,
  each once; `TestANonMapMessageStillReasksAStaleSnapshot` drives GP-3's re-ask through
  `Dashboard.Update` on a message the map does not answer. `TestTheMapLayersKeepTheirOrder` pins the
  thirteen layers' registration order (PB-4). `TestTheMessageTypesArePinned` holds the 45 `Msg` types
  the two trees declare (R8).
- **B0.11.** `TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage` finds the 25 sections under
  `livePipelines.mu` (its floor) and follows each one's static calls through the tree. It fails on a
  function reached that takes the lock again, on a call that crosses a package of the planned map,
  with no `…Locked` allowance (A-32), and on a `…Locked` function that takes the lock. A call through a
  func value passes only for the four values design AP-3.5 records, keyed by their declaration, five
  rows, and a row no section reaches fails as stale. The planned map is
  `app/testdata/sharedlock/planned.tsv`: package map §2's AS Map I, file by file, with A-23's four
  changes and design AP's 108 method rows. A file it does not place fails; it is deleted once B7 lands.
  `TestCommitAndStopAllFinishUnderTheSharedLock` runs commit, the bed's hooks, the fragment hook and
  the transmitter's re-station 200 times each, together, then `stopAll`. Its 2 s bounds progress, not
  the run (A-35): the paths' own work outside the lock, the pool scan and the config writes, takes
  3.4 s.

**Seen failing first.**
- **The union check.** A statement planted in `everyTick` failed it. A comment planted there passed.
  The first version hashed the printed source, which keeps comments, so it is now the tokens. Its
  controls hold the tokens' rules: a split qualifier, a comment and a blank line do not count, and
  another package's qualifier and a statement do.
- **The B0.8 checks.** The message pin reported a planted `plantedMsg`. The layer order failed on a
  swapped pair. The post-update order and the re-ask failed on edits to `dashboard.go`.
- **The lock check.** The planted fixture fails each of its four rules. In the tree, a call to
  `currentPool` planted in `setWatch`'s section was reported as a deadlock. A `…Locked` method planted
  in the deck's file and called there was reported as a crossing.
- **The two lock tests and mDL1.** Both are killed by the new vetted mutant `mDL1`
  (`06_docs/mutants/mDL1_commit_takes_the_shared_lock_again.py`), which plants a second lock in
  `commit`'s section; the deadlock test fails in 4.1 s with every goroutine's stack. `mDL2` needs
  the deck's `StopProducer`, so it is written and vetted in the deck batch (design AP-3.6).

**P10.** `tools/testinv` is at the density floor with checks that catch something. The comparison
fails two baseline tests mapped onto one key: without that check, a merge read the second as kept
and reported nothing (`TestCompareFailsTwoTestsMergedOntoOne`). An inventory with an unknown outcome
or a key listed twice is refused, and a run that records no test fails. Each check was seen failing
with its guard removed. Its stream reader ranges over lines rather than looping on a scanner (P10-02).
B0.10's `citeKey` measures `tools/authoring` under its stated floor; the floor is restated by A-36.
The live findings are the two test-support rows the HUM LEAD is adding (D-35).

**What the walk cannot see.** A method call on a value whose type it cannot read resolves to the
tree's methods of that name. When the tree has none, the call resolves to nothing, so a func field
on such a value goes unseen. The deadlock test runs the paths that would hide one.

Landed as `b531111d` on a full gate (2,184 s, green); pushed.

## B0.6's router trace and B0.13: the trace in its new form, the record, the floor

B0's fifth commit. B0.6's `app` trace, `TestTheDeckIsTheSameUnderRandomActs`, is the next commit's:
it drives the deck's act surface over a player and a synth that run on goroutines of their own, and a
byte-identical record of that needs its own harness work.

**What.**
- **B0.6, the router trace** (design CK-6, A-29, A-38). Each step is
  `<seed>.<step> m<index> surface=<n> cmd=<description> frame=<sha256>`, and nothing prints a Go type.
  - The Dashboard is built with a real map: the embedded basemap, offline feeds, a Settings save that
    records, a Propagation update that answers at once, and the map's ticks through `heldTicks`.
  - The alphabet is widened once: the map's keys, Propagation's, Settings', and `ctrl+b` / `ctrl+o`.
    The old trace had never reached the console.
  - Bubble Tea's timers and combinators are named; every other command is run and named by its
    reply's index in `traceReplies`, the reply fed back once.
  - Four walks on fixed seeds start from the dashboard, the map, the map in Propagation and the
    console, each place held to a fixed floor of frames.
  - The golden, 1,200 lines, was re-recorded on production code identical to `e57afeaf` (only the two
    test-support packages differ). It is on B0's list of re-recorded goldens.
- **B0.13, the record.** `releaseRecord` points at 0.19.6; the requirement checks and the FR-7 test
  read 0.19.0's record (A-37). The rulings check takes a bare suffix, a planned name and a fixed list
  of whole-file claims, and fails on a listed claim its row dropped. The 16 line citations in the
  brief, the architecture and the plan are now symbols.
- **B0.13, the floor** (D-12). `make vuln-floor` runs `govulncheck` under the toolchain `go.mod`
  names, as a plain recipe: a first version as a shell script was refused by the shell ledger's
  test, since a new script needs a ruling and this one is not necessary. It is required in
  `verify`, in CI and so in the release job. The floor is now go 1.26.11, `README.md`'s build section
  restates it, and the CHANGELOG has an `[Unreleased]` entry.

**Seen failing first.**
- **The router trace.** It failed at line 305 with zoom-in's `+` binding removed. It is
  byte-identical over three plain runs and under `-race`.
- **The reach floors.** Uniform draws showed Propagation in 0 frames and failed them, which is why
  the walks exist.
- **`vuln-floor` at go 1.26.10.** It found GO-2026-6859 reachable in `net/http@go1.26.10` (fixed in
  1.26.11) and failed. At 1.26.11 it found nothing.
- **The record checks.** On 0.19.6 they reported nine rows that name a whole file, and a plan
  citation of a file B3 creates.

**The gate, throttled from here on.** The first full gate on this commit was stopped at 33 minutes, in
`mutant-check`, by the HUM LEAD's direction: at Go's default parallelism (every package and every
thread on all 18 cores), with the machine's endpoint scanner reading each compiled test binary, the
load average reached about 50 and other applications shut down from power use or CPU temperature. A
stopped run writes no row to `06_docs/verify-runs.md`, so it is recorded here. A second run, under
`nice -n 19` with `GOFLAGS=-p=6`, was stopped too, at load 84 in `mutant-check`.
`TestEveryMutantAppliesAndCompiles` copies and compiles the whole tree once per mutant, 1,059 times,
four lanes at once, each lane's build taking its own `-p`. Every heavy run since is under macOS's
background priority, `taskpolicy -b nice -n 19` with `GOFLAGS=-p=2` and `GOMAXPROCS=4`: the
efficiency cores only, at low power, with the disk throttled. That changes the scheduling and not the
gates, so a run takes longer.

## B0.14: the testing infrastructure (D-37 to D-39)

The gate ran this machine into shutdowns (above), and the HUM LEAD asked for tests as effective,
using less, with full checks well inside 18 hours. The mutant check was about three quarters of every
gate: other gates ran 6 to 7 minutes before it began. Three rulings followed, one at a time. This
commit lands before the B0.6/B0.13 commit above, set aside as a patch until it does, so that commit
is gated by the new commit gate.

**What.**
- **D-37, the mutant check by overlay.** `judge` copies only a mutant's one target, runs the mutant
  there, and builds and vets with `-overlay` in the real tree. Every unchanged package is a cache hit,
  and the about 56 MB copy per mutant, about 59 GB written a gate, is gone. A target outside the tree
  is refused before the script runs. A script that writes to the real target is ESCAPED, a verdict the
  copy never returned. `TestEveryMutantAppliesAndCompiles` fails if `git status` differs after the
  corpus.
- **D-38, two tiers.** `MUTANT_SCOPE` is `commit` for `make verify`: `inScope` selects the mutants
  whose package, or a package it or its tests import, holds a changed file, a changed script, and every
  mutant on a change to `go.mod` or `go.sum`. `make verify-full` runs the same gates with
  `MUTANT_SCOPE=full` under its own lane in `06_docs/verify-runs.md`. CI's mutant step and the release
  job run full. On this commit the scope selects 0 of 1,059 (no product package changed); a probe line
  in `app/every.go` selected 161.
- **D-39** is the full sweep's GitHub workflow, written before B9 needs it.

**Seen failing first.**
- **`TestTheJudgeGivesEachVerdict`** on a throwaway module: the copy-based judge read the planted
  escape as INVALID, and its target can be an absolute path. With the overlay judge, all seven cases
  give their verdict.
- **`TestTheCommitScopeSelectsWhatAChangeReaches`**: with the test-import half of `reach` removed, it
  missed the mutant reached only through a test.

**Measured.**
- **A spike on 3 mutants** on the efficiency cores: 14 to 16 s each by copy, 7 to 8 s by overlay, and
  27.5 against 19.3 CPU-seconds for the widely imported `platform/render`.
- **The whole corpus by overlay**: `make mutant-check` exited 0 in 4,466 s, all 1,059 applying and
  compiling as on the last green gate, the tree unchanged. That ran on the efficiency cores at the
  lowest priority, never above 8 cores, against 1,907 to 3,512 s on every core at full power by copy.

Landed as `c68777a4` on `make verify-full` (6,803 s, green, the whole corpus); pushed. The B0.6/B0.13
patch was then applied back over it, its one conflict in the Makefile (`vuln-floor` beside
`verify-full`) resolved by keeping both.


## 2026-11-17: the P10 ledger at D-34, D-35 and A-36

The HUM LEAD approved the ledger edits ("Ledger edits approved") and vetoed none of A-34 to A-38 ("No
vetos at this time, will likely be contingent on UAT phase(s)").
- **D-34.** `platform/declset` carries "FLOOR 1.00 over 5 changed functions" and `tools/wires` "FLOOR
  0.20 over 5", each measured 2026-11-16.
- **D-35.** `modes/tty/internal/ttytest` and `app/internal/apptest` have their rows, FLOOR 0.00 over 1
  and over 2 changed functions.
- **A-36.** `tools/authoring`'s floor reads "FLOOR 0.25 over 4 changed functions".

`make p10` on the efficiency cores: the check found no live finding, every in-scope row matched one,
the 5 density rows in the diff stand at or over their floors, and all 167 rows carry a ratified date.
The mirror, `06_docs/p10-ledger.md`, is regenerated at 167 rows and committed with this entry.

## B0.6's app trace: `TestTheDeckIsTheSameUnderRandomActs`

`app/deck_trace_test.go` drives the act surface the entry point hands out (design AP-7.2) with four
fixed seeds of 150 acts each, and records every act's events, effects, reports, published messages,
voice calls and MasterControl's state, one block an act. The golden, `app/testdata/deck_trace.golden`,
is recorded on code identical to `e57afeaf`: since then B0 has added test support only (`apptest`,
`ttytest`, `platform/declset`, the shared-lock fixture). The harness's choices past the design's
words are A-39.

**Seen failing first.** The first recording never settled: a scripted read that came home at once
played the whole line-up inside one act, 10,000 steps. A read now stays on the air until the clock
moves. With `DropCard` emptied by hand, the trace fails at the first `drop` act; the file was
restored by copy.

**Measured.** 5,603 lines, 365 KB, in 1.6 s on the efficiency cores. The record is identical over 30
runs, and over 3 under `-race`. Its first form was 1.5 MB and `TestNoTrackedBinaries` refused it: a
card's id is its place's coordinates, and the proposals and the line-up were said in full after
every step. Cards are named by their place in the pool now, and a list that has not changed is said
`(same)` (A-39, point 6).

## B0.7, the pins: the comparator, its baseline, the three outside the gate, three new pins

Every pin `make alloc-budget` runs reports through `internal/allocpin`, one line a measurement, and
the recipe writes `go test -json` to a file that `tools/pins` reads, holding each to
`06_docs/perf/pins-e57afeaf.tsv`: 50 pins, each one's range over 10 clean runs, on this tree, whose
product code is `e57afeaf`'s. Any count above the range fails, any byte figure more than 0.1 % above
it, a baseline pin with no measurement, a measurement with no row, a test that reports none, and any
failed test or build in the stream. Its form past the plan's words is A-40.

**The three outside the gate (PB-7).** `TestEveryRaceSkippedPinIsInTheAllocBudget` fails a test that
skips under the race detector and measures allocations when alloc-budget's pattern misses it. On the
old pattern, `'AllocBudget'`, it named exactly PB-7's three: the Router's cost over the Observer, the
layout once a view and key, the running order's rows. They are named in the selector now. The pin
count's floor is 21.

**Three new pins.** A beat with Settings open, 2,642 allocations and 870,681 bytes: sixteen times
the idle beat, its frame laying out every tab's groups to measure the window's width. Recorded, not
cut (0.16.0 D-53). `Update` with the map open, 15 and 9,512. Time to tone, 183 to 185 allocations
(A-17's 185), its bytes not held.

**Made to repeat before the baseline was taken.** The first 10 runs showed the beat with Settings
open moving 0.4 % in bytes, and time to tone 0.2 %, both past the 0.1 % the comparator allows, so a
later run would have failed by chance. The beats now average a fixed 100 beats, which brought the
Settings beat to 0.08 %. A takeover's goroutines allocated inside the next run's window: the tone pin
now waits for them, and its count holds at 185 where it read 186. Its bytes still fell run by run,
21,726 to 21,680 over ten minutes, and its fixture is declared at the wall clock, so its count alone
is held.

**Seen failing.** The comparator on planted streams: a rise of one allocation, bytes past 0.1 %, a
fall (passes, said), a pin unmeasured, a row missing, a label twice, a test with no pin, a failed or
skipped test, a broken build, another platform (checked and said), another Go (fails). On the real
tree, one escaping 64-byte allocation planted in `advanceTicker` failed seven pins, `MAKE_EXIT=2`,
among them the idle beat at 169 allocations, which its own budget of 176 lets through. The file was
restored by copy. `make alloc-budget` on today's tree: "50 pin(s) held to the baseline, 0 fell".

**What the gates found.** The first form piped the run into the comparator, and
`TestEveryRequiredGateCanFail` showed `make alloc-budget` exiting 0 with only its `go test` red: a
pipe's exit is its last command's. The run goes to a file now. `TestEveryControlIsReached` asked for
the comparator's own control: `make pins-selftest` judges the committed baseline, every pin at its
maximum passing and six planted faults each failing, and is on all three gate lists.

**Measured, on the efficiency cores.** One alloc-budget run is about 45 s.

## B0.7, the rest: what the compiler decides, the models' sizes, the paired wall time, startup

**The compiler probe.** `tools/gcprobe` keys every escape to the heap by package, function and type,
and every inlined call by caller and callee, over the packages the binary links under `modes/tty` and
`app`. Its baselines are recorded on `e57afeaf` in a worktree, and `TestTheTreesKeepWhatTheCompilerDecided`
holds the tree to them: today's tree matches exactly, 4,001 escapes and 7,208 inlined calls.
`modes/tty`'s 4,040 inlined calls are PB-3's figure from DISCOVER, and `Router.Update`'s one escape of a
`Router` is PB-2's box at `router.go:463`. The form past the plan's words is A-41.

**Seen failing.** On a planted module with the repository's layout: a model moved to the heap fails
as a new escape of its type; a method marked `//go:noinline` fails as a lost inlining; a renamed
function fails until a `-map` carries it. On the real tree, `Router.View` planted to keep a pointer
to its model failed as `escape modes/tty | Router.View | Router`, and as one more escape of the type
`Router` across the trees. Restored by copy.

**Getting the types right.** The first record typed 331 of 2,839 keys. The compiler names a file
relative to the directory it builds in, and the probe joined that to a relative directory while it
indexed files absolutely. And the compiler places a binary expression at its operator, a call at its
bracket. The probe now makes the directory absolute, takes the innermost expression whose text is the
compiler's or that its position anchors, types a name only when it is a variable, and skips a
builtin's name for its call: 63 keys of 1,991 are left in the compiler's own text.

**The models.** `TestTheModelsAreNoLarger` holds Router, Dashboard and Broadcaster at PB-2's 7,136,
6,584 and 472 bytes. A field planted in `Broadcaster` failed both it and `Router`, which holds the
console by value.

**The paired wall time.** `make bench-pair` runs D-28's two pairs against a base commit and judges
them with benchstat at its pinned version. On planted arms through that benchstat: 5 % slower in both
pairs fails, in one pair alone passes, a benchmark that moved package keeps its pair, a base
benchmark with no pair fails, and a name two packages declare is refused. It is local and
HUM LEAD-run, declared `unlisted` with its reason.

**Startup and binary size at BUILD's open**, from `e57afeaf` built by `make build`:
32,732,082 bytes, PB's P-11. The inits take 11.5 to 12.4 ms in all over five runs, `platform/render`'s
about 7 ms of it; `modes/tty`'s init is 0.046 ms and 73,824 bytes, `app`'s 0.014 ms
(`06_docs/perf/startup-e57afeaf.txt`).

**What the gate found.** The first `make verify` failed in `race` after 1,359 s:
`TestThePublishedTreeNamesNoPersonOrMachine` read the toolchain's directory in the module cache, its
name, an at sign and its version, as an email address in both baselines. A file from the module cache
is keyed without its version now, the standard library as `std/`, so an upgrade renames no key
either, and the baselines were recorded again on `e57afeaf`: the same 4,001 escapes and 7,208
inlined calls.

**Three density rows, ratified (D-40).** `tools/pins` 0.84 over 19 functions, `tools/gcprobe` 1.00
over 32 and `tools/benchpair` 0.54 over 13, against P10-05's 2.0: development tooling never linked
into the binary, its checks where a broken input would read as clean. The HUM LEAD: "ratified; we
never inflate density with unnecessary assertions." The rows are in the ledger and its mirror.

## B0.12, the graph tool: M3's figures, rebuilt and reproduced on `e57afeaf`

`tools/graph` type-checks `modes/tty` and `app` with their tests and resolves every use to its
object, wherever it is declared. The scratch tool PLAN used saw only uses within a package
(CQ F-17). Its form past the plan's words is A-42.

**Reproduced on `e57afeaf`.** `Dashboard`'s 752 methods in 68 files (receiver `Dashboard` or
`*Dashboard`); the map's reach 72 (60 fields), Settings' 102 (93), the console's 13 (7); the map's
reach by type 93 (43 `Dashboard`, 21 `Config`, 20 `mapChoices`, 6 `setupState`, 3 `uiChoices`: members
declared outside the map's files); MS-7's 408 test functions in 86 files, 288 C3 and 120 C4. And `app`'s
unexported crossings under the planned map, 283, recorded for the first time.

**What did not reproduce, and the rulings.** The hub's fan-in is 70 counted across packages, which
the brief asks for: `app/dashboard.go` uses `tty.Dashboard`, and the scratch tool could not see it.
D-41 made it 70, the target 40. Settings' 102 reproduces only under JN's keying, a field by its bare
name. The agent first measured 103 by an object keying that was wrong, and D-42 was ruled on that
figure; by object the figure is 124, `Config` and `UIPrefs` mirroring 22 field names, and D-44
replaced D-42 with JN's keying, D-27's targets standing. MS-7's 408 needed the 2b plan's own
symbols, from `crossings-MS.tsv`: the four that go to the kit leave the map's side, the root
symbols that move join it. Its C3/C4 rule was in the lost `classify_tests.py`; MS-7's words applied
literally disagree on 127 of the 408. The committed boundary holds as data (D-43): the 91 root
symbols the design's C3 tests reach are the allowlist, and every C4 test reaches one outside it.

**Seen failing.** On a planted module: a pass-through, the map's package reading the model through
a carrier `Env` and through the hub across packages, counts each field reached, and without the
carrier on the area's row the carried field is not counted; the fan-in counts a file in `app`; a
test reaching a root field outside the allowlist is C4; a crossing under the planned map is found;
`check` fails a held figure that rose and reports a raised MS-7 count without failing.

**Today's tree** holds every figure at its baseline. MS-7 reads 289 and 122: B0's `Update` pin, its
re-ask behaviour test and its router trace reach the map's internals.

**Ratified (D-45):** `tools/graph`'s density row (D-10), 0.61 over 36 functions.

**What the gate found, for the pins.** The graph tool's second commit gate failed in
`alloc-budget`: "layout alone" read 4,543 bytes against a 10-run range of exactly 4,536, 2.5 bytes
past the comparator's 0.1 %, its count unchanged. Run alone it reads 4,536 every time. In the gate,
goroutines earlier tests left running allocate inside the window, a few allocations over 20 runs.
`allocpin` now measures each window three times and keeps the least count and bytes, since such an
allocation only adds (A-40, point 8). The next gate failed on counts: "Propagation window hit" 1,210
against 1,209, time to tone 185 against 183. A collection inside the window empties every `sync.Pool`,
and the path's next pooled buffer is a new allocation: this is M1's "flicker by one". With the
collector off inside the windows, after one collection to settle, three full runs repeated to the byte.
The baseline was recorded again by that method. The FAILED rows stay in `verify-runs.md`.

## B0.12, the surfaces held

`TestTheDecksExportedSurfaceIsHeld` (`app`) holds AP-1.5's tiers 2 and 3, the 43 exported methods on
the deck's unexported types, exactly, with the seven outside the deck listed beside them; `radioDeck`
and `mastercontrol` are held to the interfaces they are handed out as. `TestTheRootsExportOnlyWhatIsListed`
(`cmd/watchpost`) holds `modes/tty`'s 154 exported names and `app`'s 11 to
`06_docs/graph/exported-*.txt`. `TestOnlyD26sThreeExportsAreUsedOutsideByTestsAlone` finds exactly
`ChipKnown`, `FireAll` and `SameOverlay`. Their form is A-43.

**Seen failing.** A planted exported method on `radioDeck`; `RenderCast` taken off `app`'s list (the
floor first masked it, and was lowered to a floor against an empty list); a planted test-only use.
D-26's rule first found 21 names: scoped to the names the map's files declare, as MS C-6 counted, it
finds the three.

## B0's close: the golden gate, the inventory baseline, the plan's tests held

**The golden gate (B0.5).** `TestNoGoldenChangesOutsideABatchsList` (`cmd/watchpost`) holds every
file under `modes/tty`'s and `app`'s `testdata`, 108 at B0's close, to
`06_docs/testinv/goldens.tsv`, a hash and a path a file captured at this commit: a file whose content
matches a baseline file passes, so a pure rename does; any other must be named in a batch's list,
`06_docs/testinv/goldens-<batch>.txt` (A-33). B0's list names its five exactly: the router trace in
its new form, the app trace, and the shared lock's planned map and fixtures. Seen failing on a
planted re-capture of `about-133x44-ascii.golden` and a planted addition, both restored.

**The inventory baseline (B0.5, D-36).** `06_docs/testinv/plain.tsv` and `property.tsv` record 2,006
tests and subtests of the two trees, `race.tsv` 2,005 (`TestTheMapsOpenMeteoWeightIsWithinItsBudget`
is built without the race detector). The first race run lost 227 of `modes/tty`'s tests: `testinv`
ran `go test` with its default 10-minute timeout, and `modes/tty` under the race detector on the
efficiency cores runs past it. It now passes `make race`'s 30 minutes.

**The plan's tests held (A-37).** `TestEveryTestALandedBatchsPlanNamesExists` holds every test a
landed batch's section of the plan names to the tree; B0's names seven, all declared. A batch adds
itself to `landedBatches` in its closing commit.

**B0 in commits.** 13, the plan's 4: D-37 and D-38 split the gate in two mid-batch, and B0.6, B0.7 and
B0.12 each took more than one commit; the batch map says so. B0 closes on this commit's
`make verify-full`.

**B0 closed, 2026-11-20.** `make verify-full` on `d33fc847`: every gate green, the whole corpus of
1,059 mutants judged, 7,877 s on the efficiency cores. Pushed; the remote ref matched the commit.

## B1.2, the vocabulary: RM-8's and JN-5's names (D-46 to D-49, A-44)

**Put to the HUM LEAD singly.** JN-5's four that change code, not only comments: the receiver `d`
on four types (D-46, none renamed), the seven basenames `app` and `modes/tty` share (D-47, each move
batch rules), the duck path's four names (D-48, `dip` and `duck` unified) and `Dashboard` hosting
every window (D-49, its comment and row corrected with B1.3's stack).

**The one word is `duck`** (A-44). `mastercontrol.dip` is renamed for the interface call it makes,
`narrationVoice.duck`; the reverse would rename the interface, its ten test voices and the
`"duck()"` lines `app/pump_test.go` asserts. The vocabulary's duck row names the path's four calls
in order. `06_docs/perf/renames.tsv` is the release's renames file: gcprobe's tree test applies it,
seen failing without it (`mastercontrol.giveWay` and `.hold` each lose their inlined call to
`(*mastercontrol).dip`), and the scoped sweep takes it as `SCOPED_FUNCS`. `tools/mutantmap` reads
its comment lines as gcprobe does.

**The comments.** RM-8's seven places, and three of the same class outside RM's four files: the
deck's `abandonRead`, `renderSlots`, and the synth limiter's reserved pair, which called the
arbiter the Station Director. The Reader's header says one caller. `app/doc.go` names
`cmd/watchpost` as the subject of its import sentence (JN-5, RC-7 item 4).

**The changed-line table.** No test's setup or assertion line changed; one test comment
(`app/arbiter_race_test.go:25`, `dip` to `duck`). `union-decls.txt`: `(mastercontrol).dip` is
`(mastercontrol).duck`, and `giveWay` and `hold`, its two callers, re-hashed for the call's new
name. `app/testdata/declset.txt`: the one line, on B1's golden list. mE7's anchor text: `m.dip()`
to `m.duck()`, its replacement unchanged. The test inventory: one test added,
`TestTheMapsReadAsGcprobesDo` (`tools/mutantmap`, outside the two trees).

## B1.1, F-158's owed test (design AP-7.1, RM-4; A-45)

**The test.** `TestACutBackUnderAHazardReadKeepsTheBedDown` (`app/cutback_hazard_test.go`), on the
shipping path: the station goes on the air with the bed carrying, a Tornado Warning reaches the rail
and is held on the air, and the operator cuts back through MasterControl's `CutBed(false)`. The
Director answers with `Restore`, the precondition; the bed stays down while the hazard reads, and
comes back when the read ends. Stable over 50 runs under `-race` and 200 without.

**Seen failing.** Under the new mutant `mF158`, the Restore executor lifting the bed without the
arbiter, the test fails on its second assertion; mE6's killer catches it too. On a Director planted
to emit no `Restore`, the test fails with "no longer exercises F-158". Both were applied by hand and
restored by copy.

**Deck-only.** The graph tool's check (`make graph`) reports the file deck-only, and fails on a use
of `stationFrom` planted in it. The app trace reaches 22 root objects today, as AP-7.2 says it does
until the deck's prep commit, which adds it to the check's list.

**The changed-line table.** No changed test line. The test inventory: one test added in `app`, and
`TestADeckOnlyTestReachesTheDeckAlone` in `tools/graph`, outside the two trees. The corpus: one
mutant added, `mF158`.

## B1.3, RC-7's window stack and the window host (package map §9 row 2; D-49)

**The windows are a stack** (D-106, D-107), and the six places that said one window say so:
`docs/where-things-happen.md`'s window row and its `modal` vocabulary row; `docs/extending.md`'s
new-window recipe, which also names the window's keys in `window_keys.go:windowKeysOf`, the step
`TestEveryWindowDeclaresItsKeys` holds; and `open`'s comment, which said "alone" and "scrolled to the
top" where a window returned to keeps its scroll. **`Dashboard` hosts every window** (D-49): its type
comment and the Observer vocabulary row say so, with the four gates' file.

**Not here, by package map §9:** 0.16.0's required reading (`:190`, the single value as the trap;
`:204`, "three owners") moves with the console's and the deck's batches; the P10 ledger's stale reason
with D-10's re-key table; `modes/tty/doc.go:33`'s row with the move batches.

**The changed-line table.** No test line changed; no golden; no declaration.

## B1.4, M4's before: its doc findings (`../08-reports/m4-before.md`)

**Walkthrough 6** names its examples by step, not by a search: `ReduceMotion`, `reduceMotion`,
`rowReduceMotion` and `cycleReduceMotion` for the UI toggle, `MapQuakeFeed`, `mapQuakeFeed` and
`rowMapQuakes` (with `mapPrefArrow`) for the map picker. Step 10's golden is the row's own tab,
`setup-133x44-ui` or `setup-133x44-maps`. **The time row** names the two places that choose a date
themselves, `modes/tty/broadcaster_manifest.go:cardPulledShort` and `app/burst_words.go:spokenWhen`.
**`MapPropRefresh`'s two comments** said empty was each new grid; empty is hourly
(`platform/config/config.go:316-319`, `modes/tty/map_prop_settings.go:22`'s `propRefreshKeys`), and
each new grid is "every-10-min". **`cardPulledShort`'s cell count** compared forty-two cells with a
card of "about sixty"; its one card is UP NEXT (`readBody`, `broadcaster_slots.go:154`, called only from
`upNextBox`, `broadcaster_upnext.go:106`), 36 cells at the console's floor (`docs/extending.md:238`).

**The CHANGELOG** already had its `[Unreleased]` heading, opened by B0.13's floor entry, so only the
walkthrough line is new: `docs/extending.md`, under the invariants, says every user-visible change gets
an entry whichever walkthrough it follows.

**The changed-line table.** No test line changed; no golden; no declaration (two comments in
`modes/tty/dashboard.go`, one in `modes/tty/broadcaster_manifest.go`).

## B1.5, M4's midpoint (D-29; A-46)

**The trace.** A fresh blind reviewer traced the four tasks on `6060309b`, the before's brief word for
word, on a copy without this release's folder (`../08-reports/m4-mid.md`, `../08-reports/m4-mid-trace/`).
**0 of 4 pass, as before; on the sites and the routes, 2 of 4 (T3, T4), from 0.** Each of B1.4's fixes
was met and held. Every task fails D-24's ID test, on families `ids.md` has no row for (U2-, W, PD-, S-,
SAM-D-), which no batch clears: D-50 has `tools/ids` index them (B1.6).

**Fixed after the trace, the docs' and not the split's at REVIEW:**
- T1's D1: Walkthrough 6 step 10 sends a map Setting to README's MAP list; step 3 names `mapConfig`,
  where a map Setting's line is, beside `ttyConfig`;
- T2's wrong route: `modes/tty/map_boxes.go:7-8` named a legend box 0.18.0 D-103 retired; it names the
  colour row under the map and `scrubRows`, the caller the reviewer could not find;
- T3's test comment: `modes/tty/broadcaster_upnext_test.go:97-98` says 36 cells, as `cardPulledShort`'s does.

**The trace defined IDs.** Its tables are led by the ID a reviewer met, the shape of a defining row, so
`06_docs/ids.md` gained D-44, D-87, D-94 and others as 0.19.6 rows. `tools/ids/main.go:aboutIDs` now keeps
both measures' traces and keys (`08-reports/m4-`, `08-reports/m5-`) from defining any, and
`TestAReportAboutIDsDefinesNone`, which held only M5's keys, holds both: seen failing on the midpoint's
trace with the old filter, green with the new.

**The changed-line table.** One test comment (`broadcaster_upnext_test.go`); in `tools/ids`, one function
(`aboutIDs`) and its two callers, and `TestAReportAboutIDsDefinesNone`'s condition and message; no golden.

## B1.6, D-50's ID families (A-47)

**`06_docs/ids.md` resolves every ID of six more families the code cites:** U1- and U2- (UAT
findings, 67 cited), W tasks (90), PD- (5), S- (8), SAM-D- (11) and S-F (4).
`TestEveryIDOfD50sFamiliesTheCodeCitesIsIndexed` holds each to a row; `TestD50sFamiliesAreReadAsTheRecordWritesThem`
pins the forms: a task's decimal, a wave alone defining nothing, two UAT rounds as two sections, an
untabled task and an S-F finding at their mentions. `tools/authoring`'s self-test adds the families to
AP-CITE-01's controls. The index grows from 2,532 rows to 2,946, and past `TestNoTrackedBinaries`' 400 KB
floor (395,638 bytes to 468,405), so it joins `largeAllowed` with its reason.

**Two comments restated:** S-F3 and S-F4, 0.9.0 red-team findings the record never mentions
(`domains/radio/synth/install.go:252`, `:360`), now say what they ruled without the ID (A-47, after
0.19.5 D-37).

**The sweep's red baseline.** B1's first scoped sweep (`make mutant-scoped BATCH_BASE=40d3cfda`)
judged nothing: its baseline, `go test ./...` on `6060309b`, came back red while the blind trace and a
`cmd/watchpost` run shared the machine. The same command in a fresh worktree of the same commit is green
(`../.mutant-baseline-b1.log`, EXIT=0). The script discarded the baseline's output, so the cause cannot
be named; a red baseline now writes its failures into the sweep log. The sweep runs again at B1's close.

**Three mutants re-pointed.** The first gate failed at `make mutant-anchors`: m807, m809 and m810
anchored on `tools/ids/main.go` lines this batch rewrote (the qualified-mention test, now with its `own`
clause; the priority suffix, now after `idText`; the M5 reports' exclusion, now `readable`'s `aboutIDs`).
Each is re-pointed at its rule's line, applied by hand and seen killed by its own killer.

**The changed-line table.** No golden, no app or `modes/tty` declaration. `tools/ids`: `aboutIDs`
(B1.5), `readable`, `findingRows`, `samRows`, `taskRows`, `written`, `familyOrder`, and `mentionRows`'
`own` argument; `idset`: `Task`, `Grep`, `Parts`, `Written`; `tools/authoring`: `idKey` and one control.

## B1's close: the sweep's red baseline named, the inventory, the plan's tests held (A-48)

**The sweep's red baseline, named.** B1's second scoped sweep, on `6dd5bd5b`, came back red as the
first did, and B1.6's log named the test: `TestTheSweepRefusesInputsItCannotHonour`. The sweep runs its
baseline in a worktree with its own `MUTANT_KILLERS` and `MUTANT_LIST` set, and the test handed the
script the whole environment; its second case inherited `MUTANT_KILLERS=dist/mutant-scoped.killers`, a
path the worktree lacks, and was refused as a missing record. No full gate sets either variable, so
every `make verify` passed. `withoutSweepInputs` drops every `MUTANT_` variable before a case sets its
own; the old test was seen failing under the sweep's environment, the new one passing under it. The
sweep is re-run on this commit.

**The inventory (D-36).** `make test-inventory` against B0's baseline: every one of the 2,006 plain,
2,006 property and 2,005 race tests and subtests kept with its outcome; one new in each mode,
`TestACutBackUnderAHazardReadKeepsTheBedDown` (B1.1). B1 moved no test.

**The plan's tests held (A-37).** B1 joins `landedBatches`; every test its section names is declared.

**B1 in commits.** 6, the plan's 2: B1.1 to B1.4 each took a commit of their own, and the midpoint's
grade added B1.6 (D-50); the batch map says so. B1 closes on this commit's `make verify-full` and the
scoped sweep.

**B1 closed, 2026-11-21.** `make verify-full` on the closing tree: every gate green, the whole
corpus judged, 7,654 s. The scoped sweep on `da224fc8`: its baseline green, all 10 mutants CAUGHT by
their recorded killers; its one report, m810's anchor in `readable` where it sat in `followUpRows`, is
B1.6's re-pointing, and B1 closes on it (D-51). Pushed; the remote ref matched the commit.

## Before B2: the console's non-drawn references (D-52 to D-58)

**Put to the HUM LEAD singly, with their killers** (D-22, A-33): the nine references in seven tests
that design CK-5 left in class X. Two split by package, the internal half into the console with its
mutants (D-52: mAU2 and mW5; D-53); two assert the drawn value (D-54: the pool row's bold fire mark,
mAQ1 and mAQ2; D-57: an ASCII-only console frame); the two kit reads go through `frame.WeatherRow`,
already exported for production (D-55, D-56); the masthead's named inputs move to a kit-level test
(D-58: m744, m745, m747). No test-only export. Each is built in the batch whose move needs it: the
kit's in B2, the console's in B3.

**The first docs gate failed** on `TestEveryRulingNamingCodeCitesItsLine`: five sites in D-52 to D-58
named a Go file without its line or symbol. Each now cites one (`router.go:1058`, `:306`,
`view.go:renderModal`, `broadcaster_pool.go:fireBold`, `masthead.go:mastheadKeyOf`).

## B2: the kit and the test support (`modes/tty`)

**The re-check (step 1), on `76fbbb46`.** Design CK-1's and CK-4's facts against the code: the
closure's 41 declarations and 4 methods, the 25 names and 5 methods exported and 14 kept, each
exported name's non-test user outside the kit, `indentBody`'s one caller, the memo counters'
lines (`broadcaster_memo.go:139-147`, `memo.go:155-163`, `:505-513`) with no mutant on them, P3's
snapshot-message sites and `check_studs` (`scripts/lint-imports.sh:31`) all hold. Changed:
- `tickMsg`, `tickEvery`, `tick` and `armShimmer` sit 5 lines lower in `dashboard.go` (`:1058`,
  `:1064`, `:1068`, `:1122`);
- `tickEvery`'s row lists `map_prop_update.go` as a user, which only reads a struct field of the same
  name; its one root user is `Dashboard.after` (`dashboard.go:1076`), so M5's zero still holds;
- `mastheadSlot` has two users its row does not list, `body.go:32` and `broadcaster_header.go:26`,
  in the packages it names;
- the masthead guard is at `masthead_memo_test.go:56` and its named inputs span `:21-50`;
- **not in the design:** R8's pin, `cmd/watchpost/messages_test.go:pinnedMessages`, names `tickMsg`,
  so exporting it as `TickMsg` changes the pin's entry in the prep commit; the count is unchanged.

**Instruments first: a rename replaces a whole name (A-49).** B2's kit renames are bare names
(`tick`, `sides`, `trend`, `plural`), and `gcprobe` applied `renames.tsv` as a text replace in each
key's fields, which would have renamed the ticker's 96 keys with `tick`. `replaceWhole` matches a
name only where it stands whole; `TestARenameReplacesOnlyAWholeName` was seen failing under the text
replace. It lands before the prep, in a commit of its own.

## B2's prep: the kit and the test support cut in place (A-50; D-59 to D-61)

**A correction to B1's close.** B1 closed without `make p10`, which its close list named. Run on
B2's prep it found what B1 left: `tools/ids/idset`'s density, 1.00 over D-50's two functions (D-60,
a row ratified), and `app/mastercontrol.go` at 0.33 over the three functions B1.2's rename touched,
under `app`'s floor of 0.61 (D-61). The file row first ruled matched nothing, because the harness
keys density by directory; `tools/p10floor` now reads a file's floor from a note in its package's
row, seen failing under the old reading. `make p10`: 0 live, 0 unmatched, 0 unratified.

**The kit, in place (design CK-1).** 41 declarations and their methods cut whole, each with its doc
comment, into `kit_row.go`, `kit_cursor.go`, `kit_tick.go`, `kit_memo.go` and `kit_text.go`, and
into `masthead.go` for what the masthead alone reads. 25 names and 5 methods exported by
`gopls rename`, `tick` as `ShimmerTick` (D-59), and `NoTransmitter` moved in (A-31): 31 entries on
the root's export list until the move takes them out. `MemoStats` embeds its mutex, with `Hit` and
`Miss` (A-21). `renames.tsv` gains B2's 31 lines, each scoped to `modes/tty` (A-49), and
`TestTheTreesKeepWhatTheCompilerDecided` carries every key through them.

**The test support, in place (design CK-4).** The guard's engine and the pure helpers cut into
`ttytest_perturb_test.go` and `ttytest_helpers_test.go` under their exported names, for
`internal/ttytest` in the move commit.

**Mutants.** 21 re-pointed: 12 by name, 8 by path to the kit files, m241 to `NoTransmitter`. All
1,060 anchors match. m744, m745 and m747 were applied by hand: each fails
`TestTheMastheadKeyCoversItsNamedInputs` and no other test.

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| `masthead_memo_test.go` | the guard, split (D-58) | assertion | m744, m745, m747 by `TestTheMastheadKeyCoversEverythingTheMastheadShows` | by `TestTheMastheadKeyCoversItsNamedInputs` |
| `broadcaster_masthead_memo_test.go` | the console's half (D-58) | assertion, moved | (in the guard) | none recorded |
| `kit_masthead_test.go` | colour on at the start | setup | — | — |
| `broadcaster_pooldata_test.go` | the agreement line (D-55) | assertion | none | none |
| `detail_fire_test.go` | the row's count (D-56) | assertion | none | none |
| `router_lookup_test.go` | an exported message may be excused; `TickMsg`'s excuse | assertion | none | none |
| `cmd/watchpost/messages_test.go` | R8's pin: `TickMsg` for `tickMsg` | setup | — | — |
| the helpers' and the engine's call sites | renamed (A-50) | setup | — | — |

**The inventory's delta.** New: `TestTheConsolesMastheadKeyCoversEverythingItsMastheadShows` and
`TestTheMastheadKeyCoversItsNamedInputs`, the guard's two halves (D-58);
`TestARenameScopedToAPackageMapsThatPackageAlone` (`tools/mutantmap`). No test removed.

**AP-CAPS-01's soft warnings** rise from 1 to 87: the cut files' lines are new to the diff, and the
comments they carry open paragraphs as they always did. The prep moves them unchanged; nothing it
wrote opens a paragraph with AND, BUT or SO.

**The first gate failed** in `make race` on two of `cmd/watchpost`'s tests: the declaration golden
had been re-captured before D-59's rename, and D-59 to D-61 and A-50 named Go files without a line
or symbol. D-59 and D-60 now cite a symbol and a package; D-61's file floor and A-50's files are
claims about whole files, listed in `wholeFileClaims` as D-47's are.

The second gate was green, 5,306 s, and the prep is `5171b0cb`.

## B2's second prep: what the move found the prep had left (D-62, D-63; A-51 to A-53)

**The move, tried and set aside.** Built on `5171b0cb`, the move compiled `frame` and `ttytest` with
no edge back to the root, so the closure's 0 blocked edges hold, and every root reference the
compiler reported was an exported kit name. It also found four things the prep should have done
in place, and the move was set aside, unchanged, for this commit:
- `consoleCursor`'s parameter `frame` shadows the package at the one call it makes to the kit
  (design CK-10 named it for the prep);
- `missEveryMemo` sets two of the kit's unexported fields, which `tests-CK-refs.tsv` lists (rows
  120 and 121) with a zero slot for each. The zero cursor slot is not neutral: it adds 2
  allocations and 48 B to each of the five Observer miss pins, because a real miss keeps the held
  cursor and a zero slot drops it. The zero masthead slot leaves every pin equal (measured with
  each slot alone). Ruled D-62: `Field` writes the one field;
- two of the clip file's three tests call `clipToWidth`, which stays unexported in the kit, and the
  third is a console test (A-51);
- the union check counts the plan's test support as new code once its files stop being
  `_test.go` files: 13 byte-identical declarations. Ruled D-63: the union leaves out the
  test-support packages.

**The instruments, surveyed on the move.** This commit's first gate was stopped at its start
(MAKE_EXIT 143, no run row), because the move, applied in a worktree, showed one more instrument it
depends on: `gcprobe` failed on 92 keys, since its map could rename a key and not move one. A-52's
move map lands here, empty, and passes the moved tree with B2's one line. On the same worktree:
`make graph` is OK, the console's reach 13 to 10 as design CK-9 says; `make p10` has five live
findings, `walkAt` and `bump` (D-23), the kit's density (D-31) and two loops in the new `gcprobe`
code, written since with a bound; there is no P10-09 finding, because the engine's lines arrive
unchanged; and 23 anchors no longer match, 12 by path and 11 by text. Three of the 11 drop the kit
name here (A-53).

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| `tape_memo_test.go` | `missEveryMemo`: the masthead slot zeroed, the cursor un-primed through `Field` (D-62) | setup | — | — |
| `ttytest_field_test.go` | `Field`'s test (D-62) | assertion, new | — | none recorded |
| `broadcaster_heldband_test.go` | the held band's sweep across widths, from the clip file (A-51) | assertion, moved | none | none |
| `cmd/watchpost/union_decls_test.go` | test support left out, held to `lint-imports`' list both ways (D-63) | assertion | none | none |
| `tools/gcprobe/main_test.go` | the tree test reads `moves.tsv`; the move map's test (A-52) | assertion, new | — | none recorded |

The union golden's diff: `consoleCursor` edited, and `apptest`'s `Main` and `setDirs` and
`ttytest`'s `Main` left out. The five miss pins measure 5,525, 11,323, 14,235, 1,138 and 5,526
allocations, each its baseline.

**The inventory's delta.** New: `TestFieldWritesItsFieldAndRefusesWhatItCannotFind` and
`TestAMoveCarriesItsNamesKeysToTheirNewPackage` (`tools/gcprobe`). The held band's sweep keeps its
name and package. No test removed.

`make p10`: 0 live, 0 unmatched, 0 unratified. The new `gcprobe` code first measured 0.98 against
the package's floor of 1.00; the destination's package name is now read from its package clause,
which its files must agree on, rather than taken from the directory's last element.

**This commit's second gate failed** in `make lint` on two new findings, both in its own code:
staticcheck's QF1001 on a digit test in `tools/gcprobe/main.go:qualified`, which guarded nothing,
since a run that starts with a digit is never a declared name, and is gone; and `unused` on the
embedded struct in `Field`'s test, which the test now reads to show the write leaves a promoted
field alone.

The third gate was green, 5,165 s, and the second prep is `774eb4a0`.

## B2's move: the kit and the test support in their packages (D-64 to D-66; A-54)

**The move (A-54).** Eleven files `git mv`'d whole, by script, as the second prep
left them: the kit's six to `modes/tty/internal/frame`, with its masthead test and the two clip
tests; the engine, the helpers and `Field`'s test to `modes/tty/internal/ttytest`. Every other edit
to them is a package clause, an import or a qualifier the compiler asked for, in four rounds; the
headers and the comment pass follow. Each new package's `TestMain` calls `ttytest.Main`, and
CK-7's floor gains both.

**The instruments, each seen failing on a partial move.**
- **The union** (`TestTheTwoTreesDeclarationsAreConserved`) is unchanged by the move: the same
  golden, no re-capture. With `frame.Plural`'s body edited it fails, one declaration new and one
  gone; restored by copy, it passes.
- **`gcprobe`** with B2's line in `06_docs/perf/moves.tsv`: OK, 4,001 escapes in 1,992 keys and
  7,244 inlined calls in 4,571 keys, as at the baseline. Without the line: 91 failures, every kit
  key read as a new escape or a lost inlining.
- **The test inventory's comparison**, on the move's own keys (B0's plain record with the clip
  tests in `frame`): without `06_docs/testinv/accept-B2.tsv`, the two are gone and two are new;
  with it, all 2,006 kept; with it and one of the two removed, that one is lost. The three modes
  run at B2's close.
- **`make graph`**: OK, 11 figures, none above its baseline; the console's reach 13 to 10, as design
  CK-9 says. MS-7's reported figures move one test from C4 to C3,
  `TestTheOriginFollowsTheUIMode`, whose reach into the root loses `NoTransmitter`, now the kit's;
  they are re-measured at B4.4 (D-9).
- **`make mutant-anchors`**: all 1,060 match, after 12 re-points by path and 8 qualifiers (A-53).
- **The first gate on the move was red**, 5,506 s, at `mutant-check`: m241 and m425 left the tree
  uncompilable. Each deletes a line its qualifier made its file's one use of `frame`
  (`map_prop_origin.go`'s `frame.NoTransmitter`, `map_prop_ahead.go`'s `frame.Plural`), so the
  import was left unused; an anchor that matches is not a mutant that compiles. Each now drops the
  import too, asserting first that nothing else in the file names `frame`. In a worktree carrying
  the move, both compile, and each fails its recorded killer on that test's own assertion
  (`TestTheOriginFollowsTheUIMode`, `TestAnHourIsSaidSingly`), which passes on the clean tree.
- **The roots' export list**: `modes/tty` loses the kit's 31 entries, as A-50 counted.

**P10.** `make p10`'s live findings on the move were D-23's two, `walkAt` and `bump`, now rows,
and the kit's density, 0.64 across 33, ratified (D-64). `ttytest`'s row, ratified for `Main`
alone, is rewritten for the engine and the helpers at 0.14 across 14 (D-65). D-15's P10-09 row is
held: the checker raises no P10-09 finding, even for a new production package planted with
`unsafe` and `reflect`, and a row inside the diff that matches nothing fails the gate (D-66).
`make p10`: 0 live, 0 unmatched, 0 unratified, the mirror regenerated at 175 rows.

**The living documents (package map §9).** 0.19.6's as-built page,
`../03-architecture-design/as-built-packages.md`, draws B0–B2, and
`TestTheAsBuiltMapIsKeptInStep` holds it; the atlas reads it. `docs/architecture.md` gains
`frame`'s row, `docs/extending.md` the masthead key's new path, and the three `doc.go` files their
maps. The moved code's bare D-IDs take their releases (A-33); D-112, which 0.16.0's record lacks,
takes 0.16.0, the release that wrote it (A-54).

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| `cmd/watchpost/testmain_test.go` | CK-7's floor gains `frame` and `ttytest` | assertion | none | none recorded |
| `app/maps_test.go` | the as-built check reads a page and a heading per release; 0.19.6's row | assertion | none | none recorded |
| `frame/main_test.go`, `ttytest/main_test.go` | `TestMain` calls `ttytest.Main` | setup, new | — | — |
| `ttytest/helpers.go` | `MastheadMoved`'s message names the kit's `masthead.go` | message | — | — |
| `frame/clip_ansi_test.go`, `broadcaster_heldband_test.go` | each names the other's file in its new place | comment | — | — |
| the moved tests' and the root tests' call sites | qualified (`frame.`, `ttytest.`) | setup | — | — |

**The inventory's delta.** Moved, and accepted: `TestClippingCountsCellsAndNotEscapeBytes` and
`TestClippingNeverLeavesTheStylingOpen`, `modes/tty` to `modes/tty/internal/frame`. No test new;
no test removed.

**AP-CAPS-01's soft warnings** stay at 86, the second prep's count: the moved comments open their
paragraphs as they did, and nothing the move wrote opens one with AND, BUT or SO.

## B2's close: the inventory, the plan's tests held, the sweep's inputs (A-55)

**The inventory (D-36).** `make test-inventory INVENTORY_ACCEPT=06_docs/testinv/accept-B2.tsv`
against B0's baseline: every one of the 2,006 plain, 2,006 property and 2,005 race tests and
subtests kept with its outcome, the two clip tests in `modes/tty/internal/frame` by the accept
file. Four new in each mode: B1.1's, the prep's two masthead guards
(`TestTheConsolesMastheadKeyCoversEverythingItsMastheadShows`,
`TestTheMastheadKeyCoversItsNamedInputs`) and the second prep's
`TestFieldWritesItsFieldAndRefusesWhatItCannotFind`. No test removed.

**The plan's tests held (A-37).** B2 joins `landedBatches`; every test its section names is declared.

**The sweep's inputs (A-55).** `BATCH_BASE=ac68cbe8`, B1's closed commit, which derives 232 mutants,
the same 232 as `76fbbb46`; `SCOPED_PATHS` is `06_docs/mutants/scoped-paths-B2.tsv`, the two files
moved whole since that base; `SCOPED_FUNCS` is `06_docs/perf/renames.tsv`. `moves.tsv` is
`gcprobe`'s map by package, and the sweep reads its map by file or directory prefix.

**B2 in commits.** 5, the plan's 2: the instruments took a commit of their own first (A-49), and the
move's first build found what the prep had left, a second prep (D-62, D-63); the batch map says so.
The rulings before the re-check, `76fbbb46`, are counted with neither batch. B2 closes on this
commit's `make verify-full` and the scoped sweep.

## B2's closing sweep: 232 CAUGHT, 16 reports, the trace widened (D-67, D-68; A-56)

**The sweep** (`make mutant-scoped BATCH_BASE=ac68cbe8 SCOPED_PATHS=06_docs/mutants/scoped-paths-B2.tsv
SCOPED_FUNCS=06_docs/perf/renames.tsv` on `b3187507`, 3,502 s): its baseline green; 232 of 232
CAUGHT, 0 SURVIVED, 0 NO EVIDENCE; 16 reports, so B2 did not close on `b3187507`. By kind:
- **Re-pointed into the kit, 8:** the cursor's seven (m635, m636, m643 to m645, m649, m654) and
  mAG1, each anchor now in `internal/frame` while its old file stays in `modes/tty` with other code
  in it, which a file map cannot name (A-55).
- **The masthead's named inputs, 3:** m744, m745 and m747, caught by
  `TestTheMastheadKeyCoversItsNamedInputs`, the kit's half of the guard the prep split (D-58); the
  move's own check saw each fail that test and no other.
- **The import, 2:** m241 and m425 read "now in -" (A-56, below).
- **The trace, 3:** m684, mAI6 and mAS2 (D-67, below).

**The import (A-56).** The comparator takes a mutant's first changed line as its site, and the move's
fix to m241 and m425 made that line the import it dropped, outside any function. Each now keeps the
name used inside the function, as m134, m143 and m202 do; the comparator's driver puts the sites at
`propOriginNow` and `hoursWords`, and each fails its killer on that test's own assertion.

**The trace (D-67, D-68).** The Router trace caught m684, mAI6 and mAS2 in 0.19.5's record and
catches none of them now. Run alone on each, it fails at `e57afeaf` and passes at `ac68cbe8` and
at HEAD; the change is `8fee8f28`, B0.6's new walks, which never reach mute's last class, a borrowed
epicentre or a window with no bounds. No sweep compared killers in B0. Two walks are added from
the alphabet's own messages, with a Settings pool: Settings from the dashboard, tabbed to
ALERTS - TONE, a class muted and enabled again; Settings from the console, where the station's rows
are. The `M` key that opens the tone rows is not in the alphabet, and the alphabet is never
extended.
- **The golden:** the four walks' 1,200 lines byte-identical; the 600 new ones byte-identical
  recorded at HEAD and at `8fee8f28`, whose production code is `e57afeaf`'s; three plain runs and
  `-race` replay it.
- **Seen failing:** m684, mAI6 and mAS2 each fail it, and it passes on the clean tree. Two floors,
  the tone rows 50 and the station's rows 20, under the 137 and 54 frames the six walks show; with
  the two walks taken out, they fail at 21 and 0.
- **Cost:** the test takes about 12 s, from 10.

**B2 in commits, as it closes.** 6, the plan's 2; the batch map says so. B2 closes on this commit's
`make verify-full` and its scoped sweep.

**B2 closed, 2026-11-23.** `make verify-full` on the closing tree, committed as `9a3baba4`: every gate
green, the whole corpus judged, 6,837 s. The scoped sweep on `9a3baba4`: its baseline green, 232 of
232 CAUGHT, none SURVIVED, none NO EVIDENCE, in 3,086 s. Of the first sweep's 16 reports, the import's
two and the trace's three are gone. The 11 left are the move's: the eight re-points, the cursor's seven
and mAG1, each verdict and killer the record's (D-69), and the masthead's three, now killed by the kit's
half of the guard as D-58 split it, which neither root guard fails on, run by hand (D-70). B3's
instruments-first commit makes the report name the file where the file moved, and the four mutant
headers that still name the root's guard name the kit's. `make p10` on `9a3baba4`: 0 live, 0
unmatched, 0 unratified, the mirror unchanged at 175 rows. Pushed; the remote ref matched the commit.

## Before B3: the sweep's report and the masthead headers (A-57; D-69 to D-71)

**Instruments first: the sweep's report names the files, the headers their killer (A-57; D-69,
D-70).** The comparator reported an anchor by function alone, so B2's eight re-points read "sat in
focusCell, now in focusCell"; where the file is what the maps do not carry, the report now names
both files with the function. `TestAnAnchorMovedToAnotherFileSaysWhichFiles` holds it, seen failing
on the old report. The four masthead mutants' headers name the kit's half of the guard. It lands
before the prep, in a commit of its own.

**The first gate failed** (1,431 s) on `TestTheAsBuiltMapIsKeptInStep`: this section opened as
"## B3", which the test reads as B3 landed while the as-built page draws B0–B2. Until a batch's
move its work is under "Before", and its own heading opens in the move commit with the page
redrawn (D-71).

## Before B3: the console's API in place (A-58; D-72)

**The re-check (step 1), on `9aa93cfe`.** Design CK's facts against the code, by a read of every cited
site and a type-check of the console's files cut away from the root. No root name blocks the move: the
console reaches the root only through `SnapshotMsg` and `RecentSnapshotMsg` (P3) and the Observer's
card window, which the prep cuts out. Changed since the design:
- B2 moved `NoTransmitter` to the kit, so the console exports 39 declarations, not 40, and m241 no
  longer changes in B3;
- the console's two cursor methods sit in the root's cursor file, outside every file the plan names;
- `app/deck_trace_test.go`, written in B0.6, has 7 console sites the design does not list, and
  `app/schedule_test.go` has 12, not 11;
- the console's half of the masthead guard, written in B2 (D-58), reaches the Observer for its
  snapshot (D-72);
- two `router.go` anchors the design does not list lose their text when `Opening` and `Mirror` replace
  the field writes, mAQ1 and m622;
- `Dashboard.heldCursor` shares the console method's name, so a rename line names its receiver;
- `tools/graph`'s console-reach row reads files by the `broadcaster` prefix, which the move empties.

**The card window, cut (A-21).** All of `broadcaster_manage.go`, renamed by `git mv`, and the
Dashboard half of `broadcaster_detail.go` are `card_window.go`. mW6, m695 and mAL2 re-point by path.

**The console's API, in place (design CK-2).** `New(Opening)` replaces `NewBroadcaster` and the
Router's seven field writes; `SetMirror(Mirror)` its five per-update writes, one call where the last
stood, measured in the design to add no copy, escape or stack (CK-3); `SetListenerMuted` its two mute
writes; five getters its field reads; `WithSnapshot` and `WithRecent` the two snapshot arms of the
console's `Update`, the Router handing the snapshots over in `scopedMessage`. Sixteen names exported
by `gopls rename`.
- **A first rename pass renamed the wrong name:** its offsets counted characters where `gopls` counts
  bytes, and the files carry em-dashes, so `indexForSlot`'s offset landed on the type and renamed
  `Broadcaster` to `IndexForSlot` across 41 files. It was renamed back the same way, the diff checked
  to hold only the prep's own edits, and each rename then ran at a byte offset checked against the
  name, with a build after each.
- **A lost inlining, recovered:** `Router.cutBed` was at the inliner's budget at HEAD, cost 80, and
  the getter took it to 83. It changes nothing on the Router, so it returns nothing, at 79. `gcprobe`:
  4,001 escapes in 1,992 keys, none new; 7,268 inlined calls in 4,591 keys, none lost.
- **A P10-01 finding, avoided:** a `toConsole` helper calling `Update` raised one by bare name. The
  switch is in `scopedMessage`'s program-scoped arm, whose ratified row covers it with its reason
  unchanged: `Broadcaster` holds no Router field. `make p10`: 0 live, 0 unmatched, 0 unratified.
- **The clock (D-21):** `Opening.Now` is the Dashboard's; the router trace's console-clock line is
  deleted, and its 1,800-line golden replays byte-identical.

**Mutants.** 18 anchors follow the code: 3 by path to `card_window.go`, 11 by a renamed name, and
mAQ1, m622, m642 and m664 on the Router's new text, each in the function it sat in and each seen
failing its killer. Every mutant applies.

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| 30 test files | `NewBroadcaster()` as `New(Opening{})`, 62 sites | setup | — | — |
| `broadcaster_loaded_test.go`, `_memo_completeness_test.go`, `_pooldata_test.go`, `_pool_test.go` | a snapshot by `WithRecent` or `WithSnapshot`, 10 sites (design CK-1) | setup | — | — |
| `router_trace_test.go` | the console-clock line, deleted (D-21) | setup | the golden | the golden, byte-identical |
| `router_swap_test.go` | the power guard's walk reads the `Power` getter | assertion | none recorded | none recorded |
| `router_lookup_test.go` | the routing reasons say `Scoped` | setup | — | — |
| the renamed names' call sites | renamed (A-58) | setup | — | — |
| `cmd/watchpost/record_test.go` | A-21 names `card_window.go` whole | setup | — | — |
| `cmd/watchpost/testdata/union-decls.txt` | re-captured: 59 new or edited, 48 gone or edited | golden | — | — |
| `06_docs/graph/exported-tty.txt` | 28 console names until the move takes them out, `NewBroadcaster` gone | golden | — | — |

**The inventory's delta.** No test new or removed.

**AP-CAPS-01's soft warnings** rise from 86 to 103: the renames touch the console's test files, whose
comments open paragraphs as they always did. The one such line in the diff is a comment the rename
of `bcScheduledFrom` rewrote in place.

## Before B3: the tests across the console's boundary (A-59; D-52 to D-57, D-72)

**Prep 1's gate.** `make verify` green on its first run, 5,481 s, committed as `0df293cc`: the
mutants' compile check built the 689 of 1,060 mutants its changes reach.

**How the cut was found.** A type-checked walk of the package's tests, the design's landing rule
applied to every test declaration (design CK-5): the root if it reaches a root name, the console if it
reaches only console names, a pure helper with its users. It reproduces the design's twelve split
files exactly, and four whole-console files the design's count predates. Run again over the move's
file set, it reports every reach across the boundary; three are left, each package's `raceEnabled`
pair and the console's `-update-golden` flag, which the move adds (A-33).

**The cut.** Each declaration moves whole, its doc comment with it: the console's halves stay in the
files the move takes, and the root's go to ten files named for what they test. `Scoped` moves beside
the messages it names. The pure data both packages' tests build from moves into `ttytest`, its bodies
unchanged and its names exported; two package variables there are functions (P10-06).

**The rewrites (D-14, D-22).** A root test builds the console through `New`, its messages and
`ScrollQueue`, in `console_fixtures_test.go`, and reads it through its getters or as it is drawn:
- **setup:** sizes by `tea.WindowSizeMsg`, power by `StationMsg`, the pointer by `ScrollQueue`, the
  console's clock as the Dashboard's (D-21). A pointer past the pool's end, which a field write set,
  is reached as production reaches it: the pool empties under it;
- **the same value through the API:** `Width`, `Power`, `LineupSelection`, `render.Opts{ASCII}`'s
  glyphs;
- **drawn (D-22):** the height as the frame's rows; the gain as the number its control draws; the
  bed's refusal on the station row; the mute in the band; Reduce motion in the waiting rows' dots;
  the control row as the frame's line that names Quit;
- **the dark-background flag:** the design's observable, the window's ground, is one colour in every
  theme, the light ground filled from the dark, so a check of it passed against a console that never
  learns the flag. It is read as the UP NEXT box's ground under Synthwave '84, whose two modal grounds
  differ, and seen failing on that console;
- **D-54's bold mark** is matched up to its glyph: the frame re-arms each reset to its base tone, so a
  tint's closing reset is never in it;
- **D-52 and D-53** split as ruled, the console's halves `TestTheAlertBoxsWindowIsTheOneADigitBuilds`
  and `TestAnAlertCardCarriesItsCategorysTint`; **D-57's** whole frame; **D-72's** snapshot.

**The stacking test** compares the frame with the height its fixture sent, not with the frame's own
rows, which a frame cut short would shorten too.

**A-33's console golden.** The UP NEXT box over a composed card, its DATA PULL line stamped
`09/11/26  5:45:00 PM  (2 MIN AGO)`, at 160x74 on standby, plain and in colour; on
`06_docs/testinv/goldens-B3.txt`. The twelve console goldens replay byte-identical.

**The pins.** The console beat reads 377 allocations and 122,176 B on the rebuilt fixture and on
`0df293cc` alike; the console frame 291 and 9,164, a hit and a miss, on both.

**The test support's floor (D-73).** `make p10`'s floor check failed the new fixtures: `ttytest`'s
row, 0.14 over 14 changed functions, measured 0.11 over 27. The fixtures are data builders whose only
checks are their refusals, so the row is amended to 0.11 over 27, as ruled, and no check is added.

**Mutants.** Every mutant whose recorded killer is a test in a file the prep touches, 103 of them, was
applied and run against that killer, each caught: by the same test, or for mAU2 and mW5 by D-52's
console half, which the scoped sweep will report as their killer's new name. Tests no mutant covers
were proved on faults planted by hand and put back by copy: the Router's mute hand-off; the bed
refusal unmirrored, uncleared while the bed carries, uncleared on standby; the console not inheriting
`--ascii`; the dark-background flag never learned; the resize's height lost.

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| the twelve split files | each declaration whole into a file of its package (design CK-5) | setup | — | — |
| `internal/ttytest` | nine fixtures, `CheckGolden`, `GoldenNow`, `ContainsToken`, moved unchanged (A-33) | setup | — | — |
| their callers | the console's helpers as `ttytest`'s | setup | — | — |
| `card_window_open_test.go` | the console by `consoleOf`; the window router's clock the Dashboard's | setup | m755, m756, mAZ1, mAZ2, mCH4, mSC1, mW1, mW2, mW3, mW6 | the same |
| `card_window_open_test.go:TestTheAlertBoxsWayInOpensThePriorityCard` | its same-window half to the console (D-52) | assertion | mAU1, mAU2, mW5 | mAU1; mAU2 and mW5 by `TestTheAlertBoxsWindowIsTheOneADigitBuilds` |
| `card_window_tone_test.go:TestTheAlertWindowWearsTheCardsGround` | its card line to the console (D-53) | assertion | mAV1 | mAV1; the card line by `TestAnAlertCardCarriesItsCategorysTint` |
| `card_window_tone_test.go:TestTheOpenHazardWindowRefreshes` | the console by `consoleOf` | setup | mAV3 | mAV3 |
| `card_window_test.go` | the pointer by `ScrollQueue`; the line-up kept; the row's number cell written out | setup | mAL2, mAL3, mAL4, mAM1, mAO1, mV6 | the same |
| `console_enter_test.go` | the pointer by `ScrollQueue`; past the pool by the pool emptying | setup | mAB4, mAG2, mAG3 | the same |
| `console_fire_test.go` | hot as the drawn bold mark (D-54) | assertion | mAQ1, mAQ2 | the same |
| `console_mute_test.go` | the console by `New` and messages; the Router's hand-off as the band (D-22) | assertion | none recorded | none recorded; a planted fault fails it |
| `router_test.go` | the height as rows, the background as the UP NEXT ground, `--ascii` as the frame (D-57); `Width`, `Power` | assertion | none recorded | none recorded; three planted faults fail it |
| `router_bedplay_test.go` | the refusal on the station row (D-22) | assertion | none recorded | none recorded; three planted faults fail it |
| `router_diagnostics_test.go` | sizes by message; the gain and the control row as drawn; `Width`; the height as sent | setup, assertion | none recorded | none recorded |
| `request_slot_test.go`, `router_refusal_test.go` | power and size by message | setup | mRQ1, mRQ2, mSB1 | the same |
| `scroll_keys_test.go` | the pointer by `LineupSelection` | assertion | m721 | m721 |
| `cursor_test.go` | the glyphs by `render.Opts{ASCII}` | assertion | m641, m642 | the same |
| `motion_test.go` | the frame counter as the waiting rows' dots | assertion | m622, mSC2 | the same |
| `console_units_test.go` | the console's clock the Dashboard's | setup | m661, m662, m664 | the same |
| `memo_completeness_test.go` | the card window's console by `consoleOf` | setup | mV3, mV4 | the same |
| `beat_alloc_test.go` | the console by `loadedConsoleOf` | setup | m713, m737, m742, m752, mAR3, mAR5, mAR6 | the same; 377 and 122,176 B |
| `broadcaster_masthead_memo_test.go` | its snapshot `ttytest.MastheadSnap` (D-72) | setup | none recorded | none recorded |
| `broadcaster_loaded_test.go` | the station from `ttytest.Loaded` | setup | mAN1 | mAN1 |
| `golden_test.go`, `broadcaster_golden_test.go` | the golden check through `ttytest.CheckGolden` | setup | m694 | m694 |
| `broadcaster_golden_test.go` | A-33's DATA PULL golden, two files | golden | — | — |
| `cmd/watchpost/testdata/union-decls.txt` | re-captured | golden | — | — |

**The inventory's delta.** Three tests new, none removed: `TestTheAlertBoxsWindowIsTheOneADigitBuilds`,
`TestAnAlertCardCarriesItsCategorysTint`, `TestTheConsoleGoldenDrawsADataPull`.

**AP-CAPS-01's soft warnings** rise from 103 to 116: thirteen openers moved into new files or files the
prep touches, every one a line that was in the tree before it.

## Before B3: the move map, into a third package and into package scope (A-60)

**Prep 2's gate.** `make verify` green on its first run, 5,790 s, committed as `5060ceb2`.

**What the move showed.** B3's move, applied on `5060ceb2` and set aside for this commit, compiles
and vets on its first round, and `tools/gcprobe`'s tree test fails it on 34 keys, none a new escape:
- `app`'s keys name six of the console's messages as the compiler now writes them, `console.BedMsg`
  where the baseline has `tty.BedMsg`, at the same counts. The move map rewrote the source's own keys
  and the destination's, and no other package's; B2's move had no such key to show it;
- the root's 8 package-scope `[]string` escapes and its one `map[string][]string` are the console's
  once its files move. Which of a package's initializers a move takes, no name says.

**The fix (A-60).** A third package's key names a moved name through the destination, the source's
package name read from its clause; a name that did not move keeps its package. A package-scope
initializer is a key no function owns, counted across both packages as a compiler wrapper is.
`TestAThirdPackageNamesAMovedNameByItsNewPackage` and `TestAPackageScopeEscapeIsCountedAcrossTheMove`
were each seen failing on the map before the fix. On the moved tree, the fixed map compares OK: 4,001
escapes in 1,995 keys, none new, and 7,268 inlined calls in 4,599 keys, none lost; on `5060ceb2`, the
same totals in 1,992 and 4,591 keys.

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| `tools/gcprobe/main_test.go` | the move test's fixture as `movedFixture`, its source package given a file | setup | — | — |
| `tools/gcprobe/main_test.go` | a third package's keys; package scope; a source with no package refused (A-60) | assertion, new | — | none recorded |

**The inventory's delta.** Two tests new, none removed: `TestAThirdPackageNamesAMovedNameByItsNewPackage`
and `TestAPackageScopeEscapeIsCountedAcrossTheMove` (`tools/gcprobe`).

## B3: the console (`modes/tty/console`)

**The re-check (step 1)** is the first prep's, on `9aa93cfe`, in "Before B3: the console's API in
place"; the second prep's walk over the move's file set left three reaches across the boundary, each
the move's to add. On `5060ceb2` the move, applied by a type-checked plan of every use that crosses
it, compiled and vetted on its first round; it found the move map's two missing cases, which landed
first (A-60), and two tests that read a file by its path, which the move fixes.

## B3's move: the console in its package (A-61; D-74)

**The move (A-61).** Fifty-three files `git mv`'d whole, by script: the console's twelve
`broadcaster*.go` files and the 41 test files that hold its tests alone, the 37 `broadcaster*_test.go`
files and four more (`console_finish_test.go`, `heldcount_test.go`, `projection_cache_test.go`,
`station_wording_test.go`), with its fourteen goldens into `modes/tty/console/testdata`. The plan
was computed on the tree before the move, from the type checker's record of every use of a
package-level name: a root file's use of a moving name became `console.Name` (97 sites, 17 of them
non-test), and `app`'s `tty.Name` for a moving name `console.Name` (50 sites, 11 non-test: the six
messages and `MainTrackSlots`). Each edited file gained the import, or swapped it where it named
nothing else of `modes/tty`. No moving file named a root name but the three reaches prep 2 left, so
the console gains its own: `TestMain` through `ttytest.Main`, the `raceEnabled` pair and its
`-update-golden` flag (A-33). The first build and `go vet ./...` were clean; `gofmt` sorted the new
imports into their groups.

**Two tests that read a file by its path,** which no type check sees: `memo_positions_test.go` reads
the console's memo key from `console/broadcaster_memo.go`, and the rail test reads 0.16.0's mock one
directory further up.

**The comment pass (A-33).** The moved code's 405 bare citations take the release that wrote them,
which `06_docs/ids.md` says is the file's history: the first commit, following renames, whose diff
adds the ID, and the first release tag that contains it. 352 resolve to a release whose record has
the row; the 53 others were read one by one:
- 0.16.0's IDs its record never tabled (D-97, D-103, D-110, D-111, D-112, D-116, D-160) take 0.16.0,
  as A-54 ruled D-112, and so do NFR-3 and NFR-7, its requirements' bullets;
- DR-1, DR-7 and PD-3 are 0.14.0's director requirements (`multi-voice-support`);
- D-360, D-362 and D-367 are 0.19.0's, which O2 ruled; their lines were last written in 0.19.3 and
  0.19.6;
- a follow-up takes the release that opened it, the first tag containing the commit that added its
  row: F-30 0.14.0, F-55 0.15.0, F-205 0.18.0, the rest 0.16.0.
355 insertions qualify all 405, and a paragraph they pushed past its own widest line is re-wrapped
to that width, prose only. Comments elsewhere that named the console's files by their old place
name them where they are (`dashboard.go`, `masthead_memo_test.go`, `frame/clip_ansi_test.go`,
`platform/lineup/schedule_index_test.go`).

**The instruments, each seen failing on a partial move.**
- **The union** (`TestTheTwoTreesDeclarationsAreConserved`) is unchanged by the move and the comment
  pass: the same golden, no re-capture. With `Manageable`'s body edited it fails, one declaration new
  and one gone; restored by copy, it passes.
- **`gcprobe`** with B3's line in `06_docs/perf/moves.tsv`: OK, 4,001 escapes in 1,995 keys and
  7,268 inlined calls in 4,599 keys, none new and none lost. Without the line: 393 failures.
- **The test inventory's comparison**, on the move's own keys (B0's plain record with the console's
  tests in `modes/tty/console` and the clip tests in `frame`): without an accept file, 196 are gone;
  with `06_docs/testinv/accept-B3.tsv`, B2's two rows and B3's 194, all 2,006 kept; with
  `TestConsoleGoldens`' row removed, that one is lost.
- **`make graph`**: OK, 11 figures, none above its baseline. The console's reach into `dashboard.go`
  is 0 of 13: the row reads the twelve files where they are now, and their package cannot name the
  root's. Read by the old prefix, the row would count no files and say 0 for that reason instead.
- **`make mutant-anchors`**: 78 of 1,060 failed on the move, 77 by path and mAL2 by its text; all
  match after the re-points.

**Mutants.** 77 re-point by path. Three change their text (A-25): mAL2's anchor and mAB4's inserted
line take the qualifier, and mAN1, which gives the console the Observer's five day cells through the
root's `extRowDays`, writes the 5 that constant holds, since the console cannot name it. Every mutant
on a file the move edited, 97 on the importers and 77 on the console's, was applied and built: the
two that would not build were mAB4 and mAN1, before their re-texting. mAL2, mAB4 and mAN1 are each
caught by their recorded killers.

**P10.** The rail's P10-06 row re-keys to `modes/tty/console/broadcaster_rail.go`. `make p10`'s one
live finding on the move was the console's density, 0.41 across 56 changed functions, which rode the
root's row before the move: ratified at its measured figure (D-74). The root reads 0.51 across 53,
above its floor of 0.42. The seven Router rows and `Dashboard.View`'s still match, and are re-proved
per surface: the four findings left (`Router.update`, `Router.scopedMessage`,
`Router.throughToObserver`, `Dashboard.View`) are the checker matching a method's name, since the
console's `Update` and `View` are now in another package, which cannot import `modes/tty` and so
cannot hold a Router field; the Dashboard's are unchanged. The reasons stand as written. The
console's row first named the root's floor in its narrative, which `p10floor` read as the row's own,
the first figure after FLOOR; it now says "its floor". `make p10`: 0 live, 0 unmatched, 0
unratified, the mirror regenerated.

**The living documents (package map §9).** The console's `doc.go` maps its twelve files; the root's
loses their row. `docs/architecture.md` gains the console's row; `docs/extending.md`,
`docs/where-things-happen.md` (its vocabulary row's `New` among them), `docs/accepted-costs.md` and
`06_docs/follow-ups.md` name the console's files where they are, and 0.16.0's required reading's
D-51 row its `broadcaster_rail.go`. 0.19.6's as-built page draws B0–B3, the console between the root
and the kit and `app` sending it the six messages; the atlas reads it. The plan's citation of
`cardPulledShort` names its file where it is.

**The changed-line table.**

| File | Line | Kind | Mutants before | After |
|---|---|---|---|---|
| `console/main_test.go`, `console/race_test.go`, `console/norace_test.go` | `TestMain` calls `ttytest.Main`; the `-update-golden` flag; the `raceEnabled` pair (A-33) | setup, new | — | — |
| `memo_positions_test.go:TestEveryPositionFieldHasAReason` | the console's memo file by its path | setup | none recorded | none recorded |
| `console/broadcaster_rail_test.go:mockTrackColumns` | 0.16.0's mock one directory further up | setup | none recorded | none recorded |
| `cmd/watchpost/testmain_test.go` | CK-7's floor gains `modes/tty/console` | assertion | none | none recorded |
| `tools/graph/figures.go` | the console's reach reads `modes/tty/console/broadcaster` | assertion | none | none recorded |
| `06_docs/graph/exported-tty.txt` | the root's list loses the console's 39 names and methods | golden | — | — |
| `06_docs/testinv/goldens-B3.txt` | A-33's two goldens in the console's `testdata` | golden list | — | — |
| the root's and `app`'s tests | 119 qualifiers (`console.`) and their imports | setup | — | — |

**The inventory's delta.** Moved, and accepted: the console's 187 baseline tests and their subtests,
194 keys, `modes/tty` to `modes/tty/console`. No test new; no test removed.

**AP-CAPS-01's soft warnings** rise from 116 to 128: the comment pass and the qualifiers touch
paragraphs that open as they did, and every one of the 128 openers is at `5060ceb2`.
