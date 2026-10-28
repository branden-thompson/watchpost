---
title: "0.19.0 REVIEW exit red team, round 10 - the light review, verbatim"
date: 2026-10-27
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - one fresh reviewer over go-ionomaps v0.1.0-rc.15 and watchpost batch 75 alone, findings weighed by what happens (D-320), the report as delivered; only the scratch folder is shortened to <scratch>; the dispositions are in ../red-team-review.md, Round 10."
---

# Round 10, the report

Up: [red-team review](../red-team-review.md) · [round 9](review-round-9.md)


---

**R10 report: the simpler GIRO station model (go-ionomaps `f60cc5c..5396d3f`, v0.1.0-rc.15; watchpost `61a326a4..fdb4d35b`, batch 75)**

I found no Importants and seven Minors. I worked read-only in clones under `<scratch>/rt10`, with mutants applied through `-overlay`. I confirmed that the module cache's rc.15 matches 5396d3f and that tag rc.15 points at it. Nothing touched the trees, the git state, the network or the measurement on :6072/:6073.

Test runs:
- **go-ionomaps:** all packages pass on arm64, once the ignored `.a2dh*` files were copied in. The root package, `giro`, `stations`, `throttle` and `deadline` also pass under `GOARCH=amd64`.
- **watchpost:** `modes/tty`, filtered to the station and snapshot tests, passes on arm64 and amd64.

