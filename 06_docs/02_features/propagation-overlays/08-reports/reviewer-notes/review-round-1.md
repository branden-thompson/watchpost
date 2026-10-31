---
title: "0.19.0 REVIEW exit red team, round 1 - reviewer notes"
date: 2026-10-24
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "Seven fresh blind reviewers (D-245), each report filed in full as its final message gave it; only each reviewer's scratch folder is shortened to <scratch>. Every finding cites the file and line it read at watchpost 6ad1c098, go-ionomaps d80d8db7, go-tuiMaps 544b889d. The coordinator's check of each finding is in the report's table."
---

# REVIEW exit - red team, round 1: the reviewers' reports

Up: [the report](../red-team-review.md)

Prefixes: RC (Code Quality), RH (Project Hygiene), RD (Docs Quality), RB (Business Quality), RL (the REVIEW lens), RI (InfoSec), RP (Performance). The brief, the same for each but its lens, is `06_docs/red-team-brief.md`'s template.

## Code Quality (RC)

#### Code Quality review, 0.19.0, round 1 (RC-)

**Verdict: do not exit REVIEW.** I found no Critical. Six Important findings are live defects or verifiers that pass without verifying: RC-1, RC-2, RC-3, RC-4, RC-11 and RC-14 (RC-14 breaks L-4.4, a requirement). **Fix RC-4 first.** It is the BUILD red team's L-F3 class (an answer kept against a snapshot that has since been rebuilt). The fix was applied to the chart, centre and reach answers but not to the hours ahead. I reproduced it.

**Tree state.** `git status --short` was empty in all three repos at the start and at the end: watchpost 6ad1c098, go-ionomaps d80d8db7, go-tuimaps 544b889d. Every finding was reproduced against those tips. For the repros:
- RC-1 ran on a run directory I built in scratch.
- RC-4 ran in a copy of watchpost under `.../scratchpad/rt-review-r1/code/wpcopy`.
- I split the work with two forks of myself, one per library. They ran the libraries' gates on copies, because each `scripts/gate` run appends a line to a tracked file (see RC-12). Library findings marked "fork" come from them, and I re-read the key lines myself.

**Runs.** All of these exited 0:
- watchpost: `make lint` (6 baselined), `make lint-authoring` (972 files, no findings), `make p10` (0 live, 0 unmatched, 0 unratified), `make dupes` (3 groups, all ratified), `make mutant-anchors` (780), `go vet ./...`.
- watchpost, `go test -race`: modes/tty (214 s), app (206 s), platform/history, domains/propagation, platform/propanswers, platform/httpx, platform/agememo, domains/radio/synth, and tools perfsum, runlog, prepush, prehook, reachscore.
- watchpost, record tests in cmd/watchpost.
- Both libraries: `scripts/gate --quick` green (fork).

##### 1. Understandability
- **RC-10 (Minor):** `platform/history/history.go` is 2,349 lines covering six concerns: the store, two caches, pruning, roll-ups, claims and number encoding. Evidence: watchpost `platform/history/history.go:154-2349`. Simplify: Y. Action: split it into files along those lines.

##### 2. Maintainability
- **RC-4 (Important):** an hour ahead worked out from a snapshot is kept for a different snapshot.
  - The answer is accepted on `Computed` alone (watchpost `modes/tty/map_prop_ahead.go:66`). The chart, centre and reach keys carry `built` precisely because a rebuild keeps the same `Computed` (`map_prop_chart.go:22`, `map_prop_freq.go:55`).
  - Repro test: ask an hour ahead, rebuild the snapshot with another noise, then answer the ask. The rebuilt snapshot's noise is 3 and the kept hour ahead's is 0, and it stays until the next rebuild.
  - Simplify: N. Action: key the hours ahead on `built`, and add the test.
- **RC-5 (Important):** the Propagation mode's state is scattered, which is what made RC-4 possible.
  - `propSession` is the state "dropped whole" when the mode is left (C-3), but it holds 5 fields (`map_prop_update.go:37-43`). About 35 more `prop*` fields live on `mapPane` (`map_pane.go:65-107`): the asked keys, the find, the hours ahead.
  - "Made with the same options" is written three ways: `armPropOptions` (`map_prop_update.go:79`), `sameSnapshot` (`:147-149`) and the `correct` check (`:117`).
  - The next new option has to be added in four places.
  - Simplify: Y. Action: move all session-scoped state into one session struct, and use one options/snapshot key type that every answer key embeds.

##### 3. Complexity
- **RC-10 (cont.):** `propAnswer` and the `armAnswer` switch exist, by their own comment, because "the duplicate check refuses" three separate functions (`map_prop_chart.go:72-76`). The D-227 special cases sit inside the shared body (`:89-95`). That is a structure shaped by the metric, not by the requirement. Simplify: Y. Action: three arm functions, each calling `askAnswer`, with one shared "settled" check.

##### 4. Necessity
- **RC-7 (Minor). Each of these can be deleted, all in watchpost; Simplify: Y for all:**
  - `propagationHostList` (`app/mapprop.go:123-132`) has no production caller; only `mapprop_test.go:110` uses it.
  - `propagationSources` (`app/mapprop.go:134-136`) is a one-line wrapper over `ionomaps.Sources`.
  - `synth.LiveChildren` (`domains/radio/synth/children.go:89-99`) is exported production code used only by tests. Move it to an `export_test.go`.
  - `hasKey` (`map_prop_place.go:191-199`) is `slices.Contains`.
  - `ackDue` (`map_prop_ack.go:46`) is a one-line wrapper with one caller.
  - `Service.Settle`'s bool result is never read (`domains/propagation/propagation.go:192`, `app/mapprop.go:66`).
  - In runlog, `errNoGitCommand` and the repeated lane and argument checks can never fire (`tools/runlog/main.go:30-32, 53-55, 73-75, 130-132`). They are padding, the kind of check D-215 removed elsewhere.
  - `stillOrphan`'s second process read (`children.go:154, 170-175`) adds nothing to the start-time match. The window it claims to close is still open between that read and `End`.
- **Library deletions (fork), Simplify: Y:** go-ionomaps `lower.go:381, 452` (one-line wrappers), `lowergrid.go:60-62` (a branch that can never run), `reach.go:47` (the `polar` parameter exists only for a test) and `:56` (a map holding two counters), and the exported `ionomapstest` `Recorded.Solar` (`recorded.go:193`). go-tuiMaps `frame.go:1125` (dead branch).
- **RC-16 (Minor), go-tuiMaps (fork):**
  - `Daylight` and `sun.Turn` have no consumer. watchpost re-implements the same thing in `platform/astro/turn.go:16-67` with a different sun model.
  - Its comment (`turn.go:10-13`) claims to match go-tuiMaps' terminator, and nothing tests that.
  - Recommendation to the HUM LEAD: keep one implementation.

##### 5. Name semanticism
- **Minor:**
  - `hasKey` actually checks a sentence's kinds (`map_prop_place.go:191`).
  - `answered` also redraws the map (`map_prop_chart.go:158`).
  - From the ionomaps fork: `aheadcache.go` holds only `lastBackground`; `finite` also range-checks (`answers.go:430`); `frac` holds two different quantities in turn (`answers.go:277`).
  - The mechanical SN-01 checks are clean (lint-authoring green).

##### 6. Code documentation
- lint-authoring is green, and I found no AP-HIST/SN-02 case it missed in watchpost. The lint is not off.
- **RC-8 (Minor). Comments that are wrong:**
  - "A paired library's test is checked in its own repository" (watchpost `cmd/watchpost/record_test.go:76-77, 252`). No such check exists in either library; I grepped both.
  - "A process ID reused between the two is never ended" (`children.go:131-133`) overclaims; the window remains.
  - `settle` uses up the `sync.Once` (`app/mapprop.go:63-64`), so the service can never be made afterwards. Only `Ask`'s nil check (`propagation.go:171`) keeps that safe, and neither comment says so.
  - From the forks: go-ionomaps `snapshot.go:141` (TooSoon is not "the hour's budget is spent") and `state.go:16` (a failed Restore leaves the library as it was, not as new); go-tuiMaps `internal/colour/preset.go:22-25` ("one edit" took about 12 sites) and `facts.go:174`.
  - go-tuiMaps has no AP-HIST lint, and a history comment is live in `mvt/feature_test.go` (fork).

##### 7. Testability
- Everything is testable without heavy scaffolding.
- Gaps: RC-4 has no test. From the forks, RC-11 and RC-14 are untested, and so is RC-15 (go-tuiMaps `hostramp_test.go:19-26` always sets the depth before `Set`).

##### 8. P10 conformance
- **RC-6 (Important). Package-wide P10-05 exemptions now cover code they never described:**
  - The `modes/tty` density row was "ratified by precedent 2026-09-08" with a reason about the Dashboard's renderers (`.a2dh-p10-exemptions.yml:339-347`). It now exempts density 0.43 over 331 changed functions, essentially all of the Propagation mode. The `app` row (`:329-335`) exempts 0.59 over 51.
  - The ledger lint says itself that it "cannot see" a row whose code has changed.
  - go-ionomaps (fork): all 16 rows are package-wide P10-05, and only the root's figure is pinned (`exemption_test.go:24`).
  - P10 scans only the diff against the merge base (`scripts/quality/p10-unmatched.sh:14-19`; 131 rows dormant). That contradicts the "artefact, not session" rule the Makefile states for lint-authoring (`Makefile:83-88`).
  - Simplify: N. Recommendation to the HUM LEAD: re-present every package row whose function count grew, and pin each measured figure as a floor.
- go-ionomaps `tools/tables` loops are bounded only by their input (`igrf.go:63`, `main.go:45`) and P10 never scans that package (RC-12).

##### 9. Evidence soundness
- **RC-1 (Important):** `perfsum -g1` passes a run it cannot judge.
  - Reproduced exit 0 on a 2-hour run with a blank `rss_kb` (read as 0 MB, so "added RSS −500, met") and a WARN on every cycle.
  - A run shorter than 48 hours only adds a sentence (`tools/perfsum/g1.go:119-139`).
  - A bad number becomes 0 (`main.go:265`).
  - The driver's "not seen drawn" WARNs are never read (`scripts/quality/workload.expect:64`).
  - `workload.sh g1` exits 0 when an arm's driver fails (`:137`).
  - Simplify: N. Action: fail on a short run, on an unparsable column and on any WARN or FAILED phase; propagate a driver's failure.
- **RC-2 (Important):** `make test-say`, a required step, skips green on hosts whose `say` renders nothing.
  - `domains/radio/synth/synth_test.go:215-217, 231-233` skip, contradicting the test's own lines 182-184: "a skip here is a required step reporting green with nothing run".
  - By its own comment, GitHub's macOS runner always produces about 236 bytes, so CI never observes the property.
  - The recipe has no `-v`, so the skip is invisible.
  - The disposition was the agent's, not ratified (`06_docs/follow-ups.md:173`, F-52).
  - Simplify: N. Action, a recommendation to the HUM LEAD: fail, or report a distinct NOT RUN that the gate refuses.
- **RC-3 (Important):** runlog records the tree as it stands at the end of the run, not the tree the run tested.
  - `record` runs after the command (`tools/runlog/main.go:43-44`), yet the header claims the line "says exactly what passed" (`:26`, `:102`). An edit made during the 35-minute verify gets a green line.
  - A failure to record exits 0 (`:44-47`).
  - Simplify: N. Action: take the tree before and after the run, and record a mismatch as not green.
- **RC-9 (Minor):**
  - reachscore prints a met/missed verdict even after the All row printed "no figure stands" (`tools/reachscore/report.go:104-107` against `:111-115`).
  - The run-selector check counts a tagged test as selected even when the recipe omits the tag (`cmd/watchpost/gates_test.go:719-720`). Dropping `-tags property` from `make property` would select nothing and still pass.
