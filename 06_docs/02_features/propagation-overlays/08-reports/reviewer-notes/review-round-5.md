---
title: "0.19.0 REVIEW exit red team, round 5 - the reviewers' reports, verbatim"
date: 2026-10-26
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - three fresh blind reviewers (D-294), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 5."
---

# Round 5, the reports

Up: [red-team review](../red-team-review.md) · [round 4](review-round-4.md)


---

## Docs Quality, round 5 (R5D): 0.19.0 REVIEW exit red team (D-294)

**Tree state.** I reproduced every finding against these states.

| Repository | At the start | At the end |
|---|---|---|
| watchpost `af661bf5` | ` M …/02-analysis/rulings.md` (the uncommitted D-294 row, not mine) | unchanged |
| go-ionomaps `3b0ab33` | clean | ` M 06_docs/gate-runs.md` |
| go-tuimaps `17ae3de` | clean | ` M 06_docs/gate-runs.md` |

The two `gate-runs.md` changes are the green rows the required `scripts/gate --quick` writes. I left them in place: the gate record keeps its runs.

**What I ran (all exit 0):**
- watchpost:
  - `make lint` (no new findings, 6 baselined), `make lint-authoring`, `make p10` (0 live / 0 unmatched / 0 unratified), `make dupes` (2 groups, both ratified), `make mutant-anchors` (824 match);
  - `go vet` on `domains/propagation`, `tools/perfsum` and `modes/tty`;
  - `go test -race` on `domains/propagation` and `tools/perfsum`, and on `modes/tty -run 'Stations|Burst|Prop'`;
  - the committed `perfsum -g1`, built into my scratch, run on `g1-run-2026-10-19`.
- go-ionomaps and go-tuimaps: `scripts/gate --quick`, both green with P10 run.
- Scratch programs in `<scratch>/rt5-r5d/`:
  - `rx/` probes D-288's new scan pattern;
  - `aliascount/` type-checks go-tuiMaps' struct aliases.
- `go doc -all` on both libraries' public packages.
- Every test named in go-ionomaps' requirements exists (126 of 126); so does every test named in watchpost's requirements and gate roster (164 of 164).

