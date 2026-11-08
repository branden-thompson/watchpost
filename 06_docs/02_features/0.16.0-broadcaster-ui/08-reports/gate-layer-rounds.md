---
title: "0.16.0 — the gate layer's red-team rounds two to nine, and BUILD exit"
date: 2026-11-07
phase: BUILD
---

# The gate layer's red-team rounds, 0.16.0

These are the dispositions of the gate layer's rounds two to nine and of BUILD exit (2026-09-16 and
09-17), as they were recorded in `06_docs/follow-ups.md` at the time. They moved here in 0.19.5
(A-6), because that file carries only open work. The rows still open went with it as follow-ups, and
are named there; everything else below is CLOSED, DECLARED as a ceiling, or the record of a ruling.

## Round two on the gate layer, 2026-09-16 — parsed → executed

A blind adversary defeated the consolidated model six ways (C1–C6) and two Importants. Disposition:

| Finding | Disposition |
|---|---|
| C1 `.IGNORE:` / `MAKEFLAGS += -i` / `SHELL := true`; C2 `\|\| exit 0` and every other word that succeeds; C3 a checker's path inside an echo; C4 `$(VAR):`, `include`, `ifeq` | **CLOSED by execution.** The oracle runs `make <gate>` with every check red; make decides. E1–E13 are specimens. No spelling is enumerated. |
| C5 `needs:` on a never-run job, `on.push.branches`, an empty matrix | **DECLARED CEILING (G2–G4).** No local oracle for GitHub Actions; the CI checks are labelled a ratchet in their own banner. |
| C6 `if : false` | **CLOSED** — G1, a one-character regex fix, and the fourth spelling of the class that defeated three rounds. |
| I1 registry trust — `exists: func() bool { return true }`; the AST scan's escapes; `notATable` unaudited | **CLOSED** — every table declares `absent` and `satisfied` control subjects and the walk proves both functions can return false (FR-11.6); discovery covers non-test files and valueless `var x map[…]`; `notATable` is audited. **Ceiling declared:** a type alias is not seen — no type-checker is loaded. |
| I2 nine attack rows with no specimen | **CLOSED** — Sections E and F execute (68 specimens, 1 skipped by name). A16 stays a real-tree gate. |
| **F-152** *(new, found by the oracle)* `make fmt` passed with gofmt unable to run | **CLOSED** — the recipe reads gofmt's own status. FR-11.3. |
| **F-153** The oracle's verdict depends on the local make; GNU Make 3.81 cannot express `.SHELLFLAGS` | **CLOSED 2026-09-16.** GNU Make 4.4.1 installed (`brew install make`); the oracle now REQUIRES ≥ 3.82 and is COULD-NOT-RUN by name below it, so its verdict is the one CI gives; the requirement is in `docs/extending.md`. And the reviewer's E4 spelling was inert on both makes — probed directly — so E4 now uses `.SHELLFLAGS := -c :`, which works. 34/34 executed specimens pass under 4.4.1. |

## Round three on the gate layer, 2026-09-16 — what execution introduced

A third blind adversary found four Criticals in the executed oracle and named the missing instrument.

| Finding | Disposition |
|---|---|
| C1–C3 a preflight red only in the scratch tree (`test -f go.mod`, `go version`, `git diff --quiet`) masks `\|\| exit 0` behind it; `verify`'s own preflight + `-@` | **CLOSED by the POSITIVE CONTROL (FR-11.6).** Every gate is run green first; red under green is UNJUDGEABLE, by name. `verify` is judged through treelock's delegation with one checker red. H1–H5. |
| C4 four required gates not in `.PHONY`; `touch <gate>` silences them in the real tree today; pattern rules and `.DEFAULT` hide a gate from the database and "missing" read as CI-only | **CLOSED.** The four are in `.PHONY`; `TestNoFileSilencesARequiredGate` (then named for the phony audit) reads make's own database; absence is an error unless DECLARED CI-only. J1–J3. |
| I1 the control proof painted the checker, so a carrier went red for the wrong reason; last-gate-wins | **CLOSED.** Stubs answer by argument; the control status is painted alone; every carrier is tried. K1, K2. |
| I2 a checker with no extension or `.bash` had no stub and no control proof | **CLOSED.** Any `scripts/…` token is a checker. K3. |
| M3 a table chooses its own `absent` control | **CLOSED for `exists`** — the registry generates a nonce (L1). **DECLARED for `stillNeeded`** (L-ceiling). |
| M1 a real tool succeeding on an empty tree refuses a gate that can fail; M2 double-colon overwrite | **DECLARED** — the false-positive direction, loud, never silent. |
| **F-154** *(new)* The adversary's K-ok row was wrong in the LIST: `a; b` discards `a`'s status and the oracle was right to refuse it | **CLOSED** — the row is now K4 CAUGHT, `&&` is K-ok. An attack list can be wrong in the passing direction too. |