### 1. Built as the model says?
Each of the seven rules matches the code:
- **Rule 1 (order):** `stations.go:145-177`. Stations not flagged come first, in rotation from `pollAt`. The flagged ones that are due follow, those heard before going first and then the one asked longest ago. One place is kept for them.
- **Rule 2 (a miss):** `asker.go:240-251, 261-275`.
- **Rule 3 (flag after 3, every 6 h):** `stations.go:125-135, 200-216`.
- **Rule 4 (GIRO's refusals):** `asker.go:211-225`.
- **Rules 5 and 6 (deadline, cancel):** `asker.go:170-186`.
- **Rule 7 (reporting counts):** `stations.go:59-69`, `update.go:549-558`.
- **Restarts:** `heard`, `poll`, `misses` and `asked` round-trip (`state.go`). An rc.14 file still restores, because its unused keys are passed over. The skew and bounds checks are in `stations.go:314-357`.

**R10-7** `internal/giro/asker.go:253-254` with `:283-288`. A 200 reply that has readings, but which `hold` refuses, still returns `data`. Two cases do this: the reply names another station's code, or a placed station's reply puts it more than 50 km away. Such a reply clears the station's misses and its flag, and is not counted in `Failed`, yet nothing is held. Neither case has been seen; I could only construct them. Severity: Minor. Action: state it as a limit, or treat a refused hold as a miss.

### 2. Bounded?
Clean.
- `Plan` never returns more than its budget, and `thr.Start` caps a burst at `min(want, MaxBurst=40)`.
- Every ask is whole or none: `deadline.Room(ctx, 15s, 15s)` gives 0 unless 16 s are left (`asker.go:171`).
- A 429 or a 401/403 stops the burst and pauses GIRO. A burst cancelled before its first ask is undone.
- `Answered()` resets the back-off only on a 200.
- Mutant I (shortened asks) was killed by `TestABurstEndsInsideItsDeadline`.

**R10-3** `06_docs/.../update-timing-model.md:19`, `internal/stations/stations.go:140-142`. "One place in the budget is kept for a flagged station due, so one can always return" is true of the plan, not of the burst.
- The flagged come last, and an ask starts only while 16 s are left (`asker.go:171`). So the kept place is reached only if the 39 asks before it finish by about 44 s.
- The PLAN evidence (`dry/week/requests.jsonl`, 7-day replies) shows a mean of 0.99 s an ask, which leaves about 5 s.
- Any hour in which one station not yet flagged takes its full 15 s skips the kept place.

That delays a flagged station's return; it does not starve it at measured speeds. Severity: Minor. Action: add it to the model's accepted limits.

### 3. Said truthfully?
I replayed the 96 hours of the PLAN week (39 stations) through D-319's rule as watchpost applies it, using held = a reading within 72 h. The notice would show in 0 of 96 hours, and the lowest share of recent readings was 92%. So there is no false banner in normal use. Watchpost's `thinStationsWords` (`map_prop_update.go:308-319`) applies D-319 correctly, says it first, and is silent with no readings held or with no stations by choice.

**R10-1** `06_docs/02_features/go-ionomaps/03-architecture-design/update-timing-model.md:25`. Rule 7 says `StationsRecent` counts "the readings used (`Inputs.Stations`)", and that the notice shows when "fewer than half the readings used" are recent. The code and the snapshot docs say otherwise:
- `StationsRecent` is counted over the readings **held**, which is `StationsHeld` (`stations.go:59-69`, `update.go:549-550`). The model never names `StationsHeld`.
- `Inputs.Stations` counts only readings up to an hour old (`TestOnlyReadingsAnHourOldAreAssimilated`).

A host that followed this page would compare `StationsRecent*2 < Stations` and would almost never show the "fewer than half" notice. Watchpost follows the godoc and uses `StationsHeld` correctly, and it is the only host, so this is unlikely to be met. Severity: Minor. Action: reword rule 7 to name `StationsHeld` as the base.

**R10-2** `internal/stations/stations.go:161-176`, `internal/giro/asker.go:250-251, 183-185`, watchpost `modes/tty/map_prop_update.go:420-425`. In normal operation, every hour's status says "1 station gave no data".
- The place kept for flagged stations is filled each hour by one of the 79 unplaced seed codes that have never reported.
- GIRO answers a code with no data as a 200 with no reading. The evidence is LL721's 537-byte reply in `fresh/week/requests.jsonl`.
- Since A-39(3) that reply counts as a miss. `TestAStationThatGivesNoDataIsAMiss` asserts exactly this ("it and the one flagged station"). rc.14 counted such a reply as answered.

It is built to rule 2 and it is true, but in real use it is shown every hour, so the words never go quiet. Severity: Minor. Action: the HUM LEAD's veto on A-99(3)'s wording, or a stated known limit.

### 4. Tests
Mutants that were killed:
- `pollAt` moving for flagged stations as well
- `state.go` dropping `asked`
- flagged stations not ordered heard-first
- flagged stations ordered newest-asked first
- the miss count not capped at 3 on `Apply`
- shortened asks
- the notice not said first (watchpost)

The existing m131–m143 cover the flag, the 6 hours, the kept place, the clearing and the state.

**R10-4** These boundaries are not pinned; each mutant survived:
- `stations.go:54`: `recentWithin` changed from 3 h to 2 h. `giro_update_test.go:523-538` tests only 0 h and 4 h.
- `stations.go:63`: "newest" changed to whichever held reading comes first. All the fake's readings share one time.
- watchpost `map_prop_update.go:318`: `*2 <` changed to `*2 <=`. Exactly half is untested.
- watchpost `map_prop_update.go:301`: `stationsThinAfter` changed to 2.5 h. The test uses only 2 h and 4 h.

Severity: Minor. Action: add an exactly-half case, cases at 2 h 59 m and 3 h 01 m, and readings at different times.

**R10-5** `state_test.go:114`. `TestTheStateHoldsNoReading`'s list of allowed keys still names removed keys (`seeded`, `probe`, `out`) and lacks `misses` and `asked`. It passes only because no station misses in its fixture. A state file from real use, after any miss, would fail it. Severity: Minor. Action: update the key list.

### 5. Docs
The architecture's failure table, FR-5.2, `CHANGELOG.md`'s known limits, A-99, R-5.3, R-5.5, D-121 and A-39 all say what the code does. The exceptions are R10-1 and R10-6.

**R10-6** These describe a miss count that is not quite the rule:
- go-ionomaps `README.md:26` and `update.go:106`: "misses three bursts running".
- watchpost `CHANGELOG.md`'s known limits: "three hours running".

The rule is three asks in a row. With more than 39 stations not flagged, or a burst the deadline stops, those asks need not fall in consecutive hours. Separately, `snapshot.go:143`'s `StationsFailed` comment lists only "a server error, a 404, no answer" and leaves out a run-out ask and a reply with no reading. Severity: Minor. Action: reword, in one batch.

**Verdict: exit REVIEW.** There are no Importants. The seven Minors (R10-1 to R10-7) can go in one batch: four are wording or a stated limit, two are test gaps, and R10-2 is a wording choice for the HUM LEAD to veto.