**Could not verify:** go-tuiMaps `a23484f` (PR #8) is not in the local clone, and I may not fetch.

### Part 1: round 4's findings (and round 3's PARTIALs)

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R4D-1 | FIXED | `(G5)` is gone. The pattern (go-ionomaps `docids_test.go:77`, the same in go-tuimaps) catches G5, W10, INST-5 and R4C-1. `go doc` for both libraries shows no ID. Its "general" claim is overstated (R5D-7). |
| R4D-2 (R3D-8) | PARTIAL | `docs/extending.md:20-25` and `README.md:204,210-213` now say the windows' keys are fixed. Both miss the bands chart window's own hard-coded keys (R5D-4). |
| R4D-3 | FIXED | `propagation.go:310` names the 410; `propagation_test.go:175-186`; A-92 (1) at `rulings.md:415`. Nit: `noteStatus`'s doc (`:312`) still says "D-247 and D-260", while `:305` says "D-247, D-260 and D-272". |
| R4D-4 (R3D-9) | FIXED | `g1.go:108-114,156`. My run prints 47.995 (18s short), 47.975 (1m30s) and 47.969 (1m51s), matching `problem-statement.md:105` ("by 9 seconds"). |
| R4D-5 | FIXED | go-ionomaps `update.go:133` and `asker.go:146-160`; watchpost `Makefile:108-110`. `synth_test.go:249,264,275` make three renders plus one retry, 8 minutes. |
| R4D-6 | FIXED, with a new error | The fix states the GIRO limits wrongly (R5D-2), and is silent on NOAA's "no room" back-off (R5D-1). |
| R4D-7 | FIXED | `README.md:268-269`, `CHANGELOG.md:26-27`, FR-4.12. Residue: the file also holds station codes and rotation positions (go-ionomaps `state.go:41-52`). These are harmless, but "times and the flux mean" does not list them. |
| R4D, smaller | FIXED | Batch 67's list now matches `git diff --name-status 03ad4e97..d63f499e`: ten re-pointed mutants. `gates.md:29` cites m544 and m571, and both exist. |

### 1. Does each document answer the question its title asks?

**R5D-5: the mandatory first read and the record's status stop before round 4.**
- `00-REQUIRED-READING.md` is re-read at every session start, but:
  - its "Where REVIEW stands" (`:44-51`) ends at round 3 and names rc.8 and rc.7 as the candidates;
  - its Red team row (`:36`) lists rounds 1 to 3.
- `go.mod:9-10` pins rc.9 and rc.8, and round 4 (D-290 to D-293, batch 68) is missing from the page.
- `red-team-review.md:7`'s status still says "its remediation follows", while its own "Round 4, closed" table (`:484`) closes every finding.
- Neither file changed in `fe828be1` or `af661bf5`.
- Severity: Minor. Simplify/Delete? N.
- Action: add round 4's line and the candidates; update the status.

**R5D-6: the architecture's "When things fail" table lacks the failures this release added.**
- The table is at `architecture.md:164-176`. It has no row for:
  - a station's own failure (D-247);
  - GIRO hanging or stopping answering (D-286, D-291);
  - GIRO denying this host for a day (D-247; the words at `map_prop_update.go:167`);
  - a stalled NOAA (D-285).
- Each of these is something the listener sees, so the section does not answer its own question for this release's failure modes.
- Severity: Minor. Simplify/Delete? N.
- Action: add four rows, each citing go-ionomaps R-5.3.

### 2. Does anything contradict the code, or another document?

**R5D-2: go-ionomaps' README states the per-ask limits wrongly for GIRO.**
- `README.md:24-25` says an ask that runs out "counts as no answer and backs off".
- That is false for GIRO. One run-out ask is that station's failure and the burst goes on (`internal/giro/asker.go:56`, R-5.3, and `TestTimeoutsAreFailuresAndARunOfThem`: "scattered timeouts do not" stop the burst). Only three in a row, or the deadline after asks ran out, call GIRO down.
- "By the library's clock" is also wrong. The limits run on wall time (`context.WithTimeout`, `update.go:317`), not on `Options.Clock`, which a replay sets to its own fake clock.
- Introduced by R4D-6's fix.
- Severity: Minor. Simplify/Delete? N.
- Action: "a NOAA ask that runs out stops that round and backs NOAA off; a GIRO ask that runs out is its station's failure, and three in a row call GIRO down"; "in real time".

**R5D-3: `Offset`'s new doc dates the seed week differently from the struct's own value.**
- `snapshot.go:224-225` (new in rc.9) says the seed is "a week of GIRO's readings in late September 2026".
- The same struct's `Basis` (`offset.go:18`) says "a week in early October 2026", as do `constants.go:38,40` and `ahead.go:64`.
- `constants.go:19-22` gives the actual week: 2026-09-29 to 10-05.
- Severity: Minor. Simplify/Delete? N.
- Action: "the week of 2026-09-29 to 10-05", the same words in both places.

**R5D-4: D-292's docs miss the bands chart window's fixed keys.**
- `map_prop_chart.go:475` closes the chart on a literal `esc`, and `:486-490` scrolls on literal `pgup`/`pgdown`.
- Yet `README.md:208,210-213` lists PgUp/PgDn among the commands `[keys]` can rebind, and names only `chart.prev`/`chart.next` for the chart.
- `extending.md:22-25`'s list of fixed windows leaves the chart out. `README.md:212` leaves out the acknowledgement window.
- A listener who rebinds `map.scroll.up` finds the chart window still on PgUp.
- Severity: Minor. Simplify/Delete? N.
- Action: name the chart's esc and page keys as fixed, and add the acknowledgement window to the README's list.

**Nits (batch them):**
- `internal/stations/stations.go:177`: "moves past past".
- `update.go:133`'s comment omits the no-room stop.
- `reach.go:35`'s path renders broken in `go doc` ("evidence/ p533-equations.md").
- go-tuiMaps' `go doc` now says "upstream" with nothing defining it (`view.go:11`, `facts.go:310-311`, `places.go:58,162`). Only `README.md:161` names TerminalMap.

### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R5D-1: a NOAA round cut short for lack of time backs NOAA off. No requirement or host-facing doc says so, and an A-row decided it.**
- `update.go:313-316`: when the host's deadline leaves under 20 s, the round ends and sets `trouble`, which backs NOAA off 10 minutes. `update_test.go:611-621` pins this.
- What the documents say:
  - R-5.3 says only that "no room … stops NOAA's round";
  - `Update`'s doc (`update.go:95-96`) still says a deadline means "no back-off";
  - the README is silent.
- So a host whose deadline is under 20 s never asks NOAA, backs off every time, and no document tells it the minimum.
- The README's example (`README.md:35-38`) reuses one `ctx` for NOAA's update and the burst, so they share one deadline.
- A-33 (1) chose this behaviour. D-13 limits A-rows to references, wording and names.
- It is also the opposite of D-291's GIRO rule: a burst cut short while GIRO answers backs nothing off.
- Severity: Minor (watchpost gives 60 s). Simplify/Delete? N.
- Action: state the back-off and the 20 s minimum in R-5.3, `Update`'s doc and the README, and give each `Update` its own deadline in the example. Recommendation to the HUM LEAD: rule whether a slow but answering NOAA should be backed off.

**R5D-7: "The pattern is general, so a new family fails rather than passes" is overstated.**
- The claim is in go-ionomaps `docids_test.go:71-76` (the same in go-tuimaps), A-33 (6) and A-21 (1).
- My probe shows the pattern misses ID families from this release's own documents:
  - O1 and O2;
  - M5b;
  - CQ-T1 and PF-F2, which are go-ionomaps R-5.1's own sources;
  - PS-G, N3, "batch 68", "UAT 112" and m544.
- None appears in `go doc` today; I scanned for them.
- Related: the build log says "seventeen" struct aliases. I confirmed 17 by type-checking, but `aliasdoc_test.go:63` fails only below 4.
- Severity: Minor. Simplify/Delete? N.
- Action: add these to the must-catch list, or drop "general". Fail when the aliases found fall below 17.

### 4. Is the audience order right: designers, then PMs, then engineers?
Nothing found. The required reading and both READMEs lead with what the listener, or the host developer, sees.

### 5. Is a number published without its blind spots beside it (INST-5)?

**R5D-8: the Broadcaster arm's "upper bound" is stated without its two blind spots.**
- `validate-agenda.md:21` says the figures for a444b000 "are an upper bound for the branch's tip".
- First blind spot: D-293's own counter-argument, that this is a reading of the diff and not a measurement, is not beside it.
- Second blind spot: evidence outside the tree, `g1b-run-2026-10-25/NOTE-outage.txt`, records about 2 h 40 m of this machine's network outage during that run. The note says its downloads "may read low", and asks for a "note beside the D-246 verdict".
- The tree carries neither. During that window the failure paths rounds 2 to 4 changed were acting, so for that window "upper bound" does not follow.
- Severity: Minor. Simplify/Delete? N.
- Action: put both caveats beside item 3 before its verdict is read.

### What I verified clean
- Batch 68's code changes match its docs:
  - the 410 named;
  - `propSession.burst`, whose test leaves the mode with a burst in flight and opens it again;
  - perfsum's shortfall column;
  - the Makefile's arithmetic;
  - FR-4.12;
  - batch 67's mutant list;
  - the gate roster's row 29.
- The as-built map's go-ionomaps node at rc.9.
- `verify-runs.md` and `gate-runs.md` keep their FAILED rows, as the build logs say.
- R-5.3 and D-104 match `asker.go`'s `roomForAnAsk` and `endsAtTheDeadline`.
- No IDs or process words remain in either library's `go doc`.
- All 126 + 164 named tests exist.
- G1's figures, recomputed with the committed scorer.
- D-293's build `a444b000` exists, and it is batch 64.

### Opinion
**Exit REVIEW.** In this lens nothing is Critical or Important: all eight findings are Minor, and seven of them are wording or currency. Under D-294 that leads to the exit report, with these batched before VALIDATE.

Fix first: **R5D-1.** It is the one place where behaviour, not wording, went undocumented. An A-row decided it, and it runs opposite to the GIRO rule the HUM LEAD ruled in D-291.

---

## REVIEW lens, round 5 (R5L): 0.19.0 REVIEW exit (D-294)

**Tree state.** At the start: watchpost `af661bf5` showed ` M .../02-analysis/rulings.md` (the uncommitted D-294 row, not mine), and go-ionomaps `3b0ab33` and go-tuiMaps `17ae3de` were clean. At the end, go-ionomaps and go-tuiMaps each showed ` M 06_docs/gate-runs.md`. Those rows are not mine: I ran both gates in clones, and their rows went to the clones. I edited no tree and changed no git state.

Every finding below was reproduced against those SHAs, in `<scratch>/rt5-r5l/`:
- `iono/` (an rsync copy, used for probes and mutants, each restored and compared with `cmp`) and `wp/` (an rsync copy of watchpost).
- `ionoc/` and `tuic/` are clones used for the gates.
- The probes are in `probes/`.

**What I ran:**
- **watchpost (tree):** `make lint` (no new findings, 6 baselined), `lint-authoring` OK, `p10` (0 live, 0 unmatched, 0 unratified), `dupes` (2 groups, both ratified), `mutant-anchors` (824, all match).
- **watchpost (`wp/`):** `go vet` clean on `domains/propagation`, `modes/tty` and `tools/perfsum`. `-race` green on `domains/propagation` and `tools/perfsum`, and on the Propagation tests in `modes/tty`.
- **go-ionomaps `scripts/gate --quick`:** green, P10 run.
- **go-tuiMaps `scripts/gate --quick`:** green, P10 run.
- **Mutants:**
  - Batch 68's m570 and m572 are both killed.
  - Of my own 11, 8 were killed and 3 survived (Ma, Mg, Me, below).
- **Pins:** watchpost's `go.sum:13-16` names rc.9 and rc.8. The tags resolve to `3b0ab33` and `17ae3de`, and the module cache's `asker.go`, `update.go` and `stations.go` are identical to the tree.
- **Not checked:** go-tuiMaps `main`'s `a23484f`. It is not in the local repository (`origin/main` is `50350d1`), and I may not fetch.

### Part 1: round 4's findings

| Finding | Verdict | Evidence checked in code |
|---|---|---|
| R4L-1 | **FIXED** (with a new defect: R5L-1) | `asker.go:197-199,219-240`. My hang-position sweep at the real ratio (15 s ask, 60 s deadline, 0.9 s answers, scaled) covered k = 0 to 38 answers before the hang. Every case was Unavailable, paused, and ended inside the deadline. Alternating hangs and 5xx are called down. Mutant Mh (drop `last != timedOut`) is killed. |
| R4L-2 | **FIXED** | `map_prop_update.go:44,132`: the flag lives on `propSession`, which `stopPropagation` resets. m572 is killed by `TestAStoppedBurstNamesGIRO`. |
| R4L-3 | **FIXED** | `isTimeout` (`update.go:589-592`, `asker.go:400-406`) and its use at `update.go:316` and `asker.go:305`. Mi and Mj (drop `isTimeout`) are killed. NOAA's 20 s is pinned (`update_test.go:598`). Raising GIRO's askTimeout to 29 s is killed by `TestAHungGIROIsCalledDown`. Through watchpost's client, `redactErr` keeps the `%w` chain (`httpx.go:951-955`), so the client's `timeoutError` stays visible. |
| R4L-4 | **FIXED** | `propagation.go:310`: 410 is named. m570 is killed. |
| R4L-5 | **FIXED** | `docids_test.go:17` in both libraries. `go doc -all` shows no IDs. A planted `(G5, P-6)` in `Update`'s doc is caught (Mk killed). |
| R4L-6 | **FIXED** | `state_test.go:191-200`. Round 4's M9 (`state.go:125` disabled) is now killed. |
| R4L-7 | **RULED** (D-293) | `validate-agenda.md:21` names `a444b000` as an upper bound. |
| R3L-2, R3L-3, R2L-1 (PARTIAL in round 4) | **FIXED** for hangs | Covered by the same sweep and by `TestStationsTimingOutGoBehindTheRest`. The exception is R5L-1. |

### 1. Happy Path Bias

**R5L-1. One station whose ask hangs calls GIRO "unavailable" every hour, for good, while every other station answers.**
- **Severity:** Important (a recommendation against D-291's wording).
- **Simplify/Delete?** N.
- **The mechanism:**
  - `endsAtTheDeadline` calls GIRO down whenever the last ask before the deadline timed out (`go-ionomaps internal/giro/asker.go:233-239`).
  - A hang that starts after about 30 s of a burst always leaves less than 15 s, so it is always the last ask.
  - It then returns `lastFailed`, and the rotation's head moves past that station (`internal/stations/stations.go:188`), so the station is asked 39th every hour after.
- **Probe P6** (`internal/giro`, real ratio, the station hanging from 36th place):
  - Hour 0: asked 36, Unavailable.
  - Hours 1 to 9: asked 39, the hanging station 39th, **Unavailable 10 of 10 hours**.
  - The same station placed 5th or 20th: never called down.
- **Probe P7** (top level, `Update`): hours 0 to 3 each give Reason `GIROPaused`, GIRO unavailable, 38 stations, 1 failed.
  - watchpost renders that as "GIRO unavailable this hour: the readings it gave first · 38 stations … · 1 station did not answer" (`modes/tty/map_prop_update.go:296,370-375`).
  - That label says more than happened, which D-239 forbids.
- **Why the ruling missed it:** D-291's counter-argument names only *three* hanging stations; one is enough.
- **Dropping the head move does not fix it:** with Ma applied, P6 still calls GIRO down 10 of 10 hours, because the station stays late in the order.
- **Action, a fork for the HUM LEAD:**
  - Ask a station that timed out *first* next hour, where the three-in-a-row rule (D-286) can judge it; or
  - At the deadline, call GIRO down only on two or more timeouts in a row.
  - Either way, test one hanging station over 8 hours.

### 2. Systemic Failure Modes

**R5L-2. The per-ask limits now silently need the host's deadline to be at least 20 s for NOAA and 15 s for GIRO. Under that, nothing is asked, and the library says something untrue.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:** `go-ionomaps update.go:310-313` and `internal/giro/asker.go:197-199`.
- **NOAA, probe on watchpost's `Service` with `Deadline` 50 ms and then 19 s:**
  - 0 sources asked in 3.8 µs.
  - Reason `Offline`, with the error "NOAA's solar data could not be read … asked again at the first update after 12:10".
  - The next probe shows the back-off growing to 15 minutes while NOAA was never asked.
- **GIRO, probe P1** (no room at the first ask):
  - 0 asked, Outcome Answered.
  - `BurstDue` is false for the hour: the hour is spent. That contradicts `Burst`'s doc, "a burst whose context ended before its first ask spends none" (`asker.go:155-156`).
- **What documents the limit:** nothing.
  - The README (`README.md:24-26`) does not give a minimum deadline.
  - watchpost's `Options.Deadline` (`domains/propagation/propagation.go:39,104-106`) accepts any positive value.
  - Its comment ties 60 s only to the 36 s burst (`:44-46`).
- **Severity reasoning:** watchpost's default 60 s is fresh for each `Ask`, so no production path is affected.
- **Action:**
  - State the minimum deadline in the README and in `Update`'s doc.
  - When no ask fits, say so: don't call it "could not be read", don't back off, and don't spend GIRO's hour.
  - Or refuse a `Deadline` under 20 s in `NewService`.

### 3. Gate Validity

**R5L-3. Three of D-291's rules pass the suite with their code removed.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Ma survives:** `endsAtTheDeadline` returning `""` instead of `lastFailed` (`asker.go:239`). The head move at the deadline is untested, and it is the mechanism in R5L-1.
- **Mg survives:** `timedOutAll < failuresInARow` changed to `< 100` (`asker.go:234`).
  - "Three of its asks did" (A-33 (2)) has no test.
  - The "hangs alternating with a server error" subtest (`asker_test.go:349`) passes through the `last` branch instead.
- **Me survives:** dropping `l.trouble = true` on NOAA's no-room path (`update.go:311`).
  - A-33 (1), "backs NOAA off as an unanswered ask does", is not held.
  - The test's "not asked again within ten minutes" (`update_test.go:620`) cannot tell a back-off from the 10-minute base.
- **Action:** add a test that kills each one. For Mg: three scattered timeouts with an answered last ask. For Me: assert the 15-minute second step.

### 4. Adversarial Input

Nothing new beyond R5L-1 and R5L-2.
- A GIRO that answers each ask in just under 15 s is cut at the deadline: about 3 asks, not backed off. That is D-291's ruled "otherwise" case.
- How it is worded is R5L-4.
- A host error that only claims `Timeout()` is trusted host code. Pre-ask cancels are checked first (`asker.go:302-304`, `update.go:316`).

### 5. Graceful Degradation

**R5L-4. A burst cut at the deadline while GIRO was answering is now said as Fresh, with nothing about the stations not asked.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **What changed:** before batch 68, this exact case ran into the deadline and was said "GIRO's burst stopped before every station was asked" (D-286, A-92 (3)). Now the burst ends before the deadline, so `ctx.Err()` is nil and the switch at `go-ionomaps update.go:114-122` falls through to Fresh.
- **Probe P4:** GIRO answering slowly gives 20 of 39 asked, Reason Fresh, GIRO answered, 0 failed.
- **Probe P6, one hanging station early in the order:** every hour is cut at 30 to 32 of 39. Nothing says so except "1 station did not answer".
- **Inferred, not probed in watchpost:** D-286's watchpost wording (`map_prop_update.go:376-379`) and the `propSession.burst` flag are now reachable only through a cancel. A cancel's result is dropped by the session check (`:139`), so in practice the wording is dead.
- **The rule it misses:** FR-5.2 says partial data is "said as such".
- **Action, a fork for the HUM LEAD:**
  - The library reports the stations not asked (for example `Inputs.StationsUnasked`), and watchpost says "GIRO's burst cut short: N stations asked next hour"; or
  - The HUM LEAD accepts that the count and the oldest reading's time say enough.
  - Either way, retire the dead D-286 wording.

### 6. Test Confidence vs. Test Results

**R5L-5. watchpost's `TestAnUpdateHasItsOwnDeadline` now passes without asking any source.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Evidence:** `domains/propagation/propagation_test.go:232-247`.
  - Its 50 ms deadline is under NOAA's 20 s, so the library starts no ask and returns at once. My probe: 0 asked, 3.8 µs.
  - The test's name and doc ("an update whose sources never answer ends at its deadline") no longer describe what it checks.
- **Action:** give the test a deadline above the per-ask limit (or a scaled library), and assert that a source was asked and that the update ended before the deadline.

Otherwise, round 4's new tests prove behaviour: hang positions, alternation, `isTimeout` and the both-parts restore. The gaps are R5L-3's.

### 7. Dependency Fragility

The coupling between the host's deadline and the library's per-ask limits is now explicit in the library but unenforced on either side (R5L-2). Watchpost's client timeout no longer matters, because of `isTimeout`, which works through watchpost's `ReachError` and `redactErr` chain. The pins match the reviewed commits. `a23484f`'s untagged v0.2.2 requirement on go-tuiMaps `main` could not be checked locally.

### 8. Problem statement present, real, honored

PS-1 and PS-2 are present (`problem-statement.md:14-31`). PS-1 is still a hypothesis (D-217). PS-2's M1 and M4 still go to VALIDATE. Item 3 now names its build (D-293), and batch 68 changes cost only when a source is slow or failing, so D-293's upper bound holds. R5L-1 bears on PS-2: an Observer is told GIRO is down every hour while 38 of 39 stations feed the map.

### Verified clean

- R4L-1 to R4L-6 in code, R4L-7 as ruled.
- D-291 for every hang position, for alternating hangs and 5xx, and for three hanging stations.
- `isTimeout` on both sides, and NOAA's and GIRO's limits pinned.
- 410 named; the burst flag tied to its session; both state parts refused and tested.
- The doc ID scan in both libraries, with a planted ID caught.
- Pins and module-cache identity.
- watchpost's lint, lint-authoring, p10, dupes and mutant-anchors legs; vet and race on the touched packages; both quick gates.

### Opinion

**Do not exit REVIEW.** Nothing is Critical, but R5L-1 is a new Important defect introduced by round 4's fix: one slow station host, a realistic single fault, makes the map say "GIRO unavailable" every hour. That is the mislabelling D-239 rules out. Its fix is small and fits the existing logic (ask a timed-out station first next hour), and it needs a test that runs 8 hours. R5L-2 to R5L-5 are Minor and can be batched with it.

**Fix first: R5L-1.**

---

## Code Quality, round 5 (R5C): 0.19.0 REVIEW exit red team (D-294)

**Verdict: do not exit REVIEW.** There is one Important, R5C-1, and it comes from round 4's own fix. D-291's room rule keeps a full 15 s free before every GIRO ask, so a burst now has 45 s of the 60 s update instead of 60. One hanging station, which used to fit (38 answers plus one 15 s hang is about 50 s), now leaves about 9 of the 40 stations unasked every hour. The library reports that burst as `Fresh` and `Answered`, and nothing in the status says it was cut short. The words D-286 wrote for a stopped burst can now be reached only through a cancel, and watchpost drops a cancelled update's result. Everything else is Minor.

### Tree state and runs
- **`git status --short` at the start:**
  - watchpost `af661bf5`: ` M .../02-analysis/rulings.md`
  - go-ionomaps `3b0ab33`: clean
  - go-tuiMaps `17ae3de`: clean
- **Changes during my review that are not mine:** go-ionomaps and go-tuiMaps each gained ` M 06_docs/gate-runs.md`, rows 13:24:51Z and 13:33:37Z (another reviewer). My gate rows are only in my byte copies under `<scratch>/rt5-r5c/copies`.
- **I edited no tree.** Every mutant and probe was an `-overlay` from scratch, and every finding was reproduced against the tips above.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes`, `mutant-anchors` (824), `go vet ./...`.
  - `go test -race` on modes/tty, domains/propagation and tools/perfsum.
- **go-ionomaps:** `scripts/gate --quick` on the copy is green, P10 included (13:28:33Z, tree `72f8ce321657`).
- **go-tuiMaps:** `scripts/gate --quick` on the copy **FAILED**, 13:36:55Z, tree `153c740ec7a8`: `TestFixtureMemory` in the race-on leg (R5C-7). The other reviewer's run on the same tree was green.

### Part 1: round 4's findings, checked in the code

| ID | Status | Evidence |
|---|---|---|
| R4C-1 (and R3C-1) | **FIXED** for the failure it named | go-ionomaps `internal/giro/asker.go:197-200,217-240`; test `asker_test.go:317` ("a hang after 20 answers"). My scaled probe (one hang, 12 hours) stayed inside the deadline every hour. The fix's new surface is R5C-1. |
| R4C-2 (and R3C-4) | FIXED | `snapshot.go:11` stripped. The pattern is general in both `docids_test.go:15`. My own looser scan of `go doc -all` found no IDs in either library. The new break in `reach.go:35` is under R5C-5. |
| R4C-3 | FIXED | watchpost `map_prop_update.go:44,132,152,177,181`. I applied m572 by overlay: killed. |
| R4C-4 | FIXED, with new residue | `snapshot.go:154`, `asker.go:29-35,47,146-159`, `update.go:133`, `state.go:20`. The new wording is not true under D-291: R5C-5. |
| R4C-5 | FIXED | `update.go:131-140,301-307,339-341,391-392,417-420,438-441`. |
| R4C-6 | FIXED | `Makefile:108-110` against `synth_test.go:249,257,262,275`. |
| R4C-7 | FIXED | `stations.go:183-190`, `asker.go:242-250`. The doc's "moves past past" is under R5C-5. |
| R4C-8 | FIXED | i3 is m083, i4 m078, i5 m077, i8 m082; i2 is covered by the four-dead loop (`asker_test.go:162`). The new code has new survivors: R5C-3. |
| R4C-9 | FIXED | `aliasdoc_test.go` finds the aliases by type-checking. Its `strings.Contains` check is weak for one-letter field names, but nothing escapes today. |

### 1. Understandability

**R5C-1 (Important), go-ionomaps, with watchpost: D-291 cuts healthy bursts short and says nothing.**
- **What the code does.** `roomForAnAsk` refuses an ask unless a full `askTimeout` is left (`asker.go:219-225`). An ask takes about 0.92 s (PLAN), so the burst's real budget is 45 s.
- **What a cut-short burst reports.**
  - `endsAtTheDeadline` returns `""` (`asker.go:234-236`), so the Outcome stays `Answered`.
  - `Update`'s burst switch sets `Unfinished` only when `ctx.Err() != nil` (`update.go:117-124`). A burst that ends early still has a live context, so the snapshot is `Fresh`.
  - The library's own test asserts this: `asker_test.go:364` ("slow answers… want the burst cut short, Answered").
- **Probe** (overlay test in `internal/giro`, time scaled 100×: answers 9.2 ms, `askTimeout` 150 ms, deadline 600 ms):
  - One station hangs. Every hour for 12 hours, 31 of 40 are asked, the Outcome is `Answered`, 1 is failed, and the hanging station is never moved behind the rest (it sits at position 27 every hour).
  - Before D-291 the same burst finished in about 50 s.
  - A GIRO answering steadily at about 1.2 s or more an ask is now cut every hour, also silently.
- **watchpost's words.**
  - D-286's "GIRO's burst stopped before every station was asked" (`map_prop_update.go:378-380`) and the deadline words (`:239-240`) need an `Unfinished` snapshot or a `DeadlineExceeded` error.
  - The rule refuses any ask without a full ask's time left, so those come now only from a cancel, and the only cancel is leaving the mode (`:128-133`), whose result is dropped by `propGen`.
  - The one exception is a build that crosses the last few milliseconds.
  - So the words, R4C-3's fix and m568/m572 guard a path production no longer takes. The path D-291 did create, the cut-short burst, is said "Fresh".
- Simplify/Delete: N.
- **Fork for the HUM LEAD:**
  1. Give the asks near the end a shorter timeout (`min(askTimeout, time left minus a margin)`) instead of refusing them, so the room rule stops costing a healthy burst a quarter of its time.
  2. Report "stations not reached" in the snapshot (a count or an Outcome) and say it in watchpost.
  3. Keep the behaviour, record it as a known limit, and retire the dead words and tests.

  Under 1 or 2, add a library-level test of a cut-short burst's snapshot.

### 2. Maintainability

**R5C-4 (Minor), go-ionomaps: the same rule is written twice.**
- `isTimeout` appears word for word in `update.go:590` and `internal/giro/asker.go:404`.
- The room rule (deadline minus the ask's limit) is `roomForAnAsk` (`asker.go:219`) and again inline in `ask` (`update.go:311`).
- A third timeout source would mean two edits.
- Simplify/Delete: Y. Action: one internal helper for each, e.g. `room(ctx, need)` and `isTimeout`.

### 3. Complexity

- **R5C-4, cont.: `Burst` now carries six pieces of per-ask state** (`asker.go:186-209`): `inARow`, `timeouts`, `timedOutAll`, `anyAnswered`, `last`, `lastFailed`.
  - They are updated in three shapes: `count` is a method with a side effect on `a.Failed`, `timeoutsAfter` is a free function, and `timedOutAll` is counted inline.
  - Round 4 flagged two shapes; there are now three.
  - Simplify/Delete: Y. Action: a small tally type with one `add(res)`.
- Nothing is shaped by a metric in the libraries. watchpost's build log says G1's shortfall is computed inline "which held `tools/perfsum`'s density at its floor" (batch 68). That is three lines, harmless, but the choice was driven by the metric; noted, not a finding.

### 4. Necessity

- Covered under R5C-1: watchpost's `Unfinished` burst words and deadline words, `propSess.burst`, `TestAStoppedBurstNamesGIRO`, `TestADeadlineIsSaidInWords`, m568 and m572 now serve paths production does not take.
- Nothing else is dead.

### 5. Name semanticism

Folded into **R5C-5 (Minor):**
- `endsAtTheDeadline` ends the burst *before* the deadline.
- `failuresInARow` is the threshold for `timedOutAll`, a count that is not "in a row" (`asker.go:234`).
- `Unasked`'s parameter `past` makes its doc read "the rotation's head moves past past" (`stations.go:177`).

### 6. Code documentation

**R5C-5 (Minor), go-ionomaps:**
- **Exported docs that are false under D-291:**
  - `GIROUnavailable` (`snapshot.go:154`) and `Unavailable` (`asker.go:47`) say "the burst reached the host's deadline after its asks ran out". By D-291's design the burst never reaches it.
  - `Update` (`update.go:95-96`) and `Unfinished` (`snapshot.go:181`) still say a deadline stops an update, which is then said Unfinished with no back-off. A round the deadline cuts short is now `Partial`, `Offline` or `Fresh`, and NOAA's side of it *is* backed off.
- **Other wording:**
  - `ask`'s doc (`update.go:305`) says NOAA "left no room"; it was the host's update that did.
  - `reach.go:35-36` wraps a path mid-name, and `go doc` shows "evidence/ p533-equations.md".
- **Short deadlines are a trap for a library user, and the error blames NOAA.**
  - Probe: an `Update` with 15 s left makes 0 asks. It is `Offline`, NOAA is backed off, and the error reads "NOAA's solar data could not be read … asked again after 12:10".
  - A burst call with 10 s left makes 0 GIRO asks, says `Fresh`, and spends the hour; the next burst comes 61 minutes later. The throttle is not restored at `i == 0`, which contradicts `Burst`'s doc (`asker.go:155-156`: "a burst whose context ended before its first ask spends none").
  - watchpost's 60 s default never hits this, but `propagation.Options.Deadline` is settable.
- Simplify/Delete: N. Action:
  - Correct the docs.
  - Restore the throttle when no ask was made.
  - Name the deadline in the error.
  - Say the minimum deadline in the README.

No history comments (AP-HIST-01) were found in this range.

### 7. Testability

**R5C-3 (Minor), go-ionomaps: survivors in round 4's new code.** Each mutant below was applied by overlay and run against the package whole (x2 to x4 against the module); all survived:
- **x1** (`update.go:311`, drop `l.trouble = true` in the no-room path). This is A-33 (1), "a NOAA round left no room backs NOAA off", which is untested. `TestNOAAsRoundEndsInsideTheHostsDeadline`'s last check (`update_test.go:617`, no ask within 10 min − 1 s) passes on NOAA's normal 10-minute spacing alone.
- **x2** (`asker.go:234`, drop the `timedOutAll < failuresInARow` clause). This is A-33 (2), "three of its asks did", which is untested. The "hangs alternating with a server error" subtest (`asker_test.go:349`) is decided by the last ask having timed out, not by the count its doc names.
- **x3** (`asker.go:239`, `return ""`, so a call-down at the deadline moves no head): unpinned.
- **x4** (`lastFailed` set on timeouts only): unpinned.
- Simplify/Delete: N. Action: a test and a ledger row for each.

watchpost's m572, which I applied myself, is killed.

### 8. P10 conformance

- `make p10` is 0/0/0, and the go-ionomaps quick gate's P10 leg is green.
- `Burst` is 54 lines, under the limit. No exemption was self-issued.
- No floor changed in this range. Nothing found.

### 9. Evidence soundness

**R5C-6 (Minor), watchpost: `TestAnUpdateHasItsOwnDeadline` (`domains/propagation/propagation_test.go:232-246`) no longer reaches a source.**
- Its 50 ms deadline is under NOAA's 20 s room, so the library asks nothing.
- My probe: 0 asks, returns in 84 µs, `Offline`.
- Its doc says "an update whose sources never answer ends at its deadline". Removing `WithTimeout` is still killed, but the hung-source path is not exercised.
- Simplify/Delete: N. Action: reword it, or assert asks > 0 under a library-settable timeout.

Also:
- The `x1` check above proves the normal spacing, not a back-off.
- D-291 says it was "tested with realistic answer times". The tests use a 1:20 answer-to-timeout ratio against PLAN's 1:16, which is close enough.

**R5C-7 (Minor), go-tuiMaps, whole release, against D-101: NFR-3's memory gate fails under host load.**
- `memory_test.go:164`: the median of five peaks was 8.42 MB against 8 MB (peaks 7.94 to 8.50) in the race-on leg, while my other test runs shared the host. The other reviewer's run on the same tree was green.
- `propagation-fields` build-log line 149 records the same failure "while the agent built watchpost beside it".
- D-101's median of five does not absorb load: the same tree's verdict depends on the host.
- Recommendation for the HUM LEAD: gate the peak only in the race-off leg, or run it alone, or invoke D-101's second step.

### 10. Evasion lens

- R5C-1: `TestAStoppedBurstNamesGIRO` and m568/m572 meet D-286 in letter by feeding a hand-made `Unfinished` snapshot. The library no longer produces one for a burst, and the real cut-short burst says `Fresh`.
- R5C-3: the "alternating" subtest names a rule that another clause decides.
- Nothing else found.

### What I verified clean
- **go-ionomaps:**
  - `isTimeout` uses `errors.As` against the `Timeout()` interface; m076 and m080 hold.
  - The `endsRound` rename is behaviour-neutral.
  - `Unasked`'s single-code head move matches its arithmetic.
  - `movedPast` returns none for 429, denial and cancel.
  - The both-parts `ErrBadState` (m082).
  - The ID strips are doc-only.
- **go-tuiMaps:**
  - The alias test finds every exported struct alias, and there are no grouped type declarations to fool it.
  - The doc strips are doc-only.
- **watchpost:**
  - The 410 is named (m570).
  - `perfsum`'s shortfall column (m571).
  - The session-scoped burst flag.
  - The re-pin to rc.9 and rc.8.

### Opinion
**Do not exit REVIEW.** **Fix first: R5C-1.** Round 4's fix to the GIRO timing bug now costs a healthy burst stations every hour and hides it, and the status words written for exactly this case can no longer be reached. This is the third round running whose Important sits in GIRO's per-ask timing against the host's 60 s. That supports the "timing lens" option D-294 weighed and did not choose.

Scratch directory, with the probes, the mutants x1 to x4 and w572, and the gate logs: `<scratch>/rt5-r5c`