## Round four on the gate layer, 2026-09-16 — paint one thing, not the world

| Finding | Disposition |
|---|---|
| C1 red-under-red from ANY stub — `mutant-check`'s discarded verdict and `lint-injector … \|\| true` read as sound in the real tree | **CLOSED by per-key painting.** Each stub a gate reaches is painted red alone; toolchain stubs answer by sub-command. M1–M6. |
| C2 `verify` judged on `invoked[0]`; `verify: lint` + `-@` survived | **CLOSED** — every checker verify reaches through its own delegation, in turn. M7. |
| C3 the `ciOnly` escape — rule deleted, `.DEFAULT` green, one row | **CLOSED by deletion of the hatch.** Absence from the database is an error unconditionally. N1. |
| I1 the phony audit read the gate's name, not the chain | **CLOSED** — it walks prerequisites; a recipe on a non-phony node is the finding. N2. |
| I2 `absentNonce` a package constant | **CLOSED** — fresh random hex per call. N3. |
| MINOR `MAKEFLAGS=i` from a parent make; wrong diagnostic when the go stub never delegates | **CLOSED** — env stripped (N4); red-everywhere-still-green is UNJUDGEABLE by name. |
| **F-155** *(process)* Three reviewers left `zz_review*_test.go` in a shared scratch directory, breaking the package build for the next | **OPEN — a brief rule:** every adversary works in a fresh directory of its own. Added to the brief template. |

## Round five on the gate layer, 2026-09-16 — observe, don't guess

A fifth blind adversary found six Criticals, and every one was `reach()` reading recipe text — the
round-one defect reintroduced in the one function execution had not replaced. The count had risen
3→6, which by the layer's own rule is the signal to stop and look, and the look found one cause.

| Finding | Disposition |
|---|---|
| C1 two `go run ./tools/X` calls collapsed to one `go:run` key (`lint-authoring`'s self-test hid its check's discard) | **CLOSED by recording.** The go stub records `go:run:<package>`, and `ctl:` in front when it sees a self-test flag. O1. |
| C2 `$(A2DH)` never matched the regex — **the shipped `p10` safety check had never been painted red alone** | **CLOSED.** make expands it; the stub records itself. O2. Verified on the real tree: `a2dh` is in `p10`'s recorded reach. |
| C3 `$$(./scripts/x.sh)` — `(` not in the prefix class | **CLOSED.** O3. |
| C4 `$(CURDIR)/scripts/x.sh`, `sh -c "./scripts/x.sh"` | **CLOSED.** O4. |
| C5 `$(MAKE) -s target` — a short flag un-walked the recursion | **CLOSED.** Recursion is real make; the child's stubs record to the same log. O5. |
| C6 a pattern-rule prerequisite escaped both the walk and the reach | **CLOSED.** The reach is recorded through the pattern rule; the silence audit creates a file for every node make reports considering. O6. |
| I1 `MAKEFILES` not stripped — with `.SHELLFLAGS := -ec`, `;` discards turn red and the oracle certifies them (the QUIET direction) | **CLOSED.** `MAKEFILES`, `GNUMAKEFLAGS`, `MAKE`, `MAKEOVERRIDES` stripped; O7 runs E8 under a parent `MAKEFILES` and it must still be CAUGHT. |
| MINOR one-key-red is too strong for a tolerated diagnostic (`go version \|\| true`) | **REFUSED, deliberately (O9).** The relaxation would read text to decide which key is a diagnostic, and `./scripts/x.sh \|\| true` is the same shape. A diagnostic under `\|\| true` in a gate is refused with a message saying to move it out or let it fail. The real Makefile has none. |
| MINOR the phony walk refused a correct order-only directory prerequisite | **CLOSED by construction of the silence audit.** A file where `mkdir -p` needs a directory makes the gate red, which is not silence. O-ok. |
| **O10 — found by the first recorded run of the shipped tree, not by an adversary:** `release-matrix`'s `(command -v sha256sum && sha256sum … \|\| shasum …)` — `A && B \|\| C` runs C when B fails, so a failed `sha256sum` fell through to a successful `shasum` | **FIXED** as an `if`. `sha256sum` was never on the regex oracle's list, so it was never painted; the record painted it on its first run. |
| the tool list | **DECLARED.** The one enumerated list left in the executed half; a real command that exits 0 on an empty tree under `\|\| true` is not judged. |

