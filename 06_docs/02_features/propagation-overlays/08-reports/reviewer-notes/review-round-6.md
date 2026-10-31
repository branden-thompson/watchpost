---
title: "0.19.0 REVIEW exit red team, round 6 - the reviewers' reports, verbatim"
date: 2026-10-26
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - four fresh blind reviewers (D-300), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 6."
---

# Round 6, the reports

Up: [red-team review](../red-team-review.md) · [round 5](review-round-5.md)


---

## Docs Quality, round 6 (R6D): 0.19.0 REVIEW exit red team (D-300)

**Tree state.** I reproduced every finding against these states.

| Repository | At the start | At the end |
|---|---|---|
| watchpost `2ec275b5` | ` M …/02-analysis/rulings.md` (the uncommitted D-300 row, not mine) | unchanged |
| go-ionomaps `d5d4d6c` | clean | ` M 06_docs/gate-runs.md`: my FAILED quick row. I left it in place, because the gate record keeps its failures. |
| go-tuimaps `813b7b0` | clean | ` M 06_docs/gate-runs.md`: my green quick row |

**What I ran:**
- **watchpost:**
  - `make lint` (no new findings, 6 baselined), `make lint-authoring`, `make p10` (0/0/0), `make dupes` and `make mutant-anchors` (824 match). All exited 0.
  - `go vet` on `domains/propagation` and `modes/tty`.
  - `go test -race` on `domains/propagation` (whole) and on `modes/tty -run 'Stations|Burst|Prop|CutShort'`. Both ok.
- **go-tuimaps:** `scripts/gate --quick` was green.
- **go-ionomaps:** `scripts/gate --quick` **FAILED** at 16:34:06Z on tree `1136cfdfc7b5` (R6D-5).
- **Every test named in the requirements exists:** watchpost 155/155 and go-ionomaps 129/129. go-tuimaps has 40 of 42; the two missing are L-5's, and L-5 is ruled out of v0.3.0 (its D-10).
- **Scratch probes** in `<scratch>/rt6-r6d/`:
  - `noroom/` is a host program against the public API;
  - `iono/` is a copy of go-ionomaps with one probe test, `zz_probe_test.go`.

### Part 1: round 5's findings

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R5D-1 | FIXED, with a new error | No back-off when there is no room: `update.go:351-354` (it sets `noRoom` and never `trouble`) and `:149-151`. `MinDeadline` and `ErrNoRoom` are at `:324-332`. The README (`:27-31`, plus a per-update deadline in the example at `:40-44`) and R-5.3 now state it. But the stated minimum itself fails: see R6D-1. |
| R5D-2 | FIXED | `README.md:22-25`: real time, a GIRO timeout is that station's failure, "three in a row, or two as the update's deadline comes". This matches `asker.go` `tally`/`endEarly`. |
| R5D-3 | FIXED | `offset.go:18`, `snapshot.go:226`, `constants.go:38,40` and `ahead.go:64` all say "the week of 2026-09-29 to 10-05". |
| R5D-4 (and R4D-2) | FIXED | `README.md:208,212-213` and `extending.md:24-25` name the chart's esc/PgUp/PgDn and the acknowledgement window. They match `map_prop_chart.go:475,487-489`. |
| R5D-5 | FIXED | `00-REQUIRED-READING.md:36,52`; `red-team-review.md:7`. `go.mod:9-10` pins rc.10 and rc.9. |
| R5D-6 | FIXED | Five rows were added at `architecture.md:170-174`. I checked each row's words against the code: `map_prop_update.go:153,295,297,306-308,406-410`. There is a new error in the table's older row (R6D-4). |
| R5D-7 | FIXED | The must-catch list is at `docids_test.go:34`, with the same pattern in both libraries (diffed). `aliasdoc_test.go:63` fails below 17. Residue: the claim "every family this release's own documents use" (`docids_test.go:15-16`) is still too broad. The record's own mutant names (`Ma`, `Mg`, `x4`; `review-round-5.md:190,438`) slip through. None is in `go doc` today, so this is a nit. |
| R5D-8 | FIXED | `validate-agenda.md:21` carries both blind spots. They match `NOTE-outage.txt` (14:40 to 17:20 UTC, U9-U12). Nit: "U9 to U12" is not explained on the page. |
| R5D nits | FIXED | "past past" is gone. `propagation.go:313` names D-272. `go doc` renders `reach.go`'s path whole. go-tuimaps `doc.go:31-32` defines "upstream". |

### 1. Does each document answer the question its title asks?

**R6D-4: the architecture's "When things fail" table misstates what a 429 shows, and one new row describes the mechanism instead of the screen.**
- `architecture.md:168` says a 429 shows "GIRO paused". The code says "GIRO unavailable this hour: the readings it gave first":
  - a 429 sets `GIROUnavailable` (`snapshot.go:155`);
  - `giroWords` gives "GIRO unavailable this hour" (`map_prop_update.go:297`);
  - "GIRO paused…" is only the fallback when no outcome is set (`:383`).
- The "NOAA stalls" row (`:174`) gives the mechanism ("NOAA backed off") under a column titled "What the listener sees". The listener actually sees "NOAA's GloTEC … not read this update" (`:393`).
- Severity: Minor. Simplify/Delete? N. Action: quote the real words in both rows.

### 2. Does anything contradict the code, or another document?

**R6D-1: `MinDeadline` is documented as enough, but a deadline of exactly `MinDeadline` never updates, and watchpost accepts that value and pins the failure in a test.**
- The docs promise that the minimum works:
  - `update.go:324-326`: "the least time an update's context may leave it … With less, nothing is asked";
  - `README.md:27`: "leave it at least `MinDeadline`".
- The code refuses it. `ask` refuses when `left < noaaAskTimeout`, and `noaaAskTimeout = MinDeadline` (`update.go:340,351`). At the moment of that check, `left` is already below 20 s.
- My probe (`noroom/`, `context.WithTimeout(…, ionomaps.MinDeadline)`) printed: *"it left 20s, and an update needs at least 20s; nothing was asked, and nothing backed off"*. Zero asks, and no field ever.
- watchpost accepts `Deadline == MinDeadline` (`propagation.go:86`, whose comment says "too short for any ask"). Its own test then requires `ErrNoRoom` at that value (`propagation_test.go:241-248`). The build log records this as intended: "reads `ErrNoRoom` at the least deadline".
- D-298 ruled a 20 s minimum that works. Its own error message now contradicts itself.
- Introduced by round 5. No listener is affected, because watchpost always uses 60 s and no setting reaches `Deadline`.
- Severity: **Important**. This is a v0.1.0 public constant whose documented meaning is false at its own value. Simplify/Delete? N.
- Action: two options for the HUM LEAD (D-298 set the number):
  - (a) make `MinDeadline` 20 s plus a margin, and keep "at least";
  - (b) keep 20 s and refuse only `< MinDeadline` with the ask's own timeout capped by the time left.

  Either way, add a test that an update at exactly `MinDeadline` asks NOAA, and fix the wording of `errNoDeadline` and `update.go:225`.

