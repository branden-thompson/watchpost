---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — REVIEW exit red team, round 1"
date: 2026-10-24
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 REPORTED 2026-10-24: seven blind reviewers (D-245); every finding checked by the agent against the clean tips; the forks are put to the HUM LEAD one at a time, then the remediation batches"
---

# REVIEW exit — red team, round 1

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
- **go-tuiMaps, v0.3.0-rc.4:** `Daylight.Turns` (RC-14, once reproduced); the host-ramp warning on a later depth (RC-15); the release notes' sun figure (RD-11); RC-17; the frame writer's buffer (RP-4); BUILD exit ticked (RH-12); `go doc` and the aliased fields (RD-6); RC-8's comments.
- **watchpost:** the hour ahead keyed on `built` (RC-4); "or worse" (RB-1); `perfsum` failing on an unparsable number and the driver's failure carried (RC-1); `runlog` taking the tree before and after (RC-3) and logging no refused run (RD-12); both G1 comparisons in the record, the summary's "dry run" line reconciled with D-241 (RB-5, RD-1); the changelog worded as what changed until the proof run's figures are committed (RB-6, RD-2, RL-9); `reachscore`'s blind spots in its output and no verdict where no figure stands (RB-8, RC-9); the docs (RD-7, RD-9, RD-10, RH-1, RH-14); the geocoder's postcode, zone and coordinates (RI-2); a response-header cap and long validators dropped (RI-4); the state file named, not its path (RI-7); 0.19.0's gate roster and 0.18.0's status line (RH-6); the hook's message (RH-8); a test refusing a final tag on any `-rc` module (RH-9); the session state in one place (RC-5, D-237); RC-7, RC-8, RC-10; the refresh note's comment (RB-10); the centre's name lookup (RP-6).
- **The ship-together state (the agent's error, RH-10):** go-ionomaps rc.6 and go-tuiMaps rc.4 tagged at their gated tips, watchpost re-pinned and gated (RL-10, RI-8).

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
| RD-6 | Not yet checked | |
| RD-7 | CONFIRMED | `architecture.md:133` names `Refused`; `:43` names `Library` methods `Fields, Bands, Reach` it does not have. |
| RD-8 | CONFIRMED | go-ionomaps `implementation-plan.md:134` against `ahead.go:61-63`. |
| RD-9 | CONFIRMED | `problem-statement.md:64` "50 MB RSS"; `:98` "M1 and M4's baseline is not yet measured". |
| RD-10 | CONFIRMED | `README.md:516` cites go-tuiMaps D-28 (1.26.9); 1.26.10 is its D-38. |
| RD-11 | CONFIRMED | go-tuiMaps `release-notes.md:27` 0.011° over 2020 to 2035; `internal/sun/sun_test.go` holds 0.5° over a 2026 year. |
| RD-12 | CONFIRMED | `Makefile:302-303`: runlog wraps the lane that refuses. |
| RD-13, RD-14 | Forks 12; A-row | |
| RI-1 | CONFIRMED (mechanics); fork 3 | `internal/stations/stations.go:233-237` the reply's position overwrites the seed's; `internal/assimilate/assimilate.go:76` the fit centres on the mean residual. |
| RI-2, RI-3, RI-4, RI-5, RI-7 | Not yet checked | Each reviewer's repro is in its report. |
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
| RP-4, RP-5, RP-6 | Not yet checked | |
| RC-1 | CONFIRMED | `tools/perfsum/main.go:265` an unparsable number is 0; `scripts/quality/workload.sh:137` a driver's failure exits 0. |
| RC-2 | Fork 9 | |
| RC-3 | CONFIRMED | `tools/runlog/main.go:43-44` the tree is read after the run. |
| RC-4 | CONFIRMED | `map_prop_ahead.go:66` on `Computed` alone; the chart's key carries `built` (`map_prop_chart.go:22`). |
| RC-5 | CONFIRMED (structure); remediated under D-237 | |
| RC-6 | Fork 7 | |
| RC-7, RC-8, RC-10 | Minor; checked as fixed | |
| RC-9, RC-13, RC-15 | Not yet checked | |
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

## The rulings

| Fork | Ruling |
|---|---|
| 1, G1's comparison | D-246: D-236 stands; both readings recorded and to VALIDATE; a 48-hour Broadcaster arm measured and reported; the map's own cost a follow-up |
| 2, GIRO's failures | D-247: a station's failure its own; the burst stops on a 429 or three in a row; 401/403/410 a 24-hour pause, said; the failure carried all hour; the rotation moved |
| 3, GIRO's plausibility | D-248: seeded stations keep their positions (over 50 km refused); a residual bound from the recorded days; G-M3 re-run; D-105's offset stands |
| 4, no solar file | D-249: the last F10.7 mean kept in the saved state, used with its age up to 7 days; past that no field, said as the file unread |
| 5, PS-1's accuracy | D-250: no accuracy floor, recorded; the PR claims answerability for PS-1 with M1's agreement beside it; M1's scope and the chart's agreement at VALIDATE |
| 6, the hours ahead | D-251: P-6 stands, each snapshot owns its slices; D-131 amended; a heap pin over 24 hours ahead under 15 MB |