## Round six on the gate layer, 2026-09-16 — the tree is the tree

Twelve attacks, nine survived, one cause: the scratch tree was empty. The oracle now runs in a clone
of the repository with stubs by PATH alone.

| Finding | Disposition |
|---|---|
| A1 `git diff --quiet HEAD -- '*.go' && exit 0` — runs the checker in scratch, skips it in a clean clone | **CLOSED.** The scratch IS a clean repository. P1. |
| A2 two `go test` calls share `go:test`; a `\|\| echo` on the first is carried by the second | **CLOSED.** Every invocation is a key (`go:test#1`). P2, P11. |
| A3 `go build -o dist/x; dist/x \|\| true` — a compiled checker no list names; `go:build` carries the gate | **CLOSED.** The `go build` stub writes a recording stub at `-o`. P3. |
| A5 FALSE POSITIVE — `find \| xargs gofmt` refused on an empty tree | **CLOSED** by the tree (P10-ok) — **and the reviewer's spelling was itself the F-152 defect**: `bad=$$(…); test -z` throws the status away. Kept as P10, CAUGHT. |
| A6 a non-phony gate depending on `go.mod`, silenced by a file only where `go.mod` exists | **CLOSED.** P4. |
| A10 `verify-gates FAST=1` with `lint` skipping under it — three text layers passed it | **CLOSED.** `verify`'s record must contain every non-CI-only gate's own record. P6. |
| A11 `ifeq (,$(findstring n,$(MAKEFLAGS)))` hiding a node from the `-n` walk | **CLOSED.** Nodes come from the real green run. P5. |
| A7 `cd scripts && ./x.sh` — the list said red-under-green; it is recorded | **CORRECTED** in the list; P7/P7-ok. |
| A8 stub detection (`head -c 2 $(command -v go)`), A9 toolchain by absolute path, A12 `env -i` | **DECLARED** — a recipe written to escape the oracle escapes it. Printed on every passing run. |
| MINOR `make -n mutant-check` starts the sweep (the line carries `$(MAKE)`) | **KNOWN**, documented at the recipe since 0.15.0; the oracle no longer uses `-n`. |
| FR-11.2 floors `< 5`, `< 3` | **CLOSED** — `== 0`. |
| **Tree change:** 20 scripts' shebangs → `#!/usr/bin/env sh` / `env expect` | The oracle refuses an executable under `scripts/` without an env shebang: the kernel would run the interpreter by absolute path and the script for real, unrecorded. |

## Round seven on the gate layer, 2026-09-16 — the tree as CI has it, and the joints

