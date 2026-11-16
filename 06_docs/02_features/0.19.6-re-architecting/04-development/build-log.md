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
