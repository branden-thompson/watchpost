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