**R6D-2: after a burst that asked GIRO nothing, every later snapshot counts all 40 stations "not reached", and watchpost says the burst was cut short.**
- In `asker.go:197`, a granted burst zeroes its counts. Then `endEarly` sets `NotReached = len(rest)` even when `nothingAsked` (`:274-280`). That count is carried into every snapshot until the next burst (`update.go:549`).
- My probe on the copy:
  - a burst cancelled before its first ask, or one with no room, was followed by an update with `NoStations`;
  - that update returned `Reason Fresh`, `GIRO answered`, `StationsNotReached 40`, with 0 GIRO asks ever made.
- watchpost would then show "no stations by choice: the background alone · GIRO's burst cut short: 40 stations asked first next hour" (`map_prop_update.go:401-413`).
- That contradicts the field's doc ("stations the last burst did not ask", `snapshot.go:144`) and D-297's case: "ends before every station is asked, GIRO still answering".
- In watchpost the path is narrow: leave the mode before the burst's first ask, switch to no stations, then reopen.
- Severity: Minor. Simplify/Delete? N. Action: leave `NotReached` (and the outcome) untouched when nothing was asked, and add a probe-shaped test.

**R6D-3: `snapshotWords`' doc still describes a `burst` parameter that batch 69 removed.**
- `map_prop_update.go:369-370`: "burst is whether the snapshot came from GIRO's burst, whose stopping names GIRO (D-286)". The signature at `:371` has no `burst`, and the D-286 words it describes were retired.
- Introduced by batch 69. `lint-authoring` cannot see it ("a comment that is merely WRONG").
- Severity: Minor. Simplify/Delete? Y (delete the sentence).

**Nits (batch them):**
- The state file's contents disagree across docs:
  - FR-4.12 (`requirements.md:134`) says "the stations' codes and the rotation's place";
  - `README.md:270-271` and `CHANGELOG.md:26-27` say "the stations' codes" only;
  - the state also holds `Out` (`state.go:49`).
- The build log's batch 69 calls this "R5D-4's residue". It is R4D-7's (`review-round-5.md:54`).
- go-tuimaps:
  - the gate's leg label "race detector off (allocation counts live here)" (`scripts/gate:721`) does not say it now also judges NFR-3's peak (D-299);
  - NFR-3's row (`go-tuimaps/01-objectives/requirements.md:66`) does not say the peak is judged with the race detector off.

### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R6D-5: round 5's new timing test fails under load, so rc.10's "gated green" is a verdict that depends on the machine.**
- `TestABurstEndsBeforeTheHostsDeadline/slow_answers,_no_hang` failed in my quick gate: "19 asked, 21 not reached … before the deadline false" (`asker_test.go:432`).
- It also failed 1 time in 20 with `go test -race -count=20` (load average about 6).
- The same tree hash (`1136cfdfc7b5`) was green in the builder's full gate (`gate-runs.md`, 14:31:45Z).
- Cause:
  - the fake `answerIn` sleeps without honouring the ask's context (`asker_test.go:322-324`);
  - so a shortened ask whose limit has only just room for its 30 ms answer can overrun a 10 ms scaled margin (`:317`) once the race detector and load add delay.
- This is the class D-299 ruled for in go-tuiMaps, now in round 5's own new test. It is also the timing lens's ground.
- Severity: Minor (a test defect; the code honours the context). Simplify/Delete? N.
- Action: make the fake honour `ctx`, or widen the scaled margin, then run it repeatedly under race and load before calling it green.

### 4. Is the audience order right: designers, then PMs, then engineers?
Nothing found. The new README paragraphs and the failure table lead with what the host or listener gets. The only exception is the one row in R6D-4.

### 5. Is a number published without its blind spots beside it (INST-5)?
Nothing new. Item 3's bound now carries both caveats. D-299 states its medians with the host load beside them.

### What I verified clean
- R-5.3, D-105 to D-108 and A-34 match `askLimit`, `endEarly`, `tally`, `roomless` and `ask`.
- Watchpost's cut-short words and their once-per-count note (`map_prop_update.go:164-166`).
- The D-286 flag, words and mutants are fully gone: m568 and m572 are deleted, and m573 and m574 exist.
- The anchors match.
- The re-pin to rc.10 and rc.9.
- Both libraries' `go doc` are free of IDs.
- The go-tuiMaps peak is judged with the race detector off (`memory_test.go:166,174`, `race_off_test.go`).
- The required reading, the record's status and the round 5 closure table.

### Opinion
**Do not exit REVIEW until R6D-1 is fixed.** It is a one-line decision plus a test, but it is a public v0.1.0 constant whose documented meaning is false at its own value, and the host pins that failure in a test. R6D-2 to R6D-5 and the nits are Minor and can be batched before VALIDATE.

**Fix first: R6D-1.**

---

## REVIEW lens, round 6 (R6L): 0.19.0 REVIEW exit (D-300)

**Tree state at the start:**
- watchpost `2ec275b5`: ` M .../02-analysis/rulings.md`. The uncommitted row is not mine.
- go-ionomaps `d5d4d6c` and go-tuiMaps `813b7b0`: clean.

**At the end:**
- Each library also shows ` M 06_docs/gate-runs.md`. Those rows are not mine either:
  - go-ionomaps' row is `16:34Z quick FAILED: tests, race detector on`. It may be R6L-3.
  - go-tuiMaps' row is `16:46Z green`.
- My gate rows went to my clones, at 17:01Z and later.
- I edited no tree and changed no git state.

Every finding was reproduced against those SHAs. My scratch is `<scratch>/rt6-r6l/`:
- `iono/` and `wp/` are rsync copies, used for probes and mutants. Each file was restored and checked with `cmp`.
- `ionoc/` and `tuic/` are clones at the reviewed SHAs, used for the gates.

**What I ran:**
- **watchpost (tree):**
  - `make lint`: no new findings, 6 baselined.
  - `lint-authoring`: OK.
  - `p10`: 0 live, 0 unmatched, 0 unratified.
  - `dupes`: 2 groups, both ratified.
  - `mutant-anchors`: 824, all match.
  - `go vet` is clean on `domains/propagation`, `modes/tty` and `app`.
  - `-race` is green on `domains/propagation` and on the Propagation tests in `modes/tty`.
- **go-ionomaps `scripts/gate --quick` (clone):**
  - The first run FAILED. The cause is a clone artifact: the gitignored `.a2dh-p10-exemptions.yml` was missing, and `TestTheP10MirrorMatchesTheLedger` fails by design when it is.
  - With the ledger copied in, the gate is green.
  - `a2dh p10 check` gives 0 live findings.
- **go-tuiMaps `scripts/gate --quick` (clone):** green, P10 run.
- **Mutants:**
  - The registered m083 to m092 are all killed.
  - Of my own 12, 9 were killed and 3 survived (R6L-4).
- **Pins:**
  - watchpost `go.mod:9-10` and `go.sum:13-16` name rc.10 and rc.9.
  - The tags resolve to `d5d4d6c` and `813b7b0`.
  - The module cache matches both trees (`diff -rq`, code only).
- **Not checked:** go-tuiMaps `a23484f`. It is not in the local repository, and I may not fetch.

### Part 1: round 5's findings

