---
title: "0.18.0 — Observer maps — BUILD REPORT"
date: 2026-10-05
phase: BUILD — exit
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
branch: feature/map-drawing
issue: "branden-thompson/watchpost#22"
status: "APPROVED by the HUM LEAD 2026-10-05 (D-264): \"GO 4 REVIEW when ready\""
---

# 0.18.0 — Observer maps — BUILD REPORT

## Bottom line up front

**BUILD is complete and recommends proceeding to REVIEW.** 137 batches built the map window the PLAN
approved (D-56), on go-tuiMaps v0.2.0, released and pinned (D-243). The HUM LEAD closed UAT on 2026-10-03
(D-239). The BUILD-exit red team's four blind reviewers said "do not ship yet"; every one of their
findings is now fixed, ruled (D-250 to D-263) or carried to 0.19.0 with its reason
(`red-team-build.md`).

What a reader should know before the detail:

1. **What ships.** The map window (`g`): alert areas with severity words and digits, the basemap, radar
   loops with HRRR's hours ahead, temperature and feels-like, wind and gusts, rain and snow, waves, buoys
   and tides, fire, quakes, UV and air quality; Radar and Forecast modes; pan, zoom and the playback
   keys; a text description; Settings' Maps tab; MAP STATUS in the Status window; the hourly history
   recorder.
2. **What was cut to 0.19.0, by ruling.** FR-1.8, FR-5.8, FR-5.9, FR-9.5, FR-9.6, FR-7.1, FR-7.2, FR-7.5
   (D-244 to D-247) and reduce motion (D-255). RK-12's residual, the listeners those serve, is ordered
   first in 0.19.0 (D-256). Each has a follow-up row (F-191 to F-198, F-180).
3. **Known gap, left as ruled.** An alert none of whose zones could be drawn is not on the map, and the
   map's description does not name it; the station's alert list does (D-250).

## The metrics

| Metric | State at BUILD exit |
|---|---|
| M1, M1b - where is it, from the picture; in words | UAT, closed by the HUM LEAD on 2026-10-03, stands as the evidence (D-252); the answer-key test pins each scenario's words |
| M2 - never global | `TestNoFrameIsWiderThanTheRegion`, over every frame the suite draws (D-28) |
| M3 - radar honesty | `TestTheNewestFramesAgeIsAlwaysSaid`, `TestTheNewestIsTheNewestObservedFrame` |
| M4 - partial honesty | `TestAPartialAreaIsDrawnAsFoundAndSaid`; the 0-of-N case is D-250's known gap |
| M5 - time to picture | **not measured by its protocol**: measured in VALIDATE (D-251). Live first-ever opens were 7.0-9.4 s against 3.5 s; a recorded miss returns as its own ruling |
| M6 - loop smoothness | thresholds never brought back: measured in VALIDATE, ruled before SHIP (D-251) |

## The gates

- `make verify` (every gate, `06_docs/required-gates.txt`): green at batch 137, locally.
- Hosted full CI, macOS and ubuntu: green at batch 136 (run 37298568637); batch 137's run is in
  progress when this report is written.
- P10 (`make p10`, a phase-exit gate): 0 live, 0 unmatched, 0 unratified; 159 ratified rows, the nine
  newest by D-248 and D-249 after real guards were added first.
- Mutants: 378, every anchor matching the tip, checked in the gate.

Blind spots: the timings in this release were measured on one machine and on the live network; M5's
protocol run is VALIDATE's (D-251). Windows is cross-compiled and analysed, not run.

## What REVIEW inherits

- The BUILD-exit red team's carried items: F-199 to F-202 (structural, no defect today).
- VALIDATE's work, ruled now: M5 by protocol, M6's thresholds (D-251).
- SHIP's: the `release/0.18.0` squash PR closing #22, with the CHANGELOG's 0.18.0 entry as its notes'
  source, and local `main` reset to `origin/main` (D-259).

## The ruling asked

Approve BUILD exit, so 0.18.0 proceeds to REVIEW.
