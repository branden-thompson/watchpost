---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — VALIDATE report"
date: 2026-10-30
phase: VALIDATE exit
sev: SEV-0
authority: HUM LEAD
status: "SIGNED OFF by the HUM LEAD (D-341: \"Approve after the check\"); the check clean - VALIDATE exits, SHIP next."
---

# VALIDATE report — 0.19.0, go-ionomaps v0.1.0, go-tuiMaps v0.3.0

Up: [VALIDATE's agenda](../07-readiness/validate-agenda.md) · [the exit red team](red-team-validate.md) · [rulings](../02-analysis/rulings.md) · [REVIEW report](review-report.md)

## The candidates

- **watchpost** `feature/propagation-overlays`, batch 81 (`63104fe7`): `make verify` green on each of VALIDATE's batches (77 to 81); hosted CI green through batch 80 (`3df12963`), batch 81's running on its push.
- **go-ionomaps** v0.1.0-rc.17 (`37b7d42`), `feature/discover`: its full gate green; the hosted full gate green on both architectures (run 40946426497). Docs-only commits since (`01f5dbb`, `562930f`).
- **go-tuiMaps** v0.3.0-rc.10 (`5f6bf07`), `feature/propagation-fields`: no code change in VALIDATE; its README audited (`57a9c29`).

## What VALIDATE checked (STANDARD VALIDATE, the SEV-0 default)

| Step | What ran | Result |
|---|---|---|
| Environment | the release binary on live data, a scratch HOME per run; the providers' health in the Status window | the eight weather providers answering; GIRO and NOAA SWPC reached by the Propagation mode |
| Core tests | the scripted-PTY journey (`make journey`) on the real binary, a first run each time, its Propagation-mode steps built at last (W10.1) | **34 of 34 steps in 3 runs of 3** on the batch-80 build; the transcript of every run kept |
| Regression | `make verify` (every gate, the mutant corpus) per batch; hosted CI on Linux and macOS | green; the corpus's hosted cap raised from 120 to 240 minutes when a docs-only commit's run died on the clock (batch 79) |
| Performance | the frame and setup benchmarks against 0.18.2, interleaved A/B (8 runs each); CPU profiles of G1's Broadcaster arm | within 10 % except the map window, +12.5 % (0.14 ms a frame): the larger D-194 window through Charm's compositor, which D-327 leaves as it is |
| README | a fresh reviewer read all three READMEs against the code | watchpost's credits, refresh, keys and four "you"s fixed; the libraries' minors fixed |
| Red team | two fresh reviewers (D-338): code and release | the code reviewer: exit; the release reviewer: not, on three Importants - each confirmed, ruled or fixed (batch 81) |
| UAT | the HUM LEAD's own use of `./dist/watchpost` (the scored sittings not held, D-326) | UAT issue one (D-324) found and fixed |

## What VALIDATE found and fixed

| Found | How | Fixed |
|---|---|---|
| The location rows held "loading" for over a minute while NASA FIRMS hung (also in 0.18.2) | UAT issue one, the HUM LEAD (D-324) | batch 77: a host that does not answer is avoided as one that refuses; each holds at most half a lane; and (batch 81) requests already queued fail fast |
| A first run's Setup closed with esc dropped the location, and every later write failed with an internal error (also in 0.18.2) | the core journey | batch 80 (D-336): esc keeps the pick on the first run; a change with no file yet is written; batch 81 keeps the pick through a refused save and says "esc Save" |
| The bands chart stayed "being worked out" if opened within 0.6 s of a pan or resize | the core journey | batch 80 (A-104 (4)) |
| Hourly would have said STALE for most of every hour | a 100-minute live watch (D-332) | batch 79: STALE an hour past the refresh's wait |
| The Propagation mode never refreshed while the chart was open | the release reviewer (VR-1) | batch 81 (D-339) |
| Each GIRO station's data provider was not credited (D-41) | the release reviewer (VR-11) | batch 81 (D-340) |
| The Pacific scored worse than the climatology | REVIEW's agenda (D-333) | go-ionomaps rc.17 (D-334): the climatology the background over the north-west tropical Pacific - the Pacific 1.23 against 1.22 on the fresh days, 1.39 against 1.39 on the 10-minute day, 1.28 against 1.32 on PLAN's independent week, gm3 at the build |
| The chart's agreement line wrapped "64%)" onto a line of its own | the journey's chart work | batch 80 (A-102) |

## The agenda, ruled

| # | Item | Ruling |
|---|---|---|
| 1 | M1 and M4 short of their targets | shipped as is, recorded (D-328); M1 kept over ten bands, said per band (D-329); the chart says its measured agreement (D-330) |
| 2 | PS-1's claim in the PR | the words approved (D-331) |
| 3 | G1, both comparisons; the Broadcaster arm | the tip's arm missed G1 too (+1.29 % of a core, +62.5 MB); a known limit (D-274, D-335); its CPU traced to Charm's window composite, no change (D-327, F-220) |
| 4 | The default refresh | Hourly (D-332) |
| 5 | The Pacific | fixed first (D-333, D-334) |
| 6 | The memory growth's fix over days | shown: at most 0.7 MB a day over 3 days, 0.18.2 8 to 9 |
| 7 | The changelog's known limits | worded and approved (D-335), with the red team's clarifications (A-105) |
| 8 | M5 at 800 by 220 | a known limit (D-261, D-335) |
| 9 | M2, M5, M5b's scored sittings | not held for 0.19.0 (D-326) |
| 10 | The core journey (found in VALIDATE) | the first-run defect ruled (D-336); no 0.18.3 (D-337) |

Also ruled in VALIDATE: no vetoes of A-37 (3) to A-101 (D-325); the exit red team's shape (D-338).

## The metrics, as the release ships

- **M1** 53.7 % against the climatology's 52.4 % (target 57.4 %) and **M4** 68.4 % against 64.4 % (target 69.4 %): both missed, recorded, scored on six quiet days before rc.17's Pacific change; no accuracy floor (D-75, D-328).
- **M2, M5, M5b**: not measured by a sitting (D-326). **M5's** frame target met at 400 by 110 (7 ms against 16); 25 to 27 ms at 800 by 220, a known limit.
- **G1**: met against the map open in Forecast mode; missed against the map never opened (the open map's own cost, F-220) and on the Broadcaster arm, both known limits.
- **G-M3** (go-ionomaps): every target met at rc.17.
- **PS-1** is a hypothesis the HUM LEAD holds, addressed in part (D-217, D-331); **PS-2** is addressed by the Observer's map, chart and words.
- All UAT verdicts are one person's, the HUM LEAD's (D-68).

## For the HUM LEAD's veto, batched (D-13)

A-101 (batch 77's choices), A-102 (the chart's line, shortened in batch 80), A-103 (STALE by the refresh, the file's names), A-104 (the first run and the chart's settle gate, the journey), A-105 (the red team's Minors), and go-ionomaps A-41 (the box's bounds, its 4° edge, the offset kept GloTEC's).

## Before SHIP

- **Hourly's STALE bound seen live on the shipped build** (A-105 (8), the code reviewer's VC-3): done, and the condition of the sign-off (D-341) - 150 minutes on `63104fe7`, three hourly refreshes, the oldest input at most 1 h 47 min old against the 2 h bound, no STALE.
- **The libraries' final tags**: go-ionomaps v0.1.0 and go-tuiMaps v0.3.0 cut from their candidates and pinned in watchpost (the changelog names the final versions).
- **No screenshot of the map or the Propagation mode** in the README: a gap for the release, not an inaccuracy.

## What the release learned here (for REFLECT)

- **A plan task can go unbuilt through two phases.** W10.1 - the journey's Propagation steps - was never built, and the journey had not run since 2026-09-08; run in VALIDATE, it found a defect shipped in 0.18.2. An instrument nothing runs is not an instrument (the Makefile's own comment said so).
- **A probe built to answer a question can hide the case.** Every probe that pre-created a config file missed the first-run defect; every probe that waited for the snapshot missed the chart's deadlock. The journey, doing what a listener does, found both.
- **A fix can make its own failure.** Esc saving on the first run held Setup open until the save landed, and the journey's next key went into it; the fix closed the window at once.
- **A default changed in VALIDATE changes what everything else costs.** Hourly made a skipped refresh six times dearer (VR-1) and a 1-hour STALE bound wrong (D-332); both were found by looking, not by reading.
- **Figures carry their source.** Two of D-334's three sets were first a probe's; re-scored at the build, they held, but the record said otherwise until the red team asked (VC-4).
