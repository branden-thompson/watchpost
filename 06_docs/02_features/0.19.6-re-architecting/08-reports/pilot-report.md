---
title: "0.19.6 PLAN: the A-7 pilot move, measured"
date: 2026-11-15
phase: PLAN
agent: PL
status: "FILED from PLAN agent PL; its two forks ruled D-31 and D-32. Its logs and the scratch copy stayed in the session scratch directory, written `<scratch>`"
---

# The A-7 pilot move, measured

## Verdict

1. **A cross-package move is judged line by line, not file by file.** When git pairs the old and new
   paths as a rename, and the package clause guarantees it does with an edit, only lines that differ
   from the merge-base are in scope. A byte-identical function body is not judged again. A function
   whose signature the prep commit changed (an export) does count as changed. A file git does not
   pair, and a pure 100 % rename with no hunks, is judged whole.
2. **D-10's premise is false for the tool.** A density finding in `app/internal/run` is **not**
   covered by the `app` row. The harness matches file and symbol exactly. The prefix in
   `scripts/quality/p10-unmatched.sh:113-115` is only that script's own bookkeeping. A thin new
   package fails `make p10` in the batch that creates it, not at BUILD exit. **This needs a ruling.**
3. **The symbol re-key belongs in the prep commit.** If the rename lands without it,
   `TestEveryRatifiedP10RowNamesCodeThatExists` (`cmd/watchpost/docs_test.go:288`) fails at prep,
   so `make verify` goes red. `make p10` passes either way. The path re-key belongs in the move
   commit.
4. **P10-01 flags all three same-name shapes and generic recursion.** The shapes are a constructor
   pair across packages, a method pair across packages, and a method pair within one package.
5. **The move costs nothing measurable.**
   - The forwarder is inlined across the package boundary.
   - Escape decisions at both call sites are the same as before.
   - `go test ./app/...` took 2:19.6 before and 2:13.5 after, one run each, which is within noise.

## Set-up, and two things the brief did not name

- The tree is `<scratch>/pilot/tree`, built with `git -C <repo> archive e57afeaf | tar -x`. Its
  `base` commit is on `main`, and `feature/pilot` holds these commits:
  - `49f1f5b` prep;
  - `117fe1e` move;
  - `dcd45a5` probes.
  
  Two side branches hold one probe each: `probe/density` (`b528e17`) and `probe/identical`
  (`47abdd3`).
- `A2DH ?= a2dh` (`Makefile:548`) resolves on `PATH` to the harness CLI. The same binary ran in the
  copy. Every `make p10` below is `make p10 A2DH=<that binary>`, with `< /dev/null`, under
  plan-common's toolchain `PATH`.
- **The ledger and the configuration are git-ignored** (`.gitignore:15`, `.a2dh*`), so
  `git archive` leaves them out. I copied `.a2dh.yml` and `.a2dh-p10-exemptions.yml` from the
  repository into the copy, reading them only. `make p10` cannot run without them.
- Logs and JSON are in `<scratch>/pilot/`. `showp10.py` prints a `p10.json` file.
  `qualdiff.py` classifies the lines of a move diff.

## M1: does `make p10` count a byte-identical moved function as changed?

**Commands:** `make p10` at `49f1f5b` and at `117fe1e` (`p10-prep.log`, `p10-move.log`), then
`python3 showp10.py dist/p10.json`.

| Commit | `make p10` | Findings in the touched files | P10-02 on `EveryTick` |
|---|---|---|---|
| prep | exit 0, "0 live, 0 unmatched, 0 unratified", 10.9 s | `app/every.go:27` P10-05, exempted by the `app` row, "1.33 < 2.0 across 3 changed function(s)" | **absent** |
| move | exit 0, "0 live, 0 unmatched, 0 unratified", 7.5 s | `app/history.go:174` P10-05, exempted, "1.00 < 2.0 across 2 changed function(s)" | **absent** |

