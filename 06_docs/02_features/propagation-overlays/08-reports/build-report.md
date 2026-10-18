---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — BUILD report"
date: 2026-10-18
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-18 (D-233). BUILD exited on e889e72a (batch 50): its full gate green, hosted CI green on Ubuntu and macOS, P10 clean. REVIEW is open."
---

# BUILD report — 0.19.0, go-ionomaps v0.1.0, go-tuiMaps v0.3.0

Up: [rulings](../02-analysis/rulings.md) · [build log](../04-development/build-log.md) · [red team, BUILD exit](red-team-build.md) · [the M1/M4 instrument](../07-readiness/m1-m4-instrument.md)

## Bottom line

- **The three projects are built and pinned together.** watchpost `feature/propagation-overlays` runs on
  go-ionomaps `v0.1.0-rc.3` and go-tuiMaps `v0.3.0-rc.3`. Both libraries were tagged after their full
  gates, and their hosted full gates are green on their tips. watchpost took 49 batches, each committed
  on a green full gate, and it now carries `main`'s 0.18.1.
- **Two red-team rounds, both remediated.**
  - Round 1 (D-203): eight blind reviewers, two Criticals, thirteen forks (D-204 to D-217), remediated
    in batches 35 to 44.
  - Round 2 (D-225): four fresh reviewers over the remediated tips. All four said do not exit, with no
    Critical. Their four forks were ruled D-226 to D-229, and the findings remediated in both libraries'
    rc.3 and watchpost's batch 48.
  - The defect every round 2 reviewer found was one the remediation itself had brought in: the reach
    said "above the MUF" when the absorption stopped it. It is fixed.
- **M1 and M4 have their baseline, and both miss D-99's target on PLAN's week.**
  - The draw is by band (D-224), and each figure carries its 95 % paired interval (D-228).
  - M1: the shipped build agrees 53.4 % [51.0, 55.9], and so does the climatology; the lead is +0.1
    [-0.7, +0.9].
  - M4: the shipped build agrees 66.9 % [64.6, 69.1], the climatology 64.2 % [61.8, 66.6]; the lead is
    +2.6 [+1.2, +4.0]. That lead is real but short of +5 even at its interval's top. GloTEC carries it;
    the stations add about nothing on these paths.
  - **How to read them:** a constant answer scores 50 % under a balanced draw. Every column agrees with
    only 29 to 52 % of the paths WSPR heard, and with 72 to 92 % of those it did not.
  - No floor rides on M1 or M4, so the miss comes to the HUM LEAD at VALIDATE (D-118). The graded run is
    on fresh days beside REVIEW (D-221).
- **PS-1 is addressed in part (D-217):** an operator can ask from the station whether the band reaches
  the served area, and is not told when it closes. F-209 carries the rest.
- **The long runs are beside REVIEW, as D-208 ruled:** G1's 48 hours, G-M3's leave-one-out on fresh
  days (G10.4), and M1/M4's graded run.

## What was built

