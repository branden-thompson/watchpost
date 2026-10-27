---
title: "0.19.0 REVIEW exit red team, round 7 - the reviewers' reports, verbatim"
date: 2026-10-27
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - four fresh blind reviewers, each also on amd64 (D-308), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 7."
---

# Round 7, the reports

Up: [red-team review](../red-team-review.md) · [round 6](review-round-6.md)


---

## Docs Quality, round 7 (R7D): 0.19.0 REVIEW exit red team (D-308)

**Tree state.** I reproduced every finding against the start state shown below. My probes ran on a copy of go-ionomaps, never on the tree.

| Repository | At the start | At the end |
|---|---|---|
| watchpost `b16e5411` | ` M …/02-analysis/rulings.md` (the uncommitted D-308 row, not mine) | unchanged |
| go-ionomaps `68f31f3` | clean | ` M 06_docs/gate-runs.md`: my green quick row, 04:00:22Z |
| go-tuimaps `5f6bf07` | clean | ` M 06_docs/gate-runs.md`: my green quick row, 04:10:14Z |

**What I ran:**
- **watchpost:**
  - `make lint` (no new findings, 6 baselined), `make lint-authoring`, `make p10` (0/0/0), `make dupes` and `make mutant-anchors` (826 match). All exited 0.
  - `go vet` on `domains/propagation`, `app` and `modes/tty`.
  - `go test -race` on `domains/propagation` (whole), `app -run Prop|Settle|Update` and `modes/tty -run Stations|Burst|Prop|CutShort|Deadline`.
  - The same packages under `GOARCH=amd64`. Everything was ok.
- **Library gates:** `scripts/gate --quick` was green in both libraries.
- **go-ionomaps tests:**
  - `go test -race -count=10` on the round 6 timing tests (`TestABurstEndsBeforeTheHostsDeadline`, `TestOnlyTheUpdatesEndCountsStationsNotReached`, `TestAStationThatRanOutIsAskedLast`, `TestAProbeNotReachedIsAskedFirst`), at a load average of about 7. Green.
  - Under amd64: `internal/giro`, `deadline`, `stations` and `limits` with `-count=3`, plus the root package's deadline, no-length-path and fuzz-seed tests. Green.
- **Every test named in the requirements exists:** watchpost 157/157 and go-ionomaps 138/138.
- **Probes** are in `<scratch>/rt7-r7d/iono/internal/giro/zz_r7d_probe*_test.go`. They use the package's own `scaledAsks` (scaled 1:100), with a 600 ms deadline standing for 60 s. Every probe result below was reproduced on arm64 and again under amd64, with identical results.

### Part 1: round 6's findings

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R6D-1 | FIXED | go-ionomaps `update.go:329` sets `MinDeadline = 32 * time.Second`. `:353` calls `deadline.Room(ctx, 20s, 20s)`, which needs 21 s with `Margin`. watchpost `propagation_test.go:246-262` asks NOAA at exactly `MinDeadline`, and `update_test.go:605-608` does the same. `errNoDeadline` was reworded (`propagation.go:70`). |
| R6D-2 | FIXED | `asker.go:288-294` restores the counts when nothing was asked and counts `NotReached` only when the burst was neither cancelled nor empty. The snapshot doc is at `snapshot.go:144`, but see R7D-3. |
| R6D-3 | FIXED | `map_prop_update.go:367-369`: the `burst` sentence is gone. |
| R6D-4 | FIXED | `architecture.md:168` and `:174` now quote "GIRO unavailable this hour: the readings it gave first" and "NOAA's GloTEC not read this update". These match `map_prop_update.go:297,384,392` and `snapshot.go:122`. |
| R6D-5 | FIXED | `answerIn` honours its context (`asker_test.go:324-330`). It was green under race ×10 at load 7, and in my quick gate. |
| R6D nits | FIXED | The state file's contents: `requirements.md:134`, `README.md:270-271`, `CHANGELOG.md:26-27`. "R4D-7's residue": `build-log.md:877`. "U9 to U12" explained: `validate-agenda.md:21`. go-tuimaps: `scripts/gate:721` and `requirements.md:66`. `TestRaceDetectorIsToldTruly` runs and does not skip. |

### 1. Does each document answer the question its title asks?

Not fully. The title `TestAStationThatRanOutIsAskedLast` (go-ionomaps `asker_test.go:693-728`) promises more than the test checks. The test runs with no deadline, so it never reaches the case that breaks the promise. That case is R7D-1.

### 2. Does anything contradict the code, or another document?

**R7D-1: D-307's rule, "a station whose full ask ran out is asked last until it answers", is false whenever the update's end leaves that station unreached. It is then asked first.**
- **Where the rule is written:**
  - go-ionomaps R-5.3 (`requirements.md:75`): "asked last in the bursts after, until it answers";
  - `asker.go:176-177`;
  - `stations.go:59` and `:236` ("last of all");
  - go-ionomaps D-115 (`rulings.md:136`) and build log `:305`;
  - watchpost D-307 (`rulings.md:328`), whose decision adds "with two hanging stations the healthy ones are asked first every hour and all read";
  - watchpost's as-built map, node `IO`.
- **What the code does:**
  - a ran-out station at the tail is the one the deadline cuts. Its shortened ask returns `notReached`, and the burst ends through `endEarly` (`asker.go:217-218`);
  - `endEarly` calls `Unasked`, which puts the station in `leftOut` (`stations.go:231`);
  - `rotation` places `leftOut` first, before it applies `last` (`stations.go:243-254`).
- **Probe** (`TestR7DTwoHangingStations`, `TestR7DOrderTrace`): two hanging stations, the others answering at about the pace PLAN measured (0.9 to 1.1 s per ask, scaled).
  - At 11 ms (1.1 s), hours alternate. One hour asks 39 with 2 not reached. The next asks `HH......` with both hangs **first** and 14 to 16 stations not reached. That is worse than the 12 to 13 that D-307 was ruled to cure.
  - At 9 ms (0.9 s), one hanging station is asked first every hour: `H…H` and `DH…H`.
- Severity: **Important**. A ruled behaviour, written in seven places, holds only when there is no deadline, and the test that names it never sets one. Introduced by rc.11. Simplify/Delete? N.
- Action, a fork for the HUM LEAD (D-307 is theirs):
  - (a) a ran-out station stays last even when it was not reached;
  - (b) the docs say "first when the deadline left it".

  Either way, add a test over several hours, under a deadline, with two hanging stations.

