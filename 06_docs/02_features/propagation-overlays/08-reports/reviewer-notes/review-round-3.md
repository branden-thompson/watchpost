---
title: "0.19.0 REVIEW exit red team, round 3 - the reviewers' reports, verbatim"
date: 2026-10-26
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "FILED - three fresh blind reviewers (D-284), each report as delivered; only each reviewer's scratch folder is shortened to <scratch>; the verdicts, forks and remediation are in ../red-team-review.md, Round 3."
---

# Round 3, the reports

Up: [red-team review](../red-team-review.md) · [round 2](review-round-2.md) · [round 1](review-round-1.md)

Each reviewer worked read-only in a scratch directory of its own; every report is as delivered.


---

## Docs Quality, round 3 (R3D): 0.19.0 REVIEW exit red team

**Trees at the start.**
- watchpost `03ad4e97`: ` M …/02-analysis/rulings.md` (D-284). Not my change.
- go-ionomaps `e2103d7`: clean.
- go-tuimaps `474797a`: clean.

Every finding below was reproduced against these states. At the end, both libraries show ` M 06_docs/gate-runs.md`. That is the green row each required `scripts/gate --quick` writes. I left both rows in place, because the gate record keeps its runs.

**What I ran (all exit 0):**
- watchpost: `make lint`, `lint-authoring`, `p10` (0 live / 0 unmatched / 0 unratified), `dupes` and `mutant-anchors` (814 match).
- watchpost: `go vet ./tools/... ./modes/tty/`; `go test -race` on `tools/{perfsum,p10floor,finaltag,runlog}`; the record and gate tests in `cmd/watchpost`.
- go-ionomaps `scripts/gate --quick`: green, P10 ran. go-tuimaps `scripts/gate --quick`: green.
- I built the committed `perfsum` into scratch. I ran it on G1's run and on the Broadcaster run so far (read-only).
- I ran `go doc -all` on both libraries' public packages.
- I checked the 26 test names the build logs cite: all exist.

### Part 1: round 2's findings

| Finding | Verdict | Evidence |
|---|---|---|
| R2D-1 | FIXED | `tools/perfsum/g1.go:48-51,126,143,151-152`. The committed `perfsum -g1 g1-run-2026-10-19` now **exits 0**, prints the D-273 line and gives "met" against the map. |
| R2D-2 | FIXED | `00-REQUIRED-READING.md:47`, `problem-statement.md:105`, `validate-agenda.md:21,26`: one run, one machine, the noise between arms. |
| R2D-3 | FIXED in code | go-ionomaps `scripts/gate:238-245` (REFUSED, exit before any leg). go-tuimaps `scripts/gate:256,276,343` (`logging` gate). But go-tuiMaps' README still says otherwise (R3D-6). |
| R2D-4 | FIXED | `go doc -all` shows no D-/A-/R?-n IDs and no `go doc ./internal` pointers. `describe.go:15-21` and `overlays.go:36-62` are clean. The scope is narrow (R3D-10). |
| R2D-5 | FIXED | `00-REQUIRED-READING.md:21`: "will say so … (D-250, D-255)". |
| R2D-6 | FIXED | `00-REQUIRED-READING.md:45-50` is in order and dated 10-25. `red-team-review.md:179-265` and `:320-371` are the fix maps. One new error inside them (R3D-5). |
| R2D-7 | FIXED | `README.md:207`. It matches `map_prop_chart.go:24-34` and `map_pane.go:1474-1479`. |
| RD-1 | FIXED | Both comparisons are in the tree, and the scorer now agrees with D-241 (via D-273). |
| RD-12 | FIXED in code (as R2D-3) | |

### Part 2 and 3: the lens

#### 1. Does each document answer the question its title asks?

**R3D-5: The red-team record says the R2H-5 tags were made, in two places, and says they are open in a third.**
- `red-team-review.md:306` says "**v0.2.2's nested modules tagged** (R2H-5)", in bold.
- `:232` (RH-7) says "R2H-5: the nested modules tagged".
- `:353` and D-284 say R2H-5 is open.
- `git -C go-tuimaps tag -l '*/*'` shows only the `v0.2.0` and `v0.2.1` nested tags. No `*/v0.2.2` tag exists.
- Also, the H1 (`:10`) reads "red team, round 1", under a title that says "rounds 1 and 2".
- Severity: Minor. Simplify/Delete? N.
- Action: reword `:232` and `:306` to "open", and retitle the H1.

Otherwise each document answers its title.

#### 2. Does anything contradict the code, or another document?

**R3D-1: go-ionomaps' requirements were not updated for any of round 2's behaviour changes. Four rows now contradict the code.**
- `requirements.md` was last touched in `00ac871`, round 1.
- **R-5.3** (`:75`) says "three in a row stop it as a 429 does", and "a 401, 403 or 410 pauses GIRO for 24 hours". It has no 15 s timeout. The code differs:
  - three failures stop the burst only before any station answers (`internal/giro/asker.go:178-190,219`, D-271);
  - a 410 is the station's own failure (`:144-147`, D-272);
  - each ask has 15 s (`:32,234`, D-275).
- **R-5.8** (`:80`) says a refused state leaves "the library left as it was". In fact the mean and GIRO's state are each taken on their own (`state.go:105-115`, A-30 (3)).
- **R-3.3** (`:56`) and **R-9.8** (`:114`) give the residual bounds alone. They do not mention `FoF2Fold` (`internal/assimilate/onto.go:45,80`, A-31).
- Severity: **Important**: the requirements are the document that wins a conflict.
- Simplify/Delete? N.
- Action: rewrite those four rows and name their tests: `TestFailuresAfterAnAnswerAreTheStations`, `TestAHungGIROIsCalledDown`, `TestAnImplausibleReadingIsRefused`, and the restore test.

