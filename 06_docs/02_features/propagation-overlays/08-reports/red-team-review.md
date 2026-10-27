---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — REVIEW exit red team, rounds 1 to 7"
date: 2026-10-24
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1: seven blind reviewers (D-245), 17 forks ruled (D-246 to D-262), remediated in watchpost batches 59 to 65 and the libraries' candidates. ROUND 2 REPORTED 2026-10-25 (D-269): five exit, two do not; 13 forks ruled (D-270 to D-282); remediated in watchpost batch 66, go-ionomaps v0.1.0-rc.7 and go-tuiMaps v0.3.0-rc.6. ROUND 3 REPORTED 2026-10-26 (D-284): three reviewers, none exit, no Critical; five forks ruled (D-285 to D-289); remediated in watchpost batch 67, go-ionomaps v0.1.0-rc.8 and go-tuiMaps v0.3.0-rc.7. ROUND 4 REPORTED 2026-10-26 (D-290): three reviewers, Docs exit once two Minors are fixed, the REVIEW lens and Code do not, one Important between them; three forks ruled (D-291 to D-293); remediated in watchpost batch 68, go-ionomaps v0.1.0-rc.9 and go-tuiMaps v0.3.0-rc.8. ROUND 5 REPORTED 2026-10-26 (D-294): three reviewers, Docs exits, the REVIEW lens and Code do not, two Importants both in D-291's deadline rule; five forks ruled (D-295 to D-299); remediated in watchpost batch 69, go-ionomaps v0.1.0-rc.10 and go-tuiMaps v0.3.0-rc.9. ROUND 6 REPORTED 2026-10-26 (D-300): four reviewers, the REVIEW lens and Code exit, Docs and the timing lens do not, two Importants confirmed - neither in the library's timing code, which held under every timing the timing lens built; seven forks ruled (D-301 to D-307); remediated in watchpost batch 70, go-ionomaps v0.1.0-rc.11 and go-tuiMaps v0.3.0-rc.10; rc.11's hosted amd64 find fixed in rc.12 and batch 71. ROUND 7 REPORTED 2026-10-27 (D-308): four reviewers, each on amd64 too, none exit, one Important found by all four apart - D-302 and D-307 disagreeing for a station that ran out and was then not reached; three forks ruled (D-309 to D-311), the model made and approved (D-312); its remediation follows"
---

# REVIEW exit — red team, rounds 1 to 7

## Round 1 (D-245)

Up: [rulings](../02-analysis/rulings.md) · [build log](../04-development/build-log.md) · the reports in full: [review-round-1](reviewer-notes/review-round-1.md)

## Summary

- **Seven blind reviewers (D-245)** over the whole release: watchpost v0.18.2 (`a7e77990`) to `6ad1c098`, go-ionomaps from its scaffold (`8ed1f9b3`) to `d80d8db7`, go-tuiMaps v0.2.1 (`043922ec`) to `544b889d`. **Four say do not exit REVIEW** (Code, Docs, Business, the REVIEW lens); **three say exit** (InfoSec, Performance, Hygiene). **No Critical.** Every reviewer found the trees clean at the start and at the end; one, Docs, dirtied `06_docs/verify-runs.md` with a refused `make verify-docs` and reverted it (RD-12).
- **The library mislabels a GloTEC outage, found by four reviewers on their own** (RB-2, RI-6, RL-6, RC-11): when NOAA's index answers but lists no grid fresh enough (stalled, reformatted, or past the clock), `refreshGrid` returns without naming GloTEC (go-ionomaps `update.go:233-235`). The snapshot says `Fresh` while its background falls to the climatology: D-239's class ("we should never have data mislabelled").
- **One failing GIRO station blinds the stations for every hour** (RL-1): any 5xx or transport error on one station's ask is taken as GIRO refusing (`internal/giro/asker.go:152`), and with every live station in one burst the rotation never moves (`internal/stations/stations.go:167`), so that station is asked first, and stops the burst, each hour. A 403 is taken as an answer and resets the back-off (`asker.go:157`).
- **A cold start without NOAA's solar file draws no map and says `Offline`** whatever NOAA answered, naming nothing unread and never asking GIRO (RL-2, `update.go:139-140`).
- **An hour ahead asked before a rebuild is kept after it** (RC-4, `modes/tty/map_prop_ahead.go:66`): BUILD's L-F3 class, closed for the chart, the centre and the reach but not the hours ahead.
- **The record shows G1 met without the comparison D-236 said would be reported** (RB-5, RD-1, RL-3): against the map never opened, the 48-hour run measured +3.42 % CPU and +40.0 MB RSS, both missed; that summary is outside the tree only. Whether G1 stands is the HUM LEAD's (fork 1).
- **The changelog says the memory fix is done** before its proof run ends (RB-6, RD-2, RL-9).
- **An honesty wording the HUM LEAD ruled is missing:** beyond 2000 km the words say "about as good as typical", not D-129's "or worse" (RB-1).
- **An unbuilt plan task:** go-ionomaps' R-9.8 (each constant with its basis, and a release step failing past 12 months) has no test, and G10.5 appears only in the plan (RD-3).
- **The agent's error:** the floor commits took no new release candidate or re-pin, against D-206, on the agent's own reasoning; watchpost builds tags that still say `go 1.26.9` (RH-10, RL-10, RI-8).

## The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Code Quality | Do not exit | RC-4: the hour ahead kept against a rebuilt snapshot |
| Project Hygiene | Exit | RH-9: nothing refuses a release on the libraries' candidates |
| Docs Quality | Do not exit | RD-1: G1's reported comparison never reported |
| Business Quality | Do not exit | RB-5 with RB-4: the same, and the Broadcaster never measured |
| REVIEW lens | Do not exit | RL-1: one station's failure blinds the stations every hour |
| InfoSec | Exit | RI-1: three GIRO replies move the whole globe's field |
| Performance | Exit | RP-1: each hour ahead copies what the rulings say it shares |

## The forks for the HUM LEAD, in order

Each is put singly, with its evidence, options, a recommendation and the strongest counter-argument, and recorded verbatim as a D-row.

1. **G1's comparison** (RB-4, RL-3; D-236): stand, with both readings put to VALIDATE; or re-rule against the map never opened; or a reference arm with the map open and no layers; and whether a Broadcaster arm is measured.
2. **GIRO's failures** (RL-1; D-39): a station's 5xx or 404 that station's alone, the burst stopped only on a 429, on several failures in a row, or on 401/403/410 paused long; "GIRO failed this hour" carried all hour.
3. **GIRO's plausibility** (RI-1; D-101, D-61): seeded stations kept at their seed positions; readings refused beyond a residual against GloTEC; the mean offset decayed with distance or said.
4. **No solar file** (RL-2; RK-15): the last F10.7 mean kept in the state file; foF2 drawn from GloTEC without it.
5. **PS-1's accuracy** (RL-4, RB-3): the PR's words for PS-1, M1's scope, a recorded line that the release ships without an accuracy floor, and whether the chart says its measured agreement.
6. **The hours ahead: shared or owned** (RP-1): P-6 ("a snapshot in slices of its own") against D-131 (shared tables).
7. **The P10 ledger and its package rows** (RL-8, RH-1, RH-5, RC-6): kept on one disk, run by no CI; package-wide P10-05 rows now over code they never described.
8. **The default refresh** (RB-10; D-94): every 10 minutes at 13.94 MB an hour, or hourly at 2.66, with no measured accuracy between them.
9. **`make test-say`'s skip** (RC-2; F-52): a required step that skips green where `say` renders nothing.
10. **Known limits in the changelog** (RB-7).
11. **Moving between targets and FR-1.14** (RD-4; D-61, D-62).
12. **The REQUIRED-READING's read order** (RD-13).
13. **One sun** (RC-16): go-tuiMaps' `Daylight` and `sun.Turn` unused, watchpost's own model beside them.
14. **The owed wires rows** (RH-11): owed to 0.16.0's P4.
15. **NOAA's back-off floor** (RL-7; D-39): a refused NOAA asked more often than a healthy one.
16. **M5 beyond 400×110** (RP-3).
17. **go-tuiMaps' `main` red until the merge, and a go-ionomaps nightly** (RH-7).

## The agent's remediation, without a ruling

Each finding below is confirmed (the table that follows), or is checked before it is fixed; batched after the forks, each library change on its own gate and a new release candidate, watchpost re-pinned (D-206), hosted CI read before the next.