**R7D-2: D-307's tail places hanging stations side by side, just before the probe, so a healthy GIRO is called down every hour.**
- **The sequence:**
  - the ran-out stations now run consecutively at the end of the polls (`stations.go:254`);
  - the rotation probe comes straight after them (`asker.go:201-202`);
  - when the probe (a dark station) also times out, that makes three full asks run out in a row, and `stops` calls GIRO down (`asker.go:325`, D-286);
  - with no hanging probe, D-305's two-at-the-end rule (`asker.go:295-297`) fires whenever the deadline falls just after the pair.
- **Probe** (`TestR7DHangingPairAndDarkProbe`): 37 healthy stations answering in 0.3 s, two hanging live stations, and dark probes that hang.
  - Hours 1 to 4 read `…HHD` with `Outcome 1` (Unavailable): GIRO is backed off and "GIRO unavailable this hour" is shown every hour, although 37 stations answered.
  - With m100 applied to my copy (the D-307 tail removed), the same run never calls GIRO down: `..HH…D`, `Outcome 0`.
  - With probes that fail fast instead, a 0.7 s pace still calls GIRO down at the end in some hours (`TestR7DHangingPairQuickDark`, hour 4).
- **Why the setup is realistic:** a station that never answers goes dark after `darkAfter` (3 days). After that it is a probe candidate, so in D-307's own scenario hanging probes are the likely ones.
- **What it contradicts:**
  - D-305 calls two adjacent hanging stations calling GIRO down "a known limit … in some hours". D-307 now makes them adjacent every hour, at the point where the deadline falls;
  - the architecture's failure table says only a hanging GIRO is called "unavailable" (`architecture.md:171`).
- Severity: **Important**. This is the D-286/D-296 class (GIRO called down while GIRO answers) that rounds 3 to 5 each fixed, now brought back by round 6's remediation. Simplify/Delete? N.
- Action, a fork for the HUM LEAD. Options:
  - the probe goes before the ran-out tail;
  - a full ask of a station already ran-out does not count toward the run;
  - accept it as a known limit and list it under VALIDATE item 7.

**R7D-3: three library docs drift from round 6's count and run rules.**
- `snapshot.go:144` says a burst "GIRO stopped counts none". When GIRO is called down at the update's end, the code counts the stations left (`asker.go:292-298`). My probe (`TestR7DCalledDownAtTheEndCounts`) gave `Outcome 1, NotReached 37`. watchpost hides the words in that case (`map_prop_update.go:411`), so no listener sees it.
- `Asker.NotReached` (`asker.go:98`) still describes the count as it was before D-302: no "live", and no "a cancel counts none".
- `Update`'s doc (`update.go:103-105`) says "two or three in a row … call GIRO down" and drops the conditions: two only as the deadline comes, three whatever answered. The README (`:26`) states them.
- Severity: Minor. Simplify/Delete? N. Action: reword the three docs; D-302's wording, "a cancel, a 429 or a denial", is the precise one.

**R7D-4: the architecture still says only closing the mode cancels an update.**
- `architecture.md:93,114` say "closing the mode does".
- Since batch 70, the app's end cancels it too, before `settleBound`'s wait (`app/mapprop.go:121-125`, A-95 (1)). The as-built map says so; the architecture does not.
- Severity: Minor. Simplify/Delete? N. Action: add the app's end.

### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R7D-5: D-307's outcome, "the healthy ones are asked first every hour and all read" (watchpost `rulings.md:328`), is recorded as closed (`red-team-review.md:639`), but no test checks it.**
- The only D-307 test (`asker_test.go:696-728`) has one hanging station and no deadline.
- R7D-1 and R7D-2 show the claim fails under the conditions it names.
- Severity: Minor, folded into R7D-1's action. Simplify/Delete? N.

Everything else checked clean: A-36's seed is in the fuzz target (`answers_field_test.go:88`), and A-35 (7) holds in my ×10 race run.

### 4. Is the audience order right: designers, then PMs, then engineers?

Nothing found. The failure table's new rows lead with what the listener sees, and the README's deadline paragraph leads with what the host gets.

### 5. Is a number published without its blind spots beside it (INST-5)?

**R7D-6: D-305's ruled "known limit" is not carried where the release's known limits are gathered.**
- `validate-agenda.md:25` (item 7) lists the changelog's known limits; D-305's is absent. It is also absent from `CHANGELOG.md` and the go-ionomaps README.
- R7D-2 shows it is not rare.
- Severity: Minor. Simplify/Delete? N. Action: add it to item 7, with its frequency stated once R7D-2 is ruled.

Otherwise nothing new: round 6's remediation published no new figure without its conditions.

### What I verified clean
- **`internal/deadline`:**
  - `Room` and `Left` match the docs, and the margin holds at exactly `MinDeadline`;
  - two full GIRO asks fit in 32 s, so D-296's call-down holds there;
  - the P10 row matches the ledger at 1.33 over 3 functions.
- **D-302:** counts are restored when nothing was asked, `NoRoom` is reset at the top of `Burst`, and a cancel during an ask puts the interrupted station first.
- **D-303:** a probe not reached is kept and asked first; my traces show `D` at position 0 the next hour.
- **D-304's wording.** **D-305:** `tally.add` resets the run on any other result.
- **A-36:** the clamp, and the NaN guard ahead of `int()`, are green under amd64.
- **watchpost:**
  - the app-context tie and its mutant (m575), and the ordering at the app's end (`dashboard.go:132`, then `:1016`);
  - m576 and m298, which asserts its target;
  - the re-pin to rc.12 and rc.10 in `go.mod`, `THIRD_PARTY_LICENSES.md`, the as-built map and the required reading's dated list.
- **go-tuimaps:** A-23, and the gate's leg label.

### Opinion
**Do not exit REVIEW.** Round 6's own fixes all hold. But D-307, a round 6 ruling, brought in two Importants: its documented order breaks under a deadline (R7D-1), and its tail calls a healthy GIRO down every hour when the probe also hangs (R7D-2). Both need a HUM LEAD fork before any code changes. R7D-3 to R7D-6 are Minor and can be batched.

**Fix first: R7D-2.** It is the same tail placement as R7D-1, so fork the two together.

---

## REVIEW lens, round 7 (R7L): 0.19.0 REVIEW exit (D-308)

**Tree state at the start:**
- watchpost `b16e5411`: ` M .../02-analysis/rulings.md`, the uncommitted D-308 row. It is not mine.
- go-ionomaps `68f31f3` and go-tuiMaps `5f6bf07`: clean.