- **RC-11 (Important), go-ionomaps (fork; I re-read the line):** when GloTEC's index names no grid, the update returns `Fresh` with nothing in `Unread` (`update.go:230-232`). That breaks R-5.4. The fork reproduced 88 minutes of Fresh while the background fell to the climatology. Action: name GloTEC in `Unread`, and add the reformatted-index case to `TestEveryFaultIsReported`.
- **RC-13 (Minor), go-ionomaps (fork):**
  - The P10 mirror test skips (reported as a pass) where the ledger is absent (`p10_mirror_test.go:109`), and its symbol check matches any mention of the name (`:168`).
  - The golden-file parser discards `Atoi` errors (`reproduce_test.go:77-79`).
  - `MeanResidual` depends on map iteration order (`internal/giro/asker.go:211-217`).
  - `Path` accepts 63 destinations while its error says 64 (`answers.go:247`).
- **RC-14 (Important), go-tuiMaps (fork):** `Daylight.Turns` can lie in the past. The kept report is keyed only on day/night bits (`describe.go:76-88`, `report.go:366`). This breaks L-4.4.
- **RC-15 (Important), go-tuiMaps (fork):** L-2.3's host-ramp warning is checked only on `Set` (`overlays.go:306`). Changing the depth afterwards draws a breaking ramp with no warning.
- **RC-17 (Minor), go-tuiMaps (fork):**
  - The specimen test never reports a stale `.ans` file (`specimen_test.go:105-129`).
  - The README table omits the globe-muf and globe-fof2 specimens.
  - `classTokens` silently falls back to radar's tokens for an unknown preset (`blend.go:261`).

##### 10. Evasion lens
- watchpost: nothing found. `verify` is treelock, then runlog, then `$(MAKE) verify-gates` in the same Makefile, with the exit code preserved (`tools/runlog/main.go:47`). There are no sourced scripts, no PATH re-export and no CI predicate. CI's step list matches `verify-gates`.
- **RC-12 (Important), go-ionomaps (fork; I re-read the lines):**
  - `export GOTOOLCHAIN="${GOTOOLCHAIN:-$FLOOR_TOOLCHAIN}"` (`scripts/gate:81`) lets the environment override the floor toolchain. Under setup-go, CI runs 1.27.3, not go.mod's 1.26.10.
  - The P10 leg reports ok while a2dh SKIPPED `tools/tables` (`:260`).
  - Every run appends to the tracked `06_docs/gate-runs.md` (`:182`), which is F-155's hazard.
  - Action: pin the toolchain unconditionally, and fail on SKIPPED.
- go-tuiMaps: nothing found.

**Fix first: RC-4.** Then RC-14 and RC-11, which are live wrong outputs, and then RC-1, RC-2 and RC-3, the verifiers that report success without proof.

## Project Hygiene (RH)

#### RH: Project Hygiene review of 0.19.0 (blind, round 1)

**Tree state.** `git status --short` was empty in all three repos at the start and again at the end: watchpost `6ad1c098`, go-ionomaps `d80d8db7`, go-tuiMaps `544b889d`. Every run was done in fresh `git clone`s of those tips under `<scratch>/rt-review-r1/hygiene/`. Unless a finding says otherwise, it was reproduced against those clean tips.

**What I ran.** All runs used go1.27.3 with GOTOOLCHAIN=local.
- watchpost clone: `make build` passed. `go build ./cmd/watchpost` with an empty GOMODCACHE and GOCACHE passed; both library rc tags come from proxy.golang.org.
- watchpost gates: `make lint-authoring` OK (972 files). `make dupes` 3/3 ratified. `make mutant-anchors` 780 OK. `make lint` gave no new findings. `go vet ./...` clean.
- watchpost `make p10`: fails in a clean clone ("no .a2dh.yml"). It goes green (0 live / 0 unmatched / 0 unratified, 163 rows) once the two ignored local files are copied in.
- watchpost race tests: `go test -race` passed on domains/propagation, platform/{propanswers,history,httpx,geo,render}, tools/{reachscore,prepush,perfsum,runlog,prehook}, cmd/watchpost, modes/tty, app and domains/radio/synth.
- `scripts/gate --quick` was green in both library clones, with P10 NOT RUN.
- `gh` checks: hosted CI was green on 95b81078 (watchpost; 6ad1c098 was in progress and is docs-only), and full gate dispatch was green on d80d8db7 and 544b889d.
- Record checks: every watchpost commit since `tools/runlog` existed (3c45707b) has a green verify row for its exact tree. Every library commit back 14 has a gate row, except RH-10.

---

##### 1. Is anything in the tree that should not be?

Nothing. I scanned every added line in all three diffs for IP addresses, emails, tokens and home paths. The only hits were 127.0.0.1, the documentation ranges 192.0.2/203.0.113, and `example.invalid`. The committed request logs carry metadata only; the CDN location headers are stripped and there are no reply bodies. `internal/stations/seed.txt` is station metadata, as ruled (D-51). The remote branches are just `main` plus each feature branch, and there are no stashes and no extra worktrees. The ignored build artefacts `watchpost/geodata.test` (Sep 29) and `go-tuimaps/go-tuimaps.test` are untracked, so I raise no finding.

##### 2. Does the build work from a clean clone, with the documented commands?

watchpost builds and the library quick gates pass. Four problems:

- **RH-1. P10 cannot run from a clean clone, and the README does not say why.** `make p10` stops with "cannot locate the project root: no .a2dh.yml". The gate's inputs, `.a2dh.yml` and the 163-row `.a2dh-p10-exemptions.yml`, are git-ignored (`watchpost/.gitignore:15` `.a2dh*`). The README names only the `a2dh` CLI as the requirement (`watchpost/README.md:523-524`). The same applies to go-ionomaps (16 rows) and go-tuiMaps (23 rows). The committed `06_docs/p10-ledger.md` mirror is the only copy in version control.
  - Severity: Minor. Evidence: `watchpost/Makefile:531-534`, `.gitignore:15`, `README.md:523`. Simplify/Delete: N.
  - Action: name both files in the README requirement. Keeping the ledger outside the tree is ratified (REVIEW red team 2026-09-17), so I am not disputing that.
- **RH-2. go-ionomaps' README documents no way to build or test the library.** Its only sections are How it works, Using it and Licence. `scripts/gate` documents itself in its header, but the README never points to it. go-tuiMaps' README has a "Working on the library" section.
  - Severity: Minor. Evidence: `go-ionomaps/README.md:1-78`. Simplify/Delete: N.
  - Action: add a short section pointing at `scripts/gate`.
- **RH-3. The tmbench evidence cannot be re-run as its README describes.** `tmbench/` has a `go.mod` and no `go.sum`, so `go vet .` fails with "missing go.sum entry". The README says it "needs go-tuiMaps v0.2.0, from the module proxy".
  - Severity: Minor. Evidence: `go-ionomaps/06_docs/02_features/go-ionomaps/02-analysis/evidence/plan-dry-run/README.md:11`, `.../tmbench/go.mod`. Simplify/Delete: N.
  - Action: commit the `go.sum`.
- **RH-4. The status line in go-ionomaps' README is stale.** It says "v0.1.0, at release candidate 3", but rc.4 and rc.5 are tagged and watchpost pins rc.5.
  - Severity: Minor. Evidence: `go-ionomaps/README.md:8`. Simplify/Delete: N.
  - Action: correct it, or drop the rc number.

##### 3. Are the gates actually run on every path that claims to run them?

Mostly yes: watchpost CI matches `verify-gates` plus `required-gates.txt`, the mutant policy is `push` and ran on Linux, and the release workflow runs `make verify`. The gaps:

- **RH-5. P10 runs on exactly one machine, and the library gates go green without it.** go-ionomaps' `scripts/gate` says it "stands before every commit" with "P10 still runs" in quick mode. In practice it reports "green without P10" wherever `.a2dh.yml` is missing, and that includes every hosted run.
  - Severity: Minor, because it is disclosed in the gate output and in go-tuiMaps D-136. Evidence: `go-ionomaps/scripts/gate:7-11,258`, `go-tuimaps/scripts/gate:793-798`, hosted logs ("P10: NOT RUN, a2dh is not on PATH"). Simplify/Delete: N.
  - Recommendation to the HUM LEAD: record P10 at each phase exit in a gate roster (see RH-6), so "P10 clean" points to a dated run.
- **RH-6. 0.19.0 has no gate roster, though the README says P10 results are recorded in one.** Every earlier release has `07-readiness/gates.md`; 0.19.0 does not, and it adds gate tooling (`tools/prepush`, `tools/prehook`, `tools/runlog`, `make hooks`). BUILD-exit P10 is recorded only in the build report's status line. 0.18.0's roster still reads "REVIEW-exit P10 to come" and "its result is recorded here", but 0.18.0's REVIEW P10 went into `review-report.md:84` instead.
  - Severity: Minor. Evidence: `watchpost/README.md:524`; `06_docs/02_features/observer-maps/07-readiness/gates.md:7,40-41`; no `propagation-overlays/07-readiness/gates.md` exists. Simplify/Delete: N.
  - Action: create 0.19.0's roster and fix 0.18.0's status line.
- **RH-7. go-tuiMaps' `main` nightly has been red since 2026-10-21, and go-ionomaps has no nightly at all.** The cause is GO-2026-6809: `main`'s gate uses go1.27.2 and floor go1.26.9. The fix exists only on the feature branch, so until merge a new failure on `main` would be hidden by the old one. go-ionomaps only runs on push, pull request or dispatch, so a new standard-library advisory would surface only on the next push.
  - Severity: Minor. Evidence: `go-tuimaps` `main:.github/workflows/gate.yml:45` and `main:go.mod:3`, runs of 10-21 to 10-24 failing; `go-ionomaps/.github/workflows/gate.yml:8-12`. Simplify/Delete: N.
  - Action: add a scheduled scan to go-ionomaps; accept or fix the red `main` explicitly.
- **RH-8. The pre-push hook gives an error with no working fix on any branch that lacks `tools/prepush`.** On `main`, or on a release branch of the 0.18 line, it says "the hook belongs to another clone: run make hooks in this one". `make hooks` does not exist there. The 0.18.2 release branch was "pushed past the pre-push hook" for this reason.
  - Severity: Minor. Evidence: `watchpost/tools/prehook/main.go:36-38`; `propagation-overlays/04-development/build-log.md:736`. Simplify/Delete: N.
  - Action: say that the tree predates the hook and how to push past it deliberately (D-124: errors need a path).
- **RH-9. Nothing stops 0.19.0 from shipping on the libraries' release candidates.** D-3 says watchpost "ships on its final tag". `go.mod` pins `v0.1.0-rc.5` and `v0.3.0-rc.3`, and the tag-triggered release runs `make verify`, `release-matrix` and `install-test`, none of which refuses a pre-release dependency. go-ionomaps' own checklist says "The machine check: None", whereas go-tuiMaps has `--release`, which refuses a final tag while any checklist row is unticked.
  - Severity: Minor. Evidence: `watchpost/go.mod:9-10`, `.github/workflows/release.yml:24-28`, `rulings.md:24` (D-3), `go-ionomaps/.../07-readiness/release-checklist.md:9`. Simplify/Delete: N.
  - Action: add a test that fails a final tag when `go.mod` requires any `-rc` module (0.18.0 D-11: a lesson that can be a test becomes one).

##### 4. Is anything named as owed that has quietly stopped being true?

No TODO, FIXME or XXX in the code of any of the three repos. Every P10 exemption row names a file and symbol that still exist. Every follow-up row moved off 0.19.0 names its new target (0.19.1 under D-122). Three problems:

