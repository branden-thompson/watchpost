---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — BUILD report"
date: 2026-10-18
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "PRESENTED 2026-10-18; awaiting the HUM LEAD's rulings below"
---

# BUILD report — 0.19.0, go-ionomaps v0.1.0, go-tuiMaps v0.3.0

Up: [rulings](../02-analysis/rulings.md) · [build log](../04-development/build-log.md) · [red team, BUILD exit](red-team-build.md) · [the M1/M4 instrument](../07-readiness/m1-m4-instrument.md)

## Bottom line

- **The three projects are built and pinned together.**
  - watchpost `feature/propagation-overlays` runs on go-ionomaps `v0.1.0-rc.2` and go-tuiMaps `v0.3.0-rc.2`, both tagged after their full gates, hosted ones included.
  - watchpost took 47 batches, each committed on a green full gate. The record is current with the code.
- **The BUILD-exit red team's round 1 is remediated.**
  - Eight blind reviewers said do not exit, with two Criticals (D-203).
  - The thirteen forks were ruled D-204 to D-217, and the remediation ran in the report's order: CI, then go-ionomaps, then go-tuiMaps, then watchpost (batches 35 to 44).
  - Both Criticals are closed. The Observer names the place it is looking at, anywhere (AX-1, batch 38). A found place is a target row, and the map does not move (B-F1, D-204, batch 41).
- **M1 and M4 have their baseline, and both miss D-99's target on PLAN's week.**
  - The instrument is `tools/reachscore` (batches 45 to 47). Its dry run uses PLAN's recorded inputs and 7 days of WSPR spots.
  - Under the draw now ruled (by band, D-224), the shipped build agrees 53.4 % on M1 and 66.9 % on M4. The climatology agrees 53.4 % and 64.2 %. The target is the climatology's plus 5 points.
  - On M4, GloTEC carries the gain; GIRO's stations add about nothing on these paths.
  - No floor rides on M1 or M4, so a miss comes to the HUM LEAD at VALIDATE (D-118). The graded run uses fresh days beside REVIEW (D-221).
- **What the dry run taught the instrument:** the draw first ruled (band × hour, D-219) gave M1 50 % for any model whose answer depends on band and hour. It was put as fork E, and D-224 draws by band.
- **The long runs are beside REVIEW, as D-208 ruled:** G1's 48 hours, G-M3's leave-one-out on fresh days (G10.4), and M1/M4's graded run. Their verdicts come before VALIDATE.

## What was built

| Project | Built | State |
|---|---|---|
| watchpost 0.19.0 | The Propagation mode on Observer and Broadcaster: the MUF and foF2 fields, the chart from the origin to the six continents and to found places, the centre's answer, a frequency's reach, the hours ahead labelled typical, the words for every answer (O1). The Broadcaster's own map with its service area (D-195). Day and night as map detail (O3, D-199). The answers as their own package (`platform/propanswers`, D-212). Settings for the noise, the refresh, the no-readings correction, no stations (D-214) and the colour depth (D-207). | 47 batches, each committed on a green `make verify`; hosted CI read after every push since batch 34 |
| go-ionomaps v0.1.0-rc.2 | The hybrid current hour: GIRO's residuals over GloTEC, and the climatology's M(3000)F2. NRL's CCIR refit on a 30-day F10.7. Magnetic coordinates from IGRF-14. D-RAP's disturbance, NOAA's scales. The reach with its lower limit (D-205), no stations (D-214), the throttle kept across restarts (D-213). The day's climatology hours kept (D-222). | tagged on `2436c8a` after its full gate; hosted quick and full gates green |
| go-tuiMaps v0.3.0-rc.2 | Fields over water, the MUF and foF2 presets, the terminator, the reach's edge without colour, the service zone. The M5 frame brought from 21.7 ms to 6.6 ms at 400 × 110 (D-209). The colour depth exposed (A-15). | tagged on `dd67d22` after its full gate; hosted full gate green |

## The red team and its remediation

Round 1 is `red-team-build.md`: its verdicts, findings, forks and rulings, and the order the remediation
followed. The remediation:

| Batches | What |
|---|---|
| 35 | CI's actions re-pinned off Node.js 20, the runner image pinned (F-216, H-13) |
| 36 | The record: the requirements as built, every named test real, citations that resolve, runs recorded |
| 37, 40, 44 | watchpost on go-ionomaps rc.1 and rc.2, and on go-tuiMaps rc.2: the hours ahead their own, no stations, the throttle kept, the reach's edge, the colour depth, the climatology hours kept |
| 38 | The words: the centre named anywhere, read in the ruled order, the reader's place kept (AX-1 to AX-13, D-210, D-211) |
| 39, 42, 43 | The code findings: each update its own, answers for the snapshot shown, one horizon, functions under 60 lines, the hook, the answers' measured figures |
| 41 | The answers package and the found place as a target (D-212, D-204) |
| 45 to 47 | The M1/M4 instrument, its dry run, and the draw by band (D-208, D-218 to D-221, D-224) |