| Finding | Disposition |
|---|---|
| C1 the clone re-attaches a detached HEAD, keeps tags CI lacks, and rsync carries git-ignored files — `test … != HEAD \|\| exit 0`, `git describe --tags \|\| exit 0`, `test -f AGENTS.md \|\| exit 0` all certified locally and skip in CI | **CLOSED.** HEAD detached, tags deleted, only tracked + unignored files copied (pure Go, no rsync). Q1–Q3. |
| C2 verify coverage compared keys; a variable skipping one of two same-key calls shipped | **CLOSED** — by count. Q4. |
| I1 `tr / _` not injective | **CLOSED** — `%2F`. Q5. |
| I2 `go run` keyed by `$2` (`-tags`), module-path packages, `built:` outside the control obligation | **CLOSED.** Q6. |
| I3 `\b` admits `python3.12` | **CLOSED** — anchored, derived. Q7. |
| I4 a `$(MAKE)` hop with hidden output drops its node from the audit | **CLOSED** — non-phony database rules join the all-at-once set. Q8. |
| M `go build -o out/` directory form | **CLOSED.** Q9 — and found SURVIVED once during the build, fixed before the list was trusted. |
| M FR-11.5 gaps: `ORACLE_*` unnamed, only one assertion printed a ceiling, `g` shadowed | **CLOSED.** |
| M false positives: symlinked dir under scripts/, `bash -c` pipefail, `go test` of a tool as its control | **CLOSED** (dir followed and skipped; `-c` runs the real shell; the third is a design choice, now said in the message). Q11-ok, Q12-ok. |
| M ordinal race (not reproduced) | **CLOSED** by construction — lock. Q13. |
| Q10 `ORACLE_LOG=/dev/null … \|\| true` | **DECLARED**, named in the ceiling sentence. |

## Round eight on the gate layer, 2026-09-17 — the threat model, and the instrument in Go

The eighth adversary's count ROSE (2 → 4 Critical). The look found two classes the lists had never
separated: DRIFT (a spelling an author could plausibly write) and EVASION (a recipe written knowing
the oracle exists). The count rose entirely in the second. HUM LEAD ruling: the oracle enforces
against drift; evasion is a review lens (`red-team-brief.md`, question 10). Two standing rules
came with it: **everything is Go unless absolutely necessary, and shell needs a STOP and an
explanation**; **test and tooling code is a product and gets shipped-code rigor.** Both are
mechanised, because a rule an agent has to remember is a rule some session skips.

