---
title: "0.19.0 REVIEW exit red team, round 4 - the reviewers' reports, verbatim"
date: 2026-10-26
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - three fresh blind reviewers (D-290), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 4."
---

# Round 4, the reports

Up: [red-team review](../red-team-review.md) · [round 3](review-round-3.md)


---

## Docs Quality, round 4 (R4D): 0.19.0 REVIEW exit red team (D-290)

**Tree state at the start, which every finding below was reproduced against:**
- watchpost `d63f499e`: ` M …/02-analysis/rulings.md`. That is the uncommitted D-290 row, not my change.
- go-ionomaps `08f71ef`: clean.
- go-tuimaps `6dbfdf9`: clean.

At the end, both libraries also show ` M 06_docs/gate-runs.md`. Those are the green rows the required `scripts/gate --quick` writes. I left them in place, because the gate record keeps its runs.

**What I ran (all exit 0):**
- watchpost:
  - `make lint`, `lint-authoring`, `p10` (0 live / 0 unmatched / 0 unratified), `dupes` and `mutant-anchors` (821 match);
  - `go vet` and `go test -race` on `domains/propagation` and `tools/{perfsum,runlog,finaltag,p10floor}`;
  - the committed `perfsum -g1` on `g1-run-2026-10-19`, built into my scratch.
- go-ionomaps `scripts/gate --quick`: green, P10 ran. go-tuimaps `scripts/gate --quick`: green.
- `go doc -all` on both libraries' public packages.
- A scratch program (`rt4-r4d/rx`) that runs D-288's scan pattern against sample IDs.
- Checks that every test named in go-ionomaps' requirements exists (125 of 125), and that every test and mutant the gate roster cites exists.