**Why the P10-02 finding is absent.** The loop sits at `app/internal/run/every.go:36`. Git pairs the
file as a rename (`similarity index 91%`), and its only hunks are lines 1, 12 and 27. In the
harness, each rule has a scope kind (`internal/p10/p10.go:192-236`):
- P10-02, -06, -07, -08 and -09 are **line-scoped**.
- P10-01 and -04 are **span-scoped**: a finding is in scope when any line of its function changed.
- P10-05 is **pre-scoped**: it counts the functions that have a changed line.

Line 36 did not change, so the loop is not judged. `EveryTick` still counts for density because the
prep commit changed its signature at line 27. The scan is cumulative against `main`'s merge-base.

**Three controls settle the question.**

| Probe | Command | Output that decides it |
|---|---|---|
| The move with git's rename detection off (a file git does not pair) | `git config diff.renames false; a2dh p10 check --json` at `117fe1e` (`p10-move-norenames.json`) | `app/internal/run/every.go:36 P10-02-BOUNDED-LOOPS … exempted=True`: the whole file is in scope, and **the re-keyed row matches** |
| The same, with the ledger not re-keyed | the same, with `ledger-before-move.yml` (`p10-move-norenames-oldkey.json`) | `every.go:36 P10-02 … exempted=None`, exit 1: a live finding |
| A byte-identical rename in the same package | `git mv app/every.go app/every_moved.go` from `base` (`R100`), then `a2dh p10 check --json` (`p10-identical.json`) | `app/every_moved.go:36 P10-02 … exempted=None`, exit 1, **with rename detection on** |

The last row is the surprise. An `R100` rename has no `@@` hunks, so the hunk parser records
nothing for the file. The `--name-only` backstop then puts the whole file in scope
(`internal/gitscope/gitscope.go:231`, `scopeMissingChangedFiles`). So a file that moves with **no**
edit is judged whole, and a file that moves **with** an edit is judged by line.

**Answer.**

| How the file moves | What `make p10` judges |
|---|---|
| A cross-package move that git pairs | changed lines only. Untouched bodies are not judged; functions renamed in prep are. |
| A same-package rename (`R100`), a split, or a pair below git's similarity threshold | the whole file |

The ledger row has to be re-keyed in every case:
- In the whole-file case, `make p10` fails on the stale key.
- In the line case, `make p10` cannot see the stale key, but `cmd/watchpost`'s row test fails.

**`p10-unmatched.sh`'s rename rule (`:31-36`, `:56-62`, `:76-80`).** It treats the move as one file.
`git diff --name-status -M` gives `R091 app/every.go app/internal/run/every.go`, and a row keyed on
either path is tested against the pair's hunks:
- `p10-unmatched.sh p10-move.json ledger-before-move.yml` gives "every in-scope ledger entry matched
  a finding (162 dormant …)". A row that was never re-keyed reads as dormant and passes.
- Its symbol test (`:79`) searches the `@@` headers for `EveryTick(`. With no Go diff driver in this
  repository (no `.gitattributes`), both hunk headers read `import (`. A hunk that is itself a
  function's signature therefore never names that function.

This is one line for the lane that owns the B0 instruments: the script cannot see a dead row whose
only change is a signature.

**Does the density row (`app`, FLOOR 0.61, `06_docs/p10-ledger.md:122`) cover `app/internal/run` by
prefix? No.**
- **Command:** on `probe/density`, one function with no checks was added to `app/internal/run`, then
  `make p10` was run (`p10-density-probe.log`).
- **Output:** `make` exit 2, and `app/internal/run/every.go:27 P10-05-INVARIANT-DENSITY …
  exempted=None :: invariant density 1.00 < 2.0 across 2 changed function(s)`.
- **Why:** the harness keys a density finding as `filepath.Dir(file)` + `package`
  (`internal/p10/p10.go:99-117`), and `match` compares `e.File == key.File && e.Symbol == key.Symbol`
  (`internal/enforce/exempt.go:98`). The prefix rule exists only in `p10-unmatched.sh:113-115`. That
  script's header says it "mirrors the tool", and it does not.