| Project | Built | State |
|---|---|---|
| watchpost 0.19.0 | The Propagation mode on Observer and Broadcaster: the MUF and foF2 fields, the chart from the origin to the six continents and to found places, the centre's answer, a frequency's reach and the limit that stops it, the hours ahead labelled typical, words for every answer (O1). The Broadcaster's own map with its service area (D-195). Day and night as map detail (O3, D-199). The answers as their own package (`platform/propanswers`, D-212). Settings for the noise, the refresh, the no-readings correction, no stations (D-214) and the colour depth (D-207). The README and CHANGELOG say all of it. | 49 batches, each committed on a green `make verify`; on `main`'s 0.18.1 |
| go-ionomaps v0.1.0-rc.3 | The hybrid current hour: GIRO's residuals over GloTEC, and the climatology's M(3000)F2. NRL's CCIR refit on a 30-day F10.7. Magnetic coordinates from IGRF-14. D-RAP and NOAA's scales. The reach with its lower limit (D-205), read from a polar grid (D-227: 0.16 s, 0.26 s the day's first, 0.65 % of reached cells differing from their own circuits). No stations (D-214). The throttle kept across restarts, a state no list reaches refused (D-213). One climatology cache for the answers and the hours ahead (D-222). | tagged on `31c2e19`; hosted quick and full gates green on `8f6c085` |
| go-tuiMaps v0.3.0-rc.3 | Fields over water, the MUF and foF2 presets, the terminator, the reach's edge without colour, the service zone. The M5 frame: 10.2 to 11.4 ms at 400 × 110 with its contours drawn, from D-209's 33.6 ms on watchpost's UI goroutine before its memo. The colour depth exposed (A-15). `BadEdge` (D-229). | tagged on `7b7c7de`; release check green; hosted full gate green on `d93fac5` |

## The red team and its remediation

Both rounds are in `red-team-build.md`, with every finding by its ID; the reviewers' notes are in
`reviewer-notes/build-round-1.md` and `build-round-2.md`.

| Batches | What |
|---|---|
| 35 | CI's actions re-pinned off Node.js 20, the runner image pinned (F-216, H-13) |
| 36 | The record: the requirements as built, every named test real, citations that resolve, runs recorded |
| 37, 40, 44 | watchpost on go-ionomaps rc.1 and rc.2, and on go-tuiMaps rc.2 |
| 38, 39, 41 to 43 | Round 1's words, code findings, the answers package and the found target |
| 45 to 47 | The M1/M4 instrument, its dry run, and the draw by band (D-224) |
| 48 | Round 2's code findings; watchpost on both libraries' rc.3; D-215 carried into watchpost; citations by symbol |
| 49 | `main`'s 0.18.1 merged; the record's round 2 findings: this report, the architecture, the README and CHANGELOG, the instrument page |

## The metrics at BUILD exit

| Metric | Where it stands | Ruling |
|---|---|---|
| M1 (agreement with WSPR, the tower's served area) | **baseline measured, target missed on PLAN's week:** 53.4 % for both climatology and shipped (target 58.4 %); graded on fresh days beside REVIEW | D-99, D-118, D-221, D-224, D-228 |
| M4 (the listener's paths) | **baseline measured, target missed on PLAN's week:** 64.2 % for the climatology, 66.9 % shipped, lead +2.6 [+1.2, +4.0] (target 69.2 %); graded on fresh days beside REVIEW | the same |
| M2, M5, M5b (UAT) | the protocol is ready; the live sitting is VALIDATE's (D-89) | D-93 |
| G1 (station cost) | **the harness was not built at BUILD exit**, though this report said it was; built in REVIEW (batch 51, A-79), its 48-hour run beside REVIEW, holding a frequency while panning (D-227) | D-208, D-227 |
| go-ionomaps G-M2 (reproducible) | built: `TestAFieldIsReproducedFromItsInputs` holds the climatology's field to a golden both runners check (G10.1). The assimilated field is not in the golden; synthetic stations would flatten it | D-97, D-208 |
| go-ionomaps G-M4 (faults said) | built: `TestEveryFaultIsReported` (G10.2), its GIRO cases passing only when the fault is reported | D-208 |
| go-ionomaps G-M3 (error against climatology) | on at least 3 fresh days beside REVIEW (G10.4) | D-108, D-132 |
| go-ionomaps G-G1 (cost per update) | gated by allocation proxies; the wall clock is a release step | D-98, D-119 |
| go-tuiMaps M5 (the frame) | **met:** 10.2 to 11.4 ms at 400 × 110 with its contours drawn (16 ms target), 2.3 to 2.5 ms at 200 × 56 (8 ms); release row 2.8 holds v0.3.0 to the reference machine's figure | D-96, D-209 |

**What a floor miss would remove (D-208).** D-75 set two floors.
1. **The hybrid against the climatology** on foF2 and MUF(3000) in the US, judged on the fresh days
   (G10.4). If it misses, go-ionomaps ships the GloTEC-and-climatology background alone, without GIRO's
   stations, its throttle, its burst and its Settings. The M4 dry run is a hint, not the measure: on those
   paths the stations added about nothing.
2. **The hours ahead against the climatology.** Since D-109 the hours ahead *are* the climatology,
   labelled typical, so this floor is met by construction and removes nothing.

## Hosted CI

watchpost's hosted CI runs on every push, and a push cancels the run before it, so not every batch has
a run of its own. Batches 38 to 43 were each cancelled by the next push, and d6fba065's run, carrying
them, passed on both systems. Since then, every run that completed has passed: `1b3223a7` (through batch
46), `559f22db` (batch 47) and `a554c982`. Batch 48's own run, on `1e6c60b2`, was still running when
this was written; batch 49's follows it. go-ionomaps' hosted full
gate failed once in round 2, on a fuzz target's clock and not a crash, and is green on its fix
(`8f6c085`).

## The agent's errors in BUILD

| # | Error | Corrected |
|---|---|---|
| E-20 | The plan named a `tools/p10` that was li-A2DH's CLI all along | D-142 |
| — | Hosted CI went unread through batch 34: batch 28 failed on macOS, at a wall-clock fixture the agent wrote, and batch 32 on both systems; the red team found it | read on every tip since; the fixture pinned to its own clock |
| — | D-146's evidence named specimen frames that drew no MUF or foF2 at all | D-147 |
| — | The M1/M4 draw (B1) was recommended without seeing that it fixes M1 at 50 % for a band-and-hour model | fork E, D-224 |
| — | Round 1's remediation carried D-205 into go-ionomaps but not into watchpost's words: the reach said "above the MUF" when the absorption stopped it | batch 48 |
| — | Batch 45 renamed a method and added guards to satisfy P10's analysers, which D-215 forbids | batch 48: the guards removed, the shortfall put to the HUM LEAD |
| — | This report's first version misstated floor 2, left out D-217's sentence, and said hosted CI was read after every push when several runs had been cancelled | this revision |
| — | A mutant whose anchor still matched no longer compiled after batch 48 simplified its function; the full gate's first run caught it | m364 rewritten; the touched files' mutants compile-checked before the gate |
| — | This report, as approved, said G1's harness was built. It was not: W10.2 had no driver, no wire count and no summary until REVIEW, and the approval rested on that line | the harness built in batch 51; this line corrected and the HUM LEAD told |

## What REVIEW inherits

- **Beside REVIEW, verdicts before VALIDATE:** G1's 48-hour run (W10.2), holding a frequency while
  panning; G10.4's fresh-day fetch and G-M3; M1/M4's graded run on the same fresh days, by band (D-224),
  with intervals (D-228), the climatology reported on both sets of days (D-221).
- **The instrument's settings** (A-74 (5)), unless ruled otherwise: the 50 km ground-wave cut, M1's
  161 km served area, M4's 1000 km regions.
- **The evidence:** the PLAN evidence folder outside the tree holds the dry run's spots, the scenarios and
  the inputs' checksums (`m1m4/`).

## Blind spots of this report

- **The M1/M4 baseline is one week, the week the targets were set on.** The fresh days are the check.
- **WSPR measures WSPR's geography,** and WSPR alone grades M1 and M4 (D-226). A locator is a square, not
  a station. "Monitored" means monitored by US receivers. PLAN's week kept no D-RAP, so the dry run never
  said "disturbed".
- **The agent wrote the instrument that grades its own build.** The fresh days, the committed tool, its
  mutants and its fail-closed checks are the checks, not independence.
- **PS-1 rests on a hypothesis the HUM LEAD holds (D-217),** not on evidence about operators.

## The rulings asked

Each is put singly.

1. **P10 after D-215.** Writing watchpost's code plainly leaves:
   - three bare-name misreadings (P10-01): `domains/propagation`'s `New` calling go-ionomaps' `New`,
     the resolver's `Resolve` calling the geocoder's, and the history's `Put` calling its memo's;
   - two density shortfalls (P10-05): `platform/propanswers` at 0.94 and `tools/reachscore` at 1.34,
     now that their count-raising guards are gone.

   Ratify five rows until upstream fixes the analysers, or rename the three as go-ionomaps D-72 did.
2. **F-218:** a table of country names, so the words say "Japan", not "J P". It is new data with its own
   credit.
3. **The M1/M4 instrument's settings** (A-74 (5)) for the graded run.
4. **Approve BUILD exit,** so REVIEW opens with the long runs beside it.
5. **The agent's decisions batched for veto** (D-13); each stands unless vetoed: watchpost A-73 to A-76;
   go-ionomaps A-23, A-24; go-tuiMaps A-16.
