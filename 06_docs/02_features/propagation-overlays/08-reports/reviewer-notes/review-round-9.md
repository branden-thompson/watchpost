---
title: "0.19.0 REVIEW exit red team, round 9 - the reviewers' reports, verbatim"
date: 2026-10-27
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - four fresh blind reviewers, each also on amd64, each judging timing against the update's timing model as D-314 changed it (D-315), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 9."
---

# Round 9, the reports

Up: [red-team review](../red-team-review.md) · [round 8](review-round-8.md)


---

## Docs Quality, round 9 (R9D): 0.19.0 REVIEW exit red team (D-315)

**Tree state.** I ran `git status --short` at the start and at the end, and all three trees were clean both times:
- watchpost `32324c39`
- go-ionomaps `f60cc5c`
- go-tuimaps `91c3630`

Every finding was reproduced against that state. Probes and mutants ran only in my own scratch copies (`<scratch>/rt9-r9d/iono`, `iono-mut`, and `iono-rc13`, which is `git archive 59cb4a9`). The quick gate ran in its own clone, `iono-gate`.

**What I ran:**
- **watchpost lint targets:** `make lint` (no new findings, 6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes` and `mutant-anchors` (827 match). All exited 0. `go vet` was clean.
- **watchpost tests:** `go test -race` on `domains/propagation` (whole) and on `app -run AppsEnd|AsBuilt|Prop`. The same ran under `GOARCH=amd64`, all green. The app's-end test took 0.65 s on amd64.
- **go-ionomaps:** `go vet ./...` on arm64 and amd64. `-race` on `stations`, `deadline` and `giro` (the retry, hang, dead and kept-probe tests). The same under amd64. All green.
- **Quick gate:** go-ionomaps `scripts/gate --quick` is green.

### Part 1: round 8's findings

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R8D-1 | FIXED for its case; the fix brings in R9D-1 | `stations.go:177-180`: the retry is chosen from `last` inside the budget, so a station known to hang is asked every hour with 40 or more live (`TestTheRetryTakesTheStationsKnownToHangInTurn`, 45 live). |
| R8D-2 / R7D-5 | FIXED | `asker_test.go:793-850` now compares reads against every live station not hanging, two hours running. I re-applied my round-8 mutant (`rotation` skips `live[0]` when 2 or more are known to hang) on a copy. The test now kills it ("unread two hours running [AL945]"). |
| R8D-3 | FIXED | The model gains a run's row (`update-timing-model.md:96`), and `snapshot.go:144` gains its words. |
| R8D-4 | FIXED | Model `:57` now reads "before any answer, the failures-in-a-row run too". |
| R8D-5 | FIXED | Node `IO` is at rc.14 and node `L` at rc.10 (`as-built-map.md:112-113`). The atlas carries both. Wording drift remains: R9D-4(a). |
| R8D-6 | FIXED | README `:35-36` has "the probe and those known to hang apart". The citations are by repository and path, and no relative `../go-ionomaps` link is left. The ledger row `p10-ledger.md:28` now says "renewed by the agent … as A-37 (3), batched for the HUM LEAD's veto". |
| R8D-7 / R7D-6 | FIXED | Model `:136` is reworded, and its copies match: README `:45`, `validate-agenda.md:25`, CHANGELOG. The changelog now lists all six limits. |
| R8D-8 | FIXED | `00-REQUIRED-READING.md:53` gives "% of one core", "one run on one machine" and the 2 h 40 m outage. A-96 now names arm64. |

### 1. Does each document answer the question its title asks?

Nothing new found beyond the findings below. The model's section 8, "The limits this model accepts", misses one limit (R9D-1) and understates another (R9D-3). Both are filed under questions 2 and 5.

### 2. Does anything contradict the code, or another document?

**R9D-1: M2's retry starves a station known to hang that recovers but answers in more than 2 s. It is the retry every hour, gets only 2 s, and never reaches the tail. So it is never read until it goes dark (3 days), and then it waits on the probe rotation. Section 8 says it "waits for the tail". Principle 3 says "no station is starved".**
- **Where it happens:**
  - `Plan` takes the retry out of the order (`stations.go:177-180`).
  - `Burst` gives the retry a 2 s ask, and its running out is `hungAgain` (`asker.go:223-225`, `:237`).
  - `RanOut` re-appends every station that runs out (`stations.go:209`).
  - So when the tail is reached and its stations also run out, they go back behind the retried station. The same station is the retry next hour. That is not "in turn" (model `:85`).
- **Probe** `TestR9DSlowRecoveredStationAlone`: 39 live, scaled 1:100, deadline 60 s, GIRO at 0.9 s an ask. The station answers in 3 s (scaled).

  | Stations known to hang | Hours the slow station was read, of 24 | Its asks |
  |---|---|---|
  | 1 (the slow one) | 0 | retry in 24 of 24 hours |
  | 2 (the slow one and a dead one) | 0 | retry in 24 of 24 hours |
  | 3 (the slow one and two dead) | 23 | it reached the tail |

  - The results are the same on arm64 and amd64.
  - **On rc.13 (`59cb4a9`) the same probe reads it in 24 of 24 hours in all three cases.** This is a regression brought in by round 8.
- **What the docs claim, against the code:**
  - Principle 3 (`update-timing-model.md:28`): no station is starved.
  - Section 3's "so each is retried in turn: with k stations known to hang, each within k hours" (`:85`).
  - Section 8's fifth limit (`:138`).
  - The copies: README `:47`, `validate-agenda.md:25`, and CHANGELOG `:41-44` ("One answering slower than 2 seconds waits until the hour has room").
- **No test reaches it:**
  - `TestEveryStationKnownToHangIsRetriedInTurn` answers in 1.6 s (`asker_test.go:891`), inside the 2 s floor.
  - `TestTheRetryTakesTheStationsKnownToHangInTurn` uses 45 live (`stations_test.go:311-335`), so the tail is always cut.
- **How likely:** unmeasured. It needs a station that answers in 2 to 15 s. The evidence holds only PLAN's average, 39 asks in 36 s.
- **Kind (b), the model is wrong.** The code follows section 3 literally ("the retry apart"; the order kept by run-outs), but section 3's mechanics break its own sentence at `:85`, principle 3 and section 8.
  - It touches principle 3; section 1's retry row; section 3 items 2 and 5 and its known-to-hang paragraph; section 6 (the retry's order); section 8's fifth limit; and R-5.3.
  - It qualifies D-314's M2 ("the retry apart").
- **Fork for the HUM LEAD, put whole:**
  - (i) A retry that runs out keeps its place in the tail and gets a full ask when the tail is reached. That costs one more ask of a dead station each hour.
  - (ii) The retry turns strictly: no station is retried again while another known to hang waits.
  - (iii) Accept it, and section 8 states the real length: an answer slower than 2 s is unread until dark, 3 days, plus the probe rotation.
  - Whichever is chosen, add a test with a station answering slower than the floor and the tail reached.
- Severity: **Important**. Simplify/Delete? N.

**R9D-2: the retry's room is read at the burst's start, not at the retry's ask. Section 1 says the retry is "made only while 35 s are left" (`update-timing-model.md:40`).**
- `roomToRetry(ctx)` is evaluated once, in `Plan`'s call (`asker.go:205`, `:253-256`). A kept probe (item 1) then takes a full ask before the retry.
- **Probe** `TestR9DRetryRoomAfterKeptProbe`: a dead GIRO and a kept probe not known to hang.

  | Update's deadline | Left at the retry | GIRO |
  |---|---|---|
  | 36 s | 21 s | not called down (`Answered`) |
  | 40 s | 25 s | not called down (`Answered`) |
  | 45 s | 30 s | not called down (`Answered`) |
  | 50 s | 35 s | called down |

  - The retry's uncounted run-out breaks the probe's timeout run.
  - The results are the same on arm64 and amd64.
- **Control:** a mutant on my copy checks the room at the retry's ask. With it, GIRO is called down at all four deadlines.
- This breaks principle 2 for one hour, for a host giving 35 to 49 s. A-38 (2)'s "a hung GIRO is still called down" is untested with a kept probe.
- watchpost is not affected: its 60 s leaves 44 s after the probe.
- Kind (a), a departure. Severity: Minor. Simplify/Delete? N.
- Action: check the room at the retry's ask, and add a test at 40 s with a kept probe.

**R9D-4: smaller drift from the code and the model, batched.**
- **(a) The as-built map.** Node `IO` (`as-built-map.md:112`) has two gaps:
  - "its running out counting toward no run once a station has answered, and before any answer as any other" reads as covering the retry. But the retry's running out never counts (`asker.go:237`).
  - "a probe not reached asked first" leaves out M3. A kept probe known to hang is asked last (`stations.go:188`).
- **(b) The go-ionomaps README `:27-29`** has the same gap: "once a station has answered … their running out counts toward neither" omits that the retry counts nothing even before an answer (model `:60`).
- **(c) go-ionomaps R-5.5** (`requirements.md:77`) still says "39 polled in rotation plus the probe … those left out asked first next time". The model's section 3 (`:83`, `:78`) gives the retry a place and leaves stations known to hang last. R8D-1 named R-5.5 as touched, and it was not changed.
- **(d) The changelog** `:45` says "a station that has never answered". The model's limit (`:139`) is a dark probe: one with no reading for 3 days, which may have answered before. The changelog also leaves out that the deadline left it unreached.
- Severity: Minor. Simplify/Delete? N. Action: reword each one. Note that `TestTheAsBuiltMapIsKeptInStep` (`app/maps_test.go:55-81`) still pins only the batch number, so nothing holds the node's words or versions.

### 3. Is a claim made that nothing verifies, in a document that reads as verified?

These overlap with the findings above:
- **The build log claims M2 is "in turn"** (go-ionomaps `build-log.md`, round 8; watchpost batch 73). It was tested only where the tail is never reached (R9D-1).
- **A-38 (2) claims a hung GIRO "is still called down within MinDeadline".** That is untested with a kept probe (R9D-2).

Nothing else found.

### 4. Is the audience order right: designers, then PMs, then engineers?

Nothing found:
- The changelog's Known entry is now a list worded for a listener.
- The README's limits lead with what a host sees.
- The model's history-first opening suits its readers, the HUM LEAD and engineers.

### 5. Is a number published without its blind spots beside it (INST-5)?

**R9D-3: section 8's sixth limit says "with GIRO at about 1.05 s an ask or slower, that can leave 2 to 5 live stations to the hour after" (`update-timing-model.md:139`). The range was measured only at 1.05 to 1.15 s (R8T's probe), but the text applies it to "or slower".**
- **Probe** `TestR9DDarkProbeCostBySpeed`: a dark probe the deadline left, not yet known to hang, that hangs. Stations not reached, with the probe hanging against without it:

  | GIRO's pace | arm64 | amd64 |
  |---|---|---|
  | 1.1 s an ask | 5 against 0 | 6 against 0 |
  | 1.3 s an ask | 9 against 0 | 10 against 1 |
  | 1.6 s an ask | 15 against 7 | 15 against 7 |

- The copies say the same: README `:48`, `validate-agenda.md:25`, CHANGELOG `:46` ("a few").
- Kind (b): section 8 only. It is worse than stated.
- Severity: Minor. Simplify/Delete? N.
- Action: state the cost as about 15 s of asks (about 15 ÷ the pace, less the hour's slack), with the range measured and the pace at which it was measured. Change the copies with it.

**R9D-5: `app/race_test.go:5-10` publishes `raceSlowdown = 10` on "past 2 s under race on a hosted runner", with no hosted figure.**
- The factor's basis is not stated. The local ratio is 405 ms against 60 ms, about 6.75 times. How far past 2 s the runner took is unrecorded, because the test prints only the bound.
- No hosted race-leg run on batch 74 is recorded: `verify-runs.md` shows only local verify rows, and `red-team-review.md:752-754` has no result.
- Severity: Minor. Simplify/Delete? N.
- Action: have the test print the time it took, record the hosted leg's figure, and state the factor's margin against it.

### What I verified clean

- **M1:** `asker.go:237` counts a station known to hang before any answer. `TestADeadGIROIsCalledDownEveryHour` is green under race and on amd64.
- **M3:** `stations.go:188` returns first as false for a kept probe known to hang.
- **The budget:** the retry counts inside the 40 (`:180`). A dark station known to hang is never polled.
- **The model:** section 2's rows, section 4's run row and section 1's 35 s arithmetic (2 + 1 + 2 × 16) match the code, apart from R9D-2.
- **R-5.3:** every test it names exists. A-38 and D-120 restate D-314 faithfully.
- **watchpost:** batch 73's citations, its blind spots and the as-built map's versions; the atlas regenerated; the re-pin to rc.14 in `go.mod`. Batch 74's race and no-race build tags hold. All five lint targets and the quick gate are green.
- **go-tuimaps:** its code is unchanged.

### Opinion

**Do not exit REVIEW.** Round 8's fixes hold for the findings they answered. But M2's retry brings in a new starvation, R9D-1, which is Important, a regression from rc.13, and on both platforms. A recovered station that answers in 2 to 15 s is retried in the same 2 s every hour and never read. That breaks principle 3 and makes section 8's fifth limit false. R9D-2 to R9D-5 are Minor and can be batched with it.

**Fix first: R9D-1.** Put it to the HUM LEAD as a whole-model change: principle 3; sections 1, 3, 6 and 8; R-5.3; and D-314's M2. Add a test with a station answering slower than the floor and the tail reached.

Probe files, in `<scratch>/rt9-r9d/iono/internal/giro/`: `zz_r9d_probe_test.go` and `zz_r9d_probe2_test.go`. The rc.13 comparison is in `<scratch>/rt9-r9d/iono-rc13`, where `<scratch>` is `<scratch>`.

---

## REVIEW lens, round 9 (R9L): 0.19.0 REVIEW exit (D-315)

**Tree state at the start, and still at the end:** watchpost `32324c39`, go-ionomaps `f60cc5c` (v0.1.0-rc.14), go-tuiMaps `91c3630`. `git status --short` was empty in all three, before and after. I edited no tree and changed no git state.

**Where things ran:**
- The library quick gates ran in my clones at `<scratch>/rt9-r9l/iono` and `/tuimaps`.
- The probes and mutants ran on an rsync copy, `<scratch>/rt9-r9l/iprobe`. Each mutated file was restored and checked with `cmp` against the tree.
- The probes are in `<scratch>/rt9-r9l/probes/`, where `<scratch>` is `<scratch>`.

**What I ran:**
- **watchpost (tree):**
  - `make lint`: no new findings, 6 baselined.
  - `lint-authoring`: OK.
  - `p10`: 0 live.
  - `dupes`: 2 groups, both ratified.
  - `mutant-anchors`: 827, all match.
  - `go vet` is clean on `app`, `domains/propagation` and `modes/tty`.
  - `TestTheAppsEndStopsAnUpdateAndKeepsItsState` passes under `-race` on arm64, and 3 of 3 times under `GOARCH=amd64 -race`.
- **Library quick gates:** go-ionomaps green, go-tuiMaps green.
- **go-ionomaps `internal/giro` and `internal/stations`:** green under `-race` on arm64 and on amd64.
- **Pins:** the module cache's rc.14 `asker.go`, `stations.go`, `update.go`, `snapshot.go`, `deadline.go` and `throttle.go` match the tree byte for byte.

### Part 1: round 8's findings

| Finding | Verdict | Evidence checked in code |
|---|---|---|
| R8L-1 | **FIXED** | go-ionomaps `internal/giro/asker.go:237`. My 96-hour dead-GIRO probe, P2: GIRO is called down every hour from 0 to 72, on arm64 and amd64. My mutants Ma (the condition made `true`) and Mj (`code == retry` dropped) are both killed. After 72 hours a new gap opens: see R9L-1. |
| R8L-2 | **FIXED** | I re-applied N1, N2 and N3 myself. N1 (`inARow++`, `asker.go:303`) is killed by `TestAStationKnownToHangBreaksARun` and `TestADeadGIROIsCalledDownEveryHour`. N2 (`return false`, `:304`) is killed. N3 (`slices.Backward`, `stations.go:275`) is killed by `TestTheRetryTakesTheStationsKnownToHangInTurn`. |
| R8L-3 | **FIXED** | `stations.go:177-181`. I replayed my round 8 probe: 41 live stations, one hanging in hours 0 and 1. It was read in hour 2 and in 67 of the 70 hours after, on arm64 and amd64 (round 8: 0 of 71). Mutant Me (no retry) is killed by three tests. |
| R8L-4 | **FIXED** | watchpost `CHANGELOG.md:33-46` lists all six of section 8's limits. |
| R8L-5 | **FIXED** | `stations.go:189`. Mutant Mf is killed by `TestAKeptProbeKnownToHangIsAskedLast`. |

### 1. Happy Path Bias

**R9L-1. A station list left unasked for three days goes dark as a whole. watchpost then asks one GIRO station an hour and takes about 110 hours to read the 39 again. In a GIRO outage longer than three days, GIRO is never again said to be unavailable.**
- **Severity:** Important.
- **Simplify/Delete?** N.
- **Kind (b): the model is wrong.**
- **The mechanism:**
  - Whether a station is live is judged by the wall clock: data heard within 72 h, or placed by the seed and seeded within 72 h (go-ionomaps `internal/stations/stations.go:144-153`).
  - A restored state keeps `Seeded` and `Heard` exactly as they were (`stations.go:411-418,435`).
  - The Propagation mode is not remembered (D-154), and bursts run only while it is open (watchpost `modes/tty/map_prop_update.go:51`, the only caller of `PropagationUpdate`). So any user who leaves 3 days between sessions comes back to a list where no station is live.
  - With no station live, `Plan` returns only the probe (`stations.go:174-192`). That is one ask an hour, and one station re-lit each time it answers.
- **Probes, all reproduced on arm64 and on amd64:**
  - **P1:** a healthy burst on day 0 (39 live), then a restart 4 days later. Every hour the burst asks **1** station, with 0 live and 0 readings held.
  - **P3:** the same 39 stations answering again. It takes **110 hours** for all 39 to be live again: 8 at 24 h, 21 at 48 h.
  - **The contrast:** deleting the state file would do better. A cold start polls the 39 placed stations at once. So the persistence D-213 added makes the return worse than having no state at all.
  - **P2:** GIRO hung on every ask. GIRO is called down every hour up to hour 72. From hour 73: 1 ask, `Answered`, `Failed=1`, not paused.
- **Why this is against the model:**
  - Principle 2 says GIRO "is called down when nothing answers". For a burst of one ask, nothing in section 2's call-down list can apply.
  - Section 8 does not list this limit.
  - R-5.5 / D-51 say "dropped after 3 days with no data". Here the station was dropped for not being asked, not for having no data.
- **What the user sees (`map_prop_update.go:398-409`):**
  - "no station readings this hour: the background alone" for hours, then days, though "this hour" implies it will pass;
  - in an outage past three days, "1 station did not answer" each hour, never "GIRO unavailable".
- **No test covers either case:** none of go-ionomaps' tests restores a state older than `darkAfter`.
- **Recommendation to the HUM LEAD, one change:**
  - **The change, in two parts:**
    - (i) a station's three days count only time the list was being asked: when a state is resumed, the time since its last burst is added to every `Heard` and to `Seeded`;
    - (ii) with no station live, a burst polls the stations the seed places, as a cold start does.
  - **What it touches:**
    - principles 2 and 3;
    - section 3 (a new item before item 4: the placed stations when none is live);
    - section 6 (across restarts: the clock is not counted while the list was not asked);
    - section 8 (no new limit; part (ii) removes an unlisted one);
    - go-ionomaps R-5.5.
  - **What it supersedes:** D-51's "dropped after 3 days with no data", read as wall-clock time. The throttle state that D-213 keeps is unaffected.
  - **The cost:** none beyond D-87's budget of 40.
  - **Tests to add:** P1 and P3 as tests (all 39 polled in the first hour back); P2, asserting `Unavailable` at hours 73 to 96.

### 2. Systemic Failure Modes

R9L-1 is the systemic one. Every station goes dark at once, after an absence or after a long GIRO outage, and the release neither says why nor names the case in its known limits. Otherwise NOAA and GIRO keep separate 60 s updates (`update.go:122-127`), so one failing does not take the other's time.

### 3. Gate Validity

**R9L-2. The retry's room is checked before the kept probe's ask, not when the retry's turn comes. At host deadlines of about 36 to 49 s, a dead GIRO is called down only every other hour.**
- **Severity:** Minor. watchpost gives every update 60 s, where all 24 hours are called down.
- **Simplify/Delete?** N.
- **Kind (a): a departure from section 3, item 2,** "a 2 s ask, while 35 s are left".
- **The code:**
  - `roomToRetry(ctx)` is read once, at `Plan` (go-ionomaps `asker.go:205`).
  - The kept probe is then put in front of the retry (`:210-211`) and takes a full 15 s ask.
  - The retry's `hungAgain` breaks the run of timeouts (`:303`), and the deadline cuts the burst before a second counted timeout, so the end-of-update call-down (`endEarly`, `:328`) does not fire.
  - In a dead GIRO the probe is kept every hour (`Unasked`, `stations.go:234-235`).
  - So the probe-first hour is not called down. The next hour, the probe has run out and is known to hang, the burst opens with the retry, and GIRO is called down.
- **The claims this falsifies:**
  - `roomToRetry`'s doc (`asker.go:250-252`) and A-38 (2): "a hung GIRO is still called down within `MinDeadline`".
  - The gate behind that claim (m122) never has a kept probe.
- **Probe P4:** a dead GIRO for 24 hours at scaled deadlines.
  - **Result:** at 40 s and 45 s, called down in **12 of 24** hours (D.D.D…). At 35 s, 50 s and 60 s, all 24.
  - **Control:** on a copy, check the room at the retry's turn and skip the retry without it. Then all five deadlines give 24 of 24.
  - Reproduced on arm64 and on amd64.
- **Action:** read `roomToRetry` at the retry's turn, skipping the retry and keeping it known to hang. Add a dead-GIRO test with a kept probe at a 40 s deadline.

**The mutants I applied:** Ma to Mj and Ml were all killed. Mi does not compile, so it tested nothing; m122 covers it. Mk (`room > 1` → `room > 0`) survives, but it is equivalent in production because `Plan`'s budget is always `MaxBurst`. That is not a finding.

**A-98:** the race build is held to 20 s, and m575 is still killed under race. The 2 s value stays pinned in the legs without race. Sound.

### 4. Adversarial Input

- **A hostile state file:** a `Last` with duplicate codes is accepted (`stations.go:434`, unlike `LeftOut` at `:423`).
  - It is harmless: `rotation` (`:275-278`) asks each code once, and `RanOut` (`:207`) deletes every copy.
  - A `Last` naming every station is now harmless under M1.
- **A state whose `Seeded` and `Heard` are old:** see R9L-1.
- Nothing else found.

### 5. Graceful Degradation

R9L-1's words are its user-facing half: "no station readings this hour" lasts for days, and an outage of more than three days is reported as one station failing.

**R9L-3. A burst that reads one station is said as "1 stations, oldest HH:MM".**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:** watchpost `modes/tty/map_prop_update.go:399`. The tests cover only 3, 4 and 40 (`map_prop_states_test.go:99,103`, `map_prop_stations_test.go:54`).
- **Why it matters now:** it is the normal case in R9L-1's re-lighting hours.
- **Action:** the singular, as the failure count's words already have (`:405-406`), plus a test.

### 6. Test Confidence vs. Test Results

- **R9L-2:** the only test of the retry's room (`TestABurstEndsBeforeTheHostsDeadline`, "a hung GIRO at the least deadline") has no kept probe. It proves its fixture, not the order section 3 sets.
- **R9L-1:** every list test starts from a fresh `Seed(now)`, so no test restores a stale state.
- **R8L-3's replay is now real:** `TestTheRetryTakesTheStationsKnownToHangInTurn` runs with 45 live, past the budget.

### 7. Dependency Fragility

- **Pins:** rc.14 and rc.10 are pinned and match the cache.
- **`a23484f`:** still not in the local go-tuiMaps repository (`git cat-file` fails; I may not fetch), so `main`'s pins of the nested modules at v0.2.2 are still unverifiable here. This is unchanged from rounds 7 and 8.
- **Upstream:** the R9L-1 outage case is what a GIRO down for more than three days would do. GIRO has had outages of that length.

### 8. Problem statement present, real, honored

- **Present:** PS-1 and PS-2 (`problem-statement.md:14-31`).
- **Not yet measured:** PS-1 is still a hypothesis (D-217). M1 and M4 are still short of target, and go to VALIDATE.
- **The G1 re-run:** it measures `c8a99e48`, which pins go-ionomaps rc.13, not the rc.14 that will ship. rc.14 adds at most one 2 s ask an hour, so I expect no measurable cost, but the verdict should say which build it measured. Its `NOTE-load.txt` load caveat applies to round 9 as well.
- **Harm to PS-2 (R9L-1):** an Observer who opens the Propagation mode less often than every three days gets GIRO's readings from about one station for roughly the first day back, and from all 39 only after about 4.5 days. The cause is never said.

### Verified clean

- **Round 8:** all five R8L findings, checked in code and by my own mutants.
- **M1:** a dead GIRO is called down every hour up to 72 h.
- **M2:** retries go in turn ("longest ago" holds even when the tail is reached). A station known to hang is read within k hours with 41 live. The retry's 2 s ask is whole or none (Mh killed).
- **M3:** a kept probe known to hang is asked last.
- **The budget:** stays at 40, with the retry counted (Ml killed).
- **A-98's race bound, and batch 73's re-pin.**
- **Gates:** both library quick gates, watchpost's five legs, and vet.

### Opinion

**Do not exit REVIEW.**
- **R9L-1** is Important. It is the timing model's state across restarts meeting D-154 and D-51. It degrades GIRO's part for every occasional user, and it reopens principle 2 for any outage longer than three days. It is one model change (sections 3, 6 and 8, principles 2 and 3, R-5.5), not a patch.
- **R9L-2** is round 8's own retry, a small fix on the library's side.
- **R9L-3** is a line of code.

**Fix first: R9L-1.**

---

## Code Quality, round 9 (R9C): 0.19.0 REVIEW exit red team (D-315)

**Verdict: do not exit REVIEW.** There is one Important and three Minors. Every round-8 Code finding is fixed in the code. The Important is the model being wrong (kind b). The retry from M2 takes its station out of the tail every hour it is retried. So a recovered station that answers in more than 2 s is never read when it is the only station known to hang. With a GIRO slower than about 2 s an ask, no station known to hang is ever read again. Section 8 says such a station "waits for the tail", but that tail never comes.

### Tree state and runs
- **`git status --short`, at the start and at the end:** clean in all three repos (watchpost `32324c39`, go-ionomaps `f60cc5c`, go-tuiMaps `91c3630`).
- **Every finding was reproduced against those tips**, on arm64 and on amd64 (Rosetta), with the same results.
  - Probes and mutants were `-overlay` files under `<scratch>/rt9-r9c/` (`probe/`, `mut/`). No tree was edited.
- **go-ionomaps `scripts/gate --quick`**, run on a scratch clone at `f60cc5c` with the two yml files copied in: **green**.
  - Race leg 248 s, race-off leg 65 s, P10 green.
  - Its gate-runs row is in the clone only.
- **amd64:** `internal/giro`, `stations` and `deadline` are green.
- **watchpost, all exit 0:**
  - `make lint`, `lint-authoring`, `p10` (0/0/0), `dupes`, `mutant-anchors` (827), `go vet ./...`.
  - `TestTheAppsEndStopsAnUpdateAndKeepsItsState` passes:
    - `-race` on arm64: 1.84 s;
    - without race: 0.37 s;
    - `-race` on amd64: 3.14 s.
- **go-tuiMaps:** no code change since rc.10. `a23484f` is not in the local clone, and I may not fetch, so I could not check it.

### Part 1: round 8's Code findings, checked in the code
| ID | Status | Evidence |
|---|---|---|
| R8C-1 | FIXED (M1) | go-ionomaps `internal/giro/asker.go:237`: before any answer, a timeout of a station known to hang now counts. My 24-hour probe at a scaled 60 s: `Unavailable` every due hour (arm64 and amd64). `TestADeadGIROIsCalledDownEveryHour` (`asker_test.go:852`) holds. At deadlines of 35 to 50 s the retry breaks this again: see R9C-2. |
| R8C-2 | FIXED | zK (= m125) is killed by `asker_test.go:934` (`Failed != 7`). zF (= m130) is killed by `stations_test` `TestTheRetryTakesTheStationsKnownToHangInTurn`. I re-ran both. |
| R8C-3 | FIXED | `asker_test.go:803` and `:814-817`: four stations are known to hang from the start. `:833`: reads are checked against every live station not hanging. |
| R8C-4 | FIXED, but the wording carries R9C-1 | watchpost `CHANGELOG.md:33-46` has all six limits and says "until an ask of it ends any other way" (`:42`). Its `:43-44` sentence is false; see R9C-1. |
| Nits | FIXED / RULED | `State.Last`'s doc is fixed (`stations.go:363`). The `MinDeadline` literal stays, as ruled. |

### 1. Understandability
- **R9C-1 (Important), go-ionomaps: a station known to hang that answers in more than 2 s is never read again while it is the only one known to hang. With GIRO slower than about 2 s an ask, none known to hang is read. This breaks principle 3, and section 8, bullet 5 says otherwise.**
  - **Kind (b), the model is wrong.** The code follows the model:
    - section 3, item 5 asks the stations known to hang "the retry apart" (`stations.go:177-181` deletes the retry from `order`);
    - section 1 gives the retry 2 s (`asker.go:223-225`);
    - section 8, bullet 5 still says a station answering in more than 2 s "waits for the tail".
  - **Why the tail never comes:**
    - With k = 1, the only station known to hang is the retry every hour (`stations.go:177`: the first live code in `last`), so it is never in the tail.
    - With k ≥ 2, the slower station ends up alone in the list (the others answer in the tail and leave it), and then k = 1 applies.
    - With a GIRO above about 1.5 to 2 s an ask, the tail is not reached (R8T-2), and the 2 s retry runs out every time.
  - **Probes** (scaled ÷100, 600 ms update; arm64 and amd64 alike):
    - GIRO at 0.3 s an ask, one station known to hang answering in 5 s: **read 0 times in 10 hours**, still known to hang.
    - k = 2: the second is read 10 of 10 hours; the first is read 0 of 10.
    - GIRO at 2.5 s an ask for every station, three recovered stations known to hang: **read 0 times in 24 hours**.
      - 22 to 23 asks an hour.
      - `NotReached` leaves them out (`asker.go:326`), so only "1 station did not answer" is said.
  - **What ends it:** the station goes dark after 72 h (`stations.go:28`), and is then read only when the probe rotation reaches it.
  - **Tests miss it** because `TestEveryStationKnownToHangIsRetriedInTurn` (`asker_test.go:875`) recovers its stations at 1.6 s, inside the retry's 2 s.
  - **Same false claim in the docs:** watchpost `CHANGELOG.md:43-44`, VALIDATE item 7 (`validate-agenda.md:25`), go-ionomaps `README.md:47`, and the model's `update-timing-model.md:138`.
  - **What a fix touches:**
    - principle 3;
    - section 1 (the retry row);
    - section 3 (item 5 "the retry apart"; the paragraph at `:85`);
    - section 8, bullet 5;
    - D-87's budget, if one station is asked twice.
  - **Options for the HUM LEAD, whole:**
    - **(A), recommended:** a retry that runs out keeps its place in the tail, so the tail gives it a full ask when the hour reaches it.
      - It costs one ask in the budget.
      - It supersedes D-314 M2's "taken out of the poll" (A-38 (1)).
      - Section 8 then states the true bound for a GIRO slower than 2 s an ask: unread until it goes dark.
    - **(B):** the retry is a full ask when the update leaves 48 s or more.
      - This fixes both cases, but costs up to 13 s of each hour while a station is truly dead: about 14 stations fewer at 0.92 s an ask.
  - Simplify/Delete: N.
  - Action: the HUM LEAD rules the change whole. Then add a test with one recovered station at 5 s and one with a GIRO at 2.5 s.

### 2. Maintainability
- **R9C-2 (Minor; Important if go-ionomaps ships to hosts with a deadline under 51 s), go-ionomaps: the retry's room rule assumes the retry is the first ask, but a kept probe goes before it. A dead GIRO under a 35 to 50 s deadline is then said answering every other hour.**
  - **Kind (b).** The rule is written in two places that do not see each other:
    - `roomToRetry` (`asker.go:253-256`) counts the retry's ask, its margin and two full asks (35 s), as section 1 (`:40`) says;
    - `Burst` (`asker.go:207-211`) puts a kept probe first, as section 3, items 1 and 2 say.
  - **The mechanism:**
    1. After a call-down, the next burst opens with the kept probe and a station known to hang.
    2. The kept probe's timeout is counted (timeouts = 1).
    3. The retry's run-out is `hungAgain` and breaks the run (`:237`, `:296`).
    4. Only one full ask then fits, so `endEarly` (`:328`) sees one timeout, not two.
  - **Probe** (a dead GIRO, 24 h, every due burst; arm64 and amd64 alike):
    - At a scaled 40 s: `Unavailable` and `Answered` alternate every hour; the `Answered` hours have 36 → 3 stations not reached. A host following section 4 would say "cut short", with GIRO dead.
    - At 40 and 45 s with no station known to hang: called down.
    - At 60 s: called down every hour, so watchpost is not affected (the burst is its own update, `update.go:122-123`).
  - **Breaks:**
    - principle 2 ("called down when nothing answers") and principle 4;
    - `update.go:325-327` says MinDeadline is chosen "so every rule of the update holds at it". The rules hold at 32 s but fail between 35 and 50 s.
  - **Options for the HUM LEAD, whole:**
    - (a) The retry's room counts the kept probe's full ask when one opens the burst (51 s). This touches section 1's retry row and section 3, item 2.
    - (b) The retry is asked before the kept probe. This swaps section 3's items 1 and 2 and supersedes D-303's "first" for that probe.
  - Simplify/Delete: N. Action: the HUM LEAD rules it; add a test of a dead GIRO under 40 s with a kept probe.

### 3. Complexity
- Nothing is over-complex.
- Nit, not numbered: `stations.go:177` computes `IndexFunc` before it tests `withRetry`. `if withRetry { … }` would read more plainly.

### 4. Necessity
- Nothing is dead.
- `room > 1` (`stations.go:177`) matters only for a budget of 2. Production always passes 40 (`asker.go:205`), so my mutant zQ (`room > 0`) survives as equivalent there. It is defensive only; no action.

### 5. Name semanticism
- Nit, not numbered: `RanOut`'s `Source:` line (`stations.go:202`) still says "D-307, D-309", while its doc cites D-314.
- `List.last` keeps its position-based name; this was accepted in round 8.

### 6. Code documentation
- **R9C-3 (Minor), go-ionomaps: the `Inputs.StationsNotReached` doc, changed in rc.14, says "a run of failures or timeouts called down, counts none". GIRO called down by D-296 at the deadline does count.**
  - Evidence: `snapshot.go:144`.
  - Code: `asker.go:325-330` counts first, then calls down.
  - The asker's own doc (`asker.go:99`) is right.
  - Probe at a scaled 60 s: `Unavailable` with `NotReached` 34, 29, 24…
  - Section 4's two rows are ambiguous on this case.
  - watchpost guards on `GIROAnswered` (`map_prop_update.go:164`, `:411`), so no wrong words are shown.
  - Simplify/Delete: N. Action: the doc reads "a run of failures or of three timeouts called down counts none; GIRO called down at the deadline counts".
- No history in comments (AP-HIST-01) anywhere in either diff.

### 7. Testability
- **R9C-4 (Minor), go-ionomaps: two rules of the retry are unpinned. My mutants survived `./internal/giro`, `./internal/stations` and the root package (arm64).**
  - **zM:** `roomToRetry` with one margin fewer (`asker.go:255`, `3*`→`2*`).
    - The only test is at 32 s (`asker_test.go:468`), below both thresholds.
    - At 34 s the mutant retries, and a dead GIRO is not called down.
  - **zO:** the retry asked before the kept probe (`asker.go:207-214`).
    - Section 3's order of items 1 and 2 is not asserted anywhere.
  - Killed: zS (the retry chosen from dark stations), and round 8's zK and zF.
  - Simplify/Delete: N. Action: add a boundary test at 34 and 35 s, and assert a kept probe then the retry. Both fall to R9C-2's ruling.

### 8. P10 conformance
- The quick gate's P10 is green, and watchpost `make p10` is 0/0/0.
- `Burst` is 59 lines (`asker.go:190-248`), at the limit, but not shaped by it.
- `internal/stations`' renewed floor (1.29) is still the agent's, under veto (A-37 (3)). It awaits the HUM LEAD's veto window; it is not self-issued.

### 9. Evidence soundness
- A-38 (2) and the comment at `asker.go:250-252` claim the retry's room keeps a hung GIRO called down. That is true only at the one deadline tested, 32 s (R9C-2).
- Note, batch 74: `raceSlowdown = 10` (watchpost `app/race_test.go:10`) is not derived from a measured figure. The record says only "past 2 s" on the hosted runner (A-98, build log).
  - The product's 2 s bound is still pinned without race, where m577 is killed.
  - I would record the hosted figure.
- Note: `verify-runs.md` has a FAILED verify at `6b85f268`. Its cause is not in batch 73 or 74's log. The green row carries the same commit with another tree hash, batch 74's tree.

### 10. Evasion lens
- R9C-1: "no station is starved" is proven only for stations that answer inside the retry's 2 s (`asker_test.go:891`, 16 ms against a 20 ms floor). R8D-1 and R8T-2's starvation is still there above 2 s.
- R9C-2: "called down within MinDeadline" is proven at 32 s, and fails between 35 and 50 s.

### Verified clean
- **go-ionomaps:**
  - M1 at 60 s over 24 h.
  - M3: a kept probe known to hang is asked last (`stations.go:189`).
  - The retry:
    - asked short, "whole or none" (`asker.go:224`);
    - counted once in the budget;
    - never polled besides;
    - left out of `NotReached`;
    - put behind the others when it runs out (`RanOut`).
  - A 500 still clears a station known to hang.
  - `Unasked` and the `leftOut` filter with a retry.
  - m118 to m130 as recorded.
- **watchpost:**
  - The re-pin to rc.14.
  - The `raceSlowdown` mechanics; m575 is killable under race (the held ask runs to 15 s, past 20 s).
  - The six limits, R9C-1's sentence apart.

### Opinion
**Do not exit REVIEW.** R9C-1 is the same starvation as round 8's (R8D-1, R8T-2), back above 2 s, so M2's fix holds only for stations that answer fast. Present R9C-1 and R9C-2 to the HUM LEAD as one change to sections 1, 3 and 8, and batch R9C-3 and R9C-4 with it.

**Fix first: R9C-1.**

---

## Round 9, Timing lens (R9T): report

I checked the tree state at the start and again at the end, and all three repositories were clean both times (`git status --short` empty):
- watchpost `32324c39`
- go-ionomaps `f60cc5c` (rc.14)
- go-tuiMaps `91c3630`

Every finding below was reproduced against those clean tips, using scratch clones under `<scratch>/rt9-r9t/`: `iono-p` holds the probes, `iono-m` the mutants, `iono-f` the candidate fixes.

Gates and checks:
- **go-ionomaps `scripts/gate --quick`:** green.
- **go-tuiMaps `scripts/gate --quick`:** green.
- **watchpost** `make lint`, `lint-authoring`, `p10` (0/0/0), `dupes` and `mutant-anchors` (827): all green.
- **`go vet` and `-race`** on `domains/propagation`, `app` (`Prop|AppsEnd`) and `modes/tty` (`Prop`): green.
- **amd64:** go-ionomaps' `giro`, `stations`, `deadline` and `throttle` packages, the root package's timing tests and the watchpost packages above are all green under `GOARCH=amd64` (Rosetta). Each probe result below was reproduced on both arm64 and amd64.

## 1. Round 8's timing findings, checked against the code

| Finding | Verdict | Evidence |
|---|---|---|
| R8T-1 / R8C-1 / R8L-1 | FIXED | go-ionomaps `internal/giro/asker.go:237`: `(t.anyAnswered \|\| code == retry)`. `TestADeadGIROIsCalledDownEveryHour` (`asker_test.go:852`) is green on both platforms. I re-applied M1's revert and it is killed. |
| R8T-2, R8D-1, R8L-3 | FIXED for stations answering in under 2 s | `stations.go:177-181` and `asker.go:205-208,223-225`. A residue is new defect R9T-1. |
| R8T-3 | FIXED | watchpost `CHANGELOG.md:39-46`; the model, line 139 |
| R8T-4 | FIXED | m124 to m129 are registered. I re-applied X1 and X9 and both are killed. |
| R8T-5, R8D-4 | FIXED | The model, line 57: "before any answer, the failures-in-a-row run too" |
| R8C-2 | FIXED | m125, m130 |
| R8C-3, R8D-2 | FIXED | `asker_test.go:816` (the hanging stations are taken from the live list); `:833` (reads are checked against the live stations minus the hanging ones) |
| R8D-3 | FIXED | The model, line 96; `snapshot.go:144` |
| R8D-7 | FIXED | The model, line 136; `CHANGELOG.md:37-38` |
| R8L-2 | FIXED | m125, m126, m127 |
| R8L-5 | FIXED as ruled | `stations.go:189` (m123). The rule it brought in starves the probe rotation: see R9T-2. |

## 2–3. The lens

### 1. Bounded

Clean. `TestR9TSweep` ran 400 hours of bursts on each platform, the same `Asker` throughout:
- host deadlines from 20 s to 120 s (scaled 1/100);
- four modes: dead, mixed hangs and 5xx, answers just inside each limit, and healthy;
- the kept probe and the retry arising naturally.

There were 0 violations. Every ask kept the margin and the floor, and every burst ended before its deadline; the least time left at a burst's end was 7.2 ms (amd64) and 7.7 ms (arm64), against a 10 ms margin.

### 2. Truthful outcomes

**R9T-3 (Minor, kind (a): a departure from section 1).** The retry's 35 s is checked when the burst is planned, before the kept probe's full ask. So the retry is made with about 20 s left, while section 1 says it is "made only while 35 s are left".
- **Code:** `roomToRetry(ctx)` is read once, at `Plan` (go-ionomaps `asker.go:205`). The kept probe is put ahead of the retry (`:210-211`), and nothing reads the room again at the retry's own turn (`:223-225`).
- **What follows:** the kept probe runs out (a counted timeout); the retry's run-out then breaks the run (section 2); only one full ask fits after it.
- **Probe** `TestR9TKeptProbeAndRetryDeadGIRO`: GIRO dead, one station known to hang, a kept probe not known to hang.
  - At every host deadline from 36 s to 48 s: `Outcome` is Answered, not paused, `Failed` 3, `NotReached` 37.
  - Same on amd64.
  - A host would say "3 stations did not answer · GIRO's burst cut short: 37 stations asked first next hour", while GIRO answered nothing. That breaks principles 2 and 4, A-38 (2) ("a hung GIRO is still called down within `MinDeadline`") and `MinDeadline`'s doc, which says "every rule holds at it" (`update.go:325-330`).
- **Watchpost is unaffected:** it gives each update 60 s, and at 49 s or more the dead GIRO is called down.
- Severity: Minor. Simplify/Delete? N.
- **Action:** read `roomToRetry` again at the retry's own turn, and skip the retry when there is no room.
  - On `iono-f`, this fix (plus R9T-2's) leaves every suite green except the test that pins the old M3 rule.
  - With it, a dead GIRO is called down at every deadline from 32 s to 60 s.

Otherwise:
- No back-off comes from the host's deadline or its cancel: `asker.go:226-233,319-333`; `update.go:361`.
- A station's failure is its own: `asker.go:404-418`.

### 3. Edges

Clean, checked at each boundary:
- **16 s and 21 s left, the floor, and a shortened ask that answers:** `deadline.go:37-50`; `asker.go:380`.
- **The retry's floor:** the retry asks for all of the floor or nothing: `asker.go:224`.
- **32 s:** with a kept probe, the two-timeouts rule still fires (my sweep).
- **A timeout beside a cancel:** the cancel is checked first, `asker.go:376`.
- **Two timeouts with a 5xx or an answer between them:** `tally.add`, `:291-308`.
- **A host client that gives up first:** `deadline.IsTimeout`. Watchpost's client allows 30 s, longer than any ask (`platform/httpx/plain.go:79-80`).
- **A path of no length:** `TestAPathOfNoLengthIsFinite` is green on amd64.

### 4. Across hours

**R9T-1 (Important, kind (b): the model is wrong).** A lone station known to hang is always the retry, so it is never in the tail. If it recovers answering in 2 to 15 s, it never gets a full ask while it is live, even with GIRO healthy and the hour half empty.
- **Mechanism:**
  - `Plan` takes the retry out of the order (`stations.go:177-180`), and section 3, item 5 says "the retry apart".
  - With k = 1, that station is the retry every hour, and gets only a 2 s ask (`asker.go:224`).
  - It runs out, becomes `hungAgain`, and stays known to hang (`:236-238`).
- **Probe** `TestR9TSlowRecovererStarves`: the station answers in 3 s, everything else is healthy, 80 hours.

  | Case | Result |
  |---|---|
  | 39 live, GIRO at 0.92 s (PLAN's pace) | Never read; dark at 72 h, so it then waits on the probe rotation |
  | 39 live, GIRO at 1.6 s | Never read |
  | 45 live | Read in hour 77, by the probe once it had gone dark |

  The 39-live, 0.92 s case was reproduced on amd64.
- **This is a regression from M2 in the healthy case.** Under rc.13, the tail was reached at 0.92 s, so the station was read the next hour (round 8 measured it read the next hour at 1.4 s).
- **Section 8 understates it.** Its limit 5 says "one that answers slower than the retry's 2 s waits for the tail" (model, line 138); `CHANGELOG.md:43-44` and go-ionomaps `README.md:47` say the same. For k = 1 there is no tail for it.
- **Why no test sees it:** `TestEveryStationKnownToHangIsRetriedInTurn` answers at 1.6 s (`asker_test.go:891`), under the retry's 2 s.
- **Recommendation to the HUM LEAD, whole:**
  - **The change:** retry a station known to hang only when the last burst did not reach it - the deadline or the budget left it. Otherwise it keeps its full ask in the tail.
  - **No new state:** `leftOut` already records the stations a burst left (`stations.go:183,251`; section 6's `LeftOut`). The retry becomes "the first live station in `Last` that is also in `LeftOut`".
  - **It touches:** principle 3; section 1 (the retry's row); section 3, items 2 and 5; section 8, limit 5; R-5.3; the changelog's and README's limits.
  - **It supersedes:** D-314 M2's "each burst", and A-38 (1) in part.
  - **It keeps:** R8D-1's and R8T-2's bound.
  - **Alternative:** accept it, and state the limit truly in section 8: "a lone station known to hang that answers slower than 2 s is not read until it goes dark and is probed, about 3 to 6 days".
- Simplify/Delete? N.

**R9T-2 (Important, kind (b); it breaks R-5.5).** M3, together with D-303's keep, freezes the probe rotation on one dark station known to hang. No other dark station is then probed, for as long as GIRO answers slower than about 1.13 s an ask (or a live station known to hang fills the tail).
- **Mechanism:**
  - A dark probe that hung once is known to hang for good (`RanOut`, `stations.go:203-211`); dark stations never leave `last`.
  - About 158 hours later the rotation picks it again (79 dark stations, two hours each while probes hang and are kept).
  - M3 asks it last (`stations.go:189`). If the hour is short, it gets a shortened ask, runs out as `notReached`, and `Unasked` keeps its turn again (`stations.go:234-236`). The next hour is the same, and so on.
- **Probe** `TestR9TProbeRotationFreeze` (every dark station hangs; answers carry readings, so live stations stay live):

  | GIRO's pace | Result |
  |---|---|
  | 1.2 s an ask | From hour 158 to hour 339, the same probe, AC843, every hour: 181 hours and still frozen. 78 dark stations never probed again. |
  | 1.2 s an ask, amd64 | Frozen from hour 158 to the end of the run (175 hours) |
  | 0.92 s (control) | The rotation goes on |

- **What this breaks:**
  - R-5.5, "one not-live station probed per update, **in rotation**" (go-ionomaps `requirements.md:77`), which wins any conflict;
  - section 3, item 6's purpose, "so a dark station can return".
- **It is said nowhere.**
- **Introduced by round 8:** before M3, the kept probe was asked first, ran out in full, and the rotation moved on.
- **Recommendation to the HUM LEAD, whole:**
  - **The change:** a probe known to hang that a burst did not reach does not keep its turn. The rotation moves on, and the probe waits for its next turn. In code, `stations.go:234` gains `&& !l.Hangs(l.probe)`.
  - **It touches:** section 3 (item 1's exception, and its last paragraph, "except those known to hang", now covering the probe); section 4 (the deadline row's "next hour"); section 6 (`Keep` is never a station known to hang); section 8, limit 6; R-5.3's "a kept probe known to hang is asked last".
  - **It supersedes:** D-314 M3's "then it is this hour's probe". M3's outcome, that such a probe never opens a burst, is kept.
  - **Evidence:** applied on `iono-f`, `giro`, `stations`, `deadline` and the root package stay green except `TestAKeptProbeKnownToHangIsAskedLast`, which pins the current rule. The freeze is gone: 200 hours at 1.2 s, 79 probes, none kept twice.
- Simplify/Delete? N.

Clean otherwise:
- The GIRO back-off is capped at 15 minutes and reset by any answer (`throttle.go:78-88,122`).
- No hour is spent with nothing asked: `asker.go:226-228,321-323`.

### 5. Tests

**R9T-4 (Minor, kind (a)).** The new timing rules are pinned too loosely. Each mutant below was applied to a copy, and `giro`, `stations`, `deadline` and the root package were run whole (arm64):
- **MB survives.** I asked the retry before the kept probe, swapping section 3's items 1 and 2. Only the P10 mirror test failed, and it fails on an unmutated clone too (no `.a2dh` file), so nothing behavioural catches the swap.
- **MA2 survives behaviourally.** I lowered `roomToRetry`'s 35 s to 34 s (`asker.go:255`); only the anchor test (m122's line moved) and that same mirror failed. m122 is coarse (`left >= 0`).
- **MA is killed:** one full ask instead of two, killed by `TestABurstEndsBeforeTheHostsDeadline`.
- **MF is killed:** the stations known to hang counted in `NotReached`, killed by `TestHangingStationsNeverCallAHealthyGIRODown`.
- **Gaps in the tests themselves:**
  - `TestAKeptProbeKnownToHangIsAskedLast` (`stations_test.go:343-351`) is a single `Plan` call, so it cannot see R9T-2.
  - The longest timing test is 24 hours; R9T-2 appears at hour 158.
  - The retry test answers under 2 s, so it cannot see R9T-1.
- **Action:** once R9T-1 and R9T-2 are ruled, add:
  - an order case and a 35 s boundary case;
  - a slow-recoverer case with k = 1;
  - a scaled week of probe rotation with hanging dark stations at 1.2 s;
  - R9T-3's case at 40 s.

### 6. Host and library

These agree with section 7:
- **The deadline:** `domains/propagation/propagation.go:47` (60 s), `:86-87` (under `MinDeadline` refused) and `:224` (each update its own).
- **The cancel before the wait:** `app/mapprop.go:121-124`, then `app/dashboard.go:132` and `:1016`.
- **The burst asked at once after StationsNext:** `modes/tty/map_prop_update.go:180-185`.
- **The refresh tick:** `:200-209`.
- **The words:** `:292-310` and `:402-410`.
- **Batch 74:** the product bound is still held without race (`app/norace_test.go`).

One observation, not a finding: the 60 s starts at `propagation.go:224`, before go-ionomaps takes its lock (`update.go:116`). One update per session (`map_prop_update.go:52`) and a session's update unwinding at once on leave keep that wait negligible.

### 7. The model

**Kept by the code:**
- section 1, apart from the retry's 35 s (R9T-3);
- section 2: `asker.go:237,291-308,352-364`;
- section 4: `asker.go:319-333`;
- section 5: `update.go:146-168,317-323`;
- section 6: `stations.go:357-436`;
- section 7.

**Wrong, as whole recommendations above:**
- Section 3: R9T-1 (items 2 and 5) and R9T-2 (item 1 with D-303's keep).
- Section 8, limits 5 and 6: worse than stated (R9T-1, R9T-2).

## Verified clean

- **Bounded** at every timing I built: 800 swept hours, 20 s to 120 s, on both platforms.
- **M1:** a dead GIRO is called down every hour at 60 s.
- **The retry's other parts:** its rotation in turn, its budget place, its never being a dark station, and its run-out breaking both runs.
- **The tail's order,** and the stations known to hang left out of `NotReached`.
- **Every round 8 timing finding** is fixed as ruled.
- **Watchpost's side,** in full.
- **All gates named above.**

## Opinion

**Do not exit REVIEW.** Round 8's two new rules each starve something, by construction and silently, at the release's own ratios:
- M2's retry takes a lone station known to hang out of the tail, and leaves it unread at PLAN's 0.92 s (R9T-1).
- M3, beside D-303's keep, freezes the probe rotation for good once GIRO is past about 1.13 s an ask (R9T-2).

Both are model changes for the HUM LEAD, small in code.

**Fix first: R9T-2.** It is a one-line change that restores R-5.5, a requirement, indefinitely broken. Present it whole with R9T-1, since both refine D-314's M2 and M3.