**At the end:**
- Both libraries also show ` M 06_docs/gate-runs.md`. The rows are 04:00:22Z (go-ionomaps quick) and 04:10:14Z (go-tuiMaps quick). They are another reviewer's runs, not mine: my gate rows went to my clones, at 04:03:22Z and 04:13:14Z.
- I edited no tree and changed no git state.

Every finding was reproduced against those SHAs.

My scratch is `<scratch>/rt7-r7l/`:
- `iono/` and `wp/` are rsync copies, used for probes and mutants. Each file was restored and checked with `cmp`. The probe tests are kept in `probes/`.
- `ionoc/` and `tuic/` are clones at the reviewed SHAs, with the gitignored P10 ledger copied in, used for the gates.

**What I ran:**
- **watchpost (tree):**
  - `make lint`: no new findings, 6 baselined.
  - `lint-authoring`: OK.
  - `p10`: 0 live.
  - `dupes`: 2 groups, both ratified.
  - `mutant-anchors`: 826, all match.
  - `go vet` is clean on `domains/propagation`, `app` and `modes/tty`.
  - The targeted tests pass under `-race` on arm64, and pass under `GOARCH=amd64`.
- **Library quick gates (clones):** `scripts/gate --quick` is green for both.
- **go-ionomaps, repeated runs (clone):**
  - The four timing tests, 30 times under `-race` on arm64: green.
  - The same four, 15 times on amd64: green.
  - The whole suite on amd64: green.
- **Pins:**
  - `go.sum:13-16` name rc.12 and rc.10.
  - The module cache matches the rc.12 clone (`diff -rq`, code only).
  - `a23484f` is still not in the local repository.

### Part 1: round 6's findings

| Finding | Verdict | Evidence checked in code |
|---|---|---|
| R6L-1 | **FIXED** | go-ionomaps `asker.go:286-299`: a cancel sets no `NotReached`, and the counts are zeroed at grant. My mutant MA (drop `!cancelled`) is killed by `TestOnlyTheUpdatesEndCountsStationsNotReached/a_cancel`. |
| R6L-2 | **FIXED** | `update.go` `MinDeadline = 32 s`. `deadline.Room` (`internal/deadline/deadline.go:57-70`) asks at exactly that deadline. The tests at `update_test.go:603-609` and watchpost `propagation_test.go:255-262` pass on arm64 and amd64. |
| R6L-3 | **FIXED** | `asker_test.go:323-331`: `answerIn` honours its context. The 30 runs under race and the 15 on amd64 are all green. |
| R6L-4 | **FIXED** | Mc is now MA/MB, both killed. Md (m103) is killed. Mw, run on a copy of watchpost `map_prop_update.go:164`, is killed by `TestACutShortBurstIsSaid`. A-95 (1)'s AfterFunc removed is killed by `TestTheAppsEndStopsAnUpdateAndKeepsItsState`. |
| R6L-5 | **FIXED** | `asker.go:292-293` deletes the probe from the count. m099 exists; the count matched in my sweeps. |
| R6L-6 | **FIXED** | `asker.go:183`. My M6 is killed. |
| R6L-7 | **FIXED** | The sentence is gone from watchpost `map_prop_update.go:367-369`. |
| R6L-8 | **FIXED** | go-tuiMaps `race_detector_test.go`. It runs (it does not skip) both with and without `-race`, and m094 is registered. |
| R5L-5's record line | **FIXED** | `red-team-review.md:548`. |

### 1. Happy Path Bias

**R7L-1. D-307 groups the hanging stations together, and D-302's "not reached first" then moves them to the head as a block. Three hanging stations then call GIRO down every other hour while about 36 stations answer. With two, D-307's promised result ("the healthy ones are asked first every hour and all read") fails at about 1 s an ask.**
- **Severity: Important.** It is the R5L-1 class (a false "GIRO unavailable"; PS-2 harm), and round 6's remediation introduced it.
- **Simplify/Delete?** N.
- **The mechanism:**
  - `RanOut` puts a station that ran out into `l.last` (go-ionomaps `asker.go:221`), and `rotation` puts those stations together in the tail (`stations.go:253`).
  - When the update's deadline cuts that tail, `endEarly` → `Unasked(rest)` (`asker.go:299`) puts them in `leftOut`.
  - `rotation` places `leftOut` first, ahead of the `last` check (`stations.go:243`). So the hanging stations open the next burst, side by side.
  - Three of them side by side are three full timeouts in a row, and that stops the burst as GIRO down (`asker.go:325`).
