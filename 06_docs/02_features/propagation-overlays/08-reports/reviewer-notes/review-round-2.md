---
title: "0.19.0 REVIEW exit red team, round 2 - reviewer notes"
date: 2026-10-25
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "Seven fresh blind reviewers (D-269), each report filed in full as its final message gave it; only each reviewer's scratch folder is shortened to <scratch>. Every finding cites the file and line it was reproduced against."
---

# REVIEW exit - red team, round 2: the reviewers' reports

Up: [the report](../red-team-review.md)

Prefixes: R2C (Code Quality), R2H (Project Hygiene), R2D (Docs Quality), R2B (Business Quality), R2L (the REVIEW lens), R2I (InfoSec), R2P (Performance). The brief, the same for each but its lens, verified round 1's findings, hunted what round 1 missed and attacked the code the remediation brought in; the network dropped twice during the pass, and each reviewer was resumed where it stopped.

## Code Quality (R2C)

#### Code Quality, round 2 (R2C): REVIEW exit red team, 0.19.0

**Verdict: exit REVIEW.** I found no Critical. I found one Important, R2C-1. The RC-4 fix holds in the code, but the guard it depends on, the build counter `built`, has no test of its own, and the mutant record says it does. Everything else is Minor.

##### Tree state and runs

- **Tips:** watchpost 5b71df2c, go-ionomaps 8d84569, go-tuiMaps 73c602d.
- **`git status --short` at the start:**
  - watchpost: ` M .../02-analysis/rulings.md`
  - go-tuimaps: ` M 06_docs/gate-runs.md`
  - go-ionomaps: clean
- **During my review, go-ionomaps gained ` M 06_docs/gate-runs.md`.** The new row (16:09:01Z) came from someone else's in-tree `scripts/gate --quick`. I ran my gates on copies under my scratch directory, and my run's row (16:13:10Z) is in the copy only.
- **I edited no tree.** Every finding was reproduced against those tips in those states. I applied each mutant with `go test -overlay` from scratch copies.
- **watchpost runs, all exit 0:**
  - `make lint` (6 baselined), `make lint-authoring` (985 files), `make p10` (0 live; p10floor reports 23 rows at or over their floors), `make dupes` (2 groups, both ratified), `make mutant-anchors` (795), `go vet ./...`.
  - `go test -race` on: modes/tty, app, platform/{history,lineup,httpx}, domains/{propagation,locations/...}, and tools/{perfsum,runlog,finaltag,p10floor,reachscore,prehook}.
  - The record and gate tests in cmd/watchpost.
- **go-ionomaps `scripts/gate --quick` (copy): green.** That includes the tools/tables legs and P10.
- **go-tuiMaps `scripts/gate --quick` (copy): FAILED, on the P10 leg only.** a2dh reported "govulncheck FAILED — the run cannot certify this scope". This looks environmental, but I could not confirm that. Its `go vet ./...` and `go test ./...` at the tip are green.

##### Round 1's findings, verified against the code

| ID | Status | Evidence |
|---|---|---|
| RC-1 | FIXED | `tools/perfsum/main.go:265-273` (unread is NaN); `g1.go:204-208` (the arm is refused on unread RSS); `:214-218` (WARN/FAILED counted); `:143-145` (a short run is missed); `workload.sh:145-152`. Side effect: `maxOf` of an all-unread gauge prints `-Inf` in the non-G1 summaries (`main.go:448-454`). |
| RC-2 | RULED (D-254), built | `synth_test.go:181-194`, `Makefile:113`, `ci.yml:57` |
| RC-3 | PARTIAL | The tree is now read before and after the run (`runlog/main.go:43-48`, `:91-95`). Still open: a failure to record exits with the gates' code (`:51-54`). If the end-of-run tree read fails, the changed-tree check is skipped and a green run exits 0 with no record line. A-84(4) does not cover this. |
| RC-4 | FIXED in code | `map_prop_ahead.go:56,67`. Its test does not pin `built`: see R2C-1. |
| RC-5 | RULED (A-89 under D-267) | A-89's own claim is false: see R2C-2. |
| RC-6 | RULED (D-264), built | `tools/p10floor`. A gap remains: see R2C-4. |
| RC-7 | PARTIAL | `propagationHostList`, `propagationSources` and runlog's padding are gone. `ackDue`, `LiveChildren` and `Settle` are kept with reasons (A-84(12)). `hasKey` (`map_prop_place.go:191-199`, still misnamed) and `stillOrphan` (`children.go:171-175`) remain with no recorded reason, yet `red-team-review.md` says "checked as fixed". |
| RC-8 | FIXED | `record_test.go:73-75`, `children.go:130-133`, `app/mapprop.go:58-63` |
| RC-9 | PARTIAL | reachscore fixed (`report.go:115-119`). The run-selector check still ignores build tags (`cmd/watchpost/gates_test.go:719-721`, "Build tags are not consulted"), so the `make property` case passes as round 1 described. |
| RC-10 | FIXED | history.go is down to 817 lines plus seven files. The arms are written out: `map_prop_chart.go:101-112`, `map_prop_freq.go:84-97`, `map_prop_centre.go:76-85`. |
| RC-11 | FIXED | go-ionomaps `update.go:145,217-236`. Mutants killed: `short` removed, and D-RAP's age dropped. |
| RC-12 | FIXED / RULED | The floor toolchain is pinned unconditionally (`scripts/gate:89`); D-268 added the tools/tables root (`.a2dh.yml`). Each run still appends to tracked `06_docs/gate-runs.md` (`:216-224`), and that is how go-ionomaps got dirtied during this review. |
| RC-13 | PARTIAL | Fixed: `reproduce_test.go:77-82`, `p10_mirror_test.go:168-169`, `answers.go:247-250`, the readings' order. Still open: the mirror test is still `t.Skip` (reported as a pass) where the ledger is absent (`p10_mirror_test.go:108-110`). |
| RC-14 | FIXED | go-tuiMaps `describe.go:131-145`. The mutant that removes `turnPassed` is killed. |
| RC-15 | FIXED for depth | `look.go:239` (mutant killed). `Ground`'s re-check is unpinned: see R2C-6. |
| RC-16 | RULED (D-258, D-263), built | `platform/astro/turn.go` deleted; `map_prop_words.go:124` uses `DaylightAt` |
| RC-17 | FIXED | `internal/colour/blend.go:243-265` with `TestEveryPresetHasItsClassTokens` |

##### 1. Understandability
- **R2C-9 (Minor), go-ionomaps:** `Path` calls `hourFor` twice and throws the second `Hour` away (`answers.go:247-250`). A function named for its result is being called for its check. Simplify/Delete: Y. Action: a `checkPlaces(places, max)` helper, called once for the origin and once for the destinations.

##### 2. Maintainability
- **R2C-2 (Minor):** the comment and A-89(1) say "a new option is added to the update's options alone" (`map_prop_update.go:73-77`). That is false.
  - `madeWith` lists the three fields by hand (`:87-89`).
  - A new `UpdateOptions` field that is left out of it never compares equal to `propOptions()`, so `armPropOptions` (`:104`) asks again on every arm, and `sameSnapshot` ignores the field.
  - Simplify/Delete: N. Action: correct the comment, or add a test tying `madeWith` to `UpdateOptions`' fields.
- **R2C-3 (Minor), go-ionomaps:** A-29(1) says "one rule", but each freshness rule is written two or three times:
  - the grid: `update.go:227,277,406`
  - D-RAP: `:228,450`
  - scales: `:229,390`
  - Simplify/Delete: Y. Action: `gridUsable`, `drapUsable` and `scalesUsable`, written as `solarUsable` is (`:211-215`).
- **Minor duplicate:** reachscore's "columns answered different numbers of scenarios" test is written twice (`report.go:115-116`, `:127-130`). Simplify/Delete: Y.

##### 3. Complexity
- Nothing beyond R2C-1's redundancy, below.

##### 4. Necessity
- `hasKey` and `stillOrphan` (RC-7) remain.
- In production `built` alone tells the snapshots apart: `propBuilt++` runs on every `!same` (`map_prop_update.go:155-157`), and `!same` means the moment or the options differ. So `propMade`'s `computed` and `opts` are belt and braces. The tests pin those two fields and leave `built` unpinned (R2C-1).

##### 5. Name semanticism
- **Minor:**
  - `propMadeOf(s)` says "a snapshot as kept" but mixes `s`'s moment and noise with the held snapshot's build and correction (`map_prop_update.go:93-95`; the chart passes it an hour-ahead snapshot, `map_prop_chart.go:92`).
  - `hasKey` is unchanged.
  - `hourFor` (R2C-9).

##### 6. Code documentation
- **R2C-5 (Minor), go-ionomaps:** the doc of `Inputs.Unread` (`snapshot.go:134`) says "files this update asked for and did not read". After `short` that is untrue.
  - It names files never asked for: with no usable mean, no grid, D-RAP or scales is fetched (`update.go:130-139`).
  - It names files that were read but are past their age.
  - The names are also an untyped contract across the two repositories: watchpost matches `"solar"` literally (`map_prop_update.go:370`).
  - Simplify/Delete: N. Action: fix the doc, and export the four names as constants.
- **Minor, go-tuiMaps:** the RD-6 alias comments list the internal structs' fields by hand (`overlays.go:39-61`, `describe.go:16-21`). I compared them and they match today, but nothing tests them, so they will drift.
- lint-authoring is green, and I found no history comment.

##### 7. Testability
- **R2C-1 (Important): the guard the RC-4 fix depends on is untested.**
  - **Mutant 1:** `built: 0` in `propMadeOf` (`map_prop_update.go:94`) survives the whole modes/tty package.
  - **Mutant 2:** the corpus mutant m430 (`06_docs/mutants/m430_...py:3`) says "Killed by TestARebuiltSnapshotIsAnsweredAgain". At the tip that test passes against it: the test flips `correct`, so the options differ anyway (`map_prop_chart_test.go:325`).
  - m430 is killed only by `TestARefreshAnsweredTooSoonKeepsItsAnswers`, which reads the counter itself (`:372,383`).
  - The new RC-4 test changes the noise, so the options differ again (`map_prop_ahead_test.go:203`).
  - **What goes unguarded:** the case only `built` guards. A snapshot is rebuilt with other options and then rebuilt back at the same moment, and an hour ahead or chart asked before the rebuilds is answered afterwards. Dropping `built` passes every test.
  - Batch 63 re-vetted m231 and m524 (`build-log.md:794`) but not m430.
  - Simplify/Delete: N. Action: add a test that rebuilds and then rebuilds back, for the hour ahead and for the chart; correct m430's "Killed by" line; re-vet the mutants whose kill came through the options.
- **R2C-6 (Minor), go-tuiMaps:** deleting `Ground`'s `recheckHostRampsLocked` (`look.go:188`) survives the root package. A-18(2) claims all five setters re-check; I tested only `ColourDepth` (killed) and `Ground` (survives). Simplify/Delete: N. Action: extend `hostramp_test.go` to every setter.
- **R2C-7 (Minor), go-ionomaps:** the age gate on GloTEC's index ("none is fetched to be thrown away", `update.go:277`) is unpinned: removing it survives the root package. Simplify/Delete: N. Action: a test that counts grid asks against a stale index.

##### 8. P10 conformance
- `make p10`: 0 live, 23 floors held. go-ionomaps' tools/tables is now under every leg and green.
- **R2C-4 (Minor):** p10floor skips any exempted density row whose reason has no "FLOOR" and prints only the count it held (`tools/p10floor/main.go:96-99`, `:80`).
  - A row reworded without "FLOOR", or a new package-wide row, passes unguarded and unreported.
  - D-264 says every row takes a floor.
  - Simplify/Delete: N. Recommendation to the HUM LEAD: print the floorless rows, or fail on them.

