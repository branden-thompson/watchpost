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