- **Probes:** the real ratio scaled 1/100 (600 ms update, 150 ms ask, 10 ms margin), using the suite's own `burstUnder` and `answerIn`. The hanging stations were spread apart in hour 0 (asks 2, 8 and 14). I reproduced it on arm64 and on amd64 (Rosetta).
  - **Healthy stations at 11 ms (about 1.1 s an ask): GIRO called down in 5 of 12 hours.** In those hours the asks are `[0 1 2]`, all three hang, and the result is `Unavailable`, paused. watchpost shows "GIRO unavailable this hour" (`map_prop_update.go:297`), and no station is read that hour.
  - **Control:** with D-307's tail switched off on a copy (`stations.go:253` → `false &&`), GIRO is called down in 0 of 12 hours. So D-307 brought this in.
  - **Two hanging stations at 10 to 12 ms:** on alternate hours both open the burst (`[0 1]`), 30 s are spent, and 13 to 17 of 39 stations go unread. This is exactly the harm D-307 was ruled to remove.
  - **Three hanging stations at 9 ms (PLAN's rate):** 11 to 12 of 39 unread on alternate hours.
- **The docs say otherwise:** go-ionomaps R-5.3 says "a station whose full ask ran out is asked last in the bursts after, until it answers", and D-307 says the healthy stations are "asked first every hour and all read". In the code, a station already in `last` that the deadline cut goes first.
- **Why no test caught it:** `TestAStationThatRanOutIsAskedLast` runs with no deadline (`asker_test.go:705`, `context.Background()`), so the cut path never meets D-307.
- **Action (the choice is for the HUM LEAD, because D-302 and D-307 conflict):**
  - Either a station in `last` stays in the tail when it is not reached, or `Unasked` passes over codes in `last`.
  - Either way, add a multi-hour test under the scaled deadline with 2 and 3 hanging stations and healthy answers at 9 to 15 ms. It should assert that GIRO is never called down and that no hanging station opens a burst.

Otherwise the unhappy paths of D-301 to D-306 are built and tested: cancel, 429, no room, a burst that asked nothing, and a probe not reached.

### 2. Systemic Failure Modes

- R7L-1 is the systemic one: three stations' hangs are reported as GIRO's own failure, and GIRO is backed off.
- Otherwise NOAA and GIRO still take separate 60 s updates. A slow network is said truly as "cut short", and the 32 s least deadline leaves room for two full GIRO asks, so D-296 holds there.
- Nothing else fails together without being said.

### 3. Gate Validity

**R7L-2. Five rules round 6 introduced pass the suite with their code removed.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **The survivors:** my mutants, run on a copy, with only the four git-listing tests and the mutant-registry test skipped (all five need `.git`):
  - **MC:** `Plan`'s "kept probe is not live" guard dropped (`stations.go:178`). A kept probe that is now live is asked twice.
  - **MD:** `Apply`'s rule that drops an unknown `Keep` removed (`stations.go:411`). A retired code kept in the state file is asked of GIRO.
  - **ME:** the `len(s.Last)` cap removed (`stations.go:385`).
  - **MG:** `Truncate` changed back to `Round` (`update.go:228`). A-35 (2), R6L-2's "it left 19.999s / needs 20s", is untested.
  - **MI:** `RanOut` also on `failed` (`asker.go:221`). D-307's "ran out" only is untested.
- **Killed:** MA, MB, Md, M6, MF (the margin) and MH (no room on a cancel).
- **Plus R7L-1's gap:** the test that holds D-307 runs with no deadline.
- **Action:** add a test that kills each of the five, and the deadline-plus-D-307 test from R7L-1.

### 4. Adversarial Input

- **Hostile state file:** an unknown `Keep` or `Last` code is dropped (`stations.go:409-413`), and a live `Keep` falls back to the next probe. Both are safe but untested (R7L-2, MC/MD).
- **Path inputs on amd64:** a sweep of 6,440 paths × 3 snapshot times. It covered same-point, antipodal, near-antipodal (±1e-7°) and polar pairs, longitudes wrapped to ±540, and 3,000 random near-antipodal pairs, all through `Snapshot.Path` under `GOARCH=amd64`. Result: 0 non-finite answers, 0 panics, 150 refused.
- **The fuzz target only ever varies Path's destination:** its origin is fixed at (40, -100) (`answers_field_test.go:111`). My sweep covers origins, so no finding, but the fuzz target would not have found an origin-side NaN. That is a recommendation, not a defect.
- **Other float-to-int conversions** that could see NaN on amd64 (`limits/auroral.go:126,131`, `lower.go:171`) are guarded upstream (`auroral.go:116`) or only compared, never indexed.

### 5. Graceful Degradation

- **R7L-1's words are untrue:** "GIRO unavailable this hour" while about 36 stations answer.
- **Otherwise true:**
  - A cancel now says nothing about a cut (R6L-1).
  - A burst with no room keeps the last burst's counts and says Unfinished with `ErrNoRoom`.
  - The cut-short words count live stations only.

### 6. Test Confidence vs. Test Results

**R7L-3. D-304's test describes a run it does not make, and accepts either file being named.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:**
  - `update_test.go:685-687` says "with room for one ask … GloTEC's grid is named, the young D-RAP and scales are not". The setup gives room for two asks (`:626`, 290 ms).
  - Instrumented on a copy, the run reads the grid and names `[D-RAP]`, three times out of three.
  - The assertion at `:702` (`!Contains(GloTEC) && !Contains(DRAP)`) passes on either.
  - The "a young held file stands in" rule is proven only by the scales file.
- **Action:** fix the comment to match the run, and assert the exact `Unread`.

R7L-1 is the larger case under this question: D-307's test proves its own fixture (no deadline), not the behaviour under a real update.

### 7. Dependency Fragility

- **Pins and cache:** rc.12 and rc.10 are pinned and match the cache. A-36's sun-track guard holds on amd64 (sweep above).
- **`a23484f` (v0.2.2):** still unverifiable here. go-tuiMaps `main` cannot build from a clean module cache until the HUM LEAD's v0.2.2 tags exist.
- **The `Fetcher` doc** now states the context contract (`ionomaps.go:20-22`).
- **GIRO's per-ask latency is now a fragility:** R7L-1 turns on at about 1.1 s an ask, against PLAN's measured 0.92 s.

### 8. Problem statement present, real, honored

- **Present:** PS-1 and PS-2 (`problem-statement.md:14-31`).
- **Not yet measured:** PS-1 is still a hypothesis (D-217).
- **Measured and short:** M1 and M4 are graded short of target (53.7 % vs 57.4 %, 68.4 % vs 69.4 %), with no floor riding on either. They go to VALIDATE.
- **Harm to PS-2:** R7L-1. An Observer is told GIRO is down, and loses the stations' readings, every other hour under three hung stations and a moderately slow GIRO. It is the same harm R5L-1 closed.

### Verified clean

- **Round 6:** all R6L findings, checked in code and by mutant.
- **D-301:** a deadline of `MinDeadline` asks, and NOAA's room rule is all-or-none. MF is killed.
- **D-302:** a cancel or a 429 counts none, the interrupted station goes first, and a burst that asked nothing keeps the counts.
- **D-303:** the probe is kept and asked first, and not counted.
- **D-306:** `internal/deadline` is shared by NOAA and GIRO.
- **A-95 (1):** the app's context stops an update in flight.
- **A-36 on amd64:** confirmed by the path sweep.
- **Gates:** both library quick gates, watchpost's five legs, and vet, race and amd64 on the touched packages.

### Opinion

**Do not exit REVIEW.**
- R7L-1 is Important. It reopens the R5L-1 class (GIRO falsely called down) through round 6's own D-307, at a GIRO only about 20 % slower than PLAN measured.
- The fix is small: one ordering rule plus a multi-hour deadline test. R7L-2 and R7L-3 can go in the same batch.

**Fix first: R7L-1.**

---

## Code Quality, round 7 (R7C): 0.19.0 REVIEW exit red team (D-308)

**Verdict: do not exit REVIEW yet.** I found one Important and six Minors. Every round-6 Code finding is fixed in the code. The Important is in the D-307 remediation itself. A station whose full ask ran out is supposed to go last. But when the update's deadline cuts the burst short, the code puts that station first instead. So the case D-307 was ruled for still fails: at a plausible GIRO pace, the hanging stations are asked first every other hour (R7C-1).

### Tree state and runs
- **`git status --short` at the start:**
  - watchpost `b16e5411`: ` M .../02-analysis/rulings.md` (not mine).
  - go-ionomaps `68f31f3`: clean.
  - go-tuiMaps `5f6bf07`: clean.
- **At the end:** each library also shows ` M 06_docs/gate-runs.md`. Those rows are not mine (04:00:22Z and 04:10:14Z). I ran both gates on byte copies under `<scratch>/rt7-r7c/copies`, and my rows there are at 04:02:14Z and 04:11:36Z.
- **I edited no tree.** Every probe and mutant was an `-overlay` from scratch. All findings were reproduced against the three tips above.
- **Gates:**
  - go-ionomaps `scripts/gate --quick` on the copy: **green**, 306 s (race on 241 s, P10 included).
  - go-tuiMaps `scripts/gate --quick` on the copy: **green**, 561 s.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes`, `mutant-anchors` (826), `go vet ./...`.
  - `go test -race` on `domains/propagation`, `app` (filtered) and `modes/tty` (filtered). The same tests pass under `GOARCH=amd64`.
- **amd64 (Rosetta):**
  - go-ionomaps root, `internal/giro` and `internal/deadline` with `-count=3`: green.
  - The R6C-8 subtest run 10 times: green.
  - The fuzz seeds, `TestAPathOfNoLengthIsFinite`, `internal/limits` and `internal/stations`: green.
  - go-tuiMaps `TestRaceDetectorIsToldTruly`: passes.
- **Mutants (my overlays):**
  - Killed: watchpost m575 and m576; go-tuiMaps `raceDetector` flipped (fails).
  - Survived the whole go-ionomaps module: yC, yN, yM1, yM2, yE2 (R7C-2).

### Part 1: round 6's findings, checked in the code
| ID | Status | Evidence |
|---|---|---|
| R6C-1 | FIXED | `MinDeadline` is 32 s (go-ionomaps `update.go:329`). An update at exactly the minimum asks (`update_test.go:603-609`). watchpost refuses only below it (`propagation.go:86`) and asks at it (`propagation_test.go:255-261`). The error now truncates (`update.go:228`). New residue: R7C-7. |
| R6C-2 | FIXED | One rule, `deadline.Room` (`internal/deadline/deadline.go:57-70`), used by NOAA (`update.go:353`) and GIRO (`asker.go:211`). I re-ran round 6's probe with the margin scaled to 1 ms (100 ms ask, 101.3 ms deadline, solar stalled): 0 of 40 updates hit `DeadlineExceeded` on arm64, and 0 of 40 on amd64. |
| R6C-3 | FIXED (D-304) | The docs are reworded, and the test holds a young scales file (`update_test.go:705`), which pins the lacks rule. The test's own doc is false: R7C-4. |
| R6C-4 | FIXED (D-302) | A cancel now goes through `endEarly` (`asker.go:217`), and `stops` has no cancel case (`:319`). The count is taken at the update's end only (`:292-294`). The doc is fixed (`snapshot.go:144`). Old survivor yd: the code is gone. |
| R6C-5 | FIXED | `snapshotWords`' doc (`map_prop_update.go:367-369`). |
| R6C-6 | FIXED | The redundant branch is gone (`update.go:226-232`). |
| R6C-7 | FIXED | `race_detector_test.go`. My flip overlay fails the test. |
| R6C-8 | FIXED | `asker_test.go:428` now asserts `a.Failed != 1`. Under heavy load the subtest can now flake (20 ms of scaled slack) rather than pass vacuously. It ran 10 times green on amd64. |
| Q3 note | RULED (D-306) | `internal/deadline` exists, with its ledger row. |
| Recommendation | RULED (D-307) | Built, but defeated when a burst is cut short: R7C-1. |
| R5C-4 and R5C-5 residue | FIXED | The room rule is written once, and the error truncates. |

### 1. Understandability
- **R7C-1 (Important), go-ionomaps: D-307's "ran out → last" loses to "not reached → first" whenever the deadline cuts the burst.**
  - **The mechanism:**
    - `rotation` puts `leftOut` first and never checks `l.last` (`internal/stations/stations.go:243-246`).
    - A station that ran out sits at the tail, so it is the one a cut-short burst leaves. `endEarly` hands it to `Unasked` (`asker.go:299`), which puts it first next hour.
    - R-5.3 states both rules and does not say which wins.
  - **Probe:** two adjacent stations that never answer, the others answering at a scaled pace, a 600 ms update (60 s scaled), 8 hours. Reproduced on arm64 and amd64.
    - At 0.9 s per answer (PLAN's pace), one hanging station is asked **first every hour**.
    - At 1.2 s per answer, **both are asked first every other hour**, and 16 to 17 of the 37 healthy stations go unread in those hours.
    - D-307 promises "the healthy ones are asked first every hour and all read".
  - **The fix is one line:** add `&& !slices.Contains(l.last, s)` to the leftOut loop.
    - With it, the hanging stations sit at 37-39 every hour, and at 1.2 s only 2 stations go unread each hour.
    - Every existing go-ionomaps test passes both with and without the line, so the interaction is unpinned.
  - **Why the test missed it:** `TestAStationThatRanOutIsAskedLast` (`asker_test.go:696`) runs with `context.Background()`, so no deadline ever cuts its burst.
  - Simplify/Delete: N.
  - Action: the HUM LEAD rules which rule wins when they collide. I recommend "ran out wins", since it is what D-307 asked for. Then add a deadline-bound test with two hanging stations.

### 2. Maintainability
- **R7C-2 (Minor), go-ionomaps: keeping `Keep` and `Last` across restarts is unpinned.**
  - A-35 (5) says both survive a restart, and that a code the seed no longer names is dropped.
  - These mutants each survive the whole module:
    - `Apply` ignores `Keep` (yM1).
    - `Apply` ignores `Last` (yM2).
    - `Record` drops `Last` (yE2).
    - An unknown `Keep` is not dropped (yC).
    - The length check on `Last` is removed (yN).
  - Code: `stations.go:366-367` (`Record`) and `:377` onward (`Apply`). No watchpost test reads either field.
  - Simplify/Delete: N.
  - Action: a `Record`→`Apply` round-trip test with a kept probe, a `Last`, and a retired code.

### 3. Complexity
Nothing is over-complex. Two notes:
- **Shaped by a metric (noted, not numbered):** the build log says the rc.12 clamp was a helper of its own until `internal/limits`' density floor failed it (1.22 against 1.25). So the clamp is now written twice on one line (`internal/limits/budget.go:86`). This is the same thing as round 6's question 3. Recommendation for the HUM LEAD: let a ratified row allow the helper.
- The `switch` in `rotation` has an empty `case slices.Contains(out, c):`, which skips silently. It reads better with a comment.

### 4. Necessity
- Nothing is dead. `RanOut`'s empty-code guard and `Plan`'s check that a kept probe is still dark are defensive, and cost little.

### 5. Name semanticism
- `endOf` (`asker.go:246`) does not say what it holds: "what an early end puts back". A name like `restore` would say it. Nit, not numbered.

### 6. Code documentation
- **R7C-5 (Minor), go-ionomaps: two stories about when a station leaves `Last`.**
  - These say it is asked last "until it answers": R-5.3, `asker.go:177`, and `stations.go:59`.
  - These say "until an ask of it ends any other way": `RanOut`'s doc (`stations.go:188`) and A-35 (5). That is what the code does: a 5xx, 404 or 429 also releases it.
  - Simplify/Delete: Y. Action: make them say one thing.
- No history comments (AP-HIST-01) in the range.

### 7. Testability
- **R7C-3 (Minor), go-ionomaps: the rc.12 guard checks the inputs, not the value it converts.**
  - `sunAt` refuses a NaN place (`answers.go:408`), but an infinite one still reaches `int(NaN)` at `:424`.
  - **Probe:** `sunAt(+Inf, 0)` and `sunAt(0, +Inf)` panic on amd64 with "index out of range [-9223372036854775806]". On arm64 they return 24.
  - This is not reachable through the public API: `checkPlaces` bounds every place, and `Along` returns a finite place or NaN. But A-36 (2)'s "takes no step whatever the platform" holds only for NaN.
  - **The other float→int sites, checked:** `background.regrid` has fixed bounds, `magcoords` and `gyro` refuse NaN, `AuroralLoss` refuses NaN, and `lower.go:171` only compares the result. None is open.
  - Simplify/Delete: Y. Action: guard the stepped value itself (e.g. `if !(c <= 0)`), which covers every non-finite input in one place.
- R7C-2 also belongs here.

### 8. P10 conformance
- Both quick-gate P10 legs are green, and watchpost's `make p10` is 0/0/0.
- The `internal/deadline` row is RATIFIED (D-306) at its measured 1.33. No exemption was self-issued.

### 9. Evidence soundness
- **R7C-4 (Minor), go-ionomaps: `TestARoundWithNoRoomNamesWhatItLacks`' doc describes a run that does not happen.**
  - Its doc (`update_test.go:681-687`) says the round has room for one ask, reads the solar index, names the grid, and does not name D-RAP.
  - My probe, 5 of 5 runs: solar is not asked, GloTEC's index and grid are read, and `Unread` is `[D-RAP]`.
  - The assertion accepts either GloTEC or D-RAP (`:702`), so it does not pin which.
  - Simplify/Delete: N. Action: rewrite the doc, and assert exactly D-RAP.
- **R7C-6 (Minor), watchpost: the app's-end test claims GIRO's burst but runs a NOAA ask.**
  - `TestTheAppsEndStopsAnUpdateAndKeepsItsState` (`app/mapprop_test.go:202`), m575's description and batch 70's log all say "a quit during GIRO's burst … a restart within the hour asks GIRO nothing new".
  - The ask in flight is NOAA's first round. No burst is ever made. The cancel mechanism is the same, but the claim is not what the test proves.
  - Action: reword, or drive a second update into the burst.
- **R7C-7 (Minor), go-ionomaps: the reason for 32 s is pinned only by the number.**
  - D-301 picked 32 s so that D-296's call-down holds at `MinDeadline`.
  - The only pin is the literal `MinDeadline != 32*time.Second` (`update_test.go:600`). No test hangs GIRO at the least deadline.
  - Action: a scaled giro test that GIRO is called down at 2×(askTimeout+Margin).

### 10. Evasion lens
- R7C-1: D-307's test passes only because nothing in it cuts the burst. The record closes D-307 while the case it was ruled for still fails.
- R7C-2: A-35 (5)'s restart claim has no test that can fail.

### What I verified clean
- **go-ionomaps:**
  - `deadline.Room`'s regimes.
  - The NOAA window is closed (arm64 and amd64).
  - The cancel path (D-302): counts restored when nothing was asked, and `NoRoom` reset at the top of each burst.
  - A probe not reached is asked first (D-303).
  - `tally`'s run is broken by any other result (D-305).
  - The rc.12 clamp, and the NaN guard for a NaN place.
  - `roomless` with no snapshot made.
- **watchpost:**
  - The app's cancel comes before `settle` (`dashboard.go:132`, then `stopAll` at `:1016`).
  - The `AfterFunc` tie is deregistered when the update ends.
  - m575 and m576 are killed.
  - The deadline test reads the service's own deadline.
- **go-tuiMaps:** the race flag is checked against the build, and the gate's leg label is updated.

### Opinion
**Do not exit REVIEW** until R7C-1 is ruled and fixed. It is a one-line change plus one deadline-bound test, and the HUM LEAD has to rule which rule wins when the two collide. The Minors can be batched with it.

**Fix first: R7C-1.**

---

## Timing lens, round 7 (R7T): 0.19.0 REVIEW exit red team (D-308)

**Tree state.**

| Repository | At the start | At the end |
|---|---|---|
| watchpost | `b16e5411`, ` M …/02-analysis/rulings.md` (not mine) | `9a45334c` (D-308's commit, docs only: the code is `b16e5411`'s), clean |
| go-ionomaps | `68f31f3`, clean | ` M 06_docs/gate-runs.md`, a quick-gate row at 04:00:22Z (not mine) |
| go-tuimaps | `5f6bf07`, clean | ` M 06_docs/gate-runs.md` (not mine) |

- I edited no tree. My gates ran in rsync copies under `rt7-r7t/copies`. Probes and mutants went in by `-overlay` from `<scratch>/rt7-r7t`.
- Every finding was reproduced against the three start tips, on arm64 and under `GOARCH=amd64` (Rosetta).

**What I ran:**
- **watchpost:** `make lint` (6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes` and `mutant-anchors` (826 match) all exit 0. `go vet` on propagation, tty and app is clean. `-race` passes on `domains/propagation`, on `modes/tty -run Prop|Stations|Burst` and on the app's propagation tests, and the same three pass under amd64.
- **Libraries:** go-ionomaps `scripts/gate --quick` is green with P10; go-tuimaps `--quick` is green.
- **go-ionomaps packages:** giro, stations, throttle and deadline pass under `-race`. Those four and the root package pass under amd64 (66 s).
- **Probes:**
  - **Hours of bursts:** a simulator in `internal/giro`, at 1/100 and 1/50 scale, with a 60 s deadline and answers at 0.5, 0.7, 0.92, 1.15, 1.25 and 1.45 s. It hung 0 to 6 stations at chosen positions, with and without a hanging probe, over 8 to 12 hours each: about 40 configurations, more than 400 bursts.
  - **The least deadline:** a GIRO hung on every ask, under exactly 32 s, in real time.
  - **NOAA's tail:** the time from the last NOAA ask to `Update`'s return.
- **Mutants:** 18 in the library, 3 in watchpost.

### Part 1: the timing findings of rounds 3 to 6

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R3L-1 | FIXED | go-ionomaps `update.go:358` (20 s), `:360-364` (a timeout stops the round), `:362` (trouble), `:164` (back-off) |
| R3L-2, R3C-1 | FIXED | `asker.go:325`: three full asks run out in a row stop the burst whatever answered. But R7T-1 now produces such runs from stations that are not GIRO |
| R4L-1, R4C-1 | FIXED | `asker.go:211-215` (room before every ask), `:340`, `:347-348` (a shortened ask's run-out is not reached). Every simulated burst ended before its deadline, on both platforms |
| R4L-3 | FIXED | `deadline.go:58-61`; `asker.go:346`; `update.go:360` |
| R5L-1 | FIXED | `asker.go:33,295`. One hanging station, at any position, is never called down |
| R5C-1 | FIXED | `deadline.go:46-49` gives the shortened asks; the count is at `asker.go:293`; the words at `map_prop_update.go:164,412-414` |
| R5L-2 | FIXED | `update.go:353-355` (no room sets no trouble), `:161-163`; `asker.go:288-291` |
| R5L-4 | FIXED | Narrowed by D-302: `asker.go:292-294` |
| R5D-1 | FIXED | `update.go:99-111,324-329` |
| R6T-1 | FIXED, with a test gap (R7T-3) | watchpost `app/mapprop.go:121-124` (`AfterFunc`); `app/dashboard.go:132` cancels before `stopAll`; `:1016` settles. Mutant W1 (AfterFunc removed) is killed |
| R6T-2 | FIXED | `update.go:329` is 32 s. A real-time probe at exactly 32 s made 2 full asks and called GIRO down inside the update, on arm64; amd64 gives the same at 1/10 scale. watchpost accepts 32 s (`propagation.go:86`) |
| R6T-3 | FIXED | `stations.go:176-181`; `asker.go:199-200,293`. Mutants A2, A4 and A19 are killed |
| R6T-4 | FIXED | `asker.go:217-218,287,292`. Mutants A3, A5, A18 and G2 are killed |
| R6T-5 | FIXED | `deadline.go:42`; `update.go:353`; the Fetcher doc at `ionomaps.go:20`. Mutants A8 and A10 are killed |
| R6T-6 | FIXED | g2 and g14 (my G2 and G14) are killed; g13 is killed through A8 ("just under a full ask's room", `asker_test.go:467-478`); w3 (my W3) is killed |
| R6T-7 | RULED | D-305; `asker.go:262-268`, words at `:29-33` |
| R6D-1 | FIXED | `update.go:329`; the root test asks at exactly `MinDeadline`; watchpost's test asks at the least deadline |

### 1. Bounded

Yes. Every burst I simulated ended before its deadline: every hang position, 0 to 6 hanging stations, answers from 0.5 to 1.45 s, both platforms. The others:
- **At exactly 32 s in real time:** the hung GIRO was called down with time to spare.
- **NOAA's tail:** 35 to 38 ms after the last ask (parse and build, with a 577 KB synthetic grid), well inside the 1 s margin.
- **watchpost's client:** it times out at 30 s (`platform/httpx/plain.go:79-80`), after the library's 15 s and 20 s, so the library's limits rule.

### 2. Truthful outcomes

Exceptions: R7T-1 (GIRO called down while answering, every other hour) and R7T-2.

**R7T-1: D-302's "not reached, first" overrides D-307's "ran out, last". The result is that stations already known to hang are asked first, in a row, at the full 15 s. GIRO is called down while it answers, and half the network goes unread.**

**Mechanism:**
- go-ionomaps `internal/stations/stations.go:243-247`: `rotation` puts every `leftOut` station first, before the `l.last` check at `:253`.
- `Unasked` (`:213-232`), called from `endEarly` (`asker.go:299`), puts the stations not reached into `leftOut`, the ones that ran out included.
- `RanOut` runs only for asks that ended (`asker.go:221`).
- So under D-307 the hanging stations collect in the tail. The deadline then cuts after the first of them, and the rest are asked first next hour, one after another, at full asks. Three of them in a row trip D-286 (`asker.go:325`).

**Probes** (1/50 scale, answers at 0.92 s as in PLAN, a 60 s deadline; arm64 and amd64 agree):

| Hanging stations (live positions) | What happens |
|---|---|
| 4: 1, 2, 3, 4 | A steady two-hour cycle. In every other hour the burst asks three stations, all hanging, and stops: GIRO Unavailable, 0 of 35 healthy stations read. Called down 5 of 10 hours (arm64), 4 of 8 (amd64) |
| 5: 5, 15, 25, 35, 36 | Called down 5 of 8 hours, with 6 of 34 healthy read in those hours |
| 5: 0, 10, 20, 30, 38 | Never called down, but only 16 to 18 of 34 healthy read every hour |
| 4: 5, 15, 25, 35 | 26 to 28 of 35 healthy read every hour; with a hanging probe, 12 of 35 in alternate hours |
| 2: 10, 20, answers at 1.15 to 1.25 s | 19 to 21 of 37 healthy read in alternate hours |

**What the listener is told:** "GIRO unavailable this hour" (`modes/tty/map_prop_update.go:298`) every other hour, although GIRO answered every healthy station the hour before. That breaks D-247 and D-286's premise that GIRO is called down only when it stopped answering, and D-307's ruled promise: "the healthy ones are asked first every hour and all read". It lasts while the hanging stations stay live, up to 3 days (`darkAfter`, `stations.go:28`).

**It is unpinned in both directions:**
- The one-line change at `stations.go:244` (`&& !slices.Contains(l.last, s)`) passes the whole stations, giro and root suites (mutant G21 survives).
- With that change, every case above reads 34 to 37 healthy stations every hour, and no hour is called down after hour 0 (whose call-down is D-286's three hanging in a row, as ruled).
- `TestAStationThatRanOutIsAskedLast` (`asker_test.go:696`) runs with no deadline, so no station is ever "not reached".

**The rulings disagree for one station:** a station that ran out and was then not reached is "asked first next hour" under D-302 (D-110), and is also counted in the "N stations asked first next hour" words. It is "last … until it answers" under D-307 (D-115), and under `RanOut`'s doc (`stations.go:186-189`). The code chose D-302 silently.

- **Severity:** Important. It is a false GIRO outage said to the listener, deterministic, at PLAN's ratio, for days. It needs 4 or more hanging stations out of about 39, or fewer stations with a slower GIRO; how often stations hang is not measured.
- **Simplify/Delete?** N.
- **Action:** a fork for the HUM LEAD:
  - (a) D-307 wins: a station in `last` stays in the tail even when not reached, and like D-303's probe it is not counted among the stations "asked first next hour";
  - (b) D-302 wins, but a station already in `last` does not count toward D-286's or D-296's runs.

  Either way, add a test with a deadline, 4 hanging stations and 8 or more hours, that asserts every healthy station is read and GIRO is never called down.

**R7T-2 (a recommendation against D-305 and D-307): D-307's tail makes two hanging stations adjacent by construction, so D-296's call-down at the update's end fires against a healthy GIRO whenever GIRO answers fast.**

**Mechanism:**
- When GIRO answers in 0.7 s or less, both hanging stations get full asks at the end of the burst.
- The probe, which is the last ask, is then not reached or finds no room.
- `endEarly` sees two full asks run out one after another and calls GIRO down (`asker.go:295-298`).

**Probes:**
- Two hanging stations at 0.7 s an ask: called down 3 of 8 hours, the same with or without R7T-1's change.
- Three hanging stations at 0.5 s, with that change: 6 of 8 hours.

D-305 accepted "two adjacent hanging stations calling GIRO down at the end" as a known limit, meaning an occasional one. D-307, ruled the same day, makes the adjacency certain, and neither counter-argument names this.

- **Severity:** Minor. Simplify/Delete? N.
- **Action, for the HUM LEAD:** count only stations not already in `last` in the end-of-update run (this goes naturally with R7T-1 (b)).

### 3. Edges

- **16 s and 21 s left:** they hold. `deadline.go:42-49` gives a full ask when `left − 1 s ≥ need` (16 s for GIRO, 21 s for NOAA). Mutants A8 and A9 are killed.
- **The 32 s least deadline:** it holds, by the real-time probe. A deadline of exactly `MinDeadline` asks.
- **The floor:** 2 s holds; below it the burst ends (A9 is killed).
- **A shortened ask that answers:** the burst goes on and asks again. The simulator shows it at 9 to 10 s left.
- **A timeout beside a cancel:** G2 and G14 are killed.
- **Two timeouts with a 5xx between:** RULED (D-305).
- **A path of no length:** A-36's fuzz seed passes under amd64 with the root suite.
- **A transport timeout inside a shortened ask:** watchpost's dial and TLS timeouts are 10 s (`platform/httpx/httpx.go:250-258`). One that fires inside a shortened ask longer than 10 s ends the burst as "not reached" with time still left. That is negligible; I note it and do not raise it as a finding.

### 4. Across hours

- **No station starves:** `leftOut` rotates the stations that were cut.
- **No hour is spent with nothing asked:** A1, A13, A16 and A20 are killed.
- **GIRO's back-off does not grow without cause:** it doubles only across bursts with no answer, and R7T-1's healthy hours reset it (`asker.go:356`). The pause is shorter than the hourly cadence in any case.
- **The exception is R7T-1:** a third to a half of the network goes unread every hour for days, or every other hour is called down.

### 5. Tests

| Mutant | Result |
|---|---|
| A1 to A5, A8 to A10, A13, A16, A18 to A21, G2, G14 (library) | killed |
| W1 (the seam's `AfterFunc` removed), W3 (the GIRO-answered guard dropped) (watchpost) | killed |
| A6 (the kept probe's live check dropped) | survives; equivalent, since a probe cannot become live without being asked |
| G21 | survives: R7T-1 |
| W4 | survives: R7T-3 |

The simulator's ratios are realistic, with the margin, the floor and the full ask all scaled together.

**R7T-3: the value of `settleBound`, the exit's wait, is unpinned, and the R6T-1 test holds no burst in flight.**
- `app/mapprop.go:59`. Mutant W4 (`settleBound` set to 2 ms) passes the whole app package.
- `TestTheAppsEndStopsAnUpdateAndKeepsItsState` cancels NOAA's first ask with no field held, so the update's end builds nothing.
- A real quit during a burst builds the field before it keeps the state: 56 ms, or 389 ms under race, by round 6's measure. R6T-1's action asked for "a quit with a burst in flight".
- Severity: Minor. Simplify/Delete? N.
- **Action:** cancel during a burst with a field held. Assert that the kept state records the burst's hour inside `settleBound`.

### 6. Host and library

These agree:
- **Deadline:** each `Ask` gets its own 60 s (`domains/propagation/propagation.go:224`), and anything under 32 s is refused (`:86`).
- **Wait at exit:** the app's cancel reaches the update through `AfterFunc` before `settle` (`dashboard.go:132,1016`).
- **Refresh tick:** set after each result, never before NOAA's `notBefore`.
- **The burst after StationsNext:** asked at once, with a fresh deadline (`map_prop_update.go:181-187`).
- **Words:** they match the library's counts, except "GIRO unavailable this hour" under R7T-1.

### Verified clean

- **Bounded:** every update ended inside its deadline at every timing I built, on arm64 and amd64.
- **The least deadline:** at exactly 32 s, a hung GIRO is called down in real time.
- **NOAA:** its margin, its no-room handling and its tail after the last ask all hold.
- **D-302:** a cancel or a refusal counts none, and the interrupted station goes first.
- **D-303:** a probe not reached is kept, asked first and not counted.
- **The quit:** watchpost's quit stops the update.
- **Gates:** every leg named in the brief is green.

### Opinion

**Do not exit REVIEW.** The library's bounds hold everywhere. But round 6's two ordering rulings, D-302 and D-307, disagree for a station that ran out and was then not reached. The code's silent choice manufactures runs of known-hanging stations. GIRO is then called down while answering, and is said to be "unavailable this hour" every other hour for days. The fix is one line once the HUM LEAD picks a side.

**Fix first: R7T-1.**