## M2: same-name pairs under P10-01's bare-name rule

**Command:** `make p10` at `dcd45a5` (`p10-probes.log`, `p10-probes.json`).
`app/internal/pilota/pilota.go` and `app/internal/pilotb/pilotb.go` are in the probe commit.

| Shape | Output | Flagged? |
|---|---|---|
| `pilota.New` calls `pilotb.New` | `pilota.go:7 P10-01 … recursion (direct or via New)` | **yes** |
| `pilota.Adapter.Handle` calls `pilotb.Model.Handle` (across packages) | `pilota.go:13 P10-01 … via Handle` | **yes** |
| `pilotb.T.handle` calls `pilotb.U.handle` (one package) | `pilotb.go:15 P10-01 … via handle` | **yes** |

The analyzer keys its call graph by bare name: `fd.Name.Name` (`p10analyzers/stage2.go:107`) and
`f.Sel.Name` (`:130`). The receiver and the package qualifier are both dropped. Every same-name call
is therefore direct "recursion", which matches `06_docs/code-standards.md:329`.

## M3: generic recursion

`walk[T any](n int) int` calls `walk[T](n-1)` and is flagged at `pilota.go:16`, "P10-01 …
recursion (direct or via walk)". The analyzer unwraps `IndexExpr` and `IndexListExpr`
(`stage2.go:118-125`). D-23's `walkAt` and `bump` will be flagged once they are in non-test files,
so the two rows are needed.

## M4: the qualifier diff

**Command:** `python3 -I qualdiff.py <scratch>/pilot/tree 49f1f5b 117fe1e run`.

| File | Package clause | Import | Qualifier only | Other |
|---|---|---|---|---|
| `app/internal/run/every.go` | 1 | 0 | 0 | 0 |
| `app/schedule.go` | 0 | 1 | 1 | 0 |
| `app/history.go` | 0 | 1 | 1 | 0 |
| `app/schedule_test.go` | 0 | 1 | 2 | 0 |
| `06_docs/p10-ledger.md` | — | — | — | 1 (the re-keyed row) |
| `app/testdata/declset.txt` | — | — | — | 1 (`func EveryTick` removed) |

The Go files have no other changed line, as expected. **Two files outside the brief's list are
needed:**
- `app/schedule_test.go`: the tests of `EveryTick` are callers too (`:134`, `:165`).
- The declaration-set golden: `app/declset_test.go` fails on `removed: [func EveryTick]` until it is
  re-captured with `go test ./app -run DeclarationSet -args -update-declset`. The prep commit needs
  the same step for the rename.

## M5: cross-package inlining of a one-line forwarder

**Command:** `go build -gcflags=-m ./app/internal/pilota` (`inline-pilota.txt`).

**Output:** `pilota.go:25:13: inlining call to pilotb.(*Board).SetMirror`. `Mirrored`, `New` and
`Model.Handle` are inlined across the boundary too.

**The real move.**
- `EveryTick` cannot be inlined before or after the move: "unhandled op DEFER", at both
  `app/every.go:27` and `app/internal/run/every.go:27`.
- At both call sites, the `-m` lines are the same before and after (`m-prep.txt`, `m-move.txt`),
  shifted by one line and four columns. Among them is "func literal does not escape" at
  `schedule.go` and `history.go`.

## M6: the move's cost to the gate

**Command:** `time go test -count=1 ./app/...`, one run each (`t-before.log` at `base`,
`t-after.log` at `117fe1e`).

| | `app` package | Wall |
|---|---|---|
| before | 138.165 s | 2:19.55 |
| after | 132.971 s, plus `app/internal/run [no test files]` | 2:13.50 |

One run each cannot separate a 4 % difference from noise, so no added cost was observed.
**`app/internal/run` has no tests of its own.** Its two tests are
`app/schedule_test.go:128` and `:161`. They use only the standard library and the moved function,
so they can move with it.

## The authoring lint (A-7's other half)