| Finding | Disposition |
|---|---|
| C2 `bash -ec '…'` — `-c` honoured only as `$1`; the command string became the key | **CLOSED.** The stubs are Go: any `c` in the leading flags runs the real shell; a key is a file that exists. R1, R1-ok, unit-tested. |
| I1 `-o=path` unparsed; `go test -c` not a build | **CLOSED.** R2 ×2, unit-tested. |
| C3b `.x-run:` dropped as a dot-name | **CLOSED** — only GNU make's own special targets are dropped. R3. |
| C3c a stamp the recipe refuses masks the all-at-once set | **CLOSED** — database-only nodes are tried singly; the all-set is the harmless ones; a set that still breaks is UNJUDGEABLE. R8. |
| I5 the control obligation was a prefix list | **CLOSED** — every local `go run ./…` and every `built:` key is a checker; a third-party `go run host.tld/…` tool is declared trusted. R4 ×2. |
| M1 FALSE POSITIVE `$(shell go env)` at parse time | **CLOSED** — the database run's record is subtracted; and the database run now asks for a goal that does not exist, because `make -pn` makes the default goal under `-n` and `$(MAKE)` lines run even then. R5-ok. |
| M2 FALSE POSITIVE `./tools/x/` vs `./tools/x` | **CLOSED** — cleaned. R6-ok. |
| I4 `macos-latest` ships make 3.81; the oracle FAILS on that runner | **CLOSED** — ci.yml installs GNU make and puts gnubin first on macOS. R7. |
| M3 the ordinal under real concurrency | **DECLARED** in the ceiling. |
| C1 sourcing, C3a `-f`/`-C`, C4 a compensating invocation, I2 `export PATH`, I3 `GITHUB_ACTIONS` / untracked files | **EVASION — declared** in the ceiling sentence and carried by the brief's question 10. |
| ~70 lines of shell inside the oracle | **GONE.** `tools/gateoracle/` is a package: the stub is a small Go program built once per test process, uninstrumented, every decision a unit-tested function (`stub_test.go`), the four assertions a library, `cmd/watchpost` a thin caller. Found while building: one `itoa` duplicated `app/inject_debug.go`'s — `make dupes` caught it; `strconv.Itoa` now. |
| Found by `make verify`, not an adversary: the first Go stub was the TEST BINARY re-exec'd by role, and under the race detector every one of the thousands of execs started an instrumented process — `cmd/watchpost` went from 70s to the 10-minute package timeout and `race` was RED | **CLOSED.** The stub is `tools/gateoracle/stub`, a small main built once per test process, uninstrumented: 48s plain, 126s under `-race` (was >600s); no `TestMain` hook in any consumer. The notification for that verify said exit 0; the log said `Error 1`. |
| **F-156** *(new)* Port the 23 non-Go sources under `scripts/` to `tools/` | **OPEN.** Each has a ratified row in the shell ledger (`cmd/watchpost/shell_test.go`) saying why it is still shell; a new script needs a row, and a row is a ruling. The five `expect` scripts are the hard ones (a Go pty driver is the port). |
| **F-157** *(new, REVIEW 2026-09-17)* **Tier three never reaches a slot.** Measured around the real index: 3 mi → a pool of ONE place, 10 mi → 7, 50 mi → 25 — every one a city-table entry, because tier two fills `locations.PoolCap` before the zip-code places the pool's own header calls "the big value" are considered. A short-range station's listeners stand in the small places. | **OPEN — 0.16.5.** The console now says how many places are in reach at the station's radius (R4); the pool's tiering is the 0.16.5 change, and FR-8.6's text records the gap. | `domains/locations/pool.go:71-97`; `app/pool.go:399-403` | business-quality axis, REVIEW |
| **F-164** *(new, VALIDATE 2026-09-18)* **What a relay falling through to synth does to the operator's cut-over — a ruling.** With the bed carrying the programme (`CutOver{ToBed:true}`) and the relay then falling through to the synthesized broadcast (`Tuned{Live:false}`), the code releases the cut-over and the main track resumes (`carries` is relay-only from birth, D-33; the relay the operator chose is gone). The blind reviewer of the bed fix constructed the alternative — the cut-over standing — and observed the station silent with the main track paused for as long as the operator does not act: no dwell elapses under `carries`, `Ended` emits nothing. | **CLOSED 2026-09-18 — RULED (a), HUM LEAD: "as it is today."** A fall-through to synth releases the cut-over and the main track resumes; pinned by `TestAFallThroughToSynthReleasesTheCutOver`. Recorded as D-159. (b) and (c) were the alternatives presented with the reviewer's observed silent-station sequence. | `platform/lineup/bed.go` (`onTuned`), `air.go` (`advancesMonitor`) | blind reviewer of the bed fix, VALIDATE |
| **F-163** *(new, REVIEW 2026-09-17, R2 round two)* **Two rulings owed on the fault band.** (i) An escalation with `Run: 0` — the bed that could not be tuned (`platform/lineup/bed.go`, `Escalate{Reason}` with no count) — is delivered by the executor and DROPPED by the console, whose band shows only a run (`modes/tty/broadcaster.go`, `faultNotice`); the tune-failure escalation reaches nobody. (ii) Whether a routed decline BREAKS a run: today fault/decline/fault/decline/fault escalates at run=3 ("in a row" counts faults only). (iii) The same drop reaches a compose fault that EMPTIES a one-card schedule while on standby: off air the run is not counted, `stopped()` is true, the escalation carries `Run: 0` and the console shows nothing (reviewer's S7; rare — one in-flight compose across the transition). | **CLOSED 2026-09-17 (HUM LEAD: "recommendations approved").** (i) and (iii): the band shows a `Run: 0` escalation by its reason alone — "STATION FAULT — <reason>" (`TestAFaultWithNoRunShowsItsReasonAlone`); the zero message stays the clear. (ii) RULED: a decline does not break a run — "in a row" counts faults, and the code holds it (`TestAFaultRunOnAToppedOffStationEscalatesAndSitsOut`'s control). | `platform/lineup/bed.go`, `platform/lineup/fault.go`, `modes/tty/broadcaster.go` | blind reviewer of R2 round two (T1, S3, S7); the cheapest close for (i) and (iii) is the band showing a `Run: 0` escalation by its reason alone |
| **F-160** *(new, REVIEW 2026-09-17, ruling 9)* **FR-9.2 is not pinned as written:** the tower's coordinate pair as a typed value with ONE owner. The two tests that cite FR-9.2 cite an earlier release's FR-9.2 (a collision the BUILD-exit table counted as a trace). | **OPEN → 0.16.5.** A test that changes the owner's pair and finds every consumer follow, or refuses a second owner (the shape FR-9.1 asks for). Carried into REVIEW exit as recorded scope. | `platform/config` (`Station()`), `app/radio.go`, `modes/tty/masthead` | derived traceability, REVIEW |
| **F-161** *(new, REVIEW 2026-09-17, ruling 9)* **FR-3.6 is OPEN:** a card dropped at the fifteen-minute staleness bound tells the LISTENER (`dropStale` → `readInstead`), and the operator-facing notice that *names the dropped card* does not exist. | **OPEN → 0.16.5.** A console notice on the drop, naming the card; a test that drops a held card and reads the name off the console. Carried into REVIEW exit as recorded scope. | `platform/lineup/director.go` (`dropStale`), `modes/tty/broadcaster.go` | red team BUILD, ruling 9 |
| **F-162** *(new, REVIEW 2026-09-17, ruling 9)* **FR-8.8 is OPEN:** the console shows one per-CARD data-pull stamp, coloured by D-137's ladder; the per-KIND last-arrival that *moves when data arrives and visibly ages when it does not* is not built. | **OPEN → 0.16.5.** A freshness row per kind (obs, alerts, products…) fed by the pipeline's arrival times; a test that advances the clock and reads the ageing. Carried into REVIEW exit as recorded scope. | `modes/tty/broadcaster_cardrow.go`, `app/pipelines.go` | red team BUILD, ruling 9 |
| **F-159** *(new, REVIEW 2026-09-17, R2 round two)* **The fault run has two authors and one race.** The Director resets the run on ANY `Finished` (a rail's, or a ghost's for a card not held) while the executor clears the console band only on a MAIN-TRACK finish; and `Escalate` holds no resource, so it runs on a free goroutine that can draw "3 CARD(S) FAILED" after the clear for a run the Director has just zeroed. Both are Minor: the band self-corrects on the next finish. | **OPEN → 0.16.5.** The reviewer's design: carry the fault (run, reason) on `Publish`, so the console has one source and both the executor's clear and the fault message vanish. Its own red team. | `platform/lineup/director.go` (onFinished), `app/executors.go` (Escalate, the clear), `app/pump.go` | blind reviewer of R2, F6/F7 |
| **F-158** *(new, REVIEW 2026-09-17)* **The P3(d) ruling is still owed:** `app/maintrack.go` is the merge's temporary switch (`WATCHPOST_MAINTRACK`, default off), so relay-failure `NeedsRead` never reaches the Director in production; two comments said it "retires at P3(d)" while the disposition sheet says `startSynth` is Instruct — it stays. | **OPEN — a ruling.** HUM LEAD 2026-09-17 (ruling 4): Instruct is current; the switch ships dormant and the comments now say so. The ruling owed is whether the switch and the direct path retire together, and when. **The test to write before the direct path retires** (business reviewer, REVIEW 2026-09-17, Finding 4): a PRESSED cut-back (`CutOver{ToBed:false}`) while a hazard is reading on the rail makes the Director emit `Restore` (`platform/lineup/bed.go`) — the dip is lifted under a tornado read at the Director level, RS-2's exact shape — and audio survives today only because the arbiter's `releaseBed` re-settles and `mc.giveWay()` re-dips (`app/arbiter.go`). Reproduced by the reviewer in a clone (duck=0, restore=1 with the rail holding the hazard). FR-4.3's traceability row names this. | `app/maintrack.go`; `app/radio.go:623` | code-quality axis, REVIEW |
| **AP-SHELL-01** *(new rule)* | A shebang-plus-body string literal or `exec.Command("sh"\|"bash", …, "-c", …)` in any Go file is a lint finding with no exemption marker. Self-tested in both directions; it found four in the tree on its first run (the oracle's own fixture bodies) and they are gone. |

## Round nine on the gate layer, 2026-09-17 — absence must be loud (the first drift-briefed round)

1 Critical / 4 Important / 6 Minor under the drift threat model; three evasion lines reported
separately and not counted. Criticals across nine rounds: 6 → 4 → 3 → 6 → 6 → 5 → 2 → 4 → 1.

| Finding | Disposition |
|---|---|
| F1 (Critical) a tool's ABSENCE is never exercised — the stub is always on PATH, so `command -v a2dh \|\| exit 0` takes the live branch in the scratch and the skip branch on a machine without the tool | **CLOSED — the fourth verdict.** For every stubbed tool a gate recorded (not `go`, not what the OS ships), the gate is run with that tool absent from PATH and must go red OR record a fallback the green run did not (`shasum` for `sha256sum`, the shipped checksum line). S1 ×3, S1-ok ×3. **Corrected while building:** the reviewer's "the shipped p10 line with `exit 0` passes every gate" held only because absence was never simulated — `\|\| { …; exit 0; }` exits that line's shell and the next line gets 127. The oracle was right; the specimen moved to the single-shell form that actually skips. |
| F2 (Important) a refused invocation under `\|\| true` was unseen; executables outside `scripts/` were outside the shebang rule | **CLOSED.** A refusal is recorded as `unjudged:<role>` and any `unjudged:` in a green reach is UNJUDGEABLE; the shebang rule covers every tracked executable. S2 ×2. Five record artefacts under `06_docs/` lost their executable bit (a record is not a program); `06_docs/mutants/run.sh` and `tools/geotrim/refresh.sh` got env shebangs and ledger rows. |
| F3 (Important) FALSE POSITIVE — a parse-time `$(shell go env)` subtracted once where `$(MAKE)` hops re-parse | **CLOSED** — subtracted once per makefile read the run reports. S3-ok. |
| F4 (Important) the absent-node set was read after the green run mutated the tree (`touch $@`) | **CLOSED** — the scratch is `git reset --hard && git clean -fdx` before every run, and the silence audit resets before creating its files. S4. **Found while building:** the first placement of the reset deleted the audit's own files and every silence specimen SURVIVED; moved before the files, not after. |
| F5 (Important) the verdict depended on the developer's uncommitted files | **CLOSED** — the overlaid working tree is committed in the scratch. `TestADirtySourceTreeIsJudgedAsCIWouldHaveIt`. |
| F6 `go run .` not a checker; F7 `go test -c -o dir/` and an absolute `-o` dir; F8 the ceiling printed a function pointer; F9/F10/A10 declared; F11 `LC_ALL=C` | **CLOSED** (S6, S7-ok, S8, S9) or **DECLARED**. F8 is pinned by `TestCeilingNamesTheTools`. **Found while building:** the absolute `-o` directory was taken as the package — anything starting with `/` looked package-shaped; the value after `-o` is never a package. |
| S10 the instrument about itself | `builtKey`, `writeBuiltStub`, `realCommand`, `moduleOf`, `envShebang`, `parseDatabase`, `ParseRequired`, `localPackage` each have a unit test; O8, Q7, H2 renamed for the property that catches them. |

## BUILD exit, 2026-09-17 — what the day closed, and what it owes

Batch A (product): F-149, F-150, F-151 — three tests with watched reds. F-142: the roster names tests
that exist (a gate now) and every gate names a failure it watched (eight plants). Batches B and C
(records): F-143, F-136, F-145, F-146, F-147, F-123, F-144, F-141, F-148. Deferred to 0.16.5 with a
row each: F-133, F-137, F-139, F-156. **The three rulings owed were ruled as recommended (HUM LEAD, 2026-09-17):** F-140 closed with a Go existence gate; FR-5.4 holds by `TestTheStationToggleAsksTheDirectorForTheOtherState`; FR-8.6 was amended to the pool's real rule. No requirement is untraced.

**A second catch, by a gate of this release's own:** the F-143 edit to `docs/accepted-costs.md`
replaced an EMPTY slice (the end marker sat before the heading in that file), which inserted the new
paragraph between every character — a 15 MB Markdown file, committed in `6b1b621`. `TestNoTrackedBinaries`
refused it inside the sweep, which read the two known survivors as INVALID; the sweep was stopped at
87, the file restored and re-edited on the right slice (`e956747`), and the sweep re-run to completion.
The gate written on 2026-09-16 against tracked binaries caught its author's own commit the next day.

**A process catch of this exit's own:** the mutant sweep refuses a dirty tree, and the first run was
started while records were still being edited — 372 of 379 mutants SKIPPED, a void run whose promoted
log was restored, not committed. The rule "never work during verify" is the rule "never work during a
run", and it now says so.