**R3D-2: R2L-4's partial restore is still described as all-or-nothing, in both repositories and in a watchpost diagnostic.**
- go-ionomaps:
  - `state.go:15-17`: `ErrBadState`'s exported doc says "The library is left as it was";
  - `README.md:30`: "ErrBadState leaves it as it was";
  - `api-review.md:25`.
- watchpost:
  - `architecture.md:131`: "a refused state starts as new";
  - `requirements.md:134` (FR-4.12): "the service starts as new";
  - `domains/propagation/propagation.go:113-116`, the comment;
  - `:134`, the note "GIRO's throttle state was not taken, so it starts as new". That note is false when only the F10.7 mean was refused, because GIRO's state *was* taken (`state.go:108-112`). It is half false in the reverse case.
- Severity: Minor (the note goes to diagnostics only).
- Simplify/Delete? N.
- Action: the doc says which part was taken. The note names the part that was refused.

**R3D-3: `FoF2Fold` reached the code but not the library's own records.**
- The constants table, "every measured figure in one table … cannot disagree" (`constants.go:32-45`), lists `FoF2Bound` and `M3000Bound` but not `FoF2Fold`. Its basis cites D-248 alone (`:44`).
- The exported doc of `Inputs.Implausible` (`snapshot.go:131`) says "past its residual's bound". A fold refusal is counted there too (`onto.go:80-82`).
- Severity: Minor. Simplify/Delete? N.
- Action: add the fold's row (D-279, A-31) and fix the field's doc.

**R3D-6: go-tuiMaps' README says every gate run is logged, while its gate now logs no refusal.**
- `README.md:154`: "Every run leaves a line in `06_docs/gate-runs.md`."
- `scripts/gate:256,343`: a refusal leaves no line (D-43).
- The README was edited in this same candidate (`474797a`).
- Severity: Minor. Simplify/Delete? N.
- Action: "every run that reaches a leg".

**R3D-7: go-tuiMaps' release notes leave out rc.6's two behaviour changes a host can see.**
- `release-notes.md:7-36` carries rc.5's `DaylightAt`. It does not carry:
  - reply headers capped at 64 KiB (`internal/fetch/fetch.go:141,164`; A-19 (4));
  - `DaylightAt`'s `Known` false for a place off the globe (`daynight.go:43-46`).
- Severity: Minor. Simplify/Delete? N.
- Action: add them under "What changes in behaviour".

**R3D-8: `docs/extending.md` says a key is named in exactly two places. There are now four keymaps.**
- `docs/extending.md:16-18` names two keymaps.
- The code also has `defaultMapKeyMap` (`modes/tty/map_pane.go:912`) and 0.19.0's `defaultChartKeyMap` (`map_prop_chart.go:30`). The file was touched in this release (batch 15).
- Severity: Minor. Simplify/Delete? N.
- Action: name the scopes.
- A related gap, for the HUM LEAD to weigh: `README.md:210-211` says any map key can be rebound under `[keys]`, but names no action names (`chart.prev`, `chart.next`).

#### 3. Is a claim made that nothing verifies, in a document that reads as verified?

**R3D-4: The gate roster was not updated for batch 66, and one of its rows is now false.**
- `07-readiness/gates.md` is not in batch 66's diff. Row `:29` still says "a default run under 48 hours exits missed". The code now takes a run as 48 hours within one sampling interval (`g1.go:51,126`).
- Five more gate changes made in batch 66 have no row, so the roster's own rule (`:14-16`, each changed gate with a watched failure) does not hold:
  - `finaltag` refuses any `replace` of a paired library, pinned by `TestTheReleaseRunsFinaltag` (R2H-3);
  - `p10floor` fails a firing density row that states no floor (D-277);
  - `test-say` retries once, diagnosed (D-276);
  - `runlog` fails a run that writes no record (RC-3);
  - the libraries' refusal rule (D-280).
- The watched failures exist: m543 to m557.
- Severity: **Important**: this is the roster VALIDATE reads, and it is wrong about the G1 scorer.
- Simplify/Delete? N.
- Action: amend row 29 and add rows for the other five, each citing its mutant.

Otherwise the claims I checked are backed. That covers the test names, mutant files m543 to m562, `d279-verdict.md`, A-91 (2)'s budget against `map_prop_alloc_test.go`, and the gate rows behind each candidate.

#### 4. Is the audience order right: designers, then PMs, then engineers?

**R3D-10: D-281 was applied narrowly. Plan IDs and a host's internal tool remain in public docs.**
- `go doc ./ionomapstest` reads "G10.4's tools/gm3; watchpost's tools/reachscore, W10.3".
- A-30 (1) and A-19 (1) keep requirement IDs because they are "the published work's own". That reads R-7.1 the same way A-29 (13) did, the reading D-281 reversed.
- Severity: Minor. Simplify/Delete? Y.
- Action: a recommendation to the HUM LEAD, not ruled by me.

Otherwise the order is right: the required reading and the README lead with what the listener sees.

#### 5. Is a number published without its blind spots beside it (INST-5)?