##### 9. Evidence soundness
- **R2C-8 (Minor): the words and the drawn night use two different moments.**
  - The words take day or night at `shown.Now.At` (`map_prop_words.go:112`).
  - The night is drawn at that moment truncated to 5 minutes (`map_prop_draw.go:83`, `map_night.go:18`), and the words name that moment (`map_night.go:91-97`).
  - Within 5 minutes of the terminator the words and the dotted edge disagree, which is the case D-263 was meant to close.
  - Simplify/Delete: Y. Action: pass `propNightMoment()` to `propDayWords`, so there is one moment.
- **R2C-10 (Minor), go-ionomaps:** `Rejected` now includes `res.Implausible` (`update.go:437`), and `onto.go:80-82` counts a reading whose M(3000)F2 is implausible while its foF2 is still used. watchpost then says "N rows the range checks refused", so a row that shaped the field is counted as refused. Simplify/Delete: N. Action: count the M(3000)F2-only refusals apart, or reword.

##### 10. Evasion lens
- The mutant corpus's "Killed by" lines are kept by hand. `mutant-check` proves only that each mutant applies and compiles (`Makefile:160-164`), and `mutant-anchors` proves only that each anchor matches. That is how m430's claim went stale (R2C-1).
- The record says RC-7 is "checked as fixed" while `hasKey` and `stillOrphan` remain.
- RC-9's tag gap and RC-13's skip are still green without checking anything.
- p10floor passes over floorless rows (R2C-4).
- `finaltag` reads only `require` lines (`tools/finaltag/main.go:24`), so a `replace` that points a paired library at an `-rc` evades it (Minor).

##### What I verified clean
- watchpost:
  - the RC-4 drop path, and the find ending with its session (`dropFind`, with its test)
  - runlog's before/after tree check
  - perfsum's handling of unread gauges
  - the history split
  - the latitude band in `near.go`
  - D-247's words
- go-ionomaps:
  - the `solarUsable` and `short` paths
  - the toolchain pin
  - RC-13's three code fixes
- go-tuiMaps:
  - `turnPassed`: `sun.Turn` always returns a time after the moment, so there is no endless recompute
  - `DaylightAt`
  - the `classTokens` fallback
  - `ColourDepth`'s early return on the same hint, so there is no re-check every frame
- go-tuiMaps v0.2.2 (`main`): 17 files of go.mod and toolchain-floor edits only. I read its diffstat only; the `scripts/gate` diff has no toolchain lines.

**Fix first: R2C-1.** Add a test that rebuilds a snapshot and then rebuilds it back, and correct m430's record. It is RC-4's class, the one round 1 said to fix first, and today nothing would catch it coming back.

## Project Hygiene (R2H)

#### R2H: Project Hygiene, REVIEW exit red team round 2 (blind)

**Tree state.** At the start:
- watchpost `5b71df2c` was ` M .../02-analysis/rulings.md`. That is the uncommitted D-269 row.
- go-ionomaps `8d84569` was clean.
- go-tuiMaps `73c602d` was ` M 06_docs/gate-runs.md`, holding one row from 2026-10-24T21:19:58Z (see R2H-2).

At the end, the trees also held rows appended by other reviewers' gate runs: go-ionomaps `gate-runs.md`, and go-tuiMaps rows from 16:19Z to 16:23Z. None of them are mine. I ran the libraries' gates only in fresh clones under my scratch directory (`.../rt2-r2h/{io,tm,wp}-clone`).

Unless a finding says otherwise, it was reproduced against those tips.

**What I ran** (go1.27.3, GOTOOLCHAIN=local):
- **watchpost tree:**
  - `make lint-authoring` OK. `make dupes` 2/2 ratified. `make mutant-anchors` 795 OK. `make lint` gave no new findings.
  - `go vet ./...` clean.
  - `go test -race` passed on `tools/{finaltag,p10floor,runlog,prehook,prepush,perfsum}` and on synth's new test.
  - `make p10`: the first run FAILED at the `a2dh p10 check` step. The rerun was green: 0 live, 0 unmatched, 0 unratified, 165 rows, p10floor 23 rows.
- **watchpost clean clone:** `make build` OK. `make p10` fails, as documented.
- **go-ionomaps clean clone:** `scripts/gate --quick` green, with P10 NOT RUN. `go vet` in tmbench passes with no edits.
- **go-tuiMaps clean clone:** `scripts/gate --quick` **FAILED** (R2H-1). Run alone, `a2dh p10 check` on the go-tuiMaps tip gives 0 live findings.
- **Records:** every in-scope commit has a green gate or verify row for its exact tree (the commit minus its log), with one exception, R2H-2. watchpost hosted CI is green on every remediation commit through `5b71df2c`.
- **Leak scan:** I scanned the added lines of all four diffs for home paths, addresses, emails and tokens. Clean.

**Something you need to know.** go-tuiMaps' `TestALegThatIgnoresTermIsStillStopped` ran in my clone, found a `sleep 3607`, and ran `pkill -9 -f "sleep 3607"` across the whole machine. Other reviewers were running go-tuiMaps tests at the same time, so it may have killed one of their test's child processes. It touched no measurement process.

---

##### Part 1: round 1's findings

| Finding | Verdict | Evidence (checked in code) |
|---|---|---|
| RH-1 | PARTIAL | Fixed in watchpost (`README.md:524-525`) and go-ionomaps (`README.md:76-78`). go-tuiMaps' `README.md:127-129` names `.a2dh.yml` only: not the exemptions file, not the mirror (D-252). See R2H-8. |
| RH-2 | FIXED | go-ionomaps `README.md:69-79`, "Working on the library". |
| RH-3 | FIXED | `tmbench/go.sum` is tracked; `go vet .` exits 0 with no module edits. |
| RH-4 | FIXED | go-ionomaps `README.md:8`: "in release candidates (the newest tag v0.1.0-rc.N)". |
| RH-5 | RULED (D-252) | The ledger stays local; P10 is rostered per phase exit. Wording left over: R2H-8. |
| RH-6 | FIXED | `propagation-overlays/07-readiness/gates.md` exists, with the REVIEW-exit P10 row still "to come". `observer-maps/07-readiness/gates.md:7` is corrected. |
| RH-7 | PARTIAL | go-tuiMaps `main`: v0.2.2 (`50350d1`); the nightly is green on 2026-10-25 after four reds (read with `gh`). The go-ionomaps nightly cannot fire yet (R2H-6). |
| RH-8 | FIXED | `tools/prehook/main.go:38` names `git push --no-verify`, pinned by the prehook test at `:19`. |
| RH-9 | FIXED, with gaps | `tools/finaltag`, run from `release.yml:23-24`, exited 1 on the real `go.mod` in my run. The gaps are R2H-3. |
| RH-10 | FIXED | `go.mod:9-10` pins rc.6 and rc.5; both tags say `go 1.26.10`. |
| RH-11 | RULED (D-259), done | The `wires-ratified.md` exemptions fence is empty. `Event.Restored` and `State.Refused` are gone from the code. |
| RH-12 | FIXED | Row 1.1 is ticked in both checklists (go-ionomaps `:26`, go-tuiMaps `:25`). |
| RH-13 | FIXED (noted) | go-ionomaps `build-log.md:221`. |
| RH-14 | **PARTIAL** | Fixed in go-ionomaps. In watchpost, `04-development/build-log.md:4,7` still says "phase: BUILD … IN PROGRESS" and `02-analysis/rulings.md:4` still says "BUILD exit". The remediation list (`red-team-review.md:67`) assigned this to watchpost. See R2H-4. |

---

##### 1. Is anything in the tree that should not be?

Nothing committed. The 39 added files are tools, tests, mutants and docs; the leak scan is clean.

- **R2H-2. go-tuiMaps' tip commit does not carry its own gate row.** In this repository each commit carries its own run's row: the row's HEAD is the parent, and its tree is the commit minus the log (true for `c23fd89`, `c5c50e7`, `544b889d`). `73c602d`'s committed `gate-runs.md` ends on a docs-lane run marked FAILED ("nothing staged"). The green run for its exact tree (`bafce83237ce`, 21:19:58Z, the same second as the commit) has sat uncommitted in the working tree for about 18 hours. As committed, the record shows the rc tip's library state only as failed.
  - Severity: Minor
  - Evidence: go-tuimaps `06_docs/gate-runs.md:239` (working tree) against `git show 73c602d:06_docs/gate-runs.md`. My computed tree `bafce83237ce` matches the uncommitted row.
  - Simplify/Delete: N
  - Action: commit the row with the next commit, and note why it lagged.

##### 2. Does the build work from a clean clone, with the documented commands?

watchpost builds. go-ionomaps' quick gate is green. go-tuiMaps' quick gate went red in a clean clone:

- **R2H-1. go-tuiMaps' gate test hunts and kills processes across the whole machine, so the gate fails falsely when runs overlap, and kills the other run's processes.** `TestALegThatIgnoresTermIsStillStopped` decides a leftover process survived by running `pgrep -f "sleep 3607"` across the machine, then runs `pkill -9 -f "sleep 3607"`. Any other gate run on the host, whether another clone or another reviewer, makes it fail, and the kill reaches into that run. Reproduced: my clean-clone quick gate failed on exactly this ("a process that ignored TERM outlived the leg: pid 36882") while three other go-tuiMaps runs were live. The race-off pass of the same suite was green.
  - **A related symptom:** `a2dh p10 check` also fails when run concurrently in one tree. My first watchpost `make p10` failed and the rerun was green; other reviewers' go-tuiMaps rows at 16:19Z to 16:23Z read "FAILED: P10", yet run alone it gives 0 live findings. Under D-269's seven parallel reviewers, the gates stop being deterministic, and false FAILED rows are landing in the libraries' committed records.
  - Severity: Important. It fails closed, but it is destructive and reproduced today.
  - Evidence: go-tuimaps `gate_test.go:560-564`; my log `.../rt2-r2h/tm-quick.log:7-9`.
  - Simplify/Delete: N
  - Action: track the leg's own process group (a pid file, or a marker unique to the run), never a machine-wide pattern. Record that P10 is not safe to run concurrently, or serialize it as watchpost's `treelock` does (`tools/treelock/main.go:9` already says "`pgrep` IS NOT THE ANSWER").

##### 3. Are the gates actually run on every path that claims to run them?