`go run ./tools/authoring -base auto` at `117fe1e` (`authoring-move.log`) printed "authoring: OK
— 1060 Go file(s) … no findings". The moved file cites no bare decision ID, so TG-4's figure is not
exercised by this pilot and needs a file that does. `make lint-authoring` itself was not run,
because the brief did not permit it.

## What this means for the implementation plan

**Batch steps** (`implementation-plan.md`, "Every batch follows the same order"):
- **Step 3 (prep)** gains three items:
  - the ledger's **symbol** re-keys, for every row whose symbol the prep renames;
  - the regenerated mirror;
  - the declaration-set goldens. Without them `go test ./cmd/watchpost` and `go test ./app` fail at
    prep.
- **Step 4 (move)** gains three items:
  - the **path** re-keys;
  - test callers' qualifiers;
  - the declaration-set golden of the parent.

  The moved code's tests should move with it. B5 says nothing on this, while B6 says "and their
  tests".
- **Step 5 (`make p10`)** has to be expected to **fail** at the end of the batch that creates a
  thin package, under the cadence A-7 sets. D-10's "ratified as one list at BUILD exit" cannot hold
  as written. **For B5, an unverified hand count:**
  - `EveryTick` carries 2 checks and `userCacheSubdir` 1 guard (`app/dashboard.go:760`).
  - `lockedCopy` (`app/severe.go:442`), `savePreference` (`:744`), `cacheDir` (`:755`) and
    `requestStats` (`app/stats.go:238`) carry 0.
  - That puts `app/internal/run` near 0.5, below 2.0.
- **A new check for B0's instruments:** list each move commit's `.go` pairs with
  `git diff -M --name-status main`. Any `R100` pair or unpaired `A` file is judged whole by
  `make p10`, so its ledger rows matter in that batch.
- **Same-name calls.** Each one a batch creates is a P10-01 finding when its function has a changed
  line. Examples are a constructor calling a child's `New`, and an adapter calling the model's method
  of the same name. Each one is either renamed or called through a package-level value
  (`code-standards.md:329`), or it becomes a ratified row of the `Router` artefact class.

**D-10's re-key table:**
- `:63`: the symbol re-keys in B5's prep, and the path in B5's move. The reason is unchanged.
- `:122` and `:129`: "its children ride it by prefix until BUILD exit" is false. Each new child
  package whose changed functions measure under 2.0 has no row, and its finding is live.
- **The closing sentence.** It can now read: "A cross-package move that git pairs is judged by
  changed line: a byte-identical body is not judged again, and a function whose signature the prep
  changed is. A same-package rename with no edit, a split, and an unpaired file are judged whole.
  Every row is re-keyed in either case: a stale key fails `make p10` in the second case and
  `cmd/watchpost`'s row test in both."
- **New rows:** add each same-name pair a batch creates, or record the rename that avoids it.

## Choices for the lead

1. **The density of new packages (D-10's premise).**
   - A: each new package's own row, at its measured floor, ratified in the batch that creates it.
   - B: a harness change so that package rows match by prefix, which makes D-10 as written true.
   - C: hold `make p10` red for new packages until BUILD exit and record it per batch.

   **Recommended: A.** It keeps every ratification to the package it was given for, which is TG-5's
   concern. **Strongest counter:** about five ratification rounds instead of one list.
2. **Same-name pairs.** Rename them or use constructor values (no row), or ratify artefact rows.
   **Recommended: rename** where the name is not the role's word (A-14), and a row otherwise.
   **Strongest counter:** a constructor value is a package-level variable, which P10-06 itself
   questions.

## Touches other agents

- **AP:** the package name `run` collides with the local `run` in `startSchedule`
  (`app/schedule.go:141`). The code compiles and vet is clean, but `run.` is shadowed in that
  function.
- **CK:** D-23 is confirmed: generic and plain recursion are both flagged.
- **KB and the B0 instruments:** `p10-unmatched.sh`'s header claim (`:10-12`, `:114`) and its
  hunk-header symbol test.