**R3D-9: G1's span is stated more exactly than it was, and the margin it passed by is not stated.**
- `problem-statement.md:105` and build-log batch 66 say "each arm 47.97 hours". D-241 says "47.97 to 48.00".
- `phases.log` and `samples.csv` give: in use 47.995 h; the map open in Forecast mode 47.969 h (1.85 minutes short); never opened 47.975 h.
- So the judged arm passes D-273's 2-minute tolerance by 9 seconds, and nothing says so.
- The cadence `perfsum` calls "the harness's own resolution, 2 minutes" (`g1.go:48-51`) measured about 2.11 minutes: 1,363 intervals over 48 hours.
- Severity: Minor. Simplify/Delete? N.
- Action: state the per-arm spans and the margin beside "taken as 48".

**R3D-11: Two more figures are published without their limits.**
- F-220's target rests on "+3.59 % and +50.4 MB" (`follow-ups.md`, F-220). That is one run, with arm noise of at least 10.4 MB (R2D-2's own point), and the row says neither.
- `FoF2Fold` = 4.5 (`onto.go:34-42`, A-31) was set from quiet days with no storm. D-279's counter-argument names that limit, but it is not beside the figure.
- Severity: Minor. Simplify/Delete? N.
- Action: one clause each.

### What I verified clean

- R2D-1, R2D-2 and R2D-4 to R2D-7, RD-1 and RD-12, each against the code.
- The refusal rule's code in both libraries' gates.
- Every watchpost lint, P10, dupes and mutant-anchors target.
- Both libraries' quick gates.
- The new D-273 output on the real run.
- Every test and mutant the round 2 build logs name.
- The agenda's figures against D-246, D-255 and D-274.
- The as-built map's status line, the CHANGELOG against the code, and the README key table.
- The exported doc comments: free of D-/A-/R?-n IDs.

### Opinion

**Do not exit REVIEW** yet. Nothing is Critical, and every fix is to a document.

Fix first: **R3D-1**. go-ionomaps' requirements are the document that wins a conflict, and four rows describe behaviour round 2 removed. R3D-4 is a close second.

---

#### REVIEW lens, round 3 (R3L): 0.19.0 REVIEW exit (D-284)

**Tree state.** At the start: watchpost `03ad4e97` had ` M .../02-analysis/rulings.md` (the uncommitted D-284 row, not mine). go-ionomaps `e2103d7` and go-tuiMaps `474797a` were clean. At the end, both libraries showed ` M 06_docs/gate-runs.md`. Those rows are other reviewers': I edited no tree. Every finding was reproduced against those SHAs, in my scratch `.../scratchpad/rt3-r3l/`:
- `iono/`: an rsync copy, where the probes ran.
- `iono-old/`: `git archive 8d84569`, round 2's tip, used as a control.
- `ionoc/`, `tui/`, `wp/`: local clones.

**What I ran:**
- watchpost (clone): `make lint` (no new findings, 6 baselined), `make lint-authoring` OK, `make p10` (0 live, 0 unmatched, 0 unratified), `make dupes` (2 groups, both ratified), `make mutant-anchors` (814, all match). `go vet` was clean on the touched packages. `go test -race` passed on `domains/propagation`, `tools/{finaltag,runlog,perfsum,p10floor}` and the Propagation/setup tests in `modes/tty`.
- go-ionomaps `scripts/gate --quick`: **green**.
- go-tuiMaps `scripts/gate --quick`: the first run FAILED on `TestWorkspaceFileIsIgnored`. That was my own artifact: my copy step spilled `.a2dh/.gitignore` (`*`) over the clone's `.gitignore`. Once I restored it, the rerun was **green**, `TestALegThatIgnoresTermIsStillStopped` included.
- One mutant of my own (on the copy, then restored and checked with `cmp`): it survived. See R3L-4.

---

##### Part 1: round 2's findings (and round 1's PARTIALs)

| Finding | Verdict | Evidence I checked |
|---|---|---|
| RL-1 | **PARTIAL** | Its remainders: R2L-2 is fixed, R2L-1 is partial (below). |
| RL-5 | **FIXED** | `06_docs/mutants/mutants.json` holds 64 mutants: `answers.go` 4, `reach.go` 2, `lower*.go` 2, `internal/glotec` 3, `internal/limits` 8, `internal/climatology` 3. The leg is full-lane only (`scripts/gate:292`), so I did not watch it run. |
| RL-9 | **RULED** | D-255: the verdict comes at VALIDATE (`validate-agenda.md` item 7). |
| R2L-1 | **PARTIAL** | Mid-burst trios are fixed (`internal/giro/asker.go:189,219`; `TestFailuresAfterAnAnswerAreTheStations`). Three dead stations at the **head** of the rotation still alternate "GIRO unavailable": see R3L-3. |
| R2L-2 | **FIXED** | `asker.go:258-272`: every unnamed non-200 status, and a 410, is the station's failure (`TestAStationsFailureIsItsOwn` covers 400, 451, 302 and 410). watchpost notes the status once (`propagation.go:290-309`). On the wording, see R3L-5. |
| R2L-3 | **FIXED, with a regression** | A 5xx no longer stops the round (`update.go:295-310`), and the D-249 test now asserts GloTEC is drawn (`nomean_test.go:70`). But "no answer goes on" brings in R3L-1. |
| R2L-4 | **PARTIAL** | The mean and GIRO's part are restored independently, and unknown codes are dropped (`state.go:105-116`; `internal/stations/stations.go:344-366`). R2L-4's third action, "name the mean in the note", was not done. The note is now untrue in a new case: see R3L-6. |
| R2L-5 | **PARTIAL** | `Implausible` and `ImplausibleM` are now separate (`internal/assimilate/onto.go:27-28`; `update.go:480`). But "note the count once a burst" was not done (R3L-7). `Rejected` also still carries unparseable replies (`asker.go:245-246`) and moved-station refusals (`:284`), and both are called "rows the range checks refused" (`modes/tty/map_prop_update.go:154`). |
| R2L-6 | **FIXED** | go-tuiMaps `gate_test.go:584-599`: there is a per-run marker in the planted process's arguments, and `pgrep`/`pkill` use that marker alone. It was green in my gate. |