- **R2H-3. `finaltag` (RH-9's fix) reads go.mod with a regular expression, so a `replace` defeats it, and nothing pins the release step.** I applied each case to a copy in my scratch directory:

  | go.mod variant | `finaltag v0.19.0` |
  |---|---|
  | Final pin plus `replace … => …/go-ionomaps v0.1.0-rc.6` | exit 0 |
  | Final pin plus `replace` to a local path | exit 0 |
  | CRLF line endings | exit 0 |
  | Trailing space after a version | that module dropped silently |

  - The CRLF and trailing-space cases would then fail `make verify`'s tidy gate. The remote `replace` would not.
  - No test, mutant or `required-gates.txt` entry asserts that `release.yml` runs the step; deleting lines 23-24 passes everything.
  - Severity: Minor
  - Evidence: watchpost `tools/finaltag/main.go:26,64-71`; `.github/workflows/release.yml:23-24`; cases in `.../rt2-r2h/ft/cases/`.
  - Simplify/Delete: Y. `go list -m -f '{{.Version}}{{with .Replace}} {{.Version}}{{end}}' <paired>` gives the effective version, and the regular expression goes.
  - Action: use the effective module graph; pin the workflow step with a test, as `ci.yml` is pinned.
- **R2H-6. The go-ionomaps nightly (D-262/D-90) is recorded as built, but it cannot run.** The workflow itself says a schedule fires only from the default branch. The default is `main`, and `main` has no `gate.yml` at all. Until the merge, a new standard-library advisory still surfaces only at the next push or dispatch. That gap matters most during VALIDATE, when pushes are few.
  - Severity: Minor
  - Evidence: go-ionomaps `.github/workflows/gate.yml:10-12,17-19`; `git show origin/main:.github/workflows/gate.yml` → "exists on disk, but not in 'origin/main'"; `build-log.md:220` "a nightly full gate (D-90)".
  - Simplify/Delete: N
  - Action: say "from the merge" in the build log and the RH-7 disposition, or dispatch the full gate on a schedule from somewhere that can fire.
- **R2H-7. `p10floor` passes, without a word, an exempted density finding whose row states no floor.** 52 of the 76 density rows carry no `FLOOR`. D-264 says rows "ratified by precedent with no figure take their measured figure too". Today every firing density finding has a floor (I checked all 23). But a dormant row whose package gains changed functions later will cover new code with no figure, which is RC-6's class again.
  - Severity: Minor
  - Evidence: watchpost `tools/p10floor/main.go:6,93-96`; `06_docs/p10-ledger.md` section P10-05 (76 rows, 52 with no FLOOR).
  - Simplify/Delete: N
  - Action: fail closed on a firing density finding with no floor (recommendation to the HUM LEAD, under D-264).
- **R2H-10. A required gate is passing by re-running.** `make test-say` failed 2 of 6 `make verify` runs, each time on an unchanged tree: rows 00:54Z and 04:48Z, both re-run green. F-219's trigger, "a third time on one host", was set by the agent, not ratified.
  - Severity: Minor
  - Evidence: `06_docs/verify-runs.md` (rows for `85f7046e` and `8092d107`); `06_docs/follow-ups.md` F-219.
  - Simplify/Delete: N
  - Action: recommendation to the HUM LEAD: rule the re-run policy now, with the diagnosis already in hand.

##### 4. Is anything named as owed that has quietly stopped being true?

- **R2H-5. go-tuiMaps v0.2.2 tagged only the root module.** `50350d1` moved every nested module's `go` line to 1.26.10 (`cmd/tuimaps`, `examples`, `tools/*`). No `cmd/tuimaps/v0.2.2` or other nested tag exists (checked with `gh api .../tags`), although v0.2.1 and v0.2.0 tagged them all. As a result, `go install .../cmd/tuimaps@latest` still resolves to `cmd/tuimaps/v0.2.1`, with floor go 1.26.9 and library v0.2.1. D-144 says "every module's go directive is 1.26.10", which is true of the tree but of no installable nested version. The project's own rule (`radar-loops/07-readiness/release-checklist.md:59-60`) is the root first, then the nested tags.
  - Severity: Minor
  - Evidence: as above.
  - Simplify/Delete: N
  - Action: tag the nested modules at `50350d1`, or record why not.
- **R2H-11. go-ionomaps changed after rc.6 with no candidate and no stated reason (D-206).** The commits are `3c2da5a`, `14b38ff` and `8d84569`. `git diff 749b17a 8d84569` touches only `tools/`, tests and docs, so watchpost's build is unaffected. This is still RH-10's pattern: the reviewed tip is not the tag. This time no reason is written at all.
  - Severity: Minor
  - Simplify/Delete: N
  - Action: recommendation to the HUM LEAD: rule whether D-206 covers changes to tools and tests only.

Owed rows otherwise: the wires fence is empty, gates.md's REVIEW-exit P10 row is honestly "to come", and the checklists' section-0 rows are release-commit rows. Nothing else has gone stale.

##### 5. Does the record say what happened, or what somebody intended?

The tree-to-row record holds for every watchpost and go-ionomaps commit in scope. Batch 65's account of the say hangs (batches 62 and 64) matches the FAILED rows. Three gaps:

- **R2H-4. Stale headers (RH-14, not fixed in watchpost) and one more that round 1 missed.** watchpost: `04-development/build-log.md:4,7` and `02-analysis/rulings.md:4`. go-tuiMaps: `propagation-fields/02-analysis/rulings.md:4` says "phase: DISCOVER".
  - Severity: Minor
  - Simplify/Delete: N
  - Action: batch-correct them.
- **R2H-9. Round 1's record lists what was intended and never closes it.** `red-team-review.md` has no table mapping each finding to the commit that fixed it. Its remediation list (`:63-70`) is written before the work, and one of its items, RH-14 for watchpost, was not done. Its status line (`:7`) still reads "the remediation batches follow", and `00-REQUIRED-READING.md:49` says "the remediation under way". Round 2 had to rebuild the dispositions from the code.
  - Severity: Minor
  - Simplify/Delete: N
  - Action: add a closure column (finding → commit, or ruling).
- **R2H-8. The P10 wording is overstated, and one error message misleads.**
  - The go-tuiMaps README (`:135-136`) and go-ionomaps' `scripts/gate:8` say `--quick` "still runs" P10, with no qualifier; hosted and clean runs print NOT RUN.
  - The go-tuiMaps README does not name the exemptions file (RH-1).
  - watchpost's `Makefile:540` prints "p10: live findings" when the actual failure is "no .a2dh.yml" (reproduced in my clone). That is a D-124 slip.
  - Severity: Minor
  - Simplify/Delete: N
  - Action: correct the three lines.

---

##### Attack on the remediation's own code

- `finaltag` is beatable (R2H-3).
- `p10floor` is permissive by design where D-264 reads as strict (R2H-7).
- **Clean:**
  - The prehook message, with its test.
  - The `runlog` tree-before/after checks: race tests pass, and the tree matches on every remediation row.
  - Batch 65's sample report, which goes to `t.TempDir()`.
  - The gates.md roster entries.
  - tmbench `go.sum`.
  - The rc.6/rc.5 re-pin and its `go 1.26.10` floors.
  - The v0.2.2 `main` nightly.

**Verified clean:** no leaks in any diff; watchpost lint, authoring, dupes, anchors, vet and P10 (run alone); go-ionomaps quick in a clean clone; P10 on go-tuiMaps (run alone); watchpost hosted CI green on all remediation commits; go-ionomaps' and go-tuiMaps' hosted quick and full gates green at each tip.

**Opinion: exit REVIEW** on hygiene grounds. Nothing here is Critical, and every finding is a bounded fix. R2H-1 makes the gates untrustworthy under exactly the parallel review the protocol runs, so fix it before round 3 or VALIDATE reads any more gate records.

**Fix first: R2H-1.** The machine-wide `pgrep`/`pkill -9` in go-tuiMaps' `gate_test.go:561-562`.

## Docs Quality (R2D)

#### Docs Quality red-team report, REVIEW exit round 2 (R2D)

**Tree state.** I recorded each tree at the start:
- watchpost `5b71df2c` (`feature/propagation-overlays`): ` M 06_docs/02_features/propagation-overlays/02-analysis/rulings.md`. That change is not mine and was there for the whole pass.
- go-ionomaps `8d84569`: clean.
- go-tuimaps `73c602d`: ` M 06_docs/gate-runs.md`. That change is not mine.

Every finding was reproduced against these states. My required `scripts/gate --quick` runs added rows to tracked files. In go-ionomaps a green row went into `06_docs/gate-runs.md`. In go-tuimaps a FAILED row went in; it is environmental (see the note at the end). I left both rows in place, because the gate record keeps failures.

**What I ran:**
- watchpost: `make lint` (no new findings), `make lint-authoring`, `make p10` (0 live / 0 unmatched / 0 unratified; p10floor 23 rows held), `make dupes`, `make mutant-anchors` (795 match). All exit 0.
- watchpost: `go vet ./tools/... ./modes/tty/`, then `go test -race` on `tools/{perfsum,runlog,docslane,finaltag,p10floor,reachscore}`, then `cmd/watchpost` (the record tests) and targeted `modes/tty`. All green.
- go-ionomaps `scripts/gate --quick`: green.
- I compiled both go-ionomaps README snippets in my scratch module against the local tree. Both compile.
- I built the committed `perfsum` into scratch and re-ran it on G1's 48-hour run (`~/watchpost-0.19.0-plan-evidence/g1-run-2026-10-19`).

##### Round-1 findings (RD)

| Finding | Verdict | Evidence checked |
|---|---|---|
| RD-1 | **PARTIAL** | Both comparisons are now in the tree: `00-REQUIRED-READING.md:47`, `problem-statement.md:105`, `validate-agenda.md:21`. The other half of the remedy, reconciling the summary's "dry run" verdict with D-241, went the other way (see R2D-1). |
| RD-2 | FIXED | `CHANGELOG.md:30-34` now says what changed. Its 8 MB matches `platform/history/cache.go:100`. |
| RD-3 | FIXED | `constants.go:36-45`. `constants_test.go:12,33`. go-ionomaps `scripts/gate:126-132,289-290`. `release-checklist.md:17,20`. |
| RD-4 | FIXED | `modes/tty/map_prop_chart.go:24-33,478-485`. `map_prop_chart_test.go:461-467` rebinds the keys to j/k. `requirements.md:82`. |
| RD-5 | FIXED | go-ionomaps `README.md:8` drops the rc number. watchpost `go.mod:9` pins rc.6. No README test was added, but the snippets compile. |
| RD-6 | FIXED | The alias comments name the fields. I checked all four go-tuiMaps aliases against the structs and they match exactly; go-ionomaps `Source` matches `internal/terms/terms.go:11-17`. See R2D-4 for a side effect. |
| RD-7 | FIXED | `architecture.md:43,133`. go-ionomaps `api-review.md:27`. |
| RD-8 | FIXED | go-ionomaps `implementation-plan.md:134` cites `ahead.go`. `red-team-plan.md` no longer holds the stale figures. |
| RD-9 | FIXED | `problem-statement.md:64,98,105`. |
| RD-10 | FIXED | `README.md:516` now cites D-38 / D-244. |
| RD-11 | FIXED | `release-notes.md:27` matches `internal/sun/sun_test.go:42-62` (0.5°, 78 moments of 2026). The 0.011° is named as a one-off. |
| RD-12 | **PARTIAL** | watchpost is fixed (`Makefile:302-306`). The go-ionomaps gate round 1 also cited still logs its refusal (R2D-3). |
| RD-13 | FIXED | `00-REQUIRED-READING.md:17-21,28` (D-257). |
| RD-14 | RULED | go-ionomaps A-29 (13), D-267. The ruling's reason misreads its source, and the remediation spread the pattern (R2D-4). |

##### 1. Does each document answer the question its title asks?

**R2D-6: The record that round 2 starts from stops before the remediation.**
- `00-REQUIRED-READING.md:49`, "Where REVIEW stands", ends on 2026-10-24 with "the remediation under way". It does not mention:
  - batches 59-65;
  - D-265 to D-269;
  - go-tuiMaps v0.2.2;
  - round 2.
- Its dated list is out of order: `:45` (10-18) comes before `:46` (10-12 to 10-17).
- `08-reports/red-team-review.md:7` still says "the remediation batches follow".
- Nothing maps a finding to the commit that fixed it. The check table's "Not yet checked" rows (`:90,:99,:112,:120`) are never closed.
- I had to rebuild the fix map from three build logs.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? Y: a "fixed in" column replaces the prose remediation list.
- Action: add a dated 2026-10-25 line, reorder the list, and close the table.

Otherwise each document answers its title.

##### 2. Does anything contradict the code, or another document?

**R2D-1 is the main item (see question 3).**

**R2D-3: The three repositories disagree on whether a refused gate run goes into the gate record.**
- watchpost's `Makefile:302-306` now says "the record holds runs, never refusals (RD-12)".
- go-ionomaps `scripts/gate:231-232` calls `fail` on "nothing is staged" and then logs the run (`:306`).
- go-tuimaps `scripts/gate:327-330` calls `note_failure "docs lane: nothing staged"`, and its EXIT trap (`:285`) logs it.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: one rule across the three repositories. The standing rule that the gate record keeps failures does not settle whether a refusal is a run, so this goes to the HUM LEAD.

**R2D-7: The README's map key table omits the chart window's target keys.**
- D-256 made previous and next target keymap actions (`chart.prev`, `chart.next`).
- Help shows them (`modes/tty/map_pane.go:1471-1479`).
- `README.md:195-210` lists every other Propagation key and says each "is a keymap action, so `[keys]`… can rebind it". It never names these two.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: add one row for the chart window's up and down keys.

##### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R2D-1: The committed scorer fails the G1 run that the record calls met.**
- RC-1's fix makes `perfsum -g1` treat any arm under 48 hours as "a dry run, not the measure", and it exits as a miss. Evidence:
  - watchpost `tools/perfsum/g1.go:121,138,143-145`;
  - the gate roster states this rule at `07-readiness/gates.md:29`.
- G1's run measured 47.97 hours per arm (D-241; A-81).
- I re-ran the committed `perfsum -g1` on that run:
  - it **exits 1**;
  - it prints "The run is shorter than G1's 48 hours: a dry run, not the measure, and it exits as one missed (RC-1)";
  - output: `<scratch>/g1-now.md`.
- The record still says "G1 met over its 48 hours" in:
  - `00-REQUIRED-READING.md:47`;
  - `problem-statement.md:105`;
  - `validate-agenda.md:21`.
- E-19 says ruling figures come only from the committed scorer at the committed version.
- The remediation list promised "the summary's 'dry run' line reconciled with D-241". Instead the instrument now contradicts the ruling harder, and no document says so.
- Severity: **Important**. It is the verdict VALIDATE will grade.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: a fork for the HUM LEAD.
  - (a) `perfsum` carries D-241's ruling for this run explicitly, for example as a named override recorded in its output.
  - (b) The record says the committed scorer refuses this run and G1 stands by D-241's ruling alone.
  - (c) The run is repeated.
  - The agent should not choose.

**R2D-5: The required reading says "the release says so", but nothing in the release says it yet.**
- `00-REQUIRED-READING.md:21` says the answers agree little better than the climatology "and the release says so".
- Nothing in the release says it yet:
  - `CHANGELOG.md:5-34` has no accuracy line;
  - D-255 defers the known-limits list to VALIDATE's end;
  - D-250's PR wording is not written yet.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? Y.
- Action: change it to "will say so (D-250, D-255)".

##### 4. Is the audience order right: designers, then PMs, then engineers?

**R2D-4: The RD-6 fix put red-team IDs into go-tuiMaps' public API docs.**
- The fix wrote "(watchpost RD-6)" into four exported doc comments that `go doc` shows a library user: go-tuimaps `describe.go:17`, `overlays.go:40,50,58`.
- `describe.go:21` sends readers to `go doc ./internal/describe Answer`. That command works only inside the library's own checkout, not for a host.
- A-29 (13) kept the ruling IDs for go-ionomaps only. Its reason, "R-7.1 asks each exported function to cite its source", misreads R-7.1. R-7.1 asks for "the published work it implements" (go-ionomaps `01-objectives/requirements.md:92`), not ruling IDs.
- go-tuiMaps' A-18 (7) does not rule on the IDs at all.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? Y.
- Action: a recommendation to the HUM LEAD. Drop the reviewer IDs from exported comments and give the full import path, and reconsider A-29 (13) on its stated basis.

Otherwise the entry document's order is now right (RD-13).

##### 5. Is a number published without its blind spots beside it (INST-5)?

**R2D-2: "G1 met" carries no word on arm-to-arm noise or on its single run.**
- The judged figures, −0.17 % CPU and −10.4 MB, are published bare in `00-REQUIRED-READING.md:47`, `problem-statement.md:105` and `validate-agenda.md:21`.
- A negative added cost means the run's noise between arms is at least that large. The summary shows the mode in use 10.4 MB RSS and 7.3 MB heap *below* the map alone (276.3 against 286.7; 95.8 against 103.1). That noise is close to half of the 25 MB target.
- The figure comes from one 48-hour run on one machine, and nothing says so beside it.
- The same class appears in `validate-agenda.md:26`: M5's figures (24.6-26.7 ms, 6.8-7.3 ms) carry no machine or run count.
- Severity: Minor.
- Evidence: as cited; `g1-run-2026-10-19/g1-summary.md`, the arms table.
- Simplify/Delete? N.
- Action: beside each "met", state the arms' spread and "one run, one machine (Apple M5 Pro)".

The Broadcaster run's evidence folder already notes its network outage beside the D-246 verdict (`g1b-run-2026-10-25/NOTE-outage.txt`), which is the pattern to follow.

##### Attack on the remediation's new surface

- R2D-1 (perfsum's fail-closed rule against D-241), R2D-3 (RD-12 fixed in one repository of three) and R2D-4 (RD-6's comments) are defects that the fixes themselves brought in.
- New documents I checked against their sources:
  - `validate-agenda.md`: every figure matches D-246, D-266, go-ionomaps `build-log.md:185` (the Pacific 1.47 against 1.22), and `globe_bench_test.go:59-60` (800×220 is in the benchmark).
  - `gates.md`: rows 26-32 match the code.

##### Verified clean

- The README's Go floor and its hosts table.
- go-tuiMaps v0.2.2's release notes and D-144 against the `043922ec..50350d1` diff.
- go-tuiMaps' `DaylightAt` and `Daylight` docs against `daynight.go:34-42` and `report.go:121-126,362`.
- `api-review.md` against D-251.
- The constants table against R-9.8 (SD 0.225 is the ruled 2-SD band of 0.45).
- The go-ionomaps README snippets compile.
- Every watchpost lint and P10 target, and every test I ran, passed.

**Note outside my lens (for Hygiene).** go-tuiMaps `scripts/gate --quick` FAILED on two legs:
- **`tools/oracle` FuzzAgree.** The cause is the gate's shared `go.work` at the repository root: it is written at `scripts/gate:649-661` and removed at `:274`, so it collides with a reviewer's gate running at the same time. The same test passes with a private workspace (`GOWORK=<scratch>/go.work`). So my FAILED row in `go-tuimaps/06_docs/gate-runs.md` is a false red.
- **P10.** It failed only on govulncheck's DNS during today's network outage.

##### Opinion

**Do not exit REVIEW.** No finding is Critical, and the rest are cheap, but R2D-1 leaves VALIDATE grading a G1 verdict that the committed instrument refuses.

**Fix first: R2D-1.** Put the fork (scorer override, record wording, or a re-run) to the HUM LEAD before VALIDATE.

## Business Quality (R2B)

#### Business Quality review: 0.19.0 REVIEW exit, round 2 (R2B)

**Tree state.** I ran `git status --short` at the start, against these tips:
- watchpost `5b71df2c`: ` M 06_docs/.../02-analysis/rulings.md`. This is the uncommitted D-269 row, which predates my work.
- go-ionomaps `8d84569`: clean.
- go-tuiMaps `73c602d`: ` M 06_docs/gate-runs.md`, with rows other reviewers' runs had already added.

At the end the only change was rows added to the libraries' `gate-runs.md`, which `scripts/gate --quick` writes by design:
- go-ionomaps: mine is `17:42:35Z quick green`.
- go-tuiMaps: mine are one `quick FAILED` (16:19Z or 16:20Z) and `17:49:02Z quick green`. The failure was govulncheck timing out on `vuln.go.dev` during the network outage, not the code. The retry was green.

Every finding below was reproduced against these tips. My scratch is `<scratch>`. It holds a copy of go-ionomaps (`ion/`) for the mutant, and every log.

**What I ran.**
- **go-ionomaps:**
  - `go test -race ./...`: all pass.
  - `scripts/gate --quick`: green.
  - One mutant on a scratch copy: `short` never names GloTEC. It is killed: `TestEveryFaultIsReported` fails "GloTEC stale" and "an index naming no grid", and `TestAGridHeldPastItsAgeIsNamed` fails.
- **go-tuiMaps:** `scripts/gate --quick`: green on the retry.
- **watchpost:**
  - `make lint` (no new findings), `make lint-authoring`, `make p10` (0 live, 0 unmatched), `make dupes` and `make mutant-anchors` (795 anchors match): all exit 0.
  - `go vet` on `tools/...` and `domains/propagation`.
  - `go test -race` on `tools/{perfsum,reachscore,finaltag,runlog}`, `domains/propagation` and `platform/propanswers`.
  - `modes/tty`, filtered to the Propagation tests.
  - `cmd/watchpost`'s trace tests.
  - All pass.
- **The Broadcaster arm's run in progress:** the committed `tools/perfsum`, read-only, on `~/watchpost-0.19.0-plan-evidence/g1b-run-2026-10-25`.

---

##### Part 1: round 1's findings

| Finding | Verdict | Evidence I checked |
|---|---|---|
| RB-1 "or worse" | **FIXED** | watchpost `modes/tty/map_prop_words.go:270,272` now say "about as good as typical here, or worse". The tests pin it: `map_prop_words_test.go:450-452` and `map_prop_honesty_test.go:82`. The legend keeps D-198's "~typical" (A-84 (2)). |
| RB-2 GloTEC called Fresh | **FIXED** | go-ionomaps `update.go:221-236`: `short` names GloTEC whenever no grid is held or the held grid is more than an hour old. `update.go:277-278`: when the index lists no usable grid, the update leaves it to `short`. `faults_test.go:69` now also asserts `Reason != Fresh`. My mutant is killed (above). |
| RB-3 PS-1's accuracy | **RULED** (D-250) | The ruling is in `07-readiness/validate-agenda.md:19-20`, the REQUIRED-READING (`:21`) and `problem-statement.md:98`. |
| RB-4 G1's baseline, no Broadcaster measured | **RULED** (D-246), with a gap (see R2B-1) | Both comparisons are in `problem-statement.md:105`. The Broadcaster harness was built (`scripts/quality/workload.sh` g1b, A-90), and the 48-hour run is under way. |
| RB-5 G1's 48-hour figures outside the tree | **FIXED** | `00-REQUIRED-READING.md:47`, `problem-statement.md:105` and `validate-agenda.md:21` all carry +3.42 % and +40.0 MB. |
| RB-6 changelog claims the memory fix | **FIXED** | `CHANGELOG.md:30-34` now describes what changed, and says the side-by-side run will measure its effect. |
| RB-7 known limits not in the listener-facing record | **RULED** (D-255); its agenda part **PARTIAL** (see R2B-3) | `validate-agenda.md` exists. |
| RB-8 instruments do not state their blind spots | **FIXED** | `tools/reachscore/report.go:42-48,69` writes them; `columns_test.go:94` pins the section. |
| RB-9 README's version | **FIXED** | go-ionomaps `README.md:7` now says "the newest tag `v0.1.0-rc.N`" instead of a number that drifts. watchpost `go.mod:9-10` pins rc.6 and rc.5, the tags. go-ionomaps' changes since rc.6 are in `tools/` and tests only (`git diff --stat v0.1.0-rc.6 8d84569`). |
| RB-10 default refresh | Comment **FIXED**; the default **RULED** (D-253, D-265, D-266) | `map_prop_settings.go:60-62` now calls the strings ceilings and cites the measured 13.94 and 2.66 MB an hour (A-83). For the default itself, see R2B-2. |

---

##### Parts 2 and 3: the lens

###### 1. Does the work meet the requirement that was written down, or the one that was remembered?

It meets the written requirement on the points round 1 raised.
- RB-1's wording is now D-129's, verbatim.
- RB-2's "never Fresh" is pinned in the library's own fault table.
- The remediation rulings that change behaviour reached the written requirements: watchpost's `requirements.md` carries D-247, D-255 and D-256; go-ionomaps' `requirements.md` carries D-247, D-248, D-249, D-251 and D-260.

**One finding against the remediation's new code:**

**R2B-4. The documented command to judge the Broadcaster arm fails.**
- `scripts/quality/workload.sh:36-37` tells the reader to run `go run ./tools/perfsum -g1 -against idle <outdir>`.
- `-g1` takes the directory (`tools/perfsum/main.go:60`), so this spelling reads `-against` as the directory. Run as written, it exits 2: `perfsum: use: open -against/use/phases.log: no such file or directory`.
- The working spelling is `-g1 <outdir> -against idle`, as `main.go:6` documents.
- `04-development/build-log.md:799` and `validate-agenda.md:21` repeat the ambiguous short form.
- This is the command that produces a VALIDATE verdict, so whoever judges the run at hour 48 hits the error first.
- Severity: Minor.
- Evidence: watchpost `scripts/quality/workload.sh:36-37`, `tools/perfsum/main.go:6,60`.
- Simplify/Delete? N.
- Action: fix the comment to `-g1 <outdir> -against idle`.

###### 2. What does the user lose if this ships as it stands?

**R2B-1. The Broadcaster operator (PS-1, issue #25's user) is on course to pay more than G1 allows, and the record has no rule for what a miss means.**
- I ran the committed `perfsum` on the 48-hour Broadcaster run at hour 10.6. Its own label calls this "a dry run, not the measure".
- The mode in use against the console never opening its map, judged `-against idle`:
  - **+1.49 % CPU against ≤1 %: missed**;
  - **+35.3 MB RSS (medians) against ≤25 MB: missed**;
  - wire 12.02 MB an hour, met.
- The miss follows from the design.
  - The Broadcaster's map is new in 0.19.0 (D-195), so the operator pays the map's whole cost. On the Observer, that cost is +3.59 % and +50.4 MB (D-246).
  - The Observer's arm met G1 only because its baseline already has the map open.
- The record does not anticipate any of this.
  - `validate-agenda.md:21` lists the arm only as "under way".
  - `problem-statement.md:105` says only "a Broadcaster arm is measured for VALIDATE".
  - Neither says G1's targets apply to this arm, nor what follows a miss: a known limit, a re-ruling of G1, or a fix.
- Round 1 warned against exactly this pattern: a guardrail judged after its figure is seen.
- Severity: Important. G1 is a guardrail, and the user it fails is the one the release names first.
- Evidence:
  - watchpost `07-readiness/validate-agenda.md:21`, `01-objectives/problem-statement.md:105`, `02-analysis/rulings.md:267` (D-246);
  - the committed `perfsum` output above, from `go run ./tools/perfsum -g1 ~/watchpost-0.19.0-plan-evidence/g1b-run-2026-10-25 -against idle`.
- Simplify/Delete? N.
- Action: a recommendation to the HUM LEAD. Before hour 48, write onto the agenda that G1's targets apply to this arm, and name the options for a miss:
  - a known limit in D-255's changelog list;
  - the map-cost follow-up given a target;
  - G1 re-ruled for the Broadcaster.

Not new, still standing: M1 is barely above the climatology. D-250 ruled it, and it is on the agenda.

###### 3. What was cut, and is the cut recorded where the next reader will find it?

**R2B-3. Two cuts from the remediation are not recorded where the next reader looks.**
- **The open map's own cost has no follow-up row.**
  - D-246 rules "The open map's own cost (about 3.6 % and 50 MB) is a follow-up."
  - `06_docs/follow-ups.md` has no row for it. I searched for "open map", "own cost", "map open" and "50 MB", and the newest rows are F-213, F-214 and F-219.
  - The REQUIRED-READING sends every follow-up to that file alone (`00-REQUIRED-READING.md:43`).
  - With R2B-1, this is the follow-up the Broadcaster most needs.
- **The focus limit is missing from the VALIDATE agenda.**
  - D-255 names three known limits for the changelog: the focus limit, the accuracy, and the map's cost.
  - Agenda item 7 says only "worded from this agenda's rulings". The focus limit (FR-3.7, D-122) is not an agenda item, so a list worded from the agenda would leave it out.
- Severity: Minor.
- Evidence: watchpost `02-analysis/rulings.md:267` (D-246); `06_docs/follow-ups.md` (no such row); `07-readiness/validate-agenda.md:25`; `00-REQUIRED-READING.md:21`.
- Simplify/Delete? N.
- Action:
  - add an F-row for the map's cost;
  - make agenda item 7 list D-255's three limits by name, plus D-261's 800 by 220;
  - fix the tense of `00-REQUIRED-READING.md:21`: "the release says so" is not yet true. The changelog says nothing about accuracy until VALIDATE's end.

The ruled cuts round 1 checked are still findable: F-207, F-209, F-210, F-189, and FR-7 in 0.19.1.

###### 4. Is there a cheaper way to get the same outcome for the user?

**R2B-2. The cheaper Hourly setting says STALE for part of every hour by construction. The agenda weighs it on cost and accuracy only.**
- Agenda item 4 (D-253, D-266) compares 10 minutes with hourly on MB an hour and on gm3 scores.
- The picture's age comes from the snapshot's oldest input: the GloTEC grid, or the oldest GIRO reading assimilated (`modes/tty/map_prop_update.go:243-251`).
  - Past an hour it says "STALE" in the alert's tone (`:239`, `:269-270`).
  - The snapshot is rebuilt only on the refresh tick (`map_prop_settings.go:39-47`; `armPropRefresh`, `map_prop_update.go:113`).
- Readings assimilated at a burst are up to 60 minutes old (go-ionomaps `internal/hybrid/hybrid.go:18`). D-184 itself measures them ageing from about 5 to about 70 minutes between bursts.
- So on Hourly, the label goes STALE late in each hour, until the next refresh. I read this off the code and D-184's own figures, and did not run it live.
- This is honest under A-56. It is also a cost of the setting the HUM LEAD is about to weigh as the cheaper default, and the agenda does not show it.
- Severity: Minor.
- Evidence: watchpost `modes/tty/map_prop_update.go:239,243-251,269-270`; `07-readiness/validate-agenda.md:22`; `02-analysis/rulings.md:205` (D-184).
- Simplify/Delete? N.
- Action: add to agenda item 4 how much of each hour the Hourly setting is said STALE. This is a fork for the HUM LEAD (UX); I do not rule on it.

Otherwise there is nothing cheaper to recommend. gm3's 3-hour run was offline and cheap, and it settled the 3-hour question. D-266's live fetch was authorised with its cost stated.

---

##### What I verified clean

- **RB-2's fix:** pinned by mutant, and it does not mislabel a grid held past its hour (`TestAGridHeldPastItsAgeIsNamed`).
- **D-249's kept solar flux:** its age is said in the words (`map_prop_update.go:370-371`), and the snapshot is Partial while the flux is held (`update.go:226`).
- **D-247's GIRO outcome words** (`map_prop_update.go:276-285`).
- **`reachscore`'s blind spots in its output.**
- **`perfsum`'s refusals:** it fails closed under 48 hours and exits as missed (I observed this).
- **go-tuiMaps v0.2.2** (`043922ec..50350d1`): the toolchain only. go directives moved to 1.26.10, the `Version` string to 0.2.2, plus the test floor and docs. No public name or drawing changed, as its release notes say.
- **Post-rc library changes:** none reach the library code watchpost builds.
- **The trees:** watchpost's lints, P10, dupes and anchors are clean, and both libraries' quick gates are green.

##### Opinion

**Exit REVIEW**, as long as R2B-1's decision rule is on the VALIDATE agenda before the Broadcaster run reaches hour 48.
- Round 1's code defects in my lens are fixed and pinned.
- Its record gaps are closed.
- Nothing I found is a code defect in the shipped product.
- What remains is what VALIDATE decides, and it should decide it in the open.

**Fix first: R2B-1.** State now that G1 applies to the Broadcaster arm, and what a miss leads to. At hour 10.6 the arm misses both of G1's targets, and the operator is the user this release names first.

## The REVIEW lens (R2L)

#### REVIEW lens, round 2 (R2L): 0.19.0 REVIEW exit (D-269)

**Tree state.** I ran `git status --short` at the start. watchpost `5b71df2c` had ` M .../02-analysis/rulings.md`, the uncommitted D-269 row, which is not mine. go-ionomaps `8d84569` was clean. go-tuiMaps `73c602d` had ` M 06_docs/gate-runs.md`, not mine. At the end, go-ionomaps `06_docs/gate-runs.md` was also modified, by other reviewers' gate rows at 16:09, 16:15 and 17:42Z. My own gate row (16:12:55Z) is in my clone's file only.

I edited no tree. Every finding was reproduced against those SHAs, in my own copies under `.../scratchpad/rt2-r2l/`:
- `iono/`: an rsync copy where the probes ran. The probe files are `iono/r2l_probe_test.go` and `iono/r2l_state_probe_test.go`.
- `ionoc/`, `tui/`, `wp/`: local clones at the same tips, with the git-ignored `.a2dh*` files copied in.

**What I ran:**
- go-ionomaps `scripts/gate --quick`: **green**, P10 included.
- go-tuiMaps `scripts/gate --quick`: the first run FAILED for two reasons.
  - govulncheck: the DNS outage ("lookup vuln.go.dev: no such host").
  - `TestALegThatIgnoresTermIsStillStopped`: see R2L-6.
  - The rerun after the outage was **green**.
- watchpost:
  - `make lint-authoring`: OK.
  - `make dupes`: 2 groups, both ratified.
  - `make mutant-anchors`: all 795 anchors match.
  - `make lint` and `make p10`: failed on the outage, then green on the rerun. Lint has no new findings (6 baselined). P10 has 0 live, 0 unmatched, 0 unratified, and 23 rows at or over their floors.
  - `go vet` on the touched packages: clean.
  - `go test -race` on `domains/propagation`, `geodata`, `platform/history`, `tools/{perfsum,finaltag,runlog,p10floor}`, and the Propagation tests in `modes/tty`: all pass.
- One mutant of my own on the copy: the skew clamp at `stations.go:350`, `notAfter` → raw. It was killed by `TestAListStateIsCheckedOnApply`, and I restored the copy.

---

##### Part 1: round 1's REVIEW-lens findings

| Finding | Verdict | Evidence I checked |
|---|---|---|
| RL-1 | **PARTIAL** | A station's own 5xx, 404 or transport error is now counted and the burst goes on (go-ionomaps `internal/giro/asker.go:191-199,249-259`). 401/403/410 give a day's pause, said, and kept across a restart (`asker.go:207-210,121-123`; `throttle.go:94-107`). The unreached stations go first next time (`stations.go:179-189`). The outcome is said all hour (watchpost `map_prop_update.go:277-285,350-352`). Left over: R2L-1 and R2L-2. |
| RL-2 | **FIXED, with a regression** | No grid is fetched without a mean (`update.go:130-139`). The no-field snapshot is labelled `Partial` or `Offline` and carries what was unread (`update.go:146-151`). The mean is kept up to 7 days (`state.go:39-41,56-60,103-112`; `solar.go` `Resume`). D-260 brought in R2L-3. |
| RL-3 | **RULED** (D-246) | Both readings are recorded (`07-readiness/validate-agenda.md:21`). The Broadcaster's 48-hour arm is "under way", so no figure is in yet. |
| RL-4 | **RULED** (D-250) | `00-REQUIRED-READING.md:20` says the answers agree "little better than the climatology", and that no accuracy floor rides on them. |
| RL-5 | **PARTIAL** | The harness is real and re-runnable: `mutants_test.go` and `mutants_kill_test.go`, plus `scripts/gate:114-120,285`, which refuses a run that selects no mutant. But its 28 mutants are almost all the remediation's own. **None touches `answers.go`, `reach.go`, `lower*.go`, `internal/glotec`, `internal/limits` or `internal/climatology`.** Those are the hand mutants D-74's root row rests on, and RL-5 asked for them to be re-encoded. |
| RL-6 | **FIXED** | `update.go:221-236` (`short`) and `:277-279`. Held by m001 and m002. |
| RL-7 | **FIXED** | `update.go:242-248` (a floor of 10 minutes, then 15) and `:258` (a 5xx is a refusal). Held by m007 and m008. See R2L-3 for the side effect. |
| RL-8 | **RULED** (D-252) | The roster is `07-readiness/gates.md`. Its REVIEW-exit P10 row is still "to come" (`gates.md:7`). D-268's new root is in a git-ignored `.a2dh.yml` (go-ionomaps `.gitignore:22`): the same one-disk class, ruled. |
| RL-9 | **PARTIAL** | `CHANGELOG.md:32-34` now says what changed, not "bounded". But the verdict RL-9 wanted before REVIEW exits has moved to VALIDATE on the agent's agenda (`validate-agenda.md:24`), with no D-row or A-row ratifying the move. |
| RL-10 | **FIXED** | `go.mod:9-10` pins rc.6 and rc.5. rc.6 (`749b17a`) differs from `8d84569` only in `tools/` and tests. rc.5 is `c5c50e7`, which differs from `73c602d` only in docs. |
| RL-11 | **FIXED** | `throttle.go:152-157` and `stations.go:306-360` (5 minutes taken as now). Held by m011 and by my mutant. The coupling it leaves is R2L-4. |
| RL-12 | **FIXED** | `ionomapstest/replay.go:41-57,114-117` answers per address. `modes/tty/map_prop_states_test.go` walks the real library into the words. |

---

##### 1. Happy Path Bias

**R2L-1. Three stations failing next to each other make the map say "GIRO unavailable this hour", every hour for up to 3 days, while GIRO answers the rest.** Severity: **Important**.
- When every live station fits in one burst, the rotation's start never moves (`stations.go:169`). `Unasked` (`:179-189`) moves only the stations after the stop, never the three that failed. The third failure in a row then sets `Unavailable` (`asker.go:211-214`) every hour.
- Probe `TestR2LDeadTrio`, with 39 live stations and stations 19 to 21 answering 503, over 6 hours:
  - the outcome was "unavailable" every hour, including 30 minutes on;
  - 36 of 39 stations were assimilated on alternate hours, and **only 22 asked, 19 used** on the others.
- watchpost then says "GIRO unavailable this hour: the readings it gave first · 36 stations, oldest …" (`map_prop_update.go:280-283,345-352,374-379`). That contradicts itself, and it is D-239's class.
- m019 checks only that the next burst does not *start* at a failed station.
- Simplify/Delete: N.
- Action:
  - Count failures in a row only while nothing in the burst has answered yet, or ask the stations that failed last next hour.
  - Say "N stations did not answer", not "GIRO unavailable", once any station has answered.
  - Add a test with three adjacent stations failing.

**R2L-2. Any status D-247 does not name is taken as an answer: no count, no word, and the back-off resets.** Severity: **Important**.
- `ended` (`asker.go:249-259`) classifies only 404, 429, 5xx and 401/403/410. Everything else reaches `a.thr.Answered()` and `return answered` (`:233-235`): a 400, a 451, a redirect not followed.
- Probe `TestR2LAll400`, with every station answering 400, 451 or 302 over 3 hours: 120 GIRO asks, outcome "answered", 0 failed, 0 rejected.
- The map says "no station readings this hour: the background alone", and nothing goes to the diagnostics.
- A change to GIRO's query API (a 400 for every station) is RL-1(b)'s class surviving: 960 asks a day, told nothing, under RK-2.
- Recommendation to the HUM LEAD (D-247): a per-station 410 is a resource signal ("this station is gone"), yet it pauses all of GIRO for a day as a host denial.
- Simplify/Delete: N.
- Action:
  - Make any unclassified non-200 a station failure: counted, and in the run of failures.
  - Name the status once in the diagnostics.
  - Add a test.

##### 2. Systemic Failure Modes

**R2L-3. Since D-260, a 5xx (or no answer) on the daily solar file alone stops the GloTEC, D-RAP and scales asks on every update. With a mean held, the map is drawn from the climatology while GloTEC is healthy.** Severity: **Important**. The remediation brought this in.
- `refreshSolar` is asked first on each new UTC day (`update.go:322-330`). A 5xx is now `refused` (`:258`), and `refused` skips every later ask (`:131-139`).
- Probe `TestR2LSolarDownBlindsGloTEC`, a day after a good read, with a fresh grid listed, over an hour of updates:

| The solar file | NOAA asks | GloTEC asks |
|---|---|---|
| 500 | 4 | **0** |
| no answer | 4 | **0** |
| 404 | 13 | 7 |

- In the first two rows the background is the climatology, and the snapshot is `Offline`: "nothing came back from NOAA". True only because nothing else was asked.
- Before D-260 a 5xx was no refusal and the grid was fetched. A 404 still lets it through, so a 5xx now degrades more than a 404 does.
- This is RL-2's class ("one small NOAA text file decides …") brought back. It drops the source the release's measured gain rests on (mainland foF2 0.46 against the climatology's 0.71).
- The D-249 test misses it: `nomean_test.go:48-70` lists a fresh grid with the solar file down and asserts the mean, the age and the unread name. It never checks that the grid was used, and it passes with no GloTEC ask.
- Simplify/Delete: N.
- Action, a fork for the HUM LEAD (D-39's "a refusal stops the update" against D-260):
  - Stop the round only on a 429, or on a failure while no mean is usable.
  - Or ask the index before the solar file.
  - Assert `Background.FoF2 == GloTEC` in the D-249 test.

GIRO and NOAA, and two instances sharing the state file (FR-4.12), are as round 1 found: ruled, nothing new.

##### 3. Gate Validity

- No new finding. The gates the remediation added each watched a failure:
  - `tools/finaltag`: `TestAFinalTagIsRefusedOnAReleaseCandidate`. A `replace` dodge is closed by `TestGoModHasNoReplace`.
  - `tools/p10floor`: `gates.md:31`, which records a lowered copy failing.
  - The go-ionomaps mutant leg: `scripts/gate:120` fails a run that selects no mutant.
  - perfsum's fail-closed RSS and the "short run is missed" check (`tools/perfsum/g1.go`; `main.go:262-275,440-446`).
- Noted, but RULED class (D-252): D-268's `tools/tables` root and every P10 exemption still live only on this disk (go-ionomaps `.gitignore:22`).

##### 4. Adversarial Input

No new concern:
- A saved state is bounded and checked: version, size, flux bounds, codes known, no station left out twice, polls bounded (`state.go:88-115`; `stations.go:333-361`).
- Validators over 256 bytes are dropped (watchpost `propagation.go:273-287`).
- RP-6's latitude band is correct. Latitude difference never exceeds great-circle distance, which holds across the pole, and a NaN latitude gives an empty band (`geodata/near.go:154-168`).
- `DaylightAt` does not refuse a NaN place (go-tuiMaps `daynight.go:38-43`): it returns Known and "night". watchpost's places are validated upstream, so nothing is reachable.

##### 5. Graceful Degradation

R2L-3 is the main one. In addition:

**R2L-4. A restore is all or nothing: a GIRO-list state the library refuses throws away the kept F10.7 mean too.** Severity: **Minor**.
- `Restore` returns before keeping the mean when `giro.Import` fails (`state.go:103-112`).
- `Apply` refuses a whole state over one station code the seed no longer lists (`stations.go:345-349`). So a library upgrade whose seed drops a retired station refuses every host's state: GIRO is asked at once and the mean is lost. A clock stepped back more than 5 minutes does the same.
- Probe `TestR2LStateCoupling`: an unknown code, then the solar file down, gave `ErrBadState` and "no field yet".
- The diagnostic says only "GIRO's throttle state was not taken" (watchpost `propagation.go:133`).
- Simplify/Delete: N.
- Action:
  - Drop unknown codes rather than refuse the state.
  - Restore the mean on its own.
  - Name the mean in the note.

**R2L-5. `Inputs.Rejected` now mixes the burst's rows with refusals made on each rebuild, and watchpost gives the wrong cause, every update.** Severity: **Minor**.
- `Rejected` is `l.giro.Rejected + res.Implausible` (`update.go:437`), and `Implausible` is recounted on every build.
- watchpost writes "GIRO's last burst sent %d rows the range checks refused" on each applied update (`map_prop_update.go:152-154`), about 6 times an hour. The cause is often D-248's residual bound or a station moved, not a range check.
- A reading whose M(3000)F2 alone is refused is counted although its foF2 was used (`onto.go:85-88`).
- On the bound itself: 8.0 MHz sits 0.81 MHz above the largest residual seen, measured on the recorded quiet days (`onto.go:33-41`). Nothing in the tree shows it measured over the climatology background (GloTEC absent) or on a storm day, exactly D-248's counter-argument. It is a recommendation to measure.
- Simplify/Delete: N.
- Action:
  - Carry `Implausible` in a field of its own.
  - Note the count once a burst, with the right cause.

##### 6. Test Confidence vs. Test Results

- The D-249 restart test proves its fixture, not the field: see R2L-3.

**R2L-6. go-tuiMaps' `TestALegThatIgnoresTermIsStillStopped` judges, and kills, processes across the whole machine.** Severity: **Minor**. It predates the scope but sits on the gate path.
- It runs `pgrep -f "sleep 3607"` and then `pkill -9` (`gate_test.go:561-563`).
- My first quick gate failed on it ("outlived the leg: pid 34820") while other reviewers' gates ran in parallel. The rerun was green.
- The same lines can kill another run's process.
- Simplify/Delete: N.
- Action: mark the planted process uniquely (a per-test token in its arguments) or check by process group.

The rest of what I sampled asserts what it claims. The states walk does use the real library's snapshots. The mutants I ran against my copy were killed.

##### 7. Dependency Fragility

- R2L-4: a GIRO seed change breaks every saved state across an upgrade.
- The pins match the reviewed code (RL-10).
- watchpost's words now depend on go-tuiMaps `DaylightAt` (D-263), which is v0.3.0 API to freeze at its final tag.
- go-tuiMaps v0.2.2 (`50350d1`) changes only the directives, the gate toolchain and `Version`. A side note on it: the comment at `repo_test.go:92-96` is a history comment.
- No new finding beyond R2L-4.

##### 8. Problem statement present, real, honored

PS-1 and PS-2 are present and locked (`problem-statement.md:14-31`).
- **PS-2:** M4 holds a real gain.
- **PS-1:** claimed as answerability alone, as D-250 ruled. The Broadcaster's own cost (D-246's arm) is not yet measured to its end.
- R2L-1 and R2L-3 matter to both: the words a listener reads ("GIRO unavailable", "nothing came back from NOAA") can be untrue while the release's measured inputs are available. No new finding of this question's own.

---

##### Verified clean

- RL-6, RL-7, RL-10, RL-11 and RL-12 hold in code and tests.
- D-247's per-station counting and day-long denial, D-248's seed position (more than 50 km refused), and D-249's 7-day mean are built as ruled.
- RC-4's `propMade` key (`map_prop_update.go:78-95`; `map_prop_ahead.go:56-67`).
- RP-6's band of latitude.
- go-tuiMaps `DaylightAt`, `turnPassed` (RC-14), `Glyph` (RP-4), and `classTokens` returning none.
- `perfsum` fails closed.
- `finaltag`.
- All the targeted race runs and lint legs listed above.

##### Opinion

**Do not exit REVIEW.** Nothing is Critical. But R2L-1, R2L-2 and R2L-3 are library defects in failure handling, each with a probe that reproduces it, and each makes the words say something untrue (D-239). R2L-3 was brought in by round 1's own fix. At SEV-0, round 3 follows their remediation.

**Fix first: R2L-3.** One NOAA file's 5xx now blinds GloTEC, the release's main source, and the D-249 test passes over it.

## InfoSec (R2I)

#### InfoSec red team, round 2 (R2I): 0.19.0 REVIEW exit (D-269)

**Tree state at start (`git status --short`):**
- watchpost `5b71df2c`: ` M 06_docs/02_features/propagation-overlays/02-analysis/rulings.md`. This edit predates me and is not mine.
- go-ionomaps `8d84569`: clean.
- go-tuiMaps `73c602d`: ` M 06_docs/gate-runs.md`. Also predates me.

**Tree state at end:** the same, except go-ionomaps now shows ` M 06_docs/gate-runs.md`. That is the row my required `scripts/gate --quick` run appended (`2026-10-25T16:09:01Z | 8d84569 | quick | green`). go-tuiMaps' quick gate appended a row to its already-modified file.

**Where each finding was reproduced:**
- go-ionomaps: a `git archive 8d84569` copy in my scratch.
- watchpost: a scratch module with `replace` pointed at the tree at `5b71df2c`.
- Code reading: against the trees above.

Scratch: `<scratch>`

**What I ran:**
- **watchpost:** `make lint`, `lint-authoring`, `p10`, `dupes` and `mutant-anchors` all exit 0 (p10: 0 live, 0 unmatched, 0 unratified; p10floor 23 rows at floor). `go vet` is clean on propagation, locations, httpx, history, lineup, synth, finaltag, app and modes/tty. `go test -race` passes on propagation, openmeteo, geodata, httpx and history, on `app -run 'Prop|Tile|Map|Settle'` and on `modes/tty -run 'Prop|Find|Setup'`.
- **go-ionomaps `scripts/gate --quick`:** green.
- **go-tuiMaps `scripts/gate --quick`:** every gofmt, vet and test leg (race on and off) is green across all modules. The P10 leg failed only because `govulncheck` could not fetch its database ("govulncheck: fet…") during the network outage. That is the environment, not a code fault; re-run it when the network is up.
- **Mutants**, applied to scratch copies only. I checked the baseline green first, with the four repo tests that need git skipped. Killed:
  - solar: `sc.Err()` check, duplicate-day check, `maxBytes` check
  - `Apply`'s duplicate-station check
  - `Heard` keeping a placed station
  - the 50 km refusal
  - `FoF2Bound`
  - both RI-6 guards
  - `solar.Resume` in `Restore`
  - **Survived:** one, R2I-4.

---

##### Part 1: round 1's findings, checked against the code

- **RI-1: RULED (D-248), with a residual worth the HUM LEAD's eye.**
  - Fixed per D-248:
    - A placed station keeps its place: go-ionomaps `internal/stations/stations.go:268`.
    - A reply placing it more than 50 km away is refused and counted: `internal/giro/asker.go:270-276`.
    - Residual bounds: `internal/assimilate/onto.go:39-40,74-84`.
  - RI-1's exact repro (20 MHz over a 7 MHz grid) is now refused: `implausible=3`, Sydney stays 7.00.
  - **Residual.** Three seeded stations, at their seed positions, reading 14.9 MHz (just under the absolute 8 MHz bound) still give 14.90 at Sydney and at 60°S 120°W (mean residual 7.90). D-248 keeps D-105's global offset, so this is ruled.
  - **Recommendation to the HUM LEAD:**
    - A bound relative to the background. At night foF2 is 2–4 MHz, so an absolute 8 MHz lets one reading triple the field.
    - Score 999 still skips D-61's filter: `internal/giro/giro.go:200`.
    - 84 of the seed's 123 stations are unplaced and take any first position. That follows RI-1's own suggestion, but it is still an open door.
- **RI-2: PARTIAL.**
  - Fixed in watchpost `domains/locations/openmeteo/geocode.go`:
    - Off-globe refused: `:90-92`, verified with 1e300.
    - Zone must load: `:94-96`; `"../../../etc/passwd"` and OSC 52 are dropped.
    - Postcode shapes: `:110-112,153-164`. ESC-bearing US zips are dropped; `SW1A 1AA` is kept.
  - What remains is R2I-2.
- **RI-3: PARTIAL.**
  - Fixed: duplicate days count once, a scanner error is an error, and a size bound exists (go-ionomaps `internal/solar/solar.go:51,59-60,64`).
  - Still open: the line-count cap truncates silently (R2I-1).
- **RI-4: FIXED for every `httpx` client.**
  - `platform/httpx/httpx.go:264,269` caps headers at 64 KiB, held by `headers_test.go`.
  - Validators over 256 bytes are dropped: `domains/propagation/propagation.go:273-287`.
  - The tile client is not capped: R2I-3.
- **RI-5: FIXED.** `internal/stations/stations.go:355` refuses a repeated station. The new `Unasked` (`:179-189`) keeps `leftOut` unique, so a saved state still passes `Apply`. Mutant killed.
- **RI-6: FIXED.**
  - go-ionomaps `update.go:277` returns when the index lists no usable grid.
  - `short` (`update.go:221-236`) then names GloTEC unread.
  - Both mutants killed.
- **RI-7: FIXED.** `fileOnly` (`domains/propagation/propagation.go:138-151`) cuts `PathError` and `LinkError` paths to the file name. It is used at `:133` and `:186`. `CreateTemp`'s pattern path reduces to its base name too.
- **RI-8: FIXED.**
  - watchpost `go.mod:9-10` pins go-ionomaps `v0.1.0-rc.6` and go-tuiMaps `v0.3.0-rc.5`; both tags' `go.mod` say `go 1.26.10`.
  - rc.6 (`749b17a`) differs from `8d84569` only in tests, tools and docs.
  - rc.5 (`c5c50e7`) differs from `73c602d` only in docs.
  - So the reviewed code is the shipped code.

---

##### Part 2 and 3: new findings

**R2I-1: The solar file's line cap still cuts the file short silently, the same class as RI-3.**
- **What happens:** the scan stops at 1000 lines without an error. A file with 985 header or comment lines and then 30 days (9.3 KB, far under `maxBytes`) parses only the first 15 days.
- **Reproduced:** `err=nil`, `days=15`, `mean=300.0`, `ok=true`. The honest mean over 30 days is 185. Because the oldest days are read first, the newest are the ones lost.
- **Severity:** Minor (needs a malformed or impersonated NOAA reply over TLS).
- **Evidence:** go-ionomaps `internal/solar/solar.go:57` (`for n := 0; sc.Scan() && n < maxLines; n++`) and `:16-19`.
- **Simplify/Delete?** N.
- **Action:** treat reaching `maxLines` with input still left as `errLine`, or drop the line cap now that `maxBytes` bounds the work. Add the case to `TestAFileIsReadWhole`.

**R2I-2: The geocoder keeps the zone `"Local"`, and a postcode is still drawn raw where it is shown.**
- **`"Local"`:**
  - `time.LoadLocation("Local")` succeeds, so `TZ:"Local"` is kept (reproduced).
  - `app/radio.go:774` then gives that place the listener's machine clock.
- **Postcodes:**
  - The fix cleans postcodes at entry only. `modes/tty/setup_form.go:48,50,53,77,226,228,248` still concatenate `Zip` unescaped beside `render.Plain(Label)`.
  - A zip saved to `config.toml` or history before the fix, or a hand-edited config, is drawn raw on every launch. `platform/config/config.go:28` has no check.
- **Severity:** Minor.
- **Evidence:** watchpost `domains/locations/openmeteo/geocode.go:94`; `setup_form.go` lines above.
- **Simplify/Delete?** N.
- **Action:**
  - Refuse `"Local"` (and `"UTC"` is harmless).
  - Draw `render.Plain(Zip)` at those seven sites, so the draw does not trust stored text.

**R2I-3: The map-tile client's reply headers are not capped. A-84(9) says "every client's" is.**
- **Where:** watchpost's rebuilt tile transport (`app/maptiles.go:42-44`) and go-tuiMaps' own (`internal/fetch/fetch.go:134-142`) set no `MaxResponseHeaderBytes`, so they keep Go's default of about 10 MB per reply.
- **Severity:** Minor.
- **Evidence:** the two files above. go-tuiMaps has no `MaxResponseHeaderBytes` anywhere outside tests.
- **Simplify/Delete?** N.
- **Action:** cap both at 64 KiB in go-tuiMaps (a library change: new release candidate and re-pin per D-206), and extend `TestTheTileTransportMatchesTheLibrarys`. Or correct A-84's wording to "every `httpx` client".

**R2I-4: D-248's count of refused implausible readings has no test at the library level.**
- **What happens:** deleting `+ res.Implausible` survives every root-package test, so the snapshot's `Rejected` could lose the plausibility refusals unnoticed. D-248 requires them to be "refused and counted".
- **Severity:** Minor.
- **Evidence:** go-ionomaps `update.go:437`. The only test of the count is at the `assimilate` level (`internal/assimilate/onto_test.go`).
- **Simplify/Delete?** N.
- **Action:** add a `giro_update_test.go` case where a placed station reads past `FoF2Bound` and `Inputs.Rejected` counts it.

**R2I-5: The new `DaylightAt` reports `Known: true` for a place off the globe.**
- **What happens:**
  - NaN, latitude 1000 or longitude 1e300 give `Known:true, Day:false, Turns:zero`: "night, the sun neither rises nor sets".
  - Every other place-taking call in go-tuiMaps refuses such a place.
- **Reach:** watchpost is not exposed. Its one caller (`modes/tty/map_prop_words.go:124`) runs only after `cellValue` accepts the place (`:106-109`).
- **Severity:** Minor.
- **Evidence:** go-tuiMaps `daynight.go:38-43`; `internal/sun/sun.go:89` guards latitude only for `Turn`.
- **Simplify/Delete?** N.
- **Action:** return `Daylight{}` for a place off the globe, as `Places` refuses one.

---

##### The lens's questions

1. **Is every external input validated and bounded, or is the caller trusted?**
   - Mostly bounded:
     - GIRO's positions are pinned and its residuals bounded.
     - The solar file has a size and duplicate guard.
     - Reply headers are capped in `httpx`.
     - The geocoder's coordinates, zone and postcodes are checked.
     - A restored F10.7 mean is bounded by flux, days and clock (`internal/solar/solar.go:146-158`); its age is gated at 7 days (`update.go:208-214`).
   - Gaps: R2I-1, R2I-2, R2I-3, R2I-5, and RI-1's residual.
2. **Authorization on state-changing paths; are errors on destructive operations handled?**
   - No new network-facing surface.
   - `keepState`'s temp file is removed on failure and its errors are noted.
   - The history split is a pure move: code lines are identical apart from imports and comments. Clear's `RemoveAll` and `validSegment` are unchanged (`platform/history/history.go:775,799-810`).
   - The orphan sweep's remaining race is now stated (`domains/radio/synth/children.go:131-133`).
   - The new 24-hour GIRO denial (`internal/throttle/throttle.go:94-100`) is persisted. `Resume` accepts a pause up to `now+DeniedFor` (`:151`), so a local state file can hold GIRO off for a day. That needs write access to a 0600 file, and the behaviour is D-247's.
   - Nothing else found.
3. **Secrets out of logs, errors and source; crypto through vetted libraries?**
   - Nothing found.
   - `tools/finaltag` reads `"$GITHUB_REF_NAME"` as an environment variable, not through `${{ }}` interpolation (`.github/workflows/release.yml:23-24`), so there is no injection.
   - Actions are pinned by SHA.
   - No crypto of the project's own; TLS 1.2 is the minimum everywhere.
4. **Injection or traversal?**
   - R2I-2 (the zip drawn raw) is the only terminal path.
   - A zone name with `..` or a leading `/` is refused by Go before any file is read (verified).
   - GIRO asks still go through `url.Values`.
   - Otherwise clean.
5. **Do errors leak structure, paths or stack traces?**
   - No: RI-7 is fixed.
   - The new errors (`errTooLarge`, `errLine`, `errKept`, the off-globe and `notYet` messages) quote no input and no path.

##### Verified clean

- go-tuiMaps `main` `043922ec..50350d1`: `go 1.26.10` floor, version string, toolchain pin.
- go-ionomaps `Restore`: ordering leaves the library as it was on refusal.
- `Unasked`: keeps codes unique.
- 401/403/410/404/5xx handling: bounded and classified.
- `Path`: still bounded after the origin moved out of `hourFor`.
- `NearWorld`: band search is NaN-safe.
- go-tuiMaps remediation: the host-ramp re-check and `Canvas.Glyph`'s bounds.
- `propagation.NewService`: refuses a library that names no host.

##### Opinion

**Exit REVIEW.** Nothing is exploitable without impersonating a TLS-served upstream or writing to a local 0600 file, and every finding is a small change.

**Fix first: R2I-1.** It is the exact silent-truncation class round 1 named in RI-3, and it survived the fix. It is one condition plus a test.

## Performance (R2P)

#### Round-2 red team, Performance lens (R2P)

**Tree state.** At the start (`git status --short`):
- watchpost `feature/propagation-overlays` @ `5b71df2c`: ` M 06_docs/02_features/propagation-overlays/02-analysis/rulings.md`
- go-ionomaps `feature/discover` @ `8d84569`: clean
- go-tuiMaps `feature/propagation-fields` @ `73c602d`: ` M 06_docs/gate-runs.md`

While I worked, go-ionomaps also gained ` M 06_docs/gate-runs.md`. I did not cause it: another reviewer's gate run did. HEADs did not change. Every finding was reproduced against those HEADs.

I ran every gate and every program of my own on copies in `<scratch>`. Nothing was written into any tree. Timings were taken on a machine shared with the heap and G1b runs and six reviewers, so they are noisy. Allocation counts are the firmer figures.

**What I ran:**
- **Gates:**
  - go-ionomaps `scripts/gate --quick`: green.
  - go-tuiMaps `scripts/gate --quick`: every test, vet and race leg green. The P10 leg failed only because govulncheck could not resolve vuln.go.dev during the network outage; `a2dh p10 check` re-run afterwards passed with "no vulnerabilities".
  - watchpost `make lint`: no new findings. `make p10`: 0 live, 23 density rows at or over their floors.
- **Tests:** `go vet` and `go test -race` over watchpost geodata, propagation, history, propanswers, modes/tty (the prop tests), app: all ok.
- **Benchmarks:** go-tuiMaps `GlobeFieldPan` before and after RP-4's fix (the old render files on a scratch copy); go-ionomaps `Reach*` with a CPU profile; watchpost `BenchmarkPropRenderMap` and `BenchmarkNearWorldPanning`.
- **Programs of my own:** a world-latitude sweep of `NearWorld`; heap splits of the hours ahead; a fill count of the typical days; a probe of the typical-day eviction order; an 8-goroutine `-race` stress of the typical-day cache through evictions; a `DaylightAt` benchmark.

---

##### Part 1: round 1's findings (RP-)

**RP-1: PARTIAL.**
- **Fixed:** the clone-then-overwrite is gone (go-ionomaps `ahead.go:47` calls `withTables`, defined at `update.go:499`).
- **Fixed:** the pin exists. `ahead_ext_test.go:78-110` measures 10.8 MB.
- **Not fixed in practice:** typical hours are allocated per hour (`typical.go:37-46`), but the answers undo that (R2P-1).
- **What changed overall:** 11.4 MB in round 1 against 10.8 MB now, about 5%. An hour ahead still holds 0.192 MB of live heap, as before. The copies that were removed were garbage, never live heap.

**RP-2: PARTIAL.** A third day no longer empties the cache. But eviction now takes the oldest calendar day, not the least recently used one (`typical.go:79-81`, `oldest` at `:90-99`).
- **Probe:** asking D+1, then D+2, then D (a late answer of the 23:50 snapshot) evicts D+1, the current day.
- **Round-robin of D, D+1, D+2:** 21 of 30 asks still build a new day.
- **Cost of each miss:** about 3.1 MB and a full recompute, because of R2P-1.
- **In watchpost:** this happens once per transition, from an answer still in flight. It is Minor.
- **Action:** evict the least recently used day, or serve a day older than every kept one uncached (as `t == nil` does).

**RP-3: PARTIAL.**
- **Library half: FIXED.** `go-tuimaps globe_bench_test.go:17-30` builds a structured field, and 800×220 is among the sizes (D-261).
- **watchpost half: not done, and not ruled.**
  - `BenchmarkPropRenderMap` (`modes/tty/map_prop_alloc_test.go:49-55`) still redraws an unchanged view: 19.8–20.3 µs, 124 allocations. That measures the row cache, not a frame.
  - The pins still use a constant field (`map_prop_draw_test.go:20-31`).
- **Recommendation to the HUM LEAD:** either change the view on every op, or record that D-261 covers only the library's benchmark.

**RP-4: FIXED.** Evidence: `go-tuimaps internal/render/frame.go:1093` (`Grow`), `canvas.go:144` (`Glyph`), `basemap.go:556`. I measured the same benchmark on old and new render files:

| Pan frame | Before | After |
|---|---|---|
| 400×110 | 10.4–10.9 ms, 623 KB, 5,191 allocations | 7.56–7.62 ms, 384 KB, 4,201 allocations |
| 800×220 | 33.5–34.3 ms | 27.2–27.3 ms |

The vote's share of frame CPU fell from 27% to 4.9%. Night ownership is still recomputed for each cell of a changed frame, but it is now cheap. This was not the per-version cache round 1 proposed, but it is effective.

**RP-5: NOT FIXED, and never dispositioned.** `red-team-review.md:112` still reads "Not yet checked". No ruling, A-row or batch mentions it. The code is unchanged:
- `go-ionomaps answers.go:395-408`: `sunAt` still scans back 144 steps.
- `internal/magcoords/magcoords.go:112-121`: the year fraction is worked out again at every control point.

Profile now:
- `sunAt` is 13.4% of the reach's CPU;
- `yearTable.value` is 10.9%;
- `yearFraction` is 6.8%.

`BenchmarkReachFirst` takes 273 ms, 7.5 MB and 48k allocations; `ReachKept` 160 ms. It is Minor (off the UI goroutine), but it needs a disposition.

**RP-6: FIXED.** Evidence: `watchpost domains/locations/geodata/near.go:101-111` (the index by latitude), `:138`, `:154` (`inBand`).
- **Worst latitude of the world sweep (41°N):** 76 µs a step, 58 KB, 45 allocations. Round 1 measured 1.05 ms.
- **Mid-latitude ±0.9° band:** the benchmark gives 60 µs.
- The bound is sound: a great circle is never shorter than its latitude part, plus 0.01°.

---

##### Part 2 and 3: new findings

**R2P-1: The typical day's "hours allocated as asked" (D-251) is defeated by every answer, so 24 hours ahead cost almost what they did in round 1.**
- `typicalDay` opens all 24 hours (`go-ionomaps openhours.go:38-47`; `openHour` calls `kept.hour(h)` for each h).
- Every `Bands` (`answers.go:223`), `Path` (`answers.go:254`) and `Reach` (`reach.go:54`) calls it.
- So each answer allocates 24 × 16,200 cells, 3.1 MB per day, while a chart reads almost none of them.
- **Measured:**
  - A chart (Bands, plus Path to 6 targets) filled **1,032 of 388,800 cells (0.27%)**. A reach filled 52,992 (14%).
  - 24 hours ahead not answered: 7.77 MB. Answered: 8.63 MB more. The pin: 10.8 MB.
- watchpost answers the chart on every snapshot it shows, including each hour ahead. In the host's real use, both days' 48 hours are always made, which is round 1's 6.2 MB.
- The pin's own doc says "made only for the hours read" (`ahead_ext_test.go:72-77`), yet its loop calls `Bands` on every hour.
- **Severity: Important.** This is the remaining half of round 1's only Important finding, and the record says it is fixed. It is within budget today.
- **Simplify/Delete?** N.
- **Action:** open hours lazily inside `typicalDay` (on the first read of each hour), or store a typical hour sparsely (per latitude-band block). Then re-run the pin with the answers counted.

**R2P-2: The 24-hour pin bounds the heap the hours ahead add, not D-112's measure (the library's peak live heap of at most 15 MB), and G1 still never holds more than one hour ahead.**
- The pin takes `added := after - before` once the snapshot exists and fails at `added > 15<<20` (`go-ionomaps ahead_ext_test.go:89-104`).
- Round 1 measured a 2.5 MB baseline on real data. The real peak is therefore about 13.3 MB. The pin would pass a library peak near 17.5 MB.
- G1's use arm still steps one hour ahead and back with `n` (`watchpost scripts/quality/workload.expect:73-75`).
- **Severity:** Minor. **Simplify/Delete?** N.
- **Action:** measure from before `ionomaps.New` (the fixture excluded), or bound `added` at 15 MB minus the measured baseline. Or record that D-251's pin means "added".

**R2P-3: A hung GIRO now holds every GIRO burst, and the library's lock, to the full 60 s update deadline, and D-247's "three failures in a row" never fires on timeouts.**
- watchpost's propagation client sets no `Timeout` (`watchpost app/mapprop.go:37`), so httpx defaults to 30 s a request (`platform/httpx/plain.go:79-80`).
- The update deadline is 60 s (`domains/propagation/propagation.go:45`).
- A burst now continues past a station's failure. It stops only at `failuresInARow = 3` (`go-ionomaps internal/giro/asker.go:27`, `:183-184`). But three timeouts take 90 s, more than the 60 s deadline.
- **Result when GIRO hangs:**
  - The second timeout meets the deadline and the burst ends as `cancelled` (`asker.go:228`).
  - The update is Unfinished, not Unavailable, and Outcome stays Answered.
  - The lock is held 60 s instead of the 30 s before D-247.
- This is off the UI goroutine, and the hour's budget is spent, so there is no extra load on GIRO.
- **Severity:** Minor. **Simplify/Delete?** N.
- **Action (HUM LEAD fork):** choose one:
  - a GIRO per-ask timeout with `failuresInARow` × timeout under the deadline (for example 15 s);
  - count a timeout as GIRO down at once.

  Add a fake test with slow failures.

---

##### The lens, question by question

1. **I/O in a loop, or repeated work that could be memoized?**
   - No network N+1 in the new code. GIRO's per-station loop is the ruled burst.
   - RP-5's repeated per-point work stands (NOT FIXED).
   - `time.LoadLocation` runs once per geocoder search (`openmeteo/geocode.go`). That is one per user action, so fine.
   - `recheckHostRampsLocked` (go-tuiMaps `overlays.go`) runs only on a real look change. `ColourDepth` with the same hint returns early (`look.go:226-231`), so it does not run each frame.

2. **Unbounded growth?** Nothing unbounded:
   - typical days are capped at 2;
   - hours ahead at 24;
   - the history day cache at 256 entries / 8 MB (`platform/history/cache.go:100`);
   - the solar file at 256 KB;
   - response headers at 64 KB.

   What is bounded but oversized is R2P-1, and the pin's reference point is R2P-2.

3. **Complexity worse than needed?**
   - RP-6's scan is fixed.
   - `Unasked` and `Apply` are O(n²) over n ≤ about 60 stations, and `Readings` sorts at most 40. Both fine.
   - `sun.Turn` (behind `DaylightAt`) costs 4.4 µs at mid-latitude and 35 µs in polar night. It runs only in the description, so fine.

4. **Blocking work on a latency-sensitive path?**
   - Nothing new on the UI goroutine. Answers, hours ahead and updates run in commands.
   - The new burst behaviour lengthens the update (R2P-3), off the UI path.

5. **Shared state under concurrency; cache invalidation?**
   - Race-clean: `sync.Once` per hour, plus atomics, through evictions (my 8-goroutine `-race` stress).
   - The `propMade` key (`map_prop_update.go:78-95`) carries computed, built and options. The hours ahead and `propAheadAsked` are reset on every rebuild (`:155-158`). A late hour ahead of a rebuilt snapshot is dropped (`map_prop_ahead.go:68`).
   - go-tuiMaps' `turnPassed` cannot thrash, because `sun.Turn` always returns a time after its moment (`internal/sun/sun.go:98-107`).
   - The one invalidation weakness is RP-2's eviction order.

6. **Baseline realistic?**
   - The frame benchmark is now realistic in go-tuiMaps.
   - watchpost's is not (RP-3).
   - The 24-hour pin measures the increment, not the peak (R2P-2).
   - G1 still holds one hour ahead.
   - The re-measure in `m5-frames/d261-remeasure.md` sets the new benchmark beside round 1's real-field figures, so it is not a like-for-like comparison. My same-benchmark comparison above confirms the gain.

**Verified clean:**
- RP-4's and RP-6's fixes;
- the history split as a move (bounds intact, tests green);
- `propMade` keys and the answer arming (same semantics as before the split);
- go-tuiMaps `DaylightAt`, `turnPassed` and the host-ramp recheck cost;
- watchpost's header cap and validator cap;
- concurrency of the typical-day cache;
- watchpost `make lint` and `make p10`;
- both libraries' quick gates (go-tuiMaps' P10 passing once re-run after the outage).

**Opinion: exit REVIEW.** No Critical findings. Everything sits inside D-112, and the fixes that landed (RP-4, RP-6) are real and measured. **Fix R2P-1 first:** make `typicalDay` open hours as they are read, or store them sparsely. That turns D-251's claimed fix into a real one, roughly 6 MB off the 10.8 MB of 24 hours ahead. Batch with it R2P-2's pin reference and a disposition for RP-5, which round 2 found unaddressed and never ruled.