| Finding | Verdict | Evidence checked in code |
|---|---|---|
| R5L-1 | **FIXED** | go-ionomaps `internal/giro/asker.go:32,276-279` (`timeoutsAtTheEnd = 2`) and `endEarly`, which names no station to move past (`:281`). My sweep at the real ratio (scaled 1/100) put one hanging station at every position k = 1 to 39, three hours each: 0 bad hours, all 40 asked each hour, Outcome Answered, Failed 1. m087 is killed. |
| R5L-2 | **FIXED, with a boundary defect (R6L-2)** | `update.go:351-354` and `:158-160`: no back-off, `ErrNoRoom` names 20 s. `asker.go:273-275`: no hour spent. `README.md:27` and `propagation.go:86` give the minimum. m089, m090 and m091 are killed. |
| R5L-3 | **FIXED / RULED** | Me's rule is reversed by D-298, and m090 (its inverse) is killed. Mg's rule is withdrawn by D-296. Ma's head move is gone (`asker.go:281`); my Me′ (`Unasked(nil…)`) is killed. |
| R5L-4 | **FIXED, with a new defect (R6L-1)** | `snapshot.go:144` (`StationsNotReached`), `update.go:549`, and watchpost `map_prop_update.go:164-166,412-414`. The D-286 words, the flag and m568/m572 are gone. m092, m573 and m574 are killed. |
| R5L-5 | **RULED (A-94 (3))** | `propagation_test.go:239-255` now proves that the service applies a deadline (ErrNoRoom, 0 asked), and it pins R6L-2. The record (`red-team-review.md:548`) says the test is "reaching a source", but the code asserts `rec.count() == 0` (`:247`). The record is wrong. |

### 1. Happy Path Bias

**R6L-1. A host's cancel mid-burst is later said as a burst the deadline cut short.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **The mechanism:**
  - On a cancel, the `stops` path sets `NotReached = granted - i - 1` (`go-ionomaps asker.go:214-216`).
  - The burst's snapshot becomes `l.last` (`update.go:136`).
  - Every snapshot until the next burst carries the count, with GIRO Answered.