---

##### 1. Happy Path Bias

**R3L-2. If GIRO answers one station and then stalls, the burst runs the host's 60 s deadline out. It is never called down, and is said as "the update stopped before every file came".** Severity: **Minor**.
- D-271 counts failures in a row only before any answer (`asker.go:219`). D-275's 15 s timeout (`:32,234`) was sized so that three hung asks fit in 60 s. But after one answer, a stall goes on until about 4 asks later, when the host cancels the update.
- Probe `TestR3LHangAfterAnAnswer` (`iono/internal/giro/r3l_probe_test.go`), scaled down: 6 asked, 4 failed, outcome Answered, not paused, ended by the context deadline.
- The library then says `Unfinished` (`update.go:113-114`), and watchpost says "the update stopped before every file came" (`map_prop_update.go:369`). That names files, not GIRO's stations.
- `TestAHungGIROIsCalledDown`'s doc ("never run out to its deadline", `asker_test.go:176-178`) holds only before any answer. D-275's own counter-case ("Unfinished, the outcome still answered") survives in this shape.
- Simplify/Delete: N.
- Action:
  - Count the burst's elapsed time against a budget, or call GIRO down on k timeouts in a row whatever answered before (a fork, D-271 against D-275).
  - Make the Unfinished words name GIRO when the burst was the update.

##### 2. Systemic Failure Modes

**R3L-1. Since D-270, a NOAA host that accepts and never answers runs every update out to its deadline. GIRO is then never asked for as long as the stall lasts, and NOAA is not backed off.** Severity: **Important**. Round 2's remediation brought this in.
- D-270 made "no answer" go on to the next file (`update.go:298-300`). NOAA's asks keep the client's 30 s (`platform/httpx/plain.go:79-80`; D-275: "NOAA's asks keep 30 s"), and the update has 60 s (`domains/propagation/propagation.go:46,205`). Two hung asks on `services.swpc.noaa.gov` (index, then D-RAP) reach the deadline.
- `Update` then returns `unfinished` before the GIRO switch (`update.go:140-142` vs `:155-157`), with no `nextAsk` back-off (`:144`).
- Probe `TestR3LHungNOAA` (`iono/r3l_probe_test.go`): per-request 30 ms against a 60 ms deadline, a mean held, GIRO healthy, 4 hours of 10-minute refreshes:

| Tree | Reason | GIRO asks an hour | NOAA asks an hour |
|---|---|---|---|
| `e2103d7` | `Unfinished` every tick | **0** | 18 |
| `8d84569` (control) | Partial / GIROPaused / TooSoon | 40 (40 stations used) | 4 (backed off) |

- A fast refusal (the control `TestR3LRefusedNOAA`) still asks GIRO 40 times. So the trigger is the stall alone.
- What the user reads: "the update stopped before every file came … no station readings this hour: the background alone · foF2 from the climatology". That is D-239's "GIRO is asked whatever NOAA answered" broken, and RL-2's class ("one NOAA problem blinds another source") back again.
- No test has a stalling NOAA.
- Simplify/Delete: N.
- Action, a fork for the HUM LEAD (D-270 with D-275):
  - Give NOAA's asks a per-ask budget so the round fits the update (e.g. 12 s each).
  - Or stop the round on a timeout, while still going on after a 5xx.
  - Add a test with a stalling fake that asserts GIRO is asked and NOAA backs off.

R3L-1 and R3L-2 share a root: no remediation checked per-ask timeouts × asks against the host's one deadline. Nothing in the release says the two sources fail together through it.

##### 3. Gate Validity

**R3L-4. No test holds D-260's back-off for an unanswered NOAA ask, which D-270 says "still applies".** Severity: **Minor**.
- My mutant dropped `err != nil` from `l.trouble` (`update.go:297`). In the copy, with the rest of the package passing, only the anchor test noticed (`TestEveryMutantFindsItsLine`, text moved) and no behaviour test failed. The `.git`-dependent repo tests could not run in the rsync copy.
- The "down" case in `TestARefusalBacksOff` checks only the first back-off (`update_test.go:424-436`), and that equals the healthy 10-minute cadence. The 15-minute step is tested only for 429 and 503.
- m008 holds the 5xx; nothing holds the transport error.
- Simplify/Delete: N.
- Action: assert the second back-off (15 min) after two unanswered updates, and add the mutant to `mutants.json`.

Otherwise, each new mutant's row names a test that exists: m029 to m032, m037, m041 and m064, and watchpost's m558 to m562. finaltag now reads go.mod through `go mod edit -json` and refuses a replace (`tools/finaltag/main.go:60-99`). runlog fails closed on every record error (`tools/runlog/main.go:81-110`).

##### 4. Adversarial Input

