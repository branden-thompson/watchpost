---
title: "0.19.0 REVIEW exit red team, round 8 - the reviewers' reports, verbatim"
date: 2026-10-27
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - four fresh blind reviewers, each also on amd64, each judging timing against the update's timing model (D-313), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 8."
---

# Round 8, the reports

Up: [red-team review](../red-team-review.md) · [round 7](review-round-7.md)


---

## Docs Quality, round 8 (R8D): 0.19.0 REVIEW exit red team (D-313)

**Tree state.** All three trees were clean at the start: watchpost `49c65df4`, go-ionomaps `59cb4a9` and go-tuimaps `0c27321`. At the end, watchpost was unchanged. Each library showed ` M 06_docs/gate-runs.md`, holding my green `--quick` row (go-ionomaps 16:10:11Z, go-tuimaps 16:18:41Z). I reproduced every finding against the start state. Probes and mutants ran only on a copy, `<scratch>/rt8-r8d/iono`, never on a tree.

**What I ran:**
- **watchpost:** `make lint` (no new findings, 6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes` and `mutant-anchors` (827 match). All exited 0. `go vet` was clean.
- **watchpost tests:** `go test -race` on `domains/propagation` (whole) and on `app -run Prop|Settle|Update|AppsEnd|AsBuilt`. The same ran under `GOARCH=amd64`, all green. `TestTheAppsEndStopsAnUpdateAndKeepsItsState` ran ×3 under amd64: 0.57 to 0.67 s.
- **go-ionomaps:** `go vet ./...`, and `-race` on `giro`, `stations` and `deadline`. Under amd64: those three plus `limits`. All green.
- **Quick gates:** green in both libraries.
- **Requirements:** every test named in the requirements exists, in both watchpost and go-ionomaps.

### Part 1: round 7's findings

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R7D-1 | FIXED | `stations.go:255` skips stations known to hang in `leftOut`, and `:265-269` puts them last. My trace with 4 hanging stations (scaled 1:100, 12 hours, 9 and 11 ms pace) never asked one first after hour 0. The docs now say it: `asker.go:176-180`, `stations.go:59,244-247` and R-5.3. Above 39 live there is a new defect: R8D-1. |
| R7D-2 | FIXED | `asker.go:225-229` turns a repeat run-out into `hungAgain`, and `tally.add` (`:283-285`) counts it toward no run. My traces show `Outcome 0` in every hour after the first. |
| R7D-3 | FIXED | `snapshot.go:162`, `asker.go:99` and `update.go:104-106` were reworded. `snapshot.go:162` still leaves one end out: see R8D-3. |
| R7D-4 | FIXED | watchpost `architecture.md:93,114`. |
| R7D-5 | PARTIAL | `TestHangingStationsNeverCallAHealthyGIRODown` exists, but it does not check its "every answering station read" claim: see R8D-2. |
| R7D-6 | PARTIAL | `validate-agenda.md:25` and the go-ionomaps README (`:38-46`) list all five of section 8's limits. watchpost `CHANGELOG.md:32-37` lists four: "GIRO dying just as a burst reaches the stations known to hang" is missing, although D-312 and the model's section 8 say each limit goes in the changelog. |

### 1. Does each document answer the question its title asks?

**R8D-3: the model's section 4, "How a burst ends, and what it counts", leaves out one end: a burst stopped by a run of failures or timeouts.**
- What the code does on that end:
  - `stops` calls GIRO down mid-burst (`asker.go:339-342`, D-271/D-286);
  - `NotReached` stays 0;
  - `Unasked(…, movedPast)` puts the rest first and moves the head (`asker.go:233-235`).
- The table (`update-timing-model.md:87-93`) has rows for the deadline, a cancel, a 429/401/403 and no room. A run's stop is none of them.
- `snapshot.go:162` repeats the gap: it says the count is none for "a cancel, or GIRO refused with a 429 or a denial" and leaves out a run's call-down.
- Kind: (a), the model and the field doc are both incomplete against the code. Severity: Minor. Simplify/Delete? N.
- Action: add the row to section 4 ("a run of failures or timeouts | called down | spent | none | first, the head past a failed station") and add the same words to `snapshot.go:162`.

### 2. Does anything contradict the code, or another document?

**R8D-1: with 40 or more live stations, a station known to hang is never asked again. It goes unread for 3 days, then waits on the probe rotation. This breaks principle 3 ("no station is starved"), section 8's "read later in its hour" and R-5.5's "those left out asked first next time".**
- **Where it happens:**
  - `Plan` takes `order[:39]` (`stations.go:171-174`), and `rotation` puts hangs after every other station (`:255`, `:261`, `:265`);
  - with 39 or more non-hanging stations live, the hang is always cut by the budget;
  - next hour, `rotation` skips it in `leftOut` again (`:255`).
- **Probe** `TestR8DHangAboveForty` (`<scratch>/iono/internal/stations/zz_r8d_probe_test.go`): 41 live (2 probes heard), one station marked as hanging. Result: **polled in 0 of 72 hours**. It then goes dark (`darkAfter`) and waits for the probe rotation over about 83 dark codes. Same result on arm64 and amd64.
- **What is said:** nothing. The station is not in `NotReached` (budget-cut) and not in `Failed` (never asked).
- **How likely:** the seed has 39 live, so any one probe that answers makes 40 (`TestAboveFortyStationsRotate` builds exactly that).
- **Introduced by round 7** (D-309, A-37 (1)). Under rc.12 a budget-cut hang was `leftOut`, asked first next hour.
- **No test reaches it:** the new test runs on the seed's 39 live.
- **Kind (b), the model is wrong.** Section 3's order, with D-87's budget, starves by construction.
  - It touches principle 3, section 3 (item 4, and item 2's "did not reach", which does not say whether the budget's cut counts), section 4 (budget cuts are uncounted and unsaid), section 8 (the "recovers" limit) and R-5.5.
  - It qualifies D-309's "stays last".
- **Fork for the HUM LEAD, put whole:**
  - (i) a hang cut by the budget is asked within a bound, e.g. the oldest-hung takes the 39th slot every N hours;
  - (ii) accept it, and section 8 states its real length (up to 3 days live plus the probe rotation, with 40 or more live).
- Severity: **Important**. Simplify/Delete? N.

**R8D-4: section 2's evidence table says a counted timeout feeds only "the timeouts run" and that "any other end breaks a run". The code also counts it toward D-271's failures-in-a-row run.**
- `tally.add` (`asker.go:280-282`) increments `inARow` for `timedOut`.
- **Probe** `TestR8DMixedRunBeforeAnyAnswer`: the sequence 5xx, hang, 5xx before any answer gives `Outcome 1`, paused, after 3 asks. That is the same on arm64 and amd64.
- Read literally, the model would not call GIRO down here: the hang breaks the failures run.
- The code matches R-5.3 ("no answer" is a failure), and D-312's own words imply it ("counts toward no GIRO-down run, D-271's included").
- **Kind (b), the model's table is wrong.** It touches section 2 only: the row "A full ask run out, the station not known to hang" should add "and, before any answer, the failures-in-a-row run (D-271)". It supersedes nothing.
- The risk is real: section 9 says code that breaks the model is "fixed without a fork", so this row invites a wrong code change to D-271.
- Severity: Minor. Simplify/Delete? N.

**R8D-5: the as-built map was restamped for rc.13 and batch 72, but its nodes describe older code.**
- Node `IO` (`observer-maps/…/as-built-map.md:112`) still reads "three in a row calling GIRO down … two full ones … a station that ran out asked last next hour (… D-301 to D-307)". It has no D-309/D-310: nothing on stations known to hang being skipped or counting toward no run.
- Node `L` reads "go-tuiMaps v0.3.0-rc.5", against `go.mod:10`'s rc.10 and the page's own status line. This predates round 7 and is not excused by that.
- `TestTheAsBuiltMapIsKeptInStep` (`app/maps_test.go:55-81`) checks only the batch number, so the stamp passes while the text is stale.
- Severity: Minor. Simplify/Delete? N.
- Action: redraw `IO` and `L`, and regenerate the atlas (`architecture-atlas.html:904`).

**R8D-6: smaller drift from the model and from A-96, batched.**
- go-ionomaps `README.md:33-34`: "the live stations a burst did not reach … counted … and asked first next hour". It does not exclude the probe and the stations known to hang (section 4, D-309). The build log (`:59`) claims the README "says its words".
- A-96 (3) says watchpost links the model from "VALIDATE's item 7". `validate-agenda.md:25` names the model without a link.
- The two new watchpost links (`00-REQUIRED-READING.md:53`, `architecture.md:114`) are the tree's only `../go-ionomaps/` relative links. They resolve in this machine's sibling layout but break in watchpost's repository on GitHub. Elsewhere the docs cite go-ionomaps by repository and path.
- The P10 row for `internal/stations` (`p10-ledger.md:28` in go-ionomaps) says "renewed under the … model's section 10 (watchpost D-312, D-311 …)". It omits what A-37 (3) says: the agent made this renewal, and it is put to the HUM LEAD's veto. Other rows do say "as A-n, batched for the HUM LEAD's veto". As written, it reads as ratified.
- Severity: Minor. Simplify/Delete? N. Action: reword or link each one.

### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R8D-2: `TestHangingStationsNeverCallAHealthyGIRODown` does not hold "every station that answers is read each hour". D-309's ruling, the build log (`go-ionomaps build-log.md:60`) and the record (`red-team-review.md:691`, R7D-5 closed) all rest on it.**
- Its check (`asker_test.go:~810-830`) compares `liveRead` with `healthy`. `healthy` is the set of stations read in some hour since hour 1, so a station starved every hour never enters it.
- **Mutant on my copy:** whenever 2 or more stations are known to hang, `rotation` never polls `live[0]`. Every behavioural test in `internal/giro` and `internal/stations` stayed green, the new test included. The only failure was m107's anchor, which my edit moved. The cruder mutant (never polling `live[0]` at all) is caught only by unrelated tests.
- Severity: Minor (the code holds at 39 live). Simplify/Delete? N.
- Action: compare against the live stations not known to hang, in every hour after the first. Better, make it one of the cases R8D-1's ruling adds.

**R8D-7: section 8 understates its third limit.**
- It says: "two hanging stations side by side at the update's end can call GIRO down once".
- **Probe** `TestR8DThreeNewHangsEarly` (arm64): three stations hanging for the first time, at asks 2 to 4, call GIRO down *mid-burst* (D-286). The burst stops after 4 asks, so that hour reads 1 station.
- So in the hour before the stations are known to hang, three side by side anywhere stop the burst; two do so only at the end. This is worse than the model says.
- The same wording is in `validate-agenda.md:25`, the go-ionomaps README `:43` and `CHANGELOG.md:35`.
- Kind (b): section 8 (and the lists that copy it). Severity: Minor. Simplify/Delete? N.
- Action: reword section 8's third limit, and the copies with it. Fold in R7D-6's missing changelog item.

### 4. Is the audience order right: designers, then PMs, then engineers?

Nothing found:
- The changelog's Known entry and the README's limits lead with what a listener or host sees.
- The model leads with its principles before its tables.

### 5. Is a number published without its blind spots beside it (INST-5)?

**R8D-8: the required reading publishes the Broadcaster arm's miss, "+1.20 % CPU and +57.5 MB RSS" (`00-REQUIRED-READING.md:53`), with no blind spots beside it.**
- The figures match `g1b-summary.md:26-27`.
- Missing beside them: one 48-hour run on one machine, and the 2 h 40 m network outage window (U9 to U12). `validate-agenda.md:20` states both.
- Its unit is also "CPU", where the summary and item 3 say "% of one core".
- Severity: Minor. Simplify/Delete? N. Action: carry item 3's blind spots, or link to them, and use the summary's unit.
- Also minor: A-96's "60 ms, 405 ms under race" names no platform. Under amd64 emulation (no race) I saw a whole test at 0.57 to 0.67 s, comfortably inside `settleBound`. No finding, but the platform should be said.

### What I verified clean
- **Runs and passes over the restored code:**
  - `hungAgain` breaks both runs (`asker.go:283-285`);
  - `endEarly` counts only `timedOut` toward D-296 and leaves out the probe and known hangs from `NotReached` (`:307`);
  - known hangs rotate among themselves (my 3-hang traces), and every recovered one was read in the hour it recovered.
- **State across restarts:** `Apply` drops a retired `Keep` and `Last` (`stations.go:418-421`), and `Record` matches section 6.
- **R-5.3's test list, the ledger and the quick gates:** every test R-5.3 names exists. The `limits` row (1.22 over 41) cites D-311. Both quick gates are green.
- **watchpost:** the app's-end test is green under race and amd64; the architecture's cancel wording is fixed; item 7's five limits are listed; the links resolve locally; the re-pin to rc.13 is in `go.mod`.
- **go-tuimaps:** unchanged since round 7, quick gate green.

### Opinion
**Do not exit REVIEW.** Round 7's fixes hold where its reviewers probed, at 39 live. But D-309's "stays last", together with D-87's budget, starves a hanging station for days once 40 are live (R8D-1, Important, introduced by round 7). Under D-311 that is a change to the model, to put to the HUM LEAD whole. R8D-2 to R8D-8 are Minor and can be batched, with R8D-4 corrected in the model before anyone "fixes" D-271's code to match it.

**Fix first: R8D-1**, put to the HUM LEAD as a whole-model change (principle 3 and sections 3, 4 and 8, plus R-5.5). Add a test with 40 or more live (R8D-2's fix).

Probe files: `<scratch>/rt8-r8d/iono/internal/stations/zz_r8d_probe_test.go` and `<scratch>/rt8-r8d/iono/internal/giro/zz_r8d_probe{,2,3}_test.go`.

---

## REVIEW lens, round 8 (R8L): 0.19.0 REVIEW exit (D-313)

**Tree state at the start:** watchpost `49c65df4`, go-ionomaps `59cb4a9`, go-tuiMaps `0c27321`. All three were clean.

**At the end:**
- watchpost is still clean.
- go-ionomaps and go-tuiMaps each show ` M 06_docs/gate-runs.md`. Their rows are 16:10:11Z and 16:18:41Z. They belong to another reviewer, not to me: my gate rows went to my clones, at 16:27:40Z and 16:36:19Z.
- I edited no tree and changed no git state.

Every finding was reproduced against those SHAs, on rsync copies in `<scratch>/rt8-r8l/iono`. Each mutated file was restored and checked with `cmp`. The probes are kept in `<scratch>/rt8-r8l/probes/`.

**What I ran:**
- **watchpost (tree):**
  - `make lint`: no new findings, 6 baselined.
  - `lint-authoring`: OK.
  - `p10`: 0 live.
  - `dupes`: 2 groups, both ratified.
  - `mutant-anchors`: 827, all match.
  - `go vet` is clean on `app`, `domains/propagation` and `modes/tty`.
  - The targeted tests pass under `-race` on arm64 and under `GOARCH=amd64`.
- **Library quick gates (clones):** `scripts/gate --quick` is green for both.
- **Pins:** `go.sum:13-16` name rc.13 and rc.10, and the module cache's `asker.go`, `stations.go`, `update.go`, `answers.go` and `budget.go` match the tree byte for byte.

### Part 1: round 7's findings

| Finding | Verdict | Evidence checked in code |
|---|---|---|
| R7L-1 | **FIXED** | go-ionomaps `internal/stations/stations.go:255,265`: a station known to hang stays last whatever left it out. I replayed my round 7 case (hanging stations at asks 2, 8 and 14; 9 to 12 ms; dark probes hanging and not) for 12 hours each, on arm64 and amd64. GIRO was called down in **0 of 11 hours** after the first, and no hanging station opened a burst. m107, m108 and m109 are killed. |
| R7L-2 | **FIXED** | I applied MC (m117), MD (m114), ME (m115), MG (m116) and MI (m111) myself, and each is killed by its named test. |
| R7L-3 | **FIXED** | `update_test.go:685-703` now describes its own run and asserts `Unread == [D-RAP]`. It passed 20 times under `-race` and 10 times on amd64. |
| R7L recommendation | **FIXED** | `answers_field_test.go:115-116` asks a path from the fuzzed place too. |

### 1. Happy Path Bias

**R8L-1. Under a GIRO that hangs on every ask, D-310 stops calling GIRO down after about 15 hours. From then on, for as long as the outage lasts, watchpost says "3 stations did not answer" and never "GIRO unavailable".**
- **Severity:** Important.
- **Simplify/Delete?** N.
- **Kind (b): the model is wrong.** Section 8 says "GIRO dying just as a burst reaches the stations known to hang… the other stations' timeouts, or the next hour, catch it". For a GIRO that stays down, that is false.
- **The mechanism:**
  - Each hour of the outage, the three timeouts that call GIRO down also make those three stations known to hang. `RanOut` (go-ionomaps `asker.go:226`) runs before `stops` (`:233`), and section 3 defines "known to hang" without exception.
  - After about 13 hours, every live station is known to hang.
  - From then on, each timeout is `hungAgain` (`asker.go:228`), which resets the run (`:283-284`).
  - So `Outcome` stays `Answered` (`:211`), and nothing is backed off.
  - watchpost then shows `StationsFailed` as "N stations did not answer" (`modes/tty/map_prop_update.go:405-409`), and `giroWords` stays empty (`:292-299`).
- **Probe:** the scaled `burstUnder`, with GIRO hung on every ask for 24 hours (`probes/zz_r8l_p1_test.go`).
  - **Result:** GIRO is called down in hours 0 to 14, and is `Answered`, not paused, with `Failed=3`, in every hour from 15 on. Reproduced on arm64 and on amd64.
  - **Control:** with D-310's `res = hungAgain` removed on a copy, GIRO is called down in all 24 hours. So round 7 brought this in.
  - It lasts the whole outage. When the stations go dark at 3 days, only probes are asked, and a lone probe's timeout never makes a run of three.
- **The same effect from a hostile state file:** a state whose `Last` lists every seeded code (`stations.go` `Apply` accepts up to 118) does the same from the first hour.
- **Action, a recommendation to the HUM LEAD, whole:**
  - **The change:** in section 2, a full ask of a station known to hang that runs out counts toward the timeouts run **while no station in this burst has answered**, mirroring D-271's "before any answer".
  - **What it touches:**
    - principle 2;
    - section 2's table row, and its "called down" list;
    - section 8: its fourth limit becomes true as written.
  - **What it supersedes:** D-310's "adds nothing to D-286's or D-296's runs" in part, and D-312's reading ("D-271's included") for the timeouts run only.
  - **Why it keeps D-309 and D-310's outcome:** in a healthy GIRO the stations known to hang are asked after the rotation's answers (section 3, item 4), so they still count nothing.
  - **The test to add:** the 24-hour hang test above, asserting `Unavailable` every hour.

### 2. Systemic Failure Modes

- R8L-1 is the systemic one. A sustained GIRO outage is reported as three stations' failures each hour, and the outage disappears from the words.
- Otherwise NOAA and GIRO still take separate 60 s updates. A refusal-type outage (fast errors) clears the known-to-hang list (`RanOut(code, false)`), so it is unaffected.

### 3. Gate Validity

**R8L-2. Three rules round 7 introduced pass the suite with their code changed (my mutants, on a copy, the package run whole).**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **The survivors:**
  - **N1:** `hungAgain` counted toward D-271's run (`asker.go:284`, `inARow = 0` → `inARow++`). This is D-312's explicit reading, "D-271's included", and nothing tests it.
  - **N2:** `hungAgain` not counted in `Failed` (`asker.go:285`, `return true` → `false`). D-310's "counted as such", and so watchpost's "N stations did not answer", is untested.
  - **N3:** the known-to-hang tail reversed (`stations.go:265`, `range slices.Backward(l.last)`). Section 3's "in the order they ran out" is untested.
- **Killed:** my N4 (the rotation not skipping known-to-hang stations) and N5 (`known` read after `RanOut`).
- **Action:** add one assertion that kills each of the three.

### 4. Adversarial Input

- **Hostile state file:** the case is R8L-1's last point; the rest of `Apply` is now tested (m112 to m115 killed).
- **Paths:** origin and destination are both fuzzed now, and `sunAt` refuses infinities. m110 is killed, and `TestAPathOfNoLengthIsFinite` is green on amd64.
- Nothing else found.

### 5. Graceful Degradation

**R8L-3. With 40 or more live stations, a station that hangs once is never asked again for 3 days. The changelog says it "is read later in its hour".**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Kind (a): a departure from principle 3**, "no station is starved", and from section 8's "read later in its hour". The model's section 3 never names the budget (D-87, `MaxBurst` 40).
- **The mechanism:**
  - `rotation` puts the stations known to hang last (`stations.go:255,265`).
  - `Plan` cuts the order at 39 (`:173-174`), so with 40 or more live stations the station known to hang is always the one left out.
  - D-309's `!l.Hangs(s)` (`:255`) keeps it out of the head next hour.
- **Probe:** 41 live stations (two probes become live), one station known to hang, over 71 hours.
  - **Result:** it was polled in **0 of 71** hours, on arm64 and on amd64.
  - **Control:** with rc.12's head rule (`!l.Hangs(s)` removed on a copy), it was polled in **35 of 71**.
- **Why no test caught it:** the test (`asker_test.go:788`) runs with the seed's 39 live stations, so the budget never bites.
- **Watchpost's words are untrue here:** `CHANGELOG.md:36-37`, "one that recovers is read later in its hour".
- **Action:** a model change for the HUM LEAD. Section 3 should state the budget, for example: a station known to hang that the budget left out takes a place within the next hour's budget. Add a test with 40 or more live stations.

**R8L-4. The changelog lists four of section 8's five accepted limits.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Kind (a):** section 8 says each limit is listed in the changelog.
- **Evidence:** `CHANGELOG.md:32-37` omits "GIRO dying as a burst reaches the stations known to hang". VALIDATE's item 7 (`validate-agenda.md:25`) lists all five.
- **Action:** add the missing sentence; R8L-1 decides its wording.

### 6. Test Confidence vs. Test Results

**R8L-5. `TestHangingStationsNeverCallAHealthyGIRODown` cannot see a hanging probe open a burst. A kept probe known to hang is asked first, against principle 3.**
- **Severity:** Minor.
- **Simplify/Delete?** N.
- **Kind (b):** principle 3 ("asks the stations that answer before the stations known not to") conflicts with section 3, item 1, which has no "known to hang" exception, though item 2 has one.
- **The code:** `Plan` (`stations.go:178`) and `Burst` (`asker.go:204`) put the kept probe first.
- **Why the test misses it:**
  - Its `hanging` map excludes the probe (`asker_test.go:794`), so the "opened with" check (`:822`) never fires for it.
  - Its hanging-probe cases run only at 9 ms and 3 ms (`:779,781`).
- **Probe:** every dark station hanging, 170 hours.
  - **At 9 ms an ask:** no burst was opened by a probe known to hang.
  - **At 11 ms an ask:** 6 bursts were, each leaving 3 or 4 healthy stations to the next hour.
  - **At 12 ms (12-hour replica):** 6 of 11 hours fell short of reading every healthy station. The probe cut at the end is asked first, and hangs.
  - Reproduced on arm64; the replica also ran on amd64.
- **Action:** for the HUM LEAD, in one change with R8L-1 and R8L-3: section 3, item 1 excepts a probe known to hang, which then takes item 5's place. Add a hanging-probe case at 11 to 12 ms that checks what opens the burst.

R8L-3 also belongs under this question: the D-309 test proves its fixture of 39 live stations, not a network past the budget.

### 7. Dependency Fragility

- **Pins:** rc.13 and rc.10 are pinned and match the cache.
- **`a23484f` (v0.2.2):** still not in the local repository (`git cat-file` fails; I may not fetch), so go-tuiMaps `main`'s nested-module pins are still unverifiable here. This is unchanged from round 7.
- **GIRO's latency:** R8L-5's cost begins at about 1.1 s an ask, against PLAN's 0.92 s. The same sensitivity round 7 named, now much smaller.

### 8. Problem statement present, real, honored

- **Present:** PS-1 and PS-2 (`problem-statement.md:14-31`).
- **Not yet measured:** PS-1 is still a hypothesis (D-217).
- **Measured and short:**
  - M1 and M4 are short of target (53.7 % against 57.4 %, 68.4 % against 69.4 %). They go to VALIDATE.
  - The Broadcaster arm on `a444b000` **missed G1**: +1.20 % CPU and +57.5 MB RSS. The re-run on `c8a99e48` is under way.
  - Its `NOTE-load.txt` says round 8 shares the machine with both arms. Keep that beside the verdict.
- **Harm to PS-2:** R8L-1. During a long GIRO hang, an Observer is told three stations failed while every reading ages. The age is still said through "oldest HH:MM", but the cause is not.

### Verified clean

- **Round 7:** all R7L findings, checked in code and by my own mutants (m107 to m117 all killed).
- **D-309:** a station known to hang that a deadline cuts stays last, and is uncounted. My replica at 9 to 12 ms shows GIRO never called down after the first hour, on arm64 and amd64.
- **D-310:** a healthy GIRO beside hanging stations and a hanging probe is not called down.
- **R7C-3:** `sunAt` refuses infinities (m110 killed).
- **A-96:** the app's-end test with a burst in flight passes under `-race` and on amd64, and m577 is anchored.
- **The restart state:** the kept probe and the known-to-hang list survive a restart (m112 to m115 killed).
- **Gates:** both library quick gates, watchpost's five legs, and vet.

### Opinion

**Do not exit REVIEW.**
- R8L-1 is Important. Round 7's own D-310 makes the release stop saying GIRO is unavailable during any GIRO hang outage longer than about 15 hours. It is the inverse of the R5L-1 class, and it is a limit worse than the model's section 8 says.
- R8L-1, R8L-3 and R8L-5 are one model change for the HUM LEAD (sections 2, 3 and 8, principle 2, D-310 and D-312's reading), not three forks. R8L-2 and R8L-4 are small and can ride in the same batch.

**Fix first: R8L-1.**

---

## Code Quality, round 8 (R8C): 0.19.0 REVIEW exit red team (D-313)

**Verdict: do not exit REVIEW.** I found one Important and four Minors. Every round-7 Code finding is fixed in the code.

The Important is in the timing model, not in a departure from it. The model's section-2 rule is that "a station known to hang counts toward nothing". When GIRO itself hangs for about 15 hours, that rule turns every live station into a station known to hang. From then on GIRO is never called down. Each hour watchpost says only "3 stations did not answer", while GIRO is dead. Section 8 says the next hour catches this case; it does not (R8C-1).

### Tree state and runs
- **`git status --short` at the start and at the end:** clean in all three repos.
  - watchpost `49c65df4`, go-ionomaps `59cb4a9`, go-tuiMaps `0c27321`.
- **Every finding was reproduced against those tips.**
  - Probes and mutants were `-overlay` files in my scratch (`<scratch>/rt8-r8c`). I edited no tree.
  - The go-ionomaps gate ran on a byte copy, so its gate-runs row is in the copy, not the tree.
- **go-ionomaps `scripts/gate --quick` on the copy: green, 311 s.** Race leg 239 s, race-off leg 53 s, P10 green.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `lint-authoring`, `p10` (0/0/0), `dupes`, `mutant-anchors` (827), `go vet ./...`.
  - `TestTheAppsEndStopsAnUpdateAndKeepsItsState` passes with `-race` on arm64, and on amd64.
- **amd64 (Rosetta):** go-ionomaps `internal/giro`, `stations`, `limits` and `deadline` are green. So are the root tests `TestAPathOfNoLengthIsFinite`, `TestARoundWithNoRoomNamesWhatItLacks`, `TestADeadlineWithNoRoomBacksNothingOff` and the fuzz seeds. R8C-1's probe gives the same result on amd64.
- **go-tuiMaps:** no code change since round 7. `a23484f` is not in the local clone, and I may not fetch, so I could not check it.

### Part 1: round 7's Code findings, checked in the code
| ID | Status | Evidence |
|---|---|---|
| R7C-1 | FIXED | go-ionomaps `internal/stations/stations.go:255`: the stations left out are filtered by `!l.Hangs(s)`. `:265` puts the stations known to hang last. `asker.go:307` leaves them out of `NotReached`. My probe, two hanging stations over 10 hours (arm64): 37 of 37 answering stations read every hour, `NotReached` 0, GIRO never called down. |
| R7C-2 | FIXED | `stations_test.go:247`. m112 to m115 cover round 7's yM1, yE2, yC and yN. yM2 (`Apply` ignoring `Last`) is caught by the `back.Hangs` check. |
| R7C-3 | FIXED | `answers.go:408` refuses Inf as well as NaN. With finite inputs, `c` and `rise` are finite and `rise` ≥ 1e-3, so the guard on the inputs is complete. Green on amd64. |
| R7C-4 | FIXED | `update_test.go:702` asserts exactly `[D-RAP]`, and the test's doc now describes the run that happens. |
| R7C-5 | FIXED in code; residue in the changelog | The code says "until an ask of it ends any other way" everywhere. watchpost `CHANGELOG.md:37` (new in batch 72) says "until it answers again" (R8C-4). |
| R7C-6 | FIXED | watchpost `app/mapprop_test.go:272` now drives GIRO's burst. My mutant, `keepState` skipped when the update's context was cancelled (`domains/propagation/propagation.go:227`), fails at `:330` ("asked GIRO 40 times"), so the restart check can fail. |
| R7C-7 | FIXED | `asker_test.go:468`. Note, not numbered: `MinDeadline` is still a literal 32 s (`update.go:330`), not derived from the giro constants, so the same rule is written twice. |
| Q3 note, nit | FIXED | `onPath` (`internal/limits/budget.go:91`), its floor renewed under D-311. `restore` replaces `endOf`. |

### 1. Understandability
- **R8C-1 (Important), go-ionomaps: when GIRO itself hangs for many hours, every station becomes "known to hang", and GIRO is never called down again.**
  - **Kind (b), the model is wrong.** The code follows sections 2 and 3 exactly. The defect is that the rules combine badly, so section 8's limit is worse than the model says.
  - **The mechanism:**
    1. Each hour of an outage, three timeouts in a row call GIRO down (`asker.go:339`).
    2. `RanOut` also marks those three stations, and the dark probe, as known to hang (`asker.go:226`).
    3. Nothing clears a station from that list while GIRO stays silent, and section 6 keeps the list across restarts.
    4. Once every live station is on the list, each timeout becomes `hungAgain` (`:228`) and counts toward no run (`:283`).
    5. The only counted timeout left is the kept probe, asked first: one, never a run.
    6. So `endEarly`'s `t.timeouts >= timeoutsAtTheEnd` check (`:309`) never fires.
  - **Probe** (scaled ÷100, 39 live stations, 600 ms update, every ask hangs; same result on arm64 and amd64):
    - Hours 0 to 14: `Unavailable`, as it should be.
    - From hour 15 on: `Outcome=Answered`, not paused, `Failed=3`, `NotReached=0`.
    - When GIRO recovers, everything is read within the hour.
  - **What watchpost then shows:** "39 stations, oldest …· 3 stations did not answer" (`modes/tty/map_prop_update.go:405-410`), with no "GIRO unavailable" and no "cut short".
  - **What this breaks in the model:**
    - Principle 2: GIRO stopped answering and is not called down.
    - Principle 4: "said as it is".
    - Section 8, bullet 4: "the other stations' timeouts, or the next hour, catch it".
  - **Sections and rules a fix touches:** principles 2 and 4; section 2 (the known-to-hang row and the list of call-down rules); section 3 (when a station becomes known to hang); section 4's words; section 6 (the list kept across restarts); section 8, bullets 3 and 4.
  - **Options for the HUM LEAD:**
    - (A) A timeout that is part of a run that calls GIRO down does not mark its station. This re-breaks D-310's first-hour case: two stations side by side would call GIRO down every hour.
    - (B) A call-down clears the list. Same defect as (A).
    - (C) **Recommended:** add a fifth call-down rule to section 2. A burst in which no station answered, and at least `failuresInARow` asks ran out or failed, known to hang or not, calls GIRO down. With a healthy GIRO, someone always answers first, so D-309 and D-310 still hold. It supersedes nothing; it rewords section 8, bullet 4.
  - Simplify/Delete: N.
  - Action: the HUM LEAD rules the model change whole. Then add a multi-hour full-outage test, as in my probe.

### 2. Maintainability
- State lives where its lifetime says: `last` is in the list and persisted (section 6). The order rule is written once, in `rotation`.
- Note: the `MinDeadline` literal (see R7C-7 above).

### 3. Complexity
- Nothing is over-complex. `tally.add` (`asker.go:272-289`) now has an if/else and a switch with three exits. It reads correctly.
- `rotation`'s three loops each call `slices.Contains`, which is quadratic with n ≤ 123. Fine.

### 4. Necessity
- Nothing is dead.
- The `!slices.Contains(out, c)` check in `rotation`'s tail loop (`stations.go:266`) only guards against duplicate codes in `State.Last`. `Apply` does not refuse duplicates, unlike `LeftOut`, so the check is needed.

### 5. Name semanticism
- Nit, not numbered: the field `List.last` and `State.Last` are named for a position. The model's word is "known to hang", and `State.Last`'s doc (`stations.go:353`) still reads "asked last (D-307)", without D-309.
- The JSON key has to stay. The Go field could be renamed.

### 6. Code documentation
- **R8C-4 (Minor), watchpost: the changelog's Known section, new in batch 72, drops one of section 8's limits and brings back R7C-5's second story.**
  - `CHANGELOG.md:33-37` lists 4 of the 5 limits. It leaves out "GIRO dying just as a burst reaches the stations known to hang". The model requires all five in the changelog; the README and VALIDATE's item 7 have all five.
  - It also says "asked last until it answers again".
  - Simplify/Delete: N. Action: add the missing limit, worded per R8C-1's ruling, and say "until an ask of it ends any other way".
- No history in comments (AP-HIST-01) anywhere in the diff.

### 7. Testability
- **R8C-2 (Minor), go-ionomaps: two behaviours of the new code are unpinned. My mutants of each survived (arm64).**
  - **zF:** dropping the `slices.Contains(live, c)` guard in `rotation`'s tail loop (`stations.go:265-266`) survives `./internal/...`.
    - Hung probes are dark stations, and they join `last`. Without the guard they would be polled as live stations whenever fewer than 39 stations are live, which breaks R-5.5's one probe an hour.
    - The test passes only because 39 live stations fill the budget.
  - **zK:** making `hungAgain` return `false`, so it is not counted in `Failed`, survives `./internal/giro`. The model says it is "the station's own failure", and watchpost prints the count.
  - **zL:** dropping `hungAgain`'s `t.inARow = 0` survives as well. It is close to unreachable (it needs no answer before the tail), so it is noted only.
  - Simplify/Delete: N. Action: a test with fewer live stations than the budget and a hung probe; assert `Failed` in the D-310 test.

### 8. P10 conformance
- Both quick-gate P10 legs are green, and watchpost's `make p10` is 0/0/0.
- `internal/limits` was renewed at 1.22 (D-311, ratified).
- `internal/stations` was renewed at 1.29 by the agent under the model's section 10, which D-312 approved, and is listed in A-37 (3) for veto. That is not self-issued. It still awaits the HUM LEAD's veto window.

### 9. Evidence soundness
- **R8C-3 (Minor), go-ionomaps: `TestHangingStationsNeverCallAHealthyGIRODown`'s "four hanging stations" case runs with three.**
  - Stations hang only if the first hour's burst reaches them (`asker_test.go:778`, asks 5, 15, 25 and 35). The deadline cuts that burst before ask 35.
  - With a log added by overlay: "four hanging stations: hour 0 made 3 stations hang of 4 named". D-309 asked for four.
  - The code does hold with four and with five hanging from the start: my probe read 35 of 35 and 34 of 34 every hour from hour 2.
  - The read check is also only relative: `liveRead < len(healthy)` (`:826`). A station never read in any hour never joins `healthy`, so it would be invisible.
  - Simplify/Delete: N. Action: choose the hanging stations from `Live` up front, and assert reads against live stations minus hanging ones.

### 10. Evasion lens
- R8C-3: D-309's "four hanging stations" is met in the test's name, not in its run.
- R8C-1: section 8's limit, "caught by the next hour", is listed as accepted in the README and VALIDATE's agenda, but it is false for a long outage.

### Verified clean
- **go-ionomaps:**
  - `hungAgain` is decided from `Hangs`, read before `RanOut`.
  - A known station whose ask is shortened or cancelled stays known.
  - A 429 or a 5xx releases a station from the list.
  - `NotReached` leaves out the stations known to hang and the probe.
  - Restoring the counts when a burst asked nothing.
  - `Apply` drops codes the seed no longer names, and enforces its length bound.
  - `onPath`.
  - `sunAt`'s guard is complete.
  - m107 to m109 pinned; my mutants zJ, zN and zP were killed.
- **watchpost:**
  - The re-pin to rc.13.
  - The app's-end test can now fail on the burst's hour.
  - m577.
  - The architecture doc's link to the model.

### Opinion
**Do not exit REVIEW** until the HUM LEAD rules R8C-1 as a whole-model change. I recommend option (C), a burst with no answer and three asks lost calls GIRO down whoever was asked, which leaves D-309 and D-310 intact. It is a small code change plus one multi-hour outage test. R8C-2 to R8C-4 can be batched with it.

**Fix first: R8C-1.**

---

## Red team round 8: the Timing lens (R8T)

**Tree states.** At the start, `git status --short` was empty in all three repositories: watchpost `49c65df4`, go-ionomaps `59cb4a9`, go-tuiMaps `0c27321`. At the end, go-ionomaps and go-tuiMaps each show one modified file, `06_docs/gate-runs.md`, with rows timestamped 16:10:11Z and 16:18:41Z. **Those rows are not mine.** I ran every gate in scratch `git clone`s, and their rows went into the clones' own logs (16:02, 16:11 and 16:17Z). Another reviewer must have run a gate in the trees.

Every finding below was reproduced against those clean HEADs. The probes ran on a copy at `<scratch>/rt8-r8t/ionomaps`, on arm64 and on amd64 (Rosetta).

**Gates:**
- watchpost (scratch clone at `49c65df4`): `make lint`, `lint-authoring`, `p10` (0/0/0), `dupes` and `mutant-anchors` are all green. The first p10 run failed only because the clone lacked the git-ignored `.a2dh.yml`; I copied it in and the re-run was green.
- go-ionomaps: `scripts/gate --quick` is green. The first run failed only on the clone's missing ignored `.a2dh-p10-exemptions.yml`.
- go-tuiMaps: `scripts/gate --quick` is green.
- Also green: go-ionomaps `giro`, `stations`, `deadline`, `throttle` and the root package under `GOARCH=amd64`. watchpost `domains/propagation`, `app` and `modes/tty` (filtered to `Prop|GIRO|Burst`) pass with `-race`, and `go vet` is clean.

## Part 1: round 7's timing findings

| Finding | Verdict | Evidence, checked in the code |
|---|---|---|
| R7T-1, R7D-1, R7L-1, R7C-1 | **FIXED** | go-ionomaps `internal/stations/stations.go:255` (`leftOut` skips `Hangs`), `:261`, `:265-269` (the tail, in run-out order); `asker.go:307` (not counted). `TestHangingStationsNeverCallAHealthyGIRODown` (`asker_test.go:768`) passes on both platforms. My mutant X7 (rotation not skipping `Hangs`) is killed. |
| R7T-2, R7D-2 | **FIXED** | `asker.go:225-229` (`hungAgain`), `:283-285`; registered mutant m108. But its readings are unpinned (R8T-4), and it carries a side effect (R8T-1). |
| R7T-3 | **FIXED** | watchpost `app/mapprop_test.go:272-330` now cuts a real burst in flight, with a full field. m577 (`settleBound` set to 2 ms) applied to a copy was killed 3 of 3 times. A 500 ms bound still passes under race: the value is pinned by the test, not exactly, as A-96 (2) ruled. |
| R7D-5 | **FIXED** | `asker_test.go:768-860`: 7 cases × 10 hours. It asserts every healthy station read, no call-down and `NotReached` 0. |
| R7L-2 | **FIXED** | m111 (MI), m114 (MD), m115 (ME), m116 (MG) and m117 (MC) are registered. The quick gate is green. |
| R7C-2 | **FIXED** | `stations_test.go:247` round-trips the state. My X8 (`Apply` drops `Last`) is killed. |
| R7C-7 | **FIXED** | `asker_test.go:468-478`: a hung GIRO is called down within 2×(ask+margin). |

## The lens

### 1. Bounded
Clean.
- **Sweep:** 861 scaled bursts (1/100). Deadlines of 32, 33, 40, 45, 60, 90 and 120 s; a hang at each position from 1 to 41, of one station, of every station from there on, or of a pair; answers at 0.92 s.
  - Every burst ended before its deadline.
  - No ask began with less than the floor plus the margin left.
  - A single hanging station never called GIRO down.
- **Long runs:** 30 hours of a dead GIRO, and the R8T-2 runs, ended inside every deadline on both platforms.
- **NOAA:** `update.go:354` holds an ask to all-or-none with 21 s left.

### 2. Truthful outcomes
**R8T-1 (Important, kind (b): the model is wrong). A GIRO that hangs for about 15 hours is never called down again, and watchpost says "3 stations did not answer".**

The mechanism:
- Each call-down by timeouts also marks the stations that timed out as known to hang, because `RanOut` runs before `stops` (go-ionomaps `asker.go:226` before `:233`; `stations.go:197-200`).
- That is exactly the model's definition (model `:79`): a station is known to hang "from the burst in which a full ask of it runs out".
- The evidence D-286 counted as GIRO's is turned into evidence about the stations. Three more stations an hour (two in hours that open with the kept probe) become "known".
- Once fewer than three unknown live stations remain, every further timeout is `hungAgain` (`:227-229`), which counts nothing (`:283-285`). Neither D-286 nor D-296 can fire.

The probe (`TestR8TDeadGIRO`): 1/100 scale, every ask hangs from hour 1.
- Hours 1 to 15: Unavailable, 3 to 38 of 39 stations known to hang.
- **From hour 16 on: `Outcome` is Answered, not paused, `Failed` 3, `NotReached` 0**, all 39 stations known to hang.
- Reproduced identically on amd64.
- watchpost then shows "3 stations did not answer" (`modes/tty/map_prop_update.go:405-409`), never "GIRO unavailable" (`:296-297`). The 36 stations left unasked are uncounted (`asker.go:307`), so not even "cut short" is said.
- `Last` is kept across restarts (model section 6), so the state persists for up to the 72 h `darkAfter`.
- This makes section 8's limit 4 (model `:130`), "the other stations' timeouts, or the next hour, catch it", false: no hour catches it. It also breaks principle 4.

Simplify/Delete? N.

**Action: a whole recommendation to the HUM LEAD.** A known-to-hang station's run-out is `hungAgain` only once some station has answered in this burst. Before any answer it counts like any full ask run out, mirroring D-271's "before any answer".
- **Code:** `asker.go:227` gains `&& t.anyAnswered`.
- **Sections it touches:**
  - principle 2 (`:26`): the clause "never because of … a station already known to hang" becomes "known to hang, once a station has answered in the burst";
  - section 2: the row at `:57`, and the call-down list `:63-67`;
  - section 8: limit 4 (`:130`) reworded.
- **What it supersedes:** in part, D-310's "adds nothing" and D-312's approved reading "D-271's included".
- **Sections 3, 4 and 6 are unchanged.**
- **Evidence:** applied to my copy, the whole giro suite passes, including `TestHangingStationsNeverCallAHealthyGIRODown`. The dead GIRO is then called down in all 24 hours, and the network is read whole on recovery.
- **Test to add:** GIRO hung for 16 hours or more.

Otherwise:
- A cancel and no room back nothing off (`asker.go:301-313`, `update.go:353-356`).
- A shortened ask that runs out is not counted (`asker.go:361-362`).
- A station's failure is its own (`:385-398`).

### 3. Edges
Clean:
- 16 s and 21 s left: `deadline.go:42-49`. At exactly 16 s, the nanoseconds already elapsed make the ask shortened, which is harmless.
- 32 s: the test at `asker_test.go:468`.
- The floor, and a shortened ask that answers.
- A timeout beside a cancel: m103 and m104.
- Two timeouts with a 5xx or an answer between: `tally.add` (`asker.go:274-278`).
- A path of no length: m106 and m110; the root suite passes under amd64.
- A host client that times out first: `deadline.IsTimeout` reaches the inner `net.OpError` through watchpost's `redactErr` (`platform/httpx/httpx.go:951-955`).

### 4. Across hours
**R8T-2 (Minor, kind (b)). A station known to hang is never asked again while the healthy stations fill the hour, and it is never counted or said.**

The mechanism: the stations known to hang are last (`stations.go:265-269`) and uncounted (`asker.go:307`). Once the healthy stations need the whole update (about 1.55 s an ask, 1.7× PLAN's 0.92 s), the tail is never reached.

The probes (`TestR8TStarve`):
- Two stations that hung in one hour only (a GIRO hiccup), then GIRO at 1.6 s an ask: **both are unread for the next 10 hours**. `NotReached` is 3 to 4 (healthy stations only), so the words name none of them.
- At 1.4 s, they are read the next hour.
- After R8T-1's 16-hour outage, recovery at 1.6 s **leaves 6 healthy stations unread for 11 or more hours** (5 on amd64; `TestR8TRecoverSlow`).

This breaks principle 3, "no station is starved" (model `:28`). Section 8's limit 5, "read later in its hour" (`:131`), is understated.

Simplify/Delete? N.

**Action: a fork for the HUM LEAD, touching principle 3 and sections 3, 4 and 8** (and section 1 if a new ask length is chosen):
- (i) a known-to-hang station a deadline left unreached gets one short ask (the floor) at the head next hour;
- (ii) count such stations in `NotReached`, in words of their own;
- (iii) accept it, and state limit 5's threshold.

Otherwise clean:
- The back-off cannot grow without cause: it is capped at 15 min, below the hourly `Due` (`throttle.go:78-83`, `:45-56`).
- No hour is spent with nothing asked: `asker.go:302-304`.

### 5. Tests
**R8T-4 (Minor, kind (a)). Round 7's approved readings are unpinned. Six mutants, each applied to a copy, survive giro, stations, deadline and the root package** (only the git-listing, P10-mirror and mutant-registry tests skipped):
- **X1:** `hungAgain` counts toward D-271's run (`asker.go:284`, `inARow++`). This is D-312's "D-271's included".
- **X2:** `hungAgain` does not break the timeouts run (`:276`). This is the model row at `:57`.
- **X3:** `hungAgain` is not counted in `Failed` (`:285`), which drives watchpost's "N stations did not answer".
- **X5:** the tail is iterated in reverse (`stations.go:265`), against "in the order they ran out".
- **X6:** a station that runs out again keeps its place in `last` (`:197-200`).
- **X9:** a run of timeouts moves the rotation's head (`asker.go:322`), against section 3's "only when that station's own ask failed".

Killed: X4 (`known` read after `RanOut`), X7 and X8.

Simplify/Delete? N.

**Action:** one test for each survivor. For X1: a known-to-hang kept probe followed by two 5xx before any answer. For X2: a new timeout, a known one, then a new one at the deadline.

### 6. Host and library
These agree with section 7:
- The deadline: watchpost `domains/propagation/propagation.go:86` refuses anything under 32 s, and `:224` gives each update its own 60 s.
- The app's cancel comes before the wait: `app/mapprop.go:123-124`, then `dashboard.go:132` and `:1016`.
- The refresh tick, and the burst asked at once after StationsNext: `map_prop_update.go:180-187`.
- The cut-short words are said only while GIRO answered, and noted again only when the count changes: `:164`, `:411`.

Exceptions:
- The words under R8T-1.
- **R8T-3 (Minor, kind (a), docs):**
  - watchpost `CHANGELOG.md:33-37` omits the model's limit 4, though section 8 (`:125`) says each limit is listed there. VALIDATE's item 7 lists all five.
  - Section 8 omits D-303's weighed cost. With dark probes that hang, a not-reached probe opens the next hour with a full 15 s ask. At 1.05 to 1.15 s an ask, alternate hours lose 2 to 5 live stations; at 0.92 s, none (`TestR8TDarkProbesHang`).
  - Simplify/Delete? N. Action: add both.

### 7. The model
- **Kept by the code:** sections 1 (`update.go:330`, `:343`, `:354`; `deadline.go:16`, `:37-50`; `asker.go:44`), 4, 5 (`update.go:136-168`, `:317-323`), 6 (`stations.go:377-427`) and 7.
- **Section 3** is kept as written but is wrong as a definition (R8T-1), and starves under a slow GIRO (R8T-2).
- **Section 8:** limits 4 and 5 are worse than stated (R8T-1, R8T-2), and one weighed cost is missing (R8T-3).
- **R8T-5 (Minor, kind (a), wording):** section 2's table (`:56`) says a counted timeout counts toward "the timeouts run" only. The code also counts it in D-271's failures-in-a-row run (`asker.go:280-282`), and `TestTimeoutsAreFailuresAndARunOfThem` (`asker_test.go:510-525`) asserts that a 500, a timeout and a 500 call GIRO down. Simplify/Delete? N. Action: add it to the row; fold it into R8T-1's whole change.

## Verified clean
- **Bounded** at every timing I built: 861 swept bursts plus multi-hour runs, on both platforms.
- **The least deadline:** at 32 s a hung GIRO is called down inside it.
- **Edges:** NOAA's all-or-none ask, the room rule and its margin, cancel and no-room handling, and a host client's timeout.
- **Round 7's remediation:** every finding is fixed as specified.
- **watchpost:** its seam, its wait at exit (m577 killed) and its words, apart from R8T-1.
- **Gates:** all green.

## Opinion
**Do not exit REVIEW.** R8T-1 is the Important that rounds 3 to 7 kept finding, here in round 7's own cure. "Known to hang" is assigned from timeouts already counted as GIRO's, so a GIRO hung for about 15 hours of bursts is said to be answering, indefinitely. It is a model change: one line of code and three model sections, presented whole above and already proven against the suite.

**Fix first: R8T-1.** R8T-2 is the same root, a "known to hang" label that is too sticky, so fork the two together.