- **Probe P1** (`TestACancelledBurstSpendsTheHoursBudget`'s setup, cancelled at ask 5):
  - On reopening, the TooSoon snapshot gives `StationsNotReached 35, GIRO answered`.
  - The Fresh snapshot 10 minutes later is the same.
- **What watchpost does with it:**
  - The listener closes the map during the hourly burst, about 36 s after it opens.
  - watchpost notes "GIRO's burst cut short: 35 stations asked first next hour, **the update's deadline reached** (D-297)" (`map_prop_update.go:164-166`). The cause is false.
  - A-94 (2)'s premise, "a cancel's result is dropped", misses the library carrying the count.
- **Action:** carry the cause, or set NotReached only on the deadline path (`endEarly`), and test a cancel followed by a reopen.

**R6L-6. A burst that is never granted keeps the last burst's `NoRoom`, and Update then answers `ErrNoRoom` with no deadline at all, NOAA unasked.**
- **Severity:** Minor. It cannot be reached in watchpost, which calls `Restore` only in `NewService` (`propagation.go:128`).
- **Simplify/Delete?** N.
- **Evidence:**
  - `asker.go:196-198` resets the counts only when `granted > 0`.
  - The early returns at `:176-181` and a refused `Start` leave `NoRoom` stale.
  - `update.go:120-121` trusts it.
- **Probe P3:** a 1 ms burst, then `Restore` of a state whose hour is spent. Four updates with `context.Background()` each gave "it left 0s, and an update needs at least 20s", and NOAA was asked 0 times. This is a library API hazard.
- **Action:** reset `NoRoom` at the top of `Burst`.

### 2. Systemic Failure Modes

**R6L-2. The least deadline watchpost accepts can never ask anything.**
- **Severity:** Minor. Deadline is set only in code; production uses 60 s (`app/mapprop.go:43`).
- **Simplify/Delete?** N.
- **Evidence:**
  - `propagation.go:86` accepts `Deadline == MinDeadline`.
  - `update.go:351` refuses an ask when `left < 20 s`, which is always true once any time has passed.
  - watchpost's own test asserts exactly this, ErrNoRoom and 0 asked at `MinDeadline` (`propagation_test.go:241-249`).
- **Two statements say otherwise:**
  - The README, "at least `MinDeadline` (20 s)" (`README.md:27`).
  - The error's own words, "it left 19.999s, and an update needs at least 20s". Its path, "r refreshes now", fails again for good.
- **Action, a recommendation to the HUM LEAD, since A-94 (3) chose the test:**
  - Make the minimum mean something that works: either watchpost refuses `<= MinDeadline` (or adds a margin), or the library's check allows a small margin.
  - Test that the least accepted deadline asks something.

Otherwise, NOAA and GIRO take separate updates, each with a fresh 60 s (`propagation.go:224`). A slow network now ends as "cut short" with the stations rotated in first, which is true. Nothing else fails together unsaid.

### 3. Gate Validity

**R6L-4. Three rules introduced by round 5's fixes pass the suite with their code removed.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Mc survives:** the `stops` path's `NotReached = granted - i - 1` changed to `0` (`asker.go:215`). The count on a cancel, a 429 or a denial is untested, and it is R6L-1's mechanism.
- **Md survives:** `endEarly`'s `ctx.Err() == nil &&` guard dropped (`asker.go:276`). A cancel after two hangs would call GIRO down.
- **Mw survives:** watchpost's diagnostics guard `&& snap.Inputs.GIRO == ionomaps.GIROAnswered` dropped (`map_prop_update.go:164`). A-94 (1) is tested only for the status words (`TestACutShortBurstIsSaid`), not for the diagnostics.
- **Action:** add a test that kills each one.

**R6L-8. go-tuiMaps D-299's race split has no check that the race-off leg judges the peak.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:** `race_off_test.go:1` and `race_on_test.go:1`.
- Swapping the two build tags makes the race-off leg never judge NFR-3's peak, and nothing fails. Both quick legs run (`scripts/gate:720-721`).
- **Action:** a small check that `raceDetector` agrees with how the binary was built (for example, a race-only test file asserts it is true).

### 4. Adversarial Input

Nothing new beyond R6L-6.
- A GIRO answering each ask in just under 15 s is cut short and said so, with nothing backed off (D-295, as ruled).
- A GIRO that dies after about 28 s of a burst is not called down that hour: my probe, "dies at ask 31", gave Answered, 1 failed, 9 not reached. That is D-296's ruled counter-argument, and the next hour calls it down.
- A host `Fetch` that ignores its context can overrun the deadline. That is the contract; watchpost's client honours its context.

### 5. Graceful Degradation

**R6L-5. "N stations asked first next hour" counts the probe, which is not asked next hour.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:**
  - The probe is the last ask (`asker.go:190-193`).
  - `endEarly` counts `len(rest)` (`:280`).
  - `Unasked` passes over the probe (`stations.go:176,193-196`), and `nextProbe` moves on (`:241`).
- So whenever a burst with a probe is cut short, the words overstate by one. In probe P2, NotReached was 22 and 21 live stations were put first.
- **Action:** count only the live stations put first, or word the probe apart.

The cut-short, GIRO-down and no-room cases are otherwise said truly (sweep above; D-297's words gated on Answered).

**R6L-7. A stale doc names a parameter that no longer exists.**
- **Severity:** Minor (AP-HIST / docs reflect reality).
- **Simplify/Delete?** Y.
- **Evidence:** `map_prop_update.go:369-370` still says "burst is whether the snapshot came from GIRO's burst … (D-286)". The parameter and the D-286 words were removed in batch 69.
- **Action:** delete the sentence.

### 6. Test Confidence vs. Test Results

**R6L-3. The "slow answers, no hang" subtest is flaky.**
- **Severity:** Minor (the D-299 class: a verdict that depends on the machine's load).
- **Simplify/Delete?** N.
- **The defect is in the fixture:**
  - The scaled fixture's `answerIn` sleeps 30 ms and ignores the shortened ask's context (`asker_test.go:323-327`).
  - The scaled margin is 10 ms (`:315`).
  - A last ask given 20 to 30 ms finishes with 0 to 10 ms of slack, so scheduler jitter runs it past the deadline.
- **Evidence:**
  - Unmutated, 1 of 30 runs failed: `asker_test.go:432: 19 asked, 21 not reached … before the deadline false`.
  - The same failure appeared under mutants that cannot touch `internal/giro` (update.go's Mi and Mj, first pass) and under Mk.
  - So m084 to m088's "killed" verdicts rest partly on a test that can fail by itself.
- **Action:** make the fixture honour its context (`select` on `ctx.Done()`), as a real client does, and re-vet those mutants.

Otherwise, the new tests prove behaviour. The 8-hour hanging-station subtest holds R5L-1, and the NoRoom and Partial tests hold D-298.

### 7. Dependency Fragility

- **The deadline coupling:** it is now explicit and refused below 20 s, apart from R6L-2's boundary.
- **Timeouts:** `IsTimeout` is shared (`asker.go:447-450`, `update.go:358`). watchpost's 30 s client stays longer than both libraries' per-ask limits.
- **Pins and cache:** they match the reviewed commits.
- **`a23484f`'s untagged v0.2.2 requirement:** still unverifiable locally. The nested modules on go-tuiMaps `main` require tags that do not exist yet, so `main` cannot build from a clean module cache until the HUM LEAD tags them.

### 8. Problem statement present, real, honored

- **Present:** PS-1 and PS-2 (`problem-statement.md:14-31`).
- **Not yet measured:** PS-1 is still a hypothesis (D-217), and PS-2's M1 and M4 go to VALIDATE.
- **Fixed:** R5L-1's harm to PS-2 (an Observer told GIRO is down while 38 of 39 stations answer). My sweep found no position where it remains.
- **Cost:** D-295 to D-298 act only when a source is slow or the deadline is near, so the upper bound in item 3 (`validate-agenda.md:21`) still holds for batch 69.
- **Residue:** R6L-1 and R6L-5 are small untruths in words an Observer reads.

### Verified clean

- **Round 5:**
  - R5L-1 at every hang position over 3 hours.
  - R5L-2 to R5L-4 in code; R5L-5 as ruled.
  - The registered mutants m083 to m092 are all killed.
- **D-295:**
  - A burst with one hang uses its whole update (40 of 40 asked).
  - A shortened ask that runs out is "not reached", never a failure.
- **D-296:**
  - Two full timeouts in a row are needed at the end.
  - A 5xx between two hangs resets the run (my Ma is killed).
- **D-298:** NOAA's no-room path backs nothing off; GIRO's spends no hour.
- **Deleted cleanly:** batch 69's deletion of D-286's flag and words, with no Go code left behind (one comment, R6L-7).
- **D-299:** both go-tuiMaps quick legs run, and the gate is green.
- **watchpost legs:** lint, lint-authoring, p10, dupes and mutant-anchors; vet and race on the touched packages.

### Opinion

**Exit REVIEW.**
- Nothing I found is Critical or Important. D-300 makes a round with no Critical or Important the way into the exit report.
- Rounds 3 to 5's chain of timing Importants is broken: I swept one hang across every position, GIRO dying at every point and a slow GIRO, and each outcome is said truly and ends inside the deadline.
- R6L-1 to R6L-8 are Minor and can go in one batch. R6L-3 should go first in that batch, because it decides whether the gate is green.

**Fix first: R6L-1.** It is the one Minor on a common path: closing the map during the hourly burst. It tells the listener a false cause, the D-239 class, and it comes from round 5's own fix.

---

## Code Quality, round 6 (R6C): 0.19.0 REVIEW exit red team (D-300)

**Verdict: exit REVIEW.** I found no Critical and no Important. Round 5's Important (R5C-1) is fixed in the code. The round 5 fixes did add new surface, and all of it is Minor. The largest piece is the deadline arithmetic D-298 added. Its "minimum" deadline is not one: an update given exactly `MinDeadline` never asks anything (R6C-1). NOAA's room rule has no margin, so in a narrow window an update can still run into its deadline (R6C-2). Both have the same fix.

### Tree state and runs
- **`git status --short` at the start:**
  - watchpost `2ec275b5`: ` M .../02-analysis/rulings.md` (the D-300 row, not mine).
  - go-ionomaps `d5d4d6c`: clean.
  - go-tuiMaps `813b7b0`: clean.
- **At the end:** each library also shows ` M 06_docs/gate-runs.md`. That is not mine: I ran both gates on byte copies under `<scratch>/rt6-r6c/copies`.
- **I edited no tree.** Every probe and mutant was an `-overlay` from scratch. Every finding was reproduced against the three tips above.
- **go-ionomaps:** `scripts/gate --quick` on the copy is **green** at 16:31:43Z (race on 237 s, race off 42 s, P10 included).
- **go-tuiMaps:** `scripts/gate --quick` on the copy is **green** at 16:45:47Z.
  - One later root-package run of mine (with a mutant overlay) failed under host load after 177 s, and its output was not captured. Two reruns passed. I cannot attribute it, so it is not a finding.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `lint-authoring`, `p10` (0 live / 0 unmatched / 0 unratified), `dupes`, `mutant-anchors` (824), `go vet ./...`.
  - `go test -race` on `domains/propagation`, `app` and `modes/tty` (filtered to the Propagation tests).
- **Mutants (my overlays).** Scratch: `<scratch>/rt6-r6c/mutants`, `wmut`, `tmut`.
  - go-ionomaps, killed: yb, ye, yh, yi, yl, yj; yc and yg are killed at the module level.
  - go-ionomaps, survived: **yd** (whole module) and **yk** (an equivalent mutant).
  - watchpost: w1 (the 60 s default → 20 s), w2 (the refusal threshold) and w3 (cut-short words said for Unavailable) are all killed.
  - go-tuiMaps: the `raceDetector` flip **survived**.

### Part 1: round 5's findings, checked in the code
Round 5's report has no R5C-2 (its numbering skips it), so there is nothing to verify for it. Round 4 had no PARTIALs.

| ID | Status | Evidence |
|---|---|---|
| R5C-1 | **FIXED** | go-ionomaps `internal/giro/asker.go:250-263` (`askLimit`), `:207-209` (a shortened ask that runs out is `notReached`), `:272-282`. The subtest "one station hanging early" (`asker_test.go:385-396`) asks every station with NotReached 0. watchpost says the cut short at `map_prop_update.go:164-166,412-414`. New surface: R6C-1 to R6C-4. |
| R5C-3 | FIXED | x1: the no-room path sets only `noRoom` (`update.go:351-353`; m090). x2, x3 and x4: the code is gone (`endEarly` passes `""`; `lastFailed` and `timedOutAll` removed). New survivor: R6C-4. |
| R5C-4 | FIXED, residue | `TimeLeft` and `IsTimeout` are now written once (`asker.go:427-450`), and the counts are one `tally` (`:222-243`). But the room rule is still written twice, with different margins: R6C-2. |
| R5C-5 | FIXED, residue | `snapshot.go:155,182`, `asker.go:55`, `update.go:84-108`, `stations.go:177-181`. The throttle is restored when nothing was asked (`asker.go:273-275`; m086), and the error names the deadline (`update.go:225`). That error's rounding is under R6C-1. |
| R5C-6 | FIXED | `propagation_test.go:232-258` now reaches the library's no-room path and is reworded. It rests on the boundary defect in R6C-1. |
| R5C-7 | FIXED (D-299) | go-tuiMaps `memory_test.go:166,174`, `race_on_test.go`, `race_off_test.go`; the gate is green. The new judgement is unpinned: R6C-7. |

### 1. Understandability
- **R6C-2 (Minor), go-ionomaps: NOAA's room rule has no margin; GIRO's has one.**
  - **The two rules:**
    - GIRO's ask keeps `askMargin` (1 s) back (`asker.go:255`).
    - NOAA's ask starts with `left >= noaaAskTimeout` and gets the full 20 s (`update.go:351-356`), so it can end exactly at the host's deadline.
    - When that happens, `timedOut` needs `ctx.Err() == nil` (`update.go:358`), so the stall is not taken as a timeout. The round goes on, and the update returns Unfinished wrapping `DeadlineExceeded`.
  - **Probe** (`SetNOAAAskTimeout(100ms)`, solar ask stalled, deadline 100.3 ms): of 40 updates, **37 ended Unfinished with DeadlineExceeded** and 3 Offline.
  - **What it contradicts:**
    - `Update`'s doc says "An update never runs into ctx's deadline" (`update.go:99`).
    - `Unfinished`'s doc lists only a cancel or ErrNoRoom (`snapshot.go:182`).
    - In that window a stalled NOAA is not backed off.
  - **Production exposure:** the window is narrow. The earlier asks must leave within about a millisecond of 20 s.
  - Simplify/Delete: Y.
  - Action: one room helper, `room(ctx, need)` with the margin, used by both sources. That also settles R6C-1.
- Otherwise the changed code is easy to follow. `endEarly`'s doc says what it does.

### 2. Maintainability
- **R6C-4 (Minor), go-ionomaps: the host's cancel is booked two ways.**
  - **Two paths:**
    - A cancel before an ask goes to `endEarly`. It counts station *i* in `NotReached` and puts it first next hour (`asker.go:202-203`).
    - A cancel during an ask goes to `stops`. It sets `NotReached = granted - i - 1` and puts first only `asks[i+1:]` (`asker.go:214-216`). So the interrupted station is neither counted nor put first.
  - **Unpinned:** that stops-path count is pinned by nothing. Mutant yd (`a.NotReached = 0`) survives the whole module.
  - **Doc:** `Inputs.StationsNotReached` says "asked first next hour" (`snapshot.go:144`). After a denial GIRO is paused for a day, so that is false.
  - Simplify/Delete: Y.
  - Action:
    - Route `cancelled` through `endEarly` with `asks[i:granted]`, and drop the cancel case from `stops`.
    - Pin the count after a call-down, or say it is not counted.
    - Fix the doc.
- R6C-2 is also the same rule written twice (R5C-4's residue).

### 3. Complexity
Nothing is over-complex for its job.
- **One readability nit:** `tally.add` increments `timeouts` and then resets it (`asker.go:233-236`) where an if/else would read straight.
- **Shaped by a metric (noted, not numbered):**
  - The build log ("REVIEW's red team, round 5") and A-34 (5) say `TimeLeft` and `IsTimeout` went into `internal/giro` because a package of their own "failed P10's density".
  - So the root's NOAA code imports the GIRO package for a time helper (`update.go:14,351,358`).
  - Recommendation for the HUM LEAD: ratify a row for a small shared package rather than let the floor choose where the code lives.

### 4. Necessity
- **R6C-6 (Minor), go-ionomaps: a redundant branch.**
  - `roomless`'s `if !l.made` branch (`update.go:226-228`) builds the same snapshot that `early` returns when nothing has been made.
  - Mutant yk (`if false`) survives because it is equivalent.
  - Simplify/Delete: Y. Action: delete the branch.
- watchpost's `DeadlineExceeded` words (`map_prop_update.go:240`) are still reachable: R6C-2's probe produces that error. They are not dead.

### 5. Name semanticism
- `NoRoom` means two rules:
  - `Asker.NoRoom` (`asker.go:98`) is "no room for the first ask".
  - `Library.noRoom` (`ionomaps.go:84`) is "no room for any later NOAA ask, possibly after others".
- `TimeLeft` and `IsTimeout` are general helpers living in a package named for GIRO.
- Both are minor; I folded them into R6C-2's action rather than numbering them. Nothing else found.

### 6. Code documentation
- **R6C-5 (Minor), watchpost: a stale doc in batch 69.**
  - `snapshotWords`' doc still says "burst is whether the snapshot came from GIRO's burst, whose stopping names GIRO (D-286)" (`modes/tty/map_prop_update.go:369-370`).
  - Batch 69 removed that parameter. This is the only residue of the retired D-286 words (I checked with `git grep`).
  - Simplify/Delete: Y.
- **Docs that are false today:**
  - `MinDeadline` and `Update` (R6C-1).
  - `Update`/`Unfinished` (R6C-2).
  - The "files not asked" docs (R6C-3).
  - `StationsNotReached` (R6C-4).
- No history comments (AP-HIST-01) in the range.

### 7. Testability
- **R6C-7 (Minor), go-tuiMaps: D-299's judgement is unpinned.**
  - Flipping `race_off_test.go:8` to `raceDetector = true` passes (`go test .`, ok, 174.7 s). The peak is then judged nowhere, and nothing notices.
  - Simplify/Delete: N.
  - Action: a test that checks `raceDetector` against the `-race` key in `debug.ReadBuildInfo()`'s settings.
- **R6C-8 (Minor), go-ionomaps: one subtest does not check the case it names.**
  - "one hang as the update ends" (`asker_test.go:412-425`) never asserts that the hanging ask was a full one (`a.Failed == 1`).
  - If scheduler delay leaves less than 160 ms when the hang comes (scaled), the ask is shortened and the subtest still passes. m087 then survives.
  - So that mutant's kill depends on timing. This is by analysis; I did not reproduce it under load.
  - Action: assert `a.Failed == 1`.
- R6C-4: survivor yd.

### 8. P10 conformance
- `make p10` is 0/0/0, and both libraries' quick-gate P10 legs are green. `Burst` is 46 lines.
- No exemption was self-issued and no floor moved. The density-driven placement under question 3 is a recommendation, not a breach.

### 9. Evidence soundness
- **R6C-1 (Minor), go-ionomaps with watchpost: `MinDeadline` is not a working minimum.**
  - **Why it fails:** the check is strict (`left < noaaAskTimeout`), and `left` is read after the context was made (`update.go:351`). So a context of exactly `MinDeadline` is always under it.
  - **Probe:**
    - Exactly `MinDeadline`, with NOAA answering at once: ErrNoRoom, 0 asks. The error reads "it left 20s, and an update needs at least 20s" (`update.go:225` rounds).
    - `MinDeadline` + 50 ms: 3 asks.
  - **watchpost:**
    - It accepts `Deadline == MinDeadline` (`propagation.go:86`; its error at `:70` says "at least").
    - `TestAnUpdateHasItsOwnDeadline` (`propagation_test.go:241-247`) asserts that this accepted deadline asks nothing.
    - A-94 (2) (`rulings.md:423`) says watchpost's least deadline "never reaches" no room. The test proves the opposite.
  - **The doc is wrong in the other direction too:** "With less, nothing is asked" (`update.go:324-326`) is false for GIRO's burst. `TestACutShortBurstIsCounted` asks GIRO with a 4 s deadline.
  - **Production exposure:** none. `app/mapprop.go:43` sets no Deadline, so updates use 60 s.
  - Simplify/Delete: N.
  - Action:
    - Set `MinDeadline` to the ask plus the margin (with R6C-2), or have watchpost refuse `<=`.
    - Stop the error rounding to the minimum.
    - Correct the doc and A-94 (2), and re-point the watchpost test.
- R6C-3 and R6C-8 also belong here.

### 10. Evasion lens
- **R6C-3 (Minor), go-ionomaps: "the files not asked named unread" is met in letter only.**
  - **The ruling and docs:** D-298 and D-108, `Update`'s doc (`update.go:104-105`) and the README all say the files a no-room round did not ask are named in `Inputs.Unread`.
  - **The code** names only the file whose ask found no room (`update.go:384-386,415-418`). D-RAP or scales after it are skipped silently.
  - **Probe:** with scales held and young, a round cut at D-RAP gives `Unread [D-RAP]`. Scales is not asked and not named.
  - **How the test passes:** it asserts `UnreadScales` (`update_test.go:632`). It gets it from `short()` (`update.go:294-308`), because scales was never held in that test, not from the rule it names.
  - Simplify/Delete: N.
  - **Fork for the HUM LEAD:** name every file not asked; or correct the ruling's reading and the docs to "the file that found no room, and any the snapshot lacks". Either way, the test should hold a young file.
- R6C-1: watchpost "refuses a Deadline under 20 s" in letter, but 20 s itself asks nothing.

### Recommendation (D-295/D-287 territory, for the HUM LEAD and the timing lens)
- **Probe:** two adjacent stations that never answer, the rest at PLAN's pace, scaled 1:100, for 24 hours.
  - GIRO was never called down, which is correct.
  - **12 to 13 of 40 stations went unreached every hour.**
  - The pair sat at positions 21 to 23 and was asked at the full 15 s every hour. It stays live for up to `darkAfter` (3 days).
- The words say it honestly, so this is not a defect. But a third of the network unread each hour may matter. One option: a station whose last full ask ran out goes last next hour, or gets the floor.

### What I verified clean
- **go-ionomaps:**
  - `askLimit`'s three regimes and the D-296 call-down at the end (yb and yl killed).
  - `notReached` is never counted as a station's failure (yi killed).
  - A cancel before the first ask restores the hour, and NoRoom is set only with a live context (yc and yg killed by the module).
  - `l.asked` gates `roomless` (yj killed). The margin is pinned (ye killed). The ctx-cancel check is pinned (yh killed).
  - The burst NoRoom path keeps `stationsNext` and does not stick: the next burst resets it.
  - The ledger's m084 to m092 match the code. The doc scan's widened pattern still allows the published names. The doc-only diffs (`ahead.go`, `offset.go`, `reach.go`, `constants.go`) change no behaviour.
- **watchpost:**
  - The cut-short words are said only while GIRO answered, and noted once per change (m573, m574; w3 killed).
  - The default 60 s is pinned (w1 killed) and the refusal threshold too (w2 killed).
  - The retired flag leaves no code residue beyond R6C-5.
- **go-tuiMaps:** both legs run in quick mode (`scripts/gate:720-721`). The live line is still judged under race. The alias count of 17 holds.

### Opinion
**Exit REVIEW.** Round 6 found no Critical or Important in my lens. The eight Minors can be batched before VALIDATE. **Fix first: R6C-2.** One room rule with a margin, shared by NOAA and GIRO, closes the deadline overrun and makes `MinDeadline` a true minimum (R6C-1) in the same change.

---

## Timing lens, round 6 (R6T): 0.19.0 REVIEW exit red team (D-300)

**Tree state.**

| Repository | At the start | At the end |
|---|---|---|
| watchpost | `2ec275b5`, ` M …/02-analysis/rulings.md` (not mine) | unchanged |
| go-ionomaps | `d5d4d6c`, clean | ` M 06_docs/gate-runs.md` |
| go-tuimaps | `813b7b0`, clean | ` M 06_docs/gate-runs.md` |

- The two `gate-runs.md` rows are another reviewer's. One of them is go-ionomaps 16:34Z, "FAILED: tests, race detector on".
- I ran my gates in rsync copies under `<scratch>/copies`, and their rows stayed in those copies.
- I edited no tree. Every probe and mutant went in by `-overlay` from `<scratch>/rt6-r6t`.
- Every finding below was reproduced against the three tips above.

**What I ran:**
- **watchpost:**
  - `make lint` (6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes`, `mutant-anchors` (824 match): all exit 0.
  - `go vet` on propagation, tty and app: clean.
  - `-race` on `domains/propagation` and on `modes/tty -run Prop|Stations|Burst`: green.
- **Libraries:**
  - go-ionomaps `scripts/gate --quick` green with P10 (16:51Z, in the copy); go-tuimaps green with P10 (16:59Z, in the copy).
  - `-race` on giro, stations and throttle: green.
- **Pins:** rc.10 and rc.9 resolve to `d5d4d6c` and `813b7b0`. The module cache's `update.go`, `asker.go`, `stations.go` and `throttle.go` are byte-identical to the tree.
- **Probes,** in `internal/giro` at 1/100 scale (key results re-run at 1/50), with an answer at 0.92 s against a 15 s ask, PLAN's ratio:
  - deadlines from 20 s to 120 s, GIRO hung on every ask;
  - a hang beginning after k = 0 to 39 answers, over 2 hours;
  - 1 or 2 hanging stations at each position from 1 to 39, over 12 hours;
  - steady answers at 0.92, 1.3, 1.45, 1.5, 2, 5 and 14.5 s;
  - hangs alternating with a 5xx.
- **Top-level probes:** NOAA's room at exactly 20 s; how long the build takes after a burst (56 ms, 389 ms under `-race`, inside the 1 s margin); a cancelled burst's counts; a tty quit with an update in flight.
- **Mutants:** 25 applied. 20 killed; 5 survived (R6T-6).

### Part 1: the timing findings of rounds 3 to 5

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R3L-1 | FIXED | go-ionomaps `update.go:340,351-363`: each NOAA ask gets 20 s, and a timeout stops the round and sets `trouble`; `:154-161` then backs NOAA off. Mutants u3 (drop `IsTimeout`) and u4 (a timeout goes on) are both killed. |
| R3L-2 | FIXED | `asker.go:307`: three full asks run out in a row call GIRO down whatever answered. Probe: a hang after k ≤ 12 answers is Unavailable inside the update. |
| R3C-1 | FIXED | Same as R3L-2. |
| R4L-1 | FIXED, as ruled | `asker.go:250-263,272-282`. In the k = 0 to 39 sweep, every hour ended inside 60 s. For k ≤ 27 GIRO is called down that hour; for k ≥ 28 it is called down the next hour, which is the cost D-296's counter-argument accepted. |
| R4C-1 | FIXED | Same as R4L-1. |
| R4L-3 | FIXED | `asker.go:328,447-450`; `update.go:358`. u3 is killed. |
| R5L-1 | FIXED | `asker.go:32,276`. One hanging station, at each position 1 to 39, over 12 hours: never called down, never cut, the probe asked 12/12. |
| R5C-1 | FIXED | `askLimit` `asker.go:255-262`: with one hang, all 40 stations are asked every hour. A cut burst is counted (`asker.go:280`, `update.go:549`) and said (`map_prop_update.go:412-414`). |
| R5L-2 | FIXED, with residue (R6T-2) | `update.go:351-354` backs nothing off when there is no room; `roomless` is `:223-232`; `asker.go:273-275` keeps the hour; watchpost refuses a short Deadline at `propagation.go:86`. Mutants u1, u2, u5, u6 and w1 are killed. |
| R5L-4 | FIXED, with residue (R6T-4) | D-297's count and words are in place. |
| R5D-1 | FIXED | `update.go:99-108,324-327`, `README.md:27-28`, R-5.3. |

### 1. Bounded

Every GIRO burst ended inside its deadline at every timing I could construct. That covered every hang position, every deadline from 20 s to 120 s, answers just inside each limit, and slow answers up to 14.5 s.

Two qualifications:
- **NOAA has no margin.** That is R6T-5.
- **Bounding depends entirely on the host's `Fetch` returning once its context ends.** A fake that ignores the context (5 s answers) ran past the deadline every hour, 12 of 12. `ionomaps.go:17-23` (the `Fetcher` doc) never states that duty. This is a Minor, folded into R6T-5's action.

**R6T-5: NOAA's asks have no margin, so an ask can run to the host's deadline.**
- Evidence: go-ionomaps `update.go:351` refuses an ask only below `noaaAskTimeout`, and `:356` gives it the full 20 s. With 20.000 s to about 20.003 s left, the ask's limit and the host's deadline coincide.
- `:358` then requires `ctx.Err() == nil`, so the stall is not counted as a timeout, and `:154-156` says the update Unfinished: no back-off, and the error reads "its deadline passed".
- Probe at scale: 28 of 60 runs in that window ran late and were said Unfinished.
- The same window exists in production. The window is narrow, but `Update`'s doc (`:99`) says an update "never runs into ctx's deadline", and that does not hold there.
- GIRO has `askMargin` (`asker.go:43`); NOAA has none.
- Severity: Minor. Simplify/Delete? N.
- Action:
  - give NOAA an ask margin, refusing an ask when `left < noaaAskTimeout + margin`;
  - state in `Fetcher`'s doc that `Fetch` must return once `ctx` ends.

### 2. Truthful outcomes

These hold:
- a hung GIRO is called down whenever the deadline is at least 32 s;
- one station's failure is that station's own;
- a burst cut short is counted;
- NOAA's no-room path backs nothing off (u1 is killed).

Exceptions: R6T-2 (a hung GIRO not called down below 32 s), R6T-3 (the probe counted as "asked first next hour"), R6T-4 (a cancel said as the deadline) and R6T-7.

**R6T-4: a burst the host cancelled is said, all hour, as cut by "the update's deadline".**
- Mechanism:
  - On a cancel, `stops` sets `NotReached` (`asker.go:214-216`). The result Outcome stays `Answered`, and the count is carried into every later snapshot of the hour (`update.go:549`).
  - The listener leaves the mode during the burst, which runs in the first ~40 s after pressing P, and comes back.
  - watchpost then says "GIRO's burst cut short: 30 stations asked first next hour". The diagnostics add ", the update's deadline reached (D-297)" (`map_prop_update.go:164-165`).
  - The deadline part is false: the listener's own cancel cut the burst. A-94 (2)'s premise that "a cancel's result is dropped" misses that the count outlives the dropped result.
- Probe: a cancel at ask 10 gives `notReached=30`, GIRO answered, carried by the TooSoon snapshot 2 minutes later and the Fresh one 12 minutes later.
- The cancelled station itself is neither counted nor put first (`asks[i+1:]`, `:216`).
- Severity: Minor. Simplify/Delete? N.
- Action:
  - record why the burst ended (cancel or deadline), or drop "the update's deadline reached";
  - include `asks[i]` in the stations put first next hour.

### 3. Edges

- **15 s and 16 s left (GIRO):** correct. A full ask needs 16 s left and ends with 1 s to spare. A 15 s remainder gives a 14 s shortened ask. Under 3 s left, no ask starts.
- **20 s left (NOAA):** see R6T-2 and R6T-5.
- **The margin before a full ask is untested:** mutant g13 survives (R6T-6).
- **A timeout beside a cancel:** correct in the code, but untested (g2 and g14 survive).
- **Two timeouts with a 5xx between:** not a run. See R6T-7.

**R6T-2: `MinDeadline` (20 s) is not a working minimum.**
- **At exactly 20 s, nothing is ever asked.**
  - Evidence: `update.go:351`, then `roomless`.
  - The time that passes before the check leaves under 20 s, so the update returns `ErrNoRoom` at once.
  - watchpost accepts `Deadline == MinDeadline` (`propagation.go:86` refuses only values below it).
  - Its own test pins this: `propagation_test.go:241-249` asserts that the accepted least deadline asks nothing.
  - D-298 meant to refuse deadlines that cannot ask. This one is admitted.
- **From 20 s to 31 s, a GIRO hung on every ask is never called down.**
  - Probe, at both 1/100 and 1/50 scale: each hour gives 2 asks, 1 failed, 39 not reached, `Answered`, nothing backed off.
  - Evidence: `asker.go:256-262` gives the second ask a shortened limit; when it runs out it is "not reached", so `t.timeouts` stays at 1 against `timeoutsAtTheEnd` 2 (`:276`).
  - D-296 needs at least 32 s; D-286's three in a row needs at least 47 s. D-295's counter-argument, "D-286's three in a row still catches any hang begun earlier", fails below that.
- **The burst path never checks `MinDeadline` (`update.go:118-121`).** It asks with 3 to 20 s, though `MinDeadline`'s doc (`:324-326`) says that with less, nothing is asked.
- Severity: Minor. watchpost runs at 60 s, and its Deadline is not a user setting. go-ionomaps is a public library.
- Simplify/Delete? N.
- Action, a recommendation to the HUM LEAD: raise `MinDeadline` to cover two full GIRO asks plus the margin (32 s), or make the refusal exclusive and document GIRO's need. Fix the test to assert an ask at the least deadline that is accepted.

### 4. Across hours

These hold:
- no station starves or stays last for good when one station hangs or a trio hangs;
- no hour is spent with nothing asked (`asker.go:273-275`, g7 killed);
- no back-off grows without cause: `Answered()` resets it on any answer, and NOAA's no-room path is clean.

**R6T-3: a burst cut short never asks the rotation probe, and the words count the probe as "asked first next hour".**
- Mechanism:
  - The probe is always the last ask (`asker.go:189-193`).
  - `Plan` advances `probeAt` whether or not the probe is asked (`stations.go:170,241`).
  - `Unasked` passes over codes that are not live (`stations.go:193-195`).
- So every hour a burst is cut, one probe is skipped for good. Probes over 12 hours:
  - GIRO answering at 1.45 s an ask: probe asked 0 of 12 hours, cut by 1 to 3 stations every hour;
  - two hanging stations, at any position from 1 to 38: probe asked 0 of 12 hours.
- While that lasts, none of the 79 stations that are not live, and no station that has gone dark, can return. That breaks D-51 and R-5.5 ("one not-live station probed per update").
- `NotReached` counts the probe (`asker.go:280`), so "N stations asked first next hour" (`map_prop_update.go:304-308`) says one more than will happen (D-239).
- Severity: Minor. It self-heals as the live set shrinks, and it needs a slow GIRO or two hanging stations.
- Simplify/Delete? N.
- Action: keep `probeAt` when the probe was not reached, or ask the probe before the tail; count the probe apart from the live stations.

### 5. Tests

The ratios are realistic: `scaledAsks` is 9 ms against 150 ms, PLAN's 1:16.

| Mutant | Result |
|---|---|
| g1, g3 to g7, g9, g10, g12, g15 | killed |
| g11 (the no-room flag set on a cancel) | killed, by the root package |
| u1 to u6 (NOAA's side) | killed |
| w1, w2, w4 (watchpost) | killed |

**R6T-6: five timing rules pass the whole suite with their code broken.**
- **g2** `asker.go:276`, dropping `ctx.Err() == nil`: a cancel after two timeouts would call GIRO down.
- **g8** `:215`, `NotReached = 0` on a stop: the count on a stop is unpinned.
- **g13** `:256`, a full ask allowed with 15 s left instead of 16: the margin before a full ask, the boundedness rule itself, is unpinned.
- **g14** `:325-327`, the cancel check skipped: the host's cancel would count as the station's failure, and before any answer it could add to the three-in-a-row call-down.
- **w3** watchpost `map_prop_update.go:164`, the GIROAnswered clause dropped: the diagnostics would say "cut short" for a burst GIRO itself stopped, contrary to A-94 (1).
- Severity: Minor. Simplify/Delete? N.
- Action: one test and one ledger row for each.

### 6. Host and library

These agree:
- **Deadline:** each `Ask` gets its own 60 s (`propagation.go:224`). One ask at a time per session (`busy`), and a cancelled update frees the library's lock at once.
- **Refresh tick:** set after each result, so it never falls before NOAA's `notBefore`.
- **Words:** they match the library's counts, except R6T-3 and R6T-4.

The wait at exit does not.

**R6T-1: quitting during GIRO's burst does not stop it, so the hour it spent is not kept, and a restart bursts GIRO again within the hour.**
- **Evidence:**
  - `q` returns `tea.Quit` at once (`modes/tty/dashboard.go:1424-1425`); `stopPropagation` is never called.
  - The session's context comes from `context.Background()` (`map_prop_update.go:56-57`).
  - The app's `cancel()` (`app/dashboard.go:132`) does not reach that context.
  - `Settle` waits `settleBound` = 2 s (`app/mapprop.go:56-71`), while a burst runs up to about 59 s. The process exits before `keepState` (`propagation.go:226-227`).
  - The throttle's `Start` for that burst was held in memory only, so the state file still says GIRO is due.
- **What the docs claim:** `Settle`'s doc (`propagation.go:233-236`, "the mode's context cancelled, the burst stopping at its next ask") and its test (`propagation_test.go:414-416`, "a burst the quit cancelled") both describe a cancel that no quit makes. The test releases a held fake instead.
- **Probe:** an overlay tty test opens P with the update in flight and presses `q`. It gets quit=true, and the update's `ctx.Err()` is nil after quit.
- **The burst comes in the first ~40 s after P is pressed.** Opening P, quitting within that time and relaunching makes two bursts of up to 40 asks within minutes. That breaks D-213 ("a restart within the hour asks GIRO nothing new") and D-39 (at most hourly). Against GIRO's measured bucket of about 90, a second quick relaunch reaches a 429.
- Severity: Important. Simplify/Delete? N.
- Action: cancel the Propagation session on quit, or derive its context from one the app cancels before `stopAll`. Test that a quit with a burst in flight ends the update and keeps the state within `settleBound`.

### Recommendations to the HUM LEAD (D-296's choices; not findings against the code)

**R6T-7: D-296's words and two of its consequences.**
- **The words are ambiguous.** "Two full-length asks … with no answer between them" can read as letting a 5xx sit between them. The code resets the run on any result that is not a run-out (`asker.go:234-236`).
- **Hangs alternating with a 5xx, after one answer, are now never called down.** Probe: 4 of 4 hours `Answered`, 6 failed, 33 not reached each hour.
  - D-291 named this case as a defect and required a test.
  - That test is now inverted (`asker_test.go` "hangs alternating with a server error" asserts not called down).
  - D-296's counter-argument does not name it.
- **Two adjacent hanging stations call GIRO down in many hours.** At hour-0 position 17: 7 of 12 hours; at 14 to 19 and 28 to 29: 1 to 7 hours. That is R5L-1's class with two stations instead of one.
- Severity: Minor. Simplify/Delete? N.
- Action: rule whether a 5xx breaks the run, and whether the alternation case stands; then align the words and the test.

### Verified clean

- **Every GIRO hang position and every deadline from 20 s to 120 s ends inside the update.** One hanging station at any position never calls GIRO down, is never cut, and the probe is asked over 12 hours.
- **A hung GIRO is called down at 32 s and above; D-286's three in a row at 47 s and above.**
- **NOAA's rules are pinned:**
  - D-285's 20 s, its back-off and `IsTimeout` on both sides;
  - no-room handling, `ErrNoRoom` and `roomless`;
  - the GIRO hour is restored when nothing was asked.
- **The post-burst build** takes 56 ms (389 ms under `-race`), inside `askMargin`.
- **Watchpost:** the 60 s deadline per `Ask`; the session `busy` guard; the order of the refresh tick and NOAA's back-off; the pins; every gate leg named in the brief.

### Opinion

**Do not exit REVIEW.** R6T-1 is Important: on an ordinary path (open P, quit within the first minute, relaunch) it breaks a ruled guarantee, D-213 with D-39, and the test written for it claims a cancel the app never makes. The fix is small. R6T-2 to R6T-7 are Minor and can be batched.

The library's own timing code is no longer where the Important lies. Round 5's remediation holds under every timing I built, and the next defect was at the host's seam.

**Fix first: R6T-1.**