**No second round has run.** PLAN ran one, with four reviewers over the remediated record (D-128). This
report asks whether BUILD should too (ruling 1 below).

## The metrics at BUILD exit

| Metric | Where it stands | Ruling |
|---|---|---|
| M1 (agreement with WSPR, the tower's served area) | **baseline measured, target missed on PLAN's week:** 53.4 % for both climatology and shipped (target 58.4 %); graded on fresh days beside REVIEW | D-99, D-118, D-221, D-224 |
| M4 (the listener's paths) | **baseline measured, target missed on PLAN's week:** 64.2 % for the climatology, 66.9 % shipped (target 69.2 %); graded on fresh days beside REVIEW | D-99, D-118, D-221, D-224 |
| M2, M5, M5b (UAT) | the protocol is ready; the live sitting is VALIDATE's (D-89) | D-93 |
| G1 (station cost) | the harness built; its 48-hour run beside REVIEW | D-208 |
| go-ionomaps G-M2 (reproducible) | built in BUILD: `TestAFieldIsReproducedFromItsInputs`, a golden both CI runners check (G10.1) | D-97, D-208 |
| go-ionomaps G-M4 (faults said) | built in BUILD: `TestEveryFaultIsReported` (G10.2) | D-208 |
| go-ionomaps G-M3 (error against climatology) | on at least 3 fresh days beside REVIEW (G10.4) | D-108, D-132 |
| go-ionomaps G-G1 (cost per update) | gated by allocation proxies; the wall clock is a release step | D-98, D-119 |
| go-tuiMaps M5 (the frame) | **met:** 6.6 ms at 400 × 110 (16 ms target); 1.4 ms at 200 × 56 (8 ms) | D-96, D-209 |

**What a floor miss would remove (D-208).** The floors are D-75's.
1. If the hybrid does not beat the climatology on foF2 and MUF(3000) in the US, go-ionomaps ships the
   GloTEC-and-climatology background alone, without the stations. That would remove GIRO, its throttle,
   its burst and its Settings.
2. If the hours ahead are worse than the climatology at +3 h and +12 h, the hours ahead are cut from
   0.19.0.

Both are judged on the fresh days (G10.4). The M4 dry run is a hint, not the floor's measure: on those
paths the stations added about nothing over the background.

## The agent's errors in BUILD

| # | Error | Corrected |
|---|---|---|
| E-20 | The plan named a `tools/p10` that was li-A2DH's CLI all along | D-142 |
| — | Hosted CI went unread through batch 34: batch 28 failed on macOS, at a wall-clock fixture the agent wrote, and batch 32 on both systems; the red team found it | read after every push since; the fixture pinned to its own clock |
| — | D-146's evidence named specimen frames that drew no MUF or foF2 at all | D-147 |
| — | The M1/M4 draw (B1) was recommended without seeing that it fixes M1 at 50 % for a band-and-hour model; the dry run showed it | fork E, D-224 |

## What REVIEW inherits

- **Beside REVIEW, verdicts before VALIDATE:** G1's 48-hour run (W10.2); G10.4's fresh-day fetch and
  G-M3; M1/M4's graded run on the same fresh days, by band (D-224), the climatology reported on both sets
  of days (D-221).
- **The instrument's settings** (A-74 (5)), unless ruled otherwise: the 50 km ground-wave cut, M1's
  161 km served area, M4's 1000 km regions (ruling 2 below).
- **The evidence:** the PLAN evidence folder outside the tree now also holds the dry run's spots, the
  scenarios and the inputs' checksums (`m1m4/`).

## Blind spots of this report

- **The M1/M4 baseline is one week, the week the targets were set on.** The fresh days are the check.
- **WSPR measures WSPR's geography:** its stations are not where listeners and towers are. A locator is
  a square, not a station. "Monitored" means monitored by US receivers. PLAN's week kept no D-RAP, so the
  dry run never said "disturbed".
- **The agent wrote the instrument that grades its own build.** The fresh days, the committed tool and its
  mutants are the checks, not independence.
- **No second red-team round has looked at the remediation.**

## The rulings asked

Each is put singly.

1. **A second red-team round before BUILD exits?** Recommended: four blind reviewers over the remediated
   tips (the BUILD lens, Code, Docs, and the combined personas), as PLAN's round 2 did.
2. **The M1/M4 instrument's settings** (A-74 (5)): the 50 km ground-wave cut, M1's 161 km served area and
   M4's 1000 km regions, or others, for the graded run.
3. **Approve BUILD exit**, once 1 is done or declined, so REVIEW opens with the long runs beside it.
4. **The agent's decisions batched for veto** (D-13); each stands unless vetoed: watchpost A-73, A-74 (1)
   to (4), A-75; go-ionomaps A-23.