**Not verifiable:** go-tuiMaps `a23484f` (PR #8) is not in the local clone, and I may not fetch.

### Part 1: round 3's findings

| Finding | Verdict | Evidence (checked in the code) |
|---|---|---|
| R3D-1 | FIXED | go-ionomaps `requirements.md`: `:56` (R-3.3) has the fold and `ImplausibleM`; `:75` (R-5.3) matches `asker.go:26-33,227-235` and `update.go:296-316`; `:80` (R-5.8) matches `state.go:116-131`. All 125 tests it names exist. |
| R3D-2 | FIXED | `state.go:15-30,83-91`, go-ionomaps `README.md:30`, `api-review.md:25`; watchpost `architecture.md:131`, FR-4.12 (`requirements.md:134`), `propagation.go:113-153` (`restoreWords`). |
| R3D-3 | FIXED | `constants.go:43-44` (the fold row); `snapshot.go:131`. |
| R3D-4 | FIXED, one gap | `gates.md:29,34-40`. Every cited test and mutant exists. Gap: row 29 still cites "its mutants (batch 59)", not D-273's own mutant, m544. |
| R3D-5 | FIXED | `red-team-review.md:10,234,308,355` agree; no `*/v0.2.2` tag exists. |
| R3D-6 | FIXED | go-tuimaps `README.md:154`. |
| R3D-7 | FIXED | go-tuimaps `release-notes.md:21-22,34`. |
| R3D-8 | FIXED in part; the rewrite introduces a new false claim (R4D-2) | `docs/extending.md:16-22` names the four keymaps (they are at `dashboard.go:604`, `router.go:159`, `map_pane.go:914`, `map_prop_chart.go:30`). |
| R3D-9 | PARTIAL | `problem-statement.md:105` now gives the spans (47.995 / 47.969 / 47.975) and the margin. I confirmed the 9 s myself: 14:46:33 to 14:44:42 is 47 h 58 m 09 s against 47 h 58 m 00 s. The scorer's output is unchanged (R4D-4). |
| R3D-10 | RULED (D-288). Built in part. | `ionomapstest`'s doc is clean. But one plan ID survives, and the scan cannot catch it (R4D-1). |
| R3D-11 | FIXED | `follow-ups.md` F-220; `constants.go:44` ("quiet days with no storm"). |

### Parts 2 and 3: the lens

#### 1. Does each document answer the question its title asks?

**R4D-6: go-ionomaps' README and exported docs never mention the per-ask time limits rc.8 added.**
- `README.md:22-23` says "The host supplies the network …".
- It does not say that each NOAA ask is cut at 20 s (`update.go:296`), or that each GIRO ask is cut at 15 s (`internal/giro/asker.go:33`). A cut ask counts as trouble and backs off.
- A host whose client allows 30 s (watchpost's own is at `platform/httpx/plain.go:80`) cannot learn this from anything it reads.
- Severity: Minor. Simplify/Delete? N.
- Action: one clause in the README, and one in `Update`'s doc.

Otherwise each document answers its title.

#### 2. Does anything contradict the code, or another document?

**R4D-2: R3D-8's rewrite claims "a key is named once, in the keymap of the scope that owns it". Windows hard-code their keys.**
- The claim: `docs/extending.md:16`.
- Keys hard-coded in windows:
  - `map_boxes.go:868-882` (the Overlays menu: up, down, left, right, space, enter, esc);
  - `map_prop_find.go:79,98`, `map_prop_freq.go:235,248` and `map_prop_ack.go:57` (0.19.0's own windows).
- `README.md:209-210` says of the map's key table "Each one is a keymap action, so `[keys]` … can rebind it". That table includes the Overlays-menu keys (`README.md:202`), which no `[keys]` entry reaches.
- Severity: Minor. Simplify/Delete? N.
- Action: say "every scope's commands"; the keys inside a window are fixed. Whether a window's keys should be rebindable is the HUM LEAD's question (a fork, not a ruling of mine).

**R4D-3: a 410 is noted as "a status it is not known to send", while D-247 and D-272 both name it.**
- `domains/propagation/propagation.go:310` (`namedStatus`) leaves 410 out. Its own comment says it lists "the single statuses D-247 and D-260 name".
- `propagation_test.go:121-140` pins the note for a 410.
- The rulings (D-247, `rulings.md:268`; D-272, `:293`) both name a 410 explicitly.
- A-92 (1) (`rulings.md:411`) repeats the omission.
- This came in with round 3's R3L-5 fix.
- Severity: Minor (diagnostics only). Simplify/Delete? N.
- Action: add 410 to the named statuses, or have the HUM LEAD rule that a 410 is noted.

**R4D-5: three code comments contradict rc.8's own behaviour.**
- go-ionomaps `update.go:131`: "a failed or unanswered file does not [stop the update]". Since D-285, an unanswered ask stops NOAA's round (`:309-312`).
- `internal/giro/asker.go:144-153`: `Burst`'s doc omits D-286 (three timeouts in a row call GIRO down, whatever answered) and D-287 (the failed stations go to the back). The function body does both.
- watchpost `Makefile:108-109` and A-92 (6) say the say leg's 11 minutes are "at most four [renders] and one retry". `synth_test.go:257,262,273` makes at most three renders, so the most is four with the retry: 8 minutes.
- Severity: Minor. Simplify/Delete? N.
- Action: reword all three.

**R4D-7: "times only" is false for the stored state file.**
- watchpost `README.md:268` says `propagation-throttle.json` holds "times only, no reading". `CHANGELOG.md:26` says "times only". FR-4.12 says "times, never a reading".
- The state also holds NOAA's F10.7 mean (`state.go:53-55`, D-249).
- R3D-2's remediation touched FR-4.12 and left this.
- Severity: Minor (it is a claim about what is stored on the listener's disk). Simplify/Delete? N.
- Action: "times, and NOAA's solar flux mean; never a GIRO reading".

#### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R4D-1: D-288's scan cannot see a plan ID without a dot, and one survives in go-ionomaps' exported docs.**
- `go doc` for `Update` reads "assimilated onto both (G5)" (`update.go:88`). G5 is the implementation plan's task.
- The pattern (`docids_test.go:15`, the same at go-tuimaps `docids_test.go:15`) catches `[GW]\d+\.\d+` only.
- My probe confirmed it misses `G5`, `G9`, `W10`, `G-M3`, `INST-5` and any `R4?-n`, since only `R[23]` is listed.
- A-32 (6) and the record's "R3D-10: closed" claim a scan that names plan prefixes. It does not.
- Severity: Minor. Simplify/Delete? N.
- Action: delete "(G5)"; widen the pattern to dotless plan IDs and to `R\d[A-Z]-`; add those to the test's must-catch list.

**Two smaller unbacked claims:**
- Batch 67's build log (`build-log.md:847`) says m558 was re-pointed. m558 is unchanged since `03ad4e97`. m290, m325 and m398 were re-pointed and are not listed.
- `gates.md:29`: see R3D-4's gap above.
- Severity: Minor. Action: correct the list; cite m544.

#### 4. Is the audience order right: designers, then PMs, then engineers?

The order is right: the required reading and both READMEs lead with what the listener sees.

One residue for the HUM LEAD to weigh: go-ionomaps' exported docs still use the process word "PLAN's week" (`snapshot.go:224`, `constants.go:19`, `offset.go:8`) and "fixed by ruling" (`ionomaps.go:41-42`). They also cite paths that resolve only from the feature folder, not from the module (`snapshot.go:239`, `answers.go:243`, `reach.go:35`). D-288's letter does not reach these; its intent may. That is a recommendation, not a ruling of mine.

#### 5. Is a number published without its blind spots beside it (INST-5)?

**R4D-4: the committed G1 scorer still prints the figures R3D-9 asked to qualify.**
- `perfsum -g1` prints "48.0" hours for all three arms. The judged arm is 47.969 h.
- It ends with "within one sampling interval, 2 minutes, of it: the harness's own resolution" (`tools/perfsum/g1.go:153`). Its own comment, rewritten in batch 67 (`:48-51`), now says 2 minutes is only the nominal step and the measured step is 2.11 minutes.
- Under E-19, ruling figures come only from the committed scorer. Its output cannot produce the 47.969 or the 9 s that `problem-statement.md:105` publishes.
- Severity: Minor. Simplify/Delete? N.
- Action: print hours to three places and each arm's margin; say "nominal 2-minute step".

### What I verified clean
- R3D-1 to R3D-3, R3D-5 to R3D-7 and R3D-11, against the code.
- Every requirement test in go-ionomaps, and every roster test and mutant.
- `Restore`'s partial behaviour against its docs in both repositories, and the restore note's four branches.
- The fold row and `Implausible`'s doc.
- go-tuiMaps' README, release notes and exported docs: no IDs.
- `ionomapstest`'s doc.
- The G1 spans and margin, recomputed from `samples.csv` and `phases.log`.
- Every watchpost lint, P10, dupes and mutant-anchors target, and both libraries' quick gates.

### Opinion
**Exit REVIEW** once R4D-1 and R4D-3 are fixed. Nothing is Critical or Important, every finding is a Minor, and all but R4D-3 are wording. The rest can be batched before VALIDATE.

Fix first: **R4D-1**. It is the one place where a closed ruling (D-288) is not met, and the test meant to prove it was met cannot see the miss.

---

## REVIEW lens, round 4 (R4L): 0.19.0 REVIEW exit (D-290)

**Tree state.** At the start: watchpost `d63f499e` had ` M .../02-analysis/rulings.md` (the uncommitted D-290 row, not mine). go-ionomaps `08f71ef` and go-tuiMaps `6dbfdf9` were clean. At the end, both libraries showed ` M 06_docs/gate-runs.md`. Those rows are other reviewers'. I ran both gates in local clones and did not edit any tree. Every finding was reproduced against those SHAs, in `<scratch>/rt4-r4l/`:
- `iono/` is an rsync copy (probes and mutants).
- `wp/` is an rsync copy of watchpost.
- `ionoc/` and `tui/c` are clones (the gates).

**What I ran:**
- **watchpost (tree):** `make lint` (no new findings, 6 baselined), `lint-authoring` OK, `p10` (0 live, 0 unmatched, 0 unratified; 23 floors met), `dupes` (2 groups, both ratified), `mutant-anchors` (821, all match).
- **watchpost (`wp/` copy):** `go vet` clean on `domains/propagation`, `modes/tty`, `tools/runlog`. `-race` green on `domains/propagation` and `tools/runlog`, and on the Propagation tests in `modes/tty`, except my own probe.
- **go-ionomaps `scripts/gate --quick`:**
  - The first run FAILED only on `TestTheP10MirrorMatchesTheLedger`. That was my clone's fault: it lacked the untracked `.a2dh-p10-exemptions.yml`.
  - After I copied that file in, the rerun was **green**, but P10 was NOT RUN (the clone has no `.a2dh.yml`).
- **go-tuiMaps `scripts/gate --quick`:** **green**.
- **9 mutants, applied to the copy** and each restored with a check that it matched the tree again: 7 killed, 2 survived (M7, M9 below).
- **Probes** `r4l_probe_test.go`, `r4l_timing_test.go` (go-ionomaps) and `modes/tty/r4l_probe_test.go` (watchpost), in scratch only.

### Part 1: round 3's findings (and round 2's PARTIALs)

| Finding | Verdict | Evidence I checked |
|---|---|---|
| R3L-1 | **FIXED** | go-ionomaps `update.go:296,306-313`: NOAA's 20 s per ask; a timeout stops the round and sets `trouble`. `TestAStalledNOAAStopsItsRoundAndGIROIsStillAsked` (`giro_update_test.go:481`) asserts GIRO is asked and the 10-then-15-minute back-off. Mutant M5 (a timeout no longer stops the round) is killed. Remainder: R4L-1, R4L-3. |
| R3L-2 | **PARTIAL** | `asker.go:235,257`: three timeouts in a row call GIRO down, but only while 15 s or less of the burst has passed. See R4L-1. The words are fixed (`map_prop_update.go:376-379`). |
| R3L-3 | **FIXED for fast failures** | `internal/stations/stations.go:187-191`. `TestStationsDeadAtTheHeadStopOneBurstOnly` (`asker_test.go:160`) kills M1 (the head move deleted). Stations that hang rather than fail fast still stop bursts: R4L-1. |
| R3L-4 | **FIXED** | `update_test.go:443-452` tests the second back-off. M6 (`err != nil` dropped from `trouble`) is killed by `TestARefusalBacksOff`. |
| R3L-5 | **PARTIAL** | 404 and 5xx are now named (`propagation.go:310,318`), but 410 is not: R4L-4. |
| R3L-6 | **FIXED** | `state.go:125-133` (one line, a sentinel for each part); `propagation.go:141-153`; watchpost tests all three cases. The library's own test misses the case where both parts are refused: R4L-6. |
| R3L-7 | **FIXED** | `map_prop_update.go:147-166`: a count is noted only when it changes. `TestEveryRefusedReadingIsNoted` (`map_prop_states_test.go:181`) asserts the frequency. |
| R3L-8 | **FIXED** | `map_prop_update.go:388` uses `ionomaps.UnreadSolar`. |
| R2L-1 / RL-1 | **PARTIAL** | Fixed for fast failures through D-287; stations that hang are R4L-1. |
| R2L-4 | **FIXED** | The restore note now names the mean (`propagation.go:141-153`). |
| R2L-5 | **FIXED** | `Rejected` is worded for everything it counts and noted once per change (`map_prop_update.go:158`). |

---

### 1. Happy Path Bias

**R4L-1. D-286's "three timeouts in a row" fits the 60 s update only when GIRO stops in the first ~15 s of a burst. A burst of 39 takes ~36 s (D-275's PLAN figure). So a GIRO that hangs after about 16 answers is cut by the host's deadline before its third timeout: Answered, not paused, Unfinished. That is R3L-2 and R3C-1's case.** Severity: **Important**. D-286 says "D-275's sizing then holds", and its test covers only one answer before the hang.
- **Probe P3** (`r4l_timing_test.go`, scaled 1 ms = 1 s: 15 s per ask, a 60 s deadline, 0.9 s per answer). GIRO hangs after k answers:
  - k = 1 or 10: Unavailable, paused.
  - k = 16, 20 or 30: **Answered, not paused, cut by the deadline**.
- **A GIRO that alternates a hang with a 5xx after one answer** is never called down: 40 asks, 20 hangs (5 min of asks against 60 s), Answered. `timeoutsAfter` resets on any 5xx (`asker.go:212-217`).
- **Three adjacent stations that hang** defeat D-287's "cannot stop two bursts running" (`TestStationsDeadAtTheHead…` uses only 500s):
  - With instant answers (P2): Unavailable and paused **every hour** for 8 hours. D-287 puts the trio at the end of the burst, where D-286 calls GIRO down.
  - With real-scaled timings (P4): cut by the deadline every hour for 7 hours, 27 to 38 asked.
- **The NOAA side has the same arithmetic.** Up to 5 asks × 20 s (`update.go:296`) is more than 60 s. In probe P5, NOAA answered every ask in 14 to 17 s: 3 of 12 updates came back Unfinished with no back-off. GIRO was still asked (80 asks), so R3L-1's harm did not return.
- **What it costs:**
  - A hung GIRO is not backed off (D-39).
  - The library's lock is held for the whole 60 s every hour.
  - The words are now true ("GIRO's burst stopped before every station was asked").
- Simplify/Delete: N.
- Action, a fork for the HUM LEAD (D-286 against D-275):
  - Call GIRO down when the burst's remaining deadline cannot fit the remaining run of timeouts. Or count elapsed time against a burst budget.
  - Test with realistic answer times and a hang at k ≥ 16, and with a trio that hangs.

### 2. Systemic Failure Modes

R4L-1's root is still there: no remediation checks per-ask timeouts × asks against the host's one deadline. R4L-3 adds the host's client timeout as a third, unstated part of that coupling.
- When the machine's own network drops, NOAA (Offline) and GIRO (Unavailable) are each blamed on their own. Rounds 1 and 2 covered this; nothing new.

### 3. Gate Validity

**R4L-5. D-288's ID scan passes while exported docs still carry plan and parity IDs.** Severity: **Minor**. Simplify/Delete: N.
- The pattern (`docids_test.go:15` in both libraries) has no `P-\d+` and catches `G` only as `G\d+\.\d+`.
- `go doc -all` shows:
  - go-ionomaps: `(P-6)` (`snapshot.go:11`) and `(G5)` (`update.go:88`, in the same doc where batch 67 removed `G4.2`).
  - go-tuiMaps: `P-54` to `P-61` (`view.go:11,14,117`, `places.go:18,58,91,162`, `facts.go:311,341,368`).
- Action: widen the pattern to `P-\d+` and `G\d+`, then remove the IDs.

**R4L-6. The library does not test `Restore` refusing both parts.** Severity: **Minor**. Simplify/Delete: N.
- M9 (the both-parts branch disabled, `state.go:125`) survives go-ionomaps' suite. Only watchpost's test, in another repo, holds it.
- Action: add the both-parts case to `TestABadStateIsRefused` (`state_test.go:102`).

Otherwise, M1 to M6 and M8 are killed by the tests the record names.

### 4. Adversarial Input

**R4L-4. A GIRO 410 is still noted as "a status it is not known to send", though D-247 and D-272 name it ("a 410 for one station is that station's failure").** Severity: **Minor**. Simplify/Delete: N.
- Code: `namedStatus` (`domains/propagation/propagation.go:310`). Its test asserts that a 410 is noted (`propagation_test.go:126-140`).
- A-92 (1) says "D-247's statuses" but leaves 410 out.
- Action: add 410 to the named statuses and fix the test.

Otherwise:
- A server that sends headers and then trickles the body is bounded by the per-ask timeout, which also ends the body read.
- A probe station (not live) that fails is passed over by D-287's head move (`stations.go:188`).

### 5. Graceful Degradation

**R4L-2. watchpost's "this update is GIRO's burst" flag outlives a closed mode, so the first snapshot after reopening can be said as GIRO's burst.** Severity: **Minor**. Simplify/Delete: N.
- `stopPropagation` (`map_prop_update.go:126-134`) bumps `propGen` but never clears `propBurstAsked` (set at `:180`). The burst's late result is then dropped by the generation check (`:139`), so the flag is never reset.
- Probe `TestR4LStaleBurstFlag` (FAILS): a NOAA round stopped after reopening is said as "GIRO's burst stopped before every station was asked … · no station readings this hour: the background alone". The line contradicts itself.
- Action:
  - Clear the flag in `stopPropagation`.
  - Better, a fork for the HUM LEAD against A-92 (3): have the library say in the snapshot that it came from the burst, so watchpost need not infer it.

R4L-1's hours: the words are now true, but a hung GIRO is never backed off.

### 6. Test Confidence vs. Test Results

Each new test proves its fix only at the edge of its fixture:
- D-286's test: one answer, then a hang.
- D-287's test: 500s only.
- None of them uses realistic answer times. See R4L-1.

The rest proves behaviour: the stall test asserts GIRO is asked and both back-off steps, and the restore tests assert each part.

### 7. Dependency Fragility

**R4L-3. D-285 and D-286 recognise a timeout only when the library's own per-ask context fires (`update.go:308`, `asker.go:256`). So the fix depends on the host's client timeout being longer than the library's.** Severity: **Minor**. Simplify/Delete: N.
- No test holds the production values. M7 (`noaaAskTimeout` set to 45 s) **survives**. Above watchpost's 30 s client timeout (`platform/httpx/plain.go:80`), the client's error is "an error that is no timeout": the round goes on, and R3L-1 returns.
- A host whose client timeout is under 15 or 20 s gets D-270's behaviour for NOAA, and R3L-2's for GIRO.
- So D-285's "never depends on the host's timeout" is not true.
- Action:
  - Classify any timeout error (`errors.Is(err, context.DeadlineExceeded)`, or `net.Error.Timeout()`) as a timeout.
  - Add a test that pins `noaaAskTimeout`, and `3 × askTimeout`, under the documented host budget.

Pins match the reviewed code: `go.mod:9-10` names rc.8 = `08f71ef` and rc.7 = `6dbfdf9`, and `go.sum:13-16` agrees.

### 8. Problem statement present, real, honored

PS-1 and PS-2 are present (`problem-statement.md:14-31`). PS-2's M1 (53.7 %) and M4 (68.4 %) are still sent to VALIDATE (agenda item 1). PS-1 remains a hypothesis (D-217).

**R4L-7. The Broadcaster arm (PS-1's cost, agenda item 3) measures `a444b000` (batch 64), before rounds 2 and 3 changed GIRO's asker and NOAA's round. The agenda does not name the build.** Severity: **Minor**. Simplify/Delete: N.
- The commit is recorded only outside the tree (`g1b-run-2026-10-25/run.txt:1`); `validate-agenda.md:21` gives no commit.
- Action: name the build measured in item 3, and say why batches 65 to 67 do not change its cost, or measure again.

---

### Verified clean

- R3L-1's fix: a stall stops NOAA's round, backs NOAA off and still reaches the GIRO switch.
- R3L-4, R3L-6, R3L-7 and R3L-8; R2L-4 and R2L-5.
- D-287 for stations that fail fast.
- The restore note's four wordings.
- The 5xx class and 404 named.
- The lint, P10, dupes and anchor legs; both quick gates; the vet and race runs listed above.

### Opinion

**Do not exit REVIEW.** Nothing is Critical. But R4L-1 shows D-286's fix holds for only the first ~40 % of a burst, which leaves R3C-1's Important case open under the same deadline arithmetic this lens has raised in every round. Its fix and its realistic-timing test are small. R4L-2 to R4L-7 are Minor and can be batched.

**Fix first: R4L-1.** Make the burst call GIRO down whenever its remaining deadline cannot hold the remaining timeouts, and test a hang after 16 or more answers with realistic answer times.

---

## Code Quality, round 4 (R4C): REVIEW exit red team, 0.19.0

**Verdict: do not exit REVIEW.** There is one Important, R4C-1. D-286 calls a hung GIRO down only when the hang starts in roughly the first 15 s of a burst. A GIRO that hangs later still runs the host's update out and is never backed off, which is R3C-1's own failure again. D-287 makes it worse: it moves stations that hang to the end of the burst. Everything else is Minor.

### Tree state and runs
- **Tips:** watchpost `d63f499e`, go-ionomaps `08f71ef`, go-tuiMaps `6dbfdf9`.
- **`git status --short` at the start:**
  - watchpost: ` M .../02-analysis/rulings.md` (the D-290 row)
  - go-ionomaps: clean
  - go-tuiMaps: clean
- **Changes during my review that are not mine:** both libraries gained ` M 06_docs/gate-runs.md`, rows 04:07:24Z (go-ionomaps) and 04:15:17Z (go-tuiMaps). I ran both gates on byte copies in my scratch directory. My rows, 04:21:17Z and 04:28:44Z, are in the copies only, with the same tree hashes as the tips.
- **I edited no tree.** Every mutant and probe was a `-overlay` from scratch. Every finding was reproduced against the tips in the states above.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `make lint-authoring`, `make p10` (0 live / 0 unmatched / 0 unratified), `make dupes` (2 groups, both ratified), `make mutant-anchors` (821), `go vet ./...`.
  - `go test -race` on modes/tty, domains/propagation and tools/{runlog, reachscore, perfsum, p10floor}.
  - cmd/watchpost gate tests.
- **Libraries:** `scripts/gate --quick` (copies) green in both, P10 included. `go test ./...` passes in go-ionomaps.

### Part 1: round 3's findings, checked in the code

| ID | Status | Evidence |
|---|---|---|
| R3C-1 | **PARTIAL** | D-286 is built: go-ionomaps `internal/giro/asker.go:235,256-258`, test `asker_test.go:246`. It holds only for a hang that starts early in the burst: R4C-1. |
| R3C-2 | FIXED | go-ionomaps `state.go:20-30,124-133`; watchpost `domains/propagation/propagation.go` `restoreWords`. All three cases are tested in `propagation_test.go` (`TestTheThrottleStateOutlivesARestart`). |
| R3C-3 | FIXED | `propagation.go` `namedStatus` and `noteStatus` (404 named; 5xx skipped as a class). The test covers 404, 500, 503 and 418. |
| R3C-4 | **PARTIAL** | The four IDs are gone, but `snapshot.go:11` "(P-6)" remains, and the scan test misses whole families: R4C-2. |
| R3C-5 | FIXED | `modes/tty/map_prop_update.go:388` uses `ionomaps.UnreadSolar`. |
| R3C-6 | FIXED | go-ionomaps `update.go:279-290` (`trouble`), `:296-303` (`ask`'s doc names `l.trouble`). |
| R3C-7 | FIXED | Dropping `err != nil` from `l.trouble` (my i9, = m070) is killed by `TestARefusalBacksOff`. The hung fake has a guard (`asker_test.go:230-240`). |
| R3C-8 | FIXED | `tools/runlog/main_test.go:138-166`. m565 and m566 match the code. |
| R3C-9 | FIXED in code | `Makefile:111` (`-timeout 11m`); `synth_test.go:816` (a hang only when the render returned an error). The new comment's arithmetic is wrong: R4C-6. |
| R3C-10 | FIXED | `made_work` is gone from go-tuiMaps `scripts/gate`. |
| R3C-11 | FIXED (A-32 (5)) | `update.go:217-256`: every helper is a method except `gridUsable`, and its doc gives the reason. |
| R3C-12 | FIXED | `cmd/watchpost/gates_test.go:819-843`, the race-only case both ways. |

**Round 2's unrecorded Minors:**
- reachscore's check is now written once (`tools/reachscore/report.go`, `writeRow` returns `stands`): FIXED.
- `madeFromHeld`: FIXED.
- The alias field lists: PARTIAL, R4C-9.

### 1. Understandability
- **R4C-7 (Minor), go-ionomaps: what `Unasked` says it does is not what the code does.**
  - The doc says "the stations whose failures stopped it go behind the rest" (`internal/stations/stations.go:176-179`).
  - In fact `Burst` passes every failed or timed-out station of the burst on any stop: a 429, a 401/403, or a host cancel inside `ask` (`asker.go:181,194,197`). `Unasked` then moves the head past the last of them, even when that failure stopped nothing.
  - The list is built only for its last element to be read, through a loop that runs at most once: `for _, c := range failed[max(len(failed)-1, 0):]` (`stations.go:187`).
  - Simplify/Delete: **Y**. Action: pass the one code, the last station whose failure counted toward the stop, and make the doc say which stops move the head.

### 2. Maintainability
- **R4C-3 (Minor), watchpost: the flag for "the update in flight is GIRO's burst" outlives the session it describes.**
  - `propBurstAsked` lives on `mapPane` (`map_pane.go:81`), not on `propSession`.
  - `stopPropagation` resets the session and bumps `propGen` but leaves the flag set (`map_prop_update.go:126-134`). The dropped reply returns before the flag is cleared (`:139-140`).
  - **Probe** (overlay test): StationsNext, then the burst asked, then the mode left, then reopened, then NOAA's round comes back Unfinished. Result: `propSnapBurst` true, and the status reads "GIRO's burst stopped before every station was asked … no station readings this hour". That is false, and it contradicts itself.
  - Simplify/Delete: N. Action: move the flag into `propSession`, and add this case as a test.

### 3. Complexity
- R4C-7 (the one-pass loop).
- `timeoutsAfter` is a free function beside `count`, a method (`asker.go:206-221`): two shapes for one job. Cosmetic, folded into R4C-7.

### 4. Necessity
- R4C-7: `failedCodes` holds a list when one string would do. Nothing else found.

### 5. Name semanticism
- **R4C-5 (Minor), go-ionomaps: `refused` now means "stops the round".**
  - D-285 made a timed-out NOAA ask return `refused=true` (`update.go:305,308,312`).
  - The variable in `Update` (`:129-138`), the return value of each `refreshX`, and their docs ("refused is whether it was refused with a 429", `:331-332,383,409,429-430`) now name the wrong thing.
  - Simplify/Delete: N. Action: rename it `stops` (or `endsRound`) and fix the four docs.

### 6. Code documentation
- **R4C-4 (Minor), go-ionomaps: the docs went stale after D-285 and D-286.**
  - `snapshot.go:154` is the exported `GIROUnavailable`, which a library user sees. It still says "a 429, or three stations failed in a row before any answered", and leaves out three timeouts after an answer.
  - `asker.go:45` (`Unavailable`) and `asker.go:144-152` (`Burst`'s doc) say the same, without D-286.
  - `update.go:131` says "a failed or unanswered file does not [stop the update]". That is false now: an unanswered ask stops the round.
  - `ErrBadState`'s message, "the state given is not one this library keeps" (`state.go:20`), is now wrapped around states this library did write and partly took, so the watchpost note quotes a falsehood.
  - `asker.go:27-31` still says a GIRO that stops answering is "never left to run the update out". See R4C-1.
  - Simplify/Delete: N. Action: correct each.
- **R4C-6 (Minor), watchpost:**
  - `Makefile:108-109` and A-92 (6) say the say leg's 11 minutes are "at most four [renders] and one retry".
  - The test makes at most three renders (`synth_test.go:257,262,275`) and one retry: 8 minutes.
  - The budget is safe; the stated derivation is false. Action: say three and one, or derive `-timeout` from the bound.

### 7. Testability
**Mutants (overlays):**
- **go-ionomaps, 9 applied, 4 killed.**
  - i1 (no head move) is killed by `TestStationsDeadAtTheHeadStopOneBurstOnly`.
  - i7 (`timeouts > 3`) is killed.
  - i9 (= m070) is killed.
- **watchpost, 3 applied, 3 killed:** the burst flag not set, the burst flag not cleared, and the `Implausible` change-check dropped.

**R4C-8 (Minor), go-ionomaps survivors:**
- **i2:** the head moved past the *first* failed station, not the last. With four or more dead stations at the head the difference matters, and the test uses exactly three (`asker_test.go:160`).
- **i3:** timed-out stations are not queued behind the others (`asker.go:193`). D-287 is tested with 500s only.
- **i4:** `timeoutsAfter` never resets after an answer, so three scattered timeouts would call GIRO down. No test mixes timeouts with answers.
- **i5:** a timeout is not counted in `inARow` or `Failed` (`asker.go:200`). `Failed` for timeouts, and mixed failure/timeout runs before any answer, are unpinned.
- **i8:** when both parts are refused, only `ErrBadGIROState` is wrapped (`state.go:125`). It survives inside the library. Only watchpost's "neither was taken" case, run against rc.8, would catch it.
- i6 is equivalent (the host's context is done either way).
- Simplify/Delete: N. Action: a test for each of i2 to i5 and i8, with mutants.

### 8. P10 conformance
- `make p10` and both library P10 legs are green. No floor or ledger row changed in this range.
- R4C-7's one-pass loop looks shaped to avoid a nested `if`. If the density count drove it, that is metric-shaped code. I could not confirm the motive.
- No exemption was self-issued.

### 9. Evidence soundness
**R4C-1 (Important), go-ionomaps: D-286's "D-275's sizing then holds" is true only for a hang in roughly the first 15 s of a burst.**
- **The sizing check proves less than the claim.** The check is `failuresInARow*was >= 60s` (`asker_test.go:225`): three 15 s timeouts fit in 60 s. But the asks before the hang take time too. PLAN's burst of 39 took 36 s (`propagation.go:44-46`).
- **Probe,** an overlay test with time scaled 100×: asks answer in 9 ms, `askTimeout` is 150 ms, the host deadline is 600 ms, GIRO answers N stations and then hangs.
  - N=1 and N=10: called down, `Unavailable`, paused.
  - N=15, 17, 20 and 30: the host deadline hits first. Outcome `Answered`, not paused, not backed off.
  - At real scale the third timeout lands past 60 s whenever the hang begins more than about 15 s in, around the 16th of 39 asks.
  - The update is cancelled, said Unfinished, the lock is held the full minute, and the same happens every hour. That is R3C-1's effect.
- **D-287 makes it worse for stations that hang.** It puts the failed stations last.
  - Probe without a host deadline: three hanging stations, every other station answering. For five hours running, every burst asks 39, ends `Unavailable` and is paused: "GIRO unavailable this hour" although every reading came.
  - Under watchpost's real 60 s, the same trio sits after about 33 s of answers, so each hour ends Unfinished instead.
- **Alternating hangs and answers never reach three timeouts in a row.** About 20 timeouts × 15 s runs the update out. `TestAGIROHangingAfterAnAnswerIsCalledDown` hangs after the *first* answer only, so neither the test nor the record covers any of these cases.
- Simplify/Delete: N. **Fork for the HUM LEAD** (D-286 chose option 1 of R3C-1's fork; this is the evidence for option 2):
  1. A burst deadline under the host's update.
  2. Stop the burst when the time left on the host's context is under one `askTimeout`, calling GIRO down if the last asks timed out.
  3. Keep D-286, correct the comments, and record it as a known limit.
- Under 1 or 2, add a test of a hang after 20 answers under a host deadline, and a hanging-trio test.

**Other evidence points:**
- `TestStationsDeadAtTheHeadStopOneBurstOnly` asserts `len(asked) > 3`, while its doc says "every station asked". The check is weak but adequate for D-287.
- watchpost's new tests prove what they claim (w1 to w3 killed).

### 10. Evasion lens
**R4C-2 (Minor), both libraries: D-288's scan test passes while IDs remain in `go doc`.**
- D-288 says "a test scans `go doc -all` for every ID pattern".
- The regex is a denylist of prefixes (`docids_test.go:15` in both libraries). It has no `P-n`, no `task n.n`, and no `L<n>.<n>`.
- **What `go doc -all` still shows:**
  - go-ionomaps: `snapshot.go:11` "(P-6)".
  - go-tuiMaps: `view.go:11,14,55,117` (P-55, P-54, P-56, "task 14.19"); `facts.go:311,341,368` (P-57); `places.go:18,58,91,162` (P-58, P-61); `look.go:103` ("task 14.19"); `map.go:73` ("v0.2.0 L3.11, L3.13").
- go-ionomaps' `tools/gm3` command doc is full of IDs (G10.4, D-75, …). It is a command, not library API; I note it for the HUM LEAD's scope.
- Simplify/Delete: Y. Action: strip them, and widen the pattern to a generic `\b[A-Z]{1,4}\d?-\d+` / `\btask\b` form with an allowlist of published work, so a new family fails instead of passing.

**R4C-9 (Minor), go-tuiMaps: the alias field-list test checks only four aliases.**
- `aliasdoc_test.go:40-45` hand-lists Answer, Grid, Type and Image.
- Its own premise ("go doc does not show an alias's fields") applies to every struct alias. `Overlay`'s doc never names `During`, and `Feature`, `Span`, `LoopFrame`, `Finding`, `Warning`, `FetchOptions` and others are unchecked (`go doc -all .` confirms that no alias shows its fields).
- Simplify/Delete: N. Action: find every alias of an internal struct, by AST plus `go doc` of the target, instead of keeping a hand list.

### What I verified clean
- **go-ionomaps:**
  - The NOAA per-ask timeout is told from the host's cancel (`update.go:308`). m065, m066 and my i7/i9 kill the stall paths, and the stalled round reaches the GIRO switch.
  - `Restore`'s per-part sentinels in the single-part cases.
  - The `scalesUsable`/`scalesOf` method change is behaviour-neutral.
  - The ID strips in `answers.go`, `reach.go`, `ahead.go`, `ionomaps.go` and `ionomapstest` are doc-only.
- **go-tuiMaps:**
  - The ID strips are doc-only.
  - The two ragged comments are re-wrapped (`look.go:50-52`, `daynight.go:36-38`).
  - `made_work` is gone.
- **watchpost:**
  - The refusal counts are noted on change (A-92 (2)), tested and killed.
  - D-286's words via the flag, apart from R4C-3.
  - `restoreWords`.
  - The selector's race case.
  - runlog's `openLog` seam.
  - The perfsum comment.
  - reachscore's `stands` (the bare `return` still yields `true`).

**Fix first: R4C-1.** It is the only finding where the remediation leaves in place the failure its ruling was made to remove. A GIRO that hangs part-way through a burst still holds the library for the whole update every hour and is never backed off.

Scratch directory, with mutants i1-i9 and w1-w3, the probes and the gate logs: `<scratch>/rt4-r4c`