- **go-ionomaps, v0.1.0-rc.6:** GloTEC named unread when the index lists no usable grid (RB-2, RI-6, RL-6, RC-11); the rotation moved by the stations answered (RL-1, its defect half); the no-field snapshot carrying what was unread, `Offline` only when nothing came back (RL-2, its labelling half); a small clock skew clamped, not a state refused (RL-11); the typical days evicting the oldest alone (RP-2); the hour ahead built without the copy it overwrites, and typical hours allocated as asked (RP-1, its code half); the solar file's duplicate days, scanner error and size bound (RI-3); a saved state repeating a station refused (RI-5); `ionomapstest` faults per address (RL-12); a re-runnable mutant harness, the hand mutants re-encoded (RL-5); G10.5 built (RD-3); the gate's toolchain and P10 skip (RC-12, once checked); RC-13; the README, the build section and the evidence's `go.sum` (RB-9, RH-2, RH-3); `api-review.md` and the plan's figures (RD-7, RD-8); BUILD exit ticked (RH-12); RH-13 noted; stale headers (RH-14); the deletions and wrong comments (RC-7, RC-8).
- **go-tuiMaps, v0.3.0-rc.4 and rc.5:** `DaylightAt`, the sun the night is drawn with at any moment (rc.5, D-263); `Daylight.Turns` (RC-14, once reproduced); the host-ramp warning on a later depth (RC-15); the release notes' sun figure (RD-11); RC-17; the frame writer's buffer (RP-4); BUILD exit ticked (RH-12); `go doc` and the aliased fields (RD-6); RC-8's comments.
- **watchpost:** the hour ahead keyed on `built` (RC-4); "or worse" (RB-1); `perfsum` failing on an unparsable number and the driver's failure carried (RC-1); `runlog` taking the tree before and after (RC-3) and logging no refused run (RD-12); both G1 comparisons in the record, the summary's "dry run" line reconciled with D-241 (RB-5, RD-1); the changelog worded as what changed until the proof run's figures are committed (RB-6, RD-2, RL-9); `reachscore`'s blind spots in its output and no verdict where no figure stands (RB-8, RC-9); the docs (RD-7, RD-9, RD-10, RH-1, RH-14); the geocoder's postcode, zone and coordinates (RI-2); a response-header cap and long validators dropped (RI-4); the state file named, not its path (RI-7); 0.19.0's gate roster and 0.18.0's status line (RH-6); the hook's message (RH-8); a test refusing a final tag on any `-rc` module (RH-9); the session state in one place (RC-5, D-237); RC-7, RC-8, RC-10; the refresh note's comment (RB-10); the centre's name lookup (RP-6).
- **watchpost, batch 60 on the release candidates:** GIRO's outcome said all hour, its denial noted once, a station's failure counted (D-247); the kept flux's age said (D-249); the words' sun go-tuiMaps' `DaylightAt`, `astro`'s `Up` and `NextTurn` deleted (D-258, D-263); the real library walked through its states into the words (RL-12).
- **watchpost, batches 61 and 62:** every density row held to its floor (D-264); the centre's name from a band of latitude (RP-6); `platform/history` in files by concern and the answers' arms written out (RC-10); batch 63, the snapshot's one key and the state by lifetime, a find ending with its session (RC-5).
- **The ship-together state (the agent's error, RH-10):** go-ionomaps rc.6 and go-tuiMaps rc.5 tagged at their gated tips, watchpost re-pinned and gated (RL-10, RI-8).

## The agent's check of each finding

Against the clean tips. CONFIRMED: the cited lines do what the finding says. PLAUSIBLE: the claim is consistent with the code, the repro still owed. FORK: a fact confirmed, its remedy the HUM LEAD's. Not yet checked: checked before its fix, and recorded here if it does not hold.

| Finding | Verdict | What was checked |
|---|---|---|
| RB-1 | CONFIRMED | FR-5.5 (`requirements.md:144`) and D-129 say "about as good as typical there, or worse"; `modes/tty/map_prop_words.go:261,263` say "about as good as typical". D-198 ruled the legend's "~typical"; the legend stands. |
| RB-2, RI-6, RL-6, RC-11 | CONFIRMED | go-ionomaps `update.go:233-235`: `if !listed { return false }` names nothing unread. |
| RB-3 | CONFIRMED (fact); fork 5 | M1 53.7 % against the climatology's 52.4 %, the target 57.4 % (`m1-m4-instrument.md:247`). |
| RB-4, RL-3 | CONFIRMED (fact); fork 1 | The 48-hour summary: against the map never opened +3.42 % CPU, +40.0 MB, both missed; every arm an Observer. |
| RB-5, RD-1 | CONFIRMED | No file in the tree carries those figures. |
| RB-6, RD-2, RL-9 | CONFIRMED | `CHANGELOG.md:32` claims the fix; the proof run ends 2026-10-26. |
| RB-7 | CONFIRMED (fact); fork 10 | |
| RB-8 | CONFIRMED | `tools/reachscore` writes no blind spot (INST-5). |
| RB-9, RD-5, RH-4 | CONFIRMED | go-ionomaps `README.md:8` says rc.3; watchpost `go.mod:9` pins rc.5. |
| RB-10 | CONFIRMED (comment); fork 8 | `map_prop_settings.go:60` says "measured cost"; the notes are D-141's ceilings. |
| RD-3 | CONFIRMED | `TestEveryConstantCarriesItsBasis` exists nowhere; G10.5 only in go-ionomaps `implementation-plan.md:134`. |
| RD-4 | PLAUSIBLE; fork 11 | `map_prop_chart.go:476-479` literal up/down inside the chart window; D-61/D-62 give an open window its arrows. |
| RD-6 | Checked by round 2 | Its verdicts and where each gap closed: "Round 1, closed", below. |
| RD-7 | CONFIRMED | `architecture.md:133` names `Refused`; `:43` names `Library` methods `Fields, Bands, Reach` it does not have. |
| RD-8 | CONFIRMED | go-ionomaps `implementation-plan.md:134` against `ahead.go:61-63`. |
| RD-9 | CONFIRMED | `problem-statement.md:64` "50 MB RSS"; `:98` "M1 and M4's baseline is not yet measured". |
| RD-10 | CONFIRMED | `README.md:516` cites go-tuiMaps D-28 (1.26.9); 1.26.10 is its D-38. |
| RD-11 | CONFIRMED | go-tuiMaps `release-notes.md:27` 0.011° over 2020 to 2035; `internal/sun/sun_test.go` holds 0.5° over a 2026 year. |
| RD-12 | CONFIRMED | `Makefile:302-303`: runlog wraps the lane that refuses. |
| RD-13, RD-14 | Forks 12; A-row | |
| RI-1 | CONFIRMED (mechanics); fork 3 | `internal/stations/stations.go:233-237` the reply's position overwrites the seed's; `internal/assimilate/assimilate.go:76` the fit centres on the mean residual. |
| RI-2, RI-3, RI-4, RI-5, RI-7 | Checked by round 2 | Its verdicts and where each gap closed: "Round 1, closed", below. |
| RI-8, RL-10, RH-10 | CONFIRMED; the agent's error | D-206 (`rulings.md:227`): "each later library change takes a new release candidate and a re-pin"; `build-log.md:732`: "watchpost needs no re-pin". |
| RL-1 | CONFIRMED | `internal/giro/asker.go:152` any status of 500 or more, or a transport error, stops the burst as GIRO's refusal; `internal/stations/stations.go:167` `pollAt += len(poll)`; `asker.go:157` a 403 resets the back-off. |
| RL-2 | CONFIRMED (labelling); fork 4 | `update.go:139-140` a bare `Snapshot{Reason: Offline}`. |
| RL-4 | Fork 5 | |
| RL-5 | CONFIRMED | go-ionomaps tracks no mutant file. |
| RL-7 | CONFIRMED (fact); fork 15 | `update.go:200-206` the first back-off 60 s, under the 10-minute cadence. |
| RL-8, RH-1, RH-5 | CONFIRMED (fact); fork 7 | `.a2dh-p10-exemptions.yml` ignored in all three repositories; no workflow runs P10. |
| RL-11 | CONFIRMED | go-ionomaps `internal/throttle/throttle.go:124` refuses a burst a second after now. |
| RL-12 | CONFIRMED | `ionomapstest/replay.go:106` D-RAP and the scales always missing; no fault option. |
| RP-1 | CONFIRMED (code); fork 6 | `ahead.go:47` `s.clone()`; `update.go:446-451` copies what the hour ahead then replaces; `typical.go:67-69` all 24 hours' cells up front. |
| RP-2 | CONFIRMED | `typical.go:57-58` a third day empties the cache. |
| RP-3 | CONFIRMED (benchmarks' shape); fork 16 | |
| RP-4, RP-5, RP-6 | Checked by round 2 | Its verdicts and where each gap closed: "Round 1, closed", below. |
| RC-1 | CONFIRMED | `tools/perfsum/main.go:265` an unparsable number is 0; `scripts/quality/workload.sh:137` a driver's failure exits 0. |
| RC-2 | Fork 9 | |
| RC-3 | CONFIRMED | `tools/runlog/main.go:43-44` the tree is read after the run. |
| RC-4 | CONFIRMED | `map_prop_ahead.go:66` on `Computed` alone; the chart's key carries `built` (`map_prop_chart.go:22`). |
| RC-5 | CONFIRMED (structure); remediated under D-237 | |
| RC-6 | Fork 7 | |
| RC-7, RC-8, RC-10 | Minor | RC-7 was recorded here as fixed while `hasKey` and `stillOrphan` stood; round 2 found them (R2C), deleted in batch 66. |
| RC-9, RC-13, RC-15 | Checked by round 2 | Its verdicts and where each gap closed: "Round 1, closed", below. |
| RC-12 | PLAUSIBLE | go-ionomaps `scripts/gate:80` takes `GOTOOLCHAIN` from the environment; whether hosted CI sets it is read from its log. |
| RC-14 | PLAUSIBLE | go-tuiMaps `describe.go:74-88` keeps the report until a place's elevation crosses 0; a past turn needs the turn and the elevation to disagree near the horizon. |
| RC-16 | Fork 13 | |
| RC-17 | Minor | |
| RH-2, RH-3, RH-12, RH-13, RH-14 | CONFIRMED (the reviewer ran them) | |
| RH-6 | CONFIRMED | No `07-readiness/gates.md` for 0.19.0. |
| RH-7 | Fork 17 | |
| RH-8 | CONFIRMED | `tools/prehook/main.go:36-38` names `make hooks` where it does not exist (D-124). |
| RH-9 | CONFIRMED | Nothing refuses a final tag on an `-rc` module (D-3). |
| RH-11 | Fork 14 | |

## D-252's list: the package rows whose code has grown past their reasons

Put to the HUM LEAD in one batch, as D-252 rules: each row's reason then, its code now, and the figure
pinned as a floor once ratified. Measured by `a2dh p10 check` on 2026-10-24, changed functions counted from
each branch's merge base with `main`.

| Repository · row | Its reason's figure | Now | Why it grew |
|---|---|---|---|
| watchpost · `modes/tty` | none: ratified by precedent 2026-09-08, for the Dashboard's renderers | 0.43 over 335 | the whole Propagation mode, its chart, its words and its keys |
| watchpost · `app` | none: "the bubbletea model plumbing" | 0.61 over 49 | the propagation holder, its sources and its diagnostics |
| go-ionomaps · root | 0.69 over 101 (D-74, pinned) | 0.70 over 112 | the remediation's `short`, `solarUsable`, the constants, the hours' lazy cells |
| go-ionomaps · `internal/giro` | 1.44 over 16 | 1.21 over 19 | D-247's per-station failures, D-248's place check |
| go-ionomaps · `internal/stations` | 1.50 over 12 | 1.33 over 15 | D-247's unreached stations, D-248's `At`, the clock's skew |
| go-ionomaps · `internal/throttle` | 0.75 over 8 | 0.60 over 10 | D-247's day-long denial |
| go-ionomaps · `ionomapstest` (A-26, D-80) | 0.68 over 19 | 0.64 over 22 | RL-12's answers and the two NOAA files it builds |
| go-ionomaps · `tools/gm3` (A-26, D-80) | 0.83 over 18 | 0.95 over 20 | `-residuals` (D-248) |

The rows ratified by precedent with no figure (watchpost's `domains/locations`, `domains/radio/synth`,
`platform/httpx` and the like, go-ionomaps' `internal/climatology`, `internal/limits`, `internal/magcoords`)
would each take their measured figure as a floor under the same ruling (D-264 ratified all of them). Separately for the HUM LEAD: P10's
analyzer legs skip go-ionomaps' three nested modules, and only its density count reaches them, (`tools/tables` and the evidence's two) unless `.a2dh.yml` declares
each a root of its own, a change to that file only the HUM LEAD makes (RC-12).

## The rulings

| Fork | Ruling |
|---|---|
| 1, G1's comparison | D-246: D-236 stands; both readings recorded and to VALIDATE; a 48-hour Broadcaster arm measured and reported; the map's own cost a follow-up |
| 2, GIRO's failures | D-247: a station's failure its own; the burst stops on a 429 or three in a row; 401/403/410 a 24-hour pause, said; the failure carried all hour; the rotation moved |
| 3, GIRO's plausibility | D-248: seeded stations keep their positions (over 50 km refused); a residual bound from the recorded days; G-M3 re-run; D-105's offset stands |
| 4, no solar file | D-249: the last F10.7 mean kept in the saved state, used with its age up to 7 days; past that no field, said as the file unread |
| 5, PS-1's accuracy | D-250: no accuracy floor, recorded; the PR claims answerability for PS-1 with M1's agreement beside it; M1's scope and the chart's agreement at VALIDATE |
| 6, the hours ahead | D-251: P-6 stands, each snapshot owns its slices; D-131 amended; a heap pin over 24 hours ahead under 15 MB |
| 7, the P10 ledger | D-252: the ledger stays local; P10 recorded per phase exit in a roster; the READMEs name the files; the grown package rows re-presented in one list, figures pinned |
| 8, the default refresh | D-253: measured with gm3 on the fresh days (newest grid against one up to an hour old), the default ruled at VALIDATE |
| 9, test-say's skip | D-254: a non-rendering host fails under the leg unless declared mute in the hosted macOS step, which prints NOT RUN; never silent |
| 10, known limits | D-255: a known-limits list in the changelog, worded at VALIDATE's end; one VALIDATE agenda page meanwhile |
| 11, moving between targets | D-256: previous and next target keymap actions (up and down by default in the chart window), rebindable and in Help, as FR-1.14 is written |
| 12, the read order | D-257: three lines of what the release is open the REQUIRED-READING; the Phase cell a short dated list |
| 13, one sun | D-258: the words take day/night and the next turn from go-tuiMaps' Report; watchpost's Up and NextTurn deleted; RC-14 fixed first |
| 13a, the sun ahead | D-263: go-tuiMaps exports `DaylightAt(place, moment)`, the sun its night is drawn with; the words take it for an hour ahead (v0.3.0-rc.5) |
| 7a, D-252's list | D-264: every listed row re-ratified at today's figure as its floor, the precedent rows' figures too; a drop below a floor a live finding |
| 14, the owed wires rows | D-259: Event.Restored and State.Refused deleted with their rows; F-108 writes them again |
| 15, NOAA's back-off | D-260: NOAA's back-off floored at its 10-minute cadence, then 15; a NOAA 5xx a refusal |
| 16, M5's scope | D-261: M5 stays at 400x110; 800x220 recorded as a known limit and re-measured after RP-4's fixes and a structured benchmark |
| 17, go-tuiMaps' main | D-262: go-tuiMaps v0.2.2 from main on go1.27.3, floor 1.26.10; a go-ionomaps nightly |

## Round 1, closed (R2H-9)

Each round 1 finding, the commit, tag or ruling that closed it, round 2's verdict on it, and where a gap round 2 found was closed. Batch 66 is watchpost's round 2 remediation; go-ionomaps rc.7 and go-tuiMaps rc.6 the libraries'.

| Finding | Closed by | Round 2's verdict | What round 2 left open, and where it closed |
|---|---|---|---|
| RB-1 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RB-2 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RB-3 | D-250 | RULED | VALIDATE agenda item 1 |
| RB-4 | D-246 | RULED, a gap | R2B-1: D-274 |
| RB-5 | watchpost 2449ae00 (batch 59) | FIXED | R2D-2: the arms' spread said, batch 66 |
| RB-6 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RB-7 | D-255 | RULED | R2B-3: the agenda's limits, batch 66 |
| RB-8 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RB-9 | go-ionomaps README, rc.6 | FIXED | - |
| RB-10 | watchpost 2449ae00 (batch 59); D-259 | FIXED | - |
| RC-1 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RC-2 | D-254, built | RULED | - |
| RC-3 | watchpost 2449ae00 (batch 59) | PARTIAL | a failed record fails the run, batch 66 |
| RC-4 | watchpost 2449ae00 (batch 59); 8092d107 (batch 63) | FIXED in code | R2C-1: the rebuild-and-back test, batch 66 |
| RC-5 | 8092d107 (batch 63); A-89 | RULED | R2C-2: `madeWith` tied to the options, batch 66 |
| RC-6 | D-264; 85f7046e (batch 61) | RULED | R2C-4: D-277, batch 66 |
| RC-7 | watchpost 2449ae00 (batch 59) | PARTIAL | `hasKey` and `stillOrphan` deleted, batch 66 |
| RC-8 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RC-9 | watchpost 2449ae00 (batch 59) | PARTIAL | the run-selector check reads build tags, batch 66 |
| RC-10 | 59ed3997 (batch 62) | FIXED | - |
| RC-11 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RC-12 | go-ionomaps v0.1.0-rc.6 | FIXED / RULED | - |
| RC-13 | go-ionomaps v0.1.0-rc.6 | PARTIAL | the mirror test fails where it can check, go-ionomaps rc.7 |
| RC-14 | go-tuiMaps v0.3.0-rc.5 | FIXED | - |
| RC-15 | go-tuiMaps v0.3.0-rc.5 | FIXED for depth | R2C-6: every setter tested, go-tuiMaps rc.6 |
| RC-16 | D-258, D-263; go-tuiMaps v0.3.0-rc.5 | RULED | - |
| RC-17 | go-tuiMaps v0.3.0-rc.5 | FIXED | - |
| RD-1 | watchpost 2449ae00 (batch 59) | PARTIAL | R2D-2, batch 66 |
| RD-2 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RD-3 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RD-4 | watchpost 2449ae00 (batch 59); D-256 | FIXED | R2D-7: the README names the keys, batch 66 |
| RD-5 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RD-6 | go-tuiMaps v0.3.0-rc.5 | FIXED | R2D-4: D-281, go-tuiMaps rc.6 |
| RD-7 | watchpost 2449ae00 (batch 59); go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RD-8 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RD-9 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RD-10 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RD-11 | go-tuiMaps v0.3.0-rc.5 | FIXED | - |
| RD-12 | watchpost 2449ae00 (batch 59) | PARTIAL | R2D-3: D-280, all three repositories |
| RD-13 | D-257 | FIXED | - |
| RD-14 | A-29 (13), D-267 | RULED | reversed by D-281 |
| RH-1 | watchpost 2449ae00 (batch 59); go-ionomaps v0.1.0-rc.6 | PARTIAL | go-tuiMaps' README, rc.6 |
| RH-2 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RH-3 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RH-4 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RH-5 | D-252 | RULED | - |
| RH-6 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RH-7 | D-262; go-tuiMaps v0.2.2 | PARTIAL | R2H-5: the nested modules pinned to v0.2.2 on `main` (PR #8, a23484f); their tags await the HUM LEAD |
| RH-8 | watchpost 2449ae00 (batch 59) | FIXED | R2H-8: `make p10` names a missing `.a2dh.yml`, batch 66 |
| RH-9 | watchpost 2449ae00 (batch 59) | FIXED, with gaps | R2H-3: `finaltag` on the go tool's parser, batch 66 |
| RH-10 | the re-pin, dd74f0a9 (batch 60) | FIXED | - |
| RH-11 | D-259 | RULED | - |
| RH-12 | go-ionomaps v0.1.0-rc.6; go-tuiMaps v0.3.0-rc.5 | FIXED | - |
| RH-13 | go-ionomaps v0.1.0-rc.6 | FIXED (noted) | - |
| RH-14 | go-ionomaps v0.1.0-rc.6 | PARTIAL | R2H-4: watchpost's headers, batch 66 |
| RI-1 | D-248 | RULED, a residual | D-279: a relative bound, go-ionomaps rc.7 |
| RI-2 | watchpost 2449ae00 (batch 59) | PARTIAL | R2I-2: `Local` refused, postcodes drawn plain, batch 66 |
| RI-3 | go-ionomaps v0.1.0-rc.6 | PARTIAL | R2I-1: the line cap fails closed, go-ionomaps rc.7 |
| RI-4 | watchpost 2449ae00 (batch 59) | FIXED | R2I-3: the tile clients' headers capped, batch 66 and go-tuiMaps rc.6 |
| RI-5 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RI-6 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RI-7 | watchpost 2449ae00 (batch 59) | FIXED | - |
| RI-8 | the re-pin, dd74f0a9 (batch 60) | FIXED | - |
| RL-1 | D-247; go-ionomaps v0.1.0-rc.6 | PARTIAL | R2L-1, R2L-2: D-271, D-272, go-ionomaps rc.7 |
| RL-2 | D-249; go-ionomaps v0.1.0-rc.6 | FIXED, a regression | R2L-3: D-270, go-ionomaps rc.7 |
| RL-3 | D-246 | RULED | - |
| RL-4 | D-250 | RULED | - |
| RL-5 | go-ionomaps v0.1.0-rc.6 | PARTIAL | the hand mutants encoded, go-ionomaps rc.7 |
| RL-6 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RL-7 | D-260; go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RL-8 | D-252 | RULED | - |
| RL-9 | watchpost 2449ae00 (batch 59) | PARTIAL | D-255: the verdict at VALIDATE |
| RL-10 | the re-pin, dd74f0a9 (batch 60) | FIXED | - |
| RL-11 | go-ionomaps v0.1.0-rc.6 | FIXED | - |
| RL-12 | go-ionomaps v0.1.0-rc.6; dd74f0a9 (batch 60) | FIXED | - |
| RP-1 | D-251; go-ionomaps v0.1.0-rc.6 | PARTIAL | R2P-1, R2P-2: go-ionomaps rc.7 |
| RP-2 | go-ionomaps v0.1.0-rc.6 | PARTIAL | an older day served alone, go-ionomaps rc.7 |
| RP-3 | D-261, a444b000 (batch 64) | PARTIAL | D-282: watchpost's benchmark, batch 66 |
| RP-4 | go-tuiMaps v0.3.0-rc.5 | FIXED | - |
| RP-5 | none | NOT FIXED | fixed, go-ionomaps rc.7 (A-30 (6)) |
| RP-6 | 59ed3997 (batch 62) | FIXED | - |

## Round 2 (D-269)

Seven fresh blind reviewers, the same lenses, given round 1's findings and the fix commits, each told to verify the fixes in code, hunt what round 1 missed and attack the code the remediation brought in. The reports in full: [review-round-2](reviewer-notes/review-round-2.md). The network dropped twice during the pass; each reviewer was resumed where it stopped.

### The verdicts

| Lens | Verdict | Fix first |
|---|---|---|
| Code Quality (R2C) | exit | R2C-1: the build counter the RC-4 fix rests on has no test of its own; m430's "Killed by" is stale |
| Project Hygiene (R2H) | exit | R2H-1: go-tuiMaps' gate test kills `sleep 3607` across the machine, so overlapping gates fail falsely and kill each other's children |
| Docs Quality (R2D) | **do not exit** | R2D-1: the committed `perfsum` refuses the G1 run the record calls met (47.97 hours against RC-1's 48) |
| Business Quality (R2B) | exit, on R2B-1 | R2B-1: the Broadcaster arm misses G1 at hour 10.6, and nothing said what a miss means |
| The REVIEW lens (R2L) | **do not exit** | R2L-3: since D-260 a 5xx on the solar file alone stops every GloTEC ask |
| InfoSec (R2I) | exit | R2I-1: the solar file's line cap still truncates silently (RI-3's class) |
| Performance (R2P) | exit | R2P-1: every answer opens all 24 typical hours, undoing D-251 |

Round 1's findings, in the reviewers' words: most FIXED in code; PARTIAL - RC-3, RC-7, RC-9, RC-13, RH-1, RH-7, RH-14 (watchpost's headers), RD-1, RD-12, RI-2, RI-3, RL-1, RL-5, RL-9, RP-1, RP-2, RP-3; NOT FIXED - RP-5, never dispositioned. Each PARTIAL's remaining half is in the remediation below.

### The forks, ruled

| Fork | Ruling |
|---|---|
| R2L-3, the solar file's 5xx | D-270: only a 429 stops NOAA's round; a 5xx or no answer marks that file missed and the round goes on; the D-249 test asserts the grid was used |
| R2L-1, three in a row | D-271: GIRO is down only while nothing in the burst has answered; after an answer each failure is the station's own |
| R2L-2, unnamed statuses | D-272: an unnamed non-200 is the station's failure, named once; a 410 is the station's own, only 401 and 403 pause GIRO |
| R2D-1, G1's span | D-273: `perfsum` takes a run as 48 hours within one sampling interval, stated in its output |
| R2B-1, the Broadcaster's miss | D-274: G1 applies to the Broadcaster arm; a miss is a known limit and the map's cost a follow-up with a target |
| R2P-3, a hung GIRO | D-275: GIRO's asks 15 s each, so three failures fit the 60-second update |
| R2H-10, the say hang | D-276: one diagnosed retry inside the test, printed; a second hang fails |
| R2H-7, R2C-4, floorless rows | D-277: a firing density row with no floor fails; its floor set at that day's figure, an A-row |
| R2H-11, changes after a candidate | D-278: only code a host builds takes a candidate; tools, tests and docs are said in the build log |
| RI-1's residual | D-279: a bound relative to the background, measured on the recorded days, beside the 8 MHz |
| R2D-3, refusals | D-280: a lane that refuses before any leg logs nothing, in all three repositories |
| R2D-4, IDs in public docs | D-281: no red-team or ruling IDs in exported doc comments; go-ionomaps A-29 (13) reversed |
| RP-3, watchpost's bench | D-282: watchpost's benchmark pans each op over a structured field |

### The agent's remediation, round 2

- **go-ionomaps, a new candidate (v0.1.0-rc.7):** D-270, D-271, D-272, D-275 (the GIRO ask's 15 s), D-279 (the ratio, with `gm3 -residuals`), D-280, D-281; R2P-1 (typical hours opened as read) and R2P-2 (the pin from before `New`); RP-2 (the least recently used day evicted); RP-5 (`sunAt` and the year fraction worked once); R2C-3 (one freshness function each), R2C-5 (`Inputs.Unread`'s doc, the names exported), R2C-7 (the stale index's test), R2C-9 (`checkPlaces`), R2C-10 and R2L-5 (`Implausible` its own count); R2I-1 (the line cap fails closed), R2I-4 (the plausibility count's test); R2L-4 (a restore drops unknown codes and keeps the mean); RL-5 (the hand mutants of `answers.go`, `reach.go`, `lower*.go`, `internal/glotec`, `internal/limits` and `internal/climatology` in the harness); RC-13 (the mirror test fails, not skips, where it can check).
- **go-tuiMaps, a new candidate (v0.3.0-rc.6):** R2H-1 (the leg's own process, never a machine-wide pattern); R2I-3 (reply headers capped); R2I-5 (`DaylightAt` refuses a place off the globe); R2C-6 (every look setter's re-check tested); D-280, D-281; R2H-2 (its tip's gate row committed); R2H-4 (its rulings' phase header); the gate's private workspace (R2D's note). **v0.2.2's nested modules** (R2H-5): pinned to v0.2.2 on `main` (PR #8, a23484f); their tags await the HUM LEAD.
- **watchpost:** D-273, D-274 (the agenda and the follow-up row), D-276, D-277, D-282; R2C-1 (a rebuild-and-back test, m430's record), R2C-2 (`madeWith` tied to the options' fields), R2C-8 (one moment for the words and the night); R2I-2 (`Local` refused, stored postcodes drawn plain); R2H-3 (`finaltag` on the module graph, the release step pinned); R2H-4, R2H-8, R2H-9 (this section), R2D-2, R2D-5, R2D-6, R2D-7, R2B-2, R2B-3, R2B-4; RC-3 (a failed record fails the run), RC-7 (`hasKey`, `stillOrphan`), RC-9 (the build tags); the re-pin to rc.7 and rc.6 (D-206, D-278).

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R2L-1 | CONFIRMED | go-ionomaps `internal/giro/asker.go:183-214`: `count` and `stops` end the burst on any three failures in a row, answered stations before them or not |
| R2L-2 | CONFIRMED | `asker.go:229-235`: a status `ended` does not name reaches `Answered()` and returns answered |
| R2L-3 | CONFIRMED | `update.go:129-139,256-258`: a 5xx or transport error on the solar file sets `refused`, which skips the grid, D-RAP and scales asks with a usable mean held |
| R2D-1 | CONFIRMED (fact) | D-241's run is 47.97 hours an arm; `tools/perfsum/g1.go:143-145` fails a span under 48 hours; ruled D-273 |
| R2B-1 | CONFIRMED (fact) | D-246 sets no rule for a miss; ruled D-274 |
| R2C-1, R2P-1, R2H-1 | PLAUSIBLE | each reproduced by its reviewer; checked again by its fix's own test, which fails before the fix |

### Round 2, closed

Each round 2 finding and the commit, tag or ruling that closed it, for round 3 to check against the code.

| Finding | Closed by |
|---|---|
| R2L-1 | D-271; go-ionomaps v0.1.0-rc.7 |
| R2L-2 | D-272; go-ionomaps v0.1.0-rc.7 (counted, the station's own); watchpost batch 66 (noted once) |
| R2L-3 | D-270; go-ionomaps v0.1.0-rc.7 |
| R2L-4 | go-ionomaps v0.1.0-rc.7 (A-30 (3)) |
| R2L-5 | go-ionomaps v0.1.0-rc.7 (`Implausible`, `ImplausibleM`); watchpost batch 66 (each noted) |
| R2L-6 | as R2H-1 |
| R2C-1 | watchpost batch 66 (`TestARebuildAndBackIsAnsweredAgain`, m547; m430 re-pointed) |
| R2C-2 | watchpost batch 66 (`TestEveryUpdateOptionIsMadeWith`) |
| R2C-3 | go-ionomaps v0.1.0-rc.7 |
| R2C-4 | D-277; watchpost batch 66 |
| R2C-5 | go-ionomaps v0.1.0-rc.7 (the `Unread` names exported) |
| R2C-6 | go-tuiMaps v0.3.0-rc.6 |
| R2C-7 | go-ionomaps v0.1.0-rc.7 (`TestAStaleIndexFetchesNoGrid`) |
| R2C-8 | watchpost batch 66 |
| R2C-9 | go-ionomaps v0.1.0-rc.7 |
| R2C-10 | as R2L-5 |
| R2D-1 | D-273; watchpost batch 66 |
| R2D-2 | watchpost batch 66 |
| R2D-3 | D-280; all three repositories |
| R2D-4 | D-281; go-ionomaps v0.1.0-rc.7, go-tuiMaps v0.3.0-rc.6 |
| R2D-5 | watchpost batch 66 |
| R2D-6 | watchpost batch 66 |
| R2D-7 | watchpost batch 66 |
| R2H-1 | go-tuiMaps v0.3.0-rc.6 |
| R2H-2 | go-tuiMaps v0.3.0-rc.6 (the rc.5 tip's row committed) |
| R2H-3 | watchpost batch 66 |
| R2H-4 | watchpost batch 66; go-tuiMaps v0.3.0-rc.6 |
| R2H-5 | go-tuiMaps PR #8 (a23484f on `main`): the nested modules require v0.2.2; their six tags await the HUM LEAD, the tag push declined by the permission system |
| R2H-6 | go-ionomaps e2103d7 (the build log says the nightly fires from the merge; each candidate's full gate dispatched) |
| R2H-7 | D-277; watchpost batch 66 |
| R2H-8 | watchpost batch 66; go-ionomaps v0.1.0-rc.7; go-tuiMaps 474797a |
| R2H-9 | watchpost batch 66 ("Round 1, closed") |
| R2H-10 | D-276; watchpost batch 66 |
| R2H-11 | D-278 |
| R2I-1 | go-ionomaps v0.1.0-rc.7 |
| R2I-2 | watchpost batch 66 |
| R2I-3 | go-tuiMaps v0.3.0-rc.6; watchpost batch 66 |
| R2I-4 | go-ionomaps v0.1.0-rc.7 (`TestAnImplausibleReadingIsCounted`) |
| R2I-5 | go-tuiMaps v0.3.0-rc.6 |
| R2P-1 | go-ionomaps v0.1.0-rc.7 |
| R2P-2 | go-ionomaps v0.1.0-rc.7 |
| R2P-3 | D-275; go-ionomaps v0.1.0-rc.7 |
| R2B-1 | D-274; F-220; the agenda |
| R2B-2 | watchpost batch 66 (the agenda) |
| R2B-3 | watchpost batch 66 (F-220, the agenda's limits) |
| R2B-4 | watchpost batch 66 |

## Round 3 (D-284)

Three fresh blind reviewers, the two lenses that held REVIEW open in round 2 and a Code reviewer over batch 66 and the candidates; their reports verbatim in [review-round-3](reviewer-notes/review-round-3.md). Tips: watchpost `03ad4e97`, go-ionomaps `e2103d7` (v0.1.0-rc.7 `57bd7ef`), go-tuiMaps `474797a` (v0.3.0-rc.6).

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R3D) | do not exit; no Critical, every fix a document | R3D-1: go-ionomaps' requirements R-5.3, R-5.8, R-3.3 and R-9.8 contradict round 2's code (Important); R3D-4: the gate roster false about the G1 scorer and missing batch 66's gates (Important) |
| REVIEW lens (R3L) | do not exit; no Critical | R3L-1: a stalling NOAA host runs every update to its deadline, GIRO unasked and NOAA not backed off - a regression from D-270 (Important) |
| Code Quality (R3C) | do not exit; no Critical | R3C-1: a GIRO that answers once and then hangs is never called down, D-275's goal undone by D-271 (Important; R3L-2, Minor, in the REVIEW lens) |

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R3D-1 | CONFIRMED | go-ionomaps `requirements.md:75` (R-5.3) still says three failures in a row stop a burst and a 410 pauses GIRO; the file was last touched in `00ac871`, round 1 |
| R3D-4 | CONFIRMED | `07-readiness/gates.md:29` says a default run under 48 hours exits missed; `tools/perfsum/g1.go`'s `g1FullHours` takes 48 hours less one sample |
| R3L-1 | CONFIRMED | go-ionomaps `update.go:140-142` returns Unfinished before `nextAsk` (`:144`) and the GIRO switch (`:155`); two of watchpost's 30 s client timeouts (`platform/httpx/plain.go:80`) fill the 60 s update (`domains/propagation/propagation.go:47`) |
| R3C-1 | CONFIRMED | go-ionomaps `internal/giro/asker.go:219`: `stops` ends a burst on failures in a row only while nothing has answered; 39 asks after an answer, 15 s each |

### The forks, ruled

| Finding | Ruling |
|---|---|
| R3L-1 | D-285: each NOAA ask has its own 20 s; a timeout stops NOAA's round as a 429 does and backs off; a 5xx still goes on |
| R3C-1, R3L-2 | D-286: three timeouts in a row call GIRO down whatever answered; the Unfinished words name GIRO when the burst was the update |
| R3L-3 | D-287: on a stop, the failed stations go behind the ones not reached and the next rotation's head |
| R3D-10, R3C-4 | D-288: no IDs of any kind in either library's exported docs, a `go doc -all` scan test; A-30 (1) and A-19 (1) vetoed |
| R3C's P10 recommendation | D-289: a follow-up row against the harness's density counter; D-264 and D-283 stand meanwhile |

### The agent's remediation, round 3

- **go-ionomaps, a new candidate (v0.1.0-rc.8):** D-285, D-286, D-287, D-288; R3D-1 (the four requirement rows, their tests named), R3D-2 and R3L-6/R3C-2 (`Restore` names the part it refused, a sentinel for each; the docs say which part is taken), R3D-3 (the constants table's fold row; `Implausible`'s doc), R3C-6 (`nextAsk`'s parameter; `ask`'s doc), R3C-7 and R3L-4 (a two-update back-off test for an unanswered NOAA ask, with its mutant; a guard deadline in the hung-GIRO fake), R3C-11 (one shape for the freshness helpers).
- **go-tuiMaps, a new candidate (v0.3.0-rc.7):** D-288; R3D-6 (the README's run log), R3D-7 (the release notes' rc.6 changes), R3C-10 (`made_work` deleted), the two ragged comments, the aliases' field lists held by a test (round 2's unrecorded Minor).
- **watchpost:** R3D-2 and R3L-6/R3C-2 (the restore note names the part refused), R3D-4 (the gate roster), R3D-5 (this record), R3D-8 (`docs/extending.md`'s keymaps; the README's action names), R3D-9 (the arms' spans and the margin), R3D-11 (F-220's and the fold's limits), R3L-5/R3C-3 (the statuses D-247 names, 404 and 5xx among them; NOAA's words), R3L-7 (a refused count noted on change), R3L-8/R3C-5 (`ionomaps.UnreadSolar`), R3C-8 (runlog's two untested paths), R3C-9 (the say leg's budget; a hang only on an error), R3C-12 (the selector's `-race` branch), D-286's words, D-289's follow-up row; round 2's unrecorded Minors (reachscore's twice-written check, `propMadeOf`'s name); the re-pin to rc.8 and rc.7.

### Round 3, closed

Each round 3 finding and the commit, tag or ruling that closed it, for the next round to check against the code.

| Finding | Closed by |
|---|---|
| R3D-1 | go-ionomaps v0.1.0-rc.8 (the four requirement rows) |
| R3D-2 | go-ionomaps v0.1.0-rc.8 (docs, README, api-review); watchpost batch 67 (architecture, FR-4.12, the note) |
| R3D-3 | go-ionomaps v0.1.0-rc.8 (the constants table's fold; `Implausible`'s doc) |
| R3D-4 | watchpost batch 67 (the gate roster) |
| R3D-5 | watchpost batch 67 (this record) |
| R3D-6 | go-tuiMaps v0.3.0-rc.7 (the README) |
| R3D-7 | go-tuiMaps v0.3.0-rc.7 (the release notes) |
| R3D-8 | watchpost batch 67 (`docs/extending.md`; the README's action names) |
| R3D-9 | watchpost batch 67 (the arms' spans, the margin; `perfsum`'s comment) |
| R3D-10 | D-288; go-ionomaps v0.1.0-rc.8, go-tuiMaps v0.3.0-rc.7 |
| R3D-11 | watchpost batch 67 (F-220); go-ionomaps v0.1.0-rc.8 (the fold's quiet days) |
| R3L-1 | D-285; go-ionomaps v0.1.0-rc.8 |
| R3L-2 | D-286; go-ionomaps v0.1.0-rc.8; watchpost batch 67 (the words) |
| R3L-3 | D-287; go-ionomaps v0.1.0-rc.8 |
| R3L-4 | go-ionomaps v0.1.0-rc.8 (the second back-off tested) |
| R3L-5 | watchpost batch 67 |
| R3L-6 | go-ionomaps v0.1.0-rc.8 (`ErrBadGIROState`, `ErrBadMean`); watchpost batch 67 (the note) |
| R3L-7 | watchpost batch 67 |
| R3L-8 | watchpost batch 67 |
| R3C-1 | as R3L-2 |
| R3C-2 | as R3L-6 |
| R3C-3 | as R3L-5 |
| R3C-4 | D-288; go-ionomaps v0.1.0-rc.8 |
| R3C-5 | as R3L-8 |
| R3C-6 | go-ionomaps v0.1.0-rc.8 |
| R3C-7 | go-ionomaps v0.1.0-rc.8 (the back-off test; the guarded hang) |
| R3C-8 | watchpost batch 67 |
| R3C-9 | watchpost batch 67 |
| R3C-10 | go-tuiMaps v0.3.0-rc.7 |
| R3C-11 | go-ionomaps v0.1.0-rc.8 |
| R3C-12 | watchpost batch 67 |
| Round 2's unrecorded Minors | the alias field lists: go-tuiMaps v0.3.0-rc.7; `reachscore`'s check and `propMadeOf`'s name: watchpost batch 67 |
| R3C's P10 recommendation | D-289; F-221 |

## Round 4 (D-290)

Three fresh blind reviewers - Docs, the REVIEW lens, Code; their reports verbatim in [review-round-4](reviewer-notes/review-round-4.md). Tips: watchpost `d63f499e`, go-ionomaps `08f71ef` (v0.1.0-rc.8), go-tuiMaps `6dbfdf9` (v0.3.0-rc.7).

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R4D) | exit once R4D-1 and R4D-3 are fixed; no Critical or Important | R4D-1: D-288's scan misses a dotless plan ID, and `(G5)` survives |
| REVIEW lens (R4L) | do not exit; no Critical | R4L-1: D-286 holds only for a hang in the first ~15 s of a burst (Important) |
| Code Quality (R4C) | do not exit; no Critical | R4C-1: the same, found apart, D-287 putting hanging stations last (Important) |

### The agent's check of the Important finding

| Finding | Verdict | What was checked |
|---|---|---|
| R4L-1, R4C-1 | CONFIRMED | go-ionomaps `internal/giro/asker.go:182-189`: the host's context is read only before each ask, and `stops` calls GIRO down on three timeouts in a row; with about 0.9 s an answer, three 15 s timeouts after 16 answers end past 60 s |

### The forks, ruled

| Finding | Ruling |
|---|---|
| R4L-1, R4C-1 | D-291: the burst ends before the host's deadline, GIRO called down if its last ask timed out or three did |
| R4D-2 | D-292: a window's keys stay fixed; the docs say so |
| R4L-7 | D-293: the Broadcaster arm's build named, its figures an upper bound for the tip |

### The agent's remediation, round 4

- **go-ionomaps, a new candidate (v0.1.0-rc.9):** D-291; R4L-3 (any timeout error taken as one, the production timeouts pinned against a host's budget), R4C-7 (`Unasked` takes the one station whose failure stopped the burst, and only a run of failures moves the head), R4C-8 and R4L-6 (tests for the head past the last of four, timed-out stations behind, the timeout run reset, timeouts counted as failures, both parts of a state refused), R4C-5 (`refused` named for what it means, the round's stop), R4C-4 and R4D-5 (the docs D-285 and D-286 left stale; `ErrBadState`'s message), R4D-1, R4C-2 and R4L-5 (`(G5)`, `(P-6)` and the process words out of the exported docs; the scan widened to a general ID pattern with an allowlist of published work), R4D-6 (the per-ask limits in the README and `Update`'s doc).
- **go-tuiMaps, a new candidate (v0.3.0-rc.8):** R4C-2 and R4L-5 (the parity and task IDs out of its exported docs; the scan widened), R4C-9 (every alias of an internal struct checked, found by its AST).
- **watchpost:** D-292 (the docs on window keys), D-293 (the agenda names the build), R4D-3 and R4L-4 (a 410 named), R4D-4 (`perfsum` prints each arm's hours to three places and its margin, the step said nominal), R4D-5 and R4C-6 (the say leg's arithmetic), R4D-7 (the state file holds times and the flux mean), R4C-3 and R4L-2 (the burst flag in the session's state), the build log's mutant list and the roster's m544 (R4D); the re-pin to rc.9 and rc.8.

### Round 4, closed

Each round 4 finding and the commit, tag or ruling that closed it, for the next round to check against the code.

| Finding | Closed by |
|---|---|
| R4D-1 | go-ionomaps v0.1.0-rc.9 (`(G5)` gone; the scan's pattern general, with an allowlist) |
| R4D-2 | D-292; watchpost batch 68 (`docs/extending.md`, the README's table) |
| R4D-3 | watchpost batch 68 (a 410 named; A-92 (1)) |
| R4D-4 | watchpost batch 68 (`perfsum -g1`: hours to three places, the shortfall, the nominal step) |
| R4D-5 | go-ionomaps v0.1.0-rc.9 (`update.go`'s round, `Burst`'s doc); watchpost batch 68 (the Makefile and A-92 (6): three renders and a retry) |
| R4D-6 | go-ionomaps v0.1.0-rc.9 (the README and `Update`'s doc carry the per-ask limits) |
| R4D-7 | watchpost batch 68 (the README, the changelog and FR-4.12: times and the flux mean) |
| R4D, smaller | watchpost batch 68 (batch 67's mutant list; the roster's row 29 cites m544) |
| R4L-1 | D-291; go-ionomaps v0.1.0-rc.9 (`roomForAnAsk`, `endsAtTheDeadline`; NOAA's round alike) |
| R4L-2 | watchpost batch 68 (`propSession.burst`) |
| R4L-3 | go-ionomaps v0.1.0-rc.9 (`isTimeout`; NOAA's 20 s pinned) |
| R4L-4 | watchpost batch 68 (as R4D-3) |
| R4L-5 | go-ionomaps v0.1.0-rc.9 and go-tuiMaps v0.3.0-rc.8 (as R4D-1 and R4C-2) |
| R4L-6 | go-ionomaps v0.1.0-rc.9 (both parts refused, tested) |
| R4L-7 | D-293; watchpost batch 68 (the agenda's item 3) |
| R4C-1 | as R4L-1 |
| R4C-2 | go-ionomaps v0.1.0-rc.9 and go-tuiMaps v0.3.0-rc.8 (the IDs stripped; the scan general in both) |
| R4C-3 | as R4L-2 |
| R4C-4 | go-ionomaps v0.1.0-rc.9 (`GIROUnavailable`, `Unavailable`, `Burst`, `askTimeout`, `ErrBadState`'s message) |
| R4C-5 | go-ionomaps v0.1.0-rc.9 (`endsRound`) |
| R4C-6 | watchpost batch 68 (as R4D-5) |
| R4C-7 | go-ionomaps v0.1.0-rc.9 (`Unasked` takes one station; only a run of failures moves the head) |
| R4C-8 | go-ionomaps v0.1.0-rc.9 (i2 to i5 and i8 each tested, with mutants) |
| R4C-9 | go-tuiMaps v0.3.0-rc.8 (every struct alias found by type-checking) |

## Round 5 (D-294)

Three fresh blind reviewers - Docs, the REVIEW lens, Code; their reports verbatim in [review-round-5](reviewer-notes/review-round-5.md). Tips: watchpost `af661bf5`, go-ionomaps `3b0ab33` (v0.1.0-rc.9), go-tuiMaps `17ae3de` (v0.3.0-rc.8).

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R5D) | exit; eight Minors, no Critical or Important | R5D-1: a NOAA round the deadline leaves no room for backs NOAA off, decided by an A-row and documented nowhere |
| REVIEW lens (R5L) | do not exit; no Critical | R5L-1: one hanging station, asked late in the burst, calls GIRO down every hour (Important) |
| Code Quality (R5C) | do not exit; no Critical | R5C-1: D-291's room rule leaves a burst 45 s of the 60, cuts a burst with one hang short every hour, and says it Fresh (Important) |

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R5L-1 | CONFIRMED | go-ionomaps `internal/giro/asker.go:233-239`: at the deadline GIRO is called down whenever the last ask ran out; a hang begun past about 30 s of a burst leaves under 15 s, so it is always the last ask; `movedPast` then puts that station at the rotation's end (`stations.go:188`), so it is the last ask every hour. D-291's own words ("if its last ask timed out") produce it |
| R5C-1 | CONFIRMED | `asker.go:217-224`: `roomForAnAsk` refuses an ask without a full `askTimeout` left, 45 s of asking in a 60 s update; one 15 s hang and 39 answers at about 0.92 s need about 51 s. A burst so cut with GIRO answering backs nothing off (D-291), and `update.go:114-122` says the snapshot Unfinished only when the context has ended, so it is said Fresh; watchpost's D-286 words for a stopped burst are reached only by a cancel, whose result is dropped |

### The forks, ruled

| Finding | Ruling |
|---|---|
| R5C-1 | D-295: an ask with under 15 s left gets the time left less a 1 s margin, down to about 2 s; one that runs out of a shortened limit is not reached |
| R5L-1 | D-296: at the end GIRO is called down only on its last two full-length asks run out with no answer between, or D-286's three in a row; D-291's "three of its asks" withdrawn |
| R5L-4, R5C-1 (2) | D-297: the stations a burst did not reach counted on `Inputs` and said by watchpost, in place of D-286's stop words |
| R5D-1, R5L-2 | D-298: a round with no room left backs nothing off; no room for any ask is an error naming the deadline and its 20 s minimum, no GIRO hour spent; watchpost refuses a Deadline under 20 s; A-33 (1) withdrawn |
| R5C-7 | D-299: go-tuiMaps' memory peak judged in the race-off leg alone |

### The agent's remediation, round 5

- **go-ionomaps, a new candidate (v0.1.0-rc.10):** D-295 (the last asks get the time left; a shortened ask that runs out is not reached), D-296 (the call-down at the end: two full timeouts in a row), D-297 (`Inputs`' count of stations not reached), D-298 (no back-off for a round with no room; the no-room error and its minimum; no GIRO hour spent when nothing was asked); R5C-3 and R5L-3 (a test for each surviving mutant: the head at the end, the no-room back-off gone, the call-down count); R5C-4 (one room helper and one `isTimeout`; the burst's per-ask counts in one tally); R5C-5 and R5D-2 (the exported docs D-291 and D-298 change; the README's per-ask limits right for GIRO, in real time; "moves past past"; `endsAtTheDeadline` named for what it does); R5D-3 (the seed week's dates, one wording); R5D-7 (the ID scan's must-catch list widened, "general" made true or dropped); the nits (`reach.go`'s wrapped path, `update.go:133`, `noteStatus`'s doc).
- **go-tuiMaps, a new candidate (v0.3.0-rc.9):** D-299 (the memory peak judged in the race-off leg); R5D-7 (the alias test fails below the count found, seventeen; the scan's must-catch list as go-ionomaps'); "upstream" defined where go doc says it.
- **watchpost:** D-297 (the cut-short words, in place of D-286's stop words, `propSession.burst` and their tests and mutants); D-298 (a Deadline under 20 s refused); R5C-6 and R5L-5 (`TestAnUpdateHasItsOwnDeadline` reworded to what it reads: the service's own deadline, not a source reached); R5D-4 (the chart window's fixed keys, the acknowledgement window in the README); R5D-5 (the required reading and the record's status through round 5); R5D-6 (the architecture's failure table: a station's own failure, GIRO hanging, a denial, a stalled NOAA, a burst cut short); R5D-8 (item 3's two blind spots: D-293 a reading of the diff, the run's network outage); R5D-7's residue (FR-4.12: the station codes and rotation positions the state also holds); the re-pin to rc.10 and rc.9.

### Round 5, closed

Each round 5 finding and the commit, tag or ruling that closed it, for the next round to check against the code.

| Finding | Closed by |
|---|---|
| R5D-1 | D-298; go-ionomaps v0.1.0-rc.10 (no back-off for a round with no room; `MinDeadline`, `ErrNoRoom`; R-5.3, `Update`'s doc, the README and its example) |
| R5D-2 | go-ionomaps v0.1.0-rc.10 (the README's per-ask limits: GIRO's timeouts its stations' own, real time) |
| R5D-3 | go-ionomaps v0.1.0-rc.10 (the seed week, "the week of 2026-09-29 to 10-05" everywhere) |
| R5D-4 | watchpost batch 69 (the chart window's fixed keys; the acknowledgement window in the README) |
| R5D-5 | watchpost batch 69 (the required reading and the record through round 5) |
| R5D-6 | watchpost batch 69 (the architecture's failure table) |
| R5D-7 | go-ionomaps v0.1.0-rc.10 and go-tuiMaps v0.3.0-rc.9 (the scan takes every family the documents use; the alias test fails below seventeen) |
| R5D-8 | watchpost batch 69 (item 3's two blind spots) |
| R5D, nits | go-ionomaps v0.1.0-rc.10 (`Unasked`'s doc, the round's comment, `reach.go`'s path); go-tuiMaps v0.3.0-rc.9 ("upstream" defined); watchpost batch 69 (`noteStatus`'s doc; the state file's contents) |
| R5L-1 | D-296; go-ionomaps v0.1.0-rc.10 (`timeoutsAtTheEnd`; one station hanging late, eight hours, never GIRO called down) |
| R5L-2 | D-298; go-ionomaps v0.1.0-rc.10 (no hour spent, no back-off, the error naming the deadline); watchpost batch 69 (a Deadline under the least refused) |
| R5L-3 | go-ionomaps v0.1.0-rc.10 (Me is m090; Mg's rule withdrawn by D-296; Ma's head move removed, A-34 (3)) |
| R5L-4 | D-297; go-ionomaps v0.1.0-rc.10 (`Inputs.StationsNotReached`); watchpost batch 69 (the words, the dead D-286 words retired) |
| R5L-5 | watchpost batch 69 (as R5C-6) |
| R5C-1 | D-295, D-297; go-ionomaps v0.1.0-rc.10 (`askLimit`; a burst with one early hang asks every station) |
| R5C-3 | as R5L-3; x4's `lastFailed` removed with the head move |
| R5C-4 | go-ionomaps v0.1.0-rc.10 (`giro.TimeLeft` and `giro.IsTimeout`, once; the burst's `tally`) |
| R5C-5 | go-ionomaps v0.1.0-rc.10 (`GIROUnavailable`, `Unavailable`, `Update`, `Unfinished`, `ask`'s doc; `endsAtTheDeadline` replaced by `endEarly`; the throttle restored when nothing was asked) |
| R5C-6 | watchpost batch 69 (`TestAnUpdateHasItsOwnDeadline` at the least deadline) |
| R5C-7 | D-299; go-tuiMaps v0.3.0-rc.9 (the peak judged with the race detector off) |

## Round 6 (D-300)

Four fresh blind reviewers - Docs, the REVIEW lens, Code, and a timing lens; their reports verbatim in [review-round-6](reviewer-notes/review-round-6.md). Tips: watchpost `2ec275b5`, go-ionomaps `d5d4d6c` (v0.1.0-rc.10), go-tuiMaps `813b7b0` (v0.3.0-rc.9).

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R6D) | do not exit until R6D-1 is fixed | R6D-1: `MinDeadline` refuses an update given exactly it (Important) |
| REVIEW lens (R6L) | exit; eight Minors | R6L-1: a cancelled burst later said as cut short by the deadline |
| Code Quality (R6C) | exit; eight Minors | R6C-2: one room rule with a margin for NOAA and GIRO, which settles R6C-1 (R6D-1) too |
| Timing (R6T) | do not exit | R6T-1: a quit during GIRO's burst does not stop it; the hour is not kept, and a restart bursts again (Important) |

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R6T-1 | CONFIRMED, Important | watchpost `modes/tty/dashboard.go:1424-1425`: quit returns `tea.Quit` without `stopPropagation`; the session's context is `context.Background()`'s (`map_prop_update.go:56-57`), which the app's `cancel()` does not reach; `settleBound` is 2 s (`app/mapprop.go:58`) while an ask may take 15 s, so the state is not kept and D-213 with D-39 breaks on a quit within the burst |
| R6D-1 (R6C-1, R6L-2, R6T-2) | CONFIRMED; Important by Docs, Minor by the other three | go-ionomaps `update.go:351`: an ask is refused with `left < noaaAskTimeout`, and `noaaAskTimeout` is `MinDeadline`, so a context of exactly `MinDeadline` always refuses; watchpost accepts that value (`propagation.go:86`) and batch 69's test pins the refusal - the agent's own error, the test built on the defect. No listener is affected (watchpost runs at 60 s). The same arithmetic gives R6T-5, a NOAA ask with no margin, and R6T-2, a hung GIRO not called down under 32 s |

### The forks, ruled

| Finding | Ruling |
|---|---|
| R6D-1, R6C-1, R6C-2, R6L-2, R6T-2, R6T-5 | D-301: one room rule, a 1 s margin for NOAA's asks and GIRO's; `MinDeadline` 32 s, which asks |
| R6L-1, R6C-4, R6T-4, R6D-2 | D-302: the count holds only live stations the update's end left; a cancel or refusal counts none but puts its stations first, the interrupted one included |
| R6T-3, R6L-5 | D-303: a probe not reached keeps its turn and goes first next hour, uncounted |
| R6C-3 | D-304: Unread keeps D-239's lacks rule; D-298's docs reworded |
| R6T-7 | D-305: a server error breaks the run, as built; D-296's words "one after another" |
| R6C (question 3) | D-306: `internal/deadline`, its P10 row ratified |
| R6C (recommendation) | D-307: a station whose full ask ran out goes last next hour |

R6T-1 needs no fork: its fix restores what D-213 and D-39 rule.

### The agent's remediation, round 6

- **go-ionomaps, a new candidate (v0.1.0-rc.11):** D-301 (`internal/deadline`'s room rule with the 1 s margin for NOAA and GIRO; `MinDeadline` 32 s; a test that an update at exactly `MinDeadline` asks NOAA; the error stops rounding to the minimum; R6T-5's window closed), D-302 (the count set only at the update's end; a cancel through `endEarly` with the interrupted station first; a burst that asked nothing leaves the count; `NoRoom` reset at the top of `Burst`, R6L-6), D-303 (the probe kept and first after a cut), D-304 (the docs' "files not asked" reworded; the test holds a young file), D-305 (D-296's words; the alternating test's doc), D-306 (`internal/deadline` with its ratified row), D-307 (a station whose full ask ran out last next hour); R6L-3 and R6D-5 (the scaled fixture honours its context, the subtests re-run under race and load); R6C-8 (the end-of-update hang asserted full); R6T-6, R6L-4 and R6C-4 (tests for g2, g8, g13, g14 and yd); R6C-6 (`roomless`'s redundant branch); R6T-5's `Fetcher` doc (a Fetch returns once its context ends); the `tally.add` nit.
- **go-tuiMaps, a new candidate (v0.3.0-rc.10):** R6C-7 and R6L-8 (`raceDetector` checked against the binary's build settings); R6D nits (the race-off leg's label and NFR-3's row say it judges the peak).
- **watchpost:** R6T-1 (a quit cancels the Propagation session before the app's wait, and a test that a quit with a burst in flight ends the update and keeps the state inside `settleBound`; `Settle`'s doc and test made true); D-301 (the least deadline at the library's 32 s, the test asking at it; `errNoDeadline`'s words; A-94 (2) corrected); R6T-6's w3 and R6L-4's Mw (the diagnostics' GIRO-answered guard tested); R6D-3, R6C-5 and R6L-7 (`snapshotWords`' stale doc); R6D-4 (the failure table's 429 and NOAA rows quote the words); R6D nits (the state file's contents alike in FR-4.12, the README and the changelog; batch 69's "R5D-4's residue" is R4D-7's; "U9 to U12" explained); R6L's R5L-5 record line corrected; the re-pin to rc.11 and rc.10.

### Round 6, closed

Each round 6 finding and the commit, tag or ruling that closed it, for the next round to check against the code.

| Finding | Closed by |
|---|---|
| R6D-1, R6C-1, R6L-2, R6T-2 | D-301; go-ionomaps v0.1.0-rc.11 (`MinDeadline` 32 s, which asks, tested at exactly it); watchpost batch 70 (the deadline test) |
| R6C-2, R6T-5 | D-301; go-ionomaps v0.1.0-rc.11 (`deadline.Room`, one rule with a 1 s margin for NOAA and GIRO; the `Fetcher` doc) |
| R6D-2, R6L-1, R6C-4, R6T-4 | D-302; go-ionomaps v0.1.0-rc.11 (the count at the update's end only; a cancel through `endEarly`, the interrupted station first; a burst that asked nothing keeps the counts) |
| R6D-3, R6C-5, R6L-7 | watchpost batch 70 (`snapshotWords`' doc) |
| R6D-4 | watchpost batch 70 (the failure table's 429 and NOAA rows) |
| R6D-5, R6L-3 | go-ionomaps v0.1.0-rc.11 (the fixture honours its context; forty runs under race and load, green) |
| R6D, nits | watchpost batch 70 (the state file's contents; the residue's name; "U9 to U12"); go-tuiMaps v0.3.0-rc.10 (the race-off leg's label, NFR-3's row) |
| R6L-4, R6T-6 | go-ionomaps v0.1.0-rc.11 (g2 m103, g8 and Mc by D-302's test, g13 the just-under-room subtest, g14 m104); watchpost batch 70 (w3 and Mw, m576) |
| R6L-5, R6T-3 | D-303; go-ionomaps v0.1.0-rc.11 (the probe kept and first, uncounted) |
| R6L-6 | go-ionomaps v0.1.0-rc.11 (`NoRoom` reset at the top of `Burst`; m102) |
| R6L-8, R6C-7 | go-tuiMaps v0.3.0-rc.10 (`TestRaceDetectorIsToldTruly`; m094) |
| R6L (R5L-5's record line) | watchpost batch 70 (the record corrected) |
| R6C-3 | D-304; go-ionomaps v0.1.0-rc.11 (the docs reworded; the test holds a young file) |
| R6C-6 | go-ionomaps v0.1.0-rc.11 (`roomless`'s redundant branch removed) |
| R6C-8 | go-ionomaps v0.1.0-rc.11 (the end-of-update hang asserted full) |
| R6C, question 3 | D-306; go-ionomaps v0.1.0-rc.11 (`internal/deadline`, its row ratified) |
| R6C, recommendation | D-307; go-ionomaps v0.1.0-rc.11 (a station that ran out asked last; m100, m101) |
| R6T-1 | watchpost batch 70 (the update tied to the app's context; m575) |
| R6T-7 | D-305; go-ionomaps v0.1.0-rc.11 (D-106's words) |

### After round 6: the hosted gate's find

go-ionomaps rc.11's full hosted gate (run 40428744932, amd64) failed `FuzzAnswersNeverReturnNaN`: a path from a place to itself panicked in the sun track. arm64 turns the NaN to step 0 and amd64 to the least int64, so no local gate saw it. Fixed in go-ionomaps v0.1.0-rc.12 (A-36), re-pinned in watchpost batch 71, before round 7.

## Round 7 (D-308)

Four fresh blind reviewers, each also running on amd64 - Docs, the REVIEW lens, Code, and the timing lens; their reports verbatim in [review-round-7](reviewer-notes/review-round-7.md). Tips: watchpost `b16e5411`, go-ionomaps `68f31f3` (v0.1.0-rc.12), go-tuiMaps `5f6bf07` (v0.3.0-rc.10).

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R7D) | do not exit | R7D-2, with R7D-1: D-307's tail, which a deadline cut moves to the head, and which calls a healthy GIRO down beside a hanging probe (Important) |
| REVIEW lens (R7L) | do not exit | R7L-1: the same, three hanging stations calling GIRO down every other hour (Important) |
| Code Quality (R7C) | do not exit | R7C-1: the same, the one-line order (Important) |
| Timing (R7T) | do not exit | R7T-1: the same, a false outage every other hour for days at PLAN's pace (Important); the library's bounds hold everywhere |

### The agent's check of the Important finding

| Finding | Verdict | What was checked |
|---|---|---|
| R7D-1, R7L-1, R7C-1, R7T-1 | CONFIRMED, by all four apart; the agent's own implementation of D-307 | go-ionomaps `internal/stations/stations.go:243-254`: `rotation` puts the stations a burst did not reach (`leftOut`) first, before it puts the stations that ran out (`last`) last; D-307 gathers the hanging stations at the tail, the deadline cuts them, `Unasked` puts them first, and three side by side are D-286's three timeouts in a row - a healthy GIRO called down. `TestAStationThatRanOutIsAskedLast` sets no deadline, so the collision never ran |
| R7D-2 (R7T-2) | CONFIRMED | `asker.go:199-202`: the probe is the last ask, after D-307's tail; two hanging stations and a hanging probe are three timeouts in a row, and two at the end trip D-296's call-down |

### The forks, ruled

| Finding | Ruling |
|---|---|
| R7D-1, R7L-1, R7C-1, R7T-1 | D-309: D-307 wins; a station that ran out stays last when the deadline left it, uncounted |
| R7D-2, R7T-2 | D-310: an ask of a station already known to run out counts toward no GIRO-down run |
| R7C (question 3) | D-311: the clamp a helper again, `internal/limits`' floor renewed; and the direction - one harmonised model, judged against, never a round reversing a round |
| D-311's direction | D-312: the model approved - go-ionomaps `03-architecture-design/update-timing-model.md` |

### The agent's remediation, round 7, built to the model (D-312)

- **go-ionomaps, a new candidate (v0.1.0-rc.13):** the model's section 3 (a station known to hang stays last, never put first by a deadline's cut, uncounted - D-309) and section 2 (its timeout counts toward no run and breaks one - D-310), with the test over eight hours and more under a deadline with two, three and four hanging stations and a hanging probe (R7D-5); the clamp a helper, `internal/limits`' floor renewed (D-311); the model's words where the docs differ - "until an ask of it ends any other way" (R7C-5), `NotReached`, `Inputs.StationsNotReached` and `Update`'s run conditions (R7D-3); the state's kept probe and known-to-hang list held by a restart test (R7C-2; R7L-2's MD, ME); `sunAt` refusing a place that is not finite (R7C-3); `TestARoundWithNoRoomNamesWhatItLacks` saying its run and asserting its exact Unread (R7C-4, R7L-3); a scaled test of GIRO called down at twice a full ask and its margin (R7C-7); `RanOut` only on a full ask run out, tested (R7L-2's MI); the no-room error's truncation tested (R7L-2's MG); a kept probe now live asked once, tested (R7L-2's MC); the NaN fuzz target asking a path from the fuzzed place too (R7L's recommendation); the rotation's skipped case commented, `endOf` renamed; the README's known limits (model section 8).
- **watchpost:** the app's-end test with a burst in flight and a field held, `settleBound` pinned (R7T-3, R7C-6); the architecture's update cancelled at the app's end too (R7D-4) and its link to the model; the model's section 8 in VALIDATE's item 7 and the changelog's known limits (R7D-6); the re-pin to rc.13.

### Round 7, closed

Each round 7 finding and the commit, tag or ruling that closed it, for the next round to check against the code and the timing model.

| Finding | Closed by |
|---|---|
| R7D-1, R7L-1, R7C-1, R7T-1 | D-309, the model's section 3; go-ionomaps v0.1.0-rc.13 (a station known to hang stays last when a deadline left it, uncounted; `TestHangingStationsNeverCallAHealthyGIRODown`; m107, m109) |
| R7D-2, R7T-2 | D-310, the model's section 2; go-ionomaps v0.1.0-rc.13 (`hungAgain`, its own failure, counting toward no run; m108) |
| R7D-3 | go-ionomaps v0.1.0-rc.13 (`NotReached`, `Inputs.StationsNotReached` and `Update`'s run conditions in the model's words) |
| R7D-4 | watchpost batch 72 (the architecture: the app's end cancels the update; the model linked) |
| R7D-5 | go-ionomaps v0.1.0-rc.13 (D-307's outcome held under a deadline, ten hours, two to four hanging stations and a hanging probe) |
| R7D-6 | watchpost batch 72 (VALIDATE's item 7 and the changelog's known limits); go-ionomaps v0.1.0-rc.13 (the README's known limits) |
| R7L-2 | go-ionomaps v0.1.0-rc.13 (MC m117, MD m114, ME m115, MG m116, MI m111, each with its test) |
| R7L-3, R7C-4 | go-ionomaps v0.1.0-rc.13 (the test says its run and asserts D-RAP alone) |
| R7L, recommendation | go-ionomaps v0.1.0-rc.13 (the fuzz target asks a path from the fuzzed place too) |
| R7C-2 | go-ionomaps v0.1.0-rc.13 (`TestTheKeptProbeAndTheStationsThatHangSurviveARestart`; m112 to m115) |
| R7C-3 | go-ionomaps v0.1.0-rc.13 (`sunAt` refuses a place that is not finite; m110) |
| R7C-5 | go-ionomaps v0.1.0-rc.13 ("until an ask of it ends any other way", everywhere) |
| R7C-6, R7T-3 | watchpost batch 72 (a burst in flight with its field held; `settleBound` pinned, m577) |
| R7C-7 | go-ionomaps v0.1.0-rc.13 (a hung GIRO called down at the least deadline, scaled) |
| R7C, question 3 | D-311; go-ionomaps v0.1.0-rc.13 (`onPath` a helper; `internal/limits`' floor renewed) |
| R7C, nit | go-ionomaps v0.1.0-rc.13 (`endOf` renamed `restore`) |

## Round 8 (D-313)

Four fresh blind reviewers, each also running on amd64, each judging the update's timing against the timing model (D-312) - Docs, the REVIEW lens, Code, and the timing lens; their reports verbatim in [review-round-8](reviewer-notes/review-round-8.md). Tips: watchpost `49c65df4`, go-ionomaps `59cb4a9` (v0.1.0-rc.13), go-tuiMaps `0c27321` (v0.3.0-rc.10). Every round 7 finding was found fixed in the code, R7D-5 and R7D-6 partly.

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R8D) | do not exit | R8D-1: with 40 or more live stations, a station known to hang is never asked again (Important) |
| REVIEW lens (R8L) | do not exit | R8L-1: a GIRO hung for about 15 hours is never called down again (Important) |
| Code Quality (R8C) | do not exit | R8C-1: the same (Important) |
| Timing (R8T) | do not exit | R8T-1: the same (Important); the library's bounds hold everywhere |

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R8C-1, R8T-1, R8L-1 | CONFIRMED, by three apart and by the agent's own probe (arm64): GIRO hung on every ask, called down in hours 0 to 14, then `Answered`, not paused, from hour 15, the known-to-hang list holding every live station | go-ionomaps `internal/giro/asker.go`: `RanOut` marks each station whose full ask ran out, the three that call GIRO down included; once every live station is known to hang, each timeout is `hungAgain` and counts toward no run (D-310). Kind (b): the model's sections 2, 3 and 8 combine to it |
| R8D-1, R8L-3 (R8T-2) | CONFIRMED, by the reviewer's probe run on the tree: 41 live, the hanging station polled in 0 of 72 hours | `internal/stations/stations.go`: `rotation` puts stations known to hang after every other live station and `Plan` keeps the first 39, so with 39 or more others live it is always cut, and D-309 keeps it out of the head next hour; R8T-2 is the same with a slow GIRO's deadline. Kind (b) |
| R8L-5 | CONFIRMED by the code | `Plan` puts a kept probe first with no known-to-hang exception |

### The fork, ruled

| Finding | Ruling |
|---|---|
| R8C-1, R8T-1, R8L-1; R8D-1, R8L-3, R8T-2; R8L-5; R8D-3, R8D-4, R8T-5, R8D-7, R8T-3 | D-314: the model changed whole - M1 (before a burst's first answer a station known to hang counts), M2 (one station known to hang retried each burst with a 2 s ask, in turn), M3 (a kept probe known to hang asked last), M4 (the model's words made true); the Minors batched |

### The agent's remediation, round 8, to the model as D-314 changed it

- **go-ionomaps, a new candidate (v0.1.0-rc.14):** the model's revision; M1 (`TestADeadGIROIsCalledDownEveryHour`, 24 hours); M2 (`Plan`'s retry inside the budget, its room 35 s; `TestEveryStationKnownToHangIsRetriedInTurn` with GIRO at 1.6 s an ask, `TestTheRetryTakesTheStationsKnownToHangInTurn` with 45 live); M3 (`TestAKeptProbeKnownToHangIsAskedLast`); the ten-hour test's hanging stations known from the start, its reads checked against every live station not hanging (R8C-3, R8D-2); `TestAStationKnownToHangBreaksARun` (R8T-4 X1 to X3, R8L-2 N1, N2, R8C-2 zK); the tail's order, a dark station never polled, a run of timeouts leaving the head (X5, X6, X9, R8L-2 N3, R8C-2 zF); the docs to the model (R8D-3's field, the README); the stations ledger row's veto wording (R8D-6); mutants m118 to m130.
- **watchpost (batch 73):** the changelog's known limits and VALIDATE's item 7 as section 8 (R8C-4, R8L-4, R8T-3); the model cited by repository and path (R8D-6); the as-built map's nodes (R8D-5); the Broadcaster figures' blind spots (R8D-8); the re-pin to rc.14.

### Round 8, closed

| Finding | Closed by |
|---|---|
| R8C-1, R8T-1, R8L-1 | D-314 M1; go-ionomaps v0.1.0-rc.14 (m118) |
| R8D-1, R8L-3, R8T-2 | D-314 M2; go-ionomaps v0.1.0-rc.14 (m119 to m122, m128) |
| R8L-5 | D-314 M3; go-ionomaps v0.1.0-rc.14 (m123) |
| R8D-3, R8D-4, R8T-5, R8D-7, R8T-3 | D-314 M4 (the model's sections 2, 4 and 8); go-ionomaps v0.1.0-rc.14 (`Inputs.StationsNotReached`) |
| R8C-2, R8T-4, R8L-2 | go-ionomaps v0.1.0-rc.14 (m124 to m127, m129, m130; m108 re-pointed) |
| R8C-3, R8D-2 | go-ionomaps v0.1.0-rc.14 (the ten-hour test) |
| R8C-4, R8L-4 | watchpost batch 73 (the changelog's known limits) |
| R8D-5 | watchpost batch 73 (the as-built map's nodes) |
| R8D-6 | go-ionomaps v0.1.0-rc.14 (the README, the ledger row); watchpost batch 73 (item 7, the citations) |
| R8D-8 | watchpost batch 73 (the figures' blind spots and unit; A-96's platform) |
| R8C, nits | go-ionomaps v0.1.0-rc.14 (`State.Last`'s doc); `MinDeadline` written beside the giro constants stays, its reason pinned by R7C-7's test |

### After round 8: the hosted gate's find

watchpost's hosted gate on `1275b161` (batch 72's code) failed on the Ubuntu runner's race leg: the app's-end test's cut burst took past 2 s to keep its state under the race detector, which this machine's race leg (405 ms) never showed. The product bound is unchanged; the test holds the race build to the bound times the detector's slowdown, and the bound itself without race (watchpost batch 74, A-98), before round 9.

## Round 9 (D-315)

Four fresh blind reviewers, each also running on amd64, each judging against the timing model as D-314 changed it - Docs, the REVIEW lens, Code, and the timing lens; their reports verbatim in [review-round-9](reviewer-notes/review-round-9.md). Tips: watchpost `32324c39`, go-ionomaps `f60cc5c` (v0.1.0-rc.14), go-tuiMaps `91c3630` (v0.3.0-rc.10). Every round 8 finding was found fixed as ruled.

### The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Docs Quality (R9D) | do not exit | R9D-1: M2's retry starves a recovered station that answers slower than 2 s (Important) |
| REVIEW lens (R9L) | do not exit | R9L-1: a station list left for three days goes dark as a whole - one ask an hour, about 110 hours to read the 39 again, and an outage past three days never said (Important) |
| Code Quality (R9C) | do not exit | R9C-1: the same as R9D-1 (Important) |
| Timing (R9T) | do not exit | R9T-2: M3 with D-303's keep freezes the probe rotation (Important); R9T-1, the same as R9D-1 (Important); the bounds hold everywhere |

### The agent's check of the Important findings

| Finding | Verdict | What was checked |
|---|---|---|
| R9D-1, R9C-1, R9T-1 | CONFIRMED, by three apart and by the reviewer's probe run on the tree: a station answering in 3 s read in 0 of 24 hours with one or two known to hang | go-ionomaps `internal/stations/stations.go`: `Plan` takes the retry out of the order, so a lone station known to hang is never in the tail and gets only the 2 s ask. Kind (b): D-314's M2, a regression from rc.13 at PLAN's pace |
| R9T-2 | CONFIRMED by the code | `Unasked` keeps the turn of a probe the burst did not reach (D-303), and M3 asks a kept probe known to hang last, where a slow hour cuts it again. Kind (b): D-314's M3 |
| R9L-1 | CONFIRMED, by the reviewer's probe run on the tree: after four days away, one ask an hour, none live | `isLive` judges darkness by the wall clock, and a restored state keeps `Seeded` and `Heard`; with none live, `Plan` asks only the probe. Kind (b): predates round 8 (D-51 with D-213) |
| R9D-2, R9C-2, R9L-2, R9T-3 | CONFIRMED by the code, all four apart | `roomToRetry` is read when the burst is planned, before the kept probe's full ask. Kind (a): a departure from section 1, fixed without a fork |

### The fork

One whole change to the model, put to the HUM LEAD under D-311.