- **RH-10. The re-tag and re-pin owed under D-206 were skipped by the agent's own reasoning, and the result contradicts D-244.** D-206 says "each later library change takes a new release candidate and a re-pin". The floor commits `d80d8db7` and `544b889d` took neither; the build log says "watchpost needs no re-pin". D-244 says every built module's `go` directive is 1.26.10, but the tags watchpost actually builds still say `go 1.26.9`. As a result, the reviewed library tips are not what watchpost builds.
  - Severity: Minor, since there is no practical effect: watchpost's own `go 1.26.10` governs. Evidence: `watchpost/06_docs/02_features/propagation-overlays/04-development/build-log.md:732`, `rulings.md:227` (D-206) and `:265` (D-244), `git show v0.1.0-rc.5:go.mod` → `go 1.26.9`. Simplify/Delete: N.
  - Action: the HUM LEAD either ratifies the exception or rc.6 / rc.4 are tagged and re-pinned.
- **RH-11. Two "owed" wires rows name a batch that shipped three releases ago.** `Event.Restored` and `State.Refused` are owed to "P4". That is 0.16.0's P4 (`p4-build-log.md`), and 0.16.0, 0.17.0 and 0.18.x have all shipped. The rows are still unwired, and the ledger says "Every row here is OWED, not accepted".
  - Severity: Minor. Evidence: `watchpost/06_docs/wires-ratified.md:31-32`. Simplify/Delete: Y, if the members are dead: delete them or re-own them.
  - Action: put to the HUM LEAD.
- **RH-12. Neither library's release checklist ticks BUILD exit, though BUILD exited on 2026-10-18 (D-233).** The checklists' rule is that a row is ticked "in the commit that holds its evidence".
  - Severity: Minor. Evidence: `go-ionomaps/.../07-readiness/release-checklist.md:16,25`, `go-tuimaps/.../propagation-fields/07-readiness/release-checklist.md:25`. Simplify/Delete: N.
  - Action: tick them with the evidence.

##### 5. Does the record say what happened, or what somebody intended?

Mostly what happened. The BUILD-exit claim (CI green on `e889e72a`) checks out against `gh`, and the verify and gate logs match their tree hashes. Two gaps:

- **RH-13. One go-ionomaps commit changed the gate without a gate run on its tree.** `8f6c085` changes `scripts/gate`. The only nearby row is 13 s before the commit and tested `ef44f04dd713`, which is not the committed tree minus the log. Its build-log text ("48 s here on the gate's run") was evidently edited after the run.
  - Severity: Minor. Evidence: `go-ionomaps/06_docs/gate-runs.md:48`. Simplify/Delete: N.
  - Action: note it in the record; the tips since then are gated.
- **RH-14. Several front-matter headers describe an earlier phase.** watchpost's `build-log.md` says "phase: BUILD … IN PROGRESS", `rulings.md` says "BUILD exit", and go-ionomaps' `rulings.md` says "DISCOVER (intake)". `p533-equations.md` still says "Still to settle for G6", although G6 is built (`gyro.bin`).
  - Severity: Minor. Evidence: `watchpost/.../04-development/build-log.md:4,7`, `.../02-analysis/rulings.md:4`, `go-ionomaps/.../02-analysis/rulings.md:4`, `.../evidence/p533-equations.md:49`. Simplify/Delete: N.
  - Action: batch-correct them.

---

**Opinion: exit REVIEW** on hygiene grounds. I found nothing Critical or Important: the trees are clean, the clean-clone build works, and the gate logs are honest and tree-matched.

**Fix first: RH-9.** A one-line test that refuses `-rc` modules at a final tag is the only thing between SHIP and a v0.19.0 built on release candidates, against D-3.

## Docs Quality (RD)

#### Docs Quality red-team report: REVIEW exit, round 1 (D-245)

**Tree state.** At the start and at the end of my pass, `git status --short` was empty in all three trees: watchpost `6ad1c098`, go-ionomaps `d80d8db7`, go-tuimaps `544b889d`. Every finding below was reproduced against those clean tips. One exception happened during the pass. `make verify-docs` appended a row to the tracked file `06_docs/verify-runs.md` (see RD-12). I removed only that row, and the tree was clean again afterwards.

**What I ran.**
- `go doc` / `go doc -all` on both libraries' public APIs, and on `ionomapstest`.
- I compiled both go-ionomaps README snippets in my scratch module (pinned to rc.5, with a replace to the local tree). Both compile.
- `make build` into scratch, then `--help`.
- `make lint-authoring`: exit 0.
- `make verify-docs`: it refused, because the tree was unchanged.
- The libraries' docs-lane leg (`go test -count=1 ./...`): green in both. I ran it directly, because `scripts/gate --docs` with nothing staged fails and logs to `gate-runs.md`.
- `go run ./tools/gm3 -h`, `./tools/perfsum -h` and `./tools/reachscore -h`: all match their docs.
- I checked every test name in the three `requirements.md` files against the trees' `func Test…`. All exist except the three below; two of them are L-5, which is ruled out.
- I checked the README key tables, Settings labels, hosts and the Go floor against the code. All match except where a finding says otherwise.

---

##### 1. Does each document answer the question its title asks?

**RD-13 — `00-REQUIRED-READING.md` does not tell a reader what 0.19.0 is before it hands them process.**
- The task names it as the place to start. It opens with the branch, the cut point, then a Phase cell of roughly 250 words: G1, G-M3, M1/M4, A-83, heap, the red team.
- No sentence anywhere in it says what the listener gets. `requirements.md:12` does this right ("What the listener sees").
- Severity: Minor.
- Evidence: watchpost `06_docs/02_features/propagation-overlays/00-REQUIRED-READING.md:17-22`.
- Simplify/Delete? Y: split the Phase cell into a short list.
- Action: this is a read-order fork for the HUM LEAD. Option A: add a three-line "what this release is" above "Where things are". Option B: keep it as is and point designers and PMs to `requirements.md` first.

Otherwise each document answers its title. One label lags: `build-log.md` is titled "BUILD log" (phase: BUILD) but now records REVIEW batches 53-58. That is cosmetic.

##### 2. Does anything contradict the code, or another document?

**RD-4 — FR-1.14 says moving between targets is a keymap action. In the code it is hard-coded.**
- The chart window moves its target on literal `up`/`down` keys.
- The named instrument checks only `P`, `L`, `U` and `/`.
- Severity: Minor.
- Evidence:
  - watchpost `requirements.md:82`
  - `modes/tty/map_prop_chart.go:475-479`
  - `modes/tty/map_prop_draw_test.go:111`
- Simplify/Delete? N.
- Action: either make the move an action that `[keys]` can rebind, or have the HUM LEAD rule that modal arrows are exempt and amend FR-1.14 to match.

**RD-7 — Some documents still name API members that were renamed or never existed.**
- `architecture.md:133` lists `Inputs` "with `Refused`". It was renamed `Unread` at rc.3 (`go doc Inputs` shows `Unread`).
- `architecture.md:43` draws a `Library` method called `Fields`. No such method exists.
- go-ionomaps `api-review.md:27` gives `Inputs.Refused` as an "After" name, which contradicts its own rows `:20` and `:30`.
- Severity: Minor.
- Evidence: watchpost `03-architecture-design/architecture.md:43,133`; go-ionomaps `03-architecture-design/api-review.md:27`.
- Simplify/Delete? N.
- Action: correct them to the current names. The standing rule is that docs reflect current reality.