**R3L-5. watchpost's D-272 note calls a 500, 404 or 410 a status the host "is not known to send", and also fires for NOAA hosts.** Severity: **Minor**.
- `namedStatus` (`propagation.go:290`) holds only 200/304/401/403/429. Its own comment says these are "the statuses D-247 names", but D-247 names 5xx and 404 too.
- `TestAnUnnamedStatusIsNotedOnce` (`propagation_test.go:124-140`) asserts a 500 is noted.
- A NOAA 404 is "the file missed", not "counted failed (D-272)".
- Simplify/Delete: N.
- Action: name what is unknown (anything outside D-247's set), and word the note for NOAA's hosts.

Otherwise:
- `Apply` drops unknown codes and still refuses duplicates and future times (`stations.go:344-366`).
- The `seen` map is bounded by host × status.
- go-tuiMaps caps reply headers at 64 KiB (`internal/fetch/fetch.go:141,164`).
- `DaylightAt` refuses NaN and off-globe places (`daynight.go:43-46`), and watchpost handles `!Known` (`map_prop_words.go:125-127`).
- A GIRO serving 200 with garbage for every station is counted as answered and resets the back-off (`asker.go:243-247`). That is bounded by the 40-ask burst; its wording is covered under R2L-5.

##### 5. Graceful Degradation

R3L-1 is the main one: during a NOAA stall the words say what happened to the files, but not that GIRO, which is healthy, went unasked.

**R3L-3. Three dead stations at the head of the rotation make the map say "GIRO unavailable this hour", and back GIRO off, every other hour for up to 72 hours.** Severity: **Minor**.
- When every live station fits in one burst, `pollAt` advances by n, so the head never moves (`stations.go:166-169,207`). After a stop, `Unasked` puts the other 36 first, so the next burst completes with the trio last. The burst after that starts at the trio again.
- Probe `TestR3LDeadTrioAtTheHead` (`iono/internal/giro/r3l_trio_test.go`), over 8 hours: asked 3 → 40 → 3 → 40…, outcome Unavailable and paused on the even hours.
- This is the "rotate the failed ones to the back" option D-271 did not choose. That makes it a recommendation to the HUM LEAD, not a defect against the ruling.
- Simplify/Delete: N.
- Action: on a stop, queue the stations that failed behind the unreached ones and behind the next rotation's head, and add the test.

**R3L-6. If only the kept mean is refused, watchpost says "GIRO's throttle state was not taken, so it starts as new", although GIRO's state was taken.** Severity: **Minor**.
- `Restore` returns `ErrBadState` wrapping `errors.Join(errF, errG)` (`state.go:112-115`). watchpost's single note (`propagation.go:134`) predates the split.
- When both parts fail, `errors.Join` puts a newline into one diagnostics line.
- Simplify/Delete: N.
- Action: have the library say which part it refused (two sentinels, or a typed error), and word the note for each.

##### 6. Test Confidence vs. Test Results

**R3L-7. Each snapshot of the hour notes one burst's counts again, and the notes crowd the once-only notes out of the 50-slot ring.** Severity: **Minor**.
- `applyPropUpdated` notes `Rejected`, `Implausible` and `ImplausibleM` on every applied snapshot, too-soon replies included (`map_prop_update.go:153-161`). The ring keeps 50 (`app/mapproblems.go:16`) and is shared with D-272's status note and the denial note (`app/mapprop.go:113`).
- Probe `TestR3LNotesRepeat` in my `wp/` clone, since removed: **21 refusal notes** for one burst's counts over one hour of 10-minute refreshes. Batch 66 added two of the three.
- `TestEveryRefusedReadingIsNoted` checks that the words are present, not how often they come.
- Simplify/Delete: Y (note on change only).
- Action: note a count when it changes from the snapshot held, and test the frequency.

The rest of what I sampled proves behaviour, not fixtures:
- The D-249 test now asserts the background.
- The D-271 test asserts every station asked, 3 failed and no pause.
- The states walk (`map_prop_states_test.go`) still drives the real library.

##### 7. Dependency Fragility

**R3L-8. watchpost matches the literal `"solar"` in `Inputs.Unread` (`map_prop_update.go:377`), though rc.7 exports `ionomaps.UnreadSolar` for this (A-30 (4), R2C-5).** Severity: **Minor**.
- A rename in the library would drop D-249's "flux held from N days ago" in silence.
- Simplify/Delete: Y.
- Action: use the constant.

Also:
- The pins match the reviewed code: `go.mod:9-10` names rc.7 (`57bd7ef`) and rc.6 (`474797a`). `e2103d7` adds only docs.
- R3L-1 is itself a coupling across a dependency: the library's round assumes nothing about the host's per-request timeout, but its correctness depends on it.

##### 8. Problem statement present, real, honored

PS-1 and PS-2 are present and locked (`problem-statement.md:14-31`).
- **PS-2**, as measured: M1 53.7 % against its 57.4 % target (the climatology 52.4 %) and M4 68.4 % against 69.4 %, both sent to VALIDATE (`validate-agenda.md` item 1, D-250).
- **PS-1**: a hypothesis (D-217), claimed as answerability alone. The Broadcaster's cost arm is under way, ending about 2026-10-27.
- R3L-1 bears on both. During a NOAA stall the field falls to the climatology with no stations: the one input whose measured gain PS-2 rests on, the station readings, is dropped while it is available.
- No new finding of this question's own.

---

##### Verified clean

- R2L-2, R2L-3 (apart from R3L-1) and R2L-6 hold in code and tests. RL-5's encoding covers each area RL-5 named.
- D-272's 410 as the station's own. D-279's fold (`onto.go:158-167`), measured against both backgrounds (A-31).
- `heldGrid`, `gridUsable`, `drapUsable` and `scalesUsable` are one rule each, used everywhere a grid, table or scales' age is asked.
- finaltag's parser and replace refusal, runlog's fail-closed record, perfsum's `g1FullHours`.
- The quick gates, lint legs and race runs listed above.

##### Opinion

**Do not exit REVIEW.** Nothing is Critical. But R3L-1 is a regression brought in by round 2's own fix (D-270), with a probe that reproduces it against a control. In the case it covers, it removes GIRO from the map and NOAA's back-off, and the words do not say so. It is the same class this lens held REVIEW open on in round 2, and its fix is small and testable. The rest (R3L-2 to R3L-8) are Minor and could be batched.

**Fix first: R3L-1.** Fit NOAA's round inside the update's deadline, and add a stalling-fake test asserting GIRO is still asked and NOAA backs off.

---

## Code Quality, round 3 (R3C): REVIEW exit red team, 0.19.0

**Verdict: do not exit REVIEW yet.** There is one Important, R3C-1: D-275's per-ask timeout and D-271's "down only before an answer" rule combine badly. A GIRO that answers once and then hangs runs every burst to the host's deadline and is never backed off, which is exactly the outcome D-275 was ruled to prevent. Everything else is Minor. Round 2's R2C findings hold in the code apart from half of R2C-5.

### Tree state and runs
- **Tips:** watchpost `03ad4e97`, go-ionomaps `e2103d7`, go-tuiMaps `474797a`.
- **`git status --short` at the start:**
  - watchpost: ` M .../02-analysis/rulings.md`
  - go-ionomaps: clean
  - go-tuiMaps: clean
- **During my review both libraries gained ` M 06_docs/gate-runs.md`.** The new rows are 01:27:18Z and 01:34:55Z. They are not mine: I ran both gates on byte-copies in my scratch directory, and my rows (01:45:58Z, 01:53:31Z) are in the copies only, with the same tree hashes as the tips.
- **I edited no tree.** Every mutant was applied with `go test -overlay` from scratch copies. Every finding below was reproduced against those tips in those states.
- **watchpost, all exit 0:**
  - `make lint` (6 baselined), `make lint-authoring`, `make p10` (0 live; p10floor holds 23 rows), `make dupes` (2 groups, both ratified), `make mutant-anchors` (814), `go vet ./...`.
  - `go test -race` on modes/tty, app, domains/{propagation, locations/openmeteo, radio/synth} and tools/{finaltag, p10floor, perfsum, runlog}.
  - cmd/watchpost's gate and record tests.
- **go-ionomaps and go-tuiMaps `scripts/gate --quick` (copies): green, P10 included.** Round 2's go-tuiMaps P10 failure did not recur.

### Part 1: round 2's findings (and round 1's PARTIALs), checked in the code

| ID | Status | Evidence |
|---|---|---|
| R2C-1 | FIXED | `map_prop_ahead_test.go:214-246`. The `built: 0` mutant is killed by `TestARebuildAndBackIsAnsweredAgain`. m430 is re-pointed and m547 added. |
| R2C-2 | FIXED | `map_prop_update_test.go:244-271`. Dropping `NoStations` from `madeWith` is killed. |
| R2C-3 | FIXED, with an uneven shape | go-ionomaps `update.go:217-254` (see R3C-11) |
| R2C-4 | FIXED (D-277) | `tools/p10floor/main.go:81-83,103-106` |
| R2C-5 | PARTIAL | The doc is fixed and the names are exported (`snapshot.go:121-126,240`). watchpost still matches `"solar"` by hand: R3C-5. |
| R2C-6 | FIXED | All five setter mutants are killed: `look.go:133,163,187,206,238`. |
| R2C-7 | FIXED | Removing the age gate (`update.go:325`) is killed by `TestAStaleIndexFetchesNoGrid`. |
| R2C-8 | FIXED | Putting back `shown.Now.At` (`map_prop_words.go:113`) is killed. |
| R2C-9 | FIXED | `answers.go:247-249,443-458`. Dropping the origin's check is killed. |
| R2C-10 | FIXED | go-ionomaps `onto.go:88`; watchpost `map_prop_update.go:156-161`. Both mutants are killed. |
| RC-3 | FIXED in code, partly pinned | `runlog/main.go:83-108`. Two of the error paths are untested: R3C-8. |
| RC-7 | FIXED | `hasKey` and `stillOrphan` are deleted. A-91(6)'s reasoning holds: the start-time match is the guard. |
| RC-9 | FIXED | `gates_test.go:618-629`. The `-race` branch is unpinned: R3C-12. |
| RC-13 | FIXED | go-ionomaps `p10_mirror_test.go:110-113` |

**Round 2's unnumbered Minors were never recorded:**
- `finaltag`'s `replace` gap: FIXED.
- reachscore's "different numbers of scenarios" check written twice: NOT FIXED (`tools/reachscore/report.go:115-118`, `:127-131`).
- `propMadeOf`'s name: NOT FIXED (`map_prop_update.go:91-95`).
- go-tuiMaps' hand-listed alias field lists: still untested (`overlays.go:39-60`, `describe.go:15-21`). Only their IDs were stripped.

### 1. Understandability
- **R3C-6 (Minor), go-ionomaps:** `nextAsk` now receives `l.trouble`, but its parameter is still named `refused` and its doc still says "after an update without a refusal" (`update.go:276-281`, called at `:144`).
  - `trouble` covers a refusal, a NOAA-side failure or no answer.
  - `ask`'s doc does not say that it sets `l.trouble` (`update.go:289-297`). Its back-off travels through a field the reader cannot see from the call site.
  - Simplify/Delete: Y. Action: rename the parameter `trouble`, and say the side effect in `ask`'s doc.

### 2. Maintainability
- **R3C-2 (Minor), watchpost against go-ionomaps rc.7:** rc.7's `Restore` now takes GIRO's part and the F10.7 mean each on its own. It returns `ErrBadState` when either is refused, even though the other was taken (`state.go:72-116`).
  - watchpost was re-pinned to rc.7 but still writes "GIRO's throttle state was not taken, so it starts as new" for any error, and its doc still says "the library starts as new" (`domains/propagation/propagation.go:113-116,134`).
  - When only the mean is refused, the note is false. When GIRO's part is refused, the mean kept or lost goes unsaid.
  - Simplify/Delete: N. Action: the library names which part it refused in a form a host can test (a sentinel for each part); watchpost words each case; add a test.
- **R3C-11 (Minor), go-ionomaps:** R2C-3 asked for helpers "written as `solarUsable` is". There are four shapes instead (`update.go:209-254`):
  - `solarUsable` and `drapUsable` are methods;
  - `scalesUsable(held, now)` is a free function;
  - `gridUsable(valid, ok, now)` takes a flag, with a fifth helper, `heldGrid`, on top.
  - Simplify/Delete: Y. Action: one form.

### 3. Complexity
- Nothing beyond R3C-11.
- `sunAt`'s step-skip (`answers.go:416-423`) is sound: the bound holds by derivation. Its exhaustive equivalence test (`sun_test.go:59-92`) kills a too-tight `rise`.

### 4. Necessity
- **R3C-10 (Minor), go-tuiMaps:** `made_work` is set (`scripts/gate:234,664`) but nothing reads it now that the workspace lives outside the tree and `cleanup` keys on `work_dir`. Simplify/Delete: Y. Action: delete it.

### 5. Name semanticism
- R3C-6 (`nextAsk`'s `refused`).
- `propMadeOf` is still misnamed (Part 1).

### 6. Code documentation
- **R3C-4 (Minor), go-ionomaps: D-281 is incomplete.** Red-team IDs remain in exported docs that `go doc` shows a library user:
  - `ionomaps.go:26` "(I-1)". The same edit deleted "watchpost D-83" beside it.
  - `snapshot.go:181`, `Unfinished`, "(L-F4)".
  - `update.go:94`, `Update`'s doc, "(L-F4)".
  - `ahead.go:20` "(L-F2)".
  - I found these with `go doc -all .` grepped for ID patterns. go-tuiMaps' remaining `L-x` and `FR-x` IDs are its requirements, so it is clean.
  - Simplify/Delete: Y. Action: strip them, and add a test that scans `go doc -all` for red-team and ruling ID patterns.
- **R3C-3 (Minor), watchpost:** the comment says "`namedStatus` are the statuses D-247 names" (`propagation.go:287-290`). The set is {200, 304, 401, 403, 429}.
  - D-247 also names 404 and 5xx as a station's own failure (`rulings.md:268`). D-272 says to note only the statuses D-247 does not name.
  - The code notes 404 and 5xx, and `TestAnUnnamedStatusIsNotedOnce` pins a 500 as noted (`propagation_test.go:120-141`).
  - The note runs for NOAA's hosts too, where D-260 names 5xx. There the words "which it is not known to send; that ask is counted failed (D-272)" are wrong on both counts.
  - Simplify/Delete: N. Action: add 404 and 5xx to the named set, or put noting them to the HUM LEAD; then fix the comment and the test.
- **R3C-1's comment** (`asker.go:27-31`) claims "never left to run the update out". That is false; see below.
- **Cosmetic:** two re-wrapped doc comments are ragged after the ID strip: go-tuiMaps `look.go:50` (about 130 columns) and `daynight.go:36-38`.
- **Cross-lens, for the Docs reviewer:** go-ionomaps `requirements.md:75` (R-5.3) still says a 410 pauses GIRO and that three failures in a row stop a burst at any point. That contradicts D-271, D-272 and the code, and the tests it lists now test the opposite.

### 7. Testability

**Mutants (all on overlays from scratch copies):**
- **watchpost, 12 applied, 9 killed.** Survivors: w4, w9 and w10, below.
- **go-ionomaps, 13 applied, 12 killed.** i4 (the per-ask timeout not applied) is killed only by the 10-minute package timeout; see R3C-7.
- **go-tuiMaps, 7 applied, 7 killed.**

**Findings:**
- **R3C-8 (Minor), watchpost:** two of RC-3's error paths survive their mutants.
  - Mutant w4: returning `rc`, not `max(rc,1)`, when the end-of-run tree read fails (`tools/runlog/main.go:85-87`). That is the exact case round 2 described under RC-3.
  - Mutant w10: the same on the line's write or close (`:106-108`).
  - The new test and m556 cover only the `OpenFile` failure and running outside a repository (`main_test.go:120-138`).
  - Simplify/Delete: N. Action: a test for each path, and mutants for them.
- **R3C-7 (Minor), go-ionomaps:**
  - Dropping `err != nil` from `l.trouble` (`update.go:297`) survives the whole root package. So "no answer backs NOAA off" (D-260, D-270) is unpinned; it shows only on two consecutive unanswered updates (10 minutes against 15).
  - `TestAHungGIROIsCalledDown` (`internal/giro/asker_test.go:340-358`) cannot fail by itself. With `askTimeout` not applied, its fake waits on a context that never ends, so the package hangs for 10 minutes and panics. Its `took` check is never reached.
  - Simplify/Delete: N. Action: a two-update back-off test, and a guard deadline in the fake.
- **R3C-12 (Minor), watchpost:** the selector's `-race` branch (`gates_test.go:626,662`) is unpinned. Deleting it survives, since no case tests it. Action: one case with a race-tagged test, or delete the branch.

### 8. P10 conformance
- `make p10`: 0 live, 23 floors held. D-277's floorless failure is built and self-tested (`p10floor/main.go:136-150`).
- **Recommendation to the HUM LEAD (evasion class, not a finding, since D-283 ruled it):** `modes/tty`'s floor was lowered from 0.43 to 0.42 at the first batch after D-264 pinned it.
  - The cause was the metric, not lost checks. Deleting `hasKey` removed a counted loop guard, and R2I-2's fix pulled two old renderers into the changed set.
  - A floor renewed downward by a security fix suggests the density rule is punishing the wrong thing. That is worth a ruling before the next renewal.

### 9. Evidence soundness
- **R3C-1 (Important), go-ionomaps: a hung GIRO is called down only before its first answer, so D-275's goal fails.**
  - `askTimeout` (`internal/giro/asker.go:27-32`) is justified as "failuresInARow of them fit inside a host's update… never left to run the update out".
  - But `stops` ends a burst on failures in a row only while `!anyAnswered` (`:219`, D-271).
  - **Probe:** an overlay test in `internal/giro` where the first station answers 200 and every later ask hangs, with `askTimeout` set to 20 ms. Result: 40 asked, 39 timeouts, outcome `Answered`, not paused, 39 failed. At the real 15 s that is 9 min 45 s against watchpost's 60 s update.
  - **Effect:** the update is cancelled at its deadline and said `Unfinished`. The library's lock is held the full 60 s, which is D-275's own counter-argument. Nothing is backed off, so GIRO takes up to 40 hanging connections every hour.
  - The fix is new code. The two ruled changes interact, and no test covers the case.
  - Simplify/Delete: N. **Fork for the HUM LEAD:**
    1. Timeouts in a row call GIRO down whatever answered.
    2. A burst deadline under the host's update.
    3. Keep D-271 as is and correct the comment.
  - Under 1 or 2, add a test of an answer followed by hangs.
- **R3C-9 (Minor), watchpost:** D-276's retry does not fit the leg's own budget.
  - Each render has a 2-minute bound, and a run can make the control render, the hostile render, the re-check render and one retry: 8 minutes. The leg's `-timeout` is 6m (`Makefile:110`; `synth_test.go:810-826`; A-91(4)).
  - A run that retries and then renders slowly is killed by go test's timeout panic, not by the gate's diagnosed message.
  - `hung` is read from `ctx.Err()` after the render returns, so a render that finishes with its audio just as the bound passes is thrown away and retried.
  - Action: set `-timeout` from the bounds, or bound the run as a whole; take a render as hung only when it returned an error.
- `TestEveryUpdateOptionIsMadeWith` builds its snapshot with the same mapping as `madeWith`, so it acts as a tripwire for a new option, not an oracle. That is adequate for R2C-2's purpose.

### 10. Evasion lens
- **R2C-5's constant export was claimed as closing the untyped contract** (record: "the `Unread` names exported"), but the consumer still uses the literal: **R3C-5 (Minor)**, `modes/tty/map_prop_update.go:377` matches `"solar"`. Simplify/Delete: Y. Action: `ionomaps.UnreadSolar`.
- Round 2's unnumbered Minors were never carried into the record (Part 1), so "every round 2 finding closed" (D-284) does not include them.
- R3C-3: a test pins behaviour that contradicts the ruling it cites.
- R3C-8: m556's "Killed by" is true but narrower than RC-3's claim, "a run with no record exits failing".

### What I verified clean
- **watchpost:**
  - `finaltag` reads go.mod through `go mod edit -json` and refuses a `replace`; the release-step test.
  - `placeWithZip` and the stored-postcode test.
  - the geocoder refuses `Local`.
  - the tile transport's header cap, and its tie to the library's source.
  - perfsum's 2-minute tolerance at its boundary (2 minutes passes, 3 fails).
  - the three refused-reading sentences.
  - `stillOrphan`'s deletion: a reused PID is still refused by its start time.
  - the build-constraint evaluation in the run-selector check.
- **go-ionomaps:**
  - D-270's round going on after a 5xx, with the grid used.
  - D-272's status classes.
  - the fold bound and `withinFold`'s NaN and zero guards.
  - the solar file's line cap failing closed.
  - the station list dropping retired codes; `Import` is atomic.
  - the year-fraction cache keyed to the exact moment.
  - the lazily made typical rows (CAS keeps the first row made).
  - the late day served without evicting today's.
  - `checkPlaces`.
  - the docs-lane refusal logging nothing.
- **go-tuiMaps:**
  - `DaylightAt`'s off-globe guard, consistent with `view.go:42`.
  - the header cap.
  - the gate's private workspace.
  - the marker-scoped kill in the stubborn-leg test.
  - every look setter's re-check.

**Fix first: R3C-1.** It is the one place where round 2's remediation brings back the failure it was ruled to remove: a GIRO that hangs after one answer holds the library for the whole update every hour and is never backed off.

Scratch, with every mutant, the overlays and the probe: `<scratch>`