**RD-8 — The plan documents still carry superseded typical-error figures.**
- go-ionomaps `implementation-plan.md:134` gives 500-1000 km 0.91|3.24, 1000-2000 km 1.20|4.49, over 2000 km 1.36|4.58.
- The code holds 0.87|3.25, 1.22|4.33, 1.37|4.73 (`ahead.go:60-63`, D-181's re-measure, build log `:37`).
- watchpost `red-team-plan.md:146` repeats "to 1.36 | 4.58".
- Severity: Minor.
- Evidence: as above.
- Simplify/Delete? Y: cite `ahead.go` instead of copying the numbers.
- Action: point both documents at the code, or update the figures.

**RD-9 — The problem statement contradicts itself and the M1/M4 record.**
- The G1 definition (`:64`) reads "in use against not" and "50 MB RSS (target set from the dry run)".
- The ruled target (`:105`) is 25 MB, judged against the map open in Forecast mode (D-236).
- `:98` still says "M1 and M4's baseline is not yet measured (W10.3)". The graded run is done, and both missed: M1 53.7% against 57.4%, M4 68.4% against 69.4% (`m1-m4-instrument.md:12`).
- Severity: Minor.
- Evidence: watchpost `01-objectives/problem-statement.md:64,98,105`.
- Simplify/Delete? N.
- Action: update the G1 row and the "Measured on" paragraph.

**RD-10 — The README cites the wrong go-tuiMaps ruling for the Go floor.**
- `README.md:516` says the floor of 1.26.10 is required by go-tuiMaps' "D-28".
- D-28 set 1.26.9. The 1.26.10 floor is its D-38, restating watchpost D-244.
- Severity: Minor.
- Evidence: watchpost `README.md:516`; go-tuimaps `propagation-fields/02-analysis/rulings.md:49,59`.
- Simplify/Delete? N.
- Action: cite D-38 / D-244.

**RD-5 — The go-ionomaps README says it is at rc.3; rc.5 is tagged and pinned.**
- `README.md:8` says "Status: v0.1.0, at release candidate 3".
- Tags run up to `v0.1.0-rc.5`, and watchpost pins rc.5 (`go.mod:9`).
- The same README describes `ionomapstest`, which arrived in rc.4.
- Nothing tests this README. go-tuimaps has `readme_test.go`; go-ionomaps has no equivalent.
- Severity: Minor.
- Evidence: go-ionomaps `README.md:8`; watchpost `go.mod:9`.
- Simplify/Delete? Y: drop the rc number from prose.
- Action: correct it, and consider a README test like go-tuimaps'.

##### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**RD-2 — The CHANGELOG states the memory fix as done before the measurement exists.**
- `CHANGELOG.md:32-34` says "**Memory no longer grows day by day.**"
- The record says the proof is still running: "the fixed build now running beside 0.18.2 to show it" (`00-REQUIRED-READING.md:22`). The `heap-step/` run, from 2026-10-23T23:23Z, has no summary yet.
- D-242 itself names other growth that the fix does not touch: "the HTTP client's bounded caches filling and HMS's daily file".
- Severity: Important. This is user-facing.
- Evidence:
  - watchpost `CHANGELOG.md:32-34`
  - `00-REQUIRED-READING.md:22`
  - `02-analysis/rulings.md:263`
- Simplify/Delete? N.
- Action: until the comparison is summarised, word the entry as what changed ("the history store no longer holds each hour's document; its replay cache is bounded at 8 MB"). Assert "no longer grows" only from the committed summary (E-19).

**RD-3 — A normative go-ionomaps requirement has no instrument, and no release gate will catch it.**
- R-9.8 requires:
  - a constants table, each constant with its basis and date, carried in the snapshot;
  - `TestEveryConstantCarriesItsBasis`;
  - a release step that fails when any constant is more than 12 months old.
- The test does not exist. `TestTheTypicalErrorCarriesItsBasis` covers one constant only, not the kernel parameters, the −0.5 MHz seed, the 0.45 band or the 2000 km distance.
- Plan task G10.5 was never logged. The build log shows G10.1-G10.4 only.
- The release checklist's row 0.2 checks only "G0 to G9", and it has no expiry row.
- Severity: Important. The requirement reads as held, and nothing will catch the gap.
- Evidence: go-ionomaps:
  - `01-objectives/requirements.md:114`
  - `04-development/implementation-plan.md:134`
  - `04-development/build-log.md:130`
  - `07-readiness/release-checklist.md:17`
- Simplify/Delete? N.
- Action: build G10.5, or bring R-9.8 to the HUM LEAD to amend or defer. Either way, add a row to the release checklist.

**RD-11 — The go-tuiMaps release notes publish a solar-position accuracy no test holds.**
- `release-notes.md:27` says "the sun within 0.011° of NOAA's solar position over 2020 to 2035 (126 moments)". That comes from a one-off measurement (A-7).
- The test holds 0.5°, over 78 moments, all in 2026 (`internal/sun/sun_test.go:40-48,62`).
- The doc comment honestly says 0.5°.
- Severity: Minor.
- Evidence: go-tuimaps `propagation-fields/07-readiness/release-notes.md:27`; `internal/sun/sun_test.go:40-62`.
- Simplify/Delete? Y.
- Action: publish what the test holds (0.5°, 2026), or add the 2020-2035 check.

**RD-12 — The allowed docs check dirties the tree and logs a run that never ran.**
- On an unchanged tree, `make verify-docs` refuses. `tools/runlog` still appends `FAILED (exit 2) | 0` to the tracked `06_docs/verify-runs.md`.
- That leaves the tree dirty and puts a run that never happened into the gate record.
- go-ionomaps' `scripts/gate --docs` logs its "nothing staged" refusal the same way. I read that code; I did not run it.
- Severity: Minor.
- Evidence: watchpost `Makefile:302-306`; go-ionomaps `scripts/gate:182-190,197-199`.
- Simplify/Delete? N.
- Action: log only runs that ran a leg, or have the refusal skip the log.

##### 4. Is the audience order right: designers, then PMs, then engineers?

- RD-13 above covers the entry document.

**RD-14 — The libraries' public API docs are written for their own development record, not for a library user.**
- go-ionomaps `go doc -all` carries about 115 ruling IDs in 503 lines, for example "watchpost D-105 c", "R-2.6" and "watchpost D-239".
- They point into another repository's records, which a user of an MIT library cannot resolve.
- Severity: Minor.
- Evidence: go-ionomaps `options.go`/`snapshot.go` doc comments, as surfaced by `go doc UpdateOptions` and `go doc Scales`.
- Simplify/Delete? Y.
- Action: a recommendation to the HUM LEAD. Keep IDs in tests and the build log, and give library users sentences in the public doc comments.

**RD-6 — `go doc` cannot show the fields of types the READMEs and release notes tell readers to use.**
- go-ionomaps `README.md:61` says "The full API is in the package documentation (`go doc -all …`)".
- But `Source = terms.Source` shows no fields there (Name, Hosts, MaxBytes), and `ScaleUnset = scales.None` hides its value.
- go-tuiMaps has the same pattern. `go doc . Type.Edge`, `Type.Colours`, `Grid.OverWater`, `Image.MaskedByWater` and `Answer.ValueUnit` all fail, yet `release-notes.md:12-32` tells hosts to use them.
- Severity: Minor.
- Evidence:
  - go-ionomaps `README.md:61`
  - go-tuimaps `propagation-fields/07-readiness/release-notes.md:12,13,24,32`
  - `go doc . Type` output
- Simplify/Delete? N.
- Action: either document the fields where users will look (the README or the alias's own comment), or stop claiming `go doc -all` is complete.

##### 5. Is a number published without its blind spots beside it (INST-5)?

**RD-1 — "G1 met" is published without the comparison D-236 says must be reported, and that comparison misses both targets.**
- D-236 judges G1 against the map open in Forecast mode, and rules that "the comparison with the map never opened is reported."
- The record publishes the judged numbers:
  - Required reading: "G1 met over its 48 hours".
  - D-241: CPU −0.17%, RSS −10.4 MB.
  - Build log, after batch 58: −0.10%, −0.8 MB.
- The reported comparison appears nowhere in the tree. The 48-hour summary kept outside the tree has it: **+3.42% CPU against a 1% target, +40.0 MB RSS against 25 MB**, both missed. The hourly run is the same: +3.56%, +41.8 MB.
- That summary also ends "The run is shorter than G1's 48 hours: a dry run, not the measure". D-241 overrode that by ruling, but the summary's own verdict was never reconciled with the record.
- Severity: Important.
- Evidence: watchpost:
  - `00-REQUIRED-READING.md:22`
  - `02-analysis/rulings.md:257,262`
  - `04-development/build-log.md:745-748`
  - `01-objectives/problem-statement.md:105`
- Simplify/Delete? N.
- Action: put the reported comparison beside every "G1 met", and add a note that the summary's "dry run" verdict was superseded by D-241. The figures should come from the committed perfsum (E-19).

There is a related gap for the HUM LEAD to weigh. The CHANGELOG and README publish the band answers ("the bands usable near the place") with no word on how well they agreed with observation. The graded M1 run measured 53.7% against a climatology baseline of about 52%. D-118 sends that miss to VALIDATE. Whether users should be told is a UX fork, and I am not ruling on it.

---

**Verdict: do not exit REVIEW.** No finding is Critical. But RD-1, RD-2 and RD-3 are claims the record presents as held when they are not, and two of them are what VALIDATE will rely on. Each is cheap to fix, either a wording change or a ruling.

**Fix first: RD-1.** D-236 promised that the other G1 comparison would be reported, and it never was. The HUM LEAD would otherwise grade G1 at VALIDATE without seeing that against an idle watchpost, the mode in use costs 3.4 times the CPU target and 1.6 times the RSS target.

## Business Quality (RB)

### Business Quality review: 0.19.0, round 1 (RB-)

**Tree state.** `git status --short` was empty in all three repositories at the start and at the end of this pass: watchpost `6ad1c098`, go-ionomaps `d80d8db`, go-tuiMaps `544b889`. Every finding below was reproduced against those clean tips. All scratch work is in `<scratch>/rt-review-r1/business/probe`, which holds two programs, `main.go` and `stale/main.go`.

**What I ran.**
- watchpost:
  - trace tests: `cmd/watchpost` `TestEveryRequirementsTest*`, `TestEveryCodeCitationResolves` and `TestEveryRulingNamingCodeCitesItsLine` all pass.
  - packages: `platform/propanswers`, `domains/propagation`, `tools/reachscore` and `tools/perfsum` pass; `modes/tty` filtered to the propagation tests passes.
- go-tuiMaps `BenchmarkGlobeFieldPan` (go-tuiMaps M5): 10.4–10.8 ms at 400×110 and 2.5–2.7 ms at 200×56, so M5 is met.
- go-ionomaps through its public API with `ionomapstest` synthetic days, five cases: the shipped build, `NoStations`, no GloTEC, GloTEC 3 h stale, and 6 h / 2 days past the last grid.
- Evidence read: `~/watchpost-0.19.0-plan-evidence/` (G1's 48-hour and hourly summaries, the graded M1/M4 table, G-M3 on the fresh days, the heap run's `samples.csv`).

---

#### 1. Does the work meet the requirement that was written down, or the one that was remembered?

**RB-1. The Pacific's "or worse" (D-129) never reached the code, and the tests lock in the shorter wording.**
- D-129 corrected D-116 because PLAN's week showed the hybrid worse than the climatology in the Pacific. It ruled that beyond 2000 km the words say "about as good as typical here, **or worse**". FR-5.5 carries that wording.
- The code says only "about as good as typical", and the legend says "~typical".
- The fresh days make it worse: the Pacific hybrid is foF2 1.47 against the climatology's 1.22 MHz, and MUF 5.43 against 4.92 (`gm3/fresh-shipped.md`).
- D-198 shortened the **legend** to "~typical". It did not rule on the **words**.
- Severity: Important.
- Evidence:
  - watchpost `06_docs/02_features/propagation-overlays/01-objectives/requirements.md:144`
  - `02-analysis/rulings.md:150` (D-129) and `:219` (D-198)
  - `modes/tty/map_prop_words.go:261,263`
  - `modes/tty/map_prop_chart.go:362`
  - `modes/tty/map_prop_words_test.go:450-452`
  - `modes/tty/map_prop_honesty_test.go:82`
- Simplify/Delete? N.
- Action: make the words say "or worse" as D-129 ruled, and change the tests to match. Whether the legend needs it too is a fork for the HUM LEAD (layout).

**RB-2. go-ionomaps calls a snapshot `Fresh` when GloTEC's newest listed grid is stale. R-5.4 and D-239 say "never Fresh".**
- How I reproduced it: I served D-RAP and the scales, and let the GloTEC index list only a grid 3 h old.
- Result: `Reason=Fresh`, `Background.FoF2=climatology`, `Unread=[]`.
- With GloTEC withheld altogether, the same update is correctly `Partial` with `Unread=[GloTEC]`.
- Cause: in `refreshGrid`, when the index lists no fresh grid, the function returns without naming GloTEC as unread.
- The G-M4 fault table accepts this case on background alone, so the gate cannot catch it.
- A stalled index is the likeliest way a GloTEC outage actually happens.
- watchpost's own words still say "foF2 from the climatology: no GloTEC grid", so watchpost's listener is not misled. Any other host that reads `Reason` is.
- Severity: Important. It is the exact class the HUM LEAD ruled "we should never have data mislabelled".
- Evidence:
  - go-ionomaps `update.go:233-235`
  - `snapshot.go:146`
  - `06_docs/02_features/go-ionomaps/01-objectives/requirements.md:76`
  - `faults_test.go:38`
  - watchpost `modes/tty/map_prop_update.go:322`
- Simplify/Delete? N.
- Action: name GloTEC in `Unread` when the index lists no usable grid, and assert `Reason != Fresh` in the stale-GloTEC fault case.

**Checked, no finding:**
- hours ahead carry the typical error and its basis (`map_prop_ahead.go:175-178`);
- `NoStations` asks GIRO nothing (my probe: NOAA's host only);
- G-M3 and D-75's floor are met;
- go-tuiMaps M5 is met.

G-M3 pairs each held-out station with readings at their freshest (gm3 calls this "its best case"). It discloses that, and the margin (-35% against a 15% target) absorbs it.

#### 2. What does the user lose if this ships as it stands?

**RB-3. PS-1's answer agrees with what was observed little better than a coin toss, and the product does not say so.**
- Graded M1 on the fresh days: 53.7% for the shipped build against 52.4% for the climatology. D-99's target is 57.4%, so it is missed.
- When WSPR shows a served-area path open, the chart says it reaches only **32.9%** of the time. The Broadcaster operator is mostly told "closed" where voice would carry. That is PS-2's "miss ones that are open", recreated for PS-1.
- In six of M1's ten bands (20 m to 10 m) every column scores exactly 50%. There the answer is constant.
- M4 is closer: 68.4% against a 69.4% target, with a real gain from the stations on 17–12 m.
- The chart states open and closed categorically. Apart from distance to the nearest station, it says nothing about how often the current hour's answer is right.
- D-118 correctly routes the miss to VALIDATE.
- Severity: Important.
- Evidence:
  - watchpost `07-readiness/m1-m4-instrument.md:238-259`
  - `01-objectives/problem-statement.md:112` (M1's target, D-99)
  - `01-objectives/requirements.md:83` (FR-1.15)
  - `~/watchpost-0.19.0-plan-evidence/m1m4-fresh/graded-table.md`, M1's "by kind" row
- Simplify/Delete? N.
- Action: the HUM LEAD decides at VALIDATE. The VALIDATE report should say PS-1 is not met on accuracy, not just "a target missed". Whether the chart should state its measured agreement is the HUM LEAD's fork (UX).

**RB-4. On the comparison G1 was first designed around, the mode misses its CPU and memory targets. The Broadcaster station that issue #25 is about was never measured.**
- What changed:
  - A-79 designed G1's "not in use" arm as the map never opened, "the reading that can only overstate G1".
  - The dry run showed that arm failing (+3.65%, +50.9 MB).
  - D-236 then judged G1 against the map open in Forecast mode instead.
- The 48-hour run against the map never opened: **+3.42% CPU and +40.0 MB RSS, both missed**.
- The judged figures are negative (-0.17%, -10.4 MB). That shows the baseline cannot measure the mode's own cost: the Propagation mode switches off the Forecast layers it is compared against.
- All three arms were Observer runs (`g` from the dashboard).
- The Broadcaster's map is **new in 0.19.0**. For PS-1's operator, the honest baseline is the console with no map, and it was never run.
- A count narrowed after seeing it is a defect until proven otherwise. D-236 is the HUM LEAD's ruling, so this is a recommendation.
- Severity: Important.
- Evidence:
  - watchpost `02-analysis/rulings.md:257` (D-236) and `:353` (A-79)
  - `01-objectives/problem-statement.md:105`
  - `04-development/build-log.md:671`
  - `scripts/quality/workload.expect:46,74`
  - `CHANGELOG.md:15`
  - `~/watchpost-0.19.0-plan-evidence/g1-run-2026-10-19/g1-summary.md`
- Simplify/Delete? N.
- Action:
  - put both comparisons before the HUM LEAD at VALIDATE;
  - run a Broadcaster arm (console idle against console plus `g`/`P`);
  - re-rule G1, or record the miss.

**Also recorded, not new:** the screen-reader focus limit is moved to 0.19.1 (FR-3.7, RK-6, D-122).

#### 3. What was cut, and is the cut recorded where the next reader will find it?

The ruled cuts are recorded and findable in `06_docs/follow-ups.md`:
- band watch (F-209);
- observed spots (F-207);
- propagation history (F-210);
- eSSN and the planner (F-189);
- FR-7 moved to 0.19.1 with its SHIP gate.

**RB-5. The 48-hour G1 figures against the map never opened exist only outside the tree.**
- D-236 promised that comparison would be "reported".
- No file in the repository holds the 48-hour figures (+3.42%, +40.0 MB). The tree cites only the first minutes of the dry run.
- A reader of the REQUIRED-READING sees "G1 met".
- Severity: Important.
- Evidence: watchpost `00-REQUIRED-READING.md:17`, `02-analysis/rulings.md:262` (D-241), `04-development/build-log.md:747`.
- Simplify/Delete? N.
- Action: record both comparisons from the committed `perfsum` (E-19) in the problem statement's G1 row and in the build log.

**RB-6. The changelog claims a memory fix before the run meant to prove it has finished.**
- `CHANGELOG.md` says "Memory no longer grows day by day."
- The proof run (D-242: the fixed build beside 0.18.2) is 14.7 h into a planned 72 h. Both instances read about 262.5 MB RSS so far.
- D-242's own profile says part of the daily rise is bounded caches filling, which the fix does not touch.
- Severity: Minor.
- Evidence: watchpost `CHANGELOG.md:32-34`, `00-REQUIRED-READING.md:17`, `~/watchpost-0.19.0-plan-evidence/heap-step/run2.sh:6`.
- Simplify/Delete? N.
- Action: hold or soften the line until the run's figures are committed.

**RB-7. The listener-facing record carries none of the release's known limits.**
- 0.19.0's changelog does not mention:
  - the focus limit;
  - accuracy near the climatology, and worse than it in the Pacific;
  - the CPU and RSS cost of opening the mode.
- The items going to VALIDATE are scattered: M1 and M4 on the instrument page, the Pacific in batch 53's log, G1's never-opened comparison nowhere. No single agenda lists them.
- Severity: Minor.
- Evidence: watchpost `CHANGELOG.md:5-34`, `07-readiness/m1-m4-instrument.md:259`, `04-development/build-log.md:684`.
- Simplify/Delete? N.
- Action: keep one VALIDATE agenda. Whether the changelog should carry a "known limits" section is the HUM LEAD's call.

**RB-8. The instruments do not state their own blind spots in their output, which INST-5 requires.**
- `reachscore`'s verdict line says nothing about:
  - WSPR being the only reference (D-226 promised RBN "listed as a blind spot", and the record nowhere uses those words);
  - the balanced draw not being the real base rate;
  - six of M1's bands being structurally at 50%.
- Severity: Minor.
- Evidence: watchpost `tools/reachscore/report.go:105`, `06_docs/quality-plan.md:140`, `02-analysis/rulings.md:247` (D-226).
- Simplify/Delete? N.
- Action: add the blind spots to the report's output.

**RB-9. go-ionomaps' README says it is at rc.3. It is tagged rc.5, and watchpost pins rc.5.**
- Severity: Minor.
- Evidence: go-ionomaps `README.md:8`; watchpost `go.mod:9`.
- Simplify/Delete? N.
- Action: fix the line.

#### 4. Is there a cheaper way to get the same outcome for the user?

**RB-10. The default refresh, every 10 minutes, costs about 5× the hourly setting on the wire, with no measured accuracy gain.**
- Measured: 13.94 MB an hour (334 MB a day open) against 2.66 MB an hour.
- GIRO is asked hourly in either case.
- The graded accuracy evidence uses one grid per scored hour, so nothing measures what the extra GloTEC grids buy.
- For M1 specifically, the climatology alone (no network, no GIRO, no address disclosed) is within 1.3 points of the shipped build.
- Severity: Minor.
- Evidence: watchpost `modes/tty/map_prop_settings.go:12,63-71`, `04-development/build-log.md:747`, `m1m4-fresh/graded-table.md`.
- Simplify/Delete? Y, the default.
- Action: recommendation to the HUM LEAD (D-94 set the default): measure 10-minute against hourly agreement, or make hourly the default. The setting's comment also says "measured cost", but the strings it shows are the targets (19 and 3.3 MB), not the measured figures.

---

#### Opinion

**Do not exit REVIEW** as the record stands. The code is sound and the libraries do what they say, apart from RB-2. The problem is in the record.
- The guardrail's adverse 48-hour figures are not in the tree (RB-4, RB-5).
- An honesty wording the HUM LEAD ruled was dropped and then locked in by the tests (RB-1).
- The library breaks a "never" the HUM LEAD ruled five days ago (RB-2).

Each is small to fix, and RB-3 and RB-4 then go to VALIDATE as decisions made in the open.

**Fix first: RB-5 together with RB-4.** Put G1's 48-hour comparison against the map never opened (+3.42% CPU, +40 MB, both missed) into the record, and measure the Broadcaster. A guardrail met only after its baseline changed, with the Broadcaster station never measured, is the one claim here I could not defend to someone who did not build it.

## The REVIEW lens (RL)

### REVIEW lens: Principal QA Engineer, round 1 (0.19.0)

**Tree state.** At the start `git status --short` was empty in all three repos: watchpost `6ad1c098`, go-ionomaps `d80d8db`, go-tuiMaps `544b889`. Everything below was reproduced against those SHAs, in my own scratch clones under `<scratch>/rt-review-r1/review/` (`wp`, `iono`, `tui`). The probe tests are in `iono/rlprobe/probe_test.go`. At the end all three shared trees were still clean at the same heads.

**What I ran.** All of it passed:
- go-ionomaps `scripts/gate --quick`: green, with P10 "not run" because the clone has no `.a2dh.yml`; see RL-8.
- go-tuiMaps `scripts/gate --quick`: green, P10 included.
- watchpost `go vet ./...`.
- `go test -race` on `domains/propagation`, `platform/{httpx,propanswers,history,agememo,geo}`, `domains/locations/...` and `tools/{reachscore,perfsum}`.
- `go test` on `modes/tty`, `app` and `cmd/watchpost`.
- `make lint`: no new findings, 6 baselined.
- `make mutant-anchors`: 780 anchors match.
- `make p10`: 0 live, but only once the git-ignored exemptions file was copied in (RL-8).
- `a2dh p10 check` on both libraries: 0 live with the exemptions in place.

---

#### 1. Happy Path Bias

**RL-1. GIRO's replies are sorted into "refused" and "answered" so that one bad station blinds the whole field, and a ban is never seen.** Severity: **Important**.
- **(a) One station's 5xx is treated as GIRO refusing everything.** A 5xx or a transport error on any single station's ask stops the whole burst and pauses GIRO (`go-ionomaps internal/giro/asker.go:152`).
- When every live station fits in the burst, the rotation's start never moves. `pollAt += len(poll)` adds n each time, so `pollAt % n` stays the same (`internal/stations/stations.go:167,188`). The failing station is asked first every hour.
- Probe `TestOnePoisonStation` (AL945 answers 500): 1 GIRO ask an hour and 0 stations assimilated, for 5 hours running. This lasts until the 72-hour dark rule drops everyone, after which stations come back one probe an hour.
- `TestPoisonBetweenBursts`: the snapshot says `GIROPaused` only on the burst's own update. For the other ~50 minutes of each hour it is Fresh, apart from the replay's D-RAP and scales. watchpost then says "no station readings this hour: the background alone" (`watchpost modes/tty/map_prop_update.go:316`).
- So the stations, which are the release's measured gain (G-M3 mainland foF2 0.46 against 0.71), disappear while the map reads like a quiet hour.
- **(b) A 403, 401, 410 or 404 is treated as an answer.** It resets the back-off (`asker.go:157-160`) and the burst carries on.
- Probe `TestGIRO403`: 40, then 80, then 120 GIRO asks over 3 hours, 0 stations, `Rejected` 0, and nothing in the diagnostics.
- Under RK-2 (GIRO's terms, high impact), a host that GIRO has blocked keeps asking 960 times a day and is never told.

Evidence: the lines above. Simplify/Delete: N.
Action:
- Make a per-station 5xx or 404 that station's failure: count it and continue. Stop only on a 429, on N failures in a row, or on 401/403/410, which should pause long.
- Advance the rotation by the stations actually answered.
- Carry "GIRO failed this hour" in `Inputs` so the words say it all hour. Name 403s in the diagnostics.
- Add an `ionomapstest` fault option and a test for each case.

**RL-6. A GloTEC index that comes back but names no grid is never called unread.** Severity: **Minor**.
- This covers a changed naming scheme, a page over 4 MB, or every name past the clock. `refreshGrid` returns quietly (`go-ionomaps update.go:234`; `internal/glotec/index.go:25`).
- Probe `TestIndexNoNames`: GloTEC is never named in `Unread`. At t+110 min the background silently becomes the climatology, and the status is not `Partial`, which D-239 forbids ("a snapshot short of any is Partial, never Fresh").
- The words do say "no GloTEC grid", so a listener is not misled about the source. But nothing in the diagnostics points anyone at a format change.

Simplify/Delete: N.
Action: `missed("GloTEC", false)` when the index is read but lists no grid, with one diagnostic line.

#### 2. Systemic Failure Modes

**RL-9. The fix for the heap growth leaves REVIEW unproven.** Severity: **Minor**.
- D-242 rewrote the history store's caching during REVIEW (batch 56): `platform/history/history.go:272`, plus the index path. 0.18's datasets rely on that store.
- The soak that would show the growth bounded is still running. `00-REQUIRED-READING.md:22` says "the fixed build now running beside 0.18.2 to show it", and the hourly profiles in `heap-step/` are still being written today.
- The unit and property tests pass. But the defect was state that builds up over days, and only a multi-day comparison can confirm the fix.

Simplify/Delete: N.
Action: record the side-by-side's verdict, as a D-row or A-row, before REVIEW exits. Until then, nothing should say "bounded".

Also systemic, and ruled (FR-4.12): GIRO's 40-an-hour limit is kept per process, not per IP. Two instances share the state file, the last writer wins, and both burst. RL-1(a) makes this worse: each instance stalls on the same station. No new finding.

#### 3. Gate Validity

**RL-3. G1 is judged against a comparison chosen after a count, and that comparison can hardly fail.** Severity: **Important**. This is a recommendation to the HUM LEAD.
- G1 is defined as "added … with the Propagation mode in use against not" (`problem-statement.md:64`; the guardrail is issue #25's station computer).
- D-236 (`rulings.md:257`) switched the judged comparison to "the map open in Forecast mode". It did so after the dry run measured the original comparison failing (+3.65 % CPU, +50.9 MB).
- On the 48-hour run, the original reading still misses both targets: +3.42 % CPU and +40.0 MB, against 1 % and 25 MB (evidence `g1-run-2026-10-19/g1-summary.md`, outside the tree). The judged reading comes out *negative*: -0.17 % and -10.4 MB.
- The reason is that the reference arm draws weather layers the Propagation mode switches off (FR-1.4). The judged figure is propagation minus Forecast, a swap of one mode for another, not the mode's own cost. It fails only if propagation costs more than the weather map.
- The Broadcaster operator (PS-1) does not otherwise have the map open, so for them the never-opened arm is the relevant one.
- Separately, the summary accepted as G1's measure prints its own "a dry run, not the measure" (`tools/perfsum/g1.go:139`), overruled by D-241.

Simplify/Delete: N.
Action:
- Put both readings to VALIDATE, with the never-opened figure as the issue #25 number.
- Or isolate the mode's own cost with a reference arm that has the map open and no layers drawn.
- SHIP's PR must not say "G1 met" without the qualifier.

**RL-5. go-ionomaps, the release's numerical core, has no mutation tests anyone can re-run.** Severity: **Important**.
- `git ls-files | grep -i mutant` in go-ionomaps returns nothing. The evidence is 39 build-log rows of "hand mutants, all killed", in prose.
- P10 density exemptions are ratified on that evidence, for example D-74 (`go-ionomaps rulings.md:95`): the root package at 0.69 across 101 functions, "held by tests and hand mutants".
- The other two repos re-run theirs. go-tuiMaps: 74 mutants in `06_docs/mutants/mutants.json`, killed by a gate leg (`go-tuimaps scripts/gate:574-580,819`). watchpost: 780.
- So the code that turns ionosphere data into "open" or "closed" (answers, lower limit, reach, assimilation, update) has the weakest regression protection, and the exemptions it carries cannot be re-checked.

Simplify/Delete: N.
Action: port go-tuiMaps' `mutants.json` harness to go-ionomaps before v0.1.0 final. Re-encode at least the hand mutants recorded for `update.go`, `answers.go`, `internal/giro` and `internal/glotec`.

**RL-8. P10 can only be reproduced on one disk.** Severity: **Minor**.
- The exemption ledger, `.a2dh-p10-exemptions.yml`, is git-ignored in all three repos (go-ionomaps `.gitignore:22`).
- A fresh clone at the reviewed tips gives `make p10` with 23 live findings in watchpost and 16 in go-ionomaps. With the file copied in, it gives 0.
- The library gate exits 0 "green without P10" (`go-ionomaps scripts/gate:256-274`), and no CI workflow runs P10. The repo admits this: "every HUM LEAD ratification lives on one disk … CI cannot run the gate meaningfully" (`watchpost scripts/quality/p10-ledger-mirror.py:7-9`).
- NFR-4 ("P10 clean at every phase exit") can therefore be checked only on the HUM LEAD's machine.

Simplify/Delete: N.
Action: a HUM LEAD decision. At least have CI check that the committed mirror covers the findings, or back the ledger up somewhere.

#### 4. Adversarial Input

No concern:
- The coordinate, locator and frequency parsers work locally and are bounded, and NaN and Inf are refused (`platform/geo/coords.go:17-41`, `resolver.go:typedPlace`).
- A position typed in another form is refused offline.
- The propagation client is https-only, refuses private addresses, follows same-origin redirects only, caps each host's body size, allows only the hosts go-ionomaps exports, and returns redacted errors (`platform/httpx/plain.go`, `domains/propagation/propagation.go:Fetch`).
- Every library parser has a fuzz target. Restore is size-capped and checked against the clock.

#### 5. Graceful Degradation

**RL-2. One small NOAA text file decides whether there is any map at all.** Severity: **Important**.
- When the solar file cannot be read at a cold start (a NOAA outage, or a format change to a parser that matches by heuristics), `Update` returns a bare `Snapshot{Reason: Offline}` plus `ErrNoField` (`go-ionomaps update.go:139-140`).
- Probe `TestColdStartNoSolar`, 3 hours: 0 field cells and 0 GIRO asks. Six GloTEC asks, about 2.5 MB of grid each, were downloaded and thrown away. `Unread` was empty, and the snapshot said `Offline` though NOAA had answered, against D-239's definition.
- GloTEC's foF2 does not need F10.7 at all.
- RK-15's mitigation, "the last F10.7 mean held is used with its age" (`requirements.md:228`), only lasts the life of one process. The state file keeps GIRO's throttle alone (`propagation.go:keepState`), and F10.7 is public domain, so it could be kept too.
- The user is told the mode asks again "at the first update after HH:MM · R refreshes now". Pressing R before that time gets "too soon" (`map_prop_update.go:198`).

Simplify/Delete: N.
Action:
- Keep the last F10.7 mean and its date in the state file.
- Draw foF2 from GloTEC with M(3000)F2 named as unavailable, or use a named fallback mean.
- Carry `Unread` on the error snapshot, and ask for no grid while no field can be built.

**RL-7. While NOAA refuses, it is asked more often than when it is healthy.** Severity: **Minor**. The back-off is per ruling (D-39), so this is a recommendation.
- The back-off starts at 60 s, below the 10-minute cadence (`update.go:200-206`).
- Probe `TestNOAA429Cadence`, with the host asking every 15 s: NOAA was asked at +1, +2, +4, +8 and +15 minutes, 8 times in an hour of refusals against 6 when healthy.
- watchpost's 10-minute tick hides this, but every failure line offers "R refreshes now".
- A NOAA 5xx is not counted as a refusal at all, while GIRO's is (RL-1). The two are inconsistent.

Simplify/Delete: N.
Action: never let a back-off be shorter than `noaaEvery`, and treat 5xx the same way on both services.

**RL-11. A state saved a moment "in the future" is thrown away, and the library then asks GIRO at once.** Severity: **Minor**.
- `Resume` refuses a state whose last burst is even one second after now (`internal/throttle/throttle.go:124`).
- After the clock steps backwards, the state is dropped and the library starts as new, bursting at once: the direction that asks GIRO more.

Simplify/Delete: N.
Action: clamp a small skew to now rather than refusing the state.

#### 6. Test Confidence vs. Test Results

**RL-12. watchpost never sees a real go-ionomaps snapshot, and the replay cannot produce the failures above.** Severity: **Minor**.
- The TTY tests drive hand-built snapshots through a seam (`modes/tty/map_prop_draw_test.go:20-30`, `map_prop_stations_test.go:87`).
- `ionomapstest`, the one replay of the real library, answers D-RAP and the scales 404 every time (`ionomapstest/replay.go:106`). Every replayed snapshot is therefore `Partial`, and a replay can never be `Fresh`.
- It has no way to inject a per-address status or body. So the states in RL-1, RL-2 and RL-6 have no seam in any host test, and the words for them were never tested against real library output.

Simplify/Delete: N.
Action: give `ionomapstest.Options` per-address faults (a status, a body, an error). Add one watchpost test that walks the real library's states through `snapshotWords`.

The tests I sampled assert what they claim (fetch counts, the acknowledgement order, off-goroutine asks). I found no retried flaky tests. The `make race` timeout raise in batch 54 was measured, not a blind retry.

#### 7. Dependency Fragility

**RL-10. watchpost builds the libraries' release candidates, not the reviewed tips.** Severity: **Minor**.
- `go.mod:8-9` pins go-ionomaps `v0.1.0-rc.5` and go-tuiMaps `v0.3.0-rc.3`.
- Those tags are not the reviewed tips, and they say `go 1.26.9`, while D-244 says every built module is 1.26.10. Between tag and tip only the go directives and workflows differ.

Simplify/Delete: N.
Action: tag v0.1.0 and v0.3.0 from the reviewed SHAs and re-pin before VALIDATE, so UAT runs exactly what ships.

RL-1 and RL-2 are the real dependency fragility. Timeouts exist: 30 s a request and 60 s an update.

#### 8. Problem statement present, real, honored

The locked statements are present and genuine: `01-objectives/project-brief.md:67-79` and `problem-statement.md`, PS-1 and PS-2 (D-7), with a five-criteria scorecard. The brief still lists M1 to M3; M3 was retired (D-76). The brief's own status line says it is not rewritten.

**RL-4. PS-1 has no accuracy metric that can tell a right answer from a wrong one, and no floor on accuracy at all.** Severity: **Important**.
- M1 on the fresh days: 53.7 % [51.3, 56.1], where a constant answer scores 50 % by design. The lead over the climatology is +1.3 against a target of +5 (`07-readiness/m1-m4-instrument.md:247`).
- Six of the ten bands are forced to 50 % (`:256`). The instrument page itself says no model that depends on band and hour can reach +5 under the original draw (`:183-193`).
- No floor rides on M1 or M4 (D-75), and misses go to VALIDATE (`problem-statement.md:96`). So no accuracy gate in this release can fail it.
- PS-1 is a hypothesis (D-217), M3 is retired, and M2 by definition excludes accuracy (D-92).
- What SHIP can honestly copy into the PR is: "an operator can ask from the station." It cannot claim the answer is right for the served area.
- M4 (68.4 % against 64.4 %) does show a real gain for PS-2.

Simplify/Delete: N.
Action: before VALIDATE, the HUM LEAD rules three things:
- the PR's wording for PS-1, limited to answerability;
- whether M1 is re-scoped (E3, said plainly to be post hoc) or retired for near-vertical served areas;
- a recorded line that the release ships without an accuracy floor.

---

#### Opinion

**Do not exit REVIEW.** Nothing here is Critical. But RL-1 and RL-2 are library defects that silently throw away the release's measured value: the stations, and in RL-2's case the whole field. D-239 already set the bar ("we should never have data mislabelled"), and RL-1 misses it for 50 minutes of every hour. Both are small fixes for an rc.6, each with a probe that already reproduces it. RL-3 and RL-4 are forks for the HUM LEAD that SHIP's PR depends on.

**Fix first: RL-1.** A single station's 500 stalls every GIRO burst for days while the map says "no station readings this hour".

## InfoSec (RI)

#### InfoSec review, 0.19.0 (round 1)

**Tree state.** I recorded `git status --short` for each repository at the start and again at the end of my pass. All three were empty both times:
- watchpost `feature/propagation-overlays` at `6ad1c098`
- go-ionomaps `feature/discover` at `d80d8db7`
- go-tuiMaps `feature/propagation-fields` at `544b889d`

Every finding below was reproduced against those tips:
- **go-ionomaps and go-tuiMaps:** I tested `git archive` copies of the tip commits in my scratch directory.
- **watchpost:** I used a scratch module with `replace` pointed at the tree.

Nothing was written into any tree. Scratch: `<scratch>/rt-review-r1/infosec` (`iono/zz_infosec_*_test.go`, `tui/zz_infosec_test.go`, `wp/*_test.go`).

**What I ran:**
- **`go vet`:** clean on all of go-ionomaps and go-tuiMaps, and on the watchpost packages that touch input (`app`, `domains/propagation`, `platform/httpx`, `domains/locations/...`, `domains/radio/synth`, `platform/history`, `platform/propanswers`, `platform/geo`).
- **`go test -race`:** passed on those watchpost packages, on every go-ionomaps parser package (`giro`, `glotec`, `drap`, `solar`, `scales`, `stations`, `throttle`), and on go-tuiMaps `fetch`, `mvt` and `overlay`.
- **Hostile-input tests** (my own, no fuzzing):
  - GloTEC: duplicate keys, nesting 9k and 20k deep, a 7 MB padded body, a 1970 time.
  - The solar file: duplicate days, a 70 KB line, 50 MB of junk.
  - GIRO replies: forged location lines, top-of-range values with score 999, a huge `Retry-After`.
  - A hostile saved throttle state.
  - Edge places through every answer call: poles, ±540° longitude, antipodes, `Ahead(-1..30)`. No panics.
  - The geocoder with escape sequences and 1e300 coordinates.
  - The Plain client against 9 MB of headers and a gzip bomb.
  - go-tuiMaps with escape sequences in a host overlay's unit.
- **`govulncheck`:** see Q7.

---

##### Findings

**RI-1 (InfoSec): Three GIRO replies can set foF2 for the whole globe, and the field still says "background" at the far cells.**
- **What happens:**
  - A reply's location line replaces the station's position from the shipped seed list, so a station can claim to be anywhere.
  - Score 999 ("manual") skips the confidence filter.
  - The assimilation fits station residuals around their mean, so far from every station the field falls back to that mean, everywhere. Nothing checks a reading against GloTEC for plausibility.
- **Reproduced:** three replies relocated to London, each reading foF2 20.0 at score 999, over a GloTEC grid of 7.0 MHz. Result: foF2 20.0 at London (70 km away), at Sydney (nearest station 16,928 km) and at 60°S 120°W (16,218 km). An honest 7.0 left 7.0. With 40 such replies, MUF(3000) came out at 100 MHz.
- **Why it matters:** R-3.2 says a cell beyond 500 km of any station is background, yet its value moved by 13 MHz. A malfunctioning ionosonde does this as easily as a hostile one.
- **Severity:** Important.
- **Evidence (go-ionomaps):**
  - `internal/giro/giro.go:140-149` (location taken from the reply) and `:200` (999 accepted)
  - `internal/stations/stations.go:233-237` (the reply's position overwrites the seed's)
  - `internal/assimilate/assimilate.go:75-81` (the fit is about the mean residual)
  - `update.go:385-388` (MUF is foF2 × M(3000), with no upper bound)
- **Simplify/Delete?** N.
- **Action (a recommendation to the HUM LEAD, because D-101 and D-61 rule the model):**
  - Keep placed stations at their seed positions; accept a reply's position only for an unplaced station, or within about 50 km.
  - Refuse a reading whose residual against GloTEC is beyond k·σ or a fixed number of MHz.
  - Either decay the mean offset with distance, or have the snapshot say the shift was applied everywhere.

**RI-2 (InfoSec): The geocoder cleans name, admin1 and country, but not postcodes, timezone or coordinates. The postcode reaches the terminal raw.**
- **Reproduced:** with escape sequences in the reply, `Geocode` and `ResolveWorld` returned `Zip` with an OSC 52 sequence plus SGR intact, `TZ` with OSC 52, and Lat/Lon of ±1e300.
- **Where the zip is drawn:** the setup form concatenates it unescaped beside `render.Plain(Label)`.
- **What survives the renderer:** bubbletea v2's renderer drops most sequences but keeps SGR and OSC 8 (`ultraviolet styled.go:195-199`). So a compromised or impersonated Open-Meteo response can bleed styling and plant a clickable hyperlink whose visible text it chooses. The value is also saved to config and drawn on every launch.
- **The 1e300 coordinates:** the Propagation finder announces "Found …: marked" while go-tuiMaps refuses the place (`go-tuimaps view.go:42`, `places.go:133`).
- **Severity:** Minor.
- **Evidence (watchpost):**
  - `domains/locations/openmeteo/geocode.go:85` (three fields cleaned) and `:97-106` (zip, TZ, Lat/Lon unchecked)
  - `modes/tty/setup_form.go:48,50,77,226,228,248` (zip drawn raw)
  - `modes/tty/map_prop_find.go:166,183-188`
- **Simplify/Delete?** N.
- **Action:**
  - Run each postcode through `cleanField` and keep only `^\d{5}$` for US.
  - Accept the timezone only if `time.LoadLocation` accepts it.
  - Refuse a result off the globe.
  - Add both to `TestTheGeocodersTextIsCleaned`.

**RI-3 (InfoSec): The solar file's mean can be inflated by duplicate days and silently cut short by one long line, and the parser has no size bound of its own.**
- **Duplicate days:** 30 honest days at 70 sfu plus 900 repeats of yesterday at 500 gave a "30-day" mean of 486.1 over 930 days. `F107Days` reports 930.
- **One long line:** a 70 KB line stops the scanner without an error. The file then yields a mean from one day (450) with `OK=true`.
- **No size bound:** GloTEC, D-RAP and the scales each refuse an oversized body themselves; the solar parser relies on the host's cap alone.
- **Severity:** Minor.
- **Evidence (go-ionomaps):** `internal/solar/solar.go:38-54` (`sc.Err()` is never read, no size bound) and `:89-103` (every line counted, duplicates included).
- **Simplify/Delete?** N.
- **Action:** keep one flux per date; treat a scanner error as an unreadable file; add a `maxBytes` like its siblings.

**RI-4 (InfoSec): The propagation client accepts about 10 MB of response headers per reply, five times GIRO's 2 MB body cap, and echoes any ETag back.**
- **Reproduced:** 9 MB of headers were accepted and kept in `Answer.Header`. A 512 KB ETag is kept as a validator and sent back as `If-None-Match`.
- **What works:** the gzip bomb (131 KB compressed, 64 MB inflated) was refused with a 413.
- **Severity:** Minor.
- **Evidence (watchpost):** `platform/httpx/httpx.go:254-264` (no `MaxResponseHeaderBytes`), `platform/httpx/plain.go:85,166`, `domains/propagation/propagation.go:248-249`.
- **Simplify/Delete?** N.
- **Action:** set `MaxResponseHeaderBytes` (for example 64 KiB) on the Plain transport, and drop validators over about 256 bytes in the fetcher.

**RI-5 (InfoSec): A saved throttle state with duplicate left-out stations makes one burst ask the same station repeatedly.**
- **Reproduced:** ten copies of `AT138` in `out` gave ten asks of AT138 in one burst. The 40-ask cap and the hourly limit still hold, and the file is local and mode 0600, so this needs write access to the state file.
- **Severity:** Minor.
- **Evidence (go-ionomaps):** `internal/stations/stations.go:296-310` (`Apply` checks the count, not duplicates) and `:179-186` (`rotation` appends each copy).
- **Simplify/Delete?** N.
- **Action:** refuse a state whose `LeftOut` repeats a code.

**RI-6 (InfoSec): An unreadable GloTEC directory index is swallowed and not reported.**
- **Reproduced:** with an HTML index, the snapshot fell back to the climatology, and `Unread` did not name GloTEC. The listener sees the background change with no reason given, which goes against R-3.1 ("a changed format is an error").
- **Severity:** Minor.
- **Evidence (go-ionomaps):** `update.go:233-236` returns false without adding to `unread`.
- **Simplify/Delete?** N.
- **Action:** add `"GloTEC"` to `l.unread` when the index lists no grid.

**RI-7 (InfoSec): Diagnostics carry the throttle state file's absolute path, which includes the user's name.**
- **How:** `os.ReadFile`, `CreateTemp` and `Rename` errors (each includes the path) go into a note, then into `mapProblems`. That log scrubs URLs only, and feeds `/debug/counters` and the dumps.
- **The project's own standard:** watchpost already treats a home path as sensitive (`app/debug.go:61-62`, `abbreviateHome`).
- **Severity:** Minor.
- **Evidence (watchpost):** `domains/propagation/propagation.go:112-125,161` and `app/mapproblems.go:27-29`.
- **Simplify/Delete?** N.
- **Action:** name the file only, or pass it through `abbreviateHome`.

**RI-8 (InfoSec): watchpost builds against library tags whose `go` floor is 1.26.9, not the 1.26.10 that D-244 rules.**
- **What I checked:** `v0.1.0-rc.5` and `v0.3.0-rc.3` differ from the reviewed tips only in `go.mod` and CI files, so the code reviewed is the code shipped. The build is not exposed today, because watchpost's own `go 1.26.10` governs the build.
- **Severity:** Minor.
- **Evidence:** watchpost `go.mod:9-10`. The pinned tags' `go.mod:3` reads `go 1.26.9`. Ruling: watchpost `rulings.md:265`.
- **Simplify/Delete?** N.
- **Action:** pin to tags cut at the reviewed tips before SHIP.

---

##### The persona's questions

- **Is every external input validated and bounded, or is the caller trusted?**
  - Mostly bounded. These held under every hostile input I sent: GloTEC (size, key count, 72×72 layout, centres, physical bounds, nesting), D-RAP, the scales, GIRO rows (code pattern, window, row cap, bounds), the throttle (`Retry-After` clamped to 15 minutes, even for a 20-digit value), `Restore` (64 KB, clock checks) and the answer calls' place checks.
  - Gaps: GIRO's own location and values are trusted (RI-1), the geocoder's output (RI-2), the solar file (RI-3) and header size (RI-4).
- **Authorization on state-changing paths; are errors on destructive operations handled?**
  - The release adds no network-facing server; the loopback debug server is unchanged and guarded.
  - The three state-changing paths are covered: Clear history needs the confirmation and returns joined errors, the say-orphan sweep kills only a process that passes every proof, and the state write notes its failures.
  - Swallowed errors: RI-6, and RI-5 on the state input.
- **Secrets out of logs, errors and source; crypto through vetted libraries?** Nothing found. No secrets are introduced; TLS uses Go's defaults with no `InsecureSkipVerify`; no crypto of their own; the geocoder's `name` parameter is redacted; GIRO header lines are never kept.
- **Injection or traversal?**
  - The terminal-escape path is RI-2.
  - Otherwise clean:
    - History paths pass `validSegment`.
    - Grid names are rebuilt from digits only.
    - GIRO asks go through `url.Values`.
    - The WSPR ClickHouse query takes typed numbers only.
    - `say` gets its text through a file.
    - Every `exec` call passes an argument vector with no shell.
  - go-tuiMaps cleans a host's unit before drawing it (`internal/render/field.go:505`).
- **Do errors leak structure, paths or stack traces?** RI-7. Library errors never quote input. Every committed GIRO fixture uses RFC 5737 documentation addresses only.
- **Blast radius and least privilege?** Good containment:
  - The library opens no connections of its own.
  - watchpost's propagation client allows https only, on the library's exported hosts, port 443, no credentials in the address, public IPs only after DNS, same-origin redirects, per-host body caps.
  - State files are 0600 in 0700 directories.
  - The remaining reach is data integrity through GIRO (RI-1).
- **Dependencies scanned before shipping?**
  - `govulncheck` v1.8.0 on go1.27.3 found no vulnerabilities in watchpost (`make vuln`), go-ionomaps, go-tuiMaps, or the five separate tool modules (go-ionomaps `tools/tables`; go-tuiMaps `tools/answer-key`, `atlas`, `gen-assets`, `oracle`).
  - Pinning gap: RI-8.

---

##### Opinion

**Exit REVIEW.** No finding is exploitable without compromising or impersonating a TLS-served upstream, and every Minor is a small change.

**Fix first: RI-1.** Put it to the HUM LEAD as a ruling before VALIDATE: pin station positions to the seed, gate residuals for plausibility, and either decay or disclose the global offset. Three bad readings currently rewrite the whole world's field, and the snapshot calls it background.

## Performance (RP)

#### Performance review of 0.19.0 (RP-), red team round 1

**Tree state.** At the start of my pass, `git status --short` printed nothing in any of the three repositories: watchpost `feature/propagation-overlays` @ `6ad1c098`, go-ionomaps `feature/discover` @ `d80d8db7`, go-tuiMaps `feature/propagation-fields` @ `544b889d`. I reproduced every finding below against those clean trees. Nothing I ran was written inside them. All my programs, profiles and outputs are in `<scratch>/rt-review-r1/perf` (subfolders `drive/`, `race/`, `frame/` and `geo/`; text files `ionomaps-bench.txt`, `tuimaps-bench.txt`, `reach.txt`, `alloc-budget.txt`). Other reviewers were using the machine at the same time, so every wall-clock figure is noisy. I give allocation counts beside each timing and rely on them more.

**What I ran:**
- Every benchmark in go-ionomaps; go-tuiMaps' `GlobeFieldPan`, `IdleFrame`, `FrameAdvance` and `Report` benchmarks; watchpost's propagation benchmarks (`modes/tty`, `platform/propanswers`).
- `make alloc-budget`: MAKE_EXIT=0, all packages ok.
- CPU and allocation profiles of the reach and of a frame.
- My own programs against the public APIs, fed real data: the fresh recorded days in `~/watchpost-0.19.0-plan-evidence/fresh/week` through `ionomapstest.Open`/`Replay`. They cover:
  - 120 reaches (5 origins × 6 frequencies × 4 hours);
  - a 5-day soak of updates every 10 minutes;
  - concurrent Update plus answers under `-race`;
  - a midnight-crossing cache test;
  - go-tuiMaps frames with real, adversarial and 64×-sized grids, and a 4× terminal.

---

##### Is there I/O inside a loop (N+1), or repeated work that could be batched or memoized?

- **No network N+1.** GIRO's burst is up to 40 asks one after another (`go-ionomaps internal/giro/asker.go:150-160`). That is the ruled throttle (D-39), not a defect.
- GloTEC's index is fetched on every update without validators (`go-ionomaps update.go:247`). It is 38 KB gzipped on the wire (D-141), so that is fine.
- watchpost batches the chart's answers into one `Bands` call and one `Path` call (`platform/propanswers/answers.go:75,83`).
- Two cases of repeated work worth memoizing: RP-5 and RP-6.

**RP-5 — Perf. The reach recomputes, point by point, values that are fixed for the whole answer.**
- Every control point works out the year fraction again through `time.Date`/`AddDate` (`go-ionomaps internal/magcoords/magcoords.go:112`, reached from `internal/magcoords/gyro.go:98`). `time.Date` alone is 4.8% of the reach's CPU; `yearTable.value` is 17.8% cumulative.
- `sunAt` scans back up to 144 ten-minute steps for every control point in darkness (`go-ionomaps answers.go:400`): 8.6% of the reach's CPU.
- On real snapshots the reach took a mean of 111 ms and at most 265 ms, with 0.8–7.9 MB and up to 68k allocations per call. That matches FR-1.17's figures (0.16 s; 0.26 s for the day's first). It runs off the UI goroutine.
- Severity: Minor. Simplify/Delete? N.
- Action: compute the year weight once per answer, and memoize hours-since-sunset per cell on the sun track.

**RP-6 — Perf. Naming the map's centre scans the whole world table on the UI goroutine at every new centre.**
- Evidence: `watchpost app/mapnames.go:81`, which calls `domains/locations/geodata/near.go:124` (linear scan at `near.go:146-160`). The memo only catches the identical point.
- Measured: 1.05–1.08 ms per pan step. The first call costs 4.9 ms, 4.9 MB and 68k allocations, because it parses every row. One scan is about 10% of a 400×110 frame.
- Severity: Minor. Simplify/Delete? N.
- Action: bucket the parsed rows by latitude band, or by the half-degree step the centre already moves in.

##### Can any allocation, buffer or collection grow without bound as input grows?

Nothing grows without bound:
- **Caches:** the typical days are capped at 2 (`go-ionomaps typical.go:34`), the background cache holds 1, hours ahead at most 24, the map's problems list 50 (`watchpost app/mapproblems.go:16`), and the history day cache 256 entries / 8 MB (`platform/history/history.go:272`).
- **Input caps:** GloTEC's index 4 MB (`internal/glotec/index.go`), reply bodies per host (`platform/httpx/plain.go:152`), the library's saved state 64 KB, and a go-tuiMaps grid 1,048,576 cells (`internal/overlay/field.go:13`).
- **5-day soak:** the library's heap levelled at 12.6 MB from day 1 (12,639 → 12,648 KB).
- **At 64×:** a 0.25° grid costs the same per frame as a 2° one (9.6 ms smooth; Set is lazy, the first frame 10.9 ms). Frame cost follows screen size, not grid size.

The bounded collections are still too large (RP-1).

**RP-1 — Perf. Each hour ahead copies tables the rulings say it shares, and G1 never measured the worst case.**
- D-131 says the hours ahead "share the climatology cache's fields … one nearest-station table per snapshot". The code does otherwise:
  - `Ahead` calls `s.clone()` (`go-ionomaps ahead.go:47`). That copies `Now` (immediately replaced), `Disturbance` (immediately cleared), `NearestKm` and the bitsets (`update.go:446-451`).
  - It then builds two new fields of its own (`ahead.go:35`).
  - Each typical day allocates all 24 hours' cells up front, 3.1 MB, the first time any answer touches it (`typical.go:69`).
- watchpost keeps up to 24 hours ahead (`modes/tty/map_prop_ahead.go:79-82`).
- Measured with all 24 held: the live heap rose by 11.4 MB, from 2.5 to 13.95 MB. That is 189 KB per hour ahead (4.5 MB), plus two typical days (6.2 MB). `NearestKm` is not shared with the parent snapshot (pointer check: false). This sits against D-112's 15 MB proxy before any reach or chart is worked out.
- G1's use arm steps one hour ahead and straight back (`watchpost scripts/quality/workload.expect:58-59`), at 149×38 (`:32`). So the 48-hour RSS figure includes at most one hour ahead.
- Severity: Important. Simplify/Delete? Y: the clone-then-overwrite can go.
- Action:
  - Build the hour ahead without cloning the parts it overwrites.
  - Share the parent's immutable `NearestKm` and `NearestUnscored`.
  - Allocate typical-hour cells lazily, per hour.
  - Add an alloc or heap pin that holds all 24 hours.
- **Fork for the HUM LEAD:** P-6 ("a snapshot in slices of its own") conflicts with D-131 (shared). The code follows P-6. One of the two needs re-ruling.

##### Is the algorithmic complexity worse than the problem requires (nested scans, repeated sorts)?

- Assimilation is a Cholesky fit, O(n³), with n ≤ about 40 readings (`internal/assimilate/assimilate.go:137-163`). Fine.
- The reach's polar grid (2,880 nodes, worked out lazily) is sound.
- `Bands` over 64 origins took 180 ms, but no caller asks for more than one.
- Two findings:

**RP-2 — Perf. The typical-day cache throws away both days whenever a third is asked.**
- Evidence: `go-ionomaps typical.go:57-58` replaces the whole map once `len(t.days) >= keptDays`.
- Measured across midnight: the old snapshot (23:50) and the new one (00:10) answered alternately, with an hour ahead reaching the day after (D, D+1, D+2). Each round cost 14.8 MB and 86 ms and never settled. The new snapshot alone cost 1.1 MB and 2.8 ms: about 29× worse.
- It occurs only during transitions (an old snapshot still being answered, or `Ahead(24)` just after midnight).
- Severity: Minor. Simplify/Delete? N.
- Action: evict only the oldest key (keep the last 2–3).

**RP-4 — Perf. The frame writer throws away its line buffer on every row.**
- Evidence: `go-tuimaps internal/render/frame.go:1092` calls `r.line.Reset()`, which drops the buffer. Each changed row then grows its builder again by doubling.
- In the frame's memory profile, `strings.Builder` writes are about 60% of allocated bytes; that is roughly twice the output size.
- Measured per pan frame at 400×110: real field 597 KB and 6,061 allocations; adversarial field 3.8 MB and 60k allocations.
- Separately, night ownership (`Canvas.vote`/`around`, from `basemap.go:538-561`) is recomputed for every cell of every changed frame: 27% of frame CPU.
- Severity: Minor. Simplify/Delete? N.
- Action: `Grow(len(lines[row]))` after `Reset`; cache night ownership per version of the night overlay.

##### Is blocking or expensive work on a latency-sensitive or UI path that should be async or deferred?

Clean, apart from RP-6 and the frame cost in RP-3:
- **Off the UI goroutine:** Update runs off it in a goroutine with a deadline (`watchpost domains/propagation/propagation.go:169-188`), and so do the answers and hours ahead (`modes/tty/map_prop_chart.go:147-154`, `map_prop_ahead.go:50-56`).
- **The library lock:** `Update` holds the library's lock across network I/O for up to the 60 s deadline (`go-ionomaps update.go:96-97`). No UI-path call takes that lock: snapshot methods don't, and `State()` runs on the update goroutine.
- **On the UI goroutine:** `Set` costs nothing (lazy), and a reconcile skips overlays that haven't changed (`modes/tty/map_temp.go:424`).

**RP-3 — Perf (baseline). The frame evidence for M5 uses unrealistic fields.**
- `GlobeFieldPan`'s field varies with latitude only, so its contours are straight lines (`go-tuimaps globe_bench_test.go:13-20`).
- watchpost's propagation pins use a constant field (`watchpost modes/tty/map_prop_draw_test.go:20-31`, used by `map_prop_alloc_test.go:21`). The budget measures `View()`, which never calls the library.
- `BenchmarkPropRenderMap` (`map_prop_alloc_test.go:48-54`) redraws an unchanged view (18.9 µs, 124 allocations). That measures the row cache, not a frame.

Measured pan frames:

| Frame | Time per pan frame | Bytes / allocations per frame |
|---|---|---|
| Real MUF + real reach, 400×110 | 10.1–12.3 ms | 597 KB / 6,061 |
| The benchmark's own field, 400×110 | 10.7 ms | 405 KB / 2,598 |
| Random-noise field, 400×110 (adversarial) | 18–21 ms (over M5's 16 ms) | 3.8 MB / 60k |
| Real field, 800×220 terminal | 32–42 ms | 2.2 MB |

Real data meets M5, with less margin than the benchmark shows.
- Severity: Minor. Simplify/Delete? N.
- Action:
  - Give `GlobeFieldPan` a structured field: longitude structure and closed contours, shaped like a recorded field.
  - Make watchpost's frame benchmark change the view on every op.
- **For the HUM LEAD:** M5 is ruled only up to 400×110 (problem-statement Targets row M5). Whether larger terminals are in scope is your call.

##### Under concurrency, is shared state safe, and is cache invalidation actually correct?

- **Race detector:** 30 Updates with options changing (no stations / correction) alongside 6 goroutines working Ahead/Bands/Path, plus `State()`. Under `-race`: clean.
- **Typical cells:** atomics; two goroutines filling the same cell write the same value (`typical.go:12-16`).
- **Background cache key:** moment, F10.7 mean and grid pointer (`aheadcache.go:30`). Correct.
- **watchpost's answer keys:**
  - The reach key has no noise field (`map_prop_freq.go:53-58`). That is still correct: a noise change rebuilds the snapshot at the same `Computed`; `sameSnapshot` compares the noise (`map_prop_update.go:146-148`) and bumps `built`.
  - Hours ahead are dropped on any rebuild (`map_prop_update.go:124-126`).
- **No correctness defect found.** The one invalidation problem is RP-2's over-eviction.

##### Was the performance baseline measured under realistic data and load, or only the trivial case?

Mixed:
- **The reach:** go-ionomaps' benchmark uses a flat snapshot (`reach_bench_test.go:8-27`), but my real-data runs confirm its figures (real mean 111 ms against the flat 162 ms). Acceptable.
- **The chart and centre:** watchpost's benchmarks (`platform/propanswers/bench_test.go:13-33`) use a hand-made flat snapshot with no shared typical days, so they pay the day's-first cost on every op: 23 ms and 8.3 MB. On real data, warm, `Bands` plus `Path` over 6 targets took about 6 ms. Conservative, which is fine.
- **The frame:** not realistic (RP-3).
- **G1:** 149×38, one interaction an hour, a single hour ahead. Right for a station's long-run cost, but it never exercises the hours-ahead memory worst case (RP-1).
- Every target is from one machine (an Apple M5 Pro, D-F15). Slower station hardware is unmeasured; that is a recommendation, not a defect.

---

**Opinion: exit REVIEW.** There are no Critical findings and one Important (RP-1). It is bounded and inside the proxy, but it contradicts D-131 and G1 never measured it. Fix RP-1 first: stop the clone-then-overwrite in `Ahead`, share the parent's immutable tables (or have P-6 vs D-131 ruled), and allocate typical hours lazily. Then add a pin that holds all 24 hours. RP-2 to RP-6 are Minor and can be batched.
